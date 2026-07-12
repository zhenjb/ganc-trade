package types

import (
	"encoding/hex"
	"fmt"
	"strings"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const tradeHashHexBytes = 32

// ValidateBasic checks the agreement-level shape of a settlement trade.
func (t *Trade) ValidateBasic() error {
	if t == nil {
		return fmt.Errorf("trade cannot be nil")
	}
	if t.TradeId == "" {
		return fmt.Errorf("tradeId cannot be empty")
	}
	if t.Market == "" {
		return fmt.Errorf("trade %s market cannot be empty", t.TradeId)
	}
	if t.MakerOrderId == "" {
		return fmt.Errorf("trade %s makerOrderId cannot be empty", t.TradeId)
	}
	if t.TakerOrderId == "" {
		return fmt.Errorf("trade %s takerOrderId cannot be empty", t.TradeId)
	}
	if err := validateHexBytes("orderHash", t.OrderHash, tradeHashHexBytes); err != nil {
		return fmt.Errorf("trade %s: %w", t.TradeId, err)
	}
	if err := validateHexBytes("orderNullifier", t.OrderNullifier, tradeHashHexBytes); err != nil {
		return fmt.Errorf("trade %s: %w", t.TradeId, err)
	}
	if t.Owner == "" {
		return fmt.Errorf("trade %s owner cannot be empty", t.TradeId)
	}
	if err := sdk.ValidateDenom(t.Denom); err != nil {
		return fmt.Errorf("trade %s invalid denom: %w", t.TradeId, err)
	}
	if t.Side != "buy" && t.Side != "sell" {
		return fmt.Errorf("trade %s side must be buy or sell", t.TradeId)
	}
	if err := validatePositiveDecimal("amount", t.Amount); err != nil {
		return fmt.Errorf("trade %s: %w", t.TradeId, err)
	}
	if err := validatePositiveDecimal("price", t.Price); err != nil {
		return fmt.Errorf("trade %s: %w", t.TradeId, err)
	}
	if err := validatePositiveDecimal("baseQty", t.BaseQty); err != nil {
		return fmt.Errorf("trade %s: %w", t.TradeId, err)
	}
	if err := validatePositiveDecimal("quoteQty", t.QuoteQty); err != nil {
		return fmt.Errorf("trade %s: %w", t.TradeId, err)
	}
	if err := validateNonNegativeDecimal("makerFee", t.MakerFee); err != nil {
		return fmt.Errorf("trade %s: %w", t.TradeId, err)
	}
	if err := validateNonNegativeDecimal("takerFee", t.TakerFee); err != nil {
		return fmt.Errorf("trade %s: %w", t.TradeId, err)
	}
	return nil
}

func validateHexBytes(fieldName, value string, byteLen int) error {
	const hexPrefix = "0x"
	if !strings.HasPrefix(value, hexPrefix) {
		return fmt.Errorf("%s must be 0x-prefixed hex", fieldName)
	}
	encoded := strings.TrimPrefix(value, hexPrefix)
	if len(encoded) != byteLen*2 {
		return fmt.Errorf("%s must be %d bytes", fieldName, byteLen)
	}
	if _, err := hex.DecodeString(encoded); err != nil {
		return fmt.Errorf("%s must be valid hex: %w", fieldName, err)
	}
	return nil
}

func validatePositiveDecimal(fieldName, value string) error {
	dec, err := parseTradeDecimal(fieldName, value)
	if err != nil {
		return err
	}
	if dec.IsZero() || dec.IsNegative() {
		return fmt.Errorf("%s must be positive", fieldName)
	}
	return nil
}

func validateNonNegativeDecimal(fieldName, value string) error {
	dec, err := parseTradeDecimal(fieldName, value)
	if err != nil {
		return err
	}
	if dec.IsNegative() {
		return fmt.Errorf("%s cannot be negative", fieldName)
	}
	return nil
}

func parseTradeDecimal(fieldName, value string) (sdkmath.LegacyDec, error) {
	if value == "" {
		return sdkmath.LegacyDec{}, fmt.Errorf("%s cannot be empty", fieldName)
	}
	dec, err := sdkmath.LegacyNewDecFromStr(value)
	if err != nil {
		return sdkmath.LegacyDec{}, fmt.Errorf("%s must be a decimal string: %w", fieldName, err)
	}
	return dec, nil
}
