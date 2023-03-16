package mpt

import (
	ethcmn "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	authexported "github.com/okx/okbchain/libs/cosmos-sdk/x/auth/exported"
)

type StateRootRetriever interface {
	RetrieveStateRoot([]byte) ethcmn.Hash
	ModifyAccStateRoot(before []byte, rootHash ethcmn.Hash) []byte
	DecodeAccount(bz []byte) authexported.Account
}

type EmptyStateRootRetriever struct{}

func (e EmptyStateRootRetriever) RetrieveStateRoot([]byte) ethcmn.Hash {
	return ethtypes.EmptyRootHash
}

func (e EmptyStateRootRetriever) ModifyAccStateRoot(before []byte, rootHash ethcmn.Hash) []byte {
	return before
}

func (e EmptyStateRootRetriever) DecodeAccount(bz []byte) authexported.Account {
	return nil
}
