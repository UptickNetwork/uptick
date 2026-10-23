package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}

func (k Keeper) Application(goCtx context.Context, req *types.QueryApplicationRequest) (*types.QueryApplicationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	app, found := k.GetApplication(ctx, req.AppId)
	if !found {
		return nil, status.Error(codes.NotFound, "application not found")
	}
	return &types.QueryApplicationResponse{Application: app}, nil
}

func (k Keeper) Applications(goCtx context.Context, req *types.QueryApplicationsRequest) (*types.QueryApplicationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	pageReq, err := shapePageRequest(req.Pagination)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	store := prefixStore(k.store(ctx), types.KeyPrefixApplication)
	var apps []types.Application
	pageRes, err := query.FilteredPaginate(store, pageReq, func(_ []byte, value []byte, accumulate bool) (bool, error) {
		var app types.Application
		if err := k.cdc.Unmarshal(value, &app); err != nil {
			return false, err
		}
		if req.Status != types.ApplicationStatusUnspecified && app.Status != req.Status {
			return false, nil
		}
		if accumulate {
			apps = append(apps, app)
		}
		return true, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryApplicationsResponse{Applications: apps, Pagination: pageRes}, nil
}

func (k Keeper) EpochBudget(goCtx context.Context, req *types.QueryEpochBudgetRequest) (*types.QueryEpochBudgetResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	b, found := k.GetEpochBudget(ctx, req.Epoch)
	if !found {
		return nil, status.Error(codes.NotFound, "epoch budget not found")
	}
	return &types.QueryEpochBudgetResponse{Budget: b}, nil
}

func (k Keeper) Attestation(goCtx context.Context, req *types.QueryAttestationRequest) (*types.QueryAttestationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	a, found := k.GetAttestation(ctx, req.AppId, req.Epoch)
	if !found {
		return nil, status.Error(codes.NotFound, "attestation not found")
	}
	return &types.QueryAttestationResponse{Attestation: a}, nil
}

func (k Keeper) Attestations(goCtx context.Context, req *types.QueryAttestationsRequest) (*types.QueryAttestationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	pageReq, err := shapePageRequest(req.Pagination)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	store := prefixStore(k.store(ctx), types.KeyPrefixAttestation)
	var out []types.AttestationCommitment
	pageRes, err := query.FilteredPaginate(store, pageReq, func(_ []byte, value []byte, accumulate bool) (bool, error) {
		var a types.AttestationCommitment
		if err := k.cdc.Unmarshal(value, &a); err != nil {
			return false, err
		}
		if req.Status != types.AttestationStatusUnspecified && a.Status != req.Status {
			return false, nil
		}
		if accumulate {
			out = append(out, a)
		}
		return true, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryAttestationsResponse{Attestations: out, Pagination: pageRes}, nil
}

func (k Keeper) Challenge(goCtx context.Context, req *types.QueryChallengeRequest) (*types.QueryChallengeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	c, found := k.GetChallenge(ctx, req.ChallengeId)
	if !found {
		return nil, status.Error(codes.NotFound, "challenge not found")
	}
	return &types.QueryChallengeResponse{Challenge: c}, nil
}

func (k Keeper) OpenChallenges(goCtx context.Context, req *types.QueryOpenChallengesRequest) (*types.QueryOpenChallengesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	pageReq, err := shapePageRequest(req.Pagination)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Prefer open-index when filtering by app (+ optional epoch); otherwise scan challenge store.
	var prefix []byte
	if len(req.AppId) > 0 {
		if req.Epoch > 0 {
			prefix = types.OpenChallengeIndexPrefix(req.AppId, req.Epoch)
		} else {
			prefix = append(append([]byte{}, types.KeyPrefixOpenChallengeIndex...), req.AppId...)
		}
		store := prefixStore(k.store(ctx), prefix)
		var out []types.Challenge
		pageRes, err := query.Paginate(store, pageReq, func(key, _ []byte) error {
			if len(key) < 8 {
				return nil
			}
			id := binaryUint64(key[len(key)-8:])
			c, ok := k.GetChallenge(ctx, id)
			if ok {
				out = append(out, c)
			}
			return nil
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		return &types.QueryOpenChallengesResponse{Challenges: out, Pagination: pageRes}, nil
	}

	store := prefixStore(k.store(ctx), types.KeyPrefixChallenge)
	var out []types.Challenge
	pageRes, err := query.FilteredPaginate(store, pageReq, func(_ []byte, value []byte, accumulate bool) (bool, error) {
		var c types.Challenge
		if err := k.cdc.Unmarshal(value, &c); err != nil {
			return false, err
		}
		if c.Status != types.ChallengeStatusOpen {
			return false, nil
		}
		if accumulate {
			out = append(out, c)
		}
		return true, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryOpenChallengesResponse{Challenges: out, Pagination: pageRes}, nil
}

func (k Keeper) Settlement(goCtx context.Context, req *types.QuerySettlementRequest) (*types.QuerySettlementResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	s, found := k.GetSettlement(ctx, req.AppId, req.Epoch)
	if !found {
		return nil, status.Error(codes.NotFound, "settlement not found")
	}
	return &types.QuerySettlementResponse{Settlement: s}, nil
}

func (k Keeper) Settlements(goCtx context.Context, req *types.QuerySettlementsRequest) (*types.QuerySettlementsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	pageReq, err := shapePageRequest(req.Pagination)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	store := prefixStore(k.store(ctx), types.KeyPrefixSettlement)
	var out []types.Settlement
	pageRes, err := query.Paginate(store, pageReq, func(_, value []byte) error {
		var s types.Settlement
		if err := k.cdc.Unmarshal(value, &s); err != nil {
			return err
		}
		out = append(out, s)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QuerySettlementsResponse{Settlements: out, Pagination: pageRes}, nil
}

func (k Keeper) GasCredit(goCtx context.Context, req *types.QueryGasCreditRequest) (*types.QueryGasCreditResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	a, found := k.GetGasCredit(ctx, req.AppId)
	if !found {
		a = types.GasCreditAccount{AppId: req.AppId, Balance: "0", TotalIssued: "0", TotalConsumed: "0"}
	}
	return &types.QueryGasCreditResponse{Account: a}, nil
}

func (k Keeper) GasCreditConsumptions(goCtx context.Context, req *types.QueryGasCreditConsumptionsRequest) (*types.QueryGasCreditConsumptionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	pageReq, err := shapePageRequest(req.Pagination)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	prefix := append(append([]byte{}, types.KeyPrefixGasCreditConsumption...), req.AppId...)
	store := prefixStore(k.store(ctx), prefix)
	var out []types.GasCreditConsumption
	pageRes, err := query.Paginate(store, pageReq, func(_, value []byte) error {
		var c types.GasCreditConsumption
		if err := k.cdc.Unmarshal(value, &c); err != nil {
			return err
		}
		out = append(out, c)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryGasCreditConsumptionsResponse{Consumptions: out, Pagination: pageRes}, nil
}

func prefixStore(store storetypes.KVStore, pfx []byte) storetypes.KVStore {
	return prefix.NewStore(store, pfx)
}

// --- V1 entitlement flow queries ---

// Pool returns the entitlement pool record, module balance and flow status.
func (k Keeper) Pool(goCtx context.Context, req *types.QueryPoolRequest) (*types.QueryPoolResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	pool, found := k.GetEntitlementPool(ctx, req.PoolId)
	if !found {
		return nil, status.Error(codes.NotFound, "entitlement pool not found")
	}
	params := k.GetParams(ctx)
	modAddr := k.ak.GetModuleAddress(types.EntitlementPoolName)
	bal := k.bank.GetBalance(ctx, modAddr, pool.Denom)
	return &types.QueryPoolResponse{
		Pool:                   pool,
		ModuleBalance:          bal.Amount.String(),
		EntitlementFlowEnabled: types.EntitlementFlowEnabled(pool, params, types.AmountString(bal.Amount)),
	}, nil
}

// Roles returns registered role keys, optionally filtered by role and status.
func (k Keeper) Roles(goCtx context.Context, req *types.QueryRolesRequest) (*types.QueryRolesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	roles := []types.EntitlementRole{
		types.EntitlementRolePublisher,
		types.EntitlementRoleApprover,
		types.EntitlementRoleReviewer,
		types.EntitlementRoleRisk,
		types.EntitlementRoleAudit,
	}
	if req.Role != types.EntitlementRoleUnspecified {
		roles = []types.EntitlementRole{req.Role}
	}
	var keys []types.RoleKey
	for _, role := range roles {
		k.IterateRoleKeys(ctx, role, func(rk types.RoleKey) bool {
			if req.Status != types.RoleKeyStatusUnspecified && rk.Status != req.Status {
				return true
			}
			keys = append(keys, rk)
			return true
		})
	}
	return &types.QueryRolesResponse{Keys: keys}, nil
}

func (k Keeper) Candidate(goCtx context.Context, req *types.QueryCandidateRequest) (*types.QueryCandidateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cand, found := k.GetCandidate(ctx, req.EntitlementId)
	if !found {
		return nil, status.Error(codes.NotFound, "candidate not found")
	}
	return &types.QueryCandidateResponse{Candidate: cand}, nil
}

func (k Keeper) EntitlementChallenge(goCtx context.Context, req *types.QueryEntitlementChallengeRequest) (*types.QueryEntitlementChallengeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	chal, found := k.GetEntitlementChallenge(ctx, req.ChallengeId)
	if !found {
		return nil, status.Error(codes.NotFound, "entitlement challenge not found")
	}
	return &types.QueryEntitlementChallengeResponse{Challenge: chal}, nil
}

func (k Keeper) Entitlement(goCtx context.Context, req *types.QueryEntitlementRequest) (*types.QueryEntitlementResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	ent, found := k.GetEntitlement(ctx, req.EntitlementId)
	if !found {
		return nil, status.Error(codes.NotFound, "entitlement not found")
	}
	return &types.QueryEntitlementResponse{Entitlement: ent}, nil
}

func (k Keeper) EntitlementsByApp(goCtx context.Context, req *types.QueryEntitlementsByAppRequest) (*types.QueryEntitlementsByAppResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	pageReq, err := shapePageRequest(req.Pagination)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	store := prefixStore(k.store(ctx), types.KeyPrefixEntitlement)
	var out []types.Entitlement
	pageRes, err := query.FilteredPaginate(store, pageReq, func(_ []byte, value []byte, accumulate bool) (bool, error) {
		var ent types.Entitlement
		if err := k.cdc.Unmarshal(value, &ent); err != nil {
			return false, err
		}
		if ent.AppId != req.AppId {
			return false, nil
		}
		if accumulate {
			out = append(out, ent)
		}
		return true, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryEntitlementsByAppResponse{Entitlements: out, Pagination: pageRes}, nil
}
