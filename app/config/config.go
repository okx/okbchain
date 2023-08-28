package config

import (
	"fmt"
	"path"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/okx/okbchain/libs/cosmos-sdk/server"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/iavl"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/types"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	tmiavl "github.com/okx/okbchain/libs/iavl"
	iavlconfig "github.com/okx/okbchain/libs/iavl/config"
	"github.com/okx/okbchain/libs/system"
	"github.com/okx/okbchain/libs/system/trace"
	tmconfig "github.com/okx/okbchain/libs/tendermint/config"
	"github.com/okx/okbchain/libs/tendermint/consensus"
	"github.com/okx/okbchain/libs/tendermint/libs/log"
	"github.com/okx/okbchain/libs/tendermint/state"
	tmtypes "github.com/okx/okbchain/libs/tendermint/types"

	"github.com/spf13/viper"
)

var _ tmconfig.IDynamicConfig = &OkbcConfig{}
var _ iavlconfig.IDynamicConfig = &OkbcConfig{}

type OkbcConfig struct {
	// mempool.recheck
	mempoolRecheck bool
	// mempool.force_recheck_gap
	mempoolForceRecheckGap int64
	// mempool.size
	mempoolSize int
	// mempool.cache_size
	mempoolCacheSize int
	// mempool.flush
	mempoolFlush bool
	// mempool.max_tx_num_per_block
	maxTxNumPerBlock int64
	// mempool.enable_delete_min_gp_tx
	enableDeleteMinGPTx bool
	// mempool.max_gas_used_per_block
	maxGasUsedPerBlock int64
	// mempool.enable-pgu
	enablePGU bool
	// mempool.pgu-percentage-threshold
	pguPercentageThreshold int64
	// mempool.pgu-concurrency
	pguConcurrency int
	// mempool.pgu-adjustment
	pguAdjustment float64
	// mempool.pgu-persist
	pguPersist bool
	// mempool.node_key_whitelist
	nodeKeyWhitelist []string
	//mempool.check_tx_cost
	mempoolCheckTxCost bool
	//mempool.pending-pool-blacklist
	pendingPoolBlacklist string
	// p2p.sentry_addrs
	sentryAddrs []string

	// gas-limit-buffer
	gasLimitBuffer uint64

	// enable-dynamic-gp
	enableDynamicGp bool
	// dynamic-gp-weight
	dynamicGpWeight int
	// dynamic-gp-check-blocks
	dynamicGpCheckBlocks int
	// dynamic-gp-coefficient
	dynamicGpCoefficient int
	// dynamic-gp-max-gas-used
	dynamicGpMaxGasUsed int64
	// dynamic-gp-max-tx-num
	dynamicGpMaxTxNum int64
	// dynamic-gp-mode
	dynamicGpMode int

	// consensus.timeout_propose
	csTimeoutPropose time.Duration
	// consensus.timeout_propose_delta
	csTimeoutProposeDelta time.Duration
	// consensus.timeout_prevote
	csTimeoutPrevote time.Duration
	// consensus.timeout_prevote_delta
	csTimeoutPrevoteDelta time.Duration
	// consensus.timeout_precommit
	csTimeoutPrecommit time.Duration
	// consensus.timeout_precommit_delta
	csTimeoutPrecommitDelta time.Duration
	// consensus.timeout_commit
	csTimeoutCommit time.Duration

	// iavl-cache-size
	iavlCacheSize int
	// commit-gap-height
	commitGapHeight int64

	// iavl-fast-storage-cache-size
	iavlFSCacheSize int64

	// enable-wtx
	enableWtx bool

	// enable-analyzer
	enableAnalyzer bool

	deliverTxsMode int

	// active view change
	activeVC bool

	blockPartSizeBytes int
	blockCompressType  int
	blockCompressFlag  int

	// enable broadcast hasBlockPartMsg
	enableHasBlockPartMsg bool
	gcInterval            int

	iavlAcNoBatch bool

	//
	commitGapOffset int64

	maxSubscriptionClients int
}

const (
	FlagEnableDynamic = "config.enable-dynamic"

	FlagMempoolRecheck             = "mempool.recheck"
	FlagMempoolForceRecheckGap     = "mempool.force_recheck_gap"
	FlagMempoolSize                = "mempool.size"
	FlagMempoolCacheSize           = "mempool.cache_size"
	FlagMempoolFlush               = "mempool.flush"
	FlagMaxTxNumPerBlock           = "mempool.max_tx_num_per_block"
	FlagMaxGasUsedPerBlock         = "mempool.max_gas_used_per_block"
	FlagEnablePGU                  = "mempool.enable-pgu"
	FlagPGUPercentageThreshold     = "mempool.pgu-percentage-threshold"
	FlagPGUConcurrency             = "mempool.pgu-concurrency"
	FlagPGUAdjustment              = "mempool.pgu-adjustment"
	FlagPGUPersist                 = "mempool.pgu-persist"
	FlagNodeKeyWhitelist           = "mempool.node_key_whitelist"
	FlagMempoolCheckTxCost         = "mempool.check_tx_cost"
	FlagMempoolEnableDeleteMinGPTx = "mempool.enable_delete_min_gp_tx"
	FlagPendingPoolBlacklist       = "mempool.pending-pool-blacklist"
	FlagGasLimitBuffer             = "gas-limit-buffer"
	FlagEnableDynamicGp            = "enable-dynamic-gp"
	FlagDynamicGpMode              = "dynamic-gp-mode"
	FlagDynamicGpWeight            = "dynamic-gp-weight"
	FlagDynamicGpCheckBlocks       = "dynamic-gp-check-blocks"
	FlagDynamicGpCoefficient       = "dynamic-gp-coefficient"
	FlagDynamicGpMaxGasUsed        = "dynamic-gp-max-gas-used"
	FlagDynamicGpMaxTxNum          = "dynamic-gp-max-tx-num"
	FlagEnableWrappedTx            = "enable-wtx"
	FlagSentryAddrs                = "p2p.sentry_addrs"
	FlagCsTimeoutPropose           = "consensus.timeout_propose"
	FlagCsTimeoutProposeDelta      = "consensus.timeout_propose_delta"
	FlagCsTimeoutPrevote           = "consensus.timeout_prevote"
	FlagCsTimeoutPrevoteDelta      = "consensus.timeout_prevote_delta"
	FlagCsTimeoutPrecommit         = "consensus.timeout_precommit"
	FlagCsTimeoutPrecommitDelta    = "consensus.timeout_precommit_delta"
	FlagCsTimeoutCommit            = "consensus.timeout_commit"
	FlagEnableHasBlockPartMsg      = "enable-blockpart-ack"
	FlagDebugGcInterval            = "debug.gc-interval"
	FlagCommitGapOffset            = "commit-gap-offset"
	FlagMaxSubscriptionClients     = "max-subscription-clients"
)

var (
	testnetNodeIdWhitelist = []string{
		// RPC nodes for users
		"3a339568305c5aff58a1f134437b608490e2ec6d",
		"b9e7bf85886f1d11ee5079726a268401bf7b6254",
		"54c5ffc54e10a311660d16a96d54ddc59edb5555",
		"d77e385de87acdd042973c5d3029b02db8d767ff",
		"5cfdc51d1502fbe44d1b2a7f1f37e1016ad5ee97",
		"704be3bf19866f2aa5c77b09f003e2b69c552927",
		"1767342f12cb0e1e393a42c56d63d7486b2c54cd",
		"d33084a8c7bab8c9b6f286378b5e3ac197caa41a",
		"6a96b0a094ec9aaff2b7148b0c5811618b41c101",
		// RPC nodes for developers
		"3a35faa50649164d59f07f31d78946ca07464e9c",
		"cee36e7fbc99eaa02bd9af692dae367a867c43f4",
		"fbcae686695cd17ee8319bbd6b9b0aaf0f10d8c4",
		"2c34f93a8665d694e56319ccdc6738b203c33848",
		"f689ab031c0758367af229aa8df65ac69762327d",
		"58c495e040a1576ebc1f386a7dc04c4e60ee63d7",
		"a2f685db92a88c18780d8d9cb1162ab61517ae64",
		"8d4b0539b95b60e1691eac77be4aa7295645d9d9",
		"6f328902a0bf5e7b922d6a5980dd6888097db984",
		"12503ae035dd7ff04e19b0ca2c9c8b54a0a56b22",
		// validator nodes
		"c39ca38c650b920f9b6c5a9aed7ff904124ec3ad",
		"d937e21fd489809add23dc3e55ed78d947217aa8",
		"a3eb3c129e49137d5e1665bbf87b6f2be70a0b85",
		"b171a9ef83b95c28182bc7aa7ea8639d04e572e7",
		"3a700a3849c401396b1c51eb65b1cfc1a8c4394b",
		"0208e66d4ca746ec535a0bf05409dc87df408b15",
		"ed1819fa1eae52ddec4c0f8cddd80b9cb7c68a22",
		"0b3ab9597a66f2f94c8efa4ccb6ed2a1f44d4184",
		"67b29551c7c3839ad6c93379991344266aec3829",
		"cd07b20b596aac923a1d5bb022581e279755aff1",
		"6ce06a89a968a4204d9dcb470f2275767c8dfa68",
		"6dd38d96df3ccbca95769ee15bdfdd952ad007c5",
		"fcc95bfee6ea74bdf385be3a29072329603676e5",
		"7b5b3041d2b3546a236b6df7ff7e06a19a5cae46",
		"c098585e299ff7afe6f354c4431550d6919bdd0d",
		"5b44fb4af4cfb72286162cb49a3bc04cb8187775",
		"358e3399b68fb67787f1386c685db2e75352d9eb",
		"96d9cb96041c053e63ff7d0c7d81dfab706136e4",
		"0de948586fb30293d1dd14a99ebc3f719deb7c6f",
		"284e87518752c8f655fe217113fa86ba7d6ca72f",
		"7f2b8a6b9b8b12247e6992aeb32d69e169c2f5ac",
	}

	mainnetNodeIdWhitelist = []string{}

	okbcConfig *OkbcConfig
	once       sync.Once
	confLogger log.Logger
)

func GetChainMaxGasUsedPerBlock() int64 {
	return sdk.GetMaxGasUsedPerBlock()
}

func GetOkbcConfig() *OkbcConfig {
	once.Do(func() {
		okbcConfig = NewOkbcConfig()
	})
	return okbcConfig
}

func NewOkbcConfig() *OkbcConfig {
	c := defaultOkbcConfig()
	c.loadFromConfig()

	if viper.GetBool(FlagEnableDynamic) {
		if viper.IsSet(FlagApollo) {
			loaded := c.loadFromApollo()
			if !loaded {
				panic("failed to connect apollo or no config items in apollo")
			}
		} else {
			ok, err := c.loadFromLocal()
			if err != nil {
				confLogger.Error("failed to load config from local", "err", err)
			}
			if !ok {
				confLogger.Error("failed to load config from local")
			} else {
				confLogger.Info("load config from local success")
			}
		}
	}

	return c
}

func defaultOkbcConfig() *OkbcConfig {
	return &OkbcConfig{
		mempoolRecheck:         false,
		mempoolForceRecheckGap: 2000,
		commitGapHeight:        iavlconfig.DefaultCommitGapHeight,
		iavlFSCacheSize:        tmiavl.DefaultIavlFastStorageCacheSize,
	}
}

func RegisterDynamicConfig(logger log.Logger) {
	confLogger = logger
	// set the dynamic config
	okbConfig := GetOkbcConfig()
	tmconfig.SetDynamicConfig(okbConfig)
	iavlconfig.SetDynamicConfig(okbConfig)
	trace.SetDynamicConfig(okbConfig)
}

func (c *OkbcConfig) loadFromConfig() {
	c.SetMempoolRecheck(viper.GetBool(FlagMempoolRecheck))
	c.SetMempoolForceRecheckGap(viper.GetInt64(FlagMempoolForceRecheckGap))
	c.SetMempoolSize(viper.GetInt(FlagMempoolSize))
	c.SetMempoolCacheSize(viper.GetInt(FlagMempoolCacheSize))
	c.SetMempoolFlush(viper.GetBool(FlagMempoolFlush))
	c.SetMempoolCheckTxCost(viper.GetBool(FlagMempoolCheckTxCost))
	c.SetMaxTxNumPerBlock(viper.GetInt64(FlagMaxTxNumPerBlock))
	c.SetEnableDeleteMinGPTx(viper.GetBool(FlagMempoolEnableDeleteMinGPTx))
	c.SetPendingPoolBlacklist(viper.GetString(FlagPendingPoolBlacklist))
	c.SetMaxGasUsedPerBlock(viper.GetInt64(FlagMaxGasUsedPerBlock))
	c.SetEnablePGU(viper.GetBool(FlagEnablePGU))
	c.SetPGUPercentageThreshold(viper.GetInt64(FlagPGUPercentageThreshold))
	c.SetPGUConcurrency(viper.GetInt(FlagPGUConcurrency))
	c.SetPGUAdjustment(viper.GetFloat64(FlagPGUAdjustment))
	c.SetPGUPersist(viper.GetBool(FlagPGUPersist))
	c.SetGasLimitBuffer(viper.GetUint64(FlagGasLimitBuffer))

	c.SetEnableDynamicGp(viper.GetBool(FlagEnableDynamicGp))
	c.SetDynamicGpWeight(viper.GetInt(FlagDynamicGpWeight))
	c.SetDynamicGpCheckBlocks(viper.GetInt(FlagDynamicGpCheckBlocks))
	c.SetDynamicGpCoefficient(viper.GetInt(FlagDynamicGpCoefficient))
	c.SetDynamicGpMaxGasUsed(viper.GetInt64(FlagDynamicGpMaxGasUsed))
	c.SetDynamicGpMaxTxNum(viper.GetInt64(FlagDynamicGpMaxTxNum))

	c.SetDynamicGpMode(viper.GetInt(FlagDynamicGpMode))
	c.SetCsTimeoutPropose(viper.GetDuration(FlagCsTimeoutPropose))
	c.SetCsTimeoutProposeDelta(viper.GetDuration(FlagCsTimeoutProposeDelta))
	c.SetCsTimeoutPrevote(viper.GetDuration(FlagCsTimeoutPrevote))
	c.SetCsTimeoutPrevoteDelta(viper.GetDuration(FlagCsTimeoutPrevoteDelta))
	c.SetCsTimeoutPrecommit(viper.GetDuration(FlagCsTimeoutPrecommit))
	c.SetCsTimeoutPrecommitDelta(viper.GetDuration(FlagCsTimeoutPrecommitDelta))
	c.SetCsTimeoutCommit(viper.GetDuration(FlagCsTimeoutCommit))
	c.SetIavlCacheSize(viper.GetInt(iavl.FlagIavlCacheSize))
	c.SetIavlFSCacheSize(viper.GetInt64(tmiavl.FlagIavlFastStorageCacheSize))
	c.SetCommitGapHeight(viper.GetInt64(server.FlagCommitGapHeight))
	c.SetSentryAddrs(viper.GetString(FlagSentryAddrs))
	c.SetNodeKeyWhitelist(viper.GetString(FlagNodeKeyWhitelist))
	c.SetEnableWtx(viper.GetBool(FlagEnableWrappedTx))
	c.SetEnableAnalyzer(viper.GetBool(trace.FlagEnableAnalyzer))
	c.SetDeliverTxsExecuteMode(viper.GetInt(state.FlagDeliverTxsExecMode))
	c.SetCommitGapOffset(viper.GetInt64(FlagCommitGapOffset))
	c.SetBlockPartSize(viper.GetInt(server.FlagBlockPartSizeBytes))
	c.SetEnableHasBlockPartMsg(viper.GetBool(FlagEnableHasBlockPartMsg))
	c.SetGcInterval(viper.GetInt(FlagDebugGcInterval))
	c.SetIavlAcNoBatch(viper.GetBool(tmiavl.FlagIavlCommitAsyncNoBatch))
	c.SetMaxSubscriptionClients(viper.GetInt(FlagMaxSubscriptionClients))
}

func resolveNodeKeyWhitelist(plain string) []string {
	if len(plain) == 0 {
		return []string{}
	}
	return strings.Split(plain, ",")
}

func resolveSentryAddrs(plain string) []string {
	if len(plain) == 0 {
		return []string{}
	}
	return strings.Split(plain, ";")
}

func (c *OkbcConfig) loadFromApollo() bool {
	client := NewApolloClient(c)
	return client.LoadConfig()
}

func (c *OkbcConfig) loadFromLocal() (bool, error) {
	var err error
	rootDir := viper.GetString("home")
	configPath := path.Join(rootDir, "config", LocalDynamicConfigPath)
	configPath, err = filepath.Abs(configPath)
	if err != nil {
		return false, err
	}
	client, err := NewLocalClient(configPath, c, confLogger)
	if err != nil {
		return false, err
	}
	ok := client.LoadConfig()
	err = client.Enable()
	return ok, err
}

func (c *OkbcConfig) format() string {
	return fmt.Sprintf(`%s config:
	mempool.recheck: %v
	mempool.force_recheck_gap: %d
	mempool.size: %d
	mempool.cache_size: %d

	mempool.flush: %v
	mempool.max_tx_num_per_block: %d
	mempool.enable_delete_min_gp_tx: %v
	mempool.pending-pool-blacklist: %v
	mempool.max_gas_used_per_block: %d
	mempool.check_tx_cost: %v

	gas-limit-buffer: %d
	dynamic-gp-weight: %d
	dynamic-gp-check-blocks: %d
	dynamic-gp-coefficient: %d
	dynamic-gp-max-gas-used: %d
	dynamic-gp-max-tx-num: %d
	dynamic-gp-mode: %d

	consensus.timeout_propose: %s
	consensus.timeout_propose_delta: %s
	consensus.timeout_prevote: %s
	consensus.timeout_prevote_delta: %s
	consensus.timeout_precommit: %s
	consensus.timeout_precommit_delta: %s
	consensus.timeout_commit: %s
	
	iavl-cache-size: %d
    iavl-fast-storage-cache-size: %d
    commit-gap-height: %d
	enable-analyzer: %v
    iavl-commit-async-no-batch: %v
	active-view-change: %v
	max_subscription_clients: %v`, system.ChainName,
		c.GetMempoolRecheck(),
		c.GetMempoolForceRecheckGap(),
		c.GetMempoolSize(),
		c.GetMempoolCacheSize(),
		c.GetMempoolFlush(),
		c.GetMaxTxNumPerBlock(),
		c.GetEnableDeleteMinGPTx(),
		c.GetPendingPoolBlacklist(),
		c.GetMaxGasUsedPerBlock(),
		c.GetMempoolCheckTxCost(),
		c.GetGasLimitBuffer(),
		c.GetDynamicGpWeight(),
		c.GetDynamicGpCheckBlocks(),
		c.GetDynamicGpCoefficient(),
		c.GetDynamicGpMaxGasUsed(),
		c.GetDynamicGpMaxTxNum(),
		c.GetDynamicGpMode(),
		c.GetCsTimeoutPropose(),
		c.GetCsTimeoutProposeDelta(),
		c.GetCsTimeoutPrevote(),
		c.GetCsTimeoutPrevoteDelta(),
		c.GetCsTimeoutPrecommit(),
		c.GetCsTimeoutPrecommitDelta(),
		c.GetCsTimeoutCommit(),
		c.GetIavlCacheSize(),
		c.GetIavlFSCacheSize(),
		c.GetCommitGapHeight(),
		c.GetEnableAnalyzer(),
		c.GetIavlAcNoBatch(),
		c.GetActiveVC(),
		c.GetMaxSubscriptionClients(),
	)
}

func (c *OkbcConfig) update(key, value interface{}) {
	k, v := key.(string), value.(string)
	c.updateFromKVStr(k, v)
}

func (c *OkbcConfig) updateFromKVStr(k, v string) {
	switch k {
	case FlagMempoolRecheck:
		r, err := strconv.ParseBool(v)
		if err != nil {
			return
		}
		c.SetMempoolRecheck(r)
	case FlagMempoolForceRecheckGap:
		r, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return
		}
		c.SetMempoolForceRecheckGap(r)
	case FlagMempoolSize:
		r, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		c.SetMempoolSize(r)
	case FlagMempoolCacheSize:
		r, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		c.SetMempoolCacheSize(r)
	case FlagMempoolFlush:
		r, err := strconv.ParseBool(v)
		if err != nil {
			return
		}
		c.SetMempoolFlush(r)
	case FlagMaxTxNumPerBlock:
		r, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return
		}
		c.SetMaxTxNumPerBlock(r)
	case FlagMempoolEnableDeleteMinGPTx:
		r, err := strconv.ParseBool(v)
		if err != nil {
			return
		}
		c.SetEnableDeleteMinGPTx(r)
	case FlagPendingPoolBlacklist:
		c.SetPendingPoolBlacklist(v)
	case FlagNodeKeyWhitelist:
		c.SetNodeKeyWhitelist(v)
	case FlagMempoolCheckTxCost:
		r, err := strconv.ParseBool(v)
		if err != nil {
			return
		}
		c.SetMempoolCheckTxCost(r)
	case FlagSentryAddrs:
		c.SetSentryAddrs(v)
	case FlagMaxGasUsedPerBlock:
		r, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return
		}
		c.SetMaxGasUsedPerBlock(r)
	case FlagEnablePGU:
		r, err := strconv.ParseBool(v)
		if err != nil {
			return
		}
		c.SetEnablePGU(r)
	case FlagPGUPercentageThreshold:
		r, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return
		}
		c.SetPGUPercentageThreshold(r)
	case FlagPGUConcurrency:
		r, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		c.SetPGUConcurrency(r)
	case FlagPGUAdjustment:
		r, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return
		}
		c.SetPGUAdjustment(r)
	case FlagPGUPersist:
		r, err := strconv.ParseBool(v)
		if err != nil {
			return
		}
		c.SetPGUPersist(r)
	case FlagGasLimitBuffer:
		r, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return
		}
		c.SetGasLimitBuffer(r)
	case FlagEnableDynamicGp:
		r, err := strconv.ParseBool(v)
		if err != nil {
			return
		}
		c.SetEnableDynamicGp(r)
	case FlagDynamicGpWeight:
		r, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		c.SetDynamicGpWeight(r)
	case FlagDynamicGpCheckBlocks:
		r, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		c.SetDynamicGpCheckBlocks(r)
	case FlagDynamicGpCoefficient:
		r, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		c.SetDynamicGpCoefficient(r)
	case FlagDynamicGpMaxGasUsed:
		r, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return
		}
		c.SetDynamicGpMaxGasUsed(r)
	case FlagDynamicGpMaxTxNum:
		r, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return
		}
		c.SetDynamicGpMaxTxNum(r)
	case FlagDynamicGpMode:
		r, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		c.SetDynamicGpMode(r)
	case FlagCsTimeoutPropose:
		r, err := time.ParseDuration(v)
		if err != nil {
			return
		}
		c.SetCsTimeoutPropose(r)
	case FlagCsTimeoutProposeDelta:
		r, err := time.ParseDuration(v)
		if err != nil {
			return
		}
		c.SetCsTimeoutProposeDelta(r)
	case FlagCsTimeoutPrevote:
		r, err := time.ParseDuration(v)
		if err != nil {
			return
		}
		c.SetCsTimeoutPrevote(r)
	case FlagCsTimeoutPrevoteDelta:
		r, err := time.ParseDuration(v)
		if err != nil {
			return
		}
		c.SetCsTimeoutPrevoteDelta(r)
	case FlagCsTimeoutPrecommit:
		r, err := time.ParseDuration(v)
		if err != nil {
			return
		}
		c.SetCsTimeoutPrecommit(r)
	case FlagCsTimeoutPrecommitDelta:
		r, err := time.ParseDuration(v)
		if err != nil {
			return
		}
		c.SetCsTimeoutPrecommitDelta(r)
	case FlagCsTimeoutCommit:
		r, err := time.ParseDuration(v)
		if err != nil {
			return
		}
		c.SetCsTimeoutCommit(r)
	case iavl.FlagIavlCacheSize:
		r, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		c.SetIavlCacheSize(r)
	case tmiavl.FlagIavlFastStorageCacheSize:
		r, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		c.SetIavlFSCacheSize(int64(r))
	case server.FlagCommitGapHeight:
		r, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return
		}
		c.SetCommitGapHeight(r)
	case trace.FlagEnableAnalyzer:
		r, err := strconv.ParseBool(v)
		if err != nil {
			return
		}
		c.SetEnableAnalyzer(r)
	case state.FlagDeliverTxsExecMode:
		r, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		c.SetDeliverTxsExecuteMode(r)
	case server.FlagActiveViewChange:
		r, err := strconv.ParseBool(v)
		if err != nil {
			return
		}
		c.SetActiveVC(r)
	case server.FlagBlockPartSizeBytes:
		r, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		c.SetBlockPartSize(r)
	case tmtypes.FlagBlockCompressType:
		r, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		c.SetBlockCompressType(r)
	case tmtypes.FlagBlockCompressFlag:
		r, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		c.SetBlockCompressFlag(r)
	case FlagEnableHasBlockPartMsg:
		r, err := strconv.ParseBool(v)
		if err != nil {
			return
		}
		c.SetEnableHasBlockPartMsg(r)
	case FlagDebugGcInterval:
		r, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		c.SetGcInterval(r)
	case tmiavl.FlagIavlCommitAsyncNoBatch:
		r, err := strconv.ParseBool(v)
		if err != nil {
			return
		}
		c.SetIavlAcNoBatch(r)
	case FlagCommitGapOffset:
		r, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return
		}
		c.SetCommitGapOffset(r)
	case FlagMaxSubscriptionClients:
		r, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		c.SetMaxSubscriptionClients(r)
	}

}

func (c *OkbcConfig) GetEnableAnalyzer() bool {
	return c.enableAnalyzer
}
func (c *OkbcConfig) SetEnableAnalyzer(value bool) {
	c.enableAnalyzer = value
}

func (c *OkbcConfig) GetMempoolRecheck() bool {
	return c.mempoolRecheck
}
func (c *OkbcConfig) SetMempoolRecheck(value bool) {
	c.mempoolRecheck = value
}

func (c *OkbcConfig) GetMempoolForceRecheckGap() int64 {
	return c.mempoolForceRecheckGap
}
func (c *OkbcConfig) SetMempoolForceRecheckGap(value int64) {
	if value <= 0 {
		return
	}
	c.mempoolForceRecheckGap = value
}

func (c *OkbcConfig) GetMempoolSize() int {
	return c.mempoolSize
}
func (c *OkbcConfig) SetMempoolSize(value int) {
	if value < 0 {
		return
	}
	c.mempoolSize = value
}

func (c *OkbcConfig) GetMempoolCacheSize() int {
	return c.mempoolCacheSize
}
func (c *OkbcConfig) SetMempoolCacheSize(value int) {
	if value < 0 {
		return
	}
	c.mempoolCacheSize = value
}

func (c *OkbcConfig) GetMempoolFlush() bool {
	return c.mempoolFlush
}
func (c *OkbcConfig) SetMempoolFlush(value bool) {
	c.mempoolFlush = value
}

func (c *OkbcConfig) GetEnableWtx() bool {
	return c.enableWtx
}

func (c *OkbcConfig) SetDeliverTxsExecuteMode(mode int) {
	c.deliverTxsMode = mode
}

func (c *OkbcConfig) GetDeliverTxsExecuteMode() int {
	return c.deliverTxsMode
}

func (c *OkbcConfig) SetEnableWtx(value bool) {
	c.enableWtx = value
}

func (c *OkbcConfig) GetNodeKeyWhitelist() []string {
	return c.nodeKeyWhitelist
}

func (c *OkbcConfig) GetMempoolCheckTxCost() bool {
	return c.mempoolCheckTxCost
}
func (c *OkbcConfig) SetMempoolCheckTxCost(value bool) {
	c.mempoolCheckTxCost = value
}

func (c *OkbcConfig) SetNodeKeyWhitelist(value string) {
	idList := resolveNodeKeyWhitelist(value)

	for _, id := range idList {
		if id == "testnet-node-ids" {
			c.nodeKeyWhitelist = append(c.nodeKeyWhitelist, testnetNodeIdWhitelist...)
		} else if id == "mainnet-node-ids" {
			c.nodeKeyWhitelist = append(c.nodeKeyWhitelist, mainnetNodeIdWhitelist...)
		} else {
			c.nodeKeyWhitelist = append(c.nodeKeyWhitelist, id)
		}
	}
}

func (c *OkbcConfig) GetSentryAddrs() []string {
	return c.sentryAddrs
}

func (c *OkbcConfig) SetSentryAddrs(value string) {
	addrs := resolveSentryAddrs(value)
	for _, addr := range addrs {
		c.sentryAddrs = append(c.sentryAddrs, strings.TrimSpace(addr))
	}
}

func (c *OkbcConfig) GetMaxTxNumPerBlock() int64 {
	return c.maxTxNumPerBlock
}
func (c *OkbcConfig) SetMaxTxNumPerBlock(value int64) {
	if value < 0 {
		return
	}
	c.maxTxNumPerBlock = value
}

func (c *OkbcConfig) GetEnableDeleteMinGPTx() bool {
	return c.enableDeleteMinGPTx
}

func (c *OkbcConfig) SetEnableDeleteMinGPTx(enable bool) {
	c.enableDeleteMinGPTx = enable
}

func (c *OkbcConfig) GetMaxGasUsedPerBlock() int64 {
	if c.maxGasUsedPerBlock == -1 {
		return GetChainMaxGasUsedPerBlock()
	}
	return c.maxGasUsedPerBlock
}

func (c *OkbcConfig) SetMaxGasUsedPerBlock(value int64) {
	if value < -1 {
		return
	}
	c.maxGasUsedPerBlock = value
}

func (c *OkbcConfig) GetEnablePGU() bool {
	return c.enablePGU
}

func (c *OkbcConfig) SetEnablePGU(value bool) {
	c.enablePGU = value
}

func (c *OkbcConfig) GetPGUPercentageThreshold() int64 {
	return c.pguPercentageThreshold
}

func (c *OkbcConfig) SetPGUPercentageThreshold(value int64) {
	c.pguPercentageThreshold = value
}

func (c *OkbcConfig) GetPGUConcurrency() int {
	return c.pguConcurrency
}

func (c *OkbcConfig) SetPGUConcurrency(value int) {
	c.pguConcurrency = value
}

func (c *OkbcConfig) GetPGUAdjustment() float64 {
	return c.pguAdjustment
}

func (c *OkbcConfig) SetPGUAdjustment(value float64) {
	c.pguAdjustment = value
}

func (c *OkbcConfig) GetPGUPersist() bool {
	return c.pguPersist
}

func (c *OkbcConfig) SetPGUPersist(value bool) {
	c.pguPersist = value
}

func (c *OkbcConfig) GetGasLimitBuffer() uint64 {
	return c.gasLimitBuffer
}
func (c *OkbcConfig) SetGasLimitBuffer(value uint64) {
	c.gasLimitBuffer = value
}

func (c *OkbcConfig) GetEnableDynamicGp() bool {
	return c.enableDynamicGp
}

func (c *OkbcConfig) SetEnableDynamicGp(value bool) {
	c.enableDynamicGp = value
}

func (c *OkbcConfig) GetDynamicGpWeight() int {
	return c.dynamicGpWeight
}

func (c *OkbcConfig) SetDynamicGpWeight(value int) {
	if value <= 0 {
		value = 1
	} else if value > 100 {
		value = 100
	}
	c.dynamicGpWeight = value
}

func (c *OkbcConfig) GetDynamicGpCoefficient() int {
	return c.dynamicGpCoefficient
}
func (c *OkbcConfig) SetDynamicGpCoefficient(value int) {
	if value <= 0 {
		value = 1
	} else if value > 100 {
		value = 100
	}
	c.dynamicGpCoefficient = value
}

func (c *OkbcConfig) GetDynamicGpMaxGasUsed() int64 {
	return c.dynamicGpMaxGasUsed
}

func (c *OkbcConfig) SetDynamicGpMaxGasUsed(value int64) {
	if value < -1 {
		return
	}
	c.dynamicGpMaxGasUsed = value
}

func (c *OkbcConfig) GetDynamicGpMaxTxNum() int64 {
	return c.dynamicGpMaxTxNum
}

func (c *OkbcConfig) SetDynamicGpMaxTxNum(value int64) {
	if value < 0 {
		return
	}
	c.dynamicGpMaxTxNum = value
}

func (c *OkbcConfig) GetDynamicGpMode() int {
	return c.dynamicGpMode
}

func (c *OkbcConfig) SetDynamicGpMode(value int) {
	if value < 0 || value > 2 {
		return
	}
	c.dynamicGpMode = value
}

func (c *OkbcConfig) GetDynamicGpCheckBlocks() int {
	return c.dynamicGpCheckBlocks
}

func (c *OkbcConfig) SetDynamicGpCheckBlocks(value int) {
	if value <= 0 {
		value = 1
	} else if value > 100 {
		value = 100
	}
	c.dynamicGpCheckBlocks = value
}

func (c *OkbcConfig) GetCsTimeoutPropose() time.Duration {
	return c.csTimeoutPropose
}
func (c *OkbcConfig) SetCsTimeoutPropose(value time.Duration) {
	if value < 0 {
		return
	}
	c.csTimeoutPropose = value
}

func (c *OkbcConfig) GetCsTimeoutProposeDelta() time.Duration {
	return c.csTimeoutProposeDelta
}
func (c *OkbcConfig) SetCsTimeoutProposeDelta(value time.Duration) {
	if value < 0 {
		return
	}
	c.csTimeoutProposeDelta = value
}

func (c *OkbcConfig) GetCsTimeoutPrevote() time.Duration {
	return c.csTimeoutPrevote
}
func (c *OkbcConfig) SetCsTimeoutPrevote(value time.Duration) {
	if value < 0 {
		return
	}
	c.csTimeoutPrevote = value
}

func (c *OkbcConfig) GetCsTimeoutPrevoteDelta() time.Duration {
	return c.csTimeoutPrevoteDelta
}
func (c *OkbcConfig) SetCsTimeoutPrevoteDelta(value time.Duration) {
	if value < 0 {
		return
	}
	c.csTimeoutPrevoteDelta = value
}

func (c *OkbcConfig) GetCsTimeoutPrecommit() time.Duration {
	return c.csTimeoutPrecommit
}
func (c *OkbcConfig) SetCsTimeoutPrecommit(value time.Duration) {
	if value < 0 {
		return
	}
	c.csTimeoutPrecommit = value
}

func (c *OkbcConfig) GetCsTimeoutPrecommitDelta() time.Duration {
	return c.csTimeoutPrecommitDelta
}
func (c *OkbcConfig) SetCsTimeoutPrecommitDelta(value time.Duration) {
	if value < 0 {
		return
	}
	c.csTimeoutPrecommitDelta = value
}

func (c *OkbcConfig) GetCsTimeoutCommit() time.Duration {
	return c.csTimeoutCommit
}
func (c *OkbcConfig) SetCsTimeoutCommit(value time.Duration) {
	if value < 0 {
		return
	}
	c.csTimeoutCommit = value
}

func (c *OkbcConfig) GetIavlCacheSize() int {
	return c.iavlCacheSize
}
func (c *OkbcConfig) SetIavlCacheSize(value int) {
	c.iavlCacheSize = value
}

func (c *OkbcConfig) GetIavlFSCacheSize() int64 {
	return c.iavlFSCacheSize
}

func (c *OkbcConfig) SetIavlFSCacheSize(value int64) {
	c.iavlFSCacheSize = value
}

func (c *OkbcConfig) GetCommitGapHeight() int64 {
	return atomic.LoadInt64(&c.commitGapHeight)
}
func (c *OkbcConfig) SetCommitGapHeight(value int64) {
	if IsPruningOptionNothing() { // pruning nothing the gap should 1
		value = 1
	}
	if value <= 0 {
		return
	}
	atomic.StoreInt64(&c.commitGapHeight, value)
}

func IsPruningOptionNothing() bool {
	strategy := strings.ToLower(viper.GetString(server.FlagPruning))
	if strategy == types.PruningOptionNothing {
		return true
	}
	return false
}

func (c *OkbcConfig) GetActiveVC() bool {
	return c.activeVC
}
func (c *OkbcConfig) SetActiveVC(value bool) {
	c.activeVC = value
	consensus.SetActiveVC(value)
}

func (c *OkbcConfig) GetBlockPartSize() int {
	return c.blockPartSizeBytes
}
func (c *OkbcConfig) SetBlockPartSize(value int) {
	c.blockPartSizeBytes = value
	tmtypes.UpdateBlockPartSizeBytes(value)
}

func (c *OkbcConfig) GetBlockCompressType() int {
	return c.blockCompressType
}
func (c *OkbcConfig) SetBlockCompressType(value int) {
	c.blockCompressType = value
	tmtypes.BlockCompressType = value
}

func (c *OkbcConfig) GetBlockCompressFlag() int {
	return c.blockCompressFlag
}
func (c *OkbcConfig) SetBlockCompressFlag(value int) {
	c.blockCompressFlag = value
	tmtypes.BlockCompressFlag = value
}

func (c *OkbcConfig) GetGcInterval() int {
	return c.gcInterval
}

func (c *OkbcConfig) SetGcInterval(value int) {
	// close gc for debug
	if value > 0 {
		debug.SetGCPercent(-1)
	} else {
		debug.SetGCPercent(100)
	}
	c.gcInterval = value

}

func (c *OkbcConfig) GetCommitGapOffset() int64 {
	return c.commitGapOffset
}

func (c *OkbcConfig) SetCommitGapOffset(value int64) {
	if value < 0 {
		value = 0
	}
	c.commitGapOffset = value
}

func (c *OkbcConfig) GetEnableHasBlockPartMsg() bool {
	return c.enableHasBlockPartMsg
}

func (c *OkbcConfig) SetEnableHasBlockPartMsg(value bool) {
	c.enableHasBlockPartMsg = value
}

func (c *OkbcConfig) GetIavlAcNoBatch() bool {
	return c.iavlAcNoBatch
}

func (c *OkbcConfig) SetIavlAcNoBatch(value bool) {
	c.iavlAcNoBatch = value
}

func (c *OkbcConfig) SetMaxSubscriptionClients(v int) {
	if v < 0 {
		v = 0
	}
	c.maxSubscriptionClients = v
}

func (c *OkbcConfig) GetMaxSubscriptionClients() int {
	return c.maxSubscriptionClients
}

func (c *OkbcConfig) SetPendingPoolBlacklist(v string) {
	c.pendingPoolBlacklist = v
}

func (c *OkbcConfig) GetPendingPoolBlacklist() string {
	return c.pendingPoolBlacklist
}
