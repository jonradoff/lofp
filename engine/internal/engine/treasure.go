package engine

import (
	"fmt"
	"math/rand"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// generateTreasure creates loot items in a room when a monster dies.
// treasureLevel = monster's TREASURE value (0 = nothing, 1-127 = increasing rewards).
// Returns messages describing what dropped.
func (e *GameEngine) generateTreasure(roomNum int, treasureLevel int) []string {
	if treasureLevel <= 0 {
		return nil
	}
	room := e.rooms[roomNum]
	if room == nil {
		return nil
	}

	var msgs []string

	// ---- Coin drops (always if treasure > 0) ----
	copperBase := treasureLevel * 5
	coins := copperBase + rand.Intn(copperBase+1)
	if coins > 0 {
		// Drop as a money item in the room
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
		var parts []string
		if gold > 0 {
			parts = append(parts, fmt.Sprintf("%d gold", gold))
		}
		if silver > 0 {
			parts = append(parts, fmt.Sprintf("%d silver", silver))
		}
		if copper > 0 {
			parts = append(parts, fmt.Sprintf("%d copper", copper))
		}
		if len(parts) > 0 {
			msgs = append(msgs, fmt.Sprintf("Some coins scatter on the ground. (%s)", joinParts(parts)))
		}
	}

	// ---- Item drops (chance scales with treasure level) ----
	// Base drop chance: 10% + treasureLevel/2, capped at 60%
	dropChance := 10 + treasureLevel/2
	if dropChance > 60 {
		dropChance = 60
	}

	if rand.Intn(100) < dropChance {
		// Determine drop type based on treasure level
		roll := rand.Intn(100)
		switch {
		case roll < 25 && treasureLevel >= 5:
			// Weapon drop — find a level-appropriate weapon from the item database
			if item := e.randomWeaponDrop(treasureLevel); item != nil {
				ref := len(room.Items)
				item.Ref = ref
				room.Items = append(room.Items, *item)
				def := e.items[item.Archetype]
				if def != nil {
					name := e.formatItemName(def, item.Adj1, item.Adj2, item.Adj3)
					msgs = append(msgs, fmt.Sprintf("A %s lies among the remains.", name))
				}
			}

		case roll < 40 && treasureLevel >= 10:
			// Scroll drop — spell scroll with learnable spell
			if item := e.randomScrollDrop(treasureLevel); item != nil {
				ref := len(room.Items)
				item.Ref = ref
				room.Items = append(room.Items, *item)
				def := e.items[item.Archetype]
				if def != nil {
					msgs = append(msgs, "A scroll lies among the remains.")
				}
			}

		case roll < 55 && treasureLevel >= 15:
			// Locked chest — container with random contents
			if item := e.randomChestDrop(treasureLevel); item != nil {
				ref := len(room.Items)
				item.Ref = ref
				room.Items = append(room.Items, *item)
				msgs = append(msgs, "A small chest lies among the remains.")
			}

		case roll < 70 && treasureLevel >= 3:
			// Armor drop
			if item := e.randomArmorDrop(treasureLevel); item != nil {
				ref := len(room.Items)
				item.Ref = ref
				room.Items = append(room.Items, *item)
				def := e.items[item.Archetype]
				if def != nil {
					name := e.formatItemName(def, item.Adj1, item.Adj2, item.Adj3)
					msgs = append(msgs, fmt.Sprintf("Some %s lies among the remains.", name))
				}
			}
		}
	}

	// ---- Rare magic item chance (treasure >= 20, 5% chance) ----
	if treasureLevel >= 20 && rand.Intn(100) < 5 {
		// Magic bonus on the dropped weapon/armor
		// This is handled by the enchantment on items already dropped
	}

	return msgs
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
	// Find a chest item archetype
	chestArch := 0
	for num, def := range e.items {
		noun := e.nouns[def.NameID]
		if noun == "chest" && def.Weight < 1000 && def.Container != "" {
			chestArch = num
			break
		}
	}
	if chestArch == 0 {
		// Create a generic locked container
		return nil // no suitable chest item found in database
	}

	item := &gameworld.RoomItem{
		Archetype: chestArch,
		State:     "LOCKED",
		Val1:      treasureLevel + rand.Intn(20), // lock difficulty
	}

	// Chance for trap (higher treasure = more dangerous traps)
	if rand.Intn(100) < 40 {
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

func joinParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return fmt.Sprintf("%s and %s", parts[0], parts[len(parts)-1])
}
