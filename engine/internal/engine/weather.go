package engine

// WeatherNames maps weather state IDs to display names.
var WeatherNames = map[int]string{
	0: "Sunny", 1: "Partly Cloudy", 2: "Overcast", 3: "Light Rain",
	4: "Rain", 5: "Heavy Rain", 6: "Thunderstorm", 7: "Fog",
	8: "Light Snow", 9: "Snow", 10: "Heavy Snow", 11: "Sleet",
	12: "Hail", 13: "Blizzard", 14: "Hurricane",
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

// GetRoomWeather returns a weather line for an outdoor room, or "" for indoor.
func (e *GameEngine) GetRoomWeather(roomNum int) string {
	room := e.rooms[roomNum]
	if room == nil {
		return ""
	}
	if !isOutdoorTerrain(room.Terrain) {
		return ""
	}
	// Find region for the room (stored in room modifiers or default 0)
	region := 0 // default region
	desc := e.GetWeatherDesc(region)
	if desc == "" || desc == "Sunny" || desc == "Clear" {
		return ""
	}
	return "The weather is " + desc + "."
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
