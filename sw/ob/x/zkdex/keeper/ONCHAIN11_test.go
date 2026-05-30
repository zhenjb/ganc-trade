package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"ob/x/zkdex/keeper"
	"ob/x/zkdex/types"
)

func TestONCHAIN11StateQueries(t *testing.T) {
	f := initDepositFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	require.NoError(t, f.keeper.SetStateRoot(f.ctx, "0xstate-root-11"))
	require.NoError(t, f.keeper.SetNullifierUsed(f.ctx, "0xnullifier-11"))

	withdrawRecord := types.WithdrawRecord{
		WithdrawId:  "wd-onchain-11",
		Owner:       "owner-11",
		Denom:       "uusdc",
		Amount:      "40",
		Destination: "owner-11",
		Nullifier:   "0xnullifier-11",
		Claimed:     false,
	}
	require.NoError(t, f.keeper.SetWithdrawRecord(f.ctx, withdrawRecord.WithdrawId, withdrawRecord))

	moduleAddr := f.keeper.GetModuleAccountAddress().String()
	f.mockBank.escrowedCoins[moduleAddr] = sdk.NewCoins(
		sdk.NewInt64Coin("stake", 1000),
		sdk.NewInt64Coin("uusdc", 40),
	)

	rootResp, err := qs.CurrentStateRoot(f.ctx, &types.QueryCurrentStateRootRequest{})
	require.NoError(t, err)
	require.Equal(t, "0xstate-root-11", rootResp.StateRoot)

	nullifierResp, err := qs.NullifierUsed(f.ctx, &types.QueryNullifierUsedRequest{Nullifier: "0xnullifier-11"})
	require.NoError(t, err)
	require.True(t, nullifierResp.Used)

	missingNullifierResp, err := qs.NullifierUsed(f.ctx, &types.QueryNullifierUsedRequest{Nullifier: "0xmissing"})
	require.NoError(t, err)
	require.False(t, missingNullifierResp.Used)

	withdrawResp, err := qs.WithdrawRecord(f.ctx, &types.QueryWithdrawRecordRequest{WithdrawId: "wd-onchain-11"})
	require.NoError(t, err)
	require.Equal(t, &withdrawRecord, withdrawResp.Record)

	allBalanceResp, err := qs.ModuleAccountBalance(f.ctx, &types.QueryModuleAccountBalanceRequest{})
	require.NoError(t, err)
	require.Equal(t, "1000stake,40uusdc", allBalanceResp.Balance)

	denomBalanceResp, err := qs.ModuleAccountBalance(f.ctx, &types.QueryModuleAccountBalanceRequest{Denom: "uusdc"})
	require.NoError(t, err)
	require.Equal(t, "40uusdc", denomBalanceResp.Balance)

	zeroDenomBalanceResp, err := qs.ModuleAccountBalance(f.ctx, &types.QueryModuleAccountBalanceRequest{Denom: "uatom"})
	require.NoError(t, err)
	require.Equal(t, "0uatom", zeroDenomBalanceResp.Balance)
}

func TestONCHAIN11StateQueriesRejectEmptyKeys(t *testing.T) {
	f := initDepositFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	_, err := qs.WithdrawRecord(f.ctx, &types.QueryWithdrawRecordRequest{})
	require.Error(t, err)
	require.ErrorContains(t, err, "withdraw id cannot be empty")

	_, err = qs.NullifierUsed(f.ctx, &types.QueryNullifierUsedRequest{})
	require.Error(t, err)
	require.ErrorContains(t, err, "nullifier cannot be empty")
}
