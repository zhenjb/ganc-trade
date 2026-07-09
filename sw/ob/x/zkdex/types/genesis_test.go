package types_test

import (
	"strings"
	"testing"

	"ob/x/zkdex/types"

	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc:     "valid genesis state",
			genState: &types.GenesisState{},
			valid:    true,
		},
		{
			desc: "valid order nullifier normalizes",
			genState: &types.GenesisState{
				OrderNullifierUsed: []string{"0x" + strings.Repeat("A", 64)},
			},
			valid: true,
		},
		{
			desc: "duplicate order nullifier",
			genState: &types.GenesisState{
				OrderNullifierUsed: []string{"0xAB", "ab"},
			},
			valid: false,
		},
		{
			desc: "invalid order nullifier hex",
			genState: &types.GenesisState{
				OrderNullifierUsed: []string{"0xnothex"},
			},
			valid: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestGenesisStateValidateRejectsDuplicateOrderNullifier(t *testing.T) {
	genState := &types.GenesisState{
		OrderNullifierUsed: []string{
			"0x" + strings.Repeat("A", 64),
			strings.Repeat("a", 64),
		},
	}

	err := genState.Validate()
	t.Logf("validate duplicate order nullifiers: input=%v err=%v", genState.OrderNullifierUsed, err)
	require.Error(t, err)
	require.ErrorContains(t, err, "duplicate order nullifier")
}
