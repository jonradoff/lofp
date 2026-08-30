package engine

import (
	"math/rand"
	"strings"
	"time"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// Wound represents a single injury tracked on a player or monster, tied to a
// body location, a damage type (which determines its descriptive vocabulary),
// and a severity level 1-12.
type Wound struct {
	Location   string    `bson:"location" json:"location"`
	DamageType string    `bson:"damageType" json:"damageType"` // "slash", "puncture", "crush", "burn"
	Level      int       `bson:"level" json:"level"`           // 1-12
	Bleeding   bool      `bson:"bleeding" json:"bleeding"`
	CreatedAt  time.Time `bson:"createdAt" json:"createdAt"`
}

// woundWords holds the 12-level severity ladder for each damage type, indexed
// 0-11 for levels 1-12. Levels 1-2 never bleed; for slash/puncture, levels
// 3-12 do (see woundBleeds).
var woundWords = map[string][12]string{
	"slash": {
		"nicked", "cut",
		"slightly lacerated", "lacerated", "badly lacerated", "seriously lacerated",
		"slightly gashed", "gashed", "badly gashed", "seriously gashed", "terribly gashed", "hideously gashed",
	},
	"puncture": {
		"pricked", "stabbed",
		"slightly punctured", "punctured", "badly punctured", "seriously punctured",
		"slightly gored", "gored", "badly gored", "seriously gored", "terribly gored", "hideously gored",
	},
	"crush": {
		"scuffed", "bruised",
		"slightly battered", "battered", "badly battered", "crushed",
		"badly crushed", "seriously crushed", "ruptured", "badly ruptured", "terribly ruptured", "hideously ruptured",
	},
	"burn": {
		"singed", "scorched",
		"slightly burned", "burned", "badly burned", "seriously burned",
		"slightly charred", "charred", "badly charred", "seriously charred", "terribly charred", "hideously charred",
	},
}

// woundWord returns the descriptive word for a damage type at a given
// severity level (1-12, clamped).
func woundWord(damageType string, level int) string {
	if level < 1 {
		level = 1
	}
	if level > 12 {
		level = 12
	}
	words, ok := woundWords[damageType]
	if !ok {
		words = woundWords["crush"]
	}
	return words[level-1]
}

// woundBleeds reports whether a wound of the given damage type and level
// causes bleeding. Only slash and puncture wounds bleed, and only from level
// 3 onward — nicks and cuts (levels 1-2) don't break the skin enough to bleed.
func woundBleeds(damageType string, level int) bool {
	return (damageType == "slash" || damageType == "puncture") && level >= 3
}

// woundLevelFromDamage derives a 1-12 wound severity level from the damage
// dealt as a percentage of the target's max body points.
// TUNABLE — first pass; these bands aren't derived from source material.
func woundLevelFromDamage(dmg, maxHP int) int {
	if maxHP <= 0 {
		return 1
	}
	pct := float64(dmg) / float64(maxHP) * 100
	switch {
	case pct < 2:
		return 1
	case pct < 4:
		return 2
	case pct < 6:
		return 3
	case pct < 8:
		return 4
	case pct < 10:
		return 5
	case pct < 13:
		return 6
	case pct < 16:
		return 7
	case pct < 20:
		return 8
	case pct < 25:
		return 9
	case pct < 32:
		return 10
	case pct < 45:
		return 11
	default:
		return 12
	}
}

// woundBleedDrainPerMinute returns the body points a bleeding wound drains
// per minute. TUNABLE — first pass, calibrated so a level-12 wound ("hideously
// gashed") drains 3 BP/min, matching an original session-log data point.
func woundBleedDrainPerMinute(level int) int {
	if level < 3 {
		return 0
	}
	return (level + 3) / 4
}

// applyWoundToList adds a hit of the given location/damageType/level to a
// wound list, following the "one wound per body part per damage type" rule:
// if a wound already exists at that location with that damage type, its
// severity becomes the worse of the existing level and this hit's level
// (not their sum) — two level-1 nicks to the same spot leave a level-1
// wound, not a level-2 one, matching the original's per-hit (not
// cumulative) wound severity. Different locations/damage types remain
// separate wounds and their bleed drains do stack (see regen.go). canBleed
// is false for undead (RACE 22 monsters, Player.Undead) — they have no
// blood to lose, so their wounds never bleed regardless of type/severity.
func applyWoundToList(wounds []Wound, location, damageType string, level int, canBleed bool) []Wound {
	for i := range wounds {
		if wounds[i].Location == location && wounds[i].DamageType == damageType {
			if level > wounds[i].Level {
				wounds[i].Level = level
			}
			wounds[i].Bleeding = canBleed && woundBleeds(damageType, wounds[i].Level)
			wounds[i].CreatedAt = time.Now()
			return wounds
		}
	}
	return append(wounds, Wound{Location: location, DamageType: damageType, Level: level, Bleeding: canBleed && woundBleeds(damageType, level), CreatedAt: time.Now()})
}

// anyBleeding reports whether any wound in the list is currently bleeding.
func anyBleeding(wounds []Wound) bool {
	for _, w := range wounds {
		if w.Bleeding {
			return true
		}
	}
	return false
}

// buildWoundSentence renders a list of wounds as a natural-language sentence
// fragment, e.g. "a nicked head, a nicked body, a nicked and slightly
// punctured back and a lacerated left hand". Wounds sharing a location are
// grouped together (in first-seen order); if extraSuffix is non-empty (e.g.
// "is dead") it's appended as a final list item.
func buildWoundSentence(wounds []Wound, extraSuffix string) string {
	var order []string
	byLocation := make(map[string][]string)
	for _, w := range wounds {
		if _, seen := byLocation[w.Location]; !seen {
			order = append(order, w.Location)
		}
		byLocation[w.Location] = append(byLocation[w.Location], woundWord(w.DamageType, w.Level))
	}
	items := make([]string, 0, len(order)+1)
	for _, loc := range order {
		words := joinWithAnd(byLocation[loc])
		items = append(items, articleFor(words, false)+words+" "+loc)
	}
	if extraSuffix != "" {
		items = append(items, extraSuffix)
	}
	return joinWithAnd(items)
}

// wolfWoundLocations remaps the humanoid Wound.Location vocabulary to
// quadruped-appropriate terms for display only, when a wolf-form Wolfling's
// wounds are shown on LOOK — the underlying Wound.Location (used by combat,
// healing, and the WOUNDS command) is never changed, this is presentation only.
var wolfWoundLocations = map[string]string{
	"left hand":  "left forepaw",
	"right hand": "right forepaw",
	"left arm":   "left foreleg",
	"right arm":  "right foreleg",
	"left leg":   "left hind leg",
	"right leg":  "right hind leg",
}

// buildWolfWoundSentence is buildWoundSentence with locations run through
// wolfWoundLocations first — see that map's doc comment.
func buildWolfWoundSentence(wounds []Wound, extraSuffix string) string {
	remapped := make([]Wound, len(wounds))
	for i, w := range wounds {
		if alt, ok := wolfWoundLocations[w.Location]; ok {
			w.Location = alt
		}
		remapped[i] = w
	}
	return buildWoundSentence(remapped, extraSuffix)
}

// ---- Weapon/attack -> damage type mapping ----

// damageTypeForWeapon maps a weapon's item type (and, for ambiguous types,
// its noun name) to a wound damage type. nil weaponDef (natural weapons with
// no item behind them) falls through to the default (slash).
func damageTypeForWeapon(weaponDef *gameworld.ItemDef, e *GameEngine) string {
	if weaponDef == nil {
		return "slash"
	}
	switch weaponDef.Type {
	case "SLASH_WEAPON", "TWOHAND_WEAPON", "CLAW_WEAPON", "DRAKIN_SLASH":
		return "slash"
	case "PUNCTURE_WEAPON", "STABTHROWN", "BITE_WEAPON", "CHARGE_WEAPON", "BOW_WEAPON", "DRAKIN_THROWN":
		return "puncture"
	case "CRUSH_WEAPON", "BLUNT_WEAPON", "FIST_WEAPON", "DRAKIN_CRUSH":
		return "crush"
	case "POLETHROWN":
		return "puncture"
	case "POLE_WEAPON", "DRAKIN_POLE":
		name := strings.ToLower(e.nouns[weaponDef.NameID])
		switch {
		case strings.Contains(name, "lance"), strings.Contains(name, "pike"), strings.Contains(name, "trident"):
			return "puncture"
		case strings.Contains(name, "staff"), strings.Contains(name, "whip"), strings.Contains(name, "pole"):
			return "crush"
		default:
			return "slash"
		}
	case "THROWN_WEAPON":
		name := strings.ToLower(e.nouns[weaponDef.NameID])
		if strings.Contains(name, "boulder") || strings.Contains(name, "rock") || strings.Contains(name, "stone") {
			return "crush"
		}
		return "puncture"
	default:
		return "slash"
	}
}

// damageTypeForSpecAttack maps a monster's SpecDmgType (stored uppercase by
// the script parser: HEAT, COLD, ELECTRIC, SLASH, CRUSH) to a wound damage
// type. Heat/cold/electric are all described as burns per design.
func damageTypeForSpecAttack(specDmgType string) string {
	switch strings.ToUpper(specDmgType) {
	case "HEAT", "COLD", "ELECTRIC":
		return "burn"
	case "SLASH":
		return "slash"
	case "CRUSH":
		return "crush"
	default:
		return "crush"
	}
}

// dmgNounForType returns the generic combat-message noun for a wound damage
// type, matching the vocabulary attackVerb() already uses for weapon hits.
func dmgNounForType(damageType string) string {
	switch damageType {
	case "slash":
		return "slash"
	case "burn":
		return "burn"
	default:
		return "strike"
	}
}

// ---- Hit location roll ----

type locationWeight struct {
	name   string
	weight int
	mult   int // damage multiplier percentage: 100 vitals, 40 limb, 20 extremity
}

var humanLocationWeights = []locationWeight{
	{"head", 15, 100},
	{"body", 25, 100},
	{"back", 10, 100},
	{"right arm", 10, 40},
	{"left arm", 10, 40},
	{"right leg", 10, 40},
	{"left leg", 10, 40},
	{"right hand", 5, 20},
	{"left hand", 5, 20},
}

var animalLocationWeights = []locationWeight{
	{"head", 20, 100},
	{"body", 30, 100},
	{"right foreleg", 10, 40},
	{"left foreleg", 10, 40},
	{"right hind leg", 10, 40},
	{"left hind leg", 10, 40},
	{"tail", 4, 20},
	{"right paw", 3, 20},
	{"left paw", 3, 20},
}

// rollBodyPart picks a hit location for bodyType ("ANIMAL"/"AVINE" use the
// animal table, everything else the human table), returning the location
// name and its damage multiplier (100/40/20). specRank (0-5, weapon
// specialization rank) shifts 5 percentage points per rank from the
// limb/extremity pool into head+body specifically (not back) — pass 0 for
// monsters, which have no specialization.
func rollBodyPart(bodyType string, specRank int) (string, int) {
	base := humanLocationWeights
	if bodyType == "ANIMAL" || bodyType == "AVINE" {
		base = animalLocationWeights
	}

	weights := make([]int, len(base))
	for i, lw := range base {
		weights[i] = lw.weight
	}

	if specRank > 0 {
		shift := 5 * specRank
		if shift > 25 {
			shift = 25
		}
		headBodyTotal := 0
		otherTotal := 0
		for _, lw := range base {
			if lw.name == "head" || lw.name == "body" {
				headBodyTotal += lw.weight
			} else if lw.name != "back" {
				otherTotal += lw.weight
			}
		}
		if headBodyTotal > 0 && otherTotal > 0 {
			applied := 0
			for i, lw := range base {
				if lw.name == "head" || lw.name == "body" {
					add := shift * lw.weight / headBodyTotal
					weights[i] += add
					applied += add
				}
			}
			for i, lw := range base {
				if lw.name != "head" && lw.name != "body" && lw.name != "back" {
					sub := applied * lw.weight / otherTotal
					if sub > weights[i] {
						sub = weights[i]
					}
					weights[i] -= sub
				}
			}
		}
	}

	total := 0
	for _, w := range weights {
		total += w
	}
	if total <= 0 {
		return base[0].name, base[0].mult
	}
	roll := rand.Intn(total)
	cum := 0
	for i, w := range weights {
		cum += w
		if roll < cum {
			return base[i].name, base[i].mult
		}
	}
	last := len(base) - 1
	return base[last].name, base[last].mult
}
