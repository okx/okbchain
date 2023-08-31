package context

import (
	"fmt"
	"testing"

	"github.com/okx/brczero/libs/tendermint/mempool"
	"github.com/okx/brczero/libs/tendermint/rpc/client/mock"
	ctypes "github.com/okx/brczero/libs/tendermint/rpc/core/types"
	tmtypes "github.com/okx/brczero/libs/tendermint/types"
	"github.com/stretchr/testify/require"

	"github.com/okx/brczero/libs/cosmos-sdk/client/flags"
	sdkerrors "github.com/okx/brczero/libs/cosmos-sdk/types/errors"
)

type MockClient struct {
	mock.Client
	err error
}

func (c MockClient) BroadcastTxCommit(tx tmtypes.Tx) (*ctypes.ResultBroadcastTxCommit, error) {
	return nil, c.err
}

func (c MockClient) BroadcastTxAsync(tx tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
	return nil, c.err
}

func (c MockClient) BroadcastTxSync(tx tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
	return nil, c.err
}

func (c MockClient) BlockchainInfo(minHeight, maxHeight int64) (*ctypes.ResultBlockchainInfo, error) {
	return &ctypes.ResultBlockchainInfo{
		LastHeight: 0,
	}, nil
}

func (c MockClient) LatestBlockNumber() (int64, error) {
	return 0, nil
}

func CreateContextWithErrorAndMode(err error, mode string) CLIContext {
	return CLIContext{
		Client:        MockClient{err: err},
		BroadcastMode: mode,
	}
}

// Test the correct code is returned when
func TestBroadcastError(t *testing.T) {
	errors := map[error]uint32{
		mempool.ErrTxInCache:       sdkerrors.ErrTxInMempoolCache.ABCICode(),
		mempool.ErrTxTooLarge{}:    sdkerrors.ErrTxTooLarge.ABCICode(),
		mempool.ErrMempoolIsFull{}: sdkerrors.ErrMempoolIsFull.ABCICode(),
	}

	modes := []string{
		flags.BroadcastAsync,
		flags.BroadcastBlock,
		flags.BroadcastSync,
	}

	txBytes := []byte{0xA, 0xB}
	txHash := fmt.Sprintf("%X", tmtypes.Tx(txBytes).Hash())

	for _, mode := range modes {
		for err, code := range errors {
			ctx := CreateContextWithErrorAndMode(err, mode)
			resp, returnedErr := ctx.BroadcastTx(txBytes)
			require.NoError(t, returnedErr)
			require.Equal(t, code, resp.Code)
			require.Equal(t, txHash, resp.TxHash)
		}
	}

}
