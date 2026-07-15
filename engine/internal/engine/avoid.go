package engine

import (
	"context"
	"fmt"
	"strings"
)

// physicalEmoteVerbs are the interactive emotes (plus HOLD) that a player can
// block another player from using on them via AVOID.
var physicalEmoteVerbs = map[string]bool{
	"KISS": true, "NIBBLE": true, "LICK": true, "CARESS": true, "RUB": true,
	"CUDDLE": true, "SNUGGLE": true, "NUZZLE": true, "TOUCH": true, "TAP": true,
	"THUMP": true, "POKE": true, "HOLD": true,
}

// isAvoiding reports whether target has blocked actorName from using physical
// emotes/HOLD on them. AllowList overrides AvoidList for a specific name.
func (e *GameEngine) isAvoiding(actorName string, target *Player) bool {
	blocked := false
	for _, n := range target.AvoidList {
		if strings.EqualFold(n, actorName) {
			blocked = true
			break
		}
	}
	if !blocked {
		return false
	}
	for _, n := range target.AllowList {
		if strings.EqualFold(n, actorName) {
			return false
		}
	}
	return true
}

// isAllowedBy reports whether target has specifically ALLOWed actorName —
// standing consent for physical contact from that person, independent of
// (and sufficient on its own regardless of) whether the target is currently
// Submitting.
func (e *GameEngine) isAllowedBy(actorName string, target *Player) bool {
	for _, n := range target.AllowList {
		if strings.EqualFold(n, actorName) {
			return true
		}
	}
	return false
}

// avoidBlockMessage is the standard refusal shown when a physical emote/HOLD
// is blocked because the target is avoiding the actor.
func avoidBlockMessage(targetName string) *CommandResult {
	return &CommandResult{Messages: []string{fmt.Sprintf("%s does not welcome that kind of contact from you.", targetName)}}
}

func normalizePlayerName(args []string) string {
	name := strings.ToLower(strings.Join(args, " "))
	if name == "" {
		return ""
	}
	return capitalize(name)
}

// doAvoid handles AVOID [player] — blocks physical emotes/HOLD from a named player.
// With no args, lists everyone currently on the avoid list.
func (e *GameEngine) doAvoid(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		if len(player.AvoidList) == 0 {
			return &CommandResult{Messages: []string{"You are not avoiding anyone."}}
		}
		return &CommandResult{Messages: []string{"You are avoiding: " + strings.Join(player.AvoidList, ", ")}}
	}
	name := normalizePlayerName(args)
	if strings.EqualFold(name, player.FirstName) {
		return &CommandResult{Messages: []string{"You can't avoid yourself."}}
	}
	for _, n := range player.AvoidList {
		if strings.EqualFold(n, name) {
			return &CommandResult{Messages: []string{fmt.Sprintf("You are already avoiding %s.", name)}}
		}
	}
	player.AvoidList = append(player.AvoidList, name)
	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: []string{fmt.Sprintf("You will no longer accept physical contact or being held/carried by %s.", name)}}
}

// doUnavoid handles UNAVOID <player> — removes a player from the avoid list.
func (e *GameEngine) doUnavoid(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Unavoid whom?"}}
	}
	name := normalizePlayerName(args)
	for i, n := range player.AvoidList {
		if strings.EqualFold(n, name) {
			player.AvoidList = append(player.AvoidList[:i], player.AvoidList[i+1:]...)
			e.SavePlayer(ctx, player)
			return &CommandResult{Messages: []string{fmt.Sprintf("You are no longer avoiding %s.", name)}}
		}
	}
	return &CommandResult{Messages: []string{fmt.Sprintf("You are not avoiding %s.", name)}}
}

// doAllow handles ALLOW [player] — exempts a player from the avoid list,
// letting them use physical emotes/HOLD regardless of AVOID.
func (e *GameEngine) doAllow(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		if len(player.AllowList) == 0 {
			return &CommandResult{Messages: []string{"You aren't specially allowing anyone."}}
		}
		return &CommandResult{Messages: []string{"You are allowing: " + strings.Join(player.AllowList, ", ")}}
	}
	name := normalizePlayerName(args)
	if strings.EqualFold(name, player.FirstName) {
		return &CommandResult{Messages: []string{"You don't need to allow yourself."}}
	}
	for _, n := range player.AllowList {
		if strings.EqualFold(n, name) {
			return &CommandResult{Messages: []string{fmt.Sprintf("You are already allowing %s.", name)}}
		}
	}
	player.AllowList = append(player.AllowList, name)
	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: []string{fmt.Sprintf("You will allow %s to have physical contact with you, even if you're avoiding them.", name)}}
}

// doUnallow handles UNALLOW <player> — removes a player from the allow list.
func (e *GameEngine) doUnallow(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Unallow whom?"}}
	}
	name := normalizePlayerName(args)
	for i, n := range player.AllowList {
		if strings.EqualFold(n, name) {
			player.AllowList = append(player.AllowList[:i], player.AllowList[i+1:]...)
			e.SavePlayer(ctx, player)
			return &CommandResult{Messages: []string{fmt.Sprintf("You are no longer allowing %s.", name)}}
		}
	}
	return &CommandResult{Messages: []string{fmt.Sprintf("You are not allowing %s.", name)}}
}
