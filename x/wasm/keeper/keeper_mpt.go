package keeper

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	apptypes "github.com/okx/okbchain/app/types"
	clientcontext "github.com/okx/okbchain/libs/cosmos-sdk/client/context"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/mpt"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/prefix"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth"
	"github.com/okx/okbchain/x/wasm/watcher"
	"log"
)

func (k Keeper) getStorageStore(ctx sdk.Context, acc sdk.WasmAddress) sdk.KVStore {
	account := k.accountKeeper.GetAccount(ctx, sdk.WasmToAccAddress(acc))
	if account == nil && watcher.Enable() {
		account = getAccount(acc)
	}
	ethAcc := common.BytesToAddress(acc.Bytes())

	// in case of query panic
	var stateRoot common.Hash
	if account == nil {
		stateRoot = ethtypes.EmptyRootHash
	} else {
		stateRoot = account.GetStateRoot()
	}
	return k.ada.NewStore(ctx.GasMeter(), ctx.KVStore(k.storageStoreKey), mpt.AddressStoragePrefixMpt(ethAcc, stateRoot))
}

func (k Keeper) GetStorageStore4Query(ctx sdk.Context, acc sdk.WasmAddress) sdk.KVStore {
	if watcher.Enable() {
		ethAcc := common.BytesToAddress(acc.Bytes())
		store := k.ada.NewStore(ctx.GasMeter(), ctx.KVStore(k.storageStoreKey), nil)

		return prefix.NewStore(store, mpt.AddressStorageWithoutStorageRootPrefixMpt(ethAcc))
	}

	return k.getStorageStore(ctx, acc)
}

var clientCtx clientcontext.CLIContext

func SetCliContext(ctx clientcontext.CLIContext) {
	clientCtx = ctx
}

func getAccount(addr sdk.WasmAddress) *apptypes.EthAccount {
	bs, err := clientCtx.Codec.MarshalJSON(auth.NewQueryAccountParams(addr.Bytes()))
	if err != nil {
		log.Println("getAccount marshal json error", err)
		return nil
	}
	res, _, err := clientCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", auth.QuerierRoute, auth.QueryAccount), bs)
	if err != nil {
		log.Println("getAccount query with data error", err)
		return nil
	}
	var account apptypes.EthAccount
	err = clientCtx.Codec.UnmarshalJSON(res, &account)
	if err != nil {
		log.Println("getAccount unmarshal json error", err)
		return nil
	}
	return &account
}
