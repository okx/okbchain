package proxy

import (
	abcicli "github.com/okx/brczero/libs/tendermint/abci/client"
	"github.com/okx/brczero/libs/tendermint/abci/types"
)

//----------------------------------------------------------------------------------------
// Enforce which abci msgs can be sent on a connection at the type level

type AppConnConsensus interface {
	SetResponseCallback(abcicli.Callback)
	Error() error

	InitChainSync(types.RequestInitChain) (*types.ResponseInitChain, error)

	BeginBlockSync(types.RequestBeginBlock) (*types.ResponseBeginBlock, error)
	DeliverTxAsync(types.RequestDeliverTx) *abcicli.ReqRes
	PreDeliverRealTxAsync(req []byte) types.TxEssentials
	DeliverRealTxAsync(essentials types.TxEssentials) *abcicli.ReqRes
	EndBlockSync(types.RequestEndBlock) (*types.ResponseEndBlock, error)
	CommitSync(types.RequestCommit) (*types.ResponseCommit, error)
	SetOptionAsync(req types.RequestSetOption) *abcicli.ReqRes
	ParallelTxs([][]byte, bool) []*types.ResponseDeliverTx
	SetOptionSync(req types.RequestSetOption) (*types.ResponseSetOption, error)
}

type AppConnMempool interface {
	SetResponseCallback(abcicli.Callback)
	Error() error

	CheckTxAsync(types.RequestCheckTx) *abcicli.ReqRes

	FlushAsync() *abcicli.ReqRes
	FlushSync() error

	SetOptionAsync(types.RequestSetOption) *abcicli.ReqRes

	QuerySync(req types.RequestQuery) (*types.ResponseQuery, error)
}

type AppConnQuery interface {
	Error() error

	EchoSync(string) (*types.ResponseEcho, error)
	InfoSync(types.RequestInfo) (*types.ResponseInfo, error)
	QuerySync(types.RequestQuery) (*types.ResponseQuery, error)

	//	SetOptionSync(key string, value string) (res types.Result)
}

//-----------------------------------------------------------------------------------------
// Implements AppConnConsensus (subset of abcicli.Client)

type appConnConsensus struct {
	appConn abcicli.Client
}

func NewAppConnConsensus(appConn abcicli.Client) AppConnConsensus {
	return &appConnConsensus{
		appConn: appConn,
	}
}

func (app *appConnConsensus) SetResponseCallback(cb abcicli.Callback) {
	app.appConn.SetResponseCallback(cb)
}

func (app *appConnConsensus) Error() error {
	return app.appConn.Error()
}

func (app *appConnConsensus) InitChainSync(req types.RequestInitChain) (*types.ResponseInitChain, error) {
	return app.appConn.InitChainSync(req)
}

func (app *appConnConsensus) BeginBlockSync(req types.RequestBeginBlock) (*types.ResponseBeginBlock, error) {
	return app.appConn.BeginBlockSync(req)
}

func (app *appConnConsensus) DeliverTxAsync(req types.RequestDeliverTx) *abcicli.ReqRes {
	return app.appConn.DeliverTxAsync(req)
}

func (app *appConnConsensus) PreDeliverRealTxAsync(req []byte) types.TxEssentials {
	return app.appConn.PreDeliverRealTxAsync(req)
}

func (app *appConnConsensus) DeliverRealTxAsync(req types.TxEssentials) *abcicli.ReqRes {
	return app.appConn.DeliverRealTxAsync(req)
}

func (app *appConnConsensus) EndBlockSync(req types.RequestEndBlock) (*types.ResponseEndBlock, error) {
	return app.appConn.EndBlockSync(req)
}

func (app *appConnConsensus) CommitSync(req types.RequestCommit) (*types.ResponseCommit, error) {
	return app.appConn.CommitSync(req)
}

func (app *appConnConsensus) SetOptionAsync(req types.RequestSetOption) *abcicli.ReqRes {
	return app.appConn.SetOptionAsync(req)
}

func (app *appConnConsensus) SetOptionSync(req types.RequestSetOption) (*types.ResponseSetOption, error) {
	return app.appConn.SetOptionSync(req)
}

//------------------------------------------------
// Implements AppConnMempool (subset of abcicli.Client)

type appConnMempool struct {
	appConn abcicli.Client
}

func NewAppConnMempool(appConn abcicli.Client) AppConnMempool {
	return &appConnMempool{
		appConn: appConn,
	}
}

func (app *appConnMempool) SetResponseCallback(cb abcicli.Callback) {
	app.appConn.SetResponseCallback(cb)
}

func (app *appConnMempool) Error() error {
	return app.appConn.Error()
}

func (app *appConnMempool) FlushAsync() *abcicli.ReqRes {
	return app.appConn.FlushAsync()
}

func (app *appConnMempool) FlushSync() error {
	return app.appConn.FlushSync()
}

func (app *appConnMempool) CheckTxAsync(req types.RequestCheckTx) *abcicli.ReqRes {
	return app.appConn.CheckTxAsync(req)
}

func (app *appConnMempool) SetOptionAsync(req types.RequestSetOption) *abcicli.ReqRes {
	return app.appConn.SetOptionAsync(req)
}

func (app *appConnMempool) QuerySync(req types.RequestQuery) (*types.ResponseQuery, error) {
	return app.appConn.QuerySync(req)
}

func (app *appConnConsensus) ParallelTxs(txs [][]byte, onlyCalSender bool) []*types.ResponseDeliverTx {
	return app.appConn.ParallelTxs(txs, onlyCalSender)
}

//------------------------------------------------
// Implements AppConnQuery (subset of abcicli.Client)

type appConnQuery struct {
	appConn abcicli.Client
}

func NewAppConnQuery(appConn abcicli.Client) AppConnQuery {
	return &appConnQuery{
		appConn: appConn,
	}
}

func (app *appConnQuery) Error() error {
	return app.appConn.Error()
}

func (app *appConnQuery) EchoSync(msg string) (*types.ResponseEcho, error) {
	return app.appConn.EchoSync(msg)
}

func (app *appConnQuery) InfoSync(req types.RequestInfo) (*types.ResponseInfo, error) {
	return app.appConn.InfoSync(req)
}

func (app *appConnQuery) QuerySync(reqQuery types.RequestQuery) (*types.ResponseQuery, error) {
	return app.appConn.QuerySync(reqQuery)
}
