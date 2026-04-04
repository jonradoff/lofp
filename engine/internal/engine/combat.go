package engine

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// Combat stance constants
const (
	StanceNormal    = 0
	StanceOffensive = 1
	StanceDefensive = 2
	StanceBerserk   = 3
	StanceWary      = 4
)

var stanceNames = map[int]string{
	StanceNormal:    "normal",
	StanceOffensive: "offensive",
	StanceDefensive: "defensive",
	StanceBerserk:   "berserk",
	StanceWary:      "wary",
}

// CombatTarget tracks who a player/monster is fighting.
type CombatTarget struct {
	IsMonster  bool
	MonsterID  int
	PlayerName string
}

// ---- XP per build point table (from GM Manual) ----
// Index = level, value = XP needed per ONE build point at that level.
// Total build points at a level = 20 + (10 * level).
// Build points are earned incrementally as XP accumulates.
// Level increases when total build points reach 20 + 10*(level+1).

var xpPerBP = []int{
	0,     // level 0 (unused)
	100, 200, 400, 600, 800, 1000, 1200, 1400, 1600, 2000,       // 1-10
	2400, 2700, 3200, 4000, 4800, 5600, 6400, 7200, 8000, 8800,  // 11-20
	9600, 10400, 11200, 12000, 12800, 13600, 14400, 15200, 16000, 16800, // 21-30
	17600, 18400, 19200, 20000, 20800, 21600, 22400, 23200, 24000, 24800, // 31-40
	25600, 26400, 27200, 28000, 28800, 29600, 30400, 31200, 32000, 32800, // 41-50
	33600, 34400, 35200, 36000, 36800, 37600, 38400, 39200, 40000, 40800, // 51-60
	51600, 53200, 54800, 56400, 58000, 59600, 61200, 62800, 64400, 66000, // 61-70
	67600, 69200, 70800, 72400, 74000, 75600, 77200, 78800, 80400, 82000, // 71-80
	83600, 85200, 86800, 88400, 90000, 91600, 93200, 94800, 96400, 98000, // 81-90
	99600, 101200, 102800, 104400, 106000, 107600, 109200, 110800, 112400, 114000, // 91-100
}

// getXPPerBP returns the XP cost per build point at a given level.
func getXPPerBP(level int) int {
	if level <= 0 {
		return 100
	}
	if level < len(xpPerBP) {
		return xpPerBP[level]
	}
	// Formula for level 136+: 170000 + (level-135)*1600
	return 170000 + (level-135)*1600
}

// buildPointsForLevel returns total build points at a given level.
func buildPointsForLevel(level int) int {
	return 20 + 10*level
}

// recalcBuildPoints recalculates a player's build points and level from their XP.
// Build points are earned incrementally: each BP costs XP/BP at the player's current level.
func recalcBuildPoints(player *Player) (leveledUp bool) {
	// Calculate total BP earned from total XP
	xpRemaining := player.Experience
	bp := 20 // starting build points
	lvl := 1

	for {
		rate := getXPPerBP(lvl)
		targetBP := buildPointsForLevel(lvl + 1) // BP needed for next level
		bpToNextLevel := targetBP - bp
		xpForNextLevel := bpToNextLevel * rate

		if xpRemaining >= xpForNextLevel {
			xpRemaining -= xpForNextLevel
			bp = targetBP
			lvl++
		} else {
			// Partial progress within current level
			if rate > 0 {
				bp += xpRemaining / rate
			}
			break
		}

		if lvl > 200 { // safety cap
			break
		}
	}

	oldLevel := player.Level
	player.BuildPoints = bp
	player.Level = lvl

	return player.Level > oldLevel
}

// xpForNextBP returns the XP cost for the player's next build point.
func xpForNextBP(player *Player) int {
	return getXPPerBP(player.Level)
}

// xpProgressInLevel returns (xp earned in current level, xp needed for next level).
func xpProgressInLevel(player *Player) (earned int, needed int) {
	// Sum XP consumed by all levels before current
	xpConsumed := 0
	for lvl := 1; lvl < player.Level; lvl++ {
		bpInLevel := 10 // 10 BP per level
		xpConsumed += bpInLevel * getXPPerBP(lvl)
	}
	earned = player.Experience - xpConsumed
	if earned < 0 {
		earned = 0
	}
	needed = 10 * getXPPerBP(player.Level)
	return
}

// ---- Weather combat modifiers (from GM Manual) ----

var weatherAttackMod = map[int]int{
	4: -10, 5: -20, 6: -30, 7: -40, 8: -50, // rain→hurricane
	10: -10, 11: -20, 12: -30, 13: -40, 14: -50, // sleet→blizzard
}

func (e *GameEngine) weatherMod(roomNum int) int {
	room := e.rooms[roomNum]
	if room == nil || !isOutdoorTerrain(room.Terrain) {
		return 0
	}
	region := room.Region
	wea, ok := e.RegionWeather[region]
	if !ok {
		return 0
	}
	if mod, ok := weatherAttackMod[wea]; ok {
		return mod
	}
	return 0
}

// ---- Damage severity tiers (from original session capture) ----

var severityTiers = []struct {
	maxDmg int
	name   string
}{
	{5, "Puny"}, {10, "Grazing"}, {15, "Insignificant"}, {20, "Minor"},
	{25, "Passable"}, {30, "Good"}, {40, "Well-aimed"}, {50, "Masterful"},
	{60, "Grisly"}, {75, "Severe"}, {100, "Ghastly"}, {99999, "Dazzling explosive"},
}

func damageSeverity(dmg int) string {
	for _, tier := range severityTiers {
		if dmg <= tier.maxDmg {
			return tier.name
		}
	}
	return "Devastating"
}

// ---- Attack verb by weapon type (from session capture) ----

func attackVerb(weaponDef *gameworld.ItemDef) (selfVerb, thirdVerb, dmgNoun string) {
	if weaponDef == nil {
		return "swing", "swings", "strike"
	}
	switch weaponDef.Type {
	case "SLASH_WEAPON", "TWOHAND_WEAPON", "CLAW_WEAPON", "DRAKIN_SLASH":
		return "swing", "swings", "slash"
	case "PUNCTURE_WEAPON", "POLE_WEAPON", "POLETHROWN", "DRAKIN_POLE", "STABTHROWN":
		return "thrust", "thrusts", "strike"
	case "CRUSH_WEAPON", "BLUNT_WEAPON", "DRAKIN_CRUSH":
		return "swing", "swings", "strike"
	case "BOW_WEAPON", "THROWN_WEAPON", "DRAKIN_THROWN":
		return "fire", "fires", "strike"
	case "BITE_WEAPON":
		return "bite", "bites", "bite"
	default:
		return "swing", "swings", "strike"
	}
}

func monsterAttackVerb(def *gameworld.MonsterDef, items map[int]*gameworld.ItemDef) (verb, dmgNoun string) {
	if len(def.Weapons) > 0 {
		wep := items[def.Weapons[0].Archetype]
		if wep != nil {
			_, v, dn := attackVerb(wep)
			return v + " at", dn
		}
	}
	if def.BodyType == "ANIMAL" || def.BodyType == "AVINE" {
		return "slashes at", "slash"
	}
	return "swings at", "strike"
}

// ---- Weapon skill mapping ----

func weaponSkillForType(itemType string) int {
	switch itemType {
	case "SLASH_WEAPON", "TWOHAND_WEAPON":
		return 13
	case "CRUSH_WEAPON", "BLUNT_WEAPON":
		return 9
	case "PUNCTURE_WEAPON", "STABTHROWN":
		return 13
	case "POLE_WEAPON", "POLETHROWN":
		return 25
	case "BOW_WEAPON", "THROWN_WEAPON":
		return 3
	case "DRAKIN_SLASH", "DRAKIN_CRUSH", "DRAKIN_POLE", "DRAKIN_THROWN":
		return 16
	case "CLAW_WEAPON", "BITE_WEAPON":
		return 4
	default:
		return 13
	}
}

// ---- To-Hit Calculation ----

func calcToHit(attackRating, defenseRating int) int {
	toHit := 50 + defenseRating - attackRating
	if toHit < 5 {
		toHit = 5
	}
	if toHit > 95 {
		toHit = 95
	}
	return toHit
}

func playerAttackRating(player *Player, weaponDef *gameworld.ItemDef) int {
	rating := 50
	rating += player.Level * 3
	if weaponDef != nil {
		skillID := weaponSkillForType(weaponDef.Type)
		rating += player.Skills[skillID] * 5 // +5 per weapon skill rank (from skills.txt)
	} else {
		// Unarmed: martial arts skill
		rating += player.Skills[24] * 5 // Martial Arts +5 per rank
	}
	if weaponDef != nil && (weaponDef.Type == "BOW_WEAPON" || weaponDef.Type == "THROWN_WEAPON") {
		rating += player.Agility / 5
	} else {
		rating += player.Strength / 5
	}
	switch player.Stance {
	case StanceOffensive:
		rating += 15
	case StanceDefensive:
		rating -= 15
	case StanceBerserk:
		rating += 25
	case StanceWary:
		rating -= 5
	}
	switch player.Position {
	case 1:
		rating -= 20
	case 2:
		rating -= 30
	case 3:
		rating -= 10
	}
	return rating
}

func playerDefenseRating(player *Player) int {
	rating := 25
	rating += player.Level * 3
	rating += player.Skills[6] * 5 // Dodge & Parry: +5 per rank
	rating += player.Agility / 5
	// Martial Arts defense bonus: +2 per rank if unarmed
	if player.Wielded == nil {
		rating += player.Skills[24] * 2
	}
	switch player.Stance {
	case StanceOffensive:
		rating -= 15
	case StanceDefensive:
		rating += 15
	case StanceBerserk:
		rating -= 25
	case StanceWary:
		rating += 5
	}
	rating += player.DefenseBonus
	switch player.Position {
	case 1:
		rating -= 15
	case 2:
		rating -= 25
	case 3:
		rating -= 10
	}
	return rating
}

func playerArmorPercent(player *Player, items map[int]*gameworld.ItemDef) int {
	total := 0
	for _, worn := range player.Worn {
		def := items[worn.Archetype]
		if def != nil && def.Type == "ARMOR" {
			total += def.Parameter1
		}
	}
	if total > 85 {
		total = 85
	}
	return total
}

// ---- Damage Calculation ----

func playerDamage(player *Player, weaponDef *gameworld.ItemDef) int {
	if weaponDef == nil {
		return rand.Intn(3) + 1 + player.Strength/20
	}
	maxDmg := weaponDef.Parameter1
	if maxDmg <= 0 {
		maxDmg = 3
	}
	dmg := rand.Intn(maxDmg) + 1
	if weaponDef.Type == "BOW_WEAPON" || weaponDef.Type == "THROWN_WEAPON" {
		dmg += player.Agility / 10
	} else {
		dmg += player.Strength / 10
	}
	if player.Stance == StanceBerserk {
		dmg = dmg * 12 / 10
	}
	// Backstab multiplier: 2x damage + backstab skill bonus
	if player.BackstabNext {
		backstabSkill := player.Skills[2] // Backstab skill
		dmg = dmg*2 + backstabSkill
	}
	return dmg
}

// weaponCritDamage checks VAL3 for elemental crit or slayer bonus.
// Returns (extra damage, crit type description, hit).
func weaponCritDamage(wielded *InventoryItem, weaponDef *gameworld.ItemDef, monDef *gameworld.MonsterDef) (int, string) {
	if wielded == nil || weaponDef == nil {
		return 0, ""
	}
	val3 := wielded.Val3
	if val3 == 0 {
		val3 = weaponDef.Parameter3
	}
	if val3 == 0 {
		return 0, ""
	}
	val5 := wielded.Val5
	if val5 == 0 {
		// Infer crit max from weapon damage
		val5 = weaponDef.Parameter1 / 2
		if val5 < 5 {
			val5 = 5
		}
	}

	// Elemental crits (VAL3 2-18): chance-based extra damage
	switch {
	case val3 >= 2 && val3 <= 18:
		chance := 0
		dmgType := ""
		switch val3 {
		case 2:
			chance, dmgType = 50, "heat"
		case 3:
			chance, dmgType = 50, "cold"
		case 4:
			chance, dmgType = 40, "electric"
		case 5:
			chance, dmgType = 40, "heat"
		case 6:
			chance, dmgType = 40, "cold"
		case 7:
			chance, dmgType = 40, "electric"
		case 10:
			chance, dmgType = 30, "heat"
		case 11:
			chance, dmgType = 30, "cold"
		case 12:
			chance, dmgType = 30, "electric"
		case 13:
			chance, dmgType = 20, "heat"
		case 14:
			chance, dmgType = 20, "cold"
		case 15:
			chance, dmgType = 20, "electric"
		case 16:
			chance, dmgType = 10, "heat"
		case 17:
			chance, dmgType = 10, "cold"
		case 18:
			chance, dmgType = 10, "electric"
		}
		if chance > 0 && rand.Intn(100) < chance {
			extra := rand.Intn(val5) + 1
			// Apply elemental immunity
			if monDef != nil {
				immType := elementalImmunityType(dmgType)
				if level, ok := monDef.Immunities[immType]; ok {
					extra = applyImmunity(extra, level)
				}
			}
			typeNames := map[string]string{"heat": "fire", "cold": "cold", "electric": "lightning"}
			return extra, typeNames[dmgType]
		}

	// Slayer weapons (VAL3 21-32): bonus damage vs specific monster races
	case val3 >= 21 && val3 <= 32:
		if monDef != nil && monDef.Race == val3 {
			// Slayer hit! Double damage from val5
			return val5, "slayer"
		}
	}
	return 0, ""
}

// weaponPoisonLevel checks VAL4 for poison (51-100 = poison level 1-50).
func weaponPoisonLevel(wielded *InventoryItem) int {
	if wielded == nil {
		return 0
	}
	if wielded.Val4 >= 51 && wielded.Val4 <= 100 {
		return wielded.Val4 - 50
	}
	return 0
}

func monsterDamage(def *gameworld.MonsterDef) int {
	minDmg := max(1, def.Attack1/20)
	maxDmg := max(2, def.Attack1/5)
	if maxDmg <= minDmg {
		maxDmg = minDmg + 1
	}
	return rand.Intn(maxDmg-minDmg+1) + minDmg
}

func monsterSpecialDamage(def *gameworld.MonsterDef) (int, string) {
	if def.SpecUse <= 0 || rand.Intn(100) >= def.SpecUse {
		return 0, ""
	}
	dmg := def.SpecBase + rand.Intn(max(1, def.SpecDmg))
	return dmg, def.SpecDmgType
}

func applyArmor(dmg, armorPct int) int {
	reduced := dmg * (100 - armorPct) / 100
	if reduced < 0 {
		reduced = 0
	}
	return reduced
}

func applyImmunity(dmg int, immunityLevel int) int {
	switch immunityLevel {
	case 0:
		return 0
	case 1:
		return dmg / 2
	case 3:
		return dmg * 3 / 2
	case 4:
		return dmg * 2
	default:
		return dmg
	}
}

// ---- MAGICWEAPON check ----
// Some monsters require magic weapons: 1=any magic, 2=bonus>=10, 3=bonus>=21

func checkMagicWeapon(player *Player, wielded *InventoryItem, weaponDef *gameworld.ItemDef, monDef *gameworld.MonsterDef) bool {
	if monDef.MagicWeapon <= 0 {
		return true // no requirement
	}
	// Martial Arts 10+ can hit magic-required monsters (level 1 only)
	if wielded == nil && player.Skills[24] >= 10 && monDef.MagicWeapon <= 1 {
		return true
	}
	if wielded == nil || weaponDef == nil {
		return false // unarmed can't hit magic-required monsters
	}
	bonus := wielded.Val2 // VAL2 = magic bonus
	switch monDef.MagicWeapon {
	case 1:
		return bonus > 0
	case 2:
		return bonus >= 10
	case 3:
		return bonus >= 21
	}
	return true
}

// ---- Body parts ----

var bodyParts = []string{"head", "body", "right arm", "left arm", "right leg", "left leg", "back"}
var animalParts = []string{"head", "body", "right foreleg", "left foreleg", "right hind leg", "left hind leg", "tail"}

func randomBodyPart(bodyType string) string {
	if bodyType == "ANIMAL" || bodyType == "AVINE" {
		return animalParts[rand.Intn(len(animalParts))]
	}
	return bodyParts[rand.Intn(len(bodyParts))]
}

// ---- Weapon helpers ----

func weaponImmunityType(weaponDef *gameworld.ItemDef) int {
	if weaponDef == nil {
		return 1
	}
	switch weaponDef.Type {
	case "CRUSH_WEAPON", "BLUNT_WEAPON":
		return 1
	default:
		return 2
	}
}

func (e *GameEngine) weaponDisplayName(player *Player, weaponDef *gameworld.ItemDef) string {
	if weaponDef == nil {
		return "fist"
	}
	if player.Wielded != nil {
		return e.formatItemName(weaponDef, player.Wielded.Adj1, player.Wielded.Adj2, player.Wielded.Adj3)
	}
	return strings.ToLower(e.nouns[weaponDef.NameID])
}

func (e *GameEngine) monsterWeaponName(def *gameworld.MonsterDef) string {
	if len(def.Weapons) > 0 {
		wep := e.items[def.Weapons[0].Archetype]
		if wep != nil {
			adj := ""
			if def.Weapons[0].Adj > 0 {
				if a, ok := e.adjectives[def.Weapons[0].Adj]; ok {
					adj = a + " "
				}
			}
			return adj + strings.ToLower(e.nouns[wep.NameID])
		}
	}
	return "claws"
}

// ---- Arena check ----

func (e *GameEngine) isArenaRoom(roomNum int) bool {
	room := e.rooms[roomNum]
	if room == nil {
		return false
	}
	for _, mod := range room.Modifiers {
		if mod == "ARENA" {
			return true
		}
	}
	return false
}

// ---- Player attacks Monster ----

func (e *GameEngine) doAttackMonster(ctx context.Context, player *Player, target string) *CommandResult {
	if player.Dead {
		return &CommandResult{Messages: []string{"You can't do that. You are dead."}}
	}
	if player.Stunned {
		return &CommandResult{Messages: []string{"You are stunned and cannot attack!"}}
	}
	if player.Immobilized {
		return &CommandResult{Messages: []string{"You are rooted to the spot!"}}
	}
	if player.Position == 2 {
		return &CommandResult{Messages: []string{"You can't attack while laying down! Stand up first."}}
	}

	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := int(player.RoundTimeExpiry.Sub(time.Now()).Seconds()) + 1
		return &CommandResult{Messages: []string{fmt.Sprintf("[Wait %d seconds...]", remaining)}}
	}

	inst, def := e.findMonsterInRoom(player, target)
	if inst == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' here to attack.", target)}}
	}

	// Check if a guard monster intervenes
	guardInst, guardDef := e.findGuardFor(inst, player.RoomNumber)
	if guardInst != nil && guardDef != nil {
		guardName := FormatMonsterName(guardDef, e.monAdjs)
		guardArticle := articleFor(guardName, guardDef.Unique)
		if e.sendToPlayer != nil {
			e.sendToPlayer(player.FirstName, []string{fmt.Sprintf("%s%s is now guarding %s%s.", capArticle(guardArticle), guardName, articleFor(FormatMonsterName(def, e.monAdjs), def.Unique), FormatMonsterName(def, e.monAdjs))})
		}
		// Redirect attack to the guard
		inst = guardInst
		def = guardDef
	}

	name := FormatMonsterName(def, e.monAdjs)
	article := articleFor(name, def.Unique)

	var weaponDef *gameworld.ItemDef
	if player.Wielded != nil {
		weaponDef = e.items[player.Wielded.Archetype]
	}

	// Check ranged weapon is loaded
	isRangedWeapon := weaponDef != nil && (weaponDef.Type == "BOW_WEAPON" || weaponDef.Type == "HANDGUN" || weaponDef.Type == "RIFLE")
	if isRangedWeapon && (player.Wielded == nil || player.Wielded.Val3 <= 0) {
		return &CommandResult{Messages: []string{fmt.Sprintf("Your %s is not loaded! Use NOCK or LOAD first.", strings.ToLower(e.nouns[weaponDef.NameID]))}}
	}

	// Check MAGICWEAPON requirement
	if !checkMagicWeapon(player, player.Wielded, weaponDef, def) {
		texI := def.TextOverrides["TEXI"]
		if texI == "" {
			texI = fmt.Sprintf("Your weapon is not powerful enough to affect %s%s.", article, name)
		}
		return &CommandResult{Messages: []string{texI}}
	}

	// Engage
	player.CombatTarget = &CombatTarget{IsMonster: true, MonsterID: inst.ID}
	player.Joined = true
	e.monsterMgr.mu.Lock()
	for i := range e.monsterMgr.instances {
		if e.monsterMgr.instances[i].ID == inst.ID {
			if e.monsterMgr.instances[i].Target == "" {
				e.monsterMgr.instances[i].Target = player.FirstName
			}
			break
		}
	}
	e.monsterMgr.mu.Unlock()

	// Cry for law (strategy 1-25 or 101-125)
	if (def.Strategy >= 1 && def.Strategy <= 25) || (def.Strategy >= 101 && def.Strategy <= 125) {
		e.cryForLaw(player, inst, def)
	}

	// Fatigue drain for melee attacks (not ranged)
	isRanged := weaponDef != nil && (weaponDef.Type == "BOW_WEAPON" || weaponDef.Type == "THROWN_WEAPON")
	if !isRanged {
		fatCost := 1
		if weaponDef != nil && weaponDef.Weight > 5 {
			fatCost = weaponDef.Weight / 5
		}
		player.Fatigue -= fatCost
		if player.Fatigue < 0 {
			player.Fatigue = 0
		}
		if player.Fatigue <= 0 {
			return &CommandResult{Messages: []string{"You are too fatigued to attack!"}}
		}
	}

	// Apply weather modifier
	wMod := e.weatherMod(player.RoomNumber)

	// Fatigue penalty to ToHit
	fatPenalty := 0
	if player.MaxFatigue > 0 {
		fatRatio := player.Fatigue * 100 / player.MaxFatigue
		if fatRatio < 25 {
			fatPenalty = 25 // under 1/4 fatigue: -25
		} else if fatRatio < 50 {
			fatPenalty = 10 // under 1/2 fatigue: -10
		}
	}

	// Resolve to-hit
	attackRating := playerAttackRating(player, weaponDef) + wMod - fatPenalty
	monDefense := def.Defense + inst.DefenseBonus
	toHit := calcToHit(attackRating, monDefense)
	roll := rand.Intn(100) + 1

	selfVerb, thirdVerb, dmgNoun := attackVerb(weaponDef)
	weaponName := e.weaponDisplayName(player, weaponDef)
	pronoun := player.Possessive()

	result := &CommandResult{}
	var msgs []string

	msgs = append(msgs, fmt.Sprintf("You %s at %s%s with your %s.", selfVerb, article, name, weaponName))

	// Weapon clash on roll < 3 (only vs weapon-wielding monsters)
	if roll < 3 && weaponDef != nil && len(def.Weapons) > 0 {
		weaponStr := weaponDef.Weight*3 + weaponDef.Parameter1*2
		clashRoll := rand.Intn(100) + rand.Intn(100) + 2
		msgs = append(msgs, fmt.Sprintf(" [ToHit: %d, Roll: %d] Weapon Clash! [Strength: %d, 2d100 Roll: %d]", toHit, roll, weaponStr, clashRoll))
		if clashRoll > weaponStr {
			if player.Wielded != nil && player.Wielded.State == "DAMAGED" {
				msgs = append(msgs, fmt.Sprintf(" Your %s breaks!", strings.ToLower(e.nouns[weaponDef.NameID])))
				player.Wielded = nil
			} else if player.Wielded != nil {
				player.Wielded.State = "DAMAGED"
				msgs = append(msgs, fmt.Sprintf(" %s damaged!", strings.Title(strings.ToLower(e.nouns[weaponDef.NameID]))))
			}
		}
		result.Messages = msgs
		rtSec := 5
		player.RoundTimeExpiry = time.Now().Add(time.Duration(rtSec) * time.Second)
		result.Messages = append(result.Messages, fmt.Sprintf("[Round: %d sec]", rtSec))
		if player.Hidden { player.Hidden = false; result.Messages = append([]string{"You reveal yourself!"}, result.Messages...) }
		e.SavePlayer(ctx, player)
		result.PlayerState = player
		return result
	}

	// Damaged weapon penalty (-10 ToHit)
	if player.Wielded != nil && player.Wielded.State == "DAMAGED" {
		toHit += 10 // harder to hit with damaged weapon
	}

	if roll >= toHit {
		excellent := roll >= 96
		hitLabel := "Hit!"
		if excellent {
			hitLabel = "Excellent Hit!"
		}
		msgs = append(msgs, fmt.Sprintf(" [ToHit: %d, Roll: %d] %s", toHit, roll, hitLabel))

		dmg := playerDamage(player, weaponDef)
		dmg = applyArmor(dmg, def.Armor)
		immType := weaponImmunityType(weaponDef)
		if level, ok := def.Immunities[immType]; ok {
			dmg = applyImmunity(dmg, level)
		}
		if dmg <= 0 {
			dmg = 1
		}

		part := randomBodyPart(def.BodyType)
		severity := damageSeverity(dmg)
		msgs = append(msgs, fmt.Sprintf(" %s %s to %s. [%d Damage]", severity, dmgNoun, part, dmg))

		// Weapon elemental crit / slayer bonus
		if critDmg, critType := weaponCritDamage(player.Wielded, weaponDef, def); critDmg > 0 {
			dmg += critDmg
			switch critType {
			case "fire":
				msgs = append(msgs, fmt.Sprintf(" The %s radiates intense heat! [%d Damage]", strings.ToLower(e.nouns[weaponDef.NameID]), critDmg))
			case "cold":
				msgs = append(msgs, fmt.Sprintf(" The %s radiates intense cold! [%d Damage]", strings.ToLower(e.nouns[weaponDef.NameID]), critDmg))
			case "lightning":
				msgs = append(msgs, fmt.Sprintf(" The %s crackles with electricity! [%d Damage]", strings.ToLower(e.nouns[weaponDef.NameID]), critDmg))
			case "slayer":
				msgs = append(msgs, fmt.Sprintf(" Your weapon resonates against its foe! [%d Damage]", critDmg))
			}
		}

		killed := e.damageMonster(inst.ID, dmg)

		// Weapon poison
		if poisonLvl := weaponPoisonLevel(player.Wielded); poisonLvl > 0 && !killed {
			msgs = append(msgs, " Your weapon delivers its venom!")
		}

		if excellent && !killed {
			if rand.Intn(100) < 30 {
				msgs = append(msgs, " It is stunned!")
			}
		}

		if killed {
			deathText := def.TextOverrides["TEXD"]
			if deathText != "" {
				msgs = append(msgs, fmt.Sprintf(" It %s", deathText))
			} else {
				msgs = append(msgs, " It collapses, dead.")
			}
			e.handleMonsterDeath(player, inst, def)
			player.CombatTarget = nil
			player.Joined = false
		}

		result.RoomBroadcast = []string{fmt.Sprintf("%s %s at %s%s with %s %s.", player.FirstName, thirdVerb, article, name, pronoun, weaponName)}
	} else {
		msgs = append(msgs, fmt.Sprintf(" [ToHit: %d, Roll: %d] Miss.", toHit, roll))
		result.RoomBroadcast = []string{fmt.Sprintf("%s %s at %s%s but misses.", player.FirstName, thirdVerb, article, name)}
	}

	result.Messages = msgs

	// Roundtime: base 5, reduced by quickness and Combat Maneuvering
	rtSeconds := 5
	if player.Quickness > 80 {
		rtSeconds = 3
	} else if player.Quickness > 50 {
		rtSeconds = 4
	}
	// Combat Maneuvering: -1 sec per rank (from skills.txt)
	combatManeuver := player.Skills[10]
	rtSeconds -= combatManeuver
	if player.Stance == StanceBerserk {
		rtSeconds--
	}
	if rtSeconds < 2 {
		rtSeconds = 2
	}
	player.RoundTimeExpiry = time.Now().Add(time.Duration(rtSeconds) * time.Second)
	result.Messages = append(result.Messages, fmt.Sprintf("[Round: %d sec]", rtSeconds))

	// Unload ranged weapon after firing
	if isRangedWeapon && player.Wielded != nil {
		player.Wielded.Val3 = 0 // unloaded
	}

	// Attacking always reveals you (both hidden and invisible)
	if player.Hidden || player.Invisible {
		player.Hidden = false
		player.Invisible = false
		result.Messages = append([]string{"You reveal yourself!"}, result.Messages...)
	}

	e.SavePlayer(ctx, player)
	result.PlayerState = player

	return result
}

// doBackstab handles a backstab attack from hiding — bonus damage.
// Requires puncture weapon (daggers, rapiers).
func (e *GameEngine) doBackstab(ctx context.Context, player *Player, target string) *CommandResult {
	// Check weapon type — backstab requires puncture weapons
	var weaponDef *gameworld.ItemDef
	if player.Wielded != nil {
		weaponDef = e.items[player.Wielded.Archetype]
	}
	if weaponDef == nil || (weaponDef.Type != "PUNCTURE_WEAPON" && weaponDef.Type != "STABTHROWN") {
		return &CommandResult{Messages: []string{"You can only backstab with a puncture weapon such as a dagger or rapier."}}
	}

	// Backstab: attack from hidden with damage multiplier
	player.BackstabNext = true
	player.Hidden = false
	player.Invisible = false
	result := e.doAttackMonster(ctx, player, target)
	player.BackstabNext = false
	result.Messages = append([]string{"You leap from the shadows!"}, result.Messages...)
	result.RoomBroadcast = append([]string{fmt.Sprintf("%s leaps from the shadows!", player.FirstName)}, result.RoomBroadcast...)
	return result
}

// ---- Monster attacks Player ----

func (e *GameEngine) monsterAttackPlayer(inst *MonsterInstance, def *gameworld.MonsterDef, player *Player) (playerMsgs []string, roomMsgs []string) {
	if player.Dead || !inst.Alive {
		return nil, nil
	}

	name := FormatMonsterName(def, e.monAdjs)
	article := articleFor(name, def.Unique)
	capArt := capArticle(article)

	// Special attack
	if specDmg, specType := monsterSpecialDamage(def); specDmg > 0 {
		// Combat Maneuvering: 2% per rank chance to dodge special attack (max 95%)
		combatManeuver := player.Skills[10]
		dodgeChance := combatManeuver * 2
		if dodgeChance > 95 { dodgeChance = 95 }
		if dodgeChance > 0 && rand.Intn(100) < dodgeChance {
			playerMsgs = append(playerMsgs, fmt.Sprintf("%s%s uses a special attack, but you dodge it!", capArt, name))
		} else {
		specText := def.TextOverrides["TEXX"]
		if specText != "" {
			specText = strings.Replace(specText, "%s", capArt+name, 1)
			specText = strings.Replace(specText, "%s", player.FirstName, 1)
		} else {
			specText = fmt.Sprintf("%s%s uses a special attack on %s!", capArt, name, player.FirstName)
		}

		armorPct := playerArmorPercent(player, e.items)
		specDmg = applyArmor(specDmg, armorPct)

		// Endurance: 1% elemental damage reduction per rank (max 50%)
		enduranceSkill := player.Skills[11]
		if enduranceSkill > 0 && (specType == "Heat" || specType == "Cold" || specType == "Electric") {
			reduction := enduranceSkill
			if reduction > 50 { reduction = 50 }
			specDmg = specDmg * (100 - reduction) / 100
		}
		_ = specType

		part := randomBodyPart("HUMAN")
		severity := damageSeverity(specDmg)
		player.BodyPoints -= specDmg
		if player.BodyPoints < 0 {
			player.BodyPoints = 0
		}

		playerMsgs = append(playerMsgs, specText)
		playerMsgs = append(playerMsgs, fmt.Sprintf(" %s burn to %s. [%d Damage]", severity, part, specDmg))
		roomMsgs = append(roomMsgs, specText)

		if player.BodyPoints <= 0 {
			deathMsgs := e.handlePlayerDeath(player, name)
			playerMsgs = append(playerMsgs, deathMsgs...)
			return playerMsgs, roomMsgs
		}
		} // end else (didn't dodge)
	}

	// Normal attack
	monWeaponName := e.monsterWeaponName(def)
	monVerb, monDmgNoun := monsterAttackVerb(def, e.items)

	playerMsgs = append(playerMsgs, fmt.Sprintf("%s%s %s %s with its %s.", capArt, name, monVerb, player.FirstName, monWeaponName))

	// Weather modifier for monsters too
	wMod := e.weatherMod(inst.RoomNumber)
	toHit := calcToHit(def.Attack1+wMod, playerDefenseRating(player))
	roll := rand.Intn(100) + 1

	if roll >= toHit {
		excellent := roll >= 96
		hitLabel := "Hit!"
		if excellent {
			hitLabel = "Excellent Hit!"
		}
		playerMsgs = append(playerMsgs, fmt.Sprintf(" [ToHit: %d, Roll: %d] %s", toHit, roll, hitLabel))

		dmg := monsterDamage(def)
		armorPct := playerArmorPercent(player, e.items)
		dmg = applyArmor(dmg, armorPct)
		if dmg <= 0 {
			dmg = 1
		}

		part := randomBodyPart("HUMAN")
		severity := damageSeverity(dmg)
		player.BodyPoints -= dmg
		if player.BodyPoints < 0 {
			player.BodyPoints = 0
		}

		playerMsgs = append(playerMsgs, fmt.Sprintf(" %s %s to %s. [%d Damage]", severity, monDmgNoun, part, dmg))
		roomMsgs = append(roomMsgs, fmt.Sprintf("%s%s attacks %s!", capArt, name, player.FirstName))

		// Monster poison/disease/fatigue on hit
		if def.PoisonChance > 0 && rand.Intn(100) < def.PoisonChance {
			player.Poisoned = true
			playerMsgs = append(playerMsgs, " You feel poison coursing through your veins!")
		}
		if def.DiseaseChance > 0 && rand.Intn(100) < def.DiseaseChance {
			player.Diseased = true
			playerMsgs = append(playerMsgs, " You feel a sickness taking hold!")
		}
		if def.FatigueChance > 0 && rand.Intn(100) < def.FatigueChance {
			drain := def.FatigueLevel
			if drain <= 0 {
				drain = 5
			}
			player.Fatigue -= drain
			if player.Fatigue < 0 {
				player.Fatigue = 0
			}
			playerMsgs = append(playerMsgs, " You feel your life force being drained!")
		}

		if player.BodyPoints <= 0 {
			// Arena prevents full death
			if e.isArenaRoom(player.RoomNumber) {
				player.BodyPoints = 1
				playerMsgs = append(playerMsgs, " The arena's enchantment prevents your death!")
			} else {
				playerMsgs = append(playerMsgs, fmt.Sprintf(" %s%s slays %s.", capArt, name, player.FirstName))
				deathMsgs := e.handlePlayerDeath(player, name)
				playerMsgs = append(playerMsgs, deathMsgs...)
				roomMsgs = append(roomMsgs, fmt.Sprintf("%s%s slays %s!", capArt, name, player.FirstName))
			}
		}
	} else {
		playerMsgs = append(playerMsgs, fmt.Sprintf(" [ToHit: %d, Roll: %d] Miss.", toHit, roll))
	}

	return playerMsgs, roomMsgs
}

// ---- Death ----

func (e *GameEngine) handlePlayerDeath(player *Player, killerName string) []string {
	player.Dead = true
	player.CombatTarget = nil
	player.Joined = false
	player.Position = 2 // laying down

	// XP penalty: lose up to 90% of XP towards current build point
	rate := getXPPerBP(player.Level)
	if rate > 0 {
		xpInCurrentBP := player.Experience % rate
		penalty := xpInCurrentBP * 90 / 100
		player.Experience -= penalty
		if player.Experience < 0 {
			player.Experience = 0
		}
		recalcBuildPoints(player)
	}

	e.Events.Publish("combat", fmt.Sprintf("%s was killed by %s in room %d", player.FirstName, killerName, player.RoomNumber))

	return []string{
		fmt.Sprintf(" %s collapses, unconscious.", player.FirstName),
		fmt.Sprintf(" %s slays %s.", killerName, player.FirstName),
		"",
		"You are dead and can't do much of anything beside wait for someone to attempt to raise you or for Eternity, Inc. to retrieve you. Hope you paid your premium! [You may type DEPART at any time to allow Eternity, Inc. to retrieve you.]",
	}
}

func (e *GameEngine) doDepart(player *Player) *CommandResult {
	if !player.Dead {
		return &CommandResult{Messages: []string{"You are not dead."}}
	}

	player.Dead = false
	player.Position = 0
	player.Bleeding = false
	player.Stunned = false
	player.Poisoned = false
	player.Diseased = false

	player.BodyPoints = player.MaxBodyPoints / 4
	if player.BodyPoints < 1 {
		player.BodyPoints = 1
	}

	player.RoomNumber = e.startRoom

	result := e.doLook(player)
	result.Messages = append([]string{
		"Your spirit coalesces and you feel the sensation of being pulled back into the world...",
		"",
	}, result.Messages...)
	result.RoomBroadcast = []string{fmt.Sprintf("%s's spirit has returned from Eternity.", player.FirstName)}

	return result
}

// damageMonster applies damage to a monster instance. Returns true if killed.
func (e *GameEngine) damageMonster(monsterID int, dmg int) bool {
	e.monsterMgr.mu.Lock()
	defer e.monsterMgr.mu.Unlock()
	for i := range e.monsterMgr.instances {
		if e.monsterMgr.instances[i].ID == monsterID && e.monsterMgr.instances[i].Alive {
			e.monsterMgr.instances[i].CurrentHP -= dmg
			if e.monsterMgr.instances[i].CurrentHP <= 0 {
				e.monsterMgr.instances[i].Alive = false
				e.monsterMgr.instances[i].CurrentHP = 0
				e.monsterMgr.instances[i].DeathTime = time.Now()
				return true
			}
			return false
		}
	}
	return false
}

// ---- Monster Death ----

func (e *GameEngine) handleMonsterDeath(killer *Player, inst *MonsterInstance, def *gameworld.MonsterDef) {
	// XP formula: Body (not ExtraBody) + Attack/5 + Defense/5 + Armor/2 + level scaling
	xp := def.Body + def.Attack1/5 + def.Defense/5 + def.Armor/2
	if def.MagicResist > 0 {
		xp += def.MagicResist / 5
	}
	if xp < 10 {
		xp = 10
	}
	// Scale XP slightly by player level (diminishing returns for grinding weak mobs)
	if killer.Level > 1 && xp < killer.Level*5 {
		xp = max(5, xp*50/(killer.Level*5))
	}
	killer.Experience += xp

	// Alignment shift
	if def.Alignment < 0 {
		killer.Alignment += 1
	} else if def.Alignment > 0 {
		killer.Alignment -= 1
	}

	e.Events.Publish("combat", fmt.Sprintf("%s killed %s (monster %d) for %d XP in room %d",
		killer.FirstName, def.Name, def.Number, xp, killer.RoomNumber))

	// Drop monster's weapon into the room as loot
	if len(def.Weapons) > 0 && !def.Discorporate {
		room := e.rooms[killer.RoomNumber]
		if room != nil {
			// Pick a random weapon from the monster's weapon list
			wep := def.Weapons[rand.Intn(len(def.Weapons))]
			wepDef := e.items[wep.Archetype]
			if wepDef != nil {
				// Add weapon to room items
				ref := len(room.Items) // next available ref
				ri := gameworld.RoomItem{
					Ref:       ref,
					Archetype: wep.Archetype,
					Adj1:      wep.Adj,
				}
				// Apply weapon plus as Val2 (magic bonus)
				if def.WeaponPlus > 0 {
					ri.Val2 = def.WeaponPlus
				}
				room.Items = append(room.Items, ri)
				wepName := e.formatItemName(wepDef, wep.Adj, 0, 0)
				if e.localRoomBroadcast != nil {
					e.localRoomBroadcast(killer.RoomNumber, []string{fmt.Sprintf("A %s clatters to the ground.", wepName)})
				}
			}
		}
	}

	// Generate treasure drops based on monster's TREASURE level
	if def.Treasure > 0 && !def.Discorporate {
		treasureMsgs := e.generateTreasure(killer.RoomNumber, def.Treasure)
		if len(treasureMsgs) > 0 && e.localRoomBroadcast != nil {
			e.localRoomBroadcast(killer.RoomNumber, treasureMsgs)
		}
	}

	// Recalculate build points and check for level-up
	oldLevel := killer.Level
	oldBP := killer.BuildPoints
	leveledUp := recalcBuildPoints(killer)
	newBP := killer.BuildPoints

	// Tell the player
	var xpMsgs []string
	xpMsgs = append(xpMsgs, fmt.Sprintf("[+%d experience]", xp))
	if newBP > oldBP {
		xpMsgs = append(xpMsgs, fmt.Sprintf("[+%d build points! Total: %d]", newBP-oldBP, newBP))
	}

	if leveledUp {
		killer.MaxBodyPoints += killer.Constitution / 10
		killer.BodyPoints = killer.MaxBodyPoints
		killer.MaxFatigue += killer.Constitution / 15
		killer.Fatigue = killer.MaxFatigue
		xpMsgs = append(xpMsgs, fmt.Sprintf("Congratulations! You have advanced to level %d!", killer.Level))
		if e.roomBroadcast != nil {
			e.roomBroadcast(killer.RoomNumber, []string{
				fmt.Sprintf("%s has advanced to level %d!", killer.FirstName, killer.Level),
			})
		}
		_ = oldLevel
	}

	if e.sendToPlayer != nil {
		e.sendToPlayer(killer.FirstName, xpMsgs)
	}
}

// ---- Flee ----

func (e *GameEngine) doFlee(ctx context.Context, player *Player) *CommandResult {
	if player.CombatTarget == nil && !player.Joined {
		return &CommandResult{Messages: []string{"You are not in combat."}}
	}
	if player.Dead {
		return &CommandResult{Messages: []string{"You can't flee. You are dead."}}
	}
	if player.Immobilized {
		return &CommandResult{Messages: []string{"You are rooted to the spot!"}}
	}

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You have nowhere to flee!"}}
	}

	type exitInfo struct {
		dir    string
		destID int
	}
	var exits []exitInfo
	for dir, dest := range room.Exits {
		if dest > 0 {
			exits = append(exits, exitInfo{dir, dest})
		}
	}
	if len(exits) == 0 {
		return &CommandResult{Messages: []string{"There is nowhere to flee!"}}
	}

	fleeChance := 50 + player.Quickness/5 + player.Agility/10
	if player.Position != 0 {
		fleeChance -= 20
	}
	if rand.Intn(100) >= fleeChance {
		return &CommandResult{
			Messages:      []string{"You try to flee but can't get away!"},
			RoomBroadcast: []string{fmt.Sprintf("%s tries to flee but fails!", player.FirstName)},
		}
	}

	chosen := exits[rand.Intn(len(exits))]
	e.disengageCombat(player)

	dirName := directionNames[chosen.dir]
	if dirName == "" {
		dirName = strings.ToLower(chosen.dir)
	}

	oldRoom := player.RoomNumber
	player.RoomNumber = chosen.destID
	player.Position = 0
	player.Submitting = false

	result := e.doLook(player)
	result.Messages = append([]string{fmt.Sprintf("You flee %s!", dirName)}, result.Messages...)
	result.OldRoom = oldRoom
	result.OldRoomMsg = []string{fmt.Sprintf("%s flees %s!", player.FirstName, dirName)}
	result.RoomBroadcast = []string{fmt.Sprintf("%s arrives, looking panicked.", player.FirstName)}

	return result
}

func (e *GameEngine) disengageCombat(player *Player) {
	if player.CombatTarget != nil && player.CombatTarget.IsMonster {
		e.monsterMgr.mu.Lock()
		for i := range e.monsterMgr.instances {
			if e.monsterMgr.instances[i].ID == player.CombatTarget.MonsterID {
				if e.monsterMgr.instances[i].Target == player.FirstName {
					e.monsterMgr.instances[i].Target = ""
				}
				break
			}
		}
		e.monsterMgr.mu.Unlock()
	}
	player.CombatTarget = nil
	player.Joined = false
}

// ---- Stances ----

func (e *GameEngine) doStance(player *Player, stance int) *CommandResult {
	if stance == StanceBerserk && player.Race != RaceMurg {
		return &CommandResult{Messages: []string{"Only Murg can enter a berserk frenzy."}}
	}
	player.Stance = stance
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You adopt a %s combat stance.", stanceNames[stance])},
		RoomBroadcast: []string{fmt.Sprintf("%s adopts a %s combat stance.", player.FirstName, stanceNames[stance])},
	}
}

// ---- Search Dead Monster for Loot ----

func (e *GameEngine) doSearchMonster(ctx context.Context, player *Player, args []string) *CommandResult {
	target := strings.ToLower(strings.Join(args, " "))

	if e.monsterMgr == nil {
		return nil
	}

	monsters := e.monsterMgr.AllMonstersInRoom(player.RoomNumber)
	for _, inst := range monsters {
		if inst.Alive {
			continue // can only search dead monsters
		}
		def := e.monsters[inst.DefNumber]
		if def == nil {
			continue
		}
		name := strings.ToLower(FormatMonsterName(def, e.monAdjs))
		noun := strings.ToLower(def.Name)
		if !strings.HasPrefix(name, target) && !strings.HasPrefix(noun, target) {
			continue
		}

		// Check if already searched — mark via monsterMgr
		e.monsterMgr.mu.Lock()
		idx := e.monsterMgr.indexOfID(inst.ID)
		if idx >= 0 && e.monsterMgr.instances[idx].Searched {
			e.monsterMgr.mu.Unlock()
			return &CommandResult{Messages: []string{fmt.Sprintf("You have already searched the %s.", def.Name)}}
		}
		if idx >= 0 {
			e.monsterMgr.instances[idx].Searched = true
		}
		e.monsterMgr.mu.Unlock()

		displayName := FormatMonsterName(def, e.monAdjs)
		var msgs []string
		msgs = append(msgs, fmt.Sprintf("You search %s %s...", articleFor(displayName, def.Unique), displayName))

		// Treasure based on monster's Treasure level
		if def.Treasure > 0 {
			// Generate coins based on treasure level
			copperAmount := rand.Intn(def.Treasure*20) + def.Treasure*5
			gold := copperAmount / 100
			silver := (copperAmount % 100) / 10
			copper := copperAmount % 10

			var found []string
			if gold > 0 {
				player.Gold += gold
				found = append(found, fmt.Sprintf("%d gold", gold))
			}
			if silver > 0 {
				player.Silver += silver
				found = append(found, fmt.Sprintf("%d silver", silver))
			}
			if copper > 0 {
				player.Copper += copper
				found = append(found, fmt.Sprintf("%d copper", copper))
			}
			if len(found) > 0 {
				msgs = append(msgs, fmt.Sprintf("You find %s!", strings.Join(found, ", ")))
			} else {
				msgs = append(msgs, "You find nothing of value.")
			}
		} else {
			msgs = append(msgs, "You find nothing of value.")
		}

		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      msgs,
			RoomBroadcast: []string{fmt.Sprintf("%s searches %s %s.", player.FirstName, articleFor(displayName, def.Unique), displayName)},
			PlayerState:   player,
		}
	}

	return nil // not a dead monster — fall through to normal SEARCH
}

// ---- Monster Guard Behavior ----

// findGuardFor finds a guard monster for the given target monster in the same room.
func (e *GameEngine) findGuardFor(target *MonsterInstance, roomNum int) (*MonsterInstance, *gameworld.MonsterDef) {
	if e.monsterMgr == nil {
		return nil, nil
	}
	targetDef := e.monsters[target.DefNumber]
	if targetDef == nil {
		return nil, nil
	}

	e.monsterMgr.mu.RLock()
	defer e.monsterMgr.mu.RUnlock()

	for i := range e.monsterMgr.instances {
		inst := &e.monsterMgr.instances[i]
		if inst.RoomNumber != roomNum || !inst.Alive || inst.ID == target.ID {
			continue
		}
		def := e.monsters[inst.DefNumber]
		if def == nil {
			continue
		}
		if def.GuardItem == target.DefNumber {
			return inst, def
		}
	}
	return nil, nil
}

// ---- Cry for Law ----

func (e *GameEngine) cryForLaw(attacker *Player, target *MonsterInstance, targetDef *gameworld.MonsterDef) {
	// Find guard/sentry type monsters in nearby rooms and aggro them
	if e.monsterMgr == nil || e.localRoomBroadcast == nil {
		return
	}
	name := FormatMonsterName(targetDef, e.monAdjs)
	e.localRoomBroadcast(attacker.RoomNumber, []string{fmt.Sprintf("%s%s cries out for help!", capArticle(articleFor(name, targetDef.Unique)), name)})

	// Alert guards in the same room
	e.monsterMgr.mu.Lock()
	defer e.monsterMgr.mu.Unlock()
	for i := range e.monsterMgr.instances {
		inst := &e.monsterMgr.instances[i]
		if inst.RoomNumber != attacker.RoomNumber || !inst.Alive || inst.Target != "" || inst.ID == target.ID {
			continue
		}
		def := e.monsters[inst.DefNumber]
		if def == nil {
			continue
		}
		// Guards/sentries (strategy 101-200) will defend
		if def.Strategy >= 101 && def.Strategy <= 200 {
			inst.Target = attacker.FirstName
			guardName := FormatMonsterName(def, e.monAdjs)
			if e.sendToPlayer != nil {
				e.sendToPlayer(attacker.FirstName, []string{fmt.Sprintf("%s%s turns toward you with hostile intent!", capArticle(articleFor(guardName, def.Unique)), guardName)})
			}
		}
	}
}

// ---- Monster Combat AI ----

func (e *GameEngine) monsterCombatTick(inst *MonsterInstance, def *gameworld.MonsterDef) {
	if inst.Target == "" || !inst.Alive {
		return
	}

	if e.sessions == nil {
		return
	}
	var target *Player
	for _, p := range e.sessions.OnlinePlayers() {
		if p.FirstName == inst.Target && p.RoomNumber == inst.RoomNumber && !p.Dead {
			target = p
			break
		}
	}

	if target == nil {
		inst.Target = ""
		return
	}

	e.monsterMgr.mu.Unlock()
	playerMsgs, roomMsgs := e.monsterAttackPlayer(inst, def, target)
	e.monsterMgr.mu.Lock()

	if e.sendToPlayer != nil && len(playerMsgs) > 0 {
		e.sendToPlayer(target.FirstName, playerMsgs)
	}
	if e.localRoomBroadcast != nil && len(roomMsgs) > 0 {
		e.localRoomBroadcast(inst.RoomNumber, roomMsgs)
	}

	// Monster flee behavior (strategy 301-500 = flee when wounded, 501+ = fight to death)
	if inst.Alive && inst.CurrentHP > 0 {
		hpPct := inst.CurrentHP * 100 / max(1, def.Body+def.ExtraBody)
		shouldFlee := false
		switch {
		case def.Strategy >= 501:
			// Fight to death — never flee
		case def.Strategy >= 301 && def.Strategy < 500:
			shouldFlee = hpPct < 30
		case def.Strategy >= 201 && def.Strategy < 300:
			shouldFlee = hpPct < 50
		case def.Strategy >= 1 && def.Strategy < 200:
			shouldFlee = hpPct < 60
		}
		if shouldFlee {
			e.monsterFlee(inst, def)
		}
	}
}

func (e *GameEngine) monsterFlee(inst *MonsterInstance, def *gameworld.MonsterDef) {
	room := e.rooms[inst.RoomNumber]
	if room == nil {
		return
	}
	type exitInfo struct {
		dir    string
		destID int
	}
	var exits []exitInfo
	for dir, dest := range room.Exits {
		if dest > 0 {
			exits = append(exits, exitInfo{dir, dest})
		}
	}
	if len(exits) == 0 {
		return
	}
	chosen := exits[rand.Intn(len(exits))]
	name := FormatMonsterName(def, e.monAdjs)
	dirName := directionNames[chosen.dir]
	if dirName == "" {
		dirName = strings.ToLower(chosen.dir)
	}

	fleeText := def.TextOverrides["TEXF"]
	if fleeText != "" {
		e.localRoomBroadcast(inst.RoomNumber, []string{fleeText + " " + dirName + "."})
	} else {
		article := articleFor(name, def.Unique)
		e.localRoomBroadcast(inst.RoomNumber, []string{fmt.Sprintf("%s%s flees %s!", capArticle(article), name, dirName)})
	}

	inst.Target = ""
	e.monsterMgr.moveMonster(e.monsterMgr.indexOfID(inst.ID), chosen.destID)
}

func (e *GameEngine) monsterCheckAggro(player *Player, roomNum int) {
	if e.monsterMgr == nil || player.Dead || player.Hidden || player.Invisible || player.GMInvis {
		return
	}

	e.monsterMgr.mu.Lock()
	defer e.monsterMgr.mu.Unlock()

	for i := range e.monsterMgr.instances {
		inst := &e.monsterMgr.instances[i]
		if inst.RoomNumber != roomNum || !inst.Alive || inst.Sedated || inst.Target != "" {
			continue
		}
		def := e.monsters[inst.DefNumber]
		if def == nil || def.Strategy < 301 {
			continue
		}
		inst.Target = player.FirstName
		name := FormatMonsterName(def, e.monAdjs)
		article := articleFor(name, def.Unique)
		if e.sendToPlayer != nil {
			e.sendToPlayer(player.FirstName, []string{fmt.Sprintf("%s%s stands erect and closes with you.", capArticle(article), name)})
		}
		break
	}
}

// ---- Helpers ----

func articleFor(name string, unique bool) string {
	if unique {
		return ""
	}
	if len(name) > 0 && strings.ContainsRune("aeiouAEIOU", rune(name[0])) {
		return "an "
	}
	return "a "
}

func capArticle(article string) string {
	if len(article) == 0 {
		return ""
	}
	return strings.ToUpper(article[:1]) + article[1:]
}

func (mm *monsterManager) indexOfID(id int) int {
	for i := range mm.instances {
		if mm.instances[i].ID == id {
			return i
		}
	}
	return -1
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
