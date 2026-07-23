package engine

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// resolveSpecializationTarget finds the weapon the player is wielding (main
// hand first, then off-hand) whose noun name matches query. An empty query
// matches whatever is wielded in the main hand.
func (e *GameEngine) resolveSpecializationTarget(player *Player, query string) (*gameworld.ItemDef, string) {
	check := func(ii *InventoryItem) (*gameworld.ItemDef, string) {
		if ii == nil {
			return nil, ""
		}
		def := e.items[ii.Archetype]
		if def == nil {
			return nil, ""
		}
		name := strings.ToLower(e.nouns[def.NameID])
		if name == "" || (query != "" && !strings.HasPrefix(name, query)) {
			return nil, ""
		}
		return def, name
	}
	if def, name := check(player.Wielded); def != nil {
		return def, name
	}
	if def, name := check(player.OffHand); def != nil {
		return def, name
	}
	return nil, ""
}

// doSpecializeWeapon handles the SPECIALIZE command — purchase a rank of
// weapon specialization in the currently wielded weapon.
// Cost: 8 BP for rank 1, 4 BP for each additional rank (mirrors MASTER).
// Prereq: 10+ levels in the weapon's skill. Max rank: 5.
func (e *GameEngine) doSpecializeWeapon(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		var msgs []string
		for nounID, rank := range player.WeaponSpecialization {
			if rank <= 0 {
				continue
			}
			msgs = append(msgs, fmt.Sprintf("%s: rank %d/5", strings.Title(e.nouns[nounID]), rank))
		}
		if len(msgs) == 0 {
			msgs = append(msgs, "You have no weapon specializations.")
		}
		msgs = append(msgs, "Wield a weapon and type SPECIALIZE <weapon> to specialize in it.")
		return &CommandResult{Messages: msgs}
	}

	query := strings.ToLower(strings.Join(args, " "))
	weaponDef, name := e.resolveSpecializationTarget(player, query)
	if weaponDef == nil {
		return &CommandResult{Messages: []string{"You must be wielding that weapon to specialize in it."}}
	}

	skillID := weaponSkillForType(weaponDef.Type)
	if !specializableWeaponSkills[skillID] {
		return &CommandResult{Messages: []string{fmt.Sprintf("You cannot specialize in your %s.", name)}}
	}

	if player.Skills[skillID] < 10 {
		return &CommandResult{Messages: []string{fmt.Sprintf("You need at least 10 levels of %s to specialize in %s (you have %d).", SkillNames[skillID], name, player.Skills[skillID])}}
	}

	current := player.WeaponSpecialization[weaponDef.NameID]
	if current >= 5 {
		return &CommandResult{Messages: []string{fmt.Sprintf("You have fully specialized in %s (5/5).", name)}}
	}

	bpCost := 8
	if current > 0 {
		bpCost = 4
	}
	if player.BuildPoints < bpCost {
		return &CommandResult{Messages: []string{fmt.Sprintf("You need %d build points to specialize in %s (you have %d).", bpCost, name, player.BuildPoints)}}
	}

	if player.WeaponSpecialization == nil {
		player.WeaponSpecialization = make(map[int]int)
	}
	newRank := current + 1
	player.WeaponSpecialization[weaponDef.NameID] = newRank
	player.BuildPoints -= bpCost
	e.SavePlayer(ctx, player)

	msgs := []string{fmt.Sprintf("You train intensively with your %s, achieving rank %d of specialization. [-%d BP, %d BP remaining]", name, newRank, bpCost, player.BuildPoints)}
	msgs = append(msgs, fmt.Sprintf("Fatigue cost when attacking with a %s is now reduced by %d (minimum 1).", name, newRank))
	return &CommandResult{Messages: msgs, PlayerState: player}
}

// specializationTotalCost returns the cumulative BP cost to reach the given
// rank (0-5): 8 BP for rank 1, 4 BP for each additional rank.
func specializationTotalCost(rank int) int {
	if rank <= 0 {
		return 0
	}
	return 8 + (rank-1)*4
}

// resolveSpecializableWeaponNoun finds the noun ID for a weapon type by name,
// restricted to types eligible for specialization (isWeaponItemType and one of
// the specializable skills). Returns 0 if no match is found.
func (e *GameEngine) resolveSpecializableWeaponNoun(name string) (int, string) {
	name = strings.ToLower(name)
	for _, def := range e.items {
		if !specializableWeaponSkills[weaponSkillForType(def.Type)] {
			continue
		}
		if strings.ToLower(e.nouns[def.NameID]) == name {
			return def.NameID, e.nouns[def.NameID]
		}
	}
	return 0, ""
}

// gmSpecialize handles the @SPECIALIZE GM command — list or directly set a
// player's weapon specialization rank. Works like @MASTERY: list mode shows
// current ranks, set mode assigns a rank 0-5. Unlike the player-facing
// SPECIALIZE command, this bypasses the skill/BP prerequisites (GM override)
// but still spends/refunds BP for the rank change, going negative if needed.
func (e *GameEngine) gmSpecialize(ctx context.Context, args []string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Messages: []string{"Usage: @specialize <player> [<weapon name> <level>]"}}
	}
	target, err := e.resolvePlayerArg(ctx, args)
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}

	// List mode: @specialize <player>
	if len(args) < 3 {
		if len(target.WeaponSpecialization) == 0 {
			return &CommandResult{Messages: []string{fmt.Sprintf("%s has no weapon specializations.", target.FullName())}}
		}
		type specEntry struct {
			name string
			rank int
		}
		var entries []specEntry
		for nounID, rank := range target.WeaponSpecialization {
			if rank <= 0 {
				continue
			}
			entries = append(entries, specEntry{name: strings.Title(e.nouns[nounID]), rank: rank})
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
		msgs := []string{fmt.Sprintf("Weapon specializations for %s:", target.FullName())}
		for _, se := range entries {
			msgs = append(msgs, fmt.Sprintf("  %-26s rank %d / 5", se.name, se.rank))
		}
		return &CommandResult{Messages: msgs}
	}

	// Set mode: @specialize <player> <weapon name> <level>
	nounID, weaponName := e.resolveSpecializableWeaponNoun(args[1])
	if nounID == 0 {
		return &CommandResult{Messages: []string{fmt.Sprintf("Unknown specializable weapon: %s", args[1])}}
	}
	level, err := strconv.Atoi(args[2])
	if err != nil || level < 0 || level > 5 {
		return &CommandResult{Messages: []string{"Level must be 0-5."}}
	}
	if target.WeaponSpecialization == nil {
		target.WeaponSpecialization = make(map[int]int)
	}
	oldRank := target.WeaponSpecialization[nounID]

	// Spend BP when increasing, refund when decreasing. As a GM override, this
	// still applies even if it takes the player's BP below zero.
	bpDelta := specializationTotalCost(level) - specializationTotalCost(oldRank)
	target.BuildPoints -= bpDelta

	if level == 0 {
		delete(target.WeaponSpecialization, nounID)
	} else {
		target.WeaponSpecialization[nounID] = level
	}
	e.SavePlayer(ctx, target)

	displayName := strings.Title(weaponName)
	msg := fmt.Sprintf("Set %s specialization for %s to rank %d (max 5).", displayName, target.FullName(), level)
	switch {
	case bpDelta > 0:
		msg += fmt.Sprintf(" Spent %d BP (%d BP remaining).", bpDelta, target.BuildPoints)
	case bpDelta < 0:
		msg += fmt.Sprintf(" Refunded %d BP (%d BP total).", -bpDelta, target.BuildPoints)
	}
	return &CommandResult{Messages: []string{msg}}
}
