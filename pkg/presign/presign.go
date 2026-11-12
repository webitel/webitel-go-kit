package presign

import "crypto"

type Signer interface {
	Generate(data []byte) (string, error)
	Valid(plaintext, signature string) bool
}

type Encryptor interface {
	DecryptId(key string) (int64, error)
	DecryptBytes(v []byte) ([]byte, error)
	EncryptId(id int64) (string, error)
	EncryptBytes(v []byte) ([]byte, error)
}

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
