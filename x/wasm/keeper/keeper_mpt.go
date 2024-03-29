package keeper

import (
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/mpt"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
)

func (k Keeper) getStorageStore(ctx sdk.Context, acc sdk.WasmAddress) sdk.KVStore {
	account := k.accountKeeper.GetAccount(ctx, sdk.WasmToAccAddress(acc))
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
	return k.getStorageStore(ctx, acc)
}
