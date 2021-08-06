package commands

import (
	"context"
	"fmt"

	cfg "github.com/providenetwork/tendermint/config"
	"github.com/providenetwork/tendermint/scripts/keymigrate"
	"github.com/spf13/cobra"
)

func MakeKeyMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key-migrate",
		Short: "Run Database key migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			contexts := []string{
				// this is ordered to put the
				// (presumably) biggest/most important
				// subsets first.
				"blockstore",
				"state",
				"peerstore",
				"tx_index",
				"evidence",
				"light",
			}

			for idx, dbctx := range contexts {
				logger.Info("beginning a key migration",
					"dbctx", dbctx,
					"num", idx+1,
					"total", len(contexts),
				)

				db, err := cfg.DefaultDBProvider(&cfg.DBContext{
					ID:     dbctx,
					Config: config,
				})

				if err != nil {
					return fmt.Errorf("constructing database handle: %w", err)
				}

				if err = keymigrate.Migrate(ctx, db); err != nil {
					return fmt.Errorf("running migration for context %q: %w",
						dbctx, err)
				}
			}

			logger.Info("completed database migration successfully")

			return nil
		},
	}

	// allow database info to be overridden via cli
	addDBFlags(cmd)

	return cmd
}
