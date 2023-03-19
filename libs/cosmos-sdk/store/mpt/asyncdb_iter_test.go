package mpt

import (
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAsyncdbIterator(t *testing.T) {
	memDb := memorydb.New()
	asyncDb := NewAsyncKeyValueStore(memDb, true)
	iter := asyncDb.NewIterator(nil, nil)
	require.False(t, iter.Next())
	require.Nil(t, iter.Key())
	require.Nil(t, iter.Value())
	require.NoError(t, iter.Error())

}
