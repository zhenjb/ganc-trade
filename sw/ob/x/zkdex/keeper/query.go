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
