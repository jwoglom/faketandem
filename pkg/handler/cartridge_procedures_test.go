package handler

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"
)

type manualTimer struct {
	d       time.Duration
	f       func()
	stopped bool
}

func (m *manualTimer) Stop() bool {
	was := !m.stopped
	m.stopped = true
	return was
}

// manualTimers runs the router's procedure steps only when the test says so.
type manualTimers struct {
	mu    sync.Mutex
	queue []*manualTimer
}

func (m *manualTimers) schedule(d time.Duration, f func()) timerHandle {
	m.mu.Lock()
	defer m.mu.Unlock()
	timer := &manualTimer{d: d, f: f}
	m.queue = append(m.queue, timer)
	return timer
}

// pending is the delays of the steps waiting to run.
func (m *manualTimers) pending() []time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []time.Duration
	for _, timer := range m.queue {
		if !timer.stopped {
			out = append(out, timer.d)
		}
	}
	return out
}

// drain runs every waiting step, and the ones they schedule, in order.
func (m *manualTimers) drain(t *testing.T) {
	t.Helper()
	for i := 0; ; i++ {
		if i > 100 {
			t.Fatal("procedure steps keep scheduling more")
		}
		m.mu.Lock()
		var next *manualTimer
		for len(m.queue) > 0 && next == nil {
			next, m.queue = m.queue[0], m.queue[1:]
			if next.stopped {
				next = nil
			}
		}
		m.mu.Unlock()
		if next == nil {
			return
		}
		next.f()
	}
}

type procedureRig struct {
	t         *testing.T
	router    *Router
	ps        *state.PumpState
	transport *recordingTransport
	timers    *manualTimers
	clock     *state.ManualClock
}

func newProcedureRig(t *testing.T) *procedureRig {
	t.Helper()
	ps := state.NewPumpState()
	clock := state.NewFrozenClock(time.Date(2024, time.March, 5, 12, 0, 0, 0, time.UTC))
	ps.SetClock(clock)
	ps.SetPumpingSuspended(true)
	ps.Lock()
	ps.AuthKey = []byte("0123456789abcdef0123456789abcdef")
	ps.IsAuthenticated = true
	ps.Unlock()
	transport := &recordingTransport{}
	router := NewRouter(testBridge(t), ps, transport, protocol.NewTransactionManager(time.Second), "go", "", "jar", "", "java", "")
	timers := &manualTimers{}
	router.procs.schedule = timers.schedule
	return &procedureRig{t: t, router: router, ps: ps, transport: transport, timers: timers, clock: clock}
}

// route sends a request through the router as the app would and returns the pump's answer.
func (r *procedureRig) route(char bluetooth.CharacteristicType, name string, txID int, cargo map[string]interface{}) *pumpx2.ParsedMessage {
	r.t.Helper()
	before := len(r.transport.notifications)
	if cargo == nil {
		cargo = map[string]interface{}{}
	}
	if err := r.router.RouteMessage(char, &pumpx2.ParsedMessage{TxID: txID, MessageType: name, Cargo: cargo}); err != nil {
		r.t.Fatalf("%s: %v", name, err)
	}
	var fragments []string
	for _, n := range r.transport.notifications[before:] {
		if n.charType == char {
			fragments = append(fragments, hex.EncodeToString(n.data))
		}
	}
	reply, err := r.router.bridge.ParseMessage(char, fragments)
	if err != nil {
		r.t.Fatalf("the answer to %s does not parse: %v", name, err)
	}
	return reply
}

func status(t *testing.T, m *pumpx2.ParsedMessage) int {
	t.Helper()
	var s int
	if _, err := fmt.Sscan(fmt.Sprint(m.Cargo["status"]), &s); err != nil {
		t.Fatalf("%s has no numeric status: %v", m.MessageType, m.Cargo)
	}
	return s
}

type streamedMessage struct {
	Opcode uint8
	TxID   uint8
	Cargo  string
}

// streamed reassembles every CONTROL_STREAM message, checks each is signed with the session key,
// and returns it without its signature.
func (r *procedureRig) streamed() []streamedMessage {
	r.t.Helper()
	var out []streamedMessage
	var body []byte
	for _, n := range r.transport.notifications {
		if n.charType != bluetooth.CharControlStream {
			continue
		}
		body = append(body, n.data[2:]...)
		if n.data[0] != 0 {
			continue
		}
		if _, ok, err := protocol.VerifySignedMessage(body, r.ps.GetAuthKey()); err != nil || !ok {
			r.t.Fatalf("stream message %x is not signed with the session key (err %v)", body, err)
		}
		parsed, err := protocol.ParseMessageBody(body)
		if err != nil {
			r.t.Fatal(err)
		}
		cargo := parsed.Payload[:len(parsed.Payload)-protocol.SignedTrailerLength]
		out = append(out, streamedMessage{Opcode: parsed.Opcode, TxID: parsed.TxID, Cargo: hex.EncodeToString(cargo)})
		body = nil
	}
	return out
}

func (r *procedureRig) qualifyingEvents() []uint32 {
	var out []uint32
	for _, n := range r.transport.notifications {
		if n.charType == bluetooth.CharQualifyingEvents {
			out = append(out, binary.LittleEndian.Uint32(n.data))
		}
	}
	return out
}

func (r *procedureRig) lastRecord(name string) state.HistoryLogEntry {
	r.t.Helper()
	return lastRecordOfType(r.t, r.ps, name)
}

func detecting(txID uint8, percents ...int) []streamedMessage {
	var out []streamedMessage
	for _, p := range percents {
		out = append(out, streamedMessage{protocol.OpcodeDetectingCartridgeStateStreamResponse, txID, hex.EncodeToString([]byte{byte(p), 0})})
	}
	return out
}

// changeCartridgeState is a state streamed under the EnterChangeCartridgeModeRequest every test sends as 0x21.
func changeCartridgeState(id byte) streamedMessage {
	return streamedMessage{protocol.OpcodeEnterChangeCartridgeModeStateStreamResponse, 0x21, hex.EncodeToString([]byte{id})}
}

func (r *procedureRig) enterChangeCartridge(txID int) {
	r.t.Helper()
	if s := status(r.t, r.route(bluetooth.CharControl, "EnterChangeCartridgeModeRequest", txID, nil)); s != 0 {
		r.t.Fatalf("EnterChangeCartridgeModeRequest answered status %d", s)
	}
}

func TestChangeCartridgeStreamsRetractingThenReadyUnderTheEnterTxID(t *testing.T) {
	rig := newProcedureRig(t)
	rig.enterChangeCartridge(0x21)

	if got, want := rig.streamed(), []streamedMessage{changeCartridgeState(0)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("streamed %v at entry, want the retracting state", got)
	}
	if got := rig.timers.pending(); !reflect.DeepEqual(got, []time.Duration{5 * time.Second}) {
		t.Errorf("pending steps %v, want readyToChange after the default 5 s", got)
	}
	rig.timers.drain(t)

	want := []streamedMessage{changeCartridgeState(0), changeCartridgeState(2)}
	if got := rig.streamed(); !reflect.DeepEqual(got, want) {
		t.Errorf("streamed %v, want %v", got, want)
	}
	if !rig.ps.CartridgeSnapshotUnlocked().ReadyToChange {
		t.Error("the pump does not read ready to change")
	}
}

func TestCartridgeReadyStreamsAtOnceAndCancelsTheTimedOne(t *testing.T) {
	rig := newProcedureRig(t)
	if err := rig.router.CartridgeReady(); err == nil {
		t.Error("readyToChange streamed outside change cartridge mode")
	}
	rig.enterChangeCartridge(0x21)
	if err := rig.router.CartridgeReady(); err != nil {
		t.Fatal(err)
	}
	rig.timers.drain(t)
	want := []streamedMessage{changeCartridgeState(0), changeCartridgeState(2)}
	if got := rig.streamed(); !reflect.DeepEqual(got, want) {
		t.Errorf("streamed %v, want %v", got, want)
	}
}

func TestExitChangeCartridgeChecksTheNewCartridge(t *testing.T) {
	rig := newProcedureRig(t)
	rig.ps.SetReservoirLevel(12)
	rig.enterChangeCartridge(0x21)
	if err := rig.router.InsertCartridge(300); err == nil {
		t.Fatal("a cartridge went in before the pump retracted")
	}
	rig.timers.drain(t)
	if err := rig.router.InsertCartridge(300); err != nil {
		t.Fatal(err)
	}
	if got := len(rig.streamed()); got != 2 {
		t.Fatalf("inserting streamed; %d messages", got)
	}

	if s := status(t, rig.route(bluetooth.CharControl, "ExitChangeCartridgeModeRequest", 0x22, nil)); s != 0 {
		t.Fatalf("ExitChangeCartridgeModeRequest answered status %d", s)
	}
	if got := rig.ps.GetReservoirLevel(); got != 12 {
		t.Errorf("reservoir %v before the check finished", got)
	}
	rig.timers.drain(t)

	got := rig.streamed()[2:]
	if want := detecting(0x22, 0, 20, 40, 60, 80, 100); !reflect.DeepEqual(got, want) {
		t.Errorf("streamed %v, want %v", got, want)
	}
	if level := rig.ps.GetReservoirLevel(); level != 300 {
		t.Errorf("reservoir %v, want the 300 U put in", level)
	}
	if record := rig.lastRecord("CartridgeFilled"); record.Data["insulinActual"] != 300.0 || record.Data["insulinDisplay"] != 300 {
		t.Errorf("CartridgeFilled %v", record.Data)
	}
	if got := rig.qualifyingEvents(); !reflect.DeepEqual(got, []uint32{QERemainingInsulin}) {
		t.Errorf("qualifying events %v, want remainingInsulin", got)
	}

	load := rig.route(bluetooth.CharCurrentStatus, "LoadStatusRequest", 0x23, nil)
	if fmt.Sprint(load.Cargo["isLoadingActive"]) != "true" || fmt.Sprint(load.Cargo["loadStateId"]) != "1" {
		t.Errorf("LoadStatusResponse %v, want LOAD_CARTRIDGE active", load.Cargo)
	}
	if s := status(t, rig.route(bluetooth.CharControl, "EnterChangeCartridgeModeRequest", 0x24, nil)); s != 1 {
		t.Errorf("change cartridge answered %d with the new cartridge waiting for its tubing fill, want 1", s)
	}
}

func TestDetectOnInsertChecksTheCartridgeUnderTheEnterTxID(t *testing.T) {
	rig := newProcedureRig(t)
	rig.ps.UpdateCartridgeSettings(func(s *state.CartridgeSettings) { s.DetectOnInsert = true })
	rig.enterChangeCartridge(0x21)
	rig.timers.drain(t)
	if err := rig.router.InsertCartridge(200); err != nil {
		t.Fatal(err)
	}
	rig.timers.drain(t)
	if s := status(t, rig.route(bluetooth.CharControl, "ExitChangeCartridgeModeRequest", 0x22, nil)); s != 0 {
		t.Fatalf("exit answered %d", s)
	}
	rig.timers.drain(t)

	if got, want := rig.streamed()[2:], detecting(0x21, 0, 20, 40, 60, 80, 100); !reflect.DeepEqual(got, want) {
		t.Errorf("streamed %v, want %v and nothing at the exit", got, want)
	}
	if level := rig.ps.GetReservoirLevel(); level != 200 {
		t.Errorf("reservoir %v", level)
	}
}

func TestLoadStatusReportsTheChangeMode(t *testing.T) {
	rig := newProcedureRig(t)
	load := rig.route(bluetooth.CharCurrentStatus, "LoadStatusRequest", 1, nil)
	if fmt.Sprint(load.Cargo["isLoadingActive"]) != "false" {
		t.Errorf("an idle pump reads %v", load.Cargo)
	}
	rig.enterChangeCartridge(2)
	load = rig.route(bluetooth.CharCurrentStatus, "LoadStatusRequest", 3, nil)
	if fmt.Sprint(load.Cargo["isLoadingActive"]) != "true" || fmt.Sprint(load.Cargo["loadStateId"]) != "0" {
		t.Errorf("change mode reads %v, want CHANGE_CARTRIDGE active", load.Cargo)
	}
}

func fillCannula(txID uint8, id byte) streamedMessage {
	return streamedMessage{protocol.OpcodeFillCannulaStateStreamResponse, txID, hex.EncodeToString([]byte{id})}
}

func TestFillCannulaStreamsFilledAfterThePrimeGoesIn(t *testing.T) {
	rig := newProcedureRig(t)
	rig.ps.SetReservoirLevel(100)
	reply := rig.route(bluetooth.CharControl, "FillCannulaRequest", 0x30, map[string]interface{}{"primeSizeMilliUnits": 300})
	if s := status(t, reply); s != 0 {
		t.Fatalf("FillCannulaRequest answered %d", s)
	}
	if got := rig.timers.pending(); len(got) != 1 || got[0] != 6*time.Second {
		t.Errorf("pending steps %v, want the 0.3 U prime at the default 0.05 U/s", got)
	}
	rig.timers.drain(t)

	if got, want := rig.streamed(), []streamedMessage{fillCannula(0x30, 0), fillCannula(0x30, 2)}; !reflect.DeepEqual(got, want) {
		t.Errorf("streamed %v, want %v", got, want)
	}
	if level := rig.ps.GetReservoirLevel(); math.Abs(level-99.7) > 1e-9 {
		t.Errorf("reservoir %v, want 99.7", level)
	}
	record := rig.lastRecord("CannulaFilled")
	if record.Data["primeSize"] != 0.3 || record.Data["completionStatus"] != state.FillCompletionStatusCompleted {
		t.Errorf("CannulaFilled %v", record.Data)
	}
	if !bytes.Equal(record.EncodeRecord()[14:18], []byte{3, 0, 0, 0}) {
		t.Errorf("CannulaFilled completionStatus on the wire %x", record.EncodeRecord()[14:18])
	}
}

func TestFillCannulaStreamLostNeverStreamsFilled(t *testing.T) {
	rig := newProcedureRig(t)
	rig.ps.UpdateCartridgeSettings(func(s *state.CartridgeSettings) { s.FillCannulaStreamLost = true })
	rig.route(bluetooth.CharControl, "FillCannulaRequest", 0x30, map[string]interface{}{"primeSizeMilliUnits": 500})
	rig.timers.drain(t)

	if got, want := rig.streamed(), []streamedMessage{fillCannula(0x30, 0)}; !reflect.DeepEqual(got, want) {
		t.Errorf("streamed %v, want only the filling state", got)
	}
	rig.lastRecord("CannulaFilled")
}

func TestFillCannulaIsRefusedWhileDelivering(t *testing.T) {
	rig := newProcedureRig(t)
	rig.ps.SetPumpingSuspended(false)
	if s := status(t, rig.route(bluetooth.CharControl, "FillCannulaRequest", 0x30, map[string]interface{}{"primeSizeMilliUnits": 300})); s != 1 {
		t.Errorf("answered %d while delivering, want 1", s)
	}
	rig.timers.drain(t)
	if got := rig.streamed(); len(got) != 0 {
		t.Errorf("a refused fill streamed %v", got)
	}
}

func exitFillTubing(txID uint8, id byte) streamedMessage {
	return streamedMessage{protocol.OpcodeExitFillTubingModeStateStreamResponse, txID, hex.EncodeToString([]byte{id})}
}

func TestExitFillTubingStreamsAndRecordsTheFill(t *testing.T) {
	rig := newProcedureRig(t)
	rig.ps.SetReservoirLevel(100)
	rig.route(bluetooth.CharControl, "EnterFillTubingModeRequest", 0x40, nil)
	if err := rig.router.SetFillTubingButton(true); err != nil {
		t.Fatal(err)
	}
	rig.clock.Advance(8 * time.Second)
	if err := rig.router.SetFillTubingButton(false); err != nil {
		t.Fatal(err)
	}
	if s := status(t, rig.route(bluetooth.CharControl, "ExitFillTubingModeRequest", 0x41, nil)); s != 0 {
		t.Fatalf("exit answered %d", s)
	}

	got := rig.streamed()
	if want := exitFillTubing(0x41, 1); len(got) != 3 || got[2] != want {
		t.Errorf("streamed %v, want the button down and up, then %v", got, want)
	}
	if level := rig.ps.GetReservoirLevel(); level != 96 {
		t.Errorf("reservoir %v, want 100 less 8 s at 0.5 U/s", level)
	}
	if record := rig.lastRecord("TubingFilled"); record.Data["primeSize"] != 4.0 {
		t.Errorf("TubingFilled %v", record.Data)
	}

	rig.route(bluetooth.CharControl, "EnterFillTubingModeRequest", 0x42, nil)
	rig.route(bluetooth.CharControl, "ExitFillTubingModeRequest", 0x43, nil)
	if got := rig.streamed(); got[len(got)-1] != exitFillTubing(0x43, 0) {
		t.Errorf("a fill with no press streamed %v, want NOT_COMPLETE", got[len(got)-1])
	}
}

func TestThePumpReleasesTheButtonAtMaxFillSeconds(t *testing.T) {
	rig := newProcedureRig(t)
	rig.ps.UpdateCartridgeSettings(func(s *state.CartridgeSettings) { s.MaxFillSeconds = 20 })
	rig.route(bluetooth.CharControl, "EnterFillTubingModeRequest", 0x40, nil)
	if err := rig.router.SetFillTubingButton(true); err != nil {
		t.Fatal(err)
	}
	if got := rig.timers.pending(); !reflect.DeepEqual(got, []time.Duration{20 * time.Second}) {
		t.Errorf("pending steps %v, want the release at max_fill_seconds", got)
	}
	rig.clock.Advance(20 * time.Second)
	rig.timers.drain(t)

	if rig.ps.FillTubingButtonDown() {
		t.Error("the button is still down past max_fill_seconds")
	}
	if got := streamedButtonStates(t, rig.ps, rig.transport, 0x40); !reflect.DeepEqual(got, []int32{1, 0}) {
		t.Errorf("button states %v, want the press and the pump's release", got)
	}
}

func TestAReleaseCancelsThePumpsOwnStop(t *testing.T) {
	rig := newProcedureRig(t)
	rig.route(bluetooth.CharControl, "EnterFillTubingModeRequest", 0x40, nil)
	for _, down := range []bool{true, false} {
		if err := rig.router.SetFillTubingButton(down); err != nil {
			t.Fatal(err)
		}
	}
	if got := rig.timers.pending(); len(got) != 0 {
		t.Errorf("steps %v still pending after the release", got)
	}
}
