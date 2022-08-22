package keeper

import (
	v2 "github.com/comdex-official/comdex/x/vault/migrations/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	if err := v2.UpdateParams(ctx, &m.keeper.paramstore); err != nil {
		return err
	}

	return v2.MigrateStore(ctx, m.keeper.storeKey, m.keeper.cdc)
}
