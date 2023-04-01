package mpt

import (
	"encoding/binary"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/okx/okbchain/cmd/okbchaind/base"
	"github.com/okx/okbchain/libs/cosmos-sdk/server"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/mpt"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

func snapshotViewerCmd(ctx *server.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot-viewer",
		Short: "view mpt store's snapshot",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			viewSnapshot()
		},
	}

	return cmd
}

func viewSnapshot() {
	latestHeight := getLatestHeight()
	rootHash := getMptRootHash(latestHeight)
	db := mpt.InstanceOfMptStore()
	_, err := db.OpenTrie(rootHash)
	panicError(err)
	snaps, err := snapshot.NewCustom(db.TrieDB().DiskDB(), db.TrieDB(), 256, rootHash, false, false, false, base.AccountStateRootRetriever{})
	panicError(err)
	iter, err := snaps.AccountIterator(rootHash, common.Hash{})
	for iter.Next() {
		acc := base.DecodeAccount(iter.Hash().String(), iter.Account())
		fmt.Printf("%v: %v\n", iter.Hash().String(), acc)
		fmt.Printf("%v: %v\n", iter.Hash().String(), acc)
	}
}

func viewSnapshot_() {
	latestHeight := getLatestHeight()
	rootHash := getMptRootHash(latestHeight)
	db := mpt.InstanceOfMptStore()
	_, err := db.OpenTrie(rootHash)
	panicError(err)
	snaps, err := snapshot.NewCustom(db.TrieDB().DiskDB(), db.TrieDB(), 256, rootHash, false, false, false, base.AccountStateRootRetriever{})
	panicError(err)
	iter, err := snaps.AccountIterator(rootHash, common.Hash{})
	for iter.Next() {
		sIter, _ := snaps.StorageIterator(rootHash, iter.Hash(), common.Hash{})
		for sIter.Next() {
			fmt.Printf("%s: %s\n", sIter.Hash().String(), common.Bytes2Hex(sIter.Slot()))
		}
	}
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
