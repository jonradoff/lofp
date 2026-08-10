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
// appearance may be nil, or have any subset of fields set to zero/empty — any
// stat/height/weight/age that is zero, and any color/style that is empty, gets
// rolled/defaulted here. A fully-populated appearance (from a validated
// creation flow that already let the player reroll and pick colors) is used
// as-is. Callers that accept appearance input from an untrusted client (WS/REST)
// MUST run it through ValidateCharacterAppearance first.
func (e *GameEngine) CreateNewPlayer(ctx context.Context, firstName, lastName string, race, gender int, appearance *CharacterAppearance, accountID ...string) *Player {
	a := CharacterAppearance{}
	if appearance != nil {
		a = *appearance
	}

	if a.Strength == 0 || a.Agility == 0 || a.Quickness == 0 || a.Constitution == 0 ||
		a.Perception == 0 || a.Willpower == 0 || a.Empathy == 0 {
		a.Strength, a.Agility, a.Quickness, a.Constitution, a.Perception, a.Willpower, a.Empathy = RollStats(race)
	}
	if a.Height == 0 || a.Weight == 0 {
		a.Height, a.Weight = RollHeightWeight(race, gender)
	}
	if a.Age == 0 {
		a.Age = RollAge(race)
	}
	if a.EyeColor == "" {
		a.EyeColor = RandomEyeColor()
	}
	if a.SkinColor == "" {
		a.SkinColor = RandomSkinColor()
	}
	if a.HairStyle == "" {
		a.HairStyle = RandomHairStyle()
	}
	if a.HairStyle == "bald" {
		a.HairColor = ""
	} else if a.HairColor == "" {
		a.HairColor = RandomHairColor()
	}

	str, agi, qui, con, per, wil, emp := a.Strength, a.Agility, a.Quickness, a.Constitution, a.Perception, a.Willpower, a.Empathy
	bodyPts := 20 + con/2
	fatigue := 20 + (con+str)/3
	mana := emp / 2
	psi := wil / 2

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
		Height:           a.Height,
		HeightTrue:       a.Height,
		Weight:           a.Weight,
		WeightTrue:       a.Weight,
		Age:              a.Age,
		AgeTrue:          a.Age,
		EyeColor:         a.EyeColor,
		SkinColor:        a.SkinColor,
		HairStyle:        a.HairStyle,
		HairColor:        a.HairColor,
		RoomNumber:       201,
		Position:         0,
		Skills:           make(map[int]int),
		IntNums:          make(map[int]int),
		Gold:             20,
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

	// Backfill fields that didn't exist (or weren't rolled) when this character was
	// created — legacy documents from before Age/appearance were added.
	needsSave := false
	if player.Height == 0 || player.Weight == 0 {
		h, w := RollHeightWeight(player.Race, player.Gender)
		if player.Height == 0 {
			player.Height = h
			player.HeightTrue = h
		}
		if player.Weight == 0 {
			player.Weight = w
			player.WeightTrue = w
		}
		needsSave = true
	}
	if player.Age == 0 {
		age := RollAge(player.Race)
		player.Age = age
		player.AgeTrue = age
		needsSave = true
	}
	if player.EyeColor == "" {
		player.EyeColor = RandomEyeColor()
		needsSave = true
	}
	if player.SkinColor == "" {
		player.SkinColor = RandomSkinColor()
		needsSave = true
	}
	if player.HairStyle == "" {
		player.HairStyle = RandomHairStyle()
		if player.HairStyle != "bald" {
			player.HairColor = RandomHairColor()
		}
		needsSave = true
	}
	if needsSave {
		e.SavePlayer(ctx, &player)
	}

	// Maintained psi powers (e.g. Mind Over Matter 10: Flight) only track their
	// active state in-memory (ActivePsi is session-only, not persisted). If CanFly
	// survived a reload but wasn't granted by race or an active Fly spell, the
	// concentration lapsed when the player disconnected — land them and clear it.
	if player.CanFly && player.Race != RaceDrakin && (player.FlyExpiry.IsZero() || time.Now().After(player.FlyExpiry)) {
		player.CanFly = false
		if player.Position == 4 {
			player.Position = 0
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

// knownOrgNames maps organization numbers to their display names.
var knownOrgNames = map[int]string{
	1:  "Guild of Shadowmancers",
	2:  "Order of the Silver Arcana",
	3:  "Lawkeepers",
	6:  "Crimson Band",
	7:  "Technologists Guild",
	9:  "Temple of Amilor",
	10: "Church of Shemri",
	11: "Temple of Rorin",
	13: "Thieves Guild",
	14: "Cult of Dahkahn",
	15: "Foresters Guild",
	16: "T'Kasta",
	17: "The Assassins",
	20: "Order of the Way",
	21: "The Masquerade",
	22: "Eliditur Fellowship",
	23: "Dark Guard",
	24: "Church of Ordanin",
	25: "Circle of Yarin",
	26: "Guild of Bonecutters",
	27: "Teiwaz Y Perth",
	28: "Order of the Skull",
	40: "Night Shades",
	50: "Organization 50",
}

// organizationName returns the display name for an organization number.
func organizationName(org int) string {
	if n, ok := knownOrgNames[org]; ok {
		return n
	}
	return fmt.Sprintf("Organization %d", org)
}

// orgRankTitle returns the rank title for a numeric org rank, per MANUAL.DOC's
// ORGRANK table: 0-49 Initiate/Acolyte, 50-99 Journeyman/Adept, 100-199
// Master/Priest, 200+ High Master/High Priest. Guild-style words are used
// unless orgType is "TEMPLE" or "CULT" (Priest-style), matching OrgDef.OrgType.
func orgRankTitle(orgType string, rank int) string {
	isTemple := orgType == "TEMPLE" || orgType == "CULT"
	switch {
	case rank >= 200:
		if isTemple {
			return "High Priest"
		}
		return "High Master"
	case rank >= 100:
		if isTemple {
			return "Priest"
		}
		return "Master"
	case rank >= 50:
		if isTemple {
			return "Adept"
		}
		return "Journeyman"
	default:
		if isTemple {
			return "Acolyte"
		}
		return "Initiate"
	}
}

// orgTypeFor returns the OrgDef.OrgType for an org number ("GUILD" by default
// if no OrgDef is loaded for it).
func (e *GameEngine) orgTypeFor(orgNum int) string {
	if def, ok := e.orgDefs[orgNum]; ok {
		return strings.ToUpper(def.OrgType)
	}
	return "GUILD"
}

// autoTrainOrgRank advances a player's rank in whichever organization's
// training room this is, by the build points just spent training there —
// matching orgs.html's documented formula: "ORGRANK = build points spent on
// training within guild". Automatic advancement is capped at 199; 200+
// ("High Master"/"High Priest", see orgRankTitle) is reserved for GM
// assignment via @rank and is never touched here. Returns a flavor message
// ("You are now rank X...") when the displayed rank (ORGRANK/10, per
// orgs.html) increases, or "" if nothing changed.
func (e *GameEngine) autoTrainOrgRank(player *Player, roomNumber, bpSpent int) string {
	var org *gameworld.OrgDef
	for _, def := range e.orgDefs {
		if def.TrainingRoom == roomNumber && player.IsMemberOf(def.Number) {
			org = def
			break
		}
	}
	if org == nil {
		return ""
	}

	oldRank := player.RankIn(org.Number)
	if oldRank >= 199 {
		return ""
	}

	newRank := oldRank + bpSpent
	if newRank > 199 {
		newRank = 199
	}
	player.AddOrg(org.Number, newRank)

	if newRank/10 != oldRank/10 {
		return fmt.Sprintf("You are now rank %d in the %s.", newRank/10, org.Name)
	}
	return ""
}

// doEnroll handles the ENROLL command — joins an OPEN organization by standing in its training room.
func (e *GameEngine) doEnroll(ctx context.Context, player *Player) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You don't seem to be anywhere."}}
	}
	// Find an OPEN org whose training room matches the player's current room.
	var match *gameworld.OrgDef
	for _, def := range e.orgDefs {
		if def.JoinType == "OPEN" && def.TrainingRoom == player.RoomNumber {
			match = def
			break
		}
	}
	if match == nil {
		return &CommandResult{Messages: []string{"There is no open organization here to enroll in."}}
	}
	if player.IsMemberOf(match.Number) {
		return &CommandResult{Messages: []string{fmt.Sprintf("You are already a member of the %s.", match.Name)}}
	}
	player.AddOrg(match.Number, 1)
	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: []string{fmt.Sprintf("You enroll in the %s.", match.Name)}}
}

// playerBPSpent calculates total build points spent on skills.
func playerBPSpent(player *Player) int {
	total := 0
	for skillID, rank := range player.Skills {
		for r := 0; r < rank; r++ {
			total += skillBPCost(skillID, r)
		}
	}
	// 8 BP for first mastery rank, 4 BP for each additional rank
	for _, rank := range player.SpellMastery {
		if rank >= 1 {
			total += 8
		}
		if rank >= 2 {
			total += 4 * (rank - 1)
		}
	}
	// 8 BP for first specialization rank, 4 BP for each additional rank
	for _, rank := range player.WeaponSpecialization {
		if rank >= 1 {
			total += 8
		}
		if rank >= 2 {
			total += 4 * (rank - 1)
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
	bp := 30 // starting build points (matches CreateNewPlayer / recalcBuildPoints)
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

// playerLoadWeight calculates total weight of carried items (inventory + off-hand).
func playerLoadWeight(player *Player, items map[int]*gameworld.ItemDef) int {
	total := 0
	for _, ii := range player.Inventory {
		if def := items[ii.Archetype]; def != nil {
			total += def.Weight
		}
	}
	if player.OffHand != nil {
		if def := items[player.OffHand.Archetype]; def != nil {
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
		id      int
		level   int
		school  string
		name    string
		mastery int
	}
	var rows []row
	for id := range player.KnownSpells {
		spell := FindSpellByID(id)
		if spell != nil {
			rows = append(rows, row{
				id:      spell.ID,
				level:   spell.Level,
				school:  spell.School,
				name:    spell.Name,
				mastery: spellMasteryLevel(player, spell),
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].school != rows[j].school {
			return rows[i].school < rows[j].school
		}
		if rows[i].level != rows[j].level {
			return rows[i].level < rows[j].level
		}
		return rows[i].id < rows[j].id
	})

	msgs := []string{
		fmt.Sprintf("%-7s%-7s%-16s%s", "Spell", "Level", "School", "Name"),
		fmt.Sprintf("%-7s%-7s%-16s%s", "-----", "-----", "------", "----"),
	}
	for _, r := range rows {
		stars := ""
		if r.mastery > 0 {
			stars = " (" + strings.Repeat("*", r.mastery) + ")"
		}
		msgs = append(msgs, fmt.Sprintf("%-7d%-7d%-16s%s%s", r.id, r.level, r.school, r.name, stars))
	}
	return &CommandResult{Messages: msgs}
}

// doRoomRecall handles the RECALL command — recite room lore from scripts.
func (e *GameEngine) doRoomRecall(player *Player) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"Nothing comes to mind."}}
	}
	sc := &ScriptContext{Player: player, Room: room, Engine: e, activeVerb: "RECALL", activeRef: "-1"}
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

// doGuard handles the GUARD command — protect players, portals, or items.
// Requires Combat Maneuvering (skill 10): level 2+ for players, level 3+ for portals, any level for items.
// GUARD with no args clears all guard targets.
// GUARD <target> toggles guarding that target on/off.
func (e *GameEngine) doGuard(player *Player, args []string) *CommandResult {
	cmSkill := player.Skills[10]

	if len(args) == 0 {
		hasGuards := len(player.GuardTargets) > 0 || len(player.GuardPortals) > 0 || len(player.GuardItems) > 0
		if !hasGuards {
			return &CommandResult{Messages: []string{"You are not guarding anyone or anything."}}
		}
		player.GuardTargets = nil
		player.GuardPortals = nil
		player.GuardItems = nil
		return &CommandResult{
			Messages:      []string{"You relax your guard."},
			RoomBroadcast: []string{fmt.Sprintf("%s relaxes their guard.", player.FirstName)},
		}
	}

	if !player.IsGM && cmSkill < 1 {
		return &CommandResult{Messages: []string{"You have no training in Combat Maneuvering."}}
	}

	target := strings.ToLower(strings.Join(args, " "))
	room := e.rooms[player.RoomNumber]

	// Try player target first
	found := e.findPlayerInRoom(player, target)
	if found != nil {
		if !player.IsGM && cmSkill < 2 {
			return &CommandResult{Messages: []string{"You need Combat Maneuvering rank 2 to guard another player."}}
		}
		// Can't guard someone who is themselves on guard duty
		if len(found.GuardTargets) > 0 || len(found.GuardPortals) > 0 || len(found.GuardItems) > 0 {
			return &CommandResult{Messages: []string{fmt.Sprintf("%s is on guard duty and cannot be guarded.", found.FirstName)}}
		}
		// Toggle: remove if already guarding this player
		for i, t := range player.GuardTargets {
			if t == found.FirstName {
				player.GuardTargets = append(player.GuardTargets[:i], player.GuardTargets[i+1:]...)
				return &CommandResult{
					Messages:      []string{fmt.Sprintf("You stop guarding %s.", found.FirstName)},
					RoomBroadcast: []string{fmt.Sprintf("%s stops guarding %s.", player.FirstName, found.FirstName)},
				}
			}
		}
		player.GuardTargets = append(player.GuardTargets, found.FirstName)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You position yourself to guard %s.", found.FirstName)},
			RoomBroadcast: []string{fmt.Sprintf("%s moves to guard %s.", player.FirstName, found.FirstName)},
		}
	}

	if room == nil {
		return &CommandResult{Messages: []string{"You don't see that here."}}
	}

	// Try items in the room — portal or regular item
	for _, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ri.Adj1), e.getAdjName(ri.Adj2), e.getAdjName(ri.Adj3)) {
			continue
		}

		if isPortal(itemDef.Type) {
			if !player.IsGM && cmSkill < 3 {
				return &CommandResult{Messages: []string{"You need Combat Maneuvering rank 3 to guard a portal."}}
			}
			portalName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3, ri.Extend)
			arch := ri.Archetype
			for i, a := range player.GuardPortals {
				if a == arch {
					player.GuardPortals = append(player.GuardPortals[:i], player.GuardPortals[i+1:]...)
					return &CommandResult{
						Messages:      []string{fmt.Sprintf("You step away from %s.", portalName)},
						RoomBroadcast: []string{fmt.Sprintf("%s stops guarding %s.", player.FirstName, portalName)},
					}
				}
			}
			player.GuardPortals = append(player.GuardPortals, arch)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You take a guarding stance before %s.", portalName)},
				RoomBroadcast: []string{fmt.Sprintf("%s takes a guarding stance before %s.", player.FirstName, portalName)},
			}
		}

		// Regular item on the ground
		if containsFlag(itemDef.Flags, "FIXED") || itemDef.Type == "MANUSCRIPT" || itemDef.Weight >= 1000 {
			return &CommandResult{Messages: []string{"You can't guard that."}}
		}
		itemName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3, ri.Extend)
		arch := ri.Archetype
		for i, a := range player.GuardItems {
			if a == arch {
				player.GuardItems = append(player.GuardItems[:i], player.GuardItems[i+1:]...)
				return &CommandResult{
					Messages:      []string{fmt.Sprintf("You stop watching over %s.", itemName)},
					RoomBroadcast: []string{fmt.Sprintf("%s stops watching over %s.", player.FirstName, itemName)},
				}
			}
		}
		player.GuardItems = append(player.GuardItems, arch)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You stand watch over %s.", itemName)},
			RoomBroadcast: []string{fmt.Sprintf("%s stands watch over %s.", player.FirstName, itemName)},
		}
	}

	return &CommandResult{Messages: []string{"You don't see that here."}}
}

// doChant handles the CHANT command — activate a scroll.
func (e *GameEngine) doChant(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Chant what?"}}
	}
	if player.Skills[23] < 1 && !player.IsGM {
		return &CommandResult{Messages: []string{"You lack the training in Spellcraft to invoke a scroll's magic."}}
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
		if !matchesTarget(name, target, e.getAdjName(ii.Adj1), e.getAdjName(ii.Adj2), e.getAdjName(ii.Adj3)) {
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

		fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3, ii.Tail)

		player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
		player.PreparedSpell = spellNum
		player.PreparedSpellReagentArch = 0 // scroll casting never requires a reagent
		player.PreparedMoonstoneBonus = false
		scrollRT := applyRoundTime(player, 3)
		player.RoundTimeExpiry = time.Now().Add(time.Duration(scrollRT) * time.Second)
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("As you chant %s, it crumbles into dust...", fullName),
				fmt.Sprintf("The power of %s flows into you. The spell is prepared.", spell.Name),
				fmt.Sprintf("[Round: %d sec]", scrollRT),
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
					sc := &ScriptContext{Player: player, Room: room, Engine: e, activeVerb: verbUpper, activeRef: "-1"}
					sc.execBlock(block)
					result.Messages = append(result.Messages, sc.Messages...)
					result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
					// A synchronous MOVE (no PLREVENT delay) relocates the player immediately.
					if sc.MoveTo > 0 {
						if newRoom := e.rooms[sc.MoveTo]; newRoom != nil {
							player.RoomNumber = sc.MoveTo
							lookResult := e.doLook(player)
							result.Messages = append(result.Messages, lookResult.Messages...)
							result.RoomName = lookResult.RoomName
							result.RoomDesc = lookResult.RoomDesc
							result.Exits = lookResult.Exits
							result.Items = lookResult.Items
							e.applyEntryScripts(ctx, player, newRoom, result)
						}
					}
					// PLREVENT/CONTPLREVENT-deferred actions (e.g. a delayed MOVE) must be
					// scheduled, or their effects — including the move itself — are lost.
					if len(sc.DeferredSegments) > 0 {
						e.scheduleScriptSegments(player, sc.DeferredSegments)
					}
				}
			}
		}
		// Healer rooms treat sitting or laying as requesting treatment
		if (verbUpper == "SIT" || verbUpper == "LAY") && containsModifier(room.Modifiers, "HEALER") {
			e.applyHealerRoom(ctx, player, result)
		}
	}
	e.SavePlayer(ctx, player)
	return result
}

// applyHealerRoom handles physician healing when a player sits or lays in a HEALER room.
// Charges 1 copper per body point healed. If the player has no money at all they are turned away.
// If they have some money but not enough they are charged everything they have and still healed.
// Cure poison costs 10 gold (1000 copper); cure disease costs 10 gold (1000 copper).
func (e *GameEngine) applyHealerRoom(_ context.Context, player *Player, result *CommandResult) {
	treated := false

	// Handle status conditions first, independently of wounds
	if player.Poisoned {
		const poisonCost = 1000 // 10 gold in copper
		totalCopper := player.Gold*100 + player.Silver*10 + player.Copper
		if totalCopper >= poisonCost {
			e.deductCopper(player, poisonCost)
			player.Poisoned = false
			player.PoisonLevel = 0
			result.Messages = append(result.Messages, "A physician casts cure poison on you, neutralizing the toxin. You are charged 10 gold crowns.")
			result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("A physician casts cure poison on %s.", player.FirstName))
			treated = true
		} else {
			result.Messages = append(result.Messages, "A physician examines you and shakes their head. \"We can cure your poison for 10 gold crowns, but you cannot afford our services.\"")
		}
	}

	if player.Diseased {
		const diseaseCost = 1000 // 10 gold in copper
		totalCopper := player.Gold*100 + player.Silver*10 + player.Copper
		if totalCopper >= diseaseCost {
			e.deductCopper(player, diseaseCost)
			player.Diseased = false
			player.DiseaseLevel = 0
			result.Messages = append(result.Messages, "A physician casts cure disease on you, purging the illness. You are charged 10 gold crowns.")
			result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("A physician casts cure disease on %s.", player.FirstName))
			treated = true
		} else {
			result.Messages = append(result.Messages, "A physician examines you and shakes their head. \"We can cure your disease for 10 gold crowns, but you cannot afford our services.\"")
		}
	}

	// Heal wounds
	woundsNeeded := player.MaxBodyPoints - player.BodyPoints
	if woundsNeeded <= 0 && len(player.Wounds) == 0 {
		if !treated {
			result.Messages = append(result.Messages, "A physician examines you and nods. \"You appear to be in perfect health. No treatment needed.\"")
		}
		return
	}

	totalCopper := player.Gold*100 + player.Silver*10 + player.Copper
	if totalCopper <= 0 {
		result.Messages = append(result.Messages, "A physician examines you and frowns. \"I'm sorry, but we require payment for our services. Come back when you have some coin.\"")
		return
	}

	// 1 copper per body point restored, plus an additional 5 copper per
	// severity level for each wound treated. If they can't afford it all,
	// take all they have.
	severityTotal := 0
	for _, w := range player.Wounds {
		severityTotal += w.Level
	}
	cost := woundsNeeded + 5*severityTotal
	charged := cost
	if charged > totalCopper {
		charged = totalCopper
	}
	e.deductCopper(player, charged)
	player.BodyPoints = player.MaxBodyPoints
	player.Wounds = nil
	player.Bleeding = false

	if charged < cost {
		result.Messages = append(result.Messages, fmt.Sprintf("A physician tends to your wounds. You are charged %s (all you had).", formatPrice(charged)))
	} else {
		result.Messages = append(result.Messages, fmt.Sprintf("A physician tends to your wounds. You are charged %s.", formatPrice(charged)))
	}
	result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("A physician tends to %s's wounds.", player.FirstName))
}

// deductCopper deducts an amount in copper from the player's purse, breaking higher denominations as needed.
func (e *GameEngine) deductCopper(player *Player, amount int) {
	remaining := amount
	if player.Copper >= remaining {
		player.Copper -= remaining
		return
	}
	remaining -= player.Copper
	player.Copper = 0
	if remaining > 0 {
		silverNeeded := (remaining + 9) / 10
		if player.Silver >= silverNeeded {
			player.Silver -= silverNeeded
			player.Copper += silverNeeded*10 - remaining
			return
		}
		remaining -= player.Silver * 10
		player.Silver = 0
	}
	if remaining > 0 {
		goldNeeded := (remaining + 99) / 100
		player.Gold -= goldNeeded
		player.Copper += goldNeeded*100 - remaining
	}
}

// doHide handles the HIDE command.
func (e *GameEngine) doHide(ctx context.Context, player *Player) *CommandResult {
	if player.Hidden {
		return &CommandResult{Messages: []string{"You are already hidden."}}
	}
	if player.Joined {
		return &CommandResult{Messages: []string{"You can't hide while in combat!"}}
	}
	stealthSkill := effectiveStealthSkill(player)
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
	stealthSkill := effectiveStealthSkill(player)
	sneakChance := 30 + stealthSkill*5 + player.Agility/10 + player.Quickness/10

	// Harder to slip in unnoticed if someone is already in the destination room —
	// reduce the chance by the highest Perception/5 among players there.
	if room := e.rooms[player.RoomNumber]; room != nil {
		if destNum, ok := room.Exits[dir]; ok && e.sessions != nil {
			highestPerception := 0
			for _, p := range e.sessions.OnlinePlayers() {
				if p.RoomNumber == destNum && p.FirstName != player.FirstName && p.Perception > highestPerception {
					highestPerception = p.Perception
				}
			}
			sneakChance -= highestPerception / 5
		}
	}

	if sneakChance > 90 {
		sneakChance = 90
	}
	sneakSuccess := rand.Intn(100) < sneakChance
	// doMove unconditionally clears player.Hidden as a side effect of normal movement
	// (see movement.go) — restore it here based on the sneak roll rather than the
	// stealth skill silently doing nothing on success.
	result := e.doMove(ctx, player, dir)
	if result.OldRoom == 0 {
		// Move never actually happened (blocked, immobilized, wrong position, etc.) —
		// stay hidden and don't consume the roll.
		player.Hidden = true
		return result
	}
	// Sneaking is slower than a normal move (base 2 sec, halved to 1 under Haste).
	sneakRT := applyRoundTime(player, 2)
	player.RoundTimeExpiry = time.Now().Add(time.Duration(sneakRT) * time.Second)
	result.Messages = append(result.Messages, fmt.Sprintf("[Round: %d sec]", sneakRT))
	if sneakSuccess {
		player.Hidden = true
	} else {
		player.Hidden = false
		result.Messages = append(result.Messages, "You have been noticed!")
	}
	return result
}

// doAppearance handles the APPEARANCE command, letting a player set a custom line
// shown on EXAMINE after the listing of worn items (e.g. "You catch the scent of
// some exotic cologne wafting from his direction."). Since LOOK/EXAMINE already
// show a player their own appearance, APPEARANCE with no text isn't useful as a
// display — instead it removes the current line, same as APPEARANCE CLEAR.
func (e *GameEngine) doAppearance(ctx context.Context, player *Player, rawInput string) *CommandResult {
	text := extractOriginalArgs(rawInput)
	if text == "" {
		if player.Appearance == "" {
			return &CommandResult{Messages: []string{"You don't have a custom appearance line set. (usage: APPEARANCE <description>)"}}
		}
		player.Appearance = ""
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"Your custom appearance line has been cleared."}}
	}
	if strings.EqualFold(text, "clear") || strings.EqualFold(text, "off") || strings.EqualFold(text, "none") {
		player.Appearance = ""
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"Your custom appearance line has been cleared."}}
	}
	player.Appearance = text
	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: []string{fmt.Sprintf("Your appearance line is now: %s", text)}}
}

// doFly handles the FLY command.
func (e *GameEngine) doFly(ctx context.Context, player *Player) *CommandResult {
	if player.Position == 4 {
		return &CommandResult{Messages: []string{"You are already flying."}}
	}
	if player.Race != 6 && !player.CanFly && !player.MistForm {
		return &CommandResult{Messages: []string{"You can't fly."}}
	}
	room := e.rooms[player.RoomNumber]
	if !player.MistForm && room != nil && (room.Terrain == "CAVE" || room.Terrain == "DEEPCAVE" || room.Terrain == "INDOOR_FLOOR" || room.Terrain == "INDOOR_GROUND") {
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

		// Check/mark Skinned on the actual instance (AllMonstersInRoom returns copies).
		e.monsterMgr.mu.Lock()
		idx := e.monsterMgr.indexOfID(inst.ID)
		if idx >= 0 && e.monsterMgr.instances[idx].Skinned {
			e.monsterMgr.mu.Unlock()
			return &CommandResult{Messages: []string{"This corpse has already been skinned."}}
		}
		if idx >= 0 {
			e.monsterMgr.instances[idx].Skinned = true
		}
		e.monsterMgr.mu.Unlock()

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
							skinName := e.formatItemName(skinDef, 0, adj, 0)
							item := InventoryItem{
								Archetype: si.Archetype,
								Adj2:      adj,
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
