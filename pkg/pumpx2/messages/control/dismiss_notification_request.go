package control

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// DismissNotificationRequest dismisses a reminder, alert, alarm or CGM alert
// on the pump. NotificationID identifies it within its kind: a reminder or
// CGM alert by its ID, an alert or alarm by its bit in the pump's status
// bitmask. NotificationType is nil when NotificationTypeID is none.
type DismissNotificationRequest struct {
	messages.Base
	NotificationID     int64             `pumpx2:"notificationId"`
	NotificationTypeID int32             `pumpx2:"notificationTypeId"`
	NotificationType   *NotificationType `pumpx2:"notificationType"`
	ExecuteExtraAction bool              `pumpx2:"executeExtraAction"`
}

var dismissNotificationRequestProps = messages.Props{OpCode: -72, Size: 6, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "DismissNotificationResponse"}

func init() {
	messages.Register("DismissNotificationRequest", &DismissNotificationRequest{}, dismissNotificationRequestProps,
		messages.Constructor{Params: "byte[] raw", Func: NewDismissNotificationRequestFromCargo},
		messages.Constructor{
			Params: "DismissNotificationRequest.NotificationType notificationType, long notificationId, boolean executeExtraAction",
			Func:   NewDismissNotificationRequestWithExtraAction,
		},
		messages.Constructor{Params: "DismissNotificationRequest.NotificationType notificationType, long notificationId", Func: NewDismissNotificationRequest},
	)
}

// NewDismissNotificationRequest builds the request without the extra
// action.
func NewDismissNotificationRequest(notificationType NotificationType, notificationID int64) *DismissNotificationRequest {
	return NewDismissNotificationRequestWithExtraAction(notificationType, notificationID, false)
}

// NewDismissNotificationRequestWithExtraAction builds the request. Its fields
// are what the payload reads back as: NotificationID keeps its low 32 bits.
func NewDismissNotificationRequestWithExtraAction(notificationType NotificationType, notificationID int64, executeExtraAction bool) *DismissNotificationRequest {
	m := &DismissNotificationRequest{}
	m.decode(helpers.Combine(
		helpers.ToUint32(notificationID),
		helpers.FirstByteLittleEndian(int32(notificationType)),
		[]byte{helpers.Bool(executeExtraAction)},
	))
	return m
}

// NewDismissNotificationRequestFromCargo parses a request's payload.
func NewDismissNotificationRequestFromCargo(raw []byte) (*DismissNotificationRequest, error) {
	m := &DismissNotificationRequest{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *DismissNotificationRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(dismissNotificationRequestProps, raw)
	if len(raw) != dismissNotificationRequestProps.Size {
		return messages.InvalidLength("DismissNotificationRequest", len(raw), dismissNotificationRequestProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *DismissNotificationRequest) decode(raw []byte) {
	m.SetCargo(raw)
	m.NotificationID = helpers.ReadUint32(raw, 0)
	m.NotificationTypeID = int32(raw[4])
	m.ExecuteExtraAction = raw[5] != 0
	m.NotificationType = NotificationTypeFromID(m.NotificationTypeID)
}

// NotificationType is the kind of notification a DismissNotificationRequest
// dismisses.
type NotificationType int32

// The notification types, by their wire IDs.
const (
	NotificationTypeReminder NotificationType = 0
	NotificationTypeAlert    NotificationType = 1
	NotificationTypeAlarm    NotificationType = 2
	NotificationTypeCGMAlert NotificationType = 3
)

var notificationTypeNames = []string{"REMINDER", "ALERT", "ALARM", "CGM_ALERT"}

// NotificationTypeFromID is the type with wire ID id, or nil for an ID that
// is none.
func NotificationTypeFromID(id int32) *NotificationType {
	if id < 0 || int(id) >= len(notificationTypeNames) {
		return nil
	}
	t := NotificationType(id)
	return &t
}

func (t NotificationType) String() string {
	if t >= 0 && int(t) < len(notificationTypeNames) {
		return notificationTypeNames[t]
	}
	return fmt.Sprintf("NotificationType(%d)", int32(t))
}
