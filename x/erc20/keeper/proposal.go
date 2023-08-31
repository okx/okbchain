package keeper

import (
	"fmt"
	"time"

	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	"github.com/okx/brczero/x/erc20/types"
	sdkGov "github.com/okx/brczero/x/gov"
	govKeeper "github.com/okx/brczero/x/gov/keeper"
	govTypes "github.com/okx/brczero/x/gov/types"
)

var _ govKeeper.ProposalHandler = (*Keeper)(nil)

// GetMinDeposit returns min deposit
func (k Keeper) GetMinDeposit(ctx sdk.Context, content sdkGov.Content) (minDeposit sdk.SysCoins) {
	switch content.(type) {
	case types.TokenMappingProposal, types.ProxyContractRedirectProposal, types.ContractTemplateProposal:
		minDeposit = k.govKeeper.GetDepositParams(ctx).MinDeposit
	}

	return
}

// GetMaxDepositPeriod returns max deposit period
func (k Keeper) GetMaxDepositPeriod(ctx sdk.Context, content sdkGov.Content) (maxDepositPeriod time.Duration) {
	switch content.(type) {
	case types.TokenMappingProposal, types.ProxyContractRedirectProposal, types.ContractTemplateProposal:
		maxDepositPeriod = k.govKeeper.GetDepositParams(ctx).MaxDepositPeriod
	}

	return
}

// GetVotingPeriod returns voting period
func (k Keeper) GetVotingPeriod(ctx sdk.Context, content sdkGov.Content) (votingPeriod time.Duration) {
	switch content.(type) {
	case types.TokenMappingProposal, types.ProxyContractRedirectProposal, types.ContractTemplateProposal:
		votingPeriod = k.govKeeper.GetVotingParams(ctx).VotingPeriod
	}

	return
}

// CheckMsgSubmitProposal validates MsgSubmitProposal
func (k Keeper) CheckMsgSubmitProposal(ctx sdk.Context, msg govTypes.MsgSubmitProposal) sdk.Error {
	switch content := msg.Content.(type) {
	case types.TokenMappingProposal, types.ProxyContractRedirectProposal, types.ContractTemplateProposal:
		// whole target address list will be added/deleted to/from the contract deployment whitelist/contract blocked list.
		// It's not necessary to check the existence in CheckMsgSubmitProposal
		return nil
	default:
		return sdk.ErrUnknownRequest(fmt.Sprintf("unrecognized %s proposal content type: %T", types.DefaultCodespace, content))
	}
}

// nolint
func (k Keeper) AfterSubmitProposalHandler(_ sdk.Context, _ govTypes.Proposal) {}
func (k Keeper) AfterDepositPeriodPassed(_ sdk.Context, _ govTypes.Proposal)   {}
func (k Keeper) RejectedHandler(_ sdk.Context, _ govTypes.Content)             {}
func (k Keeper) VoteHandler(_ sdk.Context, _ govTypes.Proposal, _ govTypes.Vote) (string, sdk.Error) {
	return "", nil
}
