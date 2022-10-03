KEY="mykey"
CHAINID="uptick_7777-1"
MONIKER="localtestnet"
KEYRING="test"
KEYALGO="eth_secp256k1"
LOGLEVEL="debug"
# to trace evm
#TRACE="--trace"
TRACE=""

echo "xxl 0"
# validate dependencies are installed
command -v jq >/dev/null 2>&1 || {
    echo >&2 "jq not installed. More info: https://stedolan.github.io/jq/download/"
    exit 1
}

echo "xxl 1"
# remove existing daemon
rm -rf ~/.uptickd*

make install

echo "xxl 2"
uptickd config keyring-backend $KEYRING
uptickd config chain-id $CHAINID

echo "xxl 3"
# if $KEY exists it should be deleted
uptickd keys add $KEY --keyring-backend $KEYRING --algo $KEYALGO

echo "xxl 4"
# Set moniker and chain-id for Uptick (Moniker can be anything, chain-id must be an integer)
uptickd init $MONIKER --chain-id $CHAINID

# Change parameter token denominations to auptick
cat $HOME/.uptickd/config/genesis.json | jq '.app_state["staking"]["params"]["bond_denom"]="auptick"' >$HOME/.uptickd/config/tmp_genesis.json && mv $HOME/.uptickd/config/tmp_genesis.json $HOME/.uptickd/config/genesis.json
cat $HOME/.uptickd/config/genesis.json | jq '.app_state["crisis"]["constant_fee"]["denom"]="auptick"' >$HOME/.uptickd/config/tmp_genesis.json && mv $HOME/.uptickd/config/tmp_genesis.json $HOME/.uptickd/config/genesis.json
cat $HOME/.uptickd/config/genesis.json | jq '.app_state["gov"]["deposit_params"]["min_deposit"][0]["denom"]="auptick"' >$HOME/.uptickd/config/tmp_genesis.json && mv $HOME/.uptickd/config/tmp_genesis.json $HOME/.uptickd/config/genesis.json
cat $HOME/.uptickd/config/genesis.json | jq '.app_state["mint"]["params"]["mint_denom"]="auptick"' >$HOME/.uptickd/config/tmp_genesis.json && mv $HOME/.uptickd/config/tmp_genesis.json $HOME/.uptickd/config/genesis.json
cat $HOME/.uptickd/config/genesis.json | jq '.app_state["evm"]["params"]["evm_denom"]="auptick"' > $HOME/.uptickd/config/tmp_genesis.json && mv $HOME/.uptickd/config/tmp_genesis.json $HOME/.uptickd/config/genesis.json

echo "xxl 5"
# increase block time (?)
# cat $HOME/.uptickd/config/genesis.json | jq '.consensus_params["block"]["time_iota_ms"]="30000"' >$HOME/.uptickd/config/tmp_genesis.json && mv $HOME/.uptickd/config/tmp_genesis.json $HOME/.uptickd/config/genesis.json

# Set gas limit in genesis
# cat $HOME/.uptickd/config/genesis.json | jq '.consensus_params["block"]["max_gas]="10000000"' >$HOME/.uptickd/config/tmp_genesis.json && mv $HOME/.uptickd/config/tmp_genesis.json $HOME/.uptickd/config/genesis.json

echo "xxl 6"
# disable produce empty block
if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' 's/create_empty_blocks = true/create_empty_blocks = false/g' $HOME/.uptickd/config/config.toml
else
    sed -i 's/create_empty_blocks = true/create_empty_blocks = false/g' $HOME/.uptickd/config/config.toml
fi

echo "xxl 7"
if [[ $1 == "pending" ]]; then
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/create_empty_blocks_interval = "0s"/create_empty_blocks_interval = "30s"/g' $HOME/.uptickd/config/config.toml
        sed -i '' 's/timeout_propose = "3s"/timeout_propose = "30s"/g' $HOME/.uptickd/config/config.toml
        sed -i '' 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' $HOME/.uptickd/config/config.toml
        sed -i '' 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' $HOME/.uptickd/config/config.toml
        sed -i '' 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' $HOME/.uptickd/config/config.toml
        sed -i '' 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' $HOME/.uptickd/config/config.toml
        sed -i '' 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' $HOME/.uptickd/config/config.toml
        sed -i '' 's/timeout_commit = "5s"/timeout_commit = "150s"/g' $HOME/.uptickd/config/config.toml
        sed -i '' 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' $HOME/.uptickd/config/config.toml
    else
        sed -i 's/create_empty_blocks_interval = "0s"/create_empty_blocks_interval = "30s"/g' $HOME/.uptickd/config/config.toml
        sed -i 's/timeout_propose = "3s"/timeout_propose = "30s"/g' $HOME/.uptickd/config/config.toml
        sed -i 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' $HOME/.uptickd/config/config.toml
        sed -i 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' $HOME/.uptickd/config/config.toml
        sed -i 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' $HOME/.uptickd/config/config.toml
        sed -i 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' $HOME/.uptickd/config/config.toml
        sed -i 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' $HOME/.uptickd/config/config.toml
        sed -i 's/timeout_commit = "5s"/timeout_commit = "150s"/g' $HOME/.uptickd/config/config.toml
        sed -i 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' $HOME/.uptickd/config/config.toml
    fi
fi

echo "xxl 8"
# Allocate genesis accounts (cosmos formatted addresses)
uptickd add-genesis-account $KEY 100000000000000000000000000auptick --keyring-backend $KEYRING

echo "xxl 9"
# Update total supply with claim values
validators_supply=$(cat $HOME/.uptickd/config/genesis.json | jq -r '.app_state["bank"]["supply"][0]["amount"]')
# Bc is required to add this big numbers
# total_supply=$(bc <<< "$amount_to_claim+$validators_supply")
total_supply=100000000000000000000000000
cat $HOME/.uptickd/config/genesis.json | jq -r --arg total_supply "$total_supply" '.app_state["bank"]["supply"][0]["amount"]=$total_supply' >$HOME/.uptickd/config/tmp_genesis.json && mv $HOME/.uptickd/config/tmp_genesis.json $HOME/.uptickd/config/genesis.json

echo "xxl 10"
# Sign genesis transaction
uptickd gentx $KEY 1000000000000000000000auptick --keyring-backend $KEYRING --chain-id $CHAINID

echo "xxl 11"
# Collect genesis tx
uptickd collect-gentxs

echo "xxl 12"
# Run this to ensure everything worked and that the genesis file is setup correctly
uptickd validate-genesis

if [[ $1 == "pending" ]]; then
    echo "pending mode is on, please wait for the first block committed."
fi

# Start the node (remove the --pruning=nothing flag if historical queries are not needed)
uptickd start --pruning=nothing $TRACE --log_level $LOGLEVEL --minimum-gas-prices=0auptick --json-rpc.api eth,txpool,personal,net,debug,web3