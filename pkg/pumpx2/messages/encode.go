package messages

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

// Encoded is what cliparser's `encode` command prints.
type Encoded struct {
	MessageName        string // the class's simple name
	TxID               string
	Packets            []string
	CharacteristicName string
	CharacteristicUUID string
	Message            *Message
}

// EncodeParamsJSON renders encode params as the single JSON object argument
// cliparser's "encode" command takes.
func EncodeParamsJSON(params map[string]interface{}) (string, error) {
	if len(params) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Encode is cliparser's `encode <txId> <messageName> <params>` command, given
// the params as faketandem's runners render them.
func Encode(txID int, messageName string, params map[string]interface{}, env Env) (*Encoded, error) {
	paramsJSON, err := EncodeParamsJSON(params)
	if err != nil {
		return nil, err
	}
	return EncodeJSON(strconv.Itoa(txID), messageName, paramsJSON, env)
}

var jsonKeyPattern = regexp.MustCompile(`"([^"]+)"\s*:`)

// EncodeJSON is Main.encode(String txId, String messageName, String json).
func EncodeJSON(txID, messageName, paramsJSON string, env Env) (*Encoded, error) {
	if _, err := env.timeSinceReset(); err != nil {
		return nil, err
	}
	if paramsJSON == "" {
		paramsJSON = "{}"
	}
	if strings.HasPrefix(strings.TrimSpace(paramsJSON), "[") {
		return encodePositional(txID, messageName, paramsJSON, env)
	}

	values, err := parseJSONObject(paramsJSON)
	if err != nil {
		return nil, err
	}
	var orderedKeys []string
	for _, m := range jsonKeyPattern.FindAllStringSubmatch(paramsJSON, -1) {
		orderedKeys = append(orderedKeys, m[1])
	}

	c, ok := lookupEncodeClass(messageName)
	if !ok {
		return nil, exception("EncodeException", "Unknown message name: %s", messageName)
	}

	for _, ctor := range c.Constructors {
		if len(ctor.Params) != len(values) {
			continue
		}
		in := make([]jsonValue, len(ctor.Params))
		for i, p := range ctor.Params {
			if v, ok := values[p.Name]; ok {
				in[i] = v
			} else if i < len(orderedKeys) {
				v, ok := values[orderedKeys[i]]
				if !ok {
					return nil, exception("org.json.JSONException", "JSONObject[%q] not found.", orderedKeys[i])
				}
				in[i] = v
			} else {
				in[i] = jsonValue{kind: jsonAbsent}
			}
		}
		return construct(c, ctor, in, txID, env)
	}
	return nil, exception("EncodeException", "Unable to build message %s with params %s: no constructor was found with %d parameters", messageName, paramsJSON, len(values))
}

func encodePositional(txID, messageName, paramsJSON string, env Env) (*Encoded, error) {
	var raw []json.RawMessage
	dec := json.NewDecoder(strings.NewReader(paramsJSON))
	if err := dec.Decode(&raw); err != nil {
		return nil, exception("org.json.JSONException", "%v", err)
	}
	values := make([]jsonValue, len(raw))
	for i, r := range raw {
		v, err := toJSONValue(r)
		if err != nil {
			return nil, err
		}
		values[i] = v
	}
	c, ok := lookupEncodeClass(messageName)
	if !ok {
		return nil, exception("EncodeException", "Unknown message name: %s", messageName)
	}
	for _, ctor := range c.Constructors {
		if len(ctor.Params) == len(values) {
			return construct(c, ctor, values, txID, env)
		}
	}
	return nil, exception("EncodeException", "Unable to build message %s with params %s: no constructor was found with %d parameters", messageName, paramsJSON, len(values))
}

// construct runs one constructor and packetizes the result, as Main.encode does.
func construct(c *Class, ctor Constructor, in []jsonValue, txIDStr string, env Env) (*Encoded, error) {
	m := newMessage(c)
	if len(ctor.Params) == 0 {
		if err := runCtor(c, "", m, nil); err != nil {
			return nil, err
		}
		m.fillWithEmptyCargo()
	} else {
		converted := make(args, len(in))
		for i, p := range ctor.Params {
			v, err := convertToParameterType(in[i], p.Type)
			if err != nil {
				return nil, err
			}
			converted[i] = v
		}
		if err := runCtor(c, ctor.Signature(), m, converted); err != nil {
			return nil, err
		}
	}

	txInt, err := strconv.ParseInt(txIDStr, 10, 32)
	if err != nil {
		return nil, exception("java.lang.NumberFormatException", "For input string: %q", txIDStr)
	}
	txID := byte(txInt)
	packets, err := tronPackets(m, txID, env)
	if err != nil {
		return nil, err
	}
	out := &Encoded{
		MessageName:        c.SimpleName,
		TxID:               strconv.Itoa(int(int8(txID))),
		CharacteristicName: c.Props.Characteristic,
		Message:            m,
	}
	out.CharacteristicUUID, _ = CharacteristicUUID(c.Props.Characteristic)
	for _, p := range packets {
		out.Packets = append(out.Packets, hex.EncodeToString(p))
	}
	return out, nil
}

func runCtor(c *Class, sig string, m *Message, a args) (err error) {
	var fn func(*Message, args)
	if c.impl != nil {
		fn = c.impl.ctors[sig]
	}
	if fn == nil {
		if sig == "" {
			return nil
		}
		return exception("faketandem.NotPorted", "%s(%s) has no Go port", c.Name, sig)
	}
	defer func() {
		if r := recover(); r != nil {
			err = illegalArgument("Unable to build message %s: %v", c.SimpleName, fromPanic(r))
		}
	}()
	fn(m, a)
	return nil
}

// --- org.json values ---

type jsonKind int

const (
	jsonAbsent jsonKind = iota
	jsonNull
	jsonBool
	jsonString
	jsonNumber
	jsonArray
	jsonObject
)

// jsonValue is a value as org.json hands it to cliparser.
type jsonValue struct {
	kind jsonKind
	b    bool
	s    string
	n    javaNumber
	raw  string
}

func parseJSONObject(text string) (map[string]jsonValue, error) {
	var raw map[string]json.RawMessage
	dec := json.NewDecoder(strings.NewReader(text))
	if err := dec.Decode(&raw); err != nil {
		return nil, exception("org.json.JSONException", "%v", err)
	}
	out := make(map[string]jsonValue, len(raw))
	for k, r := range raw {
		v, err := toJSONValue(r)
		if err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, nil
}

func toJSONValue(r json.RawMessage) (jsonValue, error) {
	t := bytes.TrimSpace(r)
	if len(t) == 0 {
		return jsonValue{kind: jsonNull}, nil
	}
	switch t[0] {
	case 'n':
		return jsonValue{kind: jsonNull, raw: "null"}, nil
	case 't', 'f':
		return jsonValue{kind: jsonBool, b: t[0] == 't', raw: string(t)}, nil
	case '"':
		var s string
		if err := json.Unmarshal(t, &s); err != nil {
			return jsonValue{}, exception("org.json.JSONException", "%v", err)
		}
		return jsonValue{kind: jsonString, s: s, raw: string(t)}, nil
	case '[':
		return jsonValue{kind: jsonArray, raw: string(t)}, nil
	case '{':
		return jsonValue{kind: jsonObject, raw: string(t)}, nil
	}
	n, err := stringToNumber(string(t))
	if err != nil {
		// org.json keeps an unparseable bare token as a String.
		return jsonValue{kind: jsonString, s: string(t), raw: string(t)}, nil
	}
	return jsonValue{kind: jsonNumber, n: n, raw: string(t)}, nil
}

// javaNumber is the Number org.json makes of a JSON number literal.
type javaNumber struct {
	class string // "Integer", "Long", "BigInteger", "BigDecimal", "Double"
	i     int64
	big   *big.Int
	dec   decimal
	f     float64
}

// decimal is a BigDecimal: unscaled * 10^-scale.
type decimal struct {
	unscaled *big.Int
	scale    int
}

// stringToNumber is JSONObject.stringToNumber (org.json 20230618).
func stringToNumber(val string) (javaNumber, error) {
	notNumber := exception("java.lang.NumberFormatException", "val [%s] is not a valid number.", val)
	if val == "" || (val[0] != '-' && (val[0] < '0' || val[0] > '9')) {
		return javaNumber{}, notNumber
	}
	if strings.ContainsAny(val, ".eE") || val == "-0" {
		return decimalNumber(val)
	}
	if (val[0] == '0' && len(val) > 1) || (val[0] == '-' && len(val) > 2 && val[1] == '0') {
		return javaNumber{}, notNumber
	}
	bi, ok := new(big.Int).SetString(val, 10)
	if !ok {
		return javaNumber{}, notNumber
	}
	switch {
	case bitLength(bi) <= 31:
		return javaNumber{class: "Integer", i: bi.Int64()}, nil
	case bitLength(bi) <= 63:
		return javaNumber{class: "Long", i: bi.Int64()}, nil
	default:
		return javaNumber{class: "BigInteger", big: bi}, nil
	}
}

// decimalNumber is the decimal-notation branch of stringToNumber: a
// BigDecimal, except that a negative zero is kept as a Double.
func decimalNumber(val string) (javaNumber, error) {
	d, err := parseDecimal(val)
	if err != nil {
		return javaNumber{}, err
	}
	if val[0] == '-' && d.unscaled.Sign() == 0 {
		return javaNumber{class: "Double", f: math.Copysign(0, -1)}, nil
	}
	return javaNumber{class: "BigDecimal", dec: d}, nil
}

// bitLength is BigInteger.bitLength: excluding the sign bit, so -2^31 is 31.
func bitLength(b *big.Int) int {
	if b.Sign() >= 0 {
		return b.BitLen()
	}
	return new(big.Int).Not(b).BitLen()
}

// parseDecimal is new BigDecimal(String).
func parseDecimal(s string) (decimal, error) {
	bad := exception("java.lang.NumberFormatException", "Character array is missing \"exponent\" mark 'e' or 'E'.")
	mant, exp := s, 0
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		mant = s[:i]
		e, err := strconv.Atoi(strings.TrimPrefix(s[i+1:], "+"))
		if err != nil {
			return decimal{}, bad
		}
		exp = e
	}
	scale := 0
	if i := strings.IndexByte(mant, '.'); i >= 0 {
		scale = len(mant) - i - 1
		mant = mant[:i] + mant[i+1:]
	}
	u, ok := new(big.Int).SetString(mant, 10)
	if !ok {
		return decimal{}, bad
	}
	return decimal{unscaled: u, scale: scale - exp}, nil
}

func (d decimal) truncated() *big.Int {
	if d.scale <= 0 {
		return new(big.Int).Mul(d.unscaled, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-d.scale)), nil))
	}
	return new(big.Int).Quo(d.unscaled, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(d.scale)), nil))
}

func (d decimal) float64() float64 {
	r := new(big.Rat).SetInt(d.unscaled)
	p := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(abs(d.scale))), nil))
	if d.scale > 0 {
		r.Quo(r, p)
	} else {
		r.Mul(r, p)
	}
	f, _ := r.Float64()
	return f
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

// String is BigDecimal.toString.
func (d decimal) String() string {
	coeff := new(big.Int).Abs(d.unscaled).String()
	sign := ""
	if d.unscaled.Sign() < 0 {
		sign = "-"
	}
	adjusted := -d.scale + len(coeff) - 1
	if d.scale >= 0 && adjusted >= -6 {
		if d.scale == 0 {
			return sign + coeff
		}
		if len(coeff) > d.scale {
			return sign + coeff[:len(coeff)-d.scale] + "." + coeff[len(coeff)-d.scale:]
		}
		return sign + "0." + strings.Repeat("0", d.scale-len(coeff)) + coeff
	}
	out := coeff[:1]
	if len(coeff) > 1 {
		out += "." + coeff[1:]
	}
	expSign := "+"
	if adjusted < 0 {
		expSign = "-"
	}
	return sign + out + "E" + expSign + strconv.Itoa(abs(adjusted))
}

// lowBits returns x mod 2^bits as a two's-complement int64, which is how
// BigInteger.intValue/longValue narrow.
func lowBits(x *big.Int, bits uint) int64 {
	mod := new(big.Int).Lsh(big.NewInt(1), bits)
	r := new(big.Int).Mod(x, mod)
	if r.Cmp(new(big.Int).Rsh(mod, 1)) >= 0 {
		r.Sub(r, mod)
	}
	return r.Int64()
}

// javaD2L is Java's (long) cast of a double: NaN to 0, saturating.
func javaD2L(f float64) int64 {
	switch {
	case math.IsNaN(f):
		return 0
	case f >= math.MaxInt64:
		return math.MaxInt64
	case f <= math.MinInt64:
		return math.MinInt64
	}
	return int64(f)
}

// javaD2I is Java's (int) cast of a double.
func javaD2I(f float64) int32 {
	switch {
	case math.IsNaN(f):
		return 0
	case f >= math.MaxInt32:
		return math.MaxInt32
	case f <= math.MinInt32:
		return math.MinInt32
	}
	return int32(f)
}

func (n javaNumber) intValue() int32 {
	switch n.class {
	case "Integer", "Long":
		return int32(n.i)
	case "BigInteger":
		return int32(lowBits(n.big, 32))
	case "BigDecimal":
		return int32(lowBits(n.dec.truncated(), 32))
	default:
		return javaD2I(n.f)
	}
}

func (n javaNumber) longValue() int64 {
	switch n.class {
	case "Integer", "Long":
		return n.i
	case "BigInteger":
		return lowBits(n.big, 64)
	case "BigDecimal":
		return lowBits(n.dec.truncated(), 64)
	default:
		return javaD2L(n.f)
	}
}

func (n javaNumber) doubleValue() float64 {
	switch n.class {
	case "Integer", "Long":
		return float64(n.i)
	case "BigInteger":
		f, _ := new(big.Float).SetInt(n.big).Float64()
		return f
	case "BigDecimal":
		return n.dec.float64()
	default:
		return n.f
	}
}

func (n javaNumber) String() string {
	switch n.class {
	case "Integer", "Long":
		return strconv.FormatInt(n.i, 10)
	case "BigInteger":
		return n.big.String()
	case "BigDecimal":
		return n.dec.String()
	default:
		return jDouble(n.f)
	}
}

// toJavaString is value.toString() of an org.json value.
func (v jsonValue) toJavaString() string {
	switch v.kind {
	case jsonNull:
		return "null"
	case jsonBool:
		return strconv.FormatBool(v.b)
	case jsonString:
		return v.s
	case jsonNumber:
		return v.n.String()
	default:
		return v.raw
	}
}

func classCast(v jsonValue, target string) error {
	from := map[jsonKind]string{
		jsonNull: "org.json.JSONObject$Null", jsonBool: "java.lang.Boolean", jsonString: "java.lang.String",
		jsonArray: "org.json.JSONArray", jsonObject: "org.json.JSONObject",
	}[v.kind]
	if v.kind == jsonNumber {
		from = "java.lang." + v.n.class
	}
	return exception("java.lang.ClassCastException", "Cannot cast %s to %s", from, target)
}

// convertToParameterType is Main.convertToParameterType.
func convertToParameterType(v jsonValue, target string) (interface{}, error) {
	if v.kind == jsonAbsent {
		switch target {
		case "int", "long", "short", "byte", "double", "float", "boolean", "char":
			return nil, illegalArgument("Cannot pass null for primitive type %s", target)
		}
		return nil, nil
	}
	switch target {
	case "int", "java.lang.Integer", "long", "java.lang.Long", "short", "java.lang.Short", "byte", "java.lang.Byte":
		return convertToInteger(v, strings.TrimPrefix(strings.ToLower(target), "java.lang."))
	case "double", "java.lang.Double", "float", "java.lang.Float":
		return convertToFloating(v, target == "float" || target == "java.lang.Float")
	case "boolean", "java.lang.Boolean":
		return convertToBoolean(v)
	case "java.lang.String":
		s := v.toJavaString()
		return &s, nil
	case "byte[]":
		if v.kind != jsonString {
			return nil, illegalArgument("Cannot convert %s to byte[]", v.toJavaString())
		}
		b, err := decodeHex(v.s)
		if err != nil {
			return nil, illegalArgument("Cannot decode hex string to byte array: %s", v.s)
		}
		return b, nil
	case "java.math.BigInteger":
		if v.kind == jsonNumber && v.n.class == "BigInteger" {
			return v.n.big, nil
		}
	}
	return nil, classCast(v, target)
}

// convertToBoolean is Boolean.parseBoolean of a String, or the Boolean itself.
func convertToBoolean(v jsonValue) (interface{}, error) {
	switch v.kind {
	case jsonString:
		return strings.EqualFold(v.s, "true"), nil
	case jsonBool:
		return v.b, nil
	}
	return nil, classCast(v, "java.lang.Boolean")
}

// convertToInteger converts to int, long, short or byte: a String through
// parseInt and friends, a Number through its narrowing intValue/longValue.
func convertToInteger(v jsonValue, kind string) (interface{}, error) {
	bits := map[string]int{"int": 32, "integer": 32, "long": 64, "short": 16, "byte": 8}[kind]
	var i int64
	switch v.kind {
	case jsonString:
		parsed, err := parseJavaInt(v.s, bits)
		if err != nil {
			return nil, err
		}
		i = parsed
	case jsonNumber:
		if bits == 64 {
			i = v.n.longValue()
		} else {
			i = int64(v.n.intValue())
		}
	default:
		return nil, classCast(v, "java.lang.Number")
	}
	switch bits {
	case 64:
		return i, nil
	case 16:
		return int16(i), nil
	case 8:
		return int8(i), nil
	}
	return int32(i), nil
}

// convertToFloating converts to double, or float when single is set.
func convertToFloating(v jsonValue, single bool) (interface{}, error) {
	bits := 64
	if single {
		bits = 32
	}
	var f float64
	switch v.kind {
	case jsonString:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v.s), bits)
		if err != nil {
			return nil, exception("java.lang.NumberFormatException", "For input string: %q", v.s)
		}
		f = parsed
	case jsonNumber:
		f = v.n.doubleValue()
	default:
		return nil, classCast(v, "java.lang.Number")
	}
	if single {
		return float32(f), nil
	}
	return f, nil
}

// parseJavaInt is Integer.parseInt / Long.parseLong / Short.parseShort /
// Byte.parseByte.
func parseJavaInt(s string, bits int) (int64, error) {
	i, err := strconv.ParseInt(s, 10, bits)
	if err != nil || strings.HasPrefix(s, "+-") || strings.HasPrefix(s, "-+") {
		return 0, exception("java.lang.NumberFormatException", "For input string: %q", s)
	}
	return i, nil
}
