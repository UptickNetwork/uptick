package keeper

import (
	"context"
	"encoding/json"
	"testing"

	coreaddress "cosmossdk.io/core/address"
	"cosmossdk.io/log"
	rootstore "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/nft"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codecAddress "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	collectionkeeper "github.com/UptickNetwork/uptick/x/collection/keeper"
	collectiontypes "github.com/UptickNetwork/uptick/x/collection/types"
	"github.com/UptickNetwork/uptick/x/cw721/types"
)

func setupConvertKeeper(t *testing.T) (Keeper, sdk.Context, sdk.AccAddress, string, *fakeWasmKeeper) {
	t.Helper()
	sdk.GetConfig().SetBech32PrefixForAccount("uptick", "uptickpub")

	ir := codectypes.NewInterfaceRegistry()
	collectiontypes.RegisterInterfaces(ir)
	types.RegisterInterfaces(ir)
	cdc := codec.NewProtoCodec(ir)

	db := dbm.NewMemDB()
	cms := rootstore.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	cwKey := storetypes.NewKVStoreKey(types.StoreKey)
	colKey := storetypes.NewKVStoreKey(collectiontypes.StoreKey)
	cms.MountStoreWithDB(cwKey, storetypes.StoreTypeIAVL, db)
	cms.MountStoreWithDB(colKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	ctx := sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
	ak := &convertAccountKeeper{}
	bk := &convertBankKeeper{}
	nftK := collectionkeeper.NewKeeper(cdc, runtime.NewKVStoreService(colKey), ak, bk)

	owner := sdk.AccAddress(bytes20(0x11))
	contract := sdk.AccAddress(bytes20(0x22)).String()
	wasm := newFakeWasmKeeper()

	k := Keeper{
		storeKey:      cwKey,
		cdc:           cdc,
		accountKeeper: ak,
		nftKeeper:     nftK,
		wasmKeeper:    wasm,
	}

	require.NoError(t, nftK.SaveDenom(ctx, "kitty", "Kitty", "", "KIT", owner, false, false, "", "", "", ""))
	require.NoError(t, nftK.SaveNFT(ctx, "kitty", "nft1", "Spot", "ipfs://nft", "", "", owner))

	pair := types.NewTokenPair(contract, "kitty")
	k.SetTokenPair(ctx, pair)
	k.SetClassMap(ctx, pair.ClassId, pair.GetID())
	k.SetCW721Map(ctx, pair.Cw721Address, pair.GetID())

	return k, ctx, owner, contract, wasm
}

func TestConvertNFT_SuccessMintsCW721(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)
	receiver := sdk.AccAddress(bytes20(0x33)).String()

	_, err := k.ConvertNFT(ctx, &types.MsgConvertNFT{
		ClassId:         "kitty",
		NftIds:          []string{"nft1"},
		ContractAddress: contract,
		TokenIds:        []string{"1"},
		Sender:          owner.String(),
		Receiver:        receiver,
	})
	require.NoError(t, err)
	require.NotEmpty(t, k.GetNFTPairByContractTokenID(ctx, contract, "1"))
	require.Equal(t, receiver, wasm.ownerOf(contract, "1"))

	got, err := k.nftKeeper.GetNFT(ctx, "kitty", "nft1")
	require.NoError(t, err)
	require.Equal(t, types.AccModuleAddress.String(), got.GetOwner().String())
}

func TestConvertCW721_SuccessMintsNFT(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)
	receiver := sdk.AccAddress(bytes20(0x33)).String()
	wasm.setOwner(contract, "1", owner.String())
	wasm.setURI(contract, "1", "ipfs://cw")

	_, err := k.ConvertCW721(ctx, &types.MsgConvertCW721{
		ContractAddress: contract,
		TokenIds:        []string{"1"},
		ClassId:         "kitty",
		NftIds:          []string{"nft2"},
		Sender:          owner.String(),
		Receiver:        receiver,
	})
	require.NoError(t, err)
	require.Equal(t, types.AccModuleAddress.String(), wasm.ownerOf(contract, "1"))

	got, err := k.nftKeeper.GetNFT(ctx, "kitty", "nft2")
	require.NoError(t, err)
	require.Equal(t, receiver, got.GetOwner().String())
	require.Equal(t, "ipfs://cw", got.GetURI())
}

func TestRegisterCW721_RejectsExistingNativeClass(t *testing.T) {
	k, ctx, owner, contract, _ := setupConvertKeeper(t)

	_, err := k.RegisterCW721(ctx, &types.MsgConvertCW721{
		ContractAddress: contract,
		TokenIds:        []string{"1"},
		Sender:          owner.String(),
		Receiver:        owner.String(),
		ClassId:         "kitty",
	})
	require.Error(t, err)
}

func TestConvertCW721_NotOwner(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)
	wasm.setOwner(contract, "1", sdk.AccAddress(bytes20(0x44)).String())

	_, err := k.ConvertCW721(ctx, &types.MsgConvertCW721{
		ContractAddress: contract,
		TokenIds:        []string{"1"},
		ClassId:         "kitty",
		NftIds:          []string{"nft2"},
		Sender:          owner.String(),
		Receiver:        owner.String(),
	})
	require.ErrorIs(t, err, errortypes.ErrUnauthorized)
}

func TestRefundPacketToken_Success(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)
	receiver := sdk.AccAddress(bytes20(0x33)).String()

	_, err := k.ConvertNFT(ctx, &types.MsgConvertNFT{
		ClassId:         "kitty",
		NftIds:          []string{"nft1"},
		ContractAddress: contract,
		TokenIds:        []string{"1"},
		Sender:          owner.String(),
		Receiver:        receiver,
	})
	require.NoError(t, err)

	k.SetCwAddressByContractTokenId(ctx, contract, "1", owner.String())
	// An outbound IBC transfer escrows the CW721 in the module account (see
	// convertWasm2Cosmos); the refund path only applies to an escrowed token.
	// Leaving the token with the receiver would make a real CW721 contract
	// reject the module's transfer_nft — the failure that strands the packet.
	wasm.setOwner(contract, "1", types.AccModuleAddress.String())

	require.NoError(t, k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	}))
	require.Equal(t, owner.String(), wasm.ownerOf(contract, "1"))
	_, err = k.nftKeeper.GetNFT(ctx, "kitty", "nft1")
	require.Error(t, err)
	require.Empty(t, k.GetTokenUIDPairByNFTUID(ctx, types.CreateNFTUID("kitty", "nft1")))
}

func TestConvertNFT_ClassIdEqualToContractDoesNotHijack(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)

	require.NoError(t, k.nftKeeper.SaveDenom(ctx, contract, "Collide", "", "COL", owner, false, false, "", "", "", ""))
	require.NoError(t, k.nftKeeper.SaveNFT(ctx, contract, "nftx", "X", "", "", "", owner))

	_, err := k.ConvertNFT(ctx, &types.MsgConvertNFT{
		ClassId:         contract,
		NftIds:          []string{"nftx"},
		ContractAddress: contract,
		TokenIds:        []string{"99"},
		Sender:          owner.String(),
		Receiver:        owner.String(),
	})
	require.ErrorIs(t, err, types.ErrContractAddressNotCorrect)

	pair, err := k.GetPairByCW721(ctx, contract)
	require.NoError(t, err)
	require.Equal(t, "kitty", pair.ClassId)
	require.Empty(t, wasm.ownerOf(contract, "99"))
}

func TestGetPairByClass_IgnoresContractMap(t *testing.T) {
	k, ctx, _, contract, _ := setupConvertKeeper(t)

	_, err := k.GetPairByClass(ctx, contract)
	require.ErrorIs(t, err, types.ErrTokenPairNotFound)

	pair, err := k.GetPairByCW721(ctx, contract)
	require.NoError(t, err)
	require.Equal(t, "kitty", pair.ClassId)
}

func TestRegisterNFT_RejectsAlreadyRegisteredContract(t *testing.T) {
	k, ctx, owner, contract, _ := setupConvertKeeper(t)

	_, err := k.RegisterNFT(ctx, &types.MsgConvertNFT{
		ClassId:         "otherclass",
		ContractAddress: contract,
		NftIds:          []string{"nft1"},
		Sender:          owner.String(),
		Receiver:        owner.String(),
	})
	require.ErrorIs(t, err, types.ErrTokenPairAlreadyExists)
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
func (a *convertAccountKeeper) GetModuleAddress(moduleName string) sdk.AccAddress {
	return authtypes.NewModuleAddress(moduleName)
}
func (a *convertAccountKeeper) AddressCodec() coreaddress.Codec {
	return codecAddress.NewBech32Codec("uptick")
}

type convertBankKeeper struct{}

func (b *convertBankKeeper) SpendableCoins(_ context.Context, _ sdk.AccAddress) sdk.Coins {
	return nil
}

type fakeWasmKeeper struct {
	owners map[string]string
	uris   map[string]string
}

func newFakeWasmKeeper() *fakeWasmKeeper {
	return &fakeWasmKeeper{
		owners: make(map[string]string),
		uris:   make(map[string]string),
	}
}

func (f *fakeWasmKeeper) key(contract, tokenID string) string {
	return contract + "/" + tokenID
}

func (f *fakeWasmKeeper) setOwner(contract, tokenID, owner string) {
	f.owners[f.key(contract, tokenID)] = owner
}

func (f *fakeWasmKeeper) setURI(contract, tokenID, uri string) {
	f.uris[f.key(contract, tokenID)] = uri
}

func (f *fakeWasmKeeper) ownerOf(contract, tokenID string) string {
	return f.owners[f.key(contract, tokenID)]
}

func (f *fakeWasmKeeper) StoreCode(_ context.Context, _ *wasmtypes.MsgStoreCode) (*wasmtypes.MsgStoreCodeResponse, error) {
	return &wasmtypes.MsgStoreCodeResponse{CodeID: 1}, nil
}

func (f *fakeWasmKeeper) InstantiateContract(_ context.Context, msg *wasmtypes.MsgInstantiateContract) (*wasmtypes.MsgInstantiateContractResponse, error) {
	return &wasmtypes.MsgInstantiateContractResponse{Address: msg.Sender}, nil
}

func (f *fakeWasmKeeper) ExecuteContract(_ context.Context, msg *wasmtypes.MsgExecuteContract) (*wasmtypes.MsgExecuteContractResponse, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(msg.Msg, &payload); err != nil {
		return nil, err
	}
	if raw, ok := payload["mint"]; ok {
		var mint MintInfoData
		if err := json.Unmarshal(raw, &mint); err != nil {
			return nil, err
		}
		f.setOwner(msg.Contract, mint.TokenId, mint.Owner)
		f.setURI(msg.Contract, mint.TokenId, mint.TokenUri)
		return &wasmtypes.MsgExecuteContractResponse{}, nil
	}
	if raw, ok := payload["transfer_nft"]; ok {
		var xfer TransferNftData
		if err := json.Unmarshal(raw, &xfer); err != nil {
			return nil, err
		}
		f.setOwner(msg.Contract, xfer.TokenId, xfer.Recipient)
		return &wasmtypes.MsgExecuteContractResponse{}, nil
	}
	return &wasmtypes.MsgExecuteContractResponse{}, nil
}

func (f *fakeWasmKeeper) SmartContractState(_ context.Context, req *wasmtypes.QuerySmartContractStateRequest) (*wasmtypes.QuerySmartContractStateResponse, error) {
	var q map[string]json.RawMessage
	if err := json.Unmarshal(req.QueryData, &q); err != nil {
		return nil, err
	}
	if raw, ok := q["all_nft_info"]; ok {
		var inner struct {
			TokenId string `json:"token_id"`
		}
		if err := json.Unmarshal(raw, &inner); err != nil {
			return nil, err
		}
		bz, err := json.Marshal(AllNftInfo{
			Access: AllNftAccess{Owner: f.ownerOf(req.Address, inner.TokenId)},
			Info:   AllNftData{TokenUri: f.uris[f.key(req.Address, inner.TokenId)]},
		})
		if err != nil {
			return nil, err
		}
		return &wasmtypes.QuerySmartContractStateResponse{Data: bz}, nil
	}
	if _, ok := q["contract_info"]; ok {
		bz, err := json.Marshal(ContractInfoData{Name: "Kitty", Symbol: "KIT"})
		if err != nil {
			return nil, err
		}
		return &wasmtypes.QuerySmartContractStateResponse{Data: bz}, nil
	}
	return &wasmtypes.QuerySmartContractStateResponse{}, nil
}

var _ types.WasmKeeper = (*fakeWasmKeeper)(nil)
var _ types.AccountKeeper = (*convertAccountKeeper)(nil)
var _ nft.AccountKeeper = (*convertAccountKeeper)(nil)
var _ nft.BankKeeper = (*convertBankKeeper)(nil)
