package rpc

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"

	"github.com/UptickNetwork/uptick/app/keepers"
)

var overrideModules = map[string]overrideHandler{
	authtypes.ModuleName:    overrideAuthServices,
	vestingtypes.ModuleName: overrideVestingServices,
}

type overrideHandler func(cdc codec.Codec, configurator module.Configurator, appKeepers keepers.AppKeepers)

// RegisterService allows a module to register services.
func RegisterService(cdc codec.Codec, mod module.AppModule, configurator module.Configurator, appKeepers keepers.AppKeepers) {
	handler, has := overrideModules[mod.Name()]
	if has {
		handler(cdc, configurator, appKeepers)
		return
	}

	m, ok := mod.(module.HasServices)
	if ok {
		m.RegisterServices(configurator)
	}
}

func overrideVestingServices(cdc codec.Codec, configurator module.Configurator, appKeepers keepers.AppKeepers) {
	// vesting module uses grpc.ServiceRegistrar instead of module.Configurator
	// so we need to register it manually
	vestingtypes.RegisterMsgServer(configurator.MsgServer(), vesting.NewMsgServerImpl(appKeepers.AccountKeeper, appKeepers.BankKeeper))
}
