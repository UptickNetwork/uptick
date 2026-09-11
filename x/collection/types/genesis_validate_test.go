package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// ValidateGenesis must accept exactly what InitGenesis can import. These cases
// pin the creator rule: an empty creator is a legitimate on-chain state (an
// ICS-721 voucher class has no collection-level issuer), while a non-empty
// value that does not decode is corruption and must be rejected before a node
// tries to start from it.
func TestValidateGenesisCreator(t *testing.T) {
	creator := sdk.AccAddress([]byte("creator-addr-01"))

	tests := []struct {
		name    string
		creator string
		wantErr bool
	}{
		{name: "empty creator is a voucher class", creator: "", wantErr: false},
		{name: "valid creator", creator: creator.String(), wantErr: false},
		{name: "malformed creator", creator: "not-bech32", wantErr: true},
		{name: "not an address", creator: "plaindenom", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gs := GenesisState{Collections: []Collection{{
				Denom: Denom{Id: "denom1", Name: "D", Creator: tc.creator},
			}}}

			err := ValidateGenesis(gs)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

// A genesis that lists the same class twice would make InitGenesis write it
// twice with the second write silently winning, so validation must reject it --
// the NFT side already rejects duplicate token ids within a class.
func TestValidateGenesisRejectsDuplicateDenomID(t *testing.T) {
	gs := GenesisState{Collections: []Collection{
		{Denom: Denom{Id: "denom1", Name: "One"}},
		{Denom: Denom{Id: "denom1", Name: "One again"}},
	}}

	err := ValidateGenesis(gs)
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate denom id")
}
