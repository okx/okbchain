package types

import (
	"fmt"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/okx/brczero/libs/cosmos-sdk/store/mpt"
	"github.com/okx/brczero/libs/cosmos-sdk/store/types"
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
)

func (csdb *CommitStateDB) ProtectStateDBEnvironment(ctx sdk.Context) {
	subCtx, commit := ctx.CacheContextWithMultiSnapshotRWSet()
	currentGasMeter := subCtx.GasMeter()
	infGasMeter := sdk.GetReusableInfiniteGasMeter()
	subCtx.SetGasMeter(infGasMeter)
	codeWriter := csdb.db.TrieDB().DiskDB().NewBatch()
	//push dirty object to ctx
	for addr := range csdb.journal.dirties {
		obj, exist := csdb.stateObjects[addr]
		if !exist {
			// ripeMD is 'touched' at block 1714175, in tx 0x1237f737031e40bcde4a8b7e717b2d15e3ecadfe49bb1bbc71ee9deb09c6fcf2
			// That tx goes out of gas, and although the notion of 'touched' does not exist there, the
			// touch-event will still be recorded in the journal. Since ripeMD is a special snowflake,
			// it will persist in the journal even though the journal is reverted. In this special circumstance,
			// it may exist in `s.journal.dirties` but not in `s.stateObjects`.
			// Thus, we can safely ignore it here
			continue
		}
		// Using deepCopy to avoid to obj change, because obj will be used in resetchange.revert
		tempObj := obj.deepCopy(csdb)
		if tempObj.suicided || tempObj.empty() {
			csdb.deleteStateObjectForProtect(subCtx, tempObj)
		} else {
			tempObj.finaliseForProtect() // Prefetch slots in the background
			tempObj.commitStateForProtect(subCtx)
			csdb.updateStateObjectForProtect(subCtx, tempObj)

			// Write any contract code associated with the state object
			if tempObj.code != nil && tempObj.dirtyCode {
				tempObj.commitCodeForProtect(codeWriter)
				tempObj.dirtyCode = false
			}
		}
	}

	if codeWriter.ValueSize() > 0 {
		if err := codeWriter.Write(); err != nil {
			csdb.SetError(fmt.Errorf("failed to commit dirty codes: %s", err.Error()))
		}
	}

	//clear state objects and add revert handle
	for addr, preObj := range csdb.stateObjects {
		delete(csdb.stateObjects, addr)
		//when need to revertsnapshot need resetObject to csdb
		csdb.journal.append(resetObjectChange{prev: preObj})
	}
	//commit data to parent ctx
	///when need to revertsnapshot need restore ctx to prev
	csdb.CMChangeCommit(commit)

	subCtx.SetGasMeter(currentGasMeter)
	sdk.ReturnInfiniteGasMeter(infGasMeter)
}

func (csdb *CommitStateDB) CMChangeCommit(writeCacheWithRWSet func() types.MultiSnapshotWSet) {
	cmwSet := writeCacheWithRWSet()
	csdb.journal.append(cmChange{&cmwSet})
}

// updateStateObject writes the given state object to the store.
func (csdb *CommitStateDB) updateStateObjectForProtect(ctx sdk.Context, so *stateObject) error {
	// NOTE: we don't use sdk.NewCoin here to avoid panic on test importer's genesis
	newBalance := sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewDecFromBigIntWithPrec(so.Balance(), sdk.Precision)} // int2dec
	if !newBalance.IsValid() {
		return fmt.Errorf("invalid balance %s", newBalance)
	}

	//checking and reject tx if address in blacklist
	if csdb.bankKeeper.BlacklistedAddr(so.account.GetAddress()) {
		return fmt.Errorf("address <%s> in blacklist is not allowed", so.account.GetAddress().String())
	}

	coins := so.account.GetCoins()
	balance := coins.AmountOf(newBalance.Denom)
	if balance.IsZero() || !balance.Equal(newBalance.Amount) {
		coins = coins.Add(newBalance)
	}

	if err := so.account.SetCoins(coins); err != nil {
		return err
	}

	csdb.accountKeeper.SetAccount(ctx, so.account)
	if !ctx.IsCheckTx() {
		if ctx.GetWatcher().Enabled() {
			ctx.GetWatcher().SaveAccount(so.account)
		}
	}

	return nil
}

// deleteStateObject removes the given state object from the state store.
func (csdb *CommitStateDB) deleteStateObjectForProtect(ctx sdk.Context, so *stateObject) {
	so.deleted = true
	csdb.accountKeeper.RemoveAccount(ctx, so.account)
}

// finalise moves all dirty storage slots into the pending area to be hashed or
// committed later. It is invoked at the end of every transaction.
func (so *stateObject) finaliseForProtect() {
	for key, value := range so.dirtyStorage {
		so.pendingStorage[key] = value
	}
	for key := range so.dirtyStorage {
		delete(so.dirtyStorage, key)
	}
}

// commitState commits all dirty storage to a KVStore and resets
// the dirty storage slice to the empty state.
func (so *stateObject) commitStateForProtect(ctx sdk.Context) {
	// Make sure all dirty slots are finalized into the pending storage area
	so.finaliseForProtect() // Don't prefetch any more, pull directly if need be
	if len(so.pendingStorage) == 0 {
		return
	}

	store := so.stateDB.dbAdapter.NewStore(ctx.KVStore(so.stateDB.storeKey), mpt.AddressStoragePrefixMpt(so.address, so.account.StateRoot))
	for key, value := range so.pendingStorage {
		// Skip noop changes, persist actual changes
		if value == so.originStorage[key] {
			continue
		}
		so.originStorage[key] = value
		copyKey := ethcmn.CopyBytes(key[:])
		if (value == ethcmn.Hash{}) {
			store.Delete(key[:])
			if !ctx.IsCheckTx() {
				if ctx.GetWatcher().Enabled() {
					ctx.GetWatcher().SaveState(so.Address(), copyKey, ethcmn.Hash{}.Bytes())
				}
			}
		} else {
			v, _ := rlp.EncodeToBytes(ethcmn.TrimLeftZeroes(value[:]))
			store.Set(key[:], v)
			if !ctx.IsCheckTx() {
				if ctx.GetWatcher().Enabled() {
					ctx.GetWatcher().SaveState(so.Address(), copyKey, v)
				}
			}
		}
	}

	for key := range so.pendingStorage {
		delete(so.pendingStorage, key)
	}
	return
}

// commitCode persists the state object's code to the KVStore.
func (so *stateObject) commitCodeForProtect(batch ethdb.Batch) {
	rawdb.WriteCode(batch, ethcmn.BytesToHash(so.CodeHash()), so.code)
}
