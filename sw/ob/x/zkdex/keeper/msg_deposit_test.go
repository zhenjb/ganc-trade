package keeper_test

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/store/types"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"ob/testutil/sample"
	"ob/x/zkdex/keeper"
	module "ob/x/zkdex/module"
	"ob/x/zkdex/types"
)

// --- MOCK BANK KEEPER ---
type MockBankKeeper struct {
	escrowedCoins map[string]sdk.Coins
}

func NewMockBankKeeper() *MockBankKeeper {
	return &MockBankKeeper{escrowedCoins: make(map[string]sdk.Coins)}
}

func (m *MockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	m.escrowedCoins[senderAddr.String()] = m.escrowedCoins[senderAddr.String()].Add(amt...)
	return nil
}

func (m *MockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
    // Basic mock logic: subtract coins from the tracked balance
    m.escrowedCoins[recipientAddr.String()] = m.escrowedCoins[recipientAddr.String()].Sub(amt...)
    return nil
}

func (m *MockBankKeeper) SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
    return m.escrowedCoins[addr.String()]
}

// --- TEST FIXTURE ---
type depositFixture struct {
	ctx          context.Context
	keeper       keeper.Keeper
	msgServer    types.MsgServer
	addressCodec address.Codec
	mockBank     *MockBankKeeper
}

func initDepositFixture(t *testing.T) *depositFixture {
	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(storeKey)
	
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx
	authority := authtypes.NewModuleAddress(types.GovModuleName)
	mockBank := NewMockBankKeeper()

	k := keeper.NewKeeper(storeService, encCfg.Codec, addressCodec, authority, mockBank, nil)
	k.Params.Set(ctx, types.DefaultParams())
	
	return &depositFixture{
		ctx:          ctx,
		keeper:       k,
		msgServer:    keeper.NewMsgServerImpl(k),
		addressCodec: addressCodec,
		mockBank:     mockBank,
	}
}

// --- INTEGRATED TEST CASE ---

func TestMsgDeposit_FullFlow(t *testing.T) {
	f := initDepositFixture(t)
	alice := sample.AccAddress()

	fmt.Println("\n🌟 STARTING INTEGRATED TEST: The ZKDEX Deposit Lifecycle")
	fmt.Printf("📍 Alice (User) Address: %s\n", alice)

	// --- STEP 1: FIRST DEPOSIT ---
	fmt.Println("\n--- STEP 1: First Deposit (100 uatom) ---")
	msg1 := &types.MsgDeposit{
		Creator: alice,
		Denom:   "uatom",
		Amount:  "100",
	}
	resp1, err := f.msgServer.Deposit(f.ctx, msg1)
	require.NoError(t, err)
	require.NotNil(t, resp1.DepositRecord)
	require.Equal(t, alice, resp1.DepositRecord.Owner)
	require.Equal(t, "uatom", resp1.DepositRecord.Denom)
	require.Equal(t, "100", resp1.DepositRecord.Amount)
	require.False(t, resp1.DepositRecord.Processed)

	id1 := resp1.DepositRecord.DepositId
	fmt.Printf("✅ Success! ID Generated: %s\n", id1)

	// ONCHAIN-05 creates the record, ONCHAIN-04 persists it.
	storedRecord1, err := f.keeper.GetDepositRecord(f.ctx, id1)
	require.NoError(t, err)
	require.Equal(t, *resp1.DepositRecord, storedRecord1)

	// This is the same query path backend/frontend use through gRPC/REST.
	qs := keeper.NewQueryServerImpl(f.keeper)
	queryResp1, err := qs.DepositRecord(f.ctx, &types.QueryDepositRecordRequest{DepositId: id1})
	require.NoError(t, err)
	require.Equal(t, resp1.DepositRecord, queryResp1.Record)

	// --- STEP 2: CHECK UNIQUE ID & MULTIPLE DEPOSITS ---
	fmt.Println("\n--- STEP 2: Second Deposit (Same block check) ---")
	msg2 := &types.MsgDeposit{
		Creator: alice,
		Denom:   "uatom",
		Amount:  "250",
	}
	resp2, err := f.msgServer.Deposit(f.ctx, msg2)
	require.NoError(t, err)
	id2 := resp2.DepositRecord.DepositId
	fmt.Printf("✅ Success! ID Generated: %s\n", id2)

	require.NotEqual(t, id1, id2, "❌ ERROR: Collision detected! IDs must be unique.")
	fmt.Println("🛡️  Anti-collision verified.")

	// --- STEP 3: VERIFY STATE (KVSTORE) ---
	fmt.Println("\n--- STEP 3: Verifying Blockchain State ---")
	record, err := f.keeper.GetDepositRecord(f.ctx, id2)
	require.NoError(t, err)
	require.Equal(t, "250", record.Amount)
	require.False(t, record.Processed)
	fmt.Printf("📦 Store Content for ID %s: Amount=%s, Processed=%v\n", id2, record.Amount, record.Processed)

	// --- STEP 4: VERIFY BANK (MONEY FLOW) ---
	fmt.Println("\n--- STEP 4: Verifying Money Flow (Escrow) ---")
	totalInModule := f.mockBank.escrowedCoins[alice]
	fmt.Printf("💰 Total funds locked in zkdex module for Alice: %s\n", totalInModule)
	require.Equal(t, "350uatom", totalInModule.String())

	// --- STEP 5: VERIFY EVENTS (SIGNAL FOR P4 BACKEND) ---
	fmt.Println("\n--- STEP 5: Verifying Event Emission (Signal for Backend) ---")
	sdkCtx := sdk.UnwrapSDKContext(f.ctx)
	events := sdkCtx.EventManager().Events()
	
	depositEventFound := false
	for _, ev := range events {
		if ev.Type == "ob.zkdex.v1.EventDeposit" {
			depositEventFound = true
			fmt.Printf("📡 Stored event: %#v\n", ev)
			for _, attr := range ev.Attributes {
				fmt.Printf("   🏷️  Stored attr: %s=%s\n", attr.Key, attr.Value)
			}
		}
	}
	require.True(t, depositEventFound, "❌ ERROR: No EventDeposit emitted!")

	fmt.Println("🏁 ALL SYSTEMS GO: Deposit flow is solid for P1, P4, and P5.")
}

// --- VALIDATION TEST CASE ---
func TestMsgDeposit_InvalidInputs(t *testing.T) {
	f := initDepositFixture(t)
	fmt.Println("🛠️  Running Edge Case Validations...")

	testCases := []struct {
		name string
		amt  string
		den  string
	}{
		{"Negative", "-1", "uatom"},
		{"Zero", "0", "uatom"},
		{"Missing Denom", "100", ""},
	}

	for _, tc := range testCases {
		msg := &types.MsgDeposit{Creator: sample.AccAddress(), Denom: tc.den, Amount: tc.amt}
		_, err := f.msgServer.Deposit(f.ctx, msg)
		require.Error(t, err)
		fmt.Printf("✅ Correct Error for [%s]: %v\n", tc.name, err)
	}
}
