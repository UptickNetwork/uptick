package types

import (
	"github.com/UptickNetwork/uptick/x/collection/exported"
)

// NewCollection creates a new NFT Collection
//
// Callers normally pass BaseNFT values, but the parameter type is the
// exported.NFT interface, so any implementation may arrive here. A bare type
// assertion would panic on a foreign (yet perfectly valid) implementation;
// rebuild the BaseNFT from the interface getters instead.
func NewCollection(denom Denom, nfts []exported.NFT) (c Collection) {
	c.Denom = denom
	for _, n := range nfts {
		base, ok := n.(BaseNFT)
		if !ok {
			base = BaseNFT{
				Id:      n.GetID(),
				Name:    n.GetName(),
				Owner:   n.GetOwner().String(),
				URI:     n.GetURI(),
				UriHash: n.GetURIHash(),
				Data:    n.GetData(),
			}
		}
		c = c.AddNFT(base)
	}
	return c
}

// AddNFT adds an NFT to the collection
func (c Collection) AddNFT(nft BaseNFT) Collection {
	c.NFTs = append(c.NFTs, nft)
	return c
}

func (c Collection) Supply() int {
	return len(c.NFTs)
}

// NewCollection creates a new NFT Collection
func NewCollections(c ...Collection) []Collection {
	return append([]Collection{}, c...)
}
