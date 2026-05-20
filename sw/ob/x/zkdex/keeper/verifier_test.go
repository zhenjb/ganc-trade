package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"ob/x/zkdex/types"
)

func TestProofVerifierDefaultRejects(t *testing.T) {
	f := initFixture(t)

	// keeper chưa được gắn 1 bộ xác thực (verifier) nào (trống) mà có y/c gửi proof -> reject
	accepted := f.keeper.VerifyProof([]byte(`{"oldRoot":"0xold","newRoot":"0xnew"}`), []byte(`{"proof":"stub"}`))
	t.Logf("DEFAULT verifier result: accepted=%v", accepted)

	require.False(t, accepted)
}

func TestProofVerifierStubCanAcceptAndReject(t *testing.T) {
	f := initFixture(t)
	update := []byte(`{"oldRoot":"0xold","newRoot":"0xnew","depositIds":["dep-1"]}`)
	proofBundle := []byte(`{"proof":"temporary-stub-proof"}`)

	// Tạo 1 proof giả điều khiển nó là đúng
	acceptingKeeper := f.keeper.WithProofVerifier(types.StubProofVerifier{Accept: true})
	accepted := acceptingKeeper.VerifyProof(update, proofBundle)
	t.Logf("STUB accepting verifier: update=%s proofBundle=%s accepted=%v", update, proofBundle, accepted)
	require.True(t, accepted)

	rejectingKeeper := f.keeper.WithProofVerifier(types.StubProofVerifier{Accept: false})
	accepted = rejectingKeeper.VerifyProof(update, proofBundle)
	t.Logf("STUB rejecting verifier: update=%s proofBundle=%s accepted=%v", update, proofBundle, accepted)
	require.False(t, accepted)
}

// test rằng có thể dùng một function bình thường làm verifier
func TestProofVerifierFuncAdapter(t *testing.T) {
	f := initFixture(t)
	update := []byte("expected-update")
	proofBundle := []byte("expected-proof")

	adapter := types.ProofVerifierFunc(func(gotUpdate []byte, gotProofBundle []byte) bool {
		t.Logf("ADAPTER called with update=%s proofBundle=%s", gotUpdate, gotProofBundle)
		return string(gotUpdate) == string(update) && string(gotProofBundle) == string(proofBundle)
	})

	k := f.keeper.WithProofVerifier(adapter)
	require.True(t, k.VerifyProof(update, proofBundle))
	require.False(t, k.VerifyProof([]byte("tampered-update"), proofBundle))
}
