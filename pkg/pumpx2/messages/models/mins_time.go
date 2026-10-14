// Package models is pumpX2's messages.models: the values several messages
// share.
package models

import "fmt"

// MinsTime is a time of day the pump keeps as minutes since midnight.
type MinsTime struct {
	Hour int32
	Min  int32
}

// NewMinsTime splits minutes since midnight into hours and minutes.
func NewMinsTime(totalMins int32) MinsTime {
	return MinsTime{Hour: totalMins / 60, Min: totalMins % 60}
}

// Encode is the time as minutes since midnight.
func (t MinsTime) Encode() int32 { return t.Hour*60 + t.Min }

// String is the time as pumpX2 prints it, hh:mm.
func (t MinsTime) String() string { return fmt.Sprintf("%02d:%02d", t.Hour, t.Min) }
