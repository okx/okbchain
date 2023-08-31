package types

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/okx/brczero/app/crypto/ethsecp256k1"
	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	"github.com/okx/brczero/libs/cosmos-sdk/x/auth"
	"github.com/okx/brczero/libs/cosmos-sdk/x/auth/exported"
	authtypes "github.com/okx/brczero/libs/cosmos-sdk/x/auth/types"
	tmcrypto "github.com/okx/brczero/libs/tendermint/crypto"
	"github.com/okx/brczero/libs/tendermint/crypto/ed25519"
	tmamino "github.com/okx/brczero/libs/tendermint/crypto/encoding/amino"
	"github.com/okx/brczero/libs/tendermint/crypto/secp256k1"
	"github.com/okx/brczero/libs/tendermint/crypto/sr25519"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func init() {
	tmamino.RegisterKeyType(ethsecp256k1.PubKey{}, ethsecp256k1.PubKeyName)
	tmamino.RegisterKeyType(ethsecp256k1.PrivKey{}, ethsecp256k1.PrivKeyName)
}

type AccountTestSuite struct {
	suite.Suite

	account *EthAccount
}

func (suite *AccountTestSuite) SetupTest() {
	pubkey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubkey.Address())
	balance := sdk.NewCoins(NewPhotonCoin(sdk.OneInt()))
	baseAcc := auth.NewBaseAccount(addr, balance, pubkey, 10, 50)
	suite.account = &EthAccount{
		BaseAccount: baseAcc,
		CodeHash:    []byte{1, 2},
		StateRoot:   ethtypes.EmptyRootHash,
	}
}

func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, new(AccountTestSuite))
}

func (suite *AccountTestSuite) TestEthAccount_Balance() {

	testCases := []struct {
		name         string
		denom        string
		initialCoins sdk.Coins
		amount       sdk.Int
	}{
		{"positive diff", NativeToken, sdk.Coins{}, sdk.OneInt()},
		{"zero diff, same coin", NativeToken, sdk.NewCoins(NewPhotonCoin(sdk.ZeroInt())), sdk.ZeroInt()},
		{"zero diff, other coin", sdk.DefaultBondDenom, sdk.NewCoins(NewPhotonCoin(sdk.ZeroInt())), sdk.ZeroInt()},
		{"negative diff", NativeToken, sdk.NewCoins(NewPhotonCoin(sdk.NewInt(10))), sdk.NewInt(1)},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset values
			suite.account.SetCoins(tc.initialCoins)

			suite.account.SetBalance(tc.denom, sdk.NewDecFromInt(tc.amount))
			suite.Require().Equal(sdk.NewDecFromInt(tc.amount), suite.account.Balance(tc.denom))
		})
	}

}

func (suite *AccountTestSuite) TestEthermintAccountJSON() {
	bz, err := json.Marshal(suite.account)
	suite.Require().NoError(err)

	bz1, err := suite.account.MarshalJSON()
	suite.Require().NoError(err)
	suite.Require().Equal(string(bz1), string(bz))

	var a EthAccount
	suite.Require().NoError(a.UnmarshalJSON(bz))
	suite.Require().Equal(suite.account.String(), a.String())
	suite.Require().Equal(suite.account.PubKey, a.PubKey)
}

func (suite *AccountTestSuite) TestEthermintPubKeyJSON() {
	privkey, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)
	bz := privkey.PubKey().Bytes()

	pubk, err := tmamino.PubKeyFromBytes(bz)
	suite.Require().NoError(err)
	suite.Require().Equal(pubk, privkey.PubKey())
}

func (suite *AccountTestSuite) TestSecpPubKeyJSON() {
	pubkey := secp256k1.GenPrivKey().PubKey()
	bz := pubkey.Bytes()

	pubk, err := tmamino.PubKeyFromBytes(bz)
	suite.Require().NoError(err)
	suite.Require().Equal(pubk, pubkey)
}

func (suite *AccountTestSuite) TestEthermintAccount_String() {
	config := sdk.GetConfig()
	SetBech32Prefixes(config)

	bech32pubkey, err := sdk.Bech32ifyPubKey(sdk.Bech32PubKeyTypeAccPub, suite.account.PubKey)
	suite.Require().NoError(err)

	accountStr := fmt.Sprintf(`|
  address: %s
  eth_address: %s
  coins:
  - denom: %s
    amount: "1.000000000000000000"
  public_key: %s
  account_number: 10
  sequence: 50
  code_hash: "0102"
  state_root: 0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421
`, suite.account.Address, suite.account.EthAddress().String(), sdk.DefaultBondDenom, bech32pubkey)

	suite.Require().Equal(accountStr, suite.account.String())

	i, err := suite.account.MarshalYAML()
	suite.Require().NoError(err)

	var ok bool
	accountStr, ok = i.(string)
	suite.Require().True(ok)
	suite.Require().Contains(accountStr, suite.account.Address.String())
	suite.Require().Contains(accountStr, bech32pubkey)
}

func (suite *AccountTestSuite) TestEthermintAccount_MarshalJSON() {
	bz, err := suite.account.MarshalJSON()
	suite.Require().NoError(err)
	suite.Require().Contains(string(bz), suite.account.EthAddress().String())

	res := new(EthAccount)
	err = res.UnmarshalJSON(bz)
	suite.Require().NoError(err)
	suite.Require().Equal(suite.account, res)

	bech32pubkey, err := sdk.Bech32ifyPubKey(sdk.Bech32PubKeyTypeAccPub, suite.account.PubKey)
	suite.Require().NoError(err)

	// test that the sdk.AccAddress is populated from the hex address
	jsonAcc := fmt.Sprintf(
		`{"address":"","eth_address":"%s","coins":[{"denom":"aphoton","amount":"1"}],"public_key":"%s","account_number":10,"sequence":50,"code_hash":"0102"}`,
		suite.account.EthAddress().String(), bech32pubkey,
	)

	res = new(EthAccount)
	err = res.UnmarshalJSON([]byte(jsonAcc))
	suite.Require().NoError(err)
	suite.Require().Equal(suite.account.Address.String(), res.Address.String())

	jsonAcc = fmt.Sprintf(
		`{"address":"","eth_address":"","coins":[{"denom":"aphoton","amount":"1"}],"public_key":"%s","account_number":10,"sequence":50,"code_hash":"0102"}`,
		bech32pubkey,
	)

	res = new(EthAccount)
	err = res.UnmarshalJSON([]byte(jsonAcc))
	suite.Require().Error(err, "should fail if both address are empty")

	// test that the sdk.AccAddress is populated from the hex address
	jsonAcc = fmt.Sprintf(
		`{"address": "%s","eth_address":"0x0000000000000000000000000000000000000000","coins":[{"denom":"aphoton","amount":"1"}],"public_key":"%s","account_number":10,"sequence":50,"code_hash":"0102"}`,
		suite.account.Address.String(), bech32pubkey,
	)

	res = new(EthAccount)
	err = res.UnmarshalJSON([]byte(jsonAcc))
	suite.Require().Error(err, "should fail if addresses mismatch")
}

func TestEthAccountAmino(t *testing.T) {
	cdc := codec.New()
	cdc.RegisterInterface((*exported.Account)(nil), nil)
	RegisterCodec(cdc)

	cdc.RegisterInterface((*tmcrypto.PubKey)(nil), nil)
	cdc.RegisterConcrete(ed25519.PubKeyEd25519{},
		ed25519.PubKeyAminoName, nil)
	cdc.RegisterConcrete(sr25519.PubKeySr25519{},
		sr25519.PubKeyAminoName, nil)
	cdc.RegisterConcrete(secp256k1.PubKeySecp256k1{},
		secp256k1.PubKeyAminoName, nil)

	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	accounts := []EthAccount{
		{},
		{
			auth.NewBaseAccount(
				addr,
				sdk.NewCoins(NewPhotonCoin(sdk.OneInt()), sdk.Coin{"heco", sdk.Dec{big.NewInt(1)}}),
				pubKey,
				1,
				1,
			),
			ethcrypto.Keccak256(nil),
			ethtypes.EmptyRootHash,
		},
		{
			auth.NewBaseAccount(
				addr,
				sdk.NewCoins(NewPhotonCoin(sdk.ZeroInt()), sdk.Coin{"heco", sdk.Dec{big.NewInt(0)}}),
				pubKey,
				0,
				0,
			),
			ethcrypto.Keccak256(nil),
			ethtypes.EmptyRootHash,
		},
		{
			auth.NewBaseAccount(
				nil,
				nil,
				nil,
				0,
				0,
			),
			ethcrypto.Keccak256(nil),
			ethtypes.EmptyRootHash,
		},
		{
			BaseAccount: &auth.BaseAccount{},
		},
	}

	for _, testAccount := range accounts {
		data, err := cdc.MarshalBinaryBare(&testAccount)
		if err != nil {
			t.Fatal("marshal error")
		}
		require.Equal(t, len(data), 4+testAccount.AminoSize(cdc))

		var accountFromAmino exported.Account

		err = cdc.UnmarshalBinaryBare(data, &accountFromAmino)
		if err != nil {
			t.Fatal("unmarshal error")
		}

		var accountFromUnmarshaller exported.Account
		v, err := cdc.UnmarshalBinaryBareWithRegisteredUnmarshaller(data, (*exported.Account)(nil))
		require.NoError(t, err)
		accountFromUnmarshaller, ok := v.(exported.Account)
		require.True(t, ok)

		require.EqualValues(t, accountFromAmino, accountFromUnmarshaller)

		var ethAccount EthAccount
		err = ethAccount.UnmarshalFromAmino(cdc, data[4:])
		require.NoError(t, err)
		require.EqualValues(t, accountFromAmino, &ethAccount)

		dataFromMarshaller, err := cdc.MarshalBinaryBareWithRegisteredMarshaller(&testAccount)
		require.NoError(t, err)
		require.EqualValues(t, data, dataFromMarshaller)

		dataFromSizer, err := cdc.MarshalBinaryWithSizer(&testAccount, false)
		require.NoError(t, err)
		require.EqualValues(t, data, dataFromSizer)

		dataFromMarshaller, err = ethAccount.MarshalToAmino(cdc)
		if dataFromMarshaller == nil {
			dataFromMarshaller = []byte{}
		}
		require.Equal(t, data[4:], dataFromMarshaller)
	}
}

func TestEthAccountDeepCopy(t *testing.T) {
	cdc := codec.New()
	cdc.RegisterInterface((*exported.Account)(nil), nil)
	RegisterCodec(cdc)

	cdc.RegisterInterface((*tmcrypto.PubKey)(nil), nil)
	cdc.RegisterConcrete(ed25519.PubKeyEd25519{},
		ed25519.PubKeyAminoName, nil)
	cdc.RegisterConcrete(sr25519.PubKeySr25519{},
		sr25519.PubKeyAminoName, nil)
	cdc.RegisterConcrete(secp256k1.PubKeySecp256k1{},
		secp256k1.PubKeyAminoName, nil)

	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	accounts := []EthAccount{
		{
			auth.NewBaseAccount(
				addr,
				sdk.NewCoins(NewPhotonCoin(sdk.OneInt()), sdk.Coin{"heco", sdk.Dec{big.NewInt(1)}}),
				pubKey,
				1,
				1,
			),
			ethcrypto.Keccak256(nil),
			ethtypes.EmptyRootHash,
		},
		{
			auth.NewBaseAccount(
				addr,
				sdk.NewCoins(NewPhotonCoin(sdk.ZeroInt()), sdk.Coin{"heco", sdk.Dec{big.NewInt(0)}}),
				pubKey,
				0,
				0,
			),
			ethcrypto.Keccak256(nil),
			ethtypes.EmptyRootHash,
		},
		{
			auth.NewBaseAccount(
				addr,
				nil,
				nil,
				0,
				0,
			),
			ethcrypto.Keccak256(nil),
			ethtypes.EmptyRootHash,
		},
	}

	for i, testAccount := range accounts {
		temp := testAccount.DeepCopy()
		require.NotEqual(t, testAccount, temp)

		expectValue, err := cdc.MarshalBinaryBareWithRegisteredMarshaller(&testAccount)
		require.NoError(t, err)

		gotVaule, err := cdc.MarshalBinaryBareWithRegisteredMarshaller(temp)
		require.EqualValues(t, expectValue, gotVaule, fmt.Errorf("error index %d", i))
	}
}

func BenchmarkEthAccountAminoUnmarshal(b *testing.B) {
	cdc := codec.New()
	cdc.RegisterInterface((*exported.Account)(nil), nil)
	RegisterCodec(cdc)

	cdc.RegisterInterface((*tmcrypto.PubKey)(nil), nil)
	cdc.RegisterConcrete(ed25519.PubKeyEd25519{},
		ed25519.PubKeyAminoName, nil)
	cdc.RegisterConcrete(sr25519.PubKeySr25519{},
		sr25519.PubKeyAminoName, nil)
	cdc.RegisterConcrete(secp256k1.PubKeySecp256k1{},
		secp256k1.PubKeyAminoName, nil)

	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	balance := sdk.NewCoins(NewPhotonCoin(sdk.OneInt()))
	testAccount := EthAccount{
		BaseAccount: auth.NewBaseAccount(addr, balance, pubKey, 1, 1),
		CodeHash:    ethcrypto.Keccak256(nil),
		StateRoot:   ethtypes.EmptyRootHash,
	}

	data, _ := cdc.MarshalBinaryBare(&testAccount)

	b.ResetTimer()
	b.ReportAllocs()

	b.Run("amino", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var account exported.Account
			_ = cdc.UnmarshalBinaryBare(data, &account)
		}
	})

	b.Run("unmarshaller", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// var account exported.Account
			_, _ = cdc.UnmarshalBinaryBareWithRegisteredUnmarshaller(data, (*exported.Account)(nil))
		}
	})
}

func BenchmarkEthAccountAminoMarshal(b *testing.B) {
	cdc := codec.New()
	cdc.RegisterInterface((*exported.Account)(nil), nil)
	RegisterCodec(cdc)

	cdc.RegisterInterface((*tmcrypto.PubKey)(nil), nil)
	cdc.RegisterConcrete(ed25519.PubKeyEd25519{},
		ed25519.PubKeyAminoName, nil)
	cdc.RegisterConcrete(sr25519.PubKeySr25519{},
		sr25519.PubKeyAminoName, nil)
	cdc.RegisterConcrete(secp256k1.PubKeySecp256k1{},
		secp256k1.PubKeyAminoName, nil)

	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	balance := sdk.NewCoins(NewPhotonCoin(sdk.OneInt()))
	testAccount := EthAccount{
		BaseAccount: auth.NewBaseAccount(addr, balance, pubKey, 1, 1),
		CodeHash:    ethcrypto.Keccak256(nil),
		StateRoot:   ethtypes.EmptyRootHash,
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.Run("amino", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			data, _ := cdc.MarshalBinaryBare(&testAccount)
			_ = data
		}
	})

	b.Run("marshaller", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			data, _ := cdc.MarshalBinaryBareWithRegisteredMarshaller(&testAccount)
			_ = data
		}
	})

	b.Run("sizer", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			data, _ := cdc.MarshalBinaryWithSizer(&testAccount, false)
			_ = data
		}
	})
}

func (acc EthAccount) utOldCopy() sdk.Account {
	return &EthAccount{
		authtypes.NewBaseAccount(acc.Address, acc.Coins, acc.PubKey, acc.AccountNumber, acc.Sequence),
		acc.CodeHash,
		acc.StateRoot,
	}
}

func BenchmarkEthAccountCopy(b *testing.B) {
	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	balance := sdk.NewCoins(NewPhotonCoin(sdk.OneInt()))
	testAccount := EthAccount{
		BaseAccount: auth.NewBaseAccount(addr, balance, pubKey, 1, 1),
		CodeHash:    ethcrypto.Keccak256(nil),
	}

	var copied sdk.Account

	b.Run("copy", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			copied = testAccount.Copy()
		}
	})
	b.Run("old", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			copied = testAccount.utOldCopy()
		}
	})
	_ = copied
}
