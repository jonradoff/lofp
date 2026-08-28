package engine

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// --- Invisibility / Mass Invisibility / Phantom Form (225, 212, 248) ---
//
// All three have no duration — they last until the target smiles, casts a
// spell (preparing doesn't count, only the actual cast), or attacks a player
// or monster (see the SMILE handling in emotes.go, the reveal-on-cast check in
// doCastSpell, and the reveal-on-attack check in doAttackMonster). Phantom Form
// behaves identically to Invisible for every mechanical purpose (room listings,
// monster targeting, etc. — see the PhantomForm checks alongside Hidden/
// Invisible throughout combat.go/look.go) except that See Hidden (405) shows
// "a shimmering grey form" instead of revealing the caster's real name.

// castHidingSpell handles Invisibility (225), Mass Invisibility (212), and
// Phantom Form (248): self by default, or a named player in the room —
// matching the self-or-target pattern used by Camouflage/Mass Protection.
func (e *GameEngine) castHidingSpell(player *Player, spell *SpellDef, args []string) *CommandResult {
	target := player
	isSelf := true
	if len(args) > 0 {
		t := strings.ToLower(strings.Join(args, " "))
		if t != "me" && t != "myself" && t != "self" {
			found := e.findPlayerInRoom(player, t)
			if found == nil {
				return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' here.", strings.Join(args, " "))}}
			}
			target = found
			isSelf = false
		}
	}

	if spell.ID == 248 {
		target.PhantomForm = true
	} else {
		target.Invisible = true
	}

	if isSelf {
		// "You gesture." as its own message (rather than folded into one combined
		// sentence) matches every other buff spell's format and is what lets the
		// success-roll line get inserted in the right place — see the
		// "You gesture." prefix check around line 1122 in spells.go. Per the
		// original session logs, the caster feels a tingling sensation rather than
		// being told they've faded — they can't see themselves fade any more than
		// anyone else can once they're invisible.
		return &CommandResult{
			Messages:      []string{"You gesture.", "You feel a tingling sensation."},
			RoomBroadcast: []string{fmt.Sprintf("%s fades from sight.", player.DisplayNameCap())},
		}
	}
	e.SavePlayer(context.Background(), target)
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s. %s fades from sight.", spell.Name, target.FirstName, target.DisplayNameCap())},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures at %s, who fades from sight.", player.DisplayName(), target.DisplayName())},
		TargetName:    target.FirstName,
		TargetMsg:     []string{fmt.Sprintf("%s casts %s on you. You fade from sight.", player.FirstName, spell.Name)},
	}
}

// --- See Hidden (405) ---

// castSeeHiddenSpell handles See Hidden (405): reveals every other player in
// the room currently Hidden, Invisible, or in Phantom Form. Hidden players show
// as "You see something.", invisible players show their effective display name
// (real name, or "a wolf"/"some mist"/"a slime"/a disguise name if applicable —
// same as DisplayName elsewhere), and Phantom Form always shows as "You see a
// shimmering grey form." regardless of the caster's true identity. GM @invis is
// not revealed — that's an out-of-game admin tool, not something a spell should
// pierce.
func (e *GameEngine) castSeeHiddenSpell(player *Player, spell *SpellDef) *CommandResult {
	roomMsg := fmt.Sprintf("%s narrows %s eyes and gazes intently about the area.", player.DisplayNameCap(), player.Possessive())
	result := &CommandResult{
		Messages:      []string{"You narrow your eyes and gaze about the area."},
		RoomBroadcast: []string{roomMsg},
	}

	if e.sessions == nil {
		return result
	}
	for _, p := range e.sessions.OnlinePlayers() {
		if p.RoomNumber != player.RoomNumber || p == player || p.Dead {
			continue
		}
		if p.GMInvis {
			continue
		}
		switch {
		case p.PhantomForm:
			result.Messages = append(result.Messages, "You see a shimmering grey form.")
		case p.Hidden:
			result.Messages = append(result.Messages, "You see something.")
		case p.Invisible:
			result.Messages = append(result.Messages, fmt.Sprintf("You see %s.", p.DisplayName()))
		}
	}
	if len(result.Messages) == 1 {
		result.Messages = append(result.Messages, "You sense nothing hidden nearby.")
	}
	return result
}

// --- Mist Form (232) / Slime Form (245) ---

// castFormSpell handles Mist Form (232) and Slime Form (245): self-only
// transforms that last until the player types TRANSFORM to return to normal
// (see the TRANSFORM case in engine.go). While active the player cannot
// attack, defend, cast, speak, wear/remove items, use magic items, or emote —
// see IsFormLocked/formActionBlockMessage in player.go for the gates. Mist
// Form grants full immunity to physical/magical damage and the ability to
// fly/ascend/descend; Slime Form only reduces damage taken by 90% (see
// formDamageReduction). Successfully casting either spell immediately breaks
// any bond effect (Plant Snare, Imprison) since shifting shape lets the
// caster slip free.
func (e *GameEngine) castFormSpell(player *Player, spell *SpellDef) *CommandResult {
	if player.MistForm || player.SlimeForm || player.WolfForm {
		return &CommandResult{Messages: []string{"You are already in another form."}}
	}

	player.Entangles = nil
	player.ImprisonedExpiry = time.Time{}

	var bodyMsg string
	switch spell.ID {
	case 232:
		player.MistForm = true
		bodyMsg = "You feel your body changing shape. Suddenly, you realize that you are a cloud of mist! [Type TRANSFORM to assume a material shape again]"
	default: // 245
		player.SlimeForm = true
		bodyMsg = "You feel your body changing shape. Suddenly, you realize that you are a slime! [Type TRANSFORM to assume a material shape again]"
	}

	return &CommandResult{
		Messages:      []string{"You gesture.", bodyMsg},
		RoomBroadcast: []string{fmt.Sprintf("%s shudders and transforms into %s!", player.FirstName, player.DisplayName())},
	}
}

// formDamageReduction adjusts incoming physical/magical damage for a player
// currently in Mist Form or Slime Form. Mist Form grants full immunity (the
// attack passes harmlessly through); Slime Form only reduces damage to 10% of
// what was rolled (90% damage reduction). Returns the target unchanged for
// anyone in neither form.
func formDamageReduction(p *Player, dmg int) int {
	if p.MistForm {
		return 0
	}
	if p.SlimeForm {
		return dmg / 10
	}
	return dmg
}
