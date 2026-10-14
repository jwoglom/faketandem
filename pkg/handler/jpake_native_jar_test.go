package handler

import (
	"bytes"
	"encoding/hex"
	"errors"
	"os"
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
)

// These hold the native EC-JPAKE to pumpX2's in both directions: pumpX2's own
// client pairs with the Go pump side, and the Go client pairs with pumpX2's
// jpake-server. Gated on FAKETANDEM_TEST_CLIPARSER_JAR like the other jar tests.

func requireCliparserJar(t *testing.T) string {
	t.Helper()
	jarPath := os.Getenv("FAKETANDEM_TEST_CLIPARSER_JAR")
	if jarPath == "" {
		t.Skip("FAKETANDEM_TEST_CLIPARSER_JAR not set, skipping the cross-check against pumpX2's JPAKE")
	}
	return jarPath
}

func TestJPAKEAuthenticator_PairsWithPumpX2Client(t *testing.T) {
	jarPath := requireCliparserJar(t)
	// Each pairing draws fresh keys; a few dozen catch an encoding that only
	// breaks for some scalars (a short ZKP scalar breaks about 1 in 256 rounds).
	for i := 0; i < 24; i++ {
		if err := driveJPAKEClient(jarPath, "123456", pumpx2.NewNativeBridge(), NewJPAKEAuthenticator("123456")); err != nil {
			t.Fatalf("pairing %d: %v", i, err)
		}
	}
}

func TestJPAKEAuthenticator_RejectsPumpX2ClientWithWrongCode(t *testing.T) {
	jarPath := requireCliparserJar(t)
	err := driveJPAKEClient(jarPath, "111111", pumpx2.NewNativeBridge(), NewJPAKEAuthenticator("123456"))
	if !errors.Is(err, ErrJPAKEHandshakeFailed) {
		t.Fatalf("wrong pairing code: got %v, want ErrJPAKEHandshakeFailed", err)
	}
}

// withRawPackets gives a request the client's own fragments, which is how
// PumpX2JPAKEAuthenticator forwards it to jpake-server.
func withRawPackets(t *testing.T, bridge *pumpx2.Bridge, req map[string]interface{}, params map[string]interface{}) map[string]interface{} {
	t.Helper()
	name := req["messageName"].(string)
	encoded, err := bridge.EncodeMessage(0, name, params)
	if err != nil {
		t.Fatalf("encode %s: %v", name, err)
	}
	parsed, err := bridge.ParseMessage(bluetooth.CharAuthorization, encoded.Packets)
	if err != nil || parsed.MessageType != name {
		t.Fatalf("%s did not parse back: %v %v", name, parsed, err)
	}
	req["rawPacketsHex"] = encoded.Packets
	return req
}

func challengeParams(req map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"appInstanceId": 0, "centralChallenge": req["centralChallenge"]}
}

func TestGoJPAKEClient_PairsWithPumpX2Server(t *testing.T) {
	jarPath := requireCliparserJar(t)
	bridge := pumpx2.NewNativeBridge()

	for i := 0; i < 8; i++ {
		auth := NewPumpX2JPAKEAuthenticator("123456", bridge, "", "jar", "", "java", jarPath)
		client := newJPAKETestClient(t, "123456")

		req1a := client.request1a()
		resp1a := mustRound(t, auth, 1, withRawPackets(t, bridge, req1a, challengeParams(req1a)))
		req1b := client.request1b(resp1a)
		resp1b := mustRound(t, auth, 1, withRawPackets(t, bridge, req1b, challengeParams(req1b)))
		req2 := client.request2(resp1b)
		resp2 := mustRound(t, auth, 2, withRawPackets(t, bridge, req2, challengeParams(req2)))
		req3 := client.request3(resp2)
		resp3 := mustRound(t, auth, 3, withRawPackets(t, bridge, req3, map[string]interface{}{"cargo": "0000"}))
		req4 := client.request4(resp3)
		resp4 := mustRound(t, auth, 4, withRawPackets(t, bridge, req4, map[string]interface{}{
			"appInstanceId": 0, "nonce": req4["nonce"], "reserved": req4["reserved"], "hashDigest": req4["hashDigest"],
		}))

		if !client.confirms(resp4) {
			t.Fatalf("pairing %d: the Go client rejected jpake-server's key confirmation", i)
		}
		secret, err := auth.GetLongTermSecret()
		if err != nil || !bytes.Equal(secret, client.derivedSecret) {
			t.Fatalf("pairing %d: jpake-server derived %s, the Go client %s (%v)",
				i, hex.EncodeToString(secret), hex.EncodeToString(client.derivedSecret), err)
		}
		_ = auth.Close()
	}
}
