package keeper

import (
	"context"
	"fmt"

	"ob/x/zkdex/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	if k.authKeeper != nil {
		// Ensures the zkdex module account exists in x/auth (must be listed in app ModuleAccountPermissions).
		if macc := k.authKeeper.GetModuleAccount(ctx, types.ModuleName); macc == nil {
			return fmt.Errorf("zkdex module account %q is not registered in auth module permissions", types.ModuleName)
		}
	}

	// 1. Lưu giá trị CurrentStateRoot từ genState vào KVStore
    if err := k.SetStateRoot(ctx, genState.CurrentStateRoot); err != nil {
        return err
    }

	return k.Params.Set(ctx, genState.Params)
}

// ExportGenesis returns the module's exported genesis.
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	var err error

	genesis := types.DefaultGenesis()
	// 1. Lấy Params hiện tại
	genesis.Params, err = k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	// 2. Lấy State Root hiện tại từ database để gán vào file export
    root, err := k.GetStateRoot(ctx)
    if err != nil {
        return nil, err
    }
    genesis.CurrentStateRoot = root

	return genesis, nil
}
