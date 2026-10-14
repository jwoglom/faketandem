package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// SetTempRateResponse is the pump's answer to a SetTempRateRequest: Status
// 0 is success, and TempRateID is the ID the history log's
// TempRateActivated record carries.
type SetTempRateResponse struct {
	messages.Base
	Status     int32 `pumpx2:"status"`
	TempRateID int32 `pumpx2:"tempRateId"`
}

var setTempRateResponseProps = messages.Props{OpCode: -91, Size: 4, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	MinAPI: "MOBI_API_V3_5", SupportedDevices: "MOBI_ONLY", Request: "SetTempRateRequest"}

func init() {
	messages.Register("SetTempRateResponse", &SetTempRateResponse{}, setTempRateResponseProps,
		messages.Constructor{Params: "int status, int tempRateId", Func: NewSetTempRateResponse},
		messages.Constructor{Params: "byte[] raw", Func: NewSetTempRateResponseFromCargo},
	)
}

// NewSetTempRateResponse builds the response. The fields are what the
// payload reads back as.
func NewSetTempRateResponse(status, tempRateID int32) *SetTempRateResponse {
	m := &SetTempRateResponse{}
	// The last byte is not decoded; it is 0 in a captured success.
	m.decode(helpers.Combine([]byte{byte(status)}, helpers.FirstTwoBytesLittleEndian(tempRateID), []byte{0}))
	return m
}

// NewSetTempRateResponseFromCargo parses a response's payload.
func NewSetTempRateResponseFromCargo(raw []byte) (*SetTempRateResponse, error) {
	m := &SetTempRateResponse{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *SetTempRateResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setTempRateResponseProps, raw)
	if len(raw) != setTempRateResponseProps.Size {
		return messages.InvalidLength("SetTempRateResponse", len(raw), setTempRateResponseProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *SetTempRateResponse) decode(raw []byte) {
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	m.TempRateID = helpers.ReadShort(raw, 1)
}
