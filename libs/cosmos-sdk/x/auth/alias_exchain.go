package auth

import (
	"github.com/okx/brczero/libs/cosmos-sdk/x/auth/exported"
	"github.com/okx/brczero/libs/cosmos-sdk/x/auth/keeper"
)

type (
	Account       = exported.Account
	ModuleAccount = exported.ModuleAccount
	ObserverI     = keeper.ObserverI
)
