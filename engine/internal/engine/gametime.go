package engine

import (
	"fmt"
	"time"
)

var gameEpoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// broadcastOutdoor sends a message to all players in outdoor rooms.
func (e *GameEngine) broadcastOutdoor(msg string) {
	if e.localRoomBroadcast == nil {
		return
	}
	for num, room := range e.rooms {
		if isOutdoorTerrain(room.Terrain) {
			e.localRoomBroadcast(num, []string{msg})
		}
	}
}

var MonthNames = []string{
	"", "Abra", "Brama", "Manta", "Dretmar", "Alabea", "Phobrus",
	"Melma", "Banamea", "Olum", "Mixus", "Farnum", "Folster", "Feast",
}

func gameTimeSinceEpoch() time.Duration { return time.Since(gameEpoch) }
func GameMinutes() int                  { return int(gameTimeSinceEpoch().Minutes()) }
func GameHour() int                     { return GameMinutes() % 24 }
func GameDay() int                      { return (GameMinutes()/24)%343 + 1 }
func GameMonth() int {
	m := ((GameDay() - 1) / 28) + 1
	if m > 12 { m = 13 }
	return m
}
func GameYear() int { return GameMinutes()/(24*343) + 1028 }
func IsNight() bool { h := GameHour(); return h < 5 || h > 19 }
func IsDay() bool   { return !IsNight() }

// GameSeason returns the current season script key based on game month.
// Months 1-3 = Spring (PSCRIPT), 4-6 = Summer (SSCRIPT),
// 7-9 = Autumn (ASCRIPT), 10-12 = Winter (WSCRIPT).
func GameSeason() string {
	m := GameMonth()
	switch {
	case m >= 1 && m <= 3:
		return "PSCRIPT"
	case m >= 4 && m <= 6:
		return "SSCRIPT"
	case m >= 7 && m <= 9:
		return "ASCRIPT"
	default:
		return "WSCRIPT"
	}
}

// SeasonName returns a human-readable season name.
func SeasonName() string {
	switch GameSeason() {
	case "PSCRIPT":
		return "Spring"
	case "SSCRIPT":
		return "Summer"
	case "ASCRIPT":
		return "Autumn"
	case "WSCRIPT":
		return "Winter"
	}
	return "Spring"
}

func GameMonthName() string {
	m := GameMonth()
	if m >= 1 && m < len(MonthNames) {
		return MonthNames[m]
	}
	return "Feast"
}

// StartTimeCycle starts a background goroutine that publishes time change events.
func (e *GameEngine) StartTimeCycle() {
	go func() {
		lastHour := GameHour()
		wasNight := IsNight()
		ticker := time.NewTicker(10 * time.Second) // check every 10 real seconds
		defer ticker.Stop()
		for range ticker.C {
			if !e.Events.HasSubscribers() {
				lastHour = GameHour()
				wasNight = IsNight()
				continue
			}
			hour := GameHour()
			night := IsNight()
			// Check for season changes
			e.CheckSeasonChange()

			if hour != lastHour {
				period := "day"
				if night { period = "night" }
				e.Events.Publish("time", fmt.Sprintf("Hour %d:00 — %s of %s, Day %d of %s, Year %d",
					hour, period, GameMonthName(), GameDay(), GameMonthName(), GameYear()))
				if night != wasNight {
					if night {
						e.Events.Publish("time", "Night falls across the Shattered Realms.")
					} else {
						e.Events.Publish("time", "Dawn breaks across the Shattered Realms.")
					}
				}
				// Broadcast time-of-day transitions to outdoor players
				if e.localRoomBroadcast != nil {
					switch hour {
					case 5:
						e.broadcastOutdoor("The sun begins to rise in the east.")
					case 6:
						e.broadcastOutdoor("The sun rises in the east.")
					case 18:
						e.broadcastOutdoor("The sun begins to set in the west.")
					case 19:
						e.broadcastOutdoor("The sun sets in the west.")
					}
				}
				lastHour = hour
				wasNight = night
			}
		}
	}()
}
