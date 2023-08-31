package ut

import (
	"testing"
	"time"

	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	"github.com/okx/brczero/x/gov"
	"github.com/okx/brczero/x/gov/types"
	"github.com/okx/brczero/x/params"
	paramsTypes "github.com/okx/brczero/x/params/types"
	"github.com/okx/brczero/x/staking"
	"github.com/stretchr/testify/require"
)

func newTextProposal(t *testing.T, ctx sdk.Context, initialDeposit sdk.SysCoins, govHandler sdk.Handler) *sdk.Result {
	content := types.NewTextProposal("Test", "description")
	newProposalMsg := gov.NewMsgSubmitProposal(content, initialDeposit, Addrs[0])
	res, err := govHandler(ctx, newProposalMsg)
	require.Nil(t, err)
	return res
}

func TestTickPassedVotingPeriod(t *testing.T) {
	ctx, _, gk, _, _ := CreateTestInput(t, false, 1000)
	govHandler := gov.NewHandler(gk)

	inactiveQueue := gk.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, inactiveQueue.Valid())
	inactiveQueue.Close()
	activeQueue := gk.ActiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, activeQueue.Valid())
	activeQueue.Close()

	proposalCoins := sdk.SysCoins{sdk.NewInt64DecCoin(sdk.DefaultBondDenom, 500)}
	content := types.NewTextProposal("Test", "description")
	newProposalMsg := gov.NewMsgSubmitProposal(content, proposalCoins, Addrs[0])
	res, err := govHandler(ctx, newProposalMsg)
	require.Nil(t, err)
	var proposalID uint64
	gk.Cdc().MustUnmarshalBinaryLengthPrefixed(res.Data, &proposalID)

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Duration(1) * time.Second)
	ctx.SetBlockHeader(newHeader)

	newDepositMsg := gov.NewMsgDeposit(Addrs[1], proposalID, proposalCoins)
	res, err = govHandler(ctx, newDepositMsg)
	require.NotNil(t, err)

	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(gk.GetDepositParams(ctx).MaxDepositPeriod).
		Add(gk.GetVotingParams(ctx).VotingPeriod)
	ctx.SetBlockHeader(newHeader)

	inactiveQueue = gk.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, inactiveQueue.Valid())
	inactiveQueue.Close()

	activeQueue = gk.ActiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.True(t, activeQueue.Valid())
	var activeProposalID uint64
	err = gk.Cdc().UnmarshalBinaryLengthPrefixed(activeQueue.Value(), &activeProposalID)
	require.Nil(t, err)
	proposal, ok := gk.GetProposal(ctx, activeProposalID)
	require.True(t, ok)
	require.Equal(t, gov.StatusVotingPeriod, proposal.Status)
	depositsIterator := gk.GetDeposits(ctx, proposalID)
	require.NotEqual(t, depositsIterator, []gov.Deposit{})
	activeQueue.Close()

	gov.EndBlocker(ctx, gk)

	activeQueue = gk.ActiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, activeQueue.Valid())
	activeQueue.Close()
}

// test deposit is not enough when expire max deposit period
func TestEndBlockerIterateInactiveProposalsQueue(t *testing.T) {
	ctx, _, gk, _, _ := CreateTestInput(t, false, 1000)

	initialDeposit := sdk.SysCoins{sdk.NewInt64DecCoin(sdk.DefaultBondDenom, 10)}
	newTextProposal(t, ctx, initialDeposit, gov.NewHandler(gk))

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(gk.GetMaxDepositPeriod(ctx, nil))
	ctx.SetBlockHeader(newHeader)
	inactiveQueue := gk.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.True(t, inactiveQueue.Valid())
	inactiveQueue.Close()
	gov.EndBlocker(ctx, gk)
	inactiveQueue = gk.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, inactiveQueue.Valid())
	inactiveQueue.Close()
}

func TestEndBlockerIterateActiveProposalsQueue1(t *testing.T) {
	ctx, _, gk, _, _ := CreateTestInput(t, false, 1000)

	initialDeposit := sdk.SysCoins{sdk.NewInt64DecCoin(sdk.DefaultBondDenom, 150)}
	newTextProposal(t, ctx, initialDeposit, gov.NewHandler(gk))

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(gk.GetVotingPeriod(ctx, nil))
	ctx.SetBlockHeader(newHeader)
	activeQueue := gk.ActiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.True(t, activeQueue.Valid())
	activeQueue.Close()
	gov.EndBlocker(ctx, gk)
	activeQueue = gk.ActiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, activeQueue.Valid())
	activeQueue.Close()
}

// test distribute
func TestEndBlockerIterateActiveProposalsQueue2(t *testing.T) {
	ctx, _, gk, sk, _ := CreateTestInput(t, false, 100000)
	govHandler := gov.NewHandler(gk)

	ctx.SetBlockHeight(int64(sk.GetEpoch(ctx)))
	skHandler := staking.NewHandler(sk)
	valAddrs := make([]sdk.ValAddress, len(Addrs[:3]))
	for i, addr := range Addrs[:3] {
		valAddrs[i] = sdk.ValAddress(addr)
	}
	CreateValidators(t, skHandler, ctx, valAddrs, []int64{10, 10, 10})
	staking.EndBlocker(ctx, sk)

	initialDeposit := sdk.SysCoins{sdk.NewInt64DecCoin(sdk.DefaultBondDenom, 150)}
	res := newTextProposal(t, ctx, initialDeposit, gov.NewHandler(gk))

	var proposalID uint64
	gk.Cdc().MustUnmarshalBinaryLengthPrefixed(res.Data, &proposalID)

	require.Equal(t, initialDeposit, gk.SupplyKeeper().
		GetModuleAccount(ctx, types.ModuleName).GetCoins())
	newVoteMsg := gov.NewMsgVote(Addrs[0], proposalID, types.OptionNoWithVeto)
	res, err := govHandler(ctx, newVoteMsg)
	require.Nil(t, err)

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(gk.GetVotingPeriod(ctx, nil))
	ctx.SetBlockHeader(newHeader)
	activeQueue := gk.ActiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.True(t, activeQueue.Valid())
	activeQueue.Close()
	gov.EndBlocker(ctx, gk)
	activeQueue = gk.ActiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, activeQueue.Valid())
	activeQueue.Close()

	require.Equal(t, sdk.Coins(nil), gk.SupplyKeeper().GetModuleAccount(ctx, types.ModuleName).GetCoins())
}

// test passed
func TestEndBlockerIterateActiveProposalsQueue3(t *testing.T) {
	ctx, _, gk, sk, _ := CreateTestInput(t, false, 100000)
	govHandler := gov.NewHandler(gk)

	ctx.SetBlockHeight(int64(sk.GetEpoch(ctx)))
	skHandler := staking.NewHandler(sk)
	valAddrs := make([]sdk.ValAddress, len(Addrs[:4]))
	for i, addr := range Addrs[:4] {
		valAddrs[i] = sdk.ValAddress(addr)
	}
	CreateValidators(t, skHandler, ctx, valAddrs, []int64{10, 10, 10, 10})
	staking.EndBlocker(ctx, sk)

	initialDeposit := sdk.SysCoins{sdk.NewInt64DecCoin(sdk.DefaultBondDenom, 150)}
	res := newTextProposal(t, ctx, initialDeposit, gov.NewHandler(gk))
	var proposalID uint64
	gk.Cdc().MustUnmarshalBinaryLengthPrefixed(res.Data, &proposalID)

	require.Equal(t, initialDeposit, gk.SupplyKeeper().
		GetModuleAccount(ctx, types.ModuleName).GetCoins())
	newVoteMsg := gov.NewMsgVote(Addrs[0], proposalID, types.OptionYes)
	res, err := govHandler(ctx, newVoteMsg)
	require.Nil(t, err)
	newVoteMsg = gov.NewMsgVote(Addrs[1], proposalID, types.OptionYes)
	res, err = govHandler(ctx, newVoteMsg)
	require.Nil(t, err)

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(gk.GetVotingPeriod(ctx, nil))
	ctx.SetBlockHeader(newHeader)
	activeQueue := gk.ActiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.True(t, activeQueue.Valid())
	activeQueue.Close()
	gov.EndBlocker(ctx, gk)
	activeQueue = gk.ActiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, activeQueue.Valid())
	activeQueue.Close()

	require.Equal(t, sdk.Coins(nil), gk.SupplyKeeper().GetModuleAccount(ctx, types.ModuleName).GetCoins())
}

func TestEndBlockerIterateWaitingProposalsQueue(t *testing.T) {
	ctx, _, gk, sk, _ := CreateTestInput(t, false, 100000)
	govHandler := gov.NewHandler(gk)

	ctx.SetBlockHeight(int64(sk.GetEpoch(ctx)))
	skHandler := staking.NewHandler(sk)
	valAddrs := make([]sdk.ValAddress, len(Addrs[:4]))
	for i, addr := range Addrs[:4] {
		valAddrs[i] = sdk.ValAddress(addr)
	}
	CreateValidators(t, skHandler, ctx, valAddrs, []int64{10, 10, 10, 10})
	staking.EndBlocker(ctx, sk)

	initialDeposit := sdk.SysCoins{sdk.NewInt64DecCoin(sdk.DefaultBondDenom, 150)}
	paramsChanges := []params.ParamChange{{Subspace: "staking", Key: "MaxValidators", Value: "105"}}
	height := uint64(ctx.BlockHeight() + 1000)
	content := paramsTypes.NewParameterChangeProposal("Test", "", paramsChanges, height)
	newProposalMsg := gov.NewMsgSubmitProposal(content, initialDeposit, Addrs[0])
	res, err := govHandler(ctx, newProposalMsg)
	require.Nil(t, err)
	var proposalID uint64
	gk.Cdc().MustUnmarshalBinaryLengthPrefixed(res.Data, &proposalID)

	newVoteMsg := gov.NewMsgVote(Addrs[0], proposalID, types.OptionYes)
	res, err = govHandler(ctx, newVoteMsg)
	require.Nil(t, err)
	newVoteMsg = gov.NewMsgVote(Addrs[1], proposalID, types.OptionYes)
	res, err = govHandler(ctx, newVoteMsg)
	require.Nil(t, err)
	newVoteMsg = gov.NewMsgVote(Addrs[2], proposalID, types.OptionYes)
	res, err = govHandler(ctx, newVoteMsg)
	require.Nil(t, err)

	ctx.SetBlockHeight(int64(height))
	waitingQueue := gk.WaitingProposalQueueIterator(ctx, uint64(ctx.BlockHeight()))
	require.True(t, waitingQueue.Valid())
	waitingQueue.Close()
	gov.EndBlocker(ctx, gk)
	waitingQueue = gk.ActiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, waitingQueue.Valid())
	waitingQueue.Close()
}
