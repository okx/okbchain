package types

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	watcher "github.com/okx/brczero/x/evm/watcher"
)

// Copied the Account and StorageResult types since they are registered under an
// internal pkg on geth.

// AccountResult struct for account proof
type AccountResult struct {
	Address      common.Address  `json:"address"`
	AccountProof []string        `json:"accountProof"`
	Balance      *hexutil.Big    `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	StorageProof []StorageResult `json:"storageProof"`
}

// StorageResult defines the format for storage proof return
type StorageResult struct {
	Key   string       `json:"key"`
	Value *hexutil.Big `json:"value"`
	Proof []string     `json:"proof"`
}

// SendTxArgs represents the arguments to submit a new transaction into the transaction pool.
// Duplicate struct definition since geth struct is in internal package
// Ref: https://github.com/ethereum/go-ethereum/blob/release/1.9/internal/ethapi/api.go#L1346
type SendTxArgs struct {
	From     *common.Address `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
	// We accept "data" and "input" for backwards-compatibility reasons. "input" is the
	// newer name and should be preferred by clients.
	Data  *hexutil.Bytes `json:"data"`
	Input *hexutil.Bytes `json:"input"`
}

func (ca SendTxArgs) String() string {
	var arg string
	if ca.From != nil {
		arg += fmt.Sprintf("From: %s, ", ca.From.String())
	}
	if ca.To != nil {
		arg += fmt.Sprintf("To: %s, ", ca.To.String())
	}
	if ca.Gas != nil {
		arg += fmt.Sprintf("Gas: %s, ", ca.Gas.String())
	}
	if ca.GasPrice != nil {
		arg += fmt.Sprintf("GasPrice: %s, ", ca.GasPrice.String())
	}
	if ca.Value != nil {
		arg += fmt.Sprintf("Value: %s, ", ca.Value.String())
	}
	if ca.Nonce != nil {
		arg += fmt.Sprintf("Nonce: %s, ", ca.Nonce.String())
	}
	if ca.Data != nil {
		arg += fmt.Sprintf("Data: %s, ", ca.Data.String())
	}
	if ca.Input != nil {
		arg += fmt.Sprintf("Input: %s, ", ca.Input.String())
	}
	return strings.TrimRight(arg, ", ")
}

// CallArgs represents the arguments for a call.
type CallArgs struct {
	From     *common.Address `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Data     *hexutil.Bytes  `json:"data"`
}

func (ca CallArgs) String() string {
	var arg string
	if ca.From != nil {
		arg += fmt.Sprintf("From: %s, ", ca.From.String())
	}
	if ca.To != nil {
		arg += fmt.Sprintf("To: %s, ", ca.To.String())
	}
	if ca.Gas != nil {
		arg += fmt.Sprintf("Gas: %s, ", ca.Gas.String())
	}
	if ca.GasPrice != nil {
		arg += fmt.Sprintf("GasPrice: %s, ", ca.GasPrice.String())
	}
	if ca.Value != nil {
		arg += fmt.Sprintf("Value: %s, ", ca.Value.String())
	}
	if ca.Data != nil {
		arg += fmt.Sprintf("Data: %s, ", ca.Data.String())
	}
	return strings.TrimRight(arg, ", ")
}

// EthHeaderWithBlockHash represents a block header in the Ethereum blockchain with block hash generated from Tendermint Block
type EthHeaderWithBlockHash struct {
	ParentHash  common.Hash         `json:"parentHash"`
	UncleHash   common.Hash         `json:"sha3Uncles"`
	Coinbase    common.Address      `json:"miner"`
	Root        common.Hash         `json:"stateRoot"`
	TxHash      common.Hash         `json:"transactionsRoot"`
	ReceiptHash common.Hash         `json:"receiptsRoot"`
	Bloom       ethtypes.Bloom      `json:"logsBloom"`
	Difficulty  *hexutil.Big        `json:"difficulty"`
	Number      *hexutil.Big        `json:"number"`
	GasLimit    hexutil.Uint64      `json:"gasLimit"`
	GasUsed     hexutil.Uint64      `json:"gasUsed"`
	Time        hexutil.Uint64      `json:"timestamp"`
	Extra       hexutil.Bytes       `json:"extraData"`
	MixDigest   common.Hash         `json:"mixHash"`
	Nonce       ethtypes.BlockNonce `json:"nonce"`
	Hash        common.Hash         `json:"hash"`
}
type FeeHistoryResult struct {
	OldestBlock  *hexutil.Big     `json:"oldestBlock"`
	Reward       [][]*hexutil.Big `json:"reward,omitempty"`
	BaseFee      []*hexutil.Big   `json:"baseFeePerGas,omitempty"`
	GasUsedRatio []float64        `json:"gasUsedRatio"`
}

// SignTransactionResult represents a RLP encoded signed transaction.
type SignTransactionResult struct {
	Raw hexutil.Bytes        `json:"raw"`
	Tx  *watcher.Transaction `json:"tx"`
}

type GPIn3Gears struct {
	SafeLow *hexutil.Big `json:"safe_low"`
	Average *hexutil.Big `json:"average"`
	Fastest *hexutil.Big `json:"fastest"`
}

func NewGPIn3Gears(safelow, avg, fastest *big.Int) GPIn3Gears {
	return GPIn3Gears{
		SafeLow: (*hexutil.Big)(safelow),
		Average: (*hexutil.Big)(avg),
		Fastest: (*hexutil.Big)(fastest),
	}
}
