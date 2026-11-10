// Package secret resolves sensitive configuration values — passwords, tokens,
// DSNs, private keys — from sources that keep them out of plain-text config
// files and process environments:
//
//   - systemd credentials (LoadCredential=, LoadCredentialEncrypted=,
//     SetCredentialEncrypted=, ImportCredential=), exposed to the service as
//     files under $CREDENTIALS_DIRECTORY;
//   - arbitrary files, e.g. Docker or Kubernetes secrets mounted under
//     /run/secrets.
//
// A value is either a literal (returned unchanged, for backward compatibility)
// or a reference:
//
//	cred:<name>   contents of $CREDENTIALS_DIRECTORY/<name>
//	file:<path>   contents of <path>
//
// For a config key without an explicit value, [Resolver.Lookup] applies two
// conventions in order:
//
//	<ENV>_FILE                        e.g. POSTGRES_DSN_FILE=/run/secrets/pg_dsn
//	$CREDENTIALS_DIRECTORY/<key>      e.g. a systemd credential named postgres.dsn
//
// The package has no dependencies beyond the standard library, so it can be
// used directly by services that do not use appconfig.Loader.
package secret

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CredentialsDirEnv is the environment variable systemd sets to the directory
// holding the unit's decrypted credentials (see systemd.exec(5)).
const CredentialsDirEnv = "CREDENTIALS_DIRECTORY"

// Reference prefixes understood by [Resolver.Resolve] and [Resolver.Path].
const (
	PrefixCred = "cred:"
	PrefixFile = "file:"
)

// FileEnvSuffix is appended to a key's env var name to point at a file
// holding its value (POSTGRES_DSN → POSTGRES_DSN_FILE).
const FileEnvSuffix = "_FILE"

// MaxSize caps how much a single secret file may hold. It matches the systemd
// credential size limit and protects against pointing a key at a large file
// by mistake.
const MaxSize = 1 << 20

var (
	// ErrNoCredentialsDir is returned for a cred: reference when the process
	// was not started with systemd credentials.
	ErrNoCredentialsDir = errors.New("secret: " + CredentialsDirEnv + " is not set; is the service started by systemd with LoadCredential*=?")

	// ErrEmpty is returned when a referenced secret resolves to an empty value.
	ErrEmpty = errors.New("secret: value is empty")

	// ErrConflict is returned by [Resolver.Value] when a key has both an
	// explicit value and a conventional secret source.
	ErrConflict = errors.New("secret: value is set more than once")
)

// Resolver reads secrets from systemd credentials and files.
// The zero value is not usable; construct it with [New] or [FromEnv].
type Resolver struct {
	credentialsDir string
	lookupEnv      func(string) (string, bool)
}

// FromEnv returns a Resolver bound to the current process environment.
func FromEnv() *Resolver {
	return New(os.Getenv(CredentialsDirEnv), os.LookupEnv)
}

// New returns a Resolver that reads credentials from credentialsDir (may be
// empty when systemd credentials are not in use) and env vars via lookupEnv.
func New(credentialsDir string, lookupEnv func(string) (string, bool)) *Resolver {
	if lookupEnv == nil {
		lookupEnv = func(string) (string, bool) { return "", false }
	}
	return &Resolver{credentialsDir: credentialsDir, lookupEnv: lookupEnv}
}

// IsRef reports whether v is a cred: or file: reference.
func IsRef(v string) bool {
	return strings.HasPrefix(v, PrefixCred) || strings.HasPrefix(v, PrefixFile)
}

// EnvName returns the env var name viper's AutomaticEnv uses for key
// ("postgres.dsn" → "POSTGRES_DSN").
func EnvName(key string) string {
	return strings.ToUpper(strings.NewReplacer(".", "_", "-", "_").Replace(key))
}

// Resolve returns the plaintext a value stands for: the contents of the
// referenced credential or file, or v itself when it is not a reference.
// Trailing CR/LF characters are trimmed from file contents.
func (r *Resolver) Resolve(v string) (string, error) {
	if !IsRef(v) {
		return v, nil
	}
	path, err := r.Path(v)
	if err != nil {
		return "", err
	}
	return readFile(path)
}

// Path returns the filesystem path a value stands for, without reading it.
// Use it for settings that expect a path, such as a TLS private key:
// cred:<name> becomes $CREDENTIALS_DIRECTORY/<name>, file:<path> becomes
// <path>, and anything else is returned unchanged.
func (r *Resolver) Path(v string) (string, error) {
	switch {
	case strings.HasPrefix(v, PrefixCred):
		return r.credentialPath(strings.TrimPrefix(v, PrefixCred))
	case strings.HasPrefix(v, PrefixFile):
		p := strings.TrimPrefix(v, PrefixFile)
		if p == "" {
			return "", fmt.Errorf("secret: %q: empty file path", v)
		}
		return p, nil
	default:
		return v, nil
	}
}

// Source describes where [Resolver.Lookup] found a secret.
type Source struct {
	Kind string // "env-file" or "credential"
	Name string // env var name or credential name
	Path string // file the secret was read from
}

func (s Source) String() string {
	return s.Kind + " " + s.Name
}

// Lookup finds the conventional secret source for a config key that has no
// explicit value: the file named by <ENV>_FILE, then the systemd credential
// named key. It returns the source's path without reading it; ok is false
// when neither exists.
func (r *Resolver) Lookup(key string) (src Source, ok bool, err error) {
	envName := EnvName(key) + FileEnvSuffix
	if p, set := r.lookupEnv(envName); set && p != "" {
		return Source{Kind: "env-file", Name: envName, Path: p}, true, nil
	}
	if r.credentialsDir == "" {
		return Source{}, false, nil
	}
	p, err := r.credentialPath(key)
	if err != nil {
		return Source{}, false, err
	}
	switch _, err := os.Stat(p); {
	case err == nil:
		return Source{Kind: "credential", Name: key, Path: p}, true, nil
	case errors.Is(err, os.ErrNotExist):
		return Source{}, false, nil
	default:
		return Source{}, false, fmt.Errorf("secret: credential %q: %w", key, err)
	}
}

// Value resolves the final plaintext for config key given its configured
// value v (from a flag, env var or config file):
//
//   - a cred:/file: reference is read;
//   - a literal is returned unchanged;
//   - an empty value falls back to [Resolver.Lookup].
//
// Configuring a key both explicitly and through a conventional source is
// reported as [ErrConflict] rather than silently picking one, so a forgotten
// plain-text value cannot shadow the credential that replaced it.
func (r *Resolver) Value(key, v string) (string, error) {
	return r.resolve(key, v, false)
}

// PathValue is [Resolver.Value] for settings that hold a file path (TLS keys):
// it returns the path of the referenced or conventional source instead of its
// contents.
func (r *Resolver) PathValue(key, v string) (string, error) {
	return r.resolve(key, v, true)
}

func (r *Resolver) resolve(key, v string, asPath bool) (string, error) {
	src, found, err := r.Lookup(key)
	if err != nil {
		return "", err
	}

	var path string
	switch {
	case v == "" && !found:
		return "", nil
	case v == "":
		path = src.Path
	case !IsRef(v):
		if found {
			return "", conflict(key, src)
		}
		return v, nil
	default:
		if path, err = r.Path(v); err != nil {
			return "", fmt.Errorf("%s: %w", key, err)
		}
		// An explicit reference to the conventional source itself
		// (postgres.dsn: cred:postgres.dsn) is not a conflict.
		if found && filepath.Clean(path) != filepath.Clean(src.Path) {
			return "", conflict(key, src)
		}
	}

	if asPath {
		return path, nil
	}
	s, err := readFile(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", key, err)
	}
	return s, nil
}

func conflict(key string, src Source) error {
	return fmt.Errorf("%w: %s is configured and %s is also present; remove one of them", ErrConflict, key, src)
}

func (r *Resolver) credentialPath(name string) (string, error) {
	if r.credentialsDir == "" {
		return "", ErrNoCredentialsDir
	}
	// systemd credential names are plain file names; reject anything that
	// could escape the credentials directory.
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("secret: invalid credential name %q", name)
	}
	return filepath.Join(r.credentialsDir, name), nil
}

// readFile reads a secret file. Errors carry the path but never the contents.
func readFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("secret: %w", err)
	}
	defer f.Close()

	b, err := io.ReadAll(io.LimitReader(f, MaxSize+1))
	if err != nil {
		return "", fmt.Errorf("secret: read %s: %w", path, err)
	}
	if len(b) > MaxSize {
		return "", fmt.Errorf("secret: %s is larger than %d bytes", path, MaxSize)
	}
	s := strings.TrimRight(string(b), "\r\n")
	if s == "" {
		return "", fmt.Errorf("%w: %s", ErrEmpty, path)
	}
	return s, nil
}
