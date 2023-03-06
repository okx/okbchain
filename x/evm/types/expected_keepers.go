package types

import (
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth"
	authexported "github.com/okx/okbchain/libs/cosmos-sdk/x/auth/exported"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/params"
)

// AccountKeeper defines the expected account keeper interface
type AccountKeeper interface {
	NewAccountWithAddress(ctx sdk.Context, addr sdk.AccAddress) authexported.Account
	GetAllAccounts(ctx sdk.Context) (accounts []authexported.Account)
	IterateAccounts(ctx sdk.Context, cb func(account authexported.Account) bool)
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) authexported.Account
	SetAccount(ctx sdk.Context, account authexported.Account)
	RemoveAccount(ctx sdk.Context, account authexported.Account)
	SetObserverKeeper(observer auth.ObserverI)
}

type SupplyKeeper interface {
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

type Subspace interface {
	GetParamSet(ctx sdk.Context, ps params.ParamSet)
	SetParamSet(ctx sdk.Context, ps params.ParamSet)
	CustomKVStore(ctx sdk.Context) sdk.KVStore
}

type BankKeeper interface {
	BlacklistedAddr(addr sdk.AccAddress) bool
}

// StakingKeeper for validator verify
type StakingKeeper interface {
	IsValidator(ctx sdk.Context, addr sdk.AccAddress) bool
}
