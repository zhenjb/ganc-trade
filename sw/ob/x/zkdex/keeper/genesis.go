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

    // 2. Set deposit records
    for _, record := range genState.DepositRecords {
        if err := k.SetDepositRecord(ctx, record.DepositId, *record); err != nil {
            return err
        }
    }

    // 3. Set withdraw records
    for _, record := range genState.WithdrawRecords {
        if err := k.SetWithdrawRecord(ctx, record.WithdrawId, *record); err != nil {
            return err
        }
    }

    // 4. Set nullifiers used
    for _, nullifier := range genState.NullifierUsed {
        if err := k.SetNullifierUsed(ctx, nullifier); err != nil {
            return err
        }
    }

    // 5. Set deposits processed
    for _, depositId := range genState.DepositProcessed {
        if err := k.SetDepositProcessed(ctx, depositId); err != nil {
            return err
        }
    }

    // 6. Set batch records
    for _, record := range genState.BatchRecords {
        if err := k.SetBatchRecord(ctx, record.BatchId, *record); err != nil {
            return err
        }
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

    // For now, export empty records (can be extended later)
    genesis.DepositRecords = nil
    genesis.WithdrawRecords = nil
    genesis.NullifierUsed = nil
    genesis.DepositProcessed = nil
    genesis.BatchRecords = nil

	return genesis, nil
}
