package gov

import (
	"fmt"

	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"

	"github.com/okx/brczero/x/common"
	"github.com/okx/brczero/x/gov/keeper"
	"github.com/okx/brczero/x/gov/types"
)

// NewHandler handle all "gov" type messages.
func NewHandler(keeper Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		ctx.SetEventManager(sdk.NewEventManager())

		switch msg := msg.(type) {
		case MsgDeposit:
			return handleMsgDeposit(ctx, keeper, msg)

		case MsgSubmitProposal:
			return handleMsgSubmitProposal(ctx, keeper, msg)

		case MsgVote:
			return handleMsgVote(ctx, keeper, msg)
		default:
			errMsg := fmt.Sprintf("unrecognized gov message type: %T", msg)
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleMsgSubmitProposal(ctx sdk.Context, keeper keeper.Keeper, msg MsgSubmitProposal) (*sdk.Result, error) {
	err := hasOnlyDefaultBondDenom(msg.InitialDeposit)
	if err != nil {
		return sdk.EnvelopedErr{err}.Result()
	}

	// use ctx directly
	if !keeper.ProposalHandlerRouter().HasRoute(msg.Content.ProposalRoute()) {
		err = keeper.CheckMsgSubmitProposal(ctx, msg)
	} else {
		proposalHandler := keeper.ProposalHandlerRouter().GetRoute(msg.Content.ProposalRoute())
		err = proposalHandler.CheckMsgSubmitProposal(ctx, msg)
	}
	if err != nil {
		return sdk.EnvelopedErr{err}.Result()
	}

	proposal, err := keeper.SubmitProposal(ctx, msg.Content)
	if err != nil {
		return sdk.EnvelopedErr{err}.Result()
	}

	err = keeper.AddDeposit(ctx, proposal.ProposalID, msg.Proposer,
		msg.InitialDeposit, types.EventTypeSubmitProposal)
	if err != nil {
		return sdk.EnvelopedErr{err}.Result()
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Proposer.String()),
		),
	)

	return &sdk.Result{
		Data:   keeper.Cdc().MustMarshalBinaryLengthPrefixed(proposal.ProposalID),
		Events: ctx.EventManager().Events(),
	}, nil
}

func handleMsgDeposit(ctx sdk.Context, keeper keeper.Keeper, msg MsgDeposit) (*sdk.Result, error) {
	if err := hasOnlyDefaultBondDenom(msg.Amount); err != nil {
		return sdk.EnvelopedErr{err}.Result()
	}
	// check depositor has sufficient coins
	err := common.HasSufficientCoins(msg.Depositor, keeper.BankKeeper().GetCoins(ctx, msg.Depositor),
		msg.Amount)
	if err != nil {
		return common.ErrInsufficientCoins(DefaultParamspace, err.Error()).Result()
	}

	sdkErr := keeper.AddDeposit(ctx, msg.ProposalID, msg.Depositor,
		msg.Amount, types.EventTypeProposalDeposit)
	if sdkErr != nil {
		return sdk.EnvelopedErr{sdkErr}.Result()
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Depositor.String()),
		),
	)

	return &sdk.Result{Events: ctx.EventManager().Events()}, nil
}

func handleMsgVote(ctx sdk.Context, k keeper.Keeper, msg MsgVote) (*sdk.Result, error) {
	proposal, ok := k.GetProposal(ctx, msg.ProposalID)
	if !ok {
		return sdk.EnvelopedErr{types.ErrUnknownProposal(msg.ProposalID)}.Result()
	}

	err, _ := k.AddVote(ctx, msg.ProposalID, msg.Voter, msg.Option)
	if err != nil {
		return sdk.EnvelopedErr{err}.Result()
	}

	status, distribute, tallyResults := keeper.Tally(ctx, k, proposal, false)
	// update tally results after vote every time
	proposal.FinalTallyResult = tallyResults

	// this vote makes the votingPeriod end
	if status != StatusVotingPeriod {
		tagValue, logMsg := handleProposalAfterTally(ctx, k, &proposal, distribute, status)
		k.RemoveFromActiveProposalQueue(ctx, proposal.ProposalID, proposal.VotingEndTime)
		proposal.VotingEndTime = ctx.BlockHeader().Time
		k.DeleteVotes(ctx, proposal.ProposalID)
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeProposalVoteTally,
				sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.ProposalID)),
				sdk.NewAttribute(types.AttributeKeyProposalResult, tagValue),
				sdk.NewAttribute(types.AttributeKeyProposalLog, logMsg),
			),
		)
	}
	k.SetProposal(ctx, proposal)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Voter.String()),
			sdk.NewAttribute(types.AttributeKeyProposalStatus, proposal.Status.String()),
		),
	)

	return &sdk.Result{Events: ctx.EventManager().Events()}, nil
}

func handleProposalAfterTally(
	ctx sdk.Context, k keeper.Keeper, proposal *types.Proposal, distribute bool, status ProposalStatus,
) (string, string) {
	if distribute {
		k.DistributeDeposits(ctx, proposal.ProposalID)
	} else {
		k.RefundDeposits(ctx, proposal.ProposalID)
	}

	if status == StatusPassed {
		handler := k.Router().GetRoute(proposal.ProposalRoute())
		cacheCtx, writeCache := ctx.CacheContext()

		// The proposal handler may execute state mutating logic depending
		// on the proposal content. If the handler fails, no state mutation
		// is written and the error message is logged.
		err := handler(cacheCtx, proposal)
		if err == nil {
			proposal.Status = StatusPassed
			// write state to the underlying multi-store
			writeCache()
			return types.AttributeValueProposalPassed, "passed"
		}

		proposal.Status = StatusFailed
		return types.AttributeValueProposalFailed, fmt.Sprintf("passed, but failed on execution: %s",
			err.Error())
	} else if status == StatusRejected {
		if k.ProposalHandlerRouter().HasRoute(proposal.ProposalRoute()) {
			k.ProposalHandlerRouter().GetRoute(proposal.ProposalRoute()).RejectedHandler(ctx, proposal.Content)
		}
		proposal.Status = StatusRejected
		return types.AttributeValueProposalRejected, "rejected"
	}
	return "", ""
}

func hasOnlyDefaultBondDenom(decCoins sdk.SysCoins) sdk.Error {
	if len(decCoins) != 1 || decCoins[0].Denom != sdk.DefaultBondDenom || !decCoins.IsValid() {
		return types.ErrInvalidCoins()
	}
	return nil
}
