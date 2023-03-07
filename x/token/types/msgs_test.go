package types

import (
	"strconv"
	"testing"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/tendermint/crypto/secp256k1"
	"github.com/okx/okbchain/x/common"
	"github.com/stretchr/testify/require"
)

func TestNewMsgTokenIssue(t *testing.T) {
	priKey := secp256k1.GenPrivKey()
	pubKey := priKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	totalSupply := "20000"

	testCase := []struct {
		issueMsg MsgTokenIssue
		err      sdk.Error
	}{
		{NewMsgTokenIssue("bnb", "bnb", "bnb", "binance coin", totalSupply, addr, true),
			nil},
		{NewMsgTokenIssue("", "", "", "binance coin", totalSupply, addr, true),
			ErrUserInputSymbolIsEmpty()},
		{NewMsgTokenIssue("bnb", "bnb", "bnb", "binance 278343298$%%^&  coin", totalSupply, addr, true),
			ErrWholeNameIsNotValidl()},
		{NewMsgTokenIssue("bnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbnbbn", "bnb", "bnb", "binance coin", totalSupply, addr, true),
			ErrDescLenBiggerThanLimit()},
		{NewMsgTokenIssue("bnb", "bnb", "bnb", "binance coin", strconv.FormatInt(int64(99*1e10), 10), addr, true),
			ErrTotalSupplyOutOfRange()},
		{NewMsgTokenIssue("", "", "", "binance coin", totalSupply, sdk.AccAddress{}, true),
			ErrAddressIsRequired()},
		{NewMsgTokenIssue("", "", "bnb-asd", "binance coin", totalSupply, addr, true),
			ErrNotAllowedOriginalSymbol("bnb-asd")},
	}

	for _, msgCase := range testCase {
		err := msgCase.issueMsg.ValidateBasic()
		if err != nil {
			require.EqualValues(t, msgCase.err.Error(), err.Error())
		} else {
			require.EqualValues(t, err, msgCase.err)
		}

	}

	tokenIssueMsg := testCase[0].issueMsg
	signAddr := tokenIssueMsg.GetSigners()
	require.EqualValues(t, []sdk.AccAddress{addr}, signAddr)

	bz := ModuleCdc.MustMarshalJSON(tokenIssueMsg)
	require.EqualValues(t, sdk.MustSortJSON(bz), tokenIssueMsg.GetSignBytes())
	require.EqualValues(t, "token", tokenIssueMsg.Route())
	require.EqualValues(t, "issue", tokenIssueMsg.Type())
}

func TestNewMsgTokenBurn(t *testing.T) {
	priKey := secp256k1.GenPrivKey()
	pubKey := priKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	decCoin := sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(100))

	decCoin0 := decCoin
	decCoin0.Denom = ""

	decCoin1 := decCoin
	decCoin1.Denom = "1okb-ads"

	testCase := []struct {
		burnMsg MsgTokenBurn
		err     sdk.Error
	}{
		{NewMsgTokenBurn(decCoin, addr), nil},
		{NewMsgTokenBurn(decCoin0, addr), common.ErrInsufficientCoins(DefaultParamspace, "100.000000000000000000")},
		{NewMsgTokenBurn(decCoin, sdk.AccAddress{}), ErrAddressIsRequired()},
		{NewMsgTokenBurn(decCoin1, addr), common.ErrInsufficientCoins(DefaultParamspace, "100.0000000000000000001okb-ads")},
	}

	for _, msgCase := range testCase {
		err := msgCase.burnMsg.ValidateBasic()
		if err != nil {
			require.EqualValues(t, msgCase.err.Error(), err.Error())
		} else {
			require.EqualValues(t, err, msgCase.err)
		}
	}

	tokenBurnMsg := testCase[0].burnMsg
	signAddr := tokenBurnMsg.GetSigners()
	require.EqualValues(t, []sdk.AccAddress{addr}, signAddr)

	bz := ModuleCdc.MustMarshalJSON(tokenBurnMsg)
	require.EqualValues(t, sdk.MustSortJSON(bz), tokenBurnMsg.GetSignBytes())
	require.EqualValues(t, "token", tokenBurnMsg.Route())
	require.EqualValues(t, "burn", tokenBurnMsg.Type())

	err := tokenBurnMsg.ValidateBasic()
	require.NoError(t, err)
}

//tokenMintMsg := NewMsgTokenMint("btc", mintNum, testAccounts[0].baseAccount.Address)
func TestNewMsgTokenMint(t *testing.T) {
	priKey := secp256k1.GenPrivKey()
	pubKey := priKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	decCoin := sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(1000))
	decCoin0 := decCoin
	decCoin0.Denom = ""

	decCoin1 := decCoin
	decCoin1.Denom = "11234"

	decCoin2 := decCoin
	decCoin2.Amount = sdk.NewDec(TotalSupplyUpperbound + 1)

	testCase := []struct {
		mintMsg MsgTokenMint
		err     sdk.Error
	}{
		{NewMsgTokenMint(decCoin, addr), nil},
		{NewMsgTokenMint(decCoin0, addr), ErrAmountIsNotValid("1000.000000000000000000")},
		{NewMsgTokenMint(decCoin, sdk.AccAddress{}), ErrAddressIsRequired()},
		{NewMsgTokenMint(decCoin1, addr), ErrAmountIsNotValid("1000.00000000000000000011234")},
		{NewMsgTokenMint(decCoin2, addr), ErrAmountBiggerThanTotalSupplyUpperbound()},
	}

	for _, msgCase := range testCase {
		err := msgCase.mintMsg.ValidateBasic()
		if err != nil {
			require.EqualValues(t, msgCase.err.Error(), err.Error())
		} else {
			require.EqualValues(t, err, msgCase.err)
		}
	}

	tokenMintMsg := testCase[0].mintMsg
	signAddr := tokenMintMsg.GetSigners()
	require.EqualValues(t, []sdk.AccAddress{addr}, signAddr)

	bz := ModuleCdc.MustMarshalJSON(tokenMintMsg)
	require.EqualValues(t, sdk.MustSortJSON(bz), tokenMintMsg.GetSignBytes())
	require.EqualValues(t, "token", tokenMintMsg.Route())
	require.EqualValues(t, "mint", tokenMintMsg.Type())

	err := tokenMintMsg.ValidateBasic()
	require.NoError(t, err)
}

func TestNewTokenMsgSend(t *testing.T) {
	// from
	fromPriKey := secp256k1.GenPrivKey()
	fromPubKey := fromPriKey.PubKey()
	fromAddr := sdk.AccAddress(fromPubKey.Address())

	// to
	toPriKey := secp256k1.GenPrivKey()
	toPubKey := toPriKey.PubKey()
	toAddr := sdk.AccAddress(toPubKey.Address())

	coins := sdk.SysCoins{
		sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(100)),
	}

	Errorcoins := sdk.SysCoins{
		sdk.NewDecCoinFromDec("okc", sdk.NewDec(100)),
		sdk.NewDecCoinFromDec("okc", sdk.NewDec(100)),
		sdk.NewDecCoinFromDec("oke", sdk.NewDec(100)),
	}

	// not valid coins
	decCoin := sdk.SysCoin{
		Denom:  "",
		Amount: sdk.NewDec(100),
	}
	notValidCoins := sdk.SysCoins{
		decCoin,
	}

	testCase := []struct {
		sendMsg MsgSend
		err     sdk.Error
	}{
		{NewMsgTokenSend(fromAddr, toAddr, coins), nil},
		{NewMsgTokenSend(fromAddr, toAddr, sdk.SysCoins{}), common.ErrInsufficientCoins(DefaultParamspace, "")},
		{NewMsgTokenSend(fromAddr, toAddr, Errorcoins), ErrInvalidCoins("100.000000000000000000okc,100.000000000000000000okc,100.000000000000000000oke")},
		{NewMsgTokenSend(sdk.AccAddress{}, toAddr, coins), ErrAddressIsRequired()},
		{NewMsgTokenSend(fromAddr, sdk.AccAddress{}, coins), ErrAddressIsRequired()},
		{NewMsgTokenSend(fromAddr, toAddr, notValidCoins), ErrInvalidCoins("100.000000000000000000")},
	}
	for _, msgCase := range testCase {
		err := msgCase.sendMsg.ValidateBasic()
		if err != nil {
			require.EqualValues(t, msgCase.err.Error(), err.Error())
		} else {
			require.EqualValues(t, err, msgCase.err)
		}
	}

	tokenSendMsg := testCase[0].sendMsg
	signAddr := tokenSendMsg.GetSigners()
	require.EqualValues(t, []sdk.AccAddress{fromAddr}, signAddr)
	require.EqualValues(t, RouterKey, tokenSendMsg.Route())
	require.EqualValues(t, "send", tokenSendMsg.Type())

	bz := ModuleCdc.MustMarshalJSON(tokenSendMsg)
	require.EqualValues(t, sdk.MustSortJSON(bz), tokenSendMsg.GetSignBytes())

	err := tokenSendMsg.ValidateBasic()
	require.NoError(t, err)
}

func TestNewTokenMultiSend(t *testing.T) {
	common.InitConfig()
	// from
	fromPriKey := secp256k1.GenPrivKey()
	fromPubKey := fromPriKey.PubKey()
	fromAddr := sdk.AccAddress(fromPubKey.Address())

	// correct message
	coinStr := `[{"to":"ex1jedas2n0pq2c68pelztgel8ht8pz50rh7s7vfz","amount":"1` + common.NativeToken + `"}]`
	transfers, err := StrToTransfers(coinStr)
	require.Nil(t, err)

	// coins not positive
	toAddr0, err := sdk.AccAddressFromBech32("ex1jedas2n0pq2c68pelztgel8ht8pz50rh7s7vfz")
	require.Nil(t, err)
	decCoin0 := sdk.NewDecCoinFromDec(common.NativeToken, sdk.NewDec(0))
	transfers0 := []TransferUnit{
		{
			To:    toAddr0,
			Coins: sdk.SysCoins{decCoin0},
		},
	}

	// empty toAddr
	decCoin1 := sdk.NewDecCoinFromDec("obk", sdk.NewDec(100))
	transfers1 := []TransferUnit{
		{
			To:    sdk.AccAddress{},
			Coins: sdk.SysCoins{decCoin1},
		},
	}

	testCase := []struct {
		multiSendMsg MsgMultiSend
		err          sdk.Error
	}{
		{NewMsgMultiSend(fromAddr, transfers), nil},
		{NewMsgMultiSend(sdk.AccAddress{}, transfers), ErrAddressIsRequired()},
		{NewMsgMultiSend(fromAddr, make([]TransferUnit, MultiSendLimit+1)), ErrMsgTransfersAmountBiggerThanSendLimit()},
		{NewMsgMultiSend(fromAddr, transfers0), ErrInvalidCoins("0.000000000000000000okt")},
		{NewMsgMultiSend(fromAddr, transfers1), ErrAddressIsRequired()},
	}
	for _, msgCase := range testCase {
		err := msgCase.multiSendMsg.ValidateBasic()
		if err != nil {
			require.EqualValues(t, msgCase.err.Error(), err.Error())
		} else {
			require.EqualValues(t, err, msgCase.err)
		}
	}

	tokenMultiSendMsg := testCase[0].multiSendMsg
	signAddr := tokenMultiSendMsg.GetSigners()
	require.EqualValues(t, []sdk.AccAddress{fromAddr}, signAddr)

	bz := ModuleCdc.MustMarshalJSON(tokenMultiSendMsg)

	require.NoError(t, err)
	require.EqualValues(t, sdk.MustSortJSON(bz), tokenMultiSendMsg.GetSignBytes())
	require.EqualValues(t, "token", tokenMultiSendMsg.Route())
	require.EqualValues(t, "multi-send", tokenMultiSendMsg.Type())

	err = tokenMultiSendMsg.ValidateBasic()
	require.NoError(t, err)
}

func TestNewMsgTransferOwnership(t *testing.T) {
	common.InitConfig()
	// from
	fromPriKey := secp256k1.GenPrivKey()
	fromPubKey := fromPriKey.PubKey()
	fromAddr := sdk.AccAddress(fromPubKey.Address())

	// to
	toPriKey := secp256k1.GenPrivKey()
	toPubKey := toPriKey.PubKey()
	toAddr := sdk.AccAddress(toPubKey.Address())

	testCase := []struct {
		transferOwnershipMsg MsgTransferOwnership
		err                  sdk.Error
	}{
		{NewMsgTransferOwnership(fromAddr, sdk.AccAddress{}, common.NativeToken), ErrAddressIsRequired()},
		{NewMsgTransferOwnership(sdk.AccAddress{}, toAddr, common.NativeToken), ErrAddressIsRequired()},
		{NewMsgTransferOwnership(fromAddr, toAddr, ""), ErrMsgSymbolIsEmpty()},
		{NewMsgTransferOwnership(fromAddr, toAddr, "1okb-ads"), ErrConfirmOwnershipNotExistOrBlockTimeAfter()},
	}
	for _, msgCase := range testCase {
		err := msgCase.transferOwnershipMsg.ValidateBasic()
		if err != nil {
			require.EqualValues(t, msgCase.err.Error(), err.Error())
		} else {
			require.EqualValues(t, err, msgCase.err)
		}
	}

	transferOwnershipMsg := testCase[0].transferOwnershipMsg
	transferOwnershipMsg.Route()
	transferOwnershipMsg.Type()
	signAddr := transferOwnershipMsg.GetSigners()
	require.EqualValues(t, []sdk.AccAddress{fromAddr}, signAddr)
}

func TestNewMsgTokenModify(t *testing.T) {
	common.InitConfig()

	priKey := secp256k1.GenPrivKey()
	pubKey := priKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	testCase := []struct {
		tokenModifyMsg MsgTokenModify
		err            sdk.Error
	}{
		{NewMsgTokenModify("bnb", "bnb", "bnb bnb", true, true, addr),
			nil},
		{NewMsgTokenModify("", "bnb", "bnb bnb", true, true, addr),
			ErrMsgSymbolIsEmpty()},
		{NewMsgTokenModify("bnb", "bnb", "bnb bnb", true, true, sdk.AccAddress{}),
			ErrAddressIsRequired()},
		{NewMsgTokenModify("bnb", "bnb", "bnbbbbbbbbbb bnbbbbbbbbbbbbbbbbb", true, true, addr),
			ErrWholeNameIsNotValidl()},
		{NewMsgTokenModify("bnb", "bnb", "bnbbbbbbbbbb bnbbbbbbbbbbbbbbbbb", true, false, addr),
			nil},
		{NewMsgTokenModify("bnb", `bnbbbbbbbbbbbnbbbbbbbbbbnbbbbbbbbbbbnbbbbbbbbb1234
bnbbbbbbbbbbbnbbbbbbbbbbnbbbbbbbbbbbnbbbbbbbbb1234
bnbbbbbbbbbbbnbbbbbbbbbbnbbbbbbbbbbbnbbbbbbbbb1234
bnbbbbbbbbbbbnbbbbbbbbbbnbbbbbbbbbbbnbbbbbbbbb1234
bnbbbbbbbbbbbnbbbbbbbbbbnbbbbbbbbbbbnbbbbbbbbb1234
bnbbbbbbbbbbbnbbbbbbbbbbnbbbbbbbbbbbnbbbbbbbbb1234`, "bnbbbbbbbbbb", true, false, addr),
			ErrDescLenBiggerThanLimit()},
		{NewMsgTokenModify("bnb", `bnbbbbbbbbbbbnbbbbbbbbbbnbbbbbbbbbbbnbbbbbbbbb1234
bnbbbbbbbbbbbnbbbbbbbbbbnbbbbbbbbbbbnbbbbbbbbb1234
bnbbbbbbbbbbbnbbbbbbbbbbnbbbbbbbbbbbnbbbbbbbbb1234
bnbbbbbbbbbbbnbbbbbbbbbbnbbbbbbbbbbbnbbbbbbbbb1234
bnbbbbbbbbbbbnbbbbbbbbbbnbbbbbbbbbbbnbbbbbbbbb1234
bnbbbbbbbbbbbnbbbbbbbbbbnbbbbbbbbbbbnbbbbbbbbb1234`, "bnbbbbbbbbbb", false, false, addr),
			nil},
	}
	for _, msgCase := range testCase {
		err := msgCase.tokenModifyMsg.ValidateBasic()
		if err != nil {
			require.EqualValues(t, msgCase.err.Error(), err.Error())
		} else {
			require.EqualValues(t, err, msgCase.err)
		}
	}

	// correct message
	tokenEditMsg := testCase[0].tokenModifyMsg
	signAddr := tokenEditMsg.GetSigners()
	require.EqualValues(t, []sdk.AccAddress{addr}, signAddr)

	bz := ModuleCdc.MustMarshalJSON(tokenEditMsg)
	require.EqualValues(t, sdk.MustSortJSON(bz), tokenEditMsg.GetSignBytes())
	require.EqualValues(t, "edit", tokenEditMsg.Type())
	require.EqualValues(t, "token", tokenEditMsg.Route())

	err := tokenEditMsg.ValidateBasic()
	require.NoError(t, err)
}
