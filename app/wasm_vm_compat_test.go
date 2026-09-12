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

// This file covers the wasmd/wasmvm half of the migration review: the v0.4.0
// upgrade moves wasmd v0.53.3 -> v0.61.14 and wasmvm v2.1.5 -> v3.
//
// The store layout is unchanged (types/keys.go prefixes are byte-identical and
// the wasm module consensus version is 4 in both releases), but a VM major bump
// can quietly strand already-instantiated contracts: the bytecode lives on chain
// and only the node's VM is replaced under it.
//
// The other wasm-facing tests drive a fake keeper returning canned ABI values
// (x/cw721/keeper/convert_test.go: fakeWasmKeeper) and would not notice a VM that
// refuses the shipped contract. This test pushes the real one the node ships -
// release/wasm/cw721_base.wasm, the artifact the conversion flow uses - through
// the application's real wasmvm v3 instance, exercising the four entry points the
// migration could break: Create (parses the module and enforces its declared VM
// interface), Instantiate, QuerySmart, and Execute (which also mutates state).
//
// The contract declares interface_version_8 (CosmWasm 1.0+); a future bump that
// raises the minimum accepted interface shows up at Create.
func TestWasmVMv3RunsTheShippedCW721Contract(t *testing.T) {
	app, baseCtx := sharedTestApp(t)

	// Work inside a cache so the code id and contract instance cannot leak into
	// the other suites sharing the singleton app.
	ctx, _ := baseCtx.CacheContext()
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	wasmBytes, err := os.ReadFile(filepath.Join("..", "release", "wasm", "cw721_base.wasm"))
	require.NoError(t, err, "the node ships release/wasm/cw721_base.wasm; the test must be able to read it")
	require.Greater(t, len(wasmBytes), 1<<10, "a cw721 contract is far bigger than this")
	require.Equal(t, []byte{0x00, 0x61, 0x73, 0x6d}, wasmBytes[:4], "the fixture is not a wasm module")

	// The default policy enforces the chain's upload access parameter, which
	// wasmd's default genesis leaves at Everybody: if that changes, Create fails
	// with an authorization error rather than the test silently proving nothing.
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

	// Execute is the entry point the conversion paths depend on: x/cw721 mints and
	// burns through MsgExecuteContract on the bound contract.
	execMsg := []byte(`{"mint":{"token_id":"nft1","owner":"` + creator.String() + `","token_uri":"https://example.com/nft1"}}`)
	_, err = pk.Execute(ctx, contract, creator, execMsg, sdk.NewCoins())
	require.NoError(t, err, "the execute entry point must run under wasmvm v3")

	// Reading the mutation back proves the contract wrote to its own state, not
	// merely answered a query from its init state.
	got, err = app.WasmKeeper.QuerySmart(ctx, contract, []byte(`{"owner_of":{"token_id":"nft1"}}`))
	require.NoError(t, err)
	require.Contains(t, string(got), creator.String(), "the token minted in this block must be readable afterwards")

	// Pin the fixture's identity: the compatibility claim above is only as good as
	// the artifact it was made against.
	require.Equal(t, 1, bytes.Count(wasmBytes, []byte(interfaceVersionExport8)),
		"release/wasm/cw721_base.wasm changed; re-verify the interface version it declares")
}

// interfaceVersionExport8 is the marker a contract exports to declare the VM
// interface it was built against. CosmWasm 1.0 introduced it and later VMs keep
// accepting it, which is why the on-chain contracts survive the wasmvm v2 -> v3
// bump.
const interfaceVersionExport8 = "interface_version_8"

// TestWasmVMCreateEnforcesTheDeclaredInterfaceVersion is the negative control
// for the test above: without it, a VM that stopped checking the interface
// version would pass the positive case for the wrong reason.
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
	// The VM does not compare version numbers; it matches the exported marker
	// against a fixed supported set, which the tests above and below pin for wasmvm
	// v3 (interface_version_8 accepted, 7 rejected):
	//
	//	Error during static Wasm validation: Wasm contract has unknown
	//	interface_version_* marker export
	//
	// Pinned on the two sides of the bump: cosmwasm-vm 2.1.6 (wasmvm v2.1.5) and
	// 3.0.9 (wasmvm v3.0.7). A pre-1.0 marker was already unrunnable before the
	// upgrade and is not a regression of it.
	require.Contains(t, err.Error(), "interface_version",
		"the rejection must come from the interface-version gate, not from an unrelated parse failure")
}

// TestWasmVMCreateRejectsACorruptedModule is the second half of the negative
// control: it proves Create validates the module rather than accepting anything
// it is handed.
func TestWasmVMCreateRejectsACorruptedModule(t *testing.T) {
	app, baseCtx := sharedTestApp(t)

	ctx, _ := baseCtx.CacheContext()
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	wasmBytes, err := os.ReadFile(filepath.Join("..", "release", "wasm", "cw721_base.wasm"))
	require.NoError(t, err)

	corrupted := bytes.Clone(wasmBytes)
	// The code section lives past the header and the type/import/function
	// sections; flipping a byte in the middle of the module lands inside it
	// without having to parse the binary.
	corrupted[len(corrupted)/2] ^= 0xff

	creator := sdk.AccAddress(bytes.Repeat([]byte{0x7c}, 20))
	pk := wasmkeeper.NewDefaultPermissionKeeper(app.WasmKeeper)

	_, _, err = pk.Create(ctx, creator, corrupted, &wasmtypes.AllowEverybody)
	require.Error(t, err, "the VM must reject a module that no longer validates")
}
