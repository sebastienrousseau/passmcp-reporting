// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Command agentgateway-extmcp serves agentgateway's ExtMcp policy hook and
// gates MCP backends on passmcp attestations.
//
//	agentgateway-extmcp -config config.json -listen 127.0.0.1:4400
//
// The configuration and every attestation it names are loaded before the
// listener opens, so a processor that is up is one that has decided what
// it will say. SIGHUP reloads both; -reload-interval does so on a timer.
// Diagnostics go to stderr as JSON lines.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"satellion.com/passmcp-reporting/integrations/agentgateway-extmcp/gen/extmcp"
	"satellion.com/passmcp-reporting/integrations/agentgateway-extmcp/processor"
)

// stdout receives what the command prints on purpose: today, only a
// completion script. Diagnostics go to stderr.
var stdout io.Writer = os.Stdout

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stderr, nil); err != nil {
		fmt.Fprintln(os.Stderr, "agentgateway-extmcp:", err)
		os.Exit(1)
	}
}

// run is main without the process: ready is signalled with the bound
// address once the listener is open, and ctx ending stops the server.
func run(ctx context.Context, args []string, stderr io.Writer, ready chan<- net.Addr) error {
	fs := flag.NewFlagSet("agentgateway-extmcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", "", "configuration file (JSON); required")
	listen := fs.String("listen", "127.0.0.1:4400", "address to serve gRPC on")
	interval := fs.Duration("reload-interval", 0, "re-read the configuration and attestations this often; 0 disables")
	maxBytes := fs.Int64("max-bytes", processor.DefaultMaxBytes, "largest attestation accepted, in bytes")
	timeout := fs.Duration("fetch-timeout", processor.DefaultTimeout, "time allowed for one https fetch")
	level := fs.String("log-level", "info", "debug, info, warn or error")
	completion := fs.String("completion", "", "print a completion script for bash, zsh or fish, and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *completion != "" {
		return writeCompletion(stdout, fs, *completion)
	}
	if *configPath == "" {
		return errors.New("-config is required")
	}
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(*level)); err != nil {
		return fmt.Errorf("-log-level: %w", err)
	}
	log := slog.New(slog.NewJSONHandler(stderr, &slog.HandlerOptions{Level: lvl}))

	// SIGHUP's handler goes in before anything can announce the process:
	// until signal.Notify runs, a hangup takes the default action and kills
	// it, so an operator reloading just after start would stop the gate.
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	defer signal.Stop(hup)

	store := processor.NewStore(*configPath, &processor.Loader{MaxBytes: *maxBytes, Timeout: *timeout}, log)
	if err := store.Reload(ctx); err != nil {
		return err
	}

	lis, err := net.Listen("tcp", *listen)
	if err != nil {
		return err
	}
	srv := grpc.NewServer()
	extmcp.RegisterExtMcpServer(srv, processor.NewServer(store, log))
	log.Info("listening", "addr", lis.Addr().String(), "config", *configPath)
	if ready != nil {
		ready <- lis.Addr()
	}

	// The reloader is joined before run returns, so nothing it logs can land
	// after the caller believes the process has stopped.
	rctx, stopReload := context.WithCancel(ctx)
	reloaded := make(chan struct{})
	go func() { defer close(reloaded); reloadOn(rctx, store, log, *interval, hup) }()
	defer func() { stopReload(); <-reloaded }()

	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(lis) }()
	select {
	case <-ctx.Done():
		srv.GracefulStop()
		return nil
	case err := <-errc:
		return err
	}
}

// reloadOn reloads the store on SIGHUP and, when interval is positive, on
// a timer. A reload that fails is logged and the last good snapshot stays
// in service: a broken edit must not take the gate down.
func reloadOn(ctx context.Context, store *processor.Store, log *slog.Logger, interval time.Duration, hup <-chan os.Signal) {
	var tick <-chan time.Time
	if interval > 0 {
		t := time.NewTicker(interval)
		defer t.Stop()
		tick = t.C
	}
	for {
		var why string
		select {
		case <-ctx.Done():
			return
		case <-hup:
			why = "SIGHUP"
		case <-tick:
			why = "interval"
		}
		if err := store.Reload(ctx); err != nil {
			log.Error("reload failed; keeping the previous configuration", "trigger", why, "error", err.Error())
			continue
		}
		log.Info("reloaded", "trigger", why)
	}
}
