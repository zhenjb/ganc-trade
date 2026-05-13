package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"ob/x/zkdex/types"
)

func TestDepositAndWithdrawRecords(t *testing.T) {
    f := initFixture(t) 

    // --- TEST DEPOSIT ---
    aliceDeposit := types.DepositRecord{
        DepositId:     "dep-1",
        Owner:         "cosmos1alice...",
        Denom:         "uusdc",
        Amount:        "100",
        Processed:     false,
        CreatedHeight: 12345,
    }
    err := f.keeper.SetDepositRecord(f.ctx, aliceDeposit.DepositId, aliceDeposit)
    require.NoError(t, err)

    gotDeposit, _ := f.keeper.GetDepositRecord(f.ctx, "dep-1")
    t.Logf("✅ Deposit Record Saved: %+v", gotDeposit) // In ra dữ liệu Deposit

    // --- TEST WITHDRAW ---
    aliceWithdraw := types.WithdrawRecord{
       	WithdrawId:  "wd-1",
		Owner:       "cosmos1alice...",
		Denom:       "uusdc",
		Amount:      "40",
		Destination: "cosmos1alice...", 
		Nullifier:   "0x...",           
		Claimed:     false,
    }
    err = f.keeper.SetWithdrawRecord(f.ctx, aliceWithdraw.WithdrawId, aliceWithdraw)
    require.NoError(t, err)

    gotWithdraw, _ := f.keeper.GetWithdrawRecord(f.ctx, "wd-1")
    t.Logf("✅ Withdraw Record Saved: %+v", gotWithdraw) // In ra dữ liệu Withdraw
    
    // --- TEST NULLIFIER ---
    f.keeper.SetNullifierUsed(f.ctx, "0xabc123")
    used, _ := f.keeper.IsNullifierUsed(f.ctx, "0xabc123")
    t.Logf("✅ Nullifier 0xabc123 Used: %v", used)

    // --- TEST BATCH ---
    aliceBatch := types.BatchRecord{
        BatchId:      "batch-1",
        OldStateRoot: "0xrootA",
        NewStateRoot: "0xrootB",
        DepositIds:    []string{"dep-1"}, 
        WithdrawIds:   []string{"wd-1"},
        CreatedHeight: 12345,
    }

    err = f.keeper.SetBatchRecord(f.ctx, aliceBatch.BatchId, aliceBatch)
    require.NoError(t, err)

    gotBatch, err := f.keeper.GetBatchRecord(f.ctx, "batch-1")
    require.NoError(t, err)
    
    // In ra để quan sát
    t.Logf("✅ Batch Record Saved: %+v", gotBatch)
    require.Equal(t, aliceBatch, gotBatch)


    // --- TEST DEPOSIT PROCESSED ---
    // Giả sử khoản nạp 'dep-1' đã được đưa vào Batch thành công
    err = f.keeper.SetDepositProcessed(f.ctx, "dep-1")
    require.NoError(t, err)

    isProcessed, _ := f.keeper.IsDepositProcessed(f.ctx, "dep-1")
    t.Logf("✅ Deposit dep-1 Processed Status: %v", isProcessed)
}