package keeper

import (
	"math/big"

	"github.com/okx/brczero/x/evm/types"

	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"

	ethcmn "github.com/ethereum/go-ethereum/common"
)

// ----------------------------------------------------------------------------
// Setters, only for test use
// ----------------------------------------------------------------------------

// SetBalance calls CommitStateDB.SetBalance using the passed in context
func (k *Keeper) SetBalance(ctx sdk.Context, addr ethcmn.Address, amount *big.Int) {
	csdb := types.CreateEmptyCommitStateDB(k.GenerateCSDBParams(), ctx)
	csdb.SetBalance(addr, amount)
	csdb.Commit(false)
}

// SetNonce calls CommitStateDB.SetNonce using the passed in context
func (k *Keeper) SetNonce(ctx sdk.Context, addr ethcmn.Address, nonce uint64) {
	csdb := types.CreateEmptyCommitStateDB(k.GenerateCSDBParams(), ctx)
	csdb.SetNonce(addr, nonce)
	csdb.Commit(false)
}

// ----------------------------------------------------------------------------
// Getters, for test and query case
// ----------------------------------------------------------------------------

// GetBalance calls CommitStateDB.GetBalance using the passed in context
func (k *Keeper) GetBalance(ctx sdk.Context, addr ethcmn.Address) *big.Int {
	return types.CreateEmptyCommitStateDB(k.GenerateCSDBParams(), ctx).GetBalance(addr)
}

// GetCode calls CommitStateDB.GetCode using the passed in context
func (k *Keeper) GetCode(ctx sdk.Context, addr ethcmn.Address) []byte {
	return types.CreateEmptyCommitStateDB(k.GenerateCSDBParams(), ctx).GetCode(addr)
}

// GetCodeByHash calls CommitStateDB.GetCode using the passed in context
func (k *Keeper) GetCodeByHash(ctx sdk.Context, hash ethcmn.Hash) []byte {
	return types.CreateEmptyCommitStateDB(k.GenerateCSDBParams(), ctx).GetCodeByHash(hash)
}

// GetState calls CommitStateDB.GetState using the passed in context
func (k *Keeper) GetState(ctx sdk.Context, addr ethcmn.Address, hash ethcmn.Hash) ethcmn.Hash {
	return types.CreateEmptyCommitStateDB(k.GenerateCSDBParams(), ctx).GetState(addr, hash)
}

// GetState calls CommitStateDB.GetState using the passed in context
func (k *Keeper) GetStorageProof(ctx sdk.Context, addr ethcmn.Address, key ethcmn.Hash) ([][]byte, error) {
	return types.CreateEmptyCommitStateDB(k.GenerateCSDBParams(), ctx).GetStorageProof(addr, key)
}

// GetStateByKey calls CommitStateDB.GetState using the passed in context
func (k *Keeper) GetStateByKey(ctx sdk.Context, addr ethcmn.Address, key ethcmn.Hash) []byte {
	return types.CreateEmptyCommitStateDB(k.GenerateCSDBParams(), ctx).GetCommittedStateForQuery(addr, key)
}

// ----------------------------------------------------------------------------
// Auxiliary, for test and query case
// ----------------------------------------------------------------------------

// ForEachStorage calls CommitStateDB.ForEachStorage using passed in context
func (k *Keeper) ForEachStorage(ctx sdk.Context, addr ethcmn.Address, cb func(key, value ethcmn.Hash) bool) error {
	return types.CreateEmptyCommitStateDB(k.GenerateCSDBParams(), ctx).ForEachStorage(addr, cb)
}

// GetOrNewStateObject calls CommitStateDB.GetOrNetStateObject using the passed in context
func (k *Keeper) GetOrNewStateObject(ctx sdk.Context, addr ethcmn.Address) types.StateObject {
	return types.CreateEmptyCommitStateDB(k.GenerateCSDBParams(), ctx).GetOrNewStateObject(addr)
}
