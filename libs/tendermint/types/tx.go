package types

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/okx/okbchain/libs/tendermint/crypto/tmhash"

	ethcmn "github.com/ethereum/go-ethereum/common"

	abci "github.com/okx/okbchain/libs/tendermint/abci/types"
	"github.com/okx/okbchain/libs/tendermint/crypto/etherhash"
	"github.com/okx/okbchain/libs/tendermint/crypto/merkle"
	tmbytes "github.com/okx/okbchain/libs/tendermint/libs/bytes"
	"github.com/tendermint/go-amino"
)

// Tx is an arbitrary byte array.
// NOTE: Tx has no types at this level, so when wire encoded it's just length-prefixed.
// Might we want types here ?
type Tx []byte

func Bytes2Hash(txBytes []byte) string {
	txHash := Tx(txBytes).Hash()
	return ethcmn.BytesToHash(txHash).String()
}

type ethTxData struct {
	AccountNonce uint64          `json:"nonce"`
	Price        *big.Int        `json:"gasPrice"`
	GasLimit     uint64          `json:"gas"`
	Recipient    *ethcmn.Address `json:"to" rlp:"nil"` // nil means contract creation
	Amount       *big.Int        `json:"value"`
	Payload      []byte          `json:"input"`

	// signature values
	V *big.Int `json:"v"`
	R *big.Int `json:"r"`
	S *big.Int `json:"s"`

	// hash is only used when marshaling to JSON
	Hash *ethcmn.Hash `json:"hash" rlp:"-"`
}

// Hash computes the TMHASH hash of the wire encoded transaction.
func (tx Tx) Hash() []byte {
	//// if we can't get length-prefixed bytes, this tx should not be an amino-encoded tx
	//if _, err := amino.GetBinaryBareFromBinaryLengthPrefixed(tx); err != nil {
	//	// if we can't get proto tag, this tx should not be a proto-encoded tx
	//	_, _, length := protowire.ConsumeTag(tx)
	//	if length < 0 {
	//		return etherhash.Sum(tx)
	//	}
	//}
	var msg ethTxData
	if err := rlp.DecodeBytes(tx, &msg); err != nil {
		return tmhash.Sum(tx)
	}
	return etherhash.Sum(tx)
}

// String returns the hex-encoded transaction as a string.
func (tx Tx) String() string {
	return fmt.Sprintf("Tx{%X}", []byte(tx))
}

// Txs is a slice of Tx.
type Txs []Tx

// Hash returns the Merkle root hash of the transaction hashes.
// i.e. the leaves of the tree are the hashes of the txs.
func (txs Txs) Hash() []byte {
	// These allocations will be removed once Txs is switched to [][]byte,
	// ref #2603. This is because golang does not allow type casting slices without unsafe
	txBzs := make([][]byte, len(txs))
	for i := 0; i < len(txs); i++ {
		txBzs[i] = txs[i].Hash()
	}
	return merkle.SimpleHashFromByteSlices(txBzs)
}

// Index returns the index of this transaction in the list, or -1 if not found
func (txs Txs) Index(tx Tx) int {
	for i := range txs {
		if bytes.Equal(txs[i], tx) {
			return i
		}
	}
	return -1
}

// IndexByHash returns the index of this transaction hash in the list, or -1 if not found
func (txs Txs) IndexByHash(hash []byte) int {
	for i := range txs {
		if bytes.Equal(txs[i].Hash(), hash) {
			return i
		}
	}
	return -1
}

// Proof returns a simple merkle proof for this node.
// Panics if i < 0 or i >= len(txs)
// TODO: optimize this!
func (txs Txs) Proof(i int) TxProof {
	l := len(txs)
	bzs := make([][]byte, l)
	for i := 0; i < l; i++ {
		bzs[i] = txs[i].Hash()
	}
	root, proofs := merkle.SimpleProofsFromByteSlices(bzs)

	return TxProof{
		RootHash: root,
		Data:     txs[i],
		Proof:    *proofs[i],
	}
}

// TxProof represents a Merkle proof of the presence of a transaction in the Merkle tree.
type TxProof struct {
	RootHash tmbytes.HexBytes   `json:"root_hash"`
	Data     Tx                 `json:"data"`
	Proof    merkle.SimpleProof `json:"proof"`
}

// Leaf returns the hash(tx), which is the leaf in the merkle tree which this proof refers to.
func (tp TxProof) Leaf() []byte {
	return tp.Data.Hash()
}

// Validate verifies the proof. It returns nil if the RootHash matches the dataHash argument,
// and if the proof is internally consistent. Otherwise, it returns a sensible error.
func (tp TxProof) Validate(dataHash []byte) error {
	if !bytes.Equal(dataHash, tp.RootHash) {
		return errors.New("proof matches different data hash")
	}
	if tp.Proof.Index < 0 {
		return errors.New("proof index cannot be negative")
	}
	if tp.Proof.Total <= 0 {
		return errors.New("proof total must be positive")
	}
	valid := tp.Proof.Verify(tp.RootHash, tp.Leaf())
	if valid != nil {
		return errors.New("proof is not internally consistent")
	}
	return nil
}

// TxResult contains results of executing the transaction.
//
// One usage is indexing transaction results.
type TxResult struct {
	Height int64                  `json:"height"`
	Index  uint32                 `json:"index"`
	Tx     Tx                     `json:"tx"`
	Result abci.ResponseDeliverTx `json:"result"`
}

func (txResult *TxResult) UnmarshalFromAmino(cdc *amino.Codec, data []byte) error {
	var dataLen uint64 = 0
	var subData []byte

	for {
		data = data[dataLen:]

		if len(data) == 0 {
			break
		}

		pos, aminoType, err := amino.ParseProtoPosAndTypeMustOneByte(data[0])
		if err != nil {
			return err
		}
		data = data[1:]

		if aminoType == amino.Typ3_ByteLength {
			var n int
			dataLen, n, err = amino.DecodeUvarint(data)
			if err != nil {
				return err
			}

			data = data[n:]
			if len(data) < int(dataLen) {
				return fmt.Errorf("invalid data length: %d", dataLen)
			}
			subData = data[:dataLen]
		}

		switch pos {
		case 1:
			var n int
			var uvint uint64
			uvint, n, err = amino.DecodeUvarint(data)
			if err != nil {
				return err
			}
			txResult.Height = int64(uvint)
			dataLen = uint64(n)
		case 2:
			var n int
			var uvint uint64
			uvint, n, err = amino.DecodeUvarint(data)
			if err != nil {
				return err
			}
			txResult.Index = uint32(uvint)
			dataLen = uint64(n)
		case 3:
			txResult.Tx = make(Tx, dataLen)
			copy(txResult.Tx, subData)
		case 4:
			err = txResult.Result.UnmarshalFromAmino(cdc, subData)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpect feild num %d", pos)
		}
	}
	return nil
}

// ComputeAminoOverhead calculates the overhead for amino encoding a transaction.
// The overhead consists of varint encoding the field number and the wire type
// (= length-delimited = 2), and another varint encoding the length of the
// transaction.
// The field number can be the field number of the particular transaction, or
// the field number of the parenting struct that contains the transactions []Tx
// as a field (this field number is repeated for each contained Tx).
// If some []Tx are encoded directly (without a parenting struct), the default
// fieldNum is also 1 (see BinFieldNum in amino.MarshalBinaryBare).
func ComputeAminoOverhead(tx Tx, fieldNum int) int64 {
	fnum := uint64(fieldNum)
	typ3AndFieldNum := (fnum << 3) | uint64(amino.Typ3_ByteLength)
	return int64(amino.UvarintSize(typ3AndFieldNum)) + int64(amino.UvarintSize(uint64(len(tx))))
}
