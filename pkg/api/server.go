//nolint:revive // api is a standard package name for API servers
package api

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/settings"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// Server provides a WebSocket API for monitoring and controlling the pump emulator
type Server struct {
	http.Handler

	ble             *bluetooth.Ble
	conn            *websocket.Conn
	mtx             sync.Mutex
	settingsManager *settings.Manager

	// Callback for when a command is received from the websocket
	commandHandler CommandHandler
}

// CommandHandler is called when a command is received via websocket
type CommandHandler func(command string, params map[string]interface{})

// PumpState represents the current state of the pump emulator
type PumpState struct {
	Connected       bool              `json:"connected"`
	Characteristics map[string]string `json:"characteristics"`
}

// BleEvent represents a BLE event sent to websocket clients
type BleEvent struct {
	Type           string `json:"type"`
	Characteristic string `json:"characteristic,omitempty"`
	Data           string `json:"data,omitempty"`
	Message        string `json:"message,omitempty"`
	PairingCode    string `json:"pairing_code,omitempty"`
	Authenticated  *bool  `json:"authenticated,omitempty"`
}

// New creates a new API server
func New(ble *bluetooth.Ble) *Server {
	return &Server{
		ble: ble,
	}
}

// SetSettingsManager sets the settings manager for this server
func (s *Server) SetSettingsManager(manager *settings.Manager) {
	s.settingsManager = manager
}

// SetCommandHandler sets the callback for when commands are received
func (s *Server) SetCommandHandler(handler CommandHandler) {
	s.commandHandler = handler
}

// Start starts the HTTP/WebSocket server
func (s *Server) Start() {
	fmt.Println("Pump emulator web API listening on :8080")
	s.setupRoutes()
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

// SendEvent sends a BLE event to connected websocket clients
func (s *Server) SendEvent(event BleEvent) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.conn == nil {
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Errorf("Failed to marshal event: %v", err)
		return
	}

	if err := s.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Errorf("Failed to send websocket message: %v", err)
	}
}

// SendWriteEvent sends a notification that data was written to a characteristic
func (s *Server) SendWriteEvent(charType bluetooth.CharacteristicType, data []byte) {
	s.SendEvent(BleEvent{
		Type:           "write",
		Characteristic: charType.String(),
		Data:           hex.EncodeToString(data),
	})
}

// SendReadEvent sends a notification that data was read from a characteristic
func (s *Server) SendReadEvent(charType bluetooth.CharacteristicType, data []byte) {
	s.SendEvent(BleEvent{
		Type:           "read",
		Characteristic: charType.String(),
		Data:           hex.EncodeToString(data),
	})
}

// SendNotifyEvent sends a notification that data was sent via notification
func (s *Server) SendNotifyEvent(charType bluetooth.CharacteristicType, data []byte) {
	s.SendEvent(BleEvent{
		Type:           "notify",
		Characteristic: charType.String(),
		Data:           hex.EncodeToString(data),
	})
}

// SendConnectionEvent sends a connection status event
func (s *Server) SendConnectionEvent(connected bool) {
	eventType := "disconnected"
	if connected {
		eventType = "connected"
	}
	s.SendEvent(BleEvent{
		Type: eventType,
	})
}

// SendPairingState sends pairing code and authentication status to websocket clients
func (s *Server) SendPairingState(pairingCode string, authenticated bool) {
	authValue := authenticated
	s.SendEvent(BleEvent{
		Type:          "pairing_state",
		PairingCode:   pairingCode,
		Authenticated: &authValue,
	})
}

// SendPumpState sends the latest pump connection status to websocket clients
func (s *Server) SendPumpState() {
	s.sendState()
}

func (s *Server) setupRoutes() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprintf(w, "Pump Emulator API - Connect via WebSocket at /ws\n\nSettings API:\n  GET    /api/settings\n  GET    /api/settings/{messageType}\n  PUT    /api/settings/{messageType}\n  POST   /api/settings/{messageType}/reset\n\nBluetooth API:\n  GET    /api/bluetooth/discoverable\n  POST   /api/bluetooth/discoverable\n  GET    /api/bluetooth/allowpairing\n  POST   /api/bluetooth/allowpairing"); err != nil {
			log.Warnf("Failed to write response: %v", err)
		}
	})
	uiHandler := http.FileServer(http.Dir("ui"))
	http.Handle("/ui/", http.StripPrefix("/ui/", uiHandler))
	http.HandleFunc("/ui", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/", http.StatusMovedPermanently)
	})
	http.Handle("/ws", s)
	http.HandleFunc("/api/settings", s.handleSettingsAPI)
	http.HandleFunc("/api/settings/", s.handleSettingsAPI)
	http.HandleFunc("/api/bluetooth/discoverable", s.handleDiscoverableAPI)
	http.HandleFunc("/api/bluetooth/allowpairing", s.handleAllowPairingAPI)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Infof("WebSocket connection from: %s", r.Host)

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("WebSocket upgrade failed: %v", err)
		return
	}

	s.mtx.Lock()
	s.conn = ws
	s.mtx.Unlock()

	// Send initial state
	s.sendState()

	// Listen for messages
	s.reader(ws)
}

func (s *Server) sendState() {
	state := PumpState{
		Connected:       s.ble.IsConnected(),
		Characteristics: make(map[string]string),
	}

	data, err := json.Marshal(state)
	if err != nil {
		log.Errorf("Failed to marshal state: %v", err)
		return
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.conn != nil {
		if err := s.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Errorf("Failed to send state: %v", err)
		}
	}
}

func (s *Server) reader(conn *websocket.Conn) {
	defer func() {
		s.mtx.Lock()
		s.conn = nil
		s.mtx.Unlock()
		if err := conn.Close(); err != nil {
			log.Debugf("Error closing websocket: %v", err)
		}
	}()

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Infof("WebSocket read error: %v", err)
			return
		}
		log.Debugf("Received WebSocket message: %s", string(p))
		s.handleCommand(p)
	}
}

func (s *Server) handleCommand(data []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Errorf("Failed to parse command: %v", err)
		return
	}

	command, ok := msg["command"].(string)
	if !ok {
		log.Error("Command field missing or not a string")
		return
	}

	// Handle built-in commands
	switch command {
	case "getState":
		s.sendState()
		return
	case "notify":
		// Send a notification on a characteristic
		charName, _ := msg["characteristic"].(string)
		dataHex, _ := msg["data"].(string)
		s.handleNotifyCommand(charName, dataHex)
		return
	case "setCharacteristic":
		// Set data for a characteristic (for reads)
		charName, _ := msg["characteristic"].(string)
		dataHex, _ := msg["data"].(string)
		s.handleSetCharacteristicCommand(charName, dataHex)
		return
	}

	// Pass to custom handler
	if s.commandHandler != nil {
		s.commandHandler(command, msg)
	}
}

func (s *Server) handleNotifyCommand(charName string, dataHex string) {
	charType := s.parseCharacteristicName(charName)
	if charType < 0 {
		log.Errorf("Unknown characteristic: %s", charName)
		return
	}

	data, err := hex.DecodeString(dataHex)
	if err != nil {
		log.Errorf("Invalid hex data: %v", err)
		return
	}

	if err := s.ble.Notify(charType, data); err != nil {
		log.Errorf("Failed to send notification: %v", err)
	}
}

func (s *Server) handleSetCharacteristicCommand(charName string, dataHex string) {
	charType := s.parseCharacteristicName(charName)
	if charType < 0 {
		log.Errorf("Unknown characteristic: %s", charName)
		return
	}

	data, err := hex.DecodeString(dataHex)
	if err != nil {
		log.Errorf("Invalid hex data: %v", err)
		return
	}

	s.ble.SetCharacteristicData(charType, data)
}

func (s *Server) parseCharacteristicName(name string) bluetooth.CharacteristicType {
	switch name {
	case "CurrentStatus":
		return bluetooth.CharCurrentStatus
	case "QualifyingEvents":
		return bluetooth.CharQualifyingEvents
	case "HistoryLog":
		return bluetooth.CharHistoryLog
	case "Authorization":
		return bluetooth.CharAuthorization
	case "Control":
		return bluetooth.CharControl
	case "ControlStream":
		return bluetooth.CharControlStream
	default:
		return -1
	}
}

// handleSettingsAPI handles the RESTful settings API
func (s *Server) handleSettingsAPI(w http.ResponseWriter, r *http.Request) {
	if s.settingsManager == nil {
		http.Error(w, "Settings manager not initialized", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Parse the path to extract message type
	path := strings.TrimPrefix(r.URL.Path, "/api/settings")
	path = strings.Trim(path, "/")

	switch r.Method {
	case http.MethodGet:
		if path == "" {
			// GET /api/settings - list all configurations
			s.handleGetAllSettings(w, r)
		} else {
			// GET /api/settings/{messageType} - get specific configuration
			s.handleGetSetting(w, r, path)
		}

	case http.MethodPut:
		// PUT /api/settings/{messageType} - update configuration
		s.handleUpdateSetting(w, r, path)

	case http.MethodPost:
		// POST /api/settings/{messageType}/reset - reset state
		if strings.HasSuffix(path, "/reset") {
			messageType := strings.TrimSuffix(path, "/reset")
			s.handleResetSetting(w, r, messageType)
		} else {
			http.Error(w, "Invalid POST endpoint", http.StatusNotFound)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetAllSettings returns all registered settings configurations
//nolint:unparam // r is required by http.HandlerFunc interface
func (s *Server) handleGetAllSettings(w http.ResponseWriter, _ *http.Request) {
	configs := s.settingsManager.GetAllConfigs()

	if err := json.NewEncoder(w).Encode(configs); err != nil {
		log.Errorf("Failed to encode settings: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleGetSetting returns a specific settings configuration
//nolint:unparam // r is required by http.HandlerFunc interface
func (s *Server) handleGetSetting(w http.ResponseWriter, _ *http.Request, messageType string) {
	config, err := s.settingsManager.GetConfig(messageType)
	if err != nil {
		http.Error(w, fmt.Sprintf("Configuration not found: %s", err), http.StatusNotFound)
		return
	}

	if err := json.NewEncoder(w).Encode(config); err != nil {
		log.Errorf("Failed to encode setting: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleUpdateSetting updates a settings configuration
func (s *Server) handleUpdateSetting(w http.ResponseWriter, r *http.Request, messageType string) {
	if messageType == "" {
		http.Error(w, "Message type is required", http.StatusBadRequest)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Debugf("Error closing request body: %v", err)
		}
	}()

	// Parse the configuration
	var config settings.ResponseConfig
	if err := json.Unmarshal(body, &config); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse configuration: %v", err), http.StatusBadRequest)
		return
	}

	// Update the configuration
	if err := s.settingsManager.SetConfig(messageType, &config); err != nil {
		http.Error(w, fmt.Sprintf("Failed to set configuration: %v", err), http.StatusBadRequest)
		return
	}

	// Return the updated configuration
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Configuration updated for %s", messageType),
	}); err != nil {
		log.Errorf("Failed to encode update response: %v", err)
	}
}

// handleResetSetting resets the state for a settings configuration
//nolint:unparam // r is required by http.HandlerFunc interface
func (s *Server) handleResetSetting(w http.ResponseWriter, _ *http.Request, messageType string) {
	if messageType == "" {
		http.Error(w, "Message type is required", http.StatusBadRequest)
		return
	}

	if err := s.settingsManager.ResetState(messageType); err != nil {
		http.Error(w, fmt.Sprintf("Failed to reset state: %v", err), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("State reset for %s", messageType),
	}); err != nil {
		log.Errorf("Failed to encode reset response: %v", err)
	}
}

// handleDiscoverableAPI handles the Bluetooth discoverable API
func (s *Server) handleDiscoverableAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// GET /api/bluetooth/discoverable - get current discoverable state
		discoverable := s.ble.IsDiscoverable()
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"discoverable": discoverable,
		}); err != nil {
			log.Errorf("Failed to encode discoverable state: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}

	case http.MethodPost:
		// POST /api/bluetooth/discoverable - set discoverable state
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
			return
		}
		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Debugf("Error closing request body: %v", err)
			}
		}()

		var req struct {
			Discoverable bool `json:"discoverable"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse request: %v", err), http.StatusBadRequest)
			return
		}

		if err := s.ble.SetDiscoverable(req.Discoverable); err != nil {
			http.Error(w, fmt.Sprintf("Failed to set discoverable mode: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "success",
			"discoverable": req.Discoverable,
			"message":      fmt.Sprintf("Discoverable mode set to %v", req.Discoverable),
		}); err != nil {
			log.Errorf("Failed to encode discoverable response: %v", err)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAllowPairingAPI handles the Bluetooth allow pairing API
func (s *Server) handleAllowPairingAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// GET /api/bluetooth/allowpairing - get current allow pairing state
		allowPairing := s.ble.IsAllowPairing()
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"allowPairing": allowPairing,
		}); err != nil {
			log.Errorf("Failed to encode allow pairing state: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}

	case http.MethodPost:
		// POST /api/bluetooth/allowpairing - set allow pairing state
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
			return
		}
		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Debugf("Error closing request body: %v", err)
			}
		}()

		var req struct {
			AllowPairing bool `json:"allowPairing"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse request: %v", err), http.StatusBadRequest)
			return
		}

		if err := s.ble.SetAllowPairing(req.AllowPairing); err != nil {
			http.Error(w, fmt.Sprintf("Failed to set allow pairing mode: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "success",
			"allowPairing": req.AllowPairing,
			"message":      fmt.Sprintf("Allow pairing mode set to %v", req.AllowPairing),
		}); err != nil {
			log.Errorf("Failed to encode allow pairing response: %v", err)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}
