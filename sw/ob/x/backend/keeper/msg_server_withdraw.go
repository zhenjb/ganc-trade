package keeper

import (
	"context"
	"errors"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"ob/x/backend/types"
)

func (k msgServer) Withdraw(goCtx context.Context, msg *types.MsgWithdraw) (*types.MsgWithdrawResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg == nil {
		return nil, errorsmod.Wrap(errors.New("nil withdraw message"), "invalid withdraw")
	}
	if msg.Creator == "" {
		return nil, errorsmod.Wrap(errors.New("creator is required"), "invalid withdraw")
	}
	if err := msg.Amount.Validate(); err != nil {
		return nil, errorsmod.Wrap(err, "invalid withdraw amount")
	}
	if msg.Amount.IsZero() {
		return nil, errorsmod.Wrap(errors.New("withdraw amount must be > 0"), "invalid withdraw")
	}

	// Convert the address string to sdk.AccAddress.
	recipient, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}

	// Transfer tokens from the backend module account back to the user.
	err = k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		types.ModuleName,
		recipient,
		sdk.NewCoins(msg.Amount),
	)
	if err != nil {
		return nil, err
	}

	// Emit events so that off-chain indexing can track withdrawals.
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"backend_withdraw",
			sdk.NewAttribute("recipient", recipient.String()),
			sdk.NewAttribute("amount", msg.Amount.String()),
		),
	)

	return &types.MsgWithdrawResponse{}, nil
}
