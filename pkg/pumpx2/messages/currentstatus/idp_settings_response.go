package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// IDPSettingsResponse is an insulin delivery profile's settings. Its time
// segments are read one at a time with IDPSegmentRequest.
type IDPSettingsResponse struct {
	messages.Base
	IDPID int32 `pumpx2:"idpId"`
	// Name is the profile's name: up to 16 bytes of UTF-8 on the wire.
	Name                    string `pumpx2:"name"`
	NumberOfProfileSegments int32  `pumpx2:"numberOfProfileSegments"`
	// InsulinDuration is in minutes.
	InsulinDuration int32 `pumpx2:"insulinDuration"`
	// MaxBolus is in milliunits.
	MaxBolus  int32 `pumpx2:"maxBolus"`
	CarbEntry bool  `pumpx2:"carbEntry"`
}

var idpSettingsResponseProps = messages.Props{OpCode: 65, Size: 23, Type: messages.Response, Request: "IDPSettingsRequest"}

func init() {
	messages.Register("IDPSettingsResponse", &IDPSettingsResponse{}, idpSettingsResponseProps,
		messages.Constructor{
			Params: "int idpId, String name, int numberOfProfileSegments, int insulinDuration, int maxBolus, boolean carbEntry",
			Func:   NewIDPSettingsResponse,
		},
	)
}

// NewIDPSettingsResponse builds the response. It fails when name is more
// than 16 bytes of UTF-8.
func NewIDPSettingsResponse(idpID int32, name string, numberOfProfileSegments, insulinDuration, maxBolus int32, carbEntry bool) (*IDPSettingsResponse, error) {
	nameBytes, err := helpers.WriteString(name, 16)
	if err != nil {
		return nil, messages.Invalid("IDPSettingsResponse: %v", err)
	}
	m := &IDPSettingsResponse{
		IDPID:                   idpID,
		Name:                    name,
		NumberOfProfileSegments: numberOfProfileSegments,
		InsulinDuration:         insulinDuration,
		MaxBolus:                maxBolus,
		CarbEntry:               carbEntry,
	}
	m.SetCargo(helpers.Combine(
		[]byte{byte(idpID)},
		nameBytes,
		[]byte{byte(numberOfProfileSegments)},
		helpers.FirstTwoBytesLittleEndian(insulinDuration),
		helpers.FirstTwoBytesLittleEndian(maxBolus),
		[]byte{helpers.Bool(carbEntry)},
	))
	return m, nil
}

// Parse implements messages.Message.
func (m *IDPSettingsResponse) Parse(raw []byte) error {
	if len(raw) != idpSettingsResponseProps.Size {
		return messages.InvalidLength("IDPSettingsResponse", len(raw), idpSettingsResponseProps.Size)
	}
	m.SetCargo(raw)
	m.IDPID = int32(int8(raw[0]))
	m.Name = helpers.ReadString(raw, 1, 16)
	m.NumberOfProfileSegments = int32(int8(raw[17]))
	m.InsulinDuration = helpers.ReadShort(raw, 18)
	m.MaxBolus = helpers.ReadShort(raw, 20)
	m.CarbEntry = raw[22] != 0
	return nil
}
