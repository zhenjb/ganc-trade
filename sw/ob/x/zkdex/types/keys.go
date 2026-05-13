package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "zkdex"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// GovModuleName duplicates the gov module's name to avoid a dependency with x/gov.
	// It should be synced with the gov module's name if it is ever changed.
	// See: https://github.com/cosmos/cosmos-sdk/blob/v0.52.0-beta.2/x/gov/types/keys.go#L9
	GovModuleName = "gov"
)

// ParamsKey is the prefix to retrieve all Params
var ParamsKey = collections.NewPrefix("p_zkdex")

// StateRootKey is the prefix to retrieve the state root
var StateRootKey = collections.NewPrefix("sr_zkdex")

// DepositRecordKey is the prefix to retrieve deposit records
var DepositRecordKey = collections.NewPrefix("dr_zkdex")

// WithdrawRecordKey is the prefix to retrieve withdraw records
var WithdrawRecordKey = collections.NewPrefix("wr_zkdex")

// NullifierUsedKey is the prefix to retrieve used nullifiers
var NullifierUsedKey = collections.NewPrefix("nu_zkdex")

// DepositProcessedKey is the prefix to retrieve processed deposits
var DepositProcessedKey = collections.NewPrefix("dp_zkdex")

// BatchRecordKey is the prefix to retrieve batch records
var BatchRecordKey = collections.NewPrefix("br_zkdex")

func KeyPrefix(p string) []byte {
	return []byte(p)
}