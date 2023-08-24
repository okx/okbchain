package ante

import (
	"bytes"
	"encoding/hex"

	ethcmn "github.com/ethereum/go-ethereum/common"

	"github.com/okx/okbchain/app/crypto/ethsecp256k1"
	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/okbchain/libs/cosmos-sdk/types/errors"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth/exported"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth/keeper"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth/types"
	"github.com/okx/okbchain/libs/tendermint/crypto"
	"github.com/okx/okbchain/libs/tendermint/crypto/ed25519"
	"github.com/okx/okbchain/libs/tendermint/crypto/etherhash"
	"github.com/okx/okbchain/libs/tendermint/crypto/multisig"
	"github.com/okx/okbchain/libs/tendermint/crypto/secp256k1"
	types2 "github.com/okx/okbchain/libs/tendermint/types"
)

var (
	// simulation signature values used to estimate gas consumption
	simSecp256k1Pubkey secp256k1.PubKeySecp256k1
	simSecp256k1Sig    [64]byte

	_ SigVerifiableTx = (*types.StdTx)(nil) // assert StdTx implements SigVerifiableTx
	_ SigVerifiableTx = (*types.IbcTx)(nil)
)

func init() {
	// This decodes a valid hex string into a sepc256k1Pubkey for use in transaction simulation
	bz, _ := hex.DecodeString("035AD6810A47F073553FF30D2FCC7E0D3B1C0B74B61A1AAA2582344037151E143A")
	copy(simSecp256k1Pubkey[:], bz)
}

// SignatureVerificationGasConsumer is the type of function that is used to both
// consume gas when verifying signatures and also to accept or reject different types of pubkeys
// This is where apps can define their own PubKey
type SignatureVerificationGasConsumer = func(meter sdk.GasMeter, sig []byte, pubkey crypto.PubKey, params types.Params) error

// SigVerifiableTx defines a Tx interface for all signature verification decorators
type SigVerifiableTx interface {
	sdk.Tx
	GetSignatures() [][]byte
	GetSigners() []sdk.AccAddress
	GetPubKeys() []crypto.PubKey // If signer already has pubkey in context, this list will have nil in its place
	GetSignBytes(ctx sdk.Context, index int, acc exported.Account) []byte
	//for ibc tx sign direct
	VerifySequence(index int, acc exported.Account) error
}

// SetPubKeyDecorator sets PubKeys in context for any signer which does not already have pubkey set
// PubKeys must be set in context for all signers before any other sigverify decorators run
// CONTRACT: Tx must implement SigVerifiableTx interface
type SetPubKeyDecorator struct {
	ak keeper.AccountKeeper
}

func NewSetPubKeyDecorator(ak keeper.AccountKeeper) SetPubKeyDecorator {
	return SetPubKeyDecorator{
		ak: ak,
	}
}

func (spkd SetPubKeyDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	sigTx, ok := tx.(SigVerifiableTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "invalid tx type")
	}

	pubkeys := sigTx.GetPubKeys()
	signers := sigTx.GetSigners()

	for i, pk := range pubkeys {
		// PublicKey was omitted from slice since it has already been set in context
		if pk == nil {
			if !simulate {
				continue
			}
			pk = simSecp256k1Pubkey
		}

		// Only make check if simulate=false
		var valid bool
		if !simulate {
			if pk, valid = checkSigner(pk, signers[i], ctx.BlockHeight()); !valid {
				return ctx, sdkerrors.Wrapf(sdkerrors.ErrInvalidPubKey,
					"pubKey does not match signer address %s with signer index: %d", signers[i], i)
			}
		}

		acc, err := GetSignerAcc(ctx, spkd.ak, signers[i])
		if err != nil {
			return ctx, err
		}
		// account already has pubkey set,no need to reset
		if !isPubKeyNeedChange(acc.GetPubKey(), pk, ctx.BlockHeight()) {
			continue
		}
		err = acc.SetPubKey(pk)
		if err != nil {
			return ctx, sdkerrors.Wrap(sdkerrors.ErrInvalidPubKey, err.Error())
		}
		spkd.ak.SetAccount(ctx, acc)
	}

	return next(ctx, tx, simulate)
}

func checkSigner(pk crypto.PubKey, signer sdk.AccAddress, height int64) (crypto.PubKey, bool) {
	if bytes.Equal(pk.Address(), signer) {
		return pk, true
	}
	// In case that tx is created by CosmWasmJS with pubKey type of `secp256k1`
	// 	and the signer address is derived by the pubKey of `ethsecp256k1` type.
	// Let it pass after Earth height.
	if types2.HigherThanEarth(height) {
		switch v := pk.(type) {
		case secp256k1.PubKeySecp256k1:
			ethPub := ethsecp256k1.PubKey(v[:])
			return ethPub, bytes.Equal(ethPub.Address(), signer)
		case *secp256k1.PubKeySecp256k1:
			ethPub := ethsecp256k1.PubKey(v[:])
			return ethPub, bytes.Equal(ethPub.Address(), signer)
		}
	}
	return pk, false
}

func isPubKeyNeedChange(pk1, pk2 crypto.PubKey, height int64) bool {
	if pk1 == nil {
		return true
	}
	if !types2.HigherThanEarth(height) {
		return false
	}

	// check if two public keys are equal
	return pk1.Equals(pk2)
}

// Consume parameter-defined amount of gas for each signature according to the passed-in SignatureVerificationGasConsumer function
// before calling the next AnteHandler
// CONTRACT: Pubkeys are set in context for all signers before this decorator runs
// CONTRACT: Tx must implement SigVerifiableTx interface
type SigGasConsumeDecorator struct {
	ak             keeper.AccountKeeper
	sigGasConsumer SignatureVerificationGasConsumer
}

func NewSigGasConsumeDecorator(ak keeper.AccountKeeper, sigGasConsumer SignatureVerificationGasConsumer) SigGasConsumeDecorator {
	return SigGasConsumeDecorator{
		ak:             ak,
		sigGasConsumer: sigGasConsumer,
	}
}

func (sgcd SigGasConsumeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	sigTx, ok := tx.(SigVerifiableTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "invalid transaction type")
	}

	params := sgcd.ak.GetParams(ctx)
	sigs := sigTx.GetSignatures()

	// stdSigs contains the sequence number, account number, and signatures.
	// When simulating, this would just be a 0-length slice.
	signerAddrs := sigTx.GetSigners()

	for i, sig := range sigs {
		signerAcc, err := GetSignerAcc(ctx, sgcd.ak, signerAddrs[i])
		if err != nil {
			return ctx, err
		}
		pubKey := signerAcc.GetPubKey()

		if simulate && pubKey == nil {
			// In simulate mode the transaction comes with no signatures, thus if the
			// account's pubkey is nil, both signature verification and gasKVStore.Set()
			// shall consume the largest amount, i.e. it takes more gas to verify
			// secp256k1 keys than ed25519 ones.
			if pubKey == nil {
				pubKey = simSecp256k1Pubkey
			}
		}
		err = sgcd.sigGasConsumer(ctx.GasMeter(), sig, pubKey, params)
		if err != nil {
			return ctx, err
		}
	}

	return next(ctx, tx, simulate)
}

// Verify all signatures for a tx and return an error if any are invalid. Note,
// the SigVerificationDecorator decorator will not get executed on ReCheck.
//
// CONTRACT: Pubkeys are set in context for all signers before this decorator runs
// CONTRACT: Tx must implement SigVerifiableTx interface
type SigVerificationDecorator struct {
	ak keeper.AccountKeeper
}

func NewSigVerificationDecorator(ak keeper.AccountKeeper) SigVerificationDecorator {
	return SigVerificationDecorator{
		ak: ak,
	}
}

func (svd SigVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// no need to verify signatures on recheck tx
	if ctx.IsReCheckTx() {
		return next(ctx, tx, simulate)
	}
	sigTx, ok := tx.(SigVerifiableTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "invalid transaction type")
	}

	// stdSigs contains the sequence number, account number, and signatures.
	// When simulating, this would just be a 0-length slice.
	sigs := sigTx.GetSignatures()

	// stdSigs contains the sequence number, account number, and signatures.
	// When simulating, this would just be a 0-length slice.
	signerAddrs := sigTx.GetSigners()
	signerAccs := make([]exported.Account, len(signerAddrs))

	// check that signer length and signature length are the same
	if len(sigs) != len(signerAddrs) {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "invalid number of signer;  expected: %d, got %d", len(signerAddrs), len(sigs))
	}
	var txNonce uint64
	if len(sigs) == 1 && tx.GetNonce() != 0 {
		txNonce = tx.GetNonce()
	}
	for i, sig := range sigs {
		signerAccs[i], err = GetSignerAcc(ctx, svd.ak, signerAddrs[i])
		if err != nil {
			return ctx, err
		}
		if ctx.IsCheckTx() {
			if txNonce != 0 { // txNonce first
				err := nonceVerification(ctx, signerAccs[i].GetSequence(), txNonce, ethcmn.BytesToAddress(signerAddrs[i]).String(), simulate)
				if err != nil {
					return ctx, err
				}
				signerAccs[i].SetSequence(txNonce)
			} else { // for adaptive pending tx in mempool just in checkTx but not deliverTx
				pendingNonce := getCheckTxNonceFromMempool(ethcmn.BytesToAddress(signerAddrs[i]).String())
				if pendingNonce != 0 {
					signerAccs[i].SetSequence(pendingNonce)
				}
			}
		}

		// retrieve signBytes of tx
		signBytes := sigTx.GetSignBytes(ctx, i, signerAccs[i])
		err = sigTx.VerifySequence(i, signerAccs[i])
		if err != nil {
			return ctx, sdkerrors.Wrap(sdkerrors.ErrInvalidSequence, "signature verification sequence failed:"+err.Error())
		}

		// retrieve pubkey
		pubKey := signerAccs[i].GetPubKey()
		if !simulate && pubKey == nil {
			return ctx, sdkerrors.Wrap(sdkerrors.ErrInvalidPubKey, "pubkey on account is not set")
		}

		// verify signature
		if !simulate && (len(signBytes) == 0 || !verifySig(signBytes, sig, pubKey)) {
			//todo: fix sig verification
			//return ctx, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "signature verification failed; verify correct account sequence and chain-id, sign msg:"+string(signBytes))
		}
	}

	return next(ctx, tx, simulate)
}

func verifySig(signBytes, sig []byte, pubKey crypto.PubKey) bool {
	hash := etherhash.Sum(append(signBytes, sig...))
	cachePub, ok := types2.SignatureCache().Get(hash)
	if ok {
		types2.SignatureCache().Remove(hash)
		return bytes.Equal(pubKey.Bytes(), []byte(cachePub))
	}
	if !pubKey.VerifyBytes(signBytes, sig) {
		return false
	}

	types2.SignatureCache().Add(hash, string(pubKey.Bytes()))

	return true
}

// IncrementSequenceDecorator handles incrementing sequences of all signers.
// Use the IncrementSequenceDecorator decorator to prevent replay attacks. Note,
// there is no need to execute IncrementSequenceDecorator on RecheckTX since
// CheckTx would already bump the sequence number.
//
// NOTE: Since CheckTx and DeliverTx state are managed separately, subsequent and
// sequential txs orginating from the same account cannot be handled correctly in
// a reliable way unless sequence numbers are managed and tracked manually by a
// client. It is recommended to instead use multiple messages in a tx.
type IncrementSequenceDecorator struct {
	ak keeper.AccountKeeper
}

func NewIncrementSequenceDecorator(ak keeper.AccountKeeper) IncrementSequenceDecorator {
	return IncrementSequenceDecorator{
		ak: ak,
	}
}

func (isd IncrementSequenceDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	sigTx, ok := tx.(SigVerifiableTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "invalid transaction type")
	}

	if isd.JudgeIncontinuousNonce(ctx, tx, sigTx.GetSigners(), simulate) { // it's the same as handle evm tx
		return next(ctx, tx, simulate)
	}

	// increment sequence of all signers
	for index, addr := range sigTx.GetSigners() {
		acc := isd.ak.GetAccount(ctx, addr)
		// for adaptive pending tx in mempool just in checkTx but not deliverTx
		if ctx.IsCheckTx() && !ctx.IsReCheckTx() {
			pendingNonce := getCheckTxNonceFromMempool(ethcmn.BytesToAddress(addr).String())
			if pendingNonce != 0 {
				acc.SetSequence(pendingNonce)
			}
		}
		if ctx.IsCheckTx() && index == 0 { // context with the nonce of fee payer
			ctx.SetAccountNonce(acc.GetSequence())
		}
		if err := acc.SetSequence(acc.GetSequence() + 1); err != nil {
			panic(err)
		}

		isd.ak.SetAccount(ctx, acc)
	}

	return next(ctx, tx, simulate)
}

// judge the incontinuous nonce, incontinuous nonce no need increment sequence
func (isd IncrementSequenceDecorator) JudgeIncontinuousNonce(ctx sdk.Context, tx sdk.Tx, addrs []sdk.AccAddress, simulate bool) bool {
	txNonce := tx.GetNonce()
	if simulate ||
		(txNonce == 0) || // no wrapCMtx no need verify
		!ctx.IsCheckTx() || // deliverTx mode no need judge
		ctx.IsReCheckTx() {
		return false
	}
	if len(addrs) == 1 && txNonce != 0 {
		acc := isd.ak.GetAccount(ctx, addrs[0])
		if acc.GetSequence() != txNonce { // incontinuous nonce no need increment sequence
			if ctx.IsCheckTx() { // context with the nonce of fee payer
				ctx.SetAccountNonce(acc.GetSequence())
			}
			return true
		}
	}
	return false
}

// ValidateSigCountDecorator takes in Params and returns errors if there are too many signatures in the tx for the given params
// otherwise it calls next AnteHandler
// Use this decorator to set parameterized limit on number of signatures in tx
// CONTRACT: Tx must implement SigVerifiableTx interface
type ValidateSigCountDecorator struct {
	ak keeper.AccountKeeper
}

func NewValidateSigCountDecorator(ak keeper.AccountKeeper) ValidateSigCountDecorator {
	return ValidateSigCountDecorator{
		ak: ak,
	}
}

func (vscd ValidateSigCountDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	sigTx, ok := tx.(SigVerifiableTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a sigTx")
	}

	params := vscd.ak.GetParams(ctx)
	pubKeys := sigTx.GetPubKeys()

	sigCount := 0
	for _, pk := range pubKeys {
		sigCount += types.CountSubKeys(pk)
		if uint64(sigCount) > params.TxSigLimit {
			return ctx, sdkerrors.Wrapf(sdkerrors.ErrTooManySignatures,
				"signatures: %d, limit: %d", sigCount, params.TxSigLimit)
		}
	}

	return next(ctx, tx, simulate)
}

// DefaultSigVerificationGasConsumer is the default implementation of SignatureVerificationGasConsumer. It consumes gas
// for signature verification based upon the public key type. The cost is fetched from the given params and is matched
// by the concrete type.
func DefaultSigVerificationGasConsumer(
	meter sdk.GasMeter, sig []byte, pubkey crypto.PubKey, params types.Params,
) error {
	switch pubkey := pubkey.(type) {
	case ed25519.PubKeyEd25519:
		meter.ConsumeGas(params.SigVerifyCostED25519, "ante verify: ed25519")
		return sdkerrors.Wrap(sdkerrors.ErrInvalidPubKey, "ED25519 public keys are unsupported")

	case secp256k1.PubKeySecp256k1:
		meter.ConsumeGas(params.SigVerifyCostSecp256k1, "ante verify: secp256k1")
		return nil

	case multisig.PubKeyMultisigThreshold:
		var multisignature multisig.Multisignature
		codec.Cdc.MustUnmarshalBinaryBare(sig, &multisignature)

		ConsumeMultisignatureVerificationGas(meter, multisignature, pubkey, params)
		return nil

	default:
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidPubKey, "unrecognized public key type: %T", pubkey)
	}
}

// ConsumeMultisignatureVerificationGas consumes gas from a GasMeter for verifying a multisig pubkey signature
func ConsumeMultisignatureVerificationGas(meter sdk.GasMeter,
	sig multisig.Multisignature, pubkey multisig.PubKeyMultisigThreshold,
	params types.Params) {
	size := sig.BitArray.Size()
	sigIndex := 0
	for i := 0; i < size; i++ {
		if sig.BitArray.GetIndex(i) {
			DefaultSigVerificationGasConsumer(meter, sig.Sigs[sigIndex], pubkey.PubKeys[i], params)
			sigIndex++
		}
	}
}

// GetSignerAcc returns an account for a given address that is expected to sign
// a transaction.
func GetSignerAcc(ctx sdk.Context, ak keeper.AccountKeeper, addr sdk.AccAddress) (exported.Account, error) {
	if acc := ak.GetAccount(ctx, addr); acc != nil {
		return acc, nil
	}
	return nil, sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "account %s does not exist", addr)
}
