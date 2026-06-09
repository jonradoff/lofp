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

// doItemInteraction handles verbs like PULL, PUSH, TURN, RUB, TAP, TOUCH, SEARCH, DIG.
// These run IFPREVERB scripts on the target item. If no script handles it, a default message is shown.
func (e *GameEngine) doItemInteraction(ctx context.Context, player *Player, verb string, args []string) *CommandResult {
	verbLower := strings.ToLower(verb)
	if len(args) == 0 {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s what?", strings.Title(verbLower))}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}

	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			if skip > 0 { skip--; continue }
			result := &CommandResult{}
			// Run root-level IFVAR blocks on the item (fire as preamble for any verb)
			sc0 := e.RunItemScripts(player, room, &room.Items[i], itemDef)
			result.Messages = append(result.Messages, sc0.Messages...)
			result.RoomBroadcast = append(result.RoomBroadcast, sc0.RoomMsgs...)
			result.GMBroadcast = append(result.GMBroadcast, sc0.GMMsgs...)
			// Run IFPREVERB scripts (item-level + room specific-ref + room -1 catch-all)
			sc := e.RunPreverbScripts(player, room, verb, &room.Items[i], itemDef)
			result.Messages = append(result.Messages, sc.Messages...)
			result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
			result.GMBroadcast = append(result.GMBroadcast, sc.GMMsgs...)
			// Run IFVERB scripts (item-level + room specific-ref + room -1 catch-all)
			sc2 := e.RunVerbScripts(player, room, verb, &room.Items[i], itemDef)
			result.Messages = append(result.Messages, sc2.Messages...)
			result.RoomBroadcast = append(result.RoomBroadcast, sc2.RoomMsgs...)
			result.GMBroadcast = append(result.GMBroadcast, sc2.GMMsgs...)
			if (sc0.Blocked || sc.Blocked || sc2.Blocked) && sc0.MoveTo == 0 && sc.MoveTo == 0 && sc2.MoveTo == 0 {
				if len(result.Messages) == 0 {
					result.Messages = []string{"You can't do that."}
				}
				return result
			}
			moveTo := sc0.MoveTo
			if sc.MoveTo > 0 { moveTo = sc.MoveTo }
			if sc2.MoveTo > 0 { moveTo = sc2.MoveTo }
			if moveTo > 0 {
				dest := e.rooms[moveTo]
				if dest != nil {
					oldRoom := player.RoomNumber
					player.RoomNumber = moveTo
					e.SavePlayer(ctx, player)
					lookResult := e.doLook(player)
					result.Messages = append(result.Messages, lookResult.Messages...)
					result.RoomName = lookResult.RoomName
					result.RoomDesc = lookResult.RoomDesc
					result.Exits = lookResult.Exits
					result.Items = lookResult.Items
					result.OldRoom = oldRoom
					result.OldRoomMsg = []string{fmt.Sprintf("%s leaves.", player.FirstName)}
					result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s arrives.", player.FirstName))
					e.applyEntryScripts(ctx, player, dest, result)
				}
			}
			if len(result.Messages) == 0 {
				itemName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
				result.Messages = []string{fmt.Sprintf("You %s %s. Nothing happens.", verbLower, itemName)}
			}
			return result
		}
	}

	// Check all player items: inventory, worn, and wielded
	allPlayerItems := make([]InventoryItem, 0, len(player.Inventory)+len(player.Worn)+1)
	allPlayerItems = append(allPlayerItems, player.Inventory...)
	allPlayerItems = append(allPlayerItems, player.Worn...)
	if player.Wielded != nil {
		allPlayerItems = append(allPlayerItems, *player.Wielded)
	}
	for _, ii := range allPlayerItems {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) || matchesTarget(name, target, e.getAdjName(ii.Adj3)) {
			if skip > 0 { skip--; continue }
			// Create a temporary RoomItem for script context (Ref=-1 = inventory item)
			tempRI := gameworld.RoomItem{Ref: -1, Archetype: ii.Archetype,
				Adj1: ii.Adj1, Adj2: ii.Adj2, Adj3: ii.Adj3,
				Val1: ii.Val1, Val2: ii.Val2, Val3: ii.Val3, Val4: ii.Val4, Val5: ii.Val5}
			result := &CommandResult{}
			// Run root IFVAR blocks (preamble for any verb)
			sc0 := e.RunItemScripts(player, room, &tempRI, itemDef)
			result.Messages = append(result.Messages, sc0.Messages...)
			result.RoomBroadcast = append(result.RoomBroadcast, sc0.RoomMsgs...)
			// Run IFPREVERB blocks (item-level only; room scripts excluded for inventory items)
			sc1 := e.RunPreverbScripts(player, room, verb, &tempRI, itemDef)
			result.Messages = append(result.Messages, sc1.Messages...)
			result.RoomBroadcast = append(result.RoomBroadcast, sc1.RoomMsgs...)
			// Run IFVERB blocks
			sc2 := e.RunVerbScripts(player, room, verb, &tempRI, itemDef)
			result.Messages = append(result.Messages, sc2.Messages...)
			result.RoomBroadcast = append(result.RoomBroadcast, sc2.RoomMsgs...)
			blocked := sc0.Blocked || sc1.Blocked || sc2.Blocked
			if blocked {
				if len(result.Messages) == 0 {
					result.Messages = []string{"You can't do that."}
				}
				return result
			}
			if len(result.Messages) > 0 || len(result.RoomBroadcast) > 0 {
				return result
			}
			itemName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			return &CommandResult{Messages: []string{fmt.Sprintf("You %s %s. Nothing happens.", verbLower, itemName)}}
		}
	}

	// Fall back to emote if the verb has one (e.g., RUB, TAP, TOUCH targeting a player)
	if _, hasEmote := emoteTable[verb]; hasEmote {
		return e.processEmote(player, verb, args)
	}

	return &CommandResult{Messages: []string{"You don't see that here."}}
}

func (e *GameEngine) doGet(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Get what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}

	for i, ri := range room.Items {
		// Handle coin piles (State == "MONEY", may have Archetype 0)
		if ri.State == "MONEY" && (target == "coins" || target == "money" || target == "coin" || target == "gold" || target == "silver" || target == "copper") {
			coins := ri.Val1
			if coins <= 0 { coins = 1 }
			room.Items = append(room.Items[:i], room.Items[i+1:]...)
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_remove", ItemRef: ri.Ref})
			player.Copper += coins
			player.Silver += player.Copper / 10
			player.Copper = player.Copper % 10
			player.Gold += player.Silver / 10
			player.Silver = player.Silver % 10
			e.SavePlayer(ctx, player)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You pick up %d coins.", coins)},
				RoomBroadcast: []string{fmt.Sprintf("%s picks up some coins.", player.FirstName)},
			}
		}

		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		if itemDef.Weight >= 1000 {
			continue // immovable
		}
		if isPortal(itemDef.Type) {
			continue
		}
		if containsFlag(itemDef.Flags, "FIXED") || itemDef.Type == "MANUSCRIPT" {
			continue // can't pick up fixed items or manuscripts
		}

		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			if skip > 0 { skip--; continue }

			// MONEY items auto-convert to currency
			if itemDef.Type == "MONEY" || ri.State == "MONEY" {
				coins := ri.Val1
				if coins <= 0 { coins = 1 }
				room.Items = append(room.Items[:i], room.Items[i+1:]...)
				e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_remove", ItemRef: ri.Ref})
				player.Copper += coins
				// Auto-convert up
				player.Silver += player.Copper / 10
				player.Copper = player.Copper % 10
				player.Gold += player.Silver / 10
				player.Silver = player.Silver % 10
				e.SavePlayer(ctx, player)
				return &CommandResult{
					Messages:      []string{fmt.Sprintf("You pick up %d coins.", coins)},
					RoomBroadcast: []string{fmt.Sprintf("%s picks up some coins.", player.FirstName)},
				}
			}

			// Add to inventory
			newItem := InventoryItem{
				Archetype: ri.Archetype,
				Adj1: ri.Adj1, Adj2: ri.Adj2, Adj3: ri.Adj3,
				Val1: ri.Val1, Val2: ri.Val2, Val3: ri.Val3, Val4: ri.Val4, Val5: ri.Val5,
				State: ri.State,
			}
			if isContainerDef(itemDef) {
				newItem.Contents = e.roomContainerGet(player.RoomNumber, ri.Ref)
				e.roomContainerDelete(player.RoomNumber, ri.Ref)
			}
			player.Inventory = append(player.Inventory, newItem)

			// Remove from room
			room.Items = append(room.Items[:i], room.Items[i+1:]...)
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_remove", ItemRef: ri.Ref})
			e.SavePlayer(ctx, player)
			fullName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You pick up %s.", fullName)},
				RoomBroadcast: []string{fmt.Sprintf("%s picks up %s.", player.FirstName, fullName)},
			}
		}
	}

	return &CommandResult{Messages: []string{"You don't see that here."}}
}

func (e *GameEngine) doDrop(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Drop what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}

	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			if skip > 0 { skip--; continue }

			// Check room IFPREVERB DROP scripts before executing (e.g., item falls into fissure).
			tempRI := gameworld.RoomItem{Ref: -1, Archetype: ii.Archetype,
				Adj1: ii.Adj1, Adj2: ii.Adj2, Adj3: ii.Adj3,
				Val1: ii.Val1, Val2: ii.Val2, Val3: ii.Val3, Val4: ii.Val4, Val5: ii.Val5}
			sc := &ScriptContext{Player: player, Room: room, Engine: e, ItemRef: &tempRI, ItemDef: itemDef}
			// Room -1 catch-all IFPREVERB DROP scripts (fissure, pit, etc.)
			for _, block := range room.Scripts {
				if block.Type == "IFPREVERB" && len(block.Args) == 2 &&
					strings.ToUpper(block.Args[0]) == "DROP" && block.Args[1] == "-1" {
					sc.execBlock(block)
				}
			}
			// Item-level IFPREVERB DROP scripts (items that can't be dropped)
			for _, block := range itemDef.Scripts {
				if block.Type == "IFPREVERB" && len(block.Args) >= 1 &&
					strings.ToUpper(block.Args[0]) == "DROP" {
					if len(block.Args) < 2 || block.Args[1] == "-1" {
						sc.execBlock(block)
					}
				}
			}
			if sc.Blocked {
				result := &CommandResult{Messages: sc.Messages, RoomBroadcast: sc.RoomMsgs, GMBroadcast: sc.GMMsgs}
				if len(result.Messages) == 0 {
					result.Messages = []string{"You can't drop that here."}
				}
				return result
			}
			// If the script removed the item from inventory (e.g., REMOVEITEM without CLEARVERB),
			// return the script messages without executing the normal drop.
			stillPresent := false
			for _, inv := range player.Inventory {
				if inv.Archetype == ii.Archetype {
					stillPresent = true
					break
				}
			}
			if !stillPresent {
				return &CommandResult{Messages: sc.Messages, RoomBroadcast: sc.RoomMsgs, GMBroadcast: sc.GMMsgs}
			}

			droppedItem := gameworld.RoomItem{
				Ref:       len(room.Items),
				Archetype: ii.Archetype,
				Adj1: ii.Adj1, Adj2: ii.Adj2, Adj3: ii.Adj3,
				Val1: ii.Val1, Val2: ii.Val2, Val3: ii.Val3, Val4: ii.Val4, Val5: ii.Val5,
				State: ii.State,
			}
			if isContainerDef(itemDef) {
				e.roomContainerSet(player.RoomNumber, droppedItem.Ref, ii.Contents)
			}
			room.Items = append(room.Items, droppedItem)
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_add", Item: &droppedItem})
			player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
			e.SavePlayer(ctx, player)
			fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You drop %s.", fullName)},
				RoomBroadcast: []string{fmt.Sprintf("%s drops %s.", player.FirstName, fullName)},
			}
		}
	}

	return &CommandResult{Messages: []string{"You aren't carrying that."}}
}

func (e *GameEngine) doInventory(player *Player) *CommandResult {
	var msgs []string
	msgs = append(msgs, "You are carrying:")
	if len(player.Inventory) == 0 && len(player.Worn) == 0 && player.Wielded == nil {
		msgs = append(msgs, "  Nothing.")
		return &CommandResult{Messages: msgs}
	}

	if player.Wielded != nil {
		itemDef := e.items[player.Wielded.Archetype]
		if itemDef != nil {
			name := e.formatItemName(itemDef, player.Wielded.Adj1, player.Wielded.Adj2, player.Wielded.Adj3)
			msgs = append(msgs, fmt.Sprintf("  %s (wielded)", name))
		}
	}

	for _, ii := range player.Worn {
		itemDef := e.items[ii.Archetype]
		if itemDef != nil {
			name := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			msgs = append(msgs, fmt.Sprintf("  %s (worn)", name))
		}
	}

	for _, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef != nil {
			name := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			msgs = append(msgs, fmt.Sprintf("  %s", name))
		}
	}

	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) doStatus(player *Player) *CommandResult {
	recalcBuildPoints(player)

	var msgs []string

	// Organization line (if any)
	if player.Organization > 0 {
		orgName := organizationName(player.Organization)
		if orgName != "" {
			msgs = append(msgs, fmt.Sprintf("You are a member of the %s.", orgName))
		}
	}

	msgs = append(msgs,
		fmt.Sprintf("Name: %s   Race: %s   Gender: %s   Level: %d", player.FullName(), player.RaceName(), genderName(player.Gender), player.Level),
		fmt.Sprintf("Strength: %d   Agility: %d   Quickness: %d", player.Strength, player.Agility, player.Quickness),
		fmt.Sprintf("Constitution: %d   Perception: %d   Willpower: %d   Empathy: %d", player.Constitution, player.Perception, player.Willpower, player.Empathy),
	)

	// Build points
	totalBP := player.BuildPoints
	spentBP := playerBPSpent(player)
	unspentBP := totalBP - spentBP
	if unspentBP < 0 {
		unspentBP = 0
	}
	xpUntilNextBP := xpUntilNextBuildPoint(player)

	msgs = append(msgs,
		fmt.Sprintf("Build Points to date: %d", totalBP),
		fmt.Sprintf("Unspent Build Points: %d", unspentBP),
		fmt.Sprintf("Experience Points until next Build Point: %d", xpUntilNextBP),
	)

	// Attack/Defense modifiers
	var weaponDef *gameworld.ItemDef
	if player.Wielded != nil {
		weaponDef = e.items[player.Wielded.Archetype]
	}
	atkRating := playerAttackRating(player, weaponDef)
	defRating := playerDefenseRating(player)
	stanceLabel := stanceNames[player.Stance]

	msgs = append(msgs,
		fmt.Sprintf("Current Attack Modifier: %d [%s]", atkRating, stanceLabel),
		fmt.Sprintf("Current Defend Modifier: %d", defRating),
	)

	// Height/Weight/Load
	heightFeet := player.Height / 12
	heightInches := player.Height % 12
	loadWeight := playerLoadWeight(player, e.items)
	msgs = append(msgs,
		fmt.Sprintf("Height: %d'%d   Weight: %d lbs", heightFeet, heightInches, player.Weight),
		fmt.Sprintf("Load: %d lbs", loadWeight),
	)

	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) doHealth(player *Player) *CommandResult {
	healthPct := float64(player.BodyPoints) / float64(player.MaxBodyPoints) * 100
	var healthDesc string
	switch {
	case healthPct >= 100:
		healthDesc = "You are in perfect health."
	case healthPct >= 75:
		healthDesc = "You have minor injuries."
	case healthPct >= 50:
		healthDesc = "You are moderately wounded."
	case healthPct >= 25:
		healthDesc = "You are seriously wounded."
	case healthPct > 0:
		healthDesc = "You are critically wounded!"
	default:
		healthDesc = "You are dead."
	}
	return &CommandResult{Messages: []string{
		healthDesc,
		fmt.Sprintf("Body: %d/%d   Fatigue: %d/%d", player.BodyPoints, player.MaxBodyPoints, player.Fatigue, player.MaxFatigue),
		fmt.Sprintf("Mana: %d/%d   Psi: %d/%d", player.Mana, player.MaxMana, player.Psi, player.MaxPsi),
	}}
}

func (e *GameEngine) doWield(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Wield what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			continue
		}
		if skip > 0 { skip--; continue }
		// Shields are worn, not wielded — route to WEAR
		if itemDef.Type == "SHIELD" || (itemDef.WornSlot != "" && !isWeapon(itemDef.Type)) {
			return e.doWear(ctx, player, args)
		}
		if !isWeapon(itemDef.Type) {
			return &CommandResult{Messages: []string{"You can't wield that."}}
		}
		if player.Wielded != nil {
			player.Inventory = append(player.Inventory, *player.Wielded)
		}
		wielded := player.Inventory[i]
		player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
		player.Wielded = &wielded
		e.SavePlayer(ctx, player)
		fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You wield %s.", fullName)},
			RoomBroadcast: []string{fmt.Sprintf("%s wields %s.", player.FirstName, fullName)},
		}
	}
	return &CommandResult{Messages: []string{"You don't have that."}}
}

func (e *GameEngine) doUnwield(ctx context.Context, player *Player) *CommandResult {
	if player.Wielded == nil {
		return &CommandResult{Messages: []string{"You aren't wielding anything."}}
	}
	itemDef := e.items[player.Wielded.Archetype]
	wepName := "their weapon"
	if itemDef != nil {
		wepName = e.formatItemName(itemDef, player.Wielded.Adj1, player.Wielded.Adj2, player.Wielded.Adj3)
	}
	player.Inventory = append(player.Inventory, *player.Wielded)
	player.Wielded = nil
	e.SavePlayer(ctx, player)
	return &CommandResult{
		Messages:      []string{"You put away your weapon."},
		RoomBroadcast: []string{fmt.Sprintf("%s puts away %s.", player.FirstName, wepName)},
	}
}

func (e *GameEngine) doWear(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Wear what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		if itemDef.WornSlot == "" {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			if skip > 0 { skip--; continue }
			worn := player.Inventory[i]
			worn.WornSlot = itemDef.WornSlot
			player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
			player.Worn = append(player.Worn, worn)
			e.SavePlayer(ctx, player)
			fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You put on %s.", fullName)},
				RoomBroadcast: []string{fmt.Sprintf("%s puts on %s.", player.FirstName, fullName)},
			}
		}
	}
	return &CommandResult{Messages: []string{"You don't have that."}}
}

func (e *GameEngine) doRemove(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Remove what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	for i, ii := range player.Worn {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			if skip > 0 { skip--; continue }
			removed := player.Worn[i]
			removed.WornSlot = ""
			player.Worn = append(player.Worn[:i], player.Worn[i+1:]...)
			player.Inventory = append(player.Inventory, removed)
			e.SavePlayer(ctx, player)
			fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You remove %s.", fullName)},
				RoomBroadcast: []string{fmt.Sprintf("%s removes %s.", player.FirstName, fullName)},
			}
		}
	}
	return &CommandResult{Messages: []string{"You aren't wearing that."}}
}

func (e *GameEngine) doOpen(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Open what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			if skip > 0 { skip--; continue }
			if !containsFlag(itemDef.Flags, "OPENABLE") && !isPortal(itemDef.Type) {
				return &CommandResult{Messages: []string{"You can't open that."}}
			}
			if ri.State == "LOCKED" {
				return &CommandResult{Messages: []string{"It's locked."}}
			}
			if ri.State == "LATCHED" {
				return &CommandResult{Messages: []string{"It's latched shut."}}
			}
			// Check for traps (VAL4 on item)
			trapMsgs := e.checkTrap(player, &room.Items[i])

			room.Items[i].State = "OPEN"
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "OPEN"})
			fullName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
			msgs := []string{fmt.Sprintf("You open %s.", fullName)}
			if len(trapMsgs) > 0 {
				msgs = append(msgs, trapMsgs...)
			}
			return &CommandResult{Messages: msgs}
		}
	}
	// Check inventory containers
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			if skip > 0 { skip--; continue }
			if !containsFlag(itemDef.Flags, "OPENABLE") {
				return &CommandResult{Messages: []string{"You can't open that."}}
			}
			if ii.State == "LOCKED" {
				return &CommandResult{Messages: []string{"It's locked."}}
			}
			if ii.State == "LATCHED" {
				return &CommandResult{Messages: []string{"It's latched shut."}}
			}
			player.Inventory[i].State = "OPEN"
			fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			return &CommandResult{Messages: []string{fmt.Sprintf("You open %s.", fullName)}}
		}
	}
	return &CommandResult{Messages: []string{"You don't see that here."}}
}

// checkTrap checks if an item has a trap (VAL4) and triggers it. Returns messages.
func (e *GameEngine) checkTrap(player *Player, ri *gameworld.RoomItem) []string {
	if ri.Val4 == 0 {
		return nil
	}
	trapType := ri.Val4
	ri.Val4 = 0 // trap is consumed

	var msgs []string
	switch {
	case trapType == 1: // Needle, minor poison
		msgs = append(msgs, "A needle springs out and pricks your finger!")
		player.Poisoned = true
	case trapType == 2: // Gas, minor poison
		msgs = append(msgs, "A cloud of noxious gas billows out!")
		player.Poisoned = true
	case trapType == 3: // Acid
		dmg := 10 + rand.Intn(15)
		player.BodyPoints -= dmg
		if player.BodyPoints < 0 { player.BodyPoints = 0 }
		msgs = append(msgs, fmt.Sprintf("Acid sprays out! [%d Damage]", dmg))
	case trapType == 4: // Blades
		dmg := 15 + rand.Intn(20)
		player.BodyPoints -= dmg
		if player.BodyPoints < 0 { player.BodyPoints = 0 }
		msgs = append(msgs, fmt.Sprintf("Hidden blades slash at you! [%d Damage]", dmg))
	case trapType == 5: // Needle, moderate poison
		msgs = append(msgs, "A poison-coated needle jabs into your hand!")
		player.Poisoned = true
	case trapType == 7: // Needle, major poison
		msgs = append(msgs, "A large needle drives deep into your finger, delivering a potent venom!")
		player.Poisoned = true
	case trapType == 8: // Explosive
		dmg := 30 + rand.Intn(30)
		player.BodyPoints -= dmg
		if player.BodyPoints < 0 { player.BodyPoints = 0 }
		msgs = append(msgs, fmt.Sprintf("The container explodes! [%d Damage]", dmg))
	case trapType == 9: // Acid, moderate
		dmg := 20 + rand.Intn(25)
		player.BodyPoints -= dmg
		if player.BodyPoints < 0 { player.BodyPoints = 0 }
		msgs = append(msgs, fmt.Sprintf("A gout of acid sprays out! [%d Damage]", dmg))
	case trapType == 12: // Gas, moderate poison
		msgs = append(msgs, "A thick cloud of poisonous gas engulfs you!")
		player.Poisoned = true
	case trapType == 13: // Black needle, lethal
		dmg := 40 + rand.Intn(30)
		player.BodyPoints -= dmg
		if player.BodyPoints < 0 { player.BodyPoints = 0 }
		msgs = append(msgs, fmt.Sprintf("A black needle strikes you, delivering a lethal toxin! [%d Damage]", dmg))
		player.Poisoned = true
	case trapType >= 1000: // Glyph traps (spell-based)
		spellDmg := 20 + rand.Intn(40)
		player.BodyPoints -= spellDmg
		if player.BodyPoints < 0 { player.BodyPoints = 0 }
		glyphType := (trapType / 1000) % 10
		switch {
		case glyphType <= 2:
			msgs = append(msgs, fmt.Sprintf("An Inferno Glyph erupts in a blast of flame! [%d Damage]", spellDmg))
		case glyphType <= 4:
			msgs = append(msgs, fmt.Sprintf("An Ice Glyph detonates in a burst of freezing cold! [%d Damage]", spellDmg))
		case glyphType <= 6:
			msgs = append(msgs, fmt.Sprintf("A Thunder Glyph explodes with crackling energy! [%d Damage]", spellDmg))
		case glyphType <= 8:
			msgs = append(msgs, fmt.Sprintf("An Imprisonment Rune flares! You feel rooted to the spot!"))
			player.Immobilized = true
		default:
			msgs = append(msgs, fmt.Sprintf("A Symbol of Death erupts! [%d Damage]", spellDmg))
		}
	}
	return msgs
}

func (e *GameEngine) doClose(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Close what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			if skip > 0 { skip--; continue }
			room.Items[i].State = "CLOSED"
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "CLOSED"})
			fullName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
			return &CommandResult{Messages: []string{fmt.Sprintf("You close %s.", fullName)}}
		}
	}
	// Check inventory containers
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			if skip > 0 { skip--; continue }
			player.Inventory[i].State = "CLOSED"
			fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			return &CommandResult{Messages: []string{fmt.Sprintf("You close %s.", fullName)}}
		}
	}
	return &CommandResult{Messages: []string{"You don't see that here."}}
}

func (e *GameEngine) doGive(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Give what to whom? (give <item> to <player>)"}}
	}
	// Parse: give <item> to <target> OR give <target> <item>
	toIdx := -1
	for i, a := range args {
		if strings.ToUpper(a) == "TO" {
			toIdx = i
			break
		}
	}
	var itemName, targetName string
	if toIdx > 0 && toIdx < len(args)-1 {
		itemName = strings.ToLower(strings.Join(args[:toIdx], " "))
		targetName = strings.ToLower(strings.Join(args[toIdx+1:], " "))
	} else {
		return &CommandResult{Messages: []string{"Give what to whom? (give <item> to <player>)"}}
	}
	// Check for money giving: "give 5 gold to Taliesin", "give 10 kragenmark to Taliesin"
	if amount, currency, ok := parseMoneyAmount(itemName); ok {
		target := e.findPlayerInRoom(player, targetName)
		if target == nil {
			return &CommandResult{Messages: []string{"You don't see that person here."}}
		}
		return e.doGiveMoney(ctx, player, target, amount, currency)
	}

	itemName, ordSkip := parseOrdinal(itemName)
	skip := ordSkip

	// Find the item in inventory
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, itemName, e.getAdjName(ii.Adj1)) {
			continue
		}
		if skip > 0 { skip--; continue }
		// Find the target player
		target := e.findPlayerInRoom(player, targetName)
		if target == nil {
			return &CommandResult{Messages: []string{"You don't see that person here."}}
		}
		// Transfer item
		fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
		target.Inventory = append(target.Inventory, ii)
		player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
		e.SavePlayer(ctx, player)
		e.SavePlayer(ctx, target)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You give %s to %s.", fullName, target.FirstName)},
			RoomBroadcast: []string{fmt.Sprintf("%s gives %s to %s.", player.FirstName, fullName, target.FirstName)},
			TargetName:    target.FirstName,
			TargetMsg:     []string{fmt.Sprintf("%s gives you %s.", player.FirstName, fullName)},
		}
	}
	return &CommandResult{Messages: []string{"You don't have that."}}
}

// parseMoneyAmount checks if a string like "5 gold" or "10 kragenmark" is a money amount.
// Returns (amount, currency_name, true) or (0, "", false).
func parseMoneyAmount(s string) (int, string, bool) {
	parts := strings.Fields(s)
	if len(parts) < 2 {
		return 0, "", false
	}
	amount, err := strconv.Atoi(parts[0])
	if err != nil || amount <= 0 {
		return 0, "", false
	}
	currency := strings.ToLower(strings.Join(parts[1:], " "))
	// Recognize all currency types
	switch currency {
	case "gold", "crown", "crowns", "gold crown", "gold crowns":
		return amount, "gold", true
	case "silver", "shilling", "shillings", "silver shilling", "silver shillings":
		return amount, "silver", true
	case "copper", "penny", "pennies", "copper penny", "copper pennies":
		return amount, "copper", true
	case "coin", "coins":
		return amount, "copper", true
	case "kragenmark", "kragenmarks":
		return amount, "kragenmark", true
	case "danir", "danirs":
		return amount, "danir", true
	case "shard", "shards":
		return amount, "shard", true
	case "darktar", "darktars":
		return amount, "darktar", true
	case "dollar", "dollars":
		return amount, "dollar", true
	}
	return 0, "", false
}

// doGiveMoney transfers currency from one player to another.
func (e *GameEngine) doGiveMoney(ctx context.Context, giver, receiver *Player, amount int, currency string) *CommandResult {
	// Check if giver has enough
	currencyDisplay := ""
	switch currency {
	case "gold":
		if giver.Gold < amount {
			return &CommandResult{Messages: []string{fmt.Sprintf("You only have %d gold.", giver.Gold)}}
		}
		giver.Gold -= amount
		receiver.Gold += amount
		currencyDisplay = fmt.Sprintf("%d gold crown", amount)
		if amount != 1 { currencyDisplay += "s" }
	case "silver":
		if giver.Silver < amount {
			return &CommandResult{Messages: []string{fmt.Sprintf("You only have %d silver.", giver.Silver)}}
		}
		giver.Silver -= amount
		receiver.Silver += amount
		currencyDisplay = fmt.Sprintf("%d silver shilling", amount)
		if amount != 1 { currencyDisplay += "s" }
	case "copper":
		if giver.Copper < amount {
			return &CommandResult{Messages: []string{fmt.Sprintf("You only have %d copper.", giver.Copper)}}
		}
		giver.Copper -= amount
		receiver.Copper += amount
		currencyDisplay = fmt.Sprintf("%d copper penn", amount)
		if amount == 1 { currencyDisplay += "y" } else { currencyDisplay += "ies" }
	default:
		// Regional currencies — these are handled as inventory items with MONEY type
		// Find the currency item in giver's inventory
		for i, ii := range giver.Inventory {
			def := e.items[ii.Archetype]
			if def == nil || def.Type != "MONEY" {
				continue
			}
			noun := strings.ToLower(e.nouns[def.NameID])
			if noun == currency || strings.HasPrefix(noun, currency) {
				coins := ii.Val1
				if coins < amount {
					return &CommandResult{Messages: []string{fmt.Sprintf("You only have %d %s.", coins, currency)}}
				}
				if coins == amount {
					// Transfer the whole stack
					receiver.Inventory = append(receiver.Inventory, ii)
					giver.Inventory = append(giver.Inventory[:i], giver.Inventory[i+1:]...)
				} else {
					// Split the stack
					giver.Inventory[i].Val1 -= amount
					newItem := ii
					newItem.Val1 = amount
					receiver.Inventory = append(receiver.Inventory, newItem)
				}
				currencyDisplay = fmt.Sprintf("%d %s", amount, currency)
				if amount != 1 { currencyDisplay += "s" }
				e.SavePlayer(ctx, giver)
				e.SavePlayer(ctx, receiver)
				return &CommandResult{
					Messages:      []string{fmt.Sprintf("You give %s to %s.", currencyDisplay, receiver.FirstName)},
					RoomBroadcast: []string{fmt.Sprintf("%s gives some coins to %s.", giver.FirstName, receiver.FirstName)},
					TargetName:    receiver.FirstName,
					TargetMsg:     []string{fmt.Sprintf("%s gives you %s.", giver.FirstName, currencyDisplay)},
				}
			}
		}
		return &CommandResult{Messages: []string{fmt.Sprintf("You don't have any %s.", currency)}}
	}

	e.SavePlayer(ctx, giver)
	e.SavePlayer(ctx, receiver)
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You give %s to %s.", currencyDisplay, receiver.FirstName)},
		RoomBroadcast: []string{fmt.Sprintf("%s gives some coins to %s.", giver.FirstName, receiver.FirstName)},
		TargetName:    receiver.FirstName,
		TargetMsg:     []string{fmt.Sprintf("%s gives you %s.", giver.FirstName, currencyDisplay)},
	}
}

func (e *GameEngine) doEat(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Eat what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		if itemDef.Type != "FOOD" {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			if skip > 0 { skip--; continue }
			fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)

			// Run item scripts FIRST — they may set ITEMVAL3 based on adjective checks
			room := e.rooms[player.RoomNumber]
			tempRI := gameworld.RoomItem{Ref: -1, Archetype: ii.Archetype,
				Adj1: ii.Adj1, Adj2: ii.Adj2, Adj3: ii.Adj3,
				Val1: ii.Val1, Val2: ii.Val2, Val3: ii.Val3, Val4: ii.Val4, Val5: ii.Val5}
			// Run all item-level scripts (IFVAR at root level + IFVERB EAT)
			sc := e.RunItemScripts(player, room, &tempRI, itemDef)
			sc2 := e.RunVerbScripts(player, room, "EAT", &tempRI, itemDef)
			sc.Messages = append(sc.Messages, sc2.Messages...)
			// Scripts may have modified tempRI.Val3
			spellNum := tempRI.Val3

			// Bite tracking: initialize Val2 from Parameter1 on first bite
			currentBites := ii.Val2
			if currentBites == 0 && itemDef.Parameter1 > 0 {
				currentBites = itemDef.Parameter1
			}
			isFirstBite := (currentBites == 0 && itemDef.Parameter1 == 0) || (ii.Val2 == 0 && itemDef.Parameter1 > 0)

			var msgs []string
			if currentBites <= 1 {
				// Last bite (or single-bite food) — remove from inventory
				player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
				msgs = []string{fmt.Sprintf("You finish eating %s.", fullName)}
			} else {
				// Decrement bites remaining
				newVal := currentBites - 1
				player.Inventory[i].Val2 = newVal
				msgs = []string{fmt.Sprintf("You take a bite of %s. (%d bites remaining)", fullName, newVal)}
			}

			// Add any script ECHO messages
			msgs = append(msgs, sc.Messages...)

			// Spell effect fires on FIRST bite only
			if isFirstBite {
				if spellNum == 403 { // Mindlink
					player.TelepathyActive = true
					player.TelepathyExpiry = time.Now().Add(1 * time.Hour)
					msgs = append(msgs, "You feel your mind open to the thoughts of others.")
				} else if spellNum != 0 {
					msgs = append(msgs, fmt.Sprintf("[Spell #%d effect coming soon.]", spellNum))
				}
			}
			e.SavePlayer(ctx, player)
			return &CommandResult{
				Messages:      msgs,
				RoomBroadcast: []string{fmt.Sprintf("%s eats %s.", player.FirstName, fullName)},
				PlayerState:   player,
			}
		}
	}
	return &CommandResult{Messages: []string{"You don't have that."}}
}

func (e *GameEngine) doBuy(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Buy what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil || len(room.StoreItems) == 0 {
		return &CommandResult{Messages: []string{"There is nothing for sale here."}}
	}

	for _, si := range room.StoreItems {
		itemDef := e.items[si.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		adjName := ""
		if si.Adj > 0 {
			adjName = e.getAdjName(si.Adj)
		}
		if !matchesTarget(name, target, adjName) {
			continue
		}
		if skip > 0 { skip--; continue }

		// Check affordability
		totalCopper := player.Gold*100 + player.Silver*10 + player.Copper
		if totalCopper < si.Price {
			priceStr := formatPrice(si.Price)
			return &CommandResult{Messages: []string{fmt.Sprintf("You can't afford that. %s costs %s.", name, priceStr)}}
		}

		// Deduct currency efficiently (spend copper first, then silver, then gold)
		remaining := si.Price
		if player.Copper >= remaining {
			player.Copper -= remaining
			remaining = 0
		} else {
			remaining -= player.Copper
			player.Copper = 0
		}
		if remaining > 0 {
			silverNeeded := (remaining + 9) / 10 // round up
			if player.Silver >= silverNeeded {
				player.Silver -= silverNeeded
				player.Copper += silverNeeded*10 - remaining
				remaining = 0
			} else {
				remaining -= player.Silver * 10
				player.Silver = 0
			}
		}
		if remaining > 0 {
			goldNeeded := (remaining + 99) / 100 // round up
			player.Gold -= goldNeeded
			player.Copper += goldNeeded*100 - remaining
		}

		// Give item to player
		item := InventoryItem{Archetype: si.Archetype}
		if si.Adj > 0 {
			// Store adjective goes in ADJ3 (last slot) — ADJ1/ADJ2 left open for
			// crafting/enchanting. Item scripts check ITEMADJ3 for the variety.
			item.Adj3 = si.Adj
		}
		player.Inventory = append(player.Inventory, item)
		e.SavePlayer(ctx, player)

		displayName := e.formatItemName(itemDef, item.Adj1, item.Adj2, item.Adj3)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You hand over your money and retrieve your %s.", displayName)},
			RoomBroadcast: []string{fmt.Sprintf("%s purchases the %s.", player.FirstName, displayName)},
		}
	}

	return &CommandResult{Messages: []string{"That item is not for sale here."}}
}

func (e *GameEngine) doSell(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Sell what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't sell anything here."}}
	}

	// Check if room has any BUY_ modifier
	canBuy := false
	for _, mod := range room.Modifiers {
		if strings.HasPrefix(mod, "BUY_") {
			canBuy = true
			break
		}
	}
	if !canBuy {
		return &CommandResult{Messages: []string{"Nobody here is interested in buying anything."}}
	}

	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			if skip > 0 { skip--; continue }
			displayName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			// Sell value: VAL1 on item is copper value, fallback to weight-based estimate
			sellValue := ii.Val1
			if sellValue <= 0 {
				sellValue = itemDef.Weight + 1 // minimal fallback
			}
			// Merchants pay ~50% of value
			sellValue = sellValue / 2
			if sellValue < 1 { sellValue = 1 }
			player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
			// Add coins
			player.Gold += sellValue / 100
			player.Silver += (sellValue % 100) / 10
			player.Copper += sellValue % 10
			e.SavePlayer(ctx, player)
			return &CommandResult{Messages: []string{
				fmt.Sprintf("The merchant inspects %s closely.", displayName),
				fmt.Sprintf("The merchant takes the item and hands you %s.", formatPrice(sellValue)),
			}}
		}
	}

	return &CommandResult{Messages: []string{"You don't have that."}}
}

func (e *GameEngine) doAppraise(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Appraise what?"}}
	}
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}
	canBuy := false
	for _, mod := range room.Modifiers {
		if strings.HasPrefix(mod, "BUY_") {
			canBuy = true
			break
		}
	}
	if !canBuy {
		return &CommandResult{Messages: []string{"There is no merchant here to appraise your items."}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	for _, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			if skip > 0 { skip--; continue }
			displayName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			sellValue := ii.Val1
			if sellValue <= 0 {
				sellValue = itemDef.Weight + 1
			}
			sellValue = sellValue / 2
			if sellValue < 1 { sellValue = 1 }
			return &CommandResult{Messages: []string{
				fmt.Sprintf("The merchant examines %s carefully.", displayName),
				fmt.Sprintf("\"I'd give you %s for that.\"", formatPrice(sellValue)),
			}}
		}
	}
	return &CommandResult{Messages: []string{"You don't have that."}}
}

// formatPrice formats a copper amount as a readable price string.
func formatPrice(copper int) string {
	gold := copper / 100
	remainder := copper % 100
	silver := remainder / 10
	cop := remainder % 10
	var parts []string
	if gold > 0 {
		if gold == 1 {
			parts = append(parts, "1 gold crown")
		} else {
			parts = append(parts, fmt.Sprintf("%d gold crowns", gold))
		}
	}
	if silver > 0 {
		if silver == 1 {
			parts = append(parts, "1 silver shilling")
		} else {
			parts = append(parts, fmt.Sprintf("%d silver shillings", silver))
		}
	}
	if cop > 0 || len(parts) == 0 {
		if cop == 1 {
			parts = append(parts, "1 copper penny")
		} else {
			parts = append(parts, fmt.Sprintf("%d copper pennies", cop))
		}
	}
	return joinList(parts)
}

func (e *GameEngine) doDrink(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Drink what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		if itemDef.Type != "LIQUID" && itemDef.Type != "LIQCONTAINER" && itemDef.Type != "FOOD" {
			continue
		}
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			continue
		}
		if skip > 0 { skip--; continue }
		displayName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
		if itemDef.Type == "FOOD" {
			// EAT logic — redirect to doEat
			return e.doEat(ctx, player, args)
		}

		// Sip tracking: initialize Val2 from Parameter1 on first sip
		currentSips := ii.Val2
		if currentSips == 0 && itemDef.Parameter1 > 0 {
			currentSips = itemDef.Parameter1
		}
		isFirstSip := (currentSips == 0 && itemDef.Parameter1 == 0) || (ii.Val2 == 0 && itemDef.Parameter1 > 0)

		// Run item scripts for spell effects
		room := e.rooms[player.RoomNumber]
		tempRI := gameworld.RoomItem{Ref: -1, Archetype: ii.Archetype,
			Adj1: ii.Adj1, Adj2: ii.Adj2, Adj3: ii.Adj3,
			Val1: ii.Val1, Val2: ii.Val2, Val3: ii.Val3, Val4: ii.Val4, Val5: ii.Val5}
		sc := e.RunItemScripts(player, room, &tempRI, itemDef)
		spellNum := tempRI.Val3

		var msgs []string
		if currentSips <= 1 {
			// Last sip (or single-sip drink) — remove from inventory
			player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
			msgs = []string{fmt.Sprintf("You finish drinking %s.", displayName)}
		} else {
			newVal := currentSips - 1
			player.Inventory[i].Val2 = newVal
			msgs = []string{fmt.Sprintf("You take a sip from %s. (%d sips remaining)", displayName, newVal)}
		}

		msgs = append(msgs, sc.Messages...)

		// Spell effect fires on FIRST sip only
		if isFirstSip && spellNum != 0 {
			if spellNum == 403 {
				player.TelepathyActive = true
				player.TelepathyExpiry = time.Now().Add(1 * time.Hour)
				msgs = append(msgs, "You feel your mind open to the thoughts of others.")
			} else {
				msgs = append(msgs, fmt.Sprintf("[Spell #%d effect coming soon.]", spellNum))
			}
		}
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      msgs,
			RoomBroadcast: []string{fmt.Sprintf("%s drinks from %s.", player.FirstName, displayName)},
			PlayerState:   player,
		}
	}
	return &CommandResult{Messages: []string{"You don't have that."}}
}

func (e *GameEngine) doLight(ctx context.Context, player *Player, args []string, lightOn bool) *CommandResult {
	if len(args) == 0 {
		if lightOn { return &CommandResult{Messages: []string{"Light what?"}} }
		return &CommandResult{Messages: []string{"Extinguish what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil { continue }
		if !containsFlag(itemDef.Flags, "LIGHTABLE") { continue }
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ii.Adj1)) { continue }
		if skip > 0 { skip--; continue }
		displayName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
		if lightOn {
			player.Inventory[i].State = "LIT"
			e.SavePlayer(ctx, player)
			return &CommandResult{Messages: []string{fmt.Sprintf("You light %s.", displayName)}}
		}
		player.Inventory[i].State = "UNLIT"
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{fmt.Sprintf("You extinguish %s.", displayName)}}
	}
	return &CommandResult{Messages: []string{"You don't have anything to light."}}
}

func (e *GameEngine) doFlip(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 { return &CommandResult{Messages: []string{"Flip what?"}} }
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil { return &CommandResult{Messages: []string{"You can't do that here."}} }
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil { continue }
		if !containsFlag(itemDef.Flags, "FLIPABLE") { continue }
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ri.Adj1)) { continue }
		if skip > 0 { skip--; continue }
		displayName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
		if ri.State == "FLIPPED" {
			room.Items[i].State = "UNFLIPPED"
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "UNFLIPPED"})
		} else {
			room.Items[i].State = "FLIPPED"
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "FLIPPED"})
		}
		// Run IFPREVERB FLIP scripts
		sc := e.RunPreverbScripts(player, room, "FLIP", &room.Items[i], itemDef)
		result := &CommandResult{Messages: []string{fmt.Sprintf("You flip %s.", displayName)}}
		result.Messages = append(result.Messages, sc.Messages...)
		result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
		return result
	}
	return &CommandResult{Messages: []string{"You don't see anything to flip here."}}
}

func (e *GameEngine) doLatch(player *Player, args []string, latch bool) *CommandResult {
	if len(args) == 0 {
		if latch { return &CommandResult{Messages: []string{"Latch what?"}} }
		return &CommandResult{Messages: []string{"Unlatch what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil { return &CommandResult{Messages: []string{"You can't do that here."}} }
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil { continue }
		if !containsFlag(itemDef.Flags, "LATCHABLE") { continue }
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ri.Adj1)) { continue }
		if skip > 0 { skip--; continue }
		displayName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
		if latch {
			room.Items[i].State = "LATCHED"
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "LATCHED"})
			return &CommandResult{Messages: []string{fmt.Sprintf("You latch %s.", displayName)}}
		}
		room.Items[i].State = "UNLATCHED"
		e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "UNLATCHED"})
		return &CommandResult{Messages: []string{fmt.Sprintf("You unlatch %s.", displayName)}}
	}
	return &CommandResult{Messages: []string{"You don't see anything to latch here."}}
}

func (e *GameEngine) doLock(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Lock what?"}}
	}
	raw := strings.ToLower(strings.Join(args, " "))
	target, keyName := parseWithClause(raw)
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		if !containsFlag(itemDef.Flags, "LOCKABLE") {
			continue
		}
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}
		if ri.State == "LOCKED" {
			return &CommandResult{Messages: []string{"It's already locked."}}
		}
		// Find matching key
		keyItem := e.findKey(player, ri.Val3, keyName)
		if keyItem == nil {
			return &CommandResult{Messages: []string{"You don't have the right key."}}
		}
		room.Items[i].State = "LOCKED"
		e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "LOCKED"})
		displayName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
		return &CommandResult{Messages: []string{fmt.Sprintf("You lock %s.", displayName)}}
	}
	return &CommandResult{Messages: []string{"You don't see anything to lock here."}}
}

func (e *GameEngine) doUnlock(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Unlock what?"}}
	}
	raw := strings.ToLower(strings.Join(args, " "))
	target, keyName := parseWithClause(raw)
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		if !containsFlag(itemDef.Flags, "LOCKABLE") {
			continue
		}
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}
		if ri.State != "LOCKED" {
			return &CommandResult{Messages: []string{"It isn't locked."}}
		}
		// Find matching key
		keyItem := e.findKey(player, ri.Val3, keyName)
		if keyItem == nil {
			return &CommandResult{Messages: []string{"You don't have the right key."}}
		}
		room.Items[i].State = "CLOSED"
		e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "CLOSED"})
		displayName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
		return &CommandResult{Messages: []string{fmt.Sprintf("You unlock %s.", displayName)}}
	}
	return &CommandResult{Messages: []string{"You don't see anything to unlock here."}}
}

// parseWithClause splits "target with key" into (target, key). If no "with", key is "".
func parseWithClause(s string) (string, string) {
	idx := strings.Index(s, " with ")
	if idx < 0 {
		return s, ""
	}
	return strings.TrimSpace(s[:idx]), strings.TrimSpace(s[idx+6:])
}

// findKey searches the player's inventory for a KEY-type item whose Val3 matches lockVal3.
// If keyName is non-empty, the key must also match that name.
func (e *GameEngine) findKey(player *Player, lockVal3 int, keyName string) *InventoryItem {
	allItems := make([]InventoryItem, 0, len(player.Inventory))
	allItems = append(allItems, player.Inventory...)
	for i := range allItems {
		ii := &allItems[i]
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		if !strings.EqualFold(itemDef.Type, "KEY") {
			continue
		}
		if ii.Val3 != lockVal3 {
			continue
		}
		if keyName != "" {
			name := e.getItemNounName(itemDef)
			if !matchesTarget(name, keyName, e.getAdjName(ii.Adj1)) {
				continue
			}
		}
		return ii
	}
	return nil
}

func (e *GameEngine) doDeposit(ctx context.Context, player *Player, args []string) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil || !containsModifier(room.Modifiers, "BANK") {
		return &CommandResult{Messages: []string{"There is no bank here."}}
	}
	if len(args) == 0 { return &CommandResult{Messages: []string{"Deposit how much?"}} }
	amount := 0
	fmt.Sscanf(args[0], "%d", &amount)
	if amount <= 0 { return &CommandResult{Messages: []string{"Invalid amount."}} }
	totalCopper := player.Gold*100 + player.Silver*10 + player.Copper
	if totalCopper < amount {
		return &CommandResult{Messages: []string{"You don't have that much money."}}
	}
	// Deduct from carried
	remaining := amount
	if player.Copper >= remaining { player.Copper -= remaining; remaining = 0 } else { remaining -= player.Copper; player.Copper = 0 }
	if remaining > 0 { sn := (remaining+9)/10; if player.Silver >= sn { player.Silver -= sn; player.Copper += sn*10-remaining; remaining = 0 } else { remaining -= player.Silver*10; player.Silver = 0 } }
	if remaining > 0 { gn := (remaining+99)/100; player.Gold -= gn; player.Copper += gn*100-remaining }
	player.BankCopper += amount
	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: []string{fmt.Sprintf("You deposit %s.", formatPrice(amount))}}
}

func (e *GameEngine) doWithdraw(ctx context.Context, player *Player, args []string) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil || !containsModifier(room.Modifiers, "BANK") {
		return &CommandResult{Messages: []string{"There is no bank here."}}
	}
	if len(args) == 0 { return &CommandResult{Messages: []string{"Withdraw how much?"}} }
	amount := 0
	fmt.Sscanf(args[0], "%d", &amount)
	if amount <= 0 { return &CommandResult{Messages: []string{"Invalid amount."}} }
	totalBank := player.BankGold*100 + player.BankSilver*10 + player.BankCopper
	if totalBank < amount {
		return &CommandResult{Messages: []string{"You don't have that much in the bank."}}
	}
	remaining := amount
	if player.BankCopper >= remaining { player.BankCopper -= remaining; remaining = 0 } else { remaining -= player.BankCopper; player.BankCopper = 0 }
	if remaining > 0 { sn := (remaining+9)/10; if player.BankSilver >= sn { player.BankSilver -= sn; player.BankCopper += sn*10-remaining; remaining = 0 } else { remaining -= player.BankSilver*10; player.BankSilver = 0 } }
	if remaining > 0 { gn := (remaining+99)/100; player.BankGold -= gn; player.BankCopper += gn*100-remaining }
	player.Copper += amount
	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: []string{fmt.Sprintf("You withdraw %s.", formatPrice(amount))}}
}

func containsModifier(mods []string, mod string) bool {
	for _, m := range mods { if m == mod { return true } }
	return false
}

func (e *GameEngine) doRead(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Read what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"There is nothing written on it."}}
	}

	// Search room items
	for _, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			if skip > 0 { skip--; continue }
			return e.readRoomItem(room, itemDef, &ri)
		}
	}

	// Search all player items (inventory + worn + wielded)
	allReadItems := make([]InventoryItem, 0, len(player.Inventory)+len(player.Worn)+1)
	allReadItems = append(allReadItems, player.Inventory...)
	allReadItems = append(allReadItems, player.Worn...)
	if player.Wielded != nil { allReadItems = append(allReadItems, *player.Wielded) }
	for _, ii := range allReadItems {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) || matchesTarget(name, target, e.getAdjName(ii.Adj3)) {
			if skip > 0 { skip--; continue }
			return &CommandResult{Messages: []string{"There is nothing written on it."}}
		}
	}

	return &CommandResult{Messages: []string{"You don't see that here."}}
}

// SkillNames maps skill IDs to names.
var SkillNames = map[int]string{
	0: "Jeweler", 1: "Two Weapons", 2: "Backstab", 3: "Missile Weapons",
	4: "Natural Weapons", 5: "Climbing", 6: "Dodging & Parrying", 7: "Conjuration",
	8: "Weaponsmithing", 9: "Crushing Weapons", 10: "Combat Maneuvering",
	11: "Endurance", 12: "Trap & Poison Lore", 13: "Edged Weapons",
	14: "Enchantment", 15: "Dyeing/Weaving", 16: "Drakin Weapons",
	17: "Druidic Magic", 18: "Wood Lore", 19: "Thrown Weapons",
	20: "Healing", 21: "Legerdemain", 22: "Lockpicking", 23: "Spellcraft",
	24: "Martial Arts", 25: "Polearms", 26: "Psionics",
	27: "Mind over Mind", 28: "Mind over Matter", 29: "Transcendence",
	30: "Necromancy", 31: "Alchemy", 32: "Sagecraft", 33: "Stealth",
	34: "Disguise", 35: "Mining",
}

func (e *GameEngine) doUnlearn(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Unlearn what skill?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	for id, name := range SkillNames {
		if !strings.HasPrefix(strings.ToLower(name), target) {
			continue
		}
		currentLvl := player.Skills[id]
		if currentLvl <= 0 {
			return &CommandResult{Messages: []string{fmt.Sprintf("You don't have any ranks in %s.", name)}}
		}
		// Unlearn one rank, get back build points minus one
		bpBack := max(0, currentLvl-1) // return BP spent minus 1
		player.Skills[id] = currentLvl - 1
		player.BuildPoints += bpBack
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{
			fmt.Sprintf("You unlearn a rank of %s. (now rank %d, +%d build points, total BP: %d)", name, currentLvl-1, bpBack, player.BuildPoints),
		}}
	}
	return &CommandResult{Messages: []string{"You don't know that skill."}}
}

func (e *GameEngine) doUndress(ctx context.Context, player *Player) *CommandResult {
	if len(player.Worn) == 0 {
		return &CommandResult{Messages: []string{"You aren't wearing anything to remove."}}
	}
	// Remove the last worn item
	item := player.Worn[len(player.Worn)-1]
	player.Worn = player.Worn[:len(player.Worn)-1]
	player.Inventory = append(player.Inventory, item)
	e.SavePlayer(ctx, player)
	itemDef := e.items[item.Archetype]
	name := "something"
	if itemDef != nil {
		name = e.formatItemName(itemDef, item.Adj1, item.Adj2, item.Adj3)
	}
	return &CommandResult{Messages: []string{fmt.Sprintf("You remove %s.", name)}}
}

func (e *GameEngine) doMark(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		// Show marks
		if player.Marks == nil || len(player.Marks) == 0 {
			return &CommandResult{Messages: []string{"You have no marks set."}}
		}
		var msgs []string
		msgs = append(msgs, "Your marks:")
		for i := 1; i <= 10; i++ {
			if roomNum, ok := player.Marks[i]; ok {
				name := fmt.Sprintf("Room %d", roomNum)
				if r := e.rooms[roomNum]; r != nil {
					name = r.Name
				}
				if player.IsGM {
					msgs = append(msgs, fmt.Sprintf("  Mark %d: %s (%d)", i, name, roomNum))
				} else {
					msgs = append(msgs, fmt.Sprintf("  Mark %d: %s", i, name))
				}
			}
		}
		return &CommandResult{Messages: msgs}
	}
	num := 0
	fmt.Sscanf(args[0], "%d", &num)
	if num < 1 || num > 10 {
		return &CommandResult{Messages: []string{"Mark number must be 1-10."}}
	}
	if player.Marks == nil {
		player.Marks = make(map[int]int)
	}
	player.Marks[num] = player.RoomNumber
	e.SavePlayer(ctx, player)
	room := e.rooms[player.RoomNumber]
	name := fmt.Sprintf("room %d", player.RoomNumber)
	if room != nil {
		name = room.Name
	}
	return &CommandResult{Messages: []string{fmt.Sprintf("Mark %d set to %s.", num, name)}}
}

func (e *GameEngine) doBalance(player *Player) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil || !containsModifier(room.Modifiers, "BANK") {
		return &CommandResult{Messages: []string{"You need to be at a bank to check your balance."}}
	}
	msgs := []string{"=== Bank Balance ==="}
	total := player.BankGold*100 + player.BankSilver*10 + player.BankCopper
	if total == 0 {
		msgs = append(msgs, "Your account is empty.")
	} else {
		msgs = append(msgs, fmt.Sprintf("Balance: %s", formatPrice(total)))
	}
	return &CommandResult{Messages: msgs}
}

// doLoadWeapon handles NOCK/LOAD <weapon> WITH <ammo>.
// Bows/crossbows need to be loaded with arrows/bolts before firing.
func (e *GameEngine) doLoadWeapon(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Load what? Usage: NOCK <weapon> WITH <ammo>"}}
	}
	raw := strings.ToLower(strings.Join(args, " "))
	weaponTarget, ammoTarget := parseWithClause(raw)
	if ammoTarget == "" {
		return &CommandResult{Messages: []string{"Load with what? Usage: NOCK <weapon> WITH <ammo>"}}
	}

	// Find the weapon (must be wielded or in inventory)
	if player.Wielded == nil {
		return &CommandResult{Messages: []string{"You aren't wielding a ranged weapon."}}
	}
	weaponDef := e.items[player.Wielded.Archetype]
	if weaponDef == nil || (weaponDef.Type != "BOW_WEAPON" && weaponDef.Type != "HANDGUN" && weaponDef.Type != "RIFLE") {
		return &CommandResult{Messages: []string{"You aren't wielding a ranged weapon."}}
	}
	wepName := e.getItemNounName(weaponDef)
	if !strings.HasPrefix(strings.ToLower(wepName), weaponTarget) && weaponTarget != "bow" && weaponTarget != "crossbow" && weaponTarget != "gun" {
		return &CommandResult{Messages: []string{"You aren't wielding that weapon."}}
	}

	// Check if already loaded
	if player.Wielded.Val3 > 0 {
		return &CommandResult{Messages: []string{fmt.Sprintf("Your %s is already loaded.", wepName)}}
	}

	// Find ammo in inventory (must match Parameter2 ammo type on weapon)
	ammoType := weaponDef.Parameter2 // what ammo this weapon takes
	for i, ii := range player.Inventory {
		ammoDef := e.items[ii.Archetype]
		if ammoDef == nil {
			continue
		}
		ammoName := e.getItemNounName(ammoDef)
		if !strings.HasPrefix(strings.ToLower(ammoName), ammoTarget) {
			continue
		}
		// Check ammo type match
		if ammoType > 0 && ii.Archetype != ammoType {
			return &CommandResult{Messages: []string{fmt.Sprintf("You can't load your %s with that.", wepName)}}
		}
		// Load the weapon: set Val3 > 0 to indicate loaded
		player.Wielded.Val3 = ii.Archetype
		// Remove one ammo from inventory (or reduce bundle count)
		player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
		e.SavePlayer(ctx, player)

		ammoDisplayName := e.formatItemName(ammoDef, ii.Adj1, ii.Adj2, ii.Adj3)
		wepDisplayName := e.formatItemName(weaponDef, player.Wielded.Adj1, player.Wielded.Adj2, player.Wielded.Adj3)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You load your %s with %s.", wepDisplayName, ammoDisplayName)},
			RoomBroadcast: []string{fmt.Sprintf("%s loads %s %s.", player.FirstName, player.Possessive(), wepDisplayName)},
		}
	}

	return &CommandResult{Messages: []string{fmt.Sprintf("You don't have any '%s' to load.", ammoTarget)}}
}

// ---- DISARM command ----

func (e *GameEngine) doDisarm(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Disarm what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}

	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}

		// Check if item has a trap (Val4 > 0)
		if ri.Val4 == 0 {
			return &CommandResult{Messages: []string{"That doesn't appear to be trapped."}}
		}

		// Requires Trap & Poison Lore (skill #12)
		trapSkill := player.Skills[12]
		if trapSkill < 1 {
			return &CommandResult{Messages: []string{"You have no training in Trap & Poison Lore."}}
		}

		// Skill check: base 20% + skill_level * 5%, capped at 95%
		successChance := 20 + trapSkill*5
		if successChance > 95 {
			successChance = 95
		}
		roll := rand.Intn(100) + 1

		// Apply round time (5 seconds)
		player.RoundTimeExpiry = time.Now().Add(5 * time.Second)

		if roll <= successChance {
			// Success — remove the trap
			room.Items[i].Val4 = 0
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_update", ItemRef: ri.Ref, Item: &room.Items[i]})
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("[Success: %d%%, Roll: %d] Success!", successChance, roll), "You carefully disarm the trap."},
				RoomBroadcast: []string{fmt.Sprintf("%s carefully disarms a trap.", player.FirstName)},
			}
		}

		// Failure — optionally trigger the trap
		msgs := []string{fmt.Sprintf("[Success: %d%%, Roll: %d] Failure.", successChance, roll), "You are unable to disarm the trap."}
		// Critical failure (roll > 90): trigger the trap
		if roll > 90 {
			trapMsgs := e.checkTrap(player, &room.Items[i])
			if len(trapMsgs) > 0 {
				msgs = append(msgs, trapMsgs...)
				e.SavePlayer(ctx, player)
				e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_update", ItemRef: ri.Ref, Item: &room.Items[i]})
			}
		}
		return &CommandResult{Messages: msgs}
	}
	return &CommandResult{Messages: []string{"You don't see that here."}}
}

// ---- TURN command (book page-turning) ----

func (e *GameEngine) doTurnPage(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return nil // fall through to item interaction ("Turn what?")
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	// Check if the target word is "page" — shorthand for turning whatever book is around
	isPageKeyword := (target == "page")

	room := e.rooms[player.RoomNumber]

	// Search room items for a book (Val2 > 0 indicates total pages)
	if room != nil {
		for i, ri := range room.Items {
			itemDef := e.items[ri.Archetype]
			if itemDef == nil {
				continue
			}
			name := e.getItemNounName(itemDef)
			if isPageKeyword {
				// "turn page" — match any book in the room
				if ri.Val2 <= 0 {
					continue
				}
			} else {
				if !matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
					continue
				}
			}
			if skip > 0 {
				skip--
				continue
			}
			// Check if it's a book (has Val2 = total pages > 0)
			if ri.Val2 <= 0 {
				return nil // not a book, fall through to normal item interaction
			}
			// Increment page, wrap around
			currentPage := ri.Val1
			totalPages := ri.Val2
			currentPage++
			if currentPage > totalPages {
				currentPage = 1
			}
			room.Items[i].Val1 = currentPage
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_update", ItemRef: ri.Ref, Item: &room.Items[i]})
			return &CommandResult{
				Messages:      []string{"You carefully turn the page."},
				RoomBroadcast: []string{fmt.Sprintf("%s turns a page.", player.FirstName)},
			}
		}
	}

	// Search player inventory
	skip = ordSkip
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if isPageKeyword {
			if ii.Val2 <= 0 {
				continue
			}
		} else {
			if !matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
				continue
			}
		}
		if skip > 0 {
			skip--
			continue
		}
		if ii.Val2 <= 0 {
			return nil // not a book, fall through
		}
		currentPage := ii.Val1
		totalPages := ii.Val2
		currentPage++
		if currentPage > totalPages {
			currentPage = 1
		}
		player.Inventory[i].Val1 = currentPage
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      []string{"You carefully turn the page."},
			RoomBroadcast: []string{fmt.Sprintf("%s turns a page.", player.FirstName)},
		}
	}

	if isPageKeyword {
		return &CommandResult{Messages: []string{"You can't turn pages on that."}}
	}

	return nil // fall through to item interaction
}

// ---- FILL command ----

func (e *GameEngine) doFill(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"You don't have anything to fill."}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	// Find the container in player inventory (glass, cup, flask, mug, bottle, tankard, vial)
	fillableNouns := map[string]bool{"glass": true, "cup": true, "flask": true, "mug": true, "bottle": true, "tankard": true, "vial": true, "goblet": true, "chalice": true, "stein": true}
	var fillIdx int = -1
	var fillItem *InventoryItem
	var fillDef *gameworld.ItemDef
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
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
		// Check if it's a fillable container type
		if itemDef.Type == "LIQCONTAINER" || fillableNouns[strings.ToLower(name)] {
			fillIdx = i
			fillItem = &player.Inventory[i]
			fillDef = itemDef
			break
		}
	}
	if fillIdx < 0 || fillItem == nil {
		return &CommandResult{Messages: []string{"You don't have anything to fill."}}
	}

	// Find a source in the room (keg, barrel, fountain, well, spring, cauldron)
	sourceNouns := map[string]bool{"keg": true, "barrel": true, "fountain": true, "well": true, "spring": true, "cauldron": true, "cask": true, "tap": true, "spigot": true}
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"There is nothing to fill from here."}}
	}
	var sourceDef *gameworld.ItemDef
	var sourceRI *gameworld.RoomItem
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := strings.ToLower(e.getItemNounName(itemDef))
		if sourceNouns[name] || itemDef.Type == "LIQUID" || itemDef.Type == "LIQCONTAINER" {
			sourceDef = itemDef
			sourceRI = &room.Items[i]
			break
		}
	}
	if sourceDef == nil || sourceRI == nil {
		return &CommandResult{Messages: []string{"There is nothing to fill from here."}}
	}

	// Determine the drink type from the source
	drinkType := "water"
	sourceName := strings.ToLower(e.getItemNounName(sourceDef))
	if sourceRI.Extend != "" {
		drinkType = strings.ToLower(sourceRI.Extend)
	} else if strings.Contains(sourceName, "keg") || strings.Contains(sourceName, "barrel") || strings.Contains(sourceName, "cask") {
		drinkType = "ale"
	} else if strings.Contains(sourceName, "cauldron") {
		drinkType = "broth"
	}

	// Fill the container — set Val2 to a default number of sips
	fillItem.Val2 = 5
	fillItem.State = "filled"
	e.SavePlayer(ctx, player)

	displayName := e.formatItemNameNoArticle(fillDef, fillItem.Adj1, fillItem.Adj2, fillItem.Adj3)
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You fill the %s with some %s.", displayName, drinkType)},
		RoomBroadcast: []string{fmt.Sprintf("%s fills a %s with some %s.", player.FirstName, displayName, drinkType)},
	}
}
