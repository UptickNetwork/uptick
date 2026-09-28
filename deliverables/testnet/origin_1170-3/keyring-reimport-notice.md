# Notice: Re-import your accounts after upgrading to v0.4.x (keyring format change)

## TL;DR

**Your keys are NOT lost.** The v0.4.x binaries (v0.4.0 / v0.4.1) ship with cosmos-sdk v0.53.6, which uses a new local keyring record format. Accounts created with the **v0.3.3** binary (cosmos-sdk v0.50.14) can no longer be read by v0.4.x — they will print errors like:

```
migrate err for key xxx.info: unable to unmarshal item.Data:
Bytes left over in UnmarshalBinaryLengthPrefixed, should read 10 more bytes but have 167
```

or `xxx.info: key not found`.

You only need to **re-import each affected account once** with one of the procedures below. Addresses do not change.

**Not affected:**
- Accounts created (or re-imported) with a v0.4.x binary
- Node consensus keys (`priv_validator_key.json`) — these are separate from the CLI keyring
- Accounts for which you still have the mnemonic/seed phrase

---

## Option A — You have the mnemonic (recommended, 1 minute)

Re-add the account with the **v0.4.x binary**:

```bash
uptickd keys add <key-name> --recover \
  --algo eth_secp256k1 \
  --keyring-backend test
```

> ⚠️ **`--algo eth_secp256k1` is mandatory.** Uptick accounts are eth-style keys. Without this flag the account is restored with a different algorithm and you get a **different address**.

Verify the address matches your old one:

```bash
uptickd keys show <key-name> --keyring-backend test
```

---

## Option B — No mnemonic (export from the old binary, import with the new one)

Requires both binaries on the same machine:
- old: `uptickd-v0.3.3` (from the v0.3.3 release)
- new: `uptickd` v0.4.1 (`v0.4.1-fix` release, commit `cc68b3d24ab7506186f8ec89350df1271a2c3451`)

Run **locally only**. The private key never needs to be printed or shared.

```bash
# 0. Back up the keyring first
cp -a ~/.uptickd/keyring-test ~/.uptickd/keyring-test.bak-$(date +%Y%m%d-%H%M%S)

# 1. Record the current address (must match after migration)
OLD_BIN=uptickd-v0.3.3
NEW_BIN=uptickd
$OLD_BIN keys show <key-name> -a --keyring-backend test

# 2. Export the raw private key (kept in a shell variable, NOT printed)
pk=$($OLD_BIN keys unsafe-export-eth-key <key-name> --keyring-backend test 2>/dev/null | tr -d '\n')

# 3. Sanity check: must be 64 hex characters
echo "$pk" | grep -Eq '^[0-9A-Fa-f]{64}$' && echo "export OK" || { echo "export FAILED"; exit 1; }

# 4. Remove the old-format record
$OLD_BIN keys delete <key-name> --keyring-backend test -y

# 5. Import with the v0.4.x binary
#    (it will read a storage passphrase of >= 8 characters from stdin)
printf 'YourStoragePassphrase\n' | $NEW_BIN keys unsafe-import-eth-key <key-name> "$pk" --keyring-backend test

# 6. Verify: the address MUST be identical to step 1
$NEW_BIN keys show <key-name> -a --keyring-backend test
```

After the import, `keys list` will no longer print `migrate err` lines for that key.

> ⚠️ **Do NOT use `keys export` / `keys import` (armored) to move a key from v0.3.3 to v0.4.x.**
> The armored payload written by v0.3.3 is amino-encoded and v0.4.x only reads protobuf — the import fails with
> `unmarshal to types.PrivKey failed ... unrecognized prefix bytes`. Use Option A or B instead.

> ⚠️ If you usually run with `--keyring-backend os` (the default in `client.toml`), apply the same commands with `--keyring-backend os`, or move to `test`/`file` and re-import there.

---

## Security reminders

- Perform all steps on your own machine; never paste a private key or mnemonic into any chat, ticket, or website.
- The `unsafe-export-eth-key` output is your **raw private key** — keep it only in the shell variable and never write it to a shared file. If you accidentally wrote it anywhere, treat the account as compromised and move funds.
- The passphrase in step 5 only encrypts the local keyring file; it is not your account password.

---

## FAQ

**Q: Will my address / balance change?**
No. Re-importing the same private key yields the same address. Balances are on-chain and unaffected.

**Q: Do validators need to do anything for block signing?**
No. The validator consensus key lives in `priv_validator_key.json` under the node home, not in the CLI keyring. Only CLI-managed accounts (used with `--from ...`) are affected.

**Q: I don't know whether my keyring is affected.**
Run `uptickd keys list --keyring-backend test` with the v0.4.x binary. If any line prints `migrate err ...` or your key is missing while the `.info` file exists in `~/.uptickd/keyring-test/`, it is affected.

---

# 中文版：v0.4.x 升级后需要重新导入账户（keyring 格式变更）

## 一句话

**私钥没有丢。** v0.4.x（v0.4.0/v0.4.1）升级到 cosmos-sdk v0.53.6，本地 keyring 记录格式变更，v0.3.3 创建的账户在 v0.4.x 下会报 `Bytes left over in UnmarshalBinaryLengthPrefixed` / `key not found`。只需按下面对应流程**重新导入一次**，地址不变。

**不受影响**：v0.4.x 下新建的账户；节点共识密钥（`priv_validator_key.json`）；手里还有助记词的账户。

## 方案 A —— 有助记词（推荐）

```bash
uptickd keys add <key-name> --recover --algo eth_secp256k1 --keyring-backend test
```

> ⚠️ 必须带 `--algo eth_secp256k1`，否则恢复出的地址不一致。

## 方案 B —— 没有助记词（旧二进制导出 → 新二进制导入）

需要同机保留 v0.3.3 与 v0.4.1 两个二进制；全程本地操作，私钥不落盘、不外传：

```bash
# 0. 先备份 keyring
cp -a ~/.uptickd/keyring-test ~/.uptickd/keyring-test.bak-$(date +%Y%m%d-%H%M%S)

# 1. 记录当前地址（迁移后必须一致）
OLD_BIN=uptickd-v0.3.3
NEW_BIN=uptickd
$OLD_BIN keys show <key-name> -a --keyring-backend test

# 2. 导出私钥 hex（只存进 shell 变量，不打印）
pk=$($OLD_BIN keys unsafe-export-eth-key <key-name> --keyring-backend test 2>/dev/null | tr -d '\n')

# 3. 自检：必须是 64 位 hex
echo "$pk" | grep -Eq '^[0-9A-Fa-f]{64}$' && echo "export OK" || { echo "export FAILED"; exit 1; }

# 4. 删除旧格式条目
$OLD_BIN keys delete <key-name> --keyring-backend test -y

# 5. 用 v0.4.x 导入（stdin 喂入本地存储 passphrase，>=8 位）
printf 'YourStoragePassphrase\n' | $NEW_BIN keys unsafe-import-eth-key <key-name> "$pk" --keyring-backend test

# 6. 验证地址与第 1 步一致
$NEW_BIN keys show <key-name> -a --keyring-backend test
```

> ⚠️ **不要用 `keys export`/`keys import`（armored）跨 v0.3.3→v0.4.x 迁移**：内层编码格式不兼容（amino vs protobuf），必然报 `unrecognized prefix bytes` 失败。
> ⚠️ 默认 `client.toml` 是 `--keyring-backend os`（macOS 钥匙串）的用户，把上述命令的 backend 换成 `os` 执行即可。

## 安全提醒

- 全部操作在本机完成；私钥/助记词绝不发到任何聊天、工单、网页。
- `unsafe-export-eth-key` 的输出就是**裸私钥**：只允许留在 shell 变量里，若意外写到任何文件/粘贴到任何地方，请视为该账户已泄露并立即转移资产。
