package engine

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Time ratio: 4 real hours = 24 game hours → 6 game minutes per real minute.
const gameTimeRatio = 6

// gameTimeOffset is the total game-minutes elapsed, loaded from MongoDB on startup.
// All time functions derive from this plus elapsed real time since load.
var (
	gameTimeOffset    int       // game-minutes at server start (from DB)
	gameTimeLoadedAt  time.Time // real wall-clock when offset was loaded
)

func initGameTime(offset int) {
	gameTimeOffset = offset
	gameTimeLoadedAt = time.Now()
}

// GameMinutes returns total elapsed game-minutes since the world began.
func GameMinutes() int {
	elapsed := time.Since(gameTimeLoadedAt)
	return gameTimeOffset + int(elapsed.Minutes()*gameTimeRatio)
}

func GameHour() int { return GameMinutes() / 60 % 24 }
func GameDay() int  { return GameMinutes()/(60*24)%336 + 1 } // 336 days = 12 months × 28 days

// TimePeriod returns the current time of day, used by the TIME command to give
// players a more specific read than day/night. "dawn" is a single hour (6),
// not a band — scripts key exact-moment events off IFVAR TIM = <hour> (e.g. the
// JEZRAEL.SC1 fountain's dawn dance requires TIM = 6 exactly, SALOON.SCR's sunset
// vista requires TIM = 19 exactly), so the displayed label must match the hour a
// script actually checks or those events become nearly untriggerable by players
// going solely off what TIME tells them. midnight (0) and noon (12) are the same
// kind of exact-moment anchor and already sat correctly in the middle of their
// bands. The two hours freed up by narrowing dawn to a point are absorbed by its
// immediate neighbors so all 24 hours stay covered.
func TimePeriod() string {
	switch h := GameHour(); {
	case h < 2 || h >= 23:
		return "midnight"
	case h < 6:
		return "very early morning"
	case h == 6:
		return "dawn"
	case h < 11:
		return "mid morning"
	case h < 14:
		return "noon"
	case h < 17:
		return "afternoon"
	case h < 20:
		return "evening"
	default:
		return "night"
	}
}

func GameMonth() int {
	m := ((GameDay() - 1) / 28) + 1
	if m > 12 {
		m = 12
	}
	return m
}

func GameYear() int  { return GameMinutes()/(60*24*336) + 1028 }
func IsNight() bool  { h := GameHour(); return h < 5 || h > 19 }
func IsDay() bool    { return !IsNight() }

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

var MonthNames = []string{
	"", "Abra", "Brama", "Manta", "Dretmar", "Alabea", "Phobrus",
	"Melma", "Banamea", "Olum", "Mixus", "Farnum", "Folster",
}

func GameMonthName() string {
	m := GameMonth()
	if m >= 1 && m < len(MonthNames) {
		return MonthNames[m]
	}
	return "Folster"
}

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

// LoadGameTime loads the persisted game time from MongoDB.
// Returns 0 if no saved state exists (fresh world starts at minute 0).
func LoadGameTime(db *mongo.Database) int {
	if db == nil {
		initGameTime(0)
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var doc struct {
		GameMinutes int `bson:"gameMinutes"`
	}
	err := db.Collection("game_state").FindOne(ctx, bson.M{"_id": "gametime"}).Decode(&doc)
	if err != nil {
		log.Printf("No saved game time found, starting at minute 0")
		initGameTime(0)
		return 0
	}
	log.Printf("Loaded game time: %d minutes (Year %d, %s %d, %s)",
		doc.GameMinutes, 0, "", 0, "") // will be filled after initGameTime
	initGameTime(doc.GameMinutes)
	log.Printf("Game time: Year %d, %s %d (%s, %s)",
		GameYear(), GameMonthName(), GameDay()%28+1, SeasonName(),
		func() string { if IsNight() { return "night" }; return "day" }())
	return doc.GameMinutes
}

// SaveGameTime persists the current game time to MongoDB.
func SaveGameTime(db *mongo.Database) {
	if db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	mins := GameMinutes()
	opts := options.Replace().SetUpsert(true)
	_, err := db.Collection("game_state").ReplaceOne(ctx, bson.M{"_id": "gametime"},
		bson.M{"_id": "gametime", "gameMinutes": mins, "updatedAt": time.Now()}, opts)
	if err != nil {
		log.Printf("Failed to save game time: %v", err)
	}
}

// StartTimeCycle starts a background goroutine that publishes time change events
// and periodically saves game time to MongoDB.
func (e *GameEngine) StartTimeCycle() {
	go func() {
		lastHour := GameHour()
		wasNight := IsNight()
		saveCounter := 0
		ticker := time.NewTicker(10 * time.Second) // check every 10 real seconds
		defer ticker.Stop()
		for range ticker.C {
			// Save game time to DB every ~5 minutes (30 ticks × 10 sec)
			saveCounter++
			if saveCounter >= 30 {
				SaveGameTime(e.db)
				saveCounter = 0
			}

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
				e.Events.Publish("time", fmt.Sprintf("Hour %d:00 — %s of %s %d, Year %d",
					hour, period, GameMonthName(), GameDay()%28+1, GameYear()))
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
