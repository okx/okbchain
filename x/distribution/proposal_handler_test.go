package distribution

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/tendermint/crypto/ed25519"
	govtypes "github.com/okx/okbchain/x/gov/types"

	"github.com/okx/okbchain/x/distribution/keeper"
	"github.com/okx/okbchain/x/distribution/types"
)

var (
	delPk1   = ed25519.GenPrivKey().PubKey()
	delAddr1 = sdk.AccAddress(delPk1.Address())

	amount = sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(1)))
)

func testProposal(recipient sdk.AccAddress, amount sdk.Coins) govtypes.Proposal {
	return govtypes.Proposal{Content: types.NewCommunityPoolSpendProposal(
		"Test",
		"description",
		recipient,
		amount,
	)}
}

func TestProposalHandlerPassed(t *testing.T) {
	ctx, accountKeeper, k, _, supplyKeeper := keeper.CreateTestInputDefault(t, false, 10)
	recipient := delAddr1

	// add coins to the module account
	macc := k.GetDistributionAccount(ctx)
	err := macc.SetCoins(macc.GetCoins().Add(amount...))
	require.NoError(t, err)

	supplyKeeper.SetModuleAccount(ctx, macc)

	account := accountKeeper.NewAccountWithAddress(ctx, recipient)
	require.True(t, account.GetCoins().IsZero())
	accountKeeper.SetAccount(ctx, account)

	feePool := k.GetFeePool(ctx)
	feePool.CommunityPool = sdk.NewCoins(amount...)
	k.SetFeePool(ctx, feePool)

	tp := testProposal(recipient, amount)
	hdlr := NewDistributionProposalHandler(k)
	require.NoError(t, hdlr(ctx, &tp))
	require.Equal(t, accountKeeper.GetAccount(ctx, recipient).GetCoins(), amount)
}

func TestProposalHandlerFailed(t *testing.T) {
	ctx, accountKeeper, k, _, _ := keeper.CreateTestInputDefault(t, false, 10)
	recipient := delAddr1

	account := accountKeeper.NewAccountWithAddress(ctx, recipient)
	require.True(t, account.GetCoins().IsZero())
	accountKeeper.SetAccount(ctx, account)

	tp := testProposal(recipient, amount)
	hdlr := NewDistributionProposalHandler(k)
	require.Error(t, hdlr(ctx, &tp))
	require.True(t, accountKeeper.GetAccount(ctx, recipient).GetCoins().IsZero())
}
