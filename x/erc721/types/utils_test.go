package types

import (
	"encoding/hex"
	"strings"
	"testing"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Token id spelling compatibility (A-3)
//
// The fixtures are real mainnet state: the pair indexed by
// "0x6e66746d70313233343536373839,0x087254935ad3d71deacbd2789c3a954b7f9e0822"
// binds cosmos nft "nftmp123456789" of class "mp123456". The token id is the
// legacy spelling — the UTF-8 bytes of the nft id, hex-encoded — while
// v0.4.0+ derives a base-10 sha256 instead, so the two spellings of that NFT
// denote different values and only the key spelling can be unified.
// ---------------------------------------------------------------------------

const (
	sampleNFTID            = "nftmp123456789"
	sampleLegacyTokenID    = "0x6e66746d70313233343536373839"
	sampleCanonicalTokenID = "2239182361542004876924632130074681"
	sampleMaxUint256       = "115792089237316195423570985008687907853269984665640564039457584007913129639935"
	sampleUint256PlusOne   = "115792089237316195423570985008687907853269984665640564039457584007913129639936"
)

// The legacy spelling is exactly the hex of the nft id bytes, which is what
// makes LegacyEVMTokenID able to reconstruct it from the value alone.
func TestLegacyTokenIDFixtureMatchesMainnetData(t *testing.T) {
	t.Parallel()

	require.Equal(t, sampleLegacyTokenID, "0x"+hex.EncodeToString([]byte(sampleNFTID)))

	legacy, ok := LegacyEVMTokenID(sampleCanonicalTokenID)
	require.True(t, ok)
	require.Equal(t, sampleLegacyTokenID, legacy)
}

func TestParseEVMTokenID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		tokenID string
		want    string // expected value in base-10; "" means the input must be rejected
	}{
		{"canonical base-10", "7", "7"},
		{"canonical with leading zeros", "007", "7"},
		{"zero", "0", "0"},
		{"max uint256", sampleMaxUint256, sampleMaxUint256},
		{"legacy mainnet sample", sampleLegacyTokenID, sampleCanonicalTokenID},
		{"legacy short hex", "0x7", "7"},
		{"legacy padded hex", "0x07", "7"},
		{"legacy uppercase prefix", "0X7", "7"},
		{"legacy uppercase digits", "0X6E6674", "7235188"},
		{"empty", "", ""},
		{"bare 0x prefix", "0x", ""},
		{"hex digits without prefix are not decimal", "6e6674", ""},
		{"negative", "-1", ""},
		{"negative hex", "-0x1", ""},
		{"not a number", "kitty", ""},
		{"uint256 overflow", sampleUint256PlusOne, ""},
		{"hex overflow", "0x1" + strings.Repeat("0", 64), ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			n, err := ParseEVMTokenID(tc.tokenID)
			if tc.want == "" {
				require.Error(t, err, "input %q must be rejected", tc.tokenID)
				return
			}
			require.NoError(t, err, "input %q", tc.tokenID)
			require.Equal(t, tc.want, n.String())
		})
	}
}

// ValidateEVMTokenID is the creation-time gate: base-10 only, even though the
// legacy spelling parses fine.
func TestValidateEVMTokenID_RejectsLegacySpelling(t *testing.T) {
	t.Parallel()

	require.NoError(t, ValidateEVMTokenID(sampleCanonicalTokenID))
	require.NoError(t, ValidateEVMTokenID("7"))

	require.Error(t, ValidateEVMTokenID(sampleLegacyTokenID))
	require.Error(t, ValidateEVMTokenID("0x7"))
	require.Error(t, ValidateEVMTokenID(""))
	require.Error(t, ValidateEVMTokenID("kitty"))
}

func TestCanonicalEVMTokenID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in         string
		want       string
		wantLegacy bool
		wantOK     bool
	}{
		{"7", "7", false, true},
		{"007", "7", false, true},
		{sampleCanonicalTokenID, sampleCanonicalTokenID, false, true},
		{sampleLegacyTokenID, sampleCanonicalTokenID, true, true},
		{"0x07", "7", true, true},
		{"", "", false, false},
		{"0x", "", false, false},
		{"kitty", "", false, false},
	}

	for _, tc := range cases {
		canonical, legacy, err := CanonicalEVMTokenID(tc.in)
		if !tc.wantOK {
			require.Error(t, err, "input %q", tc.in)
			require.Empty(t, canonical)
			continue
		}
		require.NoError(t, err, "input %q", tc.in)
		require.Equal(t, tc.want, canonical)
		require.Equal(t, tc.wantLegacy, legacy)
	}
}

func TestLegacyEVMTokenID_RestoresDroppedLeadingNibble(t *testing.T) {
	t.Parallel()

	// hex.EncodeToString always emits an even number of nibbles, so a leading
	// byte of 0x01..0x0f would otherwise lose its high nibble and never match
	// the stored key.
	got, ok := LegacyEVMTokenID("190931555") // 0x0b616263
	require.True(t, ok)
	require.Equal(t, "0x0b616263", got)

	// An even-length hex is not padded.
	got, ok = LegacyEVMTokenID("258") // 0x0102
	require.True(t, ok)
	require.Equal(t, "0x0102", got)

	// Zero has no legacy spelling, and non-numbers are not token ids.
	_, ok = LegacyEVMTokenID("0")
	require.False(t, ok)
	_, ok = LegacyEVMTokenID("kitty")
	require.False(t, ok)
	_, ok = LegacyEVMTokenID("-1")
	require.False(t, ok)
}

func TestEVMTokenIDKeyVariants(t *testing.T) {
	t.Parallel()

	want := []string{sampleCanonicalTokenID, sampleLegacyTokenID}

	// The variant list depends on the value, never on which spelling the
	// caller happened to hold.
	require.Equal(t, want, EVMTokenIDKeyVariants(sampleCanonicalTokenID))
	require.Equal(t, want, EVMTokenIDKeyVariants(sampleLegacyTokenID))

	// One variant when the legacy spelling would be identical or absent.
	require.Equal(t, []string{"0"}, EVMTokenIDKeyVariants("0"))

	// Unparseable input still yields an exact-match lookup instead of
	// skipping the read against already-broken state.
	require.Equal(t, []string{"kitty"}, EVMTokenIDKeyVariants("kitty"))
}

// ---------------------------------------------------------------------------
// Contract address spelling compatibility (A-4)
//
// The second component of the token-UID key has its own two spellings. Both
// strings below are real mainnet state: 0x3bc44CB8… is how the pre-v0.4.0
// module stored that contract in a TokenPair record, and the lowercase form is
// what the same binding is also keyed by. Asserting that the EIP-55 encoding
// reproduces the on-chain string exactly is a cross-check against chain data,
// not a restatement of the helper.
// ---------------------------------------------------------------------------

const (
	sampleChecksummedContract = "0x3bc44CB88233f75B858d0748a45d196bE8375159"
	sampleLowerContract       = "0x3bc44cb88233f75b858d0748a45d196be8375159"
	// No letters, so both spellings are the same string: the one-variant case.
	sampleCaselessContract = "0x1111111111111111111111111111111111111111"
)

func TestContractAddressFixtureIsEIP55(t *testing.T) {
	t.Parallel()

	// Guard the fixture itself, so the tests below cannot pass vacuously.
	require.NotEqual(t, sampleLowerContract, sampleChecksummedContract)
	require.Equal(t, sampleLowerContract, strings.ToLower(sampleChecksummedContract))

	got, ok := ChecksumContractAddress(sampleLowerContract)
	require.True(t, ok)
	require.Equal(t, sampleChecksummedContract, got,
		"EIP-55 must reproduce the spelling the pre-v0.4.0 module wrote on chain")
}

func TestCanonicalContractAddress(t *testing.T) {
	t.Parallel()

	require.Equal(t, sampleLowerContract, CanonicalContractAddress(sampleChecksummedContract))
	require.Equal(t, sampleLowerContract, CanonicalContractAddress(sampleLowerContract))
	require.Equal(t, sampleCaselessContract, CanonicalContractAddress(sampleCaselessContract))
	require.Equal(t, "", CanonicalContractAddress(""))

	// Fail-safe: input that is not a hex address is passed through untouched,
	// so a malformed value still matches itself exactly instead of being
	// rewritten into something that can no longer be found.
	require.Equal(t, "Kitty", CanonicalContractAddress("Kitty"))
}

func TestChecksumContractAddress(t *testing.T) {
	t.Parallel()

	// Idempotent: the checksummed spelling maps to itself.
	got, ok := ChecksumContractAddress(sampleChecksummedContract)
	require.True(t, ok)
	require.Equal(t, sampleChecksummedContract, got)

	// An address with no letters has only one spelling.
	got, ok = ChecksumContractAddress(sampleCaselessContract)
	require.True(t, ok)
	require.Equal(t, sampleCaselessContract, got)

	for _, invalid := range []string{"", "kitty", "0x123", "0x3bc44CB88233f75B858d0748a45d196bE837515"} {
		_, ok := ChecksumContractAddress(invalid)
		require.False(t, ok, "%q is not a hex address", invalid)
	}
}

func TestContractAddressKeyVariants(t *testing.T) {
	t.Parallel()

	want := []string{sampleLowerContract, sampleChecksummedContract}

	// The variant list depends on the value, never on which spelling the
	// caller happened to hold — same shape as EVMTokenIDKeyVariants.
	require.Equal(t, want, ContractAddressKeyVariants(sampleLowerContract))
	require.Equal(t, want, ContractAddressKeyVariants(sampleChecksummedContract))

	// One variant when the two spellings coincide, or when the input is not an
	// address at all (exact-match fallback).
	require.Equal(t, []string{sampleCaselessContract}, ContractAddressKeyVariants(sampleCaselessContract))
	require.Equal(t, []string{"kitty"}, ContractAddressKeyVariants("kitty"))
}

func TestEqualContractAddress(t *testing.T) {
	t.Parallel()

	// Same 20 bytes, spelled two ways.
	require.True(t, EqualContractAddress(sampleLowerContract, sampleChecksummedContract))
	require.True(t, EqualContractAddress(sampleChecksummedContract, sampleLowerContract))
	require.True(t, EqualContractAddress(sampleLowerContract, sampleLowerContract))

	// Different addresses stay different.
	require.False(t, EqualContractAddress(sampleLowerContract, sampleCaselessContract))
	require.False(t, EqualContractAddress(sampleCaselessContract, sampleLowerContract))

	// Malformed input falls back to an exact comparison rather than being
	// widened into a match.
	require.True(t, EqualContractAddress("", ""))
	require.False(t, EqualContractAddress("", sampleLowerContract))
	require.False(t, EqualContractAddress("0xABC", "0xabc"))

	// A short "0x" string is not an address, so it never matches one by value.
	require.False(t, EqualContractAddress("0x3bc44cb8", sampleLowerContract))
}

func TestEqualEVMTokenID(t *testing.T) {
	t.Parallel()

	// Same uint256, spelled two ways.
	require.True(t, EqualEVMTokenID(sampleLegacyTokenID, sampleCanonicalTokenID))
	require.True(t, EqualEVMTokenID(sampleCanonicalTokenID, sampleLegacyTokenID))
	require.True(t, EqualEVMTokenID(sampleCanonicalTokenID, sampleCanonicalTokenID))

	// Different values stay different, and the range check is not bypassed:
	// a 257-bit id is invalid, so it cannot be "equal" to a valid one.
	require.False(t, EqualEVMTokenID("1", "2"))
	require.False(t, EqualEVMTokenID(sampleMaxUint256, sampleUint256PlusOne))

	// Malformed input is matched exactly, never by value.
	require.True(t, EqualEVMTokenID("kitty", "kitty"))
	require.False(t, EqualEVMTokenID("kitty", "kitty2"))
	require.False(t, EqualEVMTokenID("kitty", sampleCanonicalTokenID))
}

// ---------------------------------------------------------------------------
// Whole-key variants (③-a)
//
// A forward key carries both axes at once, so the spellings one binding can be
// stored under are the CROSS PRODUCT of the two variant lists. This is what the
// index prune groups by, and index 0 — canonical on both axes — is the grouping
// key, so every spelling of one binding has to report the same first element.
// ---------------------------------------------------------------------------

func TestTokenUIDKeyVariants_CrossProductOfBothAxes(t *testing.T) {
	t.Parallel()

	want := []string{
		CreateTokenUID(sampleLowerContract, sampleCanonicalTokenID),
		CreateTokenUID(sampleChecksummedContract, sampleCanonicalTokenID),
		CreateTokenUID(sampleLowerContract, sampleLegacyTokenID),
		CreateTokenUID(sampleChecksummedContract, sampleLegacyTokenID),
	}

	// The list depends on the two VALUES, never on which combination of
	// spellings the caller happened to hold.
	for _, key := range want {
		require.Equal(t, want, TokenUIDKeyVariants(key), "input %q", key)
	}

	// And index 0 is the same combination for every one of them, which is what
	// lets a reader group by it.
	for _, key := range want {
		require.Equal(t, want[0], TokenUIDKeyVariants(key)[0], "input %q", key)
	}
}

func TestTokenUIDKeyVariants_ExactMatchFallbacks(t *testing.T) {
	t.Parallel()

	// A non-address second component collapses the address axis to the literal
	// string while the token-id axis still expands.
	require.Equal(t, []string{
		CreateTokenUID("0", sampleCanonicalTokenID),
		CreateTokenUID("0", sampleLegacyTokenID),
	}, TokenUIDKeyVariants(CreateTokenUID("0", sampleCanonicalTokenID)))

	// A key that cannot be split is its own only variant, so already broken
	// state is matched exactly instead of being widened into a match.
	for _, broken := range []string{"", ",", "leading,", ",trailing", "no-comma", "kitty"} {
		require.Equal(t, []string{broken}, TokenUIDKeyVariants(broken), "input %q", broken)
	}
}

func TestSanitizeERC721Name(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"strips leading digits", "123MyToken", "MyToken"},
		{"removes invalid first char", "@abc", "abc"},
		{"keeps slash", "foo/bar", "foo/bar"},
		{"strips ibc prefix recursively", "ibc/ibc/erc721/name", "name"},
		{"strips erc721 prefix", "erc721/foo", "foo"},
		{"truncates to 128", strings.Repeat("a", 150), strings.Repeat("a", 128)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := SanitizeERC721Name(tc.in)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestEqualStringSlice(t *testing.T) {
	t.Parallel()
	require.True(t, EqualStringSlice(nil, nil))
	require.True(t, EqualStringSlice([]string{"a"}, []string{"a"}))
	require.False(t, EqualStringSlice([]string{"a"}, []string{"b"}))
	require.False(t, EqualStringSlice([]string{"a"}, []string{"a", "b"}))
}

func TestEqualMetadata(t *testing.T) {
	t.Parallel()
	base := banktypes.Metadata{
		Description: "d",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: "base", Exponent: 0, Aliases: []string{"x"}},
		},
		Base:    "base",
		Display: "disp",
		Name:    "n",
		Symbol:  "S",
	}
	other := base
	require.NoError(t, EqualMetadata(base, other))

	other.Symbol = "T"
	require.Error(t, EqualMetadata(base, other))

	dupUnits := base
	dupUnits.DenomUnits = append([]*banktypes.DenomUnit{}, base.DenomUnits...)
	dupUnits.DenomUnits = append(dupUnits.DenomUnits, &banktypes.DenomUnit{Denom: "u", Exponent: 6})
	require.Error(t, EqualMetadata(base, dupUnits))
}

func TestRemoveAddress0x(t *testing.T) {
	t.Parallel()
	require.Equal(t, "abcd", removeAddress0x("0xabcd"))
	require.Equal(t, "abcd", removeAddress0x("abcd"))
}

func TestCreateClassIDFromContractAddress(t *testing.T) {
	t.Parallel()
	addr := "0xAbCdEfAbCdEfAbCdEfAbCdEfAbCdEfAbCdEfAb"
	require.Equal(t, "uptick-AbCdEfAbCdEfAbCdEfAbCdEfAbCdEfAbCdEfAb", CreateClassIDFromContractAddress(addr))
}

func TestCreateContractAddressFromClassID(t *testing.T) {
	t.Parallel()
	classID := "uptick-AbCdEfAbCdEfAbCdEfAbCdEfAbCdEfAbCdEfAb"
	require.Equal(t, "AbCdEfAbCdEfAbCdEfAbCdEfAbCdEfAbCdEfAb", CreateContractAddressFromClassID(classID))
}

func TestCreateNFTIDFromTokenID(t *testing.T) {
	t.Parallel()
	id := CreateNFTIDFromTokenID("0x00aa")
	require.Equal(t, "uptick00aa", id)
}

func TestCreateTokenIDFromNFTID_Base10Uint256(t *testing.T) {
	t.Parallel()

	for _, nftID := range []string{
		"nft1",
		"uptick-nft1",
		strings.Repeat("a", 128),
	} {
		tokenID := CreateTokenIDFromNFTID(nftID)
		require.NotEmpty(t, tokenID)
		require.NoError(t, ValidateEVMTokenID(tokenID))
		require.NotContains(t, tokenID, "0x")
	}
}

func TestCreateTokenIDFromNFTID_Deterministic(t *testing.T) {
	t.Parallel()

	require.Equal(t, CreateTokenIDFromNFTID("nft1"), CreateTokenIDFromNFTID("nft1"))
}

func TestCreateTokenIDFromNFTID_DifferentInputs(t *testing.T) {
	t.Parallel()

	require.NotEqual(t, CreateTokenIDFromNFTID("nft1"), CreateTokenIDFromNFTID("nft2"))
}

func TestCreateTokenUIDAndNFTUID(t *testing.T) {
	t.Parallel()
	require.Equal(t, "tid,caddr", CreateTokenUID("caddr", "tid"))
	require.Equal(t, "nid,cid", CreateNFTUID("cid", "nid"))
}

func TestGetNFTFromUID(t *testing.T) {
	t.Parallel()
	a, b := GetNFTFromUID("x,y")
	require.Equal(t, "x", a)
	require.Equal(t, "y", b)
	a, b = GetNFTFromUID("bad")
	require.Equal(t, "", a)
	require.Equal(t, "", b)

	// A comma in the first field round-trips by splitting on the last comma.
	a, b = GetNFTFromUID("tok,en,0x1234")
	require.Equal(t, "tok,en", a)
	require.Equal(t, "0x1234", b)
}
