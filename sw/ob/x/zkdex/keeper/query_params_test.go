package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"ob/x/zkdex/keeper"
	"ob/x/zkdex/types"
)

func TestParamsQuery(t *testing.T) {
	f := initFixture(t)

	qs := keeper.NewQueryServerImpl(f.keeper)
	params := types.DefaultParams()
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	response, err := qs.Params(f.ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}

func TestCurrentStateRootQuery(t *testing.T) {
	f := initFixture(t)

	 qs := keeper.NewQueryServerImpl(f.keeper)
	expectedRoot := "0x1234567890abcdef"
	require.NoError(t, f.keeper.SetStateRoot(f.ctx, expectedRoot))

	response, err := qs.CurrentStateRoot(f.ctx, &types.QueryCurrentStateRootRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryCurrentStateRootResponse{StateRoot: expectedRoot}, response)
}