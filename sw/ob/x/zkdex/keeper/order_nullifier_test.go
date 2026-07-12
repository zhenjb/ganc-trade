package keeper_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"ob/x/zkdex/types"
)

func TestOrderNullifierStore(t *testing.T) {
	f := initFixture(t)
	orderNullifier := "0x" + strings.Repeat("A", 64)
	normalized := strings.Repeat("a", 64)

	used, err := f.keeper.IsOrderNullifierUsed(f.ctx, normalized)
	require.NoError(t, err)
	t.Logf("before set: input=%s normalizedKey=%s used=%t", orderNullifier, normalized, used)
	require.False(t, used)

	require.NoError(t, f.keeper.SetOrderNullifierUsed(f.ctx, orderNullifier))

	used, err = f.keeper.IsOrderNullifierUsed(f.ctx, normalized)
	require.NoError(t, err)
	t.Logf("after set: input=%s normalizedKey=%s used=%t", orderNullifier, normalized, used)
	require.True(t, used)

	withdrawNullifierUsed, err := f.keeper.IsNullifierUsed(f.ctx, normalized)
	require.NoError(t, err)
	t.Logf("separate withdrawal nullifier store: normalizedKey=%s used=%t", normalized, withdrawNullifierUsed)
	require.False(t, withdrawNullifierUsed)
}

func TestOrderNullifierGenesisRoundTrip(t *testing.T) {
	f := initFixture(t)
	genesisState := types.GenesisState{
		Params:           types.DefaultParams(),
		CurrentStateRoot: types.DefaultStateRoot,
		OrderNullifierUsed: []string{
			"0x" + strings.Repeat("A", 64),
			strings.Repeat("b", 64),
		},
	}

	require.NoError(t, f.keeper.InitGenesis(f.ctx, genesisState))
	t.Logf("init genesis orderNullifierUsed input=%v", genesisState.OrderNullifierUsed)

	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	t.Logf("export genesis orderNullifierUsed output=%v", got.OrderNullifierUsed)
	require.ElementsMatch(t, []string{
		strings.Repeat("a", 64),
		strings.Repeat("b", 64),
	}, got.OrderNullifierUsed)
}

func TestOrderNullifierAgreementOutput(t *testing.T) {
	f := initFixture(t)
	orderNullifier := "0x" + strings.Repeat("B", 64)

	p1Submit := struct {
		Actor            string `json:"actor"`
		Message          string `json:"message"`
		BatchID          string `json:"batchId"`
		OldStateRoot     string `json:"oldStateRoot"`
		NewStateRoot     string `json:"newStateRoot"`
		OrderHash        string `json:"orderHash"`
		OrderNullifier   string `json:"orderNullifier"`
		NullifierStoreOp string `json:"nullifierStoreOp"`
	}{
		Actor:            "P1",
		Message:          "MsgSubmitBatchProof.settlementUpdate.trades[]",
		BatchID:          "batch-demo-1",
		OldStateRoot:     types.DefaultStateRoot,
		NewStateRoot:     "0x" + strings.Repeat("d", 64),
		OrderHash:        "0x" + strings.Repeat("a", 64),
		OrderNullifier:   orderNullifier,
		NullifierStoreOp: "mark used after proof verification passes",
	}
	p1SubmitJSON, err := json.MarshalIndent(p1Submit, "", "  ")
	require.NoError(t, err)
	t.Logf("P1 submit sample:\n%s", p1SubmitJSON)

	require.NoError(t, f.keeper.SetOrderNullifierUsed(f.ctx, orderNullifier))
	normalized, err := types.NormalizeOrderNullifier(orderNullifier)
	require.NoError(t, err)
	used, err := f.keeper.IsOrderNullifierUsed(f.ctx, "0x"+normalized)
	require.NoError(t, err)

	p4Read := struct {
		Actor          string `json:"actor"`
		Query          string `json:"query"`
		OrderNullifier string `json:"orderNullifier"`
		StoreKey       string `json:"storeKey"`
		Used           bool   `json:"used"`
		Note           string `json:"note"`
	}{
		Actor:          "P4",
		Query:          "keeper.IsOrderNullifierUsed",
		OrderNullifier: "0x" + normalized,
		StoreKey:       normalized,
		Used:           used,
		Note:           "ONCHAIN-T02 store read; ONCHAIN-T06 will wrap this as a public query",
	}
	p4ReadJSON, err := json.MarshalIndent(p4Read, "", "  ")
	require.NoError(t, err)
	t.Logf("P4 query sample:\n%s", p4ReadJSON)
}

func TestOrderNullifierRawKVStoreOutput(t *testing.T) {
	f := initFixture(t)
	orderNullifier := "0x" + strings.Repeat("C", 64)
	normalized := strings.Repeat("c", 64)

	require.NoError(t, f.keeper.SetOrderNullifierUsed(f.ctx, orderNullifier))

	store := f.storeService.OpenKVStore(f.ctx)
	iter, err := store.Iterator(nil, nil)
	require.NoError(t, err)
	defer iter.Close()

	type rawKVPair struct {
		KeyHex     string `json:"keyHex"`
		KeyASCII   string `json:"keyAscii"`
		ValueHex   string `json:"valueHex"`
		ValueASCII string `json:"valueAscii"`
	}

	var allPairs []rawKVPair
	var orderNullifierPairs []rawKVPair
	for ; iter.Valid(); iter.Next() {
		key := append([]byte(nil), iter.Key()...)
		value := append([]byte(nil), iter.Value()...)
		pair := rawKVPair{
			KeyHex:     hex.EncodeToString(key),
			KeyASCII:   string(key),
			ValueHex:   hex.EncodeToString(value),
			ValueASCII: string(value),
		}
		allPairs = append(allPairs, pair)
		if bytes.Contains(key, []byte("onu_zkdex")) || bytes.Contains(key, []byte(normalized)) {
			orderNullifierPairs = append(orderNullifierPairs, pair)
		}
	}
	require.NotEmpty(t, orderNullifierPairs)
	require.Contains(t, orderNullifierPairs[0].KeyASCII, normalized)
	require.Equal(t, string(types.OrderNullifierUsedKey([]byte(normalized))), orderNullifierPairs[0].KeyASCII)
	require.Equal(t, "01", orderNullifierPairs[0].ValueHex)

	allPairsJSON, err := json.MarshalIndent(allPairs, "", "  ")
	require.NoError(t, err)
	t.Logf("actual raw zkdex KVStore after SetOrderNullifierUsed:\n%s", allPairsJSON)

	orderPairsJSON, err := json.MarshalIndent(orderNullifierPairs, "", "  ")
	require.NoError(t, err)
	t.Logf("actual raw order-nullifier KV pair(s):\n%s", orderPairsJSON)
}
