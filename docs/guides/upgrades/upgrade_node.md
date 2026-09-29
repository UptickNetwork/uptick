<!--
order: 1
-->

# Upgrade Node

Learn how to upgrade your full node to the latest software version {synopsis}

With every new software release, we strongly recommend validators to perform a software upgrade, in order to prevent [double signing or halting the chain during consensus](https://docs.tendermint.com/master/spec/consensus/signing.html#double-signing).

You can upgrade your node by 1) upgrading your software version and 2) upgrading your node to that version. In this guide, you can find out how to automatically upgrade your node with Cosmovisor or perform the update manually.

## Upgrading to v0.5.0

::: danger
**This release is scoped to mainnet.** The `v0.4.0` and `v0.4.1` plan names are deliberately *not*
registered in the `v0.5.0` binary, so it **cannot be used on the testnet (`origin_1170-3`)**. That
chain's last completed upgrade is `v0.4.1`, and a node whose binary has no handler for it aborts on
**every** start with `upgrade handler is missing for v0.4.1 upgrade plan` — it will not sync, not
catch up, and not participate. Do not point a testnet node at this binary and do not hand it to
anyone joining the testnet. A later release registers `v0.4.1` back.
:::

### Which chain moves in which release

| | current | this release | next release |
|---|---|---|---|
| mainnet `uptick_117-1` | `v0.3.3` | → `v0.5.0` (one plan, one hop) | → `v0.5.1` |
| testnet `origin_1170-3` | `v0.4.1` | **stays on `v0.4.1`** | → `v0.5.1` |

The two chains converge on `v0.5.1`, not here. Testnet keeps running its `v0.4.1` binary until then.

That is a mechanical requirement on the next release, not a preference. x/upgrade's startup
self-check (see *Swapping the binary* below) demands that a binary carry a handler for the last
completed upgrade of every chain it runs on. Once mainnet has executed `v0.5.0`, **`v0.5.0` becomes
mainnet's last completed upgrade**, so the `v0.5.1` binary has to register both `v0.4.1` (testnet) and
`v0.5.0` (mainnet). Getting only one of them right breaks one of the two chains on startup.

`v0.5.1` must also carry this release's erc20 migrations forward: testnet never runs the `v0.5.0`
plan, so the three steps that repair IBC vouchers — the `permissionless_registration` switch, the
`decimals()` metadata rewrite, the token-pair backfill — are the only way those reach it. They are
idempotent, which is what lets `v0.5.1` re-apply them to mainnet as well.

`v0.5.0` is the **one-hop** upgrade from `v0.3.3`: a single governance plan named `v0.5.0` runs the whole `v0.4.0` change set and then the `v0.4.1` repairs, replacing the two-plan sequence the testnet executed. The handler decides which stage to run from the node's **starting state**, not from a flag: it probes for the cosmos/evm `EvmCoinInfo` record (EVM store prefix `0x05`), which the `v0.4.0` handler writes and the legacy ethermint layout cannot, and skips the `v0.4.0` set when that record is present — the set's legacy-pair deletion is destructive to replay.

`v0.5.0` also carries two erc20 migrations that repair IBC vouchers, and turns `permissionless_registration` on:

- `decimals()` on IBC-derived ERC20 is rewritten so wallets, explorers and DEX front-ends stop reading balances 10^18 times too large. Source denoms whose exponent cannot be derived (`u…` → 6, `a…` → 18) are skipped and logged, not guessed.
- Every `ibc/` voucher that holds a supply and has no token pair yet gets a derived pair backfilled.
- `permissionless_registration` is flipped to `true` on both starting states.

::: warning
`v0.5.0` is **state machine breaking and not reversible**: it deletes the `capability` store and moves the EVM `ChainConfig` to time-based activation, exactly as `v0.4.0` did. Every validator must **have the new binary staged** before the upgrade height — but must not be **running** it there. See the next section; starting it early halts the node.
:::

The on-chain upgrade name must be `v0.5.0` **verbatim** — the store loader, including the `capability` store deletion, is keyed on the plan name and nothing else:

```bash
uptickd tx gov submit-proposal software-upgrade v0.5.0 ...
```

When using Cosmovisor, place the new binary under:

```bash
mkdir -p $DAEMON_HOME/cosmovisor/upgrades/v0.5.0/bin
cp $(which uptickd) $DAEMON_HOME/cosmovisor/upgrades/v0.5.0/bin/
```

### Swapping the binary: at the height, never before

x/upgrade does **not** let the new binary run ahead of the plan. On every block where a plan is
pending but not yet due, its `PreBlocker` refuses to start a binary whose handler list already
contains that plan's name:

```
BINARY UPDATED BEFORE TRIGGER! UPGRADE "v0.5.0" - in binary but not executed on chain. Downgrade your binary
```

So once the `v0.5.0` proposal has passed, a node started with the `v0.5.0` binary **halts** until the
upgrade height arrives, and the recovery is to put the old binary back. What this means in practice:

- **Cosmovisor**: stage the binary under `upgrades/v0.5.0/bin` and change nothing else. Cosmovisor
  runs the current binary, sees `UPGRADE "v0.5.0" NEEDED at height …`, and switches at the height —
  which is the only moment a switch is legal. This is the recommended path.
- **Manual**: run the old binary until it halts at the upgrade height with `UPGRADE NEEDED`, then
  replace the binary and restart. Do not replace it earlier, even though the file is already on disk.
- A plan whose name the running binary does **not** know is harmless: that is the normal state of
  every node between proposal and height, and the chain keeps producing blocks.

The same rule applies to the next release, and it is why the testnet's plan must be proposed while
testnet is still running its `v0.4.1` binary.

### What the node must have before the height

- The old binary has to reach the upgrade height once, because that is when it writes
  `data/upgrade-info.json`, which the new binary reads to install the store loader. Cosmovisor
  handles this; a manual operator should let the node halt rather than trying to pre-empt it.
- Run `make check-clock-skew` on each node. A host more than 60s behind the network's median clock
  rejects every block from CometBFT v0.38.22 onward.
- Update the release's version in your deployment tooling **at** the height, not before.

## Upgrading to v0.4.0

*Historical. No chain upgrades through this plan name any more: the testnet already ran `v0.4.0` and `v0.4.1`, and mainnet reaches the same state in one hop via `v0.5.0`, which replays this change set internally. What follows documents what `v0.5.0` replays, and remains the reference for the state a `v0.4.x` chain is in.*

The `v0.4.0` upgrade is a **state machine breaking** upgrade that replaces the core stack:

- EVM engine migrated from the legacy Ethermint `x/evm` to `cosmos/evm` `x/vm` (go-ethereum v1.16).
- `ibc-go` upgraded from v8 to v10; the `capability` module store is **deleted** by the upgrade handler.
- Cosmos SDK upgraded to v0.53 and `wasmd` to v0.61.
- The local `x/erc20` module is replaced by `cosmos/evm`'s `x/erc20`; new `x/erc721` and `x/cw721` modules are added.
- Legacy `EthAccount` types and `ethsecp256k1` pubkeys are migrated automatically.

::: warning
The `v0.4.0` upgrade is **not reversible**. Once the `capability` store is deleted and the EVM
`ChainConfig` is migrated to time-based activation, the chain cannot roll back. Make sure all
validators are upgraded **before** the upgrade height.
:::

When using Cosmovisor, place the new binary under:

```bash
mkdir -p $DAEMON_HOME/cosmovisor/upgrades/v0.4.0/bin
cp $(which uptickd) $DAEMON_HOME/cosmovisor/upgrades/v0.4.0/bin/
```

The on-chain upgrade name was `v0.4.0` (e.g. `uptickd tx gov submit-proposal software-upgrade v0.4.0 ...`). **It is no longer registered**: proposing `v0.4.0` against the current binary halts the upgrade height with `UPGRADE NEEDED` instead of running it. The `v0.5.0` plan replays this change set.

## Upgrading to v0.4.1

*Historical. No chain upgrades through this plan name any more. The testnet ran `v0.4.0` then `v0.4.1` and is now stopped on `v0.4.1`; mainnet reaches the same state in one hop via `v0.5.0`. The `v0.4.1` plan is not registered in the current binary — see the warning on the `v0.5.0` section above. What follows documents the repairs `v0.5.0` replays.*

The `v0.4.1` upgrade is **state-machine compatible with v0.4.0**: it bumps no module
consensus version and performs no store deletion. The one-shot repairs the handler runs are
idempotent and safe to re-apply:

- **Activates the EVM static precompiles** (`ActiveStaticPrecompiles`): v0.4.0 introduced the
  param but left it empty, so every custom precompile (bank, staking, distribution, ICS20, gov,
  slashing, bech32, p256) was inactive. The upgrade fills the list only when it is empty, so
  governance removals are preserved.
- **Enables the ICA controller submodule**: genesis templates derived from legacy `x/params`
  defaults carry `controller_enabled=false`, which blocks every ICA register. Only a
  `false -> true` flip is performed.
- Runtime-only compatibility (no migration): Keplr EIP-712 support (legacy ethermint pubkey and
  `ExtensionOptionsWeb3Tx` type-URL mapping) and the EIP-2 low-s signature check live in the
  binary's codec and ante handler.

### The two-step path from v0.3.x (superseded)

A chain on v0.3.x **cannot** jump straight to a `v0.4.1` plan: the one-shot migrations that make
v0.3.x state readable by the cosmos/evm stack — legacy `EthAccount` → `BaseAccount` rewriting,
legacy `ethsecp256k1` pubkey `Any` migration, `capability` store deletion, EVM `ChainConfig`
Block→Time migration, legacy erc20/params cleanup — only exist in the `v0.4.0` handler.

That is exactly why the one-hop `v0.5.0` plan runs the `v0.4.0` change set itself and then the
`v0.4.1` repairs, instead of asking governance to sequence two plans. Submit `v0.5.0`; do **not**
propose `v0.4.0` or `v0.4.1` on a chain that has not already run them, in any order.

When using Cosmovisor, place the new binary under:

```bash
mkdir -p $DAEMON_HOME/cosmovisor/upgrades/v0.4.1/bin
cp $(which uptickd) $DAEMON_HOME/cosmovisor/upgrades/v0.4.1/bin/
```

The on-chain upgrade name was `v0.4.1` (e.g. `uptickd tx gov submit-proposal software-upgrade v0.4.1 ...`). **It is no longer registered**: proposing `v0.4.1` against the current binary halts the upgrade height with `UPGRADE NEEDED` instead of running it.

### Operator checklist

- **Clock sync**: from CometBFT v0.38.22 the node rejects any block whose header time is at or past its own clock
  plus `consensus.block_time_tolerance` (`1m0s` by default, applied even when the key is absent from
  `config.toml`, and rejected by validation if written as `0`). Block time tracks the network's median clock, so a
  host running more than 60s slow rejects every block and stalls at one height while its log still looks healthy.
  Run `make check-clock-skew` on each node before the upgrade height: it exits 1 when the clock is behind by at
  least the tolerance, and 2 when it could not measure anything — which is not a pass.
- **Export rehearsal**: the v0.4.x `erc721`/`cw721` genesis export fails loudly on inconsistent
  per-token bindings, orphaned refund records or corrupt token pairs instead of silently dropping
  them. Before the upgrade, run `uptickd export` against a state snapshot and confirm it succeeds;
  if it reports orphaned state, fix it **before** the upgrade height.
- `uptickd migrate` performs **no** legacy offline genesis migration — upgrade chain state only
  through the in-app software-upgrade handler above.
- Client-breaking behavior changes shipped with v0.4.1: collection `MsgEditNFT`/`MsgTransferNFT`
  now treat an empty string as "keep the current value" (use the new `[remove]` sentinel to clear
  a field), `MsgIssueDenom` rejects the reserved `uptick-` prefix, and EIP-712 signatures must
  satisfy EIP-2 (low-s). See the changelog for details.

## Software Upgrade

These instructions are for full nodes that have ran on previous versions of and would like to upgrade to the latest testnet.

First, stop your instance of `uptickd`. Next, upgrade the software:

```bash
cd uptick
git fetch --all && git checkout <new_version>
make install
```

::: tip
If you have issues at this step, please check that you have the latest stable version of GO installed.
:::

You will need to ensure that the version installed matches the one needed for th testnet. Check the Uptick [release page](https://github.com/UptickNetwork/uptick/releases) for details on each release.

Verify that everything is OK. If you get something like the following, you've successfully installed Uptick on your system.

```bash
$ uptickd version --long

name: uptick
server_name: uptickd
version: 0.1.0
commit: d477e775a2596701ea215a4570e8ea9669d76edf
build_tags: netgo,ledger
go: go version go1.25.8 darwin/amd64
...
```

If the software version does not match, then please check your $PATH to ensure the correct uptickd is running.

## Upgrade Node

We highly recommend validators use Cosmovisor to run their nodes. This will make low-downtime upgrades smoother, as validators don't have to manually upgrade binaries during the upgrade. Instead users can preinstall new binaries, and cosmovisor will automatically update them based on on-chain Software Upgrade proposals.

You should review the docs for Cosmovisor located [here](https://docs.cosmos.network/master/run-node/cosmovisor.html)

If you choose to use Cosmovisor, please continue with these instructions. If you choose to upgrade your node manually instead, skip to the [the instructions without Cosmovisor](#upgrade-manually)

### Upgrade with Cosmovisor

> `cosmovisor` is a small process manager for Cosmos SDK application binaries that monitors the governance module for incoming chain upgrade proposals. If it sees a proposal that gets approved, cosmovisor can automatically download the new binary, stop the current binary, switch from the old binary to the new one, and finally restart the node with the new binary.

#### Install and Setup

To get started with [Cosmovisor](https://github.com/cosmos/cosmos-sdk/tree/master/cosmovisor) first download it

```bash
go get github.com/cosmos/cosmos-sdk/cosmovisor/cmd/cosmovisor
```

Set up the Cosmovisor environment variables. We recommend setting these in your `.profile` so it is automatically set in every session.

```bash
echo "# Setup Cosmovisor" >> ~/.profile
echo "export DAEMON_NAME=uptickd" >> ~/.profile
echo "export DAEMON_HOME=$HOME/.uptickd" >> ~/.profile
echo 'export PATH="$DAEMON_HOME/cosmovisor/current/bin:$PATH"' >> ~/.profile
source ~/.profile
```

After this, you must make the necessary folders for cosmosvisor in your daemon home directory (~/.uptickd).

```bash
mkdir -p ~/.uptickd/cosmovisor/upgrades
mkdir -p ~/.uptickd/cosmovisor/genesis/bin
cp $(which uptickd) ~/.uptickd/cosmovisor/genesis/bin/

# Verify the setup
# It should return the same version as uptickd
cosmovisor version
```

#### Preparing an Upgrade

Cosmovisor will continually poll the `$DAEMON_HOME/data/upgrade-info.json` for new upgrade instructions. When an upgrade is ready, node operators can download the new binary and place it under `$DAEMON_HOME/cosmovisor/upgrades/<name>/bin` where `<name>` is the URI-encoded name of the upgrade as specified in the upgrade module plan.

It is possible to have Cosmovisor automatically download the new binary. To do this set the following environment variable.

```bash
export DAEMON_ALLOW_DOWNLOAD_BINARIES=true
```

#### Download Genesis File

You can now download the "genesis" file for the chain. It is pre-filled with the entire genesis state and gentxs.

```bash
curl https://raw.githubusercontent.com/UptickNetwork/uptick-testnet/main/origin_1170-3/config/genesis.json > ~/.uptickd/config/genesis.json
```

We recommend using `sha256sum` to check the hash of the genesis.

```bash
cd ~/.uptickd/config
echo "2b5164f4bab00263cb424c3d0aa5c47a707184c6ff288322acc4c7e0c5f6f36f  genesis.json" | sha256sum -c
```

#### Reset Chain Database

There shouldn't be any chain database yet, but in case there is for some reason, you should reset it. This is a good idea especially if you ran `uptickd start` on an old, broken genesis file.

```bash
uptickd tendermint unsafe-reset-all
```

#### Ensure that you have set peers

In `~/.uptickd/config/config.toml` you can set your peers. See the [peers.txt](https://github.com/UptickNetwork/uptick-testnet/blob/main/origin_1170-3/peers.txt) file for a list of up to date peers.

See the [Add persistent peers section](https://docs.uptick.network/testnet/join.html#add-persistent-peers) in our docs for an automated method, but field should look something like a comma separated string of peers (do not copy this, just an example):

```bash
persistent_peers = "5576b0160761fe81ccdf88e06031a01bc8643d51@195.201.108.97:24656,13e850d14610f966de38fc2f925f6dc35c7f4bf4@176.9.60.27:26656,38eb4984f89899a5d8d1f04a79b356f15681bb78@18.169.155.159:26656,59c4351009223b3652674bd5ee4324926a5a11aa@51.15.133.26:26656,3a5a9022c8aa2214a7af26ebbfac49b77e34e5c5@65.108.1.46:26656,4fc0bea2044c9fd1ea8cc987119bb8bdff91aaf3@65.21.246.124:26656,6624238168de05893ca74c2b0270553189810aa7@95.216.100.80:26656,9d247286cd407dc8d07502240245f836e18c0517@149.248.32.208:26656,37d59371f7578101dee74d5a26c86128a229b8bf@194.163.172.168:26656,b607050b4e5b06e52c12fcf2db6930fd0937ef3b@95.217.107.96:26656,7a6bbbb6f6146cb11aebf77039089cd038003964@94.130.54.247:26656"
```

You can share your peer with

```bash
uptickd tendermint show-node-id
```

**Peer Format**: `node-id@ip:port`

**Example**: `3d892cfa787c164aca6723e689176207c1a42025@143.198.224.124:26656`

If you are relying on just seed node and no persistent peers or a low amount of them, please increase the following params in `config.toml`:

```bash
# Maximum number of inbound peers
max_num_inbound_peers = 200

# Maximum number of outbound peers to connect to, excluding persistent peers
max_num_outbound_peers = 100
```

#### Start your node

Now that everything is setup and ready to go, you can start your node.

```bash
cosmovisor start
```

You will need some way to keep the process always running. If you're on linux, you can do this by creating a service.

```bash
sudo tee /etc/systemd/system/uptickd.service > /dev/null <<EOF
[Unit]
Description=Uptick Daemon
After=network-online.target

[Service]
User=$USER
ExecStart=$(which cosmovisor) start
Restart=always
RestartSec=3
LimitNOFILE=infinity

Environment="DAEMON_HOME=$HOME/.uptickd"
Environment="DAEMON_NAME=uptickd"
Environment="DAEMON_ALLOW_DOWNLOAD_BINARIES=false"
Environment="DAEMON_RESTART_AFTER_UPGRADE=true"

[Install]
WantedBy=multi-user.target
EOF
```

Then update and start the node

```bash
sudo -S systemctl daemon-reload
sudo -S systemctl enable uptickd
sudo -S systemctl start uptickd
```

You can check the status with:

```bash
systemctl status uptickd
```

### Upgrade Manually

#### Upgrade Genesis File

:::warning
If the new version you are upgrading to has breaking changes, you will have to [export](#export-state) the state  and [restart](#restart-node) your node.

If it is **not** breaking (eg. from `v0.1.x` to `v0.1.<x+1>`), you can skip to [Restart](#restart-node) after installing the new version.
:::

To upgrade the genesis file, you can either fetch it from a trusted source or export it locally using the `uptickd export` command.

#### Fetch from a Trusted Source

If you are joining an existing testnet, you can fetch the genesis from the appropriate testnet source/repository where the genesis file is hosted.

Save the new genesis as `new_genesis.json`. Then, replace the old `genesis.json` with `new_genesis.json`.

```bash
cd $HOME/.uptickd/config
cp -f genesis.json new_genesis.json
mv new_genesis.json genesis.json
```

#### Export State

Uptick can dump the entire application state to a JSON file. This, besides upgrades, can be
useful for manual analysis of the state at a given height.

Export state with:

```bash
uptickd export > new_genesis.json
```

You can also export state from a particular height (at the end of processing the block of that height):

```bash
uptickd export --height [height] > new_genesis.json
```

If you plan to start a new network for 0 height (i.e genesis) from the exported state, export with the `--for-zero-height` flag:

```bash
uptickd export --height [height] --for-zero-height > new_genesis.json
```

Then, replace the old `genesis.json` with `new_genesis.json`.

```bash
cp -f genesis.json new_genesis.json
mv new_genesis.json genesis.json
```

At this point, you might want to run a script to update the exported genesis into a genesis state that is compatible with your new version.

You can use the `migrate` command to migrate from a given version to the next one (eg: `v0.X.X` to `v1.X.X`):

```bash
uptickd migrate [target-version] [/path/to/genesis.json] --chain-id=<new_chain_id> --genesis-time=<yyyy-mm-ddThh:mm:ssZ>
```

#### Restart Node

To restart your node once the new genesis has been updated, use the `start` command:

```bash
uptickd start
```
