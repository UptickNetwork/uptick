package keeper_test

import (
	"math/big"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/ethereum/go-ethereum/common"
	"github.com/cosmos/evm/x/vm/statedb"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/stretchr/testify/suite"

	"github.com/UptickNetwork/uptick/app"
	"github.com/UptickNetwork/uptick/contracts"
	"github.com/UptickNetwork/uptick/x/erc20/types"
)

// KeeperTestSuite is the primary test suite for ERC20 keeper integration tests.
// Each test file can define its own helper methods in the same package.
type KeeperTestSuite struct {
	suite.Suite

	ctx sdk.Context
	app *app.Uptick

	address     common.Address
	queryClient types.QueryClient

	// mintFeeCollector enables minting initial tokens to the fee collector
	// for tests that need to track token balances during conversions
	mintFeeCollector bool
}

// GetAccountWithoutBalance checks if a contract has been destroyed (suicided).
// NOTE(TODO): This is a temporary adapter; verify against current EvmKeeper API.
func (suite *KeeperTestSuite) GetAccountWithoutBalance(addr common.Address) interface{} {
	return suite.app.EvmKeeper.GetAccount(suite.ctx, addr)
}

// SetupTest creates a new app instance for each test.
// Individual test files may call SetupTest again within test cases.
func (suite *KeeperTestSuite) SetupTest() {
	t := suite.T()

	suite.app = app.NewUptick(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
		nil,
		baseapp.SetChainID("uptick_700-1"),
	)

	suite.ctx = suite.app.BaseApp.NewContext(false, tmproto.Header{
		Height:  1,
		Time:    time.Now().UTC(),
		ChainID: "uptick_700-1",
	})

	_, err := suite.app.InitChainer(suite.ctx, &abci.RequestInitChain{ChainId: "uptick_700-1"})
	suite.Require().NoError(err)
	suite.Commit()

	suite.address = common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")

	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, suite.app.Erc20Keeper)
	suite.queryClient = types.NewQueryClient(queryHelper)
}

// StateDB returns the EVM state database for low-level state manipulation.
func (suite *KeeperTestSuite) StateDB() *statedb.StateDB {
	return suite.app.EvmKeeper.EvmState(suite.ctx)
}

// Commit advances the chain one block.
func (suite *KeeperTestSuite) Commit() {
	err := suite.app.EvmKeeper.Commit(suite.ctx)
	suite.Require().NoError(err)

	header := suite.ctx.BlockHeader()
	header.Height++
	suite.ctx = suite.app.BaseApp.NewContext(false, header).
		WithBlockGasMeter(sdk.NewInfiniteGasMeter())
}

// MintERC20Token mints ERC20 tokens via the minter-burner contract.
// The caller and spender are provided for backward compatibility with
// existing test code that uses a 4-parameter signature.
func (suite *KeeperTestSuite) MintERC20Token(contractAddr, _ /*caller*/, to common.Address, amount *big.Int) *evmtypes.MsgEthereumTxResponse {
	res, err := suite.app.Erc20Keeper.CallEVM(
		suite.ctx,
		contracts.ERC20MinterBurnerDecimalsContract.ABI,
		types.ModuleAddress,
		contractAddr,
		true,
		"mint",
		to,
		amount,
	)
	suite.Require().NoError(err)
	return res
}

// BurnERC20Token burns ERC20 tokens held by the sender.
func (suite *KeeperTestSuite) BurnERC20Token(contractAddr, _ /*caller*/ common.Address, amount *big.Int) *evmtypes.MsgEthereumTxResponse {
	res, err := suite.app.Erc20Keeper.CallEVM(
		suite.ctx,
		contracts.ERC20MinterBurnerDecimalsContract.ABI,
		types.ModuleAddress,
		contractAddr,
		true,
		"burn",
		amount,
	)
	suite.Require().NoError(err)
	return res
}

// BalanceOf queries the ERC20 balance of an address.
func (suite *KeeperTestSuite) BalanceOf(contractAddr, account common.Address) *big.Int {
	res, err := suite.app.Erc20Keeper.CallEVM(
		suite.ctx,
		contracts.ERC20MinterBurnerDecimalsContract.ABI,
		types.ModuleAddress,
		contractAddr,
		false,
		"balanceOf",
		account,
	)
	if err != nil {
		suite.T().Logf("BalanceOf call failed: %v", err)
		return big.NewInt(0)
	}
	if res == nil || len(res.Ret) == 0 {
		return big.NewInt(0)
	}
	balance := new(big.Int)
	balance.SetBytes(res.Ret)
	return balance
}

// DeployContract deploys a standard ERC20 contract with name/symbol/decimals.
func (suite *KeeperTestSuite) DeployContract(name, symbol string, decimals uint8) common.Address {
	ctorArgs, err := contracts.ERC20MinterBurnerDecimalsContract.ABI.Pack("", name, symbol, decimals)
	suite.Require().NoError(err)

	data := append(common.FromHex(contracts.ERC20MinterBurnerDecimalsContract.Bin), ctorArgs...)
	res, err := suite.app.Erc20Keeper.CallEVMWithData(suite.ctx, types.ModuleAddress, nil, data, true)
	suite.Require().NoError(err)
	return res.ContractAddress
}

// DeployContractDirectBalanceManipulation deploys the direct balance manipulation contract.
func (suite *KeeperTestSuite) DeployContractDirectBalanceManipulation(name, symbol string) common.Address {
	ctorArgs, err := contracts.ERC20DirectBalanceManipulationContract.ABI.Pack("", name, symbol)
	suite.Require().NoError(err)

	data := append(common.FromHex(contracts.ERC20DirectBalanceManipulationContract.Bin), ctorArgs...)
	res, err := suite.app.Erc20Keeper.CallEVMWithData(suite.ctx, types.ModuleAddress, nil, data, true)
	suite.Require().NoError(err)
	return res.ContractAddress
}

// DeployContractMaliciousDelayed deploys the malicious delayed contract.
func (suite *KeeperTestSuite) DeployContractMaliciousDelayed(name, symbol string) common.Address {
	ctorArgs, err := contracts.ERC20MaliciousDelayedContract.ABI.Pack("", name, symbol)
	suite.Require().NoError(err)

	data := append(common.FromHex(contracts.ERC20MaliciousDelayedContract.Bin), ctorArgs...)
	res, err := suite.app.Erc20Keeper.CallEVMWithData(suite.ctx, types.ModuleAddress, nil, data, true)
	suite.Require().NoError(err)
	return res.ContractAddress
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
