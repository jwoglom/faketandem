package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// BolusCalcDataSnapshotRequest asks for what the bolus calculator starts from: IOB,
// the current BG, the profile's settings and the bolus limits.
type BolusCalcDataSnapshotRequest struct {
	messages.Base
}

var bolusCalcDataSnapshotRequestProps = messages.Props{OpCode: 114, Type: messages.Request, MinAPI: "API_V2_5", Response: "BolusCalcDataSnapshotResponse"}

func init() {
	messages.Register("BolusCalcDataSnapshotRequest", &BolusCalcDataSnapshotRequest{}, bolusCalcDataSnapshotRequestProps,
		messages.Constructor{Func: NewBolusCalcDataSnapshotRequest},
	)
}

// NewBolusCalcDataSnapshotRequest is the request, which has no payload.
func NewBolusCalcDataSnapshotRequest() *BolusCalcDataSnapshotRequest {
	m := &BolusCalcDataSnapshotRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *BolusCalcDataSnapshotRequest) Parse(raw []byte) error {
	if len(raw) != bolusCalcDataSnapshotRequestProps.Size {
		return messages.InvalidLength("BolusCalcDataSnapshotRequest", len(raw), bolusCalcDataSnapshotRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
