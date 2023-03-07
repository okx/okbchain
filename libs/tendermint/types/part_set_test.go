package types

import (
	"io/ioutil"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/okx/okbchain/libs/tendermint/crypto/merkle"
	tmrand "github.com/okx/okbchain/libs/tendermint/libs/rand"
)

const (
	testPartSize = 65536 // 64KB ...  4096 // 4KB
)

func TestBasicPartSet(t *testing.T) {
	// Construct random data of size partSize * 100
	data := tmrand.Bytes(testPartSize * 100)
	partSet := NewPartSetFromData(data, testPartSize)

	assert.NotEmpty(t, partSet.Hash())
	assert.Equal(t, 100, partSet.Total())
	assert.Equal(t, 100, partSet.BitArray().Size())
	assert.True(t, partSet.HashesTo(partSet.Hash()))
	assert.True(t, partSet.IsComplete())
	assert.Equal(t, 100, partSet.Count())

	// Test adding parts to a new partSet.
	partSet2 := NewPartSetFromHeader(partSet.Header())

	assert.True(t, partSet2.HasHeader(partSet.Header()))
	for i := 0; i < partSet.Total(); i++ {
		part := partSet.GetPart(i)
		//t.Logf("\n%v", part)
		added, err := partSet2.AddPart(part)
		if !added || err != nil {
			t.Errorf("failed to add part %v, error: %v", i, err)
		}
	}
	// adding part with invalid index
	added, err := partSet2.AddPart(&Part{Index: 10000})
	assert.False(t, added)
	assert.Error(t, err)
	// adding existing part
	added, err = partSet2.AddPart(partSet2.GetPart(0))
	assert.False(t, added)
	assert.Nil(t, err)

	assert.Equal(t, partSet.Hash(), partSet2.Hash())
	assert.Equal(t, 100, partSet2.Total())
	assert.True(t, partSet2.IsComplete())

	// Reconstruct data, assert that they are equal.
	data2Reader := partSet2.GetReader()
	data2, err := ioutil.ReadAll(data2Reader)
	require.NoError(t, err)

	assert.Equal(t, data, data2)
}

func TestWrongProof(t *testing.T) {
	// Construct random data of size partSize * 100
	data := tmrand.Bytes(testPartSize * 100)
	partSet := NewPartSetFromData(data, testPartSize)

	// Test adding a part with wrong data.
	partSet2 := NewPartSetFromHeader(partSet.Header())

	// Test adding a part with wrong trail.
	part := partSet.GetPart(0)
	part.Proof.Aunts[0][0] += byte(0x01)
	added, err := partSet2.AddPart(part)
	if added || err == nil {
		t.Errorf("expected to fail adding a part with bad trail.")
	}

	// Test adding a part with wrong bytes.
	part = partSet.GetPart(1)
	part.Bytes[0] += byte(0x01)
	added, err = partSet2.AddPart(part)
	if added || err == nil {
		t.Errorf("expected to fail adding a part with bad bytes.")
	}
}

func TestPartSetHeaderValidateBasic(t *testing.T) {
	testCases := []struct {
		testName              string
		malleatePartSetHeader func(*PartSetHeader)
		expectErr             bool
	}{
		{"Good PartSet", func(psHeader *PartSetHeader) {}, false},
		{"Negative Total", func(psHeader *PartSetHeader) { psHeader.Total = -2 }, true},
		{"Invalid Hash", func(psHeader *PartSetHeader) { psHeader.Hash = make([]byte, 1) }, true},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			data := tmrand.Bytes(testPartSize * 100)
			ps := NewPartSetFromData(data, testPartSize)
			psHeader := ps.Header()
			tc.malleatePartSetHeader(&psHeader)
			assert.Equal(t, tc.expectErr, psHeader.ValidateBasic() != nil, "Validate Basic had an unexpected result")
		})
	}
}

func TestPartValidateBasic(t *testing.T) {
	testCases := []struct {
		testName     string
		malleatePart func(*Part)
		expectErr    bool
	}{
		{"Good Part", func(pt *Part) {}, false},
		{"Negative index", func(pt *Part) { pt.Index = -1 }, true},
		{"Too big part", func(pt *Part) { pt.Bytes = make([]byte, BlockPartSizeBytes+1) }, true},
		{"Too big proof", func(pt *Part) {
			pt.Proof = merkle.SimpleProof{
				Total:    1,
				Index:    1,
				LeafHash: make([]byte, 1024*1024),
			}
		}, true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			data := tmrand.Bytes(testPartSize * 100)
			ps := NewPartSetFromData(data, testPartSize)
			part := ps.GetPart(0)
			tc.malleatePart(part)
			assert.Equal(t, tc.expectErr, part.ValidateBasic() != nil, "Validate Basic had an unexpected result")
		})
	}
}

func TestParSetHeaderProtoBuf(t *testing.T) {
	testCases := []struct {
		msg     string
		ps1     *PartSetHeader
		expPass bool
	}{
		{"success empty", &PartSetHeader{}, true},
		{"success",
			&PartSetHeader{Total: 1, Hash: []byte("hash")}, true},
	}

	for _, tc := range testCases {
		protoBlockID := tc.ps1.ToProto()

		psh, err := PartSetHeaderFromProto(&protoBlockID)
		if tc.expPass {
			require.Equal(t, tc.ps1, psh, tc.msg)
		} else {
			require.Error(t, err, tc.msg)
		}
	}
}

var partSetHeaderTestCases = []PartSetHeader{
	{},
	{12345, []byte("hash")},
	{math.MaxInt, []byte("hashhashhashhashhashhashhashhashhashhashhashhash")},
	{Total: math.MinInt, Hash: []byte{}},
}

func TestPartSetHeaderAmino(t *testing.T) {
	for _, tc := range partSetHeaderTestCases {
		bz, err := cdc.MarshalBinaryBare(&tc)
		require.NoError(t, err)

		var psh PartSetHeader
		err = cdc.UnmarshalBinaryBare(bz, &psh)
		require.NoError(t, err)

		var psh2 PartSetHeader
		err = psh2.UnmarshalFromAmino(cdc, bz)
		require.NoError(t, err)

		require.EqualValues(t, psh, psh2)

		require.EqualValues(t, len(bz), psh.AminoSize())
	}
}

func BenchmarkPartSetHeaderAminoUnmarshal(b *testing.B) {
	testData := make([][]byte, len(partSetHeaderTestCases))
	for i, tc := range partSetHeaderTestCases {
		bz, err := cdc.MarshalBinaryBare(&tc)
		require.NoError(b, err)
		testData[i] = bz
	}
	b.ResetTimer()

	b.Run("amino", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for _, data := range testData {
				var psh PartSetHeader
				err := cdc.UnmarshalBinaryBare(data, &psh)
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
				var psh PartSetHeader
				err := psh.UnmarshalFromAmino(cdc, data)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})
}

var partAminoTestCases = []Part{
	{},
	{
		Index: 2,
		Bytes: []byte("bytes"),
		Proof: merkle.SimpleProof{
			Total:    10,
			Index:    2,
			LeafHash: []byte("LeafHash"),
			Aunts:    [][]byte{[]byte("aunt1"), []byte("aunt2")},
		},
	},
	{
		Index: math.MaxInt,
		Bytes: []byte{},
	},
	{
		Index: math.MinInt,
	},
}

func TestPartAmino(t *testing.T) {
	for _, part := range partAminoTestCases {
		expectData, err := cdc.MarshalBinaryBare(part)
		require.NoError(t, err)
		var expectValue Part
		err = cdc.UnmarshalBinaryBare(expectData, &expectValue)
		require.NoError(t, err)
		var actualValue Part
		err = actualValue.UnmarshalFromAmino(cdc, expectData)
		require.NoError(t, err)

		require.EqualValues(t, expectValue, actualValue)
	}
}

func BenchmarkPartAminoUnmarshal(b *testing.B) {
	testData := make([][]byte, len(partAminoTestCases))
	for i, p := range partAminoTestCases {
		d, err := cdc.MarshalBinaryBare(p)
		require.NoError(b, err)
		testData[i] = d
	}
	b.ResetTimer()

	b.Run("amino", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for _, d := range testData {
				var v Part
				err := cdc.UnmarshalBinaryBare(d, &v)
				if err != nil {
					b.Fatal()
				}
			}
		}
	})

	b.Run("unmarshaller", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for _, d := range testData {
				var v Part
				err := v.UnmarshalFromAmino(cdc, d)
				if err != nil {
					b.Fatal()
				}
			}
		}
	})
}
