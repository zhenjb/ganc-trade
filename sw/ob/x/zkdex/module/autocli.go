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
					RpcMethod: "ModuleAccountBalance",
					Use:       "module-account-balance",
					Short:     "Shows the zkdex module account spendable balance",
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
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
