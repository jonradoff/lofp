package engine

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jonradoff/lofp/internal/gameworld"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// GMScript represents a GM-uploaded script stored in MongoDB.
type GMScript struct {
	Filename            string           `bson:"_id" json:"filename"`
	Name                string           `bson:"name" json:"name"`
	Content             string           `bson:"content" json:"content"`
	Priority            int              `bson:"priority" json:"priority"` // higher = loads sooner
	Size                int              `bson:"size" json:"size"`
	UploadedBy          string           `bson:"uploadedBy" json:"uploadedBy"`
	UploadedByAccountID string           `bson:"uploadedByAccountId" json:"uploadedByAccountId"`
	UploadedAt          time.Time        `bson:"uploadedAt" json:"uploadedAt"`
	ParseStats          ScriptApplyStats `bson:"parseStats" json:"parseStats"`
	History             []GMScriptVersion `bson:"history" json:"history"`
}

// GMScriptVersion is a historical version of a script.
type GMScriptVersion struct {
	Content    string    `bson:"content" json:"content"`
	UploadedBy string    `bson:"uploadedBy" json:"uploadedBy"`
	UploadedAt time.Time `bson:"uploadedAt" json:"uploadedAt"`
}

const gmScriptsCollection = "gm_scripts"
const maxScriptHistory = 10
const MaxScriptSize = 268245 // 110% of largest disk script (MONSTERS.SCR at 243,859 bytes)

// ListGMScripts returns all GM scripts sorted by priority (descending) then name.
func (e *GameEngine) ListGMScripts(ctx context.Context) ([]GMScript, error) {
	if e.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	opts := options.Find().SetSort(bson.D{{Key: "priority", Value: -1}, {Key: "name", Value: 1}})
	// Don't include content or history in list — just metadata
	opts.SetProjection(bson.M{"content": 0, "history": 0})
	cursor, err := e.db.Collection(gmScriptsCollection).Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	var scripts []GMScript
	if err := cursor.All(ctx, &scripts); err != nil {
		return nil, err
	}
	return scripts, nil
}

// GetGMScript returns a single GM script by filename.
func (e *GameEngine) GetGMScript(ctx context.Context, filename string) (*GMScript, error) {
	if e.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	var script GMScript
	err := e.db.Collection(gmScriptsCollection).FindOne(ctx, bson.M{"_id": filename}).Decode(&script)
	if err != nil {
		return nil, err
	}
	return &script, nil
}

// SaveGMScript upserts a GM script, pushing the previous version to history.
func (e *GameEngine) SaveGMScript(ctx context.Context, script *GMScript) error {
	if e.db == nil {
		return fmt.Errorf("database not available")
	}
	// Check if exists, push old content to history
	existing, err := e.GetGMScript(ctx, script.Filename)
	if err == nil && existing.Content != "" {
		script.History = existing.History
		// Push previous version to front of history
		prev := GMScriptVersion{
			Content:    existing.Content,
			UploadedBy: existing.UploadedBy,
			UploadedAt: existing.UploadedAt,
		}
		script.History = append([]GMScriptVersion{prev}, script.History...)
		// Trim to max history
		if len(script.History) > maxScriptHistory {
			script.History = script.History[:maxScriptHistory]
		}
	}

	opts := options.Replace().SetUpsert(true)
	_, err = e.db.Collection(gmScriptsCollection).ReplaceOne(ctx, bson.M{"_id": script.Filename}, script, opts)
	return err
}

// DeleteGMScript removes a GM script from MongoDB.
func (e *GameEngine) DeleteGMScript(ctx context.Context, filename string) error {
	if e.db == nil {
		return fmt.Errorf("database not available")
	}
	_, err := e.db.Collection(gmScriptsCollection).DeleteOne(ctx, bson.M{"_id": filename})
	return err
}

// LoadGMScripts loads all GM scripts from MongoDB and applies them to the engine.
// Called at startup after disk scripts are loaded.
func (e *GameEngine) LoadGMScripts() {
	if e.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Sort by priority desc (higher loads first), then by uploadedAt asc
	opts := options.Find().SetSort(bson.D{{Key: "priority", Value: -1}, {Key: "uploadedAt", Value: 1}})
	cursor, err := e.db.Collection(gmScriptsCollection).Find(ctx, bson.M{}, opts)
	if err != nil {
		log.Printf("Warning: could not load GM scripts: %v", err)
		return
	}
	var scripts []GMScript
	if err := cursor.All(ctx, &scripts); err != nil {
		log.Printf("Warning: could not decode GM scripts: %v", err)
		return
	}
	if len(scripts) == 0 {
		return
	}
	log.Printf("Loading %d GM scripts from MongoDB...", len(scripts))
	for _, s := range scripts {
		parsed, err := e.parseAndApplyScript(s.Content, s.Filename)
		if err != nil {
			log.Printf("Warning: GM script %q failed to parse: %v", s.Filename, err)
			continue
		}
		log.Printf("  %s (priority %d): %d rooms, %d items, %d monsters",
			s.Filename, s.Priority, parsed.Rooms, parsed.Items, parsed.Monsters)
	}
}

// parseAndApplyScript parses script content and applies it to the engine.
// Returns stats of what was applied. Imported here to avoid circular deps —
// caller must pass content, this calls out to scriptparser via a callback.
var ScriptParser func(content, filename string) (*gameworld.ParsedData, error)

func (e *GameEngine) parseAndApplyScript(content, filename string) (ScriptApplyStats, error) {
	if ScriptParser == nil {
		return ScriptApplyStats{}, fmt.Errorf("script parser not initialized")
	}
	parsed, err := ScriptParser(content, filename)
	if err != nil {
		return ScriptApplyStats{}, err
	}
	stats := e.ApplyParsedData(parsed)
	return stats, nil
}

// ScriptApplyStats summarizes what a hot-loaded script changed.
type ScriptApplyStats struct {
	Rooms     int `json:"rooms"`
	Items     int `json:"items"`
	Monsters  int `json:"monsters"`
	Nouns     int `json:"nouns"`
	Variables int `json:"variables"`
}

// ApplyParsedData hot-loads parsed script data into the running engine.
// Entities are added or overwritten by number. This is additive — it never removes entities.
func (e *GameEngine) ApplyParsedData(parsed *gameworld.ParsedData) ScriptApplyStats {
	var stats ScriptApplyStats

	for i := range parsed.Rooms {
		e.rooms[parsed.Rooms[i].Number] = &parsed.Rooms[i]
		stats.Rooms++
	}
	for i := range parsed.Items {
		e.items[parsed.Items[i].Number] = &parsed.Items[i]
		stats.Items++
	}
	for i := range parsed.Monsters {
		e.monsters[parsed.Monsters[i].Number] = &parsed.Monsters[i]
		stats.Monsters++
	}
	for i := range parsed.Nouns {
		e.nouns[parsed.Nouns[i].ID] = parsed.Nouns[i].Name
		stats.Nouns++
	}
	for i := range parsed.Adjectives {
		e.adjectives[parsed.Adjectives[i].ID] = parsed.Adjectives[i].Name
	}
	for i := range parsed.MonsterAdjs {
		e.monAdjs[parsed.MonsterAdjs[i].ID] = parsed.MonsterAdjs[i].Name
	}
	for _, v := range parsed.Variables {
		name := strings.ToUpper(v.Name)
		if !e.namedVarNames[name] {
			e.namedVarNames[name] = true
			e.NamedVars[name] = 0
			stats.Variables++
		}
	}
	if len(parsed.ForageDefs) > 0 {
		e.forageDefs = append(e.forageDefs, parsed.ForageDefs...)
	}
	if len(parsed.CEvents) > 0 {
		e.cevents = append(e.cevents, parsed.CEvents...)
	}

	if stats.Rooms > 0 || stats.Items > 0 || stats.Monsters > 0 {
		log.Printf("Hot-loaded script: %d rooms, %d items, %d monsters, %d nouns, %d vars",
			stats.Rooms, stats.Items, stats.Monsters, stats.Nouns, stats.Variables)
	}

	return stats
}

// loadPVals loads persistent global values from MongoDB.
func (e *GameEngine) loadPVals() {
	if e.db == nil {
		return
	}
	ctx := context.Background()
	var doc struct {
		Vals map[string]int `bson:"vals"`
	}
	err := e.db.Collection("pvals").FindOne(ctx, bson.M{"_id": "pvals"}).Decode(&doc)
	if err != nil {
		return // no saved pvals yet
	}
	for k, v := range doc.Vals {
		idx, err := strconv.Atoi(k)
		if err == nil {
			e.PVals[idx] = v
		}
	}
}

// savePVals saves persistent global values to MongoDB.
func (e *GameEngine) savePVals() {
	if e.db == nil {
		return
	}
	ctx := context.Background()
	vals := make(map[string]int)
	for k, v := range e.PVals {
		vals[strconv.Itoa(k)] = v
	}
	_, _ = e.db.Collection("pvals").ReplaceOne(ctx, bson.M{"_id": "pvals"}, bson.M{"_id": "pvals", "vals": vals}, options.Replace().SetUpsert(true))
}

// StartCEventLoop starts the background goroutine that fires cyclic events.
func (e *GameEngine) StartCEventLoop() {
	if len(e.cevents) == 0 {
		return
	}
	go func() {
		tick := 0
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			tick++
			for _, ce := range e.cevents {
				if ce.Cycles > 0 && tick%ce.Cycles == 0 {
					room := e.rooms[ce.Room]
					if room == nil {
						continue
					}
					sc := &ScriptContext{Room: room, Engine: e, Player: &Player{}}
					for _, block := range ce.Scripts {
						if block.Type == "ACTION" {
							for _, action := range block.Actions {
								sc.execAction(action)
							}
						} else {
							sc.execBlock(block)
						}
					}
					// Deliver ECHO messages from CEVENT scripts to players in the room
					if e.roomBroadcast != nil && len(sc.RoomMsgs) > 0 {
						e.roomBroadcast(ce.Room, sc.RoomMsgs)
					}
					e.Events.Publish("cevent", fmt.Sprintf("CEVENT %d fired in room %d (%s)", ce.ID, ce.Room, room.Name))
				}
			}
		}
	}()
}
