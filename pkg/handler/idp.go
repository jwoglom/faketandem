package handler

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// The IDP handlers serve the insulin delivery profiles held in state.PumpState. Requests are
// decoded from their raw cargo, whose layout is pumpX2's (and TandemKit's), rather than from the
// field names cliparser prints.

// requestCargo returns the raw cargo of msg, at least minLen bytes long.
func requestCargo(msg *pumpx2.ParsedMessage, minLen int) ([]byte, error) {
	raw, ok := cargoValue(msg, "cargo")
	if !ok {
		return nil, fmt.Errorf("no cargo field")
	}
	rawHex, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("cargo is %T, not a hex string", raw)
	}
	b, err := hex.DecodeString(rawHex)
	if err != nil {
		return nil, fmt.Errorf("cargo %q is not hex: %w", rawHex, err)
	}
	if len(b) < minLen {
		return nil, fmt.Errorf("cargo %q is %d bytes, want at least %d", rawHex, len(b), minLen)
	}
	return b, nil
}

// IDPHandler answers one IDP request from the pump's profiles.
type IDPHandler struct {
	bridge  *pumpx2.Bridge
	msgType string
	handle  func(h *IDPHandler, msg *pumpx2.ParsedMessage, ps *state.PumpState) (*Response, error)
}

// MessageType returns the message type this handler processes
func (h *IDPHandler) MessageType() string { return h.msgType }

// RequiresAuth returns true if this message requires authentication
func (h *IDPHandler) RequiresAuth() bool { return true }

// HandleMessage answers the request.
func (h *IDPHandler) HandleMessage(msg *pumpx2.ParsedMessage, ps *state.PumpState) (*Response, error) {
	log.Infof("Handling %s: txID=%d", h.msgType, msg.TxID)
	return h.handle(h, msg, ps)
}

func (h *IDPHandler) respond(msg *pumpx2.ParsedMessage, params map[string]interface{}) (*Response, error) {
	responseType := strings.TrimSuffix(h.msgType, "Request") + "Response"
	encoded, err := h.bridge.EncodeMessage(msg.TxID, responseType, params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", responseType, err)
	}
	return &Response{ResponseMessage: encoded, Immediate: true}, nil
}

// respondNoSuchProfile answers a read for a profile the pump does not hold. The error code a real
// pump gives is not known; 0 is pumpX2's UNDEFINED_ERROR.
func (h *IDPHandler) respondNoSuchProfile(msg *pumpx2.ParsedMessage, idpID int) (*Response, error) {
	log.Warnf("%s for profile %d, which the pump does not hold", h.msgType, idpID)
	if ErrorResponseEncoder == nil {
		return nil, fmt.Errorf("%s for unknown profile %d and no ErrorResponseEncoder", h.msgType, idpID)
	}
	encoded, err := ErrorResponseEncoder(msg.TxID, 0, msg.Opcode&0xFF, msg.MessageType)
	if err != nil {
		return nil, err
	}
	return &Response{ResponseMessage: encoded, Immediate: true}, nil
}

// NewProfileStatusHandler answers ProfileStatusRequest: the profile ids in slot order, -1 for an
// empty slot.
func NewProfileStatusHandler(bridge *pumpx2.Bridge) *IDPHandler {
	return &IDPHandler{bridge: bridge, msgType: "ProfileStatusRequest", handle: handleProfileStatus}
}

func handleProfileStatus(h *IDPHandler, msg *pumpx2.ParsedMessage, ps *state.PumpState) (*Response, error) {
	profiles := ps.IDPProfiles()
	params := map[string]interface{}{
		"numberOfProfiles":   len(profiles),
		"activeSegmentIndex": ps.ActiveIDPSegmentIndex(),
	}
	for slot := 0; slot < state.MaxIDPProfiles; slot++ {
		id := -1
		if slot < len(profiles) {
			id = profiles[slot].ID
		}
		params[fmt.Sprintf("idpSlot%dId", slot)] = id
	}
	return h.respond(msg, params)
}

// NewIDPSettingsHandler answers IDPSettingsRequest (cargo: idpId).
func NewIDPSettingsHandler(bridge *pumpx2.Bridge) *IDPHandler {
	return &IDPHandler{bridge: bridge, msgType: "IDPSettingsRequest", handle: handleIDPSettings}
}

func handleIDPSettings(h *IDPHandler, msg *pumpx2.ParsedMessage, ps *state.PumpState) (*Response, error) {
	cargo, err := requestCargo(msg, 1)
	if err != nil {
		return nil, fmt.Errorf("IDPSettingsRequest: %w", err)
	}
	idpID := int(cargo[0])
	profile, ok := ps.IDPProfile(idpID)
	if !ok {
		return h.respondNoSuchProfile(msg, idpID)
	}
	return h.respond(msg, map[string]interface{}{
		"idpId":                   profile.ID,
		"name":                    profile.Name,
		"numberOfProfileSegments": len(profile.Segments),
		"insulinDuration":         profile.InsulinDuration,
		"maxBolus":                profile.MaxBolus,
		"carbEntry":               profile.CarbEntry,
	})
}

// NewIDPSegmentHandler answers IDPSegmentRequest (cargo: idpId, segmentIndex).
func NewIDPSegmentHandler(bridge *pumpx2.Bridge) *IDPHandler {
	return &IDPHandler{bridge: bridge, msgType: "IDPSegmentRequest", handle: handleIDPSegment}
}

func handleIDPSegment(h *IDPHandler, msg *pumpx2.ParsedMessage, ps *state.PumpState) (*Response, error) {
	cargo, err := requestCargo(msg, 2)
	if err != nil {
		return nil, fmt.Errorf("IDPSegmentRequest: %w", err)
	}
	idpID, index := int(cargo[0]), int(cargo[1])
	profile, ok := ps.IDPProfile(idpID)
	if !ok || index >= len(profile.Segments) {
		return h.respondNoSuchProfile(msg, idpID)
	}
	segment := profile.Segments[index]
	return h.respond(msg, map[string]interface{}{
		"idpId":            profile.ID,
		"segmentIndex":     index,
		"profileStartTime": segment.StartTime,
		"profileBasalRate": segment.BasalRate,
		"profileCarbRatio": segment.CarbRatio,
		"profileTargetBG":  segment.TargetBG,
		"profileISF":       segment.ISF,
		"statusId":         31,
	})
}

// NewSetIDPSegmentHandler applies SetIDPSegmentRequest. Cargo: idpId, profileIndex, segmentIndex,
// operation, then startTime, basalRate (uint16), carbRatio (uint32), targetBG, ISF (uint16), all
// little-endian, and a status bitmask.
func NewSetIDPSegmentHandler(bridge *pumpx2.Bridge) *IDPHandler {
	return &IDPHandler{bridge: bridge, msgType: "SetIDPSegmentRequest", handle: handleSetIDPSegment}
}

func handleSetIDPSegment(h *IDPHandler, msg *pumpx2.ParsedMessage, ps *state.PumpState) (*Response, error) {
	cargo, err := requestCargo(msg, 17)
	if err != nil {
		return nil, fmt.Errorf("SetIDPSegmentRequest: %w", err)
	}
	segment := state.IDPSegment{
		StartTime: int(binary.LittleEndian.Uint16(cargo[4:6])),
		BasalRate: int(binary.LittleEndian.Uint16(cargo[6:8])),
		CarbRatio: binary.LittleEndian.Uint32(cargo[8:12]),
		TargetBG:  int(binary.LittleEndian.Uint16(cargo[12:14])),
		ISF:       int(binary.LittleEndian.Uint16(cargo[14:16])),
	}
	status := ps.ApplyIDPSegment(int(cargo[0]), state.IDPSegmentOperation(cargo[3]), int(cargo[2]), segment)
	// SetIDPSegmentResponse has only a raw byte[] constructor: status, then a byte pumpX2 names unknown.
	return h.respond(msg, map[string]interface{}{"raw": hex.EncodeToString([]byte{byte(status), 0})})
}

// NewCreateIDPHandler applies CreateIDPRequest. Cargo: name (17 bytes), then the first segment's
// carbRatio (uint32), startTime, basalRate, targetBG, ISF (uint16), insulinDuration (uint16), two
// bitmasks, sourceIdpId (0xFF for a new profile) and carbEntry.
func NewCreateIDPHandler(bridge *pumpx2.Bridge) *IDPHandler {
	return &IDPHandler{bridge: bridge, msgType: "CreateIDPRequest", handle: handleCreateIDP}
}

func handleCreateIDP(h *IDPHandler, msg *pumpx2.ParsedMessage, ps *state.PumpState) (*Response, error) {
	cargo, err := requestCargo(msg, 35)
	if err != nil {
		return nil, fmt.Errorf("CreateIDPRequest: %w", err)
	}
	name := strings.TrimRight(string(cargo[0:17]), "\x00")
	first := state.IDPSegment{
		CarbRatio: binary.LittleEndian.Uint32(cargo[17:21]),
		StartTime: int(binary.LittleEndian.Uint16(cargo[21:23])),
		BasalRate: int(binary.LittleEndian.Uint16(cargo[23:25])),
		TargetBG:  int(binary.LittleEndian.Uint16(cargo[25:27])),
		ISF:       int(binary.LittleEndian.Uint16(cargo[27:29])),
	}
	insulinDuration := int(binary.LittleEndian.Uint16(cargo[29:31]))
	sourceID := -1
	if cargo[33] != 0xFF {
		sourceID = int(cargo[33])
	}

	id, ok := ps.CreateIDP(name, first, insulinDuration, cargo[34] != 0, sourceID)
	status := state.IDPStatusOK
	if !ok {
		status = state.IDPStatusRejected
	}
	return h.respond(msg, map[string]interface{}{"status": status, "newIdpId": id})
}
