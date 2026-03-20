#!/bin/bash

set -euo pipefail

# 1. Khai báo các biến cơ bản
CHAIN_ID="ob"
DENOM="ATOM"
AMOUNT="1000$DENOM"
USER_NAME="alice"

echo -e "\033[0;34m[TEST] Bắt đầu quy trình kiểm tra Deposit...\033[0m"

# 2. Lấy địa chỉ ví của Alice từ bộ nhớ (keyring)
ALICE_ADDR=$(obd keys show $USER_NAME -a --keyring-backend test)
echo -e "Ví Alice: \033[0;32m$ALICE_ADDR\033[0m"

# 3. Lấy địa chỉ ví của Module Account 'backend'
# Cách này dùng query trực tiếp từ auth module
MODULE_ADDR=$(obd q auth module-account backend --output json | python3 -c 'import sys,json; d=json.load(sys.stdin); v=d.get("account",{}).get("value",{}) or {}; addr=v.get("address") or ""; print(addr)')
if [[ -z "$MODULE_ADDR" ]]; then
  echo "Không lấy được địa chỉ module account 'backend'. Kiểm tra jq và response query:" >&2
  obd q auth module-account backend --output json >&2
  exit 1
fi
echo -e "Ví Module Backend: \033[0;32m$MODULE_ADDR\033[0m"

echo "---------------------------------------------------"
echo "Số dư TRƯỚC khi Deposit:"
echo -e "\033[0;36mVí Alice:\033[0m"
obd q bank balances $ALICE_ADDR

echo -e "\033[0;36mVí Module:\033[0m"
obd q bank balances $MODULE_ADDR

echo "---------------------------------------------------"
echo -e "\033[1;33m[STEP] Thực hiện gửi 1000 $DENOM từ Alice vào Module...\033[1;33m"
# Gửi giao dịch
obd tx backend deposit $AMOUNT --from $USER_NAME --chain-id $CHAIN_ID --keyring-backend test -y

# Đợi 5 giây để block được confirm
echo "Đang đợi xử lý block..."
sleep 5

echo -e "\033[0;37m---------------------------------------------------\033[0m"
echo -e "\033[0;37mSố dư SAU khi Deposit:\033[0m"
echo -e "\033[0;36mVí Alice (Phải giảm $AMOUNT):\033[0m"
obd q bank balances $ALICE_ADDR

echo -e "\033[0;36mVí Module (Phải tăng $AMOUNT):\033[0m"
obd q bank balances $MODULE_ADDR