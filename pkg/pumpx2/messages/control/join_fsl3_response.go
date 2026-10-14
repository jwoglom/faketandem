package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// JoinFSL3Response is the pump's answer to a JoinFSL3Request: Status 0 means
// success, and StatusType qualifies it.
type JoinFSL3Response struct {
	messages.Base
	Status     int32 `pumpx2:"status"`
	StatusType int32 `pumpx2:"statusType"`
}

var joinFSL3ResponseProps = messages.Props{OpCode: 121, Size: 2, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "JoinFSL3Request"}

func init() {
	messages.Register("JoinFSL3Response", &JoinFSL3Response{}, joinFSL3ResponseProps,
		messages.Constructor{Params: "int status, int statusType", Func: NewJoinFSL3Response},
	)
}

// NewJoinFSL3Response builds the response.
func NewJoinFSL3Response(status, statusType int32) *JoinFSL3Response {
	m := &JoinFSL3Response{Status: status, StatusType: statusType}
	m.SetCargo([]byte{byte(status), byte(statusType)})
	return m
}

// Parse implements messages.Message.
func (m *JoinFSL3Response) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(joinFSL3ResponseProps, raw)
	if len(raw) != joinFSL3ResponseProps.Size {
		return messages.InvalidLength("JoinFSL3Response", len(raw), joinFSL3ResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	m.StatusType = int32(raw[1])
	return nil
}
