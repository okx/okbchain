package cli

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	wasmvm "github.com/CosmWasm/wasmvm"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"

	"github.com/okx/okbchain/x/wasm/keeper"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"github.com/okx/okbchain/libs/cosmos-sdk/client"
	clientCtx "github.com/okx/okbchain/libs/cosmos-sdk/client/context"
	"github.com/okx/okbchain/libs/cosmos-sdk/client/flags"
	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	codectypes "github.com/okx/okbchain/libs/cosmos-sdk/codec/types"
	"github.com/okx/okbchain/x/wasm/types"
)

// NewQueryCmd returns the query commands for wasm
func NewQueryCmd(cdc *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the wasm module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewCmdListCode(cdc, reg),
		NewCmdListContractByCode(cdc, reg),
		NewCmdQueryCode(cdc, reg),
		NewCmdQueryCodeInfo(cdc, reg),
		NewCmdGetContractInfo(cdc, reg),
		NewCmdGetContractHistory(cdc, reg),
		NewCmdGetContractState(cdc, reg),
		NewCmdListPinnedCode(cdc, reg),
		NewCmdLibVersion(cdc, reg),
		NewCmdListContractBlockedMethod(cdc),
		NewCmdGetParams(cdc, reg),
		NewCmdGetAddressWhitelist(cdc, reg),
	)

	return queryCmd
}

// NewCmdLibVersion gets current libwasmvm version.
func NewCmdLibVersion(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "libwasmvm-version",
		Short:   "Get libwasmvm version",
		Long:    "Get libwasmvm version",
		Aliases: []string{"lib-version"},
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			version, err := wasmvm.LibwasmvmVersion()
			if err != nil {
				return fmt.Errorf("error retrieving libwasmvm version: %w", err)
			}
			fmt.Println(version)
			return nil
		},
	}
	return cmd
}

// NewCmdListCode lists all wasm code uploaded
func NewCmdListCode(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list-code",
		Short:   "List all wasm bytecode on the chain",
		Long:    "List all wasm bytecode on the chain",
		Aliases: []string{"list-codes", "codes", "lco"},
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := clientCtx.NewCLIContext().WithProxy(m).WithInterfaceRegistry(reg)
			queryClient := types.NewQueryClient(clientCtx)

			pageReq, err := client.ReadPageRequest(withPageKeyDecoded(cmd.Flags()))
			if err != nil {
				return err
			}
			res, err := queryClient.Codes(
				context.Background(),
				&types.QueryCodesRequest{
					Pagination: pageReq,
				},
			)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "list codes")
	return cmd
}

func NewCmdGetParams(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get-params",
		Short:   "Get wasm parameters on the chain",
		Long:    "Get wasm parameters on the chain",
		Aliases: []string{"get-params", "params"},
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := clientCtx.NewCLIContext().WithProxy(m).WithInterfaceRegistry(reg)
			route := fmt.Sprintf("custom/%s/%s", types.QuerierRoute, keeper.QueryParams)

			res, _, err := clientCtx.Query(route)
			if err != nil {
				return err
			}

			var params types.Params
			m.GetCdc().MustUnmarshalJSON(res, &params)
			return clientCtx.PrintOutput(params)
		},
	}
	return cmd
}

func NewCmdGetAddressWhitelist(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get-address-whitelist",
		Short:   "Get wasm address whitelist on the chain",
		Long:    "Get wasm address whitelist on the chain",
		Aliases: []string{"whitelist", "gawl"},
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := clientCtx.NewCLIContext().WithProxy(m).WithInterfaceRegistry(reg)
			route := fmt.Sprintf("custom/%s/%s", types.QuerierRoute, keeper.QueryParams)

			res, _, err := clientCtx.Query(route)
			if err != nil {
				return err
			}

			var params types.Params
			m.GetCdc().MustUnmarshalJSON(res, &params)
			var whitelist []string
			whitelist = strings.Split(params.CodeUploadAccess.Address, ",")
			if len(whitelist) == 1 && whitelist[0] == "" {
				whitelist = []string{}
			}
			response := types.NewQueryAddressWhitelistResponse(whitelist)
			return clientCtx.PrintOutput(response)
		},
	}
	return cmd
}

// NewCmdListContractByCode lists all wasm code uploaded for given code id
func NewCmdListContractByCode(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list-contract-by-code [code_id]",
		Short:   "List wasm all bytecode on the chain for given code id",
		Long:    "List wasm all bytecode on the chain for given code id",
		Aliases: []string{"list-contracts-by-code", "list-contracts", "contracts", "lca"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := clientCtx.NewCLIContext().WithProxy(m).WithInterfaceRegistry(reg)
			queryClient := types.NewQueryClient(clientCtx)

			codeID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(withPageKeyDecoded(cmd.Flags()))
			if err != nil {
				return err
			}
			res, err := queryClient.ContractsByCode(
				context.Background(),
				&types.QueryContractsByCodeRequest{
					CodeId:     codeID,
					Pagination: pageReq,
				},
			)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "list contracts by code")
	return cmd
}

// NewCmdQueryCode returns the bytecode for a given contract
func NewCmdQueryCode(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "code [code_id] [output filename]",
		Short:   "Downloads wasm bytecode for given code id",
		Long:    "Downloads wasm bytecode for given code id",
		Aliases: []string{"source-code", "source"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := clientCtx.NewCLIContext().WithProxy(m).WithInterfaceRegistry(reg)
			queryClient := types.NewQueryClient(clientCtx)

			codeID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			res, err := queryClient.Code(
				context.Background(),
				&types.QueryCodeRequest{
					CodeId: codeID,
				},
			)
			if err != nil {
				return err
			}
			if len(res.Data) == 0 {
				return fmt.Errorf("contract not found")
			}

			fmt.Printf("Downloading wasm code to %s\n", args[1])
			return ioutil.WriteFile(args[1], res.Data, 0o600)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewCmdQueryCodeInfo returns the code info for a given code id
func NewCmdQueryCodeInfo(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "code-info [code_id]",
		Short: "Prints out metadata of a code id",
		Long:  "Prints out metadata of a code id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := clientCtx.NewCLIContext().WithProxy(m).WithInterfaceRegistry(reg)

			codeID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Code(
				context.Background(),
				&types.QueryCodeRequest{
					CodeId: codeID,
				},
			)
			if err != nil {
				return err
			}
			if res.CodeInfoResponse == nil {
				return fmt.Errorf("contract not found")
			}

			return clientCtx.PrintProto(res.CodeInfoResponse)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewCmdGetContractInfo gets details about a given contract
func NewCmdGetContractInfo(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "contract [bech32_address]",
		Short:   "Prints out metadata of a contract given its address",
		Long:    "Prints out metadata of a contract given its address",
		Aliases: []string{"meta", "c"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := clientCtx.NewCLIContext().WithProxy(m).WithInterfaceRegistry(reg)
			queryClient := types.NewQueryClient(clientCtx)

			_, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}
			res, err := queryClient.ContractInfo(
				context.Background(),
				&types.QueryContractInfoRequest{
					Address: args[0],
				},
			)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewCmdListContractBlockedMethod(m *codec.CodecProxy) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list-contract-blocked-method [bech32_address]",
		Short:   "List blocked methods of a contract given its address",
		Long:    "List blocked methods of a contract given its address",
		Aliases: []string{"lcbm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := clientCtx.NewCLIContext().WithCodec(m.GetCdc())

			_, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}
			res, _, err := clientCtx.Query(fmt.Sprintf("custom/wasm/list-contract-blocked-method/%s", args[0]))
			if err != nil {
				return err
			}
			return clientCtx.PrintOutput(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewCmdGetContractHistory prints the code history for a given contract
func NewCmdGetContractHistory(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "contract-history [bech32_address]",
		Short:   "Prints out the code history for a contract given its address",
		Long:    "Prints out the code history for a contract given its address",
		Aliases: []string{"history", "hist", "ch"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := clientCtx.NewCLIContext().WithProxy(m).WithInterfaceRegistry(reg)
			queryClient := types.NewQueryClient(clientCtx)

			_, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(withPageKeyDecoded(cmd.Flags()))
			if err != nil {
				return err
			}
			res, err := queryClient.ContractHistory(
				context.Background(),
				&types.QueryContractHistoryRequest{
					Address:    args[0],
					Pagination: pageReq,
				},
			)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "contract history")
	return cmd
}

// NewCmdGetContractState dumps full internal state of a given contract
func NewCmdGetContractState(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "contract-state",
		Short:                      "Querying commands for the wasm module",
		Aliases:                    []string{"state", "cs", "s"},
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		newCmdGetContractStateAll(m, reg),
		newCmdGetContractStateRaw(m, reg),
		newCmdGetContractStateSmart(m, reg),
	)
	return cmd
}

func newCmdGetContractStateAll(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all [bech32_address]",
		Short: "Prints out all internal state of a contract given its address",
		Long:  "Prints out all internal state of a contract given its address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := clientCtx.NewCLIContext().WithProxy(m).WithInterfaceRegistry(reg)
			queryClient := types.NewQueryClient(clientCtx)

			_, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(withPageKeyDecoded(cmd.Flags()))
			if err != nil {
				return err
			}
			res, err := queryClient.AllContractState(
				context.Background(),
				&types.QueryAllContractStateRequest{
					Address:    args[0],
					Pagination: pageReq,
				},
			)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "contract state")
	return cmd
}

func newCmdGetContractStateRaw(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	decoder := newArgDecoder(hex.DecodeString)
	cmd := &cobra.Command{
		Use:   "raw [bech32_address] [key]",
		Short: "Prints out internal state for key of a contract given its address",
		Long:  "Prints out internal state for of a contract given its address",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := clientCtx.NewCLIContext().WithProxy(m).WithInterfaceRegistry(reg)
			queryClient := types.NewQueryClient(clientCtx)

			_, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}
			queryData, err := decoder.DecodeString(args[1])
			if err != nil {
				return err
			}

			res, err := queryClient.RawContractState(
				context.Background(),
				&types.QueryRawContractStateRequest{
					Address:   args[0],
					QueryData: queryData,
				},
			)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	decoder.RegisterFlags(cmd.PersistentFlags(), "key argument")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func newCmdGetContractStateSmart(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	decoder := newArgDecoder(asciiDecodeString)
	cmd := &cobra.Command{
		Use:   "smart [bech32_address] [query]",
		Short: "Calls contract with given address with query data and prints the returned result",
		Long:  "Calls contract with given address with query data and prints the returned result",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := clientCtx.NewCLIContext().WithProxy(m).WithInterfaceRegistry(reg)
			queryClient := types.NewQueryClient(clientCtx)

			_, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}
			if args[1] == "" {
				return errors.New("query data must not be empty")
			}
			queryData, err := decoder.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("decode query: %s", err)
			}
			if !json.Valid(queryData) {
				return errors.New("query data must be json")
			}

			res, err := queryClient.SmartContractState(
				context.Background(),
				&types.QuerySmartContractStateRequest{
					Address:   args[0],
					QueryData: queryData,
				},
			)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	decoder.RegisterFlags(cmd.PersistentFlags(), "query argument")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewCmdListPinnedCode lists all wasm code ids that are pinned
func NewCmdListPinnedCode(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pinned",
		Short: "List all pinned code ids",
		Long:  "\t\tLong:    List all pinned code ids,\n",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := clientCtx.NewCLIContext().WithProxy(m).WithInterfaceRegistry(reg)
			queryClient := types.NewQueryClient(clientCtx)

			pageReq, err := client.ReadPageRequest(withPageKeyDecoded(cmd.Flags()))
			if err != nil {
				return err
			}
			res, err := queryClient.PinnedCodes(
				context.Background(),
				&types.QueryPinnedCodesRequest{
					Pagination: pageReq,
				},
			)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "list codes")
	return cmd
}

type argumentDecoder struct {
	// dec is the default decoder
	dec                func(string) ([]byte, error)
	asciiF, hexF, b64F bool
}

func newArgDecoder(def func(string) ([]byte, error)) *argumentDecoder {
	return &argumentDecoder{dec: def}
}

func (a *argumentDecoder) RegisterFlags(f *flag.FlagSet, argName string) {
	f.BoolVar(&a.asciiF, "ascii", false, "ascii encoded "+argName)
	f.BoolVar(&a.hexF, "hex", false, "hex encoded  "+argName)
	f.BoolVar(&a.b64F, "b64", false, "base64 encoded "+argName)
}

func (a *argumentDecoder) DecodeString(s string) ([]byte, error) {
	found := -1
	for i, v := range []*bool{&a.asciiF, &a.hexF, &a.b64F} {
		if !*v {
			continue
		}
		if found != -1 {
			return nil, errors.New("multiple decoding flags used")
		}
		found = i
	}
	switch found {
	case 0:
		return asciiDecodeString(s)
	case 1:
		return hex.DecodeString(s)
	case 2:
		return base64.StdEncoding.DecodeString(s)
	default:
		return a.dec(s)
	}
}

func asciiDecodeString(s string) ([]byte, error) {
	return []byte(s), nil
}

// sdk ReadPageRequest expects binary but we encoded to base64 in our marshaller
func withPageKeyDecoded(flagSet *flag.FlagSet) *flag.FlagSet {
	encoded, err := flagSet.GetString(flags.FlagPageKey)
	if err != nil {
		panic(err.Error())
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		panic(err.Error())
	}
	err = flagSet.Set(flags.FlagPageKey, string(raw))
	if err != nil {
		panic(err.Error())
	}
	return flagSet
}
