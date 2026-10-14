package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// SetSleepScheduleRequest sets Control-IQ sleep schedule Slot to the six
// bytes of a currentstatus.SleepSchedule. Flag is 3 in captures of the
// Tandem app; its meaning is unknown.
//
// pumpX2's constructors other than the one from a payload build only the
// payload, leaving the fields zero.
type SetSleepScheduleRequest struct {
	messages.Base
	Flag        int32  `pumpx2:"flag"`
	RawSchedule []byte `pumpx2:"rawSchedule"`
	Slot        int32  `pumpx2:"slot"`
}

var setSleepScheduleRequestProps = messages.Props{OpCode: -50, Size: 8, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	MinAPI: "MOBI_API_V3_5", SupportedDevices: "MOBI_ONLY", Response: "SetSleepScheduleResponse"}

func init() {
	messages.Register("SetSleepScheduleRequest", &SetSleepScheduleRequest{}, setSleepScheduleRequestProps,
		messages.Constructor{Params: "byte[] raw", Func: NewSetSleepScheduleRequestFromCargo},
		messages.Constructor{Params: "int slot, ControlIQSleepScheduleResponse.SleepSchedule schedule, int flag", Func: NewSetSleepScheduleRequest},
		messages.Constructor{Params: "int slot, byte[] rawSchedule, int flag", Func: NewSetSleepScheduleRequestFromRawSchedule},
	)
}

// NewSetSleepScheduleRequestFromCargo parses a request's payload.
func NewSetSleepScheduleRequestFromCargo(raw []byte) (*SetSleepScheduleRequest, error) {
	m := &SetSleepScheduleRequest{}
	return m, m.Parse(raw)
}

// NewSetSleepScheduleRequest builds the request's payload.
func NewSetSleepScheduleRequest(slot int32, schedule currentstatus.SleepSchedule, flag int32) *SetSleepScheduleRequest {
	return NewSetSleepScheduleRequestFromRawSchedule(slot, schedule.Build(), flag)
}

// NewSetSleepScheduleRequestFromRawSchedule builds the request's payload
// from a schedule's six bytes.
func NewSetSleepScheduleRequestFromRawSchedule(slot int32, rawSchedule []byte, flag int32) *SetSleepScheduleRequest {
	m := &SetSleepScheduleRequest{}
	m.SetCargo(helpers.Combine([]byte{byte(slot)}, rawSchedule, []byte{byte(flag)}))
	return m
}

// Parse implements messages.Message.
func (m *SetSleepScheduleRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setSleepScheduleRequestProps, raw)
	if len(raw) != setSleepScheduleRequestProps.Size {
		return messages.InvalidLength("SetSleepScheduleRequest", len(raw), setSleepScheduleRequestProps.Size)
	}
	m.SetCargo(raw)
	m.Slot = int32(int8(raw[0]))
	m.RawSchedule = helpers.DropLastN(helpers.DropFirstN(raw, 1), 1)
	m.Flag = int32(int8(raw[7]))
	return nil
}

// SleepSchedule is RawSchedule decoded. It panics on a request that was
// built rather than parsed, whose RawSchedule is nil.
func (m *SetSleepScheduleRequest) SleepSchedule() currentstatus.SleepSchedule {
	return currentstatus.ReadSleepSchedule(m.RawSchedule, 0)
}
