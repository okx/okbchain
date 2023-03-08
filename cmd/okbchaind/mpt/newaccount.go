package mpt

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/okx/okbchain/app"
	apptypes "github.com/okx/okbchain/app/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/server"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/mpt"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	authexported "github.com/okx/okbchain/libs/cosmos-sdk/x/auth/exported"
	"github.com/okx/okbchain/libs/tendermint/libs/log"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
)

type TempNewAccountPretty struct {
	Address       sdk.AccAddress    `json:"address" yaml:"address"`
	EthAddress    string            `json:"eth_address" yaml:"eth_address"`
	Coins         sdk.Coins         `json:"coins" yaml:"coins"`
	PubKey        string            `json:"public_key" yaml:"public_key"`
	AccountNumber uint64            `json:"account_number" yaml:"account_number"`
	Sequence      uint64            `json:"sequence" yaml:"sequence"`
	CodeHash      string            `json:"code_hash" yaml:"code_hash"`
	Storage       map[string]string `json:"storages" yaml:"storages"`
}

type TempModuleAccountPretty struct {
	Address       sdk.AccAddress `json:"address" yaml:"address"`
	EthAddress    string         `json:"eth_address" yaml:"eth_address"`
	Coins         sdk.Coins      `json:"coins" yaml:"coins"`
	PubKey        string         `json:"public_key" yaml:"public_key"`
	AccountNumber uint64         `json:"account_number" yaml:"account_number"`
	Sequence      uint64         `json:"sequence" yaml:"sequence"`
	Name          string         `json:"name" yaml:"name"`               // name of the module
	Permissions   []string       `json:"permissions" yaml:"permissions"` // permissions of module account
}

func AccountGetCmd(ctx *server.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [data]",
		Args:  cobra.ExactArgs(1),
		Short: "get account",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("--------- iterate %s data start ---------\n", args[0])
			iavlAccount := GetAccount(args[0])
			buff, err := json.Marshal(iavlAccount)
			if err != nil {
				fmt.Printf("Error:%s", err)
				return
			}
			if err := ioutil.WriteFile(args[0]+"iavlaccount", buff, 0555); err != nil {
				fmt.Printf("Error:%s", err)
				return
			}
			fmt.Printf("--------- iterate %s data end ---------\n", args[0])
		},
	}
	return cmd
}

// migrateAccFromIavlToMpt migrate acc data from iavl to mpt
func GetAccount(datadir string) map[string]interface{} {
	result := make(map[string]interface{}, 0)
	// 0.1 initialize App and context
	appDb := openApplicationDb(datadir)
	migrationApp := app.NewOKBChainApp(
		log.NewTMLogger(log.NewSyncWriter(os.Stdout)),
		appDb,
		nil,
		true,
		map[int64]bool{},
		0,
	)

	cmCtx := migrationApp.MockContext()

	// 1.2 update every account to mpt
	count, contractCount := 0, 0
	migrationApp.AccountKeeper.MigrateAccounts(cmCtx, func(account authexported.Account, key, value []byte) (stop bool) {
		count++
		if len(value) == 0 {
			fmt.Printf("[warning] %s has nil value\n", account.GetAddress().String())
		}
		buff, err := json.Marshal(account)
		panicError(err)

		// check if the account is a contract account
		if ethAcc, ok := account.(*apptypes.EthAccount); ok {
			var okbAcc = TempNewAccountPretty{Storage: make(map[string]string)}
			err = json.Unmarshal(buff, &okbAcc)
			panicError(err)

			if !bytes.Equal(ethAcc.CodeHash, mpt.EmptyCodeHashBytes) {
				contractCount++

				_ = migrationApp.EvmKeeper.ForEachStorage(cmCtx, ethAcc.EthAddress(), func(key, value ethcmn.Hash) bool {
					// Encoding []byte cannot fail, ok to ignore the error.
					v, _ := rlp.EncodeToBytes(ethcmn.TrimLeftZeroes(value[:]))
					okbAcc.Storage[key.String()] = hex.EncodeToString(v)
					return false
				})
			}

			result[account.GetAddress().String()] = &okbAcc
		} else {
			result[account.GetAddress().String()] = account
		}

		return false
	})

	return result
}
