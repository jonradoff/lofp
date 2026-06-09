package engine

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// doLookFull always shows the full room description (explicit LOOK command).
func (e *GameEngine) doLookFull(player *Player) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You are in a void."}}
	}
	result := e.doLook(player)
	// Always include the full description regardless of BriefMode
	if !player.Dead && result.RoomDesc == "" {
		result.RoomDesc = room.Description
	}
	return result
}

func (e *GameEngine) doLook(player *Player) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You are in a void."}}
	}

	result := &CommandResult{
		RoomName: fmt.Sprintf("[%s]", room.Name),
	}

	if player.Dead {
		result.Messages = []string{
			result.RoomName,
			"You are dead and can't do much of anything beside wait for someone to attempt to raise you or for Eternity, Inc. to retrieve you. Hope you paid your premium! [You may type DEPART at any time to allow Eternity, Inc. to retrieve you.]",
		}
		return result
	}

	if !player.BriefMode {
		result.RoomDesc = room.Description
	}

	// List visible items
	for _, ri := range room.Items {
		// Coin piles
		if ri.State == "MONEY" {
			result.Items = append(result.Items, "some coins")
			continue
		}
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		if containsFlag(itemDef.Flags, "HIDDEN") {
			continue
		}
		// Skip placeholder items (ANTI.SCR stubs and invisible items)
		nounName := e.getItemNounName(itemDef)
		if nounName == "anti-item" || nounName == "ucantsee" {
			continue
		}
		name := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
		if ri.Extend != "" {
			name += " " + ri.Extend
		}
		result.Items = append(result.Items, name)
	}

	// Collect other players in the room
	var playersHere []string
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.RoomNumber != player.RoomNumber {
				continue
			}
			if p.FirstName == player.FirstName && p.LastName == player.LastName {
				continue
			}
			if p.Hidden || p.Invisible || p.GMInvis {
				continue
			}
			posDesc := ""
			switch p.Position {
			case 1:
				posDesc = " (sitting)"
			case 2:
				posDesc = " (lying down)"
			case 3:
				posDesc = " (kneeling)"
			case 4:
				posDesc = " (flying)"
			}
			if p.WolfForm {
				playersHere = append(playersHere, "a wolf"+posDesc)
			} else {
				playersHere = append(playersHere, fmt.Sprintf("%s the %s%s", p.FullName(), p.RaceName(), posDesc))
			}
		}
	}

	// List exits
	dirNames := map[string]string{
		"N": "north", "S": "south", "E": "east", "W": "west",
		"NE": "northeast", "NW": "northwest", "SE": "southeast", "SW": "southwest",
		"U": "up", "D": "down", "O": "out", "ABOVE": "up", "BELOW": "down",
	}
	var exits []string
	for dir := range room.Exits {
		if name, ok := dirNames[dir]; ok {
			exits = append(exits, name)
		} else {
			exits = append(exits, strings.ToLower(dir))
		}
	}
	result.Exits = exits

	// Populate GMCP room data
	result.RoomExits = make(map[string]int)
	for dir, roomNum := range room.Exits {
		dirLower := strings.ToLower(dir)
		if name, ok := dirNames[dir]; ok {
			dirLower = name
		}
		result.RoomExits[dirLower] = roomNum
	}
	result.RoomTerrain = room.Terrain
	result.RoomRegion = room.Region

	// Build messages
	var msgs []string
	msgs = append(msgs, result.RoomName)
	if result.RoomDesc != "" {
		msgs = append(msgs, descriptionToMessages(result.RoomDesc)...)
	}
	if len(result.Items) > 0 {
		msgs = append(msgs, "You see "+joinList(result.Items)+".")
	}
	if len(playersHere) > 0 {
		// Format like original: "You see Player1 and Player2." or "You see Player1, Player2 and Player3."
		var pList string
		if len(playersHere) == 1 {
			pList = playersHere[0]
		} else {
			pList = strings.Join(playersHere[:len(playersHere)-1], ", ") + " and " + playersHere[len(playersHere)-1]
		}
		msgs = append(msgs, "You see "+pList+".")
	}
	// Show monsters in the room
	monsterLines := e.MonsterLookLines(player.RoomNumber)
	msgs = append(msgs, monsterLines...)
	// Show weather for outdoor rooms
	if weatherLine := e.GetRoomWeather(player.RoomNumber); weatherLine != "" {
		msgs = append(msgs, weatherLine)
	}
	if len(exits) > 0 {
		msgs = append(msgs, "Obvious exits: "+strings.Join(exits, ", ")+".")
	} else {
		msgs = append(msgs, "There are no obvious exits.")
	}
	result.Messages = msgs
	return result
}

// lookDirMap maps direction words/abbreviations to exit keys.
var lookDirMap = map[string]string{
	"n": "N", "north": "N", "s": "S", "south": "S",
	"e": "E", "east": "E", "w": "W", "west": "W",
	"ne": "NE", "northeast": "NE", "nw": "NW", "northwest": "NW",
	"se": "SE", "southeast": "SE", "sw": "SW", "southwest": "SW",
	"u": "U", "up": "U", "d": "D", "down": "D",
	"o": "O", "out": "O",
}

func (e *GameEngine) doLookAt(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return e.doLook(player)
	}

	target := strings.ToLower(strings.Join(args, " "))

	// Check for directional look (LOOK N, LOOK NORTH, etc.)
	if dir, ok := lookDirMap[target]; ok {
		room := e.rooms[player.RoomNumber]
		if room != nil {
			destNum, hasExit := room.Exits[dir]
			if !hasExit && dir == "U" {
				destNum, hasExit = room.Exits["ABOVE"]
			}
			if !hasExit && dir == "D" {
				destNum, hasExit = room.Exits["BELOW"]
			}
			if hasExit {
				if dest := e.rooms[destNum]; dest != nil {
					msgs := []string{fmt.Sprintf("[%s]", dest.Name)}
					if dest.Description != "" {
						msgs = append(msgs, descriptionToMessages(dest.Description)...)
					}
					// Show players in that room
					if e.sessions != nil {
						var playersHere []string
						for _, p := range e.sessions.OnlinePlayers() {
							if p.RoomNumber == destNum && !p.Hidden && !p.Invisible && !p.GMInvis {
								playersHere = append(playersHere, p.FirstName)
							}
						}
						if len(playersHere) > 0 {
							msgs = append(msgs, fmt.Sprintf("You see %s.", strings.Join(playersHere, ", ")))
						}
					}
					// Show room items
					for _, ri := range dest.Items {
						itemDef := e.items[ri.Archetype]
						if itemDef == nil {
							continue
						}
						itemName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
						msgs = append(msgs, fmt.Sprintf("You see %s.", itemName))
					}
					// Show monsters
					monLines := e.MonsterLookLines(destNum)
					msgs = append(msgs, monLines...)
					return &CommandResult{Messages: msgs}
				}
			}
			return &CommandResult{Messages: []string{"You see nothing of interest in that direction."}}
		}
	}

	// "look at me/myself" → examine self
	if target == "me" || target == "myself" || target == "self" {
		return e.examinePlayer(player, player)
	}

	// Check if target is a player (online, in same room)
	if found := e.findPlayerInRoom(player, target); found != nil {
		return e.examinePlayer(player, found)
	}

	// Check if target is a monster in the room
	if _, monDef := e.findMonsterInRoom(player, target); monDef != nil {
		return e.examineMonster(monDef)
	}

	// Check IN/ON/UNDER prefixes
	prefix := ""
	remaining := target
	for _, p := range []string{"in ", "on ", "under ", "behind "} {
		if strings.HasPrefix(target, p) {
			prefix = strings.ToUpper(strings.TrimSpace(p))
			remaining = strings.TrimPrefix(target, p)
			break
		}
	}
	remaining, ordSkip := parseOrdinal(remaining)
	skip := ordSkip

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You see nothing."}}
	}

	isContainer := func(def *gameworld.ItemDef) bool {
		return def.Type == "CONTAINER" || containsFlag(def.Flags, "CONTAINER") ||
			def.Container == "IN" || def.Container == "ON"
	}

	// Search room items
	for _, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, remaining, e.getAdjName(ri.Adj1)) {
			if skip > 0 { skip--; continue }
			if prefix == "IN" && isContainer(itemDef) {
				return e.lookInRoomContainer(player, itemDef, &ri)
			}
			if prefix != "" {
				return e.lookPrefixRoomItem(room, itemDef, &ri, prefix)
			}
			return e.examineRoomItem(player, room, itemDef, &ri)
		}
	}

	// Search all player items (inventory + worn + wielded)
	allItems := make([]InventoryItem, 0, len(player.Inventory)+len(player.Worn)+1)
	allItems = append(allItems, player.Inventory...)
	allItems = append(allItems, player.Worn...)
	if player.Wielded != nil { allItems = append(allItems, *player.Wielded) }
	for _, ii := range allItems {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, remaining, e.getAdjName(ii.Adj1)) || matchesTarget(name, remaining, e.getAdjName(ii.Adj3)) {
			if skip > 0 { skip--; continue }
			if prefix == "IN" && isContainer(itemDef) {
				return e.lookInContainer(player, itemDef, &ii)
			}
			if prefix != "" {
				displayName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
				return &CommandResult{Messages: []string{fmt.Sprintf("You see nothing noteworthy %s %s.", strings.ToLower(prefix), displayName)}}
			}
			msgs := []string{fmt.Sprintf("You look at your %s.", name)}
			if sm := e.scrollLookMsg(ii.Archetype, ii.Val3); sm != "" {
				msgs = append(msgs, sm)
			}
			return &CommandResult{Messages: msgs}
		}
	}

	return &CommandResult{Messages: []string{"You don't see that here."}}
}

// scrollLookMsg returns a description line if the item is a scroll, empty string otherwise.
func (e *GameEngine) scrollLookMsg(archetype int, val3 int) string {
	if archetype != 168 {
		return ""
	}
	spell := FindSpellByID(val3)
	if spell != nil {
		return fmt.Sprintf("The scroll contains the spell '%s' (spell #%d).", spell.Name, val3)
	}
	return fmt.Sprintf("The scroll contains spell #%d.", val3)
}

// findPlayerInRoom finds an online player in the same room by name (first name match).
func (e *GameEngine) findPlayerInRoom(self *Player, target string) *Player {
	if e.sessions == nil {
		return nil
	}
	for _, p := range e.sessions.OnlinePlayers() {
		if p.RoomNumber != self.RoomNumber {
			continue
		}
		if p.FirstName == self.FirstName && p.LastName == self.LastName {
			continue // skip self, handled separately
		}
		if p.Hidden || p.Invisible || p.GMInvis {
			continue
		}
		if strings.HasPrefix(strings.ToLower(p.FirstName), target) {
			return p
		}
		fullName := strings.ToLower(p.FirstName + " " + p.LastName)
		if strings.HasPrefix(fullName, target) {
			return p
		}
	}
	return nil
}

// findMonsterInRoom finds a monster in the player's room by name prefix.
// Returns the MonsterInstance and its definition, or nil if not found.
func (e *GameEngine) findMonsterInRoom(player *Player, target string) (*MonsterInstance, *gameworld.MonsterDef) {
	return e.findMonsterInRoomEx(player, target, false)
}

func (e *GameEngine) findMonsterInRoomIncludeDead(player *Player, target string) (*MonsterInstance, *gameworld.MonsterDef) {
	return e.findMonsterInRoomEx(player, target, true)
}

func (e *GameEngine) findMonsterInRoomEx(player *Player, target string, includeDead bool) (*MonsterInstance, *gameworld.MonsterDef) {
	if e.monsterMgr == nil {
		return nil, nil
	}
	var monsters []MonsterInstance
	if includeDead {
		monsters = e.monsterMgr.AllMonstersInRoom(player.RoomNumber)
	} else {
		monsters = e.monsterMgr.MonstersInRoom(player.RoomNumber)
	}
	target = strings.ToLower(strings.TrimSpace(target))
	// Strip leading articles so "a skeleton" matches "skeleton"
	for _, article := range []string{"a ", "an ", "the ", "some "} {
		if strings.HasPrefix(target, article) {
			target = strings.TrimPrefix(target, article)
			break
		}
	}
	for i := range monsters {
		def := e.monsters[monsters[i].DefNumber]
		if def == nil {
			continue
		}
		name := strings.ToLower(FormatMonsterName(def, e.monAdjs))
		noun := strings.ToLower(def.Name)
		if strings.HasPrefix(name, target) || strings.HasPrefix(noun, target) {
			return &monsters[i], def
		}
	}
	return nil, nil
}

// examineMonster returns a description of a monster.
func (e *GameEngine) examineMonster(def *gameworld.MonsterDef) *CommandResult {
	name := FormatMonsterName(def, e.monAdjs)
	var msgs []string
	if def.Description != "" {
		msgs = append(msgs, def.Description)
	} else {
		msgs = append(msgs, fmt.Sprintf("You see a %s.", name))
	}
	return &CommandResult{Messages: msgs}
}

// examinePlayer returns a description of a player as seen by the observer.
func (e *GameEngine) examinePlayer(observer *Player, target *Player) *CommandResult {
	isSelf := observer.FirstName == target.FirstName && observer.LastName == target.LastName

	var pronoun string
	if isSelf {
		pronoun = "You are"
	} else if target.Gender == 0 {
		pronoun = "He is"
	} else {
		pronoun = "She is"
	}

	msgs := []string{}
	if isSelf {
		msgs = append(msgs, "You examine yourself.")
	} else if target.Title != "" {
		msgs = append(msgs, fmt.Sprintf("Before you is %s %s.", target.Title, target.FullName()))
	} else {
		msgs = append(msgs, fmt.Sprintf("You look at %s.", target.FullName()))
	}

	// Custom @line descriptions override the auto-generated race/gender line
	if target.DescLine1 != "" || target.DescLine2 != "" || target.DescLine3 != "" {
		if target.DescLine1 != "" {
			msgs = append(msgs, target.DescLine1)
		}
		if target.DescLine2 != "" {
			msgs = append(msgs, target.DescLine2)
		}
		if target.DescLine3 != "" {
			msgs = append(msgs, target.DescLine3)
		}
	} else {
		msgs = append(msgs, fmt.Sprintf("%s a %s %s.", pronoun, target.RaceName(), genderName(target.Gender)))
	}

	heOrShe := "He"
	heOrSheLC := "him"
	if target.Gender == 1 {
		heOrShe = "She"
		heOrSheLC = "her"
	}
	if isSelf {
		heOrSheLC = "you"
	}

	// Health description
	healthPct := float64(100)
	if target.MaxBodyPoints > 0 {
		healthPct = float64(target.BodyPoints) / float64(target.MaxBodyPoints) * 100
	}
	if isSelf {
		switch {
		case healthPct >= 100:
			msgs = append(msgs, "You are in perfect health.")
		case healthPct >= 75:
			msgs = append(msgs, "You have minor injuries.")
		case healthPct >= 50:
			msgs = append(msgs, "You are moderately wounded.")
		case healthPct >= 25:
			msgs = append(msgs, "You are seriously wounded.")
		case healthPct > 0:
			msgs = append(msgs, "You are critically wounded!")
		default:
			msgs = append(msgs, "You are dead.")
		}
	} else {
		switch {
		case healthPct >= 100:
			msgs = append(msgs, fmt.Sprintf("%s appears to be in perfect health.", heOrShe))
		case healthPct >= 75:
			msgs = append(msgs, fmt.Sprintf("%s has minor injuries.", heOrShe))
		case healthPct >= 50:
			msgs = append(msgs, fmt.Sprintf("%s is moderately wounded.", heOrShe))
		case healthPct >= 25:
			msgs = append(msgs, fmt.Sprintf("%s is seriously wounded.", heOrShe))
		case healthPct > 0:
			msgs = append(msgs, fmt.Sprintf("%s is critically wounded!", heOrShe))
		default:
			msgs = append(msgs, fmt.Sprintf("%s is dead.", heOrShe))
		}
	}

	// Position
	switch target.Position {
	case 1:
		msgs = append(msgs, fmt.Sprintf("%s sitting.", pronoun))
	case 2:
		msgs = append(msgs, fmt.Sprintf("%s lying down.", pronoun))
	case 3:
		msgs = append(msgs, fmt.Sprintf("%s kneeling.", pronoun))
	}

	// Visible conditions and effects
	if target.Bleeding {
		msgs = append(msgs, fmt.Sprintf("%s bleeding.", pronoun))
	}
	if target.Stunned {
		msgs = append(msgs, fmt.Sprintf("%s stunned.", pronoun))
	}
	if target.Poisoned {
		if isSelf {
			msgs = append(msgs, "You look poisoned.")
		} else {
			msgs = append(msgs, fmt.Sprintf("%s looks poisoned.", heOrShe))
		}
	}
	if target.Diseased {
		if isSelf {
			msgs = append(msgs, "You look sickly.")
		} else {
			msgs = append(msgs, fmt.Sprintf("%s looks sickly.", heOrShe))
		}
	}
	if target.Immobilized {
		msgs = append(msgs, fmt.Sprintf("%s rooted to the spot.", pronoun))
	}

	// Guard status — check if someone is guarding this target
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.GuardTarget == target.FirstName && p.RoomNumber == target.RoomNumber {
				if isSelf {
					msgs = append(msgs, fmt.Sprintf("You are being guarded by %s.", p.FirstName))
				} else {
					msgs = append(msgs, fmt.Sprintf("%s is being guarded by %s.", target.PronounCap(), p.FirstName))
				}
				break
			}
		}
	}

	// Active spell/psi effects
	if target.DefenseBonus > 0 {
		msgs = append(msgs, fmt.Sprintf("A shimmering magical aura surrounds %s.", isSelfOr(isSelf, "you", heOrSheLC)))
	}
	if target.CanFly && target.Race != RaceDrakin {
		msgs = append(msgs, fmt.Sprintf("%s hovering in the air.", pronoun))
	}
	if target.Invisible {
		// Only visible to self or GMs
		if isSelf {
			msgs = append(msgs, "You are invisible.")
		}
	}

	// Equipment
	if target.Wielded != nil {
		wDef := e.items[target.Wielded.Archetype]
		if wDef != nil {
			name := e.formatItemName(wDef, target.Wielded.Adj1, target.Wielded.Adj2, target.Wielded.Adj3)
			if isSelf {
				msgs = append(msgs, fmt.Sprintf("You are wielding %s.", name))
			} else {
				msgs = append(msgs, fmt.Sprintf("%s wielding %s.", pronoun, name))
			}
		}
	}
	var wornNames []string
	for _, worn := range target.Worn {
		wDef := e.items[worn.Archetype]
		if wDef != nil {
			wornNames = append(wornNames, e.formatItemName(wDef, worn.Adj1, worn.Adj2, worn.Adj3))
		}
	}
	if len(wornNames) > 0 {
		if isSelf {
			msgs = append(msgs, fmt.Sprintf("You are wearing %s.", joinList(wornNames)))
		} else {
			msgs = append(msgs, fmt.Sprintf("%s wearing %s.", pronoun, joinList(wornNames)))
		}
	}

	return &CommandResult{Messages: msgs}
}

// formatItemNameNoArticle returns item name with adjectives but no article prefix.
func (e *GameEngine) formatItemNameNoArticle(def *gameworld.ItemDef, adj1, adj2, adj3 int) string {
	var parts []string
	if adj1 > 0 {
		if name, ok := e.adjectives[adj1]; ok { parts = append(parts, name) }
	}
	if adj2 > 0 {
		if name, ok := e.adjectives[adj2]; ok { parts = append(parts, name) }
	}
	if adj3 > 0 {
		if name, ok := e.adjectives[adj3]; ok { parts = append(parts, name) }
	}
	parts = append(parts, e.getItemNounName(def))
	return strings.Join(parts, " ")
}

func (e *GameEngine) formatItemName(def *gameworld.ItemDef, adj1, adj2, adj3 int) string {
	var parts []string
	if adj1 > 0 {
		if name, ok := e.adjectives[adj1]; ok {
			parts = append(parts, name)
		}
	}
	if adj2 > 0 {
		if name, ok := e.adjectives[adj2]; ok {
			parts = append(parts, name)
		}
	}
	if adj3 > 0 {
		if name, ok := e.adjectives[adj3]; ok {
			parts = append(parts, name)
		}
	}
	nounName := e.getItemNounName(def)
	parts = append(parts, nounName)

	name := strings.Join(parts, " ")
	article := strings.ToUpper(def.Article)
	if article == "" || article == "A" {
		// Auto-detect "an" for words starting with a vowel sound
		first := strings.ToLower(name[:1])
		if first == "a" || first == "e" || first == "i" || first == "o" || first == "u" {
			return "an " + name
		}
		return "a " + name
	}
	if article == "AN" {
		return "an " + name
	}
	if article == "THE" {
		return "the " + name
	}
	if article == "SOME" {
		return "some " + name
	}
	return strings.ToLower(article) + " " + name
}

func (e *GameEngine) getItemNounName(def *gameworld.ItemDef) string {
	if name, ok := e.nouns[def.NameID]; ok {
		return name
	}
	return fmt.Sprintf("item#%d", def.Number)
}

// adjByName returns the adjective ID for a given name (case-insensitive), or 0 if not found.
func (e *GameEngine) adjByName(name string) int {
	target := strings.ToLower(name)
	for id, adj := range e.adjectives {
		if strings.ToLower(adj) == target {
			return id
		}
	}
	return 0
}

func (e *GameEngine) getAdjName(adjID int) string {
	if adjID > 0 {
		if name, ok := e.adjectives[adjID]; ok {
			return name
		}
	}
	return ""
}

func (e *GameEngine) examineRoomItem(player *Player, room *gameworld.Room, def *gameworld.ItemDef, ri *gameworld.RoomItem) *CommandResult {
	result := &CommandResult{}

	// Room-scoped EXAMINE description
	refStr := fmt.Sprintf("%d", ri.Ref)
	if desc, ok := room.ItemDescriptions["EXAMINE:"+refStr]; ok {
		result.Messages = append(result.Messages, descriptionToMessages(desc)...)
	} else if isPortal(def.Type) {
		result.Messages = append(result.Messages, "You can't clearly see where it leads.")
	} else {
		result.Messages = append(result.Messages, "It is nondescript.")
	}

	// Run IFVERB LOOK scripts on the item (can add SHOWROOM output, etc.)
	sc := e.RunVerbScripts(player, room, "LOOK", ri, def)
	if len(sc.Messages) > 0 {
		result.Messages = append(result.Messages, sc.Messages...)
	}
	if len(sc.RoomMsgs) > 0 {
		result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
	}

	return result
}

func (e *GameEngine) readRoomItem(room *gameworld.Room, def *gameworld.ItemDef, ri *gameworld.RoomItem) *CommandResult {
	refStr := fmt.Sprintf("%d", ri.Ref)
	if desc, ok := room.ItemDescriptions["READ:"+refStr]; ok {
		return &CommandResult{Messages: descriptionToMessages(desc)}
	}
	return &CommandResult{Messages: []string{"There is nothing written on it."}}
}

func (e *GameEngine) lookPrefixRoomItem(room *gameworld.Room, def *gameworld.ItemDef, ri *gameworld.RoomItem, prefix string) *CommandResult {
	refStr := fmt.Sprintf("%d", ri.Ref)
	if desc, ok := room.ItemDescriptions[prefix+":"+refStr]; ok {
		return &CommandResult{Messages: descriptionToMessages(desc)}
	}
	name := e.getItemNounName(def)
	return &CommandResult{Messages: []string{fmt.Sprintf("You see nothing noteworthy %s the %s.", strings.ToLower(prefix), name)}}
}

// joinList formats a list as "a, b and c" or "a and b" or "a".
func joinList(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
	}
}

// descriptionToMessages splits a description into message lines.
func descriptionToMessages(desc string) []string {
	if strings.Contains(desc, "\n") {
		return strings.Split(desc, "\n")
	}
	return []string{desc}
}

func (e *GameEngine) formatItemLook(def *gameworld.ItemDef, ri *gameworld.RoomItem) []string {
	name := e.formatItemName(def, ri.Adj1, ri.Adj2, ri.Adj3)
	if ri.Extend != "" {
		name += " " + ri.Extend
	}
	msgs := []string{fmt.Sprintf("You examine %s.", name)}
	if isPortal(def.Type) && ri.Val2 > 0 {
		msgs = append(msgs, "It appears to lead somewhere.")
	}
	return msgs
}

// parseOrdinal extracts an ordinal prefix from a target string.
// Returns (cleanTarget, ordinalNumber). ordinal 0 means "first/default".
// "2 gate" → ("gate", 1), "other gate" → ("gate", 1), "second gate" → ("gate", 1),
// "first gate" → ("gate", 0), "3 gate" → ("gate", 2), "gate" → ("gate", 0)
func parseOrdinal(target string) (string, int) {
	ordinalWords := map[string]int{
		"first": 0, "second": 1, "third": 2, "fourth": 3, "fifth": 4,
		"sixth": 5, "seventh": 6, "eighth": 7, "ninth": 8, "tenth": 9,
		"other": 1,
	}
	parts := strings.SplitN(target, " ", 2)
	if len(parts) < 2 {
		return target, 0
	}
	first := strings.ToLower(parts[0])
	// Check word ordinals
	if ord, ok := ordinalWords[first]; ok {
		return parts[1], ord
	}
	// Check numeric: "2 gate" means 2nd (index 1)
	if num, err := strconv.Atoi(first); err == nil && num >= 1 {
		return parts[1], num - 1
	}
	// Check trailing number: "counter 2" means 2nd counter
	lastSpace := strings.LastIndex(target, " ")
	if lastSpace > 0 {
		last := target[lastSpace+1:]
		if num, err := strconv.Atoi(last); err == nil && num >= 1 {
			return target[:lastSpace], num - 1
		}
	}
	return target, 0
}

// matchesTargetOrdinal checks if a target matches, accounting for ordinal prefixes.
func matchesTargetOrdinal(nounName, cleanTarget, adjName string, skip *int) bool {
	if !matchesTarget(nounName, cleanTarget, adjName) {
		return false
	}
	if *skip > 0 {
		*skip--
		return false
	}
	return true
}

func matchesTarget(nounName, target, adjName string) bool {
	t := strings.ToLower(target)
	n := strings.ToLower(nounName)
	a := strings.ToLower(adjName)

	if t == n {
		return true
	}
	if a != "" && t == a+" "+n {
		return true
	}
	// Partial match (prefix)
	if strings.HasPrefix(n, t) {
		return true
	}
	// Match last word of noun (e.g., "tooth" matches "rat tooth")
	if idx := strings.LastIndex(n, " "); idx >= 0 {
		if strings.HasPrefix(n[idx+1:], t) {
			return true
		}
	}
	return false
}

func containsFlag(flags []string, flag string) bool {
	for _, f := range flags {
		if strings.EqualFold(f, flag) {
			return true
		}
	}
	return false
}

func isPortal(itemType string) bool {
	switch itemType {
	case "PORTAL", "PORTAL_THROUGH", "PORTAL_CLIMB", "PORTAL_UP", "PORTAL_DOWN",
		"PORTAL_OVER", "PORTAL_CLIMBUP", "PORTAL_CLIMBDOWN":
		return true
	}
	return false
}

func isWeapon(itemType string) bool {
	switch itemType {
	case "SLASH_WEAPON", "CRUSH_WEAPON", "PUNCTURE_WEAPON", "POLE_WEAPON",
		"TWOHAND_WEAPON", "BOW_WEAPON", "THROWN_WEAPON", "STABTHROWN",
		"POLETHROWN", "HANDGUN", "RIFLE", "CLAW_WEAPON", "BITE_WEAPON",
		"DRAKIN_CRUSH", "DRAKIN_POLE", "DRAKIN_SLASH", "DRAKIN_THROWN":
		return true
	}
	return false
}
