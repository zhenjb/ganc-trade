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

	"ob/x/backend/keeper"
	module "ob/x/backend/module"
	"ob/x/backend/types"
)

// mockBankKeeper is a controllable mock for types.BankKeeper.
type mockBankKeeper struct {
	moduleCoins sdk.Coins
	sendErr     error
}

func newMockBankKeeper(moduleCoins sdk.Coins) *mockBankKeeper {
	return &mockBankKeeper{moduleCoins: moduleCoins}
}

func (m *mockBankKeeper) SpendableCoins(_ context.Context, _ sdk.AccAddress) sdk.Coins {
	return nil
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, _ sdk.AccAddress, _ string, _ sdk.Coins) error {
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, _ string, _ sdk.AccAddress, amt sdk.Coins) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	if !m.moduleCoins.IsAllGTE(amt) {
		return errors.New("insufficient funds in module")
	}
	m.moduleCoins = m.moduleCoins.Sub(amt...)
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
			msg:  &types.MsgWithdraw{Creator: validAddr, Amount: sdk.NewCoin("ATOM", math.NewInt(1000))},
		},
		{
			desc: "success — partial amount",
			msg:  &types.MsgWithdraw{Creator: validAddr, Amount: sdk.NewCoin("ATOM", math.NewInt(500))},
		},
		{
			desc:    "nil message",
			msg:     nil,
			wantErr: true,
			errMsg:  "invalid withdraw",
		},
		{
			desc:    "empty creator",
			msg:     &types.MsgWithdraw{Creator: "", Amount: sdk.NewCoin("ATOM", math.NewInt(100))},
			wantErr: true,
			errMsg:  "invalid withdraw",
		},
		{
			desc:    "invalid creator address",
			msg:     &types.MsgWithdraw{Creator: "not-a-valid-address", Amount: sdk.NewCoin("ATOM", math.NewInt(100))},
			wantErr: true,
			errMsg:  "invalid creator address",
		},
		{
			desc:    "zero amount",
			msg:     &types.MsgWithdraw{Creator: validAddr, Amount: sdk.NewCoin("ATOM", math.NewInt(0))},
			wantErr: true,
			errMsg:  "invalid withdraw",
		},
		{
			desc:    "bank returns insufficient funds",
			msg:     &types.MsgWithdraw{Creator: validAddr, Amount: sdk.NewCoin("ATOM", math.NewInt(9999))},
			wantErr: true,
			errMsg:  "insufficient funds",
		},
		{
			desc:    "bank returns arbitrary error",
			msg:     &types.MsgWithdraw{Creator: validAddr, Amount: sdk.NewCoin("ATOM", math.NewInt(100))},
			bankErr: errors.New("bank exploded"),
			wantErr: true,
			errMsg:  "bank exploded",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			bank := newMockBankKeeper(moduleBalance)
			bank.sendErr = tc.bankErr
			f := initFixtureWithBank(t, bank)

			msgServer := keeper.NewMsgServerImpl(f.keeper)
			_, err := msgServer.Withdraw(f.ctx, tc.msg)

			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
