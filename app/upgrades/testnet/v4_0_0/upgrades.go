package v4_0_0

import (
	vaultV01Types "github.com/comdex-official/comdex/x/vault/migrations/v1/types"
	vaultV02 "github.com/comdex-official/comdex/x/vault/migrations/v2"
	vaultTypes "github.com/comdex-official/comdex/x/vault/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/genutil/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v4_0_0
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// This change is only for testnet upgrade
		newVM, err := mm.RunMigrations(ctx, configurator, fromVM)

		if err != nil {
			return newVM, err
		}
		return newVM, err
	}
}

func MigrateGenesis(appState types.AppMap, clientCtx client.Context) types.AppMap {

	if appState[vaultTypes.ModuleName] == nil {
		return appState
	}

	// unmarshal relative source genesis application state
	var oldVaultState vaultV01Types.GenesisState
	if err := clientCtx.Codec.UnmarshalJSON(appState[vaultTypes.ModuleName], &oldVaultState); err != nil {
		return appState
	}

	// delete deprecated x/feemarket genesis state
	delete(appState, vaultTypes.ModuleName)

	// Migrate relative source genesis application state and marshal it into
	// the respective key.
	newFeeMarketState := vaultV02.MigrateJSON(oldVaultState)

	vaultBz, err := clientCtx.Codec.MarshalJSON(&newFeeMarketState)
	if err != nil {
		return appState
	}

	appState[vaultTypes.ModuleName] = vaultBz

	return appState
}
