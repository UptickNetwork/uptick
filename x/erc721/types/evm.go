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

// ERC721Enhance represents the ERC721 token details used to map
// the token to a Cosmos NFT
type ERC721Enhance struct {
	Name    string
	Uri     string
	Data    string
	UriHash string
}

// NewERC721Enhance creates a new ERC721Data instance
func NewERC721Enhance(name string, uri string, data string, uriHash string) ERC721Enhance {
	return ERC721Enhance{
		Name:    name,
		Uri:     uri,
		Data:    data,
		UriHash: uriHash,
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
