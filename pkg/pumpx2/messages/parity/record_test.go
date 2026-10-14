package parity_test

import (
	"flag"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

var updateGolden = flag.Bool("update-golden", false, "rewrite the recorded jar answers from FAKETANDEM_TEST_CLIPARSER_JAR")

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

// TestJarAnswersMatchRecorded runs every case generated from catalog.json
// through the jar and checks the recorded answers are what it says today;
// with -update-golden it records them instead, each under the class it
// parsed as or built.
func TestJarAnswersMatchRecorded(t *testing.T) {
	jar := requireJar(t)
	cat, err := parity.LoadJarCatalog()
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]*parity.Vectors{}
	file := func(owner string) *parity.Vectors {
		path := parity.SharedVectorsPath()
		if c, ok := cat.Class(owner); ok {
			path = parity.ClassVectorsPath(c)
		}
		if files[path] == nil {
			files[path] = &parity.Vectors{}
		}
		return files[path]
	}
	for _, c := range parity.GenerateParseCases(cat) {
		got, err := jar.JarParse(c)
		if err != nil {
			t.Fatalf("%s: jar: %v", c.Key, err)
		}
		owner := parity.ParseOwner(c)
		if got.OK {
			owner = got.ToString[:strings.Index(got.ToString, "[")]
		}
		v := file(owner)
		v.Parse = append(v.Parse, parity.ParseVector{ParseCase: c, Jar: got})
	}
	for _, c := range parity.GenerateEncodeCases(cat) {
		got, err := jar.JarEncode(c)
		if err != nil {
			t.Fatalf("%s: jar: %v", c.Key, err)
		}
		v := file(parity.EncodeOwner(cat, c))
		v.Encode = append(v.Encode, parity.EncodeVector{EncodeCase: c, Jar: got})
	}
	for _, path := range parity.VectorPaths(cat) {
		want := files[path]
		if want == nil {
			want = &parity.Vectors{}
		}
		if *updateGolden {
			if err := parity.WriteVectors(path, want); err != nil {
				t.Fatal(err)
			}
			continue
		}
		recorded, err := parity.ReadVectors(path)
		if err != nil {
			t.Errorf("%v (record with -update-golden)", err)
			continue
		}
		want.Sort()
		if !reflect.DeepEqual(normalize(recorded), normalize(want)) {
			t.Errorf("%s is not what the jar answers today; re-record with -update-golden", path)
		}
	}
}

func normalize(v *parity.Vectors) *parity.Vectors {
	if len(v.Parse) == 0 {
		v.Parse = nil
	}
	if len(v.Encode) == 0 {
		v.Encode = nil
	}
	return v
}

// TestRecordedVectorsAreTheGeneratedCases holds the recorded inputs to the
// cases catalog.json generates, so a change to either is re-recorded.
func TestRecordedVectorsAreTheGeneratedCases(t *testing.T) {
	cat, err := parity.LoadJarCatalog()
	if err != nil {
		t.Fatal(err)
	}
	all, err := parity.AllVectors()
	if err != nil {
		t.Fatalf("%v (record with -update-golden)", err)
	}
	recordedParse := map[string]parity.ParseCase{}
	for _, v := range all.Parse {
		recordedParse[v.Key] = v.ParseCase
	}
	generatedParse := parity.GenerateParseCases(cat)
	if len(recordedParse) != len(generatedParse) {
		t.Errorf("%d parse vectors recorded for %d cases", len(recordedParse), len(generatedParse))
	}
	for _, c := range generatedParse {
		if got, ok := recordedParse[c.Key]; !ok || !reflect.DeepEqual(got, c) {
			t.Errorf("parse case %s is not recorded as generated", c.Key)
		}
	}
	recordedEncode := map[string]parity.EncodeCase{}
	for _, v := range all.Encode {
		recordedEncode[v.Key] = v.EncodeCase
	}
	generatedEncode := parity.GenerateEncodeCases(cat)
	if len(recordedEncode) != len(generatedEncode) {
		t.Errorf("%d encode vectors recorded for %d cases", len(recordedEncode), len(generatedEncode))
	}
	for _, c := range generatedEncode {
		if got, ok := recordedEncode[c.Key]; !ok || got != c {
			t.Errorf("encode case %s is not recorded as generated", c.Key)
		}
	}
}

func TestFileName(t *testing.T) {
	for in, want := range map[string]string{
		"ApiVersionRequest":           "api_version_request",
		"CGMHardwareInfoRequest":      "cgm_hardware_info_request",
		"Jpake1aRequest":              "jpake1a_request",
		"CgmStatusV2Response":         "cgm_status_v2_response",
		"IDPSegmentResponse":          "idp_segment_response",
		"UnknownMobiOpcode110Request": "unknown_mobi_opcode110_request",
	} {
		if got := parity.FileName(in); got != want {
			t.Errorf("FileName(%s) = %s, want %s", in, got, want)
		}
	}
}
