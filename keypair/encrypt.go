package keypair

import (
	"crypto/aes"
	"crypto/cipher"

	"golang.org/x/crypto/blake2b"
)

type EncryptedKeystore struct {
	PublicKey  string `json:"publicKey"`
	Address    string `json:"address"`
	Ciphertext []byte `json:"ciphertext"`
}

func gcmFromPassphrase(password []byte) (cipher.AEAD, error) {
	hash := blake2b.Sum256(password)

	block, err := aes.NewCipher(hash[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm, nil
}
