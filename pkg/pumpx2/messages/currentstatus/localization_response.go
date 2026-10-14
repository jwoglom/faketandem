package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// LocalizationResponse is the pump's glucose unit, selected language, region
// and a bitmask of the languages it has.
type LocalizationResponse struct {
	messages.Base
	GlucoseUOM                int32 `pumpx2:"glucoseUOM"`
	LanguageSelected          int32 `pumpx2:"languageSelected"`
	RegionSetting             int32 `pumpx2:"regionSetting"`
	LanguagesAvailableBitmask int64 `pumpx2:"languagesAvailableBitmask"`
}

var localizationResponseProps = messages.Props{OpCode: -89, Size: 7, Type: messages.Response, Request: "LocalizationRequest"}

func init() {
	messages.Register("LocalizationResponse", &LocalizationResponse{}, localizationResponseProps,
		messages.Constructor{
			Params: "int glucoseUOM, int languageSelected, int regionSetting, long languagesAvailableBitmask",
			Func:   NewLocalizationResponse,
		},
	)
}

// NewLocalizationResponse builds the response.
func NewLocalizationResponse(glucoseUOM, languageSelected, regionSetting int32, languagesAvailableBitmask int64) *LocalizationResponse {
	m := &LocalizationResponse{
		GlucoseUOM:                glucoseUOM,
		LanguageSelected:          languageSelected,
		RegionSetting:             regionSetting,
		LanguagesAvailableBitmask: languagesAvailableBitmask,
	}
	m.SetCargo(helpers.Combine([]byte{byte(glucoseUOM), byte(languageSelected), byte(regionSetting)}, helpers.ToUint32(languagesAvailableBitmask)))
	return m
}

// Parse implements messages.Message.
func (m *LocalizationResponse) Parse(raw []byte) error {
	if len(raw) != localizationResponseProps.Size {
		return messages.InvalidLength("LocalizationResponse", len(raw), localizationResponseProps.Size)
	}
	m.SetCargo(raw)
	m.GlucoseUOM = int32(int8(raw[0]))
	m.LanguageSelected = int32(int8(raw[1]))
	m.RegionSetting = int32(int8(raw[2]))
	m.LanguagesAvailableBitmask = helpers.ReadUint32(raw, 3)
	return nil
}
