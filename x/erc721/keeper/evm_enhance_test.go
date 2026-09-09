package keeper

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/erc721/contracts"
	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// These tests pin the P0 fix in QueryClassEnhance / QueryNFTEnhance
// (evm.go): an external, user-registered ERC721 contract that returns
// malformed / short / type-mismatched data must NOT panic the node.
// Before the fix, the ABI-unpacked []interface{} was indexed without a
// length guard and asserted with non comma-ok type assertions, so a
// non-conforming contract could take down the validating node.
//
// The fix added: (1) a len guard (len != 7 / len < 4 -> ErrABIUnpack),
// (2) comma-ok type assertions that degrade the offending field to its
// zero value + a Debug log instead of panicking.
//
// The single prerequisite was that fakeEVMKeeper.ApplyMessage could
// return a *controlled* Ret (it previously returned an empty one, which
// made the entire ABI-unpack path untestable). convert_test.go's
// fakeEVMKeeper now carries retBytes / retErr / seq for this purpose.
//
// NOTE on coverage of the len-guard and type-mismatch branches: a real
// go-ethereum abi.Unpack against the canonical method signature always
// returns exactly the declared number of outputs with the declared types
// (or an error). The len != 7 / len < 4 / non-string branches are pure
// defense-in-depth against ABI-library edge cases and non-conforming
// contracts, and are not directly reachable through a single legal ABI
// encoding. The tests below therefore exercise the *reachable* facets of
// the P0 surface: EVM-call failure, Unpack failure (malformed bytes),
// and the happy path (valid ABI-encoded return) — proving none of them
// panics and each returns the correct error / value.

const testEnhanceContract = "0x1111111111111111111111111111111111111111"

// packClassEnhanceOutputs ABI-encodes a getClassEnhanceInfo return (7 values:
// string,string,bool,string,bool,string,string) the way a real ERC721Uptick
// contract would, so fakeEVMKeeper can inject it as a controlled CallEVM Ret.
func packClassEnhanceOutputs(t *testing.T, data, description string, mintRestricted bool, schema string, updateRestricted bool, uri, uriHash string) []byte {
	t.Helper()
	m, ok := contracts.ERC721UpticksContract.ABI.Methods["getClassEnhanceInfo"]
	require.True(t, ok, "getClassEnhanceInfo method missing from ABI")
	bz, err := m.Outputs.Pack(data, description, mintRestricted, schema, updateRestricted, uri, uriHash)
	require.NoError(t, err, "pack getClassEnhanceInfo outputs")
	return bz
}

// packNFTEnhanceOutputs ABI-encodes a getNFTEnhanceInfo return (4 strings:
// name, uri, data, uriHash).
func packNFTEnhanceOutputs(t *testing.T, name, uri, data, uriHash string) []byte {
	t.Helper()
	m, ok := contracts.ERC721UpticksContract.ABI.Methods["getNFTEnhanceInfo"]
	require.True(t, ok, "getNFTEnhanceInfo method missing from ABI")
	bz, err := m.Outputs.Pack(name, uri, data, uriHash)
	require.NoError(t, err, "pack getNFTEnhanceInfo outputs")
	return bz
}

// packTokenURIOutputs ABI-encodes a tokenURI return (single string).
func packTokenURIOutputs(t *testing.T, uri string) []byte {
	t.Helper()
	m, ok := contracts.ERC721UpticksContract.ABI.Methods["tokenURI"]
	require.True(t, ok, "tokenURI method missing from ABI")
	bz, err := m.Outputs.Pack(uri)
	require.NoError(t, err, "pack tokenURI outputs")
	return bz
}

// TestQueryClassEnhance_EVMErrorReturnsErr pins that a failing EVM call
// (ApplyMessage returns an error) is surfaced as an error, not a panic.
func TestQueryClassEnhance_EVMErrorReturnsErr(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)
	evm := k.evmKeeper.(*fakeEVMKeeper)
	evm.retErr = errors.New("evm execution boom")
	contract := common.HexToAddress(testEnhanceContract)

	_, err := k.QueryClassEnhance(ctx, contract)
	require.Error(t, err)
	require.ErrorContains(t, err, "contract call failed")
	require.Equal(t, 1, evm.applyCalls)
}

// TestQueryClassEnhance_MalformedRetReturnsABIErr pins that bytes which are
// not a valid ABI encoding of the 7 declared outputs yield ErrABIUnpack
// (the Unpack failure path), not a panic / out-of-range access.
func TestQueryClassEnhance_MalformedRetReturnsABIErr(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)
	evm := k.evmKeeper.(*fakeEVMKeeper)
	// 4 random bytes cannot satisfy the 7-output (string,string,bool,string,
	// bool,string,string) ABI layout, so Unpack fails.
	evm.retBytes = []byte{0xde, 0xad, 0xbe, 0xef}
	contract := common.HexToAddress(testEnhanceContract)

	_, err := k.QueryClassEnhance(ctx, contract)
	require.ErrorIs(t, err, types.ErrABIUnpack)
}

// TestQueryClassEnhance_ValidRetReturnsEnhance pins the happy path: a
// well-formed ABI-encoded return is unpacked into the correct ClassEnhance
// fields. This guards against a regression that "fixes" the panic by
// always returning zero values.
func TestQueryClassEnhance_ValidRetReturnsEnhance(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)
	evm := k.evmKeeper.(*fakeEVMKeeper)
	evm.retBytes = packClassEnhanceOutputs(t, "data-val", "desc-val", true, "schema-val", false, "ipfs://uri-val", "hash-val")
	contract := common.HexToAddress(testEnhanceContract)

	ce, err := k.QueryClassEnhance(ctx, contract)
	require.NoError(t, err)
	require.Equal(t, "data-val", ce.Data)
	require.Equal(t, "desc-val", ce.Description)
	require.True(t, ce.MintRestricted)
	require.Equal(t, "schema-val", ce.Schema)
	require.False(t, ce.UpdateRestricted)
	require.Equal(t, "ipfs://uri-val", ce.Uri)
	require.Equal(t, "hash-val", ce.UriHash)
}

// TestQueryNFTEnhance_EVMErrorFallsBackToTokenURI pins the fallback contract:
// when getNFTEnhanceInfo fails, QueryNFTEnhance retries tokenURI and, on
// success, returns an NFTEnhance with only the Uri populated. The scripted
// seq drives call 0 (getNFTEnhanceInfo) to error and call 1 (tokenURI) to a
// valid ABI-encoded URI return.
func TestQueryNFTEnhance_EVMErrorFallsBackToTokenURI(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)
	evm := k.evmKeeper.(*fakeEVMKeeper)
	evm.seq = []seqResp{
		{err: errors.New("getNFTEnhanceInfo revert")},
		{ret: packTokenURIOutputs(t, "ipfs://fallback-uri")},
	}
	contract := common.HexToAddress(testEnhanceContract)

	ne, err := k.QueryNFTEnhance(ctx, contract, big.NewInt(1))
	require.NoError(t, err)
	require.Equal(t, "ipfs://fallback-uri", ne.Uri)
	require.Empty(t, ne.Name)
	require.Empty(t, ne.Data)
	require.Empty(t, ne.UriHash)
	require.Equal(t, 2, evm.applyCalls)
}

// TestQueryNFTEnhance_AllCallsFailReturnsErr pins that when both the
// getNFTEnhanceInfo and the tokenURI fallback fail, an error is returned
// rather than a zero-value NFTEnhance (which would silently mask bad data).
func TestQueryNFTEnhance_AllCallsFailReturnsErr(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)
	evm := k.evmKeeper.(*fakeEVMKeeper)
	evm.retErr = errors.New("evm down")
	contract := common.HexToAddress(testEnhanceContract)

	_, err := k.QueryNFTEnhance(ctx, contract, big.NewInt(1))
	require.Error(t, err)
	require.Equal(t, 2, evm.applyCalls)
}

// TestQueryNFTEnhance_MalformedRetFallsBackToTokenURI pins that a malformed
// getNFTEnhanceInfo return (Unpack failure) triggers the tokenURI fallback,
// and if tokenURI is well-formed the call still succeeds with a URI-only
// NFTEnhance — no panic on the malformed intermediate bytes.
func TestQueryNFTEnhance_MalformedRetFallsBackToTokenURI(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)
	evm := k.evmKeeper.(*fakeEVMKeeper)
	// call 0: malformed bytes -> Unpack fails -> fallback;
	// call 1: valid tokenURI return.
	evm.seq = []seqResp{
		{ret: []byte{0x00, 0x01, 0x02, 0x03}},
		{ret: packTokenURIOutputs(t, "ipfs://malformed-fallback")},
	}
	contract := common.HexToAddress(testEnhanceContract)

	ne, err := k.QueryNFTEnhance(ctx, contract, big.NewInt(1))
	require.NoError(t, err)
	require.Equal(t, "ipfs://malformed-fallback", ne.Uri)
}

// TestQueryNFTEnhance_ValidRetReturnsEnhance pins the happy path: a valid
// ABI-encoded getNFTEnhanceInfo return unpacks into the correct NFTEnhance
// fields (name, uri, data, uriHash).
func TestQueryNFTEnhance_ValidRetReturnsEnhance(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)
	evm := k.evmKeeper.(*fakeEVMKeeper)
	evm.retBytes = packNFTEnhanceOutputs(t, "name-val", "uri-val", "data-val", "urihash-val")
	contract := common.HexToAddress(testEnhanceContract)

	ne, err := k.QueryNFTEnhance(ctx, contract, big.NewInt(1))
	require.NoError(t, err)
	require.Equal(t, "name-val", ne.Name)
	require.Equal(t, "uri-val", ne.Uri)
	require.Equal(t, "data-val", ne.Data)
	require.Equal(t, "urihash-val", ne.UriHash)
	require.Equal(t, 1, evm.applyCalls)
}

// The restriction flags are authorization-bearing: their zero value `false`
// means "NOT restricted", i.e. the permissive end. They reach the denom via
// CreateNFTClass and gate x/collection's mint path, so a type mismatch must
// fail CLOSED (reject the conversion) rather than silently degrade.
//
// These cases are exercised against classEnhanceFromReturn directly because
// abi.Unpack against the canonical signature always produces the declared
// types — the mismatch simply cannot be produced through legal ABI encoding.
func TestClassEnhanceFromReturn_RestrictionFlagsFailClosed(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)
	contract := common.HexToAddress(testEnhanceContract)

	cases := []struct {
		name string
		ret  []interface{}
	}{
		{"mint flag not a bool", []interface{}{"d", "desc", int64(1), "schema", true, "uri", "hash"}},
		{"update flag not a bool", []interface{}{"d", "desc", true, "schema", "yes", "uri", "hash"}},
		{"both flags not bools", []interface{}{"d", "desc", nil, "schema", nil, "uri", "hash"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ce, err := k.classEnhanceFromReturn(ctx, contract, tc.ret)
			require.ErrorIs(t, err, types.ErrClassEnhanceRestrictions)
			require.Equal(t, types.ClassEnhance{}, ce,
				"no partial enhance must escape when the restriction flags are ambiguous")
		})
	}
}

// String metadata may be lost — it carries no privilege — but it must degrade
// loudly (Warn) rather than quietly (Debug).
func TestClassEnhanceFromReturn_StringFieldsDegradeButFlagsSurvive(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)
	contract := common.HexToAddress(testEnhanceContract)

	// data and schema are not strings; both bools are valid and must survive.
	ret := []interface{}{int64(42), "desc", true, nil, false, "ipfs://uri", "hash"}

	ce, err := k.classEnhanceFromReturn(ctx, contract, ret)
	require.NoError(t, err)
	require.Empty(t, ce.Data)
	require.Empty(t, ce.Schema)
	require.Equal(t, "desc", ce.Description)
	require.Equal(t, "ipfs://uri", ce.Uri)
	require.Equal(t, "hash", ce.UriHash)
	// The authorization-bearing fields keep their real values.
	require.True(t, ce.MintRestricted)
	require.False(t, ce.UpdateRestricted)
}
