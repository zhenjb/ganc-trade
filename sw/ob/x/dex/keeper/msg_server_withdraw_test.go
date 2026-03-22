package keeper_test

import (
	"context"
	"errors"
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"ob/x/dex/keeper"
	module "ob/x/dex/module"
	"ob/x/dex/types"
)

// mockBankKeeper is a controllable mock for types.BankKeeper.
type mockBankKeeper struct {
	// balances holds coins per module name (for SendCoinsFromModuleToAccount)
	balances map[string]sdk.Coins
	// sendErr, if set, is returned by SendCoinsFromModuleToAccount
	sendErr error
}

func newMockBankKeeper(moduleCoins sdk.Coins) *mockBankKeeper {
	return &mockBankKeeper{
		balances: map[string]sdk.Coins{
			types.ModuleName: moduleCoins,
		},
	}
}

func (m *mockBankKeeper) SpendableCoins(_ context.Context, _ sdk.AccAddress) sdk.Coins {
	return nil
}

func (m *mockBankKeeper) GetBalance(_ context.Context, _ sdk.AccAddress, _ string) sdk.Coin {
	return sdk.Coin{}
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, _ sdk.AccAddress, _ string, _ sdk.Coins) error {
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, moduleName string, addr sdk.AccAddress, amt sdk.Coins) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	bal := m.balances[moduleName]
	if !bal.IsAllGTE(amt) {
		return errors.New("insufficient funds in module")
	}
	m.balances[moduleName] = bal.Sub(amt...)
	return nil
}

// initFixtureWithBank creates a fixture wired with the given bank keeper mock.
func initFixtureWithBank(t *testing.T, bank types.BankKeeper) *fixture {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test_withdraw")).Ctx

	authority := authtypes.NewModuleAddress(types.GovModuleName)

	k := keeper.NewKeeper(
		storeService,
		encCfg.Codec,
		addressCodec,
		authority,
		nil,
		bank,
	)

	if err := k.Params.Set(ctx, types.DefaultParams()); err != nil {
		t.Fatalf("failed to set params: %v", err)
	}

	return &fixture{ctx: ctx, keeper: k, addressCodec: addressCodec}
}

func TestWithdraw(t *testing.T) {
	validAddr, _ := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()).
		BytesToString([]byte("signerAddr__________________"))

	moduleBalance := sdk.NewCoins(sdk.NewCoin("ATOM", math.NewInt(5000)))

	tests := []struct {
		desc    string
		msg     *types.MsgWithdraw
		bankErr error
		wantErr bool
		errMsg  string
	}{
		{
			desc: "success — full balance",
			msg:  &types.MsgWithdraw{Creator: validAddr, Amount: "1000", Denom: "ATOM"},
		},
		{
			desc: "success — partial amount",
			msg:  &types.MsgWithdraw{Creator: validAddr, Amount: "500", Denom: "ATOM"},
		},
		{
			desc:    "invalid creator address",
			msg:     &types.MsgWithdraw{Creator: "not-a-valid-address", Amount: "100", Denom: "ATOM"},
			wantErr: true,
			errMsg:  "invalid address",
		},
		{
			desc:    "invalid amount — zero",
			msg:     &types.MsgWithdraw{Creator: validAddr, Amount: "0", Denom: "ATOM"},
			wantErr: true,
			errMsg:  "invalid amount",
		},
		{
			desc:    "invalid amount — negative string",
			msg:     &types.MsgWithdraw{Creator: validAddr, Amount: "-50", Denom: "ATOM"},
			wantErr: true,
			errMsg:  "invalid amount",
		},
		{
			desc:    "invalid amount — non-numeric",
			msg:     &types.MsgWithdraw{Creator: validAddr, Amount: "abc", Denom: "ATOM"},
			wantErr: true,
			errMsg:  "invalid amount",
		},
		{
			desc:    "insufficient module funds",
			msg:     &types.MsgWithdraw{Creator: validAddr, Amount: "9999", Denom: "ATOM"},
			wantErr: true,
			errMsg:  "insufficient funds",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			bank := newMockBankKeeper(moduleBalance)
			if tc.bankErr != nil {
				bank.sendErr = tc.bankErr
			}
			f := initFixtureWithBank(t, bank)

			err := f.keeper.Withdraw(sdk.UnwrapSDKContext(f.ctx), tc.msg.Creator, tc.msg.Amount, tc.msg.Denom)

			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
