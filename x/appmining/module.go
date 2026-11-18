package appmining

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	abci "github.com/cometbft/cometbft/abci/types"

	appmodule "cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/UptickNetwork/uptick/x/appmining/client/cli"
	"github.com/UptickNetwork/uptick/x/appmining/keeper"
	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// ConsensusVersion 2 adds the V1 challenge candidate index and timeout buckets
// and the timeout sweep (migration 1→2 backfills the indexes; without it a
// pre-existing challenge would be invisible to per-candidate reads).
const ConsensusVersion = 2

var (
	_ module.AppModule        = AppModule{}
	_ module.AppModuleBasic   = AppModuleBasic{}
	_ appmodule.HasEndBlocker = AppModule{}
)

// AppModuleBasic defines basic module elements.
type AppModuleBasic struct{}

func (AppModuleBasic) Name() string { return types.ModuleName }

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

func (AppModuleBasic) ConsensusVersion() uint64 { return ConsensusVersion }

func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis: %w", types.ModuleName, err)
	}
	return types.ValidateGenesis(gs)
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

func (AppModuleBasic) GetTxCmd() *cobra.Command    { return cli.NewTxCmd() }
func (AppModuleBasic) GetQueryCmd() *cobra.Command { return cli.GetQueryCmd() }

// AppModule implements the AppModule interface.
type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

func NewAppModule(k keeper.Keeper) AppModule {
	return AppModule{AppModuleBasic: AppModuleBasic{}, keeper: k}
}

func (am AppModule) Name() string { return types.ModuleName }

func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {
	keeper.RegisterInvariants(ir, am.keeper)
}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)
	if err := cfg.RegisterMigration(types.ModuleName, 1, keeper.NewMigrator(am.keeper).Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 1 to 2: %v", types.ModuleName, err))
	}
}

// EndBlock drives the timeout settlement for challenges whose sealed resolution
// window elapsed without a reviewer decision. It is the automatic path behind
// MsgExpireEntitlementChallenge: without it the EXPIRED state (and the deposit
// refund that goes with it) would only happen if somebody chose to pay gas for
// it, which is not guaranteed for zero-deposit ("open access") challenges.
//
// It never returns an error: an EndBlocker error halts the chain, and a timeout
// settlement is not worth a liveness failure. Every problem is logged and
// retried on the next block.
func (am AppModule) EndBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	am.keeper.SweepExpiredChallenges(sdkCtx)
	return nil
}

func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var gs types.GenesisState
	cdc.MustUnmarshalJSON(data, &gs)
	am.keeper.InitGenesis(ctx, gs)
	return nil
}

func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(am.keeper.ExportGenesis(ctx))
}

func (am AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

func (am AppModule) IsOnePerModuleType() {}
func (am AppModule) IsAppModule()        {}
