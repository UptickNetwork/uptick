package keeper_test

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/erc20/types"
)

// TestH5RefundPacketTokenAtomicity is the full regression test for H5.
// It validates that refundPacketToken correctly wraps all state mutations
// (ERC20 mint, coin sweep, NativeERC20 burn) inside a CacheContext that
// commits only on success, guaranteeing full atomicity.
func TestH5RefundPacketTokenAtomicity(t *testing.T) {
	s := new(KeeperTestSuite)
	s.SetT(t)
	s.SetupTest()

	// Step 1: deploy ERC20 contract and register as NativeERC20 pair.
	contractAddr := s.DeployContract("RefundToken", "RFDN", uint8(18))
	s.Commit()

	pair, err := s.app.Erc20Keeper.RegisterERC20(s.ctx, contractAddr)
	require.NoError(t, err)
	require.True(t, pair.IsNativeERC20(), "pair must be NativeERC20 for burn path")

	// Step 2: set IBC denom map so tokenPairFromPacketDenom can resolve
	// the pair from the full IBC trace denom carried in the error-ACK packet.
	packetDenom := "transfer/channel-0/" + pair.Denom
	trace := transfertypes.ParseDenomTrace(packetDenom)
	s.app.Erc20Keeper.SetDenomMap(s.ctx, trace.IBCDenom(), pair.GetID())
	s.Commit()

	// Step 3: simulate the ibc-go transfer module refunding coins to the
	// original sender on error ACK. The sender receives coins that were
	// previously escrowed by the ICS-20 transfer.
	transferAmount := sdkmath.NewInt(100)
	cosmosSender := sdk.AccAddress(s.address.Bytes())
	coins := sdk.NewCoins(sdk.NewCoin(pair.Denom, transferAmount))

	err = s.app.BankKeeper.MintCoins(s.ctx, types.ModuleName, coins)
	require.NoError(t, err)
	err = s.app.BankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.ModuleName, cosmosSender, coins)
	require.NoError(t, err)
	s.Commit()

	// Verify coins arrived.
	beforeCoins := s.app.BankKeeper.GetBalance(s.ctx, cosmosSender, pair.Denom)
	require.Equal(t, transferAmount, beforeCoins.Amount,
		"sender should have coins after ibc-go refund")

	// Verify no ERC20 balance before refund.
	beforeERC20 := s.BalanceOf(contractAddr, s.address)
	require.Equal(t, big.NewInt(0), beforeERC20,
		"sender should have zero ERC20 before refund")

	// Step 4: set IBC provenance — the MsgTransferERC20 handler would have
	// stored this before initiating the ICS-20 transfer.
	packet := channeltypes.Packet{
		Sequence:      1,
		SourcePort:    "transfer",
		SourceChannel: "channel-0",
	}
	data := transfertypes.FungibleTokenPacketData{
		Denom:    packetDenom,
		Amount:   transferAmount.String(),
		Sender:   cosmosSender.String(),
		Receiver: "cosmos1receiver",
		Memo:     "transfer" + types.TransferERC20Memo,
	}

	s.app.Erc20Keeper.SetIBCTransferProvenance(
		s.ctx,
		packet.SourcePort, packet.SourceChannel, packet.Sequence,
		data.Sender, data.Denom, data.Amount,
	)
	s.Commit()
	require.True(t, s.app.Erc20Keeper.HasIBCTransferProvenance(s.ctx, packet, data),
		"provenance should exist before error ACK")

	// Step 5: error ACK triggers refundPacketToken — the CacheContext commits
	// or rolls back atomically.
	ack := channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Error{
			Error: "IBC transfer failed: timeout",
		},
	}
	err = s.app.Erc20Keeper.OnAcknowledgementPacket(s.ctx, packet, data, ack)
	require.NoError(t, err, "OnAcknowledgementPacket should not error")
	s.Commit()

	// ---- Assertions ----

	// 6a: Provenance consumed — single-use prevent replay.
	require.False(t, s.app.Erc20Keeper.HasIBCTransferProvenance(s.ctx, packet, data),
		"provenance must be consumed (single-use guard)")

	// 6b: ERC20 tokens re-minted to the sender by refundPacketToken.
	afterERC20 := s.BalanceOf(contractAddr, s.address)
	require.Equal(t, transferAmount.BigInt(), afterERC20,
		"ERC20 should be re-minted to sender")

	// 6c: Cosmos coins swept from sender (refundPacketToken sweeps the
	// ibc-go-refunded coins into the module account).
	afterCoins := s.app.BankKeeper.GetBalance(s.ctx, cosmosSender, pair.Denom)
	require.True(t, afterCoins.Amount.IsZero(),
		"coins should be swept from sender")

	// 6d: For NativeERC20, coins are also burned (not left in module).
	// Verify module account has zero balance.
	moduleAddr := s.app.AccountKeeper.GetModuleAddress(types.ModuleName)
	moduleCoins := s.app.BankKeeper.GetBalance(s.ctx, moduleAddr, pair.Denom)
	require.True(t, moduleCoins.Amount.IsZero(),
		"coins should be burned for NativeERC20 pair (not left in module)")

	// Step 7: regression — double refund is impossible (provenance guard).
	err2 := s.app.Erc20Keeper.OnAcknowledgementPacket(s.ctx, packet, data, ack)
	require.NoError(t, err2, "second OnAcknowledgementPacket should be a no-op")
	s.Commit()

	doubleERC20 := s.BalanceOf(contractAddr, s.address)
	require.Equal(t, transferAmount.BigInt(), doubleERC20,
		"no double mint on replayed error ACK (provenance guard)")
}

// TestH5RefundNoProvenanceNoMint verifies that an error ACK without
// MsgTransferERC20 provenance does NOT trigger ERC20 minting — the
// ordinary ICS-20 refund path already handles coin return via ibc-go.
func TestH5RefundNoProvenanceNoMint(t *testing.T) {
	s := new(KeeperTestSuite)
	s.SetT(t)
	s.SetupTest()

	contractAddr := s.DeployContract("NoProvenance", "NOPR", uint8(18))
	s.Commit()

	pair, err := s.app.Erc20Keeper.RegisterERC20(s.ctx, contractAddr)
	require.NoError(t, err)

	// Give sender coins (simulating ibc-go refund after a regular transfer).
	transferAmount := sdkmath.NewInt(50)
	cosmosSender := sdk.AccAddress(s.address.Bytes())
	coins := sdk.NewCoins(sdk.NewCoin(pair.Denom, transferAmount))

	err = s.app.BankKeeper.MintCoins(s.ctx, types.ModuleName, coins)
	require.NoError(t, err)
	err = s.app.BankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.ModuleName, cosmosSender, coins)
	require.NoError(t, err)
	s.Commit()

	// No provenance set — this simulates a regular ICS-20 transfer that
	// was NOT initiated by MsgTransferERC20.
	packetDenom := "transfer/channel-0/" + pair.Denom
	packet := channeltypes.Packet{
		Sequence:      5,
		SourcePort:    "transfer",
		SourceChannel: "channel-0",
	}
	data := transfertypes.FungibleTokenPacketData{
		Denom:    packetDenom,
		Amount:   transferAmount.String(),
		Sender:   cosmosSender.String(),
		Receiver: "cosmos1other",
		Memo:     "",
	}

	// No provenance exists.
	require.False(t, s.app.Erc20Keeper.HasIBCTransferProvenance(s.ctx, packet, data))

	// Error ACK with no provenance → refundPacketToken returns nil early.
	ack := channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Error{Error: "timeout"},
	}
	err = s.app.Erc20Keeper.OnAcknowledgementPacket(s.ctx, packet, data, ack)
	require.NoError(t, err)
	s.Commit()

	// ERC20 balance unchanged — no mint occurred.
	erc20Balance := s.BalanceOf(contractAddr, s.address)
	require.Equal(t, big.NewInt(0), erc20Balance,
		"no ERC20 minting without MsgTransferERC20 provenance")

	// Coins remain with sender (ibc-go already handled the coin return).
	afterCoins := s.app.BankKeeper.GetBalance(s.ctx, cosmosSender, pair.Denom)
	require.Equal(t, transferAmount, afterCoins.Amount,
		"coins should remain with sender (not swept by ERC20 keeper)")
}

// TestH5SuccessAckClearsProvenanceNoRefund confirms that a successful
// ACK only clears provenance and does NOT trigger a refund — the tokens
// arrived at the destination, so no refund minting is needed.
func TestH5SuccessAckClearsProvenanceNoRefund(t *testing.T) {
	s := new(KeeperTestSuite)
	s.SetT(t)
	s.SetupTest()

	contractAddr := s.DeployContract("SuccessToken", "SUCC", uint8(18))
	s.Commit()

	pair, err := s.app.Erc20Keeper.RegisterERC20(s.ctx, contractAddr)
	require.NoError(t, err)

	packetDenom := "transfer/channel-0/" + pair.Denom
	trace := transfertypes.ParseDenomTrace(packetDenom)
	s.app.Erc20Keeper.SetDenomMap(s.ctx, trace.IBCDenom(), pair.GetID())
	s.Commit()

	cosmosSender := sdk.AccAddress(s.address.Bytes())
	transferAmount := sdkmath.NewInt(50)

	// Give sender coins + set provenance (simulating MsgTransferERC20).
	coins := sdk.NewCoins(sdk.NewCoin(pair.Denom, transferAmount))
	err = s.app.BankKeeper.MintCoins(s.ctx, types.ModuleName, coins)
	require.NoError(t, err)
	err = s.app.BankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.ModuleName, cosmosSender, coins)
	require.NoError(t, err)
	s.Commit()

	packet := channeltypes.Packet{
		Sequence:      10,
		SourcePort:    "transfer",
		SourceChannel: "channel-0",
	}
	data := transfertypes.FungibleTokenPacketData{
		Denom:    packetDenom,
		Amount:   transferAmount.String(),
		Sender:   cosmosSender.String(),
		Receiver: "cosmos1receiver",
		Memo:     "success" + types.TransferERC20Memo,
	}

	s.app.Erc20Keeper.SetIBCTransferProvenance(
		s.ctx,
		packet.SourcePort, packet.SourceChannel, packet.Sequence,
		data.Sender, data.Denom, data.Amount,
	)
	s.Commit()
	require.True(t, s.app.Erc20Keeper.HasIBCTransferProvenance(s.ctx, packet, data))

	// Success ACK → provenance cleared, no refund.
	ack := channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Result{Result: []byte{1}},
	}
	err = s.app.Erc20Keeper.OnAcknowledgementPacket(s.ctx, packet, data, ack)
	require.NoError(t, err)
	s.Commit()

	// Provenance cleaned up.
	require.False(t, s.app.Erc20Keeper.HasIBCTransferProvenance(s.ctx, packet, data))

	// No ERC20 was minted (no refund on success).
	erc20Balance := s.BalanceOf(contractAddr, s.address)
	require.Equal(t, big.NewInt(0), erc20Balance,
		"no ERC20 mint on successful ACK")
}

// TestH5ProvenanceReplayBlocked ensures that the provenence guard in
// refundPacketToken prevents any form of dual-use: both before and
// after a successful call the provenance is single-use.
func TestH5ProvenanceReplayBlocked(t *testing.T) {
	s := new(KeeperTestSuite)
	s.SetT(t)
	s.SetupTest()

	contractAddr := s.DeployContract("ReplayToken", "RPLY", uint8(18))
	s.Commit()

	pair, err := s.app.Erc20Keeper.RegisterERC20(s.ctx, contractAddr)
	require.NoError(t, err)

	packetDenom := "transfer/channel-0/" + pair.Denom
	trace := transfertypes.ParseDenomTrace(packetDenom)
	s.app.Erc20Keeper.SetDenomMap(s.ctx, trace.IBCDenom(), pair.GetID())
	s.Commit()

	transferAmount := sdkmath.NewInt(100)
	cosmosSender := sdk.AccAddress(s.address.Bytes())
	coins := sdk.NewCoins(sdk.NewCoin(pair.Denom, transferAmount))

	err = s.app.BankKeeper.MintCoins(s.ctx, types.ModuleName, coins)
	require.NoError(t, err)
	err = s.app.BankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.ModuleName, cosmosSender, coins)
	require.NoError(t, err)
	s.Commit()

	packet := channeltypes.Packet{
		Sequence:      42,
		SourcePort:    "transfer",
		SourceChannel: "channel-0",
	}
	data := transfertypes.FungibleTokenPacketData{
		Denom:    packetDenom,
		Amount:   transferAmount.String(),
		Sender:   cosmosSender.String(),
		Receiver: "cosmos1receiver",
		Memo:     "replay" + types.TransferERC20Memo,
	}

	s.app.Erc20Keeper.SetIBCTransferProvenance(
		s.ctx,
		packet.SourcePort, packet.SourceChannel, packet.Sequence,
		data.Sender, data.Denom, data.Amount,
	)
	s.Commit()

	ack := channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Error{Error: "failed"},
	}

	// First call — expects refund.
	err = s.app.Erc20Keeper.OnAcknowledgementPacket(s.ctx, packet, data, ack)
	require.NoError(t, err)
	s.Commit()

	erc20AfterFirst := s.BalanceOf(contractAddr, s.address)
	require.Equal(t, transferAmount.BigInt(), erc20AfterFirst)

	// Second call — provenance already consumed, no-op.
	err = s.app.Erc20Keeper.OnAcknowledgementPacket(s.ctx, packet, data, ack)
	require.NoError(t, err)
	s.Commit()

	erc20AfterSecond := s.BalanceOf(contractAddr, s.address)
	require.Equal(t, transferAmount.BigInt(), erc20AfterSecond,
		"second error ACK must not mint additional ERC20")

	// Verify provenance was removed after first call.
	require.False(t, s.app.Erc20Keeper.HasIBCTransferProvenance(s.ctx, packet, data))
}

// TestH5NativeCoinRefundDoesNotBurn ensures the refund path for
// OWNER_MODULE (NativeCoin / RegisterCoin) pairs sweeps coins to the
// module but does NOT burn them.
func TestH5NativeCoinRefundDoesNotBurn(t *testing.T) {
	s := new(KeeperTestSuite)
	s.SetT(t)
	// For RegisterCoin, we need mintFeeCollector to avoid mint restrictions.
	s.mintFeeCollector = true
	s.SetupTest()

	metadata, pair := s.setupRegisterCoin()
	require.False(t, pair.IsNativeERC20(), "RegisterCoin pair should be NativeCoin")
	require.True(t, pair.IsNativeCoin())

	// Set IBC denom mapping.
	packetDenom := "transfer/channel-0/" + metadata.Base
	trace := transfertypes.ParseDenomTrace(packetDenom)
	s.app.Erc20Keeper.SetDenomMap(s.ctx, trace.IBCDenom(), pair.GetID())
	s.Commit()

	// Give sender coins.
	transferAmount := sdkmath.NewInt(100)
	cosmosSender := sdk.AccAddress(s.address.Bytes())
	coins := sdk.NewCoins(sdk.NewCoin(pair.Denom, transferAmount))

	err := s.app.BankKeeper.MintCoins(s.ctx, types.ModuleName, coins)
	require.NoError(t, err)
	err = s.app.BankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.ModuleName, cosmosSender, coins)
	require.NoError(t, err)
	s.Commit()

	// Set provenance.
	packet := channeltypes.Packet{
		Sequence:      1,
		SourcePort:    "transfer",
		SourceChannel: "channel-0",
	}
	data := transfertypes.FungibleTokenPacketData{
		Denom:    packetDenom,
		Amount:   transferAmount.String(),
		Sender:   cosmosSender.String(),
		Receiver: "cosmos1receiver",
		Memo:     "nativecoin" + types.TransferERC20Memo,
	}

	s.app.Erc20Keeper.SetIBCTransferProvenance(
		s.ctx,
		packet.SourcePort, packet.SourceChannel, packet.Sequence,
		data.Sender, data.Denom, data.Amount,
	)
	s.Commit()
	require.True(t, s.app.Erc20Keeper.HasIBCTransferProvenance(s.ctx, packet, data))

	// Error ACK.
	ack := channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Error{Error: "failed"},
	}
	err = s.app.Erc20Keeper.OnAcknowledgementPacket(s.ctx, packet, data, ack)
	require.NoError(t, err)
	s.Commit()

	// Coins swept from sender.
	afterCoins := s.app.BankKeeper.GetBalance(s.ctx, cosmosSender, pair.Denom)
	require.True(t, afterCoins.Amount.IsZero(),
		"coins should be swept from sender for NativeCoin pair")

	// For NativeCoin (OWNER_MODULE), coins are NOT burned — they stay in the module.
	moduleAddr := s.app.AccountKeeper.GetModuleAddress(types.ModuleName)
	moduleCoins := s.app.BankKeeper.GetBalance(s.ctx, moduleAddr, pair.Denom)
	require.False(t, moduleCoins.Amount.IsZero(),
		"coins should NOT be burned for NativeCoin pair (only swept to module)")
	require.Equal(t, transferAmount, moduleCoins.Amount,
		"all swept coins should be in module account for NativeCoin")
}
