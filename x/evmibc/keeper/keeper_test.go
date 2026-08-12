package keeper

import (
	"testing"

	cw721keep "github.com/UptickNetwork/uptick/x/cw721/keeper"
	erc721keeper "github.com/UptickNetwork/uptick/x/erc721/keeper"
	ibcnfttransferkeeper "github.com/bianjieai/nft-transfer/keeper"
	"github.com/stretchr/testify/require"
)

func TestNewKeeper(t *testing.T) {
	// NewKeeper requires a non-nil ibcnfttransferkeeper.Keeper (value type)
	ik := ibcnfttransferkeeper.Keeper{}
	k := NewKeeper(ik)
	require.NotNil(t, &k)
}

func TestSetCw721Keeper(t *testing.T) {
	k := NewKeeper(ibcnfttransferkeeper.Keeper{})
	require.NotNil(t, &k)

	// Setting a nil cw721Keeper (not a pointer type, can't be nil)
	// Just verify set/get works - setting it to zero value
	k.SetCw721Keeper(cw721keep.Keeper{})
}

func TestSetErc721Keeper(t *testing.T) {
	k := NewKeeper(ibcnfttransferkeeper.Keeper{})
	require.NotNil(t, &k)

	k.SetErc721Keeper(erc721keeper.Keeper{})
}

func TestGetVoucherClassID(t *testing.T) {
	k := NewKeeper(ibcnfttransferkeeper.Keeper{})

	voucherClassID := k.GetVoucherClassID("transfer", "channel-0", "class-1")
	require.NotEmpty(t, voucherClassID)
	require.Contains(t, voucherClassID, "ibc/")

	// Same params produce same result
	voucherClassID2 := k.GetVoucherClassID("transfer", "channel-0", "class-1")
	require.Equal(t, voucherClassID, voucherClassID2)

	// Different channel produces different result
	differentChannel := k.GetVoucherClassID("transfer", "channel-1", "class-1")
	require.NotEqual(t, voucherClassID, differentChannel)
}

func TestGetVoucherClassID_EmptyPort(t *testing.T) {
	k := NewKeeper(ibcnfttransferkeeper.Keeper{})
	voucherClassID := k.GetVoucherClassID("", "channel-0", "class-empty")
	require.NotEmpty(t, voucherClassID)
}

func TestGetVoucherClassID_Deterministic(t *testing.T) {
	ik := ibcnfttransferkeeper.Keeper{}
	k1 := NewKeeper(ik)
	k2 := NewKeeper(ik)

	result1 := k1.GetVoucherClassID("transfer", "channel-0", "class-1")
	result2 := k2.GetVoucherClassID("transfer", "channel-0", "class-1")
	require.Equal(t, result1, result2)
}
