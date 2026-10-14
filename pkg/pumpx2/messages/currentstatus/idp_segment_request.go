package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// IDPSegmentRequest asks for one time segment of an insulin delivery
// profile, by the profile's ID (not its slot) and the segment's index.
type IDPSegmentRequest struct {
	messages.Base
	IDPID        int32 `pumpx2:"idpId"`
	SegmentIndex int32 `pumpx2:"segmentIndex"`
}

var idpSegmentRequestProps = messages.Props{OpCode: 66, Size: 2, Type: messages.Request, Response: "IDPSegmentResponse"}

func init() {
	messages.Register("IDPSegmentRequest", &IDPSegmentRequest{}, idpSegmentRequestProps,
		messages.Constructor{Params: "int idpId, int segmentIndex", Func: NewIDPSegmentRequest},
	)
}

// NewIDPSegmentRequest builds the request.
func NewIDPSegmentRequest(idpID, segmentIndex int32) *IDPSegmentRequest {
	m := &IDPSegmentRequest{IDPID: idpID, SegmentIndex: segmentIndex}
	m.SetCargo([]byte{byte(idpID), byte(segmentIndex)})
	return m
}

// Parse implements messages.Message.
func (m *IDPSegmentRequest) Parse(raw []byte) error {
	if len(raw) != idpSegmentRequestProps.Size {
		return messages.InvalidLength("IDPSegmentRequest", len(raw), idpSegmentRequestProps.Size)
	}
	m.SetCargo(raw)
	m.IDPID = int32(int8(raw[0]))
	m.SegmentIndex = int32(int8(raw[1]))
	return nil
}
