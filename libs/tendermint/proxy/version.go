package proxy

import (
	abci "github.com/okx/okbchain/libs/tendermint/abci/types"
	"github.com/okx/okbchain/libs/tendermint/version"
)

// RequestInfo contains all the information for sending
// the abci.RequestInfo message during handshake with the app.
// It contains only compile-time version information.
var RequestInfo = abci.RequestInfo{
	Version:      version.Version,
	BlockVersion: version.BlockProtocol.Uint64(),
	P2PVersion:   version.P2PProtocol.Uint64(),
}

var IBCRequestInfo = abci.RequestInfo{
	Version:      version.Version,
	BlockVersion: version.IBCBlockProtocol.Uint64(),
	P2PVersion:   version.IBCP2PProtocol.Uint64(),
}
