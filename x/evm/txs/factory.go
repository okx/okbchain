package txs

import (
	"fmt"
	"github.com/okx/okbchain/x/evm/txs/base"
	"github.com/okx/okbchain/x/evm/txs/check"
	"github.com/okx/okbchain/x/evm/txs/deliver"
	"github.com/okx/okbchain/x/evm/txs/tracetxlog"
)

type factory struct {
	base.Config
}

func NewFactory(config base.Config) *factory {
	return &factory{config}
}

func (factory *factory) CreateTx() (Tx, error) {
	if factory == nil || factory.Keeper == nil {
		return nil, fmt.Errorf("evm txs factory not inited")
	}
	if factory.Ctx.IsTraceTxLog() {
		return tracetxlog.NewTx(factory.Config), nil
	} else if factory.Ctx.IsCheckTx() {
		return check.NewTx(factory.Config), nil
	}

	return deliver.NewTx(factory.Config), nil
}
