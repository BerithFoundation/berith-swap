package keypair

import (
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/keystore"
)

func ReadFromFileAndDecrypt(filename string, password string) (*Keypair, error) {
	fp, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Clean(fp))
	if err != nil {
		return nil, err
	}

	key, err := keystore.DecryptKey(data, password)

	if err != nil {
		return nil, err
	}

	return &Keypair{
		public:  &key.PrivateKey.PublicKey,
		private: key.PrivateKey,
	}, nil
}
