package zkdex

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	"ob/x/zkdex/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: types.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				{
					RpcMethod: "ModuleAccountAddress",
					Use:       "module-account-address",
					Short:     "Shows the zkdex module account address",
				},
				{
					RpcMethod:      "ModuleAccountBalance",
					Use:            "module-account-balance [denom]",
					Short:          "Shows the zkdex module account spendable balance",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "denom", Optional: true}},
				},
				{
					RpcMethod: "CurrentStateRoot",
					Use:       "current-state-root",
					Short:     "Shows the current zkdex state root",
				},
				{
					RpcMethod:      "DepositRecord",
					Use:            "deposit-record [deposit-id]",
					Short:          "Shows a zkdex deposit record",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "deposit_id"}},
				},
				{
					RpcMethod:      "DepositProcessed",
					Use:            "deposit-processed [deposit-id]",
					Short:          "Shows whether a zkdex deposit has been processed",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "deposit_id"}},
				},
				{
					RpcMethod:      "WithdrawRecord",
					Use:            "withdraw-record [withdraw-id]",
					Short:          "Shows a zkdex withdraw record",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "withdraw_id"}},
				},
				{
					RpcMethod:      "NullifierUsed",
					Use:            "nullifier-used [nullifier]",
					Short:          "Shows whether a zkdex nullifier has been used",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "nullifier"}},
				},
				{
					RpcMethod:      "BatchRecord",
					Use:            "batch-record [batch-id]",
					Short:          "Shows a zkdex batch record",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "batch_id"}},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              types.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod:      "Deposit",
					Use:            "deposit [denom] [amount]",
					Short:          "Send a zkdex deposit tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "denom"}, {ProtoField: "amount"}},
				},
				{
					RpcMethod:      "ClaimWithdraw",
					Use:            "claim-withdraw [withdraw-id]",
					Short:          "Claim a settled zkdex withdrawal",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "withdraw_id"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
