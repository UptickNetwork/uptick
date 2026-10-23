package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func addV1QueryCommands(cmd *cobra.Command) {
	cmd.AddCommand(
		CmdQueryEntitlementPool(),
		CmdQueryCandidate(),
		CmdQueryEntitlementChallenge(),
		CmdQueryEntitlement(),
		CmdQueryEntitlementsByApp(),
		CmdQueryRoles(),
	)
}

func CmdQueryEntitlementPool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entitlement-pool [pool-id]",
		Short: "Query a V1 entitlement pool (ledger + module balance + enable gate)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Pool(cmd.Context(), &types.QueryPoolRequest{PoolId: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryCandidate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "candidate [entitlement-id-hex]",
		Short: "Query a V1 published candidate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Candidate(cmd.Context(), &types.QueryCandidateRequest{EntitlementId: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryEntitlementChallenge() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entitlement-challenge [challenge-id-hex]",
		Short: "Query a V1 entitlement challenge",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).EntitlementChallenge(cmd.Context(), &types.QueryEntitlementChallengeRequest{ChallengeId: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryEntitlement() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entitlement [entitlement-id-hex]",
		Short: "Query a V1 entitlement",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Entitlement(cmd.Context(), &types.QueryEntitlementRequest{EntitlementId: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryEntitlementsByApp() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entitlements-by-app [app-id-hex]",
		Short: "Query V1 entitlements by app_id (64-hex)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).EntitlementsByApp(cmd.Context(), &types.QueryEntitlementsByAppRequest{
				AppId:      args[0],
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "entitlements")
	return cmd
}

func CmdQueryRoles() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "roles [role|all]",
		Short: "Query V1 role keys (role=publisher|approver|reviewer|risk|audit|all)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			var role types.EntitlementRole
			if args[0] != "all" {
				role, err = parseEntitlementRole(args[0])
				if err != nil {
					return err
				}
			}
			res, err := types.NewQueryClient(clientCtx).Roles(cmd.Context(), &types.QueryRolesRequest{Role: role})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
