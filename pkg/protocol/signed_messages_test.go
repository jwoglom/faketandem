package protocol

import (
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// updateSignedTable regenerates signed_messages.go from the TandemKit sources
// instead of asserting against them. Run:
//
//	go test ./pkg/protocol -run TestSignedMessageTableMatchesTandemKit -update
var updateSignedTable = flag.Bool("update", false, "regenerate signed_messages.go from the TandemKit sources")

// TestSignedMessageTableCoversTheControlPath is the table's unconditional
// guard: it runs everywhere, with no TandemKit checkout, and pins the handful
// of messages the emulator's own signing path depends on.
//
// Getting one of these wrong is not a cosmetic failure. A response the pump
// marks signed but sends unsigned is short by 24 bytes and fails the driver's
// length check; one sent signed that should not be carries a trailer the
// driver reads as cargo.
func TestSignedMessageTableCoversTheControlPath(t *testing.T) {
	signed := []string{
		"SuspendPumpingResponse",
		"ResumePumpingResponse",
		"InitiateBolusResponse",
		"CancelBolusResponse",
		"SetTempRateResponse",
		"StopTempRateResponse",
		"BolusPermissionResponse",
		"BolusPermissionReleaseResponse",
		"RemoteBgEntryResponse",
		"RemoteCarbEntryResponse",
		"ErrorResponse",
	}
	for _, name := range signed {
		if !IsSignedMessage(name) {
			t.Errorf("IsSignedMessage(%q) = false, want true", name)
		}
	}

	// Status reads are never signed; if one of these ever flipped, every
	// CURRENT_STATUS response would grow a trailer the driver cannot parse.
	unsigned := []string{
		"ApiVersionResponse",
		"InsulinStatusResponse",
		"CurrentBasalStatusResponse",
		"CurrentBolusStatusResponse",
		"HistoryLogStatusResponse",
		"PumpVersionResponse",
		"AlarmStatusResponse",
		"TimeSinceResetResponse",
	}
	for _, name := range unsigned {
		if IsSignedMessage(name) {
			t.Errorf("IsSignedMessage(%q) = true, want false", name)
		}
	}
}

// TestSignedMessageTableMatchesTandemKit re-derives the table from the Swift
// MessageProps declarations and fails if the two have drifted: a message in
// the table that TandemKit no longer marks signed, or -- the dangerous
// direction -- a signed message TandemKit knows about that the table is
// missing, which would leave its responses unsigned on the wire.
//
// Skipped when no TandemKit checkout is available.
func TestSignedMessageTableMatchesTandemKit(t *testing.T) {
	root := tandemCoreRoot(t)
	if root == "" {
		t.Skip("no TandemKit TandemCore sources found; set FAKETANDEM_TANDEMKIT_CORE to enable")
	}

	want, err := parseSwiftSignedMessages(root)
	if err != nil {
		t.Fatalf("failed to read the TandemKit message catalog: %v", err)
	}
	if len(want) == 0 {
		t.Fatalf("parsed no MessageProps declarations under %s; the parser is broken", root)
	}

	if *updateSignedTable {
		if err := writeSignedMessagesFile(want); err != nil {
			t.Fatalf("failed to regenerate signed_messages.go: %v", err)
		}
		t.Logf("regenerated signed_messages.go with %d signed messages", len(want))
		return
	}

	got := SignedMessageNames()
	if strings.Join(got, "\n") == strings.Join(want, "\n") {
		return
	}

	inTable := make(map[string]bool, len(got))
	for _, name := range got {
		inTable[name] = true
	}
	inSwift := make(map[string]bool, len(want))
	for _, name := range want {
		inSwift[name] = true
	}
	for _, name := range want {
		if !inTable[name] {
			t.Errorf("%s is signed in TandemKit but missing from the table; its responses would go out unsigned", name)
		}
	}
	for _, name := range got {
		if !inSwift[name] {
			t.Errorf("%s is in the table but TandemKit does not mark it signed", name)
		}
	}
	t.Log("run: go test ./pkg/protocol -run TestSignedMessageTableMatchesTandemKit -update")
}

// tandemCoreRoot locates a TandemCore source tree: the FAKETANDEM_TANDEMKIT_CORE
// override first, then the conventional sibling checkout.
func tandemCoreRoot(t *testing.T) string {
	t.Helper()

	if override := os.Getenv("FAKETANDEM_TANDEMKIT_CORE"); override != "" {
		return override
	}

	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	// pkg/protocol -> repo root -> the directory holding the checkouts.
	repoRoot := filepath.Dir(filepath.Dir(wd))
	for _, candidate := range []string{
		filepath.Join(repoRoot, "..", "TandemKit", "Sources", "TandemCore"),
		filepath.Join(repoRoot, "..", "..", "TandemKit", "Sources", "TandemCore"),
	} {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}

// classDecl matches a Swift class declaration; the message catalog declares one
// class per wire message, each with a static MessageProps.
var classDecl = regexp.MustCompile(`(?m)(?:public\s+)?(?:final\s+)?class\s+(\w+)\s*:\s*[^{]*\{`)

// parseSwiftSignedMessages returns the sorted names of every message class
// whose MessageProps carries signed: true.
//
// The pairing rule is positional, which is what the catalog's own layout
// allows: a class's props are the first MessageProps( that follows its
// declaration and precedes the next class declaration, so a file holding a
// request and its response (the common shape) attributes each block correctly.
func parseSwiftSignedMessages(root string) ([]string, error) {
	var signed []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".swift") {
			return nil
		}
		source, err := os.ReadFile(path) //nolint:gosec // reading a checked-out source tree
		if err != nil {
			return err
		}
		signed = append(signed, signedClassesIn(string(source))...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(signed)
	return signed, nil
}

// signedClassesIn extracts the signed message class names from one Swift file.
func signedClassesIn(source string) []string {
	var signed []string

	decls := classDecl.FindAllStringSubmatchIndex(source, -1)
	for i, decl := range decls {
		name := source[decl[2]:decl[3]]
		limit := len(source)
		if i+1 < len(decls) {
			limit = decls[i+1][0]
		}

		props := propsBlock(source[decl[1]:limit])
		if props != "" && strings.Contains(props, "signed: true") {
			signed = append(signed, name)
		}
	}
	return signed
}

// propsBlock returns the balanced MessageProps(...) argument list at the start
// of body, or "" when body declares no props.
func propsBlock(body string) string {
	start := strings.Index(body, "MessageProps(")
	if start == -1 {
		return ""
	}

	depth := 0
	for i := start + len("MessageProps(") - 1; i < len(body); i++ {
		switch body[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return body[start : i+1]
			}
		}
	}
	return ""
}

// writeSignedMessagesFile renders the generated table.
func writeSignedMessagesFile(names []string) error {
	var b strings.Builder
	b.WriteString(signedMessagesHeader)
	for _, name := range names {
		fmt.Fprintf(&b, "\t%q: true,\n", name)
	}
	b.WriteString(signedMessagesFooter)

	// gofmt the result so the generated map literal is aligned and the file
	// passes the repo's own formatting checks.
	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return fmt.Errorf("generated source does not parse: %w", err)
	}

	return os.WriteFile("signed_messages.go", formatted, 0o600)
}

const signedMessagesHeader = `// Code generated from TandemKit's TandemCore message catalog. DO NOT EDIT.
//
// Regenerate with:
//
//	go test ./pkg/protocol -run TestSignedMessageTableMatchesTandemKit -update
//
// See signed_messages_test.go for the derivation and for the guard that fails
// when this table and the catalog disagree.

package protocol

import "sort"

// signedMessages lists every message whose MessageProps declares signed: true,
// i.e. every message that carries the 24-byte timeSinceReset+HMAC-SHA1 trailer
// described in BuildMessageBody.
//
// It covers requests as well as responses. Only the responses matter to the
// emulator (it never sends a request), but the catalog marks both halves of a
// control exchange signed and splitting them here would invite the table and
// the catalog to drift apart.
var signedMessages = map[string]bool{
`

const signedMessagesFooter = `}

// IsSignedMessage reports whether the named message carries the signed
// trailer. The name is a pumpX2/TandemKit message class name, exactly as
// cliparser prints it (e.g. "SuspendPumpingResponse").
//
// An unknown name reports false: a message nobody has a catalog entry for is
// sent as-is rather than given a trailer the driver would not expect.
func IsSignedMessage(name string) bool {
	return signedMessages[name]
}

// SignedMessageNames returns the table's contents, sorted, for tests and for
// anything that needs to enumerate the signed half of the catalog.
func SignedMessageNames() []string {
	names := make([]string, 0, len(signedMessages))
	for name := range signedMessages {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
`
