package engine

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// scriptEventCycleSeconds is the real-time duration of one SETEVENT cycle.
// SETEVENT N cycles ALWAYS delays N × this many seconds.
const scriptEventCycleSeconds = 5

// ScriptSegment is a time-delayed group of script steps created by a SETEVENT/CONTEVENT
// or PLREVENT/CONTPLREVENT pair.
type ScriptSegment struct {
	RelativeSeconds int                     // seconds to wait after previous segment fires
	Steps           []gameworld.ScriptStep  // remaining actions/nested blocks to run when segment fires, in original order
	RoomNumber      int                     // sc.Room.Number at time of scheduling
}

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
	PreMoveMsgs  []string // RoomMsgs captured at the moment MOVE first fires (for old room)

	StrVars  map[int]string // %0-%9 from STRCVT
	OrigRoom *gameworld.Room // saved room for AFFECT

	// OrigRoomNum is the player's room number before any MOVE fires during this script.
	// Set once by doMove so callers can use it for OldRoom tracking after the script runs.
	OrigRoomNum int

	// Item interaction context (set when running IFPREVERB/IFVERB on a room item)
	ItemRef *gameworld.RoomItem // the room item being interacted with
	ItemDef *gameworld.ItemDef  // its archetype definition

	DummyVars      map[int]int // DUMMY1-5 temporary variables
	lastOrgChecked int         // last org number evaluated in IFVAR ORG, used to resolve ORGRANK

	// Set during RunPreverbScripts so IFPREVERB/IFTOUCH blocks nested inside
	// IFVAR trees only fire for the triggering verb and item ref.
	activeVerb string
	activeRef  string

	// sayText holds the normalized (uppercased, trailing-punctuation-trimmed) text a
	// player just said, set by RunSayScripts. IFSAY blocks match against it wherever
	// they're encountered — including nested inside a top-level IFVAR tree gating a
	// multi-stage conversation — rather than only at the room's top level.
	sayText string

	KillPlayer        bool // KILL PLAYER: set when script kills the player
	NeedsSave         bool // set by ROUTINE and similar actions that modify player state
	routineChargePaid bool // true after first charge decrement this interaction (prevents double-spend across script phases)
	preverbOnly       bool // set by RunPreverbScripts; causes execBlock to skip IFVERB blocks

	// moveGroupFired is set by doMoveGroup so that subsequent ECHO PLAYER/OTHERS in the same
	// script block are suppressed — the group has conceptually already left the source room.
	moveGroupFired bool

	// RoundTimeSet holds the value passed to EQUAL ROUNDTIME so callers can emit [Round: X sec].
	RoundTimeSet int

	// SETEVENT/CONTEVENT and PLREVENT/CONTPLREVENT deferred execution
	pendingEventDelay int  // delay set by SETEVENT (in cycles) or PLREVENT (in raw seconds)
	pendingEventIsRaw bool // true when PLREVENT set the delay (seconds); false for SETEVENT (multiply by cycle)
	DeferredSegments  []ScriptSegment // segments to run after their respective delays
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

// RunSayScripts executes IFSAY blocks when a player says something. IFSAY blocks are
// matched wherever they're found — at the room's top level, or nested inside a
// top-level IFVAR tree that gates a multi-stage conversation (e.g. "has the player
// already paid tribute?" before listening for "passage"/"wisdom"). Only IFSAY and
// IFVAR are walked at the top level; IFVERB/IFPREVERB/IFENTRY etc. have their own
// dedicated runners and must not fire just because the player said something.
func (e *GameEngine) RunSayScripts(player *Player, room *gameworld.Room, text string) *ScriptContext {
	sc := &ScriptContext{
		Player:  player,
		Room:    room,
		Engine:  e,
		sayText: strings.ToUpper(strings.TrimRight(text, ".?!")),
	}
	for _, block := range room.Scripts {
		if block.Type == "IFSAY" || block.Type == "IFVAR" {
			sc.execBlock(block)
		}
	}
	return sc
}

// matchesSayText reports whether sc.sayText matches an IFSAY block's pattern.
// IFSAY args use underscores for spaces; trailing punctuation is trimmed so
// "computer, identify" matches the pattern "COMPUTER,_IDENTIFY."
func (sc *ScriptContext) matchesSayText(args []string) bool {
	if sc.sayText == "" || len(args) < 1 {
		return false
	}
	pattern := strings.ToUpper(strings.ReplaceAll(args[0], "_", " "))
	pattern = strings.TrimRight(pattern, ".?!")
	return sc.sayText == pattern || strings.Contains(sc.sayText, pattern)
}

// RunPreverbScripts executes IFPREVERB blocks for a specific verb and item ref.
// Returns the script context. Check sc.Blocked to see if the action should be cancelled.
func (e *GameEngine) RunPreverbScripts(player *Player, room *gameworld.Room, verb string, ri *gameworld.RoomItem, def *gameworld.ItemDef) *ScriptContext {
	refStr := fmt.Sprintf("%d", ri.Ref)
	verb = strings.ToUpper(verb)

	sc := &ScriptContext{
		Player:      player,
		Room:        room,
		Engine:      e,
		ItemRef:     ri,
		ItemDef:     def,
		activeVerb:  verb,
		activeRef:   refStr,
		preverbOnly: true,
	}

	// Check room-level scripts (only for room items; inventory items have Ref=-1)
	if ri.Ref >= 0 {
		// Run scripts matching this specific item ref
		for _, block := range room.Scripts {
			if block.Type == "IFPREVERB" && len(block.Args) >= 2 {
				if verbsMatch(strings.ToUpper(block.Args[0]), verb) && block.Args[1] == refStr {
					sc.execBlock(block)
				}
			}
		}
		// Also run room catch-all scripts (IFPREVERB VERB -1) with this item as context.
		// These fire for any use of the verb in the room and use ARCHNUM/ITEMADJ internally
		// to filter which items they act on.
		for _, block := range room.Scripts {
			if block.Type == "IFPREVERB" && len(block.Args) >= 2 {
				if verbsMatch(strings.ToUpper(block.Args[0]), verb) && block.Args[1] == "-1" {
					sc.execBlock(block)
				}
			}
		}
		// Run room-level IFTOUCH blocks; execBlock filters by activeVerb and activeRef.
		for _, block := range room.Scripts {
			if block.Type == "IFTOUCH" {
				sc.execBlock(block)
			}
		}
	}

	// Check item-level scripts (on the archetype definition).
	// Item scripts may have IFPREVERB blocks nested inside IFVAR trees (e.g., the alley
	// item checks IFVAR ITEMVAL5 then IFVAR ORG before exposing IFPREVERB GO).
	// We execute all block types here; execBlock uses activeVerb/activeRef to filter
	// IFPREVERB/IFTOUCH blocks that don't match the triggering verb.
	for _, block := range def.Scripts {
		switch block.Type {
		case "IFPREVERB", "IFVERB", "IFTOUCH", "IFVAR", "IFITEM", "IFNOITEM", "IFCARRY":
			sc.execBlock(block)
		}
	}

	return sc
}

// verbsMatch reports whether two script verb keywords refer to the same player action.
// LOOK and EXAMINE are written interchangeably across the original scripts for "look
// closely at this item", even though the engine only ever dispatches one of them
// (LOOK) when running scripts for the LOOK/EXAMINE/INSPECT commands.
func verbsMatch(a, b string) bool {
	if a == b {
		return true
	}
	isLookOrExamine := func(v string) bool { return v == "LOOK" || v == "EXAMINE" }
	return isLookOrExamine(a) && isLookOrExamine(b)
}

// RunVerbScripts executes IFVERB blocks for a specific verb and item.
// RunItemScripts runs all root-level conditional blocks on an item definition
// (IFVAR blocks that aren't wrapped in IFVERB/IFPREVERB), plus any IFVERB/IFPREVERB
// blocks nested inside those IFVAR trees whose verb matches the triggering verb —
// RunVerbScripts/RunPreverbScripts only match verb blocks at the top level of
// def.Scripts, so a script like item 1669's "IFVAR ... IFVERB EXAMINE -1 ... ENDIF"
// relies on this walk to fire correctly (and only for that verb). Used for items
// that set values based on adjective checks, e.g., thesnia leaf sets ITEMVAL3=403.
func (e *GameEngine) RunItemScripts(player *Player, room *gameworld.Room, verb string, ri *gameworld.RoomItem, def *gameworld.ItemDef) *ScriptContext {
	sc := &ScriptContext{
		Player:     player,
		Room:       room,
		Engine:     e,
		ItemRef:    ri,
		ItemDef:    def,
		activeVerb: strings.ToUpper(verb),
		activeRef:  fmt.Sprintf("%d", ri.Ref),
	}
	for _, block := range def.Scripts {
		if block.Type == "IFVAR" {
			sc.execBlock(block)
		}
	}
	return sc
}

func (e *GameEngine) RunVerbScripts(player *Player, room *gameworld.Room, verb string, ri *gameworld.RoomItem, def *gameworld.ItemDef) *ScriptContext {
	verb = strings.ToUpper(verb)
	refStr := fmt.Sprintf("%d", ri.Ref)
	sc := &ScriptContext{
		Player:     player,
		Room:       room,
		Engine:     e,
		ItemRef:    ri,
		ItemDef:    def,
		activeVerb: verb,
		activeRef:  refStr,
	}

	// Check room-level IFVERB scripts (only for room items; inventory items have Ref=-1)
	if room != nil && ri.Ref >= 0 {
		// Run scripts matching this specific item ref
		for _, block := range room.Scripts {
			if block.Type == "IFVERB" && len(block.Args) >= 2 {
				if verbsMatch(strings.ToUpper(block.Args[0]), verb) && block.Args[1] == refStr {
					sc.execBlock(block)
				}
			}
		}
		// Also run room catch-all scripts (IFVERB VERB -1) with this item as context
		for _, block := range room.Scripts {
			if block.Type == "IFVERB" && len(block.Args) >= 2 {
				if verbsMatch(strings.ToUpper(block.Args[0]), verb) && block.Args[1] == "-1" {
					sc.execBlock(block)
				}
			}
		}
	}

	// Check item-level scripts (on the archetype definition)
	for _, block := range def.Scripts {
		if block.Type == "IFVERB" && len(block.Args) >= 1 {
			if verbsMatch(strings.ToUpper(block.Args[0]), verb) {
				if len(block.Args) < 2 || block.Args[1] == "-1" {
					sc.execBlock(block)
				}
			}
		}
	}

	return sc
}

// RunMonsterPreverbScript executes a monster's IFVERB/IFPREVERB -1 blocks (attached via
// SCRIPTMACRO N — see resolveMonsterMacroCalls) for a specific verb, such as EXAMINE or
// GIVE directed at the monster. item/itemDef may be nil for verbs with no associated item
// (e.g. EXAMINE); when present, the item is exposed as the "current item" (Ref -1) so
// %a and ITEMADJ/ITEMVAL checks resolve, and a script's REMOVEITEM -1 removes it from the
// player's inventory (see doRemoveItem's Ref==-1 handling).
func (e *GameEngine) RunMonsterPreverbScript(player *Player, room *gameworld.Room, monDef *gameworld.MonsterDef, verb string, item *InventoryItem, itemDef *gameworld.ItemDef) *ScriptContext {
	verb = strings.ToUpper(verb)
	sc := &ScriptContext{
		Player:      player,
		Room:        room,
		Engine:      e,
		activeVerb:  verb,
		activeRef:   "-1",
		preverbOnly: true,
	}
	if item != nil {
		sc.ItemRef = &gameworld.RoomItem{
			Ref: -1, Archetype: item.Archetype,
			Adj1: item.Adj1, Adj2: item.Adj2, Adj3: item.Adj3,
			Val1: item.Val1, Val2: item.Val2, Val3: item.Val3, Val4: item.Val4, Val5: item.Val5,
			ItemBits: item.ItemBits,
			State: item.State,
		}
		sc.ItemDef = itemDef
	}
	for _, block := range monDef.Scripts {
		if (block.Type == "IFVERB" || block.Type == "IFPREVERB") && len(block.Args) >= 2 {
			if strings.ToUpper(block.Args[0]) == verb && block.Args[1] == "-1" {
				sc.execBlock(block)
			}
		}
	}
	return sc
}

// RunRoomVerbScripts checks room-level IFVERB and IFPREVERB blocks whose item ref is -1.
// These are "bare verb" room scripts that fire for any use of the verb in the room,
// regardless of whether a specific item was targeted. Used when no item target is given,
// or after item scripts have already run (e.g., LISTEN with no args, bare GAZE).
func (e *GameEngine) RunRoomVerbScripts(player *Player, room *gameworld.Room, verb string) *ScriptContext {
	verb = strings.ToUpper(verb)
	sc := &ScriptContext{Player: player, Room: room, Engine: e, activeVerb: verb, activeRef: "-1"}
	for _, block := range room.Scripts {
		if (block.Type == "IFVERB" || block.Type == "IFPREVERB") && len(block.Args) >= 2 {
			if strings.ToUpper(block.Args[0]) == verb && block.Args[1] == "-1" {
				sc.execBlock(block)
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

	case "IFPREVERB", "IFVERB", "IFPREVERB2", "IFVERB2":
		// In preverb mode, skip IFVERB blocks entirely — they are handled by RunVerbScripts.
		if sc.preverbOnly && (block.Type == "IFVERB" || block.Type == "IFVERB2") {
			return
		}
		// These blocks only make sense within a verb-triggered context (activeVerb set by
		// RunPreverbScripts/RunVerbScripts/RunMonsterPreverbScript/RunRoomVerbScripts/
		// RunItemScripts). Non-verb walks like RunSayScripts/RunEntryScripts can still
		// descend into these nodes when they're nested inside an unrelated IFVAR tree
		// sitting next to the block that actually matched (e.g. IFSAY WINE and an
		// IFVAR-gated IFPREVERB KILL block as room-level siblings) — without this guard
		// they'd execute unconditionally instead of being ignored.
		if sc.activeVerb == "" {
			return
		}
		// Filter by verb and ref so nested IFPREVERB/IFVERB blocks inside IFVAR trees only
		// fire for the verb that actually triggered this interaction, not for every verb
		// tried against the item (LOOK and EXAMINE are treated as synonyms — scripts use
		// both keywords interchangeably for "look closely at this").
		if len(block.Args) < 1 || !verbsMatch(strings.ToUpper(block.Args[0]), sc.activeVerb) {
			return
		}
		if sc.activeRef != "" && len(block.Args) >= 2 {
			ref := block.Args[1]
			if ref != "-1" && ref != sc.activeRef {
				return
			}
		}
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
		if sc.matchesSayText(block.Args) {
			sc.execChildren(block)
		}

	case "IFTOUCH":
		// When activeVerb is set, only execute for touch-type verbs.
		if sc.activeVerb != "" {
			touchVerbs := map[string]bool{
				"TOUCH": true, "PAT": true, "FEEL": true, "RUB": true, "PET": true,
			}
			if !touchVerbs[sc.activeVerb] {
				return
			}
			if sc.activeRef != "" && len(block.Args) >= 1 {
				ref := block.Args[0]
				if ref != "-1" && ref != sc.activeRef {
					return
				}
			}
		}
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
	sc.execSteps(block.ElseBody)
}

// execChildren runs the main branch (actions and nested blocks, in original source
// order) of a script block. If a SETEVENT/CONTEVENT or PLREVENT/CONTPLREVENT pair is
// encountered, remaining work is deferred into DeferredSegments.
func (sc *ScriptContext) execChildren(block gameworld.ScriptBlock) {
	sc.execSteps(block.Body)
}

// execSteps runs a block's Body/ElseBody — actions and nested conditionals in original
// source order (e.g. "action1; IFVAR...ENDIF; action2" runs in exactly that order,
// rather than every flat action running before any nested block). If a SETEVENT/
// CONTEVENT or PLREVENT/CONTPLREVENT pair is found, the remaining steps from that point
// on are saved as a ScriptSegment for deferred execution and this returns true; callers
// must not proceed further synchronously. Returns false if all steps ran without a delay.
func (sc *ScriptContext) execSteps(steps []gameworld.ScriptStep) bool {
	for i, step := range steps {
		if step.Action != nil {
			action := *step.Action
			switch action.Command {
			case "SETEVENT":
				if len(action.Args) >= 2 {
					cycles, _ := strconv.Atoi(action.Args[1])
					sc.pendingEventDelay = cycles
					sc.pendingEventIsRaw = false
				}
				continue
			case "CONTEVENT":
				sc.DeferredSegments = append(sc.DeferredSegments, ScriptSegment{
					RelativeSeconds: sc.pendingEventDelay * scriptEventCycleSeconds,
					Steps:           steps[i+1:],
					RoomNumber:      sc.Room.Number,
				})
				return true
			case "PLREVENT":
				// PLREVENT <seconds> — player-scoped event timer; arg is raw seconds (not cycles).
				if len(action.Args) >= 1 {
					seconds, _ := strconv.Atoi(action.Args[0])
					sc.pendingEventDelay = seconds
					sc.pendingEventIsRaw = true
				}
				continue
			case "CONTPLREVENT":
				// Defer remaining steps; use raw seconds when paired with PLREVENT.
				relSecs := sc.pendingEventDelay
				if !sc.pendingEventIsRaw {
					relSecs *= scriptEventCycleSeconds
				}
				sc.pendingEventIsRaw = false
				sc.DeferredSegments = append(sc.DeferredSegments, ScriptSegment{
					RelativeSeconds: relSecs,
					Steps:           steps[i+1:],
					RoomNumber:      sc.Room.Number,
				})
				return true
			default:
				sc.execAction(action)
			}
		} else if step.Block != nil {
			// Skip nested conditional blocks once a MOVE has fired — they'd evaluate
			// against the new room instead of the original (e.g. a sibling IFVAR RNUM
			// check). Plain actions are NOT skipped: MOVE is routinely followed by
			// AFFECT <destRoom> + ECHO OTHERS in the same block to announce arrival in
			// the new room, and that idiom must keep running after MOVE fires.
			if sc.MoveTo > 0 || sc.moveGroupFired {
				continue
			}
			sc.execBlock(*step.Block)
		}
	}
	return false
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
	case "ROOMCOPY":
		sc.doRoomCopy(action.Args)
	case "SHOWROOM":
		sc.doShowRoom(action.Args)
	case "DISBAND":
		sc.doDisband()
	case "CALL":
		sc.doCallMacro(action.Args)
	case "SETEVENT":
		if len(action.Args) >= 2 {
			cycles, _ := strconv.Atoi(action.Args[1])
			sc.pendingEventDelay = cycles
		}
	case "CONTEVENT":
		// Deferred execution only works when called via execActionsUntilDelay.
		// A bare CONTEVENT outside that path is a no-op.
	case "PLREVENT", "CONTPLREVENT":
		// Legacy aliases — silently ignored.
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
	case "SETITEMADJ":
		sc.doSetItemAdj(action.Args)
	case "APPLYROLL":
		sc.doApplyRoll()
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
	case "MUL", "MULT":
		sc.doMul(action.Args)
	case "DIV":
		sc.doDiv(action.Args)
	case "MOD":
		sc.doMod(action.Args)
	case "GENMON":
		sc.doGenMon(action.Args)
	case "CALLPACK":
		sc.doCallPack()
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
	case "KILL":
		if len(action.Args) >= 1 && strings.ToUpper(action.Args[0]) == "PLAYER" {
			sc.Player.BodyPoints = 0
			sc.Player.Dead = true
			sc.KillPlayer = true
		}
	case "ROUTINE":
		sc.doRoutine(action.Args)
	case "SPELL":
		sc.doScriptSpell(action.Args)
	case "SKILLCHECK":
		sc.doSkillCheck(action.Args)
	case "APPEAR":
		if len(action.Args) >= 1 {
			text := sc.expandScriptText(strings.Join(action.Args, " "))
			sc.RoomMsgs = append(sc.RoomMsgs, text)
		}
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

	// affectRoom is true when AFFECT has switched sc.Room to a different room than the player.
	// In that case ECHO ALL / ECHO OTHERS should go directly to that room, not the player.
	affectRoom := sc.Room != nil && sc.Engine != nil && sc.Player.RoomNumber != sc.Room.Number

	// Concealed players (Invisible spell, @hide, @invis) act silently — script-driven
	// broadcasts triggered by their actions (e.g. the ECHO OTHERS "goes through the
	// hole" idiom used in place of the default move messages) must not leak to other
	// players, mirroring the concealment check already applied to hardcoded movement echoes.
	concealed := sc.Player != nil && sc.Player.IsConcealed()

	switch target {
	case "PLAYER":
		// Suppress if MOVEGROUP already fired — the player has left the source room and
		// subsequent echoes describe failure paths that no longer apply.
		if !sc.moveGroupFired {
			sc.Messages = append(sc.Messages, text)
		}
	case "ALL":
		if affectRoom {
			if !concealed && sc.Engine.roomBroadcast != nil {
				sc.Engine.roomBroadcast(sc.Room.Number, []string{text})
			}
		} else if !sc.moveGroupFired {
			sc.Messages = append(sc.Messages, text)
			if !concealed {
				sc.RoomMsgs = append(sc.RoomMsgs, text)
			}
		}
	case "OTHERS":
		if concealed {
			break
		}
		if affectRoom {
			if sc.Engine.roomBroadcast != nil {
				sc.Engine.roomBroadcast(sc.Room.Number, []string{text})
			}
		} else if !sc.moveGroupFired {
			sc.RoomMsgs = append(sc.RoomMsgs, text)
		}
	case "GROUP":
		// Send to the triggering player; group-only filtering requires
		// per-player delivery infrastructure not yet wired up.
		if !sc.moveGroupFired {
			sc.Messages = append(sc.Messages, text)
		}
	case "OTHGROUP":
		// "Others in the group" — when AFFECT has switched room context, broadcast
		// directly to that room (e.g. arrival message at destination after MOVEGROUP).
		// Otherwise treat like OTHERS in the current room.
		if concealed {
			break
		}
		if affectRoom {
			if sc.Engine.roomBroadcast != nil {
				sc.Engine.roomBroadcast(sc.Room.Number, []string{text})
			}
		} else {
			sc.RoomMsgs = append(sc.RoomMsgs, text)
		}
	}
}

// doEqual handles EQUAL <var> <value-or-var> — sets a variable.
func (sc *ScriptContext) doEqual(args []string) {
	if len(args) < 2 {
		return
	}
	sc.setVar(strings.ToUpper(args[0]), sc.resolveScriptArg(args[1]))
}

// doAdd handles ADD <var> <value-or-var> — increments a variable.
func (sc *ScriptContext) doAdd(args []string) {
	if len(args) < 2 {
		return
	}
	varName := strings.ToUpper(args[0])
	sc.setVar(varName, sc.getVar(varName)+sc.resolveScriptArg(args[1]))
}

// doSub handles SUB <var> <value-or-var> — decrements a variable.
func (sc *ScriptContext) doSub(args []string) {
	if len(args) < 2 {
		return
	}
	varName := strings.ToUpper(args[0])
	sc.setVar(varName, sc.getVar(varName)-sc.resolveScriptArg(args[1]))
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
		val := sc.resolveScriptArg(parts[1])
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
	} else if sc.Room != nil {
		ri := gameworld.RoomItem{
			Ref:       ref,
			Archetype: archetype,
			Adj1: item.Adj1, Adj2: item.Adj2, Adj3: item.Adj3,
			Val1: item.Val1, Val2: item.Val2, Val3: item.Val3, Val4: item.Val4, Val5: item.Val5,
			ItemBits: item.ItemBits,
		}
		// A NEWITEM ref can collide with an item already occupying that ref slot
		// (e.g. FULLMENE.SCR's cottage reuses ref 8, which WINOUT.SCR's base room
		// definition already uses for ground snow) — replace in place rather than
		// appending a duplicate, or a later REMOVEITEM <ref> only removes the first
		// match and strands the other (reported as a stray cottage/duplicate door
		// left behind by the Menelian's-cottage CEVENT).
		changeType := "item_add"
		replaced := false
		for i := range sc.Room.Items {
			if sc.Room.Items[i].Ref == ref && !sc.Room.Items[i].IsPut {
				sc.Room.Items[i] = ri
				changeType = "item_update"
				replaced = true
				break
			}
		}
		if !replaced {
			sc.Room.Items = append(sc.Room.Items, ri)
		}
		sc.Engine.notifyRoomChange(RoomChange{
			RoomNumber: sc.Room.Number,
			Type:       changeType,
			ItemRef:    ref,
			Item:       &ri,
		})
	}
}

// doDisband removes the player from their current group (used in scripted portals like the carriage).
func (sc *ScriptContext) doDisband() {
	if sc.Player.IsGroupLeader && len(sc.Player.GroupMembers) > 0 {
		sc.Player.GroupMembers = nil
		sc.Player.IsGroupLeader = false
	} else if sc.Player.Following != "" {
		sc.Player.Following = ""
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
		if sc.OrigRoomNum == 0 {
			sc.OrigRoomNum = sc.Player.RoomNumber
		}
		// Snapshot RoomMsgs accumulated before this MOVE so callers can send them to the old room.
		if sc.PreMoveMsgs == nil {
			sc.PreMoveMsgs = sc.RoomMsgs
			sc.RoomMsgs = nil
		}
		sc.MoveTo = dest
		sc.Player.RoomNumber = dest
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
		// Mark that the group has left the source room. Subsequent ECHO PLAYER/OTHERS in
		// the same script block are suppressed — they describe failure paths that no longer
		// apply to players who have been moved.
		sc.moveGroupFired = true
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

// doSetItemVal handles SETITEMVAL ref valIndex value — sets a Val field on a room item.
// syncItemRefToPlayerItem writes sc.ItemRef's current Adj1-3/Val1-5 back onto the
// matching player item (Worn/Inventory/Wielded/OffHand, matched by archetype). Needed
// after SETITEMADJ/SETITEMVAL -1 mutate the temporary ItemRef built for a -1 ("the item
// just interacted with") reference — without this the change never reaches the actual
// inventory item and is lost.
func (sc *ScriptContext) syncItemRefToPlayerItem() {
	if sc.ItemRef == nil {
		return
	}
	arch := sc.ItemRef.Archetype
	apply := func(item *InventoryItem) {
		item.Adj1, item.Adj2, item.Adj3 = sc.ItemRef.Adj1, sc.ItemRef.Adj2, sc.ItemRef.Adj3
		item.Val1, item.Val2, item.Val3, item.Val4, item.Val5 =
			sc.ItemRef.Val1, sc.ItemRef.Val2, sc.ItemRef.Val3, sc.ItemRef.Val4, sc.ItemRef.Val5
	}
	for i := range sc.Player.Worn {
		if sc.Player.Worn[i].Archetype == arch {
			apply(&sc.Player.Worn[i])
			sc.NeedsSave = true
			return
		}
	}
	for i := range sc.Player.Inventory {
		if sc.Player.Inventory[i].Archetype == arch {
			apply(&sc.Player.Inventory[i])
			sc.NeedsSave = true
			return
		}
	}
	if sc.Player.Wielded != nil && sc.Player.Wielded.Archetype == arch {
		apply(sc.Player.Wielded)
		sc.NeedsSave = true
		return
	}
	if sc.Player.OffHand != nil && sc.Player.OffHand.Archetype == arch {
		apply(sc.Player.OffHand)
		sc.NeedsSave = true
	}
}

// doApplyRoll handles APPLYROLL — applies the seven core stats (STR/AGI/QUI/CON/
// PER/WIL/EMP) previewed by the ROLLSTR/ROLLAGI/.../ROLLEMP pseudo-variables (see
// getVar) to the player. Those pseudo-variables derive deterministically from the
// current item's ItemVal1 (a seed rolled by RANDOM ITEMVAL1 ... and re-rollable by
// the player any number of times before committing), so APPLYROLL always applies
// exactly what the player last saw previewed. A no-op if the item has no seed yet
// (ItemVal1 == 0, i.e. never RUBbed). Level, skills, resource pool maximums (BP/
// fatigue/mana/psi), and everything else are left untouched.
func (sc *ScriptContext) doApplyRoll() {
	if sc.ItemRef == nil || sc.ItemRef.Val1 == 0 {
		return
	}
	str, agi, qui, con, per, wil, emp := RollStatsSeeded(int64(sc.ItemRef.Val1), sc.Player.Race)
	sc.Player.Strength = str
	sc.Player.Agility = agi
	sc.Player.Quickness = qui
	sc.Player.Constitution = con
	sc.Player.Perception = per
	sc.Player.Willpower = wil
	sc.Player.Empathy = emp
	sc.NeedsSave = true
}

func (sc *ScriptContext) doSetItemVal(args []string) {
	if len(args) < 3 {
		return
	}
	ref, err := strconv.Atoi(args[0])
	if err != nil {
		return
	}
	valIdx, err := strconv.Atoi(args[1])
	if err != nil {
		return
	}
	val := sc.resolveScriptArg(args[2])
	if ref == -1 {
		if sc.ItemRef == nil {
			return
		}
		switch valIdx {
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
		sc.syncItemRefToPlayerItem()
		return
	}
	if sc.Room == nil {
		return
	}
	for i := len(sc.Room.Items) - 1; i >= 0; i-- {
		if sc.Room.Items[i].Ref == ref {
			switch valIdx {
			case 1:
				sc.Room.Items[i].Val1 = val
			case 2:
				sc.Room.Items[i].Val2 = val
			case 3:
				sc.Room.Items[i].Val3 = val
			case 4:
				sc.Room.Items[i].Val4 = val
			case 5:
				sc.Room.Items[i].Val5 = val
			}
			itemCopy := sc.Room.Items[i]
			sc.Engine.notifyRoomChange(RoomChange{
				RoomNumber: sc.Room.Number,
				Type:       "item_update",
				ItemRef:    ref,
				Item:       &itemCopy,
			})
			return
		}
	}
}

// doSetItemAdj handles SETITEMADJ ref adjIndex value — sets an adjective on a room item.
// The value arg can be a variable name (e.g., ITEMADJ1) or a literal integer.
func (sc *ScriptContext) doSetItemAdj(args []string) {
	if len(args) < 3 {
		return
	}
	ref, err := strconv.Atoi(args[0])
	if err != nil {
		return
	}
	adjIdx, err := strconv.Atoi(args[1])
	if err != nil {
		return
	}
	val := sc.resolveScriptArg(args[2])
	if ref == -1 {
		if sc.ItemRef == nil {
			return
		}
		switch adjIdx {
		case 1:
			sc.ItemRef.Adj1 = val
		case 2:
			sc.ItemRef.Adj2 = val
		case 3:
			sc.ItemRef.Adj3 = val
		}
		sc.syncItemRefToPlayerItem()
		return
	}
	if sc.Room == nil {
		return
	}
	// Search from the end so NEWITEM-created items (appended last) take priority
	// over any pre-existing room items that happen to share the same ref slot.
	for i := len(sc.Room.Items) - 1; i >= 0; i-- {
		if sc.Room.Items[i].Ref == ref {
			switch adjIdx {
			case 1:
				sc.Room.Items[i].Adj1 = val
			case 2:
				sc.Room.Items[i].Adj2 = val
			case 3:
				sc.Room.Items[i].Adj3 = val
			}
			itemCopy := sc.Room.Items[i]
			sc.Engine.notifyRoomChange(RoomChange{
				RoomNumber: sc.Room.Number,
				Type:       "item_update",
				ItemRef:    ref,
				Item:       &itemCopy,
			})
			return
		}
	}
}

// doRemoveItem handles REMOVEITEM ref — removes item from the room or player inventory.
func (sc *ScriptContext) doRemoveItem(args []string) {
	if len(args) == 0 {
		return
	}
	ref, err := strconv.Atoi(args[0])
	if err != nil {
		return
	}
	if ref >= 0 && sc.Room != nil {
		// Remove the item with the given ref slot from the current room.
		for i, ri := range sc.Room.Items {
			if ri.Ref == ref {
				sc.Room.Items = append(sc.Room.Items[:i], sc.Room.Items[i+1:]...)
				sc.Engine.notifyRoomChange(RoomChange{
					RoomNumber: sc.Room.Number,
					Type:       "item_remove",
					ItemRef:    ref,
				})
				return
			}
		}
		return
	}
	if ref == -1 && sc.ItemRef != nil {
		// If the item is a room item (Ref >= 0), remove it from the room
		if sc.Room != nil && sc.ItemRef.Ref >= 0 {
			for i, ri := range sc.Room.Items {
				if ri.Ref == sc.ItemRef.Ref {
					sc.Room.Items = append(sc.Room.Items[:i], sc.Room.Items[i+1:]...)
					sc.Engine.notifyRoomChange(RoomChange{
						RoomNumber: sc.Room.Number,
						Type:       "item_remove",
						ItemRef:    sc.ItemRef.Ref,
					})
					return
				}
			}
		}
		// Otherwise remove from player inventory by archetype match. Worn items live in
		// their own slice (see doWear) and must be checked too, or REMOVEITEM -1 silently
		// no-ops after an IFCARRY match on something the player has equipped rather than
		// merely carried.
		for i, ii := range sc.Player.Inventory {
			if ii.Archetype == sc.ItemRef.Archetype {
				sc.Player.Inventory = append(sc.Player.Inventory[:i], sc.Player.Inventory[i+1:]...)
				sc.NeedsSave = true
				return
			}
		}
		for i, ii := range sc.Player.Worn {
			if ii.Archetype == sc.ItemRef.Archetype {
				sc.Player.Worn = append(sc.Player.Worn[:i], sc.Player.Worn[i+1:]...)
				sc.NeedsSave = true
				return
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

// evalIfVar evaluates IFVAR conditions like "INTNUM6 = 2" or "DUMMY3 = FLAG3".
// The right-hand side may be a literal integer or a variable name.
// ORG membership checks (= and !) are handled specially to support multiple orgs.
func (sc *ScriptContext) evalIfVar(args []string) bool {
	if len(args) < 3 {
		return false
	}
	varName := strings.ToUpper(args[0])
	op := args[1]

	// Special-case ORG equality/inequality: check OrgMemberships rather than a single int.
	if varName == "ORG" && (op == "=" || op == "!") {
		orgNum := sc.resolveScriptArg(args[2])
		sc.lastOrgChecked = orgNum
		isMember := sc.Player.IsMemberOf(orgNum)
		if op == "=" {
			return isMember
		}
		return !isMember
	}

	actual := sc.getVar(varName)
	expected := sc.resolveScriptArg(args[2])

	switch op {
	case "=", "==":
		return actual == expected
	case "!":
		return actual != expected
	case ">":
		return actual > expected
	case "<":
		return actual < expected
	case ">=", "=>":
		return actual >= expected
	case "<=", "=<":
		return actual <= expected
	}
	return false
}

// evalIfItem evaluates IFITEM conditions like "IFITEM 0 OPEN" or "IFITEM -1 CLOSED".
// With only a ref arg and no state, it checks existence only.
func (sc *ScriptContext) evalIfItem(args []string) bool {
	if len(args) < 1 {
		return false
	}
	// Handle both "IFITEM -1 WORN" and "IFITEM WORN -1" orderings.
	refStr := args[0]
	var stateStr string
	if len(args) >= 2 {
		stateStr = strings.ToUpper(args[1])
	}
	if _, err := strconv.Atoi(refStr); err != nil {
		refStr, stateStr = stateStr, strings.ToUpper(args[0])
	}
	ref, err := strconv.Atoi(refStr)
	if err != nil {
		return false
	}
	// 1-arg form: existence check only.
	if stateStr == "" {
		if ref == -1 && sc.ItemRef != nil {
			return true
		}
		if sc.Room != nil {
			for _, ri := range sc.Room.Items {
				if ri.Ref == ref {
					return true
				}
			}
		}
		return false
	}
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
	switch stateStr {
	case "OPEN":
		return state == "OPEN"
	case "CLOSED":
		return state == "CLOSED" || state == ""
	case "LOCKED":
		return state == "LOCKED"
	case "UNLOCKED":
		return state == "UNLOCKED" || state == "OPEN"
	case "WORN":
		return state == "WORN"
	case "WIELDED":
		return state == "WIELDED"
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
		if sc.ItemRef.ItemBits&(1<<idx) != 0 {
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

	// ROLLSTR/ROLLAGI/ROLLQUI/ROLLCON/ROLLPER/ROLLWIL/ROLLEMP — a preview reroll of
	// the player's seven core stats, deterministically derived from the current
	// item's ItemVal1 (a seed) and the player's race. Used by the reroll charm
	// (modern_fixes.scr) so RUB can show a new preview and EXAMINE can redisplay the
	// same preview later, without persisting all seven stats on the item — only the
	// seed. 0 if there's no current item or no seed rolled yet (ItemVal1 == 0).
	switch name {
	case "ROLLSTR", "ROLLAGI", "ROLLQUI", "ROLLCON", "ROLLPER", "ROLLWIL", "ROLLEMP":
		if sc.ItemRef == nil || sc.ItemRef.Val1 == 0 {
			return 0
		}
		str, agi, qui, con, per, wil, emp := RollStatsSeeded(int64(sc.ItemRef.Val1), sc.Player.Race)
		switch name {
		case "ROLLSTR":
			return str
		case "ROLLAGI":
			return agi
		case "ROLLQUI":
			return qui
		case "ROLLCON":
			return con
		case "ROLLPER":
			return per
		case "ROLLWIL":
			return wil
		case "ROLLEMP":
			return emp
		}
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
		// Returns primary org for non-equality comparisons (range checks etc.).
		// Equality/inequality is handled directly in evalIfVar before getVar is called.
		return sc.Player.Organization
	case "ORGRANK":
		// Return rank for the org last evaluated in an IFVAR ORG check.
		if sc.lastOrgChecked != 0 {
			return sc.Player.RankIn(sc.lastOrgChecked)
		}
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
		if sc.ItemRef != nil { return sc.ItemRef.Archetype }
		if sc.Player.Wielded != nil { return sc.Player.Wielded.Archetype }
		return 0
	// Room info
	case "RNUM":
		return sc.Player.RoomNumber
	case "OUTDOOR":
		if sc.Room != nil && isOutdoorTerrain(sc.Room.Terrain) { return 1 }
		return 0
	case "PLRSINROOM":
		roomNum := sc.Player.RoomNumber
		if sc.Room != nil { roomNum = sc.Room.Number }
		if sc.Engine.sessions != nil {
			count := 0
			for _, p := range sc.Engine.sessions.OnlinePlayers() {
				if p.RoomNumber == roomNum { count++ }
			}
			return count
		}
		return 1
	case "MONINROOM":
		roomNum := sc.Player.RoomNumber
		if sc.Room != nil { roomNum = sc.Room.Number }
		if sc.Engine.monsterMgr != nil {
			return len(sc.Engine.monsterMgr.MonstersInRoom(roomNum))
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
			region := 0
			if sc.Room != nil {
				region = sc.Room.Region
			} else if sc.Player != nil {
				if r := sc.Engine.rooms[sc.Player.RoomNumber]; r != nil {
					region = r.Region
				}
			}
			return sc.Engine.RegionWeather[region]
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
		if sc.Player.RoundTimeExpiry.After(time.Now()) {
			return sc.Player.RoundTime
		}
		return 0
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
		if sc.ItemDef != nil {
			w := sc.ItemDef.Weight
			// For containers, include the weight of everything placed inside — e.g. a
			// stream's OBJWEIGHT should grow as boulders are PUT into it so scripts can
			// track dam progress. Static def.Weight alone never changes.
			if sc.Room != nil && sc.ItemRef != nil && sc.ItemRef.Ref >= 0 && isContainerDef(sc.ItemDef) {
				w += sc.Engine.roomContainerContentsWeight(sc.Room, sc.ItemRef.Ref)
			}
			return w
		}
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
		sc.NeedsSave = true
		return
	}
	if strings.HasPrefix(name, "ITEMBIT") {
		idx, err := strconv.Atoi(name[7:])
		if err != nil || sc.ItemRef == nil {
			return
		}
		if val != 0 {
			sc.ItemRef.ItemBits |= 1 << idx
		} else {
			sc.ItemRef.ItemBits &^= 1 << idx
		}
		// For inventory/worn items (Ref=-1), sync the updated bits back to the player's
		// actual item so that SavePlayer persists them correctly (mirrors ITEMVAL below).
		if sc.ItemRef.Ref == -1 {
			arch := sc.ItemRef.Archetype
			bits := sc.ItemRef.ItemBits
			for i := range sc.Player.Worn {
				if sc.Player.Worn[i].Archetype == arch {
					sc.Player.Worn[i].ItemBits = bits
					sc.NeedsSave = true
					return
				}
			}
			for i := range sc.Player.Inventory {
				if sc.Player.Inventory[i].Archetype == arch {
					sc.Player.Inventory[i].ItemBits = bits
					sc.NeedsSave = true
					return
				}
			}
			if sc.Player.Wielded != nil && sc.Player.Wielded.Archetype == arch {
				sc.Player.Wielded.ItemBits = bits
				sc.NeedsSave = true
				return
			}
			if sc.Player.OffHand != nil && sc.Player.OffHand.Archetype == arch {
				sc.Player.OffHand.ItemBits = bits
				sc.NeedsSave = true
				return
			}
			sc.NeedsSave = true
			return
		}
		itemCopy := *sc.ItemRef
		sc.Engine.notifyRoomChange(RoomChange{
			RoomNumber: sc.Room.Number, Type: "item_update",
			ItemRef: sc.ItemRef.Ref, Item: &itemCopy,
		})
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
		// For inventory/worn items (Ref=-1), sync the updated vals back to the player's
		// actual item so that SavePlayer persists them correctly.
		if sc.ItemRef.Ref == -1 {
			arch := sc.ItemRef.Archetype
			syncVals := func(item *InventoryItem) {
				item.Val1 = sc.ItemRef.Val1
				item.Val2 = sc.ItemRef.Val2
				item.Val3 = sc.ItemRef.Val3
				item.Val4 = sc.ItemRef.Val4
				item.Val5 = sc.ItemRef.Val5
			}
			for i := range sc.Player.Worn {
				if sc.Player.Worn[i].Archetype == arch {
					syncVals(&sc.Player.Worn[i])
					break
				}
			}
			for i := range sc.Player.Inventory {
				if sc.Player.Inventory[i].Archetype == arch {
					syncVals(&sc.Player.Inventory[i])
					break
				}
			}
			if sc.Player.Wielded != nil && sc.Player.Wielded.Archetype == arch {
				syncVals(sc.Player.Wielded)
			}
			if sc.Player.OffHand != nil && sc.Player.OffHand.Archetype == arch {
				syncVals(sc.Player.OffHand)
			}
			sc.NeedsSave = true
			return
		}
		itemCopy := *sc.ItemRef
		sc.Engine.notifyRoomChange(RoomChange{
			RoomNumber: sc.Room.Number, Type: "item_update",
			ItemRef: sc.ItemRef.Ref, Item: &itemCopy,
		})
		return
	}
	if strings.HasPrefix(name, "ITEMADJ") {
		idx, err := strconv.Atoi(name[7:])
		if err != nil || sc.ItemRef == nil {
			return
		}
		switch idx {
		case 1:
			sc.ItemRef.Adj1 = val
		case 2:
			sc.ItemRef.Adj2 = val
		case 3:
			sc.ItemRef.Adj3 = val
		}
		if sc.ItemRef.Ref == -1 {
			sc.syncItemRefToPlayerItem()
			return
		}
		itemCopy2 := *sc.ItemRef
		sc.Engine.notifyRoomChange(RoomChange{
			RoomNumber: sc.Room.Number, Type: "item_update",
			ItemRef: sc.ItemRef.Ref, Item: &itemCopy2,
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
		sc.NeedsSave = true
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
		if val == 0 && sc.lastOrgChecked != 0 {
			sc.Player.RemoveOrg(sc.lastOrgChecked)
		} else if val != 0 {
			rank := sc.Player.RankIn(val)
			if rank == 0 {
				rank = 1
			}
			sc.Player.AddOrg(val, rank)
		}
	case "ORGRANK":
		if sc.lastOrgChecked != 0 {
			sc.Player.AddOrg(sc.lastOrgChecked, val)
		} else {
			sc.Player.OrgRank = val
		}
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
	case "ROUNDTIME":
		sc.Player.RoundTime = val
		if val > 0 {
			sc.Player.RoundTimeExpiry = time.Now().Add(time.Duration(val) * time.Second)
			sc.RoundTimeSet = val
			sc.NeedsSave = true
		}
	case "WEALTH":
		// WEALTH is in copper units — split into gold/silver/copper
		sc.Player.Gold = val / 100
		sc.Player.Silver = (val % 100) / 10
		sc.Player.Copper = val % 10
		sc.NeedsSave = true
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
// or any variable reference (ITEMVAL2, CARRIAGEROOM, DUMMY1, etc.).
func (sc *ScriptContext) resolveNumericArg(arg string) int {
	upper := strings.ToUpper(arg)
	val, err := strconv.Atoi(arg)
	if err != nil {
		return sc.getVar(upper)
	}
	return val
}

// resolveScriptArg resolves an argument that may be a literal integer or any
// named variable (DUMMY1, FLAG2, DANBET, ITEMVAL1, PVAL10, etc.).
func (sc *ScriptContext) resolveScriptArg(arg string) int {
	val, err := strconv.Atoi(arg)
	if err == nil {
		return val
	}
	return sc.getVar(strings.ToUpper(arg))
}

// expandScriptText replaces script placeholders in text.
// evalIfCarry checks if player carries an item with matching archetype (and optional adj).
func (sc *ScriptContext) evalIfCarry(args []string) bool {
	if len(args) < 1 {
		return false
	}
	adj := -1
	if len(args) >= 2 {
		adj = sc.resolveNumericArg(args[1])
	}
	// IFCARRY WIELDED <adj> is a special form referring to the player's currently
	// wielded weapon rather than a search by archetype number (WIELDED isn't a
	// literal archetype ID). Match it against the wielded item and expose it as
	// the "current item" so nested ITEMADJ1/ITEMADJ2/etc. checks resolve correctly.
	if strings.EqualFold(args[0], "WIELDED") {
		w := sc.Player.Wielded
		if w == nil || (adj >= 0 && w.Adj1 != adj) {
			return false
		}
		sc.ItemRef = &gameworld.RoomItem{
			Ref:       -1,
			Archetype: w.Archetype,
			Adj1:      w.Adj1,
			Adj2:      w.Adj2,
			Adj3:      w.Adj3,
			Val1:      w.Val1,
			Val2:      w.Val2,
			Val3:      w.Val3,
			Val4:      w.Val4,
			Val5:      w.Val5,
			ItemBits:  w.ItemBits,
			State:     w.State,
		}
		return true
	}
	// Archetype arg may be a literal number or a named variable (e.g. IFCARRY INTNUM83
	// INTNUM84, used to check for whatever quest item a prior CALL/RANDOM chose).
	archetype := sc.resolveNumericArg(args[0])
	// Per GMSCRIPT.DOC, IFCARRY matches anything "in the current player's inventory" —
	// in the original engine that included worn items. This engine splits worn gear into
	// its own Player.Worn slice (see doWear), so both must be searched or IFCARRY wrongly
	// reports false for a worn item (e.g. a ring checked for guild-passage access).
	for _, inv := range [][]InventoryItem{sc.Player.Inventory, sc.Player.Worn} {
		for _, ii := range inv {
			if ii.Archetype == archetype {
				// Check all three adjective slots, not just Adj1 — store-bought "variety"
				// items (e.g. STOREITEM's adjective) are placed in Adj3, not Adj1 (see
				// doBuy), and crafted/found items vary in which slot holds a given adj.
				if adj < 0 || ii.Adj1 == adj || ii.Adj2 == adj || ii.Adj3 == adj {
					// Per GMSCRIPT.DOC: "If it is found, the current item is set to the
					// first such item located." Needed for a following REMOVEITEM -1,
					// %a, or ITEMADJ*/ITEMVAL* to act on the matched item.
					sc.ItemRef = &gameworld.RoomItem{
						Ref: -1, Archetype: ii.Archetype,
						Adj1: ii.Adj1, Adj2: ii.Adj2, Adj3: ii.Adj3,
						Val1: ii.Val1, Val2: ii.Val2, Val3: ii.Val3, Val4: ii.Val4, Val5: ii.Val5,
						ItemBits: ii.ItemBits,
						State: ii.State,
					}
					if sc.Engine != nil {
						sc.ItemDef = sc.Engine.items[ii.Archetype]
					}
					return true
				}
			}
		}
	}
	return false
}

// doCallMacro handles CALL N as an inline, subroutine-style action: it runs macro N's
// script blocks immediately within the current ScriptContext, then execution continues
// with whatever follows the CALL in the calling script. This is distinct from a room/
// item's top-level CALL N (or a monster's SCRIPTMACRO N), which are resolved once at
// parse time into that room/item/monster's own Scripts (see resolveRoomMacroCalls etc.)
// rather than invoked live mid-script.
func (sc *ScriptContext) doCallMacro(args []string) {
	if len(args) < 1 || sc.Engine == nil {
		return
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return
	}
	for _, block := range sc.Engine.macros[id] {
		if block.Type == "ACTION" {
			for _, action := range block.Actions {
				sc.execAction(action)
			}
		} else {
			sc.execBlock(block)
		}
	}
}

// doRoomCopy copies the exits and description from a template room into sc.Room.
// Used by CEVENT geyser scripts to toggle which exits are open (ROOMCOPY <template>).
func (sc *ScriptContext) doRoomCopy(args []string) {
	if len(args) == 0 || sc.Room == nil || sc.Engine == nil {
		return
	}
	templateNum := sc.resolveNumericArg(args[0])
	if templateNum <= 0 {
		return
	}
	template := sc.Engine.rooms[templateNum]
	if template == nil {
		return
	}
	newExits := make(map[string]int, len(template.Exits))
	for dir, dest := range template.Exits {
		newExits[dir] = dest
	}
	sc.Room.Exits = newExits
	sc.Room.Description = template.Description
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
// Syntax: RANDOM <var> <min> <max> [seed]
// Produces a uniformly random integer in [min, max] inclusive.
func (sc *ScriptContext) doRandom(args []string) {
	if len(args) < 3 {
		return
	}
	varName := strings.ToUpper(args[0])
	min, err := strconv.Atoi(args[1])
	if err != nil {
		return
	}
	max, err := strconv.Atoi(args[2])
	if err != nil || max < min {
		return
	}
	sc.setVar(varName, min+rand.Intn(max-min+1))
}

// doSkillCheck performs a skill-based check and stores the result in a variable.
// Syntax: SKILLCHECK <skillID> <modifier> <resultVar>
// Stores a positive value on success, negative on failure.
func (sc *ScriptContext) doSkillCheck(args []string) {
	if len(args) < 3 {
		return
	}
	skillID, err := strconv.Atoi(args[0])
	if err != nil {
		return
	}
	modifier, _ := strconv.Atoi(args[1])
	resultVar := strings.ToUpper(args[2])

	skillLevel := 0
	if sc.Player.Skills != nil {
		skillLevel = sc.Player.Skills[skillID]
	}

	// Base 50% success; each skill rank adds 5%; modifier subtracts from target.
	// Clamped to [5%, 95%] so there is always some uncertainty.
	target := 50 + skillLevel*5 - modifier
	if target > 95 {
		target = 95
	} else if target < 5 {
		target = 5
	}
	roll := rand.Intn(100) + 1
	sc.setVar(resultVar, target-roll) // positive = success (roll under target)
}

// doDamagePlr applies damage to the player.
func (sc *ScriptContext) doDamagePlr(args []string) {
	if len(args) < 2 {
		return
	}
	// DAMAGEPLR [type] <amount> <text...>
	// type is optional and can be BODYONLY, CRUSH, SLASH, FIRE, etc.
	// Skip the first arg if it is non-numeric (i.e. a type keyword).
	idx := 0
	if _, err := strconv.Atoi(args[0]); err != nil {
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
	// Join remaining args and strip quotes. Underscores stand in for spaces, the same
	// convention IFSAY patterns use, since a script argument can't itself contain a
	// literal space (e.g. STRCPY 0 "some_meteoric_dust" -> "some meteoric dust").
	text := strings.Join(args[1:], " ")
	text = strings.Trim(text, "\"")
	text = strings.ReplaceAll(text, "_", " ")
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
	val := sc.resolveScriptArg(args[1])
	sc.setVar(varName, sc.getVar(varName)*val)
}

// doDiv handles DIV varName value — divides a variable.
func (sc *ScriptContext) doDiv(args []string) {
	if len(args) < 2 {
		return
	}
	varName := strings.ToUpper(args[0])
	val := sc.resolveScriptArg(args[1])
	if val == 0 {
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
	val := sc.resolveScriptArg(args[1])
	if val == 0 {
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
		sc.Engine.monsterMgr.SpawnOne(monNum, sc.Room.Number, def.Body, def.Mana)
		name := FormatMonsterName(def, sc.Engine.monAdjs)
		genText := def.TextOverrides["TEXG"]
		if genText == "" {
			genText = fmt.Sprintf("A %s appears!", name)
		}
		sc.RoomMsgs = append(sc.RoomMsgs, genText)
		sc.Engine.Events.Publish("monster", fmt.Sprintf("GENMON: %s spawned in room %d", name, sc.Room.Number))
	}
}

// doCallPack triggers the Call the Pack effect (see castCallThePack) in the script's
// room — usable from any item, room, or monster script (the "object or NPC" cast paths;
// GM casting goes through the separate @callpack command instead).
func (sc *ScriptContext) doCallPack() {
	sc.Engine.castCallThePack(sc.Room)
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

// doRoutine executes a built-in item magic routine.
// Both ROUTINE 1 and ROUTINE 2 read the spell from Val3; charges are tracked in Val2.
// Val2 == 0 means no charges remain. Charge decrements are persisted to the player's worn/inventory.
// ROUTINE 1 auto-casts the spell on the player; ROUTINE 2 preps it so the player can CAST it.
func (sc *ScriptContext) doRoutine(args []string) {
	if len(args) < 1 || sc.ItemRef == nil {
		return
	}
	routineNum, err := strconv.Atoi(args[0])
	if err != nil {
		return
	}
	if routineNum != 1 && routineNum != 2 {
		return
	}

	spellID := sc.ItemRef.Val3
	if spellID == 0 {
		sc.Messages = append(sc.Messages, "The item holds no magical energy.")
		return
	}

	spell := FindSpellByID(spellID)
	if spell == nil {
		sc.Messages = append(sc.Messages, "The item's magic is beyond your comprehension.")
		return
	}

	itemNoun := "item"
	if sc.ItemDef != nil && sc.Engine != nil {
		itemNoun = sc.Engine.getItemNounName(sc.ItemDef)
	}

	// Val2 == 0 means exhausted; Val2 > 0 means charges remain.
	if sc.ItemRef.Val2 == 0 {
		sc.Messages = append(sc.Messages, fmt.Sprintf("The %s holds no more magical power.", itemNoun))
		return
	}

	if !sc.routineChargePaid {
		newVal2 := sc.ItemRef.Val2 - 1
		sc.ItemRef.Val2 = newVal2
		sc.routineChargePaid = true
		// Persist charge change to the actual player worn/inventory item.
		arch := sc.ItemRef.Archetype
		for i := range sc.Player.Worn {
			if sc.Player.Worn[i].Archetype == arch && sc.Player.Worn[i].Val3 == spellID {
				sc.Player.Worn[i].Val2 = newVal2
				break
			}
		}
		for i := range sc.Player.Inventory {
			if sc.Player.Inventory[i].Archetype == arch && sc.Player.Inventory[i].Val3 == spellID {
				sc.Player.Inventory[i].Val2 = newVal2
				break
			}
		}
		if newVal2 == 0 {
			sc.Messages = append(sc.Messages, fmt.Sprintf("The %s flickers as its last charge is spent.", itemNoun))
		} else {
			suffix := "s"
			if newVal2 == 1 {
				suffix = ""
			}
			sc.Messages = append(sc.Messages, fmt.Sprintf("(%d charge%s remaining)", newVal2, suffix))
		}
	}

	switch routineNum {
	case 1:
		sc.applyItemSpellOnPlayer(spell)
	case 2:
		sc.Player.PreparedSpell = spellID
		sc.Player.PreparedSpellReagentArch = 0
		sc.Player.PreparedMoonstoneBonus = false
		sc.Messages = append(sc.Messages, fmt.Sprintf("The %s glows briefly. %s is prepared for casting. (CAST to release it.)", itemNoun, spell.Name))
		sc.RoomMsgs = append(sc.RoomMsgs, fmt.Sprintf("The %s in %s's hands glows briefly.", itemNoun, sc.Player.FirstName))
	}
	sc.NeedsSave = true
}

// doScriptSpell handles SPELL <id> [<chance>] — casts a spell on the player from a script.
func (sc *ScriptContext) doScriptSpell(args []string) {
	if len(args) < 1 {
		return
	}
	spellID, err := strconv.Atoi(args[0])
	if err != nil {
		return
	}
	chance := 100
	if len(args) >= 2 {
		if c, err2 := strconv.Atoi(args[1]); err2 == nil {
			chance = c
		}
	}
	if chance < 100 && rand.Intn(100) >= chance {
		return
	}
	spell := FindSpellByID(spellID)
	if spell == nil {
		return
	}
	sc.applyItemSpellOnPlayer(spell)
	sc.NeedsSave = true
}

// applyItemSpellOnPlayer applies a spell's effect on the player for item-triggered casts.
// Unlike casting a spell normally, no mana cost or skill check applies — the item does the work.
func (sc *ScriptContext) applyItemSpellOnPlayer(spell *SpellDef) {
	player := sc.Player
	switch spell.Effect {
	case "heal":
		healMin, healMax := spell.HealMin, spell.HealMax
		if healMax <= healMin {
			healMax = healMin + 1
		}
		amount := healMin + rand.Intn(healMax-healMin+1)
		// Invigoration spells restore fatigue
		if spell.ID == 334 || spell.ID == 335 {
			player.Fatigue += amount
			if player.Fatigue > player.MaxFatigue {
				player.Fatigue = player.MaxFatigue
			}
			sc.Messages = append(sc.Messages, fmt.Sprintf("The item channels %s, invigorating you! [Fatigue: %d/%d]", spell.Name, player.Fatigue, player.MaxFatigue))
		} else {
			player.BodyPoints += amount
			if player.BodyPoints > player.MaxBodyPoints {
				player.BodyPoints = player.MaxBodyPoints
			}
			sc.Messages = append(sc.Messages, fmt.Sprintf("The item channels %s, healing you for %d points. [BP: %d/%d]", spell.Name, amount, player.BodyPoints, player.MaxBodyPoints))
		}
		sc.RoomMsgs = append(sc.RoomMsgs, fmt.Sprintf("%s is bathed in healing energy.", player.FirstName))

	case "defense":
		mins, _ := applyTimedDefenseBuff(player, spell)
		sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s! (+%d defense, %d minutes)", spell.Name, spell.DefBonus, mins))
		sc.RoomMsgs = append(sc.RoomMsgs, fmt.Sprintf("A shimmer of protective energy surrounds %s.", player.FirstName))

	case "damage":
		dmgMin, dmgMax := spell.DmgMin, spell.DmgMax
		if dmgMax <= dmgMin {
			dmgMax = dmgMin + 1
		}
		amount := dmgMin + rand.Intn(dmgMax-dmgMin+1)
		player.BodyPoints -= amount
		rawBP := player.BodyPoints
		if player.BodyPoints < 0 {
			player.BodyPoints = 0
		}
		sc.Messages = append(sc.Messages, fmt.Sprintf("The item channels %s! Searing agony wracks your body. [-%d BP]", spell.Name, amount))
		sc.RoomMsgs = append(sc.RoomMsgs, fmt.Sprintf("%s cries out in pain!", player.FirstName))
		if rawBP <= 0 {
			outcomeMsgs, _ := sc.Engine.resolveDirectHitOutcome(player, rawBP, "a cursed potion")
			sc.Messages = append(sc.Messages, outcomeMsgs...)
		}

	case "buff":
		switch spell.ID {
		case 102: // Mystic Armor
			mins, _ := applyMysticArmorBuff(player, spell)
			sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s! (+%d defense, %d minutes)", spell.Name, spell.DefBonus, mins))
			sc.RoomMsgs = append(sc.RoomMsgs, fmt.Sprintf("A shimmering barrier of energy surrounds %s.", player.FirstName))
		case 207, 208, 209: // Strength I/II/III
			bonus, mins, applied, ok := applyStrengthBuff(player, spell)
			if !ok {
				sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s, but you already have a better Strength spell in place.", spell.Name))
			} else if !applied {
				sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. Your strength pulsates with renewed energy! (%d minutes remaining)", spell.Name, mins))
			} else {
				sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s! (+%d STR, %d minutes)", spell.Name, bonus, mins))
				sc.RoomMsgs = append(sc.RoomMsgs, fmt.Sprintf("%s glows with newfound strength.", player.FirstName))
			}
		case 210: // Haste
			player.HasteExpiry = time.Now().Add(20 * time.Minute)
			sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. The world slows around you!", spell.Name))
			sc.RoomMsgs = append(sc.RoomMsgs, fmt.Sprintf("%s moves with incredible speed.", player.FirstName))
		case 224: // Fly
			player.CanFly = true
			player.FlyExpiry = time.Now().Add(20 * time.Minute)
			sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. You rise into the air!", spell.Name))
			sc.RoomMsgs = append(sc.RoomMsgs, fmt.Sprintf("%s rises into the air!", player.FirstName))
		case 225: // Invisibility
			player.Invisible = true
			sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. You fade from sight.", spell.Name))
			sc.RoomMsgs = append(sc.RoomMsgs, fmt.Sprintf("%s fades from sight.", player.FirstName))
		case 347: // Divine Blessing
			sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. You feel divinely blessed.", spell.Name))
			sc.RoomMsgs = append(sc.RoomMsgs, fmt.Sprintf("A warm golden light briefly surrounds %s.", player.FirstName))
		case 506: // Resist Weather
			mins, applied := applyElementalShield(&player.ResistWeatherExpiry)
			if !applied {
				sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. Your resistance to the weather strengthens! (%d minutes remaining)", spell.Name, mins))
			} else {
				sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s! The weather's fury will no longer trouble you. (20 minutes)", spell.Name))
			}
		case 507, 508: // Heat Shield / Cold Shield
			var expiry *time.Time
			elementName := "heat"
			if spell.ID == 508 {
				expiry = &player.ColdShieldExpiry
				elementName = "cold"
			} else {
				expiry = &player.HeatShieldExpiry
			}
			mins, applied := applyElementalShield(expiry)
			if !applied {
				sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. Your resistance to %s strengthens! (%d minutes remaining)", spell.Name, elementName, mins))
			} else {
				sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s! You feel protected from %s. (50%% resistance, 20 minutes)", spell.Name, elementName))
				sc.RoomMsgs = append(sc.RoomMsgs, fmt.Sprintf("A shimmer surrounds %s.", player.FirstName))
			}
		case 509, 510: // Repel Plants / Repel Plants and Webs
			var expiry *time.Time
			scope := "plant snares"
			if spell.ID == 510 {
				expiry = &player.RepelPlantsAndWebsExpiry
				scope = "plant snares and webs"
			} else {
				expiry = &player.RepelPlantsExpiry
			}
			mins, applied := applyElementalShield(expiry)
			if !applied {
				sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. Your immunity to %s strengthens! (%d minutes remaining)", spell.Name, scope, mins))
			} else {
				sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s! You feel immune to %s. (20 minutes)", spell.Name, scope))
			}
		case 512: // True Aim
			sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. Your aim feels supernaturally true.", spell.Name))
			sc.RoomMsgs = append(sc.RoomMsgs, fmt.Sprintf("%s's eyes gleam with unnatural focus.", player.FirstName))
		case 513, 514, 515: // Agility I/II/III
			bonus, mins, applied, ok := applyAgilityBuff(player, spell)
			if !ok {
				sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s, but you already have a better Agility spell in place.", spell.Name))
			} else if !applied {
				sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. Your reflexes sharpen with renewed energy! (%d minutes remaining)", spell.Name, mins))
			} else {
				sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s! (+%d AGI, %d minutes)", spell.Name, bonus, mins))
			}
		case 521: // Camouflage
			mins, applied := applyCamouflageBuff(player)
			if !applied {
				sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. Your camouflage strengthens! (%d minutes remaining)", spell.Name, mins))
			} else {
				sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. Your skin and clothing shift to blend with your surroundings! (+10 Stealth, %d minutes)", spell.Name, mins))
				sc.RoomMsgs = append(sc.RoomMsgs, fmt.Sprintf("%s seems to blur into the background.", player.FirstName))
			}
		default:
			sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s.", spell.Name))
			sc.RoomMsgs = append(sc.RoomMsgs, fmt.Sprintf("A shimmer of magical energy surrounds %s.", player.FirstName))
		}

	case "utility":
		switch spell.ID {
		case 113: // Light
			sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. A soft glow illuminates the area.", spell.Name))
		case 114: // Mystic Key
			sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. You feel it could open any lock.", spell.Name))
		case 228: // Identify
			sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. Magical auras become clear to you.", spell.Name))
		case 338: // Unstun
			player.Stunned = false
			sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. Your mind clears!", spell.Name))
		case 401: // Dispel Lesser Magic
			sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. Lesser magical effects dissipate.", spell.Name))
		case 405: // See Hidden
			sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. You can sense hidden things nearby.", spell.Name))
		case 505: // Freedom
			if len(player.Entangles) > 0 {
				idx := rand.Intn(len(player.Entangles))
				removed := player.Entangles[idx]
				player.Entangles = append(player.Entangles[:idx], player.Entangles[idx+1:]...)
				sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. The %s releases its hold on you!", spell.Name, removed.SpellName))
			} else {
				sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s, but you aren't bound by any such magic.", spell.Name))
			}
		case 520: // Night Vision
			sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s. The darkness retreats from your eyes.", spell.Name))
		default:
			sc.Messages = append(sc.Messages, fmt.Sprintf("The item casts %s.", spell.Name))
			sc.RoomMsgs = append(sc.RoomMsgs, fmt.Sprintf("A shimmer of magical energy surrounds %s.", player.FirstName))
		}

	default:
		sc.Messages = append(sc.Messages, fmt.Sprintf("The item channels the power of %s.", spell.Name))
	}
}

func (sc *ScriptContext) expandScriptText(text string) string {
	// Player name — wolf-form Wolflings show as "a wolf" here too, same as
	// speech/emotes/movement/combat (see Player.DisplayNameCap).
	text = strings.ReplaceAll(text, "%N", sc.Player.DisplayNameCap())
	text = strings.ReplaceAll(text, "%n", sc.Player.DisplayNameCap())
	// Group name (just player name for now)
	text = strings.ReplaceAll(text, "%p", sc.Player.DisplayNameCap())
	text = strings.ReplaceAll(text, "%P", sc.Player.DisplayNameCap())
	// Item name
	if sc.ItemRef != nil && sc.ItemDef != nil {
		itemName := sc.Engine.formatItemName(sc.ItemDef, sc.ItemRef.Adj1, sc.ItemRef.Adj2, sc.ItemRef.Adj3, sc.ItemRef.Extend)
		text = strings.ReplaceAll(text, "%A", capitalize(itemName))
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
