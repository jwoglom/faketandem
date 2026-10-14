package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetSleepScheduleResponse is the pump's answer to a SetSleepScheduleRequest:
// Status 0 is success.
type SetSleepScheduleResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var setSleepScheduleResponseProps = messages.Props{OpCode: -49, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	MinAPI: "MOBI_API_V3_5", SupportedDevices: "MOBI_ONLY", Request: "SetSleepScheduleRequest"}

func init() {
	messages.Register("SetSleepScheduleResponse", &SetSleepScheduleResponse{}, setSleepScheduleResponseProps,
		messages.Constructor{Params: "int status", Func: NewSetSleepScheduleResponse},
		messages.Constructor{Params: "byte[] raw", Func: NewSetSleepScheduleResponseFromCargo},
	)
}

// NewSetSleepScheduleResponse builds the response. Status is what the
// payload's byte reads back as.
func NewSetSleepScheduleResponse(status int32) *SetSleepScheduleResponse {
	m := &SetSleepScheduleResponse{}
	m.decode([]byte{byte(status)})
	return m
}

// NewSetSleepScheduleResponseFromCargo parses a response's payload.
func NewSetSleepScheduleResponseFromCargo(raw []byte) (*SetSleepScheduleResponse, error) {
	m := &SetSleepScheduleResponse{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *SetSleepScheduleResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setSleepScheduleResponseProps, raw)
	if len(raw) != setSleepScheduleResponseProps.Size {
		return messages.InvalidLength("SetSleepScheduleResponse", len(raw), setSleepScheduleResponseProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *SetSleepScheduleResponse) decode(raw []byte) {
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
}
