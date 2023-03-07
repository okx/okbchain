package types

import (
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
)

const (
	// module name
	ModuleName = "auth"

	// StoreKey is string representation of the store key for auth
	StoreKey = "acc"

	// FeeCollectorName the root string for the fee collector account address
	FeeCollectorName = "fee_collector"

	// QuerierRoute is the querier route for acc
	QuerierRoute = StoreKey
)

var (
	// AddressStoreKeyPrefix prefix for account-by-address store
	AddressStoreKeyPrefix = []byte{0x01}

	// param key for global account number
	GlobalAccountNumberKey = []byte("globalAccountNumber")
)

// AddressStoreKey turn an address to key used to get it from the account store
func AddressStoreKey(addr sdk.AccAddress) []byte {
	return append(AddressStoreKeyPrefix, addr.Bytes()...)
}

// MakeAddressStoreKey return an address store key for the given address,
// it will try copy the key to target slice if its capacity is enough.
func MakeAddressStoreKey(addr sdk.AccAddress, target []byte) []byte {
	target = target[:0]
	if cap(target) >= len(AddressStoreKeyPrefix)+len(addr) {
		target = append(target, AddressStoreKeyPrefix...)
		return append(target, addr.Bytes()...)
	}
	return AddressStoreKey(addr)
}
