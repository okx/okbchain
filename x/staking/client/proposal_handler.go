package client

import (
	"github.com/okx/brczero/x/staking/client/cli"
	"github.com/okx/brczero/x/staking/client/rest"
	govcli "github.com/okx/brczero/x/gov/client"
)

var (
	ProposeValidatorProposalHandler = govcli.NewProposalHandler(
		cli.GetCmdProposeValidatorProposal,
		rest.ProposeValidatorProposalRESTHandler,
	)
)

