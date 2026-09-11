package app

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	storetypes "cosmossdk.io/store/types"
)

// This file closes the wasmd/wasmvm half of the migration review, which until
// now could only be argued statically.
//
// The v0.4.0 upgrade moves wasmd v0.53.3 -> v0.61.14 and wasmvm v2.1.5 -> v3.
// The store layout is unchanged (types/keys.go prefixes are byte-identical and
// the wasm module consensus version is 4 in both releases, so no module
// migration is required), but a VM major bump is exactly the kind of change
// that can quietly strand already-instantiated contracts: the bytecode lives on
// chain and only the node's VM is replaced under it.
//
// Every other wasm-facing test in this repository drives a fake keeper that
// returns canned ABI values (x/cw721/keeper/convert_test.go: fakeWasmKeeper), so
// none of them would notice a VM that refuses the shipped contract. This test
// instead pushes the real contract the node ships - release/wasm/cw721_base.wasm,
// the cw721-base artifact the old wasm-nft-convert deployment uses - through the
// application's real wasmvm v3 instance and drives all four entry points the
// migration could break:
//
//	Create      the VM parses the module and enforces its declared VM interface
//	Instantiate runs the contract's instantiate entry point
//	QuerySmart  runs the contract's query entry point
//	Execute     runs the contract's execute entry point and mutates its state
//
// The contract declares interface_version_8 (CosmWasm 1.0+); if a future bump
// raises the minimum accepted interface, the Create call is where it shows up.
func TestWasmVMv3RunsTheShippedCW721Contract(t *testing.T) {
	app, baseCtx := sharedTestApp(t)

	// Work inside a cache so the code id and the contract instance this test
	// creates cannot leak into the other suites that share the singleton app.
	ctx, _ := baseCtx.CacheContext()
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	wasmBytes, err := os.ReadFile(filepath.Join("..", "release", "wasm", "cw721_base.wasm"))
	require.NoError(t, err, "the node ships release/wasm/cw721_base.wasm; the test must be able to read it")
	require.Greater(t, len(wasmBytes), 1<<10, "a cw721 contract is far bigger than this")
	require.Equal(t, []byte{0x00, 0x61, 0x73, 0x6d}, wasmBytes[:4], "the fixture is not a wasm module")

	// The default policy enforces the chain's upload access parameter, which
	// wasmd's default genesis leaves at Everybody - the same setting the live
	// chain uses. If that ever changes, this test fails at Create with an
	// authorization error rather than silently proving nothing.
	creator := sdk.AccAddress(bytes.Repeat([]byte{0x7a}, 20))
	pk := wasmkeeper.NewDefaultPermissionKeeper(app.WasmKeeper)

	codeID, checksum, err := pk.Create(ctx, creator, wasmBytes, &wasmtypes.AllowEverybody)
	require.NoError(t, err, "wasmvm v3 must accept a contract declaring interface_version_8")
	require.NotZero(t, codeID)
	wantChecksum := sha256.Sum256(wasmBytes)
	require.Equal(t, wantChecksum[:], checksum,
		"the stored code checksum must still be sha256(wasm bytes); a change here would strand every code id already on chain")

	initMsg := []byte(`{"name":"Uptick Compatible","symbol":"UCT","minter":"` + creator.String() + `"}`)
	contract, _, err := pk.Instantiate(ctx, codeID, creator, nil, initMsg, "cw721-base-compat", sdk.NewCoins())
	require.NoError(t, err, "the instantiate entry point must run under wasmvm v3")
	require.NotEmpty(t, contract)

	got, err := app.WasmKeeper.QuerySmart(ctx, contract, []byte(`{"contract_info":{}}`))
	require.NoError(t, err, "the query entry point must run under wasmvm v3")
	require.Contains(t, string(got), "Uptick Compatible")

	// Execute is the entry point the conversion paths depend on: x/cw721 mints
	// and burns through MsgExecuteContract on the bound contract.
	execMsg := []byte(`{"mint":{"token_id":"nft1","owner":"` + creator.String() + `","token_uri":"https://example.com/nft1"}}`)
	_, err = pk.Execute(ctx, contract, creator, execMsg, sdk.NewCoins())
	require.NoError(t, err, "the execute entry point must run under wasmvm v3")

	// Reading the mutation back proves the contract could write to its own
	// state through the new VM, not just answer a query from its init state.
	got, err = app.WasmKeeper.QuerySmart(ctx, contract, []byte(`{"owner_of":{"token_id":"nft1"}}`))
	require.NoError(t, err)
	require.Contains(t, string(got), creator.String(), "the token minted in this block must be readable afterwards")

	// Pin the fixture's identity. The compatibility claim above is only as good
	// as the artifact it was made against: a cosmwasm build declares the VM
	// interface it targets, and this one targets the first stable one.
	require.Equal(t, 1, bytes.Count(wasmBytes, []byte(interfaceVersionExport8)),
		"release/wasm/cw721_base.wasm changed; re-verify the interface version it declares")
}

// interfaceVersionExport8 is the marker a contract exports to declare the VM
// interface it was built against. CosmWasm 1.0 introduced it and every later VM
// keeps accepting it, which is precisely why the on-chain contracts survive the
// wasmvm v2 -> v3 bump. The tests below pin both halves of that sentence.
const interfaceVersionExport8 = "interface_version_8"

// TestWasmVMCreateEnforcesTheDeclaredInterfaceVersion is the negative control
// for the test above. Without it, a VM that had stopped checking the interface
// version would pass the positive case for the wrong reason, and the suite would
// be silent exactly when it matters.
//
// The mutation is a same-length rename inside the export section, so the module
// stays structurally valid and the only thing the VM can object to is the
// version number itself.
func TestWasmVMCreateEnforcesTheDeclaredInterfaceVersion(t *testing.T) {
	app, baseCtx := sharedTestApp(t)

	ctx, _ := baseCtx.CacheContext()
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	wasmBytes, err := os.ReadFile(filepath.Join("..", "release", "wasm", "cw721_base.wasm"))
	require.NoError(t, err)

	downgraded := bytes.Replace(wasmBytes, []byte(interfaceVersionExport8), []byte("interface_version_7"), 1)
	require.NotEqual(t, wasmBytes, downgraded, "the fixture must still export the marker this test rewrites")

	creator := sdk.AccAddress(bytes.Repeat([]byte{0x7b}, 20))
	pk := wasmkeeper.NewDefaultPermissionKeeper(app.WasmKeeper)

	_, _, err = pk.Create(ctx, creator, downgraded, &wasmtypes.AllowEverybody)
	require.Error(t, err, "a contract declaring an interface version the VM does not support must be rejected")
	// The upstream wording is worth recording because it settles the question
	// the migration review could not answer statically. The VM does not compare
	// version numbers, it matches the exported marker against a whitelist:
	//
	//	Error during static Wasm validation: Wasm contract has unknown
	//	interface_version_* marker export
	//
	// That whitelist is identical on both sides of the upgrade - cosmwasm
	// 2.2.0 (wasmvm v2) and 3.0.7 (wasmvm v3) both declare
	// SUPPORTED_INTERFACE_VERSIONS = ["interface_version_8"] - so a contract
	// that the old node accepted cannot be rejected by the new one. A contract
	// on a pre-1.0 marker was already unrunnable before the upgrade and is not
	// a regression of it.
	require.Contains(t, err.Error(), "interface_version",
		"the rejection must come from the interface-version gate, not from an unrelated parse failure")
}

// TestWasmVMCreateRejectsACorruptedModule is the second half of the negative
// control: it proves the Create call really validates the module rather than
// accepting anything it is handed.
func TestWasmVMCreateRejectsACorruptedModule(t *testing.T) {
	app, baseCtx := sharedTestApp(t)

	ctx, _ := baseCtx.CacheContext()
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	wasmBytes, err := os.ReadFile(filepath.Join("..", "release", "wasm", "cw721_base.wasm"))
	require.NoError(t, err)

	corrupted := bytes.Clone(wasmBytes)
	// The code section lives past the header and the type/import/function
	// sections; flipping a byte in the middle of a 250 kB module lands inside it
	// without having to parse the binary.
	corrupted[len(corrupted)/2] ^= 0xff

	creator := sdk.AccAddress(bytes.Repeat([]byte{0x7c}, 20))
	pk := wasmkeeper.NewDefaultPermissionKeeper(app.WasmKeeper)

	_, _, err = pk.Create(ctx, creator, corrupted, &wasmtypes.AllowEverybody)
	require.Error(t, err, "the VM must reject a module that no longer validates")
}
