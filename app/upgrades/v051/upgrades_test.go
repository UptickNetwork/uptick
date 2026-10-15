package v051

import (
	"testing"

	"github.com/stretchr/testify/require"

	appminingtypes "github.com/UptickNetwork/uptick/x/appmining/types"
)

func TestUpgradeMetadata(t *testing.T) {
	t.Parallel()
	require.Equal(t, "v0.5.1", Upgrade.UpgradeName)
	require.NotNil(t, Upgrade.StoreUpgrades)
	require.Equal(t, []string{appminingtypes.StoreKey}, Upgrade.StoreUpgrades.Added)
	require.Empty(t, Upgrade.StoreUpgrades.Deleted)
	require.NotNil(t, Upgrade.UpgradeHandlerConstructor)
}
