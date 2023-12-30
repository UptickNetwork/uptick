package types

import (
	"github.com/cometbft/cometbft/crypto/tmhash"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common"
	ethermint "github.com/evmos/ethermint/types"
)

// NewTokenPair returns an instance of TokenPair
func NewTokenPair(erc721Address common.Address, classID string) TokenPair {
	return TokenPair{
		Erc721Address: erc721Address.String(),
		ClassId:       classID,
	}
}

// GetID returns the SHA256 hash of the ERC721 address and denomination
func (tp TokenPair) GetID() []byte {
	id := tp.Erc721Address + "|" + tp.ClassId
	return tmhash.Sum([]byte(id))
}

// GetERC721Contract casts the hex string address of the ERC21 to common.Address
func (tp TokenPair) GetERC721Contract() common.Address {
	return common.HexToAddress(tp.Erc721Address)
}

// Validate performs a stateless validation of a TokenPair
func (tp TokenPair) Validate() error {
	if err := sdk.ValidateDenom(tp.ClassId); err != nil {
		return err
	}
	return ethermint.ValidateAddress(tp.Erc721Address)
}
