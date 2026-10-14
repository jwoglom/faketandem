package messages

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// JavaException is a failure the jar reports as a thrown exception. The parity
// tests hold only *that* a call fails to the jar; the message is for logs.
type JavaException struct {
	Class   string
	Message string
}

func (e *JavaException) Error() string {
	if e.Message == "" {
		return e.Class
	}
	return e.Class + ": " + e.Message
}

func exception(class, format string, args ...interface{}) error {
	return &JavaException{Class: class, Message: fmt.Sprintf(format, args...)}
}

// illegalArgument is what commons-lang3's Validate.isTrue throws.
func illegalArgument(format string, args ...interface{}) error {
	return exception("java.lang.IllegalArgumentException", format, args...)
}

// check ports Validate.isTrue: it panics with the IllegalArgumentException,
// which the parse/constructor boundary recovers into an error. Ports call it
// where the Java does, so a port reads like the method it ports.
func check(ok bool) {
	if !ok {
		panic(illegalArgument("The validated expression is false"))
	}
}

// checkf is Validate.isTrue with a message.
func checkf(ok bool, format string, args ...interface{}) {
	if !ok {
		panic(illegalArgument(format, args...))
	}
}

// throw panics with a Java exception, for a port's explicit `throw`.
func throw(class, format string, args ...interface{}) {
	panic(exception(class, format, args...))
}

// fromPanic turns a panic inside a ported method into the exception Java
// would have thrown: a *JavaException as itself, and a Go index error as the
// ArrayIndexOutOfBoundsException of the same out-of-range read, so ports can
// index cargo as freely as the Java does. recover() only works called directly
// from the deferred function, so callers write
//
//	defer func() { if r := recover(); r != nil { err = fromPanic(r) } }()
func fromPanic(r interface{}) error {
	if je, ok := r.(*JavaException); ok {
		return je
	}
	if e, ok := r.(error); ok {
		if strings.Contains(e.Error(), "index out of range") || strings.Contains(e.Error(), "slice bounds out of range") {
			return exception("java.lang.ArrayIndexOutOfBoundsException", "%v", e)
		}
		return exception("java.lang.RuntimeException", "%v", e)
	}
	return exception("java.lang.RuntimeException", "%v", r)
}

// sb is a Java byte read as an int: raw[i] in Java is signed.
func sb(b byte) int32 { return int32(int8(b)) }

// ub is (raw[i] & 0xFF).
func ub(b byte) int32 { return int32(b) }

// --- com.jwoglom.pumpx2.pump.messages.helpers.Bytes ---

func combine(items ...[]byte) []byte {
	out := []byte{}
	for _, item := range items {
		out = append(out, item...)
	}
	return out
}

func dropFirstN(b []byte, n int) []byte {
	checkf(n >= 0 && n <= len(b), "cannot drop first %d of %d bytes", n, len(b))
	return append([]byte{}, b[n:]...)
}

func dropLastN(b []byte, n int) []byte {
	checkf(n >= 0 && n <= len(b), "cannot drop last %d of %d bytes", n, len(b))
	return append([]byte{}, b[:len(b)-n]...)
}

func reverseBytes(b []byte) []byte {
	out := make([]byte, len(b))
	for i := range b {
		out[i] = b[len(b)-1-i]
	}
	return out
}

// readShort is Bytes.readShort: an unsigned little-endian 16-bit read.
func readShort(raw []byte, i int) int32 {
	check(i >= 0 && i+1 < len(raw))
	return int32(raw[i+1])<<8 | int32(raw[i])
}

// readUint32 is Bytes.readUint32.
func readUint32(raw []byte, i int) int64 {
	if i < 0 || i+3 >= len(raw) {
		panic(exception("java.lang.ArrayIndexOutOfBoundsException", ""))
	}
	return int64(binary.LittleEndian.Uint32(raw[i:]))
}

// readUint64 is Bytes.readUint64.
func readUint64(raw []byte, i int) *big.Int {
	if i < 0 || i+7 >= len(raw) {
		panic(exception("java.lang.ArrayIndexOutOfBoundsException", "%d", i))
	}
	return new(big.Int).SetBytes(reverseBytes(raw[i : i+8]))
}

// toUint32 is Bytes.toUint32: the low four bytes of the long, little-endian.
func toUint32(v int64) []byte {
	out := make([]byte, 4)
	binary.LittleEndian.PutUint32(out, uint32(v))
	return out
}

func toUint64(v int64) []byte {
	out := make([]byte, 8)
	binary.LittleEndian.PutUint64(out, uint64(v))
	return out
}

// firstTwoBytesLittleEndian is Bytes.firstTwoBytesLittleEndian.
func firstTwoBytesLittleEndian(v int32) []byte {
	return []byte{byte(v), byte(v >> 8)}
}

// firstByteLittleEndian is Bytes.firstByteLittleEndian.
func firstByteLittleEndian(v int32) []byte {
	return []byte{byte(v)}
}

// readString is Bytes.readString: a NUL-terminated, fixed-width UTF-8 field.
func readString(raw []byte, i, length int) string {
	if i < 0 || i >= len(raw) {
		panic(exception("java.lang.ArrayIndexOutOfBoundsException", "%d", i))
	}
	checkf(length >= 0, "length must be non-negative, got %d", length)
	end := i + length
	if end > len(raw) {
		end = len(raw)
	}
	term := i
	for term < end && raw[term] != 0 {
		term++
	}
	return javaUTF8Decode(raw[i:term])
}

// writeString is Bytes.writeString.
func writeString(s string, length int) []byte {
	checkf(length >= 0, "length must be non-negative, got %d", length)
	encoded := []byte(s)
	checkf(len(encoded) <= length, "string encodes to %d bytes which does not fit a %d byte field", len(encoded), length)
	out := make([]byte, length)
	copy(out, encoded)
	return out
}

// javaUTF8Decode is new String(b, UTF_8), a port of JDK 21's
// String.decodeUTF8_UTF16: how many bytes one U+FFFD stands for differs from
// both strings.ToValidUTF8 (a whole run) and utf8.DecodeRune (each byte).
func javaUTF8Decode(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	out := make([]rune, 0, len(b))
	for i := 0; i < len(b); {
		b1 := b[i]
		i++
		var r rune
		switch {
		case b1 < 0x80:
			r = rune(b1)
		case b1 >= 0xc2 && b1 <= 0xdf:
			r, i = javaDecode2(b, b1, i)
		case b1 >= 0xe0 && b1 <= 0xef:
			r, i = javaDecode3(b, b1, i)
		case b1 >= 0xf0 && b1 <= 0xf7:
			r, i = javaDecode4(b, b1, i)
		default:
			r = utf8.RuneError
		}
		out = append(out, r)
	}
	return string(out)
}

func utf8NotCont(x byte) bool { return x&0xc0 != 0x80 }

// javaDecode2, javaDecode3 and javaDecode4 decode the sequence a lead byte
// opens, at i just past it, returning the rune and the index after what it
// consumed; a truncated sequence at the end consumes the rest of the input.
func javaDecode2(b []byte, b1 byte, i int) (rune, int) {
	if i < len(b) && !utf8NotCont(b[i]) {
		return rune(b1&0x1f)<<6 | rune(b[i]&0x3f), i + 1
	}
	return utf8.RuneError, i
}

func javaDecode3(b []byte, b1 byte, i int) (rune, int) {
	n := len(b)
	overlong := func(b2 byte) bool { return b1 == 0xe0 && b2&0xe0 == 0x80 }
	if i+1 < n {
		b2, b3 := b[i], b[i+1]
		if overlong(b2) || utf8NotCont(b2) {
			return utf8.RuneError, i
		}
		if utf8NotCont(b3) {
			return utf8.RuneError, i + 1
		}
		c := rune(b1&0x0f)<<12 | rune(b2&0x3f)<<6 | rune(b3&0x3f)
		if c >= 0xd800 && c <= 0xdfff {
			c = utf8.RuneError
		}
		return c, i + 2
	}
	if i < n && (overlong(b[i]) || utf8NotCont(b[i])) {
		return utf8.RuneError, i
	}
	return utf8.RuneError, n
}

func javaDecode4(b []byte, b1 byte, i int) (rune, int) {
	n := len(b)
	if i+2 < n {
		return javaDecodeComplete4(b, b1, i)
	}
	if b1 > 0xf4 || (i < n && javaBadSecond4(b1, b[i])) {
		return utf8.RuneError, i
	}
	i++
	if i < n && utf8NotCont(b[i]) {
		return utf8.RuneError, i
	}
	return utf8.RuneError, n
}

func javaBadSecond4(b1, b2 byte) bool {
	return (b1 == 0xf0 && (b2 < 0x90 || b2 > 0xbf)) || (b1 == 0xf4 && b2&0xf0 != 0x80) || utf8NotCont(b2)
}

// javaDecodeComplete4 is a four-byte lead with three bytes after it.
func javaDecodeComplete4(b []byte, b1 byte, i int) (rune, int) {
	b2, b3, b4 := b[i], b[i+1], b[i+2]
	uc := rune(b1&0x07)<<18 | rune(b2&0x3f)<<12 | rune(b3&0x3f)<<6 | rune(b4&0x3f)
	if !utf8NotCont(b2) && !utf8NotCont(b3) && !utf8NotCont(b4) && uc >= 0x10000 && uc <= 0x10ffff {
		return uc, i + 3
	}
	switch {
	case b1 > 0xf4 || javaBadSecond4(b1, b2):
		return utf8.RuneError, i
	case utf8NotCont(b3):
		return utf8.RuneError, i + 1
	default:
		return utf8.RuneError, i + 2
	}
}

// copyOfRange is Arrays.copyOfRange: zero-padded past the end of the source.
func copyOfRange(b []byte, from, to int) []byte {
	if from < 0 || from > len(b) {
		panic(exception("java.lang.ArrayIndexOutOfBoundsException", "%d", from))
	}
	if from > to {
		panic(illegalArgument("%d > %d", from, to))
	}
	out := make([]byte, to-from)
	copy(out, b[from:])
	return out
}

// crc16 is Bytes.calculateCRC16 (CRC-16/CCITT-FALSE), low byte first.
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

// --- How ReflectionToStringBuilder (SHORT_PREFIX_STYLE) prints values ---

const javaNull = "<null>"

func jInt(v int32) string  { return strconv.FormatInt(int64(v), 10) }
func jLong(v int64) string { return strconv.FormatInt(v, 10) }
func jBool(v bool) string  { return strconv.FormatBool(v) }

// jBytes prints a byte[] as {1,-2,3}.
func jBytes(b []byte) string {
	if b == nil {
		return javaNull
	}
	parts := make([]string, len(b))
	for i, x := range b {
		parts[i] = strconv.Itoa(int(int8(x)))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// jLongArray2D prints a long[][] as {{1,2},{3}}.
func jLongArray2D(a [][]int64) string {
	if a == nil {
		return javaNull
	}
	rows := make([]string, len(a))
	for i, row := range a {
		if row == nil {
			rows[i] = javaNull
			continue
		}
		cells := make([]string, len(row))
		for j, v := range row {
			cells[j] = jLong(v)
		}
		rows[i] = "{" + strings.Join(cells, ",") + "}"
	}
	return "{" + strings.Join(rows, ",") + "}"
}

// jCollection prints a Collection as AbstractCollection.toString does.
func jCollection(items []string) string {
	return "[" + strings.Join(items, ", ") + "]"
}

// jString prints a possibly-null String field.
func jString(s *string) string {
	if s == nil {
		return javaNull
	}
	return *s
}

// jBigInt prints a BigInteger.
func jBigInt(v *big.Int) string {
	if v == nil {
		return javaNull
	}
	return v.String()
}

// jInstant prints a java.time.Instant as DateTimeFormatter.ISO_INSTANT does:
// the fraction is left out when zero, else written in groups of three digits.
func jInstant(t time.Time) string {
	t = t.UTC()
	year := t.Year()
	var date string
	switch {
	case year > 9999:
		date = fmt.Sprintf("+%d-%02d-%02d", year, t.Month(), t.Day())
	case year < 0:
		date = fmt.Sprintf("-%04d-%02d-%02d", -year, t.Month(), t.Day())
	default:
		date = fmt.Sprintf("%04d-%02d-%02d", year, t.Month(), t.Day())
	}
	s := fmt.Sprintf("%sT%02d:%02d:%02d", date, t.Hour(), t.Minute(), t.Second())
	if ns := t.Nanosecond(); ns != 0 {
		switch {
		case ns%1000000 == 0:
			s += fmt.Sprintf(".%03d", ns/1000000)
		case ns%1000 == 0:
			s += fmt.Sprintf(".%06d", ns/1000)
		default:
			s += fmt.Sprintf(".%09d", ns)
		}
	}
	return s + "Z"
}

// jDouble is Double.toString (JDK 19+, shortest repr).
func jDouble(v float64) string { return javaFloatString(v, 64) }

func javaFloatString(v float64, bits int) string {
	switch {
	case math.IsNaN(v):
		return "NaN"
	case math.IsInf(v, 1):
		return "Infinity"
	case math.IsInf(v, -1):
		return "-Infinity"
	case v == 0:
		if math.Signbit(v) {
			return "-0.0"
		}
		return "0.0"
	}
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	// Shortest digits and decimal exponent: d.ddd x 10^exp.
	e := strconv.FormatFloat(v, 'e', -1, bits)
	mant, expStr, _ := strings.Cut(e, "e")
	exp, _ := strconv.Atoi(expStr)
	digits := strings.Replace(mant, ".", "", 1)
	if v >= 1e-3 && v < 1e7 {
		pointAt := exp + 1
		var intPart, frac string
		switch {
		case pointAt <= 0:
			intPart = "0"
			frac = strings.Repeat("0", -pointAt) + digits
		case pointAt >= len(digits):
			intPart = digits + strings.Repeat("0", pointAt-len(digits))
			frac = "0"
		default:
			intPart = digits[:pointAt]
			frac = digits[pointAt:]
		}
		return sign + intPart + "." + frac
	}
	frac := digits[1:]
	if frac == "" {
		frac = "0"
	}
	return sign + digits[:1] + "." + frac + "E" + strconv.Itoa(exp)
}

// jAutoToString is JavaHelpers.autoToString (ReflectionToStringBuilder,
// SHORT_PREFIX_STYLE) of a nested model: shortName is the class's short name
// with its outer class, e.g. "RemindersResponse.Reminder", and fields are its
// name/printed-value pairs in name order, then its superclasses'.
func jAutoToString(shortName string, fields [][2]string) string {
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = f[0] + "=" + f[1]
	}
	return shortName + "[" + strings.Join(parts, ",") + "]"
}
