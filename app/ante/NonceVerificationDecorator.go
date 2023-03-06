package ante

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/okx/okbchain/libs/cosmos-sdk/baseapp"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/okbchain/libs/cosmos-sdk/types/errors"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth"
	evmtypes "github.com/okx/okbchain/x/evm/types"
)

// NonceVerificationDecorator checks that the account nonce from the transaction matches
// the sender account sequence.
type NonceVerificationDecorator struct {
	ak auth.AccountKeeper
}

// NewNonceVerificationDecorator creates a new NonceVerificationDecorator
func NewNonceVerificationDecorator(ak auth.AccountKeeper) NonceVerificationDecorator {
	return NonceVerificationDecorator{
		ak: ak,
	}
}

// AnteHandle validates that the transaction nonce is valid (equivalent to the sender account’s
// current nonce).
func (nvd NonceVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	pinAnte(ctx.AnteTracer(), "NonceVerificationDecorator")
	msgEthTx, ok := tx.(*evmtypes.MsgEthereumTx)
	if !ok {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
	}

	if ctx.From() != "" {
		msgEthTx.SetFrom(ctx.From())
	}
	// sender address should be in the tx cache from the previous AnteHandle call
	address := msgEthTx.AccountAddress()
	if address.Empty() {
		panic("sender address cannot be empty")
	}

	acc := nvd.ak.GetAccount(ctx, address)
	if acc == nil {
		return ctx, sdkerrors.Wrapf(
			sdkerrors.ErrUnknownAddress,
			"account %s (%s) is nil", common.BytesToAddress(address.Bytes()), address,
		)
	}

	seq := acc.GetSequence()
	// if multiple transactions are submitted in succession with increasing nonces,
	// all will be rejected except the first, since the first needs to be included in a block
	// before the sequence increments
	if ctx.IsCheckTx() {
		ctx.SetAccountNonce(seq)
		// will be checkTx and RecheckTx mode
		if ctx.IsReCheckTx() {
			// recheckTx mode

			// sequence must strictly increasing
			if msgEthTx.Data.AccountNonce != seq {
				return ctx, sdkerrors.Wrapf(
					sdkerrors.ErrInvalidSequence,
					"invalid nonce; got %d, expected %d", msgEthTx.Data.AccountNonce, seq,
				)
			}
		} else {
			if baseapp.IsMempoolEnablePendingPool() {
				if msgEthTx.Data.AccountNonce < seq {
					return ctx, sdkerrors.Wrapf(
						sdkerrors.ErrInvalidSequence,
						"invalid nonce; got %d, expected %d",
						msgEthTx.Data.AccountNonce, seq,
					)
				}
			} else {
				// checkTx mode
				checkTxModeNonce := seq

				if !baseapp.IsMempoolEnableRecheck() {
					// if is enable recheck, the sequence of checkState will increase after commit(), so we do not need
					// to add pending txs len in the mempool.
					// but, if disable recheck, we will not increase sequence of checkState (even in force recheck case, we
					// will also reset checkState), so we will need to add pending txs len to get the right nonce
					gPool := baseapp.GetGlobalMempool()
					if gPool != nil {
						addr := evmtypes.EthAddressStringer(common.BytesToAddress(msgEthTx.AccountAddress().Bytes())).String()
						if pendingNonce, ok := gPool.GetPendingNonce(addr); ok {
							checkTxModeNonce = pendingNonce + 1
						}
					}
				}

				if baseapp.IsMempoolEnableSort() {
					if msgEthTx.Data.AccountNonce < seq || msgEthTx.Data.AccountNonce > checkTxModeNonce {
						return ctx, sdkerrors.Wrapf(
							sdkerrors.ErrInvalidSequence,
							"invalid nonce; got %d, expected in the range of [%d, %d]",
							msgEthTx.Data.AccountNonce, seq, checkTxModeNonce,
						)
					}
				} else {
					if msgEthTx.Data.AccountNonce != checkTxModeNonce {
						return ctx, sdkerrors.Wrapf(
							sdkerrors.ErrInvalidSequence,
							"invalid nonce; got %d, expected %d",
							msgEthTx.Data.AccountNonce, checkTxModeNonce,
						)
					}
				}
			}
		}
	} else {
		// only deliverTx mode
		if msgEthTx.Data.AccountNonce != seq {
			return ctx, sdkerrors.Wrapf(
				sdkerrors.ErrInvalidSequence,
				"invalid nonce; got %d, expected %d", msgEthTx.Data.AccountNonce, seq,
			)
		}
	}

	return next(ctx, tx, simulate)
}
