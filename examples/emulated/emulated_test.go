package emulated

import (
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/airchains-network/gnark/backend"
	"github.com/airchains-network/gnark/std"
	"github.com/airchains-network/gnark/std/math/emulated"
	"github.com/airchains-network/gnark/test"
)

func TestEmulatedArithmetic(t *testing.T) {
	assert := test.NewAssert(t)
	std.RegisterHints()

	var circuit, witness Circuit

	witness.X = emulated.ValueOf[emulated.Secp256k1Fp]("26959946673427741531515197488526605382048662297355296634326893985793")
	witness.Y = emulated.ValueOf[emulated.Secp256k1Fp]("53919893346855483063030394977053210764097324594710593268653787971586")
	witness.Res = emulated.ValueOf[emulated.Secp256k1Fp]("485279052387156144224396168012515269674445015885648619762653195154800")

	assert.ProverSucceeded(&circuit, &witness, test.WithCurves(ecc.BN254), test.WithBackends(backend.GROTH16), test.NoSerializationChecks())
}
