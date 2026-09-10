package app

import (
	"encoding/hex"
	"math/big"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	erc721contracts "github.com/UptickNetwork/uptick/x/erc721/contracts"
	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
)

// This file is the real-EVM half of the ERC721 audit response. The other ERC721
// suites drive mocks that return canned ABI values, which cannot tell whether
// the contract actually behaves as reviewed. Here the embedded ERC721Uptick
// bytecode (the artifact the node ships, rebuilt from contracts/ERC721Uptick.sol
// by contracts/gen_artifacts.py) is deployed into the app's real in-memory EVM
// and driven through the same keeper entry points the Cosmos -> ERC721
// conversion uses.
//
// Covered findings:
//
//	G-07  mint/mintEnhance/mintBatch must call the IERC721Receiver hook, reject
//	      a receiver that cannot take an ERC721, roll back on a reverting or
//	      gas-burning hook, and be all-or-nothing for a batch.
//	G-08  every burn entry point must clear the project's per-token enhance
//	      metadata, so a re-minted id never inherits a previous lifetime's data.
//	O-04  the suite itself: real bytecode deployment in a real EVM instead of an
//	      ABI-shape mock.

// receiverFixtureABIJSON exposes the getters of the hostile/observing receiver
// contracts in contracts/test/EVMReceiverFixtures.sol. Those fixtures are test
// only: nothing here is embedded by the node.
const receiverFixtureABIJSON = `[
 {"inputs":[],"name":"received","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
 {"inputs":[],"name":"lastOperator","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
 {"inputs":[],"name":"lastFrom","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
 {"inputs":[],"name":"lastTokenId","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
 {"inputs":[],"name":"lastDataLength","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
 {"inputs":[],"name":"observedOwner","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
 {"inputs":[],"name":"observedBalance","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
 {"inputs":[],"name":"observedTotalSupply","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
 {"inputs":[],"name":"observedUri","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},
 {"inputs":[],"name":"observedEnhanceName","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},
 {"inputs":[],"name":"observedEnhanceData","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},
 {"inputs":[],"name":"observedEnhanceUriHash","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},
 {"inputs":[],"name":"reentered","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
]`

var receiverFixtureABI = func() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(receiverFixtureABIJSON))
	if err != nil {
		panic(err)
	}
	return parsed
}()

// erc721EVMHarness is a live application over its real EVM. Building the
// application is the expensive part, so one harness serves every scenario;
// scenario() clones it with a freshly deployed ERC721 contract so no scenario
// can observe another's tokens.
type erc721EVMHarness struct {
	app      *Uptick
	ctx      sdk.Context
	abi      abi.ABI
	contract common.Address
}

// cosmos/evm seals its global coin/chain configuration with a package-level
// once: x/vm.SetGlobalConfigVariables panics with "EVM coin info already set"
// on the second call, so exactly one application may be initialised per
// process (see shared_testapp_test.go, which owns that instance). That is also
// why the multi-application simulation suite re-executes itself as child
// processes instead of building two apps in one test binary. This harness is
// therefore created once and shared.
var (
	erc721EVMOnce   sync.Once
	erc721EVMShared *erc721EVMHarness
)

func newERC721EVMHarness(t *testing.T) *erc721EVMHarness {
	t.Helper()

	erc721EVMOnce.Do(func() {
		erc721EVMShared = buildERC721EVMHarness(t)
	})
	require.NotNil(t, erc721EVMShared, "the shared ERC721 EVM harness failed to initialise")
	return erc721EVMShared
}

func buildERC721EVMHarness(t *testing.T) *erc721EVMHarness {
	t.Helper()

	app, ctx := sharedTestApp(t)
	return &erc721EVMHarness{
		app: app,
		abi: erc721contracts.ERC721UpticksContract.ABI,
		ctx: ctx,
	}
}

// scenario returns a harness whose ERC721 contract is freshly deployed, so the
// scenario starts from an empty collection with its own token namespace.
func (h *erc721EVMHarness) scenario(t *testing.T) *erc721EVMHarness {
	t.Helper()

	clone := *h
	require.NotEqual(t, common.Address{}, clone.deployERC721(t))
	return &clone
}

// deployERC721 deploys the embedded artifact for the seeded collection class.
// The address is derived the same way the keeper derives it, so a mismatch
// would surface as "no code at address" on the first call.
func (h *erc721EVMHarness) deployERC721(t *testing.T) common.Address {
	t.Helper()

	contract, err := h.app.Erc721Keeper.DeployERC721Contract(h.ctx, &erc721types.MsgConvertNFT{
		ClassId: simCollectionDenomID,
	})
	require.NoError(t, err, "deploying the embedded ERC721Uptick bytecode failed")
	require.NotEqual(t, common.Address{}, contract)

	// Reading the class metadata proves the bytecode is really on chain and the
	// ABI wiring works before any behaviour assertion runs.
	data, err := h.app.Erc721Keeper.QueryERC721(h.ctx, contract)
	require.NoError(t, err, "querying the freshly deployed contract failed")
	require.Equal(t, "Simulation Collection", data.Name)
	require.Equal(t, "SIM", data.Symbol)

	h.contract = contract
	return contract
}

// deployFixture deploys a test-only receiver contract, optionally passing the
// ERC721 address to its constructor.
func (h *erc721EVMHarness) deployFixture(t *testing.T, name string, ctorTarget *common.Address) common.Address {
	t.Helper()

	initCodeHex, ok := erc721ReceiverFixtures[name]
	require.Truef(t, ok, "unknown receiver fixture %q", name)
	initCode, err := hex.DecodeString(strings.TrimPrefix(initCodeHex, "0x"))
	require.NoError(t, err)

	if ctorTarget != nil {
		// The fixture constructors take one address; a single statically sized
		// argument is exactly 32 bytes of left-padded ABI encoding.
		initCode = append(initCode, common.LeftPadBytes(ctorTarget.Bytes(), 32)...)
	}

	nonce, err := h.app.AccountKeeper.GetSequence(h.ctx, erc721types.ModuleAddress.Bytes())
	require.NoError(t, err)
	addr := crypto.CreateAddress(erc721types.ModuleAddress, nonce)

	_, err = h.app.Erc721Keeper.CallEVMWithData(h.ctx, erc721types.ModuleAddress, nil, initCode, true)
	require.NoError(t, err, "deploying fixture %s failed", name)

	return addr
}

// call sends a state-changing call to the harness's ERC721 contract from the
// module account (the contract's minter/admin), the same sender the Cosmos ->
// ERC721 conversion path uses.
func (h *erc721EVMHarness) call(method string, args ...interface{}) error {
	_, err := h.app.Erc721Keeper.CallEVM(h.ctx, h.abi, erc721types.ModuleAddress, h.contract, true, method, args...)
	return err
}

// callOn sends a state-changing call to an arbitrary contract.
func (h *erc721EVMHarness) callOn(target common.Address, method string, args ...interface{}) error {
	_, err := h.app.Erc721Keeper.CallEVM(h.ctx, h.abi, erc721types.ModuleAddress, target, true, method, args...)
	return err
}

// query reads one accessor on an arbitrary contract.
func (h *erc721EVMHarness) query(t *testing.T, target common.Address, a abi.ABI, method string, args ...interface{}) []interface{} {
	t.Helper()

	res, err := h.app.Erc721Keeper.CallEVM(h.ctx, a, erc721types.ModuleAddress, target, false, method, args...)
	require.NoError(t, err, "query %s failed", method)
	ret, err := a.Unpack(method, res.Ret)
	require.NoError(t, err, "unpacking %s failed", method)
	return ret
}

func (h *erc721EVMHarness) totalSupply(t *testing.T) *big.Int {
	t.Helper()
	ret := h.query(t, h.contract, h.abi, "totalSupply")
	require.Len(t, ret, 1)
	value, ok := ret[0].(*big.Int)
	require.True(t, ok, "totalSupply did not return a uint256")
	return value
}

// ownerOf returns the owner and whether the token exists. A non-existent token
// makes the contract revert, which the harness turns into ok == false.
func (h *erc721EVMHarness) ownerOf(t *testing.T, id int64) (common.Address, bool) {
	t.Helper()

	res, err := h.app.Erc721Keeper.CallEVM(h.ctx, h.abi, erc721types.ModuleAddress, h.contract, false, "ownerOf", big.NewInt(id))
	if err != nil {
		return common.Address{}, false
	}
	ret, err := h.abi.Unpack("ownerOf", res.Ret)
	if err != nil || len(ret) != 1 {
		return common.Address{}, false
	}
	owner, ok := ret[0].(common.Address)
	if !ok {
		return common.Address{}, false
	}
	return owner, true
}

// enhance returns the per-token metadata. Unlike ownerOf this accessor does not
// revert for a missing token, so it is the right probe for leftover state.
func (h *erc721EVMHarness) enhance(t *testing.T, id int64) (name, uri, data, uriHash string) {
	t.Helper()

	ret := h.query(t, h.contract, h.abi, "getNFTEnhanceInfo", big.NewInt(id))
	require.Len(t, ret, 4)
	values := make([]string, 4)
	for i := range ret {
		s, ok := ret[i].(string)
		require.Truef(t, ok, "getNFTEnhanceInfo slot %d is not a string", i)
		values[i] = s
	}
	return values[0], values[1], values[2], values[3]
}

// requireNoToken asserts that an id has no owner and no leftover metadata.
func (h *erc721EVMHarness) requireNoToken(t *testing.T, id int64, context string) {
	t.Helper()

	_, exists := h.ownerOf(t, id)
	require.Falsef(t, exists, "%s: token %d must not exist", context, id)

	name, uri, data, uriHash := h.enhance(t, id)
	require.Emptyf(t, name, "%s: token %d left enhance name behind", context, id)
	require.Emptyf(t, uri, "%s: token %d left enhance uri behind", context, id)
	require.Emptyf(t, data, "%s: token %d left enhance data behind", context, id)
	require.Emptyf(t, uriHash, "%s: token %d left enhance uri hash behind", context, id)
}

// requireBigUint compares two uint256 values numerically. testify's Equal falls
// back to reflect.DeepEqual for *big.Int, which distinguishes big.NewInt(0)
// (nil magnitude) from an ABI-decoded zero (empty magnitude).
func requireBigUint(t *testing.T, want int64, got *big.Int, msg string) {
	t.Helper()
	require.Equalf(t, big.NewInt(want).String(), got.String(), "%s", msg)
}

// TestERC721ContractOnRealEVM groups the real-bytecode behaviour checks. Every
// scenario runs against the bytecode the node embeds, deployed in the app's own
// EVM, so a mismatch between the reviewed Solidity and the shipped artifact
// fails here.
func TestERC721ContractOnRealEVM(t *testing.T) {
	// The positive control for G-07: with _mint the hook is never called, so a
	// recorded callback is direct proof that _safeMint is in the artifact.
	t.Run("safe mint invokes the receiver hook", func(t *testing.T) {
		h := newERC721EVMHarness(t).scenario(t)

		receiver := h.deployFixture(t, "Accepting", nil)

		require.NoError(t, h.call("mint", receiver, big.NewInt(1), "ipfs://one"))

		requireBigUint(t, 1, h.uintOf(t, receiver, "received"),
			"a contract receiver must be notified; _mint would never call the hook")
		require.Equal(t, erc721types.ModuleAddress, h.addressOf(t, receiver, "lastOperator"),
			"the hook's operator must be the module account that minted")
		require.Equal(t, common.Address{}, h.addressOf(t, receiver, "lastFrom"),
			"a mint has no previous owner")
		requireBigUint(t, 1, h.uintOf(t, receiver, "lastTokenId"), "the hook must see the minted id")
		requireBigUint(t, 0, h.uintOf(t, receiver, "lastDataLength"), "a 2-arg safe mint passes no data")

		owner, exists := h.ownerOf(t, 1)
		require.True(t, exists)
		require.Equal(t, receiver, owner)

		// mintEnhance and the batch entry points must use the same safe path.
		require.NoError(t, h.call("mintEnhance", receiver, big.NewInt(2), "n2", "u2", "d2", "h2"))
		requireBigUint(t, 2, h.uintOf(t, receiver, "received"), "mintEnhance must notify the receiver")
		requireBigUint(t, 2, h.uintOf(t, receiver, "lastTokenId"), "the hook must see the enhanced id")

		require.NoError(t, h.call("mintBatch", receiver, []*big.Int{big.NewInt(3), big.NewInt(4)}))
		requireBigUint(t, 4, h.uintOf(t, receiver, "received"), "every id in a batch must notify the receiver")

		require.NoError(t, h.call("mintEnhanceBatch", receiver, []*big.Int{big.NewInt(5)}, "n5", "u5", "d5", "h5"))
		requireBigUint(t, 5, h.uintOf(t, receiver, "received"), "mintEnhanceBatch must notify the receiver")
	})

	// G-07: the token must not be handed to a contract that cannot take it.
	t.Run("mint rejects a contract without the receiver interface", func(t *testing.T) {
		h := newERC721EVMHarness(t).scenario(t)

		plain := h.deployFixture(t, "Plain", nil)
		before := h.totalSupply(t)

		err := h.call("mint", plain, big.NewInt(11), "ipfs://eleven")
		require.Error(t, err, "minting into a contract without onERC721Received must revert")
		require.Contains(t, err.Error(), "execution reverted",
			"the failure must be an EVM revert, not a Go-side rejection")

		require.Equal(t, before.String(), h.totalSupply(t).String(), "a rejected mint must not create a token")
		h.requireNoToken(t, 11, "rejected mint")

		// The enhance path shares the guard, and the metadata written before
		// _safeMint must be rolled back with the revert.
		err = h.call("mintEnhance", plain, big.NewInt(12), "n12", "u12", "d12", "h12")
		require.Error(t, err)
		h.requireNoToken(t, 12, "rejected enhance mint")
	})

	// G-07: a receiver that implements the interface but rejects the token.
	t.Run("mint rolls back when the receiver reverts", func(t *testing.T) {
		h := newERC721EVMHarness(t).scenario(t)

		reverting := h.deployFixture(t, "Reverting", nil)
		before := h.totalSupply(t)

		err := h.call("mint", reverting, big.NewInt(21), "ipfs://twentyone")
		require.Error(t, err, "a reverting receiver hook must fail the mint")
		require.Contains(t, err.Error(), "execution reverted")

		require.Equal(t, before.String(), h.totalSupply(t).String(), "the mint must roll back")
		h.requireNoToken(t, 21, "reverting receiver")
	})

	// G-07: a receiver that burns the whole gas budget must not leave a
	// half-minted token behind.
	t.Run("mint leaves no token when the receiver burns all gas", func(t *testing.T) {
		h := newERC721EVMHarness(t).scenario(t)

		burner := h.deployFixture(t, "GasBurning", nil)
		before := h.totalSupply(t)

		err := h.call("mint", burner, big.NewInt(31), "ipfs://gas")
		require.Error(t, err, "a receiver that never returns must fail the mint")

		require.Equal(t, before.String(), h.totalSupply(t).String(), "an out-of-gas hook must not leave a token")
		h.requireNoToken(t, 31, "gas burning receiver")
	})

	// G-07: one rejecting receiver must undo the whole batch.
	t.Run("batch mint is atomic", func(t *testing.T) {
		h := newERC721EVMHarness(t).scenario(t)

		reverting := h.deployFixture(t, "Reverting", nil)
		ids := []*big.Int{big.NewInt(41), big.NewInt(42), big.NewInt(43)}

		before := h.totalSupply(t)
		require.Error(t, h.call("mintBatch", reverting, ids))
		require.Equal(t, before.String(), h.totalSupply(t).String(), "no partial batch may survive")
		for _, id := range ids {
			h.requireNoToken(t, id.Int64(), "reverted batch")
		}

		before = h.totalSupply(t)
		require.Error(t, h.call("mintEnhanceBatch", reverting, ids, "n", "u", "d", "hd"))
		require.Equal(t, before.String(), h.totalSupply(t).String(), "no partial enhance batch may survive")
		for _, id := range ids {
			h.requireNoToken(t, id.Int64(), "reverted enhance batch")
		}
	})

	// G-07 + reentrancy: the hook reads the contract from inside the callback
	// and must see the token fully materialised.
	t.Run("the hook observes fully committed token state", func(t *testing.T) {
		h := newERC721EVMHarness(t).scenario(t)

		observer := h.deployFixture(t, "Observing", &h.contract)

		require.NoError(t, h.call("mintEnhance", observer, big.NewInt(51), "n51", "u51", "d51", "h51"))
		require.Equal(t, "1", h.totalSupply(t).String(), "the mint must succeed with a compliant receiver")

		require.Equal(t, observer, h.addressOf(t, observer, "observedOwner"),
			"the hook must already see the receiver as the owner")
		requireBigUint(t, 1, h.uintOf(t, observer, "observedBalance"), "the hook must already see the token")
		requireBigUint(t, 1, h.uintOf(t, observer, "observedTotalSupply"),
			"the hook must already see the token counted in totalSupply")
		require.Equal(t, "u51", h.stringOf(t, observer, "observedUri"),
			"metadata must be committed before the hook runs, not after")
		require.Equal(t, "n51", h.stringOf(t, observer, "observedEnhanceName"))
		require.Equal(t, "d51", h.stringOf(t, observer, "observedEnhanceData"))
		require.Equal(t, "h51", h.stringOf(t, observer, "observedEnhanceUriHash"))

		// The plain mint path must order its single field the same way.
		require.NoError(t, h.call("mint", observer, big.NewInt(52), "u52"))
		require.Equal(t, "u52", h.stringOf(t, observer, "observedUri"))
		require.Equal(t, observer, h.addressOf(t, observer, "observedOwner"))
	})

	// G-08: the inherited ERC721Burnable.burn used to leave _idEnhanceInfoMap
	// untouched, so a later re-mint of the same id inherited the dead token's
	// name/data/uriHash.
	t.Run("standard burn clears the enhance metadata", func(t *testing.T) {
		h := newERC721EVMHarness(t).scenario(t)

		// Mint to the module account so it owns the token and may burn it.
		owner := erc721types.ModuleAddress
		id := big.NewInt(61)

		require.NoError(t, h.call("mintEnhance", owner, id, "name-61", "uri-61", "data-61", "hash-61"))
		name, uri, data, uriHash := h.enhance(t, 61)
		require.Equal(t, "name-61", name)
		require.Equal(t, "uri-61", uri)
		require.Equal(t, "data-61", data)
		require.Equal(t, "hash-61", uriHash)

		// Standard burn, not burnEnhance: this is the entry point that bypassed
		// the cleanup.
		require.NoError(t, h.call("burn", id))
		h.requireNoToken(t, 61, "standard burn")

		// Re-mint the same id through the plain path, which only sets the URI:
		// nothing from the previous lifetime may survive.
		require.NoError(t, h.call("mint", owner, id, "uri-plain"))
		name, uri, data, uriHash = h.enhance(t, 61)
		require.Empty(t, name, "a re-minted id must not inherit the burned token's name")
		require.Equal(t, "uri-plain", uri)
		require.Empty(t, data, "a re-minted id must not inherit the burned token's data")
		require.Empty(t, uriHash, "a re-minted id must not inherit the burned token's uri hash")

		// burnEnhance is kept for ABI compatibility and must clean up too.
		require.NoError(t, h.call("burnEnhance", id))
		h.requireNoToken(t, 61, "burnEnhance")
	})

	// G-08 through the reentrant path: the receiver destroys the token it is
	// being handed while the mint frame is still on the stack. Whatever the
	// contract allows, the end state must be consistent.
	t.Run("burn from inside the hook leaves no residue", func(t *testing.T) {
		h := newERC721EVMHarness(t).scenario(t)

		burner := h.deployFixture(t, "BurnOnReceive", &h.contract)
		before := h.totalSupply(t)

		require.NoError(t, h.call("mintEnhance", burner, big.NewInt(71), "name-71", "uri-71", "data-71", "hash-71"))

		require.True(t, h.boolOf(t, burner, "reentered"), "the fixture must have re-entered and burned")
		require.Equal(t, before.String(), h.totalSupply(t).String(), "supply must be back where it started")
		h.requireNoToken(t, 71, "reentrant burn")
	})
}

func (h *erc721EVMHarness) stringOf(t *testing.T, target common.Address, method string) string {
	t.Helper()
	ret := h.query(t, target, receiverFixtureABI, method)
	require.Len(t, ret, 1)
	value, ok := ret[0].(string)
	require.Truef(t, ok, "%s did not return a string", method)
	return value
}

func (h *erc721EVMHarness) uintOf(t *testing.T, target common.Address, method string) *big.Int {
	t.Helper()
	ret := h.query(t, target, receiverFixtureABI, method)
	require.Len(t, ret, 1)
	value, ok := ret[0].(*big.Int)
	require.Truef(t, ok, "%s did not return a uint256", method)
	return value
}

func (h *erc721EVMHarness) boolOf(t *testing.T, target common.Address, method string) bool {
	t.Helper()
	ret := h.query(t, target, receiverFixtureABI, method)
	require.Len(t, ret, 1)
	value, ok := ret[0].(bool)
	require.Truef(t, ok, "%s did not return a bool", method)
	return value
}

func (h *erc721EVMHarness) addressOf(t *testing.T, target common.Address, method string) common.Address {
	t.Helper()
	ret := h.query(t, target, receiverFixtureABI, method)
	require.Len(t, ret, 1)
	value, ok := ret[0].(common.Address)
	require.Truef(t, ok, "%s did not return an address", method)
	return value
}
