package mpt

import (
	"bytes"
	"sort"

	ethstate "github.com/ethereum/go-ethereum/core/state"
)

// wrapIterator is a wrap of mpt iterator which can be iterated by the origin key.
// It is compatible with cachekv.
type wrapIterator struct {
	*mptIterator

	cacheKeys [][]byte
}

func newWrapIterator(t ethstate.Trie, start, end []byte) *wrapIterator {
	var keys [][]byte
	mptIter := newOriginIterator(t, start, end)
	for ; mptIter.Valid(); mptIter.Next() {
		keys = append(keys, mptIter.Key())
	}
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i], keys[j]) < 0
	})

	return &wrapIterator{
		mptIterator: mptIter,
		cacheKeys:   keys,
	}
}

func (it *wrapIterator) Valid() bool {
	return len(it.cacheKeys) > 0
}

func (it *wrapIterator) Next() {
	if !it.Valid() {
		return
	}
	it.cacheKeys = it.cacheKeys[1:]
}

func (it *wrapIterator) Key() []byte {
	if !it.Valid() {
		return nil
	}
	return it.cacheKeys[0]
}

func (it *wrapIterator) Value() []byte {
	if !it.Valid() {
		return nil
	}
	value, err := it.trie.TryGet(it.Key())
	if err != nil {
		return nil
	}
	return value
}

func (it *wrapIterator) Error() error {
	return nil
}
