package bridge

import (
	"berith-swap/bridge/config"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

const (
	configDir = "../../run_test/"
)

func initTestconfig(t *testing.T) *config.Config {
	lines, err := config.ParsePasswordFile(configDir + "password")
	require.NoError(t, err)

	return &config.Config{
		ChainConfig: []config.RawChainConfig{
			{
				Idx:                0,
				Name:               "berith-test",
				Endpoint:           "http://testnet.berith.co:8545",
				Owner:              "0x2345BF77D1De9eacf66FE81a09a86CfAb212a542",
				BridgeAddress:      "0x770369CD955462d2da22fa674c8da8f8B0ef4DB9",
				Erc20Address:       "",
				GasLimit:           "9000000",
				MaxGasPrice:        "30000000000",
				BlockConfirmations: "1",
				Password:           lines[0],
			},
			{
				Idx:                1,
				Name:               "klaytn-test",
				Endpoint:           "https://api.baobab.klaytn.net:8651",
				Owner:              "0x58b395A4bA195e03445a065cb6b4f195572DFBE7",
				BridgeAddress:      "",
				Erc20Address:       "0x1A89EC8D286c3060067D9D4341E9Ccf1a99653B6",
				GasLimit:           "9000000",
				MaxGasPrice:        "10000000000",
				BlockConfirmations: "1",
				Password:           lines[1],
			},
		},
		KeystorePath:   configDir + "keys",
		BlockStorePath: configDir + "blockstore",
		IsLoaded:       false,
		Verbosity:      zerolog.Level(-1),
	}
}

func newTestBridge(t *testing.T, cfg *config.Config) *Bridge {
	br := NewBridge(cfg)
	require.NotNil(t, br)
	return br
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
