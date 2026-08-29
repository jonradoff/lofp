package engine

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
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

	StatusType StatID
	StatusMsg  string
}

// spellRegistry holds all defined spells.
var spellRegistry []SpellDef

func init() {
	// Conjuration (100-144)
	conj := []SpellDef{
		{ID: 100, Name: "Flame Bolt", School: "Conjuration", Level: 1, ManaCost: 3, CastTime: 3, Effect: "damage", DmgMin: 3, DmgMax: 12, DmgType: "heat"},
		{ID: 101, Name: "Force Blade", School: "Conjuration", Level: 3, ManaCost: 5, CastTime: 3, Effect: "damage", DmgMin: 5, DmgMax: 18, DmgType: ""},
		{ID: 102, Name: "Mystic Armor", School: "Conjuration", Level: 5, ManaCost: 8, CastTime: 3, Effect: "defense", DefBonus: 20, Duration: 30 * time.Minute, Family: "armor", StatusType: DefensiveBuff},
		{ID: 103, Name: "Lightning Bolt", School: "Conjuration", Level: 7, ManaCost: 10, CastTime: 3, Effect: "damage", DmgMin: 8, DmgMax: 30, DmgType: "electric"},
		{ID: 105, Name: "Globe of Protection", School: "Conjuration", Level: 15, ManaCost: 20, CastTime: 3, Effect: "defense", DefBonus: 50, Duration: 90 * time.Minute, Family: "armor", StatusType: DefensiveBuff},
		{ID: 106, Name: "Summon Fire Elemental", School: "Conjuration", Level: 12, ManaCost: 25, CastTime: 5, Effect: "utility"},
		{ID: 107, Name: "Summon Air Elemental", School: "Conjuration", Level: 12, ManaCost: 25, CastTime: 5, Effect: "utility"},
		{ID: 108, Name: "Summon Water Elemental", School: "Conjuration", Level: 12, ManaCost: 25, CastTime: 5, Effect: "utility"},
		{ID: 109, Name: "Summon Gargoyle", School: "Conjuration", Level: 16, ManaCost: 30, CastTime: 5, Effect: "utility"},
		{ID: 112, Name: "Call Meteor", School: "Conjuration", Level: 20, ManaCost: 30, CastTime: 4, Effect: "damage", DmgMin: 25, DmgMax: 60, DmgType: "heat"},
		{ID: 113, Name: "Light", School: "Conjuration", Level: 1, ManaCost: 2, CastTime: 2, Effect: "buff", Family: "light", Duration: 30 * time.Minute, StatusType: LightBuff, StatusMsg: "A soft light surrounds you."},
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
		{ID: 200, Name: "Fear", School: "Enchantment", Level: 1, ManaCost: 3, CastTime: 3, Effect: "utility"},
		{ID: 201, Name: "Charm", School: "Enchantment", Level: 3, ManaCost: 8, CastTime: 3, Effect: "utility"},
		{ID: 202, Name: "Enchantment I", School: "Enchantment", Level: 5, ManaCost: 10, CastTime: 4, Effect: "buff", Duration: 45 * time.Minute, Family: "enchantment"},
		{ID: 207, Name: "Strength I", School: "Enchantment", Level: 4, ManaCost: 6, CastTime: 3, Effect: "buff", DefBonus: 10, Duration: 30 * time.Minute, Family: "strength", StatusType: StatStrength},
		{ID: 208, Name: "Strength II", School: "Enchantment", Level: 8, ManaCost: 10, CastTime: 3, Effect: "buff", DefBonus: 20, Duration: 45 * time.Minute, Family: "strength", StatusType: StatStrength},
		{ID: 209, Name: "Strength III", School: "Enchantment", Level: 16, ManaCost: 18, CastTime: 3, Effect: "buff", DefBonus: 30, Duration: 60 * time.Minute, Family: "strength", StatusType: StatStrength},
		{ID: 210, Name: "Haste", School: "Enchantment", Level: 5, ManaCost: 8, CastTime: 3, Effect: "buff", Duration: 60 * time.Minute, Family: "haste"},
		{ID: 211, Name: "Slow", School: "Enchantment", Level: 5, ManaCost: 8, CastTime: 3, Effect: "utility", Duration: 60 * time.Minute, Family: "slow"},
		{ID: 216, Name: "Slumber I", School: "Enchantment", Level: 2, ManaCost: 4, CastTime: 3, Effect: "utility"},
		{ID: 219, Name: "Silence", School: "Enchantment", Level: 7, ManaCost: 10, CastTime: 3, Effect: "utility"},
		{ID: 224, Name: "Fly", School: "Enchantment", Level: 11, ManaCost: 15, CastTime: 3, Effect: "buff", Duration: 45 * time.Minute},
		{ID: 225, Name: "Invisibility", School: "Enchantment", Level: 14, ManaCost: 18, CastTime: 3, Effect: "buff", Duration: 45 * time.Minute},
		{ID: 228, Name: "Identify", School: "Enchantment", Level: 7, ManaCost: 5, CastTime: 3, Effect: "utility"},
		{ID: 229, Name: "Wizard's Armor", School: "Enchantment", Level: 9, ManaCost: 12, CastTime: 3, Effect: "defense", DefBonus: 15, Duration: 45 * time.Minute},
		{ID: 234, Name: "Spell Shield", School: "Enchantment", Level: 13, ManaCost: 15, CastTime: 3, Effect: "defense", DefBonus: 25, Duration: 45 * time.Minute},
		{ID: 235, Name: "Cloak Mind", School: "Enchantment", Level: 22, ManaCost: 25, CastTime: 3, Effect: "defense", DefBonus: 25, Duration: 45 * time.Minute},
	}
	// Necromancy (301-356)
	necro := []SpellDef{
		{ID: 301, Name: "Turn Undead I", School: "Necromancy", Level: 2, ManaCost: 4, CastTime: 3, Effect: "damage", DmgMin: 5, DmgMax: 15, DmgType: ""},
		{ID: 302, Name: "Turn Undead II", School: "Necromancy", Level: 8, ManaCost: 10, CastTime: 3, Effect: "damage", DmgMin: 10, DmgMax: 30, DmgType: ""},
		{ID: 313, Name: "Body Destruction I", School: "Necromancy", Level: 1, ManaCost: 3, CastTime: 3, Effect: "damage", DmgMin: 3, DmgMax: 10, DmgType: ""},
		{ID: 314, Name: "Body Destruction II", School: "Necromancy", Level: 5, ManaCost: 7, CastTime: 3, Effect: "damage", DmgMin: 6, DmgMax: 20, DmgType: ""},
		{ID: 315, Name: "Body Destruction III", School: "Necromancy", Level: 10, ManaCost: 14, CastTime: 3, Effect: "damage", DmgMin: 12, DmgMax: 35, DmgType: ""},
		{ID: 316, Name: "Body Restoration I", School: "Necromancy", Level: 1, ManaCost: 3, CastTime: 3, Effect: "heal", HealMin: 5, HealMax: 15},
		{ID: 317, Name: "Body Restoration II", School: "Necromancy", Level: 5, ManaCost: 7, CastTime: 3, Effect: "heal", HealMin: 10, HealMax: 30},
		{ID: 318, Name: "Body Restoration III", School: "Necromancy", Level: 10, ManaCost: 14, CastTime: 3, Effect: "heal", HealMin: 20, HealMax: 50},
		{ID: 323, Name: "Spectral Fist", School: "Necromancy", Level: 3, ManaCost: 5, CastTime: 3, Effect: "damage", DmgMin: 4, DmgMax: 14, DmgType: "crushing"},
		{ID: 326, Name: "Spectral Shield", School: "Necromancy", Level: 9, ManaCost: 12, CastTime: 3, Effect: "defense", DefBonus: 20, Duration: 45 * time.Minute, StatusType: DefensiveBuff},
		{ID: 334, Name: "Invigoration I", School: "Necromancy", Level: 2, ManaCost: 4, CastTime: 3, Effect: "heal", HealMin: 3, HealMax: 10},
		{ID: 335, Name: "Invigoration II", School: "Necromancy", Level: 9, ManaCost: 10, CastTime: 3, Effect: "heal", HealMin: 8, HealMax: 25},
		{ID: 337, Name: "Reconstruction", School: "Necromancy", Level: 4, ManaCost: 6, CastTime: 3, Effect: "heal", HealMin: 5, HealMax: 20},
		{ID: 338, Name: "Unstun", School: "Necromancy", Level: 9, ManaCost: 8, CastTime: 2, Effect: "utility"},
		{ID: 339, Name: "Destroy Undead I", School: "Necromancy", Level: 3, ManaCost: 5, CastTime: 3, Effect: "damage", DmgMin: 8, DmgMax: 20, DmgType: ""},
		{ID: 340, Name: "Destroy Undead II", School: "Necromancy", Level: 8, ManaCost: 12, CastTime: 3, Effect: "damage", DmgMin: 15, DmgMax: 40, DmgType: ""},
		{ID: 341, Name: "Destroy Undead III", School: "Necromancy", Level: 13, ManaCost: 20, CastTime: 3, Effect: "damage", DmgMin: 25, DmgMax: 60, DmgType: ""},
		{ID: 343, Name: "Regeneration", School: "Necromancy", Level: 27, ManaCost: 35, CastTime: 4, Effect: "heal", HealMin: 40, HealMax: 80},
		{ID: 345, Name: "Spectral Sword", School: "Necromancy", Level: 7, ManaCost: 10, CastTime: 3, Effect: "damage", DmgMin: 6, DmgMax: 22, DmgType: ""},
		{ID: 347, Name: "Divine Blessing", School: "Necromancy", Level: 10, ManaCost: 12, CastTime: 3, Effect: "buff", Duration: 45 * time.Minute},
		{ID: 354, Name: "Rorin's Fire", School: "Necromancy", Level: 17, ManaCost: 22, CastTime: 3, Effect: "damage", DmgMin: 15, DmgMax: 40, DmgType: "heat"},
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
		{ID: 303, Name: "Cure Disease", School: "General", Level: 12, ManaCost: 16, CastTime: 3, Effect: "heal", StatusType: RemoveDisease, StatusMsg: "A purple light flashes around %d."},
	}
	// Druidic (500-538)
	druid := []SpellDef{
		{ID: 500, Name: "Plant Snare", School: "Druidic", Level: 4, ManaCost: 6, CastTime: 3, Effect: "utility"},
		{ID: 505, Name: "Freedom", School: "Druidic", Level: 9, ManaCost: 12, CastTime: 3, Effect: "utility"},
		{ID: 507, Name: "Heat Shield", School: "Druidic", Level: 7, ManaCost: 10, CastTime: 3, Effect: "buff", Duration: 45 * time.Minute},
		{ID: 508, Name: "Cold Shield", School: "Druidic", Level: 6, ManaCost: 8, CastTime: 3, Effect: "buff", Duration: 45 * time.Minute},
		{ID: 511, Name: "Carapace", School: "Druidic", Level: 8, ManaCost: 10, CastTime: 3, Effect: "defense", DefBonus: 20, Duration: 45 * time.Minute},
		{ID: 512, Name: "True Aim", School: "Druidic", Level: 15, ManaCost: 18, CastTime: 3, Effect: "buff"},
		{ID: 513, Name: "Agility I", School: "Druidic", Level: 4, ManaCost: 6, CastTime: 3, Effect: "buff", DefBonus: 10, Duration: 30 * time.Minute, Family: "agility", StatusType: StatAgility},
		{ID: 514, Name: "Agility II", School: "Druidic", Level: 11, ManaCost: 12, CastTime: 3, Effect: "buff", DefBonus: 20, Duration: 45 * time.Minute, Family: "agility", StatusType: StatAgility},
		{ID: 515, Name: "Agility III", School: "Druidic", Level: 16, ManaCost: 20, CastTime: 3, Effect: "buff", DefBonus: 30, Duration: 60 * time.Minute, Family: "agility", StatusType: StatAgility},
		{ID: 519, Name: "Sunray", School: "Druidic", Level: 13, ManaCost: 18, CastTime: 3, Effect: "damage", DmgMin: 12, DmgMax: 35, DmgType: "heat"},
		{ID: 520, Name: "Night Vision", School: "Druidic", Level: 1, ManaCost: 2, CastTime: 2, Effect: "buff", Duration: 45 * time.Minute, Family: "nightvision", StatusType: NightVisionBuff, StatusMsg: "Your eyes adjust to the darkness."},
		{ID: 521, Name: "Camouflage", School: "Druidic", Level: 7, ManaCost: 8, CastTime: 3, Effect: "buff", Duration: 45 * time.Minute},
		{ID: 523, Name: "Earth Spike", School: "Druidic", Level: 5, ManaCost: 7, CastTime: 3, Effect: "damage", DmgMin: 5, DmgMax: 18, DmgType: "crushing"},
		{ID: 524, Name: "Earth Wave", School: "Druidic", Level: 12, ManaCost: 16, CastTime: 3, Effect: "damage", DmgMin: 10, DmgMax: 30, DmgType: "crushing"},
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

	spellName := strings.Join(args, " ")
	spell := FindSpellByName(spellName)
	if spell == nil {
		return &CommandResult{Messages: []string{"You don't know that spell."}}
	}
	if !player.KnownSpells[spell.ID] && !player.IsGM {
		return &CommandResult{Messages: []string{fmt.Sprintf("You haven't learned %s.", spell.Name)}}
	}
	if player.Mana < spell.ManaCost {
		return &CommandResult{Messages: []string{fmt.Sprintf("You don't have enough mana. (%s costs %d, you have %d)", spell.Name, spell.ManaCost, player.Mana)}}
	}

	player.PreparedSpell = spell.ID
	player.RoundTimeExpiry = time.Now().Add(time.Duration(spell.CastTime) * time.Second)

	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You begin preparing %s... (type CAST to release, or CAST <target>)", spell.Name)},
		RoomBroadcast: []string{fmt.Sprintf("%s begins preparing a spell.", player.FirstName)},
	}
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
		player.PreparedSpell = spell.ID
	}

	spell := FindSpellByID(player.PreparedSpell)
	if spell == nil {
		player.PreparedSpell = 0
		return &CommandResult{Messages: []string{"Your spell fizzles."}}
	}

	// Mana cost = spell level (from LEGENDS.DOC)
	manaCost := spell.Level
	if manaCost < 1 {
		manaCost = 1
	}
	if player.Mana < manaCost {
		player.PreparedSpell = 0
		return &CommandResult{Messages: []string{fmt.Sprintf("Not enough mana! (%s requires %d, you have %d)", spell.Name, manaCost, player.Mana)}}
	}

	// Check roundtime
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := player.RoundTimeExpiry.Sub(time.Now()).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You are still preparing... %.0f seconds remaining.", remaining+0.5)}}
	}

	// Deduct mana (cost = spell level)
	player.Mana -= manaCost
	player.PreparedSpell = 0

	// Spellcraft skill check (from LEGENDS.DOC):
	// Base 25% + EMP/10 + spellcraft*5%, max 95%.
	// Roll > 98 = fumble. Roll <= 2 = spectacular success (double effect).
	spellcraftSkill := player.Skills[23]
	castChance := 25 + player.EffectiveStat(StatEmpathy)/10 + spellcraftSkill*5
	if castChance > 95 {
		castChance = 95
	}
	if player.IsGM {
		castChance = 100
	}

	castRoll := rand.Intn(100) + 1
	if castRoll == 100 && !player.IsGM {
		// Extreme failure!
		player.RoundTimeExpiry = time.Now().Add(3 * time.Second)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("[Success: %d%%, Roll %d] Extreme failure! The spell backfires!", castChance, castRoll)},
			RoomBroadcast: []string{fmt.Sprintf("Magic begins to form around %s but then fizzles.", player.FirstName)},
		}
	}

	spectacularSuccess := castRoll == 1

	if castRoll > castChance && !player.IsGM {
		player.RoundTimeExpiry = time.Now().Add(2 * time.Second)
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
		result = e.castHealSpell(player, spell, args, true)
	case "defense": //todo: add defensive spell effects to player stats
		result = e.castStatusSpell(player, spell, args, true)
	case "buff":
		result = e.castStatusSpell(player, spell, args, true)
	default:
		result.Messages = []string{fmt.Sprintf("You gesture and cast %s.", spell.Name)}
		result.RoomBroadcast = []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)}
	}

	// Prepend success roll message
	result.Messages = append([]string{successMsg}, result.Messages...)

	player.RoundTimeExpiry = time.Now().Add(time.Duration(spell.CastTime) * time.Second)
	e.SavePlayer(ctx, player)

	return result
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
					target.FirstName,
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
	)

	// Build the spell's status message.
	statusMsg := spell.StatusMsg
	if strings.Contains(statusMsg, "%d") {
		statusMsg = fmt.Sprintf(statusMsg, spell.DefBonus)
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
					player.FirstName,
					spell.Name,
				),
			)
		} else {
			result.Messages = append(
				result.Messages,
				fmt.Sprintf(
					"You gesture at %s and cast %s.",
					target.FirstName,
					spell.Name,
				),
			)

			result.RoomBroadcast = append(
				result.RoomBroadcast,
				fmt.Sprintf(
					"%s gestures at %s and casts %s.",
					player.FirstName,
					target.FirstName,
					spell.Name,
				),
			)
		}
	}

	// Effect message is useful whether cast normally or triggered by a script/item.
	if statusMsg != "" {
		result.Messages = append(result.Messages, statusMsg)
	}

	return result
}

func (e *GameEngine) castDamageSpell(player *Player, spell *SpellDef, args []string, spectacular bool) *CommandResult {
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
	dmg := rand.Intn(spell.DmgMax-spell.DmgMin+1) + spell.DmgMin

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

	// Article for monster name ("a " prefix)
	article := "a "

	// Spell flavor text based on damage type
	flavorSelf := fmt.Sprintf("%s forms a bolt of energy and hurls it at %s%s!", player.FirstName, article, name)
	flavorDmg := fmt.Sprintf("%s %s to %s. [%d Damage]", damageSeverity(dmg), spellDmgNoun(spell.DmgType), randomBodyPart(def.BodyType), dmg)
	switch spell.DmgType {
	case "heat":
		flavorSelf = fmt.Sprintf("%s forms a ball of flame and hurls it at %s%s!", player.FirstName, article, name)
		flavorDmg = fmt.Sprintf("%s burn to %s. [%d Damage]", damageSeverity(dmg), randomBodyPart(def.BodyType), dmg)
	case "cold":
		flavorSelf = fmt.Sprintf("%s forms a freezing sphere from the air and hurls it at %s%s!", player.FirstName, article, name)
		flavorDmg = fmt.Sprintf("%s blast to %s. [%d Damage]", damageSeverity(dmg), randomBodyPart(def.BodyType), dmg)
	case "electric":
		flavorSelf = fmt.Sprintf("%s releases a bolt of lightning at %s%s!", player.FirstName, article, name)
		flavorDmg = fmt.Sprintf("%s shock to %s. [%d Damage]", damageSeverity(dmg), randomBodyPart(def.BodyType), dmg)
	case "crushing":
		flavorSelf = fmt.Sprintf("%s hurls a force blast at %s%s!", player.FirstName, article, name)
		flavorDmg = fmt.Sprintf("%s strike to %s. [%d Damage]", damageSeverity(dmg), randomBodyPart(def.BodyType), dmg)
	}

	killed := e.damageMonster(player, inst.ID, dmg)

	var msgs, roomMsgs []string
	msgs = append(msgs, fmt.Sprintf("You gesture at %s%s.", article, name))
	roomMsgs = append(roomMsgs, fmt.Sprintf("%s gestures at %s%s.", player.FirstName, article, name))
	msgs = append(msgs, flavorSelf)
	roomMsgs = append(roomMsgs, flavorSelf)
	msgs = append(msgs, flavorDmg)

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

func (e *GameEngine) castHealSpell(player *Player, spell *SpellDef, args []string, showCastMessage bool) *CommandResult {
	// Heal self by default, or target if specified
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
						fmt.Sprintf("You gesture and cast %s on yourself. The poison leaves your system.", spell.Name),
					},
					RoomBroadcast: []string{
						fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name),
					},
				}
			}

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf("You gesture and cast %s on %s. The poison leaves their system.", spell.Name, targetName),
				},
				RoomBroadcast: []string{
					fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, targetName),
				},
				TargetName: target.FirstName,
				TargetMsg: []string{
					fmt.Sprintf("%s casts %s on you. The poison leaves your system.", player.FirstName, spell.Name),
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
						fmt.Sprintf("You gesture and cast %s on yourself. The disease leaves your system.", spell.Name),
					},
					RoomBroadcast: []string{
						fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name),
					},
				}
			}

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf("You gesture and cast %s on %s. The disease leaves their system.", spell.Name, targetName),
				},
				RoomBroadcast: []string{
					fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, targetName),
				},
				TargetName: target.FirstName,
				TargetMsg: []string{
					fmt.Sprintf("%s casts %s on you. The disease leaves your system.", player.FirstName, spell.Name),
				},
			}
		}

	default:

		heal := rand.Intn(spell.HealMax-spell.HealMin+1) + spell.HealMin
		target.BodyPoints += heal
		if target.BodyPoints > target.MaxBodyPoints {
			target.BodyPoints = target.MaxBodyPoints
		}

		if showCastMessage {
			if target == player {
				return &CommandResult{
					Messages:      []string{fmt.Sprintf("You gesture and cast %s on yourself, healing %d body points. [BP: %d/%d]", spell.Name, heal, target.BodyPoints, target.MaxBodyPoints)},
					RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
				}
			}

			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You gesture and cast %s on %s, healing %d body points.", spell.Name, targetName, heal)},
				RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s on %s.", player.FirstName, spell.Name, targetName)},
				TargetName:    target.FirstName,
				TargetMsg:     []string{fmt.Sprintf("%s casts %s on you, healing %d body points. [BP: %d/%d]", player.FirstName, spell.Name, heal, target.BodyPoints, target.MaxBodyPoints)},
			}
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
				fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name),
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
		)

		result := &CommandResult{}

		if showCastMessage {
			result.Messages = append(
				result.Messages,
				fmt.Sprintf("You gesture and cast %s.", spell.Name),
			)

			result.RoomBroadcast = append(
				result.RoomBroadcast,
				fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name),
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
				RoomBroadcast: []string{fmt.Sprintf("A soft glow surrounds an item %s is holding.", player.FirstName)},
			}
		}
		return &CommandResult{Messages: []string{"You don't have a weapon matching that."}}

		/*case 207, 208, 209: // Strength I-III
			player.ApplyStatEffect(
				spell.ID,
				EffectSourceSpell,
				spell.StatusType,
				spell.DefBonus,
				buffDuration,
			)

			msg = fmt.Sprintf("You gesture and cast %s. Your strength increases! (+%d STR)", spell.Name, spell.DefBonus)

		case 513, 514, 515: // Agility I-III
			player.ApplyStatEffect(
				spell.ID,
				EffectSourceSpell,
				StatAgility,
				spell.DefBonus,
				buffDuration,
			)
		*/
		msg = fmt.Sprintf("You gesture and cast %s. Your agility increases! (+%d AGI)", spell.Name, spell.DefBonus)
	}

	return &CommandResult{
		Messages:      []string{msg},
		RoomBroadcast: []string{fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name)},
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
				fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name),
			},
		}
	}

	player.ApplyStatEffect(
		spell.ID,
		EffectSourceSpell,
		DefensiveBuff,
		spell.DefBonus,
		buffDuration,
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
			fmt.Sprintf("%s gestures and casts %s.", player.FirstName, spell.Name),
		},
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

func (e *GameEngine) buildCastResult(
	player *Player,
	target *Player,
	spell *SpellDef,
	selfMsg string,
	targetMsg string,
) *CommandResult {

	if target == player {
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You gesture and cast %s on yourself. %s",
					spell.Name,
					selfMsg,
				),
			},
			RoomBroadcast: []string{
				fmt.Sprintf(
					"%s gestures and casts %s.",
					player.FirstName,
					spell.Name,
				),
			},
		}
	}

	return &CommandResult{
		Messages: []string{
			fmt.Sprintf(
				"You gesture and cast %s on %s. %s",
				spell.Name,
				target.FirstName,
				targetMsg,
			),
		},
		RoomBroadcast: []string{
			fmt.Sprintf(
				"%s gestures and casts %s on %s.",
				player.FirstName,
				spell.Name,
				target.FirstName,
			),
		},
		TargetName: target.FirstName,
		TargetMsg: []string{
			fmt.Sprintf(
				"%s casts %s on you. %s",
				player.FirstName,
				spell.Name,
				selfMsg,
			),
		},
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
