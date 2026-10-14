package handler

import (
	"crypto/hmac"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"sync"

	"github.com/jwoglom/faketandem/pkg/jpake"

	log "github.com/sirupsen/logrus"
)

// jpakeRound1HalfLength is the size of each half of a round-1 message, which
// the protocol carries in two messages (1a and 1b).
const jpakeRound1HalfLength = 165

// JPAKE handshake steps, in the order the client drives them.
const (
	jpakeAwaiting1a = iota
	jpakeAwaiting1b
	jpakeAwaiting2
	jpakeAwaiting3
	jpakeAwaiting4
	jpakeComplete
)

// JPAKEAuthenticator runs the pump's side of the EC-JPAKE pairing in Go,
// doing what pumpX2's jpake-server does: rounds 1a/1b/2 of EC-JPAKE against
// the pairing code, then the session-key exchange and key confirmation of
// rounds 3/4. A handshake fails at the same round jpake-server's does, so a
// driver sees the same thing in either mode.
type JPAKEAuthenticator struct {
	pairingCode string
	random      io.Reader

	step          int
	ecjpake       *jpake.EcJpake
	clientRound1a []byte
	serverRound1  []byte
	serverRound2  []byte
	derivedSecret []byte
	serverNonce3  []byte
	// round2Err is a client round 2 that did not verify. jpake-server answers
	// round 2 before it reads the client's, so the handshake only fails at
	// round 3; keeping the error until then reproduces that.
	round2Err error

	mutex sync.Mutex
}

// NewJPAKEAuthenticator creates the pump side of a fresh pairing against pairingCode.
func NewJPAKEAuthenticator(pairingCode string) *JPAKEAuthenticator {
	return newJPAKEAuthenticatorWithRandom(pairingCode, rand.Reader)
}

func newJPAKEAuthenticatorWithRandom(pairingCode string, random io.Reader) *JPAKEAuthenticator {
	return &JPAKEAuthenticator{
		pairingCode: pairingCode,
		random:      random,
		ecjpake:     jpake.New(jpake.Server, jpake.PairingCodeToBytes(pairingCode), random),
	}
}

// ProcessRound answers one client JPAKE message. Round 1 carries both the 1a
// and the 1b message, told apart by the order they arrive in, as jpake-server
// does.
func (j *JPAKEAuthenticator) ProcessRound(round int, requestData map[string]interface{}) (map[string]interface{}, error) {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	messageName, _ := requestData["messageName"].(string)
	switch {
	case round == 1 && j.step == jpakeAwaiting1a:
		return j.round1a(messageName, requestData)
	case round == 1 && j.step == jpakeAwaiting1b:
		return j.round1b(messageName, requestData)
	case round == 2 && j.step == jpakeAwaiting2:
		return j.round2(requestData), nil
	case round == 3 && j.step == jpakeAwaiting3:
		return j.round3()
	case round == 4 && j.step == jpakeAwaiting4:
		return j.round4(requestData)
	}
	return nil, fmt.Errorf("%w: %s (round %d) arrived at step %d", ErrJPAKEHandshakeFailed, messageName, round, j.step)
}

func (j *JPAKEAuthenticator) round1a(messageName string, requestData map[string]interface{}) (map[string]interface{}, error) {
	if messageName != "" && messageName != "Jpake1aRequest" {
		return nil, fmt.Errorf("%w: expected Jpake1aRequest, got %s", ErrJPAKEHandshakeFailed, messageName)
	}
	challenge, err := cargoHexBytes(requestData, "centralChallenge")
	if err != nil {
		return nil, err
	}
	serverRound1, err := j.ecjpake.Round1()
	if err != nil {
		return nil, fmt.Errorf("%w: generating round 1: %v", ErrJPAKEHandshakeFailed, err)
	}
	j.clientRound1a = challenge
	j.serverRound1 = serverRound1
	j.step = jpakeAwaiting1b
	return jpakeChallengeParams(serverRound1[:jpakeRound1HalfLength]), nil
}

func (j *JPAKEAuthenticator) round1b(messageName string, requestData map[string]interface{}) (map[string]interface{}, error) {
	if messageName != "" && messageName != "Jpake1bRequest" {
		return nil, fmt.Errorf("%w: expected Jpake1bRequest, got %s", ErrJPAKEHandshakeFailed, messageName)
	}
	challenge, err := cargoHexBytes(requestData, "centralChallenge")
	if err != nil {
		return nil, err
	}
	clientRound1 := append(append([]byte{}, j.clientRound1a...), challenge...)
	if _, err := j.ecjpake.ReadRound1(clientRound1); err != nil {
		return nil, fmt.Errorf("%w: client round 1 did not verify: %v", ErrJPAKEHandshakeFailed, err)
	}
	serverRound2, err := j.ecjpake.Round2()
	if err != nil {
		return nil, fmt.Errorf("%w: generating round 2: %v", ErrJPAKEHandshakeFailed, err)
	}
	j.serverRound2 = serverRound2
	j.step = jpakeAwaiting2
	return jpakeChallengeParams(j.serverRound1[jpakeRound1HalfLength:]), nil
}

func (j *JPAKEAuthenticator) round2(requestData map[string]interface{}) map[string]interface{} {
	challenge, err := cargoHexBytes(requestData, "centralChallenge")
	if err == nil {
		if _, err = j.ecjpake.ReadRound2(challenge); err == nil {
			j.derivedSecret, err = j.ecjpake.DeriveSecret()
		}
	}
	if err != nil {
		j.round2Err = fmt.Errorf("%w: client round 2 did not verify: %v", ErrJPAKEHandshakeFailed, err)
		log.Warnf("JPAKE: %v; the handshake fails at round 3, where jpake-server's does", j.round2Err)
	}
	j.step = jpakeAwaiting3
	return jpakeChallengeParams(j.serverRound2)
}

func (j *JPAKEAuthenticator) round3() (map[string]interface{}, error) {
	if j.round2Err != nil {
		return nil, j.round2Err
	}
	nonce := make([]byte, 8)
	if _, err := io.ReadFull(j.random, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate server nonce: %w", err)
	}
	j.serverNonce3 = nonce
	j.step = jpakeAwaiting4
	return map[string]interface{}{
		"appInstanceId": 0,
		"nonce":         hex.EncodeToString(nonce),
		"reserved":      hex.EncodeToString(make([]byte, 8)),
	}, nil
}

func (j *JPAKEAuthenticator) round4(requestData map[string]interface{}) (map[string]interface{}, error) {
	clientNonce, err := cargoHexBytes(requestData, "nonce")
	if err != nil {
		return nil, err
	}
	clientDigest, err := cargoHexBytes(requestData, "hashDigest")
	if err != nil {
		return nil, err
	}
	if !hmac.Equal(jpake.KeyConfirmation(j.serverNonce3, j.derivedSecret, clientNonce), clientDigest) {
		return nil, fmt.Errorf("%w: client key confirmation did not verify (wrong pairing code?)", ErrJPAKEHandshakeFailed)
	}
	nonce := make([]byte, 8)
	if _, err := io.ReadFull(j.random, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate confirmation nonce: %w", err)
	}
	j.step = jpakeComplete
	log.Infof("JPAKE handshake complete; derived secret %s", hex.EncodeToString(j.derivedSecret))
	return map[string]interface{}{
		"appInstanceId": 0,
		"nonce":         hex.EncodeToString(nonce),
		"reserved":      hex.EncodeToString(make([]byte, 8)),
		"hashDigest":    hex.EncodeToString(jpake.KeyConfirmation(j.serverNonce3, j.derivedSecret, nonce)),
	}, nil
}

func jpakeChallengeParams(challenge []byte) map[string]interface{} {
	return map[string]interface{}{
		"appInstanceId":        0,
		"centralChallengeHash": hex.EncodeToString(challenge),
	}
}

// cargoHexBytes reads a byte[] field of a parsed request, which the parser
// renders as hex.
func cargoHexBytes(requestData map[string]interface{}, key string) ([]byte, error) {
	s, ok := requestData[key].(string)
	if !ok {
		return nil, fmt.Errorf("%w: request has no %s", ErrJPAKEHandshakeFailed, key)
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("%w: request %s is not hex: %v", ErrJPAKEHandshakeFailed, key, err)
	}
	return b, nil
}

// GetSharedSecret returns the post-pairing signing key in the convention the
// pumpX2 mode established: the ASCII bytes of the derived secret's hex.
func (j *JPAKEAuthenticator) GetSharedSecret() ([]byte, error) {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	if j.step != jpakeComplete {
		return nil, fmt.Errorf("JPAKE not complete, current step: %d", j.step)
	}
	return []byte(hex.EncodeToString(j.derivedSecret)), nil
}

// GetLongTermSecret returns the derived secret itself, which a later
// quick-pair reconnect resumes from.
func (j *JPAKEAuthenticator) GetLongTermSecret() ([]byte, error) {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	if j.step != jpakeComplete {
		return nil, fmt.Errorf("JPAKE not complete, current step: %d", j.step)
	}
	return j.derivedSecret, nil
}

// IsComplete reports whether key confirmation has succeeded.
func (j *JPAKEAuthenticator) IsComplete() bool {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	return j.step == jpakeComplete
}
