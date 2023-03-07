package ante

import (
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/okbchain/libs/cosmos-sdk/types/errors"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth/keeper"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth/types"
	"github.com/okx/okbchain/libs/tendermint/crypto"
	"github.com/okx/okbchain/libs/tendermint/crypto/multisig"
)

var (
	_ TxWithMemo = (*types.StdTx)(nil) // assert StdTx implements TxWithMemo
)

// ValidateBasicDecorator will call tx.ValidateBasic and return any non-nil error.
// If ValidateBasic passes, decorator calls next AnteHandler in chain. Note,
// ValidateBasicDecorator decorator will not get executed on ReCheckTx since it
// is not dependent on application state.
type ValidateBasicDecorator struct{}

func NewValidateBasicDecorator() ValidateBasicDecorator {
	return ValidateBasicDecorator{}
}

func (vbd ValidateBasicDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// simulate means 'eth_call' or 'eth_estimateGas', when it means 'eth_estimateGas' we can not 'VerifySig'.so skip here
	if simulate {
		return next(ctx, tx, simulate)
	}

	trc := ctx.AnteTracer()
	if trc != nil {
		trc.RepeatingPin("ValidateBasicDecorator")
	}

	// no need to validate basic on recheck tx, call next antehandler
	if ctx.IsReCheckTx() {
		return next(ctx, tx, simulate)
	}
	if err := tx.ValidateBasic(); err != nil {
		return ctx, err
	}
	return next(ctx, tx, simulate)
}

// Tx must have GetMemo() method to use ValidateMemoDecorator
type TxWithMemo interface {
	sdk.Tx
	GetMemo() string
}

// ValidateMemoDecorator will validate memo given the parameters passed in
// If memo is too large decorator returns with error, otherwise call next AnteHandler
// CONTRACT: Tx must implement TxWithMemo interface
type ValidateMemoDecorator struct {
	ak keeper.AccountKeeper
}

func NewValidateMemoDecorator(ak keeper.AccountKeeper) ValidateMemoDecorator {
	return ValidateMemoDecorator{
		ak: ak,
	}
}

func (vmd ValidateMemoDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	memoTx, ok := tx.(TxWithMemo)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "invalid transaction type")
	}

	params := vmd.ak.GetParams(ctx)

	memoLength := len(memoTx.GetMemo())
	if uint64(memoLength) > params.MaxMemoCharacters {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrMemoTooLarge,
			"maximum number of characters is %d but received %d characters",
			params.MaxMemoCharacters, memoLength,
		)
	}

	return next(ctx, tx, simulate)
}

// ValidateMsgDecorator will validate msg with special requirement
//type ValidateMsgDecorator struct {
//	validateMsgHandler ValidateMsgHandler
//}
//
//func NewValidateMsgDecorator(validateMsgHandler ValidateMsgHandler) ValidateMsgDecorator {
//	return ValidateMsgDecorator{
//		validateMsgHandler: validateMsgHandler,
//	}
//}
//
//func (vmd ValidateMsgDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
//	// *ABORT* the tx in case of failing to validate it in checkTx mode
//	if ctx.IsCheckTx() && !simulate && vmd.validateMsgHandler != nil {
//		if err := vmd.validateMsgHandler(ctx, tx.GetMsgs()); err != nil {
//			return ctx, err
//		}
//	}
//	return next(ctx, tx, simulate)
//}

// ConsumeTxSizeGasDecorator will take in parameters and consume gas proportional
// to the size of tx before calling next AnteHandler. Note, the gas costs will be
// slightly over estimated due to the fact that any given signing account may need
// to be retrieved from state.
//
// CONTRACT: If simulate=true, then signatures must either be completely filled
// in or empty.
// CONTRACT: To use this decorator, signatures of transaction must be represented
// as types.StdSignature otherwise simulate mode will incorrectly estimate gas cost.
type ConsumeTxSizeGasDecorator struct {
	ak keeper.AccountKeeper
}

func NewConsumeGasForTxSizeDecorator(ak keeper.AccountKeeper) ConsumeTxSizeGasDecorator {
	return ConsumeTxSizeGasDecorator{
		ak: ak,
	}
}

func (cgts ConsumeTxSizeGasDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	sigTx, ok := tx.(SigVerifiableTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "invalid tx type")
	}
	params := cgts.ak.GetParams(ctx)
	ctx.GasMeter().ConsumeGas(params.TxSizeCostPerByte*sdk.Gas(len(ctx.TxBytes())), "txSize")

	// simulate gas cost for signatures in simulate mode
	if simulate {
		// in simulate mode, each element should be a nil signature
		sigs := sigTx.GetSignatures()
		for i, signer := range sigTx.GetSigners() {
			// if signature is already filled in, no need to simulate gas cost
			if sigs[i] != nil {
				continue
			}
			acc := cgts.ak.GetAccount(ctx, signer)

			var pubkey crypto.PubKey
			// use placeholder simSecp256k1Pubkey if sig is nil
			if acc == nil || acc.GetPubKey() == nil {
				pubkey = simSecp256k1Pubkey
			} else {
				pubkey = acc.GetPubKey()
			}
			// use stdsignature to mock the size of a full signature
			simSig := types.StdSignature{
				Signature: simSecp256k1Sig[:],
				PubKey:    pubkey,
			}
			sigBz := types.ModuleCdc.MustMarshalBinaryLengthPrefixed(simSig)
			cost := sdk.Gas(len(sigBz) + 6)

			// If the pubkey is a multi-signature pubkey, then we estimate for the maximum
			// number of signers.
			if _, ok := pubkey.(multisig.PubKeyMultisigThreshold); ok {
				cost *= params.TxSigLimit
			}

			ctx.GasMeter().ConsumeGas(params.TxSizeCostPerByte*cost, "txSize")
		}
	}

	return next(ctx, tx, simulate)
}
