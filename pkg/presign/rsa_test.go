package presign_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/webitel/webitel-go-kit/pkg/presign"
)

func TestRSASignature(t *testing.T) { checkSignature(t, legacy(t)) }

func TestRSABytes(t *testing.T) { checkBytes(t, legacy(t)) }

func TestRSAID(t *testing.T) { checkID(t, legacy(t)) }

func TestNewPrivateKeyInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.pem")
	if err := os.WriteFile(path, []byte("not a pem"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := presign.NewPrivateKey(path); err == nil {
		t.Error("NewPrivateKey: expected error for invalid PEM")
	}
	if _, err := presign.NewPrivateKey(filepath.Join(t.TempDir(), "missing.pem")); err == nil {
		t.Error("NewPrivateKey: expected error for missing file")
	}
}
