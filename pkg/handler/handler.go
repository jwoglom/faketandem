package handler

import (
	"github.com/jwoglom/faketandem/pkg/bluetooth"
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

const (
	StateChangeAuth StateChangeType = iota
	StateChangeBasal
	StateChangeBolus
	StateChangeReservoir
	StateChangeBattery
	StateChangeAlert
	StateChangeTime
)
