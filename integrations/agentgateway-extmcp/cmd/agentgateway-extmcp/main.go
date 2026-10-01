// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Command agentgateway-extmcp serves agentgateway's ExtMcp policy hook and
// gates MCP backends on passmcp attestations.
//
//	agentgateway-extmcp -config config.json -listen 127.0.0.1:4400
//	agentgateway-extmcp -config config.json -tls-cert tls.crt -tls-key tls.key
//
// The configuration and every attestation it names are loaded before the
// listener opens, so a processor that is up is one that has decided what
// it will say. SIGHUP reloads both, and the TLS key pair when there is one;
// -reload-interval does so on a timer. With -tls-cert and -tls-key the
// listener speaks TLS 1.2 or later; without them it is plaintext, for
// loopback or a private network.
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

// options are the command's flags, parsed.
type options struct {
	config, listen, level, completion, tlsCert, tlsKey string
	interval, timeout                                  time.Duration
	maxBytes                                           int64
}

// flagSet declares the command's flags over o. The completion scripts are
// generated from it, so a flag added here completes without a second list.
func flagSet(stderr io.Writer, o *options) *flag.FlagSet {
	fs := flag.NewFlagSet("agentgateway-extmcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.config, "config", "", "configuration file (JSON); required")
	fs.StringVar(&o.listen, "listen", "127.0.0.1:4400", "address to serve gRPC on")
	fs.DurationVar(&o.interval, "reload-interval", 0, "re-read the configuration and attestations this often; 0 disables")
	fs.Int64Var(&o.maxBytes, "max-bytes", processor.DefaultMaxBytes, "largest attestation accepted, in bytes")
	fs.DurationVar(&o.timeout, "fetch-timeout", processor.DefaultTimeout, "time allowed for one https fetch")
	fs.StringVar(&o.level, "log-level", "info", "debug, info, warn or error")
	fs.StringVar(&o.tlsCert, "tls-cert", "", "PEM certificate chain to serve gRPC over TLS with; needs -tls-key")
	fs.StringVar(&o.tlsKey, "tls-key", "", "PEM private key for -tls-cert")
	fs.StringVar(&o.completion, "completion", "", "print a completion script for bash, zsh or fish, and exit")
	return fs
}

// prepared is what prepare builds: the logger, the listener's transport
// credentials, and the key pair behind them, nil for plaintext.
type prepared struct {
	log   *slog.Logger
	creds []grpc.ServerOption
	certs *certPair
}

// prepare checks what parsing cannot, and builds the logger and the
// listener's transport credentials, before anything is loaded or bound.
func (o *options) prepare(stderr io.Writer) (*prepared, error) {
	if o.config == "" {
		return nil, errors.New("-config is required")
	}
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(o.level)); err != nil {
		return nil, fmt.Errorf("-log-level: %w", err)
	}
	creds, certs, err := serverTLS(o.tlsCert, o.tlsKey)
	if err != nil {
		return nil, err
	}
	log := slog.New(slog.NewJSONHandler(stderr, &slog.HandlerOptions{Level: lvl}))
	return &prepared{log: log, creds: creds, certs: certs}, nil
}

// run is main without the process: ready is signalled with the bound
// address once the listener is open, and ctx ending stops the server.
func run(ctx context.Context, args []string, stderr io.Writer, ready chan<- net.Addr) error {
	var o options
	fs := flagSet(stderr, &o)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if o.completion != "" {
		return writeCompletion(stdout, fs, o.completion)
	}
	p, err := o.prepare(stderr)
	if err != nil {
		return err
	}
	log := p.log

	// SIGHUP's handler goes in before anything can announce the process:
	// until signal.Notify runs, a hangup takes the default action and kills
	// it, so an operator reloading just after start would stop the gate.
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	defer signal.Stop(hup)

	store := processor.NewStore(o.config, &processor.Loader{MaxBytes: o.maxBytes, Timeout: o.timeout}, log)
	if err := store.Reload(ctx); err != nil {
		return err
	}

	lis, err := net.Listen("tcp", o.listen)
	if err != nil {
		return err
	}
	srv := grpc.NewServer(p.creds...)
	extmcp.RegisterExtMcpServer(srv, processor.NewServer(store, log))
	log.Info("listening", "addr", lis.Addr().String(), "config", o.config, "tls", p.certs != nil)
	if ready != nil {
		ready <- lis.Addr()
	}
	r := reloader{store: store, certs: p.certs, log: log}
	return serve(ctx, srv, lis, func(rctx context.Context) { r.on(rctx, o.interval, hup) })
}

// serve runs srv on lis, and reload beside it, until ctx ends or the
// server fails. The reloader is joined before serve returns, so nothing
// it logs can land after the caller believes the process has stopped.
func serve(ctx context.Context, srv *grpc.Server, lis net.Listener, reload func(context.Context)) error {
	rctx, stopReload := context.WithCancel(ctx)
	reloaded := make(chan struct{})
	go func() { defer close(reloaded); reload(rctx) }()
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

// reloader is what a reload re-reads: the store and, on a TLS listener,
// the key pair.
type reloader struct {
	store *processor.Store
	certs *certPair
	log   *slog.Logger
}

// on reloads on SIGHUP and, when interval is positive, on a timer. A reload
// that fails is logged and the last good snapshot, or certificate, stays in
// service: a broken edit must not take the gate down.
func (r reloader) on(ctx context.Context, interval time.Duration, hup <-chan os.Signal) {
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
		r.reload(ctx, why)
	}
}

// reload re-reads the store and the key pair once. Each is independent: a
// configuration that does not parse does not stop a rotated certificate
// being served, nor the reverse.
func (r reloader) reload(ctx context.Context, why string) {
	if err := r.store.Reload(ctx); err != nil {
		r.log.Error("reload failed; keeping the previous configuration", "trigger", why, "error", err.Error())
	} else {
		r.log.Info("reloaded", "trigger", why)
	}
	if r.certs == nil {
		return
	}
	if err := r.certs.reload(); err != nil {
		r.log.Error("TLS key pair reload failed; keeping the previous certificate", "trigger", why, "error", err.Error())
		return
	}
	r.log.Info("reloaded the TLS key pair", "trigger", why)
}
