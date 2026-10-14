package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// APIVersionResponse is the pump's API version: 2.0 for t:slim X2 software
// 7.3, 2.1 for 7.4 and later, 3.x for the Mobi.
type APIVersionResponse struct {
	messages.Base
	MajorVersion int32 `pumpx2:"majorVersion"`
	MinorVersion int32 `pumpx2:"minorVersion"`
}

var apiVersionResponseProps = messages.Props{OpCode: 33, Size: 4, Type: messages.Response, Request: "ApiVersionRequest"}

func init() {
	messages.Register("ApiVersionResponse", &APIVersionResponse{}, apiVersionResponseProps,
		messages.Constructor{Params: "int majorVersion, int minorVersion", Func: NewAPIVersionResponse},
	)
}

// NewAPIVersionResponse is the response of a pump at API version
// majorVersion.minorVersion.
func NewAPIVersionResponse(majorVersion, minorVersion int32) *APIVersionResponse {
	m := &APIVersionResponse{MajorVersion: majorVersion, MinorVersion: minorVersion}
	m.SetCargo(helpers.Combine(helpers.FirstTwoBytesLittleEndian(majorVersion), helpers.FirstTwoBytesLittleEndian(minorVersion)))
	return m
}

// Parse implements messages.Message.
func (m *APIVersionResponse) Parse(raw []byte) error {
	if len(raw) != apiVersionResponseProps.Size {
		return messages.InvalidLength("ApiVersionResponse", len(raw), apiVersionResponseProps.Size)
	}
	m.MajorVersion = helpers.ReadShort(raw, 0)
	m.MinorVersion = helpers.ReadShort(raw, 2)
	m.SetCargo(raw)
	return nil
}
