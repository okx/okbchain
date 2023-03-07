package keeper

import (
	"github.com/okx/okbchain/libs/ibc-go/modules/core/02-client/types"
	"github.com/okx/okbchain/libs/ibc-go/modules/core/exported"
)

// MustUnmarshalClientState attempts to decode and return an ClientState object from
// raw encoded bytes. It panics on error.
func (k Keeper) MustUnmarshalClientState(bz []byte) exported.ClientState {
	return types.MustUnmarshalClientState(k.cdc, bz)
}

// UnmarshalClientState attempts to decode and return an ClientState object from
// raw encoded bytes.
func (k Keeper) UnmarshalClientState(bz []byte) (exported.ClientState, error) {
	return types.UnmarshalClientState(k.cdc, bz)
}

// UnmarshalConsensusState attempts to decode and return an ConsensusState object from
// raw encoded bytes.
func (k Keeper) UnmarshalConsensusState(bz []byte) (exported.ConsensusState, error) {
	return types.UnmarshalConsensusState(k.cdc, bz)
}

// MustMarshalClientState attempts to encode an ClientState object and returns the
// raw encoded bytes. It panics on error.
func (k Keeper) MustMarshalClientState(clientState exported.ClientState) []byte {
	return types.MustMarshalClientState(k.cdc, clientState)
}

// MustUnmarshalConsensusState attempts to decode and return an ConsensusState object from
// raw encoded bytes. It panics on error.
func (k Keeper) MustUnmarshalConsensusState(bz []byte) exported.ConsensusState {
	return types.MustUnmarshalConsensusState(k.cdc, bz)
}

// MustMarshalConsensusState attempts to encode an ConsensusState object and returns the
// raw encoded bytes. It panics on error.
func (k Keeper) MustMarshalConsensusState(consensusState exported.ConsensusState) []byte {
	return types.MustMarshalConsensusState(k.cdc, consensusState)
}
