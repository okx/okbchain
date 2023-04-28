package mpt

import (
	"bytes"
	"fmt"
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

// tryDecodeStorageIteratorEnd when prefix store traverse the storage with (nil,nil) we should modify the end
// prefix.Store may give (nil,nil) as the iterator input.
// prefix.Store may instead (nil, nil) to (prefix, prefix[:len(len(prefix)-1])
// see prefix.Store.Iterator and prefix.cpIter for details
func tryDecodeStorageIteratorEnd(endIn []byte) []byte {
	if len(endIn) < minWasmStorageKeySize {
		return nil
	}
	_, _, end := decodeAddressStorageInfo(endIn)

	return end
}

// tryDecodeStorageIteratorStart check the start should great or equal to minWasmStorageKeySize
func tryDecodeStorageIteratorStart(startIn []byte) []byte {
	if len(startIn) < minWasmStorageKeySize {
		panic(fmt.Sprintf("mpt storage iterator start len should at least %v", minWasmStorageKeySize))
	}
	_, _, start := decodeAddressStorageInfo(startIn)

	return start
}

func newWrapIteratorStorage(t ethstate.Trie, startIn, endIn []byte, ascending bool) *wrapIterator {
	if len(startIn) == 0 || len(endIn) == 0 {
		panic("mpt storage iterator start or end should not be nil")
	}
	var keys [][]byte
	start := tryDecodeStorageIteratorStart(startIn)
	end := tryDecodeStorageIteratorEnd(endIn)
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
		keys = append(keys, cloneAppend(startIn[:minWasmStorageKeySize], key))
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

func cloneAppend(bz []byte, tail []byte) (res []byte) {
	res = make([]byte, len(bz)+len(tail))
	copy(res, bz)
	copy(res[len(bz):], tail)
	return
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
