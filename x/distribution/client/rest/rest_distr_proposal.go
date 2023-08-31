package rest

import (
	"net/http"

	"github.com/okx/brczero/libs/cosmos-sdk/client/context"
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	"github.com/okx/brczero/libs/cosmos-sdk/types/rest"
	"github.com/okx/brczero/libs/cosmos-sdk/x/auth/client/utils"

	comm "github.com/okx/brczero/x/common"
	"github.com/okx/brczero/x/distribution/types"
	"github.com/okx/brczero/x/gov"
	govrest "github.com/okx/brczero/x/gov/client/rest"
)

// ChangeDistributionTypeProposalRESTHandler returns a ChangeDistributionTypeProposal that exposes the change distribution type REST handler with a given sub-route.
func ChangeDistributionTypeProposalRESTHandler(cliCtx context.CLIContext) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "change_distribution_type",
		Handler:  postChangeDistributionTypeProposalHandlerFn(cliCtx),
	}
}

func postChangeDistributionTypeProposalHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ChangeDistributionTypeProposalReq
		if !rest.ReadRESTReq(w, r, cliCtx.Codec, &req) {
			return
		}

		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		content := types.NewChangeDistributionTypeProposal(req.Title, req.Description, req.Type)

		msg := gov.NewMsgSubmitProposal(content, req.Deposit, req.Proposer)
		if err := msg.ValidateBasic(); err != nil {
			comm.HandleErrorMsg(w, cliCtx, comm.CodeInvalidParam, err.Error())
			return
		}

		utils.WriteGenerateStdTxResponse(w, cliCtx, req.BaseReq, []sdk.Msg{msg})
	}
}

// WithdrawRewardEnabledProposalRESTHandler returns a WithdrawRewardEnabledProposal that exposes the set withdraw reward enabled proposal REST handler with a given sub-route.
func WithdrawRewardEnabledProposalRESTHandler(cliCtx context.CLIContext) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "withdraw_reward_enabled",
		Handler:  postWithdrawRewardEnabledProposalHandlerFn(cliCtx),
	}
}

func postWithdrawRewardEnabledProposalHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req WithdrawRewardEnabledProposalReq
		if !rest.ReadRESTReq(w, r, cliCtx.Codec, &req) {
			return
		}

		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		content := types.NewWithdrawRewardEnabledProposal(req.Title, req.Description, req.Enabled)

		msg := gov.NewMsgSubmitProposal(content, req.Deposit, req.Proposer)
		if err := msg.ValidateBasic(); err != nil {
			comm.HandleErrorMsg(w, cliCtx, comm.CodeInvalidParam, err.Error())
			return
		}

		utils.WriteGenerateStdTxResponse(w, cliCtx, req.BaseReq, []sdk.Msg{msg})
	}
}

// RewardTruncatePrecisionProposalRESTHandler returns a RewardTruncatePrecisionProposal
//that exposes the reward truncate precision proposal REST handler with a given sub-route.
func RewardTruncatePrecisionProposalRESTHandler(cliCtx context.CLIContext) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "reward_truncate_precision",
		Handler:  postRewardTruncatePrecisionProposalHandlerFn(cliCtx),
	}
}

func postRewardTruncatePrecisionProposalHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RewardTruncatePrecisionProposalReq
		if !rest.ReadRESTReq(w, r, cliCtx.Codec, &req) {
			return
		}

		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		content := types.NewRewardTruncatePrecisionProposal(req.Title, req.Description, req.Precision)

		msg := gov.NewMsgSubmitProposal(content, req.Deposit, req.Proposer)
		if err := msg.ValidateBasic(); err != nil {
			comm.HandleErrorMsg(w, cliCtx, comm.CodeInvalidParam, err.Error())
			return
		}

		utils.WriteGenerateStdTxResponse(w, cliCtx, req.BaseReq, []sdk.Msg{msg})
	}
}
