package keeper

import (
	"encoding/binary"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/mpt"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/x/evm/types"
)

/*
 * Getters for keys in x/evm/types/keys.go
 * TODO: these interfaces are used for setting/getting data in rawdb, instead of iavl.
 * TODO: delete these if we decide persist data in iavl.
 */
func (k Keeper) getBlockHashInDiskDB(ctx sdk.Context, hash []byte) (int64, bool) {
	key := types.AppendBlockHashKey(hash)
	st := ctx.MultiStore().GetKVStore(k.storeKey)
	preKey := mpt.PutStoreKey(key)
	bz := st.Get(preKey)
	if bz == nil || len(bz) == 0 {
		return 0, false
	}

	height := binary.BigEndian.Uint64(bz)
	return int64(height), true
}

func (k Keeper) setBlockHashInDiskDB(ctx sdk.Context, hash []byte, height int64) {
	key := types.AppendBlockHashKey(hash)
	bz := sdk.Uint64ToBigEndian(uint64(height))
	st := ctx.MultiStore().GetKVStore(k.storeKey)
	preKey := mpt.PutStoreKey(key)
	st.Set(preKey, bz)
}

func (k Keeper) iterateBlockHashInDiskDB(fn func(key []byte, value []byte) (stop bool)) {
	iterator := k.db.TrieDB().DiskDB().NewIterator(types.KeyPrefixBlockHash, nil)
	defer iterator.Release()
	for iterator.Next() {
		if !types.IsBlockHashKey(iterator.Key()) {
			continue
		}
		key, value := iterator.Key(), iterator.Value()
		if stop := fn(key, value); stop {
			break
		}
	}
}

func (k Keeper) getBlockBloomInDiskDB(height int64) ethtypes.Bloom {
	key := types.AppendBloomKey(height)
	bz, err := k.db.TrieDB().DiskDB().Get(key)
	if err != nil {
		return ethtypes.Bloom{}
	}

	return ethtypes.BytesToBloom(bz)
}

func (k Keeper) setBlockBloomInDiskDB(height int64, bloom ethtypes.Bloom) {
	key := types.AppendBloomKey(height)
	k.db.TrieDB().DiskDB().Put(key, bloom.Bytes())
}

func (k Keeper) iterateBlockBloomInDiskDB(fn func(key []byte, value []byte) (stop bool)) {
	iterator := k.db.TrieDB().DiskDB().NewIterator(types.KeyPrefixBloom, nil)
	defer iterator.Release()
	for iterator.Next() {
		if !types.IsBloomKey(iterator.Key()) {
			continue
		}
		key, value := iterator.Key(), iterator.Value()
		if stop := fn(key, value); stop {
			break
		}
	}
}
