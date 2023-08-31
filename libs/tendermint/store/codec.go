package store

import (
	amino "github.com/tendermint/go-amino"

	"github.com/okx/brczero/libs/tendermint/types"
)

var cdc = amino.NewCodec()

func init() {
	types.RegisterBlockAmino(cdc)
}
