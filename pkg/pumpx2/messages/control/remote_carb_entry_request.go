package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// RemoteCarbEntryRequest records the carbs entered for a bolus in progress
// (between its BolusPermissionRequest and InitiateBolusRequest, and sent
// before the latter) in the pump's history, without which the bolus shows no
// carbs. Carbs is in grams, and PumpTime is the pump's seconds since boot
// (TimeSinceResetResponse's pump time).
type RemoteCarbEntryRequest struct {
	messages.Base
	Carbs int32 `pumpx2:"carbs"`
	// Unknown is 1 in every capture.
	Unknown  int32 `pumpx2:"unknown"`
	PumpTime int64 `pumpx2:"pumpTime"`
	BolusID  int32 `pumpx2:"bolusId"`
}

var remoteCarbEntryRequestProps = messages.Props{OpCode: -14, Size: 9, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	MinAPI: "API_V2_5", Response: "RemoteCarbEntryResponse"}

func init() {
	messages.Register("RemoteCarbEntryRequest", &RemoteCarbEntryRequest{}, remoteCarbEntryRequestProps,
		messages.Constructor{Params: "int carbs, int unknown, long pumpTimeSecondsSinceBoot, int bolusId", Func: NewRemoteCarbEntryRequestWithUnknown},
		messages.Constructor{Params: "int carbs, long pumpTimeSecondsSinceBoot, int bolusId", Func: NewRemoteCarbEntryRequest},
	)
}

// NewRemoteCarbEntryRequest builds the request, with Unknown 1, for the
// bolus ID a BolusPermissionResponse granted. It fails when
// pumpTimeSecondsSinceBoot is large enough to be a Unix time instead.
func NewRemoteCarbEntryRequest(carbs int32, pumpTimeSecondsSinceBoot int64, bolusID int32) (*RemoteCarbEntryRequest, error) {
	return NewRemoteCarbEntryRequestWithUnknown(carbs, 1, pumpTimeSecondsSinceBoot, bolusID)
}

// NewRemoteCarbEntryRequestWithUnknown is NewRemoteCarbEntryRequest with
// Unknown given.
func NewRemoteCarbEntryRequestWithUnknown(carbs, unknown int32, pumpTimeSecondsSinceBoot int64, bolusID int32) (*RemoteCarbEntryRequest, error) {
	if pumpTimeSecondsSinceBoot >= helpers.January12008UnixEpoch {
		return nil, messages.Invalid("RemoteCarbEntryRequest: pump time %d is a Unix time, not seconds since boot", pumpTimeSecondsSinceBoot)
	}
	m := &RemoteCarbEntryRequest{Carbs: carbs, Unknown: unknown, PumpTime: pumpTimeSecondsSinceBoot, BolusID: bolusID}
	m.SetCargo(helpers.Combine(
		helpers.FirstTwoBytesLittleEndian(carbs),
		[]byte{byte(unknown)},
		helpers.ToUint32(pumpTimeSecondsSinceBoot),
		helpers.FirstTwoBytesLittleEndian(bolusID),
	))
	return m, nil
}

// Parse implements messages.Message.
func (m *RemoteCarbEntryRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(remoteCarbEntryRequestProps, raw)
	if len(raw) != remoteCarbEntryRequestProps.Size {
		return messages.InvalidLength("RemoteCarbEntryRequest", len(raw), remoteCarbEntryRequestProps.Size)
	}
	m.SetCargo(raw)
	m.Carbs = helpers.ReadShort(raw, 0)
	m.Unknown = int32(int8(raw[2]))
	m.PumpTime = helpers.ReadUint32(raw, 3)
	m.BolusID = helpers.ReadShort(raw, 7)
	return nil
}
