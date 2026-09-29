package v040

import (
	"context"
	"encoding/hex"
	"testing"

	coreaddress "cosmossdk.io/core/address"
	"cosmossdk.io/log"
	rootstore "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	sdkaddress "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	cosmosevmutils "github.com/cosmos/evm/utils"
	erc20keeper "github.com/cosmos/evm/x/erc20/keeper"
	erc20types "github.com/cosmos/evm/x/erc20/types"

	"github.com/UptickNetwork/uptick/app/upgrades"
)

// ---------------------------------------------------------------------------
// Minimal stubs. deleteLegacyOwnerModulePairs only touches the erc20 KVStore,
// so the keeper's other collaborators (bank / evm / staking / transfer) are
// never called at runtime. erc20keeper.NewKeeper does call
// AccountKeeper.AddressCodec() during construction, so that one method has to
// return something real; everything else is a zero value.
// ---------------------------------------------------------------------------

var (
	_ erc20types.AccountKeeper = stubAccountKeeper{}
	_ erc20types.StakingKeeper = stubStakingKeeper{}
)

type stubAccountKeeper struct{}

func (stubAccountKeeper) AddressCodec() coreaddress.Codec {
	return sdkaddress.NewBech32Codec("cosmos")
}
func (stubAccountKeeper) GetModuleAddress(string) sdk.AccAddress {
	return authtypes.NewModuleAddress(erc20types.ModuleName)
}
func (stubAccountKeeper) GetSequence(context.Context, sdk.AccAddress) (uint64, error) { return 0, nil }
func (stubAccountKeeper) GetAccount(context.Context, sdk.AccAddress) sdk.AccountI     { return nil }

type stubStakingKeeper struct{}

func (stubStakingKeeper) BondDenom(context.Context) (string, error) { return "auptick", nil }

// ---------------------------------------------------------------------------
// Legacy wire-format encoder.
//
// The legacy uptick x/erc20 module (v0.3.3) declared its TokenPair with the
// exact same field numbers and types as cosmos/evm does today:
//
//	1 erc20_address  string
//	2 denom          string
//	3 enabled        bool
//	4 contract_owner Owner   (0 unspecified / 1 module / 2 external)
//
// so a proto3 encoder written by hand produces byte-for-byte what the old
// module stored. This matters: marshalling with the *new* codec would only
// prove the new module is self-consistent, and would silently pass even if the
// field layout had changed.
// ---------------------------------------------------------------------------

func legacyTokenPairBytes(erc20Address, denom string, enabled bool, owner uint64) []byte {
	var b []byte

	// field 1: erc20_address (string)
	b = append(b, 0x0A)
	b = appendUvarint(b, uint64(len(erc20Address)))
	b = append(b, erc20Address...)

	// field 2: denom (string)
	b = append(b, 0x12)
	b = appendUvarint(b, uint64(len(denom)))
	b = append(b, denom...)

	// field 3: enabled (bool) -- proto3 omits the zero value
	if enabled {
		b = append(b, 0x18, 0x01)
	}

	// field 4: contract_owner (enum) -- proto3 omits the zero value
	if owner != 0 {
		b = append(b, 0x20)
		b = appendUvarint(b, owner)
	}

	return b
}

func appendUvarint(b []byte, v uint64) []byte {
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

// ---------------------------------------------------------------------------
// Test fixtures: the four token pairs live on Uptick mainnet (chain v0.3.3)
// as of 2026-09-13, read straight off
// https://rest.uptick.network/uptick/erc20/v1/token_pairs
//
// All four are ibc/ denoms (inbound ICS-20 vouchers) owned by OWNER_MODULE,
// which is precisely the shape deleteLegacyOwnerModulePairs is meant to drop.
// ---------------------------------------------------------------------------

type legacyPairFixture struct {
	denom string
	erc20 string
}

var mainnetLegacyPairs = []legacyPairFixture{
	{ // IRIS, channel-0
		denom: "ibc/0F807ECA029E2E205F35320770440830709E0A902C5918CFADF05736A9308040",
		erc20: "0x80b5a32E4F032B2a058b4F29EC95EEfEEB87aDcd",
	},
	{ // ATOM, channel-1
		denom: "ibc/C4CFF46FD6DE35CA4CF4CE031E643C8FDC9BA4B99AE598E9B0ED98FE3A2319F9",
		erc20: "0xd567B3d7B8FE3C79a1AD8dA978812cfC4Fa05e75",
	},
	{ // IRIS, channel-2
		denom: "ibc/D5460C6C5B9D46389463C65201200F602490933DF1A2FFFE76C2FD234E811986",
		erc20: "0x5FD55A1B9FC24967C4dB09C513C3BA0DFa7FF687",
	},
	{ // USDC, channel-3
		denom: "ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4",
		erc20: "0xecEEEfCEE421D8062EF8d6b4D814efe4dc898265",
	},
}

// externalPairFixture is an OWNER_EXTERNAL pair: an ERC20 deployed by a user
// whose address was registered via MsgRegisterERC20. The migration must keep
// these, because their ERC20 address does not depend on the STRv2 scheme.
var externalPairs = []legacyPairFixture{
	{
		denom: "erc20:0x1111111111111111111111111111111111111111",
		erc20: "0x1111111111111111111111111111111111111111",
	},
	{
		denom: "erc20:0x2222222222222222222222222222222222222222",
		erc20: "0x2222222222222222222222222222222222222222",
	},
}

// ---------------------------------------------------------------------------

func newErc20TestKeeper(t *testing.T) (erc20keeper.Keeper, sdk.Context, *storetypes.KVStoreKey) {
	t.Helper()

	reg := codectypes.NewInterfaceRegistry()
	erc20types.RegisterInterfaces(reg)
	cdc := codec.NewProtoCodec(reg)

	db := dbm.NewMemDB()
	cms := rootstore.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	key := storetypes.NewKVStoreKey(erc20types.StoreKey)
	cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	ctx := sdk.NewContext(cms, cmtproto.Header{ChainID: "uptick_117-1"}, false, log.NewNopLogger())

	k := erc20keeper.NewKeeper(
		key,
		cdc,
		authtypes.NewModuleAddress(govtypes.ModuleName),
		stubAccountKeeper{},
		nil, // BankKeeper: unused by the pair-deletion path
		nil, // EVMKeeper:  unused
		stubStakingKeeper{},
		nil, // *transferkeeper.Keeper: unused
	)
	return k, ctx, key
}

// seedLegacyPair writes a pair the way the legacy module left it in the store:
// the value is raw proto3 bytes (not re-marshalled), and all three maps
// (pair / byDenom / byERC20) are populated.
func seedLegacyPair(t *testing.T, ctx sdk.Context, key *storetypes.KVStoreKey, f legacyPairFixture, owner uint64) []byte {
	t.Helper()

	raw := legacyTokenPairBytes(f.erc20, f.denom, true, owner)

	// Recover the pair id (tmhash(erc20|denom)) through the new codec. The
	// decode is asserted separately below, so this is not circular.
	var pair erc20types.TokenPair
	require.NoError(t, codec.NewProtoCodec(codectypes.NewInterfaceRegistry()).Unmarshal(raw, &pair))
	id := pair.GetID()

	store := ctx.KVStore(key)
	prefix.NewStore(store, erc20types.KeyPrefixTokenPair).Set(id, raw)
	prefix.NewStore(store, erc20types.KeyPrefixTokenPairByDenom).Set([]byte(f.denom), id)
	prefix.NewStore(store, erc20types.KeyPrefixTokenPairByERC20).Set(common.HexToAddress(f.erc20).Bytes(), id)

	return id
}

// TestLegacyTokenPairBytesAreReadableByNewCodec pins the compatibility
// property the whole migration rests on: bytes written by the legacy module
// decode under cosmos/evm's codec.
//
// It matters because IterateTokenPairs calls cdc.MustUnmarshal, which *panics*
// on malformed input -- a proto layout mismatch would hard-stop the upgrade
// rather than fail gracefully. If someone ever changes TokenPair's field
// numbers, this test is what catches it before mainnet does.
func TestLegacyTokenPairBytesAreReadableByNewCodec(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	for _, f := range mainnetLegacyPairs {
		t.Run(f.denom[4:12], func(t *testing.T) {
			raw := legacyTokenPairBytes(f.erc20, f.denom, true, uint64(erc20types.OWNER_MODULE))

			var pair erc20types.TokenPair
			require.NoError(t, cdc.Unmarshal(raw, &pair), "legacy bytes must decode under the new codec")

			require.Equal(t, f.erc20, pair.Erc20Address)
			require.Equal(t, f.denom, pair.Denom)
			require.True(t, pair.Enabled)
			require.Equal(t, erc20types.OWNER_MODULE, pair.ContractOwner)
			require.True(t, pair.IsNativeCoin(), "OWNER_MODULE pairs are the ones this migration drops")
		})
	}
}

// TestLegacyTokenPairEncodingMatchesNewCodec cross-checks the hand-written
// encoder against the generated one. The two must agree byte for byte,
// otherwise the fixtures above would be testing a format no chain ever stored.
func TestLegacyTokenPairEncodingMatchesNewCodec(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	for _, f := range mainnetLegacyPairs {
		legacy := legacyTokenPairBytes(f.erc20, f.denom, true, uint64(erc20types.OWNER_MODULE))
		generated, err := cdc.Marshal(&erc20types.TokenPair{
			Erc20Address:  f.erc20,
			Denom:         f.denom,
			Enabled:       true,
			ContractOwner: erc20types.OWNER_MODULE,
		})
		require.NoError(t, err)
		require.Equal(t, generated, legacy,
			"hand-written proto3 encoding diverged from the generated one")
	}
}

// TestDeleteLegacyOwnerModulePairsRemovesOnlyModuleOwned is the regression
// guard for "old pairs are discarded in v0.4.x".
//
// It seeds the store with the four real mainnet pairs plus two OWNER_EXTERNAL
// ones, runs the migration, and asserts:
//   - every OWNER_MODULE pair is gone from all three maps
//   - every OWNER_EXTERNAL pair survives intact
//
// The pre-assertions on the seeded state exist so this test cannot pass by
// doing nothing: if the seeding silently failed, the "before" counts fail
// first.
func TestDeleteLegacyOwnerModulePairsRemovesOnlyModuleOwned(t *testing.T) {
	k, ctx, key := newErc20TestKeeper(t)

	type seeded struct {
		fixture legacyPairFixture
		owner   uint64
		id      []byte
	}
	var all []seeded

	for _, f := range mainnetLegacyPairs {
		all = append(all, seeded{f, uint64(erc20types.OWNER_MODULE), seedLegacyPair(t, ctx, key, f, uint64(erc20types.OWNER_MODULE))})
	}
	for _, f := range externalPairs {
		all = append(all, seeded{f, uint64(erc20types.OWNER_EXTERNAL), seedLegacyPair(t, ctx, key, f, uint64(erc20types.OWNER_EXTERNAL))})
	}

	// Before: everything is readable through the keeper.
	pairsBefore := k.GetTokenPairs(ctx)
	require.Len(t, pairsBefore, len(all), "seeding must actually land in the store")

	var moduleBefore, externalBefore int
	for _, p := range pairsBefore {
		if p.ContractOwner == erc20types.OWNER_MODULE {
			moduleBefore++
		} else {
			externalBefore++
		}
	}
	require.Equal(t, 4, moduleBefore)
	require.Equal(t, 2, externalBefore)

	// Run the migration step.
	box := upgrades.Toolbox{}
	box.Erc20Keeper = k
	require.NoError(t, deleteLegacyOwnerModulePairs(ctx, box, log.NewNopLogger()))

	// After: OWNER_MODULE pairs are gone from the pair map, byDenom and byERC20.
	for _, s := range all {
		_, found := k.GetTokenPair(ctx, s.id)
		byDenom := k.GetTokenPairID(ctx, s.fixture.denom)
		byErc20 := k.GetERC20Map(ctx, common.HexToAddress(s.fixture.erc20))

		if s.owner == uint64(erc20types.OWNER_MODULE) {
			require.False(t, found, "OWNER_MODULE pair must be deleted: %s", s.fixture.denom)
			require.Nil(t, byDenom, "byDenom map must be cleared: %s", s.fixture.denom)
			require.Nil(t, byErc20, "byERC20 map must be cleared: %s", s.fixture.erc20)
		} else {
			require.True(t, found, "OWNER_EXTERNAL pair must be preserved: %s", s.fixture.denom)
			require.Equal(t, s.id, byDenom, "OWNER_EXTERNAL byDenom must survive")
			require.Equal(t, s.id, byErc20, "OWNER_EXTERNAL byERC20 must survive")
		}
	}

	// And the keeper agrees on the final count.
	require.Len(t, k.GetTokenPairs(ctx), len(externalPairs))
}

// TestDeleteLegacyOwnerModulePairsOnEmptyStore ensures the migration is a
// no-op (not a panic) on a chain that never registered a pair -- e.g. a fresh
// init chain replaying the upgrade, or a testnet like origin_1170-3.
func TestDeleteLegacyOwnerModulePairsOnEmptyStore(t *testing.T) {
	k, ctx, _ := newErc20TestKeeper(t)

	box := upgrades.Toolbox{}
	box.Erc20Keeper = k
	require.NoError(t, deleteLegacyOwnerModulePairs(ctx, box, log.NewNopLogger()))
	require.Empty(t, k.GetTokenPairs(ctx))
}

// ---------------------------------------------------------------------------
// mainnetStoredPairHex holds the RAW values of KeyPrefixTokenPair||id as read
// straight out of mainnet's erc20 store over ABCI, one entry per fixture above
// and in the same order.
//
//	# 0x03||denom -> id
//	curl -sk --noproxy '*' "https://rpc.uptick.network/abci_query?path=%22/store/erc20/key%22\
//	&data=0x03$(printf '%s' "$DENOM" | xxd -p)"
//	# 0x01||id -> the stored pair
//	curl -sk --noproxy '*' "https://rpc.uptick.network/abci_query?path=%22/store/erc20/key%22\
//	&data=0x01$(printf '%s' "$ID" | xxd -p)"
//
// Read on 2026-09-29 at height 19,420,683 (mainnet was still on v0.3.3).
//
// Why pin raw chain bytes instead of extending the fixtures above: those are
// produced by THIS repository's encoder, so on their own they only prove the
// encoder agrees with itself. These bytes were written by the legacy module
// (v0.3.3, ethermint-era x/erc20) and were never produced by this code.
//
// The deletion this guards is a store iteration whose decode is
// cdc.MustUnmarshal, so exactly two properties have to hold against these exact
// bytes: (1) they decode under cosmos/evm's codec without panicking -- a panic
// there stops mainnet at the upgrade height rather than failing a migration --
// and (2) contract_owner decodes to OWNER_MODULE, the single value the deletion
// predicate selects on. A silent misread would turn the migration into a no-op
// and leave the deprecated pairs in place, which is the quieter failure of the
// two.
// ---------------------------------------------------------------------------
var mainnetStoredPairHex = []string{
	// IRIS, channel-0 -- 0x0a 0x2a "0x80b5a32E…" | 0x12 0x44 "ibc/0F80…" | 0x18 0x01 | 0x20 0x01
	"0a2a307838306235613332453446303332423261303538623446323945433935454566454542383761446364" +
		"12446962632f30463830374543413032394532453230354633353332303737303434303833303730394530" +
		"413930324335393138434641444630353733364139333038303430" +
		"18012001",
	// ATOM, channel-1
	"0a2a307864353637423364374238464533433739613141443864413937383831326366433446613035653735" +
		"12446962632f43344346463436464436444533354341344346344345303331453634334338464443394241" +
		"344239394145353938453942304544393846453341323331394639" +
		"18012001",
	// IRIS, channel-2
	"0a2a307835464435354131423946433234393637433464423039433531334333424130444661374646363837" +
		"12446962632f44353436304336433542394434363338393436334336353230313230304636303234393039" +
		"333344463141324646464537364332464432333445383131393836" +
		"18012001",
	// USDC, channel-3
	"0a2a3078656345454566434545343231443830363245463864366234443831346566653464633839383236" +
		"3512446962632f3634393041374541423631303539424643314344444542303539313744443730424446" +
		"3341363131363534313632413141343744423933304434304438414634" +
		"18012001",
}

// TestMainnetStoredTokenPairsDecodeAndSelectForDeletion runs the migration's own
// decode-and-select path over mainnet's actual stored bytes.
//
// Everything else in this file works on fixtures this repository generated. That
// is enough to catch a change to TokenPair's field numbers, but not enough to
// prove mainnet's store is readable: only the bytes the legacy module wrote can
// do that. Mainnet is the chain that follows v0.5.0, so this is the test that
// stands in for it.
func TestMainnetStoredTokenPairsDecodeAndSelectForDeletion(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	require.Len(t, mainnetStoredPairHex, len(mainnetLegacyPairs),
		"one stored value per fixture; a new fixture has to bring its own chain bytes")

	k, ctx, key := newErc20TestKeeper(t)

	for i, f := range mainnetLegacyPairs {
		t.Run(f.denom[4:12], func(t *testing.T) {
			stored, err := hex.DecodeString(mainnetStoredPairHex[i])
			require.NoError(t, err)

			var pair erc20types.TokenPair
			require.NoError(t, cdc.Unmarshal(stored, &pair),
				"mainnet's stored bytes must decode: IterateTokenPairs calls MustUnmarshal on exactly this input, and a failure there halts the chain")

			require.Equal(t, f.erc20, pair.Erc20Address)
			require.Equal(t, f.denom, pair.Denom)
			require.True(t, pair.Enabled)
			require.Equal(t, erc20types.OWNER_MODULE, pair.ContractOwner,
				"the deletion selects on contract_owner; OWNER_EXTERNAL here would make the mainnet deletion a silent no-op")
			require.True(t, pair.IsNativeCoin())

			// The repo's hand-written encoder has to reproduce mainnet's bytes
			// exactly. Without this, every fixture in this file could be
			// describing a format no chain ever stored.
			require.Equal(t,
				stored,
				legacyTokenPairBytes(f.erc20, f.denom, true, uint64(erc20types.OWNER_MODULE)),
				"the fixture encoder diverged from what mainnet actually stored")

			// Seed with the chain's bytes, not the encoder's.
			store := ctx.KVStore(key)
			prefix.NewStore(store, erc20types.KeyPrefixTokenPair).Set(pair.GetID(), stored)
			prefix.NewStore(store, erc20types.KeyPrefixTokenPairByDenom).Set([]byte(f.denom), pair.GetID())
			prefix.NewStore(store, erc20types.KeyPrefixTokenPairByERC20).Set(common.HexToAddress(f.erc20).Bytes(), pair.GetID())
		})
	}

	// GetTokenPairs is the iterator the migration walks. Reaching this line at
	// all means no MustUnmarshal panicked on real chain bytes.
	read := k.GetTokenPairs(ctx)
	require.Len(t, read, len(mainnetLegacyPairs),
		"every mainnet pair must be readable through the iterator the migration uses")
	for _, p := range read {
		require.Equal(t, erc20types.OWNER_MODULE, p.ContractOwner,
			"every mainnet pair has to be one the deletion selects")
	}
}

// TestDeprecatedLegacyERC20LosesItsCosmosBridgeButStaysQueryable is the
// executable form of the deprecation contract mainnet runs under after v0.5.0.
//
// The decision it pins: mainnet's four legacy IBC-ERC20 assets are deprecated
// rather than carried forward. Their counterparty chains are gone or going
// (IRIS halted, the Cosmos Hub channel closed, Noble retiring USDC), so no
// compatibility work is warranted. What the chain must still do is behave
// predictably afterwards: the asset stays *queryable* and stops being *usable*.
//
// The assertions are three distinct claims, and none of them is sufficient
// alone:
//
//   - no Cosmos bridge: MintingEnabled -- the gate MsgConvertERC20 goes through
//     -- can no longer resolve a pair from the contract address. Without this
//     the asset would still convert and the deprecation would not exist.
//   - still queryable: the contract address must be absent from BOTH precompile
//     registries. If it were registered, GetERC20PrecompileInstance would
//     intercept calls and InstantiateERC20Precompile would fail with
//     "precompile id not found" -- the pair it looks up was just deleted --
//     turning every eth_call into an error. Absent from the registries means
//     ordinary EVM execution against the deployed bytecode. Measured on
//     mainnet: both prefixes are empty for all four addresses.
//   - the interception boundary moved rather than vanished: the STRv2 derived
//     address IS registered as a dynamic precompile while the legacy address is
//     not. That is what lets an operator (and the EVM) tell the two
//     representations apart, and it is the positive control that makes the
//     "absent" assertion above mean something rather than reflecting a registry
//     that is simply never written.
//
// The keeper is built with a nil BankKeeper and a nil EVMKeeper on purpose (see
// newErc20TestKeeper). The migration completing at all is what proves it never
// reaches for the escrowed coins: on mainnet those escrowed balances equal the
// deprecated ERC20 totalSupply to the unit (16,638,820,054 across the four
// vouchers), so a version of this migration that started burning or moving them
// would panic here instead of passing. Closing that loop is a separate,
// supply-changing decision and is deliberately not taken here.
func TestDeprecatedLegacyERC20LosesItsCosmosBridgeButStaysQueryable(t *testing.T) {
	k, ctx, key := newErc20TestKeeper(t)

	// Precondition, asserted rather than assumed: with EnableErc20 off,
	// MintingEnabled short-circuits before it ever looks the pair up, and the
	// bridge assertion below would pass for the wrong reason.
	require.NoError(t, k.SetParams(ctx, erc20types.Params{EnableErc20: true}))
	require.True(t, k.IsERC20Enabled(ctx))

	// Seed the four deprecated assets with mainnet's real bytes.
	for i, f := range mainnetLegacyPairs {
		stored, err := hex.DecodeString(mainnetStoredPairHex[i])
		require.NoError(t, err)
		var pair erc20types.TokenPair
		require.NoError(t, codec.NewProtoCodec(codectypes.NewInterfaceRegistry()).Unmarshal(stored, &pair))

		store := ctx.KVStore(key)
		prefix.NewStore(store, erc20types.KeyPrefixTokenPair).Set(pair.GetID(), stored)
		prefix.NewStore(store, erc20types.KeyPrefixTokenPairByDenom).Set([]byte(f.denom), pair.GetID())
		prefix.NewStore(store, erc20types.KeyPrefixTokenPairByERC20).Set(common.HexToAddress(f.erc20).Bytes(), pair.GetID())
	}
	require.Len(t, k.GetTokenPairs(ctx), len(mainnetLegacyPairs), "seed must land before the migration runs")

	// Run the real migration with no bank and no EVM keeper available to it.
	box := upgrades.Toolbox{}
	box.Erc20Keeper = k
	require.NoError(t, deleteLegacyOwnerModulePairs(ctx, box, log.NewNopLogger()))
	require.Empty(t, k.GetTokenPairs(ctx), "all four deprecated pairs must be gone")

	for _, f := range mainnetLegacyPairs {
		addr := common.HexToAddress(f.erc20)

		// No Cosmos bridge: the ERC20 -> coin direction cannot resolve a pair.
		_, err := k.MintingEnabled(ctx, sdk.AccAddress(make([]byte, 20)), addr.Hex())
		require.ErrorIs(t, err, erc20types.ErrTokenPairNotFound,
			"%s must no longer convert: the deprecated asset has to stop being usable on the Cosmos side", f.erc20)

		// No pair by denom either, so the coin -> ERC20 direction is closed too.
		require.False(t, k.IsDenomRegistered(ctx, f.denom),
			"the voucher must be unregistered on this starting state, before any backfill")

		// Queryable: the address is not a precompile, so calls are plain EVM.
		_, found, err := k.GetERC20PrecompileInstance(ctx, addr)
		require.NoError(t, err)
		require.False(t, found,
			"%s must not be intercepted as an erc20 precompile: interception without a pair errors on every call instead of reading the deployed contract", f.erc20)
		require.False(t, k.IsNativePrecompileAvailable(ctx, addr))
		require.False(t, k.IsDynamicPrecompileAvailable(ctx, addr))
	}

	// Positive control for the two "absent" checks above: the STRv2 derived
	// address for the same denom IS registered. This is what v0.5.0's backfill
	// does (RegisterERC20Extension), reproduced at the store level so the test
	// does not depend on the EVM keeper the stub cannot provide.
	f := mainnetLegacyPairs[3] // USDC: the one voucher with a live bank-side float
	derived, err := cosmosevmutils.GetIBCDenomAddress(f.denom)
	require.NoError(t, err)
	require.NotEqual(t, common.HexToAddress(f.erc20), derived,
		"the derived address must differ from the legacy contract, or the two representations would collide")

	require.NoError(t, k.SetToken(ctx, erc20types.NewTokenPair(derived, f.denom, erc20types.OWNER_MODULE)))
	k.SetDynamicPrecompile(ctx, derived)

	require.True(t, k.IsDynamicPrecompileAvailable(ctx, derived),
		"the registry read must be able to answer true, otherwise the four absent checks above prove nothing")
	require.False(t, k.IsDynamicPrecompileAvailable(ctx, common.HexToAddress(f.erc20)),
		"registering the derived address must not resurrect the legacy contract")

	// And the coin side still has no conversion: STRv2 pairs are OWNER_MODULE by
	// construction (types.NewTokenPairSTRv2), and cosmos/evm refuses that owner
	// in both directions -- MsgConvertCoin at keeper/msg_server.go:78-79 and
	// MsgConvertERC20 at keeper/convert.go:40-42. Asserting the owner is the
	// honest form here: reaching the msg server would need a bank keeper.
	pair, found := k.GetTokenPair(ctx, k.GetTokenPairID(ctx, f.denom))
	require.True(t, found, "the backfilled voucher must be registered by denom")
	require.True(t, pair.IsNativeCoin(),
		"an STRv2 pair is OWNER_MODULE, which is exactly why the coin<->ERC20 conversion stays disabled after the upgrade")
}

// negative-control recipes (run manually, each must fail):
//
//  1. flip the deletion predicate -- it then removes the OWNER_EXTERNAL pairs
//     instead, and TestDeleteLegacyOwnerModulePairsRemovesOnlyModuleOwned fails
//     on its preservation branch:
//
//       python3 - <<'PY'
//       p = "app/upgrades/v040/upgrades.go"
//       s = open(p).read()
//       open(p, "w").write(s.replace(
//           "pair.ContractOwner != erc20types.OWNER_MODULE",
//           "pair.ContractOwner == erc20types.OWNER_MODULE"))
//       PY
//       go test -count=1 ./app/upgrades/v040/ -run TestDeleteLegacyOwnerModulePairs
//
//     ⚠️ Use python3, not `sed -i ''`: BSD sed fails silently in this sandbox and
//     the "mutation" would leave the file untouched, so the green run would look
//     like evidence when it is only proof that nothing changed.
//
//  2. corrupt one byte of mainnetStoredPairHex (e.g. change a trailing "2001" to
//     "2002") -- contract_owner decodes to OWNER_EXTERNAL and
//     TestMainnetStoredTokenPairsDecodeAndSelectForDeletion fails with
//     "the deletion selects on contract_owner". This is the mutation that shows
//     the byte-level pinning is load-bearing rather than decorative.
