package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/store" // Tên mặc định là store

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"ob/x/zkdex/types"
)

type Keeper struct {
	storeService store.KVStoreService // Đã đổi corestore -> store
	cdc          codec.Codec
	addressCodec address.Codec
	authority    []byte

	Schema           collections.Schema
	Params           collections.Item[types.Params]
	StateRoot        collections.Item[string]
	DepositRecords   collections.Map[string, types.DepositRecord]
	WithdrawRecords  collections.Map[string, types.WithdrawRecord]
	NullifierUsed    collections.Map[string, bool]
	DepositProcessed collections.Map[string, bool]
	BatchRecords     collections.Map[string, types.BatchRecord]

	bankKeeper types.BankKeeper
	authKeeper types.AuthKeeper
	verifier   types.ProofVerifier
}

func NewKeeper(
	storeService store.KVStoreService, // Đã đổi corestore -> store
	cdc codec.Codec,
	addressCodec address.Codec,
	authority []byte,

	bankKeeper types.BankKeeper,
	authKeeper types.AuthKeeper,
	verifiers ...types.ProofVerifier,
) Keeper {
	if _, err := addressCodec.BytesToString(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address %s: %s", authority, err))
	}

	sb := collections.NewSchemaBuilder(storeService)

	verifier := types.ProofVerifier(types.RejectingProofVerifier{})
	if len(verifiers) > 0 && verifiers[0] != nil {
		verifier = verifiers[0]
	}

	k := Keeper{
		storeService: storeService,
		cdc:          cdc,
		addressCodec: addressCodec,
		authority:    authority,

		bankKeeper:       bankKeeper,
		authKeeper:       authKeeper,
		verifier:         verifier,
		Params:           collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		StateRoot:        collections.NewItem(sb, types.StateRootKey, "state_root", collections.StringValue),
		DepositRecords:   collections.NewMap(sb, types.DepositRecordKey, "deposit_records", collections.StringKey, codec.CollValue[types.DepositRecord](cdc)),
		WithdrawRecords:  collections.NewMap(sb, types.WithdrawRecordKey, "withdraw_records", collections.StringKey, codec.CollValue[types.WithdrawRecord](cdc)),
		NullifierUsed:    collections.NewMap(sb, types.NullifierUsedKey, "nullifier_used", collections.StringKey, collections.BoolValue),
		DepositProcessed: collections.NewMap(sb, types.DepositProcessedKey, "deposit_processed", collections.StringKey, collections.BoolValue),
		BatchRecords:     collections.NewMap(sb, types.BatchRecordKey, "batch_records", collections.StringKey, codec.CollValue[types.BatchRecord](cdc)),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

func (k Keeper) WithProofVerifier(verifier types.ProofVerifier) Keeper {
	if verifier == nil {
		verifier = types.RejectingProofVerifier{}
	}
	k.verifier = verifier
	return k
}

func (k Keeper) VerifyProof(update []byte, proofBundle []byte) bool {
	if k.verifier == nil {
		return false
	}
	return k.verifier.VerifyProof(update, proofBundle)
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() []byte {
	return k.authority
}

// GetModuleAccountAddress returns the zkdex module account address.
func (k Keeper) GetModuleAccountAddress() sdk.AccAddress {
	return authtypes.NewModuleAddress(types.ModuleName)
}

// GetModuleAccountAddressString returns the zkdex module account address as a string.
func (k Keeper) GetModuleAccountAddressString() string {
	addr, _ := k.addressCodec.BytesToString(k.GetModuleAccountAddress())
	return addr
}

// GetModuleAccountBalance returns the spendable balance of the zkdex module account.
func (k Keeper) GetModuleAccountBalance(ctx context.Context) sdk.Coins {
	if k.bankKeeper == nil {
		return sdk.NewCoins()
	}
	return k.bankKeeper.SpendableCoins(ctx, k.GetModuleAccountAddress())
}

// EscrowFunds sends coins from an account into the zkdex module account.
func (k Keeper) EscrowFunds(ctx context.Context, sender sdk.AccAddress, amt sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, amt)
}

// ReleaseFunds sends coins from the zkdex module account to an account.
func (k Keeper) ReleaseFunds(ctx context.Context, recipient sdk.AccAddress, amt sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipient, amt)
}

func (k Keeper) SetStateRoot(ctx context.Context, root string) error {
	return k.StateRoot.Set(ctx, root)
}

func (k Keeper) GetStateRoot(ctx context.Context) (string, error) {
	root, err := k.StateRoot.Get(ctx)
	if err != nil {
		// If not found, return default
		return "0xrootA", nil
	}
	return root, nil
}

// DepositRecord methods
func (k Keeper) SetDepositRecord(ctx context.Context, depositId string, record types.DepositRecord) error {
	return k.DepositRecords.Set(ctx, depositId, record)
}

func (k Keeper) GetDepositRecord(ctx context.Context, depositId string) (types.DepositRecord, error) {
	return k.DepositRecords.Get(ctx, depositId)
}

func (k Keeper) HasDepositRecord(ctx context.Context, depositId string) (bool, error) {
	return k.DepositRecords.Has(ctx, depositId)
}

// WithdrawRecord methods
func (k Keeper) SetWithdrawRecord(ctx context.Context, withdrawId string, record types.WithdrawRecord) error {
	return k.WithdrawRecords.Set(ctx, withdrawId, record)
}

func (k Keeper) GetWithdrawRecord(ctx context.Context, withdrawId string) (types.WithdrawRecord, error) {
	return k.WithdrawRecords.Get(ctx, withdrawId)
}

func (k Keeper) HasWithdrawRecord(ctx context.Context, withdrawId string) (bool, error) {
	return k.WithdrawRecords.Has(ctx, withdrawId)
}

// Nullifier methods
func (k Keeper) SetNullifierUsed(ctx context.Context, nullifier string) error {
	return k.NullifierUsed.Set(ctx, nullifier, true)
}

func (k Keeper) IsNullifierUsed(ctx context.Context, nullifier string) (bool, error) {
	used, err := k.NullifierUsed.Get(ctx, nullifier)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return used, nil
}

// DepositProcessed methods
func (k Keeper) SetDepositProcessed(ctx context.Context, depositId string) error {
	if err := k.DepositProcessed.Set(ctx, depositId, true); err != nil {
		return err
	}

	record, err := k.GetDepositRecord(ctx, depositId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil
		}
		return err
	}

	record.Processed = true
	return k.SetDepositRecord(ctx, depositId, record)
}

func (k Keeper) IsDepositProcessed(ctx context.Context, depositId string) (bool, error) {
	processed, err := k.DepositProcessed.Get(ctx, depositId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return processed, nil
}

// BatchRecord methods
func (k Keeper) SetBatchRecord(ctx context.Context, batchId string, record types.BatchRecord) error {
	return k.BatchRecords.Set(ctx, batchId, record)
}

func (k Keeper) GetBatchRecord(ctx context.Context, batchId string) (types.BatchRecord, error) {
	return k.BatchRecords.Get(ctx, batchId)
}

func (k Keeper) HasBatchRecord(ctx context.Context, batchId string) (bool, error) {
	return k.BatchRecords.Has(ctx, batchId)
}
