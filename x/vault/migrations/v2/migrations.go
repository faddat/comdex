package v2

import (
	vaultV01Types "github.com/comdex-official/comdex/x/vault/migrations/v1/types"
	"github.com/comdex-official/comdex/x/vault/types"
)

/*func MigrateStore(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) error {
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
}*/

func MigrateJSON(oldState vaultV01Types.GenesisState) types.GenesisState {
	return types.GenesisState{
		Vaults:                      nil,
		StableMintVault:             nil,
		AppExtendedPairVaultMapping: nil,
		UserVaultAssetMapping:       nil,
	}
}
