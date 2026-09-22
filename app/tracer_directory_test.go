package app

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/stretchr/testify/require"
)

// nativeTracerNames are the tracers registered by go-ethereum's
// eth/tracers/native package, one Register call per file. Kept as an explicit
// list (rather than probing a couple of names) so that an upstream addition
// shows up as a failure to review, and so a missing blank import cannot hide
// behind a name that only the JS package happens to register.
var nativeTracerNames = []string{
	"4byteTracer",
	"callTracer",
	"flatCallTracer",
	"erc7562Tracer",
	"muxTracer",
	"noopTracer",
	"prestateTracer",
}

// TestTracerDirectoryIsPopulated pins the blank import of
// github.com/ethereum/go-ethereum/eth/tracers/native in app/app.go.
//
// Nothing in this repo, in cosmos/evm, or in the go-ethereum fork pulls that
// package in for any other reason (every other reference to it outside
// eth/tracers is a test file), so deleting the blank import compiles cleanly
// and is invisible until someone calls a trace RPC.
func TestTracerDirectoryIsPopulated(t *testing.T) {
	// Negative control: directory.IsJS answers true for every name that is NOT
	// registered, because an unknown name means "the argument is JS source"
	// (eth/tracers/dir.go, IsJS). If that ever stops holding, the loop below
	// asserts nothing and must be rewritten.
	require.True(t, tracers.DefaultDirectory.IsJS("not-a-registered-tracer"),
		"IsJS no longer reports true for unknown names; the checks below are vacuous")

	for _, name := range nativeTracerNames {
		require.Falsef(t, tracers.DefaultDirectory.IsJS(name),
			"tracer %q is not registered in tracers.DefaultDirectory: the blank import of "+
				"github.com/ethereum/go-ethereum/eth/tracers/native is missing from app/app.go "+
				"(or was reordered away). Restore it -- debug_traceCall/debug_traceTransaction "+
				"fall through to the JS path for this name and the node panics", name)
	}

	// IsJS only proves an entry exists, not that a tracer can be built from it,
	// so build one. callTracer is the one the tooling uses most.
	tr, err := tracers.DefaultDirectory.New("callTracer", &tracers.Context{}, json.RawMessage("{}"), nil)
	require.NoError(t, err, "callTracer is registered but could not be instantiated")
	require.NotNil(t, tr)
	require.NotNil(t, tr.Hooks, "callTracer came back without tracing hooks")
}

// TestInlineJSTracerCanBeInstantiated pins the blank import of
// github.com/ethereum/go-ethereum/eth/tracers/js in app/app.go.
//
// This is the import that matters for our bundler: Alto's ERC-7562 validation
// passes an inline JS tracer source in the `tracer` field of debug_traceCall, so
// the node must own a JS engine, not just a lookup table of named tracers.
func TestInlineJSTracerCanBeInstantiated(t *testing.T) {
	// First half: the registration itself. Deliberately invalid JS. With jsEval
	// registered this is a compile error; with the blank import missing,
	// directory.New reaches its final line `return d.jsEval(...)` with a nil
	// function value and the process dies with "runtime error: invalid memory
	// address or nil pointer dereference" -- the failure that broke
	// v2:aa.create against the origin chain.
	var err error
	require.NotPanics(t, func() {
		_, err = tracers.DefaultDirectory.New("((( this is not valid javascript", &tracers.Context{}, json.RawMessage("{}"), nil)
	}, "tracers.DefaultDirectory has no JS evaluator: the blank import of "+
		"github.com/ethereum/go-ethereum/eth/tracers/js is missing from app/app.go. "+
		"Every debug_traceCall carrying an inline tracer now panics the node")

	require.Error(t, err, "invalid JS was accepted instead of reported as a parse error")

	// Second half: the engine actually runs. A non-nil jsEval that cannot build
	// a tracer would still be useless, and the shape above only proves the
	// fallback is not nil. jsTracer requires result() and fault() and accepts a
	// missing step() (eth/tracers/js/goja.go, newJsTracer), so this is the
	// smallest source that must work.
	const inlineTracer = `{ result: function(ctx, db) { return { ok: true } }, fault: function(log, db) {} }`

	tr, err := tracers.DefaultDirectory.New(inlineTracer, &tracers.Context{}, json.RawMessage("{}"), nil)
	require.NoError(t, err, "an inline JS tracer was rejected by the JS engine")
	require.NotNil(t, tr)
	require.NotNil(t, tr.GetResult, "inline JS tracer came back without GetResult")
	require.NotNil(t, tr.Stop, "inline JS tracer came back without Stop")

	res, err := tr.GetResult()
	require.NoError(t, err)
	require.JSONEq(t, `{"ok":true}`, string(res), "the inline JS tracer's result() did not run")
}
