package v2

import (
	"strings"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

func precheckTestSetup(t *testing.T) (sdk.Context, storetypes.StoreKey, codec.Codec) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey("transient")
	ctx := sdktestutil.DefaultContext(key, tkey)
	return ctx, key, codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
}

func TestPrecheckLegacyStoreClean(t *testing.T) {
	ctx, key, cdc := precheckTestSetup(t)
	store := ctx.KVStore(key)

	denom := types.Denom{Id: "denom-1", Name: "Denom One", Creator: sdk.AccAddress([]byte("creator-addr-1")).String()}
	denomBz, err := cdc.Marshal(&denom)
	require.NoError(t, err)
	store.Set(KeyDenom("denom-1"), denomBz)

	owner := sdk.AccAddress([]byte("owner-addr-1")).String()
	nftBz, err := cdc.Marshal(&types.BaseNFT{Id: "token-1", Owner: owner})
	require.NoError(t, err)
	store.Set(KeyNFT("denom-1", "token-1"), nftBz)

	problems := PrecheckLegacyStore(ctx, key, cdc)
	require.Empty(t, problems, "clean legacy records must produce no problems")
}

func TestPrecheckLegacyStoreDirtyRecords(t *testing.T) {
	ctx, key, cdc := precheckTestSetup(t)
	store := ctx.KVStore(key)

	// Dirty denom: invalid bech32 creator.
	denomBz, err := cdc.Marshal(&types.Denom{Id: "denom-bad", Creator: "not-bech32"})
	require.NoError(t, err)
	store.Set(KeyDenom("denom-bad"), denomBz)

	// Dirty NFT: invalid bech32 owner.
	nftBz, err := cdc.Marshal(&types.BaseNFT{Id: "token-bad", Owner: "also-not-bech32"})
	require.NoError(t, err)
	store.Set(KeyNFT("denom-bad", "token-bad"), nftBz)

	// Corrupt value under the NFT prefix (unmarshal failure).
	store.Set(KeyNFT("denom-bad", "token-corrupt"), []byte{0xde, 0xad, 0xbe, 0xef})

	problems := PrecheckLegacyStore(ctx, key, cdc)
	require.Len(t, problems, 3, "all dirty records must be collected, not just the first")

	kinds := map[string]int{}
	for _, p := range problems {
		kinds[p.Kind]++
	}
	require.Equal(t, 1, kinds["denom"])
	require.Equal(t, 2, kinds["nft"])

	report := FormatProblems(problems)
	require.True(t, strings.Contains(report, "3 record(s)"), "report header must count problems")
	require.True(t, strings.Contains(report, "not-bech32"), "report must include the offending value")
}

func TestPrecheckLegacyStoreEmpty(t *testing.T) {
	ctx, key, cdc := precheckTestSetup(t)
	require.Empty(t, PrecheckLegacyStore(ctx, key, cdc))
}
