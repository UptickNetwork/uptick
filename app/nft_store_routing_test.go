package app

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	cosmosnftkeeper "cosmossdk.io/x/nft/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	nfttypes "github.com/UptickNetwork/uptick/x/collection/types"
)

// TestNFTStateRoutesToCollectionStore pins where the NFT module keeps its KV
// state, and that module-account permissions are not part of that answer.
//
// An external audit read the `cosmosnft.ModuleName: nil` row of maccPerms
// (app/app.go) as "the NFT keeper is not wired to a store" and rated it P0. It
// is not: maccPerms grants module-account capabilities, while KV routing is
// fixed at keeper construction - app/keepers/keepers.go builds the NFT keeper
// with NewKVStoreService(keys[nfttypes.StoreKey]), and that map entry is the
// x/collection store, whose name is "collection".
//
// The failure this guards against is silent. Passing the wrong key at
// construction still starts the node and still answers every keeper call; the
// data simply lands in another store, which shows up much later as an export
// that finds nothing. So the wiring is asserted end to end - write through the
// keeper, read the raw bytes back out of both candidate stores - rather than by
// re-reading the construction site.
func TestNFTStateRoutesToCollectionStore(t *testing.T) {
	app, parent := sharedTestApp(t)

	// The shared context is a process-wide singleton, so this test writes into
	// a cache it never flushes. The probe class stays out of every other test's
	// starting state.
	ctx, _ := parent.CacheContext()

	collectionKey := app.GetKey(nfttypes.StoreKey)
	authKey := app.GetKey(authtypes.StoreKey)
	require.Equal(t, "collection", collectionKey.Name(),
		"x/collection's StoreKey is the name the keeper store is registered under")
	require.NotEqual(t, collectionKey.Name(), authKey.Name())

	const classID = "storeroutingprobe"
	creator := sdk.AccAddress(bytes.Repeat([]byte{0xAB}, 20))

	require.NoError(t, app.NFTKeeper.SaveDenom(
		ctx, classID, "Store Routing Probe", "", "SRP", creator, false, false, "", "", "", "",
	))

	// x/nft stores a class at 0x01 || classID (keeper/keys.go: classStoreKey).
	classKey := append(bytes.Clone(cosmosnftkeeper.ClassKey), []byte(classID)...)

	require.NotEmpty(t, ctx.KVStore(collectionKey).Get(classKey),
		"the class must land in the collection store: the keeper's KVStoreService is built from keys[nfttypes.StoreKey]")

	// Sanity: the keeper reads back what it just wrote, so the bytes above are
	// the keeper's own state and not some other module's byproduct.
	denom, err := app.NFTKeeper.GetDenomInfo(ctx, classID)
	require.NoError(t, err)
	require.Equal(t, "Store Routing Probe", denom.Name)

	require.Empty(t, ctx.KVStore(authKey).Get(classKey),
		"the auth store must hold no NFT state; a maccPerms row grants module-account permissions and does not route KV writes")
}
