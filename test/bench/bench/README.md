# Bench — Mô hình 1 (on-chain orderbook + matching, `x/dex`)

Đo 2 vấn đề của P1: **slow** & **expensive**, trên 3 hệ quy mô sổ lệnh 10 / 100 / 1000.

## Chạy

```bash
# Mặc định: script tự reset chain sạch trước mỗi hệ (cần ignite)
bash test/bench/p1_measure.sh                # 10 100 1000
bash test/bench/p1_measure.sh 50 500         # scale tùy chọn

# Dùng chain đang chạy sẵn (bạn tự reset giữa các hệ):
MANAGE_CHAIN=0 bash test/bench/p1_measure.sh 100
```

Yêu cầu: `obd` đã build (qua `ignite chain serve` một lần), `jq`, `curl`, `awk`;
nếu `MANAGE_CHAIN=1` cần thêm `ignite`.

## 4 thông số đo

| Mã | Ý nghĩa | Cách lấy |
|----|---------|----------|
| A1b ⭐ | gas 1 lệnh **KHÔNG khớp** ở độ sâu sổ n | `gas_used` của tx đặt lệnh không cross |
| A1a | gas / **trade** = gas(SELL) + gas(BUY khớp) | cộng `gas_used` 2 tx |
| A3 | throughput (trade/s) | K trade ÷ (t_commit_cuối − t_broadcast_đầu) |
| A4 | RAM đỉnh của `obd` | `ps -o rss` lấy peak |

Kết quả ghi ra `test/bench/results/p1_<timestamp>.csv` + log ở `results/logs_<timestamp>/`.

## Đọc kết quả (kỳ vọng, suy từ code)

- `matching.go:53` quét **toàn bộ** orderbook mỗi lần đặt lệnh ⇒ **A1b tăng ~tuyến tính theo n**
  (bằng chứng O(n) → "expensive", kể cả lệnh không khớp vẫn tốn gas).
- Dựng sổ n lệnh ⇒ tổng chi phí ~**O(n²)**.
- **A3 giảm** khi n tăng (mỗi tx nặng hơn, block gas limit siết) → "slow".
- A4 tăng nhẹ theo n.

## Giới hạn (đọc trước khi kết luận)

- Chỉ đo P1. So với P2 (`x/zkdex`) làm ở script riêng; **verifier P2 hiện là stub**
  nên số của P2 chỉ là cận dưới.
- `orderId = market-blockHeight-creator[:6]` (msg_server_place_order.go:53): nhiều lệnh
  cùng creator trong **cùng 1 block** trùng orderId trong `Order` store. A1b không bị ảnh
  hưởng (chỉ quét sổ); A1a/A3 chỉ lấy chi phí phía **taker** nơi O(n) nằm.
- Refund phía BUY chưa cài (matching.go:41) → BUY đặt giá cao mất phần chênh vào module;
  không ảnh hưởng gas/thời gian, chỉ hao tiền BUYER (đã cấp dư).
