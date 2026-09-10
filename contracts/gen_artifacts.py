#!/usr/bin/env python3
"""Regenerate compiled_contracts/*.json from the pinned solc + OpenZeppelin.

The committed artifacts are a flat projection of `solc --combined-json`:

    {"abi": "<abi array as a JSON string>", "bin": "<hex>",
     "bin-runtime": "<hex>", "contractName": "<name>"}

`bin` is what the Go module embeds and deploys (x/erc721/contracts/erc721.go);
`bin-runtime` and `contractName` exist so verify_contracts.sh can compare the
committed JSON against a fresh rebuild field by field. The format is pinned
because verify-contracts compares bytes, not semantics.

Usage:
    ./gen_artifacts.py            # rewrite compiled_contracts/*.json in place
    ./gen_artifacts.py --check    # exit 1 if any artifact is stale

The toolchain must match verify_contracts.sh: solc 0.8.28 and
@openzeppelin/contracts 4.9.6 (see package.json / package-lock.json).
"""

import argparse
import json
import os
import shutil
import subprocess
import sys

SOLC_VERSION = "0.8.28"
ROOT = os.path.dirname(os.path.abspath(__file__))
OUT_DIR = os.path.join(ROOT, "compiled_contracts")


def find_solc() -> str:
    """Prefer an explicitly pinned compiler, fall back to PATH."""
    for candidate in (os.environ.get("SOLC"), "solc"):
        if not candidate:
            continue
        path = shutil.which(candidate) if os.sep not in candidate else candidate
        if path and os.access(path, os.X_OK):
            return path
    sys.exit("error: solc not found; install solc 0.8.28 or set $SOLC")


def check_version(solc: str) -> None:
    out = subprocess.run([solc, "--version"], capture_output=True, text=True).stdout
    found = out.split("Version: ")[-1].split("+")[0].strip()
    if found != SOLC_VERSION:
        sys.exit(f"error: solc {SOLC_VERSION} required, found {found}")


def compile_one(solc: str, name: str) -> dict:
    proc = subprocess.run(
        [
            solc,
            "--combined-json", "abi,bin,bin-runtime",
            "--base-path", ".",
            "--include-path", "node_modules",
            f"{name}.sol",
        ],
        cwd=ROOT,
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        sys.exit(f"error: solc failed for {name}.sol:\n{proc.stderr}")

    combined = json.loads(proc.stdout)["contracts"]
    matches = [k for k in combined if k.endswith(":" + name)]
    if len(matches) != 1:
        sys.exit(f"error: expected exactly one {name} in solc output, got {matches}")

    entry = combined[matches[0]]
    # The ABI is embedded as a JSON *string* to match the committed format that
    # verify-contracts.sh compares byte-for-byte.
    return {
        "abi": json.dumps(entry["abi"], separators=(",", ":")),
        "bin": entry["bin"],
        "bin-runtime": entry["bin-runtime"],
        "contractName": name,
    }


def render(artifact: dict) -> str:
    return json.dumps(artifact, separators=(",", ":"))


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true",
                        help="do not write; exit 1 if a committed artifact is stale")
    args = parser.parse_args()

    if not os.path.isdir(os.path.join(ROOT, "node_modules", "@openzeppelin", "contracts")):
        sys.exit("error: @openzeppelin/contracts not installed; run 'npm ci' in contracts/")

    solc = find_solc()
    check_version(solc)

    sources = sorted(f[:-4] for f in os.listdir(ROOT) if f.endswith(".sol"))
    if not sources:
        sys.exit("error: no *.sol sources found")

    stale = []
    for name in sources:
        wanted = render(compile_one(solc, name))
        target = os.path.join(OUT_DIR, f"{name}.json")
        current = open(target).read() if os.path.exists(target) else None
        if current == wanted:
            print(f"ok: {name}.json is up to date")
            continue
        if args.check:
            stale.append(name)
            print(f"STALE: {name}.json differs from a rebuild", file=sys.stderr)
            continue
        os.makedirs(OUT_DIR, exist_ok=True)
        with open(target, "w") as fh:
            fh.write(wanted)
        print(f"wrote compiled_contracts/{name}.json")

    # An artifact with no root .sol cannot be rebuilt or reviewed; fail loudly
    # instead of shipping an unreproducible byte blob (audit O-05).
    orphans = sorted(
        f[:-5] for f in os.listdir(OUT_DIR)
        if f.endswith(".json") and f[:-5] not in sources
    )
    if orphans:
        sys.exit(
            "error: compiled_contracts/*.json without a matching root .sol source: "
            + ", ".join(orphans)
        )

    if stale:
        sys.exit(f"error: stale artifacts: {', '.join(stale)} (run ./gen_artifacts.py)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
