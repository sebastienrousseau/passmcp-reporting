// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/tls"
	"errors"
	"fmt"
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
// the certificate is an error, so the process stops before it listens.
func loadCertPair(certFile, keyFile string) (*certPair, error) {
	kp := &certPair{certFile: certFile, keyFile: keyFile}
	if err := kp.reload(); err != nil {
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

// serverTLS returns the gRPC server options that put the listener behind
// TLS, from a PEM certificate chain and its private key, and the key pair
// to reload. Neither file named means a plaintext listener, no options
// and a nil pair; naming only one is a mistake, not a request for
// plaintext, and is refused.
//
// TLS 1.2 is the floor, and 1.3 is negotiated with any client that offers
// it; the cipher suites are the standard library's defaults, which change
// with the Go release rather than with this code.
func serverTLS(certFile, keyFile string) ([]grpc.ServerOption, *certPair, error) {
	cfg, kp, err := tlsConfig(certFile, keyFile)
	if cfg == nil || err != nil {
		return nil, nil, err
	}
	return []grpc.ServerOption{grpc.Creds(credentials.NewTLS(cfg))}, kp, nil
}

// tlsConfig is the listener's TLS configuration and its key pair, or nil
// for plaintext.
func tlsConfig(certFile, keyFile string) (*tls.Config, *certPair, error) {
	if certFile == "" && keyFile == "" {
		return nil, nil, nil
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
