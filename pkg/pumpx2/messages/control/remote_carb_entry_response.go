package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// RemoteCarbEntryResponse is the pump's answer to a RemoteCarbEntryRequest:
// Status 0 means success.
type RemoteCarbEntryResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var remoteCarbEntryResponseProps = messages.Props{OpCode: -13, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	MinAPI: "API_V2_5", Request: "RemoteCarbEntryRequest"}

func init() {
	messages.Register("RemoteCarbEntryResponse", &RemoteCarbEntryResponse{}, remoteCarbEntryResponseProps,
		messages.Constructor{Params: "int status", Func: NewRemoteCarbEntryResponse},
	)
}

// NewRemoteCarbEntryResponse builds the response.
func NewRemoteCarbEntryResponse(status int32) *RemoteCarbEntryResponse {
	m := &RemoteCarbEntryResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *RemoteCarbEntryResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(remoteCarbEntryResponseProps, raw)
	if len(raw) != remoteCarbEntryResponseProps.Size {
		return messages.InvalidLength("RemoteCarbEntryResponse", len(raw), remoteCarbEntryResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
