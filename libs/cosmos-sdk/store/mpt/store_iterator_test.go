package mpt

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	ethstate "github.com/ethereum/go-ethereum/core/state"
	"github.com/stretchr/testify/require"
)

var keyFormat = "key-%08d"

func TestStoreIterate(t *testing.T) {
	cases := []struct {
		num       int
		ascending bool
	}{
		{0, true},
		{1, true},
		{2, true},
		{100, true},
		{1000, true},
		{10000, true},
		{0, false},
		{1, false},
		{2, false},
		{100, false},
		{1000, false},
		{10000, false},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			trie, kvs := fullFillStore(c.num)
			iter := newMptIterator(trie, nil, nil, c.ascending)
			defer iter.Close()
			count := 0
			iKvs := make(map[string]string, c.num)
			var beforeKey []byte
			for ; iter.Valid(); iter.Next() {
				require.NotNil(t, iter.Key())
				curKey := iter.Key()
				iKvs[string(iter.Key())] = string(iter.Value())
				count++
				if len(beforeKey) > 0 {
					if c.ascending {
						require.Equal(t, bytes.Compare(beforeKey, curKey), -1)
					} else {
						require.Equal(t, bytes.Compare(beforeKey, curKey), 1)
					}
				}
				beforeKey = curKey
			}

			require.EqualValues(t, kvs, iKvs)
			require.Equal(t, c.num, len(iKvs))
			require.Equal(t, c.num, count)
		})
	}
}

func TestMptStorageIterate(t *testing.T) {
	var testCases = []struct {
		num         int
		start       int
		end         int
		resultCount int
		ascending   bool
	}{
		{0, 0, 0, 0, true},
		{1, 0, 0, 0, true},
		{1, 1, 0, 0, true},
		{2, 0, 0, 0, true},
		{2, 1, 1, 0, true},
		{2, 2, 1, 0, true},
		{2, 3, 1, 0, true},
		{100, 0, 0, 0, true},
		{100, 0, 100, 100, true},
		{100, 1, 100, 99, true},
		{100, 50, 60, 10, true},
		{100, 50, 50, 0, true},
		{100, 51, 50, 0, true},
		{0, 0, 0, 0, false},
		{1, 0, 0, 0, false},
		{1, 1, 0, 0, false},
		{2, 0, 0, 0, false},
		{2, 1, 1, 0, false},
		{2, 2, 1, 0, false},
		{2, 3, 1, 0, false},
		{100, 0, 0, 0, false},
		{100, 0, 100, 100, false},
		{100, 1, 100, 99, false},
		{100, 50, 60, 10, false},
		{100, 50, 50, 0, false},
		{100, 51, 50, 0, false},
	}

	for i, c := range testCases {
		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			pre := AddressStoragePrefixMpt(common.HexToAddress("0xbbe4733d85bc2b90682147779da49cab38c0aa1f"), common.HexToHash("0xb4a40e844ee4c012d4a6d9e16d4ee8dcf52ef5042da491dbc73574f6764e17d1"))
			trie, _ := fullFillStore(c.num)
			iter := newMptIterator(trie, cloneAppend(pre, []byte(fmt.Sprintf(keyFormat, c.start))),
				cloneAppend(pre, []byte(fmt.Sprintf(keyFormat, c.end))), c.ascending)
			defer iter.Close()
			iKvs := make(map[string]string, c.num)
			var beforeKey []byte
			for ; iter.Valid(); iter.Next() {
				_, _, curKey := decodeAddressStorageInfo(iter.Key())
				iKvs[string(curKey)] = string(iter.Value())
				if len(beforeKey) > 0 {
					if c.ascending {
						require.Equal(t, bytes.Compare(beforeKey, curKey), -1)
					} else {
						require.Equal(t, bytes.Compare(beforeKey, curKey), 1)
					}
				}
				beforeKey = curKey
			}
			require.Equal(t, c.resultCount, len(iKvs))
		})
	}
}

func fullFillStore(num int) (ethstate.Trie, map[string]string) {
	db := ethstate.NewDatabase(rawdb.NewMemoryDatabase())
	tr, err := db.OpenTrie(NilHash)
	if err != nil {
		panic("Fail to open root mpt: " + err.Error())
	}

	kvs := make(map[string]string, num)
	for i := 0; i < num; i++ {
		k, v := fmt.Sprintf(keyFormat, i), fmt.Sprintf("value-%d", i)
		kvs[k] = v
		if err := tr.TryUpdate([]byte(k), []byte(v)); err != nil {
			panic(err)
		}
	}
	return tr, kvs
}

func TestWrapIterator(t *testing.T) {
	total := 10000
	trie, kvs := fullFillStore(total)

	iter := newMptIterator(trie, nil, nil, true)
	defer iter.Close()
	var count int
	for ; iter.Valid(); iter.Next() {
		count++
	}
	require.Equal(t, len(kvs), count)

	startIndex := 3000
	iter2 := newMptIterator(trie, []byte(fmt.Sprintf(keyFormat, startIndex)), nil, true)
	defer iter2.Close()
	var count2 int
	for ; iter2.Valid(); iter2.Next() {
		count2++
	}
	require.Equal(t, total-startIndex, count2)

}
