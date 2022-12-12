package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// ERC721Data represents the ERC721 token details used to map
// the token to a Cosmos NFT
type ERC721Data struct {
	Name   string
	Symbol string
}

// NFTEnhance represents the ERC721 token details used to map
// the token to a Cosmos NFT
type NFTEnhance struct {
	Name    string
	Uri     string
	Data    string
	UriHash string
}

// NewNFTEnhance creates a new ERC721Data instance
func NewNFTEnhance(name string, uri string, data string, uriHash string) NFTEnhance {
	return NFTEnhance{
		Name:    name,
		Uri:     uri,
		Data:    data,
		UriHash: uriHash,
	}
}

// ClassEnhance represents the ERC721 token details used to map
// the token to a Cosmos NFT
type ClassEnhance struct {
	Data             string
	Description      string
	MintRestricted   bool
	Schema           string
	UpdateRestricted bool
	Uri              string
	UriHash          string
}

// NewClassEnhance creates a new ERC721Data instance
func NewClassEnhance(
	data string, description string, mintRestricted bool, schema string,
	updateRestricted bool, uri string, uriHash string,
) ClassEnhance {
	return ClassEnhance{
		Data:             data,
		Description:      description,
		MintRestricted:   mintRestricted,
		Schema:           schema,
		UpdateRestricted: updateRestricted,
		Uri:              uri,
		UriHash:          uriHash,
	}
}

// ERC721StringResponse defines the string value from the call response
type ERC721StringResponse struct {
	Value string
}

// NewERC721Data creates a new ERC721Data instance
func NewERC721Data(name string, symbol string) ERC721Data {
	return ERC721Data{
		Name:   name,
		Symbol: symbol,
	}
}

type ERC721TokenData struct {
	Name   string
	Symbol string
	URI    string
}

type ERC721TokenStringResponse struct {
	Value string
}

func NewERC721TokenData(name string, symbol string, uri string) ERC721TokenData {
	return ERC721TokenData{
		Name:   name,
		Symbol: symbol,
		URI:    uri,
	}
}

type ERC721TokenIDResponse struct {
	Value *big.Int
}

type ERC721TokenOwnerResponse struct {
	Value common.Address
}
