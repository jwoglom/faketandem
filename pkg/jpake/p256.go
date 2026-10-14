package jpake

import (
	"errors"
	"math/big"
)

// NIST P-256 (secp256r1), the curve pumpX2's EcJpake and the Tandem pump use.
// The arithmetic is plain affine big.Int: a handshake does a few dozen scalar
// multiplications, and crypto/elliptic's point API is deprecated.
var (
	curveP, _  = new(big.Int).SetString("ffffffff00000001000000000000000000000000ffffffffffffffffffffffff", 16)
	curveN, _  = new(big.Int).SetString("ffffffff00000000ffffffffffffffffbce6faada7179e84f3b9cac2fc632551", 16)
	curveB, _  = new(big.Int).SetString("5ac635d8aa3a93e7b3ebbd55769886bc651d06b0cc53b0f63bce3c3e27d2604b", 16)
	curveGx, _ = new(big.Int).SetString("6b17d1f2e12c4247f8bce6e563a440f277037d812deb33a0f4a13945d898c296", 16)
	curveGy, _ = new(big.Int).SetString("4fe342e2fe1a7f9b8ee7eb4a7c0f9e162bce33576b315ececbb6406837bf51f5", 16)
)

// fieldLen is the byte length of a coordinate or a scalar on this curve.
const fieldLen = 32

type point struct {
	x, y *big.Int
	inf  bool
}

var infinity = point{inf: true}

func generator() point { return point{x: new(big.Int).Set(curveGx), y: new(big.Int).Set(curveGy)} }

func (p point) equal(q point) bool {
	if p.inf || q.inf {
		return p.inf == q.inf
	}
	return p.x.Cmp(q.x) == 0 && p.y.Cmp(q.y) == 0
}

func mod(v *big.Int) *big.Int { return v.Mod(v, curveP) }

func (p point) add(q point) point {
	switch {
	case p.inf:
		return q
	case q.inf:
		return p
	case p.x.Cmp(q.x) == 0:
		if p.y.Cmp(q.y) == 0 && p.y.Sign() != 0 {
			return p.double()
		}
		return infinity
	}
	// lambda = (qy - py) / (qx - px)
	num := mod(new(big.Int).Sub(q.y, p.y))
	den := new(big.Int).ModInverse(mod(new(big.Int).Sub(q.x, p.x)), curveP)
	lambda := mod(num.Mul(num, den))
	x := new(big.Int).Mul(lambda, lambda)
	x.Sub(x, p.x).Sub(x, q.x)
	mod(x)
	y := new(big.Int).Sub(p.x, x)
	y.Mul(y, lambda).Sub(y, p.y)
	return point{x: x, y: mod(y)}
}

func (p point) double() point {
	if p.inf || p.y.Sign() == 0 {
		return infinity
	}
	// lambda = (3x^2 - 3) / 2y, since a = -3
	num := new(big.Int).Mul(p.x, p.x)
	num.Mul(num, big.NewInt(3)).Sub(num, big.NewInt(3))
	den := new(big.Int).ModInverse(mod(new(big.Int).Lsh(p.y, 1)), curveP)
	lambda := mod(num.Mul(mod(num), den))
	x := new(big.Int).Mul(lambda, lambda)
	x.Sub(x, new(big.Int).Lsh(p.x, 1))
	mod(x)
	y := new(big.Int).Sub(p.x, x)
	y.Mul(y, lambda).Sub(y, p.y)
	return point{x: x, y: mod(y)}
}

// mul is k*P for 0 <= k; every scalar EcJpake multiplies by is already reduced
// modulo the order.
func (p point) mul(k *big.Int) point {
	result := infinity
	for i := k.BitLen() - 1; i >= 0; i-- {
		result = result.double()
		if k.Bit(i) == 1 {
			result = result.add(p)
		}
	}
	return result
}

// onCurve checks y^2 = x^3 - 3x + b with both coordinates reduced.
func onCurve(x, y *big.Int) bool {
	if x.Sign() < 0 || x.Cmp(curveP) >= 0 || y.Sign() < 0 || y.Cmp(curveP) >= 0 {
		return false
	}
	return mod(new(big.Int).Mul(y, y)).Cmp(curveRHS(x)) == 0
}

func curveRHS(x *big.Int) *big.Int {
	rhs := new(big.Int).Mul(x, x)
	rhs.Mul(rhs, x)
	rhs.Sub(rhs, new(big.Int).Mul(x, big.NewInt(3)))
	rhs.Add(rhs, curveB)
	return mod(rhs)
}

// encode is Bouncy Castle's ECPoint.getEncoded(false): 0x04 || X || Y, or a
// single 0x00 for the point at infinity.
func (p point) encode() []byte {
	if p.inf {
		return []byte{0}
	}
	out := make([]byte, 1+2*fieldLen)
	out[0] = 4
	p.x.FillBytes(out[1 : 1+fieldLen])
	p.y.FillBytes(out[1+fieldLen:])
	return out
}

var errInvalidPoint = errors.New("invalid point encoding")

// decodePoint is ECCurve.decodePoint: uncompressed, compressed or infinity,
// and the point must lie on the curve.
func decodePoint(b []byte) (point, error) {
	if len(b) == 0 {
		return point{}, errInvalidPoint
	}
	switch b[0] {
	case 0:
		if len(b) != 1 {
			return point{}, errInvalidPoint
		}
		return infinity, nil
	case 2, 3:
		if len(b) != 1+fieldLen {
			return point{}, errInvalidPoint
		}
		x := new(big.Int).SetBytes(b[1:])
		if x.Cmp(curveP) >= 0 {
			return point{}, errInvalidPoint
		}
		// p = 3 mod 4, so a square root is rhs^((p+1)/4).
		exp := new(big.Int).Add(curveP, big.NewInt(1))
		exp.Rsh(exp, 2)
		y := new(big.Int).Exp(curveRHS(x), exp, curveP)
		if !onCurve(x, y) {
			return point{}, errInvalidPoint
		}
		if y.Bit(0) != uint(b[0]&1) {
			y.Sub(curveP, y)
		}
		return point{x: x, y: y}, nil
	case 4:
		if len(b) != 1+2*fieldLen {
			return point{}, errInvalidPoint
		}
		x := new(big.Int).SetBytes(b[1 : 1+fieldLen])
		y := new(big.Int).SetBytes(b[1+fieldLen:])
		if !onCurve(x, y) {
			return point{}, errInvalidPoint
		}
		return point{x: x, y: y}, nil
	}
	return point{}, errInvalidPoint
}
