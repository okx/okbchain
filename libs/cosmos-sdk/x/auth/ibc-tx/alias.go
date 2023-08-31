package ibc_tx

import (
	"github.com/okx/brczero/libs/cosmos-sdk/x/auth/ibc-tx/internal/adapter"
	ibccodec "github.com/okx/brczero/libs/cosmos-sdk/x/auth/ibc-tx/internal/pb-codec"
)

var (
	PubKeyRegisterInterfaces = ibccodec.RegisterInterfaces
	LagacyKey2PbKey          = adapter.LagacyPubkey2ProtoBuffPubkey
)
