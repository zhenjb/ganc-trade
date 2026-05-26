package keeper

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"ob/x/zkdex/types"
)

func (k msgServer) Deposit(ctx context.Context, req *types.MsgDeposit) (*types.MsgDepositResponse, error) {
	// Validate creator address
	creatorAddr, err := k.addressCodec.StringToBytes(req.Creator)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "invalid creator address")
	}

	// Validate denom
	if req.Denom == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "denom cannot be empty")
	}

	// Validate amount
	if req.Amount == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "amount cannot be empty")
	}

	// Parse amount as sdk.Coin to validate format
	coins, err := sdk.ParseCoinNormalized(req.Amount + req.Denom)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, fmt.Sprintf("invalid amount/denom: %v", err))
	}

	// Check that the amount is positive
	if !coins.IsPositive() {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "deposit amount must be positive")
	}

	// Create deposit ID using a block-height prefix plus a random suffix.
	// This avoids collisions for multiple deposits within the same block.
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	randBytes := make([]byte, 4)
	if _, err := rand.Read(randBytes); err != nil {
		return nil, errorsmod.Wrap(err, "failed to generate deposit id")
	}
	depositID := fmt.Sprintf("dep-%s-%d-%s", req.Creator, sdkCtx.BlockHeight(), hex.EncodeToString(randBytes))

	// Create DepositRecord
	depositRecord := types.DepositRecord{
		DepositId:     depositID,
		Owner:         req.Creator,
		Denom:         req.Denom,
		Amount:        req.Amount,
		Processed:     false,
		CreatedHeight: sdkCtx.BlockHeight(),
	}

	// Call x/bank.SendCoinsFromAccountToModule
	// This transfers the coins from the creator's account to the zkdex module account
	err = k.Keeper.EscrowFunds(ctx, creatorAddr, sdk.NewCoins(coins))
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to escrow funds")
	}

	// Store the deposit record
	err = k.Keeper.SetDepositRecord(ctx, depositID, depositRecord)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to store deposit record")
	}

	// Emit event
	err = sdkCtx.EventManager().EmitTypedEvent(&types.EventDeposit{
		DepositId: depositID,
		Creator:   req.Creator,
		Denom:     req.Denom,
		Amount:    req.Amount,
	})
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to emit deposit event")
	}

	return &types.MsgDepositResponse{
		DepositRecord: &depositRecord,
	}, nil
}
