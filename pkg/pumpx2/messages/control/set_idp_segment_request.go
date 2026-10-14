package control

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// SetIDPSegmentRequest modifies, creates or deletes a segment of an insulin
// delivery profile. IDPID is the profile's ID, not its slot; ProfileIndex is
// its slot (0-5) in ProfileStatusResponse. The units are IDPSegmentResponse's,
// and IDPStatusID is a currentstatus.IDPSegmentStatus bitmask of the fields
// the request sets.
type SetIDPSegmentRequest struct {
	messages.Base
	IDPID        int32                `pumpx2:"idpId"`
	ProfileIndex int32                `pumpx2:"profileIndex"`
	SegmentIndex int32                `pumpx2:"segmentIndex"`
	OperationID  int32                `pumpx2:"operationId"`
	Operation    *IDPSegmentOperation `pumpx2:"operation"`
	// ProfileStartTime is minutes since midnight.
	ProfileStartTime int32 `pumpx2:"profileStartTime"`
	// ProfileBasalRate is in milliunits per hour.
	ProfileBasalRate int32 `pumpx2:"profileBasalRate"`
	// ProfileCarbRatio is grams per unit, times 1000.
	ProfileCarbRatio int64 `pumpx2:"profileCarbRatio"`
	// ProfileTargetBG is in mg/dL.
	ProfileTargetBG int32 `pumpx2:"profileTargetBG"`
	// ProfileISF is mg/dL per unit.
	ProfileISF  int32                            `pumpx2:"profileISF"`
	IDPStatusID int32                            `pumpx2:"idpStatusId"`
	IDPStatus   []currentstatus.IDPSegmentStatus `pumpx2:"idpStatus"`
}

var setIDPSegmentRequestProps = messages.Props{OpCode: -86, Size: 17, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "SetIDPSegmentResponse"}

func init() {
	messages.Register("SetIDPSegmentRequest", &SetIDPSegmentRequest{}, setIDPSegmentRequestProps,
		messages.Constructor{
			Params: "int idpId, int profileIndex, int segmentIndex, SetIDPSegmentRequest.IDPSegmentOperation operation, int profileStartTime, " +
				"int profileBasalRate, long profileCarbRatio, int profileTargetBG, int profileISF, int idpStatusId",
			Func: NewSetIDPSegmentRequest,
		},
		messages.Constructor{Params: "byte[] raw", Func: NewSetIDPSegmentRequestFromCargo},
	)
}

// NewSetIDPSegmentRequest builds the request.
func NewSetIDPSegmentRequest(idpID, profileIndex, segmentIndex int32, operation IDPSegmentOperation, profileStartTime, profileBasalRate int32,
	profileCarbRatio int64, profileTargetBG, profileISF, idpStatusID int32) *SetIDPSegmentRequest {
	m := &SetIDPSegmentRequest{
		IDPID:            idpID,
		ProfileIndex:     profileIndex,
		SegmentIndex:     segmentIndex,
		OperationID:      int32(operation),
		Operation:        IDPSegmentOperationFromID(int32(operation)),
		ProfileStartTime: profileStartTime,
		ProfileBasalRate: profileBasalRate,
		ProfileCarbRatio: profileCarbRatio,
		ProfileTargetBG:  profileTargetBG,
		ProfileISF:       profileISF,
		IDPStatusID:      idpStatusID,
		IDPStatus:        currentstatus.IDPSegmentStatusFromBitmask(idpStatusID),
	}
	m.SetCargo(helpers.Combine(
		[]byte{byte(idpID), byte(profileIndex), byte(segmentIndex), byte(operation)},
		helpers.FirstTwoBytesLittleEndian(profileStartTime),
		helpers.FirstTwoBytesLittleEndian(profileBasalRate),
		helpers.ToUint32(profileCarbRatio),
		helpers.FirstTwoBytesLittleEndian(profileTargetBG),
		helpers.FirstTwoBytesLittleEndian(profileISF),
		[]byte{byte(idpStatusID)},
	))
	return m
}

// NewSetIDPSegmentRequestFromCargo parses a request's payload.
func NewSetIDPSegmentRequestFromCargo(raw []byte) (*SetIDPSegmentRequest, error) {
	m := &SetIDPSegmentRequest{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *SetIDPSegmentRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setIDPSegmentRequestProps, raw)
	if len(raw) != setIDPSegmentRequestProps.Size {
		return messages.InvalidLength("SetIDPSegmentRequest", len(raw), setIDPSegmentRequestProps.Size)
	}
	m.SetCargo(raw)
	m.IDPID = int32(int8(raw[0]))
	m.ProfileIndex = int32(int8(raw[1]))
	m.SegmentIndex = int32(int8(raw[2]))
	m.OperationID = int32(int8(raw[3]))
	m.Operation = IDPSegmentOperationFromID(m.OperationID)
	m.ProfileStartTime = helpers.ReadShort(raw, 4)
	m.ProfileBasalRate = helpers.ReadShort(raw, 6)
	m.ProfileCarbRatio = helpers.ReadUint32(raw, 8)
	m.ProfileTargetBG = helpers.ReadShort(raw, 12)
	m.ProfileISF = helpers.ReadShort(raw, 14)
	m.IDPStatusID = int32(int8(raw[16]))
	m.IDPStatus = currentstatus.IDPSegmentStatusFromBitmask(m.IDPStatusID)
	return nil
}

// IDPSegmentOperation is what a SetIDPSegmentRequest does to its segment.
type IDPSegmentOperation int32

// The segment operations, by their wire IDs.
const (
	// IDPSegmentOperationModifySegmentID modifies the segment at SegmentIndex.
	IDPSegmentOperationModifySegmentID IDPSegmentOperation = 0
	// IDPSegmentOperationCreateSegment creates a segment; SegmentIndex is
	// unused and always 0.
	IDPSegmentOperationCreateSegment IDPSegmentOperation = 1
	// IDPSegmentOperationDeleteSegmentID deletes the segment at SegmentIndex.
	IDPSegmentOperationDeleteSegmentID IDPSegmentOperation = 2
)

// IDPSegmentOperationFromID is the operation with wire ID id, or nil for an
// ID that is none.
func IDPSegmentOperationFromID(id int32) *IDPSegmentOperation {
	switch o := IDPSegmentOperation(id); o {
	case IDPSegmentOperationModifySegmentID, IDPSegmentOperationCreateSegment, IDPSegmentOperationDeleteSegmentID:
		return &o
	}
	return nil
}

func (o IDPSegmentOperation) String() string {
	switch o {
	case IDPSegmentOperationModifySegmentID:
		return "MODIFY_SEGMENT_ID"
	case IDPSegmentOperationCreateSegment:
		return "CREATE_SEGMENT"
	case IDPSegmentOperationDeleteSegmentID:
		return "DELETE_SEGMENT_ID"
	}
	return fmt.Sprintf("IDPSegmentOperation(%d)", int32(o))
}
