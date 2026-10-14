package pumpx2

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

// These hold the bridge, which is what handlers see, to the jar's: the
// ParsedMessage a request becomes and the EncodedMessage a response becomes,
// over every parity case. pkg/pumpx2/messages checks the port beneath it.

func charType(t *testing.T, name string) bluetooth.CharacteristicType {
	t.Helper()
	if name == "" {
		return bluetooth.CharQualifyingEvents
	}
	c, ok := bluetooth.CharacteristicTypeFromName(name)
	if !ok {
		t.Fatalf("no characteristic %q", name)
	}
	return c
}

func encodeParams(t *testing.T, text string) map[string]interface{} {
	t.Helper()
	params := map[string]interface{}{}
	dec := json.NewDecoder(strings.NewReader(text))
	dec.UseNumber()
	if err := dec.Decode(&params); err != nil {
		t.Fatalf("bad case params %s: %v", text, err)
	}
	return params
}

func jarBridge(t *testing.T, codec string) *Bridge {
	t.Helper()
	jarPath := os.Getenv("FAKETANDEM_TEST_CLIPARSER_JAR")
	if jarPath == "" {
		t.Skip("FAKETANDEM_TEST_CLIPARSER_JAR not set, skipping the bridge comparison against cliparser")
	}
	bridge, err := NewBridgeWithCodec(codec, "", "jar", "", "java", jarPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closer, ok := bridge.runner.(*ServeRunner); ok {
			_ = closer.Close()
		}
	})
	return bridge
}

func TestNativeBridgeParseMatchesJarBridge(t *testing.T) {
	jar := jarBridge(t, CodecPumpX2)
	native := NewNativeBridge()
	for _, c := range parity.ParseCases() {
		ch := charType(t, c.Characteristic)
		want, jerr := jar.ParseMessage(ch, c.Fragments)
		got, nerr := native.ParseMessage(ch, c.Fragments)
		if jerr != nil || nerr != nil {
			if (jerr == nil) != (nerr == nil) {
				t.Errorf("%s: native error %v, jar error %v", c.Key, nerr, jerr)
			}
			continue
		}
		// Where cliparser printed an exception the jar path reads its text as
		// the name and the port leaves it empty; neither reaches a handler.
		if !got.IsValid && !messages.IsMessageName(want.MessageType) {
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s:\n  native: %+v\n  jar:    %+v", c.Key, got, want)
		}
	}
}

func TestNativeBridgeEncodeMatchesJarBridge(t *testing.T) {
	jar := jarBridge(t, CodecPumpX2)
	native := NewNativeBridge()
	for _, c := range parity.EncodeCases() {
		params := encodeParams(t, c.Params)
		want, jerr := jar.EncodeMessage(c.TxID, c.Message, params)
		got, nerr := native.EncodeMessage(c.TxID, c.Message, params)
		if (jerr == nil) != (nerr == nil) {
			t.Errorf("%s: native error %v, jar error %v", c.Key, nerr, jerr)
			continue
		}
		if jerr == nil && !reflect.DeepEqual(got, want) {
			t.Errorf("%s:\n  native: %+v\n  jar:    %+v", c.Key, got, want)
		}
	}
}

func TestCompareBridgeFindsNoMismatches(t *testing.T) {
	bridge := jarBridge(t, CodecCompare)
	for _, c := range parity.ParseCases() {
		_, _ = bridge.ParseMessage(charType(t, c.Characteristic), c.Fragments) // Safe to ignore: only the comparison is under test.
	}
	for _, c := range parity.EncodeCases() {
		_, _ = bridge.EncodeMessage(c.TxID, c.Message, encodeParams(t, c.Params)) // Safe to ignore: only the comparison is under test.
	}
	if n := bridge.Mismatches(); n != 0 {
		t.Fatalf("compare mode logged %d mismatches between the Go port and cliparser", n)
	}
}

// TestNativeBridgeMatchesRecordedJar runs without a JVM: the bridge's parse of
// each case against what the jar printed for it, read the way the jar path
// reads it.
func TestNativeBridgeMatchesRecordedJar(t *testing.T) {
	golden, err := parity.ReadParseGolden()
	if err != nil {
		t.Fatal(err)
	}
	native := NewNativeBridge()
	for _, c := range parity.ParseCases() {
		got, err := native.ParseMessage(charType(t, c.Characteristic), c.Fragments)
		want := golden[c.Key]
		if err != nil {
			if want.OK {
				t.Errorf("%s: native failed (%v) where cliparser printed %s", c.Key, err, want.ToString)
			}
			continue
		}
		// No dump on record is an exception, or a field holding a tab, which
		// garbles the name on both paths alike.
		if !want.OK {
			if messages.IsMessageName(got.MessageType) {
				t.Errorf("%s: native parsed %s where cliparser printed no message", c.Key, got.MessageType)
			}
			continue
		}
		name, cargo := parseCliparserOutput(want.ToString)
		if got.MessageType != name || !reflect.DeepEqual(got.Cargo, cargo) {
			t.Errorf("%s:\n  native: %s %v\n  jar:    %s %v", c.Key, got.MessageType, got.Cargo, name, cargo)
		}
	}
}
