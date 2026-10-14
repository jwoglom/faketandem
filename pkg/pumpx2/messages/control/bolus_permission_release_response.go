package control

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
)

// BolusPermissionReleaseResponse is the pump's answer to a
// BolusPermissionReleaseRequest: Status 0 (SUCCESS) if it released the
// permission, 1 (FAILURE) if the bolus ID was not one it granted.
type BolusPermissionReleaseResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var bolusPermissionReleaseResponseProps = messages.Props{OpCode: -15, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	MinAPI: "API_V2_5", Request: "BolusPermissionReleaseRequest"}

func init() {
	messages.Register("BolusPermissionReleaseResponse", &BolusPermissionReleaseResponse{}, bolusPermissionReleaseResponseProps,
		messages.Constructor{Params: "int status", Func: NewBolusPermissionReleaseResponse},
	)
}

// NewBolusPermissionReleaseResponse builds the response.
func NewBolusPermissionReleaseResponse(status int32) *BolusPermissionReleaseResponse {
	m := &BolusPermissionReleaseResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *BolusPermissionReleaseResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(bolusPermissionReleaseResponseProps, raw)
	if len(raw) != bolusPermissionReleaseResponseProps.Size {
		return messages.InvalidLength("BolusPermissionReleaseResponse", len(raw), bolusPermissionReleaseResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}

// ReleaseStatus is Status as a ReleaseStatus, or nil for a status that is
// none.
func (m *BolusPermissionReleaseResponse) ReleaseStatus() *ReleaseStatus {
	return ReleaseStatusFromID(m.Status)
}

// ReleaseStatus is whether the pump released a bolus permission.
type ReleaseStatus int32

// The release statuses, by their wire IDs.
const (
	ReleaseStatusSuccess ReleaseStatus = 0
	ReleaseStatusFailure ReleaseStatus = 1
)

var releaseStatusNames = []string{"SUCCESS", "FAILURE"}

// ReleaseStatusFromID is the status with wire ID id, or nil for an ID that
// is none.
func ReleaseStatusFromID(id int32) *ReleaseStatus {
	if id < 0 || int(id) >= len(releaseStatusNames) {
		return nil
	}
	s := ReleaseStatus(id)
	return &s
}

func (s ReleaseStatus) String() string {
	if s >= 0 && int(s) < len(releaseStatusNames) {
		return releaseStatusNames[s]
	}
	return fmt.Sprintf("ReleaseStatus(%d)", int32(s))
}
