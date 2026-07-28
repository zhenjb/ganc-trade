#!/usr/bin/env bash
# =============================================================================
# p1_measure.sh — Đo "slow" & "expensive" của MÔ HÌNH 1 (on-chain orderbook +
# matching engine, module x/dex) trên 3 hệ quy mô sổ lệnh: 10 / 100 / 1000.
#
# Đo 4 thông số (READ-ONLY: không sửa code module, chỉ quan sát tx/chain):
#   A1a  gas / trade          = gas(place SELL @depth n) + gas(place BUY khớp)
#   A1b  gas lệnh KHÔNG khớp ⭐= gas 1 lệnh đặt vào sổ độ sâu n mà không khớp
#   A3   throughput (trade/s) = K lệnh khớp ÷ (t_commit_cuối − t_broadcast_đầu)
#   A4   RAM (peak RSS)       = đỉnh RSS của tiến trình `obd`
#
# Cơ chế: mỗi hệ chạy trên state SẠCH. Với MANAGE_CHAIN=1 (mặc định) script tự
# kill + `ignite chain serve --reset-once` để reset trước mỗi hệ.
#
# CÁCH DÙNG:
#   Terminal riêng KHÔNG cần mở sẵn chain nếu MANAGE_CHAIN=1 (script tự lo).
#     bash test/bench/p1_measure.sh                 # chạy 10 100 1000
#     bash test/bench/p1_measure.sh 50 500          # scale tùy chọn
#     MANAGE_CHAIN=0 bash test/bench/p1_measure.sh 100   # dùng chain đang chạy (bạn tự reset)
#
# BIẾN MÔI TRƯỜNG (override được):
#   SCALES, MANAGE_CHAIN, NODE, CHAIN_ID, HOME_DIR, OBDIR, OBD, IGNITE,
#   SELLER, BUYER, BASE_DENOM, QUOTE_DENOM, QTY, BASE_PRICE, GASLIM,
#   KEYRING, TX_WAIT, READY_WAIT, THROUGHPUT_K
#
# LƯU Ý TRUNG THỰC (đọc trước khi diễn giải số):
#  * gas_used là số THẬT chain tính, không phụ thuộc --gas. Ta đặt --gas cố định
#    đủ lớn để lệnh ở depth 1000 không out-of-gas; con số báo cáo là gas_used.
#  * orderId = market-blockHeight-creator[:6] (msg_server_place_order.go:53) nên
#    nhiều lệnh cùng creator trong CÙNG 1 block bị TRÙNG orderId trong Order store.
#    Không sao với A1b (chỉ đo quét sổ, không match). Với A1a/A3 (có match) điều
#    này có thể lệch phần settle của maker — ta chỉ lấy gas/thời gian của TAKER,
#    nơi chi phí O(n) thực sự nằm. Đã ghi rõ để không tô hồng.
#  * Refund phía BUY chưa cài (matching.go:41) → BUY đặt giá cao hơn giá khớp sẽ
#    "mất" phần chênh vào module. Ta cấp dư tiền cho BUYER; không ảnh hưởng gas.
#  * Đây chỉ đo P1. So sánh với P2 làm ở script riêng (P2 verifier hiện là stub).
# =============================================================================
set -u

# ----------------------------- Cấu hình ------------------------------------
SCALES_ARG=("$@")
if [ ${#SCALES_ARG[@]} -gt 0 ]; then SCALES="${SCALES_ARG[*]}"; else SCALES="${SCALES:-10 100 1000}"; fi

MANAGE_CHAIN="${MANAGE_CHAIN:-1}"
NODE="${NODE:-tcp://localhost:26657}"
CHAIN_ID="${CHAIN_ID:-ob}"
HOME_DIR="${HOME_DIR:-$HOME/.ob}"
OBDIR="${OBDIR:-$(cd "$(dirname "$0")/../../sw/ob" && pwd)}"
OBD="${OBD:-obd}"
IGNITE="${IGNITE:-ignite}"
KEYRING="${KEYRING:-test}"

SELLER="${SELLER:-bob}"          # đặt các lệnh SELL (maker) — cần BASE_DENOM
BUYER="${BUYER:-alice}"          # đặt các lệnh BUY  (taker) — cần QUOTE_DENOM
# Denom PHẢI là denom chain thực sự phát cho alice/bob (kiểm bằng
# `obd q bank balances <addr>`). Trước đây mặc định ATOM/USDT — những đồng KHÔNG
# tồn tại trên chain này (chain phát uatom/uosmo/uusdc) ⇒ mọi place-order fail ở
# DeliverTx (tx vẫn vào block và vẫn TIÊU sequence ⇒ trông như lỗi "sequence
# mismatch", nhưng gốc là denom).
BASE_DENOM="${BASE_DENOM:-uatom}"
QUOTE_DENOM="${QUOTE_DENOM:-uusdc}"
# marketId do keeper sinh: BaseDenom + "-" + QuoteDenom (msg_server_register_pairs.go)
MARKET="${BASE_DENOM}-${QUOTE_DENOM}"
QTY="${QTY:-1}"
BASE_PRICE="${BASE_PRICE:-100}"  # giá SELL chạy BASE_PRICE .. BASE_PRICE+n-1
GASLIM="${GASLIM:-50000000}"     # gas_wanted cố định (đủ lớn cho depth 1000)

TX_WAIT="${TX_WAIT:-60}"         # số giây tối đa chờ 1 tx commit
READY_WAIT="${READY_WAIT:-300}"  # số giây tối đa chờ chain lên block (lần đầu build lâu)
THROUGHPUT_K="${THROUGHPUT_K:-}" # số lệnh khớp cho phép đo A3 (mặc định = depth)

TS="$(date +%Y%m%d-%H%M%S)"
OUTDIR="$(cd "$(dirname "$0")" && pwd)/results"
mkdir -p "$OUTDIR"
CSV="$OUTDIR/p1_${TS}.csv"
LOGDIR="$OUTDIR/logs_${TS}"; mkdir -p "$LOGDIR"

C_G='\033[0;32m'; C_Y='\033[1;33m'; C_R='\033[0;31m'; C_C='\033[0;36m'; C_N='\033[0m'
info(){ echo -e "${C_C}[i]${C_N} $*"; }
ok(){   echo -e "${C_G}[✓]${C_N} $*"; }
warn(){ echo -e "${C_Y}[!]${C_N} $*" >&2; }
die(){  echo -e "${C_R}[✗]${C_N} $*" >&2; exit 1; }

QFLAGS="--node $NODE --output json"
TXFLAGS="--chain-id $CHAIN_ID --node $NODE --keyring-backend $KEYRING --gas $GASLIM --broadcast-mode sync --yes --output json"

# ----------------------------- Preflight -----------------------------------
command -v "$OBD"  >/dev/null 2>&1 || die "Không thấy '$OBD'. Đã build chain chưa? (ignite chain serve tạo ~/go/bin/obd)"
command -v jq      >/dev/null 2>&1 || die "Cần 'jq'."
command -v curl    >/dev/null 2>&1 || die "Cần 'curl'."
command -v awk     >/dev/null 2>&1 || die "Cần 'awk'."
if [ "$MANAGE_CHAIN" = "1" ]; then
  command -v "$IGNITE" >/dev/null 2>&1 || die "MANAGE_CHAIN=1 cần '$IGNITE'. Hoặc chạy MANAGE_CHAIN=0 với chain sẵn có."
  [ -d "$OBDIR" ] || die "Không thấy thư mục chain: $OBDIR"
fi
n_scales=$(echo $SCALES | wc -w)
if [ "$MANAGE_CHAIN" = "0" ] && [ "$n_scales" -gt 1 ]; then
  warn "MANAGE_CHAIN=0 nhưng có $n_scales hệ. Độ sâu sổ sẽ CỘNG DỒN giữa các hệ → sai."
  warn "Nên chạy từng hệ một và tự reset chain (rm -rf $HOME_DIR; ignite chain serve --reset-once) giữa các lần."
fi

# ----------------------------- Chain lifecycle -----------------------------
rpc_height(){ curl -s "http://${NODE#tcp://}/status" 2>/dev/null | jq -r '.result.sync_info.latest_block_height // 0' 2>/dev/null; }
block_time(){ curl -s "http://${NODE#tcp://}/block?height=$1" 2>/dev/null | jq -r '.result.block.header.time // empty' 2>/dev/null; }

wait_ready(){
  local h prev=0 stable=0
  for ((i=0;i<READY_WAIT;i++)); do
    h=$(rpc_height)
    if [ "${h:-0}" -ge 1 ] 2>/dev/null; then
      if [ "$h" -gt "$prev" ]; then prev=$h; stable=$((stable+1)); fi
      [ "$stable" -ge 2 ] && { ok "Chain sẵn sàng (height=$h)"; return 0; }
    fi
    sleep 1
  done
  die "Chain không lên block trong ${READY_WAIT}s (xem log: $LOGDIR/chain.log)"
}

stop_chain(){
  pkill -f 'ignite chain serve' 2>/dev/null
  pkill -x obd 2>/dev/null
  sleep 3
}

start_chain(){
  stop_chain
  rm -rf "$HOME_DIR"    # đảm bảo genesis sạch tuyệt đối
  info "Khởi động chain sạch (ignite chain serve --reset-once)…"
  ( cd "$OBDIR" && nohup "$IGNITE" chain serve --reset-once >"$LOGDIR/chain.log" 2>&1 & )
  sleep 5
  wait_ready
}

# ----------------------------- Account helpers -----------------------------
addr_of(){ "$OBD" keys show "$1" -a --keyring-backend "$KEYRING" 2>/dev/null; }
acc_json(){ "$OBD" q auth account "$1" $QFLAGS 2>/dev/null; }
get_seq(){ acc_json "$1" | jq -r '.account.sequence // .account.value.sequence // .sequence // "0"' 2>/dev/null; }

# Chờ 1 tx commit; in ra: "<code> <gas_used> <height>". code!=0 -> lỗi.
wait_tx(){
  local hash="$1" out code
  [ -z "$hash" ] || [ "$hash" = "null" ] && { echo "-1 0 0"; return 1; }
  for ((i=0;i<TX_WAIT;i++)); do
    out=$("$OBD" q tx "$hash" $QFLAGS 2>/dev/null)
    if [ -n "$out" ] && echo "$out" | jq -e '.txhash' >/dev/null 2>&1; then
      code=$(echo "$out" | jq -r '.code // 0')
      echo "$code $(echo "$out" | jq -r '.gas_used // 0') $(echo "$out" | jq -r '.height // 0')"
      [ "$code" = "0" ] && return 0 || return 2
    fi
    sleep 1
  done
  echo "-2 0 0"; return 1   # timeout (chưa commit)
}

# Đặt 1 lệnh với sequence chỉ định. In txhash (rỗng nếu CheckTx từ chối).
place(){
  local from="$1" seq="$2" side="$3" price="$4" resp code
  resp=$("$OBD" tx dex place-order "$MARKET" "$side" "$price" "$QTY" \
          --from "$from" --sequence "$seq" $TXFLAGS 2>/dev/null)
  code=$(echo "$resp" | jq -r '.code // 999' 2>/dev/null)
  if [ "$code" != "0" ]; then
    warn "CheckTx từ chối (from=$from seq=$seq side=$side price=$price code=$code): $(echo "$resp" | jq -r '.raw_log // "?"' | head -c 160)"
    echo ""; return 1
  fi
  echo "$resp" | jq -r '.txhash // empty'
}

# Đặt 1 lệnh RỒI CHỜ COMMIT. In "<code> <gas_used> <height>" (như wait_tx).
#
# BẮT BUỘC tuần tự — Phase 1 có HAI ràng buộc khiến bắn song song luôn hỏng:
#   (1) SDK v0.50+ kiểm sequence ở CheckTx dựa trên state ĐÃ COMMIT, nên mỗi
#       account chỉ an toàn 1 tx / block.
#   (2) orderId = fmt.Sprintf("%s-%d-%s", marketId, blockHeight, Creator[:6])
#       mà Creator[:6] == "cosmos" với MỌI địa chỉ bech32 ⇒ hai lệnh cùng market
#       trong cùng block có CÙNG orderId và ĐÈ nhau (msg_server_place_order.go +
#       matching.go:138 k.Order.Set). Nhiều tài khoản cũng không cứu được.
# ⇒ Trần nạp lệnh của Phase 1 là 1 lệnh / market / block. Đó vừa là ràng buộc
#   vận hành, vừa là MỘT KẾT QUẢ ĐO đáng ghi lại khi so với Phase 2.
#
# Sequence: resync từ chính thông báo lỗi ("expected N") — chuẩn hơn
# `q auth account` (giá trị đó trễ khi còn tx trong mempool). Lưu ý tx FAIL vẫn
# tiêu sequence, nên không được tự tăng bộ đếm khi thất bại.
place_wait(){
  local from="$1" side="$2" price="$3" seq resp code hash exp attempt
  seq=$(get_seq "$(addr_of "$from")")
  for attempt in 1 2 3; do
    resp=$("$OBD" tx dex place-order "$MARKET" "$side" "$price" "$QTY" \
            --from "$from" --sequence "$seq" $TXFLAGS 2>/dev/null)
    code=$(echo "$resp" | jq -r '.code // 999' 2>/dev/null)
    if [ "$code" = "0" ]; then
      hash=$(echo "$resp" | jq -r '.txhash // empty')
      wait_tx "$hash"          # in "<code> <gas_used> <height>"
      return
    fi
    exp=$(echo "$resp" | jq -r '.raw_log // ""' | grep -oE 'expected [0-9]+' | head -1 | awk '{print $2}')
    if [ -n "$exp" ]; then seq="$exp"; continue; fi   # lệch sequence → dùng số chain yêu cầu
    warn "CheckTx từ chối (from=$from side=$side price=$price code=$code): $(echo "$resp" | jq -r '.raw_log // "?"' | head -c 160)"
    echo "-1 0 0"; return 1
  done
  warn "hết lượt resync sequence (from=$from side=$side price=$price)"
  echo "-1 0 0"; return 1
}

# ----------------------------- RAM sampler ---------------------------------
RAM_PID=""
start_ram(){
  local f="$1"; echo 0 >"$f"
  ( peak=0
    while :; do
      pid=$(pgrep -x obd | head -1)
      if [ -n "$pid" ]; then
        rss=$(ps -o rss= -p "$pid" 2>/dev/null | tr -d ' ')
        if [ -n "$rss" ] && [ "$rss" -gt "$peak" ] 2>/dev/null; then peak=$rss; echo "$peak" >"$f"; fi
      fi
      sleep 0.5
    done ) &
  RAM_PID=$!
}
stop_ram(){ [ -n "$RAM_PID" ] && kill "$RAM_PID" 2>/dev/null; RAM_PID=""; }

cleanup(){ stop_ram; [ "$MANAGE_CHAIN" = "1" ] && stop_chain; }
trap cleanup EXIT INT TERM

# ----------------------------- Đăng ký market ------------------------------
register_market(){
  local seq h
  seq=$(get_seq "$(addr_of "$BUYER")")
  info "Đăng ký market $MARKET (tick=1 lot=1) từ $BUYER…"
  local resp
  resp=$("$OBD" tx dex register-pairs "$BASE_DENOM" "$QUOTE_DENOM" 1 1 \
          --from "$BUYER" --sequence "$seq" $TXFLAGS 2>/dev/null)
  local hash; hash=$(echo "$resp" | jq -r '.txhash // empty')
  read -r code _ _ < <(wait_tx "$hash")
  [ "$code" = "0" ] || die "Đăng ký market thất bại (code=$code). Log: $LOGDIR"
  ok "Market $MARKET đã đăng ký."
}

# ----------------------------- Một hệ (scale n) ----------------------------
run_scale(){
  local n="$1"
  echo -e "\n${C_Y}==================== HỆ n=$n ====================${C_N}"

  [ "$MANAGE_CHAIN" = "1" ] && start_chain
  local ramfile="$LOGDIR/ram_${n}.txt"; start_ram "$ramfile"

  register_market

  local saddr baddr   # sseq/baseq không còn cần: place_wait tự quản sequence
  saddr=$(addr_of "$SELLER"); baddr=$(addr_of "$BUYER")
  [ -n "$saddr" ] && [ -n "$baddr" ] || die "Không lấy được địa chỉ $SELLER/$BUYER (keyring=$KEYRING)."

  # --- SEED: n lệnh SELL giá phân biệt, chưa có BUY nên tất cả nằm im trong sổ.
  # TUẦN TỰ, chờ commit từng lệnh: Phase 1 chỉ nhận 1 lệnh / market / block
  # (sequence + orderId đụng — xem chú thích ở place_wait). Bắn song song thì
  # phần lớn lệnh sẽ bị từ chối hoặc ĐÈ nhau, depth không bao giờ đạt n.
  info "Seed $n lệnh SELL (giá $BASE_PRICE..$((BASE_PRICE+n-1)))… (tuần tự, ~1 block/lệnh)"
  local depth=0
  local i price code gas hgt
  for ((i=0;i<n;i++)); do
    price=$((BASE_PRICE+i))
    read -r code gas hgt < <(place_wait "$SELLER" SELL "$price")
    if [ "$code" = "0" ]; then
      depth=$((depth+1))
    else
      warn "seed lệnh giá $price thất bại (code=$code)"
    fi
    (( i % 10 == 0 )) && info "  …đã seed $((i+1))/$n (depth=$depth)"
  done
  ok "Độ sâu sổ đạt được: depth=$depth (mục tiêu n=$n)"
  [ "$depth" -ge 1 ] || die "Seed thất bại, depth=0. Xem $LOGDIR/chain.log"

  # --- A1b ⭐: 1 lệnh SELL nữa ở giá cao, KHÔNG có BUY → không khớp, chỉ quét sổ
  info "A1b: đo gas lệnh KHÔNG khớp ở depth≈$depth…"
  local gas_nomatch=0 hgt_b=0
  read -r code gas_nomatch hgt_b < <(place_wait "$SELLER" SELL "$((BASE_PRICE+n+5))")
  [ "$code" = "0" ] || warn "A1b tx code=$code (gas có thể không tin cậy)"
  ok "A1b gas lệnh-không-khớp = $gas_nomatch"

  # --- A1a: 1 lệnh BUY khớp lệnh SELL rẻ nhất → 1 trade. gas/trade = sell + buy
  info "A1a: đo gas 1 trade (BUY khớp) ở depth≈$depth…"
  local gas_buy=0 hgt_a=0
  read -r code gas_buy hgt_a < <(place_wait "$BUYER" BUY "$((BASE_PRICE+n+5))")
  [ "$code" = "0" ] || warn "A1a BUY tx code=$code"
  local gas_trade=$((gas_nomatch + gas_buy))
  ok "A1a gas(SELL)=$gas_nomatch + gas(BUY)=$gas_buy = gas/trade=$gas_trade"

  # --- A3: throughput = K lệnh BUY khớp ÷ Δt.
  # KHÔNG bắn song song: trần thật của Phase 1 là 1 lệnh/market/block, nên bắn
  # song song chỉ tạo ra tx bị từ chối và một con số THẤP GIẢ TẠO. Đo tuần tự
  # cho ra đúng năng lực trần: throughput ≈ 1 / block-time.
  local K="${THROUGHPUT_K:-$depth}"
  [ "$K" -gt "$depth" ] && K="$depth"
  info "A3: $K lệnh BUY khớp, tuần tự (trần cấu trúc: 1 lệnh/market/block)…"
  local committed=0 hmax=0
  local t0 t1
  t0=$(date +%s.%N)
  for ((i=0;i<K;i++)); do
    read -r code gas hgt < <(place_wait "$BUYER" BUY "$((BASE_PRICE+n+5))")
    if [ "$code" = "0" ]; then
      committed=$((committed+1))
      [ "${hgt:-0}" -gt "$hmax" ] && hmax=$hgt
    fi
  done
  t1=$(date +%s.%N)
  local dt tput
  dt=$(awk -v a="$t0" -v b="$t1" 'BEGIN{printf "%.3f", b-a}')
  tput=$(awk -v k="$committed" -v d="$dt" 'BEGIN{ if(d>0) printf "%.3f", k/d; else print "NA"}')
  ok "A3 throughput = $committed trade / ${dt}s = ${tput} trade/s (block cuối=$hmax)"

  # --- A4: RAM đỉnh
  stop_ram
  local ram_kb; ram_kb=$(cat "$ramfile" 2>/dev/null || echo 0)
  local ram_mb; ram_mb=$(awk -v k="$ram_kb" 'BEGIN{printf "%.1f", k/1024}')
  ok "A4 RAM peak (obd) = ${ram_mb} MB"

  echo "$n,$depth,$gas_nomatch,$gas_buy,$gas_trade,$committed,$dt,$tput,$ram_kb,$ram_mb" >>"$CSV"

  [ "$MANAGE_CHAIN" = "1" ] && stop_chain
}

# ----------------------------- Main ----------------------------------------
echo -e "${C_C}P1 measurement — market=$MARKET seller=$SELLER buyer=$BUYER gas=$GASLIM${C_N}"
echo -e "${C_C}Hệ: $SCALES | MANAGE_CHAIN=$MANAGE_CHAIN | node=$NODE${C_N}"
echo "scale_n,depth,A1b_gas_nomatch,A1a_gas_buy,A1a_gas_per_trade,A3_trades_committed,A3_seconds,A3_trades_per_s,A4_ram_kb,A4_ram_mb" >"$CSV"

if [ "$MANAGE_CHAIN" = "0" ]; then
  h=$(rpc_height); [ "${h:-0}" -ge 1 ] 2>/dev/null || die "MANAGE_CHAIN=0 nhưng chain chưa chạy ở $NODE."
fi

for n in $SCALES; do run_scale "$n"; done

echo -e "\n${C_G}==================== KẾT QUẢ ====================${C_N}"
column -t -s, "$CSV" 2>/dev/null || cat "$CSV"
echo -e "\nCSV: ${C_C}$CSV${C_N}"
echo -e "Log: ${C_C}$LOGDIR${C_N}"
echo -e "\n${C_Y}Đọc số:${C_N} A1b tăng ~tuyến tính theo n ⇒ chứng minh quét sổ O(n) (matching.go:53)."
echo -e "        Tổng chi phí dựng sổ ~O(n²). throughput giảm khi n tăng. Đây là 'expensive' & 'slow' của P1."
