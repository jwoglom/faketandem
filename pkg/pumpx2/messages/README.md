# pkg/pumpx2/messages

A Go implementation of [pumpX2](https://github.com/jwoglom/pumpX2)'s message library: every request and
response class pumpX2 has is a Go type here, with its fields, constructors and parse. The framework
around them frames, signs, parses, prints and builds messages exactly as the pumpX2 cliparser jar does.
faketandem needs no JVM to use them.

## Layout

| Package | pumpX2 package | Characteristic |
|---|---|---|
| `authentication` | `request.authentication`, `response.authentication` | AUTHORIZATION |
| `control` | `request.control`, `response.control` | CONTROL |
| `controlstream` | `request.controlStream`, `response.controlStream` | CONTROL_STREAM |
| `currentstatus` | `request.currentStatus`, `response.currentStatus`, and `response.ErrorResponse` | CURRENT_STATUS |
| `historylog` | `request.historyLog`, `response.historyLog` | HISTORY_LOG |

Each pumpX2 class is one file, named for the class in snake case (`CgmStatusV2Response` is
`currentstatus/cgm_status_v2_response.go`). Its test sits beside it in `<name>_test.go`. The other
packages are shared:

- `messages`: the `Message` interface and `Base`, the registry, the `Messages` enum and `OPCODES`
  table, framing and signing (`Packets`), cliparser's `parse` (`ParseFragments`) and `encode`
  (`Encode`), and `ToString`.
- `helpers`: pumpX2's `helpers.Bytes` and `helpers.Dates`.
- `models`: pumpX2's `models` that messages share (`MinsTime`, `MultiDay`, `BloodGlucoseReadingType`, …).
- `all`: imports every message package. Import it, for its side effect, wherever messages are parsed or
  built by name.
- `parity`: the checks that hold the Go types to the jar (below).

Imports go one way: `control` may import `currentstatus` (several control requests carry its enums),
never the reverse. Where a pumpX2 getter on a `currentstatus` class returns a `control` type, the Go
version is a function in `control`.

## A message type

```go
// APIVersionResponse is the pump's API version: ...
type APIVersionResponse struct {
	messages.Base
	MajorVersion int32 `pumpx2:"majorVersion"`
	MinorVersion int32 `pumpx2:"minorVersion"`
}

var apiVersionResponseProps = messages.Props{OpCode: 33, Size: 4, Type: messages.Response, Request: "ApiVersionRequest"}

func init() {
	messages.Register("ApiVersionResponse", &APIVersionResponse{}, apiVersionResponseProps,
		messages.Constructor{Params: "int majorVersion, int minorVersion", Func: NewAPIVersionResponse},
	)
}

func NewAPIVersionResponse(majorVersion, minorVersion int32) *APIVersionResponse { ... }

func (m *APIVersionResponse) Parse(raw []byte) error { ... }
```

- **Name.** The Go type is the pumpX2 class name with Go's initialisms (`ApiVersionResponse` is
  `APIVersionResponse`, `...IdRequest` is `...IDRequest`). `Register` takes the pumpX2 name.
- **Fields.** Each field pumpX2's `toString()` prints is an exported field tagged with its Java name.
  `ToString` prints the tagged fields sorted by that name, then those of an embedded message type
  (pumpX2's superclass), then the cargo, which is what commons-lang's `ReflectionToStringBuilder`
  prints. `parity/testdata/catalog.json` lists each class's printed fields. A field pumpX2 leaves out
  of `toString()` (`intMap`) has no tag.
- **Field types.** `int` is `int32`, `long` is `int64`, `boolean` is `bool`, `byte[]` is `[]byte`,
  `String` is `string`, `BigInteger` is `*big.Int`, `Instant` is `time.Time`, `Integer` and `Long`
  are `*int32` and `*int64`. An enum is a named Go type with a `String()` that returns the Java constant's
  name. A `Set` or `List` is a slice, in the order pumpX2 iterates it (declaration order for an
  `EnumSet` or a `TreeSet` of enums). A nested class is a struct whose `String()` is
  `messages.AutoToString("Outer.Inner", v)`. A Java reference pumpX2 can leave or set null, once
  parsed or built, is a pointer, slice or `*big.Int`, so that it prints `<null>`. One it never leaves
  null is a value.
- **Props.** `messages.Props` is the `@MessageProps` annotation. A zero field takes the annotation's
  default (`CURRENT_STATUS`, `API_V2_1`, `ALL`). `Request` and `Response` name the paired class.
- **Constructors.** `Register` lists every public pumpX2 constructor with parameters, in the jar's order
  (catalog.json's `constructors`). cliparser's `encode` picks the first one with as many parameters as
  it was given values, and fills them by name. `Params` is the Java parameter list with short type
  names (`String txId`, `BigInteger intMap`, `CgmStatusV2Response.CgmSensorType sensorType`), and `Func`
  is the Go constructor, which takes `int32` for `int` and so on (see `Constructor`). List the no-arg
  constructor (`messages.Constructor{Func: NewX}`) only when it does more than the zero value does; it
  is what pumpX2 parses into. A constructor returns `(*X, error)` where the Java validates, else `*X`.
  The main constructor is `NewX`. The others are named for what they take: `NewXFromCargo` for
  `(byte[] raw)`, `NewExtendedX`, and so on.
- **Parse.** It does what pumpX2's `parse(byte[])` does. A signed message first drops its trailer
  (`messages.RemoveSignedRequestHmacBytes`). It returns `messages.InvalidLength` or `messages.Invalid`
  where the Java's `Validate` or `Preconditions` throws. Java's `raw[i]` is signed, `int32(int8(raw[i]))`;
  `raw[i] & 0xFF` is `int32(raw[i])`. A read past the end panics like Go's own slicing, and
  `messages.Parse` turns it into an error, as `Messages.parse` catches every exception. Check a length
  up front where you can.
- **Subclasses.** A class that extends another message embeds it, and declares its own props and
  `Parse`, since pumpX2's `props()` is always the subclass's.
- **Enums.** `type X int32` holds the wire ID (the ordinal when the enum has none), with `XFromID`
  for pumpX2's `fromId` (returning `*X` where that can return null), an `X…` slice in declaration
  order for sets, and `XFromBitmask` returning a non-nil slice. A nested type takes its Java name.
  The exceptions are the CGM status enums: `CGMStatusResponse`, `CgmStatusV2Response` and
  `CGMStatusCResponse` each declare a `SessionState` and a `TransmitterBatteryStatus`, and the last two
  also a `CgmSensorType`. In Go these are `CGMStatusSessionState`, `CgmStatusV2SensorType`,
  `CGMStatusCTransmitterBatteryStatus` and so on.

## Checking against pumpX2

The jar is the specification, and three kinds of test hold the Go types to it.

- `parity/testdata/catalog.json` is what the jar says about every class: props, printed fields,
  constructors, the `Messages` enum and `OPCODES`. `catalog_test.go` holds the registry to it.
  `TestFieldsMatchJar` compares what each type's zero value prints with the jar's empty instance, and
  `TestEveryClassHasItsFiles` checks the layout above.
- `<package>/testdata/<Class>.jsonl` are the jar's recorded answers for that class: one JSON object
  per line, each a `parse` input (fragments and characteristic) or an `encode` input (message name,
  params and transaction ID) with what cliparser printed for it. Every class's test calls
  `parity.CheckClass(t, "<Class>")`, which runs them through the Go types. A parse vector is filed
  under the class it parsed as. `testdata/messages.jsonl` holds the vectors no one class owns.
- Each class's test also ports pumpX2's own test of it (`MessageTester.test` is `parity.TestMessage`,
  or `parity.MessageTester{Env: ...}.Test` after `initPumpState`).

With `FAKETANDEM_TEST_CLIPARSER_JAR` set to a built cliparser jar,
`go test ./pkg/pumpx2/messages/parity` regenerates every case and checks the recorded answers are what
the jar answers today. `-update-golden` re-records them. After pumpX2 changes, regenerate the catalog
with `tools/pumpx2catalog/regen.sh <jar>`, re-record, and fix the Go types until the tests pass.
A difference is fixed in the Go code, never by editing the recorded answers.
