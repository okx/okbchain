package ut

import (
	"bytes"
	"strconv"
	"testing"
	"time"

	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	types2 "github.com/okx/okbchain/libs/cosmos-sdk/codec/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/store"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/mpt"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth"
	authexported "github.com/okx/okbchain/libs/cosmos-sdk/x/auth/exported"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/bank"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/crisis"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/supply"
	abci "github.com/okx/okbchain/libs/tendermint/abci/types"
	"github.com/okx/okbchain/libs/tendermint/crypto"
	"github.com/okx/okbchain/libs/tendermint/crypto/ed25519"
	"github.com/okx/okbchain/libs/tendermint/libs/log"
	tmtypes "github.com/okx/okbchain/libs/tendermint/types"
	dbm "github.com/okx/okbchain/libs/tm-db"
	"github.com/okx/okbchain/x/gov/keeper"
	"github.com/okx/okbchain/x/gov/types"
	"github.com/okx/okbchain/x/params"
	"github.com/okx/okbchain/x/staking"
	"github.com/stretchr/testify/require"
)

var (
	// Addrs store generated addresses for test
	Addrs = createTestAddrs(500)

	DefaultMSD = sdk.NewDecWithPrec(1, 3)
)

var (
	pubkeys = []crypto.PubKey{
		ed25519.GenPrivKey().PubKey(), ed25519.GenPrivKey().PubKey(),
		ed25519.GenPrivKey().PubKey(), ed25519.GenPrivKey().PubKey(),
	}

	testDescription = staking.NewDescription("T", "E", "S", "T")
)

// nolint: unparam
func createTestAddrs(numAddrs int) []sdk.AccAddress {
	var addresses []sdk.AccAddress
	var buffer bytes.Buffer

	// start at 100 so we can make up to 999 test addresses with valid test addresses
	for i := 100; i < (numAddrs + 100); i++ {
		numString := strconv.Itoa(i)
		_, err := buffer.WriteString("A58856F0FD53BF058B4909A21AEC019107BA6") //base address string
		if err != nil {
			panic(err)
		}

		_, err = buffer.WriteString(numString) //adding on final two digits to make addresses unique
		if err != nil {
			panic(err)
		}
		res, err := sdk.AccAddressFromHex(buffer.String())
		if err != nil {
			panic(err)
		}
		addresses = append(addresses, res)
		buffer.Reset()
	}
	return addresses
}

// CreateValidators creates validators according to arguments
func CreateValidators(
	t *testing.T, stakingHandler sdk.Handler, ctx sdk.Context, addrs []sdk.ValAddress, powerAmt []int64,
) {
	require.True(t, len(addrs) <= len(pubkeys), "Not enough pubkeys specified at top of file.")

	for i := 0; i < len(addrs); i++ {
		valCreateMsg := staking.NewMsgCreateValidator(
			addrs[i], pubkeys[i],
			testDescription,
			sdk.NewDecCoinFromDec(sdk.DefaultBondDenom, DefaultMSD),
		)

		_, err := stakingHandler(ctx, valCreateMsg)
		require.Nil(t, err)
	}
}

// CreateTestInput returns keepers for test
func CreateTestInput(
	t *testing.T, isCheckTx bool, initBalance int64,
) (sdk.Context, auth.AccountKeeper, keeper.Keeper, staking.Keeper, crisis.Keeper) {
	stakingSk := sdk.NewKVStoreKey(staking.StoreKey)

	stakingTkSk := sdk.NewTransientStoreKey(staking.TStoreKey)

	keyAcc := sdk.NewKVStoreKey(auth.StoreKey)
	keyMpt := sdk.NewKVStoreKey(mpt.StoreKey)
	keyParams := sdk.NewKVStoreKey(params.StoreKey)
	tkeyParams := sdk.NewTransientStoreKey(params.TStoreKey)
	keySupply := sdk.NewKVStoreKey(supply.StoreKey)
	keyGov := sdk.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(stakingTkSk, sdk.StoreTypeTransient, nil)
	ms.MountStoreWithDB(stakingSk, sdk.StoreTypeIAVL, db)

	ms.MountStoreWithDB(keyAcc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyMpt, sdk.StoreTypeMPT, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, sdk.StoreTypeTransient, db)
	ms.MountStoreWithDB(keySupply, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyGov, sdk.StoreTypeIAVL, db)
	err := ms.LoadLatestVersion()
	require.Nil(t, err)

	ctx := sdk.NewContext(ms, abci.Header{ChainID: "okexchain"}, isCheckTx, log.NewNopLogger())
	ctx.SetConsensusParams(
		&abci.ConsensusParams{
			Validator: &abci.ValidatorParams{
				PubKeyTypes: []string{tmtypes.ABCIPubKeyTypeEd25519},
			},
		},
	)
	cdc := MakeTestCodec()
	reg := types2.NewInterfaceRegistry()
	cc := codec.NewProtoCodec(reg)
	pro := codec.NewCodecProxy(cc, cdc)

	feeCollectorAcc := supply.NewEmptyModuleAccount(auth.FeeCollectorName)
	notBondedPool := supply.NewEmptyModuleAccount(staking.NotBondedPoolName, supply.Staking)
	bondPool := supply.NewEmptyModuleAccount(staking.BondedPoolName, supply.Staking)
	govAcc := supply.NewEmptyModuleAccount(types.ModuleName, supply.Staking)

	blacklistedAddrs := make(map[string]bool)
	blacklistedAddrs[feeCollectorAcc.String()] = true
	blacklistedAddrs[notBondedPool.String()] = true
	blacklistedAddrs[bondPool.String()] = true

	pk := params.NewKeeper(cdc, keyParams, tkeyParams, ctx.Logger())
	pk.SetParams(ctx, params.DefaultParams())

	accountKeeper := auth.NewAccountKeeper(
		cdc, // amino codec
		keyMpt,
		pk.Subspace(auth.DefaultParamspace),
		auth.ProtoBaseAccount, // prototype
	)

	bk := bank.NewBaseKeeper(
		accountKeeper,
		pk.Subspace(bank.DefaultParamspace),
		blacklistedAddrs,
	)
	pk.SetBankKeeper(bk)

	maccPerms := map[string][]string{
		auth.FeeCollectorName:     nil,
		staking.NotBondedPoolName: {supply.Staking},
		staking.BondedPoolName:    {supply.Staking},
		types.ModuleName:          nil,
	}
	supplyKeeper := supply.NewKeeper(cdc, keySupply, accountKeeper, bank.NewBankKeeperAdapter(bk), maccPerms)

	initCoins := sdk.NewCoins(sdk.NewInt64DecCoin(sdk.DefaultBondDenom, initBalance))
	totalSupply := sdk.NewCoins(sdk.NewInt64DecCoin(sdk.DefaultBondDenom, initBalance*(int64(len(Addrs)))))

	supplyKeeper.SetSupply(ctx, supply.NewSupply(totalSupply))

	// for staking/distr rollback to cosmos-sdk
	stakingKeeper := staking.NewKeeper(pro, stakingSk, supplyKeeper,
		pk.Subspace(staking.DefaultParamspace))
	stakingKeeper.SetParams(ctx, staking.DefaultDposParams())
	pk.SetStakingKeeper(stakingKeeper)

	// set module accounts
	err = notBondedPool.SetCoins(totalSupply)
	require.NoError(t, err)

	supplyKeeper.SetModuleAccount(ctx, feeCollectorAcc)
	supplyKeeper.SetModuleAccount(ctx, bondPool)
	supplyKeeper.SetModuleAccount(ctx, notBondedPool)
	supplyKeeper.SetModuleAccount(ctx, govAcc)

	// fill all the addresses with some coins, set the loose pool tokens simultaneously
	for _, addr := range Addrs {
		_, err := bk.AddCoins(ctx, addr, initCoins)
		if err != nil {
			panic(err)
		}
	}

	govSubspace := pk.Subspace(types.DefaultParamspace)
	govRouter := keeper.NewRouter()
	govRouter.AddRoute(types.RouterKey, types.ProposalHandler).
		AddRoute(params.RouterKey, params.NewParamChangeProposalHandler(&pk))
	govProposalHandlerRouter := keeper.NewProposalHandlerRouter()
	govProposalHandlerRouter.AddRoute(params.RouterKey, pk)
	keeper := keeper.NewKeeper(cdc, keyGov, pk, govSubspace, supplyKeeper, stakingKeeper,
		types.DefaultCodespace, govRouter, bk, govProposalHandlerRouter, auth.FeeCollectorName)
	pk.SetGovKeeper(keeper)

	minDeposit := sdk.NewDecCoinsFromDec(sdk.DefaultBondDenom, sdk.NewDec(100))
	depositParams := types.DepositParams{
		MinDeposit:       minDeposit,
		MaxDepositPeriod: time.Hour * 24,
	}
	votingParams := types.VotingParams{
		VotingPeriod: time.Hour * 72,
	}
	tallyParams := types.TallyParams{
		Quorum:          sdk.NewDecWithPrec(334, 3),
		Threshold:       sdk.NewDecWithPrec(5, 1),
		Veto:            sdk.NewDecWithPrec(334, 3),
		YesInVotePeriod: sdk.NewDecWithPrec(667, 3),
	}
	keeper.SetProposalID(ctx, 1)
	keeper.SetDepositParams(ctx, depositParams)
	keeper.SetVotingParams(ctx, votingParams)
	keeper.SetTallyParams(ctx, tallyParams)

	crisisKeeper := crisis.NewKeeper(pk.Subspace(crisis.DefaultParamspace), 0,
		supplyKeeper, auth.FeeCollectorName)
	return ctx, accountKeeper, keeper, stakingKeeper, crisisKeeper
}

// MakeTestCodec creates a codec used only for testing
func MakeTestCodec() *codec.Codec {
	var cdc = codec.New()

	// Register Msgs
	cdc.RegisterInterface((*sdk.Msg)(nil), nil)
	cdc.RegisterConcrete(types.MsgSubmitProposal{}, "test/gov/MsgSubmitProposal", nil)
	cdc.RegisterConcrete(types.MsgDeposit{}, "test/gov/MsgDeposit", nil)
	cdc.RegisterConcrete(types.MsgVote{}, "test/gov/MsgVote", nil)

	cdc.RegisterInterface((*types.Content)(nil), nil)
	cdc.RegisterConcrete(types.TextProposal{}, "test/gov/TextProposal", nil)
	cdc.RegisterConcrete(params.ParameterChangeProposal{}, "test/params/ParameterChangeProposal", nil)
	cdc.RegisterConcrete(types.Proposal{}, "test/gov/Proposal", nil)

	// Register AppAccount
	cdc.RegisterInterface((*authexported.Account)(nil), nil)
	cdc.RegisterConcrete(&auth.BaseAccount{}, "test/gov/BaseAccount", nil)
	supply.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)

	return cdc
}
