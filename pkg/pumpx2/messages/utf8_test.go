package messages

import (
	"bufio"
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestJavaUTF8Decode holds the decoder to what Java's new String(b, UTF_8)
// makes of ill-formed input; the vectors are the JVM's own output.
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
		if got := runesHex(javaUTF8Decode(b)); got != want {
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

// TestJavaUTF8DecodeMatchesJVM replays byte strings the JVM decoded
// (testdata/java_utf8_vectors.txt.gz: hex input, then the code points Java's
// new String(b, UTF_8) gave), weighted towards lead and continuation bytes.
func TestJavaUTF8DecodeMatchesJVM(t *testing.T) {
	f, err := os.Open("testdata/java_utf8_vectors.txt.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }() // Safe to ignore: read-only.
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(gz)
	n := 0
	for sc.Scan() {
		parts := strings.SplitN(sc.Text(), " ", 2)
		b, _ := hex.DecodeString(parts[0])
		if got := runesHex(javaUTF8Decode(b)); got != parts[1] {
			t.Errorf("%s: got %s, want %s", parts[0], got, parts[1])
		}
		n++
	}
	if n == 0 {
		t.Fatal("no vectors")
	}
}
