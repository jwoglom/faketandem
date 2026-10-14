package messages_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	_ "github.com/jwoglom/faketandem/pkg/pumpx2/messages/all"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func jarCatalog(t *testing.T) *parity.JarCatalog {
	t.Helper()
	cat, err := parity.LoadJarCatalog()
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func jarProps(jc *parity.JarClass) messages.Props {
	p := jc.Props
	simple := func(name string) string { return name[strings.LastIndex(name, ".")+1:] }
	return messages.Props{
		OpCode: p.OpCode, Size: p.Size, VariableSize: p.VariableSize, Stream: p.Stream, Signed: p.Signed,
		Type: messages.MessageType(p.Type), Characteristic: messages.Characteristic(p.Characteristic),
		ModifiesInsulinDelivery: p.ModifiesInsulinDelivery, MinAPI: p.MinAPI, SupportedDevices: p.SupportedDevices,
		Response: simple(p.Response), Request: simple(p.Request),
	}
}

func TestEveryJarClassIsImplementedWithItsPropsAndConstructors(t *testing.T) {
	cat := jarCatalog(t)
	for _, jc := range cat.Classes {
		c, ok := messages.ClassByName(jc.SimpleName)
		if !ok {
			t.Errorf("%s is not implemented", jc.SimpleName)
			continue
		}
		if want := jarProps(jc); c.Props != want {
			t.Errorf("%s props:\n  go:  %+v\n  jar: %+v", jc.SimpleName, c.Props, want)
		}
		var want []string
		for _, ctor := range jc.Constructors {
			if len(ctor.Params) == 0 {
				continue
			}
			params := make([]string, len(ctor.Params))
			for i, p := range ctor.Params {
				params[i] = parity.ShortJavaType(p.Type) + " " + p.Name
			}
			want = append(want, strings.Join(params, ", "))
		}
		var got []string
		for i := range c.Constructors {
			params := make([]string, len(c.Params(i)))
			for j, p := range c.Params(i) {
				params[j] = p.Type + " " + p.Name
			}
			got = append(got, strings.Join(params, ", "))
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s constructors:\n  go:  %q\n  jar: %q", jc.SimpleName, got, want)
		}
	}
	for _, c := range messages.Classes() {
		if _, ok := cat.Class(c.Name); !ok {
			t.Errorf("%s is not a pumpX2 class", c.Name)
		}
	}
}

func TestMessagesEnumMatchesJar(t *testing.T) {
	cat := jarCatalog(t)
	var want []string
	for _, e := range cat.Messages {
		want = append(want, e.Name)
	}
	if got := messages.EnumNames(); !reflect.DeepEqual(got, want) {
		t.Errorf("Messages enum:\n  go:  %v\n  jar: %v", got, want)
	}
}

func TestOpcodeTableMatchesJar(t *testing.T) {
	cat := jarCatalog(t)
	table, err := messages.OpcodeTable()
	if err != nil {
		t.Fatal(err)
	}
	var got, want []string
	for _, e := range table {
		got = append(got, string(e.Characteristic)+"/"+strconv.Itoa(e.Opcode)+"/"+e.Class.Name)
	}
	for _, e := range cat.Opcodes {
		want = append(want, e.Characteristic+"/"+strconv.Itoa(e.Opcode)+"/"+e.Class[strings.LastIndex(e.Class, ".")+1:])
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("OPCODES:\n  go:  %v\n  jar: %v", got, want)
	}
}

// TestFieldsMatchJar holds each type's fields to the ones pumpX2's toString
// prints, by what a no-arg instance with empty cargo prints. A Java null
// that the Go type holds by value (a String, an Instant, an enum no parse
// leaves null) prints as that value's zero instead.
func TestFieldsMatchJar(t *testing.T) {
	cat := jarCatalog(t)
	for _, jc := range cat.Classes {
		c, ok := messages.ClassByName(jc.SimpleName)
		if !ok {
			continue
		}
		m, err := c.New()
		if err != nil {
			t.Errorf("%s(): %v", jc.SimpleName, err)
			continue
		}
		m.SetCargo([]byte{})
		got := splitFields(t, jc.SimpleName, messages.ToString(m))
		want := splitFields(t, jc.SimpleName, jc.EmptyToString)
		if len(got) != len(want) {
			t.Errorf("%s prints:\n  go:  %s\n  jar: %s", jc.SimpleName, messages.ToString(m), jc.EmptyToString)
			continue
		}
		for i := range want {
			if got[i] == want[i] {
				continue
			}
			name := want[i][:strings.Index(want[i], "=")]
			if strings.HasSuffix(want[i], "=<null>") && strings.HasPrefix(got[i], name+"=") && heldByValue(m, name) {
				continue
			}
			t.Errorf("%s field %d: go %s, jar %s", jc.SimpleName, i, got[i], want[i])
		}
	}
}

// splitFields splits Name[a=1,b=[X, Y],c={1,2}] into its top-level fields.
func splitFields(t *testing.T, name, s string) []string {
	t.Helper()
	if !strings.HasPrefix(s, name+"[") || !strings.HasSuffix(s, "]") {
		t.Fatalf("%q is not a %s toString", s, name)
	}
	body := s[len(name)+1 : len(s)-1]
	var out []string
	depth, start := 0, 0
	for i, r := range body {
		switch r {
		case '[', '{':
			depth++
		case ']', '}':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, body[start:i])
				start = i + 1
			}
		}
	}
	return append(out, body[start:])
}

// heldByValue reports whether the field pumpX2 calls name is a Go type with
// no nil.
func heldByValue(m messages.Message, name string) bool {
	var find func(v reflect.Value) (reflect.StructField, bool)
	find = func(v reflect.Value) (reflect.StructField, bool) {
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if f.Tag.Get("pumpx2") == name {
				return f, true
			}
			if f.Anonymous {
				if sf, ok := find(reflect.Indirect(v.Field(i))); ok {
					return sf, true
				}
			}
		}
		return reflect.StructField{}, false
	}
	f, ok := find(reflect.ValueOf(m).Elem())
	if !ok {
		return false
	}
	switch f.Type.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Interface:
		return false
	}
	return true
}

func TestSharedVectorsMatchJar(t *testing.T) {
	v, err := parity.ReadVectors(parity.SharedVectorsPath())
	if err != nil {
		t.Fatal(err)
	}
	parity.CheckVectors(t, v)
}

// TestEveryClassHasItsFiles holds the layout: each class is <name>.go in its
// characteristic's package, beside a <name>_test.go that checks it against
// the jar's recorded answers.
func TestEveryClassHasItsFiles(t *testing.T) {
	cat := jarCatalog(t)
	dir := parity.MessagesDir()
	for _, jc := range cat.Classes {
		base := filepath.Join(dir, jc.Package(), parity.FileName(jc.SimpleName))
		src, err := os.ReadFile(base + ".go")
		if err != nil {
			t.Errorf("%s: %v", jc.SimpleName, err)
			continue
		}
		if strings.Contains(string(src), "PUMPX2-STUB") {
			t.Errorf("%s is still a placeholder", jc.SimpleName)
			continue
		}
		test, err := os.ReadFile(base + "_test.go")
		if err != nil {
			t.Errorf("%s has no test: %v", jc.SimpleName, err)
			continue
		}
		if !strings.Contains(string(test), `parity.CheckClass(t, "`+jc.SimpleName+`")`) {
			t.Errorf("%s_test.go does not check %s against the jar", parity.FileName(jc.SimpleName), jc.SimpleName)
		}
	}
}
