package v2

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func MigrateStore(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) error {
	store := ctx.KVStore(storeKey)

	return migrateVault(store, cdc)
}

func migrateVault(store sdk.KVStore, cdc codec.BinaryCodec) error {
	oldStoreIter := store.Iterator(nil, nil)

	for ; oldStoreIter.Valid(); oldStoreIter.Next() {
		oldKey := oldStoreIter.Key()
		oldVal := store.Get(oldKey)

		newKey, newVal := migrateValue(cdc, oldKey, oldVal)
		store.Set(newKey, newVal)
		store.Delete(oldKey) // Delete old key, value
	}

	return nil
}
