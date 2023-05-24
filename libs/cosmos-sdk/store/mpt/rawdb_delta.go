package mpt

import (
	"encoding/json"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
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

func (delta *rawdbDelta) Marshal() []byte {
	if !produceDelta {
		return nil
	}
	delta.RLock()
	defer delta.RUnlock()

	ret, _ := json.Marshal(delta)
	return ret
}

func (delta *rawdbDelta) Unmarshal(deltaBytes []byte) error {
	if !applyDelta {
		return nil
	}

	return json.Unmarshal(deltaBytes, delta)
}

func (ms *MptStore) applyRawDBDelta(deltaBytes []byte) {
	if !applyDelta {
		return
	}
	delta := &rawdbDelta{}
	err := delta.Unmarshal(deltaBytes)
	if err != nil {
		ms.logger.Error("applyRawDBDelta unmarshal error", "err", err)
		return
	}
	codeWriter := ms.db.TrieDB().DiskDB().NewBatch()
	defer codeWriter.Reset()
	for k, v := range delta.Codes {
		rawdb.WriteCode(codeWriter, k, v)
	}
	if codeWriter.ValueSize() > 0 {
		if err = codeWriter.Write(); err != nil {
			panic(err)
		}
	}
}
