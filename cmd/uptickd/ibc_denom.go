package main

import (
	"fmt"

	"github.com/spf13/cobra"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

// IBCDenom derives the ICS-20 voucher denom that the receiving chain records
// for denom transferred over (port, channel).
//
// The path is parsed rather than assembled: when denom is itself an
// already-prefixed denom from the counterparty, only the parser produces the
// right hop list and base, and the hash the chain stores has to match it
// exactly. Two ibc-go helpers that used to do this are deprecated in v10 --
// GetDenomPrefix (replaced by NewHop) and ParseDenomTrace (now a literal alias
// of ExtractDenomFromPath) -- so this uses the replacements directly.
func IBCDenom(port, channel, denom string) (string, error) {
	if denom == "" {
		return "", fmt.Errorf("denom cannot be empty")
	}

	// Validate the hop before parsing. Without this, a port/channel pair that
	// is not in the "port/channel-<n>" shape is not recognized as a hop at all,
	// the parse then yields no trace, and the caller was told the input "is a
	// native denom" -- a message that was both wrong and unactionable, because
	// the input was never native, it was just unparseable.
	hop := transfertypes.NewHop(port, channel)
	if err := hop.Validate(); err != nil {
		return "", fmt.Errorf("invalid port/channel %q/%q: %w", port, channel, err)
	}

	fullPath := hop.String() + "/" + denom
	denomTrace := transfertypes.ExtractDenomFromPath(fullPath)
	if denomTrace.IsNative() {
		return "", fmt.Errorf(
			"no ibc trace could be parsed out of %q; the channel identifier must look like channel-<n>",
			fullPath)
	}

	return denomTrace.IBCDenom(), nil
}

func AddIBCDenomCommand(debug *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ibc-denom [port] [channel] [denom]",
		Short: "Generate ibc denom name",
		Long: `According to the target channel, port and denom provided by the user, generate the denom name after the ibc cross-chain transfer.

This is an offline calculator: it derives the voucher denom from the identifiers
alone and does not check that the channel exists on this chain.`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			denom, err := IBCDenom(args[0], args[1], args[2])
			if err != nil {
				return err
			}
			cmd.Printf("IBC Denom: %s\nPort: %s\nChannel: %s\nOriginal Denom: %s\n",
				denom, args[0], args[1], args[2])
			return nil
		},
	}
	debug.AddCommand(cmd)
	return debug
}
