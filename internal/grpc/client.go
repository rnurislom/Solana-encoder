package grpc

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"

	"wallet-monitor/internal/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type basicAuthCreds struct {
	username string
	password string
	insecure bool
}

func (b basicAuthCreds) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	encoded := base64.StdEncoding.EncodeToString([]byte(b.username + ":" + b.password))
	return map[string]string{
		"authorization": "Basic " + encoded,
	}, nil
}

func (b basicAuthCreds) RequireTransportSecurity() bool {
	return !b.insecure
}

var keepaliveParams = keepalive.ClientParameters{
	Time:                10 * time.Second,
	Timeout:             time.Second,
	PermitWithoutStream: true,
}

func Connect(cfg *config.Config) (*grpc.ClientConn, error) {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	isInsecure := cfg.Insecure || u.Scheme == "http"

	port := u.Port()
	if port == "" {
		if isInsecure {
			port = "80"
		} else {
			port = "443"
		}
	}

	hostname := u.Hostname()
	if hostname == "" {
		return nil, fmt.Errorf("endpoint URL must include a hostname")
	}

	address := hostname + ":" + port

	var opts []grpc.DialOption
	if isInsecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		pool, _ := x509.SystemCertPool()
		creds := credentials.NewClientTLSFromCert(pool, "")
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}
	opts = append(opts, grpc.WithKeepaliveParams(keepaliveParams))

	if cfg.HasBasicAuth() {
		opts = append(opts, grpc.WithPerRPCCredentials(basicAuthCreds{
			username: cfg.Username,
			password: cfg.Password,
			insecure: isInsecure,
		}))
	}

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		return nil, fmt.Errorf("grpc dial failed: %w", err)
	}

	return conn, nil
}
