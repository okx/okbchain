package watcher

import (
	"github.com/okx/okbchain/libs/cosmos-sdk/store/mpt"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
)

func rmStorageRootFromWatchKey(key []byte) []byte {
	if !mpt.IsStorageKey(key) {
		return key
	}
	newKey := make([]byte, 0, len(key)-mpt.StorageRootLen)
	newKey = append(newKey, key[:mpt.PrefixSizeInMpt+sdk.AddrLen]...)
	newKey = append(newKey, key[mpt.PrefixSizeInMpt+sdk.AddrLen+mpt.StorageRootLen:]...)

	return newKey
}
