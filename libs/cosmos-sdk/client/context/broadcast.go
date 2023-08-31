package context

import (
	"fmt"
	"strings"

	"github.com/okx/brczero/libs/cosmos-sdk/client/flags"
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/brczero/libs/cosmos-sdk/types/errors"
	"github.com/okx/brczero/libs/tendermint/mempool"
	"github.com/okx/brczero/libs/tendermint/types"
)

// BroadcastTx broadcasts a transactions either synchronously or asynchronously
// based on the context parameters. The result of the broadcast is parsed into
// an intermediate structure which is logged if the context has a logger
// defined.
func (ctx CLIContext) BroadcastTx(txBytes []byte) (res sdk.TxResponse, err error) {
	switch ctx.BroadcastMode {
	case flags.BroadcastSync:
		res, err = ctx.BroadcastTxSync(txBytes)

	case flags.BroadcastAsync:
		res, err = ctx.BroadcastTxAsync(txBytes)

	case flags.BroadcastBlock:
		res, err = ctx.BroadcastTxCommit(txBytes)

	default:
		return sdk.TxResponse{}, fmt.Errorf("unsupported return type %s; supported types: sync, async, block", ctx.BroadcastMode)
	}

	return res, err
}

// CheckTendermintError checks if the error returned from BroadcastTx is a
// Tendermint error that is returned before the tx is submitted due to
// precondition checks that failed. If an Tendermint error is detected, this
// function returns the correct code back in TxResponse.
//
// TODO: Avoid brittle string matching in favor of error matching. This requires
// a change to Tendermint's RPCError type to allow retrieval or matching against
// a concrete error type.
func (ctx CLIContext) CheckTendermintError(err error, txBytes []byte) *sdk.TxResponse {
	if err == nil {
		return nil
	}

	txBytes = mempool.GetRealTxFromWrapCMTx(txBytes)
	errStr := strings.ToLower(err.Error())
	txHash := fmt.Sprintf("%X", types.Tx(txBytes).Hash())

	switch {
	case strings.Contains(errStr, strings.ToLower(mempool.ErrTxInCache.Error())):
		return &sdk.TxResponse{
			Code:   sdkerrors.ErrTxInMempoolCache.ABCICode(),
			RawLog: errStr,
			TxHash: txHash,
		}

	case strings.Contains(errStr, "mempool is full"):
		return &sdk.TxResponse{
			Code:   sdkerrors.ErrMempoolIsFull.ABCICode(),
			RawLog: errStr,
			TxHash: txHash,
		}

	case strings.Contains(errStr, "tx too large"):
		return &sdk.TxResponse{
			Code:   sdkerrors.ErrTxTooLarge.ABCICode(),
			RawLog: errStr,
			TxHash: txHash,
		}

	default:
		return nil
	}
}

// BroadcastTxCommit broadcasts transaction bytes to a Tendermint node and
// waits for a commit. An error is only returned if there is no RPC node
// connection or if broadcasting fails.
//
// NOTE: This should ideally not be used as the request may timeout but the tx
// may still be included in a block. Use BroadcastTxAsync or BroadcastTxSync
// instead.
func (ctx CLIContext) BroadcastTxCommit(txBytes []byte) (sdk.TxResponse, error) {
	node, err := ctx.GetNode()
	if err != nil {
		return sdk.TxResponse{}, err
	}

	res, err := node.BroadcastTxCommit(txBytes)
	if err != nil {
		if errRes := ctx.CheckTendermintError(err, txBytes); errRes != nil {
			return *errRes, nil
		}

		return sdk.NewResponseFormatBroadcastTxCommit(res), err
	}

	if !res.CheckTx.IsOK() {
		return sdk.NewResponseFormatBroadcastTxCommit(res), nil
	}

	if !res.DeliverTx.IsOK() {
		return sdk.NewResponseFormatBroadcastTxCommit(res), nil
	}

	return sdk.NewResponseFormatBroadcastTxCommit(res), nil
}

// BroadcastTxSync broadcasts transaction bytes to a Tendermint node
// synchronously (i.e. returns after CheckTx execution).
func (ctx CLIContext) BroadcastTxSync(txBytes []byte) (sdk.TxResponse, error) {
	node, err := ctx.GetNode()
	if err != nil {
		return sdk.TxResponse{}, err
	}

	res, err := node.BroadcastTxSync(txBytes)
	if errRes := ctx.CheckTendermintError(err, txBytes); errRes != nil {
		return *errRes, nil
	}

	return sdk.NewResponseFormatBroadcastTx(res), err
}

// BroadcastTxAsync broadcasts transaction bytes to a Tendermint node
// asynchronously (i.e. returns immediately).
func (ctx CLIContext) BroadcastTxAsync(txBytes []byte) (sdk.TxResponse, error) {
	node, err := ctx.GetNode()
	if err != nil {
		return sdk.TxResponse{}, err
	}

	res, err := node.BroadcastTxAsync(txBytes)
	if errRes := ctx.CheckTendermintError(err, txBytes); errRes != nil {
		return *errRes, nil
	}

	return sdk.NewResponseFormatBroadcastTx(res), err
}
