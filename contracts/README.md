# Uptick ERC20 / ERC721 contracts

These contracts back the `x/erc721` conversion module and the on-chain ERC20
wrappers. The only runtime-consumed artifact is
`compiled_contracts/ERC721Uptick.json` (embedded by `x/erc721/contracts`).

## Reproducible build

The artifacts pinned in this directory were compiled with:

- `solc` **0.8.28**
- `@openzeppelin/contracts` **4.9.6** (see `package.json` / `package-lock.json`)

To rebuild deterministically:

```bash
cd contracts
npm ci
# compile each .sol with solc 0.8.28, optimizer enabled, and emit both bytecode
# (bin) and deployed-bytecode (bin-runtime) into compiled_contracts/.
make verify-contracts   # rebuild and compare against checksums.txt
```

Every artifact in `compiled_contracts/` is listed in `checksums.txt`. CI must
rerun the build and fail if any checksum differs, so a dependency bump or a
non-reproducible build cannot silently change on-chain contract bytecode.
