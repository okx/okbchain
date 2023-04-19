package keeper

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/mpt"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
)

func (k Keeper) GetStorageStore(ctx sdk.Context, acc sdk.WasmAddress) sdk.KVStore {
	account := k.accountKeeper.GetAccount(ctx, sdk.WasmToAccAddress(acc))
	ethAcc := common.BytesToAddress(acc.Bytes())

	return k.ada.NewStore(ctx.GasMeter(), ctx.KVStore(k.storageStoreKey), mpt.AddressStoragePrefixMpt(ethAcc, account.GetStateRoot()))
}
