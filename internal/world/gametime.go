package world

import "time"

// L1J game time runs at 6x real time, with a base epoch.
// Reference: l1j_java L1GameTime.java, L1GameTimeClock.java
//
// Original Java uses July 3, 2003 (1057233600000 ms), but in 2025+ the resulting
// game-time seconds exceed int32 range (>2.1 billion). The 3.80C Windows client
// passes the value to Win32 time functions that crash on negative time_t.
// Using Jan 1, 2025 keeps the value safely positive for ~11 years.

const baseTimeMillis int64 = 1735689600000 // January 1, 2025 00:00:00 UTC in milliseconds

// GameTime represents a point in L1J game time as seconds since the base epoch.
type GameTime struct {
	seconds int
}

// GameTimeNow returns the current game time derived from the system clock.
func GameTimeNow() GameTime {
	t1 := time.Now().UnixMilli() - baseTimeMillis
	t2 := int((t1 * 6) / 1000)
	t2 -= t2 % 3 // align to 3-second boundary (matches Java)
	return GameTime{seconds: t2}
}

// Seconds returns the raw game-time value for S_GameTime packet.
func (gt GameTime) Seconds() int {
	return gt.seconds
}

// calendar returns the Go time.Time corresponding to this game time.
func (gt GameTime) calendar() time.Time {
	return time.Unix(int64(gt.seconds), 0).UTC()
}

// Hour returns the game-time hour (0-23).
func (gt GameTime) Hour() int {
	return gt.calendar().Hour()
}

// Minute returns the game-time minute (0-59).
func (gt GameTime) Minute() int {
	return gt.calendar().Minute()
}

// IsNight returns true if the game-time hour is outside 6:00-17:59.
func (gt GameTime) IsNight() bool {
	h := gt.Hour()
	return h < 6 || h >= 18
}
