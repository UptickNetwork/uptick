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
# compile each .sol with solc 0.8.28, optimizer OFF, default (ipfs) metadata
# hash, and emit both bytecode (bin) and deployed-bytecode (bin-runtime) into
# compiled_contracts/.
make verify-contracts-reproducible   # rebuild; compare vs refs/ AND vs the committed
                                     # compiled_contracts/*.json (audit P3-14)
make verify-contracts                # compare committed artifacts vs checksums.txt
                                     # and the embedded x/erc721 copy
```

Every artifact in `compiled_contracts/` is listed in `checksums.txt`. CI runs
both gates, so a dependency bump or a non-reproducible build cannot silently
change on-chain contract bytecode, and the committed JSONs cannot drift from
what the pinned toolchain produces from the committed sources.

## Known trade-offs (audit P3-11)

`ERC721Uptick.mintBatch` / `mintEnhanceBatch` de-duplicate token ids with an
O(n²) pairwise comparison. This is a deliberate trade-off: `MAX_BATCH_SIZE`
bounds n at 100, so the worst case is ~5,050 comparisons — negligible against
the gas cost of the mints themselves — while keeping the check exact. Do not
"optimize" this in place: ANY byte change to the `.sol` sources (even comments)
changes the solc metadata hash and breaks the reproducible-build gate
(`checksums.txt` / `refs/`). If the check ever needs replacing, regenerate all
committed artifacts in the same change and update both gates.
