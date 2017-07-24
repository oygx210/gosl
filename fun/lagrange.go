// Copyright 2016 The Gosl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fun

import (
	"math"

	"github.com/cpmech/gosl/chk"
	"github.com/cpmech/gosl/io"
	"github.com/cpmech/gosl/la"
	"github.com/cpmech/gosl/plt"
	"github.com/cpmech/gosl/utl"
)

// Interpolation grid kinds
var (

	// UniformGridKind defines the uniform 1D grid kind
	UniformGridKind = io.NewEnum("Uniform", "fun.uniform", "U", "Uniform 1D grid")

	// ChebyGaussGridKind defines the Chebyshev-Gauss 1D grid kind
	ChebyGaussGridKind = io.NewEnum("ChebyGauss", "fun.chebygauss", "CG", "Chebyshev-Gauss 1D grid")

	// ChebyGaussLobGridKind defines the Chebyshev-Gauss-Lobatto 1D grid kind
	ChebyGaussLobGridKind = io.NewEnum("ChebyGaussLob", "fun.chebygausslob", "CGL", "Chebyshev-Gauss-Lobatto0 1D grid")
)

// LagrangeInterp implements Lagrange interpolators associated with a grid X
//
//   An interpolant I^X_N{f} (associated with a grid X; of degree N; with N+1 points)
//   is expressed in the Lagrange form as follows:
//
//                     N
//         X          ————             X
//        I {f}(x) =  \     f(x[i]) ⋅ ℓ (x)
//         N          /                i
//                    ————
//                    i = 0
//
//   where ℓ^X_i(x) is the i-th Lagrange cardinal polynomial associated with grid X and given by:
//
//                 N
//         N      ━━━━    x  -  X[j]
//        ℓ (x) = ┃  ┃  —————————————           0 ≤ i ≤ N
//         i      ┃  ┃   X[i] - X[j]
//               j = 0
//               j ≠ i
//
//   References:
//     [1] Berrut JP, Trefethen LN (2004) Barycentric Lagrange Interpolation,
//         SIAM Review Vol. 46, No. 3, pp. 501-517
//
type LagrangeInterp struct {

	// general
	N int       // degree: N = len(X)-1
	X la.Vector // grid points: len(X) = P+1; generated in [-1, 1]
	U la.Vector // function evaluated @ nodes: f(x_i)

	// barycentric
	Bary bool      // use barycentric weights [default=true]
	Eta  la.Vector // sum of log of differences: ηk = Σ ln(|xk-xl|) (k≠l)

	// computed
	D1 *la.Matrix // (dℓj/dx)(xi)
}

// NewLagrangeInterp allocates a new LagrangeInterp
//   N        -- degree
//   gridType -- type of grid; e.g. uniform
//   useLogx  -- use ln(|xk-xl|) method to compute λk
//   NOTE: the grid will be generated in [-1, 1]
func NewLagrangeInterp(N int, gridType io.Enum) (o *LagrangeInterp, err error) {

	// check
	if N < 0 {
		return nil, chk.Err("N must be at least equal to 0. N=%d is invalid\n", N)
	}

	// allocate
	o = new(LagrangeInterp)
	o.N = N

	// generate grid
	switch gridType {
	case UniformGridKind:
		o.X = utl.LinSpace(-1, 1, N+1)
	case ChebyGaussGridKind:
		o.X = ChebyshevXgauss(N)
	case ChebyGaussLobGridKind:
		o.X = ChebyshevXlob(N)
	default:
		return nil, chk.Err("cannot create grid type %q\n", gridType)
	}

	// barycentric data
	o.Bary = true
	o.Eta = make([]float64, o.N+1)

	// compute η
	for k := 0; k < o.N+1; k++ {
		for j := 0; j < o.N+1; j++ {
			if j != k {
				o.Eta[k] += math.Log(math.Abs(o.X[k] - o.X[j]))
			}
		}
	}
	return
}

// Lam computes barycentric weight λk from ηk
func (o *LagrangeInterp) Lam(k int) (λk float64) {
	λk = NegOnePowN(k+o.N) * math.Exp(-o.Eta[k])
	if math.IsInf(λk, 0) {
		chk.Panic("λ%d is infinite: %v\n", k, λk)
	}
	return
}

// Om computes the generating (nodal) polynomial associated with grid X. The nodal polynomial is
// the unique polynomial of degree N+1 and leading coefficient whose zeros are the N+1 nodes of X.
//
//                 N
//         X      ━━━━
//        ω (x) = ┃  ┃ (x - X[i])
//        N+1     ┃  ┃
//               i = 0
//
func (o *LagrangeInterp) Om(x float64) (ω float64) {
	ω = 1
	for i := 0; i < o.N+1; i++ {
		ω *= x - o.X[i]
	}
	return
}

// L computes the i-th Lagrange cardinal polynomial ℓ^X_i(x) associated with grid X
//
//                 N
//         X      ━━━━    x  -  X[j]
//        ℓ (x) = ┃  ┃  —————————————           0 ≤ i ≤ N
//         i      ┃  ┃   X[i] - X[j]
//               j = 0
//               j ≠ i
//
//   Input:
//      i -- index of X[i] point
//      x -- where to evaluate the polynomial
//   Output:
//      lix -- ℓ^X_i(x)
func (o *LagrangeInterp) L(i int, x float64) (lix float64) {

	// barycentric formula
	if o.Bary {
		if math.Abs(x-o.X[i]) < 1e-15 {
			return 1.0
		}
		var sum float64
		for j := 0; j < o.N+1; j++ {
			sum += o.Lam(j) / (x - o.X[j])
		}
		lix = (o.Lam(i) / (x - o.X[i])) / sum
		return
	}

	// standard formula
	lix = 1
	for j := 0; j < o.N+1; j++ {
		if i != j {
			lix *= (x - o.X[j]) / (o.X[i] - o.X[j])
		}
	}
	return
}

// CalcU computes f(x_i); i.e. function f(x) @ all nodes
func (o *LagrangeInterp) CalcU(f Ss) (err error) {
	if len(o.U) != o.N+1 {
		o.U = make([]float64, o.N+1)
	}
	for i := 0; i < o.N+1; i++ {
		fxi, e := f(o.X[i])
		if e != nil {
			return e
		}
		o.U[i] = fxi
	}
	return
}

// I computes the interpolation I^X_N{f}(x) @ x
//
//                     N
//         X          ————          X
//        I {f}(x) =  \     U[i] ⋅ ℓ (x)       with   U[i] = f(x[i])
//         N          /             i
//                    ————
//                    i = 0
//
//   NOTE: U[i] = f(x[i]) must be calculated with o.CalcU or set first
//
func (o *LagrangeInterp) I(x float64, f Ss) (res float64, err error) {

	// barycentric formula
	if o.Bary {
		var d, num, den float64
		for i := 0; i < o.N+1; i++ {
			d = x - o.X[i]
			if math.Abs(d) < 1e-15 {
				res = o.U[i]
				return
			}
			num += o.U[i] * o.Lam(i) / d
			den += o.Lam(i) / d
		}
		res = num / den
		return
	}

	// standard formula
	for i := 0; i < o.N+1; i++ {
		res += o.U[i] * o.L(i, x)
	}
	return
}

// CalcD1 computes the differentiation matrix D1 of the function L_i
//
//    d I{f}(x)  |         N
//   ——————————— |      =  Σ   D1_kj ⋅ f(x_j)
//        dx     |x=x_k   j=0
//
//   see [2]
//
func (o *LagrangeInterp) CalcD1() (err error) {
	o.D1 = la.NewMatrix(o.N+1, o.N+1)
	var r, v, sumRow float64
	for k := 0; k < o.N+1; k++ {
		sumRow = 0
		for j := 0; j < o.N+1; j++ {
			if k != j {
				r = NegOnePowN(k+j) * math.Exp(o.Eta[k]-o.Eta[j])
				v = r / (o.X[k] - o.X[j])
				o.D1.Set(k, j, v)
				sumRow += v
			}
		}
		o.D1.Set(k, k, -sumRow)
	}
	return
}

// CalcErrorD1 computes the maximum error due to differentiation (@ X[i]) using the D1 matrix
//   NOTE: U and D1 matrix must be computed previously
func (o *LagrangeInterp) CalcErrorD1(dfdxAna Ss) (maxDiff float64) {

	// derivative of interpolation @ x_i
	v := la.NewVector(o.N + 1)
	la.MatVecMul(v, 1, o.D1, o.U)
	io.PfYel("max(U) = %v\n", la.Vector(o.U).Max())

	// compute error
	for i := 0; i < o.N+1; i++ {
		vana, err := dfdxAna(o.X[i])
		chk.EP(err)
		diff := math.Abs(v[i] - vana)
		if diff > maxDiff {
			maxDiff = diff
		}
	}
	return
}

// EstimateLebesgue estimates the Lebesgue constant by using 10000 stations along [-1,1]
func (o *LagrangeInterp) EstimateLebesgue() (ΛN float64) {
	nsta := 10000 // generate several points along [-1,1]
	for j := 0; j < nsta; j++ {
		x := -1.0 + 2.0*float64(j)/float64(nsta-1)
		sum := math.Abs(o.L(0, x))
		for i := 1; i < o.N+1; i++ {
			sum += math.Abs(o.L(i, x))
		}
		if sum > ΛN {
			ΛN = sum
		}
	}
	return
}

// EstimateMaxErr estimates the maximum error using 10000 stations along [-1,1]
// This function also returns the location (xloc) of the estimated max error
//   Computes:
//             maxerr = max(|f(x) - I{f}(x)|)
//
//   e.g. nStations := 10000 (≥2) will generate several points along [-1,1]
//
func (o *LagrangeInterp) EstimateMaxErr(nStations int, f Ss) (maxerr, xloc float64) {
	if nStations < 2 {
		nStations = 10000
	}
	xloc = -1
	for i := 0; i < nStations; i++ {
		x := -1.0 + 2.0*float64(i)/float64(nStations-1)
		fx, err := f(x)
		if err != nil {
			chk.Panic("f(x) failed:%v\n", err)
		}
		ix, err := o.I(x, f)
		if err != nil {
			chk.Panic("I(x) failed:%v\n", err)
		}
		e := math.Abs(fx - ix)
		if math.IsNaN(e) {
			chk.Panic("error is NaN\n")
		}
		if e > maxerr {
			maxerr = e
			xloc = x
		}
	}
	return
}

// PlotLagInterpL plots cardinal polynomials ℓ
func PlotLagInterpL(N int, gridType io.Enum) {
	xx := utl.LinSpace(-1, 1, 201)
	yy := make([]float64, len(xx))
	o, _ := NewLagrangeInterp(N, gridType)
	for n := 0; n < N+1; n++ {
		for k, x := range xx {
			yy[k] = o.L(n, x)
		}
		plt.Plot(xx, yy, &plt.A{NoClip: true})
	}
	Y := make([]float64, N+1)
	plt.Plot(o.X, Y, &plt.A{C: "k", Ls: "none", M: "o", Void: true, NoClip: true})
	plt.Gll("$x$", "$\\ell(x)$", nil)
	plt.Cross(0, 0, &plt.A{C: "grey"})
	plt.HideAllBorders()
}

// PlotLagInterpW plots nodal polynomial
func PlotLagInterpW(N int, gridType io.Enum) {
	npts := 201
	xx := utl.LinSpace(-1, 1, npts)
	yy := make([]float64, len(xx))
	o, _ := NewLagrangeInterp(N, gridType)
	for k, x := range xx {
		yy[k] = o.Om(x)
	}
	Y := make([]float64, len(o.X))
	plt.Plot(o.X, Y, &plt.A{C: "k", Ls: "none", M: "o", Void: true, NoClip: true})
	plt.Plot(xx, yy, &plt.A{C: "b", Lw: 1, NoClip: true})
	plt.Gll("$x$", "$\\omega(x)$", nil)
	plt.Cross(0, 0, &plt.A{C: "grey"})
	plt.HideAllBorders()
}

// PlotLagInterpI plots Lagrange interpolation I(x) function for many degrees Nvalues
func PlotLagInterpI(Nvalues []int, gridType io.Enum, f Ss) {
	npts := 201
	xx := utl.LinSpace(-1, 1, npts)
	yy := make([]float64, len(xx))
	var err error
	for k, x := range xx {
		yy[k], err = f(x)
		chk.EP(err)
	}
	iy := make([]float64, len(xx))
	plt.Plot(xx, yy, &plt.A{C: "k", Lw: 4, NoClip: true})
	for _, N := range Nvalues {
		p, err := NewLagrangeInterp(N, gridType)
		chk.EP(err)
		p.CalcU(f)
		chk.EP(err)
		for k, x := range xx {
			iy[k], err = p.I(x, f)
			chk.EP(err)
		}
		E, xloc := p.EstimateMaxErr(0, f)
		plt.AxVline(xloc, &plt.A{C: "k", Ls: ":"})
		plt.Plot(xx, iy, &plt.A{L: io.Sf("$N=%d\\;E=%.3e$", N, E), NoClip: true})
	}
	plt.Cross(0, 0, &plt.A{C: "grey"})
	plt.Gll("$x$", "$f(x)\\quad I{f}(x)$", nil)
	plt.HideAllBorders()
}
