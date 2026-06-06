package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"ob/testutil/sample"
	"ob/x/zkdex/types"
)

func TestMsgClaimWithdraw_FullFlow(t *testing.T) {
	f := initDepositFixture(t)
	alice := sample.AccAddress()

	require.NoError(t, f.keeper.SetWithdrawRecord(f.ctx, "wd-1", types.WithdrawRecord{
		WithdrawId:  "wd-1",
		Owner:       alice,
		Denom:       "uusdc",
		Amount:      "40",
		Destination: alice,
		Nullifier:   "0xmocknullifier",
		Claimed:     false,
	}))

	resp, err := f.msgServer.ClaimWithdraw(f.ctx, &types.MsgClaimWithdraw{
		Creator:    alice,
		WithdrawId: "wd-1",
	})
	require.NoError(t, err)
	require.NotNil(t, resp.WithdrawRecord)
	require.True(t, resp.WithdrawRecord.Claimed)
	require.Equal(t, "40uusdc", f.mockBank.releasedCoins[alice].String())

	stored, err := f.keeper.GetWithdrawRecord(f.ctx, "wd-1")
	require.NoError(t, err)
	require.True(t, stored.Claimed)

	_, err = f.msgServer.ClaimWithdraw(f.ctx, &types.MsgClaimWithdraw{
		Creator:    alice,
		WithdrawId: "wd-1",
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "already claimed")
}

func TestMsgClaimWithdraw_RejectsInvalidInputs(t *testing.T) {
	f := initDepositFixture(t)
	alice := sample.AccAddress()
	bob := sample.AccAddress()

	require.NoError(t, f.keeper.SetWithdrawRecord(f.ctx, "wd-1", types.WithdrawRecord{
		WithdrawId:  "wd-1",
		Owner:       alice,
		Denom:       "uusdc",
		Amount:      "40",
		Destination: alice,
		Nullifier:   "0xmocknullifier",
		Claimed:     false,
	}))

	_, err := f.msgServer.ClaimWithdraw(f.ctx, &types.MsgClaimWithdraw{
		Creator:    alice,
		WithdrawId: "missing",
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "not found")

	_, err = f.msgServer.ClaimWithdraw(f.ctx, &types.MsgClaimWithdraw{
		Creator:    bob,
		WithdrawId: "wd-1",
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "creator must match withdraw destination")
}
