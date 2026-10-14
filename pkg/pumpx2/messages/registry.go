package messages

import (
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strings"
	"sync"
)

// Constructor is one of a message class's public pumpX2 constructors.
// cliparser's encode command picks the first, in declaration order, with as
// many parameters as it was given values, and fills them by name, so a
// message type registers every constructor pumpX2 declares, in its order.
type Constructor struct {
	// Params is the Java parameter list, e.g. "int majorVersion, long rate".
	Params string
	// Func is the Go constructor taking those parameters, with int32 for
	// int, int64 for long, int16 for short, int8 for byte, string for
	// String, []byte for byte[] and *big.Int for BigInteger, and returning
	// the message (and an error, where the Java validates). A parameter of
	// any other Java type is one cliparser cannot build, so encode never
	// calls a constructor that has one.
	Func interface{}
}

// Param is one constructor parameter.
type Param struct {
	Type string
	Name string
}

// Class is a registered message type.
type Class struct {
	// Name is pumpX2's class name, e.g. "ApiVersionResponse".
	Name  string
	Props Props
	// Constructors are the constructors with parameters, in pumpX2's order.
	Constructors []Constructor

	typ    reflect.Type
	params [][]Param
	newFn  reflect.Value
}

// IsRequest reports whether the class is a request.
func (c *Class) IsRequest() bool { return c.Props.Type == Request }

// Params returns constructor i's parameters.
func (c *Class) Params(i int) []Param { return c.params[i] }

// New is the class's no-arg constructor, which is how pumpX2 makes the
// instance it parses into.
func (c *Class) New() (Message, error) {
	if !c.newFn.IsValid() {
		return reflect.New(c.typ).Interface().(Message), nil
	}
	return call(c.newFn, nil)
}

var (
	classesByName = map[string]*Class{}
	classesByType = map[reflect.Type]*Class{}

	tablesOnce sync.Once
	tables     *opcodeTables
)

// Register adds a message type: its pumpX2 class name, a pointer to its zero
// value, its props and its constructors. A Constructor with no Params is the
// no-arg constructor, needed only when it does more than the zero value.
// Register panics on anything malformed, which fails every test of the
// package that registers it.
func Register(name string, prototype Message, props Props, constructors ...Constructor) {
	if tables != nil {
		panic(fmt.Sprintf("messages.Register(%s) after the message tables were built", name))
	}
	if _, dup := classesByName[name]; dup {
		panic(fmt.Sprintf("messages.Register: %s registered twice", name))
	}
	t := reflect.TypeOf(prototype)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("messages.Register(%s): prototype %T is not a pointer to a struct", name, prototype))
	}
	c := &Class{Name: name, Props: props.withDefaults(), typ: t.Elem()}
	for _, ctor := range constructors {
		params, err := parseParams(ctor.Params)
		if err != nil {
			panic(fmt.Sprintf("messages.Register(%s): %v", name, err))
		}
		fn := reflect.ValueOf(ctor.Func)
		if err := checkConstructorFunc(t, params, fn); err != nil {
			panic(fmt.Sprintf("messages.Register(%s): constructor (%s): %v", name, ctor.Params, err))
		}
		if len(params) == 0 {
			c.newFn = fn
			continue
		}
		c.Constructors = append(c.Constructors, ctor)
		c.params = append(c.params, params)
	}
	classesByName[name] = c
	classesByType[t.Elem()] = c
}

func parseParams(s string) ([]Param, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var out []Param
	for _, p := range strings.Split(s, ",") {
		fields := strings.Fields(p)
		if len(fields) != 2 {
			return nil, fmt.Errorf("parameter %q is not \"Type name\"", strings.TrimSpace(p))
		}
		out = append(out, Param{Type: fields[0], Name: fields[1]})
	}
	return out, nil
}

var (
	errorType  = reflect.TypeOf((*error)(nil)).Elem()
	bigIntType = reflect.TypeOf((*big.Int)(nil))
)

// goParamTypes are the Java parameter types cliparser converts a value to,
// as the Go types a constructor takes them as.
var goParamTypes = map[string]reflect.Type{
	"int":        reflect.TypeOf(int32(0)),
	"long":       reflect.TypeOf(int64(0)),
	"short":      reflect.TypeOf(int16(0)),
	"byte":       reflect.TypeOf(int8(0)),
	"boolean":    reflect.TypeOf(false),
	"double":     reflect.TypeOf(float64(0)),
	"float":      reflect.TypeOf(float32(0)),
	"String":     reflect.TypeOf(""),
	"byte[]":     reflect.TypeOf([]byte(nil)),
	"BigInteger": bigIntType,
}

func checkConstructorFunc(ptr reflect.Type, params []Param, fn reflect.Value) error {
	if !fn.IsValid() || fn.IsNil() {
		if len(params) == 0 {
			return nil
		}
		return fmt.Errorf("no func")
	}
	ft := fn.Type()
	if ft.Kind() != reflect.Func {
		return fmt.Errorf("func is a %s", ft)
	}
	if ft.NumIn() != len(params) {
		return fmt.Errorf("func takes %d arguments, not %d", ft.NumIn(), len(params))
	}
	for i, p := range params {
		if want, ok := goParamTypes[p.Type]; ok && ft.In(i) != want {
			return fmt.Errorf("parameter %s is a %s, so the func must take a %s, not a %s", p.Name, p.Type, want, ft.In(i))
		}
	}
	switch {
	case ft.NumOut() == 1 && ft.Out(0) == ptr:
	case ft.NumOut() == 2 && ft.Out(0) == ptr && ft.Out(1) == errorType:
	default:
		return fmt.Errorf("func must return %s or (%s, error), not %s", ptr, ptr, ft)
	}
	return nil
}

// call runs a constructor, turning a panic into an error as Java's would
// propagate an exception.
func call(fn reflect.Value, in []reflect.Value) (m Message, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = recovered(r)
		}
	}()
	out := fn.Call(in)
	if len(out) == 2 && !out[1].IsNil() {
		return nil, out[1].Interface().(error)
	}
	return out[0].Interface().(Message), nil
}

// ClassOf returns the class m is an instance of.
func ClassOf(m Message) *Class {
	t := reflect.TypeOf(m)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	c, ok := classesByType[t]
	if !ok {
		panic(fmt.Sprintf("messages: %T is not a registered message type", m))
	}
	return c
}

// ClassByName looks a class up by its pumpX2 name.
func ClassByName(name string) (*Class, bool) {
	c, ok := classesByName[name]
	return c, ok
}

// Classes returns every registered class, by name.
func Classes() []*Class {
	out := make([]*Class, 0, len(classesByName))
	for _, c := range classesByName {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// IsMessageName reports whether name is a registered class's name, which
// tells a parsed message from the exception text the jar path reads as a
// name when cliparser printed none.
func IsMessageName(name string) bool {
	_, ok := classesByName[name]
	return ok
}

// EnumNames returns the names of pumpX2's Messages enum, in order.
func EnumNames() []string {
	out := make([]string, len(messagesEnum))
	for i, e := range messagesEnum {
		out[i] = e.Name
	}
	return out
}

type opcodeKey struct {
	characteristic Characteristic
	opcode         int
}

type opcodeTables struct {
	opcodes map[opcodeKey]*Class
	err     error
}

// messageTables builds Messages.OPCODES from the enum on first use, once
// every message package has registered.
func messageTables() (*opcodeTables, error) {
	tablesOnce.Do(func() { tables = buildTables() })
	return tables, tables.err
}

func buildTables() *opcodeTables {
	t := &opcodeTables{opcodes: map[opcodeKey]*Class{}}
	var missing []string
	class := func(name string) *Class {
		c, ok := classesByName[name]
		if !ok {
			missing = append(missing, name)
		}
		return c
	}
	put := func(c *Class) {
		if c != nil {
			t.opcodes[opcodeKey{c.Props.Characteristic, c.Props.OpCode}] = c
		}
	}
	for _, e := range messagesEnum {
		put(class(e.Request))
		put(class(e.Response))
	}
	// pumpX2's canonical classes for the CONTROL_STREAM opcodes several
	// stream responses share.
	put(class("DetectingCartridgeStateStreamResponse"))
	put(class("ExitFillTubingModeStateStreamResponse"))
	if errorResponse := class("ErrorResponse"); errorResponse != nil {
		for _, ch := range Characteristics {
			key := opcodeKey{ch, errorResponse.Props.OpCode}
			if _, ok := t.opcodes[key]; !ok {
				t.opcodes[key] = errorResponse
			}
		}
	}
	if len(missing) > 0 {
		t.err = fmt.Errorf("pumpX2 message classes %v are not linked in: import github.com/jwoglom/faketandem/pkg/pumpx2/messages/all", missing)
	}
	return t
}

// OpcodeEntry is one row of pumpX2's Messages.OPCODES table.
type OpcodeEntry struct {
	Characteristic Characteristic
	Opcode         int
	Class          *Class
}

// OpcodeTable returns Messages.OPCODES, ordered by characteristic and opcode.
func OpcodeTable() ([]OpcodeEntry, error) {
	t, err := messageTables()
	if err != nil {
		return nil, err
	}
	order := map[Characteristic]int{}
	for i, c := range Characteristics {
		order[c] = i
	}
	out := make([]OpcodeEntry, 0, len(t.opcodes))
	for k, c := range t.opcodes {
		out = append(out, OpcodeEntry{Characteristic: k.characteristic, Opcode: k.opcode, Class: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Characteristic != out[j].Characteristic {
			return order[out[i].Characteristic] < order[out[j].Characteristic]
		}
		return out[i].Opcode < out[j].Opcode
	})
	return out, nil
}

// FromOpcode is Messages.fromOpcode: the class an opcode on a characteristic
// is.
func FromOpcode(opcode int, characteristic Characteristic) (*Class, bool) {
	t, err := messageTables()
	if err != nil {
		return nil, false
	}
	c, ok := t.opcodes[opcodeKey{characteristic, opcode}]
	return c, ok
}

// possibleCharacteristics is Messages.findPossibleCharacteristicsForOpcode.
func possibleCharacteristics(t *opcodeTables, opcode int) map[Characteristic]bool {
	out := map[Characteristic]bool{}
	for k := range t.opcodes {
		if k.opcode == opcode {
			out[k.characteristic] = true
		}
	}
	return out
}

// lookupEncodeClass is how cliparser's encode resolves a message name: the
// Messages enum in declaration order, matching the constant name (which means
// the request), then the request's class name, then the response's.
func lookupEncodeClass(messageName string) (*Class, bool) {
	for _, e := range messagesEnum {
		switch messageName {
		case e.Name, e.Request:
			return ClassByName(e.Request)
		case e.Response:
			return ClassByName(e.Response)
		}
	}
	return nil, false
}
