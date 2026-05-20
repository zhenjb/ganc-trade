package types

// ProofVerifier is the verifier boundary used by on-chain settlement messages.
// P2 can replace the stub implementation with a real ZK verifier without
// changing MsgSubmitBatchProof wiring.
type ProofVerifier interface {
	VerifyProof(update []byte, proofBundle []byte) bool
}

type ProofVerifierFunc func(update []byte, proofBundle []byte) bool

func (f ProofVerifierFunc) VerifyProof(update []byte, proofBundle []byte) bool {
	if f == nil {
		return false
	}
	return f(update, proofBundle)
}

// RejectingProofVerifier is the safe default until a stub or real verifier is
// explicitly wired in.
type RejectingProofVerifier struct{}

func (RejectingProofVerifier) VerifyProof(update []byte, proofBundle []byte) bool {
	return false
}

// StubProofVerifier is a temporary adapter for local integration while the real
// verifier is being built. Use Accept=false for negative-path tests.
type StubProofVerifier struct {
	Accept bool
}

func (v StubProofVerifier) VerifyProof(update []byte, proofBundle []byte) bool {
	return v.Accept
}
