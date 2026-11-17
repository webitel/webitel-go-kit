package presign

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/webitel/crypto/cryptobox"
)

var hashSum = hash

// Crypto implements [PreSign] on top of a cryptobox cipher.
type Crypto struct {
	box cryptobox.Cipher
}

var _ PreSign = (*Crypto)(nil)

// NewCryptoBox returns a [Crypto] backed by default box
func NewCryptoBox() (*Crypto, error) {
	box, err := cryptobox.Default()
	if err != nil {
		return nil, err
	}
	return NewCryptoBoxCipher(box)
}

// NewCryptoBoxCipher returns a [Crypto] backed by box
func NewCryptoBoxCipher(box cryptobox.Cipher) (*Crypto, error) {
	cbox := &Crypto{
		box: box,
	}
	return cbox, nil
}

// Generate returns the hex-encoded encrypted hash of data.
func (c *Crypto) Generate(data []byte) (string, error) {
	dataHash := hashSum(data)
	sign, err := c.box.Encrypt(
		context.Background(), dataHash,
	)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(sign), nil
}

// Valid reports whether signature matches plaintext.
func (c *Crypto) Valid(plaintext string, signature string) bool {
	sign, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	dataHash, err := c.box.Decrypt(
		context.Background(), sign,
	)
	if err != nil {
		return false
	}

	want := hashSum([]byte(plaintext))
	return bytes.Equal(want, dataHash)
}

// EncryptId encrypts id into a printable token.
func (c *Crypto) EncryptId(id int64) (string, error) {
	text, err := c.encryptText(
		binary.AppendVarint(nil, id),
	)
	if err != nil {
		return "", err
	}
	return string(text), nil
}

// DecryptId decrypts a token produced by EncryptId.
func (c *Crypto) DecryptId(key string) (int64, error) {
	data, err := c.decryptText([]byte(key))
	if err != nil {
		return 0, err
	}
	num, n := binary.Varint(data)
	if n <= 0 || n != len(data) {
		return 0, strconv.ErrSyntax
	}
	return num, nil
}

// EncryptBytes encrypts v into printable text bytes.
func (c *Crypto) EncryptBytes(v []byte) ([]byte, error) {
	return c.encryptText(v)
}

// DecryptBytes decrypts text produced by EncryptBytes.
func (c *Crypto) DecryptBytes(v []byte) ([]byte, error) {
	return c.decryptText(v)
}

const cipherTag = ".c1"

var cipherText = base64.RawURLEncoding

func (c *Crypto) encryptText(data []byte) (text []byte, err error) {
	blob, err := c.box.Encrypt(
		context.Background(), data,
	)
	if err != nil {
		return nil, err
	}

	return cipherText.AppendEncode([]byte(cipherTag), blob), nil
}

func (c *Crypto) decryptText(text []byte) (data []byte, err error) {
	text, v2 := bytes.CutPrefix(text, []byte(cipherTag))
	if !v2 {
		return nil, fmt.Errorf("presign: invalid syntax")
	}

	blob, err := cipherText.AppendDecode(nil, text)
	if err != nil {
		return nil, err
	}

	data, err = c.box.Decrypt(
		context.Background(), blob,
	)
	if err != nil {
		return nil, err
	}

	return data, nil
}
