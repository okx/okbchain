package cli

import (
	"fmt"

	ethcommon "github.com/ethereum/go-ethereum/common"

	"strings"

	"github.com/okx/okbchain/libs/cosmos-sdk/client"
	"github.com/okx/okbchain/libs/cosmos-sdk/client/context"
	"github.com/okx/okbchain/libs/cosmos-sdk/client/flags"
	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/version"
	"github.com/okx/okbchain/x/evm/client/rest"
	"github.com/okx/okbchain/x/evm/client/utils"
	"github.com/okx/okbchain/x/evm/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// GetQueryCmd defines evm module queries through the cli
func GetQueryCmd(moduleName string, cdc *codec.Codec) *cobra.Command {
	evmQueryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the evm module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	evmQueryCmd.AddCommand(flags.GetCommands(
		QueryEvmTxCmd(cdc),
		GetCmdGetStorageAt(moduleName, cdc),
		GetCmdGetCode(moduleName, cdc),
		GetCmdQueryParams(moduleName, cdc),
		GetCmdQueryContractDeploymentWhitelist(moduleName, cdc),
		GetCmdQueryContractBlockedList(moduleName, cdc),
		GetCmdQueryContractMethodeBlockedList(moduleName, cdc),
		GetCmdQueryManageSysContractAddress(moduleName, cdc),
	)...)
	return evmQueryCmd
}

func add0xPrefix(al types.AddressList) []string {
	var res []string
	for i := 0; i < len(al); i++ {
		// decode from bech32 when using cosmos address
		str, err := accountToHex(al[i].String())
		if err != nil {
			continue
		}
		res = append(res, str)
	}
	return res
}

// GetCmdQueryManageSysContractAddress gets the contract blocked list query command.
func GetCmdQueryManageSysContractAddress(storeName string, cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "system-contract-address",
		Short: "Query system contract address",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query the current system contract address.

Example:
$ %s query evm system-contract-address
`,
				version.ClientName,
			),
		),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)
			route := fmt.Sprintf("custom/%s/%s", storeName, types.QuerySysContractAddress)

			addr, _, err := cliCtx.QueryWithData(route, nil)
			if err != nil {
				return err
			}

			ethAddr := ethcommon.BytesToAddress(addr).Hex()
			result := utils.ResponseSysContractAddress{Address: ethAddr}
			return cliCtx.PrintOutput(result)
		},
	}
}

// GetCmdQueryContractBlockedList gets the contract blocked list query command.
func GetCmdQueryContractMethodeBlockedList(storeName string, cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "contract-method-blocked-list",
		Short: "Query the contract methode blocked list",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query the current blocked list of contract addresses during evm calling.

Example:
$ %s query evm contract-blocked-list
`,
				version.ClientName,
			),
		),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)
			route := fmt.Sprintf("custom/%s/%s", storeName, types.QueryContractMethodBlockedList)
			bz, _, err := cliCtx.QueryWithData(route, nil)
			if err != nil {
				return err
			}

			var blockedList types.BlockedContractList
			cdc.MustUnmarshalJSON(bz, &blockedList)

			results := make([]utils.ResponseBlockContract, 0)
			for i, _ := range blockedList {
				ethAddr := ethcommon.BytesToAddress(blockedList[i].Address.Bytes()).Hex()
				result := utils.ResponseBlockContract{Address: ethAddr, BlockMethods: blockedList[i].BlockMethods}
				results = append(results, result)
			}
			return cliCtx.PrintOutput(results)
		},
	}
}

// GetCmdQueryContractBlockedList gets the contract blocked list query command.
func GetCmdQueryContractBlockedList(storeName string, cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "contract-blocked-list",
		Short: "Query the contract blocked list.Deprecated",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query the current blocked list of contract addresses during evm calling.

Example:
$ %s query evm contract-blocked-list
`,
				version.ClientName,
			),
		),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)
			route := fmt.Sprintf("custom/%s/%s", storeName, types.QueryContractBlockedList)
			bz, _, err := cliCtx.QueryWithData(route, nil)
			if err != nil {
				return err
			}

			var blockedList types.AddressList
			cdc.MustUnmarshalJSON(bz, &blockedList)
			return cliCtx.PrintOutput(add0xPrefix(blockedList))
		},
	}
}

// GetCmdQueryContractDeploymentWhitelist gets the contract deployment whitelist query command.
func GetCmdQueryContractDeploymentWhitelist(storeName string, cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "contract-deployment-whitelist",
		Short: "Query the whitelist of contract deployment",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query the current whitelist of distributors for contract deployment.

Example:
$ %s query evm contract-deployment-whitelist
`,
				version.ClientName,
			),
		),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)
			route := fmt.Sprintf("custom/%s/%s", storeName, types.QueryContractDeploymentWhitelist)
			bz, _, err := cliCtx.QueryWithData(route, nil)
			if err != nil {
				return err
			}

			var whitelist types.AddressList
			cdc.MustUnmarshalJSON(bz, &whitelist)
			return cliCtx.PrintOutput(whitelist)
		},
	}
}

// QueryEvmTxCmd implements the command for the query of transactions including evm
func QueryEvmTxCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx [hash]",
		Short: "Query for all transactions including evm by hash in a committed block",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			res, err := rest.QueryTx(cliCtx, args[0])
			if err != nil {
				return err
			}

			output, ok := res.(sdk.TxResponse)
			if !ok {
				// evm tx result
				fmt.Println(string(res.([]byte)))
				return nil
			}

			if output.Empty() {
				return fmt.Errorf("no transaction found with hash %s", args[0])
			}

			return cliCtx.PrintOutput(output)
		},
	}

	return cmd
}

// GetCmdGetStorageAt queries a key in an accounts storage
func GetCmdGetStorageAt(queryRoute string, cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "storage [account] [key]",
		Short: "Gets storage for an account at a given key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := context.NewCLIContext().WithCodec(cdc)

			account, err := accountToHex(args[0])
			if err != nil {
				return errors.Wrap(err, "could not parse account address")
			}

			key := formatKeyToHash(args[1])

			res, _, err := clientCtx.Query(
				fmt.Sprintf("custom/%s/storage/%s/%s", queryRoute, account, key))

			if err != nil {
				return fmt.Errorf("could not resolve: %s", err)
			}
			var out types.QueryResStorage
			cdc.MustUnmarshalJSON(res, &out)
			return clientCtx.PrintOutput(out)
		},
	}
}

// GetCmdGetCode queries the code field of a given address
func GetCmdGetCode(queryRoute string, cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "code [account]",
		Short: "Gets code from an account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := context.NewCLIContext().WithCodec(cdc)

			account, err := accountToHex(args[0])
			if err != nil {
				return errors.Wrap(err, "could not parse account address")
			}

			res, _, err := clientCtx.Query(
				fmt.Sprintf("custom/%s/code/%s", queryRoute, account))

			if err != nil {
				return fmt.Errorf("could not resolve: %s", err)
			}

			var out types.QueryResCode
			cdc.MustUnmarshalJSON(res, &out)
			return clientCtx.PrintOutput(out)
		},
	}
}

// GetCmdQueryParams implements the query params command.
func GetCmdQueryParams(queryRoute string, cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "params",
		Short: "Query all the modifiable parameters of gov proposal",
		Long: strings.TrimSpace(`Query the all the parameters for the governance process:

$ cli query evm params
`),
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			route := fmt.Sprintf("custom/%s/%s", queryRoute, types.QueryParameters)
			bz, _, err := cliCtx.QueryWithData(route, nil)
			if err != nil {
				return err
			}

			var params types.Params
			cdc.MustUnmarshalJSON(bz, &params)
			return cliCtx.PrintOutput(params)
		},
	}
}
