package keeper_test

import (
	"github.com/okx/okbchain/libs/tendermint/crypto"
	"testing"

	"github.com/stretchr/testify/suite"
	// "github.com/tendermint/tendermint/crypto"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/ibc-go/modules/apps/transfer/types"
	ibctesting "github.com/okx/okbchain/libs/ibc-go/testing"
)

type KeeperTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA ibctesting.TestChainI
	chainB ibctesting.TestChainI
	chainC ibctesting.TestChainI

	queryClient types.QueryClient
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 3)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	//todo no query
	//queryHelper := baseapp.NewQueryServerTestHelper(suite.chainA.GetContext(), suite.chainA.GetSimApp().InterfaceRegistry())
	//types.RegisterQueryServer(queryHelper, suite.chainA.GetSimApp().TransferKeeper)
	//suite.queryClient = types.NewQueryClient(queryHelper)
}

func NewTransferPath(chainA, chainB ibctesting.TestChainI) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = ibctesting.TransferPort
	path.EndpointB.ChannelConfig.PortID = ibctesting.TransferPort

	return path
}

func (suite *KeeperTestSuite) TestGetTransferAccount() {
	expectedMaccAddr := sdk.AccAddress(crypto.AddressHash([]byte(types.ModuleName)))

	macc := suite.chainA.GetSimApp().TransferKeeper.GetTransferAccount(suite.chainA.GetContext())

	suite.Require().NotNil(macc)
	suite.Require().Equal(types.ModuleName, macc.GetName())
	suite.Require().Equal(expectedMaccAddr, macc.GetAddress())
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
