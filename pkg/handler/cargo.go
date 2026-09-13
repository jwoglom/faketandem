package handler

import (
	"encoding/json"
	"strconv"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
)

// Cargo field access helpers.
//
// Two things make raw `msg.Cargo[...]` type assertions unreliable, and both
// have bitten this package before:
//
//  1. The key names come from pumpX2's own Java field names, printed by each
//     Message subclass's generated toString() and scraped by
//     pumpx2.parseCliparserOutput. They are NOT the friendlier names a handler
//     author might guess: an InitiateBolusRequest carries "totalVolume" (in
//     milliunits) and "bolusID", not "insulin"/"units"/"bolusId"; a
//     SetTempRateRequest carries "minutes"/"percent", not "duration"/
//     "percentage"; a HistoryLogRequest carries "startLog"/"numberOfLogs", not
//     "startSequence"/"endSequence".
//
//  2. The *types* are Go ints, not float64s. parseFieldValue runs
//     strconv.Atoi first, so an integral field arrives as `int`. A
//     `.(float64)` assertion therefore never matches, and silently leaves the
//     handler with its zero-value default -- which is exactly how
//     InitiateBolusRequest ended up rejecting every bolus with "invalid bolus
//     units". Cargo maps that have made a round trip through JSON (the HTTP/
//     websocket API) do hold float64s, so both must be handled.
//
// cargoInt/cargoFloat/cargoBool accept the first key present and coerce
// whatever numeric representation is behind it.

// cargoValue returns the first of keys present in the message's cargo.
func cargoValue(msg *pumpx2.ParsedMessage, keys ...string) (interface{}, bool) {
	if msg == nil || msg.Cargo == nil {
		return nil, false
	}
	for _, key := range keys {
		if v, ok := msg.Cargo[key]; ok {
			return v, true
		}
	}
	return nil, false
}

// cargoFloat reads a numeric cargo field as a float64, accepting any of the
// integer, float, json.Number or numeric-string representations cliparser
// output and JSON round-trips can produce.
func cargoFloat(msg *pumpx2.ParsedMessage, keys ...string) (float64, bool) {
	raw, ok := cargoValue(msg, keys...)
	if !ok {
		return 0, false
	}
	return coerceFloat(raw)
}

// coerceFloat converts one cargo value to a float64. Split out from cargoFloat
// purely to keep each function's branch count reasonable.
func coerceFloat(raw interface{}) (float64, bool) {
	if f, ok := coerceSignedInt(raw); ok {
		return f, true
	}
	if f, ok := coerceUnsignedInt(raw); ok {
		return f, true
	}

	switch v := raw.(type) {
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func coerceSignedInt(raw interface{}) (float64, bool) {
	switch v := raw.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

func coerceUnsignedInt(raw interface{}) (float64, bool) {
	switch v := raw.(type) {
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	default:
		return 0, false
	}
}

// cargoInt reads a numeric cargo field as an int64.
func cargoInt(msg *pumpx2.ParsedMessage, keys ...string) (int64, bool) {
	f, ok := cargoFloat(msg, keys...)
	if !ok {
		return 0, false
	}
	return int64(f), true
}

// cargoBool reads a boolean cargo field. cliparser renders Java booleans as
// the literal strings "true"/"false"; parseFieldValue turns those into Go
// bools, but a JSON round-trip or a numeric 0/1 encoding is also accepted.
func cargoBool(msg *pumpx2.ParsedMessage, keys ...string) (bool, bool) {
	raw, ok := cargoValue(msg, keys...)
	if !ok {
		return false, false
	}
	switch v := raw.(type) {
	case bool:
		return v, true
	case string:
		b, err := strconv.ParseBool(v)
		return b, err == nil
	default:
		if f, ok := cargoFloat(msg, keys...); ok {
			return f != 0, true
		}
		return false, false
	}
}
