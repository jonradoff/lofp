package engine

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"sort"
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
	case "@GIVE":
		return e.gmGive(ctx, player, args)
	case "@TAKE":
		return e.gmTake(ctx, player, args)
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
		if player.GMInvis {
			return &CommandResult{Messages: []string{"You are already invisible."}}
		}
		player.GMInvis = true
		player.Hidden = true
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"You fade from sight."}}
	case "@VIS":
		if !player.GMInvis {
			return &CommandResult{Messages: []string{"You are already visible."}}
		}
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
	case "@TREASURE":
		return e.gmTreasure(player, args)
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
	case "@IEXAMINE", "@IEX":
    		return e.gmIExamine(ctx, player, args)
	case "@GLOSSARY":
		return e.gmGlossary(args)
	case "@PEEK":
		return e.gmPeek(player, args)
	case "@ANSWER":
		return e.gmAnswer(ctx, player)
	case "@SET":
		return e.gmSet(ctx, player, args)
	case "@SETP":
		return e.gmSetPlayer(ctx, args)
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
		return e.gmEdPlayer(ctx, player, args)
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
	case "@EDITEM", "@EDN":
		return e.gmEdItem(ctx, player, args, rawInput)
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
	case "@INITIATE":
		return e.gmInitiate(ctx, player, args)
	case "@RANK":
		return e.gmRank(ctx, player, args)
	case "@TITLE":
		return e.gmTitle(ctx, player, args, rawInput)
	case "@VERB", "@VERBS":
		return e.gmVerbs()
	case "@TRIGCEVENT", "@TCEV":
		return e.gmTrigCEvent(player, args)
	case "@MASTERY":
		return e.gmMastery(ctx, args)
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
		"@give [#] <item> to <plr> - Silently give item from your inventory to a player",
		"@glossary <word>       - Look up a noun/adj by name",
		"@gm                    - Put on Host Hat (visible as GM)",
		"@go <room#>            - Teleport to a room",
		"@goplr <name>          - Teleport to a player",
		"@grantsp <name> <sp#>  - Give spell to player",
		"@heal <name>           - Heal a player to full",
		"@help                  - This help listing",
		"@initiate [plr <org#> [remove]] - Init/remove player from org (no args = list orgs)",
		"@hide / @unhide        - Hide/show on WHO list",
		"@editem [plr] <item> <field> <val> - Edit an item in inventory",
		"@iexamine		- Examine an item in inventory",
		"@invis / @vis          - Become invisible/visible",
		"@kill <name>           - Kill a player",
		"@list                  - List all items in game",
		"@lock <item>           - Lock item silently",
		"@look <record#>        - Look inside a container",
		"@lsk                   - List all skills with IDs",
		"@mastery <plr> [sp [n]]- List/set spell mastery ranks for a player",
		"@lwho                  - Detailed player list with rooms",
		"@mlist                 - List all spawned monsters",
		"@msg                   - Toggle host messages",
		"@num <name>            - Show player info by name",
		"@open <item>           - Open item silently",
		"@peek <variable>       - View a variable value",
		"@pinv <name>           - View player inventory",
		"@psi <name> <disc#>    - Give psi discipline to player",
		"@qstat <name>          - Quick player stat view",
		"@rank <plr> <org#> <n> - Set player's org rank (must be a member)",
		"@rdata <room#>         - Show room data",
		"@rflag                 - Remove Host Hat",
		"@rnd <#>               - Generate random number 1-#",
		"@sedate <monster>      - Sedate a monster",
		"@set <variable> <val>  - Set a variable value",
		"@snd <text>            - Echo text in current room",
		"@spawn <monster#>      - Generate monster (active)",
		"@speech <name> <verb>  - Set speech pattern (e.g. says grimly)",
		"@take [#] <item> from <plr> - Silently take item from a player's inventory into yours",
		"@title <name> <title>  - Set player title (e.g. the Baroness)",
		"@treasure <level>      - Conjure a lootable chest/coffer/strongbox at the given treasure level (may be locked/trapped)",
		"@trigcevent <id>       - Immediately fire a cyclic event (for testing)",
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

// gmGive silently moves an item from the GM's inventory into a player's inventory.
// Usage: @give [#] <item name> to <player name>, e.g. "@give 2 enchanted randar
// broadsword to elara" gives Elara the second enchanted randar broadsword the GM
// is carrying.
func (e *GameEngine) gmGive(ctx context.Context, player *Player, args []string) *CommandResult {
	toIdx := -1
	for i, a := range args {
		if strings.ToUpper(a) == "TO" {
			toIdx = i
			break
		}
	}
	if toIdx <= 0 || toIdx >= len(args)-1 {
		return &CommandResult{Messages: []string{"Usage: @give <item> to <player>"}}
	}
	itemName := strings.ToLower(strings.Join(args[:toIdx], " "))
	targetName := strings.Join(args[toIdx+1:], " ")
	itemName, skip := parseOrdinal(itemName)

	target, err := e.resolvePlayerByNameLive(ctx, targetName)
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}

	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, itemName, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}
		fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3, ii.Tail)
		target.Inventory = append(target.Inventory, ii)
		player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
		e.SavePlayer(ctx, player)
		e.SavePlayer(ctx, target)
		return &CommandResult{Messages: []string{fmt.Sprintf("You silently give %s to %s.", fullName, target.FullName())}}
	}
	return &CommandResult{Messages: []string{"You don't have that."}}
}

// gmTake silently moves an item from a player's inventory into the GM's inventory.
// Usage: @take [#] <item name> from <player name>, e.g. "@take 3 babich root from
// elara" pulls the third babich root out of Elara's inventory.
func (e *GameEngine) gmTake(ctx context.Context, player *Player, args []string) *CommandResult {
	fromIdx := -1
	for i, a := range args {
		if strings.ToUpper(a) == "FROM" {
			fromIdx = i
			break
		}
	}
	if fromIdx <= 0 || fromIdx >= len(args)-1 {
		return &CommandResult{Messages: []string{"Usage: @take <item> from <player>"}}
	}
	itemName := strings.ToLower(strings.Join(args[:fromIdx], " "))
	targetName := strings.Join(args[fromIdx+1:], " ")
	itemName, skip := parseOrdinal(itemName)

	target, err := e.resolvePlayerByNameLive(ctx, targetName)
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}

	for i, ii := range target.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, itemName, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}
		fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3, ii.Tail)
		player.Inventory = append(player.Inventory, ii)
		target.Inventory = append(target.Inventory[:i], target.Inventory[i+1:]...)
		e.SavePlayer(ctx, player)
		e.SavePlayer(ctx, target)
		return &CommandResult{Messages: []string{fmt.Sprintf("You silently take %s from %s.", fullName, target.FullName())}}
	}
	return &CommandResult{Messages: []string{fmt.Sprintf("%s doesn't have that.", target.FullName())}}
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

func (e *GameEngine) gmTrigCEvent(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		var ids []string
		for _, ce := range e.cevents {
			ids = append(ids, fmt.Sprintf("%d(room %d, every %d cycles)", ce.ID, ce.Room, ce.Cycles))
		}
		if len(ids) == 0 {
			return &CommandResult{Messages: []string{"No cyclic events loaded."}}
		}
		msgs := append([]string{fmt.Sprintf("Loaded CEVENTs (%d):", len(ids))}, ids...)
		return &CommandResult{Messages: msgs}
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return &CommandResult{Messages: []string{"Usage: @trigcevent <id>"}}
	}
	for _, ce := range e.cevents {
		if ce.ID == id {
			room := e.rooms[ce.Room]
			if room == nil {
				return &CommandResult{Messages: []string{fmt.Sprintf("CEVENT %d: room %d not found.", id, ce.Room)}}
			}
			sc := &ScriptContext{Room: room, Engine: e, Player: &Player{}}
			for _, block := range ce.Scripts {
				if block.Type == "ACTION" {
					for _, action := range block.Actions {
						sc.execAction(action)
					}
				} else {
					sc.execBlock(block)
				}
			}
			if e.roomBroadcast != nil && len(sc.RoomMsgs) > 0 {
				e.roomBroadcast(ce.Room, sc.RoomMsgs)
			}
			return &CommandResult{Messages: []string{fmt.Sprintf("CEVENT %d fired in room %d (%s).", id, ce.Room, room.Name)}}
		}
	}
	return &CommandResult{Messages: []string{fmt.Sprintf("CEVENT %d not found.", id)}}
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
			name = e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3, ri.Extend)
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
	target.DiseaseLevel = 0
	target.Poisoned = false
	target.PoisonLevel = 0
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
	target.CombatTarget = nil
	target.Joined = false
	target.Position = 2 // laying down
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
	leveledUp := recalcBuildPoints(target)
	if leveledUp {
		target.MaxBodyPoints += target.Constitution / 10
		target.BodyPoints = target.MaxBodyPoints
		target.MaxFatigue += target.Constitution / 15
		target.Fatigue = target.MaxFatigue
		target.MaxMana += (target.Willpower + target.Empathy) / 15
		target.Mana = target.MaxMana
		target.MaxPsi += target.Willpower / 10
		target.Psi = target.MaxPsi
		if e.sendToPlayer != nil {
			e.sendToPlayer(target.FirstName, []string{fmt.Sprintf("Congratulations! You have advanced to level %d!", target.Level)})
		}
		if e.roomBroadcast != nil {
			e.roomBroadcast(target.RoomNumber, []string{fmt.Sprintf("%s has advanced to level %d!", target.FirstName, target.Level)})
		}
	}
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
	return e.formatItemName(def, item.Adj1, item.Adj2, item.Adj3, item.Tail)
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

// gmTreasure conjures a lootable chest/coffer/strongbox in the GM's room at the
// given treasure level, using the same generator monster kills use (randomChestDrop
// in treasure.go, populateContainerLoot in containers.go) — so the container may
// come out locked and/or trapped just like a real drop.
func (e *GameEngine) gmTreasure(player *Player, args []string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Messages: []string{"Usage: @treasure <level>  (1-100+; drives lock difficulty, trap chance, and loot quality)"}}
	}
	level, err := strconv.Atoi(args[0])
	if err != nil || level < 1 {
		return &CommandResult{Messages: []string{"Invalid treasure level. Usage: @treasure <level>"}}
	}
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You are nowhere."}}
	}

	item := e.randomChestDrop(level)
	if item == nil {
		return &CommandResult{Messages: []string{"No suitable container item found in the item table."}}
	}
	item.Ref = nextRoomItemRef(room)
	room.Items = append(room.Items, *item)
	e.populateContainerLoot(player.RoomNumber, item.Ref, level)

	def := e.items[item.Archetype]
	plainName := e.formatItemName(def, item.Adj1, item.Adj2, item.Adj3, item.Extend)
	gmName := e.formatContainerName(def, item.Adj1, item.Adj2, item.Adj3, item.State, item.Extend)
	trapNote := ""
	if item.Val4 != 0 {
		trapNote = " (trapped)"
	}

	e.Events.Publish("gm", fmt.Sprintf("GM %s conjured %s at treasure level %d in room %d", player.FirstName, plainName, level, player.RoomNumber))

	return &CommandResult{
		Messages:      []string{fmt.Sprintf("Conjured %s%s at treasure level %d (ref %d).", gmName, trapNote, level, item.Ref)},
		RoomBroadcast: []string{fmt.Sprintf("%s appears out of thin air!", capitalize(plainName))},
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
		"  UNWIELD [item]            - Stop wielding weapon/shield (or all if no argument)",
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

// gmIExamine shows full internal details of an item in a player's inventory/wielded/worn slots
func (e *GameEngine) gmIExamine(ctx context.Context, gmPlayer *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Usage: @iexamine [playername] <item>" +
			"\n   (if no player is given, examines your own inventory)"}}
	}

	// Resolve target player (first arg = player, or self)
	target := gmPlayer
	itemTarget := strings.ToLower(strings.Join(args, " "))

if len(args) >= 2 {
    if resolved, err := e.resolvePlayerArg(ctx, []string{args[0]}); err == nil {
        target = resolved
        itemTarget = strings.ToLower(strings.Join(args[1:], " "))
    }
    // else: do nothing → treat all args as item
}


	msgs := []string{
		fmt.Sprintf("=== Inventory Examine: %s ===", target.FullName()),
		fmt.Sprintf("Room %d | Total items: %d (inv + wielded + worn)", target.RoomNumber, len(target.Inventory)+len(target.Worn)+1),
	}

	found := false

	// Wielded
	if target.Wielded != nil && e.gmItemMatchesTarget(target.Wielded, itemTarget) {
		msgs = append(msgs, e.formatFullItemDebug(target.Wielded, "WIELDING"))
		found = true
	}

	// Worn items
	for i := range target.Worn {
		item := &target.Worn[i]
		if e.gmItemMatchesTarget(item, itemTarget) {
			msgs = append(msgs, e.formatFullItemDebug(item, fmt.Sprintf("WORN (%s)", item.WornSlot)))
			found = true
		}
	}

	// Inventory (and one level into any open container's contents — e.g. a
	// potion vial sitting inside an open bag)
	for i := range target.Inventory {
		item := &target.Inventory[i]
		if e.gmItemMatchesTarget(item, itemTarget) {
			msgs = append(msgs, e.formatFullItemDebug(item, fmt.Sprintf("INV #%d", i)))
			found = true
		}
		def := e.items[item.Archetype]
		if def != nil && isContainerDef(def) && containerIsOpen(def, item.State) {
			for j := range item.Contents {
				ci := &item.Contents[j]
				if e.gmItemMatchesTarget(ci, itemTarget) {
					msgs = append(msgs, e.formatFullItemDebug(ci, fmt.Sprintf("INV #%d > CONTENTS #%d", i, j)))
					found = true
				}
			}
		}
	}

	if !found {
		return &CommandResult{Messages: []string{"No matching item found in that player's inventory."}}
	}

	return &CommandResult{Messages: msgs}
}

// gmItemMatchesTarget reports whether item matches itemTarget for GM item-lookup
// commands (@iexamine, @editem) — either by substring of its formatted name, or,
// for a LIQCONTAINER holding a potion, by the potion's liquid-appearance phrase
// (e.g. "crimson potion"), regardless of which vessel currently holds it.
func (e *GameEngine) gmItemMatchesTarget(item *InventoryItem, itemTarget string) bool {
	if itemTarget == "" {
		return true
	}
	name := strings.ToLower(e.formatInventoryItemName(item))
	if strings.Contains(name, itemTarget) {
		return true
	}
	def := e.items[item.Archetype]
	if phrase := e.potionPhraseIfAny(def, item.Val2, item.Val4); phrase != "" {
		return strings.Contains(strings.ToLower(phrase), itemTarget)
	}
	return false
}

func (e *GameEngine) gmGlossary(args []string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Messages: []string{"Usage: @glossary <word>"}}
	}
	word := strings.ToLower(args[0])
	msgs := []string{fmt.Sprintf("=== Glossary: %s ===", word)}
	var matchedNounIDs []int
	for id, name := range e.nouns {
		if strings.ToLower(name) == word {
			msgs = append(msgs, fmt.Sprintf("  Noun #%d: %s", id, name))
			matchedNounIDs = append(matchedNounIDs, id)
		}
	}
	for id, name := range e.adjectives {
		if strings.ToLower(name) == word {
			msgs = append(msgs, fmt.Sprintf("  Adjective #%d: %s", id, name))
		}
	}
	// Show item archetypes that use this noun so @additem can be used directly.
	for _, nounID := range matchedNounIDs {
		var archs []string
		for num, def := range e.items {
			if def.NameID == nounID {
				archs = append(archs, fmt.Sprintf("#%d (%s)", num, def.Type))
			}
		}
		if len(archs) > 0 {
			sort.Strings(archs)
			msgs = append(msgs, fmt.Sprintf("  Item archetypes: %s", strings.Join(archs, ", ")))
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
	// If @edpl set an edit target, redirect @set to that player
	if player.GMEditTarget != "" {
		target, err := e.resolvePlayerArg(ctx, []string{player.GMEditTarget})
		if err != nil {
			return &CommandResult{Messages: []string{fmt.Sprintf("Edit target %s not found: %v", player.GMEditTarget, err)}}
		}
		result := e.gmSetOnPlayer(ctx, target, args)
		if len(result.Messages) > 0 {
			result.Messages[0] = fmt.Sprintf("[%s] %s", target.FullName(), result.Messages[0])
		}
		return result
	}
	return e.gmSetOnPlayer(ctx, player, args)
}

func (e *GameEngine) gmSetOnPlayer(ctx context.Context, player *Player, args []string) *CommandResult {
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
	case varName == "BP" || varName == "BODYPOINTS":
		player.BodyPoints = val
		if varName == "BP" {
			player.MaxBodyPoints = val // shorthand sets both
		}
	case varName == "MAXBODYPOINTS" || varName == "MAXBP":
		player.MaxBodyPoints = val
	case varName == "FATIGUE" || varName == "FAT":
		player.Fatigue = val
		player.MaxFatigue = val // always set both
	case varName == "MAXFATIGUE":
		player.MaxFatigue = val
	case varName == "MANA":
		player.Mana = val
		player.MaxMana = val
	case varName == "MAXMANA":
		player.MaxMana = val
	case varName == "PSI":
		player.Psi = val
		player.MaxPsi = val
	case varName == "MAXPSI":
		player.MaxPsi = val
	case varName == "GENDER":
		player.Gender = val
	case varName == "ROUNDTIME":
		player.RoundTime = val
	case varName == "SPELLNUM":
		if player.IntNums == nil {
			player.IntNums = make(map[int]int)
		}
		player.IntNums[0] = val
	case varName == "ORG", varName == "ORGANIZATION":
		if val == 0 {
			player.RemoveOrg(player.Organization)
		} else {
			player.AddOrg(val, 1)
		}
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

// gmSetPlayer handles @setp <player> <variable> <value> — set a variable on another player.
func (e *GameEngine) gmSetPlayer(ctx context.Context, args []string) *CommandResult {
	if len(args) < 3 {
		return &CommandResult{Messages: []string{"Usage: @setp <player> <variable> <value>"}}
	}
	target, err := e.resolvePlayerArg(ctx, args[:1])
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	// Reuse gmSet logic with the target player
	result := e.gmSet(ctx, target, args[1:])
	// Prefix the message with target name
	if len(result.Messages) > 0 {
		result.Messages[0] = fmt.Sprintf("[%s] %s", target.FullName(), result.Messages[0])
	}
	return result
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
	target, skip := parseOrdinal(target)
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You are nowhere."}}
	}
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
		room.Items[i].State = state
		return &CommandResult{Messages: []string{fmt.Sprintf("Set %s to %s.", name, state)}}
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

func (e *GameEngine) gmAnswer(ctx context.Context, player *Player) *CommandResult {
	if e.lastAssistName == "" {
		return &CommandResult{Messages: []string{"No pending assist requests."}}
	}
	oldRoom := player.RoomNumber
	player.RoomNumber = e.lastAssistRoom
	e.SavePlayer(ctx, player)
	result := e.doLook(player)
	result.Messages = append([]string{fmt.Sprintf("Answering %s's assist request. Teleported to room %d.", e.lastAssistName, e.lastAssistRoom)}, result.Messages...)
	if !player.GMInvis {
		result.OldRoomMsg = []string{fmt.Sprintf("%s vanishes.", player.FirstName)}
		result.RoomBroadcast = []string{fmt.Sprintf("%s appears.", player.FirstName)}
	}
	result.OldRoom = oldRoom
	e.lastAssistName = ""
	e.lastAssistRoom = 0
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
	return &CommandResult{Messages: []string{fmt.Sprintf("Yanked %s to room %d. (Player was offline — will see change on next login.)", target.FullName(), player.RoomNumber)}}
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

func (e *GameEngine) gmEdPlayer(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		// Clear edit target
		player.GMEditTarget = ""
		return &CommandResult{Messages: []string{"Edit target cleared. @set will now modify your own character."}}
	}
	target, err := e.resolvePlayerArg(ctx, args)
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	player.GMEditTarget = target.FirstName
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
		fmt.Sprintf("@set will now modify %s. Use @edpl (no args) to clear.", target.FirstName),
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

func (e *GameEngine) gmMastery(ctx context.Context, args []string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Messages: []string{"Usage: @mastery <player> [<spell# or name> <level>]"}}
	}
	target, err := e.resolvePlayerArg(ctx, args)
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}

	// List mode: @mastery <player>
	if len(args) < 3 {
		if len(target.SpellMastery) == 0 {
			return &CommandResult{Messages: []string{fmt.Sprintf("%s has no spell mastery.", target.FullName())}}
		}
		type mastEntry struct {
			id      int
			name    string
			rank    int
			maxRank int
		}
		var entries []mastEntry
		for spellID, rank := range target.SpellMastery {
			if rank <= 0 {
				continue
			}
			spell := FindSpellByID(spellID)
			if spell == nil {
				entries = append(entries, mastEntry{id: spellID, name: fmt.Sprintf("[unknown #%d]", spellID), rank: rank, maxRank: 0})
			} else {
				entries = append(entries, mastEntry{id: spellID, name: spell.Name, rank: rank, maxRank: spell.Level + 1})
			}
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].id < entries[j].id })
		msgs := []string{fmt.Sprintf("Spell mastery for %s:", target.FullName())}
		for _, me := range entries {
			msgs = append(msgs, fmt.Sprintf("  #%-4d %-35s rank %d / %d", me.id, me.name, me.rank, me.maxRank))
		}
		return &CommandResult{Messages: msgs}
	}

	// Set mode: @mastery <player> <spell# or name> <level>
	var spell *SpellDef
	if id, convErr := strconv.Atoi(args[1]); convErr == nil {
		spell = FindSpellByID(id)
	} else {
		spell = FindSpellByName(args[1])
	}
	if spell == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("Unknown spell: %s", args[1])}}
	}
	level, err := strconv.Atoi(args[2])
	if err != nil || level < 0 {
		return &CommandResult{Messages: []string{"Level must be a non-negative integer (use 0 to clear mastery)."}}
	}
	maxRank := spell.Level + 1
	if level > maxRank {
		return &CommandResult{Messages: []string{fmt.Sprintf("Max mastery for %s (level %d spell) is %d.", spell.Name, spell.Level, maxRank)}}
	}
	if target.SpellMastery == nil {
		target.SpellMastery = make(map[int]int)
	}
	oldRank := target.SpellMastery[spell.ID]

	// Calculate BP refund when reducing mastery.
	// Rank 1 cost 8 BP; ranks 2+ cost 4 BP each.
	var bpRefund int
	if level < oldRank {
		ranksRemoved := oldRank - level
		if level == 0 {
			bpRefund = 8 + (ranksRemoved-1)*4
		} else {
			bpRefund = ranksRemoved * 4
		}
		target.BuildPoints += bpRefund
	}

	if level == 0 {
		delete(target.SpellMastery, spell.ID)
	} else {
		target.SpellMastery[spell.ID] = level
	}
	e.SavePlayer(ctx, target)

	msg := fmt.Sprintf("Set %s mastery for %s to rank %d (max %d).", spell.Name, target.FullName(), level, maxRank)
	if bpRefund > 0 {
		msg += fmt.Sprintf(" Refunded %d BP (%d BP total).", bpRefund, target.BuildPoints)
	}
	return &CommandResult{Messages: []string{msg}}
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

// gmInitiate handles @initiate.
//
//	@initiate                          — list all organizations with their numbers
//	@initiate <player> <org#>          — add player to org (rank 1 if new member)
//	@initiate <player> <org#> remove   — remove player from org
func (e *GameEngine) gmInitiate(ctx context.Context, _ *Player, args []string) *CommandResult {
	if len(args) == 0 {
		msgs := []string{
			"=== Organizations ===",
			"Usage: @initiate <player> <org#> [remove]",
			"",
		}
		type orgEntry struct {
			num      int
			name     string
			joinType string
		}
		seen := map[int]bool{}
		var entries []orgEntry
		for _, def := range e.orgDefs {
			entries = append(entries, orgEntry{def.Number, def.Name, def.JoinType})
			seen[def.Number] = true
		}
		for num, name := range knownOrgNames {
			if !seen[num] {
				entries = append(entries, orgEntry{num, name, ""})
			}
		}
		for i := 0; i < len(entries)-1; i++ {
			for j := i + 1; j < len(entries); j++ {
				if entries[j].num < entries[i].num {
					entries[i], entries[j] = entries[j], entries[i]
				}
			}
		}
		for _, oe := range entries {
			if oe.joinType != "" {
				msgs = append(msgs, fmt.Sprintf("  %2d  %-30s [%s]", oe.num, oe.name, oe.joinType))
			} else {
				msgs = append(msgs, fmt.Sprintf("  %2d  %-30s", oe.num, oe.name))
			}
		}
		return &CommandResult{Messages: msgs}
	}
	// Single arg that isn't a number: show the player's org memberships.
	if len(args) == 1 {
		if _, err := strconv.Atoi(args[0]); err != nil {
			target, err2 := e.resolvePlayerArg(ctx, args[:1])
			if err2 != nil {
				return &CommandResult{Messages: []string{err2.Error()}}
			}
			orgs := target.OrgList()
			if len(orgs) == 0 {
				return &CommandResult{Messages: []string{fmt.Sprintf("%s is not a member of any organization.", target.FullName())}}
			}
			msgs := []string{fmt.Sprintf("=== Organizations for %s ===", target.FullName()), ""}
			for _, orgNum := range orgs {
				name := organizationName(orgNum)
				if def, ok := e.orgDefs[orgNum]; ok {
					name = def.Name
				}
				msgs = append(msgs, fmt.Sprintf("  %2d  %-30s  rank %d", orgNum, name, target.RankIn(orgNum)))
			}
			return &CommandResult{Messages: msgs}
		}
	}
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Usage: @initiate <player> <org#> [remove]  (@initiate alone lists orgs)"}}
	}
	target, err := e.resolvePlayerArg(ctx, args[:1])
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	orgNum, err2 := strconv.Atoi(args[1])
	if err2 != nil || orgNum <= 0 {
		return &CommandResult{Messages: []string{"Organization must be a positive number."}}
	}
	name := organizationName(orgNum)
	if def, ok := e.orgDefs[orgNum]; ok {
		name = def.Name
	}
	// Optional third arg: "remove"
	if len(args) >= 3 && strings.EqualFold(args[2], "remove") {
		if !target.IsMemberOf(orgNum) {
			return &CommandResult{Messages: []string{fmt.Sprintf("%s is not a member of the %s.", target.FullName(), name)}}
		}
		target.RemoveOrg(orgNum)
		e.SavePlayer(ctx, target)
		return &CommandResult{Messages: []string{fmt.Sprintf("%s has been removed from the %s.", target.FullName(), name)}}
	}
	// Add to org (rank 1 if not already a member)
	rank := target.RankIn(orgNum)
	if rank == 0 {
		rank = 1
	}
	target.AddOrg(orgNum, rank)
	e.SavePlayer(ctx, target)
	return &CommandResult{
		Messages: []string{fmt.Sprintf("%s (org %d, rank %d) has been initiated into the %s.", target.FullName(), orgNum, rank, name)},
	}
}

// gmRank handles @rank — GM sets a player's rank in an organization.
// The player must already be a member. @rank <player> <org#> <rank>
func (e *GameEngine) gmRank(ctx context.Context, _ *Player, args []string) *CommandResult {
	if len(args) < 3 {
		return &CommandResult{Messages: []string{"Usage: @rank <player> <org#> <rank>"}}
	}
	target, err := e.resolvePlayerArg(ctx, args[:1])
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	orgNum, err2 := strconv.Atoi(args[1])
	if err2 != nil || orgNum <= 0 {
		return &CommandResult{Messages: []string{"Organization must be a positive number."}}
	}
	rank, err3 := strconv.Atoi(args[2])
	if err3 != nil || rank < 0 {
		return &CommandResult{Messages: []string{"Rank must be a non-negative number."}}
	}
	name := organizationName(orgNum)
	if def, ok := e.orgDefs[orgNum]; ok {
		name = def.Name
	}
	if !target.IsMemberOf(orgNum) {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s is not a member of the %s (org %d).", target.FullName(), name, orgNum)}}
	}
	target.AddOrg(orgNum, rank)
	e.SavePlayer(ctx, target)
	return &CommandResult{
		Messages: []string{fmt.Sprintf("%s rank in the %s set to %d.", target.FullName(), name, rank)},
	}
}

// resolvePlayerArg resolves a player from the first argument (first name).
func (e *GameEngine) resolvePlayerArg(ctx context.Context, args []string) (*Player, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("usage: provide a player name")
	}
	return e.resolvePlayerByNameLive(ctx, args[0])
}

// resolvePlayerByNameLive resolves a player by name, preferring the live online
// session player (so changes are immediately visible) and falling back to a DB
// lookup for offline players.
func (e *GameEngine) resolvePlayerByNameLive(ctx context.Context, rawName string) (*Player, error) {
	name := strings.ToLower(strings.TrimSpace(rawName))
	if name == "" {
		return nil, fmt.Errorf("usage: provide a player name")
	}
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if strings.HasPrefix(strings.ToLower(p.FirstName), name) {
				return p, nil
			}
		}
	}
	return e.resolvePlayerByName(ctx, name)
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
	"@HELP", "@GO", "@ADDITEM", "@GIVE", "@TAKE", "@DELETE", "@RDATA", "@HEAL", "@KILL", "@EXP",
	"@GM", "@RFLAG", "@HIDE", "@UNHIDE", "@INVIS", "@VIS",
	"@SND", "@ANNOUNCE", "@BANNER", "@WHO", "@LWHO", "@NUM", "@QSTAT", "@PINV",
	"@GENMON", "@SPAWN", "@ACTIVATE", "@SEDATE", "@ZAP", "@TREASURE",
	"@FIND", "@LIST", "@EXAMINE", "@GLOSSARY", "@PEEK", "@SET", "@RND",
	"@OPEN", "@CLOSE", "@LOCK", "@UNLOCK",
	"@GOPLR", "@YANK", "@WHISPER", "@EDPLAYER", "@EDPL", "@EDS", "@EDSK", "@LSK", "@GRANTSP", "@PSI", "@MLIST",
	"@ECHOPLR", "@EXCLUDE", "@SPEECH", "@TITLE", "@LINE1", "@LINE2", "@LINE3", "@VERB", "@VERBS", "@TRACE",
	"@ENTRY", "@EXIT", "@SUGGEST", "@MSG", "@SAVE", "@RESTORE", "@REGISTER",
	"@ASSIST?", "@OLDCOMP", "@EDITEM", "@EDN", "@GET", "@LOOK",
	"@QUEUE", "@UNQUEUE",
	"@MASTERY",
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

// formatFullItemDebug returns a multi-line debug string for any InventoryItem
func (e *GameEngine) formatFullItemDebug(item *InventoryItem, location string) string {
	def := e.items[item.Archetype]
	baseName := "???"
	if def != nil {
		baseName = e.getItemNounName(def)
	}

	// Resolve adjective names
	adj1 := "0"
	if item.Adj1 > 0 {
		adj1 = fmt.Sprintf("%d (%s)", item.Adj1, e.getAdjName(item.Adj1))
	}
	adj2 := "0"
	if item.Adj2 > 0 {
		adj2 = fmt.Sprintf("%d (%s)", item.Adj2, e.getAdjName(item.Adj2))
	}
	adj3 := "0"
	if item.Adj3 > 0 {
		adj3 = fmt.Sprintf("%d (%s)", item.Adj3, e.getAdjName(item.Adj3))
	}

	state := ""
	if item.State != "" {
		state = fmt.Sprintf(" | State=%s", item.State)
	}
	tail := ""
	if item.Tail != "" {
		tail = fmt.Sprintf("\n  Tail=%q", item.Tail)
	}

	examineDesc := ""
	if def != nil && def.ExamineDesc != "" {
		examineDesc = fmt.Sprintf("\n  ExamineDesc=%q", def.ExamineDesc)
	}
	hardness := ""
	if def != nil && isWeaponItemType(def.Type) {
		hardness = fmt.Sprintf("\n  Hardness=%d (Weapon Clash break-resistance)", e.weaponHardness(item, def))
	}
	return fmt.Sprintf("%s: %s (arch=%d)\n"+
		"  Adj1=%s | Adj2=%s | Adj3=%s\n"+
		"  Val1=%d Val2=%d Val3=%d Val4=%d Val5=%d%s%s%s%s",
		location, baseName, item.Archetype,
		adj1, adj2, adj3,
		item.Val1, item.Val2, item.Val3, item.Val4, item.Val5,
		state, tail, examineDesc, hardness)
}

// gmEdItem implements @editem / @edn.
//
// Syntax (all tokens after the command verb):
//
//   @editem <item> <field> <value>
//       — edit an item in the GM's own inventory / wielded / worn
//
//   @editem <playername> <item> <field> <value>
//       — edit an item in another player's inventory / wielded / worn
//
// <item>  : partial name match (same as @iexamine)
// <field> : adj1 adj2 adj3 val1 val2 val3 val4 val5 state
//           archetype  (dangerous but allowed)
//           flag+<FLAG>  flag-<FLAG>   (add / remove a flag on the archetype def)
// <value> : integer for numeric fields, string for state / flags
//
// Examples:
//   @editem robe val1 1
//   @editem robe adj1 47
//   @editem robe state OPEN
//   @editem robe flag+DYEABLE
//   @editem robe flag-DYEABLE
//   @editem Moryan robe val1 1
func (e *GameEngine) gmEdItem(ctx context.Context, gmPlayer *Player, args []string, rawInput string) *CommandResult {

	// --- usage guard ---
	const usage = "Usage: @editem [player] <item> <field> <value>\n" +
		"  Fields: adj1 adj2 adj3  val1-val5  state  tail  archetype  flag+FLAG / flag-FLAG\n" +
		"  Example: @editem robe val1 1\n" +
		"  Example: @editem Moryan robe adj2 47\n" +
		"  Example: @editem gloves tail lined with palest pink silk\n" +
		"  Example: @editem gloves tail -  (clears the tail)"

	if len(args) < 3 {
		return &CommandResult{Messages: []string{usage}}
	}

	// --- resolve optional leading player name (mirrors @iexamine) ---
	// Default to the GM themselves. If args[0] resolves as a player name,
	// treat it as the target and shift remaining args left by one.
	target := gmPlayer
	remaining := args // [item] [field] [value]

	if len(args) >= 2 {
		if resolved, err := e.resolvePlayerArg(ctx, []string{args[0]}); err == nil {
			target = resolved
			remaining = args[1:]
		}
		// else: resolution failed — treat all args as (item field value) on self
	}

	if len(remaining) < 3 {
		return &CommandResult{Messages: []string{usage}}
	}

	// Field and value are always the LAST two tokens.
	// Everything before them is the (potentially multi-word) item name,
	// identical to how @iexamine joins all its args as the item name.
	field := strings.ToLower(remaining[len(remaining)-2])
	valueStr := remaining[len(remaining)-1]
	itemTarget := strings.ToLower(strings.Join(remaining[:len(remaining)-2], " "))

	// Special handling for "tail": supports multi-word values.
	// Scan left-to-right for the keyword "tail"; everything after it is the value.
	// Because args are uppercased by the command parser, we extract the value from
	// rawInput to preserve original case.
	// @editem gloves tail lined with palest pink silk → item="gloves", value="lined with palest pink silk"
	// @editem gloves tail -                           → item="gloves", value="" (clears tail)
	for i := 1; i < len(remaining); i++ {
		if strings.ToLower(remaining[i]) == "tail" {
			itemTarget = strings.ToLower(strings.Join(remaining[:i], " "))
			field = "tail"
			// Extract original-case value from rawInput (args are uppercased by the parser)
			lowerRaw := strings.ToLower(rawInput)
			if sepIdx := strings.Index(lowerRaw, " tail "); sepIdx >= 0 {
				valueStr = strings.TrimSpace(rawInput[sepIdx+6:])
			} else if strings.HasSuffix(strings.TrimSpace(lowerRaw), " tail") {
				valueStr = ""
			}
			break
		}
	}

	// --- locate the item in the target's slots ---
	type itemRef struct {
		item   *InventoryItem
		label  string
		inWorn bool   // true → lives in Worn slice (needs index)
		inInv  bool   // true → lives in Inventory slice
		wornIdx int
		invIdx  int
	}

	var found []itemRef

	if target.Wielded != nil && e.gmItemMatchesTarget(target.Wielded, itemTarget) {
		found = append(found, itemRef{item: target.Wielded, label: "WIELDING"})
	}
	for i := range target.Worn {
		if e.gmItemMatchesTarget(&target.Worn[i], itemTarget) {
			found = append(found, itemRef{
				item:    &target.Worn[i],
				label:   fmt.Sprintf("WORN (%s)", target.Worn[i].WornSlot),
				inWorn:  true,
				wornIdx: i,
			})
		}
	}
	for i := range target.Inventory {
		invItem := &target.Inventory[i]
		if e.gmItemMatchesTarget(invItem, itemTarget) {
			found = append(found, itemRef{
				item:   invItem,
				label:  fmt.Sprintf("INV #%d", i),
				inInv:  true,
				invIdx: i,
			})
		}
		// One level into any open container's contents (e.g. a potion vial
		// sitting inside an open bag) — mutated directly via pointer, so no
		// writeback bookkeeping (inWorn/inInv) is needed for these matches.
		def := e.items[invItem.Archetype]
		if def != nil && isContainerDef(def) && containerIsOpen(def, invItem.State) {
			for j := range invItem.Contents {
				ci := &invItem.Contents[j]
				if e.gmItemMatchesTarget(ci, itemTarget) {
					found = append(found, itemRef{
						item:  ci,
						label: fmt.Sprintf("INV #%d > CONTENTS #%d", i, j),
					})
				}
			}
		}
	}

	switch len(found) {
	case 0:
		return &CommandResult{Messages: []string{
			fmt.Sprintf("No item matching '%s' found in %s's inventory.", itemTarget, target.FirstName),
		}}
	case 1:
		// exactly one match — proceed
	default:
		msgs := []string{fmt.Sprintf("Multiple items match '%s' — be more specific:", itemTarget)}
		for _, f := range found {
			msgs = append(msgs, fmt.Sprintf("  %s: %s (arch=%d)", f.label,
				e.formatInventoryItemName(f.item), f.item.Archetype))
		}
		return &CommandResult{Messages: msgs}
	}

	ref := found[0]
	item := ref.item

	// --- apply the edit ---
	var changeDesc string

	switch {
	// ---- integer fields ----
	case field == "adj1", field == "adj2", field == "adj3",
		field == "val1", field == "val2", field == "val3", field == "val4", field == "val5",
		field == "archetype":

		v, err := strconv.Atoi(valueStr)
		if err != nil {
			return &CommandResult{Messages: []string{fmt.Sprintf("'%s' is not a valid integer.", valueStr)}}
		}
		old := 0
		switch field {
		case "adj1":
			old, item.Adj1 = item.Adj1, v
		case "adj2":
			old, item.Adj2 = item.Adj2, v
		case "adj3":
			old, item.Adj3 = item.Adj3, v
		case "val1":
			old, item.Val1 = item.Val1, v
		case "val2":
			old, item.Val2 = item.Val2, v
		case "val3":
			old, item.Val3 = item.Val3, v
		case "val4":
			old, item.Val4 = item.Val4, v
		case "val5":
			old, item.Val5 = item.Val5, v
		case "archetype":
			old, item.Archetype = item.Archetype, v
		}
		changeDesc = fmt.Sprintf("Set %s: %s %d → %d", ref.label, strings.ToUpper(field), old, v)

	// ---- state (string) ----
	case field == "state":
		old := item.State
		item.State = strings.ToUpper(valueStr)
		changeDesc = fmt.Sprintf("Set %s: STATE %s → %s", ref.label, old, item.State)

	// ---- tail (string, multi-word, "-" clears) ----
	case field == "tail":
		old := item.Tail
		if valueStr == "-" {
			item.Tail = ""
		} else {
			item.Tail = valueStr
		}
		changeDesc = fmt.Sprintf("Set %s: TAIL %q → %q", ref.label, old, item.Tail)

	// ---- flag+FLAG / flag-FLAG — modifies the archetype definition ----
	case strings.HasPrefix(field, "flag+"), strings.HasPrefix(field, "flag-"):
		add := strings.HasPrefix(field, "flag+")
		flagName := strings.ToUpper(valueStr)
		def := e.items[item.Archetype]
		if def == nil {
			return &CommandResult{Messages: []string{fmt.Sprintf("No item definition for archetype %d.", item.Archetype)}}
		}
		if add {
			// Add flag if not already present
			already := false
			for _, f := range def.Flags {
				if strings.EqualFold(f, flagName) {
					already = true
					break
				}
			}
			if already {
				return &CommandResult{Messages: []string{fmt.Sprintf("Flag %s already set on arch %d.", flagName, item.Archetype)}}
			}
			def.Flags = append(def.Flags, flagName)
			changeDesc = fmt.Sprintf("Added flag %s to arch %d (%s)", flagName, item.Archetype, e.getItemNounName(def))
		} else {
			// Remove flag
			newFlags := def.Flags[:0]
			removed := false
			for _, f := range def.Flags {
				if strings.EqualFold(f, flagName) {
					removed = true
					continue
				}
				newFlags = append(newFlags, f)
			}
			if !removed {
				return &CommandResult{Messages: []string{fmt.Sprintf("Flag %s not found on arch %d.", flagName, item.Archetype)}}
			}
			def.Flags = newFlags
			changeDesc = fmt.Sprintf("Removed flag %s from arch %d (%s)", flagName, item.Archetype, e.getItemNounName(def))
		}

	default:
		validFields := "adj1 adj2 adj3  val1 val2 val3 val4 val5  state  tail  archetype  flag+FLAG flag-FLAG"
		return &CommandResult{Messages: []string{
			fmt.Sprintf("Unknown field '%s'. Valid fields: %s", field, validFields),
		}}
	}

	// --- write back to the slice (worn/inv hold by value, not pointer) ---
	if ref.inWorn {
		target.Worn[ref.wornIdx] = *item
	} else if ref.inInv {
		target.Inventory[ref.invIdx] = *item
	}
	// Wielded is already a pointer — no writeback needed.

	e.SavePlayer(ctx, target)

	// Show the updated item debug block so the GM can confirm the change.
	debugLine := e.formatFullItemDebug(item, ref.label)
	return &CommandResult{Messages: []string{
		fmt.Sprintf("[%s] %s", target.FullName(), changeDesc),
		debugLine,
	}}
}
