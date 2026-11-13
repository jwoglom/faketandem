package state

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// WorkflowMode is the cartridge procedure the pump is part-way through. Only one can be open.
//
// A request to enter one while another is open is refused, and so is exiting one the pump is not
// in (pumpX2 documents that for ExitChangeCartridgeMode). That a fill-tubing mode left open is
// what refuses the next change-cartridge is inferred from a field report, not confirmed on firmware.
type WorkflowMode string

const (
	WorkflowNone            WorkflowMode = "none"
	WorkflowChangeCartridge WorkflowMode = "change_cartridge"
	WorkflowFillTubing      WorkflowMode = "fill_tubing"
)

// ParseWorkflowMode reports whether s names a workflow mode.
func ParseWorkflowMode(s string) (WorkflowMode, bool) {
	switch mode := WorkflowMode(s); mode {
	case WorkflowNone, WorkflowChangeCartridge, WorkflowFillTubing:
		return mode, true
	}
	return WorkflowNone, false
}

// WorkflowModeUnlocked is for callers already holding the state lock, as SuspendReason is.
func (ps *PumpState) WorkflowModeUnlocked() WorkflowMode {
	if ps.workflowMode == "" {
		return WorkflowNone
	}
	return ps.workflowMode
}

// CurrentWorkflowMode returns the open procedure, or WorkflowNone.
func (ps *PumpState) CurrentWorkflowMode() WorkflowMode {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.WorkflowModeUnlocked()
}

// SetWorkflowMode stages a mode directly, with no checks, for a harness that needs to start a
// test with the pump already stuck in one.
func (ps *PumpState) SetWorkflowMode(mode WorkflowMode) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.workflowMode = mode
	ps.fillTubingButtonDown = false
	ps.cartridge.readyToChange = false
	ps.cartridge.fillUnits = 0
}

// EnterWorkflow opens mode and reports whether the pump accepted it. Change-cartridge also needs
// delivery suspended (pumpX2: "Precondition: insulin must be suspended"), and no new cartridge
// waiting for its tubing fill: a Mobi refused change mode after an exit it took as the new
// cartridge being in.
func (ps *PumpState) EnterWorkflow(mode WorkflowMode) bool {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	return ps.enterWorkflowUnlocked(mode)
}

func (ps *PumpState) enterWorkflowUnlocked(mode WorkflowMode) bool {
	if open := ps.WorkflowModeUnlocked(); open != WorkflowNone {
		log.Warnf("Refusing to enter %s: %s is still open", mode, open)
		return false
	}
	if mode == WorkflowChangeCartridge && !ps.PumpingSuspended {
		log.Warnf("Refusing to enter %s: delivery is not suspended", mode)
		return false
	}
	if mode == WorkflowChangeCartridge && (ps.cartridge.detecting || ps.cartridge.loaded) {
		log.Warnf("Refusing to enter %s: a new cartridge is waiting for its tubing fill", mode)
		return false
	}

	ps.workflowMode = mode
	log.Infof("Entered %s mode", mode)
	return true
}

// ExitWorkflow closes mode and reports whether the pump was in it. Leaving change cartridge mode
// starts checking the new cartridge (ExitChangeCartridge), and leaving fill tubing mode ends the
// fill (ExitFillTubing).
func (ps *PumpState) ExitWorkflow(mode WorkflowMode) bool {
	switch mode {
	case WorkflowChangeCartridge:
		accepted, _ := ps.ExitChangeCartridge()
		return accepted
	case WorkflowFillTubing:
		accepted, _ := ps.ExitFillTubing()
		return accepted
	}
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	return ps.exitWorkflowUnlocked(mode)
}

// exitWorkflowOpen reports whether mode is the open procedure, logging the refusal when not.
func (ps *PumpState) exitWorkflowOpen(mode WorkflowMode) bool {
	if open := ps.WorkflowModeUnlocked(); open != mode {
		log.Warnf("Refusing to exit %s: pump is in %s", mode, open)
		return false
	}
	return true
}

func (ps *PumpState) exitWorkflowUnlocked(mode WorkflowMode) bool {
	if !ps.exitWorkflowOpen(mode) {
		return false
	}

	ps.workflowMode = WorkflowNone
	ps.fillTubingButtonDown = false
	log.Infof("Exited %s mode", mode)
	return true
}

// EnterFillTubing is EnterWorkflow(WorkflowFillTubing) that also keeps txID, the
// EnterFillTubingModeRequest's: the pump streams the button's state under it for
// as long as the mode is open.
func (ps *PumpState) EnterFillTubing(txID uint8) bool {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	if !ps.enterWorkflowUnlocked(WorkflowFillTubing) {
		return false
	}
	ps.fillTubingTxID = txID
	ps.fillTubingButtonDown = false
	ps.cartridge.fillUnits = 0
	return true
}

// FillTubingButtonDown reports whether the pump's button is held in fill tubing mode.
func (ps *PumpState) FillTubingButtonDown() bool {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.fillTubingButtonDown
}

// FillTubingButtonDownUnlocked is for callers already holding the state lock.
func (ps *PumpState) FillTubingButtonDownUnlocked() bool {
	return ps.fillTubingButtonDown
}

// SetFillTubingButton presses (down) or releases the pump's button, which in fill
// tubing mode fills the tubing for as long as it is held. It returns the txID the
// pump streams the button's state under, and refuses outside fill tubing mode,
// where the button does something else.
func (ps *PumpState) SetFillTubingButton(down bool) (uint8, error) {
	now := ps.Now()
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if open := ps.WorkflowModeUnlocked(); open != WorkflowFillTubing {
		return 0, fmt.Errorf("the pump is in %s, not %s", open, WorkflowFillTubing)
	}
	ps.fillTubingButtonUnlocked(down, now)
	return ps.fillTubingTxID, nil
}
