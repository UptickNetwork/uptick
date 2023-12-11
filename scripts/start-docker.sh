#!/bin/bash

# CHAINID="uptick_7777-1"
# MONIKER="mymoniker"
DATA_DIR="/data-dir"

#echo "create and add new keys"
#./uptickd keys add $KEY --home $DATA_DIR --no-backup --chain-id $CHAINID --algo "eth_secp256k1" --keyring-backend test
#echo "init Uptick with moniker=$MONIKER and chain-id=$CHAINID"
#./uptickd init $MONIKER --chain-id $CHAINID --home $DATA_DIR
#echo "prepare genesis: Allocate genesis accounts"
#./uptickd add-genesis-account \
#"$(./uptickd keys show $KEY -a --home $DATA_DIR --keyring-backend test)" 1000000000000000000auptick,1000000000000000000stake \
#--home $DATA_DIR --keyring-backend test
#echo "prepare genesis: Sign genesis transaction"
#./uptickd gentx $KEY 1000000000000000000stake --keyring-backend test --home $DATA_DIR --keyring-backend test --chain-id $CHAINID
#echo "prepare genesis: Collect genesis tx"
#./uptickd collect-gentxs --home $DATA_DIR
#echo "prepare genesis: Run validate-genesis to ensure everything worked and that the genesis file is setup correctly"
./uptickd validate-genesis --home $DATA_DIR

echo "starting uptick node $i in background ..."
./uptickd start --pruning=nothing --rpc.unsafe \
--keyring-backend test --home $DATA_DIR \
>$DATA_DIR/node.log 2>&1 & disown

echo "started uptick node"
tail -f /dev/null