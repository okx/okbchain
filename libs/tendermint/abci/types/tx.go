package types

import "math/big"

type TxEssentials interface {
	GetRaw() []byte
	TxHash() []byte
	GetEthAddr() string
	GetNonce() uint64
	GetGasPrice() *big.Int
}

type MockTx struct {
	Raw      []byte
	Hash     []byte
	From     string
	Nonce    uint64
	GasPrice *big.Int
}

func (tx MockTx) GetRaw() []byte {
	return tx.Raw
}

func (tx MockTx) TxHash() []byte {
	return tx.Hash
}

func (tx MockTx) GetEthAddr() string {
	return tx.From
}

func (tx MockTx) GetNonce() uint64 {
	return tx.Nonce
}

func (tx MockTx) GetGasPrice() *big.Int {
	return tx.GasPrice
}
