package keeper

import (
	"fmt"
	"time"

	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	"github.com/okx/brczero/x/evm/types"
	sdkGov "github.com/okx/brczero/x/gov"
	govKeeper "github.com/okx/brczero/x/gov/keeper"
	govTypes "github.com/okx/brczero/x/gov/types"
)

var _ govKeeper.ProposalHandler = (*Keeper)(nil)

// GetMinDeposit returns min deposit
func (k Keeper) GetMinDeposit(ctx sdk.Context, content sdkGov.Content) (minDeposit sdk.SysCoins) {
	switch content.(type) {
	case types.ManageContractDeploymentWhitelistProposal, types.ManageContractBlockedListProposal,
		types.ManageContractMethodBlockedListProposal, types.ManageSysContractAddressProposal, types.ManageContractByteCodeProposal:
		minDeposit = k.govKeeper.GetDepositParams(ctx).MinDeposit
	}

	return
}

// GetMaxDepositPeriod returns max deposit period
func (k Keeper) GetMaxDepositPeriod(ctx sdk.Context, content sdkGov.Content) (maxDepositPeriod time.Duration) {
	switch content.(type) {
	case types.ManageContractDeploymentWhitelistProposal, types.ManageContractBlockedListProposal,
		types.ManageContractMethodBlockedListProposal, types.ManageSysContractAddressProposal, types.ManageContractByteCodeProposal:
		maxDepositPeriod = k.govKeeper.GetDepositParams(ctx).MaxDepositPeriod
	}

	return
}

// GetVotingPeriod returns voting period
func (k Keeper) GetVotingPeriod(ctx sdk.Context, content sdkGov.Content) (votingPeriod time.Duration) {
	switch content.(type) {
	case types.ManageContractDeploymentWhitelistProposal, types.ManageContractBlockedListProposal,
		types.ManageContractMethodBlockedListProposal, types.ManageSysContractAddressProposal, types.ManageContractByteCodeProposal:
		votingPeriod = k.govKeeper.GetVotingParams(ctx).VotingPeriod
	}

	return
}

// CheckMsgSubmitProposal validates MsgSubmitProposal
func (k Keeper) CheckMsgSubmitProposal(ctx sdk.Context, msg govTypes.MsgSubmitProposal) sdk.Error {
	switch content := msg.Content.(type) {
	case types.ManageContractDeploymentWhitelistProposal, types.ManageContractBlockedListProposal:
		// whole target address list will be added/deleted to/from the contract deployment whitelist/contract blocked list.
		// It's not necessary to check the existence in CheckMsgSubmitProposal
		return nil
	case types.ManageContractMethodBlockedListProposal:
		csdb := types.CreateEmptyCommitStateDB(k.GeneratePureCSDBParams(), ctx)
		// can not delete address is not exist
		if !content.IsAdded {
			for i, _ := range content.ContractList {
				bc := csdb.GetContractMethodBlockedByAddress(content.ContractList[i].Address)
				if bc == nil {
					return types.ErrBlockedContractMethodIsNotExist(content.ContractList[i].Address, types.ErrorContractMethodBlockedIsNotExist)
				}
				if _, err := bc.BlockMethods.DeleteContractMethodMap(content.ContractList[i].BlockMethods); err != nil {
					return types.ErrBlockedContractMethodIsNotExist(content.ContractList[i].Address, err)
				}
			}
		}
		return nil
	case types.ManageSysContractAddressProposal:
		if !k.stakingKeeper.IsValidator(ctx, msg.Proposer) {
			return types.ErrCodeProposerMustBeValidator()
		}
		// can not delete system contract address that is not exist
		if !content.IsAdded {
			_, err := k.GetSysContractAddress(ctx)
			return err
		}
		if !k.IsContractAccount(ctx, content.ContractAddr) {
			return types.ErrNotContracAddress(fmt.Errorf(content.ContractAddr.String()))
		}
		return nil
	case types.ManageContractByteCodeProposal:
		if !k.stakingKeeper.IsValidator(ctx, msg.Proposer) {
			return types.ErrCodeProposerMustBeValidator()
		}
		if !k.IsContractAccount(ctx, content.Contract) {
			return types.ErrNotContracAddress(fmt.Errorf(content.Contract.String()))
		}
		if !k.IsContractAccount(ctx, content.SubstituteContract) {
			return types.ErrNotContracAddress(fmt.Errorf(content.SubstituteContract.String()))
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
