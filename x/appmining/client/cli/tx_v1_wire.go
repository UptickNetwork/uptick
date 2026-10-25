package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

const (
	flagChainID         = "v1-chain-id"
	flagProofKeyID      = "proof-key-id"
	flagProofNonce      = "proof-nonce"
	flagProofExpiresAt  = "proof-expires-at"
	flagProofSignature  = "proof-signature-base64"
	flagGasEvidenceRoot = "gas-evidence-root"
	flagReferenceDigest = "reference-digest"
	flagRiskKeyID       = "risk-key-id"
	flagRiskNonce       = "risk-nonce"
	flagRiskExpiresAt   = "risk-expires-at"
	flagRiskSignature   = "risk-signature-base64"
	flagAuditKeyID      = "audit-key-id"
	flagAuditNonce      = "audit-nonce"
	flagAuditExpiresAt  = "audit-expires-at"
	flagAuditSignature  = "audit-signature-base64"
)

func addV1WireTxCommands(cmd *cobra.Command) {
	cmd.AddCommand(
		CmdPublishCandidate(),
		CmdOpenEntitlementChallenge(),
		CmdResolveEntitlementChallenge(),
		CmdApproveEntitlement(),
		CmdFreezeEntitlement(),
		CmdRevokeEntitlement(),
	)
}

func CmdPublishCandidate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish-candidate [pool-id] [entitlement-id] [app-id] [epoch] [amount] [denom] [gas-used] [qualified-tx-count] [pav-id] [evidence-root] [rule-version]",
		Short: "Publish a V1 candidate (publisher wire proof required)",
		Args:  cobra.ExactArgs(11),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			chainID, err := cmd.Flags().GetString(flagChainID)
			if err != nil {
				return err
			}
			if chainID == "" {
				chainID = clientCtx.ChainID
			}
			proof, err := wireProofFromFlags(cmd, flagProofKeyID, flagProofNonce, flagProofExpiresAt, flagProofSignature)
			if err != nil {
				return err
			}
			gasEvidenceRoot, err := cmd.Flags().GetString(flagGasEvidenceRoot)
			if err != nil {
				return err
			}
			msg := &types.MsgPublishCandidate{
				Submitter:        clientCtx.GetFromAddress().String(),
				ChainId:          chainID,
				PoolId:           args[0],
				EntitlementId:    args[1],
				AppId:            args[2],
				Epoch:            args[3],
				Amount:           args[4],
				Denom:            args[5],
				GasUsed:          args[6],
				QualifiedTxCount: args[7],
				PavId:            args[8],
				EvidenceRoot:     args[9],
				GasEvidenceRoot:  gasEvidenceRoot,
				RuleVersion:      args[10],
				Proof:            proof,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	addWireProofFlags(cmd)
	cmd.Flags().String(flagChainID, "", "chain_id bound in wire auth (default: client chain-id)")
	cmd.Flags().String(flagGasEvidenceRoot, "", "optional gas_evidence_root (64 hex)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdOpenEntitlementChallenge() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open-entitlement-challenge [entitlement-id] [evidence-type] [evidence-digest]",
		Short: "Open a public V1 entitlement challenge (any account)",
		Long: `evidence-type is one of:
  TX_MISSING_OR_UNFINALIZED | RECEIPT_QBT_MISMATCH | DUPLICATE_OR_FAMILY_CONFLICT
  CONFIRMED_REFUND_OR_REVERSAL | APP_OR_EMITTER_INELIGIBLE | REPRODUCIBLE_CALCULATION_ERROR
or the integer enum value 1..6.`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			evType, err := parseEvidenceType(args[1])
			if err != nil {
				return err
			}
			ref, err := cmd.Flags().GetString(flagReferenceDigest)
			if err != nil {
				return err
			}
			msg := &types.MsgOpenChallenge{
				Challenger:      clientCtx.GetFromAddress().String(),
				EntitlementId:   args[0],
				EvidenceType:    evType,
				EvidenceDigest:  args[2],
				ReferenceDigest: ref,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(flagReferenceDigest, "", "optional reference_digest (64 hex)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdResolveEntitlementChallenge() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve-entitlement-challenge [pool-id] [challenge-id] [outcome] [rationale-digest]",
		Short: "Resolve an OPEN V1 entitlement challenge (reviewer wire proof)",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			chainID, err := cmd.Flags().GetString(flagChainID)
			if err != nil {
				return err
			}
			if chainID == "" {
				chainID = clientCtx.ChainID
			}
			proof, err := wireProofFromFlags(cmd, flagProofKeyID, flagProofNonce, flagProofExpiresAt, flagProofSignature)
			if err != nil {
				return err
			}
			msg := &types.MsgResolveEntitlementChallenge{
				Submitter:       clientCtx.GetFromAddress().String(),
				ChainId:         chainID,
				PoolId:          args[0],
				ChallengeId:     args[1],
				Outcome:         strings.ToUpper(strings.TrimSpace(args[2])),
				RationaleDigest: args[3],
				Proof:           proof,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	addWireProofFlags(cmd)
	cmd.Flags().String(flagChainID, "", "chain_id bound in wire auth (default: client chain-id)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdApproveEntitlement() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve-entitlement [pool-id] [entitlement-id] [challenge-resolution-hash]",
		Short: "Approve a V1 entitlement (approver wire proof; locks available→reserved)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			chainID, err := cmd.Flags().GetString(flagChainID)
			if err != nil {
				return err
			}
			if chainID == "" {
				chainID = clientCtx.ChainID
			}
			proof, err := wireProofFromFlags(cmd, flagProofKeyID, flagProofNonce, flagProofExpiresAt, flagProofSignature)
			if err != nil {
				return err
			}
			msg := &types.MsgApproveEntitlement{
				Submitter:               clientCtx.GetFromAddress().String(),
				ChainId:                 chainID,
				PoolId:                  args[0],
				EntitlementId:           args[1],
				ChallengeResolutionHash: args[2],
				Proof:                   proof,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	addWireProofFlags(cmd)
	cmd.Flags().String(flagChainID, "", "chain_id bound in wire auth (default: client chain-id)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdFreezeEntitlement() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "freeze-entitlement [pool-id] [entitlement-id] [evidence-type] [evidence-digest] [freeze-until]",
		Short: "Freeze a CLAIMABLE V1 entitlement (risk wire proof; freeze-until unix seconds)",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			chainID, err := cmd.Flags().GetString(flagChainID)
			if err != nil {
				return err
			}
			if chainID == "" {
				chainID = clientCtx.ChainID
			}
			evType, err := parseEvidenceType(args[2])
			if err != nil {
				return err
			}
			proof, err := wireProofFromFlags(cmd, flagProofKeyID, flagProofNonce, flagProofExpiresAt, flagProofSignature)
			if err != nil {
				return err
			}
			msg := &types.MsgFreezeEntitlement{
				Submitter:      clientCtx.GetFromAddress().String(),
				ChainId:        chainID,
				PoolId:         args[0],
				EntitlementId:  args[1],
				EvidenceType:   evType,
				EvidenceDigest: args[3],
				FreezeUntil:    args[4],
				Proof:          proof,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	addWireProofFlags(cmd)
	cmd.Flags().String(flagChainID, "", "chain_id bound in wire auth (default: client chain-id)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdRevokeEntitlement() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke-entitlement [pool-id] [entitlement-id] [evidence-type] [evidence-digest] [rationale-digest]",
		Short: "Revoke a frozen V1 entitlement (independent risk + audit wire proofs)",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			chainID, err := cmd.Flags().GetString(flagChainID)
			if err != nil {
				return err
			}
			if chainID == "" {
				chainID = clientCtx.ChainID
			}
			evType, err := parseEvidenceType(args[2])
			if err != nil {
				return err
			}
			risk, err := wireProofFromFlags(cmd, flagRiskKeyID, flagRiskNonce, flagRiskExpiresAt, flagRiskSignature)
			if err != nil {
				return fmt.Errorf("risk proof: %w", err)
			}
			audit, err := wireProofFromFlags(cmd, flagAuditKeyID, flagAuditNonce, flagAuditExpiresAt, flagAuditSignature)
			if err != nil {
				return fmt.Errorf("audit proof: %w", err)
			}
			msg := &types.MsgRevokeEntitlement{
				Submitter:       clientCtx.GetFromAddress().String(),
				ChainId:         chainID,
				PoolId:          args[0],
				EntitlementId:   args[1],
				EvidenceType:    evType,
				EvidenceDigest:  args[3],
				RationaleDigest: args[4],
				RiskProof:       risk,
				AuditProof:      audit,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(flagChainID, "", "chain_id bound in wire auth (default: client chain-id)")
	cmd.Flags().String(flagRiskKeyID, "", "risk WireProof.key_id")
	cmd.Flags().String(flagRiskNonce, "", "risk WireProof.nonce")
	cmd.Flags().String(flagRiskExpiresAt, "", "risk WireProof.expires_at")
	cmd.Flags().String(flagRiskSignature, "", "risk WireProof.signature_base64")
	cmd.Flags().String(flagAuditKeyID, "", "audit WireProof.key_id")
	cmd.Flags().String(flagAuditNonce, "", "audit WireProof.nonce")
	cmd.Flags().String(flagAuditExpiresAt, "", "audit WireProof.expires_at")
	cmd.Flags().String(flagAuditSignature, "", "audit WireProof.signature_base64")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func addWireProofFlags(cmd *cobra.Command) {
	cmd.Flags().String(flagProofKeyID, "", "WireProof.key_id")
	cmd.Flags().String(flagProofNonce, "", "WireProof.nonce")
	cmd.Flags().String(flagProofExpiresAt, "", "WireProof.expires_at (unix seconds decimal)")
	cmd.Flags().String(flagProofSignature, "", "WireProof.signature_base64")
}

func wireProofFromFlags(cmd *cobra.Command, keyIDFlag, nonceFlag, expiresFlag, sigFlag string) (types.WireProof, error) {
	keyID, err := cmd.Flags().GetString(keyIDFlag)
	if err != nil {
		return types.WireProof{}, err
	}
	nonce, err := cmd.Flags().GetString(nonceFlag)
	if err != nil {
		return types.WireProof{}, err
	}
	expires, err := cmd.Flags().GetString(expiresFlag)
	if err != nil {
		return types.WireProof{}, err
	}
	sig, err := cmd.Flags().GetString(sigFlag)
	if err != nil {
		return types.WireProof{}, err
	}
	if keyID == "" || nonce == "" || expires == "" || sig == "" {
		return types.WireProof{}, fmt.Errorf("wire proof flags %s/%s/%s/%s are required", keyIDFlag, nonceFlag, expiresFlag, sigFlag)
	}
	return types.WireProof{
		KeyId:           keyID,
		Nonce:           nonce,
		ExpiresAt:       expires,
		SignatureBase64: sig,
	}, nil
}

func parseEvidenceType(s string) (types.EvidenceType, error) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "1", "TX_MISSING_OR_UNFINALIZED":
		return types.EvidenceTypeTxMissingOrUnfinalized, nil
	case "2", "RECEIPT_QBT_MISMATCH":
		return types.EvidenceTypeReceiptQbtMismatch, nil
	case "3", "DUPLICATE_OR_FAMILY_CONFLICT":
		return types.EvidenceTypeDuplicateOrFamilyConflict, nil
	case "4", "CONFIRMED_REFUND_OR_REVERSAL":
		return types.EvidenceTypeConfirmedRefundOrReversal, nil
	case "5", "APP_OR_EMITTER_INELIGIBLE":
		return types.EvidenceTypeAppOrEmitterIneligible, nil
	case "6", "REPRODUCIBLE_CALCULATION_ERROR":
		return types.EvidenceTypeReproducibleCalculationError, nil
	default:
		return types.EvidenceTypeUnspecified, fmt.Errorf("evidence-type must be 1..6 or a named EvidenceType")
	}
}
