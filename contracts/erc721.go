package contracts

import (
	_ "embed" // embed compiled smart contract
	"encoding/json"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

var (
	//go:embed compiled_contracts/ERC721Uptick.json
	ERC721UptickJSON []byte // nolint: golint

	// ERC721UpticksContract is the compiled erc721 contract
	ERC721UpticksContract evmtypes.CompiledContract

	// ERC721UptickAddress is the erc721 module address
	ERC721UptickAddress common.Address
)

func init() {

	ERC721UptickAddress = common.BytesToAddress(authtypes.NewModuleAddress("erc721").Bytes())

	if err := json.Unmarshal(ERC721UptickJSON, &ERC721UpticksContract); err != nil {
		panic(err)
	}

	if len(ERC721UpticksContract.Bin) == 0 {
		panic("load contract failed")
	}
}
