package appconfig_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/webitel/webitel-go-kit/appconfig"
	"github.com/webitel/webitel-go-kit/appconfig/secret"
)

type secretsConfig struct {
	Service struct {
		SecretKey  string             `mapstructure:"secret_key"`
		Connection appconfig.GRPCConn `mapstructure:"conn"`
	} `mapstructure:"service"`
	Postgres appconfig.Postgres `mapstructure:"postgres"`
	Redis    appconfig.Redis    `mapstructure:"redis"`
	Pubsub   appconfig.Pubsub   `mapstructure:"pubsub"`
}

// clearSecretEnv isolates a test from secret-related variables in the shell.
func clearSecretEnv(t *testing.T) {
	t.Helper()
	for _, env := range []string{
		secret.CredentialsDirEnv,
		"POSTGRES_DSN", "POSTGRES_DSN_FILE",
		"POSTGRES_STANDBY_DSN", "POSTGRES_STANDBY_DSN_FILE",
		"REDIS_PASSWORD", "REDIS_PASSWORD_FILE",
		"PUBSUB_URL", "PUBSUB_URL_FILE",
		"SERVICE_SECRET_KEY", "SERVICE_SECRET_KEY_FILE",
		"SERVICE_CONN_KEY", "SERVICE_CONN_CLIENT_KEY",
	} {
		t.Setenv(env, "")
		os.Unsetenv(env)
	}
}

// credentialsDir emulates the directory systemd exposes via
// $CREDENTIALS_DIRECTORY and fills it with the given credentials.
func credentialsDir(t *testing.T, creds map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, value := range creds {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value+"\n"), 0o400); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv(secret.CredentialsDirEnv, dir)
	return dir
}

func newSecretsLoader(fs *pflag.FlagSet) *appconfig.Loader {
	loader := appconfig.NewLoader(appconfig.Sections{Postgres: true, Redis: true, Pubsub: true})
	loader.RegisterFlags(fs)
	fs.String("service.secret_key", "", "test secret")
	appconfig.RegisterGRPCConnFlags(fs, "service.conn", false)
	return loader.
		MarkSecret("service.secret_key").
		MarkSecretPath(appconfig.GRPCConnSecretPaths("service.conn")...)
}

func loadSecrets(t *testing.T, args ...string) (*appconfig.Loader, secretsConfig, error) {
	t.Helper()
	fs := newFlagSet()
	loader := newSecretsLoader(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}
	var cfg secretsConfig
	err := loader.Load(fs, &cfg)
	return loader, cfg, err
}

// ── systemd credentials ──────────────────────────────────────────────────────

func TestSecretsFromSystemdCredentials(t *testing.T) {
	clearSecretEnv(t)
	dir := credentialsDir(t, map[string]string{
		"postgres.dsn":       "postgres://db.example.com/test",
		"redis.password":     "test-redis-password",
		"pubsub.url":         "amqp://mq.example.com/",
		"service.secret_key": "test-secret-key",
		"service.conn.key":   "test-only-not-a-real-key",
	})

	_, cfg, err := loadSecrets(t)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Postgres.DSN != "postgres://db.example.com/test" {
		t.Errorf("postgres.dsn: got %q", cfg.Postgres.DSN)
	}
	if cfg.Redis.Password != "test-redis-password" {
		t.Errorf("redis.password: got %q", cfg.Redis.Password)
	}
	if cfg.Pubsub.URL != "amqp://mq.example.com/" {
		t.Errorf("pubsub.url: got %q", cfg.Pubsub.URL)
	}
	if cfg.Service.SecretKey != "test-secret-key" {
		t.Errorf("service.secret_key: got %q", cfg.Service.SecretKey)
	}
	if want := filepath.Join(dir, "service.conn.key"); cfg.Service.Connection.Key != want {
		t.Errorf("service.conn.key: want path %q, got %q", want, cfg.Service.Connection.Key)
	}
	if cfg.Service.Connection.Client.Key != "" {
		t.Errorf("service.conn.client.key: want empty without credential, got %q", cfg.Service.Connection.Client.Key)
	}
}

func TestSecretReferencesInConfigFile(t *testing.T) {
	clearSecretEnv(t)
	dir := credentialsDir(t, map[string]string{
		"im-account.postgres": "postgres://db.example.com/test",
		"im-account.tls.key":  "test-only-not-a-real-key",
	})
	tokenFile := filepath.Join(t.TempDir(), "secret_key")
	if err := os.WriteFile(tokenFile, []byte("test-secret-key\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	yaml := "postgres:\n  dsn: cred:im-account.postgres\n" +
		"service:\n  secret_key: file:" + tokenFile + "\n" +
		"  conn:\n    key: cred:im-account.tls.key\n" +
		"redis:\n  password: test-plain-password\n"
	cfgFile := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(cfgFile, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	loader, cfg, err := loadSecrets(t, "--config_file="+cfgFile)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Postgres.DSN != "postgres://db.example.com/test" {
		t.Errorf("postgres.dsn: got %q", cfg.Postgres.DSN)
	}
	if cfg.Service.SecretKey != "test-secret-key" {
		t.Errorf("service.secret_key: got %q", cfg.Service.SecretKey)
	}
	if want := filepath.Join(dir, "im-account.tls.key"); cfg.Service.Connection.Key != want {
		t.Errorf("service.conn.key: want %q, got %q", want, cfg.Service.Connection.Key)
	}
	if cfg.Redis.Password != "test-plain-password" {
		t.Errorf("redis.password: plain value must stay as is, got %q", cfg.Redis.Password)
	}

	// Services reload config in a Watch callback via loader.Viper().Unmarshal;
	// it must see resolved secrets, not the raw references.
	var reloaded secretsConfig
	if err := loader.Viper().Unmarshal(&reloaded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if reloaded.Postgres.DSN != cfg.Postgres.DSN {
		t.Errorf("reload: postgres.dsn: want resolved value, got %q", reloaded.Postgres.DSN)
	}
}

// ── <ENV>_FILE ───────────────────────────────────────────────────────────────

func TestSecretFromEnvFile(t *testing.T) {
	clearSecretEnv(t)
	p := filepath.Join(t.TempDir(), "pg_dsn")
	if err := os.WriteFile(p, []byte("postgres://db.example.com/test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSTGRES_DSN_FILE", p)

	_, cfg, err := loadSecrets(t)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Postgres.DSN != "postgres://db.example.com/test" {
		t.Errorf("postgres.dsn: got %q", cfg.Postgres.DSN)
	}
}

// ── Backward compatibility and errors ────────────────────────────────────────

func TestPlainSecretsStillWork(t *testing.T) {
	clearSecretEnv(t)
	t.Setenv("POSTGRES_DSN", "postgres://db.example.com/test")

	_, cfg, err := loadSecrets(t, "--redis.password=test-redis-password")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Postgres.DSN != "postgres://db.example.com/test" {
		t.Errorf("postgres.dsn: got %q", cfg.Postgres.DSN)
	}
	if cfg.Redis.Password != "test-redis-password" {
		t.Errorf("redis.password: got %q", cfg.Redis.Password)
	}
}

func TestPlainSecretShadowingCredentialFails(t *testing.T) {
	clearSecretEnv(t)
	credentialsDir(t, map[string]string{"postgres.dsn": "postgres://db.example.com/new"})
	t.Setenv("POSTGRES_DSN", "postgres://db.example.com/old")

	_, _, err := loadSecrets(t)
	if !errors.Is(err, secret.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
	if strings.Contains(err.Error(), "db.example.com") {
		t.Errorf("error leaks the secret: %v", err)
	}
}

// loadDefaultedToken loads a key whose flag has a non-empty default.
func loadDefaultedToken(t *testing.T, args ...string) (string, error) {
	t.Helper()
	fs := newFlagSet()
	loader := appconfig.NewLoader(appconfig.Sections{})
	loader.RegisterFlags(fs)
	fs.String("service.api_token", "test-token-default", "test secret with a default")
	loader.MarkSecret("service.api_token")
	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Service struct {
			APIToken string `mapstructure:"api_token"`
		} `mapstructure:"service"`
	}
	err := loader.Load(fs, &cfg)
	return cfg.Service.APIToken, err
}

func TestSecretDefaultIsNotAConflict(t *testing.T) {
	clearSecretEnv(t)
	t.Setenv("SERVICE_API_TOKEN", "")
	os.Unsetenv("SERVICE_API_TOKEN")

	got, err := loadDefaultedToken(t)
	if err != nil || got != "test-token-default" {
		t.Fatalf("no credential: want the default, got %q, %v", got, err)
	}

	credentialsDir(t, map[string]string{"service.api_token": "test-token-cred"})
	got, err = loadDefaultedToken(t)
	if err != nil || got != "test-token-cred" {
		t.Fatalf("credential: want it to replace the default, got %q, %v", got, err)
	}

	_, err = loadDefaultedToken(t, "--service.api_token=test-token-flag")
	if !errors.Is(err, secret.ErrConflict) {
		t.Fatalf("explicit flag and credential: want ErrConflict, got %v", err)
	}
}

func TestMarkSecretIsCaseInsensitive(t *testing.T) {
	clearSecretEnv(t)
	credentialsDir(t, map[string]string{"service.secret_key": "test-token-secret"})

	fs := newFlagSet()
	loader := appconfig.NewLoader(appconfig.Sections{})
	loader.RegisterFlags(fs)
	fs.String("service.secret_key", "", "test secret")
	loader.MarkSecret("Service.Secret_Key")
	_ = fs.Parse(nil)

	var cfg secretsConfig
	if err := loader.Load(fs, &cfg); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Service.SecretKey != "test-token-secret" {
		t.Errorf("service.secret_key: got %q", cfg.Service.SecretKey)
	}
}

func TestCredentialReferenceWithoutSystemd(t *testing.T) {
	clearSecretEnv(t)

	_, _, err := loadSecrets(t, "--postgres.dsn=cred:postgres.dsn")
	if !errors.Is(err, secret.ErrNoCredentialsDir) {
		t.Fatalf("want ErrNoCredentialsDir, got %v", err)
	}
}

func TestSecretsOnlyForEnabledSections(t *testing.T) {
	clearSecretEnv(t)
	credentialsDir(t, map[string]string{"redis.password": "test-redis-password"})

	loader := appconfig.NewLoader(appconfig.Sections{Postgres: true})
	fs := newFlagSet()
	loader.RegisterFlags(fs)
	_ = fs.Parse(nil)

	var cfg secretsConfig
	if err := loader.Load(fs, &cfg); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Redis.Password != "" {
		t.Errorf("redis section is disabled; want no password, got %q", cfg.Redis.Password)
	}
}

func TestGRPCConnSecretPaths(t *testing.T) {
	got := appconfig.GRPCConnSecretPaths("service.conn")
	want := []string{"service.conn.key", "service.conn.client.key"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("want %v, got %v", want, got)
	}
}

// ── postgres.standby_dsn (string slice secrets) ──────────────────────────────

func TestStandbyDSNSecretFromCredential(t *testing.T) {
	clearSecretEnv(t)
	credentialsDir(t, map[string]string{
		"postgres.standby_dsn": "postgres://replica.example.com/test",
	})

	_, cfg, err := loadSecrets(t)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := []string{"postgres://replica.example.com/test"}
	if !slices.Equal(cfg.Postgres.StandbyDSN, want) {
		t.Errorf("postgres.standby_dsn = %#v, want %#v", cfg.Postgres.StandbyDSN, want)
	}
}

func TestStandbyDSNSecretReferencesInConfigFile(t *testing.T) {
	clearSecretEnv(t)
	credentialsDir(t, map[string]string{
		"replica1.dsn": "postgres://r1.example.com/test",
		"replica2.dsn": "postgres://r2.example.com/test",
	})

	yaml := "postgres:\n  standby_dsn:\n    - cred:replica1.dsn\n    - cred:replica2.dsn\n"
	cfgFile := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(cfgFile, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	_, cfg, err := loadSecrets(t, "--config_file="+cfgFile)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := []string{"postgres://r1.example.com/test", "postgres://r2.example.com/test"}
	if !slices.Equal(cfg.Postgres.StandbyDSN, want) {
		t.Errorf("postgres.standby_dsn = %#v, want %#v", cfg.Postgres.StandbyDSN, want)
	}
}

func TestStandbyDSNPlainLiteralsStillWork(t *testing.T) {
	clearSecretEnv(t)

	_, cfg, err := loadSecrets(t,
		"--postgres.standby_dsn=postgres://r1.example.com/test",
		"--postgres.standby_dsn=postgres://r2.example.com/test",
	)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := []string{"postgres://r1.example.com/test", "postgres://r2.example.com/test"}
	if !slices.Equal(cfg.Postgres.StandbyDSN, want) {
		t.Errorf("postgres.standby_dsn = %#v, want %#v", cfg.Postgres.StandbyDSN, want)
	}
}

func TestStandbyDSNPlainShadowingCredentialFails(t *testing.T) {
	clearSecretEnv(t)
	credentialsDir(t, map[string]string{
		"postgres.standby_dsn": "postgres://replica.example.com/new",
	})

	_, _, err := loadSecrets(t, "--postgres.standby_dsn=postgres://replica.example.com/old")
	if !errors.Is(err, secret.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
	if strings.Contains(err.Error(), "replica.example.com") {
		t.Errorf("error leaks the secret: %v", err)
	}
}

func TestStandbyDSNSecretFromEnvFile(t *testing.T) {
	clearSecretEnv(t)
	p := filepath.Join(t.TempDir(), "standby_dsn")
	if err := os.WriteFile(p, []byte("postgres://replica.example.com/test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSTGRES_STANDBY_DSN_FILE", p)

	_, cfg, err := loadSecrets(t)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := []string{"postgres://replica.example.com/test"}
	if !slices.Equal(cfg.Postgres.StandbyDSN, want) {
		t.Errorf("postgres.standby_dsn = %#v, want %#v", cfg.Postgres.StandbyDSN, want)
	}
}
