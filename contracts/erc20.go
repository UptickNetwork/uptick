package contracts

import (
	_ "embed" // embed compiled smart contract
	"encoding/json"

	evmtypes "github.com/cosmos/evm/x/vm/types"


	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

var (
	//go:embed compiled_contracts/ERC20MinterBurnerDecimals.json
	ERC20MinterBurnerDecimalsJSON []byte // nolint: golint

	// ERC20MinterBurnerDecimalsContract is the compiled erc20 contract
	ERC20MinterBurnerDecimalsContract evmtypes.CompiledContract

	// ERC20MinterBurnerDecimalsAddress is the erc20 module address
	ERC20MinterBurnerDecimalsAddress common.Address

	//go:embed compiled_contracts/ERC20DirectBalanceManipulation.json
	ERC20DirectBalanceManipulationJSON []byte // nolint: golint

	// ERC20DirectBalanceManipulationContract is the malicious compiled erc20 contract
	ERC20DirectBalanceManipulationContract evmtypes.CompiledContract

	//go:embed compiled_contracts/ERC20MaliciousDelayed.json
	ERC20MaliciousDelayedJSON []byte // nolint: golint

	// ERC20MaliciousDelayedContract is the malicious compiled erc20 contract
	ERC20MaliciousDelayedContract evmtypes.CompiledContract
)

func init() {
	ERC20MinterBurnerDecimalsAddress = common.BytesToAddress(authtypes.NewModuleAddress("erc20").Bytes())

	if err := json.Unmarshal(ERC20MinterBurnerDecimalsJSON, &ERC20MinterBurnerDecimalsContract); err != nil {
		panic(err)
	}

	if len(ERC20MinterBurnerDecimalsContract.Bin) == 0 {
		panic("load contract failed")
	}

	if err := json.Unmarshal(ERC20DirectBalanceManipulationJSON, &ERC20DirectBalanceManipulationContract); err != nil {
		panic(err)
	}

	if len(ERC20DirectBalanceManipulationContract.Bin) == 0 {
		panic("load DirectBalanceManipulation contract failed")
	}

	if err := json.Unmarshal(ERC20MaliciousDelayedJSON, &ERC20MaliciousDelayedContract); err != nil {
		panic(err)
	}

	if len(ERC20MaliciousDelayedContract.Bin) == 0 {
		panic("load MaliciousDelayed contract failed")
	}
}
