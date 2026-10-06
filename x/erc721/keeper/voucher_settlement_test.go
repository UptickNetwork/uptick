package keeper

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/x/vm/statedb"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// settlementEVMKeeper scripts the two EVM calls the voucher settlement makes —
// ownerOf (read) and burn (write) — per contract, so the three fixtures can
// coexist without depending on store iteration order.
type settlementEVMKeeper struct {
	*fakeEVMKeeper
	ownerOfRet map[common.Address][]byte // contract → ABI-packed ownerOf output
	ownerOfErr map[common.Address]error
	burned     []burnRecord
}

type burnRecord struct {
	contract common.Address
	from     common.Address
	tokenID  *big.Int
}

func (s *settlementEVMKeeper) ApplyMessage(
	ctx sdk.Context,
	stateDB *statedb.StateDB,
	msg core.Message,
	hooks *tracing.Hooks,
	commit bool,
	a bool,
	b bool,
) (*evmtypes.MsgEthereumTxResponse, error) {
	if msg.To != nil && len(msg.Data) >= 4 {
		switch {
		case bytes.Equal(msg.Data[:4], methodSelector("ownerOf(uint256)")):
			contract := *msg.To
			if err := s.ownerOfErr[contract]; err != nil {
				return nil, err
			}
			if ret, ok := s.ownerOfRet[contract]; ok {
				return &evmtypes.MsgEthereumTxResponse{Ret: ret}, nil
			}
			return nil, evmtypes.ErrVMExecution
		case bytes.Equal(msg.Data[:4], methodSelector("burn(uint256)")):
			tokenID := new(big.Int).SetBytes(msg.Data[4:36])
			s.burned = append(s.burned, burnRecord{contract: *msg.To, from: msg.From, tokenID: tokenID})
			return &evmtypes.MsgEthereumTxResponse{}, nil
		}
	}
	return s.fakeEVMKeeper.ApplyMessage(ctx, stateDB, msg, hooks, commit, a, b)
}

// registerSettlementPair writes a voucher pair (class↔contract) plus a binding
// and, when nativeOwner is non-nil, a native NFT minted to that address. A nil
// nativeOwner leaves the native side absent (the "stale" fixture).
func registerSettlementPair(t *testing.T, k Keeper, ctx sdk.Context, classID, nftID, contractHex, evmTokenID string, nativeOwner sdk.AccAddress) common.Address {
	t.Helper()
	contract := common.HexToAddress(contractHex)
	pair := types.NewTokenPair(contract, classID)
	require.NoError(t, k.SetTokenPair(ctx, pair))
	k.SetClassMap(ctx, classID, pair.GetID())
	k.SetERC721Map(ctx, contract, pair.GetID())

	if nativeOwner != nil {
		require.NoError(t, k.nftKeeper.SaveDenom(ctx, classID, "Voucher", "", "VCH", nativeOwner, false, false, "", "", "", ""))
		require.NoError(t, k.nftKeeper.SaveNFT(ctx, classID, nftID, "Voucher NFT", "ipfs://v", "", "", nativeOwner))
	}

	require.NoError(t, k.SetNFTPairs(ctx, contractHex, evmTokenID, classID, nftID))
	return contract
}

func settlementLogCounts(t *testing.T, output []byte) map[string]float64 {
	t.Helper()
	counts := map[string]float64{}
	for _, line := range bytes.Split(output, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry["message"] != "ibc voucher pair settlement" {
			continue
		}
		for _, key := range []string{"settled", "stale_cleared", "kept", "pairs_deleted"} {
			if v, ok := entry[key].(float64); ok {
				counts[key] = v
			}
		}
	}
	require.Contains(t, counts, "settled", "expected a settlement summary log line")
	return counts
}

// T4 (R1-C B1): the settlement migration must resolve the three fixture states —
// wrapped (settle), stale (clear binding only), ambiguous (keep) — and report
// the counts in its summary log.
func TestSettleIBCVoucherPairs_Matrix(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)

	const (
		wrappedClass = "ibc/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
		wrappedCtr   = "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		staleClass   = "ibc/BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
		staleCtr     = "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		keptClass    = "ibc/CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC"
		keptCtr      = "0xcccccccccccccccccccccccccccccccccccccccc"
		evmTokenID   = "1"
	)

	// wrapped: native escrowed by module, ERC held by the user (owner).
	wrappedContract := registerSettlementPair(t, k, ctx, wrappedClass, "nftA", wrappedCtr, evmTokenID, owner)
	moveNFTToModuleErc721(t, k, ctx, owner, wrappedClass, "nftA")

	// stale: native side absent (nil owner), ERC held by the user.
	staleContract := registerSettlementPair(t, k, ctx, staleClass, "nftB", staleCtr, evmTokenID, nil)

	// kept: native held by the user (not module), ERC held by the user too —
	// an ambiguous state the migration must not touch.
	keptContract := registerSettlementPair(t, k, ctx, keptClass, "nftC", keptCtr, evmTokenID, owner)

	userHex := common.BytesToAddress(owner.Bytes())
	evm := &settlementEVMKeeper{
		fakeEVMKeeper: &fakeEVMKeeper{accounts: map[common.Address]*statedb.Account{
			wrappedContract: {CodeHash: []byte{1}},
			staleContract:   {CodeHash: []byte{1}},
			keptContract:    {CodeHash: []byte{1}},
		}},
		ownerOfRet: map[common.Address][]byte{
			wrappedContract: packOwnerOf(t, userHex),
			staleContract:   packOwnerOf(t, userHex),
			keptContract:    packOwnerOf(t, userHex),
		},
		ownerOfErr: map[common.Address]error{},
	}
	k.evmKeeper = evm

	var buf bytes.Buffer
	ctx = ctx.WithLogger(log.NewLogger(&buf, log.OutputJSONOption()))

	settled, staleCleared, kept, pairsDeleted := k.SettleIBCVoucherPairs(ctx)

	require.Equal(t, 1, settled, "the wrapped fixture must be settled")
	require.Equal(t, 1, staleCleared, "the stale fixture must be cleared")
	require.Equal(t, 1, kept, "the ambiguous fixture must be kept")
	require.Equal(t, 2, pairsDeleted, "wrapped and stale pairs are purged; kept pair is not")

	counts := settlementLogCounts(t, buf.Bytes())
	require.Equal(t, float64(1), counts["settled"])
	require.Equal(t, float64(1), counts["stale_cleared"])
	require.Equal(t, float64(1), counts["kept"])
	require.Equal(t, float64(2), counts["pairs_deleted"])

	// wrapped: ERC burned from the user, native voucher handed to the user,
	// binding gone.
	require.Len(t, evm.burned, 1, "only the wrapped fixture burns an ERC721")
	require.Equal(t, userHex, evm.burned[0].from, "wrapped burn must come from the ERC holder")
	require.Empty(t, k.GetNFTPairByClassNFTID(ctx, wrappedClass, "nftA"))
	got, err := k.nftKeeper.GetNFT(ctx, wrappedClass, "nftA")
	require.NoError(t, err)
	require.Equal(t, owner.String(), got.GetOwner().String(), "native voucher must be returned to the user")

	// stale: binding gone, no burn, native still absent.
	require.Empty(t, k.GetNFTPairByClassNFTID(ctx, staleClass, "nftB"))

	// kept: binding still present, native still held by the user.
	require.NotEmpty(t, k.GetNFTPairByClassNFTID(ctx, keptClass, "nftC"))
	gotKept, err := k.nftKeeper.GetNFT(ctx, keptClass, "nftC")
	require.NoError(t, err)
	require.Equal(t, owner.String(), gotKept.GetOwner().String())
}

// Regression (M-01): an ownerOf query FAILURE must be kept. The error domain of
// QueryERC721TokenOwner also covers EVM infrastructure failures, a nil response
// and an ABI unpack error, none of which mean "the token does not exist". Before
// the fix the settlement mapped every error to ercIsNone and, whenever the
// native side was gone, took the stale branch — deleting the binding and
// purging the pair of an ERC721 that may still have been live.
func TestSettleIBCVoucherPairs_QueryFailureIsKept(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)

	const (
		errClass = "ibc/DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"
		errCtr   = "0xdddddddddddddddddddddddddddddddddddddddd"
		nftID    = "nftZ"
		tokenID  = "1"
	)

	// Native side absent, so a mis-classified error lands in the stale branch.
	contract := registerSettlementPair(t, k, ctx, errClass, nftID, errCtr, tokenID, nil)

	k.evmKeeper = &settlementEVMKeeper{
		fakeEVMKeeper: &fakeEVMKeeper{accounts: map[common.Address]*statedb.Account{
			contract: {CodeHash: []byte{1}},
		}},
		ownerOfRet: map[common.Address][]byte{},
		ownerOfErr: map[common.Address]error{contract: evmtypes.ErrVMExecution},
	}

	settled, staleCleared, kept, pairsDeleted := k.SettleIBCVoucherPairs(ctx)
	require.Equal(t, 0, settled)
	require.Equal(t, 0, staleCleared, "a query failure must not be read as 'native+erc both gone'")
	require.Equal(t, 1, kept, "a query failure must be kept, not classified as absent")
	require.Equal(t, 0, pairsDeleted, "the pair must survive so A3 / a retry can still find it")
	require.NotEmpty(t, k.GetNFTPairByClassNFTID(ctx, errClass, nftID), "binding must survive")
}

// Regression (M-01): a malformed contract field must not be silently read as the
// zero address (common.HexToAddress maps unparseable strings there) and then
// drive an ownership classification.
func TestSettleIBCVoucherPairs_MalformedContractIsKept(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)

	const (
		badClass = "ibc/EEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE"
		badCtr   = "not-an-address"
		nftID    = "nftY"
		tokenID  = "1"
	)

	registerSettlementPair(t, k, ctx, badClass, nftID, badCtr, tokenID, nil)

	k.evmKeeper = &settlementEVMKeeper{
		fakeEVMKeeper: &fakeEVMKeeper{accounts: map[common.Address]*statedb.Account{}},
		ownerOfRet:    map[common.Address][]byte{},
		ownerOfErr:    map[common.Address]error{},
	}

	_, staleCleared, kept, pairsDeleted := k.SettleIBCVoucherPairs(ctx)
	require.Equal(t, 0, staleCleared)
	require.Equal(t, 1, kept, "a malformed contract address must not be classified")
	require.Equal(t, 0, pairsDeleted)
	require.NotEmpty(t, k.GetNFTPairByClassNFTID(ctx, badClass, nftID))
}

// T4 idempotency: a second run over fully-settled state reports all zeros.
func TestSettleIBCVoucherPairs_Idempotent(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)

	const (
		wrappedClass = "ibc/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
		wrappedCtr   = "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		evmTokenID   = "1"
	)

	wrappedContract := registerSettlementPair(t, k, ctx, wrappedClass, "nftA", wrappedCtr, evmTokenID, owner)
	moveNFTToModuleErc721(t, k, ctx, owner, wrappedClass, "nftA")

	userHex := common.BytesToAddress(owner.Bytes())
	k.evmKeeper = &settlementEVMKeeper{
		fakeEVMKeeper: &fakeEVMKeeper{accounts: map[common.Address]*statedb.Account{
			wrappedContract: {CodeHash: []byte{1}},
		}},
		ownerOfRet: map[common.Address][]byte{
			wrappedContract: packOwnerOf(t, userHex),
		},
		ownerOfErr: map[common.Address]error{},
	}

	settled, _, kept, pairsDeleted := k.SettleIBCVoucherPairs(ctx)
	require.Equal(t, 1, settled)
	require.Equal(t, 0, kept)
	require.Equal(t, 1, pairsDeleted)

	// Second run: nothing left to settle.
	settled, staleCleared, kept, pairsDeleted := k.SettleIBCVoucherPairs(ctx)
	require.Equal(t, 0, settled)
	require.Equal(t, 0, staleCleared)
	require.Equal(t, 0, kept)
	require.Equal(t, 0, pairsDeleted)
}
