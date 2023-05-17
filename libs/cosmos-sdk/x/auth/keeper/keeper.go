package keeper

import (
	"bytes"
	"encoding/binary"
	"fmt"
	ethcmn "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/okbchain/libs/cosmos-sdk/types/errors"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth/exported"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/params/subspace"
	"github.com/okx/okbchain/libs/tendermint/crypto"
	"github.com/okx/okbchain/libs/tendermint/libs/log"
)

// AccountKeeper encodes/decodes accounts using the go-amino (binary)
// encoding/decoding library.
type AccountKeeper struct {
	// The (unexposed) key used to access the store from the Context.
	// Deprecated: Use mptKey instead.
	key sdk.StoreKey

	mptKey sdk.StoreKey

	// The prototypical Account constructor.
	proto func() exported.Account

	// The codec codec for binary encoding/decoding of accounts.
	cdc *codec.Codec

	paramSubspace subspace.Subspace

	observers []ObserverI
}

// NewAccountKeeper returns a new sdk.AccountKeeper that uses go-amino to
// (binary) encode and decode concrete sdk.Accounts.
// nolint
func NewAccountKeeper(
	cdc *codec.Codec, keyMpt sdk.StoreKey, paramstore subspace.Subspace, proto func() exported.Account,
) AccountKeeper {

	return AccountKeeper{
		mptKey:        keyMpt,
		proto:         proto,
		cdc:           cdc,
		paramSubspace: paramstore.WithKeyTable(types.ParamKeyTable()),
	}
}

// Logger returns a module-specific logger.
func (ak AccountKeeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetPubKey Returns the PubKey of the account at address
func (ak AccountKeeper) GetPubKey(ctx sdk.Context, addr sdk.AccAddress) (crypto.PubKey, error) {
	acc := ak.GetAccount(ctx, addr)
	if acc == nil {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "account %s does not exist", addr)
	}
	return acc.GetPubKey(), nil
}

// GetSequence Returns the Sequence of the account at address
func (ak AccountKeeper) GetSequence(ctx sdk.Context, addr sdk.AccAddress) (uint64, error) {
	acc := ak.GetAccount(ctx, addr)
	if acc == nil {
		return 0, sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "account %s does not exist", addr)
	}
	return acc.GetSequence(), nil
}

// GetNextAccountNumber returns and increments the global account number counter.
// If the global account number is not set, it initializes it with value 0.
func (ak AccountKeeper) GetNextAccountNumber(ctx sdk.Context) uint64 {
	var accNumber uint64
	store := ak.paramSubspace.CustomKVStore(ctx)
	bz := store.Get(types.GlobalAccountNumberKey)
	if len(bz) != 0 {
		accNumber = binary.BigEndian.Uint64(bz)
	}
	temp := make([]byte, 8)
	binary.BigEndian.PutUint64(temp, accNumber+1)
	store.Set(types.GlobalAccountNumberKey, temp)

	return accNumber
}

// -----------------------------------------------------------------------------
// Misc.

func (ak AccountKeeper) decodeAccount(bz []byte) exported.Account {
	val, err := ak.cdc.UnmarshalBinaryBareWithRegisteredUnmarshaller(bz, (*exported.Account)(nil))
	if err == nil {
		return val.(exported.Account)
	}
	var acc exported.Account
	err = ak.cdc.UnmarshalBinaryBare(bz, &acc)
	if err != nil {
		panic(err)
	}
	return acc
}

func (ak AccountKeeper) RetrieveStateRoot(bz []byte) ethcmn.Hash {
	var acc exported.Account
	val, err := ak.cdc.UnmarshalBinaryBareWithRegisteredUnmarshaller(bz, &acc)
	if err == nil {
		acc = val.(exported.Account)
		fmt.Println("ans---", acc.GetAddress().String(), acc.GetStateRoot().String())
		return acc.GetStateRoot()
	}
	err = ak.cdc.UnmarshalBinaryBare(bz, &acc)
	if err == nil {
		return acc.GetStateRoot()
	}
	return ethtypes.EmptyRootHash
}

func (ak AccountKeeper) ModifyAccStateRoot(before []byte, rootHash ethcmn.Hash) []byte {
	acc := ak.decodeAccount(before)
	if bytes.Equal(acc.GetStateRoot().Bytes(), rootHash.Bytes()) {
		return before
	}

	if eAcc, ok := acc.(interface{ SetStateRoot(hash ethcmn.Hash) }); ok {
		eAcc.SetStateRoot(rootHash)
	} else {
		panic("unExcepted behavior: mpt store acc should implement SetStateRoot ")
	}
	return ak.encodeAccount(acc)
}

func (ak AccountKeeper) GetAccStateRoot(rootBytes []byte) ethcmn.Hash {
	acc := ak.decodeAccount(rootBytes)
	return acc.GetStateRoot()
}

func (ak AccountKeeper) GetStateRootAndCodeHash(bz []byte) (ethcmn.Hash, []byte) {
	acc := ak.decodeAccount(bz)

	return acc.GetStateRoot(), acc.GetCodeHash()
}
