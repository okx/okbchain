package watcher_test

import (
	"encoding/hex"
	"fmt"
	ethcommon "github.com/ethereum/go-ethereum/common"
	chaincodec "github.com/okx/okbchain/app/codec"
	"github.com/okx/okbchain/libs/cosmos-sdk/types/module"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth"
	"github.com/okx/okbchain/libs/tendermint/libs/log"

	"github.com/ethereum/go-ethereum/common"
	ethcmn "github.com/ethereum/go-ethereum/common"
	jsoniter "github.com/json-iterator/go"
	"github.com/okx/okbchain/app"
	"github.com/okx/okbchain/app/crypto/ethsecp256k1"
	ethermint "github.com/okx/okbchain/app/types"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	abci "github.com/okx/okbchain/libs/tendermint/abci/types"
	"github.com/okx/okbchain/libs/tendermint/crypto/secp256k1"
	"github.com/okx/okbchain/libs/tendermint/crypto/tmhash"
	"github.com/okx/okbchain/x/evm"
	"github.com/okx/okbchain/x/evm/types"
	evmtypes "github.com/okx/okbchain/x/evm/types"
	"github.com/okx/okbchain/x/evm/watcher"
	"github.com/spf13/viper"
	"github.com/status-im/keycard-go/hexutils"
	"github.com/stretchr/testify/require"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type KV struct {
	k []byte
	v []byte
}

func calcHash(kvs []KV) []byte {
	ha := tmhash.New()
	// calc a hash
	for _, kv := range kvs {
		ha.Write(kv.k)
		ha.Write(kv.v)
	}
	return ha.Sum(nil)
}

type WatcherTestSt struct {
	ctx     sdk.Context
	app     *app.OKBChainApp
	handler sdk.Handler
}

func setupTest() *WatcherTestSt {
	w := &WatcherTestSt{}
	checkTx := false
	chain_id := "ethermint-3"
	viper.Set(watcher.FlagFastQuery, true)
	viper.Set(sdk.FlagDBBackend, "memdb")
	viper.Set(watcher.FlagCheckWd, true)

	w.app = app.Setup(checkTx)
	w.ctx = w.app.BaseApp.NewContext(checkTx, abci.Header{Height: 1, ChainID: chain_id, Time: time.Now().UTC()})
	w.ctx.SetDeliver()
	w.handler = evm.NewHandler(w.app.EvmKeeper)

	ethermint.SetChainId(chain_id)

	params := types.DefaultParams()
	params.EnableCreate = true
	params.EnableCall = true
	w.app.EvmKeeper.SetParams(w.ctx, params)
	return w
}

func getDBKV(db *watcher.WatchStore) []KV {
	var kvs []KV
	it := db.Iterator(nil, nil)
	for it.Valid() {
		kvs = append(kvs, KV{it.Key(), it.Value()})
		it.Next()
	}
	return kvs
}

func flushDB(db *watcher.WatchStore) {
	it := db.Iterator(nil, nil)
	for it.Valid() {
		db.Delete(it.Key())
		it.Next()
	}
}

func checkWD(wdBytes []byte, w *WatcherTestSt) {
	wd := watcher.WatchData{}
	if err := wd.UnmarshalFromAmino(nil, wdBytes); err != nil {
		return
	}
	keys := make([][]byte, len(wd.Batches))
	for i, b := range wd.Batches {
		keys[i] = b.Key
	}
	w.app.EvmKeeper.Watcher.CheckWatchDB(keys, "producer--test")
}

func testWatchData(t *testing.T, w *WatcherTestSt) {
	// produce WatchData
	w.app.EvmKeeper.Watcher.Commit()
	time.Sleep(time.Millisecond)

	// get WatchData
	wdFunc := w.app.EvmKeeper.Watcher.CreateWatchDataGenerator()
	wd, err := wdFunc()
	require.Nil(t, err)
	require.NotEmpty(t, wd)

	store := watcher.InstanceOfWatchStore()
	pWd := getDBKV(store)
	checkWD(wd, w)
	flushDB(store)

	// use WatchData
	wData, err := w.app.EvmKeeper.Watcher.UnmarshalWatchData(wd)
	require.Nil(t, err)
	w.app.EvmKeeper.Watcher.ApplyWatchData(wData)
	time.Sleep(time.Millisecond)

	cWd := getDBKV(store)

	// compare db_kv of producer and consumer
	require.Equal(t, pWd, cWd, "compare len:", "pwd:", len(pWd), "cwd", len(cWd))
	pHash := calcHash(pWd)
	cHash := calcHash(cWd)
	require.NotEmpty(t, pHash)
	require.NotEmpty(t, cHash)
	require.Equal(t, pHash, cHash)

	flushDB(store)
}

func TestHandleMsgEthereumTx(t *testing.T) {
	w := setupTest()
	privkey, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	sender := ethcmn.HexToAddress(privkey.PubKey().Address().String())

	var tx *types.MsgEthereumTx

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"passed",
			func() {
				w.app.EvmKeeper.SetBalance(w.ctx, sender, big.NewInt(100))
				tx = types.NewMsgEthereumTx(0, &sender, big.NewInt(100), 3000000, big.NewInt(1), nil)

				// parse context chain ID to big.Int
				chainID, err := ethermint.ParseChainID(w.ctx.ChainID())
				require.NoError(t, err)

				// sign transaction
				err = tx.Sign(chainID, privkey.ToECDSA())
				require.NoError(t, err)
			},
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			w = setupTest() // reset
			//nolint
			tc.malleate()
			w.ctx.SetGasMeter(sdk.NewInfiniteGasMeter())
			res, err := w.handler(w.ctx, tx)

			//nolint
			if tc.expPass {
				require.NoError(t, err)
				require.NotNil(t, res)
				var expectedConsumedGas uint64 = 21000
				require.EqualValues(t, expectedConsumedGas, w.ctx.GasMeter().GasConsumed())
			} else {
				require.Error(t, err)
				require.Nil(t, res)
			}

			testWatchData(t, w)
		})
	}
}

func TestMsgEthereumTxByWatcher(t *testing.T) {
	var (
		tx   *types.MsgEthereumTx
		from = ethcmn.BytesToAddress(secp256k1.GenPrivKey().PubKey().Address())
		to   = ethcmn.BytesToAddress(secp256k1.GenPrivKey().PubKey().Address())
	)
	w := setupTest()
	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"passed",
			func() {
				tx = types.NewMsgEthereumTx(0, &to, big.NewInt(1), 100000, big.NewInt(2), []byte("test"))
				w.app.EvmKeeper.SetBalance(w.ctx, ethcmn.BytesToAddress(from.Bytes()), big.NewInt(100))
			},
			true,
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			w = setupTest() // reset
			//nolint
			tc.malleate()
			w.ctx.SetIsCheckTx(true)
			w.ctx.SetGasMeter(sdk.NewInfiniteGasMeter())
			w.ctx.SetFrom(from.String())
			res, err := w.handler(w.ctx, tx)

			//nolint
			if tc.expPass {
				require.NoError(t, err)
				require.NotNil(t, res)
				var expectedConsumedGas uint64 = 21064
				require.EqualValues(t, expectedConsumedGas, w.ctx.GasMeter().GasConsumed())
			} else {
				require.Error(t, err)
				require.Nil(t, res)
			}

			testWatchData(t, w)
		})
	}
}

func TestDeployAndCallContract(t *testing.T) {
	w := setupTest()

	// Deploy contract - Owner.sol
	gasLimit := uint64(100000000)
	gasPrice := big.NewInt(10000)

	priv, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err, "failed to create key")

	sender := ethcmn.HexToAddress(priv.PubKey().Address().String())
	w.app.EvmKeeper.SetBalance(w.ctx, sender, big.NewInt(100))

	bytecode := common.FromHex("0x608060405234801561001057600080fd5b50336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16600073ffffffffffffffffffffffffffffffffffffffff167f342827c97908e5e2f71151c08502a66d44b6f758e3ac2f1de95f02eb95f0a73560405160405180910390a36102c4806100dc6000396000f3fe608060405234801561001057600080fd5b5060043610610053576000357c010000000000000000000000000000000000000000000000000000000090048063893d20e814610058578063a6f9dae1146100a2575b600080fd5b6100606100e6565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6100e4600480360360208110156100b857600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919050505061010f565b005b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146101d1576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260138152602001807f43616c6c6572206973206e6f74206f776e65720000000000000000000000000081525060200191505060405180910390fd5b8073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f342827c97908e5e2f71151c08502a66d44b6f758e3ac2f1de95f02eb95f0a73560405160405180910390a3806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505056fea265627a7a72315820f397f2733a89198bc7fed0764083694c5b828791f39ebcbc9e414bccef14b48064736f6c63430005100032")
	tx := types.NewMsgEthereumTx(1, &sender, big.NewInt(0), gasLimit, gasPrice, bytecode)
	tx.Sign(big.NewInt(3), priv.ToECDSA())
	require.NoError(t, err)

	result, err := w.handler(w.ctx, tx)
	require.NoError(t, err, "failed to handle eth tx msg")

	resultData, err := types.DecodeResultData(result.Data)
	require.NoError(t, err, "failed to decode result data")

	testWatchData(t, w)

	// store - changeOwner
	gasLimit = uint64(100000000000)
	gasPrice = big.NewInt(100)
	receiver := common.HexToAddress(resultData.ContractAddress.String())

	storeAddr := "0xa6f9dae10000000000000000000000006a82e4a67715c8412a9114fbd2cbaefbc8181424"
	bytecode = common.FromHex(storeAddr)
	tx = types.NewMsgEthereumTx(2, &receiver, big.NewInt(0), gasLimit, gasPrice, bytecode)
	tx.Sign(big.NewInt(3), priv.ToECDSA())
	require.NoError(t, err)

	result, err = w.handler(w.ctx, tx)
	require.NoError(t, err, "failed to handle eth tx msg")

	resultData, err = types.DecodeResultData(result.Data)
	require.NoError(t, err, "failed to decode result data")

	testWatchData(t, w)

	// query - getOwner
	bytecode = common.FromHex("0x893d20e8")
	tx = types.NewMsgEthereumTx(2, &receiver, big.NewInt(0), gasLimit, gasPrice, bytecode)
	tx.Sign(big.NewInt(3), priv.ToECDSA())
	require.NoError(t, err)

	result, err = w.handler(w.ctx, tx)
	require.NoError(t, err, "failed to handle eth tx msg")

	resultData, err = types.DecodeResultData(result.Data)
	require.NoError(t, err, "failed to decode result data")

	getAddr := strings.ToLower(hexutils.BytesToHex(resultData.Ret))
	require.Equal(t, true, strings.HasSuffix(storeAddr, getAddr), "Fail to query the address")

	testWatchData(t, w)
}

type mockDuplicateAccount struct {
	*auth.BaseAccount
	Addr byte
	Seq  byte
}

func (a *mockDuplicateAccount) GetAddress() sdk.AccAddress {
	return []byte{a.Addr}
}

func newMockAccount(byteAddr, seq byte) *mockDuplicateAccount {
	ret := &mockDuplicateAccount{Addr: byteAddr, Seq: seq}
	pubkey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubkey.Address())
	baseAcc := auth.NewBaseAccount(addr, nil, pubkey, 0, 0)
	ret.BaseAccount = baseAcc
	return ret
}

func TestDuplicateAddress(t *testing.T) {
	accAdds := make([]*sdk.AccAddress, 0)
	for i := 0; i < 10; i++ {
		adds := hex.EncodeToString([]byte(fmt.Sprintf("addr-%d", i)))
		a, _ := sdk.AccAddressFromHex(adds)
		accAdds = append(accAdds, &a)
	}
	adds := hex.EncodeToString([]byte(fmt.Sprintf("addr-%d", 1)))
	a, _ := sdk.AccAddressFromHex(adds)
	accAdds = append(accAdds, &a)
	filterM := make(map[string]struct{})
	count := 0
	for _, add := range accAdds {
		_, exist := filterM[string(add.Bytes())]
		if exist {
			count++
			continue
		}
		filterM[string(add.Bytes())] = struct{}{}
	}
	require.Equal(t, 1, count)
}

func TestDuplicateWatchMessage(t *testing.T) {
	w := setupTest()
	w.app.EvmKeeper.Watcher.NewHeight(1, common.Hash{}, abci.Header{Height: 1})
	// init store
	store := watcher.InstanceOfWatchStore()
	flushDB(store)
	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	balance := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(1)))
	a1 := &ethermint.EthAccount{
		BaseAccount: auth.NewBaseAccount(addr, balance, pubKey, 1, 1),
		CodeHash:    ethcrypto.Keccak256(nil),
	}
	w.app.EvmKeeper.Watcher.SaveAccount(a1)
	a2 := &ethermint.EthAccount{
		BaseAccount: auth.NewBaseAccount(addr, balance, pubKey, 1, 2),
		CodeHash:    ethcrypto.Keccak256(nil),
	}
	w.app.EvmKeeper.Watcher.SaveAccount(a2)
	w.app.EvmKeeper.Watcher.Commit()
	time.Sleep(time.Millisecond)
	pWd := getDBKV(store)
	require.Equal(t, 1, len(pWd))
}

func TestWriteLatestMsg(t *testing.T) {
	viper.Set(watcher.FlagFastQuery, true)
	viper.Set(sdk.FlagDBBackend, "memdb")
	w := watcher.NewWatcher(log.NewTMLogger(os.Stdout))
	w.SetWatchDataManager()
	w.NewHeight(1, common.Hash{}, abci.Header{Height: 1})
	// init store
	store := watcher.InstanceOfWatchStore()
	flushDB(store)

	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	balance := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(1)))
	a1 := &ethermint.EthAccount{
		BaseAccount: auth.NewBaseAccount(addr, balance, pubKey, 1, 1),
		CodeHash:    ethcrypto.Keccak256(nil),
	}
	a11 := &ethermint.EthAccount{
		BaseAccount: auth.NewBaseAccount(addr, balance, pubKey, 1, 2),
		CodeHash:    ethcrypto.Keccak256(nil),
	}
	a111 := &ethermint.EthAccount{
		BaseAccount: auth.NewBaseAccount(addr, balance, pubKey, 1, 3),
		CodeHash:    ethcrypto.Keccak256(nil),
	}

	w.SaveAccount(a1)
	w.SaveAccount(a11)
	w.SaveAccount(a111)
	// waiting 1 second for initializing jobChan
	time.Sleep(time.Millisecond)
	w.Commit()
	time.Sleep(time.Millisecond)
	pWd := getDBKV(store)
	require.Equal(t, 1, len(pWd))

	m := watcher.NewMsgAccount(a1)
	v, err := store.Get(m.GetKey())
	require.NoError(t, err)
	has := store.Has(m.GetKey())
	require.Equal(t, has, true)
	ethAccount, err := watcher.DecodeAccount(v)
	require.NoError(t, err)
	//test decode error
	vv := v[:1]
	_, err = watcher.DecodeAccount(vv)
	require.Error(t, err)
	require.Equal(t, uint64(3), ethAccount.GetSequence())
	p := store.GetEvmParams()
	err = ParamsDeepEqual(evmtypes.DefaultParams(), p)
	require.NoError(t, err)

	expectedParams2 := evmtypes.Params{
		EnableCreate:                      true,
		EnableCall:                        true,
		EnableContractDeploymentWhitelist: true,
		EnableContractBlockedList:         true,
		MaxGasLimitPerTx:                  20000000,
	}
	store.SetEvmParams(expectedParams2)
	p = store.GetEvmParams()
	err = ParamsDeepEqual(p, expectedParams2)
	require.NoError(t, err)
}

func ParamsDeepEqual(src, dst evmtypes.Params) error {
	if src.EnableCreate != dst.EnableCreate ||
		src.EnableCall != dst.EnableCall ||
		src.EnableContractDeploymentWhitelist != dst.EnableContractDeploymentWhitelist ||
		src.EnableContractBlockedList != dst.EnableContractBlockedList {
		return fmt.Errorf("params not fit")
	}
	return nil
}

func TestDeliverRealTx(t *testing.T) {
	w := setupTest()
	bytecode := ethcommon.FromHex("0x12")
	tx := evmtypes.NewMsgEthereumTx(0, nil, big.NewInt(0), uint64(1000000), big.NewInt(10000), bytecode)
	privKey, _ := ethsecp256k1.GenerateKey()
	err := tx.Sign(big.NewInt(3), privKey.ToECDSA())
	require.NoError(t, err)
	codecProxy, _ := chaincodec.MakeCodecSuit(module.NewBasicManager())
	w.app.EvmKeeper.Watcher.RecordTxAndFailedReceipt(tx, nil, evm.TxDecoderWithHash(codecProxy))
}

func TestBaiscDBOpt(t *testing.T) {
	viper.Set(watcher.FlagFastQuery, true)
	viper.Set(sdk.FlagDBBackend, "memdb")
	store := watcher.InstanceOfWatchStore()
	store.Set([]byte("test01"), []byte("value01"))
	v, err := store.Get([]byte("test01"))
	require.NoError(t, err)
	require.Equal(t, v, []byte("value01"))
	v, err = store.Get([]byte("test no key"))
	require.Equal(t, v, []byte(nil))
	require.NoError(t, err)
	r, err := store.GetUnsafe([]byte("test01"), func(value []byte) (interface{}, error) {
		return nil, nil
	})
	require.Equal(t, r, nil)
	require.NoError(t, err)
}
