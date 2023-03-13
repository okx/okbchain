package ut

import (
	ethcmm "github.com/ethereum/go-ethereum/common"
	"testing"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/tendermint/libs/cli/flags"
	"github.com/okx/okbchain/x/gov"
	"github.com/okx/okbchain/x/gov/types"
	"github.com/okx/okbchain/x/staking"
	"github.com/stretchr/testify/require"
)

func TestNewHandler(t *testing.T) {
	ctx, _, gk, _, _ := CreateTestInput(t, false, 1000)
	govHandler := gov.NewHandler(gk)

	_, err := govHandler(ctx, sdk.NewTestMsg())
	require.NotNil(t, err)
}

func TestHandleMsgDeposit(t *testing.T) {
	ctx, _, gk, _, _ := CreateTestInput(t, false, 1000)
	govHandler := gov.NewHandler(gk)

	initialDeposit := sdk.SysCoins{sdk.NewInt64DecCoin(sdk.DefaultBondDenom, 50)}
	content := types.NewTextProposal("Test", "description")
	newProposalMsg := gov.NewMsgSubmitProposal(content, initialDeposit, Addrs[0])
	res, err := govHandler(ctx, newProposalMsg)
	require.Nil(t, err)
	var proposalID uint64
	gk.Cdc().MustUnmarshalBinaryLengthPrefixed(res.Data, &proposalID)

	newDepositMsg := gov.NewMsgDeposit(Addrs[0], proposalID,
		sdk.SysCoins{sdk.NewInt64DecCoin(sdk.DefaultBondDenom, 100)})
	res, err = govHandler(ctx, newDepositMsg)
	require.Nil(t, err)

	// nil address deposit on proposal
	newDepositMsg = gov.NewMsgDeposit(ethcmm.Address{}.Bytes(), proposalID,
		sdk.SysCoins{sdk.NewInt64DecCoin(sdk.DefaultBondDenom, 1000)})
	res, err = govHandler(ctx, newDepositMsg)
	require.NotNil(t, err)

	// deposit on proposal whose proposal id is 0
	newDepositMsg = gov.NewMsgDeposit(Addrs[0], 0,
		sdk.SysCoins{sdk.NewInt64DecCoin(sdk.DefaultBondDenom, 1000)})
	res, err = govHandler(ctx, newDepositMsg)
	require.NotNil(t, err)
}

func TestHandleMsgVote(t *testing.T) {
	ctx, _, gk, _, _ := CreateTestInput(t, false, 1000)
	govHandler := gov.NewHandler(gk)

	proposalCoins := sdk.SysCoins{sdk.NewInt64DecCoin(sdk.DefaultBondDenom, 500)}
	content := types.NewTextProposal("Test", "description")
	newProposalMsg := gov.NewMsgSubmitProposal(content, proposalCoins, Addrs[0])
	res, err := govHandler(ctx, newProposalMsg)
	require.Nil(t, err)
	var proposalID uint64
	gk.Cdc().MustUnmarshalBinaryLengthPrefixed(res.Data, &proposalID)

	newVoteMsg := gov.NewMsgVote(Addrs[4], proposalID, types.OptionYes)
	res, err = govHandler(ctx, newVoteMsg)
	require.Nil(t, err)

	newVoteMsg = gov.NewMsgVote(Addrs[4], 0, types.OptionYes)
	res, err = govHandler(ctx, newVoteMsg)
	require.NotNil(t, err)

	newVoteMsg = gov.NewMsgVote(sdk.AccAddress{}, proposalID, types.OptionYes)
	res, err = govHandler(ctx, newVoteMsg)
	require.NotNil(t, err)
}

func TestHandleMsgVote2(t *testing.T) {
	ctx, _, gk, sk, _ := CreateTestInput(t, false, 100000)
	govHandler := gov.NewHandler(gk)

	proposalCoins := sdk.SysCoins{sdk.NewInt64DecCoin(sdk.DefaultBondDenom, 500)}
	content := types.NewTextProposal("Test", "description")
	newProposalMsg := gov.NewMsgSubmitProposal(content, proposalCoins, Addrs[0])
	res, err := govHandler(ctx, newProposalMsg)
	require.Nil(t, err)
	var proposalID uint64
	gk.Cdc().MustUnmarshalBinaryLengthPrefixed(res.Data, &proposalID)

	ctx.SetBlockHeight(int64(sk.GetEpoch(ctx)))
	skHandler := staking.NewHandler(sk)
	valAddrs := make([]sdk.ValAddress, len(Addrs[:2]))
	for i, addr := range Addrs[:2] {
		valAddrs[i] = sdk.ValAddress(addr)
	}
	CreateValidators(t, skHandler, ctx, valAddrs, []int64{10, 10})
	staking.EndBlocker(ctx, sk)

	newVoteMsg := gov.NewMsgVote(Addrs[0], proposalID, types.OptionYes)
	res, err = govHandler(ctx, newVoteMsg)
	require.Nil(t, err)

	newVoteMsg = gov.NewMsgVote(Addrs[1], proposalID, types.OptionYes)
	res, err = govHandler(ctx, newVoteMsg)
	require.Nil(t, err)
}

// test distribute deposits after voting
func TestHandleMsgVote3(t *testing.T) {
	ctx, _, gk, sk, _ := CreateTestInput(t, false, 100000)
	govHandler := gov.NewHandler(gk)

	proposalCoins := sdk.SysCoins{sdk.NewInt64DecCoin(sdk.DefaultBondDenom, 500)}
	content := types.NewTextProposal("Test", "description")
	newProposalMsg := gov.NewMsgSubmitProposal(content, proposalCoins, Addrs[0])
	res, err := govHandler(ctx, newProposalMsg)
	require.Nil(t, err)
	var proposalID uint64
	gk.Cdc().MustUnmarshalBinaryLengthPrefixed(res.Data, &proposalID)

	ctx.SetBlockHeight(int64(sk.GetEpoch(ctx)))
	skHandler := staking.NewHandler(sk)
	valAddrs := make([]sdk.ValAddress, len(Addrs[:2]))
	for i, addr := range Addrs[:2] {
		valAddrs[i] = sdk.ValAddress(addr)
	}
	CreateValidators(t, skHandler, ctx, valAddrs, []int64{10, 10})
	staking.EndBlocker(ctx, sk)

	require.Equal(t, proposalCoins, gk.SupplyKeeper().
		GetModuleAccount(ctx, types.ModuleName).GetCoins())
	newVoteMsg := gov.NewMsgVote(Addrs[0], proposalID, types.OptionNoWithVeto)
	res, err = govHandler(ctx, newVoteMsg)
	require.Nil(t, err)
	require.Equal(t, sdk.Coins(nil), gk.SupplyKeeper().GetModuleAccount(ctx, types.ModuleName).GetCoins())
}

func TestHandleMsgSubmitProposal(t *testing.T) {
	ctx, _, gk, _, _ := CreateTestInput(t, false, 1000)
	log, err := flags.ParseLogLevel("*:error", ctx.Logger(), "error")
	require.Nil(t, err)
	ctx.SetLogger(log)
	handler := gov.NewHandler(gk)

	proposalCoins := sdk.SysCoins{sdk.NewInt64DecCoin("xxx", 500)}
	content := types.NewTextProposal("Test", "description")
	newProposalMsg := gov.NewMsgSubmitProposal(content, proposalCoins, Addrs[0])
	_, err = handler(ctx, newProposalMsg)
	require.NotNil(t, err)

	proposalCoins = sdk.SysCoins{sdk.NewInt64DecCoin(sdk.DefaultBondDenom, 500)}
	content = types.NewTextProposal("Test", "description")
	newProposalMsg = gov.NewMsgSubmitProposal(content, proposalCoins, ethcmm.Address{}.Bytes())
	_, err = handler(ctx, newProposalMsg)
	require.NotNil(t, err)

	//content = tokenTypes.NewDexListProposal("Test", "", keeper.Addrs[0],
	//	"btc-123", common.NativeToken, sdk.NewDec(1000), 0,
	//	4, 4, sdk.NewDec(1))
	//newProposalMsg = NewMsgSubmitProposal(content, proposalCoins, keeper.Addrs[0])
	//res = handler(ctx, newProposalMsg)
	//require.NotNil(t, err)
}
