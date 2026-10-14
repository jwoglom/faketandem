package jpake

import (
	"bytes"
	"crypto/elliptic"
	"math/big"
	"math/rand"
	"testing"
)

// TestCurveMatchesStdlib holds the affine arithmetic to crypto/elliptic's.
func TestCurveMatchesStdlib(t *testing.T) {
	curve := elliptic.P256()
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 20; i++ {
		k := new(big.Int).Rand(rng, curveN)
		wantX, wantY := curve.ScalarBaseMult(k.Bytes()) //nolint:staticcheck // the reference implementation
		got := generator().mul(k)
		if k.Sign() == 0 {
			continue
		}
		if got.x.Cmp(wantX) != 0 || got.y.Cmp(wantY) != 0 {
			t.Fatalf("k=%x: got (%x,%x), want (%x,%x)", k, got.x, got.y, wantX, wantY)
		}
		j := new(big.Int).Rand(rng, curveN)
		jx, jy := curve.ScalarBaseMult(j.Bytes())     //nolint:staticcheck // the reference implementation
		sumX, sumY := curve.Add(wantX, wantY, jx, jy) //nolint:staticcheck // the reference implementation
		sum := got.add(generator().mul(j))
		if sum.x.Cmp(sumX) != 0 || sum.y.Cmp(sumY) != 0 {
			t.Fatalf("add mismatch at %d", i)
		}
		dec, err := decodePoint(got.encode())
		if err != nil || !dec.equal(got) {
			t.Fatalf("encode/decode round trip failed: %v", err)
		}
		compressed := append([]byte{2 + byte(got.y.Bit(0))}, got.encode()[1:33]...)
		if dec, err := decodePoint(compressed); err != nil || !dec.equal(got) {
			t.Fatalf("compressed decode failed: %v", err)
		}
	}
}

func TestClientAndServerDeriveTheSameSecret(t *testing.T) {
	secret := PairingCodeToBytes("123456")
	client := New(Client, secret, nil)
	server := New(Server, secret, nil)

	c1, _ := client.Round1()
	s1, _ := server.Round1()
	if len(c1) != 330 || len(s1) != 330 {
		t.Fatalf("round 1 lengths %d/%d, want 330", len(c1), len(s1))
	}
	if _, err := server.ReadRound1(c1); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ReadRound1(s1); err != nil {
		t.Fatal(err)
	}
	c2, _ := client.Round2()
	s2, _ := server.Round2()
	if len(c2) != 165 || len(s2) != 168 {
		t.Fatalf("round 2 lengths %d/%d, want 165/168", len(c2), len(s2))
	}
	if _, err := server.ReadRound2(c2); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ReadRound2(s2); err != nil {
		t.Fatal(err)
	}
	a, _ := client.DeriveSecret()
	b, _ := server.DeriveSecret()
	if !bytes.Equal(a, b) {
		t.Fatalf("secrets differ: %x vs %x", a, b)
	}
}

func TestWrongPairingCodeDerivesADifferentSecret(t *testing.T) {
	client := New(Client, PairingCodeToBytes("123456"), nil)
	server := New(Server, PairingCodeToBytes("654321"), nil)
	c1, _ := client.Round1()
	s1, _ := server.Round1()
	if _, err := server.ReadRound1(c1); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ReadRound1(s1); err != nil {
		t.Fatal(err)
	}
	c2, _ := client.Round2()
	s2, _ := server.Round2()
	if _, err := server.ReadRound2(c2); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ReadRound2(s2); err != nil {
		t.Fatal(err)
	}
	a, _ := client.DeriveSecret()
	b, _ := server.DeriveSecret()
	if bytes.Equal(a, b) {
		t.Fatal("different pairing codes derived the same secret")
	}
}

func TestTamperedProofIsRejected(t *testing.T) {
	client := New(Client, PairingCodeToBytes("123456"), nil)
	server := New(Server, PairingCodeToBytes("123456"), nil)
	c1, _ := client.Round1()
	tampered := append([]byte{}, c1...)
	tampered[100] ^= 1
	if _, err := server.ReadRound1(tampered); err == nil {
		t.Fatal("a tampered round 1 was accepted")
	}
}
