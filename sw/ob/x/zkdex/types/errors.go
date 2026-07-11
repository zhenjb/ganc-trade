package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/zkdex module sentinel errors
var (
	ErrInvalidSigner        = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrOrderNullifierReused = errors.Register(ModuleName, 1101, "order nullifier already used")
)
