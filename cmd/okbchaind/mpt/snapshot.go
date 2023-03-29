package mpt

import (
	"encoding/binary"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/okx/okbchain/libs/cosmos-sdk/server"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/mpt"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/spf13/cobra"
	stdlog "log"
)

func genSnapCmd(ctx *server.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen-snapshot",
		Short: "generate mpt store's snapshot",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			stdlog.Println("--------- generate snapshot start ---------")
			GenSnapshot()
			stdlog.Println("--------- generate snapshot end ---------")
		},
	}
	return cmd
}

func GenSnapshot() {
	latestHeight := getLatestHeight()
	rootHash := getMptRootHash(latestHeight)
	db := mpt.InstanceOfMptStore()

	_, err := db.OpenTrie(rootHash)
	panicError(err)
	snaps, err := snapshot.NewCustom(db.TrieDB().DiskDB(), db.TrieDB(), 256, rootHash, false, true, false, accountStateRootRetriever{})
	panicError(err)

	snaps.Rebuild(rootHash)
}

type accountStateRootRetriever struct{}

func (a accountStateRootRetriever) RetrieveStateRoot(bz []byte) common.Hash {
	acc := DecodeAccount("", bz)
	return acc.GetStateRoot()
}

func (a accountStateRootRetriever) ModifyAccStateRoot(before []byte, rootHash common.Hash) []byte {
	//TODO implement me
	panic("implement me")
}

func (a accountStateRootRetriever) GetAccStateRoot(rootBytes []byte) common.Hash {
	//TODO implement me
	panic("implement me")
}

func getLatestHeight() uint64 {
	db := mpt.InstanceOfMptStore()
	rst, err := db.TrieDB().DiskDB().Get(mpt.KeyPrefixAccLatestStoredHeight)
	if err != nil || len(rst) == 0 {
		panic(fmt.Sprintf("%v %v", err, len(rst)))
	}

	return binary.BigEndian.Uint64(rst)
}

func getMptRootHash(height uint64) common.Hash {
	db := mpt.InstanceOfMptStore()
	hhash := sdk.Uint64ToBigEndian(height)
	rst, err := db.TrieDB().DiskDB().Get(append(mpt.KeyPrefixAccRootMptHash, hhash...))
	if err != nil || len(rst) == 0 {
		return common.Hash{}
	}

	return common.BytesToHash(rst)
}
