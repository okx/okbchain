package mpt

import (
	ethcmn "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type StateRootRetriever interface {
	RetrieveStateRoot([]byte) ethcmn.Hash
	ModifyAccStateRoot(before []byte, rootHash ethcmn.Hash) []byte
}

type EmptyStateRootRetriever struct{}

func (e EmptyStateRootRetriever) RetrieveStateRoot([]byte) ethcmn.Hash {
	return ethtypes.EmptyRootHash
}

func (e EmptyStateRootRetriever) ModifyAccStateRoot(before []byte, rootHash ethcmn.Hash) []byte {
	return before
}
