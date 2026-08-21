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
