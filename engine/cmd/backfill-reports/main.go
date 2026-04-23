package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jonradoff/lofp/internal/feedback"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type LogEntry struct {
	ID        bson.ObjectID `bson:"_id"`
	Timestamp time.Time     `bson:"timestamp"`
	Event     string        `bson:"event"`
	Player    string        `bson:"player"`
	Details   string        `bson:"details"`
	RoomNum   int           `bson:"roomNum"`
	RoomName  string        `bson:"roomName"`
}

func main() {
	mongoURI := os.Getenv("MONGODB_URI")
	vibectlURL := os.Getenv("VIBECTL_URL")
	vibectlKey := os.Getenv("VIBECTL_API_KEY")

	if mongoURI == "" || vibectlURL == "" || vibectlKey == "" {
		log.Fatal("Required env vars: MONGODB_URI, VIBECTL_URL, VIBECTL_API_KEY")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("MongoDB connect: %v", err)
	}
	defer client.Disconnect(ctx)

	coll := client.Database("lofp").Collection("game_logs")

	cursor, err := coll.Find(ctx, bson.M{"event": "report"}, options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}}))
	if err != nil {
		log.Fatalf("Query: %v", err)
	}
	defer cursor.Close(ctx)

	var entries []LogEntry
	if err := cursor.All(ctx, &entries); err != nil {
		log.Fatalf("Decode: %v", err)
	}

	if len(entries) == 0 {
		fmt.Println("No report entries found.")
		return
	}

	fmt.Printf("Found %d report entries to backfill.\n", len(entries))

	fb := feedback.New(vibectlURL, vibectlKey)

	// Batch in groups of 50
	const batchSize = 50
	for i := 0; i < len(entries); i += batchSize {
		end := i + batchSize
		if end > len(entries) {
			end = len(entries)
		}
		batch := entries[i:end]

		var items []feedback.BatchItem
		for _, e := range batch {
			items = append(items, feedback.NewBatchItem(e.Player, e.RoomNum, e.RoomName, e.Details))
		}

		if err := fb.SubmitBatch(items); err != nil {
			log.Fatalf("Batch submit failed at offset %d: %v", i, err)
		}
		fmt.Printf("  Submitted %d-%d of %d\n", i+1, end, len(entries))
	}

	fmt.Println("Backfill complete.")
}
