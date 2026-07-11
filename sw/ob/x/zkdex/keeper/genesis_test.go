package keeper_test

import (
	"testing"

	"ob/x/zkdex/types"

	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	batchRecord := types.BatchRecord{
		BatchId:              "batch-genesis-1",
		OldStateRoot:         types.DefaultStateRoot,
		NewStateRoot:         "0xdddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		TradeIds:             []string{"trd-genesis-1"},
		TradesRoot:           "0x5555555555555555555555555555555555555555555555555555555555555555",
		OrdersRoot:           "0x6666666666666666666666666666666666666666666666666666666666666666",
		TradeBatchCommitment: "0x5452442d47454e455349532d31",
		TradeCount:           1,
	}
	genesisState := types.GenesisState{
		Params:           types.DefaultParams(),
		CurrentStateRoot: types.DefaultStateRoot,
		BatchRecords:     []*types.BatchRecord{&batchRecord},
	}

	f := initFixture(t)
	err := f.keeper.InitGenesis(f.ctx, genesisState)
	require.NoError(t, err)
	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.NotNil(t, got)

	require.Equal(t, genesisState.Params, got.Params)
	require.Equal(t, genesisState.CurrentStateRoot, got.CurrentStateRoot)
	require.Equal(t, genesisState.BatchRecords, got.BatchRecords)
}
