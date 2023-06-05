package types

import (
	"bytes"
	"os"
	"testing"

	ethcmn "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/okx/okbchain/app/crypto/ethsecp256k1"
	ethermint "github.com/okx/okbchain/app/types"
	sdkcodec "github.com/okx/okbchain/libs/cosmos-sdk/codec"
	"github.com/okx/okbchain/libs/cosmos-sdk/store"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/prefix"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth"
	authexported "github.com/okx/okbchain/libs/cosmos-sdk/x/auth/exported"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/bank"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/gov/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/supply"
	abci "github.com/okx/okbchain/libs/tendermint/abci/types"
	tmlog "github.com/okx/okbchain/libs/tendermint/libs/log"
	tmdb "github.com/okx/okbchain/libs/tm-db"
	"github.com/okx/okbchain/x/params"
	"github.com/stretchr/testify/suite"
)

type JournalTestSuite struct {
	suite.Suite

	address ethcmn.Address
	journal *journal
	ctx     sdk.Context
	stateDB *CommitStateDB
}

func newTestCodec() *sdkcodec.Codec {
	cdc := sdkcodec.New()

	RegisterCodec(cdc)
	sdk.RegisterCodec(cdc)
	ethsecp256k1.RegisterCodec(cdc)
	sdkcodec.RegisterCrypto(cdc)
	auth.RegisterCodec(cdc)
	ethermint.RegisterCodec(cdc)

	return cdc
}

func (suite *JournalTestSuite) SetupTest() {
	suite.setup()

	privkey, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)

	suite.address = ethcmn.BytesToAddress(privkey.PubKey().Address().Bytes())
	suite.journal = newJournal()

	balance := sdk.NewCoins(ethermint.NewPhotonCoin(sdk.NewInt(100)))
	acc := &ethermint.EthAccount{
		BaseAccount: auth.NewBaseAccount(sdk.AccAddress(suite.address.Bytes()), balance, nil, 0, 0),
		CodeHash:    ethcrypto.Keccak256(nil),
	}

	suite.stateDB.accountKeeper.SetAccount(suite.ctx, acc)
	// suite.stateDB.bankKeeper.SetBalance(suite.ctx, sdk.AccAddress(suite.address.Bytes()), balance)
	suite.stateDB.SetLogs(ethcmn.BytesToHash([]byte("topic")), []*ethtypes.Log{
		{
			Address:     suite.address,
			Topics:      []ethcmn.Hash{ethcmn.BytesToHash([]byte("topic_0"))},
			Data:        []byte("data_0"),
			BlockNumber: 1,
			TxHash:      ethcmn.BytesToHash([]byte("tx_hash")),
			TxIndex:     1,
			BlockHash:   ethcmn.BytesToHash([]byte("block_hash")),
			Index:       1,
			Removed:     false,
		},
		{
			Address:     suite.address,
			Topics:      []ethcmn.Hash{ethcmn.BytesToHash([]byte("topic_1"))},
			Data:        []byte("data_1"),
			BlockNumber: 10,
			TxHash:      ethcmn.BytesToHash([]byte("tx_hash")),
			TxIndex:     0,
			BlockHash:   ethcmn.BytesToHash([]byte("block_hash")),
			Index:       0,
			Removed:     false,
		},
	})
}

// setup performs a manual setup of the GoLevelDB and mounts the required IAVL stores. We use the manual
// setup here instead of the Ethermint app test setup because the journal methods are private and using
// the latter would result in a cycle dependency. We also want to avoid declaring the journal methods public
// to maintain consistency with the Geth implementation.
func (suite *JournalTestSuite) setup() {
	authKey := sdk.NewKVStoreKey(auth.StoreKey)
	supplyKey := sdk.NewKVStoreKey(supply.StoreKey)
	paramsKey := sdk.NewKVStoreKey(params.StoreKey)
	paramsTKey := sdk.NewTransientStoreKey(params.TStoreKey)
	// bankKey := sdk.NewKVStoreKey(bank.StoreKey)

	db := tmdb.NewDB("state", tmdb.GoLevelDBBackend, "temp")
	defer func() {
		os.RemoveAll("temp")
	}()

	cms := store.NewCommitMultiStore(db)
	cms.MountStoreWithDB(authKey, sdk.StoreTypeMPT, db)
	cms.MountStoreWithDB(paramsKey, sdk.StoreTypeIAVL, db)
	cms.MountStoreWithDB(paramsTKey, sdk.StoreTypeTransient, db)

	err := cms.LoadLatestVersion()
	suite.Require().NoError(err)

	cdc := newTestCodec()

	paramsKeeper := params.NewKeeper(cdc, paramsKey, paramsTKey, tmlog.NewNopLogger())

	authSubspace := paramsKeeper.Subspace(auth.DefaultParamspace)
	bankSubspace := paramsKeeper.Subspace(bank.DefaultParamspace)
	evmSubspace := paramsKeeper.Subspace(types.DefaultParamspace).WithKeyTable(ParamKeyTable())

	ak := auth.NewAccountKeeper(cdc, authKey, authSubspace, ethermint.ProtoAccount)
	bk := bank.NewBaseKeeper(ak, bankSubspace, make(map[string]bool))
	sk := supply.NewKeeper(cdc, supplyKey, ak, bank.NewBankKeeperAdapter(bk), make(map[string][]string))
	suite.ctx = sdk.NewContext(cms, abci.Header{ChainID: "ethermint-8"}, false, tmlog.NewNopLogger())
	csdbParams := CommitStateDBParams{
		StoreKey:      authKey,
		ParamSpace:    evmSubspace,
		AccountKeeper: &ak,
		SupplyKeeper:  sk,
		BankKeeper:    bk,
		Ada:           nil,
		Cdc:           cdc,
		DB:            nil,
	}
	suite.stateDB = NewCommitStateDB(csdbParams).WithContext(suite.ctx)
	suite.stateDB.SetParams(DefaultParams())
}

func TestJournalTestSuite(t *testing.T) {
	suite.Run(t, new(JournalTestSuite))
}

func (suite *JournalTestSuite) TestJournal_append_revert() {
	testCases := []struct {
		name  string
		entry journalEntry
	}{
		{
			"createObjectChange",
			createObjectChange{
				account: &suite.address,
			},
		},
		{
			"resetObjectChange",
			resetObjectChange{
				prev: &stateObject{
					address: suite.address,
				},
			},
		},
		{
			"suicideChange",
			suicideChange{
				account:     &suite.address,
				prev:        false,
				prevBalance: sdk.OneDec(),
			},
		},
		{
			"balanceChange",
			balanceChange{
				account: &suite.address,
				prev:    sdk.OneDec(),
			},
		},
		{
			"nonceChange",
			nonceChange{
				account: &suite.address,
				prev:    1,
			},
		},
		{
			"storageChange",
			storageChange{
				account:   &suite.address,
				key:       ethcmn.BytesToHash([]byte("key")),
				prevValue: ethcmn.BytesToHash([]byte("value")),
			},
		},
		{
			"codeChange",
			codeChange{
				account:  &suite.address,
				prevCode: []byte("code"),
				prevHash: []byte("hash"),
			},
		},
		{
			"touchChange",
			touchChange{
				account: &suite.address,
			},
		},
		{
			"refundChange",
			refundChange{
				prev: 1,
			},
		},
		{
			"addPreimageChange",
			addPreimageChange{
				hash: ethcmn.BytesToHash([]byte("hash")),
			},
		},
		{
			"addLogChange",
			addLogChange{
				txhash: ethcmn.BytesToHash([]byte("hash")),
			},
		},
		{
			"addLogChange - 2 logs",
			addLogChange{
				txhash: ethcmn.BytesToHash([]byte("txhash")),
			},
		},
		{
			"accessListAddAccountChange",
			accessListAddAccountChange{
				address: &suite.address,
			},
		},
	}
	var dirtyCount int
	for i, tc := range testCases {
		suite.journal.append(tc.entry)
		suite.Require().Equal(suite.journal.length(), i+1, tc.name)
		if tc.entry.dirtied() != nil {
			dirtyCount++

			suite.Require().Equal(dirtyCount, suite.journal.dirties[suite.address], tc.name)
		}
	}

	// revert to the initial journal state
	suite.journal.revert(suite.stateDB, 0)

	// verify the dirty entry has been deleted
	idx, ok := suite.journal.dirties[suite.address]
	suite.Require().False(ok)
	suite.Require().Zero(idx)
}

func (suite *JournalTestSuite) TestJournal_preimage_revert() {
	suite.stateDB.preimages = map[ethcmn.Hash][]byte{
		ethcmn.BytesToHash([]byte("hash")):  []byte("preimage0"),
		ethcmn.BytesToHash([]byte("hash1")): []byte("preimage1"),
		ethcmn.BytesToHash([]byte("hash2")): []byte("preimage2"),
	}

	change := addPreimageChange{
		hash: ethcmn.BytesToHash([]byte("hash")),
	}

	// delete first entry
	change.revert(suite.stateDB)
	suite.Require().Len(suite.stateDB.preimages, 2)

	for key, value := range suite.stateDB.preimages {
		suite.Require().NotEqual(len("preimage"), string(value), key.String())
	}
}

func (suite *JournalTestSuite) TestJournal_createObjectChange_revert() {
	addr := ethcmn.BytesToAddress([]byte("addr"))

	suite.stateDB.stateObjects = map[ethcmn.Address]*stateObject{
		addr: &stateObject{
			address: addr,
		},
		ethcmn.BytesToAddress([]byte("addr1")): &stateObject{
			address: ethcmn.BytesToAddress([]byte("addr1")),
		},
		ethcmn.BytesToAddress([]byte("addr2")): &stateObject{
			address: ethcmn.BytesToAddress([]byte("addr2")),
		},
	}

	change := createObjectChange{
		account: &addr,
	}

	// delete first entry
	change.revert(suite.stateDB)
	suite.Require().Len(suite.stateDB.stateObjects, 2)
	suite.Require().Equal(len(suite.stateDB.stateObjects), len(suite.stateDB.stateObjects))

	for k, entry := range suite.stateDB.stateObjects {
		suite.Require().Equal(k.String(), entry.address.String())
		_, found := suite.stateDB.stateObjects[entry.address]
		suite.Require().True(found)
	}
}

func (suite *JournalTestSuite) TestJournal_dirty() {
	// dirty entry hasn't been set
	idx, ok := suite.journal.dirties[suite.address]
	suite.Require().False(ok)
	suite.Require().Zero(idx)

	// update dirty count
	suite.journal.dirty(suite.address)
	suite.Require().Equal(1, suite.journal.dirties[suite.address])
}

func (suite *JournalTestSuite) TestJournal_cmchange_revert() {
	balance := sdk.NewCoins(ethermint.NewPhotonCoin(sdk.NewInt(100)))
	acc := &ethermint.EthAccount{
		BaseAccount: auth.NewBaseAccount(sdk.AccAddress(suite.address.Bytes()), balance, nil, 0, 0),
		CodeHash:    ethcrypto.Keccak256(nil),
	}

	suite.stateDB.accountKeeper.RemoveAccount(suite.ctx, acc)

	// prepare account
	update := suite.stateDB.accountKeeper.NewAccountWithAddress(suite.ctx, sdk.AccAddress{0x1})
	update.SetCoins(sdk.NewCoins(sdk.NewCoin("okt", sdk.NewDec(1))))

	insert := suite.stateDB.accountKeeper.NewAccountWithAddress(suite.ctx, sdk.AccAddress{0x2})
	insert.SetCoins(sdk.NewCoins(sdk.NewCoin("okt", sdk.NewDec(1))))

	delete := suite.stateDB.accountKeeper.NewAccountWithAddress(suite.ctx, sdk.AccAddress{0x3})
	delete.SetCoins(sdk.NewCoins(sdk.NewCoin("okt", sdk.NewDec(1))))

	suite.stateDB.accountKeeper.SetAccount(suite.ctx, update)
	suite.stateDB.accountKeeper.SetAccount(suite.ctx, delete)

	//prepare storgae key
	suite.stateDB.dbAdapter = DefaultPrefixDb{}
	stateDB := suite.stateDB
	storeDB := stateDB.dbAdapter.NewStore(suite.ctx.KVStore(stateDB.storeKey), AddressStoragePrefix(ethcmn.Address{0x1}))
	prefixDB, ok := storeDB.(prefix.Store)
	suite.Require().True(ok)
	updateKey := []byte("1")
	insertKey := []byte("2")
	deleteKey := []byte("3")
	prefixDB.Set(updateKey, []byte("1"))
	prefixDB.Set(deleteKey, []byte("1"))

	subCtx, write := suite.ctx.CacheContextWithMultiSnapshotRWSet()
	suite.stateDB.accountKeeper.SetAccount(subCtx, insert)
	update.SetCoins(sdk.NewCoins(sdk.NewCoin("okt", sdk.NewDec(0))))
	suite.stateDB.accountKeeper.SetAccount(subCtx, update)
	suite.stateDB.accountKeeper.RemoveAccount(subCtx, delete)

	storeDB1 := stateDB.dbAdapter.NewStore(subCtx.KVStore(stateDB.storeKey), AddressStoragePrefix(ethcmn.Address{0x1}))
	prefixDB1, ok := storeDB1.(prefix.Store)
	suite.Require().True(ok)
	prefixDB1.Set(insertKey, []byte("1"))
	prefixDB1.Set(updateKey, []byte("2"))
	prefixDB1.Delete(deleteKey)

	snapshot := write()
	suite.stateDB.accountKeeper.IterateAccounts(suite.ctx, func(account authexported.Account) bool {
		if account.GetAddress().Equals(update.GetAddress()) {
			suite.Require().Equal(update.String(), account.String())
		} else if account.GetAddress().Equals(insert.GetAddress()) {
			suite.Require().Equal(insert.String(), account.String())
		} else {
			panic("must be only update or insert acc")
			return true
		}
		return false
	})
	iter := prefixDB1.Iterator(nil, nil)
	for iter.Valid() {
		if bytes.Compare(iter.Key(), updateKey) == 0 {
			suite.Require().Equal(iter.Value(), []byte("2"))
		} else if bytes.Compare(iter.Key(), insertKey) == 0 {
			suite.Require().Equal(iter.Value(), []byte("1"))
		} else {
			panic("must be only update or insert acc")
		}
		iter.Next()
	}

	change := cmChange{sets: &snapshot}

	// delete first entry
	change.revert(suite.stateDB)

	suite.stateDB.accountKeeper.IterateAccounts(suite.ctx, func(account authexported.Account) bool {
		if account.GetAddress().Equals(update.GetAddress()) {
			update.SetCoins(sdk.NewCoins(sdk.NewCoin("okt", sdk.NewDec(1))))
			suite.Require().Equal(update.String(), account.String())
		} else if account.GetAddress().Equals(insert.GetAddress()) {
			suite.Require().Equal(insert.String(), account.String())
		} else if account.GetAddress().Equals(delete.GetAddress()) {
			suite.Require().Equal(delete.String(), account.String())
		} else {
			panic("must be only update or insert acc")
			return true
		}
		return false
	})

	storeDB2 := stateDB.dbAdapter.NewStore(subCtx.KVStore(stateDB.storeKey), AddressStoragePrefix(ethcmn.Address{0x1}))
	prefixDB2, ok := storeDB2.(prefix.Store)
	suite.Require().True(ok)

	iter = prefixDB2.Iterator(nil, nil)
	for iter.Valid() {
		if bytes.Compare(iter.Key(), updateKey) == 0 {
			suite.Require().Equal(iter.Value(), []byte("1"))
		} else if bytes.Compare(iter.Key(), insertKey) == 0 {
			suite.Require().Equal(iter.Value(), []byte("1"))
		} else if bytes.Compare(iter.Key(), deleteKey) == 0 {
			suite.Require().Equal(iter.Value(), []byte("1"))
		} else {
			panic("must be only update or insert acc")
		}
		iter.Next()
	}

}
