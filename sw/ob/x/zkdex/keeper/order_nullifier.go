package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"

	"ob/x/zkdex/types"
)

func (k Keeper) SetOrderNullifierUsed(ctx context.Context, nullifier string) error {
	normalized, err := types.NormalizeOrderNullifier(nullifier)
	if err != nil {
		return err
	}
	return k.OrderNullifierUsed.Set(ctx, normalized, true)
}

func (k Keeper) IsOrderNullifierUsed(ctx context.Context, nullifier string) (bool, error) {
	normalized, err := types.NormalizeOrderNullifier(nullifier)
	if err != nil {
		return false, err
	}
	used, err := k.OrderNullifierUsed.Get(ctx, normalized)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return used, nil
}

func (k Keeper) GetAllOrderNullifiersUsed(ctx context.Context) ([]string, error) {
	nullifiers := make([]string, 0)
	if err := k.OrderNullifierUsed.Walk(ctx, nil, func(nullifier string, used bool) (bool, error) {
		if used {
			nullifiers = append(nullifiers, nullifier)
		}
		return false, nil
	}); err != nil {
		return nil, err
	}
	return nullifiers, nil
}
