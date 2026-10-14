package messages_test

import (
	"flag"
	"os"
	"reflect"
	"regexp"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

var updateGolden = flag.Bool("update-golden", false, "rewrite the recorded jar answers from FAKETANDEM_TEST_CLIPARSER_JAR")

// keep applies PARITY_FILTER, a regexp over case keys, so one class can be
// checked at a time while porting. A filtered run never rewrites the golden.
func keep(t *testing.T, key string) bool {
	f := os.Getenv("PARITY_FILTER")
	if f == "" {
		return true
	}
	re, err := regexp.Compile(f)
	if err != nil {
		t.Fatalf("bad PARITY_FILTER: %v", err)
	}
	return re.MatchString(key)
}

func requireJar(t *testing.T) *parity.Jar {
	t.Helper()
	jarPath := os.Getenv("FAKETANDEM_TEST_CLIPARSER_JAR")
	if jarPath == "" {
		t.Skip("FAKETANDEM_TEST_CLIPARSER_JAR not set, skipping the live cliparser comparison")
	}
	java := os.Getenv("FAKETANDEM_TEST_JAVA")
	if java == "" {
		java = "java"
	}
	jar, err := parity.StartJar(java, jarPath)
	if err != nil {
		t.Fatalf("failed to start cliparser serve: %v", err)
	}
	t.Cleanup(jar.Close)
	return jar
}

// TestParseMatchesJar runs every parse case through the jar and the port.
// With -update-golden it records the jar's answers as the golden file.
func TestParseMatchesJar(t *testing.T) {
	jar := requireJar(t)
	var recorded []parity.ParseResult
	mismatches := 0
	for _, c := range parity.ParseCases() {
		if !keep(t, c.Key) {
			continue
		}
		want, err := jar.JarParse(c)
		if err != nil {
			t.Fatalf("%s: jar: %v", c.Key, err)
		}
		recorded = append(recorded, want)
		if got := parity.NativeParse(c); !reflect.DeepEqual(got, want) {
			mismatches++
			if mismatches <= 50 {
				t.Errorf("%s:\n  go:  %+v (%s)\n  jar: %+v", c.Key, got, parity.NativeParseError(c), want)
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("%d parse cases differ from the jar", mismatches)
	}
	if *updateGolden && os.Getenv("PARITY_FILTER") == "" {
		if err := parity.WriteGolden(parity.ParseGoldenPath(), recorded); err != nil {
			t.Fatal(err)
		}
	}
}

// TestEncodeMatchesJar runs every encode case through the jar and the port.
func TestEncodeMatchesJar(t *testing.T) {
	jar := requireJar(t)
	var recorded []parity.EncodeResult
	mismatches := 0
	for _, c := range parity.EncodeCases() {
		if !keep(t, c.Key) {
			continue
		}
		want, err := jar.JarEncode(c)
		if err != nil {
			t.Fatalf("%s: jar: %v", c.Key, err)
		}
		recorded = append(recorded, want)
		if got := parity.NativeEncode(c); !reflect.DeepEqual(got, want) {
			mismatches++
			if mismatches <= 50 {
				t.Errorf("%s %s %s:\n  go:  %+v (%s)\n  jar: %+v", c.Key, c.Message, c.Params, got, parity.NativeEncodeError(c), want)
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("%d encode cases differ from the jar", mismatches)
	}
	if *updateGolden && os.Getenv("PARITY_FILTER") == "" {
		if err := parity.WriteGolden(parity.EncodeGoldenPath(), recorded); err != nil {
			t.Fatal(err)
		}
	}
}

// TestParseMatchesGolden holds the port to the jar's recorded parse answers.
func TestParseMatchesGolden(t *testing.T) {
	golden, err := parity.ReadParseGolden()
	if err != nil {
		t.Fatalf("no recorded jar answers (regenerate with -update-golden): %v", err)
	}
	cases := parity.ParseCases()
	if len(golden) != len(cases) {
		t.Errorf("%d recorded answers for %d cases; regenerate with -update-golden", len(golden), len(cases))
	}
	mismatches := 0
	for _, c := range cases {
		if !keep(t, c.Key) {
			continue
		}
		want, ok := golden[c.Key]
		if !ok {
			t.Errorf("%s: no recorded answer", c.Key)
			continue
		}
		if got := parity.NativeParse(c); !reflect.DeepEqual(got, want) {
			mismatches++
			if mismatches <= 50 {
				t.Errorf("%s:\n  go:  %+v (%s)\n  jar: %+v", c.Key, got, parity.NativeParseError(c), want)
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("%d parse cases differ from the recorded jar answers", mismatches)
	}
}

// TestEncodeMatchesGolden holds the port to the jar's recorded encode answers.
func TestEncodeMatchesGolden(t *testing.T) {
	golden, err := parity.ReadEncodeGolden()
	if err != nil {
		t.Fatalf("no recorded jar answers (regenerate with -update-golden): %v", err)
	}
	cases := parity.EncodeCases()
	if len(golden) != len(cases) {
		t.Errorf("%d recorded answers for %d cases; regenerate with -update-golden", len(golden), len(cases))
	}
	mismatches := 0
	for _, c := range cases {
		if !keep(t, c.Key) {
			continue
		}
		want, ok := golden[c.Key]
		if !ok {
			t.Errorf("%s: no recorded answer", c.Key)
			continue
		}
		if got := parity.NativeEncode(c); !reflect.DeepEqual(got, want) {
			mismatches++
			if mismatches <= 50 {
				t.Errorf("%s %s %s:\n  go:  %+v (%s)\n  jar: %+v", c.Key, c.Message, c.Params, got, parity.NativeEncodeError(c), want)
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("%d encode cases differ from the recorded jar answers", mismatches)
	}
}
