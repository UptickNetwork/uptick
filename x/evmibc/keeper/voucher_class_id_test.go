package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	erc721keeper "github.com/UptickNetwork/uptick/x/erc721/keeper"
)

// TestGetVoucherClassIDImplementationsAgree guards the KEEP-IN-SYNC contract
// between this package's GetVoucherClassID and the duplicated copy in
// x/erc721/keeper/keeper.go.
//
// The two keepers cannot import each other, so nothing at compile time stops
// them from drifting. If they do, the evmibc path and the erc721 path derive
// DIFFERENT voucher class IDs for the same (port, channel, classID) tuple,
// which would silently split one ICS-721 class into two unrelated namespaces —
// NFTs bridged in through one path would be unbridgeable through the other.
//
// Both methods are pure (they read no keeper state), so zero-value keepers are
// sufficient — no app/testenv wiring required.
func TestGetVoucherClassIDImplementationsAgree(t *testing.T) {
	cases := []struct {
		name    string
		port    string
		channel string
		classID string
	}{
		{"simple", "nft-transfer", "channel-0", "kitty"},
		{"empty class id", "nft-transfer", "channel-0", ""},
		{"empty channel", "nft-transfer", "", "kitty"},
		{"multi hop channel", "nft-transfer", "channel-141", "punk"},
		{"class id with slash", "nft-transfer", "channel-0", "sub/collection"},
		{"already prefixed", "nft-transfer", "channel-0", "nft-transfer/channel-5/kitty"},
		{"long class id", "nft-transfer", "channel-0", "averyveryverylongclassidentifier0123456789"},
	}

	var evmibcK Keeper
	var erc721K erc721keeper.Keeper

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := evmibcK.GetVoucherClassID(tc.port, tc.channel, tc.classID)
			want := erc721K.GetVoucherClassID(tc.port, tc.channel, tc.classID)
			require.Equal(t, want, got,
				"x/evmibc and x/erc721 GetVoucherClassID diverged for (port=%q, channel=%q, classID=%q); the KEEP-IN-SYNC duplicates must stay identical",
				tc.port, tc.channel, tc.classID,
			)
		})
	}
}
