package simulation

import (
	"encoding/json"
	"math/rand"
	"testing"

	appparams "github.com/UptickNetwork/uptick/app/params"
	collectiontypes "github.com/UptickNetwork/uptick/x/collection/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/stretchr/testify/require"
)

func TestRandomizedGenStateSetsModuleState(t *testing.T) {
	encCfg := appparams.MakeEncodingConfig()
	simState := &module.SimulationState{
		AppParams: make(simtypes.AppParams),
		Cdc:       encCfg.Codec,
		Rand:      rand.New(rand.NewSource(1)),
		GenState:  make(map[string]json.RawMessage),
		Accounts:  nil,
	}

	RandomizedGenState(simState)

	bz, ok := simState.GenState[collectiontypes.ModuleName]
	require.True(t, ok)
	require.NotEmpty(t, bz)

	var gs collectiontypes.GenesisState
	encCfg.Codec.MustUnmarshalJSON(bz, &gs)
	require.Len(t, gs.Collections, 2)
}
