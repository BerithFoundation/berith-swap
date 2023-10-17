package connection

import (
	"berith-swap/bridge/contract/consts"
	"berith-swap/bridge/message"
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestBuildQuery(t *testing.T) {
	contractAddr := common.HexToAddress("0x770369CD955462d2da22fa674c8da8f8B0ef4DB9")
	block := big.NewInt(2919328)
	query := buildQuery(contractAddr, message.Deposit, block, block)
	arg, err := toFilterArg(query)
	require.NoError(t, err)

	rpcClient, err := rpc.DialContext(context.Background(), "http://testnet.berith.co:8545")
	require.NoError(t, err)

	var result []types.Log
	err = rpcClient.CallContext(context.Background(), &result, "eth_getLogs", arg)
	require.NoError(t, err)

	abiParser, err := abi.JSON(strings.NewReader(consts.BerithSwapABI))
	require.NoError(t, err)

	type depositEvent struct {
		address common.Address
		amount  *big.Int
	}
	event, err := abiParser.Unpack("Deposit", result[0].Data)
	require.NoError(t, err)

	t.Log(event)
}

func toFilterArg(q ethereum.FilterQuery) (interface{}, error) {
	arg := map[string]interface{}{
		"address": q.Addresses,
		"topics":  q.Topics,
	}
	if q.BlockHash != nil {
		arg["blockHash"] = *q.BlockHash
		if q.FromBlock != nil || q.ToBlock != nil {
			return nil, errors.New("cannot specify both BlockHash and FromBlock/ToBlock")
		}
	} else {
		if q.FromBlock == nil {
			arg["fromBlock"] = "0x0"
		} else {
			arg["fromBlock"] = toBlockNumArg(q.FromBlock)
		}
		arg["toBlock"] = toBlockNumArg(q.ToBlock)
	}
	return arg, nil
}
