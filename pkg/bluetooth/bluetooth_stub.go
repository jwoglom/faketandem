//go:build !linux

package bluetooth

import (
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
)

// Ble represents the Bluetooth Low Energy device (stub for non-Linux platforms)
type Ble struct {
	// Data storage for each characteristic (for reads)
	charData    map[CharacteristicType][]byte
	charDataMtx sync.RWMutex

	// Handlers
	writeHandler      WriteHandler
	readHandler       ReadHandler
	connectionHandler ConnectionHandler
}

// New creates a new BLE device (stub for non-Linux platforms)
func New(adapterID string) (*Ble, error) {
	log.Warn("Bluetooth is only supported on Linux. Creating stub BLE instance.")
	return &Ble{
		charData: make(map[CharacteristicType][]byte),
	}, nil
}

// SetWriteHandler sets the callback for when data is written to any characteristic
func (b *Ble) SetWriteHandler(handler WriteHandler) {
	b.writeHandler = handler
}

// SetReadHandler sets the callback for when data is read from any characteristic
func (b *Ble) SetReadHandler(handler ReadHandler) {
	b.readHandler = handler
}

// SetConnectionHandler sets the callback for when a central connects or disconnects (no-op on non-Linux)
func (b *Ble) SetConnectionHandler(handler ConnectionHandler) {
	b.connectionHandler = handler
}

// SetCharacteristicData sets the data that will be returned when a characteristic is read
func (b *Ble) SetCharacteristicData(charType CharacteristicType, data []byte) {
	b.charDataMtx.Lock()
	defer b.charDataMtx.Unlock()
	b.charData[charType] = data
}

// Notify sends a notification on the specified characteristic (stub)
func (b *Ble) Notify(charType CharacteristicType, data []byte) error {
	log.Debugf("Notify called on non-Linux platform for %s (no-op)", charType)
	return fmt.Errorf("bluetooth not supported on this platform")
}

// IsConnected returns true if a central device is connected (always false on non-Linux)
func (b *Ble) IsConnected() bool {
	return false
}

// ShutdownConnection closes the connection with the central device (no-op)
func (b *Ble) ShutdownConnection() {
	log.Debug("ShutdownConnection called on non-Linux platform (no-op)")
}
