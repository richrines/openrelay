package funds

import (
	"encoding/hex"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	orCommon "github.com/notegio/openrelay/common"
	"github.com/notegio/openrelay/exchangecontract"
	"github.com/notegio/openrelay/types"
	// "github.com/notegio/openrelay/channels"
	"github.com/notegio/openrelay/fillbloom"
	"log"
	"math/big"
)

type FilledLookup interface {
	GetAmountCancelled(order *types.Order) (*types.Uint256, error)
	GetAmountFilled(order *types.Order) (*types.Uint256, error)
}

type rpcFilledLookup struct {
	conn bind.ContractBackend
	fillBloom *fillbloom.FillBloom
}

func (filled *rpcFilledLookup) GetAmountCancelled(order *types.Order) (*types.Uint256, error) {
	cancelledAmount := &types.Uint256{}
	if filled.fillBloom != nil && filled.fillBloom.Initialized && !filled.fillBloom.Test(order.Hash()) {
		log.Printf("Bloom filter missing order: %#x", order.Hash())
		return cancelledAmount, nil
	}
	exchange, err := exchangecontract.NewExchange(orCommon.ToGethAddress(order.ExchangeAddress), filled.conn)
	if err != nil {
		log.Printf("Error intializing exchange contract '%v': '%v'", hex.EncodeToString(order.ExchangeAddress[:]), err.Error())
		return cancelledAmount, err
	}
	hash := [32]byte{}
	copy(hash[:], order.Hash())
	amount, err := exchange.Cancelled(nil, hash)
	if err != nil {
		orderBytes := order.Bytes()
		log.Printf("Error getting cancelled amount for order '%v': '%v'", hex.EncodeToString(orderBytes[:]), err.Error())
		return cancelledAmount, err
	}
	cancelledSlice := common.LeftPadBytes(amount.Bytes(), 32)
	copy(cancelledAmount[:], cancelledSlice)
	return cancelledAmount, nil
}

func (filled *rpcFilledLookup) GetAmountFilled(order *types.Order) (*types.Uint256, error) {
	filledAmount := &types.Uint256{}
	if filled.fillBloom != nil && filled.fillBloom.Initialized && !filled.fillBloom.Test(order.Hash()) {
		log.Printf("Bloom filter missing order: %#x", order.Hash())
		return filledAmount, nil
	}
	exchange, err := exchangecontract.NewExchange(orCommon.ToGethAddress(order.ExchangeAddress), filled.conn)
	if err != nil {
		log.Printf("Error intializing exchange contract '%v': '%v'", hex.EncodeToString(order.ExchangeAddress[:]), err.Error())
		return filledAmount, err
	}
	hash := [32]byte{}
	copy(hash[:], order.Hash())
	amount, err := exchange.Filled(nil, hash)
	if err != nil {
		orderBytes := order.Bytes()
		log.Printf("Error getting filled amount for order '%v': '%v'", hex.EncodeToString(orderBytes[:]), err.Error())
		return filledAmount, err
	}
	cancelledSlice := common.LeftPadBytes(amount.Bytes(), 32)
	copy(filledAmount[:], cancelledSlice)
	return filledAmount, nil
}


func NewRPCFilledLookup(rpcURL string, fillBloom *fillbloom.FillBloom) (FilledLookup, error) {
	conn, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}
	if fillBloom != nil {
		fillBloom.Initialize(conn, 0, []common.Address{})
	}
	return &rpcFilledLookup{conn, fillBloom}, nil
}

type MockFilledLookup struct {
	cancelled *big.Int
	filled    *big.Int
	err       error
}

func (filled *MockFilledLookup) GetAmountCancelled(order *types.Order) (*types.Uint256, error) {
	result := &types.Uint256{}
	if filled.err != nil {
		return result, filled.err
	}
	filledSlice := common.LeftPadBytes(filled.cancelled.Bytes(), 32)
	copy(result[:], filledSlice)
	return result, nil
}
func (filled *MockFilledLookup) GetAmountFilled(order *types.Order) (*types.Uint256, error) {
	result := &types.Uint256{}
	if filled.err != nil {
		return result, filled.err
	}
	filledSlice := common.LeftPadBytes(filled.filled.Bytes(), 32)
	copy(result[:], filledSlice)
	return result, nil
}

func NewMockFilledLookup(cancelled, filled string, err error) FilledLookup {
	cancelledInt := new(big.Int)
	cancelledInt.SetString(cancelled, 10)
	filledInt := new(big.Int)
	filledInt.SetString(filled, 10)
	return &MockFilledLookup{cancelledInt, filledInt, err}
}
