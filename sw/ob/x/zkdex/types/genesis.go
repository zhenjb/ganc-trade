package types

import "fmt"

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:           DefaultParams(),
		CurrentStateRoot: "0xrootA", // MOCK TEST GIÁ TRỊ ROOT BAN ĐẦU
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	seenOrderNullifiers := make(map[string]struct{}, len(gs.OrderNullifierUsed))
	for _, nullifier := range gs.OrderNullifierUsed {
		normalized, err := NormalizeOrderNullifier(nullifier)
		if err != nil {
			return err
		}
		if _, ok := seenOrderNullifiers[normalized]; ok {
			return fmt.Errorf("duplicate order nullifier %s", normalized)
		}
		seenOrderNullifiers[normalized] = struct{}{}
	}
	return nil
}
