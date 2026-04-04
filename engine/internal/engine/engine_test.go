package engine

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jonradoff/lofp/internal/gameworld"
	"github.com/jonradoff/lofp/internal/scriptparser"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func loadTestParsed(t *testing.T) *gameworld.ParsedData {
	t.Helper()
	cfgPath := "../../../original/scripts/LEGENDS.CFG"
	if _, err := os.Stat(cfgPath); err != nil {
		cfgPath = "../../original/scripts/LEGENDS.CFG"
	}
	result, err := scriptparser.ParseConfig(cfgPath)
	if err != nil {
		t.Fatalf("Failed to parse scripts: %v", err)
	}
	return &gameworld.ParsedData{
		Rooms: result.Rooms, Items: result.Items, Monsters: result.Monsters,
		Nouns: result.Nouns, Adjectives: result.Adjectives, MonsterAdjs: result.MonsterAdjs,
		Variables: result.Variables, Regions: result.Regions, MonsterLists: result.MonsterLists,
		CEvents: result.CEvents, MoneyDefs: result.MoneyDefs, ForageDefs: result.ForageDefs,
		MineDefs: result.MineDefs, StartRoom: result.StartRoom, BumpRoom: result.BumpRoom,
	}
}

func connectTestDB(t *testing.T) *mongo.Database {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use the same MongoDB as the game (from env)
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		t.Skip("MONGODB_URI not set")
	}
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Skipf("Cannot connect to MongoDB: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		t.Skipf("Cannot ping MongoDB: %v", err)
	}
	return client.Database("lofp")
}

func TestLoadPlayerTaliesin(t *testing.T) {
	db := connectTestDB(t)
	ctx := context.Background()

	// Direct MongoDB query to find Taliesin
	coll := db.Collection("players")

	// First, find ALL characters with firstName "Taliesin" (including any soft-deleted)
	cursor, err := coll.Find(ctx, bson.M{"firstName": "Taliesin"})
	if err != nil {
		t.Fatalf("Find error: %v", err)
	}
	var allPlayers []Player
	if err := cursor.All(ctx, &allPlayers); err != nil {
		t.Fatalf("Cursor error: %v", err)
	}

	t.Logf("Found %d player(s) with firstName 'Taliesin':", len(allPlayers))
	for _, p := range allPlayers {
		t.Logf("  Name: %s %s, AccountID: %s, Room: %d, Level: %d, DeletedAt: %v",
			p.FirstName, p.LastName, p.AccountID, p.RoomNumber, p.Level, p.DeletedAt)
	}

	if len(allPlayers) == 0 {
		t.Fatal("No character named Taliesin found in database!")
	}

	// Now test the LoadPlayer function (which filters out soft-deleted)
	player := allPlayers[0]
	filter := bson.M{
		"firstName": player.FirstName,
		"lastName":  player.LastName,
		"deletedAt": bson.M{"$exists": false},
	}
	var loaded Player
	err = coll.FindOne(ctx, filter).Decode(&loaded)
	if err != nil {
		t.Logf("LoadPlayer filter FAILED for %s %s: %v", player.FirstName, player.LastName, err)
		t.Logf("This means the character has a deletedAt field or the name doesn't match exactly")

		// Check if deletedAt exists
		var raw bson.M
		coll.FindOne(ctx, bson.M{"firstName": "Taliesin"}).Decode(&raw)
		if raw != nil {
			if da, ok := raw["deletedAt"]; ok {
				t.Logf("  deletedAt field exists: %v", da)
			} else {
				t.Logf("  deletedAt field does NOT exist (should match)")
			}
			t.Logf("  firstName: %q, lastName: %q", raw["firstName"], raw["lastName"])
		}
	} else {
		t.Logf("LoadPlayer succeeded: %s %s (account: %s)", loaded.FirstName, loaded.LastName, loaded.AccountID)
	}

	// Check all accounts in the system
	accountColl := db.Collection("accounts")
	acCursor, _ := accountColl.Find(ctx, bson.M{})
	var accounts []bson.M
	acCursor.All(ctx, &accounts)
	t.Logf("All accounts in system:")
	for _, a := range accounts {
		name, _ := a["name"].(string)
		email, _ := a["email"].(string)
		id := a["_id"]
		t.Logf("  ID=%v name=%s email=%s", id, name, email)
	}

	// Test ListPlayersByAccount
	if player.AccountID != "" {
		accountFilter := bson.M{"accountId": player.AccountID, "deletedAt": bson.M{"$exists": false}}
		cursor2, _ := coll.Find(ctx, accountFilter)
		var accountPlayers []Player
		cursor2.All(ctx, &accountPlayers)
		t.Logf("Characters for account %s: %d", player.AccountID, len(accountPlayers))
		for _, p := range accountPlayers {
			t.Logf("  %s %s (level %d)", p.FirstName, p.LastName, p.Level)
		}
	}
}

func TestRoom225Stairway(t *testing.T) {
	db := connectTestDB(t)
	ctx := context.Background()

	// Load parsed game data
	parsed := loadTestParsed(t)
	ge := NewGameEngine(db, parsed)

	// Check room 225 items
	room := ge.rooms[225]
	if room == nil {
		t.Fatal("Room 225 not found")
	}
	t.Logf("Room 225: %s", room.Name)
	t.Logf("  Items: %d", len(room.Items))
	for i, ri := range room.Items {
		def := ge.items[ri.Archetype]
		if def == nil {
			t.Logf("  Item %d: arch=%d (nil def)", i, ri.Archetype)
			continue
		}
		noun := ge.nouns[def.NameID]
		t.Logf("  Item %d: arch=%d noun=%q type=%q val2=%d hidden=%v isPortal=%v",
			i, ri.Archetype, noun, def.Type, ri.Val2, containsFlag(def.Flags, "HIDDEN"), isPortal(def.Type))
	}

	// Check if "stair" matches item 81
	for _, ri := range room.Items {
		def := ge.items[ri.Archetype]
		if def == nil { continue }
		noun := ge.nouns[def.NameID]
		if matchesTarget(noun, "stair", "") {
			t.Logf("  MATCH: 'stair' matches noun=%q arch=%d type=%q val2=%d", noun, ri.Archetype, def.Type, ri.Val2)
		}
	}

	// Simulate a player going to stairway
	player := &Player{
		FirstName: "Test", LastName: "Player",
		RoomNumber: 225, Race: 1, Gender: 0,
		Skills: make(map[int]int), IntNums: make(map[int]int),
		BodyPoints: 50, MaxBodyPoints: 50,
	}
	result := ge.ProcessCommand(ctx, player, "go stair")
	t.Logf("GO STAIR result: %v", result.Messages)
	t.Logf("  Player room after: %d", player.RoomNumber)

	if player.RoomNumber != 298 {
		t.Errorf("Expected player in room 298 after GO STAIR, got %d", player.RoomNumber)
	}
}

func TestRoom592Exists(t *testing.T) {
	parsed := loadTestParsed(t)
	
	// Check if room 592 is in parsed results
	found := false
	for _, r := range parsed.Rooms {
		if r.Number == 592 {
			found = true
			t.Logf("Room 592 found: %s (source: %s)", r.Name, r.SourceFile)
			t.Logf("  Exits: %v", r.Exits)
			t.Logf("  Items: %d", len(r.Items))
			for i, item := range r.Items {
				t.Logf("  Item %d: arch=%d val2=%d", i, item.Archetype, item.Val2)
			}
			break
		}
	}
	if !found {
		t.Error("Room 592 NOT FOUND in parsed data!")
		
		// Check which rooms ARE parsed from AMILOR.SCR
		for _, r := range parsed.Rooms {
			if r.SourceFile == "AMILOR.SCR" {
				t.Logf("  AMILOR.SCR has room %d: %s", r.Number, r.Name)
			}
		}
	}
	
	// Also check total room count and if 591 exists
	t.Logf("Total rooms: %d", len(parsed.Rooms))
	for _, r := range parsed.Rooms {
		if r.Number == 591 {
			t.Logf("Room 591 found: %s (source: %s)", r.Name, r.SourceFile)
		}
	}
}
