package simulation

import (
	"bytes"
	"fmt"

	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	"github.com/okx/brczero/libs/ibc-go/modules/core/03-connection/types"
	host "github.com/okx/brczero/libs/ibc-go/modules/core/24-host"
	"github.com/okx/brczero/libs/ibc-go/modules/core/common"
	"github.com/okx/brczero/libs/tendermint/libs/kv"
)

// NewDecodeStore returns a decoder function closure that unmarshals the KVPair's
// Value to the corresponding connection type.
func NewDecodeStore(cdc *codec.CodecProxy, kvA, kvB kv.Pair) (string, bool) {
	switch {
	case bytes.HasPrefix(kvA.Key, host.KeyClientStorePrefix) && bytes.HasSuffix(kvA.Key, []byte(host.KeyConnectionPrefix)):
		var clientConnectionsA, clientConnectionsB types.ClientPaths
		cdc.GetProtocMarshal().MustUnmarshalBinaryBare(kvA.Value, &clientConnectionsA)
		cdc.GetProtocMarshal().MustUnmarshalBinaryBare(kvB.Value, &clientConnectionsB)
		return fmt.Sprintf("ClientPaths A: %v\nClientPaths B: %v", clientConnectionsA, clientConnectionsB), true

	case bytes.HasPrefix(kvA.Key, []byte(host.KeyConnectionPrefix)):
		var connectionA, connectionB types.ConnectionEnd
		connectionA = *common.MustUnmarshalConnection(cdc, kvA.Value)
		connectionB = *common.MustUnmarshalConnection(cdc, kvB.Value)
		return fmt.Sprintf("ConnectionEnd A: %v\nConnectionEnd B: %v", connectionA, connectionB), true

	default:
		return "", false
	}
}
