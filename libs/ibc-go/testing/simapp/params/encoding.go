package params

import (
	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	"github.com/okx/brczero/libs/cosmos-sdk/codec/types"
)

// EncodingConfig specifies the concrete encoding types to use for a given app.
// This is provided for compatibility between protobuf and amino implementations.
type EncodingConfig struct {
	InterfaceRegistry types.InterfaceRegistry
	Marshaler         codec.Codec
	//TxConfig          client.TxConfig
	//Amino             *codec.LegacyAmino
}

func (e EncodingConfig) CodecProxy() *codec.CodecProxy {
	return codec.NewCodecProxy(codec.NewProtoCodec(e.InterfaceRegistry), &e.Marshaler)
}
