package types

import (
	"errors"
	"fmt"

	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/brczero/libs/cosmos-sdk/types/errors"
)

const (
	DefaultCodespace string = ModuleName
)

var (
	// ErrChainConfigNotFound returns an error if the chain config cannot be found on the store.
	ErrChainConfigNotFound = sdkerrors.Register(DefaultCodespace, 1, "chain configuration not found")
	// ErrKeyNotFound returns an error if the target key not found in database.
	ErrKeyNotFound = sdkerrors.Register(DefaultCodespace, 2, "Key not found in database")
	// ErrUnexpectedProposalType returns an error when the proposal type is not supported in erc20 module
	ErrUnexpectedProposalType = sdkerrors.Register(DefaultCodespace, 3, "Unsupported proposal type of erc20 module")
	// ErrEmptyAddressList returns an error if the address list is empty
	ErrEmptyAddressList   = sdkerrors.Register(DefaultCodespace, 4, "Empty account address list")
	ErrIbcDenomInvalid    = sdkerrors.Register(DefaultCodespace, 5, "ibc denom is invalid")
	ErrNoContractDeployed = sdkerrors.Register(DefaultCodespace, 6, "no contract deployed")

	ErrNoContractNotAuto = errors.New("no contract found and not auto deploy for the denom ")
)

func ErrRegisteredContract(contract string) sdk.EnvelopedErr {
	return sdk.EnvelopedErr{
		Err: sdkerrors.New(
			DefaultCodespace,
			21,
			fmt.Sprintf("the contract is already registered: %s", contract),
		),
	}
}

func ErrProxyContractRedirect(denom string, tp int, addr string) sdk.EnvelopedErr {
	return sdk.EnvelopedErr{
		Err: sdkerrors.New(
			DefaultCodespace,
			22,
			fmt.Sprintf("proxy contract redirect error: denom:%s,tp:%d,addr:%s", denom, tp, addr),
		),
	}
}
