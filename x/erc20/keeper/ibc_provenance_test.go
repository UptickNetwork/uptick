package keeper_test

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/erc20/keeper"
	erc20types "github.com/UptickNetwork/uptick/x/erc20/types"
)

func newProvenanceKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(erc20types.StoreKey)
	paramsKey := storetypes.NewKVStoreKey("params")
	tkey := storetypes.NewTransientStoreKey("transient_test")
	tParamsKey := storetypes.NewTransientStoreKey("params_transient")

	ctx := testutil.DefaultContextWithKeys(
		map[string]*storetypes.KVStoreKey{
			erc20types.StoreKey: storeKey,
			"params":            paramsKey,
		},
		map[string]*storetypes.TransientStoreKey{
			"transient_test":   tkey,
			"params_transient": tParamsKey,
		},
		nil,
	)

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	legacyAmino := codec.NewLegacyAmino()
	ps := paramtypes.NewSubspace(cdc, legacyAmino, paramsKey, tParamsKey, erc20types.ModuleName)

	k := keeper.NewKeeper(cdc, storeKey, ps, nil, nil, nil)
	return *k, ctx
}

func TestIBCTransferProvenanceLifecycle(t *testing.T) {
	k, ctx := newProvenanceKeeper(t)

	packet := channeltypes.Packet{
		Sequence:           3,
		SourcePort:         "transfer",
		SourceChannel:      "channel-0",
		DestinationPort:    "transfer",
		DestinationChannel: "channel-1",
	}
	data := transfertypes.FungibleTokenPacketData{
		Sender: "uptick1sender",
		Denom:  "ibc/ABC",
		Amount: "100",
	}

	require.False(t, k.HasIBCTransferProvenance(ctx, packet, data))

	k.SetIBCTransferProvenance(ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence, data.Sender, data.Denom, data.Amount)
	require.True(t, k.HasIBCTransferProvenance(ctx, packet, data))

	require.True(t, k.ConsumeIBCTransferProvenance(ctx, packet, data))
	require.False(t, k.HasIBCTransferProvenance(ctx, packet, data))
}
