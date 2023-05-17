package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"github.com/okx/okbchain/libs/cosmos-sdk/client"
	clientCtx "github.com/okx/okbchain/libs/cosmos-sdk/client/context"
	"github.com/okx/okbchain/libs/cosmos-sdk/client/flags"
	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	codectypes "github.com/okx/okbchain/libs/cosmos-sdk/codec/types"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/okbchain/libs/cosmos-sdk/types/errors"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth/client/utils"
	"github.com/okx/okbchain/x/wasm/ioutils"
	"github.com/okx/okbchain/x/wasm/types"
)

const (
	flagAmount                 = "amount"
	flagLabel                  = "label"
	flagAdmin                  = "admin"
	flagRunAs                  = "run-as"
	flagInstantiateByEverybody = "instantiate-everybody"
	flagInstantiateByAddress   = "instantiate-only-address"
	flagProposalType           = "type"
)

// NewTxCmd returns the transaction commands for wasm
func NewTxCmd(cdc *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Wasm transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	txCmd.AddCommand(
		NewStoreCodeCmd(cdc, reg),
		NewInstantiateContractCmd(cdc, reg),
		NewExecuteContractCmd(cdc, reg),
		NewMigrateContractCmd(cdc, reg),
		NewUpdateContractAdminCmd(cdc, reg),
	)
	return txCmd
}

func NewStoreCodeCmd(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "store [wasm file]",
		Short:   "Upload a wasm binary",
		Aliases: []string{"upload", "st", "s"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(m.GetCdc()))
			clientCtx := clientCtx.NewCLIContext().WithCodec(m.GetCdc()).WithInterfaceRegistry(reg)

			msg, err := parseStoreCodeArgs(args[0], sdk.AccToAWasmddress(clientCtx.GetFromAddress()), cmd.Flags())
			if err != nil {
				return err
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}
			return utils.GenerateOrBroadcastMsgs(clientCtx, txBldr, []sdk.Msg{msg})
		},
	}

	cmd.Flags().String(flagInstantiateByEverybody, "", "Everybody can instantiate a contract from the code, optional")
	cmd.Flags().String(flagInstantiateByAddress, "", "Only this address can instantiate a contract instance from the code, optional")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewInstantiateContractCmd(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "instantiate [code_id_int64] [json_encoded_init_args] --label [text] --admin [address,optional] --amount [coins,optional]",
		Short:   "Instantiate a wasm contract",
		Aliases: []string{"start", "init", "inst", "i"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(m.GetCdc()))
			clientCtx := clientCtx.NewCLIContext().WithCodec(m.GetCdc()).WithInterfaceRegistry(reg)

			msg, err := parseInstantiateArgs(args[0], args[1], sdk.AccToAWasmddress(clientCtx.GetFromAddress()), cmd.Flags())
			if err != nil {
				return err
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return utils.GenerateOrBroadcastMsgs(clientCtx, txBldr, []sdk.Msg{msg})
		},
	}

	cmd.Flags().String(flagAmount, "", "Coins to send to the contract during instantiation")
	cmd.Flags().String(flagLabel, "default", "A human-readable name for this contract in lists")
	cmd.Flags().String(flagAdmin, "", "Address of an admin")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewExecuteContractCmd(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "execute [contract_addr_bech32] [json_encoded_send_args] --amount [coins,optional]",
		Short:   "Execute a command on a wasm contract",
		Aliases: []string{"run", "call", "exec", "ex", "e"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(m.GetCdc()))
			clientCtx := clientCtx.NewCLIContext().WithCodec(m.GetCdc()).WithInterfaceRegistry(reg)

			msg, err := parseExecuteArgs(args[0], args[1], sdk.AccToAWasmddress(clientCtx.GetFromAddress()), cmd.Flags())
			if err != nil {
				return err
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return utils.GenerateOrBroadcastMsgs(clientCtx, txBldr, []sdk.Msg{msg})
		},
	}

	cmd.Flags().String(flagAmount, "", "Coins to send to the contract along with command")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewMigrateContractCmd(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "migrate [contract_addr_bech32] [new_code_id_int64] [json_encoded_migration_args]",
		Short:   "Migrate a wasm contract to a new code version",
		Aliases: []string{"update", "mig", "m"},
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(m.GetCdc()))
			clientCtx := clientCtx.NewCLIContext().WithCodec(m.GetCdc()).WithInterfaceRegistry(reg)

			msg, err := parseMigrateContractArgs(args, clientCtx)
			if err != nil {
				return err
			}
			if err := msg.ValidateBasic(); err != nil {
				return nil
			}
			return utils.GenerateOrBroadcastMsgs(clientCtx, txBldr, []sdk.Msg{msg})
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewUpdateContractAdminCmd(m *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "set-contract-admin [contract_addr_bech32] [new_admin_addr_bech32]",
		Short:   "Set new admin for a contract",
		Aliases: []string{"new-admin", "admin", "set-adm", "sa"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(m.GetCdc()))
			clientCtx := clientCtx.NewCLIContext().WithCodec(m.GetCdc()).WithInterfaceRegistry(reg)

			msg, err := parseUpdateContractAdminArgs(args, clientCtx)
			if err != nil {
				return err
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return utils.GenerateOrBroadcastMsgs(clientCtx, txBldr, []sdk.Msg{msg})
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func parseStoreCodeArgs(file string, sender sdk.WasmAddress, flags *flag.FlagSet) (types.MsgStoreCode, error) {
	wasm, err := ioutil.ReadFile(file)
	if err != nil {
		return types.MsgStoreCode{}, err
	}

	// gzip the wasm file
	if ioutils.IsWasm(wasm) {
		wasm, err = ioutils.GzipIt(wasm)

		if err != nil {
			return types.MsgStoreCode{}, err
		}
	} else if !ioutils.IsGzip(wasm) {
		return types.MsgStoreCode{}, fmt.Errorf("invalid input file. Use wasm binary or gzip")
	}

	var perm *types.AccessConfig
	onlyAddrStr, err := flags.GetString(flagInstantiateByAddress)
	if err != nil {
		return types.MsgStoreCode{}, fmt.Errorf("instantiate by address: %s", err)
	}
	if onlyAddrStr != "" {
		allowedAddr, err := sdk.WasmAddressFromBech32(onlyAddrStr)
		if err != nil {
			return types.MsgStoreCode{}, sdkerrors.Wrap(err, flagInstantiateByAddress)
		}
		x := types.AccessTypeOnlyAddress.With(allowedAddr)
		perm = &x
	} else {
		everybodyStr, err := flags.GetString(flagInstantiateByEverybody)
		if err != nil {
			return types.MsgStoreCode{}, fmt.Errorf("instantiate by everybody: %s", err)
		}
		if everybodyStr != "" {
			ok, err := strconv.ParseBool(everybodyStr)
			if err != nil {
				return types.MsgStoreCode{}, fmt.Errorf("boolean value expected for instantiate by everybody: %s", err)
			}
			if ok {
				perm = &types.AllowEverybody
			}
		}
	}

	msg := types.MsgStoreCode{
		Sender:                sender.String(),
		WASMByteCode:          wasm,
		InstantiatePermission: perm,
	}
	return msg, nil
}

func parseInstantiateArgs(rawCodeID, initMsg string, sender sdk.WasmAddress, flags *flag.FlagSet) (types.MsgInstantiateContract, error) {
	// get the id of the code to instantiate
	codeID, err := strconv.ParseUint(rawCodeID, 10, 64)
	if err != nil {
		return types.MsgInstantiateContract{}, err
	}

	amountStr, err := flags.GetString(flagAmount)
	if err != nil {
		return types.MsgInstantiateContract{}, fmt.Errorf("amount: %s", err)
	}
	amount, err := sdk.ParseCoinsNormalized(amountStr)
	if err != nil {
		return types.MsgInstantiateContract{}, fmt.Errorf("amount: %s", err)
	}
	label, err := flags.GetString(flagLabel)
	if err != nil {
		return types.MsgInstantiateContract{}, fmt.Errorf("label: %s", err)
	}
	if label == "" {
		return types.MsgInstantiateContract{}, errors.New("label is required on all contracts")
	}
	adminStr, err := flags.GetString(flagAdmin)
	if err != nil {
		return types.MsgInstantiateContract{}, fmt.Errorf("admin: %s", err)
	}

	// build and sign the transaction, then broadcast to Tendermint
	msg := types.MsgInstantiateContract{
		Sender: sender.String(),
		CodeID: codeID,
		Label:  label,
		Funds:  sdk.CoinsToCoinAdapters(amount),
		Msg:    []byte(initMsg),
		Admin:  adminStr,
	}
	return msg, nil
}

func parseExecuteArgs(contractAddr string, execMsg string, sender sdk.WasmAddress, flags *flag.FlagSet) (types.MsgExecuteContract, error) {
	amountStr, err := flags.GetString(flagAmount)
	if err != nil {
		return types.MsgExecuteContract{}, fmt.Errorf("amount: %s", err)
	}

	amount, err := sdk.ParseCoinsNormalized(amountStr)
	if err != nil {
		return types.MsgExecuteContract{}, err
	}

	return types.MsgExecuteContract{
		Sender:   sender.String(),
		Contract: contractAddr,
		Funds:    sdk.CoinsToCoinAdapters(amount),
		Msg:      []byte(execMsg),
	}, nil
}

func parseMigrateContractArgs(args []string, cliCtx clientCtx.CLIContext) (types.MsgMigrateContract, error) {
	// get the id of the code to instantiate
	codeID, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return types.MsgMigrateContract{}, sdkerrors.Wrap(err, "code id")
	}

	migrateMsg := args[2]

	msg := types.MsgMigrateContract{
		Sender:   cliCtx.GetFromAddress().String(),
		Contract: args[0],
		CodeID:   codeID,
		Msg:      []byte(migrateMsg),
	}
	return msg, nil
}

func parseUpdateContractAdminArgs(args []string, cliCtx clientCtx.CLIContext) (types.MsgUpdateAdmin, error) {
	msg := types.MsgUpdateAdmin{
		Sender:   cliCtx.GetFromAddress().String(),
		Contract: args[0],
		NewAdmin: args[1],
	}
	return msg, nil
}
