package eth

import (
	"testing"

	evmtypes "github.com/okx/okbchain/x/evm/types"

	"github.com/stretchr/testify/require"
)

func Test_TransformDataError(t *testing.T) {

	sdkerr := newWrappedCosmosError(7, `["execution reverted","message","HexData","0x00000000000"];failed message tail`, evmtypes.ModuleName)
	err := TransformDataError(sdkerr, "eth_estimateGas").(DataError)
	require.NotNil(t, err.ErrorData())
	require.Equal(t, err.ErrorData(), "0x00000000000")
	require.Equal(t, err.ErrorCode(), VMExecuteExceptionInEstimate)
	err = TransformDataError(sdkerr, "eth_call").(DataError)
	require.NotNil(t, err.ErrorData())
	data, ok := err.ErrorData().(*wrappedEthError)
	require.True(t, ok)
	require.NotNil(t, data)
}
