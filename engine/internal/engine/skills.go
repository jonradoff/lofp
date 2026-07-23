package engine

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
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

	// Collect effective skill offers: room trainers boosted/extended by player teachers.
	// roomMaxLevel tracks how far the room's own (paid) trainer goes; ranks above
	// that — whether from a teacher extending an existing offer or a teacher
	// providing a skill the room doesn't train at all — are taught by a fellow
	// player and are free of the gold charge.
	type skillOffer struct {
		skillID      int
		maxLevel     int
		roomMaxLevel int
	}
	var offers []skillOffer
	if room != nil {
		for _, ts := range room.TrainingSkills {
			offers = append(offers, skillOffer{ts.SkillID, ts.MaxLevel, ts.MaxLevel})
		}
	}

	// Find any player teacher in this room.
	var spellTeacher *Player
	var taughtSpell *SpellDef
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.ID == player.ID || p.RoomNumber != player.RoomNumber {
				continue
			}
			if p.Teaching > 0 {
				found := false
				for i, o := range offers {
					if o.skillID == p.Teaching {
						found = true
						if p.TeachingLevel > offers[i].maxLevel {
							offers[i].maxLevel = p.TeachingLevel
						}
						break
					}
				}
				if !found {
					offers = append(offers, skillOffer{p.Teaching, p.TeachingLevel, 0})
				}
			}
			if p.TeachingSpell > 0 && spellTeacher == nil {
				sp := FindSpellByID(p.TeachingSpell)
				if sp != nil {
					spellTeacher = p
					taughtSpell = sp
				}
			}
		}
	}

	if len(args) == 0 {
		if len(offers) == 0 && spellTeacher == nil {
			return &CommandResult{Messages: []string{"There is no training available here."}}
		}
		var msgs []string
		if len(offers) > 0 {
			msgs = append(msgs, "Training available here:")
			for _, o := range offers {
				name := SkillNames[o.skillID]
				if name == "" {
					name = fmt.Sprintf("Skill #%d", o.skillID)
				}
				currentLvl := player.Skills[o.skillID]
				bpCost := skillBPCost(o.skillID, currentLvl)
				msgs = append(msgs, fmt.Sprintf("  %s (rank %d/%d, next: %d BP)", name, currentLvl, o.maxLevel, bpCost))
			}
		}
		if spellTeacher != nil {
			msgs = append(msgs, fmt.Sprintf("Spell being taught: %s is teaching \"%s\" (Number %d, level %d).",
				spellTeacher.FirstName, taughtSpell.Name, taughtSpell.ID, taughtSpell.Level))
		}
		msgs = append(msgs, fmt.Sprintf("Your build points: %d", player.BuildPoints))
		return &CommandResult{Messages: msgs}
	}

	target := strings.ToLower(strings.Join(args, " "))

	// Check for spell training from player teacher first.
	if spellTeacher != nil {
		numStr := fmt.Sprintf("%d", taughtSpell.ID)
		if target == numStr || strings.HasPrefix(strings.ToLower(taughtSpell.Name), target) {
			return e.trainSpellFromTeacher(ctx, player, spellTeacher, taughtSpell)
		}
	}

	// Skill training.
	if len(offers) == 0 {
		return &CommandResult{Messages: []string{"There is no training available here."}}
	}

	for _, o := range offers {
		name := SkillNames[o.skillID]
		numStr := fmt.Sprintf("%d", o.skillID)
		if !strings.HasPrefix(strings.ToLower(name), target) && target != numStr {
			continue
		}
		if player.Skills == nil {
			player.Skills = make(map[int]int)
		}
		currentLvl := player.Skills[o.skillID]

		if currentLvl >= o.maxLevel {
			return &CommandResult{Messages: []string{fmt.Sprintf("You have reached the maximum %s training available here (%d).", name, o.maxLevel)}}
		}

		if !checkPrerequisite(player, o.skillID) {
			prereqNames := ""
			for _, p := range SkillPrerequisites[o.skillID] {
				if prereqNames != "" {
					prereqNames += ", "
				}
				prereqNames += SkillNames[p]
			}
			return &CommandResult{Messages: []string{fmt.Sprintf("You need training in %s before you can learn %s.", prereqNames, name)}}
		}

		bpCost := skillBPCost(o.skillID, currentLvl)
		if player.BuildPoints < bpCost {
			return &CommandResult{Messages: []string{fmt.Sprintf("Not enough build points. %s costs %d BP (you have %d).", name, bpCost, player.BuildPoints)}}
		}

		// Only ranks within the room's own posted training cap are charged gold;
		// ranks made available by a fellow player's TEACH are free.
		goldCost := 0
		if player.Level > 4 && currentLvl+1 <= o.roomMaxLevel {
			goldCost = 5 * (currentLvl + 1)
		}
		if goldCost > 0 && player.Gold < goldCost {
			return &CommandResult{Messages: []string{fmt.Sprintf("Training costs %d gold crowns. You only have %d.", goldCost, player.Gold)}}
		}

		player.BuildPoints -= bpCost
		if goldCost > 0 {
			player.Gold -= goldCost
		}
		player.Skills[o.skillID] = currentLvl + 1

		if o.skillID == 11 { // Endurance: +4 max body points per rank
			player.MaxBodyPoints += 4
			player.BodyPoints += 4
		}

		var giftMsg string
		if player.RoomNumber == 298 && currentLvl == 0 {
			if school := magicSkillSchool(o.skillID); school != "" {
				if scroll, spellName := e.guildScrollGift(school); scroll != nil {
					player.Inventory = append(player.Inventory, *scroll)
					giftMsg = fmt.Sprintf("\"Congratulations on beginning your studies in %s!\" the guildmaster says, handing you a scroll of %s. \"May it serve you well on your path.\"", name, spellName)
				}
			}
		}

		e.SavePlayer(ctx, player)

		goldMsg := ""
		if goldCost > 0 {
			goldMsg = fmt.Sprintf(", %d gold", goldCost)
		}
		msgs := []string{fmt.Sprintf("You train in %s to rank %d. (-%d BP%s, %d BP remaining)", name, currentLvl+1, bpCost, goldMsg, player.BuildPoints)}
		if giftMsg != "" {
			msgs = append(msgs, giftMsg)
		}
		return &CommandResult{
			Messages:    msgs,
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

// ---- PICK (Lockpicking skill) ----

// lockpickBreakChance returns the percentage chance a lockpick breaks on extreme failure,
// based on the material adjective name.
func lockpickBreakChance(material string) int {
	switch strings.ToLower(material) {
	case "tin", "copper":
		return 60
	case "brass", "bronze":
		return 45
	case "iron":
		return 35
	case "steel":
		return 25
	case "truesteel":
		return 10
	case "randar", "elkyri":
		return 5
	default:
		return 50
	}
}

func (e *GameEngine) doPickLock(ctx context.Context, player *Player, args []string) *CommandResult {
	lockSkill := player.Skills[22]
	if lockSkill < 1 {
		return &CommandResult{Messages: []string{"You have no training in Lockpicking."}}
	}

	raw := strings.ToLower(strings.Join(args, " "))
	containerTarget := raw

	// Strip optional "with lockpick" suffix
	if idx := strings.Index(raw, " with "); idx >= 0 {
		tool := strings.TrimSpace(raw[idx+6:])
		if strings.HasPrefix(tool, "lockpick") {
			containerTarget = strings.TrimSpace(raw[:idx])
		}
	}
	containerTarget, containerMine := stripMyPrefix(containerTarget)

	// Find a lockpick in inventory, recording its index and material adjective.
	pickIdx := -1
	pickMaterial := ""
	pickName := "lockpick"
	for i, ii := range player.Inventory {
		def := e.items[ii.Archetype]
		if def == nil || def.Type != "LOCKPICK" {
			continue
		}
		pickIdx = i
		// Material is in Adj1; fall back to Adj2 if Adj1 is non-material (e.g. colour).
		pickMaterial = strings.ToLower(e.getAdjName(ii.Adj1))
		if pickMaterial == "" {
			pickMaterial = strings.ToLower(e.getAdjName(ii.Adj2))
		}
		pickName = e.formatItemName(def, ii.Adj1, ii.Adj2, ii.Adj3, ii.Tail)
		break
	}
	if pickIdx < 0 {
		return &CommandResult{Messages: []string{"You need a lockpick to do that."}}
	}

	// tryPick returns (success bool, roll int). The roll value lets the caller
	// assess how badly a failure was (higher roll = worse fumble).
	tryPick := func(lockDiff int) (bool, int) {
		chance := 30 + lockSkill*5 + player.Agility/5 - lockDiff
		if chance < 5 {
			chance = 5
		}
		if player.IsGM {
			return true, 0
		}
		r := rand.Intn(100)
		return r < chance, r
	}

	// maybeBreakPick checks for an extreme failure (≈1-in-3 of failures) and if so
	// rolls against the material's break chance. Returns an extra message if it breaks.
	maybeBreakPick := func(roll int, chance int) string {
		// Extreme failure: rolled in the top third of the failure range.
		failureRange := 100 - chance
		if failureRange <= 0 {
			failureRange = 1
		}
		if roll < 100-failureRange/3 {
			return "" // not extreme enough
		}
		if rand.Intn(100) < lockpickBreakChance(pickMaterial) {
			player.Inventory = append(player.Inventory[:pickIdx], player.Inventory[pickIdx+1:]...)
			return fmt.Sprintf("Your %s snaps with a sharp CRACK!", pickName)
		}
		return ""
	}

	// successMsg builds the success CommandResult.
	successMsg := func(displayName string) *CommandResult {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You hear a soft CLICK as you pick the lock on %s.", displayName)},
			RoomBroadcast: []string{fmt.Sprintf("You hear a soft CLICK as %s picks the lock on %s.", player.FirstName, displayName)},
		}
	}

	// failMsg builds the failure CommandResult, appending a break message if the pick snapped.
	failMsg := func(displayName, breakMsg string) *CommandResult {
		msgs := []string{fmt.Sprintf("You fiddle with the lock on %s but can't seem to get it open.", displayName)}
		if breakMsg != "" {
			msgs = append(msgs, breakMsg)
		}
		return &CommandResult{
			Messages:      msgs,
			RoomBroadcast: []string{fmt.Sprintf("%s fiddles with a lock on %s.", player.FirstName, displayName)},
		}
	}

	// Check inventory items first — a container you're holding takes priority
	// over one merely lying in the room (matches doPut/doGetAllFromContainer).
	for i, ii := range player.Inventory {
		def := e.items[ii.Archetype]
		if def == nil || strings.ToUpper(ii.State) != "LOCKED" {
			continue
		}
		if !matchesTarget(e.getItemNounName(def), containerTarget, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
			continue
		}
		displayName := e.formatItemName(def, ii.Adj1, ii.Adj2, ii.Adj3, ii.Tail)
		chance := 30 + lockSkill*5 + player.Agility/5 - ii.Val1
		if chance < 5 {
			chance = 5
		}
		ok, roll := tryPick(ii.Val1)
		if ok {
			player.Inventory[i].State = "CLOSED"
			player.Experience += 50 * ii.Val1
			e.SavePlayer(ctx, player)
			return successMsg(displayName)
		}
		breakMsg := maybeBreakPick(roll, chance)
		e.SavePlayer(ctx, player)
		return failMsg(displayName, breakMsg)
	}

	// Room items — skipped when "my" was used (e.g. "pick lock my chest").
	room := e.rooms[player.RoomNumber]
	if room != nil && !containerMine {
		for i, ri := range room.Items {
			def := e.items[ri.Archetype]
			if def == nil || strings.ToUpper(ri.State) != "LOCKED" {
				continue
			}
			if !matchesTarget(e.getItemNounName(def), containerTarget, e.getAdjName(ri.Adj1), e.getAdjName(ri.Adj2), e.getAdjName(ri.Adj3)) {
				continue
			}
			displayName := e.formatItemName(def, ri.Adj1, ri.Adj2, ri.Adj3, ri.Extend)
			chance := 30 + lockSkill*5 + player.Agility/5 - ri.Val1
			if chance < 5 {
				chance = 5
			}
			ok, roll := tryPick(ri.Val1)
			if ok {
				room.Items[i].State = "CLOSED"
				e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "CLOSED"})
				player.Experience += 50 * ri.Val1
				e.SavePlayer(ctx, player)
				return successMsg(displayName)
			}
			breakMsg := maybeBreakPick(roll, chance)
			e.SavePlayer(ctx, player)
			return failMsg(displayName, breakMsg)
		}
	}

	return &CommandResult{Messages: []string{"You don't see anything locked here."}}
}

// ---- TEND (Healing skill) ----

func (e *GameEngine) doTend(ctx context.Context, player *Player, args []string) *CommandResult {
	healSkill := player.Skills[20] // Healing
	if healSkill < 1 {
		return &CommandResult{Messages: []string{"You have no training in Healing."}}
	}

	// Can't heal during combat
	if player.CombatTarget != nil {
		return &CommandResult{Messages: []string{"You can't tend wounds while in combat!"}}
	}

	// Round timer check
	if time.Now().Before(player.RoundTimeExpiry) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You must wait %.0f more seconds.", remaining)}}
	}

	// Determine target: self, another player, or a monster/corpse in the room
	if len(args) == 0 {
		return e.tendPlayer(ctx, player, player, "yourself", healSkill)
	}
	t := strings.ToLower(strings.Join(args, " "))
	if t == "me" || t == "myself" || t == "self" {
		return e.tendPlayer(ctx, player, player, "yourself", healSkill)
	}
	if found := e.findPlayerInRoom(player, t); found != nil {
		return e.tendPlayer(ctx, player, found, found.FirstName, healSkill)
	}
	if inst, def := e.findMonsterInRoomIncludeDead(player, t); inst != nil {
		name := FormatMonsterName(def, e.monAdjs)
		return e.tendMonster(ctx, player, inst, name, healSkill)
	}
	return &CommandResult{Messages: []string{"You don't see them here."}}
}

// selectTendableWound finds the least-severe wound in wounds and checks
// whether healSkill is high enough to treat it — wounds are always removed
// least-severe-first, and if the healer can't yet manage even that one they
// can't skip ahead to a different wound.
func selectTendableWound(wounds []Wound, healSkill int) (wound Wound, idx int, failMsg string) {
	if len(wounds) == 0 {
		return Wound{}, -1, ""
	}
	idx = 0
	for i := 1; i < len(wounds); i++ {
		if wounds[i].Level < wounds[idx].Level {
			idx = i
		}
	}
	wound = wounds[idx]
	required := (wound.Level*20 + 11) / 12 // proportional 1-12 -> 2-20; skill 20 heals any severity
	if healSkill < required {
		return Wound{}, -1, fmt.Sprintf("Your Healing skill isn't advanced enough to tend that wound yet. (need %d, have %d)", required, healSkill)
	}
	return wound, idx, ""
}

// tendPlayer handles TEND when the target is a player (self or another).
func (e *GameEngine) tendPlayer(ctx context.Context, healer, target *Player, targetName string, healSkill int) *CommandResult {
	isSelf := target == healer
	if len(target.Wounds) == 0 {
		if isSelf {
			return &CommandResult{Messages: []string{"You don't need healing."}}
		}
		return &CommandResult{Messages: []string{fmt.Sprintf("%s doesn't need healing.", targetName)}}
	}
	if target.Position != 1 && target.Position != 2 {
		if isSelf {
			return &CommandResult{Messages: []string{"You must be sitting or lying down to tend your wounds."}}
		}
		return &CommandResult{Messages: []string{fmt.Sprintf("%s must be sitting or lying down to be tended.", targetName)}}
	}

	wound, idx, failMsg := selectTendableWound(target.Wounds, healSkill)
	if failMsg != "" {
		return &CommandResult{Messages: []string{failMsg}}
	}

	heal := wound.Level
	if target.Race == healer.Race {
		heal = heal * 3 / 2
	}
	target.BodyPoints += heal
	if target.BodyPoints > target.MaxBodyPoints {
		target.BodyPoints = target.MaxBodyPoints
	}
	target.Wounds = append(target.Wounds[:idx], target.Wounds[idx+1:]...)
	target.Bleeding = anyBleeding(target.Wounds)

	woundDesc := woundWord(wound.DamageType, wound.Level) + " " + wound.Location
	healer.Experience += 10 * wound.Level

	firstAidRT := applyRoundTime(healer, 5)
	healer.RoundTimeExpiry = time.Now().Add(time.Duration(firstAidRT) * time.Second)

	// If this brought the target back above 0 body points, wake them immediately
	// rather than making them wait for the next regen tick.
	wakeMsg := wakeFromUnconscious(target)

	e.SavePlayer(ctx, healer)
	if !isSelf {
		e.SavePlayer(ctx, target)
	}

	if isSelf {
		msgs := []string{fmt.Sprintf("You tend to your %s, healing %d body points. [Round: 5 sec] [BP: %d/%d]", woundDesc, heal, target.BodyPoints, target.MaxBodyPoints)}
		if wakeMsg != "" {
			msgs = append(msgs, wakeMsg)
		}
		return &CommandResult{
			Messages:      msgs,
			RoomBroadcast: []string{fmt.Sprintf("%s tends to %s wounds.", healer.FirstName, healer.Possessive())},
			PlayerState:   healer,
		}
	}

	targetMsgs := []string{fmt.Sprintf("%s tends to your %s, healing %d body points. [BP: %d/%d]", healer.FirstName, woundDesc, heal, target.BodyPoints, target.MaxBodyPoints)}
	if wakeMsg != "" {
		targetMsgs = append(targetMsgs, wakeMsg)
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You tend to %s's %s, healing %d body points.", targetName, woundDesc, heal)},
		RoomBroadcast: []string{fmt.Sprintf("%s tends to %s's wounds.", healer.FirstName, targetName)},
		TargetName:    target.FirstName,
		TargetMsg:     targetMsgs,
		PlayerState:   healer,
	}
}

// tendMonster handles TEND when the target is a monster (dead or alive).
// Monster targets have no position requirement and gain no body-point
// healing of their own from being tended — only the wound is removed and the
// healer gains XP, as if patching up a corpse or a downed creature.
func (e *GameEngine) tendMonster(ctx context.Context, healer *Player, inst *MonsterInstance, targetName string, healSkill int) *CommandResult {
	if len(inst.Wounds) == 0 {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s doesn't need healing.", targetName)}}
	}

	_, idx, failMsg := selectTendableWound(inst.Wounds, healSkill)
	if failMsg != "" {
		return &CommandResult{Messages: []string{failMsg}}
	}

	removed, ok := e.removeMonsterWound(inst.ID, idx)
	if !ok {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s doesn't need healing.", targetName)}}
	}

	woundDesc := woundWord(removed.DamageType, removed.Level) + " " + removed.Location
	healer.Experience += 10 * removed.Level

	firstAidRT := applyRoundTime(healer, 5)
	healer.RoundTimeExpiry = time.Now().Add(time.Duration(firstAidRT) * time.Second)
	e.SavePlayer(ctx, healer)

	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You tend to %s's %s. [Round: 5 sec]", targetName, woundDesc)},
		RoomBroadcast: []string{fmt.Sprintf("%s tends to %s's wounds.", healer.FirstName, targetName)},
		PlayerState:   healer,
	}
}

// magicSkillSchool returns the spell school for a magic skill, or "" if not a magic skill.
func magicSkillSchool(skillID int) string {
	switch skillID {
	case 23:
		return "General"
	case 7:
		return "Conjuration"
	case 14:
		return "Enchantment"
	case 17:
		return "Druidic"
	case 30:
		return "Necromancy"
	}
	return ""
}

// guildScrollGift returns a scroll item containing a random level-1 spell from the given school.
func (e *GameEngine) guildScrollGift(school string) (*InventoryItem, string) {
	var candidates []SpellDef
	for _, sp := range spellRegistry {
		if sp.School == school && sp.Level == 1 {
			candidates = append(candidates, sp)
		}
	}
	if len(candidates) == 0 {
		return nil, ""
	}
	spell := candidates[rand.Intn(len(candidates))]

	scrollArch := 168
	if e.items[scrollArch] == nil {
		return nil, ""
	}

	item := &InventoryItem{
		Archetype: scrollArch,
		Val3:      spell.ID,
	}
	adjWord := scrollAdjectiveWords[rand.Intn(len(scrollAdjectiveWords))]
	if adjID := e.adjByName(adjWord); adjID != 0 {
		item.Adj1 = adjID
	}
	return item, spell.Name
}
