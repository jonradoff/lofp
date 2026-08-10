package engine

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// undergarmentHiddenBy maps inner (undergarment) worn slots to the outer slots
// that layer over them and hide them from a LOOK description.
var undergarmentHiddenBy = map[string][]string{
	"WORN_TORSO1": {"WORN_TORSO2", "WORN_TORSO3", "WORN_ARMOR", "WORN_BODY"},
	"WORN_TRUNK1": {"WORN_TRUNK2"},
	"WORN_FEET1":  {"WORN_FEET2"},
}

// revealedLayerName returns the display name of the inner-layer item that becomes
// visible after removedSlot is vacated (i.e. removedSlot was the last outer layer
// covering it), or "" if nothing is revealed.
func (e *GameEngine) revealedLayerName(player *Player, removedSlot string) string {
	wornSlots := map[string]bool{}
	for _, w := range player.Worn {
		wornSlots[w.WornSlot] = true
	}
	for innerSlot, outerSlots := range undergarmentHiddenBy {
		wasCoveredByRemoved := false
		stillCovered := false
		for _, outer := range outerSlots {
			if outer == removedSlot {
				wasCoveredByRemoved = true
			} else if wornSlots[outer] {
				stillCovered = true
			}
		}
		if !wasCoveredByRemoved || stillCovered {
			continue
		}
		for _, w := range player.Worn {
			if w.WornSlot == innerSlot {
				if def := e.items[w.Archetype]; def != nil {
					return e.formatItemName(def, w.Adj1, w.Adj2, w.Adj3, w.Tail)
				}
			}
		}
	}
	return ""
}

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
		// Items PUT/NEWPUT inside a container aren't loose on the floor — they're
		// only visible via LOOK IN/EXAMINE on the container that holds them.
		if ri.IsPut {
			continue
		}
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
		name := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3, ri.Extend)
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
			if p.Hidden || p.Invisible || p.PhantomForm || p.GMInvis {
				continue
			}
			posDesc := ""
			switch {
			case p.Dead:
				posDesc = " (dead)"
			case p.Unconscious:
				posDesc = " (unconscious)"
			default:
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
			}
			if p.CarriedBy != "" {
				posDesc += fmt.Sprintf(" (carried by %s)", p.CarriedBy)
			}
			playersHere = append(playersHere, p.DisplayName()+posDesc)
		}
	}

	// List exits. ABOVE/BELOW are fly-only exits accessed via ASCEND/DESCEND — not shown here.
	dirNames := map[string]string{
		"N": "north", "S": "south", "E": "east", "W": "west",
		"NE": "northeast", "NW": "northwest", "SE": "southeast", "SW": "southwest",
		"U": "up", "D": "down", "O": "out",
	}
	var exits []string
	for dir := range room.Exits {
		if dir == "ABOVE" || dir == "BELOW" {
			continue // fly-only; not visible in Obvious exits
		}
		if name, ok := dirNames[dir]; ok {
			exits = append(exits, name)
		} else {
			exits = append(exits, strings.ToLower(dir))
		}
	}
	result.Exits = exits

	// Populate GMCP room data (also excludes ABOVE/BELOW)
	result.RoomExits = make(map[string]int)
	for dir, roomNum := range room.Exits {
		if dir == "ABOVE" || dir == "BELOW" {
			continue
		}
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
	// Players and monsters/NPCs are shown together in one sentence, matching the
	// original format (see original/legends/shirla.cap, e.g. "You see Wolfbane and
	// Delight."), followed by a separate items sentence.
	peopleHere := append(playersHere, e.monsterNamesInRoom(player.RoomNumber)...)
	if len(peopleHere) > 0 {
		msgs = append(msgs, "You see "+joinList(peopleHere)+".")
	}
	if len(result.Items) > 0 {
		verb := " are here."
		if len(result.Items) == 1 {
			verb = " is here."
		}
		msgs = append(msgs, capitalize(joinList(result.Items))+verb)
	}
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

func (e *GameEngine) doLookAt(ctx context.Context, player *Player, args []string) *CommandResult {
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
					// Show players and monsters/NPCs together in one sentence
					var peopleHere []string
					if e.sessions != nil {
						for _, p := range e.sessions.OnlinePlayers() {
							if p.RoomNumber == destNum && !p.Hidden && !p.Invisible && !p.PhantomForm && !p.GMInvis {
								peopleHere = append(peopleHere, p.DisplayName())
							}
						}
					}
					peopleHere = append(peopleHere, e.monsterNamesInRoom(destNum)...)
					if len(peopleHere) > 0 {
						msgs = append(msgs, "You see "+joinList(peopleHere)+".")
					}
					// Show room items
					var itemNames []string
					for _, ri := range dest.Items {
						if ri.IsPut {
							continue
						}
						itemDef := e.items[ri.Archetype]
						if itemDef == nil {
							continue
						}
						itemNames = append(itemNames, e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3, ri.Extend))
					}
					if len(itemNames) > 0 {
						verb := " are here."
						if len(itemNames) == 1 {
							verb = " is here."
						}
						msgs = append(msgs, capitalize(joinList(itemNames))+verb)
					}
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
	if monInst, monDef := e.findMonsterInRoom(player, target); monDef != nil {
		return e.examineMonster(monInst, monDef)
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
	remaining, mine := stripMyPrefix(remaining)
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

	// LOOK IN <eye of scrying>, once translucent — special-cased ahead of the generic
	// item search since revealing a vision and possibly crumbling the eye needs ctx (to
	// persist) and a direct Inventory index, neither of which the generic
	// examineCarriedItem path threads through. An eye that isn't translucent yet falls
	// through to the normal search below, which produces the standard "you see nothing
	// noteworthy in X" for a non-container item.
	if prefix == "IN" {
		for i := range player.Inventory {
			ii := &player.Inventory[i]
			if ii.Archetype != eyeOfScryingArch || ii.Val3 != eyeStateTranslucent {
				continue
			}
			def := e.items[ii.Archetype]
			if def == nil {
				continue
			}
			if !matchesTarget(e.getItemNounName(def), remaining, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
				continue
			}
			if skip > 0 {
				skip--
				continue
			}
			return e.lookInEye(ctx, player, i, def)
		}
	}

	// Search room items — skipped entirely when "my" was used (e.g. "look in my chest").
	if !mine {
		for _, ri := range room.Items {
			itemDef := e.items[ri.Archetype]
			if itemDef == nil {
				continue
			}
			if e.matchesItemOrPotion(itemDef, ri.Adj1, ri.Adj2, ri.Adj3, ri.Val2, ri.Val4, remaining) {
				if skip > 0 {
					skip--
					continue
				}
				// A LIQCONTAINER examined without an IN/ON/UNDER prefix implicitly behaves
				// like LOOK IN, reporting fullness and what liquid (if any) is visible —
				// matching examineCarriedItem's behavior for the same item held in hand.
				if (prefix == "" || prefix == "IN") && itemDef.Type == "LIQCONTAINER" {
					return e.lookInRoomContainer(player, itemDef, &ri)
				}
				if prefix == "IN" && isContainer(itemDef) {
					return e.lookInRoomContainer(player, itemDef, &ri)
				}
				if prefix != "" {
					return e.lookPrefixRoomItem(room, itemDef, &ri, prefix)
				}
				return e.examineRoomItem(player, room, itemDef, &ri)
			}
		}
	}

	// Search all player items (inventory + worn + wielded + off-hand)
	allItems := make([]InventoryItem, 0, len(player.Inventory)+len(player.Worn)+2)
	allItems = append(allItems, player.Inventory...)
	allItems = append(allItems, player.Worn...)
	if player.Wielded != nil {
		allItems = append(allItems, *player.Wielded)
	}
	if player.OffHand != nil {
		allItems = append(allItems, *player.OffHand)
	}
	for _, ii := range allItems {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		if e.matchesItemOrPotion(itemDef, ii.Adj1, ii.Adj2, ii.Adj3, ii.Val2, ii.Val4, remaining) {
			if skip > 0 {
				skip--
				continue
			}
			return e.examineCarriedItem(player, room, ii, itemDef, prefix)
		}
	}

	// Not found directly — check one level into any open carried container
	// (e.g. an alchemist checking a vial of potion sitting inside an open bag).
	for _, container := range e.findOpenContainers(allItems) {
		for _, ci := range container.Contents {
			cdef := e.items[ci.Archetype]
			if cdef == nil {
				continue
			}
			if e.matchesItemOrPotion(cdef, ci.Adj1, ci.Adj2, ci.Adj3, ci.Val2, ci.Val4, remaining) {
				if skip > 0 {
					skip--
					continue
				}
				return e.examineCarriedItem(player, room, ci, cdef, prefix)
			}
		}
	}

	return &CommandResult{Messages: []string{"You don't see that here."}}
}

// lookInEye handles LOOK IN <eye of scrying> once translucent (Val3 ==
// eyeStateTranslucent, set by Scry — see tryCastSpellAtEye in spells.go). Reveals a
// vision of the room where the last player death occurred (e.lastDeathRoom), then
// has a 1-in-10 chance to crumble the eye to dust — matching ITEM1.SCR's
// "IFPREVERB LOOK -1" block on item 520 exactly (RANDOM 1-10, crumble on a roll of 1).
func (e *GameEngine) lookInEye(ctx context.Context, player *Player, idx int, def *gameworld.ItemDef) *CommandResult {
	ii := &player.Inventory[idx]
	name := e.formatItemName(def, ii.Adj1, ii.Adj2, ii.Adj3, ii.Tail)

	msgs := []string{"You gaze deep into the eye..."}
	if e.lastDeathRoom != 0 {
		if dest := e.rooms[e.lastDeathRoom]; dest != nil {
			// Render the death room's look without moving the caster there — same
			// swap-and-restore trick castScrySpell uses for the mark-vision use.
			originalRoom := player.RoomNumber
			player.RoomNumber = dest.Number
			lookResult := e.doLook(player)
			player.RoomNumber = originalRoom
			msgs = append(msgs, lookResult.Messages...)
		}
	}

	var roomMsgs []string
	if !player.IsConcealed() {
		roomMsgs = append(roomMsgs, fmt.Sprintf("%s looks deep into %s.", player.DisplayNameCap(), name))
	}

	if rand.Intn(10) == 0 {
		msgs = append(msgs, "The eye crumbles.")
		roomMsgs = append(roomMsgs, "The eye crumbles.")
		player.Inventory = append(player.Inventory[:idx], player.Inventory[idx+1:]...)
	}

	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: msgs, RoomBroadcast: roomMsgs, PlayerState: player}
}

// examineCarriedItem builds the EXAMINE/LOOK result for one matched item that the
// player is carrying — held directly, worn, wielded, or nested one level inside an
// open container. A LIQCONTAINER examined without an IN/ON/UNDER prefix implicitly
// behaves like LOOK IN, reporting fullness and what liquid (if any) is visible.
func (e *GameEngine) examineCarriedItem(player *Player, room *gameworld.Room, ii InventoryItem, itemDef *gameworld.ItemDef, prefix string) *CommandResult {
	name := e.getItemNounName(itemDef)
	if (prefix == "" || prefix == "IN") && itemDef.Type == "LIQCONTAINER" {
		if !containerIsOpen(itemDef, ii.State) {
			return &CommandResult{Messages: []string{fmt.Sprintf("%s is closed.", capitalize(e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3, ii.Tail)))}}
		}
		return &CommandResult{Messages: e.potionLookInMessages(itemDef, ii.Adj1, ii.Val2, ii.Val4, ii.Tail)}
	}
	if prefix == "IN" && isContainerDef(itemDef) {
		return e.lookInContainer(player, itemDef, &ii)
	}
	if prefix != "" {
		displayName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3, ii.Tail)
		return &CommandResult{Messages: []string{fmt.Sprintf("You see nothing noteworthy %s %s.", strings.ToLower(prefix), displayName)}}
	}
	// A set ExamineDesc replaces the generic "You look at your X." opener
	// entirely, matching the original game's output (see original/legends
	// session captures — EXAMINE with a custom description prints just the
	// description, no opener line).
	var msgs []string
	if sm := e.scrollLookMsg(ii.Archetype, ii.Val3); sm != "" {
		msgs = []string{fmt.Sprintf("You look at your %s.", name), sm}
	} else if itemDef.ExamineDesc != "" {
		msgs = descriptionToMessages(itemDef.ExamineDesc)
	} else {
		msgs = []string{fmt.Sprintf("You look at your %s.", name)}
	}
	// Run IFPREVERB/IFVERB LOOK scripts on the item (Ref=-1 = inventory item, skip room scripts)
	ri := &gameworld.RoomItem{Ref: -1, Archetype: ii.Archetype, Adj1: ii.Adj1, Adj2: ii.Adj2, Adj3: ii.Adj3, Val1: ii.Val1, Val2: ii.Val2, Val3: ii.Val3, Val4: ii.Val4, Val5: ii.Val5, ItemBits: ii.ItemBits}
	// Root-level IFVAR blocks may contain nested IFVERB EXAMINE -1 blocks (e.g. the
	// lens of worlds' SHOWROOM preview); RunPreverbScripts/RunVerbScripts only match
	// top-level IFVERB blocks, so this walk is needed to reach those.
	sc0 := e.RunItemScripts(player, room, "EXAMINE", ri, itemDef)
	if len(sc0.Messages) > 0 {
		msgs = append(msgs, sc0.Messages...)
	}
	sc := e.RunPreverbScripts(player, room, "LOOK", ri, itemDef)
	if len(sc.Messages) > 0 {
		msgs = append(msgs, sc.Messages...)
	}
	// PLREVENT/CONTPLREVENT-deferred actions must be scheduled, or everything after
	// the delay is lost.
	for _, segs := range [][]ScriptSegment{sc0.DeferredSegments, sc.DeferredSegments} {
		if len(segs) > 0 {
			e.scheduleScriptSegments(player, segs)
		}
	}
	roomMsgs := append(append([]string{}, sc0.RoomMsgs...), sc.RoomMsgs...)
	return &CommandResult{Messages: msgs, RoomBroadcast: roomMsgs}
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

// findPlayerInRoom finds an online player in the same room by name (first name
// match). Supports a leading ordinal ("2 wolf", "second wolf") to disambiguate
// when multiple matches share the same displayed name — most commonly several
// wolf-form Wolflings, who all show as "a wolf" — counting matches in the same
// order they're found, same convention as item/monster ordinal targeting.
func (e *GameEngine) findPlayerInRoom(self *Player, target string) *Player {
	if e.sessions == nil {
		return nil
	}
	target, skip := parseOrdinal(target)
	for _, p := range e.sessions.OnlinePlayers() {
		if p.RoomNumber != self.RoomNumber {
			continue
		}
		if p.FirstName == self.FirstName && p.LastName == self.LastName {
			continue // skip self, handled separately
		}
		if p.Hidden || p.Invisible || p.PhantomForm || p.GMInvis {
			continue
		}
		matched := p.NameMatches(target)
		if !matched && !p.WolfForm && !p.MistForm && !p.SlimeForm && !(p.Disguised && p.ActiveDisguise.Name != "") {
			// Real full name only resolves when not presenting as someone/something
			// else — a disguised or wolf-form player can't be found by their true name.
			fullName := strings.ToLower(p.FirstName + " " + p.LastName)
			matched = strings.HasPrefix(fullName, target)
		}
		if !matched {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}
		return p
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
	// Ordinal prefix/suffix so "attack 2 skeleton" / "attack second skeleton" /
	// "attack skeleton 2" target the 2nd matching monster instead of the 1st.
	target, skip := parseOrdinal(target)
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
			if skip > 0 {
				skip--
				continue
			}
			return &monsters[i], def
		}
	}
	return nil, nil
}

// examineMonster returns a description of a monster, including its live
// wound state.
func (e *GameEngine) examineMonster(inst *MonsterInstance, def *gameworld.MonsterDef) *CommandResult {
	name := FormatMonsterName(def, e.monAdjs)
	var msgs []string
	if inst != nil && def.Description == "" && disguisableNPCNames[strings.ToLower(def.Name)] && e.monsterMgr != nil {
		if rolled, ok := e.monsterMgr.GetOrRollAppearance(inst.ID); ok {
			ageWord := ageDescriptor(rolled.AppearanceAge, rolled.AppearanceRace)
			msgs = append(msgs, fmt.Sprintf("You see %s%s, %s%s %s %s.",
				articleFor(name, def.Unique), name, articleFor(ageWord, false), ageWord, strings.ToLower(genderName(rolled.AppearanceGender)), strings.ToLower(RaceNameByID(rolled.AppearanceRace))))
			subj, hasVerb, beVerb := "He", "has", "is"
			if rolled.AppearanceGender == GenderFemale {
				subj = "She"
			}
			msgs = append(msgs, fmt.Sprintf("%s %s %s, %s and %s.",
				subj, beVerb, heightDescriptor(rolled.AppearanceHeight, rolled.AppearanceRace),
				weightDescriptor(rolled.AppearanceWeight, rolled.AppearanceRace), buildDescriptor(rolled.AppearanceStrength, rolled.AppearanceRace)))
			if rolled.AppearanceHairStyle == "bald" {
				msgs = append(msgs, fmt.Sprintf("%s %s %s eyes and %s skin. %s is completely bald.",
					subj, hasVerb, rolled.AppearanceEyeColor, rolled.AppearanceSkinColor, subj))
			} else {
				msgs = append(msgs, fmt.Sprintf("%s %s %s eyes, %s skin and %s %s hair.",
					subj, hasVerb, rolled.AppearanceEyeColor, rolled.AppearanceSkinColor, rolled.AppearanceHairStyle, rolled.AppearanceHairColor))
			}
			if !inst.Alive {
				msgs = append(msgs, "It is dead.")
			}
			return &CommandResult{Messages: msgs}
		}
	}
	if def.Description != "" {
		msgs = append(msgs, def.Description)
	} else {
		msgs = append(msgs, fmt.Sprintf("You see a %s.", name))
	}
	if inst != nil {
		deadSuffix := ""
		if !inst.Alive {
			deadSuffix = "is dead"
		}
		if len(inst.Wounds) > 0 {
			msgs = append(msgs, "It has "+buildWoundSentence(inst.Wounds, deadSuffix)+".")
		} else if !inst.Alive {
			msgs = append(msgs, "It is dead.")
		}
	}
	return &CommandResult{Messages: msgs}
}

// examinePlayer returns a description of a player as seen by the observer.
func (e *GameEngine) examinePlayer(observer *Player, target *Player) *CommandResult {
	isSelf := observer.FirstName == target.FirstName && observer.LastName == target.LastName

	// Effective (apparent) appearance — real values, overridden by the active
	// Disguise persona's set fields. A disguise is reflected back to everyone,
	// including the wearer's own LOOK/EXAMINE, and to GMs playing normally
	// (only dedicated GM admin commands see through it).
	race, gender, age, height, weight, strength, eye, skin, hairColor, hairStyle := target.EffectiveAppearance()
	displayName := target.DisplayFullName()

	var pronoun string
	if isSelf {
		pronoun = "You are"
	} else if gender == GenderMale {
		pronoun = "He is"
	} else {
		pronoun = "She is"
	}

	msgs := []string{}
	if isSelf {
		msgs = append(msgs, "You examine yourself.")
	} else if target.MistForm {
		msgs = append(msgs, "You look at some mist.")
	} else if target.SlimeForm {
		msgs = append(msgs, "You look at a slime.")
	} else if target.WolfForm {
		msgs = append(msgs, "You look at a wolf.")
	} else if target.Disguised {
		// No "You look at X." opener — matches examineMonster's NPC-style output
		// (straight into "You see ..."), so a disguise reads exactly like the
		// genuine NPC it's blending into.
	} else if target.Title != "" {
		msgs = append(msgs, fmt.Sprintf("Before you is %s %s.", target.Title, target.FullName()))
	} else {
		msgs = append(msgs, fmt.Sprintf("You look at %s.", displayName))
	}

	// Wolf form replaces custom @line descriptions and the generated human
	// description alike with a generated wolf description — fur color mirrors
	// HairColor (black if bald), everything human-specific (build, skin, gear)
	// is skipped. Human descriptions (custom or generated) resume once they
	// transform back.
	if target.MistForm {
		msgs = append(msgs, "A swirling cloud of wispy grey mist hangs in the air, formless and drifting.")
	} else if target.SlimeForm {
		msgs = append(msgs, "A shapeless, glistening mass of slime oozes slowly across the ground.")
	} else if target.WolfForm {
		ageWord := ageDescriptor(target.Age, target.Race)
		furColor := target.HairColor
		if target.HairStyle == "bald" {
			furColor = "black"
		}
		msgs = append(msgs, fmt.Sprintf("You see %s%s %s wolf with %s eyes and %s fur.",
			articleFor(ageWord, false), ageWord, strings.ToLower(genderName(target.Gender)), target.EyeColor, furColor))
	} else if !target.Disguised && (target.DescLine1 != "" || target.DescLine2 != "" || target.DescLine3 != "") {
		// Custom @line descriptions override the auto-generated race/gender/build/appearance
		// lines — but a worn disguise overrides those too, so it can't be used to reveal
		// a disguised player's real custom description.
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
		// Matches the authentic session-capture format, e.g. "You see Shirla Rennay,
		// a young female aelfen." — used even for self (the original says "You see
		// <name>...", not "You are...", when you look at yourself).
		ageWord := ageDescriptor(age, race)
		msgs = append(msgs, fmt.Sprintf("You see %s, %s%s %s %s.",
			displayName, articleFor(ageWord, false), ageWord, strings.ToLower(genderName(gender)), strings.ToLower(RaceNameByID(race))))

		msgs = append(msgs, fmt.Sprintf("%s %s, %s and %s.", pronoun,
			heightDescriptor(height, race), weightDescriptor(weight, race), buildDescriptor(strength, race)))

		subj, hasVerb, beVerb := "He", "has", "is"
		if isSelf {
			subj, hasVerb, beVerb = "You", "have", "are"
		} else if gender == GenderFemale {
			subj = "She"
		}
		if hairStyle == "bald" {
			msgs = append(msgs, fmt.Sprintf("%s %s %s eyes and %s skin. %s %s completely bald.",
				subj, hasVerb, eye, skin, subj, beVerb))
		} else {
			msgs = append(msgs, fmt.Sprintf("%s %s %s eyes, %s skin and %s %s hair.",
				subj, hasVerb, eye, skin, hairStyle, hairColor))
		}
	}

	heOrShe := "He"
	heOrSheLC := "him"
	if gender == GenderFemale {
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
		case target.Dead:
			msgs = append(msgs, "You are dead.")
		case target.Unconscious:
			msgs = append(msgs, "You are unconscious.")
		case healthPct >= 100:
			msgs = append(msgs, "You are in perfect health.")
		case healthPct >= 75:
			msgs = append(msgs, "You have minor injuries.")
		case healthPct >= 50:
			msgs = append(msgs, "You are moderately wounded.")
		case healthPct >= 25:
			msgs = append(msgs, "You are seriously wounded.")
		default:
			msgs = append(msgs, "You are critically wounded!")
		}
	} else {
		switch {
		case target.Dead:
			msgs = append(msgs, fmt.Sprintf("%s is dead.", heOrShe))
		case target.Unconscious:
			msgs = append(msgs, fmt.Sprintf("%s is unconscious.", heOrShe))
		case healthPct >= 100:
			msgs = append(msgs, fmt.Sprintf("%s appears to be in perfect health.", heOrShe))
		case healthPct >= 75:
			msgs = append(msgs, fmt.Sprintf("%s has minor injuries.", heOrShe))
		case healthPct >= 50:
			msgs = append(msgs, fmt.Sprintf("%s is moderately wounded.", heOrShe))
		case healthPct >= 25:
			msgs = append(msgs, fmt.Sprintf("%s is seriously wounded.", heOrShe))
		default:
			msgs = append(msgs, fmt.Sprintf("%s is critically wounded!", heOrShe))
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
	if len(target.Wounds) > 0 {
		woundSentence := buildWoundSentence(target.Wounds, "")
		if target.WolfForm {
			woundSentence = buildWolfWoundSentence(target.Wounds, "")
		}
		if isSelf {
			msgs = append(msgs, "You have "+woundSentence+".")
		} else {
			msgs = append(msgs, fmt.Sprintf("%s has %s.", heOrShe, woundSentence))
		}
	}

	// Guard status — player-guards-player
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.RoomNumber == target.RoomNumber && containsString(p.GuardTargets, target.FirstName) {
				if isSelf {
					msgs = append(msgs, fmt.Sprintf("You are being guarded by %s.", p.FirstName))
				} else {
					msgs = append(msgs, fmt.Sprintf("%s is being guarded by %s.", target.PronounCap(), p.FirstName))
				}
			}
		}
	}
	// Guard status — summoned creature-guards-player
	if e.monsterMgr != nil {
		e.monsterMgr.mu.RLock()
		for i := range e.monsterMgr.instances {
			g := &e.monsterMgr.instances[i]
			if g.IsSummoned && g.Alive && g.RoomNumber == target.RoomNumber && containsString(g.GuardingPlayers, target.FirstName) {
				gDef := e.monsters[g.DefNumber]
				if gDef != nil {
					gName := strings.ToLower(FormatMonsterName(gDef, e.monAdjs))
					gArticle := capArticle(articleFor(gName, gDef.Unique))
					if isSelf {
						msgs = append(msgs, fmt.Sprintf("You are being guarded by %s%s.", gArticle, gName))
					} else {
						msgs = append(msgs, fmt.Sprintf("%s is being guarded by %s%s.", target.PronounCap(), gArticle, gName))
					}
				}
			}
		}
		e.monsterMgr.mu.RUnlock()
	}

	// Active spell/psi effects
	mysticArmorActive := target.MysticArmorBonus > 0 && !target.MysticArmorExpiry.IsZero() && time.Now().Before(target.MysticArmorExpiry)
	if mysticArmorActive {
		msgs = append(msgs, fmt.Sprintf("%s outlined in glowing armor.", pronoun))
	}
	genericAura := false
	for _, b := range target.TimedDefenseBuffs {
		if time.Now().After(b.Expiry) {
			continue
		}
		switch b.SpellID {
		case 105: // Globe of Protection
			msgs = append(msgs, fmt.Sprintf("%s surrounded by a prismatic globe.", pronoun))
		case 130: // Mass Protection
			msgs = append(msgs, fmt.Sprintf("%s the center of a large, white sphere of light.", pronoun))
		case 326: // Spectral Shield
			msgs = append(msgs, fmt.Sprintf("%s protected by a ghostly, hovering shield.", pronoun))
		default:
			genericAura = true
		}
	}
	if genericAura {
		msgs = append(msgs, fmt.Sprintf("A shimmering magical aura surrounds %s.", isSelfOr(isSelf, "you", heOrSheLC)))
	}
	if target.StrengthBuffID > 0 && !target.StrengthBuffExpiry.IsZero() && time.Now().Before(target.StrengthBuffExpiry) {
		if isSelf {
			msgs = append(msgs, "You radiate with magical strength.")
		} else {
			msgs = append(msgs, fmt.Sprintf("%s radiates with magical strength.", heOrShe))
		}
	}
	if target.Position == 4 && target.Race != RaceDrakin {
		msgs = append(msgs, fmt.Sprintf("%s hovering in the air.", pronoun))
	}
	if target.Invisible {
		// Only visible to self or GMs
		if isSelf {
			msgs = append(msgs, "You are invisible.")
		}
	}
	if target.PhantomForm && isSelf {
		msgs = append(msgs, "You are in phantom form.")
	}

	// Equipment — a wolf isn't wearing or wielding anything, skip entirely.
	if !target.WolfForm && !target.MistForm && !target.SlimeForm && target.Wielded != nil {
		wDef := e.items[target.Wielded.Archetype]
		if wDef != nil {
			name := e.formatItemName(wDef, target.Wielded.Adj1, target.Wielded.Adj2, target.Wielded.Adj3, target.Wielded.Tail)
			if isSelf {
				msgs = append(msgs, fmt.Sprintf("You are wielding %s.", name))
			} else {
				msgs = append(msgs, fmt.Sprintf("%s wielding %s.", pronoun, name))
			}
		}
	}
	if !target.WolfForm && !target.MistForm && !target.SlimeForm && target.OffHand != nil {
		ohDef := e.items[target.OffHand.Archetype]
		if ohDef != nil {
			ohName := e.formatItemName(ohDef, target.OffHand.Adj1, target.OffHand.Adj2, target.OffHand.Adj3, target.OffHand.Tail)
			if ohDef.Type == "SHIELD" {
				if isSelf {
					msgs = append(msgs, fmt.Sprintf("You are carrying %s as a shield.", ohName))
				} else {
					msgs = append(msgs, fmt.Sprintf("%s carrying %s as a shield.", pronoun, ohName))
				}
			} else {
				if isSelf {
					msgs = append(msgs, fmt.Sprintf("You are wielding %s in your off hand.", ohName))
				} else {
					msgs = append(msgs, fmt.Sprintf("%s wielding %s in their off hand.", pronoun, ohName))
				}
			}
		}
	}

	if !target.WolfForm && !target.MistForm && !target.SlimeForm {
		// Build set of worn slots to determine undergarment visibility
		wornSlots := map[string]bool{}
		for _, w := range target.Worn {
			wornSlots[w.WornSlot] = true
		}
		var wornNames []string
		for _, worn := range target.Worn {
			if outerSlots, isUnder := undergarmentHiddenBy[worn.WornSlot]; isUnder {
				covered := false
				for _, outer := range outerSlots {
					if wornSlots[outer] {
						covered = true
						break
					}
				}
				if covered {
					continue
				}
			}
			wDef := e.items[worn.Archetype]
			if wDef != nil {
				wornNames = append(wornNames, e.formatItemName(wDef, worn.Adj1, worn.Adj2, worn.Adj3, worn.Tail))
			}
		}
		if len(wornNames) > 0 {
			if isSelf {
				msgs = append(msgs, fmt.Sprintf("You are wearing %s.", joinList(wornNames)))
			} else {
				msgs = append(msgs, fmt.Sprintf("%s wearing %s.", pronoun, joinList(wornNames)))
			}
		}
	}

	// Player-set custom appearance line (APPEARANCE command) — suppressed in
	// wolf form, same as @line1-3 above.
	if target.Appearance != "" && !target.WolfForm {
		msgs = append(msgs, target.Appearance)
	}

	return &CommandResult{Messages: msgs}
}

// formatItemNameNoArticle returns item name with adjectives but no article prefix.
func (e *GameEngine) formatItemNameNoArticle(def *gameworld.ItemDef, adj1, adj2, adj3 int, tail ...string) string {
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
	parts = append(parts, e.getItemNounName(def))
	name := strings.Join(parts, " ")
	if len(tail) > 0 && tail[0] != "" {
		name += " " + tail[0]
	}
	return name
}

func (e *GameEngine) formatItemName(def *gameworld.ItemDef, adj1, adj2, adj3 int, tail ...string) string {
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
	if len(tail) > 0 && tail[0] != "" {
		name += " " + tail[0]
	}
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

	// Run root-level IFVAR blocks (can contain nested IFVERB EXAMINE -1 blocks, e.g.
	// the lens of worlds' SHOWROOM preview — RunVerbScripts only matches top-level
	// IFVERB blocks, so verb-gated blocks nested inside an IFVAR tree need this too).
	sc0 := e.RunItemScripts(player, room, "EXAMINE", ri, def)
	if len(sc0.Messages) > 0 {
		result.Messages = append(result.Messages, sc0.Messages...)
	}
	if len(sc0.RoomMsgs) > 0 {
		result.RoomBroadcast = append(result.RoomBroadcast, sc0.RoomMsgs...)
	}

	// Run IFVERB LOOK scripts on the item (can add SHOWROOM output, etc.)
	sc := e.RunVerbScripts(player, room, "LOOK", ri, def)
	if len(sc.Messages) > 0 {
		result.Messages = append(result.Messages, sc.Messages...)
	}
	if len(sc.RoomMsgs) > 0 {
		result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
	}
	// PLREVENT/CONTPLREVENT-deferred actions must be scheduled, or everything after
	// the delay is lost.
	for _, segs := range [][]ScriptSegment{sc0.DeferredSegments, sc.DeferredSegments} {
		if len(segs) > 0 {
			e.scheduleScriptSegments(player, segs)
		}
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
	name := e.formatItemName(def, ri.Adj1, ri.Adj2, ri.Adj3, ri.Extend)
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
func matchesTargetOrdinal(nounName, cleanTarget string, skip *int, adjNames ...string) bool {
	if !matchesTarget(nounName, cleanTarget, adjNames...) {
		return false
	}
	if *skip > 0 {
		*skip--
		return false
	}
	return true
}

// stripMyPrefix detects a leading "my " in a target string (e.g. "my chest" from
// "look in my chest"). When present, the word is removed so noun-matching still
// works, and the returned bool tells the caller to restrict its search to the
// player's own possessions (inventory/worn/wielded) instead of also checking the room.
func stripMyPrefix(target string) (string, bool) {
	if rest, ok := strings.CutPrefix(target, "my "); ok {
		return strings.TrimSpace(rest), true
	}
	return target, false
}

// matchesTarget reports whether the player's target string refers to an item with
// the given noun and optional adjective names. The target may use any non-empty
// ordered subset of the provided adjectives as a prefix before the noun, e.g.
// "red panties", "icy broadsword", or "icy elkyri broadsword" all match correctly.
func matchesTarget(nounName, target string, adjNames ...string) bool {
	t := strings.ToLower(target)
	noun := strings.ToLower(nounName)

	// Collect non-empty adjective names in order.
	var adjs []string
	for _, a := range adjNames {
		if la := strings.ToLower(a); la != "" {
			adjs = append(adjs, la)
		}
	}

	// Match noun alone (adjectives optional).
	if nounMatches(noun, t) {
		return true
	}

	// Check each ordered subsequence of adjs prepended to the noun.
	for mask := 1; mask < (1 << len(adjs)); mask++ {
		var parts []string
		for i, a := range adjs {
			if mask&(1<<i) != 0 {
				parts = append(parts, a)
			}
		}
		prefix := strings.Join(parts, " ") + " "
		if strings.HasPrefix(t, prefix) && nounMatches(noun, t[len(prefix):]) {
			return true
		}
	}
	return false
}

// nounMatches reports whether target t refers to noun n (exact, prefix, last-word, or plural).
func nounMatches(noun, t string) bool {
	if t == noun {
		return true
	}
	if strings.HasPrefix(noun, t) {
		return true
	}
	if idx := strings.LastIndex(noun, " "); idx >= 0 {
		if strings.HasPrefix(noun[idx+1:], t) {
			return true
		}
	}
	if len(t) > 2 && t[len(t)-1] == 's' {
		stem := t[:len(t)-1]
		if stem == noun || strings.HasPrefix(noun, stem) {
			return true
		}
	}
	return false
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func containsInt(slice []int, n int) bool {
	for _, v := range slice {
		if v == n {
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
