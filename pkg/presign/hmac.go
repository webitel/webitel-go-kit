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
	"reflect"
	"sync"

	"github.com/webitel/crypto/cryptobox"
)

const (
	macInfo    = "webitel.presign.v1.hmac-sha256"
	macKeySize = sha256.Size
	macSigSize = binary.MaxVarintLen16 + sha256.Size
)

var ErrNoKeyring = errors.New("presign: keyring is required")

type MAC struct {
	keys  *cryptobox.Keyring
	cache sync.Map
}

type macSecret struct {
	src    cryptobox.Key
	secret []byte
}

var _ Signer = (*MAC)(nil)

func NewMAC() (Signer, error) {
	keys, err := cryptobox.DefaultKeys()
	if err != nil {
		return nil, err
	}
	return NewMACKeyring(keys)
}

func NewMACKeyring(keys *cryptobox.Keyring) (Signer, error) {
	if keys == nil {
		return nil, ErrNoKeyring
	}
	m := &MAC{keys: keys}
	if _, err := m.secret(context.Background(), keys.Primary()); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *MAC) Generate(data []byte) (string, error) {
	key := m.keys.Primary()
	secret, err := m.secret(context.Background(), key)
	if err != nil {
		return "", err
	}
	sig := make([]byte, 0, macSigSize)
	sig = binary.AppendUvarint(sig, uint64(key.Kid()))
	sig = macSum(sig, secret, data)
	return hex.EncodeToString(sig), nil
}

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
	secret, err := m.secret(context.Background(), key)
	if err != nil {
		return false
	}
	return hmac.Equal(sig[n:], macSum(nil, secret, []byte(plaintext)))
}

func (m *MAC) secret(ctx context.Context, key cryptobox.Key) ([]byte, error) {
	if v, ok := m.cache.Load(key.Kid()); ok {
		if cached := v.(macSecret); sameKey(cached.src, key) {
			return cached.secret, nil
		}
	}
	material, err := key.KeyMaterial(ctx)
	if err != nil {
		return nil, err
	}
	secret, err := hkdf.Key(sha256.New, material, nil, macInfo, macKeySize)
	if err != nil {
		return nil, err
	}
	m.cache.Store(key.Kid(), macSecret{src: key, secret: secret})
	return secret, nil
}

func sameKey(a, b cryptobox.Key) bool {
	return reflect.ValueOf(a).Comparable() && reflect.ValueOf(b).Comparable() && a == b
}

func macSum(dst, secret, data []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write(data)
	return h.Sum(dst)
}
