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

//openssl genrsa -out key.pem 512

type rsaKey struct {
	privateKey  *rsa.PrivateKey
	cipherBlock cipher.Block
}

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

	cipherKey := block.Bytes[0:32]
	var cipherBlock cipher.Block

	//Create a new AES cipher using the key
	cipherBlock, err = aes.NewCipher(cipherKey)
	if err != nil {
		return nil, err
	}

	return &rsaKey{
		privateKey:  pkey,
		cipherBlock: cipherBlock,
	}, nil
}

func (p *rsaKey) Generate(message []byte) (string, error) {
	hash := hash(message)
	bytes, err := rsa.SignPKCS1v15(rand.Reader, p.privateKey, crypto.SHA1, hash)
	if err != nil {
		return "", err
	}
	signature := hex.EncodeToString(bytes)

	return signature, nil
}

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

func (p *rsaKey) DecryptBytes(v []byte) ([]byte, error) {
	s, err := decrypt(p.cipherBlock, string(v))
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

func (p *rsaKey) EncryptId(id int64) (string, error) {
	return encrypt(p.cipherBlock, []byte(fmt.Sprintf("%v", id)))
}

func (p *rsaKey) EncryptBytes(v []byte) ([]byte, error) {
	s, err := encrypt(p.cipherBlock, v)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

func encrypt(block cipher.Block, plainText []byte) (string, error) {

	//Make the cipher text a byte array of size BlockSize + the length of the message
	cipherText := make([]byte, aes.BlockSize+len(plainText))

	//iv is the ciphertext up to the blocksize (16)
	iv := cipherText[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	//Encrypt the data:
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], plainText)

	//Return string encoded in base64
	return base64.RawStdEncoding.EncodeToString(cipherText), nil
}

func decrypt(block cipher.Block, secure string) (string, error) {
	//Remove base64 encoding:
	cipherText, err := base64.RawStdEncoding.DecodeString(secure)

	//IF DecodeString failed, exit:
	if err != nil {
		return "", err
	}

	//IF the length of the cipherText is less than 16 Bytes:
	if len(cipherText) < aes.BlockSize {
		return "", errors.New("ciphertext block size is too short")
	}

	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	//Decrypt the message
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)

	return string(cipherText), nil
}
