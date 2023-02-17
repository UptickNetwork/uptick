package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/exported"
)

var _ exported.NFT = BaseNFT{}

// NewBaseNFT creates a new NFT instance
func NewBaseNFT(id, name string, owner sdk.AccAddress, uri, uriHash, data string) BaseNFT {
	return BaseNFT{
		Id:      id,
		Name:    name,
		Owner:   owner.String(),
		URI:     uri,
		UriHash: uriHash,
		Data:    data,
	}
}

// GetID return the id of BaseNFT
func (bNFT BaseNFT) GetID() string {
	return bNFT.Id
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

// GetURIHash return the UriHash of BaseNFT
func (bnft BaseNFT) GetURIHash() string {
	return bnft.UriHash
}

// GetData return the Data of BaseNFT
func (bNFT BaseNFT) GetData() string {
	return bNFT.Data
}

// ----------------------------------------------------------------------------
// NFT

// NFTs define a list of NFT
type NFTs []exported.NFT

func UnmarshalNFTMetadata(cdc codec.Codec, bz []byte) (NFTMetadata, error) {
	var nftMetadata NFTMetadata
	if len(bz) == 0 {
		return nftMetadata, nil
	}

	if err := cdc.Unmarshal(bz, &nftMetadata); err != nil {
		return nftMetadata, err
	}
	return nftMetadata, nil
}
