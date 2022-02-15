package keeper

import (
	"time"

	assettypes "github.com/comdex-official/comdex/x/asset/types"
	auctiontypes "github.com/comdex-official/comdex/x/auction/types"
	"github.com/comdex-official/comdex/x/liquidation/types"
	vaulttypes "github.com/comdex-official/comdex/x/vault/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	protobuftypes "github.com/gogo/protobuf/types"
)

func (k Keeper) LiquidateVaults(ctx sdk.Context) error {
	vaults := k.GetVaults(ctx)
	for _, vault := range vaults {
		pair, found := k.GetPair(ctx, vault.PairID)
		if !found {
			continue
		}
		liquidationRatio := pair.LiquidationRatio
		assetIn, found := k.GetAsset(ctx, pair.AssetIn)
		if !found {
			continue
		}

		assetOut, found := k.GetAsset(ctx, pair.AssetOut)
		if !found {
			continue
		}
		collateralizationRatio, err := k.CalculateCollaterlizationRatio(ctx, vault.AmountIn, assetIn, vault.AmountOut, assetOut)
		if err != nil {
			continue
		}
		if sdk.Dec.LT(collateralizationRatio, liquidationRatio) {
			err := k.CreateLockedVault(ctx, vault, collateralizationRatio)
			if err != nil {
				return err
			}
			k.DeleteVault(ctx, vault.ID)
		}
	}
	return nil
}

func (k Keeper) CreateLockedVault(ctx sdk.Context, vault vaulttypes.Vault, collateralizationRatio sdk.Dec) error {

	lockedVaultId := k.GetLockedVaultID(ctx)

	var (
		value = types.LockedVault{
			LockedVaultId:                lockedVaultId,
			OriginalVaultId:              vault.ID,
			PairId:                       vault.PairID,
			Owner:                        vault.Owner,
			AmountIn:                     vault.AmountIn,
			AmountOut:                    vault.AmountOut,
			Initiator:                    types.ModuleName,
			IsAuctionComplete:            false,
			IsAuctionInProgress:          false,
			CrAtLiquidation:              collateralizationRatio,
			CurrentCollaterlisationRatio: collateralizationRatio,
			CollateralToBeAuctioned:      nil,
			LiquidationTimestamp:         time.Time{},
			SellOffHistory:               nil,
		}
	)
	k.SetLockedVault(ctx, value)
	k.SetLockedVaultID(ctx, lockedVaultId+1)

	//Create a new Data Structure with the current Params
	//Set nil for all the values not available right now
	//New function will loop over locked vaults to set all values so that they can be auctioned, seperately
	//Auction will then use the selloff amount set by lockedvault function to update params .
	//Unliquidate will take place after all the vents trigger.

	//
	return nil

}

//for first time to update the coollateralization value & sell off amount
//and if auction is complete and cr is less than 1.6
func (k Keeper) UpdateLockedVaults(ctx sdk.Context) error {
	lockedVaults := k.GetLockedVaults(ctx)
	if len(lockedVaults) == 0 {
		return nil
	}
	for _, lockedVault := range lockedVaults {
		v, _ := sdk.NewDecFromStr("1.6")

		if (!lockedVault.IsAuctionInProgress && !lockedVault.IsAuctionComplete) || (lockedVault.IsAuctionComplete && lockedVault.CurrentCollaterlisationRatio.LTE(v)) {
			pair, found := k.GetPair(ctx, lockedVault.PairId)
			if !found {
				continue
			}
			assetIn, found := k.GetAsset(ctx, pair.AssetIn)
			if !found {
				continue
			}
			assetOut, found := k.GetAsset(ctx, pair.AssetOut)
			if !found {
				continue
			}
			collateralizationRatio, err := k.CalculateCollaterlizationRatio(ctx, lockedVault.AmountIn, assetIn, lockedVault.AmountOut, assetOut)
			if err != nil {
				continue
			}
			//lockedVault.CurrentCollaterlisationRatio = collateralizationRatio

			assetInPrice, _ := k.GetPriceForAsset(ctx, assetIn.Id)
			assetOutPrice, _ := k.GetPriceForAsset(ctx, assetOut.Id)

			totalIn := lockedVault.AmountIn.Mul(sdk.NewIntFromUint64(assetInPrice)).ToDec()
			totalOut := lockedVault.AmountOut.Mul(sdk.NewIntFromUint64(assetOutPrice)).ToDec()

			var selloffAmount sdk.Dec
			var collateralToBeAuctioned sdk.Dec
			var unliquidatePoint, dividingFactor sdk.Dec
			unliquidatePoint, _ = sdk.NewDecFromStr("1.6")
			dividingFactor, _ = sdk.NewDecFromStr("0.28")
			assetOutatLiquidatePoint := totalOut.Mul(unliquidatePoint)
			collateralIn := totalIn
			assetsDifference := assetOutatLiquidatePoint.Sub(collateralIn)
			selloffAmount = (assetsDifference).Quo(dividingFactor)
			if selloffAmount.GTE(totalIn) {
				collateralToBeAuctioned = totalIn
			} else {

				collateralToBeAuctioned = selloffAmount
			}
			updatedLockedVault := types.LockedVault{
				LockedVaultId:                lockedVault.LockedVaultId,
				OriginalVaultId:              lockedVault.OriginalVaultId,
				PairId:                       lockedVault.PairId,
				Owner:                        lockedVault.Owner,
				AmountIn:                     lockedVault.AmountIn,
				AmountOut:                    lockedVault.AmountOut,
				Initiator:                    lockedVault.Initiator,
				IsAuctionComplete:            lockedVault.IsAuctionComplete,
				IsAuctionInProgress:          lockedVault.IsAuctionInProgress,
				CrAtLiquidation:              lockedVault.CrAtLiquidation,
				CurrentCollaterlisationRatio: collateralizationRatio,
				CollateralToBeAuctioned:      &collateralToBeAuctioned,
				LiquidationTimestamp:         lockedVault.LiquidationTimestamp,
				SellOffHistory:               lockedVault.SellOffHistory,
			}

			k.SetLockedVault(ctx, updatedLockedVault)

		}

	}
	return nil
}

func (k Keeper) UnliquidateLockedVaults(ctx sdk.Context) error {
	lockedVaults := k.GetLockedVaults(ctx)
	if len(lockedVaults) == 0 {
		return nil
	}
	for _, lockedVault := range lockedVaults {
		v, _ := sdk.NewDecFromStr("1.6")
		//also calculate the current collaterlization ration to ensure there is no sudden changes
		if lockedVault.IsAuctionComplete && lockedVault.CurrentCollaterlisationRatio.GTE(v) {

			var (
				id    = k.GetVaultID(ctx)
				vault = vaulttypes.Vault{
					ID:        id + 1,
					PairID:    lockedVault.PairId,
					Owner:     lockedVault.Owner,
					AmountIn:  lockedVault.AmountIn,
					AmountOut: lockedVault.AmountOut,
				}
			)
			k.SetVaultID(ctx, vault.ID)
			k.SetVault(ctx, vault)
			userAddress, err := sdk.AccAddressFromBech32(lockedVault.Owner)
			if err != nil {
				return err
			}
			k.DeleteVaultForAddressByPair(ctx, userAddress, lockedVault.PairId)

			k.SetVaultForAddressByPair(ctx, userAddress, lockedVault.PairId, vault.ID)
			//Save Locked vault historical data in a store
			//Set Auctioned historical in a store seperately
			k.DeleteLockedVault(ctx, lockedVault.LockedVaultId)
		}

	}

	return nil
}

func (k Keeper) GetModAccountBalances(ctx sdk.Context, accountName string, denom string) sdk.Int {
	macc := k.GetModuleAccount(ctx, accountName)
	return k.GetBalance(ctx, macc.GetAddress(), denom).Amount
}

func (k *Keeper) GetLockedVaultID(ctx sdk.Context) uint64 {
	var (
		store = k.Store(ctx)
		key   = types.LockedVaultIdKey
		value = store.Get(key)
	)

	if value == nil {
		return 0
	}

	var id protobuftypes.UInt64Value
	k.cdc.MustUnmarshal(value, &id)

	return id.GetValue()
}

func (k *Keeper) SetLockedVaultID(ctx sdk.Context, id uint64) {
	var (
		store = k.Store(ctx)
		key   = types.LockedVaultIdKey
		value = k.cdc.MustMarshal(
			&protobuftypes.UInt64Value{
				Value: id,
			},
		)
	)
	store.Set(key, value)
}

func (k *Keeper) SetLockedVault(ctx sdk.Context, locked_vault types.LockedVault) {
	var (
		store = k.Store(ctx)
		key   = types.LockedVaultKey(locked_vault.LockedVaultId)
		value = k.cdc.MustMarshal(&locked_vault)
	)
	store.Set(key, value)
}

func (k *Keeper) DeleteLockedVault(ctx sdk.Context, id uint64) {
	var (
		store = k.Store(ctx)
		key   = types.LockedVaultKey(id)
	)
	store.Delete(key)
}

func (k *Keeper) GetLockedVault(ctx sdk.Context, id uint64) (locked_vault types.LockedVault, found bool) {
	var (
		store = k.Store(ctx)
		key   = types.LockedVaultKey(id)
		value = store.Get(key)
	)

	if value == nil {
		return locked_vault, false
	}

	k.cdc.MustUnmarshal(value, &locked_vault)
	return locked_vault, true
}

func (k *Keeper) GetLockedVaults(ctx sdk.Context) (locked_vaults []types.LockedVault) {
	var (
		store = k.Store(ctx)
		iter  = sdk.KVStorePrefixIterator(store, types.LockedVaultKeyPrefix)
	)

	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var locked_vault types.LockedVault
		k.cdc.MustUnmarshal(iter.Value(), &locked_vault)
		locked_vaults = append(locked_vaults, locked_vault)
	}

	return locked_vaults
}

func (k *Keeper) SetFlagIsAuctionInProgress(ctx sdk.Context, id uint64, flag bool) error {

	locked_vault, found := k.GetLockedVault(ctx, id)
	if !found {
		return types.LockedVaultDoesNotExist
	}
	locked_vault.IsAuctionInProgress = flag
	k.SetLockedVault(ctx, locked_vault)
	return nil
}

func (k *Keeper) SetFlagIsAuctionComplete(ctx sdk.Context, id uint64, flag bool) error {

	locked_vault, found := k.GetLockedVault(ctx, id)
	if !found {
		return types.LockedVaultDoesNotExist
	}
	locked_vault.IsAuctionComplete = flag
	k.SetLockedVault(ctx, locked_vault)
	return nil
}

func (k *Keeper) UpdateAssetQuantitiesInLockedVault(
	ctx sdk.Context,
	collateral_auction auctiontypes.CollateralAuction,
	amountIn sdk.Int,
	assetIn assettypes.Asset,
	amountOut sdk.Int,
	assetOut assettypes.Asset,
) error {

	locked_vault, found := k.GetLockedVault(ctx, collateral_auction.LockedVaultId)
	if !found {
		return types.LockedVaultDoesNotExist
	}
	updatedAmountIn := locked_vault.AmountIn.Sub(amountIn)
	updatedAmountOut := locked_vault.AmountOut.Sub(amountOut)
	updatedCollateralizationRatio, _ := k.CalculateCollaterlizationRatio(ctx, updatedAmountIn, assetIn, updatedAmountOut, assetOut)

	locked_vault.AmountIn = updatedAmountIn
	locked_vault.AmountOut = updatedAmountOut
	locked_vault.CurrentCollaterlisationRatio = updatedCollateralizationRatio
	locked_vault.SellOffHistory = append(locked_vault.SellOffHistory, collateral_auction.String())
	k.SetLockedVault(ctx, locked_vault)
	return nil
}
