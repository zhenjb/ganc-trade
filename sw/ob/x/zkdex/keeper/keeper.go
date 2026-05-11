package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"ob/x/zkdex/types"
)

type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec
	// Address capable of executing a MsgUpdateParams message.
	// Typically, this should be the x/gov module account.
	authority []byte

	Schema collections.Schema
	Params collections.Item[types.Params]

	bankKeeper types.BankKeeper
	authKeeper types.AuthKeeper
}

func NewKeeper(
	storeService corestore.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	authority []byte,

	bankKeeper types.BankKeeper,
	authKeeper types.AuthKeeper,
) Keeper {
	if _, err := addressCodec.BytesToString(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address %s: %s", authority, err))
	}

	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		storeService: storeService,
		cdc:          cdc,
		addressCodec: addressCodec,
		authority:    authority,

		bankKeeper: bankKeeper,
		authKeeper: authKeeper,
		Params:     collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
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
