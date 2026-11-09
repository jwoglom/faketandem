package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/protocol"

	log "github.com/sirupsen/logrus"
)

// SetFillTubingButton presses (down) or releases the pump's button in fill
// tubing mode, which a person holds on a real pump to fill the tubing, and
// streams its new state to the app as a FillTubingStateStreamResponse. A press
// held past max_fill_seconds is released by the pump itself.
//
// It refuses outside fill tubing mode, and before the app has authenticated,
// since the stream message is signed with the session key.
func (r *Router) SetFillTubingButton(down bool) error {
	ps := r.pumpState
	if len(ps.GetAuthKey()) == 0 {
		return fmt.Errorf("no authenticated app to stream the button's state to")
	}

	txID, err := ps.SetFillTubingButton(down)
	if err != nil {
		return err
	}
	if down {
		r.stopFillAtLimit()
	} else {
		r.procs.stop(procedureMaxFill)
	}

	return r.stream(ps, "fill tubing button "+buttonWord(down), func(key []byte, tsr uint32) (*protocol.NativeMessage, error) {
		return protocol.BuildFillTubingStateStreamResponse(txID, down, key, tsr)
	})
}

// stopFillAtLimit releases the current press once it has filled for max_fill_seconds, as the
// pump stops a fill on its own. The pump then raises FILL_TUBING_STILL_IN_PROGRESS, which the
// emulator does not.
func (r *Router) stopFillAtLimit() {
	ps := r.pumpState
	press := ps.FillTubingPress()
	r.procs.start(procedureMaxFill, secondsDuration(ps.CartridgeSettings().MaxFillSeconds), func() {
		if !ps.FillTubingButtonDown() || ps.FillTubingPress() != press {
			return
		}
		log.Infof("Fill tubing reached max_fill_seconds; the pump releases the button")
		if err := r.SetFillTubingButton(false); err != nil {
			log.Warnf("Could not stop the fill at its limit: %v", err)
		}
	})
}

func buttonWord(down bool) string {
	if down {
		return "down"
	}
	return "up"
}
