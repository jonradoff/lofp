package engine

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
)

// SkillCost defines the build point cost for a skill.
// First = cost of first rank, PerRank = cost per additional rank.
type SkillCost struct {
	First   int
	PerRank int
}

// SkillCosts maps skill ID to build point costs (from skills.txt).
var SkillCosts = map[int]SkillCost{
	0:  {10, 5}, // Jeweler
	1:  {10, 4}, // Two Weapons
	2:  {12, 5}, // Backstab
	3:  {12, 5}, // Missile Weapons
	4:  {10, 3}, // Natural Weapons (Claws)
	5:  {6, 3},  // Climbing
	6:  {8, 4},  // Dodging & Parrying
	7:  {10, 5}, // Conjuration
	8:  {10, 5}, // Weaponsmithing
	9:  {12, 5}, // Crushing Weapons
	10: {10, 5}, // Combat Maneuvering
	11: {8, 4},  // Endurance
	12: {6, 3},  // Trap & Poison Lore
	13: {12, 5}, // Edged Weapons
	14: {10, 5}, // Enchantment
	15: {8, 4},  // Dyeing/Weaving
	16: {12, 5}, // Drakin Weapons
	17: {10, 5}, // Druidic Magic
	18: {8, 3},  // Wood Lore
	19: {12, 5}, // Thrown Weapons
	20: {20, 2}, // Healing
	21: {12, 4}, // Legerdemain
	22: {10, 4}, // Lockpicking
	23: {20, 5}, // Spellcraft
	24: {12, 5}, // Martial Arts
	25: {12, 5}, // Polearms
	26: {20, 5}, // Psionics
	27: {10, 5}, // Mind over Mind
	28: {10, 5}, // Mind over Matter
	29: {10, 2}, // Transcendence
	30: {10, 5}, // Necromancy
	31: {15, 5}, // Alchemy
	32: {5, 3},  // Sagecraft
	33: {10, 4}, // Stealth
	34: {15, 10}, // Disguise
	35: {8, 4},  // Mining
}

// skillBPCost returns the build point cost for training to the next rank.
func skillBPCost(skillID, currentRank int) int {
	cost, ok := SkillCosts[skillID]
	if !ok {
		return 5 // default
	}
	if currentRank == 0 {
		return cost.First
	}
	return cost.PerRank
}

// SkillPrerequisites maps skill ID to required prerequisite skill IDs.
// Player must have at least 1 rank in each prerequisite.
var SkillPrerequisites = map[int][]int{
	6:  {13, 16, 9, 4, 24, 3, 25}, // Dodge: any one weapon skill (OR logic)
	7:  {23},                        // Conjuration requires Spellcraft
	14: {23},                        // Enchantment requires Spellcraft
	17: {23},                        // Druidic requires Spellcraft
	30: {23},                        // Necromancy requires Spellcraft
	27: {26},                        // Mind over Mind requires Psionics
	28: {26},                        // Mind over Matter requires Psionics
	34: {33},                        // Disguise requires Stealth
}

// checkPrerequisite returns true if the player meets prerequisites for a skill.
// For Dodge/Parry (skill 6), any ONE of the weapon skills suffices (OR logic).
func checkPrerequisite(player *Player, skillID int) bool {
	prereqs, ok := SkillPrerequisites[skillID]
	if !ok {
		return true // no prerequisites
	}
	if skillID == 6 {
		// Dodge: need at least one weapon skill
		for _, p := range prereqs {
			if player.Skills[p] > 0 {
				return true
			}
		}
		return false
	}
	// Standard: need all prerequisites
	for _, p := range prereqs {
		if player.Skills[p] < 1 {
			return false
		}
	}
	return true
}

// ---- TRAIN command (with BP costs and prerequisites) ----

func (e *GameEngine) doTrainWithBP(ctx context.Context, player *Player, args []string) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil || len(room.TrainingSkills) == 0 {
		return &CommandResult{Messages: []string{"There is no training available here."}}
	}
	if len(args) == 0 {
		var msgs []string
		msgs = append(msgs, "Training available here:")
		for _, ts := range room.TrainingSkills {
			name := SkillNames[ts.SkillID]
			if name == "" {
				name = fmt.Sprintf("Skill #%d", ts.SkillID)
			}
			currentLvl := player.Skills[ts.SkillID]
			bpCost := skillBPCost(ts.SkillID, currentLvl)
			msgs = append(msgs, fmt.Sprintf("  %s (rank %d/%d, next: %d BP)", name, currentLvl, ts.MaxLevel, bpCost))
		}
		msgs = append(msgs, fmt.Sprintf("Your build points: %d", player.BuildPoints))
		return &CommandResult{Messages: msgs}
	}

	target := strings.ToLower(strings.Join(args, " "))
	for _, ts := range room.TrainingSkills {
		name := SkillNames[ts.SkillID]
		if !strings.HasPrefix(strings.ToLower(name), target) {
			continue
		}
		if player.Skills == nil {
			player.Skills = make(map[int]int)
		}
		currentLvl := player.Skills[ts.SkillID]
		if currentLvl >= ts.MaxLevel {
			return &CommandResult{Messages: []string{fmt.Sprintf("You have reached the maximum %s training available here (%d).", name, ts.MaxLevel)}}
		}

		// Check prerequisites
		if !checkPrerequisite(player, ts.SkillID) {
			prereqNames := ""
			for _, p := range SkillPrerequisites[ts.SkillID] {
				if prereqNames != "" {
					prereqNames += ", "
				}
				prereqNames += SkillNames[p]
			}
			return &CommandResult{Messages: []string{fmt.Sprintf("You need training in %s before you can learn %s.", prereqNames, name)}}
		}

		// Check build points
		bpCost := skillBPCost(ts.SkillID, currentLvl)
		if player.BuildPoints < bpCost {
			return &CommandResult{Messages: []string{fmt.Sprintf("Not enough build points. %s costs %d BP (you have %d).", name, bpCost, player.BuildPoints)}}
		}

		// Gold cost: 5 gold per training level after level 4
		goldCost := 0
		if player.Level > 4 {
			goldCost = 5 * (currentLvl + 1)
		}
		if goldCost > 0 && player.Gold < goldCost {
			return &CommandResult{Messages: []string{fmt.Sprintf("Training costs %d gold crowns. You only have %d.", goldCost, player.Gold)}}
		}

		// Deduct costs
		player.BuildPoints -= bpCost
		if goldCost > 0 {
			player.Gold -= goldCost
		}
		player.Skills[ts.SkillID] = currentLvl + 1

		// Apply Endurance bonus: +4 body points per rank
		if ts.SkillID == 11 {
			player.MaxBodyPoints += 4
			player.BodyPoints += 4
		}

		e.SavePlayer(ctx, player)

		goldMsg := ""
		if goldCost > 0 {
			goldMsg = fmt.Sprintf(", %d gold", goldCost)
		}
		return &CommandResult{
			Messages: []string{fmt.Sprintf("You train in %s to rank %d. (-%d BP%s, %d BP remaining)", name, currentLvl+1, bpCost, goldMsg, player.BuildPoints)},
			PlayerState: player,
		}
	}
	return &CommandResult{Messages: []string{"That skill is not available for training here."}}
}

// ---- ANOINT (apply poison to weapon) ----

func (e *GameEngine) doAnoint(ctx context.Context, player *Player, args []string) *CommandResult {
	trapSkill := player.Skills[12] // Trap & Poison Lore
	if trapSkill < 1 {
		return &CommandResult{Messages: []string{"You have no training in Trap & Poison Lore."}}
	}
	if player.Wielded == nil {
		return &CommandResult{Messages: []string{"You must be wielding a weapon to anoint it."}}
	}
	// Poison level = trap skill rank
	poisonLevel := trapSkill
	if poisonLevel > 50 {
		poisonLevel = 50
	}
	// Set VAL4 on wielded weapon (51-100 = poison level 1-50)
	player.Wielded.Val4 = 50 + poisonLevel
	e.SavePlayer(ctx, player)

	wepDef := e.items[player.Wielded.Archetype]
	wepName := "weapon"
	if wepDef != nil {
		wepName = e.getItemNounName(wepDef)
	}
	return &CommandResult{
		Messages: []string{fmt.Sprintf("You carefully apply a level %d poison to your %s.", poisonLevel, wepName)},
		RoomBroadcast: []string{fmt.Sprintf("%s applies something to %s weapon.", player.FirstName, player.Possessive())},
		PlayerState: player,
	}
}

// ---- TEND (Healing skill) ----

func (e *GameEngine) doTend(ctx context.Context, player *Player, args []string) *CommandResult {
	healSkill := player.Skills[20] // Healing
	if healSkill < 1 {
		return &CommandResult{Messages: []string{"You have no training in Healing."}}
	}

	// Determine target
	target := player
	targetName := "yourself"
	if len(args) > 0 {
		t := strings.ToLower(strings.Join(args, " "))
		if t != "me" && t != "myself" && t != "self" {
			found := e.findPlayerInRoom(player, t)
			if found != nil {
				target = found
				targetName = found.FirstName
			}
		}
	}

	if target.BodyPoints >= target.MaxBodyPoints && !target.Bleeding && !target.Poisoned {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s doesn't need healing.", targetName)}}
	}

	// Healing amount: 2 + healSkill*2 + random(healSkill)
	heal := 2 + healSkill*2 + rand.Intn(healSkill+1)

	// Same race bonus: +50%
	if target.Race == player.Race {
		heal = heal * 3 / 2
	}

	target.BodyPoints += heal
	if target.BodyPoints > target.MaxBodyPoints {
		target.BodyPoints = target.MaxBodyPoints
	}

	// Stop bleeding
	if target.Bleeding {
		target.Bleeding = false
	}

	e.SavePlayer(ctx, player)

	if target == player {
		return &CommandResult{
			Messages: []string{fmt.Sprintf("You tend to your wounds, healing %d body points. [BP: %d/%d]", heal, target.BodyPoints, target.MaxBodyPoints)},
			RoomBroadcast: []string{fmt.Sprintf("%s tends to %s wounds.", player.FirstName, player.Possessive())},
			PlayerState: player,
		}
	}

	return &CommandResult{
		Messages: []string{fmt.Sprintf("You tend to %s's wounds, healing %d body points.", targetName, heal)},
		RoomBroadcast: []string{fmt.Sprintf("%s tends to %s's wounds.", player.FirstName, targetName)},
		TargetName: target.FirstName,
		TargetMsg: []string{fmt.Sprintf("%s tends to your wounds, healing %d body points. [BP: %d/%d]", player.FirstName, heal, target.BodyPoints, target.MaxBodyPoints)},
	}
}
