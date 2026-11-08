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
}

// EnterWorkflow opens mode and reports whether the pump accepted it. Change-cartridge also needs
// delivery suspended (pumpX2: "Precondition: insulin must be suspended").
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

	ps.workflowMode = mode
	log.Infof("Entered %s mode", mode)
	return true
}

// ExitWorkflow closes mode and reports whether the pump was in it.
func (ps *PumpState) ExitWorkflow(mode WorkflowMode) bool {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if open := ps.WorkflowModeUnlocked(); open != mode {
		log.Warnf("Refusing to exit %s: pump is in %s", mode, open)
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
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if open := ps.WorkflowModeUnlocked(); open != WorkflowFillTubing {
		return 0, fmt.Errorf("the pump is in %s, not %s", open, WorkflowFillTubing)
	}
	ps.fillTubingButtonDown = down
	return ps.fillTubingTxID, nil
}
