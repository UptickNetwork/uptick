package keeper

import (
	"context"
	"math/big"
	"testing"

	coreaddress "cosmossdk.io/core/address"
	"cosmossdk.io/log"
	rootstore "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/nft"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codecAddress "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	collectionkeeper "github.com/UptickNetwork/uptick/x/collection/keeper"
	collectiontypes "github.com/UptickNetwork/uptick/x/collection/types"
	"github.com/UptickNetwork/uptick/x/erc721/types"
	"github.com/cosmos/evm/x/vm/statedb"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

func setupConvertKeeper(t *testing.T) (Keeper, sdk.Context, sdk.AccAddress) {
	t.Helper()
	sdk.GetConfig().SetBech32PrefixForAccount("uptick", "uptickpub")

	ir := codectypes.NewInterfaceRegistry()
	collectiontypes.RegisterInterfaces(ir)
	types.RegisterInterfaces(ir)
	cdc := codec.NewProtoCodec(ir)

	db := dbm.NewMemDB()
	cms := rootstore.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ercKey := storetypes.NewKVStoreKey(types.StoreKey)
	colKey := storetypes.NewKVStoreKey(collectiontypes.StoreKey)
	cms.MountStoreWithDB(ercKey, storetypes.StoreTypeIAVL, db)
	cms.MountStoreWithDB(colKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	ctx := sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
	ak := &convertAccountKeeper{}
	bk := &convertBankKeeper{}
	nftK := collectionkeeper.NewKeeper(cdc, runtime.NewKVStoreService(colKey), ak, bk)

	owner := sdk.AccAddress(bytes20(0x11))
	contract := common.HexToAddress("0x1111111111111111111111111111111111111111")
	evm := &fakeEVMKeeper{
		accounts: map[common.Address]*statedb.Account{
			contract: {CodeHash: []byte{1, 2, 3}},
		},
	}

	k := Keeper{
		storeKey:      ercKey,
		cdc:           cdc,
		accountKeeper: ak,
		nftKeeper:     nftK,
		evmKeeper:     evm,
	}

	require.NoError(t, nftK.SaveDenom(ctx, "kitty", "Kitty", "", "KIT", owner, false, false, "", "", "", ""))
	require.NoError(t, nftK.SaveNFT(ctx, "kitty", "nft1", "Spot", "ipfs://nft", "", "", owner))

	pair := types.NewTokenPair(contract, "kitty")
	k.SetTokenPair(ctx, pair)
	k.SetClassMap(ctx, pair.ClassId, pair.GetID())
	k.SetERC721Map(ctx, pair.GetERC721Contract(), pair.GetID())

	return k, ctx, owner
}

func TestConvertNFT_Disabled(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)
	require.NoError(t, k.SetParams(ctx, types.NewParams(false, true)))

	_, err := k.ConvertNFT(ctx, &types.MsgConvertNFT{
		ClassId:            "kitty",
		CosmosTokenIds:     []string{"nft1"},
		EvmContractAddress: "0x1111111111111111111111111111111111111111",
		EvmTokenIds:        []string{"1"},
		CosmosSender:       owner.String(),
		EvmReceiver:        "0x2222222222222222222222222222222222222222",
	})
	require.ErrorIs(t, err, types.ErrERC721Disabled)
}

func TestConvertERC721_Disabled(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)
	require.NoError(t, k.SetParams(ctx, types.NewParams(false, true)))

	_, err := k.ConvertERC721(ctx, &types.MsgConvertERC721{
		ClassId:            "kitty",
		CosmosTokenIds:     []string{"nft1"},
		EvmContractAddress: "0x1111111111111111111111111111111111111111",
		EvmTokenIds:        []string{"1"},
		CosmosSender:       owner.String(),
		CosmosReceiver:     owner.String(),
	})
	require.ErrorIs(t, err, types.ErrERC721Disabled)
}

func TestConvertNFT_SelfDestructedPair(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)
	k.evmKeeper = &fakeEVMKeeper{accounts: map[common.Address]*statedb.Account{}}

	_, err := k.ConvertNFT(ctx, &types.MsgConvertNFT{
		ClassId:            "kitty",
		CosmosTokenIds:     []string{"nft1"},
		EvmContractAddress: "0x1111111111111111111111111111111111111111",
		EvmTokenIds:        []string{"1"},
		CosmosSender:       owner.String(),
		EvmReceiver:        "0x2222222222222222222222222222222222222222",
	})
	require.ErrorIs(t, err, types.ErrInternalTokenPair)
}

func TestRegisterERC721_RejectsExistingNativeClass(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)

	_, err := k.RegisterERC721(ctx, &types.MsgConvertERC721{
		EvmContractAddress: "0x9999999999999999999999999999999999999999",
		EvmTokenIds:        []string{"1"},
		CosmosSender:       owner.String(),
		CosmosReceiver:     owner.String(),
		ClassId:            "kitty",
	})
	require.Error(t, err)
}

func TestConvertERC721_SelfDestructedPair(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)
	k.evmKeeper = &fakeEVMKeeper{accounts: map[common.Address]*statedb.Account{}}

	_, err := k.ConvertERC721(ctx, &types.MsgConvertERC721{
		ClassId:            "kitty",
		CosmosTokenIds:     []string{"nft1"},
		EvmContractAddress: "0x1111111111111111111111111111111111111111",
		EvmTokenIds:        []string{"1"},
		CosmosSender:       owner.String(),
		CosmosReceiver:     owner.String(),
	})
	require.ErrorIs(t, err, types.ErrInternalTokenPair)
}

func TestConvertNFT_SuccessMintsERC721(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)
	contract := "0x1111111111111111111111111111111111111111"
	receiver := "0x2222222222222222222222222222222222222222"

	res, err := k.ConvertNFT(ctx, &types.MsgConvertNFT{
		ClassId:            "kitty",
		CosmosTokenIds:     []string{"nft1"},
		EvmContractAddress: contract,
		EvmTokenIds:        []string{"1"},
		CosmosSender:       owner.String(),
		EvmReceiver:        receiver,
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotEmpty(t, k.GetNFTPairByContractTokenID(ctx, contract, "1"))

	got, err := k.nftKeeper.GetNFT(ctx, "kitty", "nft1")
	require.NoError(t, err)
	require.Equal(t, types.AccModuleAddress.String(), got.GetOwner().String())
}

func TestRefundPacketToken_MissingPair(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)
	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.ErrorIs(t, err, types.ErrTokenPairNotFound)
}

func TestQueryERC721DataByTokenID_DoesNotCommit(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)
	evm := k.evmKeeper.(*fakeEVMKeeper)
	contract := common.HexToAddress("0x1111111111111111111111111111111111111111")

	_, _ = k.QueryERC721DataByTokenID("tokenURI", ctx, contract, big.NewInt(1))

	require.Equal(t, 1, evm.applyCalls)
	require.False(t, evm.lastCommit)
}

func TestCallEVMWithData_UsesNonNilStateDB(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)
	evm := k.evmKeeper.(*fakeEVMKeeper)
	contract := common.HexToAddress("0x1111111111111111111111111111111111111111")

	_, err := k.CallEVMWithData(ctx, types.ModuleAddress, &contract, []byte{0x01}, false)
	require.NoError(t, err)
	require.Equal(t, 1, evm.applyCalls)
	require.NotNil(t, evm.lastStateDB)
}

func TestERC721StateDBKeeper_ModuleAccountSkipsBalanceWrite(t *testing.T) {
	base := authtypes.NewBaseAccountWithAddress(types.AccModuleAddress)
	ak := &moduleAccountKeeper{account: base}
	db := &moduleStateDBKeeper{
		current: statedb.NewAccount(7, new(uint256.Int), big.NewInt(0), evmtypes.EmptyCodeHash),
	}
	wrapper := erc721StateDBKeeper{
		Keeper:        db,
		accountKeeper: ak,
	}

	err := wrapper.SetAccount(sdk.Context{}, types.ModuleAddress, statedb.Account{
		Nonce:    8,
		Balance:  new(uint256.Int),
		CodeHash: evmtypes.EmptyCodeHash,
	})
	require.NoError(t, err)
	require.True(t, ak.setCalled)
	require.Equal(t, uint64(8), ak.account.GetSequence())
	require.Zero(t, db.nonModuleSetCalls)

	err = wrapper.SetAccount(sdk.Context{}, common.HexToAddress("0x2222222222222222222222222222222222222222"), statedb.Account{
		Nonce:    1,
		Balance:  new(uint256.Int),
		CodeHash: evmtypes.EmptyCodeHash,
	})
	require.NoError(t, err)
	require.Equal(t, 1, db.nonModuleSetCalls)
}

func bytes20(fill byte) []byte {
	b := make([]byte, 20)
	for i := range b {
		b[i] = fill
	}
	return b
}

type convertAccountKeeper struct{}

func (a *convertAccountKeeper) GetAccount(_ context.Context, _ sdk.AccAddress) sdk.AccountI {
	return nil
}
func (a *convertAccountKeeper) SetAccount(_ context.Context, _ sdk.AccountI) {}
func (a *convertAccountKeeper) GetModuleAddress(moduleName string) sdk.AccAddress {
	return authtypes.NewModuleAddress(moduleName)
}
func (a *convertAccountKeeper) AddressCodec() coreaddress.Codec {
	return codecAddress.NewBech32Codec("uptick")
}
func (a *convertAccountKeeper) GetSequence(_ context.Context, _ sdk.AccAddress) (uint64, error) {
	return 1, nil
}

type convertBankKeeper struct{}

func (b *convertBankKeeper) SpendableCoins(_ context.Context, _ sdk.AccAddress) sdk.Coins {
	return nil
}

type fakeEVMKeeper struct {
	stateDBKeeperStub

	accounts    map[common.Address]*statedb.Account
	lastCommit  bool
	lastStateDB *statedb.StateDB
	applyCalls  int
}

func (f *fakeEVMKeeper) GetParams(_ sdk.Context) evmtypes.Params { return evmtypes.Params{} }

func (f *fakeEVMKeeper) GetAccountWithoutBalance(_ sdk.Context, addr common.Address) *statedb.Account {
	if f.accounts == nil {
		return nil
	}
	return f.accounts[addr]
}

func (f *fakeEVMKeeper) EstimateGas(_ context.Context, _ *evmtypes.EthCallRequest) (*evmtypes.EstimateGasResponse, error) {
	return &evmtypes.EstimateGasResponse{Gas: 100000}, nil
}

func (f *fakeEVMKeeper) ApplyMessage(
	_ sdk.Context,
	stateDB *statedb.StateDB,
	_ core.Message,
	_ *tracing.Hooks,
	commit bool,
	_ bool,
	_ bool,
) (*evmtypes.MsgEthereumTxResponse, error) {
	f.lastCommit = commit
	f.lastStateDB = stateDB
	f.applyCalls++
	return &evmtypes.MsgEthereumTxResponse{}, nil
}

type stateDBKeeperStub struct{}

func (stateDBKeeperStub) GetAccount(sdk.Context, common.Address) *statedb.Account {
	return nil
}

func (stateDBKeeperStub) GetState(sdk.Context, common.Address, common.Hash) common.Hash {
	return common.Hash{}
}

func (stateDBKeeperStub) GetCode(sdk.Context, common.Hash) []byte {
	return nil
}

func (stateDBKeeperStub) GetCodeHash(sdk.Context, common.Address) common.Hash {
	return common.Hash{}
}

func (stateDBKeeperStub) ForEachStorage(sdk.Context, common.Address, func(common.Hash, common.Hash) bool) {
}

func (stateDBKeeperStub) SetAccount(sdk.Context, common.Address, statedb.Account) error {
	return nil
}

func (stateDBKeeperStub) DeleteState(sdk.Context, common.Address, common.Hash) {}

func (stateDBKeeperStub) SetState(sdk.Context, common.Address, common.Hash, []byte) {}

func (stateDBKeeperStub) DeleteCode(sdk.Context, []byte) {}

func (stateDBKeeperStub) SetCode(sdk.Context, []byte, []byte) {}

func (stateDBKeeperStub) DeleteAccount(sdk.Context, common.Address) error {
	return nil
}

func (stateDBKeeperStub) KVStoreKeys() map[string]*storetypes.KVStoreKey {
	return nil
}

type moduleAccountKeeper struct {
	account   sdk.AccountI
	setCalled bool
}

func (m *moduleAccountKeeper) GetAccount(context.Context, sdk.AccAddress) sdk.AccountI {
	return m.account
}

func (m *moduleAccountKeeper) SetAccount(_ context.Context, account sdk.AccountI) {
	m.account = account
	m.setCalled = true
}

func (m *moduleAccountKeeper) GetModuleAddress(moduleName string) sdk.AccAddress {
	return authtypes.NewModuleAddress(moduleName)
}

func (m *moduleAccountKeeper) GetSequence(context.Context, sdk.AccAddress) (uint64, error) {
	return 0, nil
}

type moduleStateDBKeeper struct {
	stateDBKeeperStub
	current           *statedb.Account
	nonModuleSetCalls int
}

func (m *moduleStateDBKeeper) GetAccount(sdk.Context, common.Address) *statedb.Account {
	return m.current
}

func (m *moduleStateDBKeeper) SetAccount(sdk.Context, common.Address, statedb.Account) error {
	m.nonModuleSetCalls++
	return nil
}

var _ types.EVMKeeper = (*fakeEVMKeeper)(nil)
var _ types.AccountKeeper = (*convertAccountKeeper)(nil)
var _ nft.AccountKeeper = (*convertAccountKeeper)(nil)
var _ nft.BankKeeper = (*convertBankKeeper)(nil)
