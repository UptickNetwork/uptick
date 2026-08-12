package keeper_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/mock"
	"github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/evm/x/vm/statedb"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	crypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"

	"github.com/UptickNetwork/uptick/app"
	"github.com/UptickNetwork/uptick/contracts"
	"github.com/UptickNetwork/uptick/x/erc20/types"
)

// init configures the SDK bech32 prefix to match uptick's chain configuration so
// that addresses derived from EVM addresses (sdk.AccAddress(...).String()) use the
// "uptick" prefix expected by the app, instead of the SDK default "cosmos".
func init() {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("uptick", "uptickpub")
	cfg.SetBech32PrefixForValidator("uptickvaloper", "uptickvaloperpub")
	cfg.SetBech32PrefixForConsensusNode("uptickvalcons", "uptickvalconspub")
}

// KeeperTestSuite is the primary test suite for ERC20 keeper integration tests.
// Each test file can define its own helper methods in the same package.
type KeeperTestSuite struct {
	suite.Suite

	ctx sdk.Context
	app *app.Uptick

	address     common.Address
	queryClient types.QueryClient

	// proposerAddr is the consensus address of the validator injected in the
	// genesis; it is reused as the block proposer for EVM coinbase resolution.
	proposerAddr []byte

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

	// cosmos/evm's SetGlobalConfigVariables uses process-global state that can
	// only be configured once per process. The "test" build tag enables
	// ResetTestConfig, which clears that global state so each test can InitChain
	// again. Run the erc20 keeper tests with: go test -tags=test ./x/erc20/keeper/
	evmtypes.NewEVMConfigurator().ResetTestConfig()

	suite.app = app.NewUptick(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		sims.NewAppOptionsWithFlagHome(t.TempDir()),
		nil,
		baseapp.SetChainID("uptick_700-1"),
	)

	// Use NewUncachedContext for the initial (pre-InitChain) context so that the
	// multi-store (app.cms) is available; the check/finalize states only exist
	// after InitChain/Commit in cosmos-sdk 0.53.
	suite.ctx = suite.app.BaseApp.NewUncachedContext(false, tmproto.Header{
		Height:  1,
		Time:    time.Now().UTC(),
		ChainID: "uptick_700-1",
	})

	// Initialize the chain with a genesis state that includes a validator set
	// (required by cosmos/evm modules which decode their params from the
	// KVStore during InitChain and require a non-empty validator set).
	genesisState := suite.app.DefaultGenesis()
	suite.address = common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")

	genWithVal, proposerAddr, err := suite.setupGenesisWithValidator(genesisState)
	suite.Require().NoError(err)
	suite.proposerAddr = proposerAddr
	stateBytes, err := json.MarshalIndent(genWithVal, "", "  ")
	suite.Require().NoError(err)
	_, err = suite.app.InitChain(&abci.RequestInitChain{
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: &tmproto.ConsensusParams{},
		AppStateBytes:   stateBytes,
		ChainId:         "uptick_700-1",
	})
	suite.Require().NoError(err)

	// In cosmos-sdk 0.53 the genesis state is only committed after a
	// FinalizeBlock + Commit round. Finalize and commit so that the test context
	// can observe the genesis accounts (e.g. the erc20 module account) and params.
	// The proposer address must be set so the EVM coinbase can be resolved.
	header := tmproto.Header{
		Height:          1,
		Time:            time.Now().UTC(),
		ChainID:         "uptick_700-1",
		ProposerAddress: proposerAddr,
	}
	finalizeReq := &abci.RequestFinalizeBlock{
		Height:             1,
		Time:               header.Time,
		NextValidatorsHash: []byte{},
		ProposerAddress:    proposerAddr,
	}
	_, err = suite.app.FinalizeBlock(finalizeReq)
	suite.Require().NoError(err)

	// suite.ctx must share the app's finalizeBlockState (deliverState) multi-store
	// so that EVM writes performed through CallEVM / ApplyMessage during a test
	// are persisted when Commit() runs. Commit() then advances the block and
	// re-points suite.ctx at the freshly created finalizeBlockState.
	suite.ctx = suite.app.BaseApp.NewContextLegacy(false, header)

	suite.Commit()

	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, suite.app.Erc20Keeper)
	suite.queryClient = types.NewQueryClient(queryHelper)
}

// setupGenesisWithValidator rewrites the staking, auth, bank and slashing
// module genesis entries of the default genesis to include a single bonded
// validator. This mirrors the setup cosmos/evm uses in its integration test
// network and is required so that InitChain does not fail with an empty
// validator set.
func (suite *KeeperTestSuite) setupGenesisWithValidator(genesisState map[string]json.RawMessage) (map[string]json.RawMessage, []byte, error) {
	cdc := suite.app.AppCodec()
	bondDenom := "auptick"
	bondedAmt := math.NewInt(1e18)

	// 1. Generate a validator consensus key and tm validator.
	privVal := simtestutil.NewPV()
	pubKey, err := privVal.GetPubKey()
	if err != nil {
		return nil, nil, fmt.Errorf("getting pubkey: %w", err)
	}
	tmValidator := cmttypes.NewValidator(pubKey, 1)

	// 2. Build the staking.Validator.
	cosmosPk, err := cryptocodec.FromCmtPubKeyInterface(pubKey)
	if err != nil {
		return nil, nil, fmt.Errorf("converting pubkey: %w", err)
	}
	pkAny, err := codectypes.NewAnyWithValue(cosmosPk)
	if err != nil {
		return nil, nil, fmt.Errorf("packing pubkey: %w", err)
	}
	// The operator address is derived from a dedicated account address so that
	// the self-delegation delegator (an account address) matches the validator's
	// operator account and bech32 prefixes are consistent.
	operatorAcc := sdk.AccAddress(tmValidator.Address)
	opAddr := sdk.ValAddress(operatorAcc.Bytes()).String()
	commission := stakingtypes.NewCommission(math.LegacyNewDecWithPrec(5, 2), math.LegacyNewDecWithPrec(2, 1), math.LegacyNewDecWithPrec(5, 2))
	stakingValidator := stakingtypes.Validator{
		OperatorAddress:   opAddr,
		ConsensusPubkey:   pkAny,
		Jailed:            false,
		Status:            stakingtypes.Bonded,
		Tokens:            bondedAmt,
		DelegatorShares:   math.LegacyOneDec(),
		Description:       stakingtypes.Description{},
		UnbondingHeight:   0,
		UnbondingTime:     time.Unix(0, 0).UTC(),
		Commission:        commission,
		MinSelfDelegation: math.ZeroInt(),
	}

	// 3. Create the staking genesis.
	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenom = bondDenom
	stakingGenesis := stakingtypes.NewGenesisState(
		stakingParams,
		[]stakingtypes.Validator{stakingValidator},
		[]stakingtypes.Delegation{
			stakingtypes.NewDelegation(operatorAcc.String(), opAddr, math.LegacyOneDec()),
		},
	)

	// 4. Extend the *existing* auth genesis (from DefaultGenesis) with the
	//    staking module accounts required for the bonded pool supply invariant.
	//    We must NOT replace the default auth genesis, otherwise module accounts
	//    such as the erc20 module (used as the EVM caller) would be missing and
	//    GetSequence would fail.
	authGen := authtypes.DefaultGenesisState()
	if raw, ok := genesisState[authtypes.ModuleName]; ok {
		cdc.MustUnmarshalJSON(raw, authGen)
	}
	moduleAccs := []string{
		authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String(),
		authtypes.NewModuleAddress(stakingtypes.NotBondedPoolName).String(),
		authtypes.NewModuleAddress("distribution").String(),
	}
	for _, addr := range moduleAccs {
		// Skip if the account already exists in the default genesis.
		exists := false
		for _, accAny := range authGen.Accounts {
			var acc authtypes.ModuleAccountI
			if err := cdc.UnpackAny(accAny, &acc); err == nil && acc.GetAddress().String() == addr {
				exists = true
				break
			}
		}
		if exists {
			continue
		}
		acc := authtypes.NewEmptyModuleAccount(addr)
		accAny, err := codectypes.NewAnyWithValue(acc)
		if err != nil {
			return nil, nil, fmt.Errorf("packing module account %s: %w", addr, err)
		}
		authGen.Accounts = append(authGen.Accounts, accAny)
	}
	// The EOA used as the EVM caller in Convert* flows must exist as a cosmos
	// account (CallEVMWithData reads its sequence) and hold a balance for EVM gas.
	if suite.address != (common.Address{}) {
		ea := authtypes.NewBaseAccount(suite.address.Bytes(), nil, 0, 0)
		eaAny, err := codectypes.NewAnyWithValue(ea)
		if err != nil {
			return nil, nil, fmt.Errorf("packing test EOA account: %w", err)
		}
		authGen.Accounts = append(authGen.Accounts, eaAny)
	}

	// 5. Extend the *existing* bank genesis with the bonded pool funded with the
	//    bonded amount. Appended to whatever supply/balances already exist.
	bankGen := banktypes.DefaultGenesisState()
	if raw, ok := genesisState[banktypes.ModuleName]; ok {
		cdc.MustUnmarshalJSON(raw, bankGen)
	}
	bondedPoolAddr := authtypes.NewModuleAddress(stakingtypes.BondedPoolName)
	bondedCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, bondedAmt))
	balance := banktypes.Balance{
		Address: bondedPoolAddr.String(),
		Coins:   bondedCoins,
	}
	bankGen.Balances = append(bankGen.Balances, balance)
	bankGen.Supply = bankGen.Supply.Add(bondedCoins...)
	if suite.address != (common.Address{}) {
		eaBalance := sdk.NewCoins(sdk.NewCoin("aphoton", math.NewInt(1e18)))
		bankGen.Balances = append(bankGen.Balances, banktypes.Balance{
			Address: sdk.AccAddress(suite.address.Bytes()).String(),
			Coins:   eaBalance,
		})
		bankGen.Supply = bankGen.Supply.Add(eaBalance...)
	}

	// 6. Re-marshal the patched module genesis entries back into the map.
	genesisState[stakingtypes.ModuleName] = cdc.MustMarshalJSON(stakingGenesis)
	genesisState[authtypes.ModuleName] = cdc.MustMarshalJSON(authGen)
	genesisState[banktypes.ModuleName] = cdc.MustMarshalJSON(bankGen)

	return genesisState, tmValidator.Address, nil
}

// StateDB returns the EVM state database for low-level state manipulation.
func (suite *KeeperTestSuite) StateDB() *statedb.StateDB {
	return statedb.New(suite.ctx, suite.app.EvmKeeper, statedb.NewEmptyTxConfig())
}

// Commit advances the chain one block.
// commit commits the current block and advances the block height.
//
// In cosmos-sdk 0.53, app.Commit() persists the app's finalizeBlockState (the
// deliverState). EVM writes performed by CallEVM / ApplyMessage must therefore
// land on the same branched multi-store as finalizeBlockState, otherwise
// contract code and balances are silently dropped on Commit.
//
// To guarantee this, suite.ctx is created via BaseApp.NewContextLegacy, which
// reuses finalizeBlockState.ms (see cosmos-sdk baseapp/test_helpers.go). We
// commit the current finalizeBlockState, then open a fresh FinalizeBlock round
// and point suite.ctx at the new finalizeBlockState so subsequent writes are
// again persisted by the next Commit.
func (suite *KeeperTestSuite) Commit() {
	header := suite.ctx.BlockHeader()
	header.Height++

	// suite.ctx is backed by the app's finalizeBlockState multi-store (see
	// NewContextLegacy). Writes performed through CallEVM/ApplyMessage land there
	// but are only propagated to the root commit-multi-store (and thus actually
	// persisted by app.Commit) when finalizeBlockState.ms.Write() is called.
	// cosmos-sdk triggers this inside FinalizeBlock's workingHash, which already
	// ran for the current block *before* these test-driven writes happened. We
	// therefore flush the pending writes manually so app.Commit() persists them.
	// suite.ctx is backed by the app's finalizeBlockState multi-store (see
	// NewContextLegacy). Writes performed through CallEVM/ApplyMessage land there
	// but are only propagated to the root commit-multi-store (and thus actually
	// persisted by app.Commit) when finalizeBlockState.ms.Write() is called.
	// cosmos-sdk triggers this inside FinalizeBlock's workingHash, which already
	// ran for the current block *before* these test-driven writes happened. We
	// therefore flush the pending writes manually so app.Commit() persists them.
	if cms, ok := suite.ctx.MultiStore().(storetypes.CacheMultiStore); ok {
		cms.Write()
	}

	// Persist whatever was written on the current finalizeBlockState (e.g. a
	// deployed contract or minted balance) during the previous block.
	_, err := suite.app.Commit()
	suite.Require().NoError(err)

	// Open the next block: this rebuilds finalizeBlockState on top of the
	// committed store, so previous writes are now observable.
	_, err = suite.app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height:             header.Height,
		Time:               header.Time,
		NextValidatorsHash: []byte{},
		ProposerAddress:    suite.proposerAddr,
	})
	suite.Require().NoError(err)

	// Point suite.ctx at the new finalizeBlockState so CallEVM/ApplyMessage
	// writes on it are committed by the next Commit call.
	suite.ctx = suite.app.BaseApp.NewContextLegacy(false, header)
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
	// Normalize the zero value so callers comparing with big.NewInt(0) via
	// reflect.DeepEqual don't trip on the internal nat representation difference
	// (nil vs empty slice) of *big.Int zero.
	if balance.Sign() == 0 {
		return big.NewInt(0)
	}
	return balance
}

// GrantERC20Token grants a role (e.g. "MINTER_ROLE") of an ERC20 contract to a grantee.
func (suite *KeeperTestSuite) GrantERC20Token(contractAddr, grantee, from common.Address, role string) {
	// role = keccak256(role name), e.g. MINTER_ROLE = keccak256("MINTER_ROLE")
	roleHash := crypto.Keccak256([]byte(role))
	_, err := suite.app.Erc20Keeper.CallEVM(
		suite.ctx,
		contracts.ERC20MinterBurnerDecimalsContract.ABI,
		from,
		contractAddr,
		true,
		"grantRole",
		common.BytesToHash(roleHash),
		grantee,
	)
	suite.Require().NoError(err)
}

// deployContractWithBin deploys a contract whose bytecode/abi are given, and
// returns the deployed contract address. cosmos/evm's CallEVMWithData no longer
// returns the contract address directly, so we derive it from the deployer
// (the module account) nonce via CREATE semantics.
func (suite *KeeperTestSuite) deployContractWithBin(abi evmtypes.CompiledContract, name, symbol string, extra ...interface{}) common.Address {
	args := append([]interface{}{name, symbol}, extra...)
	// Encode the Solidity constructor arguments (name, symbol, decimals) using
	// the constructor's input ABI. abi.ABI.Pack("") does not resolve the
	// constructor and would silently return an empty/erroring payload, leaving
	// name/symbol unset on-chain.
	ctorArgs, err := abi.ABI.Constructor.Inputs.Pack(args...)
	suite.Require().NoError(err)
	return suite.deployContractWithBinAndArgs(types.ModuleAddress, abi, ctorArgs)
}

// deployContractWithBinAndArgs deploys a contract with already-encoded
// constructor arguments, using from as the EVM sender (and therefore the
// contract owner and the recipient of any constructor-minted tokens).
func (suite *KeeperTestSuite) deployContractWithBinAndArgs(from common.Address, abi evmtypes.CompiledContract, ctorArgs []byte) common.Address {
	nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, from)
	// abi.Bin is a raw-byte HexString ([]byte), not a hex string, so convert
	// directly rather than via common.FromHex.
	data := append([]byte(abi.Bin), ctorArgs...)
	_, err := suite.app.Erc20Keeper.CallEVMWithData(suite.ctx, from, nil, data, true)
	suite.Require().NoError(err)

	addr := crypto.CreateAddress(from, nonce)
	return addr
}

// DeployContract deploys a standard ERC20 contract with name/symbol/decimals.
// It is deployed from the erc20 module account so the module retains ownership
// and can later mint/burn via the minter role.
func (suite *KeeperTestSuite) DeployContract(name, symbol string, decimals uint8) common.Address {
	return suite.deployContractWithBin(contracts.ERC20MinterBurnerDecimalsContract, name, symbol, decimals)
}

// DeployContractDirectBalanceManipulation deploys the malicious ERC20 contract
// whose transfer() silently skims half of every transfer to a thief address,
// which the erc20 keeper's balance-invariance check in ConvertERC20 must detect.
// Solidity constructor signature is constructor(uint256 initialSupply).
// It is deployed from the erc20 module account (like the standard contract) so
// ownership/role-granting stays consistent; the constructor mints no initial
// supply (initialSupply = 0) so the module account's pre-minted test balance is
// not polluted by the deploy-time mint.
func (suite *KeeperTestSuite) DeployContractDirectBalanceManipulation(name, symbol string) common.Address {
	ctorArgs, err := contracts.ERC20DirectBalanceManipulationContract.ABI.Constructor.Inputs.Pack(big.NewInt(0))
	suite.Require().NoError(err)
	return suite.deployContractWithBinAndArgs(types.ModuleAddress, contracts.ERC20DirectBalanceManipulationContract, ctorArgs)
}

// DeployContractMaliciousDelayed deploys the malicious ERC20 contract whose
// transfer() grants a hidden allowance to a thief address (emitting an Approve
// event), which the erc20 keeper's approval-event monitor in ConvertERC20 must detect.
// Solidity constructor signature is constructor(uint256 initialSupply).
// Deployed from the erc20 module account with initialSupply = 0, for the same
// reasons as DeployContractDirectBalanceManipulation.
func (suite *KeeperTestSuite) DeployContractMaliciousDelayed(name, symbol string) common.Address {
	ctorArgs, err := contracts.ERC20MaliciousDelayedContract.ABI.Constructor.Inputs.Pack(big.NewInt(0))
	suite.Require().NoError(err)
	return suite.deployContractWithBinAndArgs(types.ModuleAddress, contracts.ERC20MaliciousDelayedContract, ctorArgs)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
