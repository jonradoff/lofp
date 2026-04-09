package scriptparser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// ParseResult holds all data parsed from script files.
type ParseResult struct {
	Rooms       []gameworld.Room
	Items       []gameworld.ItemDef
	Monsters    []gameworld.MonsterDef
	Nouns       []gameworld.NounDef
	Adjectives  []gameworld.AdjDef
	MonsterAdjs []gameworld.MonsterAdjDef
	Variables   []gameworld.Variable
	Regions     []gameworld.Region
	MonsterLists         []gameworld.MonsterList
	SeasonalMonsterLists map[string][]gameworld.MonsterList // "PSCRIPT" -> spring MLISTs, etc.
	SeasonalRooms        map[string][]gameworld.Room        // seasonal room description overrides
	CEvents     []gameworld.CEvent
	MoneyDefs   []gameworld.MoneyDef
	ForageDefs  []gameworld.ForageDef
	MineDefs    []gameworld.MineDef
	StartRoom   int
	BumpRoom    int
}

// ParseConfig reads LEGENDS.CFG and loads all referenced script files.
// resolveFileCaseInsensitive finds a file by name with case-insensitive matching.
// The original game ran on MS-DOS (case-insensitive), so script filenames in
// LEGENDS.CFG may not match the actual file case on disk.
func resolveFileCaseInsensitive(path string) string {
	// Try exact path first
	if _, err := os.Stat(path); err == nil {
		return path
	}
	// Scan the directory for a case-insensitive match
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return path // can't read dir, return original
	}
	for _, e := range entries {
		if strings.EqualFold(e.Name(), base) {
			return filepath.Join(dir, e.Name())
		}
	}
	return path // no match found, return original (will error on open)
}

func ParseConfig(configPath string) (*ParseResult, error) {
	result := &ParseResult{
		StartRoom: 3950,
		BumpRoom:  201,
	}

	configPath = resolveFileCaseInsensitive(configPath)
	f, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	dir := filepath.Dir(configPath)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		cmd := strings.ToUpper(fields[0])
		switch cmd {
		case "STARTROOM":
			result.StartRoom, _ = strconv.Atoi(fields[1])
		case "BUMPROOM":
			result.BumpRoom, _ = strconv.Atoi(fields[1])
		case "SCRIPT":
			scriptFile := resolveFileCaseInsensitive(filepath.Join(dir, fields[1]))
			if err := parseScriptFile(scriptFile, result); err != nil {
				fmt.Printf("Warning: could not parse %s: %v\n", fields[1], err)
			}
		case "ASCRIPT", "WSCRIPT", "SSCRIPT", "PSCRIPT":
			// Parse seasonal scripts into separate per-season collections.
			// The engine selects the active season at runtime based on game calendar.
			seasonResult := &ParseResult{}
			scriptFile := resolveFileCaseInsensitive(filepath.Join(dir, fields[1]))
			if err := parseScriptFile(scriptFile, seasonResult); err != nil {
				fmt.Printf("Warning: could not parse seasonal %s: %v\n", fields[1], err)
			} else {
				if result.SeasonalMonsterLists == nil {
					result.SeasonalMonsterLists = make(map[string][]gameworld.MonsterList)
				}
				if result.SeasonalRooms == nil {
					result.SeasonalRooms = make(map[string][]gameworld.Room)
				}
				result.SeasonalMonsterLists[cmd] = append(result.SeasonalMonsterLists[cmd], seasonResult.MonsterLists...)
				result.SeasonalRooms[cmd] = append(result.SeasonalRooms[cmd], seasonResult.Rooms...)
				// Monsters/items/etc from seasonal scripts go into the main collections
				result.Monsters = append(result.Monsters, seasonResult.Monsters...)
				result.Items = append(result.Items, seasonResult.Items...)
			}
		}
	}

	// Deduplicate: when multiple definitions share the same number,
	// keep the last one (later script files override earlier ones,
	// e.g. ANTI.SCR placeholders get replaced by real ITEM*.SCR defs).
	result.Items = deduplicateItems(result.Items)
	result.Monsters = deduplicateMonsters(result.Monsters)
	result.Rooms = deduplicateRooms(result.Rooms)

	return result, scanner.Err()
}

func deduplicateItems(items []gameworld.ItemDef) []gameworld.ItemDef {
	seen := make(map[int]int) // number -> index in result
	var out []gameworld.ItemDef
	for _, item := range items {
		if idx, ok := seen[item.Number]; ok {
			out[idx] = item // replace with later definition
		} else {
			seen[item.Number] = len(out)
			out = append(out, item)
		}
	}
	return out
}

func deduplicateMonsters(monsters []gameworld.MonsterDef) []gameworld.MonsterDef {
	seen := make(map[int]int)
	var out []gameworld.MonsterDef
	for _, mon := range monsters {
		if idx, ok := seen[mon.Number]; ok {
			out[idx] = mon
		} else {
			seen[mon.Number] = len(out)
			out = append(out, mon)
		}
	}
	return out
}

func deduplicateRooms(rooms []gameworld.Room) []gameworld.Room {
	seen := make(map[int]int)
	var out []gameworld.Room
	for _, room := range rooms {
		if idx, ok := seen[room.Number]; ok {
			out[idx] = room
		} else {
			seen[room.Number] = len(out)
			out = append(out, room)
		}
	}
	return out
}

func parseScriptFile(path string, result *ParseResult) error {
	path = resolveFileCaseInsensitive(path)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	filename := filepath.Base(path)
	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	p := &fileParser{
		lines:    lines,
		pos:      0,
		filename: filename,
		result:   result,
	}
	p.parse()
	return nil
}

type fileParser struct {
	lines    []string
	pos      int
	filename string
	result   *ParseResult
}

func (p *fileParser) parse() {
	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])
		if line == "" || strings.HasPrefix(line, ";") {
			p.pos++
			continue
		}
		fields := strings.Fields(line)
		cmd := strings.ToUpper(fields[0])

		switch cmd {
		case "NUMBER":
			p.parseRoom(fields)
		case "INUMBER":
			p.parseItem(fields)
		case "MNUMBER":
			p.parseMonster(fields)
		case "NOUNDEF":
			if len(fields) >= 3 {
				id, _ := strconv.Atoi(fields[1])
				result := gameworld.NounDef{ID: id, Name: strings.Join(fields[2:], " ")}
				p.result.Nouns = append(p.result.Nouns, result)
			}
			p.pos++
		case "ADJDEF":
			if len(fields) >= 3 {
				id, _ := strconv.Atoi(fields[1])
				// Handle case like "ADJDEF 1aelfen" (no space)
				name := strings.Join(fields[2:], " ")
				p.result.Adjectives = append(p.result.Adjectives, gameworld.AdjDef{ID: id, Name: name})
			} else if len(fields) == 2 {
				// Try to parse "ADJDEF 1aelfen" format
				s := fields[1]
				i := 0
				for i < len(s) && (s[i] >= '0' && s[i] <= '9') {
					i++
				}
				if i > 0 && i < len(s) {
					id, _ := strconv.Atoi(s[:i])
					p.result.Adjectives = append(p.result.Adjectives, gameworld.AdjDef{ID: id, Name: s[i:]})
				}
			}
			p.pos++
		case "MADJDEF":
			if len(fields) >= 3 {
				id, _ := strconv.Atoi(fields[1])
				p.result.MonsterAdjs = append(p.result.MonsterAdjs, gameworld.MonsterAdjDef{ID: id, Name: strings.Join(fields[2:], " ")})
			}
			p.pos++
		case "VARIABLE":
			if len(fields) >= 2 {
				p.result.Variables = append(p.result.Variables, gameworld.Variable{Name: fields[1]})
			}
			p.pos++
		case "REGIONDEF":
			p.parseRegion(fields)
			p.pos++
		case "MLIST":
			if len(fields) >= 5 {
				room, _ := strconv.Atoi(fields[1])
				mid, _ := strconv.Atoi(fields[2])
				prob, _ := strconv.Atoi(fields[3])
				maxCount, _ := strconv.Atoi(fields[4])
				p.result.MonsterLists = append(p.result.MonsterLists, gameworld.MonsterList{
					Room: room, MonsterID: mid, Probability: prob, MaxCount: maxCount,
				})
			}
			p.pos++
		case "MONEYDEF":
			if len(fields) >= 5 {
				id, _ := strconv.Atoi(fields[1])
				name := fields[2]
				rate, _ := strconv.Atoi(fields[3])
				item, _ := strconv.Atoi(fields[4])
				p.result.MoneyDefs = append(p.result.MoneyDefs, gameworld.MoneyDef{ID: id, Name: name, ExchangeRate: rate, ItemNum: item})
			}
			p.pos++
		case "FORAGEDEF":
			if len(fields) >= 7 {
				terrain := strings.ToUpper(fields[1])
				itemNum, _ := strconv.Atoi(fields[2])
				adjNum, _ := strconv.Atoi(fields[3])
				ratio, _ := strconv.Atoi(fields[4])
				v2, _ := strconv.Atoi(fields[5])
				v5, _ := strconv.Atoi(fields[6])
				p.result.ForageDefs = append(p.result.ForageDefs, gameworld.ForageDef{
					Terrain: terrain, ItemNum: itemNum, AdjNum: adjNum, Ratio: ratio, Val2: v2, Val5: v5,
				})
			}
			p.pos++
		case "MINDEF":
			if len(fields) >= 6 {
				itemNum, _ := strconv.Atoi(fields[1])
				adjNum, _ := strconv.Atoi(fields[2])
				grade := strings.ToUpper(fields[3])
				value, _ := strconv.Atoi(fields[4])
				v2, _ := strconv.Atoi(fields[5])
				p.result.MineDefs = append(p.result.MineDefs, gameworld.MineDef{
					ItemNum: itemNum, AdjNum: adjNum, Grade: grade, Value: value, Val2: v2,
				})
			}
			p.pos++
		default:
			p.pos++
		}
	}
}

func (p *fileParser) parseRoom(fields []string) {
	if len(fields) < 2 {
		p.pos++
		return
	}
	num, _ := strconv.Atoi(fields[1])
	room := gameworld.Room{
		Number:           num,
		Exits:            make(map[string]int),
		ItemDescriptions: make(map[string]string),
		SourceFile:       p.filename,
	}
	p.pos++

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])
		if line == "" || strings.HasPrefix(line, ";") {
			p.pos++
			continue
		}
		fields := strings.Fields(line)
		cmd := strings.ToUpper(fields[0])

		// A new NUMBER or INUMBER or MNUMBER starts a new block
		if cmd == "NUMBER" || cmd == "INUMBER" || cmd == "MNUMBER" {
			break
		}

		switch cmd {
		case "NAME":
			room.Name = strings.Join(fields[1:], " ")
		case "*DESCRIPTION_START":
			if len(fields) >= 3 && strings.ToUpper(fields[1]) == "ITEM" {
				action, ref := p.parseItemDescArgs(fields[2:])
				desc := p.readDescription()
				if action != "" && ref != "" {
					room.ItemDescriptions[action+":"+ref] = desc
				}
			} else if len(fields) > 1 {
				// Other non-room description variant; consume and discard
				p.readDescription()
			} else {
				room.Description = p.readDescription()
			}
			continue // readDescription advances pos
		case "EXIT":
			if len(fields) >= 3 {
				dir := strings.ToUpper(fields[1])
				dest, _ := strconv.Atoi(fields[2])
				room.Exits[dir] = dest
			}
		case "ITEM":
			item := p.parseRoomItem(fields, false)
			room.Items = append(room.Items, item)
		case "PUT":
			item := p.parseRoomItem(fields, true)
			room.Items = append(room.Items, item)
		case "EXTEND":
			if len(fields) >= 3 {
				ref, _ := strconv.Atoi(fields[1])
				ext := strings.Join(fields[2:], " ")
				for i := range room.Items {
					if room.Items[i].Ref == ref && !room.Items[i].IsPut {
						room.Items[i].Extend = ext
					}
				}
			}
		case "STOREITEM":
			if len(fields) >= 4 {
				archetype, _ := strconv.Atoi(fields[1])
				adj, _ := strconv.Atoi(fields[2])
				price, _ := strconv.Atoi(fields[3])
				room.StoreItems = append(room.StoreItems, gameworld.StoreItem{
					Archetype: archetype, Adj: adj, Price: price,
				})
			}
		case "TRAINING":
			if len(fields) >= 3 {
				skillID, _ := strconv.Atoi(fields[1])
				maxLevel, _ := strconv.Atoi(fields[2])
				room.TrainingSkills = append(room.TrainingSkills, gameworld.TrainingDef{
					SkillID: skillID, MaxLevel: maxLevel,
				})
			}
		case "MONSTER_GROUP":
			if len(fields) >= 2 {
				room.MonsterGroup, _ = strconv.Atoi(fields[1])
			}
		case "INDOOR_FLOOR", "INDOOR_GROUND", "CAVE", "DEEPCAVE",
			"FOREST", "MOUNTAIN", "PLAIN", "SWAMP", "JUNGLE",
			"WASTE", "OUTDOOR_OTHER", "OUTDOOR_FLOOR", "AERIAL",
			"ASTRAL", "UNDERSEA":
			room.Terrain = cmd
		case "FIXED_LIGHT", "DARKNESS", "PARTIAL_DARKNESS",
			"DAY_LIGHT", "OBSCURED_DAY_LIGHT", "LIMITED_VISION":
			room.Lighting = cmd
		case "REGION":
			if len(fields) >= 2 {
				room.Region, _ = strconv.Atoi(fields[1])
			}
		case "FORGE", "LOOM", "MINEA", "MINEB", "MINEC",
			"BUY_ARMOR", "BUY_SKINS", "BUY_JEWELRY", "SUBMERGED",
			"MOVEMENT_ASTRAL":
			room.Modifiers = append(room.Modifiers, cmd)
		case "IFVERB", "IFPREVERB", "IFVERB2", "IFPREVERB2",
			"IFITEM", "IFTOUCH", "IFVAR", "IFNOITEM",
			"IFFULLDESC", "IFENTRY", "IFSEEK", "IFSAY",
			"IFCARRY":
			block := p.parseScriptBlock(fields)
			room.Scripts = append(room.Scripts, block)
			continue
		case "VARIABLE":
			// Room-scoped variable, track it
			if len(fields) >= 2 {
				p.result.Variables = append(p.result.Variables, gameworld.Variable{Name: fields[1]})
			}
		case "CALL":
			// CALL macro — requires loading MACRO definitions from ROOMX.SCR/ROOMY.SCR.
			// Not yet implemented; left as-is for now.
		case "CEVENT":
			// CEVENT <id> <cycles> <room#>
			if len(fields) >= 4 {
				id, _ := strconv.Atoi(fields[1])
				cycles, _ := strconv.Atoi(fields[2])
				roomNum, _ := strconv.Atoi(fields[3])
				ce := gameworld.CEvent{ID: id, Cycles: cycles, Room: roomNum}
				p.pos++
				// Parse script blocks inside the CEVENT
				for p.pos < len(p.lines) {
					cline := strings.TrimSpace(p.lines[p.pos])
					if cline == "" || strings.HasPrefix(cline, ";") {
						p.pos++
						continue
					}
					cf := strings.Fields(cline)
					cc := strings.ToUpper(cf[0])
					if cc == "NUMBER" || cc == "INUMBER" || cc == "MNUMBER" || cc == "CEVENT" {
						break
					}
					if strings.HasPrefix(cc, "IF") {
						block := p.parseScriptBlock(cf)
						ce.Scripts = append(ce.Scripts, block)
						continue
					}
					if cc == "CALL" || cc == "ECHO" || cc == "AFFECT" || cc == "RANDOM" || cc == "EQUAL" || cc == "ADD" || cc == "SUB" {
						ce.Scripts = append(ce.Scripts, gameworld.ScriptBlock{
							Type: "ACTION", Actions: []gameworld.ScriptAction{{Command: cc, Args: cf[1:]}},
						})
					}
					p.pos++
				}
				p.result.CEvents = append(p.result.CEvents, ce)
				continue
			}
			p.pos++
			continue
		}
		p.pos++
	}

	p.result.Rooms = append(p.result.Rooms, room)
}

func (p *fileParser) parseItem(fields []string) {
	if len(fields) < 2 {
		p.pos++
		return
	}
	num, _ := strconv.Atoi(fields[1])
	item := gameworld.ItemDef{
		Number:     num,
		Article:    "A",
		Type:       "MISC",
		SourceFile: p.filename,
	}
	p.pos++

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])
		if line == "" || strings.HasPrefix(line, ";") {
			p.pos++
			continue
		}
		fields := strings.Fields(line)
		cmd := strings.ToUpper(fields[0])

		if cmd == "NUMBER" || cmd == "INUMBER" || cmd == "MNUMBER" {
			break
		}

		switch cmd {
		case "NAME":
			if len(fields) >= 2 {
				item.NameID, _ = strconv.Atoi(fields[1])
			}
		case "WEIGHT":
			if len(fields) >= 2 {
				item.Weight, _ = strconv.Atoi(fields[1])
			}
		case "VOLUME":
			if len(fields) >= 2 {
				item.Volume, _ = strconv.Atoi(fields[1])
			}
		case "PARAMETER1":
			if len(fields) >= 2 {
				item.Parameter1, _ = strconv.Atoi(fields[1])
			}
		case "PARAMETER2":
			if len(fields) >= 2 {
				item.Parameter2, _ = strconv.Atoi(fields[1])
			}
		case "PARAMETER3":
			if len(fields) >= 2 {
				item.Parameter3, _ = strconv.Atoi(fields[1])
			}
		case "ARTICLE":
			if len(fields) >= 2 {
				item.Article = strings.ToUpper(fields[1])
			}
		case "INTERIOR":
			if len(fields) >= 2 {
				item.Interior, _ = strconv.Atoi(fields[1])
			}
		case "CONTAINER":
			if len(fields) >= 2 {
				item.Container = strings.ToUpper(fields[1])
			}
		case "SUBSTANCE":
			if len(fields) >= 2 {
				item.Substance = strings.ToUpper(fields[1])
			}
		case "*DESCRIPTION_START":
			// Items can have descriptions too (rare)
			p.readDescription()
			continue
		// Item types
		case "AMMO", "ARMOR", "BITE_WEAPON", "BOW_WEAPON", "CLAW_WEAPON",
			"CRUSH_WEAPON", "DRAKIN_CRUSH", "DRAKIN_POLE", "DRAKIN_SLASH",
			"DRAKIN_THROWN", "FOOD", "HANDGUN", "KEY", "LIQCONTAINER",
			"LIQUID", "LOCKPICK", "MINETOOL", "MISC", "MONEY",
			"POLE_WEAPON", "POLETHROWN", "PORTAL_THROUGH", "PORTAL_CLIMB",
			"PORTAL_UP", "PORTAL_DOWN", "PORTAL_CLIMBUP", "PORTAL_CLIMBDOWN",
			"PORTAL_OVER", "PORTAL", "PUNCTURE_WEAPON",
			"RIFLE", "SCROLL", "SHIELD", "SLASH_WEAPON", "STABTHROWN",
			"THROWN_WEAPON", "TRAP", "TWOHAND_WEAPON", "ORE":
			item.Type = cmd
		// Worn slots
		case "WORN_AROUND", "WORN_BACK", "WORN_BODY", "WORN_DON",
			"WORN_EAR", "WORN_FEET1", "WORN_FEET2", "WORN_HAIR",
			"WORN_HANDS", "WORN_HEAD", "WORN_NECK", "WORN_RING",
			"WORN_TORSO1", "WORN_TORSO2", "WORN_TORSO3",
			"WORN_TRUNK1", "WORN_TRUNK2", "WORN_WRIST", "WORN_ARMOR":
			item.WornSlot = cmd
		// Flags
		case "CRAFTABLE", "DYEABLE", "ENCRUSTABLE", "DYE", "FLIPABLE",
			"HIDDEN", "LATCHABLE", "LIGHTABLE", "LOCKABLE", "OPENABLE",
			"REAGENT", "SKIN", "TURNABLE", "SEALED", "MATERIAL2":
			item.Flags = append(item.Flags, cmd)
		case "IFVERB", "IFPREVERB", "IFVERB2", "IFPREVERB2",
			"IFITEM", "IFTOUCH", "IFVAR", "IFNOITEM",
			"IFSEEK", "IFSAY", "IFCARRY":
			block := p.parseScriptBlock(fields)
			item.Scripts = append(item.Scripts, block)
			continue
		case "VARIABLE":
			if len(fields) >= 2 {
				p.result.Variables = append(p.result.Variables, gameworld.Variable{Name: fields[1]})
			}
		}
		p.pos++
	}

	p.result.Items = append(p.result.Items, item)
}

func (p *fileParser) parseMonster(fields []string) {
	if len(fields) < 2 {
		p.pos++
		return
	}
	num, _ := strconv.Atoi(fields[1])
	mon := gameworld.MonsterDef{
		Number:     num,
		Speed:      3,
		Gender:     2,
		SourceFile: p.filename,
	}
	p.pos++

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])
		if line == "" || strings.HasPrefix(line, ";") {
			p.pos++
			continue
		}
		fields := strings.Fields(line)
		cmd := strings.ToUpper(fields[0])

		if cmd == "NUMBER" || cmd == "INUMBER" || cmd == "MNUMBER" {
			break
		}

		switch cmd {
		case "NAME":
			mon.Name = strings.Join(fields[1:], " ")
		case "UNIQUE":
			mon.Unique = true
		case "ADJECTIVE":
			if len(fields) >= 2 {
				mon.Adjective, _ = strconv.Atoi(fields[1])
			}
		case "HUMAN", "ANIMAL", "AVINE":
			mon.BodyType = cmd
		case "BODY":
			if len(fields) >= 2 {
				mon.Body, _ = strconv.Atoi(fields[1])
			}
		case "ATTACK1":
			if len(fields) >= 2 {
				mon.Attack1, _ = strconv.Atoi(fields[1])
			}
		case "ATTACK2":
			if len(fields) >= 2 {
				mon.Attack2, _ = strconv.Atoi(fields[1])
			}
		case "DEFENSE":
			if len(fields) >= 2 {
				mon.Defense, _ = strconv.Atoi(fields[1])
			}
		case "STRATEGY":
			if len(fields) >= 2 {
				mon.Strategy, _ = strconv.Atoi(fields[1])
			}
		case "TREASURE":
			if len(fields) >= 2 {
				mon.Treasure, _ = strconv.Atoi(fields[1])
			}
		case "SPEED":
			if len(fields) >= 2 {
				mon.Speed, _ = strconv.Atoi(fields[1])
			}
		case "ARMOR":
			if len(fields) >= 2 {
				mon.Armor, _ = strconv.Atoi(fields[1])
			}
		case "RACE":
			if len(fields) >= 2 {
				mon.Race, _ = strconv.Atoi(fields[1])
			}
		case "GENDER":
			if len(fields) >= 2 {
				mon.Gender, _ = strconv.Atoi(fields[1])
			}
		case "ALIGNMENT":
			if len(fields) >= 2 { mon.Alignment, _ = strconv.Atoi(fields[1]) }
		case "RESIST":
			if len(fields) >= 2 { mon.MagicResist, _ = strconv.Atoi(fields[1]) }
		case "MANA":
			if len(fields) >= 2 { mon.Mana, _ = strconv.Atoi(fields[1]) }
		case "SPELLUSE":
			if len(fields) >= 2 { mon.SpellUse, _ = strconv.Atoi(fields[1]) }
		case "SPELLSKILL":
			if len(fields) >= 2 { mon.SpellSkill, _ = strconv.Atoi(fields[1]) }
		case "CASTLEVEL":
			if len(fields) >= 2 { mon.CastLevel, _ = strconv.Atoi(fields[1]) }
		case "HIDESKILL":
			if len(fields) >= 2 { mon.HideSkill, _ = strconv.Atoi(fields[1]) }
		case "GUARD":
			if len(fields) >= 2 { mon.GuardItem, _ = strconv.Atoi(fields[1]) }
		case "STEALABLE":
			mon.Stealable = true
		case "ETERNAL":
			mon.Eternal = true
		case "DISCORPORATE":
			mon.Discorporate = true
		case "POISON":
			if len(fields) >= 3 {
				mon.PoisonChance, _ = strconv.Atoi(fields[1])
				mon.PoisonLevel, _ = strconv.Atoi(fields[2])
			}
		case "DISEASE":
			if len(fields) >= 3 {
				mon.DiseaseChance, _ = strconv.Atoi(fields[1])
				mon.DiseaseLevel, _ = strconv.Atoi(fields[2])
			}
		case "SKINADJ":
			if len(fields) >= 2 { mon.SkinAdj, _ = strconv.Atoi(fields[1]) }
		case "SKINITEM":
			if len(fields) >= 2 {
				mon.SkinItem, _ = strconv.Atoi(fields[1])
				sd := gameworld.SkinDrop{Archetype: mon.SkinItem}
				if len(fields) >= 3 { sd.Probability, _ = strconv.Atoi(fields[2]) }
				if len(fields) >= 4 { sd.Value, _ = strconv.Atoi(fields[3]) }
				if len(fields) >= 5 { sd.Magic, _ = strconv.Atoi(fields[4]) }
				if sd.Probability <= 0 { sd.Probability = 10 }
				mon.SkinItems = append(mon.SkinItems, sd)
			}
		case "IMMUNITY":
			if len(fields) >= 3 {
				if mon.Immunities == nil {
					mon.Immunities = make(map[int]int)
				}
				itype, _ := strconv.Atoi(fields[1])
				ilevel, _ := strconv.Atoi(fields[2])
				mon.Immunities[itype] = ilevel
			}
		case "WEAPON":
			if len(fields) >= 4 {
				arch, _ := strconv.Atoi(fields[1])
				adj, _ := strconv.Atoi(fields[2])
				prob, _ := strconv.Atoi(fields[3])
				mon.Weapons = append(mon.Weapons, gameworld.MonsterWeapon{Archetype: arch, Adj: adj, Probability: prob})
			}
		case "WEAPONPLUS":
			if len(fields) >= 2 {
				mon.WeaponPlus, _ = strconv.Atoi(fields[1])
			}
		case "MAGICWEAPON":
			if len(fields) >= 2 {
				mon.MagicWeapon, _ = strconv.Atoi(fields[1])
			}
		case "SPECUSE":
			if len(fields) >= 2 {
				mon.SpecUse, _ = strconv.Atoi(fields[1])
			}
		case "SPECUSES":
			if len(fields) >= 2 {
				mon.SpecUses, _ = strconv.Atoi(fields[1])
			}
		case "SPECBASE":
			if len(fields) >= 2 {
				mon.SpecBase, _ = strconv.Atoi(fields[1])
			}
		case "SPECDMG":
			if len(fields) >= 2 {
				mon.SpecDmg, _ = strconv.Atoi(fields[1])
			}
		case "SPECDMGTYPE":
			if len(fields) >= 2 {
				mon.SpecDmgType = strings.ToUpper(fields[1])
			}
		case "EXTRABODY":
			if len(fields) >= 2 {
				mon.ExtraBody, _ = strconv.Atoi(fields[1])
			}
		case "NONDISRUPTABLE":
			mon.NonDisruptable = true
		case "SILENCEIGNORE":
			mon.SilenceIgnore = true
		case "FATIGUE":
			if len(fields) >= 3 {
				mon.FatigueChance, _ = strconv.Atoi(fields[1])
				mon.FatigueLevel, _ = strconv.Atoi(fields[2])
			}
		case "SPELL":
			if len(fields) >= 2 {
				spellID, _ := strconv.Atoi(fields[1])
				mon.Spells = append(mon.Spells, spellID)
			}
		case "PSI":
			if len(fields) >= 2 { mon.Psi, _ = strconv.Atoi(fields[1]) }
		case "PSIUSE":
			if len(fields) >= 2 { mon.PsiUse, _ = strconv.Atoi(fields[1]) }
		case "PSISKILL":
			if len(fields) >= 2 { mon.PsiSkill, _ = strconv.Atoi(fields[1]) }
		case "PSIRESIST":
			if len(fields) >= 2 { mon.PsiResist, _ = strconv.Atoi(fields[1]) }
		case "PSILEVEL":
			if len(fields) >= 2 { mon.PsiLevel, _ = strconv.Atoi(fields[1]) }
		case "DISCIPLINE":
			if len(fields) >= 2 {
				disc, _ := strconv.Atoi(fields[1])
				mon.Disciplines = append(mon.Disciplines, disc)
			}
		case "TEXA", "TEXB", "TEXC", "TEXD", "TEXE", "TEXF", "TEXG", "TEXH",
			"TEXI", "TEXL", "TEXM", "TEXQ", "TEXR", "TEXTS", "TEXS", "TEXV", "TEXZ",
			"TEX1", "TEX2", "TEX3", "TEX4":
			if len(fields) >= 2 {
				if mon.TextOverrides == nil { mon.TextOverrides = make(map[string]string) }
				mon.TextOverrides[cmd] = strings.Join(fields[1:], " ")
			}
		case "*DESCRIPTION_START":
			mon.Description = p.readDescription()
			continue
		case "IFVERB", "IFPREVERB", "IFVERB2", "IFPREVERB2",
			"IFITEM", "IFTOUCH", "IFVAR", "IFENTRY", "IFSAY":
			block := p.parseScriptBlock(fields)
			mon.Scripts = append(mon.Scripts, block)
			continue
		}
		p.pos++
	}

	p.result.Monsters = append(p.result.Monsters, mon)
}

// parseItemDescArgs parses the args after "*DESCRIPTION_START ITEM", handling:
//   EXAM 0, READ 5, IN 3, ON 2, UNDER 1, BEHIND 0  (verb ref)
//   0 EXAM, 1 IN, 1 READ                            (ref verb - reversed)
//   4                                                (bare ref - defaults to EXAMINE)
// Returns normalized (action, ref) pair.
func (p *fileParser) parseItemDescArgs(args []string) (string, string) {
	normalizeVerb := func(v string) string {
		v = strings.ToUpper(v)
		switch v {
		case "EXAM", "EXAMINE", "EXMAINE":
			return "EXAMINE"
		case "READ", "IN", "ON", "UNDER", "BEHIND":
			return v
		}
		return ""
	}

	if len(args) == 1 {
		// Bare number: treat as EXAMINE
		if _, err := strconv.Atoi(args[0]); err == nil {
			return "EXAMINE", args[0]
		}
		return "", ""
	}

	// Try verb-ref order first
	if verb := normalizeVerb(args[0]); verb != "" {
		return verb, args[1]
	}
	// Try ref-verb order
	if _, err := strconv.Atoi(args[0]); err == nil {
		if verb := normalizeVerb(args[1]); verb != "" {
			return verb, args[0]
		}
	}
	return "", ""
}

func (p *fileParser) readDescription() string {
	p.pos++ // skip *DESCRIPTION_START
	var rawLines []string
	for p.pos < len(p.lines) {
		if strings.ToUpper(strings.TrimSpace(p.lines[p.pos])) == "*DESCRIPTION_END" {
			p.pos++
			break
		}
		rawLines = append(rawLines, p.lines[p.pos])
		p.pos++
	}

	// Detect if this is formatted text: any line has leading whitespace or
	// any line is blank (intentional spacing between stanzas, etc.)
	isFormatted := false
	for _, line := range rawLines {
		if line == "" || (len(line) > 0 && (line[0] == ' ' || line[0] == '\t')) {
			isFormatted = true
			break
		}
	}

	if isFormatted {
		return strings.Join(rawLines, "\n")
	}
	// Plain prose: trim and join into a paragraph
	var trimmed []string
	for _, line := range rawLines {
		trimmed = append(trimmed, strings.TrimSpace(line))
	}
	return strings.Join(trimmed, " ")
}

func (p *fileParser) parseRoomItem(fields []string, isPut bool) gameworld.RoomItem {
	item := gameworld.RoomItem{IsPut: isPut}
	if len(fields) >= 2 {
		item.Ref, _ = strconv.Atoi(fields[1])
		if isPut {
			item.PutIn = item.Ref
		}
	}
	if len(fields) >= 3 {
		item.Archetype, _ = strconv.Atoi(fields[2])
	}
	if len(fields) <= 3 {
		return item
	}
	// Parse additional params: ADJ1=x VAL2=y OPEN LOCKED etc.
	for _, f := range fields[3:] {
		upper := strings.ToUpper(f)
		if strings.HasPrefix(upper, "ADJ1=") {
			item.Adj1, _ = strconv.Atoi(f[5:])
		} else if strings.HasPrefix(upper, "ADJ2=") {
			item.Adj2, _ = strconv.Atoi(f[5:])
		} else if strings.HasPrefix(upper, "ADJ3=") {
			item.Adj3, _ = strconv.Atoi(f[5:])
		} else if strings.HasPrefix(upper, "VAL1=") {
			item.Val1, _ = strconv.Atoi(f[5:])
		} else if strings.HasPrefix(upper, "VAL2=") {
			item.Val2, _ = strconv.Atoi(f[5:])
		} else if strings.HasPrefix(upper, "VAL3=") {
			item.Val3, _ = strconv.Atoi(f[5:])
		} else if strings.HasPrefix(upper, "VAL4=") {
			item.Val4, _ = strconv.Atoi(f[5:])
		} else if strings.HasPrefix(upper, "VAL5=") {
			item.Val5, _ = strconv.Atoi(f[5:])
		} else {
			switch upper {
			case "OPEN", "CLOSED", "LOCKED", "UNLOCKED", "LIT", "UNLIT",
				"LATCHED", "UNLATCHED", "FLIPPED", "UNFLIPPED", "WORN":
				item.State = upper
			}
		}
	}
	return item
}

func (p *fileParser) parseScriptBlock(fields []string) gameworld.ScriptBlock {
	block := gameworld.ScriptBlock{
		Type: strings.ToUpper(fields[0]),
		Args: fields[1:],
	}
	p.pos++

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])
		if line == "" || strings.HasPrefix(line, ";") {
			p.pos++
			continue
		}
		fields := strings.Fields(line)
		cmd := strings.ToUpper(fields[0])

		if cmd == "ENDIF" {
			p.pos++
			return block
		}

		// Stop at new definition boundaries (handles missing ENDIF in scripts)
		if cmd == "NUMBER" || cmd == "INUMBER" || cmd == "MNUMBER" {
			return block
		}

		if cmd == "ELSE" {
			p.pos++
			// Parse remaining actions/children into the ELSE branch
			for p.pos < len(p.lines) {
				eline := strings.TrimSpace(p.lines[p.pos])
				if eline == "" || strings.HasPrefix(eline, ";") {
					p.pos++
					continue
				}
				efields := strings.Fields(eline)
				ecmd := strings.ToUpper(efields[0])
				if ecmd == "ENDIF" || ecmd == "NUMBER" || ecmd == "INUMBER" || ecmd == "MNUMBER" {
					break
				}
				if ecmd == "ELSE" {
					p.pos++
					break // nested ELSE not supported, just stop
				}
				if strings.HasPrefix(ecmd, "IF") {
					child := p.parseScriptBlock(efields)
					block.ElseChildren = append(block.ElseChildren, child)
					continue
				}
				block.ElseActions = append(block.ElseActions, gameworld.ScriptAction{
					Command: ecmd,
					Args:    efields[1:],
				})
				p.pos++
			}
			continue
		}

		// Nested conditional
		if strings.HasPrefix(cmd, "IF") {
			// If we're in a verb/preverb block and we see another verb/preverb block,
			// it's a sibling, not a child — implicitly close this block.
			// The original engine treated encountering a new IFPREVERB/IFVERB/IFSAY/IFENTRY
			// as an implicit ENDIF for the current verb block.
			isVerbBlock := cmd == "IFVERB" || cmd == "IFVERB2" || cmd == "IFPREVERB" || cmd == "IFPREVERB2" ||
				cmd == "IFSAY" || cmd == "IFENTRY" || cmd == "IFTOUCH" || cmd == "IFLOGIN"
			parentIsVerbBlock := block.Type == "IFVERB" || block.Type == "IFVERB2" || block.Type == "IFPREVERB" || block.Type == "IFPREVERB2" ||
				block.Type == "IFSAY" || block.Type == "IFENTRY" || block.Type == "IFTOUCH" || block.Type == "IFLOGIN"
			if isVerbBlock && parentIsVerbBlock {
				// Implicit ENDIF — return current block, let parent re-parse this line
				return block
			}
			child := p.parseScriptBlock(fields)
			block.Children = append(block.Children, child)
			continue
		}

		// It's an action
		block.Actions = append(block.Actions, gameworld.ScriptAction{
			Command: cmd,
			Args:    fields[1:],
		})
		p.pos++
	}

	return block
}

func (p *fileParser) parseRegion(fields []string) {
	if len(fields) < 3 {
		return
	}
	id, _ := strconv.Atoi(fields[1])
	// Find or create region
	var region *gameworld.Region
	for i := range p.result.Regions {
		if p.result.Regions[i].ID == id {
			region = &p.result.Regions[i]
			break
		}
	}
	if region == nil {
		p.result.Regions = append(p.result.Regions, gameworld.Region{
			ID:         id,
			Properties: make(map[string]string),
		})
		region = &p.result.Regions[len(p.result.Regions)-1]
	}
	key := strings.ToUpper(fields[2])
	val := ""
	if len(fields) >= 4 {
		val = strings.Join(fields[3:], " ")
	}
	region.Properties[key] = val

	// Set typed fields from known properties
	switch key {
	case "DEPART":
		region.DepartRoom, _ = strconv.Atoi(val)
	case "WEATHER":
		region.HasWeather = true
	case "TREASURE":
		region.Treasure, _ = strconv.Atoi(val)
	case "TELEPORT":
		region.TeleportAllowed = (strings.ToUpper(val) == "ALLOWED")
	case "SUMMONING":
		region.SummoningAllowed = (strings.ToUpper(val) == "ALLOWED")
	case "FIRE":
		region.FireMod, _ = strconv.Atoi(val)
	case "COLD":
		region.ColdMod, _ = strconv.Atoi(val)
	case "ELECTRIC":
		region.ElectricMod, _ = strconv.Atoi(val)
	case "MINE_ADJ":
		region.MineAdj, _ = strconv.Atoi(val)
	}
}

