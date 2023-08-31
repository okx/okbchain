// nolint
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/okx/brczero/libs/cosmos-sdk/client"
	"github.com/okx/brczero/libs/cosmos-sdk/client/context"
	"github.com/okx/brczero/libs/cosmos-sdk/client/flags"
	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	interfacetypes "github.com/okx/brczero/libs/cosmos-sdk/codec/types"
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	"github.com/okx/brczero/libs/cosmos-sdk/version"
	"github.com/okx/brczero/libs/cosmos-sdk/x/auth"
	"github.com/okx/brczero/libs/cosmos-sdk/x/auth/client/utils"
	"github.com/okx/brczero/x/distribution/client/common"
	"github.com/okx/brczero/x/distribution/types"
	"github.com/okx/brczero/x/gov"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd(storeKey string, cdc *codec.Codec) *cobra.Command {
	distTxCmd := &cobra.Command{
		Use:                        types.ShortUseByCli,
		Short:                      "Distribution transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	distTxCmd.AddCommand(flags.PostCommands(
		GetCmdWithdrawRewards(cdc),
		GetCmdSetWithdrawAddr(cdc),
		GetCmdWithdrawAllRewards(cdc, storeKey),
	)...)

	return distTxCmd
}

// command to replace a delegator's withdrawal address
func GetCmdSetWithdrawAddr(cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "set-withdraw-addr [withdraw-addr]",
		Short: "change the default withdraw address for rewards associated with an address",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Set the withdraw address for rewards associated with a delegator address.

Example:
$ %s tx distr set-withdraw-addr ex1cftp8q8g4aa65nw9s5trwexe77d9t6cr8ndu02 --from mykey
`,
				version.ClientName,
			),
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(cdc))
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			delAddr := cliCtx.GetFromAddress()
			withdrawAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return fmt.Errorf("invalid address：%s", args[0])
			}

			msg := types.NewMsgSetWithdrawAddress(delAddr, withdrawAddr)
			return utils.GenerateOrBroadcastMsgs(cliCtx, txBldr, []sdk.Msg{msg})
		},
	}
}

// GetCmdWithdrawRewards command to withdraw rewards
func GetCmdWithdrawRewards(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw-rewards [validator-addr]",
		Short: "withdraw rewards from a given delegation address, and optionally withdraw validator commission if the delegation address given is a validator operator",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Withdraw rewards from a given delegation address, 
and optionally withdraw validator commission if the delegation address given is a validator operator

Example:
$ %s tx distr withdraw-rewards exvaloper1alq9na49n9yycysh889rl90g9nhe58lcqkfpfg --from mykey 
$ %s tx distr withdraw-rewards exvaloper1alq9na49n9yycysh889rl90g9nhe58lcqkfpfg --from mykey --commission

If this command is used without "--commission", and the address you want to withdraw rewards is both validator and delegator, 
only the delegator's rewards can be withdrew. However, if the address you want to withdraw rewards is only the validator, 
the validator commissions will be withdrew.
Example:
$ %s tx distr withdraw-rewards exvaloper1alq9na49n9yycysh889rl90g9nhe58lcqkfpfg --from mykey(validator)			# withdraw mykey's commission only
$ %s tx distr withdraw-rewards exvaloper1alq9na49n9yycysh889rl90g9nhe58lcqkfpfg --from mykey(validator&delegator)	# withdraw mykey's reward only
`,
				version.ClientName, version.ClientName, version.ClientName, version.ClientName,
			),
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(cdc))
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			delAddr := cliCtx.GetFromAddress()
			valAddr, err := sdk.ValAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			isVal := common.IsValidator(cliCtx, cdc, sdk.ValAddress(delAddr))
			isDel := common.IsDelegator(cliCtx, cdc, delAddr)
			needWarning := false

			msgs := []sdk.Msg{}
			if viper.GetBool(flagCommission) || (isVal && !isDel) {
				msgs = append(msgs, types.NewMsgWithdrawValidatorCommission(valAddr))
			} else {
				msgs = append(msgs, types.NewMsgWithdrawDelegatorReward(delAddr, valAddr))
				if isVal && isDel {
					needWarning = true
				}
			}
			result := transWrapError(common.NewWrapError(utils.GenerateOrBroadcastMsgs(cliCtx, txBldr, msgs)), delAddr, valAddr)
			if needWarning {
				fmt.Fprintf(os.Stdout, "%s\n", fmt.Sprintf("\nFound address: \"%s\" is a validator, please add the `--commission` flag to the command line if you want to withdraw the commission, for example:\n"+
					"%s ..... --commission.\n",
					delAddr.String(), version.ClientName))
			}
			return result
		},
	}
	cmd.Flags().Bool(flagCommission, false, "withdraw validator's commission")
	return cmd
}

func transWrapError(wrapErr *common.WrapError, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	if wrapErr == nil {
		return nil
	}

	wrapErr.Trans(types.CodeEmptyDelegationDistInfo, fmt.Sprintf("found account %s is not a delegator, please check it first", delAddr.String()))
	wrapErr.Trans(types.CodeNoValidatorCommission, fmt.Sprintf("found account %s is not a validator, please check it first", delAddr.String()))
	wrapErr.Trans(types.CodeEmptyValidatorDistInfo, fmt.Sprintf("found validator address %s is not a validator, please check it first", valAddr.String()))
	wrapErr.Trans(types.CodeEmptyDelegationVoteValidator, fmt.Sprintf("found validator address %s haven't been voted, please check it first", valAddr.String()))
	if wrapErr.Changed {
		return wrapErr
	}

	return wrapErr.RawError
}

// GetCmdCommunityPoolSpendProposal implements the command to submit a community-pool-spend proposal
func GetCmdCommunityPoolSpendProposal(cdcP *codec.CodecProxy, reg interfacetypes.InterfaceRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "community-pool-spend [proposal-file]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a community pool spend proposal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit a community pool spend proposal along with an initial deposit.
The proposal details must be supplied via a JSON file.

Example:
$ %s tx gov submit-proposal community-pool-spend <path/to/proposal.json> --from=<key_or_address>

Where proposal.json contains:

{
  "title": "Community Pool Spend",
  "description": "Pay me some %s!",
  "recipient": "ex1cftp8q8g4aa65nw9s5trwexe77d9t6cr8ndu02",
  "amount": [
    {
      "denom": "%s",
      "amount": "10000"
    }
  ],
  "deposit": [
    {
      "denom": "%s",
      "amount": "10000"
    }
  ]
}
`,
				version.ClientName, sdk.DefaultBondDenom, sdk.DefaultBondDenom, sdk.DefaultBondDenom,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			cdc := cdcP.GetCdc()
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(cdc))
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			proposal, err := ParseCommunityPoolSpendProposalJSON(cdc, args[0])
			if err != nil {
				return err
			}

			from := cliCtx.GetFromAddress()
			content := types.NewCommunityPoolSpendProposal(proposal.Title, proposal.Description, proposal.Recipient, proposal.Amount)

			msg := gov.NewMsgSubmitProposal(content, proposal.Deposit, from)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return utils.GenerateOrBroadcastMsgs(cliCtx, txBldr, []sdk.Msg{msg})
		},
	}

	return cmd
}
