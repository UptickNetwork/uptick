package types

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	// v1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types"
	govcdc "github.com/cosmos/cosmos-sdk/x/gov/codec"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"

	"github.com/ethereum/go-ethereum/common"
	ethermint "github.com/evmos/ethermint/types"
)

// constants
const (
	ProposalTypeRegisterCoin         string = "RegisterCoin"
	ProposalTypeRegisterERC20        string = "RegisterERC20"
	ProposalTypeToggleTokenRelay     string = "ToggleTokenRelay" // #nosec
	ProposalTypeUpdateTokenPairERC20 string = "UpdateTokenPairERC20"
)

// Implements Proposal Interface
var (
	_ v1beta1.Content = &RegisterCoinProposal{}
	_ v1beta1.Content = &RegisterERC20Proposal{}
	_ v1beta1.Content = &ToggleTokenRelayProposal{}
	_ v1beta1.Content = &UpdateTokenPairERC20Proposal{}
)

func init() {
	v1beta1.RegisterProposalType(ProposalTypeRegisterCoin)
	v1beta1.RegisterProposalType(ProposalTypeRegisterERC20)
	v1beta1.RegisterProposalType(ProposalTypeToggleTokenRelay)
	v1beta1.RegisterProposalType(ProposalTypeUpdateTokenPairERC20)

	// v1beta1.
	govcdc.ModuleCdc.Amino.RegisterConcrete(&RegisterCoinProposal{}, "erc20/RegisterCoinProposal", nil)
	govcdc.ModuleCdc.Amino.RegisterConcrete(&RegisterERC20Proposal{}, "erc20/RegisterERC20Proposal", nil)
	govcdc.ModuleCdc.Amino.RegisterConcrete(&ToggleTokenRelayProposal{}, "erc20/ToggleTokenRelayProposal", nil)
	govcdc.ModuleCdc.Amino.RegisterConcrete(&UpdateTokenPairERC20Proposal{}, "erc20/UpdateTokenPairERC20Proposal", nil)
}

// CreateDenomDescription generates a string with the coin description
func CreateDenomDescription(address string) string {
	return fmt.Sprintf("Cosmos coin token representation of %s", address)
}

// CreateDenom generates a string the module name plus the address to avoid conflicts with names staring with a number
func CreateDenom(address string) string {
	return fmt.Sprintf("%s/%s", ModuleName, address)
}

// NewRegisterCoinProposal returns new instance of RegisterCoinProposal
func NewRegisterCoinProposal(title, description string, coinMetadata banktypes.Metadata) v1beta1.Content {
	return &RegisterCoinProposal{
		Title:       title,
		Description: description,
		Metadata:    coinMetadata,
	}
}

// ProposalRoute returns router key for this proposal
func (*RegisterCoinProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
func (*RegisterCoinProposal) ProposalType() string {
	return ProposalTypeRegisterCoin
}

// ValidateBasic performs a stateless check of the proposal fields
func (rtbp *RegisterCoinProposal) ValidateBasic() error {
	if err := rtbp.Metadata.Validate(); err != nil {
		return err
	}

	if err := ibctransfertypes.ValidateIBCDenom(rtbp.Metadata.Base); err != nil {
		return err
	}

	if err := validateIBC(rtbp.Metadata); err != nil {
		return err
	}

	return v1beta1.ValidateAbstract(rtbp)
}

func validateIBC(metadata banktypes.Metadata) error {
	// Check ibc/ denom
	denomSplit := strings.SplitN(metadata.Base, "/", 2)

	if denomSplit[0] == metadata.Base && strings.TrimSpace(metadata.Base) != "" {
		// Not IBC
		return nil
	}

	//modify the naming rules of the ibc proposal
	//if len(denomSplit) != 2 || denomSplit[0] != ibctransfertypes.DenomPrefix {
	//	// NOTE: should be unaccessible (covered on ValidateIBCDenom)
	//	return fmt.Errorf("invalid metadata. %s denomination should be prefixed with the format 'ibc/", metadata.Base)
	//}
	//
	//if !strings.Contains(metadata.Name, "channel-") {
	//	return fmt.Errorf("invalid metadata (Name) for ibc. %s should include channel", metadata.Name)
	//}
	//
	//if !strings.HasPrefix(metadata.Symbol, "ibc") {
	//	return fmt.Errorf("invalid metadata (Symbol) for ibc. %s should include \"ibc\" prefix", metadata.Symbol)
	//}

	return nil
}

// ValidateErc20Denom checks if a denom is a valid erc20/
// denomination
func ValidateErc20Denom(denom string) error {
	denomSplit := strings.SplitN(denom, "/", 2)

	if len(denomSplit) != 2 || denomSplit[0] != ModuleName {
		return fmt.Errorf("invalid denom. %s denomination should be prefixed with the format 'erc20/", denom)
	}

	return ethermint.ValidateAddress(denomSplit[1])
}

// NewRegisterERC20Proposal returns new instance of RegisterERC20Proposal
func NewRegisterERC20Proposal(title, description, erc20Addr string) v1beta1.Content {
	return &RegisterERC20Proposal{
		Title:        title,
		Description:  description,
		Erc20Address: erc20Addr,
	}
}

// ProposalRoute returns router key for this proposal
func (*RegisterERC20Proposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
func (*RegisterERC20Proposal) ProposalType() string {
	return ProposalTypeRegisterERC20
}

// ValidateBasic performs a stateless check of the proposal fields
func (rtbp *RegisterERC20Proposal) ValidateBasic() error {
	if err := ethermint.ValidateAddress(rtbp.Erc20Address); err != nil {
		return sdkerrors.Wrap(err, "ERC20 address")
	}
	return v1beta1.ValidateAbstract(rtbp)
}

// NewToggleTokenRelayProposal returns new instance of ToggleTokenRelayProposal
func NewToggleTokenRelayProposal(title, description string, token string) v1beta1.Content {
	return &ToggleTokenRelayProposal{
		Title:       title,
		Description: description,
		Token:       token,
	}
}

// ProposalRoute returns router key for this proposal
func (*ToggleTokenRelayProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
func (*ToggleTokenRelayProposal) ProposalType() string {
	return ProposalTypeToggleTokenRelay
}

// ValidateBasic performs a stateless check of the proposal fields
func (etrp *ToggleTokenRelayProposal) ValidateBasic() error {
	// check if the token is a hex address, if not, check if it is a valid SDK
	// denom
	if err := ethermint.ValidateAddress(etrp.Token); err != nil {
		if err := sdk.ValidateDenom(etrp.Token); err != nil {
			return err
		}
	}

	return v1beta1.ValidateAbstract(etrp)
}

// NewUpdateTokenPairERC20Proposal returns new instance of UpdateTokenPairERC20Proposal
func NewUpdateTokenPairERC20Proposal(title, description, erc20Addr, newERC20Addr string) v1beta1.Content {
	return &UpdateTokenPairERC20Proposal{
		Title:           title,
		Description:     description,
		Erc20Address:    erc20Addr,
		NewErc20Address: newERC20Addr,
	}
}

// ProposalRoute returns router key for this proposal
func (*UpdateTokenPairERC20Proposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
func (*UpdateTokenPairERC20Proposal) ProposalType() string {
	return ProposalTypeUpdateTokenPairERC20
}

// ValidateBasic performs a stateless check of the proposal fields
func (p *UpdateTokenPairERC20Proposal) ValidateBasic() error {
	if err := ethermint.ValidateAddress(p.Erc20Address); err != nil {
		return sdkerrors.Wrap(err, "ERC20 address")
	}

	if err := ethermint.ValidateAddress(p.NewErc20Address); err != nil {
		return sdkerrors.Wrap(err, "new ERC20 address")
	}

	return v1beta1.ValidateAbstract(p)
}

// GetERC20Address returns the common.Address representation of the ERC20 hex address
func (p UpdateTokenPairERC20Proposal) GetERC20Address() common.Address {
	return common.HexToAddress(p.Erc20Address)
}

// GetNewERC20Address returns the common.Address representation of the new ERC20 hex address
func (p UpdateTokenPairERC20Proposal) GetNewERC20Address() common.Address {
	return common.HexToAddress(p.NewErc20Address)
}
