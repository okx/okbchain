package types

import (
	"bytes"
	"errors"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"time"

	"github.com/tendermint/go-amino"

	"github.com/okx/okbchain/libs/tendermint/crypto"
	"gopkg.in/yaml.v2"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth/exported"
	cryptoamino "github.com/okx/okbchain/libs/tendermint/crypto/encoding/amino"
)

//-----------------------------------------------------------------------------
// BaseAccount

var _ exported.Account = (*BaseAccount)(nil)
var _ exported.GenesisAccount = (*BaseAccount)(nil)

// BaseAccount - a base account structure.
// This can be extended by embedding within in your AppAccount.
// However one doesn't have to use BaseAccount as long as your struct
// implements Account.
type BaseAccount struct {
	Address       sdk.AccAddress `json:"address" yaml:"address"`
	Coins         sdk.Coins      `json:"coins" yaml:"coins"`
	PubKey        crypto.PubKey  `json:"public_key" yaml:"public_key"`
	AccountNumber uint64         `json:"account_number" yaml:"account_number"`
	Sequence      uint64         `json:"sequence" yaml:"sequence"`
}

func (acc *BaseAccount) UnmarshalFromAmino(cdc *amino.Codec, data []byte) error {
	var dataLen uint64 = 0
	var subData []byte

	for {
		data = data[dataLen:]

		if len(data) == 0 {
			break
		}

		pos, aminoType, err := amino.ParseProtoPosAndTypeMustOneByte(data[0])
		if err != nil {
			return err
		}
		data = data[1:]

		if aminoType == amino.Typ3_ByteLength {
			var n int
			dataLen, n, err = amino.DecodeUvarint(data)
			if err != nil {
				return err
			}
			data = data[n:]
			if int(dataLen) > len(data) {
				return errors.New("not enough data")
			}
			subData = data[:dataLen]
		}

		switch pos {
		case 1:
			acc.Address = make([]byte, len(subData))
			copy(acc.Address, subData)
		case 2:
			var coin sdk.DecCoin
			err = coin.UnmarshalFromAmino(cdc, subData)
			if err != nil {
				return err
			}
			acc.Coins = append(acc.Coins, coin)
		case 3:
			acc.PubKey, err = cryptoamino.UnmarshalPubKeyFromAmino(cdc, subData)
			if err != nil {
				return err
			}
		case 4:
			var n int
			acc.AccountNumber, n, err = amino.DecodeUvarint(data)
			if err != nil {
				return err
			}
			dataLen = uint64(n)
		case 5:
			var n int
			acc.Sequence, n, err = amino.DecodeUvarint(data)
			if err != nil {
				return err
			}
			dataLen = uint64(n)
		}
	}
	return nil
}

func (acc BaseAccount) Copy() sdk.Account {
	return NewBaseAccount(acc.Address, acc.Coins, acc.PubKey, acc.AccountNumber, acc.Sequence)
}

func (acc *BaseAccount) AminoSize(cdc *amino.Codec) int {
	size := 0
	if len(acc.Address) != 0 {
		size += 1 + amino.ByteSliceSize(acc.Address)
	}
	for _, coin := range acc.Coins {
		coinSize := coin.AminoSize(cdc)
		size += 1 + amino.UvarintSize(uint64(coinSize)) + coinSize
	}
	if acc.PubKey != nil {
		pkSize := cryptoamino.PubKeyAminoSize(acc.PubKey, cdc)
		size += 1 + amino.UvarintSize(uint64(pkSize)) + pkSize
	}
	if acc.AccountNumber != 0 {
		size += 1 + amino.UvarintSize(acc.AccountNumber)
	}
	if acc.Sequence != 0 {
		size += 1 + amino.UvarintSize(acc.Sequence)
	}
	return size
}

func (acc *BaseAccount) MarshalToAmino(cdc *amino.Codec) ([]byte, error) {
	var buf bytes.Buffer
	buf.Grow(acc.AminoSize(cdc))
	err := acc.MarshalAminoTo(cdc, &buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (acc *BaseAccount) MarshalAminoTo(cdc *amino.Codec, buf *bytes.Buffer) error {
	// field 1
	if len(acc.Address) != 0 {
		const pbKey = 1<<3 | 2
		err := amino.EncodeByteSliceWithKeyToBuffer(buf, acc.Address, pbKey)
		if err != nil {
			return err
		}
	}

	// field 2
	for _, coin := range acc.Coins {
		const pbKey = 2<<3 | 2
		buf.WriteByte(pbKey)
		coinSize := coin.AminoSize(cdc)
		err := amino.EncodeUvarintToBuffer(buf, uint64(coinSize))
		if err != nil {
			return err
		}
		lenBeforeData := buf.Len()
		err = coin.MarshalAminoTo(cdc, buf)
		if err != nil {
			return err
		}
		if buf.Len()-lenBeforeData != coinSize {
			return amino.NewSizerError(coin, coinSize, buf.Len()-lenBeforeData)
		}
	}

	// field 3
	if acc.PubKey != nil {
		const pbKey = 3<<3 | 2
		buf.WriteByte(pbKey)
		pubKeySize := cryptoamino.PubKeyAminoSize(acc.PubKey, cdc)
		err := amino.EncodeUvarintToBuffer(buf, uint64(pubKeySize))
		if err != nil {
			return err
		}
		lenBeforeData := buf.Len()
		err = cryptoamino.MarshalPubKeyAminoTo(cdc, acc.PubKey, buf)
		if err != nil {
			return err
		}
		if buf.Len()-lenBeforeData != pubKeySize {
			return amino.NewSizerError(acc.PubKey, pubKeySize, buf.Len()-lenBeforeData)
		}
	}

	// field 4
	if acc.AccountNumber != 0 {
		const pbKey = 4 << 3
		err := amino.EncodeUvarintWithKeyToBuffer(buf, acc.AccountNumber, pbKey)
		if err != nil {
			return err
		}
	}

	// field 5
	if acc.Sequence != 0 {
		const pbKey = 5 << 3
		err := amino.EncodeUvarintWithKeyToBuffer(buf, acc.Sequence, pbKey)
		if err != nil {
			return err
		}
	}

	return nil
}

// NewBaseAccount creates a new BaseAccount object
func NewBaseAccount(address sdk.AccAddress, coins sdk.Coins,
	pubKey crypto.PubKey, accountNumber uint64, sequence uint64) *BaseAccount {

	return &BaseAccount{
		Address:       address,
		Coins:         coins,
		PubKey:        pubKey,
		AccountNumber: accountNumber,
		Sequence:      sequence,
	}
}

// ProtoBaseAccount - a prototype function for BaseAccount
func ProtoBaseAccount() exported.Account {
	return &BaseAccount{}
}

// NewBaseAccountWithAddress - returns a new base account with a given address
func NewBaseAccountWithAddress(addr sdk.AccAddress) BaseAccount {
	return BaseAccount{
		Address: addr,
	}
}

// GetAddress - Implements sdk.Account.
func (acc BaseAccount) GetAddress() sdk.AccAddress {
	return acc.Address
}

// SetAddress - Implements sdk.Account.
func (acc *BaseAccount) SetAddress(addr sdk.AccAddress) error {
	if len(acc.Address) != 0 {
		return errors.New("cannot override BaseAccount address")
	}
	acc.Address = addr
	return nil
}

// GetPubKey - Implements sdk.Account.
func (acc BaseAccount) GetPubKey() crypto.PubKey {
	return acc.PubKey
}

// SetPubKey - Implements sdk.Account.
func (acc *BaseAccount) SetPubKey(pubKey crypto.PubKey) error {
	acc.PubKey = pubKey
	return nil
}

// GetCoins - Implements sdk.Account.
func (acc *BaseAccount) GetCoins() sdk.Coins {
	return acc.Coins
}

// SetCoins - Implements sdk.Account.
func (acc *BaseAccount) SetCoins(coins sdk.Coins) error {
	acc.Coins = coins
	return nil
}

// GetAccountNumber - Implements Account
func (acc *BaseAccount) GetAccountNumber() uint64 {
	return acc.AccountNumber
}

// SetAccountNumber - Implements Account
func (acc *BaseAccount) SetAccountNumber(accNumber uint64) error {
	acc.AccountNumber = accNumber
	return nil
}

// GetSequence - Implements sdk.Account.
func (acc *BaseAccount) GetSequence() uint64 {
	return acc.Sequence
}

// SetSequence - Implements sdk.Account.
func (acc *BaseAccount) SetSequence(seq uint64) error {
	acc.Sequence = seq
	return nil
}

// SpendableCoins returns the total set of spendable coins. For a base account,
// this is simply the base coins.
func (acc *BaseAccount) SpendableCoins(_ time.Time) sdk.Coins {
	return acc.GetCoins()
}

// Validate checks for errors on the account fields
func (acc BaseAccount) Validate() error {
	if acc.PubKey != nil && acc.Address != nil &&
		!bytes.Equal(acc.PubKey.Address().Bytes(), acc.Address.Bytes()) {
		return errors.New("pubkey and address pair is invalid")
	}

	return nil
}

type baseAccountPretty struct {
	Address       sdk.AccAddress `json:"address" yaml:"address"`
	Coins         sdk.Coins      `json:"coins" yaml:"coins"`
	PubKey        string         `json:"public_key" yaml:"public_key"`
	AccountNumber uint64         `json:"account_number" yaml:"account_number"`
	Sequence      uint64         `json:"sequence" yaml:"sequence"`
}

func (acc BaseAccount) String() string {
	out, _ := acc.MarshalYAML()
	return out.(string)
}

// MarshalYAML returns the YAML representation of an account.
func (acc BaseAccount) MarshalYAML() (interface{}, error) {
	alias := baseAccountPretty{
		Address:       acc.Address,
		Coins:         acc.Coins,
		AccountNumber: acc.AccountNumber,
		Sequence:      acc.Sequence,
	}

	if acc.PubKey != nil {
		pks, err := sdk.Bech32ifyPubKey(sdk.Bech32PubKeyTypeAccPub, acc.PubKey)
		if err != nil {
			return nil, err
		}

		alias.PubKey = pks
	}

	bz, err := yaml.Marshal(alias)
	if err != nil {
		return nil, err
	}

	return string(bz), err
}

func (acc BaseAccount) GetStateRoot() ethcmn.Hash {
	return ethcmn.Hash{}
}

// NewModuleAddress creates an AccAddress from the hash of the module's name
func NewModuleAddress(name string) sdk.AccAddress {
	return sdk.AccAddress(crypto.AddressHash([]byte(name)))
}
