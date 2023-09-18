package chain

import (
	"berith-swap/bridge/config"
	"berith-swap/bridge/connection"
	"berith-swap/bridge/keypair"
	"berith-swap/bridge/transaction"
	"berith-swap/bridge/util"
	"fmt"
	"math/big"

	"github.com/rs/zerolog"
)

// Chain은 블록체인에 대한 정보를 담고 있습니다.
type Chain struct {
	Name         string
	Endpoint     string
	TransactOpts *transaction.TransactOptions
	GasLimit     *big.Int
	GasPrice     *big.Int
	EvmClient    *connection.EvmClient
	Logger       zerolog.Logger
}

// NewChain는 config를 통해 Chain을 생성합니다.
func NewChain(cfg *config.Config, idx int) (*Chain, error) {
	chainCfg := cfg.ChainConfig[idx]

	logger := NewLogger(cfg.Verbosity, chainCfg.Name)

	kp, err := keypair.GenerateKeyPair(chainCfg.Owner, cfg.KeystorePath, chainCfg.Password)
	if err != nil {
		return nil, fmt.Errorf("cannot generate keypair err:%w", err)
	}

	client, err := connection.NewEvmClient(kp, chainCfg.Endpoint, &logger)
	if err != nil {
		return nil, err
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
		Name:      chainCfg.Name,
		Endpoint:  chainCfg.Endpoint,
		EvmClient: client,
		GasLimit:  gl,
		GasPrice:  gp,
		Logger:    logger,
	}, nil
}
