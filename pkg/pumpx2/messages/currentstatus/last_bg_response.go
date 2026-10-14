package currentstatus

import (
	"fmt"
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// LastBGResponse is the last BG entered in the bolus calculator, not the
// last CGM reading: its time (pump epoch seconds), its value in mg/dL and
// where it came from.
type LastBGResponse struct {
	messages.Base
	BgTimestamp int64     `pumpx2:"bgTimestamp"`
	BgValue     int32     `pumpx2:"bgValue"`
	BgSourceID  int32     `pumpx2:"bgSourceId"`
	BgSource    *BgSource `pumpx2:"bgSource"`
}

var lastBGResponseProps = messages.Props{OpCode: 51, Size: 7, Type: messages.Response, Request: "LastBGRequest"}

func init() {
	messages.Register("LastBGResponse", &LastBGResponse{}, lastBGResponseProps,
		messages.Constructor{Params: "long bgTimestamp, int bgValue, LastBGResponse.BgSource bgSource", Func: NewLastBGResponseWithSource},
		messages.Constructor{Params: "long bgTimestamp, int bgValue, int bgSourceId", Func: NewLastBGResponse},
	)
}

// NewLastBGResponse builds the response, with the source by its wire ID.
func NewLastBGResponse(bgTimestamp int64, bgValue, bgSourceID int32) *LastBGResponse {
	m := &LastBGResponse{BgTimestamp: bgTimestamp, BgValue: bgValue, BgSourceID: bgSourceID, BgSource: BgSourceFromID(bgSourceID)}
	m.SetCargo(helpers.Combine(helpers.ToUint32(bgTimestamp), helpers.FirstTwoBytesLittleEndian(bgValue), []byte{byte(bgSourceID)}))
	return m
}

// NewLastBGResponseWithSource builds the response.
func NewLastBGResponseWithSource(bgTimestamp int64, bgValue int32, bgSource BgSource) *LastBGResponse {
	return NewLastBGResponse(bgTimestamp, bgValue, bgSource.ID())
}

// Parse implements messages.Message.
func (m *LastBGResponse) Parse(raw []byte) error {
	if len(raw) != lastBGResponseProps.Size {
		return messages.InvalidLength("LastBGResponse", len(raw), lastBGResponseProps.Size)
	}
	m.SetCargo(raw)
	m.BgTimestamp = helpers.ReadUint32(raw, 0)
	m.BgValue = helpers.ReadShort(raw, 4)
	m.BgSourceID = int32(int8(raw[6]))
	m.BgSource = BgSourceFromID(m.BgSourceID)
	return nil
}

// BgTimestampInstant is BgTimestamp as a time, failing past the range of
// Java's Instant.
func (m *LastBGResponse) BgTimestampInstant() (time.Time, error) {
	return helpers.FromJan12008EpochSecondsToDate(m.BgTimestamp)
}

// BgSource is where a bolus calculator BG came from. pumpX2 has not
// confirmed the IDs.
type BgSource int32

// The BG sources, by their wire IDs.
const (
	BgSourceManual BgSource = 0
	BgSourceCGM    BgSource = 1
)

var bgSourceNames = []string{"MANUAL", "CGM"}

// BgSourceFromID is the source with wire ID id, or nil for an ID that is
// none.
func BgSourceFromID(id int32) *BgSource {
	if id < 0 || int(id) >= len(bgSourceNames) {
		return nil
	}
	s := BgSource(id)
	return &s
}

// ID is the source's wire ID.
func (s BgSource) ID() int32 { return int32(s) }

func (s BgSource) String() string {
	if s >= 0 && int(s) < len(bgSourceNames) {
		return bgSourceNames[s]
	}
	return fmt.Sprintf("BgSource(%d)", int32(s))
}
