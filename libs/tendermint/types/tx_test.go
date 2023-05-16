package types

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/okx/okbchain/libs/tendermint/abci/types"

	"github.com/stretchr/testify/assert"

	tmrand "github.com/okx/okbchain/libs/tendermint/libs/rand"
	ctest "github.com/okx/okbchain/libs/tendermint/libs/test"
)

func makeTxs(cnt, size int) Txs {
	txs := make(Txs, cnt)
	for i := 0; i < cnt; i++ {
		txs[i] = tmrand.Bytes(size)
	}
	return txs
}

func randInt(low, high int) int {
	off := tmrand.Int() % (high - low)
	return low + off
}

func TestTxIndex(t *testing.T) {
	for i := 0; i < 20; i++ {
		txs := makeTxs(15, 60)
		for j := 0; j < len(txs); j++ {
			tx := txs[j]
			idx := txs.Index(tx)
			assert.Equal(t, j, idx)
		}
		assert.Equal(t, -1, txs.Index(nil))
		assert.Equal(t, -1, txs.Index(Tx("foodnwkf")))
	}
}

func TestTxIndexByHash(t *testing.T) {

	for i := 0; i < 20; i++ {
		txs := makeTxs(15, 60)
		for j := 0; j < len(txs); j++ {
			tx := txs[j]
			idx := txs.IndexByHash(tx.Hash())
			assert.Equal(t, j, idx)
		}
		assert.Equal(t, -1, txs.IndexByHash(nil))
		assert.Equal(t, -1, txs.IndexByHash(Tx("foodnwkf").Hash()))
	}
}

func TestValidTxProof(t *testing.T) {
	cases := []struct {
		txs Txs
	}{
		{Txs{{1, 4, 34, 87, 163, 1}}},
		{Txs{{5, 56, 165, 2}, {4, 77}}},
		{Txs{Tx("foo"), Tx("bar"), Tx("baz")}},
		{makeTxs(20, 5)},
		{makeTxs(7, 81)},
		{makeTxs(61, 15)},
	}

	for h, tc := range cases {
		txs := tc.txs
		root := txs.Hash()
		// make sure valid proof for every tx
		for i := range txs {
			tx := []byte(txs[i])
			proof := txs.Proof(i)
			assert.Equal(t, i, proof.Proof.Index, "%d: %d", h, i)
			assert.Equal(t, len(txs), proof.Proof.Total, "%d: %d", h, i)
			assert.EqualValues(t, root, proof.RootHash, "%d: %d", h, i)
			assert.EqualValues(t, tx, proof.Data, "%d: %d", h, i)
			assert.EqualValues(t, txs[i].Hash(), proof.Leaf(), "%d: %d", h, i)
			assert.Nil(t, proof.Validate(root), "%d: %d", h, i)
			assert.NotNil(t, proof.Validate([]byte("foobar")), "%d: %d", h, i)

			// read-write must also work
			var p2 TxProof
			bin, err := cdc.MarshalBinaryLengthPrefixed(proof)
			assert.Nil(t, err)
			err = cdc.UnmarshalBinaryLengthPrefixed(bin, &p2)
			if assert.Nil(t, err, "%d: %d: %+v", h, i, err) {
				assert.Nil(t, p2.Validate(root), "%d: %d", h, i)
			}
		}
	}
}

func TestTxProofUnchangable(t *testing.T) {
	// run the other test a bunch...
	for i := 0; i < 40; i++ {
		testTxProofUnchangable(t)
	}
}

func TestComputeTxsOverhead(t *testing.T) {
	cases := []struct {
		txs          Txs
		wantOverhead int
	}{
		{Txs{[]byte{6, 6, 6, 6, 6, 6}}, 2},
		// one 21 Mb transaction:
		{Txs{make([]byte, 22020096)}, 5},
		// two 21Mb/2 sized transactions:
		{Txs{make([]byte, 11010048), make([]byte, 11010048)}, 10},
		{Txs{[]byte{1, 2, 3}, []byte{1, 2, 3}, []byte{4, 5, 6}}, 6},
		{Txs{[]byte{100, 5, 64}, []byte{42, 116, 118}, []byte{6, 6, 6}, []byte{6, 6, 6}}, 8},
	}

	for _, tc := range cases {
		totalBytes := int64(0)
		totalOverhead := int64(0)
		for _, tx := range tc.txs {
			aminoOverhead := ComputeAminoOverhead(tx, 1)
			totalOverhead += aminoOverhead
			totalBytes += aminoOverhead + int64(len(tx))
		}
		bz, err := cdc.MarshalBinaryBare(tc.txs)
		assert.EqualValues(t, tc.wantOverhead, totalOverhead)
		assert.NoError(t, err)
		assert.EqualValues(t, len(bz), totalBytes)
	}
}

func TestComputeAminoOverhead(t *testing.T) {
	cases := []struct {
		tx       Tx
		fieldNum int
		want     int
	}{
		{[]byte{6, 6, 6}, 1, 2},
		{[]byte{6, 6, 6}, 16, 3},
		{[]byte{6, 6, 6}, 32, 3},
		{[]byte{6, 6, 6}, 64, 3},
		{[]byte{6, 6, 6}, 512, 3},
		{[]byte{6, 6, 6}, 1024, 3},
		{[]byte{6, 6, 6}, 2048, 4},
		{make([]byte, 64), 1, 2},
		{make([]byte, 65), 1, 2},
		{make([]byte, 127), 1, 2},
		{make([]byte, 128), 1, 3},
		{make([]byte, 256), 1, 3},
		{make([]byte, 512), 1, 3},
		{make([]byte, 1024), 1, 3},
		{make([]byte, 128), 16, 4},
	}
	for _, tc := range cases {
		got := ComputeAminoOverhead(tc.tx, tc.fieldNum)
		assert.EqualValues(t, tc.want, got)
	}
}

func testTxProofUnchangable(t *testing.T) {
	// make some proof
	txs := makeTxs(randInt(2, 100), randInt(16, 128))
	root := txs.Hash()
	i := randInt(0, len(txs)-1)
	proof := txs.Proof(i)

	// make sure it is valid to start with
	assert.Nil(t, proof.Validate(root))
	bin, err := cdc.MarshalBinaryLengthPrefixed(proof)
	assert.Nil(t, err)

	// try mutating the data and make sure nothing breaks
	for j := 0; j < 500; j++ {
		bad := ctest.MutateByteSlice(bin)
		if !bytes.Equal(bad, bin) {
			assertBadProof(t, root, bad, proof)
		}
	}
}

// This makes sure that the proof doesn't deserialize into something valid.
func assertBadProof(t *testing.T, root []byte, bad []byte, good TxProof) {
	var proof TxProof
	err := cdc.UnmarshalBinaryLengthPrefixed(bad, &proof)
	if err == nil {
		err = proof.Validate(root)
		if err == nil {
			// XXX Fix simple merkle proofs so the following is *not* OK.
			// This can happen if we have a slightly different total (where the
			// path ends up the same). If it is something else, we have a real
			// problem.
			assert.NotEqual(t, proof.Proof.Total, good.Proof.Total, "bad: %#v\ngood: %#v", proof, good)
		}
	}
}

var txResultTestCases = []TxResult{
	{},
	{Tx: []byte{}},
	{123, 123, []byte("tx bytes"), types.ResponseDeliverTx{Code: 123, Data: []byte("this is data"), Log: "log123", Info: "123info", GasWanted: 1234445, GasUsed: 98, Events: []types.Event{}, Codespace: "sssdasf"}},
	{Height: math.MaxInt64, Index: math.MaxUint32},
	{Height: math.MinInt64, Index: 0},
	{Height: -1, Index: 0},
}

func TestTxResultAmino(t *testing.T) {
	for _, txResult := range txResultTestCases {
		expectData, err := cdc.MarshalBinaryBare(txResult)
		require.NoError(t, err)

		var expectValue TxResult
		err = cdc.UnmarshalBinaryBare(expectData, &expectValue)
		require.NoError(t, err)

		var actualValue TxResult
		err = actualValue.UnmarshalFromAmino(cdc, expectData)
		require.NoError(t, err)

		require.EqualValues(t, expectValue, actualValue)
	}
}

func BenchmarkTxResultAminoUnmarshal(b *testing.B) {
	testData := make([][]byte, len(txResultTestCases))
	for i, res := range txResultTestCases {
		expectData, err := cdc.MarshalBinaryBare(res)
		require.NoError(b, err)
		testData[i] = expectData
	}
	b.ResetTimer()

	b.Run("amino", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for _, data := range testData {
				var res TxResult
				err := cdc.UnmarshalBinaryBare(data, &res)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})
	b.Run("unmarshaller", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for _, data := range testData {
				var res TxResult
				err := res.UnmarshalFromAmino(cdc, data)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})
}

func TestTxsToTxWithMetas(t *testing.T) {
	txs := makeTxs(2000000, randInt(16, 128))
	retxs := TxsToTxWithMetas(txs)

	retxs.Hash()

	for i := 0; i < len(txs); i++ {
		assert.Equal(t, []byte(txs[i]), retxs[i].GetTx())
		assert.Equal(t, txs[i].Hash(), retxs[i].Hash())
	}
	fmt.Println(len(retxs))
}
