package types

import (
	"time"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	stakingexported "github.com/okx/okbchain/x/staking/exported"

	"github.com/okx/okbchain/libs/tendermint/crypto"
)

type (
	// StakingKeeper defines the staking module interface contract needed by the
	// evidence module.
	StakingKeeper interface {
		ValidatorByConsAddr(sdk.Context, sdk.ConsAddress) stakingexported.ValidatorI

		//Append abandoned validator to staking keeper
		AppendAbandonedValidatorAddrs(ctx sdk.Context, ConsAddr sdk.ConsAddress)
	}

	// SlashingKeeper defines the slashing module interface contract needed by the
	// evidence module.
	SlashingKeeper interface {
		GetPubkey(sdk.Context, crypto.Address) (crypto.PubKey, error)
		IsTombstoned(sdk.Context, sdk.ConsAddress) bool
		HasValidatorSigningInfo(sdk.Context, sdk.ConsAddress) bool
		Tombstone(sdk.Context, sdk.ConsAddress)
		Slash(sdk.Context, sdk.ConsAddress, sdk.Dec, int64, int64)
		SlashFractionDoubleSign(sdk.Context) sdk.Dec
		Jail(sdk.Context, sdk.ConsAddress)
		JailUntil(sdk.Context, sdk.ConsAddress, time.Time)
	}
)
