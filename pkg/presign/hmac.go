package presign

import (
	"context"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math"
	"sync"

	"github.com/webitel/crypto/cryptobox"
)

const (
	macInfo    = "webitel.presign.v1.hmac-sha256"
	macKeySize = sha256.Size
	macSigSize = binary.MaxVarintLen16 + sha256.Size
)

// ErrNoKeyring is returned when a nil keyring is given.
var ErrNoKeyring = errors.New("presign: keyring is required")

// MAC is a deterministic HMAC-SHA256 [Signer] keyed from a cryptobox keyring.
type MAC struct {
	keys    *cryptobox.Keyring
	mu      sync.RWMutex
	version uint64
	secrets map[cryptobox.KID][]byte
}

var _ Signer = (*MAC)(nil)

// NewMAC returns a MAC keyed from the default cryptobox keyring.
func NewMAC() (*MAC, error) {
	keys, err := cryptobox.DefaultKeys()
	if err != nil {
		return nil, err
	}
	return NewMACKeyring(keys)
}

// NewMACKeyring returns a MAC keyed from keys.
func NewMACKeyring(keys *cryptobox.Keyring) (*MAC, error) {
	if keys == nil {
		return nil, ErrNoKeyring
	}
	m := &MAC{keys: keys}
	if _, err := m.secret(context.Background(), keys.Version(), keys.Primary()); err != nil {
		return nil, err
	}
	return m, nil
}

// Generate returns the hex-encoded signature of data under the primary key.
func (m *MAC) Generate(data []byte) (string, error) {
	key := m.keys.Primary()
	secret, err := m.secret(context.Background(), m.keys.Version(), key)
	if err != nil {
		return "", err
	}
	sig := make([]byte, 0, macSigSize)
	sig = binary.AppendUvarint(sig, uint64(key.Kid()))
	sig = macSum(sig, secret, data)
	return hex.EncodeToString(sig), nil
}

// Valid reports whether signature matches plaintext under the key it names.
func (m *MAC) Valid(plaintext, signature string) bool {
	sig, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	kid, n := binary.Uvarint(sig)
	if n <= 0 || kid > math.MaxUint16 {
		return false
	}
	key, err := m.keys.Find(cryptobox.KID(kid))
	if err != nil {
		return false
	}
	secret, err := m.secret(context.Background(), m.keys.Version(), key)
	if err != nil {
		return false
	}
	return hmac.Equal(sig[n:], macSum(nil, secret, []byte(plaintext)))
}

// secret returns the derived secret for key. The cache is dropped whenever the
// keyring version differs from the one the cache was filled under. version must
// be read before key was fetched from the keyring.
func (m *MAC) secret(ctx context.Context, version uint64, key cryptobox.Key) ([]byte, error) {
	m.mu.RLock()
	if m.version == version {
		if secret, ok := m.secrets[key.Kid()]; ok {
			m.mu.RUnlock()
			return secret, nil
		}
	}
	m.mu.RUnlock()

	material, err := key.KeyMaterial(ctx)
	if err != nil {
		return nil, err
	}
	secret, err := hkdf.Key(sha256.New, material, nil, macInfo, macKeySize)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	if m.version != version || m.secrets == nil {
		m.version = version
		m.secrets = make(map[cryptobox.KID][]byte)
	}
	m.secrets[key.Kid()] = secret
	m.mu.Unlock()

	return secret, nil
}

func macSum(dst, secret, data []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write(data)
	return h.Sum(dst)
}
