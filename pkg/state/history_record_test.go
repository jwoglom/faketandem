package state

import (
	"encoding/hex"
	"math"
	"testing"
)

// float32Bits renders a captured IEEE-754 bit pattern as the float64 the
// encoder takes, so fixtures can be reproduced exactly rather than approximated
// by a decimal literal.
func float32Bits(bits uint32) float64 {
	return float64(math.Float32frombits(bits))
}

// TestEncodeRecord_AgainstCapturedRecords reproduces history log records
// captured from real pumps, byte for byte.
//
// Every fixture is a complete 26-byte record taken from TandemKit's
// Tests/TandemCoreTests/HistoryLog fixtures, which are themselves ports of
// pumpX2's history log tests plus later Tandem Mobi BLE captures. Reproducing
// them is what pins down the type IDs, the little-endian header layout and each
// event's payload offsets.
func TestEncodeRecord_AgainstCapturedRecords(t *testing.T) {
	cases := []struct {
		name  string
		entry HistoryLogEntry
		want  string
	}{
		{
			name: "BolusActivated",
			entry: HistoryLogEntry{
				TypeID: HistoryBolusActivated, PumpTime: 446033777, Sequence: 182751,
				Data: map[string]interface{}{
					"bolusId": 1037, "selectedIob": 1, "iob": 0.0, "bolusSize": float32Bits(0x3f8ccccd),
				},
			},
			want: "370071ef951adfc902000d04010000000000cdcc8c3f00000000",
		},
		{
			name: "BolusActivatedMobiRemoteBolus",
			entry: HistoryLogEntry{
				TypeID: HistoryBolusActivated, PumpTime: 589526845, Sequence: 640754, SourceNibble: 1,
				Data: map[string]interface{}{
					"bolusId": 2903, "selectedIob": 0,
					"iob": float32Bits(0x3f9eec42), "bolusSize": float32Bits(0x3e19999a),
				},
			},
			want: "37103d772323f2c60900570b000042ec9e3f9a99193e00000000",
		},
		{
			name: "BolusCompleted",
			entry: HistoryLogEntry{
				TypeID: HistoryBolusCompleted, PumpTime: 446158750, Sequence: 186480,
				Data: map[string]interface{}{
					"completionStatusId": 3, "bolusId": 1057,
					"iob":              float32Bits(0x4069c854),
					"insulinDelivered": float32Bits(0x3fe4baf2),
					"insulinRequested": float32Bits(0x3fe4baf2),
				},
			},
			want: "14009ed7971a70d802000300210454c86940f2bae43ff2bae43f",
		},
		{
			name: "TempRateActivated",
			entry: HistoryLogEntry{
				TypeID: HistoryTempRateActivated, PumpTime: 579777478, Sequence: 449311, SourceNibble: 1,
				Data: map[string]interface{}{
					"percent": 50.0, "durationMilliseconds": 7200000.0,
					"unknown": 2, "tempRateId": 21,
				},
			},
			want: "0210c6b38e221fdb06000000484200badb4a0200150000000000",
		},
		{
			name: "TempRateCompleted",
			entry: HistoryLogEntry{
				TypeID: HistoryTempRateCompleted, PumpTime: 579784686, Sequence: 449606, SourceNibble: 1,
				Data:   map[string]interface{}{"unknown": 3, "tempRateId": 21, "timeLeft": 0},
			},
			want: "0f10eecf8e2246dc060003001500000000000000000000000000",
		},
		{
			name: "PumpingSuspended",
			entry: HistoryLogEntry{
				TypeID: HistoryPumpingSuspended, PumpTime: 445995051, Sequence: 181766,
				Data: map[string]interface{}{
					"preSuspendState": 106, "insulinAmount": 31, "reasonId": 0, "rpaTimeout": 0,
				},
			},
			want: "0b002b58951a06c602006a0000001f0000000000000000000000",
		},
		{
			name: "PumpingSuspendedMobiWithRpaTimeout",
			entry: HistoryLogEntry{
				TypeID: HistoryPumpingSuspended, PumpTime: 589553316, Sequence: 641936, SourceNibble: 1,
				Data: map[string]interface{}{
					"preSuspendState": 106, "insulinAmount": 150, "reasonId": 0, "rpaTimeout": 15,
				},
			},
			want: "0b10a4de232390cb09006a0000009600000f0000000000000000",
		},
		{
			name: "PumpingResumed",
			entry: HistoryLogEntry{
				TypeID: HistoryPumpingResumed, PumpTime: 446032479, Sequence: 182692,
				Data:   map[string]interface{}{"preResumeState": 100, "insulinAmount": 180},
			},
			want: "0c005fea951aa4c9020064000000b40000000000000000000000",
		},
		{
			name: "BasalRateChange",
			entry: HistoryLogEntry{
				TypeID: HistoryBasalRateChange, PumpTime: 445993934, Sequence: 181703,
				Data: map[string]interface{}{
					"commandBasalRate":       float32Bits(0x3edf3b64),
					"baseBasalRate":          float32Bits(0x3f4ccccd),
					"maxBasalRate":           float32Bits(0x40a00000),
					"insulinDeliveryProfile": 0, "changeTypeId": 1,
				},
			},
			want: "0300ce53951ac7c50200643bdf3ecdcc4c3f0000a04000000100",
		},
		{
			name: "BasalRateChangePumpSuspended",
			entry: HistoryLogEntry{
				TypeID: HistoryBasalRateChange, PumpTime: 446031151, Sequence: 182635,
				Data: map[string]interface{}{
					"commandBasalRate":       0.0,
					"baseBasalRate":          float32Bits(0x3fa00000),
					"maxBasalRate":           float32Bits(0x40a00000),
					"insulinDeliveryProfile": 0, "changeTypeId": 16,
				},
			},
			want: "03002fe5951a6bc90200000000000000a03f0000a04000001000",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := hex.EncodeToString(c.entry.EncodeRecord())
			if got != c.want {
				t.Errorf("EncodeRecord() =\n  %s\nwant\n  %s", got, c.want)
			}
		})
	}
}

// TestEncodeRecord_UnknownTypeIsStillWellFormed checks that an event with no
// dedicated payload encoder still produces a correctly typed, timestamped and
// sequenced 26-byte record.
func TestEncodeRecord_UnknownTypeIsStillWellFormed(t *testing.T) {
	entry := HistoryLogEntry{TypeID: HistoryNewDay, PumpTime: 1234, Sequence: 7}
	record := entry.EncodeRecord()

	if len(record) != HistoryRecordLength {
		t.Fatalf("record length = %d, want %d", len(record), HistoryRecordLength)
	}
	if got := hex.EncodeToString(record); got != "5a00d20400000700000000000000000000000000000000000000" {
		t.Errorf("record = %s", got)
	}
}

// TestEncodeRecord_TypeIdAboveByteRange guards the 12-bit type field: an ID
// above 255 must keep its high bits instead of being truncated into the low
// byte.
func TestEncodeRecord_TypeIdAboveByteRange(t *testing.T) {
	entry := HistoryLogEntry{TypeID: HistoryBolusDelivery, PumpTime: 0, Sequence: 1}
	record := entry.EncodeRecord()
	if record[0] != 0x18 || record[1] != 0x01 {
		t.Errorf("type header = %02x%02x, want 1801 (280 little-endian)", record[0], record[1])
	}
}

// TestAppendHistory_SequencesFromOne pins the sequence numbering the history
// log status response reports.
func TestAppendHistory_SequencesFromOne(t *testing.T) {
	ps := NewPumpState()

	first := ps.AppendHistory(HistoryEvent{TypeID: HistoryPumpingSuspended, Name: "PumpingSuspended"})
	second := ps.AppendHistory(HistoryEvent{TypeID: HistoryPumpingResumed, Name: "PumpingResumed"})

	if first.Sequence != 1 || second.Sequence != 2 {
		t.Fatalf("sequences = %d, %d; want 1, 2", first.Sequence, second.Sequence)
	}

	firstSeq, lastSeq := ps.GetHistoryLogSequenceRange()
	if firstSeq != 1 || lastSeq != 2 {
		t.Errorf("sequence range = %d..%d, want 1..2", firstSeq, lastSeq)
	}
	if count := ps.GetHistoryLogCount(); count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

// TestAppendHistory_ExplicitPumpTimeIsPreserved covers the backdating path a
// scenario uses to stage old history.
func TestAppendHistory_ExplicitPumpTimeIsPreserved(t *testing.T) {
	ps := NewPumpState()

	entry := ps.AppendHistory(HistoryEvent{
		TypeID:   HistoryBolusCompleted,
		Name:     "BolusCompleted",
		PumpTime: 446158750,
	})

	if entry.PumpTime != 446158750 {
		t.Errorf("PumpTime = %d, want 446158750", entry.PumpTime)
	}
	if want := ps.WallClockForPumpTime(446158750); !entry.Timestamp.Equal(want) {
		t.Errorf("Timestamp = %v, want %v", entry.Timestamp, want)
	}
}
