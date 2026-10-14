package currentstatus

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// ControlIQIOBResponse is the pump's insulin on board, in milliunits, by the
// two models it keeps. Mudaliar IOB counts bolus insulin alone and is the one
// the pump shows with Control-IQ off; Swan 6hr IOB also counts basal
// delivered above and below the profile and is the one shown with Control-IQ
// on. IOBTypeID is the IOBType shown. faketandem takes MudaliarTotalIOB as
// the full size of the boluses Mudaliar counts, and TimeRemainingSeconds as
// how long until the last of them stops counting.
type ControlIQIOBResponse struct {
	messages.Base
	MudaliarIOB          int64 `pumpx2:"mudaliarIOB"`
	TimeRemainingSeconds int64 `pumpx2:"timeRemainingSeconds"`
	MudaliarTotalIOB     int64 `pumpx2:"mudaliarTotalIOB"`
	Swan6hrIOB           int64 `pumpx2:"swan6hrIOB"`
	IOBTypeID            int32 `pumpx2:"iobType"`
}

var controlIQIOBResponseProps = messages.Props{OpCode: 109, Size: 17, Type: messages.Response, Request: "ControlIQIOBRequest"}

func init() {
	messages.Register("ControlIQIOBResponse", &ControlIQIOBResponse{}, controlIQIOBResponseProps,
		messages.Constructor{
			Params: "long mudaliarIOB, long timeRemainingSeconds, long mudaliarTotalIOB, long swan6hrIOB, int iobType",
			Func:   NewControlIQIOBResponse,
		},
	)
}

// NewControlIQIOBResponse builds the response; iobType is an IOBType ID.
func NewControlIQIOBResponse(mudaliarIOB, timeRemainingSeconds, mudaliarTotalIOB, swan6hrIOB int64, iobType int32) *ControlIQIOBResponse {
	m := &ControlIQIOBResponse{
		MudaliarIOB:          mudaliarIOB,
		TimeRemainingSeconds: timeRemainingSeconds,
		MudaliarTotalIOB:     mudaliarTotalIOB,
		Swan6hrIOB:           swan6hrIOB,
		IOBTypeID:            iobType,
	}
	m.SetCargo(helpers.Combine(
		helpers.ToUint32(mudaliarIOB),
		helpers.ToUint32(timeRemainingSeconds),
		helpers.ToUint32(mudaliarTotalIOB),
		helpers.ToUint32(swan6hrIOB),
		[]byte{byte(iobType)},
	))
	return m
}

// Parse implements messages.Message.
func (m *ControlIQIOBResponse) Parse(raw []byte) error {
	if len(raw) != controlIQIOBResponseProps.Size {
		return messages.InvalidLength("ControlIQIOBResponse", len(raw), controlIQIOBResponseProps.Size)
	}
	m.MudaliarIOB = helpers.ReadUint32(raw, 0)
	m.TimeRemainingSeconds = helpers.ReadUint32(raw, 4)
	m.MudaliarTotalIOB = helpers.ReadUint32(raw, 8)
	m.Swan6hrIOB = helpers.ReadUint32(raw, 12)
	m.IOBTypeID = int32(int8(raw[16]))
	m.SetCargo(raw)
	return nil
}

// IOBType is the model of the IOB the pump shows, or nil for an ID that is
// none.
func (m *ControlIQIOBResponse) IOBType() *IOBType { return IOBTypeFromID(m.IOBTypeID) }

// PumpDisplayedIOB is the IOB the pump shows, in milliunits: the IOBType's.
// For an unknown type it is the larger of the two, as pumpX2 documents;
// pumpX2 itself throws a NullPointerException there instead.
func (m *ControlIQIOBResponse) PumpDisplayedIOB() int64 {
	if t := m.IOBType(); t != nil {
		switch *t {
		case IOBTypeMudaliar:
			return m.MudaliarIOB
		case IOBTypeSwan6hr:
			return m.Swan6hrIOB
		}
	}
	if m.MudaliarIOB > m.Swan6hrIOB {
		return m.MudaliarIOB
	}
	return m.Swan6hrIOB
}

// IOBType is a model of insulin on board the pump keeps.
type IOBType int32

// The IOB models.
const (
	IOBTypeMudaliar IOBType = iota
	IOBTypeSwan6hr
)

var iobTypeNames = []string{"MUDALIAR", "SWAN_6HR"}

// IOBTypeFromID is the model with wire ID id, or nil for an ID that is none.
func IOBTypeFromID(id int32) *IOBType {
	if id < 0 || int(id) >= len(iobTypeNames) {
		return nil
	}
	t := IOBType(id)
	return &t
}

// ID is the model's wire ID.
func (t IOBType) ID() int32 { return int32(t) }

func (t IOBType) String() string {
	if t >= 0 && int(t) < len(iobTypeNames) {
		return iobTypeNames[t]
	}
	return fmt.Sprintf("IOBType(%d)", int32(t))
}
