package engine

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// messagesHaveRoundTime reports whether any message already displays a round-time
// indicator, so callers can skip synthesizing a duplicate "[Round: N sec]" line.
// Many original scripts bake this display directly into their own ECHO text (e.g.
// "You make your way across...%c [Round: 5 sec]"), separately from EQUAL ROUNDTIME
// setting sc.RoundTimeSet — appending both produces a doubled-up line for the player.
func messagesHaveRoundTime(msgs []string) bool {
	for _, m := range msgs {
		if strings.Contains(m, "[Round:") {
			return true
		}
	}
	return false
}

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

	destNum, ok := room.Exits[dir]
	if !ok {
		return &CommandResult{Messages: []string{"You can't go that way."}}
	}

	dest := e.rooms[destNum]
	if dest == nil {
		return &CommandResult{Messages: []string{"That way seems to lead nowhere."}}
	}

	originalRoom := player.RoomNumber
	dirNames := map[string]string{
		"N": "north", "S": "south", "E": "east", "W": "west",
		"NE": "northeast", "NW": "northwest", "SE": "southeast", "SW": "southwest",
		"U": "up", "D": "down", "O": "out",
		"ABOVE": "upward", "BELOW": "downward",
	}
	dirName := dirNames[dir]
	if dirName == "" {
		dirName = strings.ToLower(dir)
	}

	player.RoomNumber = destNum
	player.Submitting = false // moving clears submit state
	e.disengageCombat(player) // moving clears combat
	player.GuardTargets = nil // guard is room-specific
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

	// Relocate followers and summoned creatures BEFORE rendering any room look.
	// If this happened after doLook (as it used to), the leader's own room
	// render would list who's present in destNum before the group/summons had
	// actually been moved there, making it look like they didn't follow.
	var movedFollowers []*Player
	if player.IsGroupLeader && len(player.GroupMembers) > 0 && e.sessions != nil {
		for _, memberName := range player.GroupMembers {
			for _, p := range e.sessions.OnlinePlayers() {
				if p.FirstName == memberName && p.RoomNumber == originalRoom && !p.Dead {
					p.RoomNumber = destNum
					p.Submitting = false
					e.disengageCombat(p)
					e.SavePlayer(ctx, p)
					movedFollowers = append(movedFollowers, p)
					break
				}
			}
		}
	}
	var summonOldMsgs, summonRoomMsgs []string
	if e.monsterMgr != nil {
		e.monsterMgr.mu.Lock()
		for i := range e.monsterMgr.instances {
			inst := &e.monsterMgr.instances[i]
			if inst.Alive && inst.IsSummoned && inst.FollowTarget == player.FirstName && inst.RoomNumber == originalRoom {
				def := e.monsters[inst.DefNumber]
				if def != nil {
					cname := strings.ToLower(FormatMonsterName(def, e.monAdjs))
					carticle := articleFor(cname, def.Unique)
					summonOldMsgs = append(summonOldMsgs, fmt.Sprintf("%s%s follows %s %s.", capArticle(carticle), cname, player.FirstName, dirName))
					summonRoomMsgs = append(summonRoomMsgs, fmt.Sprintf("%s%s follows %s in.", capArticle(carticle), cname, player.FirstName))
				}
				e.monsterMgr.moveMonster(i, destNum)
			}
		}
		e.monsterMgr.mu.Unlock()
	}

	// A carried player travels along, silently — they're passive cargo, not
	// an active participant, so no entry scripts run for them.
	if player.Carrying != "" && e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.FirstName == player.Carrying && p.RoomNumber == originalRoom {
				p.RoomNumber = destNum
				e.SavePlayer(ctx, p)
				if e.sendToPlayer != nil {
					carriedLook := e.doLook(p)
					e.sendToPlayer(p.FirstName, carriedLook.Messages)
				}
				break
			}
		}
	}

	result := e.doLook(player)
	result.OldRoom = originalRoom
	// Concealed players (Invisible spell, @hide, @invis) move silently — no exit/entry echoes
	if !player.IsConcealed() {
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

	// Group movement echoes + each follower's own room render. Everyone's
	// RoomNumber was already updated above, so each follower's look (and the
	// leader's) correctly shows the whole group and any summons as present.
	if len(movedFollowers) > 0 {
		for _, p := range movedFollowers {
			if e.sendToPlayer != nil {
				followLook := e.doLook(p)
				e.sendToPlayer(p.FirstName, followLook.Messages)
			}
			e.applyEntryScripts(ctx, p, dest, &CommandResult{})
		}
		result.OldRoomMsg = append(result.OldRoomMsg, fmt.Sprintf("%s's group goes %s.", player.FirstName, dirName))
		result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s's group arrives.", player.FirstName))
	}

	result.OldRoomMsg = append(result.OldRoomMsg, summonOldMsgs...)
	result.RoomBroadcast = append(result.RoomBroadcast, summonRoomMsgs...)

	if player.Carrying != "" {
		result.OldRoomMsg = append(result.OldRoomMsg, fmt.Sprintf("%s carries %s %s.", player.FirstName, player.Carrying, dirName))
		result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s carries %s in.", player.FirstName, player.Carrying))
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

	// If IFENTRY moved the player synchronously (no PLREVENT delay), resolve the new room.
	if sc.MoveTo > 0 {
		if newRoom := e.rooms[sc.MoveTo]; newRoom != nil {
			lookResult := e.doLook(player)
			result.Messages = append(result.Messages, lookResult.Messages...)
			result.RoomName = lookResult.RoomName
			result.RoomDesc = lookResult.RoomDesc
			result.Exits = lookResult.Exits
			result.Items = lookResult.Items
			e.applyEntryScripts(ctx, player, newRoom, result)
		}
	}

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
			// Run remaining steps; if another SETEVENT/CONTEVENT or PLREVENT/CONTPLREVENT
			// pair is found, more segments are deferred (sc.DeferredSegments, chained below).
			sc.execSteps(seg.Steps)
			// Persist player state if modified (EQUAL) or moved (MOVE).
			if sc.NeedsSave || sc.MoveTo > 0 {
				e.SavePlayer(context.Background(), player)
			}
			// Deliver messages. For PLREVENT→MOVE sequences the player has been relocated;
			// always send transition messages since they describe the transit itself.
			if len(sc.Messages) > 0 && e.sendToPlayer != nil {
				e.sendToPlayer(player.FirstName, sc.Messages)
			}
			// If script moved the player, show the new room and run its entry scripts.
			if sc.MoveTo > 0 {
				if newRoom := e.rooms[sc.MoveTo]; newRoom != nil {
					if lookResult := e.doLook(player); e.sendToPlayer != nil {
						e.sendToPlayer(player.FirstName, lookResult.Messages)
					}
					e.applyEntryScripts(context.Background(), player, newRoom, &CommandResult{})
				}
			}
			// Deliver room broadcast messages, excluding player — ECHO ALL populates both
			// sc.Messages (sent directly above) and sc.RoomMsgs, so without the exclusion
			// player would receive the same line twice.
			if len(sc.RoomMsgs) > 0 {
				if e.roomBroadcastExclude != nil {
					e.roomBroadcastExclude(seg.RoomNumber, player.FirstName, sc.RoomMsgs)
				} else if e.roomBroadcast != nil {
					e.roomBroadcast(seg.RoomNumber, sc.RoomMsgs)
				}
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
		// PLREVENT/CONTPLREVENT-deferred actions must be scheduled, or everything
		// after the delay is lost.
		if len(sc.DeferredSegments) > 0 {
			e.scheduleScriptSegments(player, sc.DeferredSegments)
		}
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
				originalRoom := player.RoomNumber
				player.RoomNumber = sc.MoveTo
				e.SavePlayer(ctx, player)
				lookResult := e.doLook(player)
				result.Messages = append(result.Messages, lookResult.Messages...)
				result.RoomName = lookResult.RoomName
				result.RoomDesc = lookResult.RoomDesc
				result.Exits = lookResult.Exits
				result.Items = lookResult.Items
				result.OldRoom = originalRoom
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
		if posName == "" {
			posName = "not standing"
		}
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
		if skip > 0 {
			skip--
			continue
		}
		if isPortal(itemDef.Type) {
			return e.doGoPortal(ctx, player, room, &room.Items[i], itemDef)
		}
		// Non-portal item matched — run both IFPREVERB GO and IFVERB GO scripts.
		// Copy ri so that a REMOVEITEM inside the first script (e.g. the secret-passage
		// idiom: reveal a hidden item, then remove it once stepped through) can't shrink
		// room.Items out from under the second call's stale index.
		// Capture the room being left before the scripts run — a script MOVE (as used by
		// the secret-passage idiom) sets player.RoomNumber directly, so reading it after
		// would report the destination instead of the room being left.
		originalRoom := player.RoomNumber
		riCopy := ri
		sc := e.RunPreverbScripts(player, room, "GO", &riCopy, itemDef)
		sc2 := e.RunVerbScripts(player, room, "GO", &riCopy, itemDef)
		// PLREVENT/CONTPLREVENT-deferred actions must be scheduled, or everything
		// after the delay is lost — regardless of which branch below is taken.
		for _, segs := range [][]ScriptSegment{sc.DeferredSegments, sc2.DeferredSegments} {
			if len(segs) > 0 {
				e.scheduleScriptSegments(player, segs)
			}
		}
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
		moveGroupTo := sc.MoveGroupTo
		if sc2.MoveGroupTo > 0 {
			moveGroupTo = sc2.MoveGroupTo
		}
		if moveGroupTo > 0 {
			e.moveGroupToRoom(ctx, player.RoomNumber, moveGroupTo)
			return result
		}
		if blocked && moveTo == 0 {
			// CLEARVERB without MOVE or MOVEGROUP — block the normal action (deferred
			// segments were already scheduled above, regardless of branch).
			roundTimeSet := sc.RoundTimeSet
			if sc2.RoundTimeSet > 0 {
				roundTimeSet = sc2.RoundTimeSet
			}
			if roundTimeSet > 0 && !messagesHaveRoundTime(result.Messages) {
				result.Messages = append(result.Messages, fmt.Sprintf("[Round: %d sec]", roundTimeSet))
			}
			if len(result.Messages) == 0 {
				result.Messages = []string{"You can't go that way."}
			}
			return result
		}
		if moveTo > 0 {
			dest := e.rooms[moveTo]
			if dest != nil {
				player.RoomNumber = moveTo
				e.SavePlayer(ctx, player)
				lookResult := e.doLook(player)
				result.Messages = append(result.Messages, lookResult.Messages...)
				result.RoomName = lookResult.RoomName
				result.RoomDesc = lookResult.RoomDesc
				result.Exits = lookResult.Exits
				result.Items = lookResult.Items
				result.OldRoom = originalRoom
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

	// Capture room before running scripts — MOVE in scripts can change player.RoomNumber directly.
	originalRoom := player.RoomNumber

	// Run IFPREVERB GO scripts (can CLEARVERB to block)
	sc := e.RunPreverbScripts(player, room, "GO", ri, itemDef)
	// PLREVENT/CONTPLREVENT-deferred actions must be scheduled, or everything after
	// the delay is lost — regardless of which branch below is taken.
	if len(sc.DeferredSegments) > 0 {
		e.scheduleScriptSegments(player, sc.DeferredSegments)
	}
	result := &CommandResult{}
	if len(sc.Messages) > 0 {
		result.Messages = append(result.Messages, sc.Messages...)
	}
	if len(sc.GMMsgs) > 0 {
		result.GMBroadcast = append(result.GMBroadcast, sc.GMMsgs...)
	}
	if sc.MoveGroupTo > 0 {
		// sc.RoomMsgs here are pre-MOVEGROUP echoes (e.g. "%N goes through the hole")
		// describing the departure — they belong in the room being left, not wherever
		// the player ends up after moveGroupToRoom updates player.RoomNumber below.
		if len(sc.RoomMsgs) > 0 {
			result.OldRoom = originalRoom
			result.OldRoomMsg = append(result.OldRoomMsg, sc.RoomMsgs...)
		}
		if sc.RoundTimeSet > 0 && !messagesHaveRoundTime(result.Messages) {
			result.Messages = append(result.Messages, fmt.Sprintf("[Round: %d sec]", sc.RoundTimeSet))
		}
		e.moveGroupToRoom(ctx, player.RoomNumber, sc.MoveGroupTo)
		return result
	}
	if len(sc.RoomMsgs) > 0 {
		result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
	}
	if sc.Blocked && sc.MoveTo == 0 {
		// (Deferred segments were already scheduled above, regardless of branch.)
		if sc.RoundTimeSet > 0 && !messagesHaveRoundTime(result.Messages) {
			result.Messages = append(result.Messages, fmt.Sprintf("[Round: %d sec]", sc.RoundTimeSet))
		}
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
		player.RoomNumber = originalRoom
		result.Messages = append(result.Messages, "That doesn't seem to lead anywhere.")
		return result
	}
	dest := e.rooms[destNum]
	if dest == nil {
		// A portal script (e.g. MOVE ITEMVAL2) may have already set player.RoomNumber to the
		// non-existent room. Restore it so the player isn't stranded in a void.
		player.RoomNumber = originalRoom
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

	portalName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3, ri.Extend)
	player.GuardTargets = nil // guard is room-specific; clear on movement
	player.GuardPortals = nil
	player.GuardItems = nil
	player.RoomNumber = destNum
	e.SavePlayer(ctx, player)

	// Relocate followers and summoned creatures BEFORE rendering any room look
	// (see doMove for why — otherwise the group looks like it didn't follow).
	var movedFollowers []*Player
	if player.IsGroupLeader && len(player.GroupMembers) > 0 && e.sessions != nil {
		for _, memberName := range player.GroupMembers {
			for _, p := range e.sessions.OnlinePlayers() {
				if p.FirstName == memberName && p.RoomNumber == originalRoom && !p.Dead {
					p.RoomNumber = destNum
					p.Submitting = false
					e.disengageCombat(p)
					e.SavePlayer(ctx, p)
					movedFollowers = append(movedFollowers, p)
					break
				}
			}
		}
	}
	var summonOldMsgs, summonRoomMsgs []string
	if e.monsterMgr != nil {
		e.monsterMgr.mu.Lock()
		for i := range e.monsterMgr.instances {
			inst := &e.monsterMgr.instances[i]
			if inst.Alive && inst.IsSummoned && inst.FollowTarget == player.FirstName && inst.RoomNumber == originalRoom {
				def := e.monsters[inst.DefNumber]
				if def != nil {
					cname := strings.ToLower(FormatMonsterName(def, e.monAdjs))
					carticle := articleFor(cname, def.Unique)
					summonOldMsgs = append(summonOldMsgs, fmt.Sprintf("%s%s follows %s through %s.", capArticle(carticle), cname, player.FirstName, portalName))
					summonRoomMsgs = append(summonRoomMsgs, fmt.Sprintf("%s%s follows %s in.", capArticle(carticle), cname, player.FirstName))
				}
				e.monsterMgr.moveMonster(i, destNum)
			}
		}
		e.monsterMgr.mu.Unlock()
	}

	// A carried player travels through the portal along, silently.
	if player.Carrying != "" && e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.FirstName == player.Carrying && p.RoomNumber == originalRoom {
				p.RoomNumber = destNum
				e.SavePlayer(ctx, p)
				if e.sendToPlayer != nil {
					carriedLook := e.doLook(p)
					e.sendToPlayer(p.FirstName, carriedLook.Messages)
				}
				break
			}
		}
	}

	lookResult := e.doLook(player)
	result.Messages = append(result.Messages, lookResult.Messages...)
	result.RoomName = lookResult.RoomName
	result.RoomDesc = lookResult.RoomDesc
	result.Exits = lookResult.Exits
	result.Items = lookResult.Items
	result.OldRoom = originalRoom
	result.OldRoomMsg = append(result.OldRoomMsg, fmt.Sprintf("%s goes through %s.", player.FirstName, portalName))
	result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s arrives.", player.FirstName))

	// Run IFENTRY scripts at destination
	e.applyEntryScripts(ctx, player, dest, result)

	// Group movement echoes + each follower's own room render (everyone's
	// RoomNumber was already updated above, so it correctly shows the group).
	if len(movedFollowers) > 0 {
		for _, p := range movedFollowers {
			if e.sendToPlayer != nil {
				followLook := e.doLook(p)
				e.sendToPlayer(p.FirstName, followLook.Messages)
			}
			e.applyEntryScripts(ctx, p, dest, &CommandResult{})
		}
		result.OldRoomMsg = append(result.OldRoomMsg, fmt.Sprintf("%s's group goes through %s.", player.FirstName, portalName))
		result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s's group arrives.", player.FirstName))
	}

	result.OldRoomMsg = append(result.OldRoomMsg, summonOldMsgs...)
	result.RoomBroadcast = append(result.RoomBroadcast, summonRoomMsgs...)

	if player.Carrying != "" {
		result.OldRoomMsg = append(result.OldRoomMsg, fmt.Sprintf("%s carries %s through %s.", player.FirstName, player.Carrying, portalName))
		result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s carries %s in.", player.FirstName, player.Carrying))
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
		if posName == "" {
			posName = "not standing"
		}
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
			if skip > 0 {
				skip--
				continue
			}

			// Run IFVERB CLIMB scripts first (some rooms use IFVERB instead of IFPREVERB)
			sc := e.RunVerbScripts(player, room, "CLIMB", &room.Items[i], itemDef)
			// If IFVERB CLIMB produced nothing, try IFPREVERB CLIMB
			if sc.MoveTo == 0 && !sc.Blocked && len(sc.Messages) == 0 && len(sc.RoomMsgs) == 0 {
				sc = e.RunPreverbScripts(player, room, "CLIMB", &room.Items[i], itemDef)
			}
			// PLREVENT/CONTPLREVENT-deferred actions must be scheduled, or everything
			// after the delay is lost.
			if len(sc.DeferredSegments) > 0 {
				e.scheduleScriptSegments(player, sc.DeferredSegments)
			}

			// If climb scripts handled it, apply the result
			if sc.MoveTo > 0 || sc.Blocked || sc.MoveGroupTo > 0 || len(sc.Messages) > 0 || len(sc.RoomMsgs) > 0 {
				result := &CommandResult{}
				result.Messages = append(result.Messages, sc.Messages...)
				result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
				result.GMBroadcast = append(result.GMBroadcast, sc.GMMsgs...)

				if sc.MoveGroupTo > 0 {
					// MOVEGROUP: move entire group to destination. Post-MOVEGROUP echoes
					// were already suppressed in doEcho via moveGroupFired, so sc.Messages
					// only contains pre-success messages. Append round time then move.
					if sc.RoundTimeSet > 0 && !messagesHaveRoundTime(result.Messages) {
						result.Messages = append(result.Messages, fmt.Sprintf("[Round: %d sec]", sc.RoundTimeSet))
					}
					e.moveGroupToRoom(ctx, player.RoomNumber, sc.MoveGroupTo)
					return result
				}

				if sc.MoveTo > 0 {
					dest := e.rooms[sc.MoveTo]
					if dest != nil {
						originalRoom := player.RoomNumber
						player.RoomNumber = sc.MoveTo
						e.SavePlayer(ctx, player)
						lookResult := e.doLook(player)
						result.Messages = append(result.Messages, lookResult.Messages...)
						result.RoomName = lookResult.RoomName
						result.RoomDesc = lookResult.RoomDesc
						result.Exits = lookResult.Exits
						result.Items = lookResult.Items
						result.OldRoom = originalRoom
						result.OldRoomMsg = []string{fmt.Sprintf("%s leaves.", player.FirstName)}
						result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s arrives.", player.FirstName))
						e.applyEntryScripts(ctx, player, dest, result)
					}
				} else if sc.NeedsSave {
					e.SavePlayer(ctx, player)
				}

				if sc.RoundTimeSet > 0 && !messagesHaveRoundTime(result.Messages) {
					result.Messages = append(result.Messages, fmt.Sprintf("[Round: %d sec]", sc.RoundTimeSet))
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
