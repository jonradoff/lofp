package engine

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jonradoff/lofp/internal/gameworld"
)

func (e *GameEngine) doMove(ctx context.Context, player *Player, dir string) *CommandResult {
	if player.Immobilized {
		return &CommandResult{Messages: []string{"You are immobilized and cannot move!"}}
	}
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := int(player.RoundTimeExpiry.Sub(time.Now()).Seconds()) + 1
		return &CommandResult{Messages: []string{fmt.Sprintf("[Wait %d seconds...]", remaining)}}
	}
	// Normal movement reveals hidden players (but not Ethereal Projection — that's psi-maintained)
	if player.Hidden && !player.EtherealActive {
		player.Hidden = false
	}
	if player.Position != 0 && player.Position != 4 { // 4 = flying, can move
		posNames := map[int]string{1: "sitting", 2: "laying down", 3: "kneeling"}
		posName := posNames[player.Position]
		if posName == "" {
			posName = "not standing"
		}
		return &CommandResult{Messages: []string{fmt.Sprintf("You can't move while %s! Try STANDing first.", posName)}}
	}

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Error: "You are nowhere!"}
	}

	// Also check ABOVE/BELOW for U/D
	destNum, ok := room.Exits[dir]
	requiresFlight := false
	if !ok {
		if dir == "U" {
			destNum, ok = room.Exits["ABOVE"]
			if ok {
				requiresFlight = true
			}
		} else if dir == "D" {
			destNum, ok = room.Exits["BELOW"]
		}
	}
	if !ok {
		return &CommandResult{Messages: []string{"You can't go that way."}}
	}
	if requiresFlight && !player.IsFlying() {
		return &CommandResult{Messages: []string{"You leap into the air but come crashing back down. You need to be able to fly to go that way."}}
	}

	dest := e.rooms[destNum]
	if dest == nil {
		return &CommandResult{Messages: []string{"That way seems to lead nowhere."}}
	}

	oldRoom := player.RoomNumber
	dirNames := map[string]string{
		"N": "north", "S": "south", "E": "east", "W": "west",
		"NE": "northeast", "NW": "northwest", "SE": "southeast", "SW": "southwest",
		"U": "up", "D": "down", "O": "out", "ABOVE": "up", "BELOW": "down",
	}
	dirName := dirNames[dir]
	if dirName == "" {
		dirName = strings.ToLower(dir)
	}

	player.RoomNumber = destNum
	player.Submitting = false // moving clears submit state
	e.disengageCombat(player)  // moving clears combat
	player.GuardTargets = nil  // guard is room-specific
	player.GuardPortals = nil
	player.GuardItems = nil

	// Moving away from leader breaks follow
	if player.Following != "" {
		leaderHere := false
		if e.sessions != nil {
			for _, p := range e.sessions.OnlinePlayers() {
				if p.FirstName == player.Following && p.RoomNumber == destNum {
					leaderHere = true
					break
				}
			}
		}
		if !leaderHere {
			e.removeFromGroup(player)
		}
	}
	e.SavePlayer(ctx, player)
	result := e.doLook(player)
	result.OldRoom = oldRoom
	// Invisible GMs move silently — no exit/entry echoes
	if !player.GMInvis {
		if player.ExitEcho != "" {
			result.OldRoomMsg = []string{player.ExitEcho}
		} else {
			result.OldRoomMsg = []string{fmt.Sprintf("%s goes %s.", player.FirstName, dirName)}
		}
		if player.EntryEcho != "" {
			result.RoomBroadcast = []string{player.EntryEcho}
		} else {
			result.RoomBroadcast = []string{fmt.Sprintf("%s arrives.", player.FirstName)}
		}
	}

	// Run IFENTRY scripts for the destination room
	e.applyEntryScripts(ctx, player, dest, result)

	// Group movement: if leader has followers, move them too
	if player.IsGroupLeader && len(player.GroupMembers) > 0 && e.sessions != nil {
		groupDir := dirName
		for _, memberName := range player.GroupMembers {
			for _, p := range e.sessions.OnlinePlayers() {
				if p.FirstName == memberName && p.RoomNumber == oldRoom && !p.Dead {
					p.RoomNumber = destNum
					p.Submitting = false
					e.disengageCombat(p)
					e.SavePlayer(ctx, p)
					// Send the follower a look at the new room
					if e.sendToPlayer != nil {
						followLook := e.doLook(p)
						e.sendToPlayer(p.FirstName, followLook.Messages)
					}
					e.applyEntryScripts(ctx, p, dest, &CommandResult{})
					break
				}
			}
		}
		result.OldRoomMsg = append(result.OldRoomMsg, fmt.Sprintf("%s's group goes %s.", player.FirstName, groupDir))
		result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s's group arrives.", player.FirstName))
	}

	return result
}

// EnterRoom performs a look and runs IFENTRY scripts. Used on login/creation.
func (e *GameEngine) EnterRoom(ctx context.Context, player *Player) *CommandResult {
	// Show date and time on entry
	period := "day"
	if IsNight() {
		period = "night"
	}
	weather := strings.ToLower(e.GetRoomWeather(player.RoomNumber))
	var timeMsg string
	if weather != "" {
		timeMsg = fmt.Sprintf("It is %s %d, %d. It is %s. %s",
			GameMonthName(), GameDay()%28+1, GameYear(), period, weather)
	} else {
		timeMsg = fmt.Sprintf("It is %s %d, %d. It is %s.",
			GameMonthName(), GameDay()%28+1, GameYear(), period)
	}

	result := e.doLook(player)
	result.Messages = append([]string{timeMsg}, result.Messages...)
	room := e.rooms[player.RoomNumber]
	if room != nil {
		e.applyEntryScripts(ctx, player, room, result)
	}
	return result
}

// GetRoom returns the room struct for GMCP/protocol data. Returns nil if not found.
func (e *GameEngine) GetRoom(roomNumber int) *gameworld.Room {
	return e.rooms[roomNumber]
}

// buildActiveMonsterLists combines base MLISTs with the current season's MLISTs.
func (e *GameEngine) buildActiveMonsterLists() []gameworld.MonsterList {
	lists := make([]gameworld.MonsterList, len(e.baseMonsterLists))
	copy(lists, e.baseMonsterLists)
	if seasonal, ok := e.seasonalMonsterLists[e.currentSeason]; ok {
		lists = append(lists, seasonal...)
	}
	return lists
}

// CheckSeasonChange checks if the game season has changed and hot-swaps MLISTs.
func (e *GameEngine) CheckSeasonChange() {
	newSeason := GameSeason()
	if newSeason == e.currentSeason {
		return
	}
	oldSeason := e.currentSeason
	e.currentSeason = newSeason
	e.monsterLists = e.buildActiveMonsterLists()

	// Apply seasonal room overrides
	e.applySeasonalRooms()

	log.Printf("Season changed: %s -> %s. Active MLISTs: %d", oldSeason, newSeason, len(e.monsterLists))
	e.Events.Publish("time", fmt.Sprintf("The season has changed to %s.", SeasonName()))

	// Broadcast season change to outdoor players
	seasonMessages := map[string]string{
		"PSCRIPT": "The chill of winter recedes as spring arrives in the Shattered Realms. New growth appears across the land.",
		"SSCRIPT": "The warmth of summer settles over the Shattered Realms. The days grow long and hot.",
		"ASCRIPT": "A cool breeze heralds the arrival of autumn. Leaves begin to turn golden and crimson across the land.",
		"WSCRIPT": "Winter descends upon the Shattered Realms. A bitter cold wind sweeps across the land.",
	}
	if msg, ok := seasonMessages[newSeason]; ok {
		e.broadcastOutdoor(msg)
	}
}

// applySeasonalRooms applies seasonal room overrides for the current season.
// Seasonal scripts define room descriptions, exits, and items that change with the season.
func (e *GameEngine) applySeasonalRooms() {
	rooms, ok := e.seasonalRooms[e.currentSeason]
	if !ok || len(rooms) == 0 {
		return
	}
	count := 0
	for i := range rooms {
		r := &rooms[i]
		if existing := e.rooms[r.Number]; existing != nil {
			// Override description and terrain but preserve dynamic state
			existing.Name = r.Name
			existing.Description = r.Description
			existing.Terrain = r.Terrain
			existing.Exits = r.Exits
			existing.MonsterGroup = r.MonsterGroup
			if len(r.Items) > 0 {
				existing.Items = r.Items
			}
			if len(r.Modifiers) > 0 {
				existing.Modifiers = r.Modifiers
			}
			if len(r.ItemDescriptions) > 0 {
				existing.ItemDescriptions = r.ItemDescriptions
			}
			if len(r.Scripts) > 0 {
				existing.Scripts = r.Scripts
			}
			count++
		} else {
			// New room from seasonal script
			e.rooms[r.Number] = r
			count++
		}
	}
	if count > 0 {
		log.Printf("Applied %d seasonal room overrides for %s", count, SeasonName())
	}
}

// applyEntryScripts runs IFENTRY scripts and merges results into the command result.
func (e *GameEngine) applyEntryScripts(ctx context.Context, player *Player, room *gameworld.Room, result *CommandResult) {
	sc := e.RunEntryScripts(player, room)
	if len(sc.Messages) > 0 {
		result.Messages = append(result.Messages, sc.Messages...)
		e.Events.Publish("script", fmt.Sprintf("IFENTRY fired for %s in room %d (%s)", player.FirstName, room.Number, room.Name))
	}
	if len(sc.RoomMsgs) > 0 {
		result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
	}
	if len(sc.GMMsgs) > 0 {
		result.GMBroadcast = append(result.GMBroadcast, sc.GMMsgs...)
	}
	if sc.MoveGroupTo > 0 {
		e.moveGroupToRoom(ctx, room.Number, sc.MoveGroupTo)
	}
	e.SavePlayer(ctx, player)

	// Schedule any SETEVENT/CONTEVENT-deferred script segments.
	if len(sc.DeferredSegments) > 0 {
		e.scheduleScriptSegments(player, sc.DeferredSegments)
	}

	// Spawn monsters for this room if needed (demand-based)
	e.spawnForRoom(room.Number)

	// Check if hostile monsters should aggro on the player entering the room
	go e.monsterCheckAggro(player, room.Number)
}

// scheduleScriptSegments fires SETEVENT/CONTEVENT-deferred script segments in background goroutines.
// Each segment runs RelativeSeconds after the previous one fires (relative, not cumulative).
func (e *GameEngine) scheduleScriptSegments(player *Player, segments []ScriptSegment) {
	for _, seg := range segments {
		seg := seg // capture loop variable
		time.AfterFunc(time.Duration(seg.RelativeSeconds)*time.Second, func() {
			room := e.rooms[seg.RoomNumber]
			if room == nil {
				return
			}
			sc := &ScriptContext{
				Player: player,
				Room:   room,
				Engine: e,
			}
			// Run actions; if another SETEVENT/CONTEVENT pair is found, more segments are deferred.
			if !sc.execActionsUntilDelay(seg.Actions, seg.Children) {
				for _, child := range seg.Children {
					sc.execBlock(child)
				}
			}
			// Persist player state if modified (e.g. EQUAL INTNUM).
			if sc.NeedsSave {
				e.SavePlayer(context.Background(), player)
			}
			// Deliver direct-to-player messages only if player is still in this room.
			if len(sc.Messages) > 0 && e.sendToPlayer != nil && player.RoomNumber == seg.RoomNumber {
				e.sendToPlayer(player.FirstName, sc.Messages)
			}
			// Deliver room broadcast messages.
			if len(sc.RoomMsgs) > 0 && e.roomBroadcast != nil {
				e.roomBroadcast(seg.RoomNumber, sc.RoomMsgs)
			}
			// Chain further deferred segments recursively.
			if len(sc.DeferredSegments) > 0 {
				e.scheduleScriptSegments(player, sc.DeferredSegments)
			}
		})
	}
}

func (e *GameEngine) doSteal(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Steal what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Error: "You are nowhere!"}
	}

	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	// Check for scripted items first (e.g., alley portals that use STEAL as a stealth move verb)
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ri.Adj1), e.getAdjName(ri.Adj2), e.getAdjName(ri.Adj3)) {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}
		sc := e.RunPreverbScripts(player, room, "STEAL", &room.Items[i], itemDef)
		result := &CommandResult{}
		result.Messages = append(result.Messages, sc.Messages...)
		result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
		result.GMBroadcast = append(result.GMBroadcast, sc.GMMsgs...)
		if sc.MoveGroupTo > 0 {
			e.moveGroupToRoom(ctx, player.RoomNumber, sc.MoveGroupTo)
			return result
		}
		if sc.MoveTo > 0 {
			dest := e.rooms[sc.MoveTo]
			if dest != nil {
				oldRoom := player.RoomNumber
				player.RoomNumber = sc.MoveTo
				e.SavePlayer(ctx, player)
				lookResult := e.doLook(player)
				result.Messages = append(result.Messages, lookResult.Messages...)
				result.RoomName = lookResult.RoomName
				result.RoomDesc = lookResult.RoomDesc
				result.Exits = lookResult.Exits
				result.Items = lookResult.Items
				result.OldRoom = oldRoom
				result.OldRoomMsg = []string{fmt.Sprintf("%s slips away.", player.FirstName)}
				result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s slips in from somewhere.", player.FirstName))
				e.applyEntryScripts(ctx, player, dest, result)
			}
		}
		if len(result.Messages) == 0 {
			result.Messages = []string{"You can't steal that."}
		}
		return result
	}

	// No scripted item matched — pick-pockets / creature steal (TODO)
	return &CommandResult{Messages: []string{"[Stealing from creatures coming soon.]"}}
}

func (e *GameEngine) doGo(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Go where?"}}
	}
	if player.Position != 0 && player.Position != 4 {
		posNames := map[int]string{1: "sitting", 2: "laying down", 3: "kneeling"}
		posName := posNames[player.Position]
		if posName == "" { posName = "not standing" }
		return &CommandResult{Messages: []string{fmt.Sprintf("You can't move while %s! Try STANDing first.", posName)}}
	}
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := int(player.RoundTimeExpiry.Sub(time.Now()).Seconds()) + 1
		return &CommandResult{Messages: []string{fmt.Sprintf("[Wait %d seconds...]", remaining)}}
	}

	target := strings.ToLower(strings.Join(args, " "))
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Error: "You are nowhere!"}
	}

	// Try direction first
	dirMap := map[string]string{
		"north": "N", "south": "S", "east": "E", "west": "W",
		"northeast": "NE", "northwest": "NW", "southeast": "SE", "southwest": "SW",
		"up": "U", "down": "D", "out": "O",
	}
	if dir, ok := dirMap[target]; ok {
		return e.doMove(ctx, player, dir)
	}

	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	// Try portals (doors, trails, arches, etc.)
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ri.Adj1), e.getAdjName(ri.Adj2), e.getAdjName(ri.Adj3)) {
			continue
		}
		if skip > 0 { skip--; continue }
		if isPortal(itemDef.Type) {
			return e.doGoPortal(ctx, player, room, &room.Items[i], itemDef)
		}
		// Non-portal item matched — run both IFPREVERB GO and IFVERB GO scripts.
		sc := e.RunPreverbScripts(player, room, "GO", &room.Items[i], itemDef)
		sc2 := e.RunVerbScripts(player, room, "GO", &room.Items[i], itemDef)
		result := &CommandResult{}
		result.Messages = append(result.Messages, sc.Messages...)
		result.Messages = append(result.Messages, sc2.Messages...)
		result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
		result.RoomBroadcast = append(result.RoomBroadcast, sc2.RoomMsgs...)
		result.GMBroadcast = append(result.GMBroadcast, sc.GMMsgs...)
		result.GMBroadcast = append(result.GMBroadcast, sc2.GMMsgs...)
		if sc.NeedsSave || sc2.NeedsSave {
			e.SavePlayer(ctx, player)
		}
		moveTo := sc.MoveTo
		if sc2.MoveTo > 0 {
			moveTo = sc2.MoveTo
		}
		blocked := (sc.Blocked || sc2.Blocked)
		if sc.MoveGroupTo > 0 {
			e.moveGroupToRoom(ctx, player.RoomNumber, sc.MoveGroupTo)
			return result
		}
		if blocked && moveTo == 0 {
			// CLEARVERB without MOVE or MOVEGROUP — block the action
			if len(result.Messages) == 0 {
				result.Messages = []string{"You can't go that way."}
			}
			return result
		}
		if moveTo > 0 {
			dest := e.rooms[moveTo]
			if dest != nil {
				oldRoom := player.RoomNumber
				player.RoomNumber = moveTo
				e.SavePlayer(ctx, player)
				lookResult := e.doLook(player)
				result.Messages = append(result.Messages, lookResult.Messages...)
				result.RoomName = lookResult.RoomName
				result.RoomDesc = lookResult.RoomDesc
				result.Exits = lookResult.Exits
				result.Items = lookResult.Items
				result.OldRoom = oldRoom
				result.OldRoomMsg = []string{fmt.Sprintf("%s leaves.", player.FirstName)}
				result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s arrives.", player.FirstName))
				e.applyEntryScripts(ctx, player, dest, result)
			}
		}
		if len(result.Messages) == 0 {
			result.Messages = []string{"You can't go that way."}
		}
		return result
	}

	return &CommandResult{Messages: []string{"You don't see that here."}}
}

func (e *GameEngine) doGoPortal(ctx context.Context, player *Player, room *gameworld.Room, ri *gameworld.RoomItem, itemDef *gameworld.ItemDef) *CommandResult {
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := int(player.RoundTimeExpiry.Sub(time.Now()).Seconds()) + 1
		return &CommandResult{Messages: []string{fmt.Sprintf("[Wait %d seconds...]", remaining)}}
	}
	// Check if portal is closed
	state := strings.ToUpper(ri.State)
	if state == "CLOSED" || state == "LOCKED" {
		portalName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3, ri.Extend)
		return &CommandResult{Messages: []string{fmt.Sprintf("The %s is closed.", e.getItemNounName(itemDef))}, RoomBroadcast: []string{fmt.Sprintf("%s bumps into %s.", player.FirstName, portalName)}}
	}

	// Run IFPREVERB GO scripts (can CLEARVERB to block)
	sc := e.RunPreverbScripts(player, room, "GO", ri, itemDef)
	result := &CommandResult{}
	if len(sc.Messages) > 0 {
		result.Messages = append(result.Messages, sc.Messages...)
	}
	if len(sc.RoomMsgs) > 0 {
		result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
	}
	if len(sc.GMMsgs) > 0 {
		result.GMBroadcast = append(result.GMBroadcast, sc.GMMsgs...)
	}
	if sc.Blocked && sc.MoveTo == 0 {
		if len(result.Messages) == 0 {
			result.Messages = []string{"You can't go that way."}
		}
		return result
	}

	// Script MOVE overrides destination
	destNum := ri.Val2
	if sc.MoveTo > 0 {
		destNum = sc.MoveTo
	}

	if destNum <= 0 {
		result.Messages = append(result.Messages, "That doesn't seem to lead anywhere.")
		return result
	}
	dest := e.rooms[destNum]
	if dest == nil {
		result.Messages = append(result.Messages, "That doesn't seem to lead anywhere.")
		return result
	}

	// Check if any player is guarding this portal (Combat Maneuvering level 3+)
	if guardBlocked, guardPlayerMsgs, guardOldRoomMsgs := e.checkPortalGuard(player, ri.Archetype); guardBlocked {
		result.Messages = append(result.Messages, guardPlayerMsgs...)
		result.RoomBroadcast = append(result.RoomBroadcast, guardOldRoomMsgs...)
		return result
	} else if len(guardOldRoomMsgs) > 0 {
		// Bypass succeeded — show roll result to mover before the room description
		result.Messages = append(result.Messages, guardPlayerMsgs...)
		result.OldRoomMsg = append(result.OldRoomMsg, guardOldRoomMsgs...)
	}

	oldRoom := player.RoomNumber
	portalName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3, ri.Extend)
	player.GuardTargets = nil // guard is room-specific; clear on movement
	player.GuardPortals = nil
	player.GuardItems = nil
	player.RoomNumber = destNum
	e.SavePlayer(ctx, player)
	lookResult := e.doLook(player)
	result.Messages = append(result.Messages, lookResult.Messages...)
	result.RoomName = lookResult.RoomName
	result.RoomDesc = lookResult.RoomDesc
	result.Exits = lookResult.Exits
	result.Items = lookResult.Items
	result.OldRoom = oldRoom
	result.OldRoomMsg = append(result.OldRoomMsg, fmt.Sprintf("%s goes through %s.", player.FirstName, portalName))
	result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s arrives.", player.FirstName))

	// Run IFENTRY scripts at destination
	e.applyEntryScripts(ctx, player, dest, result)

	// Group movement: if leader has followers, move them through the portal too
	if player.IsGroupLeader && len(player.GroupMembers) > 0 && e.sessions != nil {
		for _, memberName := range player.GroupMembers {
			for _, p := range e.sessions.OnlinePlayers() {
				if p.FirstName == memberName && p.RoomNumber == oldRoom && !p.Dead {
					p.RoomNumber = destNum
					p.Submitting = false
					e.disengageCombat(p)
					e.SavePlayer(ctx, p)
					if e.sendToPlayer != nil {
						followLook := e.doLook(p)
						e.sendToPlayer(p.FirstName, followLook.Messages)
					}
					e.applyEntryScripts(ctx, p, dest, &CommandResult{})
					break
				}
			}
		}
		result.OldRoomMsg = append(result.OldRoomMsg, fmt.Sprintf("%s's group goes through %s.", player.FirstName, portalName))
		result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s's group arrives.", player.FirstName))
	}

	return result
}

func (e *GameEngine) doClimb(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Climb what?"}}
	}
	if player.Position != 0 && player.Position != 4 {
		posNames := map[int]string{1: "sitting", 2: "laying down", 3: "kneeling"}
		posName := posNames[player.Position]
		if posName == "" { posName = "not standing" }
		return &CommandResult{Messages: []string{fmt.Sprintf("You can't climb while %s! Try STANDing first.", posName)}}
	}
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := int(player.RoundTimeExpiry.Sub(time.Now()).Seconds()) + 1
		return &CommandResult{Messages: []string{fmt.Sprintf("[Wait %d seconds...]", remaining)}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}

	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ri.Adj1), e.getAdjName(ri.Adj2), e.getAdjName(ri.Adj3)) {
			if skip > 0 { skip--; continue }

			// Run IFVERB CLIMB scripts first (some rooms use IFVERB instead of IFPREVERB)
			sc := e.RunVerbScripts(player, room, "CLIMB", &room.Items[i], itemDef)
			// If IFVERB CLIMB produced nothing, try IFPREVERB CLIMB
			if sc.MoveTo == 0 && !sc.Blocked && len(sc.Messages) == 0 && len(sc.RoomMsgs) == 0 {
				sc = e.RunPreverbScripts(player, room, "CLIMB", &room.Items[i], itemDef)
			}

			// If climb scripts handled it, apply the result
			if sc.MoveTo > 0 || sc.Blocked || len(sc.Messages) > 0 || len(sc.RoomMsgs) > 0 {
				result := &CommandResult{}
				result.Messages = append(result.Messages, sc.Messages...)
				result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
				result.GMBroadcast = append(result.GMBroadcast, sc.GMMsgs...)
				if sc.MoveTo > 0 {
					dest := e.rooms[sc.MoveTo]
					if dest != nil {
						oldRoom := player.RoomNumber
						player.RoomNumber = sc.MoveTo
						e.SavePlayer(ctx, player)
						lookResult := e.doLook(player)
						result.Messages = append(result.Messages, lookResult.Messages...)
						result.RoomName = lookResult.RoomName
						result.RoomDesc = lookResult.RoomDesc
						result.Exits = lookResult.Exits
						result.Items = lookResult.Items
						result.OldRoom = oldRoom
						result.OldRoomMsg = []string{fmt.Sprintf("%s leaves.", player.FirstName)}
						result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s arrives.", player.FirstName))
						e.applyEntryScripts(ctx, player, dest, result)
					}
				}
				if len(result.Messages) == 0 {
					result.Messages = []string{"You can't climb that."}
				}
				return result
			}

			// No CLIMB scripts fired — fall back to portal behavior (uses Val2)
			if isPortal(itemDef.Type) {
				return e.doGoPortal(ctx, player, room, &room.Items[i], itemDef)
			}

			return &CommandResult{Messages: []string{"You can't climb that."}}
		}
	}

	return &CommandResult{Messages: []string{"You don't see that here."}}
}
