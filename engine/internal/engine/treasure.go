package engine

import (
	"fmt"
	"math/rand"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// generateTreasure creates loot items in a room.
// treasureLevel = monster's TREASURE value (0 = nothing, 1-127 = increasing rewards).
// Returns descriptions of the items/coins generated.
func (e *GameEngine) generateTreasure(roomNum int, treasureLevel int) []string {
	if treasureLevel <= 0 {
		return nil
	}

	room := e.rooms[roomNum]
	if room == nil {
		return nil
	}

	var found []string

	// ---- Coin drops (always if treasure > 0) ----
	copperBase := treasureLevel * 5
	coins := copperBase + rand.Intn(copperBase+1)

	if coins > 0 {
		// Drop as a money item in the room.
		ref := len(room.Items)

		room.Items = append(room.Items, gameworld.RoomItem{
			Ref:       ref,
			Archetype: 0, // special: money on ground
			Val1:      coins,
			State:     "MONEY",
		})

		gold := coins / 100
		silver := (coins % 100) / 10
		copper := coins % 10

		if gold > 0 {
			if gold == 1 {
				found = append(found, "1 gold coin drops to the ground")
			} else {
				found = append(found, fmt.Sprintf("%d gold coins scatter on the ground", gold))
			}
		}

		if silver > 0 {
			if silver == 1 {
				found = append(found, "1 silver coin drops to the ground")
			} else {
				found = append(found, fmt.Sprintf("%d silver coins scatter on the ground", silver))
			}
		}

		if copper > 0 {
			if copper == 1 {
				found = append(found, "1 copper coin drops to the ground")
			} else {
				found = append(found, fmt.Sprintf("%d copper coins scatter on the ground", copper))
			}
		}
	}

	// ---- Item drops (chance scales with treasure level) ----
	// Base drop chance: 10% + treasureLevel/2, capped at 60%.
	dropChance := 10 + treasureLevel/2

	if dropChance > 60 {
		dropChance = 60
	}

	//dropChance = 100 //test lets drop things all the ime

	if rand.Intn(100) < dropChance {
		// Determine drop type — all tiers available from treasure level 1.
		roll := rand.Intn(100)

		switch {
		case roll < 20:
			// Weapon drop
			if item := e.randomWeaponDrop(treasureLevel); item != nil {
				item.Ref = len(room.Items)
				room.Items = append(room.Items, *item)

				if def := e.items[item.Archetype]; def != nil {
					name := e.formatItemName(
						def,
						item.Adj1,
						item.Adj2,
						item.Adj3,
					)

					found = append(found, fmt.Sprintf("%s drops to the ground", name))
				}
			}

		case roll < 40:
			// Scroll drop
			if item := e.randomScrollDrop(treasureLevel); item != nil {
				item.Ref = len(room.Items)
				room.Items = append(room.Items, *item)

				if spell := FindSpellByID(item.Val3); spell != nil {
					found = append(
						found,
						fmt.Sprintf("you see a scroll of %s", spell.Name),
					)
				} else {
					found = append(found, "you see a scroll ")
				}
			}

		case roll < 55:
			// Locked container
			if item := e.randomChestDrop(treasureLevel); item != nil {
				ref := len(room.Items)

				item.Ref = ref
				item.State = "LOCKED"

				room.Items = append(room.Items, *item)

				// Generate the contents now. The container is already
				// populated before anyone opens it.
				e.generateChestContents(
					room,
					ref,
					treasureLevel,
				)

				if def := e.items[item.Archetype]; def != nil {
					name := e.formatItemName(
						def,
						item.Adj1,
						item.Adj2,
						item.Adj3,
					)

					found = append(found, fmt.Sprintf("%s thumps to the ground", name))
				}
			}

		case roll < 75:
			// Armor drop
			if item := e.randomArmorDrop(treasureLevel); item != nil {
				item.Ref = len(room.Items)
				room.Items = append(room.Items, *item)

				if def := e.items[item.Archetype]; def != nil {
					name := e.formatItemName(
						def,
						item.Adj1,
						item.Adj2,
						item.Adj3,
					)

					found = append(found, fmt.Sprintf("%s clatters to the ground", name))
				}
			}

		case roll < 100:
			// Gem drop
			if item := e.randomGemDrop(treasureLevel); item != nil {
				item.Ref = len(room.Items)
				room.Items = append(room.Items, *item)

				if def := e.items[item.Archetype]; def != nil {
					name := e.formatItemName(
						def,
						item.Adj1,
						item.Adj2,
						item.Adj3,
					)

					found = append(found, fmt.Sprintf("you hear a ping as a %s drops to the ground", name))

				}
			}
		}

	}

	// ---- Rare magic item chance (treasure >= 20, 5% chance) ----
	if treasureLevel >= 20 && rand.Intn(100) < 5 {

		// Magic bonus on the dropped weapon/armor.
		// This is handled by the enchantment on items already dropped.
	}

	return found
}

// randomWeaponDrop selects a random weapon appropriate for the treasure level.
func (e *GameEngine) randomWeaponDrop(treasureLevel int) *gameworld.RoomItem {
	// Collect weapons within a damage range appropriate for treasure level
	maxDmg := treasureLevel / 2
	if maxDmg < 3 {
		maxDmg = 3
	}
	if maxDmg > 30 {
		maxDmg = 30
	}

	var candidates []int
	for num, def := range e.items {
		if !isWeapon(def.Type) {
			continue
		}
		if def.Parameter1 <= 0 || def.Parameter1 > maxDmg {
			continue
		}
		if def.Weight >= 1000 {
			continue // immovable
		}
		candidates = append(candidates, num)
	}
	if len(candidates) == 0 {
		return nil
	}

	chosen := candidates[rand.Intn(len(candidates))]
	item := &gameworld.RoomItem{
		Archetype: chosen,
	}

	// Chance for magic bonus (higher treasure = higher chance and bonus)
	if treasureLevel >= 15 && rand.Intn(100) < treasureLevel/3 {
		item.Val2 = rand.Intn(treasureLevel/10+1) + 1 // +1 to +N magic bonus
	}

	// Chance for premium material adjective
	if treasureLevel >= 30 && rand.Intn(100) < 15 {
		premiumAdjs := []int{5, 434, 577} // alzyron, adamantine, uquart
		item.Adj1 = premiumAdjs[rand.Intn(len(premiumAdjs))]
	}

	return item
}

// randomArmorDrop selects random armor appropriate for treasure level.
func (e *GameEngine) randomArmorDrop(treasureLevel int) *gameworld.RoomItem {
	maxAC := treasureLevel
	if maxAC > 50 {
		maxAC = 50
	}

	var candidates []int
	for num, def := range e.items {
		if def.Type != "ARMOR" {
			continue
		}
		if def.Parameter1 <= 0 || def.Parameter1 > maxAC {
			continue
		}
		if def.Weight >= 1000 {
			continue
		}
		candidates = append(candidates, num)
	}
	if len(candidates) == 0 {
		return nil
	}

	chosen := candidates[rand.Intn(len(candidates))]
	item := &gameworld.RoomItem{
		Archetype: chosen,
	}

	// Chance for magic bonus
	if treasureLevel >= 20 && rand.Intn(100) < treasureLevel/4 {
		item.Val2 = rand.Intn(treasureLevel/15+1) + 1
	}

	return item
}

// randomScrollDrop creates a spell scroll with a learnable spell.
func (e *GameEngine) randomScrollDrop(treasureLevel int) *gameworld.RoomItem {
	// Find scroll item archetype (item 168)
	scrollArch := 168
	if e.items[scrollArch] == nil {
		// Fallback: find any SCROLL type item
		for num, def := range e.items {
			if def.Type == "SCROLL" && def.Weight < 1000 {
				scrollArch = num
				break
			}
		}
	}
	if e.items[scrollArch] == nil {
		return nil
	}

	// Pick a spell appropriate for treasure level
	maxSpellLevel := treasureLevel / 3
	if maxSpellLevel < 1 {
		maxSpellLevel = 1
	}
	if maxSpellLevel > 25 {
		maxSpellLevel = 25
	}

	var candidates []SpellDef
	for _, sp := range spellRegistry {
		if sp.Level <= maxSpellLevel && sp.Effect != "" {
			candidates = append(candidates, sp)
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	spell := candidates[rand.Intn(len(candidates))]
	return &gameworld.RoomItem{
		Archetype: scrollArch,
		Val3:      spell.ID, // spell stored on scroll
	}
}

// randomChestDrop creates a locked chest (possibly trapped) with coin contents.
func (e *GameEngine) randomChestDrop(treasureLevel int) *gameworld.RoomItem {
	// Find a chest/strongbox/coffer item archetype
	chestArch := 0
	for num, def := range e.items {
		noun := e.nouns[def.NameID]
		if (noun == "chest" || noun == "strongbox" || noun == "coffer" || noun == "lockbox" || noun == "casket") && def.Weight < 1000 {
			chestArch = num
			break
		}
	}
	// Also try any LOCKABLE container
	if chestArch == 0 {
		for num, def := range e.items {
			if def.Container != "" && containsFlag(def.Flags, "LOCKABLE") && def.Weight < 1000 {
				chestArch = num
				break
			}
		}
	}
	if chestArch == 0 {
		return nil
	}

	// Lock difficulty scales with treasure level but starts easy for rogues
	lockDiff := treasureLevel/2 + rand.Intn(10) + 5
	item := &gameworld.RoomItem{
		Archetype: chestArch,
		State:     "LOCKED",
		Val1:      lockDiff,
	}

	// Chance for trap (scales with treasure level: 10% at level 1, up to 50% at high levels)
	trapChance := 10 + treasureLevel/2
	if trapChance > 50 {
		trapChance = 50
	}
	if rand.Intn(100) < trapChance {
		trapTypes := []int{1, 2, 4, 5} // needle, gas, blades, moderate needle
		if treasureLevel >= 30 {
			trapTypes = append(trapTypes, 7, 8, 9, 12) // major needle, explosive, acid, nerve gas
		}
		if treasureLevel >= 50 {
			trapTypes = append(trapTypes, 13, 1001, 3001, 5001) // lethal needle, glyphs
		}
		item.Val4 = trapTypes[rand.Intn(len(trapTypes))]
	}

	return item
}

func (e *GameEngine) randomGemDrop(treasureLevel int) *gameworld.RoomItem {
	var candidates []int

	for i, md := range e.mineDefs {
		def := e.items[md.ItemNum]
		if def == nil {
			continue
		}

		// MINEDEF also contains ore/material entries.
		if def.Type == "ORE" || def.Type == "MATERIAL" {
			continue
		}

		switch {
		case treasureLevel < 20:
			if md.Grade != "C" {
				continue
			}

		case treasureLevel < 40:
			if md.Grade != "C" && md.Grade != "B" {
				continue
			}

		default:
			if md.Grade != "C" &&
				md.Grade != "B" &&
				md.Grade != "A" {
				continue
			}
		}

		candidates = append(candidates, i)
	}

	if len(candidates) == 0 {
		return nil
	}

	md := e.mineDefs[candidates[rand.Intn(len(candidates))]]

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

	item := &gameworld.RoomItem{
		Archetype: md.ItemNum,
		Val1:      md.Value,
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

func (e *GameEngine) generateChestContents(room *gameworld.Room, chestRef int, treasureLevel int) {
	addItem := func(item *gameworld.RoomItem) {
		if item == nil {
			return
		}

		item.Ref = chestRef
		item.IsPut = true
		item.PutIn = chestRef

		room.Items = append(room.Items, *item)
	}

	addMoney := func(denomination int, amount int) {
		if amount <= 0 {
			return
		}

		moneyArch := e.findBaseMoneyArchetype(denomination)
		if moneyArch == 0 {
			return
		}

		room.Items = append(
			room.Items,
			gameworld.RoomItem{
				Ref:       chestRef,
				Archetype: moneyArch,
				Val1:      amount,
				IsPut:     true,
				PutIn:     chestRef,
			},
		)
	}

	// Always include a better coin reward than loose treasure.
	copperBase := treasureLevel * 15

	if copperBase < 1 {
		copperBase = 1
	}

	coins := copperBase + rand.Intn(copperBase+1)

	gold := coins / 100
	remaining := coins % 100

	silver := remaining / 10
	copper := remaining % 10

	// <-- Put these three lines HERE
	addMoney(MoneyGold, gold)
	addMoney(MoneySilver, silver)
	addMoney(MoneyCopper, copper)

	// Chests always contain at least one useful item.
	switch rand.Intn(3) {
	case 0:
		addItem(e.randomWeaponDrop(treasureLevel))
	case 1:
		addItem(e.randomArmorDrop(treasureLevel))
	case 2:
		addItem(e.randomScrollDrop(treasureLevel))
	case 3:
		addItem(e.randomGemDrop(treasureLevel))
	}

	// Better chests have a chance at another useful item.
	if treasureLevel >= 20 && rand.Intn(100) < 50 {
		switch rand.Intn(3) {
		case 0:
			addItem(e.randomWeaponDrop(treasureLevel))
		case 1:
			addItem(e.randomArmorDrop(treasureLevel))
		case 2:
			addItem(e.randomScrollDrop(treasureLevel))
		case 3:
			addItem(e.randomGemDrop(treasureLevel))
		}
	}
}

func (e *GameEngine) findBaseMoneyArchetype(denomination int) int {
	for num, def := range e.items {
		if def.Type != "MONEY" {
			continue
		}

		// Base currency only; regional currencies use PARAMETER2 > 0.
		if def.Parameter2 != 0 {
			continue
		}

		if def.Parameter1 == denomination {
			return num
		}
	}

	return 0
}

func joinParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return fmt.Sprintf("%s and %s", parts[0], parts[len(parts)-1])
}

const (
	MoneyGold   = 1
	MoneySilver = 2
	MoneyCopper = 3
)
