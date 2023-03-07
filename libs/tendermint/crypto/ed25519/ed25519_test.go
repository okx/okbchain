package ed25519_test

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/okx/okbchain/libs/tendermint/crypto"
	"github.com/okx/okbchain/libs/tendermint/crypto/ed25519"
)

func TestSignAndValidateEd25519(t *testing.T) {

	privKey := ed25519.GenPrivKey()
	pubKey := privKey.PubKey()

	msg := crypto.CRandBytes(128)
	sig, err := privKey.Sign(msg)
	require.Nil(t, err)

	// Test the signature
	assert.True(t, pubKey.VerifyBytes(msg, sig))

	// Mutate the signature, just one bit.
	// TODO: Replace this with a much better fuzzer, tendermint/ed25519/issues/10
	sig[7] ^= byte(0x01)

	assert.False(t, pubKey.VerifyBytes(msg, sig))

	bytes := pubKey.Bytes()
	var recoverPubKey ed25519.PubKeyEd25519
	recoverPubKey.UnmarshalFromAmino(bytes)
	assert.Equal(t, bytes, recoverPubKey.Bytes())
}

func genPubkey(hex string) ed25519.PubKeyEd25519 {
	bytes := hexutil.MustDecode(hex)
	var recoverPubKey ed25519.PubKeyEd25519
	recoverPubKey.UnmarshalFromAmino(bytes)
	return recoverPubKey
}
