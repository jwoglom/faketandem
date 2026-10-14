// Package helpers is pumpX2's messages.helpers: the byte and date arithmetic
// every message's parse and constructors share.
//
// Like Go's own binary.LittleEndian, a read past the end of its input panics;
// a message's Parse checks the length it needs first, as the Java does, and
// messages.Parse turns a panic that gets past it into an error, as
// Messages.parse catches every exception.
package helpers

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
)

// DropFirstN returns a copy of b without its first n bytes.
func DropFirstN(b []byte, n int) []byte {
	if n < 0 || n > len(b) {
		panic(fmt.Errorf("cannot drop first %d of %d bytes", n, len(b)))
	}
	return append([]byte{}, b[n:]...)
}

// DropLastN returns a copy of b without its last n bytes.
func DropLastN(b []byte, n int) []byte {
	if n < 0 || n > len(b) {
		panic(fmt.Errorf("cannot drop last %d of %d bytes", n, len(b)))
	}
	return append([]byte{}, b[:len(b)-n]...)
}

// Reverse returns b back to front.
func Reverse(b []byte) []byte {
	out := make([]byte, len(b))
	for i := range b {
		out[i] = b[len(b)-1-i]
	}
	return out
}

// Combine concatenates its arguments into a new, never nil, slice.
func Combine(items ...[]byte) []byte {
	out := []byte{}
	for _, item := range items {
		out = append(out, item...)
	}
	return out
}

// EmptyBytes is size zero bytes.
func EmptyBytes(size int) []byte { return make([]byte, size) }

// ReadShort is an unsigned little-endian 16-bit read.
func ReadShort(raw []byte, i int) int32 {
	return int32(binary.LittleEndian.Uint16(raw[i : i+2]))
}

// ReadFloat is a little-endian IEEE 754 single.
func ReadFloat(raw []byte, i int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(raw[i : i+4]))
}

// ToFloat is f as a little-endian IEEE 754 single.
func ToFloat(f float32) []byte {
	out := make([]byte, 4)
	binary.LittleEndian.PutUint32(out, math.Float32bits(f))
	return out
}

// ReadUint32 is an unsigned little-endian 32-bit read.
func ReadUint32(raw []byte, i int) int64 {
	return int64(binary.LittleEndian.Uint32(raw[i : i+4]))
}

// ReadUint64 is an unsigned little-endian 64-bit read, which needs a big.Int
// as it needs a BigInteger in Java.
func ReadUint64(raw []byte, i int) *big.Int {
	return new(big.Int).SetUint64(binary.LittleEndian.Uint64(raw[i : i+8]))
}

// ToUint32 is the low four bytes of v, little-endian.
func ToUint32(v int64) []byte {
	out := make([]byte, 4)
	binary.LittleEndian.PutUint32(out, uint32(v))
	return out
}

// ToUint64 is v, little-endian.
func ToUint64(v int64) []byte {
	out := make([]byte, 8)
	binary.LittleEndian.PutUint64(out, uint64(v))
	return out
}

// FirstTwoBytesLittleEndian is the low two bytes of v, little-endian: what
// ReadShort reads back.
func FirstTwoBytesLittleEndian(v int32) []byte {
	return []byte{byte(v), byte(v >> 8)}
}

// FirstByteLittleEndian is the low byte of v.
func FirstByteLittleEndian(v int32) []byte {
	return []byte{byte(v)}
}

// ReadString reads a NUL-terminated UTF-8 field of at most length bytes at i,
// decoding malformed input as Java's new String(bytes, UTF_8) does.
func ReadString(raw []byte, i, length int) string {
	if i < 0 || i >= len(raw) {
		panic(fmt.Errorf("string field at %d of %d bytes", i, len(raw)))
	}
	if length < 0 {
		panic(fmt.Errorf("length must be non-negative, got %d", length))
	}
	end := i + length
	if end > len(raw) {
		end = len(raw)
	}
	term := i
	for term < end && raw[term] != 0 {
		term++
	}
	return JavaUTF8Decode(raw[i:term])
}

// WriteString encodes s as UTF-8 into a NUL-padded field of exactly length
// bytes, and fails when it does not fit.
func WriteString(s string, length int) ([]byte, error) {
	if length < 0 {
		return nil, fmt.Errorf("length must be non-negative, got %d", length)
	}
	if len(s) > length {
		return nil, fmt.Errorf("string encodes to %d bytes which does not fit a %d byte field", len(s), length)
	}
	out := make([]byte, length)
	copy(out, s)
	return out, nil
}

// CopyOfRange is Arrays.copyOfRange: b[from:to], zero-padded past the end of b.
func CopyOfRange(b []byte, from, to int) []byte {
	if from < 0 || from > len(b) {
		panic(fmt.Errorf("copy from %d of %d bytes", from, len(b)))
	}
	if from > to {
		panic(fmt.Errorf("copy range %d > %d", from, to))
	}
	out := make([]byte, to-from)
	copy(out, b[from:])
	return out
}

// CRC16 is the CRC-16/CCITT-FALSE of data, low byte first.
func CRC16(data []byte) []byte {
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

// LongValue is BigInteger.longValue: the low 64 bits, two's complement.
func LongValue(x *big.Int) int64 {
	mask := new(big.Int).Lsh(big.NewInt(1), 64)
	mask.Sub(mask, big.NewInt(1))
	return int64(new(big.Int).And(x, mask).Uint64())
}

// Bool is a boolean as the single byte pumpX2 writes for one.
func Bool(b bool) byte {
	if b {
		return 1
	}
	return 0
}
