package keeper

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"ob/x/zkdex/types"
)

type agreementProofBundle struct {
	Proof        string   `json:"proof"`
	PublicInputs []string `json:"publicInputs"`
}

type msgSubmitBatchProofVerifierInput struct {
	SettlementUpdate agreementSettlementUpdate `json:"settlementUpdate"`
	BatchCommitments agreementBatchCommitments `json:"batchCommitments"`
	PublicInputs     []string                  `json:"publicInputs"`
}

type agreementSettlementUpdate struct {
	BatchID              string                          `json:"batchId"`
	OldStateRoot         string                          `json:"oldStateRoot"`
	NewStateRoot         string                          `json:"newStateRoot"`
	Deposits             []agreementSettlementDeposit    `json:"deposits"`
	Withdrawals          []agreementSettlementWithdrawal `json:"withdrawals"`
	Trades               []agreementTrade                `json:"trades"`
	TradeBatchCommitment string                          `json:"tradeBatchCommitment"`
}

type agreementSettlementDeposit struct {
	DepositID string `json:"depositId"`
	Owner     string `json:"owner"`
	Denom     string `json:"denom"`
	Amount    string `json:"amount"`
}

type agreementSettlementWithdrawal struct {
	WithdrawID      string `json:"withdrawId"`
	Owner           string `json:"owner"`
	Denom           string `json:"denom"`
	Amount          string `json:"amount"`
	Destination     string `json:"destination"`
	DestinationHash string `json:"destinationHash"`
	Nullifier       string `json:"nullifier"`
}

type agreementTrade struct {
	TradeID        string `json:"tradeId"`
	Market         string `json:"market"`
	MakerOrderID   string `json:"makerOrderId"`
	TakerOrderID   string `json:"takerOrderId"`
	OrderHash      string `json:"orderHash"`
	OrderNullifier string `json:"orderNullifier"`
	Owner          string `json:"owner"`
	Denom          string `json:"denom"`
	Side           string `json:"side"`
	Amount         string `json:"amount"`
	Price          string `json:"price"`
	BaseQty        string `json:"baseQty"`
	QuoteQty       string `json:"quoteQty"`
	MakerFee       string `json:"makerFee"`
	TakerFee       string `json:"takerFee"`
}

type agreementBatchCommitments struct {
	DepositsRoot        string `json:"depositsRoot"`
	WithdrawalsRoot     string `json:"withdrawalsRoot"`
	NullifiersRoot      string `json:"nullifiersRoot"`
	WithdrawOutputsRoot string `json:"withdrawOutputsRoot"`
	TradesRoot          string `json:"tradesRoot"`
	OrdersRoot          string `json:"ordersRoot"`
}

const emptyPublicInputRootSentinel = "0x0000000000000000000000000000000000000000000000000000000000000000"

// verify proof
func (k msgServer) SubmitBatchProof(ctx context.Context, req *types.MsgSubmitBatchProof) (*types.MsgSubmitBatchProofResponse, error) {
	if req == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "request cannot be nil")
	}
	if _, err := k.addressCodec.StringToBytes(req.Creator); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "invalid creator address")
	}
	if len(req.ProofBundle) == 0 {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "proofBundle cannot be empty")
	}

	settlementUpdate := req.GetSettlementUpdate()
	batchCommitments := req.GetBatchCommitments()
	publicInputs, err := k.validateSettlementUpdate(ctx, settlementUpdate, batchCommitments)
	if err != nil {
		return nil, err
	}
	if err := validateProofBundlePublicInputs(req.ProofBundle, publicInputs); err != nil {
		return nil, err
	}

	verifierInput, err := buildMsgSubmitBatchProofVerifierInput(settlementUpdate, batchCommitments, publicInputs)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to encode MsgSubmitBatchProof verifier input")
	}
	if !k.Keeper.VerifyProof(verifierInput, req.ProofBundle) {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "proof verification failed")
	}
	if err := k.applySettlementUpdate(ctx, settlementUpdate, publicInputs); err != nil {
		return nil, err
	}

	return &types.MsgSubmitBatchProofResponse{
		Accepted:     true,
		PublicInputs: publicInputs,
	}, nil
}

// Check old root and create public input
func (k msgServer) validateSettlementUpdate(ctx context.Context, settlementUpdate types.SettlementUpdate, batchCommitments types.BatchCommitments) ([]string, error) {
	if settlementUpdate.BatchId == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "batchId cannot be empty")
	}
	exists, err := k.Keeper.HasBatchRecord(ctx, settlementUpdate.BatchId)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "failed to read batch record %s", settlementUpdate.BatchId)
	}
	if exists {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "batchId %s already exists", settlementUpdate.BatchId)
	}
	if settlementUpdate.OldStateRoot == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "oldStateRoot cannot be empty")
	}
	if settlementUpdate.NewStateRoot == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "newStateRoot cannot be empty")
	}

	currentRoot, err := k.Keeper.GetStateRoot(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to read current state root")
	}
	if settlementUpdate.OldStateRoot != currentRoot {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "oldStateRoot mismatch: got %s, current %s", settlementUpdate.OldStateRoot, currentRoot)
	}

	if len(settlementUpdate.Deposits) == 0 && len(settlementUpdate.Withdrawals) == 0 && len(settlementUpdate.Trades) == 0 {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "settlement update must include at least one deposit, withdrawal, or trade")
	}
	if err := k.validateSettlementDeposits(ctx, settlementUpdate.Deposits); err != nil {
		return nil, err
	}
	if err := k.validateSettlementWithdrawals(ctx, settlementUpdate.Withdrawals); err != nil {
		return nil, err
	}
	if err := k.validateSettlementTrades(ctx, settlementUpdate.Trades, settlementUpdate.TradeBatchCommitment); err != nil {
		return nil, err
	}

	publicInputs, err := derivePublicInputs(settlementUpdate, batchCommitments)
	if err != nil {
		return nil, err
	}

	return publicInputs, nil
}

func derivePublicInputs(settlementUpdate types.SettlementUpdate, batchCommitments types.BatchCommitments) ([]string, error) {
	tradesRoot := batchCommitments.TradesRoot
	ordersRoot := batchCommitments.OrdersRoot
	if len(settlementUpdate.Trades) == 0 {
		if tradesRoot != "" && tradesRoot != emptyPublicInputRootSentinel {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "tradesRoot must be empty-root sentinel when no trades are present")
		}
		if ordersRoot != "" && ordersRoot != emptyPublicInputRootSentinel {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "ordersRoot must be empty-root sentinel when no trades are present")
		}
		tradesRoot = emptyPublicInputRootSentinel
		ordersRoot = emptyPublicInputRootSentinel
	}

	publicInputs := []string{
		settlementUpdate.OldStateRoot,
		settlementUpdate.NewStateRoot,
		batchCommitments.DepositsRoot,
		batchCommitments.WithdrawalsRoot,
		batchCommitments.NullifiersRoot,
		batchCommitments.WithdrawOutputsRoot,
		tradesRoot,
		ordersRoot,
	}
	for i, input := range publicInputs {
		if err := validatePublicInputField(i, input); err != nil {
			return nil, err
		}
	}

	return publicInputs, nil
}

func validatePublicInputField(index int, input string) error {
	if input == "" {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "public input %d cannot be empty", index)
	}
	raw := strings.TrimPrefix(input, "0x")
	if len(raw) != 64 {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "public input %d must be 32-byte hex", index)
	}
	if _, err := hex.DecodeString(raw); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "public input %d must be hex encoded", index)
	}
	return nil
}

func (k msgServer) validateSettlementTrades(ctx context.Context, trades []*types.Trade, tradeBatchCommitment []byte) error {
	if len(trades) == 0 {
		return nil
	}
	if len(tradeBatchCommitment) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "tradeBatchCommitment cannot be empty when trades are present")
	}
	seenTrades := make(map[string]struct{}, len(trades))
	seenNullifiers := make(map[string]struct{}, len(trades))
	for _, trade := range trades {
		if err := trade.ValidateBasic(); err != nil {
			return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
		}
		normalizedNullifier, err := types.NormalizeOrderNullifier(trade.OrderNullifier)
		if err != nil {
			return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
		}
		if _, ok := seenTrades[trade.TradeId]; ok {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "duplicate tradeId %s", trade.TradeId)
		}
		if _, ok := seenNullifiers[normalizedNullifier]; ok {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "duplicate orderNullifier %s", trade.OrderNullifier)
		}
		used, err := k.Keeper.IsOrderNullifierUsed(ctx, trade.OrderNullifier)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to read orderNullifier %s", trade.OrderNullifier)
		}
		if used {
			return errorsmod.Wrapf(types.ErrOrderNullifierReused, "orderNullifier %s already used", trade.OrderNullifier)
		}
		seenTrades[trade.TradeId] = struct{}{}
		seenNullifiers[normalizedNullifier] = struct{}{}
	}
	return nil
}

// check unprocessed deposit
func (k msgServer) validateSettlementDeposits(ctx context.Context, deposits []*types.SettlementDeposit) error {
	seen := make(map[string]struct{}, len(deposits))
	for _, deposit := range deposits {
		if deposit == nil {
			return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "deposit cannot be nil")
		}
		if deposit.DepositId == "" {
			return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "depositId cannot be empty")
		}
		if _, ok := seen[deposit.DepositId]; ok {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "duplicate depositId %s", deposit.DepositId)
		}
		seen[deposit.DepositId] = struct{}{}

		record, err := k.Keeper.GetDepositRecord(ctx, deposit.DepositId)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "deposit %s not found", deposit.DepositId)
		}

		processed, err := k.Keeper.IsDepositProcessed(ctx, deposit.DepositId)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to read processed status for deposit %s", deposit.DepositId)
		}
		if processed || record.Processed {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "deposit %s already processed", deposit.DepositId)
		}
		if record.Owner != deposit.Owner || record.Denom != deposit.Denom || record.Amount != deposit.Amount {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "deposit %s does not match on-chain record", deposit.DepositId)
		}
	}
	return nil
}

// check unused nullifier
func (k msgServer) validateSettlementWithdrawals(ctx context.Context, withdrawals []*types.SettlementWithdrawal) error {
	seenWithdrawals := make(map[string]struct{}, len(withdrawals))
	seenNullifiers := make(map[string]struct{}, len(withdrawals))
	for _, withdrawal := range withdrawals {
		if withdrawal == nil {
			return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "withdrawal cannot be nil")
		}
		if withdrawal.WithdrawId == "" {
			return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "withdrawId cannot be empty")
		}
		if withdrawal.Owner == "" || withdrawal.Denom == "" || withdrawal.Amount == "" || withdrawal.Destination == "" {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "withdrawal %s has empty required fields", withdrawal.WithdrawId)
		}
		if withdrawal.DestinationHash == "" {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "withdrawal %s destinationHash cannot be empty", withdrawal.WithdrawId)
		}
		if withdrawal.Nullifier == "" {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "withdrawal %s nullifier cannot be empty", withdrawal.WithdrawId)
		}
		if _, ok := seenWithdrawals[withdrawal.WithdrawId]; ok {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "duplicate withdrawId %s", withdrawal.WithdrawId)
		}
		if _, ok := seenNullifiers[withdrawal.Nullifier]; ok {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "duplicate nullifier %s", withdrawal.Nullifier)
		}
		seenWithdrawals[withdrawal.WithdrawId] = struct{}{}
		seenNullifiers[withdrawal.Nullifier] = struct{}{}

		used, err := k.Keeper.IsNullifierUsed(ctx, withdrawal.Nullifier)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to read nullifier %s", withdrawal.Nullifier)
		}
		if used {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "nullifier %s already used", withdrawal.Nullifier)
		}
		exists, err := k.Keeper.HasWithdrawRecord(ctx, withdrawal.WithdrawId)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to read withdraw record %s", withdrawal.WithdrawId)
		}
		if exists {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "withdrawId %s already exists", withdrawal.WithdrawId)
		}
	}
	return nil
}

func (k msgServer) applySettlementUpdate(ctx context.Context, settlementUpdate types.SettlementUpdate, publicInputs []string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	depositIds := make([]string, 0, len(settlementUpdate.Deposits))
	withdrawIds := make([]string, 0, len(settlementUpdate.Withdrawals))
	tradeIds := make([]string, 0, len(settlementUpdate.Trades))

	if err := k.Keeper.SetStateRoot(ctx, settlementUpdate.NewStateRoot); err != nil {
		return errorsmod.Wrap(err, "failed to set new state root")
	}
	for _, deposit := range settlementUpdate.Deposits {
		if err := k.Keeper.SetDepositProcessed(ctx, deposit.DepositId); err != nil {
			return errorsmod.Wrapf(err, "failed to mark deposit %s processed", deposit.DepositId)
		}
		depositIds = append(depositIds, deposit.DepositId)
	}
	for _, withdrawal := range settlementUpdate.Withdrawals {
		if err := k.Keeper.SetNullifierUsed(ctx, withdrawal.Nullifier); err != nil {
			return errorsmod.Wrapf(err, "failed to mark nullifier %s used", withdrawal.Nullifier)
		}
		record := types.WithdrawRecord{
			WithdrawId:  withdrawal.WithdrawId,
			Owner:       withdrawal.Owner,
			Denom:       withdrawal.Denom,
			Amount:      withdrawal.Amount,
			Destination: withdrawal.Destination,
			Nullifier:   withdrawal.Nullifier,
			Claimed:     false,
		}
		if err := k.Keeper.SetWithdrawRecord(ctx, withdrawal.WithdrawId, record); err != nil {
			return errorsmod.Wrapf(err, "failed to create withdraw record %s", withdrawal.WithdrawId)
		}
		withdrawIds = append(withdrawIds, withdrawal.WithdrawId)
	}
	for _, trade := range settlementUpdate.Trades {
		if err := k.Keeper.SetOrderNullifierUsed(ctx, trade.OrderNullifier); err != nil {
			return errorsmod.Wrapf(err, "failed to mark orderNullifier %s used", trade.OrderNullifier)
		}
		tradeIds = append(tradeIds, trade.TradeId)
	}

	batchRecord := types.BatchRecord{
		BatchId:              settlementUpdate.BatchId,
		OldStateRoot:         settlementUpdate.OldStateRoot,
		NewStateRoot:         settlementUpdate.NewStateRoot,
		DepositIds:           depositIds,
		WithdrawIds:          withdrawIds,
		CreatedHeight:        sdkCtx.BlockHeight(),
		TradeIds:             tradeIds,
		DepositsRoot:         publicInputs[2],
		WithdrawalsRoot:      publicInputs[3],
		NullifiersRoot:       publicInputs[4],
		WithdrawOutputsRoot:  publicInputs[5],
		TradesRoot:           publicInputs[6],
		OrdersRoot:           publicInputs[7],
		TradeBatchCommitment: bytesToHexString(settlementUpdate.TradeBatchCommitment),
		TradeCount:           uint64(len(settlementUpdate.Trades)),
	}
	if err := k.Keeper.SetBatchRecord(ctx, settlementUpdate.BatchId, batchRecord); err != nil {
		return errorsmod.Wrapf(err, "failed to store batch record %s", settlementUpdate.BatchId)
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zkdex_batch_settlement_applied",
		sdk.NewAttribute("batch_id", settlementUpdate.BatchId),
		sdk.NewAttribute("old_state_root", settlementUpdate.OldStateRoot),
		sdk.NewAttribute("new_state_root", settlementUpdate.NewStateRoot),
	))
	if len(settlementUpdate.Trades) > 0 {
		tradeCount := len(settlementUpdate.Trades)
		fillCount := countSettlementFills(settlementUpdate.Trades)
		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			"TradeSettled",
			sdk.NewAttribute("batchId", settlementUpdate.BatchId),
			sdk.NewAttribute("batch_id", settlementUpdate.BatchId),
			sdk.NewAttribute("tradesRoot", publicInputs[6]),
			sdk.NewAttribute("trades_root", publicInputs[6]),
			sdk.NewAttribute("newStateRoot", settlementUpdate.NewStateRoot),
			sdk.NewAttribute("new_state_root", settlementUpdate.NewStateRoot),
			sdk.NewAttribute("fillCount", strconv.Itoa(fillCount)),
			sdk.NewAttribute("fill_count", strconv.Itoa(fillCount)),
			sdk.NewAttribute("trade_count", strconv.Itoa(tradeCount)),
			sdk.NewAttribute("trade_record_count", strconv.Itoa(tradeCount)),
		))
	}
	return nil
}

func countSettlementFills(trades []*types.Trade) int {
	fillIDs := make(map[string]struct{}, len(trades))
	for _, trade := range trades {
		if trade == nil {
			continue
		}
		fillIDs[settlementFillID(trade.TradeId)] = struct{}{}
	}
	return len(fillIDs)
}

func settlementFillID(tradeID string) string {
	switch {
	case strings.HasSuffix(tradeID, "-buy"):
		return strings.TrimSuffix(tradeID, "-buy")
	case strings.HasSuffix(tradeID, "-sell"):
		return strings.TrimSuffix(tradeID, "-sell")
	default:
		return tradeID
	}
}

// check public input mà proof dùng = public input của chain
func validateProofBundlePublicInputs(proofBundle []byte, publicInputs []string) error {
	var bundle agreementProofBundle
	if err := json.Unmarshal(proofBundle, &bundle); err != nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "proofBundle must be JSON")
	}
	if bundle.Proof == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "proofBundle.proof cannot be empty")
	}
	if !reflect.DeepEqual(bundle.PublicInputs, publicInputs) {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "proofBundle.publicInputs do not match derived publicInputs")
	}
	return nil
}

func buildMsgSubmitBatchProofVerifierInput(settlementUpdate types.SettlementUpdate, batchCommitments types.BatchCommitments, publicInputs []string) ([]byte, error) {
	payload := msgSubmitBatchProofVerifierInput{
		SettlementUpdate: agreementSettlementUpdate{
			BatchID:              settlementUpdate.BatchId,
			OldStateRoot:         settlementUpdate.OldStateRoot,
			NewStateRoot:         settlementUpdate.NewStateRoot,
			Deposits:             make([]agreementSettlementDeposit, 0, len(settlementUpdate.Deposits)),
			Withdrawals:          make([]agreementSettlementWithdrawal, 0, len(settlementUpdate.Withdrawals)),
			Trades:               make([]agreementTrade, 0, len(settlementUpdate.Trades)),
			TradeBatchCommitment: bytesToHexString(settlementUpdate.TradeBatchCommitment),
		},
		BatchCommitments: agreementBatchCommitments{
			DepositsRoot:        batchCommitments.DepositsRoot,
			WithdrawalsRoot:     batchCommitments.WithdrawalsRoot,
			NullifiersRoot:      batchCommitments.NullifiersRoot,
			WithdrawOutputsRoot: batchCommitments.WithdrawOutputsRoot,
			TradesRoot:          publicInputs[6],
			OrdersRoot:          publicInputs[7],
		},
		PublicInputs: publicInputs,
	}

	for _, deposit := range settlementUpdate.Deposits {
		payload.SettlementUpdate.Deposits = append(payload.SettlementUpdate.Deposits, agreementSettlementDeposit{
			DepositID: deposit.DepositId,
			Owner:     deposit.Owner,
			Denom:     deposit.Denom,
			Amount:    deposit.Amount,
		})
	}
	for _, withdrawal := range settlementUpdate.Withdrawals {
		payload.SettlementUpdate.Withdrawals = append(payload.SettlementUpdate.Withdrawals, agreementSettlementWithdrawal{
			WithdrawID:      withdrawal.WithdrawId,
			Owner:           withdrawal.Owner,
			Denom:           withdrawal.Denom,
			Amount:          withdrawal.Amount,
			Destination:     withdrawal.Destination,
			DestinationHash: withdrawal.DestinationHash,
			Nullifier:       withdrawal.Nullifier,
		})
	}
	for _, trade := range settlementUpdate.Trades {
		payload.SettlementUpdate.Trades = append(payload.SettlementUpdate.Trades, agreementTrade{
			TradeID:        trade.TradeId,
			Market:         trade.Market,
			MakerOrderID:   trade.MakerOrderId,
			TakerOrderID:   trade.TakerOrderId,
			OrderHash:      trade.OrderHash,
			OrderNullifier: trade.OrderNullifier,
			Owner:          trade.Owner,
			Denom:          trade.Denom,
			Side:           trade.Side,
			Amount:         trade.Amount,
			Price:          trade.Price,
			BaseQty:        trade.BaseQty,
			QuoteQty:       trade.QuoteQty,
			MakerFee:       trade.MakerFee,
			TakerFee:       trade.TakerFee,
		})
	}

	return json.Marshal(payload)
}

func bytesToHexString(bz []byte) string {
	if len(bz) == 0 {
		return ""
	}
	return "0x" + hex.EncodeToString(bz)
}
