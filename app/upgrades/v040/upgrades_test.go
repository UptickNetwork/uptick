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
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/app/upgrades"
	"github.com/UptickNetwork/uptick/app/upgrades/v040/legacy"
	evmsecp256k1 "github.com/cosmos/evm/crypto/ethsecp256k1"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

func TestMigrateEVMChainConfig(t *testing.T) {
	ctx := sdk.NewContext(nil, cmtproto.Header{ChainID: "uptick_117-1"}, false, log.NewNopLogger())

	require.NoError(t, migrateEVMChainConfig(ctx, upgrades.Toolbox{}, log.NewNopLogger()))

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

// TestMigrateLegacyEVMAccountsSkipsVesting ensures the account migration
// iterating over the auth store can decode (and safely skip) vesting accounts
// such as PeriodicVestingAccount. Without registering the vesting types in
// newLegacyAccountCodec, replay of an upgrade on a chain holding vesting
// accounts fails with "no concrete type registered for type URL
// /cosmos.vesting.v1beta1.PeriodicVestingAccount against interface
// *types.AccountI".
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

func TestDecodeLegacyBoolRaw(t *testing.T) {
	logger := log.NewNopLogger()
	ctx := sdk.Context{}
	box := upgrades.Toolbox{}

	require.True(t, decodeLegacyBoolRaw(ctx, box, logger, "erc20", "EnableErc20", false, []byte("true")))
	require.False(t, decodeLegacyBoolRaw(ctx, box, logger, "erc20", "EnableErc20", false, []byte("false")))
	require.False(t, decodeLegacyBoolRaw(ctx, box, logger, "erc20", "EnableErc20", false, []byte("not-a-bool")))
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
