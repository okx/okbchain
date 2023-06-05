package types

import (
	"github.com/ethereum/go-ethereum/common"
	stypes "github.com/okx/okbchain/libs/cosmos-sdk/store/types"

	abci "github.com/okx/okbchain/libs/tendermint/abci/types"
)

// Handler defines the core of the state transition function of an application.
type Handler func(ctx Context, msg Msg) (*Result, error)

// AnteHandler authenticates transactions, before their internal messages are handled.
// If newCtx.IsZero(), ctx is used instead.
type AnteHandler func(ctx Context, tx Tx, simulate bool) (newCtx Context, err error)

type PreDeliverTxHandler func(ctx Context, tx Tx, onlyVerifySig bool)

type GasRefundHandler func(ctx Context, tx Tx) (fee Coins, err error)

type AccNonceHandler func(ctx Context, address AccAddress) (nonce uint64)

type EvmSysContractAddressHandler func(ctx Context, addr AccAddress) bool

type UpdateCMTxNonceHandler func(tx Tx, nonce uint64)

type UpdateFeeCollectorAccHandler func(ctx Context, balance Coins, txFeesplit []*FeeSplitInfo) error

type GetGasConfigHandler func(ctx Context) *stypes.GasConfig

type UpdateCosmosTxCount func(ctx Context, txCount int)

type LogFix func(tx []Tx, logIndex []int, hasEnterEvmTx []bool, errs []error, resp []abci.ResponseDeliverTx) (logs [][]byte)
type UpdateFeeSplitHandler func(txHash common.Hash, addr AccAddress, fee Coins, isDelete bool)
type GetTxFeeAndFromHandler func(ctx Context, tx Tx) (Coins, bool, string, string, error, bool)
type GetTxFeeHandler func(tx Tx) Coins

type CustomizeOnStop func(ctx Context) error

type MptCommitHandler func(ctx Context)

type EvmWatcherCollector func(...IWatcher)

// AnteDecorator wraps the next AnteHandler to perform custom pre- and post-processing.
type AnteDecorator interface {
	AnteHandle(ctx Context, tx Tx, simulate bool, next AnteHandler) (newCtx Context, err error)
}

// ChainDecorator chains AnteDecorators together with each AnteDecorator
// wrapping over the decorators further along chain and returns a single AnteHandler.
//
// NOTE: The first element is outermost decorator, while the last element is innermost
// decorator. Decorator ordering is critical since some decorators will expect
// certain checks and updates to be performed (e.g. the Context) before the decorator
// is run. These expectations should be documented clearly in a CONTRACT docline
// in the decorator's godoc.
//
// NOTE: Any application that uses GasMeter to limit transaction processing cost
// MUST set GasMeter with the FIRST AnteDecorator. Failing to do so will cause
// transactions to be processed with an infinite gasmeter and open a DOS attack vector.
// Use `ante.SetUpContextDecorator` or a custom Decorator with similar functionality.
// Returns nil when no AnteDecorator are supplied.
func ChainAnteDecorators(chain ...AnteDecorator) AnteHandler {
	if len(chain) == 0 {
		return nil
	}

	// handle non-terminated decorators chain
	if (chain[len(chain)-1] != Terminator{}) {
		chain = append(chain, Terminator{})
	}

	next := ChainAnteDecorators(chain[1:]...)

	return func(ctx Context, tx Tx, simulate bool) (Context, error) {
		return chain[0].AnteHandle(ctx, tx, simulate, next)
	}
}

// Terminator AnteDecorator will get added to the chain to simplify decorator code
// Don't need to check if next == nil further up the chain
//
//	                      ______
//	                   <((((((\\\
//	                   /      . }\
//	                   ;--..--._|}
//	(\                 '--/\--'  )
//	 \\                | '-'  :'|
//	  \\               . -==- .-|
//	   \\               \.__.'   \--._
//	   [\\          __.--|       //  _/'--.
//	   \ \\       .'-._ ('-----'/ __/      \
//	    \ \\     /   __>|      | '--.       |
//	     \ \\   |   \   |     /    /       /
//	      \ '\ /     \  |     |  _/       /
//	       \  \       \ |     | /        /
//	 snd    \  \      \        /
type Terminator struct{}

const AnteTerminatorTag = "ante-terminator"

// Simply return provided Context and nil error
func (t Terminator) AnteHandle(ctx Context, _ Tx, _ bool, _ AnteHandler) (Context, error) {
	trc := ctx.AnteTracer()
	if trc != nil {
		trc.RepeatingPin(AnteTerminatorTag)
	}
	return ctx, nil
}
