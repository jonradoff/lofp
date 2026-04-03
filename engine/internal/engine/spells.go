package engine

import (
	"fmt"
	"strings"
)

// SpellDef defines a spell in the game.
type SpellDef struct {
	ID     int
	Name   string
	School string
	Level  int
}

// spellRegistry holds all defined spells.
var spellRegistry []SpellDef

func init() {
	// Conjuration (100-144)
	conj := []SpellDef{
		{100, "Flame Bolt", "Conjuration", 1}, {101, "Force Blade", "Conjuration", 3},
		{102, "Mystic Armor", "Conjuration", 5}, {103, "Lightning Bolt", "Conjuration", 7},
		{105, "Globe of Protection", "Conjuration", 15}, {106, "Summon Fire Elemental", "Conjuration", 12},
		{107, "Summon Air Elemental", "Conjuration", 12}, {108, "Summon Water Elemental", "Conjuration", 12},
		{109, "Summon Gargoyle", "Conjuration", 16}, {112, "Call Meteor", "Conjuration", 20},
		{113, "Light", "Conjuration", 1}, {114, "Mystic Key", "Conjuration", 2},
		{115, "Shockwave", "Conjuration", 4}, {116, "Thunder Call", "Conjuration", 21},
		{117, "Call Fire", "Conjuration", 8}, {118, "Flaming Sphere", "Conjuration", 13},
		{119, "Ice Bolt", "Conjuration", 3}, {120, "Frost Ray", "Conjuration", 6},
		{121, "Freezing Sphere", "Conjuration", 9}, {122, "Summon Familiar", "Conjuration", 2},
		{123, "Summon Earth Elemental", "Conjuration", 12}, {124, "Inferno Glyph", "Conjuration", 20},
		{125, "Thunder Glyph", "Conjuration", 10}, {126, "Ice Glyph", "Conjuration", 15},
		{127, "Web", "Conjuration", 10}, {130, "Mass Protection", "Conjuration", 23},
		{131, "Flaming Arrows", "Conjuration", 18}, {132, "Chain Lightning", "Conjuration", 23},
		{133, "Globe of Protection II", "Conjuration", 30}, {134, "Siryx's Terrible Tentacles", "Conjuration", 25},
		{135, "Storm Blade", "Conjuration", 24}, {136, "Inferno Blade", "Conjuration", 19},
		{137, "Winter Blade", "Conjuration", 22}, {138, "Energy Maelstrom", "Conjuration", 31},
		{139, "Sorcerous Summons I", "Conjuration", 20}, {140, "Sorcerous Summons II", "Conjuration", 35},
		{141, "Pyrotechnics", "Conjuration", 17}, {144, "Tindareth's Chaotic Summons", "Conjuration", 28},
	}
	// Enchantment (200-250)
	ench := []SpellDef{
		{200, "Fear", "Enchantment", 1}, {201, "Charm", "Enchantment", 3},
		{202, "Enchantment I", "Enchantment", 4}, {203, "Enchantment II", "Enchantment", 15},
		{204, "Enchantment III", "Enchantment", 30}, {205, "Command", "Enchantment", 6},
		{206, "Domination I", "Enchantment", 12}, {207, "Strength I", "Enchantment", 4},
		{208, "Strength II", "Enchantment", 8}, {209, "Strength III", "Enchantment", 16},
		{210, "Haste", "Enchantment", 5}, {211, "Slow", "Enchantment", 5},
		{212, "Mass Invisibility", "Enchantment", 25}, {213, "Bend Space I", "Enchantment", 17},
		{214, "Domination II", "Enchantment", 24}, {215, "Scry", "Enchantment", 10},
		{216, "Slumber I", "Enchantment", 2}, {217, "Slumber II", "Enchantment", 6},
		{218, "Slumber III", "Enchantment", 18}, {219, "Silence", "Enchantment", 7},
		{220, "Dancing Blade", "Enchantment", 1}, {221, "Dancing Sword", "Enchantment", 6},
		{222, "Bend Space II", "Enchantment", 23}, {224, "Fly", "Enchantment", 11},
		{225, "Invisibility", "Enchantment", 14}, {226, "Paranoia", "Enchantment", 3},
		{227, "Imprisonment Rune", "Enchantment", 13}, {228, "Identify", "Enchantment", 7},
		{229, "Wizard's Armor", "Enchantment", 9}, {230, "Disjunction", "Enchantment", 21},
		{231, "Imprison", "Enchantment", 19}, {232, "Mist Form", "Enchantment", 20},
		{234, "Spell Shield", "Enchantment", 13}, {235, "Cloak Mind", "Enchantment", 22},
		{243, "Charge Wand", "Enchantment", 26}, {244, "Enchant an Item", "Enchantment", 31},
		{245, "Slime Form", "Enchantment", 13}, {246, "Yshtarin's Confounding Translocation", "Enchantment", 29},
		{248, "Phantom Form", "Enchantment", 34},
	}
	// Necromancy (301-356)
	necro := []SpellDef{
		{301, "Turn Undead I", "Necromancy", 2}, {302, "Turn Undead II", "Necromancy", 8},
		{303, "Cure Poison", "Necromancy", 11}, {304, "Turn Undead III", "Necromancy", 16},
		{305, "Breath of Life", "Necromancy", 14}, {306, "Animate Skeleton", "Necromancy", 6},
		{307, "Animate Zombie", "Necromancy", 10}, {308, "Control Undead I", "Necromancy", 7},
		{309, "Control Undead II", "Necromancy", 13}, {310, "Control Undead III", "Necromancy", 21},
		{311, "Speak with Dead", "Necromancy", 3}, {312, "Wail of the Banshee", "Necromancy", 20},
		{313, "Body Destruction I", "Necromancy", 1}, {314, "Body Destruction II", "Necromancy", 5},
		{315, "Body Destruction III", "Necromancy", 10}, {316, "Body Restoration I", "Necromancy", 1},
		{317, "Body Restoration II", "Necromancy", 5}, {318, "Body Restoration III", "Necromancy", 10},
		{319, "Cure Disease", "Necromancy", 12}, {320, "Contagion", "Necromancy", 23},
		{321, "Poison", "Necromancy", 17}, {322, "Symbol of Death", "Necromancy", 25},
		{323, "Spectral Fist", "Necromancy", 3}, {326, "Spectral Shield", "Necromancy", 9},
		{334, "Invigoration I", "Necromancy", 2}, {335, "Invigoration II", "Necromancy", 9},
		{336, "Wight Animation", "Necromancy", 17}, {337, "Reconstruction", "Necromancy", 4},
		{338, "Unstun", "Necromancy", 9}, {339, "Destroy Undead I", "Necromancy", 3},
		{340, "Destroy Undead II", "Necromancy", 8}, {341, "Destroy Undead III", "Necromancy", 13},
		{343, "Regeneration", "Necromancy", 27}, {345, "Spectral Sword", "Necromancy", 7},
		{347, "Divine Blessing", "Necromancy", 10}, {351, "Wither Limb", "Necromancy", 24},
		{352, "Raise Undead", "Necromancy", 23}, {353, "Summon Spectral Warrior", "Necromancy", 32},
		{354, "Rorin's Fire", "Necromancy", 17},
	}
	// General (400-415)
	gen := []SpellDef{
		{400, "Detect Magic", "General", 1}, {401, "Dispel Lesser Magic", "General", 5},
		{403, "Mindlink", "General", 9}, {404, "Aura Sense", "General", 14},
		{405, "See Hidden", "General", 3}, {406, "Dispel Invisibility", "General", 8},
		{407, "Analyze Ore", "General", 3}, {408, "Truename", "General", 18},
	}
	// Druidic (500-538)
	druid := []SpellDef{
		{500, "Plant Snare", "Druidic", 4}, {501, "Call Storm", "Druidic", 23},
		{502, "Disperse Storm", "Druidic", 19}, {503, "Call Lightning", "Druidic", 17},
		{504, "Call Animal", "Druidic", 1}, {505, "Freedom", "Druidic", 9},
		{506, "Resist Weather", "Druidic", 3}, {507, "Heat Shield", "Druidic", 7},
		{508, "Cold Shield", "Druidic", 6}, {509, "Repel Plants", "Druidic", 10},
		{510, "Repel Plants and Webs", "Druidic", 18}, {511, "Carapace", "Druidic", 8},
		{512, "True Aim", "Druidic", 15}, {513, "Agility I", "Druidic", 4},
		{514, "Agility II", "Druidic", 11}, {515, "Agility III", "Druidic", 16},
		{516, "Wall of Thorns", "Druidic", 14}, {517, "Stick to Snake", "Druidic", 5},
		{518, "Claw Growth", "Druidic", 2}, {519, "Sunray", "Druidic", 13},
		{520, "Night Vision", "Druidic", 1}, {521, "Camouflage", "Druidic", 7},
		{522, "Insect Swarm", "Druidic", 25}, {523, "Earth Spike", "Druidic", 5},
		{524, "Earth Wave", "Druidic", 12}, {528, "Free Action", "Druidic", 20},
		{531, "Tree Door", "Druidic", 10}, {532, "Ride the Lightning", "Druidic", 34},
		{533, "Commune to Nature", "Druidic", 27}, {534, "Claws of the Elder Wolf", "Druidic", 21},
		{535, "Form Lock", "Druidic", 18}, {536, "Wolf Form", "Druidic", 26},
	}

	spellRegistry = append(spellRegistry, conj...)
	spellRegistry = append(spellRegistry, ench...)
	spellRegistry = append(spellRegistry, necro...)
	spellRegistry = append(spellRegistry, gen...)
	spellRegistry = append(spellRegistry, druid...)
}

// FindSpellByName finds a spell by prefix match on name.
func FindSpellByName(input string) *SpellDef {
	input = strings.ToLower(input)
	// Exact match first
	for i := range spellRegistry {
		if strings.ToLower(spellRegistry[i].Name) == input {
			return &spellRegistry[i]
		}
	}
	// Prefix match
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

// doCast handles the CAST command.
func (e *GameEngine) doCast(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Cast what spell?"}}
	}
	spellName := strings.ToLower(strings.Join(args, " "))
	spell := FindSpellByName(spellName)
	if spell == nil {
		return &CommandResult{Messages: []string{"You don't know that spell."}}
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You begin casting %s... [Spells are not yet active. Coming soon.]", spell.Name)},
		RoomBroadcast: []string{fmt.Sprintf("%s begins casting a spell.", player.FirstName)},
	}
}
