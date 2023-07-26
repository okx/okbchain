package keeper

import (
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	authexported "github.com/okx/okbchain/libs/cosmos-sdk/x/auth/exported"
	evmtypes "github.com/okx/okbchain/x/evm/types"
	wasmtypes "github.com/okx/okbchain/x/wasm/types"
)

type EVMKeeper interface {
	GetChainConfig(ctx sdk.Context) (evmtypes.ChainConfig, bool)
	GenerateCSDBParams() evmtypes.CommitStateDBParams
	GetParams(ctx sdk.Context) evmtypes.Params
	GetCallToCM() vm.CallToWasmByPrecompile
	GetBlockHash() ethcmn.Hash
	AddInnerTx(...interface{})
	AddContract(...interface{})
}

type WASMKeeper interface {
	// Execute executes the contract instance
	Execute(ctx sdk.Context, contractAddress sdk.WasmAddress, caller sdk.WasmAddress, msg []byte, coins sdk.Coins) ([]byte, error)
	GetParams(ctx sdk.Context) wasmtypes.Params
	NewQueryHandler(ctx sdk.Context, contractAddress sdk.WasmAddress) wasmvmtypes.Querier
	RuntimeGasForContract(ctx sdk.Context) uint64
}

// AccountKeeper defines the expected account keeper interface
type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) authexported.Account
	SetAccount(ctx sdk.Context, acc authexported.Account)
	NewAccountWithAddress(ctx sdk.Context, addr sdk.AccAddress) authexported.Account
}

type BankKeeper interface {
	SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
}
