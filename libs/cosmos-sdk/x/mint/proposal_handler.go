package mint

import (
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/mint/internal/types"
	"github.com/okx/okbchain/x/common"
	"reflect"

	govTypes "github.com/okx/okbchain/x/gov/types"
)

const InvokeExtraProposalName = "InvokeExtraProposal"

// NewManageTreasuresProposalHandler handles "gov" type message in "mint"
func NewManageTreasuresProposalHandler(k *Keeper) govTypes.Handler {
	return func(ctx sdk.Context, proposal *govTypes.Proposal) (err sdk.Error) {
		switch content := proposal.Content.(type) {
		case types.ManageTreasuresProposal:
			return handleManageTreasuresProposal(ctx, k, proposal)
		case types.ExtraProposal:
			return handleExtraProposal(ctx, k, content)
		default:
			return common.ErrUnknownProposalType(types.DefaultCodespace, content.ProposalType())
		}
	}
}

func handleManageTreasuresProposal(ctx sdk.Context, k *Keeper, proposal *govTypes.Proposal) sdk.Error {
	// check
	manageTreasuresProposal, ok := proposal.Content.(types.ManageTreasuresProposal)
	if !ok {
		return types.ErrUnexpectedProposalType
	}

	if manageTreasuresProposal.IsAdded {
		// add/update treasures into state
		if err := k.UpdateTreasures(ctx, manageTreasuresProposal.Treasures); err != nil {
			return types.ErrTreasuresInternal(err)
		}
		return nil
	}

	// delete treasures into state
	if err := k.DeleteTreasures(ctx, manageTreasuresProposal.Treasures); err != nil {
		return types.ErrTreasuresInternal(err)
	}
	return nil
}

func handleExtraProposal(ctx sdk.Context, k *Keeper, p types.ExtraProposal) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = types.ErrHandleExtraProposal
		}
	}()

	f := reflect.ValueOf(k).MethodByName(InvokeExtraProposalName)
	result := f.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(p.Action), reflect.ValueOf(p.Extra)})
	rErr := result[0].Interface()
	err, _ = rErr.(error)
	return err
}
