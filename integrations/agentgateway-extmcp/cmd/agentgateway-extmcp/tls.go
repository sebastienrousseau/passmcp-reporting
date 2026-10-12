// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"strings"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// certPair is the listener's certificate and key, read from two PEM files
// and swapped in place when they are read again. Every handshake takes the
// pair current at that moment, so a rotated certificate is served to the
// next connection without a restart; connections already open keep the
// one they negotiated.
type certPair struct {
	certFile, keyFile string
	current           atomic.Pointer[tls.Certificate]
}

// loadCertPair reads the pair once. A bad path or a key that does not match
// the certificate is an error, so the process stops before it listens. A
// file that is not there also names the way out: the container image
// names the pair in its default arguments, so this, not errNoKeyPair, is
// what an operator who mounted none sees.
func loadCertPair(certFile, keyFile string) (*certPair, error) {
	kp := &certPair{certFile: certFile, keyFile: keyFile}
	if err := kp.reload(); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w; the processor serves TLS only: provide the key pair, or use -plaintext on a loopback -listen address", err)
		}
		return nil, err
	}
	return kp, nil
}

// reload reads the pair again and swaps it in if it loads. If it does not,
// the error is returned and the certificate in service stays, the rule
// reloadOn applies to a configuration that does not parse: a certificate
// manager caught between writing the certificate and the key must not
// take the listener down.
func (kp *certPair) reload() error {
	cert, err := tls.LoadX509KeyPair(kp.certFile, kp.keyFile)
	if err != nil {
		return fmt.Errorf("-tls-cert/-tls-key: %w", err)
	}
	kp.current.Store(&cert)
	return nil
}

// certificate is the tls.Config GetCertificate callback: the pair in
// service now.
func (kp *certPair) certificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return kp.current.Load(), nil
}

// errNoKeyPair is the refusal to start without a key pair or -plaintext.
var errNoKeyPair = errors.New("no TLS key pair: give -tls-cert and -tls-key, or -plaintext to serve without TLS on a loopback -listen address (127.0.0.0/8, ::1 or localhost)")

// transport decides how the listener on listen is secured: TLS from the
// key pair, or, only when plaintext is asked for, nothing. Plaintext is an
// explicit opt-in, and it is refused beside a key pair, since one of the
// two would be ignored, and on any address that is not loopback.
func transport(listen, certFile, keyFile string, plaintext bool) ([]grpc.ServerOption, *certPair, error) {
	if !plaintext {
		return serverTLS(certFile, keyFile)
	}
	if certFile != "" || keyFile != "" {
		return nil, nil, errors.New("-plaintext cannot be combined with -tls-cert or -tls-key: choose TLS or plaintext")
	}
	return nil, nil, loopbackOnly(listen)
}

// loopbackOnly accepts a -listen address whose host is a literal loopback
// IP (127.0.0.0/8, ::1, or IPv4 loopback written as IPv6) or the name
// localhost, and refuses everything else: an empty host or a wildcard,
// which bind every interface, a routable address, and every other host
// name. A name other than localhost resolves to whatever DNS or
// /etc/hosts says on the day, so it is no evidence the listener stays on
// this host; it fails closed. localhost is taken by name, and where it
// resolved is checked again once bound (boundToLoopback).
func loopbackOnly(listen string) error {
	host, _, err := net.SplitHostPort(listen)
	if err != nil {
		return fmt.Errorf("-listen: %w", err)
	}
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return nil
	}
	return fmt.Errorf("-plaintext serves only on a loopback -listen address (127.0.0.0/8, ::1 or localhost), not %q; give -tls-cert and -tls-key to listen there", host)
}

// listen opens the listener on addr. A plaintext one that did not land on
// a loopback address is closed rather than served.
func listen(addr string, plaintext bool) (net.Listener, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil || !plaintext {
		return lis, err
	}
	if err := boundToLoopback(lis.Addr()); err != nil {
		_ = lis.Close()
		return nil, err
	}
	return lis, nil
}

// boundToLoopback refuses a bound address that is not a loopback TCP one.
func boundToLoopback(a net.Addr) error {
	if t, ok := a.(*net.TCPAddr); ok && t.IP.IsLoopback() {
		return nil
	}
	return fmt.Errorf("-plaintext: the listener bound %s, which is not loopback; refusing to serve without TLS", a)
}

// serverTLS returns the gRPC server options that put the listener behind
// TLS, from a PEM certificate chain and its private key, and the key pair
// to reload. Neither file named is refused: plaintext takes -plaintext,
// so a forgotten flag cannot open the listener in the clear. Naming only
// one is a mistake, not a request for plaintext, and is refused too.
//
// TLS 1.2 is the floor, and 1.3 is negotiated with any client that offers
// it; the cipher suites are the standard library's defaults, which change
// with the Go release rather than with this code.
func serverTLS(certFile, keyFile string) ([]grpc.ServerOption, *certPair, error) {
	cfg, kp, err := tlsConfig(certFile, keyFile)
	if err != nil {
		return nil, nil, err
	}
	return []grpc.ServerOption{grpc.Creds(credentials.NewTLS(cfg))}, kp, nil
}

// tlsConfig is the listener's TLS configuration and its key pair.
func tlsConfig(certFile, keyFile string) (*tls.Config, *certPair, error) {
	if certFile == "" && keyFile == "" {
		return nil, nil, errNoKeyPair
	}
	if certFile == "" || keyFile == "" {
		return nil, nil, errors.New("-tls-cert and -tls-key must be given together")
	}
	kp, err := loadCertPair(certFile, keyFile)
	if err != nil {
		return nil, nil, err
	}
	return &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: kp.certificate,
	}, kp, nil
}
