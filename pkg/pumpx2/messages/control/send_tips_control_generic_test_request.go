package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// SendTipsControlGenericTestRequest sends the pump a generic TIPS control
// test command: six unsigned 32-bit parameters, which pumpX2 does not
// interpret.
type SendTipsControlGenericTestRequest struct {
	messages.Base
	Param1 int64 `pumpx2:"param1"`
	Param2 int64 `pumpx2:"param2"`
	Param3 int64 `pumpx2:"param3"`
	Param4 int64 `pumpx2:"param4"`
	Param5 int64 `pumpx2:"param5"`
	Param6 int64 `pumpx2:"param6"`
}

var sendTipsControlGenericTestRequestProps = messages.Props{OpCode: 118, Size: 24, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "SendTipsControlGenericTestResponse"}

func init() {
	messages.Register("SendTipsControlGenericTestRequest", &SendTipsControlGenericTestRequest{}, sendTipsControlGenericTestRequestProps,
		messages.Constructor{
			Params: "long param1, long param2, long param3, long param4, long param5, long param6",
			Func:   NewSendTipsControlGenericTestRequest,
		},
	)
}

// NewSendTipsControlGenericTestRequest builds the request.
func NewSendTipsControlGenericTestRequest(param1, param2, param3, param4, param5, param6 int64) *SendTipsControlGenericTestRequest {
	m := &SendTipsControlGenericTestRequest{Param1: param1, Param2: param2, Param3: param3, Param4: param4, Param5: param5, Param6: param6}
	m.SetCargo(helpers.Combine(
		helpers.ToUint32(param1), helpers.ToUint32(param2), helpers.ToUint32(param3),
		helpers.ToUint32(param4), helpers.ToUint32(param5), helpers.ToUint32(param6),
	))
	return m
}

// Parse implements messages.Message.
func (m *SendTipsControlGenericTestRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(sendTipsControlGenericTestRequestProps, raw)
	if len(raw) != sendTipsControlGenericTestRequestProps.Size {
		return messages.InvalidLength("SendTipsControlGenericTestRequest", len(raw), sendTipsControlGenericTestRequestProps.Size)
	}
	m.SetCargo(raw)
	m.Param1 = helpers.ReadUint32(raw, 0)
	m.Param2 = helpers.ReadUint32(raw, 4)
	m.Param3 = helpers.ReadUint32(raw, 8)
	m.Param4 = helpers.ReadUint32(raw, 12)
	m.Param5 = helpers.ReadUint32(raw, 16)
	m.Param6 = helpers.ReadUint32(raw, 20)
	return nil
}
