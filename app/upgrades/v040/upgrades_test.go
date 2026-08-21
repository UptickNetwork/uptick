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
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/app/upgrades"
	"github.com/UptickNetwork/uptick/app/upgrades/v040/legacy"
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
