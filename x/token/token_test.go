package token

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"

	app "github.com/okx/okbchain/app/types"

	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/bank"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/mock"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/supply"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/supply/exported"
	abci "github.com/okx/okbchain/libs/tendermint/abci/types"
	"github.com/okx/okbchain/libs/tendermint/crypto"
	"github.com/okx/okbchain/libs/tendermint/crypto/secp256k1"
	"github.com/okx/okbchain/x/common"
	"github.com/okx/okbchain/x/common/version"
	"github.com/okx/okbchain/x/token/types"
	"github.com/stretchr/testify/require"
)

var mockBlockHeight int64 = -1

type MockDexApp struct {
	*mock.App

	keyToken  *sdk.KVStoreKey
	keyLock   *sdk.KVStoreKey
	keySupply *sdk.KVStoreKey

	bankKeeper   bank.Keeper
	tokenKeeper  Keeper
	supplyKeeper supply.Keeper
}

func registerCodec(cdc *codec.Codec) {
	RegisterCodec(cdc)
	supply.RegisterCodec(cdc)
}

func getEndBlocker(keeper Keeper) sdk.EndBlocker {
	return func(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
		return abci.ResponseEndBlock{}
	}
}

// initialize the mock application for this module
func getMockDexApp(t *testing.T, numGenAccs int) (mockDexApp *MockDexApp, keeper Keeper, addrs []sdk.AccAddress) {

	mapp := mock.NewApp()
	//mapp.Cdc = makeCodec()
	registerCodec(mapp.Cdc.GetCdc())
	app.RegisterCodec(mapp.Cdc.GetCdc())

	mockDexApp = &MockDexApp{
		App: mapp,

		keyToken:  sdk.NewKVStoreKey("token"),
		keyLock:   sdk.NewKVStoreKey("lock"),
		keySupply: sdk.NewKVStoreKey(supply.StoreKey),
	}

	feeCollectorAcc := supply.NewEmptyModuleAccount(auth.FeeCollectorName)
	blacklistedAddrs := make(map[string]bool)
	blacklistedAddrs[feeCollectorAcc.Address.String()] = true

	mockDexApp.bankKeeper = bank.NewBaseKeeper(
		mockDexApp.AccountKeeper,
		mockDexApp.ParamsKeeper.Subspace(bank.DefaultParamspace),
		blacklistedAddrs,
	)

	maccPerms := map[string][]string{
		auth.FeeCollectorName: nil,
		types.ModuleName:      {supply.Minter, supply.Burner},
	}
	mockDexApp.supplyKeeper = supply.NewKeeper(mockDexApp.Cdc.GetCdc(), mockDexApp.keySupply, mockDexApp.AccountKeeper, bank.NewBankKeeperAdapter(mockDexApp.bankKeeper), maccPerms)
	mockDexApp.tokenKeeper = NewKeeper(
		mockDexApp.bankKeeper,
		mockDexApp.ParamsKeeper.Subspace(DefaultParamspace),
		auth.FeeCollectorName,
		mockDexApp.supplyKeeper,
		mockDexApp.keyToken,
		mockDexApp.keyLock,
		mockDexApp.Cdc.GetCdc(),
		true, mapp.AccountKeeper)

	handler := NewTokenHandler(mockDexApp.tokenKeeper, version.CurrentProtocolVersion)

	mockDexApp.Router().AddRoute(RouterKey, handler)
	mockDexApp.QueryRouter().AddRoute(QuerierRoute, NewQuerier(mockDexApp.tokenKeeper))

	mockDexApp.SetEndBlocker(getEndBlocker(mockDexApp.tokenKeeper))
	mockDexApp.SetInitChainer(getInitChainer(mockDexApp.App, mockDexApp.bankKeeper, mockDexApp.supplyKeeper, []exported.ModuleAccountI{feeCollectorAcc}))

	intQuantity := int64(100000)
	valTokens := sdk.NewDec(intQuantity)
	coins := sdk.SysCoins{
		sdk.NewDecCoinFromDec(common.NativeToken, valTokens),
		sdk.NewDecCoinFromDec(common.TestToken, valTokens),
	}

	genAccs, addrs, _, _ := mock.CreateGenAccounts(numGenAccs, coins)

	// todo: checkTx in mock app
	mockDexApp.SetAnteHandler(nil)

	app := mockDexApp
	require.NoError(t, app.CompleteSetup(
		app.keyToken,
		app.keyLock,
		app.keySupply,
	))
	// TODO: set genesis
	app.BaseApp.NewContext(true, abci.Header{})
	mock.SetGenesis(mockDexApp.App, genAccs)

	for i := 0; i < numGenAccs; i++ {
		mock.CheckBalance(t, app.App, addrs[i], coins)
		mockDexApp.TotalCoinsSupply = mockDexApp.TotalCoinsSupply.Add2(coins)
	}

	return mockDexApp, mockDexApp.tokenKeeper, addrs
}

// initialize the mock application for this module
func getMockDexAppEx(t *testing.T, numGenAccs int) (mockDexApp *MockDexApp, keeper Keeper, h sdk.Handler) {

	mapp := mock.NewApp()
	//mapp.Cdc = makeCodec()
	registerCodec(mapp.Cdc.GetCdc())

	mockDexApp = &MockDexApp{
		App: mapp,

		keySupply: sdk.NewKVStoreKey(supply.StoreKey),
		keyToken:  sdk.NewKVStoreKey("token"),
		keyLock:   sdk.NewKVStoreKey("lock"),
	}

	feeCollectorAcc := supply.NewEmptyModuleAccount(auth.FeeCollectorName)
	blacklistedAddrs := make(map[string]bool)
	blacklistedAddrs[feeCollectorAcc.String()] = true

	mockDexApp.bankKeeper = bank.NewBaseKeeper(
		mockDexApp.AccountKeeper,
		mockDexApp.ParamsKeeper.Subspace(bank.DefaultParamspace),
		blacklistedAddrs,
	)

	maccPerms := map[string][]string{
		auth.FeeCollectorName: nil,
		types.ModuleName:      nil,
	}
	mockDexApp.supplyKeeper = supply.NewKeeper(
		mockDexApp.Cdc.GetCdc(),
		mockDexApp.keySupply,
		mockDexApp.AccountKeeper,
		bank.NewBankKeeperAdapter(mockDexApp.bankKeeper),
		maccPerms)

	mockDexApp.tokenKeeper = NewKeeper(
		mockDexApp.bankKeeper,
		mockDexApp.ParamsKeeper.Subspace(DefaultParamspace),
		auth.FeeCollectorName,
		mockDexApp.supplyKeeper,
		mockDexApp.keyToken,
		mockDexApp.keyLock,
		mockDexApp.Cdc.GetCdc(),
		true, mockDexApp.AccountKeeper)

	// for staking/distr rollback to cosmos-sdk
	//store.NewKVStoreKey(staking.DelegatorPoolKey),
	//store.NewKVStoreKey(staking.RedelegationKeyM),
	//store.NewKVStoreKey(staking.RedelegationActonKey),
	//store.NewKVStoreKey(staking.UnbondingKey),

	handler := NewTokenHandler(mockDexApp.tokenKeeper, version.CurrentProtocolVersion)

	mockDexApp.Router().AddRoute(RouterKey, handler)
	mockDexApp.QueryRouter().AddRoute(QuerierRoute, NewQuerier(mockDexApp.tokenKeeper))

	mockDexApp.SetEndBlocker(getEndBlocker(mockDexApp.tokenKeeper))
	mockDexApp.SetInitChainer(getInitChainer(mockDexApp.App, mockDexApp.bankKeeper, mockDexApp.supplyKeeper, []exported.ModuleAccountI{feeCollectorAcc}))

	intQuantity := int64(10000000)
	valTokens := sdk.NewDec(intQuantity)
	coins := sdk.SysCoins{
		sdk.NewDecCoinFromDec(common.NativeToken, valTokens),
		sdk.NewDecCoinFromDec(common.TestToken, valTokens),
	}

	genAccs, _, _, _ := mock.CreateGenAccounts(numGenAccs, coins)

	// todo: checkTx in mock app
	mockDexApp.SetAnteHandler(nil)

	app := mockDexApp
	mockDexApp.MountStores(
		app.keyToken,
		app.keyLock,
		app.keySupply,
	)

	require.NoError(t, mockDexApp.CompleteSetup())
	mock.SetGenesis(mockDexApp.App, genAccs)
	//app.BaseApp.NewContext(true, abci.Header{})
	return mockDexApp, mockDexApp.tokenKeeper, handler
}

func getInitChainer(mapp *mock.App, bankKeeper bank.Keeper, supplyKeeper supply.Keeper,
	blacklistedAddrs []exported.ModuleAccountI) sdk.InitChainer {
	return func(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
		mapp.InitChainer(ctx, req)
		// set module accounts
		for _, macc := range blacklistedAddrs {
			supplyKeeper.SetModuleAccount(ctx, macc)
		}
		bankKeeper.SetSendEnabled(ctx, true)
		supplyKeeper.SetSupply(ctx, supply.NewSupply(sdk.Coins{}))
		return abci.ResponseInitChain{}
	}
}

func getTokenSymbol(ctx sdk.Context, keeper Keeper, prefix string) string {
	store := ctx.KVStore(keeper.tokenStoreKey)
	iter := sdk.KVStorePrefixIterator(store, types.TokenKey)
	defer iter.Close()
	for iter.Valid() {
		var token types.Token
		tokenBytes := iter.Value()
		keeper.cdc.MustUnmarshalBinaryBare(tokenBytes, &token)
		if strings.HasPrefix(token.Symbol, prefix) {
			return token.Symbol
		}
		iter.Next()
	}
	return ""
}

type testAccount struct {
	addrKeys    *mock.AddrKeys
	baseAccount types.DecAccount
}

func mockApplyBlock(t *testing.T, app *MockDexApp, txs []*auth.StdTx, height int64) sdk.Context {
	mockBlockHeight++
	app.BeginBlock(abci.RequestBeginBlock{Header: abci.Header{Height: height}})

	ctx := app.BaseApp.NewContext(false, abci.Header{})
	//ctx = ctx.WithTxBytes([]byte("90843555124EBF16EB13262400FB8CF639E6A772F437E37A0A141FE640A0B203"))
	param := types.DefaultParams()
	app.tokenKeeper.SetParams(ctx, param)
	for _, tx := range txs {
		app.Deliver(tx)
	}
	app.EndBlock(abci.RequestEndBlock{})
	app.Commit(abci.RequestCommit{})
	return ctx
}

func CreateGenAccounts(numAccs int, genCoins sdk.SysCoins) (genAccs []types.DecAccount, atList TestAccounts) {

	for i := 0; i < numAccs; i++ {
		privKey := secp256k1.GenPrivKey()
		pubKey := privKey.PubKey()
		addr := sdk.AccAddress(pubKey.Address())

		ak := mock.NewAddrKeys(addr, pubKey, privKey)
		testAccount := &testAccount{&ak,
			types.DecAccount{
				Address: addr,
				Coins:   genCoins},
		}
		atList = append(atList, testAccount)

		genAccs = append(genAccs, testAccount.baseAccount)
	}
	return
}

type TestAccounts []*testAccount

// GenTx generates a signed mock transaction.
func GenTx(msgs []sdk.Msg, accnums []uint64, seq []uint64, priv ...crypto.PrivKey) *auth.StdTx {
	// Make the transaction free
	fee := auth.StdFee{
		// just for test - 0.01okt as fixed fee
		Amount: sdk.NewDecCoinsFromDec(sdk.DefaultBondDenom, sdk.MustNewDecFromStr("0.01")),
		Gas:    200000,
	}

	sigs := make([]auth.StdSignature, len(priv))
	memo := "testmemotestmemo"

	for i, p := range priv {
		sig, err := p.Sign(auth.StdSignBytes("", accnums[i], seq[i], fee, msgs, memo))
		if err != nil {
			panic(err)
		}

		sigs[i] = auth.StdSignature{
			PubKey:    p.PubKey(),
			Signature: sig,
		}
	}

	return auth.NewStdTx(msgs, fee, sigs, memo)
}

func createTokenMsg(t *testing.T, app *MockDexApp, ctx sdk.Context, account *testAccount, tokenMsg sdk.Msg) *auth.StdTx {
	accs := app.AccountKeeper.GetAccount(ctx, account.baseAccount.Address)
	accNum := accs.GetAccountNumber()
	seqNum := accs.GetSequence()

	// todo:
	//tokenIssueMsg.Sender = account.addrKeys.Address
	tx := GenTx([]sdk.Msg{tokenMsg}, []uint64{accNum}, []uint64{seqNum}, account.addrKeys.PrivKey)
	app.Check(tx)
	//if !res.IsOK() {
	//	panic("something wrong in checking transaction")
	//}
	return tx
}

type MsgFaked struct {
	FakeID int
}

func (msg MsgFaked) Route() string { return "token" }

func (msg MsgFaked) Type() string { return "issue" }

func (msg MsgFaked) ValidateBasic() sdk.Error {
	return nil
}

func (msg MsgFaked) GetSignBytes() []byte {
	return sdk.MustSortJSON([]byte("1"))
}

func (msg MsgFaked) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{}
}

func newFakeMsg() MsgFaked {
	return MsgFaked{
		FakeID: 0,
	}
}

func TestMsgTokenChown(t *testing.T) {
	//change token owner
	intQuantity := int64(30000)
	// to
	toPriKey := secp256k1.GenPrivKey()
	toPubKey := toPriKey.PubKey()
	toAddr := sdk.AccAddress(toPubKey.Address())
	//init accounts
	genAccs, testAccounts := CreateGenAccounts(1,
		sdk.SysCoins{
			sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(intQuantity)),
		},
	)

	fromAddr := testAccounts[0].addrKeys.Address
	//gen app and keepper
	app, keeper, handler := getMockDexAppEx(t, 0)
	mock.SetGenesis(app.App, types.DecAccountArrToBaseAccountArr(genAccs))

	//build context
	ctx := app.BaseApp.NewContext(true, abci.Header{})
	ctx.SetTxBytes([]byte("90843555124EBF16EB13262400FB8CF639E6A772F437E37A0A141FE640A0B203"))
	var TokenChown []*auth.StdTx
	var TokenIssue []*auth.StdTx

	//test fake message
	if handler != nil {
		handler(ctx, newFakeMsg())
	}

	//issue token to FromAddress
	tokenIssueMsg := types.NewMsgTokenIssue(common.NativeToken, common.NativeToken, common.NativeToken, "okcoin", "1000", testAccounts[0].baseAccount.Address, true)
	TokenIssue = append(TokenIssue, createTokenMsg(t, app, ctx, testAccounts[0], tokenIssueMsg))

	//test error supply coin issue(TotalSupply > (9*1e10))
	MsgErrorSupply := types.NewMsgTokenIssue("okc", "okc", "okc", "okccc", strconv.FormatInt(int64(10*1e10), 10), testAccounts[0].baseAccount.Address, true)
	TokenIssue = append(TokenIssue, createTokenMsg(t, app, ctx, testAccounts[0], MsgErrorSupply))

	//test error tokenDesc (length > 256)
	MsgErrorName := types.NewMsgTokenIssue(`ok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-b
ok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-b
ok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-b
ok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-bok-b`,
		common.NativeToken, common.NativeToken, "okcoin", "2100", testAccounts[0].baseAccount.Address, true)
	TokenIssue = append(TokenIssue, createTokenMsg(t, app, ctx, testAccounts[0], MsgErrorName))

	ctx = mockApplyBlock(t, app, TokenIssue, 3)
	_, err := handleMsgTokenIssue(ctx, keeper, MsgErrorSupply, nil)
	require.NotNil(t, err)
	//require.NotNil(t, handleMsgTokenIssue(ctx, keeper, MsgErrorName, nil))

	//test if zzb is not exist
	invalidmsg := types.NewMsgTransferOwnership(fromAddr, toAddr, "zzb")
	TokenChown = append(TokenChown, createTokenMsg(t, app, ctx, testAccounts[0], invalidmsg))

	//test addTokenSuffix->ValidSymbol
	addTokenSuffix(ctx, keeper, "notexist")

	//normal test
	symbName := "okb-b85" //addTokenSuffix(ctx,keeper,common.NativeToken)
	//change owner from F to T
	tokenChownMsg := types.NewMsgTransferOwnership(fromAddr, toAddr, symbName)
	TokenChown = append(TokenChown, createTokenMsg(t, app, ctx, testAccounts[0], tokenChownMsg))

	ctx = mockApplyBlock(t, app, TokenChown, 4)
}

func TestHandleMsgTokenIssueFails(t *testing.T) {
	var TokenIssue []*auth.StdTx
	genAccs, testAccounts := CreateGenAccounts(1,
		sdk.SysCoins{
			sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(30000)),
		},
	)
	app, keeper, _ := getMockDexAppEx(t, 0)
	mock.SetGenesis(app.App, types.DecAccountArrToBaseAccountArr(genAccs))

	//build context
	ctx := mockApplyBlock(t, app, TokenIssue, 3)
	cases := []struct {
		info     string
		msg      types.MsgTokenIssue
		expected string
		panic    bool
	}{
		{
			"Error Get Decimal From Decimal String",
			types.NewMsgTokenIssue("", common.NativeToken, common.NativeToken, "okcoin", "", testAccounts[0].baseAccount.Address, true),
			"create a decimal from an input decimal string failed: create a decimal from an input decimal string failed: decimal string cannot be empty",
			false,
		},
		{
			"Error Invalid Coins",
			types.NewMsgTokenIssue("", common.NativeToken, "a.b", "okcoin", "9999", testAccounts[0].baseAccount.Address, true),
			"invalid coins: invalid coins: a.b",
			false,
		},
		{
			"Error Mint Coins Failed",
			types.NewMsgTokenIssue("", common.NativeToken, common.NativeToken, "okcoin", "9999", testAccounts[0].baseAccount.Address, true),
			"not have permission to mint should panic",
			true,
		},
	}
	for _, tc := range cases {

		if tc.panic {
			require.Panics(t, func() { handleMsgTokenIssue(ctx, keeper, tc.msg, nil) })
		} else {
			_, err := handleMsgTokenIssue(ctx, keeper, tc.msg, nil)
			require.Equal(t, err.Error(), tc.expected)
		}
	}
}

func TestUpdateUserTokenRelationship(t *testing.T) {
	intQuantity := int64(30000)
	genAccs, testAccounts := CreateGenAccounts(2,
		sdk.SysCoins{
			sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(intQuantity)),
		})

	app, keeper, _ := getMockDexApp(t, 0)
	mock.SetGenesis(app.App, types.DecAccountArrToBaseAccountArr(genAccs))

	ctx := app.BaseApp.NewContext(true, abci.Header{})
	ctx.SetTxBytes([]byte("90843555124EBF16EB13262400FB8CF639E6A772F437E37A0A141FE640A0B203"))

	var tokenIssue []*auth.StdTx

	totalSupplyStr := "500"
	tokenIssueMsg := types.NewMsgTokenIssue("bnb", "", "bnb", "binance coin", totalSupplyStr, testAccounts[0].baseAccount.Address, true)
	tokenIssue = append(tokenIssue, createTokenMsg(t, app, ctx, testAccounts[0], tokenIssueMsg))

	ctx = mockApplyBlock(t, app, tokenIssue, 3)

	tokens := keeper.GetUserTokensInfo(ctx, testAccounts[0].baseAccount.Address)
	require.EqualValues(t, 1, len(tokens))

	tokenName := getTokenSymbol(ctx, keeper, "bnb")
	// ===============

	var TokenChown []*auth.StdTx

	//test if zzb is not exist
	chownMsg := types.NewMsgTransferOwnership(testAccounts[0].baseAccount.Address, testAccounts[1].baseAccount.Address, tokenName)
	TokenChown = append(TokenChown, createTokenMsg(t, app, ctx, testAccounts[0], chownMsg))

	ctx = mockApplyBlock(t, app, TokenChown, 4)

	tokens = keeper.GetUserTokensInfo(ctx, testAccounts[0].baseAccount.Address)
	require.EqualValues(t, 1, len(tokens))

	confirmMsg := types.NewMsgConfirmOwnership(testAccounts[1].baseAccount.Address, tokenName)
	ctx = mockApplyBlock(t, app, []*auth.StdTx{createTokenMsg(t, app, ctx, testAccounts[1], confirmMsg)}, 5)

	tokens = keeper.GetUserTokensInfo(ctx, testAccounts[0].baseAccount.Address)
	require.EqualValues(t, 0, len(tokens))

	tokens = keeper.GetUserTokensInfo(ctx, testAccounts[1].baseAccount.Address)
	require.EqualValues(t, 1, len(tokens))
}

func TestCreateTokenIssue(t *testing.T) {
	intQuantity := int64(3000)
	genAccs, testAccounts := CreateGenAccounts(1,
		sdk.SysCoins{
			sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(intQuantity)),
		})

	app, keeper, _ := getMockDexApp(t, 0)
	mock.SetGenesis(app.App, types.DecAccountArrToBaseAccountArr(genAccs))

	ctx := app.BaseApp.NewContext(true, abci.Header{})
	ctx.SetTxBytes([]byte("90843555124EBF16EB13262400FB8CF639E6A772F437E37A0A141FE640A0B203"))

	var tokenIssue []*auth.StdTx

	totalSupply := int64(500)
	totalSupplyStr := "500"
	tokenIssueMsg := types.NewMsgTokenIssue("bnb", "", "bnb", "binance coin", totalSupplyStr, testAccounts[0].baseAccount.Address, true)
	tokenIssue = append(tokenIssue, createTokenMsg(t, app, ctx, testAccounts[0], tokenIssueMsg))

	// not valid symbol
	//tokenIssueMsg = types.NewMsgTokenIssue("bnba123451fadfasdf", "bnba123451fadfasdf", "bnba123451fadfasdf", totalSupply, testAccounts[0].baseAccount.Address, true)
	//tokenIssue = append(tokenIssue, createTokenMsg(t, app, ctx, testAccounts[0], tokenIssueMsg))

	// Total exceeds the upper limit
	tokenIssueMsg = types.NewMsgTokenIssue("btc", "btc", "btc", "bitcoin", strconv.FormatInt(types.TotalSupplyUpperbound+1, 10), testAccounts[0].baseAccount.Address, true)
	tokenIssue = append(tokenIssue, createTokenMsg(t, app, ctx, testAccounts[0], tokenIssueMsg))

	// not enough okbs
	tokenIssueMsg = types.NewMsgTokenIssue("xmr", "xmr", "xmr", "Monero", totalSupplyStr, testAccounts[0].baseAccount.Address, true)
	tokenIssue = append(tokenIssue, createTokenMsg(t, app, ctx, testAccounts[0], tokenIssueMsg))

	ctx = mockApplyBlock(t, app, tokenIssue, 3)

	tokenName := getTokenSymbol(ctx, keeper, "bnb")
	//feeIssue, err := sdk.NewDecFromStr(DefaultFeeIssue)
	//require.EqualValues(t, nil, err)
	feeIssue := keeper.GetParams(ctx).FeeIssue.Amount
	coins := sdk.SysCoins{
		sdk.NewDecCoinFromDec(tokenName, sdk.NewDec(totalSupply)),
		sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(intQuantity).Sub(feeIssue)),
	}
	require.EqualValues(t, coins, app.AccountKeeper.GetAccount(ctx, testAccounts[0].addrKeys.Address).GetCoins())
	tokenStoreKeyNum, lockStoreKeyNum := keeper.getNumKeys(ctx)
	require.Equal(t, int64(3), tokenStoreKeyNum)
	require.Equal(t, int64(0), lockStoreKeyNum)

	tokenInfo := keeper.GetTokenInfo(ctx, tokenName)
	require.EqualValues(t, sdk.MustNewDecFromStr("500"), tokenInfo.OriginalTotalSupply)
}

func TestCreateTokenBurn(t *testing.T) {
	intQuantity := int64(2511)

	genAccs, testAccounts := CreateGenAccounts(1,
		sdk.SysCoins{
			sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(intQuantity)),
		})

	_, testAccounts2 := CreateGenAccounts(1,
		sdk.SysCoins{
			sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(intQuantity)),
		})

	app, keeper, _ := getMockDexApp(t, 0)
	mock.SetGenesis(app.App, types.DecAccountArrToBaseAccountArr(genAccs))

	ctx := app.NewContext(true, abci.Header{})
	var tokenMsgs []*auth.StdTx

	tokenIssueMsg := types.NewMsgTokenIssue("btc", "btc", "btc", "bitcoin", "1000", testAccounts[0].baseAccount.Address, true)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenIssueMsg))
	ctx = mockApplyBlock(t, app, tokenMsgs, 3)

	tokenMsgs = tokenMsgs[:0]

	burnNum := "100"
	//mockToken(app.tokenKeeper, ctx, testAccounts[0].baseAccount.Address, intQuantity)

	_, err := sdk.ParseDecCoin("-10000btc")
	require.Error(t, err)

	decCoin, err := sdk.ParseDecCoin("10000btc")
	require.Nil(t, err)
	// total exceeds the upper limit
	tokenBurnMsg := types.NewMsgTokenBurn(decCoin, testAccounts[0].baseAccount.Address)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenBurnMsg))
	//mockApplyBlock(t, app, tokenMsgs)

	// not the token's owner
	tokenBurnMsg = types.NewMsgTokenBurn(decCoin, testAccounts2[0].baseAccount.Address)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenBurnMsg))

	tokenSymbol := getTokenSymbol(ctx, keeper, "btc")

	decCoin, err = sdk.ParseDecCoin(burnNum + tokenSymbol)
	require.Nil(t, err)
	// normal case
	tokenBurnMsg = types.NewMsgTokenBurn(decCoin, testAccounts[0].baseAccount.Address)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenBurnMsg))

	decCoin, err = sdk.ParseDecCoin(burnNum + "btc")
	require.Nil(t, err)
	// not enough fees
	tokenBurnMsg = types.NewMsgTokenBurn(decCoin, testAccounts[0].baseAccount.Address)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenBurnMsg))

	ctx = mockApplyBlock(t, app, tokenMsgs, 4)

	//fee, err := sdk.NewDecFromStr("0.0125")
	fee, err := sdk.NewDecFromStr("0.0")
	require.Nil(t, err)
	validTxNum := sdk.NewDec(2)
	coins := sdk.SysCoins{
		sdk.NewDecCoinFromDec(tokenSymbol, sdk.NewDec(900)),
		sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(1).Add(fee.Mul(validTxNum))),
	}

	require.EqualValues(t, coins, app.AccountKeeper.GetAccount(ctx, testAccounts[0].addrKeys.Address).GetCoins())
}

func TestCreateTokenMint(t *testing.T) {
	intQuantity := int64(5011)

	genAccs, testAccounts := CreateGenAccounts(1,
		sdk.SysCoins{
			sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(intQuantity)),
		})

	_, testAccounts2 := CreateGenAccounts(1,
		sdk.SysCoins{
			sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(intQuantity)),
		})

	app, keeper, _ := getMockDexApp(t, 0)
	mock.SetGenesis(app.App, types.DecAccountArrToBaseAccountArr(genAccs))

	ctx := app.NewContext(true, abci.Header{})
	var tokenMsgs []*auth.StdTx

	tokenIssueMsg := types.NewMsgTokenIssue("btc", "btc", "btc", "bitcoin", "1000", testAccounts[0].baseAccount.Address, true)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenIssueMsg))
	ctx = mockApplyBlock(t, app, tokenMsgs, 3)
	tokenMsgs = tokenMsgs[:0]

	tokenIssueMsg = types.NewMsgTokenIssue("xmr", "xmr", "xmr", "monero", "1000", testAccounts[0].baseAccount.Address, false)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenIssueMsg))
	ctx = mockApplyBlock(t, app, tokenMsgs, 4)

	var mintNum int64 = 1000
	// normal case
	btcTokenSymbol := getTokenSymbol(ctx, keeper, "btc")
	decCoin := sdk.NewDecCoinFromDec(btcTokenSymbol, sdk.NewDec(mintNum))
	tokenMintMsg := types.NewMsgTokenMint(decCoin, testAccounts[0].baseAccount.Address)

	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenMintMsg))

	// Total exceeds the upper limit
	decCoin.Amount = sdk.NewDec(types.TotalSupplyUpperbound)
	tokenMintMsg = types.NewMsgTokenMint(decCoin, testAccounts[0].baseAccount.Address)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenMintMsg))

	// not the token's owner
	decCoin.Amount = sdk.NewDec(mintNum)
	tokenMintMsg = types.NewMsgTokenMint(decCoin, testAccounts2[0].baseAccount.Address)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenMintMsg))

	// token not mintable
	xmrTokenSymbol := getTokenSymbol(ctx, keeper, "xmr")
	decCoin.Denom = xmrTokenSymbol
	decCoin.Amount = sdk.NewDec(mintNum)
	tokenMintMsg = types.NewMsgTokenMint(decCoin, testAccounts[0].baseAccount.Address)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenMintMsg))

	// not enough fees
	decCoin.Denom = btcTokenSymbol
	decCoin.Amount = sdk.NewDec(mintNum)
	tokenMintMsg = types.NewMsgTokenMint(decCoin, testAccounts[0].baseAccount.Address)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenMintMsg))

	ctx = mockApplyBlock(t, app, tokenMsgs, 5)

	//validTxNum := sdk.NewInt(2)
	coins := sdk.MustParseCoins(btcTokenSymbol, "2000")
	//coins = append(coins, sdk.MustParseCoins(common.NativeToken, "1.0375")...)
	coins = append(coins, sdk.MustParseCoins(common.NativeToken, "1.0")...)
	coins = append(coins, sdk.MustParseCoins(xmrTokenSymbol, "1000")...)

	require.EqualValues(t, coins, app.AccountKeeper.GetAccount(ctx, testAccounts[0].addrKeys.Address).GetCoins())
}

func TestCreateMsgTokenSend(t *testing.T) {
	intQuantity := int64(100000)

	genAccs, testAccounts := CreateGenAccounts(2,
		sdk.SysCoins{
			sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(intQuantity)),
		})

	app, keeper, _ := getMockDexApp(t, 0)
	mock.SetGenesis(app.App, types.DecAccountArrToBaseAccountArr(genAccs))

	ctx := app.NewContext(true, abci.Header{})
	var tokenMsgs []*auth.StdTx

	tokenIssueMsg := types.NewMsgTokenIssue("btc", "btc", "btc", "bitcoin", "1000", testAccounts[0].baseAccount.Address, true)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenIssueMsg))
	ctx = mockApplyBlock(t, app, tokenMsgs, 3)
	tokenMsgs = tokenMsgs[:0]

	tokenName := getTokenSymbol(ctx, keeper, "btc")
	coins := sdk.SysCoins{
		sdk.NewDecCoinFromDec(tokenName, sdk.NewDec(100)),
	}
	tokenSendMsg := types.NewMsgTokenSend(testAccounts[0].baseAccount.Address, testAccounts[1].baseAccount.Address, coins)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenSendMsg))

	coins = sdk.SysCoins{
		sdk.NewDecCoinFromDec("btc", sdk.NewDec(10000)),
	}
	// not enough coins
	tokenSendMsg = types.NewMsgTokenSend(testAccounts[0].baseAccount.Address, testAccounts[1].baseAccount.Address, coins)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenSendMsg))

	ctx = mockApplyBlock(t, app, tokenMsgs, 4)

	accounts := app.AccountKeeper.GetAllAccounts(ctx)
	for _, acc := range accounts {
		if acc.GetAddress().Equals(testAccounts[0].baseAccount.Address) {
			senderCoins := sdk.MustParseCoins(tokenName, "900")
			senderCoins = append(senderCoins, sdk.MustParseCoins(common.NativeToken, "97500")...)
			require.EqualValues(t, senderCoins, acc.GetCoins())
		} else if acc.GetAddress().Equals(testAccounts[1].baseAccount.Address) {
			receiverCoins := sdk.MustParseCoins(tokenName, "100")
			receiverCoins = append(receiverCoins, sdk.MustParseCoins(common.NativeToken, "100000")...)
			require.EqualValues(t, receiverCoins, acc.GetCoins())
		}
	}

	// len(MsgTokenSend.Amount) > 1
	tokenMsgs = tokenMsgs[:0]
	coins = sdk.SysCoins{
		sdk.NewDecCoinFromDec(tokenName, sdk.NewDec(100)),
		sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(100)),
	}
	tokenSendMsg = types.NewMsgTokenSend(testAccounts[0].baseAccount.Address, testAccounts[1].baseAccount.Address, coins)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenSendMsg))

	ctx = mockApplyBlock(t, app, tokenMsgs, 5)

	accounts = app.AccountKeeper.GetAllAccounts(ctx)
	for _, acc := range accounts {
		if acc.GetAddress().Equals(testAccounts[0].baseAccount.Address) {
			senderCoins := sdk.MustParseCoins(tokenName, "800")
			senderCoins = append(senderCoins, sdk.MustParseCoins(common.NativeToken, "97400")...)
			require.EqualValues(t, senderCoins.String(), acc.GetCoins().String())
		} else if acc.GetAddress().Equals(testAccounts[1].baseAccount.Address) {
			receiverCoins := sdk.MustParseCoins(tokenName, "200")
			receiverCoins = append(receiverCoins, sdk.MustParseCoins(common.NativeToken, "100100")...)
			require.EqualValues(t, receiverCoins, acc.GetCoins())
		}
	}
}

func TestCreateMsgMultiSend(t *testing.T) {
	intQuantity := int64(100000)

	genAccs, testAccounts := CreateGenAccounts(2,
		sdk.SysCoins{
			sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(intQuantity)),
		})

	app, keeper, _ := getMockDexApp(t, 0)
	mock.SetGenesis(app.App, types.DecAccountArrToBaseAccountArr(genAccs))
	// set anteHandler for mock app
	app.PushAnteHandler(auth.NewAnteHandler(
		app.AccountKeeper,
		app.supplyKeeper,
		auth.DefaultSigVerificationGasConsumer,
	))

	ctx := app.NewContext(true, abci.Header{})
	var tokenMsgs []*auth.StdTx

	tokenIssueMsg := types.NewMsgTokenIssue("btc", "btc", "btc", "bitcoin", "1000", testAccounts[0].baseAccount.Address, true)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenIssueMsg))
	ctx = mockApplyBlock(t, app, tokenMsgs, 3)

	tokenMsgs = tokenMsgs[:0]
	btcSymbol := getTokenSymbol(ctx, keeper, "btc")
	multiSendStr := `[{"to":"` + testAccounts[1].baseAccount.Address.String() + `","amount":"1` + common.NativeToken + `,2` + btcSymbol + `"}]`
	transfers, err := types.StrToTransfers(multiSendStr)
	require.NoError(t, err)
	multiSend := types.NewMsgMultiSend(testAccounts[0].baseAccount.Address, transfers)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], multiSend))
	ctx = mockApplyBlock(t, app, tokenMsgs, 4)

	// insufficient coins for multi-send
	multiSendStr = `[{"to":"` + testAccounts[1].baseAccount.Address.String() + `","amount":"1` + common.NativeToken + `,2000` + btcSymbol + `"}]`
	transfers, err = types.StrToTransfers(multiSendStr)
	require.NoError(t, err)
	multiSend = types.NewMsgMultiSend(testAccounts[0].baseAccount.Address, transfers)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], multiSend))
	ctx = mockApplyBlock(t, app, tokenMsgs, 5)

	accounts := app.AccountKeeper.GetAllAccounts(ctx)
	for _, acc := range accounts {
		if acc.GetAddress().Equals(testAccounts[0].baseAccount.Address) {
			senderCoins := sdk.MustParseCoins(btcSymbol, "998")
			senderCoins = append(senderCoins, sdk.MustParseCoins(common.NativeToken, "97498.97000000")...)
			require.EqualValues(t, senderCoins, acc.GetCoins())
		} else if acc.GetAddress().Equals(testAccounts[1].baseAccount.Address) {
			receiverCoins := sdk.MustParseCoins(btcSymbol, "2")
			receiverCoins = append(receiverCoins, sdk.MustParseCoins(common.NativeToken, "100001")...)
			require.EqualValues(t, receiverCoins, acc.GetCoins())
		}
	}
}

func TestCreateMsgTokenModify(t *testing.T) {
	intQuantity := int64(100000)

	genAccs, testAccounts := CreateGenAccounts(2,
		sdk.SysCoins{
			sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(intQuantity)),
		})

	app, keeper, _ := getMockDexApp(t, 0)
	mock.SetGenesis(app.App, types.DecAccountArrToBaseAccountArr(genAccs))

	ctx := app.NewContext(true, abci.Header{})
	var tokenMsgs []*auth.StdTx

	tokenIssueMsg := types.NewMsgTokenIssue("btc", "btc", "btc", "bitcoin", "1000", testAccounts[0].baseAccount.Address, true)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenIssueMsg))
	ctx = mockApplyBlock(t, app, tokenMsgs, 3)

	tokenMsgs = tokenMsgs[:0]
	btcTokenSymbol := getTokenSymbol(ctx, keeper, "btc")

	// normal case
	tokenEditMsg := types.NewMsgTokenModify(btcTokenSymbol, "desc0", "whole name0", true, true, testAccounts[0].baseAccount.Address)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenEditMsg))
	ctx = mockApplyBlock(t, app, tokenMsgs, 4)
	token := keeper.GetTokenInfo(ctx, btcTokenSymbol)
	require.EqualValues(t, "desc0", token.Description)
	require.EqualValues(t, "whole name0", token.WholeName)

	tokenMsgs = tokenMsgs[:0]
	tokenEditMsg = types.NewMsgTokenModify(btcTokenSymbol, "desc1", "whole name1", false, true, testAccounts[0].baseAccount.Address)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenEditMsg))
	ctx = mockApplyBlock(t, app, tokenMsgs, 5)
	token = keeper.GetTokenInfo(ctx, btcTokenSymbol)
	require.EqualValues(t, "desc0", token.Description)
	require.EqualValues(t, "whole name1", token.WholeName)

	tokenMsgs = tokenMsgs[:0]
	tokenEditMsg = types.NewMsgTokenModify(btcTokenSymbol, "desc2", "whole name2", true, false, testAccounts[0].baseAccount.Address)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenEditMsg))
	ctx = mockApplyBlock(t, app, tokenMsgs, 6)
	token = keeper.GetTokenInfo(ctx, btcTokenSymbol)
	require.EqualValues(t, "desc2", token.Description)
	require.EqualValues(t, "whole name1", token.WholeName)

	tokenMsgs = tokenMsgs[:0]
	tokenEditMsg = types.NewMsgTokenModify(btcTokenSymbol, "desc3", "whole name2", false, false, testAccounts[0].baseAccount.Address)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenEditMsg))
	ctx = mockApplyBlock(t, app, tokenMsgs, 7)
	token = keeper.GetTokenInfo(ctx, btcTokenSymbol)
	require.EqualValues(t, "desc2", token.Description)
	require.EqualValues(t, "whole name1", token.WholeName)

	// error case
	tokenMsgs = tokenMsgs[:0]
	tokenEditMsg = types.NewMsgTokenModify("btcTokenSymbol", "desc4", "whole name4", true, true, testAccounts[0].baseAccount.Address)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenEditMsg))
	ctx = mockApplyBlock(t, app, tokenMsgs, 8)

	tokenMsgs = tokenMsgs[:0]
	tokenEditMsg = types.NewMsgTokenModify(btcTokenSymbol, "desc5", "whole name5", true, true, testAccounts[1].baseAccount.Address)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenEditMsg))
	ctx = mockApplyBlock(t, app, tokenMsgs, 9)

	tokenMsgs = tokenMsgs[:0]
	tokenEditMsg = types.NewMsgTokenModify(btcTokenSymbol, "desc6", "whole nasiangrueinvowfoij;oeasifnroeinagoirengodd   me6", true, true, testAccounts[0].baseAccount.Address)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenEditMsg))
	ctx = mockApplyBlock(t, app, tokenMsgs, 10)

	tokenMsgs = tokenMsgs[:0]
	tokenEditMsg = types.NewMsgTokenModify(btcTokenSymbol, `bnbbbbbbbbbbbnbbbbbbbbbbnbbbbbbbbbbbnbbbbbbbbb1234
bnbbbbbbbbbbbnbbbbbbbbbbnbbbbbbbbbbbnbbbbbbbbb1234
bnbbbbbbbbbbbnbbbbbbbbbbnbbbbbbbbbbbnbbbbbbbbb1234
bnbbbbbbbbbbbnbbbbbbbbbbnbbbbbbbbbbbnbbbbbbbbb1234
bnbbbbbbbbbbbnbbbbbbbbbbnbbbbbbbbbbbnbbbbbbbbb1234
bnbbbbbbbbbbbnbbbbbbbbbbnbbbbbbbbbbbnbbbbbbbbb1234`, "whole name7", true, true, testAccounts[0].baseAccount.Address)
	tokenMsgs = append(tokenMsgs, createTokenMsg(t, app, ctx, testAccounts[0], tokenEditMsg))
	ctx = mockApplyBlock(t, app, tokenMsgs, 11)

	token = keeper.GetTokenInfo(ctx, btcTokenSymbol)
	require.EqualValues(t, "desc2", token.Description)
	require.EqualValues(t, "whole name1", token.WholeName)
}

func getMockAppToHandleFee(t *testing.T, initBalance int64, numAcc int) (app *MockDexApp, testAccounts TestAccounts) {
	intQuantity := int64(initBalance)
	genAccs, testAccounts := CreateGenAccounts(numAcc,
		sdk.SysCoins{
			sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(intQuantity)),
		})

	app, _, _ = getMockDexApp(t, 0)
	mock.SetGenesis(app.App, types.DecAccountArrToBaseAccountArr(genAccs))
	app.PushAnteHandler(auth.NewAnteHandler(
		app.AccountKeeper,
		app.supplyKeeper,
		auth.DefaultSigVerificationGasConsumer,
	))

	return app, testAccounts

}

func TestTxFailedFeeTable(t *testing.T) {

	app, testAccounts := getMockAppToHandleFee(t, 10, 1)
	ctx := app.BaseApp.NewContext(true, abci.Header{})

	// to
	toPriKey := secp256k1.GenPrivKey()
	toPubKey := toPriKey.PubKey()
	toAddr := sdk.AccAddress(toPubKey.Address())

	// failed issue msg : not enough okbs .
	failedIssueMsg := types.NewMsgTokenIssue("xmr", "xmr", "xmr", "Monero", "500", testAccounts[0].baseAccount.Address, true)
	// failed mint msg : no such token
	decCoin := sdk.NewDecCoinFromDec("nob", sdk.NewDec(200))
	failedMintMsg := types.NewMsgTokenMint(decCoin, testAccounts[0].baseAccount.Address)
	// failed burn msg : no such token
	failedBurnMsg := types.NewMsgTokenBurn(decCoin, testAccounts[0].baseAccount.Address)
	// failed edit msg : no such token
	failedEditMsg := types.NewMsgTokenModify("nob", "desc0", "whole name0", true, true, testAccounts[0].baseAccount.Address)
	// failed send msg: no such token
	fialedSendMsg := types.NewMsgTokenSend(testAccounts[0].baseAccount.Address, toAddr, sdk.SysCoins{decCoin})

	// failed MultiSend msg: no such token
	multiSendStr := `[{"to":"` + toAddr.String() + `","amount":"1` + common.NativeToken + `,2` + "nob" + `"}]`
	transfers, err := types.StrToTransfers(multiSendStr)
	require.Nil(t, err)
	failedMultiSendMsg := types.NewMsgMultiSend(testAccounts[0].baseAccount.Address, transfers)

	// failed TransferOwnership msg: no such token
	failedChownMsg := types.NewMsgTransferOwnership(testAccounts[0].baseAccount.Address, toAddr, "nob")

	failTestSets := []struct {
		name    string
		balance string
		msg     *auth.StdTx
	}{
		// 0.01okt as fixed fee in each stdTx
		{"fail to issue : 0.01", "9.990000000000000000", createTokenMsg(t, app, ctx, testAccounts[0], failedIssueMsg)},
		{"fail to mint  : 0.01", "9.980000000000000000", createTokenMsg(t, app, ctx, testAccounts[0], failedMintMsg)},
		{"fail to burn  : 0.01", "9.970000000000000000", createTokenMsg(t, app, ctx, testAccounts[0], failedBurnMsg)},
		{"fail to modify: 0.01", "9.960000000000000000", createTokenMsg(t, app, ctx, testAccounts[0], failedEditMsg)},
		{"fail to send  : 0.01", "9.950000000000000000", createTokenMsg(t, app, ctx, testAccounts[0], fialedSendMsg)},
		{"fail to multi : 0.01", "9.940000000000000000", createTokenMsg(t, app, ctx, testAccounts[0], failedMultiSendMsg)},
		{"fail to chown : 0.01", "9.930000000000000000", createTokenMsg(t, app, ctx, testAccounts[0], failedChownMsg)},
	}
	for i, tt := range failTestSets {
		t.Run(tt.name, func(t *testing.T) {
			ctx = mockApplyBlock(t, app, []*auth.StdTx{tt.msg}, int64(i+3))
			require.Equal(t, tt.balance, app.AccountKeeper.GetAccount(ctx, testAccounts[0].addrKeys.Address).GetCoins().AmountOf(common.NativeToken).String())
		})
	}

}

func TestTxSuccessFeeTable(t *testing.T) {
	app, testAccounts := getMockAppToHandleFee(t, 30000, 1)
	ctx := app.BaseApp.NewContext(true, abci.Header{})
	// to
	toPriKey := secp256k1.GenPrivKey()
	toPubKey := toPriKey.PubKey()
	toAddr := sdk.AccAddress(toPubKey.Address())

	// successful issue msg
	successfulIssueMsg := types.NewMsgTokenIssue("xxb", "xxb", "xxb", "xx coin", "500", testAccounts[0].baseAccount.Address, true)

	symbolAfterIssue, ok := addTokenSuffix(ctx, app.tokenKeeper, "xxb")
	require.True(t, ok)

	decCoin := sdk.NewDecCoinFromDec(symbolAfterIssue, sdk.NewDec(50))
	successfulMintMsg := types.NewMsgTokenMint(decCoin, testAccounts[0].baseAccount.Address)

	successfulBurnMsg := types.NewMsgTokenBurn(decCoin, testAccounts[0].baseAccount.Address)

	successfulSendMsg := types.NewMsgTokenSend(testAccounts[0].baseAccount.Address, toAddr, sdk.SysCoins{decCoin})

	// multi send
	multiSendStr := `[{"to":"` + toAddr.String() + `","amount":" 10` + common.NativeToken + `,20` + symbolAfterIssue + `"}]`
	transfers, err := types.StrToTransfers(multiSendStr)
	require.Nil(t, err)
	successfulMultiSendMsg := types.NewMsgMultiSend(testAccounts[0].baseAccount.Address, transfers)

	successfulEditMsg := types.NewMsgTokenModify(symbolAfterIssue, "edit msg", "xxb coin ", true, true, testAccounts[0].baseAccount.Address)

	successfulChownMsg := types.NewMsgTransferOwnership(testAccounts[0].baseAccount.Address, toAddr, symbolAfterIssue)

	successfulTestSets := []struct {
		description string
		balance     string
		msg         sdk.Msg
		account     *testAccount
	}{
		// 0.01okt as fixed fee in each stdTx
		{"success to issue : 2500+0.01", "27499.990000000000000000", successfulIssueMsg, testAccounts[0]},
		{"success to mint  : 10+0.01", "27489.980000000000000000", successfulMintMsg, testAccounts[0]},
		{"success to burn  : 10+0.01", "27479.970000000000000000", successfulBurnMsg, testAccounts[0]},
		{"success to send  : 0.01", "27479.960000000000000000", successfulSendMsg, testAccounts[0]},
		{"success to multi : 10(amount of transfer) +0.01", "27469.950000000000000000", successfulMultiSendMsg, testAccounts[0]},
		{"success to modify: 0.01", "27469.940000000000000000", successfulEditMsg, testAccounts[0]},
		{"success to chown : 10+0.01", "27459.930000000000000000", successfulChownMsg, testAccounts[0]},
	}
	for i, tt := range successfulTestSets {
		t.Run(tt.description, func(t *testing.T) {
			stdTx := createTokenMsg(t, app, ctx, tt.account, tt.msg)
			ctx = mockApplyBlock(t, app, []*auth.StdTx{stdTx}, int64(i+3))
			require.Equal(t, tt.balance, app.AccountKeeper.GetAccount(ctx, testAccounts[0].addrKeys.Address).GetCoins().AmountOf(common.NativeToken).String())
		})
	}
}

func TestBlockedAddrSend(t *testing.T) {
	app, testAccounts := getMockAppToHandleFee(t, 30000, 1)
	ctx := app.BaseApp.NewContext(true, abci.Header{})
	// blocked addr
	blockedAddr := supply.NewModuleAddress(auth.FeeCollectorName)
	// unblocked addr
	toPriKey := secp256k1.GenPrivKey()
	toPubKey := toPriKey.PubKey()
	validAddr := sdk.AccAddress(toPubKey.Address())

	// build send msg
	decCoin := sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(50))
	successfulSendMsg := types.NewMsgTokenSend(testAccounts[0].baseAccount.Address, validAddr, sdk.SysCoins{decCoin})
	failedSendMsg := types.NewMsgTokenSend(testAccounts[0].baseAccount.Address, blockedAddr, sdk.SysCoins{decCoin})

	// build multi-send msg
	multiSendStr := `[{"to":"` + validAddr.String() + `","amount":" 100` + common.NativeToken + `"}]`
	transfers, err := types.StrToTransfers(multiSendStr)
	require.NoError(t, err)
	successfulMultiSendMsg := types.NewMsgMultiSend(testAccounts[0].baseAccount.Address, transfers)

	multiSendStr = `[{"to":"` + blockedAddr.String() + `","amount":" 100` + common.NativeToken + `"}]`
	transfers, err = types.StrToTransfers(multiSendStr)
	require.NoError(t, err)
	failedMultiSendMsg := types.NewMsgMultiSend(testAccounts[0].baseAccount.Address, transfers)

	successfulTestSets := []struct {
		description string
		balance     string
		msg         sdk.Msg
		account     *testAccount
	}{
		// 0.01okt as fixed fee in each stdTx
		{"success to send  : 50+0.01", "29949.990000000000000000", successfulSendMsg, testAccounts[0]},
		{"fail to send  : 0.01", "29949.980000000000000000", failedSendMsg, testAccounts[0]},
		{"success to multi-send  : 100+0.01", "29849.970000000000000000", successfulMultiSendMsg, testAccounts[0]},
		{"fail to multi-send  : 0.01", "29849.960000000000000000", failedMultiSendMsg, testAccounts[0]},
	}
	for i, tt := range successfulTestSets {
		t.Run(tt.description, func(t *testing.T) {
			stdTx := createTokenMsg(t, app, ctx, tt.account, tt.msg)
			ctx = mockApplyBlock(t, app, []*auth.StdTx{stdTx}, int64(i+3))
			require.Equal(t, tt.balance, app.AccountKeeper.GetAccount(ctx, testAccounts[0].addrKeys.Address).GetCoins().AmountOf(common.NativeToken).String())
		})
	}

}

func TestHandleTransferOwnership(t *testing.T) {
	common.InitConfig()
	app, keeper, testAccounts := getMockDexApp(t, 2)
	app.BeginBlock(abci.RequestBeginBlock{Header: abci.Header{Height: 2}})
	ctx := app.BaseApp.NewContext(false, abci.Header{}).WithBlockHeight(3)
	ctxPassedOwnershipConfirmWindow := app.BaseApp.NewContext(false, abci.Header{}).WithBlockTime(ctx.BlockTime().Add(types.DefaultOwnershipConfirmWindow * 2))
	handler := NewTokenHandler(keeper, version.ProtocolVersionV0)

	param := types.DefaultParams()
	app.tokenKeeper.SetParams(ctx, param)

	// issue token
	symbol := "xxb"
	msgNewIssue := types.NewMsgTokenIssue("xxb desc", symbol, symbol, symbol,
		"1000000", testAccounts[0], true)
	_, err := handler(ctx, msgNewIssue)
	require.Nil(t, err)

	tokenName := getTokenSymbol(ctx, keeper, symbol)

	// test case
	tests := []struct {
		ctx         sdk.Context
		msg         sdk.Msg
		expectedMsg string
	}{
		// case 1. sender is not the owner of token
		{
			ctx:         ctx,
			msg:         types.NewMsgTransferOwnership(testAccounts[1], testAccounts[0], tokenName),
			expectedMsg: fmt.Sprintf("input from address is not equal token owner: input from address is not equal token owner: %s", testAccounts[1]),
		},
		// case 2. transfer ownership to testAccounts[1] successfully
		{
			ctx:         ctx,
			msg:         types.NewMsgTransferOwnership(testAccounts[0], testAccounts[1], tokenName),
			expectedMsg: "",
		},
		// case 3. confirm ownership not exists
		{
			ctx:         ctx,
			msg:         types.NewMsgConfirmOwnership(testAccounts[1], "not-exist-token"),
			expectedMsg: fmt.Sprintf("get confirm ownership failed: get confirm ownership info failed"),
		},
		//// case 4. sender is not the owner of ConfirmOwnership
		{
			ctx:         ctx,
			msg:         types.NewMsgConfirmOwnership(testAccounts[0], tokenName),
			expectedMsg: fmt.Sprintf("input address is not equal confirm ownership address: input address (%s) is not equal confirm ownership address", testAccounts[0]),
		},
		// case 5. confirm ownership expired
		{
			ctx:         ctxPassedOwnershipConfirmWindow,
			msg:         types.NewMsgConfirmOwnership(testAccounts[1], tokenName),
			expectedMsg: fmt.Sprintf("confirm ownership not exist or blocktime after: confirm ownership not exist or blocktime after"),
		},
		// case 6. confirm ownership successfully
		{
			ctx:         ctx,
			msg:         types.NewMsgTransferOwnership(testAccounts[0], testAccounts[1], tokenName),
			expectedMsg: "",
		},
		{
			ctx:         ctx,
			msg:         types.NewMsgConfirmOwnership(testAccounts[1], tokenName),
			expectedMsg: "",
		},

		// case 7. transfer ownership to testAccounts[0] successfully
		{
			ctx:         ctx,
			msg:         types.NewMsgTransferOwnership(testAccounts[1], testAccounts[0], tokenName),
			expectedMsg: "",
		},
		// case 8. confirm ownership exists but expired, and transfer to black hole successfully
		{
			ctx:         ctxPassedOwnershipConfirmWindow,
			msg:         types.NewMsgTransferOwnership(testAccounts[1], common.BlackHoleAddress(), tokenName),
			expectedMsg: "",
		},
	}

	for _, testCase := range tests {
		_, err := handler(testCase.ctx, testCase.msg)

		if err != nil {
			require.EqualValues(t, testCase.expectedMsg, err.Error())
		} else {
			require.EqualValues(t, testCase.expectedMsg, "")
		}
	}

	token := keeper.GetTokenInfo(ctx, tokenName)
	require.True(t, token.Owner.Equals(common.BlackHoleAddress()))

}

func TestWalletTokenTransfer(t *testing.T) {
	app, keeper, addrs := getMockDexApp(t, 2)
	//tokenTransferMsg :=
	app.BeginBlock(abci.RequestBeginBlock{Header: abci.Header{Height: 2}})
	ctx := app.BaseApp.NewContext(false, abci.Header{}).WithBlockHeight(3)
	app.BaseApp.NewContext(false, abci.Header{}).WithBlockTime(ctx.BlockTime().Add(types.DefaultOwnershipConfirmWindow * 2))

	tests := []struct {
		info     string
		ctx      sdk.Context
		msg      sdk.Msg
		expected func()
		pass     bool
	}{
		{
			"succ with transfer and balance equal",
			ctx,
			&bank.MsgSendAdapter{
				FromAddress: addrs[0].String(),
				ToAddress:   addrs[1].String(),
				Amount:      sdk.CoinAdapters{sdk.NewCoinAdapter(sdk.DefaultBondDenom, sdk.NewIntFromBigInt(big.NewInt(1000000000000000000)))},
			},
			func() {
				require.Equal(t, app.AccountKeeper.GetAccount(ctx, addrs[0]).GetCoins().AmountOf(common.NativeToken).String(), "99999.000000000000000000")
				require.Equal(t, app.AccountKeeper.GetAccount(ctx, addrs[1]).GetCoins().AmountOf(common.NativeToken).String(), "100001.000000000000000000")
			},
			true,
		},
		{
			"failure insufficient funds",
			ctx,
			&bank.MsgSendAdapter{
				FromAddress: addrs[0].String(),
				ToAddress:   addrs[1].String(),
				Amount:      sdk.CoinAdapters{sdk.NewCoinAdapter(sdk.DefaultBondDenom, sdk.NewIntFromBigInt(new(big.Int).Mul(big.NewInt(1000000000000000000), big.NewInt(100000))))},
			},
			func() {
			},
			false,
		},
	}
	handler := NewTokenHandler(keeper, version.ProtocolVersionV0)
	for _, tc := range tests {
		_, err := handler(ctx, tc.msg)
		tc.expected()
		if tc.pass {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}
