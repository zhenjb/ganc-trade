# REGISTER
obd tx dex register-pairs uatom uusdc 0.01 1 --from alice --chain-id ob -y
sleep 1
obd tx dex register-pairs uusdc uatom 0.01 1 --from alice --chain-id ob -y
sleep 1


# LIST
obd q dex list-market
sleep 0.5
obd q dex list-order
sleep 0.5
obd q dex list-orderbook
sleep 0.5


# BALANCE
obd q bank balances $(obd keys show alice -a)
sleep 2
obd q bank balances $(obd keys show bob -a)
sleep 2


# SELL
obd tx dex place-order uatom-uusdc SELL 9 10 --from bob -y 
sleep 2


# BALANCE
obd q bank balances $(obd keys show alice -a)
sleep 2
obd q bank balances $(obd keys show bob -a)
sleep 2


# BUY
obd tx dex place-order uatom-uusdc BUY 1 10 --from alice -y
sleep 2
obd tx dex place-order uatom-uusdc BUY 9 10 --from alice -y
sleep 2


# BALANCE
obd q bank balances $(obd keys show alice -a)
sleep 2
obd q bank balances $(obd keys show bob -a)
sleep 2


# LIST
obd q dex list-market
sleep 0.5
obd q dex list-order
sleep 0.5
obd q dex list-orderbook
sleep 0.5
