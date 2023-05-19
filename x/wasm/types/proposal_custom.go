package types

import (
	"encoding/json"
	"fmt"
	"strings"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	govtypes "github.com/okx/okbchain/x/gov/types"
)

const (
	maxAddressListLength       = 100
	maxMethodListLength        = 100
	MaxGasFactor         int64 = 10000000
)

// ProposalRoute returns the routing key of a parameter change proposal.
func (p UpdateDeploymentWhitelistProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns the type
func (p UpdateDeploymentWhitelistProposal) ProposalType() string {
	return string(ProposalTypeUpdateDeploymentWhitelist)
}

// ValidateBasic validates the proposal
func (p UpdateDeploymentWhitelistProposal) ValidateBasic() error {
	if err := validateProposalCommons(p.Title, p.Description); err != nil {
		return err
	}
	l := len(p.DistributorAddrs)
	if l == 0 || l > maxAddressListLength {
		return fmt.Errorf("invalid distributor addresses len: %d", l)
	}
	return validateDistributorAddrs(p.DistributorAddrs)
}

// MarshalYAML pretty prints the wasm byte code
func (p UpdateDeploymentWhitelistProposal) MarshalYAML() (interface{}, error) {
	return struct {
		Title            string   `yaml:"title"`
		Description      string   `yaml:"description"`
		DistributorAddrs []string `yaml:"distributor_addresses"`
	}{
		Title:            p.Title,
		Description:      p.Description,
		DistributorAddrs: p.DistributorAddrs,
	}, nil
}

func validateDistributorAddrs(addrs []string) error {
	if IsNobody(addrs) {
		return nil
	}
	if IsAllAddress(addrs) {
		return nil
	}
	for _, addr := range addrs {
		if _, err := sdk.WasmAddressFromBech32(addr); err != nil {
			return err
		}
	}
	return nil
}

func IsNobody(addrs []string) bool {
	if len(addrs) == 1 && addrs[0] == "nobody" {
		return true
	}
	return false
}

func IsAllAddress(addrs []string) bool {
	return len(addrs) == 1 && addrs[0] == "all"
}

// ProposalRoute returns the routing key of a parameter change proposal.
func (p UpdateWASMContractMethodBlockedListProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns the type
func (p UpdateWASMContractMethodBlockedListProposal) ProposalType() string {
	return string(ProposalTypeUpdateWasmContractMethodBlockedList)
}

// ValidateBasic validates the proposal
func (p UpdateWASMContractMethodBlockedListProposal) ValidateBasic() error {
	if err := validateProposalCommons(p.Title, p.Description); err != nil {
		return err
	}
	return validateContractMethods(p.BlockedMethods)
}

func validateContractMethods(methods *ContractMethods) error {
	l := len(methods.Methods)
	if l == 0 || l > maxMethodListLength {
		return fmt.Errorf("invalid contract methods len: %d", l)
	}
	if _, err := sdk.WasmAddressFromBech32(methods.ContractAddr); err != nil {
		return err
	}
	return nil
}

// MarshalYAML pretty prints the wasm byte code
func (p UpdateWASMContractMethodBlockedListProposal) MarshalYAML() (interface{}, error) {
	var methods []string
	for _, method := range p.BlockedMethods.Methods {
		methods = append(methods, method.FullName())
	}
	return struct {
		Title        string   `yaml:"title"`
		Description  string   `yaml:"description"`
		ContractAddr string   `yaml:"contract_address"`
		Methods      []string `yaml:"methods"`
		IsDelete     bool     `yaml:"is_delete"`
	}{
		Title:        p.Title,
		Description:  p.Description,
		ContractAddr: p.BlockedMethods.ContractAddr,
		Methods:      methods,
		IsDelete:     p.IsDelete,
	}, nil
}

func (c *Method) FullName() string {
	if len(c.Extra) == 0 {
		return c.Name
	}
	return c.Name + " " + c.Extra
}

func (c *ContractMethods) DeleteMethods(methods []*Method) {
	for _, method := range methods {
		for i := range c.Methods {
			if c.Methods[i].FullName() == method.FullName() {
				//delete method
				c.Methods = append(c.Methods[:i], c.Methods[i+1:]...)
				break
			}
		}
	}
}

func (c *ContractMethods) AddMethods(methods []*Method) {
	for _, method := range methods {
		var exist bool
		for i := range c.Methods {
			if c.Methods[i].FullName() == method.FullName() {
				exist = true
				break
			}
		}
		if exist {
			exist = false
		} else {
			c.Methods = append(c.Methods, method)
		}
	}
}

func (c *ContractMethods) IsMethodBlocked(method string) bool {
	if c == nil {
		return false
	}
	for _, m := range c.Methods {
		if m.FullName() == method {
			return true
		}
	}
	return false
}

func FindContractMethods(cms []*ContractMethods, contractAddr string) *ContractMethods {
	for _, cm := range cms {
		if cm.ContractAddr == contractAddr {
			return cm
		}
	}
	return nil
}

var _ govtypes.Content = &ExtraProposal{}

// ProposalRoute returns the routing key of a parameter change proposal.
func (p ExtraProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns the type
func (p ExtraProposal) ProposalType() string {
	return string(ProposalTypeExtra)
}

// ValidateBasic validates the proposal
func (p ExtraProposal) ValidateBasic() error {
	if err := validateProposalCommons(p.Title, p.Description); err != nil {
		return err
	}

	if len(strings.TrimSpace(p.Action)) == 0 {
		return govtypes.ErrInvalidProposalContent("extra proposal's action is required")
	}
	if len(p.Action) > govtypes.MaxExtraActionLength {
		return govtypes.ErrInvalidProposalContent("extra proposal's action length is bigger than max length")
	}
	if len(strings.TrimSpace(p.Extra)) == 0 {
		return govtypes.ErrInvalidProposalContent("extra proposal's extra is required")
	}
	if len(p.Extra) > govtypes.MaxExtraBodyLength {
		return govtypes.ErrInvalidProposalContent("extra proposal's extra body length is bigger than max length")
	}
	switch p.Action {
	case ActionModifyGasFactor:
		_, err := NewActionModifyGasFactor(p.Extra)
		return err
	default:
		return ErrUnknownExtraProposalAction
	}
}

type GasFactor struct {
	Factor string `json:"factor" yaml:"factor"`
}

func NewActionModifyGasFactor(data string) (sdk.Dec, error) {
	var param GasFactor
	err := json.Unmarshal([]byte(data), &param)
	if err != nil {
		return sdk.Dec{}, ErrExtraProposalParams(fmt.Sprintf("parse json error, expect like {\"factor\":\"14\"}, but get:%s", data))
	}

	result, err := sdk.NewDecFromStr(param.Factor)
	if err != nil {
		return sdk.Dec{}, ErrExtraProposalParams(fmt.Sprintf("parse factor error, %s", err.Error()))
	}

	if result.IsNil() || result.IsNegative() || result.IsZero() {
		return sdk.Dec{}, ErrExtraProposalParams(fmt.Sprintf("parse factor error, expect factor positive and 18 precision, but get %s", param.Factor))
	}

	if result.GT(sdk.NewDec(MaxGasFactor)) {
		return sdk.Dec{}, ErrExtraProposalParams(fmt.Sprintf("max gas factor:%v, but get:%s", MaxGasFactor, param.Factor))
	}

	return result, nil
}

// MarshalYAML pretty prints the wasm byte code
func (p ExtraProposal) MarshalYAML() (interface{}, error) {
	return struct {
		Title       string `yaml:"title"`
		Description string `yaml:"description"`
		Action      string `yaml:"action"`
		Extra       string `yaml:"extra"`
	}{
		Title:       p.Title,
		Description: p.Description,
		Action:      p.Action,
		Extra:       p.Extra,
	}, nil
}
