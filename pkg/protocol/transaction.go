package protocol

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// PendingRequest represents a request waiting for a response
type PendingRequest struct {
	TxID         uint8
	MessageType  string
	Timestamp    time.Time
	ResponseChan chan []byte
	Timeout      time.Duration
}

// TransactionManager manages transaction IDs and pending requests
type TransactionManager struct {
	nextTxID    uint8
	mutex       sync.Mutex
	pendingReqs map[uint8]*PendingRequest
	timeout     time.Duration
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(defaultTimeout time.Duration) *TransactionManager {
	return &TransactionManager{
		nextTxID:    0,
		pendingReqs: make(map[uint8]*PendingRequest),
		timeout:     defaultTimeout,
	}
}

// AllocateTxID allocates a new transaction ID
func (tm *TransactionManager) AllocateTxID() uint8 {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	txID := tm.nextTxID
	tm.nextTxID++ // Will wrap around at 256

	log.Tracef("Allocated transaction ID: %d", txID)
	return txID
}

// GetNextTxID returns the next transaction ID without allocating it
func (tm *TransactionManager) GetNextTxID() uint8 {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	return tm.nextTxID
}

// SetNextTxID sets the next transaction ID (useful for synchronization)
func (tm *TransactionManager) SetNextTxID(txID uint8) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	log.Debugf("Set next transaction ID: %d (was %d)", txID, tm.nextTxID)
	tm.nextTxID = txID
}

// RegisterRequest registers a pending request
func (tm *TransactionManager) RegisterRequest(txID uint8, messageType string, responseChan chan []byte) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	if _, exists := tm.pendingReqs[txID]; exists {
		return fmt.Errorf("transaction ID %d already in use", txID)
	}

	req := &PendingRequest{
		TxID:         txID,
		MessageType:  messageType,
		Timestamp:    time.Now(),
		ResponseChan: responseChan,
		Timeout:      tm.timeout,
	}

	tm.pendingReqs[txID] = req

	log.Tracef("Registered pending request: txID=%d, messageType=%s", txID, messageType)

	// Start timeout goroutine
	go tm.handleTimeout(req)

	return nil
}

// handleTimeout handles request timeout
func (tm *TransactionManager) handleTimeout(req *PendingRequest) {
	timer := time.NewTimer(req.Timeout)
	defer timer.Stop()

	<-timer.C

	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// Check if request still exists
	if _, exists := tm.pendingReqs[req.TxID]; exists {
		log.Warnf("Request timed out: txID=%d, messageType=%s, age=%v",
			req.TxID, req.MessageType, time.Since(req.Timestamp))

		// Close channel to signal timeout
		close(req.ResponseChan)

		// Remove from pending
		delete(tm.pendingReqs, req.TxID)
	}
}

// CompleteRequest completes a pending request with a response
func (tm *TransactionManager) CompleteRequest(txID uint8, response []byte) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	req, exists := tm.pendingReqs[txID]
	if !exists {
		return fmt.Errorf("no pending request for transaction ID %d", txID)
	}

	log.Tracef("Completing request: txID=%d, messageType=%s, age=%v",
		txID, req.MessageType, time.Since(req.Timestamp))

	// Send response
	select {
	case req.ResponseChan <- response:
		// Response sent successfully
	default:
		log.Warnf("Response channel closed or full for txID=%d", txID)
	}

	// Remove from pending
	delete(tm.pendingReqs, txID)

	return nil
}

// CancelRequest cancels a pending request
func (tm *TransactionManager) CancelRequest(txID uint8) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	if req, exists := tm.pendingReqs[txID]; exists {
		log.Debugf("Canceling request: txID=%d, messageType=%s", txID, req.MessageType)
		close(req.ResponseChan)
		delete(tm.pendingReqs, txID)
	}
}

// GetPendingRequest returns a pending request
func (tm *TransactionManager) GetPendingRequest(txID uint8) (*PendingRequest, bool) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	req, exists := tm.pendingReqs[txID]
	return req, exists
}

// ClearAll clears all pending requests
func (tm *TransactionManager) ClearAll() {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	for txID, req := range tm.pendingReqs {
		close(req.ResponseChan)
		delete(tm.pendingReqs, txID)
	}

	log.Debug("Cleared all pending requests")
}

// GetStats returns statistics about the transaction manager
func (tm *TransactionManager) GetStats() map[string]interface{} {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	return map[string]interface{}{
		"nextTxID":       tm.nextTxID,
		"pendingCount":   len(tm.pendingReqs),
		"defaultTimeout": tm.timeout.String(),
	}
}
