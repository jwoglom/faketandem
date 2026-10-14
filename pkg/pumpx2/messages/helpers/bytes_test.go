package helpers

import (
	"bytes"
	"math/big"
	"testing"
)

func TestLittleEndianReadsAndWrites(t *testing.T) {
	raw := []byte{0x9a, 0x29, 0xff, 0xff, 0xff, 0xff, 0x00, 0x80}
	if got := ReadShort(raw, 0); got != 10650 {
		t.Errorf("ReadShort = %d", got)
	}
	if got := ReadUint32(raw, 2); got != 4294967295 {
		t.Errorf("ReadUint32 = %d", got)
	}
	if got := ReadUint64(raw, 0); got.Cmp(new(big.Int).SetUint64(0x8000ffffffff299a)) != 0 {
		t.Errorf("ReadUint64 = %s", got)
	}
	if got := ToUint32(-1); !bytes.Equal(got, []byte{0xff, 0xff, 0xff, 0xff}) {
		t.Errorf("ToUint32(-1) = %x", got)
	}
	if got := FirstTwoBytesLittleEndian(70000); !bytes.Equal(got, []byte{0x70, 0x11}) {
		t.Errorf("FirstTwoBytesLittleEndian(70000) = %x", got)
	}
	if got := ReadFloat(ToFloat(1.5), 0); got != 1.5 {
		t.Errorf("ReadFloat(ToFloat(1.5)) = %v", got)
	}
}

func TestLongValueKeepsTheLow64Bits(t *testing.T) {
	x, _ := new(big.Int).SetString("18446744073709551617", 10)
	if got := LongValue(x); got != 1 {
		t.Errorf("LongValue(2^64+1) = %d", got)
	}
	if got := LongValue(big.NewInt(-2)); got != -2 {
		t.Errorf("LongValue(-2) = %d", got)
	}
}

func TestStringFields(t *testing.T) {
	if got := ReadString([]byte("abc\x00def"), 0, 8); got != "abc" {
		t.Errorf("ReadString = %q", got)
	}
	field, err := WriteString("ab", 4)
	if err != nil || !bytes.Equal(field, []byte{'a', 'b', 0, 0}) {
		t.Errorf("WriteString = %x, %v", field, err)
	}
	if _, err := WriteString("abcde", 4); err == nil {
		t.Error("WriteString fitted five bytes into four")
	}
}

func TestCRC16(t *testing.T) {
	if got := CRC16([]byte{0x20, 0x02, 0x00}); !bytes.Equal(got, []byte{0x38, 0x2c}) {
		t.Errorf("CRC16 = %x", got)
	}
}
