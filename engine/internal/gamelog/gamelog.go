package gamelog

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type EventType string

const (
	EventLogin           EventType = "login"            // User authenticated via Google
	EventLogout          EventType = "logout"           // User session ended
	EventGameEnter       EventType = "game_enter"       // Character entered the game world
	EventGameExit        EventType = "game_exit"        // Character left the game world
	EventCharacterCreate EventType = "character_create"  // New character created
	EventLevelUp         EventType = "level_up"
	EventGMGrant         EventType = "gm_grant"
	EventGMRevoke        EventType = "gm_revoke"
	EventGMCommand       EventType = "gm_command"        // GM action (script upload, etc.)
	EventReport          EventType = "report"            // Player REPORT command
)

type LogEntry struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Timestamp time.Time     `bson:"timestamp" json:"timestamp"`
	Event     EventType     `bson:"event" json:"event"`
	Player    string        `bson:"player" json:"player"`
	AccountID string        `bson:"accountId,omitempty" json:"accountId,omitempty"`
	Details   string        `bson:"details,omitempty" json:"details,omitempty"`
	RoomNum   int           `bson:"roomNum,omitempty" json:"roomNum,omitempty"`
	RoomName  string        `bson:"roomName,omitempty" json:"roomName,omitempty"`
	// Resolved marks a log entry (chiefly a player REPORT) as addressed, so the
	// admin Logs UI can tell what still needs work from what's already been
	// handled. ResolvedBy/ResolvedAt record who cleared it and when.
	Resolved   bool       `bson:"resolved,omitempty" json:"resolved,omitempty"`
	ResolvedBy string     `bson:"resolvedBy,omitempty" json:"resolvedBy,omitempty"`
	ResolvedAt *time.Time `bson:"resolvedAt,omitempty" json:"resolvedAt,omitempty"`
}

type Logger struct {
	coll *mongo.Collection
}

func New(db *mongo.Database) *Logger {
	if db == nil {
		return &Logger{}
	}
	coll := db.Collection("game_logs")

	// Create TTL index to auto-expire old logs (90 days)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "timestamp", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(90 * 24 * 60 * 60),
	})
	// Index for querying by event type and player
	coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "event", Value: 1}, {Key: "timestamp", Value: -1}},
	})
	coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "player", Value: 1}, {Key: "timestamp", Value: -1}},
	})

	return &Logger{coll: coll}
}

func (l *Logger) Log(event EventType, player, accountID, details string, roomNum int, roomName string) {
	if l.coll == nil {
		return
	}
	entry := LogEntry{
		Timestamp: time.Now(),
		Event:     event,
		Player:    player,
		AccountID: accountID,
		Details:   details,
		RoomNum:   roomNum,
		RoomName:  roomName,
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := l.coll.InsertOne(ctx, entry); err != nil {
			log.Printf("gamelog: failed to write log: %v", err)
		}
	}()
}

// resolvedFilter selects on the Resolved flag: "true" or "false" filters to that
// state, anything else (including "") returns entries regardless of it.
func (l *Logger) Query(ctx context.Context, eventFilter string, playerFilter string, resolvedFilter string, limit int) ([]LogEntry, error) {
	if l.coll == nil {
		return nil, nil
	}
	filter := bson.M{}
	if eventFilter != "" {
		filter["event"] = eventFilter
	}
	if playerFilter != "" {
		filter["player"] = bson.M{"$regex": regexp.QuoteMeta(playerFilter), "$options": "i"}
	}
	switch resolvedFilter {
	case "true":
		filter["resolved"] = true
	case "false":
		filter["resolved"] = bson.M{"$ne": true}
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}).SetLimit(int64(limit))
	cursor, err := l.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	var entries []LogEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// SetResolved marks (or unmarks) a log entry as resolved and returns the updated
// entry. Used by the admin Logs UI's "mark done" action on a player REPORT.
func (l *Logger) SetResolved(ctx context.Context, id string, resolved bool, resolvedBy string) (*LogEntry, error) {
	if l.coll == nil {
		return nil, fmt.Errorf("no database connection")
	}
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid log id: %w", err)
	}
	update := bson.M{"resolved": resolved}
	if resolved {
		update["resolvedBy"] = resolvedBy
		update["resolvedAt"] = time.Now()
	} else {
		update["resolvedBy"] = ""
		update["resolvedAt"] = nil
	}
	res := l.coll.FindOneAndUpdate(ctx, bson.M{"_id": oid}, bson.M{"$set": update},
		options.FindOneAndUpdate().SetReturnDocument(options.After))
	var entry LogEntry
	if err := res.Decode(&entry); err != nil {
		return nil, err
	}
	return &entry, nil
}
