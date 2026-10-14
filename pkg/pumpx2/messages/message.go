package messages

import (
	"fmt"
	"math/big"
	"strings"
	"time"
)

// Message is one instance of a pumpX2 message class: its cargo and the fields
// toString() prints, each held already printed the way Java prints it.
type Message struct {
	Class *Class
	Cargo []byte

	fields []string
}

// classImpl is the hand port of one class.
type classImpl struct {
	// init runs what every constructor runs first: the class's field
	// initializers. Nil when it has none.
	init func(m *Message)
	// parse ports parse(byte[] raw). Messages.parse calls it on a fresh no-arg
	// instance. Like every port it fails by panicking with a *JavaException
	// (check, checkf, throw), or with a Go index error where the Java would
	// throw ArrayIndexOutOfBoundsException.
	parse func(m *Message, raw []byte)
	// ctors ports the public constructors, keyed by Constructor.Signature().
	// The no-arg constructor is "" and need only be registered when its body
	// does something. A constructor whose parameters cliparser cannot convert
	// (enums, collections) never runs under encode and need not be ported.
	ctors map[string]func(m *Message, a args)
}

var registered = map[string]*classImpl{}

// register attaches a class's port to the catalog. It panics on a name the
// catalog does not have or a constructor signature the class does not declare,
// so a port cannot drift from the jar it was written against unnoticed.
func register(name string, impl classImpl) {
	c, ok := classesByName[name]
	if !ok {
		panic(fmt.Sprintf("register: no class %s in the pumpX2 catalog", name))
	}
	if _, dup := registered[name]; dup {
		panic(fmt.Sprintf("register: %s registered twice", name))
	}
	for sig := range impl.ctors {
		if sig == "" {
			continue
		}
		found := false
		for _, ctor := range c.Constructors {
			if ctor.Signature() == sig {
				found = true
			}
		}
		if !found {
			panic(fmt.Sprintf("register: %s declares no constructor (%s)", name, sig))
		}
	}
	registered[name] = &impl
	c.impl = &impl
}

// Ported reports whether a class has a hand port.
func (c *Class) Ported() bool { return c.impl != nil }

// newMessage allocates an instance with every field at its Java default and
// runs the class's field initializers.
func newMessage(c *Class) *Message {
	m := &Message{Class: c, fields: make([]string, len(c.Fields))}
	for i, f := range c.Fields {
		m.fields[i] = javaDefault(f.Type)
	}
	if c.impl != nil && c.impl.init != nil {
		c.impl.init(m)
	}
	return m
}

func javaDefault(typ string) string {
	switch typ {
	case "int", "long", "short", "byte":
		return "0"
	case "boolean":
		return "false"
	case "double", "float":
		return "0.0"
	case "char":
		return "\x00"
	default:
		return javaNull
	}
}

// newInstance is Class.newInstance(): the no-arg constructor.
func newInstance(c *Class) (m *Message, err error) {
	m = newMessage(c)
	if c.impl != nil {
		if ctor, ok := c.impl.ctors[""]; ok {
			defer func() {
				if r := recover(); r != nil {
					err = fromPanic(r)
				}
			}()
			ctor(m, nil)
		}
	}
	return m, nil
}

// ToString is what the message's toString() prints:
// SimpleName[field=value,...,cargo={...}].
func (m *Message) ToString() string {
	var b strings.Builder
	b.WriteString(m.Class.SimpleName)
	b.WriteByte('[')
	for i, f := range m.Class.Fields {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(f.Name)
		b.WriteByte('=')
		if f.Name == "cargo" && f.DeclaredIn == "Message" {
			b.WriteString(jBytes(m.Cargo))
			continue
		}
		b.WriteString(m.fields[i])
	}
	b.WriteByte(']')
	return b.String()
}

// set stores a field's printed value. An unknown name is a porting bug.
func (m *Message) set(name, printed string) {
	i, ok := m.Class.fieldIndex[name]
	if !ok {
		panic(fmt.Sprintf("%s has no field %q", m.Class.Name, name))
	}
	m.fields[i] = printed
}

func (m *Message) setInt(name string, v int32)     { m.set(name, jInt(v)) }
func (m *Message) setLong(name string, v int64)    { m.set(name, jLong(v)) }
func (m *Message) setBool(name string, v bool)     { m.set(name, jBool(v)) }
func (m *Message) setBytes(name string, v []byte)  { m.set(name, jBytes(v)) }
func (m *Message) setString(name string, v string) { m.set(name, v) }
func (m *Message) setNull(name string)             { m.set(name, javaNull) }
func (m *Message) setBigInt(name string, v *big.Int) {
	m.set(name, jBigInt(v))
}
func (m *Message) setInstant(name string, v time.Time) { m.set(name, jInstant(v)) }

// setObject stores an object field by its toString(), e.g. an enum constant's
// name or a nested model's own toString.
func (m *Message) setObject(name, toString string) { m.set(name, toString) }

// setEnumSet stores a Set of enum constants. Every such Set in pumpX2 is a
// TreeSet or EnumSet, so it prints in declaration (ordinal) order; callers
// pass the names in that order.
func (m *Message) setEnumSet(name string, names []string) {
	m.set(name, jCollection(names))
}

// parseSelf is a constructor's own call to parse(raw).
func (m *Message) parseSelf(raw []byte) { m.Class.impl.parse(m, raw) }

// fillWithEmptyCargo is Message.fillWithEmptyCargo.
func (m *Message) fillWithEmptyCargo() { m.Cargo = []byte{} }

// removeSignedRequestHmacBytes is Message.removeSignedRequestHmacBytes.
func (m *Message) removeSignedRequestHmacBytes(raw []byte) []byte {
	if m.Class.Props.Signed && len(raw) == m.Class.Props.Size+24 {
		return append([]byte{}, raw[:len(raw)-24]...)
	}
	return raw
}

// size is props().size().
func (m *Message) size() int { return m.Class.Props.Size }

// args are a constructor's arguments after cliparser's conversion: int32 for
// int, int64 for long, int16 for short, int8 for byte, bool, string, []byte,
// float64 for double, float32 for float, and *big.Int for BigInteger.
type args []interface{}

func (a args) Int(i int) int32         { return a[i].(int32) }
func (a args) Long(i int) int64        { return a[i].(int64) }
func (a args) Short(i int) int16       { return a[i].(int16) }
func (a args) Byte(i int) int8         { return a[i].(int8) }
func (a args) Bool(i int) bool         { return a[i].(bool) }
func (a args) Bytes(i int) []byte      { return a[i].([]byte) }
func (a args) Double(i int) float64    { return a[i].(float64) }
func (a args) Float(i int) float32     { return a[i].(float32) }
func (a args) BigInt(i int) *big.Int   { return a[i].(*big.Int) }
func (a args) StringArg(i int) *string { return a[i].(*string) }

// String returns a String argument; cliparser never passes null for one.
func (a args) String(i int) string { return *a.StringArg(i) }

// runParse runs a class's ported parse on a fresh instance.
func runParse(c *Class, raw []byte) (m *Message, err error) {
	if c.impl == nil || c.impl.parse == nil {
		return nil, exception("faketandem.NotPorted", "%s has no Go port of parse()", c.Name)
	}
	m, err = newInstance(c)
	if err != nil {
		return nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			err = fromPanic(r)
		}
	}()
	c.impl.parse(m, raw)
	return m, nil
}

// Parse decodes cargo (signed trailer included, when the message has one)
// into an instance of c, as Messages.parse does once the packets are joined.
func Parse(c *Class, cargo []byte) (*Message, error) {
	return runParse(c, cargo)
}
