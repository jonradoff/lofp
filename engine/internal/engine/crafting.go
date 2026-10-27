package engine

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/metavert-io/lofp/internal/gameworld"
)

type mineOreDef struct {
	Archetype int
	Adj       int
	Grade     string
	Value     int
	Rarity    int
}

var mineOreTable = []mineOreDef{
	// Grade C
	{1369, 893, "C", 600, 0}, // graphite
	{1369, 68, "C", 1000, 0}, // copper
	{1369, 744, "C", 200, 0}, // tin
	{1369, 383, "C", 800, 1}, // diamontine

	// Grade B
	{1369, 68, "B", 1200, 0},  // copper
	{1369, 292, "B", 4000, 0}, // silver
	{1369, 328, "B", 2000, 0}, // titanium
	{1369, 109, "B", 5000, 5}, // elkyri
	{1369, 918, "B", 800, 2},  // bekk-ite
	{1369, 893, "B", 1000, 0}, // graphite
	{1369, 109, "B", 7000, 8}, // elkyri
	{1369, 256, "B", 5000, 8}, // randar

	// Grade A
	{1369, 147, "A", 800, 0},    // gold
	{1369, 292, "A", 800, 0},    // silver
	{1369, 563, "A", 1000, 0},   // platinum
	{1369, 109, "A", 10000, 15}, // elkyri
	{1369, 256, "A", 7500, 15},  // randar
	{1369, 918, "A", 1000, 5},   // bekk-ite
	{1369, 1130, "A", 1600, 10}, // albescent
	{1369, 643, "A", 1600, 10},  // shadowsteel

	// Special iron ore archetype
	{1371, 169, "C", 400, 0}, // iron
}

// ---- MINING ----
/*
func (e *GameEngine) doMineReal(ctx context.Context, player *Player) *CommandResult {

	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := int(time.Until(player.RoundTimeExpiry).Seconds()) + 1

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("[Wait %d seconds...]", remaining),
			},
			PlayerState: player,
		}
	}

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't mine here."}}
	}

	// Determine mine grade
	grade := ""
	if containsModifier(room.Modifiers, "MINEA") {
		grade = "A"
	} else if containsModifier(room.Modifiers, "MINEB") {
		grade = "B"
	} else if containsModifier(room.Modifiers, "MINEC") {
		grade = "C"
	}
	if grade == "" {
		return &CommandResult{Messages: []string{"There is nothing to mine here."}}
	}

	// Check for mining tool (wielded or in inventory)
	isMiningTool := func(def *gameworld.ItemDef) bool {
		if def == nil {
			return false
		}
		noun := strings.ToLower(e.nouns[def.NameID])
		return noun == "pickaxe" || noun == "pick-axe" || noun == "hammer" || noun == "shovel" || def.Type == "MINETOOL"
	}
	hasTool := false
	if player.Wielded != nil && isMiningTool(e.items[player.Wielded.Archetype]) {
		hasTool = true
	}
	if !hasTool {
		for _, ii := range player.Inventory {
			if isMiningTool(e.items[ii.Archetype]) {
				hasTool = true
				break
			}
		}
	}
	if !hasTool {
		return &CommandResult{Messages: []string{"You need a mining tool (pickaxe, hammer, or shovel) to mine."}}
	}

	// Mining skill check
	miningSkill := player.Skills[35]
	if miningSkill < 1 {
		return &CommandResult{Messages: []string{"You have no training in Mining."}}
	}

	// A legitimate mining attempt takes 10 seconds,
	// whether or not useful ore is found.
	player.RoundTimeExpiry = time.Now().Add(10 * time.Second)

	// Success chance: base 30% + mining*5 + STR/10
	chance := 30 + miningSkill*5 + player.EffectiveStat(StatStrength)/10
	if chance > 90 {
		chance = 90
	}

	player.Fatigue -= 2
	if player.Fatigue < 0 {
		player.Fatigue = 0
	}

	if rand.Intn(100) >= chance {
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages: []string{
				"You swing at the rock face but find nothing useful.",
				"[Round: 10 sec]",
			},
			RoomBroadcast: []string{fmt.Sprintf("%s swings a mining tool at the rock.", player.FirstName)},
		}
	}

	// Pick ore type and metal adjective based on grade
	// Grade A: iron/steel metals, Grade B: copper/bronze, Grade C: tin/copper
	type metalChoice struct {
		adj    int
		name   string
		weight int
	}
	var metals []metalChoice
	switch grade {
	case "A":
		metals = []metalChoice{
			{e.adjByName("iron"), "iron", 40},
			{e.adjByName("steel"), "steel", 20},
			{e.adjByName("bronze"), "bronze", 25},
			{e.adjByName("copper"), "copper", 15},
		}
	case "B":
		metals = []metalChoice{
			{e.adjByName("copper"), "copper", 40},
			{e.adjByName("bronze"), "bronze", 30},
			{e.adjByName("iron"), "iron", 20},
			{e.adjByName("tin"), "tin", 10},
		}
	default: // C
		metals = []metalChoice{
			{e.adjByName("copper"), "copper", 40},
			{e.adjByName("tin"), "tin", 35},
			{e.adjByName("iron"), "iron", 25},
		}
	}
	// Weighted random selection
	totalWeight := 0
	for _, m := range metals {
		totalWeight += m.weight
	}
	pick := rand.Intn(totalWeight)
	chosenAdj := 0
	for _, m := range metals {
		pick -= m.weight
		if pick < 0 {
			chosenAdj = m.adj
			break
		}
	}

	// Find ore item archetype
	oreArch := 1369
	oreDef := e.items[oreArch]
	if oreDef == nil {
		for num, def := range e.items {
			if def.Type == "ORE" {
				oreArch = num
				oreDef = def
				break
			}
		}
	}

	if oreDef == nil {
		return &CommandResult{Messages: []string{
			"You chip away at the rock but find nothing.",
			"[Round: 10 sec]",
		}, PlayerState: player}
	}

	// Purity based on grade: A=50-100, B=30-70, C=10-40
	purity := 0
	switch grade {
	case "A":
		purity = 50 + rand.Intn(51)
	case "B":
		purity = 30 + rand.Intn(41)
	case "C":
		purity = 10 + rand.Intn(31)
	}

	ore := InventoryItem{
		Archetype: oreArch,
		Adj1:      chosenAdj, // metal type adjective
		Val3:      purity,
	}
	player.Inventory = append(player.Inventory, ore)

	// Successful mining earns experience.
	xpMsgs := e.awardExperience(player, 50)

	e.SavePlayer(ctx, player)

	qualityDesc := "poor"
	if purity > 70 {
		qualityDesc = "excellent"
	} else if purity > 50 {
		qualityDesc = "good"
	} else if purity > 30 {
		qualityDesc = "fair"
	}

	displayName := e.formatItemName(
		oreDef,
		ore.Adj1,
		ore.Adj2,
		ore.Adj3,
	)

	messages := []string{
		fmt.Sprintf(
			"You chip away at the rock and extract some %s looking %s!",
			qualityDesc,
			displayName,
		),
	}

	messages = append(messages, xpMsgs...)

	messages = append(
		messages,
		"[Round: 10 sec]",
	)

	return &CommandResult{
		Messages: messages,
		RoomBroadcast: []string{
			fmt.Sprintf(
				"%s mines some ore from the rock.",
				player.FirstName,
			),
		},
		PlayerState: player,
	}
}
*/

func (e *GameEngine) doMineReal(ctx context.Context, player *Player) *CommandResult {

	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := int(time.Until(player.RoundTimeExpiry).Seconds()) + 1

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("[Wait %d seconds...]", remaining),
			},
			PlayerState: player,
		}
	}

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{
			Messages: []string{"You can't mine here."},
		}
	}

	// ------------------------------------------------------------
	// DETERMINE MINE GRADE
	//
	// MINEA can produce A, B, or C.
	// MINEB can produce B or C.
	// MINEC can produce C only.
	// ------------------------------------------------------------

	mineGrade := ""

	if containsModifier(room.Modifiers, "MINEA") {
		mineGrade = "A"
	} else if containsModifier(room.Modifiers, "MINEB") {
		mineGrade = "B"
	} else if containsModifier(room.Modifiers, "MINEC") {
		mineGrade = "C"
	}

	if mineGrade == "" {
		return &CommandResult{
			Messages: []string{"There is nothing to mine here."},
		}
	}

	// ------------------------------------------------------------
	// CHECK FOR MINING TOOL
	// ------------------------------------------------------------

	isMiningTool := func(def *gameworld.ItemDef) bool {
		if def == nil {
			return false
		}

		noun := strings.ToLower(e.nouns[def.NameID])

		return noun == "pickaxe" ||
			noun == "pick-axe" ||
			noun == "hammer" ||
			noun == "shovel" ||
			def.Type == "MINETOOL"
	}

	hasTool := false

	if player.Wielded != nil {
		if isMiningTool(e.items[player.Wielded.Archetype]) {
			hasTool = true
		}
	}

	if !hasTool {
		for _, ii := range player.Inventory {
			if isMiningTool(e.items[ii.Archetype]) {
				hasTool = true
				break
			}
		}
	}

	if !hasTool {
		return &CommandResult{
			Messages: []string{
				"You need a mining tool (pickaxe, hammer, or shovel) to mine.",
			},
		}
	}

	// ------------------------------------------------------------
	// MINING SKILL
	// ------------------------------------------------------------

	miningSkill := player.Skills[35]

	if miningSkill < 1 {
		return &CommandResult{
			Messages: []string{
				"You have no training in Mining.",
			},
		}
	}

	// Every legitimate mining attempt takes 10 seconds.
	player.RoundTimeExpiry = time.Now().Add(10 * time.Second)

	// Success chance is kept in its own helper so combat,
	// fatigue, tools, room difficulty, etc. can be added later.
	chance := e.miningSuccessChance(player)

	// Mining is tiring.
	player.Fatigue -= 2
	if player.Fatigue < 0 {
		player.Fatigue = 0
	}

	// ------------------------------------------------------------
	// MINING SUCCESS CHECK
	// ------------------------------------------------------------

	if rand.Intn(100) >= chance {
		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages: []string{
				"You swing at the rock face but find nothing useful.",
				"[Round: 10 sec]",
			},
			RoomBroadcast: []string{
				fmt.Sprintf(
					"%s swings a mining tool at the rock.",
					player.FirstName,
				),
			},
			PlayerState: player,
		}
	}

	// ------------------------------------------------------------
	// SELECT A/B/C GRADE
	//
	// rollMineGrade() currently contains:
	//
	// MINEA: A 3%, B 22%, C 75%
	// MINEB: B 20%, C 80%
	// MINEC: C 100%
	//
	// Keeping this separate makes those numbers easy to tune.
	// ------------------------------------------------------------

	selectedGrade := rollMineGrade(mineGrade)

	// ------------------------------------------------------------
	// BUILD THE MINEDEF POOL
	//
	// Only MINEDEF entries belonging to the selected grade are
	// eligible.
	//
	// Within that grade, copper value is used as reverse rarity:
	//
	//     weight = 1 / sqrt(value)
	//
	// High-value results are therefore less common.
	//
	// Duplicate MINEDEF entries remain separate entries.
	// ------------------------------------------------------------

	type mineChoice struct {
		def    gameworld.MineDef
		weight float64
	}

	var choices []mineChoice
	totalWeight := 0.0

	for _, md := range e.mineDefs {

		if !strings.EqualFold(md.Grade, selectedGrade) {
			continue
		}

		value := md.Value
		if value <= 0 {
			value = 1
		}

		weight := 1.0 / math.Sqrt(float64(value))

		choices = append(
			choices,
			mineChoice{
				def:    md,
				weight: weight,
			},
		)

		totalWeight += weight
	}

	if len(choices) == 0 || totalWeight <= 0 {
		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages: []string{
				"You chip away at the rock but find nothing useful.",
				"[Round: 10 sec]",
			},
			PlayerState: player,
		}
	}

	// ------------------------------------------------------------
	// SELECT A MINEDEF
	// ------------------------------------------------------------

	pick := rand.Float64() * totalWeight

	chosen := choices[len(choices)-1].def

	for _, choice := range choices {
		pick -= choice.weight

		if pick <= 0 {
			chosen = choice.def
			break
		}
	}

	itemDef := e.items[chosen.ItemNum]

	if itemDef == nil {
		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages: []string{
				"You chip away at the rock but find nothing useful.",
				"[Round: 10 sec]",
			},
			PlayerState: player,
		}
	}

	// ------------------------------------------------------------
	// CREATE THE MINED ITEM
	//
	// ORE:
	//   Adj1 = MINEDEF adjective
	//   Val2 = MINEDEF Val2
	//   Val3 = purity
	//
	// GEM:
	//   createGemFromMineDef() handles:
	//   - size
	//   - quality
	//   - adjusted value
	//   - MINEDEF variant adjective
	// ------------------------------------------------------------

	var minedItem InventoryItem
	purity := 0

	if itemDef.Type == "ORE" {

		// Ore Val3 is purity.
		//
		// Legacy documentation defines this as the percentage
		// chance of successfully refining useful metal from it.
		//
		// We have not found the original purity-generation formula,
		// so this is currently a 1-100 roll.
		purity = rand.Intn(100) + 1

		minedItem = InventoryItem{
			Archetype: chosen.ItemNum,
			Val2:      chosen.Val2,
			Val3:      purity,
		}

		if chosen.AdjNum > 0 {
			minedItem.Adj1 = chosen.AdjNum
		}

	} else {

		// Non-ORE MINEDEF entries are gems.
		// Use the exact same size/quality/value system used
		// elsewhere for generated gems.
		minedItem = e.createGemFromMineDef(chosen)
	}

	// ------------------------------------------------------------
	// GIVE ITEM TO PLAYER
	// ------------------------------------------------------------

	player.Inventory = append(
		player.Inventory,
		minedItem,
	)

	// Successful mining earns experience.
	xpMsgs := e.awardExperience(player, 50)

	e.SavePlayer(ctx, player)

	// ------------------------------------------------------------
	// BUILD DISPLAY NAME
	// ------------------------------------------------------------

	displayName := e.formatItemName(
		itemDef,
		minedItem.Adj1,
		minedItem.Adj2,
		minedItem.Adj3,
	)

	var findMessage string

	if itemDef.Type == "ORE" {

		// Ore gets a human-readable purity description.
		qualityDesc := "poor"

		switch {
		case purity > 70:
			qualityDesc = "excellent"

		case purity > 50:
			qualityDesc = "good"

		case purity > 30:
			qualityDesc = "fair"
		}

		findMessage = fmt.Sprintf(
			"You chip away at the rock and extract %s of %s quality!",
			displayName,
			qualityDesc,
		)

	} else {

		// Gem size and quality are already part of its adjectives,
		// e.g.:
		//
		// a large chipped onyx
		// a tiny flawless alexandrite
		// a huge faceted aquamarine
		findMessage = fmt.Sprintf(
			"You chip away at the rock and extract %s!",
			displayName,
		)
	}

	// ------------------------------------------------------------
	// RESULT
	// ------------------------------------------------------------

	messages := []string{
		findMessage,
	}

	messages = append(
		messages,
		xpMsgs...,
	)

	messages = append(
		messages,
		"[Round: 10 sec]",
	)

	return &CommandResult{
		Messages: messages,

		RoomBroadcast: []string{
			fmt.Sprintf(
				"%s mines something from the rock.",
				player.FirstName,
			),
		},

		PlayerState: player,
	}
}

func (e *GameEngine) miningSuccessChance(player *Player) int {
	if player == nil {
		return 0
	}

	miningSkill := player.Skills[35]

	// Current mining success formula.
	chance := 30 +
		miningSkill*5 +
		player.EffectiveStat(StatStrength)/10

	// Future modifiers go here.
	//
	// For example:
	//
	// if player.CombatTarget != nil {
	//     chance -= 25
	// }
	//
	// We can also add modifiers here later for:
	// fatigue, tool quality, room difficulty, injuries, etc.

	if chance < 5 {
		chance = 5
	}

	if chance > 90 {
		chance = 90
	}

	return chance
}

func rollMineGrade(mineGrade string) string {
	roll := rand.Intn(100) + 1

	switch mineGrade {

	case "A":
		// MINEA
		//
		// A:  3%
		// B: 22%
		// C: 75%
		switch {
		case roll <= 3:
			return "A"

		case roll <= 25:
			return "B"

		default:
			return "C"
		}

	case "B":
		// MINEB
		//
		// B: 20%
		// C: 80%
		if roll <= 20 {
			return "B"
		}

		return "C"

	default:
		// MINEC
		//
		// C: 100%
		return "C"
	}
}

func (e *GameEngine) createGemFromMineDef(md gameworld.MineDef) InventoryItem {
	// Size:
	// tiny 15%, small 20%, normal 35%, large 20%, huge 10%
	sizeAdj := 0

	switch roll := rand.Intn(100); {
	case roll < 15:
		sizeAdj = 327 // tiny
	case roll < 35:
		sizeAdj = 294 // small
	case roll < 70:
		// normal
	case roll < 90:
		sizeAdj = 178 // large
	default:
		sizeAdj = 163 // huge
	}

	// Quality:
	// damaged 8%, chipped 12%, normal 45%, polished 15%,
	// faceted 10%, brilliant 7%, flawless 3%
	qualityAdj := 0

	switch roll := rand.Intn(100); {
	case roll < 8:
		qualityAdj = 83 // damaged
	case roll < 20:
		qualityAdj = 53 // chipped
	case roll < 65:
		// normal
	case roll < 80:
		qualityAdj = 241 // polished
	case roll < 90:
		qualityAdj = 118 // faceted
	case roll < 97:
		qualityAdj = 37 // brilliant
	default:
		qualityAdj = 129 // flawless
	}

	adjustedValue := gemValue(
		md.Value,
		sizeAdj,
		qualityAdj,
	)

	item := InventoryItem{
		Archetype: md.ItemNum,
		Val1:      adjustedValue,
		Val2:      md.Val2,
		Adj1:      sizeAdj,
		Adj2:      qualityAdj,
	}

	// MINEDEF variant: fire, black, blue, star, etc.
	if md.AdjNum > 0 {
		item.Adj3 = md.AdjNum
	}

	return item
}

// ---- SMELTING ----

func (e *GameEngine) doSmelt(ctx context.Context, player *Player, args []string) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil || !containsModifier(room.Modifiers, "FORGE") {
		return &CommandResult{
			Messages: []string{"You need to be at a forge to smelt ore."},
		}
	}

	// Respect existing roundtime.
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := int(time.Until(player.RoundTimeExpiry).Seconds()) + 1

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("[Wait %d seconds...]", remaining),
			},
			PlayerState: player,
		}
	}

	smithSkill := player.Skills[8]
	if smithSkill < 1 {
		return &CommandResult{
			Messages: []string{"You have no training in Weaponsmithing."},
		}
	}

	// Support an optional ordinal at the end:
	//   SMELT ORE
	//   SMELT ORE 2
	//   SMELT IRON ORE
	//   SMELT COPPER ORE 2
	ordinal := 1

	if len(args) > 1 {
		if n, err := strconv.Atoi(args[len(args)-1]); err == nil && n > 0 {
			ordinal = n
			args = args[:len(args)-1]
		}
	}

	// No target means simply SMELT ORE.
	target := "ore"
	if len(args) > 0 {
		target = strings.ToLower(strings.Join(args, " "))
	}

	matchCount := 0

	for i := range player.Inventory {
		ii := &player.Inventory[i]

		def := e.items[ii.Archetype]
		if def == nil || def.Type != "ORE" {
			continue
		}

		nounName := strings.ToLower(e.getItemNounName(def))

		fullName := strings.ToLower(
			e.formatItemName(def, ii.Adj1, ii.Adj2, ii.Adj3),
		)

		// Strip articles/quantity words so:
		// "copper ore" matches "some copper ore".
		searchName := fullName
		searchName = strings.TrimPrefix(searchName, "a ")
		searchName = strings.TrimPrefix(searchName, "an ")
		searchName = strings.TrimPrefix(searchName, "the ")
		searchName = strings.TrimPrefix(searchName, "some ")

		// Match either the basic noun ("ore") or the complete
		// formatted name ("iron ore", "copper ore", etc.).
		if !strings.HasPrefix(nounName, target) &&
			!strings.HasPrefix(searchName, target) {
			continue
		}

		matchCount++
		if matchCount != ordinal {
			continue
		}

		const rtSec = 10

		// A real smelting attempt takes 10 seconds,
		// whether it succeeds or fails.
		player.RoundTimeExpiry = time.Now().Add(
			time.Duration(rtSec) * time.Second,
		)
		player.RoundTime = rtSec

		// VAL3 is ore purity and therefore the base chance
		// of successfully refining this particular piece of ore.
		purity := ii.Val3
		if purity <= 0 {
			purity = 30
		}

		// Save the properties we need before removing the ore
		// from inventory.
		outputArch := def.Parameter2
		adj1 := ii.Adj1
		val2 := ii.Val2

		if outputArch <= 0 {
			outputArch = 1370 // generic metal fallback
		}

		// The ore is consumed whether smelting succeeds or fails.
		player.Inventory = append(
			player.Inventory[:i],
			player.Inventory[i+1:]...,
		)

		// Purity + Weaponsmithing determine success.
		smeltChance := purity + smithSkill*2

		if smeltChance > 95 {
			smeltChance = 95
		}

		if rand.Intn(100) >= smeltChance {
			e.SavePlayer(ctx, player)

			return &CommandResult{
				Messages: []string{
					"You heat the ore in the forge, but it crumbles to useless slag.",
					fmt.Sprintf("[Round: %d sec]", rtSec),
				},
				RoomBroadcast: []string{
					fmt.Sprintf(
						"%s works at the forge.",
						player.FirstName,
					),
				},
				PlayerState: player,
			}
		}

		// Successful refinement.
		outputDef := e.items[outputArch]
		if outputDef == nil {
			e.SavePlayer(ctx, player)

			return &CommandResult{
				Messages: []string{
					"The ore refines but produces nothing useful.",
					fmt.Sprintf("[Round: %d sec]", rtSec),
				},
				PlayerState: player,
			}
		}

		material := InventoryItem{
			Archetype: outputArch,
			Adj1:      adj1, // preserve metal type
			Val2:      val2, // preserve material properties
		}

		player.Inventory = append(
			player.Inventory,
			material,
		)

		// Successful smelting earns experience.
		xpMsgs := e.awardExperience(player, 50)

		e.SavePlayer(ctx, player)

		matName := e.formatItemName(
			outputDef,
			material.Adj1,
			material.Adj2,
			material.Adj3,
		)

		// formatItemName already supplies "some" where appropriate.
		messages := []string{
			fmt.Sprintf(
				"You smelt the ore in the forge and produce %s!",
				matName,
			),
		}

		messages = append(messages, xpMsgs...)

		messages = append(
			messages,
			fmt.Sprintf("[Round: %d sec]", rtSec),
		)

		return &CommandResult{
			Messages: messages,
			RoomBroadcast: []string{
				fmt.Sprintf(
					"%s works at the forge, smelting ore.",
					player.FirstName,
				),
			},
			PlayerState: player,
		}
	}

	// We found matching ore, but not the requested occurrence.
	if matchCount > 0 {
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You don't have that many matching pieces of ore. (Found %d)",
					matchCount,
				),
			},
		}
	}

	return &CommandResult{
		Messages: []string{"You don't have that ore to smelt."},
	}
}

// ---- CRAFTING (FORGE/CRAFT) ----

// metalDifficulty returns the quench success rate and XP award for a metal adjective name.
func metalDifficulty(metal string) (int, int) {
	switch strings.ToLower(metal) {
	case "copper":
		return 70, 100
	case "iron", "brass", "bronze":
		return 55, 200
	case "steel":
		return 45, 400
	case "truesteel":
		return 35, 800
	default:
		// exotic metals: randar, elkyri, etc.
		return 25, 1500
	}
}

func (e *GameEngine) doCraft(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		// List available recipes
		var recipes []string
		for _, def := range e.items {
			if containsFlag(def.Flags, "CRAFTABLE") {
				name := e.nouns[def.NameID]
				if name != "" {
					recipes = append(recipes, name)
				}
			}
		}
		if len(recipes) == 0 {
			return &CommandResult{Messages: []string{"No craftable recipes are known."}}
		}
		sort.Strings(recipes)
		msgs := []string{"You can craft the following items:"}
		for _, r := range recipes {
			msgs = append(msgs, fmt.Sprintf("  %s", r))
		}
		msgs = append(msgs, "Use CRAFT <item> to begin crafting.")
		return &CommandResult{Messages: msgs}
	}

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't craft here."}}
	}

	// Determine which workshop we're in
	isForge := containsModifier(room.Modifiers, "FORGE")
	isLoom := containsModifier(room.Modifiers, "LOOM")
	isFletcher := containsModifier(room.Modifiers, "FLETCHER")
	if !isForge && !isLoom && !isFletcher {
		return &CommandResult{Messages: []string{"You need to be at a workshop (forge, loom, or fletcher) to craft."}}
	}

	target := strings.ToLower(strings.Join(args, " "))

	// Find a CRAFTABLE item matching the target
	for _, def := range e.items {
		if !containsFlag(def.Flags, "CRAFTABLE") {
			continue
		}
		name := strings.ToLower(e.nouns[def.NameID])
		if !strings.HasPrefix(name, target) {
			continue
		}

		// Check workshop match
		skillNeeded := 0
		skillID := 0
		skillName := ""

		if isWeapon(def.Type) || def.Type == "ARMOR" {
			if !isForge {
				return &CommandResult{Messages: []string{"You need a forge to craft that."}}
			}
			skillID = 8
			skillName = "Weaponsmithing"
			if isWeapon(def.Type) {
				skillNeeded = def.Parameter1
			} else {
				skillNeeded = def.Weight / 3
			}
		} else if def.Substance == "WOOD" {
			if !isFletcher {
				return &CommandResult{Messages: []string{"You need a fletcher's workshop to craft that."}}
			}
			skillID = 18
			skillName = "Wood Lore"
			skillNeeded = def.Parameter1
		} else if def.Substance == "CLOTH" || def.Substance == "SOFTMETAL" {
			if !isLoom && !isForge {
				return &CommandResult{Messages: []string{"You need a loom or forge to craft that."}}
			}
			if def.Substance == "SOFTMETAL" {
				skillID = 0
				skillName = "Jeweler"
			} else {
				skillID = 15
				skillName = "Dyeing/Weaving"
			}
			skillNeeded = def.Parameter2
		} else {
			if isForge {
				skillID = 8
				skillName = "Weaponsmithing"
				skillNeeded = def.Parameter1
			} else {
				skillID = 18
				skillName = "Wood Lore"
				skillNeeded = def.Parameter1
			}
		}

		playerSkill := player.Skills[skillID]
		if playerSkill < skillNeeded {
			return &CommandResult{Messages: []string{
				fmt.Sprintf("Your %s skill (%d) is not high enough to craft that. You need at least %d.", skillName, playerSkill, skillNeeded),
			}}
		}

		// For weapons at the forge, enter the CRAFT→WORK cycle instead of instant creation
		if isForge && isWeapon(def.Type) {
			player.CraftingItem = name
			player.CraftingStep = 1
			player.CraftingMetal = "" // will be set by WORK <metal>
			return &CommandResult{
				Messages: []string{
					fmt.Sprintf("You begin to plan the crafting of your %s...", name),
					"[Next, work your item from a substance, e.g., \"WORK IRON.\"]",
				},
				RoomBroadcast: []string{fmt.Sprintf("%s studies a forge, planning something.", player.FirstName)},
			}
		}

		// Non-weapon crafting: immediate creation (original behavior)
		// Check for material in inventory
		materialFound := false
		materialIdx := -1
		materialAdj := 0
		for j, ii := range player.Inventory {
			mDef := e.items[ii.Archetype]
			if mDef == nil {
				continue
			}
			if mDef.Type == "MATERIAL" || mDef.Type == "MATERIAL2" {
				// Check if material's PARAMETER2 matches the skill
				if mDef.Parameter2 == skillID || mDef.Parameter2 == 0 {
					materialFound = true
					materialIdx = j
					if mDef.Parameter1 > 0 {
						materialAdj = mDef.Parameter1 // adjective from material
					}
					break
				}
			}
		}

		if !materialFound {
			return &CommandResult{Messages: []string{"You don't have the right materials. You need refined material (smelt ore, or forage wood/cloth)."}}
		}

		// Consume material
		player.Inventory = append(player.Inventory[:materialIdx], player.Inventory[materialIdx+1:]...)

		// Create the item
		item := InventoryItem{
			Archetype: def.Number,
			Adj1:      materialAdj, // material adjective
		}
		player.Inventory = append(player.Inventory, item)
		e.SavePlayer(ctx, player)

		itemName := e.formatItemName(def, item.Adj1, item.Adj2, item.Adj3)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You carefully craft %s!", itemName)},
			RoomBroadcast: []string{fmt.Sprintf("%s works diligently at the workshop.", player.FirstName)},
			PlayerState:   player,
		}
	}

	return &CommandResult{Messages: []string{fmt.Sprintf("You don't know how to craft '%s'.", target)}}
}

// ---- WORK (Forging Cycle) ----

func (e *GameEngine) doWork(ctx context.Context, player *Player, args []string) *CommandResult {
	if player.CraftingStep <= 0 {
		return &CommandResult{Messages: []string{"You aren't crafting anything. Use CRAFT <item> first."}}
	}

	room := e.rooms[player.RoomNumber]
	if room == nil || !containsModifier(room.Modifiers, "FORGE") {
		return &CommandResult{Messages: []string{"You need to be at a forge to work metal."}}
	}

	// Check roundtime
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You are still working... %.0f seconds remaining.", remaining+0.5)}}
	}

	switch player.CraftingStep {
	case 1: // Planned → need WORK <metal> to heat
		if len(args) == 0 {
			return &CommandResult{Messages: []string{"Work with what metal? e.g., WORK IRON"}}
		}
		metal := strings.ToLower(strings.Join(args, " "))
		// "work metal" by itself — prompt for specific type
		if metal == "metal" {
			return &CommandResult{Messages: []string{"Which metal? e.g., WORK IRON, WORK STEEL, WORK COPPER"}}
		}

		// Find matching material in inventory
		materialIdx := -1
		materialAdj := 0

		// Search inventory for the material
		for j, ii := range player.Inventory {
			mDef := e.items[ii.Archetype]
			if mDef == nil {
				continue
			}

			// Check if it's a material for Weaponsmithing (Skill 8)
			if (mDef.Type == "MATERIAL" || mDef.Type == "MATERIAL2" || mDef.Type == "MISC") && (mDef.Parameter2 == 8 || mDef.Parameter2 == 0) {

				// Get the names for comparison
				mNoun := strings.ToLower(e.getItemNounName(mDef)) // e.g., "metal"
				instAdj := strings.ToLower(e.getAdjName(ii.Adj1)) // e.g., "copper"

				// FLEXIBLE MATCHING:
				// Check if the user's input matches the adjective, the noun, or the combined name
				fullItemName := instAdj + " " + mNoun // e.g., "copper metal"

				if metal == instAdj || metal == mNoun || metal == fullItemName ||
					strings.Contains(fullItemName, metal) {

					materialIdx = j
					materialAdj = ii.Adj1
					break
				}
			}
		}
		if materialIdx < 0 {
			// Also accept the metal name directly as a known metal type
			knownMetals := []string{"copper", "iron", "brass", "bronze", "steel", "truesteel", "randar", "elkyri"}
			validMetal := false
			for _, km := range knownMetals {
				if metal == km {
					validMetal = true
					break
				}
			}
			if !validMetal {
				return &CommandResult{Messages: []string{fmt.Sprintf("You don't have any %s metal to work with.", metal)}}
			}
			// Look for any material in inventory
			for j, ii := range player.Inventory {
				mDef := e.items[ii.Archetype]
				if mDef == nil {
					continue
				}
				if mDef.Type == "MATERIAL" || mDef.Type == "MATERIAL2" {
					if mDef.Parameter2 == 8 || mDef.Parameter2 == 0 {
						materialIdx = j
						if mDef.Parameter1 > 0 {
							materialAdj = mDef.Parameter1
						}
						break
					}
				}
			}
			if materialIdx < 0 {
				return &CommandResult{Messages: []string{fmt.Sprintf("You don't have any %s metal to work with.", metal)}}
			}
		}

		// Consume material
		player.Inventory = append(player.Inventory[:materialIdx], player.Inventory[materialIdx+1:]...)
		_ = materialAdj

		player.CraftingMetal = metal
		player.CraftingStep = 2
		player.RoundTimeExpiry = time.Now().Add(15 * time.Second)
		player.RoundTime = 15
		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You place some %s metal into a mold in the forge and heat it until it is roughly the shape you desire.", metal)},
			RoomBroadcast: []string{fmt.Sprintf("%s works diligently at the forge.", player.FirstName)},
			PlayerState:   player,
		}
	case 2: // Heated → Hammer
		player.CraftingStep = 3
		player.RoundTimeExpiry = time.Now().Add(15 * time.Second)
		player.RoundTime = 15
		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You remove the %s metal from the forge and begin to hammer it into shape on the anvil.", player.CraftingMetal)},
			RoomBroadcast: []string{fmt.Sprintf("%s works diligently at the forge.", player.FirstName)},
			PlayerState:   player,
		}

	case 3: // Hammered → Quench (skill check)
		smithSkill := player.Skills[8]
		baseChance, _ := metalDifficulty(player.CraftingMetal)
		// Add skill bonus: +3% per skill level
		chance := baseChance + smithSkill*3
		if chance > 95 {
			chance = 95
		}

		roll := rand.Intn(100) + 1
		player.RoundTimeExpiry = time.Now().Add(15 * time.Second)
		player.RoundTime = 15

		if roll > chance {
			// Fail: restart from heating step
			player.CraftingStep = 2
			e.SavePlayer(ctx, player)
			return &CommandResult{
				Messages:      []string{"You quench the hot metal in a pool of water. After some examination, you surmise that it will require more work."},
				RoomBroadcast: []string{fmt.Sprintf("%s works diligently at the forge.", player.FirstName)},
				PlayerState:   player,
			}
		}

		// Progress or almost done
		player.CraftingStep = 4
		e.SavePlayer(ctx, player)

		// High roll = almost done message
		if roll <= chance/2 {
			return &CommandResult{
				Messages:      []string{"You quench the hot metal in a pool of water. It looks like it is almost finished!"},
				RoomBroadcast: []string{fmt.Sprintf("%s works diligently at the forge.", player.FirstName)},
				PlayerState:   player,
			}
		}
		return &CommandResult{
			Messages:      []string{"You quench the hot metal in a pool of water. Pleased with your progress, you surmise that it will only require a little more work."},
			RoomBroadcast: []string{fmt.Sprintf("%s works diligently at the forge.", player.FirstName)},
			PlayerState:   player,
		}

	case 4: // Quenched → Buff
		player.CraftingStep = 5
		player.RoundTimeExpiry = time.Now().Add(15 * time.Second)
		player.RoundTime = 15
		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages:      []string{"You buff the metal, smoothing and polishing the surface. Your weapon is nearly complete!"},
			RoomBroadcast: []string{fmt.Sprintf("%s works diligently at the forge.", player.FirstName)},
			PlayerState:   player,
		}

	case 5: // Buffed → Sharpen (complete!)
		player.CraftingStep = 0
		player.RoundTimeExpiry = time.Now().Add(15 * time.Second)
		player.RoundTime = 15

		// Find the CRAFTABLE item definition matching the crafting item
		var weaponDef *gameworld.ItemDef
		for _, def := range e.items {
			if !containsFlag(def.Flags, "CRAFTABLE") {
				continue
			}
			if !isWeapon(def.Type) {
				continue
			}
			name := strings.ToLower(e.nouns[def.NameID])
			if name == player.CraftingItem {
				weaponDef = def
				break
			}
		}

		if weaponDef == nil {
			player.CraftingMetal = ""
			player.CraftingItem = ""
			e.SavePlayer(ctx, player)
			return &CommandResult{Messages: []string{"Something went wrong with your crafting."}}
		}

		// Find the adjective ID for the metal name
		metalAdj := 0
		metalLower := strings.ToLower(player.CraftingMetal)
		for id, adjName := range e.adjectives {
			if strings.ToLower(adjName) == metalLower {
				metalAdj = id
				break
			}
		}

		// Create the weapon
		item := InventoryItem{
			Archetype: weaponDef.Number,
			Adj1:      metalAdj,
		}
		player.Inventory = append(player.Inventory, item)

		// Award XP
		_, xpAward := metalDifficulty(player.CraftingMetal)
		player.Experience += xpAward

		itemName := e.formatItemName(weaponDef, item.Adj1, item.Adj2, item.Adj3)
		craftingMetal := player.CraftingMetal
		craftingItem := player.CraftingItem

		player.CraftingMetal = ""
		player.CraftingItem = ""
		e.SavePlayer(ctx, player)

		msgs := []string{
			fmt.Sprintf("You carefully sharpen your weapon on a large whetstone until its cutting edge is honed to deadly precision. Your %s %s is complete!", craftingMetal, craftingItem),
		}
		if xpAward > 0 {
			msgs = append(msgs, fmt.Sprintf("You have been awarded %d experience points.", xpAward))
		}

		return &CommandResult{
			Messages:      msgs,
			RoomBroadcast: []string{fmt.Sprintf("%s finishes crafting %s!", player.FirstName, itemName)},
			PlayerState:   player,
		}

	default:
		player.CraftingStep = 0
		player.CraftingItem = ""
		player.CraftingMetal = ""
		return &CommandResult{Messages: []string{"Your crafting state was invalid. It has been reset."}}
	}
}

// ---- REPAIR ----

func (e *GameEngine) doRepair(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Repair what?"}}
	}

	room := e.rooms[player.RoomNumber]
	if room == nil || !containsModifier(room.Modifiers, "FORGE") {
		return &CommandResult{Messages: []string{"You need to be at a forge to repair weapons."}}
	}

	smithSkill := player.Skills[8]
	if smithSkill < 1 {
		return &CommandResult{Messages: []string{"You have no training in Weaponsmithing."}}
	}

	// Check roundtime
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You are still working... %.0f seconds remaining.", remaining+0.5)}}
	}

	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	// Find the weapon in inventory with DAMAGED state
	for i, ii := range player.Inventory {
		def := e.items[ii.Archetype]
		if def == nil {
			continue
		}

		if !isWeapon(def.Type) {
			continue
		}

		name := strings.ToLower(e.getItemNounName(def))
		if !strings.HasPrefix(name, target) {
			continue
		}

		if skip > 0 {
			skip--
			continue
		}

		if ii.State != "DAMAGED" {
			return &CommandResult{
				Messages: []string{"That doesn't need repair."},
			}
		}

		// Skill check: base 40% + smithSkill*5
		chance := 40 + smithSkill*5
		if chance > 95 {
			chance = 95
		}
		roll := rand.Intn(100) + 1

		player.RoundTimeExpiry = time.Now().Add(10 * time.Second)
		player.RoundTime = 10

		itemName := e.formatItemName(def, ii.Adj1, ii.Adj2, ii.Adj3)

		if roll > chance {
			e.SavePlayer(ctx, player)
			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"[Success: %d%%, Roll %d] You are unable to repair the weapon.",
						chance,
						roll,
					),
					fmt.Sprintf("[Round: %d sec]", player.RoundTime),
				},
				RoomBroadcast: []string{fmt.Sprintf("%s works at the forge, trying to repair a weapon.", player.FirstName)},
				PlayerState:   player,
			}
		}

		// Success: remove DAMAGED state
		player.Inventory[i].State = ""

		xpMsgs := e.awardExperience(player, 50)

		e.SavePlayer(ctx, player)

		messages := []string{
			fmt.Sprintf(
				"[Success: %d%%, Roll %d] You carefully repair your %s.",
				chance,
				roll,
				itemName,
			),
			fmt.Sprintf("[Round: %d sec]", player.RoundTime),
		}

		messages = append(messages, xpMsgs...)

		return &CommandResult{
			Messages: messages,
			RoomBroadcast: []string{
				fmt.Sprintf(
					"%s works at the forge, repairing a weapon.",
					player.FirstName,
				),
			},
			PlayerState: player,
		}
	}

	return &CommandResult{Messages: []string{"That doesn't need repair."}}
}

// ---- FORAGING ----
func (e *GameEngine) doForageReal(ctx context.Context, player *Player) *CommandResult {

	// Check existing roundtime.
	if time.Now().Before(player.RoundTimeExpiry) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("[Wait %.0f seconds...]", remaining),
			},
		}
	}

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{
			Messages: []string{"You can't forage here."},
		}
	}

	terrain := room.Terrain

	// Find FORAGEDEF entries matching this terrain.
	var candidates []gameworld.ForageDef

	for _, fd := range e.forageDefs {
		if strings.EqualFold(fd.Terrain, terrain) {
			candidates = append(candidates, fd)
		}
	}

	// No forage table for this terrain.
	if len(candidates) == 0 {
		return &CommandResult{
			Messages: []string{"There is nothing to forage here."},
		}
	}

	// Foraging requires Woodlore (skill 18).
	woodlore := player.Skills[18]

	if woodlore <= 0 {
		return &CommandResult{
			Messages: []string{
				"You lack the woodlore necessary to forage effectively.",
			},
		}
	}

	// Every valid forage attempt takes 20 seconds.
	rtSec := 20
	player.RoundTimeExpiry = time.Now().Add(
		time.Duration(rtSec) * time.Second,
	)

	// LOFP-style skill check:
	// 25% base + 5% per Woodlore rank.
	chance := 25 + woodlore*5

	if chance > 95 {
		chance = 95
	}

	forageRoll := rand.Intn(100) + 1

	// Failed forage attempt.
	if forageRoll > chance {
		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages: []string{
				"You search the area but find nothing useful.",
				fmt.Sprintf("[Round: %d sec]", rtSec),
			},
			RoomBroadcast: []string{
				fmt.Sprintf(
					"%s forages in the area.",
					player.FirstName,
				),
			},
			PlayerState: player,
		}
	}

	// Successful forage attempt.
	// Ratio determines the relative probability of which
	// forageable item is found.
	totalRatio := 0

	for _, fd := range candidates {
		if fd.Ratio > 0 {
			totalRatio += fd.Ratio
		}
	}

	// Invalid/empty forage table.
	if totalRatio <= 0 {
		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages: []string{
				"You search the area but find nothing useful.",
				fmt.Sprintf("[Round: %d sec]", rtSec),
			},
			RoomBroadcast: []string{
				fmt.Sprintf(
					"%s forages in the area.",
					player.FirstName,
				),
			},
			PlayerState: player,
		}
	}

	// Weighted random selection from the FORAGEDEF table.
	roll := rand.Intn(totalRatio)
	cumulative := 0

	for _, fd := range candidates {
		if fd.Ratio <= 0 {
			continue
		}

		cumulative += fd.Ratio

		if roll >= cumulative {
			continue
		}

		itemDef := e.items[fd.ItemNum]
		if itemDef == nil {
			continue
		}

		item := InventoryItem{
			Archetype: fd.ItemNum,
			Val2:      fd.Val2,
			Val5:      fd.Val5,
		}

		if fd.AdjNum > 0 {
			item.Adj1 = fd.AdjNum
		}

		player.Inventory = append(player.Inventory, item)

		// Successful forage earns experience.
		xpMsgs := e.awardExperience(player, 100)

		e.SavePlayer(ctx, player)

		itemName := e.formatItemName(
			itemDef,
			item.Adj1,
			item.Adj2,
			item.Adj3,
		)

		messages := []string{
			fmt.Sprintf(
				"You search the area and find %s!",
				itemName,
			),
		}

		messages = append(messages, xpMsgs...)

		messages = append(
			messages,
			fmt.Sprintf("[Round: %d sec]", rtSec),
		)

		return &CommandResult{
			Messages: messages,
			RoomBroadcast: []string{
				fmt.Sprintf(
					"%s forages in the area.",
					player.FirstName,
				),
			},
			PlayerState: player,
		}
	}

	// Bad item reference or otherwise unusable forage entry.
	e.SavePlayer(ctx, player)

	return &CommandResult{
		Messages: []string{
			"You search the area but find nothing useful.",
			fmt.Sprintf("[Round: %d sec]", rtSec),
		},
		RoomBroadcast: []string{
			fmt.Sprintf(
				"%s forages in the area.",
				player.FirstName,
			),
		},
		PlayerState: player,
	}
}

// doForageFallback provides generic foraging when no ForageDefs are loaded.
func (e *GameEngine) doForageFallback(ctx context.Context, player *Player, terrain string) *CommandResult {
	// Generic items by terrain
	type fallbackItem struct {
		name string
		arch int
	}
	var items []fallbackItem

	// Try to find common forage items in the database
	for num, def := range e.items {
		if def.Weight >= 1000 || def.Weight <= 0 {
			continue
		}
		noun := strings.ToLower(e.nouns[def.NameID])
		switch terrain {
		case "FOREST":
			if noun == "bark" || noun == "branch" || noun == "root" || noun == "leaf" || noun == "berry" || noun == "mushroom" {
				items = append(items, fallbackItem{noun, num})
			}
		case "MOUNTAIN":
			if noun == "crystal" || noun == "stone" || noun == "moss" || noun == "lichen" {
				items = append(items, fallbackItem{noun, num})
			}
		case "PLAIN":
			if noun == "grass" || noun == "flower" || noun == "cotton" || noun == "herb" {
				items = append(items, fallbackItem{noun, num})
			}
		case "SWAMP":
			if noun == "moss" || noun == "reed" || noun == "root" || noun == "vine" {
				items = append(items, fallbackItem{noun, num})
			}
		case "JUNGLE":
			if noun == "vine" || noun == "fruit" || noun == "flower" || noun == "leaf" {
				items = append(items, fallbackItem{noun, num})
			}
		}
	}

	// 30% chance of finding nothing
	if rand.Intn(100) < 30 || len(items) == 0 {
		return &CommandResult{
			Messages:      []string{"You search the area but find nothing useful."},
			RoomBroadcast: []string{fmt.Sprintf("%s forages in the area.", player.FirstName)},
		}
	}

	chosen := items[rand.Intn(len(items))]
	item := InventoryItem{Archetype: chosen.arch}
	player.Inventory = append(player.Inventory, item)
	e.SavePlayer(ctx, player)

	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You search the area and find some %s!", chosen.name)},
		RoomBroadcast: []string{fmt.Sprintf("%s forages in the area.", player.FirstName)},
		PlayerState:   player,
	}
}

func (e *GameEngine) resolveReagentType(item *InventoryItem) int {
	if item == nil {
		return 0
	}

	// An explicit value on this particular item always wins.
	// This handles normally foraged, skinned, and scripted reagents.
	if item.Val5 > 0 {
		return item.Val5
	}

	matchesForage := func(fd gameworld.ForageDef) bool {
		if fd.ItemNum != item.Archetype || fd.Val5 <= 0 {
			return false
		}

		if fd.AdjNum <= 0 {
			return true
		}

		return item.Adj1 == fd.AdjNum ||
			item.Adj2 == fd.AdjNum ||
			item.Adj3 == fd.AdjNum
	}

	// Check permanent/base forage definitions.
	for _, fd := range e.baseForageDefs {
		if matchesForage(fd) {
			item.Val5 = fd.Val5
			return item.Val5
		}
	}

	// Check every seasonal forage definition.
	for _, defs := range e.seasonalForageDefs {
		for _, fd := range defs {
			if matchesForage(fd) {
				item.Val5 = fd.Val5
				return item.Val5
			}
		}
	}

	// No FORAGEDEF matched. Check monster SKINITEM definitions.
	// This handles shop-bought creature components such as newt eyes:
	// STOREITEM gives us the archetype/adjective but not Val5.
	for _, mon := range e.monsters {
		if mon == nil || mon.SkinAdj <= 0 {
			continue
		}

		// The shop item's adjective must identify the same creature skin.
		if item.Adj1 != mon.SkinAdj &&
			item.Adj2 != mon.SkinAdj &&
			item.Adj3 != mon.SkinAdj {
			continue
		}

		for _, sd := range mon.SkinItems {
			if sd.Archetype == item.Archetype && sd.Val5 > 0 {
				item.Val5 = sd.Val5
				return item.Val5
			}
		}
	}

	// Legacy store-only reagent: newt eye.
	// The alchemy guide classifies newt eyes as Mild Magic,
	// corresponding to reagent type 9 (Miscellaneous, common).
	if item.Archetype == 520 &&
		(item.Adj1 == 1105 || item.Adj2 == 1105 || item.Adj3 == 1105) {
		item.Val5 = 9
		return item.Val5
	}

	return 0
}

// ---- DYEING ----

func (e *GameEngine) doDye(ctx context.Context, player *Player, args []string) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil || !containsModifier(room.Modifiers, "LOOM") {
		return &CommandResult{Messages: []string{"You need to be at a loom to dye items."}}
	}
	if player.Skills[15] < 1 {
		return &CommandResult{Messages: []string{"You have no training in Dyeing/Weaving."}}
	}
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Dye what? Usage: DYE <item> WITH <dye>"}}
	}

	raw := strings.ToLower(strings.Join(args, " "))
	itemTarget, dyeTarget := parseWithClause(raw)
	if dyeTarget == "" {
		return &CommandResult{Messages: []string{"Dye with what? Usage: DYE <item> WITH <dye>"}}
	}

	// Find the item to dye in inventory (must be DYEABLE)
	var targetItem *InventoryItem
	var targetIdx int
	var targetDef *gameworld.ItemDef
	for i, ii := range player.Inventory {
		def := e.items[ii.Archetype]
		if def == nil || !containsFlag(def.Flags, "DYEABLE") {
			continue
		}
		name := strings.ToLower(e.getItemNounName(def))
		if strings.HasPrefix(name, itemTarget) {
			targetItem = &player.Inventory[i]
			targetIdx = i
			targetDef = def
			break
		}
	}
	if targetItem == nil {
		return &CommandResult{Messages: []string{"You don't have a dyeable item matching that."}}
	}
	_ = targetIdx

	// Find the dye in inventory (must have DYE flag)
	for j, ii := range player.Inventory {
		def := e.items[ii.Archetype]
		if def == nil || !containsFlag(def.Flags, "DYE") {
			continue
		}
		name := strings.ToLower(e.getItemNounName(def))
		if strings.HasPrefix(name, dyeTarget) || strings.Contains(name, dyeTarget) {
			// Apply dye: color goes to Adj2, preserving material adjective in Adj1
			// PARAMETER1 = color adjective, PARAMETER3 = optional texture adjective
			if def.Parameter1 > 0 {
				targetItem.Adj2 = def.Parameter1
			}
			if def.Parameter3 > 0 {
				targetItem.Adj3 = def.Parameter3
			}
			// Consume the dye
			player.Inventory = append(player.Inventory[:j], player.Inventory[j+1:]...)
			e.SavePlayer(ctx, player)

			dyedName := e.formatItemName(targetDef, targetItem.Adj1, targetItem.Adj2, targetItem.Adj3)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You carefully dye the material. It is now %s.", dyedName)},
				RoomBroadcast: []string{fmt.Sprintf("%s works at the loom, dyeing materials.", player.FirstName)},
				PlayerState:   player,
			}
		}
	}

	return &CommandResult{Messages: []string{"You don't have that dye."}}
}

// ---- ANALYZE (ore purity) ----

func (e *GameEngine) doAnalyze(ctx context.Context, player *Player, args []string, oreSpell bool) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Analyze what?"}}
	}

	// Support an optional ordinal at the end:
	//   ANALYZE LEG
	//   ANALYZE LEG 2
	//   ANALYZE SPIDER LEG
	//   ANALYZE ARACHEN LEG
	ordinal := 1

	if len(args) > 1 {
		if n, err := strconv.Atoi(args[len(args)-1]); err == nil && n > 0 {
			ordinal = n
			args = args[:len(args)-1]
		}
	}

	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Analyze what?"}}
	}

	target := strings.ToLower(strings.Join(args, " "))
	matchCount := 0

	for i := range player.Inventory {
		ii := &player.Inventory[i]

		def := e.items[ii.Archetype]
		if def == nil {
			continue
		}

		nounName := strings.ToLower(e.getItemNounName(def))

		fullName := strings.ToLower(
			e.formatItemName(def, ii.Adj1, ii.Adj2, ii.Adj3),
		)

		// Strip the article from the formatted name so:
		// "spider leg" matches "a spider leg"
		searchName := fullName
		searchName = strings.TrimPrefix(searchName, "a ")
		searchName = strings.TrimPrefix(searchName, "an ")
		searchName = strings.TrimPrefix(searchName, "the ")
		searchName = strings.TrimPrefix(searchName, "some ")

		// Match either the noun ("leg") or the compound item name
		// ("spider leg", "arachen leg", etc.).
		if !strings.HasPrefix(nounName, target) &&
			!strings.HasPrefix(searchName, target) {
			continue
		}

		matchCount++
		if matchCount != ordinal {
			continue
		}

		if oreSpell && def.Type != "ORE" {
			return &CommandResult{
				Messages: []string{"That is not ore."},
			}
		}

		if def.Type == "ORE" {
			if !oreSpell {
				miningSkill := player.Skills[35]

				if miningSkill < 3 {
					return &CommandResult{
						Messages: []string{
							"You don't have enough mining skill to analyze this ore. (Need Mining 3+)",
						},
					}
				}
			}

			purity := ii.Val3
			desc := "poor"

			if purity > 80 {
				desc = "nearly solid metal"
			} else if purity > 60 {
				desc = "excellent"
			} else if purity > 40 {
				desc = "good"
			} else if purity > 20 {
				desc = "fair"
			}

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"You examine the ore carefully. It appears to be of %s quality. (Purity: %d%%)",
						desc,
						purity,
					),
				},
			}
		}

		alchemyRank := player.Skills[31] // whatever your actual skill lookup is

		if alchemyRank <= 0 {
			return &CommandResult{
				Messages: []string{
					"You lack the alchemical knowledge needed to analyze that.",
				},
			}
		}

		// Reagent analysis for alchemy
		if containsFlag(def.Flags, "REAGENT") {
			reagentTypes := map[int]string{
				1:  "Power 1",
				2:  "Power 2",
				3:  "Power 3",
				4:  "Health",
				5:  "Harm",
				6:  "Body",
				7:  "Resist",
				8:  "Enhancement",
				9:  "Miscellaneous, common",
				10: "Miscellaneous, uncommon",
				11: "Miscellaneous, rare",
				12: "Mind",
				13: "Protect",
			}

			// Val5 isn't always stored, so resolve it from FORAGEDEF if necessary.
			rType := e.resolveReagentType(ii)

			typeName := reagentTypes[rType]
			if typeName == "" {
				typeName = "unknown"
			}

			itemName := e.formatItemName(
				def,
				ii.Adj1,
				ii.Adj2,
				ii.Adj3,
			)

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"You analyze %s. Alchemical properties: %s.",
						itemName,
						typeName,
					),
				},
			}
		}

		return &CommandResult{
			Messages: []string{
				"You can't determine anything special about that.",
			},
		}
	}

	return &CommandResult{Messages: []string{"You don't have that."}}
}

// ---- BREW (Alchemy) ----

// Alchemy recipe: catalyst type (1-3) + two reagent types (A-J) → potion spell
type alchemyRecipe struct {
	code     string // e.g. "1AB"
	catalyst int    // 1=mild, 2=strong, 3=very strong
	reagent1 string // A-J
	reagent2 string // A-J
	spellID  int    // resulting potion spell
	level    int    // minimum alchemy level
	name     string
	color    string
}

var alchemyRecipes = []alchemyRecipe{
	{"1AB", 1, "A", "B", 316, 1, "Body Restoration I", "Green"},
	{"1AF", 1, "A", "F", 313, 1, "Body Destruction I", "Dark"},
	{"1AH", 1, "A", "H", 520, 1, "Night Vision", "Black"},
	{"1AG", 1, "A", "G", 518, 2, "Claw Growth", "Ebony"},
	{"1AC", 1, "A", "C", 339, 3, "Destroy Undead I", "White"},
	{"1CH", 1, "C", "H", 506, 3, "Resist Weather", "Blue"},
	{"1EG", 1, "E", "G", 226, 3, "Paranoia", "Pink"},
	{"1AI", 1, "A", "I", 207, 4, "Strength I", "Rose"},
	{"1GI", 1, "G", "I", 513, 4, "Agility I", "Khaki"},
	{"1AD", 1, "A", "D", 102, 5, "Mystic Armor", "Silvery Blue"},
	{"2AF", 2, "A", "F", 314, 5, "Body Destruction II", "Dark"},
	{"2CF", 2, "C", "F", 317, 5, "Body Restoration II", "Green"},
	{"2CG", 2, "C", "G", 401, 5, "Dispel Lesser Magic", "Red"},
	{"2HI", 2, "H", "I", 210, 5, "Haste", "Sea Blue"},
	{"2DG", 2, "D", "G", 521, 7, "Camouflage", "Forest Camo"},
	{"2AG", 2, "A", "G", 511, 8, "Carapace", "Brown"},
	{"2AI", 2, "A", "I", 208, 8, "Strength II", "Rose"},
	{"2AC", 2, "A", "C", 335, 9, "Invigoration II", "Pink"},
	{"2EG", 2, "E", "G", 403, 9, "Mindlink", "Azure"},
	{"2CH", 2, "C", "H", 509, 10, "Repel Plants", "Mossy Green"},
	{"3AF", 3, "A", "F", 315, 10, "Body Destruction III", "Dark"},
	{"2BI", 2, "B", "I", 303, 11, "Cure Poison", "Purple"},
	{"2GI", 2, "G", "I", 514, 11, "Agility II", "Khaki"},
	{"3AG", 3, "A", "G", 224, 11, "Fly", "Cerulean"},
	{"2BD", 2, "B", "D", 319, 12, "Cure Disease", "Pale Blue"},
	{"2DI", 2, "D", "I", 234, 13, "Spell Shield", "Silvery Blue"},
	{"3DG", 3, "D", "G", 225, 14, "Invisibility", "White"},
	{"3AD", 3, "A", "D", 105, 15, "Globe of Protection", "Violet"},
	{"3AI", 3, "A", "I", 209, 16, "Strength III", "Rose"},
	{"3GI", 3, "G", "I", 515, 16, "Agility III", "Khaki"},
	{"3CH", 3, "C", "H", 510, 18, "Repel Plants & Webs", "Forest Green"},
	{"3FJ", 3, "F", "J", 232, 20, "Mist Form", "Gray"},
}

// Reagent type letters mapped to val5 values
var reagentLetters = map[int]string{
	6:  "A",
	4:  "B",
	7:  "C", // Resist
	13: "D",
	12: "E",
	5:  "F",
	8:  "G",
	9:  "H",
	10: "I",
	11: "J",
}

var catalystFromVal5 = map[int]int{1: 1, 2: 2, 3: 3}

func (e *GameEngine) doBrew(ctx context.Context, player *Player, args []string) *CommandResult {
	if player.Skills[31] < 1 {
		return &CommandResult{Messages: []string{"You have no training in Alchemy."}}
	}

	if len(args) == 0 {
		var msgs []string
		msgs = append(msgs, "=== Alchemy Recipes (by level) ===")

		for _, r := range alchemyRecipes {
			if r.level <= player.Skills[31] {
				msgs = append(
					msgs,
					fmt.Sprintf("  Level %2d: %s (%s)", r.level, r.name, r.color),
				)
			}
		}

		if len(msgs) == 1 {
			msgs = append(msgs, "  You don't know any recipes at your current level.")
		}

		return &CommandResult{Messages: msgs}
	}

	// BREW <reagent> IN <container>
	raw := strings.ToLower(strings.Join(args, " "))
	reagentTarget, containerTarget := parseInClause(raw)

	if containerTarget == "" {
		return &CommandResult{
			Messages: []string{"Brew in what? Usage: BREW <reagent> IN <flask/vial>"},
		}
	}

	// Find the reagent.
	reagentIdx := -1

	for i := range player.Inventory {
		ii := &player.Inventory[i]
		def := e.items[ii.Archetype]
		if def == nil {
			continue
		}

		if !containsFlag(def.Flags, "REAGENT") {
			continue
		}

		name := strings.ToLower(e.getItemNounName(def))

		if strings.HasPrefix(name, reagentTarget) ||
			strings.Contains(name, reagentTarget) {
			reagentIdx = i
			break
		}
	}

	if reagentIdx < 0 {
		return &CommandResult{
			Messages: []string{"You don't have that reagent."},
		}
	}

	// Resolve the reagent type before doing anything with the inventory.
	reagentItem := &player.Inventory[reagentIdx]
	reagentVal5 := e.resolveReagentType(reagentItem)

	if reagentVal5 <= 0 {
		return &CommandResult{
			Messages: []string{"That ingredient has no known alchemical properties."},
		}
	}

	reagentDef := e.items[reagentItem.Archetype]
	reagentName := "the reagent"

	if reagentDef != nil {
		reagentName = e.getItemNounName(reagentDef)
	}

	// Find the container.
	containerIdx := -1

	for i := range player.Inventory {
		if i == reagentIdx {
			continue
		}

		ii := &player.Inventory[i]
		def := e.items[ii.Archetype]
		if def == nil {
			continue
		}

		name := strings.ToLower(e.getItemNounName(def))

		if def.Type == "LIQCONTAINER" || def.Container == "IN" {
			if strings.HasPrefix(name, containerTarget) ||
				strings.Contains(name, containerTarget) {
				containerIdx = i
				break
			}
		}
	}

	if containerIdx < 0 {
		return &CommandResult{
			Messages: []string{
				"You don't have a suitable container. You need a flask, vial, or bottle.",
			},
		}
	}

	// Consume the reagent.
	player.Inventory = append(
		player.Inventory[:reagentIdx],
		player.Inventory[reagentIdx+1:]...,
	)

	// Removing the reagent may shift the container index.
	if containerIdx > reagentIdx {
		containerIdx--
	}

	container := &player.Inventory[containerIdx]

	// Val5 = number of ingredients currently in the brew.
	container.Val5++

	// Val4 stores the reagent Val5 values.
	// Each reagent occupies two decimal digits.
	container.Val4 = container.Val4*100 + reagentVal5

	// We don't have a complete potion yet.
	if container.Val5 < 3 {
		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You add %s to the brew. (%d/3 ingredients)",
					reagentName,
					container.Val5,
				),
			},
			RoomBroadcast: []string{
				fmt.Sprintf(
					"%s adds an ingredient to a bubbling brew.",
					player.FirstName,
				),
			},
			PlayerState: player,
		}
	}

	// Three ingredients have been added.
	code3 := container.Val4 % 100
	code2 := (container.Val4 / 100) % 100
	code1 := (container.Val4 / 10000) % 100

	codes := []int{code1, code2, code3}

	// Find exactly one catalyst and exactly two normal reagents.
	catLevel := 0
	var reagentCodes []int

	for _, code := range codes {
		if catalyst := catalystFromVal5[code]; catalyst > 0 {
			// More than one catalyst is invalid.
			if catLevel != 0 {
				container.Val4 = 0
				container.Val5 = 0

				e.SavePlayer(ctx, player)

				return &CommandResult{
					Messages: []string{
						"A foul odor rises from the brew. The combination produces nothing useful.",
					},
					RoomBroadcast: []string{
						fmt.Sprintf("%s's brew emits a foul odor.", player.FirstName),
					},
					PlayerState: player,
				}
			}

			catLevel = catalyst
			continue
		}

		// Must be a recognized non-catalyst reagent.
		if _, ok := reagentLetters[code]; ok {
			reagentCodes = append(reagentCodes, code)
		}
	}

	// A valid potion requires:
	//   1 catalyst
	//   2 reagent categories
	if catLevel == 0 || len(reagentCodes) != 2 {
		container.Val4 = 0
		container.Val5 = 0

		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages: []string{
				"A foul odor rises from the brew. The combination produces nothing useful.",
			},
			RoomBroadcast: []string{
				fmt.Sprintf("%s's brew emits a foul odor.", player.FirstName),
			},
			PlayerState: player,
		}
	}

	reagent1 := reagentLetters[reagentCodes[0]]
	reagent2 := reagentLetters[reagentCodes[1]]

	// Find the recipe.
	for _, recipe := range alchemyRecipes {
		if recipe.catalyst != catLevel {
			continue
		}

		// Reagent order does not matter.
		if !((reagent1 == recipe.reagent1 && reagent2 == recipe.reagent2) ||
			(reagent1 == recipe.reagent2 && reagent2 == recipe.reagent1)) {
			continue
		}

		// Valid recipe, but the player's Alchemy skill is too low.
		if player.Skills[31] < recipe.level {
			container.Val4 = 0
			container.Val5 = 0

			e.SavePlayer(ctx, player)

			return &CommandResult{
				Messages: []string{
					"The brew bubbles violently! It's a valid recipe, but beyond your current skill.",
					fmt.Sprintf(
						"(Requires Alchemy level %d, you have %d)",
						recipe.level,
						player.Skills[31],
					),
				},
				PlayerState: player,
			}
		}

		// Success.
		container.Val3 = recipe.spellID
		// Potions normally contain 2-5 sips, limited by container capacity.
		maxSips := 5

		if containerDef := e.items[container.Archetype]; containerDef != nil &&
			containerDef.Interior > 0 &&
			containerDef.Interior < maxSips {
			maxSips = containerDef.Interior
		}

		minSips := 2
		if maxSips < minSips {
			minSips = maxSips
		}

		container.Val2 = minSips
		if maxSips > minSips {
			container.Val2 += rand.Intn(maxSips - minSips + 1)
		}

		// Clear the temporary brewing state.
		container.Val4 = 0
		container.Val5 = 0

		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"The brew shimmers magically! You have created a %s potion! (%s, %d sips)",
					recipe.name,
					recipe.color,
					container.Val2,
				),
			},
			RoomBroadcast: []string{
				fmt.Sprintf(
					"%s completes a potion that shimmers with magical energy!",
					player.FirstName,
				),
			},
			PlayerState: player,
		}
	}

	// Three legitimate ingredients, but they don't form a known recipe.
	container.Val4 = 0
	container.Val5 = 0

	e.SavePlayer(ctx, player)

	return &CommandResult{
		Messages: []string{
			"A foul odor rises from the brew. The combination produces nothing useful.",
		},
		RoomBroadcast: []string{
			fmt.Sprintf("%s's brew emits a foul odor.", player.FirstName),
		},
		PlayerState: player,
	}
}

// parseInClause splits "X in Y" into (X, Y).
func parseInClause(s string) (string, string) {
	idx := strings.Index(s, " in ")
	if idx < 0 {
		return s, ""
	}
	return strings.TrimSpace(s[:idx]), strings.TrimSpace(s[idx+4:])
}
