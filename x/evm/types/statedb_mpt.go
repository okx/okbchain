package types

import (
	"errors"
	"fmt"
	ethermint "github.com/okx/okbchain/app/types"

	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/mpt"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/tendermint/libs/log"
)

func (csdb *CommitStateDB) CommitMpt(prefetcher *mpt.TriePrefetcher) (ethcmn.Hash, error) {
	// Commit objects to the trie, measuring the elapsed time
	codeWriter := csdb.db.TrieDB().DiskDB().NewBatch()

	for addr := range csdb.stateObjectsDirty {
		if obj := csdb.stateObjects[addr]; !obj.deleted {
			// Write any contract code associated with the state object
			if obj.code != nil && obj.dirtyCode {
				rawdb.WriteCode(codeWriter, ethcmn.BytesToHash(obj.CodeHash()), obj.code)
				obj.dirtyCode = false
			}

			// Write any storage changes in the state object to its storage trie
			if err := obj.CommitTrie(csdb.db); err != nil {
				return ethcmn.Hash{}, err
			}

			accProto := csdb.accountKeeper.GetAccount(csdb.ctx, obj.account.Address)
			if ethermintAccount, ok := accProto.(*ethermint.EthAccount); ok {
				ethermintAccount.StateRoot = obj.account.StateRoot
				//fmt.Println("fffffffff", ethermintAccount.EthAddress().String(), ethermintAccount.StateRoot.String())
				csdb.accountKeeper.SetAccount(csdb.ctx, ethermintAccount)
			}
		}
	}

	for addr := range csdb.stateObjectsDirty {
		delete(csdb.stateObjectsDirty, addr)
	}

	if codeWriter.ValueSize() > 0 {
		if err := codeWriter.Write(); err != nil {
			csdb.SetError(fmt.Errorf("failed to commit dirty codes: %s", err.Error()))
		}
	}

	return ethcmn.Hash{}, nil
}

func (csdb *CommitStateDB) ForEachStorageMpt(so *stateObject, cb func(key, value ethcmn.Hash) (stop bool)) error {
	it := trie.NewIterator(so.getTrie(csdb.db).NodeIterator(nil))
	for it.Next() {
		key := ethcmn.BytesToHash(so.trie.GetKey(it.Key))
		if value, dirty := so.dirtyStorage[key]; dirty {
			if cb(key, value) {
				return nil
			}
			continue
		}

		if len(it.Value) > 0 {
			_, content, _, err := rlp.Split(it.Value)
			if err != nil {
				return err
			}
			if cb(key, ethcmn.BytesToHash(content)) {
				return nil
			}
		}
	}

	return nil
}

func (csdb *CommitStateDB) GetCodeByHashInRawDB(hash ethcmn.Hash) []byte {
	code, err := csdb.db.ContractCode(ethcmn.Hash{}, hash)
	if err != nil {
		return nil
	}

	return code
}

func (csdb *CommitStateDB) setHeightHashInRawDB(height uint64, hash ethcmn.Hash) {
	key := AppendHeightHashKey(height)
	csdb.db.TrieDB().DiskDB().Put(key, hash.Bytes())
}

func (csdb *CommitStateDB) getHeightHashInRawDB(height uint64) ethcmn.Hash {
	key := AppendHeightHashKey(height)
	bz, err := csdb.db.TrieDB().DiskDB().Get(key)
	if err != nil {
		return ethcmn.Hash{}
	}
	return ethcmn.BytesToHash(bz)
}

// getDeletedStateObject is similar to getStateObject, but instead of returning
// nil for a deleted state object, it returns the actual object with the deleted
// flag set. This is needed by the state journal to revert to the correct s-
// destructed object instead of wiping all knowledge about the state object.
func (csdb *CommitStateDB) getDeletedStateObject(addr ethcmn.Address) *stateObject {
	// Prefer live objects if any is available
	if obj := csdb.stateObjects[addr]; obj != nil {
		if _, ok := csdb.updatedAccount[addr]; ok {
			delete(csdb.updatedAccount, addr)
			if err := obj.UpdateAccInfo(); err != nil {
				csdb.SetError(err)
				return nil
			}
		}
		return obj
	}

	// otherwise, attempt to fetch the account from the account mapper
	acc := csdb.accountKeeper.GetAccount(csdb.ctx, sdk.AccAddress(addr.Bytes()))
	if acc == nil {
		csdb.SetError(fmt.Errorf("no account found for address: %s", addr.String()))
		return nil
	}

	// insert the state object into the live set
	so := newStateObject(csdb, acc)
	csdb.setStateObject(so)

	return so
}

func (csdb *CommitStateDB) MarkUpdatedAcc(addList []ethcmn.Address) {
	for _, addr := range addList {
		csdb.updatedAccount[addr] = struct{}{}
	}
}

// TODO this line code only get contract_storage_merkle_proof, have not acc_merkle_proof
// GetStorageProof returns the Merkle proof for given storage slot.
func (csdb *CommitStateDB) GetStorageProof(a ethcmn.Address, key ethcmn.Hash) ([][]byte, error) {
	var proof mpt.ProofList
	addrTrie := csdb.StorageTrie(a)
	if addrTrie == nil {
		return proof, errors.New("storage trie for requested address does not exist")
	}
	err := addrTrie.Prove(crypto.Keccak256(key.Bytes()), 0, &proof)
	return proof, err
}

// ----------------------------------------------------------------------------
// Proof related
// ----------------------------------------------------------------------------

// GetProof returns the Merkle proof for a given account.
func (csdb *CommitStateDB) GetProof(addr ethcmn.Address) ([][]byte, error) {
	return csdb.GetProofByHash(crypto.Keccak256Hash(addr.Bytes()))
}

// GetProofByHash returns the Merkle proof for a given account.
func (csdb *CommitStateDB) GetProofByHash(addrHash ethcmn.Hash) ([][]byte, error) {
	var proof mpt.ProofList
	//Todo need to call acc mpt trie
	//err := csdb.trie.Prove(addrHash[:], 0, &proof)
	return proof, nil
}

func (csdb *CommitStateDB) Logger() log.Logger {
	return csdb.ctx.Logger().With("module", ModuleName)
}

// StartPrefetcher initializes a new trie prefetcher to pull in nodes from the
// state trie concurrently while the state is mutated so that when we reach the
// commit phase, most of the needed data is already hot.
func (csdb *CommitStateDB) StartPrefetcher(namespace string) {

	if csdb.prefetcher != nil {
		csdb.prefetcher.Close()
		csdb.prefetcher = nil
	}

	//csdb.prefetcher = mpt.NewTriePrefetcher(csdb.db, types2.EmptyRootHash, namespace)
}

// StopPrefetcher terminates a running prefetcher and reports any leftover stats
// from the gathered metrics.
func (csdb *CommitStateDB) StopPrefetcher() {
	if csdb.prefetcher != nil {
		csdb.prefetcher.Close()
		csdb.prefetcher = nil
	}
}
