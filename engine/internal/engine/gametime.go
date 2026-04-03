package engine

import "time"

var gameEpoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func gameTimeSinceEpoch() time.Duration { return time.Since(gameEpoch) }
func GameMinutes() int                  { return int(gameTimeSinceEpoch().Minutes()) }
func GameHour() int                     { return GameMinutes() % 24 }
func GameDay() int                      { return (GameMinutes()/24)%343 + 1 }
func GameMonth() int                    { return ((GameDay() - 1) / 28) + 1 } // 1-12, feast days = month 13
func GameYear() int                     { return GameMinutes()/(24*343) + 1028 }
func IsNight() bool                     { h := GameHour(); return h < 5 || h > 19 }
func IsDay() bool                       { return !IsNight() }
