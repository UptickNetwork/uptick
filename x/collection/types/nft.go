package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/exported"
)

var _ exported.NFT = BaseNFT{}

// NewBaseNFT creates a new NFT instance
func NewBaseNFT(id, name string, owner sdk.AccAddress, uri, data string) BaseNFT {
	return BaseNFT{
		ID:    id,
		Name:  name,
		Owner: owner.String(),
		URI:   uri,
		Data:  data,
	}
}

// GetID return the id of BaseNFT
func (bNFT BaseNFT) GetID() string {
	return bNFT.ID
}

// GetName return the name of BaseNFT
func (bNFT BaseNFT) GetName() string {
	return bNFT.Name
}

// GetOwner return the owner of BaseNFT
func (bNFT BaseNFT) GetOwner() sdk.AccAddress {
	owner, _ := sdk.AccAddressFromBech32(bNFT.Owner)
	return owner
}

// GetURI return the URI of BaseNFT
func (bNFT BaseNFT) GetURI() string {
	return bNFT.URI
}

// GetData return the Data of BaseNFT
func (bNFT BaseNFT) GetData() string {
	return bNFT.Data
}

// ----------------------------------------------------------------------------
// NFT

// NFTs define a list of NFT
type NFTs []exported.NFT

// NewNFTs creates a new set of NFTs
func NewNFTs(nfts ...exported.NFT) NFTs {
	if len(nfts) == 0 {
		return NFTs{}
	}
	return NFTs(nfts)
}
