package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// ExtendedBolusStatusV2Response is ExtendedBolusStatusResponse with the
// pump's time since reset (seconds) appended. Duration is in seconds.
type ExtendedBolusStatusV2Response struct {
	messages.Base
	BolusStatus           int32 `pumpx2:"bolusStatus"`
	BolusID               int32 `pumpx2:"bolusId"`
	Timestamp             int64 `pumpx2:"timestamp"`
	RequestedVolume       int64 `pumpx2:"requestedVolume"`
	Duration              int64 `pumpx2:"duration"`
	BolusSource           int32 `pumpx2:"bolusSource"`
	SecondsSincePumpReset int64 `pumpx2:"secondsSincePumpReset"`
}

var extendedBolusStatusV2ResponseProps = messages.Props{OpCode: -73, Size: 22, Type: messages.Response, Request: "ExtendedBolusStatusV2Request"}

func init() {
	messages.Register("ExtendedBolusStatusV2Response", &ExtendedBolusStatusV2Response{}, extendedBolusStatusV2ResponseProps,
		messages.Constructor{
			Params: "int bolusStatus, int bolusId, long timestamp, long requestedVolume, long duration, int bolusSource, long secondsSincePumpReset",
			Func:   NewExtendedBolusStatusV2Response,
		},
	)
}

// NewExtendedBolusStatusV2Response builds the response.
func NewExtendedBolusStatusV2Response(bolusStatus, bolusID int32, timestamp, requestedVolume, duration int64, bolusSource int32,
	secondsSincePumpReset int64) *ExtendedBolusStatusV2Response {
	m := &ExtendedBolusStatusV2Response{
		BolusStatus:           bolusStatus,
		BolusID:               bolusID,
		Timestamp:             timestamp,
		RequestedVolume:       requestedVolume,
		Duration:              duration,
		BolusSource:           bolusSource,
		SecondsSincePumpReset: secondsSincePumpReset,
	}
	m.SetCargo(helpers.Combine(
		[]byte{byte(bolusStatus)},
		helpers.FirstTwoBytesLittleEndian(bolusID),
		[]byte{0, 0},
		helpers.ToUint32(timestamp),
		helpers.ToUint32(requestedVolume),
		helpers.ToUint32(duration),
		[]byte{byte(bolusSource)},
		helpers.ToUint32(secondsSincePumpReset),
	))
	return m
}

// Parse implements messages.Message. Bytes 3 and 4 are reserved.
func (m *ExtendedBolusStatusV2Response) Parse(raw []byte) error {
	if len(raw) != extendedBolusStatusV2ResponseProps.Size {
		return messages.InvalidLength("ExtendedBolusStatusV2Response", len(raw), extendedBolusStatusV2ResponseProps.Size)
	}
	m.SetCargo(raw)
	m.BolusStatus = int32(int8(raw[0]))
	m.BolusID = helpers.ReadShort(raw, 1)
	m.Timestamp = helpers.ReadUint32(raw, 5)
	m.RequestedVolume = helpers.ReadUint32(raw, 9)
	m.Duration = helpers.ReadUint32(raw, 13)
	m.BolusSource = int32(int8(raw[17]))
	m.SecondsSincePumpReset = helpers.ReadUint32(raw, 18)
	return nil
}
