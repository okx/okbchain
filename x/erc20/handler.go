package erc20

import (
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/okbchain/libs/cosmos-sdk/types/errors"
)

// NewHandler returns a handler for erc20 type messages.
func NewHandler(k Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) (result *sdk.Result, err error) {
		ctx.SetEventManager(sdk.NewEventManager())

		switch msg := msg.(type) {
		default:
			return nil, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized %s message type: %T", ModuleName, msg)
		}
	}
}
