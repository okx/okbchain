package keeper_test

import "github.com/okx/brczero/x/feesplit/types"

func (suite *KeeperTestSuite) TestParams() {
	params := suite.app.FeeSplitKeeper.GetParams(suite.ctx)
	suite.Require().Equal(types.DefaultParams(), params)
	params.EnableFeeSplit = true
	suite.app.FeeSplitKeeper.SetParams(suite.ctx, params)
	newParams := suite.app.FeeSplitKeeper.GetParams(suite.ctx)
	suite.Require().Equal(newParams, params)
}
