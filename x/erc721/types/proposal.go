package types

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	gov "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	ethermint "github.com/evmos/ethermint/types"

	"github.com/cosmos/cosmos-sdk/x/nft"
)

// constants
const (
	ProposalTypeRegisterNFT           string = "RegisterNFT"
	ProposalTypeRegisterERC721        string = "RegisterERC721"
	ProposalTypeToggleTokenConversion string = "ToggleNFTConversion" // #nosec
)

// Implements Proposal Interface
var (
	_ gov.Content = &RegisterNFTProposal{}
	_ gov.Content = &RegisterERC721Proposal{}
	_ gov.Content = &ToggleTokenConversionProposal{}
)

func init() {
	gov.RegisterProposalType(ProposalTypeRegisterNFT)
	gov.RegisterProposalType(ProposalTypeRegisterERC721)
	gov.RegisterProposalType(ProposalTypeToggleTokenConversion)
}

// CreateClass generates a string the module name plus the address to avoid conflicts with names staring with a number
func CreateClassID(address string) string {
	//xxl TODO
	return fmt.Sprintf("%s-%s", ModuleName, address)
}

// NewRegisterNFTProposal returns new instance of RegisterCoinProposal
func NewRegisterNFTProposal(title, description string, class nft.Class) gov.Content {
	return &RegisterNFTProposal{
		Title:       title,
		Description: description,
		Class:       class,
	}
}

// ProposalRoute returns router key for this proposal
func (*RegisterNFTProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
func (*RegisterNFTProposal) ProposalType() string {
	return ProposalTypeRegisterNFT
}

// ValidateBasic performs a stateless check of the proposal fields
func (rtbp *RegisterNFTProposal) ValidateBasic() error {
	return gov.ValidateAbstract(rtbp)
}

// ValidateErc721Class checks if a denom is a valid erc721/class
func ValidateErc721Class(class string) error {
	classSplit := strings.SplitN(class, "/", 2)

	if len(classSplit) != 2 || classSplit[0] != ModuleName {
		return fmt.Errorf("invalid denom. %s denomination should be prefixed with the format 'erc721/", class)
	}

	return ethermint.ValidateAddress(classSplit[1])
}

// NewRegisterERC721Proposal returns new instance of RegisterERC721Proposal
func NewRegisterERC721Proposal(title, description, erc721Addr string) gov.Content {
	return &RegisterERC721Proposal{
		Title:         title,
		Description:   description,
		Erc721Address: erc721Addr,
	}
}

// ProposalRoute returns router key for this proposal
func (*RegisterERC721Proposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
func (*RegisterERC721Proposal) ProposalType() string {
	return ProposalTypeRegisterERC721
}

// ValidateBasic performs a stateless check of the proposal fields
func (rtbp *RegisterERC721Proposal) ValidateBasic() error {
	if err := ethermint.ValidateAddress(rtbp.Erc721Address); err != nil {
		return sdkerrors.Wrap(err, "ERC721 address")
	}
	return gov.ValidateAbstract(rtbp)
}

// NewToggleTokenConversionProposal returns new instance of ToggleTokenConversionProposal
func NewToggleTokenConversionProposal(title, description string, token string) gov.Content {
	return &ToggleTokenConversionProposal{
		Title:       title,
		Description: description,
		Token:       token,
	}
}

// ProposalRoute returns router key for this proposal
func (*ToggleTokenConversionProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
func (*ToggleTokenConversionProposal) ProposalType() string {
	return ProposalTypeToggleTokenConversion
}

// ValidateBasic performs a stateless check of the proposal fields
func (ttcp *ToggleTokenConversionProposal) ValidateBasic() error {
	// check if the token is a hex address, if not, check if it is a valid SDK denom
	if err := ethermint.ValidateAddress(ttcp.Token); err != nil {
		if err := sdk.ValidateDenom(ttcp.Token); err != nil {
			return err
		}
	}

	return gov.ValidateAbstract(ttcp)
}
