package types

import (
	"fmt"
	"time"

	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	"github.com/okx/brczero/x/params/subspace"
)

// Parameter store key
var (
	ParamStoreKeyDepositParams = []byte("depositparams")
	ParamStoreKeyVotingParams  = []byte("votingparams")
	ParamStoreKeyTallyParams   = []byte("tallyparams")
)

// Key declaration for parameters
func ParamKeyTable() subspace.KeyTable {
	return subspace.NewKeyTable(
		subspace.ParamSetPairs{
			{ParamStoreKeyDepositParams, DepositParams{}, validateDepositParams},
			{ParamStoreKeyVotingParams, VotingParams{}, validateVotingParams},
			{ParamStoreKeyTallyParams, TallyParams{}, validateTallyParams},
		}...,
	)
}

// Param around deposits for governance
type DepositParams struct {
	MinDeposit       sdk.SysCoins  `json:"min_deposit,omitempty" yaml:"min_deposit,omitempty"`               //  Minimum deposit for a proposal to enter voting period.
	MaxDepositPeriod time.Duration `json:"max_deposit_period,omitempty" yaml:"max_deposit_period,omitempty"` //  Maximum period for okb holders to deposit on a proposal. Initial value: 2 months
}

// NewDepositParams creates a new DepositParams object
func NewDepositParams(minDeposit sdk.SysCoins, maxDepositPeriod time.Duration) DepositParams {
	return DepositParams{
		MinDeposit:       minDeposit,
		MaxDepositPeriod: maxDepositPeriod,
	}
}

func (dp DepositParams) String() string {
	return fmt.Sprintf(`Deposit Params:
  Min Deposit:        %s
  Max Deposit Period: %s`, dp.MinDeposit, dp.MaxDepositPeriod)
}

// Checks equality of DepositParams
func (dp DepositParams) Equal(dp2 DepositParams) bool {
	return dp.MinDeposit.IsEqual(dp2.MinDeposit) && dp.MaxDepositPeriod == dp2.MaxDepositPeriod
}

func validateDepositParams(i interface{}) error {
	v, ok := i.(DepositParams)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if !v.MinDeposit.IsValid() {
		return fmt.Errorf("invalid minimum deposit: %s", v.MinDeposit)
	}
	if v.MaxDepositPeriod <= 0 {
		return fmt.Errorf("maximum deposit period must be positive: %d", v.MaxDepositPeriod)
	}

	return nil
}

// Param around Tallying votes in governance
type TallyParams struct {
	Quorum          sdk.Dec `json:"quorum,omitempty" yaml:"quorum,omitempty"`                         //  Minimum percentage of total stake needed to vote for a result to be considered valid
	Threshold       sdk.Dec `json:"threshold,omitempty" yaml:"threshold,omitempty"`                   //  Minimum proportion of Yes votes for proposal to pass. Initial value: 0.5
	Veto            sdk.Dec `json:"veto,omitempty" yaml:"veto,omitempty"`                             //  Minimum value of Veto votes to Total votes ratio for proposal to be vetoed. Initial value: 1/3
	YesInVotePeriod sdk.Dec `json:"yes_in_vote_period,omitempty" yaml:"yes_in_vote_period,omitempty"` //
}

// NewTallyParams creates a new TallyParams object
func NewTallyParams(quorum, threshold, veto sdk.Dec) TallyParams {
	return TallyParams{
		Quorum:    quorum,
		Threshold: threshold,
		Veto:      veto,
	}
}

func (tp TallyParams) String() string {
	return fmt.Sprintf(`Tally Params:
  Quorum:             %s
  Threshold:          %s
  Veto:               %s`,
		tp.Quorum, tp.Threshold, tp.Veto)
}

func validateTallyParams(i interface{}) error {
	v, ok := i.(TallyParams)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.Quorum.IsNegative() {
		return fmt.Errorf("quorom cannot be negative: %s", v.Quorum)
	}
	if v.Quorum.GT(sdk.OneDec()) {
		return fmt.Errorf("quorom too large: %s", v)
	}
	if !v.Threshold.IsPositive() {
		return fmt.Errorf("vote threshold must be positive: %s", v.Threshold)
	}
	if v.Threshold.GT(sdk.OneDec()) {
		return fmt.Errorf("vote threshold too large: %s", v)
	}
	if !v.Veto.IsPositive() {
		return fmt.Errorf("veto threshold must be positive: %s", v.Threshold)
	}
	if v.Veto.GT(sdk.OneDec()) {
		return fmt.Errorf("veto threshold too large: %s", v)
	}

	return nil
}

// Param around Voting in governance
type VotingParams struct {
	VotingPeriod time.Duration `json:"voting_period,omitempty" yaml:"voting_period,omitempty"` //  Length of the voting period.
}

// NewVotingParams creates a new VotingParams object
func NewVotingParams(votingPeriod time.Duration) VotingParams {
	return VotingParams{
		VotingPeriod: votingPeriod,
	}
}

func (vp VotingParams) String() string {
	return fmt.Sprintf(`Voting Params:
  Voting Period:      %s`, vp.VotingPeriod)
}

func validateVotingParams(i interface{}) error {
	v, ok := i.(VotingParams)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.VotingPeriod <= 0 {
		return fmt.Errorf("voting period must be positive: %s", v.VotingPeriod)
	}

	return nil
}

// Params returns all of the governance params
type Params struct {
	VotingParams  VotingParams  `json:"voting_params" yaml:"voting_params"`
	TallyParams   TallyParams   `json:"tally_params" yaml:"tally_params"`
	DepositParams DepositParams `json:"deposit_params" yaml:"deposit_parmas"`
}

func (gp Params) String() string {
	return gp.VotingParams.String() + "\n" +
		gp.TallyParams.String() + "\n" +
		gp.DepositParams.String()
}

func NewParams(vp VotingParams, tp TallyParams, dp DepositParams) Params {
	return Params{
		VotingParams:  vp,
		DepositParams: dp,
		TallyParams:   tp,
	}
}
