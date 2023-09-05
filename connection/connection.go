package connection

import (
	"berith-swap/config"
	"berith-swap/keypair"
	"berith-swap/util"
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog/log"
)

var BlockRetryInterval = time.Second * 5

type Connection interface {
	Connect() error
	Keypair() *keypair.Keypair
	GetOpts() *bind.TransactOpts
	GetCallOpts() *bind.CallOpts
	LockAndUpdateOpts() error
	UnlockOpts()
	Client() *ethclient.Client
	EnsureHasBytecode(address common.Address) error
	LatestBlock() (*big.Int, error)
	WaitForBlock(block *big.Int, delay *big.Int) error
	Close()
}

type RpcConnection struct {
	endPoint    string
	keyPair     *keypair.Keypair
	gasLimit    *big.Int
	maxGasPrice *big.Int
	conn        *ethclient.Client
	nonce       uint64
	opts        *bind.TransactOpts
	optsLock    sync.Mutex
	callOpts    *bind.CallOpts
	stop        chan struct{}
}

// base: http
func NewRpcConnection(cfg *config.RawChainConfig, kp *keypair.Keypair) (*RpcConnection, error) {
	gl, err := util.StringToBig(cfg.GasLimit, 10)
	if err != nil {
		return nil, err
	}

	gp, err := util.StringToBig(cfg.MaxGasPrice, 10)
	if err != nil {
		return nil, err
	}
	return &RpcConnection{
		endPoint:    cfg.Endpoint,
		keyPair:     kp,
		gasLimit:    gl,
		maxGasPrice: gp,
		stop:        make(chan struct{}),
	}, nil
}

func (r *RpcConnection) Connect() error {
	log.Info().Msgf("Connecting to %s...", r.endPoint)

	rpcClient, err := rpc.DialHTTP(r.endPoint)
	if err != nil {
		return err
	}

	r.conn = ethclient.NewClient(rpcClient)

	opts, _, err := r.newTransactionOpts(big.NewInt(0), r.gasLimit, r.maxGasPrice)
	if err != nil {
		return err
	}
	r.opts = opts
	r.nonce = 0
	r.callOpts = &bind.CallOpts{From: r.keyPair.CommonAddress()}

	return nil
}

func (r *RpcConnection) newTransactionOpts(value, gasLimit, gasPrice *big.Int) (*bind.TransactOpts, uint64, error) {
	privateKey := r.keyPair.PrivateKey()
	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	ctx := context.Background()

	nonce, err := r.conn.PendingNonceAt(ctx, address)
	if err != nil {
		return nil, 0, err
	}

	id, err := r.conn.ChainID(ctx)
	if err != nil {
		return nil, 0, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, id)
	if err != nil {
		return nil, 0, err
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = value
	auth.GasLimit = gasLimit.Uint64()
	auth.GasPrice = gasPrice
	auth.Context = context.Background()

	return auth, nonce, nil
}

func (r *RpcConnection) Keypair() *keypair.Keypair {
	return r.keyPair
}

func (r *RpcConnection) Client() *ethclient.Client {
	return r.conn
}

func (r *RpcConnection) GetOpts() *bind.TransactOpts {
	return r.opts
}

func (r *RpcConnection) GetCallOpts() *bind.CallOpts {
	return r.callOpts
}

func (r *RpcConnection) SafeEstimateGas(ctx context.Context) (*big.Int, error) {

	var suggestedGasPrice *big.Int

	if suggestedGasPrice == nil {
		log.Debug().Msg("Fetching gasPrice from node")
		nodePriceEstimate, err := r.conn.SuggestGasPrice(context.TODO())
		if err != nil {
			return nil, err
		} else {
			suggestedGasPrice = nodePriceEstimate
		}
	}

	gasPrice := multiplyGasPrice(suggestedGasPrice, big.NewFloat(float64(1)))

	if gasPrice.Cmp(r.maxGasPrice) == 1 {
		return r.maxGasPrice, nil
	} else {
		return gasPrice, nil
	}
}

func (r *RpcConnection) EstimateGasLondon(ctx context.Context, baseFee *big.Int) (*big.Int, *big.Int, error) {
	var maxPriorityFeePerGas *big.Int
	var maxFeePerGas *big.Int

	if r.maxGasPrice.Cmp(baseFee) < 0 {
		maxPriorityFeePerGas = big.NewInt(1000000000)
		maxFeePerGas = new(big.Int).Add(r.maxGasPrice, maxPriorityFeePerGas)
		return maxPriorityFeePerGas, maxFeePerGas, nil
	}

	maxPriorityFeePerGas, err := r.conn.SuggestGasTipCap(context.TODO())
	if err != nil {
		return nil, nil, err
	}

	maxFeePerGas = new(big.Int).Add(
		maxPriorityFeePerGas,
		new(big.Int).Mul(baseFee, big.NewInt(2)),
	)

	if maxFeePerGas.Cmp(maxPriorityFeePerGas) < 0 {
		return nil, nil, fmt.Errorf("maxFeePerGas (%v) < maxPriorityFeePerGas (%v)", maxFeePerGas, maxPriorityFeePerGas)
	}

	if maxFeePerGas.Cmp(r.maxGasPrice) == 1 {
		maxPriorityFeePerGas.Sub(r.maxGasPrice, baseFee)
		maxFeePerGas = r.maxGasPrice
	}
	return maxPriorityFeePerGas, maxFeePerGas, nil
}

func multiplyGasPrice(gasEstimate *big.Int, gasMultiplier *big.Float) *big.Int {

	gasEstimateFloat := new(big.Float).SetInt(gasEstimate)

	result := gasEstimateFloat.Mul(gasEstimateFloat, gasMultiplier)

	gasPrice := new(big.Int)

	result.Int(gasPrice)

	return gasPrice
}

func (r *RpcConnection) LockAndUpdateOpts() error {
	r.optsLock.Lock()

	head, err := r.conn.HeaderByNumber(context.TODO(), nil)
	if err != nil {
		r.UnlockOpts()
		return err
	}

	if head.BaseFee != nil {
		r.opts.GasTipCap, r.opts.GasFeeCap, err = r.EstimateGasLondon(context.TODO(), head.BaseFee)
		if err != nil {
			r.UnlockOpts()
			return err
		}

		r.opts.GasPrice = nil
	} else {
		var gasPrice *big.Int
		gasPrice, err = r.SafeEstimateGas(context.TODO())
		if err != nil {
			r.UnlockOpts()
			return err
		}
		r.opts.GasPrice = gasPrice
	}

	nonce, err := r.conn.PendingNonceAt(context.Background(), r.opts.From)
	if err != nil {
		r.optsLock.Unlock()
		return err
	}
	r.opts.Nonce.SetUint64(nonce)
	return nil
}

func (r *RpcConnection) UnlockOpts() {
	r.optsLock.Unlock()
}

func (r *RpcConnection) LatestBlock() (*big.Int, error) {
	header, err := r.conn.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	return header.Number, nil
}

func (r *RpcConnection) EnsureHasBytecode(addr common.Address) error {
	code, err := r.conn.CodeAt(context.Background(), addr, nil)
	if err != nil {
		return err
	}

	if len(code) == 0 {
		return fmt.Errorf("no bytecode found at %s", addr.Hex())
	}
	return nil
}

func (r *RpcConnection) WaitForBlock(targetBlock *big.Int, delay *big.Int) error {
	for {
		select {
		case <-r.stop:
			return errors.New("connection terminated")
		default:
			currBlock, err := r.LatestBlock()
			if err != nil {
				return err
			}

			if delay != nil {
				currBlock.Sub(currBlock, delay)
			}

			// Equal or greater than target
			if currBlock.Cmp(targetBlock) >= 0 {
				return nil
			}
			log.Trace().Any("target", targetBlock).Any("current", currBlock).Any("delay", delay).Msg("Block not ready, waiting")
			time.Sleep(BlockRetryInterval)
			continue
		}
	}
}

func (r *RpcConnection) Close() {
	if r.conn != nil {
		r.conn.Close()
	}
	close(r.stop)
}
