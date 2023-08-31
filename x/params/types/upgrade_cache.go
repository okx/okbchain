package types

import (
	"fmt"
	"sync"

	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	"github.com/okx/brczero/libs/cosmos-sdk/store/prefix"
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	"github.com/okx/brczero/libs/tendermint/libs/log"
)

var (
	upgradeInfoPreifx = []byte("upgrade")
)

type UpgradeCache struct {
	storeKey *sdk.KVStoreKey
	logger   log.Logger
	cdc      *codec.Codec

	readyLock       sync.Mutex
	upgradeReadyMap map[string][]func(UpgradeInfo)
}

func NewUpgreadeCache(storeKey *sdk.KVStoreKey, logger log.Logger, cdc *codec.Codec) *UpgradeCache {
	return &UpgradeCache{
		storeKey: storeKey,
		logger:   logger,
		cdc:      cdc,

		upgradeReadyMap: make(map[string][]func(UpgradeInfo)),
	}
}

func (uc *UpgradeCache) ReadUpgradeInfo(ctx sdk.Context, name string) (UpgradeInfo, error) {
	return readUpgradeInfoFromStore(ctx, name, uc.storeKey, uc.cdc)
}

func (uc *UpgradeCache) ClaimReadyForUpgrade(name string, cb func(UpgradeInfo)) {
	uc.writeClaim(name, cb)
}

func (uc *UpgradeCache) QueryReadyForUpgrade(name string) ([]func(UpgradeInfo), bool) {
	return uc.readClaim(name)
}

func (uc *UpgradeCache) WriteUpgradeInfo(ctx sdk.Context, info UpgradeInfo, forceCover bool) sdk.Error {
	return writeUpgradeInfoToStore(ctx, info, forceCover, uc.storeKey, uc.cdc, uc.logger)
}

func (uc *UpgradeCache) IsUpgradeExist(ctx sdk.Context, name string) bool {
	store := getUpgradeStore(ctx, uc.storeKey)
	return store.Has([]byte(name))
}

func (uc *UpgradeCache) IterateAllUpgradeInfo(ctx sdk.Context, cb func(info UpgradeInfo) (stop bool)) sdk.Error {
	store := getUpgradeStore(ctx, uc.storeKey)
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		data := iterator.Value()

		var info UpgradeInfo
		uc.cdc.MustUnmarshalJSON(data, &info)

		if stop := cb(info); stop {
			break
		}
	}

	return nil
}

func (uc *UpgradeCache) readClaim(name string) ([]func(UpgradeInfo), bool) {
	uc.readyLock.Lock()
	defer uc.readyLock.Unlock()

	cb, ok := uc.upgradeReadyMap[name]
	return cb, ok
}

func (uc *UpgradeCache) writeClaim(name string, cb func(UpgradeInfo)) {
	uc.readyLock.Lock()
	defer uc.readyLock.Unlock()

	readies, ok := uc.upgradeReadyMap[name]
	if !ok {
		uc.upgradeReadyMap[name] = []func(UpgradeInfo){cb}
	} else {
		uc.upgradeReadyMap[name] = append(readies, cb)
	}
}

func readUpgradeInfoFromStore(ctx sdk.Context, name string, skey *sdk.KVStoreKey, cdc *codec.Codec) (UpgradeInfo, sdk.Error) {
	store := getUpgradeStore(ctx, skey)

	data := store.Get([]byte(name))
	if len(data) == 0 {
		err := fmt.Errorf("upgrade '%s' is not exist", name)
		return UpgradeInfo{}, err
	}

	var info UpgradeInfo
	cdc.MustUnmarshalJSON(data, &info)
	return info, nil
}

func writeUpgradeInfoToStore(ctx sdk.Context, info UpgradeInfo, forceCover bool, skey *sdk.KVStoreKey, cdc *codec.Codec, logger log.Logger) sdk.Error {
	key := []byte(info.Name)

	store := getUpgradeStore(ctx, skey)
	if !forceCover && store.Has(key) {
		logger.Error("upgrade proposal name has been exist", "proposal name", info.Name)
		return sdk.ErrInternal(fmt.Sprintf("upgrade proposal name '%s' has been exist", info.Name))
	}

	data := cdc.MustMarshalJSON(info)
	store.Set(key, data)

	return nil
}

func getUpgradeStore(ctx sdk.Context, skey *sdk.KVStoreKey) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(skey), upgradeInfoPreifx)
}
