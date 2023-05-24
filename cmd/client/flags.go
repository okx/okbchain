package client

import (
	"github.com/spf13/cobra"

	"github.com/okx/okbchain/app"
	"github.com/okx/okbchain/app/config"
	"github.com/okx/okbchain/app/rpc"
	"github.com/okx/okbchain/app/rpc/backend"
	"github.com/okx/okbchain/app/rpc/monitor"
	"github.com/okx/okbchain/app/rpc/namespaces/eth"
	"github.com/okx/okbchain/app/rpc/namespaces/eth/filters"
	"github.com/okx/okbchain/app/rpc/websockets"
	"github.com/okx/okbchain/app/types"
	"github.com/okx/okbchain/app/utils/sanity"
	"github.com/okx/okbchain/libs/system/trace"
	"github.com/okx/okbchain/libs/tendermint/consensus"
	"github.com/okx/okbchain/libs/tendermint/libs/automation"
	tmtypes "github.com/okx/okbchain/libs/tendermint/types"
	tmdb "github.com/okx/okbchain/libs/tm-db"
	evmtypes "github.com/okx/okbchain/x/evm/types"
	"github.com/okx/okbchain/x/evm/watcher"
	"github.com/okx/okbchain/x/infura"
	"github.com/okx/okbchain/x/token"
	"github.com/okx/okbchain/x/wasm"
)

func RegisterAppFlag(cmd *cobra.Command) {
	cmd.Flags().Bool(watcher.FlagFastQuery, false, "Enable the fast query mode for rpc queries")
	cmd.Flags().Bool(watcher.FlagFastQueryForWasm, false, "Enable the fast query mode for wasm tx")
	cmd.Flags().Uint64(eth.FlagFastQueryThreshold, 10, "Set the threshold of fast query")
	cmd.Flags().String(eth.FlagE2cWasmMsgHelperAddr, "", "Set the e2c wasm msg helper contract address")
	cmd.Flags().Int(watcher.FlagFastQueryLru, 1000, "Set the size of LRU cache under fast-query mode")
	cmd.Flags().Int(backend.FlagApiBackendBlockLruCache, 30000, "Set the size of block LRU cache for backend mem cache")
	cmd.Flags().Int(backend.FlagApiBackendTxLruCache, 100000, "Set the size of tx LRU cache for backend mem cache")
	cmd.Flags().Bool(watcher.FlagCheckWd, false, "Enable check watchDB in log")
	cmd.Flags().Bool(rpc.FlagPersonalAPI, true, "Enable the personal_ prefixed set of APIs in the Web3 JSON-RPC spec")
	cmd.Flags().Bool(rpc.FlagDebugAPI, false, "Enable the debug_ prefixed set of APIs in the Web3 JSON-RPC spec")
	cmd.Flags().Bool(evmtypes.FlagEnableBloomFilter, true, "Enable bloom filter for event logs")
	cmd.Flags().Int64(filters.FlagGetLogsHeightSpan, 2000, "config the block height span for get logs")
	// register application rpc to nacos
	cmd.Flags().String(rpc.FlagRestApplicationName, "", "rest application name in  nacos")
	cmd.Flags().MarkHidden(rpc.FlagRestApplicationName)
	cmd.Flags().String(rpc.FlagRestNacosUrls, "", "nacos server urls for discovery service of rest api")
	cmd.Flags().MarkHidden(rpc.FlagRestNacosUrls)
	cmd.Flags().String(rpc.FlagRestNacosNamespaceId, "", "nacos namepace id for discovery service of rest api")
	cmd.Flags().MarkHidden(rpc.FlagRestNacosNamespaceId)
	cmd.Flags().String(rpc.FlagExternalListenAddr, "127.0.0.1:26659", "Set the rest-server external ip and port, when it is launched by Docker")
	// register tendermint rpc to nacos
	cmd.Flags().String(rpc.FlagNacosTmrpcUrls, "", "nacos server urls for discovery service of tendermint rpc")
	cmd.Flags().MarkHidden(rpc.FlagNacosTmrpcUrls)
	cmd.Flags().String(rpc.FlagNacosTmrpcNamespaceID, "", "nacos namepace id for discovery service of tendermint rpc")
	cmd.Flags().MarkHidden(rpc.FlagNacosTmrpcNamespaceID)
	cmd.Flags().String(rpc.FlagNacosTmrpcAppName, "", " tendermint rpc name in nacos")
	cmd.Flags().MarkHidden(rpc.FlagNacosTmrpcAppName)
	cmd.Flags().String(rpc.FlagRpcExternalAddr, "127.0.0.1:26657", "Set the rpc-server external ip and port, when it is launched by Docker (default \"127.0.0.1:26657\")")

	cmd.Flags().String(rpc.FlagRateLimitAPI, "", "Set the RPC API to be controlled by the rate limit policy, such as \"eth_getLogs,eth_newFilter,eth_newBlockFilter,eth_newPendingTransactionFilter,eth_getFilterChanges\"")
	cmd.Flags().Int(rpc.FlagRateLimitCount, 0, "Set the count of requests allowed per second of rpc rate limiter")
	cmd.Flags().Int(rpc.FlagRateLimitBurst, 1, "Set the concurrent count of requests allowed of rpc rate limiter")
	cmd.Flags().Uint64(config.FlagGasLimitBuffer, 50, "Percentage to increase gas limit")
	cmd.Flags().String(rpc.FlagDisableAPI, "", "Set the RPC API to be disabled, such as \"eth_getLogs,eth_newFilter,eth_newBlockFilter,eth_newPendingTransactionFilter,eth_getFilterChanges\"")

	cmd.Flags().Bool(config.FlagEnableDynamicGp, false, "Enable node to dynamic support gas price suggest")
	cmd.Flags().MarkHidden(config.FlagEnableDynamicGp)
	cmd.Flags().Int64(config.FlagDynamicGpMaxTxNum, 300, "If tx number in the block is more than this, the network is congested.")
	cmd.Flags().Int64(config.FlagDynamicGpMaxGasUsed, tmtypes.NoGasUsedCap, "If the block gas used is more than this, the network is congested.")
	cmd.Flags().Int(config.FlagDynamicGpWeight, 80, "The recommended weight of dynamic gas price [1,100])")
	cmd.Flags().Int(config.FlagDynamicGpCheckBlocks, 5, "The recommended number of blocks checked of dynamic gas price [1,100])")
	cmd.Flags().Int(config.FlagDynamicGpCoefficient, 1, "Adjustment coefficient of dynamic gas price [1,100])")
	cmd.Flags().Int(config.FlagDynamicGpMode, tmtypes.MinimalGpMode, "Dynamic gas price mode (0: higher price|1: normal price|2: minimal price) is used to manage flags")

	cmd.Flags().Bool(config.FlagEnableHasBlockPartMsg, false, "Enable peer to broadcast HasBlockPartMessage")
	cmd.Flags().Bool(eth.FlagEnableMultiCall, false, "Enable node to support the eth_multiCall RPC API")
	cmd.Flags().Bool(eth.FlagAllowUnprotectedTxs, false, "Allow for unprotected (non EIP155 signed) transactions to be submitted via RPC")

	cmd.Flags().Bool(token.FlagOSSEnable, false, "Enable the function of exporting account data and uploading to oss")
	cmd.Flags().String(token.FlagOSSEndpoint, "", "The OSS datacenter endpoint such as http://oss-cn-hangzhou.aliyuncs.com")
	cmd.Flags().String(token.FlagOSSAccessKeyID, "", "The OSS access key Id")
	cmd.Flags().String(token.FlagOSSAccessKeySecret, "", "The OSS access key secret")
	cmd.Flags().String(token.FlagOSSBucketName, "", "The OSS bucket name")
	cmd.Flags().String(token.FlagOSSObjectPath, "", "The OSS object path")

	cmd.Flags().Bool(eth.FlagEnableTxPool, false, "Enable the function of txPool to support concurrency call eth_sendRawTransaction")
	cmd.Flags().Uint64(eth.TxPoolCap, 10000, "Set the txPool slice max length")
	cmd.Flags().Int(eth.BroadcastPeriodSecond, 10, "every BroadcastPeriodSecond second check the txPool, and broadcast when it's eligible")

	cmd.Flags().Bool(monitor.FlagEnableMonitor, false, "Enable the rpc monitor and register rpc metrics to prometheus")

	cmd.Flags().String(rpc.FlagKafkaAddr, "", "The address of kafka cluster to consume pending txs")
	cmd.Flags().String(rpc.FlagKafkaTopic, "", "The topic that the kafka writer will produce messages to")

	cmd.Flags().Bool(config.FlagEnableDynamic, false, "Enable dynamic configuration for nodes")
	cmd.Flags().String(config.FlagApollo, "", "Apollo connection config(IP|AppID|NamespaceName) for dynamic configuration")

	cmd.Flags().Bool(config.FlagPprofAutoDump, false, "Enable auto dump pprof")
	cmd.Flags().String(config.FlagPprofCollectInterval, "5s", "Interval for pprof dump loop")
	cmd.Flags().Int(config.FlagPprofCpuTriggerPercentMin, 45, "TriggerPercentMin of cpu to dump pprof")
	cmd.Flags().Int(config.FlagPprofCpuTriggerPercentDiff, 50, "TriggerPercentDiff of cpu to dump pprof")
	cmd.Flags().Int(config.FlagPprofCpuTriggerPercentAbs, 50, "TriggerPercentAbs of cpu to dump pprof")
	cmd.Flags().Int(config.FlagPprofMemTriggerPercentMin, 70, "TriggerPercentMin of mem to dump pprof")
	cmd.Flags().Int(config.FlagPprofMemTriggerPercentDiff, 50, "TriggerPercentDiff of mem to dump pprof")
	cmd.Flags().Int(config.FlagPprofMemTriggerPercentAbs, 75, "TriggerPercentAbs of cpu mem dump pprof")

	cmd.Flags().String(app.Elapsed, app.DefaultElapsedSchemas, "schemaName=1|0,,,")

	cmd.Flags().String(config.FlagPprofCoolDown, "3m", "The cool down time after every type of pprof dump")
	cmd.Flags().Int64(config.FlagPprofAbciElapsed, 5000, "Elapsed time of abci in millisecond for pprof dump")
	cmd.Flags().Bool(config.FlagPprofUseCGroup, false, "Use cgroup when okbchaind run in docker")

	cmd.Flags().String(tmdb.FlagGoLeveldbOpts, "", "Options of goleveldb. (cache_size=128MB,handlers_num=1024)")
	cmd.Flags().String(tmdb.FlagRocksdbOpts, "", "Options of rocksdb. (block_size=4KB,block_cache=1GB,statistics=true,allow_mmap_reads=true,max_open_files=-1,unordered_write=true,pipelined_write=true)")
	cmd.Flags().String(types.FlagNodeMode, "", "Node mode (rpc|val|archive) is used to manage flags")

	cmd.Flags().Bool(consensus.EnablePrerunTx, true, "enable proactively runtx mode, default close")
	cmd.Flags().String(automation.ConsensusRole, "", "consensus role")
	cmd.Flags().String(automation.ConsensusTestcase, "", "consensus test case file")

	cmd.Flags().Bool(app.FlagEnableRepairState, false, "Enable auto repair state on start")

	cmd.Flags().Bool(trace.FlagEnableAnalyzer, false, "Enable auto open log analyzer")
	cmd.Flags().Bool(sanity.FlagDisableSanity, false, "Disable sanity check")
	cmd.Flags().Int(tmtypes.FlagSigCacheSize, 200000, "Maximum number of signatures in the cache")
	cmd.Flags().Int(app.FlagGolangMaxThreads, 0, "Maximum number of golang threads")

	cmd.Flags().Int64(config.FlagCommitGapOffset, 0, "Offset to stagger ac ahead of proposal")
	cmd.Flags().MarkHidden(config.FlagCommitGapOffset)

	// flags for infura rpc
	cmd.Flags().Bool(infura.FlagEnable, false, "Enable infura rpc service")
	cmd.Flags().String(infura.FlagRedisUrl, "", "Redis url(host:port) of infura rpc service")
	cmd.Flags().String(infura.FlagRedisAuth, "", "Redis auth of infura rpc service")
	cmd.Flags().Int(infura.FlagRedisDB, 0, "Redis db of infura rpc service")
	cmd.Flags().String(infura.FlagMysqlUrl, "", "Mysql url(host:port) of infura rpc service")
	cmd.Flags().String(infura.FlagMysqlUser, "", "Mysql user of infura rpc service")
	cmd.Flags().String(infura.FlagMysqlPass, "", "Mysql password of infura rpc service")
	cmd.Flags().String(infura.FlagMysqlDB, "infura", "Mysql db name of infura rpc service")
	cmd.Flags().Int(infura.FlagCacheQueueSize, 0, "Cache queue size of infura rpc service")
	cmd.Flags().Int(config.FlagDebugGcInterval, 0, "Force gc every n heights for debug")
	cmd.Flags().String(rpc.FlagWebsocket, "8546", "websocket port to listen to")
	cmd.Flags().Int(backend.FlagLogsLimit, 0, "Maximum number of logs returned when calling eth_getLogs")
	cmd.Flags().Int(backend.FlagLogsTimeout, 60, "Maximum query duration when calling eth_getLogs")
	cmd.Flags().Int(websockets.FlagSubscribeLimit, 15, "Maximum subscription on a websocket connection")
	wasm.AddModuleInitFlags(cmd)
}
