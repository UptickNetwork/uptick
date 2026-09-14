package v040

import (
	"context"
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

// negative-control recipe (run manually, must fail):
//
//	sed -i '' 's/pair.ContractOwner != erc20types.OWNER_MODULE/pair.ContractOwner == erc20types.OWNER_MODULE/' upgrades.go
//	go test ./app/upgrades/v040/ -run TestDeleteLegacyOwnerModulePairsRemovesOnlyModuleOwned
//
// flipping the predicate makes the migration delete the OWNER_EXTERNAL pairs
// instead; the test then fails on the preservation branch. Verified red.
