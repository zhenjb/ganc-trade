package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"ob/x/zkdex/types"
)

var _ types.QueryServer = queryServer{}

// NewQueryServerImpl returns an implementation of the QueryServer interface
// for the provided Keeper.
func NewQueryServerImpl(k Keeper) types.QueryServer {
	return queryServer{k}
}

type queryServer struct {
	k Keeper
}

func (q queryServer) ModuleAccountAddress(ctx context.Context, req *types.QueryModuleAccountAddressRequest) (*types.QueryModuleAccountAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	addr := q.k.GetModuleAccountAddressString()
	return &types.QueryModuleAccountAddressResponse{Address: addr}, nil
}

func (q queryServer) ModuleAccountBalance(ctx context.Context, req *types.QueryModuleAccountBalanceRequest) (*types.QueryModuleAccountBalanceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	coins := q.k.GetModuleAccountBalance(ctx)
	return &types.QueryModuleAccountBalanceResponse{Balance: coins.String()}, nil
}

func (q queryServer) CurrentStateRoot(ctx context.Context, req *types.QueryCurrentStateRootRequest) (*types.QueryCurrentStateRootResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	root, err := q.k.GetStateRoot(ctx)
	if err != nil {
		return nil, err
	}
	return &types.QueryCurrentStateRootResponse{StateRoot: root}, nil
}

func (q queryServer) DepositRecord(ctx context.Context, req *types.QueryDepositRecordRequest) (*types.QueryDepositRecordResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	record, err := q.k.GetDepositRecord(ctx, req.DepositId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "deposit record not found")
	}
	return &types.QueryDepositRecordResponse{Record: &record}, nil
}

func (q queryServer) WithdrawRecord(ctx context.Context, req *types.QueryWithdrawRecordRequest) (*types.QueryWithdrawRecordResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	record, err := q.k.GetWithdrawRecord(ctx, req.WithdrawId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "withdraw record not found")
	}
	return &types.QueryWithdrawRecordResponse{Record: &record}, nil
}

func (q queryServer) NullifierUsed(ctx context.Context, req *types.QueryNullifierUsedRequest) (*types.QueryNullifierUsedResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	used, err := q.k.IsNullifierUsed(ctx, req.Nullifier)
	if err != nil {
		return nil, err
	}
	return &types.QueryNullifierUsedResponse{Used: used}, nil
}

func (q queryServer) DepositProcessed(ctx context.Context, req *types.QueryDepositProcessedRequest) (*types.QueryDepositProcessedResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	processed, err := q.k.IsDepositProcessed(ctx, req.DepositId)
	if err != nil {
		return nil, err
	}
	return &types.QueryDepositProcessedResponse{Processed: processed}, nil
}

func (q queryServer) BatchRecord(ctx context.Context, req *types.QueryBatchRecordRequest) (*types.QueryBatchRecordResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	record, err := q.k.GetBatchRecord(ctx, req.BatchId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "batch record not found")
	}
	return &types.QueryBatchRecordResponse{Record: &record}, nil
}
