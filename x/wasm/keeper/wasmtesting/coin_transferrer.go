package wasmtesting

import sdk "github.com/okx/brczero/libs/cosmos-sdk/types"

type MockCoinTransferrer struct {
	TransferCoinsFn func(ctx sdk.Context, fromAddr sdk.WasmAddress, toAddr sdk.WasmAddress, amt sdk.Coins) error
}

func (m *MockCoinTransferrer) TransferCoins(ctx sdk.Context, fromAddr sdk.WasmAddress, toAddr sdk.WasmAddress, amt sdk.Coins) error {
	if m.TransferCoinsFn == nil {
		panic("not expected to be called")
	}
	return m.TransferCoinsFn(ctx, fromAddr, toAddr, amt)
}
