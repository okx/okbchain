package net

import (
	"fmt"

	"github.com/okx/brczero/app/rpc/monitor"
	ethermint "github.com/okx/brczero/app/types"
	"github.com/okx/brczero/libs/cosmos-sdk/client/context"
	"github.com/okx/brczero/libs/tendermint/libs/log"
	rpcclient "github.com/okx/brczero/libs/tendermint/rpc/client"
	"github.com/spf13/viper"
)

const (
	NameSpace = "net"
)

// PublicNetAPI is the eth_ prefixed set of APIs in the Web3 JSON-RPC spec.
type PublicNetAPI struct {
	networkVersion uint64
	logger         log.Logger
	Metrics        *monitor.RpcMetrics
	tmClient       rpcclient.Client
}

// NewAPI creates an instance of the public Net Web3 API.
func NewAPI(clientCtx context.CLIContext, log log.Logger) *PublicNetAPI {
	// parse the chainID from a integer string
	chainIDEpoch, err := ethermint.ParseChainID(clientCtx.ChainID)
	if err != nil {
		panic(err)
	}

	api := &PublicNetAPI{
		networkVersion: chainIDEpoch.Uint64(),
		logger:         log.With("module", "json-rpc", "namespace", NameSpace),
		tmClient:       clientCtx.Client,
	}
	if viper.GetBool(monitor.FlagEnableMonitor) {
		api.Metrics = monitor.MakeMonitorMetrics(NameSpace)
	}
	return api
}

// Version returns the current ethereum protocol version.
func (api *PublicNetAPI) Version() string {
	monitor := monitor.GetMonitor("net_version", api.logger, api.Metrics).OnBegin()
	defer monitor.OnEnd()
	return fmt.Sprintf("%d", api.networkVersion)
}

// Listening returns if client is actively listening for network connections.
func (api *PublicNetAPI) Listening() bool {
	monitor := monitor.GetMonitor("net_listening", api.logger, api.Metrics).OnBegin()
	defer monitor.OnEnd()
	netInfo, err := api.tmClient.NetInfo()
	if err != nil {
		return false
	}
	return netInfo.Listening
}

// PeerCount returns the number of peers currently connected to the client.
func (api *PublicNetAPI) PeerCount() int {
	monitor := monitor.GetMonitor("net_peerCount", api.logger, api.Metrics).OnBegin()
	defer monitor.OnEnd()
	netInfo, err := api.tmClient.NetInfo()
	if err != nil {
		return 0
	}
	return len(netInfo.Peers)
}
