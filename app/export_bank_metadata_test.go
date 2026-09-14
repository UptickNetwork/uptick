package app

import (
	"encoding/json"
	"strings"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
)

// Guards for finding F-2. ibc-go v10.5.0's denom-trace migration
// (setDenomMetadataWithDenomTrace) wrote IBC voucher metadata with
// DenomUnits[0] = {BaseDenom, 0} while Base = IBCDenom(), so every chain
// holding IBC vouchers failed its own validate-genesis on the exported file:
// "metadata's first denomination unit must be the one with base denom ...".
//
// The fixtures below are anchored to the three entries observed on the local
// upgrade chain (base=ibc/4EB1D3A1..., first unit=auoc, display=
// transfer/channel-3/auoc), i.e. the real shape the migration produces.
//
// Every test documents its reverse control: the end-to-end guard runs the real
// export twice on the same planted state (normalisation off -> the bad
// metadata survives; on -> it is repaired), so disabling the production change
// turns it red without any overlay.

const (
	// f2FixtureBase is a valid ibc-style denom (ibc/ + 64 hex chars), unique
	// enough to never collide with real state.
	f2FixtureBase = "ibc/4eb1d3a1cc4cc7a949a6ceb0d1e97fd37c71ee8754fa9ab6fc3e0f0b6da7f2e1"
	f2FixtureSrc  = "auocx"
	f2FixturePath = "transfer/channel-99/auocx"
)

// f2IbcVoucherMetadata is what ibc-go v10.5.0's migration plants: it fails
// bank's own Metadata.Validate().
func f2IbcVoucherMetadata() banktypes.Metadata {
	return banktypes.Metadata{
		Description: "IBC token from " + f2FixturePath,
		DenomUnits:  []*banktypes.DenomUnit{{Denom: f2FixtureSrc, Exponent: 0}},
		Base:        f2FixtureBase,
		Display:     f2FixturePath,
		Name:        f2FixturePath + " IBC token",
		Symbol:      "AUOCX",
	}
}

func f2ValidNativeMetadata() banktypes.Metadata {
	return banktypes.Metadata{
		Description: "native token",
		DenomUnits:  []*banktypes.DenomUnit{{Denom: "auoc", Exponent: 0}},
		Base:        "auoc",
		Display:     "auoc",
		Name:        "Uptick Coin",
		Symbol:      "AUOC",
	}
}

func f2Codec() codec.Codec {
	return codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
}

func f2GenesisWithMetadata(t *testing.T, metas ...banktypes.Metadata) map[string]json.RawMessage {
	t.Helper()
	cdc := f2Codec()
	raw, err := cdc.MarshalJSON(&banktypes.GenesisState{DenomMetadata: metas})
	require.NoError(t, err)
	return map[string]json.RawMessage{banktypes.ModuleName: raw}
}

func f2DenomMetadataFromExport(t *testing.T, appState json.RawMessage) []banktypes.Metadata {
	t.Helper()
	var sections map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(appState, &sections), "exported app state must be a module map")
	bankRaw, ok := sections[banktypes.ModuleName]
	require.True(t, ok, "exported app state must contain the bank section")
	var bankGen banktypes.GenesisState
	require.NoError(t, json.Unmarshal(bankRaw, &bankGen))
	return bankGen.DenomMetadata
}

func TestNormalizeRepairsIbcVoucherMetadata(t *testing.T) {
	cdc := f2Codec()
	bad := f2IbcVoucherMetadata()
	require.Error(t, bad.Validate(),
		"fixture must reproduce the defect: bank itself rejects the migration's metadata")

	genState := f2GenesisWithMetadata(t, bad)
	diags, err := normalizeExportedBankDenomMetadata(cdc, genState)
	require.NoError(t, err)
	require.Len(t, diags, 1)
	require.Equal(t, exportIssueDenomMetadataNormalized, diags[0].Kind)
	require.Equal(t, banktypes.ModuleName, diags[0].Module)
	require.Equal(t, f2FixtureBase, diags[0].Key)
	require.Contains(t, diags[0].Detail, f2FixtureSrc, "the detail must name the original first unit")
	require.Contains(t, diags[0].Detail, f2FixturePath, "the detail must name the original display")

	var bankGen banktypes.GenesisState
	require.NoError(t, cdc.UnmarshalJSON(genState[banktypes.ModuleName], &bankGen))
	require.Len(t, bankGen.DenomMetadata, 1)
	got := bankGen.DenomMetadata[0]
	require.NoError(t, got.Validate(), "the repaired metadata must pass bank's own validation")
	require.Equal(t, f2FixtureBase, got.DenomUnits[0].Denom, "first unit must be the base denom")
	require.Equal(t, uint32(0), got.DenomUnits[0].Exponent)
	require.Equal(t, f2FixtureBase, got.Display,
		"display must fall back to the base: keeping the trace path would require inventing an exponent")
	require.Equal(t, f2IbcVoucherMetadata().Name, got.Name, "name keeps the IBC path")
	require.Equal(t, f2IbcVoucherMetadata().Description, got.Description)
	require.Equal(t, "AUOCX", got.Symbol)
}

func TestNormalizeLeavesValidMetadataUntouched(t *testing.T) {
	cdc := f2Codec()
	valid := f2ValidNativeMetadata()
	require.NoError(t, valid.Validate())

	genState := f2GenesisWithMetadata(t, valid)
	before := string(genState[banktypes.ModuleName])
	diags, err := normalizeExportedBankDenomMetadata(cdc, genState)
	require.NoError(t, err)
	require.Empty(t, diags, "valid metadata must produce no diagnostics")
	require.Equal(t, before, string(genState[banktypes.ModuleName]),
		"a genesis without the defect must be re-encoded byte-identically (i.e. not at all)")
}

func TestNormalizeDropsUnrepairableMetadata(t *testing.T) {
	cdc := f2Codec()
	unfixable := f2IbcVoucherMetadata()
	unfixable.Name = "" // bank rejects empty names and the repair does not invent one
	require.Error(t, unfixable.Validate())

	genState := f2GenesisWithMetadata(t, unfixable)
	diags, err := normalizeExportedBankDenomMetadata(cdc, genState)
	require.NoError(t, err)
	require.Len(t, diags, 1)
	require.Equal(t, exportIssueDenomMetadataDropped, diags[0].Kind)
	require.Equal(t, f2FixtureBase, diags[0].Key)

	var bankGen banktypes.GenesisState
	require.NoError(t, cdc.UnmarshalJSON(genState[banktypes.ModuleName], &bankGen))
	require.Empty(t, bankGen.DenomMetadata,
		"unrepairable metadata must be gone from the exported genesis; balances are untouched by metadata")
}

func TestNormalizeWithoutBankSectionIsANoOp(t *testing.T) {
	cdc := f2Codec()
	diags, err := normalizeExportedBankDenomMetadata(cdc, map[string]json.RawMessage{})
	require.NoError(t, err)
	require.Empty(t, diags)
}

func TestExportNormalizesIbcVoucherMetadataEndToEnd(t *testing.T) {
	appInst, ctx := sharedTestApp(t)

	bad := f2IbcVoucherMetadata()
	require.Error(t, bad.Validate(), "fixture must reproduce the defect")
	appInst.BankKeeper.SetDenomMetaData(ctx, bad)
	t.Cleanup(func() {
		// The bank keeper exposes no delete API; clear the entry through the
		// store so the shared application is left as it was found.
		store := ctx.KVStore(appInst.GetKey(banktypes.StoreKey))
		var doomed [][]byte
		iter := storetypes.KVStorePrefixIterator(store, banktypes.DenomMetadataPrefix)
		for ; iter.Valid(); iter.Next() {
			if strings.Contains(string(iter.Key()), f2FixtureBase) {
				doomed = append(doomed, iter.Key())
			}
		}
		_ = iter.Close()
		for _, key := range doomed {
			store.Delete(key)
		}
	})

	// ---- reverse control: normalisation off -> the defect survives the export
	normalizeExportedBankDenomMetadataEnabled = false
	exportTestHome(t, appInst, t.TempDir())
	exported, err := appInst.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)
	survivor, found := f2FindMetadata(t, f2DenomMetadataFromExport(t, exported.AppState), f2FixtureBase)
	require.True(t, found, "reverse control: with normalisation off the export must still carry the bad metadata")
	require.Error(t, survivor.Validate())
	require.Equal(t, f2FixtureSrc, survivor.DenomUnits[0].Denom)

	// ---- fix: normalisation on -> the exported genesis is valid
	normalizeExportedBankDenomMetadataEnabled = true
	exportTestHome(t, appInst, t.TempDir())
	exported, err = appInst.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)

	repaired, found := f2FindMetadata(t, f2DenomMetadataFromExport(t, exported.AppState), f2FixtureBase)
	require.True(t, found, "the metadata entry must survive as a valid entry, not vanish silently")
	require.NoError(t, repaired.Validate(),
		"the exported genesis must now pass bank's own validation (validate-genesis)")
	require.Equal(t, f2FixtureBase, repaired.DenomUnits[0].Denom)
	require.Equal(t, f2FixtureBase, repaired.Display)
}

func f2FindMetadata(t *testing.T, metas []banktypes.Metadata, base string) (banktypes.Metadata, bool) {
	t.Helper()
	for _, md := range metas {
		if md.Base == base {
			return md, true
		}
	}
	return banktypes.Metadata{}, false
}
