package simapp

import (
	"github.com/okx/okbchain/libs/cosmos-sdk/codec/types"
	ibc_tx "github.com/okx/okbchain/libs/cosmos-sdk/x/auth/ibc-tx"
	simappparams "github.com/okx/okbchain/libs/ibc-go/testing/simapp/params"
)

// MakeTestEncodingConfig creates an EncodingConfig for testing. This function
// should be used only in tests or when creating a new app instance (NewApp*()).
// App user shouldn't create new codecs - use the app.AppCodec instead.
// [DEPRECATED]
func MakeTestEncodingConfig() simappparams.EncodingConfig {
	encodingConfig := simappparams.MakeTestEncodingConfig()
	//std.RegisterLegacyAminoCodec(encodingConfig.Amino)
	ibc_tx.PubKeyRegisterInterfaces(encodingConfig.InterfaceRegistry)
	//ModuleBasics.RegisterLegacyAminoCodec(encodingConfig.Amino)
	interfaceReg := types.NewInterfaceRegistry()
	ModuleBasics.RegisterInterfaces(interfaceReg)
	return encodingConfig
}
