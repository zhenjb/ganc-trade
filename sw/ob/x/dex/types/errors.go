package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/dex module sentinel errors
var (
	ErrInvalidSigner  = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidAddress = errors.Register(ModuleName, 1101, "invalid address")
	ErrInvalidAmount  = errors.Register(ModuleName, 1102, "invalid amount")
)
