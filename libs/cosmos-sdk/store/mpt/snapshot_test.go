package mpt

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	tmlog "github.com/okx/okbchain/libs/tendermint/libs/log"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMptStoreSnapshotDeleteAccount(t *testing.T) {
	db := memorydb.New()
	ethDb := rawdb.NewDatabase(db)
	stateDb := state.NewDatabase(ethDb)

	trie, err := stateDb.OpenTrie(common.Hash{})
	require.NoError(t, err)

	mptStore := &MptStore{
		trie:         trie,
		db:           stateDb,
		logger:       tmlog.NewNopLogger(),
		originalRoot: trie.Hash(),
		retriever:    EmptyStateRootRetriever{},
		triegc:       prque.New(nil),
	}
	SetSnapshotRebuild(true)
	err = mptStore.openSnapshot()
	require.NoError(t, err)
	SetSnapshotRebuild(false)
	err = mptStore.openSnapshot()
	require.NoError(t, err)
	addr := AddressStoreKey(common.Hash{1}.Bytes())
	value := "value1"
	addr2 := AddressStoreKey(common.Hash{2}.Bytes())
	value2 := "value2"
	mptStore.Set(addr, []byte(value))
	mptStore.Set(addr2, []byte(value2))
	mptStore.Delete(addr)
	mptStore.CommitterCommit(nil)
	v := mptStore.Get(addr)
	require.Nil(t, v)
}
