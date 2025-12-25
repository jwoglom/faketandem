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
	Connected      bool                       `json:"connected"`
	Characteristics map[string]string         `json:"characteristics"`
}

// BleEvent represents a BLE event sent to websocket clients
type BleEvent struct {
	Type           string `json:"type"`
	Characteristic string `json:"characteristic,omitempty"`
	Data           string `json:"data,omitempty"`
	Message        string `json:"message,omitempty"`
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
	http.ListenAndServe(":8080", nil)
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

func (s *Server) setupRoutes() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Pump Emulator API - Connect via WebSocket at /ws\n\nSettings API:\n  GET    /api/settings\n  GET    /api/settings/{messageType}\n  PUT    /api/settings/{messageType}\n  POST   /api/settings/{messageType}/reset")
	})
	http.Handle("/ws", s)
	http.HandleFunc("/api/settings", s.handleSettingsAPI)
	http.HandleFunc("/api/settings/", s.handleSettingsAPI)
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
		conn.Close()
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
func (s *Server) handleGetAllSettings(w http.ResponseWriter, r *http.Request) {
	configs := s.settingsManager.GetAllConfigs()

	if err := json.NewEncoder(w).Encode(configs); err != nil {
		log.Errorf("Failed to encode settings: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleGetSetting returns a specific settings configuration
func (s *Server) handleGetSetting(w http.ResponseWriter, r *http.Request, messageType string) {
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
	defer r.Body.Close()

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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Configuration updated for %s", messageType),
	})
}

// handleResetSetting resets the state for a settings configuration
func (s *Server) handleResetSetting(w http.ResponseWriter, r *http.Request, messageType string) {
	if messageType == "" {
		http.Error(w, "Message type is required", http.StatusBadRequest)
		return
	}

	if err := s.settingsManager.ResetState(messageType); err != nil {
		http.Error(w, fmt.Sprintf("Failed to reset state: %v", err), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("State reset for %s", messageType),
	})
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}
