package client

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/okx/okbchain/app/crypto/ethkeystore"
	"github.com/okx/okbchain/app/crypto/hd"
	"github.com/okx/okbchain/libs/cosmos-sdk/client/flags"
	"github.com/okx/okbchain/libs/cosmos-sdk/client/input"
	"github.com/okx/okbchain/libs/cosmos-sdk/crypto/keys"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
)

// UnsafeExportEthKeyCommand exports a key with the given name as a private key in hex format.
func UnsafeExportEthKeyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unsafe-export-eth-key [name]",
		Short: "**UNSAFE** Export an Ethereum private key",
		Long:  `**UNSAFE** Export an Ethereum private key unencrypted to use in dev tooling`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inBuf := bufio.NewReader(cmd.InOrStdin())

			kb, err := keys.NewKeyring(
				sdk.KeyringServiceName(),
				viper.GetString(flags.FlagKeyringBackend),
				viper.GetString(flags.FlagHome),
				inBuf,
				hd.EthSecp256k1Options()...,
			)
			if err != nil {
				return err
			}

			decryptPassword := ""
			conf := true
			keyringBackend := viper.GetString(flags.FlagKeyringBackend)
			switch keyringBackend {
			case keys.BackendFile:
				decryptPassword, err = input.GetPassword(
					"**WARNING this is an unsafe way to export your unencrypted private key**\nEnter key password:",
					inBuf)
			case keys.BackendOS:
				conf, err = input.GetConfirmation(
					"**WARNING** this is an unsafe way to export your unencrypted private key, are you sure?",
					inBuf)
			}
			if err != nil || !conf {
				return err
			}

			// Exports private key from keybase using password
			privKey, err := kb.ExportPrivateKeyObject(args[0], decryptPassword)
			if err != nil {
				return err
			}

			// Converts tendermint  key to ethereum key
			ethKey, err := ethkeystore.EncodeTmKeyToEthKey(privKey)
			if err != nil {
				return fmt.Errorf("invalid private key type, must be Ethereum key: %T", privKey)
			}

			// Formats key for output
			privB := ethcrypto.FromECDSA(ethKey)
			keyS := strings.ToLower(hexutil.Encode(privB)[2:])

			fmt.Println(keyS)

			return nil
		},
	}
}

// ExportEthCompCommand exports a key with the given name as a keystore file.
func ExportEthCompCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "export-eth-comp [name] [dir]",
		Short:  "Export an Ethereum private keystore directory (This command has been deprecated,please use 'okbchaincli keys export-eth-keystore')",
		Hidden: true,
		Long: `Export an Ethereum private keystore file encrypted to use in eth client import.

	The parameters of scrypt encryption algorithm is StandardScryptN and StandardScryptN`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("This command has been deprecated,please use 'okbchaincli keys export-eth-keystore'")
		},
	}
}

// ExportEthKeyStoreCommand exports a key with the given name as a keystore file.
func ExportEthKeyStoreCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "export-eth-keystore [name] [dir]",
		Short: "Export an Ethereum private keystore directory",
		Long: `Export an Ethereum private keystore file encrypted to use in eth client import.

	The parameters of scrypt encryption algorithm is StandardScryptN and StandardScryptN`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			inBuf := bufio.NewReader(cmd.InOrStdin())
			accountName := args[0]
			dir := args[1]

			kb, err := keys.NewKeyring(
				sdk.KeyringServiceName(),
				viper.GetString(flags.FlagKeyringBackend),
				viper.GetString(flags.FlagHome),
				inBuf,
				hd.EthSecp256k1Options()...,
			)
			if err != nil {
				return err
			}

			decryptPassword := ""
			conf := true
			keyringBackend := viper.GetString(flags.FlagKeyringBackend)
			switch keyringBackend {
			case keys.BackendFile:
				decryptPassword, err = input.GetPassword(
					"Enter passphrase to decrypt your key:",
					inBuf)
			case keys.BackendOS:
				conf, err = input.GetConfirmation(
					"Decrypt your key by os passphrase. Are you sure?",
					inBuf)
			}
			if err != nil || !conf {
				return err
			}

			// Get keystore password
			encryptPassword, err := input.GetPassword("Enter passphrase to encrypt the exported keystore file:", inBuf)
			if err != nil {
				return err
			}

			// exports private key from keybase using password
			privKey, err := kb.ExportPrivateKeyObject(accountName, decryptPassword)
			if err != nil {
				return err
			}

			// Exports private key from keybase using password
			fileName, err := ethkeystore.CreateKeystoreByTmKey(privKey, dir, encryptPassword)
			if err != nil {
				return err
			}

			fmt.Printf("The keystore has exported to: %s \n", fileName)
			return nil
		},
	}
}

// ImportEthKeyStoreCommand import keystore file.
func ImportEthKeyStoreCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-eth-keystore [name] [file]",
		Short: "Import an Ethereum private keystore",
		Long:  `Import an Ethereum private keystore file.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			algo, err := cmd.Flags().GetString(flagKeyAlgo)
			if err != nil {
				return err
			}
			inBuf := bufio.NewReader(cmd.InOrStdin())
			accountName := args[0]
			file := args[1]

			// Get keystore password
			decryptPassword, err := input.GetPassword("Enter passphrase to decrypt keystore file:", inBuf)
			if err != nil {
				return err
			}

			privkeyArmor, err := ethkeystore.ImportKeyStoreFile(decryptPassword, decryptPassword, file, keys.SigningAlgo(algo))
			if err != nil {
				return err
			}
			buf := bufio.NewReader(cmd.InOrStdin())
			kb, err := keys.NewKeyring(sdk.KeyringServiceName(), viper.GetString(flags.FlagKeyringBackend), viper.GetString(flags.FlagHome), buf)
			if err != nil {
				return err
			}

			return kb.ImportPrivKey(accountName, privkeyArmor, decryptPassword)
		},
	}
	cmd.Flags().String(flagKeyAlgo, string(hd.EthSecp256k1), "Key signing algorithm to generate keys for")
	return cmd
}
