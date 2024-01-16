package keeper

import (
	"encoding/json"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/evmos/ethermint/server/config"
	evmtypes "github.com/evmos/ethermint/x/evm/types"

	"github.com/UptickNetwork/uptick/contracts"
	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// DeployERC721Contract creates and deploys an ERC721 contract on the EVM with the
// erc20 module account as owner.
func (k Keeper) DeployERC721Contract(
	ctx sdk.Context,
	msg *types.MsgConvertNFT,
) (common.Address, error) {

	class, err := k.nftKeeper.GetDenomInfo(ctx, msg.ClassId)
	if err != nil {
		return common.Address{}, sdkerrors.Wrapf(types.ErrABIPack, "nft class is invalid %s: %s", class.Id, err.Error())
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
		k.Logger(ctx).Error("QueryClassEnhance resRet", "error", err.Error())
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
			return types.NFTEnhance{}, nil
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

	//uptick bug fix: 11/07 get from tokenUri
	erc721 := contracts.ERC721UpticksContract.ABI
	res, err := k.CallEVM(ctx, erc721, types.ModuleAddress, contract, true, queryFuncName, tokenID)
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
		uriRes    types.ERC721TokenStringResponse
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

	if err := erc721.UnpackIntoInterface(&symbolRes, "symbol", res.Ret); err != nil {
		return types.ERC721TokenData{}, sdkerrors.Wrapf(
			types.ErrABIUnpack, "failed to unpack uri: %s", err.Error(),
		)
	}

	return types.NewERC721TokenData(nameRes.Value, symbolRes.Value, uriRes.Value), nil
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

	gasCap := config.DefaultGasCap
	if commit {
		args, err := json.Marshal(evmtypes.TransactionArgs{
			From: &from,
			To:   contract,
			Data: (*hexutil.Bytes)(&data),
		})
		if err != nil {
			return nil, sdkerrors.Wrapf(sdkerrors.ErrJSONMarshal, "failed to marshal tx args: %s", err.Error())
		}

		gasRes, err := k.evmKeeper.EstimateGas(sdk.WrapSDKContext(ctx), &evmtypes.EthCallRequest{
			Args:   args,
			GasCap: config.DefaultGasCap,
		})
		if err != nil {
			return nil, err
		}
		gasCap = gasRes.Gas
	}

	msg := ethtypes.NewMessage(
		from,
		contract,
		nonce,
		big.NewInt(0), // amount
		gasCap,        // gasLimit
		big.NewInt(0), // gasFeeCap
		big.NewInt(0), // gasTipCap
		big.NewInt(0), // gasPrice
		data,
		ethtypes.AccessList{}, // AccessList
		!commit,               // isFake
	)

	res, err := k.evmKeeper.ApplyMessage(ctx, msg, evmtypes.NewNoOpTracer(), commit)
	if err != nil {
		return nil, err
	}

	if res.Failed() {
		return nil, sdkerrors.Wrap(evmtypes.ErrVMExecution, res.VmError)
	}

	return res, nil
}
