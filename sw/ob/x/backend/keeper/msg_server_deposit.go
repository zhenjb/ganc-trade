package keeper

import (
	"context"
	"errors"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"ob/x/backend/types"
)

func (k msgServer) Deposit(goCtx context.Context, msg *types.MsgDeposit) (*types.MsgDepositResponse, error) {
	// Unwrap context từ Go Context sang sdk.Context.
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg == nil {
		return nil, errorsmod.Wrap(errors.New("nil deposit message"), "invalid deposit")
	}
	if msg.Creator == "" {
		return nil, errorsmod.Wrap(errors.New("creator is required"), "invalid deposit")
	}
	if err := msg.Amount.Validate(); err != nil {
		return nil, errorsmod.Wrap(err, "invalid deposit amount")
	}
	if msg.Amount.IsZero() {
		return nil, errorsmod.Wrap(errors.New("deposit amount must be > 0"), "invalid deposit")
	}

	// Chuyển đổi address string -> sdk.AccAddress.
	depositor, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}

	// Chuyển token vào Module Account (lock onchain cho "backend").
	err = k.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		depositor,
		types.ModuleName,
		sdk.NewCoins(msg.Amount),
	)
	if err != nil {
		return nil, err
	}

	// Emit event để offchain có thể index/matching sau này.
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"backend_deposit",
			sdk.NewAttribute("depositor", depositor.String()),
			sdk.NewAttribute("amount", msg.Amount.String()),
		),
	)

	return &types.MsgDepositResponse{}, nil
}
