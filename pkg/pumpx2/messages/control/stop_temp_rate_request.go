package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// StopTempRateRequest stops the running temp basal rate. The pump answers
// with the stopped temp rate's ID.
type StopTempRateRequest struct {
	messages.Base
}

var stopTempRateRequestProps = messages.Props{OpCode: -90, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, MinAPI: "MOBI_API_V3_5", SupportedDevices: "MOBI_ONLY", Response: "StopTempRateResponse"}

func init() {
	messages.Register("StopTempRateRequest", &StopTempRateRequest{}, stopTempRateRequestProps,
		messages.Constructor{Func: NewStopTempRateRequest},
	)
}

// NewStopTempRateRequest is the request, with no payload.
func NewStopTempRateRequest() *StopTempRateRequest {
	m := &StopTempRateRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *StopTempRateRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(stopTempRateRequestProps, raw)
	if len(raw) != stopTempRateRequestProps.Size {
		return messages.InvalidLength("StopTempRateRequest", len(raw), stopTempRateRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
