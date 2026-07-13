package keeper_test

import (
	"encoding/json"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
	nonCrossingTradesRoot        = "0x" + strings.Repeat("7", 64)
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
	require.Empty(t, batchRecord.TradeIds)
	require.Equal(t, depositsRoot, batchRecord.DepositsRoot)
	require.Equal(t, withdrawalsRoot, batchRecord.WithdrawalsRoot)
	require.Equal(t, nullifiersRoot, batchRecord.NullifiersRoot)
	require.Equal(t, withdrawOutputsRoot, batchRecord.WithdrawOutputsRoot)
	require.Equal(t, emptyPublicInputRootSentinel, batchRecord.TradesRoot)
	require.Equal(t, emptyPublicInputRootSentinel, batchRecord.OrdersRoot)
	require.Equal(t, uint64(0), batchRecord.TradeCount)
	logBatchRecord(t, "accepted deposit+withdraw no-trade batch record", batchRecord)
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
	require.Contains(t, string(gotVerifierUpdate), `"tradeBatchCommitment":"0x5452442d41544f4d2d555344432d62617463682d39"`)
	require.Contains(t, string(gotVerifierUpdate), `"tradesRoot":"`+tradesRoot+`"`)
	require.Contains(t, string(gotVerifierUpdate), `"ordersRoot":"`+ordersRoot+`"`)

	stateRoot, err := f.keeper.GetStateRoot(f.ctx)
	require.NoError(t, err)
	require.Equal(t, newStateRootD, stateRoot)

	orderNullifierUsed, err := f.keeper.IsOrderNullifierUsed(f.ctx, agreementTrade().OrderNullifier)
	require.NoError(t, err)
	require.True(t, orderNullifierUsed)
	logSubmitBatchProofState(t, "accepted trade-only batch state", f, settlementUpdate, resp.PublicInputs)

	batchRecord, err := f.keeper.GetBatchRecord(f.ctx, "batch-9")
	require.NoError(t, err)
	require.Empty(t, batchRecord.DepositIds)
	require.Empty(t, batchRecord.WithdrawIds)
	require.Equal(t, []string{"trd-1"}, batchRecord.TradeIds)
	require.Equal(t, emptyPublicInputRootSentinel, batchRecord.DepositsRoot)
	require.Equal(t, emptyPublicInputRootSentinel, batchRecord.WithdrawalsRoot)
	require.Equal(t, emptyPublicInputRootSentinel, batchRecord.NullifiersRoot)
	require.Equal(t, emptyPublicInputRootSentinel, batchRecord.WithdrawOutputsRoot)
	require.Equal(t, tradesRoot, batchRecord.TradesRoot)
	require.Equal(t, ordersRoot, batchRecord.OrdersRoot)
	require.Equal(t, "0x5452442d41544f4d2d555344432d62617463682d39", batchRecord.TradeBatchCommitment)
	require.Equal(t, uint64(1), batchRecord.TradeCount)
	logBatchRecord(t, "accepted trade-only batch record", batchRecord)
	requireTradeSettledEvent(t, f, settlementUpdate.BatchId, tradesRoot, newStateRootD, "1")
}

func TestMsgSubmitBatchProofONCHAINT05ApplyTradeUpdateDoD(t *testing.T) {
	f := initFixture(t)
	settlementUpdate, batchCommitments, proofBundle := validTradeOnlyMsgSubmitBatchProof(t, f)
	msgServer := keeper.NewMsgServerImpl(f.keeper.WithProofVerifier(types.StubProofVerifier{Accept: true}))

	beforeStateRoot, err := f.keeper.GetStateRoot(f.ctx)
	require.NoError(t, err)
	require.Equal(t, settlementUpdate.OldStateRoot, beforeStateRoot)
	t.Logf("ONCHAIN-T05 before apply: currentStateRoot=%s oldStateRoot=%s newStateRoot=%s", beforeStateRoot, settlementUpdate.OldStateRoot, settlementUpdate.NewStateRoot)
	logOrderNullifiersUsed(t, "ONCHAIN-T05 before apply orderNullifiers", f, settlementUpdate.Trades)

	resp, err := msgServer.SubmitBatchProof(f.ctx, &types.MsgSubmitBatchProof{
		Creator:          sample.AccAddress(),
		SettlementUpdate: settlementUpdate,
		BatchCommitments: batchCommitments,
		ProofBundle:      proofBundle,
	})
	require.NoError(t, err)
	require.True(t, resp.Accepted)

	// ONCHAIN-T05 DoD: sau apply, currentStateRoot = newStateRoot.
	currentStateRoot, err := f.keeper.GetStateRoot(f.ctx)
	require.NoError(t, err)
	require.Equal(t, settlementUpdate.NewStateRoot, currentStateRoot)
	t.Logf("ONCHAIN-T05 currentStateRoot applied: currentStateRoot=%s newStateRoot=%s", currentStateRoot, settlementUpdate.NewStateRoot)

	// ONCHAIN-T05 DoD: mọi orderNullifier trong batch trả về used=true.
	for _, trade := range settlementUpdate.Trades {
		used, err := f.keeper.IsOrderNullifierUsed(f.ctx, trade.OrderNullifier)
		require.NoError(t, err)
		require.True(t, used)
		t.Logf("ONCHAIN-T05 orderNullifier used: tradeId=%s orderNullifier=%s used=%v", trade.TradeId, trade.OrderNullifier, used)
	}
	logOrderNullifiersUsed(t, "ONCHAIN-T05 after apply orderNullifiers", f, settlementUpdate.Trades)

	// ONCHAIN-T05 DoD: event TradeSettled xuất hiện trong ctx.EventManager với đúng attribute.
	requireTradeSettledEvent(t, f, settlementUpdate.BatchId, tradesRoot, settlementUpdate.NewStateRoot, "1")

	batchRecord, err := f.keeper.GetBatchRecord(f.ctx, settlementUpdate.BatchId)
	require.NoError(t, err)
	require.Equal(t, []string{"trd-1"}, batchRecord.TradeIds)
	require.Equal(t, tradesRoot, batchRecord.TradesRoot)
	require.Equal(t, ordersRoot, batchRecord.OrdersRoot)
	require.Equal(t, uint64(len(settlementUpdate.Trades)), batchRecord.TradeCount)
	logSubmitBatchProofState(t, "ONCHAIN-T05 applied trade update state", f, settlementUpdate, resp.PublicInputs)
	logBatchRecord(t, "ONCHAIN-T05 applied trade update batch record", batchRecord)
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

	orderNullifierUsed, err := f.keeper.IsOrderNullifierUsed(f.ctx, agreementTrade().OrderNullifier)
	require.NoError(t, err)
	require.True(t, orderNullifierUsed)
	logSubmitBatchProofState(t, "accepted mixed batch state", f, settlementUpdate, resp.PublicInputs)

	batchRecord, err := f.keeper.GetBatchRecord(f.ctx, "batch-1")
	require.NoError(t, err)
	require.Equal(t, []string{"dep-1"}, batchRecord.DepositIds)
	require.Equal(t, []string{"wd-1"}, batchRecord.WithdrawIds)
	require.Equal(t, []string{"trd-1"}, batchRecord.TradeIds)
	require.Equal(t, depositsRoot, batchRecord.DepositsRoot)
	require.Equal(t, withdrawalsRoot, batchRecord.WithdrawalsRoot)
	require.Equal(t, nullifiersRoot, batchRecord.NullifiersRoot)
	require.Equal(t, withdrawOutputsRoot, batchRecord.WithdrawOutputsRoot)
	require.Equal(t, tradesRoot, batchRecord.TradesRoot)
	require.Equal(t, ordersRoot, batchRecord.OrdersRoot)
	require.Equal(t, "0x5452442d4d495845442d41544f4d2d555344432d62617463682d31", batchRecord.TradeBatchCommitment)
	require.Equal(t, uint64(1), batchRecord.TradeCount)
	logBatchRecord(t, "accepted mixed batch record", batchRecord)
	requireTradeSettledEvent(t, f, settlementUpdate.BatchId, tradesRoot, newStateRootB, "1")
}

func TestMsgSubmitBatchProofRejectsReusedOrderNullifierBeforeVerify(t *testing.T) {
	f := initFixture(t)
	settlementUpdate, batchCommitments, proofBundle := validTradeOnlyMsgSubmitBatchProof(t, f)
	orderNullifier := settlementUpdate.Trades[0].OrderNullifier
	// đánh dấu đã sử dụng
	require.NoError(t, f.keeper.SetOrderNullifierUsed(f.ctx, orderNullifier))

	verifierCalls := 0
	k := f.keeper.WithProofVerifier(types.ProofVerifierFunc(func(update []byte, proof []byte) bool {
		verifierCalls++
		return true
	}))
	msgServer := keeper.NewMsgServerImpl(k)

	_, submitErr := msgServer.SubmitBatchProof(f.ctx, &types.MsgSubmitBatchProof{
		Creator:          sample.AccAddress(),
		SettlementUpdate: settlementUpdate,
		BatchCommitments: batchCommitments,
		ProofBundle:      proofBundle,
	})
	require.Error(t, submitErr)
	require.ErrorIs(t, submitErr, types.ErrOrderNullifierReused)
	require.ErrorContains(t, submitErr, "orderNullifier")
	require.ErrorContains(t, submitErr, "already used")
	// Bằng 0: tức là hệ thống đã reject ngay từ vòng validate nullifier
	// chứ không chạy xuống bước verify proof bên dưới
	require.Zero(t, verifierCalls)

	stateRoot, err := f.keeper.GetStateRoot(f.ctx)
	require.NoError(t, err)
	// StateRoot của blockchain phải giữ nguyên
	require.Equal(t, oldStateRootC, stateRoot)

	batchExists, err := f.keeper.HasBatchRecord(f.ctx, settlementUpdate.BatchId)
	require.NoError(t, err)
	// trường hợp false: Batch ko đc lưu lại
	require.False(t, batchExists)
	logSubmitBatchProofState(t, "rejected reused order-nullifier state", f, settlementUpdate, nil)
	t.Logf("reused orderNullifier reject detail: orderNullifier=%s verifierCalls=%d err=%v", orderNullifier, verifierCalls, submitErr)
}

func TestMsgSubmitBatchProofONCHAINT07RejectsNonCrossingBadTradeRootWithoutMutatingState(t *testing.T) {
	f := initFixture(t)
	settlementUpdate, batchCommitments, _ := validTradeOnlyMsgSubmitBatchProof(t, f)
	batchCommitments.TradesRoot = nonCrossingTradesRoot
	proofBundle := proofBundleJSON(t, tradePublicInputs(settlementUpdate, batchCommitments))

	var verifierInput msgSubmitBatchProofVerifierInputLog
	verifierCalls := 0
	k := f.keeper.WithProofVerifier(types.ProofVerifierFunc(func(update []byte, proof []byte) bool {
		verifierCalls++
		require.NoError(t, json.Unmarshal(update, &verifierInput))
		t.Logf("ONCHAIN-T07 mock verifier input for non-crossing vector: tradesRoot=%s ordersRoot=%s tradeId=%s price=%s proofBundle=%s",
			verifierInput.BatchCommitments.TradesRoot,
			verifierInput.BatchCommitments.OrdersRoot,
			verifierInput.SettlementUpdate.Trades[0].TradeID,
			verifierInput.SettlementUpdate.Trades[0].Price,
			string(proof),
		)
		return verifierInput.BatchCommitments.TradesRoot != nonCrossingTradesRoot
	}))
	msgServer := keeper.NewMsgServerImpl(k)

	beforeRoot, err := f.keeper.GetStateRoot(f.ctx)
	require.NoError(t, err)
	t.Logf("ONCHAIN-T07 before non-crossing reject: currentStateRoot=%s batchId=%s badTradesRoot=%s ordersRoot=%s",
		beforeRoot, settlementUpdate.BatchId, batchCommitments.TradesRoot, batchCommitments.OrdersRoot)

	_, submitErr := msgServer.SubmitBatchProof(f.ctx, &types.MsgSubmitBatchProof{
		Creator:          sample.AccAddress(),
		SettlementUpdate: settlementUpdate,
		BatchCommitments: batchCommitments,
		ProofBundle:      proofBundle,
	})
	require.Error(t, submitErr)
	require.ErrorIs(t, submitErr, sdkerrors.ErrUnauthorized)
	require.ErrorContains(t, submitErr, "proof verification failed")
	require.Equal(t, 1, verifierCalls)

	requireTradeBatchRejectInvariant(t, "ONCHAIN-T07 non-crossing/bad-trade-root reject state", f, settlementUpdate)
	t.Logf("ONCHAIN-T07 non-crossing reject detail: verifierCalls=%d tradeId=%s price=%s badTradesRoot=%s err=%v",
		verifierCalls, settlementUpdate.Trades[0].TradeId, settlementUpdate.Trades[0].Price, batchCommitments.TradesRoot, submitErr)
}

func TestMsgSubmitBatchProofRejectsDuplicateOrderNullifierInBatch(t *testing.T) {
	f := initFixture(t)
	settlementUpdate, batchCommitments, proofBundle := validTradeOnlyMsgSubmitBatchProof(t, f)
	duplicateTrade := *settlementUpdate.Trades[0]
	duplicateTrade.TradeId = "trd-2"
	settlementUpdate.Trades = append(settlementUpdate.Trades, &duplicateTrade)

	msgServer := keeper.NewMsgServerImpl(f.keeper.WithProofVerifier(types.StubProofVerifier{Accept: true}))
	_, submitErr := msgServer.SubmitBatchProof(f.ctx, &types.MsgSubmitBatchProof{
		Creator:          sample.AccAddress(),
		SettlementUpdate: settlementUpdate,
		BatchCommitments: batchCommitments,
		ProofBundle:      proofBundle,
	})
	require.Error(t, submitErr)
	require.ErrorContains(t, submitErr, "duplicate orderNullifier")

	stateRoot, err := f.keeper.GetStateRoot(f.ctx)
	require.NoError(t, err)
	require.Equal(t, oldStateRootC, stateRoot)
	orderNullifierUsed, err := f.keeper.IsOrderNullifierUsed(f.ctx, duplicateTrade.OrderNullifier)
	require.NoError(t, err)
	require.False(t, orderNullifierUsed)
	logSubmitBatchProofState(t, "rejected duplicate order-nullifier state", f, settlementUpdate, nil)
	t.Logf("duplicate orderNullifier reject detail: tradeIds=%v orderNullifier=%s err=%v", []string{"trd-1", "trd-2"}, duplicateTrade.OrderNullifier, submitErr)
}

// khi proof bị sai
func TestMsgSubmitBatchProofRejectsBadTradeProofWithoutMutatingState(t *testing.T) {
	f := initFixture(t)
	settlementUpdate, batchCommitments, proofBundle := validTradeOnlyMsgSubmitBatchProof(t, f)
	msgServer := keeper.NewMsgServerImpl(f.keeper.WithProofVerifier(types.StubProofVerifier{Accept: false}))

	_, submitErr := msgServer.SubmitBatchProof(f.ctx, &types.MsgSubmitBatchProof{
		Creator:          sample.AccAddress(),
		SettlementUpdate: settlementUpdate,
		BatchCommitments: batchCommitments,
		ProofBundle:      proofBundle,
	})
	require.Error(t, submitErr)
	require.ErrorContains(t, submitErr, "proof verification failed")

	stateRoot, err := f.keeper.GetStateRoot(f.ctx)
	require.NoError(t, err)
	require.Equal(t, oldStateRootC, stateRoot)
	orderNullifierUsed, err := f.keeper.IsOrderNullifierUsed(f.ctx, settlementUpdate.Trades[0].OrderNullifier)
	require.NoError(t, err)
	require.False(t, orderNullifierUsed)
	batchExists, err := f.keeper.HasBatchRecord(f.ctx, settlementUpdate.BatchId)
	require.NoError(t, err)
	require.False(t, batchExists)
	logSubmitBatchProofState(t, "rejected bad trade proof state", f, settlementUpdate, nil)
	t.Logf("bad trade proof reject detail: proofBundle=%s err=%v", string(proofBundle), submitErr)
}

func TestMsgSubmitBatchProofONCHAINT07RejectsEmptyProofBundleWithoutMutatingState(t *testing.T) {
	f := initFixture(t)
	settlementUpdate, batchCommitments, _ := validTradeOnlyMsgSubmitBatchProof(t, f)
	msgServer := keeper.NewMsgServerImpl(f.keeper.WithProofVerifier(types.StubProofVerifier{Accept: true}))

	_, submitErr := msgServer.SubmitBatchProof(f.ctx, &types.MsgSubmitBatchProof{
		Creator:          sample.AccAddress(),
		SettlementUpdate: settlementUpdate,
		BatchCommitments: batchCommitments,
		ProofBundle:      nil,
	})
	require.Error(t, submitErr)
	require.ErrorIs(t, submitErr, sdkerrors.ErrInvalidRequest)
	require.ErrorContains(t, submitErr, "proofBundle cannot be empty")

	requireTradeBatchRejectInvariant(t, "ONCHAIN-T07 empty proof reject state", f, settlementUpdate)
	t.Logf("ONCHAIN-T07 empty proof reject detail: proofBundleLen=%d batchId=%s err=%v", 0, settlementUpdate.BatchId, submitErr)
}

func TestMsgSubmitBatchProofONCHAINT07RejectsMixedBatchWhenTradeProofFailsAllOrNothing(t *testing.T) {
	f := initFixture(t)
	settlementUpdate, batchCommitments, proofBundle := validMixedMsgSubmitBatchProof(t, f)
	msgServer := keeper.NewMsgServerImpl(f.keeper.WithProofVerifier(types.StubProofVerifier{Accept: false}))

	_, submitErr := msgServer.SubmitBatchProof(f.ctx, &types.MsgSubmitBatchProof{
		Creator:          sample.AccAddress(),
		SettlementUpdate: settlementUpdate,
		BatchCommitments: batchCommitments,
		ProofBundle:      proofBundle,
	})
	require.Error(t, submitErr)
	require.ErrorIs(t, submitErr, sdkerrors.ErrUnauthorized)
	require.ErrorContains(t, submitErr, "proof verification failed")

	requireMixedBatchRejectInvariant(t, "ONCHAIN-T07 mixed batch bad-proof all-or-nothing state", f, settlementUpdate)
	t.Logf("ONCHAIN-T07 mixed all-or-nothing reject detail: depositId=%s withdrawId=%s tradeId=%s proofBundle=%s err=%v",
		settlementUpdate.Deposits[0].DepositId,
		settlementUpdate.Withdrawals[0].WithdrawId,
		settlementUpdate.Trades[0].TradeId,
		string(proofBundle),
		submitErr)
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
		TradeBatchCommitment: []byte("TRD-ATOM-USDC-batch-9"),
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
	settlementUpdate.TradeBatchCommitment = []byte("TRD-MIXED-ATOM-USDC-batch-1")
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

func tradePublicInputs(settlementUpdate types.SettlementUpdate, batchCommitments types.BatchCommitments) []string {
	return []string{
		settlementUpdate.OldStateRoot,
		settlementUpdate.NewStateRoot,
		batchCommitments.DepositsRoot,
		batchCommitments.WithdrawalsRoot,
		batchCommitments.NullifiersRoot,
		batchCommitments.WithdrawOutputsRoot,
		batchCommitments.TradesRoot,
		batchCommitments.OrdersRoot,
	}
}

type msgSubmitBatchProofVerifierInputLog struct {
	SettlementUpdate struct {
		Trades []struct {
			TradeID string `json:"tradeId"`
			Price   string `json:"price"`
		} `json:"trades"`
	} `json:"settlementUpdate"`
	BatchCommitments struct {
		TradesRoot string `json:"tradesRoot"`
		OrdersRoot string `json:"ordersRoot"`
	} `json:"batchCommitments"`
	PublicInputs []string `json:"publicInputs"`
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

func logBatchRecord(t *testing.T, title string, batchRecord types.BatchRecord) {
	t.Helper()

	bz, err := json.MarshalIndent(batchRecord, "", "  ")
	require.NoError(t, err)
	t.Logf("%s:\n%s", title, bz)
}

func logOrderNullifiersUsed(t *testing.T, title string, f *fixture, trades []*types.Trade) {
	t.Helper()

	type orderNullifierLog struct {
		TradeID        string `json:"tradeId"`
		OrderNullifier string `json:"orderNullifier"`
		Used           bool   `json:"used"`
	}
	rows := make([]orderNullifierLog, 0, len(trades))
	for _, trade := range trades {
		used, err := f.keeper.IsOrderNullifierUsed(f.ctx, trade.OrderNullifier)
		require.NoError(t, err)
		rows = append(rows, orderNullifierLog{
			TradeID:        trade.TradeId,
			OrderNullifier: trade.OrderNullifier,
			Used:           used,
		})
	}
	bz, err := json.MarshalIndent(rows, "", "  ")
	require.NoError(t, err)
	t.Logf("%s:\n%s", title, bz)
}

func requireTradeSettledEvent(t *testing.T, f *fixture, batchID, expectedTradesRoot, expectedNewStateRoot, expectedFillCount string) {
	t.Helper()

	sdkCtx := sdk.UnwrapSDKContext(f.ctx)
	events := sdkCtx.EventManager().Events()
	for _, event := range events {
		if event.Type != "TradeSettled" {
			continue
		}
		attrs := make(map[string]string, len(event.Attributes))
		for _, attr := range event.Attributes {
			attrs[attr.Key] = attr.Value
		}
		bz, err := json.MarshalIndent(map[string]any{
			"type":       event.Type,
			"attributes": attrs,
		}, "", "  ")
		require.NoError(t, err)
		t.Logf("TradeSettled event observed:\n%s", bz)

		if attrs["batch_id"] != batchID {
			continue
		}
		require.Equal(t, batchID, attrs["batchId"])
		require.Equal(t, expectedTradesRoot, attrs["tradesRoot"])
		require.Equal(t, expectedTradesRoot, attrs["trades_root"])
		require.Equal(t, expectedNewStateRoot, attrs["newStateRoot"])
		require.Equal(t, expectedNewStateRoot, attrs["new_state_root"])
		require.Equal(t, expectedFillCount, attrs["fillCount"])
		require.Equal(t, expectedFillCount, attrs["fill_count"])
		require.Equal(t, expectedFillCount, attrs["trade_count"])
		return
	}
	require.Failf(t, "missing TradeSettled event", "batchID=%s tradesRoot=%s newStateRoot=%s fillCount=%s", batchID, expectedTradesRoot, expectedNewStateRoot, expectedFillCount)
}

func logSubmitBatchProofState(t *testing.T, title string, f *fixture, settlementUpdate types.SettlementUpdate, publicInputs []string) {
	t.Helper()

	type batchRecordLog struct {
		BatchID              string   `json:"batchId"`
		OldStateRoot         string   `json:"oldStateRoot"`
		NewStateRoot         string   `json:"newStateRoot"`
		DepositIds           []string `json:"depositIds"`
		WithdrawIds          []string `json:"withdrawIds"`
		TradeIds             []string `json:"tradeIds"`
		DepositsRoot         string   `json:"depositsRoot"`
		WithdrawalsRoot      string   `json:"withdrawalsRoot"`
		NullifiersRoot       string   `json:"nullifiersRoot"`
		WithdrawOutputsRoot  string   `json:"withdrawOutputsRoot"`
		TradesRoot           string   `json:"tradesRoot"`
		OrdersRoot           string   `json:"ordersRoot"`
		TradeBatchCommitment string   `json:"tradeBatchCommitment"`
		TradeCount           uint64   `json:"tradeCount"`
	}
	type orderNullifierLog struct {
		TradeID        string `json:"tradeId"`
		OrderNullifier string `json:"orderNullifier"`
		Used           bool   `json:"used"`
	}
	type submitStateLog struct {
		BatchID         string              `json:"batchId"`
		CurrentRoot     string              `json:"currentRoot"`
		ExpectedOld     string              `json:"expectedOld"`
		ExpectedNew     string              `json:"expectedNew"`
		BatchExists     bool                `json:"batchExists"`
		BatchRecord     *batchRecordLog     `json:"batchRecord,omitempty"`
		PublicInputs    []string            `json:"publicInputs,omitempty"`
		OrderNullifiers []orderNullifierLog `json:"orderNullifiers,omitempty"`
	}

	stateRoot, err := f.keeper.GetStateRoot(f.ctx)
	require.NoError(t, err)
	batchExists, err := f.keeper.HasBatchRecord(f.ctx, settlementUpdate.BatchId)
	require.NoError(t, err)
	var batchRecord *batchRecordLog
	if batchExists {
		record, err := f.keeper.GetBatchRecord(f.ctx, settlementUpdate.BatchId)
		require.NoError(t, err)
		batchRecord = &batchRecordLog{
			BatchID:              record.BatchId,
			OldStateRoot:         record.OldStateRoot,
			NewStateRoot:         record.NewStateRoot,
			DepositIds:           record.DepositIds,
			WithdrawIds:          record.WithdrawIds,
			TradeIds:             record.TradeIds,
			DepositsRoot:         record.DepositsRoot,
			WithdrawalsRoot:      record.WithdrawalsRoot,
			NullifiersRoot:       record.NullifiersRoot,
			WithdrawOutputsRoot:  record.WithdrawOutputsRoot,
			TradesRoot:           record.TradesRoot,
			OrdersRoot:           record.OrdersRoot,
			TradeBatchCommitment: record.TradeBatchCommitment,
			TradeCount:           record.TradeCount,
		}
	}

	orderNullifiers := make([]orderNullifierLog, 0, len(settlementUpdate.Trades))
	for _, trade := range settlementUpdate.Trades {
		used, err := f.keeper.IsOrderNullifierUsed(f.ctx, trade.OrderNullifier)
		require.NoError(t, err)
		orderNullifiers = append(orderNullifiers, orderNullifierLog{
			TradeID:        trade.TradeId,
			OrderNullifier: trade.OrderNullifier,
			Used:           used,
		})
	}

	bz, err := json.MarshalIndent(submitStateLog{
		BatchID:         settlementUpdate.BatchId,
		CurrentRoot:     stateRoot,
		ExpectedOld:     settlementUpdate.OldStateRoot,
		ExpectedNew:     settlementUpdate.NewStateRoot,
		BatchExists:     batchExists,
		BatchRecord:     batchRecord,
		PublicInputs:    publicInputs,
		OrderNullifiers: orderNullifiers,
	}, "", "  ")
	require.NoError(t, err)
	t.Logf("%s:\n%s", title, bz)
}

func requireTradeBatchRejectInvariant(t *testing.T, title string, f *fixture, settlementUpdate types.SettlementUpdate) {
	t.Helper()

	stateRoot, err := f.keeper.GetStateRoot(f.ctx)
	require.NoError(t, err)
	require.Equal(t, settlementUpdate.OldStateRoot, stateRoot)

	batchExists, err := f.keeper.HasBatchRecord(f.ctx, settlementUpdate.BatchId)
	require.NoError(t, err)
	require.False(t, batchExists)

	type orderNullifierLog struct {
		TradeID        string `json:"tradeId"`
		OrderNullifier string `json:"orderNullifier"`
		Used           bool   `json:"used"`
	}
	orderNullifiers := make([]orderNullifierLog, 0, len(settlementUpdate.Trades))
	for _, trade := range settlementUpdate.Trades {
		used, err := f.keeper.IsOrderNullifierUsed(f.ctx, trade.OrderNullifier)
		require.NoError(t, err)
		require.False(t, used)
		orderNullifiers = append(orderNullifiers, orderNullifierLog{
			TradeID:        trade.TradeId,
			OrderNullifier: trade.OrderNullifier,
			Used:           used,
		})
	}

	bz, err := json.MarshalIndent(map[string]any{
		"batchId":         settlementUpdate.BatchId,
		"currentRoot":     stateRoot,
		"expectedOld":     settlementUpdate.OldStateRoot,
		"expectedNew":     settlementUpdate.NewStateRoot,
		"batchExists":     batchExists,
		"orderNullifiers": orderNullifiers,
	}, "", "  ")
	require.NoError(t, err)
	t.Logf("%s:\n%s", title, bz)
}

func requireMixedBatchRejectInvariant(t *testing.T, title string, f *fixture, settlementUpdate types.SettlementUpdate) {
	t.Helper()

	requireTradeBatchRejectInvariant(t, title, f, settlementUpdate)

	type depositLog struct {
		DepositID string `json:"depositId"`
		Processed bool   `json:"processed"`
	}
	deposits := make([]depositLog, 0, len(settlementUpdate.Deposits))
	for _, deposit := range settlementUpdate.Deposits {
		processed, err := f.keeper.IsDepositProcessed(f.ctx, deposit.DepositId)
		require.NoError(t, err)
		require.False(t, processed)
		deposits = append(deposits, depositLog{
			DepositID: deposit.DepositId,
			Processed: processed,
		})
	}

	type withdrawalLog struct {
		WithdrawID    string `json:"withdrawId"`
		Nullifier     string `json:"nullifier"`
		NullifierUsed bool   `json:"nullifierUsed"`
		RecordExists  bool   `json:"recordExists"`
	}
	withdrawals := make([]withdrawalLog, 0, len(settlementUpdate.Withdrawals))
	for _, withdrawal := range settlementUpdate.Withdrawals {
		nullifierUsed, err := f.keeper.IsNullifierUsed(f.ctx, withdrawal.Nullifier)
		require.NoError(t, err)
		require.False(t, nullifierUsed)
		recordExists, err := f.keeper.HasWithdrawRecord(f.ctx, withdrawal.WithdrawId)
		require.NoError(t, err)
		require.False(t, recordExists)
		withdrawals = append(withdrawals, withdrawalLog{
			WithdrawID:    withdrawal.WithdrawId,
			Nullifier:     withdrawal.Nullifier,
			NullifierUsed: nullifierUsed,
			RecordExists:  recordExists,
		})
	}

	bz, err := json.MarshalIndent(map[string]any{
		"batchId":     settlementUpdate.BatchId,
		"deposits":    deposits,
		"withdrawals": withdrawals,
	}, "", "  ")
	require.NoError(t, err)
	t.Logf("%s core-operation invariant:\n%s", title, bz)
}

func agreementTrade() *types.Trade {
	return &types.Trade{
		TradeId:        "trd-1",
		Market:         "ATOM/USDC",
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
