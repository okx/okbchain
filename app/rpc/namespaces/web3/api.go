package web3

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/okx/brczero/app/rpc/monitor"
	"github.com/okx/brczero/libs/cosmos-sdk/version"
	"github.com/okx/brczero/libs/tendermint/libs/log"
	"github.com/spf13/viper"
)

const (
	NameSpace = "web3"
)

// PublicWeb3API is the web3_ prefixed set of APIs in the Web3 JSON-RPC spec.
type PublicWeb3API struct {
	logger  log.Logger
	Metrics *monitor.RpcMetrics
}

// NewAPI creates an instance of the Web3 API.
func NewAPI(log log.Logger) *PublicWeb3API {
	api := &PublicWeb3API{
		logger: log.With("module", "json-rpc", "namespace", NameSpace),
	}
	if viper.GetBool(monitor.FlagEnableMonitor) {
		api.Metrics = monitor.MakeMonitorMetrics(NameSpace)
	}
	return api
}

// ClientVersion returns the client version in the Web3 user agent format.
func (api *PublicWeb3API) ClientVersion() string {
	monitor := monitor.GetMonitor("web3_clientVersion", api.logger, api.Metrics).OnBegin()
	defer monitor.OnEnd()
	info := version.NewInfo()
	return fmt.Sprintf("%s-%s", info.Name, info.Version)
}

// Sha3 returns the keccak-256 hash of the passed-in input.
func (api *PublicWeb3API) Sha3(input hexutil.Bytes) hexutil.Bytes {
	monitor := monitor.GetMonitor("web3_sha3", api.logger, api.Metrics).OnBegin()
	defer monitor.OnEnd()
	return crypto.Keccak256(input)
}
