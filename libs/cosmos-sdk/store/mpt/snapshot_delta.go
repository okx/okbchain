package mpt

import (
	"encoding/json"
	ethcmn "github.com/ethereum/go-ethereum/common"
	mpttype "github.com/okx/okbchain/libs/cosmos-sdk/store/mpt/types"
)

type snapshotDelta struct {
	DeltaSnapshotDestructs   map[ethcmn.Hash]struct{}               `json:"snap_destructs"`
	DeltaSnapshotAccounts    map[ethcmn.Hash][]byte                 `json:"snap_accounts"`
	DeltaSnapshotSnapStorage map[ethcmn.Hash]map[ethcmn.Hash][]byte `json:"snap_storage"`
}

func newSnapshotDelta() *snapshotDelta {
	return &snapshotDelta{
		DeltaSnapshotDestructs:   make(map[ethcmn.Hash]struct{}),
		DeltaSnapshotAccounts:    make(map[ethcmn.Hash][]byte),
		DeltaSnapshotSnapStorage: make(map[ethcmn.Hash]map[ethcmn.Hash][]byte),
	}
}

func (delta *snapshotDelta) addDestruct(key []byte) {
	if !produceDelta || delta == nil {
		return
	}
	hash := mpttype.Keccak256HashWithSyncPool(key)
	delete(delta.DeltaSnapshotAccounts, hash)
	delete(delta.DeltaSnapshotSnapStorage, hash)
	delta.DeltaSnapshotDestructs[hash] = struct{}{}
}

func (delta *snapshotDelta) addAccount(key []byte, account []byte) {
	if !produceDelta || delta == nil {
		return
	}
	hash := mpttype.Keccak256HashWithSyncPool(key)
	delete(delta.DeltaSnapshotDestructs, hash)
	delta.DeltaSnapshotAccounts[hash] = account
}

func (delta *snapshotDelta) addStorage(addr ethcmn.Address, realKey, value []byte) {
	if !produceDelta || delta == nil {
		return
	}
	hash := mpttype.Keccak256HashWithSyncPool(AddressStoreKey(addr.Bytes()))
	if _, ok := delta.DeltaSnapshotSnapStorage[hash]; !ok {
		delta.DeltaSnapshotSnapStorage[hash] = make(map[ethcmn.Hash][]byte)
	}

	key := mpttype.Keccak256HashWithSyncPool(realKey)
	delta.DeltaSnapshotSnapStorage[hash][key] = value
}

func (delta *snapshotDelta) resetSnapshotDelta() {
	if !produceDelta || delta == nil {
		return
	}
	delta.DeltaSnapshotDestructs = make(map[ethcmn.Hash]struct{})
	delta.DeltaSnapshotAccounts = make(map[ethcmn.Hash][]byte)
	delta.DeltaSnapshotSnapStorage = make(map[ethcmn.Hash]map[ethcmn.Hash][]byte)
}

func (delta *snapshotDelta) getDestructs() map[ethcmn.Hash]struct{} {
	return delta.DeltaSnapshotDestructs
}

func (delta *snapshotDelta) getAccounts() map[ethcmn.Hash][]byte {
	return delta.DeltaSnapshotAccounts
}

func (delta *snapshotDelta) getStorage() map[ethcmn.Hash]map[ethcmn.Hash][]byte {
	return delta.DeltaSnapshotSnapStorage
}

func (delta *snapshotDelta) Marshal() []byte {
	ret, _ := json.Marshal(delta)
	return ret
}

func (delta *snapshotDelta) Unmarshal(deltaBytes []byte) error {
	return json.Unmarshal(deltaBytes, delta)
}
