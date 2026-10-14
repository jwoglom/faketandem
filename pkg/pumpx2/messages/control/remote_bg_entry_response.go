package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// RemoteBgEntryResponse is the pump's answer to a RemoteBgEntryRequest:
// Status 0 means success.
type RemoteBgEntryResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var remoteBgEntryResponseProps = messages.Props{OpCode: -73, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	MinAPI: "API_V2_5", Request: "RemoteBgEntryRequest"}

func init() {
	messages.Register("RemoteBgEntryResponse", &RemoteBgEntryResponse{}, remoteBgEntryResponseProps,
		messages.Constructor{Params: "int status", Func: NewRemoteBgEntryResponse},
	)
}

// NewRemoteBgEntryResponse builds the response.
func NewRemoteBgEntryResponse(status int32) *RemoteBgEntryResponse {
	m := &RemoteBgEntryResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *RemoteBgEntryResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(remoteBgEntryResponseProps, raw)
	if len(raw) != remoteBgEntryResponseProps.Size {
		return messages.InvalidLength("RemoteBgEntryResponse", len(raw), remoteBgEntryResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}

// IsSuccess reports whether the pump took the BG.
func (m *RemoteBgEntryResponse) IsSuccess() bool { return m.Status == 0 }
