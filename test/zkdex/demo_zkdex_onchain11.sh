#!/bin/bash

# Demo ONCHAIN-10 + ONCHAIN-11:
# - ONCHAIN-11: query root/nullifier/withdrawRecord/moduleBalance theo denom.
# - ONCHAIN-10: claim withdraw, chuyển tiền từ module account về user, mark claimed=true.

GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

SIGNER="${SIGNER:-alice}"
CHAIN_ID="${CHAIN_ID:-ob}"
NODE="${NODE:-tcp://localhost:26657}"
ASSET_DENOM="${ASSET_DENOM:-USDT}"
DEPOSIT_AMOUNT="${DEPOSIT_AMOUNT:-100}"
WITHDRAW_AMOUNT="${WITHDRAW_AMOUNT:-40}"
TX_FEE="${TX_FEE:-0USDT}"
BINARY="${BINARY:-obd}"

RUN_ID="$(date +%s)"
BATCH_ID="batch-onchain11-${RUN_ID}"
WITHDRAW_ID="wd-onchain11-${RUN_ID}"
NULLIFIER="0xmocknullifierONCHAIN11${RUN_ID}"
DESTINATION_HASH="0xmockdestinationhashONCHAIN11${RUN_ID}"
NEW_STATE_ROOT="0xrootONCHAIN11${RUN_ID}"
PROOF_BUNDLE_FILE="proof_bundle_onchain11_${RUN_ID}.json"

cleanup() {
    rm -f "$PROOF_BUNDLE_FILE"
}
trap cleanup EXIT

extract_json() {
    sed -n '/^{/,$p'
}

fail() {
    echo -e "  ${RED}✗ $1${NC}"
    exit 1
}

assert_eq() {
    local actual="$1"
    local expected="$2"
    local label="$3"
    if [ "$actual" != "$expected" ]; then
        fail "${label}: kỳ vọng '${expected}', nhận '${actual}'"
    fi
    echo -e "  ${GREEN}✓${NC} ${label}: ${GREEN}${actual}${NC}"
}

wait_blocks() {
    local label="$1"
    echo -e "        Chờ 5 giây để ${label} được commit..."
    for i in {5..1}; do
        echo -e "        Đang chờ... $i"
        sleep 1
    done
}

query_json() {
    "$BINARY" q zkdex "$@" --node "$NODE" -o json 2>/dev/null \
        || "$BINARY" q zkdex "$@" -o json 2>/dev/null \
        || "$BINARY" q zkdex "$@" --node "$NODE" 2>/dev/null \
        || "$BINARY" q zkdex "$@" 2>/dev/null
}

query_bank_json() {
    "$BINARY" q bank "$@" --node "$NODE" -o json 2>/dev/null \
        || "$BINARY" q bank "$@" -o json 2>/dev/null \
        || "$BINARY" q bank "$@" --node "$NODE" 2>/dev/null \
        || "$BINARY" q bank "$@" 2>/dev/null
}

require_query_json() {
    local output
    if ! output=$(query_json "$@"); then
        echo -e "  ${RED}✗ Query thất bại:${NC} ${BINARY} q zkdex $*"
        "$BINARY" q zkdex "$@" --node "$NODE" -o json 2>&1 || true
        exit 1
    fi
    printf '%s' "$output"
}

require_bank_json() {
    local output
    if ! output=$(query_bank_json "$@"); then
        echo -e "  ${RED}✗ Bank query thất bại:${NC} ${BINARY} q bank $*"
        "$BINARY" q bank "$@" --node "$NODE" -o json 2>&1 || true
        exit 1
    fi
    printf '%s' "$output"
}

extract_state_root() {
    local payload="$1"
    local root
    root=$(printf '%s' "$payload" | jq -r '.state_root // .stateRoot // empty' 2>/dev/null)
    if [ -n "$root" ] && [ "$root" != "null" ]; then
        printf '%s' "$root"
        return 0
    fi

    printf '%s\n' "$payload" | sed -n 's/^[[:space:]]*state_root:[[:space:]]*//p; s/^[[:space:]]*stateRoot:[[:space:]]*//p' | head -n 1 | tr -d '"'
}

extract_claimed() {
    jq -r '
      if (.record.claimed | type) == "object" then
        .record.claimed.value | tostring
      else
        .record.claimed | tostring
      end
    ' 2>/dev/null
}

coin_amount() {
    local coin="$1"
    local denom="$2"
    local amount

    amount="${coin%$denom}"
    if [ "$amount" = "$coin" ] || [ -z "$amount" ]; then
        printf '0'
        return 0
    fi
    printf '%s' "$amount"
}

bank_balance_amount() {
    local address="$1"
    local denom="$2"
    local payload

    payload=$(require_bank_json balances "$address")
    printf '%s' "$payload" | jq -r --arg denom "$denom" '
      ([.balances[]? | select(.denom == $denom) | .amount][0] // "0")
    '
}

assert_tx_accepted() {
    local txhash="$1"
    local label="$2"
    local result code raw_log

    result=$("$BINARY" query tx "$txhash" --node "$NODE" -o json 2>/dev/null)
    code=$(printf '%s' "$result" | jq -r '.code // .tx_response.code // 0')
    if [ "$code" != "0" ] && [ "$code" != "null" ] && [ -n "$code" ]; then
        raw_log=$(printf '%s' "$result" | jq -r '.raw_log // .tx_response.raw_log // empty')
        echo "$result" | jq 2>/dev/null || echo "$result"
        fail "${label} bị reject trong block. raw_log=${raw_log}"
    fi
    echo -e "  ${GREEN}✓${NC} ${label} accepted: ${GREEN}${txhash}${NC}"
}

submit_tx_json() {
    local output status json txhash code label
    label="$1"
    shift

    output=$("$@" 2>&1)
    status=$?
    json=$(printf '%s\n' "$output" | extract_json)
    txhash=$(printf '%s' "$json" | jq -r '.txhash // empty' 2>/dev/null)
    code=$(printf '%s' "$json" | jq -r '.code // 0' 2>/dev/null)

    if [ "$status" -ne 0 ] || [ -z "$txhash" ] || [ "$txhash" = "null" ] || [ "$code" != "0" ]; then
        echo "$output"
        echo -e "  ${RED}✗ ${label} thất bại trước khi vào block.${NC}"
        return 1
    fi
    printf '%s' "$txhash"
}

echo -e "${CYAN}======================================================================${NC}"
echo -e "${CYAN}  KỊCH BẢN DEMO MVP ZKDEX - TASK ONCHAIN-10 + ONCHAIN-11              ${NC}"
echo -e "${CYAN}======================================================================${NC}"

command -v "$BINARY" >/dev/null 2>&1 || fail "Không tìm thấy binary '${BINARY}'. Có thể đặt BINARY=/path/to/obd."
command -v jq >/dev/null 2>&1 || fail "Script cần jq để kiểm tra JSON."

echo -e "\n${YELLOW}[BƯỚC 1]${NC} Đọc currentStateRoot và ví signer..."
ROOT_JSON=$(require_query_json current-state-root)
OLD_STATE_ROOT=$(extract_state_root "$ROOT_JSON")
if [ -z "$OLD_STATE_ROOT" ] || [ "$OLD_STATE_ROOT" = "null" ]; then
    echo "$ROOT_JSON" | jq 2>/dev/null || echo "$ROOT_JSON"
    fail "Không đọc được currentStateRoot."
fi

SIGNER_ADDR=$("$BINARY" keys show "$SIGNER" -a --keyring-backend test 2>/dev/null)
if [ -z "$SIGNER_ADDR" ]; then
    fail "Không lấy được địa chỉ ví của ${SIGNER}."
fi
echo -e "  ${GREEN}✓${NC} oldStateRoot: ${GREEN}${OLD_STATE_ROOT}${NC}"
echo -e "  ${GREEN}✓${NC} signer: ${GREEN}${SIGNER}${NC} = ${CYAN}${SIGNER_ADDR}${NC}"
echo -e "        newStateRoot: ${CYAN}${NEW_STATE_ROOT}${NC}"
echo -e "        withdrawId: ${CYAN}${WITHDRAW_ID}${NC}"
echo -e "        nullifier: ${CYAN}${NULLIFIER}${NC}"

MODULE_BALANCE_INITIAL_JSON=$(require_query_json module-account-balance "$ASSET_DENOM")
MODULE_BALANCE_INITIAL=$(printf '%s' "$MODULE_BALANCE_INITIAL_JSON" | jq -r '.balance // empty')
MODULE_BALANCE_INITIAL_AMOUNT=$(coin_amount "$MODULE_BALANCE_INITIAL" "$ASSET_DENOM")
echo -e "  ${GREEN}✓${NC} module balance ban đầu (${ASSET_DENOM}): ${GREEN}${MODULE_BALANCE_INITIAL}${NC}"

echo -e "\n${YELLOW}[BƯỚC 2]${NC} Tạo deposit record thật để module có custody balance..."
if ! DEPOSIT_TXHASH=$(submit_tx_json "Deposit" \
  "$BINARY" tx zkdex deposit "$ASSET_DENOM" "$DEPOSIT_AMOUNT" \
  --from "$SIGNER" \
  --chain-id "$CHAIN_ID" \
  --keyring-backend test \
  --node "$NODE" \
  --gas auto \
  --gas-adjustment 1.3 \
  --fees "$TX_FEE" \
  -y -o json); then
    exit 1
fi
echo -e "  ${GREEN}✓${NC} Deposit tx đã gửi: ${GREEN}${DEPOSIT_TXHASH}${NC}"
wait_blocks "deposit record"
assert_tx_accepted "$DEPOSIT_TXHASH" "Deposit tx"

DEPOSIT_RESULT=$("$BINARY" query tx "$DEPOSIT_TXHASH" --node "$NODE" -o json 2>/dev/null)
DEPOSIT_ID=$(printf '%s' "$DEPOSIT_RESULT" | jq -r '
  [
    (.events? // [])[]?,
    (.tx_response.events? // [])[]?,
    .logs[]?.events[]?,
    .tx_response.logs[]?.events[]?
  ]
  | .[]?
  | select(.type == "ob.zkdex.v1.EventDeposit" or .type == "EventDeposit")
  | .attributes[]?
  | select(.key == "deposit_id" or .key == "depositId")
  | .value
' 2>/dev/null | head -n 1 | tr -d '"')

if [ -z "$DEPOSIT_ID" ]; then
    echo "$DEPOSIT_RESULT" | jq 2>/dev/null || echo "$DEPOSIT_RESULT"
    fail "Không đọc được deposit_id từ deposit tx."
fi
echo -e "  ${GREEN}✓${NC} DepositRecord: ${GREEN}${DEPOSIT_ID}${NC}"

echo -e "\n${YELLOW}[BƯỚC 3]${NC} SubmitBatchProof để sinh WithdrawRecord chưa claim..."
cat << EOF > "$PROOF_BUNDLE_FILE"
{
  "proof": "0xmockproof",
  "publicInputs": [
    "${OLD_STATE_ROOT}",
    "${NEW_STATE_ROOT}",
    "0xdepositsRoot",
    "0xwithdrawalsRoot",
    "0xnullifiersRoot",
    "0xwithdrawOutputsRoot"
  ],
  "verificationKeyId": "v1"
}
EOF

SETTLEMENT_UPDATE="{\"batchId\":\"${BATCH_ID}\",\"oldStateRoot\":\"${OLD_STATE_ROOT}\",\"newStateRoot\":\"${NEW_STATE_ROOT}\",\"deposits\":[{\"depositId\":\"${DEPOSIT_ID}\",\"owner\":\"${SIGNER_ADDR}\",\"denom\":\"${ASSET_DENOM}\",\"amount\":\"${DEPOSIT_AMOUNT}\"}],\"withdrawals\":[{\"withdrawId\":\"${WITHDRAW_ID}\",\"owner\":\"${SIGNER_ADDR}\",\"denom\":\"${ASSET_DENOM}\",\"amount\":\"${WITHDRAW_AMOUNT}\",\"destination\":\"${SIGNER_ADDR}\",\"destinationHash\":\"${DESTINATION_HASH}\",\"nullifier\":\"${NULLIFIER}\"}]}"
BATCH_COMMITMENTS='{"depositsRoot":"0xdepositsRoot","withdrawalsRoot":"0xwithdrawalsRoot","nullifiersRoot":"0xnullifiersRoot","withdrawOutputsRoot":"0xwithdrawOutputsRoot"}'

if ! SUBMIT_TXHASH=$(submit_tx_json "SubmitBatchProof" \
  "$BINARY" tx zkdex submit-batch-proof \
  --from "$SIGNER" \
  --chain-id "$CHAIN_ID" \
  --keyring-backend test \
  --node "$NODE" \
  --gas auto \
  --gas-adjustment 1.3 \
  --fees "$TX_FEE" \
  --settlement-update "$SETTLEMENT_UPDATE" \
  --batch-commitments "$BATCH_COMMITMENTS" \
  --proof-bundle "./${PROOF_BUNDLE_FILE}" \
  -y -o json); then
    exit 1
fi
echo -e "  ${GREEN}✓${NC} SubmitBatchProof tx đã gửi: ${GREEN}${SUBMIT_TXHASH}${NC}"
wait_blocks "settlement update"
assert_tx_accepted "$SUBMIT_TXHASH" "SubmitBatchProof tx"

echo -e "\n${YELLOW}[BƯỚC 4]${NC} ONCHAIN-11: query root/nullifier/withdrawRecord/moduleBalance..."
ROOT_AFTER_JSON=$(require_query_json current-state-root)
ROOT_AFTER=$(extract_state_root "$ROOT_AFTER_JSON")
assert_eq "$ROOT_AFTER" "$NEW_STATE_ROOT" "Query current-state-root"

NULLIFIER_JSON=$(require_query_json nullifier-used "$NULLIFIER")
NULLIFIER_USED=$(printf '%s' "$NULLIFIER_JSON" | jq -r '(.used // false) | tostring')
assert_eq "$NULLIFIER_USED" "true" "Query nullifier-used"

WITHDRAW_JSON=$(require_query_json withdraw-record "$WITHDRAW_ID")
WITHDRAW_OWNER=$(printf '%s' "$WITHDRAW_JSON" | jq -r '.record.owner // empty')
WITHDRAW_AMOUNT_ONCHAIN=$(printf '%s' "$WITHDRAW_JSON" | jq -r '.record.amount // empty')
WITHDRAW_DENOM=$(printf '%s' "$WITHDRAW_JSON" | jq -r '.record.denom // empty')
WITHDRAW_DESTINATION=$(printf '%s' "$WITHDRAW_JSON" | jq -r '.record.destination // empty')
WITHDRAW_CLAIMED_BEFORE=$(printf '%s' "$WITHDRAW_JSON" | extract_claimed)
assert_eq "$WITHDRAW_OWNER" "$SIGNER_ADDR" "Query withdraw-record.owner"
assert_eq "$WITHDRAW_AMOUNT_ONCHAIN" "$WITHDRAW_AMOUNT" "Query withdraw-record.amount"
assert_eq "$WITHDRAW_DENOM" "$ASSET_DENOM" "Query withdraw-record.denom"
assert_eq "$WITHDRAW_DESTINATION" "$SIGNER_ADDR" "Query withdraw-record.destination"
assert_eq "$WITHDRAW_CLAIMED_BEFORE" "false" "Query withdraw-record.claimed trước claim"

MODULE_BALANCE_BEFORE_JSON=$(require_query_json module-account-balance "$ASSET_DENOM")
MODULE_BALANCE_BEFORE=$(printf '%s' "$MODULE_BALANCE_BEFORE_JSON" | jq -r '.balance // empty')
EXPECTED_MODULE_BALANCE_BEFORE="$((MODULE_BALANCE_INITIAL_AMOUNT + DEPOSIT_AMOUNT))${ASSET_DENOM}"
assert_eq "$MODULE_BALANCE_BEFORE" "$EXPECTED_MODULE_BALANCE_BEFORE" "Query module-account-balance ${ASSET_DENOM} trước claim"

ALICE_BALANCE_BEFORE_CLAIM=$(bank_balance_amount "$SIGNER_ADDR" "$ASSET_DENOM")
echo -e "  ${GREEN}✓${NC} Bank balance ${SIGNER} trước claim: ${GREEN}${ALICE_BALANCE_BEFORE_CLAIM}${ASSET_DENOM}${NC}"

echo -e "\n${YELLOW}[BƯỚC 5]${NC} ONCHAIN-10: claim withdraw và xác minh claimed=true..."
if ! CLAIM_TXHASH=$(submit_tx_json "ClaimWithdraw" \
  "$BINARY" tx zkdex claim-withdraw "$WITHDRAW_ID" \
  --from "$SIGNER" \
  --chain-id "$CHAIN_ID" \
  --keyring-backend test \
  --node "$NODE" \
  --gas auto \
  --gas-adjustment 1.3 \
  --fees "$TX_FEE" \
  -y -o json); then
    exit 1
fi
echo -e "  ${GREEN}✓${NC} ClaimWithdraw tx đã gửi: ${GREEN}${CLAIM_TXHASH}${NC}"
wait_blocks "withdraw claim"
assert_tx_accepted "$CLAIM_TXHASH" "ClaimWithdraw tx"

CLAIM_TX_RESULT=$("$BINARY" query tx "$CLAIM_TXHASH" --node "$NODE" -o json 2>/dev/null)
CLAIM_EVENT_FOUND=$(printf '%s' "$CLAIM_TX_RESULT" | jq -r '
  [
    (.events? // [])[]?,
    (.tx_response.events? // [])[]?,
    .logs[]?.events[]?,
    .tx_response.logs[]?.events[]?
  ]
  | any(.type == "zkdex_withdraw_claimed")
')
assert_eq "$CLAIM_EVENT_FOUND" "true" "Event zkdex_withdraw_claimed"

WITHDRAW_AFTER_JSON=$(require_query_json withdraw-record "$WITHDRAW_ID")
WITHDRAW_CLAIMED_AFTER=$(printf '%s' "$WITHDRAW_AFTER_JSON" | extract_claimed)
assert_eq "$WITHDRAW_CLAIMED_AFTER" "true" "WithdrawRecord.claimed sau claim"

MODULE_BALANCE_AFTER_JSON=$(require_query_json module-account-balance "$ASSET_DENOM")
MODULE_BALANCE_AFTER=$(printf '%s' "$MODULE_BALANCE_AFTER_JSON" | jq -r '.balance // empty')
EXPECTED_MODULE_BALANCE_AFTER="$((MODULE_BALANCE_INITIAL_AMOUNT + DEPOSIT_AMOUNT - WITHDRAW_AMOUNT))${ASSET_DENOM}"
assert_eq "$MODULE_BALANCE_AFTER" "$EXPECTED_MODULE_BALANCE_AFTER" "Module balance sau claim"

ALICE_BALANCE_AFTER_CLAIM=$(bank_balance_amount "$SIGNER_ADDR" "$ASSET_DENOM")
EXPECTED_ALICE_BALANCE_AFTER="$((ALICE_BALANCE_BEFORE_CLAIM + WITHDRAW_AMOUNT))"
assert_eq "$ALICE_BALANCE_AFTER_CLAIM" "$EXPECTED_ALICE_BALANCE_AFTER" "Bank balance ${SIGNER} tăng đúng ${WITHDRAW_AMOUNT}${ASSET_DENOM} sau claim"

echo -e "\n${YELLOW}[BƯỚC 6]${NC} Negative check: claim lần 2 phải bị reject..."
SECOND_CLAIM_OUTPUT=$("$BINARY" tx zkdex claim-withdraw "$WITHDRAW_ID" \
  --from "$SIGNER" \
  --chain-id "$CHAIN_ID" \
  --keyring-backend test \
  --node "$NODE" \
  --gas auto \
  --gas-adjustment 1.3 \
  --fees "$TX_FEE" \
  -y -o json 2>&1)
SECOND_CLAIM_JSON=$(printf '%s\n' "$SECOND_CLAIM_OUTPUT" | extract_json)
SECOND_CLAIM_TXHASH=$(printf '%s' "$SECOND_CLAIM_JSON" | jq -r '.txhash // empty' 2>/dev/null)

if [ -n "$SECOND_CLAIM_TXHASH" ] && [ "$SECOND_CLAIM_TXHASH" != "null" ]; then
    wait_blocks "second claim rejection"
    SECOND_CLAIM_RESULT=$("$BINARY" query tx "$SECOND_CLAIM_TXHASH" --node "$NODE" -o json 2>/dev/null)
    SECOND_CLAIM_CODE=$(printf '%s' "$SECOND_CLAIM_RESULT" | jq -r '.code // .tx_response.code // 0')
    if [ "$SECOND_CLAIM_CODE" = "0" ] || [ "$SECOND_CLAIM_CODE" = "null" ] || [ -z "$SECOND_CLAIM_CODE" ]; then
        echo "$SECOND_CLAIM_RESULT" | jq 2>/dev/null || echo "$SECOND_CLAIM_RESULT"
        fail "Claim lần 2 đáng ra phải bị reject nhưng tx accepted."
    fi
    echo -e "  ${GREEN}✓${NC} Claim lần 2 bị reject trong block như kỳ vọng."
else
    if printf '%s' "$SECOND_CLAIM_OUTPUT" | grep -qi "already claimed"; then
        echo -e "  ${GREEN}✓${NC} Claim lần 2 bị reject ngay ở CLI/node: already claimed."
    else
        echo "$SECOND_CLAIM_OUTPUT"
        fail "Không xác định được kết quả claim lần 2."
    fi
fi

echo -e "\n${YELLOW}[OUTPUT MẪU]${NC} WithdrawRecord cuối cùng:"
printf '%s' "$WITHDRAW_AFTER_JSON" | jq
echo -e "\n${YELLOW}[OUTPUT MẪU]${NC} Module balance ${ASSET_DENOM} cuối cùng:"
printf '%s' "$MODULE_BALANCE_AFTER_JSON" | jq

cleanup
echo -e "\n${GREEN}======================================================================${NC}"
echo -e "${GREEN}  PASS: ONCHAIN-10 claim withdraw và ONCHAIN-11 state queries hoạt động đúng. ${NC}"
echo -e "${GREEN}======================================================================${NC}"
