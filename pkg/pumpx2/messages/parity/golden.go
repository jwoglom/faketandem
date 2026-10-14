package parity

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// GoldenDir is where the jar's recorded answers live.
func GoldenDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata")
}

// ParseGoldenPath and EncodeGoldenPath are the recorded answers, one JSON
// object per case, keyed like the cases.
func ParseGoldenPath() string  { return filepath.Join(GoldenDir(), "parse.jsonl.gz") }
func EncodeGoldenPath() string { return filepath.Join(GoldenDir(), "encode.jsonl.gz") }

// WriteGolden writes one JSON object per row.
func WriteGolden(path string, rows interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := writeGzipLines(f, rows, filepath.Base(path)); err != nil {
		_ = f.Close() // Safe to ignore: the write already failed.
		return err
	}
	return f.Close()
}

func writeGzipLines(f *os.File, rows interface{}, name string) error {
	gz, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		return err
	}
	// A fixed header keeps the file byte-identical across regenerations.
	gz.Name = name
	w := bufio.NewWriter(gz)
	b, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	var items []json.RawMessage
	if err := json.Unmarshal(b, &items); err != nil {
		return err
	}
	for _, item := range items {
		if _, err := w.Write(append(item, '\n')); err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return gz.Close()
}

// ReadParseGolden loads the recorded parse answers by key.
func ReadParseGolden() (map[string]ParseResult, error) {
	out := map[string]ParseResult{}
	err := readGolden(ParseGoldenPath(), func(line []byte) error {
		var r ParseResult
		if err := json.Unmarshal(line, &r); err != nil {
			return err
		}
		out[r.Key] = r
		return nil
	})
	return out, err
}

// ReadEncodeGolden loads the recorded encode answers by key.
func ReadEncodeGolden() (map[string]EncodeResult, error) {
	out := map[string]EncodeResult{}
	err := readGolden(EncodeGoldenPath(), func(line []byte) error {
		var r EncodeResult
		if err := json.Unmarshal(line, &r); err != nil {
			return err
		}
		out[r.Key] = r
		return nil
	})
	return out, err
}

func readGolden(path string, row func([]byte) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }() // Safe to ignore: read-only.
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	sc := bufio.NewScanner(gz)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		if err := row(sc.Bytes()); err != nil {
			return err
		}
	}
	return sc.Err()
}
