package counter

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"

	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"

	"github.com/okx/brczero/libs/tendermint/abci/example/code"
	"github.com/okx/brczero/libs/tendermint/abci/types"
)

type MockExTxInfo struct {
	Sender      string   `json:"sender"`
	SenderNonce uint64   `json:"sender_nonce"`
	GasPrice    *big.Int `json:"gas_price"`
	Nonce       uint64   `json:"nonce"`
}

type Application struct {
	types.BaseApplication

	hashCount int
	txCount   int
	serial    bool
}

func NewApplication(serial bool) *Application {
	return &Application{serial: serial}
}

func (app *Application) Info(req types.RequestInfo) types.ResponseInfo {
	return types.ResponseInfo{Data: fmt.Sprintf("{\"hashes\":%v,\"txs\":%v}", app.hashCount, app.txCount)}
}

func (app *Application) SetOption(req types.RequestSetOption) types.ResponseSetOption {
	key, value := req.Key, req.Value
	if key == "serial" && value == "on" {
		app.serial = true
	} else {
		/*
			TODO Panic and have the ABCI server pass an exception.
			The client can call SetOptionSync() and get an `error`.
			return types.ResponseSetOption{
				Error: fmt.Sprintf("Unknown key (%s) or value (%s)", key, value),
			}
		*/
		return types.ResponseSetOption{}
	}

	return types.ResponseSetOption{}
}

func (app *Application) DeliverTx(req types.RequestDeliverTx) types.ResponseDeliverTx {
	if app.serial {
		if len(req.Tx) > 8 {
			return types.ResponseDeliverTx{
				Code: code.CodeTypeEncodingError,
				Log:  fmt.Sprintf("Max tx size is 8 bytes, got %d", len(req.Tx))}
		}
		tx8 := make([]byte, 8)
		copy(tx8[len(tx8)-len(req.Tx):], req.Tx)
		txValue := binary.BigEndian.Uint64(tx8)
		if txValue != uint64(app.txCount) {
			return types.ResponseDeliverTx{
				Code: code.CodeTypeBadNonce,
				Log:  fmt.Sprintf("Invalid nonce. Expected %v, got %v", app.txCount, txValue)}
		}
	}
	app.txCount++
	return types.ResponseDeliverTx{Code: code.CodeTypeOK}
}

type tx struct {
	sdk.BaseTx
}

func (tx *tx) GetGasPrice() *big.Int {
	return big.NewInt(1)
}

func (app *Application) CheckTx(req types.RequestCheckTx) types.ResponseCheckTx {
	if app.serial {
		if len(req.Tx) > 8 {
			return types.ResponseCheckTx{
				Code: code.CodeTypeEncodingError,
				Log:  fmt.Sprintf("Max tx size is 8 bytes, got %d", len(req.Tx))}
		}
		tx8 := make([]byte, 8)
		copy(tx8[len(tx8)-len(req.Tx):], req.Tx)
		txValue := binary.BigEndian.Uint64(tx8)
		if txValue < uint64(app.txCount) {
			return types.ResponseCheckTx{
				Code: code.CodeTypeBadNonce,
				Log:  fmt.Sprintf("Invalid nonce. Expected >= %v, got %v", app.txCount, txValue)}
		}
	}
	data, _ := json.Marshal(&MockExTxInfo{Sender: fmt.Sprintf("%+x", req.Tx), GasPrice: big.NewInt(1)})
	return types.ResponseCheckTx{Tx: &tx{sdk.BaseTx{From: fmt.Sprintf("%+x", req.Tx)}}, Code: code.CodeTypeOK, Data: data}
}

func (app *Application) Commit(req types.RequestCommit) types.ResponseCommit {
	app.hashCount++
	if app.txCount == 0 {
		return types.ResponseCommit{}
	}
	hash := make([]byte, 8)
	binary.BigEndian.PutUint64(hash, uint64(app.txCount))
	return types.ResponseCommit{Data: hash}
}

func (app *Application) Query(reqQuery types.RequestQuery) types.ResponseQuery {
	switch reqQuery.Path {
	case "hash":
		return types.ResponseQuery{Value: []byte(fmt.Sprintf("%v", app.hashCount))}
	case "tx":
		return types.ResponseQuery{Value: []byte(fmt.Sprintf("%v", app.txCount))}
	default:
		return types.ResponseQuery{Log: fmt.Sprintf("Invalid query path. Expected hash or tx, got %v", reqQuery.Path)}
	}
}
