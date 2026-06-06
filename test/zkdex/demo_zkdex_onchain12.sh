#!/bin/bash

# Demo ONCHAIN-12 Failure tests:
# - invalid proof/publicInputs mismatch must be rejected
# - duplicate nullifier must be rejected
# - claim withdraw twice must be rejected

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
BATCH_ID_BAD_PROOF="batch-onchain12-badproof-${RUN_ID}"
BATCH_ID_VALID="batch-onchain12-valid-${RUN_ID}"
BATCH_ID_DUP_NULLIFIER="batch-onchain12-dupnull-${RUN_ID}"
WITHDRAW_ID_BAD_PROOF="wd-onchain12-badproof-${RUN_ID}"
WITHDRAW_ID_VALID="wd-onchain12-valid-${RUN_ID}"
WITHDRAW_ID_DUP_NULLIFIER="wd-onchain12-dupnull-${RUN_ID}"
NULLIFIER="0xmocknullifierONCHAIN12${RUN_ID}"
DESTINATION_HASH="0xmockdestinationhashONCHAIN12${RUN_ID}"
ROOT_BAD_PROOF="0xrootONCHAIN12BADPROOF${RUN_ID}"
ROOT_VALID="0xrootONCHAIN12VALID${RUN_ID}"
ROOT_DUP_NULLIFIER="0xrootONCHAIN12DUPNULL${RUN_ID}"
PROOF_BAD_FILE="proof_bundle_onchain12_bad_${RUN_ID}.json"
PROOF_VALID_FILE="proof_bundle_onchain12_valid_${RUN_ID}.json"
PROOF_DUP_FILE="proof_bundle_onchain12_dup_${RUN_ID}.json"

cleanup() {
    rm -f "$PROOF_BAD_FILE" "$PROOF_VALID_FILE" "$PROOF_DUP_FILE"
}
trap cleanup EXIT

extract_json() {
    sed -n '/^{/,$p'
}

fail() {
    echo -e "  ${RED}✗ $1${NC}" >&2
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

bank_balance_amount() {
    local address="$1"
    local denom="$2"
    local payload

    payload=$(require_bank_json balances "$address")
    printf '%s' "$payload" | jq -r --arg denom "$denom" '
      ([.balances[]? | select(.denom == $denom) | .amount][0] // "0")
    '
}

print_alice_balance() {
    local label="$1"
    local amount

    amount=$(bank_balance_amount "$SIGNER_ADDR" "$ASSET_DENOM")
    echo -e "  ${CYAN}→${NC} Balance ${SIGNER} ${label}: ${CYAN}${amount}${ASSET_DENOM}${NC}" >&2
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

submit_expect_rejected() {
    local label expected output status json txhash code result block_code raw_log
    label="$1"
    expected="$2"
    shift 2

    output=$("$@" 2>&1)
    status=$?
    json=$(printf '%s\n' "$output" | extract_json)
    txhash=$(printf '%s' "$json" | jq -r '.txhash // empty' 2>/dev/null)
    code=$(printf '%s' "$json" | jq -r '.code // 0' 2>/dev/null)

    if [ "$status" -ne 0 ] || { [ -n "$code" ] && [ "$code" != "0" ] && [ "$code" != "null" ]; }; then
        if printf '%s' "$output" | grep -qi "$expected"; then
            echo -e "  ${GREEN}✓${NC} ${label} bị reject trước block: ${CYAN}${expected}${NC}"
            return 0
        fi
        echo "$output"
        fail "${label}: tx bị reject trước block nhưng không thấy lỗi kỳ vọng '${expected}'."
    fi

    if [ -z "$txhash" ] || [ "$txhash" = "null" ]; then
        echo "$output"
        fail "${label}: không lấy được txhash hoặc lỗi reject rõ ràng."
    fi

    wait_blocks "$label rejection"
    result=$("$BINARY" query tx "$txhash" --node "$NODE" -o json 2>/dev/null)
    block_code=$(printf '%s' "$result" | jq -r '.code // .tx_response.code // 0')
    raw_log=$(printf '%s' "$result" | jq -r '.raw_log // .tx_response.raw_log // empty')
    if [ "$block_code" = "0" ] || [ "$block_code" = "null" ] || [ -z "$block_code" ]; then
        echo "$result" | jq 2>/dev/null || echo "$result"
        fail "${label}: đáng ra phải bị reject nhưng tx accepted."
    fi
    if ! printf '%s\n%s' "$raw_log" "$result" | grep -qi "$expected"; then
        echo "$result" | jq 2>/dev/null || echo "$result"
        fail "${label}: raw_log không chứa lỗi kỳ vọng '${expected}'."
    fi
    echo -e "  ${GREEN}✓${NC} ${label} bị reject trong block: ${CYAN}${expected}${NC}"
}

write_proof_bundle() {
    local file="$1"
    local old_root="$2"
    local new_root="$3"
    local deposits_root="$4"
    local withdrawals_root="$5"
    local nullifiers_root="$6"
    local withdraw_outputs_root="$7"

    cat << EOF > "$file"
{
  "proof": "0xmockproof",
  "publicInputs": [
    "${old_root}",
    "${new_root}",
    "${deposits_root}",
    "${withdrawals_root}",
    "${nullifiers_root}",
    "${withdraw_outputs_root}"
  ],
  "verificationKeyId": "v1"
}
EOF
}

make_settlement_update() {
    local batch_id="$1"
    local old_root="$2"
    local new_root="$3"
    local deposit_id="$4"
    local withdraw_id="$5"
    local nullifier="$6"

    printf '{"batchId":"%s","oldStateRoot":"%s","newStateRoot":"%s","deposits":[{"depositId":"%s","owner":"%s","denom":"%s","amount":"%s"}],"withdrawals":[{"withdrawId":"%s","owner":"%s","denom":"%s","amount":"%s","destination":"%s","destinationHash":"%s","nullifier":"%s"}]}' \
      "$batch_id" "$old_root" "$new_root" "$deposit_id" "$SIGNER_ADDR" "$ASSET_DENOM" "$DEPOSIT_AMOUNT" \
      "$withdraw_id" "$SIGNER_ADDR" "$ASSET_DENOM" "$WITHDRAW_AMOUNT" "$SIGNER_ADDR" "$DESTINATION_HASH" "$nullifier"
}

create_deposit() {
    local label="$1"
    local output status json txhash code result deposit_id

    print_alice_balance "trước ${label} deposit"
    output=$("$BINARY" tx zkdex deposit "$ASSET_DENOM" "$DEPOSIT_AMOUNT" \
      --from "$SIGNER" \
      --chain-id "$CHAIN_ID" \
      --keyring-backend test \
      --node "$NODE" \
      --gas auto \
      --gas-adjustment 1.3 \
      --fees "$TX_FEE" \
      -y -o json 2>&1)
    status=$?
    json=$(printf '%s\n' "$output" | extract_json)
    txhash=$(printf '%s' "$json" | jq -r '.txhash // empty' 2>/dev/null)
    code=$(printf '%s' "$json" | jq -r '.code // 0' 2>/dev/null)

    if [ "$status" -ne 0 ] || [ -z "$txhash" ] || [ "$txhash" = "null" ] || [ "$code" != "0" ]; then
        echo "$output"
        fail "${label}: không tạo được deposit record."
    fi
    echo -e "  ${GREEN}✓${NC} ${label} deposit tx đã gửi: ${GREEN}${txhash}${NC}" >&2
    wait_blocks "${label} deposit record" >&2
    assert_tx_accepted "$txhash" "${label} Deposit tx" >&2
    print_alice_balance "sau ${label} deposit"

    result=$("$BINARY" query tx "$txhash" --node "$NODE" -o json 2>/dev/null)
    deposit_id=$(printf '%s' "$result" | jq -r '
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

    if [ -z "$deposit_id" ]; then
        echo "$result" | jq 2>/dev/null || echo "$result"
        fail "${label}: không đọc được deposit_id từ deposit tx."
    fi
    echo -e "  ${GREEN}✓${NC} ${label} DepositRecord: ${GREEN}${deposit_id}${NC}" >&2
    printf '%s' "$deposit_id"
}

BATCH_COMMITMENTS='{"depositsRoot":"0xdepositsRoot","withdrawalsRoot":"0xwithdrawalsRoot","nullifiersRoot":"0xnullifiersRoot","withdrawOutputsRoot":"0xwithdrawOutputsRoot"}'

echo -e "${CYAN}======================================================================${NC}"
echo -e "${CYAN}  KỊCH BẢN DEMO MVP ZKDEX - TASK ONCHAIN-12 FAILURE TESTS             ${NC}"
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
echo -e "        shared nullifier: ${CYAN}${NULLIFIER}${NC}"
print_alice_balance "ban đầu"

echo -e "\n${YELLOW}[BƯỚC 2]${NC} Tạo deposit cho case invalid proof..."
if ! DEPOSIT_BAD_PROOF_ID=$(create_deposit "Bad-proof"); then
    exit 1
fi
SETTLEMENT_BAD_PROOF=$(make_settlement_update "$BATCH_ID_BAD_PROOF" "$OLD_STATE_ROOT" "$ROOT_BAD_PROOF" "$DEPOSIT_BAD_PROOF_ID" "$WITHDRAW_ID_BAD_PROOF" "${NULLIFIER}-bad")
write_proof_bundle "$PROOF_BAD_FILE" "$OLD_STATE_ROOT" "0xtamperedONCHAIN12${RUN_ID}" "0xdepositsRoot" "0xwithdrawalsRoot" "0xnullifiersRoot" "0xwithdrawOutputsRoot"

echo -e "\n${YELLOW}[BƯỚC 3]${NC} Failure vector 1: invalid proof/publicInputs mismatch phải bị reject..."
submit_expect_rejected "Invalid proof bundle" "publicInputs do not match" \
  "$BINARY" tx zkdex submit-batch-proof \
  --from "$SIGNER" \
  --chain-id "$CHAIN_ID" \
  --keyring-backend test \
  --node "$NODE" \
  --gas auto \
  --gas-adjustment 1.3 \
  --fees "$TX_FEE" \
  --settlement-update "$SETTLEMENT_BAD_PROOF" \
  --batch-commitments "$BATCH_COMMITMENTS" \
  --proof-bundle "./${PROOF_BAD_FILE}" \
  -y -o json

ROOT_AFTER_BAD_JSON=$(require_query_json current-state-root)
ROOT_AFTER_BAD=$(extract_state_root "$ROOT_AFTER_BAD_JSON")
assert_eq "$ROOT_AFTER_BAD" "$OLD_STATE_ROOT" "State root không đổi sau invalid proof"

# kiểm tra DepositRecord bad-proof vẫn chưa processed
BAD_DEPOSIT_PROCESSED_JSON=$(require_query_json deposit-processed "$DEPOSIT_BAD_PROOF_ID")
BAD_DEPOSIT_PROCESSED_INDEX=$(printf '%s' "$BAD_DEPOSIT_PROCESSED_JSON" | jq -r '(.processed // false) | tostring')
assert_eq "$BAD_DEPOSIT_PROCESSED_INDEX" "false" "DepositProcessed bad-proof vẫn false"

# kiểm tra không có WithdrawRecord bad-proof
if query_json withdraw-record "$WITHDRAW_ID_BAD_PROOF" >/dev/null 2>&1; then
    fail "Invalid proof không được tạo WithdrawRecord bad-proof."
else
    echo -e "  ${GREEN}✓${NC} Invalid proof không tạo WithdrawRecord bad-proof"
fi



echo -e "\n${YELLOW}[BƯỚC 4]${NC} Submit batch hợp lệ để tạo nullifier và WithdrawRecord..."
if ! DEPOSIT_VALID_ID=$(create_deposit "Valid"); then
    exit 1
fi
SETTLEMENT_VALID=$(make_settlement_update "$BATCH_ID_VALID" "$OLD_STATE_ROOT" "$ROOT_VALID" "$DEPOSIT_VALID_ID" "$WITHDRAW_ID_VALID" "$NULLIFIER")
write_proof_bundle "$PROOF_VALID_FILE" "$OLD_STATE_ROOT" "$ROOT_VALID" "0xdepositsRoot" "0xwithdrawalsRoot" "0xnullifiersRoot" "0xwithdrawOutputsRoot"

if ! VALID_TXHASH=$(submit_tx_json "Valid SubmitBatchProof" \
  "$BINARY" tx zkdex submit-batch-proof \
  --from "$SIGNER" \
  --chain-id "$CHAIN_ID" \
  --keyring-backend test \
  --node "$NODE" \
  --gas auto \
  --gas-adjustment 1.3 \
  --fees "$TX_FEE" \
  --settlement-update "$SETTLEMENT_VALID" \
  --batch-commitments "$BATCH_COMMITMENTS" \
  --proof-bundle "./${PROOF_VALID_FILE}" \
  -y -o json); then
    exit 1
fi
echo -e "  ${GREEN}✓${NC} Valid SubmitBatchProof tx đã gửi: ${GREEN}${VALID_TXHASH}${NC}"
wait_blocks "valid settlement"
assert_tx_accepted "$VALID_TXHASH" "Valid SubmitBatchProof tx"

ROOT_AFTER_VALID_JSON=$(require_query_json current-state-root)
ROOT_AFTER_VALID=$(extract_state_root "$ROOT_AFTER_VALID_JSON")
assert_eq "$ROOT_AFTER_VALID" "$ROOT_VALID" "State root sau valid batch"

NULLIFIER_JSON=$(require_query_json nullifier-used "$NULLIFIER")
NULLIFIER_USED=$(printf '%s' "$NULLIFIER_JSON" | jq -r '(.used // false) | tostring')
assert_eq "$NULLIFIER_USED" "true" "Nullifier đã được mark used"

WITHDRAW_JSON=$(require_query_json withdraw-record "$WITHDRAW_ID_VALID")
WITHDRAW_CLAIMED_BEFORE=$(printf '%s' "$WITHDRAW_JSON" | extract_claimed)
assert_eq "$WITHDRAW_CLAIMED_BEFORE" "false" "WithdrawRecord chưa claimed"

echo -e "\n${YELLOW}[BƯỚC 5]${NC} Failure vector 2: duplicate nullifier phải bị reject..."
if ! DEPOSIT_DUP_ID=$(create_deposit "Duplicate-nullifier"); then
    exit 1
fi

# Xài lại nullifier cũ
SETTLEMENT_DUP=$(make_settlement_update "$BATCH_ID_DUP_NULLIFIER" "$ROOT_VALID" "$ROOT_DUP_NULLIFIER" "$DEPOSIT_DUP_ID" "$WITHDRAW_ID_DUP_NULLIFIER" "$NULLIFIER")
write_proof_bundle "$PROOF_DUP_FILE" "$ROOT_VALID" "$ROOT_DUP_NULLIFIER" "0xdepositsRoot" "0xwithdrawalsRoot" "0xnullifiersRoot" "0xwithdrawOutputsRoot"

submit_expect_rejected "Duplicate nullifier" "already used" \
  "$BINARY" tx zkdex submit-batch-proof \
  --from "$SIGNER" \
  --chain-id "$CHAIN_ID" \
  --keyring-backend test \
  --node "$NODE" \
  --gas auto \
  --gas-adjustment 1.3 \
  --fees "$TX_FEE" \
  --settlement-update "$SETTLEMENT_DUP" \
  --batch-commitments "$BATCH_COMMITMENTS" \
  --proof-bundle "./${PROOF_DUP_FILE}" \
  -y -o json

# Đảm bảo state root không đổi sau khi reject duplicate nullifier
ROOT_AFTER_DUP_JSON=$(require_query_json current-state-root)
ROOT_AFTER_DUP=$(extract_state_root "$ROOT_AFTER_DUP_JSON")
assert_eq "$ROOT_AFTER_DUP" "$ROOT_VALID" "State root không đổi sau duplicate nullifier"


echo -e "\n${YELLOW}[BƯỚC 6]${NC} Claim withdraw lần đầu phải thành công..."
ALICE_BALANCE_BEFORE_CLAIM=$(bank_balance_amount "$SIGNER_ADDR" "$ASSET_DENOM")
echo -e "  ${CYAN}→${NC} Balance ${SIGNER} trước withdraw claim: ${CYAN}${ALICE_BALANCE_BEFORE_CLAIM}${ASSET_DENOM}${NC}"
if ! CLAIM_TXHASH=$(submit_tx_json "ClaimWithdraw" \
  "$BINARY" tx zkdex claim-withdraw "$WITHDRAW_ID_VALID" \
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

WITHDRAW_AFTER_CLAIM_JSON=$(require_query_json withdraw-record "$WITHDRAW_ID_VALID")
WITHDRAW_CLAIMED_AFTER=$(printf '%s' "$WITHDRAW_AFTER_CLAIM_JSON" | extract_claimed)
assert_eq "$WITHDRAW_CLAIMED_AFTER" "true" "WithdrawRecord.claimed sau claim"

ALICE_BALANCE_AFTER_CLAIM=$(bank_balance_amount "$SIGNER_ADDR" "$ASSET_DENOM")
echo -e "  ${CYAN}→${NC} Balance ${SIGNER} sau withdraw claim: ${CYAN}${ALICE_BALANCE_AFTER_CLAIM}${ASSET_DENOM}${NC}"
EXPECTED_ALICE_BALANCE_AFTER="$((ALICE_BALANCE_BEFORE_CLAIM + WITHDRAW_AMOUNT))"
assert_eq "$ALICE_BALANCE_AFTER_CLAIM" "$EXPECTED_ALICE_BALANCE_AFTER" "Bank balance ${SIGNER} tăng đúng ${WITHDRAW_AMOUNT}${ASSET_DENOM}"

echo -e "\n${YELLOW}[BƯỚC 7]${NC} Failure vector 3: claim withdraw lần 2 phải bị reject..."
ALICE_BALANCE_BEFORE_SECOND_CLAIM=$(bank_balance_amount "$SIGNER_ADDR" "$ASSET_DENOM")
echo -e "  ${CYAN}→${NC} Balance ${SIGNER} trước claim lần 2: ${CYAN}${ALICE_BALANCE_BEFORE_SECOND_CLAIM}${ASSET_DENOM}${NC}"
submit_expect_rejected "Claim twice" "already claimed" \
  "$BINARY" tx zkdex claim-withdraw "$WITHDRAW_ID_VALID" \
  --from "$SIGNER" \
  --chain-id "$CHAIN_ID" \
  --keyring-backend test \
  --node "$NODE" \
  --gas auto \
  --gas-adjustment 1.3 \
  --fees "$TX_FEE" \
  -y -o json

WITHDRAW_FINAL_JSON=$(require_query_json withdraw-record "$WITHDRAW_ID_VALID")
WITHDRAW_FINAL_CLAIMED=$(printf '%s' "$WITHDRAW_FINAL_JSON" | extract_claimed)
assert_eq "$WITHDRAW_FINAL_CLAIMED" "true" "WithdrawRecord vẫn claimed=true sau claim twice rejection"
ALICE_BALANCE_AFTER_SECOND_CLAIM=$(bank_balance_amount "$SIGNER_ADDR" "$ASSET_DENOM")
echo -e "  ${CYAN}→${NC} Balance ${SIGNER} sau claim lần 2 bị reject: ${CYAN}${ALICE_BALANCE_AFTER_SECOND_CLAIM}${ASSET_DENOM}${NC}"
assert_eq "$ALICE_BALANCE_AFTER_SECOND_CLAIM" "$ALICE_BALANCE_BEFORE_SECOND_CLAIM" "Bank balance không đổi sau claim lần 2 bị reject"

echo -e "\n${YELLOW}[OUTPUT MẪU]${NC} WithdrawRecord cuối cùng:"
printf '%s' "$WITHDRAW_FINAL_JSON" | jq
echo -e "\n${YELLOW}[OUTPUT MẪU]${NC} NullifierUsed:"
printf '%s' "$NULLIFIER_JSON" | jq

cleanup
echo -e "\n${GREEN}======================================================================${NC}"
echo -e "${GREEN}  PASS: ONCHAIN-12 invalid proof, duplicate nullifier, claim twice đều bị reject. ${NC}"
echo -e "${GREEN}======================================================================${NC}"
