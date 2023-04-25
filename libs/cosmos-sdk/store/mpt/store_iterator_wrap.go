package mpt

import (
	"bytes"
	"sort"

	ethstate "github.com/ethereum/go-ethereum/core/state"
)

// wrapIterator is a wrap of mpt iterator which can be iterated by the origin key.
// It is compatible with cachekv iterator.
type wrapIterator struct {
	*mptIterator

	start, end []byte
	cacheKeys  [][]byte

	isStorage bool
}

func newWrapIterator(t ethstate.Trie, start, end []byte, ascending bool) *wrapIterator {
	if IsStorageKey(start) {
		return newWrapIteratorStorage(t, start, end, ascending)
	}
	return newWrapIteratorAcc(t, start, end, ascending)
}

func newWrapIteratorAcc(t ethstate.Trie, start, end []byte, ascending bool) *wrapIterator {
	var keys [][]byte
	mptIter := newOriginIterator(t, nil, nil)
	for ; mptIter.Valid(); mptIter.Next() {
		key := mptIter.Key()
		if start != nil && bytes.Compare(key, start) < 0 {
			//start is included
			continue
		}
		if end != nil && bytes.Compare(key, end) >= 0 {
			//end is not included
			continue
		}
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if ascending {
			return bytes.Compare(keys[i], keys[j]) < 0
		}
		return bytes.Compare(keys[i], keys[j]) >= 0
	})

	return &wrapIterator{
		mptIterator: mptIter,
		start:       start,
		end:         end,
		cacheKeys:   keys,
	}
}

func newWrapIteratorStorage(t ethstate.Trie, startIn, endIn []byte, ascending bool) *wrapIterator {
	var keys [][]byte
	_, _, start := decodeAddressStorageInfo(startIn)
	_, _, end := decodeAddressStorageInfo(endIn)
	mptIter := newOriginIterator(t, nil, nil)
	for ; mptIter.Valid(); mptIter.Next() {
		key := mptIter.Key()
		if len(start) != 0 && bytes.Compare(key, start) < 0 {
			//start is included
			continue
		}
		if len(end) != 0 && bytes.Compare(key, end) >= 0 {
			//end is not included
			continue
		}
		keys = append(keys, append(startIn[:minWasmStorageKeySize], key...))
	}
	sort.Slice(keys, func(i, j int) bool {
		if ascending {
			return bytes.Compare(keys[i], keys[j]) < 0
		}
		return bytes.Compare(keys[i], keys[j]) >= 0
	})

	return &wrapIterator{
		mptIterator: mptIter,
		start:       startIn,
		end:         endIn,
		cacheKeys:   keys,
		isStorage:   true,
	}
}

func (it *wrapIterator) Domain() ([]byte, []byte) {
	return it.start, it.end
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
	key := it.Key()
	if it.isStorage {
		_, _, key = decodeAddressStorageInfo(key)
	}
	value, err := it.trie.TryGet(key)
	if err != nil {
		return nil
	}
	return value
}

func (it *wrapIterator) Error() error {
	return nil
}
