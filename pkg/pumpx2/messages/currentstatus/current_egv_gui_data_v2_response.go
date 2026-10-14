package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CurrentEgvGuiDataV2Response is the latest CGM reading, laid out as
// CurrentEGVGuiDataResponse: when it was taken (pump epoch seconds), the
// reading (mg/dL), its status (an EGVStatus ID) and the trend rate.
type CurrentEgvGuiDataV2Response struct {
	messages.Base
	BgReadingTimestampSeconds int64      `pumpx2:"bgReadingTimestampSeconds"`
	CgmReading                int32      `pumpx2:"cgmReading"`
	EgvStatusID               int32      `pumpx2:"egvStatusId"`
	EgvStatus                 *EGVStatus `pumpx2:"egvStatus"`
	TrendRate                 int32      `pumpx2:"trendRate"`
}

var currentEgvGuiDataV2ResponseProps = messages.Props{OpCode: -63, Size: 8, Type: messages.Response, MinAPI: "API_FUTURE", Request: "CurrentEgvGuiDataV2Request"}

func init() {
	messages.Register("CurrentEgvGuiDataV2Response", &CurrentEgvGuiDataV2Response{}, currentEgvGuiDataV2ResponseProps,
		messages.Constructor{Params: "long bgReadingTimestampSeconds, int cgmReading, int egvStatusId, int trendRate", Func: NewCurrentEgvGuiDataV2Response},
	)
}

// NewCurrentEgvGuiDataV2Response builds the response.
func NewCurrentEgvGuiDataV2Response(bgReadingTimestampSeconds int64, cgmReading, egvStatusID, trendRate int32) *CurrentEgvGuiDataV2Response {
	m := &CurrentEgvGuiDataV2Response{
		BgReadingTimestampSeconds: bgReadingTimestampSeconds,
		CgmReading:                cgmReading,
		EgvStatusID:               egvStatusID,
		EgvStatus:                 EGVStatusFromID(egvStatusID),
		TrendRate:                 trendRate,
	}
	m.SetCargo(egvGuiDataCargo(bgReadingTimestampSeconds, cgmReading, egvStatusID, trendRate))
	return m
}

// Parse implements messages.Message.
func (m *CurrentEgvGuiDataV2Response) Parse(raw []byte) error {
	if len(raw) != currentEgvGuiDataV2ResponseProps.Size {
		return messages.InvalidLength("CurrentEgvGuiDataV2Response", len(raw), currentEgvGuiDataV2ResponseProps.Size)
	}
	m.SetCargo(raw)
	m.BgReadingTimestampSeconds = helpers.ReadUint32(raw, 0)
	m.CgmReading = helpers.ReadShort(raw, 4)
	m.EgvStatusID = int32(int8(raw[6]))
	m.EgvStatus = EGVStatusFromID(m.EgvStatusID)
	m.TrendRate = int32(int8(raw[7]))
	return nil
}
