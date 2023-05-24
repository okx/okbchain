package mpt

import (
	"encoding/json"
	ethcmn "github.com/ethereum/go-ethereum/common"
	mpttype "github.com/okx/okbchain/libs/cosmos-sdk/store/mpt/types"
)

type snapshotDelta struct {
	SnapDestructs map[ethcmn.Hash]struct{}               `json:"snap_destructs"`
	SnapAccounts  map[ethcmn.Hash][]byte                 `json:"snap_accounts"`
	SnapStorage   map[ethcmn.Hash]map[ethcmn.Hash][]byte `json:"snap_storage"`
}

func newSnapshotDelta() *snapshotDelta {
	return &snapshotDelta{
		SnapDestructs: make(map[ethcmn.Hash]struct{}),
		SnapAccounts:  make(map[ethcmn.Hash][]byte),
		SnapStorage:   make(map[ethcmn.Hash]map[ethcmn.Hash][]byte),
	}
}

func (delta *snapshotDelta) addDestruct(key []byte) {
	if !produceDelta || delta == nil {
		return
	}
	hash := mpttype.Keccak256HashWithSyncPool(key)
	delete(delta.SnapAccounts, hash)
	delete(delta.SnapStorage, hash)
	delta.SnapDestructs[hash] = struct{}{}
}

func (delta *snapshotDelta) addAccount(key []byte, account []byte) {
	if !produceDelta || delta == nil {
		return
	}
	hash := mpttype.Keccak256HashWithSyncPool(key)
	delete(delta.SnapDestructs, hash)
	delta.SnapAccounts[hash] = account
}

func (delta *snapshotDelta) addStorage(addr ethcmn.Address, realKey, value []byte) {
	if !produceDelta || delta == nil {
		return
	}
	hash := mpttype.Keccak256HashWithSyncPool(AddressStoreKey(addr.Bytes()))
	if _, ok := delta.SnapStorage[hash]; !ok {
		delta.SnapStorage[hash] = make(map[ethcmn.Hash][]byte)
	}

	key := mpttype.Keccak256HashWithSyncPool(realKey)
	delta.SnapStorage[hash][key] = value
}

func (delta *snapshotDelta) resetSnapshotDelta() {
	if !produceDelta || delta == nil {
		return
	}
	delta.SnapDestructs = make(map[ethcmn.Hash]struct{})
	delta.SnapAccounts = make(map[ethcmn.Hash][]byte)
	delta.SnapStorage = make(map[ethcmn.Hash]map[ethcmn.Hash][]byte)
}

func (delta *snapshotDelta) getDestructs() map[ethcmn.Hash]struct{} {
	return delta.SnapDestructs
}

func (delta *snapshotDelta) getAccounts() map[ethcmn.Hash][]byte {
	return delta.SnapAccounts
}

func (delta *snapshotDelta) getStorage() map[ethcmn.Hash]map[ethcmn.Hash][]byte {
	return delta.SnapStorage
}

func (delta *snapshotDelta) Marshal() []byte {
	ret, _ := json.Marshal(delta)
	return ret
}

func (delta *snapshotDelta) Unmarshal(deltaBytes []byte) error {
	return json.Unmarshal(deltaBytes, delta)
}
