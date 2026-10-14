package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ProfileStatusResponse is which insulin delivery profiles the pump holds.
// Each slot holds a profile ID to ask IDPSettingsRequest about, or -1 when
// empty. Slot 0 always holds the active profile, so no profile exists when
// it is -1. ActiveSegmentIndex is the active profile's segment in force.
type ProfileStatusResponse struct {
	messages.Base
	NumberOfProfiles   int32 `pumpx2:"numberOfProfiles"`
	IDPSlot0ID         int32 `pumpx2:"idpSlot0Id"`
	IDPSlot1ID         int32 `pumpx2:"idpSlot1Id"`
	IDPSlot2ID         int32 `pumpx2:"idpSlot2Id"`
	IDPSlot3ID         int32 `pumpx2:"idpSlot3Id"`
	IDPSlot4ID         int32 `pumpx2:"idpSlot4Id"`
	IDPSlot5ID         int32 `pumpx2:"idpSlot5Id"`
	ActiveSegmentIndex int32 `pumpx2:"activeSegmentIndex"`
}

var profileStatusResponseProps = messages.Props{OpCode: 63, Size: 8, Type: messages.Response, Request: "ProfileStatusRequest"}

func init() {
	messages.Register("ProfileStatusResponse", &ProfileStatusResponse{}, profileStatusResponseProps,
		messages.Constructor{
			Params: "int numberOfProfiles, int idpSlot0Id, int idpSlot1Id, int idpSlot2Id, int idpSlot3Id, int idpSlot4Id, int idpSlot5Id, int activeSegmentIndex",
			Func:   NewProfileStatusResponse,
		},
	)
}

// NewProfileStatusResponse builds the response; each value goes on the wire
// as one byte.
func NewProfileStatusResponse(numberOfProfiles, idpSlot0ID, idpSlot1ID, idpSlot2ID, idpSlot3ID, idpSlot4ID, idpSlot5ID, activeSegmentIndex int32) *ProfileStatusResponse {
	m := &ProfileStatusResponse{
		NumberOfProfiles:   numberOfProfiles,
		IDPSlot0ID:         idpSlot0ID,
		IDPSlot1ID:         idpSlot1ID,
		IDPSlot2ID:         idpSlot2ID,
		IDPSlot3ID:         idpSlot3ID,
		IDPSlot4ID:         idpSlot4ID,
		IDPSlot5ID:         idpSlot5ID,
		ActiveSegmentIndex: activeSegmentIndex,
	}
	m.SetCargo([]byte{byte(numberOfProfiles), byte(idpSlot0ID), byte(idpSlot1ID), byte(idpSlot2ID), byte(idpSlot3ID),
		byte(idpSlot4ID), byte(idpSlot5ID), byte(activeSegmentIndex)})
	return m
}

// Parse implements messages.Message.
func (m *ProfileStatusResponse) Parse(raw []byte) error {
	if len(raw) != profileStatusResponseProps.Size {
		return messages.InvalidLength("ProfileStatusResponse", len(raw), profileStatusResponseProps.Size)
	}
	m.SetCargo(raw)
	m.NumberOfProfiles = int32(int8(raw[0]))
	m.IDPSlot0ID = int32(int8(raw[1]))
	m.IDPSlot1ID = int32(int8(raw[2]))
	m.IDPSlot2ID = int32(int8(raw[3]))
	m.IDPSlot3ID = int32(int8(raw[4]))
	m.IDPSlot4ID = int32(int8(raw[5]))
	m.IDPSlot5ID = int32(int8(raw[6]))
	m.ActiveSegmentIndex = int32(int8(raw[7]))
	return nil
}

// IDPSlotIDs are the first NumberOfProfiles slots' profile IDs. Like pumpX2's
// getIdpSlotIds, it panics when NumberOfProfiles is not 0 to 6.
func (m *ProfileStatusResponse) IDPSlotIDs() []int32 {
	return []int32{m.IDPSlot0ID, m.IDPSlot1ID, m.IDPSlot2ID, m.IDPSlot3ID, m.IDPSlot4ID, m.IDPSlot5ID}[:m.NumberOfProfiles]
}
