package std

import (
	"sync"

	"github.com/airchains-network/gnark/constraint/solver"
	"github.com/airchains-network/gnark/std/algebra/emulated/sw_emulated"
	"github.com/airchains-network/gnark/std/algebra/native/sw_bls12377"
	"github.com/airchains-network/gnark/std/algebra/native/sw_bls24315"
	"github.com/airchains-network/gnark/std/evmprecompiles"
	"github.com/airchains-network/gnark/std/internal/logderivarg"
	"github.com/airchains-network/gnark/std/math/bits"
	"github.com/airchains-network/gnark/std/math/bitslice"
	"github.com/airchains-network/gnark/std/math/cmp"
	"github.com/airchains-network/gnark/std/math/emulated"
	"github.com/airchains-network/gnark/std/rangecheck"
	"github.com/airchains-network/gnark/std/selector"
)

var registerOnce sync.Once

// RegisterHints register all gnark/std hints
// In the case where the Solver/Prover code is loaded alongside the circuit, this is not useful.
// However, if a Solver/Prover services consumes serialized constraint systems, it has no way to
// know which hints were registered; caller code may add them through backend.WithHints(...).
func RegisterHints() {
	registerOnce.Do(registerHints)
}

func registerHints() {
	// note that importing these packages may already trigger a call to solver.RegisterHint(...)
	solver.RegisterHint(sw_bls24315.DecomposeScalarG1)
	solver.RegisterHint(sw_bls12377.DecomposeScalarG1)
	solver.RegisterHint(sw_bls24315.DecomposeScalarG2)
	solver.RegisterHint(sw_bls12377.DecomposeScalarG2)
	solver.RegisterHint(bits.GetHints()...)
	solver.RegisterHint(cmp.GetHints()...)
	solver.RegisterHint(selector.GetHints()...)
	solver.RegisterHint(emulated.GetHints()...)
	solver.RegisterHint(rangecheck.GetHints()...)
	solver.RegisterHint(evmprecompiles.GetHints()...)
	solver.RegisterHint(logderivarg.GetHints()...)
	solver.RegisterHint(bitslice.GetHints()...)
	solver.RegisterHint(sw_emulated.GetHints()...)
}
