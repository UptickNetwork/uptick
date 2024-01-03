package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// CW721Data represents the CW721 token details used to map
// the token to a Cosmos NFT
type CW721Data struct {
	Name   string
	Symbol string
}

// NFTEnhance represents the CW721 token details used to map
// the token to a Cosmos NFT
type NFTEnhance struct {
	Name    string
	Uri     string
	Data    string
	UriHash string
}

// NewNFTEnhance creates a new CW721Data instance
func NewNFTEnhance(name string, uri string, data string, uriHash string) NFTEnhance {
	return NFTEnhance{
		Name:    name,
		Uri:     uri,
		Data:    data,
		UriHash: uriHash,
	}
}

// ClassEnhance represents the CW721 token details used to map
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

// NewClassEnhance creates a new CW721Data instance
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

// CW721StringResponse defines the string value from the call response
type CW721StringResponse struct {
	Value string
}

// NewCW721Data creates a new CW721Data instance
func NewCW721Data(name string, symbol string) CW721Data {
	return CW721Data{
		Name:   name,
		Symbol: symbol,
	}
}

type CW721TokenData struct {
	Name   string
	Symbol string
	URI    string
}

type CW721TokenStringResponse struct {
	Value string
}

func NewCW721TokenData(name string, symbol string, uri string) CW721TokenData {
	return CW721TokenData{
		Name:   name,
		Symbol: symbol,
		URI:    uri,
	}
}

type CW721TokenIDResponse struct {
	Value *big.Int
}

type CW721TokenOwnerResponse struct {
	Value common.Address
}
