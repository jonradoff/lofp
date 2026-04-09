package gameworld

// ParsedData holds all data loaded from script files, used to initialize the engine.
type ParsedData struct {
	Rooms        []Room
	Items        []ItemDef
	Monsters     []MonsterDef
	Nouns        []NounDef
	Adjectives   []AdjDef
	MonsterAdjs  []MonsterAdjDef
	Variables    []Variable
	Regions      []Region
	MonsterLists         []MonsterList
	SeasonalMonsterLists map[string][]MonsterList // "PSCRIPT"/"SSCRIPT"/"ASCRIPT"/"WSCRIPT" -> seasonal MLISTs
	SeasonalRooms        map[string][]Room        // seasonal room description overrides
	CEvents      []CEvent
	MoneyDefs    []MoneyDef
	ForageDefs   []ForageDef
	MineDefs     []MineDef
	StartRoom    int
	BumpRoom     int
}
