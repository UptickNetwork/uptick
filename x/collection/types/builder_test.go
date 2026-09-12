package types

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"cosmossdk.io/x/nft"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// newBuilderTestClassBuilder wires the real proto codec + interface registry so
// BuildMetadata exercises the same UnpackAny path the app does (a bare
// InterfaceRegistry would make every Data non-nil case fail for the wrong
// reason).
func newBuilderTestClassBuilder() (ClassBuilder, sdk.AccAddress) {
	registry := codectypes.NewInterfaceRegistry()
	RegisterInterfaces(registry)

	// Exactly 20 bytes: AddressFromBech32 enforces the account address length,
	// so a short fixture would fail the round-trip for an unrelated reason.
	moduleAddr := sdk.AccAddress([]byte("collection-modaddr10"))
	return NewClassBuilder(codec.NewProtoCodec(registry), func(string) sdk.AccAddress {
		return moduleAddr
	}), moduleAddr
}

// decodeClassMetadata reverses BuildMetadata into the key/value bag it encodes.
func decodeClassMetadata(t *testing.T, encoded string) map[string]MediaField {
	t.Helper()
	raw, err := Base64.DecodeString(encoded)
	require.NoError(t, err)

	var decoded map[string]MediaField
	require.NoError(t, json.Unmarshal(raw, &decoded))
	return decoded
}

// TestBuildMetadataToleratesNilClassData pins the P2-8 fix: a class stored
// without the DenomMetadata wrapper — a record that predates it, or one written
// straight through the base nft keeper — must still be encodable for ICS-721.
//
// It used to return "unsupported class metadata", because UnpackAny(nil, ...)
// succeeds and leaves the message nil, so the type assertion rejected the
// class. InterNftKeeper.GetClass then answered not-found and every ICS-721
// transfer of a class that plainly exists aborted.
func TestBuildMetadataToleratesNilClassData(t *testing.T) {
	cb, moduleAddr := newBuilderTestClassBuilder()

	encoded, err := cb.BuildMetadata(nft.Class{
		Id:          "legacy",
		Name:        "Legacy",
		Symbol:      "LEG",
		Description: "issued before the metadata wrapper existed",
		Uri:         "ipfs://legacy",
		UriHash:     "hash-legacy",
		Data:        nil,
	})
	require.NoError(t, err, "a class with nil Data must still be exportable over ICS-721")

	decoded := decodeClassMetadata(t, encoded)

	// The class-level fields survive untouched.
	require.Equal(t, "Legacy", decoded[ClassKeyName].Value)
	require.Equal(t, "LEG", decoded[ClassKeySymbol].Value)
	require.Equal(t, "issued before the metadata wrapper existed", decoded[ClassKeyDescription].Value)
	require.Equal(t, "hash-legacy", decoded[ClassKeyURIhash].Value)

	// The wrapper fields degrade to the same zero values GetDenomInfo reports
	// for the same record, so the query API and the ICS-721 export agree on
	// what an unrecorded class says.
	require.Equal(t, "", decoded[ClassKeySchema].Value)
	require.Equal(t, false, decoded[ClassKeyMintRestricted].Value)
	require.Equal(t, false, decoded[ClassKeyUpdateRestricted].Value)

	// The creator is the one field the encoder cannot leave blank — an empty
	// string fails AddressFromBech32 and would sink the export one line after
	// the degradation. It falls back to the module address, exactly the default
	// Build applies to an inbound packet with no creator.
	require.Equal(t, hex.EncodeToString(moduleAddr), decoded[ClassKeyCreator].Value)
}

// TestBuildMetadataPreservesRecordedCreator is the reverse sentinel for
// TestBuildMetadataToleratesNilClassData: filling an empty creator must not
// start overriding a creator that IS recorded, and the restriction flags must
// still come from the record.
func TestBuildMetadataPreservesRecordedCreator(t *testing.T) {
	cb, _ := newBuilderTestClassBuilder()
	creator := sdk.AccAddress([]byte("abcdefghijabcdefghij"))

	anyVal, err := codectypes.NewAnyWithValue(&DenomMetadata{
		Creator:          creator.String(),
		Schema:           `{"type":"object"}`,
		MintRestricted:   true,
		UpdateRestricted: true,
	})
	require.NoError(t, err)

	encoded, err := cb.BuildMetadata(nft.Class{Id: "issued", Name: "Issued", Data: anyVal})
	require.NoError(t, err)

	decoded := decodeClassMetadata(t, encoded)
	require.Equal(t, hex.EncodeToString(creator), decoded[ClassKeyCreator].Value)
	require.Equal(t, `{"type":"object"}`, decoded[ClassKeySchema].Value)
	require.Equal(t, true, decoded[ClassKeyMintRestricted].Value)
	require.Equal(t, true, decoded[ClassKeyUpdateRestricted].Value)
}

// TestBuildMetadataRejectsForeignData pins that the tolerance above is scoped to
// *missing* metadata only. A class whose Data is present but holds a different
// message type is a corrupt record, not a legacy one, and must still fail —
// otherwise the nil-Data degradation would have quietly become "accept
// anything".
func TestBuildMetadataRejectsForeignData(t *testing.T) {
	cb, _ := newBuilderTestClassBuilder()

	anyVal, err := codectypes.NewAnyWithValue(&NFTMetadata{Name: "not-a-denom"})
	require.NoError(t, err)

	_, err = cb.BuildMetadata(nft.Class{Id: "corrupt", Data: anyVal})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported class metadata")
}

// newBuilderTestTokenBuilder wires the real proto codec + interface registry so
// TokenBuilder.BuildMetadata exercises the same UnpackAny path the app does
// (mirrors newBuilderTestClassBuilder for the class side).
func newBuilderTestTokenBuilder() TokenBuilder {
	registry := codectypes.NewInterfaceRegistry()
	RegisterInterfaces(registry)
	return NewTokenBuilder(codec.NewProtoCodec(registry))
}

// TestTokenBuildMetadataToleratesNilData pins the B-1 fix on the token side: an
// NFT stored without the NFTMetadata wrapper — a record that predates it, or
// one written straight through the base nft keeper — must still be encodable
// for ICS-721, exactly as TestBuildMetadataToleratesNilClassData requires for
// the class side.
//
// Before the fix this returned "unsupported nft metadata", because
// UnpackAny(nil, ...) succeeds and leaves the message nil, so the type
// assertion rejected the token. InterNftKeeper.GetNFT
// (x/internft/keeper.go:152) then answered not-found and nft-transfer aborted
// the recv/ack of a token that plainly exists on chain.
func TestTokenBuildMetadataToleratesNilData(t *testing.T) {
	tb := newBuilderTestTokenBuilder()

	encoded, err := tb.BuildMetadata(nft.NFT{
		ClassId: "legacy",
		Id:      "1",
		Uri:     "ipfs://token-1",
		UriHash: "hash-token-1",
		Data:    nil,
	})
	require.NoError(t, err, "a token with nil Data must still be exportable over ICS-721")

	// The token bag shares the class decoder: TokenKeyName / TokenKeyURIhash
	// carry the same "irismod:" keys the class encoder uses.
	decoded := decodeClassMetadata(t, encoded)

	// The token-level field survives untouched; the metadata name degrades to
	// the zero value, which is what the class side reports for the same input.
	require.Equal(t, "", decoded[TokenKeyName].Value)
	require.Equal(t, "hash-token-1", decoded[TokenKeyURIhash].Value)
}

// TestTokenBuildMetadataPreservesRecordedName is the reverse sentinel for
// TestTokenBuildMetadataToleratesNilData: a token that DOES carry metadata must
// still have its name and uri_hash exported, so the tolerance cannot have
// become "ignore whatever is stored".
func TestTokenBuildMetadataPreservesRecordedName(t *testing.T) {
	tb := newBuilderTestTokenBuilder()

	anyVal, err := codectypes.NewAnyWithValue(&NFTMetadata{Name: "Recorded"})
	require.NoError(t, err)

	encoded, err := tb.BuildMetadata(nft.NFT{
		ClassId: "issued",
		Id:      "1",
		UriHash: "hash-issued",
		Data:    anyVal,
	})
	require.NoError(t, err)

	decoded := decodeClassMetadata(t, encoded)
	require.Equal(t, "Recorded", decoded[TokenKeyName].Value)
	require.Equal(t, "hash-issued", decoded[TokenKeyURIhash].Value)
}

// TestTokenBuildMetadataRejectsForeignData pins that the token-side tolerance
// is scoped to *missing* metadata only. A token whose Data is present but holds
// a different message type is a corrupt record, not a legacy one, and must
// still fail — the same judgement the class side makes.
func TestTokenBuildMetadataRejectsForeignData(t *testing.T) {
	tb := newBuilderTestTokenBuilder()

	anyVal, err := codectypes.NewAnyWithValue(&DenomMetadata{})
	require.NoError(t, err)

	_, err = tb.BuildMetadata(nft.NFT{ClassId: "corrupt", Id: "1", Data: anyVal})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported nft metadata")
}
