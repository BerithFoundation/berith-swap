package bridge

import (
	chain "berith-swap/bridge/chain"
	"berith-swap/bridge/config"
	"berith-swap/bridge/connection"
	"berith-swap/bridge/contract"
	"berith-swap/bridge/evmgaspricer"
	"berith-swap/bridge/keypair"
	"berith-swap/bridge/transaction"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

const (
	senderIdx   = 0
	receiverIdx = 1
	testPrivKey = "99a524626eedfb39288b4be1d5533882e3c36cff3a1b4de6f6ab9b45cf1d13b0"
)

var (
	wg            sync.WaitGroup
	gasPrice      = big.NewInt(25000000000)
	defaultTxOpts = transaction.TransactOptions{
		GasLimit: big.NewInt(9000000).Uint64(),
	}
)

func TestBridgeStartStop(t *testing.T) {
	cfg := initTestconfig(t)
	bridge := newTestBridge(t, cfg)

	errCh := make(chan error)
	go func(ch chan error) {
		err := bridge.Start()
		errCh <- err
	}(errCh)

	time.Sleep(5 * time.Second)

	bridge.Stop()

	t.Log(<-errCh)
}

// Receiver를 지정하지 않은 Deposit event를 Sender와 동일한 Receiver로 재 설정하여 처리하는가?
func TestInvalidReceiver(t *testing.T) {

	cfg := initTestconfig(t)
	senderCfg := cfg.ChainConfig[senderIdx]
	bridge := newTestBridge(t, cfg)

	bridgeCt, owner := newBridgeContract(t, senderCfg)

	var testCases = []struct {
		name    string
		address common.Address
	}{
		{
			name:    "pass",
			address: owner,
		},
		{
			name:    "pass zero address",
			address: common.Address{},
		},
	}

	before, err := bridge.rc.erc20Contract.GetBalance(owner)
	require.NoError(t, err)

	sendAmt := big.NewInt(0).Mul(big.NewInt(1), big.NewInt(1e18))

	defaultTxOpts.Value = sendAmt

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := bridgeCt.Deposit(tc.address, defaultTxOpts)
			require.NoError(t, err)

			go bridge.Start()

			checkBalance(t, bridge.rc.erc20Contract, before, sendAmt, owner)

			bridge.Stop()
		})
	}
}

func newBridgeContract(t *testing.T, chainCfg config.RawChainConfig) (*contract.BridgeContract, common.Address) {
	pk, err := crypto.HexToECDSA(testPrivKey)
	require.NoError(t, err)
	require.NotNil(t, pk)
	testKp := keypair.NewKeypairFromPrivateKey(pk)
	testLogger := chain.NewLogger(zerolog.Level(-1), "tester")

	client, err := connection.NewEvmClient(testKp, chainCfg.Endpoint, &testLogger)
	require.NoError(t, err)

	gasPricer := evmgaspricer.NewLondonGasPriceClient(
		client,
		&evmgaspricer.GasPricerOpts{UpperLimitFeePerGas: gasPrice},
	)

	trans := transaction.NewSignAndSendTransactor(transaction.NewTransaction, gasPricer, client)

	return contract.NewBridgeContract(client, common.HexToAddress(chainCfg.BridgeAddress), trans, &testLogger), client.From()
}

func checkBalance(t *testing.T, erc20 *contract.ERC20Contract, before, amt *big.Int, addr common.Address) bool {
	retry := 50
	for retry > 0 {
		after, err := erc20.GetBalance(addr)
		require.NoError(t, err)
		amtEther := new(big.Int).Div(amt, big.NewInt(1e18))

		check := new(big.Int).Add(before, amtEther).Cmp(after) == 0
		if check {
			erc20.Logger.Debug().Msg("balance checking passed")
			return true
		}
		time.Sleep(5 * time.Second)
		retry--
	}
	t.Fatal("balance checking failed")
	return false
}
