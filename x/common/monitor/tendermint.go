package monitor

import (
	"fmt"
	"github.com/okx/okbchain/libs/cosmos-sdk/server"
	tmcli "github.com/okx/okbchain/libs/tendermint/rpc/client"
	tmhttp "github.com/okx/okbchain/libs/tendermint/rpc/client/http"
	"github.com/spf13/viper"
	"strings"
	"sync"
)

const (
	loopBackAddr = "tcp://127.0.0.1"
)

var (
	tmMonitor     *TendermintMonitor
	initTmMonitor sync.Once
)

// GetTendermintMonitor gets the global instance of TendermintMonitor
func GetTendermintMonitor() *TendermintMonitor {
	initTmMonitor.Do(func() {
		tmMonitor = NewTendermintMonitor(viper.GetString(server.FlagLocalRpcPort))
	})

	return tmMonitor
}

// TendermintMonitor - structure of monitor for block/mempool monitoring
type TendermintMonitor struct {
	enable    bool
	rpcClient tmcli.Client
	status    tendermintStatus
}

// NewTendermintMonitor creates a new instance of TendermintMonitor
func NewTendermintMonitor(portInput string) *TendermintMonitor {
	if len(portInput) == 0 {
		// disable the tendermint monitor
		return &TendermintMonitor{
			enable: false,
		}
	}

	rpcCli, err := tmhttp.New(fmt.Sprintf("%s:%d", loopBackAddr, ParsePort(portInput)), "/websocket")
	if err != nil {
		panic(fmt.Sprintf("fail to init a rpc client in tendermint monitor: %s", err.Error()))
	}

	return &TendermintMonitor{
		enable:    true,
		rpcClient: rpcCli,
	}
}

// reset resets the status of TendermintMonitor
func (tm *TendermintMonitor) reset() {
	tm.status.blockSize = -1
	tm.status.uncomfirmedTxNum = -1
	tm.status.uncormfirmedTxTotalSize = -1
}

// Run starts monitoring
func (tm *TendermintMonitor) Run(height int64) error {
	// TendermintMonitor disabled
	if !tm.enable {
		return nil
	}

	tm.reset()
	err := tm.getTendermintStatus(height)
	if err != nil {
		return err
	}

	return nil
}

// GetResultString gets the format string result
func (tm *TendermintMonitor) GetResultString() string {
	// TendermintMonitor disabled
	if !tm.enable {
		return ""
	}

	return fmt.Sprintf(",BlockSize<%.2fKB>, MemPoolTx<%d>, MemPoolSize<%.2fKB>,",
		float64(tm.status.blockSize)/1024,
		tm.status.uncomfirmedTxNum,
		float64(tm.status.uncormfirmedTxTotalSize)/1024)
}

type tendermintStatus struct {
	blockSize               int
	uncomfirmedTxNum        int
	uncormfirmedTxTotalSize int64
}

func (tm *TendermintMonitor) getTendermintStatus(height int64) error {
	block, err := tm.rpcClient.Block(&height)
	if err != nil {
		return fmt.Errorf("failed to query block on height %d", height)
	}

	uncomfirmedRes, err := tm.rpcClient.NumUnconfirmedTxs()
	if err != nil {
		return fmt.Errorf("failed to query mempool result on height %d", height)
	}

	// update status
	tm.status.blockSize = block.Block.Size()
	tm.status.uncomfirmedTxNum = uncomfirmedRes.Total
	tm.status.uncormfirmedTxTotalSize = uncomfirmedRes.TotalBytes

	return nil
}

// CombineMonitorsRes combines all the monitors' results
func CombineMonitorsRes(res ...string) string {
	var builder strings.Builder
	for _, r := range res {
		builder.WriteString(r)
	}

	return strings.Trim(strings.TrimSpace(builder.String()), ",")
}
