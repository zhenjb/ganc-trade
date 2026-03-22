package keeper

import (
	"context"

	"ob/x/dex/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) Withdraw(goCtx context.Context, msg *types.MsgWithdraw) (*types.MsgWithdrawResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.Keeper.Withdraw(ctx, msg.Creator, msg.Amount, msg.Denom); err != nil {
		return nil, err
	}

	return &types.MsgWithdrawResponse{}, nil
}
