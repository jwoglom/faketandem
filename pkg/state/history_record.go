package state

import (
	"encoding/binary"
	"math"
	"time"
)

// HistoryRecordLength is the fixed size of a history log record on the wire.
// Every record is exactly this long regardless of event type: a 10-byte common
// header followed by 16 bytes of event-specific payload, zero-padded.
const HistoryRecordLength = 26

// Common header offsets shared by every history log record.
//
//	0..1   typeId, little-endian uint16. Only the low 12 bits are the type; the
//	       high nibble is a source marker the driver masks off (& 0x0FFF).
//	2..5   pumpTimeSec, little-endian uint32, seconds since the Tandem epoch.
//	6..9   sequenceNum, little-endian uint32.
//	10..25 event payload.
const (
	historyPayloadOffset = 10
)

// HistoryEvent describes one history log record to append.
//
// This is the type the scenario API uses to stage history: build a
// HistoryEvent, hand it to PumpState.AppendHistory, and the pump will report it
// in HistoryLogStatusResponse and stream it in response to a HistoryLogRequest.
//
//	ps.AppendHistory(state.HistoryEvent{
//	    TypeID: state.HistoryBolusActivated,
//	    Name:   "BolusActivated",
//	    Fields: map[string]interface{}{"bolusId": 1037, "bolusSize": 1.1},
//	})
//
// Fields are looked up by the names pumpX2/TandemKit give the record's fields
// (see encodeHistoryPayload for the per-type list); anything absent encodes as
// zero, and anything unrecognized is ignored but kept in the stored entry for
// the state snapshot API.
type HistoryEvent struct {
	// TypeID is the record's pumpX2 type ID; use the History* constants in
	// history_types.go.
	TypeID int
	// Name is a human-readable label, used for logging and the state snapshot.
	Name string
	// PumpTime is the record's timestamp in seconds since the Tandem epoch.
	// Zero means "stamp it with the pump's current clock", which is what live
	// events want; a scenario backdating history sets it explicitly (see
	// PumpTimeSeconds).
	PumpTime uint32
	// Fields carries the event payload.
	Fields map[string]interface{}
	// SourceNibble is the high nibble of the type ID word. Real captures carry
	// 1 on a Mobi and 0 on older pumps; it does not affect decoding, since the
	// driver masks it off. Leave it 0 unless reproducing a capture byte-exactly.
	SourceNibble uint8
}

// AppendHistory stores a history log event and returns the stored entry,
// including the sequence number it was assigned.
//
// Sequence numbers are allocated from 1 upward and never reused, which is what
// HistoryLogStatusResponse reports and what a HistoryLogRequest addresses.
// Appending is the only way entries enter the log, so the stored order is
// always ascending by sequence.
func (ps *PumpState) AppendHistory(event HistoryEvent) HistoryLogEntry {
	pumpTime := event.PumpTime
	var timestamp time.Time
	if pumpTime == 0 {
		timestamp = ps.currentTimeOrNow()
		pumpTime = PumpTimeSeconds(timestamp)
	} else {
		timestamp = PumpTimeToWallClock(pumpTime)
	}

	ps.HistoryLog.mutex.Lock()
	defer ps.HistoryLog.mutex.Unlock()

	entry := HistoryLogEntry{
		Sequence:     ps.HistoryLog.NextSequence,
		TypeID:       event.TypeID,
		Type:         event.Name,
		Timestamp:    timestamp,
		PumpTime:     pumpTime,
		Data:         event.Fields,
		SourceNibble: event.SourceNibble,
	}
	ps.HistoryLog.Entries = append(ps.HistoryLog.Entries, entry)
	ps.HistoryLog.NextSequence++

	return entry
}

// currentTimeOrNow reads the pump's modeled clock, falling back to wall clock
// before the first simulator tick has set it.
func (ps *PumpState) currentTimeOrNow() time.Time {
	ps.mutex.RLock()
	current := ps.CurrentTime
	ps.mutex.RUnlock()
	if current.IsZero() {
		return time.Now()
	}
	return current
}

// EncodeRecord renders the entry as the 26 bytes a HistoryLogStreamResponse
// carries.
//
// The common header is always written; the payload depends on the type ID. An
// entry whose type ID has no payload encoder here still produces a valid,
// correctly-typed and correctly-sequenced record with a zero payload, which is
// enough for the driver to accept and order it.
func (e HistoryLogEntry) EncodeRecord() []byte {
	record := make([]byte, HistoryRecordLength)

	// Low 12 bits are the type; the high nibble is the source marker.
	binary.LittleEndian.PutUint16(record[0:2], uint16(e.TypeID&0x0FFF)|(uint16(e.SourceNibble&0x0F)<<12))
	binary.LittleEndian.PutUint32(record[2:6], e.PumpTime)
	binary.LittleEndian.PutUint32(record[6:10], e.Sequence)

	encodeHistoryPayload(record[historyPayloadOffset:], e.TypeID, e.Data)

	return record
}

// encodeHistoryPayload writes the 16 event-specific payload bytes for typeID.
//
// Offsets below are relative to the start of the payload (byte 10 of the
// record) and were read off TandemKit's per-event `init(cargo:)` accessors --
// which is the layout the driver actually parses, and which for the two temp
// rate events differs from the same class's own buildCargo (a known pumpX2 bug
// where buildCargo writes tempRateId two bytes early). Captured records from
// real pumps agree with the parse offsets, so those are what we emit.
func encodeHistoryPayload(payload []byte, typeID int, fields map[string]interface{}) {
	switch typeID {
	case HistoryBolusActivated:
		// bolusId u16 @0, selectedIob u8 @2, iob f32 @4, bolusSize f32 @8
		binary.LittleEndian.PutUint16(payload[0:2], uint16(fieldInt(fields, "bolusId", "bolusID")))
		payload[2] = uint8(fieldInt(fields, "selectedIob"))
		putFloat32(payload[4:8], fieldFloat(fields, "iob"))
		putFloat32(payload[8:12], fieldFloat(fields, "bolusSize", "units", "unitsTotal"))

	case HistoryBolusCompleted:
		// completionStatusId u16 @0, bolusId u16 @2, iob f32 @4,
		// insulinDelivered f32 @8, insulinRequested f32 @12
		binary.LittleEndian.PutUint16(payload[0:2], uint16(fieldIntDefault(fields, 3, "completionStatusId")))
		binary.LittleEndian.PutUint16(payload[2:4], uint16(fieldInt(fields, "bolusId", "bolusID")))
		putFloat32(payload[4:8], fieldFloat(fields, "iob"))
		putFloat32(payload[8:12], fieldFloat(fields, "insulinDelivered", "unitsDelivered"))
		putFloat32(payload[12:16], fieldFloat(fields, "insulinRequested", "unitsTotal"))

	case HistoryTempRateActivated:
		// percent f32 @0, durationMilliseconds f32 @4, unknown u16 @8,
		// tempRateId u16 @10
		putFloat32(payload[0:4], fieldFloat(fields, "percent", "percentage"))
		putFloat32(payload[4:8], fieldFloat(fields, "durationMilliseconds", "durationMs"))
		binary.LittleEndian.PutUint16(payload[8:10], uint16(fieldInt(fields, "unknown")))
		binary.LittleEndian.PutUint16(payload[10:12], uint16(fieldInt(fields, "tempRateId")))

	case HistoryTempRateCompleted:
		// unknown u16 @0, tempRateId u16 @2, timeLeft u32 @4
		binary.LittleEndian.PutUint16(payload[0:2], uint16(fieldInt(fields, "unknown")))
		binary.LittleEndian.PutUint16(payload[2:4], uint16(fieldInt(fields, "tempRateId")))
		binary.LittleEndian.PutUint32(payload[4:8], uint32(fieldInt(fields, "timeLeft")))

	case HistoryPumpingSuspended:
		// preSuspendState u32 @0, insulinAmount u16 @4, reasonId u8 @6,
		// rpaTimeout u8 @7. 106 / reason 0 / 15-minute resume-pump alert are
		// what every observed user-initiated suspend carries.
		binary.LittleEndian.PutUint32(payload[0:4], uint32(fieldIntDefault(fields, 106, "preSuspendState")))
		binary.LittleEndian.PutUint16(payload[4:6], uint16(fieldInt(fields, "insulinAmount")))
		payload[6] = uint8(fieldInt(fields, "reasonId"))
		payload[7] = uint8(fieldIntDefault(fields, 15, "rpaTimeout"))

	case HistoryPumpingResumed:
		// preResumeState u32 @0, insulinAmount u16 @4
		binary.LittleEndian.PutUint32(payload[0:4], uint32(fieldIntDefault(fields, 100, "preResumeState")))
		binary.LittleEndian.PutUint16(payload[4:6], uint16(fieldInt(fields, "insulinAmount")))

	case HistoryBasalRateChange:
		// commandBasalRate f32 @0, baseBasalRate f32 @4, maxBasalRate f32 @8,
		// insulinDeliveryProfile u16 @12, changeTypeId u8 @14
		putFloat32(payload[0:4], fieldFloat(fields, "commandBasalRate", "commandedRate"))
		putFloat32(payload[4:8], fieldFloat(fields, "baseBasalRate", "profileRate"))
		putFloat32(payload[8:12], fieldFloat(fields, "maxBasalRate"))
		binary.LittleEndian.PutUint16(payload[12:14], uint16(fieldInt(fields, "insulinDeliveryProfile")))
		payload[14] = uint8(fieldInt(fields, "changeTypeId"))

	case HistoryBasalDelivery:
		// commandedRateSource u16 @0, basalDeliveryFlags u16 @2,
		// commandedRate u16 @4, profileBasalRate u16 @6, algorithmRate u16 @8,
		// tempRate u16 @10 -- all in milli-units/hour.
		binary.LittleEndian.PutUint16(payload[0:2], uint16(fieldInt(fields, "commandedRateSource")))
		binary.LittleEndian.PutUint16(payload[2:4], uint16(fieldInt(fields, "basalDeliveryFlags")))
		binary.LittleEndian.PutUint16(payload[4:6], uint16(fieldInt(fields, "commandedRate")))
		binary.LittleEndian.PutUint16(payload[6:8], uint16(fieldInt(fields, "profileBasalRate")))
		binary.LittleEndian.PutUint16(payload[8:10], uint16(fieldInt(fields, "algorithmRate")))
		binary.LittleEndian.PutUint16(payload[10:12], uint16(fieldInt(fields, "tempRate")))

	case HistoryBolusDelivery:
		// bolusID u16 @0, bolusDeliveryStatus u8 @2, bolusTypeBitmask u8 @3,
		// bolusSource u8 @4, remoteId u8 @5, requestedNow u16 @6,
		// requestedLater u16 @8, correction u16 @10,
		// extendedDurationRequested u16 @12, deliveredTotal u16 @14
		binary.LittleEndian.PutUint16(payload[0:2], uint16(fieldInt(fields, "bolusId", "bolusID")))
		payload[2] = uint8(fieldInt(fields, "bolusDeliveryStatus"))
		payload[3] = uint8(fieldInt(fields, "bolusTypeBitmask", "typeBitmask"))
		payload[4] = uint8(fieldInt(fields, "bolusSource", "sourceId"))
		payload[5] = uint8(fieldInt(fields, "remoteId"))
		binary.LittleEndian.PutUint16(payload[6:8], uint16(fieldInt(fields, "requestedNow")))
		binary.LittleEndian.PutUint16(payload[8:10], uint16(fieldInt(fields, "requestedLater")))
		binary.LittleEndian.PutUint16(payload[10:12], uint16(fieldInt(fields, "correction")))
		binary.LittleEndian.PutUint16(payload[12:14], uint16(fieldInt(fields, "extendedDurationRequested")))
		binary.LittleEndian.PutUint16(payload[14:16], uint16(fieldInt(fields, "deliveredTotal")))

	default:
		// No payload encoder: a zeroed payload with a correct header still
		// decodes as this event type at this time and sequence.
		writeRawPayload(payload, fields)
	}
}

// writeRawPayload lets a scenario supply payload bytes directly for an event
// type with no dedicated encoder, via a "raw" field holding a []byte.
func writeRawPayload(payload []byte, fields map[string]interface{}) {
	raw, ok := fields["raw"].([]byte)
	if !ok {
		return
	}
	copy(payload, raw)
}

func putFloat32(dst []byte, value float64) {
	binary.LittleEndian.PutUint32(dst, math.Float32bits(float32(value)))
}

// fieldInt reads the first present key as an integer, accepting any of the
// numeric types a JSON decode or a Go caller might have put in the map.
func fieldInt(fields map[string]interface{}, keys ...string) int {
	return fieldIntDefault(fields, 0, keys...)
}

func fieldIntDefault(fields map[string]interface{}, fallback int, keys ...string) int {
	for _, key := range keys {
		value, ok := fields[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case int:
			return v
		case int32:
			return int(v)
		case int64:
			return int(v)
		case uint8:
			return int(v)
		case uint16:
			return int(v)
		case uint32:
			return int(v)
		case uint64:
			return int(v)
		case float32:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return fallback
}

// fieldFloat reads the first present key as a float.
func fieldFloat(fields map[string]interface{}, keys ...string) float64 {
	for _, key := range keys {
		value, ok := fields[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int32:
			return float64(v)
		case int64:
			return float64(v)
		case uint32:
			return float64(v)
		}
	}
	return 0
}
