package presign

import "crypto"

// Signer signs data and verifies signatures.
type Signer interface {
	Generate(data []byte) (string, error)
	Valid(plaintext, signature string) bool
}

// Encryptor encrypts and decrypts identifiers and byte slices.
type Encryptor interface {
	DecryptId(key string) (int64, error)
	DecryptBytes(v []byte) ([]byte, error)
	EncryptId(id int64) (string, error)
	EncryptBytes(v []byte) ([]byte, error)
}

// PreSign combines [Signer] and [Encryptor].
type PreSign interface {
	Signer
	Encryptor
}

func hash(msg []byte) []byte {
	sh := crypto.SHA1.New()
	sh.Write(msg)
	hash := sh.Sum(nil)
	return hash
}
