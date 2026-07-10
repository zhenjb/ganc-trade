package keeper_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"ob/testutil/sample"
	"ob/x/zkdex/keeper"
	"ob/x/zkdex/types"
)

var (
	emptyPublicInputRootSentinel = "0x" + strings.Repeat("0", 64)
	oldStateRootA                = "0x" + strings.Repeat("a", 64)
	newStateRootB                = "0x" + strings.Repeat("b", 64)
	oldStateRootC                = "0x" + strings.Repeat("c", 64)
	newStateRootD                = "0x" + strings.Repeat("d", 64)
	depositsRoot                 = "0x" + strings.Repeat("1", 64)
	withdrawalsRoot              = "0x" + strings.Repeat("2", 64)
	nullifiersRoot               = "0x" + strings.Repeat("3", 64)
	withdrawOutputsRoot          = "0x" + strings.Repeat("4", 64)
	tradesRoot                   = "0x" + strings.Repeat("5", 64)
	ordersRoot                   = "0x" + strings.Repeat("6", 64)
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
	expectedPublicInputs := []string{
		oldStateRootA,
		newStateRootB,
		depositsRoot,
		withdrawalsRoot,
		nullifiersRoot,
		withdrawOutputsRoot,
		emptyPublicInputRootSentinel,
		emptyPublicInputRootSentinel,
	}
	require.Equal(t, expectedPublicInputs, resp.PublicInputs)
	logPublicInputs(t, "accepted deposit+withdraw no-trade batch public inputs", resp.PublicInputs)
	require.Equal(t, proofBundle, gotProofBundle)
	verifierInputJSON, err := json.MarshalIndent(json.RawMessage(gotVerifierUpdate), "", "  ")
	require.NoError(t, err)
	t.Logf("verifier input JSON sent to mock verifier:\n%s", verifierInputJSON)
	require.Contains(t, string(gotVerifierUpdate), `"publicInputs":["`+strings.Join(expectedPublicInputs, `","`)+`"]`)

	stateRoot, err := f.keeper.GetStateRoot(f.ctx)
	require.NoError(t, err)
	require.Equal(t, newStateRootB, stateRoot)

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
	require.Equal(t, oldStateRootA, batchRecord.OldStateRoot)
	require.Equal(t, newStateRootB, batchRecord.NewStateRoot)
	require.Equal(t, []string{"dep-1"}, batchRecord.DepositIds)
	require.Equal(t, []string{"wd-1"}, batchRecord.WithdrawIds)
}

func TestMsgSubmitBatchProofAcceptsTradeOnlyBatch(t *testing.T) {
	f := initFixture(t)
	creator := sample.AccAddress()
	settlementUpdate, batchCommitments, proofBundle := validTradeOnlyMsgSubmitBatchProof(t, f)

	var gotVerifierUpdate []byte
	k := f.keeper.WithProofVerifier(types.ProofVerifierFunc(func(update []byte, proof []byte) bool {
		gotVerifierUpdate = append([]byte(nil), update...)
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
	expectedPublicInputs := []string{
		oldStateRootC,
		newStateRootD,
		emptyPublicInputRootSentinel,
		emptyPublicInputRootSentinel,
		emptyPublicInputRootSentinel,
		emptyPublicInputRootSentinel,
		tradesRoot,
		ordersRoot,
	}
	require.Equal(t, expectedPublicInputs, resp.PublicInputs)
	logPublicInputs(t, "accepted trade-only batch public inputs", resp.PublicInputs)
	require.Contains(t, string(gotVerifierUpdate), `"tradeId":"trd-1"`)
	require.Contains(t, string(gotVerifierUpdate), `"tradeBatchCommitment":"0x5452442d41544f4d2d555344542d62617463682d39"`)
	require.Contains(t, string(gotVerifierUpdate), `"tradesRoot":"`+tradesRoot+`"`)
	require.Contains(t, string(gotVerifierUpdate), `"ordersRoot":"`+ordersRoot+`"`)

	stateRoot, err := f.keeper.GetStateRoot(f.ctx)
	require.NoError(t, err)
	require.Equal(t, newStateRootD, stateRoot)

	batchRecord, err := f.keeper.GetBatchRecord(f.ctx, "batch-9")
	require.NoError(t, err)
	require.Empty(t, batchRecord.DepositIds)
	require.Empty(t, batchRecord.WithdrawIds)
}

func TestMsgSubmitBatchProofAcceptsMixedDepositWithdrawTradeBatch(t *testing.T) {
	f := initFixture(t)
	creator := sample.AccAddress()
	settlementUpdate, batchCommitments, proofBundle := validMixedMsgSubmitBatchProof(t, f)

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
	expectedPublicInputs := []string{
		oldStateRootA,
		newStateRootB,
		depositsRoot,
		withdrawalsRoot,
		nullifiersRoot,
		withdrawOutputsRoot,
		tradesRoot,
		ordersRoot,
	}
	require.Equal(t, expectedPublicInputs, resp.PublicInputs)
	logPublicInputs(t, "accepted mixed deposit+withdraw+trade batch public inputs", resp.PublicInputs)
	require.Equal(t, proofBundle, gotProofBundle)
	require.Contains(t, string(gotVerifierUpdate), `"depositId":"dep-1"`)
	require.Contains(t, string(gotVerifierUpdate), `"withdrawId":"wd-1"`)
	require.Contains(t, string(gotVerifierUpdate), `"tradeId":"trd-1"`)
	require.Contains(t, string(gotVerifierUpdate), `"tradesRoot":"`+tradesRoot+`"`)
	require.Contains(t, string(gotVerifierUpdate), `"ordersRoot":"`+ordersRoot+`"`)
	require.Contains(t, string(gotVerifierUpdate), `"publicInputs":["`+strings.Join(expectedPublicInputs, `","`)+`"]`)

	stateRoot, err := f.keeper.GetStateRoot(f.ctx)
	require.NoError(t, err)
	require.Equal(t, newStateRootB, stateRoot)

	processed, err := f.keeper.IsDepositProcessed(f.ctx, "dep-1")
	require.NoError(t, err)
	require.True(t, processed)

	nullifierUsed, err := f.keeper.IsNullifierUsed(f.ctx, "0xmocknullifier")
	require.NoError(t, err)
	require.True(t, nullifierUsed)

	batchRecord, err := f.keeper.GetBatchRecord(f.ctx, "batch-1")
	require.NoError(t, err)
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
				settlementUpdate.OldStateRoot = "0x" + strings.Repeat("f", 64)
				*proofBundle = proofBundleJSON(t, noTradePublicInputs(*settlementUpdate, *batchCommitments))
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
				publicInputs := noTradePublicInputs(*settlementUpdate, *batchCommitments)
				publicInputs[1] = "0x" + strings.Repeat("e", 64)
				*proofBundle = proofBundleJSON(t, publicInputs)
			},
			errText: "publicInputs do not match",
		},
		{
			name: "bad public input length",
			mutate: func(t *testing.T, f *fixture, settlementUpdate *types.SettlementUpdate, batchCommitments *types.BatchCommitments, proofBundle *[]byte) {
				batchCommitments.DepositsRoot = "0xshort"
				*proofBundle = proofBundleJSON(t, noTradePublicInputs(*settlementUpdate, *batchCommitments))
			},
			errText: "public input 2 must be 32-byte hex",
		},
		{
			name: "no-trade bad trades root",
			mutate: func(t *testing.T, f *fixture, settlementUpdate *types.SettlementUpdate, batchCommitments *types.BatchCommitments, proofBundle *[]byte) {
				batchCommitments.TradesRoot = tradesRoot
				*proofBundle = proofBundleJSON(t, noTradePublicInputs(*settlementUpdate, *batchCommitments))
			},
			errText: "tradesRoot must be empty-root sentinel",
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

func validTradeOnlyMsgSubmitBatchProof(t *testing.T, f *fixture) (types.SettlementUpdate, types.BatchCommitments, []byte) {
	t.Helper()

	require.NoError(t, f.keeper.SetStateRoot(f.ctx, oldStateRootC))

	settlementUpdate := types.SettlementUpdate{
		BatchId:              "batch-9",
		OldStateRoot:         oldStateRootC,
		NewStateRoot:         newStateRootD,
		Trades:               []*types.Trade{agreementTrade()},
		TradeBatchCommitment: []byte("TRD-ATOM-USDT-batch-9"),
	}
	batchCommitments := types.BatchCommitments{
		DepositsRoot:        emptyPublicInputRootSentinel,
		WithdrawalsRoot:     emptyPublicInputRootSentinel,
		NullifiersRoot:      emptyPublicInputRootSentinel,
		WithdrawOutputsRoot: emptyPublicInputRootSentinel,
		TradesRoot:          tradesRoot,
		OrdersRoot:          ordersRoot,
	}
	proofBundle := proofBundleJSON(t, []string{
		settlementUpdate.OldStateRoot,
		settlementUpdate.NewStateRoot,
		batchCommitments.DepositsRoot,
		batchCommitments.WithdrawalsRoot,
		batchCommitments.NullifiersRoot,
		batchCommitments.WithdrawOutputsRoot,
		batchCommitments.TradesRoot,
		batchCommitments.OrdersRoot,
	})

	return settlementUpdate, batchCommitments, proofBundle
}

func validMixedMsgSubmitBatchProof(t *testing.T, f *fixture) (types.SettlementUpdate, types.BatchCommitments, []byte) {
	t.Helper()

	settlementUpdate, batchCommitments, _ := validMsgSubmitBatchProof(t, f)
	settlementUpdate.Trades = []*types.Trade{agreementTrade()}
	settlementUpdate.TradeBatchCommitment = []byte("TRD-MIXED-ATOM-USDT-batch-1")
	batchCommitments.TradesRoot = tradesRoot
	batchCommitments.OrdersRoot = ordersRoot
	proofBundle := proofBundleJSON(t, []string{
		settlementUpdate.OldStateRoot,
		settlementUpdate.NewStateRoot,
		batchCommitments.DepositsRoot,
		batchCommitments.WithdrawalsRoot,
		batchCommitments.NullifiersRoot,
		batchCommitments.WithdrawOutputsRoot,
		batchCommitments.TradesRoot,
		batchCommitments.OrdersRoot,
	})

	return settlementUpdate, batchCommitments, proofBundle
}

func validMsgSubmitBatchProof(t *testing.T, f *fixture) (types.SettlementUpdate, types.BatchCommitments, []byte) {
	t.Helper()

	require.NoError(t, f.keeper.SetStateRoot(f.ctx, oldStateRootA))
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
		OldStateRoot: oldStateRootA,
		NewStateRoot: newStateRootB,
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
		DepositsRoot:        depositsRoot,
		WithdrawalsRoot:     withdrawalsRoot,
		NullifiersRoot:      nullifiersRoot,
		WithdrawOutputsRoot: withdrawOutputsRoot,
	}
	proofBundle := proofBundleJSON(t, noTradePublicInputs(settlementUpdate, batchCommitments))

	return settlementUpdate, batchCommitments, proofBundle
}

func noTradePublicInputs(settlementUpdate types.SettlementUpdate, batchCommitments types.BatchCommitments) []string {
	return []string{
		settlementUpdate.OldStateRoot,
		settlementUpdate.NewStateRoot,
		batchCommitments.DepositsRoot,
		batchCommitments.WithdrawalsRoot,
		batchCommitments.NullifiersRoot,
		batchCommitments.WithdrawOutputsRoot,
		emptyPublicInputRootSentinel,
		emptyPublicInputRootSentinel,
	}
}

func logPublicInputs(t *testing.T, title string, publicInputs []string) {
	t.Helper()

	type publicInputLog struct {
		Index int    `json:"index"`
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	names := []string{
		"oldStateRoot",
		"newStateRoot",
		"depositsRoot",
		"withdrawalsRoot",
		"nullifiersRoot",
		"withdrawOutputsRoot",
		"tradesRoot",
		"ordersRoot",
	}
	rows := make([]publicInputLog, 0, len(publicInputs))
	for i, value := range publicInputs {
		name := "unknown"
		if i < len(names) {
			name = names[i]
		}
		rows = append(rows, publicInputLog{
			Index: i,
			Name:  name,
			Value: value,
		})
	}
	bz, err := json.MarshalIndent(rows, "", "  ")
	require.NoError(t, err)
	t.Logf("%s:\n%s", title, bz)
}

func agreementTrade() *types.Trade {
	return &types.Trade{
		TradeId:        "trd-1",
		Market:         "ATOM/USDT",
		MakerOrderId:   "ord-077",
		TakerOrderId:   "ord-101",
		OrderHash:      "0x" + strings.Repeat("a", 64),
		OrderNullifier: "0x" + strings.Repeat("b", 64),
		Owner:          "cosmos1alice",
		Denom:          "uatom",
		Side:           "buy",
		Amount:         "3",
		Price:          "10.20",
		BaseQty:        "3",
		QuoteQty:       "30.60",
		MakerFee:       "0.03",
		TakerFee:       "0.06",
	}
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
