package keeper_test

import (
	"context"
	"testing"

	coreaddress "cosmossdk.io/core/address"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	rootstore "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codecAddress "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/keeper"
	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func init() {
	sdk.GetConfig().SetBech32PrefixForAccount("uptick", "uptickpub")
}

type mockBank struct {
	balances map[string]math.Int
}

func newMockBank() *mockBank {
	return &mockBank{balances: map[string]math.Int{}}
}

func (b *mockBank) key(addr sdk.AccAddress, denom string) string {
	return addr.String() + "/" + denom
}

func (b *mockBank) GetBalance(_ context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	v, ok := b.balances[b.key(addr, denom)]
	if !ok {
		return sdk.NewCoin(denom, math.ZeroInt())
	}
	return sdk.NewCoin(denom, v)
}

func (b *mockBank) SendCoinsFromAccountToModule(_ context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	mod := authtypes.NewModuleAddress(recipientModule)
	for _, c := range amt {
		sk := b.key(senderAddr, c.Denom)
		cur, ok := b.balances[sk]
		if !ok || cur.LT(c.Amount) {
			return types.ErrPoolInsufficient
		}
		b.balances[sk] = cur.Sub(c.Amount)
		mk := b.key(mod, c.Denom)
		mv, ok := b.balances[mk]
		if !ok {
			mv = math.ZeroInt()
		}
		b.balances[mk] = mv.Add(c.Amount)
	}
	return nil
}

func (b *mockBank) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	mod := authtypes.NewModuleAddress(senderModule)
	for _, c := range amt {
		mk := b.key(mod, c.Denom)
		cur, ok := b.balances[mk]
		if !ok || cur.LT(c.Amount) {
			return types.ErrPoolInsufficient
		}
		b.balances[mk] = cur.Sub(c.Amount)
		rk := b.key(recipientAddr, c.Denom)
		rv, ok := b.balances[rk]
		if !ok {
			rv = math.ZeroInt()
		}
		b.balances[rk] = rv.Add(c.Amount)
	}
	return nil
}

func (b *mockBank) fundModule(module, denom string, amount math.Int) {
	mod := authtypes.NewModuleAddress(module)
	b.balances[b.key(mod, denom)] = amount
}

func (b *mockBank) fundAccount(addr sdk.AccAddress, denom string, amount math.Int) {
	b.balances[b.key(addr, denom)] = amount
}

type mockAccountKeeper struct {
	codec coreaddress.Codec
}

func (m mockAccountKeeper) AddressCodec() coreaddress.Codec { return m.codec }
func (m mockAccountKeeper) GetModuleAddress(name string) sdk.AccAddress {
	return authtypes.NewModuleAddress(name)
}
func (m mockAccountKeeper) GetModuleAccount(_ context.Context, name string) sdk.ModuleAccountI {
	return authtypes.NewEmptyModuleAccount(name)
}

type testEnv struct {
	k      keeper.Keeper
	ctx    sdk.Context
	bank   *mockBank
	server types.MsgServer
	auth   sdk.AccAddress
	owner  sdk.AccAddress
	sub    sdk.AccAddress
	payer  sdk.AccAddress
}

func setupTest(t *testing.T) testEnv {
	t.Helper()
	ir := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(ir)
	cdc := codec.NewProtoCodec(ir)

	db := dbm.NewMemDB()
	cms := rootstore.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	key := storetypes.NewKVStoreKey(types.StoreKey)
	cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 1}, false, log.NewNopLogger())
	bank := newMockBank()
	ak := mockAccountKeeper{codec: codecAddress.NewBech32Codec("uptick")}
	k := keeper.NewKeeper(cdc, key, bank, ak)
	params := types.DefaultParams()
	auth := sdk.AccAddress(bytes20(0x01))
	owner := sdk.AccAddress(bytes20(0x02))
	sub := sdk.AccAddress(bytes20(0x03))
	payer := sdk.AccAddress(bytes20(0x04))
	params.Authority = auth.String()
	params.PauseAuthority = auth.String() // P0 default: same as authority; tests may split
	require.NoError(t, k.SetParams(ctx, params))
	bank.fundModule(types.ModuleName, params.ChallengeBondDenom, math.NewInt(1_000_000_000))

	return testEnv{
		k:      k,
		ctx:    ctx,
		bank:   bank,
		server: keeper.NewMsgServerImpl(k),
		auth:   auth,
		owner:  owner,
		sub:    sub,
		payer:  payer,
	}
}

func bytes20(seed byte) []byte {
	b := make([]byte, 20)
	for i := range b {
		b[i] = seed
	}
	return b
}

func hash32(seed byte) []byte {
	h := make([]byte, 32)
	for i := range h {
		h[i] = seed + byte(i)
	}
	return h
}

func (e testEnv) withHeight(h int64) testEnv {
	e.ctx = e.ctx.WithBlockHeight(h)
	return e
}
