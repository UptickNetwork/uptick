package app

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"

	"github.com/UptickNetwork/uptick/app/keepers"
)

// TestICS721LegacyPortStillResolves pins the routing of the pre-v0.4.0 ICS-721
// port, nft-transfer.
//
// v0.4.0 renamed the module's port to "nonfungibletokentransfer" because ibc-go
// v10 resolves the module from the packet's port alone and its router rejects
// non-alphanumeric keys. Channels opened under the old name are still in the
// core channel store, so losing the route breaks every packet on them -
// including the acknowledgement and timeout callbacks that unescrow the sender's
// NFT, stranding it permanently.
//
// The route is registered under the alphanumeric key "nft", which the core port
// keeper still maps onto "nft-transfer" through its substring fallback. That is
// indirect, so the test asserts the resolution the core actually performs
// (PortKeeper.Route) rather than the router's key set: an ibc-go change that
// drops the fallback would otherwise surface only as stranded escrowed NFTs.
//
// The modules are compared by dynamic type, not value: they are keeper graphs
// with reference cycles, and testify's value diffing recurses until the stack
// overflows. Type identity settles the question being asked, which is which IBC
// stack answers the port.
func TestICS721LegacyPortStillResolves(t *testing.T) {
	app, _ := sharedTestApp(t)
	ports := app.IBCKeeper.PortKeeper

	// If the port ever reverts to the legacy name, the two lookups below collapse
	// into one and this test stops proving anything.
	require.NotEqual(t, ibcnfttransfertypes.PortID, keepers.LegacyNFTTransferPortID,
		"the module port and the legacy port must be distinct for a separate legacy route to mean anything")

	currentModule, ok := ports.Route(ibcnfttransfertypes.PortID)
	require.True(t, ok, "the ICS-721 module port %q must resolve", ibcnfttransfertypes.PortID)

	legacyModule, ok := ports.Route(keepers.LegacyNFTTransferPortID)
	require.True(t, ok,
		"packets on a pre-v0.4.0 %s/* channel must still resolve or their escrowed NFTs are stranded",
		keepers.LegacyNFTTransferPortID)

	// Both ports must reach the same stack - the EVM IBC middleware wrapping the
	// ICS-721 module - or the legacy channels' escrow, mint and refund logic runs
	// against the wrong state.
	require.Equal(t, fmt.Sprintf("%T", currentModule), fmt.Sprintf("%T", legacyModule),
		"the legacy ICS-721 port must resolve to the same IBC stack as the current one")

	// The ICS-20 port "transfer" is a suffix of "nft-transfer", so it competes for
	// the same fallback lookup; the legacy ICS-721 port must not land on it.
	transferModule, ok := ports.Route(ibctransfertypes.PortID)
	require.True(t, ok, "the ICS-20 transfer port must keep resolving")
	require.NotEqual(t, fmt.Sprintf("%T", transferModule), fmt.Sprintf("%T", legacyModule),
		"the legacy ICS-721 port must not resolve to the ICS-20 transfer stack")
}
