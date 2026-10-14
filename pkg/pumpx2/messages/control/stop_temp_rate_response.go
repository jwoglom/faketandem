package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// StopTempRateResponse is the pump's answer to a StopTempRateRequest:
// Status 0 is success, and TempRateID is the stopped temp rate's ID, which
// the history log's TempRateCompleted record carries.
type StopTempRateResponse struct {
	messages.Base
	Status     int32 `pumpx2:"status"`
	TempRateID int32 `pumpx2:"tempRateId"`
}

var stopTempRateResponseProps = messages.Props{OpCode: -89, Size: 3, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	MinAPI: "MOBI_API_V3_5", SupportedDevices: "MOBI_ONLY", Request: "StopTempRateRequest"}

func init() {
	messages.Register("StopTempRateResponse", &StopTempRateResponse{}, stopTempRateResponseProps,
		messages.Constructor{Params: "int status, int tempRateId", Func: NewStopTempRateResponse},
		messages.Constructor{Params: "byte[] raw", Func: NewStopTempRateResponseFromCargo},
	)
}

// NewStopTempRateResponse builds the response. The fields are what the
// payload reads back as.
func NewStopTempRateResponse(status, tempRateID int32) *StopTempRateResponse {
	m := &StopTempRateResponse{}
	m.decode(helpers.Combine([]byte{byte(status)}, helpers.FirstTwoBytesLittleEndian(tempRateID)))
	return m
}

// NewStopTempRateResponseFromCargo parses a response's payload.
func NewStopTempRateResponseFromCargo(raw []byte) (*StopTempRateResponse, error) {
	m := &StopTempRateResponse{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *StopTempRateResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(stopTempRateResponseProps, raw)
	if len(raw) != stopTempRateResponseProps.Size {
		return messages.InvalidLength("StopTempRateResponse", len(raw), stopTempRateResponseProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *StopTempRateResponse) decode(raw []byte) {
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	m.TempRateID = helpers.ReadShort(raw, 1)
}
