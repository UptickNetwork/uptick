package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

func (s *KeeperTestSuite) TestMsgServerNFTLifecycleAuthorization() {
	creator := sdk.AccAddress([]byte("creator"))
	other := sdk.AccAddress([]byte("other"))
	recipient := sdk.AccAddress([]byte("recipient"))
	goCtx := sdk.WrapSDKContext(s.ctx)

	_, err := s.keeper.IssueDenom(goCtx, &types.MsgIssueDenom{
		Id:               "denom1",
		Name:             "Denom One",
		Symbol:           "ONE",
		Schema:           "",
		Sender:           creator.String(),
		MintRestricted:   true,
		UpdateRestricted: false,
	})
	s.Require().NoError(err)

	mintMsg := &types.MsgMintNFT{
		DenomId:   "denom1",
		Id:        "nft1",
		Name:      "NFT One",
		URI:       "ipfs://nft1",
		Data:      "",
		Sender:    creator.String(),
		Recipient: recipient.String(),
	}
	_, err = s.keeper.MintNFT(goCtx, mintMsg)
	s.Require().NoError(err)

	mintMsg.Sender = other.String()
	_, err = s.keeper.MintNFT(goCtx, mintMsg)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "not allowed to mint NFT")

	_, err = s.keeper.TransferNFT(goCtx, &types.MsgTransferNFT{
		DenomId:   "denom1",
		Id:        "nft1",
		Sender:    other.String(),
		Recipient: creator.String(),
	})
	s.Require().Error(err)

	_, err = s.keeper.TransferNFT(goCtx, &types.MsgTransferNFT{
		DenomId:   "denom1",
		Id:        "nft1",
		Sender:    recipient.String(),
		Recipient: creator.String(),
	})
	s.Require().NoError(err)

	// M-1: the transfer above left every optional field empty (the REST/gRPC
	// shape). The metadata must survive untouched -- empty string means "do
	// not modify", never "clear".
	nft, err := s.keeper.GetNFT(s.ctx, "denom1", "nft1")
	s.Require().NoError(err)
	s.Require().Equal("NFT One", nft.GetName())
	s.Require().Equal("ipfs://nft1", nft.GetURI())

	_, err = s.keeper.BurnNFT(goCtx, &types.MsgBurnNFT{
		DenomId: "denom1",
		Id:      "nft1",
		Sender:  other.String(),
	})
	s.Require().Error(err)

	_, err = s.keeper.BurnNFT(goCtx, &types.MsgBurnNFT{
		DenomId: "denom1",
		Id:      "nft1",
		Sender:  creator.String(),
	})
	s.Require().NoError(err)

	_, err = s.keeper.GetNFT(s.ctx, "denom1", "nft1")
	s.Require().Error(err)
}

// TestUpdateRestrictedAllowsPureTransfer verifies that for an UpdateRestricted
// denom a pure ownership transfer MUST succeed even when the caller echoes the
// existing metadata, while a real metadata change must remain blocked.
func (s *KeeperTestSuite) TestUpdateRestrictedAllowsPureTransfer() {
	creator := sdk.AccAddress([]byte("creator2"))
	recipient := sdk.AccAddress([]byte("recipient2"))
	goCtx := sdk.WrapSDKContext(s.ctx)

	_, err := s.keeper.IssueDenom(goCtx, &types.MsgIssueDenom{
		Id:               "denomlocked",
		Name:             "Locked",
		Symbol:           "LCK",
		Schema:           "",
		Sender:           creator.String(),
		MintRestricted:   true,
		UpdateRestricted: true,
	})
	s.Require().NoError(err)

	_, err = s.keeper.MintNFT(goCtx, &types.MsgMintNFT{
		DenomId:   "denomlocked",
		Id:        "nft1",
		Name:      "NFT One",
		URI:       "ipfs://nft1",
		Data:      "",
		Sender:    creator.String(),
		Recipient: recipient.String(),
	})
	s.Require().NoError(err)

	// Pure transfer: echo the SAME metadata. This must not be treated as an
	// update, so UpdateRestricted must not block it.
	_, err = s.keeper.TransferNFT(goCtx, &types.MsgTransferNFT{
		DenomId:   "denomlocked",
		Id:        "nft1",
		Name:      "NFT One",
		URI:       "ipfs://nft1",
		Data:      "",
		UriHash:   "",
		Sender:    recipient.String(),
		Recipient: creator.String(),
	})
	s.Require().NoError(err)

	// Setting a genuinely different URI on an UpdateRestricted denom is blocked.
	_, err = s.keeper.TransferNFT(goCtx, &types.MsgTransferNFT{
		DenomId:   "denomlocked",
		Id:        "nft1",
		Name:      "NFT One",
		URI:       "ipfs://changed",
		Data:      "",
		Sender:    creator.String(),
		Recipient: recipient.String(),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "restricted to update NFT")
}

// TestMsgServerEditNFT covers the EditNFT branch: on a non-restricted denom the
// owner can update metadata, while an UpdateRestricted denom reject edits.
func (s *KeeperTestSuite) TestMsgServerEditNFT() {
	creator := sdk.AccAddress([]byte("editor-creator"))
	goCtx := sdk.WrapSDKContext(s.ctx)

	_, err := s.keeper.IssueDenom(goCtx, &types.MsgIssueDenom{
		Id:               "denomedit",
		Name:             "Editable",
		Symbol:           "EDT",
		Schema:           "",
		Sender:           creator.String(),
		MintRestricted:   true,
		UpdateRestricted: false,
	})
	s.Require().NoError(err)
	_, err = s.keeper.MintNFT(goCtx, &types.MsgMintNFT{
		DenomId: "denomedit", Id: "nft1", Name: "Old", URI: "ipfs://old", Data: "",
		Sender: creator.String(), Recipient: creator.String(),
	})
	s.Require().NoError(err)

	_, err = s.keeper.EditNFT(goCtx, &types.MsgEditNFT{
		DenomId: "denomedit", Id: "nft1", Name: "New", URI: "ipfs://new", Sender: creator.String(),
	})
	s.Require().NoError(err)
	nft, err := s.keeper.GetNFT(s.ctx, "denomedit", "nft1")
	s.Require().NoError(err)
	s.Require().Equal("New", nft.GetName())
	s.Require().Equal("ipfs://new", nft.GetURI())

	// A non-owner cannot edit.
	_, err = s.keeper.EditNFT(goCtx, &types.MsgEditNFT{
		DenomId: "denomedit", Id: "nft1", Name: "Hacked", Sender: sdk.AccAddress([]byte("evil")).String(),
	})
	s.Require().Error(err)
}

// TestMsgServerTransferDenom covers the TransferDenom branch: only the current
// denom creator may transfer ownership.
func (s *KeeperTestSuite) TestMsgServerTransferDenom() {
	creator := sdk.AccAddress([]byte("creator-denom"))
	newOwner := sdk.AccAddress([]byte("new-denom-owner"))
	goCtx := sdk.WrapSDKContext(s.ctx)

	_, err := s.keeper.IssueDenom(goCtx, &types.MsgIssueDenom{
		Id: "denomtransfer", Name: "Transfer", Symbol: "TRF", Schema: "",
		Sender: creator.String(), MintRestricted: true, UpdateRestricted: false,
	})
	s.Require().NoError(err)

	// A non-creator cannot transfer the denom.
	_, err = s.keeper.TransferDenom(goCtx, &types.MsgTransferDenom{
		Id: "denomtransfer", Sender: sdk.AccAddress([]byte("evil")).String(), Recipient: newOwner.String(),
	})
	s.Require().Error(err)

	// The creator transfers ownership.
	_, err = s.keeper.TransferDenom(goCtx, &types.MsgTransferDenom{
		Id: "denomtransfer", Sender: creator.String(), Recipient: newOwner.String(),
	})
	s.Require().NoError(err)
	denom, err := s.keeper.GetDenomInfo(s.ctx, "denomtransfer")
	s.Require().NoError(err)
	s.Require().Equal(newOwner.String(), denom.Creator)
}

// TestMsgServerIssueDenomReservedPrefix covers M-8 (2026-09-04 decision A):
// the "uptick-" prefix is reserved for module-derived class IDs (erc721/cw721
// bridging derives "uptick-<contract>"). User-facing issuance must reject it
// so a pre-minted denom can never permanently block contract registration.
func (s *KeeperTestSuite) TestMsgServerIssueDenomReservedPrefix() {
	creator := sdk.AccAddress([]byte("prefix-creator"))
	goCtx := sdk.WrapSDKContext(s.ctx)

	for _, id := range []string{
		"uptick-b37eb5464b45a8097cbbb7c22727a6b259a3d85e", // erc721-derived shape
		"uptick-wasm-contract",                            // cw721-derived shape
		"uptick-anything",
	} {
		_, err := s.keeper.IssueDenom(goCtx, &types.MsgIssueDenom{
			Id: id, Name: "Squat", Symbol: "SQT", Sender: creator.String(),
		})
		s.Require().Error(err, id)
		s.Require().ErrorContains(err, "reserved", id)
	}

	// Non-reserved IDs keep working.
	_, err := s.keeper.IssueDenom(goCtx, &types.MsgIssueDenom{
		Id: "plaindenom", Name: "Plain", Symbol: "PLN", Sender: creator.String(),
	})
	s.Require().NoError(err)
}

// TestMsgServerEditNFTEmptyFieldsKeepMetadata covers the M-1 decision (A):
// empty optional fields mean "do not modify"; clearing requires the explicit
// [remove] sentinel.
func (s *KeeperTestSuite) TestMsgServerEditNFTEmptyFieldsKeepMetadata() {
	creator := sdk.AccAddress([]byte("m1-creator"))
	goCtx := sdk.WrapSDKContext(s.ctx)

	_, err := s.keeper.IssueDenom(goCtx, &types.MsgIssueDenom{
		Id: "denomm1", Name: "M1", Symbol: "M1", Sender: creator.String(),
	})
	s.Require().NoError(err)
	_, err = s.keeper.MintNFT(goCtx, &types.MsgMintNFT{
		DenomId: "denomm1", Id: "nft1", Name: "Keep", URI: "ipfs://keep",
		UriHash: "hash1", Data: `{"k":"v"}`,
		Sender: creator.String(), Recipient: creator.String(),
	})
	s.Require().NoError(err)

	// EditNFT with every optional field empty: metadata must survive.
	_, err = s.keeper.EditNFT(goCtx, &types.MsgEditNFT{
		DenomId: "denomm1", Id: "nft1", Sender: creator.String(),
	})
	s.Require().NoError(err)
	got, err := s.keeper.GetNFT(s.ctx, "denomm1", "nft1")
	s.Require().NoError(err)
	s.Require().Equal("Keep", got.GetName())
	s.Require().Equal("ipfs://keep", got.GetURI())
	s.Require().Equal("hash1", got.GetURIHash())
	s.Require().Equal(`{"k":"v"}`, got.GetData())

	// Explicit [remove] sentinel clears a field.
	_, err = s.keeper.EditNFT(goCtx, &types.MsgEditNFT{
		DenomId: "denomm1", Id: "nft1", URI: types.RemoveField, Sender: creator.String(),
	})
	s.Require().NoError(err)
	got, err = s.keeper.GetNFT(s.ctx, "denomm1", "nft1")
	s.Require().NoError(err)
	s.Require().Equal("", got.GetURI())
	// ...while the other fields are still intact.
	s.Require().Equal("Keep", got.GetName())
	s.Require().Equal("hash1", got.GetURIHash())
}
