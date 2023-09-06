package keypair

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func ReadFromFileAndDecrypt(filename string, password []byte) (*Keypair, error) {
	fp, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Clean(fp))
	if err != nil {
		return nil, err
	}

	keydata := new(EncryptedKeystore)
	err = json.Unmarshal(data, keydata)
	if err != nil {
		return nil, err
	}

	return DecryptKeypair(keydata.PublicKey, keydata.Ciphertext, password)
}

func DecryptKeypair(expectedPubK string, data, password []byte) (*Keypair, error) {
	pk, err := Decrypt(data, password)
	if err != nil {
		return nil, err
	}
	kp, err := DecodeKeypair(pk)
	if err != nil {
		return nil, err
	}

	// Check that the decoding matches what was expected
	if kp.PublicKey() != expectedPubK {
		return nil, fmt.Errorf("unexpected key file data, file may be corrupt or have been tampered with")
	}
	return kp, nil
}

// Decrypt uses AES to decrypt ciphertext with the symmetric key deterministically created from `password`
func Decrypt(data, password []byte) ([]byte, error) {
	gcm, err := gcmFromPassphrase(password)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		if err.Error() == "cipher: message authentication failed" {
			err = errors.New(err.Error() + ". Incorrect Password.")
		}
		return nil, err
	}

	return plaintext, nil
}

// DecodeKeypair turns input bytes into a keypair based on the specified key type
func DecodeKeypair(in []byte) (kp *Keypair, err error) {
	kp = &Keypair{}
	err = kp.Decode(in)

	return kp, err
}
