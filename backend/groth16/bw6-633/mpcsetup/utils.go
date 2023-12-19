// Copyright 2020 ConsenSys Software Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by gnark DO NOT EDIT

package mpcsetup

import (
	"bytes"
	"math/big"
	"math/bits"
	"runtime"

	"github.com/consensys/gnark-crypto/ecc"
	curve "github.com/consensys/gnark-crypto/ecc/bw6-633"
	"github.com/consensys/gnark-crypto/ecc/bw6-633/fr"
	"github.com/airchains-network/gnark/internal/utils"
)

type PublicKey struct {
	SG  curve.G1Affine
	SXG curve.G1Affine
	XR  curve.G2Affine
}

func newPublicKey(x fr.Element, challenge []byte, dst byte) PublicKey {
	var pk PublicKey
	_, _, g1, _ := curve.Generators()

	var s fr.Element
	var sBi big.Int
	s.SetRandom()
	s.BigInt(&sBi)
	pk.SG.ScalarMultiplication(&g1, &sBi)

	// compute x*sG1
	var xBi big.Int
	x.BigInt(&xBi)
	pk.SXG.ScalarMultiplication(&pk.SG, &xBi)

	// generate R based on sG1, sxG1, challenge, and domain separation tag (tau, alpha or beta)
	R := genR(pk.SG, pk.SXG, challenge, dst)

	// compute x*spG2
	pk.XR.ScalarMultiplication(&R, &xBi)
	return pk
}

func bitReverse[T any](a []T) {
	n := uint64(len(a))
	nn := uint64(64 - bits.TrailingZeros64(n))

	for i := uint64(0); i < n; i++ {
		irev := bits.Reverse64(i) >> nn
		if irev > i {
			a[i], a[irev] = a[irev], a[i]
		}
	}
}

// Returns [1, a, a², ..., aⁿ⁻¹ ] in Montgomery form
func powers(a fr.Element, n int) []fr.Element {
	result := make([]fr.Element, n)
	result[0] = fr.NewElement(1)
	for i := 1; i < n; i++ {
		result[i].Mul(&result[i-1], &a)
	}
	return result
}

// Returns [aᵢAᵢ, ...] in G1
func scaleG1InPlace(A []curve.G1Affine, a []fr.Element) {
	utils.Parallelize(len(A), func(start, end int) {
		var tmp big.Int
		for i := start; i < end; i++ {
			a[i].BigInt(&tmp)
			A[i].ScalarMultiplication(&A[i], &tmp)
		}
	})
}

// Returns [aᵢAᵢ, ...] in G2
func scaleG2InPlace(A []curve.G2Affine, a []fr.Element) {
	utils.Parallelize(len(A), func(start, end int) {
		var tmp big.Int
		for i := start; i < end; i++ {
			a[i].BigInt(&tmp)
			A[i].ScalarMultiplication(&A[i], &tmp)
		}
	})
}

// Check e(a₁, a₂) = e(b₁, b₂)
func sameRatio(a1, b1 curve.G1Affine, a2, b2 curve.G2Affine) bool {
	if !a1.IsInSubGroup() || !b1.IsInSubGroup() || !a2.IsInSubGroup() || !b2.IsInSubGroup() {
		panic("invalid point not in subgroup")
	}
	var na2 curve.G2Affine
	na2.Neg(&a2)
	res, err := curve.PairingCheck(
		[]curve.G1Affine{a1, b1},
		[]curve.G2Affine{na2, b2})
	if err != nil {
		panic(err)
	}
	return res
}

// returns a = ∑ rᵢAᵢ, b = ∑ rᵢBᵢ
func merge(A, B []curve.G1Affine) (a, b curve.G1Affine) {
	nc := runtime.NumCPU()
	r := make([]fr.Element, len(A))
	for i := 0; i < len(A); i++ {
		r[i].SetRandom()
	}
	a.MultiExp(A, r, ecc.MultiExpConfig{NbTasks: nc / 2})
	b.MultiExp(B, r, ecc.MultiExpConfig{NbTasks: nc / 2})
	return
}

// L1 = ∑ rᵢAᵢ, L2 = ∑ rᵢAᵢ₊₁ in G1
func linearCombinationG1(A []curve.G1Affine) (L1, L2 curve.G1Affine) {
	nc := runtime.NumCPU()
	n := len(A)
	r := make([]fr.Element, n-1)
	for i := 0; i < n-1; i++ {
		r[i].SetRandom()
	}
	L1.MultiExp(A[:n-1], r, ecc.MultiExpConfig{NbTasks: nc / 2})
	L2.MultiExp(A[1:], r, ecc.MultiExpConfig{NbTasks: nc / 2})
	return
}

// L1 = ∑ rᵢAᵢ, L2 = ∑ rᵢAᵢ₊₁ in G2
func linearCombinationG2(A []curve.G2Affine) (L1, L2 curve.G2Affine) {
	nc := runtime.NumCPU()
	n := len(A)
	r := make([]fr.Element, n-1)
	for i := 0; i < n-1; i++ {
		r[i].SetRandom()
	}
	L1.MultiExp(A[:n-1], r, ecc.MultiExpConfig{NbTasks: nc / 2})
	L2.MultiExp(A[1:], r, ecc.MultiExpConfig{NbTasks: nc / 2})
	return
}

// Generate R in G₂ as Hash(gˢ, gˢˣ, challenge, dst)
func genR(sG1, sxG1 curve.G1Affine, challenge []byte, dst byte) curve.G2Affine {
	var buf bytes.Buffer
	buf.Grow(len(challenge) + curve.SizeOfG1AffineUncompressed*2)
	buf.Write(sG1.Marshal())
	buf.Write(sxG1.Marshal())
	buf.Write(challenge)
	spG2, err := curve.HashToG2(buf.Bytes(), []byte{dst})
	if err != nil {
		panic(err)
	}
	return spG2
}
