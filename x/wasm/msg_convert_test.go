package wasm

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/x/wasm/types"
)

var (
	addr, _ = sdk.AccAddressFromHex("B2910E22Bb23D129C02d122B77B462ceB0E89Db9")
)

func testMustAccAddressFromBech32(addr string) sdk.AccAddress {
	re, err := sdk.AccAddressFromBech32(addr)
	if err != nil {
		panic(err)
	}
	return re
}

func newTestSysCoin(i int64, precison int64) sdk.SysCoin {
	return sdk.NewDecCoinFromDec(sdk.DefaultBondDenom, sdk.NewDecWithPrec(i, precison))
}

func TestMsgStoreCode(t *testing.T) {
	msg := types.MsgStoreCode{
		Sender:                "0x67582AB2adb08a8583A181b7745762B53710e9B1",
		WASMByteCode:          []byte("hello"),
		InstantiatePermission: &types.AccessConfig{3, "0x67582AB2adb08a8583A181b7745762B53710e9B1"},
	}
	d, err := json.Marshal(msg)
	assert.NoError(t, err)
	fmt.Println(string(d))
}

func TestMsgInstantiateContract(t *testing.T) {
	msg := types.MsgInstantiateContract{
		Sender: "0x67582AB2adb08a8583A181b7745762B53710e9B1",
		Admin:  "0x67582AB2adb08a8583A181b7745762B53710e9B2",
		CodeID: 2,
		Label:  "hello",
		Msg:    []byte("{\"balance\":{\"address\":\"0xCf164e001d86639231d92Ab1D71DB8353E43C295\"}}"),
		Funds:  sdk.CoinsToCoinAdapters([]sdk.DecCoin{sdk.NewDecCoin("mytoken", sdk.NewInt(10))}),
	}
	d, err := json.Marshal(msg)
	assert.NoError(t, err)
	fmt.Println(string(d))
}

func TestMsgExecuteContract(t *testing.T) {
	msg := types.MsgExecuteContract{
		Sender:   "0x67582AB2adb08a8583A181b7745762B53710e9B1",
		Contract: "0x67582AB2adb08a8583A181b7745762B53710e9B2",
		Msg:      []byte("{\"balance\":{\"address\":\"0xCf164e001d86639231d92Ab1D71DB8353E43C295\"}}"),
		Funds:    sdk.CoinsToCoinAdapters([]sdk.DecCoin{sdk.NewDecCoin("mytoken", sdk.NewInt(10))}),
	}
	d, err := json.Marshal(msg)
	assert.NoError(t, err)
	fmt.Println(string(d))
}

func TestMsgMigrateContract(t *testing.T) {
	msg := types.MsgMigrateContract{
		Sender:   "0x67582AB2adb08a8583A181b7745762B53710e9B1",
		Contract: "0x67582AB2adb08a8583A181b7745762B53710e9B2",
		CodeID:   1,
		Msg:      []byte("{\"balance\":{\"address\":\"0xCf164e001d86639231d92Ab1D71DB8353E43C295\"}}"),
	}
	d, err := json.Marshal(msg)
	assert.NoError(t, err)
	fmt.Println(string(d))
}

func TestMsgUpdateAdmin(t *testing.T) {
	msg := types.MsgUpdateAdmin{
		Sender:   "0x67582AB2adb08a8583A181b7745762B53710e9B1",
		NewAdmin: "0x67582AB2adb08a8583A181b7745762B53710e9B3",
		Contract: "0x67582AB2adb08a8583A181b7745762B53710e9B2",
	}
	d, err := json.Marshal(msg)
	assert.NoError(t, err)
	fmt.Println(string(d))
}

func TestConvertMsgStoreCode(t *testing.T) {
	//addr, err := sdk.AccAddressFromHex("B2910E22Bb23D129C02d122B77B462ceB0E89Db9")
	//require.NoError(t, err)

	testcases := []struct {
		msgstr  string
		res     types.MsgStoreCode
		fnCheck func(msg sdk.Msg, err error, res types.MsgStoreCode)
	}{
		{
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"wasm_byte_code\":\"aGVsbG8=\",\"instantiate_permission\":{\"permission\":\"OnlyAddress\",\"address\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\"}}",
			res: types.MsgStoreCode{
				Sender:                "0x67582AB2adb08a8583A181b7745762B53710e9B1",
				WASMByteCode:          []byte("hello"),
				InstantiatePermission: &types.AccessConfig{2, "0x67582AB2adb08a8583A181b7745762B53710e9B1"},
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgStoreCode) {
				require.NoError(t, err)
				require.Equal(t, *msg.(*types.MsgStoreCode), res)
			},
		},
		{
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"wasm_byte_code\":\"aGVsbG8=\",\"instantiate_permission\":{\"address\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\"}}",
			res: types.MsgStoreCode{
				Sender:                "0x67582AB2adb08a8583A181b7745762B53710e9B1",
				WASMByteCode:          []byte("hello"),
				InstantiatePermission: &types.AccessConfig{0, "0x67582AB2adb08a8583A181b7745762B53710e9B1"},
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgStoreCode) {
				require.Error(t, err)
				require.Nil(t, msg)
			},
		},
		{
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"wasm_byte_code\":\"aGVsbG8=\"}",
			res: types.MsgStoreCode{
				Sender:       "0x67582AB2adb08a8583A181b7745762B53710e9B1",
				WASMByteCode: []byte("hello"),
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgStoreCode) {
				require.NoError(t, err)
				require.Equal(t, *msg.(*types.MsgStoreCode), res)
			},
		},
		{
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"wasm_byte_code\":\"aGVsbG8=\",\"instantiate_permission\":{\"permission\":\"OnlyAddress\",\"address\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\"}}",
			res: types.MsgStoreCode{
				Sender:                "0xbbE4733d85bc2b90682147779DA49caB38C0aA1F",
				WASMByteCode:          []byte("hello"),
				InstantiatePermission: &types.AccessConfig{2, "0x67582AB2adb08a8583A181b7745762B53710e9B1"},
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgStoreCode) {
				require.Equal(t, ErrCheckSignerFail, err)
				require.Nil(t, msg)
			},
		},
	}

	for _, ts := range testcases {
		msg, err := ConvertMsgStoreCode([]byte(ts.msgstr), ts.res.GetSigners(), 2)
		ts.fnCheck(msg, err, ts.res)
	}
}

func TestConvertMsgInstantiateContract(t *testing.T) {
	//addr, err := sdk.AccAddressFromHex("B2910E22Bb23D129C02d122B77B462ceB0E89Db9")
	//require.NoError(t, err)

	testcases := []struct {
		msgstr  string
		res     types.MsgInstantiateContract
		fnCheck func(msg sdk.Msg, err error, res types.MsgInstantiateContract)
	}{
		{
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"admin\":\"0x67582AB2adb08a8583A181b7745762B53710e9B2\",\"code_id\":2,\"label\":\"hello\",\"msg\":{\"balance\":{\"address\":\"0xCf164e001d86639231d92Ab1D71DB8353E43C295\"}},\"funds\":[{\"denom\":\"mytoken\",\"amount\":\"10000000000000000000\"}]}",
			res: types.MsgInstantiateContract{
				Sender: "0x67582AB2adb08a8583A181b7745762B53710e9B1",
				Admin:  "0x67582AB2adb08a8583A181b7745762B53710e9B2",
				CodeID: 2,
				Label:  "hello",
				Msg:    []byte("{\"balance\":{\"address\":\"0xCf164e001d86639231d92Ab1D71DB8353E43C295\"}}"),
				Funds:  sdk.CoinsToCoinAdapters([]sdk.DecCoin{sdk.NewDecCoin("mytoken", sdk.NewInt(10))}),
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgInstantiateContract) {
				require.NoError(t, err)
				require.Equal(t, *msg.(*types.MsgInstantiateContract), res)
			},
		},
		{ // Msg need "{}" and Funds field can not fill:
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"admin\":\"0x67582AB2adb08a8583A181b7745762B53710e9B2\",\"code_id\":2,\"label\":\"hello\",\"msg\":{}}",
			res: types.MsgInstantiateContract{
				Sender: "0x67582AB2adb08a8583A181b7745762B53710e9B1",
				Admin:  "0x67582AB2adb08a8583A181b7745762B53710e9B2",
				CodeID: 2,
				Label:  "hello",
				Msg:    []byte("{}"),
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgInstantiateContract) {
				require.NoError(t, err)
				require.Equal(t, *msg.(*types.MsgInstantiateContract), res)
			},
		},
		// error
		{ // no Msg field
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"admin\":\"0x67582AB2adb08a8583A181b7745762B53710e9B2\",\"code_id\":2,\"label\":\"hello\",\"funds\":[]}",
			res: types.MsgInstantiateContract{
				Sender: "0x67582AB2adb08a8583A181b7745762B53710e9B1",
				Admin:  "0x67582AB2adb08a8583A181b7745762B53710e9B2",
				CodeID: 2,
				Label:  "hello",
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgInstantiateContract) {
				require.Error(t, err)
				require.Nil(t, msg)
			},
		},
		{
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"admin\":\"0x67582AB2adb08a8583A181b7745762B53710e9B2\",\"code_id\":2,\"label\":\"hello\",\"msg\":{\"balance\":{\"address\":\"0xCf164e001d86639231d92Ab1D71DB8353E43C295\"}},\"funds\":[{\"denom\":\"mytoken\",\"amount\":\"10000000000000000000\"}]}",
			res: types.MsgInstantiateContract{
				Sender: "0xbbE4733d85bc2b90682147779DA49caB38C0aA1F",
				Admin:  "0x67582AB2adb08a8583A181b7745762B53710e9B2",
				CodeID: 2,
				Label:  "hello",
				Msg:    []byte("{\"balance\":{\"address\":\"0xCf164e001d86639231d92Ab1D71DB8353E43C295\"}}"),
				Funds:  sdk.CoinsToCoinAdapters([]sdk.DecCoin{sdk.NewDecCoin("mytoken", sdk.NewInt(10))}),
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgInstantiateContract) {
				require.Equal(t, ErrCheckSignerFail, err)
				require.Nil(t, msg)
			},
		},
	}

	for _, ts := range testcases {
		msg, err := ConvertMsgInstantiateContract([]byte(ts.msgstr), ts.res.GetSigners(), 2)
		ts.fnCheck(msg, err, ts.res)
	}
}

func TestConvertMsgExecuteContract(t *testing.T) {
	testcases := []struct {
		msgstr  string
		res     types.MsgExecuteContract
		fnCheck func(msg sdk.Msg, err error, res types.MsgExecuteContract)
	}{
		{
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"contract\":\"0x67582AB2adb08a8583A181b7745762B53710e9B2\",\"msg\":{\"balance\":{\"address\":\"0xCf164e001d86639231d92Ab1D71DB8353E43C295\"}},\"funds\":[{\"denom\":\"mytoken\",\"amount\":\"10000000000000000000\"}]}",
			res: types.MsgExecuteContract{
				Sender:   "0x67582AB2adb08a8583A181b7745762B53710e9B1",
				Contract: "0x67582AB2adb08a8583A181b7745762B53710e9B2",
				Msg:      []byte("{\"balance\":{\"address\":\"0xCf164e001d86639231d92Ab1D71DB8353E43C295\"}}"),
				Funds:    sdk.CoinsToCoinAdapters([]sdk.DecCoin{sdk.NewDecCoin("mytoken", sdk.NewInt(10))}),
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgExecuteContract) {
				require.NoError(t, err)
				require.Equal(t, *msg.(*types.MsgExecuteContract), res)
			},
		},
		{ // Msg need "{}" and Funds field can not fill:
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"contract\":\"0x67582AB2adb08a8583A181b7745762B53710e9B2\",\"msg\":{}}",
			res: types.MsgExecuteContract{
				Sender:   "0x67582AB2adb08a8583A181b7745762B53710e9B1",
				Contract: "0x67582AB2adb08a8583A181b7745762B53710e9B2",
				Msg:      []byte("{}"),
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgExecuteContract) {
				require.NoError(t, err)
				require.Equal(t, *msg.(*types.MsgExecuteContract), res)
			},
		},
		// error
		{ // no Msg field
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"contract\":\"0x67582AB2adb08a8583A181b7745762B53710e9B2\",\"funds\":[{\"denom\":\"mytoken\",\"amount\":\"10000000000000000000\"}]}",
			res: types.MsgExecuteContract{
				Sender:   "0x67582AB2adb08a8583A181b7745762B53710e9B1",
				Contract: "0x67582AB2adb08a8583A181b7745762B53710e9B2",
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgExecuteContract) {
				require.Error(t, err)
				require.Nil(t, msg)
			},
		},
		{
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"contract\":\"0x67582AB2adb08a8583A181b7745762B53710e9B2\",\"msg\":{\"balance\":{\"address\":\"0xCf164e001d86639231d92Ab1D71DB8353E43C295\"}},\"funds\":[{\"denom\":\"mytoken\",\"amount\":\"10000000000000000000\"}]}",
			res: types.MsgExecuteContract{
				Sender:   "0xbbE4733d85bc2b90682147779DA49caB38C0aA1F",
				Contract: "0x67582AB2adb08a8583A181b7745762B53710e9B2",
				Msg:      []byte("{\"balance\":{\"address\":\"0xCf164e001d86639231d92Ab1D71DB8353E43C295\"}}"),
				Funds:    sdk.CoinsToCoinAdapters([]sdk.DecCoin{sdk.NewDecCoin("mytoken", sdk.NewInt(10))}),
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgExecuteContract) {
				require.Equal(t, ErrCheckSignerFail, err)
				require.Nil(t, msg)
			},
		},
	}

	for _, ts := range testcases {
		msg, err := ConvertMsgExecuteContract([]byte(ts.msgstr), ts.res.GetSigners(), 2)
		ts.fnCheck(msg, err, ts.res)
	}
}

func TestConvertMsgMigrateContract(t *testing.T) {
	testcases := []struct {
		msgstr  string
		res     types.MsgMigrateContract
		fnCheck func(msg sdk.Msg, err error, res types.MsgMigrateContract)
	}{
		{
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"contract\":\"0x67582AB2adb08a8583A181b7745762B53710e9B2\",\"code_id\":1,\"msg\":{\"balance\":{\"address\":\"0xCf164e001d86639231d92Ab1D71DB8353E43C295\"}}}",
			res: types.MsgMigrateContract{
				Sender:   "0x67582AB2adb08a8583A181b7745762B53710e9B1",
				Contract: "0x67582AB2adb08a8583A181b7745762B53710e9B2",
				CodeID:   1,
				Msg:      []byte("{\"balance\":{\"address\":\"0xCf164e001d86639231d92Ab1D71DB8353E43C295\"}}"),
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgMigrateContract) {
				require.NoError(t, err)
				require.Equal(t, *msg.(*types.MsgMigrateContract), res)
			},
		},
		{ // Msg need "{}"
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"contract\":\"0x67582AB2adb08a8583A181b7745762B53710e9B2\",\"code_id\":1,\"msg\":{}}",
			res: types.MsgMigrateContract{
				Sender:   "0x67582AB2adb08a8583A181b7745762B53710e9B1",
				Contract: "0x67582AB2adb08a8583A181b7745762B53710e9B2",
				CodeID:   1,
				Msg:      []byte("{}"),
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgMigrateContract) {
				require.NoError(t, err)
				require.Equal(t, *msg.(*types.MsgMigrateContract), res)
			},
		},
		// error
		{ // no code id
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"contract\":\"0x67582AB2adb08a8583A181b7745762B53710e9B2\",\"msg\":{\"balance\":{\"address\":\"0xCf164e001d86639231d92Ab1D71DB8353E43C295\"}}}",
			res: types.MsgMigrateContract{
				Sender:   "0x67582AB2adb08a8583A181b7745762B53710e9B1",
				Contract: "0x67582AB2adb08a8583A181b7745762B53710e9B2",
				Msg:      []byte("{\"balance\":{\"address\":\"0xCf164e001d86639231d92Ab1D71DB8353E43C295\"}}"),
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgMigrateContract) {
				require.Error(t, err)
				require.Nil(t, msg)
			},
		},
		{ // no Msg field
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"contract\":\"0x67582AB2adb08a8583A181b7745762B53710e9B2\",\"code_id\":1}",
			res: types.MsgMigrateContract{
				Sender:   "0x67582AB2adb08a8583A181b7745762B53710e9B1",
				Contract: "0x67582AB2adb08a8583A181b7745762B53710e9B2",
				CodeID:   1,
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgMigrateContract) {
				require.Error(t, err)
				require.Nil(t, msg)
			},
		},
		{
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"contract\":\"0x67582AB2adb08a8583A181b7745762B53710e9B2\",\"code_id\":1,\"msg\":{\"balance\":{\"address\":\"0xCf164e001d86639231d92Ab1D71DB8353E43C295\"}}}",
			res: types.MsgMigrateContract{
				Sender:   "0xbbE4733d85bc2b90682147779DA49caB38C0aA1F",
				Contract: "0x67582AB2adb08a8583A181b7745762B53710e9B2",
				CodeID:   1,
				Msg:      []byte("{\"balance\":{\"address\":\"0xCf164e001d86639231d92Ab1D71DB8353E43C295\"}}"),
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgMigrateContract) {
				require.Equal(t, ErrCheckSignerFail, err)
				require.Nil(t, msg)
			},
		},
	}
	for _, ts := range testcases {
		msg, err := ConvertMsgMigrateContract([]byte(ts.msgstr), ts.res.GetSigners(), 2)
		ts.fnCheck(msg, err, ts.res)
	}
}

func TestConvertMsgUpdateAdmin(t *testing.T) {
	testcases := []struct {
		msgstr  string
		res     types.MsgUpdateAdmin
		fnCheck func(msg sdk.Msg, err error, res types.MsgUpdateAdmin)
	}{
		{
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"new_admin\":\"0x67582AB2adb08a8583A181b7745762B53710e9B3\",\"contract\":\"0x67582AB2adb08a8583A181b7745762B53710e9B2\"}",
			res: types.MsgUpdateAdmin{
				Sender:   "0x67582AB2adb08a8583A181b7745762B53710e9B1",
				NewAdmin: "0x67582AB2adb08a8583A181b7745762B53710e9B3",
				Contract: "0x67582AB2adb08a8583A181b7745762B53710e9B2",
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgUpdateAdmin) {
				require.NoError(t, err)
				require.Equal(t, *msg.(*types.MsgUpdateAdmin), res)
			},
		},
		// error
		{
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"new_admin\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"contract\":\"0x67582AB2adb08a8583A181b7745762B53710e9B2\"}",
			res: types.MsgUpdateAdmin{
				Sender:   "0x67582AB2adb08a8583A181b7745762B53710e9B1",
				NewAdmin: "0x67582AB2adb08a8583A181b7745762B53710e9B1",
				Contract: "0x67582AB2adb08a8583A181b7745762B53710e9B2",
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgUpdateAdmin) {
				require.Error(t, err)
				require.Nil(t, msg)
			},
		},
		{
			msgstr: "{\"sender\":\"0x67582AB2adb08a8583A181b7745762B53710e9B1\",\"new_admin\":\"0x67582AB2adb08a8583A181b7745762B53710e9B3\",\"contract\":\"0x67582AB2adb08a8583A181b7745762B53710e9B2\"}",
			res: types.MsgUpdateAdmin{
				Sender:   "0xbbE4733d85bc2b90682147779DA49caB38C0aA1F",
				NewAdmin: "0x67582AB2adb08a8583A181b7745762B53710e9B3",
				Contract: "0x67582AB2adb08a8583A181b7745762B53710e9B2",
			},
			fnCheck: func(msg sdk.Msg, err error, res types.MsgUpdateAdmin) {
				require.Equal(t, ErrCheckSignerFail, err)
				require.Nil(t, msg)
			},
		},
	}
	for _, ts := range testcases {
		msg, err := ConvertMsgUpdateAdmin([]byte(ts.msgstr), ts.res.GetSigners(), 2)
		ts.fnCheck(msg, err, ts.res)
	}
}
