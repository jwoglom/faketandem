package parity

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
)

// CheckClass holds the Go implementation of a message class to every answer
// the jar gave for it (its testdata/<Class>.jsonl): each parse prints what
// cliparser printed, and each encode makes the packets cliparser made.
func CheckClass(t *testing.T, className string) {
	t.Helper()
	cat, err := LoadJarCatalog()
	if err != nil {
		t.Fatal(err)
	}
	jc, ok := cat.Class(className)
	if !ok {
		t.Fatalf("the jar has no class %s", className)
	}
	v, err := ReadVectors(ClassVectorsPath(jc))
	if err != nil {
		t.Fatalf("no recorded jar answers for %s (record them with -update-golden): %v", className, err)
	}
	if len(v.Encode) == 0 {
		t.Fatalf("%s has no recorded encode answers", className)
	}
	CheckVectors(t, v)
}

// CheckVectors holds the Go implementation to recorded jar answers.
func CheckVectors(t *testing.T, v *Vectors) {
	t.Helper()
	for _, pv := range v.Parse {
		if got := NativeParse(pv.ParseCase); !reflect.DeepEqual(got, pv.Jar) {
			t.Errorf("parse %s %v:\n  go:  %+v (%s)\n  jar: %+v", pv.Key, pv.Fragments, got, NativeParseError(pv.ParseCase), pv.Jar)
		}
	}
	for _, ev := range v.Encode {
		if got := NativeEncode(ev.EncodeCase); !reflect.DeepEqual(got, ev.Jar) {
			t.Errorf("encode %s %s %s:\n  go:  %+v (%s)\n  jar: %+v", ev.Key, ev.Message, ev.Params, got, NativeEncodeError(ev.EncodeCase), ev.Jar)
		}
	}
}

// TestMessage is pumpX2's MessageTester.test: rawHex (and extraHexPackets,
// for a message that spans several) parses on characteristic into a message
// that prints as expected does, with expected's cargo, and for an unsigned
// message, expected frames into expectedPackets packets, the first of which
// is rawHex when there is only one. It returns the parsed message.
func TestMessage(t *testing.T, rawHex string, txID, expectedPackets int, characteristic messages.Characteristic, expected messages.Message, extraHexPackets ...string) messages.Message {
	t.Helper()
	return MessageTester{}.Test(t, rawHex, txID, expectedPackets, characteristic, expected, extraHexPackets...)
}

// MessageTester runs pumpX2's MessageTester checks under a pump state:
// MessageTester.initPumpState's authentication key and time since reset.
type MessageTester struct {
	Env messages.Env
}

// Test is TestMessage under the tester's pump state.
func (mt MessageTester) Test(t *testing.T, rawHex string, txID, expectedPackets int, characteristic messages.Characteristic, expected messages.Message, extraHexPackets ...string) messages.Message {
	t.Helper()
	class := messages.ClassOf(expected)
	if class.Props.Characteristic != characteristic {
		t.Fatalf("%s is on %s, not %s", class.Name, class.Props.Characteristic, characteristic)
	}
	fragments := append([]string{rawHex}, extraHexPackets...)
	parsed, err := messages.ParseFragments(string(characteristic), fragments, mt.Env)
	if err != nil {
		t.Fatalf("%s does not parse: %v", strings.Join(fragments, " "), err)
	}
	if messages.ClassOf(parsed) != class {
		t.Fatalf("parsed a %s, want a %s", messages.ClassOf(parsed).Name, class.Name)
	}
	if got, want := messages.ToString(parsed), messages.ToString(expected); got != want {
		t.Errorf("parsed:\n  %s\nwant:\n  %s", got, want)
	}
	if !class.Props.Signed {
		packets, err := messages.Packets(expected, byte(txID), mt.Env)
		if err != nil {
			t.Fatalf("%s does not packetize: %v", class.Name, err)
		}
		if len(packets) != expectedPackets {
			t.Errorf("%d packets, want %d", len(packets), expectedPackets)
		}
		if len(extraHexPackets) == 0 && len(packets) == 1 && hex.EncodeToString(packets[0]) != rawHex {
			t.Errorf("packet %x, want %s", packets[0], rawHex)
		}
	}
	if !bytes.Equal(parsed.Cargo(), expected.Cargo()) {
		t.Errorf("cargo %x, want %x", parsed.Cargo(), expected.Cargo())
	}
	return parsed
}

// TestMessagePackets is pumpX2's MessageTester.testMultiplePackets: the
// packets parse on characteristic into a message of expected's class and
// cargo, and expected frames into exactly those packets.
func TestMessagePackets(t *testing.T, rawHex []string, txID int, characteristic messages.Characteristic, expected messages.Message) messages.Message {
	t.Helper()
	class := messages.ClassOf(expected)
	if class.Props.Characteristic != characteristic {
		t.Fatalf("%s is on %s, not %s", class.Name, class.Props.Characteristic, characteristic)
	}
	parsed, err := messages.ParseFragments(string(characteristic), rawHex, messages.Env{})
	if err != nil {
		t.Fatalf("%v does not parse: %v", rawHex, err)
	}
	if messages.ClassOf(parsed) != class {
		t.Fatalf("parsed a %s, want a %s", messages.ClassOf(parsed).Name, class.Name)
	}
	packets, err := messages.Packets(expected, byte(txID), messages.Env{})
	if err != nil {
		t.Fatalf("%s does not packetize: %v", class.Name, err)
	}
	if len(packets) != len(rawHex) {
		t.Fatalf("%d packets, want %d", len(packets), len(rawHex))
	}
	for i, p := range packets {
		if hex.EncodeToString(p) != rawHex[i] {
			t.Errorf("packet %d: %x, want %s", i, p, rawHex[i])
		}
	}
	if !bytes.Equal(parsed.Cargo(), expected.Cargo()) {
		t.Errorf("cargo %x, want %x", parsed.Cargo(), expected.Cargo())
	}
	return parsed
}
