package watcher

import (
	cryptocodec "github.com/okx/brczero/app/crypto/ethsecp256k1"
	app "github.com/okx/brczero/app/types"
	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	"github.com/okx/brczero/libs/cosmos-sdk/x/auth/exported"
)

var WatchCdc *codec.Codec

func init() {
	WatchCdc = codec.New()
	cryptocodec.RegisterCodec(WatchCdc)
	codec.RegisterCrypto(WatchCdc)
	WatchCdc.RegisterInterface((*exported.Account)(nil), nil)
	app.RegisterCodec(WatchCdc)
}

func EncodeAccount(acc *app.EthAccount) ([]byte, error) {
	bz, err := WatchCdc.MarshalBinaryWithSizer(acc, false)
	if err != nil {
		return nil, err
	}
	return bz, nil
}

func DecodeAccount(bz []byte) (*app.EthAccount, error) {
	var acc app.EthAccount
	val, err := WatchCdc.UnmarshalBinaryBareWithRegisteredUnmarshaller(bz, &acc)
	if err != nil {
		return nil, err
	}
	return val.(*app.EthAccount), nil
}
