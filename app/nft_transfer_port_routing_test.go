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
// v10 removed the capability-based port to module binding and made the port
// router reject non-alphanumeric keys. The channels opened under the old name
// are still in the IBC core channel store, and core resolves the module from the
// packet's port, so losing the route breaks every packet on those channels -
// including the acknowledgement and timeout callbacks that unescrow the sender's
// NFT, which strands that NFT permanently.
//
// The route is registered under an alphanumeric key ("nft") that the core port
// keeper still maps onto "nft-transfer" through its substring fallback. That is
// indirect, so this test asserts the resolution the core actually performs
// (PortKeeper.Route) rather than the router's key set: a change in ibc-go that
// drops the fallback would otherwise fail silently, in the form of escrowed NFTs
// that can no longer be returned.
//
// The modules are compared by dynamic type rather than by value: they are keeper
// graphs with reference cycles, and testify's value diffing recurses through
// them until the stack overflows. Type identity is enough here - the question is
// which IBC stack answers the port, and the ICS-721 stack and the ICS-20 stack
// are different types.
func TestICS721LegacyPortStillResolves(t *testing.T) {
	app, _ := sharedTestApp(t)
	ports := app.IBCKeeper.PortKeeper

	// If the module's port ever reverts to the legacy name, the two lookups below
	// collapse into one and this test stops proving anything.
	require.NotEqual(t, ibcnfttransfertypes.PortID, keepers.LegacyNFTTransferPortID,
		"the module port and the legacy port must be distinct for a separate legacy route to mean anything")

	currentModule, ok := ports.Route(ibcnfttransfertypes.PortID)
	require.True(t, ok, "the ICS-721 module port %q must resolve", ibcnfttransfertypes.PortID)

	legacyModule, ok := ports.Route(keepers.LegacyNFTTransferPortID)
	require.True(t, ok,
		"packets on a pre-v0.4.0 %s/* channel must still resolve or their escrowed NFTs are stranded",
		keepers.LegacyNFTTransferPortID)

	// Both ports must reach the same stack: the EVM IBC middleware wrapping the
	// ICS-721 IBC module. Anything else would run the legacy channels' escrow,
	// mint and refund logic against the wrong state.
	require.Equal(t, fmt.Sprintf("%T", currentModule), fmt.Sprintf("%T", legacyModule),
		"the legacy ICS-721 port must resolve to the same IBC stack as the current one")

	// The ICS-20 transfer module owns a port that is a suffix of the legacy
	// ICS-721 port, so it competes for the same fallback lookup. It must keep its
	// own exact port, and the legacy ICS-721 port must not land on it.
	transferModule, ok := ports.Route(ibctransfertypes.PortID)
	require.True(t, ok, "the ICS-20 transfer port must keep resolving")
	require.NotEqual(t, fmt.Sprintf("%T", transferModule), fmt.Sprintf("%T", legacyModule),
		"the legacy ICS-721 port must not resolve to the ICS-20 transfer stack")
}
