package typesadapter

import (
	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	interfacetypes "github.com/okx/brczero/libs/cosmos-sdk/codec/types"
	"github.com/okx/brczero/libs/cosmos-sdk/types"
	txmsg "github.com/okx/brczero/libs/cosmos-sdk/types/ibc-adapter"
	"github.com/okx/brczero/libs/cosmos-sdk/types/msgservice"
)

var (
	cdc *codec.Codec
)

func init() {
	cdc = codec.New()
	cdc.RegisterConcrete(MsgSend{}, "cosmos-sdk/MsgSend", nil)
	cdc.RegisterConcrete(MsgMultiSend{}, "cosmos-sdk/MultiSend", nil)
}

func RegisterInterface(registry interfacetypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*txmsg.Msg)(nil),
		&MsgSend{},
		&MsgMultiSend{},
	)
	registry.RegisterImplementations(
		(*types.MsgProtoAdapter)(nil),
		&MsgSend{},
		&MsgMultiSend{},
	)
	registry.RegisterImplementations(
		(*types.Msg)(nil),
		&MsgSend{},
		&MsgMultiSend{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
