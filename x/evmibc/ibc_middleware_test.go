package evmibc

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
)

func TestOnRecvPacket_UnmarshalError(t *testing.T) {
	im := IBCMiddleware{}
	ack := im.OnRecvPacket(sdk.Context{}, "", channeltypes.Packet{Data: []byte("not-json")}, nil)
	require.False(t, ack.Success())
}

func TestOnTimeoutPacket_UnmarshalError(t *testing.T) {
	im := IBCMiddleware{}
	err := im.OnTimeoutPacket(sdk.Context{}, "", channeltypes.Packet{Data: []byte("not-json")}, nil)
	require.Error(t, err)
}
