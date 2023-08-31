package types

import (
	"github.com/okx/brczero/libs/system"
)

// Disable followings after milestoneMercuryHeight
// 1. TransferToContractBlock
// 2. ChangeEvmDenomByProposal
// 3. BankTransferBlock
// 4. ibc

var (
	milestoneEarthHeight   int64
	milestoneVenus4Height  int64
	milestoneMercuryHeight int64
	milestoneVenus7Height  int64

	// note: it stores the earlies height of the node,and it is used by cli
	nodePruneHeight int64
)

const (
	MainNet = system.Chain + "-196"
	TestNet = system.TestnetPrefix + "-195"

	MILESTONE_EARTH   = "earth"
	MILESTONE_Venus4  = "venus4"
	MILESTONE_MERCURY = "mercury"

	MILESTONE_VENUS7_NAME = "venus7"
)

func SetupMainNetEnvironment(pruneH int64) {
	nodePruneHeight = pruneH
}

func SetupTestNetEnvironment(pruneH int64) {
	nodePruneHeight = pruneH
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

func IsPrivateNet() bool {
	return !IsMainNet() && !IsTestNet()
}

func GetStartBlockHeight() int64 {
	return 0
}

func GetNodePruneHeight() int64 {
	return nodePruneHeight
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

func InitMilestoneEarthHeight(h int64) {
	milestoneEarthHeight = h
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

// ==================================
// =========== Mercury ===============
func UnittestOnlySetMilestoneMercuryHeight(h int64) {
	milestoneEarthHeight = h
}

func SetMilestoneMercuryHeight(h int64) {
	milestoneMercuryHeight = h
}

func HigherThanMercury(h int64) bool {
	if milestoneMercuryHeight == 0 {
		return false
	}
	return h >= milestoneMercuryHeight
}

func GetMercuryHeight() int64 {
	return milestoneMercuryHeight
}

func InitMilestoneMercuryHeight(h int64) {
	milestoneMercuryHeight = h
}

// =========== Mercury ===============
// ==================================

// ==================================
// =========== Venus7 ===============
func HigherThanVenus7(h int64) bool {
	if milestoneVenus7Height == 0 {
		return false
	}
	return h > milestoneVenus7Height
}

func InitMilestoneVenus7Height(h int64) {
	milestoneVenus7Height = h
}

func GetVenus7Height() int64 {
	return milestoneVenus7Height
}

// =========== Venus7 ===============
// ==================================
