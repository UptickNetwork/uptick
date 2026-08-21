package main

import (
	"fmt"

	"github.com/spf13/cobra"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

func IBCDenom(port, channel, denom string) (string, error) {
	sourcePrefix := transfertypes.GetDenomPrefix(port, channel)
	prefixedDenom := sourcePrefix + denom
	denomTrace := transfertypes.ParseDenomTrace(prefixedDenom)
	if denomTrace.IsNative() {
		return "", fmt.Errorf("'%s' is a native denom", prefixedDenom)
	}
	return denomTrace.IBCDenom(), nil
}

func AddIbcCaclulateCommand(debug *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ibc-denom [port] [channel] [denom]",
		Short: "Generate ibc denom name",
		Long:  `According to the target channel, port and denom provided by the user, generate the denom name after the ibc cross-chain transfer`,
		Args:  cobra.ExactArgs(3),
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
