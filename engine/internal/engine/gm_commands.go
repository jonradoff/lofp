package engine

import (
	"context"
	"fmt"
	"math/rand"
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
		return &CommandResult{Messages: []string{"You don your Host Hat. You are now visible as a GM."}}
	case "@RFLAG":
		player.GMHat = false
		return &CommandResult{Messages: []string{"You remove your Host Hat."}}
	case "@HIDE":
		player.GMHidden = true
		return &CommandResult{Messages: []string{"You are now hidden from the WHO list."}}
	case "@UNHIDE":
		player.GMHidden = false
		return &CommandResult{Messages: []string{"You are now visible on the WHO list."}}
	case "@INVIS":
		player.GMInvis = true
		player.Hidden = true
		return &CommandResult{Messages: []string{"You fade from sight."}}
	case "@VIS":
		player.GMInvis = false
		player.Hidden = false
		return &CommandResult{Messages: []string{"You become visible again."}}
	case "@SND":
		if len(args) == 0 {
			return &CommandResult{Messages: []string{"Usage: @snd <text>"}}
		}
		text := extractRawArgs(rawInput, 1)
		return &CommandResult{Messages: []string{text}}
	case "@ANNOUNCE":
		return e.gmAnnounce(args, rawInput)
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
		return &CommandResult{Messages: []string{"Monster destroyed."}}
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
		return &CommandResult{Messages: []string{"Speech pattern updated."}}
	case "@LINE1", "@LINE2", "@LINE3":
		return &CommandResult{Messages: []string{"Description line updated."}}
	case "@ENTRY":
		return &CommandResult{Messages: []string{"Entry echo set."}}
	case "@EXIT":
		return &CommandResult{Messages: []string{"Exit echo set."}}
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
		"=== GM Commands ===",
		"@help                         - This help listing",
		"@go <room#>                   - Teleport to a room",
		"@goplr <name>                 - Teleport to a player",
		"@yank <name>                  - Yank a player to your room",
		"@additem <archnum>            - Add item to current room",
		"@delete <item name>           - Delete an item from the room",
		"@get <record#>                - Pick up item by record number",
		"@find <archnum>               - Find all instances of an item",
		"@list                         - List all items",
		"@look <record#>               - Look inside a container",
		"@examine <#>                  - Show type info for a number",
		"@glossary <word>              - Look up a noun/adj by name",
		"@rdata <room#>                - Show room data",
		"@heal <name>                  - Heal a player to full",
		"@kill <name>                  - Kill a player",
		"@exp <name> <points>          - Grant experience",
		"@eds <name> <skill#> <level>  - Set a player's skill",
		"@grantsp <name> <spell>       - Give spell to player",
		"@psi <name> <discipline#>     - Give psi technique",
		"@edplayer <name>              - Show player edit info",
		"@qstat <name>                 - Quick player stat view",
		"@pinv <name>                  - View player inventory",
		"@who                          - List all players with details",
		"@lwho                         - Detailed list with room numbers",
		"@num <name>                   - Show player info by name",
		"@announce <mode> <msg>        - Announce (1=global 2=mindlink)",
		"@snd <text>                   - Echo text in current room",
		"@echoplr <name> <text>        - Echo text to a player",
		"@exclude <name> <text>        - Echo text to room except player",
		"@whisper <name> <text>        - Whisper to player anywhere",
		"@genmon <monster#>            - Generate monster (sedated)",
		"@spawn <monster#>             - Generate monster (active)",
		"@activate                     - Activate a sedated monster",
		"@sedate <monster name>        - Sedate a monster",
		"@zap <monster name>           - Destroy a monster",
		"@gm                          - Put on Host Hat",
		"@rflag                       - Remove Host Hat",
		"@hide / @unhide              - Hide/show on WHO list",
		"@invis / @vis                - Become invisible/visible",
		"@open / @close <item>        - Open/close item silently",
		"@lock / @unlock <item>       - Lock/unlock item silently",
		"@peek <variable>             - View a variable value",
		"@set <variable> <value>      - Set a variable value",
		"@rnd <#>                     - Generate random number 1-#",
		"@suggest <text>              - Add a suggestion",
		"@msg                         - Toggle host messages",
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
	player.RoomNumber = num
	e.SavePlayer(ctx, player)
	result := e.doLook(player)
	result.Messages = append([]string{fmt.Sprintf("Teleported to room %d.", num)}, result.Messages...)
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
	target, err := e.resolvePlayerByName(ctx, args[0])
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	pts, err := strconv.Atoi(args[1])
	if err != nil {
		return &CommandResult{Messages: []string{"Invalid point amount."}}
	}
	target.Experience += pts
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
				msgs = append(msgs, fmt.Sprintf("  MLIST: room %d, monster %d (%s), min %d, max %d", ml.Room, ml.MonsterID, defName, ml.Min, ml.Max))
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
	player.RoomNumber = target.RoomNumber
	e.SavePlayer(ctx, player)
	result := e.doLook(player)
	result.Messages = append([]string{fmt.Sprintf("Teleported to %s (room %d).", target.FullName(), target.RoomNumber)}, result.Messages...)
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

func (e *GameEngine) gmAnnounce(args []string, rawInput string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Usage: @announce <mode> <message>"}}
	}
	mode := args[0]
	text := extractRawArgs(rawInput, 2)
	modeStr := "global"
	if mode == "2" {
		modeStr = "mindlink"
	}
	return &CommandResult{Messages: []string{fmt.Sprintf("[%s announcement] %s", modeStr, text)}}
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
	target, err := e.resolvePlayerByName(ctx, args[0])
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
	target, err := e.resolvePlayerByName(ctx, args[0])
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	return &CommandResult{Messages: []string{fmt.Sprintf("Granted spell %s to %s.", args[1], target.FullName())}}
}

func (e *GameEngine) gmPsi(ctx context.Context, args []string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Usage: @psi <name> <discipline#>"}}
	}
	target, err := e.resolvePlayerByName(ctx, args[0])
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
	return &CommandResult{Messages: []string{fmt.Sprintf("[Echo to %s] %s", args[0], text)}}
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
	return e.resolvePlayerByName(ctx, args[0])
}

// resolvePlayerByName looks up a player by first name.
func (e *GameEngine) resolvePlayerByName(ctx context.Context, name string) (*Player, error) {
	if e.db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	coll := e.db.Collection("players")
	var player Player
	err := coll.FindOne(ctx, bson.M{"firstName": bson.M{"$regex": "^" + name + "$", "$options": "i"}}).Decode(&player)
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
	"@SND", "@ANNOUNCE", "@WHO", "@LWHO", "@NUM", "@QSTAT", "@PINV",
	"@GENMON", "@SPAWN", "@ACTIVATE", "@SEDATE", "@ZAP",
	"@FIND", "@LIST", "@EXAMINE", "@GLOSSARY", "@PEEK", "@SET", "@RND",
	"@OPEN", "@CLOSE", "@LOCK", "@UNLOCK",
	"@GOPLR", "@YANK", "@WHISPER", "@EDPLAYER", "@EDPL", "@EDS", "@EDSK", "@LSK", "@GRANTSP", "@PSI", "@MLIST",
	"@ECHOPLR", "@EXCLUDE", "@SPEECH", "@LINE1", "@LINE2", "@LINE3",
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
