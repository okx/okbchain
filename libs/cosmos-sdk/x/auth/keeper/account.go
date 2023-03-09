package keeper

import (
	"encoding/hex"
	"fmt"
	"github.com/okx/okbchain/common"
	"github.com/tendermint/go-amino"
	"sync"

	"github.com/okx/okbchain/libs/cosmos-sdk/store/mpt"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth/exported"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth/types"
)

// NewAccountWithAddress implements sdk.AccountKeeper.
func (ak AccountKeeper) NewAccountWithAddress(ctx sdk.Context, addr sdk.AccAddress) exported.Account {
	acc := ak.proto()
	err := acc.SetAddress(addr)
	if err != nil {
		panic(err)
	}
	return ak.NewAccount(ctx, acc)
}

// NewAccount sets the next account number to a given account interface
func (ak AccountKeeper) NewAccount(ctx sdk.Context, acc exported.Account) exported.Account {
	//fmt.Println("setAccount:" + acc.GetAddress().String())
	lists := []string{
		"ex1yl6hdjhmkf37639730gffanpzndzdpmhxkqv75",
		"ex183rfa8tvtp6ax7jr7dfaf7ywv870sykxm0qde8",
		"ex1fl48vsnmsdzcv85q5d2q4z5ajdha8yu3ajl2sq",
		"ex1tygms3xhhs3yv487phx3dw4a95jn7t7lfjrmx5",
		"ex1d0h6sf72cve3s50zdwgee4ncvya9l5radj0pjf",
		"ex10d07y265gmmuvt4z0w9aw880jnsr700jjt9qly",
		"ex1jv65s3grqf6v6jl3dp4t6c9t9rk99cd80kjeqg",
	}
	exist := false
	for _, v := range lists {
		if v == acc.GetAddress().String() {
			exist = true
			break
		}
	}
	if acc.GetAddress().String() == "ex1jv65s3grqf6v6jl3dp4t6c9t9rk99cd80kjeqg" {
		fmt.Println(-2)
	}
	if !exist {
		//if o, ok := acc.(exported.ModuleAccount); ok {
		//fmt.Println(-1, acc.GetAddress().String(), o.GetName())
		//} else {
		//fmt.Println(-2, acc.GetAddress().String())
		//}
	} // ex18jwndnml9nfcczjmzyx8wuvl2t96c6g5vdzupg,ex1ajryl6vm2wtsfwy89tzezpn77gkcx65d3u0arz
	//fmt.Println("newAccount:" + acc.GetAddress().String())
	if err := acc.SetAccountNumber(ak.GetNextAccountNumber(ctx)); err != nil {
		panic(err)
	}
	return acc
}

var addrStoreKeyPool = &sync.Pool{
	New: func() interface{} {
		return &[33]byte{}
	},
}

// GetAccount implements sdk.AccountKeeper.
func (ak AccountKeeper) GetAccount(ctx sdk.Context, addr sdk.AccAddress) exported.Account {

	key := ak.mptKey
	store := ctx.GetReusableKVStore(key)
	keyTarget := addrStoreKeyPool.Get().(*[33]byte)
	defer func() {
		addrStoreKeyPool.Put(keyTarget)
		ctx.ReturnKVStore(store)
	}()

	bz := store.Get(types.MakeAddressStoreKey(addr, keyTarget[:0]))
	if bz == nil {
		return nil
	}
	acc := ak.decodeAccount(bz)

	return acc
}

// GetAllAccounts returns all accounts in the accountKeeper.
func (ak AccountKeeper) GetAllAccounts(ctx sdk.Context) (accounts []exported.Account) {
	ak.IterateAccounts(ctx,
		func(acc exported.Account) (stop bool) {
			accounts = append(accounts, acc)
			return false
		})
	return accounts
}

// SetAccount implements sdk.AccountKeeper.
func (ak AccountKeeper) SetAccount(ctx sdk.Context, acc exported.Account) {
	//if acc.GetAddress().String() == "ex1yl6hdjhmkf37639730gffanpzndzdpmhxkqv75" {
	//	fmt.Println(1)
	//}
	//if acc.GetAddress().String() == "ex1hrwe9uhcm6wnhaj4n9a0k22c5xh8h9q5t5n3xe" {
	//	fmt.Println(2)
	//}
	addr := acc.GetAddress()

	key := ak.mptKey
	store := ctx.GetReusableKVStore(key)
	defer ctx.ReturnKVStore(store)

	bz := ak.encodeAccount(acc)

	storeAccKey := types.AddressStoreKey(addr)
	store.Set(storeAccKey, bz)

	if acc.GetAddress().String() == "ex1jv65s3grqf6v6jl3dp4t6c9t9rk99cd80kjeqg" {
		fmt.Println(-3333333333333)
		defer func() {
			count := 0
			ak.IterateAccounts(ctx, common.FFF(&count))
			fmt.Println("key", hex.EncodeToString(storeAccKey))
		}()
	}

	if ctx.IsDeliver() {
		mpt.GAccToPrefetchChannel <- [][]byte{storeAccKey}
	}

	if !ctx.IsCheckTx() && !ctx.IsReCheckTx() {
		if ak.observers != nil {
			for _, observer := range ak.observers {
				if observer != nil {
					observer.OnAccountUpdated(acc)
				}
			}
		}
	}
}

func (ak *AccountKeeper) encodeAccount(acc exported.Account) (bz []byte) {
	var err error
	if accSizer, ok := acc.(amino.MarshalBufferSizer); ok {
		bz, err = ak.cdc.MarshalBinaryWithSizer(accSizer, false)
		if err == nil {
			return bz
		}
	}

	bz, err = ak.cdc.MarshalBinaryBareWithRegisteredMarshaller(acc)
	if err != nil {
		bz, err = ak.cdc.MarshalBinaryBare(acc)
	}
	if err != nil {
		panic(err)
	}
	return bz
}

// RemoveAccount removes an account for the account mapper store.
// NOTE: this will cause supply invariant violation if called
func (ak AccountKeeper) RemoveAccount(ctx sdk.Context, acc exported.Account) {
	addr := acc.GetAddress()
	store := ctx.KVStore(ak.mptKey)
	storeAccKey := types.AddressStoreKey(addr)
	store.Delete(storeAccKey)

	if ctx.IsDeliver() {
		mpt.GAccToPrefetchChannel <- [][]byte{storeAccKey}
	}
}

// IterateAccounts iterates over all the stored accounts and performs a callback function
func (ak AccountKeeper) IterateAccounts(ctx sdk.Context, cb func(account exported.Account) (stop bool)) {

	store := ctx.KVStore(ak.mptKey)
	iterator := sdk.KVStorePrefixIterator(store, types.AddressStoreKeyPrefix)

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		account := ak.decodeAccount(iterator.Value())

		if cb(account) {
			break
		}
	}
}

// IterateAccounts iterates over all the stored accounts and performs a callback function
//
//	TODO by yxq: deprecated
func (ak AccountKeeper) MigrateAccounts(ctx sdk.Context, cb func(account exported.Account, key, value []byte) (stop bool)) {

	store := ctx.KVStore(ak.mptKey)
	iterator := sdk.KVStorePrefixIterator(store, types.AddressStoreKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		account := ak.decodeAccount(iterator.Value())
		if cb(account, iterator.Key(), iterator.Value()) {
			break
		}
	}
}

func (ak AccountKeeper) GetEncodedAccountSize(acc exported.Account) int {
	if sizer, ok := acc.(amino.Sizer); ok {
		// typeprefix + amino bytes
		return 4 + sizer.AminoSize(ak.cdc)
	} else {
		return len(ak.cdc.MustMarshalBinaryBare(acc))
	}
}
