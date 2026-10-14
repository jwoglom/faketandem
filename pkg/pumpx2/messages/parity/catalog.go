package parity

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// JarCatalog is what tools/pumpx2catalog/CatalogDump read out of the
// cliparser jar by reflection (testdata/catalog.json): every message class's
// @MessageProps, the fields its toString prints, its public constructors,
// and the Messages enum and OPCODES table. The Go message types are held to
// it, and the parity cases are generated from it.
type JarCatalog struct {
	Characteristics []struct {
		Name string `json:"name"`
		UUID string `json:"uuid"`
	} `json:"characteristics"`
	Messages []struct {
		Name     string `json:"name"`
		Request  string `json:"request"`
		Response string `json:"response"`
	} `json:"messages"`
	Opcodes []struct {
		Characteristic string `json:"characteristic"`
		Opcode         int    `json:"opcode"`
		Class          string `json:"class"`
	} `json:"opcodes"`
	Classes []*JarClass `json:"classes"`
}

// JarClass is one message class as the jar describes it.
type JarClass struct {
	// Name is the class name below com.jwoglom.pumpx2.pump.messages, e.g.
	// "request.currentStatus.ApiVersionRequest".
	Name       string `json:"name"`
	SimpleName string `json:"simpleName"`
	Props      struct {
		OpCode                  int    `json:"opCode"`
		Size                    int    `json:"size"`
		VariableSize            bool   `json:"variableSize"`
		Stream                  bool   `json:"stream"`
		Signed                  bool   `json:"signed"`
		Type                    string `json:"type"`
		Characteristic          string `json:"characteristic"`
		ModifiesInsulinDelivery bool   `json:"modifiesInsulinDelivery"`
		MinAPI                  string `json:"minApi"`
		SupportedDevices        string `json:"supportedDevices"`
		Response                string `json:"response"`
		Request                 string `json:"request"`
	} `json:"props"`
	Fields []struct {
		Name       string `json:"name"`
		Type       string `json:"type"`
		DeclaredIn string `json:"declaredIn"`
	} `json:"fields"`
	Constructors []struct {
		Params []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"params"`
	} `json:"constructors"`
	EmptyToString string `json:"emptyToString"`
}

// IsRequest reports whether the class is a request.
func (c *JarClass) IsRequest() bool { return c.Props.Type == "REQUEST" }

// Package is the Go package under pkg/pumpx2/messages the class's type
// lives in: its pumpX2 package, lower-cased, or for ErrorResponse, which
// pumpX2 keeps at the root, its characteristic's.
func (c *JarClass) Package() string {
	parts := strings.Split(c.Name, ".")
	if len(parts) == 3 {
		return strings.ToLower(parts[1])
	}
	return strings.ToLower(strings.ReplaceAll(c.Props.Characteristic, "_", ""))
}

// ShortJavaType is a Java type as source code in the message's own package
// would write it: java.lang.String is String, and a nested class is
// Outer.Inner.
func ShortJavaType(fqn string) string {
	var b strings.Builder
	word := ""
	flush := func() {
		if i := strings.LastIndex(word, "."); i >= 0 {
			word = word[i+1:]
		}
		b.WriteString(strings.ReplaceAll(word, "$", "."))
		word = ""
	}
	for _, r := range fqn {
		switch r {
		case '<', '>', ',', '[', ']', ' ':
			flush()
			b.WriteRune(r)
		default:
			word += string(r)
		}
	}
	flush()
	return b.String()
}

var (
	catalogOnce sync.Once
	catalog     *JarCatalog
	catalogErr  error
)

// LoadJarCatalog reads testdata/catalog.json.
func LoadJarCatalog() (*JarCatalog, error) {
	catalogOnce.Do(func() {
		var b []byte
		b, catalogErr = os.ReadFile(filepath.Join(ownDir(), "testdata", "catalog.json"))
		if catalogErr != nil {
			return
		}
		catalog = &JarCatalog{}
		catalogErr = json.Unmarshal(b, catalog)
	})
	return catalog, catalogErr
}

// Class finds a class by simple name.
func (c *JarCatalog) Class(simpleName string) (*JarClass, bool) {
	for _, jc := range c.Classes {
		if jc.SimpleName == simpleName {
			return jc, true
		}
	}
	return nil, false
}

func ownDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}

// MessagesDir is pkg/pumpx2/messages.
func MessagesDir() string { return filepath.Dir(ownDir()) }
