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

// weaponSharpnessBonus computes the non-magical to-hit bonus (Val1) for a weapon
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

// replaceOilyAdj swaps adjective 221 (oily) or 684 (oiled) to 752 (iridescent).
// Oily/oiled metals must not carry elemental crit properties into crafted items.
func replaceOilyAdj(adj int) int {
	if adj == 221 || adj == 684 {
		return 752
	}
	return adj
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
		// Oily/oiled metals become iridescent in the finished piece.
		item := InventoryItem{
			Archetype: def.Number,
			Adj1:      replaceOilyAdj(craftAdj(0)),
			Adj2:      replaceOilyAdj(craftAdj(1)),
			Adj3:      replaceOilyAdj(craftAdj(2)),
		}
		player.Inventory = append(player.Inventory, item)

		// XP award: scale by skill level required (weaponsmithing uses metalDifficulty instead).
		xpAward := 0
		if skillID != 8 && def.Parameter2 > 0 {
			xpAward = def.Parameter2 * 20
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
				}
				sc := e.RunPreverbScripts(player, room, "WORK", &tempRI, itemDef)
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
		// smelt→forge pipeline. Convert it now to a non-magical to-hit bonus (Val1).
		// Val2 on the finished weapon means magical enchantment, so it must be zeroed.
		smithSkill := player.Skills[8]
		sharpness := weaponSharpnessBonus(player.CraftingVal2, smithSkill)
		val3 := player.CraftingVal3
		rAdj1 := replaceOilyAdj(adj1)
		rAdj2 := replaceOilyAdj(player.CraftingAdj2)
		rAdj3 := replaceOilyAdj(player.CraftingAdj3)
		if rAdj1 != adj1 || rAdj2 != player.CraftingAdj2 || rAdj3 != player.CraftingAdj3 {
			val3 = 0
		}
		item := InventoryItem{
			Archetype: weaponDef.Number,
			Adj1:      rAdj1,
			Adj2:      rAdj2,
			Adj3:      rAdj3,
			Val1:      sharpness, // non-magical quality bonus
			Val2:      0,         // no magical enchantment from forging
			Val3:      val3,      // elemental crit type (from ore); 0 if oily/oiled
			Val4:      player.CraftingVal4,
			Val5:      player.CraftingVal5,
		}
		player.Inventory = append(player.Inventory, item)

		// Award XP: 25 per skill level required + metal quality bonus + sharpness bonus
		baseSkill := weaponDef.Parameter1
		if baseSkill < 1 {
			baseSkill = 1
		}
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

	// Find the weapon in inventory with DAMAGED state
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
			return &CommandResult{Messages: []string{"That doesn't need repair."}}
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
			dyeRT := applyRoundTime(player, 15)
			player.RoundTimeExpiry = time.Now().Add(time.Duration(dyeRT) * time.Second)
			player.RoundTime = dyeRT
			e.SavePlayer(ctx, player)

			dyedName := e.formatItemName(targetDef, targetItem.Adj1, targetItem.Adj2, targetItem.Adj3, targetItem.Tail)
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
			itemName := e.formatItemName(def, ii.Adj1, ii.Adj2, ii.Adj3, ii.Tail)
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

// ---- ENCRUST ----

func (e *GameEngine) doEncrust(ctx context.Context, player *Player, args []string) *CommandResult {
	if player.Skills[0] < 3 {
		return &CommandResult{Messages: []string{"You need Jeweler skill level 3 to encrust items."}}
	}
	room := e.rooms[player.RoomNumber]
	if room == nil || (!containsModifier(room.Modifiers, "FORGE") && !containsModifier(room.Modifiers, "LOOM")) {
		return &CommandResult{Messages: []string{"You need to be at a forge or jeweler's workshop to encrust items."}}
	}
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You must wait %.0f more seconds.", remaining)}}
	}
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Encrust what? Usage: ENCRUST <item> WITH <gem>"}}
	}

	raw := strings.ToLower(strings.Join(args, " "))
	itemTarget, gemTarget := parseWithClause(raw)
	if gemTarget == "" {
		return &CommandResult{Messages: []string{"Encrust with what gem? Usage: ENCRUST <item> WITH <gem>"}}
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
		return &CommandResult{Messages: []string{"That item already has too many adjectives to encrust with a gem."}}
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
	encrustedAdjID := e.adjByName("encrusted")
	if encrustedAdjID == 0 {
		encrustedAdjID = 114
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

	// Apply: Adj1=gem noun, Adj2=encrusted, Adj3=existing
	newAdj1, newAdj2, newAdj3 := gemAdjID, encrustedAdjID, existingAdj
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

// findCraftMaterial searches inventory for the first item whose noun or adjective
// starts with target and checks whether it is a valid crafting material for the
// given skill ID (8=weaponsmithing/jeweler, 15=weaving, 18=wood).  Returns the
// inventory index, item snapshot, and item def on success.  On failure it returns
// idx=-1 and a human-readable error message.
func (e *GameEngine) findCraftMaterial(player *Player, target string, matSkillID int) (idx int, item InventoryItem, def *gameworld.ItemDef, errMsg string) {
	foundNoun := false
	for j, ii := range player.Inventory {
		mDef := e.items[ii.Archetype]
		if mDef == nil {
			continue
		}
		noun := strings.ToLower(e.getItemNounName(mDef))
		adj1 := strings.ToLower(e.getAdjName(ii.Adj1))
		adj2 := strings.ToLower(e.getAdjName(ii.Adj2))
		adj3 := strings.ToLower(e.getAdjName(ii.Adj3))
		matches := strings.HasPrefix(noun, target) ||
			(adj1 != "" && strings.HasPrefix(adj1, target)) ||
			(adj2 != "" && strings.HasPrefix(adj2, target)) ||
			(adj3 != "" && strings.HasPrefix(adj3, target))
		if !matches {
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
			return -1, InventoryItem{}, nil, fmt.Sprintf("That %s is not suitable material for that item.", target)
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

	xpAward := 0
	if craftDef.Parameter2 > 0 {
		xpAward = craftDef.Parameter2 * 20
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
		player.CraftingAdj1 = replaceOilyAdj(adj1)
		player.CraftingAdj2 = replaceOilyAdj(adj2)
		player.CraftingAdj3 = replaceOilyAdj(adj3)
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
