// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/tls"
	"errors"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// serverTLS returns the gRPC server options that put the listener behind
// TLS, from a PEM certificate chain and its private key. Neither file
// named means a plaintext listener and no options; naming only one is a
// mistake, not a request for plaintext, and is refused.
//
// The key pair is read once, here, so a bad path or a mismatched key
// stops the process before it listens. TLS 1.2 is the floor, and 1.3 is
// negotiated with any client that offers it; the cipher suites are the
// standard library's defaults, which change with the Go release rather
// than with this code. To rotate the certificate, restart the process.
func serverTLS(certFile, keyFile string) ([]grpc.ServerOption, error) {
	cfg, err := tlsConfig(certFile, keyFile)
	if cfg == nil || err != nil {
		return nil, err
	}
	return []grpc.ServerOption{grpc.Creds(credentials.NewTLS(cfg))}, nil
}

// tlsConfig is the listener's TLS configuration, or nil for plaintext.
func tlsConfig(certFile, keyFile string) (*tls.Config, error) {
	if certFile == "" && keyFile == "" {
		return nil, nil
	}
	if certFile == "" || keyFile == "" {
		return nil, errors.New("-tls-cert and -tls-key must be given together")
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("-tls-cert/-tls-key: %w", err)
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}, nil
}
