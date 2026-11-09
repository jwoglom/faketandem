package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// CartridgeHandler handles cartridge change mode requests
type CartridgeHandler struct {
	bridge       *pumpx2.Bridge
	msgType      string
	responseType string
}

// NewCartridgeHandler creates a new cartridge mode handler
func NewCartridgeHandler(bridge *pumpx2.Bridge, msgType string) *CartridgeHandler {
	responseType := msgType
	if len(responseType) > 7 && responseType[len(responseType)-7:] == "Request" {
		responseType = responseType[:len(responseType)-7] + "Response"
	}
	return &CartridgeHandler{
		bridge:       bridge,
		msgType:      msgType,
		responseType: responseType,
	}
}

// MessageType returns the message type this handler processes
func (h *CartridgeHandler) MessageType() string {
	return h.msgType
}

// RequiresAuth returns true if this message requires authentication
func (h *CartridgeHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a cartridge mode request
func (h *CartridgeHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling %s: txID=%d", h.msgType, msg.TxID)

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		h.responseType,
		map[string]interface{}{
			"status": 0,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", h.responseType, err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// CartridgeModeHandler enters or leaves a cartridge procedure, answering nonzero when the pump
// would refuse (see state.WorkflowMode), and streams the procedure's progress on CONTROL_STREAM.
type CartridgeModeHandler struct {
	*CartridgeHandler
	router *Router
	mode   state.WorkflowMode
	enter  bool
}

// NewCartridgeModeHandler creates a handler that enters (or, with enter false, exits) mode.
func NewCartridgeModeHandler(router *Router, msgType string, mode state.WorkflowMode, enter bool) *CartridgeModeHandler {
	return &CartridgeModeHandler{
		CartridgeHandler: NewCartridgeHandler(router.bridge, msgType),
		router:           router,
		mode:             mode,
		enter:            enter,
	}
}

// HandleMessage processes an enter or exit request for the handler's mode.
func (h *CartridgeModeHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling %s: txID=%d", h.msgType, msg.TxID)

	txID := uint8(msg.TxID)
	var accepted bool
	var stream []*protocol.NativeMessage
	switch {
	case h.enter && h.mode == state.WorkflowFillTubing:
		accepted = pumpState.EnterFillTubing(txID)
	case h.enter:
		if accepted = pumpState.EnterChangeCartridge(txID); accepted {
			stream = h.router.startChangeCartridge(pumpState, txID)
		}
	case h.mode == state.WorkflowFillTubing:
		accepted, stream = h.router.exitFillTubing(pumpState, txID)
	default:
		accepted, stream = h.router.exitChangeCartridge(pumpState, txID)
	}

	response, err := h.encodeStatus(msg.TxID, accepted)
	if err != nil {
		return nil, err
	}
	return &Response{
		ResponseMessage:     response,
		NativeNotifications: stream,
		Immediate:           true,
	}, nil
}

func (h *CartridgeHandler) encodeStatus(txID int, accepted bool) (*pumpx2.EncodedMessage, error) {
	status := 1
	if accepted {
		status = 0
	}
	response, err := h.bridge.EncodeMessage(txID, h.responseType, map[string]interface{}{"status": status})
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", h.responseType, err)
	}
	return response, nil
}

// FillCannulaHandler primes a Mobi's cannula. The pump answers at once and streams
// FillCannulaStateStreamResponse CANNULA_FILLED once the prime has gone in, at the bolus rate.
type FillCannulaHandler struct {
	*CartridgeHandler
	router *Router
}

// NewFillCannulaHandler creates the FillCannulaRequest handler.
func NewFillCannulaHandler(router *Router) *FillCannulaHandler {
	return &FillCannulaHandler{CartridgeHandler: NewCartridgeHandler(router.bridge, "FillCannulaRequest"), router: router}
}

// HandleMessage starts the prime, or refuses it unless delivery is suspended with no procedure open.
func (h *FillCannulaHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	milliunits, _ := cargoInt(msg, "primeSizeMilliUnits")
	log.Infof("Handling FillCannulaRequest: txID=%d, prime %d mU", msg.TxID, milliunits)

	units := float64(milliunits) / 1000
	accepted := milliunits > 0 && pumpState.StartFillCannula()
	var stream []*protocol.NativeMessage
	if accepted {
		stream = h.router.startFillCannula(pumpState, uint8(msg.TxID), units)
	}

	response, err := h.encodeStatus(msg.TxID, accepted)
	if err != nil {
		return nil, err
	}
	return &Response{ResponseMessage: response, NativeNotifications: stream, Immediate: true}, nil
}

// LoadStatusHandler answers LoadStatusRequest from the open procedure (state.CurrentLoadStatus).
type LoadStatusHandler struct {
	bridge *pumpx2.Bridge
}

// MessageType returns the message type this handler processes
func (h *LoadStatusHandler) MessageType() string { return "LoadStatusRequest" }

// RequiresAuth returns true
func (h *LoadStatusHandler) RequiresAuth() bool { return true }

// HandleMessage reports the load status.
func (h *LoadStatusHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	load := pumpState.CurrentLoadStatus()
	active := 0
	if load.Active {
		active = 1
	}
	// The deprecated (status, unknown) constructor is the one cliparser can reach; unknown is bytes 1-2.
	response, err := h.bridge.EncodeMessage(msg.TxID, "LoadStatusResponse", map[string]interface{}{
		"status":  active,
		"unknown": load.LoadStateID | load.PrimeStatusID<<8,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode LoadStatusResponse: %w", err)
	}
	return &Response{ResponseMessage: response, Immediate: true}, nil
}
