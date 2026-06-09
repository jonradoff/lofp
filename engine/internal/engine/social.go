package engine

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// doEmoteWithScripts checks object and room scripts before falling back to processEmote.
// Priority: (1) item/room scripts when a target is named, (2) room bare-verb scripts, (3) processEmote.
// Used by emote verbs (KICK, GAZE, LISTEN, etc.) that can also trigger scripted interactions.
func (e *GameEngine) doEmoteWithScripts(ctx context.Context, player *Player, verb string, args []string) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if len(args) > 0 {
		target := strings.ToLower(strings.Join(args, " "))
		if result := e.runVerbScriptsForTarget(ctx, player, room, verb, target); result != nil {
			return result
		}
	}
	// Check room-level bare-verb scripts (IFVERB/IFPREVERB VERB -1)
	if room != nil {
		sc := e.RunRoomVerbScripts(player, room, verb)
		if sc.Blocked || len(sc.Messages) > 0 || len(sc.RoomMsgs) > 0 {
			result := &CommandResult{Messages: sc.Messages, RoomBroadcast: sc.RoomMsgs, GMBroadcast: sc.GMMsgs}
			if sc.Blocked && len(result.Messages) == 0 {
				result.Messages = []string{"You can't do that."}
			}
			if !sc.Blocked {
				// Script ran but didn't block — also run the normal emote
				emResult := e.processEmote(player, verb, args)
				result.Messages = append(result.Messages, emResult.Messages...)
				result.RoomBroadcast = append(result.RoomBroadcast, emResult.RoomBroadcast...)
				result.TargetName = emResult.TargetName
				result.TargetMsg = emResult.TargetMsg
			}
			return result
		}
	}
	return e.processEmote(player, verb, args)
}

// runVerbScriptsForTarget finds a named target in the room or player inventory and runs all
// verb scripts (root IFVAR, IFPREVERB, IFVERB) on it. Returns nil if the target is not found
// or if the scripts produce no output and do not block the action. Used by doEmoteWithScripts
// so that emote verbs check scripts before falling back to the emote table.
func (e *GameEngine) runVerbScriptsForTarget(ctx context.Context, player *Player, room *gameworld.Room, verb string, target string) *CommandResult {
	if room == nil {
		return nil
	}
	target, skip := parseOrdinal(target)

	// Search room items
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			if skip > 0 { skip--; continue }
			result := &CommandResult{}
			sc0 := e.RunItemScripts(player, room, &room.Items[i], itemDef)
			sc1 := e.RunPreverbScripts(player, room, verb, &room.Items[i], itemDef)
			sc2 := e.RunVerbScripts(player, room, verb, &room.Items[i], itemDef)
			result.Messages = append(result.Messages, sc0.Messages...)
			result.Messages = append(result.Messages, sc1.Messages...)
			result.Messages = append(result.Messages, sc2.Messages...)
			result.RoomBroadcast = append(result.RoomBroadcast, sc0.RoomMsgs...)
			result.RoomBroadcast = append(result.RoomBroadcast, sc1.RoomMsgs...)
			result.RoomBroadcast = append(result.RoomBroadcast, sc2.RoomMsgs...)
			result.GMBroadcast = append(result.GMBroadcast, sc0.GMMsgs...)
			result.GMBroadcast = append(result.GMBroadcast, sc1.GMMsgs...)
			result.GMBroadcast = append(result.GMBroadcast, sc2.GMMsgs...)
			blocked := sc0.Blocked || sc1.Blocked || sc2.Blocked
			moveTo := sc0.MoveTo
			if sc1.MoveTo > 0 { moveTo = sc1.MoveTo }
			if sc2.MoveTo > 0 { moveTo = sc2.MoveTo }
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
				return result
			}
			if blocked {
				if len(result.Messages) == 0 {
					result.Messages = []string{"You can't do that."}
				}
				return result
			}
			if len(result.Messages) > 0 || len(result.RoomBroadcast) > 0 {
				return result
			}
			return nil // Item found but no scripts produced output — fall back to emote
		}
	}

	// Search player items (inventory, worn, wielded)
	allPlayerItems := make([]InventoryItem, 0, len(player.Inventory)+len(player.Worn)+1)
	allPlayerItems = append(allPlayerItems, player.Inventory...)
	allPlayerItems = append(allPlayerItems, player.Worn...)
	if player.Wielded != nil {
		allPlayerItems = append(allPlayerItems, *player.Wielded)
	}
	for _, ii := range allPlayerItems {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) || matchesTarget(name, target, e.getAdjName(ii.Adj3)) {
			if skip > 0 { skip--; continue }
			tempRI := gameworld.RoomItem{Ref: -1, Archetype: ii.Archetype,
				Adj1: ii.Adj1, Adj2: ii.Adj2, Adj3: ii.Adj3,
				Val1: ii.Val1, Val2: ii.Val2, Val3: ii.Val3, Val4: ii.Val4, Val5: ii.Val5}
			result := &CommandResult{}
			sc0 := e.RunItemScripts(player, room, &tempRI, itemDef)
			sc1 := e.RunPreverbScripts(player, room, verb, &tempRI, itemDef)
			sc2 := e.RunVerbScripts(player, room, verb, &tempRI, itemDef)
			result.Messages = append(result.Messages, sc0.Messages...)
			result.Messages = append(result.Messages, sc1.Messages...)
			result.Messages = append(result.Messages, sc2.Messages...)
			result.RoomBroadcast = append(result.RoomBroadcast, sc0.RoomMsgs...)
			result.RoomBroadcast = append(result.RoomBroadcast, sc1.RoomMsgs...)
			result.RoomBroadcast = append(result.RoomBroadcast, sc2.RoomMsgs...)
			blocked := sc0.Blocked || sc1.Blocked || sc2.Blocked
			if blocked {
				if len(result.Messages) == 0 {
					result.Messages = []string{"You can't do that."}
				}
				return result
			}
			if len(result.Messages) > 0 || len(result.RoomBroadcast) > 0 {
				return result
			}
			return nil // Item found but no scripts produced output
		}
	}

	return nil // Target not found — let caller fall back to emote
}

func (e *GameEngine) doWhisper(player *Player, args []string, rawInput string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Whisper to whom?"}}
	}
	targetName := strings.ToLower(args[0])

	// Proximity whisper: "whisper close ..." or "whisper those ..."
	if targetName == "close" || targetName == "those" {
		text := extractRawArgs(rawInput, 2)
		if text == "" {
			return &CommandResult{Messages: []string{"Whisper what?"}}
		}
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You whisper to those close, \"%s\"", text)},
			RoomBroadcast: []string{fmt.Sprintf("%s whispers to those close, \"%s\"", player.FirstName, text)},
		}
	}

	found := e.findPlayerInRoom(player, targetName)
	if found == nil {
		return &CommandResult{Messages: []string{"You don't see that person here."}}
	}
	// Get the whisper text (everything after the target name)
	text := extractRawArgs(rawInput, 2)
	if text == "" {
		return &CommandResult{Messages: []string{"Whisper what?"}}
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You whisper to %s.", found.FirstName)},
		RoomBroadcast: []string{fmt.Sprintf("%s whispers to %s.", player.FirstName, found.FirstName)},
		TargetName:    found.FirstName, // exclude target from room broadcast — they get WhisperMsg instead
		WhisperTarget: found.FirstName,
		WhisperMsg:    fmt.Sprintf("%s whispers to you, \"%s\"", player.FirstName, text),
	}
}

func (e *GameEngine) doYell(player *Player, args []string, rawInput string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Yell what?"}}
	}
	text := extractRawArgs(rawInput, 1)
	adverb := ""
	if player.SpeechAdverb != "" {
		adverb = player.SpeechAdverb + " "
	}

	// Yell is heard in adjacent rooms too
	room := e.rooms[player.RoomNumber]
	if room != nil && e.roomBroadcast != nil {
		adjacentMsg := fmt.Sprintf("You hear someone yell, \"%s\"", text)
		for _, destNum := range room.Exits {
			if destNum > 0 && destNum != player.RoomNumber {
				e.roomBroadcast(destNum, []string{adjacentMsg})
			}
		}
	}

	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You %syell, \"%s\"", adverb, text)},
		RoomBroadcast: []string{fmt.Sprintf("%s %syells, \"%s\"", player.FirstName, adverb, text)},
	}
}

// doPray handles the PRAY command — triggers IFVERB PRAY scripts or generic prayer.
func (e *GameEngine) doPray(player *Player) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room != nil {
		sc := &ScriptContext{Player: player, Room: room, Engine: e}
		for _, block := range room.Scripts {
			if block.Type == "IFVERB" && len(block.Args) >= 2 {
				if strings.ToUpper(block.Args[0]) == "PRAY" && block.Args[1] == "-1" {
					sc.execBlock(block)
				}
			}
		}
		if len(sc.Messages) > 0 {
			result := &CommandResult{Messages: sc.Messages}
			if len(sc.RoomMsgs) > 0 {
				result.RoomBroadcast = sc.RoomMsgs
			}
			return result
		}
	}
	pronoun := "his"
	if player.Gender == GenderFemale {
		pronoun = "her"
	}
	return &CommandResult{
		Messages:      []string{"You pray."},
		RoomBroadcast: []string{fmt.Sprintf("%s bows %s head and prays.", player.FirstName, pronoun)},
	}
}

// doContact handles the CONTACT command — psionic telepathic whisper.
func (e *GameEngine) doContact(player *Player, args []string, rawInput string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Contact whom with what message?"}}
	}
	// CONTACT requires psionic ability (Psionics skill 26 or any psionic school skill)
	if player.Skills[26] < 1 && player.Skills[27] < 1 && player.Skills[28] < 1 && player.Skills[29] < 1 {
		return &CommandResult{Messages: []string{"You do not possess psionic abilities."}}
	}
	targetName := strings.ToLower(args[0])
	// Find the target among all online players (not just same room)
	var found *Player
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.FirstName == player.FirstName && p.LastName == player.LastName {
				continue
			}
			if strings.HasPrefix(strings.ToLower(p.FirstName), targetName) {
				found = p
				break
			}
		}
	}
	if found == nil {
		return &CommandResult{Messages: []string{"You cannot sense that person."}}
	}
	text := extractRawArgs(rawInput, 2)
	if text == "" {
		return &CommandResult{Messages: []string{"Contact whom with what message?"}}
	}
	player.RoundTimeExpiry = time.Now().Add(2 * time.Second)
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You contact %s with your thoughts.", found.FirstName), "[Round: 2 sec]"},
		WhisperTarget: found.FirstName,
		WhisperMsg:    fmt.Sprintf("You feel the touch of %s's mind: \"%s\"", player.FirstName, text),
	}
}

// doFollow handles the FOLLOW command — join a group.
func (e *GameEngine) doFollow(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Follow whom?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	found := e.findPlayerInRoom(player, target)
	if found == nil {
		return &CommandResult{Messages: []string{"They are not here."}}
	}
	if player.Following != "" {
		return &CommandResult{Messages: []string{fmt.Sprintf("You are already following %s.", player.Following)}}
	}
	if found.Following == player.FirstName {
		return &CommandResult{Messages: []string{"You can't follow someone who is following you."}}
	}
	player.Following = found.FirstName
	found.IsGroupLeader = true
	// Add to leader's group members (avoid duplicates)
	alreadyIn := false
	for _, m := range found.GroupMembers {
		if m == player.FirstName {
			alreadyIn = true
			break
		}
	}
	if !alreadyIn {
		found.GroupMembers = append(found.GroupMembers, player.FirstName)
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You are now following %s.", found.FirstName)},
		RoomBroadcast: []string{fmt.Sprintf("%s is now following %s.", player.FirstName, found.FirstName)},
	}
}

// doHold handles the HOLD command (group) — leader adds a member.
func (e *GameEngine) doHold(player *Player, found *Player) *CommandResult {
	if found.Following != "" {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s is already following someone.", found.FirstName)}}
	}
	found.Following = player.FirstName
	player.IsGroupLeader = true
	alreadyIn := false
	for _, m := range player.GroupMembers {
		if m == found.FirstName {
			alreadyIn = true
			break
		}
	}
	if !alreadyIn {
		player.GroupMembers = append(player.GroupMembers, found.FirstName)
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("%s takes %s by the hand.", player.FirstName, found.FirstName)},
		RoomBroadcast: []string{fmt.Sprintf("%s takes %s by the hand.", player.FirstName, found.FirstName)},
	}
}

// moveGroupToRoom moves all online players in srcRoom to destRoom (for MOVEGROUP script command).
func (e *GameEngine) moveGroupToRoom(ctx context.Context, srcRoom, destRoom int) {
	dest := e.rooms[destRoom]
	if dest == nil || e.sessions == nil {
		return
	}
	for _, p := range e.sessions.OnlinePlayers() {
		if p.RoomNumber == srcRoom && !p.Dead {
			p.RoomNumber = destRoom
			p.Submitting = false
			e.disengageCombat(p)
			e.SavePlayer(ctx, p)
			if e.sendToPlayer != nil {
				lookResult := e.doLook(p)
				e.sendToPlayer(p.FirstName, lookResult.Messages)
			}
			e.applyEntryScripts(ctx, p, dest, &CommandResult{})
		}
	}
}

func (e *GameEngine) removeFromGroup(player *Player) {
	if player.Following == "" {
		return
	}
	leaderName := player.Following
	player.Following = ""
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.FirstName == leaderName {
				for i, m := range p.GroupMembers {
					if m == player.FirstName {
						p.GroupMembers = append(p.GroupMembers[:i], p.GroupMembers[i+1:]...)
						break
					}
				}
				if len(p.GroupMembers) == 0 {
					p.IsGroupLeader = false
				}
				break
			}
		}
	}
}

// doLeave handles the LEAVE command — stop following.
func (e *GameEngine) doLeave(player *Player) *CommandResult {
	if player.Following == "" {
		return &CommandResult{Messages: []string{"You are not following anyone."}}
	}
	leaderName := player.Following
	e.removeFromGroup(player)
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You stop following %s.", leaderName)},
		RoomBroadcast: []string{fmt.Sprintf("%s stops following %s.", player.FirstName, leaderName)},
	}
}

// doDisband handles the DISBAND command — leader disbands their group.
func (e *GameEngine) doDisband(player *Player) *CommandResult {
	if !player.IsGroupLeader || len(player.GroupMembers) == 0 {
		return &CommandResult{Messages: []string{"You don't have a group to disband."}}
	}
	// Clear Following on all members
	if e.sessions != nil {
		for _, memberName := range player.GroupMembers {
			for _, p := range e.sessions.OnlinePlayers() {
				if p.FirstName == memberName {
					p.Following = ""
					if e.sendToPlayer != nil {
						e.sendToPlayer(p.FirstName, []string{fmt.Sprintf("%s disbands the group.", player.FirstName)})
					}
					break
				}
			}
		}
	}
	player.GroupMembers = nil
	player.IsGroupLeader = false
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You disband your group.")},
		RoomBroadcast: []string{fmt.Sprintf("%s disbands their group.", player.FirstName)},
	}
}

func (e *GameEngine) doWho(player *Player) *CommandResult {
	var names []string
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.GMHidden {
				continue
			}
			name := p.FirstName
			if p.IsBot {
				name += " [Bot]"
			}
			if p.IsGM && p.GMHat {
				name += " [Host]"
			}
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return &CommandResult{Messages: []string{"No adventurers are in the Realms."}}
	}
	sort.Strings(names)
	// Build 4-column grid, 19-char columns
	var msgs []string
	for i := 0; i < len(names); i += 4 {
		line := ""
		for j := 0; j < 4 && i+j < len(names); j++ {
			line += fmt.Sprintf("%-19s", names[i+j])
		}
		msgs = append(msgs, line)
	}
	msgs = append(msgs, "")
	count := len(names)
	if count == 1 {
		msgs = append(msgs, "There is 1 adventurer in the Realms.")
	} else {
		msgs = append(msgs, fmt.Sprintf("There are %d adventurers in the Realms.", count))
	}
	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) doHelp() *CommandResult {
	return &CommandResult{Messages: []string{
		"=== Legends of Future Past - Commands ===",
		"Movement: N, S, E, W, NE, NW, SE, SW, UP, DOWN, OUT, GO <portal>",
		"Looking: LOOK, LOOK <item>, LOOK IN/ON/UNDER <item>, EXAMINE <item>",
		"Items: GET <item>, DROP <item>, INVENTORY, WIELD <weapon>, UNWIELD",
		"Wear: WEAR <item>, REMOVE <item>",
		"Containers: OPEN <item>, CLOSE <item>, LOOK IN <item>",
		"           GET <item> FROM <container>, GET ALL FROM <container>",
		"           PUT <item> IN <container>, DUMP <container>",
		"           GET ALL, GET ALL <noun>",
		"Selling:   SELL <item>, SELL ALL <noun>",
		"Info: STATUS, HEALTH, WEALTH, SKILLS, WHO",
		"Combat: ATTACK <target>, ADVANCE <target>, RETREAT",
		"Social: '<message> (say), ACT <action>, WHISPER <person> <msg>",
		"Position: SIT, STAND, KNEEL, LAY",
		"Settings: BRIEF, FULL",
		"System: HELP, ADVICE, QUIT",
	}}
}

func (e *GameEngine) doThink(player *Player, rawInput string) *CommandResult {
	text := extractOriginalArgs(rawInput)
	if text == "" {
		return &CommandResult{Messages: []string{"Think what?"}}
	}
	if !player.TelepathyActive {
		return &CommandResult{Messages: []string{"You don't have telepathic ability right now."}}
	}
	return &CommandResult{
		Messages:        []string{"You project your thoughts."},
		TelepathyMsg:    text,
		TelepathySender: player.FirstName,
	}
}

func (e *GameEngine) doCant(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Cant what?"}}
	}
	// Requires Legerdemain (skill 21) rank 6+ or Stealth (skill 5)
	if player.Skills[21] < 6 && player.Skills[5] < 1 && !player.IsGM {
		return &CommandResult{Messages: []string{"You don't know how to speak in cant."}}
	}
	text := strings.Join(args, " ")
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You cant, \"%s\"", text)},
		RoomBroadcast: []string{fmt.Sprintf("%s mutters something under their breath.", player.FirstName)},
		CantMsg:       text,
		CantSender:    player.FirstName,
	}
}

func (e *GameEngine) doSpeech(ctx context.Context, player *Player, args []string, rawInput string) *CommandResult {
	if len(args) == 0 {
		if player.SpeechAdverb != "" {
			return &CommandResult{Messages: []string{fmt.Sprintf("Your speech manner is: %s. Use SPEECH CLEAR to remove it.", player.SpeechAdverb)}}
		}
		return &CommandResult{Messages: []string{"You have no speech manner set. Use SPEECH <adverb> to set one (e.g. SPEECH gently)."}}
	}
	adverb := strings.ToLower(args[0])
	if adverb == "clear" || adverb == "none" || adverb == "off" {
		player.SpeechAdverb = ""
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"Speech manner cleared."}}
	}
	player.SpeechAdverb = adverb
	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: []string{fmt.Sprintf("You will now %s say things.", adverb)}}
}

func (e *GameEngine) doInfo(player *Player) *CommandResult {
	return &CommandResult{Messages: []string{
		fmt.Sprintf("Name: %s", player.FullName()),
		fmt.Sprintf("Race: %s   Gender: %s   Level: %d", player.RaceName(), genderName(player.Gender), player.Level),
		"",
		fmt.Sprintf("Strength: %-3d   Agility: %-3d   Quickness: %d", player.Strength, player.Agility, player.Quickness),
		fmt.Sprintf("Constitution: %-3d   Perception: %-3d   Willpower: %-3d   Empathy: %d", player.Constitution, player.Perception, player.Willpower, player.Empathy),
		"",
		fmt.Sprintf("Body Points: %d/%d   Fatigue: %d/%d", player.BodyPoints, player.MaxBodyPoints, player.Fatigue, player.MaxFatigue),
		fmt.Sprintf("Mana: %d/%d   Psi: %d/%d", player.Mana, player.MaxMana, player.Psi, player.MaxPsi),
		"",
		fmt.Sprintf("Experience: %d   Build Points: %d", player.Experience, player.Experience/100),
	}}
}

func isSelfOr(isSelf bool, selfText, otherText string) string {
	if isSelf { return selfText }
	return otherText
}

func genderName(g int) string {
	switch g {
	case 0:
		return "Male"
	case 1:
		return "Female"
	default:
		return "Unknown"
	}
}
