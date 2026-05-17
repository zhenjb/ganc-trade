package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/store/types"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stretchr/testify/require"

	"ob/x/zkdex/keeper"
	module "ob/x/zkdex/module"
	"ob/x/zkdex/types"
)

type queryDepositFixture struct {
	ctx          context.Context
	keeper       keeper.Keeper
	addressCodec address.Codec
}

func initQueryDepositFixture(t *testing.T) *queryDepositFixture {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

	authority := authtypes.NewModuleAddress(types.GovModuleName)

	k := keeper.NewKeeper(
		storeService,
		encCfg.Codec,
		addressCodec,
		authority,
		nil,
		nil,
	)

	require.NoError(t, k.Params.Set(ctx, types.DefaultParams()))

	return &queryDepositFixture{
		ctx:          ctx,
		keeper:       k,
		addressCodec: addressCodec,
	}
}

func TestQueryDepositRecord(t *testing.T) {
	f := initQueryDepositFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	record := types.DepositRecord{
		DepositId:     "dep-1",
		Owner:         "cosmos1alice...",
		Denom:         "uusdc",
		Amount:        "100",
		Processed:     false,
		CreatedHeight: 12345,
	}
	require.NoError(t, f.keeper.SetDepositRecord(f.ctx, record.DepositId, record))
	t.Logf("SETUP: saved deposit record: deposit_id=%s owner=%s amount=%s%s processed=%v created_height=%d",
		record.DepositId,
		record.Owner,
		record.Amount,
		record.Denom,
		record.Processed,
		record.CreatedHeight,
	)

	resp, err := qs.DepositRecord(f.ctx, &types.QueryDepositRecordRequest{DepositId: record.DepositId})
	require.NoError(t, err)
	require.Equal(t, &record, resp.Record)
	t.Logf("QUERY DepositRecord(%q): record=%+v", record.DepositId, resp.Record)

	processedResp, err := qs.DepositProcessed(f.ctx, &types.QueryDepositProcessedRequest{DepositId: record.DepositId})
	require.NoError(t, err)
	require.False(t, processedResp.Processed)
	t.Logf("QUERY DepositProcessed(%q) before settlement: processed=%v", record.DepositId, processedResp.Processed)

	require.NoError(t, f.keeper.SetDepositProcessed(f.ctx, record.DepositId))
	t.Logf("ACTION: marked deposit_id=%s as processed", record.DepositId)

	resp, err = qs.DepositRecord(f.ctx, &types.QueryDepositRecordRequest{DepositId: record.DepositId})
	require.NoError(t, err)
	require.True(t, resp.Record.Processed)
	t.Logf("QUERY DepositRecord(%q) after settlement: record=%+v", record.DepositId, resp.Record)

	processedResp, err = qs.DepositProcessed(f.ctx, &types.QueryDepositProcessedRequest{DepositId: record.DepositId})
	require.NoError(t, err)
	require.True(t, processedResp.Processed)
	t.Logf("QUERY DepositProcessed(%q) after settlement: processed=%v", record.DepositId, processedResp.Processed)
}

func TestQueryDepositRecordErrors(t *testing.T) {
	f := initQueryDepositFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	// client gửi request nil
	_, err := qs.DepositRecord(f.ctx, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	t.Logf("ERROR CASE DepositRecord(nil): code=%s err=%v", status.Code(err), err)

	// DepositRecord với deposit_id rỗng
	_, err = qs.DepositRecord(f.ctx, &types.QueryDepositRecordRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	t.Logf("ERROR CASE DepositRecord(empty deposit_id): code=%s err=%v", status.Code(err), err)

	// DepositRecord với ID không tồn tại
	_, err = qs.DepositRecord(f.ctx, &types.QueryDepositRecordRequest{DepositId: "missing"})
	require.Equal(t, codes.NotFound, status.Code(err))
	t.Logf("ERROR CASE DepositRecord(missing): code=%s err=%v", status.Code(err), err)

	// query processed status nhưng request nil
	_, err = qs.DepositProcessed(f.ctx, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	t.Logf("ERROR CASE DepositProcessed(nil): code=%s err=%v", status.Code(err), err)

	// DepositProcessed với deposit_id rỗng
	_, err = qs.DepositProcessed(f.ctx, &types.QueryDepositProcessedRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	t.Logf("ERROR CASE DepositProcessed(empty deposit_id): code=%s err=%v", status.Code(err), err)
}