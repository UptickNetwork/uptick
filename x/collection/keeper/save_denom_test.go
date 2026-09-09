package keeper

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

// TestSaveDenom_RejectsInvalidIDs pins the ninth-round fix: SaveDenom now
// runs types.ValidateDenomID (the baseline charset/length/NUL/comma/prefix
// policy) as a redundant backstop at the shared entry point used by genesis
// import, user issuance (MsgIssueDenom runs the stricter ValidateIssueDenomID
// upstream) and module-derived classes.
//
// The critical guard is the last group: "uptick-<suffix>" ids must STILL be
// accepted by SaveDenom, because erc721/cw721 Convert derives class ids of
// the form "uptick-<hex>" — exactly the reserved shape that
// ValidateIssueDenomID rejects but ValidateDenomID allows. A regression that
// tightens SaveDenom to ValidateIssueDenomID would break module-derived
// classes at conversion time.
func (s *KeeperTestSuite) TestSaveDenom_RejectsInvalidIDs() {
	creator := sdk.AccAddress([]byte("creator"))

	cases := []struct {
		name    string
		denomID string
	}{
		{"empty", ""},
		{"too short (len<3)", "ab"},
		{"contains comma", "ab,cd"},
		{"contains NUL", "ab\x00cd"},
		{"hyphen outside charset", "ab-cd"},
		{"uppercase start", "Abc"},
		{"too long (len>128)", strings.Repeat("a", 129)},
		{"uptick empty suffix", "uptick-"},
		{"uptick suffix with slash", "uptick-ab/c"},
	}

	for _, tc := range cases {
		s.Run("reject/"+tc.name, func() {
			err := s.keeper.SaveDenom(s.ctx, tc.denomID, "n", "", "S", creator, false, false, "", "", "", "")
			s.Require().Error(err, "denomID %q must be rejected", tc.denomID)
			s.Require().ErrorIs(err, types.ErrInvalidDenom)
		})
	}

	// Legal uptick-<suffix> ids must still pass — this is the ninth-round
	// invariant: SaveDenom uses the baseline ValidateDenomID, not the
	// issuance-only ValidateIssueDenomID.
	for _, id := range []string{"uptick-1a2b3c", "uptick-module"} {
		s.Run("accept/"+id, func() {
			err := s.keeper.SaveDenom(s.ctx, id, "n", "", "S", creator, false, false, "", "", "", "")
			s.Require().NoError(err)
		})
	}
}

// TestSaveDenom_RejectsOversizedSchemaAndData pins the DoS guard added in the
// ninth round: the schema is embedded in the first ERC721 deployment calldata,
// so an unbounded schema can permanently DoS conversion; data is arbitrary
// metadata. Both are bounded at MaxDenomSchemaLen / MaxDenomDataLen.
func (s *KeeperTestSuite) TestSaveDenom_RejectsOversizedSchemaAndData() {
	creator := sdk.AccAddress([]byte("creator"))

	s.Run("schema too long", func() {
		tooLong := strings.Repeat("x", types.MaxDenomSchemaLen+1)
		err := s.keeper.SaveDenom(s.ctx, "overschema", "n", tooLong, "S", creator, false, false, "", "", "", "")
		s.Require().Error(err)
		s.Require().ErrorIs(err, types.ErrInvalidDenom)
	})

	s.Run("data too long", func() {
		tooLong := strings.Repeat("x", types.MaxDenomDataLen+1)
		err := s.keeper.SaveDenom(s.ctx, "overdata", "n", "", "S", creator, false, false, "", "", "", tooLong)
		s.Require().Error(err)
		s.Require().ErrorIs(err, types.ErrInvalidDenom)
	})

	// At-limit values are accepted (boundary check, not off-by-one).
	s.Run("schema at limit accepted", func() {
		atLimit := strings.Repeat("x", types.MaxDenomSchemaLen)
		err := s.keeper.SaveDenom(s.ctx, "atlimitschema", "n", atLimit, "S", creator, false, false, "", "", "", "")
		s.Require().NoError(err)
	})

	s.Run("data at limit accepted", func() {
		atLimit := strings.Repeat("x", types.MaxDenomDataLen)
		err := s.keeper.SaveDenom(s.ctx, "atlimitdata", "n", "", "S", creator, false, false, "", "", "", atLimit)
		s.Require().NoError(err)
	})
}
