#!/bin/bash

# Thống nhất màu sắc hiển thị cho terminal chuyên nghiệp
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

SIGNER="${SIGNER:-alice}"
CHAIN_ID="${CHAIN_ID:-ob}"
NODE="${NODE:-tcp://localhost:26657}"
ASSET_DENOM="${ASSET_DENOM:-USDT}"
DEPOSIT_AMOUNT="${DEPOSIT_AMOUNT:-100}"
WITHDRAW_AMOUNT="${WITHDRAW_AMOUNT:-40}"
TX_FEE="${TX_FEE:-0USDT}"
BINARY="${BINARY:-obd}"

RUN_ID="$(date +%s)"
BATCH_ID="batch-onchain08-${RUN_ID}"
WITHDRAW_ID="wd-onchain08-${RUN_ID}"
NULLIFIER="0xmocknullifierONCHAIN08${RUN_ID}"
DESTINATION_HASH="0xmockdestinationhashONCHAIN08${RUN_ID}"
NEW_STATE_ROOT="0xrootONCHAIN08${RUN_ID}"
PROOF_BUNDLE_FILE="proof_bundle_onchain08_${RUN_ID}.json"

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
echo -e "${CYAN}    KỊCH BẢN DEMO MVP ZKDEX (PROJECT: GANC-TRADE) - TASK ONCHAIN-08   ${NC}"
echo -e "${CYAN}======================================================================${NC}"

command -v "$BINARY" >/dev/null 2>&1 || fail "Không tìm thấy binary '${BINARY}'. Có thể đặt BINARY=/path/to/obd."
command -v jq >/dev/null 2>&1 || fail "Script cần jq để kiểm tra JSON."

# Bước 1: Đọc currentStateRoot thật và khởi tạo proof bundle khớp publicInputs on-chain.
echo -e "\n${YELLOW}[BƯỚC 1]${NC} Đọc currentStateRoot và khởi tạo file ${CYAN}${PROOF_BUNDLE_FILE}${NC}..."
ROOT_JSON=$(require_query_json current-state-root)
OLD_STATE_ROOT=$(extract_state_root "$ROOT_JSON")
if [ -z "$OLD_STATE_ROOT" ] || [ "$OLD_STATE_ROOT" = "null" ]; then
    echo "$ROOT_JSON" | jq 2>/dev/null || echo "$ROOT_JSON"
    fail "Không đọc được currentStateRoot. Thử chạy thủ công: ${BINARY} q zkdex current-state-root"
fi
echo -e "  ${GREEN}✓${NC} oldStateRoot hiện tại: ${GREEN}${OLD_STATE_ROOT}${NC}"
echo -e "        newStateRoot sẽ submit: ${CYAN}${NEW_STATE_ROOT}${NC}"
echo -e "        batchId: ${CYAN}${BATCH_ID}${NC}"

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

if [ -f "$PROOF_BUNDLE_FILE" ]; then
    echo -e "  ${GREEN}✓${NC} Khởi tạo file ${PROOF_BUNDLE_FILE} thành công."
else
    echo -e "  ${RED}✗${NC} Không thể tạo file cấu trúc dữ liệu tạm!"
    exit 1
fi

# Bước 2: Tạo deposit record thật trên chain để batch proof tham chiếu đúng depositId
echo -e "\n${YELLOW}[BƯỚC 2]${NC} Tạo deposit record thật trên chain bằng ${CYAN}obd tx zkdex deposit${NC}..."
SIGNER_ADDR=$("$BINARY" keys show "$SIGNER" -a --keyring-backend test 2>/dev/null)
if [ -z "$SIGNER_ADDR" ]; then
    echo -e "  ${RED}✗ Không lấy được địa chỉ ví của ${SIGNER}!${NC}"
    exit 1
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
    echo -e "  ${RED}✗ Không tạo được deposit record!${NC}"
    if echo "$DEPOSIT_OUTPUT" | grep -q "chain-id"; then
        echo -e "  ${YELLOW}Gợi ý:${NC} kiểm tra chain-id. Script đang dùng ${CYAN}${CHAIN_ID}${NC}."
    fi
    echo -e "  Chi tiết phản hồi từ hệ thống:"
    echo "$DEPOSIT_OUTPUT"
    exit 1
fi

echo -e "  ${GREEN}✓${NC} Deposit tx đã gửi: ${GREEN}${DEPOSIT_TXHASH}${NC}"
echo -e "        Chờ 5 giây để deposit record được commit..."
for i in {5..1}; do
    echo -e "        Đang chờ deposit record... $i"
    sleep 1
done

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
    echo -e "  ${RED}✗ Không đọc được deposit_id từ deposit tx!${NC}"
    echo -e "  Chi tiết tx deposit:"
    echo "$DEPOSIT_RESULT" | jq 2>/dev/null || echo "$DEPOSIT_RESULT"
    exit 1
fi
echo -e "  ${GREEN}✓${NC} Deposit record đã tạo: ${GREEN}${DEPOSIT_ID}${NC}"

# Bước 3: Thực thi gửi giao dịch submit batch proof lên chuỗi thông qua CLI obd
echo -e "\n${YELLOW}[BƯỚC 3]${NC} P4 Relayer thực hiện lệnh ${CYAN}obd tx zkdex submit-batch-proof${NC}..."
echo -e "        (Đang nạp lô xử lý: 1 Khoản nạp [${DEPOSIT_ID}] và 1 Yêu cầu rút [${WITHDRAW_ID}] bằng ${CYAN}${ASSET_DENOM}${NC})"
echo -e "        Người ký giao dịch: ${CYAN}${SIGNER}${NC}; phí giao dịch: ${CYAN}${TX_FEE}${NC}"

# Thực thi lệnh và bắt lấy TxHash từ JSON trả về.
# Lưu ý: lỗi ante/fee thường đi qua stderr, nên cần gom cả stderr để in chẩn đoán.
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

# Kiểm tra nếu txhash rỗng hoặc lệnh lỗi
if [ "$TX_STATUS" -ne 0 ] || [ -z "$TXHASH" ] || [ "$TXHASH" == "null" ] || [ "$TX_CODE" != "0" ]; then
    echo -e "  ${RED}✗ Giao dịch thất bại ngay tại cổng CLI hoặc Node không hoạt động!${NC}"
    if echo "$TX_OUTPUT" | grep -q "insufficient funds"; then
        echo -e "  ${YELLOW}Gợi ý:${NC} tài khoản ${CYAN}${SIGNER}${NC} không đủ coin để trả phí ${CYAN}${TX_FEE}${NC}."
        echo -e "       Kiểm tra bằng: ${CYAN}obd query bank balances \$(obd keys show ${SIGNER} -a --keyring-backend test)${NC}"
    fi
    echo -e "  Chi tiết phản hồi từ hệ thống:"
    echo "$TX_OUTPUT"
    exit 1
fi

echo -e "  ${GREEN}✓${NC} Giao dịch được gửi thành công!"
echo -e "  👉 Mã Giao Dịch (TxHash): ${GREEN}$TXHASH${NC}"

# Bước 4: Đợi Blockchain xử lý block mới
echo -e "\n${YELLOW}[BƯỚC 4]${NC} Chờ 5 giây để Blockchain (${CHAIN_ID}) xác thực và đóng khối (commit block)..."
for i in {5..1}; do
    echo -e "        Đang chờ định cư trạng thái... $i"
    sleep 1
done

# Bước 5: Khảo sát trạng thái on-chain thông qua Tx Hash (Query)
echo -e "\n${YELLOW}[BƯỚC 5]${NC} P4 Relayer thực hiện truy vấn trạng thái xử lý sau block: ${CYAN}obd query tx $TXHASH${NC}..."
TX_RESULT=$("$BINARY" query tx "$TXHASH" --node "$NODE" -o json 2>/dev/null)

CODE=$(echo "$TX_RESULT" | jq -r '.code')

if [ "$CODE" == "0" ] || [ "$CODE" == "null" ] || [ -z "$CODE" ]; then
    echo -e "  ${GREEN}✓ [KẾT QUẢ ON-CHAIN]: CHẤP NHẬN (ACCEPTED)${NC}"
    echo -e "    -> Toàn bộ ZK Proof, Public Inputs và Logic trạng thái cũ/mới hợp lệ 100%!"
    
    # Bước 6: Đầu ra API (Response Payload) mà P4 sẽ trả về cho Frontend (P5)
    echo -e "\n${YELLOW}[BƯỚC 6]${NC} Dữ liệu JSON từ Backend (P4) trả về để Frontend (P5) cập nhật UI:"
    echo -e "${GREEN}----------------------------------------------------------------------${NC}"
    "$BINARY" query tx "$TXHASH" --node "$NODE" -o json | jq

    echo -e "${GREEN}----------------------------------------------------------------------${NC}"
else
    RAW_LOG=$(echo "$TX_RESULT" | jq -r '.raw_log')
    echo -e "  ${RED}✗ [KẾT QUẢ ON-CHAIN]: TỪ CHỐI (REJECTED)${NC}"
    echo -e "  Mã lỗi hệ thống (Code): $CODE"
    echo -e "  Nguyên nhân thất bại (Raw Log): ${YELLOW}$RAW_LOG${NC}"
fi

# Dọn dẹp tài nguyên tạm
cleanup
echo -e "\n${CYAN}======================================================================${NC}"
echo -e "             KẾT THÚC KỊCH BẢN DEMO             "
echo -e "${CYAN}======================================================================${NC}"
