obd tx dex register-pairs ATOM USDT 0.01 1 --from alice --chain-id ob -y
sleep 1

obd q dex list-market
sleep 1
obd q dex list-order
sleep 1
obd q dex list-orderbook
sleep 1


obd tx dex place-order ATOM-USDT BUY 5 10 --from alice -y
sleep 1
obd tx dex place-order ATOM-USDT BUY 4 10 --from alice -y
sleep 1
obd tx dex place-order ATOM-USDT SELL 5 10 --from bob -y
sleep 1
obd tx dex place-order ATOM-USDT SELL 4 10 --from bob -y
sleep 1
obd tx dex place-order ATOM-USDT BUY 9 10 --from alice -y
sleep 1
obd tx dex place-order ATOM-USDT BUY 8 10 --from alice -y
sleep 1
obd tx dex place-order ATOM-USDT SELL 7 10 --from bob -y
sleep 1
obd tx dex place-order ATOM-USDT SELL 6 10 --from bob -y
sleep 1
obd tx dex place-order ATOM-USDT BUY 7 10 --from alice -y
sleep 1
obd tx dex place-order ATOM-USDT BUY 6 10 --from alice -y
sleep 1
obd tx dex place-order ATOM-USDT SELL 5 10 --from bob -y 
sleep 1
obd tx dex place-order ATOM-USDT SELL 4 10 --from bob -y
sleep 1


# LIST
obd q dex list-order
sleep 1
obd q dex list-orderbook
sleep 1