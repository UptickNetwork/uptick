package main

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/spf13/cobra"

	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/app"
	v2 "github.com/UptickNetwork/uptick/x/collection/migrations/v2"
	collectiontypes "github.com/UptickNetwork/uptick/x/collection/types"
)

// PrecheckCollectionMigrationCmd returns a command that runs the collection
// module's v1->v2 legacy-store precheck against a node's on-disk state, without
// migrating or writing anything.
//
// WHY THIS IS A COMMAND AND NOT AN UPGRADE-HANDLER STEP
// -----------------------------------------------------
// v2.PrecheckLegacyStore documents that it "must run on the pre-migration store,
// i.e. before the module's RunMigrations call in the upgrade handler". Before
// wiring it into the v0.4.0 handler, that premise was verified and found false:
//
//   - The collection module has reported ConsensusVersion 2 and registered its
//     1->2 migration in RegisterServices since long before v0.3.3
//     (`git show v0.3.3:x/collection/module/module.go`: ConsensusVersion returns
//     2 and RegisterMigration(types.ModuleName, 1, m.Migrate1to2) is present).
//   - Any chain running v0.3.3 therefore already has "collection" == 2 in its
//     x/upgrade version map (set at genesis and/or by the v0.3.3 handler's own
//     RunMigrations; the v0.3.3 handler is app/upgrades/v033/upgrades.go).
//   - module.Manager.RunMigrations is a no-op when fromVersion == toVersion
//     (cosmos-sdk@v0.53.6/types/module/configurator.go:126), so the v0.4.0
//     upgrade does NOT run the collection 1->2 migration at all.
//
// Calling the precheck inside the v0.4.0 handler would therefore be not merely
// unnecessary but a no-op: because the migration never runs at v0.4.0, a
// handler step would be dead weight. The check iterates the legacy,
// "/"-delimited prefixes (KeyDenom("") = 0x04 + "/", KeyNFT("", "") = 0x01 +
// "/"), which live in a different namespace from the bare-prefix keys the
// post-migration nft keeper writes (0x01 + classID, 0x04 + owner,
// 0x05 + classID). On an already-migrated store the legacy keys are gone and
// the scan simply finds nothing, so it is inert rather than harmful: a
// post-migration store, and a store seeded with post-migration nft key shapes,
// both report zero problems.
//
// Exposing the check offline is what remains useful: an operator can run it
// against a pre-upgrade database (a backup or a snapshot) before scheduling an
// upgrade, which is the - and the only - point at which the legacy records it
// looks for can still exist. On a clean store the command prints that nothing
// would abort the migration; on a dirty one it fails with the complete list of
// offending records instead of the single first error v2.Migrate would return.
func PrecheckCollectionMigrationCmd(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "precheck-collection-migration",
		Short: "Dry-run the collection module v1->v2 legacy-store precheck (read-only)",
		Long: "Scan the collection module's legacy (irismod/nft-shaped) store on disk with the " +
			"same iteration and parsing logic the 1->2 migration uses, and report every record " +
			"that would make the migration abort.\n\n" +
			"The command is strictly read-only: it opens the application database at the latest " +
			"committed height, performs no writes, and does not run the migration. Point it at a " +
			"pre-upgrade database copy; once the module has migrated, the legacy records are gone " +
			"and the check has nothing left to look at.\n\n" +
			"On a clean store it prints a confirmation and exits 0. On a dirty store it exits " +
			"non-zero with the FULL problem report (not just the first offending record).",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config

			// The server context is populated by the root command's
			// PersistentPreRunE (server.InterceptConfigsPreRunHandler), the same
			// path the `export` command relies on.
			homeDir, _ := cmd.Flags().GetString(flags.FlagHome)
			config.SetRoot(homeDir)

			db, err := dbm.NewDB(
				"application",
				server.GetAppDBBackend(serverCtx.Viper),
				filepath.Join(config.RootDir, "data"),
			)
			if err != nil {
				return fmt.Errorf("open application database under %s: %w", config.RootDir, err)
			}
			defer func() {
				if cerr := db.Close(); cerr != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: closing application database: %v\n", cerr)
				}
			}()

			// loadLatest=true reads the latest committed version; the app is only
			// used to resolve the collection store key and codec, never to build a
			// block.
			uptickApp := app.NewUptick(serverCtx.Logger, db, nil, true, serverCtx.Viper, nil)
			defer func() { _ = uptickApp.Close() }()

			ctx := uptickApp.NewContext(true)
			return runCollectionPrecheck(
				ctx,
				uptickApp.GetKey(collectiontypes.StoreKey),
				uptickApp.AppCodec(),
				cmd.OutOrStdout(),
			)
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	return cmd
}

// runCollectionPrecheck is the testable core of the command. It scans the
// legacy collection store and reports the outcome:
//
//   - a clean store prints a confirmation and returns nil;
//   - a dirty store returns an error whose message is the FULL report (every
//     offending record, not just the first), which is exactly what
//     v2.PrecheckLegacyStore is designed to produce so an operator sees the
//     whole problem before any state change.
//
// Keeping the store access in the caller lets this be exercised with an
// in-memory store in tests, without building a full application.
func runCollectionPrecheck(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.Codec, out io.Writer) error {
	problems := v2.PrecheckLegacyStore(ctx, storeKey, cdc)
	if len(problems) == 0 {
		fmt.Fprintln(out, "collection legacy-store precheck: clean - no record would abort the v1->v2 migration")
		return nil
	}
	return errors.New(v2.FormatProblems(problems))
}
