package module

import (
	"testing"

	"github.com/UptickNetwork/uptick/x/collection/types"
	"github.com/stretchr/testify/require"
)

func TestAppModuleBasic_Name(t *testing.T) {
	var basic AppModuleBasic
	require.Equal(t, types.ModuleName, basic.Name())
}

func TestAppModule_NameAndConsensusVersion(t *testing.T) {
	var am AppModule
	require.Equal(t, types.ModuleName, am.Name())
	require.Equal(t, uint64(2), am.ConsensusVersion())
}
