package currentstatus

import (
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// LastBolusStatusV3Response is a Mobi's last standard bolus and last
// extended bolus: each one's status (0 invalid, 3 stopping, 4 finished), ID,
// time, volumes delivered and requested, end reason, source, type and the
// pump's seconds since reset. LastBolusTypeBitmask says which are present:
// bit 0 the standard, bit 1 the extended. Timestamps are pump epoch
// seconds, volumes milliunits, and ExtendedBolusDuration seconds.
//
// pumpX2's get*BolusSource and get*BolusTypes read the source and type as
// BolusDeliveryHistoryLog's enums, which have no Go type yet.
type LastBolusStatusV3Response struct {
	messages.Base
	LastBolusTypeBitmask               int32  `pumpx2:"lastBolusTypeBitmask"`
	StandardBolusStatusID              int32  `pumpx2:"standardBolusStatusId"`
	StandardBolusID                    int32  `pumpx2:"standardBolusId"`
	StandardUnknown                    []byte `pumpx2:"standardUnknown"`
	StandardBolusTimestamp             int64  `pumpx2:"standardBolusTimestamp"`
	StandardBolusDeliveredVolume       int64  `pumpx2:"standardBolusDeliveredVolume"`
	StandardBolusEndReasonID           int32  `pumpx2:"standardBolusEndReasonId"`
	StandardBolusSourceID              int32  `pumpx2:"standardBolusSourceId"`
	StandardBolusTypeBitmask           int32  `pumpx2:"standardBolusTypeBitmask"`
	StandardBolusRequestedVolume       int64  `pumpx2:"standardBolusRequestedVolume"`
	StandardBolusSecondsSincePumpReset int64  `pumpx2:"standardBolusSecondsSincePumpReset"`
	ExtendedBolusStatusID              int32  `pumpx2:"extendedBolusStatusId"`
	ExtendedBolusID                    int32  `pumpx2:"extendedBolusId"`
	ExtendedUnknown                    []byte `pumpx2:"extendedUnknown"`
	ExtendedBolusTimestamp             int64  `pumpx2:"extendedBolusTimestamp"`
	ExtendedBolusDeliveredVolume       int64  `pumpx2:"extendedBolusDeliveredVolume"`
	ExtendedBolusEndReasonID           int32  `pumpx2:"extendedBolusEndReasonId"`
	ExtendedBolusSourceID              int32  `pumpx2:"extendedBolusSourceId"`
	ExtendedBolusTypeBitmask           int32  `pumpx2:"extendedBolusTypeBitmask"`
	ExtendedBolusRequestedVolume       int64  `pumpx2:"extendedBolusRequestedVolume"`
	ExtendedBolusSecondsSincePumpReset int64  `pumpx2:"extendedBolusSecondsSincePumpReset"`
	ExtendedBolusDuration              int64  `pumpx2:"extendedBolusDuration"`
}

var lastBolusStatusV3ResponseProps = messages.Props{OpCode: -69, Size: 53, Type: messages.Response, MinAPI: "MOBI_API_V3_5", Request: "LastBolusStatusV3Request"}

func init() {
	messages.Register("LastBolusStatusV3Response", &LastBolusStatusV3Response{}, lastBolusStatusV3ResponseProps,
		messages.Constructor{
			Params: "int lastBolusTypeBitmask, int standardBolusStatusId, int standardBolusId, byte[] standardUnknown, long standardBolusTimestamp, " +
				"long standardBolusDeliveredVolume, int standardBolusEndReasonId, int standardBolusSourceId, int standardBolusTypeBitmask, " +
				"long standardBolusRequestedVolume, long standardBolusSecondsSincePumpReset, int extendedBolusStatusId, int extendedBolusId, " +
				"byte[] extendedUnknown, long extendedBolusTimestamp, long extendedBolusDeliveredVolume, int extendedBolusEndReasonId, " +
				"int extendedBolusSourceId, int extendedBolusTypeBitmask, long extendedBolusRequestedVolume, long extendedBolusSecondsSincePumpReset, " +
				"long extendedBolusDuration",
			Func: NewLastBolusStatusV3Response,
		},
	)
}

// NewLastBolusStatusV3Response builds the response. Each unknown is the two
// bytes after its bolus ID; pumpX2 writes whatever it is given there, so
// other lengths build a payload of another size.
func NewLastBolusStatusV3Response(
	lastBolusTypeBitmask, standardBolusStatusID, standardBolusID int32, standardUnknown []byte,
	standardBolusTimestamp, standardBolusDeliveredVolume int64,
	standardBolusEndReasonID, standardBolusSourceID, standardBolusTypeBitmask int32,
	standardBolusRequestedVolume, standardBolusSecondsSincePumpReset int64,
	extendedBolusStatusID, extendedBolusID int32, extendedUnknown []byte,
	extendedBolusTimestamp, extendedBolusDeliveredVolume int64,
	extendedBolusEndReasonID, extendedBolusSourceID, extendedBolusTypeBitmask int32,
	extendedBolusRequestedVolume, extendedBolusSecondsSincePumpReset, extendedBolusDuration int64,
) *LastBolusStatusV3Response {
	m := &LastBolusStatusV3Response{
		LastBolusTypeBitmask:               lastBolusTypeBitmask,
		StandardBolusStatusID:              standardBolusStatusID,
		StandardBolusID:                    standardBolusID,
		StandardUnknown:                    standardUnknown,
		StandardBolusTimestamp:             standardBolusTimestamp,
		StandardBolusDeliveredVolume:       standardBolusDeliveredVolume,
		StandardBolusEndReasonID:           standardBolusEndReasonID,
		StandardBolusSourceID:              standardBolusSourceID,
		StandardBolusTypeBitmask:           standardBolusTypeBitmask,
		StandardBolusRequestedVolume:       standardBolusRequestedVolume,
		StandardBolusSecondsSincePumpReset: standardBolusSecondsSincePumpReset,
		ExtendedBolusStatusID:              extendedBolusStatusID,
		ExtendedBolusID:                    extendedBolusID,
		ExtendedUnknown:                    extendedUnknown,
		ExtendedBolusTimestamp:             extendedBolusTimestamp,
		ExtendedBolusDeliveredVolume:       extendedBolusDeliveredVolume,
		ExtendedBolusEndReasonID:           extendedBolusEndReasonID,
		ExtendedBolusSourceID:              extendedBolusSourceID,
		ExtendedBolusTypeBitmask:           extendedBolusTypeBitmask,
		ExtendedBolusRequestedVolume:       extendedBolusRequestedVolume,
		ExtendedBolusSecondsSincePumpReset: extendedBolusSecondsSincePumpReset,
		ExtendedBolusDuration:              extendedBolusDuration,
	}
	m.SetCargo(helpers.Combine(
		[]byte{byte(lastBolusTypeBitmask), byte(standardBolusStatusID)},
		helpers.FirstTwoBytesLittleEndian(standardBolusID),
		standardUnknown,
		helpers.ToUint32(standardBolusTimestamp),
		helpers.ToUint32(standardBolusDeliveredVolume),
		[]byte{byte(standardBolusEndReasonID), byte(standardBolusSourceID), byte(standardBolusTypeBitmask)},
		helpers.ToUint32(standardBolusRequestedVolume),
		helpers.ToUint32(standardBolusSecondsSincePumpReset),
		[]byte{byte(extendedBolusStatusID)},
		helpers.FirstTwoBytesLittleEndian(extendedBolusID),
		extendedUnknown,
		helpers.ToUint32(extendedBolusTimestamp),
		helpers.ToUint32(extendedBolusDeliveredVolume),
		[]byte{byte(extendedBolusEndReasonID), byte(extendedBolusSourceID), byte(extendedBolusTypeBitmask)},
		helpers.ToUint32(extendedBolusRequestedVolume),
		helpers.ToUint32(extendedBolusSecondsSincePumpReset),
		helpers.ToUint32(extendedBolusDuration),
	))
	return m
}

// Parse implements messages.Message.
func (m *LastBolusStatusV3Response) Parse(raw []byte) error {
	if len(raw) != lastBolusStatusV3ResponseProps.Size {
		return messages.InvalidLength("LastBolusStatusV3Response", len(raw), lastBolusStatusV3ResponseProps.Size)
	}
	m.SetCargo(raw)
	m.LastBolusTypeBitmask = int32(raw[0])
	m.StandardBolusStatusID = int32(raw[1])
	m.StandardBolusID = helpers.ReadShort(raw, 2)
	m.StandardUnknown = []byte{raw[4], raw[5]}
	m.StandardBolusTimestamp = helpers.ReadUint32(raw, 6)
	m.StandardBolusDeliveredVolume = helpers.ReadUint32(raw, 10)
	m.StandardBolusEndReasonID = int32(raw[14])
	m.StandardBolusSourceID = int32(raw[15])
	m.StandardBolusTypeBitmask = int32(raw[16])
	m.StandardBolusRequestedVolume = helpers.ReadUint32(raw, 17)
	m.StandardBolusSecondsSincePumpReset = helpers.ReadUint32(raw, 21)
	m.ExtendedBolusStatusID = int32(raw[25])
	m.ExtendedBolusID = helpers.ReadShort(raw, 26)
	m.ExtendedUnknown = []byte{raw[28], raw[29]}
	m.ExtendedBolusTimestamp = helpers.ReadUint32(raw, 30)
	m.ExtendedBolusDeliveredVolume = helpers.ReadUint32(raw, 34)
	m.ExtendedBolusEndReasonID = int32(raw[38])
	m.ExtendedBolusSourceID = int32(raw[39])
	m.ExtendedBolusTypeBitmask = int32(raw[40])
	m.ExtendedBolusRequestedVolume = helpers.ReadUint32(raw, 41)
	m.ExtendedBolusSecondsSincePumpReset = helpers.ReadUint32(raw, 45)
	m.ExtendedBolusDuration = helpers.ReadUint32(raw, 49)
	return nil
}

// IsStandardBolusPresent reports whether the response holds a standard
// bolus.
func (m *LastBolusStatusV3Response) IsStandardBolusPresent() bool {
	return m.LastBolusTypeBitmask&0x01 != 0
}

// IsExtendedBolusPresent reports whether the response holds an extended
// bolus.
func (m *LastBolusStatusV3Response) IsExtendedBolusPresent() bool {
	return m.LastBolusTypeBitmask&0x02 != 0
}

// StandardBolusTimestampInstant is StandardBolusTimestamp as a time, failing
// past the range of Java's Instant.
func (m *LastBolusStatusV3Response) StandardBolusTimestampInstant() (time.Time, error) {
	return helpers.FromJan12008EpochSecondsToDate(m.StandardBolusTimestamp)
}

// ExtendedBolusTimestampInstant is ExtendedBolusTimestamp as a time, failing
// past the range of Java's Instant.
func (m *LastBolusStatusV3Response) ExtendedBolusTimestampInstant() (time.Time, error) {
	return helpers.FromJan12008EpochSecondsToDate(m.ExtendedBolusTimestamp)
}
