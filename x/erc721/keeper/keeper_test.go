package keeper

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	nftkeeper "github.com/UptickNetwork/uptick/x/collection/keeper"
	"github.com/UptickNetwork/uptick/x/erc721/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	ibcnfttransferkeeper "github.com/bianjieai/nft-transfer/keeper"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	"github.com/stretchr/testify/require"
)

func TestNewKeeper_erc721(t *testing.T) {
	// NewKeeper takes: storeKey, cdc, ak, nk, ek, ik
	var sk storetypes.StoreKey = nil
	var cdc codec.BinaryCodec = nil
	var ak types.AccountKeeper = nil
	var nk nftkeeper.Keeper
	var ek types.EVMKeeper = nil
	var ik ibcnfttransferkeeper.Keeper

	k := NewKeeper(sk, cdc, ak, nk, ek, ik)
	require.NotNil(t, &k)
	require.Nil(t, k.storeKey)
}

func TestSetICS4Wrapper_FirstCall(t *testing.T) {
	var sk storetypes.StoreKey = nil
	var cdc codec.BinaryCodec = nil
	var ak types.AccountKeeper = nil
	var nk nftkeeper.Keeper
	var ek types.EVMKeeper = nil
	var ik ibcnfttransferkeeper.Keeper

	k := NewKeeper(sk, cdc, ak, nk, ek, ik)

	// First call should succeed
	require.NotPanics(t, func() {
		k.SetICS4Wrapper(nil)
	})
}

func TestSetICS4Wrapper_PanicsIfAlreadySet(t *testing.T) {
	var sk storetypes.StoreKey = nil
	var cdc codec.BinaryCodec = nil
	var ak types.AccountKeeper = nil
	var nk nftkeeper.Keeper
	var ek types.EVMKeeper = nil
	var ik ibcnfttransferkeeper.Keeper

	k := NewKeeper(sk, cdc, ak, nk, ek, ik)
	// Pre-set to non-nil
	k.ics4Wrapper = &mockICS4Wrapper{}

	require.Panics(t, func() {
		k.SetICS4Wrapper(nil)
	})
}

func TestGetVoucherClassID_ValidInput(t *testing.T) {
	k := setupBasicKeeper(t)

	voucherClassID := k.GetVoucherClassID(ibcnfttransfertypes.PortID, "channel-0", "class-1")
	require.NotEmpty(t, voucherClassID)
	require.Contains(t, voucherClassID, "ibc/")
}

func TestGetVoucherClassID_Deterministic(t *testing.T) {
	k := setupBasicKeeper(t)

	result1 := k.GetVoucherClassID(ibcnfttransfertypes.PortID, "channel-0", "class-1")
	result2 := k.GetVoucherClassID(ibcnfttransfertypes.PortID, "channel-0", "class-1")
	require.Equal(t, result1, result2)
}

func TestGetVoucherClassID_DifferentChannels(t *testing.T) {
	k := setupBasicKeeper(t)

	a := k.GetVoucherClassID(ibcnfttransfertypes.PortID, "channel-0", "class-1")
	b := k.GetVoucherClassID(ibcnfttransfertypes.PortID, "channel-1", "class-1")
	require.NotEqual(t, a, b)
}

func TestGetVoucherClassID_DifferentPorts(t *testing.T) {
	k := setupBasicKeeper(t)

	a := k.GetVoucherClassID(ibcnfttransfertypes.PortID, "channel-0", "class-1")
	b := k.GetVoucherClassID("transfer", "channel-0", "class-1")
	require.NotEqual(t, a, b)
}

func TestGetVoucherClassID_DifferentClassIDs(t *testing.T) {
	k := setupBasicKeeper(t)

	a := k.GetVoucherClassID(ibcnfttransfertypes.PortID, "channel-0", "class-1")
	b := k.GetVoucherClassID(ibcnfttransfertypes.PortID, "channel-0", "class-2")
	require.NotEqual(t, a, b)
}

func TestGetVoucherClassID_EmptyChannel(t *testing.T) {
	k := setupBasicKeeper(t)

	result := k.GetVoucherClassID(ibcnfttransfertypes.PortID, "", "class-empty")
	require.NotEmpty(t, result)
}

func TestGetVoucherClassID_EmptyPort(t *testing.T) {
	k := setupBasicKeeper(t)

	result := k.GetVoucherClassID("", "channel-0", "class-empty")
	require.NotEmpty(t, result)
}

func setupBasicKeeper(t *testing.T) Keeper {
	t.Helper()
	var sk storetypes.StoreKey = nil
	var cdc codec.BinaryCodec = nil
	var ak types.AccountKeeper = nil
	var nk nftkeeper.Keeper
	var ek types.EVMKeeper = nil
	var ik ibcnfttransferkeeper.Keeper

	return NewKeeper(sk, cdc, ak, nk, ek, ik)
}

// mockICS4Wrapper is a minimal mock for testing
type mockICS4Wrapper struct {
	capturedPort    string
	capturedChannel string
	capturedData    []byte
}

func (m *mockICS4Wrapper) WriteAcknowledgement(
	ctx sdk.Context,
	packet ibcexported.PacketI,
	acknowledgement ibcexported.Acknowledgement,
) error {
	return nil
}

func (m *mockICS4Wrapper) SendPacket(
	ctx sdk.Context,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	m.capturedPort = sourcePort
	m.capturedChannel = sourceChannel
	m.capturedData = data
	return 0, nil
}

func (m *mockICS4Wrapper) GetAppVersion(
	ctx sdk.Context,
	portID string,
	channelID string,
) (string, bool) {
	return "", false
}
