package protocol

import (
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/avereha/pod/pkg/bluetooth"
	log "github.com/sirupsen/logrus"
)

// PacketBuffer holds packets being assembled into a complete message
type PacketBuffer struct {
	CharType      bluetooth.CharacteristicType
	TxID          uint8
	Packets       [][]byte
	ExpectedCount int
	Timestamp     time.Time
}

// IsComplete returns true if all packets have been received
func (pb *PacketBuffer) IsComplete() bool {
	return len(pb.Packets) == pb.ExpectedCount
}

// AssembleMessage combines all packets into a single message
func (pb *PacketBuffer) AssembleMessage() ([]byte, error) {
	if !pb.IsComplete() {
		return nil, fmt.Errorf("cannot assemble incomplete message: have %d/%d packets",
			len(pb.Packets), pb.ExpectedCount)
	}

	// Calculate total size
	totalSize := 0
	for _, packet := range pb.Packets {
		payload, err := GetPacketPayload(packet)
		if err != nil {
			return nil, fmt.Errorf("invalid packet: %w", err)
		}
		totalSize += len(payload)
	}

	// Combine all payloads
	message := make([]byte, 0, totalSize)
	for _, packet := range pb.Packets {
		payload, _ := GetPacketPayload(packet)
		message = append(message, payload...)
	}

	log.Debugf("Assembled message: txID=%d, packets=%d, size=%d bytes, hex=%s",
		pb.TxID, len(pb.Packets), len(message), hex.EncodeToString(message))

	return message, nil
}

// Reassembler manages the reassembly of multi-packet messages
type Reassembler struct {
	buffers      map[string]*PacketBuffer
	mutex        sync.RWMutex
	timeout      time.Duration
	cleanupTimer *time.Ticker
	stopCleanup  chan bool
}

// NewReassembler creates a new packet reassembler
func NewReassembler(timeout time.Duration) *Reassembler {
	r := &Reassembler{
		buffers:      make(map[string]*PacketBuffer),
		timeout:      timeout,
		cleanupTimer: time.NewTicker(timeout / 2),
		stopCleanup:  make(chan bool),
	}

	// Start cleanup goroutine
	go r.cleanupLoop()

	return r
}

// Stop stops the reassembler and cleanup goroutine
func (r *Reassembler) Stop() {
	r.stopCleanup <- true
	r.cleanupTimer.Stop()
}

// cleanupLoop periodically removes old incomplete buffers
func (r *Reassembler) cleanupLoop() {
	for {
		select {
		case <-r.cleanupTimer.C:
			r.cleanupOldBuffers()
		case <-r.stopCleanup:
			return
		}
	}
}

// cleanupOldBuffers removes buffers that have timed out
func (r *Reassembler) cleanupOldBuffers() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	for key, buffer := range r.buffers {
		if now.Sub(buffer.Timestamp) > r.timeout {
			log.Warnf("Removing timed out buffer: %s (age: %v, packets: %d/%d)",
				key, now.Sub(buffer.Timestamp), len(buffer.Packets), buffer.ExpectedCount)
			delete(r.buffers, key)
		}
	}
}

// bufferKey creates a unique key for a packet buffer
func (r *Reassembler) bufferKey(charType bluetooth.CharacteristicType, txID uint8) string {
	return fmt.Sprintf("%s-%d", charType, txID)
}

// AddPacket adds a packet to the reassembler
// Returns (completeMessage, isComplete, error)
func (r *Reassembler) AddPacket(charType bluetooth.CharacteristicType, packet []byte) ([]byte, bool, error) {
	// Parse packet header
	header, err := ParsePacketHeader(packet)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse packet header: %w", err)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := r.bufferKey(charType, header.TxID)

	// Get or create buffer
	buffer, exists := r.buffers[key]
	if !exists {
		// First packet - calculate expected count
		expectedCount := int(header.RemainingPackets) + 1

		buffer = &PacketBuffer{
			CharType:      charType,
			TxID:          header.TxID,
			Packets:       make([][]byte, 0, expectedCount),
			ExpectedCount: expectedCount,
			Timestamp:     time.Now(),
		}
		r.buffers[key] = buffer

		log.Debugf("Created new packet buffer: key=%s, expectedPackets=%d", key, expectedCount)
	}

	// Add packet to buffer
	buffer.Packets = append(buffer.Packets, packet)
	buffer.Timestamp = time.Now() // Update timestamp

	log.Tracef("Added packet to buffer: key=%s, packets=%d/%d",
		key, len(buffer.Packets), buffer.ExpectedCount)

	// Check if complete
	if buffer.IsComplete() {
		// Assemble message
		message, err := buffer.AssembleMessage()
		if err != nil {
			delete(r.buffers, key) // Remove invalid buffer
			return nil, false, fmt.Errorf("failed to assemble message: %w", err)
		}

		// Remove buffer
		delete(r.buffers, key)

		return message, true, nil
	}

	return nil, false, nil
}

// Reset clears all buffers
func (r *Reassembler) Reset() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.buffers = make(map[string]*PacketBuffer)
	log.Debug("Reassembler buffers cleared")
}

// GetStats returns statistics about the reassembler
func (r *Reassembler) GetStats() map[string]interface{} {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return map[string]interface{}{
		"activeBuffers": len(r.buffers),
		"timeout":       r.timeout.String(),
	}
}
