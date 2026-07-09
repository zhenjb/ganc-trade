package types

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// NormalizeOrderNullifier validates a hex order nullifier and returns the
// canonical store key form: lowercase without a 0x prefix.
func NormalizeOrderNullifier(nullifier string) (string, error) {
	normalized := strings.TrimSpace(nullifier)
	if normalized == "" {
		return "", fmt.Errorf("order nullifier cannot be empty")
	}
	if strings.HasPrefix(strings.ToLower(normalized), "0x") {
		normalized = normalized[2:]
	}
	if normalized == "" {
		return "", fmt.Errorf("order nullifier cannot be empty")
	}
	normalized = strings.ToLower(normalized)
	if _, err := hex.DecodeString(normalized); err != nil {
		return "", fmt.Errorf("order nullifier must be valid hex: %w", err)
	}
	return normalized, nil
}
