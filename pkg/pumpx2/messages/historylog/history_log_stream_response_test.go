package historylog_test

import (
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/historylog"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestHistoryLogStreamResponse(t *testing.T) {
	for _, tc := range []struct {
		name     string
		hex      string
		packets  int
		streamID int32
		records  []string
		typeIDs  []int32
	}{
		{
			name: "id 90", hex: "000081001c01045a00800d941a00000000000000009add200200000000000000004834", packets: 2, streamID: 4,
			records: []string{"5a00800d941a00000000000000009add20020000000000000000"}, typeIDs: []int32{90},
		},
		{
			name: "id 23", hex: "000081001c010a17002bc9b31a350b0000640000006400000001000000f5765c003b5c", packets: 2, streamID: 10,
			records: []string{"17002bc9b31a350b0000640000006400000001000000f5765c00"}, typeIDs: []int32{23},
		},
		{
			name: "id 17", hex: "000081001c010b110054c9b31a360b000003185d000100000000000000000000006e81", packets: 2, streamID: 11,
			records: []string{"110054c9b31a360b000003185d00010000000000000000000000"}, typeIDs: []int32{17},
		},
		{
			name: "id 17 again", hex: "000081001c010c110055c9b31a370b0000c3185d00000000000000000000000000c06e", packets: 2, streamID: 12,
			records: []string{"110055c9b31a370b0000c3185d00000000000000000000000000"}, typeIDs: []int32{17},
		},
		{
			name: "id 22", hex: "000081001c010d160056c9b31a380b0000640000006400000001000000ef1d5d00a77d", packets: 2, streamID: 13,
			records: []string{"160056c9b31a380b0000640000006400000001000000ef1d5d00"}, typeIDs: []int32{22},
		},
		{
			name: "id 10", hex: "000081001c010e0a0057c9b31a390b000001030000df1e5d0001205d0034215d000c57", packets: 2, streamID: 14,
			records: []string{"0a0057c9b31a390b000001030000df1e5d0001205d0034215d00"}, typeIDs: []int32{10},
		},
		{
			name: "id 81", hex: "000081001c010f51005fc9b31a3a0b00000000000000000000000000000164701091ea", packets: 2, streamID: 15,
			records: []string{"51005fc9b31a3a0b000000000000000000000000000001647010"}, typeIDs: []int32{81},
		},
		{
			name: "multi 1", packets: 5, streamID: 8,
			hex: "0000810050030807013352331b6e0f000013010000000000000000000000000000db002751331b6d0f000000000000000000000000000006000000ed002651331b6c0f000000000000000000000000000000000000c651",
			records: []string{
				"07013352331b6e0f000013010000000000000000000000000000",
				"db002751331b6d0f000000000000000000000000000006000000",
				"ed002651331b6c0f000000000000000000000000000000000000",
			},
			typeIDs: []int32{263, 219, 237},
		},
		{
			name: "multi 2", packets: 5, streamID: 8,
			hex: "0000810050030817012651331b6b0f00000000000000002003ffffffff00000000db00fb4f331b6a0f0000000000000000000000000000060000001701fa4f331b690f00000000000000002003ffffffff00000000cfc8",
			records: []string{
				"17012651331b6b0f00000000000000002003ffffffff00000000",
				"db00fb4f331b6a0f000000000000000000000000000006000000",
				"1701fa4f331b690f00000000000000002003ffffffff00000000",
			},
			typeIDs: []int32{279, 219, 279},
		},
		{
			name: "multi 3", packets: 5, streamID: 8,
			hex: "00008100500308db00cf4e331b680f0000000000000000000000000000060000001701ce4e331b670f00000000000000002003ffffffff0000000008011a4e331b660f0000100100000d04080000000000000000000614",
			records: []string{
				"db00cf4e331b680f000000000000000000000000000006000000",
				"1701ce4e331b670f00000000000000002003ffffffff00000000",
				"08011a4e331b660f0000100100000d0408000000000000000000",
			},
			typeIDs: []int32{219, 279, 264},
		},
		{
			name: "multi 4", packets: 2, streamID: 8,
			hex:     "000081001c01080801ca4d331b650f0000100100000d0408000000000000000000b442",
			records: []string{"0801ca4d331b650f0000100100000d0408000000000000000000"},
			typeIDs: []int32{264},
		},
		{
			// Each record's first two bytes carry 1 in their top four bits.
			name: "multi 5", packets: 5, streamID: 40,
			hex: "00008100500328db109ecd2720499e0100020000000200000000000000050000002c119ecd2720489e01008500000000000000000000000000000022119ecd2720479e0100d9132820840000000000000000000000676e",
			records: []string{
				"db109ecd2720499e010002000000020000000000000005000000",
				"2c119ecd2720489e010085000000000000000000000000000000",
				"22119ecd2720479e0100d9132820840000000000000000000000",
			},
			typeIDs: []int32{219, 300, 290},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			records := make([][]byte, len(tc.records))
			for i, r := range tc.records {
				records[i], _ = hex.DecodeString(r)
			}
			expected := historylog.NewHistoryLogStreamResponse(int32(len(records)), tc.streamID, records)
			parsed := parity.TestMessage(t, tc.hex, 0, tc.packets, messages.HistoryLog, expected).(*historylog.HistoryLogStreamResponse)
			if parsed.NumberOfHistoryLogs != int32(len(records)) || parsed.StreamID != tc.streamID {
				t.Errorf("%d records in stream %d", parsed.NumberOfHistoryLogs, parsed.StreamID)
			}
			var typeIDs []int32
			for _, r := range parsed.HistoryLogs {
				typeIDs = append(typeIDs, r.TypeID())
			}
			if !reflect.DeepEqual(typeIDs, tc.typeIDs) {
				t.Errorf("type IDs %v, want %v", typeIDs, tc.typeIDs)
			}
		})
	}
}

func TestHistoryLogStreamResponseRefusesACountItsRecordsDoNotMatch(t *testing.T) {
	m := historylog.NewHistoryLogStreamResponse(2, 1, [][]byte{make([]byte, 26)})
	if err := (&historylog.HistoryLogStreamResponse{}).Parse(m.Cargo()); err == nil {
		t.Error("a stream of one record claiming two parsed")
	}
}

func TestHistoryLogStreamResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "HistoryLogStreamResponse")
}
