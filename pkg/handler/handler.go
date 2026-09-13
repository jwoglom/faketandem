package handler

import (
	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"
)

// MessageHandler handles a specific message type
type MessageHandler interface {
	// HandleMessage processes a message and returns a response
	HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error)

	// MessageType returns the message type this handler processes
	MessageType() string

	// RequiresAuth returns true if this message requires authentication
	RequiresAuth() bool
}

// Response represents the response from a message handler
type Response struct {
	// Response message to send (if any)
	ResponseMessage *pumpx2.EncodedMessage

	// NativeResponse is a response built by the Go encoder in pkg/protocol
	// instead of by the pumpX2 cliparser, for messages cliparser cannot encode
	// (HistoryLogStreamResponse, ErrorResponse, AlarmStatusResponse). It carries
	// its own characteristic, so it is sent there regardless of Characteristic
	// below. A handler may set this instead of, or in addition to,
	// ResponseMessage.
	NativeResponse *protocol.NativeMessage

	// NativeNotifications are further natively-encoded messages to notify after
	// the main response has gone out, in order. This is how a handler answers a
	// request and then streams: HistoryLogRequest replies with a
	// HistoryLogResponse on CURRENT_STATUS and puts the
	// HistoryLogStreamResponse messages here.
	NativeNotifications []*protocol.NativeMessage

	// Characteristic to send on (defaults to same as request)
	Characteristic bluetooth.CharacteristicType

	// Whether to send immediately or queue
	Immediate bool

	// Additional notifications to send
	Notifications []*Notification

	// State changes to apply
	StateChanges []StateChange
}

// Notification represents a notification to send on a characteristic
type Notification struct {
	Characteristic bluetooth.CharacteristicType
	Message        *pumpx2.EncodedMessage
}

// StateChange represents a change to pump state
type StateChange struct {
	Type StateChangeType
	Data interface{}
}

// StateChangeType identifies the type of state change
type StateChangeType int

// State change type constants
const (
	// StateChangeAuth indicates authentication state changed
	StateChangeAuth StateChangeType = iota
	// StateChangeBasal indicates basal rate changed
	StateChangeBasal
	// StateChangeBolus indicates bolus state changed
	StateChangeBolus
	// StateChangeReservoir indicates reservoir level changed
	StateChangeReservoir
	// StateChangeBattery indicates battery level changed
	StateChangeBattery
	// StateChangeAlert indicates alert state changed
	StateChangeAlert
	// StateChangeTime indicates time since reset changed
	StateChangeTime
	// StateChangeSuspend indicates pump suspend/resume
	StateChangeSuspend
)
