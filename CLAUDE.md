# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Uptick Network is a Cosmos SDK-based blockchain network designed for NFTs and RWAs (Real World Assets). It integrates:
- **Cosmos SDK v0.53.6** - Core blockchain framework
- **CometBFT v0.38.21** - Consensus engine (formerly Tendermint)
- **ibc-go v10.5.0** - Inter-Blockchain Communication protocol
- **cosmos/evm v0.6.2** - Official EVM integration for Ethereum compatibility
- **wasmd v0.61.14 / wasmvm v3** - WebAssembly smart contracts

The chain is fully interoperable with both EVM and IBC chains, supporting cross-chain NFT and token transfers.

## Build and Development Commands

### Building
```bash
# Build the uptickd binary
make build

# Install uptickd to $GOPATH/bin
make install

# Build for Linux
make build-linux

# Build Docker image
make build-docker
```

### Testing
```bash
# Run all unit tests
make test

# Run unit tests with coverage
make test-unit-cover

# Run race condition tests
make test-race

# Run RPC tests
make test-rpc

# Run simulation tests
make test-sim-nondeterminism
make test-sim-multi-seed-short
```

### Linting and Formatting
```bash
# Run linter (golangci-lint)
make lint

# Auto-fix linting issues
make lint-fix

# Format Go code (gofmt, misspell, goimports)
make format
```

**Important**: Always run `make format` before committing. The CI requires `make lint` to pass.

### Protobuf
```bash
# Generate protobuf code
make proto-gen

# Format proto files
make proto-format

# Lint proto files
make proto-lint

# Run all proto commands (format, lint, gen)
make proto-all
```

All protobuf operations run in Docker containers for deterministic builds.

### Local Testnet
```bash
# Start a 4-node local testnet
make localnet-start

# Stop the testnet
make localnet-stop

# Clean testnet data
make localnet-clean
```

### Documentation
```bash
# Serve docs locally at localhost:8080
make docs-serve

# Build docs site
make build-docs
```

## Architecture

### Directory Structure

- **`app/`** - Main application setup and configuration
  - `app.go` - Core application initialization, module wiring
  - `ante/` - Ante handlers for transaction preprocessing
  - `keepers/` - Keeper initialization and dependency injection
  - `upgrades/` - Chain upgrade handlers

- **`x/`** - Custom Cosmos SDK modules
  - `collection/` - NFT collection management (Cosmos-native)
  - `erc721/` - Bidirectional ERC721 ↔ Cosmos NFT conversion
  - `cw721/` - Bidirectional CW721 ↔ Cosmos NFT conversion
  - `evmibc/` - IBC integration for EVM/CW721 NFT conversion
  - `internft/` - Internal NFT module
  - `nft/` - NFT module wrapper

- **`cmd/uptickd/`** - CLI binary entry point

- **`proto/uptick/`** - Protobuf definitions for custom modules

- **`contracts/`** - Solidity smart contracts for EVM integration

- **`ibc/`** - IBC-related configurations

- **`testutil/`** - Testing utilities and helpers

### Key Architectural Patterns

**Module Structure**: Each custom module in `x/` follows the standard Cosmos SDK module pattern:
- `keeper/` - State management and business logic
- `types/` - Message types, keys, and codec registration
- `client/cli/` - CLI commands
- `spec/` - Module specification and documentation

**EVM Integration**: The chain uses the official `cosmos/evm` module to run the EVM alongside the Cosmos SDK. The Cosmos SDK `x/erc20` integration enables token conversion between EVM (ERC20) and Cosmos bank-module representations.

**NFT Architecture**: Uptick supports multiple NFT standards:
- Cosmos-native NFTs via the `collection` module
- EVM NFTs (ERC721) via cosmos/evm
- CosmWasm NFTs (CW721) via wasmd
- Cross-chain NFT transfers via IBC and the `nft-transfer` module

**Protobuf Code Generation**: All state types, messages, and queries are defined in `.proto` files under `proto/uptick/`. Generated Go code lives alongside the proto definitions.

## Development Workflow

### Branching
- Base all work on the `development` branch (not `main`)
- Branch naming: `{moniker}/{issue#}-branch-name` for core developers
- Target PRs to `development`

### Before Submitting a PR
1. Run `make format` to format code
2. Run `make lint` to check for linting errors
3. Run `make test` to ensure tests pass
4. Update relevant documentation in `/docs`
5. Add a changelog entry to `CHANGELOG.md` under "Unreleased"

### Testing Requirements
- `development` must always pass: `make lint`, `make test`, `make test-race`, `make test-rpc`
- PRs require two approvals before merge
- Submit PRs in Draft mode initially

### Dependencies
- Managed via Go modules (`go.mod`)
- Run `go mod tidy` if dependencies are modified
- Several key dependencies use custom forks (see `replace` directives in `go.mod`)

## Important Notes

- **Go Version**: Requires Go 1.25.8+
- **Main Binary**: `uptickd` (not `uptick`)
- **Protobuf Path**: For IDE support, configure protobuf paths to include `proto/` and `third_party/proto/`
- **Docker Required**: For protobuf generation and reproducible builds
- **No Force Push**: Never force push to `development`
