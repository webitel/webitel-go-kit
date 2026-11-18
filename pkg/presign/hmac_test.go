package presign_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/webitel/crypto/cryptobox"
	"github.com/webitel/webitel-go-kit/pkg/presign"
)

const macInput = "/sys/media/42/stream?domain_id=1&expires=1767225600000"

func macKeyring(t *testing.T, spec string) *cryptobox.Keyring {
	t.Helper()
	keys, err := cryptobox.ParseKeyring(spec)
	if err != nil {
		t.Fatalf("ParseKeyring: %v", err)
	}
	return keys
}

func macSigner(t *testing.T, keys *cryptobox.Keyring) presign.Signer {
	t.Helper()
	signer, err := presign.NewMACKeyring(keys)
	if err != nil {
		t.Fatalf("NewMACKeyring: %v", err)
	}
	return signer
}

func macGenerate(t *testing.T, signer presign.Signer, data string) string {
	t.Helper()
	sig, err := signer.Generate([]byte(data))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return sig
}

func TestMACDeterministic(t *testing.T) {
	signer := macSigner(t, macKeyring(t, "first-test-phrase-2026!"))

	first := macGenerate(t, signer, macInput)
	second := macGenerate(t, signer, macInput)
	if first != second {
		t.Fatalf("same input gave different signatures: %s != %s", first, second)
	}
	if other := macGenerate(t, signer, macInput+"&x=1"); other == first {
		t.Fatalf("different input gave same signature %s", first)
	}
	if len(first) != 66 {
		t.Fatalf("signature length = %d, want 66 (1 byte kid + 32 bytes sum, hex)", len(first))
	}
	if !strings.HasPrefix(first, "01") {
		t.Fatalf("signature %s must start with kid 01", first)
	}
}

func TestMACStableAcrossInstances(t *testing.T) {
	const spec = "first-test-phrase-2026!"
	a := macGenerate(t, macSigner(t, macKeyring(t, spec)), macInput)
	b := macGenerate(t, macSigner(t, macKeyring(t, spec)), macInput)
	if a != b {
		t.Fatalf("same keyring gave different signatures: %s != %s", a, b)
	}
}

func TestMACValid(t *testing.T) {
	signer := macSigner(t, macKeyring(t, "first-test-phrase-2026!"))
	sig := macGenerate(t, signer, macInput)

	if !signer.Valid(macInput, sig) {
		t.Fatalf("Valid(%q) = false", sig)
	}

	flipLast := sig[:len(sig)-1] + "0"
	if sig[len(sig)-1] == '0' {
		flipLast = sig[:len(sig)-1] + "1"
	}

	invalid := map[string]struct{ text, sig string }{
		"tampered text":  {macInput + "&x=1", sig},
		"tampered sum":   {macInput, flipLast},
		"truncated sum":  {macInput, sig[:len(sig)-2]},
		"extended sum":   {macInput, sig + "00"},
		"unknown kid":    {macInput, "02" + sig[2:]},
		"kid only":       {macInput, sig[:2]},
		"empty":          {macInput, ""},
		"not hex":        {macInput, "zz" + sig[2:]},
		"upper overflow": {macInput, "ffffffff0f" + sig[2:]},
	}
	for name, tc := range invalid {
		t.Run(name, func(t *testing.T) {
			if signer.Valid(tc.text, tc.sig) {
				t.Fatalf("Valid(%q, %q) = true", tc.text, tc.sig)
			}
		})
	}
}

func TestMACOtherKeyRejected(t *testing.T) {
	sig := macGenerate(t, macSigner(t, macKeyring(t, "first-test-phrase-2026!")), macInput)
	other := macSigner(t, macKeyring(t, "other-test-phrase-2026!"))
	if other.Valid(macInput, sig) {
		t.Fatal("signature from another key material must be rejected")
	}
}

func TestMACRotation(t *testing.T) {
	keys := macKeyring(t, "first-test-phrase-2026!")
	signer := macSigner(t, keys)
	old := macGenerate(t, signer, macInput)

	next, err := cryptobox.NewTextKey(2, "second-test-phrase-2026!")
	if err != nil {
		t.Fatalf("NewTextKey: %v", err)
	}
	if err := keys.Rotate(next); err != nil {
		t.Fatalf("Rotate: %v", err)
	}

	fresh := macGenerate(t, signer, macInput)
	if !strings.HasPrefix(fresh, "02") {
		t.Fatalf("signature after rotation %s must start with kid 02", fresh)
	}
	if !signer.Valid(macInput, old) {
		t.Fatal("signature from previous key must stay valid after rotation")
	}
	if !signer.Valid(macInput, fresh) {
		t.Fatal("signature from new primary key must be valid")
	}

	if err := keys.Retire(1); err != nil {
		t.Fatalf("Retire: %v", err)
	}
	if signer.Valid(macInput, old) {
		t.Fatal("signature from retired key must be rejected")
	}
}

func TestMACKeyringReload(t *testing.T) {
	keys := macKeyring(t, "first-test-phrase-2026!")
	signer := macSigner(t, keys)
	old := macGenerate(t, signer, macInput)

	if err := keys.Load("replaced-test-phrase-2026!"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if signer.Valid(macInput, old) {
		t.Fatal("signature from replaced material under same kid must be rejected")
	}
	fresh := macGenerate(t, signer, macInput)
	if fresh == old {
		t.Fatal("signature must change after key material reload")
	}
	if !signer.Valid(macInput, fresh) {
		t.Fatal("signature from reloaded material must be valid")
	}
}

func TestNewMACKeyringNil(t *testing.T) {
	if _, err := presign.NewMACKeyring(nil); !errors.Is(err, presign.ErrNoKeyring) {
		t.Fatalf("NewMACKeyring(nil) error = %v, want %v", err, presign.ErrNoKeyring)
	}
}

func TestNewMACDefaultKeys(t *testing.T) {
	signer, err := presign.NewMAC()
	if err != nil {
		t.Fatalf("NewMAC: %v", err)
	}
	keys, err := cryptobox.DefaultKeys()
	if err != nil {
		t.Fatalf("DefaultKeys: %v", err)
	}
	sig := macGenerate(t, signer, macInput)
	if !macSigner(t, keys).Valid(macInput, sig) {
		t.Fatal("NewMAC must sign with the default keyring")
	}
}
