<!--
order: 1
-->

# Joining a Testnet

This document outlines the steps to join an existing testnet {synopsis}

## Pick a Testnet

You specify the network you want to join by setting the **genesis file** and **seeds**. If you need more information about past networks, check our [testnets repo](https://github.com/UptickNetwork/uptick-testnet).

| Network Chain ID | Description                       | Site                                                                     | Version                                               |
|------------------|-----------------------------------|--------------------------------------------------------------------------|-------------------------------------------------------|
| `uptick_7000-2`   | Uptick Testnet | [uptick_7000-2 testnet](https://github.com/UptickNetwork/uptick-testnet/tree/main/uptick_7000-2) | [`v0.2.x`](https://github.com/UptickNetwork/uptick/releases) |

## Public Endpoints

- GRPC: 52.220.252.160:9090

- RPC: http://52.220.252.160:26657

- REST: http://52.220.252.160:1317

- JSON RPC: http://52.220.252.160:8545

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
uptickd config chain-id uptick_7000-2
```

## Initialize Node

We need to initialize the node to create all the necessary validator and node configuration files:

```bash
# initialize node configurations
uptickd init <your_custom_moniker> --chain-id uptick_7000-2

# download testnel public genesis.json
curl -o $HOME/.uptickd/config/genesis.json https://raw.githubusercontent.com/UptickNetwork/uptick-testnet/main/uptick_7000-2/genesis.json


```

## Start testnet

The final step is to [start the nodes](./../quickstart/run_node#start-node). Once enough voting power (+2/3) from the genesis validators is up-and-running, the testnet will start producing blocks.

```bash
uptickd start
```

##  Install script
```bash
wget https://raw.githubusercontent.com/UptickNetwork/uptick-testnet/main/uptick_7000-2/node.sh && chmod +x node.sh
./node.sh

```

## Status Sync



## Run a Testnet Validator

Claim your testnet {{ $themeConfig.project.testnet_denom }} on the [faucet](./faucet.md) using your validator account address and submit your validator account address:
> NOTE: Until `uptickd status 2>&1 | jq ."SyncInfo"."catching_up"` got false, create your validator. If your validator is jailed, unjail it via `uptickd tx slashing unjail --from <wallet name> --chain-id uptick_7000-2 -y -b block`.

::: tip
For more details on how to configure your validator, follow the validator [setup](./../guides/validators/setup.md) instructions.
:::
```bash
uptickd tx staking create-validator \
  --amount=5000000000000000000auptick \
  --pubkey=$(uptickd tendermint show-validator) \
  --moniker=<$moniker>" \
  --chain-id=uptick_7000-2 \
  --commission-rate="0.10" \
  --commission-max-rate="0.20" \
  --commission-max-change-rate="0.01" \
  --min-self-delegation="1000000" \
  --gas="auto" \
  --from=<$wallet name> \
  -y \
  -b block
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
