package exported

import (
	storetypes "github.com/okx/brczero/libs/cosmos-sdk/store/types"
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
)

type SizerAccountKeeper interface {
	GetEncodedAccountSize(acc Account) int
}

type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) Account
}

func TryAddGetAccountGas(gasMeter sdk.GasMeter, ak SizerAccountKeeper, acc Account) (bool, sdk.Gas) {
	if ak == nil || gasMeter == nil || acc == nil {
		return false, 0
	}
	size := ak.GetEncodedAccountSize(acc)
	if size == 0 {
		return false, 0
	}
	kvGasConfig := storetypes.KVGasConfig()
	gas := kvGasConfig.ReadCostFlat + storetypes.Gas(size)*kvGasConfig.ReadCostPerByte
	gasMeter.ConsumeGas(gas, "x/bank/internal/keeper/keeper.BaseSendKeeper")
	return true, gas
}

func GetAccountGas(ak SizerAccountKeeper, acc Account) (sdk.Gas, bool) {
	if acc == nil || ak == nil {
		return 0, false
	}
	size := ak.GetEncodedAccountSize(acc)
	if size == 0 {
		return 0, false
	}
	kvGasConfig := storetypes.KVGasConfig()
	gas := kvGasConfig.ReadCostFlat + storetypes.Gas(size)*kvGasConfig.ReadCostPerByte
	return gas, true
}

func GetAccountAndGas(ctx *sdk.Context, keeper AccountKeeper, addr sdk.AccAddress) (Account, sdk.Gas) {
	gasMeter := ctx.GasMeter()
	tmpGasMeter := sdk.GetReusableInfiniteGasMeter()
	ctx.SetGasMeter(tmpGasMeter)
	defer func() {
		ctx.SetGasMeter(gasMeter)
		sdk.ReturnInfiniteGasMeter(tmpGasMeter)
	}()

	acc := keeper.GetAccount(*ctx, addr)

	gasUsed := tmpGasMeter.GasConsumed()
	gasMeter.ConsumeGas(gasUsed, "get account")

	return acc, gasUsed
}
