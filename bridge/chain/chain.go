package chain

import (
	"berith-swap/bridge/config"
	"berith-swap/bridge/connection"
	"berith-swap/bridge/keypair"
	"berith-swap/bridge/transaction"
	"berith-swap/bridge/util"
	"fmt"
	"math/big"
	"os"

	"github.com/rs/zerolog"
)

type Chain struct {
	Name         string
	Endpoint     string
	TransactOpts *transaction.TransactOptions
	StartBlock   *big.Int
	GasLimit     *big.Int
	GasPrice     *big.Int
	EvmClient    *connection.EvmClient
	Logger       zerolog.Logger
	stop         chan struct{}
}

func NewChain(cfg *config.Config, idx int) (*Chain, error) {
	logger := zerolog.New(os.Stdout).Level(cfg.Verbosity)

	chainCfg := cfg.ChainConfig[idx]

	kp, err := keypair.GenerateKeyPair(chainCfg.Exchanger, cfg.KeystorePath, chainCfg.Password)
	if err != nil {
		return nil, fmt.Errorf("cannot generate keypair err:%w", err)
	}

	client, err := connection.NewEvmClient(kp, chainCfg.Endpoint, &logger)
	if err != nil {
		return nil, err
	}

	stop := make(chan struct{})

	var startBlock *big.Int
	if cfg.LatestBlock {
		curr, err := client.LatestBlock()
		if err != nil {
			return nil, err
		}
		startBlock = curr
	}

	gl, err := util.StringToBig(chainCfg.GasLimit, 10)
	if err != nil {
		return nil, fmt.Errorf("cannot convert gas-limit string to big int %w", err)
	}

	gp, err := util.StringToBig(chainCfg.MaxGasPrice, 10)
	if err != nil {
		return nil, fmt.Errorf("cannot convert gas-price string to big int %w", err)
	}

	return &Chain{
		Name:               chainCfg.Name,
		Endpoint:           chainCfg.Endpoint,
		EvmClient:          client,
		StartBlock:         startBlock,
		BlockConfirmations: blockConfirmations,
		GasLimit:           gl,
		GasPrice:           gp,
		stop:               stop,
		Logger:             logger,
	}, nil
}

func (c *Chain) Stop() {
	close(c.stop)
	if c.EvmClient != nil {
		c.EvmClient.Close()
	}
}
