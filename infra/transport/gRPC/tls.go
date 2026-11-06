package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/webitel/webitel-go-kit/appconfig"
)

// ServerTLSConfig builds the server config from conn.TLS.
func ServerTLSConfig(conn appconfig.GRPCConn) (*tls.Config, error) {
	if !conn.VerifyCerts {
		return nil, nil
	}
	cert, err := tls.LoadX509KeyPair(conn.Cert, conn.Key)
	if err != nil {
		return nil, fmt.Errorf("grpc tls: server: %w", err)
	}
	cfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}
	if conn.MTLS {
		if cfg.ClientCAs, err = loadCA(conn.CA); err != nil {
			return nil, fmt.Errorf("grpc tls: server: %w", err)
		}
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return cfg, nil
}

// ClientTLSConfig builds the client tls.Config from conn.Client.
func ClientTLSConfig(conn appconfig.GRPCConn) (*tls.Config, error) {
	if !conn.VerifyCerts {
		return nil, nil
	}
	pool, err := loadCA(conn.Client.CA)
	if err != nil {
		return nil, fmt.Errorf("grpc tls: client: %w", err)
	}
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    pool,
	}
	if conn.Client.Cert != "" || conn.Client.Key != "" {
		cert, err := tls.LoadX509KeyPair(conn.Client.Cert, conn.Client.Key)
		if err != nil {
			return nil, fmt.Errorf("grpc tls: client: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	return cfg, nil
}

// ServerCredentials returns credentials for grpc.Creds; insecure when verify_certs is off.
func ServerCredentials(conn appconfig.GRPCConn) (credentials.TransportCredentials, error) {
	cfg, err := ServerTLSConfig(conn)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return insecure.NewCredentials(), nil
	}
	return credentials.NewTLS(cfg), nil
}

// ClientCredentials returns credentials for grpc.WithTransportCredentials; insecure when verify_certs is off.
func ClientCredentials(conn appconfig.GRPCConn) (credentials.TransportCredentials, error) {
	cfg, err := ClientTLSConfig(conn)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return insecure.NewCredentials(), nil
	}
	return credentials.NewTLS(cfg), nil
}

func loadCA(file string) (*x509.CertPool, error) {
	if file == "" {
		return nil, errors.New("CA path is empty")
	}
	pem, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read CA: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("CA %q has no valid PEM certificates", file)
	}
	return pool, nil
}
