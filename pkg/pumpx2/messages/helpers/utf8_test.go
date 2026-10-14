package helpers

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestJavaUTF8Decode(t *testing.T) {
	cases := map[string]string{
		"c190ef00":     "fffd,fffd,fffd,0,",
		"f340dfaf1bff": "fffd,40,7ef,1b,fffd,",
		"ef":           "fffd,",
		"4d8c99cdccd2": "4d,fffd,fffd,fffd,fffd,fffd,",
		"e0a0":         "fffd,",
		"f09080":       "fffd,",
		"eda080":       "fffd,",
		"f4908080":     "fffd,fffd,fffd,fffd,",
		"c3a9":         "e9,",
	}
	for in, want := range cases {
		b, _ := hex.DecodeString(in)
		if got := runesHex(JavaUTF8Decode(b)); got != want {
			t.Errorf("%s: got %s, want %s", in, got, want)
		}
	}
}

func runesHex(s string) string {
	var b strings.Builder
	for _, r := range s {
		fmt.Fprintf(&b, "%x,", r)
	}
	return b.String()
}

// The vectors are the JVM's own decoding (hex input, then the code points
// new String(b, UTF_8) gave), weighted towards lead and continuation bytes.
func TestJavaUTF8DecodeMatchesJVM(t *testing.T) {
	f, err := os.Open("testdata/java_utf8_vectors.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }() // Safe to ignore: read-only.
	sc := bufio.NewScanner(f)
	n := 0
	for sc.Scan() {
		parts := strings.SplitN(sc.Text(), " ", 2)
		b, _ := hex.DecodeString(parts[0])
		if got := runesHex(JavaUTF8Decode(b)); got != parts[1] {
			t.Errorf("%s: got %s, want %s", parts[0], got, parts[1])
		}
		n++
	}
	if n == 0 {
		t.Fatal("no vectors")
	}
}
