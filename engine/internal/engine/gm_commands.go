package engine

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	"github.com/jonradoff/lofp/internal/gameworld"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// processGMCommand dispatches all @-prefixed GM commands.
func (e *GameEngine) processGMCommand(ctx context.Context, player *Player, verb string, args []string, rawInput string) *CommandResult {
	verb = resolveGMVerb(verb)
	switch verb {
	case "@HELP":
		return e.gmHelp()
	case "@GO":
		return e.gmGo(ctx, player, args)
	case "@ADDITEM":
		return e.gmAddItem(ctx, player, args)
	case "@DELETE":
		return e.gmDelete(ctx, player, args)
	case "@RDATA":
		return e.gmRData(player, args)
	case "@HEAL":
		return e.gmHeal(ctx, player, args)
	case "@KILL":
		return e.gmKill(ctx, player, args)
	case "@EXP":
		return e.gmExp(ctx, player, args)
	case "@GM":
		player.GMHat = true
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"You don your Host Hat. You are now visible as a GM."}}
	case "@RFLAG":
		player.GMHat = false
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"You remove your Host Hat."}}
	case "@HIDE":
		player.GMHidden = true
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"You are now hidden from the WHO list."}}
	case "@UNHIDE":
		player.GMHidden = false
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"You are now visible on the WHO list."}}
	case "@INVIS":
		player.GMInvis = true
		player.Hidden = true
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"You fade from sight."}}
	case "@VIS":
		player.GMInvis = false
		player.Hidden = false
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"You become visible again."}}
	case "@SND":
		if len(args) == 0 {
			return &CommandResult{Messages: []string{"Usage: @snd <text>"}}
		}
		text := extractRawArgs(rawInput, 1)
		return &CommandResult{Messages: []string{text}}
	case "@ANNOUNCE":
		return e.gmAnnounce(player, args, rawInput)
	case "@BANNER":
		return e.gmBanner(player, args, rawInput)
	case "@WHO":
		return e.gmWho(ctx)
	case "@LWHO":
		return e.gmLWho(ctx)
	case "@NUM":
		return e.gmNum(ctx, args)
	case "@QSTAT":
		return e.gmQStat(ctx, args)
	case "@PINV":
		return e.gmPInv(ctx, args)
	case "@GENMON":
		return e.gmGenMon(player, args)
	case "@SPAWN":
		return e.gmSpawn(player, args)
	case "@ACTIVATE":
		return &CommandResult{Messages: []string{"Monster activated."}}
	case "@SEDATE":
		return &CommandResult{Messages: []string{"Monster sedated."}}
	case "@ZAP":
		return e.gmZap(player, args)
	case "@MLIST":
		return e.gmMList()
	case "@FIND":
		return e.gmFind(args)
	case "@LIST":
		return e.gmList()
	case "@EXAMINE":
		return e.gmExamine(args)
	case "@GLOSSARY":
		return e.gmGlossary(args)
	case "@PEEK":
		return e.gmPeek(player, args)
	case "@SET":
		return e.gmSet(ctx, player, args)
	case "@RND":
		return e.gmRnd(args)
	case "@OPEN":
		return e.gmOpenCloseLock(player, args, "OPEN")
	case "@CLOSE":
		return e.gmOpenCloseLock(player, args, "CLOSED")
	case "@LOCK":
		return e.gmOpenCloseLock(player, args, "LOCKED")
	case "@UNLOCK":
		return e.gmOpenCloseLock(player, args, "UNLOCKED")
	case "@GOPLR":
		return e.gmGoPlr(ctx, player, args)
	case "@YANK":
		return e.gmYank(ctx, player, args)
	case "@WHISPER":
		return e.gmWhisper(args, rawInput)
	case "@EDPLAYER", "@EDPL":
		return e.gmEdPlayer(ctx, args)
	case "@EDS", "@EDSK":
		return e.gmEds(ctx, args)
	case "@LSK":
		return e.gmLsk()
	case "@GRANTSP":
		return e.gmGrantSp(ctx, args)
	case "@PSI":
		return e.gmPsi(ctx, args)
	case "@ECHOPLR":
		return e.gmEchoPlr(args, rawInput)
	case "@EXCLUDE":
		return e.gmExclude(args, rawInput)
	case "@SPEECH":
		return e.gmSpeech(ctx, player, args, rawInput)
	case "@LINE1":
		return e.gmSetLine(ctx, player, args, rawInput, 1)
	case "@LINE2":
		return e.gmSetLine(ctx, player, args, rawInput, 2)
	case "@LINE3":
		return e.gmSetLine(ctx, player, args, rawInput, 3)
	case "@ENTRY":
		return e.gmSetEntryExit(ctx, player, args, rawInput, "entry")
	case "@EXIT":
		return e.gmSetEntryExit(ctx, player, args, rawInput, "exit")
	case "@SUGGEST":
		return &CommandResult{Messages: []string{"Suggestion recorded. Thank you!"}}
	case "@MSG":
		return &CommandResult{Messages: []string{"Host message viewing toggled."}}
	case "@SAVE":
		return &CommandResult{Messages: []string{"NPC slot saved."}}
	case "@RESTORE":
		return &CommandResult{Messages: []string{"NPC slot restored."}}
	case "@REGISTER":
		return &CommandResult{Messages: []string{"Player registered."}}
	case "@ASSIST?":
		return &CommandResult{Messages: []string{"No pending assist requests."}}
	case "@OLDCOMP":
		return &CommandResult{Messages: []string{"Script compilation is not available in this version."}}
	case "@EDITEM":
		return &CommandResult{Messages: []string{"Item editor not yet implemented."}}
	case "@EDN":
		return &CommandResult{Messages: []string{"Item editor not yet implemented."}}
	case "@GET":
		return e.gmGet(ctx, player, args)
	case "@LOOK":
		return e.gmLookContainer(player, args)
	case "@QUEUE":
		return &CommandResult{Messages: []string{"Monster queue updated."}}
	case "@UNQUEUE":
		return &CommandResult{Messages: []string{"Item removed from monster queue."}}
	case "@TRACE":
		player.GMTrace = !player.GMTrace
		if player.GMTrace {
			return &CommandResult{Messages: []string{"Script tracing ON. You will see debug output for script execution."}}
		}
		return &CommandResult{Messages: []string{"Script tracing OFF."}}
	case "@TITLE":
		return e.gmTitle(ctx, player, args, rawInput)
	case "@VERB", "@VERBS":
		return e.gmVerbs()
	default:
		return &CommandResult{Messages: []string{fmt.Sprintf("Unknown GM command: %s", strings.ToLower(verb))}}
	}
}

// extractRawArgs gets the raw input text after skipping N words.
func extractRawArgs(rawInput string, skip int) string {
	fields := strings.Fields(rawInput)
	if len(fields) <= skip {
		return ""
	}
	return strings.Join(fields[skip:], " ")
}

func (e *GameEngine) gmHelp() *CommandResult {
	return &CommandResult{Messages: []string{
		"=== GM Commands (alphabetical) ===",
		"@activate              - Activate a sedated monster",
		"@additem <archnum>     - Add item to current room",
		"@announce <mode> <msg> - Announce (1=global 2=mindlink)",
		"@close <item>          - Close item silently",
		"@delete <item>         - Delete an item from the room",
		"@echoplr <name> <text> - Echo text to a player",
		"@edpl <name>           - Show/edit player fields",
		"@edsk <name> <sk> <lv> - Set a player's skill level",
		"@examine <#>           - Show type info for a number",
		"@exclude <name> <text> - Echo to room except player",
		"@exp <name> <points>   - Grant experience",
		"@find <archnum>        - Find all instances of an item",
		"@genmon <monster#>     - Generate monster (sedated)",
		"@get <record#>         - Pick up item by record number",
		"@glossary <word>       - Look up a noun/adj by name",
		"@gm                    - Put on Host Hat (visible as GM)",
		"@go <room#>            - Teleport to a room",
		"@goplr <name>          - Teleport to a player",
		"@grantsp <name> <sp#>  - Give spell to player",
		"@heal <name>           - Heal a player to full",
		"@help                  - This help listing",
		"@hide / @unhide        - Hide/show on WHO list",
		"@invis / @vis          - Become invisible/visible",
		"@kill <name>           - Kill a player",
		"@list                  - List all items in game",
		"@lock <item>           - Lock item silently",
		"@look <record#>        - Look inside a container",
		"@lsk                   - List all skills with IDs",
		"@lwho                  - Detailed player list with rooms",
		"@mlist                 - List all spawned monsters",
		"@msg                   - Toggle host messages",
		"@num <name>            - Show player info by name",
		"@open <item>           - Open item silently",
		"@peek <variable>       - View a variable value",
		"@pinv <name>           - View player inventory",
		"@psi <name> <disc#>    - Give psi discipline to player",
		"@qstat <name>          - Quick player stat view",
		"@rdata <room#>         - Show room data",
		"@rflag                 - Remove Host Hat",
		"@rnd <#>               - Generate random number 1-#",
		"@sedate <monster>      - Sedate a monster",
		"@set <variable> <val>  - Set a variable value",
		"@snd <text>            - Echo text in current room",
		"@spawn <monster#>      - Generate monster (active)",
		"@speech <name> <verb>  - Set speech pattern (e.g. says grimly)",
		"@title <name> <title>  - Set player title (e.g. the Baroness)",
		"@unlock <item>         - Unlock item silently",
		"@whisper <name> <text> - Whisper to player anywhere",
		"@who                   - List all players with details",
		"@yank <name>           - Yank a player to your room",
		"@zap <monster>         - Destroy a monster",
		"",
		"@line1/2/3 [name] <text> - Set description lines (-none- to clear, x to reset all)",
		"@entry <text>          - Set custom room entry message",
		"@exit <text>           - Set custom room exit message",
		"@verb                  - List ALL game verbs with parameters",
	}}
}

func (e *GameEngine) gmGo(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Messages: []string{"Usage: @go <room#>"}}
	}
	num, err := strconv.Atoi(args[0])
	if err != nil {
		return &CommandResult{Messages: []string{"Invalid room number."}}
	}
	room := e.rooms[num]
	if room == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("Room %d does not exist.", num)}}
	}
	oldRoom := player.RoomNumber
	player.RoomNumber = num
	e.SavePlayer(ctx, player)
	result := e.doLook(player)
	result.Messages = append([]string{fmt.Sprintf("Teleported to room %d.", num)}, result.Messages...)
	// Broadcast exit/entry echoes (invisible GMs are completely silent)
	if !player.GMInvis {
		if player.ExitEcho != "" {
			result.OldRoomMsg = []string{player.ExitEcho}
		} else {
			result.OldRoomMsg = []string{fmt.Sprintf("%s vanishes.", player.FirstName)}
		}
		if player.EntryEcho != "" {
			result.RoomBroadcast = []string{player.EntryEcho}
		} else {
			result.RoomBroadcast = []string{fmt.Sprintf("%s appears.", player.FirstName)}
		}
	}
	result.OldRoom = oldRoom
	return result
}

func (e *GameEngine) gmAddItem(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Messages: []string{"Usage: @additem <archetype#>"}}
	}
	arch, err := strconv.Atoi(args[0])
	if err != nil {
		return &CommandResult{Messages: []string{"Invalid item number."}}
	}
	itemDef := e.items[arch]
	if itemDef == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("Item archetype %d does not exist.", arch)}}
	}
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You are nowhere."}}
	}
	ri := gameworld.RoomItem{Archetype: arch, Ref: len(room.Items)}
	room.Items = append(room.Items, ri)
	name := e.getItemNounName(itemDef)
	return &CommandResult{Messages: []string{fmt.Sprintf("Added %s (archetype %d) to the room.", name, arch)}}
}

func (e *GameEngine) gmDelete(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Messages: []string{"Usage: @delete <item name>"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You are nowhere."}}
	}
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := strings.ToLower(e.getItemNounName(itemDef))
		if strings.Contains(name, target) {
			room.Items = append(room.Items[:i], room.Items[i+1:]...)
			return &CommandResult{Messages: []string{fmt.Sprintf("Deleted %s from the room.", name)}}
		}
	}
	return &CommandResult{Messages: []string{"Item not found in this room."}}
}

func (e *GameEngine) gmRData(player *Player, args []string) *CommandResult {
	num := player.RoomNumber
	if len(args) >= 1 {
		n, err := strconv.Atoi(args[0])
		if err == nil {
			num = n
		}
	}
	room := e.rooms[num]
	if room == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("Room %d does not exist.", num)}}
	}
	msgs := []string{
		fmt.Sprintf("=== Room Data: %d ===", room.Number),
		fmt.Sprintf("Name: %s", room.Name),
		fmt.Sprintf("Terrain: %s | Lighting: %s", room.Terrain, room.Lighting),
		fmt.Sprintf("Source: %s", room.SourceFile),
	}
	if room.Description != "" {
		msgs = append(msgs, fmt.Sprintf("Desc: %s", room.Description))
	}
	msgs = append(msgs, fmt.Sprintf("Exits: %d", len(room.Exits)))
	for dir, dest := range room.Exits {
		destRoom := e.rooms[dest]
		destName := "???"
		if destRoom != nil {
			destName = destRoom.Name
		}
		msgs = append(msgs, fmt.Sprintf("  %s -> %d (%s)", dir, dest, destName))
	}
	msgs = append(msgs, fmt.Sprintf("Items: %d", len(room.Items)))
	for _, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		name := "???"
		if itemDef != nil {
			name = e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
		}
		msgs = append(msgs, fmt.Sprintf("  Ref=%d Arch=%d %s", ri.Ref, ri.Archetype, name))
	}
	if room.MonsterGroup > 0 {
		msgs = append(msgs, fmt.Sprintf("Monster Group: %d", room.MonsterGroup))
	}
	if len(room.Modifiers) > 0 {
		msgs = append(msgs, fmt.Sprintf("Modifiers: %s", strings.Join(room.Modifiers, ", ")))
	}
	msgs = append(msgs, fmt.Sprintf("Scripts: %d blocks", len(room.Scripts)))
	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) gmHeal(ctx context.Context, player *Player, args []string) *CommandResult {
	target, err := e.resolvePlayerArg(ctx, args)
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	target.BodyPoints = target.MaxBodyPoints
	target.Fatigue = target.MaxFatigue
	target.Mana = target.MaxMana
	target.Psi = target.MaxPsi
	target.Bleeding = false
	target.Stunned = false
	target.Diseased = false
	target.Poisoned = false
	target.Unconscious = false
	target.Dead = false
	e.SavePlayer(ctx, target)
	return &CommandResult{Messages: []string{fmt.Sprintf("Healed %s to full.", target.FullName())}}
}

func (e *GameEngine) gmKill(ctx context.Context, player *Player, args []string) *CommandResult {
	target, err := e.resolvePlayerArg(ctx, args)
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	target.BodyPoints = 0
	target.Dead = true
	e.SavePlayer(ctx, target)
	return &CommandResult{Messages: []string{fmt.Sprintf("%s has been slain.", target.FullName())}}
}

func (e *GameEngine) gmExp(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Usage: @exp <name> <points>"}}
	}
	target, err := e.resolvePlayerArg(ctx, args)
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	pts, err := strconv.Atoi(args[len(args)-1])
	if err != nil {
		return &CommandResult{Messages: []string{"Invalid point amount."}}
	}
	target.Experience += pts
	recalcBuildPoints(target)
	e.SavePlayer(ctx, target)
	return &CommandResult{Messages: []string{fmt.Sprintf("Granted %d experience to %s. Total: %d", pts, target.FullName(), target.Experience)}}
}

func (e *GameEngine) gmWho(ctx context.Context) *CommandResult {
	msgs := []string{"=== Online Players ==="}
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			status := ""
			if p.Dead {
				status = " DEAD"
			}
			if p.IsGM && p.GMHat {
				status += " [GM]"
			}
			msgs = append(msgs, fmt.Sprintf("  %s the %s [Lvl %d] Room %d%s",
				p.FullName(), p.RaceName(), p.Level, p.RoomNumber, status))
		}
	}
	if len(msgs) == 1 {
		msgs = append(msgs, "  No players online.")
	}
	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) gmLWho(ctx context.Context) *CommandResult {
	msgs := []string{"=== Detailed Online Player List ==="}
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			roomName := "???"
			if r := e.rooms[p.RoomNumber]; r != nil {
				roomName = r.Name
			}
			msgs = append(msgs, fmt.Sprintf("  %-20s Lvl:%-3d Room:%-5d (%s) HP:%d/%d GM:%v",
				p.FullName(), p.Level, p.RoomNumber, roomName, p.BodyPoints, p.MaxBodyPoints, p.IsGM))
		}
	}
	if len(msgs) == 1 {
		msgs = append(msgs, "  No players online.")
	}
	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) gmNum(ctx context.Context, args []string) *CommandResult {
	target, err := e.resolvePlayerArg(ctx, args)
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	roomName := "???"
	if r := e.rooms[target.RoomNumber]; r != nil {
		roomName = r.Name
	}
	return &CommandResult{Messages: []string{
		fmt.Sprintf("Player: %s", target.FullName()),
		fmt.Sprintf("Race: %s | Gender: %s | Level: %d", target.RaceName(), genderName(target.Gender), target.Level),
		fmt.Sprintf("Room: %d (%s)", target.RoomNumber, roomName),
		fmt.Sprintf("GM: %v", target.IsGM),
	}}
}

func (e *GameEngine) gmQStat(ctx context.Context, args []string) *CommandResult {
	target, err := e.resolvePlayerArg(ctx, args)
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	return &CommandResult{Messages: []string{
		fmt.Sprintf("=== Quick Stats: %s ===", target.FullName()),
		fmt.Sprintf("Race: %s | Gender: %s | Level: %d | XP: %d", target.RaceName(), genderName(target.Gender), target.Level, target.Experience),
		fmt.Sprintf("STR:%d AGI:%d QUI:%d CON:%d PER:%d WIL:%d EMP:%d",
			target.Strength, target.Agility, target.Quickness, target.Constitution,
			target.Perception, target.Willpower, target.Empathy),
		fmt.Sprintf("HP:%d/%d FT:%d/%d MP:%d/%d PSI:%d/%d",
			target.BodyPoints, target.MaxBodyPoints, target.Fatigue, target.MaxFatigue,
			target.Mana, target.MaxMana, target.Psi, target.MaxPsi),
		fmt.Sprintf("Gold:%d Silver:%d Copper:%d", target.Gold, target.Silver, target.Copper),
		fmt.Sprintf("Room: %d | GM: %v", target.RoomNumber, target.IsGM),
	}}
}

func (e *GameEngine) gmPInv(ctx context.Context, args []string) *CommandResult {
	target, err := e.resolvePlayerArg(ctx, args)
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	msgs := []string{fmt.Sprintf("=== Inventory: %s ===", target.FullName())}
	if target.Wielded != nil {
		name := e.formatInventoryItemName(target.Wielded)
		msgs = append(msgs, fmt.Sprintf("  [Wielded] %s", name))
	}
	for _, item := range target.Worn {
		name := e.formatInventoryItemName(&item)
		msgs = append(msgs, fmt.Sprintf("  [Worn: %s] %s", item.WornSlot, name))
	}
	for i, item := range target.Inventory {
		name := e.formatInventoryItemName(&item)
		msgs = append(msgs, fmt.Sprintf("  %d. %s (arch=%d)", i, name, item.Archetype))
	}
	if len(target.Inventory) == 0 && target.Wielded == nil && len(target.Worn) == 0 {
		msgs = append(msgs, "  (empty)")
	}
	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) formatInventoryItemName(item *InventoryItem) string {
	def := e.items[item.Archetype]
	if def == nil {
		return fmt.Sprintf("item#%d", item.Archetype)
	}
	return e.formatItemName(def, item.Adj1, item.Adj2, item.Adj3)
}

func (e *GameEngine) gmGenMon(player *Player, args []string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Messages: []string{"Usage: @genmon <monster#>"}}
	}
	num, err := strconv.Atoi(args[0])
	if err != nil {
		return &CommandResult{Messages: []string{"Invalid monster number."}}
	}
	mon := e.monsters[num]
	if mon == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("Monster %d does not exist.", num)}}
	}
	name := FormatMonsterName(mon, e.monAdjs)
	e.monsterMgr.SpawnOne(num, player.RoomNumber, mon.Body)
	e.monsterMgr.SetSedated(e.monsterMgr.lastSpawnedID(), true)
	e.Events.Publish("monster", fmt.Sprintf("GM %s generated %s (sedated) in room %d", player.FirstName, name, player.RoomNumber))
	return &CommandResult{Messages: []string{fmt.Sprintf("Generated %s (sedated) in room %d.", name, player.RoomNumber)}}
}

func (e *GameEngine) gmSpawn(player *Player, args []string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Messages: []string{"Usage: @spawn <monster#>"}}
	}
	num, err := strconv.Atoi(args[0])
	if err != nil {
		return &CommandResult{Messages: []string{"Invalid monster number."}}
	}
	mon := e.monsters[num]
	if mon == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("Monster %d does not exist.", num)}}
	}
	name := FormatMonsterName(mon, e.monAdjs)
	e.monsterMgr.SpawnOne(num, player.RoomNumber, mon.Body)
	e.Events.Publish("monster", fmt.Sprintf("GM %s spawned %s (active) in room %d", player.FirstName, name, player.RoomNumber))
	// Broadcast the monster's arrival to the room
	genText := mon.TextOverrides["TEXG"]
	if genText == "" {
		genText = fmt.Sprintf("A %s appears!", name)
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("Spawned %s (active) in room %d.", name, player.RoomNumber)},
		RoomBroadcast: []string{genText},
	}
}

func (e *GameEngine) gmSpeech(ctx context.Context, player *Player, args []string, rawInput string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Usage: @speech <player> <verb phrase>  (e.g., @speech Taliesin says grimly, @speech Scratch squawks)"}}
	}
	targetName := args[0]
	speechVerb := extractRawArgs(rawInput, 2) // everything after @speech <player>

	// Find target player
	var target *Player
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if strings.HasPrefix(strings.ToLower(p.FirstName), strings.ToLower(targetName)) {
				target = p
				break
			}
		}
	}
	if target == nil {
		if dbPlayer, err := e.resolvePlayerByName(ctx, targetName); err == nil {
			target = dbPlayer
		}
	}
	if target == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("Player '%s' not found.", targetName)}}
	}

	if strings.ToLower(speechVerb) == "clear" || speechVerb == "" {
		target.SpeechAdverb = ""
		e.SavePlayer(ctx, target)
		return &CommandResult{Messages: []string{fmt.Sprintf("Speech pattern cleared for %s.", target.FirstName)}}
	}

	target.SpeechAdverb = speechVerb
	e.SavePlayer(ctx, target)
	return &CommandResult{Messages: []string{fmt.Sprintf("Speech pattern for %s set to: %s %ss", target.FirstName, target.FirstName, speechVerb)}}
}

func (e *GameEngine) gmTitle(ctx context.Context, player *Player, args []string, rawInput string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Usage: @title <player> <title>  (e.g., @title Moryan the Baroness)  Use 'clear' to remove."}}
	}
	targetName := args[0]

	// Find target player (online first, then DB)
	var target *Player
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if strings.HasPrefix(strings.ToLower(p.FirstName), strings.ToLower(targetName)) {
				target = p
				break
			}
		}
	}
	if target == nil {
		if dbPlayer, err := e.resolvePlayerByName(ctx, targetName); err == nil {
			target = dbPlayer
		}
	}
	if target == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("Player '%s' not found.", targetName)}}
	}

	title := extractRawArgs(rawInput, 2) // everything after @title <player>
	if strings.ToLower(title) == "clear" || title == "" {
		target.Title = ""
		e.SavePlayer(ctx, target)
		return &CommandResult{Messages: []string{fmt.Sprintf("Title cleared for %s.", target.FirstName)}}
	}

	target.Title = title
	e.SavePlayer(ctx, target)
	return &CommandResult{Messages: []string{fmt.Sprintf("Title for %s set to: %s", target.FirstName, title)}}
}

func (e *GameEngine) gmSetLine(ctx context.Context, player *Player, args []string, rawInput string, lineNum int) *CommandResult {
	// @line1 <player#> <text> OR @line1 <text> (self)
	// "-none-" removes the line, "x" resets all lines
	if len(args) == 0 {
		return &CommandResult{Messages: []string{fmt.Sprintf("Usage: @line%d <text> (set on yourself) or @line%d <player> <text>", lineNum, lineNum)}}
	}

	target := player
	text := extractRawArgs(rawInput, 1)

	// Check if first arg is a player name (search all online players, then DB)
	if len(args) >= 2 {
		targetName := strings.ToLower(args[0])
		var found *Player
		// Search all online players (not just current room)
		if e.sessions != nil {
			for _, p := range e.sessions.OnlinePlayers() {
				if strings.HasPrefix(strings.ToLower(p.FirstName), targetName) {
					found = p
					break
				}
			}
		}
		// Fall back to DB lookup
		if found == nil {
			if dbPlayer, err := e.resolvePlayerByName(ctx, args[0]); err == nil {
				found = dbPlayer
			}
		}
		if found != nil {
			target = found
			text = extractRawArgs(rawInput, 2)
		}
	}

	if strings.ToLower(text) == "-none-" || text == "" {
		text = ""
	}
	if strings.ToLower(text) == "x" {
		target.DescLine1 = ""
		target.DescLine2 = ""
		target.DescLine3 = ""
		e.SavePlayer(ctx, target)
		return &CommandResult{Messages: []string{fmt.Sprintf("All description lines cleared for %s.", target.FirstName)}}
	}

	switch lineNum {
	case 1:
		target.DescLine1 = text
	case 2:
		target.DescLine2 = text
	case 3:
		target.DescLine3 = text
	}
	e.SavePlayer(ctx, target)

	if text == "" {
		return &CommandResult{Messages: []string{fmt.Sprintf("Description line %d cleared for %s.", lineNum, target.FirstName)}}
	}
	return &CommandResult{Messages: []string{fmt.Sprintf("Description line %d set for %s: %s", lineNum, target.FirstName, text)}}
}

func (e *GameEngine) gmSetEntryExit(ctx context.Context, player *Player, args []string, rawInput string, which string) *CommandResult {
	text := extractRawArgs(rawInput, 1)
	if text == "" {
		if which == "entry" {
			player.EntryEcho = ""
		} else {
			player.ExitEcho = ""
		}
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{fmt.Sprintf("%s echo cleared.", strings.Title(which))}}
	}

	if which == "entry" {
		player.EntryEcho = text
	} else {
		player.ExitEcho = text
	}
	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: []string{fmt.Sprintf("%s echo set: %s", strings.Title(which), text)}}
}

func (e *GameEngine) gmZap(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Usage: @zap <monster name>"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	if e.monsterMgr == nil {
		return &CommandResult{Messages: []string{"No monsters."}}
	}
	inst, def := e.findMonsterInRoom(player, target)
	if inst == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("No monster matching '%s' found in this room.", target)}}
	}
	name := FormatMonsterName(def, e.monAdjs)
	// Kill and remove from room tracking
	e.monsterMgr.mu.Lock()
	for i := range e.monsterMgr.instances {
		if e.monsterMgr.instances[i].ID == inst.ID {
			e.monsterMgr.instances[i].Alive = false
			// Remove from room index
			roomIndices := e.monsterMgr.monstersByRoom[inst.RoomNumber]
			for j, idx := range roomIndices {
				if idx == i {
					e.monsterMgr.monstersByRoom[inst.RoomNumber] = append(roomIndices[:j], roomIndices[j+1:]...)
					break
				}
			}
			break
		}
	}
	e.monsterMgr.mu.Unlock()
	return &CommandResult{Messages: []string{fmt.Sprintf("Destroyed %s.", name)}}
}

func (e *GameEngine) gmVerbs() *CommandResult {
	verbs := []string{
		"=== All Game Verbs (alphabetical) ===",
		"",
		"--- Movement ---",
		"  CLIMB <target>            - Climb a portal or climbable item",
		"  D / DOWN                  - Move down",
		"  E / EAST                  - Move east",
		"  FLY                       - Take flight (Drakin or spell)",
		"  ASCEND                    - Fly upward",
		"  DESCEND                   - Fly downward",
		"  GO <portal>               - Go through a portal, door, or stairway",
		"  LAND                      - Stop flying",
		"  N / NORTH                 - Move north",
		"  NE / NORTHEAST            - Move northeast",
		"  NW / NORTHWEST            - Move northwest",
		"  O / OUT                   - Move out",
		"  S / SOUTH                 - Move south",
		"  SE / SOUTHEAST            - Move southeast",
		"  SNEAK <direction>         - Move while hidden",
		"  SW / SOUTHWEST            - Move southwest",
		"  U / UP                    - Move up",
		"  W / WEST                  - Move west",
		"",
		"--- Combat ---",
		"  ADVANCE                   - Advance toward target",
		"  ATTACK <target>           - Attack a monster",
		"  BACKSTAB <target>         - Attack from hiding (puncture weapon required)",
		"  BERSERK                   - Berserk stance (Murg only)",
		"  BITE <target>             - Bite attack (Drakin/Wolfling/Murg)",
		"  DEFENSIVE                 - Defensive stance (+15 def, -15 att)",
		"  FLEE                      - Escape combat",
		"  GUARD <target>            - Guard another player",
		"  KILL <target>             - Attack a monster (alias for ATTACK)",
		"  MODERATE / NORMAL         - Normal combat stance",
		"  OFFENSIVE                 - Offensive stance (+15 att, -15 def)",
		"  RETREAT                   - Retreat from combat",
		"  WARY                      - Wary stance (-5 att, +5 def)",
		"",
		"--- Magic ---",
		"  CAST [target]             - Release prepared spell",
		"  PREPARE <spell>           - Prepare a spell for casting",
		"  INVOKE <spell>            - Prepare a spell (alias for PREPARE)",
		"",
		"--- Psionics ---",
		"  PROJECT [target]          - Project prepared discipline",
		"  PSI <discipline>          - Prepare a psionic discipline",
		"",
		"--- Items ---",
		"  BUY <item>                - Purchase from a shop",
		"  CLOSE <item>              - Close a door/container",
		"  DIG                       - Dig in the ground",
		"  DROP <item>               - Drop an item",
		"  EAT <item>                - Eat food",
		"  DRINK <item>              - Drink a liquid",
		"  FLIP <item>               - Flip a flippable item",
		"  GET <item>                - Pick up an item",
		"  GIVE <item> TO <player>   - Give an item or money to another player",
		"  LATCH <item>              - Latch a latchable item",
		"  LIGHT <item>              - Light a lightable item",
		"  LOAD <weapon> WITH <ammo> - Load a ranged weapon (alias: NOCK)",
		"  LOCK <item> [WITH <key>]  - Lock a lockable item",
		"  NOCK <weapon> WITH <ammo> - Load a ranged weapon",
		"  OPEN <item>               - Open a door/container",
		"  PULL <item>               - Pull an item",
		"  PUSH <item>               - Push an item",
		"  PUT <item> IN <container> - Place item in a container",
		"  READ <item>               - Read text on an item",
		"  REMOVE <item>             - Remove worn item",
		"  RUB <item>                - Rub an item",
		"  SEARCH <target>           - Search an item or dead monster",
		"  SELL <item>               - Sell an item at a merchant",
		"  SKIN <target>             - Skin a dead monster",
		"  TAP <item>                - Tap an item",
		"  TOUCH <item>              - Touch an item",
		"  TURN <item>               - Turn an item",
		"  UNDRESS                   - Remove outermost worn item",
		"  UNLATCH <item>            - Unlatch a latched item",
		"  UNLOCK <item> [WITH <key>]- Unlock a locked item",
		"  UNWIELD                   - Stop wielding weapon",
		"  WEAR <item>               - Wear armor/clothing",
		"  WIELD <item>              - Wield a weapon",
		"",
		"--- Crafting ---",
		"  ANALYZE <item>            - Analyze ore purity or reagent properties",
		"  BREW [reagent IN flask]   - Brew alchemy potion (or list recipes)",
		"  CRAFT <item>              - Craft at workshop (FORGE/LOOM/FLETCHER)",
		"  DYE <item> WITH <dye>     - Dye a material at a LOOM room",
		"  FORAGE                    - Forage materials in outdoor terrain",
		"  FORGE <item>              - Craft at a FORGE room",
		"  MINE                      - Mine ore in MINEA/B/C rooms",
		"  SMELT [ore]               - Smelt ore into metal at a FORGE room",
		"  WEAVE <item>              - Weave at a LOOM room",
		"",
		"--- Communication ---",
		"  '<message>                - Say something (shortcut for SAY)",
		"  ACT <action>              - Freeform roleplay action",
		"  CANT <message>            - Covert message (requires Legerdemain)",
		"  RECITE <text>             - Recite text (use \\ for line breaks)",
		"  REPORT <message>          - File a report (broadcast to GMs, logged)",
		"  SAY <message>             - Speak in the room",
		"  THINK <message>           - Telepathic broadcast",
		"  WHISPER <player> <message>- Whisper to a player in the room",
		"  YELL <message>            - Shout loudly",
		"",
		"--- Information ---",
		"  BALANCE                   - Check bank balance",
		"  CREDITS                   - Game credits",
		"  EXP / EXPERIENCE          - Experience and level progress",
		"  HEALTH                    - Body point summary",
		"  HELP                      - Command list",
		"  INVENTORY                 - List carried items",
		"  LOOK [target]             - Examine room, item, player, or monster",
		"  EXAMINE <target>          - Examine (alias for LOOK)",
		"  RECALL [topic]            - Recall lore about the room or an item",
		"  SKILLS                    - List trained skills",
		"  SPELL                     - List known spells",
		"  STATUS                    - Full character stats",
		"  TIME                      - In-game time and date",
		"  VERSION                   - Game version",
		"  WEALTH                    - Currency summary",
		"  WHO                       - List online players",
		"",
		"--- Skills ---",
		"  ANOINT                    - Poison your weapon (Trap & Poison Lore)",
		"  BLEND                     - Hide in mountain/cave (Highlander only)",
		"  HIDE                      - Attempt to hide in shadows",
		"  MARK [1-10]               - Set a teleport mark",
		"  REVEAL / UNHIDE           - Come out of hiding",
		"  TEND [player]             - Heal wounds (Healing skill)",
		"  TRAIN [skill]             - Train a skill (in training rooms)",
		"  UNLEARN <skill>           - Unlearn one rank of a skill",
		"",
		"--- Position ---",
		"  KNEEL                     - Kneel down",
		"  LAY                       - Lay down",
		"  SIT                       - Sit down",
		"  STAND                     - Stand up",
		"",
		"--- Social ---",
		"  SUBMIT                    - Accept intimate emotes from others",
		"  UNSUBMIT                  - Stop submitting",
		"  DEPART                    - Return from death via Eternity, Inc.",
		"  QUIT                      - Leave the game",
		"",
		"--- Settings ---",
		"  BRIEF                     - Toggle brief room descriptions",
		"  FULL                      - Toggle full room descriptions",
		"  PROMPT                    - Toggle prompt indicators",
		"  UNPROMPT                  - Turn off prompt indicators",
		"",
		"--- Emotes (150+) ---",
		"  applaud, babble, bark, bat, beam, blink, blush, bounce, bow,",
		"  caress, chuckle, clap, claw, comfort, cough, crack, cringe,",
		"  cry, cuddle, curse, curtsy, dance, dip, duck, fidget, frown,",
		"  fume, furrow, gasp, gaze, gesture, giggle, glare, grin, groan,",
		"  growl, grunt, gulp, handshake, headshake, hiss, hold, howl,",
		"  hug, hula, jig, jump, kick, kiss, knock, laugh, lean, lick,",
		"  massage, moan, mumble, nibble, nod, nudge, nuzzle, pace, pant,",
		"  peer, pet, pinch, play, point, poke, pout, punch, purr, roar,",
		"  roll, salute, scowl, scream, shrug, sing, slap, smile, smirk,",
		"  snicker, sniff, snore, snort, snuggle, spit, stare, stretch,",
		"  swoon, tap, thump, tickle, toast, twirl, wag, wait, wave,",
		"  wince, wink, write, yawn, yowl",
		"  (Plus race-specific: flick, bare, spread, fold, swish, rubears,",
		"   pullbeard, scratch, chase, scent, whine, droop)",
		"",
		"  Target emotes: <verb> <player/item/monster>",
		"  Self emotes:   <verb> me",
		"  Kiss parts:    kiss <player> <head|nose|lips|ears|neck|chest|hand|...>",
	}
	return &CommandResult{Messages: verbs}
}

func (e *GameEngine) gmLsk() *CommandResult {
	var msgs []string
	msgs = append(msgs, "=== Skills and Build Point Costs ===")
	// Build point costs: generally skill level * 2 for combat skills, varies by type
	for id := 0; id <= 35; id++ {
		name := SkillNames[id]
		if name == "" {
			continue
		}
		msgs = append(msgs, fmt.Sprintf("  %2d: %s", id, name))
	}
	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) gmMList() *CommandResult {
	if e.monsterMgr == nil {
		return &CommandResult{Messages: []string{"Monster manager not initialized."}}
	}
	e.monsterMgr.mu.RLock()
	defer e.monsterMgr.mu.RUnlock()

	var msgs []string
	msgs = append(msgs, fmt.Sprintf("=== Monster List (%d total, %d monster lists) ===", len(e.monsterMgr.instances), len(e.monsterLists)))

	// Count by status
	alive, dead, sedated := 0, 0, 0
	roomCounts := make(map[int]int)
	for _, inst := range e.monsterMgr.instances {
		if inst.Alive {
			if inst.Sedated {
				sedated++
			} else {
				alive++
			}
			roomCounts[inst.RoomNumber]++
		} else {
			dead++
		}
	}
	msgs = append(msgs, fmt.Sprintf("Alive: %d  Dead: %d  Sedated: %d  Rooms with monsters: %d", alive, dead, sedated, len(roomCounts)))

	// Show first 30 alive monsters
	count := 0
	for _, inst := range e.monsterMgr.instances {
		if !inst.Alive {
			continue
		}
		def := e.monsters[inst.DefNumber]
		if def == nil {
			continue
		}
		name := FormatMonsterName(def, e.monAdjs)
		status := "active"
		if inst.Sedated {
			status = "sedated"
		}
		target := ""
		if inst.Target != "" {
			target = fmt.Sprintf(" → attacking %s", inst.Target)
		}
		msgs = append(msgs, fmt.Sprintf("  #%d %s (def %d) room %d HP %d/%d [%s]%s",
			inst.ID, name, inst.DefNumber, inst.RoomNumber, inst.CurrentHP, def.Body+def.ExtraBody, status, target))
		count++
		if count >= 30 {
			msgs = append(msgs, fmt.Sprintf("  ... and %d more", alive-30))
			break
		}
	}

	if alive == 0 {
		msgs = append(msgs, "  No alive monsters in the world.")
		msgs = append(msgs, fmt.Sprintf("  Monster lists loaded: %d entries", len(e.monsterLists)))
		if len(e.monsterLists) > 0 {
			for i, ml := range e.monsterLists {
				if i >= 10 { break }
				def := e.monsters[ml.MonsterID]
				defName := "???"
				if def != nil { defName = def.Name }
				msgs = append(msgs, fmt.Sprintf("  MLIST: room %d, monster %d (%s), prob %d%%, max %d", ml.Room, ml.MonsterID, defName, ml.Probability, ml.MaxCount))
			}
		}
	}

	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) gmFind(args []string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Messages: []string{"Usage: @find <archetype#>"}}
	}
	arch, err := strconv.Atoi(args[0])
	if err != nil {
		return &CommandResult{Messages: []string{"Invalid archetype number."}}
	}
	itemDef := e.items[arch]
	name := fmt.Sprintf("item#%d", arch)
	if itemDef != nil {
		name = e.getItemNounName(itemDef)
	}
	msgs := []string{fmt.Sprintf("=== Finding %s (arch %d) ===", name, arch)}
	count := 0
	for _, room := range e.rooms {
		for _, ri := range room.Items {
			if ri.Archetype == arch {
				msgs = append(msgs, fmt.Sprintf("  Room %d (%s) ref=%d", room.Number, room.Name, ri.Ref))
				count++
			}
		}
	}
	msgs = append(msgs, fmt.Sprintf("Found %d instances.", count))
	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) gmList() *CommandResult {
	msgs := []string{"=== Item Types Summary ==="}
	typeCounts := make(map[string]int)
	for _, item := range e.items {
		typeCounts[item.Type]++
	}
	for t, c := range typeCounts {
		msgs = append(msgs, fmt.Sprintf("  %s: %d", t, c))
	}
	msgs = append(msgs, fmt.Sprintf("Total unique items: %d", len(e.items)))
	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) gmExamine(args []string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Messages: []string{"Usage: @examine <item#>"}}
	}
	num, err := strconv.Atoi(args[0])
	if err != nil {
		return &CommandResult{Messages: []string{"Invalid number."}}
	}
	msgs := []string{fmt.Sprintf("=== Examine #%d ===", num)}
	if itemDef := e.items[num]; itemDef != nil {
		msgs = append(msgs, fmt.Sprintf("Item: %s (type=%s, weight=%d, vol=%d)",
			e.getItemNounName(itemDef), itemDef.Type, itemDef.Weight, itemDef.Volume))
		msgs = append(msgs, fmt.Sprintf("  Article: %s | NameID: %d | Source: %s", itemDef.Article, itemDef.NameID, itemDef.SourceFile))
		if len(itemDef.Flags) > 0 {
			msgs = append(msgs, fmt.Sprintf("  Flags: %s", strings.Join(itemDef.Flags, ", ")))
		}
		msgs = append(msgs, fmt.Sprintf("  Params: P1=%d P2=%d P3=%d", itemDef.Parameter1, itemDef.Parameter2, itemDef.Parameter3))
	} else {
		msgs = append(msgs, "  No item with this number.")
	}
	if mon := e.monsters[num]; mon != nil {
		name := mon.Name
		if name == "" {
			name = fmt.Sprintf("monster#%d", num)
		}
		msgs = append(msgs, fmt.Sprintf("Monster: %s", name))
	}
	if room := e.rooms[num]; room != nil {
		msgs = append(msgs, fmt.Sprintf("Room: %s (%s)", room.Name, room.Terrain))
	}
	if noun, ok := e.nouns[num]; ok {
		msgs = append(msgs, fmt.Sprintf("Noun: %s", noun))
	}
	if adj, ok := e.adjectives[num]; ok {
		msgs = append(msgs, fmt.Sprintf("Adjective: %s", adj))
	}
	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) gmGlossary(args []string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Messages: []string{"Usage: @glossary <word>"}}
	}
	word := strings.ToLower(args[0])
	msgs := []string{fmt.Sprintf("=== Glossary: %s ===", word)}
	for id, name := range e.nouns {
		if strings.ToLower(name) == word {
			msgs = append(msgs, fmt.Sprintf("  Noun #%d: %s", id, name))
		}
	}
	for id, name := range e.adjectives {
		if strings.ToLower(name) == word {
			msgs = append(msgs, fmt.Sprintf("  Adjective #%d: %s", id, name))
		}
	}
	if len(msgs) == 1 {
		msgs = append(msgs, "  Not found.")
	}
	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) gmPeek(player *Player, args []string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Messages: []string{"Usage: @peek <variable>"}}
	}
	varName := strings.ToUpper(args[0])
	switch {
	case varName == "ROOMNUM":
		return &CommandResult{Messages: []string{fmt.Sprintf("%s = %d", varName, player.RoomNumber)}}
	case varName == "LEVEL":
		return &CommandResult{Messages: []string{fmt.Sprintf("%s = %d", varName, player.Level)}}
	case varName == "EXPERIENCE":
		return &CommandResult{Messages: []string{fmt.Sprintf("%s = %d", varName, player.Experience)}}
	case varName == "GOLD":
		return &CommandResult{Messages: []string{fmt.Sprintf("%s = %d", varName, player.Gold)}}
	case varName == "DEAD":
		val := 0
		if player.Dead {
			val = 1
		}
		return &CommandResult{Messages: []string{fmt.Sprintf("%s = %d", varName, val)}}
	case varName == "ROUNDTIME":
		return &CommandResult{Messages: []string{fmt.Sprintf("%s = %d", varName, player.RoundTime)}}
	case varName == "SPELLNUM":
		val := player.IntNums[0]
		return &CommandResult{Messages: []string{fmt.Sprintf("SPELLNUM = %d", val)}}
	case strings.HasPrefix(varName, "INTNUM"):
		numStr := strings.TrimPrefix(varName, "INTNUM")
		idx, err := strconv.Atoi(numStr)
		if err != nil {
			return &CommandResult{Messages: []string{"Invalid INTNUM index."}}
		}
		return &CommandResult{Messages: []string{fmt.Sprintf("%s = %d", varName, player.IntNums[idx])}}
	case strings.HasPrefix(varName, "PVAL"):
		idx, _ := strconv.Atoi(varName[4:])
		return &CommandResult{Messages: []string{fmt.Sprintf("%s = %d", varName, e.PVals[idx])}}
	default:
		// Check named global variables (DANWATER, etc.)
		if e.namedVarNames[varName] {
			return &CommandResult{Messages: []string{fmt.Sprintf("%s = %d", varName, e.NamedVars[varName])}}
		}
		// Try using script getVar as a fallback
		sc := &ScriptContext{Player: player, Room: e.rooms[player.RoomNumber], Engine: e}
		val := sc.getVar(varName)
		if val != 0 {
			return &CommandResult{Messages: []string{fmt.Sprintf("%s = %d", varName, val)}}
		}
		return &CommandResult{Messages: []string{fmt.Sprintf("Unknown variable: %s", varName)}}
	}
}

func (e *GameEngine) gmSet(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Usage: @set <variable> <value>"}}
	}
	varName := strings.ToUpper(args[0])
	val, err := strconv.Atoi(args[1])
	if err != nil {
		return &CommandResult{Messages: []string{"Invalid value."}}
	}
	switch {
	case varName == "LEVEL":
		player.Level = val
	case varName == "EXPERIENCE":
		player.Experience = val
	case varName == "GOLD":
		player.Gold = val
	case varName == "SILVER":
		player.Silver = val
	case varName == "COPPER":
		player.Copper = val
	case varName == "STRENGTH":
		player.Strength = val
	case varName == "AGILITY":
		player.Agility = val
	case varName == "QUICKNESS":
		player.Quickness = val
	case varName == "CONSTITUTION":
		player.Constitution = val
	case varName == "PERCEPTION":
		player.Perception = val
	case varName == "WILLPOWER":
		player.Willpower = val
	case varName == "EMPATHY":
		player.Empathy = val
	case varName == "BODYPOINTS":
		player.BodyPoints = val
	case varName == "MAXBODYPOINTS":
		player.MaxBodyPoints = val
	case varName == "FATIGUE":
		player.Fatigue = val
	case varName == "MAXFATIGUE":
		player.MaxFatigue = val
	case varName == "MANA":
		player.Mana = val
	case varName == "MAXMANA":
		player.MaxMana = val
	case varName == "PSI":
		player.Psi = val
	case varName == "MAXPSI":
		player.MaxPsi = val
	case varName == "ROUNDTIME":
		player.RoundTime = val
	case varName == "SPELLNUM":
		if player.IntNums == nil {
			player.IntNums = make(map[int]int)
		}
		player.IntNums[0] = val
	case varName == "ORG", varName == "ORGANIZATION":
		player.Organization = val
	case strings.HasPrefix(varName, "INTNUM"):
		numStr := strings.TrimPrefix(varName, "INTNUM")
		idx, err := strconv.Atoi(numStr)
		if err != nil {
			return &CommandResult{Messages: []string{"Invalid INTNUM index."}}
		}
		if player.IntNums == nil {
			player.IntNums = make(map[int]int)
		}
		player.IntNums[idx] = val
	default:
		return &CommandResult{Messages: []string{fmt.Sprintf("Unknown variable: %s", varName)}}
	}
	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: []string{fmt.Sprintf("Set %s = %d", varName, val)}}
}

func (e *GameEngine) gmRnd(args []string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Messages: []string{"Usage: @rnd <max>"}}
	}
	max, err := strconv.Atoi(args[0])
	if err != nil || max < 1 {
		return &CommandResult{Messages: []string{"Invalid number."}}
	}
	result := rand.Intn(max) + 1
	return &CommandResult{Messages: []string{fmt.Sprintf("Random (1-%d): %d", max, result)}}
}

func (e *GameEngine) gmOpenCloseLock(player *Player, args []string, state string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Messages: []string{fmt.Sprintf("Usage: @%s <item name>", strings.ToLower(state))}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You are nowhere."}}
	}
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := strings.ToLower(e.getItemNounName(itemDef))
		if strings.Contains(name, target) {
			room.Items[i].State = state
			return &CommandResult{Messages: []string{fmt.Sprintf("Set %s to %s.", name, state)}}
		}
	}
	return &CommandResult{Messages: []string{"Item not found."}}
}

func (e *GameEngine) gmGoPlr(ctx context.Context, player *Player, args []string) *CommandResult {
	target, err := e.resolvePlayerArg(ctx, args)
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	oldRoom := player.RoomNumber
	player.RoomNumber = target.RoomNumber
	e.SavePlayer(ctx, player)
	result := e.doLook(player)
	result.Messages = append([]string{fmt.Sprintf("Teleported to %s (room %d).", target.FullName(), target.RoomNumber)}, result.Messages...)
	if !player.GMInvis {
		if player.ExitEcho != "" {
			result.OldRoomMsg = []string{player.ExitEcho}
		} else {
			result.OldRoomMsg = []string{fmt.Sprintf("%s vanishes.", player.FirstName)}
		}
		if player.EntryEcho != "" {
			result.RoomBroadcast = []string{player.EntryEcho}
		} else {
			result.RoomBroadcast = []string{fmt.Sprintf("%s appears.", player.FirstName)}
		}
	}
	result.OldRoom = oldRoom
	return result
}

func (e *GameEngine) gmYank(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Messages: []string{"Usage: @yank <player name>"}}
	}
	targetName := args[0]
	// Check online players first so we update the live session pointer
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if strings.EqualFold(p.FirstName, targetName) {
				p.RoomNumber = player.RoomNumber
				e.SavePlayer(ctx, p)
				// Send yanked player visual feedback
				if e.sendToPlayer != nil {
					lookResult := e.doLook(p)
					msgs := append([]string{"You feel a strange pulling sensation...", ""}, lookResult.Messages...)
					e.sendToPlayer(p.FirstName, msgs)
				}
				return &CommandResult{Messages: []string{fmt.Sprintf("Yanked %s to room %d.", p.FullName(), player.RoomNumber)}}
			}
		}
	}
	// Fall back to DB lookup for offline players
	target, err := e.resolvePlayerArg(ctx, args)
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	target.RoomNumber = player.RoomNumber
	e.SavePlayer(ctx, target)
	return &CommandResult{Messages: []string{fmt.Sprintf("Yanked %s to room %d (offline).", target.FullName(), player.RoomNumber)}}
}

func (e *GameEngine) gmWhisper(args []string, rawInput string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Usage: @whisper <name> <text>"}}
	}
	text := extractRawArgs(rawInput, 2)
	return &CommandResult{Messages: []string{fmt.Sprintf("You whisper to %s: %s", args[0], text)}}
}

func (e *GameEngine) gmAnnounce(player *Player, args []string, rawInput string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Usage: @announce <mode> <message>"}}
	}
	mode := args[0]
	text := extractRawArgs(rawInput, 2)

	var msg string
	switch mode {
	case "0":
		// Mindlink — psionic-style broadcast
		msg = fmt.Sprintf("A mindlink announcement resonates in your mind: %s", text)
	default:
		// Mode 1 (and anything else) — global announcement
		msg = fmt.Sprintf("[Global Announcement] %s", text)
	}

	// Deliver to all online players except the sender (who gets it via CommandResult)
	if e.sessions != nil && e.sendToPlayer != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.FirstName != player.FirstName {
				e.sendToPlayer(p.FirstName, []string{msg})
			}
		}
	}

	return &CommandResult{Messages: []string{msg}}
}

func (e *GameEngine) gmBanner(player *Player, args []string, rawInput string) *CommandResult {
	if len(args) == 0 {
		// Clear banner
		e.SetBanner("")
		return &CommandResult{Messages: []string{"Login banner cleared."}}
	}
	text := extractRawArgs(rawInput, 1)
	e.SetBanner(text)

	// Broadcast notice to all online players
	notice := fmt.Sprintf("[Server Notice] %s", text)
	if e.sessions != nil && e.sendToPlayer != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.FirstName != player.FirstName {
				e.sendToPlayer(p.FirstName, []string{notice})
			}
		}
	}
	return &CommandResult{Messages: []string{notice, fmt.Sprintf("Banner set: %s", text)}}
}

func (e *GameEngine) gmEdPlayer(ctx context.Context, args []string) *CommandResult {
	target, err := e.resolvePlayerArg(ctx, args)
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	return &CommandResult{Messages: []string{
		fmt.Sprintf("=== Player Edit: %s ===", target.FullName()),
		fmt.Sprintf("Race: %d (%s) | Gender: %d (%s) | Level: %d", target.Race, target.RaceName(), target.Gender, genderName(target.Gender), target.Level),
		fmt.Sprintf("XP: %d | GM: %v", target.Experience, target.IsGM),
		fmt.Sprintf("STR:%d AGI:%d QUI:%d CON:%d PER:%d WIL:%d EMP:%d",
			target.Strength, target.Agility, target.Quickness, target.Constitution,
			target.Perception, target.Willpower, target.Empathy),
		fmt.Sprintf("HP:%d/%d FT:%d/%d MP:%d/%d PSI:%d/%d",
			target.BodyPoints, target.MaxBodyPoints, target.Fatigue, target.MaxFatigue,
			target.Mana, target.MaxMana, target.Psi, target.MaxPsi),
		fmt.Sprintf("Gold:%d Silver:%d Copper:%d", target.Gold, target.Silver, target.Copper),
		fmt.Sprintf("Room: %d | Position: %d | Dead: %v", target.RoomNumber, target.Position, target.Dead),
		fmt.Sprintf("Skills: %v", target.Skills),
		"Use @set <variable> <value> to modify.",
	}}
}

func (e *GameEngine) gmEds(ctx context.Context, args []string) *CommandResult {
	if len(args) < 3 {
		return &CommandResult{Messages: []string{"Usage: @eds <name> <skill#> <level>"}}
	}
	target, err := e.resolvePlayerArg(ctx, args)
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	skillNum, err := strconv.Atoi(args[1])
	if err != nil {
		return &CommandResult{Messages: []string{"Invalid skill number."}}
	}
	level, err := strconv.Atoi(args[2])
	if err != nil {
		return &CommandResult{Messages: []string{"Invalid level."}}
	}
	if target.Skills == nil {
		target.Skills = make(map[int]int)
	}
	target.Skills[skillNum] = level
	e.SavePlayer(ctx, target)
	return &CommandResult{Messages: []string{fmt.Sprintf("Set skill %d to level %d for %s.", skillNum, level, target.FullName())}}
}

func (e *GameEngine) gmGrantSp(ctx context.Context, args []string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Usage: @grantsp <name> <spell>"}}
	}
	target, err := e.resolvePlayerArg(ctx, args)
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	return &CommandResult{Messages: []string{fmt.Sprintf("Granted spell %s to %s.", args[1], target.FullName())}}
}

func (e *GameEngine) gmPsi(ctx context.Context, args []string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Usage: @psi <name> <discipline#>"}}
	}
	target, err := e.resolvePlayerArg(ctx, args)
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	return &CommandResult{Messages: []string{fmt.Sprintf("Granted psi discipline %s to %s.", args[1], target.FullName())}}
}

func (e *GameEngine) gmEchoPlr(args []string, rawInput string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Usage: @echoplr <name> <text>"}}
	}
	text := extractRawArgs(rawInput, 2)
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("[Echo to %s] %s", args[0], text)},
		WhisperTarget: args[0],
		WhisperMsg:    text,
	}
}

func (e *GameEngine) gmExclude(args []string, rawInput string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Usage: @exclude <name> <text>"}}
	}
	text := extractRawArgs(rawInput, 2)
	return &CommandResult{Messages: []string{fmt.Sprintf("[Room echo, excluding %s] %s", args[0], text)}}
}

func (e *GameEngine) gmGet(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Messages: []string{"Usage: @get <archetype#>"}}
	}
	arch, err := strconv.Atoi(args[0])
	if err != nil {
		return &CommandResult{Messages: []string{"Invalid record number."}}
	}
	itemDef := e.items[arch]
	if itemDef == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("Item %d does not exist.", arch)}}
	}
	invItem := InventoryItem{Archetype: arch}
	player.Inventory = append(player.Inventory, invItem)
	e.SavePlayer(ctx, player)
	name := e.getItemNounName(itemDef)
	return &CommandResult{Messages: []string{fmt.Sprintf("Added %s (arch %d) to your inventory.", name, arch)}}
}

func (e *GameEngine) gmLookContainer(player *Player, args []string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Messages: []string{"Usage: @look <archetype#>"}}
	}
	arch, err := strconv.Atoi(args[0])
	if err != nil {
		return &CommandResult{Messages: []string{"Invalid record number."}}
	}
	itemDef := e.items[arch]
	if itemDef == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("Item %d does not exist.", arch)}}
	}
	name := e.getItemNounName(itemDef)
	msgs := []string{fmt.Sprintf("=== Container: %s (arch %d) ===", name, arch)}
	msgs = append(msgs, fmt.Sprintf("Type: %s | Interior: %d", itemDef.Type, itemDef.Interior))
	if itemDef.Container != "" {
		msgs = append(msgs, fmt.Sprintf("Container: %s", itemDef.Container))
	}
	return &CommandResult{Messages: msgs}
}

// resolvePlayerArg resolves a player from the first argument (first name).
func (e *GameEngine) resolvePlayerArg(ctx context.Context, args []string) (*Player, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("usage: provide a player name")
	}
	name := strings.ToLower(args[0])
	// Prefer the live online session player (so changes are immediately visible)
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if strings.HasPrefix(strings.ToLower(p.FirstName), name) {
				return p, nil
			}
		}
	}
	// Fall back to DB lookup for offline players
	return e.resolvePlayerByName(ctx, args[0])
}

// ResolvePlayerByName looks up a player by first name (public for API layer).
func (e *GameEngine) ResolvePlayerByName(ctx context.Context, name string) (*Player, error) {
	return e.resolvePlayerByName(ctx, name)
}

// resolvePlayerByName looks up a player by first name.
func (e *GameEngine) resolvePlayerByName(ctx context.Context, name string) (*Player, error) {
	if e.db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	coll := e.db.Collection("players")
	var player Player
	// Use case-insensitive regex with escaped input to prevent regex injection
	safeName := regexp.QuoteMeta(name)
	err := coll.FindOne(ctx, bson.M{"firstName": bson.M{"$regex": "^" + safeName + "$", "$options": "i"}}).Decode(&player)
	if err != nil {
		return nil, fmt.Errorf("player '%s' not found", name)
	}
	return &player, nil
}

// SetPlayerGM sets or clears the GM flag on a player by first name. Used by admin API.
func (e *GameEngine) SetPlayerGM(ctx context.Context, firstName string, isGM bool) (*Player, error) {
	player, err := e.resolvePlayerByName(ctx, firstName)
	if err != nil {
		return nil, err
	}
	player.IsGM = isGM
	e.SavePlayer(ctx, player)
	return player, nil
}

// GetPlayer returns a player by first name. Used by admin API.
func (e *GameEngine) GetPlayer(ctx context.Context, firstName string) (*Player, error) {
	return e.resolvePlayerByName(ctx, firstName)
}

// allGMVerbs is the canonical list of all GM command verbs (with @ prefix).
var allGMVerbs = []string{
	"@HELP", "@GO", "@ADDITEM", "@DELETE", "@RDATA", "@HEAL", "@KILL", "@EXP",
	"@GM", "@RFLAG", "@HIDE", "@UNHIDE", "@INVIS", "@VIS",
	"@SND", "@ANNOUNCE", "@BANNER", "@WHO", "@LWHO", "@NUM", "@QSTAT", "@PINV",
	"@GENMON", "@SPAWN", "@ACTIVATE", "@SEDATE", "@ZAP",
	"@FIND", "@LIST", "@EXAMINE", "@GLOSSARY", "@PEEK", "@SET", "@RND",
	"@OPEN", "@CLOSE", "@LOCK", "@UNLOCK",
	"@GOPLR", "@YANK", "@WHISPER", "@EDPLAYER", "@EDPL", "@EDS", "@EDSK", "@LSK", "@GRANTSP", "@PSI", "@MLIST",
	"@ECHOPLR", "@EXCLUDE", "@SPEECH", "@TITLE", "@LINE1", "@LINE2", "@LINE3", "@VERB", "@VERBS", "@TRACE",
	"@ENTRY", "@EXIT", "@SUGGEST", "@MSG", "@SAVE", "@RESTORE", "@REGISTER",
	"@ASSIST?", "@OLDCOMP", "@EDITEM", "@EDN", "@GET", "@LOOK",
	"@QUEUE", "@UNQUEUE",
}

// resolveGMVerb resolves a GM command abbreviation to its canonical form.
func resolveGMVerb(input string) string {
	// Exact match first
	for _, v := range allGMVerbs {
		if v == input {
			return v
		}
	}
	// Prefix match — must be unique
	var match string
	for _, v := range allGMVerbs {
		if strings.HasPrefix(v, input) {
			if match != "" {
				return input // ambiguous
			}
			match = v
		}
	}
	if match != "" {
		return match
	}
	return input
}
