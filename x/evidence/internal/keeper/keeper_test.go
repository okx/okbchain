package keeper_test

import (
	"encoding/hex"
	"os"
	"testing"

	staking "github.com/okx/brczero/x/staking/types"

	"github.com/okx/brczero/app"
	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	"github.com/okx/brczero/libs/tendermint/libs/log"
	dbm "github.com/okx/brczero/libs/tm-db"

	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	"github.com/okx/brczero/libs/cosmos-sdk/x/supply"
	"github.com/okx/brczero/x/evidence"
	"github.com/okx/brczero/x/evidence/exported"
	"github.com/okx/brczero/x/evidence/internal/keeper"
	"github.com/okx/brczero/x/evidence/internal/types"

	abci "github.com/okx/brczero/libs/tendermint/abci/types"
	"github.com/okx/brczero/libs/tendermint/crypto"
	"github.com/okx/brczero/libs/tendermint/crypto/ed25519"
	"github.com/stretchr/testify/suite"
)

var (
	pubkeys = []crypto.PubKey{
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB50"),
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB51"),
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB52"),
	}

	valAddresses = []sdk.ValAddress{
		sdk.ValAddress(pubkeys[0].Address()),
		sdk.ValAddress(pubkeys[1].Address()),
		sdk.ValAddress(pubkeys[2].Address()),
	}

	initAmt   = sdk.NewIntFromUint64(20000)
	initCoins = sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, initAmt))
)

func newPubKey(pk string) (res crypto.PubKey) {
	pkBytes, err := hex.DecodeString(pk)
	if err != nil {
		panic(err)
	}

	var pubkey ed25519.PubKeyEd25519
	copy(pubkey[:], pkBytes)

	return pubkey
}

type KeeperTestSuite struct {
	suite.Suite

	ctx     sdk.Context
	querier sdk.Querier
	keeper  keeper.Keeper
	app     *app.BRCZeroApp
}

func MakeOKEXApp() *app.BRCZeroApp {
	genesisState := app.NewDefaultGenesisState()
	db := dbm.NewMemDB()
	okexapp := app.NewBRCZeroApp(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, nil, true, map[int64]bool{}, 0)

	stateBytes, err := codec.MarshalJSONIndent(okexapp.Codec(), genesisState)
	if err != nil {
		panic(err)
	}
	okexapp.InitChain(
		abci.RequestInitChain{
			Validators:    []abci.ValidatorUpdate{},
			AppStateBytes: stateBytes,
		},
	)
	return okexapp
}

func (suite *KeeperTestSuite) SetupTest() {
	checkTx := false

	app := MakeOKEXApp()
	// get the app's codec and register custom testing types
	cdc := app.Codec()
	cdc.RegisterConcrete(types.TestEquivocationEvidence{}, "test/TestEquivocationEvidence", nil)

	// recreate keeper in order to use custom testing types
	evidenceKeeper := evidence.NewKeeper(
		cdc, app.GetKey(evidence.StoreKey), app.GetSubspace(evidence.ModuleName), app.StakingKeeper, app.SlashingKeeper,
	)
	router := evidence.NewRouter()
	router = router.AddRoute(types.TestEvidenceRouteEquivocation, types.TestEquivocationHandler(*evidenceKeeper))
	evidenceKeeper.SetRouter(router)

	suite.ctx = app.BaseApp.NewContext(checkTx, abci.Header{Height: 1})
	suite.querier = keeper.NewQuerier(*evidenceKeeper)
	suite.keeper = *evidenceKeeper
	suite.app = app
	suite.app.StakingKeeper.SetParams(suite.ctx, staking.DefaultDposParams())
}

func (suite *KeeperTestSuite) populateEvidence(ctx sdk.Context, numEvidence int) []exported.Evidence {
	evidence := make([]exported.Evidence, numEvidence)

	for i := 0; i < numEvidence; i++ {
		pk := ed25519.GenPrivKey()
		sv := types.TestVote{
			ValidatorAddress: pk.PubKey().Address(),
			Height:           int64(i),
			Round:            0,
		}

		sig, err := pk.Sign(sv.SignBytes(ctx.ChainID()))
		suite.NoError(err)
		sv.Signature = sig

		evidence[i] = types.TestEquivocationEvidence{
			Power:      100,
			TotalPower: 100000,
			PubKey:     pk.PubKey(),
			VoteA:      sv,
			VoteB:      sv,
		}

		suite.Nil(suite.keeper.SubmitEvidence(ctx, evidence[i]))
	}

	return evidence
}

func (suite *KeeperTestSuite) populateValidators(ctx sdk.Context) {
	// add accounts and set total supply
	totalSupplyAmt := initAmt.MulRaw(int64(len(valAddresses)))
	totalSupply := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, totalSupplyAmt))
	suite.app.SupplyKeeper.SetSupply(ctx, supply.NewSupply(totalSupply))

	for _, addr := range valAddresses {
		_, err := suite.app.BankKeeper.AddCoins(ctx, sdk.AccAddress(addr), initCoins)
		suite.NoError(err)
	}
}

func (suite *KeeperTestSuite) TestSubmitValidEvidence() {
	ctx := suite.ctx.WithIsCheckTx(false)
	pk := ed25519.GenPrivKey()
	sv := types.TestVote{
		ValidatorAddress: pk.PubKey().Address(),
		Height:           11,
		Round:            0,
	}

	sig, err := pk.Sign(sv.SignBytes(ctx.ChainID()))
	suite.NoError(err)
	sv.Signature = sig

	e := types.TestEquivocationEvidence{
		Power:      100,
		TotalPower: 100000,
		PubKey:     pk.PubKey(),
		VoteA:      sv,
		VoteB:      sv,
	}

	suite.Nil(suite.keeper.SubmitEvidence(ctx, e))

	res, ok := suite.keeper.GetEvidence(ctx, e.Hash())
	suite.True(ok)
	suite.Equal(e, res)
}

func (suite *KeeperTestSuite) TestSubmitValidEvidence_Duplicate() {
	ctx := suite.ctx.WithIsCheckTx(false)
	pk := ed25519.GenPrivKey()
	sv := types.TestVote{
		ValidatorAddress: pk.PubKey().Address(),
		Height:           11,
		Round:            0,
	}

	sig, err := pk.Sign(sv.SignBytes(ctx.ChainID()))
	suite.NoError(err)
	sv.Signature = sig

	e := types.TestEquivocationEvidence{
		Power:      100,
		TotalPower: 100000,
		PubKey:     pk.PubKey(),
		VoteA:      sv,
		VoteB:      sv,
	}

	suite.Nil(suite.keeper.SubmitEvidence(ctx, e))
	suite.Error(suite.keeper.SubmitEvidence(ctx, e))

	res, ok := suite.keeper.GetEvidence(ctx, e.Hash())
	suite.True(ok)
	suite.Equal(e, res)
}

func (suite *KeeperTestSuite) TestSubmitInvalidEvidence() {
	ctx := suite.ctx.WithIsCheckTx(false)
	pk := ed25519.GenPrivKey()
	e := types.TestEquivocationEvidence{
		Power:      100,
		TotalPower: 100000,
		PubKey:     pk.PubKey(),
		VoteA: types.TestVote{
			ValidatorAddress: pk.PubKey().Address(),
			Height:           10,
			Round:            0,
		},
		VoteB: types.TestVote{
			ValidatorAddress: pk.PubKey().Address(),
			Height:           11,
			Round:            0,
		},
	}

	suite.Error(suite.keeper.SubmitEvidence(ctx, e))

	res, ok := suite.keeper.GetEvidence(ctx, e.Hash())
	suite.False(ok)
	suite.Nil(res)
}

func (suite *KeeperTestSuite) TestIterateEvidence() {
	ctx := suite.ctx.WithIsCheckTx(false)
	numEvidence := 100
	suite.populateEvidence(ctx, numEvidence)

	evidence := suite.keeper.GetAllEvidence(ctx)
	suite.Len(evidence, numEvidence)
}

func (suite *KeeperTestSuite) TestGetEvidenceHandler() {
	handler, err := suite.keeper.GetEvidenceHandler(types.TestEquivocationEvidence{}.Route())
	suite.NoError(err)
	suite.NotNil(handler)

	handler, err = suite.keeper.GetEvidenceHandler("invalidHandler")
	suite.Error(err)
	suite.Nil(handler)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
