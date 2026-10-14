package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CGMHardwareInfoResponse is the CGM transmitter's ID (a G6's, such as
// "8RR239"), in a 16-byte field, and one undocumented byte after it.
type CGMHardwareInfoResponse struct {
	messages.Base
	HardwareInfoString string `pumpx2:"hardwareInfoString"`
	LastByte           int32  `pumpx2:"lastByte"`
}

var cgmHardwareInfoResponseProps = messages.Props{OpCode: 97, Size: 17, Type: messages.Response, Request: "CGMHardwareInfoRequest"}

func init() {
	messages.Register("CGMHardwareInfoResponse", &CGMHardwareInfoResponse{}, cgmHardwareInfoResponseProps,
		messages.Constructor{Params: "String hardwareInfoString, int lastByte", Func: NewCGMHardwareInfoResponse},
	)
}

// NewCGMHardwareInfoResponse builds the response. It fails when
// hardwareInfoString is longer than 16 bytes of UTF-8.
func NewCGMHardwareInfoResponse(hardwareInfoString string, lastByte int32) (*CGMHardwareInfoResponse, error) {
	field, err := helpers.WriteString(hardwareInfoString, 16)
	if err != nil {
		return nil, messages.Invalid("CGMHardwareInfoResponse: %v", err)
	}
	m := &CGMHardwareInfoResponse{HardwareInfoString: hardwareInfoString, LastByte: lastByte}
	m.SetCargo(helpers.Combine(field, []byte{byte(lastByte)}))
	return m, nil
}

// Parse implements messages.Message.
func (m *CGMHardwareInfoResponse) Parse(raw []byte) error {
	if len(raw) != cgmHardwareInfoResponseProps.Size {
		return messages.InvalidLength("CGMHardwareInfoResponse", len(raw), cgmHardwareInfoResponseProps.Size)
	}
	m.HardwareInfoString = helpers.ReadString(raw, 0, 16)
	m.LastByte = int32(int8(raw[16]))
	m.SetCargo(raw)
	return nil
}
