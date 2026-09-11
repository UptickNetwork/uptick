package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
)

// MainnetChainID is the chain id of the Uptick mainnet, used as the default by
// `init`.
const MainnetChainID = "uptick_117-1"

// FlagGenesisTime defines the genesis time in string format
const FlagGenesisTime = "genesis-time"

// MigrateGenesisCmd returns a command to execute genesis state migration.
//
// The command is deliberately inert. This build registers no genutil migration
// callbacks — the `migrationMap` this used to consult was a permanently empty
// package var that nothing ever wrote to — so the "read genesis → look up
// callback → re-marshal" pipeline below the guard was unreachable code. Keeping
// it around invited the (false) impression that offline migration works.
//
// Today the command does the only thing it could ever have done: refuse and
// point at the in-place x/upgrade handler.
func MigrateGenesisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate TARGET_VERSION GENESIS_FILE",
		Short: "Migrate genesis to a specified target version",
		Long: "Offline genesis migration is disabled in this build. " +
			"Upgrade chain state in place with a governance MsgSoftwareUpgrade plan.",
		Example: fmt.Sprintf(
			// v0.4.x upgrades run in place via the on-chain x/upgrade handler,
			// not offline genesis migration; the command returns an error.
			"%s migrate v3 /path/to/genesis.json --chain-id=uptick_117-1 --genesis-time=2022-04-01T17:00:00Z\n"+
				"# offline migration is disabled in this build; use MsgSoftwareUpgrade instead",
			version.AppName,
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, _ []string) error {
			// v0.4.x registers no offline genesis migrations; state upgrades run
			// in place through the on-chain x/upgrade handler. Fail with clear
			// guidance instead of half-running a pipeline that cannot work.
			return fmt.Errorf(
				"offline genesis migration is not supported: %s registers no legacy genesis migration callbacks. "+
					"Upgrade chain state with the in-app software-upgrade handler (governance MsgSoftwareUpgrade plan) as documented in docs/guides/upgrades; "+
					"do not hand-edit or re-generate genesis files for a chain upgrade",
				version.AppName,
			)
		},
	}

	cmd.Flags().String(FlagGenesisTime, "", "override genesis time")
	cmd.Flags().String(flags.FlagChainID, "", "override genesis chain-id")

	return cmd
}
