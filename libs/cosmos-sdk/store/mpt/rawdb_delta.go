package mpt

import (
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/trie"
	"sync"
)

var (
	rawdbDeltaInstance *rawdbDelta
	rawdbOnce          sync.Once
)

func GetRawDBDeltaInstance() *rawdbDelta {
	rawdbOnce.Do(func() {
		rawdbDeltaInstance = newRawdbDelta()
	})
	return rawdbDeltaInstance
}

type rawdbDelta struct {
	sync.RWMutex
	Codes map[ethcmn.Hash][]byte
}

func newRawdbDelta() *rawdbDelta {
	return &rawdbDelta{
		Codes: make(map[ethcmn.Hash][]byte),
	}
}

func (delta *rawdbDelta) SetCode(hash ethcmn.Hash, code []byte) {
	if !produceDelta {
		return
	}
	delta.Lock()
	defer delta.Unlock()
	delta.Codes[hash] = code
}

func (delta *rawdbDelta) reset() {
	delta.Codes = make(map[ethcmn.Hash][]byte)
}

func (delta *rawdbDelta) fillCodeDelta(mptDelta *trie.MptDelta) {
	if !produceDelta {
		return
	}
	delta.RLock()
	defer delta.RUnlock()

	mptDelta.CodeKV = make([]*trie.DeltaKV, 0, len(delta.Codes))
	for k, v := range delta.Codes {
		mptDelta.CodeKV = append(mptDelta.CodeKV, &trie.DeltaKV{k.Bytes(), v})
	}
}

func (ms *MptStore) applyRawDBDelta(delta []*trie.DeltaKV) {
	if !applyDelta {
		return
	}
	codeWriter := ms.db.TrieDB().DiskDB().NewBatch()
	defer codeWriter.Reset()
	for _, code := range delta {
		rawdb.WriteCode(codeWriter, ethcmn.BytesToHash(code.Key), code.Val)
	}
	if codeWriter.ValueSize() > 0 {
		if err := codeWriter.Write(); err != nil {
			panic(err)
		}
	}
}
