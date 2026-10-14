// Package parity holds the Go message types to the pumpX2 cliparser jar: it
// generates the parse and encode cases, drives the jar to record its answers
// (each class's beside its type, in testdata/<Class>.jsonl), and checks the
// Go types against those answers, with or without a JVM.
package parity

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	// Every message type, for the cases' answers from the Go implementation.
	_ "github.com/jwoglom/faketandem/pkg/pumpx2/messages/all"
)

// ParseCase is one `cliparser parse` invocation: the fragments of one
// message and the characteristic faketandem names for them.
type ParseCase struct {
	Key            string   `json:"key"`
	Characteristic string   `json:"characteristic"`
	Fragments      []string `json:"fragments"`
}

// EncodeCase is one `cliparser encode` invocation.
type EncodeCase struct {
	Key     string `json:"key"`
	TxID    int    `json:"txId"`
	Message string `json:"message"`
	Params  string `json:"params"`
}

// HistoryLogStreamResponse prints every history log record it carries, and
// faketandem does not implement pumpX2's 185 history log classes; it is a
// pump to app stream that no client ever writes, so its parse is left out.
const unportedParse = "response.historyLog.HistoryLogStreamResponse"

// GenerateParseCases returns the parse inputs: several cargos per OPCODES
// entry, signed and unsigned where the message is signed, a wrong size, and
// the same message under no characteristic and another one, then framing
// faults. Each is keyed characteristic/opcode/class/variant.
func GenerateParseCases(cat *JarCatalog) []ParseCase {
	rng := rand.New(rand.NewSource(20260101))
	var out []ParseCase
	for _, e := range cat.Opcodes {
		class := cat.mustClass(e.Class)
		if class.Name == unportedParse {
			continue
		}
		size := class.Props.Size
		cargos := map[string][]byte{
			"zeros":  make([]byte, size),
			"ones":   fill(size, 0xFF),
			"rand1":  randBytes(rng, size, 256),
			"rand2":  randBytes(rng, size, 256),
			"small1": randBytes(rng, size, 3),
			"small2": randBytes(rng, size, 2),
			"sparse": sparse(rng, size),
			"long":   randBytes(rng, size+1, 256),
		}
		if size > 0 {
			cargos["empty"] = []byte{}
			cargos["short"] = randBytes(rng, size-1, 4)
		}
		if class.Props.Signed {
			for _, name := range []string{"zeros", "rand1", "small1"} {
				cargos[name+"+signed"] = append(append([]byte{}, cargos[name]...), randBytes(rng, 24, 256)...)
			}
		}
		chunk := 18
		if e.Characteristic == "CONTROL" && class.IsRequest() {
			chunk = 40
		}
		names := sortedKeys(cargos)
		for _, name := range names {
			txID := byte(rng.Intn(256))
			key := fmt.Sprintf("%s/%d/%s/%s", e.Characteristic, e.Opcode, class.SimpleName, name)
			out = append(out, ParseCase{
				Key:            key,
				Characteristic: e.Characteristic,
				Fragments:      Frame(int8(e.Opcode), txID, cargos[name], chunk),
			})
		}
		// The same bytes at the 18-byte chunking, on no characteristic (a
		// QualifyingEvents write), and on each other characteristic.
		zeros := cargos["zeros"]
		txID := byte(rng.Intn(256))
		if chunk != 18 {
			out = append(out, ParseCase{Key: fmt.Sprintf("%s/%d/%s/zeros-chunk18", e.Characteristic, e.Opcode, class.SimpleName),
				Characteristic: e.Characteristic, Fragments: Frame(int8(e.Opcode), txID, zeros, 18)})
		}
		out = append(out, ParseCase{Key: fmt.Sprintf("%s/%d/%s/zeros-nochar", e.Characteristic, e.Opcode, class.SimpleName),
			Characteristic: "", Fragments: Frame(int8(e.Opcode), txID, zeros, chunk)})
		for _, ch := range cat.Characteristics {
			other := ch.Name
			if other == e.Characteristic {
				continue
			}
			out = append(out, ParseCase{Key: fmt.Sprintf("%s/%d/%s/zeros-on-%s", e.Characteristic, e.Opcode, class.SimpleName, other),
				Characteristic: other, Fragments: Frame(int8(e.Opcode), txID, zeros, chunk)})
		}
	}
	return append(out, framingFaultCases(cat)...)
}

func (c *JarCatalog) mustClass(name string) *JarClass {
	for _, jc := range c.Classes {
		if jc.Name == name {
			return jc
		}
	}
	panic("catalog.json names unknown class " + name)
}

// framingFaultCases are a bad CRC, a message txId that differs from the
// fragment's, a missing fragment, an opcode nothing has, and a runt.
func framingFaultCases(cat *JarCatalog) []ParseCase {
	var out []ParseCase
	for _, e := range cat.Opcodes {
		class := cat.mustClass(e.Class)
		if class.Name == unportedParse || class.Props.Size == 0 {
			continue
		}
		frags := Frame(int8(e.Opcode), 9, make([]byte, class.Props.Size), 18)
		bad := append([]string{}, frags...)
		last := bad[len(bad)-1]
		bad[len(bad)-1] = last[:len(last)-2] + flipHex(last[len(last)-2:])
		out = append(out, ParseCase{Key: fmt.Sprintf("%s/%d/%s/badcrc", e.Characteristic, e.Opcode, class.SimpleName),
			Characteristic: e.Characteristic, Fragments: bad})
		mismatched := append([]string{}, frags...)
		first, _ := hex.DecodeString(mismatched[0])
		first[3] = 10
		mismatched[0] = hex.EncodeToString(first)
		out = append(out, ParseCase{Key: fmt.Sprintf("%s/%d/%s/txid-mismatch", e.Characteristic, e.Opcode, class.SimpleName),
			Characteristic: e.Characteristic, Fragments: mismatched})
		if len(frags) > 1 {
			out = append(out, ParseCase{Key: fmt.Sprintf("%s/%d/%s/missing-fragment", e.Characteristic, e.Opcode, class.SimpleName),
				Characteristic: e.Characteristic, Fragments: frags[:len(frags)-1]})
		}
	}
	for _, opcode := range []int8{1, 3, 99, -1, -128} {
		out = append(out, ParseCase{Key: fmt.Sprintf("unknown-opcode/%d", opcode), Characteristic: "CURRENT_STATUS",
			Fragments: Frame(opcode, 1, []byte{1, 2}, 18)})
	}
	out = append(out, ParseCase{Key: "runt", Characteristic: "CURRENT_STATUS", Fragments: []string{"0001"}})
	return out
}

func flipHex(s string) string {
	b, _ := hex.DecodeString(s)
	for i := range b {
		b[i] ^= 0xFF
	}
	return hex.EncodeToString(b)
}

// Frame builds the fragments a client writes for one message: opcode, txId,
// length, cargo and CRC, cut into chunk-byte payloads behind a descending
// remaining-fragments counter and the txId.
func Frame(opcode int8, txID byte, cargo []byte, chunk int) []string {
	body := append([]byte{byte(opcode), txID, byte(len(cargo))}, cargo...)
	body = append(body, crc16(body)...)
	var chunks [][]byte
	for i := 0; i < len(body); i += chunk {
		end := i + chunk
		if end > len(body) {
			end = len(body)
		}
		chunks = append(chunks, body[i:end])
	}
	out := make([]string, len(chunks))
	for i, c := range chunks {
		out[i] = hex.EncodeToString(append([]byte{byte(len(chunks) - 1 - i), txID}, c...))
	}
	return out
}

func crc16(data []byte) []byte {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return []byte{byte(crc), byte(crc >> 8)}
}

func fill(n int, b byte) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}

func randBytes(rng *rand.Rand, n, max int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(rng.Intn(max))
	}
	return out
}

func sparse(rng *rand.Rand, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		if rng.Intn(4) == 0 {
			out[i] = byte(rng.Intn(256))
		}
	}
	return out
}

func sortedKeys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// GenerateEncodeCases returns the encode inputs: for every class, its no-arg
// form and several argument sets for each of its constructors, named by the
// constructor's parameters, plus name and count mismatches. Each is keyed
// class/constructor/variant.
func GenerateEncodeCases(cat *JarCatalog) []EncodeCase {
	rng := rand.New(rand.NewSource(20260102))
	var out []EncodeCase
	add := func(key, message string, params map[string]interface{}) {
		text := "{}"
		if len(params) > 0 {
			b, err := json.Marshal(params)
			if err != nil {
				panic(err)
			}
			text = string(b)
		}
		out = append(out, EncodeCase{Key: key, TxID: rng.Intn(300), Message: message, Params: text})
	}
	classes := append([]*JarClass(nil), cat.Classes...)
	sort.Slice(classes, func(i, j int) bool { return classes[i].Name < classes[j].Name })
	for _, c := range classes {
		add(c.SimpleName+"/noargs", c.SimpleName, nil)
		for ci, ctor := range c.Constructors {
			if len(ctor.Params) == 0 {
				continue
			}
			types := make([]string, len(ctor.Params))
			for i, p := range ctor.Params {
				types[i] = p.Type
			}
			signature := strings.Join(types, ",")
			for _, flavor := range []string{"zero", "typical", "rand1", "rand2", "rand3", "max", "neg", "odd"} {
				params := map[string]interface{}{}
				for _, p := range ctor.Params {
					params[p.Name] = paramValue(rng, flavor, p.Type, c.Props.Size)
				}
				add(fmt.Sprintf("%s/ctor%d(%s)/%s", c.SimpleName, ci, signature, flavor), c.SimpleName, params)
			}
			// The same values under names the constructor does not have,
			// which cliparser assigns by sorted key order instead.
			params := map[string]interface{}{}
			for i, p := range ctor.Params {
				params[fmt.Sprintf("p%02d", i)] = paramValue(rng, "typical", p.Type, c.Props.Size)
			}
			add(fmt.Sprintf("%s/ctor%d(%s)/positional-names", c.SimpleName, ci, signature), c.SimpleName, params)
		}
		params := map[string]interface{}{}
		for i := 0; i < 13; i++ {
			params[fmt.Sprintf("x%02d", i)] = 1
		}
		add(c.SimpleName+"/no-such-arity", c.SimpleName, params)
	}
	for _, e := range cat.Messages {
		add("enum/"+e.Name, e.Name, nil)
	}
	add("unknown-name", "NoSuchMessage", nil)
	return out
}

// paramValue picks one argument of a Java type for a flavor of case.
func paramValue(rng *rand.Rand, flavor, typ string, size int) interface{} {
	switch typ {
	case "int", "short", "byte", "long":
		return intValue(rng, flavor, typ)
	case "boolean":
		switch flavor {
		case "zero", "neg":
			return false
		case "typical", "max":
			return true
		default:
			return rng.Intn(2) == 1
		}
	case "java.lang.String":
		return stringValue(rng, flavor)
	case "byte[]":
		return bytesValue(rng, flavor, size)
	case "java.math.BigInteger":
		if flavor == "max" {
			return json.Number("18446744073709551615")
		}
		return rng.Intn(1000)
	}
	// Enums, collections and models: cliparser cannot convert to them.
	if flavor == "odd" {
		return "VALUE"
	}
	return rng.Intn(4)
}

func stringValue(rng *rand.Rand, flavor string) string {
	switch flavor {
	case "zero":
		return ""
	case "typical":
		return "Profile 1"
	case "max":
		return strings.Repeat("x", 40)
	case "neg":
		return "caf\u00e9 \u2713"
	case "odd":
		return "a,b=c]"
	default:
		return randString(rng)
	}
}

// bytesValue is a hex byte[] argument, mostly sized to the message's cargo.
func bytesValue(rng *rand.Rand, flavor string, size int) string {
	switch flavor {
	case "zero":
		return hex.EncodeToString(make([]byte, size))
	case "typical", "neg":
		return hex.EncodeToString(randBytes(rng, size, 4))
	case "max":
		return hex.EncodeToString(fill(size, 0xFF))
	case "odd":
		return "abc"
	default:
		return hex.EncodeToString(randBytes(rng, rng.Intn(size+8), 256))
	}
}

func intValue(rng *rand.Rand, flavor, typ string) interface{} {
	switch flavor {
	case "zero":
		return 0
	case "typical":
		return 1 + rng.Intn(200)
	case "rand1":
		return rng.Intn(65536)
	case "rand2":
		if typ == "long" {
			return rng.Int63n(1 << 33)
		}
		return int32(rng.Uint32())
	case "rand3":
		return rng.Intn(256)
	case "max":
		if typ == "long" {
			return int64(9223372036854775807)
		}
		return 2147483647
	case "neg":
		return -1 - rng.Intn(1000)
	default: // odd: a value only a lossy conversion takes
		if typ == "long" {
			return json.Number("9223372036854775808")
		}
		return []interface{}{1.5, "7", 3628697757, true}[rng.Intn(4)]
	}
}

func randString(rng *rand.Rand) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 -_"
	n := rng.Intn(20)
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[rng.Intn(len(alphabet))]
	}
	return string(b)
}

// NativeParse runs one parse case through the Go message types.
func NativeParse(c ParseCase) ParseResult {
	m, err := messages.ParseFragments(c.Characteristic, c.Fragments, messages.Env{})
	if err != nil {
		return ParseResult{}
	}
	// Read the way the jar's output is read, so a string field holding a tab
	// truncates both sides alike.
	dump, ok := MessageDump(messages.ToString(m))
	return ParseResult{OK: ok, ToString: dump}
}

// NativeEncode runs one encode case through the Go message types.
func NativeEncode(c EncodeCase) EncodeResult {
	e, err := messages.EncodeJSON(fmt.Sprintf("%d", c.TxID), c.Message, c.Params, messages.Env{})
	if err != nil {
		return EncodeResult{}
	}
	return EncodeResult{OK: true, MessageName: e.MessageName, TxID: e.TxID, Packets: e.Packets,
		Characteristic: e.CharacteristicUUID}
}

// NativeParseError describes why the Go message types failed a parse case, for test
// output.
func NativeParseError(c ParseCase) string {
	if _, err := messages.ParseFragments(c.Characteristic, c.Fragments, messages.Env{}); err != nil {
		return err.Error()
	}
	return "ok"
}

// NativeEncodeError describes why the Go message types failed an encode case.
func NativeEncodeError(c EncodeCase) string {
	if _, err := messages.EncodeJSON(fmt.Sprintf("%d", c.TxID), c.Message, c.Params, messages.Env{}); err != nil {
		return err.Error()
	}
	return "ok"
}
