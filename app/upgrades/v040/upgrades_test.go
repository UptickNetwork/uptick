package v040

import (
	"testing"

	"cosmossdk.io/log"
	rootstore "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/UptickNetwork/uptick/app/upgrades"
	"github.com/UptickNetwork/uptick/app/upgrades/v040/legacy"
	evmsecp256k1 "github.com/cosmos/evm/crypto/ethsecp256k1"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

func TestMigrateEVMChainConfig(t *testing.T) {
	ctx := sdk.NewContext(nil, cmtproto.Header{ChainID: "uptick_117-1"}, false, log.NewNopLogger())

	require.NoError(t, migrateEVMChainConfig(ctx, log.NewNopLogger()))

	cfg := evmtypes.GetChainConfig()
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.ShanghaiTime)
	require.NotNil(t, cfg.CancunTime)
	require.NotNil(t, cfg.PragueTime)
	require.Nil(t, cfg.OsakaTime)
	require.Nil(t, cfg.VerkleTime)
}

func TestMigrateLegacyEVMAccounts(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	authtypes.RegisterInterfaces(cdc.InterfaceRegistry())

	db := dbm.NewMemDB()
	cms := rootstore.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	authKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	cms.MountStoreWithDB(authKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	addr := sdk.AccAddress([]byte("legacyaddr"))
	legacyAccount := &legacy.EthAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address:       addr.String(),
			PubKey:        nil,
			AccountNumber: 7,
			Sequence:      3,
		},
		CodeHash: "0xabcdef",
	}

	migrationCdc := newLegacyAccountCodec()
	legacyBytes, err := migrationCdc.MarshalInterface(legacyAccount)
	require.NoError(t, err)

	store := prefix.NewStore(cms.GetKVStore(authKey), []byte(authtypes.AddressStoreKeyPrefix))
	store.Set(addr.Bytes(), legacyBytes)

	ctx := sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
	require.NoError(t, migrateLegacyEVMAccounts(ctx, authKey, cdc, nil, log.NewNopLogger()))

	var accountI sdk.AccountI
	require.NoError(t, cdc.UnmarshalInterface(store.Get(addr.Bytes()), &accountI))
	baseAccount, ok := accountI.(*authtypes.BaseAccount)
	require.True(t, ok)
	require.Equal(t, addr.String(), baseAccount.Address)
	require.Equal(t, uint64(7), baseAccount.AccountNumber)
	require.Equal(t, uint64(3), baseAccount.Sequence)
}

// TestMigrateLegacyEVMAccountsSkipsVesting ensures the account migration can
// decode (and safely skip) vesting accounts such as PeriodicVestingAccount.
func TestMigrateLegacyEVMAccountsSkipsVesting(t *testing.T) {
	reg := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(reg)
	vestingtypes.RegisterInterfaces(reg)
	legacy.RegisterInterfaces(reg)
	reg.RegisterImplementations((*cryptotypes.PubKey)(nil), &evmsecp256k1.PubKey{})
	cdc := codec.NewProtoCodec(reg)

	db := dbm.NewMemDB()
	cms := rootstore.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	authKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	cms.MountStoreWithDB(authKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	vestAddr := sdk.AccAddress([]byte("vestingaddr"))
	vestAcc, err := vestingtypes.NewPeriodicVestingAccount(
		&authtypes.BaseAccount{Address: vestAddr.String(), AccountNumber: 1, Sequence: 0},
		sdk.NewCoins(sdk.NewInt64Coin("auptick", 1000)),
		1000,
		vestingtypes.Periods{{Length: 100, Amount: sdk.NewCoins(sdk.NewInt64Coin("auptick", 1000))}},
	)
	require.NoError(t, err)
	// Simulate a pre-upgrade vesting account carrying the legacy Ethermint pubkey.
	legacyPk := &legacy.EthSecp256k1PubKey{Key: make([]byte, 33)}
	legacyPkAny, err := codectypes.NewAnyWithValue(legacyPk)
	require.NoError(t, err)
	vestAcc.BaseVestingAccount.BaseAccount.PubKey = legacyPkAny

	bz, err := cdc.MarshalInterface(vestAcc)
	require.NoError(t, err)
	store := prefix.NewStore(cms.GetKVStore(authKey), []byte(authtypes.AddressStoreKeyPrefix))
	store.Set(vestAddr, bz)

	ctx := sdk.NewContext(cms, cmtproto.Header{ChainID: "origin_1170-3"}, false, log.NewNopLogger())
	require.NoError(t, migrateLegacyEVMAccounts(ctx, authKey, cdc, nil, log.NewNopLogger()))

	// The vesting account must remain intact and decodable after migration.
	var acc sdk.AccountI
	require.NoError(t, cdc.UnmarshalInterface(store.Get(vestAddr), &acc))
	vest2, ok := acc.(*vestingtypes.PeriodicVestingAccount)
	require.True(t, ok, "expected PeriodicVestingAccount to be preserved, got %T", acc)
	// The legacy pubkey must be rewritten to the v0.4.0 key type.
	newPk := vest2.GetPubKey()
	_, isNew := newPk.(*evmsecp256k1.PubKey)
	require.True(t, isNew, "expected migrated cosmos/evm pubkey, got %T", newPk)
}

// TestMigrateLegacyEVMAccountsAggregatesFailures ensures the auth migration
// scans the full store and reports every undecodable account in one aggregated
// error (fail-closed: a skipped legacy EthAccount would be unreadable after
// the upgrade) instead of aborting at the first bad record.
func TestMigrateLegacyEVMAccountsAggregatesFailures(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	authtypes.RegisterInterfaces(cdc.InterfaceRegistry())

	db := dbm.NewMemDB()
	cms := rootstore.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	authKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	cms.MountStoreWithDB(authKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	store := prefix.NewStore(cms.GetKVStore(authKey), []byte(authtypes.AddressStoreKeyPrefix))

	// A healthy legacy EthAccount that should be migratable.
	goodAddr := sdk.AccAddress([]byte("goodaccount"))
	goodAccount := &legacy.EthAccount{
		BaseAccount: &authtypes.BaseAccount{Address: goodAddr.String(), AccountNumber: 1, Sequence: 1},
	}
	migrationCdc := newLegacyAccountCodec()
	goodBytes, err := migrationCdc.MarshalInterface(goodAccount)
	require.NoError(t, err)
	store.Set(goodAddr.Bytes(), goodBytes)

	// Two corrupted records (invalid protobuf for any registered account type).
	badAddr1 := sdk.AccAddress([]byte("badaccount1"))
	badAddr2 := sdk.AccAddress([]byte("badaccount2"))
	store.Set(badAddr1.Bytes(), []byte{0xde, 0xad, 0xbe, 0xef})
	store.Set(badAddr2.Bytes(), []byte{0x00, 0xff})

	ctx := sdk.NewContext(cms, cmtproto.Header{ChainID: "uptick_117-1"}, false, log.NewNopLogger())
	err = migrateLegacyEVMAccounts(ctx, authKey, cdc, nil, log.NewNopLogger())

	// Fail-closed: one aggregated error covering both bad records — proving the
	// scan continued past the first failure instead of stopping there.
	require.Error(t, err)
	require.Contains(t, err.Error(), "2 legacy auth account(s) failed to migrate")
}

func TestDecodeLegacyBoolRaw(t *testing.T) {
	logger := log.NewNopLogger()

	require.True(t, decodeLegacyBoolRaw(logger, "erc20", "EnableErc20", []byte("true")))
	require.False(t, decodeLegacyBoolRaw(logger, "erc20", "EnableErc20", []byte("false")))
	// Corrupt raw value must fail-closed to false (disabled), never re-enable a
	// feature governance had turned off.
	require.False(t, decodeLegacyBoolRaw(logger, "erc20", "EnableErc20", []byte("not-a-bool")))
}

func TestReadLegacyBoolParamRaw(t *testing.T) {
	db := dbm.NewMemDB()
	cms := rootstore.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	paramsKey := storetypes.NewKVStoreKey(paramstypes.StoreKey)
	cms.MountStoreWithDB(paramsKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	ctx := sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
	ctx.KVStore(paramsKey).Set([]byte("erc20/EnableErc20"), []byte("false"))

	require.False(t, readLegacyBoolParamRaw(ctx, paramsKey, "erc20", "EnableErc20", true, log.NewNopLogger()))
	require.True(t, readLegacyBoolParamRaw(ctx, paramsKey, "erc20", "Missing", true, log.NewNopLogger()))
}

func TestDeleteLegacyParamsSubspace(t *testing.T) {
	db := dbm.NewMemDB()
	cms := rootstore.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	paramsKey := storetypes.NewKVStoreKey(paramstypes.StoreKey)
	cms.MountStoreWithDB(paramsKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	ctx := sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
	store := ctx.KVStore(paramsKey)
	store.Set([]byte("erc20/EnableErc20"), []byte("true"))
	store.Set([]byte("erc721/EnableErc721"), []byte("true"))
	store.Set([]byte("other/Keep"), []byte("true"))

	deleteLegacyParamsSubspace(ctx, paramsKey, log.NewNopLogger(), "erc20", "erc721")

	require.Nil(t, store.Get([]byte("erc20/EnableErc20")))
	require.Nil(t, store.Get([]byte("erc721/EnableErc721")))
	require.NotNil(t, store.Get([]byte("other/Keep")))
}

// TestMigrateLegacyEVMAccountsSkipsMultisig ensures the migration can decode
// multisig accounts whose pubkey is a legacy amino multisig key
// (/cosmos.crypto.multisig.LegacyAminoPubKey) and leaves them intact.
func TestMigrateLegacyEVMAccountsSkipsMultisig(t *testing.T) {
	reg := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(reg)
	vestingtypes.RegisterInterfaces(reg)
	cryptocodec.RegisterInterfaces(reg)
	legacy.RegisterInterfaces(reg)
	reg.RegisterImplementations((*cryptotypes.PubKey)(nil), &evmsecp256k1.PubKey{})
	cdc := codec.NewProtoCodec(reg)

	db := dbm.NewMemDB()
	cms := rootstore.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	authKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	cms.MountStoreWithDB(authKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	msAddr := sdk.AccAddress([]byte("multisigaddr"))
	subPk := ed25519.GenPrivKey().PubKey()
	msPk := multisig.NewLegacyAminoPubKey(1, []cryptotypes.PubKey{subPk})
	msAny, err := codectypes.NewAnyWithValue(msPk)
	require.NoError(t, err)
	acc := &authtypes.BaseAccount{
		Address:       msAddr.String(),
		AccountNumber: 2,
		Sequence:      0,
		PubKey:        msAny,
	}

	bz, err := cdc.MarshalInterface(acc)
	require.NoError(t, err)
	store := prefix.NewStore(cms.GetKVStore(authKey), []byte(authtypes.AddressStoreKeyPrefix))
	store.Set(msAddr, bz)

	ctx := sdk.NewContext(cms, cmtproto.Header{ChainID: "origin_1170-3"}, false, log.NewNopLogger())
	require.NoError(t, migrateLegacyEVMAccounts(ctx, authKey, cdc, nil, log.NewNopLogger()))

	var got sdk.AccountI
	require.NoError(t, cdc.UnmarshalInterface(store.Get(msAddr), &got))
	pk := got.GetPubKey()
	_, isMs := pk.(*multisig.LegacyAminoPubKey)
	require.True(t, isMs, "expected multisig pubkey to be preserved, got %T", pk)
}

func TestRepairEvmDenomMetadata(t *testing.T) {
	key := storetypes.NewKVStoreKey(banktypes.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx

	reg := codectypes.NewInterfaceRegistry()
	banktypes.RegisterInterfaces(reg)
	authtypes.RegisterInterfaces(reg)
	cdc := codec.NewProtoCodec(reg)

	ctrl := gomock.NewController(t)
	authKeeper := banktestutil.NewMockAccountKeeper(ctrl)
	authKeeper.EXPECT().AddressCodec().Return(address.NewBech32Codec("cosmos")).AnyTimes()

	bk := bankkeeper.NewBaseKeeper(
		cdc,
		runtime.NewKVStoreService(key),
		authKeeper,
		map[string]bool{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		log.NewNopLogger(),
	)

	box := upgrades.Toolbox{}
	box.BankKeeper = bk

	// Scenario: Display unit is missing from DenomUnits (display "origin" while
	// units only list "auoc"/"uoc"). This is the origin testnet metadata shape
	// that made decimals resolve to 0 and panic InitEvmCoinInfo.
	bk.SetDenomMetaData(ctx, banktypes.Metadata{
		Description: "Origin token",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: "auoc", Exponent: 0, Aliases: []string{}},
			{Denom: "uoc", Exponent: 18, Aliases: []string{}},
		},
		Base:    "auoc",
		Display: "origin",
		Name:    "Origin",
		Symbol:  "UOC",
	})

	repairEvmDenomMetadata(ctx, box, "auoc", log.NewNopLogger())
	got, found := bk.GetDenomMetaData(ctx, "auoc")
	require.True(t, found)
	require.Equal(t, "uoc", got.Display)
}
