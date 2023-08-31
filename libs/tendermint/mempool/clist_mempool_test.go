package mempool

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math/big"
	mrand "math/rand"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/okx/brczero/libs/tendermint/libs/clist"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	amino "github.com/tendermint/go-amino"

	"github.com/okx/brczero/libs/tendermint/abci/example/counter"
	"github.com/okx/brczero/libs/tendermint/abci/example/kvstore"
	abci "github.com/okx/brczero/libs/tendermint/abci/types"
	cfg "github.com/okx/brczero/libs/tendermint/config"
	"github.com/okx/brczero/libs/tendermint/libs/log"
	tmrand "github.com/okx/brczero/libs/tendermint/libs/rand"
	"github.com/okx/brczero/libs/tendermint/proxy"
	"github.com/okx/brczero/libs/tendermint/types"
)

const (
	BlockMaxTxNum = 300
)

// A cleanupFunc cleans up any config / test files created for a particular
// test.
type cleanupFunc func()

func newMempoolWithApp(cc proxy.ClientCreator) (*CListMempool, cleanupFunc) {
	return newMempoolWithAppAndConfig(cc, cfg.ResetTestRoot("mempool_test"))
}

func newMempoolWithAppAndConfig(cc proxy.ClientCreator, config *cfg.Config) (*CListMempool, cleanupFunc) {
	appConnMem, _ := cc.NewABCIClient()
	appConnMem.SetLogger(log.TestingLogger().With("module", "abci-client", "connection", "mempool"))
	err := appConnMem.Start()
	if err != nil {
		panic(err)
	}
	mempool := NewCListMempool(config.Mempool, appConnMem, 0)
	mempool.SetLogger(log.TestingLogger())
	return mempool, func() { os.RemoveAll(config.RootDir) }
}

func ensureNoFire(t *testing.T, ch <-chan struct{}, timeoutMS int) {
	timer := time.NewTimer(time.Duration(timeoutMS) * time.Millisecond)
	select {
	case <-ch:
		t.Fatal("Expected not to fire")
	case <-timer.C:
	}
}

func ensureFire(t *testing.T, ch <-chan struct{}, timeoutMS int) {
	timer := time.NewTimer(time.Duration(timeoutMS) * time.Millisecond)
	select {
	case <-ch:
	case <-timer.C:
		t.Fatal("Expected to fire")
	}
}

func checkTxs(t *testing.T, mempool Mempool, count int, peerID uint16) types.Txs {
	txs := make(types.Txs, count)
	txInfo := TxInfo{SenderID: peerID}
	for i := 0; i < count; i++ {
		txBytes := make([]byte, 20)
		txs[i] = txBytes
		_, err := rand.Read(txBytes)
		if err != nil {
			t.Error(err)
		}
		if err := mempool.CheckTx(txBytes, nil, txInfo); err != nil {
			// Skip invalid txs.
			// TestMempoolFilters will fail otherwise. It asserts a number of txs
			// returned.
			if IsPreCheckError(err) {
				continue
			}
			t.Fatalf("CheckTx failed: %v while checking #%d tx", err, i)
		}
	}
	return txs
}

func TestReapMaxBytesMaxGas(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempool, cleanup := newMempoolWithApp(cc)
	defer cleanup()

	// Ensure gas calculation behaves as expected
	checkTxs(t, mempool, 1, UnknownPeerID)
	tx0 := mempool.TxsFront().Value.(*mempoolTx)
	// assert that kv store has gas wanted = 1.
	require.Equal(t, app.CheckTx(abci.RequestCheckTx{Tx: tx0.tx}).GasWanted, int64(1), "KVStore had a gas value neq to 1")
	require.Equal(t, tx0.gasWanted, int64(1), "transactions gas was set incorrectly")
	// ensure each tx is 20 bytes long
	require.Equal(t, len(tx0.tx), 20, "Tx is longer than 20 bytes")
	mempool.Flush()

	// each table driven test creates numTxsToCreate txs with checkTx, and at the end clears all remaining txs.
	// each tx has 20 bytes + amino overhead = 21 bytes, 1 gas
	tests := []struct {
		numTxsToCreate int
		maxBytes       int64
		maxGas         int64
		expectedNumTxs int
	}{
		{20, -1, -1, 20},
		{20, -1, 0, 1},
		{20, -1, 10, 10},
		{20, -1, 30, 20},
		{20, 0, -1, 0},
		{20, 0, 10, 0},
		{20, 10, 10, 0},
		{20, 22, 10, 1},
		{20, 220, -1, 10},
		{20, 220, 5, 5},
		{20, 220, 10, 10},
		{20, 220, 15, 10},
		{20, 20000, -1, 20},
		{20, 20000, 5, 5},
		{20, 20000, 30, 20},
		{2000, -1, -1, 300},
	}
	for tcIndex, tt := range tests {
		checkTxs(t, mempool, tt.numTxsToCreate, UnknownPeerID)
		got := mempool.ReapMaxBytesMaxGas(tt.maxBytes, tt.maxGas)
		assert.Equal(t, tt.expectedNumTxs, len(got), "Got %d txs, expected %d, tc #%d",
			len(got), tt.expectedNumTxs, tcIndex)
		mempool.Flush()
	}
}

func TestMempoolFilters(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempool, cleanup := newMempoolWithApp(cc)
	defer cleanup()
	emptyTxArr := []types.Tx{[]byte{}}

	nopPreFilter := func(tx types.Tx) error { return nil }
	nopPostFilter := func(tx types.Tx, res *abci.ResponseCheckTx) error { return nil }

	// each table driven test creates numTxsToCreate txs with checkTx, and at the end clears all remaining txs.
	// each tx has 20 bytes + amino overhead = 21 bytes, 1 gas
	tests := []struct {
		numTxsToCreate int
		preFilter      PreCheckFunc
		postFilter     PostCheckFunc
		expectedNumTxs int
	}{
		{10, nopPreFilter, nopPostFilter, 10},
		{10, PreCheckAminoMaxBytes(10), nopPostFilter, 0},
		{10, PreCheckAminoMaxBytes(20), nopPostFilter, 0},
		{10, PreCheckAminoMaxBytes(22), nopPostFilter, 10},
		{10, nopPreFilter, PostCheckMaxGas(-1), 10},
		{10, nopPreFilter, PostCheckMaxGas(0), 0},
		{10, nopPreFilter, PostCheckMaxGas(1), 10},
		{10, nopPreFilter, PostCheckMaxGas(3000), 10},
		{10, PreCheckAminoMaxBytes(10), PostCheckMaxGas(20), 0},
		{10, PreCheckAminoMaxBytes(30), PostCheckMaxGas(20), 10},
		{10, PreCheckAminoMaxBytes(22), PostCheckMaxGas(1), 10},
		{10, PreCheckAminoMaxBytes(22), PostCheckMaxGas(0), 0},
	}
	for tcIndex, tt := range tests {
		mempool.Update(1, emptyTxArr, abciResponses(len(emptyTxArr), abci.CodeTypeOK), tt.preFilter, tt.postFilter)
		checkTxs(t, mempool, tt.numTxsToCreate, UnknownPeerID)
		require.Equal(t, tt.expectedNumTxs, mempool.Size(), "mempool had the incorrect size, on test case %d", tcIndex)
		mempool.Flush()
	}
}

func TestMempoolUpdate(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempool, cleanup := newMempoolWithApp(cc)
	defer cleanup()

	// 1. Adds valid txs to the cache
	{
		mempool.Update(1, []types.Tx{[]byte{0x01}}, abciResponses(1, abci.CodeTypeOK), nil, nil)
		err := mempool.CheckTx([]byte{0x01}, nil, TxInfo{})
		if assert.Error(t, err) {
			assert.Equal(t, ErrTxInCache, err)
		}
	}

	// 2. Removes valid txs from the mempool
	{
		err := mempool.CheckTx([]byte{0x02}, nil, TxInfo{})
		require.NoError(t, err)
		mempool.Update(1, []types.Tx{[]byte{0x02}}, abciResponses(1, abci.CodeTypeOK), nil, nil)
		assert.Zero(t, mempool.Size())
	}

	// 3. Removes invalid transactions from the cache and the mempool (if present)
	{
		err := mempool.CheckTx([]byte{0x03}, nil, TxInfo{})
		require.NoError(t, err)
		mempool.Update(1, []types.Tx{[]byte{0x03}}, abciResponses(1, 1), nil, nil)
		assert.Zero(t, mempool.Size())

		err = mempool.CheckTx([]byte{0x03}, nil, TxInfo{})
		assert.NoError(t, err)
	}
}

func TestTxsAvailable(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempool, cleanup := newMempoolWithApp(cc)
	defer cleanup()
	mempool.EnableTxsAvailable()

	timeoutMS := 500

	// with no txs, it shouldnt fire
	ensureNoFire(t, mempool.TxsAvailable(), timeoutMS)

	// send a bunch of txs, it should only fire once
	txs := checkTxs(t, mempool, 100, UnknownPeerID)
	ensureFire(t, mempool.TxsAvailable(), timeoutMS)
	ensureNoFire(t, mempool.TxsAvailable(), timeoutMS)

	// call update with half the txs.
	// it should fire once now for the new height
	// since there are still txs left
	committedTxs, txs := txs[:50], txs[50:]
	if err := mempool.Update(1, committedTxs, abciResponses(len(committedTxs), abci.CodeTypeOK), nil, nil); err != nil {
		t.Error(err)
	}
	ensureFire(t, mempool.TxsAvailable(), timeoutMS)
	ensureNoFire(t, mempool.TxsAvailable(), timeoutMS)

	// send a bunch more txs. we already fired for this height so it shouldnt fire again
	moreTxs := checkTxs(t, mempool, 50, UnknownPeerID)
	ensureNoFire(t, mempool.TxsAvailable(), timeoutMS)

	// now call update with all the txs. it should not fire as there are no txs left
	committedTxs = append(txs, moreTxs...) //nolint: gocritic
	if err := mempool.Update(2, committedTxs, abciResponses(len(committedTxs), abci.CodeTypeOK), nil, nil); err != nil {
		t.Error(err)
	}
	ensureNoFire(t, mempool.TxsAvailable(), timeoutMS)

	// send a bunch more txs, it should only fire once
	checkTxs(t, mempool, 100, UnknownPeerID)
	ensureFire(t, mempool.TxsAvailable(), timeoutMS)
	ensureNoFire(t, mempool.TxsAvailable(), timeoutMS)
}

func TestSerialReap(t *testing.T) {
	app := counter.NewApplication(true)
	app.SetOption(abci.RequestSetOption{Key: "serial", Value: "on"})
	cc := proxy.NewLocalClientCreator(app)

	mempool, cleanup := newMempoolWithApp(cc)
	defer cleanup()
	mempool.config.MaxTxNumPerBlock = 10000

	appConnCon, _ := cc.NewABCIClient()
	appConnCon.SetLogger(log.TestingLogger().With("module", "abci-client", "connection", "consensus"))
	err := appConnCon.Start()
	require.Nil(t, err)

	cacheMap := make(map[string]struct{})
	deliverTxsRange := func(start, end int) {
		// Deliver some txs.
		for i := start; i < end; i++ {

			// This will succeed
			txBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(txBytes, uint64(i))
			err := mempool.CheckTx(txBytes, nil, TxInfo{})
			_, cached := cacheMap[string(txBytes)]
			if cached {
				require.NotNil(t, err, "expected error for cached tx")
			} else {
				require.Nil(t, err, "expected no err for uncached tx")
			}
			cacheMap[string(txBytes)] = struct{}{}

			// Duplicates are cached and should return error
			err = mempool.CheckTx(txBytes, nil, TxInfo{})
			require.NotNil(t, err, "Expected error after CheckTx on duplicated tx")
		}
	}

	reapCheck := func(exp int) {
		txs := mempool.ReapMaxBytesMaxGas(-1, -1)
		require.Equal(t, len(txs), exp, fmt.Sprintf("Expected to reap %v txs but got %v", exp, len(txs)))
	}

	updateRange := func(start, end int) {
		txs := make([]types.Tx, 0)
		for i := start; i < end; i++ {
			txBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(txBytes, uint64(i))
			txs = append(txs, txBytes)
		}
		if err := mempool.Update(0, txs, abciResponses(len(txs), abci.CodeTypeOK), nil, nil); err != nil {
			t.Error(err)
		}
	}

	commitRange := func(start, end int) {
		// Deliver some txs.
		for i := start; i < end; i++ {
			txBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(txBytes, uint64(i))
			res, err := appConnCon.DeliverTxSync(abci.RequestDeliverTx{Tx: txBytes})
			if err != nil {
				t.Errorf("client error committing tx: %v", err)
			}
			if res.IsErr() {
				t.Errorf("error committing tx. Code:%v result:%X log:%v",
					res.Code, res.Data, res.Log)
			}
		}
		res, err := appConnCon.CommitSync(abci.RequestCommit{})
		if err != nil {
			t.Errorf("client error committing: %v", err)
		}
		if len(res.Data) != 8 {
			t.Errorf("error committing. Hash:%X", res.Data)
		}
	}

	//----------------------------------------

	// Deliver some txs.
	deliverTxsRange(0, 100)

	// Reap the txs.
	reapCheck(100)

	// Reap again.  We should get the same amount
	reapCheck(100)

	// Deliver 0 to 999, we should reap 900 new txs
	// because 100 were already counted.
	deliverTxsRange(0, 1000)

	// Reap the txs.
	reapCheck(BlockMaxTxNum)

	// Reap again.  We should get the same amount
	reapCheck(BlockMaxTxNum)

	// Commit from the conensus AppConn
	commitRange(0, BlockMaxTxNum)
	updateRange(0, BlockMaxTxNum)

	// We should have 500 left.
	reapCheck(BlockMaxTxNum)

	// Deliver 100 invalid txs and 100 valid txs
	deliverTxsRange(900, 1100)

	// We should have 300 now.
	reapCheck(BlockMaxTxNum)
}

// Size of the amino encoded TxMessage is the length of the
// encoded byte array, plus 1 for the struct field, plus 4
// for the amino prefix.
func txMessageSize(tx types.Tx) int {
	return amino.ByteSliceSize(tx) + 1 + 4
}

func TestMempoolMaxMsgSize(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempl, cleanup := newMempoolWithApp(cc)
	defer cleanup()

	maxTxSize := mempl.config.MaxTxBytes
	maxMsgSize := calcMaxMsgSize(maxTxSize)

	testCases := []struct {
		len int
		err bool
	}{
		// check small txs. no error
		{10, false},
		{1000, false},
		{1000000, false},

		// check around maxTxSize
		// changes from no error to error
		{maxTxSize - 2, false},
		{maxTxSize - 1, false},
		{maxTxSize, false},
		{maxTxSize + 1, true},
		{maxTxSize + 2, true},

		// check around maxMsgSize. all error
		{maxMsgSize - 1, true},
		{maxMsgSize, true},
		{maxMsgSize + 1, true},
	}

	for i, testCase := range testCases {
		caseString := fmt.Sprintf("case %d, len %d", i, testCase.len)

		tx := tmrand.Bytes(testCase.len)
		err := mempl.CheckTx(tx, nil, TxInfo{})
		msg := &TxMessage{tx, ""}
		encoded := cdc.MustMarshalBinaryBare(msg)
		require.Equal(t, len(encoded), txMessageSize(tx), caseString)
		if !testCase.err {
			require.True(t, len(encoded) <= maxMsgSize, caseString)
			require.NoError(t, err, caseString)
		} else {
			require.True(t, len(encoded) > maxMsgSize, caseString)
			require.Equal(t, err, ErrTxTooLarge{maxTxSize, testCase.len}, caseString)
		}
	}

}

func TestMempoolTxsBytes(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	config := cfg.ResetTestRoot("mempool_test")
	config.Mempool.MaxTxsBytes = 10
	mempool, cleanup := newMempoolWithAppAndConfig(cc, config)
	defer cleanup()

	// 1. zero by default
	assert.EqualValues(t, 0, mempool.TxsBytes())

	// 2. len(tx) after CheckTx
	err := mempool.CheckTx([]byte{0x01}, nil, TxInfo{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, mempool.TxsBytes())

	// 3. zero again after tx is removed by Update
	mempool.Update(1, []types.Tx{[]byte{0x01}}, abciResponses(1, abci.CodeTypeOK), nil, nil)
	assert.EqualValues(t, 0, mempool.TxsBytes())

	// 4. zero after Flush
	err = mempool.CheckTx([]byte{0x02, 0x03}, nil, TxInfo{})
	require.NoError(t, err)
	assert.EqualValues(t, 2, mempool.TxsBytes())

	mempool.Flush()
	assert.EqualValues(t, 0, mempool.TxsBytes())

	// 5. ErrMempoolIsFull is returned when/if MaxTxsBytes limit is reached.
	err = mempool.CheckTx([]byte{0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04}, nil, TxInfo{})
	require.NoError(t, err)
	err = mempool.CheckTx([]byte{0x05}, nil, TxInfo{})
	if assert.Error(t, err) {
		assert.IsType(t, ErrMempoolIsFull{}, err)
	}

	// 6. zero after tx is rechecked and removed due to not being valid anymore
	app2 := counter.NewApplication(true)
	cc = proxy.NewLocalClientCreator(app2)
	mempool, cleanup = newMempoolWithApp(cc)
	defer cleanup()

	txBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(txBytes, uint64(0))

	err = mempool.CheckTx(txBytes, nil, TxInfo{})
	require.NoError(t, err)
	assert.EqualValues(t, 8, mempool.TxsBytes())

	appConnCon, _ := cc.NewABCIClient()
	appConnCon.SetLogger(log.TestingLogger().With("module", "abci-client", "connection", "consensus"))
	err = appConnCon.Start()
	require.Nil(t, err)
	defer appConnCon.Stop()
	res, err := appConnCon.DeliverTxSync(abci.RequestDeliverTx{Tx: txBytes})
	require.NoError(t, err)
	require.EqualValues(t, 0, res.Code)
	res2, err := appConnCon.CommitSync(abci.RequestCommit{})
	require.NoError(t, err)
	require.NotEmpty(t, res2.Data)
	// Pretend like we committed nothing so txBytes gets rechecked and removed.
	// our config recheck flag default is false so cannot rechecked to remove unavailable txs
	// add config to check whether to assert mempool txsbytes
	height := int64(1)
	mempool.Update(height, []types.Tx{}, abciResponses(0, abci.CodeTypeOK), nil, nil)
	if cfg.DynamicConfig.GetMempoolRecheck() || height%cfg.DynamicConfig.GetMempoolForceRecheckGap() == 0 {
		assert.EqualValues(t, 0, mempool.TxsBytes())
	} else {
		assert.EqualValues(t, len(txBytes), mempool.TxsBytes())
	}
}

func abciResponses(n int, code uint32) []*abci.ResponseDeliverTx {
	responses := make([]*abci.ResponseDeliverTx, 0, n)
	for i := 0; i < n; i++ {
		responses = append(responses, &abci.ResponseDeliverTx{Code: code})
	}
	return responses
}

func TestAddAndSortTx(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	config := cfg.ResetTestRoot("mempool_test")
	config.Mempool.SortTxByGp = true
	mempool, cleanup := newMempoolWithAppAndConfig(cc, config)
	defer cleanup()

	//tx := &mempoolTx{height: 1, gasWanted: 1, tx:[]byte{0x01}}
	testCases := []struct {
		Tx *mempoolTx
	}{
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("1"), from: "18", realTx: abci.MockTx{GasPrice: big.NewInt(3780)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("2"), from: "6", realTx: abci.MockTx{GasPrice: big.NewInt(5853)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("3"), from: "7", realTx: abci.MockTx{GasPrice: big.NewInt(8315)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("4"), from: "10", realTx: abci.MockTx{GasPrice: big.NewInt(9526)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("5"), from: "15", realTx: abci.MockTx{GasPrice: big.NewInt(9140)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("6"), from: "9", realTx: abci.MockTx{GasPrice: big.NewInt(9227)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("7"), from: "3", realTx: abci.MockTx{GasPrice: big.NewInt(761)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("8"), from: "18", realTx: abci.MockTx{GasPrice: big.NewInt(9740)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("9"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(6574)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("10"), from: "8", realTx: abci.MockTx{GasPrice: big.NewInt(9656)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("11"), from: "12", realTx: abci.MockTx{GasPrice: big.NewInt(6554)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("12"), from: "16", realTx: abci.MockTx{GasPrice: big.NewInt(5609)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("13"), from: "6", realTx: abci.MockTx{GasPrice: big.NewInt(2791), Nonce: 1}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("14"), from: "18", realTx: abci.MockTx{GasPrice: big.NewInt(2698), Nonce: 1}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("15"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(6925), Nonce: 1}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("16"), from: "3", realTx: abci.MockTx{GasPrice: big.NewInt(3171)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("17"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(2965), Nonce: 2}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("18"), from: "19", realTx: abci.MockTx{GasPrice: big.NewInt(2484)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("19"), from: "13", realTx: abci.MockTx{GasPrice: big.NewInt(9722)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("20"), from: "7", realTx: abci.MockTx{GasPrice: big.NewInt(4236), Nonce: 1}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("21"), from: "18", realTx: abci.MockTx{GasPrice: big.NewInt(1780)}}},
	}

	for _, exInfo := range testCases {
		mempool.addTx(exInfo.Tx)
	}
	require.Equal(t, 18, mempool.txs.Len(), fmt.Sprintf("Expected to txs length %v but got %v", 18, mempool.txs.Len()))

	// The txs in mempool should sorted, the output should be (head -> tail):
	//
	//Address:  18  , GasPrice:  9740  , Nonce:  0
	//Address:  13  , GasPrice:  9722  , Nonce:  0
	//Address:  8  , GasPrice:  9656  , Nonce:  0
	//Address:  10  , GasPrice:  9526  , Nonce:  0
	//Address:  9  , GasPrice:  9227  , Nonce:  0
	//Address:  15  , GasPrice:  9140  , Nonce:  0
	//Address:  7  , GasPrice:  8315  , Nonce:  0
	//Address:  1  , GasPrice:  6574  , Nonce:  0
	//Address:  1  , GasPrice:  6925  , Nonce:  1
	//Address:  12  , GasPrice:  6554  , Nonce:  0
	//Address:  6  , GasPrice:  5853  , Nonce:  0
	//Address:  16  , GasPrice:  5609  , Nonce:  0
	//Address:  7  , GasPrice:  4236  , Nonce:  1
	//Address:  3  , GasPrice:  3171  , Nonce:  0
	//Address:  1  , GasPrice:  2965  , Nonce:  2
	//Address:  6  , GasPrice:  2791  , Nonce:  1
	//Address:  18  , GasPrice:  2698  , Nonce:  1
	//Address:  19  , GasPrice:  2484  , Nonce:  0

	require.Equal(t, 3, mempool.GetUserPendingTxsCnt("1"))
	require.Equal(t, 1, mempool.GetUserPendingTxsCnt("15"))
	require.Equal(t, 2, mempool.GetUserPendingTxsCnt("18"))

	require.Equal(t, "18", mempool.txs.Front().Address)
	require.Equal(t, big.NewInt(9740), mempool.txs.Front().GasPrice)
	require.Equal(t, uint64(0), mempool.txs.Front().Nonce)

	require.Equal(t, "19", mempool.txs.Back().Address)
	require.Equal(t, big.NewInt(2484), mempool.txs.Back().GasPrice)
	require.Equal(t, uint64(0), mempool.txs.Back().Nonce)

	require.Equal(t, true, checkTx(mempool.txs.Front()))

	addressList := mempool.GetAddressList()
	for _, addr := range addressList {
		require.Equal(t, true, checkAccNonce(addr, mempool.txs.Front()))
	}

	txs := mempool.ReapMaxBytesMaxGas(-1, -1)
	require.Equal(t, 18, len(txs), fmt.Sprintf("Expected to reap %v txs but got %v", 18, len(txs)))

	mempool.Flush()
	require.Equal(t, 0, mempool.txs.Len())
	require.Equal(t, 0, mempool.txs.BroadcastLen())
	require.Equal(t, 0, len(mempool.GetAddressList()))

}

func TestReplaceTx(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	config := cfg.ResetTestRoot("mempool_test")
	mempool, cleanup := newMempoolWithAppAndConfig(cc, config)
	defer cleanup()

	//tx := &mempoolTx{height: 1, gasWanted: 1, tx:[]byte{0x01}}
	testCases := []struct {
		Tx *mempoolTx
	}{
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("10000"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(9740)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("10001"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(5853), Nonce: 1}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("10002"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(8315), Nonce: 2}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("10003"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(9526), Nonce: 3}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("10004"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(9140), Nonce: 4}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("10002"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(9227), Nonce: 2}}},
	}

	for _, exInfo := range testCases {
		mempool.addTx(exInfo.Tx)
	}
	require.Equal(t, 5, mempool.txs.Len(), fmt.Sprintf("Expected to txs length %v but got %v", 5, mempool.txs.Len()))

	var nonces []uint64
	var gasPrices []uint64
	for e := mempool.txs.Front(); e != nil; e = e.Next() {
		nonces = append(nonces, e.Nonce)
		gasPrices = append(gasPrices, e.GasPrice.Uint64())
	}

	require.Equal(t, []uint64{0, 1, 2, 3, 4}, nonces)
	require.Equal(t, []uint64{9740, 5853, 9227, 9526, 9140}, gasPrices)
}

func TestAddAndSortTxByRandom(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	config := cfg.ResetTestRoot("mempool_test")
	mempool, cleanup := newMempoolWithAppAndConfig(cc, config)
	defer cleanup()

	AddrNonce := make(map[string]int)
	for i := 0; i < 1000; i++ {
		mempool.addTx(generateNode(AddrNonce, i))
	}

	require.Equal(t, true, checkTx(mempool.txs.Front()))
	addressList := mempool.GetAddressList()
	for _, addr := range addressList {
		require.Equal(t, true, checkAccNonce(addr, mempool.txs.Front()))
	}
}

func TestReapUserTxs(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	config := cfg.ResetTestRoot("mempool_test")
	mempool, cleanup := newMempoolWithAppAndConfig(cc, config)
	defer cleanup()

	//tx := &mempoolTx{height: 1, gasWanted: 1, tx:[]byte{0x01}}
	testCases := []struct {
		Tx *mempoolTx
	}{
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("1"), from: "18", realTx: abci.MockTx{GasPrice: big.NewInt(9740)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("2"), from: "6", realTx: abci.MockTx{GasPrice: big.NewInt(5853)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("3"), from: "7", realTx: abci.MockTx{GasPrice: big.NewInt(8315)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("4"), from: "10", realTx: abci.MockTx{GasPrice: big.NewInt(9526)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("5"), from: "15", realTx: abci.MockTx{GasPrice: big.NewInt(9140)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("6"), from: "9", realTx: abci.MockTx{GasPrice: big.NewInt(9227)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("7"), from: "3", realTx: abci.MockTx{GasPrice: big.NewInt(761)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("8"), from: "18", realTx: abci.MockTx{GasPrice: big.NewInt(3780)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("9"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(6574)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("10"), from: "8", realTx: abci.MockTx{GasPrice: big.NewInt(9656)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("11"), from: "12", realTx: abci.MockTx{GasPrice: big.NewInt(6554)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("12"), from: "16", realTx: abci.MockTx{GasPrice: big.NewInt(5609)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("13"), from: "6", realTx: abci.MockTx{GasPrice: big.NewInt(2791), Nonce: 1}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("14"), from: "18", realTx: abci.MockTx{GasPrice: big.NewInt(2698), Nonce: 1}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("15"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(6925), Nonce: 1}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("16"), from: "3", realTx: abci.MockTx{GasPrice: big.NewInt(3171)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("17"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(2965), Nonce: 2}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("18"), from: "19", realTx: abci.MockTx{GasPrice: big.NewInt(2484)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("19"), from: "13", realTx: abci.MockTx{GasPrice: big.NewInt(9722)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("20"), from: "7", realTx: abci.MockTx{GasPrice: big.NewInt(4236), Nonce: 1}}},
	}

	for _, exInfo := range testCases {
		mempool.addTx(exInfo.Tx)
	}
	require.Equal(t, 18, mempool.txs.Len(), fmt.Sprintf("Expected to txs length %v but got %v", 18,
		mempool.txs.Len()))

	require.Equal(t, 3, mempool.ReapUserTxsCnt("1"), fmt.Sprintf("Expected to txs length of %s "+
		"is %v but got %v", "1", 3, mempool.ReapUserTxsCnt("1")))

	require.Equal(t, 0, mempool.ReapUserTxsCnt("111"), fmt.Sprintf("Expected to txs length of %s "+
		"is %v but got %v", "111", 0, mempool.ReapUserTxsCnt("111")))

	require.Equal(t, 3, len(mempool.ReapUserTxs("1", -1)), fmt.Sprintf("Expected to txs length "+
		"of %s is %v but got %v", "1", 3, len(mempool.ReapUserTxs("1", -1))))

	require.Equal(t, 3, len(mempool.ReapUserTxs("1", 100)), fmt.Sprintf("Expected to txs length "+
		"of %s is %v but got %v", "1", 3, len(mempool.ReapUserTxs("1", 100))))

	require.Equal(t, 0, len(mempool.ReapUserTxs("111", -1)), fmt.Sprintf("Expected to txs length "+
		"of %s is %v but got %v", "111", 0, len(mempool.ReapUserTxs("111", -1))))

	require.Equal(t, 0, len(mempool.ReapUserTxs("111", 100)), fmt.Sprintf("Expected to txs length "+
		"of %s is %v but got %v", "111", 0, len(mempool.ReapUserTxs("111", 100))))
}

func generateNode(addrNonce map[string]int, idx int) *mempoolTx {
	mrand.Seed(time.Now().UnixNano())
	addr := strconv.Itoa(mrand.Int()%1000 + 1)
	gasPrice := mrand.Int()%100000 + 1

	nonce := 0
	if n, ok := addrNonce[addr]; ok {
		if gasPrice%177 == 0 {
			nonce = n - 1
		} else {
			nonce = n
		}
	}
	addrNonce[addr] = nonce + 1

	tx := &mempoolTx{
		height:    1,
		gasWanted: int64(idx),
		tx:        []byte(strconv.Itoa(idx)),
		from:      addr,
		realTx: abci.MockTx{
			GasPrice: big.NewInt(int64(gasPrice)),
			Nonce:    uint64(nonce),
		},
	}

	return tx
}

func checkAccNonce(addr string, head *clist.CElement) bool {
	nonce := uint64(0)

	for head != nil {
		if head.Address == addr {
			if head.Nonce != nonce {
				return false
			}
			nonce++
		}

		head = head.Next()
	}

	return true
}

func checkTx(head *clist.CElement) bool {
	for head != nil {
		next := head.Next()
		if next == nil {
			break
		}

		if head.Address == next.Address {
			if head.Nonce >= next.Nonce {
				return false
			}
		} else {
			if head.GasPrice.Cmp(next.GasPrice) < 0 {
				return false
			}
		}

		head = head.Next()
	}

	return true
}

func TestMultiPriceBump(t *testing.T) {
	tests := []struct {
		rawPrice    *big.Int
		priceBump   uint64
		targetPrice *big.Int
	}{
		{big.NewInt(1), 0, big.NewInt(1)},
		{big.NewInt(10), 1, big.NewInt(10)},
		{big.NewInt(100), 0, big.NewInt(100)},
		{big.NewInt(100), 5, big.NewInt(105)},
		{big.NewInt(100), 50, big.NewInt(150)},
		{big.NewInt(100), 150, big.NewInt(250)},
	}
	for _, tt := range tests {
		require.True(t, tt.targetPrice.Cmp(MultiPriceBump(tt.rawPrice, int64(tt.priceBump))) == 0)
	}
}

func TestAddAndSortTxConcurrency(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	config := cfg.ResetTestRoot("mempool_test")
	config.Mempool.SortTxByGp = true
	mempool, cleanup := newMempoolWithAppAndConfig(cc, config)
	defer cleanup()

	//tx := &mempoolTx{height: 1, gasWanted: 1, tx:[]byte{0x01}}
	type Case struct {
		Tx *mempoolTx
	}

	testCases := []Case{
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("1"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(3780)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("2"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(3780), Nonce: 1}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("3"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(5315), Nonce: 2}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("4"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(4526), Nonce: 3}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("5"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(2140), Nonce: 4}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("6"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(4227), Nonce: 5}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("7"), from: "2", realTx: abci.MockTx{GasPrice: big.NewInt(2161)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("8"), from: "2", realTx: abci.MockTx{GasPrice: big.NewInt(5740), Nonce: 1}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("9"), from: "2", realTx: abci.MockTx{GasPrice: big.NewInt(6574), Nonce: 2}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("10"), from: "2", realTx: abci.MockTx{GasPrice: big.NewInt(9630), Nonce: 3}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("11"), from: "2", realTx: abci.MockTx{GasPrice: big.NewInt(6554), Nonce: 4}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("12"), from: "2", realTx: abci.MockTx{GasPrice: big.NewInt(5609), Nonce: 2}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("13"), from: "3", realTx: abci.MockTx{GasPrice: big.NewInt(2791)}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("14"), from: "3", realTx: abci.MockTx{GasPrice: big.NewInt(2698), Nonce: 1}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("15"), from: "2", realTx: abci.MockTx{GasPrice: big.NewInt(6925), Nonce: 3}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("16"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(4171), Nonce: 3}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("17"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(2965), Nonce: 2}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("18"), from: "3", realTx: abci.MockTx{GasPrice: big.NewInt(2484), Nonce: 2}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("19"), from: "3", realTx: abci.MockTx{GasPrice: big.NewInt(9722), Nonce: 1}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("20"), from: "2", realTx: abci.MockTx{GasPrice: big.NewInt(4236), Nonce: 3}}},
		{&mempoolTx{height: 1, gasWanted: 1, tx: []byte("21"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(8780), Nonce: 4}}},
	}

	var wait sync.WaitGroup
	for _, exInfo := range testCases {
		wait.Add(1)
		go func(p Case) {
			mempool.addTx(p.Tx)
			wait.Done()
		}(exInfo)
	}

	wait.Wait()
}

func TestTxID(t *testing.T) {
	var bytes = make([]byte, 256)
	for i := 0; i < 10; i++ {
		_, err := rand.Read(bytes)
		require.NoError(t, err)
		require.Equal(t, amino.HexEncodeToStringUpper(bytes), fmt.Sprintf("%X", bytes))
	}
}

func BenchmarkTxID(b *testing.B) {
	var bytes = make([]byte, 256)
	_, _ = rand.Read(bytes)
	var res string
	b.Run("fmt", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			res = fmt.Sprintf("%X", bytes)
		}
	})
	b.Run("amino", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			res = amino.HexEncodeToStringUpper(bytes)
		}
	})
	_ = res
}

func TestReplaceTxWithMultiAddrs(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	config := cfg.ResetTestRoot("mempool_test")
	mempool, cleanup := newMempoolWithAppAndConfig(cc, config)
	defer cleanup()

	tx1 := &mempoolTx{height: 1, gasWanted: 1, tx: []byte("10002"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(9740), Nonce: 1}}
	mempool.addTx(tx1)
	tx2 := &mempoolTx{height: 1, gasWanted: 1, tx: []byte("90000"), from: "2", realTx: abci.MockTx{GasPrice: big.NewInt(10717), Nonce: 1}}
	mempool.addTx(tx2)
	tx3 := &mempoolTx{height: 1, gasWanted: 1, tx: []byte("90000"), from: "3", realTx: abci.MockTx{GasPrice: big.NewInt(10715), Nonce: 1}}
	mempool.addTx(tx3)
	tx4 := &mempoolTx{height: 1, gasWanted: 1, tx: []byte("10001"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(10716), Nonce: 2}}
	mempool.addTx(tx4)
	tx5 := &mempoolTx{height: 1, gasWanted: 1, tx: []byte("10001"), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(10712), Nonce: 1}}
	mempool.addTx(tx5)

	var nonces []uint64
	for e := mempool.txs.Front(); e != nil; e = e.Next() {
		if e.Address == "1" {
			nonces = append(nonces, e.Nonce)
		}
	}
	require.Equal(t, []uint64{1, 2}, nonces)
}

func BenchmarkMempoolLogUpdate(b *testing.B) {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "benchmark")
	var options []log.Option
	options = append(options, log.AllowErrorWith("module", "benchmark"))
	logger = log.NewFilter(logger, options...)

	mem := &CListMempool{height: 123456, logger: logger}
	addr := "address"
	nonce := uint64(123456)

	b.Run("pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			mem.logUpdate(addr, nonce)
		}
	})

	b.Run("logger", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			mem.logger.Debug("mempool update", "address", addr, "nonce", nonce)
		}
	})
}

func BenchmarkMempoolLogAddTx(b *testing.B) {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "benchmark")
	var options []log.Option
	options = append(options, log.AllowErrorWith("module", "benchmark"))
	logger = log.NewFilter(logger, options...)

	mem := &CListMempool{height: 123456, logger: logger, txs: NewBaseTxQueue()}
	tx := []byte("tx")

	memTx := &mempoolTx{
		height: mem.Height(),
		tx:     tx,
	}

	r := &abci.Response_CheckTx{}

	b.Run("pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			mem.logAddTx(memTx, r)
		}
	})

	b.Run("logger", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			mem.logger.Info("Added good transaction",
				"tx", txIDStringer{tx, mem.height},
				"res", r,
				"height", memTx.height,
				"total", mem.Size(),
			)
		}
	})
}

func TestTxOrTxHashToKey(t *testing.T) {
	var tx = make([]byte, 256)
	rand.Read(tx)

	txhash := types.Tx(tx).Hash()

	require.Equal(t, txKey(tx), txOrTxHashToKey(tx, nil))
	require.Equal(t, txKey(tx), txOrTxHashToKey(tx, txhash))
	require.Equal(t, txKey(tx), txOrTxHashToKey(tx, types.Tx(tx).Hash()))
	txhash[0] += 1
	require.NotEqual(t, txKey(tx), txOrTxHashToKey(tx, txhash))
}

func TestCListMempool_GetEnableDeleteMinGPTx(t *testing.T) {

	testCases := []struct {
		name     string
		prepare  func(mempool *CListMempool, tt *testing.T)
		execFunc func(mempool *CListMempool, tt *testing.T)
	}{
		{
			name: "normal mempool is full add tx failed, disableDeleteMinGPTx",
			prepare: func(mempool *CListMempool, tt *testing.T) {
				mempool.Flush()
				err := mempool.CheckTx([]byte{0x01}, nil, TxInfo{})
				require.NoError(tt, err)
			},
			execFunc: func(mempool *CListMempool, tt *testing.T) {
				err := mempool.CheckTx([]byte{0x02}, nil, TxInfo{})
				require.Error(tt, err)
				_, ok := err.(ErrMempoolIsFull)
				require.True(t, ok)
			},
		},
		{
			name: "normal mempool is full add tx failed, enableDeleteMinGPTx",
			prepare: func(mempool *CListMempool, tt *testing.T) {
				mempool.Flush()
				err := mempool.CheckTx([]byte{0x02}, nil, TxInfo{})
				require.NoError(tt, err)
				moc := cfg.MockDynamicConfig{}
				moc.SetEnableDeleteMinGPTx(true)
				cfg.SetDynamicConfig(moc)
			},
			execFunc: func(mempool *CListMempool, tt *testing.T) {
				err := mempool.CheckTx([]byte{0x03}, nil, TxInfo{})
				require.NoError(tt, err)
				require.Equal(tt, 1, mempool.Size())
				tx := mempool.txs.Back().Value.(*mempoolTx).tx
				require.Equal(tt, byte(0x02), tx[0])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			app := kvstore.NewApplication()
			cc := proxy.NewLocalClientCreator(app)
			mempool, cleanup := newMempoolWithApp(cc)
			mempool.config.MaxTxsBytes = 1 //  in unit test we only use tx bytes to  control mempool weather full
			defer cleanup()

			tc.prepare(mempool, tt)
			tc.execFunc(mempool, tt)
		})
	}

}

func TestConsumePendingtxConcurrency(t *testing.T) {

	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mem, cleanup := newMempoolWithApp(cc)
	defer cleanup()
	mem.pendingPool = newPendingPool(500000, 3, 10, 500000)

	for i := 0; i < 10000; i++ {
		mem.pendingPool.addTx(&mempoolTx{height: 1, gasWanted: 1, tx: []byte(strconv.Itoa(i)), from: "1", realTx: abci.MockTx{GasPrice: big.NewInt(3780), Nonce: uint64(i)}})
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	startWg := &sync.WaitGroup{}
	startWg.Add(1)
	go func() {
		startWg.Wait()
		mem.consumePendingTx("1", 0)
		wg.Done()
	}()
	startWg.Done()
	mem.consumePendingTx("1", 5000)
	wg.Wait()
	require.Equal(t, 0, mem.pendingPool.Size())
}

func TestCheckAndGetWrapCMTx(t *testing.T) {
	wCMTx := &types.WrapCMTx{Tx: []byte("123456"), Nonce: 2}
	wdata, err := cdc.MarshalJSON(wCMTx)
	assert.NoError(t, err)

	testcase := []struct {
		tx     types.Tx
		txInfo TxInfo
		res    *types.WrapCMTx
	}{
		{
			tx:     []byte("123"),
			txInfo: TxInfo{wrapCMTx: &types.WrapCMTx{Tx: []byte("123"), Nonce: 1}},
			res:    &types.WrapCMTx{Tx: []byte("123"), Nonce: 1},
		},
		{
			tx:     []byte("123"),
			txInfo: TxInfo{},
			res:    nil,
		},
		{
			tx:     wdata,
			txInfo: TxInfo{},
			res:    wCMTx,
		},
	}

	clistMem := &CListMempool{}
	for _, tc := range testcase {
		re := clistMem.CheckAndGetWrapCMTx(tc.tx, tc.txInfo)
		if re != nil {
			assert.Equal(t, *re, *tc.res)
		} else {
			assert.Equal(t, re, tc.res)
		}
	}
}
