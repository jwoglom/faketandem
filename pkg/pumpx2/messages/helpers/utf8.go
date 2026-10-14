package helpers

import "unicode/utf8"

// JavaUTF8Decode is new String(b, UTF_8), a port of JDK 21's
// String.decodeUTF8_UTF16: how many bytes one U+FFFD stands for differs from
// both strings.ToValidUTF8 (a whole run) and utf8.DecodeRune (each byte).
func JavaUTF8Decode(b []byte) string {
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
