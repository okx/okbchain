package base

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/okx/okbchain/app/crypto/ethsecp256k1"
	ethereumTx "github.com/okx/okbchain/app/types"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/x/evm/types"
)

func Test_getSender(t *testing.T) {
	chainID := "evm-3"
	ethereumTx.SetChainId(chainID)
	privateKey, _ := ethsecp256k1.GenerateKey()
	sender := common.HexToAddress(privateKey.PubKey().Address().String())

	msg := types.NewMsgEthereumTx(0, &sender, big.NewInt(100), 3000000, big.NewInt(1), nil)

	// parse context chain ID to big.Int
	chainIDEpoch, _ := ethereumTx.ParseChainID(chainID)
	// sign transaction
	msg.Sign(chainIDEpoch, privateKey.ToECDSA())

	ctxWithFrom := sdk.Context{}
	ctxWithFrom.SetIsCheckTx(true)
	ctxWithFrom.SetFrom(sender.String())

	type args struct {
		ctx          *sdk.Context
		chainIDEpoch *big.Int
		msg          *types.MsgEthereumTx
	}
	tests := []struct {
		name       string
		args       args
		wantSender common.Address
		wantErr    bool
	}{
		{"1. get sender from verify sig", args{ctx: &sdk.Context{}, chainIDEpoch: chainIDEpoch, msg: msg}, sender, false},
		{"2. get sender from context", args{ctx: &ctxWithFrom, chainIDEpoch: chainIDEpoch, msg: msg}, sender, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSender, err := getSender(tt.args.ctx, tt.args.chainIDEpoch, tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("getSender() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotSender, tt.wantSender) {
				t.Errorf("getSender() gotSender = %v, want %v", gotSender, tt.wantSender)
			}
		})
	}
}
