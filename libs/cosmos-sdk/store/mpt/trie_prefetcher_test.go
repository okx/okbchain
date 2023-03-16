// Code is copied from https://github.com/ethereum/go-ethereum/blob/v1.10.25/core/state/trie_prefetcher_test.go

package mpt

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	ethstate "github.com/ethereum/go-ethereum/core/state"
)

func filledStateDB() (*ethstate.StateDB, ethstate.Database, common.Hash) {
	db := ethstate.NewDatabase(rawdb.NewMemoryDatabase())
	originalRoot := common.Hash{}
	state, _ := ethstate.New(originalRoot, db, nil)

	// Create an account and check if the retrieved balance is correct
	addr := common.HexToAddress("0xaffeaffeaffeaffeaffeaffeaffeaffeaffeaffe")
	skey := common.HexToHash("aaa")
	sval := common.HexToHash("bbb")

	state.SetBalance(addr, big.NewInt(42)) // Change the account trie
	state.SetCode(addr, []byte("hello"))   // Change an external metadata
	state.SetState(addr, skey, sval)       // Change the storage trie
	for i := 0; i < 100; i++ {
		sk := common.BigToHash(big.NewInt(int64(i)))
		state.SetState(addr, sk, sk) // Change the storage trie
	}
	return state, db, originalRoot
}

func TestUseAfterClose(t *testing.T) {
	_, db, originalRoot := filledStateDB()
	prefetcher := newTriePrefetcher(db, originalRoot, "")
	skey := common.HexToHash("aaa")
	prefetcher.prefetch(common.Hash{}, originalRoot, [][]byte{skey.Bytes()})
	a := prefetcher.trie(common.Hash{}, originalRoot)
	prefetcher.close()
	b := prefetcher.trie(common.Hash{}, originalRoot)
	if a == nil {
		t.Fatal("Prefetching before close should not return nil")
	}
	if b != nil {
		t.Fatal("Trie after close should return nil")
	}
}
