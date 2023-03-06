package types

import (
	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
)

// RegisterCodec registers concrete types for codec
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterConcrete(MsgCreateValidator{}, "okexchain/staking/MsgCreateValidator", nil)
	cdc.RegisterConcrete(MsgEditValidator{}, "okexchain/staking/MsgEditValidator", nil)
	cdc.RegisterConcrete(MsgEditValidatorCommissionRate{}, "okexchain/staking/MsgEditValidatorCommissionRate", nil)
	cdc.RegisterConcrete(MsgDestroyValidator{}, "okexchain/staking/MsgDestroyValidator", nil)
	cdc.RegisterConcrete(MsgDeposit{}, "okexchain/staking/MsgDeposit", nil)
	cdc.RegisterConcrete(MsgWithdraw{}, "okexchain/staking/MsgWithdraw", nil)
	cdc.RegisterConcrete(MsgAddShares{}, "okexchain/staking/MsgAddShares", nil)
	cdc.RegisterConcrete(MsgRegProxy{}, "okexchain/staking/MsgRegProxy", nil)
	cdc.RegisterConcrete(MsgBindProxy{}, "okexchain/staking/MsgBindProxy", nil)
	cdc.RegisterConcrete(MsgUnbindProxy{}, "okexchain/staking/MsgUnbindProxy", nil)
	cdc.RegisterConcrete(ProposeValidatorProposal{}, ProposeValidatorProposalName, nil)
	cdc.RegisterConcrete(MsgDepositMinSelfDelegation{}, "okexchain/staking/MsgDepositMinSelfDelegation", nil)
}

// ModuleCdc is generic sealed codec to be used throughout this module
var ModuleCdc *codec.Codec

func init() {
	ModuleCdc = codec.New()
	RegisterCodec(ModuleCdc)
	codec.RegisterCrypto(ModuleCdc)
	ModuleCdc.Seal()
}
