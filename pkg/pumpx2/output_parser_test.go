package pumpx2

import (
	"testing"
)

// realJpake1aRawFragments are the actual raw BLE fragments captured from a real
// Tandem Mobi + official app pairing attempt (Authorization characteristic,
// txId=4), used to guard against a regression of the bug where ParseMessage
// silently produced an empty MessageType/opcode=0 for every message.
var realJpake1aRawFragments = []string{
	"09042004a70000410477521493da112577faa707",
	"0804c9c92a68e4b40cc46df17b306f52f32631af",
	"0704cd88d27a74f8ba9401e9aea18bcdb6f2c678",
	"06043cd475269208b03b9c7fa5c7a342eacaed41",
	"050404ec21fa73c0b5c2984d842a22a2db1df426",
	"0404a2793811949552a69108f8f10aad6c8f8c22",
	"0304cc3c9443848a1833f425b6cbef4658b2a86d",
	"02049b162b0b6c645e1d2993d920c143c44c4b68",
	"01047dd833cb7888682f66da86e0f0eb0b3abb59",
	"0004135c704fbbd824ecc9aa",
}

func TestOpcodeAndTxIDFromFirstFragment(t *testing.T) {
	opcode, txID, ok := opcodeAndTxIDFromFirstFragment(realJpake1aRawFragments)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if opcode != 32 {
		t.Errorf("expected opcode=32 (Jpake1aRequest), got %d", opcode)
	}
	if txID != 4 {
		t.Errorf("expected txID=4, got %d", txID)
	}
}

func TestOpcodeAndTxIDFromFirstFragment_Empty(t *testing.T) {
	if _, _, ok := opcodeAndTxIDFromFirstFragment(nil); ok {
		t.Error("expected ok=false for empty input")
	}
}

func TestParseCliparserOutput_RealJpake1aRequest(t *testing.T) {
	// Real cliparser "parse" command stdout for the fragments above. Note the
	// "Unable to parse: ..." prefix -- this is pumpX2's own parseOpcode() noise
	// (it naively hex-decodes the whole space-joined argument and always fails
	// for multi-fragment input), which must be ignored rather than break parsing.
	output := "Unable to parse: org.apache.commons.codec.DecoderException: Odd length for hex string\t" +
		"Jpake1aRequest[appInstanceId=0,centralChallenge={65,4,119,82,-116},cargo={0,0,65,4,119,82,-116}]"

	name, cargo := parseCliparserOutput(output)
	if name != "Jpake1aRequest" {
		t.Errorf("expected message name Jpake1aRequest, got %q", name)
	}
	if cargo["appInstanceId"] != 0 {
		t.Errorf("expected appInstanceId=0, got %v", cargo["appInstanceId"])
	}
	if cargo["centralChallenge"] != "410477528c" { // {65,4,119,82,-116} as hex
		t.Errorf("expected centralChallenge hex, got %v", cargo["centralChallenge"])
	}
}

func TestParseCliparserOutput_NoFields(t *testing.T) {
	name, cargo := parseCliparserOutput("54\tApiVersionRequest[]")
	if name != "ApiVersionRequest" {
		t.Errorf("expected ApiVersionRequest, got %q", name)
	}
	if len(cargo) != 0 {
		t.Errorf("expected empty cargo, got %v", cargo)
	}
}

func TestParseFieldValue(t *testing.T) {
	cases := []struct {
		in   string
		want interface{}
	}{
		{"0", 0},
		{"true", true},
		{"false", false},
		{"hello", "hello"},
		{"{1,2,3}", "010203"},
		{"{-1}", "ff"},
		{"{}", ""},
	}
	for _, c := range cases {
		got := parseFieldValue(c.in)
		if got != c.want {
			t.Errorf("parseFieldValue(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
