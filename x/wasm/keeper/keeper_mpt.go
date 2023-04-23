package keeper

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	apptypes "github.com/okx/okbchain/app/types"
	clientcontext "github.com/okx/okbchain/libs/cosmos-sdk/client/context"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/mpt"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/prefix"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth"
	"github.com/okx/okbchain/x/wasm/watcher"
	"log"
)

func (k Keeper) GetStorageStore(ctx sdk.Context, acc sdk.WasmAddress) sdk.KVStore {
	account := k.accountKeeper.GetAccount(ctx, sdk.WasmToAccAddress(acc))
	ethAcc := common.BytesToAddress(acc.Bytes())

	return k.ada.NewStore(ctx.GasMeter(), ctx.KVStore(k.storageStoreKey), mpt.AddressStoragePrefixMpt(ethAcc, account.GetStateRoot()))
}

func (k Keeper) GetStorageStoreWatch(ctx sdk.Context, acc sdk.WasmAddress) sdk.KVStore {
	account := k.accountKeeper.GetAccount(ctx, sdk.WasmToAccAddress(acc))
	ethAcc := common.BytesToAddress(acc.Bytes())

	store := k.ada.NewStore(ctx.GasMeter(), ctx.KVStore(k.storageStoreKey), nil)
	return prefix.NewStore(store, mpt.AddressStoragePrefixMpt(ethAcc, account.GetStateRoot()))
}

func (k Keeper) getStorageStoreW(ctx sdk.Context, acc sdk.WasmAddress) sdk.KVStore {
	if watcher.Enable() {
		account := getAccount(acc)
		ethAcc := common.BytesToAddress(acc.Bytes())

		store := k.ada.NewStore(ctx.GasMeter(), ctx.KVStore(k.storageStoreKey), nil)
		return prefix.NewStore(store, mpt.AddressStoragePrefixMpt(ethAcc, account.GetStateRoot()))
	}
	return k.GetStorageStore(ctx, acc)
}

func (k Keeper) GetStorageStoreW(ctx sdk.Context, acc sdk.WasmAddress) sdk.KVStore {
	if watcher.Enable() {
		ethAcc := common.BytesToAddress(acc.Bytes())
		store := k.ada.NewStore(ctx.GasMeter(), ctx.KVStore(k.storageStoreKey), nil)

		return prefix.NewStore(store, mpt.AddressStorageWithoutStorageRootPrefixMpt(ethAcc))
	}

	account := k.accountKeeper.GetAccount(ctx, sdk.WasmToAccAddress(acc))
	ethAcc := common.BytesToAddress(acc.Bytes())

	pre := mpt.AddressStoragePrefixMpt(ethAcc, account.GetStateRoot())
	log.Printf("prefix %s\n", fmt.Sprintf("%x", pre))

	return prefix.NewStore(ctx.KVStore(k.storageStoreKey), pre)
	// return k.GetStorageStoreWatch(ctx, acc)
}

var clientCtx clientcontext.CLIContext

func SetCliContext(ctx clientcontext.CLIContext) {
	clientCtx = ctx
}

func getAccount(addr sdk.WasmAddress) *apptypes.EthAccount {
	bs, err := clientCtx.Codec.MarshalJSON(auth.NewQueryAccountParams(addr.Bytes()))
	if err != nil {
		log.Println("GetAllBalances marshal json error", err)
		return nil
	}
	res, _, err := clientCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", auth.QuerierRoute, auth.QueryAccount), bs)
	if err != nil {
		log.Println("GetAllBalances query with data error", err)
		return nil
	}
	var account apptypes.EthAccount
	err = clientCtx.Codec.UnmarshalJSON(res, &account)
	if err != nil {
		log.Println("GetAllBalances unmarshal json error", err)
		return nil
	}
	return &account
}
