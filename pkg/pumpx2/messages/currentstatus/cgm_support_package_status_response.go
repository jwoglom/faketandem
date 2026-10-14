package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CgmSupportPackageStatusResponse says whether the request succeeded (Status
// 0 is an ACK, 1 a NACK) and whether the CGM support package is valid.
type CgmSupportPackageStatusResponse struct {
	messages.Base
	Status   int32 `pumpx2:"status"`
	Validity bool  `pumpx2:"validity"`
}

var cgmSupportPackageStatusResponseProps = messages.Props{OpCode: -55, Size: 2, Type: messages.Response, MinAPI: "API_FUTURE", Request: "CgmSupportPackageStatusRequest"}

func init() {
	messages.Register("CgmSupportPackageStatusResponse", &CgmSupportPackageStatusResponse{}, cgmSupportPackageStatusResponseProps,
		messages.Constructor{Params: "int status, boolean validity", Func: NewCgmSupportPackageStatusResponse},
	)
}

// NewCgmSupportPackageStatusResponse builds the response.
func NewCgmSupportPackageStatusResponse(status int32, validity bool) *CgmSupportPackageStatusResponse {
	m := &CgmSupportPackageStatusResponse{Status: status, Validity: validity}
	m.SetCargo([]byte{byte(status), helpers.Bool(validity)})
	return m
}

// Parse implements messages.Message.
func (m *CgmSupportPackageStatusResponse) Parse(raw []byte) error {
	if len(raw) != cgmSupportPackageStatusResponseProps.Size {
		return messages.InvalidLength("CgmSupportPackageStatusResponse", len(raw), cgmSupportPackageStatusResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(raw[0])
	m.Validity = raw[1] != 0
	return nil
}

// Success reports whether the pump acknowledged the request.
func (m *CgmSupportPackageStatusResponse) Success() bool { return m.Status == 0 }
