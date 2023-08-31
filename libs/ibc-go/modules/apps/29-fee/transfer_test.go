package fee_test

import (
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	txmsg "github.com/okx/brczero/libs/cosmos-sdk/types/ibc-adapter"
	"github.com/okx/brczero/libs/ibc-go/modules/apps/29-fee/types"
	transfertypes "github.com/okx/brczero/libs/ibc-go/modules/apps/transfer/types"
	clienttypes "github.com/okx/brczero/libs/ibc-go/modules/core/02-client/types"
	ibctesting "github.com/okx/brczero/libs/ibc-go/testing"
)

// Integration test to ensure ics29 works with ics20
func (suite *FeeTestSuite) TestFeeTransfer() {
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	feeTransferVersion := string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: types.Version, AppVersion: transfertypes.Version}))
	path.EndpointA.ChannelConfig.Version = feeTransferVersion
	path.EndpointB.ChannelConfig.Version = feeTransferVersion
	path.EndpointA.ChannelConfig.PortID = transfertypes.PortID
	path.EndpointB.ChannelConfig.PortID = transfertypes.PortID

	suite.coordinator.Setup(path)

	// set up coin & ics20 packet
	coin := ibctesting.TestCoin
	fee := types.Fee{
		RecvFee:    defaultRecvFee,
		AckFee:     defaultAckFee,
		TimeoutFee: defaultTimeoutFee,
	}
	coin.Denom = sdk.DefaultIbcWei
	msgs := []txmsg.Msg{
		types.NewMsgPayPacketFee(fee, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, suite.chainA.SenderAccount().GetAddress().String(), nil),
		transfertypes.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, coin, suite.chainA.SenderAccount().GetAddress(), suite.chainB.SenderAccount().GetAddress().String(), clienttypes.NewHeight(0, 100), 0),
	}
	res, err := suite.chainA.SendMsgs(msgs...)
	suite.Require().NoError(err) // message committed

	// after incentivizing the packets
	originalChainASenderAccountBalance := sdk.NewCoins(suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount().GetAddress(), ibctesting.TestCoin.Denom))

	packet, err := ibctesting.ParsePacketFromEvents(res.Events)
	suite.Require().NoError(err)

	// register counterparty address on chainB
	// relayerAddress is address of sender account on chainB, but we will use it on chainA
	// to differentiate from the chainA.SenderAccount for checking successful relay payouts
	relayerAddress := suite.chainB.SenderAccount().GetAddress()

	msgRegister := types.NewMsgRegisterCounterpartyPayee(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, suite.chainB.SenderAccount().GetAddress().String(), relayerAddress.String())
	_, err = suite.chainB.SendMsgs(msgRegister)
	suite.Require().NoError(err) // message committed

	// relay packet
	err = path.RelayPacketV4(packet)
	suite.Require().NoError(err) // relay committed

	// ensure relayers got paid
	// relayer for forward relay: chainB.SenderAccount
	// relayer for reverse relay: chainA.SenderAccount

	// check forward relay balance
	suite.Require().Equal(
		fee.RecvFee.ToCoins(),
		sdk.NewCoins(suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainB.SenderAccount().GetAddress(), ibctesting.TestCoin.Denom)),
	)

	suite.Require().Equal(
		fee.AckFee.Add(fee.TimeoutFee...).ToCoins(), // ack fee paid, timeout fee refunded
		sdk.NewCoins(suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount().GetAddress(), ibctesting.TestCoin.Denom)).Sub(originalChainASenderAccountBalance),
	)
}
