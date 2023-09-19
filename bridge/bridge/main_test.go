package bridge

import (
	"berith-swap/bridge/config"
	"berith-swap/bridge/connection"
	"berith-swap/bridge/contract"
	"berith-swap/bridge/evmgaspricer"
	"berith-swap/bridge/keypair"
	"berith-swap/bridge/transaction"
	"berith-swap/logger"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

const (
	configDir = "../../run_test/"
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

func newTestBridge(t *testing.T, cfg *config.Config) *Bridge {
	br := NewBridge(cfg)
	require.NotNil(t, br)
	return br
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func testNewBridgeContract(t *testing.T, chainCfg *config.RawChainConfig) (*contract.BridgeContract, common.Address) {
	pk, err := crypto.HexToECDSA(testPrivKey)
	require.NoError(t, err)
	require.NotNil(t, pk)
	testKp := keypair.NewKeypairFromPrivateKey(pk)
	testLogger := logger.NewLogger(zerolog.Level(-1), "tester")

	client, err := connection.NewEvmClient(testKp, chainCfg.Endpoint, &testLogger)
	require.NoError(t, err)

	gasPricer := evmgaspricer.NewLondonGasPriceClient(
		client,
		&evmgaspricer.GasPricerOpts{UpperLimitFeePerGas: gasPrice},
	)

	trans := transaction.NewSignAndSendTransactor(transaction.NewTransaction, gasPricer, client)

	return contract.NewBridgeContract(client, common.HexToAddress(chainCfg.BridgeAddress), trans, &testLogger), client.From()
}
