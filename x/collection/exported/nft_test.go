package exported

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type mockNFT struct{}

func (mockNFT) GetID() string            { return "id" }
func (mockNFT) GetName() string          { return "name" }
func (mockNFT) GetOwner() sdk.AccAddress { return sdk.AccAddress("owner") }
func (mockNFT) GetURI() string           { return "uri" }
func (mockNFT) GetURIHash() string       { return "hash" }
func (mockNFT) GetData() string          { return "data" }

func TestNFTInterfaceCompileCheck(t *testing.T) {
	var _ NFT = mockNFT{}
}
