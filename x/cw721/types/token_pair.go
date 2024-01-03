package types

import (
	"github.com/cometbft/cometbft/crypto/tmhash"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common"
	ethermint "github.com/evmos/ethermint/types"
)

// NewTokenPair returns an instance of TokenPair
func NewTokenPair(cw721Address string, classID string) TokenPair {
	return TokenPair{
		Cw721Address: cw721Address,
		ClassId:      classID,
	}
}

// GetID returns the SHA256 hash of the ERC721 address and denomination
func (tp TokenPair) GetID() []byte {
	id := tp.Cw721Address + "|" + tp.ClassId
	return tmhash.Sum([]byte(id))
}

// GetCW721Contract casts the hex string address of the ERC21 to common.Address
func (tp TokenPair) GetCW721Contract() common.Address {
	return common.HexToAddress(tp.Cw721Address)
}

// Validate performs a stateless validation of a TokenPair
func (tp TokenPair) Validate() error {

	if err := sdk.ValidateDenom(tp.ClassId); err != nil {
		return err
	}
	return ethermint.ValidateAddress(tp.Cw721Address)
}
