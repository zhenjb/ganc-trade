package types_test

import (
	"strings"
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	"ob/x/zkdex/types"
)

func TestSettlementUpdateRoundTripWithTrade(t *testing.T) {
	update := types.SettlementUpdate{
		BatchId:              "batch-9",
		OldStateRoot:         "0xrootC",
		NewStateRoot:         "0xrootD",
		Trades:               []*types.Trade{agreementTrade()},
		TradeBatchCommitment: []byte("TRD-ATOM-USDT-batch-9"),
	}

	bz, err := proto.Marshal(&update)
	require.NoError(t, err)

	var decoded types.SettlementUpdate
	require.NoError(t, proto.Unmarshal(bz, &decoded))
	require.Equal(t, update.BatchId, decoded.BatchId)
	require.Empty(t, decoded.Deposits)
	require.Empty(t, decoded.Withdrawals)
	require.Len(t, decoded.Trades, 1)
	require.Equal(t, agreementTrade(), decoded.Trades[0])
	require.Equal(t, update.TradeBatchCommitment, decoded.TradeBatchCommitment)
	require.NoError(t, decoded.Trades[0].ValidateBasic())
}

func TestSettlementUpdateRoundTripWithoutTrade(t *testing.T) {
	update := types.SettlementUpdate{
		BatchId:      "batch-legacy",
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
	}

	bz, err := proto.Marshal(&update)
	require.NoError(t, err)

	var decoded types.SettlementUpdate
	require.NoError(t, proto.Unmarshal(bz, &decoded))
	require.Equal(t, update.BatchId, decoded.BatchId)
	require.Len(t, decoded.Deposits, 1)
	require.Empty(t, decoded.Trades)
	require.Empty(t, decoded.TradeBatchCommitment)
}

func TestTradeValidateBasicRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(trade *types.Trade)
		errText string
	}{
		{
			name: "bad hash",
			mutate: func(trade *types.Trade) {
				trade.OrderHash = "0xnot-hex"
			},
			errText: "orderHash must be 32 bytes",
		},
		{
			name: "bad denom",
			mutate: func(trade *types.Trade) {
				trade.Denom = "bad denom"
			},
			errText: "invalid denom",
		},
		{
			name: "bad side",
			mutate: func(trade *types.Trade) {
				trade.Side = "hold"
			},
			errText: "side must be buy or sell",
		},
		{
			name: "bad amount",
			mutate: func(trade *types.Trade) {
				trade.Amount = "five"
			},
			errText: "amount must be a decimal string",
		},
		{
			name: "negative fee",
			mutate: func(trade *types.Trade) {
				trade.MakerFee = "-0.01"
			},
			errText: "makerFee cannot be negative",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			trade := agreementTrade()
			tc.mutate(trade)

			err := trade.ValidateBasic()
			require.Error(t, err)
			require.ErrorContains(t, err, tc.errText)
		})
	}
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
