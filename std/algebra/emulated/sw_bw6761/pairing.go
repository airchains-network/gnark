package sw_bw6761

import (
	"errors"
	"fmt"
	"math/big"

	bw6761 "github.com/consensys/gnark-crypto/ecc/bw6-761"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/algebra/emulated/fields_bw6761"
	"github.com/consensys/gnark/std/math/emulated"
)

type curveF emulated.Field[emulated.BW6761Fp]

type Pairing struct {
	*fields_bw6761.Ext6
	curveF emulated.Field[emulated.BW6761Fp]
}

func NewPairing(api frontend.API) (*Pairing, error) {
	ba, err := emulated.NewField[emulated.BW6761Fp](api)
	if err != nil {
		return nil, fmt.Errorf("new base api: %w", err)
	}
	return &Pairing{
		Ext6: fields_bw6761.NewExt6(ba),
	}, nil
}

// GT target group of the pairing
type GT = fields_bw6761.E6

func NewGT(v bw6761.GT) GT {
	return GT{
		B0: fields_bw6761.E3{
			A0: emulated.ValueOf[emulated.BW6761Fp](v.B0.A0),
			A1: emulated.ValueOf[emulated.BW6761Fp](v.B0.A1),
			A2: emulated.ValueOf[emulated.BW6761Fp](v.B0.A2),
		},
		B1: fields_bw6761.E3{
			A0: emulated.ValueOf[emulated.BW6761Fp](v.B1.A0),
			A1: emulated.ValueOf[emulated.BW6761Fp](v.B1.A1),
			A2: emulated.ValueOf[emulated.BW6761Fp](v.B1.A2),
		},
	}
}

// Pair calculates the reduced pairing for a set of points e(P, Q).
//
// This function doesn't check that the inputs are in the correct subgroup. See IsInSubGroup.
func (pr Pairing) Pair(P *G1Affine, Q *G2Affine) (*GT, error) {
	f, err := pr.MillerLoop(P, Q)
	if err != nil {
		return nil, err
	}
	return pr.FinalExponentiation(f), nil
}

// FinalExponentiation computes the exponentiation (∏ᵢ zᵢ)ᵈ
// where d = (p^6-1)/r = (p^6-1)/Φ_6(p) ⋅ Φ_6(p)/r = (p^3-1)(p+1)(p^2 - p +1)/r
// we use instead d=s ⋅ (p^3-1)(p+1)(p^2 - p +1)/r
// where s is the cofactor 12(x_0+1) (El Housni and Guillevic)
func (pr Pairing) FinalExponentiation(z *GT, _z ...*GT) *GT {

	result := pr.Set(z)

	for _, a := range _z {
		result = pr.Mul(result, a)
	}

	// Easy part
	// (p^3-1)(p+1)
	buf := pr.Conjugate(result)
	result = pr.Inverse(result)
	buf = pr.Mul(buf, result)
	result = pr.Frobenius(buf)
	result = pr.Mul(result, buf)

	// Hard part (up to permutation)
	// El Housni and Guillevic
	// https://eprint.iacr.org/2020/351.pdf
	m1 := pr.Expt(result)
	_m1 := pr.Conjugate(m1)
	m2 := pr.Expt(m1)
	_m2 := pr.Conjugate(m2)
	m3 := pr.Expt(m2)
	f0 := pr.Frobenius(result)
	f0 = pr.Mul(f0, result)
	f0 = pr.Mul(f0, m2)
	m2 = pr.CyclotomicSquare(_m1)
	f0 = pr.Mul(f0, m2)
	f0_36 := pr.CyclotomicSquare(f0)
	f0_36 = pr.CyclotomicSquare(f0_36)
	f0_36 = pr.CyclotomicSquare(f0_36)
	f0_36 = pr.Mul(f0_36, f0)
	f0_36 = pr.CyclotomicSquare(f0_36)
	f0_36 = pr.CyclotomicSquare(f0_36)
	g0 := pr.Mul(result, m1)
	g0 = pr.Frobenius(g0)
	g0 = pr.Mul(g0, m3)
	g0 = pr.Mul(g0, _m2)
	g0 = pr.Mul(g0, _m1)
	g1 := pr.Expt(g0)
	_g1 := pr.Conjugate(g1)
	g2 := pr.Expt(g1)
	g3 := pr.Expt(g2)
	_g3 := pr.Conjugate(g3)
	g4 := pr.Expt(g3)
	_g4 := pr.Conjugate(g4)
	g5 := pr.Expt(g4)
	_g5 := pr.Conjugate(g5)
	g6 := pr.Expt(g5)
	gA := pr.Mul(g3, _g5)
	gA = pr.CyclotomicSquare(gA)
	gA = pr.Mul(gA, g6)
	gA = pr.Mul(gA, g1)
	gA = pr.Mul(gA, g0)
	g034 := pr.Mul(g0, g3)
	g034 = pr.Mul(g034, _g4)
	gB := pr.CyclotomicSquare(g034)
	gB = pr.Mul(gB, g034)
	gB = pr.Mul(gB, g5)
	gB = pr.Mul(gB, _g1)
	_g1g2 := pr.Mul(_g1, g2)
	gC := pr.Mul(_g3, _g1g2)
	gC = pr.CyclotomicSquare(gC)
	gC = pr.Mul(gC, _g1g2)
	gC = pr.Mul(gC, g0)
	gC = pr.CyclotomicSquare(gC)
	gC = pr.Mul(gC, g2)
	gC = pr.Mul(gC, g0)
	gC = pr.Mul(gC, g4)

	// ht, hy = 13, 9
	// c1 = ht**2+3*hy**2 = 412
	h1 := pr.Expc1(gA)
	// c2 = ht+hy = 22
	h2 := pr.Expc2(gB)
	h2g2C := pr.CyclotomicSquare(gC)
	h2g2C = pr.Mul(h2g2C, h2)
	h4 := pr.CyclotomicSquare(h2g2C)
	h4 = pr.Mul(h4, h2g2C)
	h4 = pr.CyclotomicSquare(h4)
	result = pr.Mul(h1, h4)
	result = pr.Mul(result, f0_36)

	return result
}

// lineEvaluation represents a sparse Fp12 Elmt (result of the line evaluation)
// line: 1 + R0(x/y) + R1(1/y) = 0 instead of R0'*y + R1'*x + R2' = 0 This
// makes the multiplication by lines (MulBy034) and between lines (Mul034By034)
// circuit-efficient.
type lineEvaluation struct {
	R0, R1 emulated.Element[emulated.BW6761Fp]
}

// seed x₀=9586122913090633729
//
// x₀+1 in 2-NAF
var loopCounter1 = [64]int8{
	0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, 1, 0,
	0, 1, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 0, 1,
}

// x₀^3-x₀^2-x₀ in 2-NAF
var loopCounter2 = [190]int8{
	-1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 1, 0, 0, -1, 0, 1, 0, -1,
	0, 0, 0, 0, -1, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 1, 0, 1, 0, 0, 0, 1, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, -1, 0, 0, 1, 0, 0, 0, -1, 0, 0,
	-1, 0, 1, 0, -1, 0, 0, 0, 1, 0, 0, 1, 0, -1, 0, 1, 0, 1, 0, 0, 0, 1, 0, -1, 0,
	-1, 0, 0, 0, 0, 0, 1, 0, 0, 1,
}

func (pr Pairing) firstLoop(P *G1Affine, Q *G2Affine) (*GT, error) {
	res := pr.Ext6.One()

	var l1, l2 *lineEvaluation
	var yInv, xOverY *emulated.Element[emulated.BW6761Fp]

	Qacc := Q
	QNeg := &G2Affine{X: Q.X, Y: *pr.curveF.Neg(&Q.Y)}
	// P and Q are supposed to be on G1 and G2 respectively of prime order r.
	// The point (x,0) is of order 2. But this function does not check
	// subgroup membership.
	yInv = pr.curveF.Inverse(&P.Y)
	xOverY = pr.curveF.MulMod(&P.X, yInv)

	for i := 62; i >= 0; i-- {
		// mutualize the square among n Miller loops
		// (∏ᵢfᵢ)²
		res = pr.Square(res)

		switch loopCounter1[i] {

		case 0:
			// precompute lines
			// Qacc ← 2Qacc and l1 the tangent ℓ passing 2Qacc
			Qacc, l1 = pr.doubleStep(Qacc)

			// line evaluation at P
			l1.R0 = *pr.curveF.Mul(&l1.R0, xOverY)
			l1.R1 = *pr.curveF.Mul(&l1.R1, yInv)
			res = pr.MulBy014(res, &l1.R1, &l1.R0)

		case 1:
			// Qacc ← 2Qacc+Q,
			// l1 the line ℓ passing Qacc and Q
			// l2 the line ℓ passing (Qacc+Q) and Qacc
			Qacc, l1, l2 = pr.doubleAndAddStep(Qacc, Q)

			// line evaluation at P
			l1.R0 = *pr.curveF.Mul(&l1.R0, xOverY)
			l1.R1 = *pr.curveF.Mul(&l1.R1, yInv)
			res = pr.MulBy014(res, &l1.R1, &l1.R0)

			// line evaluation at P
			l2.R0 = *pr.curveF.Mul(&l2.R0, xOverY)
			l2.R1 = *pr.curveF.Mul(&l2.R1, yInv)
			res = pr.MulBy014(res, &l2.R1, &l2.R0)

		case -1:
			// Qacc ← 2Qacc-Q,
			// l1 the line ℓ passing Qacc and -Q
			// l2 the line ℓ passing (Qacc-Q) and Qacc
			Qacc, l1, l2 = pr.doubleAndAddStep(Qacc, QNeg)

			// line evaluation at P
			l1.R0 = *pr.curveF.Mul(&l1.R0, xOverY)
			l1.R1 = *pr.curveF.Mul(&l1.R1, yInv)
			res = pr.MulBy014(res, &l1.R1, &l1.R0)

			// line evaluation at P
			l2.R0 = *pr.curveF.Mul(&l2.R0, xOverY)
			l2.R1 = *pr.curveF.Mul(&l2.R1, yInv)
			res = pr.MulBy014(res, &l2.R1, &l2.R0)

		default:
			return nil, errors.New("invalid loopCounter")
		}
	}

	return res, nil

}

func (pr Pairing) secondLoop(P *G1Affine, Q *G2Affine) (*GT, error) {
	res := pr.Ext6.One()

	var l1, l2 *lineEvaluation
	var yInv, xOverY *emulated.Element[emulated.BW6761Fp]

	Qacc := Q
	QNeg := &G2Affine{X: Q.X, Y: *pr.curveF.Neg(&Q.Y)}
	// P and Q are supposed to be on G1 and G2 respectively of prime order r.
	// The point (x,0) is of order 2. But this function does not check
	// subgroup membership.
	yInv = pr.curveF.Inverse(&P.Y)
	xOverY = pr.curveF.MulMod(&P.X, yInv)

	for i := 188; i >= 0; i-- {
		// mutualize the square among n Miller loops
		// (∏ᵢfᵢ)²
		res = pr.Square(res)

		switch loopCounter2[i] {

		case 0:
			// precompute lines
			// Qacc ← 2Qacc and l1 the tangent ℓ passing 2Qacc
			Qacc, l1 = pr.doubleStep(Qacc)

			// line evaluation at P
			l1.R0 = *pr.curveF.Mul(&l1.R0, xOverY)
			l1.R1 = *pr.curveF.Mul(&l1.R1, yInv)
			res = pr.MulBy014(res, &l1.R1, &l1.R0)

		case 1:
			// Qacc ← 2Qacc+Q,
			// l1 the line ℓ passing Qacc and Q
			// l2 the line ℓ passing (Qacc+Q) and Qacc
			Qacc, l1, l2 = pr.doubleAndAddStep(Qacc, Q)

			// line evaluation at P
			l1.R0 = *pr.curveF.Mul(&l1.R0, xOverY)
			l1.R1 = *pr.curveF.Mul(&l1.R1, yInv)
			res = pr.MulBy014(res, &l1.R1, &l1.R0)

			// line evaluation at P
			l2.R0 = *pr.curveF.Mul(&l2.R0, xOverY)
			l2.R1 = *pr.curveF.Mul(&l2.R1, yInv)
			res = pr.MulBy014(res, &l2.R1, &l2.R0)

		case -1:
			// Qacc ← 2Qacc-Q,
			// l1 the line ℓ passing Qacc and -Q
			// l2 the line ℓ passing (Qacc-Q) and Qacc
			Qacc, l1, l2 = pr.doubleAndAddStep(Qacc, QNeg)

			// line evaluation at P
			l1.R0 = *pr.curveF.Mul(&l1.R0, xOverY)
			l1.R1 = *pr.curveF.Mul(&l1.R1, yInv)
			res = pr.MulBy014(res, &l1.R1, &l1.R0)

			// line evaluation at P
			l2.R0 = *pr.curveF.Mul(&l2.R0, xOverY)
			l2.R1 = *pr.curveF.Mul(&l2.R1, yInv)
			res = pr.MulBy014(res, &l2.R1, &l2.R0)

		default:
			return nil, errors.New("invalid loopCounter")
		}
	}

	return res, nil

}

// MillerLoop computes the Miller loop
// Naive BW6 Miller loop: Eq (1) in [bw6-post]
// f_{u+1,Q}(P) * (f_{u^3-u^2-u,Q}(P))^q
//
// [https://hackmd.io/@gnark/BW6-761-changes]
func (pr Pairing) MillerLoop(P *G1Affine, Q *G2Affine) (*GT, error) {
	// f_{u+1,Q}(P)
	ml1, _ := pr.firstLoop(P, Q)
	// f_{u^3-u^2-u,Q}(P)
	ml2, _ := pr.secondLoop(P, Q)
	ml := pr.Frobenius(ml2)
	ml = pr.Mul(ml, ml1)
	return ml, nil
}

// doubleAndAddStep doubles p1 and adds p2 to the result in affine coordinates, and evaluates the line in Miller loop
// https://eprint.iacr.org/2022/1162 (Section 6.1)
func (pr Pairing) doubleAndAddStep(p1, p2 *G2Affine) (*G2Affine, *lineEvaluation, *lineEvaluation) {

	var line1, line2 lineEvaluation
	var p G2Affine

	// compute λ1 = (y2-y1)/(x2-x1)
	n := pr.curveF.Sub(&p1.Y, &p2.Y)
	d := pr.curveF.Sub(&p1.X, &p2.X)
	l1 := pr.curveF.Div(n, d)

	// compute x3 =λ1²-x1-x2
	x3 := pr.curveF.Mul(l1, l1)
	x3 = pr.curveF.Sub(x3, &p1.X)
	x3 = pr.curveF.Sub(x3, &p2.X)

	// omit y3 computation

	// compute line1
	line1.R0 = *pr.curveF.Neg(l1)
	line1.R1 = *pr.curveF.Mul(l1, &p1.X)
	line1.R1 = *pr.curveF.Sub(&line1.R1, &p1.Y)

	// compute λ2 = -λ1-2y1/(x3-x1)
	n = pr.curveF.Add(&p1.Y, &p1.Y)
	d = pr.curveF.Sub(x3, &p1.X)
	l2 := pr.curveF.Div(n, d)
	l2 = pr.curveF.Add(l2, l1)
	l2 = pr.curveF.Neg(l2)

	// compute x4 = λ2²-x1-x3
	x4 := pr.curveF.Mul(l2, l2)
	x4 = pr.curveF.Sub(x4, &p1.X)
	x4 = pr.curveF.Sub(x4, x3)

	// compute y4 = λ2(x1 - x4)-y1
	y4 := pr.curveF.Sub(&p1.X, x4)
	y4 = pr.curveF.Mul(l2, y4)
	y4 = pr.curveF.Sub(y4, &p1.Y)

	p.X = *x4
	p.Y = *y4

	// compute line2
	line2.R0 = *pr.curveF.Neg(l2)
	line2.R1 = *pr.curveF.Mul(l2, &p1.X)
	line2.R1 = *pr.curveF.Sub(&line2.R1, &p1.Y)

	return &p, &line1, &line2
}

// doubleStep doubles a point in affine coordinates, and evaluates the line in Miller loop
// https://eprint.iacr.org/2022/1162 (Section 6.1)
func (pr Pairing) doubleStep(p1 *G2Affine) (*G2Affine, *lineEvaluation) {

	var p G2Affine
	var line lineEvaluation

	// λ = 3x²/2y
	n := pr.curveF.Mul(&p1.X, &p1.X)
	three := big.NewInt(3)
	n = pr.curveF.MulConst(n, three)
	d := pr.curveF.Add(&p1.Y, &p1.Y)
	λ := pr.curveF.Div(n, d)

	// xr = λ²-2x
	xr := pr.curveF.Mul(λ, λ)
	xr = pr.curveF.Sub(xr, &p1.X)
	xr = pr.curveF.Sub(xr, &p1.X)

	// yr = λ(x-xr)-y
	yr := pr.curveF.Sub(&p1.X, xr)
	yr = pr.curveF.Mul(λ, yr)
	yr = pr.curveF.Sub(yr, &p1.Y)

	p.X = *xr
	p.Y = *yr

	line.R0 = *pr.curveF.Neg(λ)
	line.R1 = *pr.curveF.Mul(λ, &p1.X)
	line.R1 = *pr.curveF.Sub(&line.R1, &p1.Y)

	return &p, &line

}

// addStep adds two points in affine coordinates, and evaluates the line in Miller loop
// https://eprint.iacr.org/2022/1162 (Section 6.1)
func (pr Pairing) addStep(p1, p2 *G2Affine) (*G2Affine, *lineEvaluation) {

	// compute λ = (y2-y1)/(x2-x1)
	p2ypy := pr.curveF.Sub(&p2.Y, &p1.Y)
	p2xpx := pr.curveF.Sub(&p2.X, &p1.X)
	λ := pr.curveF.Div(p2ypy, p2xpx)

	// xr = λ²-x1-x2
	λλ := pr.curveF.Mul(λ, λ)
	p2xpx = pr.curveF.Add(&p1.X, &p2.X)
	xr := pr.curveF.Sub(λλ, p2xpx)

	// yr = λ(x1-xr) - y1
	pxrx := pr.curveF.Sub(&p1.X, xr)
	λpxrx := pr.curveF.Mul(λ, pxrx)
	yr := pr.curveF.Sub(λpxrx, &p1.Y)

	var res G2Affine
	res.X = *xr
	res.Y = *yr

	var line lineEvaluation
	line.R0 = *pr.curveF.Neg(λ)
	line.R1 = *pr.curveF.Mul(λ, &p1.X)
	line.R1 = *pr.curveF.Sub(&line.R1, &p1.Y)

	return &res, &line

}

// lineCompute computes the line that goes through p1 and p2 but does not compute p1+p2
func (pr Pairing) lineCompute(p1, p2 *G2Affine) *lineEvaluation {

	// compute λ = (y2-y1)/(x2-x1)
	qypy := pr.curveF.Sub(&p2.Y, &p1.Y)
	qxpx := pr.curveF.Sub(&p2.X, &p1.X)
	λ := pr.curveF.Div(qypy, qxpx)

	var line lineEvaluation
	line.R0 = *pr.curveF.Neg(λ)
	line.R1 = *pr.curveF.Mul(λ, &p1.X)
	line.R1 = *pr.curveF.Sub(&line.R1, &p1.Y)

	return &line

}
