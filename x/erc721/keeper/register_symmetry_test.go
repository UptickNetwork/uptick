package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/erc721/contracts"
	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// These tests pin the round-10 symmetry fix (G-1): x/erc721 and x/cw721 expose
// the same two class-registration conditions to operators, so they must surface
// the same error codes. Before the fix x/erc721 had them inverted relative to
// x/cw721 (and inverted relative to this file's own RegisterNFT entry point).

const (
	registerTestContract = "0x1111111111111111111111111111111111111111"
	registerTestClass    = "kitty"
)

// packSingleString encodes a single-string method output (name / symbol), which
// is what QueryERC721 needs before CreateNFTClass reaches the state checks.
func packSingleString(t *testing.T, method, value string) []byte {
	t.Helper()
	m, ok := contracts.ERC721UpticksContract.ABI.Methods[method]
	require.True(t, ok, "%s missing from ABI", method)
	bz, err := m.Outputs.Pack(value)
	require.NoError(t, err, "pack %s outputs", method)
	return bz
}

// Script QueryERC721's name+symbol calls. The third call (getClassEnhanceInfo)
// then falls through to retBytes, whose Unpack failure exercises the
// "contract exposes no enhance metadata" fallback path — which must NOT be
// fatal, or every ERC721 lacking getClassEnhanceInfo would become unregistrable.
func scriptERC721NameSymbol(t *testing.T, k Keeper) {
	t.Helper()
	evm := k.evmKeeper.(*fakeEVMKeeper)
	evm.seq = []seqResp{
		{ret: packSingleString(t, "name", "Kitty")},
		{ret: packSingleString(t, "symbol", "KIT")},
	}
}

func TestCreateNFTClass_AlreadyRegisteredReportsAlreadyExists(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)
	scriptERC721NameSymbol(t, k)

	// setupConvertKeeper registers "kitty" in the class map, so this is the
	// plain user-facing conflict: the caller asked for something that exists.
	err := k.CreateNFTClass(ctx, &types.MsgConvertERC721{
		ClassId:            registerTestClass,
		EvmContractAddress: registerTestContract,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrTokenPairAlreadyExists,
		"an already-registered class is a caller conflict (code 7), not broken internal state (code 5)\n\tname: %s", registerTestClass)
	require.NotErrorIs(t, err, types.ErrInternalTokenPair)
}

func TestCreateNFTClass_NativeDenomWithoutPairReportsInternal(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)
	scriptERC721NameSymbol(t, k)

	// A native denom exists but nothing registered it as an ERC721 pair: the
	// two namespaces have drifted, which is an inconsistent-state condition.
	require.NoError(t, k.nftKeeper.SaveDenom(ctx, "doge", "Doge", "", "DOGE", owner, false, false, "", "", "", ""))
	require.False(t, k.IsClassRegistered(ctx, "doge"))

	err := k.CreateNFTClass(ctx, &types.MsgConvertERC721{
		ClassId:            "doge",
		EvmContractAddress: registerTestContract,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInternalTokenPair,
		"a native denom without a pair is drifted module state (code 5), not an already-exists conflict (code 7)")
	require.NotErrorIs(t, err, types.ErrTokenPairAlreadyExists)
}

// The two modules must agree. This guards the symmetry itself, not just each
// side in isolation: a future change that fixes one and misses the other is
// exactly how the divergence was introduced.
func TestRegisterErrorCodes_AgreeWithCW721(t *testing.T) {
	require.Equal(t, "erc721", types.ModuleName)

	// x/cw721 pairs IsClassRegistered -> AlreadyExists and GetDenomInfo ->
	// InternalTokenPair. x/erc721 now matches (see the two tests above).
	require.NotEqual(t, types.ErrTokenPairAlreadyExists, types.ErrInternalTokenPair)

	// Sanity: the codes the modules rely on are the ones documented in comments.
	require.Contains(t, types.ErrTokenPairAlreadyExists.Error(), "token pair already exists")
	require.Contains(t, types.ErrInternalTokenPair.Error(), "internal nft token mapping error")
}

// Guard against silent reintroduction: a valid, unregistered class must still
// be creatable, i.e. the guards above did not become a blanket rejection.
func TestCreateNFTClass_FreshClassStillSucceeds(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)
	scriptERC721NameSymbol(t, k)

	err := k.CreateNFTClass(ctx, &types.MsgConvertERC721{
		ClassId:            "brandnew",
		EvmContractAddress: registerTestContract,
	})
	require.NoError(t, err, "a fresh class must still register despite the tightened error mapping")

	// CreateNFTClass writes the native denom; the erc721 class map is populated
	// by the caller that wraps it (ConvertERC721), so assert on what this
	// function actually owns.
	denom, err := k.nftKeeper.GetDenomInfo(ctx, "brandnew")
	require.NoError(t, err)
	require.Equal(t, "Kitty", denom.Name, "the denom is populated from the contract's name() return")
}
