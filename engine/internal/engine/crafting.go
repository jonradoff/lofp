package engine

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// ---- MINING ----

// regionElementalType returns "heat", "cold", or "electric" for the dominant element.
// mineAdjName is the adjective name for the region's MINE_ADJ (e.g. "fiery", "icy") and
// is used as a fallback when the region has no explicit FireMod/ColdMod/ElectricMod.
func regionElementalType(region *gameworld.Region, mineAdjName string) string {
	if region == nil {
		return ""
	}
	best, elem := 0, ""
	if region.FireMod > best {
		best, elem = region.FireMod, "heat"
	}
	if region.ColdMod > best {
		best, elem = region.ColdMod, "cold"
	}
	if region.ElectricMod > best {
		elem = "electric"
	}
	if elem != "" {
		return elem
	}
	// No explicit elemental mod — infer from the mine adjective name
	adj := strings.ToLower(mineAdjName)
	switch {
	case strings.Contains(adj, "fiery") || strings.Contains(adj, "fire") || strings.Contains(adj, "hot") || strings.Contains(adj, "lava") || strings.Contains(adj, "burn"):
		return "heat"
	case strings.Contains(adj, "icy") || strings.Contains(adj, "ice") || strings.Contains(adj, "cold") || strings.Contains(adj, "frost") || strings.Contains(adj, "snow"):
		return "cold"
	case strings.Contains(adj, "electric") || strings.Contains(adj, "lightning") || strings.Contains(adj, "spark") || strings.Contains(adj, "static"):
		return "electric"
	}
	return ""
}

// oreColorVal returns the hidden "color" of ore (1=purple, 2=indigo, 3=blue) based on
// mine grade. Color determines the maximum sharpness a weapon forged from this metal
// can achieve. All current mines are capped at blue (3) per the original game.
func oreColorVal(grade string) int {
	n := rand.Intn(100)
	switch grade {
	case "A":
		if n < 5 {
			return 1 // purple: rare in grade A
		} else if n < 30 {
			return 2 // indigo
		}
		return 3 // blue: most common in grade A
	case "B":
		if n < 25 {
			return 1 // purple
		} else if n < 75 {
			return 2 // indigo: most common in grade B
		}
		return 3 // blue: uncommon
	default: // C
		if n < 60 {
			return 1 // purple: most common in grade C
		} else if n < 95 {
			return 2 // indigo
		}
		return 3 // blue: very rare in grade C
	}
}

// weaponSharpnessBonus computes the non-magical to-hit bonus (Sharpness) for a weapon
// freshly forged from metal with the given color (1=purple, 2=indigo, 3=blue).
// Formula from the original game: base range from color + randomised smith skill bonus
// → upper max → final roll. Mirrors the documented sharpness system.
func weaponSharpnessBonus(color int, smithSkill int) int {
	if color <= 0 {
		return 0
	}
	// Base sharpness range: purple 1-5, indigo 6-10, blue 11-15
	baseMin := (color-1)*5 + 1
	baseMax := color * 5
	base := baseMin + rand.Intn(baseMax-baseMin+1)
	// Smith skill shifts the ceiling: -5 to +min(smithSkill, 15)
	skillCap := smithSkill
	if skillCap > 15 {
		skillCap = 15
	}
	skillBonus := rand.Intn(skillCap+6) - 5 // range: -5 to +skillCap
	upperMax := base + skillBonus
	if upperMax < 1 {
		return 0
	}
	return rand.Intn(upperMax) + 1 // 1 to upperMax
}

// elementalVal3 maps an element type + ore purity to the weapon crit Val3 constant
// (see weaponCritDamage in combat.go: 2=50%heat, 3=50%cold, 4=40%elec;
// down to 16=10%heat, 17=10%cold, 18=10%elec for poor-quality ores).
func elementalVal3(elemType string, purity int) int {
	offset := 0
	switch elemType {
	case "heat":
		offset = 0
	case "cold":
		offset = 1
	case "electric":
		offset = 2
	default:
		return 0
	}
	switch {
	case purity > 70:
		return 2 + offset // excellent: 50%/50%/40% proc
	case purity > 50:
		return 10 + offset // good: 30% proc
	case purity > 30:
		return 13 + offset // fair: 20% proc
	default:
		return 16 + offset // poor: 10% proc
	}
}

// canonicalizeOilyAdjs rewrites a crafted item's adjective list so that oily/oiled
// metal (adj 221 or 684) becomes the pair Adj1=752 (iridescent), Adj2=684 (oiled) —
// the exact order eddie3.scr's cave rooms check (ITEMADJ1=752, ITEMADJ2=684) to grant
// round-time avoidance. Oily/oiled metals must not carry elemental crit properties
// into crafted items. Any other adjective present is preserved in the remaining slot.
func canonicalizeOilyAdjs(adjs [3]int) [3]int {
	isOily := false
	var rest []int
	for _, a := range adjs {
		if a == 221 || a == 684 {
			isOily = true
			continue
		}
		if a > 0 {
			rest = append(rest, a)
		}
	}
	if !isOily {
		return adjs
	}
	result := [3]int{752, 684, 0}
	if len(rest) > 0 {
		result[2] = rest[0]
	}
	return result
}

// oreXPBase returns base XP for mining a given metal type.
func oreXPBase(metalName string) int {
	switch strings.ToLower(metalName) {
	case "tin", "copper":
		return 25
	case "iron", "bronze", "brass":
		return 50
	case "steel":
		return 75
	case "silver":
		return 100
	case "gold":
		return 200
	case "truesteel":
		return 150
	default:
		return 300 // randar, elkyri, and other exotics
	}
}

// oreQualityXPMult returns the XP multiplier for ore quality based on purity.
func oreQualityXPMult(purity int) float64 {
	switch {
	case purity > 70:
		return 2.0
	case purity > 50:
		return 1.5
	case purity > 30:
		return 1.0
	default:
		return 0.5
	}
}

// gemXPBase returns base XP for mining a gem of the given name.
func gemXPBase(gemName string) int {
	switch strings.ToLower(gemName) {
	case "crystal", "quartz":
		return 50
	case "citrine", "garnet", "amethyst", "topaz", "tourmaline", "aquamarine":
		return 100
	case "pearl", "onyx", "sardonyx":
		return 150
	case "opal":
		return 200
	case "emerald", "sapphire":
		return 250
	case "ruby", "jacinth":
		return 350
	case "diamond":
		return 500
	default:
		return 75
	}
}

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

	// Round time check
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You must wait %.0f more seconds.", remaining)}}
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

	mineRoundTime := applyRoundTime(player, 10)
	player.RoundTimeExpiry = time.Now().Add(time.Duration(mineRoundTime) * time.Second)
	player.RoundTime = mineRoundTime

	if rand.Intn(100) >= chance {
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      []string{"You swing at the rock face but find nothing useful.", fmt.Sprintf("[Round: %d sec]", mineRoundTime)},
			RoomBroadcast: []string{fmt.Sprintf("%s swings a mining tool at the rock.", player.FirstName)},
			PlayerState:   player,
		}
	}

	// Look up the room's region for special ore/gem generation
	var region *gameworld.Region
	if room.Region > 0 {
		region = e.regions[room.Region]
	}

	// Determine if this region produces a special elemental ore adjective (e.g. "icy")
	specialAdjID := 0
	if region != nil && region.MineAdj > 0 && rand.Intn(100) < 50 {
		specialAdjID = region.MineAdj
	}

	// Gem mining: chance increases with grade; regional mines may produce themed gems
	gemChance := 3
	switch grade {
	case "A":
		gemChance = 15
	case "B":
		gemChance = 8
	}
	if rand.Intn(100) < gemChance {
		result := e.doMineGemAttempt(ctx, player, grade, specialAdjID, mineRoundTime)
		if result != nil {
			return result
		}
	}

	// Pick ore type and metal adjective based on grade and region
	type metalChoice struct {
		adj    int
		name   string
		weight int
	}
	var metals []metalChoice

	// Deep Realms detection: explicit Region 2, OR rooms named after the deep areas
	// (many DEEP1/DEEP2 rooms are named "Deep Realms"/"Subterranean" without a REGION tag)
	roomNameLower := strings.ToLower(room.Name)
	isDeepRealms := room.Region == 2 ||
		strings.Contains(roomNameLower, "deep realm") ||
		strings.Contains(roomNameLower, "subterranean")

	// Exotic region: extraplanar, elemental, or has special mine adjective (ice plane, fire plane, etc.)
	isExoticRegion := false
	if !isDeepRealms && region != nil {
		_, isExtraplanar := region.Properties["EXTRAPLANAR"]
		isExoticRegion = region.MineAdj > 0 ||
			isExtraplanar ||
			region.FireMod+region.ColdMod+region.ElectricMod >= 100
	}

	switch {
	case isDeepRealms && grade == "A" && rand.Intn(100) < 30:
		// Grade A deep realms: significant chance of exotic metals
		metals = []metalChoice{
			{e.adjByName("truesteel"), "truesteel", 50},
			{e.adjByName("randar"), "randar", 30},
			{e.adjByName("elkyri"), "elkyri", 20},
		}
	case isDeepRealms && grade == "B" && rand.Intn(100) < 10:
		// Grade B deep realms: small chance of truesteel; randar/elkyri extremely rare
		metals = []metalChoice{
			{e.adjByName("truesteel"), "truesteel", 70},
			{e.adjByName("randar"), "randar", 20},
			{e.adjByName("elkyri"), "elkyri", 10},
		}
	case isExoticRegion && grade == "A" && rand.Intn(100) < 25:
		// Grade A exotic planes (ice, fire, electric): chance of exotic metals
		metals = []metalChoice{
			{e.adjByName("truesteel"), "truesteel", 45},
			{e.adjByName("randar"), "randar", 35},
			{e.adjByName("elkyri"), "elkyri", 20},
		}
	case isExoticRegion && grade == "B" && rand.Intn(100) < 8:
		// Grade B exotic planes: mostly truesteel with rare randar/elkyri
		metals = []metalChoice{
			{e.adjByName("truesteel"), "truesteel", 75},
			{e.adjByName("randar"), "randar", 20},
			{e.adjByName("elkyri"), "elkyri", 5},
		}
	case grade == "A":
		metals = []metalChoice{
			{e.adjByName("iron"), "iron", 35},
			{e.adjByName("steel"), "steel", 17},
			{e.adjByName("bronze"), "bronze", 22},
			{e.adjByName("copper"), "copper", 14},
			{e.adjByName("silver"), "silver", 8},
			{e.adjByName("gold"), "gold", 4},
		}
	case grade == "B":
		metals = []metalChoice{
			{e.adjByName("copper"), "copper", 37},
			{e.adjByName("bronze"), "bronze", 28},
			{e.adjByName("iron"), "iron", 19},
			{e.adjByName("tin"), "tin", 10},
			{e.adjByName("silver"), "silver", 5},
			{e.adjByName("gold"), "gold", 1},
		}
	default: // C
		metals = []metalChoice{
			{e.adjByName("copper"), "copper", 40},
			{e.adjByName("tin"), "tin", 35},
			{e.adjByName("iron"), "iron", 25},
		}
	}

	// Weighted random selection of metal type
	totalWeight := 0
	for _, m := range metals {
		totalWeight += m.weight
	}
	pick := rand.Intn(totalWeight)
	chosenMetal := metals[0]
	for _, m := range metals {
		pick -= m.weight
		if pick < 0 {
			chosenMetal = m
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
		return &CommandResult{Messages: []string{"You chip away at the rock but find nothing."}}
	}

	// Purity based on grade: A=50-100, B=30-70, C=10-40
	// Mining skill adds 1% purity per 2 skill levels (capped at 100).
	skillBonus := miningSkill / 2
	purity := 0
	switch grade {
	case "A":
		purity = 50 + rand.Intn(51)
	case "B":
		purity = 30 + rand.Intn(41)
	case "C":
		purity = 10 + rand.Intn(31)
	}
	purity += skillBonus
	if purity > 100 {
		purity = 100
	}

	ore := InventoryItem{
		Archetype: oreArch,
		Val1:      purity,          // purity: chance of successful smelt
		Val2:      oreColorVal(grade), // color: determines max weapon sharpness
	}
	if specialAdjID != 0 {
		// Elemental ore: special adj in Adj1, metal type in Adj2 ("icy iron ore")
		ore.Adj1 = specialAdjID
		ore.Adj2 = chosenMetal.adj
		// Set Val3 so the elemental property propagates through smelt → forge into the weapon
		ore.Val3 = elementalVal3(regionElementalType(region, e.adjectives[specialAdjID]), purity)
	} else {
		ore.Adj1 = chosenMetal.adj
	}

	// Award XP based on metal tier and purity quality
	xpBase := oreXPBase(chosenMetal.name)
	if specialAdjID != 0 {
		xpBase = 800 // Elemental ores are especially valuable
	}
	xp := int(float64(xpBase) * oreQualityXPMult(purity))
	player.Experience += xp

	player.Inventory = append(player.Inventory, ore)
	e.SavePlayer(ctx, player)

	qualityDesc := "poor"
	if purity > 70 {
		qualityDesc = "excellent"
	} else if purity > 50 {
		qualityDesc = "good"
	} else if purity > 30 {
		qualityDesc = "fair"
	}

	displayName := e.formatItemNameNoArticle(oreDef, ore.Adj1, ore.Adj2, ore.Adj3)
	return &CommandResult{
		Messages: []string{
			fmt.Sprintf("You chip away at the rock and extract some %s looking %s!", qualityDesc, displayName),
			fmt.Sprintf("[Round: %d sec]", mineRoundTime),
			fmt.Sprintf("You have been awarded %d experience points.", xp),
		},
		RoomBroadcast: []string{fmt.Sprintf("%s mines some ore from the rock.", player.FirstName)},
		PlayerState:   player,
	}
}

// doMineGemAttempt tries to mine a gem. Grade affects quality and size distribution.
// Returns nil if no gem candidates exist (caller falls back to ore).
func (e *GameEngine) doMineGemAttempt(ctx context.Context, player *Player, grade string, specialAdjID int, roundTime int) *CommandResult {
	var candidates []int
	for num := 99; num <= 122; num++ {
		if e.items[num] != nil {
			candidates = append(candidates, num)
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	chosen := candidates[rand.Intn(len(candidates))]
	def := e.items[chosen]

	// Quality: grade A skews toward flawless/perfect, grade C toward cracked/chipped
	type gemQualEntry struct {
		adjName string
		valMult float64
		xpMult  float64
		weight  int
	}
	var qualEntries []gemQualEntry
	switch grade {
	case "A":
		qualEntries = []gemQualEntry{
			{"cracked", 0.25, 0.50, 5},
			{"chipped", 0.60, 0.75, 10},
			{"", 1.00, 1.00, 35},
			{"flawless", 2.00, 1.50, 30},
			{"perfect", 4.00, 2.00, 20},
		}
	case "B":
		qualEntries = []gemQualEntry{
			{"cracked", 0.25, 0.50, 10},
			{"chipped", 0.60, 0.75, 20},
			{"", 1.00, 1.00, 40},
			{"flawless", 2.00, 1.50, 20},
			{"perfect", 4.00, 2.00, 10},
		}
	default: // C
		qualEntries = []gemQualEntry{
			{"cracked", 0.25, 0.50, 25},
			{"chipped", 0.60, 0.75, 30},
			{"", 1.00, 1.00, 35},
			{"flawless", 2.00, 1.50, 8},
			{"perfect", 4.00, 2.00, 2},
		}
	}
	qualWeights := make([]int, len(qualEntries))
	for i, q := range qualEntries {
		qualWeights[i] = q.weight
	}
	chosenQ := qualEntries[weightedPickMine(qualWeights)]

	// Size: grade A yields larger gems, grade C yields smaller ones
	type gemSizeEntry struct {
		adjName string
		valMult float64
		xpMult  float64
		weight  int
	}
	var sizeEntries []gemSizeEntry
	switch grade {
	case "A":
		sizeEntries = []gemSizeEntry{
			{"tiny", 0.50, 0.50, 5},
			{"small", 0.75, 0.75, 15},
			{"", 1.00, 1.00, 50},
			{"large", 1.50, 1.50, 20},
			{"huge", 2.00, 2.00, 10},
		}
	case "B":
		sizeEntries = []gemSizeEntry{
			{"tiny", 0.50, 0.50, 10},
			{"small", 0.75, 0.75, 20},
			{"", 1.00, 1.00, 50},
			{"large", 1.50, 1.50, 15},
			{"huge", 2.00, 2.00, 5},
		}
	default: // C
		sizeEntries = []gemSizeEntry{
			{"tiny", 0.50, 0.50, 25},
			{"small", 0.75, 0.75, 30},
			{"", 1.00, 1.00, 38},
			{"large", 1.50, 1.50, 5},
			{"huge", 2.00, 2.00, 2},
		}
	}
	sizeWeights := make([]int, len(sizeEntries))
	for i, s := range sizeEntries {
		sizeWeights[i] = s.weight
	}
	chosenSize := sizeEntries[weightedPickMine(sizeWeights)]

	// Combined value multiplier in Val2
	gem := InventoryItem{
		Archetype: chosen,
		Val2:      int(chosenQ.valMult * chosenSize.valMult * 100),
	}

	// Adj order: regional → size → quality (display: "icy huge perfect ruby")
	adjSlot := 0
	setGemAdj := func(id int) {
		switch adjSlot {
		case 0:
			gem.Adj1 = id
		case 1:
			gem.Adj2 = id
		case 2:
			gem.Adj3 = id
		}
		adjSlot++
	}
	if specialAdjID != 0 {
		setGemAdj(specialAdjID)
	}
	if chosenSize.adjName != "" {
		setGemAdj(e.adjByName(chosenSize.adjName))
	}
	if chosenQ.adjName != "" {
		setGemAdj(e.adjByName(chosenQ.adjName))
	}

	gemName := strings.ToLower(e.nouns[def.NameID])
	xp := int(float64(gemXPBase(gemName)) * chosenQ.xpMult * chosenSize.xpMult)
	if specialAdjID != 0 {
		xp = xp * 2 // Bonus XP for rare elemental gems
	}

	player.Inventory = append(player.Inventory, gem)
	player.Experience += xp
	e.SavePlayer(ctx, player)

	gemDisplay := e.formatItemName(def, gem.Adj1, gem.Adj2, gem.Adj3, gem.Tail)
	return &CommandResult{
		Messages: []string{
			fmt.Sprintf("You chip at the rock and uncover %s!", gemDisplay),
			fmt.Sprintf("[Round: %d sec]", roundTime),
			fmt.Sprintf("You have been awarded %d experience points.", xp),
		},
		RoomBroadcast: []string{fmt.Sprintf("%s discovers a gem while mining!", player.FirstName)},
		PlayerState:   player,
	}
}

// weightedPickMine returns the index of the selected weight in a slice.
func weightedPickMine(weights []int) int {
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


// ---- SMELTING ----

// smeltXPAward returns XP for successfully smelting ore of the given metal adjective name.
func smeltXPAward(metalName string) int {
	switch strings.ToLower(metalName) {
	case "tin", "copper":
		return 5
	case "bronze":
		return 10
	case "iron", "brass":
		return 15
	case "steel":
		return 25
	case "silver":
		return 30
	case "gold":
		return 40
	case "truesteel":
		return 50
	default: // exotic: randar, elkyri, etc.
		return 100
	}
}

func (e *GameEngine) doSmelt(ctx context.Context, player *Player, args []string) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil || !containsModifier(room.Modifiers, "FORGE") {
		return &CommandResult{Messages: []string{"You need to be at a forge to smelt ore."}}
	}

	smithSkill := player.Skills[8]
	if smithSkill < 1 {
		return &CommandResult{Messages: []string{"You have no training in Weaponsmithing."}}
	}

	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You must wait %.0f more seconds.", remaining)}}
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

		// Purity check: VAL1 = percentage chance of successful refinement
		purity := ii.Val1
		if purity <= 0 {
			purity = 30
		}

		// Remove ore
		player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)

		smeltRT := applyRoundTime(player, 5)
		player.RoundTimeExpiry = time.Now().Add(time.Duration(smeltRT) * time.Second)
		player.RoundTime = smeltRT

		// Roll against purity + skill bonus
		smeltChance := purity + smithSkill*2
		if smeltChance > 95 {
			smeltChance = 95
		}

		if rand.Intn(100) >= smeltChance {
			e.SavePlayer(ctx, player)
			return &CommandResult{
				Messages:      []string{"You heat the ore in the forge, but it crumbles to useless slag.", fmt.Sprintf("[Round: %d sec]", smeltRT)},
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
			return &CommandResult{Messages: []string{"The ore refines but produces nothing useful.", fmt.Sprintf("[Round: %d sec]", smeltRT)}}
		}

		material := InventoryItem{
			Archetype: outputArch,
			Adj1:      ii.Adj1, // preserve leading adjective (metal type or elemental)
			Adj2:      ii.Adj2, // preserve secondary adj (metal type when Adj1 is elemental)
			Val2:      ii.Val2, // transfer material properties
			Val3:      ii.Val3, // propagate elemental combat type into forged weapon
		}
		player.Inventory = append(player.Inventory, material)

		// Determine metal name for XP: for elemental ores Adj1 is the elemental adj, Adj2 is the metal
		metalAdjID := ii.Adj1
		if ii.Adj2 != 0 {
			metalAdjID = ii.Adj2
		}
		metalName := strings.ToLower(e.adjectives[metalAdjID])
		xpAward := smeltXPAward(metalName)
		player.Experience += xpAward

		e.SavePlayer(ctx, player)

		matName := e.formatItemName(outputDef, material.Adj1, material.Adj2, material.Adj3, material.Tail)
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("You smelt the ore in the forge and produce some %s!", matName),
				fmt.Sprintf("[Round: %d sec]", smeltRT),
				fmt.Sprintf("You have been awarded %d experience points.", xpAward),
			},
			RoomBroadcast: []string{fmt.Sprintf("%s works at the forge, smelting ore.", player.FirstName)},
			PlayerState:   player,
		}
	}

	return &CommandResult{Messages: []string{"You don't have any ore to smelt."}}
}

// ---- CRAFTING (FORGE/CRAFT) ----

// metalDifficulty returns the quench success rate for a metal adjective name.
func metalDifficulty(metal string) int {
	switch strings.ToLower(metal) {
	case "copper":
		return 70
	case "iron", "brass", "bronze":
		return 55
	case "steel":
		return 45
	case "truesteel":
		return 35
	default:
		// exotic metals: randar, elkyri, etc.
		return 25
	}
}

// metalQualityXPBonus returns the XP bonus for metal quality when completing a weapon.
func metalQualityXPBonus(metal string) int {
	switch strings.ToLower(metal) {
	case "copper":
		return 0
	case "iron", "brass", "bronze":
		return 25
	case "steel":
		return 75
	case "truesteel":
		return 150
	default:
		return 300
	}
}

func (e *GameEngine) doCraft(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		type craftEntry struct {
			name  string
			level int
		}
		grouped := map[string][]craftEntry{}
		for _, def := range e.items {
			if !containsFlag(def.Flags, "CRAFTABLE") {
				continue
			}
			name := e.nouns[def.NameID]
			if name == "" {
				continue
			}
			var skillID int
			var skillName string
			var skillNeeded int
			// PARAMETER1 = Weaponsmithing level; PARAMETER2 = Jeweler/Weaving level.
			// Substance/type order: cloth and wood first, then weapon/armor, then
			// use PARAMETER2 > 0 to identify Jeweler items regardless of substance
			// (STONE, HARDMETAL, SOFTMETAL, etc. all use PARAMETER2 for Jeweler).
			if def.Substance == "CLOTH" {
				skillID = 15
				skillName = "Dyeing/Weaving"
				skillNeeded = def.Parameter2
			} else if def.Substance == "WOOD" {
				skillID = 18
				skillName = "Wood Lore"
				skillNeeded = def.Parameter1
			} else if isWeapon(def.Type) {
				skillID = 8
				skillName = "Weaponsmithing"
				skillNeeded = def.Parameter1
			} else if def.Type == "ARMOR" {
				skillID = 8
				skillName = "Weaponsmithing"
				skillNeeded = def.Weight / 3
			} else if def.Parameter2 > 0 {
				skillID = 0
				skillName = "Jeweler"
				skillNeeded = def.Parameter2
			} else {
				skillID = 8
				skillName = "Weaponsmithing"
				skillNeeded = def.Parameter1
			}
			playerLevel, hasSkill := player.Skills[skillID]
			if hasSkill && playerLevel >= skillNeeded {
				grouped[skillName] = append(grouped[skillName], craftEntry{name: name, level: skillNeeded})
			}
		}
		if len(grouped) == 0 {
			return &CommandResult{Messages: []string{"You don't have the crafting skills to make anything yet."}}
		}
		skillOrder := []string{"Weaponsmithing", "Wood Lore", "Dyeing/Weaving", "Jeweler"}
		msgs := []string{"Items you can craft:"}
		for _, skillName := range skillOrder {
			entries, ok := grouped[skillName]
			if !ok {
				continue
			}
			sort.Slice(entries, func(i, j int) bool {
				if entries[i].level != entries[j].level {
					return entries[i].level < entries[j].level
				}
				return entries[i].name < entries[j].name
			})
			msgs = append(msgs, fmt.Sprintf("\n%s:", skillName))
			for _, entry := range entries {
				if (entry.level > 0) {
					msgs = append(msgs, fmt.Sprintf("  %-30s (requires level %d)", entry.name, entry.level))
				}
			}
		}
		msgs = append(msgs, "\nUse CRAFT <item> to begin crafting.")
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

	// Round time check
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You must wait %.0f more seconds.", remaining)}}
	}

	fullInput := strings.ToLower(strings.Join(args, " "))
	target, materialTarget := parseWithClause(fullInput)
	var cleanMaterialTarget string
	materialSkip := 0
	if materialTarget != "" {
		cleanMaterialTarget, materialSkip = parseOrdinal(materialTarget)
	}

	// Build a sorted item list so CRAFTABLE matching is deterministic regardless of map order.
	// When multiple items share a noun (e.g. two "medal" archetypes at different levels),
	// sorted order lets us reliably prefer the lowest-level one the player can actually craft.
	type numDef struct {
		num int
		def *gameworld.ItemDef
	}
	sortedItems := make([]numDef, 0, len(e.items))
	for n, d := range e.items {
		sortedItems = append(sortedItems, numDef{n, d})
	}
	sort.Slice(sortedItems, func(i, j int) bool { return sortedItems[i].num < sortedItems[j].num })

	// Saved errors from matched items the player can't craft yet.
	var workshopErrMsg string
	var skillErrMsg string
	var skillErrNeeded int // track lowest skillNeeded among skill-blocked items

	// Find a CRAFTABLE item matching the target
	for _, nd := range sortedItems {
		def := nd.def
		if !containsFlag(def.Flags, "CRAFTABLE") {
			continue
		}
		name := strings.ToLower(e.nouns[def.NameID])
		if !strings.HasPrefix(name, target) {
			continue
		}

		// PARAMETER1 = Weaponsmithing level; PARAMETER2 = Jeweler/Weaving level.
		skillNeeded := 0
		skillID := 0
		skillName := ""

		if def.Substance == "CLOTH" {
			if !isLoom && !isForge {
				if workshopErrMsg == "" {
					workshopErrMsg = "You need a loom or forge to craft that."
				}
				continue
			}
			skillID = 15
			skillName = "Dyeing/Weaving"
			skillNeeded = def.Parameter2
		} else if def.Substance == "WOOD" {
			if !isFletcher {
				if workshopErrMsg == "" {
					workshopErrMsg = "You need a fletcher's workshop to craft that."
				}
				continue
			}
			skillID = 18
			skillName = "Wood Lore"
			skillNeeded = def.Parameter1
		} else if isWeapon(def.Type) {
			if !isForge {
				if workshopErrMsg == "" {
					workshopErrMsg = "You need a forge to craft that."
				}
				continue
			}
			skillID = 8
			skillName = "Weaponsmithing"
			skillNeeded = def.Parameter1
		} else if def.Type == "ARMOR" {
			if !isForge {
				if workshopErrMsg == "" {
					workshopErrMsg = "You need a forge to craft that."
				}
				continue
			}
			skillID = 8
			skillName = "Weaponsmithing"
			skillNeeded = def.Weight / 3
		} else if def.Parameter2 > 0 {
			if !isLoom && !isForge {
				if workshopErrMsg == "" {
					workshopErrMsg = "You need a loom or forge to craft that."
				}
				continue
			}
			skillID = 0
			skillName = "Jeweler"
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
			// Record the skill error for the lowest-requirement match (closest to craftable).
			if skillErrMsg == "" || skillNeeded < skillErrNeeded {
				skillErrMsg = fmt.Sprintf("Your %s skill (%d) is not high enough to craft that. You need at least %d.", skillName, playerSkill, skillNeeded)
				skillErrNeeded = skillNeeded
			}
			continue
		}

		// For weapons at the forge, enter the CRAFT→WORK cycle instead of instant creation
		if isForge && isWeapon(def.Type) {
			player.CraftingItem = name
			player.CraftingStep = 1
			player.CraftingSkill = "weaponsmithing"
			player.CraftingMetal = "" // will be set by WORK <metal>
			return &CommandResult{
				Messages: []string{
					fmt.Sprintf("You begin to plan the crafting of your %s...", name),
					"[Next, work your item from a substance, e.g., \"WORK IRON.\"]",
				},
				RoomBroadcast: []string{fmt.Sprintf("%s studies a forge, planning something.", player.FirstName)},
			}
		}

		// Jewelry: enter the CRAFT→WORK cycle
		if skillID == 0 {
			player.CraftingItem = name
			player.CraftingStep = 1
			player.CraftingSkill = "jewelry"
			player.CraftingMetal = ""
			return &CommandResult{
				Messages: []string{
					fmt.Sprintf("You begin planning the crafting of your %s, selecting the right techniques.", name),
					"[Next, WORK <material> to shape it into your design, e.g., \"WORK GOLD.\"]",
				},
				RoomBroadcast: []string{fmt.Sprintf("%s studies their tools, planning something.", player.FirstName)},
			}
		}

		// Dyeing/Weaving: enter the CRAFT→WORK cycle
		if skillID == 15 {
			player.CraftingItem = name
			player.CraftingStep = 1
			player.CraftingSkill = "weaving"
			player.CraftingMetal = ""
			return &CommandResult{
				Messages: []string{
					fmt.Sprintf("You set up your workspace to begin crafting a %s.", name),
					"[Next, WORK <material> to begin weaving, e.g., \"WORK HIDE\" or \"WORK CLOTH.\"]",
				},
				RoomBroadcast: []string{fmt.Sprintf("%s prepares their workspace at the loom.", player.FirstName)},
			}
		}

		// Wood Lore: enter the CRAFT→WORK cycle
		if skillID == 18 {
			player.CraftingItem = name
			player.CraftingStep = 1
			player.CraftingSkill = "wood"
			player.CraftingMetal = ""
			return &CommandResult{
				Messages: []string{
					fmt.Sprintf("You examine your materials, planning how to craft a %s.", name),
					"[Next, WORK <material> to begin shaping, e.g., \"WORK BRANCH\" or \"WORK WOOD.\"]",
				},
				RoomBroadcast: []string{fmt.Sprintf("%s examines their materials, planning something.", player.FirstName)},
			}
		}

		// Non-weapon crafting: immediate creation (armor and misc fallback)
		// Check for material in inventory.
		// MATERIAL type (metals) is stored via item.Type; MATERIAL2 (cloth/skin) is stored as a flag.
		materialFound := false
		materialIdx := -1
		var matItem InventoryItem
		var matDef *gameworld.ItemDef
		skipRemaining := materialSkip
		for j, ii := range player.Inventory {
			mDef := e.items[ii.Archetype]
			if mDef == nil {
				continue
			}
			// Jeweler (skillID 0) uses the same metal pool as Weaponsmithing (skillID 8).
			matSkillID := skillID
			if matSkillID == 0 {
				matSkillID = 8
			}
			// Metal materials load as Type=="MISC" at runtime (same situation handled in doWork).
			// Accept MATERIAL/MATERIAL2 items with matching or wildcard param2, and MISC items
			// only when param2 matches the exact skill (not wildcard 0, which is too broad).
			isValidMaterial := false
			if mDef.Type == "MATERIAL" || containsFlag(mDef.Flags, "MATERIAL2") {
				isValidMaterial = mDef.Parameter2 == matSkillID || mDef.Parameter2 == 0
			} else if mDef.Type == "MISC" {
				isValidMaterial = mDef.Parameter2 == matSkillID
			}
			if !isValidMaterial {
				continue
			}
			// WITH clause: filter by material name/adjective if specified.
			if cleanMaterialTarget != "" {
				matNoun := strings.ToLower(e.getItemNounName(mDef))
				if !matchesTarget(matNoun, cleanMaterialTarget, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) &&
					!strings.HasPrefix(strings.ToLower(e.getAdjName(ii.Adj1)), cleanMaterialTarget) &&
					!strings.HasPrefix(strings.ToLower(e.getAdjName(ii.Adj2)), cleanMaterialTarget) &&
					!strings.HasPrefix(strings.ToLower(e.getAdjName(ii.Adj3)), cleanMaterialTarget) {
					continue
				}
			}
			// Ordinal skipping: "3 doeskin" skips the first two matches.
			if skipRemaining > 0 {
				skipRemaining--
				continue
			}
			materialFound = true
			materialIdx = j
			matItem = ii
			matDef = mDef
			break
		}

		if !materialFound {
			return &CommandResult{Messages: []string{"You don't have the right materials. You need refined material (smelt ore, or forage wood/cloth)."}}
		}

		// Build the crafted item's adjective list:
		//   1. Carry over all instance adjectives from the material (e.g., worg, green).
		//   2. Append the material-type adjective from Parameter1 (e.g., fur=1206, hide=410).
		// This produces names like "worg fur bag" or "green fur satchel".
		var craftAdjs []int
		for _, a := range []int{matItem.Adj1, matItem.Adj2, matItem.Adj3} {
			if a > 0 {
				craftAdjs = append(craftAdjs, a)
			}
		}
		if matDef.Parameter1 > 0 {
			alreadyPresent := false
			for _, a := range craftAdjs {
				if a == matDef.Parameter1 {
					alreadyPresent = true
					break
				}
			}
			if !alreadyPresent {
				craftAdjs = append(craftAdjs, matDef.Parameter1)
			}
		}
		craftAdj := func(i int) int {
			if i < len(craftAdjs) {
				return craftAdjs[i]
			}
			return 0
		}

		// Consume material
		player.Inventory = append(player.Inventory[:materialIdx], player.Inventory[materialIdx+1:]...)

		// Create the item with the full material name as adjectives.
		// Oily/oiled metals become iridescent (752) + oiled (684) in the finished piece.
		finishedAdjs := canonicalizeOilyAdjs([3]int{craftAdj(0), craftAdj(1), craftAdj(2)})

		// Val1 = copper value per GMSCRIPT.DOC; computeSellValue() pays out half of it on
		// sale. Only metal (Weaponsmithing, skillID 8) items reach this path — scale with
		// the skill level required, capped at what the raw material actually cost
		// (matItem.Val1, stamped at purchase in doBuy) so resale can never exceed half the
		// material's cost. See the identical cap in doWork's weapon-finishing step.
		itemValue := 0
		if skillID == 8 {
			itemValue = skillNeeded * 30
			if matItem.Val1 > 0 && itemValue > matItem.Val1 {
				itemValue = matItem.Val1
			}
			if itemValue < 1 {
				itemValue = 1
			}
		}
		item := InventoryItem{
			Archetype: def.Number,
			Adj1:      finishedAdjs[0],
			Adj2:      finishedAdjs[1],
			Adj3:      finishedAdjs[2],
			Val1:      itemValue,
		}
		player.Inventory = append(player.Inventory, item)

		// XP award: scale by skill level required (weaponsmithing uses metalDifficulty instead).
		// Jewelry/weaving items always carry this on Parameter2 (Parameter1 unused there).
		// Wood Lore items are inconsistent in the source data — many non-launcher items
		// (instruments, staves, etc.) never got a Parameter2 assigned, so fall back to
		// Parameter1 (their only other small, per-item difficulty-shaped field) rather
		// than silently awarding zero.
		xpAward := 0
		if skillID != 8 {
			if def.Parameter2 > 0 {
				xpAward = def.Parameter2 * 20
			} else if def.Parameter1 > 0 {
				xpAward = def.Parameter1 * 20
			}
		}
		if xpAward > 0 {
			player.Experience += xpAward
		}
		craftRT := applyRoundTime(player, 15)
		player.RoundTimeExpiry = time.Now().Add(time.Duration(craftRT) * time.Second)
		player.RoundTime = craftRT
		e.SavePlayer(ctx, player)

		itemName := e.formatItemName(def, item.Adj1, item.Adj2, item.Adj3, item.Tail)
		msgs := []string{fmt.Sprintf("You carefully craft %s!", itemName), fmt.Sprintf("[Round: %d sec]", craftRT)}
		if xpAward > 0 {
			msgs = append(msgs, fmt.Sprintf("You have been awarded %d experience points.", xpAward))
		}
		return &CommandResult{
			Messages:      msgs,
			RoomBroadcast: []string{fmt.Sprintf("%s works diligently at the workshop.", player.FirstName)},
			PlayerState:   player,
		}
	}

	// No craftable item was found or created — report the most specific error.
	if skillErrMsg != "" {
		return &CommandResult{Messages: []string{skillErrMsg}}
	}
	if workshopErrMsg != "" {
		return &CommandResult{Messages: []string{workshopErrMsg}}
	}
	return &CommandResult{Messages: []string{fmt.Sprintf("You don't know how to craft '%s'.", target)}}
}

// ---- WORK (Forging Cycle) ----

func (e *GameEngine) doWork(ctx context.Context, player *Player, args []string) *CommandResult {
	// Item scripts must fire before crafting logic — IFPREVERB WORK on an item (e.g., pelt →
	// treated pelt) should preempt the forge workflow.
	if len(args) > 0 {
		target := strings.ToLower(strings.Join(args, " "))
		target, _ = parseOrdinal(target)
		room := e.rooms[player.RoomNumber]

		// Check room items first
		if room != nil {
			for i, ri := range room.Items {
				itemDef := e.items[ri.Archetype]
				if itemDef == nil {
					continue
				}
				if matchesTarget(e.getItemNounName(itemDef), target, e.getAdjName(ri.Adj1), e.getAdjName(ri.Adj2), e.getAdjName(ri.Adj3)) {
					sc := e.RunPreverbScripts(player, room, "WORK", &room.Items[i], itemDef)
					// PLREVENT/CONTPLREVENT-deferred actions must be scheduled, or
					// everything after the delay is lost.
					if len(sc.DeferredSegments) > 0 {
						e.scheduleScriptSegments(player, sc.DeferredSegments)
					}
					if sc.Blocked || len(sc.Messages) > 0 || len(sc.RoomMsgs) > 0 {
						e.SavePlayer(ctx, player)
						result := &CommandResult{Messages: sc.Messages, RoomBroadcast: sc.RoomMsgs, GMBroadcast: sc.GMMsgs, PlayerState: player}
						if len(result.Messages) == 0 {
							result.Messages = []string{"You can't do that."}
						}
						return result
					}
				}
			}
		}

		// Check player items (inventory, worn, wielded)
		allItems := make([]InventoryItem, 0, len(player.Inventory)+len(player.Worn)+1)
		allItems = append(allItems, player.Inventory...)
		allItems = append(allItems, player.Worn...)
		if player.Wielded != nil {
			allItems = append(allItems, *player.Wielded)
		}
		for _, ii := range allItems {
			itemDef := e.items[ii.Archetype]
			if itemDef == nil {
				continue
			}
			if matchesTarget(e.getItemNounName(itemDef), target, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
				tempRI := gameworld.RoomItem{
					Ref: -1, Archetype: ii.Archetype,
					Adj1: ii.Adj1, Adj2: ii.Adj2, Adj3: ii.Adj3,
					Val1: ii.Val1, Val2: ii.Val2, Val3: ii.Val3, Val4: ii.Val4, Val5: ii.Val5,
					ItemBits: ii.ItemBits,
				}
				sc := e.RunPreverbScripts(player, room, "WORK", &tempRI, itemDef)
				// PLREVENT/CONTPLREVENT-deferred actions must be scheduled, or
				// everything after the delay is lost.
				if len(sc.DeferredSegments) > 0 {
					e.scheduleScriptSegments(player, sc.DeferredSegments)
				}
				if sc.Blocked || len(sc.Messages) > 0 || len(sc.RoomMsgs) > 0 {
					e.SavePlayer(ctx, player)
					result := &CommandResult{Messages: sc.Messages, RoomBroadcast: sc.RoomMsgs, GMBroadcast: sc.GMMsgs, PlayerState: player}
					if len(result.Messages) == 0 {
						result.Messages = []string{"You can't do that."}
					}
					return result
				}
			}
		}
	}

	if player.CraftingStep <= 0 {
		return &CommandResult{Messages: []string{"You aren't crafting anything. Use CRAFT <item> first."}}
	}

	switch player.CraftingSkill {
	case "jewelry":
		return e.doWorkJewelry(ctx, player, args)
	case "weaving":
		return e.doWorkWeaving(ctx, player, args)
	case "wood":
		return e.doWorkWood(ctx, player, args)
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
		if metal == "metal" {
			return &CommandResult{Messages: []string{"Which metal? e.g., WORK IRON, WORK STEEL, WORK COPPER"}}
		}

		// Strip trailing " metal" so "work copper metal" == "work copper"
		metalSearch := strings.TrimSuffix(metal, " metal")

		// Find matching material in inventory — check all adj slots (Adj1/Adj2/Adj3).
		// Purchased metals arrive with the metal type adj in Adj3 (doBuy stores si.Adj
		// there); mined/smelted metals have the metal adj in Adj1; elemental metals have
		// an elemental prefix in Adj1 and the metal type in Adj2.
		materialIdx := -1

		for j, ii := range player.Inventory {
			mDef := e.items[ii.Archetype]
			if mDef == nil {
				continue
			}
			isMaterial := mDef.Type == "MATERIAL" || containsFlag(mDef.Flags, "MATERIAL2") || mDef.Type == "MISC"
			if !isMaterial || (mDef.Parameter2 != 8 && mDef.Parameter2 != 0) {
				continue
			}

			// Check every adj slot for the requested metal name
			adjIDs := [3]int{ii.Adj1, ii.Adj2, ii.Adj3}
			metalAdjSlot := -1
			for k, adjID := range adjIDs {
				if adjID > 0 && strings.ToLower(e.getAdjName(adjID)) == metalSearch {
					metalAdjSlot = k
					break
				}
			}
			// Fallback: MATERIAL items may encode the metal type in Parameter1 with no
			// instance adj set (e.g., purchased steel arch 1372 has PARAMETER1=310 "steel"
			// but Adj1/Adj2/Adj3 are all 0 after doBuy).
			if metalAdjSlot < 0 && mDef.Type == "MATERIAL" && mDef.Parameter1 > 0 &&
				strings.ToLower(e.getAdjName(mDef.Parameter1)) == metalSearch {
				metalAdjSlot = 3 // sentinel: matched via Parameter1
			}
			if metalAdjSlot < 0 {
				continue
			}

			materialIdx = j
			// Normalize adj layout so the crafted weapon shows the right name.
			// Purchased metals have the metal adj in slot 2 (Adj3); move it to
			// slot 0 (Adj1) for weapon naming. Parameter1-matched items get their
			// built-in metal adj in slot 0. All other layouts are preserved.
			if metalAdjSlot == 3 {
				player.CraftingAdj1 = mDef.Parameter1
				player.CraftingAdj2 = 0
				player.CraftingAdj3 = 0
			} else if metalAdjSlot == 2 {
				player.CraftingAdj1 = ii.Adj3
				player.CraftingAdj2 = 0
				player.CraftingAdj3 = 0
			} else {
				player.CraftingAdj1 = ii.Adj1
				player.CraftingAdj2 = ii.Adj2
				player.CraftingAdj3 = ii.Adj3
			}
			player.CraftingVal1 = ii.Val1
			player.CraftingVal2 = ii.Val2
			player.CraftingVal3 = ii.Val3
			player.CraftingVal4 = ii.Val4
			player.CraftingVal5 = ii.Val5
			break
		}

		if materialIdx < 0 {
			return &CommandResult{Messages: []string{fmt.Sprintf("You don't have any %s metal to work with.", metalSearch)}}
		}

		// Consume material
		player.Inventory = append(player.Inventory[:materialIdx], player.Inventory[materialIdx+1:]...)

		player.CraftingMetal = metalSearch
		player.CraftingStep = 2
		smithRT := applyRoundTime(player, 15)
		player.RoundTimeExpiry = time.Now().Add(time.Duration(smithRT) * time.Second)
		player.RoundTime = smithRT
		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You place some %s metal into a mold in the forge and heat it until it is roughly the shape you desire.", metalSearch)},
			RoomBroadcast: []string{fmt.Sprintf("%s works diligently at the forge.", player.FirstName)},
			PlayerState:   player,
		}
	case 2: // Heated → Hammer
		player.CraftingStep = 3
		smithRT2 := applyRoundTime(player, 15)
		player.RoundTimeExpiry = time.Now().Add(time.Duration(smithRT2) * time.Second)
		player.RoundTime = smithRT2
		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You remove the %s metal from the forge and begin to hammer it into shape on the anvil.", player.CraftingMetal)},
			RoomBroadcast: []string{fmt.Sprintf("%s works diligently at the forge.", player.FirstName)},
			PlayerState:   player,
		}

	case 3: // Hammered → Quench (skill check)
		smithSkill := player.Skills[8]
		baseChance := metalDifficulty(player.CraftingMetal)
		// Add skill bonus: +3% per skill level
		chance := baseChance + smithSkill*3
		if chance > 95 {
			chance = 95
		}

		roll := rand.Intn(100) + 1
		smithRT3 := applyRoundTime(player, 15)
		player.RoundTimeExpiry = time.Now().Add(time.Duration(smithRT3) * time.Second)
		player.RoundTime = smithRT3

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
		smithRT4 := applyRoundTime(player, 15)
		player.RoundTimeExpiry = time.Now().Add(time.Duration(smithRT4) * time.Second)
		player.RoundTime = smithRT4
		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages:      []string{"You buff the metal, smoothing and polishing the surface. Your weapon is nearly complete!"},
			RoomBroadcast: []string{fmt.Sprintf("%s works diligently at the forge.", player.FirstName)},
			PlayerState:   player,
		}

	case 5: // Buffed → Sharpen (complete!)
		player.CraftingStep = 0
		smithRT5 := applyRoundTime(player, 15)
		player.RoundTimeExpiry = time.Now().Add(time.Duration(smithRT5) * time.Second)
		player.RoundTime = smithRT5

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
			player.CraftingSkill = ""
			player.CraftingAdj1, player.CraftingAdj2, player.CraftingAdj3 = 0, 0, 0
			player.CraftingVal1, player.CraftingVal2, player.CraftingVal3, player.CraftingVal4, player.CraftingVal5 = 0, 0, 0, 0, 0
			e.SavePlayer(ctx, player)
			return &CommandResult{Messages: []string{"Something went wrong with your crafting."}}
		}

		// Create the weapon with all adj/val values transferred from the source material.
		// If the material carried no Adj1 (e.g. plain "steel" arch 1372), derive one from
		// the metal name the player typed so the weapon shows as "a steel dagger" etc.
		adj1 := player.CraftingAdj1
		if adj1 == 0 && player.CraftingMetal != "" {
			adj1 = e.adjByName(player.CraftingMetal)
		}
		// CraftingVal2 held the ore's color (1=purple, 2=indigo, 3=blue) through the
		// smelt→forge pipeline. Convert it now to a non-magical to-hit bonus (Sharpness).
		// Val2 on the finished weapon means magical enchantment, so it must be zeroed.
		smithSkill := player.Skills[8]
		sharpness := weaponSharpnessBonus(player.CraftingVal2, smithSkill)
		val3 := player.CraftingVal3
		origAdjs := [3]int{adj1, player.CraftingAdj2, player.CraftingAdj3}
		finishedAdjs := canonicalizeOilyAdjs(origAdjs)
		if finishedAdjs != origAdjs {
			val3 = 0
		}

		baseSkill := weaponDef.Parameter1
		if baseSkill < 1 {
			baseSkill = 1
		}
		// Val1 = copper value per GMSCRIPT.DOC; computeSellValue() pays out half of it
		// when the item is sold, matching the standard shop margin. Scale the value with
		// the skill level required to craft the weapon, but never let it exceed what the
		// raw material actually cost (player.CraftingVal1, stamped at purchase in doBuy)
		// — so the eventual sale (half of Val1) is capped at half the material cost.
		// Otherwise players could launder cheap material into a profit by forging and
		// reselling it. Materials with no known purchase price (e.g. mined ore) leave
		// the level-scaled value uncapped.
		itemValue := baseSkill * 30
		if player.CraftingVal1 > 0 && itemValue > player.CraftingVal1 {
			itemValue = player.CraftingVal1
		}
		if itemValue < 1 {
			itemValue = 1
		}

		item := InventoryItem{
			Archetype: weaponDef.Number,
			Adj1:      finishedAdjs[0],
			Adj2:      finishedAdjs[1],
			Adj3:      finishedAdjs[2],
			Val1:      itemValue,
			Val2:      0,         // no magical enchantment from forging
			Val3:      val3,      // elemental crit type (from ore); 0 if oily/oiled
			Val4:      player.CraftingVal4,
			Val5:      player.CraftingVal5,
			Sharpness: sharpness, // non-magical quality bonus, forged into the weapon
		}
		player.Inventory = append(player.Inventory, item)

		// Award XP: 25 per skill level required + metal quality bonus + sharpness bonus
		sharpnessBonus := 0
		switch {
		case sharpness >= 10:
			sharpnessBonus = 100
		case sharpness >= 7:
			sharpnessBonus = 50
		case sharpness >= 4:
			sharpnessBonus = 25
		case sharpness >= 1:
			sharpnessBonus = 10
		}
		xpAward := baseSkill*25 + metalQualityXPBonus(player.CraftingMetal) + sharpnessBonus
		player.Experience += xpAward

		itemName := e.formatItemName(weaponDef, item.Adj1, item.Adj2, item.Adj3, item.Tail)
		craftingMetal := player.CraftingMetal
		craftingItem := player.CraftingItem

		player.CraftingMetal = ""
		player.CraftingItem = ""
		player.CraftingSkill = ""
		player.CraftingAdj1, player.CraftingAdj2, player.CraftingAdj3 = 0, 0, 0
		player.CraftingVal1, player.CraftingVal2, player.CraftingVal3, player.CraftingVal4, player.CraftingVal5 = 0, 0, 0, 0, 0
		e.SavePlayer(ctx, player)

		sharpnessDesc := "rather dull"
		switch {
		case sharpness >= 10:
			sharpnessDesc = "exceptionally sharp"
		case sharpness >= 7:
			sharpnessDesc = "very sharp"
		case sharpness >= 4:
			sharpnessDesc = "sharp"
		case sharpness >= 1:
			sharpnessDesc = "serviceable"
		}
		msgs := []string{
			fmt.Sprintf("You carefully sharpen your weapon on a large whetstone until its cutting edge is honed to deadly precision. Your %s %s is complete! The blade is %s. (+%d non-magical bonus)", craftingMetal, craftingItem, sharpnessDesc, sharpness),
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
		player.CraftingSkill = ""
		player.CraftingMetal = ""
		player.CraftingAdj1, player.CraftingAdj2, player.CraftingAdj3 = 0, 0, 0
		player.CraftingVal1, player.CraftingVal2, player.CraftingVal3, player.CraftingVal4, player.CraftingVal5 = 0, 0, 0, 0, 0
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

	// Find the weapon in inventory with DAMAGED state. Keep scanning past
	// name-matches that aren't damaged — a player carrying more than one item
	// with the same name (e.g. two steel stilettos, one damaged, one not)
	// would otherwise always hit whichever one happens to sit first in the
	// slice and be wrongly told "That doesn't need repair" even though a
	// damaged match exists later on.
	found := false
	for i, ii := range player.Inventory {
		def := e.items[ii.Archetype]
		if def == nil {
			continue
		}
		if !isWeapon(def.Type) {
			continue
		}
		name := e.getItemNounName(def)
		if !matchesTarget(name, target, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
			continue
		}
		found = true

		if ii.State != "DAMAGED" {
			continue
		}

		// Skill check: base 40% + smithSkill*5
		chance := 40 + smithSkill*5
		if chance > 95 {
			chance = 95
		}
		roll := rand.Intn(100) + 1

		repairRT := applyRoundTime(player, 10)
		player.RoundTimeExpiry = time.Now().Add(time.Duration(repairRT) * time.Second)
		player.RoundTime = repairRT

		itemName := e.formatItemName(def, ii.Adj1, ii.Adj2, ii.Adj3, ii.Tail)

		if roll > chance {
			e.SavePlayer(ctx, player)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("[Success: %d%%, Roll %d] You are unable to repair the weapon.", chance, roll)},
				RoomBroadcast: []string{fmt.Sprintf("%s works at the forge, trying to repair a weapon.", player.FirstName)},
				PlayerState:   player,
			}
		}

		// Success: remove DAMAGED state and the "damaged" adjective (83) from Adj1
		player.Inventory[i].State = ""
		if player.Inventory[i].Adj1 == 83 {
			player.Inventory[i].Adj1 = 0
		}
		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages:      []string{fmt.Sprintf("[Success: %d%%, Roll %d] You carefully repair your %s.", chance, roll, itemName)},
			RoomBroadcast: []string{fmt.Sprintf("%s works at the forge, repairing a weapon.", player.FirstName)},
			PlayerState:   player,
		}
	}

	if found {
		return &CommandResult{Messages: []string{"That doesn't need repair."}}
	}
	return &CommandResult{Messages: []string{"You aren't carrying that."}}
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

	// FORAGE requires Wood Lore (skill #18) per LEGENDS.DOC: "Allows a character
	// with Wood Lore skill to search a wilderness area for useful substances."
	woodLore := player.Skills[18]
	if woodLore < 1 {
		return &CommandResult{Messages: []string{"You have no training in Wood Lore."}}
	}

	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You must wait %.0f more seconds.", remaining)}}
	}
	rt := applyRoundTime(player, 10)
	player.RoundTimeExpiry = time.Now().Add(time.Duration(rt) * time.Second)
	player.RoundTime = rt

	// Success chance grows with Wood Lore skill and Perception, mirroring MINE's
	// skill-scaled chance (base 30% + skill*5 + STR/10, capped 90%).
	chance := 30 + woodLore*6 + player.Perception/10
	if chance > 90 {
		chance = 90
	}
	if rand.Intn(100) >= chance {
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      []string{"You search the area but find nothing useful.", fmt.Sprintf("[Round: %d sec]", rt)},
			RoomBroadcast: []string{fmt.Sprintf("%s forages in the area.", player.FirstName)},
			PlayerState:   player,
		}
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
		return e.doForageFallback(ctx, player, rt)
	}

	// Weighted random selection. Wood Lore skill biases the odds toward rarer
	// (lower-ratio) finds: each candidate's weight gets a bonus proportional to
	// how rare it is and the player's skill, so a novice mostly finds common
	// items while a master has a real shot at the rare ones (e.g. mandrake root).
	maxRatio := 0
	for _, fd := range candidates {
		if fd.Ratio > maxRatio {
			maxRatio = fd.Ratio
		}
	}
	weights := make([]int, len(candidates))
	totalWeight := 0
	for i, fd := range candidates {
		w := fd.Ratio + (maxRatio-fd.Ratio)*woodLore/10
		if w < 1 {
			w = 1
		}
		weights[i] = w
		totalWeight += w
	}
	if totalWeight <= 0 {
		return e.doForageFallback(ctx, player, rt)
	}

	roll := rand.Intn(totalWeight)
	cumulative := 0
	for i, fd := range candidates {
		cumulative += weights[i]
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
				// Item scripts (e.g. the herb IFVAR ITEMADJ3 = ... blocks in
				// ITEMNWPN.SCR) check ITEMADJ3 for the variety, same as doBuy's
				// StoreItem adjective. Adj1 here would leave EAT unable to find
				// the match, so e.g. foraged Coriam Seed never grants its Empathy
				// bonus even though the store-bought one does.
				item.Adj3 = fd.AdjNum
			}
			player.Inventory = append(player.Inventory, item)
			e.SavePlayer(ctx, player)

			itemName := e.formatItemName(itemDef, 0, 0, item.Adj3)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You search the area and find %s!", itemName), fmt.Sprintf("[Round: %d sec]", rt)},
				RoomBroadcast: []string{fmt.Sprintf("%s forages in the area.", player.FirstName)},
				PlayerState:   player,
			}
		}
	}

	e.SavePlayer(ctx, player)
	return &CommandResult{
		Messages:    []string{"You search but find nothing useful.", fmt.Sprintf("[Round: %d sec]", rt)},
		PlayerState: player,
	}
}

// doForageFallback provides generic foraging when no ForageDefs are loaded for
// this terrain. rt is the round time already applied by doForageReal.
func (e *GameEngine) doForageFallback(ctx context.Context, player *Player, rt int) *CommandResult {
	room := e.rooms[player.RoomNumber]
	terrain := ""
	if room != nil {
		terrain = room.Terrain
	}

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

	if len(items) == 0 {
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      []string{"You search the area but find nothing useful.", fmt.Sprintf("[Round: %d sec]", rt)},
			RoomBroadcast: []string{fmt.Sprintf("%s forages in the area.", player.FirstName)},
			PlayerState:   player,
		}
	}

	chosen := items[rand.Intn(len(items))]
	item := InventoryItem{Archetype: chosen.arch}
	player.Inventory = append(player.Inventory, item)
	e.SavePlayer(ctx, player)

	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You search the area and find some %s!", chosen.name), fmt.Sprintf("[Round: %d sec]", rt)},
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
	// A LOOM room alone isn't enough — dyeing needs an actual cauldron to soak the
	// material in (item 1169/1505 in ITEM1.SCR, both named "cauldron"). Most LOOM
	// rooms (player housing, guild halls, etc.) don't have one; this is what stops
	// DYE from being usable anywhere a loom happens to be, e.g. out in the woods.
	hasCauldron := false
	for _, ri := range room.Items {
		if ri.IsPut {
			continue
		}
		def := e.items[ri.Archetype]
		if def != nil && strings.EqualFold(e.getItemNounName(def), "cauldron") {
			hasCauldron = true
			break
		}
	}
	if !hasCauldron {
		return &CommandResult{Messages: []string{"You need a cauldron here to dye items."}}
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

	// Find the item to dye (must be DYEABLE): inventory first, then loose on the
	// ground, then inside any container in the room (e.g. cloth left in the
	// weaver's cauldron to soak before dyeing).
	var targetItem *InventoryItem
	var targetDef *gameworld.ItemDef
	var roomTargetRI *gameworld.RoomItem      // matched loose on the ground
	var containerRef = -1                     // matched inside a container's dynamic (PUT-at-runtime) contents
	var containerIdx = -1                     // ...at this index within that container's contents
	var containerTargetRI *gameworld.RoomItem // matched via a script-authored PUT <ref> <archetype> item

	for i, ii := range player.Inventory {
		def := e.items[ii.Archetype]
		if def == nil || !containsFlag(def.Flags, "DYEABLE") {
			continue
		}
		name := strings.ToLower(e.getItemNounName(def))
		if strings.HasPrefix(name, itemTarget) {
			targetItem = &player.Inventory[i]
			targetDef = def
			break
		}
	}

	if targetItem == nil {
		for i := range room.Items {
			ri := &room.Items[i]
			if ri.IsPut {
				continue // inside something; not loose on the ground
			}
			def := e.items[ri.Archetype]
			if def == nil || !containsFlag(def.Flags, "DYEABLE") {
				continue
			}
			name := strings.ToLower(e.getItemNounName(def))
			if strings.HasPrefix(name, itemTarget) {
				roomTargetRI = ri
				targetDef = def
				break
			}
		}
	}

	if targetItem == nil && roomTargetRI == nil {
	searchContainers:
		for _, ri := range room.Items {
			if ri.IsPut {
				continue // this item is itself inside something; not a container to search
			}
			cdef := e.items[ri.Archetype]
			if cdef == nil || !isContainerDef(cdef) {
				continue
			}
			for i, ci := range e.roomContainerGet(room.Number, ri.Ref) {
				def := e.items[ci.Archetype]
				if def == nil || !containsFlag(def.Flags, "DYEABLE") {
					continue
				}
				name := strings.ToLower(e.getItemNounName(def))
				if strings.HasPrefix(name, itemTarget) {
					containerRef, containerIdx = ri.Ref, i
					targetDef = def
					break searchContainers
				}
			}
			for j := range room.Items {
				ri2 := &room.Items[j]
				if !ri2.IsPut || ri2.PutIn != ri.Ref {
					continue
				}
				def := e.items[ri2.Archetype]
				if def == nil || !containsFlag(def.Flags, "DYEABLE") {
					continue
				}
				name := strings.ToLower(e.getItemNounName(def))
				if strings.HasPrefix(name, itemTarget) {
					containerTargetRI = ri2
					targetDef = def
					break searchContainers
				}
			}
		}
	}

	if targetItem == nil && roomTargetRI == nil && containerRef == -1 && containerTargetRI == nil {
		return &CommandResult{Messages: []string{"You don't have a dyeable item matching that."}}
	}

	// Find the dye in inventory (must have DYE flag)
	for j, ii := range player.Inventory {
		def := e.items[ii.Archetype]
		if def == nil || !containsFlag(def.Flags, "DYE") {
			continue
		}
		name := strings.ToLower(e.getItemNounName(def))
		if strings.HasPrefix(name, dyeTarget) || strings.Contains(name, dyeTarget) {
			// Apply dye, preserving the material adjective in Adj1. Per the DYE-flagged
			// reagents in ITEM1.SCR (e.g. "Spleen for inky black": PARAMETER2=167 inky,
			// PARAMETER3=25 black), PARAMETER3 is the base color adjective and PARAMETER2
			// an optional modifier preceding it ("cobalt blue", "deep brown") — PARAMETER1
			// is not color data.
			var dyedName string
			switch {
			case targetItem != nil:
				if def.Parameter2 > 0 {
					targetItem.Adj2 = def.Parameter2
				}
				if def.Parameter3 > 0 {
					targetItem.Adj3 = def.Parameter3
				}
				dyedName = e.formatItemName(targetDef, targetItem.Adj1, targetItem.Adj2, targetItem.Adj3, targetItem.Tail)
			case roomTargetRI != nil:
				if def.Parameter2 > 0 {
					roomTargetRI.Adj2 = def.Parameter2
				}
				if def.Parameter3 > 0 {
					roomTargetRI.Adj3 = def.Parameter3
				}
				dyedName = e.formatItemName(targetDef, roomTargetRI.Adj1, roomTargetRI.Adj2, roomTargetRI.Adj3, roomTargetRI.Extend)
				itemCopy := *roomTargetRI
				e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_update", ItemRef: roomTargetRI.Ref, Item: &itemCopy})
			case containerRef != -1:
				contents := e.roomContainerGet(room.Number, containerRef)
				ci := &contents[containerIdx]
				if def.Parameter2 > 0 {
					ci.Adj2 = def.Parameter2
				}
				if def.Parameter3 > 0 {
					ci.Adj3 = def.Parameter3
				}
				e.roomContainerSet(room.Number, containerRef, contents)
				dyedName = e.formatItemName(targetDef, ci.Adj1, ci.Adj2, ci.Adj3, ci.Tail)
			case containerTargetRI != nil:
				if def.Parameter2 > 0 {
					containerTargetRI.Adj2 = def.Parameter2
				}
				if def.Parameter3 > 0 {
					containerTargetRI.Adj3 = def.Parameter3
				}
				dyedName = e.formatItemName(targetDef, containerTargetRI.Adj1, containerTargetRI.Adj2, containerTargetRI.Adj3, containerTargetRI.Extend)
				itemCopy := *containerTargetRI
				e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_update", ItemRef: containerTargetRI.Ref, Item: &itemCopy})
			}

			// Consume the dye
			player.Inventory = append(player.Inventory[:j], player.Inventory[j+1:]...)
			dyeRT := applyRoundTime(player, 15)
			player.RoundTimeExpiry = time.Now().Add(time.Duration(dyeRT) * time.Second)
			player.RoundTime = dyeRT
			e.SavePlayer(ctx, player)

			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You carefully dye the material. It is now %s.", dyedName), "[Round: 15 sec]"},
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

	for i := range player.Inventory {
		ii := &player.Inventory[i]
		def := e.items[ii.Archetype]
		if e.matchesItemOrPotion(def, ii.Adj1, ii.Adj2, ii.Adj3, ii.Val2, ii.Val4, target) {
			return e.analyzeItem(player, ii, def)
		}
	}

	// Not found directly — check one level into any open carried container
	// (e.g. an alchemist analyzing a potion sitting inside an open bag).
	for _, container := range e.findOpenContainers(player.Inventory) {
		for i := range container.Contents {
			ci := &container.Contents[i]
			cdef := e.items[ci.Archetype]
			if e.matchesItemOrPotion(cdef, ci.Adj1, ci.Adj2, ci.Adj3, ci.Val2, ci.Val4, target) {
				return e.analyzeItem(player, ci, cdef)
			}
		}
	}

	return &CommandResult{Messages: []string{"You don't have that."}}
}

// analyzeItem performs the ANALYZE effect on one matched item: ore quality,
// alchemical reagent properties, or — for a LIQCONTAINER holding a potion —
// identifying its bound spell, gated on the Alchemy skill (31).
func (e *GameEngine) analyzeItem(player *Player, ii *InventoryItem, def *gameworld.ItemDef) *CommandResult {
	if def == nil {
		return &CommandResult{Messages: []string{"You can't determine anything special about that."}}
	}

	if def.Type == "ORE" {
		miningSkill := player.Skills[35]
		if miningSkill < 3 {
			return &CommandResult{Messages: []string{"You don't have enough mining skill to analyze this ore. (Need Mining 3+)"}}
		}
		purity := ii.Val1
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
		msg := fmt.Sprintf("You examine the ore carefully. It appears to be of %s quality. (Purity: %d%%)", desc, purity)
		// At Mining 5+ the smith's eye can gauge the metal's color potential.
		if miningSkill >= 5 && ii.Val2 > 0 {
			colorNames := map[int]string{1: "purple", 2: "indigo", 3: "blue"}
			if cName, ok := colorNames[ii.Val2]; ok {
				msg += fmt.Sprintf(" The metal has a %s quality.", cName)
			}
		}
		return &CommandResult{Messages: []string{msg}}
	}

	if def.Type == "LIQCONTAINER" {
		itemName := e.formatItemName(def, ii.Adj1, ii.Adj2, ii.Adj3, ii.Tail)
		if !containerIsOpen(def, ii.State) {
			return &CommandResult{Messages: []string{fmt.Sprintf("%s is closed.", capitalize(itemName))}}
		}
		if ii.Val2 <= 0 {
			return &CommandResult{Messages: []string{fmt.Sprintf("%s is empty; there's nothing to analyze.", capitalize(itemName))}}
		}
		alchemySkill := player.Skills[31]
		if alchemySkill < 1 {
			return &CommandResult{Messages: []string{"You don't know enough alchemy to analyze potions. (Requires Alchemy 1+)"}}
		}
		if ii.Val3 == 0 {
			return &CommandResult{Messages: []string{fmt.Sprintf("You analyze %s. It's an ordinary liquid — no magical properties.", itemName)}}
		}
		spell := FindSpellByID(ii.Val3)
		if spell == nil {
			return &CommandResult{Messages: []string{fmt.Sprintf("You analyze %s, but can't identify its magical properties.", itemName)}}
		}
		return &CommandResult{Messages: []string{fmt.Sprintf("You analyze %s. It contains the effects of '%s' (spell #%d).", itemName, spell.Name, spell.ID)}}
	}

	// Reagent analysis for alchemy
	if containsFlag(def.Flags, "REAGENT") {
		itemName := e.formatItemName(def, ii.Adj1, ii.Adj2, ii.Adj3, ii.Tail)
		if tier, ok := lookupAlchemyCatalyst(ii.Archetype, ii.Adj1, ii.Adj2, ii.Adj3); ok {
			tierNames := map[int]string{1: "Mild Catalyst", 2: "Strong Catalyst", 3: "Very Strong Catalyst"}
			return &CommandResult{Messages: []string{fmt.Sprintf("You analyze %s. Alchemical properties: %s.", itemName, tierNames[tier])}}
		}
		if letter, ok := lookupAlchemyLetter(ii.Archetype, ii.Adj1, ii.Adj2, ii.Adj3); ok {
			return &CommandResult{Messages: []string{fmt.Sprintf("You analyze %s. Alchemical properties: %s.", itemName, alchemyLetterNames[letter])}}
		}
		return &CommandResult{Messages: []string{fmt.Sprintf("You analyze %s. It has no notable alchemical properties.", itemName)}}
	}

	return &CommandResult{Messages: []string{"You can't determine anything special about that."}}
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
}

// alchemyRecipes intentionally omits the "Bottle Color" column from
// original/legends/alchemy.txt — that was just the compiling player's own notes for
// tracking which container held which potion, not a game mechanic, so it has no bearing
// on brewing and isn't shown to players. The actual container's appearance when examined
// is a random liquid-appearance adjective (see potionLiquidAdjIDs), same as any other potion.
var alchemyRecipes = []alchemyRecipe{
	{"1AB", 1, "A", "B", 316, 1, "Body Restoration I"},
	{"1AF", 1, "A", "F", 313, 1, "Body Destruction I"},
	{"1AH", 1, "A", "H", 520, 1, "Night Vision"},
	{"1AG", 1, "A", "G", 518, 2, "Claw Growth"},
	{"1AC", 1, "A", "C", 339, 3, "Destroy Undead I"},
	{"1CH", 1, "C", "H", 506, 3, "Resist Weather"},
	{"1EG", 1, "E", "G", 226, 3, "Paranoia"},
	{"1AI", 1, "A", "I", 207, 4, "Strength I"},
	{"1GI", 1, "G", "I", 513, 4, "Agility I"},
	{"1AD", 1, "A", "D", 102, 5, "Mystic Armor"},
	{"2AF", 2, "A", "F", 314, 5, "Body Destruction II"},
	{"2CF", 2, "C", "F", 317, 5, "Body Restoration II"},
	{"2CG", 2, "C", "G", 401, 5, "Dispel Lesser Magic"},
	{"2HI", 2, "H", "I", 210, 5, "Haste"},
	{"2DG", 2, "D", "G", 521, 7, "Camouflage"},
	{"2AG", 2, "A", "G", 511, 8, "Carapace"},
	{"2AI", 2, "A", "I", 208, 8, "Strength II"},
	{"2AC", 2, "A", "C", 335, 9, "Invigoration II"},
	{"2EG", 2, "E", "G", 403, 9, "Mindlink"},
	{"2CH", 2, "C", "H", 509, 10, "Repel Plants"},
	{"3AF", 3, "A", "F", 315, 10, "Body Destruction III"},
	{"2BI", 2, "B", "I", 303, 11, "Cure Poison"},
	{"2GI", 2, "G", "I", 514, 11, "Agility II"},
	{"3AG", 3, "A", "G", 224, 11, "Fly"},
	{"2BD", 2, "B", "D", 319, 12, "Cure Disease"},
	{"2DI", 2, "D", "I", 234, 13, "Spell Shield"},
	{"3DG", 3, "D", "G", 225, 14, "Invisibility"},
	{"3AD", 3, "A", "D", 105, 15, "Globe of Protection"},
	{"3AI", 3, "A", "I", 209, 16, "Strength III"},
	{"3GI", 3, "G", "I", 515, 16, "Agility III"},
	{"3CH", 3, "C", "H", 510, 18, "Repel Plants & Webs"},
	{"3FJ", 3, "F", "J", 232, 20, "Mist Form"},
}

// alchemyReagentKey identifies a specific alchemical ingredient by its item archetype
// (INUMBER) and adjective. Reagent items are generic nouns (e.g. archetype 494 is just
// "root") distinguished only by which adjective is attached (542=mandrake, 531=babich, ...),
// so — unlike most of the crafting system — alchemy can't key off a flag+Val on the
// archetype; it has to key off the specific (archetype, adjective) pair, sourced by
// cross-referencing original/legends/alchemy.txt against the real item/adjective/
// FORAGEDEF/STOREITEM data in original/scripts/. adj=0 means the ingredient carries no
// distinguishing adjective (e.g. plain "garlic", "emeralds", "moonstones").
type alchemyReagentKey struct {
	archetype int
	adj       int
}

// alchemyCatalystTier maps a catalyst ingredient to its grade: 1=Mild (MCAT), 2=Strong
// (SCAT), 3=Very Strong (VCAT). Only mandrake root (tier 1) and meteoric dust (assigned
// here to tier 2) are currently confirmed obtainable in the world data (store/forage);
// the rest are included so the recipe still resolves correctly if such an item is ever
// placed, looted, or GM-granted.
var alchemyCatalystTier = map[alchemyReagentKey]int{
	{494, 542}:   1, // mandrake root
	{520, 723}:   1, // sharkhor eyes
	{1439, 723}:  1,
	{515, 654}:   1, // beetle hair
	{512, 1115}:  2, // fungal dust
	{1152, 1115}: 2,
	{520, 653}:   2, // ant eyes
	{1439, 653}:  2,
	{515, 653}:   2, // ant hair
	{527, 653}:   2, // ant legs
	{510, 142}:   2, // giant tooth
	{512, 1104}:  2, // meteoric dust (confirmed purchasable — see CHLRFALL.SCR-style stores)
	{1152, 1104}: 2,
	{512, 1072}:  3, // doubloon dust
	{1152, 1072}: 3,
	{108, 0}:     3, // emeralds
}

// alchemyReagentLetter maps an effect ingredient to its category letter (A-J), matching
// the column headers in original/legends/alchemy.txt (A:BODY, B:HLTH, C:NEGN, D:PROT,
// E:MIND, F:HARM, G:ENHC, H:MMAG, I:SMAG, J:VMAG). "Pearls" are deliberately omitted —
// the source doc lists them as a filler ingredient under nearly every category, so they
// have no single deterministic letter.
var alchemyReagentLetter = map[alchemyReagentKey]string{
	{1502, 1103}: "A", // muur crystals
	{1690, 609}:  "A", // onoki moss
	{1858, 1416}: "A", // wrinkled mushroom (real forageable Body reagent, FORAGEDEF val5=6)
	{275, 531}:   "B", // babich blossoms
	{494, 531}:   "B", // babich root
	{1501, 0}:    "C", // garlic
	{1681, 1244}: "C", // tricana seedlings
	{113, 0}:     "D", // moonstones
	{1442, 196}:  "D", // mirmach bark (confirmed forageable, FORAGEDEF val5=13)
	{521, 433}:   "D", // troll tongue
	{504, 567}:   "E", // ettin beards
	{514, 431}:   "E", // goblin hides
	{587, 554}:   "F", // deadly nightshade (confirmed purchasable)
	{520, 799}:   "F", // toad eyes
	{1439, 799}:  "F",
	{1650, 468}:  "G", // golden flowers
	{504, 798}:   "G", // spriggan beards
	{1153, 37}:   "G", // brilliant feathers
	{1689, 608}:  "G", // roseate eggshells
	{507, 556}:   "G", // skrag scales
	{520, 1105}:  "H", // newt eyes (confirmed purchasable)
	{1439, 1105}: "H",
	{1144, 0}:    "I", // coconut
	{1450, 0}:    "I",
	{1680, 1236}: "I", // manango pits
	{528, 545}:   "I", // spider fangs
	{505, 1115}:  "I", // fungal claws
	{1688, 1235}: "J", // riyong mushroom
	{1858, 1235}: "J",
	{118, 0}:     "J", // rubies
	{1441, 145}:  "J", // glowing fungus
}

var alchemyLetterNames = map[string]string{
	"A": "Body", "B": "Health", "C": "Negation", "D": "Protection", "E": "Mind",
	"F": "Harmful", "G": "Enhancing", "H": "Mild magic", "I": "Strong magic", "J": "Very strong magic",
}

// lookupAlchemyCatalyst checks all three adjective slots of an item instance (bought
// reagents carry their identifying adjective in Adj3; foraged/room-placed ones in Adj1)
// against alchemyCatalystTier.
func lookupAlchemyCatalyst(archetype, adj1, adj2, adj3 int) (tier int, ok bool) {
	for _, a := range []int{adj1, adj2, adj3, 0} {
		if t, found := alchemyCatalystTier[alchemyReagentKey{archetype, a}]; found {
			return t, true
		}
	}
	return 0, false
}

// lookupAlchemyLetter is the same lookup as lookupAlchemyCatalyst but against
// alchemyReagentLetter.
func lookupAlchemyLetter(archetype, adj1, adj2, adj3 int) (letter string, ok bool) {
	for _, a := range []int{adj1, adj2, adj3, 0} {
		if l, found := alchemyReagentLetter[alchemyReagentKey{archetype, a}]; found {
			return l, true
		}
	}
	return "", false
}

// alchemyReagentCode resolves an ingredient to a single packed code: 1-3 for a catalyst
// tier, or 11-20 for a letter category (A=11 .. J=20), or 0 if it's not a recognized
// alchemical ingredient (brewing with it will simply fail to match any recipe).
func alchemyReagentCode(archetype, adj1, adj2, adj3 int) int {
	if tier, ok := lookupAlchemyCatalyst(archetype, adj1, adj2, adj3); ok {
		return tier
	}
	if letter, ok := lookupAlchemyLetter(archetype, adj1, adj2, adj3); ok {
		return 11 + int(letter[0]-'A')
	}
	return 0
}

// alchemyCodeToLetter reverses the letter half of alchemyReagentCode's encoding.
func alchemyCodeToLetter(code int) string {
	if code < 11 || code > 20 {
		return ""
	}
	return string(rune('A' + code - 11))
}

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
				msgs = append(msgs, fmt.Sprintf("  Level %2d: %s", r.level, r.name))
			}
		}
		if len(msgs) == 1 {
			msgs = append(msgs, "  You don't know any recipes at your current level.")
		}
		return &CommandResult{Messages: msgs}
	}

	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You must wait %.0f more seconds.", remaining)}}
	}

	// BREW <reagent> IN <container>
	raw := strings.ToLower(strings.Join(args, " "))
	reagentTarget, containerTarget := parseInClause(raw)
	if containerTarget == "" {
		return &CommandResult{Messages: []string{"Brew in what? Usage: BREW <reagent> IN <flask/vial>"}}
	}

	// Find the reagent. Reagent items are generic nouns (e.g. "root") distinguished only
	// by their adjective (e.g. "mandrake root"), so this must match on noun+adjectives —
	// not the bare noun — like every other item lookup in the codebase.
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
		name := e.getItemNounName(def)
		if matchesTarget(name, reagentTarget, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
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
		if def.Type != "LIQCONTAINER" && def.Container != "IN" {
			continue
		}
		name := e.getItemNounName(def)
		if matchesTarget(name, containerTarget, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
			containerIdx = i
			break
		}
	}
	if containerIdx < 0 {
		return &CommandResult{Messages: []string{"You don't have a suitable container. You need a flask, vial, or bottle."}}
	}

	// Resolve the reagent's alchemical code (catalyst tier 1-3, or letter category 11-20)
	// from its actual archetype+adjective — not a fabricated per-instance flag — before
	// it's consumed. An unrecognized REAGENT item still gets added (code 0) so the brew
	// fails naturally as an invalid combination rather than being silently rejected here.
	code := alchemyReagentCode(reagentItem.Archetype, reagentItem.Adj1, reagentItem.Adj2, reagentItem.Adj3)

	reagentDef := e.items[reagentItem.Archetype]
	reagentName := "the reagent"
	if reagentDef != nil {
		reagentName = e.formatItemName(reagentDef, reagentItem.Adj1, reagentItem.Adj2, reagentItem.Adj3, reagentItem.Tail)
	}

	// Consume reagent
	player.Inventory = append(player.Inventory[:reagentIdx], player.Inventory[reagentIdx+1:]...)
	// Fix index if container was after reagent
	if containerIdx > reagentIdx {
		containerIdx--
	}
	container := &player.Inventory[containerIdx]

	// Track ingredients in container's Val fields: Val5 = count added so far (0-3),
	// Val4 = the 3 codes packed 2 digits each (order they were added — brewing order
	// doesn't matter, so this is decoded as an unordered set of 3 once complete).
	container.Val5++
	container.Val4 = container.Val4*100 + code

	// Every brew step takes a round, same as the other crafting skills (15 sec, 7 if hasted).
	brewRT := applyRoundTime(player, 15)
	player.RoundTimeExpiry = time.Now().Add(time.Duration(brewRT) * time.Second)
	player.RoundTime = brewRT

	if container.Val5 < 3 {
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You add %s to the brew. (%d/3 ingredients)", reagentName, container.Val5), fmt.Sprintf("[Round: %d sec]", brewRT)},
			RoomBroadcast: []string{fmt.Sprintf("%s adds an ingredient to a bubbling brew.", player.FirstName)},
			PlayerState:   player,
		}
	}

	// 3 ingredients added — attempt to brew! Decode the three codes; brewing order
	// doesn't matter, so identify whichever one is the catalyst (any order) and treat
	// the other two as the pair of effect letters.
	codes := [3]int{
		container.Val4 % 100,
		(container.Val4 / 100) % 100,
		(container.Val4 / 10000) % 100,
	}
	catLevel := 0
	var letters []string
	for _, c := range codes {
		if c >= 1 && c <= 3 {
			catLevel = c
			continue
		}
		letters = append(letters, alchemyCodeToLetter(c))
	}

	// Try to match a recipe: need exactly one catalyst and two valid letters.
	if catLevel > 0 && len(letters) == 2 {
		for _, recipe := range alchemyRecipes {
			if recipe.catalyst != catLevel {
				continue
			}
			match := (letters[0] == recipe.reagent1 && letters[1] == recipe.reagent2) ||
				(letters[0] == recipe.reagent2 && letters[1] == recipe.reagent1)
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
						fmt.Sprintf("[Round: %d sec]", brewRT),
					},
					PlayerState: player,
				}
			}

			// Success! Create potion — sip count matches the container's own capacity
			// (Vial:2, Flask:5, Flagon:6, Bottle:10, Ewer:6), not a fixed random range.
			// The container's appearance gets a random liquid-appearance adjective, same
			// as any other potion (randomPotionDrop) — its look gives no hint of the
			// bound effect, which is only revealed by ANALYZE or by drinking it.
			containerDef := e.items[container.Archetype]
			sips := 5
			if containerDef != nil && containerDef.Interior > 0 {
				sips = containerDef.Interior
			}
			container.Val3 = recipe.spellID // spell stored in container
			container.Val4 = potionLiquidAdjIDs[rand.Intn(len(potionLiquidAdjIDs))]
			container.Val5 = 0
			container.Val2 = sips

			// XP award scales with the potion's recipe level, same formula (level*20)
			// used for jewelry/wood/weaving crafts.
			xpAward := recipe.level * 20
			player.Experience += xpAward
			e.SavePlayer(ctx, player)

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf("The brew shimmers magically! You have created a %s potion! (%d sips)", recipe.name, container.Val2),
					fmt.Sprintf("[Round: %d sec]", brewRT),
					fmt.Sprintf("You have been awarded %d experience points.", xpAward),
				},
				RoomBroadcast: []string{fmt.Sprintf("%s completes a potion that shimmers with magical energy!", player.FirstName)},
				PlayerState:   player,
			}
		}
	}

	// No match — failed recipe
	container.Val4 = 0
	container.Val5 = 0
	e.SavePlayer(ctx, player)
	return &CommandResult{
		Messages:      []string{"A foul odor rises from the brew. The combination produces nothing useful.", fmt.Sprintf("[Round: %d sec]", brewRT)},
		RoomBroadcast: []string{fmt.Sprintf("%s's brew emits a foul odor.", player.FirstName)},
		PlayerState:   player,
	}
}

// ---- ENCRUST / ENGRAVE HELPERS ----

// findJewelerItem searches inventory and worn (and optionally wielded) for the first item
// passing filter that matches target. If no match is found with the full target, it
// progressively drops trailing words — handling natural-language input like "barrette set"
// when the item noun is just "barrette".
// Returns (invIdx, wornIdx, wieldedMatch, def); negative indices mean not found there.
func (e *GameEngine) findJewelerItem(player *Player, target string, inclWielded bool, filter func(*gameworld.ItemDef) bool) (invIdx, wornIdx int, wieldedMatch bool, def *gameworld.ItemDef) {
	words := strings.Fields(target)
	for len(words) > 0 {
		t := strings.Join(words, " ")
		for i, ii := range player.Inventory {
			d := e.items[ii.Archetype]
			if d == nil || !filter(d) {
				continue
			}
			if matchesTarget(e.getItemNounName(d), t, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
				return i, -1, false, d
			}
		}
		for i, ii := range player.Worn {
			d := e.items[ii.Archetype]
			if d == nil || !filter(d) {
				continue
			}
			if matchesTarget(e.getItemNounName(d), t, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
				return -1, i, false, d
			}
		}
		if inclWielded && player.Wielded != nil {
			d := e.items[player.Wielded.Archetype]
			if d != nil && filter(d) && matchesTarget(e.getItemNounName(d), t, e.getAdjName(player.Wielded.Adj1), e.getAdjName(player.Wielded.Adj2), e.getAdjName(player.Wielded.Adj3)) {
				return -1, -1, true, d
			}
		}
		words = words[:len(words)-1]
	}
	return -1, -1, false, nil
}

// ---- ENCRUST / INLAY / INSET ----

// doGemAdorn implements the shared logic behind ENCRUST, INLAY and INSET: each
// sets a gem into an ENCRUSTABLE item, differing only in the verb used for
// messages and the resulting adjective (e.g. "encrusted", "inlaid", "inset").
func (e *GameEngine) doGemAdorn(ctx context.Context, player *Player, args []string, verb string, resultAdjName string, fallbackAdjID int) *CommandResult {
	verbUpper := strings.ToUpper(verb)
	if player.Skills[0] < 3 {
		return &CommandResult{Messages: []string{fmt.Sprintf("You need Jeweler skill level 3 to %s items.", verb)}}
	}
	room := e.rooms[player.RoomNumber]
	if room == nil || (!containsModifier(room.Modifiers, "FORGE") && !containsModifier(room.Modifiers, "LOOM")) {
		return &CommandResult{Messages: []string{fmt.Sprintf("You need to be at a forge or jeweler's workshop to %s items.", verb)}}
	}
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You must wait %.0f more seconds.", remaining)}}
	}
	if len(args) == 0 {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s what? Usage: %s <item> WITH <gem>", strings.Title(verb), verbUpper)}}
	}

	raw := strings.ToLower(strings.Join(args, " "))
	itemTarget, gemTarget := parseWithClause(raw)
	if gemTarget == "" {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s with what gem? Usage: %s <item> WITH <gem>", strings.Title(verb), verbUpper)}}
	}

	itemIdx, wornIdx, _, targetDef := e.findJewelerItem(player, itemTarget, false, func(d *gameworld.ItemDef) bool {
		return containsFlag(d.Flags, "ENCRUSTABLE")
	})
	if itemIdx < 0 && wornIdx < 0 {
		return &CommandResult{Messages: []string{"You don't have an encrustable item matching that."}}
	}

	// Check that at least 2 adj slots are free
	var a1, a2, a3 int
	if itemIdx >= 0 {
		a1, a2, a3 = player.Inventory[itemIdx].Adj1, player.Inventory[itemIdx].Adj2, player.Inventory[itemIdx].Adj3
	} else {
		a1, a2, a3 = player.Worn[wornIdx].Adj1, player.Worn[wornIdx].Adj2, player.Worn[wornIdx].Adj3
	}
	freeSlots := 0
	for _, a := range []int{a1, a2, a3} {
		if a == 0 {
			freeSlots++
		}
	}
	if freeSlots < 2 {
		return &CommandResult{Messages: []string{fmt.Sprintf("That item already has too many adjectives to %s with a gem.", verb)}}
	}

	// Find gem in inventory (archetypes 99-123)
	gemIdx := -1
	var gemDef *gameworld.ItemDef
	for j, ii := range player.Inventory {
		def := e.items[ii.Archetype]
		if def == nil || ii.Archetype < 99 || ii.Archetype > 123 {
			continue
		}
		if matchesTarget(e.getItemNounName(def), gemTarget, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
			gemIdx = j
			gemDef = def
			break
		}
	}
	if gemIdx < 0 {
		return &CommandResult{Messages: []string{"You don't have that gem."}}
	}

	gemNoun := strings.ToLower(e.getItemNounName(gemDef))
	gemAdjID := e.adjByName(gemNoun)
	if gemAdjID == 0 {
		return &CommandResult{Messages: []string{"That gem type has no corresponding adjective."}}
	}
	resultAdjID := e.adjByName(resultAdjName)
	if resultAdjID == 0 {
		resultAdjID = fallbackAdjID
	}

	// First existing non-zero adj shifts to Adj3
	existingAdj := a1
	if existingAdj == 0 {
		existingAdj = a2
	}

	// Consume gem, adjusting item inventory index if it shifted
	player.Inventory = append(player.Inventory[:gemIdx], player.Inventory[gemIdx+1:]...)
	if itemIdx >= 0 && gemIdx < itemIdx {
		itemIdx--
	}

	// Apply: Adj1=gem noun, Adj2=result adjective, Adj3=existing
	newAdj1, newAdj2, newAdj3 := gemAdjID, resultAdjID, existingAdj
	var displayTail string
	if itemIdx >= 0 {
		player.Inventory[itemIdx].Adj1 = newAdj1
		player.Inventory[itemIdx].Adj2 = newAdj2
		player.Inventory[itemIdx].Adj3 = newAdj3
		displayTail = player.Inventory[itemIdx].Tail
	} else {
		player.Worn[wornIdx].Adj1 = newAdj1
		player.Worn[wornIdx].Adj2 = newAdj2
		player.Worn[wornIdx].Adj3 = newAdj3
		displayTail = player.Worn[wornIdx].Tail
	}

	rt := applyRoundTime(player, 30)
	player.RoundTimeExpiry = time.Now().Add(time.Duration(rt) * time.Second)
	player.RoundTime = rt
	e.SavePlayer(ctx, player)

	itemName := e.formatItemName(targetDef, newAdj1, newAdj2, newAdj3, displayTail)
	return &CommandResult{
		Messages: []string{
			fmt.Sprintf("You carefully set the %s into the piece, creating %s.", gemNoun, itemName),
			fmt.Sprintf("[Round: %d sec]", rt),
		},
		RoomBroadcast: []string{fmt.Sprintf("%s works carefully at the jeweler's bench.", player.FirstName)},
		PlayerState:   player,
	}
}

func (e *GameEngine) doEncrust(ctx context.Context, player *Player, args []string) *CommandResult {
	return e.doGemAdorn(ctx, player, args, "encrust", "encrusted", 114)
}

func (e *GameEngine) doInlay(ctx context.Context, player *Player, args []string) *CommandResult {
	return e.doGemAdorn(ctx, player, args, "inlay", "inlaid", 168)
}

func (e *GameEngine) doInset(ctx context.Context, player *Player, args []string) *CommandResult {
	return e.doGemAdorn(ctx, player, args, "inset", "inset", 650)
}

// ---- HIGHLANDER GEM MOLDING ----

// moldFailChance returns a Highlander's percent chance of botching a MOLD
// attempt: 20% at level 1, dropping 1% per level, bottoming out at a
// permanent 1% — there's always some risk, however skilled they get.
func moldFailChance(level int) int {
	chance := 20 - (level - 1)
	if chance < 1 {
		chance = 1
	}
	return chance
}

// moldValueBonusPercent returns the percentage a gem's value multiplier
// increases by on a successful mold: 25% at level 1, +5% per level, capped
// at a 100% increase.
func moldValueBonusPercent(level int) int {
	bonus := 25 + (level-1)*5
	if bonus > 100 {
		bonus = 100
	}
	return bonus
}

// doMold handles the Highlander MOLD ability ("MOLD chipped diamond", "MOLD 3 diamond"):
// works a chipped or cracked gem into a polished one, raising its sale value. A botched
// attempt (see moldFailChance) ruins the gem with the damaged adjective instead, and a
// damaged or already-polished gem can't be molded again.
func (e *GameEngine) doMold(ctx context.Context, player *Player, args []string) *CommandResult {
	if player.Race != RaceHighlander {
		return &CommandResult{Messages: []string{"Only Highlanders know how to mold gemstones."}}
	}
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Mold what? Usage: MOLD <gem>"}}
	}
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You are still recovering from your last action. (%.0f seconds remaining)", remaining+0.5)}}
	}

	target := strings.ToLower(strings.Join(args, " "))
	cleanTarget, skip := parseOrdinal(target)

	idx := -1
	var def *gameworld.ItemDef
	for i, inv := range player.Inventory {
		d := e.items[inv.Archetype]
		if d == nil || inv.Archetype < 99 || inv.Archetype > 123 {
			continue
		}
		if matchesTargetOrdinal(e.getItemNounName(d), cleanTarget, &skip, e.getAdjName(inv.Adj1), e.getAdjName(inv.Adj2), e.getAdjName(inv.Adj3)) {
			idx = i
			def = d
			break
		}
	}
	if idx < 0 {
		return &CommandResult{Messages: []string{"You don't have that gem."}}
	}

	// ADJDEF fallbacks per ADJNOUN.SCR, in case an adjective got renamed and the
	// name lookup below misses: 241 polished, 83 damaged, 53 chipped, 384 cracked.
	polishedID := e.adjByName("polished")
	if polishedID == 0 {
		polishedID = 241
	}
	damagedID := e.adjByName("damaged")
	if damagedID == 0 {
		damagedID = 83
	}
	chippedID := e.adjByName("chipped")
	if chippedID == 0 {
		chippedID = 53
	}
	crackedID := e.adjByName("cracked")
	if crackedID == 0 {
		crackedID = 384
	}

	gem := &player.Inventory[idx]
	slots := [3]*int{&gem.Adj1, &gem.Adj2, &gem.Adj3}

	for _, a := range slots {
		if polishedID != 0 && *a == polishedID {
			return &CommandResult{Messages: []string{"That gem has already been polished to perfection — there's nothing left to mold."}}
		}
		if damagedID != 0 && *a == damagedID {
			return &CommandResult{Messages: []string{"That gem was ruined in an earlier molding attempt — it's beyond saving now."}}
		}
	}

	var flawSlot *int
	for _, a := range slots {
		if (chippedID != 0 && *a == chippedID) || (crackedID != 0 && *a == crackedID) {
			flawSlot = a
			break
		}
	}
	if flawSlot == nil {
		return &CommandResult{Messages: []string{"That gem has no flaws for you to mold away."}}
	}

	itemName := e.formatItemName(def, gem.Adj1, gem.Adj2, gem.Adj3, gem.Tail)

	moldRT := applyRoundTime(player, 5)
	player.RoundTimeExpiry = time.Now().Add(time.Duration(moldRT) * time.Second)
	player.RoundTime = moldRT

	failChance := moldFailChance(player.Level)
	roll := rand.Intn(100) + 1

	if roll <= failChance {
		*flawSlot = damagedID
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("Your hands slip! You botch the working and damage your %s.", itemName),
				fmt.Sprintf("[Round: %d sec]", moldRT),
			},
			RoomBroadcast: []string{fmt.Sprintf("%s carefully works a gemstone between %s fingers.", player.FirstName, player.Possessive())},
			PlayerState:   player,
		}
	}

	*flawSlot = polishedID
	bonusPct := moldValueBonusPercent(player.Level)
	baseVal := gem.Val2
	if baseVal <= 0 {
		baseVal = 100
	}
	gem.Val2 = baseVal + baseVal*bonusPct/100

	newName := e.formatItemName(def, gem.Adj1, gem.Adj2, gem.Adj3, gem.Tail)
	e.SavePlayer(ctx, player)
	return &CommandResult{
		Messages: []string{
			fmt.Sprintf("Your skilled hands work the flaws out of your %s, leaving your %s!", itemName, newName),
			fmt.Sprintf("[Round: %d sec]", moldRT),
		},
		RoomBroadcast: []string{fmt.Sprintf("%s carefully works a gemstone between %s fingers.", player.FirstName, player.Possessive())},
		PlayerState:   player,
	}
}

// ---- ENGRAVE ----

// doEngrave engraves text onto an ENCRUSTABLE or HARDMETAL item.
// rawInput is the original-case command string so the engraving text preserves capitalisation.
func (e *GameEngine) doEngrave(ctx context.Context, player *Player, args []string, rawInput string) *CommandResult {
	if player.Skills[0] < 3 {
		return &CommandResult{Messages: []string{"You need Jeweler skill level 3 to engrave items."}}
	}
	room := e.rooms[player.RoomNumber]
	if room == nil || (!containsModifier(room.Modifiers, "FORGE") && !containsModifier(room.Modifiers, "LOOM")) {
		return &CommandResult{Messages: []string{"You need to be at a forge or jeweler's workshop to engrave items."}}
	}
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You must wait %.0f more seconds.", remaining)}}
	}
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Engrave what? Usage: ENGRAVE <item> WITH <text>"}}
	}

	// Item target comes from lowercased args; engrave text comes from original-case input
	// to preserve the player's capitalisation.
	rawLower := strings.ToLower(strings.Join(args, " "))
	itemTarget, _ := parseWithClause(rawLower)

	rawInputLower := strings.ToLower(rawInput)
	withIdx := strings.Index(rawInputLower, " with ")
	if withIdx < 0 {
		return &CommandResult{Messages: []string{"Engrave with what text? Usage: ENGRAVE <item> WITH <text>"}}
	}
	engraveText := strings.TrimSpace(rawInput[withIdx+6:])
	if engraveText == "" {
		return &CommandResult{Messages: []string{"What text would you like to engrave?"}}
	}
	if len(engraveText) > 60 {
		return &CommandResult{Messages: []string{"Engraving text is too long (maximum 60 characters)."}}
	}

	isEngraveTarget := func(d *gameworld.ItemDef) bool {
		return containsFlag(d.Flags, "ENCRUSTABLE") || d.Substance == "HARDMETAL"
	}
	itemIdx, wornIdx, wieldedMatch, targetDef := e.findJewelerItem(player, itemTarget, true, isEngraveTarget)
	if itemIdx < 0 && wornIdx < 0 && !wieldedMatch {
		return &CommandResult{Messages: []string{"You don't have an engraveable item matching that. Items must be encrustable or made of hard metal."}}
	}

	tail := engraveText
	var displayAdj1, displayAdj2, displayAdj3 int
	if itemIdx >= 0 {
		player.Inventory[itemIdx].Tail = tail
		displayAdj1 = player.Inventory[itemIdx].Adj1
		displayAdj2 = player.Inventory[itemIdx].Adj2
		displayAdj3 = player.Inventory[itemIdx].Adj3
	} else if wornIdx >= 0 {
		player.Worn[wornIdx].Tail = tail
		displayAdj1 = player.Worn[wornIdx].Adj1
		displayAdj2 = player.Worn[wornIdx].Adj2
		displayAdj3 = player.Worn[wornIdx].Adj3
	} else {
		player.Wielded.Tail = tail
		displayAdj1 = player.Wielded.Adj1
		displayAdj2 = player.Wielded.Adj2
		displayAdj3 = player.Wielded.Adj3
	}

	rt := applyRoundTime(player, 30)
	player.RoundTimeExpiry = time.Now().Add(time.Duration(rt) * time.Second)
	player.RoundTime = rt
	e.SavePlayer(ctx, player)

	itemName := e.formatItemName(targetDef, displayAdj1, displayAdj2, displayAdj3, tail)
	return &CommandResult{
		Messages: []string{
			fmt.Sprintf("You carefully engrave the inscription onto the item. It is now %s.", itemName),
			fmt.Sprintf("[Round: %d sec]", rt),
		},
		RoomBroadcast: []string{fmt.Sprintf("%s carefully engraves an inscription onto an item.", player.FirstName)},
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

// ---- NON-WEAPON CRAFTING WORK CYCLES ----

// findCraftMaterial searches inventory for an item whose noun/adjectives match
// target (via matchesTarget, so both a bare noun like "skin" and a fully
// qualified name like "brown snake skin" work) and checks whether it is a valid
// crafting material for the given skill ID (8=weaponsmithing/jeweler, 15=weaving,
// 18=wood). If a matching item isn't a suitable material, the search keeps going
// rather than stopping, so an unsuitable match (e.g. a crafted "cotton jacket"
// worn/carried alongside raw cotton) doesn't shadow a valid material later in the
// inventory. Returns the inventory index, item snapshot, and item def on success.
// On failure it returns idx=-1 and a human-readable error message.
func (e *GameEngine) findCraftMaterial(player *Player, target string, matSkillID int) (idx int, item InventoryItem, def *gameworld.ItemDef, errMsg string) {
	foundNoun := false
	for j, ii := range player.Inventory {
		mDef := e.items[ii.Archetype]
		if mDef == nil {
			continue
		}
		if !matchesTarget(e.getItemNounName(mDef), target, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
			continue
		}
		foundNoun = true
		var isValid bool
		if mDef.Type == "MATERIAL" || containsFlag(mDef.Flags, "MATERIAL2") {
			isValid = mDef.Parameter2 == matSkillID || mDef.Parameter2 == 0
		} else if mDef.Type == "MISC" {
			isValid = mDef.Parameter2 == matSkillID
		}
		if !isValid {
			continue
		}
		return j, ii, mDef, ""
	}
	if !foundNoun {
		return -1, InventoryItem{}, nil, fmt.Sprintf("You don't have any %s.", target)
	}
	return -1, InventoryItem{}, nil, fmt.Sprintf("That %s is not suitable material for that item.", target)
}

// buildCraftAdjs computes the adjective list for a crafted item from the source material.
func buildCraftAdjs(matItem InventoryItem, matDef *gameworld.ItemDef) (adj1, adj2, adj3 int) {
	var adjs []int
	for _, a := range []int{matItem.Adj1, matItem.Adj2, matItem.Adj3} {
		if a > 0 {
			adjs = append(adjs, a)
		}
	}
	if matDef.Parameter1 > 0 {
		present := false
		for _, a := range adjs {
			if a == matDef.Parameter1 {
				present = true
				break
			}
		}
		if !present {
			adjs = append(adjs, matDef.Parameter1)
		}
	}
	get := func(i int) int {
		if i < len(adjs) {
			return adjs[i]
		}
		return 0
	}
	return get(0), get(1), get(2)
}

// buildWeaveCraftAdjs computes the adjective list for a woven garment. Unlike
// buildCraftAdjs (which packs adjectives into the first free slot), the raw
// cloth's material type (matDef.Parameter1 — e.g. 72=cotton, 356=wool) is always
// pinned to Adj3, since FAYDINDR.SCR room 315's garment-finishing contraption
// checks ITEMADJ3 to decide whether a garment is cotton or wool. Any other
// adjectives on the raw cloth (e.g. a two-word dye color like "olive green"
// applied before weaving, split across Adj2/Adj3 — see doDye) are carried into
// Adj1/Adj2 instead of being squeezed out; dyeing with a single-word color
// still leaves just one.
func buildWeaveCraftAdjs(matItem InventoryItem, matDef *gameworld.ItemDef) (adj1, adj2, adj3 int) {
	var dyeAdjs []int
	for _, a := range []int{matItem.Adj1, matItem.Adj2, matItem.Adj3} {
		if a > 0 && a != matDef.Parameter1 {
			dyeAdjs = append(dyeAdjs, a)
		}
	}
	if len(dyeAdjs) > 0 {
		adj1 = dyeAdjs[0]
	}
	if len(dyeAdjs) > 1 {
		adj2 = dyeAdjs[1]
	}
	return adj1, adj2, matDef.Parameter1
}

// completeCraft creates the finished item for jewelry, weaving, or wood crafts and resets state.
func (e *GameEngine) completeCraft(ctx context.Context, player *Player, completionMsg, broadcastMsg string) *CommandResult {
	var craftDef *gameworld.ItemDef
	for _, def := range e.items {
		if !containsFlag(def.Flags, "CRAFTABLE") {
			continue
		}
		if strings.ToLower(e.nouns[def.NameID]) == player.CraftingItem {
			craftDef = def
			break
		}
	}

	resetState := func() {
		player.CraftingStep = 0
		player.CraftingItem = ""
		player.CraftingSkill = ""
		player.CraftingMetal = ""
		player.CraftingAdj1, player.CraftingAdj2, player.CraftingAdj3 = 0, 0, 0
		player.CraftingVal1, player.CraftingVal2, player.CraftingVal3, player.CraftingVal4, player.CraftingVal5 = 0, 0, 0, 0, 0
	}

	if craftDef == nil {
		resetState()
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"Something went wrong with your crafting."}}
	}

	item := InventoryItem{
		Archetype: craftDef.Number,
		Adj1:      player.CraftingAdj1,
		Adj2:      player.CraftingAdj2,
		Adj3:      player.CraftingAdj3,
	}
	player.Inventory = append(player.Inventory, item)

	// Jewelry/weaving items always carry difficulty on Parameter2 (Parameter1 unused
	// there). Wood Lore items are inconsistent in the source data — many non-launcher
	// items (instruments, staves, etc.) never got a Parameter2 assigned, so fall back
	// to Parameter1 (their only other small, per-item difficulty-shaped field) rather
	// than silently awarding zero.
	xpAward := 0
	if craftDef.Parameter2 > 0 {
		xpAward = craftDef.Parameter2 * 20
	} else if craftDef.Parameter1 > 0 {
		xpAward = craftDef.Parameter1 * 20
	}
	if xpAward > 0 {
		player.Experience += xpAward
	}

	rt := applyRoundTime(player, 15)
	player.RoundTimeExpiry = time.Now().Add(time.Duration(rt) * time.Second)
	player.RoundTime = rt

	itemName := e.formatItemName(craftDef, item.Adj1, item.Adj2, item.Adj3, item.Tail)
	resetState()
	e.SavePlayer(ctx, player)

	msgs := []string{completionMsg, fmt.Sprintf("[Round: %d sec]", rt)}
	if xpAward > 0 {
		msgs = append(msgs, fmt.Sprintf("You have been awarded %d experience points.", xpAward))
	}
	return &CommandResult{
		Messages:      msgs,
		RoomBroadcast: []string{fmt.Sprintf("%s finishes crafting %s!", broadcastMsg, itemName)},
		PlayerState:   player,
	}
}

// doWorkJewelry handles the WORK cycle for Jeweler items (3 steps after CRAFT).
func (e *GameEngine) doWorkJewelry(ctx context.Context, player *Player, args []string) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil || (!containsModifier(room.Modifiers, "FORGE") && !containsModifier(room.Modifiers, "LOOM")) {
		return &CommandResult{Messages: []string{"You need to be at a forge or jeweler's workshop to work jewelry."}}
	}
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You are still working... %.0f seconds remaining.", remaining+0.5)}}
	}

	switch player.CraftingStep {
	case 1:
		if len(args) == 0 {
			return &CommandResult{Messages: []string{fmt.Sprintf("Work with what material to craft your %s? e.g., WORK GOLD", player.CraftingItem)}}
		}
		target := strings.ToLower(strings.Join(args, " "))
		target, _ = parseOrdinal(target)

		idx, matItem, matDef, errMsg := e.findCraftMaterial(player, target, 8)
		if idx < 0 {
			return &CommandResult{Messages: []string{errMsg}}
		}

		adj1, adj2, adj3 := buildCraftAdjs(matItem, matDef)
		player.Inventory = append(player.Inventory[:idx], player.Inventory[idx+1:]...)
		finishedAdjs := canonicalizeOilyAdjs([3]int{adj1, adj2, adj3})
		player.CraftingAdj1 = finishedAdjs[0]
		player.CraftingAdj2 = finishedAdjs[1]
		player.CraftingAdj3 = finishedAdjs[2]
		player.CraftingMetal = target
		player.CraftingStep = 2

		rt := applyRoundTime(player, 15)
		player.RoundTimeExpiry = time.Now().Add(time.Duration(rt) * time.Second)
		player.RoundTime = rt
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You carefully work the %s, shaping it into the base form of the %s.", target, player.CraftingItem), fmt.Sprintf("[Round: %d sec]", rt)},
			RoomBroadcast: []string{fmt.Sprintf("%s works diligently at the workshop.", player.FirstName)},
			PlayerState:   player,
		}

	case 2:
		player.CraftingStep = 3
		rt := applyRoundTime(player, 15)
		player.RoundTimeExpiry = time.Now().Add(time.Duration(rt) * time.Second)
		player.RoundTime = rt
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You use fine jeweler's tools to engrave and refine the %s, adding intricate detail.", player.CraftingItem), fmt.Sprintf("[Round: %d sec]", rt)},
			RoomBroadcast: []string{fmt.Sprintf("%s works diligently at the workshop.", player.FirstName)},
			PlayerState:   player,
		}

	case 3:
		mat, item := player.CraftingMetal, player.CraftingItem
		return e.completeCraft(ctx, player,
			fmt.Sprintf("You carefully polish the %s %s to a gleaming shine. Your work is complete!", mat, item),
			player.FirstName)
	}

	player.CraftingStep = 0
	player.CraftingItem = ""
	player.CraftingSkill = ""
	player.CraftingMetal = ""
	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: []string{"Your crafting state was invalid. It has been reset."}}
}

// doWorkWeaving handles the WORK cycle for Dyeing/Weaving cloth items (3 steps after CRAFT).
func (e *GameEngine) doWorkWeaving(ctx context.Context, player *Player, args []string) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil || (!containsModifier(room.Modifiers, "LOOM") && !containsModifier(room.Modifiers, "FORGE")) {
		return &CommandResult{Messages: []string{"You need to be at a loom to work cloth."}}
	}
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You are still working... %.0f seconds remaining.", remaining+0.5)}}
	}

	switch player.CraftingStep {
	case 1:
		if len(args) == 0 {
			return &CommandResult{Messages: []string{fmt.Sprintf("Work with what material to craft your %s? e.g., WORK HIDE", player.CraftingItem)}}
		}
		target := strings.ToLower(strings.Join(args, " "))
		target, _ = parseOrdinal(target)

		idx, matItem, matDef, errMsg := e.findCraftMaterial(player, target, 15)
		if idx < 0 {
			return &CommandResult{Messages: []string{errMsg}}
		}

		adj1, adj2, adj3 := buildWeaveCraftAdjs(matItem, matDef)
		player.Inventory = append(player.Inventory[:idx], player.Inventory[idx+1:]...)
		player.CraftingAdj1 = adj1
		player.CraftingAdj2 = adj2
		player.CraftingAdj3 = adj3
		player.CraftingMetal = target
		player.CraftingStep = 2

		rt := applyRoundTime(player, 15)
		player.RoundTimeExpiry = time.Now().Add(time.Duration(rt) * time.Second)
		player.RoundTime = rt
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You mount the %s on the loom and begin weaving it into shape for the %s.", target, player.CraftingItem), fmt.Sprintf("[Round: %d sec]", rt)},
			RoomBroadcast: []string{fmt.Sprintf("%s works diligently at the loom.", player.FirstName)},
			PlayerState:   player,
		}

	case 2:
		player.CraftingStep = 3
		rt := applyRoundTime(player, 15)
		player.RoundTimeExpiry = time.Now().Add(time.Duration(rt) * time.Second)
		player.RoundTime = rt
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You carefully cut and stitch the fabric, shaping it into the form of the %s.", player.CraftingItem), fmt.Sprintf("[Round: %d sec]", rt)},
			RoomBroadcast: []string{fmt.Sprintf("%s works diligently at the loom.", player.FirstName)},
			PlayerState:   player,
		}

	case 3:
		mat, item := player.CraftingMetal, player.CraftingItem
		return e.completeCraft(ctx, player,
			fmt.Sprintf("You add finishing touches, hemming the edges neatly. Your %s %s is complete!", mat, item),
			player.FirstName)
	}

	player.CraftingStep = 0
	player.CraftingItem = ""
	player.CraftingSkill = ""
	player.CraftingMetal = ""
	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: []string{"Your crafting state was invalid. It has been reset."}}
}

// doWorkWood handles the WORK cycle for Wood Lore items (3 steps after CRAFT).
func (e *GameEngine) doWorkWood(ctx context.Context, player *Player, args []string) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil || !containsModifier(room.Modifiers, "FLETCHER") {
		return &CommandResult{Messages: []string{"You need to be at a fletcher's workshop to work wood."}}
	}
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You are still working... %.0f seconds remaining.", remaining+0.5)}}
	}

	switch player.CraftingStep {
	case 1:
		if len(args) == 0 {
			return &CommandResult{Messages: []string{fmt.Sprintf("Work with what material to craft your %s? e.g., WORK BRANCH", player.CraftingItem)}}
		}
		target := strings.ToLower(strings.Join(args, " "))
		target, _ = parseOrdinal(target)

		idx, matItem, matDef, errMsg := e.findCraftMaterial(player, target, 18)
		if idx < 0 {
			return &CommandResult{Messages: []string{errMsg}}
		}

		adj1, adj2, adj3 := buildCraftAdjs(matItem, matDef)
		player.Inventory = append(player.Inventory[:idx], player.Inventory[idx+1:]...)
		player.CraftingAdj1 = adj1
		player.CraftingAdj2 = adj2
		player.CraftingAdj3 = adj3
		player.CraftingMetal = target
		player.CraftingStep = 2

		rt := applyRoundTime(player, 15)
		player.RoundTimeExpiry = time.Now().Add(time.Duration(rt) * time.Second)
		player.RoundTime = rt
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You begin carving the %s, shaping it roughly into the form of a %s.", target, player.CraftingItem), fmt.Sprintf("[Round: %d sec]", rt)},
			RoomBroadcast: []string{fmt.Sprintf("%s works diligently at the fletcher's workshop.", player.FirstName)},
			PlayerState:   player,
		}

	case 2:
		player.CraftingStep = 3
		rt := applyRoundTime(player, 15)
		player.RoundTimeExpiry = time.Now().Add(time.Duration(rt) * time.Second)
		player.RoundTime = rt
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You sand and smooth the %s, refining its shape and removing rough edges.", player.CraftingItem), fmt.Sprintf("[Round: %d sec]", rt)},
			RoomBroadcast: []string{fmt.Sprintf("%s works diligently at the fletcher's workshop.", player.FirstName)},
			PlayerState:   player,
		}

	case 3:
		mat, item := player.CraftingMetal, player.CraftingItem
		return e.completeCraft(ctx, player,
			fmt.Sprintf("You apply finishing oil, bringing out the natural grain of the wood. Your %s %s is complete!", mat, item),
			player.FirstName)
	}

	player.CraftingStep = 0
	player.CraftingItem = ""
	player.CraftingSkill = ""
	player.CraftingMetal = ""
	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: []string{"Your crafting state was invalid. It has been reset."}}
}

// doAdviceCrafting gives crafting tips tailored to the player's current room.
// Away from a workshop, it points the player toward the known crafting guilds.
// At a forge, loom, or fletcher's bench, a master crafter steps up with advice
// specific to that trade.
func (e *GameEngine) doAdviceCrafting(player *Player) *CommandResult {
	room := e.rooms[player.RoomNumber]
	var mods []string
	if room != nil {
		mods = room.Modifiers
	}
	hasForge := containsModifier(mods, "FORGE")
	hasLoom := containsModifier(mods, "LOOM")
	hasFletcher := containsModifier(mods, "FLETCHER")
	isMiningShop := player.RoomNumber == 394

	if !hasForge && !hasLoom && !hasFletcher && !isMiningShop {
		return &CommandResult{Messages: []string{
			"You aren't near any crafting workshops right now. Master crafters can be found at:",
			"- The Foundry, southwest of the Physicians Guild — smithing: smelting ore, forging weapons and armor, repairs",
			"- The Crafter's Guild, southwest of the Adventurers' Guild — Weaver Workshop and Jeweler Workshop under one roof",
			"- The Bowyer & Fletcher shop, on First Street — Wood Lore and fletching",
			"- New Havarth Mining Company, in Bazaar North — train Mining and buy your ore-working supplies",
			"Foraging isn't tied to a guild — head out to forests, mountains, plains, swamps, or jungles with Wood Lore trained.",
			"Alchemy needs no workshop either — BREW works with any vial, bottle, or flask.",
			"Visit one of these and try ADVICE CRAFTING again for advice specific to that trade.",
		}}
	}

	var msgs []string
	if isMiningShop {
		msgs = append(msgs,
			"The clerk looks up from his ledger and offers some pointers:",
			"- MINE requires a mining tool, like a pickaxe or miner's hammer, and some training in Mining",
			"- Ore has to be carried to a forge and SMELTed before you can work it into anything",
			"- Higher purity ore smelts more reliably — the deeper and tougher the vein, the better the yield",
		)
	}
	if hasForge {
		msgs = append(msgs,
			"A master smith looks up from the forge, wiping soot from her hands, and offers some pointers:",
			"- SMELT ore into metal bars before you can forge anything with it",
			"- CRAFT <item> to start shaping metal into a weapon or piece of armor",
			"- WORK <material> to continue a piece you've already started",
			"- REPAIR patches up damaged weapons and armor",
			"- Truesteel and other rare metals are trickier to quench than iron or bronze — expect more failures",
		)
	}
	if hasLoom {
		msgs = append(msgs,
			"A master weaver glances over from the loom and offers some pointers:",
			"- WORK PELT or WORK HIDE prepares skinned materials before you weave or craft with them",
			"- CRAFT <item> at the loom to weave cloth and leather goods",
			"- DYE lets you recolor a finished cloth item",
		)
	}
	if hasFletcher {
		msgs = append(msgs,
			"The old bowyer sets down his work and offers some pointers:",
			"- This shop is where you train Wood Lore, the skill behind foraging in the wild",
			"- CRAFT <item> here to start shaping wood, then WORK <wood name> (e.g. teak, rosewood) to continue it",
		)
	}
	if hasForge || hasLoom {
		msgs = append(msgs, "- If you've trained as a Jeweler, this workshop doubles as a bench for ENCRUST, INLAY, INSET, and ENGRAVE")
	}
	return &CommandResult{Messages: msgs}
}
