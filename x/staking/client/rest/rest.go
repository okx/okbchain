package rest

import (
	"github.com/gorilla/mux"

	"github.com/okx/okbchain/libs/cosmos-sdk/client/context"
)

// RegisterRoutes registers staking-related REST handlers to a router
func RegisterRoutes(cliCtx context.CLIContext, r *mux.Router) {
	registerQueryRoutes(cliCtx, r)
	// TODO: low priority
	//registerTxRoutes(cliCtx, r)
}
