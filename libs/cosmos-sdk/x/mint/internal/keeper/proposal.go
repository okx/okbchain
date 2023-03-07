package keeper

import (
	"fmt"
	"time"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/mint/internal/types"

	sdkGov "github.com/okx/okbchain/x/gov"
	govKeeper "github.com/okx/okbchain/x/gov/keeper"
	govTypes "github.com/okx/okbchain/x/gov/types"
)

var _ govKeeper.ProposalHandler = (*Keeper)(nil)

// GetMinDeposit returns min deposit
func (k Keeper) GetMinDeposit(ctx sdk.Context, content sdkGov.Content) (minDeposit sdk.SysCoins) {
	switch content.(type) {
	case types.ManageTreasuresProposal:
		minDeposit = k.govKeeper.GetDepositParams(ctx).MinDeposit
	}

	return
}

// GetMaxDepositPeriod returns max deposit period
func (k Keeper) GetMaxDepositPeriod(ctx sdk.Context, content sdkGov.Content) (maxDepositPeriod time.Duration) {
	switch content.(type) {
	case types.ManageTreasuresProposal:
		maxDepositPeriod = k.govKeeper.GetDepositParams(ctx).MaxDepositPeriod
	}

	return
}

// GetVotingPeriod returns voting period
func (k Keeper) GetVotingPeriod(ctx sdk.Context, content sdkGov.Content) (votingPeriod time.Duration) {
	switch content.(type) {
	case types.ManageTreasuresProposal:
		votingPeriod = k.govKeeper.GetVotingParams(ctx).VotingPeriod
	}

	return
}

// CheckMsgSubmitProposal validates MsgSubmitProposal
func (k Keeper) CheckMsgSubmitProposal(ctx sdk.Context, msg govTypes.MsgSubmitProposal) sdk.Error {
	switch content := msg.Content.(type) {
	case types.ManageTreasuresProposal:
		if !k.sk.IsValidator(ctx, msg.Proposer) {
			return types.ErrProposerMustBeValidator
		}
		treasures := k.GetTreasures(ctx)
		if content.IsAdded {
			result := types.InsertAndUpdateTreasures(treasures, content.Treasures)
			if err := types.ValidateTreasures(result); err != nil {
				return types.ErrTreasuresInternal(err)
			}
		} else {
			result, err := types.DeleteTreasures(treasures, content.Treasures)
			if err != nil {
				return types.ErrTreasuresInternal(err)
			}
			if err := types.ValidateTreasures(result); err != nil {
				return types.ErrTreasuresInternal(err)
			}
		}
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
