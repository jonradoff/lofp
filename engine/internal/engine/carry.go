package engine

import (
	"context"
	"fmt"
	"strings"
)

// doCarry handles CARRY <player> — pick up a submitting or dead player so
// they travel with you between rooms until released (see doRelease) or the
// carry is broken (RELEASE/PUTDOWN, the carried player acting or
// un-submitting, or the carrier entering combat).
func (e *GameEngine) doCarry(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Carry whom?"}}
	}
	if player.Carrying != "" {
		return &CommandResult{Messages: []string{fmt.Sprintf("You are already carrying %s. RELEASE them first.", player.Carrying)}}
	}
	if player.CarriedBy != "" {
		return &CommandResult{Messages: []string{"You can't carry anyone while you are being carried."}}
	}

	targetName := strings.ToLower(strings.Join(args, " "))
	found := e.findPlayerInRoom(player, targetName)
	if found == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' here.", targetName)}}
	}
	if found == player {
		return &CommandResult{Messages: []string{"You can't carry yourself."}}
	}
	if !found.Dead && !found.Submitting {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s must be submitting or dead to be carried.", found.FirstName)}}
	}
	if !found.Dead && e.isAvoiding(player.FirstName, found) {
		return avoidBlockMessage(found.FirstName)
	}
	if found.CarriedBy != "" {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s is already being carried by %s.", found.FirstName, found.CarriedBy)}}
	}

	player.Carrying = found.FirstName
	found.CarriedBy = player.FirstName
	e.SavePlayer(ctx, player)
	e.SavePlayer(ctx, found)

	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You pick %s up.", found.FirstName)},
		RoomBroadcast: []string{fmt.Sprintf("%s picks %s up.", player.FirstName, found.FirstName)},
		TargetName:    found.FirstName,
		TargetMsg:     []string{fmt.Sprintf("%s picks you up.", player.FirstName)},
	}
}

// doReleaseCarry handles RELEASE/PUTDOWN [player] — sets down whoever the
// player is currently carrying.
func (e *GameEngine) doReleaseCarry(ctx context.Context, player *Player) *CommandResult {
	if player.Carrying == "" {
		return &CommandResult{Messages: []string{"You aren't carrying anyone."}}
	}
	carriedName := player.Carrying
	player.Carrying = ""
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.FirstName == carriedName {
				p.CarriedBy = ""
				if p.Dead {
					p.Position = 2 // laying down
				}
				e.SavePlayer(ctx, p)
				break
			}
		}
	}
	e.SavePlayer(ctx, player)
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You set %s down.", carriedName)},
		RoomBroadcast: []string{fmt.Sprintf("%s sets %s down.", player.FirstName, carriedName)},
	}
}

// doLayCarried handles LAY <player> — sets down whoever the player is
// carrying, always leaving them in the laying position (unlike RELEASE/
// PUTDOWN/DROP, which only lay a carried player down if they're dead).
func (e *GameEngine) doLayCarried(ctx context.Context, player *Player, args []string) *CommandResult {
	if player.Carrying == "" {
		return &CommandResult{Messages: []string{"You aren't carrying anyone."}}
	}
	targetName := strings.ToLower(strings.Join(args, " "))
	if !strings.HasPrefix(strings.ToLower(player.Carrying), targetName) {
		return &CommandResult{Messages: []string{fmt.Sprintf("You aren't carrying %s.", strings.Join(args, " "))}}
	}
	carriedName := player.Carrying
	player.Carrying = ""
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.FirstName == carriedName {
				p.CarriedBy = ""
				p.Position = 2 // laying down
				e.SavePlayer(ctx, p)
				break
			}
		}
	}
	e.SavePlayer(ctx, player)
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You lay %s down.", carriedName)},
		RoomBroadcast: []string{fmt.Sprintf("%s lays %s down.", player.FirstName, carriedName)},
	}
}

// breakCarryAsCarrier ends the carry relationship when the carrier is the one
// whose action broke it (e.g. entering combat), notifying both sides.
func (e *GameEngine) breakCarryAsCarrier(ctx context.Context, player *Player) {
	if player.Carrying == "" {
		return
	}
	carriedName := player.Carrying
	player.Carrying = ""
	e.SavePlayer(ctx, player)
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.FirstName == carriedName {
				p.CarriedBy = ""
				if p.Dead {
					p.Position = 2 // laying down
				}
				e.SavePlayer(ctx, p)
				if e.sendToPlayer != nil {
					e.sendToPlayer(p.FirstName, []string{fmt.Sprintf("%s sets you down abruptly!", player.FirstName)})
				}
				break
			}
		}
	}
	if e.sendToPlayer != nil {
		e.sendToPlayer(player.FirstName, []string{fmt.Sprintf("You set %s down.", carriedName)})
	}
}

// breakCarryAsCarried ends the carry relationship when the carried player is
// the one whose state/action broke it (un-submitting, or acting on their
// own), notifying the carrier.
func (e *GameEngine) breakCarryAsCarried(ctx context.Context, player *Player, carrierNotice string) {
	if player.CarriedBy == "" {
		return
	}
	carrierName := player.CarriedBy
	player.CarriedBy = ""
	if player.Dead {
		player.Position = 2 // laying down
	}
	e.SavePlayer(ctx, player)
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.FirstName == carrierName {
				p.Carrying = ""
				e.SavePlayer(ctx, p)
				if e.sendToPlayer != nil && carrierNotice != "" {
					e.sendToPlayer(p.FirstName, []string{carrierNotice})
				}
				break
			}
		}
	}
}
