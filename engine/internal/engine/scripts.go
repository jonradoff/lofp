package engine

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// ScriptContext holds state for script execution within a single trigger.
type ScriptContext struct {
	Player   *Player
	Room     *gameworld.Room
	Engine   *GameEngine
	Messages []string // ECHO PLAYER messages to send to the player
	RoomMsgs []string // ECHO ALL / ECHO OTHERS messages for the room
	GMMsgs   []string // GMMSG messages for gamemasters
	Blocked      bool // CLEARVERB: block the triggering action
	MoveTo       int  // MOVE: destination room (0 = no move)
	MoveGroupTo  int  // MOVEGROUP: move all players in room to destination

	StrVars  map[int]string // %0-%9 from STRCVT
	OrigRoom *gameworld.Room // saved room for AFFECT

	// Item interaction context (set when running IFPREVERB/IFVERB on a room item)
	ItemRef *gameworld.RoomItem // the room item being interacted with
	ItemDef *gameworld.ItemDef  // its archetype definition

	DummyVars map[int]int // DUMMY1-5 temporary variables
}

// RunEntryScripts executes all IFENTRY script blocks for a room.
func (e *GameEngine) RunEntryScripts(player *Player, room *gameworld.Room) *ScriptContext {
	sc := &ScriptContext{
		Player: player,
		Room:   room,
		Engine: e,
	}
	for _, block := range room.Scripts {
		if block.Type == "IFENTRY" {
			sc.execBlock(block)
		}
	}
	return sc
}

// RunSayScripts executes IFSAY blocks when a player says something.
func (e *GameEngine) RunSayScripts(player *Player, room *gameworld.Room, text string) *ScriptContext {
	sc := &ScriptContext{
		Player: player,
		Room:   room,
		Engine: e,
	}
	textUpper := strings.ToUpper(text)
	for _, block := range room.Scripts {
		if block.Type == "IFSAY" && len(block.Args) >= 1 {
			// IFSAY args use underscores for spaces
			pattern := strings.ToUpper(strings.ReplaceAll(block.Args[0], "_", " "))
			if textUpper == pattern || strings.Contains(textUpper, pattern) {
				sc.execBlock(block)
			}
		}
	}
	return sc
}

// RunPreverbScripts executes IFPREVERB blocks for a specific verb and item ref.
// Returns the script context. Check sc.Blocked to see if the action should be cancelled.
func (e *GameEngine) RunPreverbScripts(player *Player, room *gameworld.Room, verb string, ri *gameworld.RoomItem, def *gameworld.ItemDef) *ScriptContext {
	sc := &ScriptContext{
		Player:  player,
		Room:    room,
		Engine:  e,
		ItemRef: ri,
		ItemDef: def,
	}
	refStr := fmt.Sprintf("%d", ri.Ref)
	verb = strings.ToUpper(verb)

	// Check room-level scripts
	for _, block := range room.Scripts {
		if block.Type == "IFPREVERB" && len(block.Args) >= 2 {
			if strings.ToUpper(block.Args[0]) == verb && block.Args[1] == refStr {
				sc.execBlock(block)
			}
		}
	}

	// Check item-level scripts (on the archetype definition)
	for _, block := range def.Scripts {
		if block.Type == "IFPREVERB" && len(block.Args) >= 1 {
			if strings.ToUpper(block.Args[0]) == verb {
				// Item scripts use -1 as self-reference
				if len(block.Args) < 2 || block.Args[1] == "-1" {
					sc.execBlock(block)
				}
			}
		}
	}

	return sc
}

// RunVerbScripts executes IFVERB blocks for a specific verb and item.
// RunItemScripts runs all root-level conditional blocks on an item definition
// (IFVAR blocks that aren't wrapped in IFVERB/IFPREVERB). Used for items that
// set values based on adjective checks, e.g., thesnia leaf sets ITEMVAL3=403.
func (e *GameEngine) RunItemScripts(player *Player, room *gameworld.Room, ri *gameworld.RoomItem, def *gameworld.ItemDef) *ScriptContext {
	sc := &ScriptContext{
		Player:  player,
		Room:    room,
		Engine:  e,
		ItemRef: ri,
		ItemDef: def,
	}
	for _, block := range def.Scripts {
		if block.Type == "IFVAR" {
			sc.execBlock(block)
		}
	}
	return sc
}

func (e *GameEngine) RunVerbScripts(player *Player, room *gameworld.Room, verb string, ri *gameworld.RoomItem, def *gameworld.ItemDef) *ScriptContext {
	sc := &ScriptContext{
		Player:  player,
		Room:    room,
		Engine:  e,
		ItemRef: ri,
		ItemDef: def,
	}
	refStr := fmt.Sprintf("%d", ri.Ref)
	verb = strings.ToUpper(verb)

	// Check room-level IFVERB scripts (e.g., IFVERB PUSH 0 in room definition)
	if room != nil {
		for _, block := range room.Scripts {
			if block.Type == "IFVERB" && len(block.Args) >= 2 {
				if strings.ToUpper(block.Args[0]) == verb && block.Args[1] == refStr {
					sc.execBlock(block)
				}
			}
		}
	}

	// Check item-level scripts (on the archetype definition)
	for _, block := range def.Scripts {
		if block.Type == "IFVERB" && len(block.Args) >= 1 {
			if strings.ToUpper(block.Args[0]) == verb {
				if len(block.Args) < 2 || block.Args[1] == "-1" {
					sc.execBlock(block)
				}
			}
		}
	}

	return sc
}

// execBlock executes a script block if its condition is met.
func (sc *ScriptContext) execBlock(block gameworld.ScriptBlock) {
	// GM trace output
	if sc.Player.GMTrace {
		args := strings.Join(block.Args, " ")
		sc.Messages = append(sc.Messages, fmt.Sprintf("[TRACE] %s %s", block.Type, args))
	}

	switch block.Type {
	case "IFENTRY":
		sc.execChildren(block)

	case "IFPREVERB", "IFVERB":
		sc.execChildren(block)

	case "IFVAR":
		result := sc.evalIfVar(block.Args)
		if sc.Player.GMTrace {
			sc.Messages = append(sc.Messages, fmt.Sprintf("[TRACE]   IFVAR %s → %v", strings.Join(block.Args, " "), result))
		}
		if result {
			sc.execChildren(block)
		} else {
			sc.execElse(block)
		}

	case "IFITEM":
		if sc.evalIfItem(block.Args) {
			sc.execChildren(block)
		} else {
			sc.execElse(block)
		}

	case "IFNOITEM":
		if !sc.evalIfItem(block.Args) {
			sc.execChildren(block)
		} else {
			sc.execElse(block)
		}

	case "IFSAY":
		// Condition already matched by caller
		sc.execChildren(block)

	case "IFTOUCH":
		// Condition already matched by caller (touch-type verb)
		sc.execChildren(block)

	case "IFCARRY":
		if sc.evalIfCarry(block.Args) {
			sc.execChildren(block)
		} else {
			sc.execElse(block)
		}

	case "IFLOGIN":
		sc.execChildren(block)

	case "IFFULLDESC":
		if !sc.Player.BriefMode {
			sc.execChildren(block)
		} else {
			sc.execElse(block)
		}

	case "IFIN":
		if sc.evalIfIn(block.Args) {
			sc.execChildren(block)
		} else {
			sc.execElse(block)
		}
	}
}

// execElse runs the ELSE branch of a conditional block (if it has one).
func (sc *ScriptContext) execElse(block gameworld.ScriptBlock) {
	for _, action := range block.ElseActions {
		sc.execAction(action)
	}
	for _, child := range block.ElseChildren {
		sc.execBlock(child)
	}
}

// execChildren runs the actions and nested blocks within a script block.
func (sc *ScriptContext) execChildren(block gameworld.ScriptBlock) {
	for _, action := range block.Actions {
		sc.execAction(action)
	}
	for _, child := range block.Children {
		sc.execBlock(child)
	}
}

// execAction executes a single script action.
func (sc *ScriptContext) execAction(action gameworld.ScriptAction) {
	switch action.Command {
	case "ECHO":
		sc.doEcho(action.Args)
	case "EQUAL":
		sc.doEqual(action.Args)
	case "NEWITEM":
		sc.doNewItem(action.Args)
	case "GMMSG":
		sc.doGMMsg(action.Args)
	case "CLEARVERB":
		sc.Blocked = true
	case "MOVE":
		sc.doMove(action.Args)
	case "MOVEGROUP":
		sc.doMoveGroup(action.Args)
	case "SHOWROOM":
		sc.doShowRoom(action.Args)
	case "PLREVENT", "CONTPLREVENT":
		// Timing/event delay — not yet implemented; ignore silently
	case "AFFECT":
		sc.doAffect(action.Args)
	case "RANDOM":
		sc.doRandom(action.Args)
	case "DAMAGEPLR":
		sc.doDamagePlr(action.Args)
	case "STRCVT":
		sc.doStrCvt(action.Args)
	case "STRCPY":
		sc.doStrCpy(action.Args)
	case "STRCAT":
		sc.doStrCat(action.Args)
	case "POSITION":
		sc.doPosition(action.Args)
	case "ADD":
		sc.doAdd(action.Args)
	case "SUB":
		sc.doSub(action.Args)
	case "SETITEMVAL":
		sc.doSetItemVal(action.Args)
	case "REMOVEITEM":
		sc.doRemoveItem(action.Args)
	case "LOCK":
		sc.doItemState(action.Args, "LOCKED")
	case "UNLOCK":
		sc.doItemState(action.Args, "UNLOCKED")
	case "OPEN":
		sc.doItemState(action.Args, "OPEN")
	case "CLOSE":
		sc.doItemState(action.Args, "CLOSED")
	case "GFLAG":
		sc.doGFlag(action.Args)
	case "RELOGIN":
		if len(action.Args) >= 1 {
			dest := sc.resolveNumericArg(action.Args[0])
			if dest > 0 {
				sc.Player.RoomNumber = dest
			}
		}
	case "MUL":
		sc.doMul(action.Args)
	case "DIV":
		sc.doDiv(action.Args)
	case "MOD":
		sc.doMod(action.Args)
	case "GENMON":
		sc.doGenMon(action.Args)
	case "ZAPMON":
		// Remove all monsters from current room
		if sc.Engine.monsterMgr != nil {
			sc.Engine.monsterMgr.ClearRoom(sc.Room.Number)
			sc.Engine.Events.Publish("monster", fmt.Sprintf("ZAPMON: monsters cleared from room %d", sc.Room.Number))
		}
	case "NEWPUT":
		sc.doNewPut(action.Args)
	case "RECALC":
		// TODO: recalculate player offense/defense after stat changes
	case "DAMAGE":
		// TODO: deal damage to current target (monster or item)
		if len(action.Args) >= 1 {
			amount, _ := strconv.Atoi(action.Args[0])
			sc.Player.BodyPoints -= amount
			if sc.Player.BodyPoints < 0 {
				sc.Player.BodyPoints = 0
			}
		}
	case "DROPLOC":
		// TODO: set room where defeated players are moved
	case "CHANNEL":
		// TODO: set communication channel for room
	}
}

// doEcho handles ECHO PLAYER, ECHO ALL, ECHO OTHERS.
func (sc *ScriptContext) doEcho(args []string) {
	if len(args) < 2 {
		return
	}
	target := strings.ToUpper(args[0])
	text := strings.Join(args[1:], " ")
	text = sc.expandScriptText(text)

	switch target {
	case "PLAYER":
		sc.Messages = append(sc.Messages, text)
	case "ALL":
		sc.Messages = append(sc.Messages, text)
		sc.RoomMsgs = append(sc.RoomMsgs, text)
	case "OTHERS":
		sc.RoomMsgs = append(sc.RoomMsgs, text)
	}
}

// doEqual handles EQUAL INTNUMn value — sets a variable on the player.
func (sc *ScriptContext) doEqual(args []string) {
	if len(args) < 2 {
		return
	}
	varName := strings.ToUpper(args[0])
	val, err := strconv.Atoi(args[1])
	if err != nil {
		return
	}
	sc.setVar(varName, val)
}

// doAdd handles ADD INTNUMn value — increments a variable.
func (sc *ScriptContext) doAdd(args []string) {
	if len(args) < 2 {
		return
	}
	varName := strings.ToUpper(args[0])
	val, err := strconv.Atoi(args[1])
	if err != nil {
		return
	}
	sc.setVar(varName, sc.getVar(varName)+val)
}

// doSub handles SUB INTNUMn value — decrements a variable.
func (sc *ScriptContext) doSub(args []string) {
	if len(args) < 2 {
		return
	}
	varName := strings.ToUpper(args[0])
	val, err := strconv.Atoi(args[1])
	if err != nil {
		return
	}
	sc.setVar(varName, sc.getVar(varName)-val)
}

// doNewItem handles NEWITEM ref archetype [ADJ1=n] [ADJ2=n] [VAL1=n] ...
// ref -1 means add to player inventory.
func (sc *ScriptContext) doNewItem(args []string) {
	if len(args) < 2 {
		return
	}
	ref, err := strconv.Atoi(args[0])
	if err != nil {
		return
	}
	archetype, err := strconv.Atoi(args[1])
	if err != nil {
		return
	}

	item := InventoryItem{Archetype: archetype}
	for _, arg := range args[2:] {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToUpper(parts[0])
		val, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		switch key {
		case "ADJ1":
			item.Adj1 = val
		case "ADJ2":
			item.Adj2 = val
		case "ADJ3":
			item.Adj3 = val
		case "VAL1":
			item.Val1 = val
		case "VAL2":
			item.Val2 = val
		case "VAL3":
			item.Val3 = val
		case "VAL4":
			item.Val4 = val
		case "VAL5":
			item.Val5 = val
		}
	}

	if ref == -1 {
		sc.Player.Inventory = append(sc.Player.Inventory, item)
	}
}

// doGMMsg broadcasts a message to all online GMs.
func (sc *ScriptContext) doGMMsg(args []string) {
	if len(args) == 0 {
		return
	}
	text := strings.Join(args, " ")
	text = sc.expandScriptText(text)
	sc.GMMsgs = append(sc.GMMsgs, fmt.Sprintf("[GM] %s", text))
}

// doMove handles MOVE <room> or MOVE ITEMVAL2, etc.
func (sc *ScriptContext) doMove(args []string) {
	if len(args) == 0 {
		return
	}
	dest := sc.resolveNumericArg(args[0])
	if dest > 0 {
		sc.MoveTo = dest
	}
}

// doMoveGroup handles MOVEGROUP <room> — moves all players in the room to destination.
func (sc *ScriptContext) doMoveGroup(args []string) {
	if len(args) == 0 {
		return
	}
	dest := sc.resolveNumericArg(args[0])
	if dest > 0 {
		sc.MoveGroupTo = dest
	}
}

// doShowRoom handles SHOWROOM <room> or SHOWROOM ITEMVAL2, etc.
func (sc *ScriptContext) doShowRoom(args []string) {
	if len(args) == 0 {
		return
	}
	roomNum := sc.resolveNumericArg(args[0])
	if roomNum > 0 {
		if room := sc.Engine.rooms[roomNum]; room != nil {
			sc.Messages = append(sc.Messages, fmt.Sprintf("[%s]", room.Name))
			if room.Description != "" {
				sc.Messages = append(sc.Messages, descriptionToMessages(room.Description)...)
			}
		}
	}
}

// doSetItemVal handles SETITEMVAL ref valIndex value.
func (sc *ScriptContext) doSetItemVal(args []string) {
	// Not yet fully implemented; needs room item mutation
}

// doRemoveItem handles REMOVEITEM ref — removes item from player or room.
func (sc *ScriptContext) doRemoveItem(args []string) {
	if len(args) == 0 {
		return
	}
	ref, err := strconv.Atoi(args[0])
	if err != nil {
		return
	}
	if ref == -1 && sc.ItemRef != nil {
		// Remove current item from inventory (by archetype match)
		for i, ii := range sc.Player.Inventory {
			if ii.Archetype == sc.ItemRef.Archetype {
				sc.Player.Inventory = append(sc.Player.Inventory[:i], sc.Player.Inventory[i+1:]...)
				break
			}
		}
	}
}

// doItemState sets the state of a room item (LOCK, UNLOCK, OPEN, CLOSE).
func (sc *ScriptContext) doItemState(args []string, state string) {
	if len(args) == 0 {
		return
	}
	ref, err := strconv.Atoi(args[0])
	if err != nil {
		return
	}
	for i := range sc.Room.Items {
		if sc.Room.Items[i].Ref == ref && !sc.Room.Items[i].IsPut {
			sc.Room.Items[i].State = state
			sc.Engine.notifyRoomChange(RoomChange{RoomNumber: sc.Room.Number, Type: "item_state", ItemRef: ref, NewState: state})
			break
		}
	}
}

// evalIfVar evaluates IFVAR conditions like "INTNUM6 = 2" or "ITEMBIT5 = 0".
func (sc *ScriptContext) evalIfVar(args []string) bool {
	if len(args) < 3 {
		return false
	}
	varName := strings.ToUpper(args[0])
	op := args[1]
	expected, err := strconv.Atoi(args[2])
	if err != nil {
		return false
	}

	actual := sc.getVar(varName)

	switch op {
	case "=":
		return actual == expected
	case "!":
		return actual != expected
	case ">":
		return actual > expected
	case "<":
		return actual < expected
	case ">=":
		return actual >= expected
	case "<=":
		return actual <= expected
	}
	return false
}

// evalIfItem evaluates IFITEM conditions like "IFITEM 0 OPEN" or "IFITEM -1 CLOSED".
func (sc *ScriptContext) evalIfItem(args []string) bool {
	if len(args) < 2 {
		return false
	}
	ref, err := strconv.Atoi(args[0])
	if err != nil {
		return false
	}
	expectedState := strings.ToUpper(args[1])

	var ri *gameworld.RoomItem
	if ref == -1 && sc.ItemRef != nil {
		ri = sc.ItemRef
	} else {
		for i := range sc.Room.Items {
			if sc.Room.Items[i].Ref == ref && !sc.Room.Items[i].IsPut {
				ri = &sc.Room.Items[i]
				break
			}
		}
	}
	if ri == nil {
		return false
	}

	state := strings.ToUpper(ri.State)
	switch expectedState {
	case "OPEN":
		return state == "OPEN"
	case "CLOSED":
		return state == "CLOSED" || state == ""
	case "LOCKED":
		return state == "LOCKED"
	case "UNLOCKED":
		return state == "UNLOCKED" || state == "OPEN"
	}
	return false
}

// getVar retrieves a variable value for the player or current item.
func (sc *ScriptContext) getVar(name string) int {
	if strings.HasPrefix(name, "DUMMY") {
		idx, err := strconv.Atoi(name[5:])
		if err != nil {
			return 0
		}
		if sc.DummyVars != nil {
			return sc.DummyVars[idx]
		}
		return 0
	}
	if strings.HasPrefix(name, "INTNUM") {
		idx, err := strconv.Atoi(name[6:])
		if err != nil {
			return 0
		}
		if sc.Player.IntNums == nil {
			return 0
		}
		return sc.Player.IntNums[idx]
	}
	if strings.HasPrefix(name, "ITEMBIT") {
		idx, err := strconv.Atoi(name[7:])
		if err != nil || sc.ItemRef == nil {
			return 0
		}
		if sc.ItemRef.Val4&(1<<idx) != 0 {
			return 1
		}
		return 0
	}
	if strings.HasPrefix(name, "ITEMVAL") {
		idx, err := strconv.Atoi(name[7:])
		if err != nil || sc.ItemRef == nil {
			return 0
		}
		switch idx {
		case 1:
			return sc.ItemRef.Val1
		case 2:
			return sc.ItemRef.Val2
		case 3:
			return sc.ItemRef.Val3
		case 4:
			return sc.ItemRef.Val4
		case 5:
			return sc.ItemRef.Val5
		}
		return 0
	}
	if strings.HasPrefix(name, "ITEMADJ") {
		idx, err := strconv.Atoi(name[7:])
		if err != nil || sc.ItemRef == nil {
			return 0
		}
		switch idx {
		case 1:
			return sc.ItemRef.Adj1
		case 2:
			return sc.ItemRef.Adj2
		case 3:
			return sc.ItemRef.Adj3
		}
		return 0
	}
	// SKILL variables — stored in player Skills map
	if strings.HasPrefix(name, "SKILL") {
		idx, err := strconv.Atoi(name[5:])
		if err != nil {
			return 0
		}
		if sc.Player.Skills == nil {
			return 0
		}
		return sc.Player.Skills[idx]
	}
	// EXIT variables
	if strings.HasPrefix(name, "EXIT") {
		dir := name[4:] // e.g., EXITN -> N, EXITS -> S
		if sc.Room != nil {
			if dest, ok := sc.Room.Exits[dir]; ok {
				return dest
			}
		}
		return 0
	}
	// FLAG variables
	if strings.HasPrefix(name, "FLAG") {
		idx, err := strconv.Atoi(name[4:])
		if err != nil {
			return 0
		}
		switch idx {
		case 1: return sc.Player.Flag1
		case 2: return sc.Player.Flag2
		case 3: return sc.Player.Flag3
		case 4: return sc.Player.Flag4
		}
		return 0
	}

	if strings.HasPrefix(name, "PVAL") {
		idx, err := strconv.Atoi(name[4:])
		if err != nil {
			return 0
		}
		if sc.Engine.PVals != nil {
			return sc.Engine.PVals[idx]
		}
		return 0
	}

	switch name {
	// Player level/race
	case "LEV":
		return sc.Player.Level
	case "RAC":
		return sc.Player.Race
	case "MISTFORM":
		if sc.Player.Race == 8 { return 1 }
		return 0
	// Player stats
	case "STR", "STRT":
		return sc.Player.Strength
	case "AGI", "AGIT":
		return sc.Player.Agility
	case "CON", "CONT":
		return sc.Player.Constitution
	case "QUI", "QUIT":
		return sc.Player.Quickness
	case "WIL", "WILT":
		return sc.Player.Willpower
	case "PER", "PERT":
		return sc.Player.Perception
	case "EMP", "EMPT":
		return sc.Player.Empathy
	// Player resources
	case "BODYPOINTS":
		return sc.Player.BodyPoints
	case "MAXBODY":
		return sc.Player.MaxBodyPoints
	case "FATPOINTS":
		return sc.Player.Fatigue
	case "MAXFAT":
		return sc.Player.MaxFatigue
	case "MANAPOINTS":
		return sc.Player.Mana
	case "MAXMANA":
		return sc.Player.MaxMana
	case "PSIPOINTS":
		return sc.Player.Psi
	case "MAXPSI":
		return sc.Player.MaxPsi
	// Player state
	case "DEAD":
		if sc.Player.Dead { return 1 }
		return 0
	case "FLYING":
		if sc.Player.Position == 4 { return 1 }
		return 0
	case "KNEELING":
		if sc.Player.Position == 3 { return 1 }
		return 0
	case "LAYING":
		if sc.Player.Position == 2 { return 1 }
		return 0
	case "SITTING":
		if sc.Player.Position == 1 { return 1 }
		return 0
	case "STANDING":
		if sc.Player.Position == 0 { return 1 }
		return 0
	case "HIDDEN":
		if sc.Player.Hidden { return 1 }
		return 0
	// Organization
	case "ORG":
		return sc.Player.Organization
	case "ORGRANK":
		return sc.Player.OrgRank
	case "ALIGN":
		return sc.Player.Alignment
	case "POINTS":
		return sc.Player.BuildPoints
	// Wielded
	case "WIELDED":
		if sc.Player.Wielded != nil { return 1 }
		return 0
	case "ARCHNUM":
		if sc.Player.Wielded != nil { return sc.Player.Wielded.Archetype }
		return 0
	// Room info
	case "RNUM":
		return sc.Player.RoomNumber
	case "OUTDOOR":
		if sc.Room != nil && isOutdoorTerrain(sc.Room.Terrain) { return 1 }
		return 0
	case "PLRSINROOM":
		if sc.Engine.sessions != nil {
			count := 0
			for _, p := range sc.Engine.sessions.OnlinePlayers() {
				if p.RoomNumber == sc.Player.RoomNumber { count++ }
			}
			return count
		}
		return 1
	case "MONINROOM":
		if sc.Engine.monsterMgr != nil {
			return len(sc.Engine.monsterMgr.MonstersInRoom(sc.Player.RoomNumber))
		}
		return 0
	// Time
	case "TIM":
		return GameHour()
	case "DAY":
		if IsDay() { return 1 }
		return 0
	case "NIGHT":
		if IsNight() { return 1 }
		return 0
	case "DATE":
		return GameDay()
	case "MONTH":
		return GameMonth()
	case "YEAR":
		return GameYear()
	// Weather
	case "WEA":
		if sc.Engine.RegionWeather != nil {
			return sc.Engine.RegionWeather[0] // default region
		}
		return 0
	// Gender
	case "GEN", "GENT":
		return sc.Player.Gender
	// Physical attributes
	case "HEI":
		return sc.Player.Height
	case "HEIT":
		return sc.Player.HeightTrue
	case "WEI":
		return sc.Player.Weight
	case "WEIT":
		return sc.Player.WeightTrue
	case "AGE":
		return sc.Player.Age
	case "AGET":
		return sc.Player.AgeTrue
	// Form states
	case "WOLFFORM":
		if sc.Player.WolfForm { return 1 }
		return 0
	case "SLIMEFORM":
		if sc.Player.SlimeForm { return 1 }
		return 0
	case "OTHERFORM":
		if sc.Player.WolfForm || sc.Player.SlimeForm || (sc.Player.Race == 8 && sc.Player.Hidden) { return 1 }
		return 0
	case "UNDEAD":
		if sc.Player.Undead { return 1 }
		return 0
	case "DISGUISED":
		if sc.Player.Disguised { return 1 }
		return 0
	case "SLEEPING":
		if sc.Player.Sleeping { return 1 }
		return 0
	case "SUBMITTING":
		if sc.Player.Submitting { return 1 }
		return 0
	case "ROUNDTIME":
		return sc.Player.RoundTime
	case "SPELLNUM":
		return sc.Player.PreparedSpell
	case "POSITION":
		return sc.Player.Position
	// Wealth
	case "WEALTH":
		return sc.Player.Gold*100 + sc.Player.Silver*10 + sc.Player.Copper
	// Room
	case "WILDERNESS":
		if sc.Room != nil {
			switch sc.Room.Terrain {
			case "FOREST", "MOUNTAIN", "PLAIN", "SWAMP", "JUNGLE", "WASTE":
				return 1
			}
		}
		return 0
	case "ASTRAL":
		if sc.Room != nil && sc.Room.Terrain == "ASTRAL" { return 1 }
		return 0
	case "TERRAIN":
		if sc.Room != nil {
			// Return a stable hash for terrain comparisons
			terrainMap := map[string]int{
				"INDOOR_FLOOR": 1, "INDOOR_GROUND": 2, "CAVE": 3, "DEEPCAVE": 4,
				"FOREST": 5, "MOUNTAIN": 6, "PLAIN": 7, "SWAMP": 8, "JUNGLE": 9,
				"WASTE": 10, "OUTDOOR_OTHER": 11, "OUTDOOR_FLOOR": 12, "AERIAL": 13,
				"ASTRAL": 14, "UNDERSEA": 15,
			}
			return terrainMap[sc.Room.Terrain]
		}
		return 0
	case "REGION":
		if sc.Room != nil {
			return sc.Room.Region
		}
		return 0
	case "MOVEABLE":
		if sc.Player.Position == 0 && !sc.Player.Immobilized && !sc.Player.Stunned {
			return 1
		}
		return 0
	case "DEPARTROOM":
		return 0 // TODO: track last departure room
	case "WEALTH1":
		return sc.Player.Gold*100 + sc.Player.Silver*10 + sc.Player.Copper
	case "WEALTH2", "WEALTH3", "WEALTH4", "WEALTH5", "WEALTH6", "WEALTH7", "WEALTH8", "WEALTH9":
		return 0 // TODO: multi-currency per region
	case "OBJWEIGHT":
		if sc.ItemDef != nil { return sc.ItemDef.Weight }
		return 0
	case "PLAYERNUM":
		return 0 // TODO: unique player number
	case "WARRANT":
		return sc.Player.Warrant
	case "GFLAG1":
		return sc.Player.IntNums[901]
	case "GFLAG2":
		return sc.Player.IntNums[902]
	case "GFLAG3":
		return sc.Player.IntNums[903]
	case "GFLAG4":
		return sc.Player.IntNums[904]
	case "NUMPLRS":
		if sc.Engine.sessions != nil {
			return len(sc.Engine.sessions.OnlinePlayers())
		}
		return 0
	case "ARENADEATH":
		return sc.Player.IntNums[905]
	}
	// Check named global variables (DANWATER, TECHSWITCH, etc.)
	if sc.Engine.namedVarNames[name] {
		return sc.Engine.NamedVars[name]
	}
	return 0
}

// setVar sets a variable value on the player or current item.
func (sc *ScriptContext) setVar(name string, val int) {
	if strings.HasPrefix(name, "DUMMY") {
		idx, _ := strconv.Atoi(name[5:])
		if sc.DummyVars == nil {
			sc.DummyVars = make(map[int]int)
		}
		sc.DummyVars[idx] = val
		return
	}
	if strings.HasPrefix(name, "INTNUM") {
		idx, err := strconv.Atoi(name[6:])
		if err != nil {
			return
		}
		if sc.Player.IntNums == nil {
			sc.Player.IntNums = make(map[int]int)
		}
		sc.Player.IntNums[idx] = val
		return
	}
	if strings.HasPrefix(name, "ITEMVAL") {
		idx, err := strconv.Atoi(name[7:])
		if err != nil || sc.ItemRef == nil {
			return
		}
		switch idx {
		case 1:
			sc.ItemRef.Val1 = val
		case 2:
			sc.ItemRef.Val2 = val
		case 3:
			sc.ItemRef.Val3 = val
		case 4:
			sc.ItemRef.Val4 = val
		case 5:
			sc.ItemRef.Val5 = val
		}
		itemCopy := *sc.ItemRef
		sc.Engine.notifyRoomChange(RoomChange{
			RoomNumber: sc.Room.Number, Type: "item_update",
			ItemRef: sc.ItemRef.Ref, Item: &itemCopy,
		})
		return
	}
	if strings.HasPrefix(name, "FLAG") {
		idx, _ := strconv.Atoi(name[4:])
		switch idx {
		case 1: sc.Player.Flag1 = val
		case 2: sc.Player.Flag2 = val
		case 3: sc.Player.Flag3 = val
		case 4: sc.Player.Flag4 = val
		}
		return
	}
	if strings.HasPrefix(name, "PVAL") {
		idx, _ := strconv.Atoi(name[4:])
		if sc.Engine.PVals == nil {
			sc.Engine.PVals = make(map[int]int)
		}
		sc.Engine.PVals[idx] = val
		sc.Engine.savePVals()
		return
	}
	switch name {
	case "ORG":
		sc.Player.Organization = val
	case "ORGRANK":
		sc.Player.OrgRank = val
	case "ALIGN":
		sc.Player.Alignment = val
	case "BODYPOINTS":
		sc.Player.BodyPoints = val
	case "MANAPOINTS":
		sc.Player.Mana = val
	case "PSIPOINTS":
		sc.Player.Psi = val
	case "FATPOINTS":
		sc.Player.Fatigue = val
	case "WEALTH":
		// WEALTH is in copper units — split into gold/silver/copper
		sc.Player.Gold = val / 100
		sc.Player.Silver = (val % 100) / 10
		sc.Player.Copper = val % 10
	default:
		// Check named global variables (DANWATER, TECHSWITCH, etc.)
		if sc.Engine.namedVarNames[name] {
			sc.Engine.NamedVars[name] = val
			// Publish to event monitor and hub for cross-machine sync
			sc.Engine.Events.Publish("world", fmt.Sprintf("Variable %s = %d", name, val))
			if sc.Engine.onRoomChange != nil {
				sc.Engine.onRoomChange(RoomChange{
					Type: "named_var", NewState: fmt.Sprintf("%s=%d", name, val),
				})
			}
		}
	}
}


// resolveNumericArg resolves a script argument that can be a literal number
// or a variable reference like ITEMVAL2.
func (sc *ScriptContext) resolveNumericArg(arg string) int {
	upper := strings.ToUpper(arg)
	if strings.HasPrefix(upper, "ITEMVAL") {
		return sc.getVar(upper)
	}
	val, err := strconv.Atoi(arg)
	if err != nil {
		return 0
	}
	return val
}

// expandScriptText replaces script placeholders in text.
// evalIfCarry checks if player carries an item with matching archetype (and optional adj).
func (sc *ScriptContext) evalIfCarry(args []string) bool {
	if len(args) < 1 {
		return false
	}
	archetype, err := strconv.Atoi(args[0])
	if err != nil {
		return false
	}
	adj := -1
	if len(args) >= 2 {
		adj, _ = strconv.Atoi(args[1])
	}
	for _, ii := range sc.Player.Inventory {
		if ii.Archetype == archetype {
			if adj < 0 || ii.Adj1 == adj {
				return true
			}
		}
	}
	return false
}

// doAffect switches script context to a different room.
func (sc *ScriptContext) doAffect(args []string) {
	if len(args) == 0 {
		return
	}
	roomNum := sc.resolveNumericArg(args[0])
	if roomNum > 0 {
		if room := sc.Engine.rooms[roomNum]; room != nil {
			if sc.OrigRoom == nil {
				sc.OrigRoom = sc.Room
			}
			sc.Room = room
		}
	}
}

// doRandom sets a variable to a random value.
func (sc *ScriptContext) doRandom(args []string) {
	if len(args) < 2 {
		return
	}
	varName := strings.ToUpper(args[0])
	max, err := strconv.Atoi(args[1])
	if err != nil || max <= 0 {
		return
	}
	sc.setVar(varName, rand.Intn(max))
}

// doDamagePlr applies damage to the player.
func (sc *ScriptContext) doDamagePlr(args []string) {
	if len(args) < 2 {
		return
	}
	// DAMAGEPLR BODYONLY <amount> <text...>
	// or DAMAGEPLR <amount> <text...>
	idx := 0
	if strings.ToUpper(args[0]) == "BODYONLY" {
		idx = 1
	}
	if idx >= len(args) {
		return
	}
	amount, err := strconv.Atoi(args[idx])
	if err != nil {
		return
	}
	sc.Player.BodyPoints -= amount
	if sc.Player.BodyPoints < 0 {
		sc.Player.BodyPoints = 0
	}
	if idx+1 < len(args) {
		text := strings.Join(args[idx+1:], " ")
		sc.Messages = append(sc.Messages, sc.expandScriptText(text))
	}
}

// doStrCvt converts a variable to a string for %0-%9 substitution.
func (sc *ScriptContext) doStrCvt(args []string) {
	if len(args) < 2 {
		return
	}
	digit, err := strconv.Atoi(args[0])
	if err != nil || digit < 0 || digit > 9 {
		return
	}
	varName := strings.ToUpper(args[1])
	val := sc.getVar(varName)
	if sc.StrVars == nil {
		sc.StrVars = make(map[int]string)
	}
	sc.StrVars[digit] = strconv.Itoa(val)
}

// doStrCpy sets a string variable directly: STRCPY <digit> "<text>"
func (sc *ScriptContext) doStrCpy(args []string) {
	if len(args) < 2 {
		return
	}
	digit, err := strconv.Atoi(args[0])
	if err != nil || digit < 0 || digit > 9 {
		return
	}
	// Join remaining args and strip quotes
	text := strings.Join(args[1:], " ")
	text = strings.Trim(text, "\"")
	if sc.StrVars == nil {
		sc.StrVars = make(map[int]string)
	}
	sc.StrVars[digit] = text
}

// doStrCat appends to a string variable: STRCAT <digit> "<text>"
func (sc *ScriptContext) doStrCat(args []string) {
	if len(args) < 2 {
		return
	}
	digit, err := strconv.Atoi(args[0])
	if err != nil || digit < 0 || digit > 9 {
		return
	}
	text := strings.Join(args[1:], " ")
	text = strings.Trim(text, "\"")
	if sc.StrVars == nil {
		sc.StrVars = make(map[int]string)
	}
	sc.StrVars[digit] += text
}

// doPosition forces the player into a position.
func (sc *ScriptContext) doPosition(args []string) {
	if len(args) == 0 {
		return
	}
	switch strings.ToUpper(args[0]) {
	case "STAND":
		sc.Player.Position = 0
	case "SIT":
		sc.Player.Position = 1
	case "LAY":
		sc.Player.Position = 2
	case "KNEEL":
		sc.Player.Position = 3
	}
}

// doGFlag sets FLAG for all players in the room.
func (sc *ScriptContext) doGFlag(args []string) {
	if len(args) < 2 {
		return
	}
	idx, _ := strconv.Atoi(args[0])
	val, _ := strconv.Atoi(args[1])
	if sc.Engine.sessions != nil {
		for _, p := range sc.Engine.sessions.OnlinePlayers() {
			if p.RoomNumber == sc.Room.Number {
				switch idx {
				case 1:
					p.Flag1 = val
				case 2:
					p.Flag2 = val
				case 3:
					p.Flag3 = val
				case 4:
					p.Flag4 = val
				}
			}
		}
	}
}

// doMul handles MUL varName value — multiplies a variable.
func (sc *ScriptContext) doMul(args []string) {
	if len(args) < 2 {
		return
	}
	varName := strings.ToUpper(args[0])
	val, err := strconv.Atoi(args[1])
	if err != nil {
		return
	}
	sc.setVar(varName, sc.getVar(varName)*val)
}

// doDiv handles DIV varName value — divides a variable.
func (sc *ScriptContext) doDiv(args []string) {
	if len(args) < 2 {
		return
	}
	varName := strings.ToUpper(args[0])
	val, err := strconv.Atoi(args[1])
	if err != nil || val == 0 {
		return
	}
	sc.setVar(varName, sc.getVar(varName)/val)
}

// doMod handles MOD varName value — modulo a variable.
func (sc *ScriptContext) doMod(args []string) {
	if len(args) < 2 {
		return
	}
	varName := strings.ToUpper(args[0])
	val, err := strconv.Atoi(args[1])
	if err != nil || val == 0 {
		return
	}
	sc.setVar(varName, sc.getVar(varName)%val)
}

// doGenMon spawns a monster in the current room.
func (sc *ScriptContext) doGenMon(args []string) {
	if len(args) == 0 {
		return
	}
	monNum, err := strconv.Atoi(args[0])
	if err != nil {
		return
	}
	def := sc.Engine.monsters[monNum]
	if def == nil {
		return
	}
	if sc.Engine.monsterMgr != nil {
		sc.Engine.monsterMgr.SpawnOne(monNum, sc.Room.Number, def.Body)
		name := FormatMonsterName(def, sc.Engine.monAdjs)
		genText := def.TextOverrides["TEXG"]
		if genText == "" {
			genText = fmt.Sprintf("A %s appears!", name)
		}
		sc.RoomMsgs = append(sc.RoomMsgs, genText)
		sc.Engine.Events.Publish("monster", fmt.Sprintf("GENMON: %s spawned in room %d", name, sc.Room.Number))
	}
}

// doNewPut handles NEWPUT ref archetype [key=value...] — places item inside a container in the room.
func (sc *ScriptContext) doNewPut(args []string) {
	if len(args) < 2 {
		return
	}
	ref, _ := strconv.Atoi(args[0])
	archetype, _ := strconv.Atoi(args[1])
	item := gameworld.RoomItem{Ref: ref, Archetype: archetype, IsPut: true}
	for _, arg := range args[2:] {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToUpper(parts[0])
		val, _ := strconv.Atoi(parts[1])
		switch key {
		case "ADJ1":
			item.Adj1 = val
		case "ADJ2":
			item.Adj2 = val
		case "ADJ3":
			item.Adj3 = val
		case "VAL1":
			item.Val1 = val
		case "VAL2":
			item.Val2 = val
		case "VAL3":
			item.Val3 = val
		case "VAL4":
			item.Val4 = val
		case "VAL5":
			item.Val5 = val
		}
	}
	// Find the PutIn target ref
	if ref >= 0 {
		for i := range sc.Room.Items {
			if sc.Room.Items[i].Ref == ref && !sc.Room.Items[i].IsPut {
				item.PutIn = ref
				break
			}
		}
	}
	sc.Room.Items = append(sc.Room.Items, item)
}

// evalIfIn checks if a container in the room holds an item of the given archetype.
func (sc *ScriptContext) evalIfIn(args []string) bool {
	if len(args) < 2 || sc.Room == nil {
		return false
	}
	containerRef, _ := strconv.Atoi(args[0])
	archName := strings.ToUpper(args[1])
	archNum := sc.getVar(archName)
	if archNum == 0 {
		archNum, _ = strconv.Atoi(args[1])
	}
	for _, ri := range sc.Room.Items {
		if ri.IsPut && ri.PutIn == containerRef && ri.Archetype == archNum {
			return true
		}
	}
	return false
}

func (sc *ScriptContext) expandScriptText(text string) string {
	// Player name
	text = strings.ReplaceAll(text, "%N", sc.Player.FirstName)
	text = strings.ReplaceAll(text, "%n", sc.Player.FirstName)
	// Group name (just player name for now)
	text = strings.ReplaceAll(text, "%p", sc.Player.FirstName)
	text = strings.ReplaceAll(text, "%P", sc.Player.FirstName)
	// Item name
	if sc.ItemRef != nil && sc.ItemDef != nil {
		itemName := sc.Engine.formatItemName(sc.ItemDef, sc.ItemRef.Adj1, sc.ItemRef.Adj2, sc.ItemRef.Adj3)
		text = strings.ReplaceAll(text, "%a", itemName)
	}
	// Monster name (empty for now)
	text = strings.ReplaceAll(text, "%m", "")
	// Newline
	text = strings.ReplaceAll(text, "%c", "\n")
	// Gender-based pronouns (canonical from manual)
	if sc.Player.Gender == 0 {
		text = strings.ReplaceAll(text, "%h", "his")
		text = strings.ReplaceAll(text, "%s", "he")
		text = strings.ReplaceAll(text, "%i", "him")
	} else {
		text = strings.ReplaceAll(text, "%h", "her")
		text = strings.ReplaceAll(text, "%s", "she")
		text = strings.ReplaceAll(text, "%i", "her")
	}
	// Legacy aliases
	text = strings.ReplaceAll(text, "%e", func() string { if sc.Player.Gender == 0 { return "he" }; return "she" }())
	text = strings.ReplaceAll(text, "%o", func() string { if sc.Player.Gender == 0 { return "him" }; return "her" }())
	// STRCVT %0-%9
	if sc.StrVars != nil {
		for i := 0; i <= 9; i++ {
			if v, ok := sc.StrVars[i]; ok {
				text = strings.ReplaceAll(text, fmt.Sprintf("%%%d", i), v)
			}
		}
	}
	return text
}
