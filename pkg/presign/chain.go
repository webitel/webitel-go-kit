package presign

import "cmp"

// PreSignChain tries implementations in order: the first one signs and
// encrypts, every one is tried when verifying and decrypting.
type PreSignChain []PreSign

var _ PreSign = (PreSignChain)(nil)

// NewPreSignChain returns a [PreSignChain] which implements [PreSign] and signs and encrypts with cryptobox.
// When pemLocation is not empty, the legacy RSA key also verifies and decrypts old data.
func NewPreSignChain(pemLocation string) (PreSignChain, error) {
	modern, err := NewCryptoBox()
	if err != nil {
		return nil, err
	}
	if pemLocation == "" {
		return PreSignChain{modern}, nil
	}
	legacy, err := NewPrivateKey(pemLocation)
	if err != nil {
		return nil, err
	}
	return PreSignChain{modern, legacy}, nil
}

// Generate signs data with the first implementation.
func (x PreSignChain) Generate(data []byte) (string, error) {
	return x.latest().Generate(data)
}

func (x PreSignChain) Valid(plaintext string, signature string) bool {
	for _, spec := range x {
		if spec.Valid(plaintext, signature) {
			return true // break
		}
	}
	return false
}

func (x PreSignChain) EncryptId(id int64) (string, error) {
	return x.latest().EncryptId(id)
}

func (x PreSignChain) DecryptId(key string) (int64, error) {
	var err error // remember: first (latest)
	for _, sup := range x {
		oid, res := sup.DecryptId(key)
		if res == nil {
			return oid, nil
		}
		err = cmp.Or(err, res) // first
	}
	return 0, err
}

func (x PreSignChain) EncryptBytes(data []byte) ([]byte, error) {
	return x.latest().EncryptBytes(data)
}

func (x PreSignChain) DecryptBytes(text []byte) ([]byte, error) {
	var err error // remember: first (latest)
	for _, sup := range x {
		data, res := sup.DecryptBytes(text)
		if res == nil {
			return data, nil
		}
		err = cmp.Or(err, res) // first
	}
	return nil, err
}

func (x PreSignChain) latest() PreSign {
	return x[0]
}
