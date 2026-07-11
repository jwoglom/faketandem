package handler

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
)

// TestEncodeClientRequest_ExcludesCargoField guards against a regression where
// encodeClientRequest passed cliparser's own "cargo" field (present on every
// parsed message's output, redundant with the message's other named fields)
// through to EncodeMessage. cliparser's "encode" command picks a constructor by
// matching parameter *count*, so the extra key made it fail to find one with
// "no constructor was found with 3 parameters" for any message re-encoded this
// way -- confirmed against a real cliparser jar. Skipped unless
// FAKETANDEM_TEST_CLIPARSER_JAR is set, since CI has no jar available.
func TestEncodeClientRequest_ExcludesCargoField(t *testing.T) {
	jarPath := os.Getenv("FAKETANDEM_TEST_CLIPARSER_JAR")
	if jarPath == "" {
		t.Skip("FAKETANDEM_TEST_CLIPARSER_JAR not set, skipping real jar integration test")
	}

	bridge, err := pumpx2.NewBridge("", "jar", "", "java", jarPath)
	if err != nil {
		t.Fatalf("failed to create bridge: %v", err)
	}

	auth := NewPumpX2JPAKEAuthenticator("123456", bridge, "", "jar", "", "java", jarPath)

	// Real centralChallenge bytes captured from an actual Jpake1aRequest, plus
	// the redundant "cargo" field that a real ParseMessage call would also
	// produce -- encodeClientRequest must strip it before calling EncodeMessage.
	requestData := map[string]interface{}{
		"messageName":      "Jpake1aRequest",
		"appInstanceId":    0,
		"centralChallenge": "41045483658e8ea056f5b4d1454c13740db3a9712830938ea074fb0096489f4d5a8a16fa09767adcbdc6e8f74550d91c5ebe9fa3a18f91c2e73d12e182a2cb60a64f41049da3799b6ba274f3a83ee4b8b4e456cd262292db6dd35f62b91843e1e418700c1a97be5d09e26bd5a11956ee6f4819c09f71f60522e1418aa0a9e6afb07390512011b80880ed77972d4435cdfb223a7f30c54bff805c1308796e36f5b468e62c1f",
		"cargo":            "000041045483658e8ea056f5b4d1454c13740db3a9712830938ea074fb0096489f4d5a8a16fa09767adcbdc6e8f74550d91c5ebe9fa3a18f91c2e73d12e182a2cb60a64f41049da3799b6ba274f3a83ee4b8b4e456cd262292db6dd35f62b91843e1e418700c1a97be5d09e26bd5a11956ee6f4819c09f71f60522e1418aa0a9e6afb07390512011b80880ed77972d4435cdfb223a7f30c54bff805c1308796e36f5b468e62c1f",
	}

	result := auth.encodeClientRequest(requestData)
	if result == "00" {
		t.Fatal("encodeClientRequest fell back to the \"00\" placeholder -- encoding failed (see logs)")
	}

	// A real round-trip re-encode of the exact fragments the phone sent should
	// reproduce the exact same bytes, other than txID (encodeClientRequest always
	// re-encodes with txID=0, since jpake-server doesn't validate it).
	const expectedFirstFragment = "09002000a7000041045483658e8ea056f5b4d145"
	if len(result) < len(expectedFirstFragment) || result[:len(expectedFirstFragment)] != expectedFirstFragment {
		t.Errorf("expected re-encoded output to start with %q, got %q", expectedFirstFragment, result)
	}
}

// TestConvertServerResponseToParams_Jpake1aResponse guards against a regression
// where jpake-server's own response envelope -- the full JSON dumped after
// "JPAKE_1A: " for its own display/debugging, shaped as
// {messageName, txId, messageParams, characteristicName, characteristic, packets}
// -- was passed through to bridge.EncodeMessage wholesale instead of being
// converted to the flat {appInstanceId, centralChallengeHash} shape
// Jpake1aResponse's real 2-arg constructor needs. Passing the 6-key envelope
// made cliparser's "encode" (which picks a constructor by matching parameter
// count) fail with "no constructor was found with 6 parameters" -- confirmed
// against a real cliparser jar using an actual captured JPAKE_1A envelope.
func TestConvertServerResponseToParams_Jpake1aResponse(t *testing.T) {
	const envelopeJSON = `{"messageName":"Jpake1aResponse","txId":"0","messageParams":[0,[65,4,-73,-44,-10,-109,-70,-31,65,118,56,51,-121,-38,68,-85,69,33,117,-9,-17,-119,0,-77,88,-105,100,71,111,-30,-67,51,-104,65,-42,41,-42,105,-120,81,-46,38,-11,-104,20,59,70,-99,103,-78,-127,97,50,-124,8,7,122,-107,-56,82,-14,110,31,-11,-87,-83]],"characteristicName":"AUTHORIZATION","packets":["09002100a700004104b7d4f693bae14176383387"],"characteristic":"7b83fff9-9f77-4e5c-8064-aae2c24838b9"}`

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(envelopeJSON), &envelope); err != nil {
		t.Fatalf("failed to parse test envelope: %v", err)
	}

	params, err := convertServerResponseToParams(envelope)
	if err != nil {
		t.Fatalf("convertServerResponseToParams failed: %v", err)
	}

	if params["appInstanceId"] != float64(0) {
		t.Errorf("expected appInstanceId=0, got %v", params["appInstanceId"])
	}

	const expectedHashPrefix = "4104b7d4f693bae14176383387"
	hash, ok := params["centralChallengeHash"].(string)
	if !ok || len(hash) < len(expectedHashPrefix) || hash[:len(expectedHashPrefix)] != expectedHashPrefix {
		t.Errorf("expected centralChallengeHash to start with %q, got %v", expectedHashPrefix, params["centralChallengeHash"])
	}

	if _, hasMessageName := params["messageName"]; hasMessageName {
		t.Error("converted params should not contain the envelope's own messageName key")
	}
	if _, hasPackets := params["packets"]; hasPackets {
		t.Error("converted params should not contain the envelope's own packets key")
	}
}

// TestConvertServerResponseToParams_UnknownMessage guards against silently
// returning an empty/nil params map for a response type we don't have a
// constructor-parameter mapping for.
func TestConvertServerResponseToParams_UnknownMessage(t *testing.T) {
	envelope := map[string]interface{}{
		"messageName":  "SomeFutureResponse",
		"messageParams": []interface{}{0},
	}

	if _, err := convertServerResponseToParams(envelope); err == nil {
		t.Error("expected an error for an unmapped response message name")
	}
}
