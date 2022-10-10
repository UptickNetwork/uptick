package contracts

import (
	_ "embed" // embed compiled smart contract
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	evmtypes "github.com/evmos/ethermint/x/evm/types"

	"github.com/UptickNetwork/uptick/x/erc20/types"
)

var (
	//go:embed compiled_contracts/ERC721PresetMinterPauserAutoId.json
	ERC721PresetMinterPauserAutoIdJSON []byte // nolint: golint

	// ERC721PresetMinterPauserAutoIdsContract is the compiled erc721 contract
	ERC721PresetMinterPauserAutoIdsContract evmtypes.CompiledContract

	// ERC721PresetMinterPauserAutoIdAddress is the erc721 module address
	ERC721PresetMinterPauserAutoIdAddress common.Address
)

func init() {
	ERC721PresetMinterPauserAutoIdAddress = types.ModuleAddress

	if err := json.Unmarshal(ERC721PresetMinterPauserAutoIdJSON, &ERC721PresetMinterPauserAutoIdsContract); err != nil {
		panic(err)
	}

	if len(ERC721PresetMinterPauserAutoIdsContract.Bin) == 0 {
		panic("load contract failed")
	}
}
