package keeper

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/mpt"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/prefix"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/x/wasm/watcher"
	"reflect"
)

func (k Keeper) getStorageStore(ctx sdk.Context, acc sdk.WasmAddress) sdk.KVStore {
	account := k.accountKeeper.GetAccount(ctx, sdk.WasmToAccAddress(acc))
	fmt.Println("fffffffffffff", account == nil, &k.accountKeeper)
	ethAcc := common.BytesToAddress(acc.Bytes())

	// in case of query panic
	var stateRoot common.Hash
	if account == nil {
		stateRoot = ethtypes.EmptyRootHash
	} else {
		stateRoot = account.GetStateRoot()
	}
	return k.ada.NewStore(ctx, k.storageStoreKey, mpt.AddressStoragePrefixMpt(ethAcc, stateRoot))
}

func (k Keeper) GetStorageStore4Query(ctx sdk.Context, acc sdk.WasmAddress) sdk.KVStore {
	if watcher.Enable() {
		ethAcc := common.BytesToAddress(acc.Bytes())
		//stateRoot := sdk.GlobalAccStateRootForWasmFastQuery.GetStateRoot(acc.String())

		//fmt.Println("ffff --- ggggg", acc.String(), stateRoot.String())
		fmt.Println("rrrrrrr", &k.accountKeeper, reflect.TypeOf(k.accountKeeper))
		account := k.accountKeeper.GetAccount(ctx, sdk.WasmToAccAddress(acc))
		fmt.Println("from accoutnKeeper", account.GetStateRoot().String())
		return watcher.NewReadStore(nil, prefix.NewStore(ctx.KVStore(k.storageStoreKey), mpt.AddressStoragePrefixMpt(ethAcc, account.GetStateRoot())))

	}

	return k.getStorageStore(ctx, acc)
}
