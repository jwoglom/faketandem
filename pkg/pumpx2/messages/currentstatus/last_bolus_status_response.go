package currentstatus

import (
	"fmt"
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// LastBolusStatusResponse is the pump's last bolus: its ID, when it ended
// (pump epoch seconds), the volume delivered in milliunits, how it ended
// (BolusStatusID), its source and type, and its extended part's duration.
// LastBolusStatusV2Response adds the volume requested.
//
// Status is not a command status: pumpX2 sees 1 alongside valid bolus data.
// pumpX2's getBolusSource and getBolusType read BolusSourceID and
// BolusTypeBitmask as BolusDeliveryHistoryLog's enums, which have no Go
// type yet.
type LastBolusStatusResponse struct {
	messages.Base
	Status                int32  `pumpx2:"status"`
	BolusID               int32  `pumpx2:"bolusId"`
	Timestamp             int64  `pumpx2:"timestamp"`
	DeliveredVolume       int64  `pumpx2:"deliveredVolume"`
	BolusStatusID         int32  `pumpx2:"bolusStatusId"`
	BolusSourceID         int32  `pumpx2:"bolusSourceId"`
	BolusTypeBitmask      int32  `pumpx2:"bolusTypeBitmask"`
	ExtendedBolusDuration int64  `pumpx2:"extendedBolusDuration"`
	Unknown               []byte `pumpx2:"unknown"`
}

var lastBolusStatusResponseProps = messages.Props{OpCode: 49, Size: 20, Type: messages.Response, Request: "LastBolusStatusRequest"}

func init() {
	messages.Register("LastBolusStatusResponse", &LastBolusStatusResponse{}, lastBolusStatusResponseProps,
		messages.Constructor{
			Params: "int status, int bolusId, long timestamp, long deliveredVolume, int bolusStatusId, int bolusSourceId, int bolusTypeBitmask, " +
				"long extendedBolusDuration, byte[] unknown",
			Func: NewLastBolusStatusResponse,
		},
	)
}

// NewLastBolusStatusResponse builds the response. unknown is the two bytes
// after the bolus ID; pumpX2 writes whatever it is given there, so other
// lengths build a payload of another size.
func NewLastBolusStatusResponse(status, bolusID int32, timestamp, deliveredVolume int64, bolusStatusID, bolusSourceID, bolusTypeBitmask int32,
	extendedBolusDuration int64, unknown []byte) *LastBolusStatusResponse {
	m := &LastBolusStatusResponse{
		Status:                status,
		BolusID:               bolusID,
		Timestamp:             timestamp,
		DeliveredVolume:       deliveredVolume,
		BolusStatusID:         bolusStatusID,
		BolusSourceID:         bolusSourceID,
		BolusTypeBitmask:      bolusTypeBitmask,
		ExtendedBolusDuration: extendedBolusDuration,
		Unknown:               unknown,
	}
	m.SetCargo(lastBolusStatusCargo(status, bolusID, unknown, timestamp, deliveredVolume, bolusStatusID, bolusSourceID, bolusTypeBitmask, extendedBolusDuration))
	return m
}

// Parse implements messages.Message.
func (m *LastBolusStatusResponse) Parse(raw []byte) error {
	if len(raw) != lastBolusStatusResponseProps.Size {
		return messages.InvalidLength("LastBolusStatusResponse", len(raw), lastBolusStatusResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	m.BolusID = helpers.ReadShort(raw, 1)
	m.Unknown = []byte{raw[3], raw[4]}
	m.Timestamp = helpers.ReadUint32(raw, 5)
	m.DeliveredVolume = helpers.ReadUint32(raw, 9)
	m.BolusStatusID = int32(int8(raw[13]))
	m.BolusSourceID = int32(int8(raw[14]))
	m.BolusTypeBitmask = int32(int8(raw[15]))
	m.ExtendedBolusDuration = helpers.ReadUint32(raw, 16)
	return nil
}

// BolusStatus is BolusStatusID as a BolusStatus, or nil for an ID that is
// none.
func (m *LastBolusStatusResponse) BolusStatus() *BolusStatus {
	return BolusStatusFromID(m.BolusStatusID)
}

// TimestampInstant is Timestamp as a time, failing past the range of Java's
// Instant.
func (m *LastBolusStatusResponse) TimestampInstant() (time.Time, error) {
	return helpers.FromJan12008EpochSecondsToDate(m.Timestamp)
}

// lastBolusStatusCargo is the first 20 bytes LastBolusStatusResponse and
// LastBolusStatusV2Response share.
func lastBolusStatusCargo(status, bolusID int32, unknown []byte, timestamp, deliveredVolume int64, bolusStatusID, bolusSourceID, bolusTypeBitmask int32,
	extendedBolusDuration int64) []byte {
	return helpers.Combine(
		[]byte{byte(status)},
		helpers.FirstTwoBytesLittleEndian(bolusID),
		unknown,
		helpers.ToUint32(timestamp),
		helpers.ToUint32(deliveredVolume),
		[]byte{byte(bolusStatusID), byte(bolusSourceID), byte(bolusTypeBitmask)},
		helpers.ToUint32(extendedBolusDuration),
	)
}

// BolusStatus is how a bolus ended: pumpX2's
// LastBolusStatusAbstractResponse.BolusStatus. The IDs match the completion
// status of the BolusCompleted and BolexCompleted history log entries.
// pumpX2 has seen only COMPLETE, STOPPED_USER_TERMINATED (which a
// CancelBolusRequest also gives) and STOPPED_ALARM.
type BolusStatus int32

// The bolus statuses.
const (
	BolusStatusStoppedUserTerminated BolusStatus = iota
	BolusStatusStoppedAlarm
	BolusStatusStoppedMalfunction
	BolusStatusComplete
	BolusStatusStoppedWireless
	BolusStatusRejectedWireless
	BolusStatusTerminatedPLGS
)

var bolusStatusNames = []string{
	"STOPPED_USER_TERMINATED", "STOPPED_ALARM", "STOPPED_MALFUNCTION", "COMPLETE", "STOPPED_WIRELESS", "REJECTED_WIRELESS", "TERMINATED_PLGS",
}

// BolusStatusFromID is the status with wire ID id, or nil for an ID that is
// none.
func BolusStatusFromID(id int32) *BolusStatus {
	if id < 0 || int(id) >= len(bolusStatusNames) {
		return nil
	}
	s := BolusStatus(id)
	return &s
}

// ID is the status's wire ID.
func (s BolusStatus) ID() int32 { return int32(s) }

func (s BolusStatus) String() string {
	if s >= 0 && int(s) < len(bolusStatusNames) {
		return bolusStatusNames[s]
	}
	return fmt.Sprintf("BolusStatus(%d)", int32(s))
}
