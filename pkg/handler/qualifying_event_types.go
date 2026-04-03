package handler

// Qualifying event bitmask IDs matching pumpX2's QualifyingEvent enum.
// These are sent as bitmask notifications on the QualifyingEvents characteristic.
const (
	QEAlert                  uint32 = 1
	QEAlarm                  uint32 = 2
	QEReminder               uint32 = 4
	QEMalfunction            uint32 = 8
	QECGMAlert               uint32 = 16
	QEHomeScreenChange       uint32 = 32
	QEPumpSuspend            uint32 = 64
	QEPumpResume             uint32 = 128
	QETimeChange             uint32 = 256
	QEBasalChange            uint32 = 512
	QEBolusChange            uint32 = 1024
	QEIOBChange              uint32 = 2048
	QEExtendedBolusChange    uint32 = 4096
	QEProfileChange          uint32 = 8192
	QEBG                     uint32 = 16384
	QECGMChange              uint32 = 32768
	QEBattery                uint32 = 65536
	QEBasalIQ                uint32 = 131072
	QERemainingInsulin       uint32 = 262144
	QEPumpCommSuspended      uint32 = 524288
	QEActiveProfileSegChange uint32 = 1048576
	QEBasalIQStatus          uint32 = 2097152
	QEControlIQInfo          uint32 = 4194304
	QEControlIQSleep         uint32 = 8388608
	QEGlobalPumpSettings     uint32 = 16777216
	QESnoozeStatus           uint32 = 33554432
	QEPumpingStatus          uint32 = 67108864
	QEPumpReset              uint32 = 134217728
	QEHeartbeat              uint32 = 268435456
	QEBolusPermissionRevoked uint32 = 2147483648
)
