package engine

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/metavert-io/lofp/internal/gameworld"
)

// ScriptContext holds state for script execution within a single trigger.
type ScriptContext struct {
	Player      *Player
	Room        *gameworld.Room
	Engine      *GameEngine
	Messages    []string // ECHO PLAYER messages to send to the player
	RoomMsgs    []string // ECHO ALL / ECHO OTHERS messages for the room
	GMMsgs      []string // GMMSG messages for gamemasters
	Blocked     bool     // CLEARVERB: block the triggering action
	MoveTo      int      // MOVE: destination room (0 = no move)
	MoveGroupTo int      // MOVEGROUP: move all players in room to destination
	MoveAllTo   int      // MOVEALL: move all players in room to destination

	StrVars  map[int]string  // %0-%9 from STRCVT
	OrigRoom *gameworld.Room // saved room for AFFECT

	// Item interaction context (set when running IFPREVERB/IFVERB on a room item)
	ItemRef *gameworld.RoomItem // the room item being interacted with
	ItemDef *gameworld.ItemDef  // its archetype definition

	DummyVars map[int]int // DUMMY1-5 temporary variables

	PlayerEventEnabled bool
	PlayerEventDelay   time.Duration
	EventDelay         time.Duration
	Suspended          bool
	RoomEvent          bool

	RemoteRoomMsgs []ScriptRoomMessage
}

type ScriptRoomMessage struct {
	RoomNumber int
	Text       string
}

// RunEntryScripts executes all IFENTRY script blocks for a room.
func (e *GameEngine) RunEntryScripts(player *Player, room *gameworld.Room) *ScriptContext {
	sc := &ScriptContext{
		Player:             player,
		Room:               room,
		Engine:             e,
		PlayerEventEnabled: true,
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

func isTouchVerb(verb string) bool {
	switch strings.ToUpper(verb) {
	case "GET", "TAKE", "KISS", "LICK",
		"OPEN", "CLOSE",
		"TOUCH", "PULL", "PUSH",
		"RUB", "TAP":
		return true
	default:
		return false
	}
}

func (e *GameEngine) RunPreverbScripts(
	player *Player,
	room *gameworld.Room,
	verb string,
	ri *gameworld.RoomItem,
	def *gameworld.ItemDef,
	ri2 ...*gameworld.RoomItem,
) *ScriptContext {

	sc := &ScriptContext{
		Player:             player,
		Room:               room,
		Engine:             e,
		ItemRef:            ri,
		ItemDef:            def,
		PlayerEventEnabled: true,
	}

	verb = strings.ToUpper(verb)

	refStr := "-1"
	if ri != nil {
		refStr = fmt.Sprintf("%d", ri.Ref)
	}

	//
	// ROOM SCRIPTS
	//
	if room != nil {
		for _, block := range room.Scripts {

			// TOUCH-style event.
			if ri != nil && isTouchVerb(verb) {
				sc.execEventTree(
					block,
					"IFTOUCH",
					verb,
					refStr,
				)
			}

			// Normal preverb.
			sc.execEventTree(
				block,
				"IFPREVERB",
				verb,
				refStr,
			)
		}
	}

	//
	// ITEM ARCHETYPE SCRIPTS
	//
	if def != nil {
		for _, block := range def.Scripts {

			// TOUCH-style event.
			if ri != nil && isTouchVerb(verb) {
				sc.execEventTree(
					block,
					"IFTOUCH",
					verb,
					refStr,
				)
			}

			// Normal preverb.
			sc.execEventTree(
				block,
				"IFPREVERB",
				verb,
				refStr,
			)
		}
	}

	//
	// SECOND-OBJECT PREVERBS
	//
	// Used by two-object commands such as:
	//
	//   PUT <object1> IN <object2>
	//
	// ri  = object #1 (the item being acted upon)
	// ri2 = object #2 (the destination/secondary object)
	//
	if len(ri2) > 0 && ri2[0] != nil {

		second := ri2[0]
		secondRefStr := fmt.Sprintf("%d", second.Ref)

		// Room scripts may contain IFPREVERB2 blocks.
		if room != nil {
			for _, block := range room.Scripts {
				sc.execEventTree(
					block,
					"IFPREVERB2",
					verb,
					secondRefStr,
				)
			}
		}

		// The second object's archetype may also contain
		// IFPREVERB2 blocks.
		secondDef := e.items[second.Archetype]

		if secondDef != nil {
			for _, block := range secondDef.Scripts {
				sc.execEventTree(
					block,
					"IFPREVERB2",
					verb,
					secondRefStr,
				)
			}
		}
	}

	sc.flushRemoteRoomMessages()

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

	// ------------------------------------------------------------
	// ROOM IFVERB
	// ------------------------------------------------------------

	if room != nil {
		for _, block := range room.Scripts {
			if block.Type == "IFVERB" && len(block.Args) >= 2 {
				if strings.ToUpper(block.Args[0]) == verb &&
					block.Args[1] == refStr {

					sc.execBlock(block)
				}
			}
		}
	}

	// ------------------------------------------------------------
	// ITEM ARCHETYPE IFVERB
	// ------------------------------------------------------------

	for _, block := range def.Scripts {
		if block.Type == "IFVERB" && len(block.Args) >= 1 {
			if strings.ToUpper(block.Args[0]) == verb {
				if len(block.Args) < 2 || block.Args[1] == "-1" {
					sc.execBlock(block)
				}
			}
		}
	}

	// ------------------------------------------------------------
	// TRAIT IFVERB
	//
	// Traits are attached to the item in order:
	//
	// TRAIT GENERIC_SWORD
	// TRAIT FANCY_SWORD
	//
	// If more than one trait defines the same IFVERB,
	// the LAST attached trait wins.
	// ------------------------------------------------------------

	var traitBlock *gameworld.ScriptBlock

	traitNames := append([]string{}, def.Traits...)

	if ri != nil {
		traitNames = append(traitNames, ri.Traits...)
	}

	for _, traitName := range traitNames {
		trait := e.traits[strings.ToUpper(traitName)]
		if trait == nil {
			continue
		}

		for i := range trait.Scripts {
			block := &trait.Scripts[i]

			if block.Type != "IFVERB" {
				continue
			}

			if len(block.Args) < 1 {
				continue
			}

			if strings.ToUpper(block.Args[0]) != verb {
				continue
			}

			if len(block.Args) >= 2 && block.Args[1] != "-1" {
				continue
			}

			traitBlock = block
		}
	}

	if traitBlock != nil {
		sc.execBlock(*traitBlock)
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

	case "IFPREVERB", "IFVERB", "IFPREVERB2":
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

// execElse runs the ELSE branch of a conditional block.
func (sc *ScriptContext) execElse(block gameworld.ScriptBlock) {
	sc.execStatements(block.ElseStatements)
}

// execChildren runs the ordered statements within a script block.
func (sc *ScriptContext) execChildren(block gameworld.ScriptBlock) {
	sc.execStatements(block.Statements)
}

func (sc *ScriptContext) execStatements(statements []gameworld.ScriptStatement) {
	for i := 0; i < len(statements); i++ {
		stmt := statements[i]

		if stmt.Action != nil {
			action := *stmt.Action
			cmd := strings.ToUpper(action.Command)

			// --------------------------------------------------------
			// PLAYER EVENT
			//
			// PLREVENT <delay>
			// CONTPLREVENT
			//
			// Everything after CONTPLREVENT resumes after the delay.
			// --------------------------------------------------------
			if sc.PlayerEventEnabled && cmd == "PLREVENT" {
				sc.execAction(action)

				if i+1 < len(statements) &&
					statements[i+1].Action != nil &&
					strings.EqualFold(
						statements[i+1].Action.Command,
						"CONTPLREVENT",
					) {

					sc.schedulePlayerEvent(
						sc.PlayerEventDelay,
						statements[i+2:],
					)

					sc.Suspended = true
					return
				}

				continue
			}

			if sc.PlayerEventEnabled && cmd == "CONTPLREVENT" {
				continue
			}

			// --------------------------------------------------------
			// GENERAL SCRIPT EVENT
			//
			// SETEVENT <event> <delay> [ALWAYS]
			// CONTEVENT <event>
			//
			// Everything after CONTEVENT resumes after the delay.
			// --------------------------------------------------------
			if cmd == "SETEVENT" {
				sc.execAction(action)

				if i+1 < len(statements) &&
					statements[i+1].Action != nil &&
					strings.EqualFold(
						statements[i+1].Action.Command,
						"CONTEVENT",
					) {

					sc.scheduleEvent(
						sc.EventDelay,
						statements[i+2:],
					)

					sc.Suspended = true
					return
				}

				continue
			}

			// CONTEVENT itself is only a continuation marker.
			if cmd == "CONTEVENT" {
				continue
			}

			sc.execAction(action)

			if sc.Suspended {
				return
			}

			continue
		}

		if stmt.Block != nil {
			sc.execBlock(*stmt.Block)

			if sc.Suspended {
				return
			}
		}
	}
}

func (sc *ScriptContext) scheduleEvent(delay time.Duration, statements []gameworld.ScriptStatement) {
	if len(statements) == 0 {
		return
	}

	// SETEVENT is a room/world event, not a player event.
	// Preserve the script context needed to continue execution,
	// but messages produced by the continuation are delivered
	// according to the affected room when the event fires.
	next := *sc

	next.Messages = nil
	next.RoomMsgs = nil
	next.GMMsgs = nil
	next.Suspended = false
	next.EventDelay = 0

	continuation := append(
		[]gameworld.ScriptStatement(nil),
		statements...,
	)

	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()

		<-timer.C

		if next.Engine == nil {
			return
		}

		// Run the delayed continuation.
		next.execStatements(continuation)

		// SETEVENT is room/world based.
		//
		// Anything generated as a direct player message should only
		// reach the original player if that player is STILL in the
		// room currently being affected.
		if len(next.Messages) > 0 &&
			next.Player != nil &&
			next.Room != nil &&
			next.Player.RoomNumber == next.Room.Number &&
			next.Engine.sendToPlayer != nil {

			next.Engine.sendToPlayer(
				next.Player.FirstName,
				next.Messages,
			)
		}

		// RoomMsgs go to everyone else in the currently affected room.
		if len(next.RoomMsgs) > 0 &&
			next.Room != nil &&
			next.Engine.roomBroadcast != nil {

			excludeName := ""

			// Only exclude the original player if they're actually
			// present in this affected room and therefore received
			// their copy through Messages above.
			if next.Player != nil &&
				next.Player.RoomNumber == next.Room.Number {

				excludeName = next.Player.FirstName
			}

			next.Engine.roomBroadcast(
				next.Room.Number,
				excludeName,
				next.RoomMsgs,
			)
		}

		// Persist player changes made by the continuation, if any.
		if next.Player != nil {
			next.Engine.SavePlayer(
				context.Background(),
				next.Player,
			)
		}
	}()
}

// scriptTreeContainsEvent reports whether this block contains the event
// we're currently dispatching. It does not execute anything.
func scriptTreeContainsEvent(block gameworld.ScriptBlock, eventType string, verb string, refStr string) bool {
	eventType = strings.ToUpper(eventType)
	verb = strings.ToUpper(verb)

	if strings.ToUpper(block.Type) == eventType {
		switch eventType {
		case "IFPREVERB", "IFPREVERB2", "IFVERB":
			if len(block.Args) < 1 {
				return false
			}

			if strings.ToUpper(block.Args[0]) != verb {
				return false
			}

			// Item archetype scripts normally use -1.
			// Room scripts may use a specific room-item ref.
			if len(block.Args) >= 2 &&
				block.Args[1] != "-1" &&
				block.Args[1] != refStr {
				return false
			}

			return true

		case "IFTOUCH":
			if len(block.Args) < 1 {
				return false
			}

			return block.Args[0] == "-1" ||
				block.Args[0] == refStr
		}
	}

	for _, child := range block.Children {
		if scriptTreeContainsEvent(child, eventType, verb, refStr) {
			return true
		}
	}

	for _, child := range block.ElseChildren {
		if scriptTreeContainsEvent(child, eventType, verb, refStr) {
			return true
		}
	}

	return false
}

func (sc *ScriptContext) execEventTree(block gameworld.ScriptBlock, eventType string, verb string, refStr string) {
	// Nothing for this event exists anywhere below here.
	if !scriptTreeContainsEvent(block, eventType, verb, refStr) {
		return
	}

	blockType := strings.ToUpper(block.Type)

	// If we've reached an event block, only execute it if it is
	// the event we're actually dispatching.
	switch blockType {
	case "IFPREVERB", "IFVERB", "IFPREVERB2":
		if blockType != eventType || len(block.Args) < 1 {
			return
		}

		if !strings.EqualFold(block.Args[0], verb) {
			return
		}

		if len(block.Args) >= 2 &&
			block.Args[1] != "-1" &&
			block.Args[1] != refStr {
			return
		}

		sc.execBlock(block)
		return

	case "IFTOUCH":
		if eventType != "IFTOUCH" || len(block.Args) < 1 {
			return
		}

		if block.Args[0] != "-1" && block.Args[0] != refStr {
			return
		}

		sc.execBlock(block)
		return
	}

	// This is a wrapper condition. Evaluate it, but recursively dispatch
	// only the branch that can contain our current event.
	switch blockType {
	case "IFVAR":
		result := sc.evalIfVar(block.Args)

		if sc.Player.GMTrace {
			sc.Messages = append(
				sc.Messages,
				fmt.Sprintf(
					"[TRACE]   IFVAR %s → %v",
					strings.Join(block.Args, " "),
					result,
				),
			)
		}

		if result {
			sc.execEventStatements(
				block.Actions,
				block.Children,
				eventType,
				verb,
				refStr,
			)
		} else {
			sc.execEventStatements(
				block.ElseActions,
				block.ElseChildren,
				eventType,
				verb,
				refStr,
			)
		}

	case "IFITEM":
		if sc.evalIfItem(block.Args) {
			sc.execEventStatements(
				block.Actions,
				block.Children,
				eventType,
				verb,
				refStr,
			)
		} else {
			sc.execEventStatements(
				block.ElseActions,
				block.ElseChildren,
				eventType,
				verb,
				refStr,
			)
		}

	case "IFNOITEM":
		if !sc.evalIfItem(block.Args) {
			sc.execEventStatements(
				block.Actions,
				block.Children,
				eventType,
				verb,
				refStr,
			)
		} else {
			sc.execEventStatements(
				block.ElseActions,
				block.ElseChildren,
				eventType,
				verb,
				refStr,
			)
		}

	case "IFCARRY":
		if sc.evalIfCarry(block.Args) {
			sc.execEventStatements(
				block.Actions,
				block.Children,
				eventType,
				verb,
				refStr,
			)
		} else {
			sc.execEventStatements(
				block.ElseActions,
				block.ElseChildren,
				eventType,
				verb,
				refStr,
			)
		}

	case "IFIN":
		if sc.evalIfIn(block.Args) {
			sc.execEventStatements(
				block.Actions,
				block.Children,
				eventType,
				verb,
				refStr,
			)
		} else {
			sc.execEventStatements(
				block.ElseActions,
				block.ElseChildren,
				eventType,
				verb,
				refStr,
			)
		}

	default:
		// Unknown/non-conditional wrapper: don't execute its actions.
		// Just continue looking through its children.
		for _, child := range block.Children {
			sc.execEventTree(child, eventType, verb, refStr)
		}

		for _, child := range block.ElseChildren {
			sc.execEventTree(child, eventType, verb, refStr)
		}
	}
}

func (sc *ScriptContext) execEventStatements(actions []gameworld.ScriptAction, children []gameworld.ScriptBlock, eventType string, verb string, refStr string) {
	// These actions belong to the conditional wrapper itself, so they
	// execute when that branch is selected.
	for _, action := range actions {
		sc.execAction(action)
	}

	// But nested event blocks are filtered to the event we're dispatching.
	for _, child := range children {
		sc.execEventTree(child, eventType, verb, refStr)
	}
}

// execAction executes a single script action.
func (sc *ScriptContext) execAction(action gameworld.ScriptAction) {
	switch action.Command {
	case "SPELL":
		sc.doSpell(action.Args)
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
	case "MOVEALL":
		sc.doMoveAll(action.Args)
	case "SHOWROOM":
		sc.doShowRoom(action.Args)
	case "PLREVENT":
		if sc.PlayerEventEnabled && len(action.Args) > 0 {
			seconds := sc.resolveNumericArg(action.Args[0])
			if seconds < 0 {
				seconds = 0
			}

			sc.PlayerEventDelay =
				time.Duration(seconds) * time.Second
		}

	case "CONTPLREVENT":
	// Marker used by PLREVENT continuation handling.

	case "SETEVENT":
		if len(action.Args) >= 2 {
			delay := sc.resolveNumericArg(action.Args[1])
			if delay < 0 {
				delay = 0
			}

			// For now, treat LOFP event timing as 3-second cycles.
			sc.EventDelay = time.Duration(delay) * 3 * time.Second
		}

	case "CONTEVENT":
	// Continuation marker handled by execStatements.

	case "AFFECT":
		sc.doAffect(action.Args)
	case "RANDOM":
		sc.doRandom(action.Args)
	case "DAMAGEPLR":
		sc.doDamagePlr(action.Args)
	case "DAMAGEALL":
		sc.doDamageAll(action.Args)
	case "HEALPLR":
		sc.doHealPlr(action.Args)
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
				sc.Player.ResetRoomVars()
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
		sc.doDamage(action.Args)
	case "DROPLOC":
		// TODO: set room where defeated players are moved
	case "CHANNEL":
		// TODO: set communication channel for room
	}
}

func (sc *ScriptContext) schedulePlayerEvent(
	delay time.Duration,
	statements []gameworld.ScriptStatement,
) {
	if len(statements) == 0 {
		return
	}

	// Make a continuation context. Do NOT keep using the caller's
	// temporary RoomItem because the original command is about to return.
	next := *sc

	next.Messages = nil
	next.RoomMsgs = nil
	next.GMMsgs = nil
	next.Suspended = false
	next.PlayerEventDelay = 0

	// Movement requests from the original part of the script have
	// already been handled. The continuation starts clean.
	next.MoveTo = 0
	next.MoveGroupTo = 0
	next.MoveAllTo = 0

	if sc.ItemRef != nil {
		itemCopy := *sc.ItemRef
		next.ItemRef = &itemCopy
	}

	continuation := append(
		[]gameworld.ScriptStatement(nil),
		statements...,
	)

	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()

		<-timer.C

		// Remember where the player was when the continuation actually ran.
		// This is the source room for delayed movement.
		originalRoom := 0
		if next.Player != nil {
			originalRoom = next.Player.RoomNumber
		}

		next.execStatements(continuation)

		// Persist ITEMVAL changes back to the actual player item.
		next.persistPlayerEventItem()

		if next.Engine == nil || next.Player == nil {
			return
		}

		bg := context.Background()

		// --------------------------------------------------------
		// Delayed MOVE
		// --------------------------------------------------------
		if next.MoveTo > 0 {
			if next.Engine.rooms[next.MoveTo] != nil {
				next.Player.ResetRoomVars()
				next.Player.RoomNumber = next.MoveTo

				next.Engine.disengageCombat(next.Player)

				next.Engine.SavePlayer(
					bg,
					next.Player,
				)
			}
		}

		// --------------------------------------------------------
		// Delayed MOVEGROUP
		// --------------------------------------------------------
		if next.MoveGroupTo > 0 {
			next.Engine.moveGroupToRoom(
				bg,
				originalRoom,
				next.MoveGroupTo,
			)
		}

		// --------------------------------------------------------
		// Delayed MOVEALL
		// --------------------------------------------------------
		if next.MoveAllTo > 0 {
			next.Engine.moveGroupToRoom(
				bg,
				originalRoom,
				next.MoveAllTo,
			)
		}

		next.Engine.SavePlayer(
			bg,
			next.Player,
		)

		// --------------------------------------------------------
		// Send delayed player messages.
		// --------------------------------------------------------
		if len(next.Messages) > 0 &&
			next.Engine.sendToPlayer != nil {

			next.Engine.sendToPlayer(
				next.Player.FirstName,
				next.Messages,
			)
		}

		// --------------------------------------------------------
		// Send delayed room messages.
		// --------------------------------------------------------
		if len(next.RoomMsgs) > 0 &&
			next.Engine.roomBroadcast != nil &&
			next.Room != nil {

			excludeName := next.Player.FirstName

			next.Engine.roomBroadcast(
				next.Room.Number,
				excludeName,
				next.RoomMsgs,
			)
		}
	}()
}

func (sc *ScriptContext) persistPlayerEventItem() {
	if sc.Player == nil ||
		sc.ItemRef == nil ||
		sc.ItemRef.Ref != -1 {
		return
	}

	src := sc.ItemRef

	copyVals := func(ii *InventoryItem) bool {
		if ii == nil {
			return false
		}

		if ii.Archetype != src.Archetype ||
			ii.Adj1 != src.Adj1 ||
			ii.Adj2 != src.Adj2 ||
			ii.Adj3 != src.Adj3 {
			return false
		}

		ii.Val1 = src.Val1
		ii.Val2 = src.Val2
		ii.Val3 = src.Val3
		ii.Val4 = src.Val4
		ii.Val5 = src.Val5

		return true
	}

	for i := range sc.Player.Inventory {
		if copyVals(&sc.Player.Inventory[i]) {
			return
		}
	}

	for i := range sc.Player.Worn {
		if copyVals(&sc.Player.Worn[i]) {
			return
		}
	}

	if sc.Player.Wielded != nil {
		copyVals(sc.Player.Wielded)
	}
}

// doEcho handles ECHO PLAYER, ECHO ALL, ECHO GROUP, ECHO OTHERS.
func (sc *ScriptContext) doEcho(args []string) {
	if len(args) < 2 {
		return
	}

	target := strings.ToUpper(args[0])
	text := strings.Join(args[1:], " ")
	text = sc.expandScriptText(text)

	switch target {
	case "PLAYER":
		// Explicitly addressed to the player who triggered the script.
		sc.Messages = append(sc.Messages, text)

	case "ALL", "GROUP":
		if sc.Room == nil {
			return
		}

		// If the triggering player is physically in the room currently
		// being affected, give them their direct copy.
		if sc.Player != nil &&
			sc.Player.RoomNumber == sc.Room.Number {

			sc.Messages = append(sc.Messages, text)
		}

		// If this is the player's current room, preserve the existing
		// RoomBroadcast behavior.
		if sc.Player != nil &&
			sc.Player.RoomNumber == sc.Room.Number {

			sc.RoomMsgs = append(sc.RoomMsgs, text)
			return
		}

		// AFFECT has switched the script to another room.
		// Preserve that destination instead of losing it in RoomMsgs.
		sc.RemoteRoomMsgs = append(
			sc.RemoteRoomMsgs,
			ScriptRoomMessage{
				RoomNumber: sc.Room.Number,
				Text:       text,
			},
		)

	case "OTHERS":
		if sc.Room == nil {
			return
		}

		// Normal same-room broadcast.
		if sc.Player != nil &&
			sc.Player.RoomNumber == sc.Room.Number {

			sc.RoomMsgs = append(sc.RoomMsgs, text)
			return
		}

		// Remote room selected by AFFECT.
		sc.RemoteRoomMsgs = append(
			sc.RemoteRoomMsgs,
			ScriptRoomMessage{
				RoomNumber: sc.Room.Number,
				Text:       text,
			},
		)
	}
}

func (sc *ScriptContext) flushRemoteRoomMessages() {
	if sc == nil ||
		sc.Engine == nil ||
		sc.Engine.roomBroadcast == nil ||
		len(sc.RemoteRoomMsgs) == 0 {

		return
	}

	for _, msg := range sc.RemoteRoomMsgs {
		if msg.RoomNumber <= 0 || msg.Text == "" {
			continue
		}

		sc.Engine.roomBroadcast(
			msg.RoomNumber,
			"",
			[]string{msg.Text},
		)
	}

	sc.RemoteRoomMsgs = nil
}

func (sc *ScriptContext) doDamage(args []string) {
	if len(args) < 2 {
		return
	}

	damageType := strings.ToUpper(args[0])
	amount := sc.resolveNumericArg(args[1])

	sc.Player.BodyPoints -= amount
	if sc.Player.BodyPoints < 0 {
		sc.Player.BodyPoints = 0
	}

	if len(args) > 2 {
		text := strings.Join(args[2:], " ")
		text = sc.expandScriptText(text)

		sc.Messages = append(sc.Messages, text)
		sc.RoomMsgs = append(sc.RoomMsgs, text)
	}

	_ = damageType
}

// doEqual handles EQUAL INTNUMn value — sets a variable on the player.
func (sc *ScriptContext) doEqual(args []string) {
	if len(args) < 2 {
		return
	}

	varName := strings.ToUpper(args[0])
	val := sc.resolveNumericArg(args[1])

	sc.setVar(varName, val)

	if sc.Player != nil && sc.Player.GMTrace {
		sc.Messages = append(
			sc.Messages,
			fmt.Sprintf("[TRACE]   %s = %d", varName, val),
		)
	}
}

// doAdd handles ADD variable value — increments a variable.
func (sc *ScriptContext) doAdd(args []string) {
	if len(args) < 2 {
		return
	}

	varName := strings.ToUpper(args[0])
	sc.setVar(varName, sc.getVar(varName)+sc.resolveNumericArg(args[1]))
}

// doSub handles SUB variable value — decrements a variable.
func (sc *ScriptContext) doSub(args []string) {
	if len(args) < 2 {
		return
	}

	varName := strings.ToUpper(args[0])

	val := sc.resolveNumericArg(args[1])

	sc.setVar(varName, sc.getVar(varName)-val)
}

// doNewItem handles NEWITEM ref archetype [ADJ1=n] [ADJ2=n] [VAL1=n] ...
// ref -1 means add to player inventory.
// doNewItem handles NEWITEM ref archetype [ADJ1=n] [ADJ2=n] [VAL1=n] ...
// ref -1 means add to player inventory.
// Otherwise ref is the room item reference number.
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

	// Parse optional values once.
	adj1, adj2, adj3 := -1, -1, -1
	val1, val2, val3, val4, val5 := 0, 0, 0, 0, 0

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
			adj1 = val
		case "ADJ2":
			adj2 = val
		case "ADJ3":
			adj3 = val
		case "VAL1":
			val1 = val
		case "VAL2":
			val2 = val
		case "VAL3":
			val3 = val
		case "VAL4":
			val4 = val
		case "VAL5":
			val5 = val
		}
	}

	// -1 means create the item in the player's inventory.
	if ref == -1 {
		item := InventoryItem{
			Archetype: archetype,
			Adj1:      adj1,
			Adj2:      adj2,
			Adj3:      adj3,
			Val1:      val1,
			Val2:      val2,
			Val3:      val3,
			Val4:      val4,
			Val5:      val5,
		}

		sc.Player.Inventory = append(sc.Player.Inventory, item)
		return
	}

	// Otherwise create the item in the room using the supplied
	// room item reference number.
	item := gameworld.RoomItem{
		Ref:       ref,
		Archetype: archetype,
		Adj1:      adj1,
		Adj2:      adj2,
		Adj3:      adj3,
		Val1:      val1,
		Val2:      val2,
		Val3:      val3,
		Val4:      val4,
		Val5:      val5,
	}

	// NEWITEM for an existing ref replaces that room item.
	for i := range sc.Room.Items {
		if sc.Room.Items[i].Ref == ref {
			sc.Room.Items[i] = item
			return
		}
	}

	// Ref isn't currently present, so add it.
	sc.Room.Items = append(sc.Room.Items, item)
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
	if dest <= 0 {
		return
	}

	if sc.Engine.rooms[dest] == nil {
		return
	}

	sc.Player.ResetRoomVars()
	sc.Player.RoomNumber = dest

	// Keep this so callers can detect that a scripted MOVE occurred
	// and handle LOOK / persistence / entry-script presentation.
	sc.MoveTo = dest
}

// doMoveGroup handles MOVEGROUP <room> — moves all players in the room to destination.
func (sc *ScriptContext) doMoveGroup(args []string) {
	if len(args) == 0 {
		return
	}
	dest := sc.resolveNumericArg(args[0])
	if dest > 0 {
		sc.MoveGroupTo = dest
		//	sc.Player.RoomNumber = dest

	}
}

// doMoveAll handles MOVEALL <room> — moves all players in the room.
func (sc *ScriptContext) doMoveAll(args []string) {
	if len(args) == 0 {
		return
	}

	dest := sc.resolveNumericArg(args[0])
	if dest <= 0 {
		return
	}

	if sc.Engine.rooms[dest] == nil {
		return
	}

	sc.MoveAllTo = dest
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
	if len(args) < 3 || sc.Room == nil {
		return
	}

	ref, err := strconv.Atoi(args[0])
	if err != nil {
		return
	}

	valIndex, err := strconv.Atoi(args[1])
	if err != nil || valIndex < 1 || valIndex > 5 {
		return
	}

	// The value may be a literal number or a script variable.
	value, err := strconv.Atoi(args[2])
	if err != nil {
		value = sc.getVar(args[2])
	}

	for i := range sc.Room.Items {
		if sc.Room.Items[i].Ref != ref {
			continue
		}

		switch valIndex {
		case 1:
			sc.Room.Items[i].Val1 = value
		case 2:
			sc.Room.Items[i].Val2 = value
		case 3:
			sc.Room.Items[i].Val3 = value
		case 4:
			sc.Room.Items[i].Val4 = value
		case 5:
			sc.Room.Items[i].Val5 = value
		}

		return
	}
}

// doRemoveItem handles REMOVEITEM ref — removes item from player or room.
// doRemoveItem handles REMOVEITEM ref.
// ref -1 means remove the currently active item.
func (sc *ScriptContext) doRemoveItem(args []string) {
	if len(args) == 0 {
		return
	}

	ref, err := strconv.Atoi(args[0])
	if err != nil {
		return
	}

	// ------------------------------------------------------------
	// REMOVEITEM -1 = remove the currently active item.
	// ------------------------------------------------------------
	if ref == -1 && sc.ItemRef != nil {

		// First check whether the active item is a room item.
		if sc.Room != nil {
			for i := range sc.Room.Items {
				ri := sc.Room.Items[i]

				if ri.Ref == sc.ItemRef.Ref &&
					ri.Archetype == sc.ItemRef.Archetype &&
					ri.Adj1 == sc.ItemRef.Adj1 &&
					ri.Adj2 == sc.ItemRef.Adj2 &&
					ri.Adj3 == sc.ItemRef.Adj3 {

					sc.Room.Items = append(
						sc.Room.Items[:i],
						sc.Room.Items[i+1:]...,
					)

					sc.Engine.notifyRoomChange(RoomChange{
						RoomNumber: sc.Room.Number,
						Type:       "item_remove",
						ItemRef:    ri.Ref,
					})

					if sc.Player != nil && sc.Player.GMTrace {
						sc.Messages = append(
							sc.Messages,
							fmt.Sprintf(
								"[TRACE]   REMOVEITEM -1 removed room item ref=%d archetype=%d",
								ri.Ref,
								ri.Archetype,
							),
						)
					}

					return
				}
			}
		}

		// If it wasn't in the room, try player inventory.
		for i, ii := range sc.Player.Inventory {
			if ii.Archetype == sc.ItemRef.Archetype &&
				ii.Adj1 == sc.ItemRef.Adj1 &&
				ii.Adj2 == sc.ItemRef.Adj2 &&
				ii.Adj3 == sc.ItemRef.Adj3 &&
				ii.Val1 == sc.ItemRef.Val1 &&
				ii.Val2 == sc.ItemRef.Val2 &&
				ii.Val3 == sc.ItemRef.Val3 &&
				ii.Val4 == sc.ItemRef.Val4 &&
				ii.Val5 == sc.ItemRef.Val5 {

				sc.Player.Inventory = append(
					sc.Player.Inventory[:i],
					sc.Player.Inventory[i+1:]...,
				)

				if sc.Player.GMTrace {
					sc.Messages = append(
						sc.Messages,
						fmt.Sprintf(
							"[TRACE]   REMOVEITEM -1 removed inventory item archetype=%d",
							ii.Archetype,
						),
					)
				}

				return
			}
		}

		if sc.Player.GMTrace {
			sc.Messages = append(
				sc.Messages,
				"[TRACE]   REMOVEITEM -1 could not find active item",
			)
		}

		return
	}

	// ------------------------------------------------------------
	// REMOVEITEM <ref> = remove a specific room item reference.
	// ------------------------------------------------------------
	if ref >= 0 && sc.Room != nil {
		for i := range sc.Room.Items {
			ri := sc.Room.Items[i]

			if ri.Ref != ref || ri.IsPut {
				continue
			}

			sc.Room.Items = append(
				sc.Room.Items[:i],
				sc.Room.Items[i+1:]...,
			)

			sc.Engine.notifyRoomChange(RoomChange{
				RoomNumber: sc.Room.Number,
				Type:       "item_remove",
				ItemRef:    ref,
			})

			if sc.Player != nil && sc.Player.GMTrace {
				sc.Messages = append(
					sc.Messages,
					fmt.Sprintf(
						"[TRACE]   REMOVEITEM %d removed room item archetype=%d",
						ref,
						ri.Archetype,
					),
				)
			}

			return
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

	actual := sc.getVar(varName)
	expected := sc.resolveNumericArg(args[2])

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

// evalIfItem evaluates IFITEM conditions like:
// IFITEM 1
// IFITEM 0 OPEN
// IFITEM -1 CLOSED
func (sc *ScriptContext) evalIfItem(args []string) bool {
	if len(args) < 1 {
		return false
	}

	ref, err := strconv.Atoi(args[0])
	if err != nil {
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

	// IFITEM <ref> by itself simply tests whether the item exists.
	if len(args) == 1 {
		return ri != nil
	}

	if ri == nil {
		return false
	}

	expectedState := strings.ToUpper(args[1])
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

	case "WORN":
		for i := range sc.Player.Worn {
			item := &sc.Player.Worn[i]

			if item.Archetype == ri.Archetype &&
				item.Adj1 == ri.Adj1 &&
				item.Adj2 == ri.Adj2 &&
				item.Adj3 == ri.Adj3 {
				return true
			}
		}

	case "WIELDED":
		if sc.Player.Wielded == nil {
			return false
		}

		item := sc.Player.Wielded

		return item.Archetype == ri.Archetype &&
			item.Adj1 == ri.Adj1 &&
			item.Adj2 == ri.Adj2 &&
			item.Adj3 == ri.Adj3
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
		case 1:
			return sc.Player.Flag1
		case 2:
			return sc.Player.Flag2
		case 3:
			return sc.Player.Flag3
		case 4:
			return sc.Player.Flag4
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
		if sc.Player.Race == 8 {
			return 1
		}
		return 0
	// Player stats
	// case "STR" returns the base strength, while "STRT" returns the effective strength after modifiers.
	case "STR":
		return sc.Player.Strength
	case "STRT":
		return sc.Player.EffectiveStat(StatStrength)

	case "AGI":
		return sc.Player.Agility
	case "AGIT":
		return sc.Player.EffectiveStat(StatAgility)

	case "CON":
		return sc.Player.Constitution
	case "CONT":
		return sc.Player.EffectiveStat(StatConstitution)

	case "QUI":
		return sc.Player.Quickness
	case "QUIT":
		return sc.Player.EffectiveStat(StatQuickness)

	case "WIL":
		return sc.Player.Willpower
	case "WILT":
		return sc.Player.EffectiveStat(StatWillpower)

	case "PER":
		return sc.Player.Perception
	case "PERT":
		return sc.Player.EffectiveStat(StatPerception)

	case "EMP":
		return sc.Player.Empathy
	case "EMPT":
		return sc.Player.EffectiveStat(StatEmpathy)
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
		if sc.Player.Dead {
			return 1
		}
		return 0
	case "FLYING":
		if sc.Player.Position == 4 {
			return 1
		}
		return 0
	case "KNEELING":
		if sc.Player.Position == 3 {
			return 1
		}
		return 0
	case "LAYING":
		if sc.Player.Position == 2 {
			return 1
		}
		return 0
	case "SITTING":
		if sc.Player.Position == 1 {
			return 1
		}
		return 0
	case "STANDING":
		if sc.Player.Position == 0 {
			return 1
		}
		return 0
	case "HIDDEN":
		if sc.Player.Hidden {
			return 1
		}
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
		if sc.Player.Wielded != nil {
			return 1
		}
		return 0

	case "ARCHNUM":
		if sc.ItemRef != nil {
			return sc.ItemRef.Archetype
		}
		return -1
	// Room info
	case "RNUM":
		return sc.Player.RoomNumber
	case "OUTDOOR":
		if sc.Room != nil && isOutdoorTerrain(sc.Room.Terrain) {
			return 1
		}
		return 0
	case "PLRSINROOM":
		if sc.Engine.sessions == nil || sc.Room == nil {
			return 0
		}

		count := 0

		for _, p := range sc.Engine.sessions.OnlinePlayers() {
			roomNumber := p.RoomNumber

			// MOVE is applied after the script finishes. While the script
			// is still running, treat the player executing it as already
			// being in the MOVE destination.
			if sc.Player != nil &&
				p == sc.Player &&
				sc.MoveTo > 0 {

				roomNumber = sc.MoveTo
			}

			if roomNumber == sc.Room.Number {
				count++
			}
		}

		return count
	case "MONINROOM":
		if sc.Engine.monsterMgr != nil && sc.Room != nil {
			return len(sc.Engine.monsterMgr.MonstersInRoom(sc.Room.Number))
		}
		return 0

	// Time
	case "TIM":
		return GameHour()
	case "DAY":
		if IsDay() {
			return 1
		}
		return 0
	case "NIGHT":
		if IsNight() {
			return 1
		}
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
		if sc.Player.WolfForm {
			return 1
		}
		return 0
	case "SLIMEFORM":
		if sc.Player.SlimeForm {
			return 1
		}
		return 0
	case "OTHERFORM":
		if sc.Player.WolfForm || sc.Player.SlimeForm || (sc.Player.Race == 8 && sc.Player.Hidden) {
			return 1
		}
		return 0
	case "UNDEAD":
		if sc.Player.Undead {
			return 1
		}
		return 0
	case "DISGUISED":
		if sc.Player.Disguised {
			return 1
		}
		return 0
	case "SLEEPING":
		if sc.Player.Sleeping {
			return 1
		}
		return 0
	case "SUBMITTING":
		if sc.Player.Submitting {
			return 1
		}
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
		if sc.Room != nil && sc.Room.Terrain == "ASTRAL" {
			return 1
		}
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
			return sc.ItemDef.Weight
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
		case 1:
			sc.Player.Flag1 = val
		case 2:
			sc.Player.Flag2 = val
		case 3:
			sc.Player.Flag3 = val
		case 4:
			sc.Player.Flag4 = val
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
		if val > sc.Player.MaxBodyPoints {
			val = sc.Player.MaxBodyPoints
		}
		if val < 0 {
			val = 0
		}
		sc.Player.BodyPoints = val
	case "ROUNDTIME":
		sc.Player.RoundTime = val
		sc.Player.RoundTimeExpiry = time.Now().Add(time.Duration(val) * time.Second)
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

func (sc *ScriptContext) resolveNumericArg(arg string) int {
	// First try a literal number.
	if val, err := strconv.Atoi(arg); err == nil {
		return val
	}

	// Otherwise treat it as a script variable.
	return sc.getVar(strings.ToUpper(arg))
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

	if sc.Player.HasItem(archetype, adj) {
		return true
	}

	/*for _, ii := range sc.Player.Inventory {
		if ii.Archetype == archetype {
			if adj < 0 || ii.Adj1 == adj {
				return true
			}
		}
	}*/

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
	if len(args) < 4 {
		return
	}

	varName := strings.ToUpper(args[0])

	start, err1 := strconv.Atoi(args[1])
	end, err2 := strconv.Atoi(args[2])
	modifier, err3 := strconv.Atoi(args[3])

	if err1 != nil || err2 != nil || err3 != nil {
		return
	}

	if end < start {
		return
	}

	value := rand.Intn(end-start+1) + start + modifier

	sc.setVar(varName, value)

	if sc.Player != nil && sc.Player.GMTrace {
		sc.Messages = append(
			sc.Messages,
			fmt.Sprintf("[TRACE]   %s = %d", varName, value),
		)
	}
}

func (sc *ScriptContext) doDamagePlr(args []string) {
	if len(args) < 1 {
		return
	}

	idx := 0
	bodyOnly := false
	damageType := ""

	// Optional BODYONLY keyword.
	if strings.EqualFold(args[idx], "BODYONLY") {
		bodyOnly = true
		idx++
	}

	if idx >= len(args) {
		return
	}

	// Support both:
	// DAMAGEPLR 15 message...
	// DAMAGEPLR ELECTRIC 15 message...
	if _, err := strconv.Atoi(args[idx]); err != nil {
		damageType = strings.ToUpper(args[idx])
		idx++
	}

	if idx >= len(args) {
		return
	}

	/*amount, err := strconv.Atoi(args[idx])
	if err != nil {
		return
	}
	idx++  */

	amount := sc.resolveNumericArg(args[idx])
	idx++

	finalDamage := amount

	if !bodyOnly && damageType != "" {
		finalDamage = sc.applyPlayerResistance(damageType, amount)
	}

	if finalDamage < 0 {
		finalDamage = 0
	}

	sc.Player.BodyPoints -= finalDamage
	if sc.Player.BodyPoints < 0 {
		sc.Player.BodyPoints = 0
	}

	if idx < len(args) {
		text := strings.Join(args[idx:], " ")
		sc.Messages = append(sc.Messages, sc.expandScriptText(text))
	}
}

// doDamageAll handles DAMAGEALL using the same damage logic as DAMAGEPLR,
// but applies it to every living player in the current room.
func (sc *ScriptContext) doDamageAll(args []string) {
	if len(args) < 1 {
		return
	}

	if sc.Engine == nil || sc.Player == nil || sc.Engine.sessions == nil {
		return
	}

	roomNumber := sc.Player.RoomNumber

	for _, p := range sc.Engine.sessions.OnlinePlayers() {
		if p == nil || p.Dead || p.RoomNumber != roomNumber {
			continue
		}

		// Use a copy of the current script context so DAMAGEPLR can
		// operate normally against this player.
		damageCtx := *sc
		damageCtx.Player = p

		// Don't inherit messages intended for the original player.
		damageCtx.Messages = nil
		damageCtx.RoomMsgs = nil
		damageCtx.GMMsgs = nil

		damageCtx.doDamagePlr(args)

		// The player who triggered the script uses the normal script
		// message path.
		if p == sc.Player {
			sc.Messages = append(
				sc.Messages,
				damageCtx.Messages...,
			)
			continue
		}

		// Other players need their DAMAGEALL message delivered directly.
		if len(damageCtx.Messages) > 0 &&
			sc.Engine.sendToPlayer != nil {

			sc.Engine.sendToPlayer(
				p.FirstName,
				damageCtx.Messages,
			)
		}
	}
}

func (sc *ScriptContext) applyPlayerResistance(damageType string, amount int) int {
	resistance := 0

	switch strings.ToUpper(damageType) {
	case "BURN", "FIRE":
		resistance = 0 //todo: sc.Player.FireResistance
	case "ELECTRIC", "LIGHTNING":
		resistance = 0 //todo sc.Player.ElectricResistance
	case "COLD", "ICE":
		resistance = 0 //todo: sc.Player.ColdResistance
	case "POISON":
		resistance = 0 //todo: sc.Player.PoisonResistance
	}

	// Assuming resistance is a percentage from 0 to 100.
	if resistance < 0 {
		resistance = 0
	}
	if resistance > 100 {
		resistance = 100
	}

	return amount * (100 - resistance) / 100
}

func (sc *ScriptContext) doHealPlr(args []string) {
	if len(args) < 1 {
		return
	}

	amount := sc.resolveNumericArg(args[0])
	if amount < 0 {
		amount = 0
	}

	oldBP := sc.Player.BodyPoints

	sc.Player.BodyPoints += amount
	if sc.Player.BodyPoints > sc.Player.MaxBodyPoints {
		sc.Player.BodyPoints = sc.Player.MaxBodyPoints
	}

	actualHeal := sc.Player.BodyPoints - oldBP

	if len(args) > 1 {
		text := strings.Join(args[1:], " ")
		text = strings.ReplaceAll(text, "%D", strconv.Itoa(actualHeal))
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

	result := sc.getVar(varName) * val
	sc.setVar(varName, result)

	if sc.Player != nil && sc.Player.GMTrace {
		sc.Messages = append(
			sc.Messages,
			fmt.Sprintf("[TRACE]   %s = %d", varName, result),
		)
	}
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
	mod := sc.resolveNumericArg(args[1])

	if mod == 0 {
		return
	}

	result := sc.getVar(varName) % mod
	sc.setVar(varName, result)

	if sc.Player != nil && sc.Player.GMTrace {
		sc.Messages = append(
			sc.Messages,
			fmt.Sprintf("[TRACE]   %s = %d", varName, result),
		)
	}
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

// doNewPut handles NEWPUT ref archetype [key=value...] — places item inside a container in the room.
func (sc *ScriptContext) doNewPut(args []string) {

	if len(args) < 2 {
		return
	}

	ref, _ := strconv.Atoi(args[0])
	archetype, _ := strconv.Atoi(args[1])

	item := gameworld.RoomItem{
		Ref:       ref,
		Archetype: archetype,
		Adj1:      -1,
		Adj2:      -1,
		Adj3:      -1,
		IsPut:     true,
	}

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
	text = strings.ReplaceAll(text, "%e", func() string {
		if sc.Player.Gender == 0 {
			return "he"
		}
		return "she"
	}())
	text = strings.ReplaceAll(text, "%o", func() string {
		if sc.Player.Gender == 0 {
			return "him"
		}
		return "her"
	}())
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

func (sc *ScriptContext) doSpell(args []string) {
	if len(args) == 0 {
		return
	}

	spellID := sc.resolveNumericArg(args[0])
	if spellID <= 0 {
		return
	}

	spell := FindSpellByID(spellID)
	if spell == nil {
		return
	}

	switch spell.Effect {
	case "heal":
		result := sc.Engine.castHealSpell(
			sc.Player,
			spell,
			nil,
			false,
		)

		if result != nil {
			sc.Messages = append(sc.Messages, result.Messages...)
			sc.RoomMsgs = append(sc.RoomMsgs, result.RoomBroadcast...)
		}
	default:
		result := sc.Engine.castStatusSpell(
			sc.Player,
			spell,
			nil,
			false,
		)

		if result != nil {
			sc.Messages = append(sc.Messages, result.Messages...)
			sc.RoomMsgs = append(sc.RoomMsgs, result.RoomBroadcast...)
		}
	}
}
