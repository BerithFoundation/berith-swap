package contract

import (
	"berith-swap/bridge/config"
	"berith-swap/bridge/connection"
	"berith-swap/bridge/keypair"
	"berith-swap/bridge/transaction"
	"berith-swap/logger"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

const (
	configDir   = "../../run_test/"
	SenderIdx   = 0
	ReceiverIdx = 1
)

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

func newTestBridgeContract(t *testing.T, limitGP *big.Int) *BridgeContract {
	cfg := initTestconfig(t)
	chainCfg := cfg.ChainConfig[SenderIdx]

	logger := logger.NewLogger(cfg.Verbosity, chainCfg.Name)

	kp, err := keypair.GenerateKeyPair(chainCfg.Owner, cfg.KeystorePath, chainCfg.Password)
	require.NoError(t, err)

	client, err := connection.NewEvmClient(kp, chainCfg.Endpoint, &logger)
	require.NoError(t, err)

	bridgeAddr := common.HexToAddress(chainCfg.BridgeAddress)

	tran, err := InitializeTransactor(limitGP, transaction.NewTransaction, client)
	require.NoError(t, err)

	return NewBridgeContract(client, bridgeAddr, tran, &logger)
}

func newTestERC20Contract(t *testing.T, limitGP *big.Int) *ERC20Contract {
	cfg := initTestconfig(t)
	chainCfg := cfg.ChainConfig[ReceiverIdx]

	logger := logger.NewLogger(cfg.Verbosity, chainCfg.Name)

	kp, err := keypair.GenerateKeyPair(chainCfg.Owner, cfg.KeystorePath, chainCfg.Password)
	require.NoError(t, err)

	client, err := connection.NewEvmClient(kp, chainCfg.Endpoint, &logger)
	require.NoError(t, err)

	bridgeAddr := common.HexToAddress(chainCfg.BridgeAddress)

	tran, err := InitializeTransactor(limitGP, transaction.NewTransaction, client)
	require.NoError(t, err)

	return NewERC20Contract(client, bridgeAddr, tran, &logger)
}
