package control

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
)

// SetPumpSoundsRequest sets how the pump announces each kind of event. The
// *AnnunRaw fields are currentstatus.AnnunciationEnum IDs, which the
// *Annun fields decode (nil for an ID that is none); CgmAlertAnnunA and B
// together are a CgmAlertAnnunciationEnum. ChangeBitmaskRaw flags the
// kinds the pump applies. The payload's first byte, 0 in captures, is not
// decoded.
type SetPumpSoundsRequest struct {
	messages.Base
	AlarmAnnun         *currentstatus.AnnunciationEnum `pumpx2:"alarmAnnun"`
	AlarmAnnunRaw      int32                           `pumpx2:"alarmAnnunRaw"`
	AlertAnnun         *currentstatus.AnnunciationEnum `pumpx2:"alertAnnun"`
	AlertAnnunRaw      int32                           `pumpx2:"alertAnnunRaw"`
	CgmAlertAnnun      *CgmAlertAnnunciationEnum       `pumpx2:"cgmAlertAnnun"`
	CgmAlertAnnunA     int32                           `pumpx2:"cgmAlertAnnunA"`
	CgmAlertAnnunB     int32                           `pumpx2:"cgmAlertAnnunB"`
	ChangeBitmask      []ChangeBitmask                 `pumpx2:"changeBitmask"`
	ChangeBitmaskRaw   int32                           `pumpx2:"changeBitmaskRaw"`
	GeneralAnnun       *currentstatus.AnnunciationEnum `pumpx2:"generalAnnun"`
	GeneralAnnunRaw    int32                           `pumpx2:"generalAnnunRaw"`
	QuickBolusAnnun    *currentstatus.AnnunciationEnum `pumpx2:"quickBolusAnnun"`
	QuickBolusAnnunRaw int32                           `pumpx2:"quickBolusAnnunRaw"`
	ReminderAnnun      *currentstatus.AnnunciationEnum `pumpx2:"reminderAnnun"`
	ReminderAnnunRaw   int32                           `pumpx2:"reminderAnnunRaw"`
}

var setPumpSoundsRequestProps = messages.Props{OpCode: -28, Size: 9, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "SetPumpSoundsResponse"}

func init() {
	messages.Register("SetPumpSoundsRequest", &SetPumpSoundsRequest{}, setPumpSoundsRequestProps,
		messages.Constructor{Params: "byte[] raw", Func: NewSetPumpSoundsRequestFromCargo},
		messages.Constructor{
			Params: "int firstByteUnknown, int quickBolusAnnunRaw, int generalAnnunRaw, int reminderAnnunRaw, int alertAnnunRaw, int alarmAnnunRaw, " +
				"int cgmAlertAnnunA, int cgmAlertAnnunB, int changeBitmaskRaw",
			Func: NewSetPumpSoundsRequestWithFirstByte,
		},
		messages.Constructor{
			Params: "PumpGlobalsResponse.AnnunciationEnum quickBolusAnnun, PumpGlobalsResponse.AnnunciationEnum generalAnnun, " +
				"PumpGlobalsResponse.AnnunciationEnum reminderAnnun, PumpGlobalsResponse.AnnunciationEnum alertAnnun, " +
				"PumpGlobalsResponse.AnnunciationEnum alarmAnnun, SetPumpSoundsRequest.CgmAlertAnnunciationEnum cgmAlertAnnun, " +
				"SetPumpSoundsRequest.ChangeBitmask changeBitmask",
			Func: NewSetPumpSoundsRequest,
		},
		messages.Constructor{
			Params: "int quickBolusAnnunRaw, int generalAnnunRaw, int reminderAnnunRaw, int alertAnnunRaw, int alarmAnnunRaw, " +
				"int cgmAlertAnnunA, int cgmAlertAnnunB, int changeBitmaskRaw",
			Func: NewSetPumpSoundsRequestFromRaw,
		},
	)
}

// NewSetPumpSoundsRequestFromCargo parses a request's payload.
func NewSetPumpSoundsRequestFromCargo(raw []byte) (*SetPumpSoundsRequest, error) {
	m := &SetPumpSoundsRequest{}
	return m, m.Parse(raw)
}

// NewSetPumpSoundsRequest sets every kind of event's annunciation, applying
// the one kind changeBitmask flags.
func NewSetPumpSoundsRequest(quickBolusAnnun, generalAnnun, reminderAnnun, alertAnnun, alarmAnnun currentstatus.AnnunciationEnum,
	cgmAlertAnnun CgmAlertAnnunciationEnum, changeBitmask ChangeBitmask) *SetPumpSoundsRequest {
	return NewSetPumpSoundsRequestWithFirstByte(0, quickBolusAnnun.ID(), generalAnnun.ID(), reminderAnnun.ID(), alertAnnun.ID(), alarmAnnun.ID(),
		cgmAlertAnnun.IDA(), cgmAlertAnnun.IDB(), changeBitmask.ID())
}

// NewSetPumpSoundsRequestFromRaw builds the request from wire values.
func NewSetPumpSoundsRequestFromRaw(quickBolusAnnunRaw, generalAnnunRaw, reminderAnnunRaw, alertAnnunRaw, alarmAnnunRaw,
	cgmAlertAnnunA, cgmAlertAnnunB, changeBitmaskRaw int32) *SetPumpSoundsRequest {
	return NewSetPumpSoundsRequestWithFirstByte(0, quickBolusAnnunRaw, generalAnnunRaw, reminderAnnunRaw, alertAnnunRaw, alarmAnnunRaw,
		cgmAlertAnnunA, cgmAlertAnnunB, changeBitmaskRaw)
}

// NewSetPumpSoundsRequestWithFirstByte builds the request from wire values,
// the undecoded first byte included.
func NewSetPumpSoundsRequestWithFirstByte(firstByteUnknown, quickBolusAnnunRaw, generalAnnunRaw, reminderAnnunRaw, alertAnnunRaw, alarmAnnunRaw,
	cgmAlertAnnunA, cgmAlertAnnunB, changeBitmaskRaw int32) *SetPumpSoundsRequest {
	m := &SetPumpSoundsRequest{}
	m.decode([]byte{byte(firstByteUnknown), byte(quickBolusAnnunRaw), byte(generalAnnunRaw), byte(reminderAnnunRaw), byte(alertAnnunRaw),
		byte(alarmAnnunRaw), byte(cgmAlertAnnunA), byte(cgmAlertAnnunB), byte(changeBitmaskRaw)})
	return m
}

// Parse implements messages.Message.
func (m *SetPumpSoundsRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setPumpSoundsRequestProps, raw)
	if len(raw) != setPumpSoundsRequestProps.Size {
		return messages.InvalidLength("SetPumpSoundsRequest", len(raw), setPumpSoundsRequestProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *SetPumpSoundsRequest) decode(raw []byte) {
	m.SetCargo(raw)
	m.QuickBolusAnnunRaw = int32(int8(raw[1]))
	m.GeneralAnnunRaw = int32(int8(raw[2]))
	m.ReminderAnnunRaw = int32(int8(raw[3]))
	m.AlertAnnunRaw = int32(int8(raw[4]))
	m.AlarmAnnunRaw = int32(int8(raw[5]))
	m.CgmAlertAnnunA = int32(int8(raw[6]))
	m.CgmAlertAnnunB = int32(int8(raw[7]))
	m.ChangeBitmaskRaw = int32(int8(raw[8]))
	m.QuickBolusAnnun = currentstatus.AnnunciationEnumFromID(m.QuickBolusAnnunRaw)
	m.GeneralAnnun = currentstatus.AnnunciationEnumFromID(m.GeneralAnnunRaw)
	m.ReminderAnnun = currentstatus.AnnunciationEnumFromID(m.ReminderAnnunRaw)
	m.AlertAnnun = currentstatus.AnnunciationEnumFromID(m.AlertAnnunRaw)
	m.AlarmAnnun = currentstatus.AnnunciationEnumFromID(m.AlarmAnnunRaw)
	m.CgmAlertAnnun = CgmAlertAnnunciationEnumFromIDs(m.CgmAlertAnnunA, m.CgmAlertAnnunB)
	m.ChangeBitmask = ChangeBitmaskFromBitmask(m.ChangeBitmaskRaw)
}

// CgmAlertAnnunciationEnum is how the pump announces CGM alerts, two bytes
// on the wire.
type CgmAlertAnnunciationEnum int32

// The CGM alert annunciations.
const (
	CgmAlertAnnunciationVibrate CgmAlertAnnunciationEnum = iota
	CgmAlertAnnunciationBeep
	CgmAlertAnnunciationHypoRepeat
)

var cgmAlertAnnunciations = []struct {
	name     string
	idA, idB int32
}{
	{"VIBRATE", 0, 0},
	{"BEEP", 0, 2},
	{"HYPO_REPEAT", 0, 3},
}

// CgmAlertAnnunciationEnumFromIDs is the annunciation with those two wire
// bytes, or nil for a pair that is none.
func CgmAlertAnnunciationEnumFromIDs(idA, idB int32) *CgmAlertAnnunciationEnum {
	for i, a := range cgmAlertAnnunciations {
		if a.idA == idA && a.idB == idB {
			e := CgmAlertAnnunciationEnum(i)
			return &e
		}
	}
	return nil
}

// IDA is the annunciation's first wire byte.
func (e CgmAlertAnnunciationEnum) IDA() int32 { return cgmAlertAnnunciations[e].idA }

// IDB is the annunciation's second wire byte.
func (e CgmAlertAnnunciationEnum) IDB() int32 { return cgmAlertAnnunciations[e].idB }

func (e CgmAlertAnnunciationEnum) String() string {
	if e >= 0 && int(e) < len(cgmAlertAnnunciations) {
		return cgmAlertAnnunciations[e].name
	}
	return fmt.Sprintf("CgmAlertAnnunciationEnum(%d)", int32(e))
}

// ChangeBitmask is one kind of event a SetPumpSoundsRequest applies, as a
// bit of ChangeBitmaskRaw.
type ChangeBitmask int32

// The kinds of event, by their bits.
const (
	ChangeBitmaskQuickBolus ChangeBitmask = 2
	ChangeBitmaskGeneral    ChangeBitmask = 4
	ChangeBitmaskReminder   ChangeBitmask = 8
	ChangeBitmaskAlert      ChangeBitmask = 16
	ChangeBitmaskAlarm      ChangeBitmask = 32
	ChangeBitmaskCgmAlert   ChangeBitmask = 64
)

// ChangeBitmasks are the kinds of event in pumpX2's declaration order.
var ChangeBitmasks = []ChangeBitmask{
	ChangeBitmaskQuickBolus, ChangeBitmaskGeneral, ChangeBitmaskReminder, ChangeBitmaskAlert, ChangeBitmaskAlarm, ChangeBitmaskCgmAlert,
}

// ChangeBitmaskFromBitmask is the kinds whose bits mask sets, in
// declaration order; never nil.
func ChangeBitmaskFromBitmask(mask int32) []ChangeBitmask {
	out := []ChangeBitmask{}
	for _, b := range ChangeBitmasks {
		if mask&int32(b) != 0 {
			out = append(out, b)
		}
	}
	return out
}

// ChangeBitmaskToBitmask sums the kinds' bits.
func ChangeBitmaskToBitmask(items ...ChangeBitmask) int32 {
	var mask int32
	for _, b := range items {
		mask += int32(b)
	}
	return mask
}

// ID is the kind's bit.
func (b ChangeBitmask) ID() int32 { return int32(b) }

func (b ChangeBitmask) String() string {
	switch b {
	case ChangeBitmaskQuickBolus:
		return "QUICK_BOLUS"
	case ChangeBitmaskGeneral:
		return "GENERAL"
	case ChangeBitmaskReminder:
		return "REMINDER"
	case ChangeBitmaskAlert:
		return "ALERT"
	case ChangeBitmaskAlarm:
		return "ALARM"
	case ChangeBitmaskCgmAlert:
		return "CGM_ALERT"
	}
	return fmt.Sprintf("ChangeBitmask(%d)", int32(b))
}
