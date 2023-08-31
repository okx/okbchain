package client

import (
	"github.com/okx/brczero/x/feesplit/client/cli"
	"github.com/okx/brczero/x/feesplit/client/rest"
	govcli "github.com/okx/brczero/x/gov/client"
)

var (
	// FeeSplitSharesProposalHandler alias gov NewProposalHandler
	FeeSplitSharesProposalHandler = govcli.NewProposalHandler(
		cli.GetCmdFeeSplitSharesProposal,
		rest.FeeSplitSharesProposalRESTHandler,
	)
)
