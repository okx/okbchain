package coregrpc_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/okx/okbchain/libs/tendermint/abci/example/kvstore"
	cfg "github.com/okx/okbchain/libs/tendermint/config"
	core_grpc "github.com/okx/okbchain/libs/tendermint/rpc/grpc"
	rpctest "github.com/okx/okbchain/libs/tendermint/rpc/test"
)

func TestMain(m *testing.M) {
	// start a tendermint node in the background to test against
	app := kvstore.NewApplication()
	node := rpctest.StartTendermint(app)

	code := m.Run()

	// and shut down proper at the end
	rpctest.StopTendermint(node)
	os.Exit(code)
}

func TestBroadcastTx(t *testing.T) {
	setMocConfig(100)
	res, err := rpctest.GetGRPCClient().BroadcastTx(
		context.Background(),
		&core_grpc.RequestBroadcastTx{Tx: []byte("this is a tx")},
	)
	require.NoError(t, err)
	require.EqualValues(t, 0, res.CheckTx.Code)
	require.EqualValues(t, 0, res.DeliverTx.Code)
}

func setMocConfig(clientNum int) {
	moc := cfg.MockDynamicConfig{}
	moc.SetMaxSubscriptionClients(100)

	cfg.SetDynamicConfig(moc)
}
