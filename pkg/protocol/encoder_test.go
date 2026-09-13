package protocol

import (
	"encoding/hex"
	"strings"
	"testing"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex fixture %q: %v", s, err)
	}
	return b
}

// TestCalculateCRC16_AgainstCapturedPackets checks the CRC against complete
// packets captured from real pumps, taken from TandemKit's
// Tests/TandemCoreTests fixtures. Each fixture is a whole single-fragment
// packet: [remaining][txId][opcode][txId][length][cargo...][crcLo][crcHi], so
// the CRC input is bytes 2..len-2 and the expected output is the last two.
func TestCalculateCRC16_AgainstCapturedPackets(t *testing.T) {
	fixtures := []struct {
		name   string
		packet string
	}{
		// CurrentBatteryV1Response, 99/100.
		{"CurrentBatteryV1Response", "0003350302636452b9"},
		{"CurrentBatteryV1ResponseFullCharge", "00043504026464e871"},
		// AlarmStatusResponse with an empty bitmask, and with
		// PUMP_RESET_ALARM|RESUME_PUMP_ALARM2 set.
		{"AlarmStatusResponseEmpty", "000347030800000000000000005721"},
		{"AlarmStatusResponsePumpResetAndResume", "000c470c0808008000000000001cbd"},
	}

	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			packet := mustHex(t, f.packet)
			body := packet[2 : len(packet)-2]
			want := packet[len(packet)-2:]
			got := CalculateCRC16(body)
			if hex.EncodeToString(got) != hex.EncodeToString(want) {
				t.Errorf("CRC16 = %x, want %x", got, want)
			}
		})
	}
}

// TestBuildMessageBody_ReproducesCapturedPackets rebuilds whole captured pump
// packets from (opcode, txId, cargo) and asserts the encoder is byte-for-byte
// identical, framing and CRC included.
func TestBuildMessageBody_ReproducesCapturedPackets(t *testing.T) {
	cases := []struct {
		name   string
		opcode uint8
		txID   uint8
		cargo  string
		want   string // full fragment, framing bytes included
	}{
		{"CurrentBatteryV1Response", 53, 3, "6364", "0003350302636452b9"},
		{"AlarmStatusResponseEmpty", 71, 3, "0000000000000000", "000347030800000000000000005721"},
		{"AlarmStatusResponsePumpResetAndResume", 71, 12, "0800800000000000", "000c470c0808008000000000001cbd"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fragments, err := EncodeMessage(MessageSpec{
				Opcode: c.opcode,
				TxID:   c.txID,
				Cargo:  mustHex(t, c.cargo),
			})
			if err != nil {
				t.Fatalf("EncodeMessage: %v", err)
			}
			if len(fragments) != 1 {
				t.Fatalf("expected a single fragment, got %d", len(fragments))
			}
			if got := hex.EncodeToString(fragments[0]); got != c.want {
				t.Errorf("fragment = %s, want %s", got, c.want)
			}
		})
	}
}

// TestFragmentMessage_RemainingCounters checks the fragment framing the
// receiving side depends on: a descending remaining-packet counter in byte 0,
// the txId repeated in byte 1, and 18 message bytes per fragment.
func TestFragmentMessage_RemainingCounters(t *testing.T) {
	body := make([]byte, 40)
	for i := range body {
		body[i] = byte(i)
	}

	fragments, err := FragmentMessage(body, 7, DefaultMaxChunkPayload)
	if err != nil {
		t.Fatalf("FragmentMessage: %v", err)
	}
	if len(fragments) != 3 {
		t.Fatalf("expected 3 fragments for 40 bytes at 18 bytes/fragment, got %d", len(fragments))
	}

	var rebuilt []byte
	for i, f := range fragments {
		if want := byte(len(fragments) - i - 1); f[0] != want {
			t.Errorf("fragment %d remaining = %d, want %d", i, f[0], want)
		}
		if f[1] != 7 {
			t.Errorf("fragment %d txId = %d, want 7", i, f[1])
		}
		rebuilt = append(rebuilt, f[2:]...)
	}
	if hex.EncodeToString(rebuilt) != hex.EncodeToString(body) {
		t.Errorf("reassembled body does not match the original")
	}
}

// TestBuildMessageBody_Signed checks the signed-message trailer layout: 24
// bytes made of a little-endian timeSinceReset followed by a 20-byte HMAC-SHA1
// over everything before it, and a cargo length that includes the trailer.
//
// The expected bytes come from the pumpX2 cliparser jar 1.9.1, which was run as
//
//	PUMP_AUTHENTICATION_KEY=00112233445566778899aabbccddeeff \
//	PUMP_TIME_SINCE_RESET=1000 java -jar cliparser encode 5 \
//	    SuspendPumpingResponse '{"status":0}'
//
// Note cliparser uses the raw bytes of the environment string as the HMAC key,
// so the key here is the ASCII of that hex, not its decoded value. The
// jar-gated parity test in pkg/handler re-derives this live.
func TestBuildMessageBody_Signed(t *testing.T) {
	fragments, err := EncodeMessage(MessageSpec{
		Opcode:         0x9d,
		TxID:           5,
		Cargo:          []byte{0x00},
		Signed:         true,
		AuthKey:        []byte("00112233445566778899aabbccddeeff"),
		TimeSinceReset: 1000,
	})
	if err != nil {
		t.Fatalf("EncodeMessage: %v", err)
	}

	want := []string{
		"01059d051900e803000019d9e37838ca2332a565",
		"0005bba6d7908d8f0668a9598e9c",
	}
	got := make([]string, 0, len(fragments))
	for _, f := range fragments {
		got = append(got, hex.EncodeToString(f))
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("signed fragments =\n  %v\nwant\n  %v", got, want)
	}

	// The header's length byte must cover cargo + trailer.
	if fragments[0][4] != 1+SignedTrailerLength {
		t.Errorf("payload length byte = %d, want %d", fragments[0][4], 1+SignedTrailerLength)
	}
}

func TestBuildMessageBody_SignedRequiresKey(t *testing.T) {
	_, err := BuildMessageBody(MessageSpec{Opcode: 1, TxID: 1, Cargo: []byte{0}, Signed: true})
	if err == nil {
		t.Fatal("expected an error when signing without an authentication key")
	}
}

func TestBuildMessageBody_RejectsOversizedPayload(t *testing.T) {
	_, err := BuildMessageBody(MessageSpec{Opcode: 1, TxID: 1, Cargo: make([]byte, 256)})
	if err == nil {
		t.Fatal("expected an error for a cargo that does not fit the one-byte length field")
	}
}
