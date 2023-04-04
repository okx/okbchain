package base

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth/exported"
	auth "github.com/okx/okbchain/libs/cosmos-sdk/x/auth/types"
	"github.com/status-im/keycard-go/hexutils"
)

type AccountStateRootRetriever struct{}

func (a AccountStateRootRetriever) RetrieveStateRoot(bz []byte) common.Hash {
	acc := DecodeAccount("", bz)
	return acc.GetStateRoot()
}

func (a AccountStateRootRetriever) ModifyAccStateRoot(before []byte, rootHash common.Hash) []byte {
	//TODO implement me
	panic("implement me")
}

func (a AccountStateRootRetriever) GetAccStateRoot(rootBytes []byte) common.Hash {
	//TODO implement me
	panic("implement me")
}

func (a AccountStateRootRetriever) GetStateRootAndCodeHash(bz []byte) (common.Hash, []byte) {
	acc := DecodeAccount("", bz)

	return acc.GetStateRoot(), acc.GetCodeHash()
}

func DecodeAccount(key string, bz []byte) exported.Account {
	val, err := auth.ModuleCdc.UnmarshalBinaryBareWithRegisteredUnmarshaller(bz, (*exported.Account)(nil))
	if err == nil {
		return val.(exported.Account)
	}
	var acc exported.Account
	err = auth.ModuleCdc.UnmarshalBinaryBare(bz, &acc)
	if err != nil {
		fmt.Printf(" key(%s) value(%s) err(%s)\n", key, hexutils.BytesToHex(bz), err)
		panic(err)
	}
	return acc
}
