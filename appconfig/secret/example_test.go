package secret_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/webitel/webitel-go-kit/appconfig/secret"
)

// A service that does not use appconfig.Loader resolves its secrets directly.
// Under systemd, CREDENTIALS_DIRECTORY is set by the unit's LoadCredential*=
// settings; here a temporary directory stands in for it.
func ExampleResolver_Value() {
	dir, _ := os.MkdirTemp("", "creds")
	defer os.RemoveAll(dir)
	_ = os.WriteFile(filepath.Join(dir, "service.token"), []byte("test-token-example\n"), 0o400)
	os.Setenv(secret.CredentialsDirEnv, dir)
	defer os.Unsetenv(secret.CredentialsDirEnv)

	r := secret.FromEnv()

	// Unset in config: picked up from the credential named after the key.
	token, err := r.Value("service.token", "")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(token)

	// A literal is still accepted.
	plain, _ := r.Value("service.other_token", "test-token-literal")
	fmt.Println(plain)
	// Output:
	// test-token-example
	// test-token-literal
}

// Settings that expect a file path, such as a TLS key, resolve to the path.
func ExampleResolver_Path() {
	r := secret.New("/run/credentials/webitel-im-account.service", nil)
	p, _ := r.Path("cred:tls.key")
	fmt.Println(p)
	// Output: /run/credentials/webitel-im-account.service/tls.key
}
