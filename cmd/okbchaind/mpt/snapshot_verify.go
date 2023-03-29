package mpt

import (
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/okx/okbchain/libs/cosmos-sdk/server"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/mpt"
	"github.com/spf13/cobra"
	stdlog "log"
)

func verifySnapshotCmd(ctx *server.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify-snapshot",
		Short: "verify mpt store's snapshot",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			stdlog.Println("--------- verify snapshot start ---------")
			verifySnapshot()
			stdlog.Println("--------- verify snapshot end ---------")
		},
	}

	return cmd
}

func verifySnapshot() {
	latestHeight := getLatestHeight()
	rootHash := getMptRootHash(latestHeight)
	db := mpt.InstanceOfMptStore()
	_, err := db.OpenTrie(rootHash)
	panicError(err)
	snaps, err := snapshot.NewCustom(db.TrieDB().DiskDB(), db.TrieDB(), 256, rootHash, false, false, false, accountStateRootRetriever{})
	panicError(err)
	err = snaps.Verify(rootHash)
	if err != nil {
		stdlog.Printf("snapshot state is stale, please generate it again. error %v\n", err)
	}

	stdlog.Println("snapshot state verify ok.")
}
