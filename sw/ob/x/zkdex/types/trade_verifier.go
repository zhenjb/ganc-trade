package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"slices"
	"strings"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/witness"
)

const tradePublicInputCount = 8

// GazkTradeV1VerifyingKeyHex must be the verifyingKey field from
// gazk-trade-v1.verifier-artifact.json exported by the matching gazk key store.
const GazkTradeV1VerifyingKeyHex = "0xd66f3e1a87e6fa38935f9d7c7350e494693d2c8b0979fe793106caccb0526b2dd0b6e085bf6c8189f34e1b3fcef62fbf282b3af8c2195c0438ffa4cde9b9b073c02c388b35e08552ed609ecbf89ecc41eff6315a5554860b65bad0903665e19f0b81141057fa6572a68bfb3b2e59b3bb8b60c8c831d67c6e202c4744205311c2c19425d6835ba961901fe8371fe68d64dc8ca43930f179ceb1e778d61be101e321327aece0a3b7d7f70812fc3779d61f0289d818c15c711af97679d5f5cc4dcacdb7a7ca41b3a8f05077f5c9df20e7644e7eeebe3378a3d0f2c521153aa19861de5f91634a86098b9042bb3f68005b465c4f9dbf6ff124d9a45f40ce671630f8004da135cd4196833430401de65ee5c0ed896e7927f23c3c1a0220919745304f00000009d5d9fb072084b4004a0690750181bc4824b3332234869300f788da6bf5907307dd8d460bd0a5d47d37d408878173ec9be3352d8b909e4280da14765febda363ca354f609b0321dc0973c65c05096c7ebe792dc0f950b53a88f4d69901525ace889baef358fb3268c5b52c4ed0c25c8683c7948f504ec6abcca30b1b4781d789fe2c9ad86b887037f963178c8fa80360ce50d222b6af4c63f468ceb8d12891607ece0c598aa4eca59c57f9cff0f397a0f7e435750e2e9006d06c378bda4d5c2fae3f83f5ab92192bd443f46e32993cf2bdb7879d1093e85ec4cae2da369d1360ac4c47688bdd988723611255ea36d824e275716fb09204f2718349a2c752e4606dca5f4626cb4dc91d8ca0eeda5d6dca46ce32e698550822ab17f6ef677749a440000000000000000"

type TradeProofVerifier struct {
	verifyingKeys map[string]groth16.VerifyingKey
}

type tradeVerifierUpdate struct {
	PublicInputs []string `json:"publicInputs"`
}

type tradeProofBundle struct {
	Proof             string   `json:"proof"`
	PublicInputs      []string `json:"publicInputs"`
	VKID              string   `json:"vkId"`
	VerificationKeyID string   `json:"verificationKeyId"`
}

func NewTradeProofVerifier(vkHexByID map[string]string) TradeProofVerifier {
	verifier := TradeProofVerifier{verifyingKeys: make(map[string]groth16.VerifyingKey, len(vkHexByID))}
	for vkID, vkHex := range vkHexByID {
		vk, err := readBN254VerifyingKeyHex(vkHex)
		if err != nil {
			continue
		}
		verifier.verifyingKeys[vkID] = vk
	}
	return verifier
}

func (v TradeProofVerifier) VerifyProof(update []byte, proofBundle []byte) bool {
	var verifierUpdate tradeVerifierUpdate
	if err := json.Unmarshal(update, &verifierUpdate); err != nil {
		return false
	}
	if len(verifierUpdate.PublicInputs) != tradePublicInputCount {
		return false
	}

	var bundle tradeProofBundle
	if err := json.Unmarshal(proofBundle, &bundle); err != nil {
		return false
	}
	vkID := bundle.VKID
	if vkID == "" {
		vkID = bundle.VerificationKeyID
	}
	if vkID == "" {
		vkID = TradeVerifierVKID
	}
	if vkID != TradeVerifierVKID {
		return false
	}
	if !slices.Equal(bundle.PublicInputs, verifierUpdate.PublicInputs) {
		return false
	}
	vk, ok := v.verifyingKeys[vkID]
	if !ok {
		return false
	}

	proof, err := readBN254ProofHex(bundle.Proof)
	if err != nil {
		return false
	}
	publicWitness, err := buildTradePublicWitness(verifierUpdate.PublicInputs)
	if err != nil {
		return false
	}
	return groth16.Verify(proof, vk, publicWitness) == nil
}

func readBN254VerifyingKeyHex(vkHex string) (groth16.VerifyingKey, error) {
	raw, err := decodeHexField(vkHex)
	if err != nil {
		return nil, err
	}
	vk := groth16.NewVerifyingKey(ecc.BN254)
	if _, err := vk.ReadFrom(bytes.NewReader(raw)); err != nil {
		return nil, err
	}
	return vk, nil
}

func readBN254ProofHex(proofHex string) (groth16.Proof, error) {
	raw, err := decodeHexField(proofHex)
	if err != nil {
		return nil, err
	}
	proof := groth16.NewProof(ecc.BN254)
	if _, err := proof.ReadFrom(bytes.NewReader(raw)); err != nil {
		return nil, err
	}
	return proof, nil
}

func buildTradePublicWitness(publicInputs []string) (witness.Witness, error) {
	w, err := witness.New(ecc.BN254.ScalarField())
	if err != nil {
		return nil, err
	}
	values := make(chan any, len(publicInputs))
	for _, input := range publicInputs {
		value, err := publicInputBigInt(input)
		if err != nil {
			close(values)
			return nil, err
		}
		values <- value
	}
	close(values)
	if err := w.Fill(len(publicInputs), 0, values); err != nil {
		return nil, err
	}
	return w, nil
}

func publicInputBigInt(input string) (*big.Int, error) {
	raw, err := decodeHexField(input)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(raw), nil
}

func decodeHexField(value string) ([]byte, error) {
	raw := strings.TrimPrefix(strings.TrimSpace(value), "0x")
	return hex.DecodeString(raw)
}
