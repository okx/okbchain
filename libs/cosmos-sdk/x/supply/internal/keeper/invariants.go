package keeper

import (
	"fmt"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth/exported"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/supply/internal/types"
)

// RegisterInvariants register all supply invariants
func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	ir.RegisterRoute(types.ModuleName, "total-supply", TotalSupply(k))
}

// AllInvariants runs all invariants of the supply module.
func AllInvariants(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		return TotalSupply(k)(ctx)
	}
}

// TotalSupply checks that the total supply reflects all the coins held in accounts
func TotalSupply(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var expectedTotal sdk.Coins
		supply := k.GetSupply(ctx)
		count := 0
		k.ak.IterateAccounts(ctx, func(acc exported.Account) bool {
			if aaaa, ok := acc.(exported.ModuleAccount); ok {
				fmt.Println(aaaa.GetAddress().String(), "xxxx余额:"+aaaa.GetCoins().String(), aaaa.GetName())
			} else {
				fmt.Println(acc.GetAddress().String(), "xxxx余额:"+acc.GetCoins().String())
			}
			count++
			expectedTotal = expectedTotal.Add(acc.GetCoins()...)
			return false
		})
		fmt.Println(count, expectedTotal.String(), "\n\n")

		var supplyCoins sdk.DecCoins
		for _, coin := range supply.GetTotal() {
			if !coin.Amount.IsZero() {
				supplyCoins = supplyCoins.Add(coin)
			}
		}

		broken := !expectedTotal.IsEqual(supplyCoins)

		return sdk.FormatInvariant(types.ModuleName, "total supply",
			fmt.Sprintf(
				"\tsum of accounts coins: %v\n"+
					"\tsupply.Total:          %v\n",
				expectedTotal, supply.GetTotal())), broken
	}
}
