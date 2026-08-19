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
