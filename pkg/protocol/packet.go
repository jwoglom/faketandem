package protocol

import (
	"encoding/hex"
	"fmt"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	log "github.com/sirupsen/logrus"
)

// PacketHeader represents the header of a packet
type PacketHeader struct {
	RemainingPackets uint8
	TxID             uint8
}

// ParsePacketHeader parses the packet header from data
func ParsePacketHeader(data []byte) (*PacketHeader, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("packet too short for header: %d bytes", len(data))
	}

	return &PacketHeader{
		RemainingPackets: data[0],
		TxID:             data[1],
	}, nil
}

// GetPacketPayload extracts the payload (data after header) from a packet
func GetPacketPayload(data []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("packet too short: %d bytes", len(data))
	}

	return data[2:], nil
}

// GetChunkSize returns the chunk size for a given characteristic
func GetChunkSize(charType bluetooth.CharacteristicType) int {
	switch charType {
	case bluetooth.CharAuthorization:
		return 40 // Authorization uses 40-byte chunks
	case bluetooth.CharControl:
		return 18 // Control uses 18-byte chunks
	case bluetooth.CharControlStream:
		return 18 // ControlStream uses 18-byte chunks
	default:
		return 18 // Default to 18-byte chunks
	}
}

// AssemblePackets takes a full message and breaks it into packets
// Returns a slice of packets ready to send
func AssemblePackets(charType bluetooth.CharacteristicType, txID uint8, message []byte) ([][]byte, error) {
	chunkSize := GetChunkSize(charType)

	// Calculate how many bytes we can fit in each packet (minus 2-byte header)
	payloadSize := chunkSize - 2

	// Calculate number of packets needed
	totalPackets := (len(message) + payloadSize - 1) / payloadSize

	if totalPackets > 255 {
		return nil, fmt.Errorf("message too large: would require %d packets", totalPackets)
	}

	packets := make([][]byte, 0, totalPackets)

	for i := 0; i < totalPackets; i++ {
		// Calculate payload for this packet
		start := i * payloadSize
		end := start + payloadSize
		if end > len(message) {
			end = len(message)
		}

		payload := message[start:end]

		// Create packet with header
		packet := make([]byte, 2+len(payload))
		packet[0] = uint8(totalPackets - i - 1) // Remaining packets after this one
		packet[1] = txID
		copy(packet[2:], payload)

		packets = append(packets, packet)

		log.Tracef("Created packet %d/%d: remaining=%d, txID=%d, size=%d",
			i+1, totalPackets, packet[0], packet[1], len(packet))
	}

	return packets, nil
}

// LogPacket logs a packet in a readable format
func LogPacket(direction string, charType bluetooth.CharacteristicType, data []byte) {
	if len(data) < 2 {
		log.Warnf("%s packet on %s too short: %s", direction, charType, hex.EncodeToString(data))
		return
	}

	header, _ := ParsePacketHeader(data)
	payload, _ := GetPacketPayload(data)

	log.Debugf("%s packet on %s: remaining=%d, txID=%d, payload=%s",
		direction, charType, header.RemainingPackets, header.TxID, hex.EncodeToString(payload))
}
