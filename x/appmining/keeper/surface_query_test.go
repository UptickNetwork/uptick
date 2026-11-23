package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/UptickNetwork/uptick/x/appmining/keeper"
	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// The gRPC query surface is part of the module contract: the UGE service and
// any explorer read state through it, and an endpoint that silently returns an
// empty page (or panics on a nil request) is a production incident even though
// no test wrote a transaction. Before this file eleven endpoints were at 0%
// statement coverage; these tests pin the success, not-found and nil-request
// contract of each.

func TestSurface_QueryParams(t *testing.T) {
	e := setupTest(t)
	resp, err := e.k.Params(e.ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, e.k.GetParams(e.ctx).Authority, resp.Params.Authority)
}

func TestSurface_QueryApplication(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)

	resp, err := e.k.Application(e.ctx, &types.QueryApplicationRequest{AppId: appID})
	require.NoError(t, err)
	require.Equal(t, e.owner.String(), resp.Application.Owner)

	_, err = e.k.Application(e.ctx, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = e.k.Application(e.ctx, &types.QueryApplicationRequest{AppId: hash32(0x99)})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestSurface_QueryApplications(t *testing.T) {
	e := setupTest(t)
	active := registerApp(t, e)
	suspended := hash32(0xBB)
	_, err := e.server.RegisterApplication(e.ctx, &types.MsgRegisterApplication{
		Authority: e.auth.String(), AppId: suspended, Owner: e.owner.String(), AttestationSubmitter: e.sub.String(),
	})
	require.NoError(t, err)
	_, err = e.server.SetApplicationStatus(e.ctx, &types.MsgSetApplicationStatus{
		Authority: e.auth.String(), AppId: suspended, Status: types.ApplicationStatusSuspended,
	})
	require.NoError(t, err)

	all, err := e.k.Applications(e.ctx, &types.QueryApplicationsRequest{})
	require.NoError(t, err)
	require.Len(t, all.Applications, 2)

	onlySuspended, err := e.k.Applications(e.ctx, &types.QueryApplicationsRequest{Status: types.ApplicationStatusSuspended})
	require.NoError(t, err)
	require.Len(t, onlySuspended.Applications, 1)
	require.Equal(t, suspended, onlySuspended.Applications[0].AppId)

	onlyActive, err := e.k.Applications(e.ctx, &types.QueryApplicationsRequest{Status: types.ApplicationStatusActive})
	require.NoError(t, err)
	require.Len(t, onlyActive.Applications, 1)
	require.Equal(t, active, onlyActive.Applications[0].AppId)

	// Pagination is honoured: one entry per page, two pages, no overlap.
	page1, err := e.k.Applications(e.ctx, &types.QueryApplicationsRequest{
		Pagination: &query.PageRequest{Limit: 1},
	})
	require.NoError(t, err)
	require.Len(t, page1.Applications, 1)
	require.NotEmpty(t, page1.Pagination.NextKey)
	page2, err := e.k.Applications(e.ctx, &types.QueryApplicationsRequest{
		Pagination: &query.PageRequest{Key: page1.Pagination.NextKey, Limit: 1},
	})
	require.NoError(t, err)
	require.Len(t, page2.Applications, 1)
	require.NotEqual(t, page1.Applications[0].AppId, page2.Applications[0].AppId)

	_, err = e.k.Applications(e.ctx, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSurface_QueryEpochBudget(t *testing.T) {
	e := setupTest(t)
	setBudget(t, e, 1, "1000")

	resp, err := e.k.EpochBudget(e.ctx, &types.QueryEpochBudgetRequest{Epoch: 1})
	require.NoError(t, err)
	require.Equal(t, "1000", resp.Budget.Limit)

	_, err = e.k.EpochBudget(e.ctx, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = e.k.EpochBudget(e.ctx, &types.QueryEpochBudgetRequest{Epoch: 99})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestSurface_QueryAttestationAndList(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	setBudget(t, e, 1, "1000")
	_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "100"),
	})
	require.NoError(t, err)

	resp, err := e.k.Attestation(e.ctx, &types.QueryAttestationRequest{AppId: appID, Epoch: 1})
	require.NoError(t, err)
	require.Equal(t, types.AttestationStatusChallengeWindow, resp.Attestation.Status)

	_, err = e.k.Attestation(e.ctx, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = e.k.Attestation(e.ctx, &types.QueryAttestationRequest{AppId: appID, Epoch: 2})
	require.Equal(t, codes.NotFound, status.Code(err))

	list, err := e.k.Attestations(e.ctx, &types.QueryAttestationsRequest{})
	require.NoError(t, err)
	require.Len(t, list.Attestations, 1)

	filtered, err := e.k.Attestations(e.ctx, &types.QueryAttestationsRequest{
		Status: types.AttestationStatusChallengeWindow,
	})
	require.NoError(t, err)
	require.Len(t, filtered.Attestations, 1)

	empty, err := e.k.Attestations(e.ctx, &types.QueryAttestationsRequest{
		Status: types.AttestationStatusFinalized,
	})
	require.NoError(t, err)
	require.Empty(t, empty.Attestations)

	_, err = e.k.Attestations(e.ctx, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

// openLegacyChallenge publishes an attestation and opens a challenge on it, so
// the challenge queries have something to read.
func openLegacyChallenge(t *testing.T, e testEnv, appID []byte, epoch uint64) uint64 {
	t.Helper()
	setBudget(t, e, epoch, "1000")
	_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: e.sub.String(), Attestation: sampleAttestation(appID, epoch, "100"),
	})
	require.NoError(t, err)
	bond, err := types.ParseAmount(types.DefaultChallengeBondAmount)
	require.NoError(t, err)
	e.bank.fundAccount(e.owner, types.DefaultChallengeBondDenom, bond)
	resp, err := e.server.ChallengeAttestation(e.ctx, &types.MsgChallengeAttestation{
		Challenger: e.owner.String(), AppId: appID, Epoch: epoch, EvidenceHash: hash32(0x09),
	})
	require.NoError(t, err)
	return resp.ChallengeId
}

func TestSurface_QueryChallenge(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	id := openLegacyChallenge(t, e, appID, 1)

	resp, err := e.k.Challenge(e.ctx, &types.QueryChallengeRequest{ChallengeId: id})
	require.NoError(t, err)
	require.Equal(t, types.ChallengeStatusOpen, resp.Challenge.Status)

	_, err = e.k.Challenge(e.ctx, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = e.k.Challenge(e.ctx, &types.QueryChallengeRequest{ChallengeId: 999})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestSurface_QueryOpenChallenges(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	id := openLegacyChallenge(t, e, appID, 1)

	byEpoch, err := e.k.OpenChallenges(e.ctx, &types.QueryOpenChallengesRequest{AppId: appID, Epoch: 1})
	require.NoError(t, err)
	require.Len(t, byEpoch.Challenges, 1)
	require.Equal(t, id, byEpoch.Challenges[0].ChallengeId)

	byApp, err := e.k.OpenChallenges(e.ctx, &types.QueryOpenChallengesRequest{AppId: appID})
	require.NoError(t, err)
	require.Len(t, byApp.Challenges, 1)

	all, err := e.k.OpenChallenges(e.ctx, &types.QueryOpenChallengesRequest{})
	require.NoError(t, err)
	require.Len(t, all.Challenges, 1)

	// Resolved challenges leave the open index.
	_, err = e.server.ResolveChallenge(e.ctx, &types.MsgResolveChallenge{
		Authority: e.auth.String(), ChallengeId: id, Uphold: false, ResolutionHash: hash32(0x03),
	})
	require.NoError(t, err)
	all, err = e.k.OpenChallenges(e.ctx, &types.QueryOpenChallengesRequest{})
	require.NoError(t, err)
	require.Empty(t, all.Challenges)

	_, err = e.k.OpenChallenges(e.ctx, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSurface_QueryGasCredit(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)

	// Unknown app reads as a zeroed account rather than NotFound: the PostHandler
	// calls this on every sponsored tx and must not fail on a fresh app.
	empty, err := e.k.GasCredit(e.ctx, &types.QueryGasCreditRequest{AppId: appID})
	require.NoError(t, err)
	require.Equal(t, "0", empty.Account.Balance)

	acct := e.k.GetOrCreateGasCredit(e.ctx, appID)
	acct.Balance = "500"
	acct.TotalIssued = "500"
	e.k.SetGasCredit(e.ctx, acct)
	resp, err := e.k.GasCredit(e.ctx, &types.QueryGasCreditRequest{AppId: appID})
	require.NoError(t, err)
	require.Equal(t, "500", resp.Account.Balance)

	_, err = e.k.GasCredit(e.ctx, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSurface_QueryGasCreditConsumptions(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)

	empty, err := e.k.GasCreditConsumptions(e.ctx, &types.QueryGasCreditConsumptionsRequest{AppId: appID})
	require.NoError(t, err)
	require.Empty(t, empty.Consumptions)

	e.k.SetGasCreditConsumption(e.ctx, types.GasCreditConsumption{
		AppId: appID, GasPayer: e.payer.String(), SourceTxHash: hash32(0x0a),
		Amount: "10", ConsumedHeight: 1,
	})
	list, err := e.k.GasCreditConsumptions(e.ctx, &types.QueryGasCreditConsumptionsRequest{AppId: appID})
	require.NoError(t, err)
	require.Len(t, list.Consumptions, 1)
	require.Equal(t, "10", list.Consumptions[0].Amount)

	// Consumptions are namespaced per app.
	other, err := e.k.GasCreditConsumptions(e.ctx, &types.QueryGasCreditConsumptionsRequest{AppId: hash32(0xBB)})
	require.NoError(t, err)
	require.Empty(t, other.Consumptions)

	_, err = e.k.GasCreditConsumptions(e.ctx, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSurface_QueryEntitlementChallenge(t *testing.T) {
	e, _, _, _, c := setupP2(t)
	resp, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
		Challenger: e.owner.String(), EntitlementId: c.EntitlementID,
		EvidenceType: types.EvidenceTypeConfirmedRefundOrReversal, EvidenceDigest: digestOf(0x0b),
	})
	require.NoError(t, err)

	got, err := e.k.EntitlementChallenge(e.ctx, &types.QueryEntitlementChallengeRequest{ChallengeId: resp.ChallengeId})
	require.NoError(t, err)
	require.Equal(t, types.EntitlementChallengeStatusOpen, got.Challenge.Status)

	_, err = e.k.EntitlementChallenge(e.ctx, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = e.k.EntitlementChallenge(e.ctx, &types.QueryEntitlementChallengeRequest{ChallengeId: "absent"})
	require.Equal(t, codes.NotFound, status.Code(err))
}

// invariantRegistry is the minimal sdk.InvariantRegistry needed to exercise
// RegisterInvariants; the module only calls RegisterRoute.
type invariantRegistry struct {
	routes []string
}

func (r *invariantRegistry) RegisterRoute(moduleName, route string, _ sdk.Invariant) {
	r.routes = append(r.routes, moduleName+"/"+route)
}

// TestSurface_Invariants runs every module invariant against a healthy store.
// An invariant that reports broken on genesis-like state would halt the chain on
// the first upgrade, so "clean state is clean" is worth asserting directly.
func TestSurface_Invariants(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	setBudget(t, e, 1, "1000")
	_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "100"),
	})
	require.NoError(t, err)

	reg := &invariantRegistry{}
	keeper.RegisterInvariants(reg, e.k)
	require.Len(t, reg.routes, 8)

	invariants := map[string]sdk.Invariant{
		"challenge-index":             keeper.ChallengeIndexInvariant(e.k),
		"entitlement-pool":            keeper.EntitlementPoolInvariant(e.k),
		"epoch-spent":                 keeper.EpochSpentInvariant(e.k),
		"one-settlement":              keeper.OneSettlementInvariant(e.k),
		"gas-credit-conservation":     keeper.GasCreditConservationInvariant(e.k),
		"open-challenge-escrow":       keeper.OpenChallengeEscrowInvariant(e.k),
		"open-budget-pool":            keeper.OpenBudgetPoolInvariant(e.k),
		"settlement-attestation-pair": keeper.SettlementAttestationInvariant(e.k),
	}
	for name, inv := range invariants {
		msg, broken := inv(e.ctx)
		require.False(t, broken, "%s reported broken: %s", name, msg)
	}
}
