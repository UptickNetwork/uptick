package keeper

import (
	"testing"

	nfttransfertypes "github.com/bianjieai/nft-transfer/types"
	"github.com/stretchr/testify/require"
)

func TestGetVoucherClassID(t *testing.T) {
	k := Keeper{}
	port := "transfer"
	channel := "channel-7"
	classID := "artwork-001"

	got := k.GetVoucherClassID(port, channel, classID)

	classPrefix := nfttransfertypes.GetClassPrefix(port, channel)
	expected := nfttransfertypes.ParseClassTrace(classPrefix + classID).IBCClassID()
	require.Equal(t, expected, got)
}
