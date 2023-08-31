package subspace

import (
	"github.com/okx/brczero/libs/cosmos-sdk/x/params/subspace"
)

type (
	ParamSetPairs    = subspace.ParamSetPairs
	KeyTable         = subspace.KeyTable
	ValueValidatorFn = subspace.ValueValidatorFn
)

var (
	NewKeyTable     = subspace.NewKeyTable
	NewParamSetPair = subspace.NewParamSetPair

	StoreKey  = subspace.StoreKey
	TStoreKey = subspace.TStoreKey
)
