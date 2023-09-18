package keypair

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/stretchr/testify/require"
)

var ks *keystore.KeyStore

const (
	testKeyDir = "./testkey"
	testPW     = "0000"
)

func makeTestEthKeystore(dir string) *keystore.KeyStore {
	return keystore.NewKeyStore(dir, keystore.StandardScryptN, keystore.StandardScryptP)
}

func getTestKeyStore(t *testing.T) *keystore.KeyStore {
	if ks == nil {

		if _, err := os.Stat(testKeyDir); os.IsNotExist(err) {
			err := os.MkdirAll(testKeyDir, 0700)
			require.NoError(t, err)
		}

		ks = makeTestEthKeystore(testKeyDir)
	}
	return ks
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
