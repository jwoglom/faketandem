package protocol

import (
	"encoding/hex"
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
)

// TestChunkPayloadForMTU pins the arithmetic that connects an ATT MTU to this
// protocol's per-fragment payload size: a notification carries MTU-3 bytes
// (ATT opcode plus attribute handle), of which the first two are the
// [remainingPackets][txId] framing.
func TestChunkPayloadForMTU(t *testing.T) {
	cases := []struct {
		mtu  int
		want int
	}{
		// The spec floor is the emulator's default and must land exactly on
		// the 18-byte chunking pumpX2's Packetize and the cliparser jar use.
		{bluetooth.DefaultATTMTU, DefaultMaxChunkPayload},
		{27, 22},
		{185, 180},
		{247, 242},
		{bluetooth.MaxATTMTU, 512},
		// Below the floor there is no smaller legal framing to fall back to,
		// so the default stands rather than producing negative chunks.
		{0, DefaultMaxChunkPayload},
		{10, DefaultMaxChunkPayload},
	}

	for _, tc := range cases {
		if got := ChunkPayloadForMTU(tc.mtu); got != tc.want {
			t.Errorf("ChunkPayloadForMTU(%d) = %d, want %d", tc.mtu, got, tc.want)
		}
	}
}

// TestRefragmentHexPreservesTheMessage checks that re-framing for a different
// MTU changes only the framing: the message bytes, the txId and the
// remaining-fragment counters must all come out right.
func TestRefragmentHexPreservesTheMessage(t *testing.T) {
	const txID = 9

	body, err := BuildMessageBody(MessageSpec{Opcode: 0x25, TxID: txID, Cargo: make([]byte, 60)})
	if err != nil {
		t.Fatalf("BuildMessageBody: %v", err)
	}
	for i := range body[3 : len(body)-2] {
		body[3+i] = byte(i)
	}

	original, err := FragmentMessage(body, txID, DefaultMaxChunkPayload)
	if err != nil {
		t.Fatalf("FragmentMessage: %v", err)
	}
	originalHex := make([]string, 0, len(original))
	for _, f := range original {
		originalHex = append(originalHex, hex.EncodeToString(f))
	}
	if len(originalHex) < 2 {
		t.Fatalf("test message should need several fragments at the default chunk size, got %d", len(originalHex))
	}

	for _, chunkPayload := range []int{DefaultMaxChunkPayload, 22, 180, 512} {
		refragmented, err := RefragmentHex(originalHex, txID, chunkPayload)
		if err != nil {
			t.Fatalf("RefragmentHex(%d): %v", chunkPayload, err)
		}

		rebuilt, err := MessageBodyFromFragmentsHex(refragmented)
		if err != nil {
			t.Fatalf("reassembling the %d-byte-chunk framing: %v", chunkPayload, err)
		}
		if hex.EncodeToString(rebuilt) != hex.EncodeToString(body) {
			t.Errorf("at chunk %d the message bytes changed:\n got %x\nwant %x", chunkPayload, rebuilt, body)
		}

		assertFraming(t, refragmented, txID, chunkPayload)
	}

	// Re-framing at the size the message already uses must be a byte-for-byte
	// no-op, which is what makes the default MTU cost nothing.
	same, err := RefragmentHex(originalHex, txID, DefaultMaxChunkPayload)
	if err != nil {
		t.Fatalf("RefragmentHex: %v", err)
	}
	for i := range same {
		if same[i] != originalHex[i] {
			t.Errorf("fragment %d changed at the same chunk size: %s vs %s", i, same[i], originalHex[i])
		}
	}
}

// assertFraming checks each fragment's [remaining][txId] header and that none
// carries more message bytes than the chunk size allows.
func assertFraming(t *testing.T, fragments []string, txID byte, chunkPayload int) {
	t.Helper()

	for i, fragHex := range fragments {
		frag, err := hex.DecodeString(fragHex)
		if err != nil {
			t.Fatalf("fragment %d is not hex: %v", i, err)
		}
		if want := byte(len(fragments) - i - 1); frag[0] != want {
			t.Errorf("at chunk %d, fragment %d remaining = %d, want %d", chunkPayload, i, frag[0], want)
		}
		if frag[1] != txID {
			t.Errorf("at chunk %d, fragment %d txId = %d, want %d", chunkPayload, i, frag[1], txID)
		}
		if len(frag)-2 > chunkPayload {
			t.Errorf("at chunk %d, fragment %d carries %d message bytes", chunkPayload, i, len(frag)-2)
		}
	}
}

// TestRefragmentHexIntoOneFragment is the shape a real Mobi over iOS produces:
// the whole response in a single notification with remaining=0.
func TestRefragmentHexIntoOneFragment(t *testing.T) {
	body, err := BuildMessageBody(MessageSpec{Opcode: 0x25, TxID: 3, Cargo: make([]byte, 40)})
	if err != nil {
		t.Fatalf("BuildMessageBody: %v", err)
	}
	fragments, err := FragmentMessage(body, 3, DefaultMaxChunkPayload)
	if err != nil {
		t.Fatalf("FragmentMessage: %v", err)
	}
	hexFragments := make([]string, 0, len(fragments))
	for _, f := range fragments {
		hexFragments = append(hexFragments, hex.EncodeToString(f))
	}

	out, err := RefragmentHex(hexFragments, 3, ChunkPayloadForMTU(247))
	if err != nil {
		t.Fatalf("RefragmentHex: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d fragments at a 247-byte MTU, want 1", len(out))
	}
	single, err := hex.DecodeString(out[0])
	if err != nil {
		t.Fatalf("fragment is not hex: %v", err)
	}
	if single[0] != 0 {
		t.Errorf("remaining counter = %d, want 0", single[0])
	}
}
