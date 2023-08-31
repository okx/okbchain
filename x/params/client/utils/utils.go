package utils

import (
	"encoding/json"
	"io/ioutil"

	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	"github.com/okx/brczero/libs/cosmos-sdk/types/rest"
	"github.com/okx/brczero/libs/cosmos-sdk/x/params"
)

type (
	// ParamChangesJSON defines a slice of ParamChangeJSON objects which can be
	// converted to a slice of ParamChange objects
	ParamChangesJSON []ParamChangeJSON

	// ParamChangeJSON defines a parameter change used in JSON input
	// this allows values to be specified in raw JSON instead of being string encoded
	ParamChangeJSON struct {
		Subspace string          `json:"subspace" yaml:"subspace"`
		Key      string          `json:"key" yaml:"key"`
		Value    json.RawMessage `json:"value" yaml:"value"`
	}

	// ParamChangeProposalJSON defines a ParameterChangeProposal with a deposit used to parse parameter change proposals
	// from a JSON file
	ParamChangeProposalJSON struct {
		Title       string           `json:"title" yaml:"title"`
		Description string           `json:"description" yaml:"description"`
		Changes     ParamChangesJSON `json:"changes" yaml:"changes"`
		Deposit     sdk.SysCoins     `json:"deposit" yaml:"deposit"`
		Height      uint64           `json:"height" yaml:"height"`
	}

	// ParamChangeProposalReq defines a parameter change proposal request body
	ParamChangeProposalReq struct {
		BaseReq     rest.BaseReq     `json:"base_req" yaml:"base_req"`
		Title       string           `json:"title" yaml:"title"`
		Description string           `json:"description" yaml:"description"`
		Changes     ParamChangesJSON `json:"changes" yaml:"changes"`
		Proposer    sdk.AccAddress   `json:"proposer" yaml:"proposer"`
		Deposit     sdk.SysCoins     `json:"deposit" yaml:"deposit"`
		Height      uint64           `json:"height" yaml:"height"`
	}
)

// ToParamChange converts a ParamChangeJSON object to ParamChange
func (pcj ParamChangeJSON) ToParamChange() params.ParamChange {
	return params.NewParamChange(pcj.Subspace, pcj.Key, string(pcj.Value))
}

// ToParamChanges converts a slice of ParamChangeJSON objects to a slice of ParamChange
func (pcj ParamChangesJSON) ToParamChanges() []params.ParamChange {
	res := make([]params.ParamChange, len(pcj))
	for i, pc := range pcj {
		res[i] = pc.ToParamChange()
	}
	return res
}

// ParseParamChangeProposalJSON reads and parses a ParamChangeProposalJSON from file
func ParseParamChangeProposalJSON(cdc *codec.Codec, proposalFile string) (ParamChangeProposalJSON, error) {
	var proposal ParamChangeProposalJSON

	contents, err := ioutil.ReadFile(proposalFile)
	if err != nil {
		return proposal, err
	}

	if err := cdc.UnmarshalJSON(contents, &proposal); err != nil {
		return proposal, err
	}

	return proposal, nil
}
