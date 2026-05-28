package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"ob/x/zkdex/types"
)

func (k msgServer) ClaimWithdraw(ctx context.Context, req *types.MsgClaimWithdraw) (*types.MsgClaimWithdrawResponse, error) {
	if req == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "request cannot be nil")
	}
	creatorAddr, err := k.addressCodec.StringToBytes(req.Creator)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "invalid creator address")
	}
	if req.WithdrawId == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "withdrawId cannot be empty")
	}

	record, err := k.Keeper.GetWithdrawRecord(ctx, req.WithdrawId)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "withdraw %s not found", req.WithdrawId)
	}
	if record.Claimed {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "withdraw %s already claimed", req.WithdrawId)
	}
	if record.Destination == "" {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "withdraw %s destination cannot be empty", req.WithdrawId)
	}
	if record.Destination != req.Creator {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "creator must match withdraw destination")
	}

	coin, err := sdk.ParseCoinNormalized(record.Amount + record.Denom)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid withdraw amount/denom for %s", req.WithdrawId)
	}
	if !coin.IsPositive() {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "withdraw %s amount must be positive", req.WithdrawId)
	}

	if err := k.Keeper.ReleaseFunds(ctx, sdk.AccAddress(creatorAddr), sdk.NewCoins(coin)); err != nil {
		return nil, errorsmod.Wrap(err, "failed to release funds")
	}

	record.Claimed = true
	if err := k.Keeper.SetWithdrawRecord(ctx, req.WithdrawId, record); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to mark withdraw %s claimed", req.WithdrawId)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zkdex_withdraw_claimed",
		sdk.NewAttribute("withdraw_id", record.WithdrawId),
		sdk.NewAttribute("creator", req.Creator),
		sdk.NewAttribute("denom", record.Denom),
		sdk.NewAttribute("amount", record.Amount),
	))

	return &types.MsgClaimWithdrawResponse{WithdrawRecord: &record}, nil
}
