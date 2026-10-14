package messages

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ToString is what pumpX2's toString() prints for m, which is how cliparser's
// parse command shows a message and how faketandem reads one: commons-lang's
// ReflectionToStringBuilder in SHORT_PREFIX_STYLE.
//
// The fields printed are those tagged `pumpx2:"<java field name>"`, sorted by
// that name, then those of an embedded message type (pumpX2's superclass),
// then the cargo, as Java prints a class's own fields before its
// superclass's. A field's Java null is a nil pointer, slice or *big.Int.
func ToString(m Message) string {
	return AutoToString(ClassOf(m).Name, m)
}

// AutoToString prints v, a message or a pointer to or value of any other
// struct with tagged fields (a nested model such as RemindersResponse's
// Reminder), as ToString does, under the class name name.
func AutoToString(name string, v interface{}) string {
	val := reflect.ValueOf(v)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	var parts []string
	appendFields(&parts, val)
	if m, ok := v.(Message); ok {
		parts = append(parts, "cargo="+javaBytes(m.Cargo()))
	}
	return name + "[" + strings.Join(parts, ",") + "]"
}

var baseType = reflect.TypeOf(Base{})

type taggedField struct {
	name  string
	value reflect.Value
}

func appendFields(parts *[]string, v reflect.Value) {
	t := v.Type()
	var own []taggedField
	var supers []reflect.Value
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous {
			if f.Type != baseType {
				supers = append(supers, reflect.Indirect(v.Field(i)))
			}
			continue
		}
		if name, ok := f.Tag.Lookup("pumpx2"); ok && name != "-" {
			own = append(own, taggedField{name, v.Field(i)})
		}
	}
	sort.SliceStable(own, func(i, j int) bool { return own[i].name < own[j].name })
	for _, f := range own {
		*parts = append(*parts, f.name+"="+javaValue(f.value))
	}
	for _, s := range supers {
		appendFields(parts, s)
	}
}

const javaNull = "<null>"

var (
	timeType     = reflect.TypeOf(time.Time{})
	stringerType = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
)

// javaValue prints one field value as ReflectionToStringBuilder does.
func javaValue(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Interface:
		if v.IsNil() {
			return javaNull
		}
	}
	switch {
	case v.Type() == timeType:
		return javaInstant(v.Interface().(time.Time))
	case v.Type() == bigIntType:
		return v.Interface().(*big.Int).String()
	case v.Type().Implements(stringerType):
		return v.Interface().(fmt.Stringer).String()
	}
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return javaValue(v.Elem())
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Float32:
		return javaFloatString(v.Float(), 32)
	case reflect.Float64:
		return javaFloatString(v.Float(), 64)
	case reflect.String:
		return v.String()
	case reflect.Slice:
		return javaSlice(v)
	}
	panic(fmt.Sprintf("messages: no Java form for a field of type %s", v.Type()))
}

// javaSlice prints a slice of plain numbers or of slices as a Java array,
// {1,-2}, and any other slice, enums included, as a Java collection, [A, B].
func javaSlice(v reflect.Value) string {
	elem := v.Type().Elem()
	if elem.Kind() == reflect.Uint8 {
		return javaBytes(v.Bytes())
	}
	items := make([]string, v.Len())
	for i := range items {
		items[i] = javaValue(v.Index(i))
	}
	if elem.Implements(stringerType) {
		return "[" + strings.Join(items, ", ") + "]"
	}
	switch elem.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64, reflect.Bool, reflect.Slice:
		return "{" + strings.Join(items, ",") + "}"
	}
	return "[" + strings.Join(items, ", ") + "]"
}

// javaBytes prints a byte[]: signed, as {1,-2,3}.
func javaBytes(b []byte) string {
	if b == nil {
		return javaNull
	}
	parts := make([]string, len(b))
	for i, x := range b {
		parts[i] = strconv.Itoa(int(int8(x)))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// javaInstant prints a java.time.Instant as DateTimeFormatter.ISO_INSTANT
// does: the fraction is left out when zero, else written in groups of three
// digits.
func javaInstant(t time.Time) string {
	t = t.UTC()
	year := t.Year()
	var date string
	switch {
	case year > 9999:
		date = fmt.Sprintf("+%d-%02d-%02d", year, t.Month(), t.Day())
	case year < 0:
		date = fmt.Sprintf("-%04d-%02d-%02d", -year, t.Month(), t.Day())
	default:
		date = fmt.Sprintf("%04d-%02d-%02d", year, t.Month(), t.Day())
	}
	s := fmt.Sprintf("%sT%02d:%02d:%02d", date, t.Hour(), t.Minute(), t.Second())
	if ns := t.Nanosecond(); ns != 0 {
		switch {
		case ns%1000000 == 0:
			s += fmt.Sprintf(".%03d", ns/1000000)
		case ns%1000 == 0:
			s += fmt.Sprintf(".%06d", ns/1000)
		default:
			s += fmt.Sprintf(".%09d", ns)
		}
	}
	return s + "Z"
}

// javaFloatString is Double.toString (bits 64) or Float.toString (bits 32),
// JDK 19 and later: the shortest decimal that reads back to the same value.
func javaFloatString(v float64, bits int) string {
	switch {
	case math.IsNaN(v):
		return "NaN"
	case math.IsInf(v, 1):
		return "Infinity"
	case math.IsInf(v, -1):
		return "-Infinity"
	case v == 0:
		if math.Signbit(v) {
			return "-0.0"
		}
		return "0.0"
	}
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	e := strconv.FormatFloat(v, 'e', -1, bits)
	mant, expStr, _ := strings.Cut(e, "e")
	exp, _ := strconv.Atoi(expStr)
	digits := strings.Replace(mant, ".", "", 1)
	if v >= 1e-3 && v < 1e7 {
		pointAt := exp + 1
		var intPart, frac string
		switch {
		case pointAt <= 0:
			intPart = "0"
			frac = strings.Repeat("0", -pointAt) + digits
		case pointAt >= len(digits):
			intPart = digits + strings.Repeat("0", pointAt-len(digits))
			frac = "0"
		default:
			intPart = digits[:pointAt]
			frac = digits[pointAt:]
		}
		return sign + intPart + "." + frac
	}
	frac := digits[1:]
	if frac == "" {
		frac = "0"
	}
	return sign + digits[:1] + "." + frac + "E" + strconv.Itoa(exp)
}
