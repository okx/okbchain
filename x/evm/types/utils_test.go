package types

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync"
	"testing"

	ethcmn "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/tendermint/global"
	"github.com/stretchr/testify/require"
)

func TestEvmDataEncoding(t *testing.T) {
	addr := ethcmn.HexToAddress("0x5dE8a020088a2D6d0a23c204FFbeD02790466B49")
	bloom := ethtypes.BytesToBloom([]byte{0x1, 0x3})
	ret := []byte{0x5, 0x8}

	data := ResultData{
		ContractAddress: addr,
		Bloom:           bloom,
		Logs: []*ethtypes.Log{{
			Data:        []byte{1, 2, 3, 4},
			BlockNumber: 17,
		}},
		Ret: ret,
	}

	enc, err := EncodeResultData(&data)
	require.NoError(t, err)

	res, err := DecodeResultData(enc)
	require.NoError(t, err)
	require.Equal(t, addr, res.ContractAddress)
	require.Equal(t, bloom, res.Bloom)
	require.Equal(t, data.Logs, res.Logs)
	require.Equal(t, ret, res.Ret)

	// error check
	_, err = DecodeResultData(enc[1:])
	require.Error(t, err)
}

func TestValidateSigner(t *testing.T) {
	const digest = "default digest"
	digestHash := crypto.Keccak256([]byte(digest))
	priv, err := crypto.GenerateKey()
	require.NotNil(t, priv)
	require.NoError(t, err)

	ethAddr := crypto.PubkeyToAddress(priv.PublicKey)
	require.NoError(t, err)

	sig, err := crypto.Sign(digestHash, priv)
	require.NoError(t, err)

	err = ValidateSigner(digestHash, sig, ethAddr)
	require.NoError(t, err)

	// different eth address
	otherEthAddr := ethcmn.BytesToAddress([]byte{1})
	err = ValidateSigner(digestHash, sig, otherEthAddr)
	require.Error(t, err)

	// invalid digestHash
	err = ValidateSigner(digestHash[1:], sig, otherEthAddr)
	require.Error(t, err)
}

func TestResultData_String(t *testing.T) {
	const expectedResultDataStr = `ResultData:
	ContractAddress: 0x5dE8a020088a2D6d0a23c204FFbeD02790466B49
	Bloom: 259
	Ret: [5 8]
	TxHash: 0x0000000000000000000000000000000000000000000000000000000000000000	
	Logs: 
		{0x0000000000000000000000000000000000000000 [] [1 2 3 4] 17 0x0000000000000000000000000000000000000000000000000000000000000000 0 0x0000000000000000000000000000000000000000000000000000000000000000 0 false}
 		{0x0000000000000000000000000000000000000000 [] [5 6 7 8] 18 0x0000000000000000000000000000000000000000000000000000000000000000 0 0x0000000000000000000000000000000000000000000000000000000000000000 0 false}`
	addr := ethcmn.HexToAddress("0x5dE8a020088a2D6d0a23c204FFbeD02790466B49")
	bloom := ethtypes.BytesToBloom([]byte{0x1, 0x3})
	ret := []byte{0x5, 0x8}

	data := ResultData{
		ContractAddress: addr,
		Bloom:           bloom,
		Logs: []*ethtypes.Log{
			{
				Data:        []byte{1, 2, 3, 4},
				BlockNumber: 17,
			},
			{
				Data:        []byte{5, 6, 7, 8},
				BlockNumber: 18,
			}},
		Ret: ret,
	}

	require.True(t, strings.EqualFold(expectedResultDataStr, data.String()))
}

func TestTxDecoder(t *testing.T) {
	expectUint64, expectedBigInt, expectedBytes := uint64(1024), big.NewInt(1024), []byte("default payload")
	expectedEthAddr := ethcmn.BytesToAddress([]byte("test_address"))
	expectedEthMsg := NewMsgEthereumTx(expectUint64, &expectedEthAddr, expectedBigInt, expectUint64, expectedBigInt, expectedBytes)

	// register codec
	cdc := codec.New()
	cdc.RegisterInterface((*sdk.Tx)(nil), nil)
	RegisterCodec(cdc)

	txbytes := cdc.MustMarshalBinaryLengthPrefixed(expectedEthMsg)
	txDecoder := TxDecoder(cdc)
	tx, err := txDecoder(txbytes)
	require.Error(t, err)

	rlpBytes, err := rlp.EncodeToBytes(&expectedEthMsg)
	require.Nil(t, err)
	tx, err = txDecoder(rlpBytes)
	require.NoError(t, err)

	msgs := tx.GetMsgs()
	require.Equal(t, 1, len(msgs))
	require.NoError(t, msgs[0].ValidateBasic())
	require.True(t, strings.EqualFold(expectedEthMsg.Route(), msgs[0].Route()))
	require.True(t, strings.EqualFold(expectedEthMsg.Type(), msgs[0].Type()))

	require.NoError(t, tx.ValidateBasic())

	// error check
	_, err = txDecoder([]byte{})
	require.Error(t, err)

	_, err = txDecoder(txbytes[1:])
	require.Error(t, err)

	for _, c := range []struct {
		curHeight          int64
		enableAminoDecoder bool
		enableRLPDecoder   bool
	}{
		{999, false, true},
		{999, false, true},
		{1000, false, true},
		{1500, false, true},
	} {
		_, err = TxDecoder(cdc)(txbytes, c.curHeight)
		require.Equal(t, c.enableAminoDecoder, err == nil)
		_, err = TxDecoder(cdc)(rlpBytes, c.curHeight)
		require.Equal(t, c.enableRLPDecoder, err == nil)

		// use global height when height is not pass through parameters.
		global.SetGlobalHeight(c.curHeight)
		_, err = TxDecoder(cdc)(txbytes)
		require.Equal(t, c.enableAminoDecoder, err == nil)
		_, err = TxDecoder(cdc)(rlpBytes)
		require.Equal(t, c.enableRLPDecoder, err == nil)
	}
}

func TestEthLogAmino(t *testing.T) {
	tests := []ethtypes.Log{
		{},
		{Topics: []ethcmn.Hash{}, Data: []byte{}},
		{
			Address: ethcmn.HexToAddress("0x5dE8a020088a2D6d0a23c204FFbeD02790466B49"),
			Topics: []ethcmn.Hash{
				ethcmn.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
				ethcmn.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
				ethcmn.HexToHash("0x1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF"),
			},
			Data:        []byte{1, 2, 3, 4},
			BlockNumber: 17,
			TxHash:      ethcmn.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
			TxIndex:     123456,
			BlockHash:   ethcmn.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
			Index:       543121,
			Removed:     false,
		},
		{
			Address: ethcmn.HexToAddress("0x5dE8a020088a2D6d0a23c204FFbeD02790466B49"),
			Topics: []ethcmn.Hash{
				ethcmn.HexToHash("0x00000000FF0000000000000000000AC0000000000000EF000000000000000000"),
				ethcmn.HexToHash("0x1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF"),
			},
			Data:        []byte{5, 6, 7, 8},
			BlockNumber: math.MaxUint64,
			TxHash:      ethcmn.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
			TxIndex:     math.MaxUint,
			BlockHash:   ethcmn.HexToHash("0x1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF"),
			Index:       math.MaxUint,
			Removed:     true,
		},
	}
	cdc := codec.New()
	for _, test := range tests {
		bz, err := cdc.MarshalBinaryBare(test)
		require.NoError(t, err)

		bz2, err := MarshalEthLogToAmino(&test)
		require.NoError(t, err)
		require.EqualValues(t, bz, bz2)

		var expect ethtypes.Log
		err = cdc.UnmarshalBinaryBare(bz, &expect)
		require.NoError(t, err)

		actual, err := UnmarshalEthLogFromAmino(bz)
		require.NoError(t, err)
		require.EqualValues(t, expect, *actual)
	}
}

func TestResultDataAmino(t *testing.T) {
	addr := ethcmn.HexToAddress("0x5dE8a020088a2D6d0a23c204FFbeD02790466B49")
	bloom := ethtypes.BytesToBloom([]byte{0x1, 0x3, 0x5, 0x7})
	ret := []byte{0x5, 0x8}

	cdc := codec.New()
	cdc.RegisterInterface((*sdk.Tx)(nil), nil)
	RegisterCodec(cdc)

	testDataSet := []ResultData{
		{},
		{Logs: []*ethtypes.Log{}, Ret: []byte{}},
		{
			ContractAddress: addr,
			Bloom:           bloom,
			Logs: []*ethtypes.Log{
				{
					Data:        []byte{1, 2, 3, 4},
					BlockNumber: 17,
					Index:       10,
				},
				{
					Data:        []byte{1, 2, 3, 4},
					BlockNumber: 17,
					Index:       10,
				},
				{
					Data:        []byte{1, 2, 3, 4},
					BlockNumber: 17,
					Index:       10,
				},
				nil,
			},
			Ret:    ret,
			TxHash: ethcmn.HexToHash("0x00"),
		},
		{
			ContractAddress: addr,
			Bloom:           bloom,
			Logs: []*ethtypes.Log{
				nil,
				{
					Removed: true,
				},
			},
			Ret:    ret,
			TxHash: ethcmn.HexToHash("0x00"),
		},
	}

	for i, data := range testDataSet {
		expect, err := cdc.MarshalBinaryBare(data)
		require.NoError(t, err)

		actual, err := data.MarshalToAmino(cdc)
		require.NoError(t, err)
		require.EqualValues(t, expect, actual)
		t.Log(fmt.Sprintf("%d pass\n", i))

		var expectRd ResultData
		err = cdc.UnmarshalBinaryBare(expect, &expectRd)
		require.NoError(t, err)
		var actualRd ResultData
		err = actualRd.UnmarshalFromAmino(cdc, expect)
		require.NoError(t, err)
		require.EqualValues(t, expectRd, actualRd)

		encoded, err := EncodeResultData(&data)
		require.NoError(t, err)
		decodedRd, err := DecodeResultData(encoded)
		require.NoError(t, err)
		require.EqualValues(t, expectRd, decodedRd)
	}
}

func BenchmarkDecodeResultData(b *testing.B) {
	addr := ethcmn.HexToAddress("0x5dE8a020088a2D6d0a23c204FFbeD02790466B49")
	bloom := ethtypes.BytesToBloom([]byte{0x1, 0x3})
	ret := []byte{0x5, 0x8}

	data := ResultData{
		ContractAddress: addr,
		Bloom:           bloom,
		Logs: []*ethtypes.Log{{
			Data:        []byte{1, 2, 3, 4},
			BlockNumber: 17,
		}},
		Ret:    ret,
		TxHash: ethcmn.BigToHash(big.NewInt(10)),
	}

	enc, err := EncodeResultData(&data)
	require.NoError(b, err)
	b.ResetTimer()
	b.Run("amino", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var rd ResultData
			err = ModuleCdc.UnmarshalBinaryLengthPrefixed(enc, &rd)
			if err != nil {
				panic("err should be nil")
			}
		}
	})
	b.Run("unmarshaler", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err = DecodeResultData(enc)
			if err != nil {
				panic("err should be nil")
			}
		}
	})
}

func TestEthStringer(t *testing.T) {
	max := 10
	wg := &sync.WaitGroup{}
	wg.Add(max)
	for i := 0; i < max; i++ {
		go func() {
			addr := GenerateEthAddress()
			h := addr.Hash()
			require.Equal(t, addr.String(), EthAddressStringer(addr).String())
			require.Equal(t, h.String(), EthHashStringer(h).String())
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkEthAddressStringer(b *testing.B) {
	addr := GenerateEthAddress()
	b.ResetTimer()
	b.Run("eth", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = addr.String()
		}
	})
	b.Run("okbc stringer", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = EthAddressStringer(addr).String()
		}
	})
}

func BenchmarkEthHashStringer(b *testing.B) {
	addr := GenerateEthAddress()
	h := addr.Hash()
	b.ResetTimer()
	b.Run("eth", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = h.String()
		}
	})
	b.Run("okbc stringer", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = EthHashStringer(h).String()
		}
	})
}

// ForEachStorage iterates over each storage items, all invoke the provided
// callback on each key, value pair.
func (csdb *CommitStateDB) ForEachStorageForTest(ctx sdk.Context, stateobj StateObject, cb func(key, value ethcmn.Hash) (stop bool)) error {
	obj := stateobj.(*stateObject)
	store := ctx.KVStore(csdb.storeKey)
	prefix := AddressStoragePrefix(obj.Address())
	iterator := sdk.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := ethcmn.BytesToHash(iterator.Key())
		value := ethcmn.BytesToHash(iterator.Value())

		if value, dirty := obj.dirtyStorage[key]; dirty {
			if cb(key, value) {
				break
			}
			continue
		}

		// check if iteration stops
		if cb(key, value) {
			return nil
		}
	}

	return nil
}

func (csdb *CommitStateDB) DeepCopyForTest(src *CommitStateDB) *CommitStateDB {

	newStateObjDirty := map[ethcmn.Address]struct{}{}
	for k, v := range src.stateObjectsDirty {
		newStateObjDirty[k] = v
	}

	newStateObjPending := map[ethcmn.Address]struct{}{}
	for k, v := range src.stateObjectsPending {
		newStateObjPending[k] = v
	}

	newStateObj := make(map[ethcmn.Address]*stateObject, 0)
	for k, v := range src.stateObjects {
		newStateObj[k] = v.deepCopy(csdb)
	}

	dst := &CommitStateDB{
		stateObjectsDirty:   newStateObjDirty,
		stateObjectsPending: newStateObjPending,
		stateObjects:        newStateObj,
	}
	return dst
}

func (csdb *CommitStateDB) EqualForTest(dst *CommitStateDB) bool {
	if len(csdb.stateObjectsDirty) != len(dst.stateObjectsDirty) {
		return false
	}

	if len(csdb.stateObjectsPending) != len(dst.stateObjectsPending) {
		return false
	}
	if len(csdb.stateObjects) != len(dst.stateObjects) {
		return false
	}

	for k, _ := range csdb.stateObjects {
		temp := dst.stateObjects[k]
		temp1 := dst.stateObjects[k]
		if temp.account.String() != temp1.account.String() {
			return false
		}
		temp.account = nil

		temp1.account = nil
		if temp != temp1 {
			return false
		}
	}
	return true
}
