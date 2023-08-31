// nolint noalias
package types

import (
	"github.com/okx/brczero/libs/tendermint/crypto"
	"github.com/okx/brczero/libs/tendermint/crypto/secp256k1"

	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
)

// NewTestMsg generates a test message
func NewTestMsg(addrs ...sdk.AccAddress) *sdk.TestMsg {
	return sdk.NewTestMsg(addrs...)
}

// KeyTestPubAddr generates a test key pair
func KeyTestPubAddr() (crypto.PrivKey, crypto.PubKey, sdk.AccAddress) {
	key := secp256k1.GenPrivKey()
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, pub, addr
}
