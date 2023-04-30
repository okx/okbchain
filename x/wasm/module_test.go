package wasm

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/okx/okbchain/x/wasm/keeper/testdata"

	"github.com/dvsekhvalnov/jose2go/base64url"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/types/module"
	authkeeper "github.com/okx/okbchain/libs/cosmos-sdk/x/auth/keeper"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/bank"
	bankkeeper "github.com/okx/okbchain/libs/cosmos-sdk/x/bank"
	abci "github.com/okx/okbchain/libs/tendermint/abci/types"
	"github.com/okx/okbchain/libs/tendermint/crypto"
	"github.com/okx/okbchain/libs/tendermint/crypto/ed25519"
	"github.com/okx/okbchain/libs/tendermint/libs/kv"
	types2 "github.com/okx/okbchain/libs/tendermint/types"
	stakingkeeper "github.com/okx/okbchain/x/staking/keeper"
	"github.com/okx/okbchain/x/wasm/keeper"
	"github.com/okx/okbchain/x/wasm/types"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var zeroCoins sdk.Coins

type testData struct {
	module        module.AppModule
	ctx           sdk.Context
	acctKeeper    authkeeper.AccountKeeper
	keeper        Keeper
	bankKeeper    bankkeeper.Keeper
	stakingKeeper stakingkeeper.Keeper
	faucet        *keeper.TestFaucet
}

func setupTest(t *testing.T) testData {
	ctx, keepers := CreateTestInput(t, false, SupportedFeatures)
	cdc := keeper.MakeTestCodec(t)
	data := testData{
		module:        NewAppModule(cdc, keepers.WasmKeeper),
		ctx:           ctx,
		acctKeeper:    keepers.AccountKeeper,
		keeper:        *keepers.WasmKeeper,
		bankKeeper:    keepers.BankKeeper,
		stakingKeeper: keepers.StakingKeeper,
		faucet:        keepers.Faucet,
	}

	return data
}

func keyPubAddr() (crypto.PrivKey, crypto.PubKey, sdk.WasmAddress) {
	key := ed25519.GenPrivKey()
	pub := key.PubKey()
	addr := sdk.WasmAddress(pub.Address())
	return key, pub, addr
}

func mustLoad(path string) []byte {
	bz, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return bz
}

var (
	_, _, addrAcc1 = keyPubAddr()
	addr1          = addrAcc1.String()
	testContract   = mustLoad("./keeper/testdata/hackatom.wasm")
	maskContract   = testdata.ReflectContractWasm()
	oldContract    = mustLoad("./testdata/escrow_0.7.wasm")
)

func TestHandleCreate(t *testing.T) {
	types2.UnittestOnlySetMilestoneEarthHeight(1)
	cases := map[string]struct {
		msg     sdk.Msg
		isValid bool
	}{
		"empty": {
			msg:     &MsgStoreCode{},
			isValid: false,
		},
		"invalid wasm": {
			msg: &MsgStoreCode{
				Sender:       addr1,
				WASMByteCode: []byte("foobar"),
			},
			isValid: false,
		},
		"valid wasm": {
			msg: &MsgStoreCode{
				Sender:       addr1,
				WASMByteCode: testContract,
			},
			isValid: true,
		},
		"other valid wasm": {
			msg: &MsgStoreCode{
				Sender:       addr1,
				WASMByteCode: maskContract,
			},
			isValid: true,
		},
		"old wasm (0.7)": {
			msg: &MsgStoreCode{
				Sender:       addr1,
				WASMByteCode: oldContract,
			},
			isValid: false,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			data := setupTest(t)

			h := data.module.NewHandler()
			q := data.module.NewQuerierHandler()

			res, err := h(data.ctx, tc.msg)
			if !tc.isValid {
				require.Error(t, err, "%#v", res)
				assertCodeList(t, q, data.ctx, 0)
				assertCodeBytes(t, q, data.ctx, 1, nil)
				return
			}
			require.NoError(t, err)
			assertCodeList(t, q, data.ctx, 1)
		})
	}
}

type initMsg struct {
	Verifier    sdk.WasmAddress `json:"verifier"`
	Beneficiary sdk.WasmAddress `json:"beneficiary"`
}

type state struct {
	Verifier    string `json:"verifier"`
	Beneficiary string `json:"beneficiary"`
	Funder      string `json:"funder"`
}

func TestHandleInstantiate(t *testing.T) {
	types2.UnittestOnlySetMilestoneEarthHeight(1)
	data := setupTest(t)
	creator := data.faucet.NewFundedAccount(data.ctx, sdk.NewInt64Coin("denom", 100000))

	h := data.module.NewHandler()
	q := data.module.NewQuerierHandler()

	msg := &MsgStoreCode{
		Sender:       creator.String(),
		WASMByteCode: testContract,
	}
	res, err := h(data.ctx, msg)
	require.NoError(t, err)
	assertStoreCodeResponse(t, res.Data, 1)

	_, _, bob := keyPubAddr()
	_, _, fred := keyPubAddr()

	initMsg := initMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	// create with no balance is also legal
	initCmd := MsgInstantiateContract{
		Sender: creator.String(),
		CodeID: firstCodeID,
		Msg:    initMsgBz,
		Funds:  nil,
	}
	res, err = h(data.ctx, &initCmd)
	require.NoError(t, err)
	contractBech32Addr := parseInitResponse(t, res.Data)

	require.Equal(t, "0x5A8D648DEE57b2fc90D98DC17fa887159b69638b", contractBech32Addr)
	// this should be standard x/wasm init event, nothing from contract
	require.Equal(t, 3, len(res.Events), prettyEvents(res.Events))
	require.Equal(t, "message", res.Events[0].Type)
	assertAttribute(t, "module", "wasm", res.Events[0].Attributes[0])
	require.Equal(t, "instantiate", res.Events[1].Type)
	require.Equal(t, "wasm", res.Events[2].Type)
	assertAttribute(t, "_contract_address", contractBech32Addr, res.Events[2].Attributes[0])

	assertCodeList(t, q, data.ctx, 1)
	assertCodeBytes(t, q, data.ctx, 1, testContract)

	assertContractList(t, q, data.ctx, 1, []string{contractBech32Addr})
	assertContractInfo(t, q, data.ctx, contractBech32Addr, 1, creator)
	assertContractState(t, q, data.ctx, contractBech32Addr, state{
		Verifier:    fred.String(),
		Beneficiary: bob.String(),
		Funder:      creator.String(),
	})
}

func TestHandleExecute(t *testing.T) {
	types2.UnittestOnlySetMilestoneEarthHeight(1)
	data := setupTest(t)
	types2.UnittestOnlySetMilestoneEarthHeight(1)
	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	topUp := sdk.NewCoins(sdk.NewInt64Coin("denom", 5000))

	creator := data.faucet.NewFundedAccount(data.ctx, deposit.Add(deposit...)...)
	fred := data.faucet.NewFundedAccount(data.ctx, topUp...)

	h := data.module.NewHandler()
	q := data.module.NewQuerierHandler()

	msg := &MsgStoreCode{
		Sender:       creator.String(),
		WASMByteCode: testContract,
	}
	res, err := h(data.ctx, msg)
	require.NoError(t, err)
	assertStoreCodeResponse(t, res.Data, 1)

	_, _, bob := keyPubAddr()
	initMsg := initMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	initCmd := MsgInstantiateContract{
		Sender: creator.String(),
		CodeID: firstCodeID,
		Msg:    initMsgBz,
		Funds:  sdk.CoinsToCoinAdapters(deposit),
	}
	res, err = h(data.ctx, &initCmd)
	require.NoError(t, err)
	contractBech32Addr := parseInitResponse(t, res.Data)

	require.Equal(t, "0x5A8D648DEE57b2fc90D98DC17fa887159b69638b", contractBech32Addr)
	// this should be standard x/wasm message event,  init event, plus a bank send event (2), with no custom contract events
	require.Equal(t, 3, len(res.Events), prettyEvents(res.Events))
	require.Equal(t, "message", res.Events[0].Type)
	assertAttribute(t, "module", "wasm", res.Events[0].Attributes[0])
	require.Equal(t, "instantiate", res.Events[1].Type)
	require.Equal(t, "wasm", res.Events[2].Type)
	assertAttribute(t, "_contract_address", contractBech32Addr, res.Events[2].Attributes[0])

	// ensure bob doesn't exist
	bobAcct := data.acctKeeper.GetAccount(data.ctx, sdk.WasmToAccAddress(bob))
	require.Nil(t, bobAcct)

	// ensure funder has reduced balance
	creatorAcct := data.acctKeeper.GetAccount(data.ctx, sdk.WasmToAccAddress(creator))
	require.NotNil(t, creatorAcct)
	// we started at 2*deposit, should have spent one above
	assert.Equal(t, deposit, bank.NewBankKeeperAdapter(data.bankKeeper).GetAllBalances(data.ctx, creatorAcct.GetAddress()))

	// ensure contract has updated balance
	contractAddr, _ := sdk.WasmAddressFromBech32(contractBech32Addr)
	contractAcct := data.acctKeeper.GetAccount(data.ctx, sdk.WasmToAccAddress(contractAddr))
	require.NotNil(t, contractAcct)
	assert.Equal(t, deposit, bank.NewBankKeeperAdapter(data.bankKeeper).GetAllBalances(data.ctx, contractAcct.GetAddress()))

	execCmd := MsgExecuteContract{
		Sender:   fred.String(),
		Contract: contractBech32Addr,
		Msg:      []byte(`{"release":{}}`),
		Funds:    sdk.CoinsToCoinAdapters(topUp),
	}
	res, err = h(data.ctx, &execCmd)
	require.NoError(t, err)
	// from https://github.com/CosmWasm/cosmwasm/blob/master/contracts/hackatom/src/contract.rs#L167
	assertExecuteResponse(t, res.Data, []byte{0xf0, 0x0b, 0xaa})

	// this should be standard message event, plus x/wasm init event, plus 2 bank send event, plus a special event from the contract
	require.Equal(t, 5, len(res.Events), prettyEvents(res.Events))

	assert.Equal(t, "message", res.Events[0].Type)
	assertAttribute(t, "module", "wasm", res.Events[0].Attributes[0])

	assert.Equal(t, "execute", res.Events[1].Type)

	// custom contract event attribute
	assert.Equal(t, "wasm", res.Events[2].Type)
	assertAttribute(t, "_contract_address", contractBech32Addr, res.Events[2].Attributes[0])
	assertAttribute(t, "action", "release", res.Events[2].Attributes[1])
	// custom contract event
	assert.Equal(t, "wasm-hackatom", res.Events[3].Type)
	assertAttribute(t, "_contract_address", contractBech32Addr, res.Events[3].Attributes[0])
	assertAttribute(t, "action", "release", res.Events[3].Attributes[1])
	// second transfer (this without conflicting message)
	assert.Equal(t, "transfer", res.Events[4].Type)
	assertAttribute(t, "recipient", bob.String(), res.Events[4].Attributes[0])
	assertAttribute(t, "sender", contractBech32Addr, res.Events[4].Attributes[1])
	assertAttribute(t, "amount", "105000.000000000000000000denom", res.Events[4].Attributes[2])
	// finally, standard x/wasm tag

	// ensure bob now exists and got both payments released
	bobAcct = data.acctKeeper.GetAccount(data.ctx, sdk.WasmToAccAddress(bob))
	require.NotNil(t, bobAcct)
	balance := bank.NewBankKeeperAdapter(data.bankKeeper).GetAllBalances(data.ctx, bobAcct.GetAddress())
	assert.Equal(t, deposit.Add(topUp...), balance)

	// ensure contract has updated balance

	contractAcct = data.acctKeeper.GetAccount(data.ctx, sdk.WasmToAccAddress(contractAddr))
	require.NotNil(t, contractAcct)
	assert.Equal(t, zeroCoins, bank.NewBankKeeperAdapter(data.bankKeeper).GetAllBalances(data.ctx, contractAcct.GetAddress()))

	// ensure all contract state is as after init
	assertCodeList(t, q, data.ctx, 1)
	assertCodeBytes(t, q, data.ctx, 1, testContract)

	assertContractList(t, q, data.ctx, 1, []string{contractBech32Addr})
	assertContractInfo(t, q, data.ctx, contractBech32Addr, 1, creator)
	assertContractState(t, q, data.ctx, contractBech32Addr, state{
		Verifier:    fred.String(),
		Beneficiary: bob.String(),
		Funder:      creator.String(),
	})
}

func TestHandleExecuteEscrow(t *testing.T) {
	types2.UnittestOnlySetMilestoneEarthHeight(1)
	data := setupTest(t)
	types2.UnittestOnlySetMilestoneEarthHeight(1)
	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	topUp := sdk.NewCoins(sdk.NewInt64Coin("denom", 5000))
	creator := data.faucet.NewFundedAccount(data.ctx, deposit.Add(deposit...)...)
	fred := data.faucet.NewFundedAccount(data.ctx, topUp...)

	h := data.module.NewHandler()

	msg := &MsgStoreCode{
		Sender:       creator.String(),
		WASMByteCode: testContract,
	}
	res, err := h(data.ctx, msg)
	require.NoError(t, err)

	_, _, bob := keyPubAddr()
	initMsg := map[string]interface{}{
		"verifier":    fred.String(),
		"beneficiary": bob.String(),
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	initCmd := MsgInstantiateContract{
		Sender: creator.String(),
		CodeID: firstCodeID,
		Msg:    initMsgBz,
		Funds:  sdk.CoinsToCoinAdapters(deposit),
	}
	res, err = h(data.ctx, &initCmd)
	require.NoError(t, err)
	contractBech32Addr := parseInitResponse(t, res.Data)
	require.Equal(t, "0x5A8D648DEE57b2fc90D98DC17fa887159b69638b", contractBech32Addr)

	handleMsg := map[string]interface{}{
		"release": map[string]interface{}{},
	}
	handleMsgBz, err := json.Marshal(handleMsg)
	require.NoError(t, err)

	execCmd := MsgExecuteContract{
		Sender:   fred.String(),
		Contract: contractBech32Addr,
		Msg:      handleMsgBz,
		Funds:    sdk.CoinsToCoinAdapters(topUp),
	}
	res, err = h(data.ctx, &execCmd)
	require.NoError(t, err)
	// from https://github.com/CosmWasm/cosmwasm/blob/master/contracts/hackatom/src/contract.rs#L167
	assertExecuteResponse(t, res.Data, []byte{0xf0, 0x0b, 0xaa})

	// ensure bob now exists and got both payments released
	bobAcct := data.acctKeeper.GetAccount(data.ctx, sdk.WasmToAccAddress(bob))
	require.NotNil(t, bobAcct)
	balance := bank.NewBankKeeperAdapter(data.bankKeeper).GetAllBalances(data.ctx, bobAcct.GetAddress())
	assert.Equal(t, deposit.Add(topUp...), balance)

	// ensure contract has updated balance
	contractAddr, _ := sdk.WasmAddressFromBech32(contractBech32Addr)
	contractAcct := data.acctKeeper.GetAccount(data.ctx, sdk.WasmToAccAddress(contractAddr))
	require.NotNil(t, contractAcct)
	assert.Equal(t, zeroCoins, bank.NewBankKeeperAdapter(data.bankKeeper).GetAllBalances(data.ctx, contractAcct.GetAddress()))
}

func TestReadWasmConfig(t *testing.T) {
	defaults := DefaultWasmConfig()
	specs := map[string]struct {
		src AppOptionsMock
		exp types.WasmConfig
	}{
		"set query gas limit via opts": {
			src: AppOptionsMock{
				"wasm.query_gas_limit": 1,
			},
			exp: types.WasmConfig{
				SimulationGasLimit: defaults.SimulationGasLimit,
				SmartQueryGasLimit: 1,
				MemoryCacheSize:    defaults.MemoryCacheSize,
			},
		},
		"set cache via opts": {
			src: AppOptionsMock{
				"wasm.memory_cache_size": 2,
			},
			exp: types.WasmConfig{
				SimulationGasLimit: defaults.SimulationGasLimit,
				MemoryCacheSize:    2,
				SmartQueryGasLimit: defaults.SmartQueryGasLimit,
			},
		},
		"set debug via opts": {
			src: AppOptionsMock{
				"trace": true,
			},
			exp: types.WasmConfig{
				SimulationGasLimit: defaults.SimulationGasLimit,
				SmartQueryGasLimit: defaults.SmartQueryGasLimit,
				MemoryCacheSize:    defaults.MemoryCacheSize,
				ContractDebugMode:  true,
			},
		},
		"all defaults when no options set": {
			exp: defaults,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			viper.Reset()
			for k, v := range spec.src {
				viper.Set(k, v)
			}
			got, err := ReadWasmConfig()
			require.NoError(t, err)
			assert.Equal(t, spec.exp, got)
			viper.Reset()
		})
	}
}

type AppOptionsMock map[string]interface{}

func (a AppOptionsMock) Get(s string) interface{} {
	return a[s]
}

type prettyEvent struct {
	Type string
	Attr []sdk.Attribute
}

func prettyEvents(evts []sdk.Event) string {
	res := make([]prettyEvent, len(evts))
	for i, e := range evts {
		res[i] = prettyEvent{
			Type: e.Type,
			Attr: prettyAttrs(e.Attributes),
		}
	}
	bz, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(bz)
}

func prettyAttrs(attrs []kv.Pair) []sdk.Attribute {
	pretty := make([]sdk.Attribute, len(attrs))
	for i, a := range attrs {
		pretty[i] = prettyAttr(a)
	}
	return pretty
}

func prettyAttr(attr kv.Pair) sdk.Attribute {
	return sdk.NewAttribute(string(attr.Key), string(attr.Value))
}

func assertAttribute(t *testing.T, key string, value string, attr kv.Pair) {
	t.Helper()
	assert.Equal(t, key, string(attr.Key), prettyAttr(attr))
	assert.Equal(t, value, string(attr.Value), prettyAttr(attr))
}

func assertCodeList(t *testing.T, q sdk.Querier, ctx sdk.Context, expectedNum int) {
	bz, sdkerr := q(ctx, []string{QueryListCode}, abci.RequestQuery{})
	require.NoError(t, sdkerr)

	if len(bz) == 0 {
		require.Equal(t, expectedNum, 0)
		return
	}

	var res []CodeInfo
	err := json.Unmarshal(bz, &res)
	require.NoError(t, err)

	assert.Equal(t, expectedNum, len(res))
}

func assertCodeBytes(t *testing.T, q sdk.Querier, ctx sdk.Context, codeID uint64, expectedBytes []byte) {
	path := []string{QueryGetCode, fmt.Sprintf("%d", codeID)}
	bz, sdkerr := q(ctx, path, abci.RequestQuery{})
	require.NoError(t, sdkerr)

	if len(expectedBytes) == 0 {
		require.Equal(t, len(bz), 0, "%q", string(bz))
		return
	}
	var res map[string]interface{}
	err := json.Unmarshal(bz, &res)
	require.NoError(t, err)

	require.Contains(t, res, "data")
	b, err := base64url.Decode(res["data"].(string))
	require.NoError(t, err)
	assert.Equal(t, expectedBytes, b)
	assert.EqualValues(t, codeID, res["id"])
}

func assertContractList(t *testing.T, q sdk.Querier, ctx sdk.Context, codeID uint64, expContractAddrs []string) {
	bz, sdkerr := q(ctx, []string{QueryListContractByCode, fmt.Sprintf("%d", codeID)}, abci.RequestQuery{})
	require.NoError(t, sdkerr)

	if len(bz) == 0 {
		require.Equal(t, len(expContractAddrs), 0)
		return
	}

	var res []string
	err := json.Unmarshal(bz, &res)
	require.NoError(t, err)

	hasAddrs := make([]string, len(res))
	for i, r := range res {
		hasAddrs[i] = r
	}

	assert.Equal(t, expContractAddrs, hasAddrs)
}

func assertContractState(t *testing.T, q sdk.Querier, ctx sdk.Context, contractBech32Addr string, expected state) {
	t.Helper()
	path := []string{QueryGetContractState, contractBech32Addr, keeper.QueryMethodContractStateAll}
	bz, sdkerr := q(ctx, path, abci.RequestQuery{})
	require.NoError(t, sdkerr)

	var res []Model
	err := json.Unmarshal(bz, &res)
	require.NoError(t, err)
	require.Equal(t, 1, len(res), "#v", res)
	require.Equal(t, []byte("config"), []byte(res[0].Key))

	expectedBz, err := json.Marshal(expected)
	require.NoError(t, err)
	assert.Equal(t, expectedBz, res[0].Value)
}

func assertContractInfo(t *testing.T, q sdk.Querier, ctx sdk.Context, contractBech32Addr string, codeID uint64, creator sdk.WasmAddress) {
	t.Helper()
	path := []string{QueryGetContract, contractBech32Addr}
	bz, sdkerr := q(ctx, path, abci.RequestQuery{})
	require.NoError(t, sdkerr)

	var res ContractInfo
	err := json.Unmarshal(bz, &res)
	require.NoError(t, err)

	assert.Equal(t, codeID, res.CodeID)
	assert.Equal(t, creator.String(), res.Creator)
}
