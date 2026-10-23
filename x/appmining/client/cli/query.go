package cli

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// GetQueryCmd returns the query commands for this module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the appmining module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		CmdQueryParams(),
		CmdQueryApplication(),
		CmdQueryApplications(),
		CmdQueryEpochBudget(),
		CmdQueryAttestation(),
		CmdQueryAttestations(),
		CmdQueryChallenge(),
		CmdQuerySettlement(),
		CmdQuerySettlements(),
		CmdQueryOpenChallenges(),
		CmdQueryGasCredit(),
		CmdQueryGasCreditConsumptions(),
	)
	addV1QueryCommands(cmd)
	return cmd
}

func CmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query module params",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryApplication() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "application [app-id-hex]",
		Short: "Query an application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			appID, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("app-id-hex: %w", err)
			}
			res, err := types.NewQueryClient(clientCtx).Application(cmd.Context(), &types.QueryApplicationRequest{AppId: appID})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryEpochBudget() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "epoch-budget [epoch]",
		Short: "Query an epoch budget",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			epoch, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).EpochBudget(cmd.Context(), &types.QueryEpochBudgetRequest{Epoch: epoch})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryAttestation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attestation [app-id-hex] [epoch]",
		Short: "Query an attestation",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			appID, err := hex.DecodeString(args[0])
			if err != nil {
				return err
			}
			epoch, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Attestation(cmd.Context(), &types.QueryAttestationRequest{AppId: appID, Epoch: epoch})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryChallenge() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "challenge [challenge-id]",
		Short: "Query a challenge by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("challenge-id: %w", err)
			}
			res, err := types.NewQueryClient(clientCtx).Challenge(cmd.Context(), &types.QueryChallengeRequest{ChallengeId: id})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQuerySettlement() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "settlement [app-id-hex] [epoch]",
		Short: "Query a settlement",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			appID, err := hex.DecodeString(args[0])
			if err != nil {
				return err
			}
			epoch, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Settlement(cmd.Context(), &types.QuerySettlementRequest{AppId: appID, Epoch: epoch})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryApplications() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "applications",
		Short: "List applications (paginated)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Applications(cmd.Context(), &types.QueryApplicationsRequest{Pagination: pageReq})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "applications")
	return cmd
}

func CmdQueryAttestations() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attestations",
		Short: "List attestations (paginated)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Attestations(cmd.Context(), &types.QueryAttestationsRequest{Pagination: pageReq})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "attestations")
	return cmd
}

func CmdQuerySettlements() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "settlements",
		Short: "List settlements (paginated)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Settlements(cmd.Context(), &types.QuerySettlementsRequest{Pagination: pageReq})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "settlements")
	return cmd
}

func CmdQueryOpenChallenges() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open-challenges",
		Short: "List open challenges (paginated)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).OpenChallenges(cmd.Context(), &types.QueryOpenChallengesRequest{Pagination: pageReq})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "open-challenges")
	return cmd
}

func CmdQueryGasCredit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gas-credit [app-id-hex]",
		Short: "Query gas credit account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			appID, err := hex.DecodeString(args[0])
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).GasCredit(cmd.Context(), &types.QueryGasCreditRequest{AppId: appID})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryGasCreditConsumptions() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gas-credit-consumptions [app-id-hex]",
		Short: "List gas credit consumptions for an app (paginated)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			appID, err := hex.DecodeString(args[0])
			if err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).GasCreditConsumptions(cmd.Context(), &types.QueryGasCreditConsumptionsRequest{
				AppId:      appID,
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "gas-credit-consumptions")
	return cmd
}
