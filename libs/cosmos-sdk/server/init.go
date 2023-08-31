package server

import (
	"fmt"

	"github.com/okx/brczero/libs/cosmos-sdk/crypto/keys"

	clkeys "github.com/okx/brczero/libs/cosmos-sdk/client/keys"
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
)

// GenerateCoinKey returns the address of a public key, along with the secret
// phrase to recover the private key.
func GenerateCoinKey() (sdk.AccAddress, string, error) {

	// generate a private key, with recovery phrase
	info, secret, err := clkeys.NewInMemoryKeyBase().CreateMnemonic(
		"name", keys.English, "pass", keys.Secp256k1, "")
	if err != nil {
		return sdk.AccAddress([]byte{}), "", err
	}
	addr := info.GetPubKey().Address()
	return sdk.AccAddress(addr), secret, nil
}

// GenerateSaveCoinKey returns the address of a public key, along with the secret
// phrase to recover the private key.
func GenerateSaveCoinKey(keybase keys.Keybase, keyName, keyPass string, overwrite bool, mnemonicInput string) (sdk.AccAddress, string, error) {
	// ensure no overwrite
	if !overwrite {
		_, err := keybase.Get(keyName)
		if err == nil {
			return sdk.AccAddress([]byte{}), "", fmt.Errorf(
				"key already exists, overwrite is disabled")
		}
	}

	// generate a private key, with recovery phrase
	info, secret, err := keybase.CreateMnemonic(keyName, keys.English, keyPass, keys.Secp256k1, mnemonicInput)
	if err != nil {
		return sdk.AccAddress([]byte{}), "", err
	}

	return sdk.AccAddress(info.GetPubKey().Address()), secret, nil
}
