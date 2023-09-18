package keypair

import (
	"crypto/rand"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/stretchr/testify/require"
)

var (
	testBytes = make([]byte, 32)
)

// testGenerateEthKey는 테스트를 위한 ethereum key를 생성합니다.
func testGenerateEthKey(t *testing.T) accounts.Account {
	_, err := rand.Read(testBytes)
	require.NoError(t, err)

	ks := getTestKeyStore(t)
	acc, err := ks.NewAccount(testPW)
	require.NoError(t, err)
	require.NotEmpty(t, acc)
	return acc
}

// testGenerateSignedHash는 테스트를 위한 ethereum key로 서명된 hash를 생성합니다.
func testGenerateSignedHash(t *testing.T, ks *keystore.KeyStore, acc accounts.Account) []byte {
	ks.Unlock(acc, testPW)
	signed, err := ks.SignHash(acc, testBytes)
	require.NoError(t, err)
	require.NotNil(t, signed)
	return signed
}

// TestEncriptKeyfile는 이더리움 형식으로 생성된 keyfile의 서명과 decoding된 keyfile의 서명이 같은지 테스트합니다.
func TestEncriptKeyfile(t *testing.T) {
	acc := testGenerateEthKey(t)
	signed := testGenerateSignedHash(t, getTestKeyStore(t), acc)
	kp, err := GenerateKeyPair(acc.Address.Hex(), testKeyDir, "0000")
	require.NoError(t, err)
	require.NotNil(t, kp)
	require.Equal(t, kp.Address(), acc.Address.Hex())
	require.NoError(t, os.RemoveAll(testKeyDir))

	signed2, err := kp.Sign(testBytes)
	require.NoError(t, err)
	require.Equal(t, signed, signed2)
}
