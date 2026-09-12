package main

import (
	"bytes"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	v2 "github.com/UptickNetwork/uptick/x/collection/migrations/v2"
	collectiontypes "github.com/UptickNetwork/uptick/x/collection/types"
)

// precheckCmdStore mirrors the in-memory store setup the migration precheck's
// own unit tests use, so the command's core is exercised against a real KV
// store without building an application.
func precheckCmdStore(t *testing.T) (sdk.Context, storetypes.StoreKey, codec.Codec) {
	t.Helper()
	key := storetypes.NewKVStoreKey(collectiontypes.StoreKey)
	tkey := storetypes.NewTransientStoreKey("transient")
	ctx := sdktestutil.DefaultContext(key, tkey)
	return ctx, key, codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
}

// TestRunCollectionPrecheckClean pins the "clean store continues" half: a store
// with no offending legacy records must let the command succeed and print the
// confirmation.
func TestRunCollectionPrecheckClean(t *testing.T) {
	ctx, key, cdc := precheckCmdStore(t)
	var out bytes.Buffer

	require.NoError(t, runCollectionPrecheck(ctx, key, cdc, &out))
	require.Contains(t, out.String(), "clean")
}

// TestRunCollectionPrecheckDirtyReportsEveryRecord pins the "dirty store aborts
// with the FULL report" half: unlike v2.Migrate, which returns on the first bad
// record, the command must surface EVERY record that would abort the migration
// so the operator sees the whole problem in one run.
func TestRunCollectionPrecheckDirtyReportsEveryRecord(t *testing.T) {
	ctx, key, cdc := precheckCmdStore(t)
	store := ctx.KVStore(key)

	// Dirty denom: invalid bech32 creator.
	denomBz, err := cdc.Marshal(&collectiontypes.Denom{Id: "denom-bad", Creator: "not-bech32"})
	require.NoError(t, err)
	store.Set(v2.KeyDenom("denom-bad"), denomBz)

	// Dirty NFT: invalid bech32 owner.
	nftBz, err := cdc.Marshal(&collectiontypes.BaseNFT{Id: "token-bad", Owner: "also-not-bech32"})
	require.NoError(t, err)
	store.Set(v2.KeyNFT("denom-bad", "token-bad"), nftBz)

	// Corrupt value under the NFT prefix (unmarshal failure).
	store.Set(v2.KeyNFT("denom-bad", "token-corrupt"), []byte{0xde, 0xad})

	var out bytes.Buffer
	err = runCollectionPrecheck(ctx, key, cdc, &out)

	require.Error(t, err, "a dirty store must fail the command")
	require.Contains(t, err.Error(), "3 record(s)",
		"the report must count every offending record, not just the first")
	require.Contains(t, err.Error(), "not-bech32")
	require.Contains(t, err.Error(), "also-not-bech32")
	require.NotContains(t, out.String(), "clean",
		"a dirty store must not print the clean confirmation")
}
