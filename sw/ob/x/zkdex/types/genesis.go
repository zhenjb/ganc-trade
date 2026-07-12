package types

import "fmt"

const DefaultStateRoot = "0x0000000000000000000000000000000000000000000000000000000000000000"

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:           DefaultParams(),
		CurrentStateRoot: DefaultStateRoot,
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
