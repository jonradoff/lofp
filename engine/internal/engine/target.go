package engine

import (
	"fmt"
	"strings"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// maxTargets is the most creatures a player can hold in their target list
// at once (TARGET command), used by multi-target spells like Chain Lightning.
const maxTargets = 6

// targetEntry pairs a resolved, still-valid monster instance snapshot with
// its definition, in the order the target was added to the player's list.
type targetEntry struct {
	Inst MonsterInstance
	Def  *gameworld.MonsterDef
}

// doTarget handles TARGET [ordinal] <name>, building up to maxTargets
// monsters in the player's room for multi-target spells (Chain Lightning,
// Flaming Arrows, Siryx's Terrible Tentacles).
func (e *GameEngine) doTarget(player *Player, args []string) *CommandResult {
	if player.Dead {
		return &CommandResult{Messages: []string{"You can't target anything while dead."}}
	}
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Target what?"}}
	}

	e.pruneTargets(player)

	targetArg := strings.Join(args, " ")
	inst, _, outOfRange := e.findMonsterInRoomForTarget(player, targetArg)
	if outOfRange {
		return &CommandResult{Messages: []string{"What are you referring to?  Please be more specific."}}
	}
	if inst == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' here.", targetArg)}}
	}

	for _, id := range player.Targets {
		if id == inst.ID {
			return &CommandResult{Messages: []string{"That is already being targeted."}}
		}
	}

	if len(player.Targets) >= maxTargets {
		return &CommandResult{Messages: []string{"You are already targeting the maximum of 6 creatures."}}
	}

	player.Targets = append(player.Targets, inst.ID)

	msgs := []string{"Targets now include:"}
	for _, id := range player.Targets {
		if name, ok := e.targetDisplayName(id); ok {
			msgs = append(msgs, name)
		}
	}
	return &CommandResult{Messages: msgs}
}

// targetDisplayName returns the capitalized, articled display name ("A
// greater werewolf") for a still-living monster instance ID.
func (e *GameEngine) targetDisplayName(monsterID int) (string, bool) {
	if e.monsterMgr == nil {
		return "", false
	}
	e.monsterMgr.mu.RLock()
	defer e.monsterMgr.mu.RUnlock()
	for i := range e.monsterMgr.instances {
		if e.monsterMgr.instances[i].ID == monsterID {
			def := e.monsters[e.monsterMgr.instances[i].DefNumber]
			if def == nil {
				return "", false
			}
			name := FormatMonsterName(def, e.monAdjs)
			article := articleFor(name, def.Unique)
			return capArticle(article) + name, true
		}
	}
	return "", false
}

// findMonsterInRoomForTarget is like findMonsterInRoom, but distinguishes "no
// such creature here at all" from "there aren't that many of them" so TARGET
// can report the ordinal-out-of-range case distinctly, e.g. "target 2 laash"
// when only one laash is present.
func (e *GameEngine) findMonsterInRoomForTarget(player *Player, target string) (inst *MonsterInstance, def *gameworld.MonsterDef, outOfRange bool) {
	if e.monsterMgr == nil {
		return nil, nil, false
	}
	monsters := e.monsterMgr.MonstersInRoom(player.RoomNumber)
	target = strings.ToLower(strings.TrimSpace(target))
	target, skip := parseOrdinal(target)
	for _, article := range []string{"a ", "an ", "the ", "some "} {
		if strings.HasPrefix(target, article) {
			target = strings.TrimPrefix(target, article)
			break
		}
	}
	anyMatch := false
	for i := range monsters {
		d := e.monsters[monsters[i].DefNumber]
		if d == nil {
			continue
		}
		name := strings.ToLower(FormatMonsterName(d, e.monAdjs))
		noun := strings.ToLower(d.Name)
		if strings.HasPrefix(name, target) || strings.HasPrefix(noun, target) {
			anyMatch = true
			if skip > 0 {
				skip--
				continue
			}
			return &monsters[i], d, false
		}
	}
	return nil, nil, anyMatch
}

// pruneTargets drops targeted monster IDs that have died or left the room
// since they were added (killed, fled, wandered off, etc.).
func (e *GameEngine) pruneTargets(player *Player) {
	if len(player.Targets) == 0 || e.monsterMgr == nil {
		return
	}
	e.monsterMgr.mu.RLock()
	defer e.monsterMgr.mu.RUnlock()
	valid := player.Targets[:0]
	for _, id := range player.Targets {
		for i := range e.monsterMgr.instances {
			if e.monsterMgr.instances[i].ID == id {
				if e.monsterMgr.instances[i].Alive && e.monsterMgr.instances[i].RoomNumber == player.RoomNumber {
					valid = append(valid, id)
				}
				break
			}
		}
	}
	player.Targets = valid
}

// resolveTargets prunes stale entries and returns the player's current
// target list resolved to (instance, definition) pairs, in the order the
// targets were added.
func (e *GameEngine) resolveTargets(player *Player) []targetEntry {
	e.pruneTargets(player)
	if len(player.Targets) == 0 || e.monsterMgr == nil {
		return nil
	}
	e.monsterMgr.mu.RLock()
	defer e.monsterMgr.mu.RUnlock()
	entries := make([]targetEntry, 0, len(player.Targets))
	for _, id := range player.Targets {
		for i := range e.monsterMgr.instances {
			if e.monsterMgr.instances[i].ID == id {
				if def := e.monsters[e.monsterMgr.instances[i].DefNumber]; def != nil {
					entries = append(entries, targetEntry{Inst: e.monsterMgr.instances[i], Def: def})
				}
				break
			}
		}
	}
	return entries
}

// removeTargetID removes a monster instance ID from a target list, if present.
func removeTargetID(targets []int, id int) []int {
	for i, t := range targets {
		if t == id {
			return append(targets[:i], targets[i+1:]...)
		}
	}
	return targets
}
