package helpers

import (
	"fmt"
	"time"
)

// January12008UnixEpoch is the pump's epoch: its timestamps count seconds
// from 2008-01-01T00:00:00Z.
const January12008UnixEpoch = 1199145600

const (
	minInstantSecond = -31557014167219200
	maxInstantSecond = 31556889864403199
)

// FromJan12008ToUnixEpochSeconds converts a pump timestamp to Unix seconds.
func FromJan12008ToUnixEpochSeconds(seconds int64) int64 {
	return seconds + January12008UnixEpoch
}

// FromJan12008EpochSecondsToDate is a pump timestamp as a time, failing
// where java.time.Instant's range ends.
func FromJan12008EpochSecondsToDate(seconds int64) (time.Time, error) {
	return instantOfEpochSecond(FromJan12008ToUnixEpochSeconds(seconds))
}

// FromInstantToJan12008EpochSeconds is a time as a pump timestamp.
func FromInstantToJan12008EpochSeconds(t time.Time) int64 {
	return t.Unix() - January12008UnixEpoch
}

// FromJan12008EpochDaysToDate is a count of days since the pump's epoch as a
// time.
func FromJan12008EpochDaysToDate(days int64) (time.Time, error) {
	return instantOfEpochSecond(FromJan12008ToUnixEpochSeconds(days * 24 * 60 * 60))
}

func instantOfEpochSecond(s int64) (time.Time, error) {
	if s < minInstantSecond || s > maxInstantSecond {
		return time.Time{}, fmt.Errorf("instant %d exceeds minimum or maximum instant", s)
	}
	return time.Unix(s, 0).UTC(), nil
}
