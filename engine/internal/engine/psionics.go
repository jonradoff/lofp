package engine

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// PsiDiscipline defines a psionic discipline.
type PsiDiscipline struct {
	ID       int
	Name     string
	School   string // "Mind over Matter" or "Mind over Mind"
	Level    int
	PsiCost  int
	RoundSec int    // roundtime seconds
	Effect   string // "damage", "defense", "heal", "buff", "utility"
	DmgMin   int
	DmgMax   int
	DmgType  string
	DefBonus int
	HealMin  int
	HealMax  int
}

var psiRegistry []PsiDiscipline

func init() {
	mom := []PsiDiscipline{
		{1, "Kinetic Thrust", "Mind over Matter", 1, 3, 5, "damage", 4, 15, "crushing", 0, 0, 0},
		{2, "Levitate", "Mind over Matter", 2, 4, 5, "buff", 0, 0, "", 0, 0, 0},
		{3, "Pyrokinetics", "Mind over Matter", 3, 6, 5, "damage", 8, 25, "heat", 0, 0, 0},
		{4, "Cryokinetics", "Mind over Matter", 4, 8, 5, "damage", 8, 25, "cold", 0, 0, 0},
		{5, "Capacitance", "Mind over Matter", 5, 2, 5, "utility", 0, 0, "", 0, 0, 0},
		{6, "Electrify", "Mind over Matter", 6, 10, 5, "damage", 10, 30, "electric", 0, 0, 0},
		{7, "Strengthen Steel", "Mind over Matter", 7, 8, 5, "buff", 0, 0, "", 0, 0, 0},
		{8, "Manipulate Lock", "Mind over Matter", 8, 6, 5, "utility", 0, 0, "", 0, 0, 0},
		{9, "Wall of Force", "Mind over Matter", 9, 12, 5, "defense", 0, 0, "", 25, 0, 0},
		{10, "Flight", "Mind over Matter", 10, 10, 5, "buff", 0, 0, "", 0, 0, 0},
		{11, "Call on Inner Power", "Mind over Matter", 13, 0, 20, "utility", 0, 0, "", 0, 0, 0},
		{12, "Teleportation", "Mind over Matter", 18, 15, 5, "utility", 0, 0, "", 0, 0, 0},
		{13, "Force Field", "Mind over Matter", 20, 20, 5, "defense", 0, 0, "", 75, 0, 0},
		{14, "Immobilize", "Mind over Matter", 22, 18, 5, "utility", 0, 0, "", 0, 0, 0},
	}
	moo := []PsiDiscipline{
		{50, "Telepathy", "Mind over Mind", 1, 2, 5, "utility", 0, 0, "", 0, 0, 0},
		{51, "Contact", "Mind over Mind", 2, 3, 5, "utility", 0, 0, "", 0, 0, 0},
		{52, "Psychic Probe", "Mind over Mind", 3, 5, 5, "utility", 0, 0, "", 0, 0, 0},
		{53, "Psychic Blast", "Mind over Mind", 4, 8, 0, "damage", 5, 20, "", 0, 0, 0},
		{54, "Psychic Screen", "Mind over Mind", 5, 6, 5, "defense", 0, 0, "", 15, 0, 0},
		{55, "Psychic Crush", "Mind over Mind", 11, 14, 0, "damage", 10, 35, "", 0, 0, 0},
		{56, "Terror", "Mind over Mind", 15, 16, 0, "damage", 8, 25, "", 0, 0, 0},
		{57, "Psychic Shield", "Mind over Mind", 10, 10, 5, "defense", 0, 0, "", 25, 0, 0},
		{58, "Psychic Barrier", "Mind over Mind", 16, 14, 5, "defense", 0, 0, "", 35, 0, 0},
		{59, "Confuse", "Mind over Mind", 13, 12, 0, "utility", 0, 0, "", 0, 0, 0},
		{60, "Focus Skill", "Mind over Mind", 14, 8, 5, "buff", 0, 0, "", 0, 0, 0},
		{61, "Domination", "Mind over Mind", 18, 20, 5, "utility", 0, 0, "", 0, 0, 0},
		{62, "Disruption", "Mind over Mind", 19, 16, 0, "utility", 0, 0, "", 0, 0, 0},
		{63, "Psychic Fortress", "Mind over Mind", 20, 18, 5, "defense", 0, 0, "", 50, 0, 0},
		{64, "Warp Mind", "Mind over Mind", 22, 22, 0, "utility", 0, 0, "", 0, 0, 0},
		{65, "Pain", "Mind over Mind", 21, 20, 0, "damage", 20, 50, "", 0, 0, 0},
	}
	psiRegistry = append(psiRegistry, mom...)
	psiRegistry = append(psiRegistry, moo...)
}

// FindPsiByName finds a discipline by prefix match.
func FindPsiByName(input string) *PsiDiscipline {
	input = strings.ToLower(input)
	for i := range psiRegistry {
		if strings.ToLower(psiRegistry[i].Name) == input {
			return &psiRegistry[i]
		}
	}
	var match *PsiDiscipline
	for i := range psiRegistry {
		if strings.HasPrefix(strings.ToLower(psiRegistry[i].Name), input) {
			if match != nil {
				return nil
			}
			match = &psiRegistry[i]
		}
	}
	return match
}

// FindPsiByID returns a discipline by ID.
func FindPsiByID(id int) *PsiDiscipline {
	for i := range psiRegistry {
		if psiRegistry[i].ID == id {
			return &psiRegistry[i]
		}
	}
	return nil
}

// doPreparePsi handles PSI [discipline#|name].
// With no args: lists known disciplines and active maintained powers.
// With a number or name: activates the discipline. For maintained powers, toggles on/off.
func (e *GameEngine) doPreparePsi(player *Player, args []string) *CommandResult {
	if player.Dead {
		return &CommandResult{Messages: []string{"You can't use psionics while dead."}}
	}

	// Check base psionic skill
	if player.Skills[26] == 0 && !player.IsGM {
		return &CommandResult{Messages: []string{"You have no training in Psionics."}}
	}

	// Initialize ActivePsi map if nil
	if player.ActivePsi == nil {
		player.ActivePsi = make(map[int]bool)
	}

	// No args: list disciplines
	if len(args) == 0 {
		return e.listPsiDisciplines(player)
	}

	// Try to find discipline by number first, then by name
	input := strings.Join(args, " ")
	var disc *PsiDiscipline
	var discNum int
	if _, err := fmt.Sscanf(args[0], "%d", &discNum); err == nil {
		disc = FindPsiByID(discNum)
	}
	if disc == nil {
		disc = FindPsiByName(input)
	}
	if disc == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("You don't know a discipline called '%s'.", input)}}
	}

	// Check if this is a maintained power that's already active — toggle off
	isMaintained := disc.Effect == "buff" || disc.Effect == "defense"
	if isMaintained && player.ActivePsi[disc.ID] {
		delete(player.ActivePsi, disc.ID)
		// Remove defense bonus if applicable
		if disc.DefBonus > 0 {
			player.DefenseBonus -= disc.DefBonus
		}
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You release your concentration on %s.", disc.Name)},
			RoomBroadcast: []string{fmt.Sprintf("%s relaxes %s concentration.", player.FirstName, player.Possessive())},
		}
	}

	if player.Psi < disc.PsiCost {
		return &CommandResult{Messages: []string{fmt.Sprintf("Not enough psi points. (%s costs %d, you have %d)", disc.Name, disc.PsiCost, player.Psi)}}
	}

	// Instantaneous powers that don't need a target activate immediately
	if isMaintained {
		player.Psi -= disc.PsiCost
		player.ActivePsi[disc.ID] = true
		if disc.DefBonus > 0 {
			player.DefenseBonus += disc.DefBonus
		}
		var msg string
		switch disc.ID {
		case 2: // Levitate
			msg = "You feel yourself becoming lighter."
		case 7: // Strengthen Steel
			msg = "You focus your will into your weapon."
		case 10: // Flight
			msg = "You rise into the air."
			player.Position = 4 // flying
		case 9: // Wall of Force
			msg = "A shimmering wall of force surrounds you."
		case 13: // Force Field
			msg = "A powerful force field envelops you."
		case 54: // Psychic Screen
			msg = "A psychic screen forms around your mind."
		case 57: // Psychic Shield
			msg = "A psychic shield protects your mind."
		case 58: // Psychic Barrier
			msg = "A psychic barrier reinforces your defenses."
		case 60: // Focus Skill
			msg = "You sharpen your mental focus."
		case 63: // Psychic Fortress
			msg = "An impenetrable psychic fortress surrounds your mind."
		default:
			msg = fmt.Sprintf("You activate %s.", disc.Name)
		}
		result := &CommandResult{
			Messages:      []string{msg},
			RoomBroadcast: []string{fmt.Sprintf("%s concentrates intently.", player.FirstName)},
		}
		if disc.RoundSec > 0 {
			player.RoundTimeExpiry = time.Now().Add(time.Duration(disc.RoundSec) * time.Second)
			result.Messages = append(result.Messages, fmt.Sprintf(" [Round: %d sec]", disc.RoundSec))
		}
		return result
	}

	// Offensive/utility powers that need a target: prepare for PROJECT
	player.PreparedPsi = disc.ID
	if disc.RoundSec > 0 {
		player.RoundTimeExpiry = time.Now().Add(time.Duration(disc.RoundSec) * time.Second)
	}

	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You focus your mind on %s... (type PROJECT to release, or PROJECT <target>)", disc.Name)},
		RoomBroadcast: []string{fmt.Sprintf("%s closes %s eyes in concentration.", player.FirstName, player.Possessive())},
	}
}

// listPsiDisciplines shows all known disciplines and their active status.
func (e *GameEngine) listPsiDisciplines(player *Player) *CommandResult {
	var msgs []string
	msgs = append(msgs, "Your psionic disciplines:")
	msgs = append(msgs, "")

	// Determine which disciplines the player knows based on skill levels
	momLevel := player.Skills[28] // Mind over Matter
	mooLevel := player.Skills[27] // Mind over Mind
	psiLevel := player.Skills[26] // base Psionics

	if psiLevel == 0 && !player.IsGM {
		return &CommandResult{Messages: []string{"You have no training in Psionics."}}
	}

	msgs = append(msgs, fmt.Sprintf("%-4s %-25s %-6s %-6s %s", "#", "Discipline", "Cost", "School", "Status"))
	msgs = append(msgs, fmt.Sprintf("%-4s %-25s %-6s %-6s %s", "--", "----------", "----", "------", "------"))

	for _, disc := range psiRegistry {
		// Check if player has enough skill to know this discipline
		var skillLevel int
		if disc.School == "Mind over Matter" {
			skillLevel = momLevel
		} else {
			skillLevel = mooLevel
		}
		if skillLevel < disc.Level && !player.IsGM {
			continue
		}

		status := ""
		if player.ActivePsi != nil && player.ActivePsi[disc.ID] {
			status = "[ACTIVE]"
		}
		if player.PreparedPsi == disc.ID {
			status = "[PREPARED]"
		}

		schoolAbbr := "MoMat"
		if disc.School == "Mind over Mind" {
			schoolAbbr = "MoMnd"
		}

		msgs = append(msgs, fmt.Sprintf("%-4d %-25s %-6d %-6s %s", disc.ID, disc.Name, disc.PsiCost, schoolAbbr, status))
	}

	return &CommandResult{Messages: msgs}
}

// doProjectPsi handles PROJECT [target].
func (e *GameEngine) doProjectPsi(ctx context.Context, player *Player, args []string) *CommandResult {
	if player.Dead {
		return &CommandResult{Messages: []string{"You can't use psionics while dead."}}
	}
	if player.PreparedPsi == 0 {
		return &CommandResult{Messages: []string{"You have no discipline prepared. Use PSI <discipline> first."}}
	}

	disc := FindPsiByID(player.PreparedPsi)
	if disc == nil {
		player.PreparedPsi = 0
		return &CommandResult{Messages: []string{"Your focus dissipates."}}
	}

	if player.Psi < disc.PsiCost {
		player.PreparedPsi = 0
		return &CommandResult{Messages: []string{fmt.Sprintf("Not enough psi points! (%s requires %d, you have %d)", disc.Name, disc.PsiCost, player.Psi)}}
	}

	// Check roundtime
	if disc.RoundSec > 0 && player.RoundTimeExpiry.After(time.Now()) {
		remaining := player.RoundTimeExpiry.Sub(time.Now()).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You are still focusing... %.0f seconds remaining.", remaining+0.5)}}
	}

	// Deduct psi
	player.Psi -= disc.PsiCost
	player.PreparedPsi = 0

	// Skill check
	psiSkill := player.Skills[26]
	schoolSkill := 0
	if disc.School == "Mind over Matter" {
		schoolSkill = player.Skills[28]
	} else {
		schoolSkill = player.Skills[27]
	}
	castChance := 50 + (psiSkill+schoolSkill)*3 + player.Willpower/5 - disc.Level*2
	if castChance < 15 {
		castChance = 15
	}
	if player.IsGM {
		castChance = 100
	}
	if rand.Intn(100) >= castChance {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You project %s but fail to focus the energy!", disc.Name)},
			RoomBroadcast: []string{fmt.Sprintf("%s concentrates intensely but nothing happens.", player.FirstName)},
		}
	}

	result := &CommandResult{}

	switch disc.Effect {
	case "damage":
		result = e.projectDamage(player, disc, args)
	case "defense":
		player.DefenseBonus += disc.DefBonus
		result.Messages = []string{fmt.Sprintf("You project %s! (+%d defense)", disc.Name, disc.DefBonus)}
		result.RoomBroadcast = []string{fmt.Sprintf("%s concentrates and a shimmering barrier appears.", player.FirstName)}
	case "buff":
		result = e.projectBuff(player, disc)
	default:
		result.Messages = []string{fmt.Sprintf("You project %s.", disc.Name)}
		result.RoomBroadcast = []string{fmt.Sprintf("%s concentrates intensely.", player.FirstName)}
	}

	if disc.RoundSec > 0 {
		player.RoundTimeExpiry = time.Now().Add(time.Duration(disc.RoundSec) * time.Second)
	}
	e.SavePlayer(ctx, player)

	return result
}

func (e *GameEngine) projectDamage(player *Player, disc *PsiDiscipline, args []string) *CommandResult {
	targetName := ""
	if len(args) > 0 {
		targetName = strings.Join(args, " ")
	} else if player.CombatTarget != nil && player.CombatTarget.IsMonster {
		e.monsterMgr.mu.RLock()
		for _, inst := range e.monsterMgr.instances {
			if inst.ID == player.CombatTarget.MonsterID && inst.Alive {
				def := e.monsters[inst.DefNumber]
				if def != nil {
					targetName = def.Name
				}
			}
		}
		e.monsterMgr.mu.RUnlock()
	}
	if targetName == "" {
		return &CommandResult{Messages: []string{"Project at what? Specify a target."}}
	}

	inst, def := e.findMonsterInRoom(player, targetName)
	if inst == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' here.", targetName)}}
	}
	name := FormatMonsterName(def, e.monAdjs)

	dmg := rand.Intn(disc.DmgMax-disc.DmgMin+1) + disc.DmgMin

	// Psi resistance
	resist := def.PsiResist
	if resist <= 0 {
		resist = def.MagicResist
	}
	if resist > 0 && rand.Intn(100) < resist {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You project %s at a %s, but it resists!", disc.Name, name)},
			RoomBroadcast: []string{fmt.Sprintf("%s concentrates at a %s, but it resists!", player.FirstName, name)},
		}
	}

	// Elemental immunity
	if disc.DmgType != "" {
		immType := elementalImmunityType(disc.DmgType)
		if level, ok := def.Immunities[immType]; ok {
			dmg = applyImmunity(dmg, level)
		}
	}

	if dmg <= 0 {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You project %s at a %s, but it seems unaffected!", disc.Name, name)},
			RoomBroadcast: []string{fmt.Sprintf("%s concentrates at a %s!", player.FirstName, name)},
		}
	}

	killed := e.damageMonster(inst.ID, dmg)
	if killed {
		deathText := def.TextOverrides["TEXD"]
		deathMsg := fmt.Sprintf("A %s collapses, dead!", name)
		if deathText != "" {
			deathMsg = fmt.Sprintf("A %s %s", name, deathText)
		}
		e.handleMonsterDeath(player, inst, def)
		player.CombatTarget = nil
		player.Joined = false
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You project %s at a %s for %d damage!", disc.Name, name, dmg), deathMsg},
			RoomBroadcast: []string{fmt.Sprintf("%s focuses psychic energy at a %s!", player.FirstName, name), deathMsg},
		}
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You project %s at a %s for %d damage!", disc.Name, name, dmg)},
		RoomBroadcast: []string{fmt.Sprintf("%s focuses psychic energy at a %s!", player.FirstName, name)},
	}
}

func (e *GameEngine) projectBuff(player *Player, disc *PsiDiscipline) *CommandResult {
	msg := fmt.Sprintf("You project %s.", disc.Name)
	switch disc.ID {
	case 2: // Levitate
		player.CanFly = true
		msg = "You project Levitate. You feel lighter and begin to float."
	case 7: // Strengthen Steel
		msg = "You project Strengthen Steel. Your weapon gleams with psychic energy. (+15 weapon bonus)"
	case 10: // Flight
		player.CanFly = true
		msg = "You project Flight. You rise into the air!"
	case 60: // Focus Skill
		msg = "You project Focus Skill. Your mind sharpens. (+25 to next skill roll)"
	}
	return &CommandResult{
		Messages:      []string{msg},
		RoomBroadcast: []string{fmt.Sprintf("%s concentrates intensely.", player.FirstName)},
	}
}
