package keys

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/okx/okbchain/libs/tendermint/crypto"
	"github.com/okx/okbchain/libs/tendermint/crypto/multisig"
	"github.com/okx/okbchain/libs/tendermint/crypto/secp256k1"
	"github.com/stretchr/testify/require"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
)

func TestBech32KeysOutput(t *testing.T) {
	tmpKey := secp256k1.GenPrivKey().PubKey()
	bechTmpKey := sdk.MustBech32ifyPubKey(sdk.Bech32PubKeyTypeAccPub, tmpKey)
	tmpAddr := sdk.AccAddress(tmpKey.Address().Bytes())

	multisigPks := multisig.NewPubKeyMultisigThreshold(1, []crypto.PubKey{tmpKey})
	multiInfo := NewMultiInfo("multisig", multisigPks)
	accAddr := sdk.AccAddress(multiInfo.GetPubKey().Address().Bytes())
	bechPubKey := sdk.MustBech32ifyPubKey(sdk.Bech32PubKeyTypeAccPub, multiInfo.GetPubKey())

	expectedOutput := NewKeyOutput(multiInfo.GetName(), multiInfo.GetType().String(), accAddr.String(), bechPubKey)
	expectedOutput.EthAddress = common.BytesToAddress(accAddr).Hex()
	expectedOutput.OperAddress = sdk.ValAddress(accAddr).String()
	expectedOutput.Threshold = 1
	expectedOutput.PubKeys = []multisigPubKeyOutput{{tmpAddr.String(), bechTmpKey, 1}}

	outputs, err := Bech32KeysOutput([]Info{multiInfo})
	require.NoError(t, err)
	require.Equal(t, expectedOutput, outputs[0])
}
