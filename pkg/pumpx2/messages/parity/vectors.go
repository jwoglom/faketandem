package parity

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// ParseVector is a parse case and what the jar printed for it.
type ParseVector struct {
	ParseCase
	Jar ParseResult `json:"jar"`
}

// EncodeVector is an encode case and what the jar printed for it.
type EncodeVector struct {
	EncodeCase
	Jar EncodeResult `json:"jar"`
}

// Vectors are recorded jar answers. A message class's are in its package's
// testdata/<Class>.jsonl; the cases that belong to no one class (unknown
// opcodes and names, a runt, the Messages enum's names) are in
// pkg/pumpx2/messages/testdata/messages.jsonl. Each line is one JSON object,
// {"parse": ...} or {"encode": ...}.
type Vectors struct {
	Parse  []ParseVector
	Encode []EncodeVector
}

type vectorLine struct {
	Parse  *ParseVector  `json:"parse,omitempty"`
	Encode *EncodeVector `json:"encode,omitempty"`
}

// FileName is the Go file name, without its extension, of a message class:
// its name in snake case, e.g. cgm_status_v2_response for
// CgmStatusV2Response.
func FileName(className string) string {
	r := []rune(className)
	var b strings.Builder
	for i, c := range r {
		if i > 0 && unicode.IsUpper(c) {
			prev := r[i-1]
			nextLower := i+1 < len(r) && unicode.IsLower(r[i+1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
				b.WriteByte('_')
			}
		}
		b.WriteRune(unicode.ToLower(c))
	}
	return b.String()
}

// ClassVectorsPath is where a class's vectors are recorded.
func ClassVectorsPath(c *JarClass) string {
	return filepath.Join(MessagesDir(), c.Package(), "testdata", c.SimpleName+".jsonl")
}

// SharedVectorsPath is where the vectors no one class owns are recorded.
func SharedVectorsPath() string {
	return filepath.Join(MessagesDir(), "testdata", "messages.jsonl")
}

// ReadVectors reads a vectors file.
func ReadVectors(path string) (*Vectors, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	v := &Vectors{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for n := 1; sc.Scan(); n++ {
		var line vectorLine
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, n, err)
		}
		switch {
		case line.Parse != nil:
			v.Parse = append(v.Parse, *line.Parse)
		case line.Encode != nil:
			v.Encode = append(v.Encode, *line.Encode)
		default:
			return nil, fmt.Errorf("%s:%d: neither a parse nor an encode vector", path, n)
		}
	}
	return v, sc.Err()
}

// Sort puts the vectors in key order, as they are recorded.
func (v *Vectors) Sort() {
	sort.SliceStable(v.Parse, func(i, j int) bool { return v.Parse[i].Key < v.Parse[j].Key })
	sort.SliceStable(v.Encode, func(i, j int) bool { return v.Encode[i].Key < v.Encode[j].Key })
}

// WriteVectors writes a vectors file: the parse vectors, then the encode
// vectors, each in key order.
func WriteVectors(path string, v *Vectors) error {
	v.Sort()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for i := range v.Parse {
		if err := enc.Encode(vectorLine{Parse: &v.Parse[i]}); err != nil {
			return err
		}
	}
	for i := range v.Encode {
		if err := enc.Encode(vectorLine{Encode: &v.Encode[i]}); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644) //nolint:gosec // test data, read by the tests
}

// VectorPaths lists every recorded vectors file: each class's, then the
// shared one.
func VectorPaths(cat *JarCatalog) []string {
	var out []string
	for _, c := range cat.Classes {
		out = append(out, ClassVectorsPath(c))
	}
	return append(out, SharedVectorsPath())
}

// AllVectors reads every recorded vector.
func AllVectors() (*Vectors, error) {
	cat, err := LoadJarCatalog()
	if err != nil {
		return nil, err
	}
	all := &Vectors{}
	for _, path := range VectorPaths(cat) {
		v, err := ReadVectors(path)
		if err != nil {
			return nil, err
		}
		all.Parse = append(all.Parse, v.Parse...)
		all.Encode = append(all.Encode, v.Encode...)
	}
	return all, nil
}

// ParseOwner is the class a parse case was generated for, or "" for one no
// one class owns.
func ParseOwner(c ParseCase) string {
	parts := strings.Split(c.Key, "/")
	if len(parts) == 4 {
		return parts[2]
	}
	return ""
}

// EncodeOwner is the class an encode case was generated for, or "" for one
// no one class owns.
func EncodeOwner(cat *JarCatalog, c EncodeCase) string {
	name := strings.SplitN(c.Key, "/", 2)[0]
	if _, ok := cat.Class(name); ok {
		return name
	}
	return ""
}
