package types

import (
	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
)

// RegisterCodec registers concrete types on codec codec
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterConcrete(MsgWithdrawValidatorCommission{}, "okexchain/distribution/MsgWithdrawReward", nil)
	cdc.RegisterConcrete(MsgWithdrawDelegatorReward{}, "okexchain/distribution/MsgWithdrawDelegatorReward", nil)
	cdc.RegisterConcrete(MsgSetWithdrawAddress{}, "okexchain/distribution/MsgModifyWithdrawAddress", nil)
	cdc.RegisterConcrete(CommunityPoolSpendProposal{}, "okexchain/distribution/CommunityPoolSpendProposal", nil)
	cdc.RegisterConcrete(ChangeDistributionTypeProposal{}, "okexchain/distribution/ChangeDistributionTypeProposal", nil)
	cdc.RegisterConcrete(WithdrawRewardEnabledProposal{}, "okexchain/distribution/WithdrawRewardEnabledProposal", nil)
	cdc.RegisterConcrete(RewardTruncatePrecisionProposal{}, "okexchain/distribution/RewardTruncatePrecisionProposal", nil)
	cdc.RegisterConcrete(MsgWithdrawDelegatorAllRewards{}, "okexchain/distribution/MsgWithdrawDelegatorAllRewards", nil)
}

// ModuleCdc generic sealed codec to be used throughout module
var ModuleCdc *codec.Codec

func init() {
	ModuleCdc = codec.New()
	RegisterCodec(ModuleCdc)
	codec.RegisterCrypto(ModuleCdc)
	ModuleCdc.Seal()
}
