// Package messages is a Go port of the pumpX2 message library, as the
// cliparser jar drives it: it resolves, decodes and prints request messages
// exactly as `cliparser parse` does, and builds messages exactly as
// `cliparser encode` does, so faketandem needs no JVM.
//
// The catalog (catalog_gen.go) is generated from the jar by
// tools/pumpx2catalog/regen.sh. Each class's parse() and constructors are
// ported by hand in the msg_*.go files, and the parity tests hold every one of
// them to the jar's own output.
package messages

import (
	"fmt"
	"sort"
	"strings"
)

// Props mirrors pumpX2's @MessageProps annotation.
type Props struct {
	OpCode                  int
	Size                    int
	VariableSize            bool
	Stream                  bool
	Signed                  bool
	Type                    string // "REQUEST" or "RESPONSE"
	Characteristic          string // a Characteristic enum constant name, e.g. "CURRENT_STATUS"
	ModifiesInsulinDelivery bool
	MinAPI                  string
	SupportedDevices        string
	Response                string // class name, for a request
	Request                 string // class name, for a response
}

// Field is one field toString() prints, in ReflectionToStringBuilder's order:
// the class's own fields sorted by name, then each superclass's.
type Field struct {
	Name       string
	Type       string // Java generic type name, e.g. "int", "byte[]", "java.util.Set<...>"
	DeclaredIn string
}

// Param is one constructor parameter.
type Param struct {
	Name string
	Type string
}

// Constructor is one public constructor, in getConstructors() order, which is
// the order cliparser's encode tries them in.
type Constructor struct {
	Params []Param
}

// Signature is the constructor's comma-joined parameter types, the key a
// ported constructor is registered under.
func (c Constructor) Signature() string {
	types := make([]string, len(c.Params))
	for i, p := range c.Params {
		types[i] = p.Type
	}
	return strings.Join(types, ",")
}

// Class describes one pumpX2 message class.
type Class struct {
	// Name is the class name without pumpX2's "com.jwoglom.pumpx2.pump.messages."
	// prefix, e.g. "request.currentStatus.ApiVersionRequest".
	Name         string
	SimpleName   string
	Props        Props
	Fields       []Field
	Constructors []Constructor
	// EmptyToString is what a no-arg instance with empty cargo prints, which the
	// tests hold the Go defaults to.
	EmptyToString      string
	EmptyToStringError string

	fieldIndex map[string]int
	impl       *classImpl
}

// IsRequest reports whether the class is a request message.
func (c *Class) IsRequest() bool {
	return c.Props.Type == "REQUEST"
}

// Field returns the named field, if the class prints one by that name.
func (c *Class) Field(name string) (Field, bool) {
	i, ok := c.fieldIndex[name]
	if !ok {
		return Field{}, false
	}
	return c.Fields[i], true
}

type characteristicInfo struct {
	Name string
	UUID string
}

type enumEntry struct {
	Name     string
	Request  string
	Response string
}

type opcodeEntry struct {
	Characteristic string
	Opcode         int
	Class          string
}

// OpcodeEntry is one row of pumpX2's Messages.OPCODES table.
type OpcodeEntry struct {
	Characteristic string
	Opcode         int
	Class          *Class
}

// OpcodeTable returns Messages.OPCODES, ordered by characteristic and opcode.
func OpcodeTable() []OpcodeEntry {
	out := make([]OpcodeEntry, len(catalogOpcodes))
	for i, o := range catalogOpcodes {
		out[i] = OpcodeEntry{Characteristic: o.Characteristic, Opcode: o.Opcode, Class: classesByName[o.Class]}
	}
	return out
}

// EnumNames returns the Messages enum's constant names, in declaration order.
func EnumNames() []string {
	out := make([]string, len(catalogMessages))
	for i, e := range catalogMessages {
		out[i] = e.Name
	}
	return out
}

// CharacteristicNames returns pumpX2's Characteristic constants, in order.
func CharacteristicNames() []string {
	out := make([]string, len(catalogCharacteristics))
	for i, c := range catalogCharacteristics {
		out[i] = c.Name
	}
	return out
}

type opcodeKey struct {
	characteristic string
	opcode         int
}

var (
	classesByName   = map[string]*Class{}
	opcodeTable     = map[opcodeKey]*Class{}
	characteristics = map[string]characteristicInfo{}
)

func init() {
	for _, c := range catalogClasses {
		c.fieldIndex = make(map[string]int, len(c.Fields))
		for i, f := range c.Fields {
			c.fieldIndex[f.Name] = i
		}
		classesByName[c.Name] = c
	}
	for _, o := range catalogOpcodes {
		opcodeTable[opcodeKey{o.Characteristic, o.Opcode}] = mustClass(o.Class)
	}
	for _, ch := range catalogCharacteristics {
		characteristics[ch.Name] = ch
	}
	for _, e := range catalogMessages {
		mustClass(e.Request)
		mustClass(e.Response)
	}
}

func mustClass(name string) *Class {
	c, ok := classesByName[name]
	if !ok {
		panic(fmt.Sprintf("pumpX2 catalog names unknown class %s", name))
	}
	return c
}

// Classes returns every class in the catalog, in a stable order.
func Classes() []*Class {
	out := append([]*Class(nil), catalogClasses...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// IsMessageName reports whether name is some class's simple name, which tells
// a parsed message from the exception text the jar path reads as a name when
// cliparser printed none.
func IsMessageName(name string) bool {
	for _, c := range catalogClasses {
		if c.SimpleName == name {
			return true
		}
	}
	return false
}

// ClassByName looks a class up by its prefix-less name.
func ClassByName(name string) (*Class, bool) {
	c, ok := classesByName[name]
	return c, ok
}

// FromOpcode is Messages.fromOpcode: the class the OPCODES table maps a
// characteristic and opcode to.
func FromOpcode(opcode int, characteristic string) (*Class, bool) {
	c, ok := opcodeTable[opcodeKey{characteristic, opcode}]
	return c, ok
}

// possibleCharacteristics is Messages.findPossibleCharacteristicsForOpcode.
func possibleCharacteristics(opcode int) map[string]bool {
	out := map[string]bool{}
	for k := range opcodeTable {
		if k.opcode == opcode {
			out[k.characteristic] = true
		}
	}
	return out
}

// CharacteristicUUID returns the UUID pumpX2 gives a Characteristic constant.
func CharacteristicUUID(name string) (string, bool) {
	c, ok := characteristics[name]
	return c.UUID, ok
}

// lookupEncodeClass is how cliparser's encode resolves a message name: the
// Messages enum in declaration order, matching the constant name (which means
// the request), then the request's simple name, then the response's.
func lookupEncodeClass(messageName string) (*Class, bool) {
	for _, e := range catalogMessages {
		req := classesByName[e.Request]
		resp := classesByName[e.Response]
		switch messageName {
		case e.Name, req.SimpleName:
			return req, true
		case resp.SimpleName:
			return resp, true
		}
	}
	return nil, false
}
