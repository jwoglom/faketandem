package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/jwoglom/faketandem/pkg/jpake"
)

// jpakeTestClient is the phone's side of the pairing, as pumpX2's
// JpakeAuthBuilder runs it, producing requests in the shape the parser gives
// the handler.
type jpakeTestClient struct {
	t             *testing.T
	ecjpake       *jpake.EcJpake
	round1        []byte
	serverRound1a []byte
	derivedSecret []byte
	serverNonce3  []byte
	clientNonce4  []byte
}

func newJPAKETestClient(t *testing.T, pairingCode string) *jpakeTestClient {
	return &jpakeTestClient{t: t, ecjpake: jpake.New(jpake.Client, jpake.PairingCodeToBytes(pairingCode), nil)}
}

func challengeRequest(name string, challenge []byte) map[string]interface{} {
	return map[string]interface{}{"messageName": name, "appInstanceId": 0, "centralChallenge": hex.EncodeToString(challenge)}
}

func (c *jpakeTestClient) request1a() map[string]interface{} {
	round1, err := c.ecjpake.Round1()
	if err != nil {
		c.t.Fatal(err)
	}
	c.round1 = round1
	return challengeRequest("Jpake1aRequest", round1[:165])
}

func (c *jpakeTestClient) request1b(resp1a map[string]interface{}) map[string]interface{} {
	c.serverRound1a = hexParam(c.t, resp1a, "centralChallengeHash")
	return challengeRequest("Jpake1bRequest", c.round1[165:])
}

func (c *jpakeTestClient) request2(resp1b map[string]interface{}) map[string]interface{} {
	serverRound1 := append(append([]byte{}, c.serverRound1a...), hexParam(c.t, resp1b, "centralChallengeHash")...)
	if _, err := c.ecjpake.ReadRound1(serverRound1); err != nil {
		c.t.Fatalf("client could not verify the pump's round 1: %v", err)
	}
	round2, err := c.ecjpake.Round2()
	if err != nil {
		c.t.Fatal(err)
	}
	return challengeRequest("Jpake2Request", round2)
}

func (c *jpakeTestClient) request3(resp2 map[string]interface{}) map[string]interface{} {
	if _, err := c.ecjpake.ReadRound2(hexParam(c.t, resp2, "centralChallengeHash")); err != nil {
		c.t.Fatalf("client could not verify the pump's round 2: %v", err)
	}
	secret, err := c.ecjpake.DeriveSecret()
	if err != nil {
		c.t.Fatal(err)
	}
	c.derivedSecret = secret
	return map[string]interface{}{"messageName": "Jpake3SessionKeyRequest", "challengeParam": 0}
}

func (c *jpakeTestClient) request4(resp3 map[string]interface{}) map[string]interface{} {
	c.serverNonce3 = hexParam(c.t, resp3, "nonce")
	c.clientNonce4 = make([]byte, 8)
	if _, err := rand.Read(c.clientNonce4); err != nil {
		c.t.Fatal(err)
	}
	return map[string]interface{}{
		"messageName": "Jpake4KeyConfirmationRequest",
		"nonce":       hex.EncodeToString(c.clientNonce4),
		"reserved":    hex.EncodeToString(make([]byte, 8)),
		"hashDigest":  hex.EncodeToString(jpake.KeyConfirmation(c.serverNonce3, c.derivedSecret, c.clientNonce4)),
	}
}

// confirms reports whether the pump's round 4 digest is the one the client expects.
func (c *jpakeTestClient) confirms(resp4 map[string]interface{}) bool {
	want := jpake.KeyConfirmation(c.serverNonce3, c.derivedSecret, hexParam(c.t, resp4, "nonce"))
	return bytes.Equal(want, hexParam(c.t, resp4, "hashDigest"))
}

func hexParam(t *testing.T, params map[string]interface{}, key string) []byte {
	t.Helper()
	s, ok := params[key].(string)
	if !ok {
		t.Fatalf("response has no %s: %v", key, params)
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("%s is not hex: %v", key, err)
	}
	return b
}

func mustRound(t *testing.T, auth JPAKEAuthenticatorInterface, round int, req map[string]interface{}) map[string]interface{} {
	t.Helper()
	resp, err := auth.ProcessRound(round, req)
	if err != nil {
		t.Fatalf("%s (round %d): %v", req["messageName"], round, err)
	}
	return resp
}

func TestJPAKEAuthenticator_FullPairing(t *testing.T) {
	auth := NewJPAKEAuthenticator("123456")
	client := newJPAKETestClient(t, "123456")

	resp1a := mustRound(t, auth, 1, client.request1a())
	resp1b := mustRound(t, auth, 1, client.request1b(resp1a))
	resp2 := mustRound(t, auth, 2, client.request2(resp1b))
	resp3 := mustRound(t, auth, 3, client.request3(resp2))
	resp4 := mustRound(t, auth, 4, client.request4(resp3))

	if !client.confirms(resp4) {
		t.Fatal("the client rejected the pump's key confirmation")
	}
	if len(hexParam(t, resp1a, "centralChallengeHash")) != 165 || len(hexParam(t, resp1b, "centralChallengeHash")) != 165 {
		t.Error("round 1 halves are not 165 bytes")
	}
	if len(hexParam(t, resp2, "centralChallengeHash")) != 168 {
		t.Error("the pump's round 2 is not 168 bytes")
	}
	if !auth.IsComplete() {
		t.Fatal("authenticator not complete after round 4")
	}
	longTerm, _ := auth.GetLongTermSecret()
	if !bytes.Equal(longTerm, client.derivedSecret) {
		t.Errorf("long-term secret %x, client derived %x", longTerm, client.derivedSecret)
	}
	// The signing key keeps the pumpX2 mode's convention: the hex string's bytes.
	shared, _ := auth.GetSharedSecret()
	if string(shared) != hex.EncodeToString(client.derivedSecret) {
		t.Errorf("shared secret %q, want the hex of %x", shared, client.derivedSecret)
	}
}

func TestJPAKEAuthenticator_WrongPairingCodeFailsAtKeyConfirmation(t *testing.T) {
	auth := NewJPAKEAuthenticator("123456")
	client := newJPAKETestClient(t, "111111")

	resp1a := mustRound(t, auth, 1, client.request1a())
	resp1b := mustRound(t, auth, 1, client.request1b(resp1a))
	resp2 := mustRound(t, auth, 2, client.request2(resp1b))
	resp3 := mustRound(t, auth, 3, client.request3(resp2))
	if _, err := auth.ProcessRound(4, client.request4(resp3)); !errors.Is(err, ErrJPAKEHandshakeFailed) {
		t.Fatalf("round 4 with the wrong code: got %v, want ErrJPAKEHandshakeFailed", err)
	}
	if auth.IsComplete() {
		t.Error("a rejected pairing is complete")
	}
}

func TestJPAKEAuthenticator_TamperedRound1FailsAt1b(t *testing.T) {
	auth := NewJPAKEAuthenticator("123456")
	client := newJPAKETestClient(t, "123456")

	resp1a := mustRound(t, auth, 1, client.request1a())
	req1b := client.request1b(resp1a)
	tampered := hexParamString(req1b["centralChallenge"])
	tampered[40] ^= 1
	req1b["centralChallenge"] = hex.EncodeToString(tampered)
	if _, err := auth.ProcessRound(1, req1b); !errors.Is(err, ErrJPAKEHandshakeFailed) {
		t.Fatalf("tampered 1b: got %v, want ErrJPAKEHandshakeFailed", err)
	}
}

func TestJPAKEAuthenticator_TamperedRound2FailsAtRound3(t *testing.T) {
	auth := NewJPAKEAuthenticator("123456")
	client := newJPAKETestClient(t, "123456")

	resp1a := mustRound(t, auth, 1, client.request1a())
	resp1b := mustRound(t, auth, 1, client.request1b(resp1a))
	req2 := client.request2(resp1b)
	tampered := hexParamString(req2["centralChallenge"])
	tampered[100] ^= 1
	req2["centralChallenge"] = hex.EncodeToString(tampered)
	// jpake-server answers round 2 before reading the client's, so the pump
	// still answers it and the handshake fails one round later.
	resp2 := mustRound(t, auth, 2, req2)
	if _, err := auth.ProcessRound(3, client.request3(resp2)); !errors.Is(err, ErrJPAKEHandshakeFailed) {
		t.Fatalf("round 3 after a bad round 2: got %v, want ErrJPAKEHandshakeFailed", err)
	}
}

func TestJPAKEAuthenticator_OutOfOrderRoundsFail(t *testing.T) {
	client := newJPAKETestClient(t, "123456")
	req1a := client.request1a()

	first1b := challengeRequest("Jpake1bRequest", make([]byte, 165))
	if _, err := NewJPAKEAuthenticator("123456").ProcessRound(1, first1b); !errors.Is(err, ErrJPAKEHandshakeFailed) {
		t.Errorf("1b first: got %v, want ErrJPAKEHandshakeFailed", err)
	}
	auth := NewJPAKEAuthenticator("123456")
	mustRound(t, auth, 1, req1a)
	if _, err := auth.ProcessRound(2, challengeRequest("Jpake2Request", make([]byte, 165))); !errors.Is(err, ErrJPAKEHandshakeFailed) {
		t.Errorf("round 2 before 1b: got %v, want ErrJPAKEHandshakeFailed", err)
	}
}

// A long-term secret from a native pairing is what a quick-pair reconnect
// resumes from.
func TestJPAKEAuthenticator_LongTermSecretResumesQuickPair(t *testing.T) {
	auth := NewJPAKEAuthenticator("123456")
	client := newJPAKETestClient(t, "123456")
	resp1a := mustRound(t, auth, 1, client.request1a())
	resp1b := mustRound(t, auth, 1, client.request1b(resp1a))
	resp2 := mustRound(t, auth, 2, client.request2(resp1b))
	resp3 := mustRound(t, auth, 3, client.request3(resp2))
	mustRound(t, auth, 4, client.request4(resp3))
	longTerm, _ := auth.GetLongTermSecret()

	quick := NewQuickReconnectJPAKEAuthenticator(longTerm)
	qResp3 := mustRound(t, quick, 3, map[string]interface{}{"messageName": "Jpake3SessionKeyRequest"})
	qResp4 := mustRound(t, quick, 4, client.request4(qResp3))
	if !client.confirms(qResp4) {
		t.Fatal("the client rejected the quick-pair key confirmation")
	}
}

func hexParamString(v interface{}) []byte {
	b, _ := hex.DecodeString(v.(string))
	return b
}
