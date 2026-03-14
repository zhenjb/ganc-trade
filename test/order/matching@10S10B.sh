obd tx dex register-pairs ATOM USDT 0.01 1 --from alice --chain-id ob -y
sleep 0.5

obd q dex list-market
sleep 0.5
obd q dex list-order
sleep 0.5
obd q dex list-orderbook
sleep 0.5


# SELL
obd tx dex place-order ATOM-USDT SELL 9 10 --from bob -y 
sleep 2
obd tx dex place-order ATOM-USDT SELL 8 10 --from bob -y
sleep 2
obd tx dex place-order ATOM-USDT SELL 7 10 --from bob -y
sleep 2
obd tx dex place-order ATOM-USDT SELL 6 10 --from bob -y
sleep 2
obd tx dex place-order ATOM-USDT SELL 5 10 --from bob -y
sleep 2
obd tx dex place-order ATOM-USDT SELL 4 10 --from bob -y
sleep 2
obd tx dex place-order ATOM-USDT SELL 3 10 --from bob -y
sleep 2
obd tx dex place-order ATOM-USDT SELL 2 10 --from bob -y
sleep 2
obd tx dex place-order ATOM-USDT SELL 1 10 --from bob -y
sleep 2


# BUY
obd tx dex place-order ATOM-USDT BUY 1 10 --from alice -y
sleep 2
obd tx dex place-order ATOM-USDT BUY 2 10 --from alice -y
sleep 2
obd tx dex place-order ATOM-USDT BUY 3 10 --from alice -y
sleep 2
obd tx dex place-order ATOM-USDT BUY 4 10 --from alice -y
sleep 2
obd tx dex place-order ATOM-USDT BUY 5 10 --from alice -y
sleep 2
obd tx dex place-order ATOM-USDT BUY 6 10 --from alice -y
sleep 2
obd tx dex place-order ATOM-USDT BUY 7 10 --from alice -y
sleep 2
obd tx dex place-order ATOM-USDT BUY 8 10 --from alice -y
sleep 2
obd tx dex place-order ATOM-USDT BUY 9 10 --from alice -y
sleep 2

# LIST
obd q dex list-order
sleep 0.5
obd q dex list-orderbook
sleep 0.5