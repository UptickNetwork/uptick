package app

import "testing"

// BenchmarkSimulation measures the cost of one full simulation cycle: building
// the application, generating its genesis, executing the simulated operations
// and exporting the result.
//
// It replaces the invariant benchmark the Makefile target used to name. SDK
// v0.53 turned module.Manager.RegisterInvariants into a no-op and x/simulation
// stopped asserting invariants, so nothing in the simulation stack can check
// them any more and a benchmark named after them would only report timings for
// work that never happens. This benchmark measures the harness the Makefile
// simulation targets actually drive, and fails if the run produced no state.
func BenchmarkSimulation(b *testing.B) {
	if !*simEnabled {
		b.Skip("simulation disabled: pass -Enabled=true to run")
	}

	config := simConfig(b, 1, 5)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		exported := runSimulation(b, config, nil)
		if len(exported.AppState) == 0 {
			b.Fatal("the simulation exported an empty application state")
		}
	}
}
