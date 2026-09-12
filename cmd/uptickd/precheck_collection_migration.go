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
// WHY A COMMAND, NOT AN UPGRADE-HANDLER STEP
// ------------------------------------------
// v2.PrecheckLegacyStore says it "must run on the pre-migration store". That is
// false here: the collection module already reports ConsensusVersion 2 and
// registers its 1->2 migration at v0.3.3, so every v0.3.3 chain has
// "collection" == 2 in its x/upgrade version map, and RunMigrations skips a
// module when fromVersion == toVersion (cosmos-sdk@v0.53.6/types/module/
// configurator.go:126) -- the v0.4.0 upgrade never runs the 1->2 migration, so a
// handler step would be dead weight.
//
// The check iterates the legacy, "/"-delimited prefixes (KeyDenom("") = 0x04+"/",
// KeyNFT("", "") = 0x01+"/"), a different namespace from the bare-prefix keys the
// post-migration nft keeper writes (0x01+classID, 0x04+classID, 0x05+classID), so
// on an already-migrated store it finds nothing and is inert, not harmful.
//
// Running it offline against a pre-upgrade database (backup or snapshot) -- the
// only point at which the legacy records can still exist -- is what remains
// useful: a clean store prints that nothing would abort the migration, a dirty
// one fails with the complete list rather than v2.Migrate's single first error.
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

			// loadLatest=true reads the latest committed version; the app only
			// resolves the collection store key and codec, it is never used to
			// build a block.
			//
			// The app OWNS db (app.NewUptick -> baseapp.NewBaseApp(db)), and
			// uptickApp.Close() closes that same handle (baseapp.go:1188). Closing
			// db here as well would make leveldb report "leveldb: closed" on the
			// success path -- a warning on a run whose whole point is a clean
			// signal. The app's Close below is therefore the ONLY close.
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
//     offending record, not just the first), which is what v2.PrecheckLegacyStore
//     produces so an operator sees the whole problem before any state change.
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
