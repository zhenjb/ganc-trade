package types

// MsgWithdraw is the message to withdraw coins from the dex module account back to the user.
type MsgWithdraw struct {
	Creator string
	Amount  string
	Denom   string
}

type MsgWithdrawResponse struct{}
