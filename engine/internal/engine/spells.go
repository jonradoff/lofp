package engine

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/metavert-io/lofp/internal/gameworld"
)

// SpellDef defines a spell in the game.
type SpellDef struct {
	ID       int
	Name     string
	School   string
	Level    int
	ManaCost int
	CastTime int    // seconds
	Effect   string // "damage", "heal", "defense", "buff", "utility"
	DmgMin   int
	DmgMax   int
	HealMin  int
	HealMax  int
	DefBonus int
	DmgType  string        // "heat", "cold", "electric", "crushing", ""
	Duration time.Duration // seconds; 0 = instant/permanent
	Family   string        // "", "agility", "strength", "armor", etc.
	Ticks    int           `bson:"ticks,omitempty" json:"ticks,omitempty"` // number of ticks for periodic effects

	StatusType StatID
	StatusMsg  string
}

// spellRegistry holds all defined spells.
var spellRegistry []SpellDef

func init() {
	// Conjuration (100-144)
	conj := []SpellDef{
		{ID: 100, Name: "Flame Bolt", School: "Conjuration", Level: 1, ManaCost: 3, CastTime: 3, Effect: "damage", DmgMin: 3, DmgMax: 12, DmgType: "heat"},
		{ID: 101, Name: "Force Blade", School: "Conjuration", Level: 3, ManaCost: 5, CastTime: 3, Effect: "damage", DmgMin: 5, DmgMax: 18, DmgType: "force", StatusMsg: "Air coalesces from %s's hand and slams at %s with incredible force!"},
		{ID: 102, Name: "Mystic Armor", School: "Conjuration", Level: 5, ManaCost: 8, CastTime: 3, Effect: "defense", DefBonus: 20, Duration: 30 * time.Minute, Family: "armor", StatusType: DefensiveBuff},
		{ID: 103, Name: "Lightning Bolt", School: "Conjuration", Level: 7, ManaCost: 10, CastTime: 3, Effect: "damage", DmgMin: 8, DmgMax: 30, DmgType: "electric"},
		{ID: 105, Name: "Globe of Protection", School: "Conjuration", Level: 15, ManaCost: 20, CastTime: 3, Effect: "defense", DefBonus: 50, Duration: 90 * time.Minute, Family: "armor", StatusType: DefensiveBuff},
		{ID: 106, Name: "Summon Fire Elemental", School: "Conjuration", Level: 12, ManaCost: 25, CastTime: 5, Effect: "summon"},
		{ID: 107, Name: "Summon Air Elemental", School: "Conjuration", Level: 12, ManaCost: 25, CastTime: 5, Effect: "summon"},
		{ID: 108, Name: "Summon Water Elemental", School: "Conjuration", Level: 12, ManaCost: 25, CastTime: 5, Effect: "summon"},
		{ID: 109, Name: "Summon Gargoyle", School: "Conjuration", Level: 16, ManaCost: 30, CastTime: 5, Effect: "summon"},
		{ID: 112, Name: "Call Meteor", School: "Conjuration", Level: 20, ManaCost: 30, CastTime: 4, Effect: "damage", DmgMin: 25, DmgMax: 60, DmgType: "heat"},
		{ID: 113, Name: "Light", School: "Conjuration", Level: 1, ManaCost: 2, CastTime: 2, Effect: "buff", Family: "light", Duration: 30 * time.Minute, StatusType: LightBuff, StatusMsg: "A soft light surrounds you."},
		{ID: 114, Name: "Mystic Key", School: "Conjuration", Level: 2, ManaCost: 10, CastTime: 5, Effect: "utility"},
		{ID: 115, Name: "Shockwave", School: "Conjuration", Level: 4, ManaCost: 6, CastTime: 3, Effect: "damage", DmgMin: 4, DmgMax: 15, DmgType: "crushing"},
		{ID: 116, Name: "Thunder Call", School: "Conjuration", Level: 21, ManaCost: 28, CastTime: 4, Effect: "damage", DmgMin: 20, DmgMax: 50, DmgType: "electric"},
		{ID: 117, Name: "Call Fire", School: "Conjuration", Level: 8, ManaCost: 12, CastTime: 3, Effect: "damage", DmgMin: 10, DmgMax: 25, DmgType: "heat"},
		{ID: 118, Name: "Flaming Sphere", School: "Conjuration", Level: 13, ManaCost: 18, CastTime: 3, Effect: "damage", DmgMin: 15, DmgMax: 40, DmgType: "heat"},
		{ID: 119, Name: "Ice Bolt", School: "Conjuration", Level: 3, ManaCost: 5, CastTime: 3, Effect: "damage", DmgMin: 4, DmgMax: 16, DmgType: "cold"},
		{ID: 120, Name: "Frost Ray", School: "Conjuration", Level: 6, ManaCost: 8, CastTime: 3, Effect: "damage", DmgMin: 7, DmgMax: 22, DmgType: "cold"},
		{ID: 121, Name: "Freezing Sphere", School: "Conjuration", Level: 9, ManaCost: 14, CastTime: 3, Effect: "damage", DmgMin: 10, DmgMax: 30, DmgType: "cold"},
		{ID: 122, Name: "Summon Familiar", School: "Conjuration", Level: 2, ManaCost: 10, CastTime: 5, Effect: "summon"},
		{ID: 123, Name: "Summon Earth Elemental", School: "Conjuration", Level: 12, ManaCost: 25, CastTime: 5, Effect: "summon"},
		{ID: 124, Name: "Inferno Glyph", School: "Conjuration", Level: 20, ManaCost: 25, CastTime: 4, Effect: "damage", DmgMin: 20, DmgMax: 55, DmgType: "heat"},
		{ID: 125, Name: "Thunder Glyph", School: "Conjuration", Level: 10, ManaCost: 15, CastTime: 3, Effect: "damage", DmgMin: 12, DmgMax: 30, DmgType: "electric"},
		{ID: 126, Name: "Ice Glyph", School: "Conjuration", Level: 15, ManaCost: 20, CastTime: 3, Effect: "damage", DmgMin: 15, DmgMax: 40, DmgType: "cold"},
		{ID: 127, Name: "Web", School: "Conjuration", Level: 10, ManaCost: 12, CastTime: 3, Effect: "debuff", Duration: 2 * time.Minute, DefBonus: -25, StatusMsg: "Thick strands of sticky webbing cover %s!", StatusType: RestrainedEffect},
		{ID: 130, Name: "Mass Protection", School: "Conjuration", Level: 23, ManaCost: 30, CastTime: 4, Effect: "defense", DefBonus: 25, Duration: 45 * time.Minute, Family: "armor"},
		{ID: 131, Name: "Flaming Arrows", School: "Conjuration", Level: 18, ManaCost: 22, CastTime: 3, Effect: "damage", DmgMin: 15, DmgMax: 35, DmgType: "heat"},
		{ID: 132, Name: "Chain Lightning", School: "Conjuration", Level: 23, ManaCost: 28, CastTime: 4, Effect: "damage", DmgMin: 20, DmgMax: 50, DmgType: "electric"},
		{ID: 133, Name: "Globe of Protection II", School: "Conjuration", Level: 30, ManaCost: 40, CastTime: 4, Effect: "defense", DefBonus: 100, Duration: 90 * time.Minute, Family: "armor"},
		{ID: 134, Name: "Siryx's Terrible Tentacles", School: "Conjuration", Level: 25, ManaCost: 35, CastTime: 4, Effect: "damage", DmgMin: 20, DmgMax: 55, DmgType: "crushing"},
		{ID: 135, Name: "Storm Blade", School: "Conjuration", Level: 24, ManaCost: 30, CastTime: 3, Effect: "buff"},
		{ID: 136, Name: "Inferno Blade", School: "Conjuration", Level: 19, ManaCost: 25, CastTime: 3, Effect: "buff"},
		{ID: 137, Name: "Winter Blade", School: "Conjuration", Level: 22, ManaCost: 28, CastTime: 3, Effect: "buff"},
		{ID: 138, Name: "Energy Maelstrom", School: "Conjuration", Level: 31, ManaCost: 45, CastTime: 5, Effect: "damage", DmgMin: 30, DmgMax: 75, DmgType: "electric"},
		{ID: 141, Name: "Pyrotechnics", School: "Conjuration", Level: 17, ManaCost: 20, CastTime: 3, Effect: "damage", DmgMin: 12, DmgMax: 35, DmgType: "heat"},
	}
	// Enchantment (200-250)
	ench := []SpellDef{
		{ID: 200, Name: "Fear", School: "Enchantment", Level: 1, ManaCost: 3, CastTime: 3, Effect: "debuff", Duration: 5 * time.Second, Ticks: 5, StatusMsg: "%s is overcome with fear!", StatusType: FearEffect},
		{ID: 201, Name: "Charm", School: "Enchantment", Level: 3, ManaCost: 8, CastTime: 3, Effect: "debuff", Duration: 5 * time.Second, StatusType: CharmEffect, Ticks: 5, StatusMsg: "%s gets a friendly look."},
		{ID: 202, Name: "Enchantment I", School: "Enchantment", Level: 5, ManaCost: 10, CastTime: 4, Effect: "buff", Duration: 45 * time.Minute, Family: "enchantment"},
		{ID: 207, Name: "Strength I", School: "Enchantment", Level: 4, ManaCost: 6, CastTime: 3, Effect: "buff", DefBonus: 10, Duration: 30 * time.Minute, Family: "strength", StatusType: StatStrength},
		{ID: 208, Name: "Strength II", School: "Enchantment", Level: 8, ManaCost: 10, CastTime: 3, Effect: "buff", DefBonus: 20, Duration: 45 * time.Minute, Family: "strength", StatusType: StatStrength},
		{ID: 209, Name: "Strength III", School: "Enchantment", Level: 16, ManaCost: 18, CastTime: 3, Effect: "buff", DefBonus: 30, Duration: 60 * time.Minute, Family: "strength", StatusType: StatStrength},
		{ID: 210, Name: "Haste", School: "Enchantment", Level: 5, ManaCost: 8, CastTime: 3, Effect: "buff", Duration: 60 * time.Minute, Family: "haste"},
		{ID: 211, Name: "Slow", School: "Enchantment", Level: 5, ManaCost: 8, CastTime: 3, Effect: "utility", Duration: 60 * time.Minute, Family: "slow"},
		{ID: 216, Name: "Slumber I", School: "Enchantment", Level: 2, ManaCost: 4, CastTime: 3, Effect: "utility"},
		{ID: 219, Name: "Silence", School: "Enchantment", Level: 7, ManaCost: 10, CastTime: 3, Effect: "utility"},
		{ID: 220, Name: "Dancing Blade", School: "Enchantment", Level: 1, ManaCost: 3, CastTime: 3, Effect: "damage", DmgMin: 1, DmgMax: 8, DmgType: "slashing", StatusMsg: "An animate blade leaps forth from %s's hand and slashes at %s with expert skill!"},
		{ID: 221, Name: "Dancing Sword", School: "Enchantment", Level: 6, ManaCost: 8, CastTime: 3, Effect: "damage", DmgMin: 1, DmgMax: 13, DmgType: "slashing", StatusMsg: "An animate sword leaps forth from %s's hand and slashes at %s with expert skill!"},
		{ID: 224, Name: "Fly", School: "Enchantment", Level: 11, ManaCost: 15, CastTime: 3, Effect: "buff", Duration: 45 * time.Minute},
		{ID: 225, Name: "Invisibility", School: "Enchantment", Level: 14, ManaCost: 18, CastTime: 3, Effect: "buff", Duration: 35 * time.Minute, StatusType: InvisibilityEffect, StatusMsg: "%s fades from view."},
		{ID: 228, Name: "Identify", School: "Enchantment", Level: 7, ManaCost: 5, CastTime: 3, Effect: "utility"},
		{ID: 229, Name: "Wizard's Armor", School: "Enchantment", Level: 9, ManaCost: 12, CastTime: 3, Effect: "defense", DefBonus: 15, Duration: 45 * time.Minute},
		{ID: 234, Name: "Spell Shield", School: "Enchantment", Level: 13, ManaCost: 15, CastTime: 3, Effect: "defense", DefBonus: 25, Duration: 45 * time.Minute},
		{ID: 235, Name: "Cloak Mind", School: "Enchantment", Level: 22, ManaCost: 25, CastTime: 3, Effect: "defense", DefBonus: 25, Duration: 45 * time.Minute},
	}
	// Necromancy (301-356)
	necro := []SpellDef{
		{ID: 301, Name: "Turn Undead I", School: "Necromancy", Level: 2, ManaCost: 4, CastTime: 3, Effect: "damage", DmgMin: 3, DmgMax: 8, DmgType: "undead", Duration: 30 * time.Second, StatusMsg: "%s is overcome with fear!"},
		{ID: 302, Name: "Turn Undead II", School: "Necromancy", Level: 8, ManaCost: 10, CastTime: 3, Effect: "damage", DmgMin: 6, DmgMax: 15, DmgType: "undead", Duration: 30 * time.Second, StatusMsg: "%s is overcome with fear!"},
		{ID: 304, Name: "Turn Undead III", School: "Necromancy", Level: 16, ManaCost: 22, CastTime: 3, Effect: "damage", DmgMin: 10, DmgMax: 25, DmgType: "undead", Duration: 30 * time.Second, StatusMsg: "%s is overcome with fear!"},
		{ID: 313, Name: "Body Destruction I", School: "Necromancy", Level: 1, ManaCost: 3, CastTime: 3, Effect: "damage", DmgMin: 3, DmgMax: 10, DmgType: ""},
		{ID: 314, Name: "Body Destruction II", School: "Necromancy", Level: 5, ManaCost: 7, CastTime: 3, Effect: "damage", DmgMin: 6, DmgMax: 20, DmgType: ""},
		{ID: 315, Name: "Body Destruction III", School: "Necromancy", Level: 10, ManaCost: 14, CastTime: 3, Effect: "damage", DmgMin: 12, DmgMax: 35, DmgType: ""},
		{ID: 316, Name: "Body Restoration I", School: "Necromancy", Level: 1, ManaCost: 3, CastTime: 3, Effect: "heal", HealMin: 5, HealMax: 15, StatusType: StatBodyPoint},
		{ID: 317, Name: "Body Restoration II", School: "Necromancy", Level: 5, ManaCost: 7, CastTime: 3, Effect: "heal", HealMin: 10, HealMax: 30, StatusType: StatBodyPoint},
		{ID: 318, Name: "Body Restoration III", School: "Necromancy", Level: 10, ManaCost: 14, CastTime: 3, Effect: "heal", HealMin: 20, HealMax: 50, StatusType: StatBodyPoint},
		{ID: 323, Name: "Spectral Fist", School: "Necromancy", Level: 3, ManaCost: 5, CastTime: 3, Effect: "damage", DmgMin: 4, DmgMax: 14, DmgType: "crushing"},
		{ID: 326, Name: "Spectral Shield", School: "Necromancy", Level: 9, ManaCost: 12, CastTime: 3, Effect: "defense", DefBonus: 20, Duration: 45 * time.Minute, StatusType: DefensiveBuff},
		{ID: 334, Name: "Invigoration I", School: "Necromancy", Level: 2, ManaCost: 4, CastTime: 3, Effect: "heal", HealMin: 3, HealMax: 10, StatusType: StatFatigue},
		{ID: 335, Name: "Invigoration II", School: "Necromancy", Level: 9, ManaCost: 10, CastTime: 3, Effect: "heal", HealMin: 8, HealMax: 25, StatusType: StatFatigue},
		{ID: 337, Name: "Reconstruction", School: "Necromancy", Level: 4, ManaCost: 6, CastTime: 3, Effect: "heal", HealMin: 5, HealMax: 20},
		{ID: 338, Name: "Unstun", School: "Necromancy", Level: 9, ManaCost: 8, CastTime: 2, Effect: "utility"},
		{ID: 339, Name: "Destroy Undead I", School: "Necromancy", Level: 3, ManaCost: 5, CastTime: 3, Effect: "damage", DmgMin: 8, DmgMax: 20, DmgType: "undead"},
		{ID: 340, Name: "Destroy Undead II", School: "Necromancy", Level: 8, ManaCost: 12, CastTime: 3, Effect: "damage", DmgMin: 15, DmgMax: 40, DmgType: "undead"},
		{ID: 341, Name: "Destroy Undead III", School: "Necromancy", Level: 13, ManaCost: 20, CastTime: 3, Effect: "damage", DmgMin: 25, DmgMax: 60, DmgType: "undead"},
		{ID: 343, Name: "Regeneration", School: "Necromancy", Level: 27, ManaCost: 35, CastTime: 4, Effect: "heal", HealMin: 40, HealMax: 80},
		{ID: 345, Name: "Spectral Sword", School: "Necromancy", Level: 7, ManaCost: 10, CastTime: 3, Effect: "damage", DmgMin: 6, DmgMax: 22, DmgType: ""},
		{ID: 347, Name: "Divine Blessing", School: "Necromancy", Level: 10, ManaCost: 12, CastTime: 3, Effect: "buff", Duration: 45 * time.Minute},
		{ID: 354, Name: "Rorin's Fire", School: "Necromancy", Level: 17, ManaCost: 22, CastTime: 3, Effect: "damage", DmgMin: 15, DmgMax: 40, DmgType: "heat"},
		{ID: 355, Name: "Summon Greater Wight", School: "Necromancy", Level: 17, ManaCost: 35, CastTime: 5, Effect: "summon"},
	}
	// General (400-415)
	gen := []SpellDef{
		{ID: 400, Name: "Detect Magic", School: "General", Level: 1, ManaCost: 2, CastTime: 2, Effect: "utility"},
		{ID: 401, Name: "Dispel Lesser Magic", School: "General", Level: 5, ManaCost: 8, CastTime: 3, Effect: "utility"},
		{ID: 403, Name: "Mindlink", School: "General", Level: 9, ManaCost: 12, CastTime: 3, Effect: "buff", Family: "mind", Duration: 90 * time.Minute},
		{ID: 405, Name: "See Hidden", School: "General", Level: 3, ManaCost: 5, CastTime: 3, Effect: "utility"},
		{ID: 406, Name: "Dispel Invisibility", School: "General", Level: 8, ManaCost: 10, CastTime: 3, Effect: "utility"},
		{ID: 407, Name: "Analyze Ore", School: "General", Level: 3, ManaCost: 4, CastTime: 3, Effect: "utility"},
		{ID: 303, Name: "Cure Poison", School: "General", Level: 11, ManaCost: 12, CastTime: 3, Effect: "heal", StatusType: RemovePoison, StatusMsg: "A blue light flashes around %d."},
		{ID: 319, Name: "Cure Disease", School: "General", Level: 12, ManaCost: 16, CastTime: 3, Effect: "heal", StatusType: RemoveDisease, StatusMsg: "A purple light flashes around %d."},
	}
	// Druidic (500-538)
	druid := []SpellDef{
		{ID: 500, Name: "Plant Snare", School: "Druidic", Level: 4, ManaCost: 6, CastTime: 3, Effect: "debuff", Duration: 2 * time.Minute, DefBonus: -25, StatusMsg: "Plant tendrils shoot up from the ground and ensnare %s!", StatusType: RestrainedEffect},
		{ID: 504, Name: "Call Animal", School: "Druidic", Level: 1, ManaCost: 5, CastTime: 5, Effect: "summon"},
		{ID: 505, Name: "Freedom", School: "Druidic", Level: 9, ManaCost: 12, CastTime: 3, Effect: "utility"},
		{ID: 506, Name: "Resist Weather", School: "Druidic", Level: 3, ManaCost: 6, CastTime: 3, Effect: "buff", Duration: 45 * time.Minute, Family: "weather", StatusType: ResistWeatherEffect, StatusMsg: "Wind circles you in a sudden spiral, carrying the scent of earth and rain before fading into stillness."},
		{ID: 507, Name: "Heat Shield", School: "Druidic", Level: 7, ManaCost: 10, CastTime: 3, Effect: "buff", DefBonus: 50, Duration: 45 * time.Minute, Family: "heat shield", StatusType: ColdResistance, StatusMsg: "A translucent blue sphere surrounds you."},
		{ID: 508, Name: "Cold Shield", School: "Druidic", Level: 6, ManaCost: 8, CastTime: 3, Effect: "buff", DefBonus: 50, Duration: 45 * time.Minute, Family: "cold shield", StatusType: HeatResistance, StatusMsg: "A translucent red sphere surrounds you."},
		{ID: 511, Name: "Carapace", School: "Druidic", Level: 8, ManaCost: 10, CastTime: 3, Effect: "defense", DefBonus: 20, Duration: 45 * time.Minute},
		{ID: 512, Name: "True Aim", School: "Druidic", Level: 15, ManaCost: 18, CastTime: 3, Effect: "buff"},
		{ID: 513, Name: "Agility I", School: "Druidic", Level: 4, ManaCost: 6, CastTime: 3, Effect: "buff", DefBonus: 10, Duration: 30 * time.Minute, Family: "agility", StatusType: StatAgility},
		{ID: 514, Name: "Agility II", School: "Druidic", Level: 11, ManaCost: 12, CastTime: 3, Effect: "buff", DefBonus: 20, Duration: 45 * time.Minute, Family: "agility", StatusType: StatAgility},
		{ID: 515, Name: "Agility III", School: "Druidic", Level: 16, ManaCost: 20, CastTime: 3, Effect: "buff", DefBonus: 30, Duration: 60 * time.Minute, Family: "agility", StatusType: StatAgility},
		{ID: 517, Name: "Sticks to Snake", School: "Druidic", Level: 5, ManaCost: 8, CastTime: 5, Effect: "summon", StatusMsg: "You call upon the power of nature to transform a stick into a snake!"},
		{ID: 518, Name: "Claw Growth", School: "Druidic", Level: 2, ManaCost: 4, CastTime: 3, Effect: "buff", DefBonus: 4, Duration: 90 * time.Minute, Family: "clawgrowth", StatusType: ClawGrowth},
		{ID: 519, Name: "Sunray", School: "Druidic", Level: 13, ManaCost: 18, CastTime: 3, Effect: "damage", DmgMin: 12, DmgMax: 35, DmgType: "heat"},
		{ID: 520, Name: "Night Vision", School: "Druidic", Level: 1, ManaCost: 2, CastTime: 2, Effect: "buff", Duration: 45 * time.Minute, Family: "nightvision", StatusType: NightVisionBuff, StatusMsg: "Your eyes adjust to the darkness."},
		{ID: 521, Name: "Camouflage", School: "Druidic", Level: 7, ManaCost: 8, CastTime: 3, Effect: "buff", Duration: 45 * time.Minute},
		{ID: 523, Name: "Earth Spike", School: "Druidic", Level: 5, ManaCost: 7, CastTime: 3, Effect: "damage", DmgMin: 5, DmgMax: 18, DmgType: "crushing", StatusMsg: "A jagged spike of earth erupts beneath %s, impaling it!"},
		{ID: 524, Name: "Earth Wave", School: "Druidic", Level: 12, ManaCost: 16, CastTime: 3, Effect: "damage", DmgMin: 10, DmgMax: 30, DmgType: "crushing", StatusMsg: "The ground shakes violently as a wave of earth crashes down on %s!"},
	}

	spellRegistry = append(spellRegistry, conj...)
	spellRegistry = append(spellRegistry, ench...)
	spellRegistry = append(spellRegistry, necro...)
	spellRegistry = append(spellRegistry, gen...)
	spellRegistry = append(spellRegistry, druid...)
}

var summonComponents = map[int]string{
	122: "termite eye",      // Summon Familiar
	106: "garnet",           // Fire Elemental
	107: "opal",             // Air Elemental
	108: "aquamarine",       // Water Elemental
	109: "diamond",          // Gargoyle
	123: "tourmaline stone", // Earth Elemental
	355: "wight dust",       // Greater Wight
}

// FindSpellByID returns a spell by its numeric ID.
func FindSpellByID(id int) *SpellDef {
	for i := range spellRegistry {
		if spellRegistry[i].ID == id {
			return &spellRegistry[i]
		}
	}
	return nil
}

// FindSpellByName finds a spell by prefix match on name.
func FindSpellByName(input string) *SpellDef {
	input = strings.ToLower(input)
	for i := range spellRegistry {
		if strings.ToLower(spellRegistry[i].Name) == input {
			return &spellRegistry[i]
		}
	}
	var match *SpellDef
	for i := range spellRegistry {
		if strings.HasPrefix(strings.ToLower(spellRegistry[i].Name), input) {
			if match != nil {
				return nil // ambiguous
			}
			match = &spellRegistry[i]
		}
	}
	return match
}

// spellSchoolSkill returns the skill ID for a spell school.
func spellSchoolSkill(school string) int {
	switch school {
	case "Conjuration":
		return 7
	case "Enchantment":
		return 14
	case "Necromancy":
		return 30
	case "General":
		return 23 // Spellcraft
	case "Druidic":
		return 17
	default:
		return 23
	}
}

// doLearn handles the LEARN command — learn a spell from a scroll.
// The scroll's Val3 holds the spell number. The player must have the
// appropriate magic school skill at a sufficient level.
func (e *GameEngine) doLearn(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Learn from what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target = strings.TrimPrefix(target, "my ")
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		if !strings.Contains(strings.ToUpper(itemDef.Type), "SCROLL") {
			continue
		}
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}

		spellNum := ii.Val3
		if spellNum == 0 {
			return &CommandResult{Messages: []string{"This scroll holds no magical inscription."}}
		}

		spell := FindSpellByID(spellNum)
		if spell == nil {
			return &CommandResult{Messages: []string{"The scroll's magic is beyond comprehension."}}
		}

		// Check if already known
		if player.KnownSpells != nil {
			if _, known := player.KnownSpells[spellNum]; known {
				return &CommandResult{Messages: []string{fmt.Sprintf("You already know %s.", spell.Name)}}
			}
		}

		// Map spell school name to required skill ID
		requiredSkill := schoolSkillID(spell.School)
		if requiredSkill < 0 {
			return &CommandResult{Messages: []string{"You cannot learn spells of that school."}}
		}

		// Player must have the school skill at a level >= spell level
		playerSkillLevel := player.Skills[requiredSkill]
		if playerSkillLevel < spell.Level {
			return &CommandResult{Messages: []string{
				fmt.Sprintf("You need %s rank %d to learn %s (you have rank %d).",
					SkillNames[requiredSkill], spell.Level, spell.Name, playerSkillLevel),
			}}
		}

		// Consume the scroll and add the spell
		fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
		player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
		if player.KnownSpells == nil {
			player.KnownSpells = make(map[int]bool)
		}
		player.KnownSpells[spellNum] = true

		player.RoundTimeExpiry = time.Now().Add(5 * time.Second)
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("You study %s carefully...", fullName),
				fmt.Sprintf("You learn %s! The scroll crumbles to dust.", spell.Name),
				"[Round: 5 sec]",
			},
			RoomBroadcast: []string{
				fmt.Sprintf("%s studies a scroll, which crumbles away.", player.FirstName),
			},
		}
	}
	return &CommandResult{Messages: []string{"You don't have that."}}
}

// schoolSkillID returns the skill ID required for a given magic school name.
// Returns -1 if the school is unknown.
func schoolSkillID(school string) int {
	switch strings.ToLower(school) {
	case "conjuration":
		return 7
	case "enchantment":
		return 14
	case "druidic":
		return 17
	case "general":
		return 23 // Spellcraft
	case "necromancy":
		return 30
	default:
		return -1
	}
}

// doPrepareSpell handles PREPARE/INVOKE <spell>.
func (e *GameEngine) doPrepareSpell(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Prepare what spell?"}}
	}

	if player.Dead {
		return &CommandResult{Messages: []string{"You can't cast spells while dead."}}
	}

	if player.Unconscious {
		return &CommandResult{Messages: []string{"You can't cast spells while unconscous."}}
	}
	// Split:
	// PREPARE <spell> WITH <component>
	withIndex := -1
	for i, arg := range args {
		if strings.EqualFold(arg, "with") {
			withIndex = i
			break
		}
	}

	spellArgs := args
	componentArgs := []string{}

	if withIndex >= 0 {
		spellArgs = args[:withIndex]

		if withIndex+1 < len(args) {
			componentArgs = args[withIndex+1:]
		}
	}

	if len(spellArgs) == 0 {
		return &CommandResult{Messages: []string{"Prepare what spell?"}}
	}

	spellText := strings.Join(spellArgs, " ")

	// Allow both numeric IDs and spell names.
	var spell *SpellDef

	if spellID, err := strconv.Atoi(spellText); err == nil {
		spell = FindSpellByID(spellID)
	} else {
		spell = FindSpellByName(spellText)
	}

	if spell == nil {
		return &CommandResult{Messages: []string{"You don't know that spell."}}
	}

	if !player.KnownSpells[spell.ID] && !player.IsGM {
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("You haven't learned %s.", spell.Name),
			},
		}
	}

	if player.Mana < spell.ManaCost {
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You don't have enough mana. (%s costs %d, you have %d)",
					spell.Name,
					spell.ManaCost,
					player.Mana,
				),
			},
		}
	}

	// ---------------------------------------------------------
	// MATERIAL COMPONENT
	// ---------------------------------------------------------

	componentName, needsComponent := summonComponents[spell.ID]

	// Aelfen do not need a material component for Summon Familiar.
	if spell.ID == 122 && player.Race == RaceAelfen {
		needsComponent = false
	}

	if needsComponent {
		// No WITH supplied.
		if len(componentArgs) == 0 {
			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"This spell requires a material component: %s",
						componentName,
					),
					fmt.Sprintf(
						`[Use "PREPARE %d WITH %s" to properly invoke this spell]`,
						spell.ID,
						componentName,
					),
				},
			}
		}

		componentTarget := strings.ToLower(strings.Join(componentArgs, " "))
		componentTarget = strings.TrimSpace(componentTarget)
		componentTarget = strings.TrimPrefix(componentTarget, "my ")

		foundIndex := -1
		foundName := ""

		for i, ii := range player.Inventory {
			itemDef := e.items[ii.Archetype]
			if itemDef == nil {
				continue
			}

			fullName := strings.ToLower(
				e.formatItemName(
					itemDef,
					ii.Adj1,
					ii.Adj2,
					ii.Adj3,
				),
			)

			noun := strings.ToLower(e.getItemNounName(itemDef))

			// Strip articles from formatted item name.
			checkName := strings.TrimSpace(fullName)
			checkName = strings.TrimPrefix(checkName, "a ")
			checkName = strings.TrimPrefix(checkName, "an ")
			checkName = strings.TrimPrefix(checkName, "some ")
			checkName = strings.TrimPrefix(checkName, "the ")

			// The required component must actually be present
			// in the item's resolved name.
			required := strings.ToLower(componentName)

			if strings.Contains(checkName, required) ||
				strings.EqualFold(noun, required) {

				// If the player supplied a component name,
				// make sure this particular item matches it too.
				if componentTarget == "" ||
					strings.Contains(checkName, componentTarget) ||
					strings.Contains(componentTarget, checkName) {

					foundIndex = i
					foundName = fullName
					break
				}
			}
		}

		if foundIndex < 0 {
			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"You do not have the required material component: %s.",
						componentName,
					),
				},
			}
		}

		// Consume component AT PREPARE TIME.
		player.Inventory = append(
			player.Inventory[:foundIndex],
			player.Inventory[foundIndex+1:]...,
		)

		// Keep article/capitalization close to original game text.
		foundName = strings.TrimSpace(foundName)

		if len(foundName) > 0 {
			foundName = strings.ToUpper(foundName[:1]) + foundName[1:]
		}

		// Finish preparation below, but remember the component message.
		player.PreparedSpell = spell.ID

		castTime := e.spellCastTime(player, spell)

		player.RoundTimeExpiry = time.Now().Add(time.Duration(castTime) * time.Second)

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"%s turns to dust as it is absorbed by the magic of your spell!",
					foundName,
				),
				fmt.Sprintf(
					"You prepare the %s spell.",
					spell.Name,
				),
				fmt.Sprintf(
					"[Round: %d sec]", castTime,
				),
			},
			RoomBroadcast: []string{
				fmt.Sprintf("%s prepares a spell.", e.playerDisplayName(player)),
			},
		}
	}

	// ---------------------------------------------------------
	// NORMAL SPELL
	// ---------------------------------------------------------

	player.PreparedSpell = spell.ID
	castTime := e.spellCastTime(player, spell)
	player.RoundTimeExpiry = time.Now().Add(time.Duration(castTime) * time.Second)

	return &CommandResult{
		Messages: []string{
			fmt.Sprintf("You prepare the %s spell.", spell.Name),
			fmt.Sprintf("[Round: %d sec]", castTime),
		},
		RoomBroadcast: []string{
			fmt.Sprintf("%s prepares a spell.", e.playerDisplayName(player)),
		},
	}
}

func (e *GameEngine) tryInterruptPlayerSpell(player *Player) bool {
	if player == nil {
		return false
	}

	// No spell is currently prepared.
	if player.PreparedSpell <= 0 {
		return false
	}

	// Wizard's Armor prevents spell disruption.
	// Uncomment when Wizard's Armor is implemented.
	//
	// if _, ok := player.HasStatEffect(WizardsArmorEffect); ok {
	// 	return false
	// }

	// Successful damaging hits have a 50% chance
	// to disrupt the prepared spell.
	if rand.Intn(2) != 0 {
		return false
	}

	player.PreparedSpell = 0

	return true
}

func (e *GameEngine) tryInterruptMonsterSpell(inst *MonsterInstance, def *gameworld.MonsterDef) bool {
	if inst == nil || def == nil {
		return false
	}

	// No spell is currently being prepared.
	if inst.PreparedSpell <= 0 {
		return false
	}

	// NONDISRUPTABLE monsters cannot have their spells disrupted.
	// Wire this in once NONDISRUPTABLE is implemented.
	//
	// if def.NonDisruptable {
	// 	return false
	// }

	// 50% chance to disrupt.
	if rand.Intn(2) != 0 {
		return false
	}

	// Cancel the pending spell completely.
	inst.PreparedSpell = 0
	inst.SpellReadyAt = time.Time{}

	return true
}

// doCastSpell handles CAST [target].
func (e *GameEngine) doCastSpell(ctx context.Context, player *Player, args []string) *CommandResult {
	if player.Dead {
		return &CommandResult{
			Messages: []string{"You can't cast spells while dead."},
		}
	}

	if player.Unconscious {
		return &CommandResult{
			Messages: []string{"You can't cast spells while unconscous."},
		}
	}

	if player.Wielded != nil {
		return &CommandResult{
			Messages: []string{"You'll have to unwield to do that."},
		}
	}

	if player.Offhand != nil {
		offhandDef := e.items[player.Offhand.Archetype]
		if offhandDef == nil || offhandDef.Type != "SHIELD" {
			return &CommandResult{
				Messages: []string{"You'll have to unwield to do that."},
			}
		}
	}

	// Once CAST is accepted as an action, invisibility is broken.
	// This happens even if no spell is prepared or the spell later fails.
	var revealMsg string
	if e.breakInvisibility(player) {
		revealMsg = "You suddenly fade into view."
	}

	// Helper so every early return after this point preserves
	// the invisibility reveal message.
	withReveal := func(messages []string) []string {
		if revealMsg == "" {
			return messages
		}

		return append(
			[]string{revealMsg},
			messages...,
		)
	}

	// If no spell is prepared, try to prepare+cast in one step.
	if player.PreparedSpell == 0 {
		if len(args) == 0 {
			return &CommandResult{
				Messages: withReveal([]string{
					"You have no spell prepared. Use PREPARE <spell> first.",
				}),
			}
		}

		// Try direct cast: "cast flame bolt <target>"
		spellName := strings.Join(args, " ")
		spell := FindSpellByName(spellName)

		if spell == nil {
			return &CommandResult{
				Messages: withReveal([]string{
					fmt.Sprintf(
						"You don't know a spell called '%s'.",
						spellName,
					),
				}),
			}
		}

		if !player.KnownSpells[spell.ID] && !player.IsGM {
			return &CommandResult{
				Messages: withReveal([]string{
					fmt.Sprintf(
						"You haven't learned %s.",
						spell.Name,
					),
				}),
			}
		}

		player.PreparedSpell = spell.ID
	}

	spell := FindSpellByID(player.PreparedSpell)
	if spell == nil {
		player.PreparedSpell = 0

		return &CommandResult{
			Messages: withReveal([]string{
				"Your spell fizzles.",
			}),
		}
	}

	// Mana cost = spell level (from LEGENDS.DOC)
	// Base mana cost is the spell's level.
	// Spell mastery reduces the cost by 2 per rank,
	// to a minimum of half the original cost, rounded down.
	manaCost := e.spellManaCost(player, spell)

	if player.Mana < manaCost {
		player.PreparedSpell = 0

		return &CommandResult{
			Messages: withReveal([]string{
				fmt.Sprintf(
					"Not enough mana! (%s requires %d, you have %d)",
					spell.Name,
					manaCost,
					player.Mana,
				),
			}),
		}
	}

	// Check roundtime.
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()

		return &CommandResult{
			Messages: withReveal([]string{
				fmt.Sprintf(
					"You are still preparing... %.0f seconds remaining.",
					remaining+0.5,
				),
			}),
		}
	}

	// Deduct mana and consume the prepared spell.
	player.Mana -= manaCost
	player.PreparedSpell = 0

	// Spellcraft skill check.
	spellcraftSkill := player.Skills[23]
	castChance := 25 +
		player.EffectiveStat(StatEmpathy)/10 +
		spellcraftSkill*5

	if castChance > 95 {
		castChance = 95
	}

	if player.IsGM {
		castChance = 100
	}

	castRoll := rand.Intn(100) + 1

	// Extreme failure.
	if castRoll == 100 && !player.IsGM {
		player.RoundTimeExpiry = time.Now().Add(3 * time.Second)

		return &CommandResult{
			Messages: withReveal([]string{
				fmt.Sprintf(
					"[Success: %d%%, Roll %d] Extreme failure! The spell backfires!",
					castChance,
					castRoll,
				),
			}),
			RoomBroadcast: []string{
				fmt.Sprintf(
					"Magic begins to form around %s but then fizzles.",
					e.playerDisplayName(player),
				),
			},
		}
	}

	spectacularSuccess := castRoll == 1

	// Normal failure.
	if castRoll > castChance && !player.IsGM {
		player.RoundTimeExpiry = time.Now().Add(2 * time.Second)

		return &CommandResult{
			Messages: withReveal([]string{
				fmt.Sprintf(
					"[Success: %d%%, Roll %d] Failure.",
					castChance,
					castRoll,
				),
			}),
			RoomBroadcast: []string{
				fmt.Sprintf(
					"Magic begins to form around %s but then fizzles.",
					e.playerDisplayName(player),
				),
			},
		}
	}

	// Show success roll to caster.
	successMsg := fmt.Sprintf(
		"[Success: %d%%, Roll %d] Success!",
		castChance,
		castRoll,
	)

	if spectacularSuccess {
		successMsg = fmt.Sprintf(
			"[Success: %d%%, Roll %d] Spectacular success!",
			castChance,
			castRoll,
		)
	}

	result := &CommandResult{}

	switch spell.Effect {

	case "damage":
		result = e.castDamageSpell(
			player,
			spell,
			args,
			spectacularSuccess,
		)

	case "heal":
		result = e.castHealSpell(
			player,
			spell,
			args,
			true,
		)

	case "defense":
		result = e.castStatusSpell(
			player,
			spell,
			args,
			true,
		)

	case "buff":
		result = e.castStatusSpell(
			player,
			spell,
			args,
			true,
		)

	case "summon":
		result = e.castSummonSpell(
			player,
			spell,
			args,
		)

	case "debuff":
		result = e.castDebuff(
			player,
			spell,
			args,
			true,
		)

	case "utility":
		result = e.castUtilitySpell(
			ctx,
			player,
			spell,
			args,
		)

	default:
		result.Messages = []string{
			fmt.Sprintf(
				"You gesture and cast %s.",
				spell.Name,
			),
		}

		result.RoomBroadcast = []string{
			fmt.Sprintf(
				"%s gestures and casts %s.",
				e.playerDisplayName(player),
				spell.Name,
			),
		}
	}

	// Prepend success roll.
	result.Messages = append(
		[]string{successMsg},
		result.Messages...,
	)

	// If casting broke invisibility, that message comes first.
	result.Messages = withReveal(result.Messages)

	castTime := e.spellCastTime(player, spell)
	player.RoundTimeExpiry = time.Now().Add(time.Duration(castTime) * time.Second)

	e.SavePlayer(ctx, player)

	return result
}

func (e *GameEngine) castMysticKey(player *Player, args []string) *CommandResult {

	if len(args) == 0 {
		return &CommandResult{
			Messages: []string{"Cast Mystic Key on what?"},
		}
	}

	// Allow:
	// CAST CHEST
	// CAST ON CHEST
	if strings.EqualFold(args[0], "on") {
		args = args[1:]
	}

	if len(args) == 0 {
		return &CommandResult{
			Messages: []string{"Cast Mystic Key on what?"},
		}
	}

	targetName := strings.ToLower(strings.Join(args, " "))
	targetName = strings.TrimPrefix(targetName, "my ")

	targetName, targetSkip := parseOrdinal(targetName)

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{
			Messages: []string{"You can't do that here."},
		}
	}

	var inventoryTarget *InventoryItem
	var roomTarget *gameworld.RoomItem
	var targetDef *gameworld.ItemDef

	// ------------------------------------------------------------
	// Search carried containers first.
	// ------------------------------------------------------------

	skip := targetSkip

	for i := range player.Inventory {
		ii := &player.Inventory[i]

		def := e.items[ii.Archetype]
		if def == nil {
			continue
		}

		if def.Container == "" {
			continue
		}

		noun := e.getItemNounName(def)

		matched :=
			matchesTarget(
				noun,
				targetName,
				e.getAdjName(ii.Adj1),
			) ||
				matchesTarget(
					noun,
					targetName,
					e.getAdjName(ii.Adj2),
				) ||
				matchesTarget(
					noun,
					targetName,
					e.getAdjName(ii.Adj3),
				)

		if !matched {
			continue
		}

		if skip > 0 {
			skip--
			continue
		}

		inventoryTarget = ii
		targetDef = def
		break
	}

	// ------------------------------------------------------------
	// If not carried, search room containers.
	// ------------------------------------------------------------

	if inventoryTarget == nil {
		skip = targetSkip

		for i := range room.Items {
			ri := &room.Items[i]

			if ri.IsPut {
				continue
			}

			def := e.items[ri.Archetype]
			if def == nil {
				continue
			}

			if !containsFlag(def.Flags, "LOCKABLE") {
				continue
			}

			noun := e.getItemNounName(def)

			matched :=
				matchesTarget(
					noun,
					targetName,
					e.getAdjName(ri.Adj1),
				) ||
					matchesTarget(
						noun,
						targetName,
						e.getAdjName(ri.Adj2),
					) ||
					matchesTarget(
						noun,
						targetName,
						e.getAdjName(ri.Adj3),
					)

			if !matched {
				continue
			}

			if skip > 0 {
				skip--
				continue
			}

			roomTarget = ri
			targetDef = def
			break
		}
	}

	if inventoryTarget == nil && roomTarget == nil {
		return &CommandResult{
			Messages: []string{"You don't see that here."},
		}
	}

	if targetDef == nil || !containsFlag(targetDef.Flags, "LOCKABLE") {
		return &CommandResult{
			Messages: []string{"That doesn't have a lock."},
		}
	}

	// ------------------------------------------------------------
	// Get current state and lock difficulty.
	// ------------------------------------------------------------

	state := ""
	lockDifficulty := 0

	if inventoryTarget != nil {
		state = inventoryTarget.State
		lockDifficulty = inventoryTarget.Val1
	} else {
		state = roomTarget.State
		lockDifficulty = roomTarget.Val1
	}

	if !strings.EqualFold(state, "LOCKED") {
		return &CommandResult{
			Messages: []string{"It isn't locked."},
		}
	}

	// ------------------------------------------------------------
	// Mystic Key acts as exactly 7 ranks of Lockpicking.
	//
	// Same formula as PICK:
	//
	// 30
	// + (ranks - 1) * 5
	// + PER / 10
	// - lock difficulty
	// ------------------------------------------------------------

	const mysticKeyRanks = 7

	chance := 30 +
		(mysticKeyRanks-1)*5 +
		player.Perception/10 -
		lockDifficulty

	if chance < 1 {
		chance = 1
	}

	if chance > 95 {
		chance = 95
	}

	roll := rand.Intn(100) + 1

	// ------------------------------------------------------------
	// Display name.
	// ------------------------------------------------------------

	var displayName string

	if inventoryTarget != nil {
		displayName = e.formatItemName(
			targetDef,
			inventoryTarget.Adj1,
			inventoryTarget.Adj2,
			inventoryTarget.Adj3,
		)
	} else {
		displayName = e.formatItemName(
			targetDef,
			roomTarget.Adj1,
			roomTarget.Adj2,
			roomTarget.Adj3,
		)
	}

	result := &CommandResult{}

	// ------------------------------------------------------------
	// SUCCESS
	// ------------------------------------------------------------

	if roll <= chance {
		if inventoryTarget != nil {
			inventoryTarget.State = "CLOSED"
		} else {
			roomTarget.State = "CLOSED"
		}

		// Legacy Mystic Key does NOT show another success roll.
		result.Messages = []string{
			fmt.Sprintf(
				"You hear a click from within %s.",
				displayName,
			),
		}

		result.RoomBroadcast = []string{
			fmt.Sprintf(
				"%s gestures toward %s.",
				e.playerDisplayName(player),
				displayName,
			),
		}

		return result
	}

	// ------------------------------------------------------------
	// FAILURE
	//
	// Spell itself succeeded, but the magical lockpick attempt
	// failed. Legacy behavior does not expose this hidden roll.
	// ------------------------------------------------------------

	result.Messages = []string{
		fmt.Sprintf(
			"%s shudders for a moment.",
			capArticle(displayName),
		),
	}

	result.RoomBroadcast = []string{
		fmt.Sprintf(
			"%s gestures toward %s.",
			e.playerDisplayName(player),
			displayName,
		),
	}

	return result
}

func (e *GameEngine) castSummonSpell(player *Player, spell *SpellDef, args []string) *CommandResult {

	if e.playerHasSummon(player) {
		return &CommandResult{
			Messages: []string{
				"You already have a summoned creature under your control.",
			},
		}
	}

	switch spell.ID {
	case 122, 504: // Summon Familiar and druid call animal
		familiarPool := []int{58, 9, 10, 11, 8, 7}
		monsterNum := familiarPool[rand.Intn(len(familiarPool))]
		return e.summonCreature(player, monsterNum)
	case 517: // Stick to snake  -- summon viper
		return e.summonCreature(player, 15)
	case 106: // Summon Fire Elemental
		return e.summonCreature(player, 5)
	case 107: // Summon Air Elemental
		return e.summonCreature(player, 3)

	case 108: // Summon Water Elemental
		return e.summonCreature(player, 6)

	case 123: // Summon Earth Elemental
		return e.summonCreature(player, 4)

	case 109: // Summon Gargoyle
		return e.summonCreature(player, 0)

	case 355: // Summon Greater Wight
		return e.summonCreature(player, 53)

	default:
		return &CommandResult{
			Messages: []string{"Nothing answers your summons."},
		}
	}
}

func (e *GameEngine) castDebuff(player *Player, spell *SpellDef, args []string, showCastMessage bool) *CommandResult {

	if len(args) == 0 {
		return &CommandResult{
			Messages: []string{"Cast at what?"},
		}
	}

	targetName := strings.Join(args, " ")

	// ---------------------------------------------------------
	// MONSTER TARGET
	// ---------------------------------------------------------

	inst, def := e.findMonsterInRoom(player, targetName)

	if inst != nil {

		name := FormatMonsterName(def, e.monAdjs)
		article := articleFor(name, def.Unique)
		displayName := article + name

		duration := spell.Duration
		duration = e.spellDuration(player, spell, duration)

		// Determine which monster immunity applies to this status.
		switch spell.StatusType {

		case RestrainedEffect:
			duration = durationByImmunity(duration, monsterImmunity(def, gameworld.ImmunityEnsnare))

		case FearEffect:
			duration = durationByImmunity(duration, monsterImmunity(def, gameworld.ImmunityNervousSystem))
		}

		// Immune to the effect.
		if duration <= 0 {
			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"%s%s is unaffected.",
						capArticle(article),
						name,
					),
				},
			}
		}

		// Apply the status.
		e.applyMonsterStatEffect(
			inst.ID,
			spell.ID,
			EffectSourceSpell,
			spell.StatusType,
			spell.DefBonus,
			duration,
			spell.Ticks,
		)

		// Build the spell-specific message from the definition.
		statusMsg := spell.StatusMsg

		if strings.Contains(statusMsg, "%s") {
			statusMsg = fmt.Sprintf(statusMsg, displayName)
		}

		result := &CommandResult{}

		if showCastMessage {
			result.Messages = append(
				result.Messages,
				fmt.Sprintf(
					"You gesture at %s and cast %s.",
					displayName,
					spell.Name,
				),
			)

			result.RoomBroadcast = append(
				result.RoomBroadcast,
				fmt.Sprintf(
					"%s gestures at %s and casts %s.",
					e.playerDisplayName(player),
					displayName,
					spell.Name,
				),
			)
		}

		if statusMsg != "" {
			result.Messages = append(
				result.Messages,
				statusMsg,
			)

			result.RoomBroadcast = append(
				result.RoomBroadcast,
				statusMsg,
			)
		}

		// Fear causes an immediate flee.
		if spell.StatusType == FearEffect {

			e.monsterFlee(
				inst,
				def,
			)

			if player.CombatTarget != nil &&
				player.CombatTarget.IsMonster &&
				player.CombatTarget.MonsterID == inst.ID {

				player.CombatTarget = nil
				player.Joined = false
			}
		}

		return result
	}

	// ---------------------------------------------------------
	// PLAYER TARGET
	// ---------------------------------------------------------

	//var target *Player

	target := e.findPlayerInRoom(player, targetName)

	if target != nil {

		target.ApplyStatEffect(
			spell.ID,
			EffectSourceSpell,
			spell.StatusType,
			spell.DefBonus,
			e.spellDuration(player, spell, spell.Duration),
			spell.Ticks,
		)

		statusMsg := spell.StatusMsg

		if strings.Contains(statusMsg, "%s") {
			statusMsg = fmt.Sprintf(
				statusMsg,
				target.FirstName,
			)
		}

		result := &CommandResult{}

		if showCastMessage {
			result.Messages = append(
				result.Messages,
				fmt.Sprintf(
					"You gesture at %s and cast %s.",
					e.playerDisplayName(target),
					spell.Name,
				),
			)

			result.RoomBroadcast = append(
				result.RoomBroadcast,
				fmt.Sprintf(
					"%s gestures at %s and casts %s.",
					e.playerDisplayName(player),
					e.playerDisplayName(target),
					spell.Name,
				),
			)
		}

		if statusMsg != "" {
			result.Messages = append(
				result.Messages,
				statusMsg,
			)

			result.RoomBroadcast = append(
				result.RoomBroadcast,
				statusMsg,
			)
		}

		return result
	}

	return &CommandResult{
		Messages: []string{"You don't see that here."},
	}
}

func (e *GameEngine) spellDuration(player *Player, spell *SpellDef, baseDuration time.Duration) time.Duration {
	if player == nil || spell == nil || baseDuration <= 0 {
		return baseDuration
	}

	rank := e.effectiveSpellMastery(player, spell)
	if rank <= 0 {
		return baseDuration
	}

	// Each rank of mastery increases duration by 10% of
	// the spell's base duration.
	bonus := (baseDuration * time.Duration(rank)) / 10

	return baseDuration + bonus
}

func monsterImmunity(def *gameworld.MonsterDef, immunityType int) int {
	if level, ok := def.Immunities[immunityType]; ok {
		return level
	}

	return 2
}

func (e *GameEngine) summonCreature(player *Player, monsterNum int) *CommandResult {

	def := e.monsters[monsterNum]
	if def == nil || e.monsterMgr == nil {
		return &CommandResult{
			Messages: []string{"Nothing answers your summons."},
		}
	}

	inst := e.monsterMgr.SpawnOne(
		monsterNum,
		player.RoomNumber,
		def.Body+def.ExtraBody,
		def.Mana,
	)

	if inst == nil {
		return &CommandResult{
			Messages: []string{"Your summoning fails."},
		}
	}

	inst.CommanderID = player.FirstName
	inst.ControlType = "SUMMONED"

	displayName := FormatMonsterName(def, e.monAdjs)

	return &CommandResult{
		Messages: []string{
			fmt.Sprintf("A %s appears before you.", capArticle(displayName)),
		},
		RoomBroadcast: []string{
			fmt.Sprintf(
				"%s appears before %s.",
				capArticle(displayName),
				e.playerDisplayName(player),
			),
		},
	}
}

func (e *GameEngine) playerHasSummon(player *Player) bool {
	if e.monsterMgr == nil {
		return false
	}

	e.monsterMgr.mu.Lock()
	defer e.monsterMgr.mu.Unlock()

	for i := range e.monsterMgr.instances {
		inst := &e.monsterMgr.instances[i]

		if inst.Alive &&
			inst.CommanderID == player.FirstName {
			return true
		}
	}

	return false
}

func (e *GameEngine) castStatusSpell(player *Player, spell *SpellDef, args []string, showCastMessage bool) *CommandResult {

	// Default target is the caster.
	target := player

	// If arguments were supplied, resolve another player in the room.
	if len(args) > 0 {
		targetArgs := args

		// Support: CAST ON BOB
		if strings.EqualFold(targetArgs[0], "ON") {
			targetArgs = targetArgs[1:]
		}

		if len(targetArgs) > 0 {
			targetName := strings.Join(targetArgs, " ")

			found := e.findPlayerInRoom(player, targetName)
			if found == nil {
				return &CommandResult{
					Messages: []string{
						fmt.Sprintf("You don't see '%s' here.", targetName),
					},
				}
			}

			target = found
		}
	}

	duration := spell.Duration
	if duration == 0 {
		duration = 30 * time.Minute
	}

	duration = e.spellDuration(player, spell, duration)

	// Check existing effects on the TARGET, not the caster.
	if !prepareSpellFamilyEffect(target, spell) {
		if target == player {
			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"A stronger %s spell is already affecting you.",
						spell.Family,
					),
				},
			}
		}

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"A stronger %s spell is already affecting %s.",
					spell.Family,
					e.playerDisplayName(target),
				),
			},
		}
	}

	// Apply the actual status effect to the target.
	target.ApplyStatEffect(
		spell.ID,
		EffectSourceSpell,
		spell.StatusType,
		spell.DefBonus,
		duration,
		spell.Ticks,
	)

	// Apply any state associated with the status effect.
	switch spell.StatusType {
	case InvisibilityEffect:
		target.Invisible = true
	}

	// Build the spell's status message.
	statusMsg := spell.StatusMsg

	if strings.Contains(statusMsg, "%d") {
		statusMsg = fmt.Sprintf(statusMsg, spell.DefBonus)
	}

	if strings.Contains(statusMsg, "%s") {
		statusMsg = fmt.Sprintf(statusMsg, e.playerDisplayName(target))
	}

	result := &CommandResult{}

	// Normal player casting.
	if showCastMessage {
		if target == player {
			result.Messages = append(
				result.Messages,
				fmt.Sprintf("You gesture and cast %s.", spell.Name),
			)

			result.RoomBroadcast = append(
				result.RoomBroadcast,
				fmt.Sprintf(
					"%s gestures and casts %s.",
					e.playerDisplayName(player),
					spell.Name,
				),
			)
		} else {
			result.Messages = append(
				result.Messages,
				fmt.Sprintf(
					"You gesture at %s and cast %s.",
					e.playerDisplayName(target),
					spell.Name,
				),
			)

			result.RoomBroadcast = append(
				result.RoomBroadcast,
				fmt.Sprintf(
					"%s gestures at %s and casts %s.",
					e.playerDisplayName(player),
					e.playerDisplayName(target),
					spell.Name,
				),
			)
		}
	}

	// Effect message is useful whether cast normally or triggered by a script/item.
	if statusMsg != "" {
		result.Messages = append(
			result.Messages,
			statusMsg,
		)
	}

	return result
}

func (e *GameEngine) castDamageSpell(player *Player, spell *SpellDef, args []string, spectacular bool) *CommandResult {

	// ------------------------------------------------------------
	// FIND TARGET
	// ------------------------------------------------------------

	targetName := ""

	if len(args) > 0 {
		targetName = strings.Join(args, " ")
	} else if player.CombatTarget != nil && player.CombatTarget.IsMonster {
		// Auto-target current combat target.
		e.monsterMgr.mu.RLock()

		for _, inst := range e.monsterMgr.instances {
			if inst.ID == player.CombatTarget.MonsterID && inst.Alive {

				// An invisible monster cannot be auto-targeted.
				if inst.HasStatEffect(InvisibilityEffect) {
					break
				}

				def := e.monsters[inst.DefNumber]

				if def != nil {
					targetName = def.Name
				}

				break
			}
		}

		e.monsterMgr.mu.RUnlock()
	}

	if targetName == "" {
		return &CommandResult{
			Messages: []string{
				"Cast at what? Specify a target.",
			},
		}
	}

	inst, def := e.findMonsterInRoom(player, targetName)

	if inst == nil {
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You don't see '%s' here.",
					targetName,
				),
			},
		}
	}

	name := FormatMonsterName(def, e.monAdjs)

	// ------------------------------------------------------------
	// ROLL DAMAGE
	// ------------------------------------------------------------

	var dmg int

	if spell.ID == 220 || spell.ID == 221 {
		dmg = dancingWeaponDamage(player, spell)
	} else {
		dmg = rand.Intn(spell.DmgMax-spell.DmgMin+1) + spell.DmgMin
	}

	//apply mastery bonus
	dmg = e.applySpellMasteryDamage(player, spell, dmg)

	// ------------------------------------------------------------
	// MAGIC RESISTANCE
	// ------------------------------------------------------------

	if def.MagicResist > 0 {
		resistRoll := rand.Intn(100)

		if resistRoll < def.MagicResist {
			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"You gesture and cast %s at a %s, but it resists the spell!",
						spell.Name,
						name,
					),
				},
				RoomBroadcast: []string{
					fmt.Sprintf(
						"%s casts %s at a %s, but it resists!",
						e.playerDisplayName(player),
						spell.Name,
						name,
					),
				},
			}
		}
	}

	// ------------------------------------------------------------
	// DAMAGE-TYPE IMMUNITY
	// ------------------------------------------------------------

	if spell.DmgType == "undead" {
		// IMMUNITY 8 is special.
		//
		// 0 = not undead-affectable
		// 1 = extremely powerful undead
		// 2 = greater undead
		// 3 = lesser undead
		undeadPower := def.Immunities[gameworld.ImmunityUndead]

		dmg = applyImmunity(dmg, undeadPower)

	} else if spell.DmgType == "slashing" {

		if level, ok := def.Immunities[gameworld.ImmunityEdged]; ok {
			dmg = applyImmunity(dmg, level)
		}

	} else if spell.DmgType != "" {

		immType := elementalImmunityType(spell.DmgType)

		if level, ok := def.Immunities[immType]; ok {
			dmg = applyImmunity(dmg, level)
		}
	}

	if dmg <= 0 {
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You cast %s at a %s, but it seems unaffected!",
					spell.Name,
					name,
				),
			},
			RoomBroadcast: []string{
				fmt.Sprintf(
					"%s casts %s at a %s!",
					e.playerDisplayName(player),
					spell.Name,
					name,
				),
			},
		}
	}

	// ------------------------------------------------------------
	// SPECTACULAR SUCCESS
	// ------------------------------------------------------------

	if spectacular {
		dmg *= 2
	}

	// ------------------------------------------------------------
	// SPELL FLAVOR
	// ------------------------------------------------------------

	article := "a "

	// Default spell flavor.
	flavorSelf := fmt.Sprintf(
		"%s forms a bolt of energy and hurls it at %s%s!",
		e.playerDisplayName(player),
		article,
		name,
	)

	flavorDmg := fmt.Sprintf(
		"%s %s to %s. [%d Damage]",
		damageSeverity(dmg),
		spellDmgNoun(spell.DmgType),
		randomBodyPart(def.BodyType),
		dmg,
	)

	// Damage-type-specific flavor.
	switch spell.DmgType {

	case "slashing":
		flavorDmg = fmt.Sprintf(
			"%s slash to %s. [%d Damage]",
			damageSeverity(dmg),
			randomBodyPart(def.BodyType),
			dmg,
		)

	case "heat":
		flavorSelf = fmt.Sprintf(
			"%s forms a ball of flame and hurls it at %s%s!",
			e.playerDisplayName(player),
			article,
			name,
		)

		flavorDmg = fmt.Sprintf(
			"%s burn to %s. [%d Damage]",
			damageSeverity(dmg),
			randomBodyPart(def.BodyType),
			dmg,
		)

	case "cold":
		flavorSelf = fmt.Sprintf(
			"%s forms a freezing sphere from the air and hurls it at %s%s!",
			e.playerDisplayName(player),
			article,
			name,
		)

		flavorDmg = fmt.Sprintf(
			"%s blast to %s. [%d Damage]",
			damageSeverity(dmg),
			randomBodyPart(def.BodyType),
			dmg,
		)

	case "electric":
		flavorSelf = fmt.Sprintf(
			"%s releases a bolt of lightning at %s%s!",
			e.playerDisplayName(player),
			article,
			name,
		)

		flavorDmg = fmt.Sprintf(
			"%s shock to %s. [%d Damage]",
			damageSeverity(dmg),
			randomBodyPart(def.BodyType),
			dmg,
		)

	case "crushing":
		flavorSelf = fmt.Sprintf(
			"%s hurls a force blast at %s%s!",
			e.playerDisplayName(player),
			article,
			name,
		)

		flavorDmg = fmt.Sprintf(
			"%s strike to %s. [%d Damage]",
			damageSeverity(dmg),
			randomBodyPart(def.BodyType),
			dmg,
		)

	case "undead":
		flavorSelf = fmt.Sprintf(
			"%s directs a surge of necromantic power at %s%s!",
			e.playerDisplayName(player),
			article,
			name,
		)

		flavorDmg = fmt.Sprintf(
			"%s blast to %s. [%d Damage]",
			damageSeverity(dmg),
			randomBodyPart(def.BodyType),
			dmg,
		)
	}

	if spell.StatusMsg != "" {
		flavorSelf = fmt.Sprintf(
			spell.StatusMsg,
			e.playerDisplayName(player),
			article+name,
		)
	}

	// ------------------------------------------------------------
	// APPLY DAMAGE
	// ------------------------------------------------------------

	damageResult := e.damageMonster(player, inst.ID, dmg, false)
	killed := damageResult.Killed

	// ------------------------------------------------------------
	// TURN UNDEAD
	//
	// Turn Undead is a damage spell with an additional chance
	// to inflict Fear on a surviving undead monster.
	// ------------------------------------------------------------

	turned := false

	isTurnUndead :=
		spell.ID == 301 ||
			spell.ID == 302 ||
			spell.ID == 304

	if !killed && isTurnUndead {
		undeadPower := def.Immunities[gameworld.ImmunityUndead]

		if canTurnUndead(spell.ID, undeadPower) {
			spellcraft := player.Skills[23]

			turnChance := turnUndeadChance(
				spellcraft,
				undeadPower,
			)

			if rand.Intn(100)+1 <= turnChance {
				turned = true

				e.applyMonsterStatEffect(
					inst.ID,
					spell.ID,
					EffectSourceSpell,
					FearEffect,
					0,
					spell.Duration,
					spell.Ticks,
				)
			}
		}
	}

	// ------------------------------------------------------------
	// BUILD MESSAGES
	// ------------------------------------------------------------

	var msgs, roomMsgs []string

	msgs = append(
		msgs,
		fmt.Sprintf(
			"You gesture at %s%s.",
			article,
			name,
		),
	)

	roomMsgs = append(
		roomMsgs,
		fmt.Sprintf(
			"%s gestures at %s%s.",
			e.playerDisplayName(player),
			article,
			name,
		),
	)

	msgs = append(
		msgs,
		flavorSelf,
	)

	roomMsgs = append(
		roomMsgs,
		flavorSelf,
	)

	msgs = append(
		msgs,
		flavorDmg,
	)

	// ------------------------------------------------------------
	// SUCCESSFUL TURN
	// ------------------------------------------------------------

	if turned {
		msgs = append(
			msgs,
			fmt.Sprintf(
				"The %s recoils and flees!",
				name,
			),
		)

		roomMsgs = append(
			roomMsgs,
			fmt.Sprintf(
				"The %s recoils and flees!",
				name,
			),
		)

		// Immediate flee. FearEffect will cause continued fleeing
		// on subsequent monster action ticks.
		e.monsterFlee(
			inst,
			def,
		)

		// Turning also disengages the caster from this monster.
		if player.CombatTarget != nil &&
			player.CombatTarget.IsMonster &&
			player.CombatTarget.MonsterID == inst.ID {

			player.CombatTarget = nil
			player.Joined = false
		}

	} else if !killed && isTurnUndead {
		msgs = append(
			msgs,
			fmt.Sprintf(
				"The %s resists being turned!",
				name,
			),
		)
	}

	if damageResult.BrokeCharm {
		msgs = append(
			msgs,
			fmt.Sprintf("The pain snaps the %s out of the charm.", name),
		)
	}

	if damageResult.DisruptedSpell {
		msgs = append(
			msgs,
			fmt.Sprintf("%s%s's spell is disrupted!", article, name),
		)
	}

	// ------------------------------------------------------------
	// MONSTER DEATH
	// ------------------------------------------------------------

	if killed {
		deathText := def.TextOverrides["TEXD"]

		if deathText != "" {
			msgs = append(
				msgs,
				fmt.Sprintf(
					"A %s %s",
					name,
					deathText,
				),
			)

			roomMsgs = append(
				roomMsgs,
				fmt.Sprintf(
					"A %s %s",
					name,
					deathText,
				),
			)

		} else {
			msgs = append(
				msgs,
				"He collapses, dead.",
			)

			roomMsgs = append(
				roomMsgs,
				fmt.Sprintf(
					"A %s collapses, dead!",
					name,
				),
			)
		}

		e.handleMonsterDeath(
			player,
			inst,
			def,
		)

		player.CombatTarget = nil
		player.Joined = false
	}

	return &CommandResult{
		Messages:      msgs,
		RoomBroadcast: roomMsgs,
	}
}

func durationByImmunity(base time.Duration, immunity int) time.Duration {
	switch immunity {
	case 0:
		return 0
	case 1:
		return base / 2
	case 2:
		return base
	case 3:
		return base * 3 / 2
	case 4:
		return base * 2
	default:
		return base
	}
}

func (e *GameEngine) castHealSpell(player *Player, spell *SpellDef, args []string, showCastMessage bool) *CommandResult {
	// Heal self by default, or target if specified.
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

	switch spell.StatusType {

	case RemovePoison:
		target.RemoveStatEffectsBySource(EffectSourcePoison)
		target.Poisoned = false

		if showCastMessage {
			if target == player {
				return &CommandResult{
					Messages: []string{
						fmt.Sprintf(
							"You gesture and cast %s on yourself. The poison leaves your system.",
							spell.Name,
						),
					},
					RoomBroadcast: []string{
						fmt.Sprintf(
							"%s gestures and casts %s.",
							e.playerDisplayName(player),
							spell.Name,
						),
					},
				}
			}

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"You gesture and cast %s on %s. The poison leaves their system.",
						spell.Name,
						targetName,
					),
				},
				RoomBroadcast: []string{
					fmt.Sprintf(
						"%s gestures and casts %s on %s.",
						e.playerDisplayName(player),
						spell.Name,
						e.playerDisplayName(target),
					),
				},
				TargetName: e.playerDisplayName(target),
				TargetMsg: []string{
					fmt.Sprintf(
						"%s casts %s on you. The poison leaves your system.",
						e.playerDisplayName(player),
						spell.Name,
					),
				},
			}
		}

	case RemoveDisease:
		target.RemoveStatEffectsBySource(EffectSourceDisease)
		target.Diseased = false

		if showCastMessage {
			if target == player {
				return &CommandResult{
					Messages: []string{
						fmt.Sprintf(
							"You gesture and cast %s on yourself. The disease leaves your system.",
							spell.Name,
						),
					},
					RoomBroadcast: []string{
						fmt.Sprintf(
							"%s gestures and casts %s.",
							e.playerDisplayName(player),
							spell.Name,
						),
					},
				}
			}

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"You gesture and cast %s on %s. The disease leaves their system.",
						spell.Name,
						targetName,
					),
				},
				RoomBroadcast: []string{
					fmt.Sprintf(
						"%s gestures and casts %s on %s.",
						e.playerDisplayName(player),
						spell.Name,
						e.playerDisplayName(target),
					),
				},
				TargetName: e.playerDisplayName(target),
				TargetMsg: []string{
					fmt.Sprintf(
						"%s casts %s on you. The disease leaves your system.",
						e.playerDisplayName(player),
						spell.Name,
					),
				},
			}
		}

	case StatFatigue:
		heal := rand.Intn(spell.HealMax-spell.HealMin+1) + spell.HealMin

		target.Fatigue += heal
		if target.Fatigue > target.MaxFatigue {
			target.Fatigue = target.MaxFatigue
		}

		if showCastMessage {
			if target == player {
				return &CommandResult{
					Messages: []string{
						fmt.Sprintf(
							"You gesture and cast %s on yourself. You feel invigorated. [Fatigue: %d/%d]",
							spell.Name,
							target.Fatigue,
							target.MaxFatigue,
						),
					},
					RoomBroadcast: []string{
						fmt.Sprintf(
							"%s gestures and casts %s. %s looks invigorated.",
							e.playerDisplayName(player),
							spell.Name,
							e.playerDisplayName(player),
						),
					},
				}
			}

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"You gesture and cast %s on %s. %s looks invigorated.",
						spell.Name,
						targetName,
						targetName,
					),
				},
				RoomBroadcast: []string{
					fmt.Sprintf(
						"%s gestures at %s. %s looks invigorated.",
						e.playerDisplayName(player),
						e.playerDisplayName(target),
						e.playerDisplayName(target),
					),
				},
				TargetName: e.playerDisplayName(target),
				TargetMsg: []string{
					fmt.Sprintf(
						"%s gestures at you. You feel invigorated. [Fatigue: %d/%d]",
						e.playerDisplayName(player),
						target.Fatigue,
						target.MaxFatigue,
					),
				},
			}
		}

		// Non-cast application.
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You feel invigorated. [Fatigue: %d/%d]",
					target.Fatigue,
					target.MaxFatigue,
				),
			},
		}

	case StatBodyPoint:
		heal := rand.Intn(spell.HealMax-spell.HealMin+1) + spell.HealMin

		target.BodyPoints += heal
		if target.BodyPoints > target.MaxBodyPoints {
			target.BodyPoints = target.MaxBodyPoints
		}

		if showCastMessage {
			if target == player {
				return &CommandResult{
					Messages: []string{
						fmt.Sprintf(
							"You gesture and cast %s on yourself, healing %d body points. [BP: %d/%d]",
							spell.Name,
							heal,
							target.BodyPoints,
							target.MaxBodyPoints,
						),
					},
					RoomBroadcast: []string{
						fmt.Sprintf(
							"%s gestures and casts %s.",
							e.playerDisplayName(player),
							spell.Name,
						),
					},
				}
			}

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"You gesture and cast %s on %s, healing %d body points.",
						spell.Name,
						targetName,
						heal,
					),
				},
				RoomBroadcast: []string{
					fmt.Sprintf(
						"%s gestures and casts %s on %s.",
						e.playerDisplayName(player),
						spell.Name,
						e.playerDisplayName(target),
					),
				},
				TargetName: e.playerDisplayName(target),
				TargetMsg: []string{
					fmt.Sprintf(
						"%s casts %s on you, healing %d body points. [BP: %d/%d]",
						e.playerDisplayName(player),
						spell.Name,
						heal,
						target.BodyPoints,
						target.MaxBodyPoints,
					),
				},
			}
		}

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You feel a little better. [BP: %d/%d]",
					target.BodyPoints,
					target.MaxBodyPoints,
				),
			},
		}

	default:
		// Preserve the old behavior for any heal spell that hasn't
		// explicitly been assigned a healing resource yet.
		heal := rand.Intn(spell.HealMax-spell.HealMin+1) + spell.HealMin

		target.BodyPoints += heal
		if target.BodyPoints > target.MaxBodyPoints {
			target.BodyPoints = target.MaxBodyPoints
		}

		if showCastMessage {
			if target == player {
				return &CommandResult{
					Messages: []string{
						fmt.Sprintf(
							"You gesture and cast %s on yourself, healing %d body points. [BP: %d/%d]",
							spell.Name,
							heal,
							target.BodyPoints,
							target.MaxBodyPoints,
						),
					},
					RoomBroadcast: []string{
						fmt.Sprintf(
							"%s gestures and casts %s.",
							e.playerDisplayName(player),
							spell.Name,
						),
					},
				}
			}

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"You gesture and cast %s on %s, healing %d body points.",
						spell.Name,
						targetName,
						heal,
					),
				},
				RoomBroadcast: []string{
					fmt.Sprintf(
						"%s gestures and casts %s on %s.",
						e.playerDisplayName(player),
						spell.Name,
						e.playerDisplayName(target),
					),
				},
				TargetName: e.playerDisplayName(target),
				TargetMsg: []string{
					fmt.Sprintf(
						"%s casts %s on you, healing %d body points. [BP: %d/%d]",
						e.playerDisplayName(player),
						spell.Name,
						heal,
						target.BodyPoints,
						target.MaxBodyPoints,
					),
				},
			}
		}

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You feel a little better. [BP: %d/%d]",
					target.BodyPoints,
					target.MaxBodyPoints,
				),
			},
		}
	}

	return &CommandResult{}
}

func (e *GameEngine) castBuffSpell(player *Player, spell *SpellDef, args []string, showCastMessage bool) *CommandResult {

	msg := fmt.Sprintf("You gesture and cast %s.", spell.Name)

	buffDuration := spell.Duration
	if buffDuration == 0 {
		buffDuration = 30 * time.Minute
	}

	// Check for existing spell family effect
	if !prepareSpellFamilyEffect(player, spell) {
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You gesture and cast %s, but a stronger %s spell is already affecting you.",
					spell.Name,
					spell.Family,
				),
			},
			RoomBroadcast: []string{
				fmt.Sprintf("%s gestures and casts %s.", e.playerDisplayName(player), spell.Name),
			},
		}
	}

	// Generic stat/status buff
	if spell.StatusType != 0 {

		player.ApplyStatEffect(
			spell.ID,
			EffectSourceSpell,
			spell.StatusType,
			spell.DefBonus,
			buffDuration,
			spell.Ticks,
		)

		result := &CommandResult{}

		if showCastMessage {
			result.Messages = append(
				result.Messages,
				fmt.Sprintf("You gesture and cast %s.", spell.Name),
			)

			result.RoomBroadcast = append(
				result.RoomBroadcast,
				fmt.Sprintf("%s gestures and casts %s.", e.playerDisplayName(player), spell.Name),
			)
		}

		if spell.StatusMsg != "" {
			result.Messages = append(result.Messages, spell.StatusMsg)
		}

		return result
	}

	switch spell.ID {
	case 202: // Enchantment I — enchant a weapon in inventory
		if len(args) == 0 {
			return &CommandResult{Messages: []string{"Enchant what? Specify a weapon in your inventory."}}
		}
		target := strings.ToLower(strings.Join(args, " "))
		for i, ii := range player.Inventory {
			def := e.items[ii.Archetype]
			if def == nil || !isWeapon(def.Type) {
				continue
			}
			name := strings.ToLower(e.getItemNounName(def))
			if !strings.HasPrefix(name, target) && !strings.Contains(name, target) {
				continue
			}
			// Check if already enchanted (Val1 > 0 means has magical edge bonus)
			if ii.Val1 > 0 {
				return &CommandResult{Messages: []string{"That weapon already has magical properties."}}
			}
			// Apply enchantment: +10 to hit via Val1
			player.Inventory[i].Val1 = 10
			itemName := e.formatItemName(def, ii.Adj1, ii.Adj2, ii.Adj3)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("A soft glow surrounds %s and then sinks into it.", itemName)},
				RoomBroadcast: []string{fmt.Sprintf("A soft glow surrounds an item %s is holding.", e.playerDisplayName(player))},
			}
		}
		return &CommandResult{Messages: []string{"You don't have a weapon matching that."}}

	}

	return &CommandResult{
		Messages:      []string{msg},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", e.playerDisplayName(player), spell.Name)},
	}
}

func (e *GameEngine) castDefenseSpell(player *Player, spell *SpellDef, args []string) *CommandResult {

	buffDuration := spell.Duration
	if buffDuration == 0 {
		buffDuration = 30 * time.Minute
	}

	if !prepareSpellFamilyEffect(player, spell) {
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You gesture and cast %s, but a stronger %s spell is already affecting you.",
					spell.Name,
					spell.Family,
				),
			},
			RoomBroadcast: []string{
				fmt.Sprintf("%s gestures and casts %s.", e.playerDisplayName(player), spell.Name),
			},
		}
	}

	player.ApplyStatEffect(
		spell.ID,
		EffectSourceSpell,
		DefensiveBuff,
		spell.DefBonus,
		buffDuration,
		spell.Ticks,
	)

	return &CommandResult{
		Messages: []string{
			fmt.Sprintf(
				"You gesture and cast %s. A protective force surrounds you! (+%d defense)",
				spell.Name,
				spell.DefBonus,
			),
		},
		RoomBroadcast: []string{
			fmt.Sprintf("%s gestures and casts %s.", e.playerDisplayName(player), spell.Name),
		},
	}
}

func turnUndeadChance(spellcraft int, undeadPower int) int {
	chance := 40 + spellcraft*5

	switch undeadPower {
	case 3: // lesser undead
		// no penalty

	case 2: // greater undead
		chance -= 25

	case 1: // extremely powerful undead
		chance -= 50
	}

	if chance < 5 {
		chance = 5
	}
	if chance > 95 {
		chance = 95
	}

	return chance
}
func canTurnUndead(spellID int, undeadPower int) bool {
	switch spellID {
	case 301: // Turn Undead I - lesser undead
		return undeadPower == 3

	case 302: // Turn Undead II - lesser and greater undead
		return undeadPower >= 2

	case 304: // Turn Undead III -
		return undeadPower > 1

	default:
		return false
	}
}

func addItemAdjective(item *InventoryItem, adj int) bool {
	if item == nil || adj <= 0 {
		return false
	}

	// Don't add the same adjective twice.
	if item.Adj1 == adj || item.Adj2 == adj || item.Adj3 == adj {
		return false
	}

	if item.Adj1 == 0 {
		item.Adj1 = adj
		return true
	}

	if item.Adj2 == 0 {
		item.Adj2 = adj
		return true
	}

	if item.Adj3 == 0 {
		item.Adj3 = adj
		return true
	}

	// All adjective slots are already occupied.
	return false
}

func (e *GameEngine) traitAdjective(traitName string) int {
	trait := e.traits[strings.ToUpper(traitName)]
	if trait == nil || trait.Power <= 0 {
		return 0
	}

	for _, block := range trait.Scripts {
		if strings.ToUpper(block.Type) != "ADJDEF" || len(block.Args) == 0 {
			continue
		}

		adj, err := strconv.Atoi(block.Args[0])
		if err != nil {
			continue
		}

		return adj
	}

	return 0
}

func (e *GameEngine) castUtilitySpell(ctx context.Context, player *Player, spell *SpellDef, args []string) *CommandResult {

	result := &CommandResult{}

	switch spell.ID {

	case 114: // Mystic Key
		result = e.castMysticKey(player, args)

	case 407: // Analyze Ore
		if len(args) == 0 {
			return &CommandResult{
				Messages: []string{"Analyze what?"},
			}
		}

		return e.doAnalyze(ctx, player, args, true)
	case 405: // See Hidden
		result.Messages = []string{
			"You narrow your eyes and gaze about the area.",
		}

		// ------------------------------------------------------------
		// PLAYERS
		//
		// Hidden players are identified.
		// Invisible players are detected only as "something".
		// Nothing is actually revealed or removed.
		// ------------------------------------------------------------

		if e.sessions != nil {
			for _, target := range e.sessions.OnlinePlayers() {
				if target == nil {
					continue
				}

				// Don't detect yourself.
				if target == player {
					continue
				}

				if target.RoomNumber != player.RoomNumber {
					continue
				}

				// Invisibility takes precedence over hiding.
				if _, ok := player.HasStatEffect(InvisibilityEffect); ok {
					result.Messages = append(result.Messages, "You see something.")
					continue
				}

				if target.Hidden {
					result.Messages = append(
						result.Messages,
						fmt.Sprintf(
							"You see %s.",
							e.playerDisplayName(target),
						),
					)
				}
			}
		}

		e.monsterMgr.mu.RLock()

		for _, inst := range e.monsterMgr.instances {
			if !inst.Alive {
				continue
			}

			if inst.RoomNumber != player.RoomNumber {
				continue
			}

			if inst.HasStatEffect(InvisibilityEffect) {
				result.Messages = append(
					result.Messages,
					"You see something.",
				)
				continue
			}

			if inst.HasStatEffect(InvisibilityEffect) {
				def := e.monsters[inst.DefNumber]
				if def == nil {
					continue
				}

				name := FormatMonsterName(def, e.monAdjs)
				article := articleFor(name, def.Unique)

				result.Messages = append(
					result.Messages,
					fmt.Sprintf(
						"You see %s%s.",
						article,
						name,
					),
				)
			}
		}

		e.monsterMgr.mu.RUnlock()

		result.RoomBroadcast = append(
			result.RoomBroadcast,
			fmt.Sprintf(
				"%s narrows their eyes and gazes carefully about the area.",
				e.playerDisplayName(player),
			),
		)
	case 401: // Dispel Lesser Magic

		// ------------------------------------------------------------
		// NO TARGET = SELF
		// ------------------------------------------------------------

		if len(args) == 0 {
			removed, ok := e.dispelRandomPlayerSpellEffect(player, spell)

			if !ok {
				result.Messages = []string{
					"Nothing happens.",
				}
				break
			}

			result.Messages = []string{
				fmt.Sprintf(
					"A deep red light twinkles around %s.",
					e.playerDisplayName(player),
				),
			}

			if removedSpell := FindSpellByID(removed.EffectID); removedSpell != nil {
				result.Messages = append(
					result.Messages,
					fmt.Sprintf(
						"The effects of %s are dispelled.",
						removedSpell.Name,
					),
				)
			}

			break
		}

		targetName := strings.Join(args, " ")

		// ------------------------------------------------------------
		// PLAYER TARGET
		// ------------------------------------------------------------

		if e.sessions != nil {
			for _, target := range e.sessions.OnlinePlayers() {
				if target == nil {
					continue
				}

				if target.RoomNumber != player.RoomNumber {
					continue
				}

				if !strings.EqualFold(target.FirstName, targetName) {
					continue
				}

				removed, ok := e.dispelRandomPlayerSpellEffect(target, spell)

				if !ok {
					result.Messages = []string{
						"Nothing happens.",
					}
					break
				}

				result.Messages = []string{
					fmt.Sprintf(
						"A deep red light twinkles around %s.",
						e.playerDisplayName(target),
					),
				}

				if removedSpell := FindSpellByID(removed.EffectID); removedSpell != nil {
					result.Messages = append(
						result.Messages,
						fmt.Sprintf(
							"The effects of %s are dispelled from %s.",
							removedSpell.Name,
							e.playerDisplayName(target),
						),
					)
				}

				break
			}

			if len(result.Messages) > 0 {
				break
			}
		}

		// ------------------------------------------------------------
		// MONSTER TARGET
		// ------------------------------------------------------------

		foundMonster := false

		if e.monsterMgr != nil {
			e.monsterMgr.mu.Lock()

			for i := range e.monsterMgr.instances {
				inst := &e.monsterMgr.instances[i]

				if !inst.Alive {
					continue
				}

				if inst.RoomNumber != player.RoomNumber {
					continue
				}

				def := e.monsters[inst.DefNumber]
				if def == nil {
					continue
				}

				name := FormatMonsterName(def, e.monAdjs)

				if !strings.EqualFold(name, targetName) &&
					!strings.EqualFold(def.Name, targetName) {
					continue
				}

				foundMonster = true

				removed, ok := e.dispelRandomMonsterSpellEffect(inst, spell)

				if !ok {
					result.Messages = []string{
						"Nothing happens.",
					}
					break
				}

				article := articleFor(name, def.Unique)

				result.Messages = []string{
					fmt.Sprintf(
						"A deep red light twinkles around %s%s.",
						article,
						name,
					),
				}

				if removedSpell := FindSpellByID(removed.EffectID); removedSpell != nil {
					result.Messages = append(
						result.Messages,
						fmt.Sprintf(
							"The effects of %s are dispelled from %s%s.",
							removedSpell.Name,
							article,
							name,
						),
					)
				}

				break
			}

			e.monsterMgr.mu.Unlock()
		}

		if foundMonster {
			break
		}

		// ------------------------------------------------------------
		// NOTHING MATCHED
		// ------------------------------------------------------------

		result.Messages = []string{
			"What are you referring to? Please be more specific.",
		}

	case 406: // Dispel Invisibility
		result.Messages = []string{
			"You gesture, sending a wave of magic through the area.",
		}

		dispelled := false

		// Players
		if e.sessions != nil {
			for _, target := range e.sessions.OnlinePlayers() {
				if target == nil || target.RoomNumber != player.RoomNumber {
					continue
				}

				if e.breakInvisibility(target) {
					result.Messages = append(
						result.Messages,
						fmt.Sprintf("%s suddenly appears!", e.playerDisplayName(target)),
					)

					dispelled = true
				}
			}
		}

		// Monsters
		e.monsterMgr.mu.Lock()

		for i := range e.monsterMgr.instances {
			inst := &e.monsterMgr.instances[i]

			if !inst.Alive || inst.RoomNumber != player.RoomNumber {
				continue
			}

			if e.breakMonsterInvisibility(inst) {
				def := e.monsters[inst.DefNumber]
				if def != nil {
					name := FormatMonsterName(def, e.monAdjs)

					result.Messages = append(
						result.Messages,
						fmt.Sprintf("%s suddenly appears!", name),
					)
				}

				dispelled = true
			}
		}

		e.monsterMgr.mu.Unlock()

		if !dispelled {
			result.Messages = append(
				result.Messages,
				"Nothing happens.",
			)
		}

	case 228: // Identify
		if len(args) > 0 && strings.EqualFold(args[0], "on") {
			args = args[1:]
		}

		result = e.doItemInteraction(
			ctx,
			player,
			"IDENTIFY",
			args,
		)

		result.RoomBroadcast = append(
			result.RoomBroadcast,
			fmt.Sprintf(
				"%s gestures thoughtfully, studying something nearby.",
				e.playerDisplayName(player),
			))

	case 400: // Detect Magic
		if len(args) > 0 && strings.EqualFold(args[0], "on") {
			args = args[1:]
		}

		result = e.doItemInteraction(
			ctx,
			player,
			"DETECTMAGIC",
			args,
		)

		// If item lookup failed, keep the lookup message.
		if len(result.Messages) > 0 {
			return result
		}

		switch {
		case result.MagicPower == 0:
			result.Messages = []string{
				"You sense no magic.",
			}

		case result.MagicPower == 1:
			result.Messages = []string{
				"You sense a faint magical aura.",
			}

		case result.MagicPower <= 3:
			result.Messages = []string{
				"You sense a noticeable magical aura.",
			}

		case result.MagicPower <= 6:
			result.Messages = []string{
				"You sense a strong magical aura.",
			}

		default:
			result.Messages = []string{
				"You sense an overwhelming magical aura.",
			}
		}
		result.RoomBroadcast = append(
			result.RoomBroadcast,
			fmt.Sprintf(
				"%s gestures thoughtfully, studying something nearby for signs of magic.",
				e.playerDisplayName(player),
			))

	default:
		result.Messages = []string{
			fmt.Sprintf(
				"You gesture and cast %s.",
				spell.Name,
			),
		}
	}

	return result
}

// spellDmgNoun returns a damage noun for the spell's damage type.
func spellDmgNoun(dmgType string) string {
	switch dmgType {
	case "heat":
		return "burn"
	case "cold":
		return "blast"
	case "electric":
		return "shock"
	case "crushing":
		return "strike"
	default:
		return "blast"
	}
}

func prepareSpellFamilyEffect(player *Player, spell *SpellDef) bool {
	if spell.Family == "" {
		return true
	}

	for i := len(player.ActiveStatEffects) - 1; i >= 0; i-- {
		effect := player.ActiveStatEffects[i]

		if effect.Source != EffectSourceSpell {
			continue
		}

		existingSpell := FindSpellByID(effect.EffectID)
		if existingSpell == nil || existingSpell.Family != spell.Family {
			continue
		}

		// Do not let a weaker spell replace a stronger one.
		if existingSpell.Level > spell.Level {
			return false
		}

		// Same spell stays in place so ApplyStatEffect refreshes it.
		if existingSpell.ID == spell.ID {
			return true
		}

		// Stronger incoming spell replaces the weaker family member.
		player.ActiveStatEffects = append(
			player.ActiveStatEffects[:i],
			player.ActiveStatEffects[i+1:]...,
		)
	}

	return true
}

func (e *GameEngine) identifyItem(player *Player, item *InventoryItem) []string {
	if player == nil || item == nil {
		return []string{"You cannot identify that."}
	}

	def := e.items[item.Archetype]
	if def == nil {
		return []string{"You cannot identify that."}
	}

	// Identification belongs to this specific item instance.
	item.Identified = true

	// Apply identification adjectives from magical traits.
	applyTrait := func(traitName string) {
		adj := e.traitAdjective(traitName)
		if adj > 0 {
			addItemAdjective(item, adj)
		}
	}

	// Archetype traits first.
	for _, traitName := range def.Traits {
		applyTrait(traitName)
	}

	// Instance traits second.
	for _, traitName := range item.Traits {
		applyTrait(traitName)
	}

	itemName := e.formatItemName(
		def,
		item.Adj1,
		item.Adj2,
		item.Adj3,
	)

	messages := []string{
		fmt.Sprintf("You identify %s.", itemName),
	}

	enchantments := e.itemEnchantments(item, def)

	if len(enchantments) == 0 {
		messages = append(messages, "You detect no magical enchantments.")
	} else {
		if len(enchantments) == 1 {
			messages = append(messages, "It bears the following enchantment:")
		} else {
			messages = append(messages, "It bears the following enchantments:")
		}

		for _, enchantment := range enchantments {
			messages = append(messages, "  "+enchantment)
		}
	}
	return messages
}

func (e *GameEngine) dispelRandomPlayerSpellEffect(target *Player, dispelSpell *SpellDef) (*StatEffect, bool) {
	if target == nil || dispelSpell == nil {
		return nil, false
	}

	var eligible []StatEffect
	now := time.Now()

	for _, effect := range target.ActiveStatEffects {
		if effect.Source != EffectSourceSpell {
			continue
		}

		if !effect.Permanent && !effect.ExpiresAt.After(now) {
			continue
		}

		effectSpell := FindSpellByID(effect.EffectID)
		if effectSpell == nil {
			continue
		}

		if effectSpell.Level > dispelSpell.Level {
			continue
		}

		eligible = append(eligible, effect)
	}

	if len(eligible) == 0 {
		return nil, false
	}

	chosen := eligible[rand.Intn(len(eligible))]

	if !target.RemoveActiveStatEffect(chosen.EffectID, chosen.Source) {
		return nil, false
	}

	return &chosen, true
}

func (e *GameEngine) dispelRandomMonsterSpellEffect(target *MonsterInstance, dispelSpell *SpellDef) (*StatEffect, bool) {
	if target == nil || dispelSpell == nil {
		return nil, false
	}

	var eligible []StatEffect
	now := time.Now()

	for _, effect := range target.ActiveStatEffects {
		if effect.Source != EffectSourceSpell {
			continue
		}

		if !effect.Permanent && !effect.ExpiresAt.After(now) {
			continue
		}

		effectSpell := FindSpellByID(effect.EffectID)
		if effectSpell == nil {
			continue
		}

		if effectSpell.Level > dispelSpell.Level {
			continue
		}

		eligible = append(eligible, effect)
	}

	if len(eligible) == 0 {
		return nil, false
	}

	chosen := eligible[rand.Intn(len(eligible))]

	if !target.RemoveActiveStatEffect(chosen.EffectID, chosen.Source) {
		return nil, false
	}

	return &chosen, true
}

func (p *Player) SpellMasteryRank(spellID int) int {
	if p == nil || p.SpellMastery == nil {
		return 0
	}

	return p.SpellMastery[spellID]
}

func maxSpellMastery(spell *SpellDef) int {
	if spell == nil {
		return 0
	}

	// Full mastery = spell level + 1.
	return spell.Level + 1
}

func minimumSpellMana(spell *SpellDef) int {
	if spell == nil {
		return 0
	}

	// Half the original mana cost, rounded down.
	// Original mana cost is the spell's level.
	return spell.Level / 2
}

func (e *GameEngine) effectiveSpellMastery(player *Player, spell *SpellDef) int {
	if player == nil || spell == nil {
		return 0
	}

	rank := player.SpellMasteryRank(spell.ID)
	if rank <= 0 {
		return 0
	}

	// Mastery benefits are suppressed while the caster
	// is affected by a time-altering spell.
	_, hasHaste := player.HasStatEffect(HasteEffect)
	_, hasSlow := player.HasStatEffect(SlowEffect)

	if hasHaste || hasSlow {
		return 0
	}

	return rank
}

func (e *GameEngine) spellManaCost(player *Player, spell *SpellDef) int {
	if spell == nil {
		return 0
	}

	base := spell.Level
	if base < 1 {
		base = 1
	}

	rank := e.effectiveSpellMastery(player, spell)

	if rank <= 0 {
		return base
	}

	cost := base - (rank * 2)

	minCost := base / 2
	if minCost < 1 {
		minCost = 1
	}

	if cost < minCost {
		cost = minCost
	}

	return cost
}

func (e *GameEngine) spellCastTime(player *Player, spell *SpellDef) int {
	if spell == nil {
		return 0
	}

	castTime := spell.CastTime

	// Historical rule specifically says mastery reduces
	// three-second casting time to two seconds.
	if e.effectiveSpellMastery(player, spell) > 0 &&
		castTime == 3 {
		return 2
	}

	return castTime
}

func (e *GameEngine) applySpellMasteryDamage(player *Player, spell *SpellDef, damage int) int {
	if player == nil || spell == nil {
		return damage
	}

	rank := e.effectiveSpellMastery(player, spell)

	if rank <= 0 {
		return damage
	}

	// Reconstruction based on the historical description of
	// approximately 1.5 points/rank. We're choosing +2/rank.
	return damage + (rank * 2)
}
