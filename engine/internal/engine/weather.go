package engine

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
)

// WeatherNames maps weather state IDs to display names (from GM Manual).
var WeatherNames = map[int]string{
	0:  "Sunny",
	1:  "Partly Cloudy",
	2:  "Overcast",
	3:  "Light Rain",
	4:  "Moderate Rain",
	5:  "Heavy Rain",
	6:  "Thunderstorm",
	7:  "Gale",
	8:  "Hurricane",
	9:  "Hail",
	10: "Sleet",
	11: "Snow Flurries",
	12: "Moderate Snow",
	13: "Heavy Snow",
	14: "Blizzard",
}

// Weather transition messages broadcast to outdoor players.
var weatherTransitionMessages = map[[2]int]string{
	{0, 1}:  "A few clouds drift across the sky.",
	{0, 2}:  "The sky darkens and becomes overcast.",
	{1, 0}:  "The clouds overhead drift away, leaving only clear skies.",
	{1, 2}:  "The sky darkens as clouds gather overhead.",
	{2, 0}:  "The clouds break apart, revealing clear blue sky.",
	{2, 1}:  "The overcast sky begins to lighten as gaps appear in the clouds.",
	{2, 3}:  "A light rain begins to fall.",
	{3, 2}:  "The rain tapers off and the clouds begin to thin.",
	{3, 4}:  "The rain intensifies to a steady downpour.",
	{4, 3}:  "The rain lessens to a light drizzle.",
	{4, 5}:  "The rain grows heavier, driven by gusting wind.",
	{5, 4}:  "The downpour eases somewhat.",
	{5, 6}:  "Thunder rumbles in the distance as lightning flashes across the sky.",
	{6, 5}:  "The thunder fades, though heavy rain continues.",
	{6, 2}:  "The storm passes, leaving only overcast skies.",
	{2, 11}: "Snowflakes begin to fall gently from the gray sky.",
	{11, 2}: "The snow stops and the clouds begin to thin.",
	{11, 12}: "The snow picks up, falling more heavily now.",
	{12, 11}: "The snowfall lightens to scattered flurries.",
	{12, 13}: "It is snowing heavily.",
	{13, 12}: "The heavy snow begins to let up.",
	{13, 14}: "A howling blizzard descends, reducing visibility to almost nothing.",
	{14, 13}: "The blizzard eases to heavy snow.",
}

// weatherTransitions defines likely next states from each weather state.
// Each entry is [nextState, weight]. Higher weight = more likely.
var weatherTransitions = map[int][][2]int{
	0:  {{0, 60}, {1, 30}, {2, 10}},                // Sunny → stay, partly cloudy, overcast
	1:  {{0, 30}, {1, 40}, {2, 30}},                // Partly Cloudy → clear, stay, overcast
	2:  {{1, 20}, {2, 40}, {3, 25}, {11, 15}},      // Overcast → clearing, stay, rain, snow
	3:  {{2, 25}, {3, 40}, {4, 35}},                // Light Rain → clear, stay, moderate
	4:  {{3, 30}, {4, 35}, {5, 35}},                // Moderate Rain → lighter, stay, heavier
	5:  {{4, 35}, {5, 30}, {6, 35}},                // Heavy Rain → moderate, stay, storm
	6:  {{5, 40}, {2, 40}, {6, 20}},                // Thunderstorm → heavy rain, overcast, stay
	7:  {{5, 50}, {6, 30}, {7, 20}},                // Gale → heavy rain, storm, stay
	8:  {{7, 50}, {5, 30}, {8, 20}},                // Hurricane → gale, heavy rain, stay
	9:  {{2, 50}, {3, 30}, {9, 20}},                // Hail → overcast, rain, stay
	10: {{2, 40}, {11, 30}, {10, 30}},              // Sleet → overcast, flurries, stay
	11: {{2, 25}, {11, 35}, {12, 40}},              // Snow Flurries → overcast, stay, moderate
	12: {{11, 30}, {12, 35}, {13, 35}},             // Moderate Snow → flurries, stay, heavy
	13: {{12, 35}, {13, 35}, {14, 30}},             // Heavy Snow → moderate, stay, blizzard
	14: {{13, 50}, {14, 30}, {12, 20}},             // Blizzard → heavy, stay, moderate
}

// GetWeatherDesc returns a weather description for a given region.
func (e *GameEngine) GetWeatherDesc(region int) string {
	if e.RegionWeather == nil {
		return ""
	}
	state, ok := e.RegionWeather[region]
	if !ok {
		state = 0
	}
	if name, ok := WeatherNames[state]; ok {
		return name
	}
	return "Clear"
}

// weatherRoomDesc maps weather state IDs to immersive room description lines.
var weatherRoomDesc = map[int]string{
	1:  "A few white clouds drift lazily across an otherwise blue sky.",
	2:  "The sky above is heavy with gray clouds, blocking out the sun.",
	3:  "A light rain falls steadily, pattering softly on the ground.",
	4:  "A steady rain comes down, soaking everything in sight.",
	5:  "Heavy rain lashes down, driven by gusting winds.",
	6:  "Thunder rumbles overhead and lightning flickers through the roiling storm clouds.",
	7:  "A powerful gale tears through, bending trees and howling between every gap.",
	8:  "A furious hurricane rages, the wind a deafening roar and the rain nearly horizontal.",
	9:  "Chunks of ice clatter down from a bruised sky, bouncing off the ground in every direction.",
	10: "Wet sleet stings any exposed skin, half-rain half-ice, coating surfaces in a thin glaze.",
	11: "Light snowflakes drift down from gray clouds, dusting the ground in white.",
	12: "Snow falls steadily, blanketing everything in a deepening layer of white.",
	13: "Heavy snow swirls down in thick curtains, making it hard to see far.",
	14: "A howling blizzard rages, the world reduced to a churning wall of white.",
}

// GetRoomWeather returns a weather line for an outdoor room, or "" for indoor.
func (e *GameEngine) GetRoomWeather(roomNum int) string {
	room := e.rooms[roomNum]
	if room == nil {
		return ""
	}
	if !isOutdoorTerrain(room.Terrain) {
		return ""
	}
	region := room.Region
	state, ok := e.RegionWeather[region]
	if !ok {
		state = 0
	}
	if state == 0 { // Sunny — no extra line
		return ""
	}
	if desc, ok := weatherRoomDesc[state]; ok {
		return desc
	}
	return ""
}

// temperatureBySeason gives a baseline Fahrenheit temperature for each season.
var temperatureBySeason = map[string]int{
	"PSCRIPT": 55, // Spring
	"SSCRIPT": 85, // Summer
	"ASCRIPT": 55, // Autumn
	"WSCRIPT": 25, // Winter
}

// temperatureByPeriod adjusts the seasonal baseline for time of day (see TimePeriod).
var temperatureByPeriod = map[string]int{
	"midnight":            -12,
	"very early morning":  -10,
	"dawn":                -5,
	"mid morning":         0,
	"noon":                8,
	"afternoon":           6,
	"evening":             -2,
	"night":               -8,
}

// temperatureByWeather adjusts the temperature for the current weather state.
var temperatureByWeather = map[int]int{
	0: 5, 1: 2, 2: 0, 3: -3, 4: -5, 5: -7, 6: -8, 7: -10, 8: -12,
	9: -10, 10: -15, 11: -18, 12: -22, 13: -26, 14: -32,
}

// GetTemperature returns the current temperature (degrees Fahrenheit) for a region,
// derived from season, time of day, and the region's current weather state.
func (e *GameEngine) GetTemperature(region int) int {
	state := 0
	if e.RegionWeather != nil {
		if s, ok := e.RegionWeather[region]; ok {
			state = s
		}
	}
	return temperatureBySeason[GameSeason()] + temperatureByPeriod[TimePeriod()] + temperatureByWeather[state]
}

// TemperatureDesc returns a short descriptive word for a Fahrenheit temperature.
func TemperatureDesc(f int) string {
	switch {
	case f < 20:
		return "frigid"
	case f < 32:
		return "freezing"
	case f < 45:
		return "cold"
	case f < 60:
		return "cool"
	case f < 75:
		return "mild"
	case f < 85:
		return "warm"
	case f < 95:
		return "hot"
	default:
		return "sweltering"
	}
}

// isOutdoorTerrain returns true if the terrain type is outdoors.
func isOutdoorTerrain(terrain string) bool {
	switch terrain {
	case "FOREST", "MOUNTAIN", "PLAIN", "SWAMP", "JUNGLE",
		"WASTE", "OUTDOOR_OTHER", "OUTDOOR_FLOOR", "AERIAL":
		return true
	}
	return false
}

// snowStates are weather states that only make sense when it's cold enough for
// precipitation to fall as ice or snow rather than rain: Sleet, Snow Flurries,
// Moderate Snow, Heavy Snow, Blizzard.
var snowStates = map[int]bool{10: true, 11: true, 12: true, 13: true, 14: true}

// advanceWeather randomly transitions weather for all regions and broadcasts changes.
//
// The random walk previously had no concept of season at all, so a region could
// wander from Overcast into Snow Flurries and all the way up to Blizzard purely by
// chance regardless of what season it was — meanwhile GetTemperature is computed
// independently from season/time-of-day/weather-state, so in Summer (85°F baseline)
// it stayed a mild 55-65°F even while the weather type had drifted into "Blizzard."
// Players saw "It is snowing heavily" right next to "around 61 degrees." Snow-family
// states are now only reachable outside Summer.
func (e *GameEngine) advanceWeather() {
	if e.RegionWeather == nil {
		return
	}

	snowOK := GameSeason() != "SSCRIPT" // not Summer

	// Collect all regions that have outdoor rooms
	regionSet := make(map[int]bool)
	for _, room := range e.rooms {
		if room.Region > 0 {
			regionSet[room.Region] = true
		}
	}
	// Always include region 0 (default)
	regionSet[0] = true

	for region := range regionSet {
		oldState := e.RegionWeather[region]

		// Season just turned to Summer while this region was sitting in a snow
		// state (e.g. a lingering Blizzard from Spring) — clear it back to Overcast
		// rather than let it randomly walk deeper into snow in the middle of July.
		if !snowOK && snowStates[oldState] {
			newState := 2 // Overcast
			e.RegionWeather[region] = newState
			e.broadcastWeatherChange(region, oldState, newState)
			continue
		}

		transitions := weatherTransitions[oldState]
		if !snowOK {
			filtered := make([][2]int, 0, len(transitions))
			for _, t := range transitions {
				if !snowStates[t[0]] {
					filtered = append(filtered, t)
				}
			}
			transitions = filtered
		}
		if len(transitions) == 0 {
			continue
		}

		// Weighted random selection
		totalWeight := 0
		for _, t := range transitions {
			totalWeight += t[1]
		}
		roll := rand.Intn(totalWeight)
		newState := oldState
		for _, t := range transitions {
			roll -= t[1]
			if roll < 0 {
				newState = t[0]
				break
			}
		}

		if newState != oldState {
			e.RegionWeather[region] = newState
			e.broadcastWeatherChange(region, oldState, newState)
		}
	}
}

// broadcastWeatherChange announces a weather transition to all outdoor rooms
// in the given region, picking a scripted transition line if one exists for
// this exact oldState->newState pair, falling back to a generic line otherwise.
func (e *GameEngine) broadcastWeatherChange(region, oldState, newState int) {
	if e.localRoomBroadcast == nil {
		return
	}
	msg := weatherTransitionMessages[[2]int{oldState, newState}]
	if msg == "" {
		if newState == 0 {
			msg = "The skies clear."
		} else if desc, ok := weatherRoomDesc[newState]; ok {
			msg = desc
		} else {
			msg = "The weather shifts."
		}
	}
	for num, room := range e.rooms {
		if room.Region == region && isOutdoorTerrain(room.Terrain) {
			e.localRoomBroadcast(num, []string{msg})
		}
	}
}

// weatherValueList returns a comma-separated "id=Name" listing of all known
// weather states, in ID order, for GM command usage/error text.
func weatherValueList() string {
	ids := make([]int, 0, len(WeatherNames))
	for id := range WeatherNames {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("%d=%s", id, WeatherNames[id])
	}
	return strings.Join(parts, ", ")
}

// gmWeather handles @weather [value] — shows the current weather (and
// temperature) for the GM's region with no args, or sets the region's
// weather state and broadcasts the transition when given a value.
func (e *GameEngine) gmWeather(player *Player, args []string) *CommandResult {
	room := e.rooms[player.RoomNumber]
	region := 0
	if room != nil {
		region = room.Region
	}

	if len(args) == 0 {
		state := 0
		if e.RegionWeather != nil {
			state = e.RegionWeather[region]
		}
		temp := e.GetTemperature(region)
		return &CommandResult{Messages: []string{
			fmt.Sprintf("Region %d weather: %d (%s). Temperature: %d degrees (%s).",
				region, state, WeatherNames[state], temp, TemperatureDesc(temp)),
		}}
	}

	newState, err := strconv.Atoi(args[0])
	if err != nil {
		return &CommandResult{Messages: []string{"Usage: @weather [value]. Valid values: " + weatherValueList()}}
	}
	name, ok := WeatherNames[newState]
	if !ok {
		return &CommandResult{Messages: []string{"Invalid weather value. Valid values: " + weatherValueList()}}
	}

	if e.RegionWeather == nil {
		e.RegionWeather = make(map[int]int)
	}
	oldState := e.RegionWeather[region]
	e.RegionWeather[region] = newState
	if newState != oldState {
		e.broadcastWeatherChange(region, oldState, newState)
	}

	return &CommandResult{Messages: []string{fmt.Sprintf("Region %d weather set to %d (%s).", region, newState, name)}}
}

// doWeather handles the WEATHER command, showing the player the current
// conditions and temperature for their region.
func (e *GameEngine) doWeather(player *Player) *CommandResult {
	room := e.rooms[player.RoomNumber]
	region := 0
	if room != nil {
		region = room.Region
	}
	temp := e.GetTemperature(region)
	tempDesc := TemperatureDesc(temp)

	if room == nil || !isOutdoorTerrain(room.Terrain) {
		return &CommandResult{Messages: []string{
			fmt.Sprintf("You can't see the sky from here, but it feels %s, around %d degrees.", tempDesc, temp),
		}}
	}

	state, ok := e.RegionWeather[region]
	if !ok {
		state = 0
	}
	msgs := []string{fmt.Sprintf("The weather is currently %s.", WeatherNames[state])}
	if desc, ok := weatherRoomDesc[state]; ok {
		msgs = append(msgs, desc)
	}
	msgs = append(msgs, fmt.Sprintf("It feels %s, around %d degrees.", tempDesc, temp))
	return &CommandResult{Messages: msgs}
}

// StartWeatherCycle starts a background goroutine that changes weather periodically.
func (e *GameEngine) StartWeatherCycle() {
	go func() {
		// Weather changes every 5-10 game hours (5-10 real minutes)
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			e.advanceWeather()
		}
	}()
}
