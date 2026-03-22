package keeper

import (
	"ob/x/dex/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Withdraw sends coins from the dex module account back to the user.
func (k Keeper) Withdraw(ctx sdk.Context, creator string, amount string, denom string) error {
	userAddr, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return types.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}

	amt, ok := math.NewIntFromString(amount)
	if !ok || !amt.IsPositive() {
		return types.ErrInvalidAmount.Wrap("amount must be a positive integer")
	}

	coins := sdk.NewCoins(sdk.NewCoin(denom, amt))

	return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, userAddr, coins)
}
