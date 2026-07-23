package engine

// containers.go — full container system
//
// Storage model
// ─────────────
// Inventory containers: Contents []InventoryItem on InventoryItem (added to player.go).
//   Persists to MongoDB automatically because InventoryItem is the bson-serialised type.
//
// Room containers (bins, dropped chests, etc.): transient map on GameEngine:
//   roomContainerContents map[string][]InventoryItem
//   key = "<roomNumber>:<itemRef>"
// This is intentionally transient. Room containers are short-lived (monster drops, bins
// that reset when a room resets). If you need persistence, serialise this map to MongoDB.
//
// Container capacity is governed by two ItemDef fields:
//   Interior  int  — max number of items
//   Volume    int  — max total volume of all items combined
// Both come from the script parser via gameworld.ItemDef. If Interior == 0 the container
// is treated as unlimited-item-count (volume still applies).

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// ── helper: is this item definition a container? ────────────────────────────

func isContainerDef(def *gameworld.ItemDef) bool {
	return def.Container == "IN" || def.Container == "ON" ||
		containsFlag(def.Flags, "CONTAINER") || def.Type == "CONTAINER"
}

// containerIsOpen reports whether a container's contents are currently accessible.
// Containers without the OPENABLE flag (mugs, chalices, cauldrons, pools) have no lid
// to close and are always accessible. Only OPENABLE containers need their state to be "OPEN".
func containerIsOpen(def *gameworld.ItemDef, state string) bool {
	if !containsFlag(def.Flags, "OPENABLE") {
		return true
	}
	return strings.ToUpper(state) == "OPEN"
}

// containerKey returns the map key for a room-level container.
func containerKey(roomNum, itemRef int) string {
	return fmt.Sprintf("%d:%d", roomNum, itemRef)
}

// ── GameEngine initialisation ────────────────────────────────────────────────

// initContainerMap must be called from NewGameEngine (add one line there).
func (e *GameEngine) initContainerMap() {
	if e.roomContainerContents == nil {
		e.roomContainerContents = make(map[string][]InventoryItem)
	}
}

// roomContainerGet returns the contents slice for a room container (nil if empty/unknown).
func (e *GameEngine) roomContainerGet(roomNum, itemRef int) []InventoryItem {
	if e.roomContainerContents == nil {
		return nil
	}
	return e.roomContainerContents[containerKey(roomNum, itemRef)]
}

// roomContainerSet overwrites the contents for a room container.
func (e *GameEngine) roomContainerSet(roomNum, itemRef int, contents []InventoryItem) {
	if e.roomContainerContents == nil {
		e.roomContainerContents = make(map[string][]InventoryItem)
	}
	if len(contents) == 0 {
		delete(e.roomContainerContents, containerKey(roomNum, itemRef))
	} else {
		e.roomContainerContents[containerKey(roomNum, itemRef)] = contents
	}
}

// roomContainerDelete removes the entry for a room container (e.g. when it is picked up).
func (e *GameEngine) roomContainerDelete(roomNum, itemRef int) {
	if e.roomContainerContents != nil {
		delete(e.roomContainerContents, containerKey(roomNum, itemRef))
	}
}

// ── capacity checking ────────────────────────────────────────────────────────

// containerVolumeUsed sums the Volume field of all items in a contents slice.
func (e *GameEngine) containerVolumeUsed(contents []InventoryItem) int {
	total := 0
	for _, item := range contents {
		if def := e.items[item.Archetype]; def != nil {
			total += def.Volume
		}
	}
	return total
}

// roomContainerContentsWeight sums the weight of everything currently inside a room-level
// container: both items placed at runtime via the PUT command (roomContainerContents) and
// items authored directly into the container by the script (PUT <ref> <archetype> lines,
// stored as RoomItem entries with IsPut && PutIn == ref). Used so that OBJWEIGHT reflects a
// container's true accumulated weight (e.g. boulders dropped into a stream to build a dam)
// rather than just the container's own static definition weight.
func (e *GameEngine) roomContainerContentsWeight(room *gameworld.Room, ref int) int {
	total := 0
	for _, ii := range e.roomContainerGet(room.Number, ref) {
		if def := e.items[ii.Archetype]; def != nil {
			total += def.Weight
		}
	}
	if room != nil {
		for _, ri2 := range room.Items {
			if !ri2.IsPut || ri2.PutIn != ref {
				continue
			}
			if def := e.items[ri2.Archetype]; def != nil {
				total += def.Weight
			}
		}
	}
	return total
}

// containerCanFit returns ("", true) if item fits, or an error message if not.
func (e *GameEngine) containerCanFit(containerDef *gameworld.ItemDef, contents []InventoryItem, itemDef *gameworld.ItemDef) (string, bool) {
	// Interior == 0 means no item-count limit in the scripts
	if containerDef.Interior > 0 && len(contents) >= containerDef.Interior {
		return "There isn't enough room in there.", false
	}
	if containerDef.Volume > 0 {
		used := e.containerVolumeUsed(contents)
		if used+itemDef.Volume > containerDef.Volume {
			return "That won't fit in there.", false
		}
	}
	return "", true
}

// ── container name with open/closed tag ─────────────────────────────────────

// formatContainerName returns the item name with "(open)" or "(closed)" appended
// when the item is a container. Non-containers are returned unchanged.
func (e *GameEngine) formatContainerName(def *gameworld.ItemDef, adj1, adj2, adj3 int, state string, tail ...string) string {
	t := ""
	if len(tail) > 0 {
		t = tail[0]
	}
	base := e.formatItemName(def, adj1, adj2, adj3, t)
	if !isContainerDef(def) {
		return base
	}
	if strings.ToUpper(state) == "LOCKED" {
		return base + " (locked)"
	}
	if containerIsOpen(def, state) {
		return base + " (open)"
	}
	return base + " (closed)"
}

// ── Potions ───────────────────────────────────────────────────────────────
// Potion containers (bottle/flask/vial) store their liquid directly on the
// vessel: Val2 = sips remaining, Val3 = spell ID cast on drinking, Val4 = the
// liquid's appearance adjective (e.g. "reeking", "colorless"). All potion
// containers share the same capacity regardless of archetype.
const potionContainerCapacity = 12

// potionFullnessDesc describes how full a potion container is: full, 3/4 full,
// half full, or 1/4 full — down to "almost empty" specifically when one sip remains.
func potionFullnessDesc(sips, capacity int) string {
	if capacity <= 0 {
		capacity = potionContainerCapacity
	}
	if sips == 1 {
		return "almost empty"
	}
	frac := float64(sips) / float64(capacity)
	switch {
	case frac >= 0.875:
		return "full"
	case frac >= 0.625:
		return "3/4 full"
	case frac >= 0.375:
		return "half full"
	default:
		return "1/4 full"
	}
}

// potionLookInMessages builds the "LOOK IN <potion container>" response: fullness,
// then what liquid is visible inside (named by its Val4 appearance adjective).
// adj1 is the vessel's own material adjective (if any), used so the container
// reference matches its normal display name (e.g. "the glass flask").
func (e *GameEngine) potionLookInMessages(def *gameworld.ItemDef, adj1, sips, liquidAdjID int, tail string) []string {
	containerRef := "the " + e.formatItemNameNoArticle(def, adj1, 0, 0, tail)
	if sips <= 0 {
		return []string{fmt.Sprintf("%s is empty.", capitalize(containerRef))}
	}
	fullnessMsg := fmt.Sprintf("It is %s.", potionFullnessDesc(sips, potionContainerCapacity))
	liquidWord := e.getAdjName(liquidAdjID)
	if liquidAdjID == 0 || liquidWord == "" {
		// Plain filled liquid (e.g. water/ale from FILL) — not a magical potion.
		return []string{fullnessMsg, fmt.Sprintf("In %s you see some liquid.", containerRef)}
	}
	return []string{
		fullnessMsg,
		fmt.Sprintf("In %s you see a %s potion.", containerRef, liquidWord),
	}
}

// matchesItemOrPotion reports whether an item matches target — either by its own
// name/adjectives, or, if it's a LIQCONTAINER currently holding a potion, by the
// potion's liquid-appearance phrase (e.g. "crimson potion"). This lets players
// reference a potion by what it looks like, regardless of which vessel — vial,
// flask, or bottle — currently holds it.
func (e *GameEngine) matchesItemOrPotion(def *gameworld.ItemDef, adj1, adj2, adj3, val2, val4 int, target string) bool {
	if def == nil {
		return false
	}
	if matchesTarget(e.getItemNounName(def), target, e.getAdjName(adj1), e.getAdjName(adj2), e.getAdjName(adj3)) {
		return true
	}
	if def.Type == "LIQCONTAINER" && val2 > 0 && val4 != 0 {
		return matchesTarget("potion", target, e.getAdjName(val4))
	}
	return false
}

// potionPhraseIfAny returns the "<color> potion" description for an item that's a
// LIQCONTAINER currently holding a potion, or "" if not applicable. Used by GM
// commands (@iexamine, @editem) which match by substring against a formatted name
// rather than matchesTarget.
func (e *GameEngine) potionPhraseIfAny(def *gameworld.ItemDef, val2, val4 int) string {
	if def == nil || def.Type != "LIQCONTAINER" || val2 <= 0 || val4 == 0 {
		return ""
	}
	adjName := e.getAdjName(val4)
	if adjName == "" {
		return ""
	}
	return adjName + " potion"
}

// findOpenContainers returns pointers to every item in inv that is currently an
// open container, so callers can search one level into carried Contents (e.g. a
// vial of potion sitting inside an open backpack).
func (e *GameEngine) findOpenContainers(inv []InventoryItem) []*InventoryItem {
	var out []*InventoryItem
	for i := range inv {
		def := e.items[inv[i].Archetype]
		if def == nil || !isContainerDef(def) || !containerIsOpen(def, inv[i].State) {
			continue
		}
		out = append(out, &inv[i])
	}
	return out
}

// ── OPEN (enhanced) ─────────────────────────────────────────────────────────
// The existing doOpen in engine.go already handles the basic open logic.
// We replace lookInContainer with a real implementation and wire container
// contents into the inventory display / GET command via the new helpers here.

// lookInContainer is called from doLookAt when a player types LOOK IN <container>.
// This replaces the stub in engine.go.
func (e *GameEngine) lookInContainer(player *Player, def *gameworld.ItemDef, ii *InventoryItem) *CommandResult {
	displayName := e.formatContainerName(def, ii.Adj1, ii.Adj2, ii.Adj3, ii.State, ii.Tail)

	state := strings.ToUpper(ii.State)
	if state == "LOCKED" {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s is locked.", displayName)}}
	}
	if !containerIsOpen(def, ii.State) {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s is closed.", displayName)}}
	}

	if def.Type == "LIQCONTAINER" {
		return &CommandResult{Messages: e.potionLookInMessages(def, ii.Adj1, ii.Val2, ii.Val4, ii.Tail)}
	}

	if len(ii.Contents) == 0 {
		return &CommandResult{Messages: []string{fmt.Sprintf("You look inside %s. It is empty.", displayName)}}
	}

	msgs := []string{fmt.Sprintf("Inside %s you see:", displayName)}
	for _, ci := range ii.Contents {
		if ci.State == "MONEY" {
			msgs = append(msgs, "  "+formatCoinStr(ci.Val1))
			continue
		}
		ciDef := e.items[ci.Archetype]
		if ciDef == nil {
			continue
		}
		msgs = append(msgs, "  "+e.formatItemName(ciDef, ci.Adj1, ci.Adj2, ci.Adj3, ci.Tail))
	}
	return &CommandResult{Messages: msgs}
}

// lookInRoomContainer is called when a player types LOOK IN <container> for a room item.
func (e *GameEngine) lookInRoomContainer(player *Player, def *gameworld.ItemDef, ri *gameworld.RoomItem) *CommandResult {
	displayName := e.formatContainerName(def, ri.Adj1, ri.Adj2, ri.Adj3, ri.State, ri.Extend)
	state := strings.ToUpper(ri.State)
	if state == "LOCKED" {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s is locked.", displayName)}}
	}
	if !containerIsOpen(def, ri.State) {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s is closed.", displayName)}}
	}

	if def.Type == "LIQCONTAINER" {
		return &CommandResult{Messages: e.potionLookInMessages(def, ri.Adj1, ri.Val2, ri.Val4, ri.Extend)}
	}

	// Dynamic contents (dropped loot, items placed at runtime via PUT)
	contents := e.roomContainerGet(player.RoomNumber, ri.Ref)

	// Script-placed contents (IsPut == true, PutIn == this container's Ref)
	room := e.rooms[player.RoomNumber]
	if room != nil {
		for _, ri2 := range room.Items {
			if !ri2.IsPut || ri2.PutIn != ri.Ref {
				continue
			}
			iDef := e.items[ri2.Archetype]
			if iDef == nil {
				continue
			}
			contents = append(contents, InventoryItem{
				Archetype: ri2.Archetype,
				Adj1:      ri2.Adj1, Adj2: ri2.Adj2, Adj3: ri2.Adj3,
				Val1: ri2.Val1, Val2: ri2.Val2, Val3: ri2.Val3, Val4: ri2.Val4, Val5: ri2.Val5,
				Sharpness: ri2.Sharpness,
				HardnessMod: ri2.HardnessMod,
				ItemBits:  ri2.ItemBits,
				State:     ri2.State,
			})
		}
	}

	if len(contents) == 0 {
		return &CommandResult{Messages: []string{fmt.Sprintf("You look inside %s. It is empty.", displayName)}}
	}

	msgs := []string{fmt.Sprintf("Inside %s you see:", displayName)}
	for _, ci := range contents {
		if ci.State == "MONEY" {
			msgs = append(msgs, "  "+formatCoinStr(ci.Val1))
			continue
		}
		ciDef := e.items[ci.Archetype]
		if ciDef == nil {
			continue
		}
		msgs = append(msgs, "  "+e.formatItemName(ciDef, ci.Adj1, ci.Adj2, ci.Adj3, ci.Tail))
	}
	return &CommandResult{Messages: msgs}
}

// ── PUT command ──────────────────────────────────────────────────────────────
// "PUT <item> IN <container>"  — places an inventory item into a container.
// Works for containers in inventory AND containers in the room.

// runPutIntoTargetScripts executes any IFPREVERB2 PUT script defined for a PUT target
// (def) — whether that target is a room item (roomScripts + its ref) or something the
// player is carrying (no room scripts, ref "-1"). srcDef is the item being placed;
// ItemRef/ItemDef are set to it so IFVAR ARCHNUM and REMOVEITEM -1 resolve correctly.
// handled is false when no script intercepted the verb (CLEARVERB/MOVE), meaning the
// caller should fall through to normal container-put handling for def.
func (e *GameEngine) runPutIntoTargetScripts(ctx context.Context, player *Player, room *gameworld.Room, def *gameworld.ItemDef, srcDef *gameworld.ItemDef, srcItem InventoryItem, roomScripts []gameworld.ScriptBlock, refStr string) (result *CommandResult, handled bool) {
	sc := &ScriptContext{
		Player: player,
		Room:   room,
		Engine: e,
		ItemRef: &gameworld.RoomItem{
			Archetype: srcDef.Number, Ref: -1,
			Adj1: srcItem.Adj1, Adj2: srcItem.Adj2, Adj3: srcItem.Adj3,
			Val1: srcItem.Val1, Val2: srcItem.Val2, Val3: srcItem.Val3, Val4: srcItem.Val4, Val5: srcItem.Val5,
			Sharpness: srcItem.Sharpness, HardnessMod: srcItem.HardnessMod, ItemBits: srcItem.ItemBits, State: srcItem.State,
		},
		ItemDef: srcDef,
	}
	isPutBlock := func(block gameworld.ScriptBlock) bool {
		return block.Type == "IFPREVERB2" && len(block.Args) >= 2 &&
			strings.ToUpper(block.Args[0]) == "PUT" &&
			(block.Args[1] == refStr || block.Args[1] == "-1")
	}
	for _, block := range roomScripts {
		if isPutBlock(block) {
			sc.execBlock(block)
		}
	}
	// Item-defined IFPREVERB2 PUT scripts — e.g. the druid's slime mold (item 309)
	// defines its dissolve-on-PUT script once on the item itself rather than
	// duplicating it per room instance, and it applies whether the mold is sitting
	// in a room or being carried in the player's own inventory.
	for _, block := range def.Scripts {
		if isPutBlock(block) {
			sc.execBlock(block)
		}
	}
	if sc.NeedsSave {
		e.SavePlayer(ctx, player)
	}
	// PLREVENT/CONTPLREVENT-deferred actions (e.g. the garment-finishing contraption's
	// close-wait-reopen sequence) must be scheduled, or everything after the delay is lost.
	if len(sc.DeferredSegments) > 0 {
		e.scheduleScriptSegments(player, sc.DeferredSegments)
	}
	// A matched IFPREVERB2 PUT block "handles" the put whenever it actually did
	// something — echoed a message, blocked the action, moved the player, or deferred
	// further steps — not just when it explicitly CLEARVERBed. Without this, a
	// no-CLEARVERB success branch (e.g. one that echoes progress and defers the rest via
	// PLREVENT) falls through to the generic container-transfer code below, which
	// silently discards the script's messages and duplicates/overrides its own handling.
	if !sc.Blocked && sc.MoveTo == 0 && len(sc.Messages) == 0 && len(sc.RoomMsgs) == 0 &&
		len(sc.GMMsgs) == 0 && len(sc.DeferredSegments) == 0 {
		return nil, false
	}
	result = &CommandResult{Messages: sc.Messages, RoomBroadcast: sc.RoomMsgs, GMBroadcast: sc.GMMsgs}
	if sc.MoveTo > 0 {
		if dest := e.rooms[sc.MoveTo]; dest != nil {
			origRoom := sc.OrigRoomNum
			if origRoom == 0 {
				origRoom = player.RoomNumber
			}
			player.RoomNumber = sc.MoveTo
			e.SavePlayer(ctx, player)
			lookResult := e.doLook(player)
			result.Messages = append(result.Messages, lookResult.Messages...)
			result.RoomName = lookResult.RoomName
			result.RoomDesc = lookResult.RoomDesc
			result.Exits = lookResult.Exits
			result.Items = lookResult.Items
			result.OldRoom = origRoom
			result.OldRoomMsg = []string{fmt.Sprintf("%s leaves.", player.FirstName)}
			result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s arrives.", player.FirstName))
			e.applyEntryScripts(ctx, player, dest, result)
		}
	}
	return result, true
}

func (e *GameEngine) doPut(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Put what?"}}
	}

	// Expect "PUT <item> IN <container>"
	raw := strings.ToLower(strings.Join(args, " "))
	inIdx := strings.Index(raw, " in ")
	if inIdx < 0 {
		return &CommandResult{Messages: []string{"Put what in what? (usage: PUT <item> IN <container>)"}}
	}
	itemTarget := strings.TrimSpace(raw[:inIdx])
	containerTarget := strings.TrimSpace(raw[inIdx+4:])

	if itemTarget == "" || containerTarget == "" {
		return &CommandResult{Messages: []string{"Put what in what? (usage: PUT <item> IN <container>)"}}
	}

	itemTarget, _ = stripMyPrefix(itemTarget)
	containerTarget, containerMine := stripMyPrefix(containerTarget)
	itemTarget, itemSkip := parseOrdinal(itemTarget)
	containerTarget, _ = parseOrdinal(containerTarget)

	// Find the item to put (must be in inventory, not wielded or worn)
	itemIdx := -1
	var srcItem InventoryItem
	var srcDef *gameworld.ItemDef
	skip := itemSkip
	for i, ii := range player.Inventory {
		def := e.items[ii.Archetype]
		if def == nil {
			continue
		}
		if matchesTarget(e.getItemNounName(def), itemTarget, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
			if skip > 0 {
				skip--
				continue
			}
			itemIdx = i
			srcItem = ii
			srcDef = def
			break
		}
	}
	if itemIdx < 0 {
		return &CommandResult{Messages: []string{"You aren't carrying that."}}
	}

	itemFullName := e.formatItemName(srcDef, srcItem.Adj1, srcItem.Adj2, srcItem.Adj3, srcItem.Tail)

	room := e.rooms[player.RoomNumber]

	// Try inventory targets first — real containers, and non-container script items
	// (e.g. the druid's slime mold) that define an IFPREVERB2 PUT script on themselves.
	for i, ii := range player.Inventory {
		if i == itemIdx {
			continue // can't put item into itself
		}
		def := e.items[ii.Archetype]
		if def == nil {
			continue
		}
		if !matchesTarget(e.getItemNounName(def), containerTarget, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
			continue
		}

		if result, handled := e.runPutIntoTargetScripts(ctx, player, room, def, srcDef, srcItem, nil, "-1"); handled {
			return result
		}

		if !isContainerDef(def) {
			continue
		}
		state := strings.ToUpper(ii.State)
		if state == "LOCKED" {
			return &CommandResult{Messages: []string{"That container is locked."}}
		}
		if !containerIsOpen(def, ii.State) {
			return &CommandResult{Messages: []string{"That container is closed."}}
		}
		if errMsg, ok := e.containerCanFit(def, ii.Contents, srcDef); !ok {
			return &CommandResult{Messages: []string{errMsg}}
		}
		// Transfer
		player.Inventory[i].Contents = append(player.Inventory[i].Contents, srcItem)
		player.Inventory = append(player.Inventory[:itemIdx], player.Inventory[itemIdx+1:]...)
		e.SavePlayer(ctx, player)
		cName := e.formatContainerName(def, ii.Adj1, ii.Adj2, ii.Adj3, ii.State, ii.Tail)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You put %s in %s.", itemFullName, cName)},
			RoomBroadcast: []string{fmt.Sprintf("%s puts %s in %s.", player.FirstName, itemFullName, cName)},
		}
	}

	// Try room containers — skipped when "my" was used (e.g. "put ring in my chest").
	if room != nil && !containerMine {
		for _, ri := range room.Items {
			def := e.items[ri.Archetype]
			if def == nil {
				continue
			}
			if !matchesTarget(e.getItemNounName(def), containerTarget, e.getAdjName(ri.Adj1), e.getAdjName(ri.Adj2), e.getAdjName(ri.Adj3)) {
				continue
			}

			refStr := fmt.Sprintf("%d", ri.Ref)
			if result, handled := e.runPutIntoTargetScripts(ctx, player, room, def, srcDef, srcItem, room.Scripts, refStr); handled {
				return result
			}

			if !isContainerDef(def) {
				continue
			}
			state := strings.ToUpper(ri.State)
			if state == "LOCKED" {
				return &CommandResult{Messages: []string{"That container is locked."}}
			}
			if !containerIsOpen(def, ri.State) {
				return &CommandResult{Messages: []string{"That container is closed."}}
			}
			contents := e.roomContainerGet(player.RoomNumber, ri.Ref)
			if errMsg, ok := e.containerCanFit(def, contents, srcDef); !ok {
				return &CommandResult{Messages: []string{errMsg}}
			}
			contents = append(contents, srcItem)
			e.roomContainerSet(player.RoomNumber, ri.Ref, contents)
			player.Inventory = append(player.Inventory[:itemIdx], player.Inventory[itemIdx+1:]...)
			e.SavePlayer(ctx, player)
			cName := e.formatContainerName(def, ri.Adj1, ri.Adj2, ri.Adj3, ri.State, ri.Extend)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You put %s in %s.", itemFullName, cName)},
				RoomBroadcast: []string{fmt.Sprintf("%s puts %s in %s.", player.FirstName, itemFullName, cName)},
			}
		}
	}

	return &CommandResult{Messages: []string{"You don't see that container here."}}
}

// doPour handles "POUR <container> INTO <container>", moving liquid (sips, spell,
// appearance) from a carried liquid container into another. Both ends must be
// carried LIQCONTAINER items; the destination must be empty.
func (e *GameEngine) doPour(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Pour what?"}}
	}

	raw := strings.ToLower(strings.Join(args, " "))
	sepIdx := strings.Index(raw, " into ")
	sepLen := 6
	if sepIdx < 0 {
		sepIdx = strings.Index(raw, " in ")
		sepLen = 4
	}
	if sepIdx < 0 {
		return &CommandResult{Messages: []string{"Pour what into what? (usage: POUR <container> INTO <container>)"}}
	}
	srcTarget := strings.TrimSpace(raw[:sepIdx])
	dstTarget := strings.TrimSpace(raw[sepIdx+sepLen:])
	if srcTarget == "" || dstTarget == "" {
		return &CommandResult{Messages: []string{"Pour what into what? (usage: POUR <container> INTO <container>)"}}
	}
	srcTarget, srcSkip := parseOrdinal(srcTarget)
	dstTarget, dstSkip := parseOrdinal(dstTarget)

	srcIdx := -1
	var srcDef *gameworld.ItemDef
	skip := srcSkip
	for i, ii := range player.Inventory {
		def := e.items[ii.Archetype]
		if def == nil {
			continue
		}
		if !matchesTarget(e.getItemNounName(def), srcTarget, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}
		srcIdx = i
		srcDef = def
		break
	}
	if srcIdx < 0 {
		return &CommandResult{Messages: []string{"You aren't carrying that."}}
	}
	if srcDef.Type != "LIQCONTAINER" {
		return &CommandResult{Messages: []string{"You can't pour that."}}
	}
	srcName := e.formatItemName(srcDef, player.Inventory[srcIdx].Adj1, player.Inventory[srcIdx].Adj2, player.Inventory[srcIdx].Adj3, player.Inventory[srcIdx].Tail)
	if !containerIsOpen(srcDef, player.Inventory[srcIdx].State) {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s is closed.", srcName)}}
	}
	if player.Inventory[srcIdx].Val2 <= 0 {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s is empty.", srcName)}}
	}

	dstIdx := -1
	var dstDef *gameworld.ItemDef
	skip = dstSkip
	for i, ii := range player.Inventory {
		if i == srcIdx {
			continue
		}
		def := e.items[ii.Archetype]
		if def == nil {
			continue
		}
		if !matchesTarget(e.getItemNounName(def), dstTarget, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}
		dstIdx = i
		dstDef = def
		break
	}
	if dstIdx < 0 {
		return &CommandResult{Messages: []string{"You don't have that."}}
	}
	if dstDef.Type != "LIQCONTAINER" {
		return &CommandResult{Messages: []string{"You can't pour anything into that."}}
	}
	dstName := e.formatItemName(dstDef, player.Inventory[dstIdx].Adj1, player.Inventory[dstIdx].Adj2, player.Inventory[dstIdx].Adj3, player.Inventory[dstIdx].Tail)
	if !containerIsOpen(dstDef, player.Inventory[dstIdx].State) {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s is closed.", dstName)}}
	}
	if player.Inventory[dstIdx].Val2 > 0 {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s already contains something.", dstName)}}
	}

	player.Inventory[dstIdx].Val2 = player.Inventory[srcIdx].Val2
	player.Inventory[dstIdx].Val3 = player.Inventory[srcIdx].Val3
	player.Inventory[dstIdx].Val4 = player.Inventory[srcIdx].Val4
	player.Inventory[srcIdx].Val2 = 0
	player.Inventory[srcIdx].Val3 = 0
	player.Inventory[srcIdx].Val4 = 0

	e.SavePlayer(ctx, player)
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You pour the liquid from %s into %s.", srcName, dstName)},
		RoomBroadcast: []string{fmt.Sprintf("%s pours liquid from %s into %s.", player.FirstName, srcName, dstName)},
	}
}

// ── GET (enhanced) ───────────────────────────────────────────────────────────
// Extends the existing doGet with:
//   GET ALL                   — pick up everything from the floor
//   GET ALL <noun>            — pick up all matching items from floor
//   GET ALL COIN/MONEY        — pick up all coin piles
//   GET <item> FROM <container> — take one item from a container
//   GET ALL FROM <container>  — take all items from a container

func (e *GameEngine) doGetEnhanced(ctx context.Context, player *Player, verb string, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Get what?"}}
	}

	raw := strings.ToLower(strings.Join(args, " "))

	// ── GET ALL FROM <container> ──────────────────────────────────────────
	if strings.HasPrefix(raw, "all from ") {
		containerTarget := strings.TrimPrefix(raw, "all from ")
		containerTarget, _ = parseOrdinal(containerTarget)
		return e.doGetAllFromContainer(ctx, player, containerTarget)
	}

	// ── GET <item> FROM <container> ───────────────────────────────────────
	fromIdx := strings.Index(raw, " from ")
	if fromIdx > 0 {
		itemTarget := strings.TrimSpace(raw[:fromIdx])
		containerTarget := strings.TrimSpace(raw[fromIdx+6:])
		return e.doGetFromContainer(ctx, player, itemTarget, containerTarget)
	}

	// ── GET ALL ───────────────────────────────────────────────────────────
	if raw == "all" {
		return e.doGetAll(ctx, player, "")
	}

	// ── GET ALL <noun> ────────────────────────────────────────────────────
	if strings.HasPrefix(raw, "all ") {
		noun := strings.TrimPrefix(raw, "all ")
		return e.doGetAll(ctx, player, noun)
	}

	// ── standard single-item GET ──────────────────────────────────────────
	return e.doGet(ctx, player, verb, args)
}

// doGetAll picks up all matching items (or all items when noun == "") from the room floor.
func (e *GameEngine) doGetAll(ctx context.Context, player *Player, noun string) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}

	isCoinNoun := noun == "coin" || noun == "coins" || noun == "money" ||
		noun == "gold" || noun == "silver" || noun == "copper"

	var msgs []string
	var roomMsgs []string
	var kept []gameworld.RoomItem
	gotSomething := false

	for _, ri := range room.Items {
		// Coin pile — only pick up if no noun given, or noun is a coin word
		if ri.State == "MONEY" {
			if noun == "" || isCoinNoun {
				coins := ri.Val1
				if coins <= 0 {
					coins = 1
				}
				pickupMsg := e.addCoinsToPlayer(player, ri.Archetype, coins)
				msgs = append(msgs, pickupMsg)
				e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_remove", ItemRef: ri.Ref})
				gotSomething = true
			} else {
				kept = append(kept, ri)
			}
			continue
		}

		def := e.items[ri.Archetype]
		if def == nil || def.Weight >= 1000 || isPortal(def.Type) ||
			containsFlag(def.Flags, "FIXED") || def.Type == "MANUSCRIPT" {
			kept = append(kept, ri)
			continue
		}

		// If a coin noun was specified, skip all non-money items
		if isCoinNoun {
			if def.Type != "MONEY" && ri.State != "MONEY" {
				kept = append(kept, ri)
				continue
			}
		}

		// If a non-coin noun was specified, only pick up matching items
		if noun != "" && !isCoinNoun {
			if !matchesTarget(e.getItemNounName(def), noun, e.getAdjName(ri.Adj1), e.getAdjName(ri.Adj2), e.getAdjName(ri.Adj3)) {
				kept = append(kept, ri)
				continue
			}
		}

		// MONEY-type item archetype
		if def.Type == "MONEY" || ri.State == "MONEY" {
			coins := ri.Val1
			if coins <= 0 {
				coins = 1
			}
			pickupMsg := e.addCoinsToPlayer(player, ri.Archetype, coins)
			msgs = append(msgs, pickupMsg)
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_remove", ItemRef: ri.Ref})
			gotSomething = true
			continue
		}

		// Normal item — check for item guard
		fullName := e.formatItemName(def, ri.Adj1, ri.Adj2, ri.Adj3, ri.Extend)
		if guardBlocked, guardPlayerMsgs, guardRoomBroadcasts := e.checkItemGuard(player, ri.Archetype, fullName); guardBlocked {
			msgs = append(msgs, guardPlayerMsgs...)
			roomMsgs = append(roomMsgs, guardRoomBroadcasts...)
			kept = append(kept, ri)
			continue
		} else if len(guardRoomBroadcasts) > 0 {
			// Guard bypassed — show bypass info to mover and echo to room
			msgs = append(msgs, guardPlayerMsgs...)
			roomMsgs = append(roomMsgs, guardRoomBroadcasts...)
		}

		newInvItem := InventoryItem{
			Archetype: ri.Archetype,
			Adj1:      ri.Adj1, Adj2: ri.Adj2, Adj3: ri.Adj3,
			Val1: ri.Val1, Val2: ri.Val2, Val3: ri.Val3, Val4: ri.Val4, Val5: ri.Val5,
			Sharpness: ri.Sharpness,
			HardnessMod: ri.HardnessMod,
			ItemBits:  ri.ItemBits,
			State:     ri.State,
		}
		if isContainerDef(def) {
			newInvItem.Contents = e.roomContainerGet(player.RoomNumber, ri.Ref)
			e.roomContainerDelete(player.RoomNumber, ri.Ref)
		}
		player.Inventory = append(player.Inventory, newInvItem)
		msgs = append(msgs, fmt.Sprintf("You pick up %s.", fullName))
		e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_remove", ItemRef: ri.Ref})
		gotSomething = true
	}

	room.Items = kept

	if !gotSomething && len(msgs) == 0 {
		if noun != "" {
			return &CommandResult{Messages: []string{fmt.Sprintf("You don't see any %s here.", noun)}}
		}
		return &CommandResult{Messages: []string{"There is nothing here to pick up."}}
	}

	if gotSomething {
		e.SavePlayer(ctx, player)
	}
	var finalRoomMsgs []string
	finalRoomMsgs = append(finalRoomMsgs, roomMsgs...)
	if gotSomething {
		finalRoomMsgs = append(finalRoomMsgs, fmt.Sprintf("%s picks up several items.", player.FirstName))
	}
	return &CommandResult{
		Messages:      msgs,
		RoomBroadcast: finalRoomMsgs,
	}
}

// doGetFromContainer takes one item from a named container.
func (e *GameEngine) doGetFromContainer(ctx context.Context, player *Player, itemTarget, containerTarget string) *CommandResult {
	itemTarget, _ = stripMyPrefix(itemTarget)
	containerTarget, containerMine := stripMyPrefix(containerTarget)
	itemTarget, itemSkip := parseOrdinal(itemTarget)
	containerTarget, _ = parseOrdinal(containerTarget)

	// ── Inventory containers ──────────────────────────────────────────────
	for ci, container := range player.Inventory {
		cDef := e.items[container.Archetype]
		if cDef == nil || !isContainerDef(cDef) {
			continue
		}
		if !matchesTarget(e.getItemNounName(cDef), containerTarget, e.getAdjName(container.Adj1), e.getAdjName(container.Adj2), e.getAdjName(container.Adj3)) {
			continue
		}
		if strings.ToUpper(container.State) == "LOCKED" {
			return &CommandResult{Messages: []string{"That container is locked."}}
		}
		if !containerIsOpen(cDef, container.State) {
			return &CommandResult{Messages: []string{"That container is closed."}}
		}
		skip := itemSkip
		for ii, item := range container.Contents {
			iDef := e.items[item.Archetype]
			if iDef == nil {
				continue
			}
			if !matchesTarget(e.getItemNounName(iDef), itemTarget, e.getAdjName(item.Adj1), e.getAdjName(item.Adj2), e.getAdjName(item.Adj3)) {
				continue
			}
			if skip > 0 {
				skip--
				continue
			}
			// Move item from container to top-level inventory
			player.Inventory[ci].Contents = append(container.Contents[:ii], container.Contents[ii+1:]...)
			player.Inventory = append(player.Inventory, item)
			e.SavePlayer(ctx, player)
			iName := e.formatItemName(iDef, item.Adj1, item.Adj2, item.Adj3, item.Tail)
			cName := e.formatContainerName(cDef, container.Adj1, container.Adj2, container.Adj3, container.State, container.Tail)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You take %s from %s.", iName, cName)},
				RoomBroadcast: []string{fmt.Sprintf("%s takes something from %s.", player.FirstName, cName)},
			}
		}
		return &CommandResult{Messages: []string{"You don't see that in there."}}
	}

	// ── Room containers (skipped when "my" was used) ────────────────────────
	room := e.rooms[player.RoomNumber]
	if room != nil && !containerMine {
		for _, ri := range room.Items {
			cDef := e.items[ri.Archetype]
			if cDef == nil || !isContainerDef(cDef) {
				continue
			}
			if !matchesTarget(e.getItemNounName(cDef), containerTarget, e.getAdjName(ri.Adj1), e.getAdjName(ri.Adj2), e.getAdjName(ri.Adj3)) {
				continue
			}
			if strings.ToUpper(ri.State) == "LOCKED" {
				return &CommandResult{Messages: []string{"That container is locked."}}
			}
			if !containerIsOpen(cDef, ri.State) {
				return &CommandResult{Messages: []string{"That container is closed."}}
			}
			contents := e.roomContainerGet(player.RoomNumber, ri.Ref)
			skip := itemSkip
			for ii, item := range contents {
				iDef := e.items[item.Archetype]
				if iDef == nil {
					continue
				}
				if !matchesTarget(e.getItemNounName(iDef), itemTarget, e.getAdjName(item.Adj1), e.getAdjName(item.Adj2), e.getAdjName(item.Adj3)) {
					continue
				}
				if skip > 0 {
					skip--
					continue
				}
				contents = append(contents[:ii], contents[ii+1:]...)
				e.roomContainerSet(player.RoomNumber, ri.Ref, contents)
				player.Inventory = append(player.Inventory, item)
				e.SavePlayer(ctx, player)
				iName := e.formatItemName(iDef, item.Adj1, item.Adj2, item.Adj3, item.Tail)
				cName := e.formatContainerName(cDef, ri.Adj1, ri.Adj2, ri.Adj3, ri.State, ri.Extend)
				return &CommandResult{
					Messages:      []string{fmt.Sprintf("You take %s from %s.", iName, cName)},
					RoomBroadcast: []string{fmt.Sprintf("%s takes something from %s.", player.FirstName, cName)},
				}
			}
			return &CommandResult{Messages: []string{"You don't see that in there."}}
		}
	}

	return &CommandResult{Messages: []string{"You don't see that container here."}}
}

// doGetAllFromContainer empties a container into the player's inventory.
func (e *GameEngine) doGetAllFromContainer(ctx context.Context, player *Player, containerTarget string) *CommandResult {
	containerTarget, containerMine := stripMyPrefix(containerTarget)
	// ── Inventory containers ──────────────────────────────────────────────
	for ci, container := range player.Inventory {
		cDef := e.items[container.Archetype]
		if cDef == nil || !isContainerDef(cDef) {
			continue
		}
		if !matchesTarget(e.getItemNounName(cDef), containerTarget, e.getAdjName(container.Adj1), e.getAdjName(container.Adj2), e.getAdjName(container.Adj3)) {
			continue
		}
		if strings.ToUpper(container.State) == "LOCKED" {
			return &CommandResult{Messages: []string{"That container is locked."}}
		}
		if !containerIsOpen(cDef, container.State) {
			return &CommandResult{Messages: []string{"That container is closed."}}
		}
		if len(container.Contents) == 0 {
			cName := e.formatContainerName(cDef, container.Adj1, container.Adj2, container.Adj3, container.State, container.Tail)
			return &CommandResult{Messages: []string{fmt.Sprintf("%s is empty.", cName)}}
		}
		var msgs []string
		for _, item := range container.Contents {
			iDef := e.items[item.Archetype]
			if iDef == nil {
				continue
			}
			player.Inventory = append(player.Inventory, item)
			msgs = append(msgs, fmt.Sprintf("You take %s.", e.formatItemName(iDef, item.Adj1, item.Adj2, item.Adj3, item.Tail)))
		}
		player.Inventory[ci].Contents = nil
		e.SavePlayer(ctx, player)
		cName := e.formatContainerName(cDef, container.Adj1, container.Adj2, container.Adj3, container.State, container.Tail)
		return &CommandResult{
			Messages:      msgs,
			RoomBroadcast: []string{fmt.Sprintf("%s empties %s.", player.FirstName, cName)},
		}
	}

	// ── Room containers (skipped when "my" was used) ────────────────────────
	room := e.rooms[player.RoomNumber]
	if room != nil && !containerMine {
		for _, ri := range room.Items {
			cDef := e.items[ri.Archetype]
			if cDef == nil || !isContainerDef(cDef) {
				continue
			}
			if !matchesTarget(e.getItemNounName(cDef), containerTarget, e.getAdjName(ri.Adj1), e.getAdjName(ri.Adj2), e.getAdjName(ri.Adj3)) {
				continue
			}
			if strings.ToUpper(ri.State) == "LOCKED" {
				return &CommandResult{Messages: []string{"That container is locked."}}
			}
			if !containerIsOpen(cDef, ri.State) {
				return &CommandResult{Messages: []string{"That container is closed."}}
			}
			contents := e.roomContainerGet(player.RoomNumber, ri.Ref)
			if len(contents) == 0 {
				cName := e.formatContainerName(cDef, ri.Adj1, ri.Adj2, ri.Adj3, ri.State, ri.Extend)
				return &CommandResult{Messages: []string{fmt.Sprintf("%s is empty.", cName)}}
			}
			var msgs []string
			for _, item := range contents {
				iDef := e.items[item.Archetype]
				if iDef == nil {
					continue
				}
				player.Inventory = append(player.Inventory, item)
				msgs = append(msgs, fmt.Sprintf("You take %s.", e.formatItemName(iDef, item.Adj1, item.Adj2, item.Adj3, item.Tail)))
			}
			e.roomContainerSet(player.RoomNumber, ri.Ref, nil)
			e.SavePlayer(ctx, player)
			cName := e.formatContainerName(cDef, ri.Adj1, ri.Adj2, ri.Adj3, ri.State, ri.Extend)
			return &CommandResult{
				Messages:      msgs,
				RoomBroadcast: []string{fmt.Sprintf("%s empties %s.", player.FirstName, cName)},
			}
		}
	}

	return &CommandResult{Messages: []string{"You don't see that container here."}}
}

// ── DUMP command ─────────────────────────────────────────────────────────────
// DUMP <container> — empties a carriable inventory container onto the floor.
// Immovable containers (weight >= 1000, e.g. bins) reject the command.

func (e *GameEngine) doDump(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Dump what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}

	for ci, container := range player.Inventory {
		cDef := e.items[container.Archetype]
		if cDef == nil || !isContainerDef(cDef) {
			continue
		}
		if !matchesTarget(e.getItemNounName(cDef), target, e.getAdjName(container.Adj1), e.getAdjName(container.Adj2), e.getAdjName(container.Adj3)) {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}
		if cDef.Weight >= 1000 {
			return &CommandResult{Messages: []string{"You can't dump that — it's too heavy to move."}}
		}
		if len(container.Contents) == 0 {
			cName := e.formatContainerName(cDef, container.Adj1, container.Adj2, container.Adj3, container.State, container.Tail)
			return &CommandResult{Messages: []string{fmt.Sprintf("%s is already empty.", cName)}}
		}

		var msgs []string
		for _, item := range container.Contents {
			iDef := e.items[item.Archetype]
			droppedItem := gameworld.RoomItem{
				Ref:       len(room.Items),
				Archetype: item.Archetype,
				Adj1:      item.Adj1, Adj2: item.Adj2, Adj3: item.Adj3,
				Val1: item.Val1, Val2: item.Val2, Val3: item.Val3, Val4: item.Val4, Val5: item.Val5,
				Sharpness: item.Sharpness,
				HardnessMod: item.HardnessMod,
				ItemBits:  item.ItemBits,
				State:     item.State,
			}
			room.Items = append(room.Items, droppedItem)
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_add", Item: &droppedItem})
			if iDef != nil {
				msgs = append(msgs, fmt.Sprintf("%s tumbles out.", e.formatItemName(iDef, item.Adj1, item.Adj2, item.Adj3, item.Tail)))
			}
		}
		player.Inventory[ci].Contents = nil
		e.SavePlayer(ctx, player)

		cName := e.formatContainerName(cDef, container.Adj1, container.Adj2, container.Adj3, container.State, container.Tail)
		msgs = append([]string{fmt.Sprintf("You dump out %s.", cName)}, msgs...)
		return &CommandResult{
			Messages:      msgs,
			RoomBroadcast: []string{fmt.Sprintf("%s dumps out %s.", player.FirstName, cName)},
		}
	}

	return &CommandResult{Messages: []string{"You aren't carrying that."}}
}

// ── SELL ALL <noun> ──────────────────────────────────────────────────────────

func (e *GameEngine) doSellAll(ctx context.Context, player *Player, noun string) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't sell anything here."}}
	}
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

	target, _ := parseOrdinal(noun)

	var kept []InventoryItem
	totalValue := 0
	sold := 0

	for _, ii := range player.Inventory {
		def := e.items[ii.Archetype]
		if def == nil {
			kept = append(kept, ii)
			continue
		}
		if !matchesTarget(e.getItemNounName(def), target, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
			kept = append(kept, ii)
			continue
		}
		sellValue := ii.Val1
		if sellValue <= 0 {
			sellValue = def.Weight + 1
		}
		sellValue = sellValue / 2
		if sellValue < 1 {
			sellValue = 1
		}
		totalValue += sellValue
		sold++
	}

	if sold == 0 {
		return &CommandResult{Messages: []string{fmt.Sprintf("You don't have any %s to sell.", noun)}}
	}

	player.Inventory = kept
	player.Gold += totalValue / 100
	player.Silver += (totalValue % 100) / 10
	player.Copper += totalValue % 10
	e.SavePlayer(ctx, player)

	return &CommandResult{Messages: []string{
		fmt.Sprintf("You sell %d %s.", sold, noun),
		fmt.Sprintf("The merchant hands you %s.", formatPrice(totalValue)),
	}}
}

// formatCoinStr converts a copper-unit coin value to a readable string.
func formatCoinStr(val int) string {
	var parts []string
	if g := val / 100; g > 0 {
		parts = append(parts, fmt.Sprintf("%d gold", g))
	}
	if s := (val % 100) / 10; s > 0 {
		parts = append(parts, fmt.Sprintf("%d silver", s))
	}
	if c := val % 10; c > 0 {
		parts = append(parts, fmt.Sprintf("%d copper", c))
	}
	if len(parts) == 0 {
		return "some coins"
	}
	return "some coins (" + strings.Join(parts, ", ") + ")"
}

// ── Container loot generation ─────────────────────────────────────────────────
// Called from generateTreasure (treasure.go) when a chest/coffer drops.
// Populates the room container's contents via roomContainerSet.

// Scroll adjectives used when generating scroll drops.
var scrollAdjectiveWords = []string{
	"battered", "ragged", "decaying", "vellum", "crumpled",
	"ricepaper", "parchment", "decrepit", "tattered", "old",
	"ancient", "dusty", "moldy", "faint",
}

// gemQuality defines quality tiers for gems.
type gemQuality struct {
	adjName    string
	multiplier float64
}

var gemQualities = []gemQuality{
	{"cracked", 0.25},
	{"chipped", 0.60},
	{"", 1.0},
	{"flawless", 2.0},
	{"perfect", 4.0},
}

// gemSize defines size tiers for gems.
type gemSize struct {
	adjName    string
	multiplier float64
}

var gemSizes = []gemSize{
	{"tiny", 0.50},
	{"small", 0.75},
	{"", 1.00},
	{"large", 1.50},
	{"huge", 2.00},
}

// gemQualWeights returns quality selection weights biased by treasure level.
func gemQualWeights(treasureLevel int) []int {
	switch {
	case treasureLevel >= 30:
		return []int{2, 8, 30, 35, 25}
	case treasureLevel >= 15:
		return []int{5, 15, 50, 20, 10}
	default:
		return []int{15, 25, 45, 12, 3}
	}
}

// gemSizeWeights returns size selection weights biased by treasure level.
func gemSizeWeights(treasureLevel int) []int {
	switch {
	case treasureLevel >= 30:
		return []int{2, 8, 45, 30, 15}
	case treasureLevel >= 15:
		return []int{5, 15, 55, 18, 7}
	default:
		return []int{15, 25, 50, 8, 2}
	}
}

// weightedPickGem picks an index using weighted random selection.
func weightedPickGem(weights []int) int {
	total := 0
	for _, w := range weights {
		total += w
	}
	if total == 0 {
		return 0
	}
	r := rand.Intn(total)
	for i, w := range weights {
		r -= w
		if r < 0 {
			return i
		}
	}
	return len(weights) - 1
}

// populateContainerLoot fills a freshly-dropped chest/coffer with loot.
// roomNum and itemRef identify where to store the contents.
// treasureLevel drives quantity and quality.
// Item count follows a 2d4 bell curve (2–8 total), with coins always occupying one slot.
func (e *GameEngine) populateContainerLoot(roomNum, itemRef, treasureLevel int) {
	if treasureLevel <= 0 {
		return
	}

	// 2d4 gives a bell curve of 2–8; one slot is always coins
	numItems := rand.Intn(4) + 1 + rand.Intn(4) + 1

	var contents []InventoryItem

	// ── Always: Coins ──────────────────────────────────────────────────
	coinAmount := treasureLevel*3 + rand.Intn(treasureLevel*2+1)
	if coinAmount > 0 {
		contents = append(contents, InventoryItem{
			Archetype: 0,
			Val1:      coinAmount,
			State:     "MONEY",
		})
	}

	// ── Extra slots: gems, jewelry, or scrolls ──────────────────────────
	extras := numItems - 1
	for i := 0; i < extras; i++ {
		switch rand.Intn(4) {
		case 0:
			if item := e.randomGemDrop(treasureLevel); item != nil {
				contents = append(contents, inventoryFromRoomItem(item))
			}
		case 1:
			if item := e.randomJewelryDrop(treasureLevel); item != nil {
				contents = append(contents, inventoryFromRoomItem(item))
			}
		case 2:
			if item := e.randomScrollDropWithAdj(treasureLevel); item != nil {
				contents = append(contents, inventoryFromRoomItem(item))
			}
		case 3:
			if item := e.randomPotionDrop(treasureLevel); item != nil {
				contents = append(contents, inventoryFromRoomItem(item))
			}
		}
	}

	if len(contents) > 0 {
		e.roomContainerSet(roomNum, itemRef, contents)
	}
}

// inventoryFromRoomItem converts a *gameworld.RoomItem to an InventoryItem.
func inventoryFromRoomItem(ri *gameworld.RoomItem) InventoryItem {
	return InventoryItem{
		Archetype: ri.Archetype,
		Adj1:      ri.Adj1, Adj2: ri.Adj2, Adj3: ri.Adj3,
		Val1: ri.Val1, Val2: ri.Val2, Val3: ri.Val3, Val4: ri.Val4, Val5: ri.Val5,
		Sharpness: ri.Sharpness,
		HardnessMod: ri.HardnessMod,
		ItemBits:  ri.ItemBits,
		State:     ri.State,
		Tail:      ri.Extend,
	}
}

// randomScrollDropWithAdj creates a scroll item with a random appearance adjective.
// This replaces/supplements randomScrollDrop in treasure.go for container loot.
func (e *GameEngine) randomScrollDropWithAdj(treasureLevel int) *gameworld.RoomItem {
	item := e.randomScrollDrop(treasureLevel)
	if item == nil {
		return nil
	}
	// Apply a random appearance adjective (stored in Adj1)
	adjWord := scrollAdjectiveWords[rand.Intn(len(scrollAdjectiveWords))]
	item.Adj1 = e.adjByName(adjWord) // 0 if not in adjective table — graceful fallback
	return item
}

// randomGemDrop selects a random gem item and assigns a quality adjective.
// Gems occupy item numbers 99–123 in the original scripts.
func (e *GameEngine) randomGemDrop(treasureLevel int) *gameworld.RoomItem {
	var candidates []int
	for num := 99; num <= 123; num++ {
		if e.items[num] != nil {
			candidates = append(candidates, num)
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	chosen := candidates[rand.Intn(len(candidates))]

	// Pick quality and size, both biased by treasure level
	q := gemQualities[weightedPickGem(gemQualWeights(treasureLevel))]
	sz := gemSizes[weightedPickGem(gemSizeWeights(treasureLevel))]

	item := &gameworld.RoomItem{Archetype: chosen}

	// Adj order: size first (if any), then quality (if any)
	adjSlot := 0
	setAdj := func(id int) {
		switch adjSlot {
		case 0:
			item.Adj1 = id
		case 1:
			item.Adj2 = id
		case 2:
			item.Adj3 = id
		}
		adjSlot++
	}
	if sz.adjName != "" {
		setAdj(e.adjByName(sz.adjName))
	}
	if q.adjName != "" {
		setAdj(e.adjByName(q.adjName))
	}

	// Combined value multiplier: quality × size
	item.Val2 = int(q.multiplier * sz.multiplier * 100)

	return item
}

// isJewelryBeneficialSpell returns true for spells appropriate for treasure-generated magic jewelry.
// Only healing, defensive, and self-buff spells are allowed — nothing that targets enemies,
// no weapon enchantments, and no crowd-control or debuff spells.
func isJewelryBeneficialSpell(sp SpellDef) bool {
	// All damage spells excluded
	if sp.Effect == "damage" {
		return false
	}
	// Explicitly excluded spell IDs
	excluded := map[int]bool{
		127: true, // Web
		135: true, // Storm Blade (weapon enchant)
		136: true, // Inferno Blade (weapon enchant)
		137: true, // Winter Blade (weapon enchant)
		200: true, // Fear
		201: true, // Charm
		202: true, // Enchantment I (weapon enchant)
		203: true, // Enchantment II (weapon enchant)
		204: true, // Enchantment III (weapon enchant)
		211: true, // Slow
		216: true, // Slumber I
		219: true, // Silence
		400: true, // Detect Magic (passive/object-targeted)
		403: true, // Mindlink (communication)
		406: true, // Dispel Invisibility (targets others)
		407: true, // Analyze Ore (crafting only)
		500: true, // Plant Snare
	}
	return !excluded[sp.ID]
}

// randomJewelryDrop selects a magical jewelry item from the document-defined archetype list
// and assigns a beneficial spell (Val3) with a limited number of charges (Val2).
// Val2 == 0 means the item has no magical charges.
// ROUTINE1 items auto-cast their spell; ROUTINE2 items prep it for CAST.
func (e *GameEngine) randomJewelryDrop(treasureLevel int) *gameworld.RoomItem {
	// ROUTINE1: auto-casts the spell when activated (con/rub/turn triggers)
	routine1 := []int{1059, 149, 169, 150, 170, 154, 174}
	// ROUTINE2: prepares the spell for the player to CAST (con/rub/wave triggers)
	routine2 := []int{188, 183, 176, 187, 180, 184, 177, 185, 178, 186, 179, 155}

	var candidates []int
	for _, arch := range append(routine1, routine2...) {
		// Only the generic catalog is safe for randomized loot — see
		// maxGenericTreasureItemNumber in treasure.go. Item 1059 slips past this
		// (a plain stone ring, harmless), but the guard stays here in case this
		// list is ever extended with a higher-numbered archetype that turns out
		// to be a unique/story item like the ones that motivated this cap.
		if arch >= maxGenericTreasureItemNumber {
			continue
		}
		if e.items[arch] != nil {
			candidates = append(candidates, arch)
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	chosen := candidates[rand.Intn(len(candidates))]
	item := &gameworld.RoomItem{Archetype: chosen}

	// 40% chance of a spell trigger at treasure level >= 1
	if treasureLevel >= 1 && rand.Intn(100) < 40 {
		maxSpellLevel := treasureLevel / 2
		if maxSpellLevel < 1 {
			maxSpellLevel = 1
		}
		if maxSpellLevel > 15 {
			maxSpellLevel = 15
		}
		var spellCandidates []SpellDef
		for _, sp := range spellRegistry {
			if sp.Level <= maxSpellLevel && sp.Effect != "" && isJewelryBeneficialSpell(sp) {
				spellCandidates = append(spellCandidates, sp)
			}
		}
		if len(spellCandidates) > 0 {
			sp := spellCandidates[rand.Intn(len(spellCandidates))]
			item.Val3 = sp.ID
			baseCharges := 10 + rand.Intn(treasureLevel/2+3)
			if baseCharges > 30 {
				baseCharges = 30
			}
			item.Val2 = baseCharges
		}
	}

	return item
}

// ── doInventory (enhanced to show container contents) ────────────────────────
// Call this from the INVENTORY case in ProcessCommand instead of the existing doInventory.

func (e *GameEngine) doInventoryEnhanced(player *Player) *CommandResult {
	var msgs []string
	msgs = append(msgs, "You are carrying:")
	if len(player.Inventory) == 0 && len(player.Worn) == 0 && player.Wielded == nil && player.OffHand == nil {
		msgs = append(msgs, "  Nothing.")
		return &CommandResult{Messages: msgs}
	}

	if player.Wielded != nil {
		itemDef := e.items[player.Wielded.Archetype]
		if itemDef != nil {
			name := e.formatItemName(itemDef, player.Wielded.Adj1, player.Wielded.Adj2, player.Wielded.Adj3, player.Wielded.Tail)
			msgs = append(msgs, fmt.Sprintf("  %s (wielded)", name))
		}
	}
	if player.OffHand != nil {
		itemDef := e.items[player.OffHand.Archetype]
		if itemDef != nil {
			name := e.formatItemName(itemDef, player.OffHand.Adj1, player.OffHand.Adj2, player.OffHand.Adj3, player.OffHand.Tail)
			label := "(off-hand weapon)"
			if itemDef.Type == "SHIELD" {
				label = "(shield)"
			}
			msgs = append(msgs, fmt.Sprintf("  %s %s", name, label))
		}
	}

	for _, ii := range player.Worn {
		itemDef := e.items[ii.Archetype]
		if itemDef != nil {
			name := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3, ii.Tail)
			msgs = append(msgs, fmt.Sprintf("  %s (worn)", name))
		}
	}

	// Collect inventory item names, then join multiple per line
	var itemNames []string
	for _, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		itemNames = append(itemNames, e.formatContainerName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3, ii.State, ii.Tail))
	}

	// Print one item per line
	for _, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		msgs = append(msgs, "  "+e.formatContainerName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3, ii.State, ii.Tail))
	}

	return &CommandResult{Messages: msgs}
}

// ── container lock-state fix on pickup ───────────────────────────────────────
// The reported bug: containers dropped by monsters are always "LOCKED" but become
// unlocked once picked up. The fix is in generateTreasure / randomChestDrop: the
// chest's State should only be "LOCKED" when Val1 (lock difficulty) > 0.
// This helper is called from doGet (engine.go) when picking up a container.
// It preserves the lock state correctly by NOT resetting it on pickup.
//
// The existing doGet code already copies State from RoomItem to InventoryItem, so no
// change is needed there — the bug was that randomChestDrop always sets State="LOCKED"
// regardless of whether a lock was actually assigned. Fix is in treasure.go:
//   Only set State="LOCKED" when lockDiff > 0 (which it always is per current code,
//   so actually the real fix is: containers with no trap and low treasureLevel
//   have a chance of being unlocked already).
//
// See fixChestLockState below — call this from randomChestDrop.

// FixChestInitialState applies a chance that a dropped chest is already unlocked.
// treasureLevel < 10: 40% chance unlocked. 10-19: 20%. 20+: always locked.
func FixChestInitialState(item *gameworld.RoomItem, treasureLevel int) {
	if item.State != "LOCKED" {
		return
	}
	var unlockChance int
	switch {
	case treasureLevel < 10:
		unlockChance = 40
	case treasureLevel < 20:
		unlockChance = 20
	default:
		unlockChance = 0
	}
	if unlockChance > 0 && rand.Intn(100) < unlockChance {
		item.State = "CLOSED"
	}
}
