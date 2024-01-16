package keeper

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	"math/big"

	"github.com/UptickNetwork/uptick/x/cw721/types"
)

// ContractInfo {"data":{"name":"uptick test collection","symbol":"uptick-test-01"}}
type ContractInfo struct {
	Data ContractInfoData `json:"data"`
}

type ContractInfoData struct {
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}
type Result struct {
	Data *string `json:"data,omitempty"`
}

// QueryCW721 returns the data of a deployed CW721 contract
func (k Keeper) QueryCW721(
	ctx sdk.Context,
	contractAddress string,
) (types.CW721Data, error) {

	contractInfo := make(map[string]Result)
	contractInfo["contract_info"] = Result{}
	jsonStr, err := json.Marshal(contractInfo)

	if err != nil {
		return types.CW721Data{}, err
	} else {
		contractInfoResult, err := k.QueryWasmState(ctx,
			&wasmtypes.QuerySmartContractStateRequest{
				Address:   contractAddress,
				QueryData: jsonStr,
			})

		if err != nil {
			return types.CW721Data{}, err
		}

		var contractInfoResultJson ContractInfoData
		json.Unmarshal(contractInfoResult.Data, &contractInfoResultJson)

		return types.CW721Data{
			Name:   contractInfoResultJson.Name,
			Symbol: contractInfoResultJson.Symbol,
		}, nil

	}

}

type AllNftInfo struct {
	Access AllNftAccess `json:"access"`
	Info   AllNftData   `json:"info"`
}

type AllNftAccess struct {
	Owner string `json:"owner"`
}

type AllNftData struct {
	TokenUri  string      `json:"token_uri"`
	Extension interface{} `json:"extension"`
}

// QueryCW721AllNftInfo returns the owner of given tokenID
// QueryCW721TokenOwner -> QueryCW721TokenOwner
func (k Keeper) QueryCW721AllNftInfo(
	ctx sdk.Context,
	contractAddress string,
	tokenId string,
) (AllNftInfo, error) {

	// `{"all_nft_info":{"token_id":"abc125"}}`
	allNftInfoCondition := make(map[string]map[string]string)
	subCondition := make(map[string]string)
	subCondition["token_id"] = tokenId
	allNftInfoCondition["all_nft_info"] = subCondition

	jsonConditionStr, err := json.Marshal(allNftInfoCondition)
	if err != nil {
		return AllNftInfo{}, err
	} else {
		allNftInfo, err := k.QueryWasmState(ctx,
			&wasmtypes.QuerySmartContractStateRequest{
				Address:   contractAddress,
				QueryData: jsonConditionStr,
			})
		if err != nil {
			return AllNftInfo{}, err
		}

		var allContractInfoResultJson AllNftInfo
		json.Unmarshal(allNftInfo.Data, &allContractInfoResultJson)

		return allContractInfoResultJson, nil
	}

}

type StoreCache struct {
	sync.Mutex
	contracts map[string][]byte
}

var contractsCache = StoreCache{contracts: make(map[string][]byte)}

func getContractBytes(contract string) ([]byte, error) {
	contractsCache.Lock()
	bz, found := contractsCache.contracts[contract]
	contractsCache.Unlock()
	if found {
		return bz, nil
	}
	contractsCache.Lock()
	defer contractsCache.Unlock()

	var err error
	bz, err = getBytesFromUrl(contract)
	if err != nil {
		return nil, err
	}

	contractsCache.contracts[contract] = bz
	return bz, nil
}

func getBytesFromUrl(url string) ([]byte, error) {

	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	// bz, err := ioutil.ReadAll(response.Body)

	bz, err := ioutil.ReadFile("/Users/xuxinlai/my/mul/gon/v2/ics721-setup/cw721_base.wasm")
	return bz, err

}

// StoreWasmContract creates and deploys an CW721 contract on the EVM with the
// erc721 module account as owner.
func (k Keeper) StoreWasmContract(
	ctx sdk.Context,
	contractFile string,
	creator string,
) (uint64, error) {

	bin, err := getContractBytes(contractFile)
	if err != nil {
		k.Logger(ctx).Error("getContractBytes ", "err :", err)
		return 0, err
	}

	// wasmkeeper.NewDefaultPermissionKeeper()
	wasmMsgServer := wasmkeeper.NewMsgServerImpl(&k.cwKeeper)
	res, err := wasmMsgServer.StoreCode(sdk.WrapSDKContext(ctx), &wasmtypes.MsgStoreCode{
		Sender:       creator,
		WASMByteCode: bin,
	})
	if err != nil {
		k.Logger(ctx).Error("StoreCode ", "err :", err)
		return 0, err
	}
	return res.CodeID, nil
}

type InstantiateInfo struct {
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Minter string `json:"minter"`
}

// InstantiateWasmContract creates and deploys an CW721 contract on the EVM with the
// erc721 module account as owner.
func (k Keeper) InstantiateWasmContract(
	ctx sdk.Context,
	senderAddr string,
	codeId uint64,
	label string,
	name string,
	symbol string,
	minter string,
) (string, error) {

	var instantiateInfo InstantiateInfo
	instantiateInfo.Name = name
	instantiateInfo.Symbol = symbol
	instantiateInfo.Minter = minter
	instantiateInfoJsonStr, err := json.Marshal(instantiateInfo)
	if err != nil {
		return "", err
	}
	wasmMsgServer := wasmkeeper.NewMsgServerImpl(&k.cwKeeper)
	initMsg := wasmtypes.MsgInstantiateContract{
		Sender: senderAddr, Admin: senderAddr, CodeID: codeId,
		Label: label, Msg: wasmtypes.RawContractMessage(instantiateInfoJsonStr),
		Funds: sdk.NewCoins(),
	}
	res, err := wasmMsgServer.InstantiateContract(sdk.WrapSDKContext(ctx), &initMsg)
	if err != nil {
		return "", err
	}
	return res.Address, nil
}

// MintInfo {"mint":{"token_id":"abc126","owner":"uptick100s3yp8l3atuuvx98jmftttxzy4ee5mg2n79fx","token_uri":"http://test.com"}}
type MintInfo struct {
	Mint MintInfoData `json:"mint"`
}

type MintInfoData struct {
	TokenId  string `json:"token_id"`
	Owner    string `json:"owner"`
	TokenUri string `json:"token_uri"`
}

// MintCw721 the contract and get the result
func (k Keeper) MintCw721(
	ctx sdk.Context,
	contractAddress string,
	tokenId string,
	sender string,
	tokenUri string,
) (*wasmtypes.MsgExecuteContractResponse, error) {

	var mintInfoData MintInfoData
	var mintInfo MintInfo

	mintInfoData.TokenId = tokenId
	mintInfoData.Owner = sender
	mintInfoData.TokenUri = tokenUri

	mintInfo.Mint = mintInfoData
	mintJsonStr, err := json.Marshal(mintInfo)
	if err != nil {
		return nil, err
	} else {

		// Execute the contract
		funds := sdk.NewCoins(sdk.NewCoin("uptick", sdk.ZeroInt()))
		// msgBytes := wasmtypes.RawContractMessage(`{"mint":{"token_id":"abc126","owner":"uptick100s3yp8l3atuuvx98jmftttxzy4ee5mg2n79fx","token_uri":"http://test.com"}}`)
		execMsg := wasmtypes.MsgExecuteContract{
			Sender:   sender,
			Contract: contractAddress,
			Msg:      wasmtypes.RawContractMessage(mintJsonStr),
			Funds:    funds,
		}
		_, err := k.ExecWasmMsg(ctx, &execMsg)
		if err != nil {
			return nil, err
		} else {
			return nil, nil
		}
	}

}

// TransferNftInfo
// {"transfer_nft":{"recipient":"uptick1n3t0zuwq4u47ke48qm3pfhj96f4ujhs70f52sg","token_id":"abc123"}}
type TransferNftInfo struct {
	TransferNft TransferNftData `json:"transfer_nft"`
}

type TransferNftData struct {
	Recipient string `json:"recipient"`
	TokenId   string `json:"token_id"`
}

// TransferCw721 the contract and get the result
func (k Keeper) TransferCw721(
	ctx sdk.Context,
	contractAddress string,
	tokenId string,
	recipient string,
	sender string,
) (*wasmtypes.MsgExecuteContractResponse, error) {

	var transferNftInfo TransferNftInfo
	var transferNftData TransferNftData

	transferNftData.TokenId = tokenId
	transferNftData.Recipient = recipient
	transferNftInfo.TransferNft = transferNftData

	transferJsonStr, err := json.Marshal(transferNftInfo)
	if err != nil {
		return nil, err
	} else {

		// Execute the contract
		funds := sdk.NewCoins(sdk.NewCoin("uptick", sdk.ZeroInt()))
		execMsg := wasmtypes.MsgExecuteContract{
			Sender:   sender,
			Contract: contractAddress,
			Msg:      wasmtypes.RawContractMessage(transferJsonStr),
			Funds:    funds,
		}

		_, err := k.ExecWasmMsg(ctx, &execMsg)
		if err != nil {
			return nil, err
		} else {
			return nil, nil
		}
	}

}

// ExecWasmMsg exec the contract and get the result
func (k Keeper) ExecWasmMsg(
	ctx sdk.Context,
	execMsg *wasmtypes.MsgExecuteContract) (*wasmtypes.MsgExecuteContractResponse, error) {

	if err := execMsg.ValidateBasic(); err != nil {
		return nil, sdkerrors.Wrapf(types.ErrABIPack, "nft class is invalid %s: %s", execMsg.Msg, err.Error())
	}

	wasmMsgServer := wasmkeeper.NewMsgServerImpl(&k.cwKeeper)
	return wasmMsgServer.ExecuteContract(sdk.WrapSDKContext(ctx), execMsg)
}

// QueryWasmState for query (rsp *types.QuerySmartContractStateResponse, err error)
// func (q grpcQuerier) SmartContractState(c context.Context, req *types.QuerySmartContractStateRequest)
// (rsp *types.QuerySmartContractStateResponse, err error) {
func (k Keeper) QueryWasmState(
	ctx sdk.Context,
	req *wasmtypes.QuerySmartContractStateRequest) (*wasmtypes.QuerySmartContractStateResponse, error) {
	return wasmkeeper.Querier(&k.cwKeeper).SmartContractState(ctx, req)

}

// ---------------------------------------------------------------------------------------------------------------------

// QueryClassEnhance returns the data of a deployed CW721 contract
// TODO
func (k Keeper) QueryClassEnhance(
	ctx sdk.Context,
	contract common.Address,
) (types.ClassEnhance, error) {

	return types.ClassEnhance{}, nil
}

// QueryNFTEnhance returns the data of a deployed CW721 contract
// TODO
func (k Keeper) QueryNFTEnhance(
	ctx sdk.Context,
	contract common.Address,
	tokenID *big.Int,
) (types.NFTEnhance, error) {

	return types.NFTEnhance{}, nil
}

// QueryCW721Token returns the data of a CW721 token
// TODO
func (k Keeper) QueryCW721Token(
	ctx sdk.Context,
	contract common.Address,
) (types.CW721TokenData, error) {

	return types.CW721TokenData{}, nil
}
