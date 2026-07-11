package pumpx2

import (
	"encoding/hex"
	"strconv"
	"strings"
)

// opcodeAndTxIDFromFirstFragment extracts the opcode and txId directly from the
// first raw BLE fragment, per pumpX2's real wire format: byte[0]/byte[1] are the
// fragment-level [remaining][txId] framing this project's own reassembler adds,
// byte[2] is the message opcode, byte[3] is the message-level txId (usually the
// same value as the fragment-level txId, but pumpX2 defines it separately).
// We compute these ourselves rather than trust the cliparser CLI's own opcode
// output, since parseOpcode() in pumpX2's Main.java naively hex-decodes the
// entire (space-joined, multi-fragment) argument string and always fails with a
// decode error whenever a message spans more than one fragment.
func opcodeAndTxIDFromFirstFragment(rawPacketsHex []string) (opcode int, txID int, ok bool) {
	if len(rawPacketsHex) == 0 {
		return 0, 0, false
	}
	b, err := hex.DecodeString(rawPacketsHex[0])
	if err != nil || len(b) < 4 {
		return 0, 0, false
	}
	return int(int8(b[2])), int(b[3]), true
}

// parseCliparserOutput extracts the message name and cargo fields from the
// cliparser "parse" command's stdout. The real shape varies by how many
// leading tab-separated fields precede the message dump:
//
//	<opcode>\t<FullyQualifiedClassName>\t<MessageName>[<field>=<value>,...]
//
// but when parse()'s own naive parseOpcode() fails to hex-decode the (space-
// joined, multi-fragment) argument as a single blob, it instead prepends an
// error field before the tab, e.g.:
//
//	Unable to parse: ...DecoderException: Odd length for hex string\t<MessageName>[...]
//
// Either way, the message dump is always the LAST tab-separated field, so
// split on the last tab rather than the first -- splitting on the first tab
// left the FQCN+tab+MessageName glued together as a single bogus "message
// name" for single-fragment messages, which have no leading error field.
func parseCliparserOutput(output string) (messageName string, cargo map[string]interface{}) {
	cargo = make(map[string]interface{})

	tail := output
	if idx := strings.LastIndex(output, "\t"); idx != -1 {
		tail = output[idx+1:]
	}
	tail = strings.TrimSpace(tail)

	open := strings.Index(tail, "[")
	closeIdx := strings.LastIndex(tail, "]")
	if open == -1 || closeIdx == -1 || closeIdx < open {
		// No bracketed field list (e.g. an error message, or a message with no
		// fields at all); treat everything up to the first "[" (or the whole
		// string) as the message name.
		if open != -1 {
			return strings.TrimSpace(tail[:open]), cargo
		}
		return strings.TrimSpace(tail), cargo
	}

	messageName = strings.TrimSpace(tail[:open])
	fields := splitTopLevel(tail[open+1:closeIdx], ',')
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		eq := strings.Index(field, "=")
		if eq == -1 {
			continue
		}
		key := strings.TrimSpace(field[:eq])
		value := strings.TrimSpace(field[eq+1:])
		cargo[key] = parseFieldValue(value)
	}

	return messageName, cargo
}

// splitTopLevel splits s on sep, ignoring occurrences of sep nested inside {} or
// [] (Java's Arrays.toString/List.toString use these for nested values).
func splitTopLevel(s string, sep byte) []string {
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		case sep:
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// parseFieldValue converts a single Java toString() field value into a Go value.
// Byte-array fields render as "{65,4,119,...}" (signed decimal bytes); these are
// converted to a hex string. Everything else is parsed as a number if possible,
// falling back to the raw string.
func parseFieldValue(value string) interface{} {
	if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
		inner := value[1 : len(value)-1]
		if strings.TrimSpace(inner) == "" {
			return ""
		}
		parts := splitTopLevel(inner, ',')
		buf := make([]byte, 0, len(parts))
		for _, p := range parts {
			n, err := strconv.Atoi(strings.TrimSpace(p))
			if err != nil {
				// Not a byte array after all; return the literal text.
				return value
			}
			buf = append(buf, byte(int8(n)))
		}
		return hex.EncodeToString(buf)
	}

	if i, err := strconv.Atoi(value); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}
	if value == "true" || value == "false" {
		return value == "true"
	}
	return value
}
