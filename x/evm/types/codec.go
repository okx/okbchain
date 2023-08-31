package types

import (
	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	"github.com/okx/brczero/libs/system"
	"github.com/tendermint/go-amino"
)

// ModuleCdc defines the evm module's codec
var ModuleCdc = codec.New()

const (
	MsgEthereumTxName = "ethermint/MsgEthereumTx"
	ChainConfigName   = "ethermint/ChainConfig"
	TxDataName        = "ethermint/TxData"

	ManageContractDeploymentWhitelistProposalName = system.Chain+"/evm/ManageContractDeploymentWhitelistProposal"
	ManageContractBlockedListProposalName         = system.Chain+"/evm/ManageContractBlockedListProposal"
)

// RegisterCodec registers all the necessary types and interfaces for the
// evm module
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterConcrete(&MsgEthereumTx{}, MsgEthereumTxName, nil)
	cdc.RegisterConcrete(TxData{}, TxDataName, nil)
	cdc.RegisterConcrete(ChainConfig{}, ChainConfigName, nil)
	cdc.RegisterConcrete(ManageContractDeploymentWhitelistProposal{}, ManageContractDeploymentWhitelistProposalName, nil)
	cdc.RegisterConcrete(ManageContractBlockedListProposal{}, ManageContractBlockedListProposalName, nil)
	cdc.RegisterConcrete(ManageContractMethodBlockedListProposal{}, system.Chain+"/evm/ManageContractMethodBlockedListProposal", nil)
	cdc.RegisterConcrete(ManageSysContractAddressProposal{}, system.Chain+"/evm/ManageSysContractAddressProposal", nil)
	cdc.RegisterConcrete(ManageContractByteCodeProposal{}, system.Chain+"/evm/ManageContractBytecode", nil)

	cdc.RegisterConcreteUnmarshaller(ChainConfigName, func(c *amino.Codec, bytes []byte) (interface{}, int, error) {
		var cc ChainConfig
		err := cc.UnmarshalFromAmino(c, bytes)
		if err != nil {
			return ChainConfig{}, 0, err
		} else {
			return cc, len(bytes), nil
		}
	})
	cdc.RegisterConcreteUnmarshaller(MsgEthereumTxName, func(c *amino.Codec, bytes []byte) (interface{}, int, error) {
		var msg MsgEthereumTx
		err := msg.UnmarshalFromAmino(c, bytes)
		if err != nil {
			return nil, 0, err
		}
		return &msg, len(bytes), nil
	})
	cdc.RegisterConcreteUnmarshaller(TxDataName, func(c *amino.Codec, bytes []byte) (interface{}, int, error) {
		var tx TxData
		err := tx.UnmarshalFromAmino(c, bytes)
		if err != nil {
			return nil, 0, err
		}
		return tx, len(bytes), nil
	})
}

func init() {
	RegisterCodec(ModuleCdc)
	codec.RegisterCrypto(ModuleCdc)
	ModuleCdc.Seal()
}
