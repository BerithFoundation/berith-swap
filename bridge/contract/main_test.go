package contract

import (
	"berith-swap/bridge/config"
	"berith-swap/bridge/connection"
	"berith-swap/bridge/keypair"
	"berith-swap/bridge/transaction"
	"berith-swap/logger"
	"crypto/rand"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

var ks *keystore.KeyStore

const (
	testKeyDir  = "./testkey"
	testAccount = "a52438aefe8932786f260882a8867afa3b09165f"
	testPW      = "0000"
	configDir   = "../../run_test/"
	SenderIdx   = 0
	ReceiverIdx = 1
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func makeTestEthKeystore(dir string) *keystore.KeyStore {
	return keystore.NewKeyStore(dir, keystore.StandardScryptN, keystore.StandardScryptP)
}

func testGenerateKeypair(t *testing.T) *keypair.Keypair {
	var testBytes = make([]byte, 32)

	_, err := rand.Read(testBytes)
	require.NoError(t, err)

	ks := getTestKeyStore(t)
	acc, err := ks.NewAccount(testPW)
	require.NoError(t, err)
	require.NotEmpty(t, acc)

	kp, err := keypair.GenerateKeyPair(acc.Address.Hex(), testKeyDir, testPW)
	require.NoError(t, err)
	require.NotNil(t, kp)
	return kp
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

func removeTestKeyStore(t *testing.T) {
	if _, err := os.Stat(testKeyDir); os.IsExist(err) {
		err := os.RemoveAll(testKeyDir)
		require.NoError(t, err)
	}
}

func initTestconfig(t *testing.T) *config.Config {
	lines, err := config.ParsePasswordFile(configDir + "password")
	require.NoError(t, err)

	cfg := new(config.Config)
	err = config.LoadConfig(configDir+"config.json", cfg)
	require.NoError(t, err)

	for _, chain := range cfg.ChainConfig {
		switch chain.Idx {
		case SenderIdx:
			chain.Name = "berith-test"
			chain.Password = lines[SenderIdx]
		case ReceiverIdx:
			chain.Name = "klaytn-test"
			chain.Password = lines[ReceiverIdx]
		}
		chain.BlockConfirmations = "1"
	}
	cfg.KeystorePath = configDir + "keys"
	cfg.BlockStorePath = configDir + "blockstore"
	cfg.IsLoaded = false
	cfg.Verbosity = zerolog.Level(-1)

	return cfg
}

func newTestSwapContract(t *testing.T, kp *keypair.Keypair) *SwapContract {
	cfg := initTestconfig(t)
	chainCfg := cfg.ChainConfig[SenderIdx]

	logger := logger.NewLogger(cfg.Verbosity, chainCfg.Name)

	client, err := connection.NewEvmClient(kp, chainCfg.Endpoint, &logger)
	require.NoError(t, err)

	bridgeAddr := common.HexToAddress(chainCfg.SwapAddress)

	tran, err := InitializeTransactor(BerithGasPrice, transaction.NewTransaction, client)
	require.NoError(t, err)

	return NewSwapContract(client, bridgeAddr, tran, &logger)
}

func newTestERC20Contract(t *testing.T, gasPayLimit *big.Int) *ERC20Contract {
	cfg := initTestconfig(t)
	chainCfg := cfg.ChainConfig[ReceiverIdx]

	logger := logger.NewLogger(cfg.Verbosity, chainCfg.Name)

	kp, err := keypair.GenerateKeyPair(chainCfg.Owner, cfg.KeystorePath, chainCfg.Password)
	require.NoError(t, err)

	client, err := connection.NewEvmClient(kp, chainCfg.Endpoint, &logger)
	require.NoError(t, err)

	erc20Addr := common.HexToAddress(chainCfg.Erc20Address)

	tran, err := InitializeTransactor(gasPayLimit, transaction.NewTransaction, client)
	require.NoError(t, err)

	return NewERC20Contract(client, erc20Addr, tran, &logger)
}
