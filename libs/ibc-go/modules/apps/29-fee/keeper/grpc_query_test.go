package keeper_test

import (
	"fmt"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	channeltypes "github.com/okx/okbchain/libs/ibc-go/modules/core/04-channel/types"

	"github.com/okx/okbchain/libs/cosmos-sdk/types/query"
	"github.com/okx/okbchain/libs/ibc-go/modules/apps/29-fee/types"
	ibctesting "github.com/okx/okbchain/libs/ibc-go/testing"
	"github.com/okx/okbchain/libs/tendermint/crypto/secp256k1"
)

func (suite *KeeperTestSuite) TestQueryIncentivizedPackets() {
	var (
		req             *types.QueryIncentivizedPacketsRequest
		expectedPackets []types.IdentifiedPacketFees
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {
				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

				fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
				packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount().GetAddress().String(), []string(nil))

				for i := 0; i < 3; i++ {
					// escrow packet fees for three different packets
					packetID := channeltypes.NewPacketId(ibctesting.MockFeePort, ibctesting.FirstChannelID, uint64(i+1))
					suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees([]types.PacketFee{packetFee}))

					expectedPackets = append(expectedPackets, types.NewIdentifiedPacketFees(packetID, []types.PacketFee{packetFee}))
				}

				req = &types.QueryIncentivizedPacketsRequest{
					Pagination: &query.PageRequest{
						Limit:      5,
						CountTotal: false,
					},
					QueryHeight: 0,
				}
			},
			true,
		},
		{
			"empty pagination",
			func() {
				expectedPackets = nil
				req = &types.QueryIncentivizedPacketsRequest{}
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate() // malleate mutates test data

			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())
			res, err := suite.queryClient.IncentivizedPackets(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expectedPackets, res.IncentivizedPackets)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryIncentivizedPacket() {
	var req *types.QueryIncentivizedPacketRequest

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"fees not found for packet id",
			func() {
				req = &types.QueryIncentivizedPacketRequest{
					PacketId:    channeltypes.NewPacketId(ibctesting.MockFeePort, ibctesting.FirstChannelID, 100),
					QueryHeight: 0,
				}
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

			packetID := channeltypes.NewPacketId(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)
			fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount().GetAddress().String(), []string(nil))

			packetFees := []types.PacketFee{packetFee, packetFee, packetFee}
			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))

			req = &types.QueryIncentivizedPacketRequest{
				PacketId:    packetID,
				QueryHeight: 0,
			}

			tc.malleate() // malleate mutates test data

			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())
			res, err := suite.queryClient.IncentivizedPacket(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(types.NewIdentifiedPacketFees(packetID, []types.PacketFee{packetFee, packetFee, packetFee}), res.IncentivizedPacket)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryIncentivizedPacketsForChannel() {
	var (
		req                     *types.QueryIncentivizedPacketsForChannelRequest
		expIdentifiedPacketFees []*types.IdentifiedPacketFees
	)

	fee := types.Fee{
		AckFee:     sdk.CoinAdapters{sdk.CoinAdapter{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}},
		RecvFee:    sdk.CoinAdapters{sdk.CoinAdapter{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}},
		TimeoutFee: sdk.CoinAdapters{sdk.CoinAdapter{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}},
	}

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty pagination",
			func() {
				expIdentifiedPacketFees = nil
				req = &types.QueryIncentivizedPacketsForChannelRequest{}
			},
			true,
		},
		{
			"success",
			func() {
				req = &types.QueryIncentivizedPacketsForChannelRequest{
					Pagination: &query.PageRequest{
						Limit:      5,
						CountTotal: false,
					},
					PortId:      ibctesting.MockFeePort,
					ChannelId:   ibctesting.FirstChannelID,
					QueryHeight: 0,
				}
			},
			true,
		},
		{
			"no packets for specified channel",
			func() {
				expIdentifiedPacketFees = nil
				req = &types.QueryIncentivizedPacketsForChannelRequest{
					Pagination: &query.PageRequest{
						Limit:      5,
						CountTotal: false,
					},
					PortId:      ibctesting.MockFeePort,
					ChannelId:   "channel-10",
					QueryHeight: 0,
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			// setup
			refundAcc := suite.chainA.SenderAccount().GetAddress()
			packetFee := types.NewPacketFee(fee, refundAcc.String(), nil)
			packetFees := types.NewPacketFees([]types.PacketFee{packetFee, packetFee, packetFee})

			identifiedFees1 := types.NewIdentifiedPacketFees(channeltypes.NewPacketId(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1), packetFees.PacketFees)
			identifiedFees2 := types.NewIdentifiedPacketFees(channeltypes.NewPacketId(ibctesting.MockFeePort, ibctesting.FirstChannelID, 2), packetFees.PacketFees)
			identifiedFees3 := types.NewIdentifiedPacketFees(channeltypes.NewPacketId(ibctesting.MockFeePort, ibctesting.FirstChannelID, 3), packetFees.PacketFees)

			expIdentifiedPacketFees = append(expIdentifiedPacketFees, &identifiedFees1, &identifiedFees2, &identifiedFees3)

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)
			for _, identifiedPacketFees := range expIdentifiedPacketFees {
				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), identifiedPacketFees.PacketId, types.NewPacketFees(identifiedPacketFees.PacketFees))
			}

			tc.malleate()
			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())

			res, err := suite.queryClient.IncentivizedPacketsForChannel(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expIdentifiedPacketFees, res.IncentivizedPackets)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryTotalRecvFees() {
	var req *types.QueryTotalRecvFeesRequest

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"packet not found",
			func() {
				req.PacketId = channeltypes.NewPacketId(ibctesting.MockFeePort, ibctesting.FirstChannelID, 100)
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

			packetID := channeltypes.NewPacketId(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)

			fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount().GetAddress().String(), []string(nil))

			packetFees := []types.PacketFee{packetFee, packetFee, packetFee}
			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))

			req = &types.QueryTotalRecvFeesRequest{
				PacketId: packetID,
			}

			tc.malleate()

			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())
			res, err := suite.queryClient.TotalRecvFees(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				// expected total is three times the default recv fee
				expectedFees := defaultRecvFee.Add(defaultRecvFee...).Add(defaultRecvFee...)
				suite.Require().Equal(expectedFees, res.RecvFees)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryTotalAckFees() {
	var req *types.QueryTotalAckFeesRequest

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"packet not found",
			func() {
				req.PacketId = channeltypes.NewPacketId(ibctesting.MockFeePort, ibctesting.FirstChannelID, 100)
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

			packetID := channeltypes.NewPacketId(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)

			fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount().GetAddress().String(), []string(nil))

			packetFees := []types.PacketFee{packetFee, packetFee, packetFee}
			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))

			req = &types.QueryTotalAckFeesRequest{
				PacketId: packetID,
			}

			tc.malleate()

			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())
			res, err := suite.queryClient.TotalAckFees(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				// expected total is three times the default acknowledgement fee
				expectedFees := defaultAckFee.Add(defaultAckFee...).Add(defaultAckFee...)
				suite.Require().Equal(expectedFees, res.AckFees)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryTotalTimeoutFees() {
	var req *types.QueryTotalTimeoutFeesRequest

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"packet not found",
			func() {
				req.PacketId = channeltypes.NewPacketId(ibctesting.MockFeePort, ibctesting.FirstChannelID, 100)
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

			packetID := channeltypes.NewPacketId(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)

			fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount().GetAddress().String(), []string(nil))

			packetFees := []types.PacketFee{packetFee, packetFee, packetFee}
			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))

			req = &types.QueryTotalTimeoutFeesRequest{
				PacketId: packetID,
			}

			tc.malleate()

			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())
			res, err := suite.queryClient.TotalTimeoutFees(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				// expected total is three times the default acknowledgement fee
				expectedFees := defaultTimeoutFee.Add(defaultTimeoutFee...).Add(defaultTimeoutFee...)
				suite.Require().Equal(expectedFees, res.TimeoutFees)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryPayee() {
	var req *types.QueryPayeeRequest

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"payee address not found: invalid channel",
			func() {
				req.ChannelId = "invalid-channel-id"
			},
			false,
		},
		{
			"payee address not found: invalid relayer address",
			func() {
				req.Relayer = "invalid-addr"
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			pk := secp256k1.GenPrivKey().PubKey()
			expPayeeAddr := sdk.AccAddress(pk.Address())

			suite.chainA.GetSimApp().IBCFeeKeeper.SetPayeeAddress(
				suite.chainA.GetContext(),
				suite.chainA.SenderAccount().GetAddress().String(),
				expPayeeAddr.String(),
				suite.path.EndpointA.ChannelID,
			)

			req = &types.QueryPayeeRequest{
				ChannelId: suite.path.EndpointA.ChannelID,
				Relayer:   suite.chainA.SenderAccount().GetAddress().String(),
			}

			tc.malleate()

			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())
			res, err := suite.queryClient.Payee(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expPayeeAddr.String(), res.PayeeAddress)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryCounterpartyPayee() {
	var req *types.QueryCounterpartyPayeeRequest

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"counterparty address not found: invalid channel",
			func() {
				req.ChannelId = "invalid-channel-id"
			},
			false,
		},
		{
			"counterparty address not found: invalid address",
			func() {
				req.Relayer = "invalid-addr"
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			pk := secp256k1.GenPrivKey().PubKey()
			expCounterpartyPayeeAddr := sdk.AccAddress(pk.Address())

			suite.chainA.GetSimApp().IBCFeeKeeper.SetCounterpartyPayeeAddress(
				suite.chainA.GetContext(),
				suite.chainA.SenderAccount().GetAddress().String(),
				expCounterpartyPayeeAddr.String(),
				suite.path.EndpointA.ChannelID,
			)

			req = &types.QueryCounterpartyPayeeRequest{
				ChannelId: suite.path.EndpointA.ChannelID,
				Relayer:   suite.chainA.SenderAccount().GetAddress().String(),
			}

			tc.malleate()

			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())
			res, err := suite.queryClient.CounterpartyPayee(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expCounterpartyPayeeAddr.String(), res.CounterpartyPayee)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryFeeEnabledChannels() {
	var (
		req                   *types.QueryFeeEnabledChannelsRequest
		expFeeEnabledChannels []types.FeeEnabledChannel
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"success: empty pagination",
			func() {
				req = &types.QueryFeeEnabledChannelsRequest{}
			},
			true,
		},
		{
			"success: with multiple fee enabled channels",
			func() {
				suite.coordinator.Setup(suite.pathAToC)

				expChannel := types.FeeEnabledChannel{
					PortId:    suite.pathAToC.EndpointA.ChannelConfig.PortID,
					ChannelId: suite.pathAToC.EndpointA.ChannelID,
				}

				expFeeEnabledChannels = append(expFeeEnabledChannels, expChannel)
			},
			true,
		},
		{
			"success: pagination with multiple fee enabled channels",
			func() {
				// start at index 1, as channel-0 is already added to expFeeEnabledChannels below
				for i := 1; i < 10; i++ {
					channelID := channeltypes.FormatChannelIdentifier(uint64(i))
					suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, channelID)

					expChannel := types.FeeEnabledChannel{
						PortId:    ibctesting.MockFeePort,
						ChannelId: channelID,
					}

					if i < 5 { // add only the first 5 channels, as our default pagination limit is 5
						expFeeEnabledChannels = append(expFeeEnabledChannels, expChannel)
					}
				}

				suite.chainA.Coordinator().CommitBlock(suite.chainA)
			},
			true,
		},
		{
			"empty response",
			func() {
				suite.chainA.GetSimApp().IBCFeeKeeper.DeleteFeeEnabled(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
				expFeeEnabledChannels = nil

				suite.chainA.Coordinator().CommitBlock(suite.chainA)
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			suite.coordinator.Setup(suite.path)

			expChannel := types.FeeEnabledChannel{
				PortId:    suite.path.EndpointA.ChannelConfig.PortID,
				ChannelId: suite.path.EndpointA.ChannelID,
			}

			expFeeEnabledChannels = []types.FeeEnabledChannel{expChannel}

			req = &types.QueryFeeEnabledChannelsRequest{
				Pagination: &query.PageRequest{
					Limit:      5,
					CountTotal: false,
				},
				QueryHeight: 0,
			}

			tc.malleate()

			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())
			res, err := suite.queryClient.FeeEnabledChannels(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expFeeEnabledChannels, res.FeeEnabledChannels)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryFeeEnabledChannel() {
	var req *types.QueryFeeEnabledChannelRequest

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"fee not enabled on channel",
			func() {
				req.ChannelId = "invalid-channel-id"
				req.PortId = "invalid-port-id"
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			suite.coordinator.Setup(suite.path)

			req = &types.QueryFeeEnabledChannelRequest{
				PortId:    suite.path.EndpointA.ChannelConfig.PortID,
				ChannelId: suite.path.EndpointA.ChannelID,
			}

			tc.malleate()

			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())
			res, err := suite.queryClient.FeeEnabledChannel(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().True(res.FeeEnabled)
			} else {
				suite.Require().False(res.FeeEnabled)
			}
		})
	}
}
