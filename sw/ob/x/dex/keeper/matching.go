package keeper

import (
	"fmt"
	"sort"

	"ob/x/dex/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SettleAssets handles the swap between two parties
func (k Keeper) SettleAssets(ctx sdk.Context, taker, maker sdk.AccAddress, takerSide string, qty, quote math.Int, m types.Market) error {
	if takerSide == "BUY" {
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, taker, sdk.NewCoins(sdk.NewCoin(m.BaseDenom, qty))); err != nil {
			return err
		}
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, maker, sdk.NewCoins(sdk.NewCoin(m.QuoteDenom, quote))); err != nil {
			return err
		}
	} else {
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, taker, sdk.NewCoins(sdk.NewCoin(m.QuoteDenom, quote))); err != nil {
			return err
		}
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, maker, sdk.NewCoins(sdk.NewCoin(m.BaseDenom, qty))); err != nil {
			return err
		}
	}
	return nil
}

// HandleRefund returns unused escrowed funds
func (k Keeper) HandleRefund(ctx sdk.Context, addr sdk.AccAddress, remaining math.Int, side string, m types.Market) {
	if remaining.IsZero() {
		return
	}
	if side == "SELL" {
		_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, sdk.NewCoins(sdk.NewCoin(m.BaseDenom, remaining)))
	} else {
		// Refund logic for BUY side can be added here
	}
}

func (k Keeper) MatchOrders(ctx sdk.Context, takerOrder *types.Order, takerId string, market types.Market) error {
	targetSide := "SELL"
	if takerOrder.Side == "SELL" {
		targetSide = "BUY"
	}

	// 1. Fetch & Sort Candidates
	var candidates []types.Orderbook
	iter, err := k.Orderbook.Iterate(ctx, nil)
	if err != nil {
		return err
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		val, _ := iter.Value()
		if val.MarketId == takerOrder.MarketId && val.Side == targetSide {
			// Price compatibility check
			if takerOrder.OrderType == "LIMIT" {
				tPrice, _ := math.NewIntFromString(takerOrder.Price)
				mPrice, _ := math.NewIntFromString(val.Price)
				if takerOrder.Side == "BUY" && mPrice.GT(tPrice) {
					continue
				}
				if takerOrder.Side == "SELL" && mPrice.LT(tPrice) {
					continue
				}
			}
			candidates = append(candidates, val)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		pi, _ := math.NewIntFromString(candidates[i].Price)
		pj, _ := math.NewIntFromString(candidates[j].Price)
		if targetSide == "BUY" {
			return pi.GT(pj)
		}
		return pi.LT(pj)
	})

	takerRemaining, _ := math.NewIntFromString(takerOrder.Remaining)
	takerAddr, _ := sdk.AccAddressFromBech32(takerOrder.Creator)

	// 2. Execution Loop
	for _, entry := range candidates {
		if takerRemaining.IsZero() {
			break
		}

		makerOrder, _ := k.Order.Get(ctx, entry.OrderId)
		makerRemaining, _ := math.NewIntFromString(makerOrder.Remaining)
		makerAddr, _ := sdk.AccAddressFromBech32(makerOrder.Creator)

		matchQty := math.MinInt(takerRemaining, makerRemaining)
		tradePrice, _ := math.NewIntFromString(makerOrder.Price)
		totalQuote := matchQty.Mul(tradePrice)

		if err := k.SettleAssets(ctx, takerAddr, makerAddr, takerOrder.Side, matchQty, totalQuote, market); err != nil {
			return err
		}

		makerRemaining = makerRemaining.Sub(matchQty)
		takerRemaining = takerRemaining.Sub(matchQty)
		makerOrder.Remaining = makerRemaining.String()

		if makerRemaining.IsZero() {
			makerOrder.Status = "FILLED"
			_ = k.Orderbook.Remove(ctx, fmt.Sprintf("%s|%s|%s|%s", entry.MarketId, entry.Side, entry.Price, entry.OrderId))
		}
		_ = k.Order.Set(ctx, entry.OrderId, makerOrder)
	}

	// 3. Finalize Taker & Update Orderbook
	takerOrder.Remaining = takerRemaining.String()
	if takerRemaining.IsZero() {
		takerOrder.Status = "FILLED"
	} else {
		if takerOrder.OrderType == "MARKET" {
			takerOrder.Status = "CANCELLED_PARTIAL"
			k.HandleRefund(ctx, takerAddr, takerRemaining, takerOrder.Side, market)
		} else {
			takerOrder.Status = "PARTIAL"
			orderbookKey := fmt.Sprintf("%s|%s|%s|%s", takerOrder.MarketId, takerOrder.Side, takerOrder.Price, takerId)
			_ = k.Orderbook.Set(ctx, orderbookKey, types.Orderbook{
				MarketId: takerOrder.MarketId,
				Side:     takerOrder.Side,
				Price:    takerOrder.Price,
				OrderId:  takerId,
			})
		}
	}

	return k.Order.Set(ctx, takerId, *takerOrder)
}
