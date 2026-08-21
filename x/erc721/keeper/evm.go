package keeper

import (
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/cosmos/evm/server/config"
	"github.com/cosmos/evm/x/vm/statedb"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	sdkerrors "cosmossdk.io/errors"
	"github.com/UptickNetwork/uptick/x/erc721/contracts"
	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// erc721StateDBKeeper wraps the EVM keeper's StateDB storage interface so that
// the erc721 module account can be updated as an EVM sender without triggering
// cosmos/evm's module-account balance guard. ERC721 contract deployments and
// conversion calls always use a zero native value, so only the module account
// nonce needs to be persisted.
type erc721StateDBKeeper struct {
	statedb.Keeper
	accountKeeper types.AccountKeeper
}

// SetAccount delegates non-module accounts to the EVM keeper. For the erc721
// module account it updates only the sequence and leaves the bank balance
// untouched, avoiding the SetBalanceWithLocked module-account restriction.
func (w erc721StateDBKeeper) SetAccount(ctx sdk.Context, addr common.Address, account statedb.Account) error {
	if addr != types.ModuleAddress {
		return w.Keeper.SetAccount(ctx, addr, account)
	}

	acct := w.accountKeeper.GetAccount(ctx, addr.Bytes())
	if acct == nil {
		return sdkerrors.Wrap(errortypes.ErrUnknownAddress, "erc721 module account not found")
	}

	if !evmtypes.IsEmptyCodeHash(account.CodeHash) {
		return sdkerrors.Wrap(errortypes.ErrUnauthorized, "erc721 module account cannot be assigned contract code")
	}

	current := w.Keeper.GetAccount(ctx, addr)
	if current == nil || current.Balance == nil || account.Balance == nil || current.Balance.Cmp(account.Balance) != 0 {
		return sdkerrors.Wrap(errortypes.ErrUnauthorized, "erc721 module account balance cannot be updated")
	}

	if err := acct.SetSequence(account.Nonce); err != nil {
		return err
	}

	w.accountKeeper.SetAccount(ctx, acct)
	return nil
}

// DeployERC721Contract creates and deploys an ERC721 contract on the EVM with the
// erc20 module account as owner.
func (k Keeper) DeployERC721Contract(
	ctx sdk.Context,
	msg *types.MsgConvertNFT,
) (common.Address, error) {

	class, err := k.nftKeeper.GetDenomInfo(ctx, msg.ClassId)
	if err != nil {
		return common.Address{}, sdkerrors.Wrapf(types.ErrABIPack, "nft class is invalid %s: %s", msg.ClassId, err.Error())
	}

	ctorArgs, err := contracts.ERC721UpticksContract.ABI.Pack(
		"",
		class.Name,
		class.Symbol,
		class.Uri,
		class.Data,
		class.Description,
		class.MintRestricted,
		class.Schema,
		class.UpdateRestricted,
		class.UriHash,
	)
	if err != nil {
		return common.Address{}, sdkerrors.Wrapf(types.ErrABIPack, "nft class is invalid %s: %s", class.Id, err.Error())
	}

	data := make([]byte, len(contracts.ERC721UpticksContract.Bin)+len(ctorArgs))
	copy(data[:len(contracts.ERC721UpticksContract.Bin)], contracts.ERC721UpticksContract.Bin)
	copy(data[len(contracts.ERC721UpticksContract.Bin):], ctorArgs)

	nonce, err := k.accountKeeper.GetSequence(ctx, types.ModuleAddress.Bytes())
	if err != nil {
		return common.Address{}, err
	}

	contractAddr := crypto.CreateAddress(types.ModuleAddress, nonce)
	if _, err = k.CallEVMWithData(ctx, types.ModuleAddress, nil, data, true); err != nil {
		return common.Address{}, sdkerrors.Wrapf(err, "failed to deploy contract for %s", class.Id)
	}

	return contractAddr, nil
}

// QueryERC721 returns the data of a deployed ERC721 contract
func (k Keeper) QueryERC721(
	ctx sdk.Context,
	contract common.Address,
) (types.ERC721Data, error) {
	var (
		nameRes   types.ERC721StringResponse
		symbolRes types.ERC721StringResponse
	)

	erc721 := contracts.ERC721UpticksContract.ABI

	// Name
	res, err := k.CallEVM(ctx, erc721, types.ModuleAddress, contract, false, "name")
	if err != nil {
		return types.ERC721Data{}, err
	}

	if err := erc721.UnpackIntoInterface(&nameRes, "name", res.Ret); err != nil {
		return types.ERC721Data{}, sdkerrors.Wrapf(
			types.ErrABIUnpack, "failed to unpack name: %s", err.Error(),
		)
	}

	// Symbol
	res, err = k.CallEVM(ctx, erc721, types.ModuleAddress, contract, false, "symbol")
	if err != nil {
		return types.ERC721Data{}, err
	}

	if err := erc721.UnpackIntoInterface(&symbolRes, "symbol", res.Ret); err != nil {
		return types.ERC721Data{}, sdkerrors.Wrapf(
			types.ErrABIUnpack, "failed to unpack symbol: %s", err.Error(),
		)
	}

	return types.NewERC721Data(nameRes.Value, symbolRes.Value), nil
}

// QueryClassEnhance returns the data of a deployed ERC721 contract
func (k Keeper) QueryClassEnhance(
	ctx sdk.Context,
	contract common.Address,
) (types.ClassEnhance, error) {

	erc721 := contracts.ERC721UpticksContract.ABI

	// Name
	res, err := k.CallEVM(ctx, erc721, types.ModuleAddress, contract, false, "getClassEnhanceInfo")
	if err != nil {
		return types.ClassEnhance{}, err
	}

	ret, err := erc721.Unpack("getClassEnhanceInfo", res.Ret)
	if err != nil {
		return types.ClassEnhance{}, sdkerrors.Wrapf(types.ErrABIUnpack, "failed to unpack getClassEnhanceInfo: %s", err.Error())
	}

	if len(ret) != 7 {
		return types.ClassEnhance{}, nil
	}

	return types.NewClassEnhance(
		ret[0].(string), ret[1].(string), ret[2].(bool), ret[3].(string),
		ret[4].(bool), ret[5].(string), ret[6].(string),
	), nil
}

// QueryNFTEnhance returns the data of a deployed ERC721 contract
func (k Keeper) QueryNFTEnhance(
	ctx sdk.Context,
	contract common.Address,
	tokenID *big.Int,
) (types.NFTEnhance, error) {

	retEnhance, err := k.QueryERC721DataByTokenID("getNFTEnhanceInfo", ctx, contract, tokenID)
	if err != nil {
		retTokenUri, err := k.QueryERC721DataByTokenID("tokenURI", ctx, contract, tokenID)
		if err != nil {
			return types.NFTEnhance{}, err
		} else {
			return types.NewNFTEnhance("", retTokenUri[0].(string), "", ""), nil
		}
	} else {
		return types.NewNFTEnhance(retEnhance[0].(string), retEnhance[1].(string), retEnhance[2].(string), retEnhance[3].(string)), nil
	}
}

func (k Keeper) QueryERC721DataByTokenID(
	queryFuncName string,
	ctx sdk.Context,
	contract common.Address,
	tokenID *big.Int) ([]interface{}, error) {

	// Query the token data from the ERC721 contract.
	erc721 := contracts.ERC721UpticksContract.ABI
	res, err := k.CallEVM(ctx, erc721, types.ModuleAddress, contract, false, queryFuncName, tokenID)
	if err != nil {
		return nil, err
	}
	ret, err := erc721.Unpack(queryFuncName, res.Ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// QueryERC721Token returns the data of a ERC721 token
func (k Keeper) QueryERC721Token(
	ctx sdk.Context,
	contract common.Address,
) (types.ERC721TokenData, error) {
	var (
		nameRes   types.ERC721TokenStringResponse
		symbolRes types.ERC721TokenStringResponse
	)

	erc721 := contracts.ERC721UpticksContract.ABI

	// Name
	res, err := k.CallEVM(ctx, erc721, types.ModuleAddress, contract, false, "name")
	if err != nil {
		return types.ERC721TokenData{}, err
	}

	if err := erc721.UnpackIntoInterface(&nameRes, "name", res.Ret); err != nil {
		return types.ERC721TokenData{}, sdkerrors.Wrapf(
			types.ErrABIUnpack, "failed to unpack name: %s", err.Error(),
		)
	}

	// Symbol
	res, err = k.CallEVM(ctx, erc721, types.ModuleAddress, contract, false, "symbol")
	if err != nil {
		return types.ERC721TokenData{}, err
	}

	if err := erc721.UnpackIntoInterface(&symbolRes, "symbol", res.Ret); err != nil {
		return types.ERC721TokenData{}, sdkerrors.Wrapf(
			types.ErrABIUnpack, "failed to unpack symbol: %s", err.Error(),
		)
	}

	// tokenURI requires a tokenId; this helper is a contract probe and does not
	// take one, so URI is left empty.
	return types.NewERC721TokenData(nameRes.Value, symbolRes.Value, ""), nil
}

// QueryERC721TokenOwner returns the owner of given tokenID
func (k Keeper) QueryERC721TokenOwner(
	ctx sdk.Context,
	contract common.Address,
	tokenID *big.Int,
) (common.Address, error) {
	var ownerRes types.ERC721TokenOwnerResponse

	erc721 := contracts.ERC721UpticksContract.ABI

	// Name
	res, err := k.CallEVM(ctx, erc721, types.ModuleAddress, contract, false, "ownerOf", tokenID)
	if err != nil {
		return common.Address{}, err
	}

	if err := erc721.UnpackIntoInterface(&ownerRes, "ownerOf", res.Ret); err != nil {
		return common.Address{}, sdkerrors.Wrapf(
			types.ErrABIUnpack, "failed to unpack owner: %s", err.Error(),
		)
	}

	return ownerRes.Value, nil
}

// CallEVM performs a smart contract method call using given args
func (k Keeper) CallEVM(
	ctx sdk.Context,
	abi abi.ABI,
	from, contract common.Address,
	commit bool,
	method string,
	args ...interface{},
) (*evmtypes.MsgEthereumTxResponse, error) {

	data, err := abi.Pack(method, args...)
	if err != nil {
		return nil, sdkerrors.Wrap(
			types.ErrABIPack,
			sdkerrors.Wrap(err, "failed to create transaction data").Error(),
		)
	}

	resp, err := k.CallEVMWithData(ctx, from, &contract, data, commit)
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "contract call failed: method '%s', contract '%s'", method, contract)
	}
	return resp, nil
}

// CallEVMWithData performs a smart contract method call using contract data
func (k Keeper) CallEVMWithData(
	ctx sdk.Context,
	from common.Address,
	contract *common.Address,
	data []byte,
	commit bool,
) (*evmtypes.MsgEthereumTxResponse, error) {
	nonce, err := k.accountKeeper.GetSequence(ctx, from.Bytes())
	if err != nil {
		return nil, err
	}

	// NOTE: Do NOT use EstimateGas here. EstimateGasInternal simulates the call
	// with the official *Keeper (not the erc721StateDBKeeper wrapper) and calls
	// k.SetAccount directly on the module account, which trips cosmos/evm's
	// SetBalanceWithLocked module-account guard ("not allowed to receive funds").
	// Deployment / mint / transfer all use the module account as sender with
	// zero value, so a fixed gas cap (DefaultGasCap) is sufficient.
	gasCap := config.DefaultGasCap

	msg := core.Message{
		From:            from,
		To:              contract,
		Nonce:           nonce,
		Value:           big.NewInt(0),
		GasLimit:        gasCap,
		GasPrice:        big.NewInt(0),
		GasFeeCap:       big.NewInt(0),
		GasTipCap:       big.NewInt(0),
		Data:            data,
		AccessList:      ethtypes.AccessList{},
		SkipNonceChecks: !commit,
	}

	stateDB := statedb.New(
		ctx,
		erc721StateDBKeeper{
			Keeper:        k.evmKeeper,
			accountKeeper: k.accountKeeper,
		},
		statedb.NewEmptyTxConfig(),
	)

	res, err := k.evmKeeper.ApplyMessage(ctx, stateDB, msg, evmtypes.NewNoOpTracer(), commit, false, false)
	if err != nil {
		return nil, err
	}

	if res.Failed() {
		return nil, sdkerrors.Wrap(evmtypes.ErrVMExecution, res.VmError)
	}

	return res, nil
}
