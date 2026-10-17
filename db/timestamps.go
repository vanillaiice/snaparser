package db

// Snapchat exports can put Unix milliseconds in "Created(microseconds)".
// For Snapchat-era dates, milliseconds fall in this range while actual Unix
// microseconds are larger. Keep zero/unknown values and real microseconds as is.
// This is specific to Snapchat history, not a general Unix timestamp detector.
// Keep the bounds in sync with migration 000003 for already imported archives.
func snapTimestampMicroseconds(value int64) int64 {
	if value >= 1_000_000_000_000 && value < 100_000_000_000_000 {
		return value * 1000
	}

	return value
}
