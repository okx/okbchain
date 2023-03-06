package types

import (
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	authtypes "github.com/okx/okbchain/libs/cosmos-sdk/x/auth"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth/exported"
)

// AccountKeeper defines the expected account keeper (noalias)
type AccountKeeper interface {
	IterateAccounts(ctx sdk.Context, process func(exported.Account) (stop bool))
	GetAccount(sdk.Context, sdk.AccAddress) exported.Account
	SetAccount(sdk.Context, exported.Account)
	NewAccount(sdk.Context, exported.Account) exported.Account
}

// BankKeeper defines the expected bank keeper (noalias)
type BankKeeper interface {
	CM40BankKeeper
	SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	DelegateCoins(ctx sdk.Context, fromAdd, toAddr sdk.AccAddress, amt sdk.Coins) error
	UndelegateCoins(ctx sdk.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error

	SubtractCoins(ctx sdk.Context, addr sdk.AccAddress, amt sdk.Coins) (sdk.Coins, error)
	AddCoins(ctx sdk.Context, addr sdk.AccAddress, amt sdk.Coins) (sdk.Coins, error)

	BlacklistedAddr(addr sdk.AccAddress) bool
}

type CM40BankKeeper interface {
	HasBalance(ctx sdk.Context, addr sdk.AccAddress, amt sdk.Coin) bool
	BlockedAddr(address sdk.AccAddress) bool
}

type CM40AccountKeeper interface {
	NewAccount(ctx sdk.Context, acc authtypes.Account) authtypes.Account
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) authtypes.Account
	SetAccount(ctx sdk.Context, acc authtypes.Account)
}
