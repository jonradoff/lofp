package engine

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// SpellDef defines a spell in the game.
type SpellDef struct {
	ID       int
	Name     string
	School   string
	Level    int
	ManaCost int
	CastTime int // seconds
	Effect   string // "damage", "heal", "defense", "buff", "utility"
	DmgMin   int
	DmgMax   int
	HealMin  int
	HealMax  int
	DefBonus int
	DmgType  string // "heat", "cold", "electric", "crushing", ""
}

// spellRegistry holds all defined spells.
var spellRegistry []SpellDef

func init() {
	// Conjuration (100-144)
	conj := []SpellDef{
		{ID: 100, Name: "Flame Bolt", School: "Conjuration", Level: 1, ManaCost: 3, CastTime: 3, Effect: "damage", DmgMin: 3, DmgMax: 12, DmgType: "heat"},
		{ID: 101, Name: "Force Blade", School: "Conjuration", Level: 3, ManaCost: 5, CastTime: 3, Effect: "damage", DmgMin: 5, DmgMax: 18, DmgType: ""},
		{ID: 102, Name: "Mystic Armor", School: "Conjuration", Level: 5, ManaCost: 8, CastTime: 3, Effect: "buff", DefBonus: 20},
		{ID: 103, Name: "Lightning Bolt", School: "Conjuration", Level: 7, ManaCost: 10, CastTime: 3, Effect: "damage", DmgMin: 8, DmgMax: 30, DmgType: "electric"},
		{ID: 105, Name: "Globe of Protection", School: "Conjuration", Level: 15, ManaCost: 20, CastTime: 3, Effect: "defense", DefBonus: 50},
		{ID: 106, Name: "Summon Fire Elemental", School: "Conjuration", Level: 12, ManaCost: 25, CastTime: 5, Effect: "utility"},
		{ID: 107, Name: "Summon Air Elemental", School: "Conjuration", Level: 12, ManaCost: 25, CastTime: 5, Effect: "utility"},
		{ID: 108, Name: "Summon Water Elemental", School: "Conjuration", Level: 12, ManaCost: 25, CastTime: 5, Effect: "utility"},
		{ID: 109, Name: "Summon Gargoyle", School: "Conjuration", Level: 16, ManaCost: 30, CastTime: 5, Effect: "utility"},
		{ID: 112, Name: "Call Meteor", School: "Conjuration", Level: 20, ManaCost: 30, CastTime: 4, Effect: "damage", DmgMin: 25, DmgMax: 60, DmgType: "heat"},
		{ID: 113, Name: "Light", School: "Conjuration", Level: 1, ManaCost: 2, CastTime: 2, Effect: "utility"},
		{ID: 114, Name: "Mystic Key", School: "Conjuration", Level: 2, ManaCost: 4, CastTime: 3, Effect: "utility"},
		{ID: 115, Name: "Shockwave", School: "Conjuration", Level: 4, ManaCost: 6, CastTime: 3, Effect: "damage", DmgMin: 4, DmgMax: 15, DmgType: "crushing"},
		{ID: 116, Name: "Thunder Call", School: "Conjuration", Level: 21, ManaCost: 28, CastTime: 4, Effect: "damage", DmgMin: 20, DmgMax: 50, DmgType: "electric"},
		{ID: 117, Name: "Call Fire", School: "Conjuration", Level: 8, ManaCost: 12, CastTime: 3, Effect: "damage", DmgMin: 10, DmgMax: 25, DmgType: "heat"},
		{ID: 118, Name: "Flaming Sphere", School: "Conjuration", Level: 13, ManaCost: 18, CastTime: 3, Effect: "damage", DmgMin: 15, DmgMax: 40, DmgType: "heat"},
		{ID: 119, Name: "Ice Bolt", School: "Conjuration", Level: 3, ManaCost: 5, CastTime: 3, Effect: "damage", DmgMin: 4, DmgMax: 16, DmgType: "cold"},
		{ID: 120, Name: "Frost Ray", School: "Conjuration", Level: 6, ManaCost: 8, CastTime: 3, Effect: "damage", DmgMin: 7, DmgMax: 22, DmgType: "cold"},
		{ID: 121, Name: "Freezing Sphere", School: "Conjuration", Level: 9, ManaCost: 14, CastTime: 3, Effect: "damage", DmgMin: 10, DmgMax: 30, DmgType: "cold"},
		{ID: 122, Name: "Summon Familiar", School: "Conjuration", Level: 2, ManaCost: 10, CastTime: 5, Effect: "utility"},
		{ID: 123, Name: "Summon Earth Elemental", School: "Conjuration", Level: 12, ManaCost: 25, CastTime: 5, Effect: "utility"},
		{ID: 124, Name: "Inferno Glyph", School: "Conjuration", Level: 20, ManaCost: 25, CastTime: 4, Effect: "damage", DmgMin: 20, DmgMax: 55, DmgType: "heat"},
		{ID: 125, Name: "Thunder Glyph", School: "Conjuration", Level: 10, ManaCost: 15, CastTime: 3, Effect: "damage", DmgMin: 12, DmgMax: 30, DmgType: "electric"},
		{ID: 126, Name: "Ice Glyph", School: "Conjuration", Level: 15, ManaCost: 20, CastTime: 3, Effect: "damage", DmgMin: 15, DmgMax: 40, DmgType: "cold"},
		{ID: 127, Name: "Web", School: "Conjuration", Level: 10, ManaCost: 12, CastTime: 3, Effect: "utility"},
		{ID: 130, Name: "Mass Protection", School: "Conjuration", Level: 23, ManaCost: 30, CastTime: 4, Effect: "defense", DefBonus: 25},
		{ID: 131, Name: "Flaming Arrows", School: "Conjuration", Level: 18, ManaCost: 22, CastTime: 3, Effect: "damage", DmgMin: 15, DmgMax: 35, DmgType: "heat"},
		{ID: 132, Name: "Chain Lightning", School: "Conjuration", Level: 23, ManaCost: 28, CastTime: 4, Effect: "damage", DmgMin: 20, DmgMax: 50, DmgType: "electric"},
		{ID: 133, Name: "Globe of Protection II", School: "Conjuration", Level: 30, ManaCost: 40, CastTime: 4, Effect: "defense", DefBonus: 100},
		{ID: 134, Name: "Siryx's Terrible Tentacles", School: "Conjuration", Level: 25, ManaCost: 35, CastTime: 4, Effect: "utility", DmgMin: 20, DmgMax: 55, DmgType: "crushing"},
		{ID: 135, Name: "Storm Blade", School: "Conjuration", Level: 24, ManaCost: 30, CastTime: 3, Effect: "buff"},
		{ID: 136, Name: "Inferno Blade", School: "Conjuration", Level: 19, ManaCost: 25, CastTime: 3, Effect: "buff"},
		{ID: 137, Name: "Winter Blade", School: "Conjuration", Level: 22, ManaCost: 28, CastTime: 3, Effect: "buff"},
		{ID: 138, Name: "Energy Maelstrom", School: "Conjuration", Level: 31, ManaCost: 45, CastTime: 5, Effect: "damage", DmgMin: 30, DmgMax: 75, DmgType: "electric"},
		{ID: 141, Name: "Pyrotechnics", School: "Conjuration", Level: 17, ManaCost: 20, CastTime: 3, Effect: "damage", DmgMin: 12, DmgMax: 35, DmgType: "heat"},
		{ID: 139, Name: "Sorcerous Summons I", School: "Conjuration", Level: 20, ManaCost: 20, CastTime: 5, Effect: "utility"},
		{ID: 140, Name: "Sorcerous Summons II", School: "Conjuration", Level: 35, ManaCost: 35, CastTime: 5, Effect: "utility"},
		{ID: 144, Name: "Tindarath's Chaotic Summons", School: "Conjuration", Level: 28, ManaCost: 28, CastTime: 5, Effect: "utility"},
	}
	// Enchantment (200-250)
	ench := []SpellDef{
		{ID: 200, Name: "Fear", School: "Enchantment", Level: 1, ManaCost: 3, CastTime: 3, Effect: "utility"},
		{ID: 201, Name: "Charm", School: "Enchantment", Level: 3, ManaCost: 8, CastTime: 3, Effect: "utility"},
		{ID: 202, Name: "Enchantment I", School: "Enchantment", Level: 4, ManaCost: 10, CastTime: 4, Effect: "buff"},
		{ID: 203, Name: "Enchantment II", School: "Enchantment", Level: 15, ManaCost: 20, CastTime: 4, Effect: "buff"},
		{ID: 204, Name: "Enchantment III", School: "Enchantment", Level: 23, ManaCost: 28, CastTime: 4, Effect: "buff"},
		{ID: 207, Name: "Strength I", School: "Enchantment", Level: 4, ManaCost: 6, CastTime: 3, Effect: "buff"},
		{ID: 208, Name: "Strength II", School: "Enchantment", Level: 8, ManaCost: 10, CastTime: 3, Effect: "buff"},
		{ID: 209, Name: "Strength III", School: "Enchantment", Level: 16, ManaCost: 18, CastTime: 3, Effect: "buff"},
		{ID: 210, Name: "Haste", School: "Enchantment", Level: 5, ManaCost: 8, CastTime: 3, Effect: "buff"},
		{ID: 211, Name: "Slow", School: "Enchantment", Level: 5, ManaCost: 8, CastTime: 3, Effect: "utility"},
		{ID: 216, Name: "Slumber I", School: "Enchantment", Level: 2, ManaCost: 4, CastTime: 3, Effect: "utility"},
		{ID: 217, Name: "Slumber II", School: "Enchantment", Level: 6, ManaCost: 8, CastTime: 3, Effect: "utility"},
		{ID: 218, Name: "Slumber III", School: "Enchantment", Level: 18, ManaCost: 16, CastTime: 3, Effect: "utility"},
		{ID: 219, Name: "Silence", School: "Enchantment", Level: 7, ManaCost: 10, CastTime: 3, Effect: "utility"},
		{ID: 224, Name: "Fly", School: "Enchantment", Level: 11, ManaCost: 15, CastTime: 3, Effect: "buff"},
		{ID: 225, Name: "Invisibility", School: "Enchantment", Level: 14, ManaCost: 18, CastTime: 3, Effect: "buff"},
		{ID: 228, Name: "Identify", School: "Enchantment", Level: 7, ManaCost: 5, CastTime: 3, Effect: "utility"},
		{ID: 229, Name: "Wizard's Armor", School: "Enchantment", Level: 9, ManaCost: 12, CastTime: 3, Effect: "defense", DefBonus: 15},
		{ID: 234, Name: "Spell Shield", School: "Enchantment", Level: 13, ManaCost: 15, CastTime: 3, Effect: "defense", DefBonus: 25},
		{ID: 235, Name: "Cloak Mind", School: "Enchantment", Level: 22, ManaCost: 25, CastTime: 3, Effect: "defense", DefBonus: 25},
		{ID: 205, Name: "Command", School: "Enchantment", Level: 6, ManaCost: 6, CastTime: 3, Effect: "utility"},
		{ID: 206, Name: "Domination I", School: "Enchantment", Level: 12, ManaCost: 12, CastTime: 4, Effect: "utility"},
		{ID: 212, Name: "Mass Invisibility", School: "Enchantment", Level: 25, ManaCost: 25, CastTime: 4, Effect: "buff"},
		{ID: 213, Name: "Bend Space I", School: "Enchantment", Level: 17, ManaCost: 17, CastTime: 4, Effect: "utility"},
		{ID: 214, Name: "Domination II", School: "Enchantment", Level: 24, ManaCost: 24, CastTime: 4, Effect: "utility"},
		{ID: 215, Name: "Scry", School: "Enchantment", Level: 10, ManaCost: 10, CastTime: 4, Effect: "utility"},
		{ID: 220, Name: "Dancing Blade", School: "Enchantment", Level: 1, ManaCost: 1, CastTime: 3, Effect: "damage", DmgMin: 2, DmgMax: 8, DmgType: ""},
		{ID: 221, Name: "Dancing Sword", School: "Enchantment", Level: 6, ManaCost: 6, CastTime: 3, Effect: "damage", DmgMin: 6, DmgMax: 20, DmgType: ""},
		{ID: 222, Name: "Bend Space II", School: "Enchantment", Level: 23, ManaCost: 23, CastTime: 4, Effect: "utility"},
		{ID: 226, Name: "Paranoia", School: "Enchantment", Level: 3, ManaCost: 3, CastTime: 3, Effect: "utility"},
		{ID: 227, Name: "Imprisonment Rune", School: "Enchantment", Level: 13, ManaCost: 13, CastTime: 4, Effect: "utility"},
		{ID: 230, Name: "Disjunction", School: "Enchantment", Level: 21, ManaCost: 21, CastTime: 4, Effect: "utility"},
		{ID: 231, Name: "Imprison", School: "Enchantment", Level: 19, ManaCost: 19, CastTime: 4, Effect: "utility"},
		{ID: 232, Name: "Mist Form", School: "Enchantment", Level: 20, ManaCost: 20, CastTime: 4, Effect: "utility"},
		{ID: 243, Name: "Charge Wand", School: "Enchantment", Level: 26, ManaCost: 26, CastTime: 5, Effect: "utility"},
		{ID: 244, Name: "Enchant an Item", School: "Enchantment", Level: 31, ManaCost: 31, CastTime: 5, Effect: "utility"},
		{ID: 245, Name: "Slime Form", School: "Enchantment", Level: 13, ManaCost: 13, CastTime: 4, Effect: "utility"},
		{ID: 246, Name: "Yshtarin's Confounding Translocation", School: "Enchantment", Level: 29, ManaCost: 29, CastTime: 5, Effect: "utility"},
		{ID: 248, Name: "Phantom Form", School: "Enchantment", Level: 34, ManaCost: 34, CastTime: 4, Effect: "buff"},
	}
	// Necromancy (301-356)
	necro := []SpellDef{
		{ID: 301, Name: "Turn Undead I", School: "Necromancy", Level: 2, ManaCost: 4, CastTime: 3, Effect: "damage", DmgMin: 5, DmgMax: 15, DmgType: ""},
		{ID: 302, Name: "Turn Undead II", School: "Necromancy", Level: 8, ManaCost: 10, CastTime: 3, Effect: "damage", DmgMin: 10, DmgMax: 30, DmgType: ""},
		{ID: 303, Name: "Cure Poison", School: "Necromancy", Level: 11, ManaCost: 12, CastTime: 3, Effect: "utility"},
		{ID: 319, Name: "Cure Disease", School: "Necromancy", Level: 12, ManaCost: 14, CastTime: 3, Effect: "utility"},
		{ID: 313, Name: "Body Destruction I", School: "Necromancy", Level: 1, ManaCost: 3, CastTime: 3, Effect: "damage", DmgMin: 3, DmgMax: 10, DmgType: ""},
		{ID: 314, Name: "Body Destruction II", School: "Necromancy", Level: 5, ManaCost: 7, CastTime: 3, Effect: "damage", DmgMin: 6, DmgMax: 20, DmgType: ""},
		{ID: 315, Name: "Body Destruction III", School: "Necromancy", Level: 10, ManaCost: 14, CastTime: 3, Effect: "damage", DmgMin: 12, DmgMax: 35, DmgType: ""},
		{ID: 316, Name: "Body Restoration I", School: "Necromancy", Level: 1, ManaCost: 3, CastTime: 3, Effect: "heal", HealMin: 5, HealMax: 15},
		{ID: 317, Name: "Body Restoration II", School: "Necromancy", Level: 5, ManaCost: 7, CastTime: 3, Effect: "heal", HealMin: 10, HealMax: 30},
		{ID: 318, Name: "Body Restoration III", School: "Necromancy", Level: 10, ManaCost: 14, CastTime: 3, Effect: "heal", HealMin: 20, HealMax: 50},
		{ID: 323, Name: "Spectral Fist", School: "Necromancy", Level: 3, ManaCost: 5, CastTime: 3, Effect: "damage", DmgMin: 4, DmgMax: 14, DmgType: "crushing"},
		{ID: 326, Name: "Spectral Shield", School: "Necromancy", Level: 9, ManaCost: 12, CastTime: 3, Effect: "defense", DefBonus: 20},
		{ID: 334, Name: "Invigoration I", School: "Necromancy", Level: 2, ManaCost: 4, CastTime: 3, Effect: "heal", HealMin: 3, HealMax: 10},
		{ID: 335, Name: "Invigoration II", School: "Necromancy", Level: 9, ManaCost: 10, CastTime: 3, Effect: "heal", HealMin: 8, HealMax: 25},
		{ID: 337, Name: "Reconstruction", School: "Necromancy", Level: 4, ManaCost: 6, CastTime: 3, Effect: "heal", HealMin: 5, HealMax: 20},
		{ID: 338, Name: "Unstun", School: "Necromancy", Level: 9, ManaCost: 8, CastTime: 2, Effect: "utility"},
		{ID: 339, Name: "Destroy Undead I", School: "Necromancy", Level: 3, ManaCost: 5, CastTime: 3, Effect: "damage", DmgMin: 8, DmgMax: 20, DmgType: ""},
		{ID: 340, Name: "Destroy Undead II", School: "Necromancy", Level: 8, ManaCost: 12, CastTime: 3, Effect: "damage", DmgMin: 15, DmgMax: 40, DmgType: ""},
		{ID: 341, Name: "Destroy Undead III", School: "Necromancy", Level: 13, ManaCost: 20, CastTime: 3, Effect: "damage", DmgMin: 25, DmgMax: 60, DmgType: ""},
		{ID: 343, Name: "Regeneration", School: "Necromancy", Level: 27, ManaCost: 35, CastTime: 4, Effect: "heal", HealMin: 40, HealMax: 80},
		{ID: 345, Name: "Spectral Sword", School: "Necromancy", Level: 7, ManaCost: 10, CastTime: 3, Effect: "damage", DmgMin: 6, DmgMax: 22, DmgType: ""},
		{ID: 347, Name: "Divine Blessing", School: "Necromancy", Level: 10, ManaCost: 12, CastTime: 3, Effect: "buff"},
		{ID: 354, Name: "Rorin's Fire", School: "Necromancy", Level: 17, ManaCost: 22, CastTime: 3, Effect: "damage", DmgMin: 15, DmgMax: 40, DmgType: "heat"},
		{ID: 305, Name: "Breath of Life", School: "Necromancy", Level: 14, ManaCost: 14, CastTime: 5, Effect: "utility"},
		{ID: 306, Name: "Animate Skeleton", School: "Necromancy", Level: 6, ManaCost: 6, CastTime: 5, Effect: "utility"},
		{ID: 307, Name: "Animate Zombie", School: "Necromancy", Level: 10, ManaCost: 10, CastTime: 5, Effect: "utility"},
		{ID: 308, Name: "Control Undead I", School: "Necromancy", Level: 7, ManaCost: 7, CastTime: 4, Effect: "utility"},
		{ID: 309, Name: "Control Undead II", School: "Necromancy", Level: 13, ManaCost: 13, CastTime: 4, Effect: "utility"},
		{ID: 311, Name: "Speak With Dead", School: "Necromancy", Level: 3, ManaCost: 3, CastTime: 4, Effect: "utility"},
		{ID: 353, Name: "Summon Spectral Warrior", School: "Necromancy", Level: 32, ManaCost: 32, CastTime: 5, Effect: "utility"},
	}
	// General (400-415)
	gen := []SpellDef{
		{ID: 400, Name: "Detect Magic", School: "General", Level: 1, ManaCost: 2, CastTime: 2, Effect: "utility"},
		{ID: 401, Name: "Dispel Lesser Magic", School: "General", Level: 5, ManaCost: 8, CastTime: 3, Effect: "utility"},
		{ID: 403, Name: "Mindlink", School: "General", Level: 9, ManaCost: 12, CastTime: 3, Effect: "utility"},
		{ID: 405, Name: "See Hidden", School: "General", Level: 3, ManaCost: 5, CastTime: 3, Effect: "utility"},
		{ID: 406, Name: "Dispel Invisibility", School: "General", Level: 8, ManaCost: 10, CastTime: 3, Effect: "utility"},
		{ID: 407, Name: "Analyze Ore", School: "General", Level: 3, ManaCost: 4, CastTime: 3, Effect: "utility"},
		{ID: 404, Name: "Aura Sense", School: "General", Level: 14, ManaCost: 14, CastTime: 3, Effect: "utility"},
		{ID: 408, Name: "Truename", School: "General", Level: 18, ManaCost: 18, CastTime: 4, Effect: "utility"},
		{ID: 412, Name: "Bloodsight", School: "General", Level: 9, ManaCost: 9, CastTime: 3, Effect: "utility"},
	}
	// Druidic (500-538)
	druid := []SpellDef{
		{ID: 500, Name: "Plant Snare", School: "Druidic", Level: 4, ManaCost: 6, CastTime: 3, Effect: "utility"},
		{ID: 505, Name: "Freedom", School: "Druidic", Level: 9, ManaCost: 12, CastTime: 3, Effect: "utility"},
		{ID: 507, Name: "Heat Shield", School: "Druidic", Level: 7, ManaCost: 10, CastTime: 3, Effect: "buff"},
		{ID: 508, Name: "Cold Shield", School: "Druidic", Level: 6, ManaCost: 8, CastTime: 3, Effect: "buff"},
		{ID: 511, Name: "Carapace", School: "Druidic", Level: 8, ManaCost: 10, CastTime: 3, Effect: "defense", DefBonus: 20},
		{ID: 512, Name: "True Aim", School: "Druidic", Level: 15, ManaCost: 18, CastTime: 3, Effect: "buff"},
		{ID: 513, Name: "Agility I", School: "Druidic", Level: 4, ManaCost: 6, CastTime: 3, Effect: "buff"},
		{ID: 514, Name: "Agility II", School: "Druidic", Level: 11, ManaCost: 12, CastTime: 3, Effect: "buff"},
		{ID: 515, Name: "Agility III", School: "Druidic", Level: 16, ManaCost: 20, CastTime: 3, Effect: "buff"},
		{ID: 519, Name: "Sunray", School: "Druidic", Level: 13, ManaCost: 18, CastTime: 3, Effect: "damage", DmgMin: 12, DmgMax: 35, DmgType: "heat"},
		{ID: 520, Name: "Night Vision", School: "Druidic", Level: 1, ManaCost: 2, CastTime: 2, Effect: "utility"},
		{ID: 521, Name: "Camouflage", School: "Druidic", Level: 7, ManaCost: 8, CastTime: 3, Effect: "buff"},
		{ID: 523, Name: "Earth Spike", School: "Druidic", Level: 5, ManaCost: 7, CastTime: 3, Effect: "damage", DmgMin: 5, DmgMax: 18, DmgType: "crushing"},
		{ID: 524, Name: "Earth Wave", School: "Druidic", Level: 12, ManaCost: 16, CastTime: 3, Effect: "damage", DmgMin: 10, DmgMax: 30, DmgType: "crushing"},
		{ID: 501, Name: "Call Storm", School: "Druidic", Level: 23, ManaCost: 23, CastTime: 5, Effect: "utility"},
		{ID: 502, Name: "Disperse Storm", School: "Druidic", Level: 19, ManaCost: 19, CastTime: 4, Effect: "utility"},
		{ID: 503, Name: "Call Lightning", School: "Druidic", Level: 17, ManaCost: 17, CastTime: 3, Effect: "damage", DmgMin: 20, DmgMax: 55, DmgType: "electric"},
		{ID: 504, Name: "Call Animal", School: "Druidic", Level: 1, ManaCost: 1, CastTime: 4, Effect: "utility"},
		{ID: 506, Name: "Resist Weather", School: "Druidic", Level: 3, ManaCost: 3, CastTime: 3, Effect: "buff"},
		{ID: 509, Name: "Repel Plants", School: "Druidic", Level: 10, ManaCost: 10, CastTime: 3, Effect: "buff"},
		{ID: 510, Name: "Repel Plants and Webs", School: "Druidic", Level: 18, ManaCost: 18, CastTime: 3, Effect: "buff"},
		{ID: 516, Name: "Wall of Thorns", School: "Druidic", Level: 14, ManaCost: 14, CastTime: 4, Effect: "utility"},
		{ID: 517, Name: "Stick to Snake", School: "Druidic", Level: 5, ManaCost: 5, CastTime: 4, Effect: "utility"},
		{ID: 518, Name: "Claw Growth", School: "Druidic", Level: 2, ManaCost: 2, CastTime: 3, Effect: "buff"},
		{ID: 522, Name: "Insect Swarm", School: "Druidic", Level: 25, ManaCost: 25, CastTime: 4, Effect: "damage", DmgMin: 15, DmgMax: 40, DmgType: ""},
		{ID: 526, Name: "Creeping Death", School: "Druidic", Level: 22, ManaCost: 22, CastTime: 4, Effect: "damage", DmgMin: 18, DmgMax: 45, DmgType: ""},
		{ID: 528, Name: "Free Action", School: "Druidic", Level: 20, ManaCost: 20, CastTime: 3, Effect: "buff"},
		{ID: 531, Name: "Tree Door", School: "Druidic", Level: 10, ManaCost: 10, CastTime: 4, Effect: "utility"},
		{ID: 532, Name: "Ride the Lightning", School: "Druidic", Level: 24, ManaCost: 24, CastTime: 3, Effect: "defense", DefBonus: 50},
		{ID: 533, Name: "Commune with Nature", School: "Druidic", Level: 27, ManaCost: 27, CastTime: 5, Effect: "utility"},
		{ID: 534, Name: "Claws of the Elder Wolf", School: "Druidic", Level: 21, ManaCost: 21, CastTime: 3, Effect: "buff"},
		{ID: 535, Name: "Form Lock", School: "Druidic", Level: 18, ManaCost: 18, CastTime: 4, Effect: "utility"},
	}

	spellRegistry = append(spellRegistry, conj...)
	spellRegistry = append(spellRegistry, ench...)
	spellRegistry = append(spellRegistry, necro...)
	spellRegistry = append(spellRegistry, gen...)
	spellRegistry = append(spellRegistry, druid...)
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

// spellMasteryLevel returns the player's purchased mastery rank for a spell.
func spellMasteryLevel(player *Player, spell *SpellDef) int {
	if player.SpellMastery == nil {
		return 0
	}
	return player.SpellMastery[spell.ID]
}

// doMasterSpell handles the MASTER command — purchase a rank of spell mastery.
// Cost: 8 BP for rank 1, 4 BP for each additional rank.
// Prereq: school skill >= (currentRank+2) × spellLevel.
// Max ranks: spellLevel + 1.
func (e *GameEngine) doMasterSpell(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Master which spell? (MASTER <spell name or number>)"}}}

	// Resolve spell by number or name
	query := strings.ToLower(strings.Join(args, " "))
	var spell *SpellDef
	if n, err := strconv.Atoi(query); err == nil {
		spell = FindSpellByID(n)
	} else {
		for id := range player.KnownSpells {
			s := FindSpellByID(id)
			if s != nil && strings.EqualFold(s.Name, query) {
				spell = s
				break
			}
		}
	}
	if spell == nil {
		return &CommandResult{Messages: []string{"You don't know that spell."}}
	}
	if !player.KnownSpells[spell.ID] {
		return &CommandResult{Messages: []string{"You don't know that spell."}}
	}

	current := 0
	if player.SpellMastery != nil {
		current = player.SpellMastery[spell.ID]
	}
	maxRanks := spell.Level + 1
	if current >= maxRanks {
		return &CommandResult{Messages: []string{fmt.Sprintf("You have fully mastered %s (%d/%d ranks).", spell.Name, current, maxRanks)}}
	}

	// Skill prerequisite: school skill >= 2 × spellLevel (fixed threshold, does not grow with mastery rank)
	schoolSkillID := spellSchoolSkill(spell.School)
	required := 2 * spell.Level
	if player.Skills[schoolSkillID] < required {
		return &CommandResult{Messages: []string{fmt.Sprintf("You need %s skill %d to master %s (rank %d). You have %d.", spell.School, required, spell.Name, current+1, player.Skills[schoolSkillID])}}
	}

	// Build point cost
	bpCost := 8
	if current > 0 {
		bpCost = 4
	}
	if player.BuildPoints < bpCost {
		return &CommandResult{Messages: []string{fmt.Sprintf("You need %d build points to master %s (you have %d).", bpCost, spell.Name, player.BuildPoints)}}
	}

	if player.SpellMastery == nil {
		player.SpellMastery = make(map[int]int)
	}
	player.SpellMastery[spell.ID] = current + 1
	player.BuildPoints -= bpCost

	newRank := current + 1
	stars := strings.Repeat("*", newRank)
	msgs := []string{fmt.Sprintf("You study %s intensely, achieving rank %d of mastery. (%s) [-%d BP, %d BP remaining]",
		spell.Name, newRank, stars, bpCost, player.BuildPoints)}
	// Cast time is a flat -1s for any mastery (3→2), shown only on rank 1
	if newRank == 1 {
		newCastTime := spell.CastTime - 1
		if newCastTime < 2 {
			newCastTime = 2
		}
		msgs = append(msgs, fmt.Sprintf("Casting time for %s is now %d seconds (when not hasted).", spell.Name, newCastTime))
	}
	dmgBonus := newRank * 3 / 2
	if spell.Effect == "damage" {
		msgs = append(msgs, fmt.Sprintf("Mastery damage bonus: +%d (when not hasted).", dmgBonus))
	}
	newCost := effectiveManaCost(spell, newRank)
	msgs = append(msgs, fmt.Sprintf("Mana cost for %s is now %d.", spell.Name, newCost))

	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: msgs}
}

// effectiveManaCost returns the mana cost after mastery reduction.
// Base cost = spell level; each mastery rank reduces by 2; floor at half base (min 1).
func effectiveManaCost(spell *SpellDef, mastery int) int {
	base := spell.Level
	if base < 1 {
		base = 1
	}
	minCost := base / 2
	if minCost < 1 {
		minCost = 1
	}
	cost := base - mastery*2
	if cost < minCost {
		cost = minCost
	}
	return cost
}

// isHasted returns true if the player is currently under any haste effect (spell or psionic).
func isHasted(player *Player) bool {
	return !player.HasteExpiry.IsZero() && time.Now().Before(player.HasteExpiry)
}

// effectiveCastTime returns the roundtime for preparing a spell, applying mastery and haste/slow.
// Any mastery (rank 1+) reduces cast time by 1 second (flat, not per-rank), minimum 2 seconds.
// Mastery is cancelled by haste. Slow doubles the resulting time.
func effectiveCastTime(spell *SpellDef, mastery int, player *Player) int {
	base := spell.CastTime
	if isHasted(player) {
		// Haste cancels mastery; haste halves the base time
		return base / 2
	}
	result := base
	if mastery > 0 {
		result = base - 1
		if result < 2 {
			result = 2
		}
	}
	if !player.SlowExpiry.IsZero() && time.Now().Before(player.SlowExpiry) {
		result *= 2
	}
	return result
}

// masteryDamageBonus returns the flat damage bonus from mastery on an offensive spell.
// ~+1.5 damage per rank, only when not hasted (haste cancels mastery per MAGIC.TXT).
func masteryDamageBonus(mastery int, player *Player) int {
	if mastery <= 0 || isHasted(player) {
		return 0
	}
	return mastery * 3 / 2
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
        if !matchesTarget(name, target, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
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
        fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3, ii.Tail)
        player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
        if player.KnownSpells == nil {
            player.KnownSpells = make(map[int]bool)
        }
        player.KnownSpells[spellNum] = true

        studyRT := applyRoundTime(player, 5)
        player.RoundTimeExpiry = time.Now().Add(time.Duration(studyRT) * time.Second)
        e.SavePlayer(ctx, player)
        return &CommandResult{
            Messages: []string{
                fmt.Sprintf("You study %s carefully...", fullName),
                fmt.Sprintf("You learn %s! The scroll crumbles to dust.", spell.Name),
                fmt.Sprintf("[Round: %d sec]", studyRT),
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

// trainSpellFromTeacher handles learning a spell from a player teacher via TRAIN.
func (e *GameEngine) trainSpellFromTeacher(ctx context.Context, player *Player, teacher *Player, spell *SpellDef) *CommandResult {
	if player.KnownSpells[spell.ID] {
		return &CommandResult{Messages: []string{fmt.Sprintf("You already know %s.", spell.Name)}}
	}

	requiredSkill := schoolSkillID(spell.School)
	if requiredSkill < 0 {
		return &CommandResult{Messages: []string{"You cannot learn spells of that school."}}
	}

	playerSkillLevel := player.Skills[requiredSkill]
	if playerSkillLevel < spell.Level {
		return &CommandResult{Messages: []string{
			fmt.Sprintf("You need %s rank %d to learn %s (you have rank %d).",
				SkillNames[requiredSkill], spell.Level, spell.Name, playerSkillLevel),
		}}
	}

	if player.KnownSpells == nil {
		player.KnownSpells = make(map[int]bool)
	}
	player.KnownSpells[spell.ID] = true
	learnRT := applyRoundTime(player, 5)
	player.RoundTimeExpiry = time.Now().Add(time.Duration(learnRT) * time.Second)
	e.SavePlayer(ctx, player)

	return &CommandResult{
		Messages: []string{
			fmt.Sprintf("You study %s's instruction carefully...", teacher.FirstName),
			fmt.Sprintf("You learn %s!", spell.Name),
			fmt.Sprintf("[Round: %d sec]", learnRT),
		},
		RoomBroadcast: []string{
			fmt.Sprintf("%s learns %s from %s.", player.FirstName, spell.Name, teacher.FirstName),
		},
	}
}

// isUndeadOnlySpell returns true for Necromancy spells that only affect undead creatures
// (RACE 22 in the monster script data): Turn Undead I/II and Destroy Undead I/II/III.
func isUndeadOnlySpell(spellID int) bool {
	switch spellID {
	case 301, 302, 339, 340, 341:
		return true
	}
	return false
}

// spellReagentArch returns the required inventory item archetype for spells that need a reagent.
// Returns 0 if no reagent is required.
func spellReagentArch(spellID int) int {
	switch spellID {
	case 203:
		return 520 // spider eye
	case 204:
		return 526 // imp toe
	case 106:
		return 109 // garnet for fire elemental
	case 107:
		return 116 // opal for air elemental
	case 108:
		return 102 // aquamarine for water elemental
	case 109:
		return 107 // diamond for gargoyle
	case 123:
		return 122 // tourmaline for earth elemental
	case 305:
		return 494 // mandrake root for Breath of Life
	case 353:
		return 512 // ghoul dust for Summon Spectral Warrior
	}
	return 0
}

// spellReagentName returns a display name for the required reagent.
func spellReagentName(spellID int) string {
	switch spellID {
	case 203:
		return "a spider eye"
	case 204:
		return "an imp toe"
	case 106:
		return "a garnet"
	case 107:
		return "an opal"
	case 108:
		return "an aquamarine"
	case 109:
		return "a diamond"
	case 123:
		return "a tourmaline"
	case 305:
		return "some mandrake root"
	case 353:
		return "some ghoul dust"
	}
	return ""
}

// spellReagentBaseName returns the bare noun (no article) for the reagent message.
func spellReagentBaseName(spellID int) string {
	switch spellID {
	case 203:
		return "spider eye"
	case 204:
		return "imp toe"
	case 106:
		return "garnet"
	case 107:
		return "opal"
	case 108:
		return "aquamarine"
	case 109:
		return "diamond"
	case 123:
		return "tourmaline"
	case 305:
		return "mandrake root"
	case 353:
		return "ghoul dust"
	}
	return "reagent"
}

// spellReagentConsumeMessage returns the message shown when a spell's reagent is
// consumed during PREPARE. Most spells share the generic wording; a few have
// their own flavor text.
func spellReagentConsumeMessage(spellID int) string {
	if spellID == 305 {
		return "Some mandrake root turns to dust as it is absorbed by the magic of your spell!"
	}
	if spellID == 353 {
		return "The ghoul dust swirls away into nothingness as it is consumed by the spell!"
	}
	return fmt.Sprintf("The %s turns to dust as it is consumed by the spell.", spellReagentBaseName(spellID))
}

// spellConsumesReagentAtPrepare returns true for spells whose reagent is consumed during PREPARE
// rather than at CAST time (elementals, gargoyle summons, and Breath of Life).
func spellConsumesReagentAtPrepare(spellID int) bool {
	switch spellID {
	case 106, 107, 108, 109, 123, 305, 353:
		return true
	}
	return false
}

// doPrepareSpell handles PREPARE/INVOKE <spell> [WITH <reagent>].
func (e *GameEngine) doPrepareSpell(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Prepare what spell?"}}
	}
	if player.Dead {
		return &CommandResult{Messages: []string{"You can't cast spells while dead."}}
	}
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You are still recovering from your last action. (%.0f seconds remaining)", remaining+0.5)}}
	}

	// Parse optional "WITH <reagent>" suffix: "203 with spider eye" → spellPart="203", reagentArg="spider eye"
	joined := strings.ToLower(strings.Join(args, " "))
	var spellPart, reagentArg string
	if idx := strings.Index(joined, " with "); idx >= 0 {
		spellPart = strings.TrimSpace(strings.Join(args, " ")[:idx])
		reagentArg = strings.TrimSpace(strings.Join(args, " ")[idx+6:])
	} else {
		spellPart = strings.Join(args, " ")
	}

	var spell *SpellDef
	if id, err := strconv.Atoi(strings.TrimSpace(spellPart)); err == nil {
		spell = FindSpellByID(id)
	} else {
		spell = FindSpellByName(strings.TrimSpace(spellPart))
	}
	if spell == nil {
		return &CommandResult{Messages: []string{"You don't know that spell."}}
	}
	if !player.KnownSpells[spell.ID] && !player.IsGM {
		return &CommandResult{Messages: []string{fmt.Sprintf("You haven't learned %s.", spell.Name)}}
	}
	if player.Mana < spell.ManaCost {
		return &CommandResult{Messages: []string{fmt.Sprintf("You don't have enough mana. (%s costs %d, you have %d)", spell.Name, spell.ManaCost, player.Mana)}}
	}

	// Check reagent requirement
	reqArch := spellReagentArch(spell.ID)
	if reqArch > 0 {
		if reagentArg == "" {
			return &CommandResult{Messages: []string{fmt.Sprintf("%s requires a reagent. Use: PREPARE %d WITH %s", spell.Name, spell.ID, spellReagentName(spell.ID))}}
		}
		// Find the reagent in inventory by name match and archetype
		reagentArg = strings.ToLower(reagentArg)
		found := false
		for i, ii := range player.Inventory {
			def := e.items[ii.Archetype]
			if def == nil || ii.Archetype != reqArch {
				continue
			}
			if spell.ID == 353 && ii.Adj1 != 546 { // must specifically be ghoul dust
				continue
			}
			noun := e.getItemNounName(def)
			if matchesTarget(noun, reagentArg, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
				player.PreparedSpellReagentArch = ii.Archetype
				if spellConsumesReagentAtPrepare(spell.ID) {
					player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
				}
				found = true
				break
			}
		}
		if !found {
			return &CommandResult{Messages: []string{fmt.Sprintf("%s requires %s, which you don't have.", spell.Name, spellReagentName(spell.ID))}}
		}
	} else {
		player.PreparedSpellReagentArch = 0
	}

	player.PreparedSpell = spell.ID
	mastery := spellMasteryLevel(player, spell)
	prepRT := effectiveCastTime(spell, mastery, player)
	player.RoundTimeExpiry = time.Now().Add(time.Duration(prepRT) * time.Second)

	var prepMsgs []string
	if spellConsumesReagentAtPrepare(spell.ID) {
		prepMsgs = append(prepMsgs, spellReagentConsumeMessage(spell.ID))
	}
	prepMsgs = append(prepMsgs, fmt.Sprintf("You begin preparing %s... (type CAST to release, or CAST <target>)", spell.Name))
	prepMsgs = append(prepMsgs, fmt.Sprintf("[Round: %d sec]", prepRT))
	return &CommandResult{
		Messages:      prepMsgs,
		RoomBroadcast: []string{fmt.Sprintf("%s begins preparing a spell.", player.FirstName)},
	}
}

// doRelease handles RELEASE — cancels a prepared spell or psionic discipline
// without casting it. Falls back to releasing a carried player if nothing is
// prepared, since RELEASE/PUTDOWN also breaks a CARRY.
func (e *GameEngine) doRelease(ctx context.Context, player *Player) *CommandResult {
	if player.PreparedSpell != 0 {
		player.PreparedSpell = 0
		player.PreparedSpellReagentArch = 0
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"You release your prepared spell."}}
	}
	if player.PreparedPsi != 0 {
		player.PreparedPsi = 0
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"You release your prepared discipline."}}
	}
	if player.Carrying != "" {
		return e.doReleaseCarry(ctx, player)
	}
	return &CommandResult{Messages: []string{"You have nothing prepared."}}
}

// doCastSpell handles CAST [target].
func (e *GameEngine) doCastSpell(ctx context.Context, player *Player, args []string) *CommandResult {
	if player.Dead {
		return &CommandResult{Messages: []string{"You can't cast spells while dead."}}
	}

	// If no spell prepared, try to prepare+cast in one step
	if player.PreparedSpell == 0 {
		if len(args) == 0 {
			return &CommandResult{Messages: []string{"You have no spell prepared. Use PREPARE <spell> first."}}
		}
		// Try direct cast: "cast flame bolt <target>"
		spellName := strings.Join(args, " ")
		spell := FindSpellByName(spellName)
		if spell == nil {
			return &CommandResult{Messages: []string{fmt.Sprintf("You don't know a spell called '%s'.", spellName)}}
		}
		if !player.KnownSpells[spell.ID] && !player.IsGM {
			return &CommandResult{Messages: []string{fmt.Sprintf("You haven't learned %s.", spell.Name)}}
		}
		if spellReagentArch(spell.ID) > 0 {
			return &CommandResult{Messages: []string{fmt.Sprintf("%s requires a reagent. Use: PREPARE %d WITH %s", spell.Name, spell.ID, spellReagentName(spell.ID))}}
		}
		player.PreparedSpell = spell.ID
	}

	spell := FindSpellByID(player.PreparedSpell)
	if spell == nil {
		player.PreparedSpell = 0
		return &CommandResult{Messages: []string{"Your spell fizzles."}}
	}

	// Mana cost = spell level, reduced by mastery (from LEGENDS.DOC)
	mastery := spellMasteryLevel(player, spell)
	manaCost := effectiveManaCost(spell, mastery)
	if player.Mana < manaCost {
		player.PreparedSpell = 0
		return &CommandResult{Messages: []string{fmt.Sprintf("Not enough mana! (%s requires %d, you have %d)", spell.Name, manaCost, player.Mana)}}
	}

	// Check roundtime
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := player.RoundTimeExpiry.Sub(time.Now()).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You are still preparing... %.0f seconds remaining.", remaining+0.5)}}
	}

	// Deduct mana
	player.Mana -= manaCost
	player.PreparedSpell = 0

	// Spellcraft skill check (from LEGENDS.DOC):
	// Base 25% + EMP/10 + spellcraft*5%, max 95%.
	// Roll > 98 = fumble. Roll <= 2 = spectacular success (double effect).
	spellcraftSkill := player.Skills[23]
	castChance := 25 + player.Empathy/10 + spellcraftSkill*5
	if castChance > 95 {
		castChance = 95
	}
	if player.IsGM {
		castChance = 100
	}

	castRoll := rand.Intn(100) + 1
	if castRoll == 100 && !player.IsGM {
		// Extreme failure!
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("[Success: %d%%, Roll %d] Extreme failure! The spell backfires!", castChance, castRoll)},
			RoomBroadcast: []string{fmt.Sprintf("Magic begins to form around %s but then fizzles.", player.FirstName)},
		}
	}

	spectacularSuccess := castRoll == 1

	if castRoll > castChance && !player.IsGM {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("[Success: %d%%, Roll %d] Failure.", castChance, castRoll)},
			RoomBroadcast: []string{fmt.Sprintf("Magic begins to form around %s but then fizzles.", player.FirstName)},
		}
	}

	// Show success roll to caster
	successMsg := fmt.Sprintf("[Success: %d%%, Roll %d] Success!", castChance, castRoll)
	if spectacularSuccess {
		successMsg = fmt.Sprintf("[Success: %d%%, Roll %d] Spectacular success!", castChance, castRoll)
	}

	result := &CommandResult{}

	switch spell.Effect {
	case "damage":
		result = e.castDamageSpell(player, spell, args, spectacularSuccess)
	case "heal":
		result = e.castHealSpell(ctx, player, spell, args)
	case "defense":
		result = e.castTimedDefenseSpell(player, spell, args)
	case "buff":
		result = e.castBuffSpell(player, spell, args)
	case "utility":
		switch spell.ID {
		case 114: // Mystic Key
			result = e.castMysticKey(player, args)
		case 127: // Web
			result = e.castWebSpell(player, spell, args)
		case 134: // Siryx's Terrible Tentacles
			result = e.castTentaclesSpell(player, spell, args)
		case 200: // Fear
			result = e.castFearSpell(player, spell, args)
		case 201: // Charm
			result = e.castCharmSpell(player, spell, args)
		case 211: // Slow
			result = e.castSlowSpell(ctx, player, spell, args)
		case 216, 217, 218: // Slumber I/II/III
			result = e.castSlumberSpell(player, spell, args)
		case 303: // Cure Poison
			result = e.castCureSpell(player, spell, args, "poisoned", func(p *Player) bool { return p.Poisoned }, func(p *Player) { p.Poisoned = false; p.PoisonLevel = 0 })
		case 319: // Cure Disease
			result = e.castCureSpell(player, spell, args, "diseased", func(p *Player) bool { return p.Diseased }, func(p *Player) { p.Diseased = false; p.DiseaseLevel = 0 })
		case 122: // Summon Familiar
			result = e.castSummonFamiliar(player)
		case 504: // Call Animal
			result = e.castCallAnimal(player)
		case 517: // Stick to Snake
			result = e.castStickToSnake(player)
		case 106, 107, 108, 109, 123: // Summon Fire/Air/Water/Gargoyle/Earth Elemental
			result = e.castSummonElemental(player, spell)
		case 306: // Animate Skeleton
			result = e.castAnimateUndead(player, 1, "The ground trembles as bones knit together, rising as")
		case 307: // Animate Zombie
			result = e.castAnimateUndead(player, 2, "The earth churns as a rotting corpse claws free, becoming")
		case 308, 309: // Control Undead I/II
			result = e.castControlUndead(player, spell, args)
		case 311: // Speak with Dead
			result = e.castSpeakWithDead(player, args)
		case 353: // Summon Spectral Warrior
			result = e.castSummonSpectralWarrior(player)
		case 400: // Detect Magic
			result = e.castDetectMagic(player, args)
		case 305: // Breath of Life
			result = e.castBreathOfLife(ctx, player, args)
		case 213: // Bend Space I
			result = e.castBendSpaceI(ctx, player, args)
		case 222: // Bend Space II
			result = e.castBendSpaceII(ctx, player, args)
		default:
			result.Messages = []string{fmt.Sprintf("You gesture and cast %s.", spell.Name)}
			result.RoomBroadcast = []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)}
		}
	default:
		result.Messages = []string{fmt.Sprintf("You gesture and cast %s.", spell.Name)}
		result.RoomBroadcast = []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)}
	}

	// Prepend success roll message
	result.Messages = append([]string{successMsg}, result.Messages...)

	e.SavePlayer(ctx, player)

	return result
}

func (e *GameEngine) castDamageSpell(player *Player, spell *SpellDef, args []string, spectacular bool) *CommandResult {
	// Chain Lightning and Flaming Arrows hit every creature in the player's
	// TARGET list at once when one has been built up; fall through to the
	// normal single-target resolution below if no (valid) list exists.
	if spell.ID == 132 && len(e.resolveTargets(player)) > 0 {
		return e.castChainLightningSpell(player, spell, args, spectacular)
	}
	if spell.ID == 131 && len(e.resolveTargets(player)) > 0 {
		return e.castFlamingArrowsSpell(player, spell, args, spectacular)
	}

	// Find target
	targetName := ""
	if len(args) > 0 {
		targetName = strings.Join(args, " ")
	} else if player.CombatTarget != nil && player.CombatTarget.IsMonster {
		// Auto-target current combat target
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
		return &CommandResult{Messages: []string{"Cast at what? Specify a target."}}
	}

	inst, def := e.findMonsterInRoom(player, targetName)
	if inst == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' here.", targetName)}}
	}

	name := FormatMonsterName(def, e.monAdjs)

	// Turn Undead I/II and Destroy Undead I/II/III only affect the undead (RACE 22).
	if isUndeadOnlySpell(spell.ID) && def.Race != 22 {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s at a %s, but it has no effect — only the undead are vulnerable to this magic!", spell.Name, name)},
			RoomBroadcast: []string{fmt.Sprintf("%s casts %s at a %s, but nothing happens.", player.FirstName, spell.Name, name)},
		}
	}

	// Article for monster name ("a " prefix)
	article := "a "

	// Call Meteor is a special cast that hammers the target with both heat
	// and crushing damage in a single blow, each rolled independently.
	if spell.ID == 112 {
		return e.castMeteorSpell(player, spell, inst, def, name, article, spectacular)
	}

	dmg := rand.Intn(spell.DmgMax-spell.DmgMin+1) + spell.DmgMin
	dmg += masteryDamageBonus(spellMasteryLevel(player, spell), player)

	// Apply magic resistance
	if def.MagicResist > 0 {
		resistRoll := rand.Intn(100)
		if resistRoll < def.MagicResist {
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You gesture and cast %s at a %s, but it resists the spell!", spell.Name, name)},
				RoomBroadcast: []string{fmt.Sprintf("%s casts %s at a %s, but it resists!", player.FirstName, spell.Name, name)},
			}
		}
	}

	// Apply elemental immunity
	if spell.DmgType != "" {
		immType := elementalImmunityType(spell.DmgType)
		if level, ok := def.Immunities[immType]; ok {
			dmg = applyImmunity(dmg, level)
		}
	}

	if dmg <= 0 {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You cast %s at a %s, but it seems unaffected!", spell.Name, name)},
			RoomBroadcast: []string{fmt.Sprintf("%s casts %s at a %s!", player.FirstName, spell.Name, name)},
		}
	}

	if spectacular {
		dmg = dmg * 2
	}

	// Generic spell flavor text based on damage type. The caster sees a
	// second-person ("You...") version; onlookers see a third-person version.
	flavorSelf := fmt.Sprintf("You form a bolt of energy and hurl it at %s%s!", article, name)
	flavorRoom := fmt.Sprintf("%s forms a bolt of energy and hurls it at %s%s!", player.FirstName, article, name)
	flavorDmg := fmt.Sprintf("%s %s to %s. [%d Damage]", damageSeverity(dmg), spellDmgNoun(spell.DmgType), randomBodyPart(def.BodyType), dmg)
	switch spell.DmgType {
	case "heat":
		flavorSelf = fmt.Sprintf("You form a ball of flame and hurl it at %s%s!", article, name)
		flavorRoom = fmt.Sprintf("%s forms a ball of flame and hurls it at %s%s!", player.FirstName, article, name)
		flavorDmg = fmt.Sprintf("%s burn to %s. [%d Damage]", damageSeverity(dmg), randomBodyPart(def.BodyType), dmg)
	case "cold":
		flavorSelf = fmt.Sprintf("You form a freezing sphere from the air and hurl it at %s%s!", article, name)
		flavorRoom = fmt.Sprintf("%s forms a freezing sphere from the air and hurls it at %s%s!", player.FirstName, article, name)
		flavorDmg = fmt.Sprintf("%s blast to %s. [%d Damage]", damageSeverity(dmg), randomBodyPart(def.BodyType), dmg)
	case "electric":
		flavorSelf = fmt.Sprintf("You release a bolt of lightning at %s%s!", article, name)
		flavorRoom = fmt.Sprintf("%s releases a bolt of lightning at %s%s!", player.FirstName, article, name)
		flavorDmg = fmt.Sprintf("%s shock to %s. [%d Damage]", damageSeverity(dmg), randomBodyPart(def.BodyType), dmg)
	case "crushing":
		flavorSelf = fmt.Sprintf("You hurl a force blast at %s%s!", article, name)
		flavorRoom = fmt.Sprintf("%s hurls a force blast at %s%s!", player.FirstName, article, name)
		flavorDmg = fmt.Sprintf("%s strike to %s. [%d Damage]", damageSeverity(dmg), randomBodyPart(def.BodyType), dmg)
	}

	// Generic pre-cast gesture line; some spells (e.g. Earth Spike) gesture
	// at something other than the target and override this below.
	gestureSelf := fmt.Sprintf("You gesture at %s%s.", article, name)
	gestureRoom := fmt.Sprintf("%s gestures at %s%s.", player.FirstName, article, name)

	// Per-spell custom flavor overrides the generic damage-type text above.
	switch spell.ID {
	case 103: // Lightning Bolt
		flavorSelf = fmt.Sprintf("You hurl a bolt of lightning at %s%s!", article, name)
		flavorRoom = fmt.Sprintf("%s hurls a bolt of lightning at %s%s!", player.FirstName, article, name)
	case 120: // Frost Ray
		flavorSelf = fmt.Sprintf("You point your finger at %s%s and a ray of intense cold shoots forth!", article, name)
		flavorRoom = fmt.Sprintf("%s points a finger at %s%s and a ray of intense cold shoots forth!", player.FirstName, article, name)
	case 345: // Spectral Sword
		flavorSelf = fmt.Sprintf("A ghostly sword materializes before you and slashes at %s%s!", article, name)
		flavorRoom = fmt.Sprintf("A ghostly sword materializes before %s and slashes at %s%s!", player.FirstName, article, name)
	case 523: // Earth Spike
		gestureSelf = "You gesture towards the ground."
		gestureRoom = fmt.Sprintf("%s gestures towards the ground.", player.FirstName)
		flavorSelf = fmt.Sprintf("As you beckon to the ground, a horrible spike thrusts up from the earth and impales %s%s!", article, name)
		flavorRoom = fmt.Sprintf("As %s beckons to the ground, a horrible spike thrusts up from the earth and impales %s%s!", player.FirstName, article, name)
	}

	killed, woke := e.damageMonster(inst.ID, dmg)

	var msgs, roomMsgs []string
	if woke {
		msgs = append(msgs, fmt.Sprintf("The %s wakes up, startled by your spell!", name))
	}
	msgs = append(msgs, gestureSelf)
	roomMsgs = append(roomMsgs, gestureRoom)
	msgs = append(msgs, flavorSelf)
	roomMsgs = append(roomMsgs, flavorRoom)
	msgs = append(msgs, flavorDmg)
	roomMsgs = append(roomMsgs, flavorDmg)

	if killed {
		deathText := def.TextOverrides["TEXD"]
		if deathText != "" {
			msgs = append(msgs, fmt.Sprintf("A %s %s", name, deathText))
			roomMsgs = append(roomMsgs, fmt.Sprintf("A %s %s", name, deathText))
		} else {
			msgs = append(msgs, "He collapses, dead.")
			roomMsgs = append(roomMsgs, fmt.Sprintf("A %s collapses, dead!", name))
		}
		e.handleMonsterDeath(player, inst, def)
		player.CombatTarget = nil
		player.Joined = false
	}

	return &CommandResult{Messages: msgs, RoomBroadcast: roomMsgs}
}

// castMeteorSpell handles Call Meteor (112), which hammers the target with
// both heat and crushing damage from a single meteor strike, each rolled
// and displayed as its own damage line.
func (e *GameEngine) castMeteorSpell(player *Player, spell *SpellDef, inst *MonsterInstance, def *gameworld.MonsterDef, name, article string, spectacular bool) *CommandResult {
	heatDmg := rand.Intn(spell.DmgMax-spell.DmgMin+1) + spell.DmgMin
	heatDmg += masteryDamageBonus(spellMasteryLevel(player, spell), player)
	crushDmg := rand.Intn(spell.DmgMax-spell.DmgMin+1) + spell.DmgMin
	crushDmg += masteryDamageBonus(spellMasteryLevel(player, spell), player)

	// A single magic resistance roll covers both damage components.
	if def.MagicResist > 0 {
		resistRoll := rand.Intn(100)
		if resistRoll < def.MagicResist {
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You gesture and cast %s at a %s, but it resists the spell!", spell.Name, name)},
				RoomBroadcast: []string{fmt.Sprintf("%s casts %s at a %s, but it resists!", player.FirstName, spell.Name, name)},
			}
		}
	}

	if level, ok := def.Immunities[elementalImmunityType("heat")]; ok {
		heatDmg = applyImmunity(heatDmg, level)
	}
	if level, ok := def.Immunities[elementalImmunityType("crushing")]; ok {
		crushDmg = applyImmunity(crushDmg, level)
	}
	if heatDmg < 0 {
		heatDmg = 0
	}
	if crushDmg < 0 {
		crushDmg = 0
	}

	if spectacular {
		heatDmg *= 2
		crushDmg *= 2
	}

	totalDmg := heatDmg + crushDmg
	if totalDmg <= 0 {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You cast %s at a %s, but it seems unaffected!", spell.Name, name)},
			RoomBroadcast: []string{fmt.Sprintf("%s casts %s at a %s!", player.FirstName, spell.Name, name)},
		}
	}

	flavorSelf := fmt.Sprintf("You point to the sky. A moment later, a meteor screeches from the heavens and hammers %s%s!", article, name)
	flavorRoom := fmt.Sprintf("%s points to the sky. A moment later, a meteor screeches from the heavens and hammers %s%s!", player.FirstName, article, name)

	killed, woke := e.damageMonster(inst.ID, totalDmg)

	var msgs, roomMsgs []string
	if woke {
		msgs = append(msgs, fmt.Sprintf("The %s wakes up, startled by your spell!", name))
	}
	msgs = append(msgs, fmt.Sprintf("You gesture at %s%s.", article, name))
	roomMsgs = append(roomMsgs, fmt.Sprintf("%s gestures at %s%s.", player.FirstName, article, name))
	msgs = append(msgs, flavorSelf)
	roomMsgs = append(roomMsgs, flavorRoom)
	if heatDmg > 0 {
		burnLine := fmt.Sprintf("%s burn to %s. [%d Damage]", damageSeverity(heatDmg), randomBodyPart(def.BodyType), heatDmg)
		msgs = append(msgs, burnLine)
		roomMsgs = append(roomMsgs, burnLine)
	}
	if crushDmg > 0 {
		blowLine := fmt.Sprintf("%s blow to %s. [%d Damage]", damageSeverity(crushDmg), randomBodyPart(def.BodyType), crushDmg)
		msgs = append(msgs, blowLine)
		roomMsgs = append(roomMsgs, blowLine)
	}

	if killed {
		deathText := def.TextOverrides["TEXD"]
		if deathText != "" {
			msgs = append(msgs, fmt.Sprintf("A %s %s", name, deathText))
			roomMsgs = append(roomMsgs, fmt.Sprintf("A %s %s", name, deathText))
		} else {
			msgs = append(msgs, "He collapses, dead.")
			roomMsgs = append(roomMsgs, fmt.Sprintf("A %s collapses, dead!", name))
		}
		e.handleMonsterDeath(player, inst, def)
		player.CombatTarget = nil
		player.Joined = false
	}

	return &CommandResult{Messages: msgs, RoomBroadcast: roomMsgs}
}

// resolveChainStart picks which resolved target entry a multi-target spell
// should start/gesture at: the one matching the CAST argument if given
// (matched the same way findMonsterInRoom matches names), else the first
// entry in the target list.
func (e *GameEngine) resolveChainStart(entries []targetEntry, args []string) (int, bool) {
	if len(args) == 0 {
		return 0, true
	}
	named := strings.ToLower(strings.TrimSpace(strings.Join(args, " ")))
	for _, article := range []string{"a ", "an ", "the ", "some "} {
		if strings.HasPrefix(named, article) {
			named = strings.TrimPrefix(named, article)
			break
		}
	}
	for i, en := range entries {
		name := strings.ToLower(FormatMonsterName(en.Def, e.monAdjs))
		noun := strings.ToLower(en.Def.Name)
		if strings.HasPrefix(name, named) || strings.HasPrefix(noun, named) {
			return i, true
		}
	}
	return 0, false
}

// chainOrderIndices returns the traversal order for a chaining multi-target
// spell over n resolved targets: the start index first, then the rest of
// the target list in reverse of the order they were added (most recently
// added target is the next hop, so casting always visits everyone targeted).
func chainOrderIndices(n, startIdx int) []int {
	order := []int{startIdx}
	for i := n - 1; i >= 0; i-- {
		if i != startIdx {
			order = append(order, i)
		}
	}
	return order
}

// castChainLightningSpell handles Chain Lightning (132) when the caster has
// a multi-target TARGET list: the bolt arcs from the caster to the named (or
// first) target, then chains through the remaining targeted creatures in
// reverse of the order they were added to the list (most recently added
// first), so casting always visits every currently-targeted creature.
func (e *GameEngine) castChainLightningSpell(player *Player, spell *SpellDef, args []string, spectacular bool) *CommandResult {
	entries := e.resolveTargets(player)
	if len(entries) == 0 {
		return &CommandResult{Messages: []string{"Cast at what? Specify a target."}}
	}

	startIdx, found := e.resolveChainStart(entries, args)
	if !found {
		return &CommandResult{Messages: []string{fmt.Sprintf("You aren't targeting '%s'.", strings.Join(args, " "))}}
	}

	order := chainOrderIndices(len(entries), startIdx)

	startName := strings.ToLower(FormatMonsterName(entries[startIdx].Def, e.monAdjs))
	startArticle := articleFor(startName, entries[startIdx].Def.Unique)

	msgs := []string{fmt.Sprintf("You gesture at %s%s.", startArticle, startName)}
	roomMsgs := []string{fmt.Sprintf("%s gestures at %s%s.", player.FirstName, startArticle, startName)}

	prevSelf := player.FirstName
	prevRoom := player.FirstName

	for _, idx := range order {
		en := entries[idx]
		name := strings.ToLower(FormatMonsterName(en.Def, e.monAdjs))
		article := articleFor(name, en.Def.Unique)
		fullName := article + name

		if en.Def.MagicResist > 0 && rand.Intn(100) < en.Def.MagicResist {
			line := fmt.Sprintf("Lightning arcs from %s to %s%s, but it resists!", prevSelf, article, name)
			msgs = append(msgs, line)
			roomMsgs = append(roomMsgs, fmt.Sprintf("Lightning arcs from %s to %s%s, but it resists!", prevRoom, article, name))
			prevSelf, prevRoom = fullName, fullName
			continue
		}

		dmg := rand.Intn(spell.DmgMax-spell.DmgMin+1) + spell.DmgMin
		dmg += masteryDamageBonus(spellMasteryLevel(player, spell), player)
		if level, ok := en.Def.Immunities[elementalImmunityType(spell.DmgType)]; ok {
			dmg = applyImmunity(dmg, level)
		}
		if spectacular {
			dmg *= 2
		}

		if dmg <= 0 {
			line := fmt.Sprintf("Lightning arcs from %s to %s%s, but it seems unaffected!", prevSelf, article, name)
			msgs = append(msgs, line)
			roomMsgs = append(roomMsgs, fmt.Sprintf("Lightning arcs from %s to %s%s, but it seems unaffected!", prevRoom, article, name))
			prevSelf, prevRoom = fullName, fullName
			continue
		}

		arcSelf := fmt.Sprintf("Lightning arcs from %s to %s%s!", prevSelf, article, name)
		arcRoom := fmt.Sprintf("Lightning arcs from %s to %s%s!", prevRoom, article, name)
		dmgLine := fmt.Sprintf(" %s %s to %s. [%d Damage]", damageSeverity(dmg), spellDmgNoun(spell.DmgType), randomBodyPart(en.Def.BodyType), dmg)
		msgs = append(msgs, arcSelf, dmgLine)
		roomMsgs = append(roomMsgs, arcRoom, dmgLine)

		killed, _ := e.damageMonster(en.Inst.ID, dmg)
		if killed {
			deathText := en.Def.TextOverrides["TEXD"]
			if deathText != "" {
				msgs = append(msgs, fmt.Sprintf("A %s %s", name, deathText))
				roomMsgs = append(roomMsgs, fmt.Sprintf("A %s %s", name, deathText))
			} else {
				msgs = append(msgs, "It collapses, dead.")
				roomMsgs = append(roomMsgs, fmt.Sprintf("A %s collapses, dead!", name))
			}
			instCopy := en.Inst
			e.handleMonsterDeath(player, &instCopy, en.Def)
			player.Targets = removeTargetID(player.Targets, en.Inst.ID)
			if player.CombatTarget != nil && player.CombatTarget.MonsterID == en.Inst.ID {
				player.CombatTarget = nil
				player.Joined = false
			}
		}

		prevSelf, prevRoom = fullName, fullName
	}

	return &CommandResult{Messages: msgs, RoomBroadcast: roomMsgs}
}

// castFlamingArrowsSpell handles Flaming Arrows (131) when the caster has a
// multi-target TARGET list: one arrow flies from the caster and strikes each
// targeted creature independently (no chaining between targets).
func (e *GameEngine) castFlamingArrowsSpell(player *Player, spell *SpellDef, args []string, spectacular bool) *CommandResult {
	entries := e.resolveTargets(player)
	if len(entries) == 0 {
		return &CommandResult{Messages: []string{"Cast at what? Specify a target."}}
	}

	var msgs, roomMsgs []string
	for _, en := range entries {
		name := strings.ToLower(FormatMonsterName(en.Def, e.monAdjs))
		article := articleFor(name, en.Def.Unique)

		if en.Def.MagicResist > 0 && rand.Intn(100) < en.Def.MagicResist {
			line := fmt.Sprintf("A flaming arrow flies from %s and strikes %s%s, but it resists!", player.FirstName, article, name)
			msgs = append(msgs, line)
			roomMsgs = append(roomMsgs, line)
			continue
		}

		dmg := rand.Intn(spell.DmgMax-spell.DmgMin+1) + spell.DmgMin
		dmg += masteryDamageBonus(spellMasteryLevel(player, spell), player)
		if level, ok := en.Def.Immunities[elementalImmunityType(spell.DmgType)]; ok {
			dmg = applyImmunity(dmg, level)
		}
		if spectacular {
			dmg *= 2
		}

		if dmg <= 0 {
			line := fmt.Sprintf("A flaming arrow flies from %s and strikes %s%s, but it seems unaffected!", player.FirstName, article, name)
			msgs = append(msgs, line)
			roomMsgs = append(roomMsgs, line)
			continue
		}

		flyLine := fmt.Sprintf("A flaming arrow flies from %s and strikes %s%s!", player.FirstName, article, name)
		dmgLine := fmt.Sprintf(" %s %s to %s. [%d Damage]", damageSeverity(dmg), spellDmgNoun(spell.DmgType), randomBodyPart(en.Def.BodyType), dmg)
		msgs = append(msgs, flyLine, dmgLine)
		roomMsgs = append(roomMsgs, flyLine, dmgLine)

		killed, _ := e.damageMonster(en.Inst.ID, dmg)
		if killed {
			deathText := en.Def.TextOverrides["TEXD"]
			if deathText != "" {
				msgs = append(msgs, fmt.Sprintf("A %s %s", name, deathText))
				roomMsgs = append(roomMsgs, fmt.Sprintf("A %s %s", name, deathText))
			} else {
				msgs = append(msgs, "It collapses, dead.")
				roomMsgs = append(roomMsgs, fmt.Sprintf("A %s collapses, dead!", name))
			}
			instCopy := en.Inst
			e.handleMonsterDeath(player, &instCopy, en.Def)
			player.Targets = removeTargetID(player.Targets, en.Inst.ID)
			if player.CombatTarget != nil && player.CombatTarget.MonsterID == en.Inst.ID {
				player.CombatTarget = nil
				player.Joined = false
			}
		}
	}

	return &CommandResult{Messages: msgs, RoomBroadcast: roomMsgs}
}

// castTentaclesSpell handles Siryx's Terrible Tentacles (134): tentacles
// burst from the ground and immobilize every creature in the caster's
// TARGET list (falling back to a single named/auto target if no list has
// been built), each taking periodic crushing damage from tentacleDamageTick
// until it dies, breaks free, or the spell expires. Unlike Web, there is no
// body-point limit on what it can immobilize.
func (e *GameEngine) castTentaclesSpell(player *Player, spell *SpellDef, args []string) *CommandResult {
	entries := e.resolveTargets(player)
	if len(entries) == 0 {
		targetName := strings.Join(args, " ")
		if targetName == "" {
			targetName = e.autoTargetMonsterName(player)
		}
		if targetName == "" {
			return &CommandResult{Messages: []string{"Cast Siryx's Terrible Tentacles at what?"}}
		}
		inst, def := e.findMonsterInRoom(player, targetName)
		if inst == nil {
			return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' here.", targetName)}}
		}
		entries = []targetEntry{{Inst: *inst, Def: def}}
	}

	startIdx, found := e.resolveChainStart(entries, args)
	if !found {
		return &CommandResult{Messages: []string{fmt.Sprintf("You aren't targeting '%s'.", strings.Join(args, " "))}}
	}
	startName := strings.ToLower(FormatMonsterName(entries[startIdx].Def, e.monAdjs))
	startArticle := articleFor(startName, entries[startIdx].Def.Unique)

	msgs := []string{
		fmt.Sprintf("You gesture at %s%s.", startArticle, startName),
		"Black tentacles burst forth from the ground!",
	}
	roomMsgs := []string{
		fmt.Sprintf("%s gestures at %s%s.", player.FirstName, startArticle, startName),
		"Black tentacles burst forth from the ground!",
	}

	for _, en := range entries {
		name := strings.ToLower(FormatMonsterName(en.Def, e.monAdjs))
		article := articleFor(name, en.Def.Unique)

		if en.Def.Discorporate {
			line := fmt.Sprintf("The tentacles pass right through %s%s!", article, name)
			msgs = append(msgs, line)
			roomMsgs = append(roomMsgs, line)
			continue
		}
		if en.Def.MagicResist > 0 && rand.Intn(100) < en.Def.MagicResist {
			line := fmt.Sprintf("%s%s resists the tentacles!", capArticle(article), name)
			msgs = append(msgs, line)
			roomMsgs = append(roomMsgs, line)
			continue
		}

		line := fmt.Sprintf("Tentacles grab hold of %s%s!", article, name)
		msgs = append(msgs, line)
		roomMsgs = append(roomMsgs, line)

		e.monsterMgr.mu.Lock()
		for i := range e.monsterMgr.instances {
			if e.monsterMgr.instances[i].ID == en.Inst.ID {
				e.monsterMgr.instances[i].Tentacled = true
				e.monsterMgr.instances[i].TentacleExpiry = time.Now().Add(60 * time.Second)
				e.monsterMgr.instances[i].TentacleCasterName = player.FirstName
				e.monsterMgr.instances[i].Target = ""
				break
			}
		}
		e.monsterMgr.mu.Unlock()
	}

	return &CommandResult{Messages: msgs, RoomBroadcast: roomMsgs}
}

func (e *GameEngine) castHealSpell(ctx context.Context, player *Player, spell *SpellDef, args []string) *CommandResult {
	// Reconstruction (337) only heals the undead; it has its own resolution logic.
	if spell.ID == 337 {
		return e.castReconstruction(player, spell, args)
	}

	// Resolve target: self by default, or named player in room
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

	amount := rand.Intn(spell.HealMax-spell.HealMin+1) + spell.HealMin

	// Body Restoration I/II/III (316/317/318) sear the undead with holy energy instead of healing them.
	if (spell.ID == 316 || spell.ID == 317 || spell.ID == 318) && target.Undead {
		return e.castBodyRestorationOnUndead(ctx, player, target, spell, targetName, amount)
	}

	// Regeneration (343) heals once immediately, then again every minute for 5 more minutes.
	if spell.ID == 343 {
		target.RegenerationAmount = amount
		target.RegenerationTicksLeft = 5
	}

	// Invigoration spells restore fatigue instead of body points
	if spell.ID == 334 || spell.ID == 335 {
		target.Fatigue += amount
		if target.Fatigue > target.MaxFatigue {
			target.Fatigue = target.MaxFatigue
		}
		if target == player {
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You gesture and cast %s on yourself. You feel invigorated! [Fatigue: %d/%d]", spell.Name, target.Fatigue, target.MaxFatigue)},
				RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
			}
		}
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s.", spell.Name, targetName)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, targetName)},
			TargetName:    target.FirstName,
			TargetMsg:     []string{fmt.Sprintf("%s casts %s on you. You feel invigorated! [Fatigue: %d/%d]", player.FirstName, spell.Name, target.Fatigue, target.MaxFatigue)},
		}
	}

	target.BodyPoints += amount
	if target.BodyPoints > target.MaxBodyPoints {
		target.BodyPoints = target.MaxBodyPoints
	}

	if target == player {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s on yourself, healing %d body points. [BP: %d/%d]", spell.Name, amount, target.BodyPoints, target.MaxBodyPoints)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
		}
	}

	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s, healing %d body points.", spell.Name, targetName, amount)},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, targetName)},
		TargetName:    target.FirstName,
		TargetMsg:     []string{fmt.Sprintf("%s casts %s on you, healing %d body points. [BP: %d/%d]", player.FirstName, spell.Name, amount, target.BodyPoints, target.MaxBodyPoints)},
	}
}

// castBodyRestorationOnUndead handles Body Restoration I/II/III (316/317/318) when the
// target is undead: the healing energy sears the undead flesh as damage instead.
func (e *GameEngine) castBodyRestorationOnUndead(ctx context.Context, player, target *Player, spell *SpellDef, targetName string, dmg int) *CommandResult {
	target.BodyPoints -= dmg
	if target.BodyPoints < 0 {
		target.BodyPoints = 0
	}

	if target == player {
		msgs := []string{fmt.Sprintf("You gesture and cast %s on yourself, but your undead flesh sears with holy energy! [%d Damage]", spell.Name, dmg)}
		if target.BodyPoints <= 0 {
			msgs = append(msgs, e.handlePlayerDeath(target, "the necromantic backlash")...)
		}
		e.SavePlayer(ctx, target)
		return &CommandResult{Messages: msgs, RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)}}
	}

	result := &CommandResult{
		Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s, but it sears %s's undead flesh! [%d Damage]", spell.Name, targetName, targetName, dmg)},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, targetName)},
		TargetName:    target.FirstName,
		TargetMsg:     []string{fmt.Sprintf("%s casts %s on you! Holy energy sears your undead flesh! [%d Damage]", player.FirstName, spell.Name, dmg)},
		PlayerState:   target,
	}
	if target.BodyPoints <= 0 {
		deathMsgs := e.handlePlayerDeath(target, "the necromantic backlash")
		result.TargetMsg = append(result.TargetMsg, deathMsgs...)
	}
	e.SavePlayer(ctx, target)
	return result
}

// castReconstruction handles Reconstruction (337) — heals only undead targets;
// living targets are unaffected.
func (e *GameEngine) castReconstruction(player *Player, spell *SpellDef, args []string) *CommandResult {
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

	if !target.Undead {
		if target == player {
			return &CommandResult{Messages: []string{fmt.Sprintf("You gesture and cast %s, but the spell fizzles — you are not undead.", spell.Name)}}
		}
		return &CommandResult{Messages: []string{fmt.Sprintf("You gesture and cast %s on %s, but the spell fizzles — %s is not undead.", spell.Name, targetName, targetName)}}
	}

	amount := rand.Intn(spell.HealMax-spell.HealMin+1) + spell.HealMin
	target.BodyPoints += amount
	if target.BodyPoints > target.MaxBodyPoints {
		target.BodyPoints = target.MaxBodyPoints
	}

	if target == player {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s, knitting your undead flesh back together, healing %d body points. [BP: %d/%d]", spell.Name, amount, target.BodyPoints, target.MaxBodyPoints)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
		}
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s, knitting %s's undead flesh back together, healing %d body points.", spell.Name, targetName, targetName, amount)},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, targetName)},
		TargetName:    target.FirstName,
		TargetMsg:     []string{fmt.Sprintf("%s casts %s on you, restoring %d body points. [BP: %d/%d]", player.FirstName, spell.Name, amount, target.BodyPoints, target.MaxBodyPoints)},
		PlayerState:   target,
	}
}

func (e *GameEngine) castCureSpell(player *Player, spell *SpellDef, args []string, condition string, check func(*Player) bool, clear func(*Player)) *CommandResult {
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

	if !check(target) {
		if target == player {
			return &CommandResult{Messages: []string{fmt.Sprintf("You gesture and cast %s, but you are not %s.", spell.Name, condition)}}
		}
		return &CommandResult{
			Messages: []string{fmt.Sprintf("You gesture and cast %s on %s, but %s is not %s.", spell.Name, targetName, targetName, condition)},
		}
	}
	clear(target)
	if target == player {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s. You feel the %s leave your body!", spell.Name, condition)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on themselves.", player.FirstName, spell.Name)},
			PlayerState:   target,
		}
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s.", spell.Name, targetName)},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, targetName)},
		TargetName:    target.FirstName,
		TargetMsg:     []string{fmt.Sprintf("%s casts %s on you. You feel the %s leave your body!", player.FirstName, spell.Name, condition)},
		PlayerState:   target,
	}
}

// castBreathOfLife handles necromancy spell 305 — Breath of Life.
// Revives a dead player in the room: clears their Dead status and restores 1-10 body points.
func (e *GameEngine) castBreathOfLife(ctx context.Context, player *Player, args []string) *CommandResult {
	player.PreparedSpellReagentArch = 0
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Cast Breath of Life on whom? Specify a dead player's body."}}
	}
	targetName := strings.ToLower(strings.Join(args, " "))
	target := e.findPlayerInRoom(player, targetName)
	if target == nil {
		return &CommandResult{Messages: []string{"You don't see that here."}}
	}
	if !target.Dead {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s is not dead.", target.FirstName)}}
	}

	target.Dead = false
	target.BodyPoints = rand.Intn(10) + 1
	if target.BodyPoints > target.MaxBodyPoints {
		target.BodyPoints = target.MaxBodyPoints
	}
	e.SavePlayer(ctx, target)

	return &CommandResult{
		Messages: []string{
			fmt.Sprintf("You kneel down and lay your hands upon %s's body.", target.FirstName),
			"You feel like the conduit for an overwhelming force as you complete the spell.",
			fmt.Sprintf("A few seconds later, %s rises from death!", target.FirstName),
		},
		RoomBroadcast: []string{
			fmt.Sprintf("%s kneels down and lays hands upon %s's body.", player.FirstName, target.FirstName),
			fmt.Sprintf("A few seconds later, %s rises from death!", target.FirstName),
		},
		TargetName: target.FirstName,
		TargetMsg: []string{
			fmt.Sprintf("%s kneels down and lays hands upon your body, chanting words of power.", player.FirstName),
			"You feel a sudden rush of vitality pull you back from the void!",
		},
		PlayerState: target,
	}
}

// resolveBendSpaceMark parses the mark-number argument for a Bend Space spell
// (e.g. "CAST 2") and resolves it to the caster's destination room.
func (e *GameEngine) resolveBendSpaceMark(player *Player, args []string) (*gameworld.Room, string) {
	if len(args) == 0 {
		return nil, "Cast to which mark? (e.g. CAST 2)"
	}
	num, err := strconv.Atoi(args[0])
	if err != nil || num < 1 || num > 10 {
		return nil, "Mark number must be 1-10."
	}
	if player.Marks == nil {
		return nil, "You have no marks set. Use MARK <1-10> to mark a location first."
	}
	roomNum, ok := player.Marks[num]
	if !ok {
		return nil, fmt.Sprintf("Mark %d is not set.", num)
	}
	dest := e.rooms[roomNum]
	if dest == nil {
		return nil, "That mark leads to a place that no longer exists."
	}
	if dest.Number == player.RoomNumber {
		return nil, "You are already there."
	}
	return dest, ""
}

// castBendSpaceI teleports the caster alone to a marked location (spell 213).
// A GM caster stays hidden/invisible and generates no messages in either room.
func (e *GameEngine) castBendSpaceI(ctx context.Context, player *Player, args []string) *CommandResult {
	dest, errMsg := e.resolveBendSpaceMark(player, args)
	if errMsg != "" {
		return &CommandResult{Messages: []string{errMsg}}
	}

	if !player.IsGM {
		player.Hidden = false
		player.Invisible = false
	}

	originalRoom := player.RoomNumber
	player.RoomNumber = dest.Number
	e.SavePlayer(ctx, player)

	lookResult := e.doLook(player)
	result := &CommandResult{
		Messages: append([]string{
			"You gesture into the air.",
			"The color and detail of your surroundings swirl together. After a brief moment...",
		}, lookResult.Messages...),
		RoomName: lookResult.RoomName,
		RoomDesc: lookResult.RoomDesc,
		Exits:    lookResult.Exits,
		Items:    lookResult.Items,
		OldRoom:  originalRoom,
	}

	if !player.IsGM {
		result.OldRoomMsg = []string{fmt.Sprintf("%s gestures into the air and then vanishes with a soft *bamf* sound!", player.FirstName)}
		result.RoomBroadcast = []string{fmt.Sprintf("%s appears out of nowhere!", player.FirstName)}
	}

	e.applyEntryScripts(ctx, player, dest, result)
	return result
}

// castBendSpaceII teleports the caster and their entire group to a marked
// location (spell 222). A GM caster stays hidden/invisible and generates no
// messages in either room; group members simply travel along silently.
func (e *GameEngine) castBendSpaceII(ctx context.Context, player *Player, args []string) *CommandResult {
	dest, errMsg := e.resolveBendSpaceMark(player, args)
	if errMsg != "" {
		return &CommandResult{Messages: []string{errMsg}}
	}

	if !player.IsGM {
		player.Hidden = false
		player.Invisible = false
	}

	originalRoom := player.RoomNumber
	player.RoomNumber = dest.Number
	e.SavePlayer(ctx, player)

	// Bring the caster's group along BEFORE rendering anyone's look — otherwise
	// the caster's (and followers') own room render would list who's present
	// at dest before the group had actually arrived, making it look like they
	// didn't follow (same bug class as doMove).
	var movedFollowers []*Player
	if player.IsGroupLeader && len(player.GroupMembers) > 0 && e.sessions != nil {
		for _, memberName := range player.GroupMembers {
			for _, p := range e.sessions.OnlinePlayers() {
				if p.FirstName != memberName || p.RoomNumber != originalRoom || p.Dead {
					continue
				}
				p.RoomNumber = dest.Number
				p.Submitting = false
				e.disengageCombat(p)
				e.SavePlayer(ctx, p)
				movedFollowers = append(movedFollowers, p)
				break
			}
		}
	}

	lookResult := e.doLook(player)
	result := &CommandResult{
		Messages: append([]string{
			"You gesture into the air.",
			"The color and detail of your surroundings swirl together. After a brief moment...",
		}, lookResult.Messages...),
		RoomName: lookResult.RoomName,
		RoomDesc: lookResult.RoomDesc,
		Exits:    lookResult.Exits,
		Items:    lookResult.Items,
		OldRoom:  originalRoom,
	}

	if !player.IsGM {
		result.OldRoomMsg = []string{fmt.Sprintf("%s gestures and %s group vanishes one by one!", player.FirstName, player.Possessive())}
		result.RoomBroadcast = []string{fmt.Sprintf("%s's group appears out of nowhere!", player.FirstName)}
	}

	e.applyEntryScripts(ctx, player, dest, result)

	// Bring-along echoes, silently — the vanish/appear echoes above already
	// describe the whole group leaving/arriving together.
	for _, p := range movedFollowers {
		if e.sendToPlayer != nil {
			followLook := e.doLook(p)
			followMsgs := append([]string{
				"The color and detail of your surroundings swirl together. After a brief moment...",
			}, followLook.Messages...)
			e.sendToPlayer(p.FirstName, followMsgs)
		}
		e.applyEntryScripts(ctx, p, dest, &CommandResult{})
	}

	return result
}

func (e *GameEngine) castBuffSpell(player *Player, spell *SpellDef, args []string) *CommandResult {
	msg := fmt.Sprintf("You gesture and cast %s.", spell.Name)
	switch spell.ID {
	case 102: // Mystic Armor
		return e.castMysticArmor(player, spell, args)
	case 202, 203, 204: // Enchantment I/II/III
		return e.castEnchantmentSpell(player, spell, args)
	case 207, 208, 209: // Strength I/II/III
		return e.castStrengthSpell(player, spell, args)
	case 210: // Haste
		return e.castHasteSpell(player, spell, args)
	case 224: // Fly
		return e.castFlySpell(player, spell)
	case 225: // Invisibility
		player.Invisible = true
		msg = fmt.Sprintf("You gesture and cast %s. You fade from sight.", spell.Name)
	case 507, 508: // Heat Shield / Cold Shield
		return e.castElementalShieldSpell(player, spell, args)
	case 513, 514, 515: // Agility I/II/III
		return e.castAgilitySpell(player, spell, args)
	}
	return &CommandResult{
		Messages:      []string{msg},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
	}
}

// applyMysticArmorBuff applies or extends the Mystic Armor (spell 102) timed defense
// buff. Shared by CAST (castMysticArmor) and item-triggered casts
// (applyItemSpellOnPlayer) so a Mystic Armor potion behaves like the spell.
// Returns minutes remaining and whether this was a brand-new buff.
func applyMysticArmorBuff(target *Player, spell *SpellDef) (mins int, applied bool) {
	const maxDuration = 4 * time.Hour
	const stackDuration = 20 * time.Minute

	curActive := target.MysticArmorBonus > 0 && !target.MysticArmorExpiry.IsZero() && time.Now().Before(target.MysticArmorExpiry)
	if curActive {
		newExpiry := target.MysticArmorExpiry.Add(stackDuration)
		maxExpiry := time.Now().Add(maxDuration)
		if newExpiry.After(maxExpiry) {
			newExpiry = maxExpiry
		}
		target.MysticArmorExpiry = newExpiry
		return int(time.Until(target.MysticArmorExpiry).Minutes()) + 1, false
	}

	bonus := spell.DefBonus // 20
	target.MysticArmorBonus = bonus
	target.MysticArmorExpiry = time.Now().Add(stackDuration)
	target.DefenseBonus += bonus
	return 20, true
}

// castMysticArmor handles Mystic Armor (spell 102) as a temporary defense buff.
// Initial cast: +20 defense for 20 minutes. Repeated casts extend duration by 20 minutes
// up to a 4-hour cap without stacking the defense bonus. Accepts an optional target name.
func (e *GameEngine) castMysticArmor(player *Player, spell *SpellDef, args []string) *CommandResult {
	// Resolve target: self by default, or named player in room
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

	mins, applied := applyMysticArmorBuff(target, spell)

	if !applied {
		if isSelf {
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You gesture and cast %s. The magical barrier around you strengthens! (%d minutes remaining)", spell.Name, mins)},
				RoomBroadcast: []string{fmt.Sprintf("%s gestures and a shimmering barrier surrounds them.", player.FirstName)},
			}
		}
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s, extending their barrier. (%d minutes remaining)", spell.Name, target.FirstName, mins)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and a shimmering barrier surrounds %s.", player.FirstName, target.FirstName)},
			TargetName:    target.FirstName,
			TargetMsg:     []string{fmt.Sprintf("%s casts %s on you, extending your barrier. (%d minutes remaining)", player.FirstName, spell.Name, mins)},
		}
	}

	if isSelf {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and %s takes effect! (+%d defense, 20 minutes)", spell.Name, spell.DefBonus)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and a shimmering barrier surrounds them.", player.FirstName)},
		}
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s.", spell.Name, target.FirstName)},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures and a shimmering barrier surrounds %s.", player.FirstName, target.FirstName)},
		TargetName:    target.FirstName,
		TargetMsg:     []string{fmt.Sprintf("%s casts %s on you! (+%d defense, 20 minutes)", player.FirstName, spell.Name, spell.DefBonus)},
	}
}

// applyElementalShield activates or extends a Heat Shield / Cold Shield buff on the
// given expiry field (20 minutes per cast, extends by 20 min on recast, 4-hour cap —
// same convention as the other timed buffs). Shared by CAST (castElementalShieldSpell)
// and item-triggered casts (applyItemSpellOnPlayer). Returns minutes remaining and
// whether this was a brand-new application (false = only extended an active one).
func applyElementalShield(expiry *time.Time) (mins int, applied bool) {
	const maxDuration = 4 * time.Hour
	const stackDuration = 20 * time.Minute
	now := time.Now()

	if !expiry.IsZero() && now.Before(*expiry) {
		newExpiry := expiry.Add(stackDuration)
		if cap := now.Add(maxDuration); newExpiry.After(cap) {
			newExpiry = cap
		}
		*expiry = newExpiry
		return int(time.Until(*expiry).Minutes()) + 1, false
	}

	*expiry = now.Add(stackDuration)
	return 20, true
}

// castElementalShieldSpell handles Heat Shield (507) and Cold Shield (508): while
// active, damage of the matching element taken by the target is halved. See
// monsterAttackPlayer and checkTrap for where the reduction is applied.
func (e *GameEngine) castElementalShieldSpell(player *Player, spell *SpellDef, args []string) *CommandResult {
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

	var expiry *time.Time
	elementName := "heat"
	if spell.ID == 508 {
		expiry = &target.ColdShieldExpiry
		elementName = "cold"
	} else {
		expiry = &target.HeatShieldExpiry
	}

	mins, applied := applyElementalShield(expiry)

	if !applied {
		if isSelf {
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You gesture and cast %s. Your resistance to %s strengthens! (%d minutes remaining)", spell.Name, elementName, mins)},
				RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
			}
		}
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s, extending their protection. (%d minutes remaining)", spell.Name, target.FirstName, mins)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, target.FirstName)},
			TargetName:    target.FirstName,
			TargetMsg:     []string{fmt.Sprintf("%s casts %s on you, extending your protection. (%d minutes remaining)", player.FirstName, spell.Name, mins)},
		}
	}

	if isSelf {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s. You feel protected from %s! (50%% resistance, 20 minutes)", spell.Name, elementName)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
		}
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s.", spell.Name, target.FirstName)},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, target.FirstName)},
		TargetName:    target.FirstName,
		TargetMsg:     []string{fmt.Sprintf("%s casts %s on you. You feel protected from %s! (50%% resistance, 20 minutes)", player.FirstName, spell.Name, elementName)},
	}
}

// applyTimedDefenseBuff applies or extends a timed defense buff (all "defense"-effect
// spells except Mystic Armor, which has its own dedicated system below). Shared by
// CAST (castTimedDefenseSpell) and item-triggered casts (applyItemSpellOnPlayer) so a
// defense-spell potion behaves exactly like the spell of the same name. Returns
// minutes remaining and whether this was a brand-new buff (false = extended one).
func applyTimedDefenseBuff(target *Player, spell *SpellDef) (mins int, applied bool) {
	const maxDuration = 4 * time.Hour
	const stackDuration = 20 * time.Minute
	now := time.Now()

	for i := range target.TimedDefenseBuffs {
		b := &target.TimedDefenseBuffs[i]
		if b.SpellID == spell.ID && now.Before(b.Expiry) {
			newExpiry := b.Expiry.Add(stackDuration)
			if cap := now.Add(maxDuration); newExpiry.After(cap) {
				newExpiry = cap
			}
			b.Expiry = newExpiry
			return int(time.Until(b.Expiry).Minutes()) + 1, false
		}
	}

	target.TimedDefenseBuffs = append(target.TimedDefenseBuffs, TimedDefenseBuff{
		SpellID:   spell.ID,
		SpellName: spell.Name,
		Bonus:     spell.DefBonus,
		Expiry:    now.Add(stackDuration),
	})
	target.DefenseBonus += spell.DefBonus
	return 20, true
}

// castTimedDefenseSpell handles all defense spells (except Mystic Armor which has its own function).
// It mirrors castMysticArmor: self-cast by default, or targets a named player in the room.
// First cast applies spell.DefBonus for 20 minutes. Additional casts extend by 20 min (4-hour cap)
// without re-adding the bonus. On expiry, regen.go removes the bonus and notifies the player.
func (e *GameEngine) castTimedDefenseSpell(player *Player, spell *SpellDef, args []string) *CommandResult {
	// Resolve target: self by default, or a named player in the room.
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

	mins, applied := applyTimedDefenseBuff(target, spell)

	if !applied {
		if !isSelf {
			e.SavePlayer(context.Background(), target)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s, extending their protection. (%d minutes remaining)", spell.Name, target.FirstName, mins)},
				RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, target.FirstName)},
				TargetName:    target.FirstName,
				TargetMsg:     []string{fmt.Sprintf("%s casts %s on you, extending your protection. (%d minutes remaining)", player.FirstName, spell.Name, mins)},
			}
		}
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s. The protection around you strengthens! (%d minutes remaining)", spell.Name, mins)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
		}
	}

	if !isSelf {
		e.SavePlayer(context.Background(), target)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s.", spell.Name, target.FirstName)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, target.FirstName)},
			TargetName:    target.FirstName,
			TargetMsg:     []string{fmt.Sprintf("%s casts %s on you! (+%d defense, 20 minutes)", player.FirstName, spell.Name, spell.DefBonus)},
		}
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You gesture and %s takes effect! (+%d defense, 20 minutes)", spell.Name, spell.DefBonus)},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
	}
}

// castEnchantmentSpell handles Enchantment I (202), II (203), III (204).
// Weapons receive +10/+20/+30; armor receives +5/+10/+15.
// Spells 203 and 204 require a reagent verified at PREPARE time; it is consumed here.
func (e *GameEngine) castEnchantmentSpell(player *Player, spell *SpellDef, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Enchant what? Specify a weapon or armor in your possession."}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, skip := parseOrdinal(target)

	adjIDs := map[int]int{202: 523, 203: 524, 204: 525} // enchanted, ensorcelled, eldritch
	weaponBonus := map[int]int{202: 10, 203: 20, 204: 30}
	armorBonus := map[int]int{202: 5, 203: 10, 204: 15}
	adjID := adjIDs[spell.ID]

	// Consume the reagent verified at PREPARE time. PreparedSpellReagentArch is only
	// set when the player self-prepared the spell with a reagent in hand; spells
	// chanted from a scroll (or otherwise prepared without going through PREPARE)
	// leave it at 0, so no reagent is required in that case.
	if player.PreparedSpellReagentArch != 0 {
		reqArch := player.PreparedSpellReagentArch
		consumed := false
		for i, ii := range player.Inventory {
			if ii.Archetype == reqArch {
				player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
				consumed = true
				break
			}
		}
		if !consumed {
			player.PreparedSpellReagentArch = 0
			return &CommandResult{Messages: []string{fmt.Sprintf("You no longer have the required reagent (%s). The spell fizzles.", spellReagentName(spell.ID))}}
		}
		player.PreparedSpellReagentArch = 0
	}

	type candidate struct {
		item *InventoryItem
		def  *gameworld.ItemDef
	}
	var candidates []candidate
	for i := range player.Inventory {
		def := e.items[player.Inventory[i].Archetype]
		if def != nil && (isWeapon(def.Type) || def.Type == "ARMOR" || def.Type == "SHIELD") {
			candidates = append(candidates, candidate{&player.Inventory[i], def})
		}
	}
	for i := range player.Worn {
		def := e.items[player.Worn[i].Archetype]
		if def != nil && def.Type == "ARMOR" {
			candidates = append(candidates, candidate{&player.Worn[i], def})
		}
	}
	if player.Wielded != nil {
		def := e.items[player.Wielded.Archetype]
		if def != nil && isWeapon(def.Type) {
			candidates = append(candidates, candidate{player.Wielded, def})
		}
	}
	if player.OffHand != nil {
		def := e.items[player.OffHand.Archetype]
		if def != nil && (def.Type == "SHIELD" || isWeapon(def.Type)) {
			candidates = append(candidates, candidate{player.OffHand, def})
		}
	}

	for _, c := range candidates {
		name := e.getItemNounName(c.def)
		if !matchesTargetOrdinal(name, target, &skip, e.getAdjName(c.item.Adj1), e.getAdjName(c.item.Adj2), e.getAdjName(c.item.Adj3)) {
			continue
		}
		if c.item.Val2 > 0 {
			return &CommandResult{Messages: []string{"That item already bears a magical enchantment."}}
		}
		oldName := e.formatItemName(c.def, c.item.Adj1, c.item.Adj2, c.item.Adj3, c.item.Tail)
		if c.def.Type == "ARMOR" || c.def.Type == "SHIELD" {
			c.item.Val2 = armorBonus[spell.ID]
		} else {
			c.item.Val2 = weaponBonus[spell.ID]
		}
		// Only add the enchantment adjective if there's a free adj slot; an item with
		// all three slots already occupied keeps its existing adjectives and just
		// receives the magical bonus (Val2, set above).
		if c.item.Adj1 == 0 || c.item.Adj2 == 0 || c.item.Adj3 == 0 {
			c.item.Adj3 = c.item.Adj2
			c.item.Adj2 = c.item.Adj1
			c.item.Adj1 = adjID
		}
		newName := e.formatItemName(c.def, c.item.Adj1, c.item.Adj2, c.item.Adj3, c.item.Tail)
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("A soft glow surrounds %s and then sinks into it.", oldName),
				fmt.Sprintf("It is now %s!", newName),
			},
			RoomBroadcast: []string{fmt.Sprintf("A soft glow surrounds an item %s is holding.", player.FirstName)},
		}
	}
	return &CommandResult{Messages: []string{"You don't have a weapon, armor, or shield matching that."}}
}

// strengthBuffTier returns 0=none, 1=Str I, 2=Str II, 3=Str III.
func strengthBuffTier(buffID int) int {
	switch buffID {
	case 207:
		return 1
	case 208:
		return 2
	case 209:
		return 3
	}
	return 0
}

// applyStrengthBuff applies, extends, or upgrades a timed Strength buff on target.
// Shared by CAST (castStrengthSpell) and item-triggered casts (applyItemSpellOnPlayer)
// so a Strength III potion behaves exactly like the spell of the same name.
// Returns the bonus this application represents, minutes remaining, whether a new/
// upgraded buff was applied (false = only extended an existing same-tier buff), and
// ok=false if a lower-tier spell was blocked by an active higher-tier buff.
func applyStrengthBuff(target *Player, spell *SpellDef) (bonus, mins int, applied, ok bool) {
	const maxDuration = 4 * time.Hour
	const stackDuration = 20 * time.Minute

	newTier := strengthBuffTier(spell.ID)
	curTier := strengthBuffTier(target.StrengthBuffID)
	curActive := target.StrengthBuffID > 0 && !target.StrengthBuffExpiry.IsZero() && time.Now().Before(target.StrengthBuffExpiry)

	// Lower-tier spell cast while a better buff is active → no effect
	if curActive && curTier > newTier {
		return 0, 0, false, false
	}

	bonusMap := map[int]int{207: 10, 208: 20, 209: 30}
	newBonus := bonusMap[spell.ID]

	// Same-tier spell cast while that buff is active → extend time only
	if curActive && curTier == newTier {
		newExpiry := target.StrengthBuffExpiry.Add(stackDuration)
		maxExpiry := time.Now().Add(maxDuration)
		if newExpiry.After(maxExpiry) {
			newExpiry = maxExpiry
		}
		target.StrengthBuffExpiry = newExpiry
		return newBonus, int(time.Until(target.StrengthBuffExpiry).Minutes()) + 1, false, true
	}

	// Remove any existing lower-tier buff before applying upgrade
	if curActive && curTier < newTier {
		target.Strength -= target.StrengthBuffBonus
	}

	target.Strength += newBonus
	target.StrengthBuffID = spell.ID
	target.StrengthBuffBonus = newBonus
	target.StrengthBuffExpiry = time.Now().Add(stackDuration)
	return newBonus, 20, true, true
}

func (e *GameEngine) castStrengthSpell(player *Player, spell *SpellDef, args []string) *CommandResult {
	// Resolve target: default self, otherwise find by name in room
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

	bonus, mins, applied, ok := applyStrengthBuff(target, spell)
	if !ok {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and begin to cast %s, but the target already has a better Strength spell in place.", spell.Name)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures, but the spell fades.", player.FirstName)},
		}
	}

	if !applied {
		if isSelf {
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You gesture and cast %s. Your strength pulsates with renewed energy! (%d minutes remaining)", spell.Name, mins)},
				RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
			}
		}
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s, extending their strength buff. (%d minutes remaining)", spell.Name, target.FirstName, mins)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, target.FirstName)},
			TargetName:    target.FirstName,
			TargetMsg:     []string{fmt.Sprintf("%s casts %s on you, renewing your strength. (%d minutes remaining)", player.FirstName, spell.Name, mins)},
		}
	}

	strengthFeelingMap := map[int]string{
		207: "stronger",
		208: "much stronger",
		209: "immensely stronger",
	}
	feeling := strengthFeelingMap[spell.ID]

	if isSelf {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s. You feel %s! (+%d STR, 20 minutes)", spell.Name, feeling, bonus)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
		}
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s.", spell.Name, target.FirstName)},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, target.FirstName)},
		TargetName:    target.FirstName,
		TargetMsg:     []string{fmt.Sprintf("%s casts %s on you. You feel %s! (+%d STR, 20 minutes)", player.FirstName, spell.Name, feeling, bonus)},
	}
}

// agilityBuffTier returns 0=none, 1=Agi I, 2=Agi II, 3=Agi III.
func agilityBuffTier(buffID int) int {
	switch buffID {
	case 513:
		return 1
	case 514:
		return 2
	case 515:
		return 3
	}
	return 0
}

// applyAgilityBuff mirrors applyStrengthBuff for the Agility I/II/III tier system.
func applyAgilityBuff(target *Player, spell *SpellDef) (bonus, mins int, applied, ok bool) {
	const maxDuration = 4 * time.Hour
	const stackDuration = 20 * time.Minute

	newTier := agilityBuffTier(spell.ID)
	curTier := agilityBuffTier(target.AgilityBuffID)
	curActive := target.AgilityBuffID > 0 && !target.AgilityBuffExpiry.IsZero() && time.Now().Before(target.AgilityBuffExpiry)

	if curActive && curTier > newTier {
		return 0, 0, false, false
	}

	bonusMap := map[int]int{513: 10, 514: 20, 515: 30}
	newBonus := bonusMap[spell.ID]

	if curActive && curTier == newTier {
		newExpiry := target.AgilityBuffExpiry.Add(stackDuration)
		maxExpiry := time.Now().Add(maxDuration)
		if newExpiry.After(maxExpiry) {
			newExpiry = maxExpiry
		}
		target.AgilityBuffExpiry = newExpiry
		return newBonus, int(time.Until(target.AgilityBuffExpiry).Minutes()) + 1, false, true
	}

	if curActive && curTier < newTier {
		target.Agility -= target.AgilityBuffBonus
	}

	target.Agility += newBonus
	target.AgilityBuffID = spell.ID
	target.AgilityBuffBonus = newBonus
	target.AgilityBuffExpiry = time.Now().Add(stackDuration)
	return newBonus, 20, true, true
}

// castAgilitySpell handles Agility I/II/III (513/514/515) as a tiered, timed buff —
// same 20-minutes-per-cast/extend/upgrade model as castStrengthSpell.
func (e *GameEngine) castAgilitySpell(player *Player, spell *SpellDef, args []string) *CommandResult {
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

	bonus, mins, applied, ok := applyAgilityBuff(target, spell)
	if !ok {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and begin to cast %s, but the target already has a better Agility spell in place.", spell.Name)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures, but the spell fades.", player.FirstName)},
		}
	}

	if !applied {
		if isSelf {
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You gesture and cast %s. Your reflexes sharpen with renewed energy! (%d minutes remaining)", spell.Name, mins)},
				RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
			}
		}
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s, extending their agility buff. (%d minutes remaining)", spell.Name, target.FirstName, mins)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, target.FirstName)},
			TargetName:    target.FirstName,
			TargetMsg:     []string{fmt.Sprintf("%s casts %s on you, renewing your agility. (%d minutes remaining)", player.FirstName, spell.Name, mins)},
		}
	}

	agilityFeelingMap := map[int]string{
		513: "more agile",
		514: "much more agile",
		515: "incredibly agile",
	}
	feeling := agilityFeelingMap[spell.ID]

	if isSelf {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s. You feel %s! (+%d AGI, 20 minutes)", spell.Name, feeling, bonus)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
		}
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s.", spell.Name, target.FirstName)},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, target.FirstName)},
		TargetName:    target.FirstName,
		TargetMsg:     []string{fmt.Sprintf("%s casts %s on you. You feel %s! (+%d AGI, 20 minutes)", player.FirstName, spell.Name, feeling, bonus)},
	}
}

func (e *GameEngine) castSlowSpell(ctx context.Context, player *Player, spell *SpellDef, args []string) *CommandResult {
	const duration = 20 * time.Minute
	const maxDuration = 4 * time.Hour

	target := player
	if len(args) > 0 {
		t := strings.ToLower(strings.Join(args, " "))
		if t != "me" && t != "myself" && t != "self" {
			found := e.findPlayerInRoom(player, t)
			if found == nil {
				return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' here.", strings.Join(args, " "))}}
			}
			target = found
		}
	}

	isSelf := target == player

	// If target has Haste, cancel it and return them to normal speed
	if !target.HasteExpiry.IsZero() && time.Now().Before(target.HasteExpiry) {
		target.HasteExpiry = time.Time{}
		e.SavePlayer(ctx, target)
		if isSelf {
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You gesture and cast %s on yourself. Your haste fades and you return to normal speed.", spell.Name)},
				RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on themselves.", player.FirstName, spell.Name)},
			}
		}
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s. Their haste fades away!", spell.Name, target.FirstName)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, target.FirstName)},
			TargetName:    target.FirstName,
			TargetMsg:     []string{fmt.Sprintf("%s casts %s on you. Your haste fades and you return to normal speed!", player.FirstName, spell.Name)},
		}
	}

	// Apply or extend Slow
	if !target.SlowExpiry.IsZero() && time.Now().Before(target.SlowExpiry) {
		target.SlowExpiry = target.SlowExpiry.Add(duration)
		if target.SlowExpiry.After(time.Now().Add(maxDuration)) {
			target.SlowExpiry = time.Now().Add(maxDuration)
		}
		mins := int(time.Until(target.SlowExpiry).Minutes()) + 1
		e.SavePlayer(ctx, target)
		if isSelf {
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You gesture and cast %s on yourself, extending the slowness. (%d minutes remaining)", spell.Name, mins)},
				RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on themselves.", player.FirstName, spell.Name)},
			}
		}
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s, extending their slowness. (%d minutes remaining)", spell.Name, target.FirstName, mins)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, target.FirstName)},
			TargetName:    target.FirstName,
			TargetMsg:     []string{fmt.Sprintf("%s casts %s on you. Everything feels even slower! (%d minutes remaining)", player.FirstName, spell.Name, mins)},
		}
	}

	target.SlowExpiry = time.Now().Add(duration)
	e.SavePlayer(ctx, target)
	if isSelf {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s on yourself. Everything seems to slow down!", spell.Name)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on themselves.", player.FirstName, spell.Name)},
		}
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s.", spell.Name, target.FirstName)},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, target.FirstName)},
		TargetName:    target.FirstName,
		TargetMsg:     []string{fmt.Sprintf("%s casts %s on you. Everything seems to slow down!", player.FirstName, spell.Name)},
	}
}

func (e *GameEngine) castHasteSpell(player *Player, spell *SpellDef, args []string) *CommandResult {
	const hasteDuration = 20 * time.Minute
	const hasteMaxDuration = 4 * time.Hour

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

	if !target.SlowExpiry.IsZero() && time.Now().Before(target.SlowExpiry) {
		target.SlowExpiry = time.Time{}
		if isSelf {
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You gesture and cast %s. The slow haze lifts and you return to normal speed.", spell.Name)},
				RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
			}
		}
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s. Their slow haze lifts!", spell.Name, target.FirstName)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, target.FirstName)},
			TargetName:    target.FirstName,
			TargetMsg:     []string{fmt.Sprintf("%s casts %s on you. The slow haze lifts and you return to normal speed!", player.FirstName, spell.Name)},
		}
	}

	if !target.HasteExpiry.IsZero() && time.Now().Before(target.HasteExpiry) {
		target.HasteExpiry = target.HasteExpiry.Add(hasteDuration)
		if target.HasteExpiry.After(time.Now().Add(hasteMaxDuration)) {
			target.HasteExpiry = time.Now().Add(hasteMaxDuration)
		}
		mins := int(time.Until(target.HasteExpiry).Minutes()) + 1
		if isSelf {
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You gesture and cast %s. Your speed is already heightened! (%d minutes remaining)", spell.Name, mins)},
				RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
			}
		}
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s, extending their haste. (%d minutes remaining)", spell.Name, target.FirstName, mins)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, target.FirstName)},
			TargetName:    target.FirstName,
			TargetMsg:     []string{fmt.Sprintf("%s casts %s on you. Your speed is already heightened! (%d minutes remaining)", player.FirstName, spell.Name, mins)},
		}
	}

	target.HasteExpiry = time.Now().Add(hasteDuration)
	if isSelf {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s. The world seems to slow down around you!", spell.Name)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
		}
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s.", spell.Name, target.FirstName)},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, target.FirstName)},
		TargetName:    target.FirstName,
		TargetMsg:     []string{fmt.Sprintf("%s casts %s on you. The world seems to slow down around you!", player.FirstName, spell.Name)},
	}
}

// castFlySpell handles Fly (spell 224) as a temporary flight buff.
// Initial cast: flight for 20 minutes. Recasting while already active extends
// the duration by 20 minutes up to a 4-hour cap.
func (e *GameEngine) castFlySpell(player *Player, spell *SpellDef) *CommandResult {
	const flyDuration = 20 * time.Minute
	const flyMaxDuration = 4 * time.Hour

	if !player.FlyExpiry.IsZero() && time.Now().Before(player.FlyExpiry) {
		player.FlyExpiry = player.FlyExpiry.Add(flyDuration)
		if player.FlyExpiry.After(time.Now().Add(flyMaxDuration)) {
			player.FlyExpiry = time.Now().Add(flyMaxDuration)
		}
		mins := int(time.Until(player.FlyExpiry).Minutes()) + 1
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture and cast %s. You are already aloft! (%d minutes remaining)", spell.Name, mins)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
		}
	}

	player.CanFly = true
	player.FlyExpiry = time.Now().Add(flyDuration)
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You gesture and cast %s. You rise into the air!", spell.Name)},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
	}
}

func elementalImmunityType(dmgType string) int {
	switch strings.ToLower(dmgType) {
	case "heat":
		return 3
	case "electric":
		return 4
	case "cold":
		return 5
	case "crushing":
		return 1
	default:
		return -1
	}
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

// castMysticKey attempts to magically unlock a locked item in the room.
// Success chance scales with Conjuration skill and Empathy, reduced by lock difficulty (Val1).
func (e *GameEngine) castMysticKey(player *Player, args []string) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}
	target := ""
	if len(args) > 0 {
		target = strings.ToLower(strings.Join(args, " "))
	}

	conjSkill := player.Skills[7] // Conjuration skill

	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil || ri.State != "LOCKED" {
			continue
		}
		name := e.getItemNounName(itemDef)
		if target != "" && !matchesTarget(name, target, e.getAdjName(ri.Adj1), e.getAdjName(ri.Adj2), e.getAdjName(ri.Adj3)) {
			continue
		}
		displayName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3, ri.Extend)
		lockDiff := ri.Val1
		chance := 40 + conjSkill*3 + player.Empathy/5 - lockDiff
		if chance < 10 {
			chance = 10
		}
		if player.IsGM {
			chance = 100
		}
		if rand.Intn(100) < chance {
			room.Items[i].State = "CLOSED"
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "CLOSED"})
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You gesture at %s... A shimmer of arcane light plays over the lock and it clicks open!", displayName)},
				RoomBroadcast: []string{fmt.Sprintf("%s gestures and a shimmer of light dances over %s.", player.FirstName, displayName)},
			}
		}
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You gesture at %s but the Mystic Key fades without opening it.", displayName)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures at %s but nothing seems to happen.", player.FirstName, displayName)},
		}
	}
	return &CommandResult{Messages: []string{"You don't see anything locked here."}}
}

// isLivingCreature returns false for undead, demons, and elementals that are immune
// to mind-affecting magic (Slumber, Fear, Charm).
func isLivingCreature(def *gameworld.MonsterDef) bool {
	if def.Discorporate {
		return false
	}
	n := strings.ToLower(def.Name)
	for _, w := range []string{"demon", "elemental", "golem", "skeleton", "zombie", "ghoul", "wraith", "specter", "spectre", "ghost", "lich", "vampire", "banshee"} {
		if strings.Contains(n, w) {
			return false
		}
	}
	return true
}

// autoTargetMonsterName returns the name of the player's current combat target monster, or "".
func (e *GameEngine) autoTargetMonsterName(player *Player) string {
	if player.CombatTarget == nil || !player.CombatTarget.IsMonster {
		return ""
	}
	e.monsterMgr.mu.RLock()
	defer e.monsterMgr.mu.RUnlock()
	for _, inst := range e.monsterMgr.instances {
		if inst.ID == player.CombatTarget.MonsterID && inst.Alive {
			if def := e.monsters[inst.DefNumber]; def != nil {
				return def.Name
			}
		}
	}
	return ""
}

func (e *GameEngine) castSlumberSpell(player *Player, spell *SpellDef, args []string) *CommandResult {
	targetName := strings.Join(args, " ")
	if targetName == "" {
		targetName = e.autoTargetMonsterName(player)
	}
	if targetName == "" {
		return &CommandResult{Messages: []string{fmt.Sprintf("Cast %s at what?", spell.Name)}}
	}

	inst, def := e.findMonsterInRoom(player, targetName)
	if inst == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' here.", targetName)}}
	}

	name := strings.ToLower(FormatMonsterName(def, e.monAdjs))
	article := articleFor(name, def.Unique)

	if !isLivingCreature(def) {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("The %s is unaffected -- sleep magic doesn't work on the unliving.", name)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures at %s%s, but it is unaffected.", player.FirstName, article, name)},
		}
	}

	// Body point cap per tier
	totalBody := def.Body + def.ExtraBody
	var bodyLimit int
	switch spell.ID {
	case 216:
		bodyLimit = 80
	case 217:
		bodyLimit = 200
	case 218:
		bodyLimit = 450
	}
	if totalBody > bodyLimit {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("The %s is too powerful to be affected by %s.", name, spell.Name)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures at %s%s, but it shrugs off the sleep magic.", player.FirstName, article, name)},
		}
	}

	if def.MagicResist > 0 && rand.Intn(100) < def.MagicResist {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("The %s resists your sleep magic!", name)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures at %s%s, but it resists!", player.FirstName, article, name)},
		}
	}

	e.monsterMgr.mu.Lock()
	for i := range e.monsterMgr.instances {
		if e.monsterMgr.instances[i].ID == inst.ID {
			e.monsterMgr.instances[i].Sleeping = true
			e.monsterMgr.instances[i].SleepExpiry = time.Now().Add(60 * time.Second)
			e.monsterMgr.instances[i].SleepStand = false
			e.monsterMgr.instances[i].Target = ""
			break
		}
	}
	e.monsterMgr.mu.Unlock()

	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You gesture and cast %s at %s%s!", spell.Name, article, name), fmt.Sprintf("The %s slumps into a deep sleep!", name)},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures at %s%s -- it slumps into a deep sleep!", player.FirstName, article, name)},
	}
}

func (e *GameEngine) castWebSpell(player *Player, spell *SpellDef, args []string) *CommandResult {
	targetName := strings.Join(args, " ")
	if targetName == "" {
		targetName = e.autoTargetMonsterName(player)
	}
	if targetName == "" {
		return &CommandResult{Messages: []string{"Cast Web at what?"}}
	}

	inst, def := e.findMonsterInRoom(player, targetName)
	if inst == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' here.", targetName)}}
	}

	name := strings.ToLower(FormatMonsterName(def, e.monAdjs))
	article := articleFor(name, def.Unique)

	// Discorporate/ethereal creatures can't be webbed
	if def.Discorporate {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("The webs pass right through the %s!", name)},
			RoomBroadcast: []string{fmt.Sprintf("%s casts Web at %s%s, but the webs pass through it!", player.FirstName, article, name)},
		}
	}

	if def.MagicResist > 0 && rand.Intn(100) < def.MagicResist {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("The %s resists the webs!", name)},
			RoomBroadcast: []string{fmt.Sprintf("%s casts Web at %s%s, but it resists!", player.FirstName, article, name)},
		}
	}

	e.monsterMgr.mu.Lock()
	for i := range e.monsterMgr.instances {
		if e.monsterMgr.instances[i].ID == inst.ID {
			e.monsterMgr.instances[i].Webbed = true
			e.monsterMgr.instances[i].WebExpiry = time.Now().Add(60 * time.Second)
			break
		}
	}
	e.monsterMgr.mu.Unlock()

	webbedMsg := fmt.Sprintf("%s%s is covered with strands of sticky webbing!", capArticle(article), name)
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You gesture and cast %s at %s%s!", spell.Name, article, name), webbedMsg},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s at %s%s!", player.FirstName, spell.Name, article, name), webbedMsg},
	}
}

func (e *GameEngine) castFearSpell(player *Player, spell *SpellDef, args []string) *CommandResult {
	targetName := strings.Join(args, " ")
	if targetName == "" {
		targetName = e.autoTargetMonsterName(player)
	}
	if targetName == "" {
		return &CommandResult{Messages: []string{"Cast Fear at what?"}}
	}

	inst, def := e.findMonsterInRoom(player, targetName)
	if inst == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' here.", targetName)}}
	}

	name := strings.ToLower(FormatMonsterName(def, e.monAdjs))
	article := articleFor(name, def.Unique)

	if !isLivingCreature(def) {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("The %s is unaffected -- fear magic doesn't work on the unliving.", name)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures at %s%s, but it is unaffected.", player.FirstName, article, name)},
		}
	}

	// Fear only works on weaker creatures (≤ 100 body)
	if def.Body+def.ExtraBody > 100 {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("The %s is too strong-willed to be affected by Fear.", name)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures at %s%s, but it is unaffected.", player.FirstName, article, name)},
		}
	}

	if def.MagicResist > 0 && rand.Intn(100) < def.MagicResist {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("The %s resists your fear magic!", name)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures at %s%s, but it resists!", player.FirstName, article, name)},
		}
	}

	e.monsterMgr.mu.Lock()
	for i := range e.monsterMgr.instances {
		if e.monsterMgr.instances[i].ID == inst.ID {
			e.monsterMgr.instances[i].Feared = true
			e.monsterMgr.instances[i].FearExpiry = time.Now().Add(60 * time.Second)
			break
		}
	}
	e.monsterMgr.mu.Unlock()

	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You gesture and cast %s at %s%s!", spell.Name, article, name), fmt.Sprintf("The %s is overcome with terror and flees!", name)},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures at %s%s -- it is overcome with terror!", player.FirstName, article, name)},
	}
}

func (e *GameEngine) castCharmSpell(player *Player, spell *SpellDef, args []string) *CommandResult {
	targetName := strings.Join(args, " ")
	if targetName == "" {
		targetName = e.autoTargetMonsterName(player)
	}
	if targetName == "" {
		return &CommandResult{Messages: []string{"Cast Charm at what?"}}
	}

	inst, def := e.findMonsterInRoom(player, targetName)
	if inst == nil {
		return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' here.", targetName)}}
	}

	name := strings.ToLower(FormatMonsterName(def, e.monAdjs))
	article := articleFor(name, def.Unique)

	if !isLivingCreature(def) {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("The %s is unaffected -- charm magic doesn't work on the unliving.", name)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures at %s%s, but it is unaffected.", player.FirstName, article, name)},
		}
	}

	// Charm only works on weaker creatures (≤ 150 body)
	if def.Body+def.ExtraBody > 150 {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("The %s is too strong-willed to be charmed.", name)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures at %s%s, but it is unaffected.", player.FirstName, article, name)},
		}
	}

	if def.MagicResist > 0 && rand.Intn(100) < def.MagicResist {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("The %s resists your charm magic!", name)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures at %s%s, but it resists!", player.FirstName, article, name)},
		}
	}

	e.monsterMgr.mu.Lock()
	for i := range e.monsterMgr.instances {
		if e.monsterMgr.instances[i].ID == inst.ID {
			e.monsterMgr.instances[i].Charmed = true
			e.monsterMgr.instances[i].CharmExpiry = time.Now().Add(60 * time.Second)
			e.monsterMgr.instances[i].CharmTarget = player.FirstName
			e.monsterMgr.instances[i].Target = ""
			break
		}
	}
	e.monsterMgr.mu.Unlock()

	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You gesture and cast %s at %s%s!", spell.Name, article, name), fmt.Sprintf("The %s regards you with friendly eyes.", name)},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures at %s%s -- it seems calmed.", player.FirstName, article, name)},
	}
}

// castDetectMagic handles spell 400 — Detect Magic.
// Examines a named item (inventory, worn, or in the room) for a stored spell (Val3) and
// reports the intensity of the magical glow based on the stored spell's level.
func (e *GameEngine) castDetectMagic(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Detect magic on what? Specify an item."}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	type candidate struct {
		item *InventoryItem
		def  *gameworld.ItemDef
	}

	matches := func(ii *InventoryItem) bool {
		def := e.items[ii.Archetype]
		if def == nil {
			return false
		}
		return matchesTarget(e.getItemNounName(def), target, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3))
	}

	// Search inventory, worn, wielded, off-hand, then room items — respecting ordinal skip.
	var found *candidate
	for i := range player.Inventory {
		if !matches(&player.Inventory[i]) {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}
		found = &candidate{&player.Inventory[i], e.items[player.Inventory[i].Archetype]}
		break
	}
	if found == nil {
		for i := range player.Worn {
			if !matches(&player.Worn[i]) {
				continue
			}
			if skip > 0 {
				skip--
				continue
			}
			found = &candidate{&player.Worn[i], e.items[player.Worn[i].Archetype]}
			break
		}
	}
	if found == nil && player.Wielded != nil && matches(player.Wielded) {
		if skip > 0 {
			skip--
		} else {
			found = &candidate{player.Wielded, e.items[player.Wielded.Archetype]}
		}
	}
	if found == nil && player.OffHand != nil && matches(player.OffHand) {
		if skip > 0 {
			skip--
		} else {
			found = &candidate{player.OffHand, e.items[player.OffHand.Archetype]}
		}
	}
	if found == nil {
		room := e.rooms[player.RoomNumber]
		if room != nil {
			for _, ri := range room.Items {
				def := e.items[ri.Archetype]
				if def == nil {
					continue
				}
				if !matchesTarget(e.getItemNounName(def), target, e.getAdjName(ri.Adj1), e.getAdjName(ri.Adj2), e.getAdjName(ri.Adj3)) {
					continue
				}
				if skip > 0 {
					skip--
					continue
				}
				tmp := InventoryItem{
					Archetype: ri.Archetype,
					Adj1: ri.Adj1, Adj2: ri.Adj2, Adj3: ri.Adj3,
					Val2: ri.Val2, Val3: ri.Val3, Val4: ri.Val4,
				}
				found = &candidate{&tmp, def}
				break
			}
		}
	}

	if found == nil {
		return &CommandResult{Messages: []string{"You don't see that here."}}
	}

	itemName := e.formatItemName(found.def, found.item.Adj1, found.item.Adj2, found.item.Adj3, found.item.Tail)

	if found.item.Val3 == 0 {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("%s reveals no magical aura.", itemName)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures at %s.", player.FirstName, itemName)},
		}
	}

	// Val3 = spell ID, Val2 = charges remaining. A spent item has the spell imprint
	// but no charges — it shows a faint residual aura.
	if found.item.Val2 == 0 {
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("%s reveals a faint magical residue — it has been completely drained of power.", itemName)},
			RoomBroadcast: []string{fmt.Sprintf("%s gestures at %s.", player.FirstName, itemName)},
		}
	}

	storedSpell := FindSpellByID(found.item.Val3)
	var glow string
	var spellLevel int
	if storedSpell != nil {
		spellLevel = storedSpell.Level
	}
	switch {
	case spellLevel >= 21:
		glow = "glows a brilliant blue"
	case spellLevel >= 11:
		glow = "glows a bright blue"
	case spellLevel >= 6:
		glow = "glows blue"
	default:
		glow = "glows a soft blue"
	}

	chargeWord := "charge"
	if found.item.Val2 != 1 {
		chargeWord = "charges"
	}
	msg := fmt.Sprintf("%s %s (%d %s remaining).", itemName, glow, found.item.Val2, chargeWord)
	return &CommandResult{
		Messages:      []string{msg},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures at %s, which %s.", player.FirstName, itemName, glow)},
	}
}
