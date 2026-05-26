#!/bin/bash

# Demo ONCHAIN-09: sau khi SubmitBatchProof được accept, settlement update phải
# thật sự được apply vào on-chain state.

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
BATCH_ID="batch-onchain09-${RUN_ID}"
WITHDRAW_ID="wd-onchain09-${RUN_ID}"
NULLIFIER="0xmocknullifier${RUN_ID}"
DESTINATION_HASH="0xmockdestinationhash${RUN_ID}"
NEW_STATE_ROOT="0xrootONCHAIN09${RUN_ID}"
PROOF_BUNDLE_FILE="proof_bundle_onchain09_${RUN_ID}.json"

cleanup() {
    rm -f "$PROOF_BUNDLE_FILE"
}
trap cleanup EXIT

extract_json() {
    sed -n '/^{/,$p'
}

wait_blocks() {
    local label="$1"
    echo -e "        Chờ 5 giây để ${label} được commit..."
    for i in {5..1}; do
        echo -e "        Đang chờ... $i"
        sleep 1
    done
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

query_json() {
    "$BINARY" q zkdex "$@" --node "$NODE" -o json 2>/dev/null \
        || "$BINARY" q zkdex "$@" -o json 2>/dev/null \
        || "$BINARY" q zkdex "$@" --node "$NODE" 2>/dev/null \
        || "$BINARY" q zkdex "$@" 2>/dev/null
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

echo -e "${CYAN}======================================================================${NC}"
echo -e "${CYAN}    KỊCH BẢN DEMO MVP ZKDEX (PROJECT: GANC-TRADE) - TASK ONCHAIN-09   ${NC}"
echo -e "${CYAN}======================================================================${NC}"

command -v "$BINARY" >/dev/null 2>&1 || fail "Không tìm thấy binary '${BINARY}'. Có thể đặt BINARY=/path/to/obd."
command -v jq >/dev/null 2>&1 || fail "Script cần jq để kiểm tra JSON."

echo -e "\n${YELLOW}[BƯỚC 1]${NC} Đọc currentStateRoot hiện tại để dùng làm oldStateRoot..."
ROOT_JSON=$(require_query_json current-state-root)
OLD_STATE_ROOT=$(extract_state_root "$ROOT_JSON")
if [ -z "$OLD_STATE_ROOT" ] || [ "$OLD_STATE_ROOT" = "null" ]; then
    echo "$ROOT_JSON" | jq 2>/dev/null || echo "$ROOT_JSON"
    fail "Không đọc được currentStateRoot. Thử chạy thủ công: ${BINARY} q zkdex current-state-root"
fi
echo -e "  ${GREEN}✓${NC} oldStateRoot hiện tại: ${GREEN}${OLD_STATE_ROOT}${NC}"
echo -e "        newStateRoot sẽ apply: ${CYAN}${NEW_STATE_ROOT}${NC}"
echo -e "        batchId: ${CYAN}${BATCH_ID}${NC}"
echo -e "        withdrawId: ${CYAN}${WITHDRAW_ID}${NC}"
echo -e "        nullifier: ${CYAN}${NULLIFIER}${NC}"

echo -e "\n${YELLOW}[BƯỚC 2]${NC} Tạo deposit record thật để batch proof tham chiếu đúng depositId..."
SIGNER_ADDR=$("$BINARY" keys show "$SIGNER" -a --keyring-backend test 2>/dev/null)
if [ -z "$SIGNER_ADDR" ]; then
    fail "Không lấy được địa chỉ ví của ${SIGNER}."
fi
echo -e "        Địa chỉ ${SIGNER}: ${CYAN}${SIGNER_ADDR}${NC}"

DEPOSIT_OUTPUT=$("$BINARY" tx zkdex deposit "$ASSET_DENOM" "$DEPOSIT_AMOUNT" \
  --from "$SIGNER" \
  --chain-id "$CHAIN_ID" \
  --keyring-backend test \
  --node "$NODE" \
  --gas auto \
  --gas-adjustment 1.3 \
  --fees "$TX_FEE" \
  -y -o json 2>&1)
DEPOSIT_STATUS=$?
DEPOSIT_JSON=$(printf '%s\n' "$DEPOSIT_OUTPUT" | extract_json)
DEPOSIT_TXHASH=$(printf '%s' "$DEPOSIT_JSON" | jq -r '.txhash // empty' 2>/dev/null)
DEPOSIT_CODE=$(printf '%s' "$DEPOSIT_JSON" | jq -r '.code // 0' 2>/dev/null)

if [ "$DEPOSIT_STATUS" -ne 0 ] || [ -z "$DEPOSIT_TXHASH" ] || [ "$DEPOSIT_CODE" != "0" ]; then
    echo "$DEPOSIT_OUTPUT"
    fail "Không tạo được deposit record."
fi
echo -e "  ${GREEN}✓${NC} Deposit tx đã gửi: ${GREEN}${DEPOSIT_TXHASH}${NC}"
wait_blocks "deposit record"

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
echo -e "  ${GREEN}✓${NC} Deposit record đã tạo: ${GREEN}${DEPOSIT_ID}${NC}"
echo -e "        DepositRecord hiện tại:"
require_query_json deposit-record "$DEPOSIT_ID" | jq

echo -e "\n${YELLOW}[BƯỚC 3]${NC} Tạo proof bundle có publicInputs khớp old/new root và commitments..."
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
echo -e "  ${GREEN}✓${NC} Đã tạo ${CYAN}${PROOF_BUNDLE_FILE}${NC}"

echo -e "\n${YELLOW}[BƯỚC 4]${NC} Snapshot TRƯỚC khi apply settlement update..."
ROOT_BEFORE_JSON=$(require_query_json current-state-root)
ROOT_BEFORE=$(extract_state_root "$ROOT_BEFORE_JSON")
assert_eq "$ROOT_BEFORE" "$OLD_STATE_ROOT" "currentStateRoot trước apply"

DEPOSIT_RECORD_BEFORE_JSON=$(require_query_json deposit-record "$DEPOSIT_ID")
DEPOSIT_PROCESSED_BEFORE_IN_RECORD=$(printf '%s' "$DEPOSIT_RECORD_BEFORE_JSON" | jq -r '(.record.processed // .deposit_record.processed // .depositRecord.processed // false) | tostring')
assert_eq "$DEPOSIT_PROCESSED_BEFORE_IN_RECORD" "false" "DepositRecord.processed trước apply"

DEPOSIT_PROCESSED_BEFORE_JSON=$(require_query_json deposit-processed "$DEPOSIT_ID")
DEPOSIT_PROCESSED_BEFORE=$(printf '%s' "$DEPOSIT_PROCESSED_BEFORE_JSON" | jq -r '(.processed // false) | tostring')
assert_eq "$DEPOSIT_PROCESSED_BEFORE" "false" "DepositProcessed index trước apply"

NULLIFIER_BEFORE_JSON=$(require_query_json nullifier-used "$NULLIFIER")
NULLIFIER_USED_BEFORE=$(printf '%s' "$NULLIFIER_BEFORE_JSON" | jq -r '(.used // false) | tostring')
assert_eq "$NULLIFIER_USED_BEFORE" "false" "NullifierUsed trước apply"

if WITHDRAW_BEFORE_JSON=$(query_json withdraw-record "$WITHDRAW_ID"); then
    echo "$WITHDRAW_BEFORE_JSON" | jq 2>/dev/null || echo "$WITHDRAW_BEFORE_JSON"
    fail "WithdrawRecord ${WITHDRAW_ID} đã tồn tại trước apply."
fi
echo -e "  ${GREEN}✓${NC} WithdrawRecord trước apply: ${GREEN}chưa tồn tại${NC}"

if BATCH_BEFORE_JSON=$(query_json batch-record "$BATCH_ID"); then
    echo "$BATCH_BEFORE_JSON" | jq 2>/dev/null || echo "$BATCH_BEFORE_JSON"
    fail "BatchRecord ${BATCH_ID} đã tồn tại trước apply."
fi
echo -e "  ${GREEN}✓${NC} BatchRecord trước apply: ${GREEN}chưa tồn tại${NC}"

echo -e "\n${YELLOW}[BƯỚC 5]${NC} SubmitBatchProof; ONCHAIN-09 sẽ apply state nếu proof được accept..."
TX_OUTPUT=$("$BINARY" tx zkdex submit-batch-proof \
  --from "$SIGNER" \
  --chain-id "$CHAIN_ID" \
  --keyring-backend test \
  --node "$NODE" \
  --gas auto \
  --gas-adjustment 1.3 \
  --fees "$TX_FEE" \
  --settlement-update "{\"batchId\":\"${BATCH_ID}\",\"oldStateRoot\":\"${OLD_STATE_ROOT}\",\"newStateRoot\":\"${NEW_STATE_ROOT}\",\"deposits\":[{\"depositId\":\"${DEPOSIT_ID}\",\"owner\":\"${SIGNER_ADDR}\",\"denom\":\"${ASSET_DENOM}\",\"amount\":\"${DEPOSIT_AMOUNT}\"}],\"withdrawals\":[{\"withdrawId\":\"${WITHDRAW_ID}\",\"owner\":\"${SIGNER_ADDR}\",\"denom\":\"${ASSET_DENOM}\",\"amount\":\"${WITHDRAW_AMOUNT}\",\"destination\":\"${SIGNER_ADDR}\",\"destinationHash\":\"${DESTINATION_HASH}\",\"nullifier\":\"${NULLIFIER}\"}]}" \
  --batch-commitments '{"depositsRoot":"0xdepositsRoot","withdrawalsRoot":"0xwithdrawalsRoot","nullifiersRoot":"0xnullifiersRoot","withdrawOutputsRoot":"0xwithdrawOutputsRoot"}' \
  --proof-bundle "./${PROOF_BUNDLE_FILE}" \
  -y -o json 2>&1)
TX_STATUS=$?
TX_JSON=$(printf '%s\n' "$TX_OUTPUT" | extract_json)
TXHASH=$(printf '%s' "$TX_JSON" | jq -r '.txhash // empty' 2>/dev/null)
TX_CODE=$(printf '%s' "$TX_JSON" | jq -r '.code // 0' 2>/dev/null)

if [ "$TX_STATUS" -ne 0 ] || [ -z "$TXHASH" ] || [ "$TXHASH" = "null" ] || [ "$TX_CODE" != "0" ]; then
    echo "$TX_OUTPUT"
    fail "SubmitBatchProof thất bại trước khi vào block."
fi
echo -e "  ${GREEN}✓${NC} SubmitBatchProof tx đã gửi: ${GREEN}${TXHASH}${NC}"
wait_blocks "settlement update"

echo -e "\n${YELLOW}[BƯỚC 6]${NC} Kiểm tra tx đã accepted và có event apply settlement..."
TX_RESULT=$("$BINARY" query tx "$TXHASH" --node "$NODE" -o json 2>/dev/null)
CODE=$(printf '%s' "$TX_RESULT" | jq -r '.code // .tx_response.code // 0')
if [ "$CODE" != "0" ] && [ "$CODE" != "null" ] && [ -n "$CODE" ]; then
    RAW_LOG=$(printf '%s' "$TX_RESULT" | jq -r '.raw_log // .tx_response.raw_log // empty')
    echo "$TX_RESULT" | jq 2>/dev/null || echo "$TX_RESULT"
    fail "Tx bị reject trong block. raw_log=${RAW_LOG}"
fi
EVENT_FOUND=$(printf '%s' "$TX_RESULT" | jq -r '
  [
    (.events? // [])[]?,
    (.tx_response.events? // [])[]?,
    .logs[]?.events[]?,
    .tx_response.logs[]?.events[]?
  ]
  | any(.type == "zkdex_batch_settlement_applied")
')
assert_eq "$EVENT_FOUND" "true" "Event zkdex_batch_settlement_applied đã được emit"

echo -e "\n${YELLOW}[BƯỚC 7]${NC} Snapshot SAU khi apply: xác minh ONCHAIN-09 đã ghi đủ compact state..."

ROOT_AFTER_JSON=$(require_query_json current-state-root)
ROOT_AFTER=$(extract_state_root "$ROOT_AFTER_JSON")
assert_eq "$ROOT_AFTER" "$NEW_STATE_ROOT" "currentStateRoot đã chuyển sang newStateRoot"

DEPOSIT_RECORD_JSON=$(require_query_json deposit-record "$DEPOSIT_ID")
DEPOSIT_PROCESSED_IN_RECORD=$(printf '%s' "$DEPOSIT_RECORD_JSON" | jq -r '(.record.processed // .deposit_record.processed // .depositRecord.processed // false) | tostring')
assert_eq "$DEPOSIT_PROCESSED_IN_RECORD" "true" "DepositRecord.processed"

DEPOSIT_PROCESSED_JSON=$(require_query_json deposit-processed "$DEPOSIT_ID")
DEPOSIT_PROCESSED=$(printf '%s' "$DEPOSIT_PROCESSED_JSON" | jq -r '(.processed // false) | tostring')
assert_eq "$DEPOSIT_PROCESSED" "true" "DepositProcessed index"

NULLIFIER_JSON=$(require_query_json nullifier-used "$NULLIFIER")
NULLIFIER_USED=$(printf '%s' "$NULLIFIER_JSON" | jq -r '(.used // false) | tostring')
assert_eq "$NULLIFIER_USED" "true" "NullifierUsed"

WITHDRAW_JSON=$(require_query_json withdraw-record "$WITHDRAW_ID")
WITHDRAW_OWNER=$(printf '%s' "$WITHDRAW_JSON" | jq -r '.record.owner // empty')
WITHDRAW_AMOUNT_ONCHAIN=$(printf '%s' "$WITHDRAW_JSON" | jq -r '.record.amount // empty')
WITHDRAW_DENOM=$(printf '%s' "$WITHDRAW_JSON" | jq -r '.record.denom // empty')
WITHDRAW_DESTINATION=$(printf '%s' "$WITHDRAW_JSON" | jq -r '.record.destination // empty')
WITHDRAW_NULLIFIER=$(printf '%s' "$WITHDRAW_JSON" | jq -r '.record.nullifier // empty')
WITHDRAW_CLAIMED_PRESENT=$(printf '%s' "$WITHDRAW_JSON" | jq -r '.record | has("claimed")' 2>/dev/null)
WITHDRAW_CLAIMED=$(printf '%s' "$WITHDRAW_JSON" | jq -r '
  if (.record.claimed | type) == "object" then
    .record.claimed.value | tostring
  else
    .record.claimed | tostring
  end
' 2>/dev/null)
assert_eq "$WITHDRAW_OWNER" "$SIGNER_ADDR" "WithdrawRecord.owner"
assert_eq "$WITHDRAW_AMOUNT_ONCHAIN" "$WITHDRAW_AMOUNT" "WithdrawRecord.amount"
assert_eq "$WITHDRAW_DENOM" "$ASSET_DENOM" "WithdrawRecord.denom"
assert_eq "$WITHDRAW_DESTINATION" "$SIGNER_ADDR" "WithdrawRecord.destination"
assert_eq "$WITHDRAW_NULLIFIER" "$NULLIFIER" "WithdrawRecord.nullifier"
if [ "$WITHDRAW_CLAIMED_PRESENT" != "true" ]; then
    echo "$WITHDRAW_JSON" | jq 2>/dev/null || echo "$WITHDRAW_JSON"
    fail "WithdrawRecord.claimed không xuất hiện trong JSON query. Hãy rebuild/install lại ${BINARY} sau khi proto bỏ omitempty cho field claimed."
fi
assert_eq "$WITHDRAW_CLAIMED" "false" "WithdrawRecord.claimed"

BATCH_JSON=$(require_query_json batch-record "$BATCH_ID")
BATCH_OLD_ROOT=$(printf '%s' "$BATCH_JSON" | jq -r '.record.old_state_root // .record.oldStateRoot // empty')
BATCH_NEW_ROOT=$(printf '%s' "$BATCH_JSON" | jq -r '.record.new_state_root // .record.newStateRoot // empty')
BATCH_DEPOSIT_ID=$(printf '%s' "$BATCH_JSON" | jq -r '.record.deposit_ids[0] // .record.depositIds[0] // empty')
BATCH_WITHDRAW_ID=$(printf '%s' "$BATCH_JSON" | jq -r '.record.withdraw_ids[0] // .record.withdrawIds[0] // empty')
assert_eq "$BATCH_OLD_ROOT" "$OLD_STATE_ROOT" "BatchRecord.oldStateRoot"
assert_eq "$BATCH_NEW_ROOT" "$NEW_STATE_ROOT" "BatchRecord.newStateRoot"
assert_eq "$BATCH_DEPOSIT_ID" "$DEPOSIT_ID" "BatchRecord.depositIds[0]"
assert_eq "$BATCH_WITHDRAW_ID" "$WITHDRAW_ID" "BatchRecord.withdrawIds[0]"

# Dọn dẹp tài nguyên tạm như demo ONCHAIN-08.
cleanup
echo -e "\n${GREEN}======================================================================${NC}"
echo -e "${GREEN}  ONCHAIN-09 PASS: valid update + proof đã được apply vào on-chain state. ${NC}"
echo -e "${GREEN}======================================================================${NC}"
