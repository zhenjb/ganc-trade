package keeper

import (
	"context"
	"fmt"

	"ob/x/dex/types"
	"cosmossdk.io/math"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) PlaceOrder(goCtx context.Context, msg *types.MsgPlaceOrder) (*types.MsgPlaceOrderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	market, err := k.Keeper.Market.Get(ctx, msg.MarketId)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrKeyNotFound, "market %s not found", msg.MarketId)
	}

	quantity, ok := math.NewIntFromString(msg.Quantity)
	if !ok {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "invalid quantity format")
	}

	orderType := "LIMIT"
	if msg.Price == "0" {
		orderType = "MARKET"
	}

	creatorAddr, _ := sdk.AccAddressFromBech32(msg.Creator)
	var escrowCoins sdk.Coins

	if msg.Side == "BUY" {
		price, _ := math.NewIntFromString(msg.Price)
		if orderType == "MARKET" {
			escrowCoins = sdk.NewCoins(sdk.NewCoin(market.QuoteDenom, price)) 
		} else {
			escrowCoins = sdk.NewCoins(sdk.NewCoin(market.QuoteDenom, price.Mul(quantity)))
		}
	} else {
		escrowCoins = sdk.NewCoins(sdk.NewCoin(market.BaseDenom, quantity))
	}

	// Escrow Logic 
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAddr, types.ModuleName, escrowCoins); err != nil {
		return nil, err
	}

	// Order Init
	orderId := fmt.Sprintf("%s-%d-%s", msg.MarketId, ctx.BlockHeight(), msg.Creator[:6])
	newOrder := types.Order{
		MarketId:  msg.MarketId,
		OrderType: orderType,
		Side:      msg.Side,
		Price:     msg.Price,
		Quantity:  msg.Quantity,
		Remaining: msg.Quantity,
		Status:    "OPEN",
		Creator:   msg.Creator,
	}

	// Call MatchOrders from Keeper
	if err := k.Keeper.MatchOrders(ctx, &newOrder, orderId, market); err != nil {
		return nil, err
	}

	return &types.MsgPlaceOrderResponse{OrderId: orderId}, nil
}