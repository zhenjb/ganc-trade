package keeper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"ob/x/zkdex/keeper"
	zkdex "ob/x/zkdex/module"
	"ob/x/zkdex/types"
)

type stubBankKeeper struct{}

func (stubBankKeeper) SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	if addr.Equals(authtypes.NewModuleAddress(types.ModuleName)) {
		return sdk.NewCoins(sdk.NewInt64Coin("stake", 1000))
	}
	return sdk.NewCoins()
}

func (stubBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	return nil
}

func (stubBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func TestModuleAccountQuery(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig(zkdex.AppModule{})
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

	authority := authtypes.NewModuleAddress(types.GovModuleName)
	k := keeper.NewKeeper(storeService, encCfg.Codec, addressCodec, authority, stubBankKeeper{}, nil)

	qs := keeper.NewQueryServerImpl(k)

	addrResp, err := qs.ModuleAccountAddress(ctx, &types.QueryModuleAccountAddressRequest{})
	require.NoError(t, err)
	require.Equal(t, k.GetModuleAccountAddressString(), addrResp.Address)

	balanceResp, err := qs.ModuleAccountBalance(ctx, &types.QueryModuleAccountBalanceRequest{})
	require.NoError(t, err)
	require.Equal(t, "1000stake", balanceResp.Balance)
}
