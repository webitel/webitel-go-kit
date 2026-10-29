package grpc_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/webitel/webitel-go-kit/appconfig"
	wgrpc "github.com/webitel/webitel-go-kit/infra/transport/gRPC"
)

type ca struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
	file string
}

func newCA(t *testing.T, dir, name string) *ca {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return &ca{cert: cert, key: key, file: writePEM(t, dir, name+".pem", "CERTIFICATE", der)}
}

// issue signs a leaf for both server and client use and returns cert and key file paths.
func (c *ca) issue(t *testing.T, dir, name string) appconfig.TLS {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: name},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, c.cert, &key.PublicKey, c.key)
	require.NoError(t, err)
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	return appconfig.TLS{
		CA:   c.file,
		Cert: writePEM(t, dir, name+".crt", "CERTIFICATE", der),
		Key:  writePEM(t, dir, name+".key", "EC PRIVATE KEY", keyDER),
	}
}

func writePEM(t *testing.T, dir, name, typ string, der []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}), 0o600))
	return p
}

func TestTLSDisabled(t *testing.T) {
	conn := appconfig.GRPCConn{}

	sc, err := wgrpc.ServerTLSConfig(conn)
	require.NoError(t, err)
	require.Nil(t, sc)
	cc, err := wgrpc.ClientTLSConfig(conn)
	require.NoError(t, err)
	require.Nil(t, cc)

	creds, err := wgrpc.ServerCredentials(conn)
	require.NoError(t, err)
	require.Equal(t, "insecure", creds.Info().SecurityProtocol)
	creds, err = wgrpc.ClientCredentials(conn)
	require.NoError(t, err)
	require.Equal(t, "insecure", creds.Info().SecurityProtocol)
}

func TestTLSConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	authority := newCA(t, dir, "ca")
	leaf := authority.issue(t, dir, "leaf")
	conn := appconfig.GRPCConn{VerifyCerts: true, MTLS: true, TLS: leaf, Client: leaf}

	sc, err := wgrpc.ServerTLSConfig(conn)
	require.NoError(t, err)
	require.Equal(t, uint16(tls.VersionTLS12), sc.MinVersion)
	require.Len(t, sc.Certificates, 1)
	require.NotNil(t, sc.ClientCAs)
	require.Equal(t, tls.RequireAndVerifyClientCert, sc.ClientAuth)

	cc, err := wgrpc.ClientTLSConfig(conn)
	require.NoError(t, err)
	require.False(t, cc.InsecureSkipVerify)
	require.Equal(t, uint16(tls.VersionTLS12), cc.MinVersion)
	require.NotNil(t, cc.RootCAs)
	require.Len(t, cc.Certificates, 1)

	conn.MTLS = false
	sc, err = wgrpc.ServerTLSConfig(conn)
	require.NoError(t, err)
	require.Nil(t, sc.ClientCAs)
	require.Equal(t, tls.NoClientCert, sc.ClientAuth)
	// Without a configured client cert/key no certificate is presented.
	conn.Client = appconfig.TLS{CA: leaf.CA}
	cc, err = wgrpc.ClientTLSConfig(conn)
	require.NoError(t, err)
	require.Empty(t, cc.Certificates)
}

func TestTLSConfigErrors(t *testing.T) {
	dir := t.TempDir()
	authority := newCA(t, dir, "ca")
	leaf := authority.issue(t, dir, "leaf")
	garbage := filepath.Join(dir, "garbage.pem")
	require.NoError(t, os.WriteFile(garbage, []byte("not pem"), 0o600))
	missing := filepath.Join(dir, "missing.pem")

	t.Run("server missing cert", func(t *testing.T) {
		bad := leaf
		bad.Cert = missing
		_, err := wgrpc.ServerTLSConfig(appconfig.GRPCConn{VerifyCerts: true, TLS: bad})
		require.Error(t, err)
	})
	t.Run("server mtls missing CA", func(t *testing.T) {
		bad := leaf
		bad.CA = missing
		_, err := wgrpc.ServerTLSConfig(appconfig.GRPCConn{VerifyCerts: true, MTLS: true, TLS: bad})
		require.Error(t, err)
	})
	t.Run("server mtls garbage CA", func(t *testing.T) {
		bad := leaf
		bad.CA = garbage
		_, err := wgrpc.ServerTLSConfig(appconfig.GRPCConn{VerifyCerts: true, MTLS: true, TLS: bad})
		require.ErrorContains(t, err, "no valid PEM")
	})
	t.Run("client garbage CA", func(t *testing.T) {
		bad := leaf
		bad.CA = garbage
		_, err := wgrpc.ClientTLSConfig(appconfig.GRPCConn{VerifyCerts: true, Client: bad})
		require.ErrorContains(t, err, "no valid PEM")
	})
	t.Run("client cert without key", func(t *testing.T) {
		bad := leaf
		bad.Key = ""
		_, err := wgrpc.ClientTLSConfig(appconfig.GRPCConn{VerifyCerts: true, Client: bad})
		require.Error(t, err)
	})
	t.Run("client cert file missing", func(t *testing.T) {
		bad := leaf
		bad.Cert = missing
		_, err := wgrpc.ClientTLSConfig(appconfig.GRPCConn{VerifyCerts: true, Client: bad})
		require.Error(t, err)
	})
	t.Run("client empty CA path", func(t *testing.T) {
		_, err := wgrpc.ClientTLSConfig(appconfig.GRPCConn{VerifyCerts: true})
		require.ErrorContains(t, err, "CA path is empty")
	})
	t.Run("credentials propagate error", func(t *testing.T) {
		creds, err := wgrpc.ServerCredentials(appconfig.GRPCConn{VerifyCerts: true, TLS: appconfig.TLS{Cert: missing, Key: missing}})
		require.Error(t, err)
		require.Nil(t, creds)
		creds, err = wgrpc.ClientCredentials(appconfig.GRPCConn{VerifyCerts: true})
		require.Error(t, err)
		require.Nil(t, creds)
	})
}

// check starts a health server with serverConn and calls it with clientConn.
func check(t *testing.T, serverConn, clientConn appconfig.GRPCConn) error {
	t.Helper()
	sc, err := wgrpc.ServerCredentials(serverConn)
	require.NoError(t, err)
	cc, err := wgrpc.ClientCredentials(clientConn)
	require.NoError(t, err)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer(grpc.Creds(sc))
	healthpb.RegisterHealthServer(srv, health.NewServer())
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(cc))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
	return err
}

func TestHandshake(t *testing.T) {
	dir := t.TempDir()
	authority := newCA(t, dir, "ca")
	server := authority.issue(t, dir, "server")
	client := authority.issue(t, dir, "client")
	other := newCA(t, dir, "other-ca")
	stranger := other.issue(t, dir, "stranger")

	t.Run("mtls ok", func(t *testing.T) {
		require.NoError(t, check(t,
			appconfig.GRPCConn{VerifyCerts: true, MTLS: true, TLS: server},
			appconfig.GRPCConn{VerifyCerts: true, MTLS: true, Client: client}))
	})
	t.Run("mtls client signed by foreign CA is rejected", func(t *testing.T) {
		require.Error(t, check(t,
			appconfig.GRPCConn{VerifyCerts: true, MTLS: true, TLS: server},
			appconfig.GRPCConn{VerifyCerts: true, MTLS: true, Client: appconfig.TLS{CA: authority.file, Cert: stranger.Cert, Key: stranger.Key}}))
	})
	t.Run("mtls client without certificate is rejected", func(t *testing.T) {
		require.Error(t, check(t,
			appconfig.GRPCConn{VerifyCerts: true, MTLS: true, TLS: server},
			appconfig.GRPCConn{VerifyCerts: true, Client: appconfig.TLS{CA: authority.file}}))
	})
	t.Run("client presents its certificate even when its own mtls is off", func(t *testing.T) {
		require.NoError(t, check(t,
			appconfig.GRPCConn{VerifyCerts: true, MTLS: true, TLS: server},
			appconfig.GRPCConn{VerifyCerts: true, MTLS: false, Client: client}))
	})
	t.Run("tls only: client without certificate is accepted", func(t *testing.T) {
		require.NoError(t, check(t,
			appconfig.GRPCConn{VerifyCerts: true, TLS: server},
			appconfig.GRPCConn{VerifyCerts: true, Client: appconfig.TLS{CA: authority.file}}))
	})
	t.Run("tls only: client with certificate is accepted", func(t *testing.T) {
		require.NoError(t, check(t,
			appconfig.GRPCConn{VerifyCerts: true, TLS: server},
			appconfig.GRPCConn{VerifyCerts: true, Client: client}))
	})
	t.Run("client rejects server signed by foreign CA", func(t *testing.T) {
		require.Error(t, check(t,
			appconfig.GRPCConn{VerifyCerts: true, TLS: stranger},
			appconfig.GRPCConn{VerifyCerts: true, Client: client}))
	})
	t.Run("plaintext client against tls server fails", func(t *testing.T) {
		require.Error(t, check(t,
			appconfig.GRPCConn{VerifyCerts: true, TLS: server},
			appconfig.GRPCConn{}))
	})
}
