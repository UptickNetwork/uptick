package keeper

import (
	"context"
	"math/big"
	"strings"

	"cosmossdk.io/math"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/contracts"
	appType "github.com/UptickNetwork/uptick/types"
	"github.com/UptickNetwork/uptick/x/erc20/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

var _ types.MsgServer = &Keeper{}

// TransferERC20 converts ERC20 tokens into native Cosmos nft for both
// Cosmos-native and ERC20 TokenPair Owners and transfer through IBC
func (k Keeper) TransferERC20(
	goCtx context.Context,
	msg *types.MsgTransferERC20,
) (
	*types.MsgTransferERC20Response, error,
) {

	cosmosSender, err := appType.ConvertAddressEvm2Cosmos(msg.EvmSender)
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "failed to ConvertAddressEvm2Cosmos %v", err)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	convertMsg := types.MsgConvertERC20{
		ContractAddress: msg.EvmContractAddress,
		Amount:          msg.Amount,
		Receiver:        cosmosSender,
		Sender:          msg.EvmSender,
	}
	k.ConvertERC20(ctx, &convertMsg)
	receiver, err := sdk.AccAddressFromBech32(cosmosSender)
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "failed to AccAddressFromBech32 %v-%v", receiver, err)
	}
	sender := common.HexToAddress(msg.EvmSender)
	pair, err := k.MintingEnabled(ctx, sdk.AccAddress(sender.Bytes()), receiver, msg.EvmContractAddress)
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "failed to MintingEnabled %v", err)
	}

	coins := sdk.Coins{sdk.Coin{Denom: pair.Denom, Amount: msg.Amount}}
	ibcMsg := ibctransfertypes.MsgTransfer{
		SourcePort:       msg.SourcePort,
		SourceChannel:    msg.SourceChannel,
		Sender:           cosmosSender,
		Token:            coins[0],
		Receiver:         msg.CosmosReceiver,
		TimeoutHeight:    msg.TimeoutHeight,
		TimeoutTimestamp: msg.TimeoutTimestamp,
		Memo:             msg.Memo + types.TransferERC20Memo,
	}
	_, err = k.ibcKeeper.Transfer(goCtx, &ibcMsg)
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "failed to ibc Transfer%v", err)
	}

	return &types.MsgTransferERC20Response{}, nil

}

// ConvertCoin converts Cosmos-native Coins into ERC20 tokens for both
// Cosmos-native and ERC20 TokenPair Owners
func (k Keeper) ConvertCoin(
	goCtx context.Context,
	msg *types.MsgConvertCoin,
) (*types.MsgConvertCoinResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Error checked during msg validation
	receiver := common.HexToAddress(msg.Receiver)
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)

	pair, err := k.MintingEnabled(ctx, sender, sdk.AccAddress(receiver.Bytes()), msg.Coin.Denom)
	if err != nil {
		return nil, err
	}

	// Remove token pair if contract is suicided
	erc20 := common.HexToAddress(pair.Erc20Address)
	acc := k.evmKeeper.GetAccountWithoutBalance(ctx, erc20)

	if acc == nil || !acc.IsContract() {
		k.DeleteTokenPair(ctx, pair)
		k.Logger(ctx).Debug(
			"deleting selfdestructed token pair from state",
			"contract", pair.Erc20Address,
		)
		// NOTE: return nil error to persist the changes from the deletion
		return nil, nil
	}

	// Check ownership and execute conversion
	switch {
	case pair.IsNativeCoin():
		return k.convertCoinNativeCoin(ctx, pair, msg, receiver, sender) // case 1.1
	case pair.IsNativeERC20():
		return k.convertCoinNativeERC20(ctx, pair, msg, receiver, sender) // case 2.2
	default:
		return nil, types.ErrUndefinedOwner
	}
}

// ConvertERC20 converts ERC20 tokens into Cosmos-native Coins for both
// Cosmos-native and ERC20 TokenPair Owners
func (k Keeper) ConvertERC20(
	goCtx context.Context,
	msg *types.MsgConvertERC20,
) (*types.MsgConvertERC20Response, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Error checked during msg validation
	receiver, _ := sdk.AccAddressFromBech32(msg.Receiver)
	sender := common.HexToAddress(msg.Sender)

	pair, err := k.MintingEnabled(ctx, sdk.AccAddress(sender.Bytes()), receiver, msg.ContractAddress)
	if err != nil {
		return nil, err
	}

	// Remove token pair if contract is suicided
	erc20 := common.HexToAddress(pair.Erc20Address)
	acc := k.evmKeeper.GetAccountWithoutBalance(ctx, erc20)

	if acc == nil || !acc.IsContract() {
		k.DeleteTokenPair(ctx, pair)
		k.Logger(ctx).Debug(
			"deleting selfdestructed token pair from state",
			"contract", pair.Erc20Address,
		)
		// NOTE: return nil error to persist the changes from the deletion
		return nil, nil
	}

	// Check ownership
	switch {
	case pair.IsNativeCoin():
		return k.convertERC20NativeCoin(ctx, pair, msg, receiver, sender) // case 1.2
	case pair.IsNativeERC20():
		return k.convertERC20NativeToken(ctx, pair, msg, receiver, sender) // case 2.1
	default:
		return nil, types.ErrUndefinedOwner
	}
}

// convertCoinNativeCoin handles the Coin conversion flow for a native coin
// token pair:
//   - Escrow Coins on module account (Coins are not burned)
//   - Mint Tokens and send to receiver
//   - Check if token balance increased by amount
func (k Keeper) convertCoinNativeCoin(
	ctx sdk.Context,
	pair types.TokenPair,
	msg *types.MsgConvertCoin,
	receiver common.Address,
	sender sdk.AccAddress,
) (*types.MsgConvertCoinResponse, error) {
	// NOTE: ignore validation from NewCoin constructor
	coins := sdk.Coins{msg.Coin}
	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
	contract := pair.GetERC20Contract()
	balanceToken := k.balanceOf(ctx, erc20, contract, receiver)

	// Escrow Coins on module account
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, coins); err != nil {
		return nil, sdkerrors.Wrap(err, "failed to escrow coins")
	}

	// Mint Tokens and send to receiver
	_, err := k.CallEVM(ctx, erc20, types.ModuleAddress, contract, true, "mint", receiver, msg.Coin.Amount.BigInt())
	if err != nil {
		return nil, err
	}

	// Check expected Receiver balance after transfer execution
	tokens := msg.Coin.Amount.BigInt()
	balanceTokenAfter := k.balanceOf(ctx, erc20, contract, receiver)
	exp := big.NewInt(0).Add(balanceToken, tokens)
	if r := balanceTokenAfter.Cmp(exp); r != 0 {
		return nil, sdkerrors.Wrapf(
			types.ErrBalanceInvariance,
			"invalid token balance - expected: %v, actual: %v",
			exp, balanceTokenAfter,
		)
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertCoin,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
				sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Coin.Amount.String()),
				sdk.NewAttribute(types.AttributeKeyCosmosCoin, msg.Coin.Denom),
				sdk.NewAttribute(types.AttributeKeyERC20Token, pair.Erc20Address),
			),
		},
	)

	return &types.MsgConvertCoinResponse{}, nil
}

// convertERC20NativeCoin handles the erc20 conversion flow for a native coin token pair:
//   - Burn escrowed tokens
//   - Unescrow coins that have been previously escrowed with ConvertCoin
//   - Check if coin balance increased by amount
//   - Check if token balance decreased by amount
func (k Keeper) convertERC20NativeCoin(
	ctx sdk.Context,
	pair types.TokenPair,
	msg *types.MsgConvertERC20,
	receiver sdk.AccAddress,
	sender common.Address,
) (*types.MsgConvertERC20Response, error) {
	// NOTE: coin fields already validated
	coins := sdk.Coins{sdk.Coin{Denom: pair.Denom, Amount: msg.Amount}}

	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
	contract := pair.GetERC20Contract()
	balanceCoin := k.bankKeeper.GetBalance(ctx, receiver, pair.Denom)
	balanceToken := k.balanceOf(ctx, erc20, contract, sender)

	// Burn escrowed tokens
	_, err := k.CallEVM(ctx, erc20, types.ModuleAddress, contract, true, "burnCoins", sender, msg.Amount)
	if err != nil {
		return nil, err
	}

	// Unescrow Coins and send to receiver
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiver, coins); err != nil {
		return nil, err
	}
	// Check expected Receiver balance after transfer execution
	balanceCoinAfter := k.bankKeeper.GetBalance(ctx, receiver, pair.Denom)
	expCoin := balanceCoin.Add(coins[0])
	if ok := balanceCoinAfter.IsEqual(expCoin); !ok {
		return nil, sdkerrors.Wrapf(
			types.ErrBalanceInvariance,
			"invalid coin balance - expected: %v, actual: %v",
			expCoin, balanceCoinAfter,
		)
	}

	// Check expected Sender balance after transfer execution
	tokens := coins[0].Amount.BigInt()
	balanceTokenAfter := k.balanceOf(ctx, erc20, contract, sender)
	expToken := big.NewInt(0).Sub(balanceToken, tokens)
	if r := balanceTokenAfter.Cmp(expToken); r != 0 {
		return nil, sdkerrors.Wrapf(
			types.ErrBalanceInvariance,
			"invalid token balance - expected: %v, actual: %v",
			expToken, balanceTokenAfter,
		)
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertERC20,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
				sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.String()),
				sdk.NewAttribute(types.AttributeKeyCosmosCoin, pair.Denom),
				sdk.NewAttribute(types.AttributeKeyERC20Token, msg.ContractAddress),
			),
		},
	)

	return &types.MsgConvertERC20Response{}, nil
}

// convertERC20NativeToken handles the erc20 conversion flow for a native erc20 token pair:
//   - Escrow tokens on module account (Don't burn as module is not contract owner)
//   - Mint coins on module
//   - Send minted coins to the receiver
//   - Check if coin balance increased by amount
//   - Check if token balance decreased by amount
//   - Check for unexpected `appove` event in logs
func (k Keeper) convertERC20NativeToken(
	ctx sdk.Context,
	pair types.TokenPair,
	msg *types.MsgConvertERC20,
	receiver sdk.AccAddress,
	sender common.Address,
) (*types.MsgConvertERC20Response, error) {
	// NOTE: coin fields already validated
	coins := sdk.Coins{sdk.Coin{Denom: pair.Denom, Amount: msg.Amount}}
	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
	contract := pair.GetERC20Contract()
	balanceCoin := k.bankKeeper.GetBalance(ctx, receiver, pair.Denom)
	balanceToken := k.balanceOf(ctx, erc20, contract, types.ModuleAddress)

	// Escrow tokens on module account
	transferData, err := erc20.Pack("transfer", types.ModuleAddress, msg.Amount)
	if err != nil {
		return nil, err
	}

	res, err := k.CallEVMWithData(ctx, sender, &contract, transferData, true)
	if err != nil {
		return nil, err
	}

	// Check unpackedRet execution
	var unpackedRet types.ERC20BoolResponse
	if err := erc20.UnpackIntoInterface(&unpackedRet, "transfer", res.Ret); err != nil {
		return nil, err
	}

	if !unpackedRet.Value {
		return nil, sdkerrors.Wrap(errortypes.ErrLogic, "failed to execute transfer")
	}

	// Check expected escrow balance after transfer execution
	tokens := coins[0].Amount.BigInt()
	balanceTokenAfter := k.balanceOf(ctx, erc20, contract, types.ModuleAddress)
	expToken := big.NewInt(0).Add(balanceToken, tokens)
	if r := balanceTokenAfter.Cmp(expToken); r != 0 {
		return nil, sdkerrors.Wrapf(
			types.ErrBalanceInvariance,
			"invalid token balance - expected: %v, actual: %v",
			expToken, balanceTokenAfter,
		)
	}

	// Mint coins
	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
		return nil, err
	}

	// Send minted coins to the receiver
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiver, coins); err != nil {
		return nil, err
	}

	// Check expected Receiver balance after transfer execution
	balanceCoinAfter := k.bankKeeper.GetBalance(ctx, receiver, pair.Denom)
	expCoin := balanceCoin.Add(coins[0])
	if ok := balanceCoinAfter.IsEqual(expCoin); !ok {
		return nil, sdkerrors.Wrapf(
			types.ErrBalanceInvariance,
			"invalid coin balance - expected: %v, actual: %v",
			expCoin, balanceCoinAfter,
		)
	}

	// Check for unexpected `appove` event in logs
	if err := k.monitorApprovalEvent(res); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertERC20,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
				sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.String()),
				sdk.NewAttribute(types.AttributeKeyCosmosCoin, pair.Denom),
				sdk.NewAttribute(types.AttributeKeyERC20Token, msg.ContractAddress),
			),
		},
	)

	return &types.MsgConvertERC20Response{}, nil
}

// convertCoinNativeERC20 handles the Coin conversion flow for a native ERC20
// token pair:
//   - Escrow Coins on module account
//   - Unescrow Tokens that have been previously escrowed with ConvertERC20 and send to receiver
//   - Burn escrowed Coins
//   - Check if token balance increased by amount
//   - Check for unexpected `appove` event in logs
func (k Keeper) convertCoinNativeERC20(
	ctx sdk.Context,
	pair types.TokenPair,
	msg *types.MsgConvertCoin,
	receiver common.Address,
	sender sdk.AccAddress,
) (*types.MsgConvertCoinResponse, error) {
	// NOTE: ignore validation from NewCoin constructor
	coins := sdk.Coins{msg.Coin}

	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
	contract := pair.GetERC20Contract()
	balanceToken := k.balanceOf(ctx, erc20, contract, receiver)

	// Escrow Coins on module account
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, coins); err != nil {
		return nil, sdkerrors.Wrap(err, "failed to escrow coins")
	}

	// Unescrow Tokens and send to receiver
	res, err := k.CallEVM(ctx, erc20, types.ModuleAddress, contract, true, "transfer", receiver, msg.Coin.Amount.BigInt())
	if err != nil {
		return nil, err
	}

	// Check unpackedRet execution
	var unpackedRet types.ERC20BoolResponse
	if err := erc20.UnpackIntoInterface(&unpackedRet, "transfer", res.Ret); err != nil {
		return nil, err
	}

	if !unpackedRet.Value {
		return nil, sdkerrors.Wrap(errortypes.ErrLogic, "failed to execute unescrow tokens from user")
	}

	// Check expected Receiver balance after transfer execution
	tokens := msg.Coin.Amount.BigInt()
	balanceTokenAfter := k.balanceOf(ctx, erc20, contract, receiver)
	exp := big.NewInt(0).Add(balanceToken, tokens)
	if r := balanceTokenAfter.Cmp(exp); r != 0 {
		return nil, sdkerrors.Wrapf(
			types.ErrBalanceInvariance,
			"invalid token balance - expected: %v, actual: %v",
			exp, balanceTokenAfter,
		)
	}

	// Burn escrowed Coins
	err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to burn coins")
	}

	// Check for unexpected `appove` event in logs
	if err := k.monitorApprovalEvent(res); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertCoin,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
				sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Coin.Amount.String()),
				sdk.NewAttribute(types.AttributeKeyCosmosCoin, msg.Coin.Denom),
				sdk.NewAttribute(types.AttributeKeyERC20Token, pair.Erc20Address),
			),
		},
	)

	return &types.MsgConvertCoinResponse{}, nil
}

// balanceOf queries an account's balance for a given ERC20 contract
func (k Keeper) balanceOf(
	ctx sdk.Context,
	abi abi.ABI,
	contract, account common.Address,
) *big.Int {
	res, err := k.CallEVM(ctx, abi, types.ModuleAddress, contract, false, "balanceOf", account)
	if err != nil {
		return nil
	}

	unpacked, err := abi.Unpack("balanceOf", res.Ret)
	if err != nil || len(unpacked) == 0 {
		return nil
	}

	balance, ok := unpacked[0].(*big.Int)
	if !ok {
		return nil
	}

	return balance
}

// monitorApprovalEvent returns an error if the given transactions logs include
// an unexpected `approve` event
func (k Keeper) monitorApprovalEvent(res *evmtypes.MsgEthereumTxResponse) error {
	if res == nil || len(res.Logs) == 0 {
		return nil
	}

	logApprovalSig := []byte("Approval(address,address,uint256)")
	logApprovalSigHash := crypto.Keccak256Hash(logApprovalSig)

	for _, log := range res.Logs {
		if log.Topics[0] == logApprovalSigHash.Hex() {
			return sdkerrors.Wrapf(
				types.ErrUnexpectedEvent, "unexpected Approval event",
			)
		}
	}

	return nil
}

// refundPacketToken will unescrow and send back the tokens back to sender
// if the sending chain was the source chain. Otherwise, the sent tokens
// were burnt in the original send so new tokens are minted and sent to
// the sending address.
func (k Keeper) refundPacketToken(ctx sdk.Context, packet channeltypes.Packet, data transfertypes.FungibleTokenPacketData) error {
	// NOTE: packet data type already checked in handler.go
	if !strings.Contains(data.Memo, types.TransferERC20Memo) {
		return nil
	}

	// parse the transfer amount
	transferAmount, ok := math.NewIntFromString(data.Amount)
	if !ok {
		return sdkerrors.Wrapf(transfertypes.ErrInvalidAmount, "unable to parse transfer amount (%s) into math.Int", data.Amount)
	}

	// decode the sender address
	evmSender, err := appType.ConvertAddressCosmos2Evm(data.Sender)
	if err != nil {
		return err
	}
	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
	trace := transfertypes.ParseDenomTrace(data.Denom)

	id := k.GetDenomMap(ctx, trace.IBCDenom())
	pair, isExist := k.GetTokenPair(ctx, id)
	if !isExist {
		return sdkerrors.Wrapf(types.ErrInternalTokenPair, "pair is not exist by id  (%s)", id)
	}

	contract := pair.GetERC20Contract()
	// Mint Tokens and send to receiver
	_, err = k.CallEVM(ctx, erc20, types.ModuleAddress, contract, true, "mint", common.HexToAddress(evmSender), transferAmount.BigInt())
	if err != nil {
		return err
	}
	return nil
}
