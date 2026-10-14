package control

import (
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// ChangeTimeDateRequest sets the pump's clock. TandemEpochTime is seconds
// since the pump's epoch, 2008-01-01T00:00:00Z, and Instant is that time.
type ChangeTimeDateRequest struct {
	messages.Base
	TandemEpochTime int64     `pumpx2:"tandemEpochTime"`
	Instant         time.Time `pumpx2:"instant"`
}

var changeTimeDateRequestProps = messages.Props{OpCode: -42, Size: 4, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "ChangeTimeDateResponse"}

func init() {
	messages.Register("ChangeTimeDateRequest", &ChangeTimeDateRequest{}, changeTimeDateRequestProps,
		messages.Constructor{Params: "byte[] raw", Func: NewChangeTimeDateRequestFromCargo},
		messages.Constructor{Params: "Instant timestamp", Func: NewChangeTimeDateRequestAt},
		messages.Constructor{Params: "long tandemEpochTime", Func: NewChangeTimeDateRequest},
	)
}

// NewChangeTimeDateRequest builds the request from a pump timestamp, of
// which it keeps the low 32 bits.
func NewChangeTimeDateRequest(tandemEpochTime int64) *ChangeTimeDateRequest {
	m := &ChangeTimeDateRequest{}
	m.decode(helpers.ToUint32(tandemEpochTime))
	return m
}

// NewChangeTimeDateRequestAt builds the request for a time, to the second.
func NewChangeTimeDateRequestAt(timestamp time.Time) *ChangeTimeDateRequest {
	return NewChangeTimeDateRequest(helpers.FromInstantToJan12008EpochSeconds(timestamp))
}

// NewChangeTimeDateRequestFromCargo parses a request's payload.
func NewChangeTimeDateRequestFromCargo(raw []byte) (*ChangeTimeDateRequest, error) {
	m := &ChangeTimeDateRequest{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *ChangeTimeDateRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(changeTimeDateRequestProps, raw)
	if len(raw) != changeTimeDateRequestProps.Size {
		return messages.InvalidLength("ChangeTimeDateRequest", len(raw), changeTimeDateRequestProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *ChangeTimeDateRequest) decode(raw []byte) {
	m.SetCargo(raw)
	m.TandemEpochTime = helpers.ReadUint32(raw, 0)
	// Any uint32 of seconds is within an Instant's range.
	m.Instant, _ = helpers.FromJan12008EpochSecondsToDate(m.TandemEpochTime)
}
