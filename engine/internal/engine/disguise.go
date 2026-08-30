package engine

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// disguiseFieldLevel maps each disguise field name to the Disguise skill (34)
// rank that unlocks it. Race requires the most training (and a custom name
// requires more than that), height/weight slightly less — per the design
// order the user specified.
var disguiseFieldLevel = map[string]int{
	"name":      1,
	"gender":    2,
	"haircolor": 3,
	"hairstyle": 3,
	"skincolor": 4,
	"eyecolor":  4,
	"age":       5,
	"strength":  6,
	"height":    7,
	"weight":    8,
	"race":      9,
}

// disguiseFieldOrder is disguiseFieldLevel's keys in display order for the
// bare "disguise" instructions and "disguise list <#>" detail view.
var disguiseFieldOrder = []string{
	"name", "gender", "haircolor", "hairstyle", "skincolor", "eyecolor",
	"age", "strength", "height", "weight", "race",
}

// disguiseCommonNames are the generic town NPC types a disguised player can
// pass as (see disguisableNPCNames in monsters.go, which grants those NPCs
// full descriptions to blend into). Below Disguise rank 10, "name" must be
// one of these; at rank 10+ one of these is still always treated as the NPC
// type it is (see setDisguiseName) rather than becoming a custom persona.
var disguiseCommonNames = []string{"commoner", "trader", "merchant", "lawkeeper", "beggar", "priest"}

// disguiseSlotCount returns how many save slots a Disguise rank grants: one
// slot every 2 levels, minimum 1 (once trained at all), maximum 5.
func disguiseSlotCount(level int) int {
	if level <= 0 {
		return 0
	}
	return min((level+1)/2, 5)
}

func (e *GameEngine) doDisguise(ctx context.Context, player *Player, args []string) *CommandResult {
	level := player.Skills[34]
	if level <= 0 {
		return &CommandResult{Messages: []string{"You have no training in the Disguise skill."}}
	}

	if len(args) == 0 {
		return e.disguiseInstructions(level)
	}

	switch strings.ToLower(args[0]) {
	case "list":
		if len(args) >= 2 {
			return e.disguiseListSlot(player, level, args[1])
		}
		return e.disguiseListAll(player, level)
	case "apply":
		if len(args) < 2 {
			return &CommandResult{Messages: []string{"Apply which slot? (DISGUISE APPLY <#>)"}}
		}
		return e.disguiseApply(ctx, player, level, args[1])
	case "remove":
		return e.disguiseRemove(ctx, player)
	case "clear":
		if len(args) < 2 {
			return &CommandResult{Messages: []string{"Clear which slot? (DISGUISE CLEAR <#>)"}}
		}
		return e.disguiseClear(ctx, player, level, args[1])
	}

	// disguise <slot> <field> <value...>
	slot, err := strconv.Atoi(args[0])
	if err != nil {
		return &CommandResult{Messages: []string{"Usage: DISGUISE <slot#> <field> <value>, DISGUISE APPLY <slot#>, DISGUISE REMOVE, DISGUISE CLEAR <slot#>, or DISGUISE LIST."}}
	}
	maxSlots := disguiseSlotCount(level)
	if slot < 1 || slot > maxSlots {
		return &CommandResult{Messages: []string{fmt.Sprintf("Your Disguise skill only grants you %d save slot(s).", maxSlots)}}
	}
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Set which field? (name, gender, haircolor, hairstyle, skincolor, eyecolor, age, strength, height, weight, race)"}}
	}
	field := strings.ToLower(args[1])
	requiredLevel, ok := disguiseFieldLevel[field]
	if !ok {
		return &CommandResult{Messages: []string{fmt.Sprintf("Unknown disguise field %q.", field)}}
	}
	if level < requiredLevel {
		return &CommandResult{Messages: []string{fmt.Sprintf("Your Disguise skill isn't advanced enough to change %s yet. (need rank %d, have %d)", field, requiredLevel, level)}}
	}
	if len(args) < 3 {
		if field == "name" {
			return &CommandResult{Messages: []string{fmt.Sprintf("Set the name to what? You can use a basic persona (%s)%s.", strings.Join(disguiseCommonNames, ", "), func() string {
				if level >= 10 {
					return ", or any custom first/last name"
				}
				return ""
			}())}}
		}
		return &CommandResult{Messages: []string{fmt.Sprintf("Set %s to what?", field)}}
	}
	persona := player.DisguiseSlots[slot]
	if field == "name" {
		if err := setDisguiseName(&persona, args[2:], level); err != nil {
			return &CommandResult{Messages: []string{err.Error()}}
		}
	} else if err := setDisguiseField(&persona, field, strings.Join(args[2:], " ")); err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	if player.DisguiseSlots == nil {
		player.DisguiseSlots = make(map[int]DisguisePersona)
	}
	player.DisguiseSlots[slot] = persona
	e.SavePlayer(ctx, player)
	return &CommandResult{
		Messages:    []string{fmt.Sprintf("Disguise slot %d: %s set.", slot, field)},
		PlayerState: player,
	}
}

// setDisguiseName validates and stores the "name" field. A single word
// matching one of disguiseCommonNames is always treated as that generic NPC
// type — lowercase, no last name, no article/capitalization of a proper name
// — regardless of rank: rank 10 unlocks a CUSTOM name as an additional
// option, it doesn't stop "commoner" from meaning the generic commoner. Any
// other name requires rank 10, and becomes a title-cased "First [Last]"
// persona (e.g. "disguise 1 name Tania Chanlin" -> Name="Tania",
// LastName="Chanlin") regardless of how the player typed the casing.
func setDisguiseName(persona *DisguisePersona, words []string, level int) error {
	if len(words) == 0 {
		return fmt.Errorf("set the name to what?")
	}
	if len(words) == 1 && containsString(disguiseCommonNames, strings.ToLower(words[0])) {
		persona.Name = strings.ToLower(words[0])
		persona.LastName = ""
		return nil
	}
	if level < 10 {
		return fmt.Errorf("until Disguise rank 10, your name must be one of: %s", strings.Join(disguiseCommonNames, ", "))
	}
	first := words[0]
	last := strings.Join(words[1:], " ")
	if !validDisguiseName(first) || (last != "" && !validDisguiseName(last)) {
		return fmt.Errorf("that's not a valid name (letters, spaces, apostrophes and hyphens only, 2-30 characters per part)")
	}
	persona.Name = titleCase(first)
	persona.LastName = titleCase(last)
	return nil
}

// titleCase capitalizes the first letter of each space-separated word and
// lowercases the rest, so a name typed in any case ("TANIA CHANLIN", "tania
// chanlin") normalizes to "Tania Chanlin".
func titleCase(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		words[i] = capitalize(w)
	}
	return strings.Join(words, " ")
}

// setDisguiseField validates value for field and, if valid, writes it into persona.
func setDisguiseField(persona *DisguisePersona, field, value string) error {
	lower := strings.ToLower(value)
	switch field {
	case "gender":
		if lower != "male" && lower != "female" {
			return fmt.Errorf("gender must be male or female")
		}
		persona.Gender = lower
	case "haircolor":
		if !containsString(HairColors, lower) {
			return fmt.Errorf("hair color must be one of: %s", strings.Join(HairColors, ", "))
		}
		persona.HairColor = lower
	case "hairstyle":
		if !containsString(HairStyles, lower) {
			return fmt.Errorf("hair style must be one of: %s", strings.Join(HairStyles, ", "))
		}
		persona.HairStyle = lower
	case "skincolor":
		if !containsString(SkinColors, lower) {
			return fmt.Errorf("skin color must be one of: %s", strings.Join(SkinColors, ", "))
		}
		persona.SkinColor = lower
	case "eyecolor":
		if !containsString(EyeColors, lower) {
			return fmt.Errorf("eye color must be one of: %s", strings.Join(EyeColors, ", "))
		}
		persona.EyeColor = lower
	case "age":
		n, err := strconv.Atoi(value)
		if err != nil || n < 16 || n > 900 {
			return fmt.Errorf("age must be a number between 16 and 900")
		}
		persona.Age = n
	case "strength":
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 130 {
			return fmt.Errorf("strength must be a number between 1 and 130")
		}
		persona.Strength = n
	case "height":
		n, err := strconv.Atoi(value)
		if err != nil || n < 40 || n > 100 {
			return fmt.Errorf("height must be a number of inches between 40 and 100")
		}
		persona.Height = n
	case "weight":
		n, err := strconv.Atoi(value)
		if err != nil || n < 60 || n > 400 {
			return fmt.Errorf("weight must be a number of pounds between 60 and 400")
		}
		persona.Weight = n
	case "race":
		for _, r := range PlayableRaces {
			if strings.ToLower(RaceNameByID(r)) == lower {
				persona.Race = r
				return nil
			}
		}
		return fmt.Errorf("race must be one of: %s", raceNameList())
	}
	return nil
}

func validDisguiseName(name string) bool {
	if len(name) < 2 || len(name) > 30 {
		return false
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && r != ' ' && r != '\'' && r != '-' {
			return false
		}
	}
	return true
}

func raceNameList() string {
	names := make([]string, len(PlayableRaces))
	for i, r := range PlayableRaces {
		names[i] = RaceNameByID(r)
	}
	return strings.Join(names, ", ")
}

func (e *GameEngine) disguiseApply(ctx context.Context, player *Player, level int, slotArg string) *CommandResult {
	slot, err := strconv.Atoi(slotArg)
	if err != nil {
		return &CommandResult{Messages: []string{"Apply which slot number?"}}
	}
	maxSlots := disguiseSlotCount(level)
	if slot < 1 || slot > maxSlots {
		return &CommandResult{Messages: []string{fmt.Sprintf("Your Disguise skill only grants you %d save slot(s).", maxSlots)}}
	}
	persona, ok := player.DisguiseSlots[slot]
	if !ok {
		return &CommandResult{Messages: []string{fmt.Sprintf("Disguise slot %d is empty.", slot)}}
	}
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You must wait %.0f more seconds.", remaining)}}
	}
	player.ActiveDisguise = persona
	player.Disguised = true
	rt := applyRoundTime(player, 30)
	player.RoundTimeExpiry = time.Now().Add(time.Duration(rt) * time.Second)
	player.RoundTime = rt
	e.SavePlayer(ctx, player)
	return &CommandResult{
		Messages:    []string{fmt.Sprintf("You disguise yourself as %s.", player.DisplayFullName()), fmt.Sprintf("[Round: %d sec]", rt)},
		PlayerState: player,
	}
}

func (e *GameEngine) disguiseRemove(ctx context.Context, player *Player) *CommandResult {
	if !player.Disguised {
		return &CommandResult{Messages: []string{"You are not disguised."}}
	}
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := time.Until(player.RoundTimeExpiry).Seconds()
		return &CommandResult{Messages: []string{fmt.Sprintf("You must wait %.0f more seconds.", remaining)}}
	}
	player.Disguised = false
	player.ActiveDisguise = DisguisePersona{}
	rt := applyRoundTime(player, 15)
	player.RoundTimeExpiry = time.Now().Add(time.Duration(rt) * time.Second)
	player.RoundTime = rt
	e.SavePlayer(ctx, player)
	return &CommandResult{
		Messages:    []string{"You remove your disguise, revealing your true self.", fmt.Sprintf("[Round: %d sec]", rt)},
		PlayerState: player,
	}
}

func (e *GameEngine) disguiseClear(ctx context.Context, player *Player, level int, slotArg string) *CommandResult {
	slot, err := strconv.Atoi(slotArg)
	if err != nil {
		return &CommandResult{Messages: []string{"Clear which slot number?"}}
	}
	maxSlots := disguiseSlotCount(level)
	if slot < 1 || slot > maxSlots {
		return &CommandResult{Messages: []string{fmt.Sprintf("Your Disguise skill only grants you %d save slot(s).", maxSlots)}}
	}
	if _, ok := player.DisguiseSlots[slot]; !ok {
		return &CommandResult{Messages: []string{fmt.Sprintf("Disguise slot %d is already empty.", slot)}}
	}
	delete(player.DisguiseSlots, slot)
	e.SavePlayer(ctx, player)
	return &CommandResult{
		Messages:    []string{fmt.Sprintf("Disguise slot %d cleared.", slot)},
		PlayerState: player,
	}
}

func (e *GameEngine) disguiseListAll(player *Player, level int) *CommandResult {
	maxSlots := disguiseSlotCount(level)
	msgs := []string{fmt.Sprintf("%-4s %-16s %-12s %-6s %s", "Slot", "Name", "Race", "Age", "Gender")}
	any := false
	for slot := 1; slot <= maxSlots; slot++ {
		p, ok := player.DisguiseSlots[slot]
		if !ok {
			continue
		}
		any = true
		name, race, age, gender := "(your own)", "(your own)", "(your own)", "(your own)"
		if p.Name != "" {
			name = p.Name
			if p.LastName != "" {
				name += " " + p.LastName
			}
		}
		if p.Race != 0 {
			race = RaceNameByID(p.Race)
		}
		if p.Age != 0 {
			age = strconv.Itoa(p.Age)
		}
		if p.Gender != "" {
			gender = capitalize(p.Gender)
		}
		msgs = append(msgs, fmt.Sprintf("%-4d %-16s %-12s %-6s %s", slot, name, race, age, gender))
	}
	if !any {
		msgs = append(msgs, fmt.Sprintf("You have no saved disguises yet. (%d slot(s) available)", maxSlots))
	}
	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) disguiseListSlot(player *Player, level int, slotArg string) *CommandResult {
	slot, err := strconv.Atoi(slotArg)
	if err != nil {
		return &CommandResult{Messages: []string{"List which slot number?"}}
	}
	maxSlots := disguiseSlotCount(level)
	if slot < 1 || slot > maxSlots {
		return &CommandResult{Messages: []string{fmt.Sprintf("Your Disguise skill only grants you %d save slot(s).", maxSlots)}}
	}
	p, ok := player.DisguiseSlots[slot]
	if !ok {
		return &CommandResult{Messages: []string{fmt.Sprintf("Disguise slot %d is empty.", slot)}}
	}
	field := func(label, val string) string {
		if val == "" {
			val = "(your own)"
		}
		return fmt.Sprintf("  %-10s %s", label+":", val)
	}
	race, age, strength, height, weight := "", "", "", "", ""
	if p.Race != 0 {
		race = RaceNameByID(p.Race)
	}
	if p.Age != 0 {
		age = strconv.Itoa(p.Age)
	}
	if p.Strength != 0 {
		strength = strconv.Itoa(p.Strength)
	}
	if p.Height != 0 {
		height = strconv.Itoa(p.Height)
	}
	if p.Weight != 0 {
		weight = strconv.Itoa(p.Weight)
	}
	name := p.Name
	if p.LastName != "" {
		name += " " + p.LastName
	}
	msgs := []string{fmt.Sprintf("Disguise slot %d:", slot)}
	msgs = append(msgs,
		field("Name", name), field("Gender", capitalize(p.Gender)), field("Race", race),
		field("Age", age), field("Strength", strength), field("Height", height), field("Weight", weight),
		field("Hairstyle", p.HairStyle), field("Haircolor", p.HairColor),
		field("Skin", p.SkinColor), field("Eyes", p.EyeColor),
	)
	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) disguiseInstructions(level int) *CommandResult {
	msgs := []string{
		fmt.Sprintf("Disguise (rank %d): DISGUISE <slot> <field> <value> to compose a disguise, DISGUISE APPLY <slot> to wear it, DISGUISE REMOVE to drop it, DISGUISE CLEAR <slot> to reset a slot to defaults, DISGUISE LIST to see your saved slots.", level),
		fmt.Sprintf("You have %d save slot(s).", disguiseSlotCount(level)),
		"Fields you can currently set:",
	}
	for _, f := range disguiseFieldOrder {
		if level >= disguiseFieldLevel[f] {
			msgs = append(msgs, fmt.Sprintf("  %s (rank %d)", f, disguiseFieldLevel[f]))
		}
	}
	msgs = append(msgs, fmt.Sprintf("Your disguise name can always be a basic persona: %s.", strings.Join(disguiseCommonNames, ", ")))
	if level < 10 {
		msgs = append(msgs, "At rank 10, you'll also be able to use a custom first/last name.")
	} else {
		msgs = append(msgs, "At rank 10+, you can also use a custom first/last name.")
	}
	return &CommandResult{Messages: msgs}
}
