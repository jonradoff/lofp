package engine

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jonradoff/lofp/internal/gameworld"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// CreateNewPlayer creates a new player character and saves it to MongoDB.
func (e *GameEngine) CreateNewPlayer(ctx context.Context, firstName, lastName string, race, gender int, accountID ...string) *Player {
	ranges := RaceStatRanges[race]
	rollStat := func(idx int) int {
		r := ranges[idx]
		return r[0] + rand.Intn(r[1]-r[0]+1)
	}

	str := rollStat(0)
	agi := rollStat(1)
	qui := rollStat(2)
	con := rollStat(3)
	per := rollStat(4)
	wil := rollStat(5)
	emp := rollStat(6)

	bodyPts := 20 + con/2
	fatigue := 20 + (con+str)/3
	mana := emp / 2
	psi := wil / 2

	heightWeightRanges := map[int][4]int{
		1: {62, 76, 120, 220},
		2: {66, 80, 100, 170},
		3: {48, 58, 130, 200},
		4: {64, 74, 130, 200},
		5: {62, 74, 150, 230},
		6: {68, 82, 150, 250},
		7: {60, 74, 150, 250},
		8: {58, 72, 80, 130},
	}
	hw := heightWeightRanges[race]
	if hw == [4]int{} {
		hw = [4]int{62, 76, 120, 220}
	}
	height := hw[0] + rand.Intn(hw[1]-hw[0]+1)
	weight := hw[2] + rand.Intn(hw[3]-hw[2]+1)
	if gender == 1 {
		height -= 2 + rand.Intn(3)
		weight -= 10 + rand.Intn(20)
		if height < hw[0]-4 {
			height = hw[0] - 4
		}
		if weight < hw[2]-20 {
			weight = hw[2] - 20
		}
	}

	now := time.Now()
	player := &Player{
		FirstName:        firstName,
		LastName:         lastName,
		Race:             race,
		Gender:           gender,
		Level:            1,
		BuildPoints:      30,
		Strength:         str,
		Agility:          agi,
		Quickness:        qui,
		Constitution:     con,
		Perception:       per,
		Willpower:        wil,
		Empathy:          emp,
		BodyPoints:       bodyPts,
		MaxBodyPoints:    bodyPts,
		Fatigue:          fatigue,
		MaxFatigue:       fatigue,
		Mana:             mana,
		MaxMana:          mana,
		Psi:              psi,
		MaxPsi:           psi,
		Height:           height,
		HeightTrue:       height,
		Weight:           weight,
		WeightTrue:       weight,
		RoomNumber:       201,
		Position:         0,
		Skills:           make(map[int]int),
		IntNums:          make(map[int]int),
		Gold:             5,
		Silver:           10,
		Copper:           50,
		PromptMode:       true,
		SuppressLogon:    true,
		SuppressLogoff:   true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if race == RaceEphemeral {
		player.TelepathyActive = true
	}
	if len(accountID) > 0 && accountID[0] != "" {
		player.AccountID = accountID[0]
	}

	if e.db != nil {
		coll := e.db.Collection("players")
		res, err := coll.InsertOne(ctx, player)
		if err != nil {
			log.Printf("Failed to insert player: %v", err)
		} else {
			player.ID = res.InsertedID.(bson.ObjectID)
		}
	}

	return player
}

// LoadPlayer loads a non-deleted player from MongoDB by first+last name.
func (e *GameEngine) LoadPlayer(ctx context.Context, firstName, lastName string) (*Player, error) {
	if e.db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	coll := e.db.Collection("players")
	var player Player
	filter := bson.M{"firstName": firstName, "lastName": lastName, "deletedAt": bson.M{"$exists": false}}
	err := coll.FindOne(ctx, filter).Decode(&player)
	if err != nil {
		return nil, err
	}

	if player.Height == 0 || player.Weight == 0 {
		heightWeightRanges := map[int][4]int{
			1: {62, 76, 120, 220}, 2: {66, 80, 100, 170}, 3: {48, 58, 130, 200},
			4: {64, 74, 130, 200}, 5: {62, 74, 150, 230}, 6: {68, 82, 150, 250},
			7: {60, 74, 150, 250}, 8: {58, 72, 80, 130},
		}
		hw := heightWeightRanges[player.Race]
		if hw == [4]int{} {
			hw = [4]int{62, 76, 120, 220}
		}
		if player.Height == 0 {
			h := hw[0] + rand.Intn(hw[1]-hw[0]+1)
			if player.Gender == 1 {
				h -= 2 + rand.Intn(3)
			}
			player.Height = h
			player.HeightTrue = h
		}
		if player.Weight == 0 {
			w := hw[2] + rand.Intn(hw[3]-hw[2]+1)
			if player.Gender == 1 {
				w -= 10 + rand.Intn(20)
			}
			player.Weight = w
			player.WeightTrue = w
		}
		e.SavePlayer(ctx, &player)
	}

	return &player, nil
}

// ListPlayers returns all saved characters, sorted by updatedAt descending.
func (e *GameEngine) ListPlayers(ctx context.Context) ([]Player, error) {
	if e.db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	coll := e.db.Collection("players")
	opts := options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}})
	cursor, err := coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	var players []Player
	if err := cursor.All(ctx, &players); err != nil {
		return nil, err
	}
	return players, nil
}

// ListPlayersByAccount returns all non-deleted characters belonging to an account.
func (e *GameEngine) ListPlayersByAccount(ctx context.Context, accountID string) ([]Player, error) {
	if e.db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	coll := e.db.Collection("players")
	filter := bson.M{"accountId": accountID, "deletedAt": bson.M{"$exists": false}}
	cursor, err := coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	var players []Player
	if err := cursor.All(ctx, &players); err != nil {
		return nil, err
	}
	return players, nil
}

// SoftDeletePlayer soft-deletes a character by setting deletedAt.
func (e *GameEngine) SoftDeletePlayer(ctx context.Context, firstName, accountID string) error {
	if e.db == nil {
		return fmt.Errorf("no database connection")
	}
	coll := e.db.Collection("players")
	now := time.Now()
	filter := bson.M{"firstName": firstName, "accountId": accountID, "deletedAt": bson.M{"$exists": false}}
	update := bson.M{"$set": bson.M{"deletedAt": now}}
	result, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("character not found or not owned by you")
	}
	return nil
}

// IsFirstNameTaken checks if a non-deleted character with this first name exists.
func (e *GameEngine) IsFirstNameTaken(ctx context.Context, firstName string) (bool, error) {
	if e.db == nil {
		return false, nil
	}
	coll := e.db.Collection("players")
	count, err := coll.CountDocuments(ctx, bson.M{
		"firstName": bson.M{"$regex": "^" + regexp.QuoteMeta(firstName) + "$", "$options": "i"},
		"deletedAt": bson.M{"$exists": false},
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListDeletedPlayers returns all soft-deleted characters.
func (e *GameEngine) ListDeletedPlayers(ctx context.Context) ([]Player, error) {
	if e.db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	coll := e.db.Collection("players")
	opts := options.Find().SetSort(bson.D{{Key: "deletedAt", Value: -1}})
	cursor, err := coll.Find(ctx, bson.M{"deletedAt": bson.M{"$exists": true}}, opts)
	if err != nil {
		return nil, err
	}
	var players []Player
	if err := cursor.All(ctx, &players); err != nil {
		return nil, err
	}
	return players, nil
}

// RecoverPlayer un-deletes a character, optionally renaming if name conflicts.
func (e *GameEngine) RecoverPlayer(ctx context.Context, firstName string, newFirstName string) (*Player, error) {
	if e.db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	coll := e.db.Collection("players")

	var player Player
	err := coll.FindOne(ctx, bson.M{"firstName": firstName, "deletedAt": bson.M{"$exists": true}}).Decode(&player)
	if err != nil {
		return nil, fmt.Errorf("deleted character '%s' not found", firstName)
	}

	targetName := firstName
	if newFirstName != "" {
		targetName = newFirstName
	}
	taken, _ := e.IsFirstNameTaken(ctx, targetName)
	if taken {
		return nil, fmt.Errorf("name '%s' is already taken by an active character", targetName)
	}

	update := bson.M{"$unset": bson.M{"deletedAt": ""}}
	if newFirstName != "" && newFirstName != firstName {
		update = bson.M{
			"$unset": bson.M{"deletedAt": ""},
			"$set":   bson.M{"firstName": newFirstName},
		}
	}
	_, err = coll.UpdateOne(ctx, bson.M{"_id": player.ID}, update)
	if err != nil {
		return nil, err
	}

	if newFirstName != "" {
		player.FirstName = newFirstName
	}
	player.DeletedAt = nil
	return &player, nil
}

// ReassignCharacter changes the accountId of a character to a new account.
func (e *GameEngine) ReassignCharacter(ctx context.Context, firstName, newAccountID string) (*Player, error) {
	player, err := e.resolvePlayerByName(ctx, firstName)
	if err != nil {
		return nil, err
	}
	coll := e.db.Collection("players")
	_, err = coll.UpdateOne(ctx,
		bson.M{"_id": player.ID},
		bson.M{"$set": bson.M{"accountId": newAccountID}},
	)
	if err != nil {
		return nil, err
	}
	player.AccountID = newAccountID
	return player, nil
}

// SavePlayer persists the player state to MongoDB.
func (e *GameEngine) SavePlayer(ctx context.Context, player *Player) {
	if e.db == nil {
		return
	}
	player.UpdatedAt = time.Now()
	coll := e.db.Collection("players")
	if !player.ID.IsZero() {
		_, err := coll.ReplaceOne(ctx, bson.M{"_id": player.ID}, player, options.Replace().SetUpsert(true))
		if err != nil {
			log.Printf("Failed to save player %s: %v", player.FullName(), err)
		}
	}
}

// GenerateAPIKey creates a new API key for a character. Returns the raw key (shown once).
func (e *GameEngine) GenerateAPIKey(ctx context.Context, firstName, accountID string, allowGM bool) (string, error) {
	if e.db == nil {
		return "", fmt.Errorf("no database")
	}
	raw := make([]byte, 32)
	if _, err := cryptorand.Read(raw); err != nil {
		return "", err
	}
	key := "lofp_" + hex.EncodeToString(raw)
	prefix := key[:13]

	hash := sha256.Sum256([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	coll := e.db.Collection("players")
	filter := bson.M{"firstName": firstName, "accountId": accountID, "deletedAt": bson.M{"$exists": false}}
	update := bson.M{"$set": bson.M{
		"apiKeyHash":   hashStr,
		"apiKeyPrefix": prefix,
		"botGMAllowed": allowGM,
	}}
	result, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return "", err
	}
	if result.MatchedCount == 0 {
		return "", fmt.Errorf("character not found or not owned by you")
	}
	return key, nil
}

// RevokeAPIKey removes the API key from a character.
func (e *GameEngine) RevokeAPIKey(ctx context.Context, firstName, accountID string) error {
	if e.db == nil {
		return fmt.Errorf("no database")
	}
	coll := e.db.Collection("players")
	filter := bson.M{"firstName": firstName, "accountId": accountID}
	update := bson.M{"$unset": bson.M{"apiKeyHash": "", "apiKeyPrefix": "", "botGMAllowed": ""}}
	_, err := coll.UpdateOne(ctx, filter, update)
	return err
}

// ValidateAPIKey checks an API key and returns the associated player.
func (e *GameEngine) ValidateAPIKey(ctx context.Context, key string) (*Player, error) {
	if e.db == nil {
		return nil, fmt.Errorf("no database")
	}
	hash := sha256.Sum256([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	coll := e.db.Collection("players")
	var player Player
	err := coll.FindOne(ctx, bson.M{"apiKeyHash": hashStr, "deletedAt": bson.M{"$exists": false}}).Decode(&player)
	if err != nil {
		return nil, fmt.Errorf("invalid API key")
	}
	player.IsBot = true
	return &player, nil
}

// doSet handles the SET command for toggling player settings.
func (e *GameEngine) doSet(ctx context.Context, player *Player, args []string) *CommandResult {
	onOff := func(suppressed bool) string {
		if suppressed {
			return "OFF"
		}
		return "ON"
	}
	onOffBrief := func(enabled bool) string {
		if enabled {
			return "ON"
		}
		return "OFF"
	}

	if len(args) == 1 {
		switch args[0] {
		case "ACTBRIEF":
			player.ActBrief = !player.ActBrief
			e.SavePlayer(ctx, player)
			return &CommandResult{Messages: []string{fmt.Sprintf("Actbrief is now %s.", onOffBrief(player.ActBrief))}}
		case "RPBRIEF":
			player.RPBrief = !player.RPBrief
			e.SavePlayer(ctx, player)
			return &CommandResult{Messages: []string{fmt.Sprintf("RPbrief is now %s.", onOffBrief(player.RPBrief))}}
		}
	}

	if len(args) == 0 {
		briefMode := "OFF"
		if player.BriefMode {
			briefMode = "ON"
		}
		promptMode := "ON"
		if !player.PromptMode {
			promptMode = "OFF"
		}
		lines := []string{
			"Current Settings:",
			fmt.Sprintf("  Full:                %s", func() string {
				if player.BriefMode {
					return "OFF"
				}
				return "ON"
			}()),
			fmt.Sprintf("  Brief:               %s", briefMode),
			fmt.Sprintf("  Prompt:              %s", promptMode),
			fmt.Sprintf("  Logon messages:      %s", onOff(player.SuppressLogon)),
			fmt.Sprintf("  Logoff messages:     %s", onOff(player.SuppressLogoff)),
			fmt.Sprintf("  Disconnect messages: %s", onOff(player.SuppressDisconnect)),
			fmt.Sprintf("  RPbrief:             %s", onOffBrief(player.RPBrief)),
			fmt.Sprintf("  Battlebrief:         %s", onOffBrief(player.BattleBrief)),
			fmt.Sprintf("  Actionbrief:         %s", onOffBrief(player.ActionBrief)),
			fmt.Sprintf("  Actbrief:            %s", onOffBrief(player.ActBrief)),
			"",
			"Type SET <setting> ON/OFF to change a setting.",
		}
		return &CommandResult{Messages: lines}
	}

	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Usage: SET <setting> ON/OFF"}}
	}

	setting := args[0]
	value := args[1]

	var turnOn bool
	switch value {
	case "ON", "TRUE", "YES":
		turnOn = true
	case "OFF", "FALSE", "NO":
		turnOn = false
	default:
		return &CommandResult{Messages: []string{"Usage: SET <setting> ON/OFF"}}
	}

	var msg string
	switch setting {
	case "LOGON":
		player.SuppressLogon = !turnOn
		msg = fmt.Sprintf("Logon messages are now %s.", value)
	case "LOGOFF":
		player.SuppressLogoff = !turnOn
		msg = fmt.Sprintf("Logoff messages are now %s.", value)
	case "DISCONNECT":
		player.SuppressDisconnect = !turnOn
		msg = fmt.Sprintf("Disconnect messages are now %s.", value)
	case "RPBRIEF":
		player.RPBrief = turnOn
		msg = fmt.Sprintf("RPbrief is now %s.", value)
	case "BATTLEBRIEF":
		player.BattleBrief = turnOn
		msg = fmt.Sprintf("Battlebrief is now %s.", value)
	case "ACTIONBRIEF":
		player.ActionBrief = turnOn
		msg = fmt.Sprintf("Actionbrief is now %s.", value)
	case "ACTBRIEF":
		player.ActBrief = turnOn
		msg = fmt.Sprintf("Actbrief is now %s.", value)
	case "FULL":
		player.BriefMode = !turnOn
		msg = fmt.Sprintf("Full room descriptions are now %s.", value)
	case "BRIEF":
		player.BriefMode = turnOn
		msg = fmt.Sprintf("Brief mode is now %s.", value)
	case "PROMPT":
		player.PromptMode = turnOn
		msg = fmt.Sprintf("Prompt mode is now %s.", value)
	default:
		return &CommandResult{Messages: []string{
			"Unknown setting. Valid settings: FULL, BRIEF, PROMPT, LOGON, LOGOFF, DISCONNECT, RPBRIEF, BATTLEBRIEF, ACTIONBRIEF, ACTBRIEF",
		}}
	}

	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: []string{msg}}
}

// doInitiate handles the INITIATE command — GM sets a player's organization.
func (e *GameEngine) doInitiate(ctx context.Context, player *Player, args []string) *CommandResult {
	if !player.IsGM {
		return &CommandResult{Messages: []string{"Only GMs can initiate players into organizations."}}
	}
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Usage: INITIATE <player> <org#>"}}
	}
	target, err := e.resolvePlayerArg(ctx, args[:1])
	if err != nil {
		return &CommandResult{Messages: []string{err.Error()}}
	}
	orgNum, err2 := strconv.Atoi(args[1])
	if err2 != nil || orgNum < 0 {
		return &CommandResult{Messages: []string{"Organization must be a number (0 to remove, 1-12 to set)."}}
	}
	target.Organization = orgNum
	e.SavePlayer(ctx, target)
	if orgNum == 0 {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s has been removed from their organization.", target.FullName())}}
	}
	orgName := organizationName(orgNum)
	return &CommandResult{
		Messages: []string{fmt.Sprintf("%s has been initiated into the %s (org %d).", target.FullName(), orgName, orgNum)},
	}
}

// organizationName returns the display name for an organization number.
func organizationName(org int) string {
	names := map[int]string{
		1:  "Adventurer's Guild",
		2:  "Order of Paladins",
		3:  "Mage's Guild",
		4:  "Thieves' Guild",
		5:  "Church of Gaea",
		6:  "Church of Finvarra",
		7:  "Church of Arawn",
		8:  "Church of Duach",
		9:  "Order of Rangers",
		10: "Order of Druids",
		11: "Church of Brigit",
		12: "Order of Bards",
	}
	if n, ok := names[org]; ok {
		return n
	}
	return ""
}

// playerBPSpent calculates total build points spent on skills.
func playerBPSpent(player *Player) int {
	total := 0
	for skillID, rank := range player.Skills {
		for r := 0; r < rank; r++ {
			total += skillBPCost(skillID, r)
		}
	}
	return total
}

// xpUntilNextBuildPoint returns remaining XP needed for the next BP.
func xpUntilNextBuildPoint(player *Player) int {
	rate := getXPPerBP(player.Level)
	if rate <= 0 {
		return 0
	}
	bp := 20
	lvl := 1
	xpRemaining := player.Experience

	for lvl < 200 {
		r := getXPPerBP(lvl)
		targetBP := buildPointsForLevel(lvl + 1)
		bpToNext := targetBP - bp
		xpForLevel := bpToNext * r

		if xpRemaining >= xpForLevel {
			xpRemaining -= xpForLevel
			bp = targetBP
			lvl++
		} else {
			if r > 0 {
				partialXP := xpRemaining % r
				return r - partialXP
			}
			return 0
		}
	}
	return 0
}

// playerLoadWeight calculates total weight of carried items.
func playerLoadWeight(player *Player, items map[int]*gameworld.ItemDef) int {
	total := 0
	for _, ii := range player.Inventory {
		if def := items[ii.Archetype]; def != nil {
			total += def.Weight
		}
	}
	return total
}

// doSpellList shows the player's known spells.
func (e *GameEngine) doSpellList(player *Player) *CommandResult {
	if len(player.KnownSpells) == 0 {
		return &CommandResult{Messages: []string{"You don't know any spells."}}
	}

	type row struct {
		id    int
		level int
		name  string
	}
	var rows []row
	for id := range player.KnownSpells {
		spell := FindSpellByID(id)
		if spell != nil {
			rows = append(rows, row{id: spell.ID, level: spell.Level, name: spell.Name})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].id < rows[j].id })

	msgs := []string{
		fmt.Sprintf("%-11s%-11s%s", "Spell", "Level", "Name"),
		fmt.Sprintf("%-11s%-11s%s", "-----", "-----", "----"),
	}
	for _, r := range rows {
		msgs = append(msgs, fmt.Sprintf("%-11d%-11d%s", r.id, r.level, r.name))
	}
	return &CommandResult{Messages: msgs}
}

// doRoomRecall handles the RECALL command — recite room lore from scripts.
func (e *GameEngine) doRoomRecall(player *Player) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"Nothing comes to mind."}}
	}
	sc := &ScriptContext{Player: player, Room: room, Engine: e}
	for _, block := range room.Scripts {
		if block.Type == "IFVERB" && len(block.Args) >= 2 {
			if strings.EqualFold(block.Args[0], "RECALL") && block.Args[1] == "-1" {
				sc.execBlock(block)
			}
		}
	}
	if len(sc.Messages) > 0 {
		return &CommandResult{Messages: sc.Messages}
	}
	return &CommandResult{Messages: []string{"Nothing comes to mind about this place."}}
}

// doGuard handles the GUARD command — protect another player.
func (e *GameEngine) doGuard(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		if player.GuardTarget == "" {
			return &CommandResult{Messages: []string{"You are not guarding anyone."}}
		}
		old := player.GuardTarget
		player.GuardTarget = ""
		return &CommandResult{
			Messages:      []string{"You stop guarding."},
			RoomBroadcast: []string{fmt.Sprintf("%s stops guarding %s.", player.FirstName, old)},
		}
	}
	target := strings.ToLower(strings.Join(args, " "))
	found := e.findPlayerInRoom(player, target)
	if found == nil {
		return &CommandResult{Messages: []string{"They are not here."}}
	}
	player.GuardTarget = found.FirstName
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You are now guarding %s.", found.FirstName)},
		RoomBroadcast: []string{fmt.Sprintf("%s is now guarding %s.", player.FirstName, found.FirstName)},
	}
}

// doChant handles the CHANT command — activate a scroll.
func (e *GameEngine) doChant(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Chant what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target = strings.TrimPrefix(target, "my ")
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		if !strings.Contains(strings.ToUpper(itemDef.Type), "SCROLL") {
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

		spellNum := ii.Val3
		if spellNum == 0 {
			return &CommandResult{Messages: []string{"This scroll holds no magical inscription."}}
		}

		spell := FindSpellByID(spellNum)
		if spell == nil {
			return &CommandResult{Messages: []string{"The scroll's magic is indecipherable."}}
		}

		fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)

		player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
		player.PreparedSpell = spellNum
		player.RoundTimeExpiry = time.Now().Add(3 * time.Second)
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("As you chant %s, it crumbles into dust...", fullName),
				fmt.Sprintf("The power of %s flows into you. The spell is prepared.", spell.Name),
				"[Round: 3 sec]",
			},
			RoomBroadcast: []string{
				fmt.Sprintf("%s chants from %s which crumbles into dust.", player.FirstName, fullName),
			},
		}
	}
	return &CommandResult{Messages: []string{"You don't have that."}}
}

// doPositionWithScripts changes the player's position and fires room scripts.
func (e *GameEngine) doPositionWithScripts(ctx context.Context, player *Player, verb, selfMsg, roomMsg string) *CommandResult {
	result := &CommandResult{
		Messages:      []string{selfMsg},
		RoomBroadcast: []string{roomMsg},
		PlayerState:   player,
	}
	room := e.rooms[player.RoomNumber]
	if room != nil {
		verbUpper := strings.ToUpper(verb)
		for _, block := range room.Scripts {
			if block.Type == "IFVERB" && len(block.Args) >= 2 {
				if strings.ToUpper(block.Args[0]) == verbUpper && block.Args[1] == "-1" {
					sc := &ScriptContext{Player: player, Room: room, Engine: e}
					sc.execBlock(block)
					result.Messages = append(result.Messages, sc.Messages...)
					result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
				}
			}
		}
	}
	e.SavePlayer(ctx, player)
	return result
}

// doHide handles the HIDE command.
func (e *GameEngine) doHide(ctx context.Context, player *Player) *CommandResult {
	if player.Hidden {
		return &CommandResult{Messages: []string{"You are already hidden."}}
	}
	if player.Joined {
		return &CommandResult{Messages: []string{"You can't hide while in combat!"}}
	}
	stealthSkill := player.Skills[33]
	hideChance := 25 + stealthSkill*5 + player.Agility/10
	if hideChance > 95 {
		hideChance = 95
	}
	if rand.Intn(100) >= hideChance {
		return &CommandResult{
			Messages:      []string{"You fail to find a suitable hiding place."},
			RoomBroadcast: []string{fmt.Sprintf("%s looks around nervously.", player.FirstName)},
		}
	}
	player.Hidden = true
	e.SavePlayer(ctx, player)
	return &CommandResult{
		Messages:      []string{"You slip into hiding."},
		RoomBroadcast: []string{fmt.Sprintf("%s fades into the shadows.", player.FirstName)},
	}
}

// doSneak handles the SNEAK command — move while hidden.
func (e *GameEngine) doSneak(ctx context.Context, player *Player, args []string) *CommandResult {
	if !player.Hidden {
		return &CommandResult{Messages: []string{"You must be hidden first. Try HIDE."}}
	}
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Sneak where?"}}
	}
	dir := strings.ToUpper(args[0])
	dirMap := map[string]string{
		"N": "N", "NORTH": "N", "S": "S", "SOUTH": "S",
		"E": "E", "EAST": "E", "W": "W", "WEST": "W",
		"NE": "NE", "NORTHEAST": "NE", "NW": "NW", "NORTHWEST": "NW",
		"SE": "SE", "SOUTHEAST": "SE", "SW": "SW", "SOUTHWEST": "SW",
		"U": "U", "UP": "U", "D": "D", "DOWN": "D", "O": "O", "OUT": "O",
	}
	if mapped, ok := dirMap[dir]; ok {
		dir = mapped
	}
	stealthSkill := player.Skills[33]
	sneakChance := 30 + stealthSkill*5 + player.Agility/10
	if sneakChance > 90 {
		sneakChance = 90
	}
	result := e.doMove(ctx, player, dir)
	if rand.Intn(100) >= sneakChance {
		player.Hidden = false
		result.Messages = append(result.Messages, "You have been noticed!")
	}
	return result
}

// doFly handles the FLY command.
func (e *GameEngine) doFly(ctx context.Context, player *Player) *CommandResult {
	if player.Position == 4 {
		return &CommandResult{Messages: []string{"You are already flying."}}
	}
	if player.Race != 6 && !player.CanFly {
		return &CommandResult{Messages: []string{"You can't fly."}}
	}
	room := e.rooms[player.RoomNumber]
	if room != nil && (room.Terrain == "CAVE" || room.Terrain == "DEEPCAVE" || room.Terrain == "INDOOR_FLOOR" || room.Terrain == "INDOOR_GROUND") {
		return &CommandResult{Messages: []string{"There isn't enough room to fly here."}}
	}
	player.Position = 4
	e.SavePlayer(ctx, player)
	return &CommandResult{
		Messages:      []string{"You take to the air!"},
		RoomBroadcast: []string{fmt.Sprintf("%s takes flight!", player.FirstName)},
	}
}

// doSkin handles the SKIN command — skin a dead monster for components.
func (e *GameEngine) doSkin(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Skin what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))

	if e.monsterMgr == nil {
		return &CommandResult{Messages: []string{"You don't see that here."}}
	}

	monsters := e.monsterMgr.AllMonstersInRoom(player.RoomNumber)
	for _, inst := range monsters {
		if inst.Alive {
			continue
		}
		def := e.monsters[inst.DefNumber]
		if def == nil {
			continue
		}
		name := strings.ToLower(FormatMonsterName(def, e.monAdjs))
		noun := strings.ToLower(def.Name)
		if !strings.HasPrefix(name, target) && !strings.HasPrefix(noun, target) {
			continue
		}

		if def.Discorporate {
			return &CommandResult{Messages: []string{"There is nothing left to skin."}}
		}

		if inst.Skinned {
			return &CommandResult{Messages: []string{"This corpse has already been skinned."}}
		}

		if len(def.SkinItems) == 0 && def.SkinAdj == 0 {
			return &CommandResult{Messages: []string{fmt.Sprintf("You can't skin a %s.", def.Name)}}
		}

		displayName := FormatMonsterName(def, e.monAdjs)
		var skinMsgs []string

		if len(def.SkinItems) > 0 {
			totalProb := 0
			for _, si := range def.SkinItems {
				totalProb += si.Probability
			}
			if totalProb > 0 {
				roll := rand.Intn(totalProb)
				cumProb := 0
				for _, si := range def.SkinItems {
					cumProb += si.Probability
					if roll < cumProb {
						skinDef := e.items[si.Archetype]
						if skinDef != nil {
							adj := def.SkinAdj
							skinName := e.formatItemName(skinDef, adj, 0, 0)
							item := InventoryItem{
								Archetype: si.Archetype,
								Adj1:      adj,
							}
							player.Inventory = append(player.Inventory, item)
							skinMsgs = append(skinMsgs, fmt.Sprintf("You carefully skin %s%s and obtain %s.", articleFor(displayName, def.Unique), displayName, skinName))
						}
						break
					}
				}
			}
		}

		if len(skinMsgs) == 0 {
			skinMsgs = append(skinMsgs, fmt.Sprintf("You skin %s %s but find nothing useful.", articleFor(displayName, def.Unique), displayName))
		}

		inst.Skinned = true
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      skinMsgs,
			RoomBroadcast: []string{fmt.Sprintf("%s skins %s %s.", player.FirstName, articleFor(displayName, def.Unique), displayName)},
		}
	}

	return &CommandResult{Messages: []string{"You don't see a dead creature to skin here."}}
}

// doTeach handles the TEACH command — sets up teaching of a skill.
func (e *GameEngine) doTeach(ctx context.Context, player *Player, args []string) *CommandResult {
	// No args: stop teaching
	if len(args) == 0 {
		if player.Teaching == 0 && player.TeachingSpell == 0 {
			return &CommandResult{Messages: []string{"You are not teaching anything."}}
		}
		player.Teaching = 0
		player.TeachingLevel = 0
		player.TeachingSpell = 0
		return &CommandResult{
			Messages:      []string{"You stop teaching."},
			RoomBroadcast: []string{fmt.Sprintf("%s stops teaching.", player.FirstName)},
		}
	}

	// Enforce one teacher per room
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.ID == player.ID {
				continue
			}
			if p.RoomNumber == player.RoomNumber && (p.Teaching > 0 || p.TeachingSpell > 0) {
				return &CommandResult{Messages: []string{fmt.Sprintf("%s is already teaching in this room.", p.FirstName)}}
			}
		}
	}

	target := strings.Join(args, " ")

	// Determine if arg refers to a spell (number >= 100, or a spell name match)
	var spell *SpellDef
	if id, err := strconv.Atoi(target); err == nil {
		if id >= 100 {
			spell = FindSpellByID(id)
		}
		// id < 100 falls through to skill handling below
	} else {
		spell = FindSpellByName(target)
	}

	if spell != nil {
		if !player.KnownSpells[spell.ID] && !player.IsGM {
			return &CommandResult{Messages: []string{fmt.Sprintf("You don't know the spell %s.", spell.Name)}}
		}
		player.Teaching = 0
		player.TeachingLevel = 0
		player.TeachingSpell = spell.ID
		msg := fmt.Sprintf("You are now teaching the spell \"%s.\" (Number %d, level %d).", spell.Name, spell.ID, spell.Level)
		broadcast := fmt.Sprintf("%s is now teaching the spell \"%s.\" (Number %d, level %d).", player.FirstName, spell.Name, spell.ID, spell.Level)
		return &CommandResult{
			Messages:      []string{msg},
			RoomBroadcast: []string{broadcast},
		}
	}

	// Skill teaching: match by number or name
	targetLower := strings.ToLower(target)
	skillNum, err := strconv.Atoi(target)
	if err != nil {
		skillNum = 0
		for id, name := range SkillNames {
			if strings.HasPrefix(strings.ToLower(name), targetLower) {
				skillNum = id
				break
			}
		}
		if skillNum == 0 {
			return &CommandResult{Messages: []string{"Unknown skill or spell."}}
		}
	}
	skillName, ok := SkillNames[skillNum]
	if !ok {
		return &CommandResult{Messages: []string{"Unknown skill."}}
	}
	teacherLevel := player.Skills[skillNum]
	if teacherLevel < 1 {
		return &CommandResult{Messages: []string{"You don't have any training in that skill."}}
	}
	player.Teaching = skillNum
	player.TeachingLevel = teacherLevel
	player.TeachingSpell = 0
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You are now teaching %s (skill #%d, up to level %d).", skillName, skillNum, teacherLevel)},
		RoomBroadcast: []string{fmt.Sprintf("%s is now teaching %s (up to level %d).", player.FirstName, skillName, teacherLevel)},
	}
}
