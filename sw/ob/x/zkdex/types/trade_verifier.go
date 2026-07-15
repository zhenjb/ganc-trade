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
const GazkTradeV1VerifyingKeyHex = "0x91d157df050fb5dc9366961cbb2c319add0f4c0f9e2486ca801dc6d84de3661b865fae33815dee73d18ffd2636a885bff640283972d81dd0a66e962b0104d49dc72b4f49f26f96376eccfde58bfa5c0742fd6c1bf6d9e47bcc475ae8f6014809248438abf70c7c80c47f116f09dedb47b265b6d42884e443f2ae5c0e35bb62138ee4fcf27dd595fd75e7e70856266ff9add7e89d914d4272a02d9e59cb711a3b2c21d0659b57ce8f0578a4fb8500e740525c5171d5f09dade1cf9c8b6129fc3fccaacfff2f2651e18a2e52fb8a254e33178d4b262fffca8336d3178dbe85f962c974d472fa965ca0be2a6e7a278274a3cbfcd6e5d338f8ce28ed15ee029f92f4058bcf325f0502ced36739487c6ef2efe04372dcb21d069bd5be8ab090d30e4f00000009e86f9f7d34a5c33b703e1902e83c1bd475e349e2de48d35340ba0ddd4e4f8669a045951b7686ea6913329754596e47ba24d54f84137e001d5bc3eac03ab1dbb8dc0272acefcb4a2d4b11eb6543e89d761ca26f819bc1e0e4b90b1c0b786f004d86656a1b13103559dec63d82f3b3de1e8db3814b925c0a88f8ad87de6175159890bff21a69859ba9b7a3b89a279bcd010f490c56c4e1e8fc3b37409b13b6fff1a1ff274c34aec8ff5fbfc38ea994215cfef5cf24b313b681d9888f8cae057af9d987dcb5cfdc62fd493cf7c0d34c502e2692a7ec4b1ba1160f36d84b3116b526adb4ed27494f74b859542392eac1aa2a344240410c1f0c3edd5df1ab64b383d8a18057f8b5abd9d7ae8e751dbd037bb3077745d057983c6ccaf59100b97b18560000000000000000"

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
