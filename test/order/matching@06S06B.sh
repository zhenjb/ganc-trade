#!/bin/bash

set -euo pipefail

CHAIN_ID="${CHAIN_ID:-ob}"
BASE_DENOM="${BASE_DENOM:-uatom}"
QUOTE_DENOM="${QUOTE_DENOM:-uusdc}"
MARKET_ID="${BASE_DENOM}-${QUOTE_DENOM}"
WAIT_SECONDS="${WAIT_SECONDS:-1}"

run_tx() {
  local label="$1"
  shift

  echo "[TX] $label"
  local result
  result=$("$@" --chain-id "$CHAIN_ID" -y -o json)

  local code
  code=$(echo "$result" | jq -r '.code // 0')
  if [[ "$code" != "0" ]]; then
    echo "[ERR] broadcast failed: $label code=$code" >&2
    echo "$result" | jq . >&2
    exit 1
  fi

  local txhash
  txhash=$(echo "$result" | jq -r '.txhash')
  echo "[FIN] txhash=$txhash"
  sleep "$WAIT_SECONDS"

  local query_result
  query_result=$(obd q tx "$txhash" -o json)

  local deliver_code
  deliver_code=$(echo "$query_result" | jq -r '.code // 0')
  if [[ "$deliver_code" != "0" ]]; then
    echo "[ERR] on-chain failed: $label code=$deliver_code" >&2
    echo "$query_result" | jq . >&2
    exit 1
  fi
}

query_state() {
  obd q dex list-market
  sleep "$WAIT_SECONDS"
  obd q dex list-order
  sleep "$WAIT_SECONDS"
  obd q dex list-orderbook
  sleep "$WAIT_SECONDS"
}

run_tx "register market ${MARKET_ID}" \
  obd tx dex register-pairs "$BASE_DENOM" "$QUOTE_DENOM" 0.01 1 --from alice

query_state

run_tx "BUY price=5 qty=10" obd tx dex place-order "$MARKET_ID" BUY 5 10 --from alice
run_tx "BUY price=4 qty=10" obd tx dex place-order "$MARKET_ID" BUY 4 10 --from alice
run_tx "SELL price=5 qty=10" obd tx dex place-order "$MARKET_ID" SELL 5 10 --from bob
run_tx "SELL price=4 qty=10" obd tx dex place-order "$MARKET_ID" SELL 4 10 --from bob
run_tx "BUY price=9 qty=10" obd tx dex place-order "$MARKET_ID" BUY 9 10 --from alice
run_tx "BUY price=8 qty=10" obd tx dex place-order "$MARKET_ID" BUY 8 10 --from alice
run_tx "SELL price=7 qty=10" obd tx dex place-order "$MARKET_ID" SELL 7 10 --from bob
run_tx "SELL price=6 qty=10" obd tx dex place-order "$MARKET_ID" SELL 6 10 --from bob
run_tx "BUY price=7 qty=10" obd tx dex place-order "$MARKET_ID" BUY 7 10 --from alice
run_tx "BUY price=6 qty=10" obd tx dex place-order "$MARKET_ID" BUY 6 10 --from alice
run_tx "SELL price=5 qty=10" obd tx dex place-order "$MARKET_ID" SELL 5 10 --from bob
run_tx "SELL price=4 qty=10" obd tx dex place-order "$MARKET_ID" SELL 4 10 --from bob

query_state
