package core

import (
	rpc "github.com/okx/okbchain/libs/tendermint/rpc/jsonrpc/server"
)

// TODO: better system than "unsafe" prefix
// NOTE: Amino is registered in rpc/core/types/codec.go.

var Routes = map[string]*rpc.RPCFunc{
	// subscribe/unsubscribe are reserved for websocket events.
	"subscribe":       rpc.NewWSRPCFunc(Subscribe, "query"),
	"unsubscribe":     rpc.NewWSRPCFunc(Unsubscribe, "query"),
	"unsubscribe_all": rpc.NewWSRPCFunc(UnsubscribeAll, ""),

	// info API
	"health":                   rpc.NewRPCFunc(Health, ""),
	"status":                   rpc.NewRPCFunc(Status, ""),
	"net_info":                 rpc.NewRPCFunc(NetInfo, ""),
	"blockchain":               rpc.NewRPCFunc(BlockchainInfo, "minHeight,maxHeight"),
	"genesis":                  rpc.NewRPCFunc(Genesis, ""),
	"block":                    rpc.NewRPCFunc(CM40Block, "height"),
	"cm39_block":               rpc.NewRPCFunc(Block, "height"),
	"block_by_hash":            rpc.NewRPCFunc(BlockByHash, "hash"),
	"block_info":               rpc.NewRPCFunc(BlockInfo, "height"),
	"block_results":            rpc.NewRPCFunc(BlockResults, "height"),
	"commit":                   rpc.NewRPCFunc(CommitIBC, "height"),
	"tx":                       rpc.NewRPCFunc(Tx, "hash,prove"),
	"tx_search":                rpc.NewRPCFunc(TxSearch, "query,prove,page,per_page,order_by"),
	"validators":               rpc.NewRPCFunc(Validators, "height,page,per_page"),
	"dump_consensus_state":     rpc.NewRPCFunc(DumpConsensusState, ""),
	"consensus_state":          rpc.NewRPCFunc(ConsensusState, ""),
	"consensus_params":         rpc.NewRPCFunc(ConsensusParams, "height"),
	"unconfirmed_txs":          rpc.NewRPCFunc(UnconfirmedTxs, "limit"),
	"num_unconfirmed_txs":      rpc.NewRPCFunc(NumUnconfirmedTxs, ""),
	"user_unconfirmed_txs":     rpc.NewRPCFunc(UserUnconfirmedTxs, "address,limit"),
	"user_num_unconfirmed_txs": rpc.NewRPCFunc(UserNumUnconfirmedTxs, "address"),
	"get_address_list":         rpc.NewRPCFunc(GetAddressList, ""),
	"block_search":             rpc.NewRPCFunc(BlockSearch, "query,page,per_page,order_by"),

	// tx broadcast API
	"broadcast_tx_commit": rpc.NewRPCFunc(BroadcastTxCommit, "tx"),
	"broadcast_tx_sync":   rpc.NewRPCFunc(BroadcastTxSync, "tx"),
	"broadcast_tx_async":  rpc.NewRPCFunc(BroadcastTxAsync, "tx"),

	// abci API
	"abci_query": rpc.NewRPCFunc(ABCIQuery, "path,data,height,prove"),
	"abci_info":  rpc.NewRPCFunc(ABCIInfo, ""),

	// evidence API
	"broadcast_evidence": rpc.NewRPCFunc(BroadcastEvidence, "evidence"),

	"tx_simulate_gas": rpc.NewRPCFunc(TxSimulateGasCost, "hash"),

	"get_enable_delete_min_gp_tx": rpc.NewRPCFunc(GetEnableDeleteMinGPTx, ""),
}

func AddUnsafeRoutes() {
	// control API
	Routes["dial_seeds"] = rpc.NewRPCFunc(UnsafeDialSeeds, "seeds")
	Routes["dial_peers"] = rpc.NewRPCFunc(UnsafeDialPeers, "peers,persistent")
	Routes["unsafe_flush_mempool"] = rpc.NewRPCFunc(UnsafeFlushMempool, "")

	// profiler API
	Routes["unsafe_start_cpu_profiler"] = rpc.NewRPCFunc(UnsafeStartCPUProfiler, "filename")
	Routes["unsafe_stop_cpu_profiler"] = rpc.NewRPCFunc(UnsafeStopCPUProfiler, "")
	Routes["unsafe_write_heap_profile"] = rpc.NewRPCFunc(UnsafeWriteHeapProfile, "filename")
}
