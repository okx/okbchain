package types

import (
	"time"

	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/okx/brczero/libs/tendermint/crypto"
)

type Account interface {
	Copy() Account
	GetAddress() AccAddress
	SetAddress(AccAddress) error
	GetPubKey() crypto.PubKey
	SetPubKey(crypto.PubKey) error
	GetAccountNumber() uint64
	SetAccountNumber(uint64) error
	GetSequence() uint64
	SetSequence(uint64) error
	GetCoins() Coins
	SetCoins(Coins) error
	SpendableCoins(blockTime time.Time) Coins
	String() string
	GetStateRoot() ethcmn.Hash
	GetCodeHash() []byte
}

type ModuleAccount interface {
	Account

	GetName() string
	GetPermissions() []string
	HasPermission(string) bool
}
