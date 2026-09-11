package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// recordingRegistry counts the invariant routes a manager tries to register.
type recordingRegistry struct{ routes []string }

func (r *recordingRegistry) RegisterRoute(moduleName, route string, _ sdk.Invariant) {
	r.routes = append(r.routes, moduleName+"/"+route)
}

// TestModuleManagerRegistersNoInvariants is a REVERSE PIN, not a description of
// desirable behaviour.
//
// It pins what this repository currently depends on: cosmos-sdk v0.53.6 ships
// module.Manager.RegisterInvariants as a deliberate no-op
// (types/module/module.go:454-457), so app.CrisisKeeper ends up with an empty
// route set even though app/app.go calls it on a manager that holds dozens of
// modules. Everything downstream of that emptiness is therefore a no-op too --
// x/crisis' EndBlocker, its InitGenesis assertion, and the zero-height export
// assertion in app/export.go all iterate zero routes and log that they checked
// something.
//
// It fails for the right reason. The day an SDK upgrade turns that no-op back
// into a real fan-out, x/collection's SupplyInvariant becomes live inside
// x/crisis -- in EndBlocker on every inv-check-period height, in InitGenesis on
// every chain start, and in the zero-height export -- and the invariant walks
// every class and every NFT. A supply mismatch on live state would then panic
// instead of being reported, which is a chain-halt and a node-won't-start
// decision rather than a dependency-bump detail.
//
// If this test fails: read the comment above app.mm.RegisterInvariants in
// app/app.go and the reachability note in
// x/collection/keeper/invariants.go, measure the invariant against a real
// chain state, and only then decide whether the wiring is wanted.
func TestModuleManagerRegistersNoInvariants(t *testing.T) {
	app, _ := sharedTestApp(t)

	require.NotEmpty(t, app.mm.Modules,
		"the manager holds no modules, so registering nothing proves nothing")

	rec := &recordingRegistry{}
	app.mm.RegisterInvariants(rec)

	require.Empty(t, rec.routes,
		"module.Manager.RegisterInvariants now registers routes: x/crisis will run them "+
			"in EndBlocker/InitGenesis and a failing invariant will halt instead of reporting")

	require.Empty(t, app.CrisisKeeper.Routes(),
		"x/crisis already holds invariant routes; re-decide the wiring before shipping")
}
