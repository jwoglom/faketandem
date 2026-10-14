package currentstatus

import (
	"fmt"
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CurrentBolusStatusResponse is the bolus being requested or delivered: its
// status (a CurrentBolusStatus ID), ID, when it entered that status (pump
// epoch seconds), the volume requested (milliunits), and its source and type.
// When no bolus is current it reads all zeros (Valid is false), and
// LastBolusStatusRequest reports the last one.
//
// BolusSourceID and BolusTypeBitmask are BolusDeliveryHistoryLog's
// BolusSource and BolusType; pumpX2's getters for those enums are left out
// until the history log package has them.
type CurrentBolusStatusResponse struct {
	messages.Base
	StatusID         int32 `pumpx2:"statusId"`
	BolusID          int32 `pumpx2:"bolusId"`
	Timestamp        int64 `pumpx2:"timestamp"`
	RequestedVolume  int64 `pumpx2:"requestedVolume"`
	BolusSourceID    int32 `pumpx2:"bolusSourceId"`
	BolusTypeBitmask int32 `pumpx2:"bolusTypeBitmask"`
}

var currentBolusStatusResponseProps = messages.Props{OpCode: 45, Size: 15, Type: messages.Response, Request: "CurrentBolusStatusRequest"}

func init() {
	messages.Register("CurrentBolusStatusResponse", &CurrentBolusStatusResponse{}, currentBolusStatusResponseProps,
		messages.Constructor{
			Params: "int statusId, int bolusId, long timestamp, long requestedVolume, int bolusSourceId, int bolusTypeBitmask",
			Func:   NewCurrentBolusStatusResponse,
		},
	)
}

// NewCurrentBolusStatusResponse builds the response.
func NewCurrentBolusStatusResponse(statusID, bolusID int32, timestamp, requestedVolume int64, bolusSourceID, bolusTypeBitmask int32) *CurrentBolusStatusResponse {
	m := &CurrentBolusStatusResponse{
		StatusID:         statusID,
		BolusID:          bolusID,
		Timestamp:        timestamp,
		RequestedVolume:  requestedVolume,
		BolusSourceID:    bolusSourceID,
		BolusTypeBitmask: bolusTypeBitmask,
	}
	m.SetCargo(helpers.Combine(
		[]byte{byte(statusID)},
		helpers.FirstTwoBytesLittleEndian(bolusID),
		[]byte{0, 0},
		helpers.ToUint32(timestamp),
		helpers.ToUint32(requestedVolume),
		[]byte{byte(bolusSourceID), byte(bolusTypeBitmask)},
	))
	return m
}

// Parse implements messages.Message.
func (m *CurrentBolusStatusResponse) Parse(raw []byte) error {
	if len(raw) != currentBolusStatusResponseProps.Size {
		return messages.InvalidLength("CurrentBolusStatusResponse", len(raw), currentBolusStatusResponseProps.Size)
	}
	m.SetCargo(raw)
	m.StatusID = int32(int8(raw[0]))
	m.BolusID = helpers.ReadShort(raw, 1)
	m.Timestamp = helpers.ReadUint32(raw, 5)
	m.RequestedVolume = helpers.ReadUint32(raw, 9)
	m.BolusSourceID = int32(int8(raw[13]))
	m.BolusTypeBitmask = int32(int8(raw[14]))
	return nil
}

// Status is the bolus's status, or nil for an ID that is none.
func (m *CurrentBolusStatusResponse) Status() *CurrentBolusStatus {
	return CurrentBolusStatusFromID(m.StatusID)
}

// Time is when the bolus entered its status, as pumpX2 reads Timestamp: the
// pump's clock, as UTC. For a bolus delivering, it is when delivery started.
func (m *CurrentBolusStatusResponse) Time() time.Time {
	return time.Unix(helpers.FromJan12008ToUnixEpochSeconds(m.Timestamp), 0).UTC()
}

// Valid reports whether a bolus is current: false when the status is
// ALREADY_DELIVERED_OR_INVALID and the bolus ID and timestamp are 0.
func (m *CurrentBolusStatusResponse) Valid() bool {
	return m.StatusID != CurrentBolusStatusAlreadyDeliveredOrInvalid.ID() || m.BolusID != 0 || m.Timestamp != 0
}

// CurrentBolusStatus is where the current bolus is.
type CurrentBolusStatus int32

// The bolus statuses, by their wire IDs.
const (
	CurrentBolusStatusRequesting                CurrentBolusStatus = 2
	CurrentBolusStatusDelivering                CurrentBolusStatus = 1
	CurrentBolusStatusAlreadyDeliveredOrInvalid CurrentBolusStatus = 0
)

// CurrentBolusStatusFromID is the status with wire ID id, or nil for an ID
// that is none.
func CurrentBolusStatusFromID(id int32) *CurrentBolusStatus {
	if id < 0 || id > 2 {
		return nil
	}
	s := CurrentBolusStatus(id)
	return &s
}

// ID is the status's wire ID.
func (s CurrentBolusStatus) ID() int32 { return int32(s) }

func (s CurrentBolusStatus) String() string {
	switch s {
	case CurrentBolusStatusRequesting:
		return "REQUESTING"
	case CurrentBolusStatusDelivering:
		return "DELIVERING"
	case CurrentBolusStatusAlreadyDeliveredOrInvalid:
		return "ALREADY_DELIVERED_OR_INVALID"
	}
	return fmt.Sprintf("CurrentBolusStatus(%d)", int32(s))
}
