package handler

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// minStreamDelay keeps a stream message the pump times from a request behind that request's own
// response, which goes out after its handler returns.
const minStreamDelay = 50 * time.Millisecond

// The pump's in-progress states that pumpX2 does not name. The emulator streams 0 before the state
// pumpX2 knows; the ID is not confirmed on firmware.
const (
	changeCartridgeStateRetracting int32 = 0
	fillCannulaStateFilling        int32 = 0
)

// detectionSteps is how many times the pump reports a cartridge check's progress after 0%.
const detectionSteps = 5

type timerHandle interface{ Stop() bool }

type scheduleFunc func(time.Duration, func()) timerHandle

func realSchedule(d time.Duration, f func()) timerHandle { return time.AfterFunc(d, f) }

// procedureTimers runs the pump's timed procedure steps, one pending step per name, so a later
// step replaces or cancels an earlier one.
type procedureTimers struct {
	mu       sync.Mutex
	schedule scheduleFunc
	pending  map[string]timerHandle
	gen      map[string]uint64
}

func newProcedureTimers() *procedureTimers {
	return &procedureTimers{schedule: realSchedule, pending: map[string]timerHandle{}, gen: map[string]uint64{}}
}

func (p *procedureTimers) start(name string, d time.Duration, f func()) {
	if d < minStreamDelay {
		d = minStreamDelay
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if t := p.pending[name]; t != nil {
		t.Stop()
	}
	p.gen[name]++
	gen := p.gen[name]
	p.pending[name] = p.schedule(d, func() {
		p.mu.Lock()
		current := p.gen[name] == gen
		if current {
			delete(p.pending, name)
		}
		p.mu.Unlock()
		if current {
			f()
		}
	})
}

func (p *procedureTimers) stop(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if t := p.pending[name]; t != nil {
		t.Stop()
		delete(p.pending, name)
	}
	p.gen[name]++
}

const (
	procedureReady       = "change_cartridge_ready"
	procedureDetect      = "cartridge_detection"
	procedureFillCannula = "fill_cannula"
	procedureMaxFill     = "fill_tubing_max"
)

type streamBuilder func(authKey []byte, timeSinceReset uint32) (*protocol.NativeMessage, error)

// buildStream signs a CONTROL_STREAM message with the session key, or fails before the app has
// authenticated.
func buildStream(ps *state.PumpState, build streamBuilder) (*protocol.NativeMessage, error) {
	authKey := ps.GetAuthKey()
	if len(authKey) == 0 {
		return nil, fmt.Errorf("no authenticated app to stream to")
	}
	return build(authKey, ps.GetTimeSinceReset())
}

// stream sends a CONTROL_STREAM message now and records it, with note, in the request log.
func (r *Router) stream(ps *state.PumpState, note string, build streamBuilder) error {
	msg, err := buildStream(ps, build)
	if err != nil {
		return err
	}
	if r.ble == nil {
		return nil
	}
	log.Infof("Streaming %s under txID=%d (%s)", msg.MessageType, msg.TxID, note)
	sent, dropped, sendErr := r.sendNative(msg)
	if r.requestLog != nil && !dropped {
		if sendErr != nil {
			note += "; send error: " + sendErr.Error()
		}
		r.requestLog.RecordResponse(msg.Characteristic.String(), msg.MessageType, int(msg.Opcode), int(msg.TxID),
			msg.PacketsHex(), sent, "", note)
	}
	return sendErr
}

// streamLater builds a stream message to go out after a request's response, or nil before the
// app has authenticated.
func streamLater(ps *state.PumpState, build streamBuilder) []*protocol.NativeMessage {
	msg, err := buildStream(ps, build)
	if err != nil {
		log.Debugf("Not streaming: %v", err)
		return nil
	}
	return []*protocol.NativeMessage{msg}
}

func (r *Router) notifyRemainingInsulin() {
	if r.ble == nil || r.qeNotifier == nil {
		return
	}
	if err := r.qeNotifier.NotifyBitmask(QERemainingInsulin); err != nil {
		log.Warnf("Failed to notify remaining insulin: %v", err)
	}
}

func secondsDuration(s float64) time.Duration {
	return time.Duration(math.Round(s * float64(time.Second)))
}

// startChangeCartridge times the retraction of a change cartridge mode the pump just entered,
// and returns the stream message to send after the response.
func (r *Router) startChangeCartridge(ps *state.PumpState, txID uint8) []*protocol.NativeMessage {
	r.procs.stop(procedureDetect)
	r.procs.start(procedureReady, secondsDuration(ps.CartridgeSettings().ReadySeconds), func() {
		if err := r.readyToChange(ps); err != nil {
			log.Warnf("Could not stream readyToChange: %v", err)
		}
	})
	return streamLater(ps, func(key []byte, tsr uint32) (*protocol.NativeMessage, error) {
		return protocol.BuildEnterChangeCartridgeModeStateStreamResponse(txID, changeCartridgeStateRetracting, key, tsr)
	})
}

// CartridgeReady finishes the retraction now: the pump streams readyToChange, after which a
// cartridge can be put in. It refuses outside change cartridge mode.
func (r *Router) CartridgeReady() error {
	r.procs.stop(procedureReady)
	return r.readyToChange(r.pumpState)
}

func (r *Router) readyToChange(ps *state.PumpState) error {
	txID, err := ps.MarkReadyToChange()
	if err != nil {
		return err
	}
	return r.stream(ps, "ready to change", func(key []byte, tsr uint32) (*protocol.NativeMessage, error) {
		return protocol.BuildEnterChangeCartridgeModeStateStreamResponse(txID, 2, key, tsr)
	})
}

// InsertCartridge puts a new cartridge of units into a pump ready for it. With detect_on_insert
// the pump checks it at once, under the EnterChangeCartridgeModeRequest's txID; otherwise at the
// ExitChangeCartridgeModeRequest.
func (r *Router) InsertCartridge(units float64) error {
	ps := r.pumpState
	if ps.CartridgeSettings().DetectOnInsert && len(ps.GetAuthKey()) == 0 {
		return fmt.Errorf("no authenticated app to stream the cartridge check to")
	}
	detect, txID, err := ps.InsertCartridge(units)
	if err != nil || !detect {
		return err
	}
	return r.detectionStep(ps, txID, 0)
}

// exitChangeCartridge leaves change cartridge mode and, when the pump takes the exit as a new
// cartridge being in, starts checking it and returns the first progress message.
func (r *Router) exitChangeCartridge(ps *state.PumpState, txID uint8) (bool, []*protocol.NativeMessage) {
	accepted, detect := ps.ExitChangeCartridge()
	if !accepted {
		return false, nil
	}
	r.procs.stop(procedureReady)
	if !detect {
		return true, nil
	}
	r.scheduleDetectionStep(ps, txID, 1)
	return true, streamLater(ps, func(key []byte, tsr uint32) (*protocol.NativeMessage, error) {
		return protocol.BuildDetectingCartridgeStateStreamResponse(txID, 0, key, tsr)
	})
}

func (r *Router) scheduleDetectionStep(ps *state.PumpState, txID uint8, step int) {
	interval := secondsDuration(ps.CartridgeSettings().DetectSeconds / detectionSteps)
	r.procs.start(procedureDetect, interval, func() {
		if err := r.detectionStep(ps, txID, step); err != nil {
			log.Warnf("Could not stream the cartridge check: %v", err)
		}
	})
}

// detectionStep reports the check at step of detectionSteps. The last one fills the reservoir
// before it streams 100%, so a driver that reads the pump on it finds the new cartridge.
func (r *Router) detectionStep(ps *state.PumpState, txID uint8, step int) error {
	if step < detectionSteps {
		r.scheduleDetectionStep(ps, txID, step+1)
	} else if units, ok := ps.FinishCartridgeDetection(); ok {
		log.Infof("Cartridge checked: %.1f U", units)
		r.notifyRemainingInsulin()
	}
	percent := int32(step * 100 / detectionSteps)
	return r.stream(ps, fmt.Sprintf("cartridge check %d%%", percent), func(key []byte, tsr uint32) (*protocol.NativeMessage, error) {
		return protocol.BuildDetectingCartridgeStateStreamResponse(txID, percent, key, tsr)
	})
}

// exitFillTubing leaves fill tubing mode and returns the stream message the pump sends after the
// response: TUBING_FILLED when the tubing took insulin, NOT_COMPLETE otherwise.
func (r *Router) exitFillTubing(ps *state.PumpState, txID uint8) (bool, []*protocol.NativeMessage) {
	accepted, units := ps.ExitFillTubing()
	if !accepted {
		return false, nil
	}
	r.procs.stop(procedureMaxFill)
	stateID := int32(0)
	if units > 0 {
		stateID = 1
		r.notifyRemainingInsulin()
	}
	return true, streamLater(ps, func(key []byte, tsr uint32) (*protocol.NativeMessage, error) {
		return protocol.BuildExitFillTubingModeStateStreamResponse(txID, stateID, key, tsr)
	})
}

// startFillCannula times a cannula prime of units the pump just accepted, and returns the
// stream message to send after the response.
func (r *Router) startFillCannula(ps *state.PumpState, txID uint8, units float64) []*protocol.NativeMessage {
	duration := secondsDuration(units / ps.GetBolusRate())
	r.procs.start(procedureFillCannula, duration, func() {
		if !ps.FinishFillCannula(units) {
			return
		}
		r.notifyRemainingInsulin()
		if ps.CartridgeSettings().FillCannulaStreamLost {
			log.Infof("Cannula filled with %.2f U; its stream is lost", units)
			return
		}
		if err := r.stream(ps, "cannula filled", func(key []byte, tsr uint32) (*protocol.NativeMessage, error) {
			return protocol.BuildFillCannulaStateStreamResponse(txID, 2, key, tsr)
		}); err != nil {
			log.Warnf("Could not stream cannulaFilled: %v", err)
		}
	})
	return streamLater(ps, func(key []byte, tsr uint32) (*protocol.NativeMessage, error) {
		return protocol.BuildFillCannulaStateStreamResponse(txID, fillCannulaStateFilling, key, tsr)
	})
}
