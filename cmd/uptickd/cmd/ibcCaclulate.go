package cmd

import (
	"github.com/spf13/cobra"

	"github.com/UptickNetwork/uptick/x/erc20/types"
)

func AddIbcCaclulateCommand(debug *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ibc-denom [port] [channel] [denom]",
		Short: "Generate ibc denom name",
		Long:  `According to the target channel, port and denom provided by the user, generate the denom name after the ibc cross-chain transfer`,
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			denom, err := types.IBCDenom(args[0], args[1], args[2])
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
