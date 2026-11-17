package presign_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/webitel/webitel-go-kit/pkg/presign"
)

const (
	testKeyring = "presign-test-phrase-2026!"
	testText    = "https://example.org/files/2026/08/03/image_20260803_153728.jpg"
	testID      = int64(56843)
)

func TestMain(m *testing.M) {
	if os.Getenv("CRYPTO_CIPHER_KEYRING") == "" && os.Getenv("CRYPTO_CIPHER_KEYFILE") == "" {
		os.Setenv("CRYPTO_CIPHER_KEYRING", testKeyring)
	}

	m.Run()
}

func legacyKeyfile(t *testing.T) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	data := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	path := filepath.Join(t.TempDir(), "key.pem")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func legacy(t *testing.T) presign.PreSign {
	t.Helper()
	return legacyFrom(t, legacyKeyfile(t))
}

func legacyFrom(t *testing.T, path string) presign.PreSign {
	t.Helper()

	impl, err := presign.NewPrivateKey(path)
	if err != nil {
		t.Fatalf("legacy: configuration failed: %v", err)
	}
	return impl
}

func modern(t *testing.T) presign.PreSign {
	t.Helper()

	impl, err := presign.NewCryptoBox()
	if err != nil {
		t.Fatalf("modern: configuration failed: %v", err)
	}
	return impl
}

func latest(t *testing.T, pemLocation string) presign.PreSign {
	t.Helper()

	impl, err := presign.NewPreSignChain(pemLocation)
	if err != nil {
		t.Fatalf("latest: configuration failed: %v", err)
	}
	return impl
}

func checkSignature(t *testing.T, impl presign.PreSign) {
	t.Helper()

	sign, err := impl.Generate([]byte(testText))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !impl.Valid(testText, sign) {
		t.Error("Valid: signature rejected")
	}
	if impl.Valid(testText+"x", sign) {
		t.Error("Valid: signature accepted for other text")
	}
}

func checkBytes(t *testing.T, impl presign.PreSign) {
	t.Helper()

	blob, err := impl.EncryptBytes([]byte(testText))
	if err != nil {
		t.Fatalf("EncryptBytes: %v", err)
	}
	got, err := impl.DecryptBytes(blob)
	if err != nil {
		t.Fatalf("DecryptBytes: %v", err)
	}
	if !bytes.Equal(got, []byte(testText)) {
		t.Error("DecryptBytes: unexpected data")
	}
}

func checkID(t *testing.T, impl presign.PreSign) {
	t.Helper()

	token, err := impl.EncryptId(testID)
	if err != nil {
		t.Fatalf("EncryptId: %v", err)
	}
	got, err := impl.DecryptId(token)
	if err != nil {
		t.Fatalf("DecryptId: %v", err)
	}
	if got != testID {
		t.Errorf("DecryptId = %d, want %d", got, testID)
	}
}
