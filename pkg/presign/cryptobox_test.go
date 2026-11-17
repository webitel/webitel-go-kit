package presign_test

import "testing"

func TestCryptoBoxSignature(t *testing.T) { checkSignature(t, modern(t)) }

func TestCryptoBoxBytes(t *testing.T) { checkBytes(t, modern(t)) }

func TestCryptoBoxID(t *testing.T) { checkID(t, modern(t)) }

func TestCryptoBoxDecryptIdEmpty(t *testing.T) {
	impl := modern(t)

	blob, err := impl.EncryptBytes(nil)
	if err != nil {
		t.Fatalf("EncryptBytes: %v", err)
	}
	if _, err := impl.DecryptId(string(blob)); err == nil {
		t.Error("DecryptId: expected error for empty payload")
	}
}
