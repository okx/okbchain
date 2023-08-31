package evm_test

import (
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/okx/brczero/x/evm"
	"github.com/okx/brczero/x/evm/types"
	govtypes "github.com/okx/brczero/x/gov/types"
)

func (suite *EvmTestSuite) TestProposalHandler_ManageContractDeploymentWhitelistProposal() {
	addr1 := ethcmn.BytesToAddress([]byte{0x0}).Bytes()
	addr2 := ethcmn.BytesToAddress([]byte{0x1}).Bytes()

	proposal := types.NewManageContractDeploymentWhitelistProposal(
		"default title",
		"default description",
		types.AddressList{addr1, addr2},
		true,
	)

	suite.govHandler = evm.NewManageContractDeploymentWhitelistProposalHandler(suite.app.EvmKeeper)
	govProposal := govtypes.Proposal{
		Content: proposal,
	}

	testCases := []struct {
		msg                   string
		prepare               func()
		targetAddrListToCheck types.AddressList
	}{
		{
			"add address into whitelist",
			func() {},
			types.AddressList{addr1, addr2},
		},
		{
			"add address repeatedly",
			func() {},
			types.AddressList{addr1, addr2},
		},
		{
			"delete an address from whitelist",
			func() {
				proposal.IsAdded = false
				proposal.DistributorAddrs = types.AddressList{addr1}
				govProposal.Content = proposal
			},
			types.AddressList{addr2},
		},
		{
			"delete an address from whitelist",
			func() {
				proposal.IsAdded = false
				proposal.DistributorAddrs = types.AddressList{addr1}
				govProposal.Content = proposal
			},
			types.AddressList{addr2},
		},
		{
			"delete two addresses from whitelist which contains one of them only",
			func() {
				proposal.DistributorAddrs = types.AddressList{addr1, addr2}
				govProposal.Content = proposal
			},
			types.AddressList{},
		},
		{
			"delete two addresses from whitelist which contains none of them",
			func() {},
			types.AddressList{},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.msg, func() {
			tc.prepare()

			err := suite.govHandler(suite.ctx, &govProposal)
			suite.Require().NoError(err)

			// check the whitelist with target address list
			curWhitelist := suite.stateDB.GetContractDeploymentWhitelist()
			suite.Require().Equal(len(tc.targetAddrListToCheck), len(curWhitelist))

			for i, addr := range curWhitelist {
				suite.Require().Equal(tc.targetAddrListToCheck[i], addr)
			}
		})
	}
}

func (suite *EvmTestSuite) TestProposalHandler_ManageContractBlockedListProposal() {
	addr1 := ethcmn.BytesToAddress([]byte{0x0}).Bytes()
	addr2 := ethcmn.BytesToAddress([]byte{0x1}).Bytes()

	proposal := types.NewManageContractBlockedListProposal(
		"default title",
		"default description",
		types.AddressList{addr1, addr2},
		true,
	)

	suite.govHandler = evm.NewManageContractDeploymentWhitelistProposalHandler(suite.app.EvmKeeper)
	govProposal := govtypes.Proposal{
		Content: proposal,
	}

	testCases := []struct {
		msg                   string
		prepare               func()
		targetAddrListToCheck types.AddressList
	}{
		{
			"add address into blocked list",
			func() {},
			types.AddressList{addr1, addr2},
		},
		{
			"add address repeatedly",
			func() {},
			types.AddressList{addr1, addr2},
		},
		{
			"delete an address from blocked list",
			func() {
				proposal.IsAdded = false
				proposal.ContractAddrs = types.AddressList{addr1}
				govProposal.Content = proposal
			},
			types.AddressList{addr2},
		},
		{
			"delete an address from blocked list",
			func() {
				proposal.IsAdded = false
				proposal.ContractAddrs = types.AddressList{addr1}
				govProposal.Content = proposal
			},
			types.AddressList{addr2},
		},
		{
			"delete two addresses from blocked list which contains one of them only",
			func() {
				proposal.ContractAddrs = types.AddressList{addr1, addr2}
				govProposal.Content = proposal
			},
			types.AddressList{},
		},
		{
			"delete two addresses from blocked list which contains none of them",
			func() {},
			types.AddressList{},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.msg, func() {
			tc.prepare()

			err := suite.govHandler(suite.ctx, &govProposal)
			suite.Require().NoError(err)

			// check the blocked list with target address list
			curBlockedList := suite.stateDB.GetContractBlockedList()
			suite.Require().Equal(len(tc.targetAddrListToCheck), len(curBlockedList))

			for i, addr := range curBlockedList {
				suite.Require().Equal(tc.targetAddrListToCheck[i], addr)
			}
		})
	}
}

func (suite *EvmTestSuite) TestProposalHandler_ManageContractMethodBlockedListProposal() {
	addr1 := ethcmn.BytesToAddress([]byte{0x0}).Bytes()
	addr2 := ethcmn.BytesToAddress([]byte{0x1}).Bytes()
	bcMethodOne1 := types.BlockedContract{
		Address: addr1,
		BlockMethods: types.ContractMethods{
			types.ContractMethod{
				Sign:  "aaaa",
				Extra: "aaaa()",
			},
		},
	}
	bcMethodTwo1 := types.BlockedContract{
		Address: addr2,
		BlockMethods: types.ContractMethods{
			types.ContractMethod{
				Sign:  "bbbb",
				Extra: "bbbb()",
			},
		},
	}

	bcMethodOne2 := types.BlockedContract{
		Address: addr1,
		BlockMethods: types.ContractMethods{
			types.ContractMethod{
				Sign:  "bbbb",
				Extra: "bbbb()",
			},
		},
	}
	expectBcMethodOne2 := types.NewBlockContract(addr1, bcMethodOne1.BlockMethods)
	expectBcMethodOne2.BlockMethods = append(expectBcMethodOne2.BlockMethods, bcMethodOne2.BlockMethods...)

	proposal := types.NewManageContractMethodBlockedListProposal(
		"default title",
		"default description",
		types.BlockedContractList{bcMethodOne1, bcMethodTwo1},
		true,
	)

	suite.govHandler = evm.NewManageContractDeploymentWhitelistProposalHandler(suite.app.EvmKeeper)
	govProposal := govtypes.Proposal{
		Content: proposal,
	}

	testCases := []struct {
		msg                   string
		prepare               func()
		targetAddrListToCheck types.BlockedContractList
		success               bool
	}{
		{
			"add address into blocked list",
			func() {},
			types.BlockedContractList{bcMethodOne1, bcMethodTwo1},
			true,
		},
		{
			"add address repeatedly",
			func() {},
			types.BlockedContractList{bcMethodOne1, bcMethodTwo1},
			true,
		},
		{
			"add method into contract method blocked list",
			func() {
				//reset data
				suite.stateDB.DeleteContractBlockedList(types.AddressList{addr1, addr2})
				suite.stateDB.InsertContractMethodBlockedList(types.BlockedContractList{bcMethodOne1})
				proposal.ContractList = types.BlockedContractList{bcMethodOne2, bcMethodTwo1}
				govProposal.Content = proposal
			},
			types.BlockedContractList{*expectBcMethodOne2, bcMethodTwo1},
			true,
		},
		{
			"add method into contract method which has same addr int blocked list",
			func() {
				//reset data
				suite.stateDB.DeleteContractBlockedList(types.AddressList{addr1, addr2})
				suite.stateDB.SetContractBlockedList(types.AddressList{addr1})
				proposal.ContractList = types.BlockedContractList{bcMethodOne1, bcMethodTwo1}
				govProposal.Content = proposal
			},
			types.BlockedContractList{bcMethodOne1, bcMethodTwo1},
			true,
		},
		{
			"delete all method from blocked list",
			func() {
				proposal.IsAdded = false
				proposal.ContractList = types.BlockedContractList{bcMethodOne1}
				govProposal.Content = proposal
			},
			types.BlockedContractList{bcMethodTwo1},
			true,
		},
		{
			"delete a method from blocked list",
			func() {
				//reset data
				suite.stateDB.DeleteContractBlockedList(types.AddressList{addr1, addr2})
				suite.stateDB.InsertContractMethodBlockedList(types.BlockedContractList{*expectBcMethodOne2, bcMethodTwo1})
				proposal.IsAdded = false
				proposal.ContractList = types.BlockedContractList{bcMethodOne1, bcMethodTwo1}
				govProposal.Content = proposal
			},
			types.BlockedContractList{bcMethodOne2},
			true,
		},
		{
			"delete a method from blocked list which is contract all method blocked",
			func() {
				//reset data
				suite.stateDB.DeleteContractBlockedList(types.AddressList{addr1, addr2})
				suite.stateDB.SetContractBlockedList(types.AddressList{addr1})
				proposal.IsAdded = false
				proposal.ContractList = types.BlockedContractList{bcMethodOne1, bcMethodTwo1}
				govProposal.Content = proposal
			},
			types.BlockedContractList{types.BlockedContract{Address: addr1}},
			false,
		},
		{
			"delete two addresses from blocked list which contains one of them only",
			func() {
				//reset data
				suite.stateDB.DeleteContractBlockedList(types.AddressList{addr1, addr2})
				suite.stateDB.InsertContractMethodBlockedList(types.BlockedContractList{bcMethodTwo1})

				proposal.IsAdded = false
				proposal.ContractList = types.BlockedContractList{bcMethodOne1, bcMethodTwo1}
				govProposal.Content = proposal
			},
			types.BlockedContractList{bcMethodTwo1},
			false,
		},
		{
			"delete two addresses from blocked list which contains none of them",
			func() {
				//reset data
				suite.stateDB.DeleteContractBlockedList(types.AddressList{addr1, addr2})
				proposal.IsAdded = false
				proposal.ContractList = types.BlockedContractList{bcMethodOne1, bcMethodTwo1}
				govProposal.Content = proposal
			},
			types.BlockedContractList{},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.msg, func() {
			tc.prepare()

			err := suite.govHandler(suite.ctx, &govProposal)
			if tc.success {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

			// check the blocked list with target address list
			curBlockedList := suite.stateDB.GetContractMethodBlockedList()
			suite.Require().Equal(len(tc.targetAddrListToCheck), len(curBlockedList))
			ok := types.BlockedContractListIsEqual(suite.T(), curBlockedList, tc.targetAddrListToCheck)
			suite.Require().True(ok)
		})
	}
}

func (suite *EvmTestSuite) TestProposalHandler_ManageSysContractAddressProposal() {
	addr1 := ethcmn.BytesToAddress([]byte{0x3}).Bytes()
	addr2 := ethcmn.BytesToAddress([]byte{0x4}).Bytes()

	suite.govHandler = evm.NewManageContractDeploymentWhitelistProposalHandler(suite.app.EvmKeeper)

	govProposal := &govtypes.Proposal{}

	testCases := []struct {
		msg     string
		prepare func()
		fnCheck func()
		success bool
	}{
		{
			msg: "add a sys contract address addr1",
			prepare: func() {
				proposal := types.NewManageSysContractAddressProposal(
					"default title",
					"default description",
					addr1,
					true,
				)
				govProposal.Content = proposal
			},
			fnCheck: func() {
				reAddr, err := suite.app.EvmKeeper.GetSysContractAddress(suite.ctx)
				suite.Require().NoError(err)
				suite.Require().Equal(addr1, reAddr[:])
			},
			success: true,
		},
		{
			msg: "add a sys contract address addr2",
			prepare: func() {
				proposal := types.NewManageSysContractAddressProposal(
					"default title",
					"default description",
					addr2,
					true,
				)
				govProposal.Content = proposal
			},
			fnCheck: func() {
				reAddr, err := suite.app.EvmKeeper.GetSysContractAddress(suite.ctx)
				suite.Require().NoError(err)
				suite.Require().Equal(addr2, reAddr[:])
			},
			success: true,
		},
		{
			msg: "del a sys contract address",
			prepare: func() {
				proposal := types.NewManageSysContractAddressProposal(
					"default title",
					"default description",
					addr2,
					false,
				)
				govProposal.Content = proposal
			},
			fnCheck: func() {
				reAddr, err := suite.app.EvmKeeper.GetSysContractAddress(suite.ctx)
				suite.Require().Error(err)
				suite.Require().Nil(reAddr)
			},
			success: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.msg, func() {
			tc.prepare()

			err := suite.govHandler(suite.ctx, govProposal)
			if tc.success {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
