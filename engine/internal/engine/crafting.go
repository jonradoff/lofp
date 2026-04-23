package engine

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// ---- MINING ----

func (e *GameEngine) doMineReal(ctx context.Context, player *Player) *CommandResult {
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

	// Check for mining tool
	hasTool := false
	if player.Wielded != nil {
		wDef := e.items[player.Wielded.Archetype]
		if wDef != nil {
			noun := strings.ToLower(e.nouns[wDef.NameID])
			if noun == "pickaxe" || noun == "hammer" || noun == "shovel" || wDef.Type == "MINETOOL" {
				hasTool = true
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

	// Success chance: base 30% + mining*5 + STR/10
	chance := 30 + miningSkill*5 + player.Strength/10
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
			Messages:      []string{"You swing at the rock face but find nothing useful."},
			RoomBroadcast: []string{fmt.Sprintf("%s swings a mining tool at the rock.", player.FirstName)},
		}
	}

	// Find ore item (archetype 1369 = general ore, 1371 = iron ore)
	oreArch := 1369
	if grade == "A" && rand.Intn(100) < 30 {
		oreArch = 1371 // iron ore more common in grade A
	}
	oreDef := e.items[oreArch]
	if oreDef == nil {
		// Fallback: find any ORE type item
		for num, def := range e.items {
			if def.Type == "ORE" {
				oreArch = num
				oreDef = def
				break
			}
		}
	}
	if oreDef == nil {
		return &CommandResult{Messages: []string{"You chip away at the rock but find nothing."}}
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
		Val3:      purity, // purity percentage
	}
	player.Inventory = append(player.Inventory, ore)
	e.SavePlayer(ctx, player)

	oreName := e.getItemNounName(oreDef)
	qualityDesc := "poor"
	if purity > 70 {
		qualityDesc = "excellent"
	} else if purity > 50 {
		qualityDesc = "good"
	} else if purity > 30 {
		qualityDesc = "fair"
	}

	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You chip away at the rock and extract some %s looking %s!", qualityDesc, oreName)},
		RoomBroadcast: []string{fmt.Sprintf("%s mines some ore from the rock.", player.FirstName)},
		PlayerState:   player,
	}
}

// ---- SMELTING ----

func (e *GameEngine) doSmelt(ctx context.Context, player *Player, args []string) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil || !containsModifier(room.Modifiers, "FORGE") {
		return &CommandResult{Messages: []string{"You need to be at a forge to smelt ore."}}
	}

	smithSkill := player.Skills[8]
	if smithSkill < 1 {
		return &CommandResult{Messages: []string{"You have no training in Weaponsmithing."}}
	}

	// Find ore in inventory
	target := "ore"
	if len(args) > 0 {
		target = strings.ToLower(strings.Join(args, " "))
	}

	for i, ii := range player.Inventory {
		def := e.items[ii.Archetype]
		if def == nil || def.Type != "ORE" {
			continue
		}
		name := strings.ToLower(e.getItemNounName(def))
		if !strings.HasPrefix(name, target) && target != "ore" {
			continue
		}

		// Purity check: VAL3 = percentage chance of successful refinement
		purity := ii.Val3
		if purity <= 0 {
			purity = 30
		}

		// Remove ore
		player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)

		// Roll against purity + skill bonus
		smeltChance := purity + smithSkill*2
		if smeltChance > 95 {
			smeltChance = 95
		}

		if rand.Intn(100) >= smeltChance {
			e.SavePlayer(ctx, player)
			return &CommandResult{
				Messages:      []string{"You heat the ore in the forge, but it crumbles to useless slag."},
				RoomBroadcast: []string{fmt.Sprintf("%s works at the forge.", player.FirstName)},
				PlayerState:   player,
			}
		}

		// Success: create material item
		outputArch := def.Parameter2
		if outputArch <= 0 {
			outputArch = 1370 // default to generic metal
		}
		outputDef := e.items[outputArch]
		if outputDef == nil {
			e.SavePlayer(ctx, player)
			return &CommandResult{Messages: []string{"The ore refines but produces nothing useful."}}
		}

		material := InventoryItem{
			Archetype: outputArch,
			Val2:      ii.Val2, // transfer material properties
		}
		player.Inventory = append(player.Inventory, material)
		e.SavePlayer(ctx, player)

		matName := e.formatItemName(outputDef, material.Adj1, material.Adj2, material.Adj3)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You smelt the ore in the forge and produce some %s!", matName)},
			RoomBroadcast: []string{fmt.Sprintf("%s works at the forge, smelting ore.", player.FirstName)},
			PlayerState:   player,
		}
	}

	return &CommandResult{Messages: []string{"You don't have any ore to smelt."}}
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
		return &CommandResult{Messages: []string{"Craft what? Specify the item you want to make."}}
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
		remaining := player.RoundTimeExpiry.Sub(time.Now()).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You are still working... %.0f seconds remaining.", remaining+0.5)}}
	}

	switch player.CraftingStep {
	case 1: // Planned → need WORK <metal> to heat
		if len(args) == 0 {
			return &CommandResult{Messages: []string{"Work with what metal? e.g., WORK IRON"}}
		}
		metal := strings.ToLower(strings.Join(args, " "))

		// Find matching material in inventory
		materialIdx := -1
		materialAdj := 0
		for j, ii := range player.Inventory {
			mDef := e.items[ii.Archetype]
			if mDef == nil {
				continue
			}
			if mDef.Type == "MATERIAL" || mDef.Type == "MATERIAL2" {
				if mDef.Parameter2 == 8 || mDef.Parameter2 == 0 { // weaponsmithing material
					mName := strings.ToLower(e.getItemNounName(mDef))
					adjName := ""
					if mDef.Parameter1 > 0 {
						adjName = strings.ToLower(e.getAdjName(mDef.Parameter1))
					}
					if strings.Contains(mName, metal) || strings.Contains(adjName, metal) || strings.HasPrefix(metal, adjName) {
						materialIdx = j
						if mDef.Parameter1 > 0 {
							materialAdj = mDef.Parameter1
						}
						break
					}
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
		remaining := player.RoundTimeExpiry.Sub(time.Now()).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You are still working... %.0f seconds remaining.", remaining+0.5)}}
	}

	target := strings.ToLower(strings.Join(args, " "))

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

		if ii.State != "DAMAGED" {
			return &CommandResult{Messages: []string{"That doesn't need repair."}}
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
				Messages:      []string{fmt.Sprintf("[Success: %d%%, Roll %d] You are unable to repair the weapon.", chance, roll)},
				RoomBroadcast: []string{fmt.Sprintf("%s works at the forge, trying to repair a weapon.", player.FirstName)},
				PlayerState:   player,
			}
		}

		// Success: remove DAMAGED state
		player.Inventory[i].State = ""
		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages:      []string{fmt.Sprintf("[Success: %d%%, Roll %d] You carefully repair your %s.", chance, roll, itemName)},
			RoomBroadcast: []string{fmt.Sprintf("%s works at the forge, repairing a weapon.", player.FirstName)},
			PlayerState:   player,
		}
	}

	return &CommandResult{Messages: []string{"That doesn't need repair."}}
}

// ---- FORAGING ----

func (e *GameEngine) doForageReal(ctx context.Context, player *Player) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't forage here."}}
	}

	terrain := room.Terrain
	switch terrain {
	case "FOREST", "MOUNTAIN", "PLAIN", "SWAMP", "JUNGLE":
		// OK
	default:
		return &CommandResult{Messages: []string{"There is nothing to forage here."}}
	}

	// Check for forage definitions matching this terrain
	var candidates []gameworld.ForageDef
	for _, fd := range e.forageDefs {
		if strings.EqualFold(fd.Terrain, terrain) {
			candidates = append(candidates, fd)
		}
	}

	// If no ForageDefs loaded, use generic fallback
	if len(candidates) == 0 {
		return e.doForageFallback(ctx, player, terrain)
	}

	// Weighted random selection
	totalRatio := 0
	for _, fd := range candidates {
		totalRatio += fd.Ratio
	}
	if totalRatio <= 0 {
		return e.doForageFallback(ctx, player, terrain)
	}

	roll := rand.Intn(totalRatio)
	cumulative := 0
	for _, fd := range candidates {
		cumulative += fd.Ratio
		if roll < cumulative {
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
			e.SavePlayer(ctx, player)

			itemName := e.formatItemName(itemDef, item.Adj1, 0, 0)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You search the area and find some %s!", itemName)},
				RoomBroadcast: []string{fmt.Sprintf("%s forages in the area.", player.FirstName)},
				PlayerState:   player,
			}
		}
	}

	return &CommandResult{Messages: []string{"You search but find nothing useful."}}
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

func (e *GameEngine) doAnalyze(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Analyze what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))

	for _, ii := range player.Inventory {
		def := e.items[ii.Archetype]
		if def == nil {
			continue
		}
		name := strings.ToLower(e.getItemNounName(def))
		if !strings.HasPrefix(name, target) {
			continue
		}

		if def.Type == "ORE" {
			miningSkill := player.Skills[35]
			if miningSkill < 3 {
				return &CommandResult{Messages: []string{"You don't have enough mining skill to analyze this ore. (Need Mining 3+)"}}
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
			return &CommandResult{Messages: []string{fmt.Sprintf("You examine the ore carefully. It appears to be of %s quality. (Purity: %d%%)", desc, purity)}}
		}

		// Reagent analysis for alchemy
		if containsFlag(def.Flags, "REAGENT") {
			reagentTypes := map[int]string{
				1: "Power (mild)", 2: "Power (strong)", 3: "Power (very strong)",
				4: "Health", 5: "Harm", 6: "Body", 7: "Resist",
				8: "Enhancement", 9: "Misc (common)", 10: "Misc (uncommon)",
				11: "Misc (rare)", 12: "Mind", 13: "Protection",
			}
			rType := ii.Val5
			typeName := reagentTypes[rType]
			if typeName == "" {
				typeName = "unknown"
			}
			itemName := e.formatItemName(def, ii.Adj1, ii.Adj2, ii.Adj3)
			return &CommandResult{Messages: []string{fmt.Sprintf("You analyze %s. Alchemical properties: %s.", itemName, typeName)}}
		}

		return &CommandResult{Messages: []string{"You can't determine anything special about that."}}
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
	6: "A", 4: "B", 0: "C", 13: "D", 12: "E", 5: "F",
	8: "G", 9: "H", 10: "I", 11: "J",
}
var catalystFromVal5 = map[int]int{1: 1, 2: 2, 3: 3}

func (e *GameEngine) doBrew(ctx context.Context, player *Player, args []string) *CommandResult {
	if player.Skills[31] < 1 {
		return &CommandResult{Messages: []string{"You have no training in Alchemy."}}
	}
	if len(args) == 0 {
		// List known recipes
		var msgs []string
		msgs = append(msgs, "=== Alchemy Recipes (by level) ===")
		for _, r := range alchemyRecipes {
			if r.level <= player.Skills[31] {
				msgs = append(msgs, fmt.Sprintf("  Level %2d: %s (%s)", r.level, r.name, r.color))
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
		return &CommandResult{Messages: []string{"Brew in what? Usage: BREW <reagent> IN <flask/vial>"}}
	}

	// Find the reagent
	reagentIdx := -1
	var reagentItem *InventoryItem
	for i, ii := range player.Inventory {
		def := e.items[ii.Archetype]
		if def == nil {
			continue
		}
		if !containsFlag(def.Flags, "REAGENT") {
			continue
		}
		name := strings.ToLower(e.getItemNounName(def))
		if strings.HasPrefix(name, reagentTarget) || strings.Contains(name, reagentTarget) {
			reagentIdx = i
			reagentItem = &player.Inventory[i]
			break
		}
	}
	if reagentIdx < 0 || reagentItem == nil {
		return &CommandResult{Messages: []string{"You don't have that reagent."}}
	}

	// Find the container (flask, vial, bottle)
	containerIdx := -1
	for i, ii := range player.Inventory {
		def := e.items[ii.Archetype]
		if def == nil {
			continue
		}
		name := strings.ToLower(e.getItemNounName(def))
		if def.Type == "LIQCONTAINER" || def.Container == "IN" {
			if strings.HasPrefix(name, containerTarget) || strings.Contains(name, containerTarget) {
				containerIdx = i
				break
			}
		}
	}
	if containerIdx < 0 {
		return &CommandResult{Messages: []string{"You don't have a suitable container. You need a flask, vial, or bottle."}}
	}

	// Add reagent to the brew (track via container's Val fields)
	// Val3 = accumulated recipe code character, Val5 = number of ingredients added
	container := &player.Inventory[containerIdx]
	reagentType := reagentLetters[reagentItem.Val5]
	if reagentType == "" {
		reagentType = "H" // default to mild magic
	}

	// Consume reagent
	player.Inventory = append(player.Inventory[:reagentIdx], player.Inventory[reagentIdx+1:]...)
	// Fix index if container was after reagent
	if containerIdx > reagentIdx {
		containerIdx--
	}
	container = &player.Inventory[containerIdx]

	// Track ingredients in container's Val fields
	ingredients := container.Val5
	container.Val5 = ingredients + 1

	// Store reagent type codes: Val4 encodes up to 3 ingredient types
	// Simple encoding: multiply previous by 100 and add new
	container.Val4 = container.Val4*100 + reagentItem.Val5

	reagentDef := e.items[reagentItem.Archetype]
	reagentName := "the reagent"
	if reagentDef != nil {
		reagentName = e.getItemNounName(reagentDef)
	}

	if container.Val5 < 3 {
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You add %s to the brew. (%d/3 ingredients)", reagentName, container.Val5)},
			RoomBroadcast: []string{fmt.Sprintf("%s adds an ingredient to a bubbling brew.", player.FirstName)},
			PlayerState:   player,
		}
	}

	// 3 ingredients added — attempt to brew!
	// Decode the three ingredients
	code3 := container.Val4 % 100
	code2 := (container.Val4 / 100) % 100
	code1 := (container.Val4 / 10000) % 100

	letter1 := reagentLetters[code1]
	letter2 := reagentLetters[code2]
	letter3 := reagentLetters[code3]

	// Determine catalyst level from first ingredient
	catLevel := catalystFromVal5[code1]
	if catLevel == 0 {
		catLevel = 1
	}

	// Try to match a recipe
	for _, recipe := range alchemyRecipes {
		if recipe.catalyst != catLevel {
			continue
		}
		// Check if ingredients match (order doesn't matter for reagent1/reagent2)
		match := false
		if (letter2 == recipe.reagent1 && letter3 == recipe.reagent2) ||
			(letter2 == recipe.reagent2 && letter3 == recipe.reagent1) ||
			(letter1 == recipe.reagent1 && letter3 == recipe.reagent2) ||
			(letter1 == recipe.reagent2 && letter3 == recipe.reagent1) {
			match = true
		}
		_ = letter1 // catalyst is consumed too
		if !match {
			continue
		}

		// Check alchemy skill
		if player.Skills[31] < recipe.level {
			container.Val4 = 0
			container.Val5 = 0
			e.SavePlayer(ctx, player)
			return &CommandResult{
				Messages: []string{
					"The brew bubbles violently! It's a valid recipe, but beyond your current skill.",
					fmt.Sprintf("(Requires Alchemy level %d, you have %d)", recipe.level, player.Skills[31]),
				},
				PlayerState: player,
			}
		}

		// Success! Create potion
		container.Val3 = recipe.spellID // spell stored in container
		container.Val4 = 0
		container.Val5 = 0
		container.Val2 = 2 + rand.Intn(4) // 2-5 sips
		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("The brew shimmers magically! You have created a %s potion! (%s, %d sips)", recipe.name, recipe.color, container.Val2),
			},
			RoomBroadcast: []string{fmt.Sprintf("%s completes a potion that shimmers with magical energy!", player.FirstName)},
			PlayerState:   player,
		}
	}

	// No match — failed recipe
	container.Val4 = 0
	container.Val5 = 0
	e.SavePlayer(ctx, player)
	return &CommandResult{
		Messages:      []string{"A foul odor rises from the brew. The combination produces nothing useful."},
		RoomBroadcast: []string{fmt.Sprintf("%s's brew emits a foul odor.", player.FirstName)},
		PlayerState:   player,
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
