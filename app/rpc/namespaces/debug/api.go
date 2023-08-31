package debug

import (
	"encoding/json"
	"fmt"

	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	"github.com/spf13/viper"

	"github.com/ethereum/go-ethereum/common"
	clientcontext "github.com/okx/brczero/libs/cosmos-sdk/client/context"

	"github.com/okx/brczero/app/rpc/backend"
	"github.com/okx/brczero/app/rpc/monitor"
	"github.com/okx/brczero/libs/tendermint/libs/log"

	evmtypes "github.com/okx/brczero/x/evm/types"
)

const (
	NameSpace = "debug"
)

// PublicTxPoolAPI offers and API for the transaction pool. It only operates on data that is non confidential.
type PublicDebugAPI struct {
	clientCtx clientcontext.CLIContext
	logger    log.Logger
	backend   backend.Backend
	Metrics   *monitor.RpcMetrics
}

// NewPublicTxPoolAPI creates a new tx pool service that gives information about the transaction pool.
func NewAPI(clientCtx clientcontext.CLIContext, log log.Logger, backend backend.Backend) *PublicDebugAPI {
	api := &PublicDebugAPI{
		clientCtx: clientCtx,
		backend:   backend,
		logger:    log.With("module", "json-rpc", "namespace", "debug"),
	}
	if viper.GetBool(monitor.FlagEnableMonitor) {
		api.Metrics = monitor.MakeMonitorMetrics(NameSpace)
	}
	return api
}

// TraceTransaction returns the structured logs created during the execution of EVM
// and returns them as a JSON object.
func (api *PublicDebugAPI) TraceTransaction(txHash common.Hash, config evmtypes.TraceConfig) (interface{}, error) {
	monitor := monitor.GetMonitor("debug_traceTransaction", api.logger, api.Metrics).OnBegin()
	defer monitor.OnEnd()
	err := evmtypes.TestTracerConfig(&config)
	if err != nil {
		return nil, fmt.Errorf("tracer err : %s", err.Error())
	}
	configBytes, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	queryParam := sdk.QueryTraceTx{
		TxHash:      txHash,
		ConfigBytes: configBytes,
	}
	queryBytes, err := json.Marshal(&queryParam)
	if err != nil {
		return nil, err
	}
	_, err = api.clientCtx.Client.Tx(txHash.Bytes(), false)
	if err != nil {
		return nil, err
	}
	resTrace, _, err := api.clientCtx.QueryWithData("app/trace", queryBytes)
	if err != nil {
		return nil, err
	}

	var res sdk.Result
	if err := api.clientCtx.Codec.UnmarshalBinaryBare(resTrace, &res); err != nil {
		return nil, err
	}
	var decodedResult interface{}
	if err := json.Unmarshal(res.Data, &decodedResult); err != nil {
		return nil, err
	}

	return decodedResult, nil
}
