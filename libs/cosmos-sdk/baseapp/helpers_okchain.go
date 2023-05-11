package baseapp

import (
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/okbchain/libs/cosmos-sdk/types/errors"
	abci "github.com/okx/okbchain/libs/tendermint/abci/types"
	tmtypes "github.com/okx/okbchain/libs/tendermint/types"
)

func (app *BaseApp) PushAnteHandler(ah sdk.AnteHandler) {
	app.anteHandler = ah
}

func (app *BaseApp) GetDeliverStateCtx() sdk.Context {
	return app.deliverState.ctx
}

// TraceTx returns the trace log for the target tx
// To trace the target tx, the context must be set to the specific block at first,
// and the predesessors in the same block must be run before tracing the tx.
// The runtx procedure for TraceTx is nearly same with that for DeliverTx,  but the
// state was saved in different Cache in app.
func (app *BaseApp) TraceTx(queryTraceTx sdk.QueryTraceTx, targetTx sdk.Tx, txIndex uint32, block *tmtypes.Block) (*sdk.Result, error) {

	//get first tx
	targetTxData := queryTraceTx.TxHash.Bytes()
	var initialTxBytes []byte
	predesessors := block.Txs[:txIndex]
	if len(predesessors) == 0 {
		initialTxBytes = targetTxData
	} else {
		initialTxBytes = predesessors[0]
	}

	//begin trace block to init traceState and traceBlockCache
	traceState, err := app.beginBlockForTracing(initialTxBytes, block)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to beginblock for tracing")
	}

	traceState.ctx.SetIsTraceTxLog(false)
	//pre deliver prodesessor tx to get the right state
	for _, predesessor := range block.Txs[:txIndex] {
		tx, err := app.txDecoder(predesessor, nil, block.Height)
		if err != nil {
			return nil, sdkerrors.Wrap(err, "invalid prodesessor")
		}
		app.tracetx(predesessor, tx, block.Height, traceState)
		//ignore the err when run prodesessor
	}

	//trace tx
	traceState.ctx.SetIsTraceTxLog(true)
	traceState.ctx.SetTraceTxLogConfig(queryTraceTx.ConfigBytes)
	info, err := app.tracetx(targetTxData, targetTx, block.Height, traceState)
	if info == nil {
		return nil, err
	}
	return info.result, err
}
func (app *BaseApp) tracetx(txBytes []byte, tx sdk.Tx, height int64, traceState *state) (info *runTxInfo, err error) {

	mode := runTxModeTrace
	//prepare runTxInfo to runtx
	info = &runTxInfo{}
	//init info.ctx
	info.ctx = traceState.ctx
	info.ctx.SetTxBytes(txBytes).
		SetVoteInfos(app.voteInfos).
		SetConsensusParams(app.consensusParams)

	err = app.runtxWithInfo(info, mode, txBytes, tx, height)
	return info, err
}
func (app *BaseApp) beginBlockForTracing(firstTx []byte, block *tmtypes.Block) (*state, error) {

	req := abci.RequestBeginBlock{
		Hash:   block.Hash(),
		Header: tmtypes.TM2PB.Header(&block.Header),
	}

	//set traceState instead of app.deliverState
	//need to reset to version = req.Header.Height-1
	traceState, err := app.newTraceState(req.Header, req.Header.Height-1)
	if err != nil {
		return nil, err
	}

	// use the same block gas meter with deliver mode
	var gasMeter sdk.GasMeter
	if maxGas := app.getMaximumBlockGas(); maxGas > 0 {
		gasMeter = sdk.NewGasMeter(maxGas)
	} else {
		gasMeter = sdk.NewInfiniteGasMeter()
	}

	traceState.ctx.SetBlockGasMeter(gasMeter)

	//set the trace mode to prevent the ante handler to check the nounce
	traceState.ctx.SetIsTraceTx(true)
	traceState.ctx.SetIsCheckTx(true)

	//app begin block
	if app.beginBlocker != nil {
		_ = app.beginBlocker(traceState.ctx, req)
	}

	// No need to set the signed validators for addition to context in deliverTx
	return traceState, nil
}
