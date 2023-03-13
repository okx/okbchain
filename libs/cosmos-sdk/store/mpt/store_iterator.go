package mpt

import (
	"bytes"
	"encoding/hex"
	"fmt"
	ethstate "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/types"
	db "github.com/okx/okbchain/libs/tm-db"
)

var _ types.Iterator = (*mptIterator)(nil)

type mptIterator struct {
	// Domain
	start, end []byte

	// Underlying store
	iterator *trie.Iterator
	trie     ethstate.Trie

	mdb  *db.MemDB
	iter db.Iterator

	valid bool
}

func newMptIterator(t ethstate.Trie, start, end []byte) *mptIterator {
	iter := &mptIterator{
		iterator: trie.NewIterator(t.NodeIterator(start)),
		trie:     t,
		start:    types.Cp(start),
		end:      nil, // enforce end is nil, because trie iterator origin key is out of order
		valid:    true,
		mdb:      db.NewMemDB(),
	}
	//iter.Next()

	////just for test
	it := trie.NewIterator(t.NodeIterator(nil))
	for it.Next() {
		originKey := t.GetKey(it.Key)
		if bytes.Compare(originKey, start) >= 0 {
			continue
		}
		dkey := make([]byte, len(originKey))
		dvalue := make([]byte, len(it.Value))
		copy(dkey, originKey)
		copy(dvalue, it.Value)
		iter.mdb.Set(dkey, dvalue)

		str := hex.EncodeToString(it.Value)
		fmt.Printf("\"%s\", key %v, originKey %v \n", str, it.Key, originKey)
	}

	iter.iter, _ = iter.mdb.Iterator(start, nil)

	return iter
}

func (it *mptIterator) Domain() (start []byte, end []byte) {
	return it.start, it.end
}

func (it *mptIterator) Valid() bool {
	//return it.valid
	return it.iter.Valid()
}

func (it *mptIterator) Next() {
	//if !it.iterator.Next() || it.iterator.Key == nil {
	//	it.valid = false
	//}
	it.iter.Next()
}

func (it *mptIterator) Key() []byte {
	//key := it.iterator.Key
	//originKey := it.trie.GetKey(key)
	//return originKey
	return it.iter.Key()
}

func (it *mptIterator) Value() []byte {
	//value := it.iterator.Value
	//return value
	return it.iter.Value()
}

func (it *mptIterator) Error() error {
	return it.iterator.Err
}

func (it *mptIterator) Close() {
	return
}
