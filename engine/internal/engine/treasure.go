package engine

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// maxGenericTreasureItemNumber caps procedurally-generated loot (weapons, armor,
// jewelry, scrolls) to the low, tightly-curated item catalog. ANTI.SCR reserves
// placeholder stubs across the entire item number space, and later-loading area
// scripts fill in real content on top of those stubs throughout the whole range —
// including unique, story-specific items (e.g. item 1914, an elaborately described
// one-off helm, and item 1915, an amulet that teleports players to a special event
// room). Those items happen to match the generic Type/stat filters below, so without
// this cap they could be handed out as ordinary mob loot. Only items below this
// number are guaranteed to be the generic, repeatable catalog meant for randomized
// drops.
const maxGenericTreasureItemNumber = 210

// Potion container archetypes: bottle, flask, vial (all share the same 12-sip
// capacity, per potionContainerCapacity in containers.go).
var potionContainerArchetypes = []int{158, 159, 167}

// Material adjectives for flask/vial only — bottles are plain glass and instead
// let their liquid's own color/appearance show through (see potionVesselPhrase).
var potionMaterialAdjIDs = []int{244, 143, 172, 218, 77, 224, 251}

// Appearance adjectives describing a potion's liquid (color, smell, texture).
var potionLiquidAdjIDs = []int{
	260, 42, 1044, 135, 136, 19, 56, 58, 75, 62, 259, 225,
	358, 152, 29, 25, 151, 221, 246, 247, 316, 347, 464, 499,
}

// potionSpellIDs lists every spell that can be bound to a random potion (Val3).
// Their SpellDef.Level values already match the levels documented for potions.
var potionSpellIDs = []int{
	316, 317, 318, 313, 314, 315, 326, 334, 335, 343, 403,
	507, 508, 511, 513, 514, 515, 520, 521, 506, 518,
	102, 105, 133, 207, 208, 209, 210, 224, 225, 229, 234, 235,
	230, 232, 245, 248,
}

// nextRoomItemRef returns a ref value that is not already used by any item in the room.
// Room scripts use non-sequential refs (e.g., 0, 1, 2, 4), so len(room.Items) can
// collide with an existing ref. Using maxRef+1 is always safe.
func nextRoomItemRef(room *gameworld.Room) int {
	max := -1
	for _, ri := range room.Items {
		if ri.Ref > max {
			max = ri.Ref
		}
	}
	return max + 1
}

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
		ref := nextRoomItemRef(room)
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
		// Determine drop type — all tiers available from treasure level 1
		roll := rand.Intn(100)
		switch {
		case roll < 20:
			// Weapon drop
			if item := e.randomWeaponDrop(treasureLevel); item != nil {
				ref := nextRoomItemRef(room)
				item.Ref = ref
				room.Items = append(room.Items, *item)
				def := e.items[item.Archetype]
				if def != nil {
					name := e.formatItemName(def, item.Adj1, item.Adj2, item.Adj3, item.Extend)
					msgs = append(msgs, fmt.Sprintf("You see %s lies among the remains.", name))
				}
			}

		case roll < 40:
			// Scroll drop — spell scroll with learnable spell (available at all levels)
			if item := e.randomScrollDrop(treasureLevel); item != nil {
				ref := nextRoomItemRef(room)
				item.Ref = ref
				room.Items = append(room.Items, *item)
				msgs = append(msgs, "A scroll lies among the remains.")
			}

		case roll < 55:
			// Locked container
			if item := e.randomChestDrop(treasureLevel); item != nil {
				ref := nextRoomItemRef(room)
				item.Ref = ref
				room.Items = append(room.Items, *item)
				e.populateContainerLoot(roomNum, item.Ref, treasureLevel)
				msgs = append(msgs, "A container lies among the remains.")
			}

		case roll < 70:
			// Potion drop
			if item := e.randomPotionDrop(treasureLevel); item != nil {
				ref := nextRoomItemRef(room)
				item.Ref = ref
				room.Items = append(room.Items, *item)
				def := e.items[item.Archetype]
				if def != nil {
					name := e.formatItemName(def, item.Adj1, item.Adj2, item.Adj3, item.Extend)
					msgs = append(msgs, fmt.Sprintf("You see %s among the remains.", name))
				}
			}

		case roll < 88:
			// Armor drop
			if item := e.randomArmorDrop(treasureLevel); item != nil {
				ref := nextRoomItemRef(room)
				item.Ref = ref
				room.Items = append(room.Items, *item)
				def := e.items[item.Archetype]
				if def != nil {
					name := e.formatItemName(def, item.Adj1, item.Adj2, item.Adj3, item.Extend)
					msgs = append(msgs, fmt.Sprintf("Some %s lies among the remains.", name))
				}
			}

		default:
			// Jewelry drop (remaining 25% of item drops, requires treasure >= 5)
			if treasureLevel >= 5 {
				if item := e.randomJewelryDrop(treasureLevel); item != nil {
					ref := nextRoomItemRef(room)
					item.Ref = ref
					room.Items = append(room.Items, *item)
					def := e.items[item.Archetype]
					if def != nil {
						name := e.formatItemName(def, item.Adj1, item.Adj2, item.Adj3, item.Extend)
						if item.Val3 > 0 {
							msgs = append(msgs, fmt.Sprintf("A %s glimmers with a faint magical light among the remains.", name))
						} else {
							msgs = append(msgs, fmt.Sprintf("A %s lies among the remains.", name))
						}
					}
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
		if num >= maxGenericTreasureItemNumber {
			continue
		}
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
		if num >= maxGenericTreasureItemNumber {
			continue
		}
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
		// Fallback: find any SCROLL type item in the generic catalog
		for num, def := range e.items {
			if num < maxGenericTreasureItemNumber && def.Type == "SCROLL" && def.Weight < 1000 {
				scrollArch = num
				break
			}
		}
	}
	if e.items[scrollArch] == nil {
		return nil
	}

	// Pick a spell appropriate for treasure level
	maxSpellLevel := int(math.Ceil(float64(treasureLevel) / 1.5))
	if maxSpellLevel < 1 {
		maxSpellLevel = 1
	}
	if maxSpellLevel > 30 {
		maxSpellLevel = 30
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
	ri := &gameworld.RoomItem{
		Archetype: scrollArch,
		Val3:      spell.ID,
	}
	adjWord := scrollAdjectiveWords[rand.Intn(len(scrollAdjectiveWords))]
	if adjID := e.adjByName(adjWord); adjID != 0 {
		ri.Adj1 = adjID
	}
	return ri
}

// randomPotionDrop creates a random potion (a closed bottle, flask, or vial holding
// 2-10 sips of a liquid bound to a random spell). The spell's level is capped by
// half the treasure level (e.g. a treasure-10 monster can drop up to a level-5 spell).
// Flask/vial get a random material adjective; bottles show their liquid's own color
// instead. The container starts closed — OPEN reveals the liquid's appearance.
func (e *GameEngine) randomPotionDrop(treasureLevel int) *gameworld.RoomItem {
	arch := potionContainerArchetypes[rand.Intn(len(potionContainerArchetypes))]
	if e.items[arch] == nil {
		return nil
	}
	item := &gameworld.RoomItem{Archetype: arch}

	if arch == 159 || arch == 167 { // flask or vial — opaque, gets a material adjective
		item.Adj1 = potionMaterialAdjIDs[rand.Intn(len(potionMaterialAdjIDs))]
	}
	item.Val4 = potionLiquidAdjIDs[rand.Intn(len(potionLiquidAdjIDs))]

	sips := (1 + rand.Intn(5)) + (1 + rand.Intn(5)) // 2-10
	if sips > potionContainerCapacity {
		sips = potionContainerCapacity
	}
	item.Val2 = sips

	maxLevel := treasureLevel / 2
	if maxLevel < 1 {
		maxLevel = 1
	}
	targetLevel := 1 + rand.Intn(maxLevel)

	var candidates []int
	for _, id := range potionSpellIDs {
		if sp := FindSpellByID(id); sp != nil && sp.Level <= targetLevel {
			candidates = append(candidates, id)
		}
	}
	if len(candidates) == 0 {
		for _, id := range potionSpellIDs {
			if sp := FindSpellByID(id); sp != nil && sp.Level == 1 {
				candidates = append(candidates, id)
			}
		}
	}
	if len(candidates) > 0 {
		item.Val3 = candidates[rand.Intn(len(candidates))]
	}

	return item
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
	if trapChance > 50 { trapChance = 50 }
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

        FixChestInitialState(item, treasureLevel)

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
