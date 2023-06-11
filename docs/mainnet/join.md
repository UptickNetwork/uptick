<!--
order: 1
-->

# Joining a Mainnet

This document outlines the steps to join an existing mainnet {synopsis}

## Pick a Testnet

You specify the network you want to join by setting the **genesis file** and **seeds**. If you need more information about past networks, check our [mainnet repo](https://github.com/UptickNetwork/uptick-mainnet).

| Network Chain ID | Description                       | Site                                                                     | Version                                               |
|------------------|-----------------------------------|--------------------------------------------------------------------------|-------------------------------------------------------|
| `uptick_117-1`   | Uptick mainnet | [uptick_117-1](https://github.com/UptickNetwork/uptick-mainnet/tree/main/uptick_117-1) | [`v0.2.4`](https://github.com/UptickNetwork/uptick/tree/v0.2.4) |

## Install `uptickd`

Follow the [installation](./../quickstart/installation) document to install the {{ $themeConfig.project.name }} binary `{{ $themeConfig.project.binary }}`.

:::warning
Make sure you have the right version of `{{ $themeConfig.project.binary }}` installed.
:::

### Save Chain ID

We recommend saving the mainnet `chain-id` into your `{{ $themeConfig.project.binary }}`'s `client.toml`. This will make it so you do not have to manually pass in the `chain-id` flag for every CLI command.

::: tip
See the Official [Chain IDs](./../basics/chain_id.md#official-chain-ids) for reference.
:::

```bash
uptickd config chain-id uptick_117-1
```

## Initialize Node

We need to initialize the node to create all the necessary validator and node configuration files:

```bash
# initialize node configurations
uptickd init <moniker> --chain-id uptick_117-1

# download testnel public genesis.json
curl -o $HOME/.uptickd/config/genesis.json https://raw.githubusercontent.com/UptickNetwork/uptick-mainnet/master/uptick_117-1/genesis.json

```

## Start testnet

The final step is to [start the nodes](./../quickstart/run_node#start-node). Once enough voting power (+2/3) from the genesis validators is up-and-running, the testnet will start producing blocks.

```bash
# start the node (you can also use "nohup" or "systemd" to run in the background)
uptickd start
```

::: tip
You may see some connection errors, it does not matter, the P2P network is trying to find available connections

Try to add some of the [Community Peers](https://github.com/UptickNetwork/uptick-mainnet/tree/main/uptick_117-1) to persistent_peers in the config.toml

If you want to quickly start the node and join Uptick without historical data, you can consider using the [state_sync](./../guides/statesync/statesync.md) function.
:::



## Status Sync

To quickly get started, node operators can choose to sync via State Sync. State Sync works by replaying larger chunks of application state directly rather than replaying individual blocks or consensus rounds.

The newest state sync configs can be found [here](https://explorer.uptick.network/uptick-network-mainnet/statesync). Please remember to modify state sync configs.

```bash
# initialize node configurations
uptickd init <moniker> --chain-id uptick_117-1

# download testnel public genesis.json
curl -o $HOME/.uptickd/config/genesis.json https://raw.githubusercontent.com/UptickNetwork/uptick-mainnet/master/uptick_117-1/genesis.json

# Configure State sync
[statesync]
enable = true
rpc_servers = "http://18.138.220.30:26657,http://18.141.43.191:26657"
trust_height = 12000
trust_hash = "dee636061e072ba3e0fee408718b7aff97bd8d4a2a27c695c8d4c8b87081d698"
trust_period = "168h"  # 2/3 of unbonding time

# start the node (you can also use "nohup" or "systemd" to run in the background)
uptickd start
```


## Run a  Validator

Confirm your node has caught-up

```bash
# if you have not installed jq
# apt-get update && apt-get install -y jq

# if the output is false, means your node has caught-up
uptickd status | jq .sync_info.catching_up
```

::: tip
For more details on how to configure your validator, follow the validator [setup](./../guides/validators/setup.md) instructions.
:::
```bash
uptickd tx staking create-validator \
  --amount=5000000000000000000auptick \
  --chain-id=uptick_117-1 \
  --commission-rate="0.10" \
  --commission-max-rate="0.20" \
  --commission-max-change-rate="0.01" \
  --min-self-delegation="1000000" \
  --moniker="moniker" \
  --identity="identity" \
  --website="website" \
  --details="details" \
  --from=<$Validator wallet name> \
  -y -b block
```

## Upgrading Your Node

> NOTE: These instructions are for full nodes that have ran on previous versions of and would like to upgrade to the latest testnet.

### Reset Data

:::warning
If the version <new_version> you are upgrading to is not breaking from the previous one, you **should not** reset the data. If this is the case you can skip to [Restart](#restart)
:::

First, remove the outdated files and reset the data.

```bash
rm $HOME/.uptickd/config/addrbook.json $HOME/.uptickd/config/genesis.json
uptickd tendermint unsafe-reset-all
```

Your node is now in a pristine state while keeping the original `priv_validator.json` and `config.toml`. If you had any sentry nodes or full nodes setup before,
your node will still try to connect to them, but may fail if they haven't also
been upgraded.

::: danger Warning
Make sure that every node has a unique `priv_validator.json`. Do not copy the `priv_validator.json` from an old node to multiple new nodes. Running two nodes with the same `priv_validator.json` will cause you to double sign.
:::

### Restart

To restart your node, just type:

```bash
uptickd start
```

### State Syncing a Node

If you want to join the network using State Sync (quick, but not applicable for archive nodes), check our [State Sync](./../guides/statesync/statesync.md#State-Sync)page
