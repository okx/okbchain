package types

var (
	// MinterKey is used for the keeper store
	MinterKey = []byte{0x00}
	// TreasuresKey is used for the keeper store
	TreasuresKey = []byte{0x01}
)

// nolint
const (
	// ModuleName
	ModuleName = "mint"

	// DefaultParamspace params keeper
	DefaultParamspace = ModuleName

	// StoreKey is the default store key for mint
	StoreKey = ModuleName

	// QuerierRoute is the querier route for the minting store.
	QuerierRoute = StoreKey

	// Query endpoints supported by the minting querier
	QueryParameters       = "parameters"
	QueryInflation        = "inflation"
	QueryAnnualProvisions = "annual_provisions"
	QueryTreasures        = "treasures"
	QueryBlockRewards     = "block_rewards"
)
