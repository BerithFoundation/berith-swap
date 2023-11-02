package connection

import (
	"berith-swap/bridge/keypair"
	"berith-swap/bridge/message"
	"berith-swap/bridge/transaction"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type EvmClient struct {
	*ethclient.Client
	rpcClient *rpc.Client
	chainId   *big.Int
	signer    keypair.Signer
	nonce     *big.Int
	nonceLock sync.Mutex
	logger    *zerolog.Logger
}

func NewEvmClient(s *keypair.Keypair, endPoint string, logger *zerolog.Logger) (*EvmClient, error) {
	logger.Info().Msgf("Connecting to evm chain... url:%s", endPoint)

	ctx := context.Background()
	rpcClient, err := rpc.DialContext(ctx, endPoint)
	if err != nil {
		log.Err(err).Msg("cannot dial to rpc node")
		return nil, err
	}

	chainId := new(hexutil.Big)
	err = rpcClient.CallContext(ctx, chainId, "eth_chainId")
	if err != nil {
		log.Err(err).Msg("cannot get chain id")
		return nil, err
	}

	client := EvmClient{
		Client:    ethclient.NewClient(rpcClient),
		rpcClient: rpcClient,
		chainId:   (*big.Int)(chainId),
		signer:    s,
		logger:    logger,
	}

	return &client, nil
}

func (c *EvmClient) LockNonce() {
	c.nonceLock.Lock()
}

func (c *EvmClient) UnlockNonce() {
	c.nonceLock.Unlock()
}

func (c *EvmClient) UnsafeNonce() (*big.Int, error) {
	var err error
	for i := 0; i <= 10; i++ {
		if c.nonce == nil {
			nonce, err := c.PendingNonceAt(context.Background(), c.signer.CommonAddress())
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			c.nonce = big.NewInt(0).SetUint64(nonce)
			return c.nonce, nil
		} else if n, err := c.PendingNonceAt(context.Background(), c.signer.CommonAddress()); c.nonce.Uint64() < n { // bridge 실행 도중 다른 경로를 통해 컨트랙트를 직접 호출하여 nonce가 변경되었을 때
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			c.logger.Debug().Msgf("nonce too low. update from %d to %d", c.nonce.Uint64(), n)
			c.nonce = big.NewInt(0).SetUint64(n)
		}
		return c.nonce, nil
	}
	return nil, err
}

func (c *EvmClient) UnsafeIncreaseNonce() error {
	nonce, err := c.UnsafeNonce()
	if err != nil {
		return err
	}
	c.nonce = nonce.Add(nonce, big.NewInt(1))
	return nil
}

type headerNumber struct {
	Number *big.Int `json:"number"           gencodec:"required"`
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	return hexutil.EncodeBig(number)
}

func (h *headerNumber) UnmarshalJSON(input []byte) error {
	type headerNumber struct {
		Number *hexutil.Big `json:"number" gencodec:"required"`
	}
	var dec headerNumber
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Number == nil {
		return errors.New("missing required field 'number' for Header")
	}
	h.Number = (*big.Int)(dec.Number)
	return nil
}

func (c *EvmClient) LatestBlockNumber() (*big.Int, error) {
	var head *headerNumber
	err := c.rpcClient.CallContext(context.Background(), &head, "eth_getBlockByNumber", toBlockNumArg(nil), false)
	if err == nil && head == nil {
		err = ethereum.NotFound
	}
	if err != nil {
		return nil, err
	}
	return head.Number, nil
}

func (c *EvmClient) LatestBlock() (*types.Block, error) {
	var block *types.Block

	block, err := c.Client.BlockByNumber(context.Background(), nil)
	if err == nil && block == nil {
		err = ethereum.NotFound
	}
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (c *EvmClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return c.Client.SuggestGasPrice(ctx)
}

// PendingNonceAt returns the account nonce of the given account in the pending state.
// This is the nonce that should be used for the next transaction.
func (c *EvmClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	var result hexutil.Uint64
	err := c.rpcClient.CallContext(ctx, &result, "eth_getTransactionCount", account, "pending")
	return uint64(result), err
}

func (c *EvmClient) WaitAndReturnTxReceipt(h common.Hash) (*types.Receipt, error) {
	retry := 50
	for retry > 0 {
		receipt, err := c.Client.TransactionReceipt(context.Background(), h)
		if err != nil {
			retry--
			time.Sleep(5 * time.Second)
			continue
		}
		if receipt.Status != 1 {
			return receipt, fmt.Errorf("transaction failed on chain. Receipt status %v", receipt.Status)
		}
		return receipt, nil
	}
	return nil, errors.New("tx did not appear")
}

func (c *EvmClient) GetTransactionByHash(h common.Hash) (tx *types.Transaction, isPending bool, err error) {
	return c.Client.TransactionByHash(context.Background(), h)
}

func (c *EvmClient) FetchEventLogs(ctx context.Context, contractAddress common.Address, methodSig message.EventSig, startBlock *big.Int, endBlock *big.Int) ([]types.Log, error) {
	logs, err := c.FilterLogs(ctx, buildQuery(contractAddress, methodSig, startBlock, endBlock))
	if err != nil {
		return []types.Log{}, err
	}

	validLogs := make([]types.Log, 0)
	for _, log := range logs {
		if log.Removed {
			continue
		}

		validLogs = append(validLogs, log)
	}
	return validLogs, nil
}

// SendRawTransaction accepts rlp-encode of signed transaction and sends it via RPC call
func (c *EvmClient) SendRawTransaction(ctx context.Context, tx []byte) ([]byte, error) {
	var hex hexutil.Bytes
	err := c.rpcClient.CallContext(ctx, &hex, "eth_sendRawTransaction", hexutil.Encode(tx))
	return hex, err
}

func (c *EvmClient) CallContract(ctx context.Context, callArgs map[string]interface{}, blockNumber *big.Int) ([]byte, error) {
	var hex hexutil.Bytes
	err := c.rpcClient.CallContext(ctx, &hex, "eth_call", callArgs, toBlockNumArg(blockNumber))
	if err != nil {
		return nil, err
	}
	return hex, nil
}

func (c *EvmClient) CallContext(ctx context.Context, target interface{}, rpcMethod string, args ...interface{}) error {
	err := c.rpcClient.CallContext(ctx, target, rpcMethod, args...)
	if err != nil {
		return err
	}
	return nil
}

func (c *EvmClient) PendingCallContract(ctx context.Context, callArgs map[string]interface{}) ([]byte, error) {
	var hex hexutil.Bytes
	err := c.rpcClient.CallContext(ctx, &hex, "eth_call", callArgs, "pending")
	if err != nil {
		return nil, err
	}
	return hex, nil
}

func (c *EvmClient) From() common.Address {
	return c.signer.CommonAddress()
}

func (c *EvmClient) SignAndSendTransaction(ctx context.Context, tx transaction.CommonTransaction) (common.Hash, error) {
	id, err := c.ChainID(ctx)
	if err != nil {
		// panic(err)
		// Probably chain does not support chainID eg. CELO
		id = nil
	}
	rawTx, err := tx.RawWithSignature(c.signer, id)
	if err != nil {
		return common.Hash{}, err
	}
	hex, err := c.SendRawTransaction(ctx, rawTx)
	if err != nil {
		return common.Hash{}, err
	}
	if common.BytesToHash(hex) != tx.Hash() {
		return common.BytesToHash(hex), nil
	}
	return tx.Hash(), nil
}

func (c *EvmClient) BaseFee() (*big.Int, error) {
	head, err := c.HeaderByNumber(context.TODO(), nil)
	if err != nil {
		return nil, err
	}
	return head.BaseFee, nil
}

func (c *EvmClient) ChainId() *big.Int {
	return c.chainId
}

// EnsureHasBytecode asserts if contract code exists at the specified address
func (c *EvmClient) EnsureHasBytecode(addr common.Address) error {
	code, err := c.CodeAt(context.Background(), addr, nil)
	if err != nil {
		return err
	}

	if len(code) == 0 {
		return fmt.Errorf("no bytecode found at %s", addr.Hex())
	}
	return nil
}

func buildQuery(contract common.Address, sig message.EventSig, startBlock *big.Int, endBlock *big.Int) ethereum.FilterQuery {
	query := ethereum.FilterQuery{
		FromBlock: startBlock,
		ToBlock:   endBlock,
		Addresses: []common.Address{contract},
		Topics: [][]common.Hash{
			{sig.GetTopic()},
		},
	}
	return query
}
