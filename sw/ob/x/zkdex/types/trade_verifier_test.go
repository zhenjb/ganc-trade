package types

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTradeProofVerifierRejectsWhenVKMissing(t *testing.T) {
	verifier := NewTradeProofVerifier(map[string]string{TradeVerifierVKID: ""})
	update := tradeVerifierUpdate{PublicInputs: validTradeVerifierPublicInputs()}
	updateJSON, err := json.Marshal(update)
	require.NoError(t, err)

	bundle := tradeProofBundle{
		Proof:        "0x" + strings.Repeat("00", 128),
		PublicInputs: update.PublicInputs,
		VKID:         TradeVerifierVKID,
	}
	bundleJSON, err := json.Marshal(bundle)
	require.NoError(t, err)

	require.False(t, verifier.VerifyProof(updateJSON, bundleJSON))
}

func TestDefaultProofVerifierLoadsEmbeddedTradeVK(t *testing.T) {
	verifier, ok := DefaultProofVerifier().(TradeProofVerifier)
	require.True(t, ok)
	require.Contains(t, verifier.verifyingKeys, TradeVerifierVKID)
	require.Len(t, GazkTradeV1VerifyingKeyHex, 1178)
}

func TestTradeProofVerifierAcceptsRealGazkProofAndRejectsTamper(t *testing.T) {
	update := tradeVerifierUpdate{PublicInputs: realGazkTradePublicInputs()}
	updateJSON, err := json.Marshal(update)
	require.NoError(t, err)

	bundle := tradeProofBundle{
		Proof:             realGazkTradeProof,
		PublicInputs:      update.PublicInputs,
		VerificationKeyID: TradeVerifierVKID,
	}
	bundleJSON, err := json.Marshal(bundle)
	require.NoError(t, err)

	verifier := DefaultProofVerifier()
	require.True(t, verifier.VerifyProof(updateJSON, bundleJSON))

	tamperedInputs := realGazkTradePublicInputs()
	tamperedInputs[7] = "0x" + strings.Repeat("00", 32)
	tamperedBundle := bundle
	tamperedBundle.PublicInputs = tamperedInputs
	tamperedBundleJSON, err := json.Marshal(tamperedBundle)
	require.NoError(t, err)
	require.False(t, verifier.VerifyProof(updateJSON, tamperedBundleJSON))

	fakeProofBundle := bundle
	fakeProofBundle.Proof = "0x" + strings.Repeat("00", 128)
	fakeProofBundleJSON, err := json.Marshal(fakeProofBundle)
	require.NoError(t, err)
	require.False(t, verifier.VerifyProof(updateJSON, fakeProofBundleJSON))
}

func TestTradeProofVerifierRejectsPublicInputMismatch(t *testing.T) {
	verifier := NewTradeProofVerifier(map[string]string{TradeVerifierVKID: "0x00"})
	update := tradeVerifierUpdate{PublicInputs: validTradeVerifierPublicInputs()}
	updateJSON, err := json.Marshal(update)
	require.NoError(t, err)

	bundleInputs := validTradeVerifierPublicInputs()
	bundleInputs[1] = "0x" + strings.Repeat("ff", 32)
	bundle := tradeProofBundle{
		Proof:        "0x" + strings.Repeat("00", 128),
		PublicInputs: bundleInputs,
		VKID:         TradeVerifierVKID,
	}
	bundleJSON, err := json.Marshal(bundle)
	require.NoError(t, err)

	require.False(t, verifier.VerifyProof(updateJSON, bundleJSON))
}

func TestTradeProofVerifierUsesDefaultTradeVKIDWhenBundleOmitsID(t *testing.T) {
	verifier := NewTradeProofVerifier(map[string]string{TradeVerifierVKID: ""})
	update := tradeVerifierUpdate{PublicInputs: validTradeVerifierPublicInputs()}
	updateJSON, err := json.Marshal(update)
	require.NoError(t, err)

	bundle := tradeProofBundle{
		Proof:        "0x" + strings.Repeat("00", 128),
		PublicInputs: update.PublicInputs,
	}
	bundleJSON, err := json.Marshal(bundle)
	require.NoError(t, err)

	require.False(t, verifier.VerifyProof(updateJSON, bundleJSON))
}

func TestBuildTradePublicWitnessUsesEightInputs(t *testing.T) {
	w, err := buildTradePublicWitness(validTradeVerifierPublicInputs())
	require.NoError(t, err)
	publicW, err := w.Public()
	require.NoError(t, err)
	require.Len(t, publicW.Vector(), tradePublicInputCount)
}

func TestBuildTradePublicWitnessRejectsBadHex(t *testing.T) {
	publicInputs := validTradeVerifierPublicInputs()
	publicInputs[0] = "0xnothex"

	_, err := buildTradePublicWitness(publicInputs)
	require.Error(t, err)
}

func validTradeVerifierPublicInputs() []string {
	inputs := make([]string, tradePublicInputCount)
	for i := range inputs {
		inputs[i] = "0x" + hex.EncodeToString([]byte{
			byte(i), byte(i), byte(i), byte(i), byte(i), byte(i), byte(i), byte(i),
			byte(i), byte(i), byte(i), byte(i), byte(i), byte(i), byte(i), byte(i),
			byte(i), byte(i), byte(i), byte(i), byte(i), byte(i), byte(i), byte(i),
			byte(i), byte(i), byte(i), byte(i), byte(i), byte(i), byte(i), byte(i),
		})
	}
	return inputs
}

const realGazkTradeProof = "0xe4b3cf6f1548ca6fdd311f938265de1c8d45e723c855d86dc91ed44ebcc6f665e504ad60954d18feeb87e4421cbff0ea1cb6f5955db5fd64c8c6d93ae8930d04287cd895782f6e3eceba1392841855c6bfca79b5b6d3211e6e87ecb6497220a5c9a9e7bd35a3d21a76628256e6820a04c99f781417312391b69d30b498ccb844000000004000000000000000000000000000000000000000000000000000000000000000"

func realGazkTradePublicInputs() []string {
	return []string{
		"0x0a4d64ea88ef3cc9aba4acfecb0265bab6827c7dcf098ad9159ecd0abe581f20",
		"0x163344c18521a703479fee2735e0774ea55807d8708e1ce5fb1e60b476ec13a8",
		"0x0d5927e78698013a0fd5f14aa2e4a5839f76b72b6c66ccfc0f4d82e54514f6ee",
		"0x254d20a0127f09e171a08cb3989c1226f9342586d5c1b14bac101bc1a4a5a7e4",
		"0x0fbda30e91ff91a81172d80bd6ddea3357cc9792740ed401ae03bf4bb6b49580",
		"0x1b61e64bfdfd5dab18034894d36fc218dedf961eaf097af54f03dd088c60a8b2",
		"0x248b83eaa221b628ddc792d382383fb91364a0f576517b4163f352d189e4a243",
		"0x03086491bf9b2331cdd798742caf45142d40e3cf462cf6fa4f2363067067fb48",
	}
}
