package presign

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
)

type rsaKey struct {
	privateKey  *rsa.PrivateKey
	cipherBlock cipher.Block
}

// NewPrivateKey loads the legacy RSA private key from the PEM file at pemLocation.
func NewPrivateKey(pemLocation string) (PreSign, error) {
	var pkey *rsa.PrivateKey
	cert, err := os.ReadFile(pemLocation)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(cert)
	if block == nil {
		return nil, errors.New("decode certificate")
	}

	switch block.Type {
	case "PRIVATE KEY":
		var key any
		key, err = x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		pkey = key.(*rsa.PrivateKey)
	case "RSA PRIVATE KEY":
		pkey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("Unknown block type \"%s\"", block.Type)
	}

	if err != nil {
		return nil, err
	}

	if len(block.Bytes) < 32 {
		return nil, errors.New("private key too short")
	}

	cipherKey := block.Bytes[0:32]
	var cipherBlock cipher.Block

	cipherBlock, err = aes.NewCipher(cipherKey)
	if err != nil {
		return nil, err
	}

	return &rsaKey{
		privateKey:  pkey,
		cipherBlock: cipherBlock,
	}, nil
}

// Generate returns the hex-encoded RSA-PKCS1v15 signature of message.
func (p *rsaKey) Generate(message []byte) (string, error) {
	hash := hash(message)
	bytes, err := rsa.SignPKCS1v15(rand.Reader, p.privateKey, crypto.SHA1, hash)
	if err != nil {
		return "", err
	}
	signature := hex.EncodeToString(bytes)

	return signature, nil
}

// Valid reports whether signature matches plaintext.
func (p *rsaKey) Valid(plaintext, signature string) bool {
	sig, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	hashed := hash([]byte(plaintext))

	err = rsa.VerifyPKCS1v15(&p.privateKey.PublicKey, crypto.SHA1, hashed[:], sig)
	if err != nil {
		return false
	}

	return true
}

// DecryptId decrypts a token produced by EncryptId.
func (p *rsaKey) DecryptId(key string) (int64, error) {
	val, appErr := decrypt(p.cipherBlock, key)
	if appErr != nil {
		return 0, appErr
	}

	id, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}

	return int64(id), nil
}

// DecryptBytes decrypts text produced by EncryptBytes.
func (p *rsaKey) DecryptBytes(v []byte) ([]byte, error) {
	s, err := decrypt(p.cipherBlock, string(v))
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

// EncryptId encrypts id into a printable token.
func (p *rsaKey) EncryptId(id int64) (string, error) {
	return encrypt(p.cipherBlock, []byte(fmt.Sprintf("%v", id)))
}

// EncryptBytes encrypts v into printable text bytes.
func (p *rsaKey) EncryptBytes(v []byte) ([]byte, error) {
	s, err := encrypt(p.cipherBlock, v)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

func encrypt(block cipher.Block, plainText []byte) (string, error) {

	cipherText := make([]byte, aes.BlockSize+len(plainText))

	iv := cipherText[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], plainText)

	return base64.RawStdEncoding.EncodeToString(cipherText), nil
}

func decrypt(block cipher.Block, secure string) (string, error) {
	cipherText, err := base64.RawStdEncoding.DecodeString(secure)

	if err != nil {
		return "", err
	}

	if len(cipherText) < aes.BlockSize {
		return "", errors.New("ciphertext block size is too short")
	}

	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)

	return string(cipherText), nil
}
