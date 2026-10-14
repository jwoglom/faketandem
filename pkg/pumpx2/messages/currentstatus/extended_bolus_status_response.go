package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// ExtendedBolusStatusResponse is the extended bolus in progress: its status,
// ID, start (pump epoch seconds), volume requested (milliunits), duration
// and source. It reads all zeros when the latest bolus is not an extended
// one.
type ExtendedBolusStatusResponse struct {
	messages.Base
	BolusStatus     int32 `pumpx2:"bolusStatus"`
	BolusID         int32 `pumpx2:"bolusId"`
	Timestamp       int64 `pumpx2:"timestamp"`
	RequestedVolume int64 `pumpx2:"requestedVolume"`
	Duration        int64 `pumpx2:"duration"`
	BolusSource     int32 `pumpx2:"bolusSource"`
}

var extendedBolusStatusResponseProps = messages.Props{OpCode: 47, Size: 18, Type: messages.Response, Request: "ExtendedBolusStatusRequest"}

func init() {
	messages.Register("ExtendedBolusStatusResponse", &ExtendedBolusStatusResponse{}, extendedBolusStatusResponseProps,
		messages.Constructor{
			Params: "int bolusStatus, int bolusId, long timestamp, long requestedVolume, long duration, int bolusSource",
			Func:   NewExtendedBolusStatusResponse,
		},
	)
}

// NewExtendedBolusStatusResponse builds the response. As pumpX2 does, it
// writes the timestamp from byte 3 and pads the end, while Parse reads it
// from byte 5, so what it builds parses back to other values.
func NewExtendedBolusStatusResponse(bolusStatus, bolusID int32, timestamp, requestedVolume, duration int64, bolusSource int32) *ExtendedBolusStatusResponse {
	m := &ExtendedBolusStatusResponse{
		BolusStatus:     bolusStatus,
		BolusID:         bolusID,
		Timestamp:       timestamp,
		RequestedVolume: requestedVolume,
		Duration:        duration,
		BolusSource:     bolusSource,
	}
	m.SetCargo(helpers.Combine(
		[]byte{byte(bolusStatus)},
		helpers.FirstTwoBytesLittleEndian(bolusID),
		helpers.ToUint32(timestamp),
		helpers.ToUint32(requestedVolume),
		helpers.ToUint32(duration),
		[]byte{byte(bolusSource), 0, 0},
	))
	return m
}

// Parse implements messages.Message.
func (m *ExtendedBolusStatusResponse) Parse(raw []byte) error {
	if len(raw) != extendedBolusStatusResponseProps.Size {
		return messages.InvalidLength("ExtendedBolusStatusResponse", len(raw), extendedBolusStatusResponseProps.Size)
	}
	m.SetCargo(raw)
	m.BolusStatus = int32(int8(raw[0]))
	m.BolusID = helpers.ReadShort(raw, 1)
	m.Timestamp = helpers.ReadUint32(raw, 5)
	m.RequestedVolume = helpers.ReadUint32(raw, 9)
	m.Duration = helpers.ReadUint32(raw, 13)
	m.BolusSource = int32(int8(raw[17]))
	return nil
}
