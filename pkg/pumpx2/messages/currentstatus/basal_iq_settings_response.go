package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// BasalIQSettingsResponse is the Basal-IQ settings: HypoMinimization, and
// whether the pump alerts when Basal-IQ suspends and resumes insulin.
// pumpX2 documents none of them.
type BasalIQSettingsResponse struct {
	messages.Base
	HypoMinimization int32 `pumpx2:"hypoMinimization"`
	SuspendAlert     int32 `pumpx2:"suspendAlert"`
	ResumeAlert      int32 `pumpx2:"resumeAlert"`
}

var basalIQSettingsResponseProps = messages.Props{OpCode: 99, Size: 3, Type: messages.Response, Request: "BasalIQSettingsRequest"}

func init() {
	messages.Register("BasalIQSettingsResponse", &BasalIQSettingsResponse{}, basalIQSettingsResponseProps,
		messages.Constructor{Params: "int hypoMinimization, int suspendAlert, int resumeAlert", Func: NewBasalIQSettingsResponse},
	)
}

// NewBasalIQSettingsResponse builds the response.
func NewBasalIQSettingsResponse(hypoMinimization, suspendAlert, resumeAlert int32) *BasalIQSettingsResponse {
	m := &BasalIQSettingsResponse{HypoMinimization: hypoMinimization, SuspendAlert: suspendAlert, ResumeAlert: resumeAlert}
	m.SetCargo([]byte{byte(hypoMinimization), byte(suspendAlert), byte(resumeAlert)})
	return m
}

// Parse implements messages.Message.
func (m *BasalIQSettingsResponse) Parse(raw []byte) error {
	if len(raw) != basalIQSettingsResponseProps.Size {
		return messages.InvalidLength("BasalIQSettingsResponse", len(raw), basalIQSettingsResponseProps.Size)
	}
	m.SetCargo(raw)
	m.HypoMinimization = int32(int8(raw[0]))
	m.SuspendAlert = int32(int8(raw[1]))
	m.ResumeAlert = int32(int8(raw[2]))
	return nil
}
