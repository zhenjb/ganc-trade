package keeper_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"ob/testutil/sample"
	"ob/x/zkdex/keeper"
	"ob/x/zkdex/types"
)

func TestMsgSubmitBatchProofValidationAccepts(t *testing.T) {
	f := initFixture(t)
	creator := sample.AccAddress()
	settlementUpdate, batchCommitments, proofBundle := validMsgSubmitBatchProof(t, f)

	var gotVerifierUpdate []byte
	var gotProofBundle []byte
	k := f.keeper.WithProofVerifier(types.ProofVerifierFunc(func(update []byte, proof []byte) bool {
		gotVerifierUpdate = append([]byte(nil), update...)
		gotProofBundle = append([]byte(nil), proof...)
		return true
	}))
	msgServer := keeper.NewMsgServerImpl(k)

	resp, err := msgServer.SubmitBatchProof(f.ctx, &types.MsgSubmitBatchProof{
		Creator:          creator,
		SettlementUpdate: settlementUpdate,
		BatchCommitments: batchCommitments,
		ProofBundle:      proofBundle,
	})
	require.NoError(t, err)
	require.True(t, resp.Accepted)
	require.Equal(t, []string{
		"0xrootA",
		"0xrootB",
		"0xdepositsRoot",
		"0xwithdrawalsRoot",
		"0xnullifiersRoot",
		"0xwithdrawOutputsRoot",
	}, resp.PublicInputs)
	require.Equal(t, proofBundle, gotProofBundle)
	// kiểm tra xem dữ liệu JSON mà Keeper tự đóng gói để ném vào hàm Verify có chứa đúng mảng publicInputs mong muốn hay không
	require.Contains(t, string(gotVerifierUpdate), `"publicInputs":["0xrootA","0xrootB","0xdepositsRoot","0xwithdrawalsRoot","0xnullifiersRoot","0xwithdrawOutputsRoot"]`)

	stateRoot, err := f.keeper.GetStateRoot(f.ctx)
	require.NoError(t, err)
	require.Equal(t, "0xrootB", stateRoot)

	processed, err := f.keeper.IsDepositProcessed(f.ctx, "dep-1")
	require.NoError(t, err)
	require.True(t, processed)
	depositRecord, err := f.keeper.GetDepositRecord(f.ctx, "dep-1")
	require.NoError(t, err)
	require.True(t, depositRecord.Processed)

	nullifierUsed, err := f.keeper.IsNullifierUsed(f.ctx, "0xmocknullifier")
	require.NoError(t, err)
	require.True(t, nullifierUsed)

	withdrawRecord, err := f.keeper.GetWithdrawRecord(f.ctx, "wd-1")
	require.NoError(t, err)
	require.Equal(t, types.WithdrawRecord{
		WithdrawId:  "wd-1",
		Owner:       "cosmos1alice",
		Denom:       "uusdc",
		Amount:      "40",
		Destination: "cosmos1alice",
		Nullifier:   "0xmocknullifier",
		Claimed:     false,
	}, withdrawRecord)

	batchRecord, err := f.keeper.GetBatchRecord(f.ctx, "batch-1")
	require.NoError(t, err)
	require.Equal(t, "0xrootA", batchRecord.OldStateRoot)
	require.Equal(t, "0xrootB", batchRecord.NewStateRoot)
	require.Equal(t, []string{"dep-1"}, batchRecord.DepositIds)
	require.Equal(t, []string{"wd-1"}, batchRecord.WithdrawIds)
}

func TestMsgSubmitBatchProofValidationRejectsBadInputs(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(t *testing.T, f *fixture, settlementUpdate *types.SettlementUpdate, batchCommitments *types.BatchCommitments, proofBundle *[]byte)
		errText string
	}{
		{
			name: "old root mismatch",
			mutate: func(t *testing.T, f *fixture, settlementUpdate *types.SettlementUpdate, batchCommitments *types.BatchCommitments, proofBundle *[]byte) {
				settlementUpdate.OldStateRoot = "0xwrong"
				*proofBundle = proofBundleJSON(t, []string{"0xwrong", settlementUpdate.NewStateRoot, batchCommitments.DepositsRoot, batchCommitments.WithdrawalsRoot, batchCommitments.NullifiersRoot, batchCommitments.WithdrawOutputsRoot})
			},
			errText: "oldStateRoot mismatch",
		},
		{
			name: "processed deposit",
			mutate: func(t *testing.T, f *fixture, settlementUpdate *types.SettlementUpdate, batchCommitments *types.BatchCommitments, proofBundle *[]byte) {
				require.NoError(t, f.keeper.SetDepositProcessed(f.ctx, "dep-1"))
			},
			errText: "already processed",
		},
		{
			name: "used nullifier",
			mutate: func(t *testing.T, f *fixture, settlementUpdate *types.SettlementUpdate, batchCommitments *types.BatchCommitments, proofBundle *[]byte) {
				require.NoError(t, f.keeper.SetNullifierUsed(f.ctx, "0xmocknullifier"))
			},
			errText: "already used",
		},
		{
			name: "existing withdraw record",
			mutate: func(t *testing.T, f *fixture, settlementUpdate *types.SettlementUpdate, batchCommitments *types.BatchCommitments, proofBundle *[]byte) {
				require.NoError(t, f.keeper.SetWithdrawRecord(f.ctx, "wd-1", types.WithdrawRecord{
					WithdrawId: "wd-1",
					Owner:      "cosmos1alice",
					Denom:      "uusdc",
					Amount:     "40",
				}))
			},
			errText: "already exists",
		},
		{
			name: "existing batch record",
			mutate: func(t *testing.T, f *fixture, settlementUpdate *types.SettlementUpdate, batchCommitments *types.BatchCommitments, proofBundle *[]byte) {
				require.NoError(t, f.keeper.SetBatchRecord(f.ctx, "batch-1", types.BatchRecord{
					BatchId: "batch-1",
				}))
			},
			errText: "already exists",
		},
		{
			name: "proof public inputs mismatch",
			mutate: func(t *testing.T, f *fixture, settlementUpdate *types.SettlementUpdate, batchCommitments *types.BatchCommitments, proofBundle *[]byte) {
				*proofBundle = proofBundleJSON(t, []string{settlementUpdate.OldStateRoot, "0xtampered", batchCommitments.DepositsRoot, batchCommitments.WithdrawalsRoot, batchCommitments.NullifiersRoot, batchCommitments.WithdrawOutputsRoot})
			},
			errText: "publicInputs do not match",
		},
		{
			name: "verifier rejects",
			mutate: func(t *testing.T, f *fixture, settlementUpdate *types.SettlementUpdate, batchCommitments *types.BatchCommitments, proofBundle *[]byte) {
			},
			errText: "proof verification failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := initFixture(t)
			settlementUpdate, batchCommitments, proofBundle := validMsgSubmitBatchProof(t, f)
			tc.mutate(t, f, &settlementUpdate, &batchCommitments, &proofBundle)

			verifier := types.ProofVerifier(types.StubProofVerifier{Accept: true})
			if tc.name == "verifier rejects" {
				verifier = types.StubProofVerifier{Accept: false}
			}
			msgServer := keeper.NewMsgServerImpl(f.keeper.WithProofVerifier(verifier))

			_, err := msgServer.SubmitBatchProof(f.ctx, &types.MsgSubmitBatchProof{
				Creator:          sample.AccAddress(),
				SettlementUpdate: settlementUpdate,
				BatchCommitments: batchCommitments,
				ProofBundle:      proofBundle,
			})
			require.Error(t, err)
			require.ErrorContains(t, err, tc.errText)
		})
	}
}

func validMsgSubmitBatchProof(t *testing.T, f *fixture) (types.SettlementUpdate, types.BatchCommitments, []byte) {
	t.Helper()

	require.NoError(t, f.keeper.SetStateRoot(f.ctx, "0xrootA"))
	require.NoError(t, f.keeper.SetDepositRecord(f.ctx, "dep-1", types.DepositRecord{
		DepositId:     "dep-1",
		Owner:         "cosmos1alice",
		Denom:         "uusdc",
		Amount:        "100",
		Processed:     false,
		CreatedHeight: 1,
	}))

	settlementUpdate := types.SettlementUpdate{
		BatchId:      "batch-1",
		OldStateRoot: "0xrootA",
		NewStateRoot: "0xrootB",
		Deposits: []*types.SettlementDeposit{
			{
				DepositId: "dep-1",
				Owner:     "cosmos1alice",
				Denom:     "uusdc",
				Amount:    "100",
			},
		},
		Withdrawals: []*types.SettlementWithdrawal{
			{
				WithdrawId:      "wd-1",
				Owner:           "cosmos1alice",
				Denom:           "uusdc",
				Amount:          "40",
				Destination:     "cosmos1alice",
				DestinationHash: "0xmockdestinationhash",
				Nullifier:       "0xmocknullifier",
			},
		},
	}
	batchCommitments := types.BatchCommitments{
		DepositsRoot:        "0xdepositsRoot",
		WithdrawalsRoot:     "0xwithdrawalsRoot",
		NullifiersRoot:      "0xnullifiersRoot",
		WithdrawOutputsRoot: "0xwithdrawOutputsRoot",
	}
	proofBundle := proofBundleJSON(t, []string{
		settlementUpdate.OldStateRoot,
		settlementUpdate.NewStateRoot,
		batchCommitments.DepositsRoot,
		batchCommitments.WithdrawalsRoot,
		batchCommitments.NullifiersRoot,
		batchCommitments.WithdrawOutputsRoot,
	})

	return settlementUpdate, batchCommitments, proofBundle
}

func proofBundleJSON(t *testing.T, publicInputs []string) []byte {
	t.Helper()

	bz, err := json.Marshal(map[string]any{
		"proof":             "0xmockproof",
		"publicInputs":      publicInputs,
		"verificationKeyId": "v1",
	})
	require.NoError(t, err)
	return bz
}
