package main

import (
	"github.com/okx/okbchain/libs/cosmos-sdk/server"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/iavl"
	sm "github.com/okx/okbchain/libs/tendermint/state"
	"github.com/spf13/cobra"
	"log"
	"strconv"
)

func BlockInfoCommand(ctx *server.Context) *cobra.Command {
	iavl.SetLogger(ctx.Logger.With("module", "block"))
	return blockInfoCommand
}

var blockInfoCommand = &cobra.Command{
	Use: "block",
}

var txCommand = &cobra.Command{
	Use: "tx",
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir, heightStr := args[0], args[1]
		height, _ := strconv.Atoi(heightStr)

		// viper.Set(sdk.FlagDBBackend, "rocksdb")
		storeDB, err := sdk.NewDB(stateDB, dataDir)
		panicError(err)
		defer storeDB.Close()
		// 	blockStore := store.NewBlockStore(storeDB)

		results, err := sm.LoadABCIResponses(storeDB, int64(height))
		if err != nil {
			panic(err)
		}
		log.Println(results.DeliverTxs)

		return nil
	},
}

func init() {
	blockInfoCommand.AddCommand(txCommand)
}
