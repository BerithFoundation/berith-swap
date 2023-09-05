package chain

import (
	"berith-swap/blockstore"
	"berith-swap/config"
	"berith-swap/connection"
	"berith-swap/contract"
	"berith-swap/util"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

	eth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	BlockRetryLimit           = 5
	BlockRetryInterval        = time.Second * 5
	DefaultBlockConfirmations = big.NewInt(10)
)

type Chain struct {
	Name               string                  `json:"name"`
	Endpoint           string                  `json:"endpoint"`
	TokenContract      *contract.ERC20Contract `json:"tokenContract"`
	BlockStore         *blockstore.Blockstore  `json:"blockStore"`
	StartBlock         *big.Int                `json:"startBlock"`
	BlockConfirmations *big.Int                `json:"blockConfirmations"`
	Connection         connection.Connection   `json:"connection"`
	Logger             zerolog.Logger
	stop               chan struct{}
}

func NewChain(cfg *config.Config, idx int) (*Chain, error) {
	logger := zerolog.New(os.Stdout).Level(cfg.Verbosity)

	chainCfg := cfg.Chains[idx]

	blockStore, err := setupBlockstore(cfg.BlockStorePath, chainCfg.Name)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize blockstore err:%w", err)
	}

	kp, err := GenerateKeyPair(chainCfg.Exchanger, cfg.KeystorePath, chainCfg.Password)
	if err != nil {
		return nil, fmt.Errorf("cannot generate keypair err:%w", err)
	}

	var newErc20 *contract.ERC20Contract
	if chainCfg.Erc20Address != "" {
		newErc20, err = contract.InitErc20Contract(hex.EncodeToString(crypto.FromECDSA(kp.PrivateKey())), chainCfg.Erc20Address)
		if err != nil {
			return nil, fmt.Errorf("cannot init erc20 contract err:%w", err)
		}
	}

	stop := make(chan struct{})
	conn, err := connection.NewRpcConnection(&chainCfg, kp)
	if err != nil {
		return nil, err
	}
	err = conn.Connect()
	if err != nil {
		return nil, err
	}

	if newErc20 != nil {
		err = conn.EnsureHasBytecode(common.HexToAddress(chainCfg.Erc20Address))
		if err != nil {
			return nil, err
		}
	}

	var startBlock *big.Int
	if cfg.LatestBlock {
		curr, err := conn.LatestBlock()
		if err != nil {
			return nil, err
		}
		startBlock = curr
	}

	blockConfirmations, err := util.StringToBig(chainCfg.BlockConfirmations, 10)
	if err != nil {
		logger.Error().Msgf("cannot get block confirmations from config. set default:%d", DefaultBlockConfirmations.Int64())
		blockConfirmations = DefaultBlockConfirmations
	}

	return &Chain{
		Name:               chainCfg.Name,
		Endpoint:           chainCfg.Endpoint,
		BlockStore:         blockStore,
		Connection:         conn,
		StartBlock:         startBlock,
		BlockConfirmations: blockConfirmations,
		stop:               stop,
		Logger:             logger,
	}, nil
}

func setupBlockstore(bsPath, ChainName string) (*blockstore.Blockstore, error) {
	bs, err := blockstore.NewBlockstore(bsPath, ChainName)
	if err != nil {
		return nil, err
	}

	return bs, nil
}

func (c *Chain) Stop() {
	close(c.stop)
	if c.Connection != nil {
		c.Connection.Close()
	}
}

func (c *Chain) pollBlocks() error {
	var currentBlock = c.StartBlock
	log.Info().Msgf("Polling Blocks...", "block", currentBlock)

	var retry = BlockRetryLimit
	for {
		select {
		case <-c.stop:
			return errors.New("polling terminated")
		default:
			// No more retries, goto next block
			if retry == 0 {
				log.Error().Msg("Polling failed, retries exceeded")
				c.Connection.Close()
				return nil
			}

			latestBlock, err := c.Connection.LatestBlock()
			if err != nil {
				log.Error().Any("block", currentBlock).Err(err).Msg("Unable to get latest block")
				retry--
				time.Sleep(BlockRetryInterval)
				continue
			}

			// Sleep if the difference is less than BlockDelay; (latest - current) < BlockDelay
			if big.NewInt(0).Sub(latestBlock, currentBlock).Cmp(c.BlockConfirmations) == -1 {
				log.Debug().Any("target", currentBlock).Any("latest", latestBlock).Msg("Block not ready, will retry")
				time.Sleep(BlockRetryInterval)
				continue
			}

			// Coin 수신 체인 전용
			err = c.getDepositEventsForBlock(currentBlock)
			if err != nil {
				c.Logger.Error().Err(err).Any("block", currentBlock).Msg("Failed to get events for block")
				retry--
				continue
			}

			// Write to block store. Not a critical operation, no need to retry
			err = c.BlockStore.StoreBlock(currentBlock)
			if err != nil {
				c.Logger.Error().Err(err).Any("block", currentBlock).Msg("Failed to write latest block to blockstore")
			}

			// Goto next block and reset retry counter
			currentBlock.Add(currentBlock, big.NewInt(1))
			retry = BlockRetryLimit
		}
	}
}

// getDepositEventsForBlock looks for the deposit event in the latest block
func (c *Chain) getDepositEventsForBlock(latestBlock *big.Int) error {
	c.Logger.Debug().Any("block", latestBlock).Msg("Querying block for deposit events")
	query := buildQuery(c.Connection.Keypair().CommonAddress(), latestBlock, latestBlock)

	// querying for logs
	logs, err := c.Connection.Client().FilterLogs(context.Background(), query)
	if err != nil {
		return fmt.Errorf("unable to Filter Logs: %w", err)
	}

	// read through the log events and handle their deposit event if handler is recognized
	for _, log := range logs {
		tx, pending, err := c.Connection.Client().TransactionByHash(context.Background(), log.TxHash)
		if err != nil {
			return fmt.Errorf("error cannot get transaction by hash. hash:%s, err:%w", log.TxHash, err)
		}
		if !pending {
			signer := types.LatestSignerForChainID(tx.ChainId())
			sender, err := types.Sender(signer, tx)
			if err != nil {
				return fmt.Errorf("error cannot get sender from transaction. err:%w", err)
			}
			// TODO: sender address를 어떻게 상대 체인 스트리머에게 전송할 수 있을까?
		}
	}

	return nil
}

// buildQuery constructs a query by receiver.
func buildQuery(receiver common.Address, startBlock *big.Int, endBlock *big.Int) eth.FilterQuery {
	query := eth.FilterQuery{
		FromBlock: startBlock,
		ToBlock:   endBlock,
		Addresses: []common.Address{receiver},
	}
	return query
}
