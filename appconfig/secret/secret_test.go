package secret_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/webitel/webitel-go-kit/appconfig/secret"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func envOf(m map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) {
		v, ok := m[k]
		return v, ok
	}
}

func TestEnvName(t *testing.T) {
	for key, want := range map[string]string{
		"postgres.dsn":       "POSTGRES_DSN",
		"service.secret_key": "SERVICE_SECRET_KEY",
		"service.http-token": "SERVICE_HTTP_TOKEN",
	} {
		if got := secret.EnvName(key); got != want {
			t.Errorf("EnvName(%q): want %q, got %q", key, want, got)
		}
	}
}

func TestResolve(t *testing.T) {
	creds := t.TempDir()
	other := t.TempDir()
	writeFile(t, creds, "postgres.dsn", "postgres://db.example.com/test\n")
	filePath := writeFile(t, other, "token", "test-token-file\r\n")
	writeFile(t, creds, "empty", "\n")

	r := secret.New(creds, nil)

	tests := []struct {
		name    string
		in      string
		want    string
		wantErr string
	}{
		{name: "literal is returned as is", in: "test-token-literal", want: "test-token-literal"},
		{name: "empty literal", in: "", want: ""},
		{name: "credential, trailing newline trimmed", in: "cred:postgres.dsn", want: "postgres://db.example.com/test"},
		{name: "file, trailing CRLF trimmed", in: "file:" + filePath, want: "test-token-file"},
		{name: "missing credential", in: "cred:nope", wantErr: "no such file"},
		{name: "empty credential", in: "cred:empty", wantErr: "value is empty"},
		{name: "credential name escaping the directory", in: "cred:../token", wantErr: "invalid credential name"},
		{name: "empty credential name", in: "cred:", wantErr: "invalid credential name"},
		{name: "empty file path", in: "file:", wantErr: "empty file path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Resolve(tt.in)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("want error containing %q, got %v (value %q)", tt.wantErr, err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		})
	}
}

func TestResolveWithoutCredentialsDir(t *testing.T) {
	_, err := secret.New("", nil).Resolve("cred:postgres.dsn")
	if !errors.Is(err, secret.ErrNoCredentialsDir) {
		t.Fatalf("want ErrNoCredentialsDir, got %v", err)
	}
}

func TestResolveTooLarge(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "big", strings.Repeat("x", secret.MaxSize+1))
	if _, err := secret.New("", nil).Resolve("file:" + p); err == nil || !strings.Contains(err.Error(), "larger than") {
		t.Fatalf("want size error, got %v", err)
	}
}

func TestErrorsDoNotLeakContents(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "postgres.dsn", "test-password-must-not-leak")
	r := secret.New(dir, envOf(map[string]string{"POSTGRES_DSN_FILE": "/nonexistent"}))

	_, err := r.Value("postgres.dsn", "")
	if err == nil {
		t.Fatal("want error")
	}
	if strings.Contains(err.Error(), "test-password-must-not-leak") {
		t.Fatalf("error leaks secret contents: %v", err)
	}
}

func TestPath(t *testing.T) {
	r := secret.New("/run/credentials/test.service", nil)
	for in, want := range map[string]string{
		"cred:service.conn.key":      "/run/credentials/test.service/service.conn.key",
		"file:/etc/test/tls/key.pem": "/etc/test/tls/key.pem",
		"/etc/test/tls/key.pem":      "/etc/test/tls/key.pem",
	} {
		got, err := r.Path(in)
		if err != nil {
			t.Fatalf("Path(%q): %v", in, err)
		}
		if got != want {
			t.Errorf("Path(%q): want %q, got %q", in, want, got)
		}
	}
}

func TestValue(t *testing.T) {
	creds := t.TempDir()
	other := t.TempDir()
	writeFile(t, creds, "postgres.dsn", "test-dsn-from-credential")
	writeFile(t, creds, "custom-name", "test-dsn-from-named-credential")
	envFile := writeFile(t, other, "redis_password", "test-password-from-env-file")

	r := secret.New(creds, envOf(map[string]string{
		"REDIS_PASSWORD_FILE": envFile,
		"PUBSUB_URL_FILE":     "", // set but empty is treated as unset
	}))

	tests := []struct {
		name    string
		key     string
		in      string
		want    string
		wantErr error
	}{
		{name: "unset key, no source", key: "pubsub.url", in: "", want: ""},
		{name: "literal, no source", key: "pubsub.url", in: "amqp://mq.example.com/", want: "amqp://mq.example.com/"},
		{name: "unset key picks up credential named after it", key: "postgres.dsn", in: "", want: "test-dsn-from-credential"},
		{name: "unset key picks up <ENV>_FILE", key: "redis.password", in: "", want: "test-password-from-env-file"},
		{name: "explicit reference to the same credential", key: "postgres.dsn", in: "cred:postgres.dsn", want: "test-dsn-from-credential"},
		{name: "explicit reference without conventional source", key: "pubsub.url", in: "cred:custom-name", want: "test-dsn-from-named-credential"},
		{name: "literal shadowing a credential", key: "postgres.dsn", in: "test-dsn-literal", wantErr: secret.ErrConflict},
		{name: "literal shadowing <ENV>_FILE", key: "redis.password", in: "test-password-literal", wantErr: secret.ErrConflict},
		{name: "reference to another source than the conventional one", key: "postgres.dsn", in: "cred:custom-name", wantErr: secret.ErrConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Value(tt.key, tt.in)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("want %v, got %v (value %q)", tt.wantErr, err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		})
	}
}

func TestEnvFileTakesPrecedenceOverCredential(t *testing.T) {
	creds := t.TempDir()
	writeFile(t, creds, "postgres.dsn", "test-dsn-from-credential")
	envFile := writeFile(t, t.TempDir(), "dsn", "test-dsn-from-env-file")

	r := secret.New(creds, envOf(map[string]string{"POSTGRES_DSN_FILE": envFile}))
	got, err := r.Value("postgres.dsn", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "test-dsn-from-env-file" {
		t.Errorf("want value from POSTGRES_DSN_FILE, got %q", got)
	}
}

func TestPathValue(t *testing.T) {
	creds := t.TempDir()
	keyPath := writeFile(t, creds, "service.conn.key", "test-only-not-a-real-key")
	r := secret.New(creds, nil)

	got, err := r.PathValue("service.conn.key", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != keyPath {
		t.Errorf("unset key: want %q, got %q", keyPath, got)
	}

	got, err = r.PathValue("service.conn.client.key", "/etc/test/client.key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/etc/test/client.key" {
		t.Errorf("plain path: want it unchanged, got %q", got)
	}

	if _, err := r.PathValue("service.conn.key", "/etc/test/server.key"); !errors.Is(err, secret.ErrConflict) {
		t.Errorf("path shadowing a credential: want ErrConflict, got %v", err)
	}
}

func TestFromEnv(t *testing.T) {
	creds := t.TempDir()
	writeFile(t, creds, "service.secret_key", "test-secret-key")
	t.Setenv(secret.CredentialsDirEnv, creds)

	got, err := secret.FromEnv().Value("service.secret_key", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "test-secret-key" {
		t.Errorf("want test-secret-key, got %q", got)
	}
}
