package keeper_test

import (
	govtypes "github.com/okx/okbchain/x/gov/types"

	// govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	// upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/okx/okbchain/libs/ibc-go/modules/core/02-client/types"
	"github.com/okx/okbchain/libs/ibc-go/modules/core/exported"
	ibctmtypes "github.com/okx/okbchain/libs/ibc-go/modules/light-clients/07-tendermint/types"
	ibctesting "github.com/okx/okbchain/libs/ibc-go/testing"
)

func (suite *KeeperTestSuite) TestClientUpdateProposal() {
	var (
		subject, substitute                       string
		subjectClientState, substituteClientState exported.ClientState
		content                                   govtypes.Content
		err                                       error
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"valid update client proposal", func() {
				content = types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, substitute)
			}, true,
		},
		{
			"subject and substitute use different revision numbers", func() {
				tmClientState, ok := substituteClientState.(*ibctmtypes.ClientState)
				suite.Require().True(ok)
				consState, found := suite.chainA.App().GetIBCKeeper().ClientKeeper.GetClientConsensusState(suite.chainA.GetContext(), substitute, tmClientState.LatestHeight)
				suite.Require().True(found)
				newRevisionNumber := tmClientState.GetLatestHeight().GetRevisionNumber() + 1

				tmClientState.LatestHeight = types.NewHeight(newRevisionNumber, tmClientState.GetLatestHeight().GetRevisionHeight())

				suite.chainA.App().GetIBCKeeper().ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), substitute, tmClientState.LatestHeight, consState)
				clientStore := suite.chainA.App().GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), substitute)
				ibctmtypes.SetProcessedTime(clientStore, tmClientState.LatestHeight, 100)
				ibctmtypes.SetProcessedHeight(clientStore, tmClientState.LatestHeight, types.NewHeight(0, 1))
				suite.chainA.App().GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), substitute, tmClientState)

				content = types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, substitute)
			}, true,
		},
		{
			"cannot use localhost as subject", func() {
				content = types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, exported.Localhost, substitute)
			}, false,
		},
		{
			"cannot use localhost as substitute", func() {
				content = types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, exported.Localhost)
			}, false,
		},
		{
			"cannot use solomachine as substitute for tendermint client", func() {
				solomachine := ibctesting.NewSolomachine(suite.T(), suite.cdc, "solo machine", "", 1)
				solomachine.Sequence = subjectClientState.GetLatestHeight().GetRevisionHeight() + 1
				substituteClientState = solomachine.ClientState()
				suite.chainA.App().GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), substitute, substituteClientState)
				content = types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, substitute)
			}, false,
		},
		{
			"subject client does not exist", func() {
				content = types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, ibctesting.InvalidID, substitute)
			}, false,
		},
		{
			"substitute client does not exist", func() {
				content = types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, ibctesting.InvalidID)
			}, false,
		},
		{
			"subject and substitute have equal latest height", func() {
				tmClientState, ok := subjectClientState.(*ibctmtypes.ClientState)
				suite.Require().True(ok)
				tmClientState.LatestHeight = substituteClientState.GetLatestHeight().(types.Height)
				suite.chainA.App().GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), subject, tmClientState)

				content = types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, substitute)
			}, false,
		},
		{
			"update fails, client is not frozen or expired", func() {
				tmClientState, ok := subjectClientState.(*ibctmtypes.ClientState)
				suite.Require().True(ok)
				tmClientState.FrozenHeight = types.ZeroHeight()
				suite.chainA.App().GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), subject, tmClientState)

				content = types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, substitute)
			}, false,
		},
		{
			"substitute is frozen", func() {
				tmClientState, ok := substituteClientState.(*ibctmtypes.ClientState)
				suite.Require().True(ok)
				tmClientState.FrozenHeight = types.NewHeight(0, 1)
				suite.chainA.App().GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), substitute, tmClientState)

				content = types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, substitute)
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			subjectPath := ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupClients(subjectPath)
			subject = subjectPath.EndpointA.ClientID
			subjectClientState = suite.chainA.GetClientState(subject)

			substitutePath := ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupClients(substitutePath)
			substitute = substitutePath.EndpointA.ClientID

			// update substitute twice
			substitutePath.EndpointA.UpdateClient()
			substitutePath.EndpointA.UpdateClient()
			substituteClientState = suite.chainA.GetClientState(substitute)

			tmClientState, ok := subjectClientState.(*ibctmtypes.ClientState)
			suite.Require().True(ok)
			tmClientState.AllowUpdateAfterMisbehaviour = true
			tmClientState.AllowUpdateAfterExpiry = true
			tmClientState.FrozenHeight = tmClientState.LatestHeight
			suite.chainA.App().GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), subject, tmClientState)

			tmClientState, ok = substituteClientState.(*ibctmtypes.ClientState)
			suite.Require().True(ok)
			tmClientState.AllowUpdateAfterMisbehaviour = true
			tmClientState.AllowUpdateAfterExpiry = true
			suite.chainA.App().GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), substitute, tmClientState)

			tc.malleate()

			updateProp, ok := content.(*types.ClientUpdateProposal)
			suite.Require().True(ok)
			err = suite.chainA.App().GetIBCKeeper().ClientKeeper.ClientUpdateProposal(suite.chainA.GetContext(), updateProp)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}

}
