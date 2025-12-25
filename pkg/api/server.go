package api

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// Server provides a WebSocket API for monitoring and controlling the pump emulator
type Server struct {
	http.Handler

	ble  *bluetooth.Ble
	conn *websocket.Conn
	mtx  sync.Mutex

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
		fmt.Fprintf(w, "Pump Emulator API - Connect via WebSocket at /ws")
	})
	http.Handle("/ws", s)
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

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}
