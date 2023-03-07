package types

import (
	"github.com/okx/okbchain/libs/system"
	"strconv"
	"sync"
)

// Disable followings after milestoneMercuryHeight
// 1. TransferToContractBlock
// 2. ChangeEvmDenomByProposal
// 3. BankTransferBlock
// 4. ibc

var (
	MILESTONE_MARS_HEIGHT string
	milestoneMarsHeight   int64

	milestoneEarthHeight  int64
	milestoneVenus4Height int64

	// note: it stores the earlies height of the node,and it is used by cli
	nodePruneHeight int64

	once sync.Once
)

const (
	MainNet = system.Chain + "-66"
	TestNet = system.Chain + "-65"

	MILESTONE_EARTH  = "earth"
	MILESTONE_Venus4 = "venus4"
)

const (
	TestNetChangeChainId = 2270901
	TestNetChainName1    = system.Chain + "-65"
)

func init() {
	once.Do(func() {
		milestoneMarsHeight = string2number(MILESTONE_MARS_HEIGHT)
	})
}

func string2number(input string) int64 {
	if len(input) == 0 {
		input = "0"
	}
	res, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		panic(err)
	}
	return res
}

func SetupMainNetEnvironment(pruneH int64) {
	nodePruneHeight = pruneH
}

func SetupTestNetEnvironment(pruneH int64) {
	nodePruneHeight = pruneH
}

// use MPT storage model to replace IAVL storage model
func HigherThanMars(height int64) bool {
	if milestoneMarsHeight == 0 {
		return false
	}
	return height >= milestoneMarsHeight
}

// 2322600 is mainnet GenesisHeight
func IsMainNet() bool {
	//return MILESTONE_GENESIS_HEIGHT == "2322600"
	return false
}

// 1121818 is testnet GenesisHeight
func IsTestNet() bool {
	//return MILESTONE_GENESIS_HEIGHT == "1121818"
	return false
}

func GetStartBlockHeight() int64 {
	return 0
}

func GetNodePruneHeight() int64 {
	return nodePruneHeight
}

func GetMarsHeight() int64 {
	return milestoneMarsHeight
}

// can be used in unit test only
func UnittestOnlySetMilestoneMarsHeight(height int64) {
	milestoneMarsHeight = height
}

// ==================================
// =========== Earth ===============
func UnittestOnlySetMilestoneEarthHeight(h int64) {
	milestoneEarthHeight = h
}

func SetMilestoneEarthHeight(h int64) {
	milestoneEarthHeight = h
}

func HigherThanEarth(h int64) bool {
	if milestoneEarthHeight == 0 {
		return false
	}
	return h >= milestoneEarthHeight
}

func GetEarthHeight() int64 {
	return milestoneEarthHeight
}

// =========== Earth ===============
// ==================================

// ==================================
// =========== Venus4 ===============
func HigherThanVenus4(h int64) bool {
	if milestoneVenus4Height == 0 {
		return false
	}
	return h > milestoneVenus4Height
}

func SetMilestoneVenus4Height(h int64) {
	milestoneVenus4Height = h
}

func UnittestOnlySetMilestoneVenus4Height(h int64) {
	milestoneVenus4Height = h
}

func GetVenus4Height() int64 {
	return milestoneVenus4Height
}

// =========== Venus4 ===============
// ==================================
