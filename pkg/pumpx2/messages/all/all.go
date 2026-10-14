// Package all links every pumpX2 message type into the program, as the
// image/png import links PNG decoding: import it for its side effect,
// wherever messages are parsed or built by name.
package all

import (
	_ "github.com/jwoglom/faketandem/pkg/pumpx2/messages/authentication"
	_ "github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	_ "github.com/jwoglom/faketandem/pkg/pumpx2/messages/controlstream"
	_ "github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	_ "github.com/jwoglom/faketandem/pkg/pumpx2/messages/historylog"
)
