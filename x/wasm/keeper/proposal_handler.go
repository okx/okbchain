package keeper

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/okbchain/libs/cosmos-sdk/types/errors"
	types2 "github.com/okx/okbchain/libs/tendermint/types"
	govtypes "github.com/okx/okbchain/x/gov/types"
	"github.com/okx/okbchain/x/wasm/types"
)

// NewWasmProposalHandler creates a new governance Handler for wasm proposals
func NewWasmProposalHandler(k decoratedKeeper, enabledProposalTypes []types.ProposalType) govtypes.Handler {
	return NewWasmProposalHandlerX(NewGovPermissionKeeper(k), enabledProposalTypes)
}

// NewWasmProposalHandlerX creates a new governance Handler for wasm proposals
func NewWasmProposalHandlerX(k types.ContractOpsKeeper, enabledProposalTypes []types.ProposalType) govtypes.Handler {
	enabledTypes := make(map[string]struct{}, len(enabledProposalTypes))
	for i := range enabledProposalTypes {
		enabledTypes[string(enabledProposalTypes[i])] = struct{}{}
	}
	return func(ctx sdk.Context, proposal *govtypes.Proposal) sdk.Error {
		if !types2.HigherThanEarth(ctx.BlockHeight()) {
			errMsg := fmt.Sprintf("wasm not supprt at height %d", ctx.BlockHeight())
			return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
		}
		content := proposal.Content
		if content == nil {
			return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "content must not be empty")
		}
		if _, ok := enabledTypes[content.ProposalType()]; !ok {
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unsupported wasm proposal content type: %q", content.ProposalType())
		}
		switch c := content.(type) {
		case *types.StoreCodeProposal:
			return handleStoreCodeProposal(ctx, k, *c)
		case *types.InstantiateContractProposal:
			return handleInstantiateProposal(ctx, k, *c)
		case *types.MigrateContractProposal:
			return handleMigrateProposal(ctx, k, *c)
		case *types.SudoContractProposal:
			return handleSudoProposal(ctx, k, *c)
		case *types.ExecuteContractProposal:
			return handleExecuteProposal(ctx, k, *c)
		case *types.UpdateAdminProposal:
			return handleUpdateAdminProposal(ctx, k, *c)
		case *types.ClearAdminProposal:
			return handleClearAdminProposal(ctx, k, *c)
		case *types.PinCodesProposal:
			return handlePinCodesProposal(ctx, k, *c)
		case *types.UnpinCodesProposal:
			return handleUnpinCodesProposal(ctx, k, *c)
		case *types.UpdateInstantiateConfigProposal:
			return handleUpdateInstantiateConfigProposal(ctx, k, *c)
		case *types.UpdateDeploymentWhitelistProposal:
			return handleUpdateDeploymentWhitelistProposal(ctx, k, *c)
		case *types.ExtraProposal:
			return handleExtraProposal(ctx, k, *c)
		case *types.UpdateWASMContractMethodBlockedListProposal:
			return handleUpdateWASMContractMethodBlockedListProposal(ctx, k, *c)
		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized wasm proposal content type: %T", c)
		}
	}
}

func handleStoreCodeProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.StoreCodeProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	runAsAddr, err := sdk.WasmAddressFromBech32(p.RunAs)
	if err != nil {
		return sdkerrors.Wrap(err, "run as address")
	}
	result, err := types.ConvertAccessConfig(*p.InstantiatePermission)
	if err != nil {
		return err
	}
	codeID, err := k.Create(ctx, runAsAddr, p.WASMByteCode, &result)
	if err != nil {
		return err
	}
	return k.PinCode(ctx, codeID)
}

func handleInstantiateProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.InstantiateContractProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}
	runAsAddr, err := sdk.WasmAddressFromBech32(p.RunAs)
	if err != nil {
		return sdkerrors.Wrap(err, "run as address")
	}
	var adminAddr sdk.WasmAddress
	if p.Admin != "" {
		if adminAddr, err = sdk.WasmAddressFromBech32(p.Admin); err != nil {
			return sdkerrors.Wrap(err, "admin")
		}
	}

	_, data, err := k.Instantiate(ctx, p.CodeID, runAsAddr, adminAddr, p.Msg, p.Label, sdk.CoinAdaptersToCoins(p.Funds))
	if err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeGovContractResult,
		sdk.NewAttribute(types.AttributeKeyResultDataHex, hex.EncodeToString(data)),
	))
	return nil
}

func handleMigrateProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.MigrateContractProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	contractAddr, err := sdk.WasmAddressFromBech32(p.Contract)
	if err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	if err != nil {
		return sdkerrors.Wrap(err, "run as address")
	}

	if err = k.ClearContractAdmin(ctx, contractAddr, contractAddr); err != nil {
		return err
	}

	// runAs is not used if this is permissioned, so just put any valid address there (second contractAddr)
	data, err := k.Migrate(ctx, contractAddr, contractAddr, p.CodeID, p.Msg)
	if err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeGovContractResult,
		sdk.NewAttribute(types.AttributeKeyResultDataHex, hex.EncodeToString(data)),
	))
	return nil
}

func handleSudoProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.SudoContractProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	contractAddr, err := sdk.WasmAddressFromBech32(p.Contract)
	if err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	data, err := k.Sudo(ctx, contractAddr, p.Msg)
	if err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeGovContractResult,
		sdk.NewAttribute(types.AttributeKeyResultDataHex, hex.EncodeToString(data)),
	))
	return nil
}

func handleExecuteProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.ExecuteContractProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	contractAddr, err := sdk.WasmAddressFromBech32(p.Contract)
	if err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	runAsAddr, err := sdk.WasmAddressFromBech32(p.RunAs)
	if err != nil {
		return sdkerrors.Wrap(err, "run as address")
	}
	data, err := k.Execute(ctx, contractAddr, runAsAddr, p.Msg, sdk.CoinAdaptersToCoins(p.Funds))
	if err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeGovContractResult,
		sdk.NewAttribute(types.AttributeKeyResultDataHex, hex.EncodeToString(data)),
	))
	return nil
}

func handleUpdateAdminProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.UpdateAdminProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}
	contractAddr, err := sdk.WasmAddressFromBech32(p.Contract)
	if err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	newAdminAddr, err := sdk.WasmAddressFromBech32(p.NewAdmin)
	if err != nil {
		return sdkerrors.Wrap(err, "run as address")
	}

	return k.UpdateContractAdmin(ctx, contractAddr, nil, newAdminAddr)
}

func handleClearAdminProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.ClearAdminProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	contractAddr, err := sdk.WasmAddressFromBech32(p.Contract)
	if err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	if err := k.ClearContractAdmin(ctx, contractAddr, nil); err != nil {
		return err
	}
	return nil
}

func handlePinCodesProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.PinCodesProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}
	for _, v := range p.CodeIDs {
		if err := k.PinCode(ctx, v); err != nil {
			return sdkerrors.Wrapf(err, "code id: %d", v)
		}
	}
	return nil
}

func handleUnpinCodesProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.UnpinCodesProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}
	for _, v := range p.CodeIDs {
		if err := k.UnpinCode(ctx, v); err != nil {
			return sdkerrors.Wrapf(err, "code id: %d", v)
		}
	}
	return nil
}

func handleUpdateInstantiateConfigProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.UpdateInstantiateConfigProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	for _, accessConfigUpdate := range p.AccessConfigUpdates {
		result, err := types.ConvertAccessConfig(accessConfigUpdate.InstantiatePermission)
		if err != nil {
			return err
		}

		if err := k.SetAccessConfig(ctx, accessConfigUpdate.CodeID, result); err != nil {
			return sdkerrors.Wrapf(err, "code id: %d", accessConfigUpdate.CodeID)
		}
	}
	return nil
}

func handleUpdateDeploymentWhitelistProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.UpdateDeploymentWhitelistProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	var config types.AccessConfig
	if types.IsNobody(p.DistributorAddrs) {
		config.Permission = types.AccessTypeNobody
	} else if types.IsAllAddress(p.DistributorAddrs) {
		config.Permission = types.AccessTypeEverybody
	} else {
		sort.Strings(p.DistributorAddrs)
		config.Permission = types.AccessTypeOnlyAddress
		config.Address = strings.Join(p.DistributorAddrs, ",")
	}
	result, err := types.ConvertAccessConfig(config)
	if err != nil {
		return err
	}
	k.UpdateUploadAccessConfig(ctx, result)
	return nil
}

func handleExtraProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.ExtraProposal) (err error) {
	return k.InvokeExtraProposal(ctx, p.Action, p.Extra)
}

func handleUpdateWASMContractMethodBlockedListProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.UpdateWASMContractMethodBlockedListProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}
	contractAddr, err := sdk.WasmAddressFromBech32(p.BlockedMethods.ContractAddr)
	if err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	if err = k.ClearContractAdmin(ctx, contractAddr, contractAddr); err != nil {
		return err
	}
	p.BlockedMethods.ContractAddr = contractAddr.String()
	return k.UpdateContractMethodBlockedList(ctx, p.BlockedMethods, p.IsDelete)
}
