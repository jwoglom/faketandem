package currentstatus

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// PumpGlobalsResponse is the pump's quick bolus settings and how it
// announces each kind of event. The *Annun fields are AnnunciationEnum IDs.
//
// pumpX2's getQuickBolusIncrement and getQuickBolusMode read this response
// into SetQuickBolusSettingsRequest's types, so here they are functions of
// the control package, which may import this one.
type PumpGlobalsResponse struct {
	messages.Base
	QuickBolusEnabledRaw int32 `pumpx2:"quickBolusEnabledRaw"`
	// QuickBolusIncrementUnits is in milliunits, QuickBolusIncrementCarbs in
	// grams times 1000.
	QuickBolusIncrementUnits int32 `pumpx2:"quickBolusIncrementUnits"`
	QuickBolusIncrementCarbs int32 `pumpx2:"quickBolusIncrementCarbs"`
	QuickBolusEntryType      int32 `pumpx2:"quickBolusEntryType"`
	QuickBolusStatus         int32 `pumpx2:"quickBolusStatus"`
	ButtonAnnun              int32 `pumpx2:"buttonAnnun"`
	QuickBolusAnnun          int32 `pumpx2:"quickBolusAnnun"`
	BolusAnnun               int32 `pumpx2:"bolusAnnun"`
	ReminderAnnun            int32 `pumpx2:"reminderAnnun"`
	AlertAnnun               int32 `pumpx2:"alertAnnun"`
	AlarmAnnun               int32 `pumpx2:"alarmAnnun"`
	FillTubingAnnun          int32 `pumpx2:"fillTubingAnnun"`
}

var pumpGlobalsResponseProps = messages.Props{OpCode: 87, Size: 14, Type: messages.Response, Request: "PumpGlobalsRequest"}

func init() {
	messages.Register("PumpGlobalsResponse", &PumpGlobalsResponse{}, pumpGlobalsResponseProps,
		messages.Constructor{
			Params: "int quickBolusEnabledRaw, int quickBolusIncrementUnits, int quickBolusIncrementCarbs, int quickBolusEntryType, int quickBolusStatus, " +
				"int buttonAnnun, int quickBolusAnnun, int bolusAnnun, int reminderAnnun, int alertAnnun, int alarmAnnun, int fillTubingAnnun",
			Func: NewPumpGlobalsResponse,
		},
	)
}

// NewPumpGlobalsResponse builds the response.
func NewPumpGlobalsResponse(quickBolusEnabledRaw, quickBolusIncrementUnits, quickBolusIncrementCarbs, quickBolusEntryType, quickBolusStatus,
	buttonAnnun, quickBolusAnnun, bolusAnnun, reminderAnnun, alertAnnun, alarmAnnun, fillTubingAnnun int32) *PumpGlobalsResponse {
	m := &PumpGlobalsResponse{
		QuickBolusEnabledRaw:     quickBolusEnabledRaw,
		QuickBolusIncrementUnits: quickBolusIncrementUnits,
		QuickBolusIncrementCarbs: quickBolusIncrementCarbs,
		QuickBolusEntryType:      quickBolusEntryType,
		QuickBolusStatus:         quickBolusStatus,
		ButtonAnnun:              buttonAnnun,
		QuickBolusAnnun:          quickBolusAnnun,
		BolusAnnun:               bolusAnnun,
		ReminderAnnun:            reminderAnnun,
		AlertAnnun:               alertAnnun,
		AlarmAnnun:               alarmAnnun,
		FillTubingAnnun:          fillTubingAnnun,
	}
	m.SetCargo(helpers.Combine(
		[]byte{byte(quickBolusEnabledRaw)},
		helpers.FirstTwoBytesLittleEndian(quickBolusIncrementUnits),
		helpers.FirstTwoBytesLittleEndian(quickBolusIncrementCarbs),
		[]byte{byte(quickBolusEntryType), byte(quickBolusStatus), byte(buttonAnnun), byte(quickBolusAnnun), byte(bolusAnnun),
			byte(reminderAnnun), byte(alertAnnun), byte(alarmAnnun), byte(fillTubingAnnun)},
	))
	return m
}

// Parse implements messages.Message.
func (m *PumpGlobalsResponse) Parse(raw []byte) error {
	if len(raw) != pumpGlobalsResponseProps.Size {
		return messages.InvalidLength("PumpGlobalsResponse", len(raw), pumpGlobalsResponseProps.Size)
	}
	m.SetCargo(raw)
	m.QuickBolusEnabledRaw = int32(int8(raw[0]))
	m.QuickBolusIncrementUnits = helpers.ReadShort(raw, 1)
	m.QuickBolusIncrementCarbs = helpers.ReadShort(raw, 3)
	m.QuickBolusEntryType = int32(int8(raw[5]))
	m.QuickBolusStatus = int32(int8(raw[6]))
	m.ButtonAnnun = int32(int8(raw[7]))
	m.QuickBolusAnnun = int32(int8(raw[8]))
	m.BolusAnnun = int32(int8(raw[9]))
	m.ReminderAnnun = int32(int8(raw[10]))
	m.AlertAnnun = int32(int8(raw[11]))
	m.AlarmAnnun = int32(int8(raw[12]))
	m.FillTubingAnnun = int32(int8(raw[13]))
	return nil
}

// QuickBolusEnabled reports whether the quick bolus button is on.
func (m *PumpGlobalsResponse) QuickBolusEnabled() bool { return m.QuickBolusEnabledRaw == 1 }

// AnnunciationEnum is how the pump announces an event.
type AnnunciationEnum int32

// The annunciations, by their wire IDs.
const (
	AnnunciationAudioHigh   AnnunciationEnum = 0
	AnnunciationAudioMedium AnnunciationEnum = 1
	AnnunciationAudioLow    AnnunciationEnum = 2
	AnnunciationVibrate     AnnunciationEnum = 3
)

var annunciationNames = []string{"AUDIO_HIGH", "AUDIO_MEDIUM", "AUDIO_LOW", "VIBRATE"}

// AnnunciationEnumFromID is the annunciation with wire ID id, or nil for an
// ID that is none.
func AnnunciationEnumFromID(id int32) *AnnunciationEnum {
	if id < 0 || int(id) >= len(annunciationNames) {
		return nil
	}
	a := AnnunciationEnum(id)
	return &a
}

// ID is the annunciation's wire ID.
func (a AnnunciationEnum) ID() int32 { return int32(a) }

func (a AnnunciationEnum) String() string {
	if a >= 0 && int(a) < len(annunciationNames) {
		return annunciationNames[a]
	}
	return fmt.Sprintf("AnnunciationEnum(%d)", int32(a))
}
