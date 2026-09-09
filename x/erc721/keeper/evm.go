package keeper

import (
	"math/big"

	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

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
		return types.ClassEnhance{}, sdkerrors.Wrapf(
			types.ErrABIUnpack,
			"unexpected getClassEnhanceInfo length %d, expected 7",
			len(ret),
		)
	}

	return k.classEnhanceFromReturn(ctx, contract, ret)
}

// classEnhanceFromReturn maps an ABI-unpacked getClassEnhanceInfo return onto a
// ClassEnhance. The caller guarantees len(ret) == 7.
//
// This is a separate method purely so the type-mismatch policy below is unit
// testable: abi.Unpack against the canonical signature always yields the
// declared types, so a contract answering with an unexpected type in one slot
// can only be reproduced by calling this with hand-built values (see
// evm_enhance_test.go). Kept inline, that behaviour would be both unreachable
// and untested.
//
// Policy (round 10, G-3) — the two field kinds are treated deliberately
// DIFFERENTLY:
//   - string fields (data/description/schema/uri/uriHash) degrade to "" and log
//     at Warn. Losing metadata is not a safety problem and the string zero
//     value carries no privilege. Warn rather than Debug because this silently
//     discards contract-declared data.
//   - bool fields (mintRestricted/updateRestricted) must NOT degrade. Their zero
//     value `false` means "NOT restricted" — the permissive end of the scale.
//     They reach the denom through CreateNFTClass and gate x/collection's mint
//     path, so degrading would silently switch an authorization OFF. Ambiguity
//     here therefore surfaces ErrClassEnhanceRestrictions instead.
func (k Keeper) classEnhanceFromReturn(
	ctx sdk.Context,
	contract common.Address,
	ret []interface{},
) (types.ClassEnhance, error) {
	// Comma-ok assertions: these values come from an external, user-registered
	// contract, so a mismatch must never panic the validating node.
	data, ok0 := ret[0].(string)
	description, ok1 := ret[1].(string)
	mintRestricted, ok2 := ret[2].(bool)
	schema, ok3 := ret[3].(string)
	updateRestricted, ok4 := ret[4].(bool)
	uri, ok5 := ret[5].(string)
	uriHash, ok6 := ret[6].(string)

	if !ok0 || !ok1 || !ok3 || !ok5 || !ok6 {
		// Warn, not Debug: this is silently discarding contract-declared
		// metadata, and Debug is invisible at the default log level.
		k.Logger(ctx).Warn(
			"QueryClassEnhance: unexpected string field type in getClassEnhanceInfo return, degrading to empty",
			"contract", contract.String(),
			"ok", []bool{ok0, ok1, ok3, ok5, ok6},
		)
	}

	if !ok2 || !ok4 {
		return types.ClassEnhance{}, sdkerrors.Wrapf(
			types.ErrClassEnhanceRestrictions,
			"cannot decode restriction flags from getClassEnhanceInfo (mint ok=%v, update ok=%v), contract %s",
			ok2, ok4, contract.String(),
		)
	}

	return types.NewClassEnhance(
		data, description, mintRestricted, schema,
		updateRestricted, uri, uriHash,
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
		}
		if len(retTokenUri) < 1 {
			return types.NewNFTEnhance("", "", "", ""), nil
		}
		uri, ok := retTokenUri[0].(string)
		if !ok {
			return types.NewNFTEnhance("", "", "", ""), nil
		}
		return types.NewNFTEnhance("", uri, "", ""), nil
	}
	if len(retEnhance) < 4 {
		return types.NFTEnhance{}, sdkerrors.Wrapf(
			types.ErrABIUnpack,
			"unexpected getNFTEnhanceInfo length %d, expected >= 4",
			len(retEnhance),
		)
	}
	enhance := make([]string, 4)
	for i := 0; i < 4; i++ {
		s, ok := retEnhance[i].(string)
		if !ok {
			enhance[i] = ""
			continue
		}
		enhance[i] = s
	}
	return types.NewNFTEnhance(enhance[0], enhance[1], enhance[2], enhance[3]), nil
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
	//
	// The internal budget must also be capped at the SDK transaction's
	// remaining gas. A fixed 25M cap lets a caller-supplied external contract
	// (e.g. name() during RegisterERC721) burn the full budget inside the EVM
	// interpreter before the post-hoc ConsumeGas below ever runs — the SDK gas
	// meter only panics afterwards, and CPU already spent cannot be rolled
	// back. min(remaining, DefaultGasCap) bounds execution to what the tx
	// actually paid for. Infinite gas meters (queries, genesis, simulation)
	// report math.MaxUint64 from GasRemaining and keep the full cap, so there
	// is no unsigned underflow and no special casing needed.
	gasCap := config.DefaultGasCap
	if remaining := ctx.GasMeter().GasRemaining(); remaining < gasCap {
		gasCap = remaining
	}

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

	res, err := k.evmKeeper.ApplyMessage(ctx, stateDB, msg, evmtypes.NewNoOpTracer(), commit, false, true)
	if err != nil {
		return nil, err
	}

	// Charge the EVM gas used back to the SDK gas meter. Passing internal=true
	// makes cosmos/evm report the true gas used (skipping the fee-market
	// min-gas floor), so a native-module EVM call counts against the transaction
	// budget instead of exposing an unbounded 25M×N CPU surface.
	if res != nil && res.GasUsed > 0 {
		ctx.GasMeter().ConsumeGas(res.GasUsed, "erc721 evm call")
	}

	if res == nil {
		return nil, sdkerrors.Wrap(evmtypes.ErrVMExecution, "nil response from ApplyMessage")
	}

	if res.Failed() {
		return nil, sdkerrors.Wrap(evmtypes.ErrVMExecution, res.VmError)
	}

	return res, nil
}
