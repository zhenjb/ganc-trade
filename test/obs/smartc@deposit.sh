#!/bin/bash

set -euo pipefail

# Declare basic variables
CHAIN_ID="ob"
DENOM="uatom"
AMOUNT_VALUE="1000"
AMOUNT="${AMOUNT_VALUE}${DENOM}"
USER_NAME="alice"

B='\033[0;34m'
G='\033[0;32m'
R='\033[0;31m'
Y='\033[1;33m'
NC='\033[0m'

echo -e "${Y}[TEST]         Deposit token with smart contract...${NC}"

# Retrieve Alice's wallet address from memory (keyring).
echo -e "${G}[SYS]          Call 'obd keys show .. -a --keyring-backend test${NC}'"
ALICE_ADDR=$(obd keys show $USER_NAME -a --keyring-backend test)
echo -e "${B}[FIN]          Alice Wallet: ${Y}$ALICE_ADDR${NC}"

# Get the wallet address of the 'backend' Account Module.
echo -e "${G}[SYS]          Call 'obd q auth module-account backend --ouput json ...${NC}"
MODULE_ADDR=$(obd q auth module-account backend --output json | python3 -c 'import sys,json; d=json.load(sys.stdin); v=d.get("account",{}).get("value",{}) or {}; addr=v.get("address") or ""; print(addr)')
if [[ -z "$MODULE_ADDR" ]]; then
  echo "${R}[SYS]          Unable to retrieve the address of the 'backend' account module. Check the query and response query:" >&2
  obd q auth module-account backend --output json >&2
  exit 1
fi
echo -e "${B}[FIN]          SmartC Wallet: ${Y}$MODULE_ADDR${NC}"


# Balance
echo -e "${G}[SYS]          Call 'obd q bank balances .. -o json ...'${NC}"
BALANCE_ALICE=$(obd q bank balances $ALICE_ADDR -o json | jq -r '[.balances[] | .amount + .denom] | join(", ")')
echo -e "${B}[BALANCE]      Alice Wallet: $BALANCE_ALICE${NC}"
echo -e "${G}[SYS]          Call 'obd q bank balances .. -o json ...'${NC}"
BALANCE_SC=$(obd q bank balances $MODULE_ADDR -o json | jq -r '[.balances[] | .amount + .denom] | join(", ")')
echo -e "${B}[BALANCE]      SmartC Wallet: $BALANCE_SC${NC}"


# Transaction
echo -e "${Y}[SYS]          Send $AMOUNT from Alice to the Module....${NC}"
echo -e "${G}[SYS]          Call 'obd tx backend deposit .. --from .. --chain-id .. --keyring-backend test -y -o json'${NC}"
TX_RESULT=$(obd tx backend deposit $AMOUNT --from $USER_NAME --chain-id $CHAIN_ID --keyring-backend test -y -o json)
TX_CODE=$(echo "$TX_RESULT" | jq -r '.code // 0')
if [[ "$TX_CODE" != "0" ]]; then
  echo -e "${R}[ERR]          Deposit rejected during broadcast: code=$TX_CODE${NC}" >&2
  echo "$TX_RESULT" | jq . >&2
  exit 1
fi
HANDLE_TRANSACTION=$(echo "$TX_RESULT" | jq -r '.txhash')
echo -e "${B}[FIN]          Deposit Tx: ${Y}$HANDLE_TRANSACTION${NC}"
echo -e "${G}[SYS]          In progress ...${NC}"
echo -e "${B}[TRANS]        Waiting  ...${NC}"
sleep 5

TX_QUERY=$(obd q tx "$HANDLE_TRANSACTION" -o json)
DELIVER_CODE=$(echo "$TX_QUERY" | jq -r '.code // 0')
if [[ "$DELIVER_CODE" != "0" ]]; then
  echo -e "${R}[ERR]          Deposit rejected on-chain: code=$DELIVER_CODE${NC}" >&2
  echo "$TX_QUERY" | jq . >&2
  exit 1
fi


# Balance
echo -e "${G}[SYS]          Call 'obd q bank balances .. -o json ...'${NC}"
BALANCE_ALICE=$(obd q bank balances $ALICE_ADDR -o json | jq -r '[.balances[] | .amount + .denom] | join(", ")')
echo -e "${B}[BALANCE]      Alice Wallet: $BALANCE_ALICE${NC}"
echo -e "${G}[SYS]          Call 'obd q bank balances .. -o json ...'${NC}"
BALANCE_SC=$(obd q bank balances $MODULE_ADDR -o json | jq -r '[.balances[] | .amount + .denom] | join(", ")')
echo -e "${B}[BALANCE]      SmartC Wallet: $BALANCE_SC${NC}"
