package types

import (
	"bytes"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/okx/okbchain/libs/tendermint/crypto/merkle"
)

// TxWithMeta contains tx hash
type TxWithMeta struct {
	Tx
	TxHash []byte
}

func NewTxWithMeta(tx Tx) *TxWithMeta {
	return &TxWithMeta{Tx: tx}
}

func (tx *TxWithMeta) Hash() []byte {
	if len(tx.TxHash) == 0 {
		tx.TxHash = ethcommon.CopyBytes(tx.Tx.Hash())
	}
	return ethcommon.CopyBytes(tx.TxHash)
}

func (tx *TxWithMeta) GetTx() []byte {
	return tx.Tx
}

type TxWithMetas []*TxWithMeta

// Hash returns the Merkle root hash of the transaction hashes.
// i.e. the leaves of the tree are the hashes of the txs.
func (txs TxWithMetas) Hash() []byte {
	// These allocations will be removed once Txs is switched to [][]byte,
	// ref #2603. This is because golang does not allow type casting slices without unsafe
	txBzs := make([][]byte, len(txs))
	for i := 0; i < len(txs); i++ {
		txBzs[i] = txs[i].Hash()
	}
	return merkle.SimpleHashFromByteSlices(txBzs)
}

// Index returns the index of this transaction in the list, or -1 if not found
func (txs TxWithMetas) Index(tx Tx) int {
	for i := range txs {
		if bytes.Equal(txs[i].Tx, tx) {
			return i
		}
	}
	return -1
}

// IndexByHash returns the index of this transaction hash in the list, or -1 if not found
func (txs TxWithMetas) IndexByHash(hash []byte) int {
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
func (txs TxWithMetas) Proof(i int) TxProof {
	l := len(txs)
	bzs := make([][]byte, l)
	for i := 0; i < l; i++ {
		bzs[i] = txs[i].Hash()
	}
	root, proofs := merkle.SimpleProofsFromByteSlices(bzs)

	return TxProof{
		RootHash: root,
		Data:     txs[i].Tx,
		Proof:    *proofs[i],
	}
}

func (txs TxWithMetas) GetTxWithMetas() interface{} {
	return txs
}

func TxsToTxWithMetas(ts Txs) TxWithMetas {
	txs := make(TxWithMetas, len(ts))
	num := len(ts)
	for i := 0; i < num; i++ {
		txs[i] = NewTxWithMeta(ts[i])
	}
	return txs
}
