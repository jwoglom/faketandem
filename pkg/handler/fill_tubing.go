package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/protocol"

	log "github.com/sirupsen/logrus"
)

// SetFillTubingButton presses (down) or releases the pump's button in fill
// tubing mode, which a person holds on a real pump to fill the tubing, and
// streams its new state to the app as a FillTubingStateStreamResponse.
//
// It refuses outside fill tubing mode, and before the app has authenticated,
// since the stream message is signed with the session key.
func (r *Router) SetFillTubingButton(down bool) error {
	authKey := r.pumpState.GetAuthKey()
	if len(authKey) == 0 {
		return fmt.Errorf("no authenticated app to stream the button's state to")
	}

	txID, err := r.pumpState.SetFillTubingButton(down)
	if err != nil {
		return err
	}

	msg, err := protocol.BuildFillTubingStateStreamResponse(txID, down, authKey, r.pumpState.GetTimeSinceReset())
	if err != nil {
		return err
	}

	log.Infof("Fill tubing button %s; streaming it under txID=%d", buttonWord(down), txID)
	sent, dropped, sendErr := r.sendNative(msg)
	if r.requestLog != nil && !dropped {
		note := "fill tubing button " + buttonWord(down)
		if sendErr != nil {
			note += "; send error: " + sendErr.Error()
		}
		r.requestLog.RecordResponse(msg.Characteristic.String(), msg.MessageType, int(msg.Opcode), int(msg.TxID),
			msg.PacketsHex(), sent, "", note)
	}
	return sendErr
}

func buttonWord(down bool) string {
	if down {
		return "down"
	}
	return "up"
}
