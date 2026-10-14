package currentstatus

import (
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// LastBolusStatusV2Response is LastBolusStatusResponse with the volume
// requested for the bolus, in milliunits, after it. The two bytes after the
// bolus ID are not kept.
//
// As for LastBolusStatusResponse, pumpX2's getBolusSource and getBolusType
// have no Go counterpart yet.
type LastBolusStatusV2Response struct {
	messages.Base
	Status                int32 `pumpx2:"status"`
	BolusID               int32 `pumpx2:"bolusId"`
	Timestamp             int64 `pumpx2:"timestamp"`
	DeliveredVolume       int64 `pumpx2:"deliveredVolume"`
	BolusStatusID         int32 `pumpx2:"bolusStatusId"`
	BolusSourceID         int32 `pumpx2:"bolusSourceId"`
	BolusTypeBitmask      int32 `pumpx2:"bolusTypeBitmask"`
	ExtendedBolusDuration int64 `pumpx2:"extendedBolusDuration"`
	RequestedVolume       int64 `pumpx2:"requestedVolume"`
}

var lastBolusStatusV2ResponseProps = messages.Props{OpCode: -91, Size: 24, Type: messages.Response, MinAPI: "API_V2_5", Request: "LastBolusStatusV2Request"}

func init() {
	messages.Register("LastBolusStatusV2Response", &LastBolusStatusV2Response{}, lastBolusStatusV2ResponseProps,
		messages.Constructor{
			Params: "int status, int bolusId, long timestamp, long deliveredVolume, int bolusStatusId, int bolusSourceId, int bolusTypeBitmask, " +
				"long extendedBolusDuration, long requestedVolume",
			Func: NewLastBolusStatusV2Response,
		},
	)
}

// NewLastBolusStatusV2Response builds the response, with zeros after the
// bolus ID.
func NewLastBolusStatusV2Response(status, bolusID int32, timestamp, deliveredVolume int64, bolusStatusID, bolusSourceID, bolusTypeBitmask int32,
	extendedBolusDuration, requestedVolume int64) *LastBolusStatusV2Response {
	m := &LastBolusStatusV2Response{
		Status:                status,
		BolusID:               bolusID,
		Timestamp:             timestamp,
		DeliveredVolume:       deliveredVolume,
		BolusStatusID:         bolusStatusID,
		BolusSourceID:         bolusSourceID,
		BolusTypeBitmask:      bolusTypeBitmask,
		ExtendedBolusDuration: extendedBolusDuration,
		RequestedVolume:       requestedVolume,
	}
	m.SetCargo(helpers.Combine(
		lastBolusStatusCargo(status, bolusID, []byte{0, 0}, timestamp, deliveredVolume, bolusStatusID, bolusSourceID, bolusTypeBitmask, extendedBolusDuration),
		helpers.ToUint32(requestedVolume),
	))
	return m
}

// Parse implements messages.Message.
func (m *LastBolusStatusV2Response) Parse(raw []byte) error {
	if len(raw) != lastBolusStatusV2ResponseProps.Size {
		return messages.InvalidLength("LastBolusStatusV2Response", len(raw), lastBolusStatusV2ResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	m.BolusID = helpers.ReadShort(raw, 1)
	m.Timestamp = helpers.ReadUint32(raw, 5)
	m.DeliveredVolume = helpers.ReadUint32(raw, 9)
	m.BolusStatusID = int32(int8(raw[13]))
	m.BolusSourceID = int32(int8(raw[14]))
	m.BolusTypeBitmask = int32(int8(raw[15]))
	m.ExtendedBolusDuration = helpers.ReadUint32(raw, 16)
	m.RequestedVolume = helpers.ReadUint32(raw, 20)
	return nil
}

// BolusStatus is BolusStatusID as a BolusStatus, or nil for an ID that is
// none.
func (m *LastBolusStatusV2Response) BolusStatus() *BolusStatus {
	return BolusStatusFromID(m.BolusStatusID)
}

// TimestampInstant is Timestamp as a time, failing past the range of Java's
// Instant.
func (m *LastBolusStatusV2Response) TimestampInstant() (time.Time, error) {
	return helpers.FromJan12008EpochSecondsToDate(m.Timestamp)
}
