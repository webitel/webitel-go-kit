package presign_test

import "testing"

func TestPreSignedModernOnly(t *testing.T) {
	impl := latest(t, "")
	checkSignature(t, impl)
	checkBytes(t, impl)
	checkID(t, impl)
}

func TestPreSignedWithLegacy(t *testing.T) {
	path := legacyKeyfile(t)
	impl := latest(t, path)
	checkSignature(t, impl)
	checkBytes(t, impl)
	checkID(t, impl)
}

func TestPreSignedAcceptsLegacyOutput(t *testing.T) {
	path := legacyKeyfile(t)
	impl := latest(t, path)
	old := legacyFrom(t, path)

	sign, err := old.Generate([]byte(testText))
	if err != nil {
		t.Fatalf("legacy Generate: %v", err)
	}
	if !impl.Valid(testText, sign) {
		t.Error("Valid: legacy signature rejected")
	}

	blob, err := old.EncryptBytes([]byte(testText))
	if err != nil {
		t.Fatalf("legacy EncryptBytes: %v", err)
	}
	got, err := impl.DecryptBytes(blob)
	if err != nil {
		t.Fatalf("DecryptBytes: %v", err)
	}
	if string(got) != testText {
		t.Error("DecryptBytes: unexpected data for legacy blob")
	}

	token, err := old.EncryptId(testID)
	if err != nil {
		t.Fatalf("legacy EncryptId: %v", err)
	}
	id, err := impl.DecryptId(token)
	if err != nil {
		t.Fatalf("DecryptId: %v", err)
	}
	if id != testID {
		t.Errorf("DecryptId = %d, want %d", id, testID)
	}
}

func TestPreSignedEncodesWithModern(t *testing.T) {
	path := legacyKeyfile(t)
	impl := latest(t, path)
	old := legacyFrom(t, path)
	box := modern(t)

	sign, err := impl.Generate([]byte(testText))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !box.Valid(testText, sign) {
		t.Error("Generate: signature not produced by cryptobox")
	}
	if old.Valid(testText, sign) {
		t.Error("Generate: signature accepted by legacy key")
	}

	blob, err := impl.EncryptBytes([]byte(testText))
	if err != nil {
		t.Fatalf("EncryptBytes: %v", err)
	}
	if got, err := box.DecryptBytes(blob); err != nil || string(got) != testText {
		t.Errorf("EncryptBytes: cryptobox decrypt = %q, %v", got, err)
	}
	if _, err := old.DecryptBytes(blob); err == nil {
		t.Error("EncryptBytes: blob decrypted by legacy key")
	}
}
