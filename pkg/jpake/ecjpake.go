// Package jpake implements the EC-JPAKE pairing the Tandem pump runs, as a
// port of pumpX2's io.particle.crypto.EcJpake (itself mbed TLS's EC-JPAKE, the
// Thread specification's variant) and of the session-key rounds pumpX2's
// jpake-server adds on top.
package jpake

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
)

// Role is which side of the exchange this peer is.
type Role int

// The two peers. The pump is the server.
const (
	Client Role = iota
	Server
)

var (
	clientID = []byte("client")
	serverID = []byte("server")
)

// curveID is the TLS NamedCurve id of secp256r1 (RFC 4492 5.1.1), which the
// server's round 2 names ahead of its point.
const curveID = 23

// ErrInvalidState is a round taken out of order.
var ErrInvalidState = errors.New("invalid protocol state")

// EcJpake is one side of an EC-JPAKE exchange.
type EcJpake struct {
	role         Role
	myID, peerID []byte
	s            *big.Int
	rand         io.Reader

	xm1, xm2           *big.Int
	Xm1, Xm2           point
	Xp1, Xp2, Xp       point
	hasPeerRound1      bool
	hasPeerRound2      bool
	myRound1, myRound2 []byte
	derivedSecret      []byte
}

// New starts an exchange keyed on secret, drawing randomness from random
// (crypto/rand when nil).
func New(role Role, secret []byte, random io.Reader) *EcJpake {
	if random == nil {
		random = rand.Reader
	}
	j := &EcJpake{role: role, s: new(big.Int).SetBytes(secret), rand: random}
	if role == Client {
		j.myID, j.peerID = clientID, serverID
	} else {
		j.myID, j.peerID = serverID, clientID
	}
	return j
}

// Round1 is this peer's first-round message: two public keys, each with a
// zero-knowledge proof of its private key. It is 330 bytes on P-256.
func (j *EcJpake) Round1() ([]byte, error) {
	if j.myRound1 != nil {
		return j.myRound1, nil
	}
	var out bytes.Buffer
	g := generator()
	var err error
	if j.xm1, j.Xm1, err = j.genKeyPair(g); err != nil {
		return nil, err
	}
	writePoint(&out, j.Xm1)
	if err := j.writeZkp(&out, g, j.xm1, j.Xm1); err != nil {
		return nil, err
	}
	if j.xm2, j.Xm2, err = j.genKeyPair(g); err != nil {
		return nil, err
	}
	writePoint(&out, j.Xm2)
	if err := j.writeZkp(&out, g, j.xm2, j.Xm2); err != nil {
		return nil, err
	}
	j.myRound1 = out.Bytes()
	return j.myRound1, nil
}

// ReadRound1 takes the peer's first-round message and verifies both proofs.
// It returns how many bytes it read; trailing bytes are not an error.
func (j *EcJpake) ReadRound1(data []byte) (int, error) {
	if j.hasPeerRound1 {
		return 0, ErrInvalidState
	}
	in := bytes.NewReader(data)
	g := generator()
	var err error
	if j.Xp1, err = readPoint(in); err != nil {
		return 0, err
	}
	if err := j.readZkp(in, g, j.Xp1); err != nil {
		return 0, err
	}
	if j.Xp2, err = readPoint(in); err != nil {
		return 0, err
	}
	if err := j.readZkp(in, g, j.Xp2); err != nil {
		return 0, err
	}
	j.hasPeerRound1 = true
	return len(data) - in.Len(), nil
}

// Round2 is this peer's second-round message: 168 bytes for the server, whose
// message starts with the curve id, and 165 for the client.
func (j *EcJpake) Round2() ([]byte, error) {
	if j.myRound2 != nil {
		return j.myRound2, nil
	}
	if !j.hasPeerRound1 || j.myRound1 == nil {
		return nil, ErrInvalidState
	}
	var out bytes.Buffer
	g := j.Xp1.add(j.Xp2).add(j.Xm1)
	xm, err := j.mulSecret(j.xm2, false)
	if err != nil {
		return nil, err
	}
	xmPoint := g.mul(xm)
	if j.role == Server {
		out.Write([]byte{3, 0, curveID})
	}
	writePoint(&out, xmPoint)
	if err := j.writeZkp(&out, g, xm, xmPoint); err != nil {
		return nil, err
	}
	j.myRound2 = out.Bytes()
	return j.myRound2, nil
}

// ReadRound2 takes the peer's second-round message and verifies its proof.
func (j *EcJpake) ReadRound2(data []byte) (int, error) {
	if !j.hasPeerRound1 || j.myRound1 == nil || j.hasPeerRound2 {
		return 0, ErrInvalidState
	}
	in := bytes.NewReader(data)
	if j.role == Client {
		if err := readCurveID(in); err != nil {
			return 0, err
		}
	}
	g := j.Xm1.add(j.Xm2).add(j.Xp1)
	var err error
	if j.Xp, err = readPoint(in); err != nil {
		return 0, err
	}
	if err := j.readZkp(in, g, j.Xp); err != nil {
		return 0, err
	}
	j.hasPeerRound2 = true
	return len(data) - in.Len(), nil
}

// DeriveSecret is the shared secret: SHA-256 of the x coordinate of the
// shared point, written at the full field length.
func (j *EcJpake) DeriveSecret() ([]byte, error) {
	if j.derivedSecret != nil {
		return j.derivedSecret, nil
	}
	if !j.hasPeerRound2 {
		return nil, ErrInvalidState
	}
	xm2s, err := j.mulSecret(j.xm2, true)
	if err != nil {
		return nil, err
	}
	k := j.Xp.add(j.Xp2.mul(xm2s)).mul(j.xm2)
	if k.inf {
		return nil, errors.New("shared point is the point at infinity")
	}
	x := make([]byte, fieldLen)
	k.x.FillBytes(x)
	sum := sha256.Sum256(x)
	j.derivedSecret = sum[:]
	return j.derivedSecret, nil
}

func (j *EcJpake) readZkp(in *bytes.Reader, g, x point) error {
	v, err := readPoint(in)
	if err != nil {
		return err
	}
	r, err := readNum(in)
	if err != nil {
		return err
	}
	h := zkpHash(g, v, x, j.peerID)
	if !g.mul(new(big.Int).Mod(r, curveN)).add(x.mul(h)).equal(v) {
		return errors.New("zero-knowledge proof validation failed")
	}
	return nil
}

func (j *EcJpake) writeZkp(out *bytes.Buffer, g point, x *big.Int, xPoint point) error {
	v, vPoint, err := j.genKeyPair(g)
	if err != nil {
		return err
	}
	h := zkpHash(g, vPoint, xPoint, j.myID)
	r := new(big.Int).Mul(x, h)
	r.Sub(v, r).Mod(r, curveN)
	writePoint(out, vPoint)
	writeNum(out, r)
	return nil
}

// zkpHash is SHA-256 over G, V and X (each length-prefixed) and the signer's
// id, reduced modulo the order.
func zkpHash(g, v, x point, id []byte) *big.Int {
	var buf bytes.Buffer
	for _, p := range []point{g, v, x} {
		enc := p.encode()
		_ = binary.Write(&buf, binary.BigEndian, uint32(len(enc))) // Safe to ignore: bytes.Buffer writes cannot fail.
		buf.Write(enc)
	}
	_ = binary.Write(&buf, binary.BigEndian, uint32(len(id))) // Safe to ignore: bytes.Buffer writes cannot fail.
	buf.Write(id)
	sum := sha256.Sum256(buf.Bytes())
	return new(big.Int).Mod(new(big.Int).SetBytes(sum[:]), curveN)
}

// mulSecret is x * (b*n + s) mod n, negated when asked, for a random 128-bit
// b that blinds the secret.
func (j *EcJpake) mulSecret(x *big.Int, negate bool) (*big.Int, error) {
	blind := make([]byte, 16)
	if _, err := io.ReadFull(j.rand, blind); err != nil {
		return nil, err
	}
	b := new(big.Int).SetBytes(blind)
	b.Mul(b, curveN).Add(b, j.s)
	r := new(big.Int).Mul(x, b)
	if negate {
		r.Neg(r)
	}
	return r.Mod(r, curveN), nil
}

// genKeyPair draws a private key uniformly from [1, n-1].
func (j *EcJpake) genKeyPair(g point) (*big.Int, point, error) {
	max := new(big.Int).Sub(curveN, big.NewInt(1))
	k, err := rand.Int(j.rand, max)
	if err != nil {
		return nil, point{}, err
	}
	k.Add(k, big.NewInt(1))
	return k, g.mul(k), nil
}

func writePoint(out *bytes.Buffer, p point) {
	enc := p.encode()
	out.WriteByte(byte(len(enc)))
	out.Write(enc)
}

// writeNum writes a scalar zero-padded to the full order length, as pumpX2
// does, so a round is always its nominal size; a peer reads it as an integer
// either way.
func writeNum(out *bytes.Buffer, v *big.Int) {
	enc := make([]byte, fieldLen)
	v.FillBytes(enc)
	out.WriteByte(byte(len(enc)))
	out.Write(enc)
}

func readField(in *bytes.Reader) ([]byte, error) {
	n, err := in.ReadByte()
	if err != nil {
		return nil, errors.New("unexpected end of stream")
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(in, b); err != nil {
		return nil, errors.New("unexpected end of stream")
	}
	return b, nil
}

func readPoint(in *bytes.Reader) (point, error) {
	b, err := readField(in)
	if err != nil {
		return point{}, err
	}
	return decodePoint(b)
}

func readNum(in *bytes.Reader) (*big.Int, error) {
	b, err := readField(in)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(b), nil
}

func readCurveID(in *bytes.Reader) error {
	kind, err := in.ReadByte()
	if err != nil {
		return errors.New("unexpected end of stream")
	}
	if kind != 3 {
		return errors.New("invalid message")
	}
	var id [2]byte
	if _, err := io.ReadFull(in, id[:]); err != nil {
		return errors.New("unexpected end of stream")
	}
	if got := binary.BigEndian.Uint16(id[:]); got != curveID {
		return fmt.Errorf("unexpected curve type %d", got)
	}
	return nil
}
