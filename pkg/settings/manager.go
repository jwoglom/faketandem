package settings

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// ResponseMode defines how a message response behaves
type ResponseMode string

const (
	// ModeConstant always returns the same configured value
	ModeConstant ResponseMode = "constant"

	// ModeIncremental cycles through an array of values
	ModeIncremental ResponseMode = "incremental"

	// ModeTimeBased returns values based on elapsed time since first request
	ModeTimeBased ResponseMode = "time_based"
)

// ResponseConfig defines the configuration for a message type's response
type ResponseConfig struct {
	// Mode determines the response behavior
	Mode ResponseMode `json:"mode"`

	// Value is used for ModeConstant - the single response value
	Value map[string]interface{} `json:"value,omitempty"`

	// Values is used for ModeIncremental and ModeTimeBased - array of possible responses
	Values []map[string]interface{} `json:"values,omitempty"`

	// TimingSeconds is used for ModeTimeBased - when each value becomes active (in seconds from start)
	// Must match length of Values array
	TimingSeconds []int `json:"timing_seconds,omitempty"`

	// CurrentIndex tracks the current position (for ModeIncremental)
	CurrentIndex int `json:"current_index,omitempty"`

	// StartTime tracks when the first request was made (for ModeTimeBased)
	StartTime time.Time `json:"start_time,omitempty"`
}

// Manager manages configurable settings responses
type Manager struct {
	configs map[string]*ResponseConfig
	mutex   sync.RWMutex
}

// NewManager creates a new settings manager
func NewManager() *Manager {
	return &Manager{
		configs: make(map[string]*ResponseConfig),
	}
}

// RegisterDefault registers a default configuration for a message type
func (m *Manager) RegisterDefault(messageType string, config *ResponseConfig) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Validate the configuration
	if err := m.validateConfig(config); err != nil {
		return fmt.Errorf("invalid config for %s: %w", messageType, err)
	}

	m.configs[messageType] = config
	log.Infof("Registered settings for %s: mode=%s", messageType, config.Mode)

	return nil
}

// GetResponse returns the appropriate response for a message type
func (m *Manager) GetResponse(messageType string) (map[string]interface{}, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	config, exists := m.configs[messageType]
	if !exists {
		return nil, fmt.Errorf("no configuration for message type: %s", messageType)
	}

	switch config.Mode {
	case ModeConstant:
		return m.getConstantResponse(config)

	case ModeIncremental:
		return m.getIncrementalResponse(config)

	case ModeTimeBased:
		return m.getTimeBasedResponse(config)

	default:
		return nil, fmt.Errorf("unknown response mode: %s", config.Mode)
	}
}

// getConstantResponse returns the constant value
func (m *Manager) getConstantResponse(config *ResponseConfig) (map[string]interface{}, error) {
	if config.Value == nil {
		return nil, fmt.Errorf("constant mode requires 'value' field")
	}
	return config.Value, nil
}

// getIncrementalResponse returns the next value in the array and advances the index
func (m *Manager) getIncrementalResponse(config *ResponseConfig) (map[string]interface{}, error) {
	if len(config.Values) == 0 {
		return nil, fmt.Errorf("incremental mode requires 'values' array")
	}

	// Get current value
	value := config.Values[config.CurrentIndex]

	// Advance to next index (wrap around)
	config.CurrentIndex = (config.CurrentIndex + 1) % len(config.Values)

	log.Debugf("Incremental response: index=%d/%d", config.CurrentIndex, len(config.Values))

	return value, nil
}

// getTimeBasedResponse returns the appropriate value based on elapsed time
func (m *Manager) getTimeBasedResponse(config *ResponseConfig) (map[string]interface{}, error) {
	if len(config.Values) == 0 {
		return nil, fmt.Errorf("time_based mode requires 'values' array")
	}

	if len(config.TimingSeconds) != len(config.Values) {
		return nil, fmt.Errorf("time_based mode requires timing_seconds array matching values length")
	}

	// Initialize start time on first request
	if config.StartTime.IsZero() {
		config.StartTime = time.Now()
	}

	// Calculate elapsed time
	elapsedSeconds := int(time.Since(config.StartTime).Seconds())

	// Find the appropriate value based on elapsed time
	valueIndex := 0
	for i, timing := range config.TimingSeconds {
		if elapsedSeconds >= timing {
			valueIndex = i
		} else {
			break
		}
	}

	log.Debugf("Time-based response: elapsed=%ds, using index=%d (timing=%ds)",
		elapsedSeconds, valueIndex, config.TimingSeconds[valueIndex])

	return config.Values[valueIndex], nil
}

// SetConfig updates the configuration for a message type
func (m *Manager) SetConfig(messageType string, config *ResponseConfig) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if err := m.validateConfig(config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Reset state when config changes
	config.CurrentIndex = 0
	config.StartTime = time.Time{}

	m.configs[messageType] = config
	log.Infof("Updated settings for %s: mode=%s", messageType, config.Mode)

	return nil
}

// GetConfig retrieves the current configuration for a message type
func (m *Manager) GetConfig(messageType string) (*ResponseConfig, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	config, exists := m.configs[messageType]
	if !exists {
		return nil, fmt.Errorf("no configuration for message type: %s", messageType)
	}

	// Return a copy to prevent external modification
	configCopy := *config
	return &configCopy, nil
}

// GetAllConfigs returns all registered configurations
func (m *Manager) GetAllConfigs() map[string]*ResponseConfig {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return copies to prevent external modification
	result := make(map[string]*ResponseConfig)
	for msgType, config := range m.configs {
		configCopy := *config
		result[msgType] = &configCopy
	}

	return result
}

// ResetState resets the state (current index, start time) for a message type
func (m *Manager) ResetState(messageType string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	config, exists := m.configs[messageType]
	if !exists {
		return fmt.Errorf("no configuration for message type: %s", messageType)
	}

	config.CurrentIndex = 0
	config.StartTime = time.Time{}

	log.Infof("Reset state for %s", messageType)

	return nil
}

// validateConfig validates a response configuration
func (m *Manager) validateConfig(config *ResponseConfig) error {
	switch config.Mode {
	case ModeConstant:
		if config.Value == nil {
			return fmt.Errorf("constant mode requires 'value' field")
		}

	case ModeIncremental:
		if len(config.Values) == 0 {
			return fmt.Errorf("incremental mode requires non-empty 'values' array")
		}

	case ModeTimeBased:
		if len(config.Values) == 0 {
			return fmt.Errorf("time_based mode requires non-empty 'values' array")
		}
		if len(config.TimingSeconds) != len(config.Values) {
			return fmt.Errorf("time_based mode requires timing_seconds array matching values length (got %d timings, %d values)",
				len(config.TimingSeconds), len(config.Values))
		}
		// Verify timings are in ascending order
		for i := 1; i < len(config.TimingSeconds); i++ {
			if config.TimingSeconds[i] < config.TimingSeconds[i-1] {
				return fmt.Errorf("timing_seconds must be in ascending order")
			}
		}

	default:
		return fmt.Errorf("unknown response mode: %s (valid modes: constant, incremental, time_based)", config.Mode)
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling to handle time formatting
func (c *ResponseConfig) MarshalJSON() ([]byte, error) {
	type Alias ResponseConfig
	return json.Marshal(&struct {
		StartTime string `json:"start_time,omitempty"`
		*Alias
	}{
		StartTime: func() string {
			if c.StartTime.IsZero() {
				return ""
			}
			return c.StartTime.Format(time.RFC3339)
		}(),
		Alias: (*Alias)(c),
	})
}
