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
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MaxBankItems is the maximum number of items a player can store in their bank.
const MaxBankItems = 25

var validNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z'-]{2,19}$`) // min 3 chars total

// reservedExactNames are blocked as whole-word matches only.
// Monster/creature names and game terms — "Pendragon" is fine, "Dragon" is not.
var reservedExactNames = map[string]bool{
	// Monster/creature names
	"skeleton": true, "zombie": true, "ghoul": true, "vampire": true, "lich": true,
	"goblin": true, "ogre": true, "troll": true, "orc": true, "dragon": true,
	"demon": true, "devil": true, "golem": true, "elemental": true, "gargoyle": true,
	"spider": true, "rat": true, "wolf": true, "bear": true, "snake": true,
	"guard": true, "sentry": true, "bandit": true, "thief": true, "assassin": true,
	"mummy": true, "wraith": true, "spectre": true, "ghost": true, "wight": true,
	"banshee": true, "ursine": true, "giant": true, "drake": true, "wyrm": true,
	// Game terms
	"admin": true, "moderator": true, "gamemaster": true, "system": true, "server": true,
	"god": true, "goddess": true, "eternity": true, "legends": true,
}

// reservedSubstrings are blocked as substrings — these are slurs and profanity
// where even partial matches (e.g., in compound names) should be caught.
var reservedSubstrings = []string{
	"fuck", "shit", "cunt", "nigger", "nigga", "faggot",
	"nazi", "hitler",
}

// Resolve verb abbreviations to canonical full form for ALL verbs.
// This ensures IFPREVERB matching works regardless of abbreviation used.
// Direction shortcuts map to both full name (for scripts) and short form (for movement).
var dirFullNames = map[string]string{
	"N": "NORTH", "S": "SOUTH", "E": "EAST", "W": "WEST",
	"NE": "NORTHEAST", "NW": "NORTHWEST", "SE": "SOUTHEAST", "SW": "SOUTHWEST",
	"U": "UP", "D": "DOWN", "O": "OUT",
}

var dirMap = map[string]string{
	"N": "N", "NORTH": "N", "S": "S", "SOUTH": "S",
	"E": "E", "EAST": "E", "W": "W", "WEST": "W",
	"NE": "NE", "NORTHEAST": "NE", "NW": "NW", "NORTHWEST": "NW",
	"SE": "SE", "SOUTHEAST": "SE", "SW": "SW", "SOUTHWEST": "SW",
	"U": "U", "UP": "U", "D": "D", "DOWN": "D",
	"O": "O", "OUT": "O",
}

type VisibilityLevel int

const (
	VisibilityDark VisibilityLevel = iota
	VisibilityPartial
	VisibilityLimited
	VisibilityClear
)

// ValidateCharacterInput checks character creation parameters.
func ValidateCharacterInput(firstName, lastName string, race, gender int) error {
	if !validNamePattern.MatchString(firstName) {
		return fmt.Errorf("first name must be 1-20 letters (may include ' and -)")
	}
	if !validNamePattern.MatchString(lastName) {
		return fmt.Errorf("last name must be 1-20 letters (may include ' and -)")
	}
	fnLower := strings.ToLower(firstName)
	lnLower := strings.ToLower(lastName)

	// Exact match against reserved names (monster names, game terms)
	if reservedExactNames[fnLower] {
		return fmt.Errorf("that first name is reserved. Please choose another")
	}
	if reservedExactNames[lnLower] {
		return fmt.Errorf("that last name is reserved. Please choose another")
	}

	// Substring match for slurs/profanity only
	for _, word := range reservedSubstrings {
		if strings.Contains(fnLower, word) || strings.Contains(lnLower, word) {
			return fmt.Errorf("that name contains an inappropriate word. Please choose another")
		}
	}

	if race < 1 || race > 8 {
		return fmt.Errorf("invalid race")
	}
	if gender < 0 || gender > 1 {
		return fmt.Errorf("invalid gender")
	}
	return nil
}

// SessionProvider gives the engine read access to online session info.
type SessionProvider interface {
	// OnlinePlayers returns the list of currently connected players.
	OnlinePlayers() []*Player
}

// RoomChange describes a mutation to room state that must be synced across machines.
type RoomChange struct {
	RoomNumber int                 `json:"roomNumber"`
	Type       string              `json:"type"` // "item_state", "item_update", "item_add", "item_remove"
	ItemRef    int                 `json:"itemRef,omitempty"`
	Item       *gameworld.RoomItem `json:"item,omitempty"` // full item snapshot for item_add or item_update
	NewState   string              `json:"newState,omitempty"`
}

// RoomChangeCallback is called whenever room state is mutated locally.
type RoomChangeCallback func(change RoomChange)

// RoomBroadcastFunc sends messages to all players in a room (used by background tasks).
type RoomBroadcastFunc func(roomNumber int, messages []string)

// LocalRoomBroadcastFunc sends messages to players on THIS machine only (not via hub).
// Used for monster ambient text and combat which is per-machine.
type LocalRoomBroadcastFunc func(roomNumber int, messages []string)

// PlayerMessageFunc sends messages to a specific player by name (used by background tasks).
type PlayerMessageFunc func(playerName string, messages []string)

// GameEngine holds the loaded game world and processes commands.
type GameEngine struct {
	db                   *mongo.Database
	nouns                map[int]string
	adjectives           map[int]string
	monAdjs              map[int]string
	items                map[int]*gameworld.ItemDef
	traits               map[string]*gameworld.TraitDef
	rooms                map[int]*gameworld.Room
	monsters             map[int]*gameworld.MonsterDef
	startRoom            int
	departRoom           int // safe room for DEPART (bump room)
	sessions             SessionProvider
	onRoomChange         RoomChangeCallback
	roomBroadcast        RoomBroadcastFunc
	localRoomBroadcast   LocalRoomBroadcastFunc
	sendToPlayer         PlayerMessageFunc
	monsterMgr           *monsterManager
	RegionWeather        map[int]int                        // region -> weather state
	monsterLists         []gameworld.MonsterList            // base + current season MLISTs
	baseMonsterLists     []gameworld.MonsterList            // always-loaded MLISTs
	seasonalMonsterLists map[string][]gameworld.MonsterList // per-season MLISTs
	seasonalRooms        map[string][]gameworld.Room        // per-season room overrides
	currentSeason        string                             // current active season key
	cevents              []gameworld.CEvent
	baseForageDefs       []gameworld.ForageDef
	forageDefs           []gameworld.ForageDef
	seasonalForageDefs   map[string][]gameworld.ForageDef // per-season forage values
	mineDefs             []gameworld.MineDef              //MINING definitions GEMS AND MINERALS
	PVals                map[int]int                      // persistent global values
	NamedVars            map[string]int                   // VARIABLE-defined global named variables (DANWATER, etc.)
	namedVarNames        map[string]bool                  // set of valid named variable names
	Events               *EventBus
	Banner               string // active login banner; in-memory so it works even if MongoDB is down
	lastAssistName       string // last player who used ASSIST (for @answer)
	lastAssistRoom       int    // room number of last ASSIST
}

// SetSessionProvider sets the session provider (called by API layer after init).
func (e *GameEngine) SetSessionProvider(sp SessionProvider) {
	e.sessions = sp
}

// SetRoomChangeCallback sets the callback for cross-machine room state sync.
func (e *GameEngine) SetRoomChangeCallback(cb RoomChangeCallback) {
	e.onRoomChange = cb
}

// SetRoomBroadcast sets the function used by background tasks to send messages to rooms.
func (e *GameEngine) SetRoomBroadcast(fn RoomBroadcastFunc) {
	e.roomBroadcast = fn
}

// SetLocalRoomBroadcast sets a local-only broadcast (no hub). Used for monster activity.
func (e *GameEngine) SetLocalRoomBroadcast(fn LocalRoomBroadcastFunc) {
	e.localRoomBroadcast = fn
}

// SetSendToPlayer sets the function for sending targeted messages from background tasks.
func (e *GameEngine) SetSendToPlayer(fn PlayerMessageFunc) {
	e.sendToPlayer = fn
}

// GetBanner returns the current login banner (empty if none).
func (e *GameEngine) GetBanner() string {
	return e.Banner
}

// SetBanner sets the in-memory banner and persists it to MongoDB (best-effort).
func (e *GameEngine) SetBanner(text string) {
	e.Banner = text
	go e.saveBanner(text)
}

// LoadBanner loads the banner from MongoDB on startup (best-effort; in-memory is authoritative at runtime).
func (e *GameEngine) LoadBanner() {
	if e.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var doc struct {
		Text string `bson:"text"`
	}
	err := e.db.Collection("game_state").FindOne(ctx, bson.M{"_id": "banner"}).Decode(&doc)
	if err == nil && doc.Text != "" {
		e.Banner = doc.Text
		log.Printf("Loaded login banner: %q", e.Banner)
	}
}

func (e *GameEngine) saveBanner(text string) {
	if e.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if text == "" {
		e.db.Collection("game_state").DeleteOne(ctx, bson.M{"_id": "banner"})
	} else {
		opts := options.Replace().SetUpsert(true)
		e.db.Collection("game_state").ReplaceOne(ctx, bson.M{"_id": "banner"},
			bson.M{"_id": "banner", "text": text}, opts)
	}
}

// GMScript represents a GM-uploaded script stored in MongoDB.
type GMScript struct {
	Filename            string            `bson:"_id" json:"filename"`
	Name                string            `bson:"name" json:"name"`
	Content             string            `bson:"content" json:"content"`
	Priority            int               `bson:"priority" json:"priority"` // higher = loads sooner
	Size                int               `bson:"size" json:"size"`
	UploadedBy          string            `bson:"uploadedBy" json:"uploadedBy"`
	UploadedByAccountID string            `bson:"uploadedByAccountId" json:"uploadedByAccountId"`
	UploadedAt          time.Time         `bson:"uploadedAt" json:"uploadedAt"`
	ParseStats          ScriptApplyStats  `bson:"parseStats" json:"parseStats"`
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
	for i := range parsed.Traits {
		name := strings.ToUpper(parsed.Traits[i].Name)
		e.traits[name] = &parsed.Traits[i]
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

// notifyRoomChange fires the callback if set.
func (e *GameEngine) notifyRoomChange(change RoomChange) {
	if e.onRoomChange != nil {
		e.onRoomChange(change)
	}
	if e.Events != nil {
		e.Events.Publish("world", fmt.Sprintf("Room %d: %s (ref %d, state: %s)", change.RoomNumber, change.Type, change.ItemRef, change.NewState))
	}
}

// ApplyRoomChange applies a remote room state change from another machine.
func (e *GameEngine) ApplyRoomChange(change RoomChange) {
	room := e.rooms[change.RoomNumber]
	if room == nil {
		return
	}
	switch change.Type {
	case "item_state":
		for i := range room.Items {
			if room.Items[i].Ref == change.ItemRef && !room.Items[i].IsPut {
				room.Items[i].State = change.NewState
				break
			}
		}
	case "item_update":
		// Full item snapshot update (state, vals, adjs, etc.)
		if change.Item != nil {
			for i := range room.Items {
				if room.Items[i].Ref == change.ItemRef && !room.Items[i].IsPut {
					room.Items[i] = *change.Item
					break
				}
			}
		}
	case "item_add":
		if change.Item != nil {
			room.Items = append(room.Items, *change.Item)
		}
	case "item_remove":
		for i := range room.Items {
			if room.Items[i].Ref == change.ItemRef && !room.Items[i].IsPut {
				room.Items = append(room.Items[:i], room.Items[i+1:]...)
				break
			}
		}
	case "named_var":
		// Sync a named variable from another machine: "VARNAME=VALUE"
		if change.NewState != "" {
			parts := strings.SplitN(change.NewState, "=", 2)
			if len(parts) == 2 {
				name := parts[0]
				val, _ := strconv.Atoi(parts[1])
				if e.namedVarNames[name] {
					e.NamedVars[name] = val
				}
			}
		}
	}
}

// NewGameEngine creates an engine with lookups from parsed data.
func NewGameEngine(db *mongo.Database, parsed *gameworld.ParsedData) *GameEngine {
	e := &GameEngine{
		db:         db,
		nouns:      make(map[int]string),
		adjectives: make(map[int]string),
		monAdjs:    make(map[int]string),
		items:      make(map[int]*gameworld.ItemDef),
		traits:     make(map[string]*gameworld.TraitDef),
		rooms:      make(map[int]*gameworld.Room),
		monsters:   make(map[int]*gameworld.MonsterDef),
		startRoom:  parsed.StartRoom,
		departRoom: parsed.BumpRoom,
	}
	for i := range parsed.Nouns {
		e.nouns[parsed.Nouns[i].ID] = parsed.Nouns[i].Name
	}
	for i := range parsed.Adjectives {
		e.adjectives[parsed.Adjectives[i].ID] = parsed.Adjectives[i].Name
	}
	for i := range parsed.MonsterAdjs {
		e.monAdjs[parsed.MonsterAdjs[i].ID] = parsed.MonsterAdjs[i].Name
	}
	for i := range parsed.Items {
		e.items[parsed.Items[i].Number] = &parsed.Items[i]
	}
	for i := range parsed.Traits {
		name := strings.ToUpper(parsed.Traits[i].Name)
		e.traits[name] = &parsed.Traits[i]
	}
	for i := range parsed.Rooms {
		e.rooms[parsed.Rooms[i].Number] = &parsed.Rooms[i]
	}
	for i := range parsed.Monsters {
		e.monsters[parsed.Monsters[i].Number] = &parsed.Monsters[i]
	}

	// Load forage definitions
	e.forageDefs = parsed.ForageDefs
	e.baseForageDefs = e.forageDefs
	e.seasonalForageDefs = parsed.SeasonalForageDefs

	// Load mining definitions
	e.mineDefs = parsed.MineDefs

	// Initialize event bus for admin monitoring
	e.Events = NewEventBus()

	// Load persisted game time from MongoDB (must happen before season determination)
	LoadGameTime(db)

	// Initialize monster manager with season-aware MLIST selection
	e.monsterMgr = newMonsterManager()
	e.baseMonsterLists = parsed.MonsterLists
	e.seasonalMonsterLists = parsed.SeasonalMonsterLists
	e.seasonalRooms = parsed.SeasonalRooms
	e.currentSeason = GameSeason()
	e.applySeasonalRooms()
	e.forageDefs = e.buildActiveForageDefs()
	e.monsterLists = e.buildActiveMonsterLists()
	count := e.monsterMgr.SpawnInitialMonsters(e.monsterLists, e.monsters)
	log.Printf("Season: %s (%s). Base MLISTs: %d, Seasonal: %d, Total: %d",
		SeasonName(), e.currentSeason, len(e.baseMonsterLists),
		len(e.seasonalMonsterLists[e.currentSeason]), len(e.monsterLists))
	e.Events.Publish("monster", fmt.Sprintf("Spawned %d monsters across the world (season: %s)", count, SeasonName()))

	// Initialize weather (all regions sunny)
	e.RegionWeather = make(map[int]int)

	// Initialize named variables from VARIABLE definitions (all default to 0)
	e.NamedVars = make(map[string]int)
	e.namedVarNames = make(map[string]bool)
	for _, v := range parsed.Variables {
		name := strings.ToUpper(v.Name)
		e.namedVarNames[name] = true
		e.NamedVars[name] = 0
	}

	// Store CEvents
	e.cevents = parsed.CEvents

	// Initialize PVals
	e.PVals = make(map[int]int)
	e.loadPVals()

	return e
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

			// Process player timers/status effects.
			if e.sessions != nil {
				for _, player := range e.sessions.OnlinePlayers() {
					messages := e.UpdatePlayerTimers(player)

					if len(messages) > 0 && e.sendToPlayer != nil {
						e.sendToPlayer(player.FirstName, messages)
					}
				}
			}

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

// CommandResult is what gets sent back to the client.
type CommandResult struct {
	Messages         []string `json:"messages"`
	RoomName         string   `json:"roomName,omitempty"`
	RoomDesc         string   `json:"roomDesc,omitempty"`
	Exits            []string `json:"exits,omitempty"`
	Items            []string `json:"items,omitempty"`
	Error            string   `json:"error,omitempty"`
	Quit             bool     `json:"quit,omitempty"`
	PromptIndicators string   `json:"promptIndicators,omitempty"`
	PlayerState      *Player  `json:"playerState,omitempty"`

	// GMCP: room exits as direction→roomNumber map for automapper
	RoomExits   map[string]int `json:"-"`
	RoomTerrain string         `json:"-"`
	RoomRegion  int            `json:"-"`

	// Multiplayer: messages broadcast to others in the same room.
	// OldRoom is set on movement to broadcast departure to the room left.
	RoomBroadcast []string `json:"-"`
	OldRoom       int      `json:"-"`
	OldRoomMsg    []string `json:"-"`
	// Whisper: targeted message to a specific player (only they see the content).
	WhisperTarget string `json:"-"`
	WhisperMsg    string `json:"-"`
	// TargetMsg: second-person message sent to the emote target (they see "X kicks you."
	// instead of the RoomBroadcast). The target is excluded from RoomBroadcast.
	TargetName string   `json:"-"`
	TargetMsg  []string `json:"-"`
	// GlobalBroadcast: sent to all online players.
	GlobalBroadcast []string `json:"-"`
	// GMBroadcast: sent to all online GMs.
	GMBroadcast []string `json:"-"`
	// TelepathyMsg: telepathy message to send to telepathy-enabled players.
	TelepathyMsg    string `json:"-"`
	TelepathySender string `json:"-"`
	// CantMsg: thieves' cant — delivered only to players with Stealth/Legerdemain.
	CantMsg    string `json:"-"`
	CantSender string `json:"-"`
	// LogEvent: optional event to log (type, detail).
	LogEventType   string `json:"-"`
	LogEventDetail string `json:"-"`

	MagicPower int `json:"-"`
}

// extractOriginalArgs returns the original-case text after the first word of input.
func extractOriginalArgs(input string) string {
	trimmed := strings.TrimSpace(input)
	idx := strings.IndexByte(trimmed, ' ')
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(trimmed[idx+1:])
}

const maxInputLength = 500

// ProcessCommand parses and executes a player command.
func (e *GameEngine) ProcessCommand(ctx context.Context, player *Player, input string) *CommandResult {
	input = strings.TrimSpace(input)
	if input == "" {
		return &CommandResult{Messages: []string{"What would you like to do?"}}
	}
	if len(input) > maxInputLength {
		input = input[:maxInputLength]
	}

	// Clean up stale follow state — if leader is no longer online, clear Following
	if player.Following != "" && e.sessions != nil {
		leaderOnline := false
		for _, p := range e.sessions.OnlinePlayers() {
			if p.FirstName == player.Following {
				leaderOnline = true
				break
			}
		}
		if !leaderOnline {
			e.removeFromGroup(player)
		}
	}

	// Dead players can only DEPART, LOOK, WHO, QUIT, EXP, STATUS, HEALTH
	if player.Dead {
		verb := strings.ToUpper(strings.Fields(input)[0])
		switch verb {
		case "DEPART", "LOOK", "WHO", "QUIT", "EXP", "EXPERIENCE", "STATUS", "HEALTH", "HELP":
			// allowed — fall through to normal processing
		default:
			return &CommandResult{Messages: []string{"You are dead and can't do much of anything. Type DEPART to allow Eternity, Inc. to retrieve you."}}
		}
	}

	// Handle speech
	if strings.HasPrefix(input, "'") || strings.HasPrefix(input, "\"") {
		msg := input[1:]
		verb := "say"
		thirdVerb := "says"
		if strings.HasSuffix(msg, "?") {
			verb = "ask"
			thirdVerb = "asks"
		} else if strings.HasSuffix(msg, "!") {
			verb = "exclaim"
			thirdVerb = "exclaims"
		}
		// Custom speech pattern overrides the verb (e.g., "says grimly", "squawks")
		if player.SpeechAdverb != "" {
			result := &CommandResult{
				Messages:      []string{fmt.Sprintf("You %s, \"%s\"", player.SpeechAdverb, msg)},
				RoomBroadcast: []string{fmt.Sprintf("%s %ss, \"%s\"", player.FirstName, player.SpeechAdverb, msg)},
			}
			// Run IFSAY scripts
			room := e.rooms[player.RoomNumber]
			if room != nil {
				sc := e.RunSayScripts(player, room, msg)
				if len(sc.Messages) > 0 {
					result.Messages = append(result.Messages, sc.Messages...)
				}
				if len(sc.RoomMsgs) > 0 {
					result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
				}
			}
			return result
		}
		result := &CommandResult{
			Messages:      []string{fmt.Sprintf("You %s, \"%s\"", verb, msg)},
			RoomBroadcast: []string{fmt.Sprintf("%s %s, \"%s\"", player.FirstName, thirdVerb, msg)},
		}
		// Run IFSAY scripts
		room := e.rooms[player.RoomNumber]
		if room != nil {
			sc := e.RunSayScripts(player, room, msg)
			if len(sc.Messages) > 0 {
				result.Messages = append(result.Messages, sc.Messages...)
				e.Events.Publish("script", fmt.Sprintf("IFSAY triggered by %s in room %d: \"%s\"", player.FirstName, room.Number, msg))
			}
			if len(sc.RoomMsgs) > 0 {
				result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
			}
			if len(sc.GMMsgs) > 0 {
				result.GMBroadcast = append(result.GMBroadcast, sc.GMMsgs...)
			}
		}
		return result
	}

	parts := strings.Fields(strings.ToUpper(input))
	if len(parts) == 0 {
		return &CommandResult{Messages: []string{"What would you like to do?"}}
	}

	verb := parts[0]
	args := parts[1:]

	// GM commands (@ prefix) — silent fail if not GM, also check bot GM permission
	if strings.HasPrefix(verb, "@") {
		if !player.IsGM {
			return &CommandResult{Messages: []string{fmt.Sprintf("I don't understand \"%s\". Type HELP for commands.", strings.ToLower(input))}}
		}
		if player.IsBot && !player.BotGMAllowed {
			return &CommandResult{Messages: []string{"This bot does not have permission to use GM commands."}}
		}
		return e.processGMCommand(ctx, player, verb, args, input)
	}

	// Resolve the verb to its full canonical name for script matching
	canonicalVerb := verb
	if full, ok := dirFullNames[verb]; ok {
		canonicalVerb = full
	}

	if dir, ok := dirMap[verb]; ok {
		// Check for IFPREVERB scripts on the direction using canonical verb name
		room := e.rooms[player.RoomNumber]

		if room != nil {
			sc := &ScriptContext{
				Player: player,
				Room:   room,
				Engine: e,
			}

			for _, block := range room.Scripts {
				if block.Type == "IFPREVERB" && len(block.Args) >= 2 {
					if strings.ToUpper(block.Args[0]) == canonicalVerb &&
						block.Args[1] == "-1" {

						sc.execBlock(block)
					}
				}
			}

			// MOVEGROUP: move all players in this room to destination
			if sc.MoveGroupTo > 0 {
				e.moveGroupToRoom(
					ctx,
					player.RoomNumber,
					sc.MoveGroupTo,
				)
			}

			if sc.Blocked || sc.MoveTo > 0 {
				result := &CommandResult{}

				result.Messages = append(
					result.Messages,
					sc.Messages...,
				)

				result.RoomBroadcast = append(
					result.RoomBroadcast,
					sc.RoomMsgs...,
				)

				// --------------------------------------------------------
				// SCRIPT MOVE
				// --------------------------------------------------------
				if sc.MoveTo > 0 {
					dest := e.rooms[sc.MoveTo]

					if dest != nil {
						oldRoom := player.RoomNumber

						player.RoomNumber = sc.MoveTo
						e.SavePlayer(ctx, player)

						// Bring commanded followers with the player.
						e.moveCommandedFollowers(
							player,
							oldRoom,
							sc.MoveTo,
						)

						lookResult := e.doLook(player)

						result.Messages = append(
							result.Messages,
							lookResult.Messages...,
						)

						result.RoomName = lookResult.RoomName
						result.RoomDesc = lookResult.RoomDesc
						result.Exits = lookResult.Exits
						result.Items = lookResult.Items

						result.OldRoom = oldRoom

						result.OldRoomMsg = []string{
							fmt.Sprintf(
								"%s leaves.",
								player.FirstName,
							),
						}

						result.RoomBroadcast = append(
							result.RoomBroadcast,
							fmt.Sprintf(
								"%s arrives.",
								player.FirstName,
							),
						)

						e.applyEntryScripts(
							ctx,
							player,
							dest,
							result,
						)
					}
				}

				if len(result.Messages) == 0 {
					result.Messages = []string{
						"You can't go that way.",
					}
				}

				return result
			}

			// Scripts ran but didn't block — proceed with normal movement.
			if len(sc.Messages) > 0 {
				moveResult := e.doMove(
					ctx,
					player,
					dir,
				)

				moveResult.Messages = append(
					sc.Messages,
					moveResult.Messages...,
				)

				return moveResult
			}
		}

		return e.doMove(
			ctx,
			player,
			dir,
		)
	}
	// Resolve verb abbreviations — try exact match first, then unique prefix
	verb = resolveVerb(verb)

	switch verb {
	case "LOOK", "EXAMINE", "INSPECT":
		if len(args) == 0 {
			return e.doLookFull(player)
		}
		return e.doLookAt(player, args)
	case "GO":
		return e.doGo(ctx, player, args)
	case "CLIMB":
		return e.doClimb(ctx, player, args)
	case "GET", "TAKE":
		result := e.doGet(ctx, player, args)
		e.RefreshEncumbrance(player)
		return result
	case "DOATEST":
		return e.doATest(ctx, player)
	case "DROP":
		result := e.doDrop(ctx, player, args)
		e.RefreshEncumbrance(player)
		return result
	case "INVENTORY":
		result := e.doInventory(player)
		e.RefreshEncumbrance(player)
		return result
	case "STATUS":
		if len(args) > 0 {
			t := strings.ToLower(strings.Join(args, " "))
			if t == "me" || t == "myself" || t == "self" {
				return e.doStatus(player)
			}
			if found := e.findPlayerInRoom(player, t); found != nil {
				return e.doStatus(found)
			}
			return &CommandResult{Messages: []string{"You don't see that person here."}}
		}
		return e.doStatus(player)
	case "HEAL":
		return e.doTend(ctx, player, args)
	case "HEALTH", "DIAGNOSE":
		if len(args) > 0 {
			t := strings.ToLower(strings.Join(args, " "))
			if t == "me" || t == "myself" || t == "self" {
				return e.doHealth(player)
			}
			if found := e.findPlayerInRoom(player, t); found != nil {
				return e.doHealth(found)
			}
			return &CommandResult{Messages: []string{"You don't see that person here."}}
		}
		return e.doHealth(player)
	case "WIELD":
		result := e.doWield(ctx, player, args)
		e.RefreshEncumbrance(player)
		return result
	case "UNWIELD":
		result := e.doUnwield(ctx, player, args)
		e.RefreshEncumbrance(player)
		return result
	case "WEAR":
		result := e.doWear(ctx, player, args)
		e.RefreshEncumbrance(player)
		return result
	case "REMOVE":
		result := e.doRemove(ctx, player, args)
		e.RefreshEncumbrance(player)
		return result
	case "OPEN":
		return e.doOpen(player, args)
	case "CLOSE":
		return e.doClose(player, args)
	case "SIT":
		if player.Position == 1 {
			return &CommandResult{Messages: []string{"You are already sitting."}}
		}
		player.Position = 1
		return e.doPositionWithScripts(ctx, player, verb, "You sit down.", fmt.Sprintf("%s sits down.", player.FirstName))
	case "STAND":
		if player.Position == 0 {
			return &CommandResult{Messages: []string{"You are already standing."}}
		}
		player.Position = 0
		return e.doPositionWithScripts(ctx, player, verb, "You stand up.", fmt.Sprintf("%s stands up.", player.FirstName))
	case "KNEEL":
		if player.Position == 3 {
			return &CommandResult{Messages: []string{"You are already kneeling."}}
		}
		player.Position = 3
		return e.doPositionWithScripts(ctx, player, verb, "You kneel down.", fmt.Sprintf("%s kneels down.", player.FirstName))
	case "LAY":
		if player.Position == 2 {
			return &CommandResult{Messages: []string{"You are already lying down."}}
		}
		player.Position = 2
		return e.doPositionWithScripts(ctx, player, verb, "You lie down.", fmt.Sprintf("%s lies down.", player.FirstName))
	case "PRAY":
		return e.doPray(player)
	case "BRIEF":
		player.BriefMode = true
		return &CommandResult{Messages: []string{"Brief mode on."}}
	case "FULL":
		player.BriefMode = false
		return &CommandResult{Messages: []string{"Full descriptions on."}}
	case "PROMPT":
		player.PromptMode = !player.PromptMode
		if player.PromptMode {
			return &CommandResult{Messages: []string{"Status prompt on."}}
		}
		return &CommandResult{Messages: []string{"Status prompt off."}}
	case "WHO":
		return e.doWho(player)
	case "SKILLS":
		var skillMsgs []string
		skillMsgs = append(skillMsgs, "=== Your Skills ===")
		hasSkills := false
		for id := 0; id <= 35; id++ {
			lvl := player.Skills[id]
			if lvl > 0 {
				name := SkillNames[id]
				if name == "" {
					name = fmt.Sprintf("Skill #%d", id)
				}
				skillMsgs = append(skillMsgs, fmt.Sprintf("  %s: rank %d", name, lvl))
				hasSkills = true
			}
		}
		if !hasSkills {
			skillMsgs = append(skillMsgs, "  You have no trained skills yet.")
		}
		skillMsgs = append(skillMsgs, fmt.Sprintf("Build Points: %d", player.BuildPoints))
		return &CommandResult{Messages: skillMsgs}
	case "WEALTH":
		g := player.Gold
		s := player.Silver
		c := player.Copper
		return &CommandResult{Messages: []string{fmt.Sprintf("You have %d gold crowns, %d silver shillings, and %d copper pennies.", g, s, c)}}
	case "COUNT":
		if len(args) > 0 && strings.ToUpper(args[0]) == "MONEY" {
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You count your money. You have %d gold crowns, %d silver shillings, and %d copper pennies.", player.Gold, player.Silver, player.Copper)},
				RoomBroadcast: []string{fmt.Sprintf("%s counts %s money.", player.FirstName, player.Possessive())},
			}
		}
		return &CommandResult{Messages: []string{"Count what?"}}
	case "EXPERIENCE", "EXP":
		// Recalc to make sure BP is current
		recalcBuildPoints(player)
		spentBP := playerBPSpent(player)
		totalBP := player.BuildPoints + spentBP
		xpUntilNext := xpUntilNextBuildPoint(player)

		// Build points
		//	totalBP := player.BuildPoints
		//	spentBP := playerBPSpent(player)
		unspentBP := totalBP - spentBP

		return &CommandResult{Messages: []string{
			fmt.Sprintf("Experience: %d", player.Experience),
			fmt.Sprintf("Build Points to date: %d", totalBP),
			fmt.Sprintf("Unspent Build Points: %d", unspentBP),
			fmt.Sprintf("Experience Points until next Build Point: %d", xpUntilNext),
		}}
	case "INFO":
		return e.doInfo(player)
	case "REROLL":
		return e.doRerollStats(ctx, player, args)
	case "TIME":
		period := "day"
		if IsNight() {
			period = "night"
		}
		moonPhases := []string{"new", "waxing crescent", "first quarter", "waxing gibbous", "full", "waning gibbous", "last quarter", "waning crescent"}
		greatMoon := moonPhases[GameDay()%8]
		phulcrus := moonPhases[(GameDay()+4)%8]
		return &CommandResult{Messages: []string{
			fmt.Sprintf("It is %s %d, %d (Year of the Wyrm).", GameMonthName(), GameDay()%28+1, GameYear()),
			fmt.Sprintf("It is %s. The season is %s.", period, SeasonName()),
			fmt.Sprintf("The Great Moon is %s and Phulcrus is %s.", greatMoon, phulcrus),
		}}
	case "WHISPER":
		return e.doWhisper(player, args, input)
	case "CONTACT":
		return e.doContact(player, args, input)
	case "YELL":
		return e.doYell(player, args, input)
	case "GIVE":
		result := e.doGive(ctx, player, args)
		e.RefreshEncumbrance(player)
		return result
	case "PICK", "LOCKPICK":
		return e.doPick(ctx, player, args)
	case "EAT":
		result := e.doEat(ctx, player, args)
		e.RefreshEncumbrance(player)
		return result
	case "SPEECH":
		return &CommandResult{Messages: []string{"Speech patterns are set by gamemasters. Ask a GM if you'd like a custom speech style."}}
	case "QUIT":
		return &CommandResult{Messages: []string{"You fade from the Shattered Realms..."}, Quit: true,
			GlobalBroadcast: []string{fmt.Sprintf("** %s has just left the Realms.", player.FirstName)}}
	case "HELP":
		return e.doHelp()
	case "ADVICE":
		return &CommandResult{Messages: []string{
			"Welcome, adventurer! Here are some tips:",
			"- Use LOOK to examine your surroundings",
			"- Move with N, S, E, W, NE, NW, SE, SW, or GO <portal>",
			"- GET and DROP items, WIELD weapons, WEAR armor",
			"- Check your STATUS, HEALTH, INVENTORY, and WEALTH",
			"- Type HELP for a full command list",
		}}
	case "HOLD":
		// HOLD <player> → group hold; otherwise fallthrough to emote
		if len(args) > 0 {
			target := strings.ToLower(strings.Join(args, " "))
			if found := e.findPlayerInRoom(player, target); found != nil {
				return e.doHold(player, found)
			}
		}
		return e.processEmote(player, verb, args)
	case "SING":
		// If args provided, treat as SAY variant with sing/sings
		if len(args) > 0 {
			text := extractOriginalArgs(input)
			adverb := ""
			if player.SpeechAdverb != "" {
				adverb = player.SpeechAdverb + " "
			}
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You %ssing, \"%s\"", adverb, text)},
				RoomBroadcast: []string{fmt.Sprintf("%s %ssings, \"%s\"", player.FirstName, adverb, text)},
			}
		}
		return e.processEmote(player, verb, args)
	case "PLAY":
		// If wielding an instrument, produce special music output
		if player.Wielded != nil {
			wDef := e.items[player.Wielded.Archetype]
			if wDef != nil {
				wieldedNoun := strings.ToLower(e.getItemNounName(wDef))
				wieldedFullName := e.formatItemName(wDef, player.Wielded.Adj1, player.Wielded.Adj2, player.Wielded.Adj3)
				instruments := []string{"harp", "lyre", "violin", "flute", "drum", "horn", "lute"}
				for _, inst := range instruments {
					if strings.Contains(wieldedNoun, inst) || strings.Contains(strings.ToLower(wieldedFullName), inst) {
						return &CommandResult{
							Messages:      []string{fmt.Sprintf("You play your %s, filling the air with beautiful music.", wieldedFullName)},
							RoomBroadcast: []string{fmt.Sprintf("%s plays %s, filling the air with beautiful music.", player.FirstName, wieldedFullName)},
						}
					}
				}
			}
		}
		return e.processEmote(player, verb, args)
	case "TAP":
		// If wielding a staff in a dark room, produce light flavor text
		if player.Wielded != nil {
			wDef := e.items[player.Wielded.Archetype]
			if wDef != nil {
				wieldedNoun := strings.ToLower(e.getItemNounName(wDef))
				wieldedFullName := e.formatItemName(wDef, player.Wielded.Adj1, player.Wielded.Adj2, player.Wielded.Adj3)
				if strings.Contains(wieldedNoun, "staff") || strings.Contains(strings.ToLower(wieldedFullName), "staff") {
					room := e.rooms[player.RoomNumber]
					isDark := room != nil && (room.Terrain == "CAVE" || room.Terrain == "DEEPCAVE" || room.Terrain == "UNDERGROUND")
					if isDark {
						return &CommandResult{
							Messages: []string{
								"You rap your staff on the ground.",
								"A small orb of light appears and floats beside you.",
								// NOTE: full light system integration pending — this is flavor text only for now
							},
							RoomBroadcast: []string{fmt.Sprintf("%s raps %s staff on the ground. A small orb of light appears.", player.FirstName, player.Possessive())},
						}
					}
				}
			}
		}
		// If args provided, check item interaction first
		if len(args) > 0 {
			return e.doItemInteraction(ctx, player, "TAP", args)
		}
		return e.processEmote(player, verb, args)
	// Roleplay verbs — dispatched via emote table
	case "SMILE", "BOW", "CURTSEY", "CURTSY", "WAVE", "NOD", "LAUGH", "CHUCKLE",
		"GRIN", "FROWN", "SIGH", "SHRUG", "WINK", "CRY", "DANCE",
		"HUG", "KISS", "POKE", "TICKLE", "SLAP", "HOWL",
		"PACE", "FIDGET", "SHIVER", "SNORT", "GROAN", "MUMBLE",
		"BABBLE", "BEAM", "SWOON", "TOAST", "SHUDDER", "POINT",
		"KICK", "KNOCK", "PET", "PUNCH", "SPIT",
		"GAZE", "GLARE", "SCOWL", "COMFORT", "YAWN",
		"BLINK", "BLUSH", "CRINGE", "CUDDLE", "COUGH", "FURROW",
		"GASP", "GIGGLE", "GRIMACE", "GROWL", "GULP", "JUMP",
		"LEAN", "NUZZLE", "PANT", "PONDER", "POUT", "ROLL",
		"SCREAM", "SMIRK", "SNICKER", "SALUTE", "STRETCH",
		"TWIRL", "WINCE", "WHISTLE", "MUTTER", "CARESS", "NUDGE",
		"ARCH", "RAISE", "HEAD", "SCRATCH", "CLAP",
		// Additional emotes
		"LICK", "NIBBLE", "BARK", "CLAW", "CURSE", "DUCK", "HISS",
		"HULA", "JIG", "MOAN", "MASSAGE", "PINCH",
		"PURR", "ROAR", "SNARL", "SNUGGLE", "WAG", "WAIT", "WRITE",
		"YOWL", "STOMP", "APPLAUD", "PEER", "GRUNT", "DIP",
		"HANDRAISE", "HANDSHAKE", "HEADSHAKE", "GESTURE",
		// Additional self-emotes
		"FUME", "SQUINT", "HUM", "SNIFFLE", "SLOUCH", "SNORE", "SNEEZE",
		"STARE", "PUCKER", "CRACK", "BOUNCE", "STRIKE", "CLUTCH",
		"WIPE", "GRIT", "TOSS", "ATTENTION", "TONGUE", "WRINKLE", "PUFF",
		"DIZZY", "BAT",
		// Race-specific emotes (handled by race check in processEmote)
		"FLICK", "BARE", "SPREAD", "FOLD", "SWISH",
		"RUBEARS", "PULLBEARD", "SCENT", "WHINE", "DROOP", "CHASE":

		if len(args) > 0 {
			return e.doItemInteraction(ctx, player, verb, args)
		}
		return e.processEmote(player, verb, args)
	case "ACT":
		if len(args) == 0 {
			return &CommandResult{Messages: []string{"Act how?"}}
		}
		action := extractOriginalArgs(input)
		var actMsg string
		if player.ActBrief {
			actMsg = fmt.Sprintf("%s %s", player.FirstName, action)
		} else {
			actMsg = fmt.Sprintf("(%s %s)", player.FirstName, action)
		}
		return &CommandResult{Messages: []string{actMsg}, RoomBroadcast: []string{actMsg}}
	case "EMOTE":
		if player.Race != RaceMechanoid {
			return &CommandResult{Messages: []string{"Only mechanoids can toggle their emotional state."}}
		}
		if player.Emotional {
			return &CommandResult{Messages: []string{"You are already in emotional mode."}}
		}
		player.Emotional = true
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"You engage your emotional subroutines."}}
	case "UNEMOTE":
		if player.Race != RaceMechanoid {
			return &CommandResult{Messages: []string{"Only mechanoids can toggle their emotional state."}}
		}
		if !player.Emotional {
			return &CommandResult{Messages: []string{"Your emotional subroutines are already disengaged."}}
		}
		player.Emotional = false
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"You disengage your emotional subroutines."}}
	case "RECITE":
		if len(args) == 0 {
			return &CommandResult{Messages: []string{"Recite what?"}}
		}
		text := extractOriginalArgs(input)
		text = strings.Trim(text, "'\"")
		// Support \ as line break for poetry/songs
		lines := strings.Split(text, "\\")
		var selfMsgs, roomMsgs []string
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if i == 0 {
				selfMsgs = append(selfMsgs, fmt.Sprintf("You recite, '%s", line))
				roomMsgs = append(roomMsgs, fmt.Sprintf("%s recites, '%s", player.FirstName, line))
			} else {
				selfMsgs = append(selfMsgs, fmt.Sprintf("  %s", line))
				roomMsgs = append(roomMsgs, fmt.Sprintf("  %s", line))
			}
		}
		if len(selfMsgs) > 0 {
			selfMsgs[len(selfMsgs)-1] += "'"
			roomMsgs[len(roomMsgs)-1] += "'"
		}
		return &CommandResult{Messages: selfMsgs, RoomBroadcast: roomMsgs}
	case "READ":
		return e.doRead(player, args)
	case "SEARCH":
		if len(args) > 0 {
			// Try to search a dead monster first
			if result := e.doSearchMonster(ctx, player, args); result != nil {
				return result
			}
			return e.doItemInteraction(ctx, player, verb, args)
		}

		// ------------------------------------------------------------
		// Bare SEARCH: run IFPREVERB SEARCH -1 first.
		// ------------------------------------------------------------
		room := e.rooms[player.RoomNumber]

		if room != nil {
			sc := e.RunPreverbScripts(player, room, "SEARCH", nil, nil)

			// CLEARVERB means the script replaces normal SEARCH.
			if sc.Blocked {
				e.SavePlayer(ctx, player)

				return &CommandResult{
					Messages:      sc.Messages,
					RoomBroadcast: sc.RoomMsgs,
					GMBroadcast:   sc.GMMsgs,
					PlayerState:   player,
				}
			}
		}

		// Bare SEARCH: scan the area for hidden players
		player.RoundTime = 5
		player.RoundTimeExpiry = time.Now().Add(5 * time.Second)

		msgs := []string{"You search the area.", "[Round: 5 sec]"}
		perceptionCheck := player.EffectiveStat(StatPerception) + player.Skills[33]*5 // Stealth skill helps detection
		var revealed []string
		if e.sessions != nil {
			for _, p := range e.sessions.OnlinePlayers() {
				if p.RoomNumber == player.RoomNumber && p.Hidden && !p.EtherealActive && p.FirstName != player.FirstName {
					// Perception vs their stealth
					stealthRating := p.Agility + p.Skills[33]*5
					if rand.Intn(100)+perceptionCheck > stealthRating {
						p.Hidden = false
						revealed = append(revealed, p.FirstName)
					}
				}
			}
		}
		if len(revealed) > 0 {
			for _, name := range revealed {
				msgs = append(msgs, fmt.Sprintf("You discover %s hiding here!", name))
			}
		}
		return &CommandResult{Messages: msgs}
	case "PULL", "PUSH", "RUB", "TOUCH", "DIG", "USE", "THUMP", "CONCENTRATE":
		result := e.doItemInteraction(ctx, player, verb, args)
		// If item interaction found nothing, fall back to emote for verbs that have emote entries
		if result != nil && len(result.Messages) > 0 && result.Messages[0] != "You don't see that here." {
			return result
		}
		if verb == "THUMP" {
			return e.processEmote(player, verb, args)
		}
		if verb == "CONCENTRATE" {
			return &CommandResult{Messages: []string{"You concentrate deeply."}}
		}

		return result
	case "TURN":
		if result := e.doTurnPage(ctx, player, args); result != nil {
			return result
		}
		return e.doItemInteraction(ctx, player, verb, args)
	case "RECALL":
		if len(args) == 0 {
			return e.doRoomRecall(player)
		}
		return e.doItemInteraction(ctx, player, verb, args)
	//case "CONCENTRATE":

	//	return &CommandResult{Messages: []string{"You concentrate deeply."}}
	case "BUY", "ORDER":
		return e.doBuy(ctx, player, args)
	case "SELL":
		return e.doSell(ctx, player, args)
	case "APPRAISE":
		return e.doAppraise(player, args)
	case "DRINK", "SIP":
		return e.doDrink(ctx, player, args)
	case "LIGHT":
		return e.doLight(ctx, player, args, true)
	case "EXTINGUISH", "DOUSE":
		return e.doLight(ctx, player, args, false)
	case "FLIP":
		return e.doFlip(ctx, player, args)
	case "LATCH":
		return e.doLatch(player, args, true)
	case "UNLATCH":
		return e.doLatch(player, args, false)
	case "DEPOSIT":
		return e.doDeposit(ctx, player, args)
	case "WITHDRAW":
		return e.doWithdraw(ctx, player, args)
	case "TRAIN":
		return e.doTrainWithBP(ctx, player, args)
	case "MINE":
		return e.doMineReal(ctx, player)
	case "FORAGE":
		return e.doForageReal(ctx, player)
	case "SMELT":
		return e.doSmelt(ctx, player, args)
	case "CRAFT", "FORGE":
		return e.doCraft(ctx, player, args)
	case "DYE":
		return e.doDye(ctx, player, args)
	case "BREW":
		return e.doBrew(ctx, player, args)
	case "ANALYZE":
		return e.doAnalyze(ctx, player, args)
	case "WEAVE":
		return e.doCraft(ctx, player, args) // weave uses same craft logic at LOOM
	case "WORK":
		return e.doWork(ctx, player, args)
	case "REPAIR":
		return e.doRepair(ctx, player, args)
	// === MOVEMENT/STEALTH ===
	case "HIDE":
		return e.doHide(ctx, player)
	case "REVEAL", "UNHIDE":
		if !player.Hidden {
			return &CommandResult{Messages: []string{"You are not hidden."}}
		}
		player.Hidden = false
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      []string{"You reveal yourself."},
			RoomBroadcast: []string{fmt.Sprintf("%s reveals themselves.", player.FirstName)},
		}
	case "SNEAK":
		return e.doSneak(ctx, player, args)
	case "FLY":
		return e.doFly(ctx, player)
	case "ASCEND":
		if player.Position != 4 {
			return &CommandResult{Messages: []string{"You must be flying to ascend."}}
		}
		return e.doMove(ctx, player, "U")
	case "DESCEND":
		if player.Position != 4 {
			return &CommandResult{Messages: []string{"You must be flying to descend."}}
		}
		return e.doMove(ctx, player, "D")
	case "LAND":
		if player.Position != 4 {
			return &CommandResult{Messages: []string{"You aren't flying."}}
		}
		player.Position = 0
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"You land."}, RoomBroadcast: []string{fmt.Sprintf("%s lands.", player.FirstName)}}
	// === ITEM INTERACTION ===
	case "PUT", "PLACE":
		return e.doPut(ctx, player, args)
	case "FILL":
		return e.doFill(ctx, player, args)
	case "MARK":
		return e.doMark(ctx, player, args)
	case "UNDRESS":
		return e.doUndress(ctx, player)
	case "SKIN":
		return e.doSkin(ctx, player, args)
	// === INFO ===
	case "BALANCE":
		return e.doBalance(player)
	case "SPELL":
		return e.doSpellList(player)
	case "UNPROMPT":
		player.PromptMode = false
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"Prompt indicators off."}}
	case "VERSION", "NEWS", "NOTES":
		return &CommandResult{Messages: []string{"Legends of Future Past v11.5.11"}}
	case "CREDITS":
		return &CommandResult{Messages: []string{
			"",
			"  LEGENDS OF FUTURE PAST",
			"  ======================",
			"",
			"  Original Game (1992-1999)",
			"  Copyright (c) 1992-1999 Inner Circle Software / NovaLink USA Corp",
			"",
			"  Created & Programmed by .... Jon Radoff",
			"  Additional Programming ..... Ichiro Lambe",
			"  Co-Producer ................ Angela Bull",
			"  Legends Manager ............ Gary Whitten",
			"  World Building ............. Gary Whitten, David Goodman,",
			"                               Tony Spataro, Stacy Jannis,",
			"                               Kevin Jepson, Daniel Brainerd,",
			"                               Michael Hjerppe",
			"  Documentation .............. Gary Whitten",
			"  Quality Assurance .......... David Goodman, Stacy Jannis",
			"  Published by ............... NovaLink USA",
			"",
			"  2026 Re-Release",
			"  ---------------",
			"  Reimplemented from original script files and documentation",
			"  by Jon Radoff (https://metavert.io) using Claude Code.",
			"",
			"  Special thanks to David Goodman for supplying much of the",
			"  original materials used to reconstruct the game.",
			"",
			"  Available under the MIT License.",
			"  https://github.com/jonradoff/lofp",
			"",
		}}
	// === COMMUNICATION ===
	case "THINK":
		return e.doThink(player, input)
	case "TELEPATHY":
		if player.Race == RaceEphemeral { // Ephemeral - innate
			player.TelepathyActive = !player.TelepathyActive
			e.SavePlayer(ctx, player)
			if player.TelepathyActive {
				return &CommandResult{Messages: []string{"You open your mind to telepathic communication."}, PlayerState: player}
			}
			return &CommandResult{Messages: []string{"You close your mind to telepathic communication."}, PlayerState: player}
		}
		if !player.TelepathyActive {
			return &CommandResult{Messages: []string{"You don't have telepathic ability right now."}}
		}
		player.TelepathyActive = false
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"You close your mind to telepathic communication."}, PlayerState: player}
	case "DEPART":
		return e.doDepart(player)
	case "CANT":
		return e.doCant(player, args)
	// === COMBAT ===
	case "ATTACK", "KILL", "SLAY", "SMITE", "HIT":
		if len(args) == 0 {
			return &CommandResult{Messages: []string{"Attack what?"}}
		}
		return e.doAttackMonster(ctx, player, strings.Join(args, " "))
	case "FLEE":
		return e.doFlee(ctx, player)
	case "ADVANCE":
		if len(args) == 0 {
			return &CommandResult{Messages: []string{"Advance on what?"}}
		}
		target := strings.Join(args, " ")
		// Try monster first
		inst, def := e.findMonsterInRoom(player, target)
		if inst != nil {
			name := FormatMonsterName(def, e.monAdjs)
			article := articleFor(name, def.Unique)
			player.CombatTarget = &CombatTarget{IsMonster: true, MonsterID: inst.ID}
			player.Joined = true
			e.monsterMgr.mu.Lock()
			for i := range e.monsterMgr.instances {
				if e.monsterMgr.instances[i].ID == inst.ID && e.monsterMgr.instances[i].Target == "" {
					e.monsterMgr.instances[i].Target = player.FirstName
				}
			}
			e.monsterMgr.mu.Unlock()
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You advance toward %s%s.", article, name)},
				RoomBroadcast: []string{fmt.Sprintf("%s advances toward %s%s.", player.FirstName, article, name)},
			}
		}
		// Try player
		found := e.findPlayerInRoom(player, strings.ToLower(target))
		if found != nil {
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You advance toward %s.", found.FirstName)},
				RoomBroadcast: []string{fmt.Sprintf("%s advances toward %s.", player.FirstName, found.FirstName)},
			}
		}
		return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' here.", target)}}
	case "RETREAT":
		if player.CombatTarget == nil && !player.Joined {
			return &CommandResult{Messages: []string{"You are not engaged with anything."}}
		}
		e.disengageCombat(player)
		return &CommandResult{
			Messages:      []string{"You retreat."},
			RoomBroadcast: []string{fmt.Sprintf("%s retreats.", player.FirstName)},
		}
	case "GUARD":
		return e.doGuard(player, args)
	case "BACKSTAB":
		if !player.Hidden {
			return &CommandResult{Messages: []string{"You must be hidden to backstab!"}}
		}
		if len(args) == 0 {
			return &CommandResult{Messages: []string{"Backstab what?"}}
		}
		return e.doBackstab(ctx, player, strings.Join(args, " "))
	case "BITE":
		if player.Race != RaceDrakin && player.Race != RaceWolfling && player.Race != RaceMurg {
			return &CommandResult{Messages: []string{"Your race cannot bite effectively in combat."}}
		}
		if len(args) == 0 {
			return &CommandResult{Messages: []string{"Bite what?"}}
		}
		return e.doAttackMonster(ctx, player, strings.Join(args, " "))
	case "AVOID":
		return &CommandResult{Messages: []string{"[Avoid coming soon.]"}}
	case "BERSERK", "FRENZY":
		return e.doStance(player, StanceBerserk)
	case "DEFENSIVE":
		return e.doStance(player, StanceDefensive)
	case "OFFENSIVE":
		return e.doStance(player, StanceOffensive)
	case "WARY":
		return e.doStance(player, StanceWary)
	case "MODERATE", "NORMAL":
		return e.doStance(player, StanceNormal)
	case "PREPARE", "INVOKE":
		return e.doPrepareSpell(player, args)
	case "CAST":
		return e.doCastSpell(ctx, player, args)
	case "PSI":
		return e.doPreparePsi(player, args)
	case "PROJECT":
		return e.doProjectPsi(ctx, player, args)
	case "CHANT":
		return e.doChant(ctx, player, args)
	case "COMMAND":
		return e.doCommandCreature(player, args)
	case "MASTER":
		return &CommandResult{Messages: []string{"[Spell mastery coming soon.]"}}
	case "NOCK", "LOAD":
		return e.doLoadWeapon(ctx, player, args)
	case "SPECIALIZE":
		return &CommandResult{Messages: []string{"[Weapon specialization coming soon.]"}}
	// === SKILL-BASED (TODO: implement) ===
	case "DISARM":
		return e.doDisarm(ctx, player, args)
	case "STEAL", "FILCH", "ROB":
		return &CommandResult{Messages: []string{"[Stealing coming soon.]"}} // TODO: pick pockets, requires Legerdemain
	case "STALK":
		return &CommandResult{Messages: []string{"[Stalking coming soon.]"}} // TODO: secretly follow someone
	case "TEACH":
		return e.doTeach(ctx, player, args)
	case "SELFTRAIN":
		return &CommandResult{Messages: []string{"[Self-training coming soon.]"}} // TODO: train self at +1 cost
	case "UNLEARN":
		return e.doUnlearn(ctx, player, args)
	case "LEARN":
		return e.doLearn(ctx, player, args)
	case "ANOINT":
		return e.doAnoint(ctx, player, args)
	case "TRAP":
		return &CommandResult{Messages: []string{"[Trap setting coming soon.]"}} // TODO: place trap on container
	case "SURVEY":
		return &CommandResult{Messages: []string{"[Mining survey coming soon.]"}} // TODO: survey area for minerals
	case "SPLIT":
		return &CommandResult{Messages: []string{"[Coin splitting coming soon.]"}} // TODO: divide coins among group
	// === RACIAL/SPECIAL (TODO: implement) ===
	case "BLEND":
		if player.Race != RaceHighlander {
			return &CommandResult{Messages: []string{"Only Highlanders can blend with their surroundings."}}
		}
		room := e.rooms[player.RoomNumber]
		if room == nil || (room.Terrain != "MOUNTAIN" && room.Terrain != "CAVE" && room.Terrain != "DEEPCAVE") {
			return &CommandResult{Messages: []string{"You can only blend in mountainous or cavernous terrain."}}
		}
		player.Hidden = true
		return &CommandResult{
			Messages:      []string{"You blend into the rocky surroundings, becoming nearly invisible."},
			RoomBroadcast: []string{fmt.Sprintf("%s seems to meld into the rock.", player.FirstName)},
		}
	case "CALL":
		return &CommandResult{Messages: []string{"[Aelfen familiar coming soon.]"}} // TODO: call woodland creature
	case "TRANSFORM":
		if player.Race != RaceWolfling {
			return &CommandResult{Messages: []string{"Only wolflings can transform."}}
		}
		player.RoundTimeExpiry = time.Now().Add(7 * time.Second)
		if player.WolfForm {
			// Wolf → human
			player.WolfForm = false
			e.SavePlayer(ctx, player)
			return &CommandResult{
				Messages:      []string{"You howl in pain as your body undergoes a metamorphosis and resumes humanoid form.", "[Round: 7 sec]"},
				RoomBroadcast: []string{fmt.Sprintf("A wolf shudders and transforms, resuming the shape of %s. Where the wolf stood, %s rises in humanoid form.", player.FirstName, player.Pronoun())},
			}
		}
		// Human → wolf
		player.WolfForm = true
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      []string{"You groan in pain as your body undergoes a metamorphosis and assumes the form of a wolf.", "[Round: 7 sec]"},
			RoomBroadcast: []string{fmt.Sprintf("Without warning, %s howls and collapses to the ground, shaking. Undergoing a terrible transformation, %s changes shape into that of a wolf!", player.FirstName, player.Pronoun())},
		}
	case "MOLD":
		return &CommandResult{Messages: []string{"[Gem molding coming soon.]"}} // TODO: highlander gem improvement
	case "DISGUISE":
		return &CommandResult{Messages: []string{"[Disguise coming soon.]"}} // TODO: requires Disguise skill
	case "SUBMIT":
		if player.Submitting {
			return &CommandResult{Messages: []string{"You are already submitting."}}
		}
		player.Submitting = true
		return &CommandResult{
			Messages:      []string{"You submit, accepting whatever may come."},
			RoomBroadcast: []string{fmt.Sprintf("%s submits.", player.FirstName)},
		}
	case "UNSUBMIT":
		if !player.Submitting {
			return &CommandResult{Messages: []string{"You are not submitting."}}
		}
		player.Submitting = false
		return &CommandResult{
			Messages:      []string{"You stop submitting."},
			RoomBroadcast: []string{fmt.Sprintf("%s stops submitting.", player.FirstName)},
		}
	case "ARREST":
		return &CommandResult{Messages: []string{"[Arrest coming soon.]"}} // TODO: lawkeeper arrest
	case "ENROLL":
		return &CommandResult{Messages: []string{"[Organization enrollment coming soon.]"}} // TODO: join open org
	case "INITIATE":
		return e.doInitiate(ctx, player, args)
	case "FOLLOW":
		return e.doFollow(player, args)
	case "JOIN":
		// JOIN with args → group follow; JOIN alone → old stub
		if len(args) > 0 {
			return e.doFollow(player, args)
		}
		return &CommandResult{Messages: []string{"Follow whom?"}}
	case "LEAVE":
		return e.doLeave(player)
	case "DISBAND":
		return e.doDisband(player)
	case "TEND":
		return e.doTend(ctx, player, args)
	case "BREAK":
		return &CommandResult{Messages: []string{"[Object destruction coming soon.]"}} // TODO: destroy item with another item
	case "ASSIST":
		room := e.rooms[player.RoomNumber]
		roomName := "unknown"
		if room != nil {
			roomName = room.Name
		}
		e.lastAssistName = player.FirstName
		e.lastAssistRoom = player.RoomNumber
		return &CommandResult{
			Messages:    []string{"Your request for assistance has been noted. A gamemaster will be with you as soon as possible."},
			GMBroadcast: []string{fmt.Sprintf("[GM] %s is requesting assistance at %s (room %d). Use @answer to respond.", player.FirstName, roomName, player.RoomNumber)},
		}
	case "REPORT":
		if len(args) == 0 {
			return &CommandResult{Messages: []string{"Report what? Usage: REPORT <message>"}}
		}
		reportText := strings.Join(strings.Fields(input)[1:], " ")
		room := e.rooms[player.RoomNumber]
		roomName := "unknown"
		if room != nil {
			roomName = room.Name
		}
		e.Events.Publish("report", fmt.Sprintf("[REPORT] %s (room %d %s): %s", player.FirstName, player.RoomNumber, roomName, reportText))
		return &CommandResult{
			Messages:       []string{"Your report has been filed. Thank you!"},
			GMBroadcast:    []string{fmt.Sprintf("[REPORT] %s (room %d, %s): %s", player.FirstName, player.RoomNumber, roomName, reportText)},
			LogEventType:   "report",
			LogEventDetail: reportText,
		}
	case "LOCK":
		return e.doLock(ctx, player, args)
	case "UNLOCK":
		return e.doUnlock(ctx, player, args)
	case "POUR":
		return &CommandResult{Messages: []string{"[Liquid transfer coming soon.]"}}
	case "ACTBRIEF":
		return e.doSet(ctx, player, []string{"ACTBRIEF"})
	case "RPBRIEF":
		return e.doSet(ctx, player, []string{"RPBRIEF"})
	case "SET":
		return e.doSet(ctx, player, args)
	case "REPORTS":
		return e.doReports(ctx, player, args)
	case "REPORTCOMPLETE":
		return e.doReportComplete(ctx, player, args)
	case "SNIFF", "SMELL":
		if len(args) > 0 {
			return e.doItemInteraction(ctx, player, "SMELL", args)
		}

		// Check for room-level IFVERB SMELL -1 first.
		if result := e.doRoomScriptVerb(ctx, player, "SMELL"); result != nil {
			return result
		}

		// No room script, use normal emote.
		return e.processEmote(player, verb, args)
	case "LISTEN":
		if len(args) > 0 {
			return e.doItemInteraction(ctx, player, "LISTEN", args)
		}

		// Check for room-level IFVERB LISTEN -1 first.
		if result := e.doRoomScriptVerb(ctx, player, "LISTEN"); result != nil {
			return result
		}

		// No room script, use normal emote.
		return e.processEmote(player, "LISTEN", args)
	default:
		// Run room-level IFVERB <verb> -1 scripts first.
		// Example: IFVERB PAY -1
		if result := e.doRoomScriptVerb(ctx, player, verb); result != nil {
			return result
		}
		return &CommandResult{Messages: []string{fmt.Sprintf("I don't understand \"%s\". Type HELP for commands.", strings.ToLower(input))}}
	}
}

func (e *GameEngine) UpdatePlayerTimers(player *Player) []string {
	var messages []string

	messages = append(messages, e.ProcessPeriodicStatEffects(player)...)
	messages = append(messages, e.RemoveExpiredStatEffects(player)...)

	// Later:
	// messages = append(messages, e.RemoveExpiredResistances(player)...)
	// messages = append(messages, e.RemoveExpiredConditions(player)...)

	return messages
}

func (e *GameEngine) ProcessPeriodicStatEffects(player *Player) []string {
	now := time.Now()

	if len(player.ActiveStatEffects) == 0 {
		return nil
	}

	var messages []string
	active := player.ActiveStatEffects[:0]

	for _, effect := range player.ActiveStatEffects {

		if effect.Source != EffectSourcePoison &&
			effect.Source != EffectSourceDisease {
			active = append(active, effect)
			continue
		}

		if effect.ExpiresAt.After(now) {
			active = append(active, effect)
			continue
		}

		switch effect.Source {
		case EffectSourcePoison:
			damage := -effect.Modifier

			if damage > 0 {
				player.BodyPoints -= damage
				if player.BodyPoints < 0 {
					player.BodyPoints = 0
				}

				messages = append(
					messages,
					fmt.Sprintf(
						"Poison burns in your veins. [%d Damage]",
						damage,
					),
				)
			}

		case EffectSourceDisease:
			drain := -effect.Modifier

			if drain > 0 {
				player.Fatigue -= drain
				if player.Fatigue < 0 {
					player.Fatigue = 0
				}

				messages = append(
					messages,
					fmt.Sprintf(
						"Disease saps your strength. [%d Fatigue]",
						drain,
					),
				)
			}
		}

		effect.Modifier++ // -5 -> -4

		if effect.Modifier >= 0 {
			switch effect.Source {
			case EffectSourcePoison:
				player.Poisoned = false
				messages = append(
					messages,
					"The poison finally leaves your system.",
				)

			case EffectSourceDisease:
				player.Diseased = false
				messages = append(
					messages,
					"You finally recover from the disease.",
				)
			}

			continue
		}

		effect.ExpiresAt = now.Add(time.Minute)
		active = append(active, effect)
	}

	player.ActiveStatEffects = active

	return messages
}

func (e *GameEngine) RemoveExpiredStatEffects(player *Player) []string {
	now := time.Now()

	if len(player.ActiveStatEffects) == 0 {
		return nil
	}

	var messages []string
	active := player.ActiveStatEffects[:0]

	for _, effect := range player.ActiveStatEffects {

		// Periodic effects are handled elsewhere.
		if effect.Source == EffectSourcePoison ||
			effect.Source == EffectSourceDisease {
			active = append(active, effect)
			continue
		}

		// No expiration = persistent effect.
		if effect.Permanent {
			active = append(active, effect)
			continue
		}

		if effect.ExpiresAt.After(now) {
			active = append(active, effect)
			continue
		}

		switch effect.Source {
		case EffectSourceSpell:
			if spell := FindSpellByID(effect.EffectID); spell != nil {
				messages = append(
					messages,
					fmt.Sprintf(
						"The effects of %s wear off.",
						spell.Name,
					),
				)
			} else {
				messages = append(
					messages,
					"A magical effect wears off.",
				)
			}

		case EffectSourcePotion:
			messages = append(
				messages,
				"The effects of a potion wear off.",
			)

		default:
			messages = append(
				messages,
				"An effect wears off.",
			)
		}
	}

	player.ActiveStatEffects = active

	return messages
}

// allVerbs is the canonical list of all recognized command verbs.
// Abbreviation resolution matches against this list.
var allVerbs = []string{
	"LOOK", "EXAMINE", "INSPECT", "GO", "GET", "TAKE", "DROP",
	"INVENTORY", "STATUS", "HEALTH", "DIAGNOSE",
	"WIELD", "UNWIELD", "WEAR", "REMOVE",
	"OPEN", "CLOSE", "SIT", "STAND", "KNEEL", "LAY",
	"BRIEF", "FULL", "PROMPT", "WHO", "SKILLS", "WEALTH",
	"QUIT", "HELP", "ADVICE", "ASSIST", "ACT", "EMOTE", "RECITE", "READ", "CLIMB",
	"PULL", "PUSH", "TURN", "RUB", "TAP", "TOUCH", "SEARCH", "DIG", "RECALL", "USE", "PRAY",
	"CAST", "CONCENTRATE", "BUY", "SELL",
	"DRINK", "SIP", "LIGHT", "EXTINGUISH", "DOUSE",
	"FLIP", "LATCH", "UNLATCH",
	"DEPOSIT", "WITHDRAW", "TRAIN",
	"MINE", "FORAGE",
	"CRAFT", "FORGE", "SMELT", "WEAVE", "DYE", "BREW", "ANALYZE", "WORK", "REPAIR", "LEARN",
	// Movement/stealth
	"HIDE", "SNEAK", "FLY", "ASCEND", "DESCEND", "LAND",
	// Interaction
	"PUT", "PLACE", "FILL", "MARK", "UNDRESS", "SKIN", "PAY",
	// Info
	"BALANCE", "SPELL", "BRIEF", "FULL", "PROMPT", "UNPROMPT", "VERSION", "CREDITS",
	// Communication
	"THINK", "CANT",
	// Combat (TODO: implement)
	"ATTACK", "KILL", "SLAY", "SMITE", "ADVANCE", "RETREAT", "GUARD",
	"BACKSTAB", "BITE", "AVOID", "BERSERK", "FRENZY",
	"DEFENSIVE", "OFFENSIVE", "WARY", "NORMAL",
	"INVOKE", "PREPARE", "CHANT", "COMMAND", "MASTER",
	"NOCK", "LOAD", "SPECIALIZE",
	// Skill-based (TODO: implement)
	"DISARM", "STEAL", "FILCH", "ROB", "STALK",
	"TEACH", "SELFTRAIN", "UNLEARN", "LEARN",
	"ANOINT", "POISON", "TRAP",
	"SURVEY", "SPLIT",
	// Racial (TODO: implement)
	"BLEND", "CALL", "TRANSFORM", "MOLD",
	"DISGUISE", "SUBMIT", "UNSUBMIT", "ARREST",
	"ENROLL", "INITIATE", "JOIN", "FOLLOW", "LEAVE", "DISBAND",
	"TEND", "BREAK",
	"SNIFF", "SMELL", "LISTEN",
	// Communication
	"WHISPER", "YELL", "SPEECH", "THINK", "TELEPATHY", "CONTACT",
	// Interaction
	"GIVE", "EAT", "COUNT", "DEPART",
	// Info
	"TIME", "EXPERIENCE", "INFO",
	// Roleplay verbs
	"SMILE", "BOW", "CURTSEY", "WAVE", "NOD", "LAUGH", "CHUCKLE",
	"GRIN", "FROWN", "SIGH", "SHRUG", "WINK", "CRY", "DANCE",
	"HUG", "KISS", "POKE", "TICKLE", "SLAP", "HOWL", "SING",
	"PACE", "FIDGET", "SHIVER", "SNORT", "GROAN", "MUMBLE",
	"BABBLE", "BEAM", "SWOON", "TOAST", "SHUDDER", "POINT",
	"KICK", "KNOCK", "TOUCH", "RUB", "PET", "PUNCH", "SPIT",
	"GAZE", "GLARE", "SCOWL", "COMFORT", "RECITE", "YAWN",
	// New emotes
	"BLINK", "BLUSH", "CRINGE", "CUDDLE", "COUGH", "FURROW",
	"GASP", "GIGGLE", "GRIMACE", "GROWL", "GULP", "JUMP",
	"LEAN", "NUZZLE", "PANT", "PONDER", "POUT", "ROLL",
	"SCREAM", "SMIRK", "SNICKER", "SALUTE", "STRETCH", "TAP",
	"TWIRL", "WINCE", "WHISTLE", "MUTTER", "CARESS", "NUDGE",
	"ARCH", "RAISE", "HEAD", "SCRATCH", "CLAP",
	// Additional emotes
	"LICK", "NIBBLE", "BARK", "CLAW", "CURSE", "DUCK", "HISS",
	"HOLD", "HULA", "JIG", "MOAN", "MASSAGE", "PINCH", "PLAY",
	"PURR", "ROAR", "SNARL", "SNUGGLE", "WAG", "WAIT", "WRITE",
	"YOWL", "THUMP", "APPLAUD", "PEER", "GRUNT", "DIP",
	"HANDRAISE", "HANDSHAKE", "HEADSHAKE", "PICK", "GESTURE",
	"CURTSY",
	// Additional verbs
	"ORDER", "UNLIGHT", "IGNITE", "QUAFF", "SHOUT",
	"LOCK", "UNLOCK", "POUR", "UNEMOTE", "ACTBRIEF", "RPBRIEF",
	"FLEE", "MODERATE", "HIT", "PSI", "PROJECT", "DEPART", "REVEAL", "UNHIDE", "REPORT", "SET",
	// Self-emotes
	"FUME", "SQUINT", "HUM", "SNIFFLE", "SLOUCH", "SNORE", "SNEEZE",
	"STARE", "PUCKER", "CRACK", "BOUNCE", "STRIKE", "CLUTCH",
	"WIPE", "GRIT", "TOSS", "ATTENTION", "TONGUE", "WRINKLE", "PUFF",
	"DIZZY", "BAT",
	// Race-specific
	"FLICK", "BARE", "SPREAD", "FOLD", "SWISH",
	"RUBEARS", "PULLBEARD", "SCENT", "WHINE", "DROOP",
}

// verbAliases maps short exact aliases that should bypass prefix matching.
// These are kept for single-letter or legacy shortcuts.
var verbAliases = map[string]string{
	"L": "LOOK", "I": "INVENTORY", "Q": "QUIT", "X": "QUIT",
	"INV": "INVENTORY", "STAT": "STATUS", "UNUSE": "UNWIELD",
	"DON": "WEAR", "EXIT": "QUIT", "SKILL": "SKILLS",
	"WHI": "WHISPER", "THIN": "THINK", "CONTA": "CONTACT",
	"DI":    "DIAGNOSE",
	"ORDER": "BUY", "UNLIGHT": "EXTINGUISH", "IGNITE": "LIGHT",
	"QUAFF": "DRINK", "SHOUT": "YELL", "A": "ATTACK",
	"PLACE": "PUT", "TRANS": "TRANSFORM",
	"PSIONICS": "PSI",
}

// resolveVerb resolves a typed verb to its canonical form.
// First checks exact aliases, then tries unique prefix matching against allVerbs.
func resolveVerb(input string) string {
	// Exact alias match
	if canonical, ok := verbAliases[input]; ok {
		return canonical
	}
	// Exact match in verb list
	for _, v := range allVerbs {
		if v == input {
			return v
		}
	}
	// Prefix match — must be unique
	var match string
	for _, v := range allVerbs {
		if strings.HasPrefix(v, input) {
			if match != "" {
				// Ambiguous — return input unchanged so it falls through to "don't understand"
				return input
			}
			match = v
		}
	}
	if match != "" {
		return match
	}
	return input
}

func (e *GameEngine) doMove(ctx context.Context, player *Player, dir string) *CommandResult {

	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := int(time.Until(player.RoundTimeExpiry).Seconds()) + 1
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("[Wait %d seconds...]", remaining),
			},
		}
	}

	if player.Immobilized {
		return &CommandResult{
			Messages: []string{"You are immobilized and cannot move!"},
		}
	}

	if player.CombatTarget != nil {
		return &CommandResult{
			Messages: []string{"You are engaged in combat! Try FLEE."},
		}
	}

	// Normal movement reveals hidden players
	// (but not Ethereal Projection — that's psi-maintained).
	if player.Hidden && !player.EtherealActive {
		player.Hidden = false
	}

	if player.Position != 0 && player.Position != 4 { // 4 = flying
		posNames := map[int]string{
			1: "sitting",
			2: "laying down",
			3: "kneeling",
		}

		posName := posNames[player.Position]
		if posName == "" {
			posName = "not standing"
		}

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You can't move while %s! Try STANDing first.",
					posName,
				),
			},
		}
	}

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{
			Error: "You are nowhere!",
		}
	}

	// ------------------------------------------------------------
	// Resolve destination.
	// ------------------------------------------------------------

	destNum, ok := room.Exits[dir]
	requiresFlight := false

	if !ok {
		if dir == "U" {
			destNum, ok = room.Exits["ABOVE"]
			if ok {
				requiresFlight = true
			}
		} else if dir == "D" {
			destNum, ok = room.Exits["BELOW"]
		}
	}

	if !ok {
		return &CommandResult{
			Messages: []string{"You can't go that way."},
		}
	}

	if requiresFlight && !player.IsFlying() {
		return &CommandResult{
			Messages: []string{
				"You leap into the air but come crashing back down. You need to be able to fly to go that way.",
			},
		}
	}

	dest := e.rooms[destNum]
	if dest == nil {
		return &CommandResult{
			Messages: []string{"That way seems to lead nowhere."},
		}
	}

	oldRoom := player.RoomNumber

	dirNames := map[string]string{
		"N":     "north",
		"S":     "south",
		"E":     "east",
		"W":     "west",
		"NE":    "northeast",
		"NW":    "northwest",
		"SE":    "southeast",
		"SW":    "southwest",
		"U":     "up",
		"D":     "down",
		"O":     "out",
		"ABOVE": "up",
		"BELOW": "down",
	}

	dirName := dirNames[dir]
	if dirName == "" {
		dirName = strings.ToLower(dir)
	}

	// ------------------------------------------------------------
	// Move the leader.
	// ------------------------------------------------------------

	player.RoomNumber = destNum
	player.Submitting = false

	e.SavePlayer(ctx, player)

	// ------------------------------------------------------------
	// If this player is following somebody, moving away from the
	// leader breaks follow.
	// ------------------------------------------------------------

	if player.Following != "" {
		leaderHere := false

		if e.sessions != nil {
			for _, p := range e.sessions.OnlinePlayers() {
				if p.FirstName == player.Following &&
					p.RoomNumber == destNum {

					leaderHere = true
					break
				}
			}
		}

		if !leaderHere {
			e.removeFromGroup(player)
		}
	}

	// ------------------------------------------------------------
	// Move ALL followers before doing any LOOK.
	//
	// This is important because a follower may be carrying the
	// group's light source. roomVisibility() needs everybody to
	// already be in the destination.
	// ------------------------------------------------------------

	var movedFollowers []*Player

	if player.IsGroupLeader &&
		len(player.GroupMembers) > 0 &&
		e.sessions != nil {

		for _, memberName := range player.GroupMembers {
			for _, p := range e.sessions.OnlinePlayers() {
				if p.FirstName != memberName {
					continue
				}

				if p.RoomNumber != oldRoom {
					continue
				}

				if p.Dead {
					continue
				}

				p.RoomNumber = destNum
				p.Submitting = false

				e.disengageCombat(p)
				e.SavePlayer(ctx, p)

				movedFollowers = append(movedFollowers, p)

				break
			}
		}
	}

	// Commanded creatures following the player move through the portal too
	e.moveCommandedFollowers(player, oldRoom, destNum)

	// ------------------------------------------------------------
	// NOW generate the leader's room look.
	//
	// At this point all followers — and their light sources — are
	// physically in the destination room.
	// ------------------------------------------------------------

	result := e.doLook(player)
	result.OldRoom = oldRoom

	// ------------------------------------------------------------
	// Movement messaging.
	// ------------------------------------------------------------

	if !player.GMInvis {
		if player.ExitEcho != "" {
			result.OldRoomMsg = []string{
				player.ExitEcho,
			}
		} else {
			result.OldRoomMsg = []string{
				fmt.Sprintf(
					"%s goes %s.",
					player.FirstName,
					dirName,
				),
			}
		}

		if player.EntryEcho != "" {
			result.RoomBroadcast = []string{
				player.EntryEcho,
			}
		} else {
			result.RoomBroadcast = []string{
				fmt.Sprintf(
					"%s arrives.",
					player.FirstName,
				),
			}
		}
	}

	if len(movedFollowers) > 0 {
		result.OldRoomMsg = append(
			result.OldRoomMsg,
			fmt.Sprintf(
				"%s's group goes %s.",
				player.FirstName,
				dirName,
			),
		)

		result.RoomBroadcast = append(
			result.RoomBroadcast,
			fmt.Sprintf(
				"%s's group arrives.",
				player.FirstName,
			),
		)
	}

	// ------------------------------------------------------------
	// Send followers their room look.
	//
	// Again, everyone has already moved, so shared lighting works
	// for them too.
	// ------------------------------------------------------------

	for _, p := range movedFollowers {
		if e.sendToPlayer != nil {
			followLook := e.doLook(p)

			e.sendToPlayer(
				p.FirstName,
				followLook.Messages,
			)
		}
	}

	// ------------------------------------------------------------
	// Run entry scripts after movement/look state is settled.
	// ------------------------------------------------------------

	e.applyEntryScripts(
		ctx,
		player,
		dest,
		result,
	)

	for _, p := range movedFollowers {
		e.applyEntryScripts(
			ctx,
			p,
			dest,
			&CommandResult{},
		)
	}

	return result
}

func (mm *monsterManager) MoveMonster(idx, destRoom int) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if idx < 0 || idx >= len(mm.instances) {
		return
	}

	oldRoom := mm.instances[idx].RoomNumber

	// Remove from old room index.
	roomList := mm.monstersByRoom[oldRoom]
	for i, monsterIdx := range roomList {
		if monsterIdx == idx {
			mm.monstersByRoom[oldRoom] = append(roomList[:i], roomList[i+1:]...)
			break
		}
	}

	// Move the actual monster.
	mm.instances[idx].RoomNumber = destRoom

	// Add to new room index.
	mm.monstersByRoom[destRoom] = append(mm.monstersByRoom[destRoom], idx)
}

// EnterRoom performs a look and runs IFENTRY scripts. Used on login/creation.
func (e *GameEngine) EnterRoom(ctx context.Context, player *Player) *CommandResult {
	// Show date and time on entry
	period := "day"
	if IsNight() {
		period = "night"
	}
	weather := strings.ToLower(e.GetRoomWeather(player.RoomNumber))
	var timeMsg string
	if weather != "" {
		timeMsg = fmt.Sprintf("It is %s %d, %d. It is %s. %s",
			GameMonthName(), GameDay()%28+1, GameYear(), period, weather)
	} else {
		timeMsg = fmt.Sprintf("It is %s %d, %d. It is %s.",
			GameMonthName(), GameDay()%28+1, GameYear(), period)
	}

	result := e.doLook(player)
	result.Messages = append([]string{timeMsg}, result.Messages...)
	room := e.rooms[player.RoomNumber]
	if room != nil {
		e.applyEntryScripts(ctx, player, room, result)
	}
	return result
}

// GetRoom returns the room struct for GMCP/protocol data. Returns nil if not found.
func (e *GameEngine) GetRoom(roomNumber int) *gameworld.Room {
	return e.rooms[roomNumber]
}

// buildActiveMonsterLists combines base MLISTs with the current season's MLISTs.
func (e *GameEngine) buildActiveMonsterLists() []gameworld.MonsterList {
	lists := make([]gameworld.MonsterList, len(e.baseMonsterLists))
	copy(lists, e.baseMonsterLists)
	if seasonal, ok := e.seasonalMonsterLists[e.currentSeason]; ok {
		lists = append(lists, seasonal...)
	}
	return lists
}

// CheckSeasonChange checks if the game season has changed and hot-swaps MLISTs.
func (e *GameEngine) CheckSeasonChange() {
	newSeason := GameSeason()
	if newSeason == e.currentSeason {
		return
	}
	oldSeason := e.currentSeason
	e.currentSeason = newSeason
	e.monsterLists = e.buildActiveMonsterLists()

	// Apply seasonal room overrides
	e.applySeasonalRooms()
	//applies base forage with seasonal forage
	e.forageDefs = e.buildActiveForageDefs()

	log.Printf("Season changed: %s -> %s. Active MLISTs: %d", oldSeason, newSeason, len(e.monsterLists))
	e.Events.Publish("time", fmt.Sprintf("The season has changed to %s.", SeasonName()))

	// Broadcast season change to outdoor players
	seasonMessages := map[string]string{
		"PSCRIPT": "The chill of winter recedes as spring arrives in the Shattered Realms. New growth appears across the land.",
		"SSCRIPT": "The warmth of summer settles over the Shattered Realms. The days grow long and hot.",
		"ASCRIPT": "A cool breeze heralds the arrival of autumn. Leaves begin to turn golden and crimson across the land.",
		"WSCRIPT": "Winter descends upon the Shattered Realms. A bitter cold wind sweeps across the land.",
	}
	if msg, ok := seasonMessages[newSeason]; ok {
		e.broadcastOutdoor(msg)
	}
}

// buildActiveForageDefs combines base FORAGEDEFs with the current season's FORAGEDEFs.
func (e *GameEngine) buildActiveForageDefs() []gameworld.ForageDef {

	active := make([]gameworld.ForageDef, len(e.baseForageDefs))

	copy(active, e.baseForageDefs)

	seasonal := e.seasonalForageDefs[e.currentSeason]

	for _, sdef := range seasonal {
		replaced := false

		for i, def := range active {
			if def.Terrain == sdef.Terrain &&
				def.ItemNum == sdef.ItemNum &&
				def.AdjNum == sdef.AdjNum {

				active[i] = sdef
				replaced = true
				break
			}
		}

		if !replaced {
			active = append(active, sdef)
		}
	}

	return active
}

// applySeasonalRooms applies seasonal room overrides for the current season.
// Seasonal scripts define room descriptions, exits, and items that change with the season.
func (e *GameEngine) applySeasonalRooms() {
	rooms, ok := e.seasonalRooms[e.currentSeason]
	if !ok || len(rooms) == 0 {
		return
	}
	count := 0
	for i := range rooms {
		r := &rooms[i]
		if existing := e.rooms[r.Number]; existing != nil {
			// Override description and terrain but preserve dynamic state
			existing.Name = r.Name
			existing.Description = r.Description
			existing.Terrain = r.Terrain
			existing.Exits = r.Exits
			existing.MonsterGroup = r.MonsterGroup
			if len(r.Items) > 0 {
				existing.Items = r.Items
			}
			if len(r.Modifiers) > 0 {
				existing.Modifiers = r.Modifiers
			}
			if len(r.ItemDescriptions) > 0 {
				existing.ItemDescriptions = r.ItemDescriptions
			}
			if len(r.Scripts) > 0 {
				existing.Scripts = r.Scripts
			}
			count++
		} else {
			// New room from seasonal script
			e.rooms[r.Number] = r
			count++
		}
	}
	if count > 0 {
		log.Printf("Applied %d seasonal room overrides for %s", count, SeasonName())
	}
}

// applyEntryScripts runs IFENTRY scripts and merges results into the command result.
func (e *GameEngine) applyEntryScripts(ctx context.Context, player *Player, room *gameworld.Room, result *CommandResult) {
	sc := e.RunEntryScripts(player, room)
	if len(sc.Messages) > 0 {
		result.Messages = append(result.Messages, sc.Messages...)
		e.Events.Publish("script", fmt.Sprintf("IFENTRY fired for %s in room %d (%s)", player.FirstName, room.Number, room.Name))
	}
	if len(sc.RoomMsgs) > 0 {
		result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
	}
	if len(sc.GMMsgs) > 0 {
		result.GMBroadcast = append(result.GMBroadcast, sc.GMMsgs...)
	}
	if sc.MoveGroupTo > 0 {
		e.moveGroupToRoom(ctx, room.Number, sc.MoveGroupTo)
	}
	e.SavePlayer(ctx, player)

	// Spawn monsters for this room if needed (demand-based)
	e.spawnForRoom(room.Number)

	// Check if hostile monsters should aggro on the player entering the room
	go e.monsterCheckAggro(player, room.Number)
}

// doLookFull always shows the full room description (explicit LOOK command).
func (e *GameEngine) doLookFull(player *Player) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You are in a void."}}
	}
	result := e.doLook(player)
	// Always include the full description regardless of BriefMode
	if !player.Dead && e.canPlayerSee(player, room) && result.RoomDesc == "" {
		result.RoomDesc = room.Description
	}
	return result
}

func (e *GameEngine) doLook(player *Player) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You are in a void."}}
	}

	result := &CommandResult{
		RoomName: fmt.Sprintf("[%s]", room.Name),
	}

	if player.Dead {
		result.Messages = []string{
			result.RoomName,
			"You are dead and can't do much of anything beside wait for someone to attempt to raise you or for Eternity, Inc. to retrieve you. Hope you paid your premium! [You may type DEPART at any time to allow Eternity, Inc. to retrieve you.]",
		}
		return result
	}

	if !e.canPlayerSee(player, room) {
		result.Messages = []string{"It is too dark to see."}
		return result
	}

	if !player.BriefMode {
		result.RoomDesc = room.Description
	}

	// List visible items
	for _, ri := range room.Items {

		if ri.IsPut {
			continue
		}
		// Coin piles
		if ri.State == "MONEY" {
			result.Items = append(result.Items, "some coins")
			continue
		}
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		if containsFlag(itemDef.Flags, "HIDDEN") {
			continue
		}
		// Skip placeholder items (ANTI.SCR stubs and invisible items)
		nounName := e.getItemNounName(itemDef)
		if nounName == "anti-item" || nounName == "ucantsee" {
			continue
		}
		name := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
		if ri.Extend != "" {
			name += " " + ri.Extend
		}
		result.Items = append(result.Items, name)
	}

	// Collect other players in the room
	var playersHere []string
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.RoomNumber != player.RoomNumber {
				continue
			}
			if p.FirstName == player.FirstName && p.LastName == player.LastName {
				continue
			}
			if p.Hidden || p.Invisible || p.GMInvis {
				continue
			}
			posDesc := ""
			switch p.Position {
			case 1:
				posDesc = " (sitting)"
			case 2:
				posDesc = " (lying down)"
			case 3:
				posDesc = " (kneeling)"
			case 4:
				posDesc = " (flying)"
			}
			if p.WolfForm {
				playersHere = append(playersHere, "a wolf"+posDesc)
			} else {
				playersHere = append(playersHere, fmt.Sprintf("%s the %s%s", p.FullName(), p.RaceName(), posDesc))
			}
		}
	}

	// List exits
	dirNames := map[string]string{
		"N": "north", "S": "south", "E": "east", "W": "west",
		"NE": "northeast", "NW": "northwest", "SE": "southeast", "SW": "southwest",
		"U": "up", "D": "down", "O": "out", "ABOVE": "up", "BELOW": "down",
	}
	var exits []string
	for dir := range room.Exits {
		if name, ok := dirNames[dir]; ok {
			exits = append(exits, name)
		} else {
			exits = append(exits, strings.ToLower(dir))
		}
	}
	result.Exits = exits

	// Populate GMCP room data
	result.RoomExits = make(map[string]int)
	for dir, roomNum := range room.Exits {
		dirLower := strings.ToLower(dir)
		if name, ok := dirNames[dir]; ok {
			dirLower = name
		}
		result.RoomExits[dirLower] = roomNum
	}
	result.RoomTerrain = room.Terrain
	result.RoomRegion = room.Region

	// Build messages
	var msgs []string
	msgs = append(msgs, result.RoomName)
	if result.RoomDesc != "" {
		msgs = append(msgs, descriptionToMessages(result.RoomDesc)...)
	}
	if len(result.Items) > 0 {
		msgs = append(msgs, "You see "+joinList(result.Items)+".")
	}
	if len(playersHere) > 0 {
		// Format like original: "You see Player1 and Player2." or "You see Player1, Player2 and Player3."
		var pList string
		if len(playersHere) == 1 {
			pList = playersHere[0]
		} else {
			pList = strings.Join(playersHere[:len(playersHere)-1], ", ") + " and " + playersHere[len(playersHere)-1]
		}
		msgs = append(msgs, "You see "+pList+".")
	}
	// Show monsters in the room
	monsterLines := e.MonsterLookLines(player.RoomNumber)
	msgs = append(msgs, monsterLines...)
	// Show weather for outdoor rooms
	if weatherLine := e.GetRoomWeather(player.RoomNumber); weatherLine != "" {
		msgs = append(msgs, weatherLine)
	}
	if len(exits) > 0 {
		msgs = append(msgs, "Obvious exits: "+strings.Join(exits, ", ")+".")
	} else {
		msgs = append(msgs, "There are no obvious exits.")
	}
	result.Messages = msgs
	return result
}

func (e *GameEngine) itemExamineStats(def *gameworld.ItemDef) []string {
	if def == nil {
		return nil
	}

	var msgs []string

	if strings.HasSuffix(def.Type, "_WEAPON") {
		weaponType := strings.TrimSuffix(def.Type, "_WEAPON")
		weaponType = strings.ReplaceAll(weaponType, "_", " ")
		weaponType = strings.ToLower(weaponType)

		msgs = append(msgs, fmt.Sprintf("Weapon type: %s", weaponType))
	}

	if def.Weight > 0 {
		msgs = append(msgs, fmt.Sprintf("Weight: %d", def.Weight))
	}

	if def.Substance != "" {
		material := strings.ToLower(def.Substance)
		material = strings.ReplaceAll(material, "_", " ")

		if material == "hardmetal" {
			material = "hard metal"
		}

		msgs = append(msgs, fmt.Sprintf(
			"Material: %s",
			strings.Title(material),
		))
	}

	if def.Parameter1 > 0 &&
		strings.HasSuffix(def.Type, "_WEAPON") {

		msgs = append(msgs, fmt.Sprintf(
			"Damage: %d",
			def.Parameter1,
		))
	}
	return msgs
}

// lookDirMap maps direction words/abbreviations to exit keys.
var lookDirMap = map[string]string{
	"n": "N", "north": "N", "s": "S", "south": "S",
	"e": "E", "east": "E", "w": "W", "west": "W",
	"ne": "NE", "northeast": "NE", "nw": "NW", "northwest": "NW",
	"se": "SE", "southeast": "SE", "sw": "SW", "southwest": "SW",
	"u": "U", "up": "U", "d": "D", "down": "D",
	"o": "O", "out": "O",
}

func (e *GameEngine) doLookAt(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return e.doLook(player)
	}

	target := strings.ToLower(strings.Join(args, " "))
	target = strings.TrimSpace(target)

	// LOOK AT <thing> should behave like EXAMINE <thing>.
	if strings.HasPrefix(target, "at ") {
		target = strings.TrimSpace(strings.TrimPrefix(target, "at "))
	}

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{
			Messages: []string{"You see nothing."},
		}
	}

	// Check for directional look (LOOK N, LOOK NORTH, etc.)
	if dir, ok := lookDirMap[target]; ok {
		destNum, hasExit := room.Exits[dir]

		if !hasExit && dir == "U" {
			destNum, hasExit = room.Exits["ABOVE"]
		}

		if !hasExit && dir == "D" {
			destNum, hasExit = room.Exits["BELOW"]
		}

		if !hasExit {
			return &CommandResult{
				Messages: []string{
					"You see nothing of interest in that direction.",
				},
			}
		}

		dest := e.rooms[destNum]
		if dest == nil {
			return &CommandResult{
				Messages: []string{
					"You see nothing of interest in that direction.",
				},
			}
		}

		// Directional look uses the visibility of the DESTINATION room.
		if e.roomVisibility(player, dest) == VisibilityDark {
			return &CommandResult{
				Messages: []string{
					"It is too dark to see in that direction.",
				},
			}
		}

		msgs := []string{
			fmt.Sprintf("[%s]", dest.Name),
		}

		if dest.Description != "" {
			msgs = append(
				msgs,
				descriptionToMessages(dest.Description)...,
			)
		}

		// Show players in that room.
		if e.sessions != nil {
			var playersHere []string

			for _, p := range e.sessions.OnlinePlayers() {
				if p.RoomNumber == destNum &&
					!p.Hidden &&
					!p.Invisible &&
					!p.GMInvis {

					playersHere = append(playersHere, p.FirstName)
				}
			}

			if len(playersHere) > 0 {
				msgs = append(
					msgs,
					fmt.Sprintf(
						"You see %s.",
						strings.Join(playersHere, ", "),
					),
				)
			}
		}

		// Show top-level room items only.
		for _, ri := range dest.Items {
			if ri.IsPut {
				continue
			}

			itemDef := e.items[ri.Archetype]
			if itemDef == nil {
				continue
			}

			// Don't expose hidden room machinery/items.
			if containsFlag(itemDef.Flags, "HIDDEN") {
				continue
			}

			itemName := e.formatItemName(
				itemDef,
				ri.Adj1,
				ri.Adj2,
				ri.Adj3,
			)

			msgs = append(
				msgs,
				fmt.Sprintf("You see %s.", itemName),
			)
		}

		// Show monsters.
		monLines := e.MonsterLookLines(destNum)
		msgs = append(msgs, monLines...)

		return &CommandResult{
			Messages: msgs,
		}
	}

	// Everything below this point is looking at the CURRENT room.
	if e.roomVisibility(player, room) == VisibilityDark {
		return &CommandResult{
			Messages: []string{
				"It is too dark to see anything.",
			},
		}
	}

	// Look at self.
	if target == "me" || target == "myself" || target == "self" {
		return e.examinePlayer(player, player)
	}

	// Look at another player.
	if found := e.findPlayerInRoom(player, target); found != nil {
		return e.examinePlayer(player, found)
	}

	// Look at a monster.
	if _, monDef := e.findMonsterInRoom(player, target); monDef != nil {
		return e.examineMonster(monDef)
	}

	// Check IN / ON / UNDER / BEHIND prefixes.
	prefix := ""
	remaining := target

	for _, p := range []string{
		"in ",
		"on ",
		"out",
		"under ",
		"behind ",
	} {
		if strings.HasPrefix(target, p) {
			prefix = strings.ToUpper(strings.TrimSpace(p))
			remaining = strings.TrimPrefix(target, p)
			break
		}
	}

	remaining, ordSkip := parseOrdinal(remaining)
	skip := ordSkip

	isContainer := func(def *gameworld.ItemDef) bool {
		return def.Type == "CONTAINER" ||
			containsFlag(def.Flags, "CONTAINER") ||
			def.Container == "IN" ||
			def.Container == "ON" ||
			def.Container == "UNDER" ||
			def.Container == "BEHIND"
	}

	// ------------------------------------------------------------
	// ROOM ITEMS
	// ------------------------------------------------------------
	for i, ri := range room.Items {
		if ri.IsPut {
			continue
		}

		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}

		name := e.getItemNounName(itemDef)

		if !matchesTarget(name, remaining, e.getAdjName(ri.Adj1)) &&
			!matchesTarget(name, remaining, e.getAdjName(ri.Adj2)) &&
			!matchesTarget(name, remaining, e.getAdjName(ri.Adj3)) {
			continue
		}

		// If this is a relationship-based LOOK and this visible object
		// uses a different relationship, keep looking for another
		// same-named room item.
		if (prefix == "IN" ||
			prefix == "ON" ||
			prefix == "UNDER" ||
			prefix == "BEHIND") &&
			itemDef.Container != "" &&
			itemDef.Container != prefix {
			continue
		}

		if skip > 0 {
			skip--
			continue
		}

		// LOOK IN / ON / BEHIND <container>
		if (prefix == "IN" ||
			prefix == "ON" ||
			prefix == "UNDER" ||
			prefix == "BEHIND") &&
			isContainer(itemDef) {

			displayName := e.formatItemName(
				itemDef,
				ri.Adj1,
				ri.Adj2,
				ri.Adj3,
			)

			// Room-authored relationship description takes priority
			// over dynamically listing PUT contents.
			refStr := fmt.Sprintf("%d", ri.Ref)

			if desc, ok := room.ItemDescriptions[prefix+":"+refStr]; ok {
				return &CommandResult{
					Messages: descriptionToMessages(desc),
				}
			}

			// Only IN containers need to be open.
			if prefix == "IN" &&
				ri.State != "OPEN" &&
				ri.State != "" {

				return &CommandResult{
					Messages: []string{
						fmt.Sprintf(
							"You'll need to open %s first.",
							displayName,
						),
					},
				}
			}

			var msgs []string

			switch prefix {
			case "ON":
				msgs = []string{
					fmt.Sprintf("You look on %s.", displayName),
				}

			case "UNDER":
				msgs = []string{
					fmt.Sprintf("You look under %s.", displayName),
				}

			case "BEHIND":
				msgs = []string{
					fmt.Sprintf("You look behind %s.", displayName),
				}

			default:
				msgs = []string{
					fmt.Sprintf("You look in %s.", displayName),
				}
			}

			found := false
			usedVolume := 0

			for _, child := range room.Items {
				if !child.IsPut || child.PutIn != ri.Ref {
					continue
				}

				childDef := e.items[child.Archetype]
				if childDef == nil {
					continue
				}

				usedVolume += childDef.Volume

				if childDef.Type == "MONEY" {
					switch childDef.Parameter1 {
					case 1:
						msgs = append(msgs, "You see some gold coins.")
					case 2:
						msgs = append(msgs, "You see some silver coins.")
					case 3:
						msgs = append(msgs, "You see some copper coins.")
					default:
						msgs = append(msgs, "You see some coins.")
					}

					found = true
					continue
				}

				childName := e.formatItemName(
					childDef,
					child.Adj1,
					child.Adj2,
					child.Adj3,
				)

				msgs = append(
					msgs,
					fmt.Sprintf("You see %s.", childName),
				)

				found = true
			}

			if itemDef.Interior > 0 {
				msgs = append(
					msgs,
					fmt.Sprintf(
						"Capacity: %d/%d",
						usedVolume,
						itemDef.Interior,
					),
				)
			}

			if !found {
				switch prefix {
				case "ON":
					msgs = append(msgs, "There is nothing on it.")

				case "BEHIND":
					msgs = append(msgs, "There is nothing behind it.")
				case "UNDER":
					msgs = append(msgs, "There is nothing under it.")

				default:
					msgs = append(msgs, "It is empty.")
				}
			}

			return &CommandResult{
				Messages: msgs,
			}
		}

		// Existing UNDER / other prefix handling.
		if prefix != "" {
			return e.lookPrefixRoomItem(
				room,
				itemDef,
				&ri,
				prefix,
			)
		}

		// Run EXAMINE preverb scripts before normal examine behavior.
		sc := e.RunPreverbScripts(
			player,
			room,
			"EXAMINE",
			&room.Items[i],
			itemDef,
		)

		result := &CommandResult{
			Messages:      append([]string{}, sc.Messages...),
			RoomBroadcast: append([]string{}, sc.RoomMsgs...),
			GMBroadcast:   append([]string{}, sc.GMMsgs...),
		}

		if sc.Blocked {
			if len(result.Messages) == 0 {
				result.Messages = []string{"You can't do that."}
			}

			return result
		}

		examineResult := e.examineRoomItem(
			player,
			room,
			itemDef,
			&room.Items[i],
		)

		result.Messages = append(
			result.Messages,
			examineResult.Messages...,
		)

		result.RoomBroadcast = append(
			result.RoomBroadcast,
			examineResult.RoomBroadcast...,
		)

		result.GMBroadcast = append(
			result.GMBroadcast,
			examineResult.GMBroadcast...,
		)

		return result
	}

	// ------------------------------------------------------------
	// PLAYER ITEMS
	// ------------------------------------------------------------

	// Search all player items (inventory + worn + wielded).
	allItems := make(
		[]InventoryItem,
		0,
		len(player.Inventory)+len(player.Worn)+1,
	)

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

		name := e.getItemNounName(itemDef)

		if !matchesTarget(name, remaining, e.getAdjName(ii.Adj1)) &&
			!matchesTarget(name, remaining, e.getAdjName(ii.Adj2)) &&
			!matchesTarget(name, remaining, e.getAdjName(ii.Adj3)) {
			continue
		}

		if skip > 0 {
			skip--
			continue
		}

		if prefix == "IN" && isContainer(itemDef) {
			return e.lookInContainer(
				player,
				itemDef,
				&ii,
			)
		}

		if prefix != "" {
			displayName := e.formatItemName(
				itemDef,
				ii.Adj1,
				ii.Adj2,
				ii.Adj3,
			)

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"You see nothing noteworthy %s %s.",
						strings.ToLower(prefix),
						displayName,
					),
				},
			}
		}

		tempRI := gameworld.RoomItem{
			Ref:       -1,
			Archetype: ii.Archetype,
			Adj1:      ii.Adj1,
			Adj2:      ii.Adj2,
			Adj3:      ii.Adj3,
			Val1:      ii.Val1,
			Val2:      ii.Val2,
			Val3:      ii.Val3,
			Val4:      ii.Val4,
			Val5:      ii.Val5,
			Traits:    ii.Traits,
		}

		// Run EXAMINE preverb scripts before normal examine behavior.
		sc := e.RunPreverbScripts(
			player,
			room,
			"EXAMINE",
			&tempRI,
			itemDef,
		)

		result := &CommandResult{
			Messages:      append([]string{}, sc.Messages...),
			RoomBroadcast: append([]string{}, sc.RoomMsgs...),
			GMBroadcast:   append([]string{}, sc.GMMsgs...),
		}

		if sc.Blocked {
			if len(result.Messages) == 0 {
				result.Messages = []string{"You can't do that."}
			}

			return result
		}

		stats := e.itemExamineStats(itemDef)

		if len(stats) > 0 {
			result.Messages = append(result.Messages, stats...)
		} else {
			result.Messages = append(
				result.Messages,
				fmt.Sprintf("You look at your %s.", name),
			)
		}

		if strings.HasSuffix(itemDef.Type, "_WEAPON") {
			if ii.State == "DAMAGED" {
				result.Messages = append(result.Messages, "Condition: Damaged")
			} else {
				result.Messages = append(result.Messages, "Condition: Undamaged")
			}
		}

		if sm := e.scrollLookMsg(ii.Archetype, ii.Val3); sm != "" {
			result.Messages = append(result.Messages, sm)
		}

		if ii.Identified {
			enchantments := e.itemEnchantments(&ii, itemDef)

			if len(enchantments) > 0 {
				result.Messages = append(result.Messages, "Enchantments:")
				for _, enchantment := range enchantments {
					result.Messages = append(result.Messages, "  "+enchantment)
				}
			}
		}

		return result
	}

	return &CommandResult{
		Messages: []string{
			"You don't see that here.",
		},
	}
}

func (e *GameEngine) itemEnchantments(item *InventoryItem, def *gameworld.ItemDef) []string {
	if item == nil || def == nil {
		return nil
	}

	var result []string
	seen := make(map[string]bool)

	addTrait := func(name string) {
		name = strings.ToUpper(name)

		if seen[name] {
			return
		}

		trait := e.traits[name]
		if trait == nil || trait.Power <= 0 {
			return
		}

		var display string

		if strings.HasPrefix(name, "MAGIC_PLUS_") {
			bonus := strings.TrimPrefix(name, "MAGIC_PLUS_")
			display = "Magic +" + bonus
		} else {
			display = strings.ReplaceAll(name, "_", " ")
			display = strings.ToLower(display)

			words := strings.Fields(display)
			for i := range words {
				words[i] = strings.ToUpper(words[i][:1]) + words[i][1:]
			}

			display = strings.Join(words, " ")
		}

		seen[name] = true
		result = append(result, display)
	}

	// Archetype traits first.
	for _, name := range def.Traits {
		addTrait(name)
	}

	// Instance traits second.
	for _, name := range item.Traits {
		addTrait(name)
	}

	return result
}

func (e *GameEngine) doCommandCreature(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{
			Messages: []string{"Command your creature to do what?"},
		}
	}

	if e.monsterMgr == nil {
		return &CommandResult{
			Messages: []string{"You have no commanded creature here."},
		}
	}

	command := strings.ToUpper(strings.Join(args, " "))

	e.monsterMgr.mu.Lock()

	creatureIdx := -1

	var def *gameworld.MonsterDef

	for i := range e.monsterMgr.instances {
		inst := &e.monsterMgr.instances[i]

		if inst.Alive && inst.CommanderID == player.FirstName {
			creatureIdx = i
			def = e.monsters[inst.DefNumber]
			break
		}
	}

	if creatureIdx == -1 || def == nil {
		e.monsterMgr.mu.Unlock()

		return &CommandResult{
			Messages: []string{"You have no creature under your command."},
		}
	}

	creature := &e.monsterMgr.instances[creatureIdx]
	name := FormatMonsterName(def, e.monAdjs)

	moveDir := dirMap[command]

	if moveDir != "" {
		creature.Following = ""

		if !e.moveMonsterDirection(creatureIdx, creature, def, moveDir) {
			e.monsterMgr.mu.Unlock()

			dirName := directionNames[moveDir]
			if dirName == "" {
				dirName = strings.ToLower(moveDir)
			}

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf("Your %s cannot go %s.", name, dirName),
				},
			}
		}

		e.monsterMgr.mu.Unlock()

		watching := creature.Watching
		roomNum := creature.RoomNumber
		commanderName := creature.CommanderID

		if watching && commanderName != "" && e.sendToPlayer != nil {
			for _, commander := range e.sessions.OnlinePlayers() {
				if commander.FirstName != commanderName {
					continue
				}

				messages := e.commandedCreatureLook(
					commander,
					roomNum,
				)

				e.sendToPlayer(
					commander.FirstName,
					messages,
				)

				break
			}
		}

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("Your %s obeys.", name),
			},
		}
	}

	// COMMAND KILL <target>
	if command == "KILL" {
		e.monsterMgr.mu.Unlock()

		return &CommandResult{
			Messages: []string{"Kill what?"},
		}
	}

	if strings.HasPrefix(command, "KILL ") {
		targetText := strings.TrimSpace(command[5:])
		targetLower := strings.ToLower(targetText)

		targetIdx := -1
		var targetDef *gameworld.MonsterDef

		for i := range e.monsterMgr.instances {
			target := &e.monsterMgr.instances[i]

			if !target.Alive ||
				target.RoomNumber != creature.RoomNumber ||
				target.ID == creature.ID {
				continue
			}

			td := e.monsters[target.DefNumber]
			if td == nil {
				continue
			}

			targetName := strings.ToLower(
				FormatMonsterName(td, e.monAdjs),
			)

			noun := strings.ToLower(td.Name)

			if strings.HasPrefix(targetName, targetLower) ||
				strings.HasPrefix(noun, targetLower) {
				targetIdx = i
				targetDef = td
				break
			}
		}

		if targetIdx == -1 || targetDef == nil {
			e.monsterMgr.mu.Unlock()

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"Your %s does not see '%s' here.",
						name,
						strings.ToLower(targetText),
					),
				},
			}
		}

		target := &e.monsterMgr.instances[targetIdx]
		targetName := FormatMonsterName(targetDef, e.monAdjs)

		creature.Target = ""
		creature.TargetMonsterID = target.ID
		creature.Following = ""
		creature.Guarding = ""

		// Target turns to fight the commanded creature.
		target.Target = ""
		target.TargetMonsterID = creature.ID

		e.monsterMgr.mu.Unlock()

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"Your %s attacks %s%s.",
					name,
					articleFor(targetName, targetDef.Unique),
					targetName,
				),
			},
		}
	}

	switch command {
	case "WATCH":
		creature.Watching = true

		e.monsterMgr.mu.Unlock()

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You begin seeing through your %s's eyes.",
					name,
				),
			},
		}

	case "UNWATCH":
		creature.Watching = false

		e.monsterMgr.mu.Unlock()

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You stop seeing through your %s's eyes.",
					name,
				),
			},
		}

	case "GUARD ME", "GUARD":
		creature.Guarding = player.FirstName
		creature.Following = player.FirstName

		// Guarding cancels any current combat order.
		creature.Target = ""
		creature.TargetMonsterID = -1

		e.monsterMgr.mu.Unlock()

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"Your %s begins guarding you.",
					name,
				),
			},
			RoomBroadcast: []string{
				fmt.Sprintf(
					"%s moves protectively alongside %s.",
					name,
					player.FirstName,
				),
			},
		}
	case "FOLLOW ME", "FOLL ME", "FOLLOW", "FOLL":
		creature.Following = player.FirstName

		e.monsterMgr.mu.Unlock()

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("Your %s acknowledges your command.", name),
			},
			RoomBroadcast: []string{
				fmt.Sprintf("%s is now following %s.", name, player.FirstName),
			},
		}

	case "UNFOLLOW", "UNFOLL", "STOP FOLLOWING":
		creature.Following = ""

		e.monsterMgr.mu.Unlock()

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("Your %s stops following you.", name),
			},
			RoomBroadcast: []string{
				fmt.Sprintf("%s stops following %s.", name, player.FirstName),
			},
		}

	case "BEGONE":
		if creature.ControlType != "SUMMONED" {
			e.monsterMgr.mu.Unlock()

			return &CommandResult{
				Messages: []string{"That creature cannot be dismissed."},
			}
		}

		monsterID := creature.ID

		// RemoveMonster locks the manager itself,
		// so release this lock first.
		e.monsterMgr.mu.Unlock()

		e.monsterMgr.RemoveMonster(monsterID)

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("Your %s vanishes.", name),
			},
			RoomBroadcast: []string{
				fmt.Sprintf("%s vanishes.", name),
			},
		}
	case "LOOK":
		roomNum := creature.RoomNumber

		// We're done reading the monster.
		e.monsterMgr.mu.Unlock()

		// Create a temporary copy of the player located where the creature is.
		// The real player never moves.
		observer := *player
		observer.RoomNumber = roomNum

		look := e.doLook(&observer)

		messages := []string{
			fmt.Sprintf("Through your %s, you receive a telepathic image:", name),
		}

		messages = append(messages, look.Messages...)

		return &CommandResult{
			Messages: messages,
		}

	default:
		e.monsterMgr.mu.Unlock()

		return &CommandResult{
			Messages: []string{"Your creature does not understand that command."},
		}
	}
}

func (e *GameEngine) commandedCreatureLook(player *Player, roomNum int) []string {
	observer := *player
	observer.RoomNumber = roomNum

	result := e.doLook(&observer)
	return result.Messages
}

func (mm *monsterManager) RemoveMonster(id int) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	for idx := range mm.instances {
		inst := &mm.instances[idx]

		if inst.ID != id {
			continue
		}

		roomNum := inst.RoomNumber

		// Remove it from the room index.
		roomList := mm.monstersByRoom[roomNum]

		for i, roomIdx := range roomList {
			if roomIdx == idx {
				mm.monstersByRoom[roomNum] =
					append(roomList[:i], roomList[i+1:]...)
				break
			}
		}

		// Retire the instance.
		inst.Alive = false
		inst.Watching = false
		return
	}
}

// scrollLookMsg returns a description line if the item is a scroll, empty string otherwise.
func (e *GameEngine) scrollLookMsg(archetype int, val3 int) string {
	if archetype != 168 {
		return ""
	}
	spell := FindSpellByID(val3)
	if spell != nil {
		return fmt.Sprintf("The scroll contains the spell '%s' (spell #%d).", spell.Name, val3)
	}
	return fmt.Sprintf("The scroll contains spell #%d.", val3)
}

// findPlayerInRoom finds an online player in the same room by name (first name match).
func (e *GameEngine) findPlayerInRoom(self *Player, target string) *Player {
	if e.sessions == nil {
		return nil
	}

	//normalize target because were using lowercase
	target = strings.ToLower(strings.TrimSpace(target))

	for _, p := range e.sessions.OnlinePlayers() {
		if p.RoomNumber != self.RoomNumber {
			continue
		}
		if p.FirstName == self.FirstName && p.LastName == self.LastName {
			continue // skip self, handled separately
		}
		if p.Hidden || p.Invisible || p.GMInvis {
			continue
		}
		if strings.HasPrefix(strings.ToLower(p.FirstName), target) {
			return p
		}
		fullName := strings.ToLower(p.FirstName + " " + p.LastName)
		if strings.HasPrefix(fullName, target) {
			return p
		}
	}
	return nil
}

// findMonsterInRoom finds a monster in the player's room by name prefix.
// Returns the MonsterInstance and its definition, or nil if not found.
func (e *GameEngine) findMonsterInRoom(player *Player, target string) (*MonsterInstance, *gameworld.MonsterDef) {
	return e.findMonsterInRoomEx(player, target, false)
}

func (e *GameEngine) findMonsterInRoomIncludeDead(player *Player, target string) (*MonsterInstance, *gameworld.MonsterDef) {
	return e.findMonsterInRoomEx(player, target, true)
}

func (e *GameEngine) findMonsterInRoomEx(player *Player, target string, includeDead bool) (*MonsterInstance, *gameworld.MonsterDef) {
	if e.monsterMgr == nil {
		return nil, nil
	}
	var monsters []MonsterInstance
	if includeDead {
		monsters = e.monsterMgr.AllMonstersInRoom(player.RoomNumber)
	} else {
		monsters = e.monsterMgr.MonstersInRoom(player.RoomNumber)
	}
	target = strings.ToLower(strings.TrimSpace(target))
	// Strip leading articles so "a skeleton" matches "skeleton"
	for _, article := range []string{"a ", "an ", "the ", "some "} {
		if strings.HasPrefix(target, article) {
			target = strings.TrimPrefix(target, article)
			break
		}
	}
	for i := range monsters {
		def := e.monsters[monsters[i].DefNumber]
		if def == nil {
			continue
		}
		name := strings.ToLower(FormatMonsterName(def, e.monAdjs))
		noun := strings.ToLower(def.Name)
		if strings.HasPrefix(name, target) || strings.HasPrefix(noun, target) {
			return &monsters[i], def
		}
	}
	return nil, nil
}

// examineMonster returns a description of a monster.
func (e *GameEngine) examineMonster(def *gameworld.MonsterDef) *CommandResult {
	name := FormatMonsterName(def, e.monAdjs)
	var msgs []string
	if def.Description != "" {
		msgs = append(msgs, def.Description)
	} else {
		msgs = append(msgs, fmt.Sprintf("You see a %s.", name))
	}
	return &CommandResult{Messages: msgs}
}

// examinePlayer returns a description of a player as seen by the observer.
func (e *GameEngine) examinePlayer(observer *Player, target *Player) *CommandResult {
	isSelf := observer.FirstName == target.FirstName && observer.LastName == target.LastName

	var pronoun string
	if isSelf {
		pronoun = "You are"
	} else if target.Gender == 0 {
		pronoun = "He is"
	} else {
		pronoun = "She is"
	}

	msgs := []string{}
	if isSelf {
		msgs = append(msgs, "You examine yourself.")
	} else if target.Title != "" {
		msgs = append(msgs, fmt.Sprintf("Before you is %s %s.", target.Title, target.FullName()))
	} else {
		msgs = append(msgs, fmt.Sprintf("You look at %s.", target.FullName()))
	}

	// Custom @line descriptions override the auto-generated race/gender line
	if target.DescLine1 != "" || target.DescLine2 != "" || target.DescLine3 != "" {
		if target.DescLine1 != "" {
			msgs = append(msgs, target.DescLine1)
		}
		if target.DescLine2 != "" {
			msgs = append(msgs, target.DescLine2)
		}
		if target.DescLine3 != "" {
			msgs = append(msgs, target.DescLine3)
		}
	} else {
		msgs = append(msgs, fmt.Sprintf("%s a %s %s.", pronoun, target.RaceName(), genderName(target.Gender)))
	}

	heOrShe := "He"
	heOrSheLC := "him"
	if target.Gender == 1 {
		heOrShe = "She"
		heOrSheLC = "her"
	}
	if isSelf {
		heOrSheLC = "you"
	}

	// Health description
	healthPct := float64(100)
	if target.MaxBodyPoints > 0 {
		healthPct = float64(target.BodyPoints) / float64(target.MaxBodyPoints) * 100
	}
	if isSelf {
		switch {
		case healthPct >= 100:
			msgs = append(msgs, "You are in perfect health.")
		case healthPct >= 75:
			msgs = append(msgs, "You have minor injuries.")
		case healthPct >= 50:
			msgs = append(msgs, "You are moderately wounded.")
		case healthPct >= 25:
			msgs = append(msgs, "You are seriously wounded.")
		case healthPct > 0:
			msgs = append(msgs, "You are critically wounded!")
		default:
			msgs = append(msgs, "You are dead.")
		}
	} else {
		switch {
		case healthPct >= 100:
			msgs = append(msgs, fmt.Sprintf("%s appears to be in perfect health.", heOrShe))
		case healthPct >= 75:
			msgs = append(msgs, fmt.Sprintf("%s has minor injuries.", heOrShe))
		case healthPct >= 50:
			msgs = append(msgs, fmt.Sprintf("%s is moderately wounded.", heOrShe))
		case healthPct >= 25:
			msgs = append(msgs, fmt.Sprintf("%s is seriously wounded.", heOrShe))
		case healthPct > 0:
			msgs = append(msgs, fmt.Sprintf("%s is critically wounded!", heOrShe))
		default:
			msgs = append(msgs, fmt.Sprintf("%s is dead.", heOrShe))
		}
	}

	// Position
	switch target.Position {
	case 1:
		msgs = append(msgs, fmt.Sprintf("%s sitting.", pronoun))
	case 2:
		msgs = append(msgs, fmt.Sprintf("%s lying down.", pronoun))
	case 3:
		msgs = append(msgs, fmt.Sprintf("%s kneeling.", pronoun))
	}

	// Visible conditions and effects
	if target.Bleeding {
		msgs = append(msgs, fmt.Sprintf("%s bleeding.", pronoun))
	}
	if target.Stunned {
		msgs = append(msgs, fmt.Sprintf("%s stunned.", pronoun))
	}
	if target.Poisoned {
		if isSelf {
			msgs = append(msgs, "You look poisoned.")
		} else {
			msgs = append(msgs, fmt.Sprintf("%s looks poisoned.", heOrShe))
		}
	}
	if target.Diseased {
		if isSelf {
			msgs = append(msgs, "You look sickly.")
		} else {
			msgs = append(msgs, fmt.Sprintf("%s looks sickly.", heOrShe))
		}
	}
	if target.Immobilized {
		msgs = append(msgs, fmt.Sprintf("%s rooted to the spot.", pronoun))
	}

	// Guard status — check if someone is guarding this target
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.GuardTarget == target.FirstName && p.RoomNumber == target.RoomNumber {
				if isSelf {
					msgs = append(msgs, fmt.Sprintf("You are being guarded by %s.", p.FirstName))
				} else {
					msgs = append(msgs, fmt.Sprintf("%s is being guarded by %s.", target.PronounCap(), p.FirstName))
				}
				break
			}
		}
	}

	// Active spell/psi effects
	if target.DefenseBonus > 0 {
		msgs = append(msgs, fmt.Sprintf("A shimmering magical aura surrounds %s.", isSelfOr(isSelf, "you", heOrSheLC)))
	}
	if target.CanFly && target.Race != RaceDrakin {
		msgs = append(msgs, fmt.Sprintf("%s hovering in the air.", pronoun))
	}
	if target.Invisible {
		// Only visible to self or GMs
		if isSelf {
			msgs = append(msgs, "You are invisible.")
		}
	}

	// Equipment
	if target.Wielded != nil {
		wDef := e.items[target.Wielded.Archetype]
		if wDef != nil {
			name := e.formatItemName(wDef, target.Wielded.Adj1, target.Wielded.Adj2, target.Wielded.Adj3)
			if isSelf {
				msgs = append(msgs, fmt.Sprintf("You are wielding %s.", name))
			} else {
				msgs = append(msgs, fmt.Sprintf("%s wielding %s.", pronoun, name))
			}
		}
	}
	var wornNames []string
	for _, worn := range target.Worn {
		wDef := e.items[worn.Archetype]
		if wDef != nil {
			wornNames = append(wornNames, e.formatItemName(wDef, worn.Adj1, worn.Adj2, worn.Adj3))
		}
	}
	if len(wornNames) > 0 {
		if isSelf {
			msgs = append(msgs, fmt.Sprintf("You are wearing %s.", joinList(wornNames)))
		} else {
			msgs = append(msgs, fmt.Sprintf("%s wearing %s.", pronoun, joinList(wornNames)))
		}
	}

	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) doGo(ctx context.Context, player *Player, args []string) *CommandResult {

	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := int(time.Until(player.RoundTimeExpiry).Seconds()) + 1
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("[Wait %d seconds...]", remaining),
			},
		}
	}

	if len(args) == 0 {
		return &CommandResult{
			Messages: []string{"Go where?"},
		}
	}

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{
			Messages: []string{"You can't go that way."},
		}
	}

	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	// ------------------------------------------------------------
	// 1. Normal directional exits.
	// ------------------------------------------------------------

	directionAliases := map[string]string{
		"n":         "N",
		"north":     "N",
		"s":         "S",
		"south":     "S",
		"e":         "E",
		"east":      "E",
		"w":         "W",
		"west":      "W",
		"ne":        "NE",
		"northeast": "NE",
		"nw":        "NW",
		"northwest": "NW",
		"se":        "SE",
		"southeast": "SE",
		"sw":        "SW",
		"southwest": "SW",
		"u":         "U",
		"up":        "U",
		"d":         "D",
		"down":      "D",
		"o":         "O",
		"out":       "O",
		"above":     "ABOVE",
		"below":     "BELOW",
	}

	if dir, ok := directionAliases[target]; ok {
		if destNum, exists := room.Exits[dir]; exists {

			dest := e.rooms[destNum]
			if dest == nil {
				return &CommandResult{
					Messages: []string{"You can't go that way."},
				}
			}

			oldRoom := player.RoomNumber

			// Move leader first.
			player.RoomNumber = destNum
			player.Submitting = false
			e.disengageCombat(player)
			e.SavePlayer(ctx, player)

			// ----------------------------------------------------
			// Move ALL followers before generating anyone's LOOK.
			//
			// This matters for shared light sources:
			// if a follower has a lit torch, they need to already
			// be in the destination when roomVisibility() runs.
			// ----------------------------------------------------

			var movedFollowers []*Player

			if player.IsGroupLeader &&
				len(player.GroupMembers) > 0 &&
				e.sessions != nil {

				for _, memberName := range player.GroupMembers {

					for _, p := range e.sessions.OnlinePlayers() {

						if p.FirstName != memberName {
							continue
						}

						if p.RoomNumber != oldRoom {
							continue
						}

						if p.Dead {
							continue
						}

						p.RoomNumber = destNum
						p.Submitting = false
						e.disengageCombat(p)
						e.SavePlayer(ctx, p)

						movedFollowers = append(movedFollowers, p)

						break
					}
				}
			}

			// ----------------------------------------------------
			// Everyone is now physically in the destination.
			// Leader LOOK can correctly detect group light.
			// ----------------------------------------------------

			result := e.doLook(player)

			result.OldRoom = oldRoom

			result.OldRoomMsg = []string{
				fmt.Sprintf("%s leaves.", player.FirstName),
			}

			result.RoomBroadcast = append(
				result.RoomBroadcast,
				fmt.Sprintf("%s arrives.", player.FirstName),
			)

			// Group movement messages.
			if len(movedFollowers) > 0 {
				result.OldRoomMsg = append(
					result.OldRoomMsg,
					fmt.Sprintf("%s's group leaves.", player.FirstName),
				)

				result.RoomBroadcast = append(
					result.RoomBroadcast,
					fmt.Sprintf("%s's group arrives.", player.FirstName),
				)
			}

			// ----------------------------------------------------
			// Now send followers their room LOOKs.
			// Everyone has moved, so they also see shared light.
			// ----------------------------------------------------

			for _, p := range movedFollowers {

				if e.sendToPlayer != nil {
					followLook := e.doLook(p)
					e.sendToPlayer(
						p.FirstName,
						followLook.Messages,
					)
				}

				e.applyEntryScripts(
					ctx,
					p,
					dest,
					&CommandResult{},
				)
			}

			// Leader entry scripts.
			e.applyEntryScripts(
				ctx,
				player,
				dest,
				result,
			)

			return result
		}
	}

	// ------------------------------------------------------------
	// 2. GO <item>
	//
	// Handles real portals and scripted items such as paths,
	// ladders, arches, trees, etc.
	// ------------------------------------------------------------

	for i, ri := range room.Items {

		if ri.IsPut {
			continue
		}

		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}

		name := e.getItemNounName(itemDef)

		// Match the noun against any of the item's adjectives.
		if !matchesTarget(name, target, e.getAdjName(ri.Adj1)) &&
			!matchesTarget(name, target, e.getAdjName(ri.Adj2)) &&
			!matchesTarget(name, target, e.getAdjName(ri.Adj3)) {

			continue
		}

		if skip > 0 {
			skip--
			continue
		}

		// --------------------------------------------------------
		// Real portal.
		// --------------------------------------------------------

		if isPortal(itemDef.Type) {
			return e.doGoPortal(
				ctx,
				player,
				room,
				&room.Items[i],
				itemDef,
			)
		}

		// --------------------------------------------------------
		// Scripted non-portal item.
		// --------------------------------------------------------

		result := &CommandResult{}
		originalRoom := player.RoomNumber

		// --------------------------------------------------------
		// First: IFPREVERB GO
		// --------------------------------------------------------

		pre := e.RunPreverbScripts(
			player,
			room,
			"GO",
			&room.Items[i],
			itemDef,
		)

		result.Messages = append(
			result.Messages,
			pre.Messages...,
		)

		result.RoomBroadcast = append(
			result.RoomBroadcast,
			pre.RoomMsgs...,
		)

		result.GMBroadcast = append(
			result.GMBroadcast,
			pre.GMMsgs...,
		)

		if e.applyGoScriptResult(
			ctx,
			player,
			originalRoom,
			pre,
			result,
		) {
			return result
		}

		// --------------------------------------------------------
		// Second: IFVERB GO
		// --------------------------------------------------------

		verb := e.RunVerbScripts(
			player,
			room,
			"GO",
			&room.Items[i],
			itemDef,
		)

		result.Messages = append(
			result.Messages,
			verb.Messages...,
		)

		result.RoomBroadcast = append(
			result.RoomBroadcast,
			verb.RoomMsgs...,
		)

		result.GMBroadcast = append(
			result.GMBroadcast,
			verb.GMMsgs...,
		)

		if e.applyGoScriptResult(
			ctx,
			player,
			originalRoom,
			verb,
			result,
		) {
			return result
		}

		// Item matched, but neither PREVERB nor VERB handled GO.
		if len(result.Messages) == 0 {
			result.Messages = []string{
				"You can't go that way.",
			}
		}

		return result
	}

	return &CommandResult{
		Messages: []string{"You can't go that way."},
	}
}

func (e *GameEngine) applyGoScriptResult(
	ctx context.Context,
	player *Player,
	originalRoom int,
	sc *ScriptContext,
	result *CommandResult,
) bool {

	// MOVEGROUP requested by the script.
	// moveGroupToRoom handles movement and sends LOOK to moved players.
	if sc.MoveGroupTo > 0 {
		e.moveGroupToRoom(
			ctx,
			originalRoom,
			sc.MoveGroupTo,
		)

		dest := e.rooms[player.RoomNumber]
		if dest != nil {
			lookResult := e.doLook(player)

			result.Messages = append(
				result.Messages,
				lookResult.Messages...,
			)

			result.RoomName = lookResult.RoomName
			result.RoomDesc = lookResult.RoomDesc
			result.Exits = lookResult.Exits
			result.Items = lookResult.Items
			result.OldRoom = originalRoom
		}

		return true

	}

	// MOVE requested by the script.
	if sc.MoveTo > 0 {
		dest := e.rooms[sc.MoveTo]
		if dest == nil {
			return true
		}

		player.RoomNumber = sc.MoveTo
		e.SavePlayer(ctx, player)

		lookResult := e.doLook(player)

		result.Messages = append(
			result.Messages,
			lookResult.Messages...,
		)

		result.RoomName = lookResult.RoomName
		result.RoomDesc = lookResult.RoomDesc
		result.Exits = lookResult.Exits
		result.Items = lookResult.Items
		result.OldRoom = originalRoom

		e.applyEntryScripts(
			ctx,
			player,
			dest,
			result,
		)

		return true
	}

	// Some script actions may perform movement directly.
	if player.RoomNumber != originalRoom {
		dest := e.rooms[player.RoomNumber]

		e.SavePlayer(ctx, player)

		if dest != nil {
			lookResult := e.doLook(player)

			result.Messages = append(
				result.Messages,
				lookResult.Messages...,
			)

			result.RoomName = lookResult.RoomName
			result.RoomDesc = lookResult.RoomDesc
			result.Exits = lookResult.Exits
			result.Items = lookResult.Items
			result.OldRoom = originalRoom

			e.applyEntryScripts(
				ctx,
				player,
				dest,
				result,
			)
		}

		return true
	}

	// CLEARVERB with no movement means the script blocked GO.
	if sc.Blocked {
		if len(result.Messages) == 0 {
			result.Messages = []string{
				"You can't go that way.",
			}
		}

		return true
	}

	return false
}

func (e *GameEngine) movePlayerToRoom(ctx context.Context, player *Player, roomNum int, result *CommandResult) {
	room := e.rooms[roomNum]
	if room == nil {
		return
	}

	oldRoom := player.RoomNumber

	player.RoomNumber = roomNum
	e.SavePlayer(ctx, player)

	look := e.doLook(player)

	result.Messages = append(result.Messages, look.Messages...)
	result.RoomName = look.RoomName
	result.RoomDesc = look.RoomDesc
	result.Exits = look.Exits
	result.Items = look.Items
	result.OldRoom = oldRoom

	e.applyEntryScripts(ctx, player, room, result)
}

func (e *GameEngine) doGoPortal(ctx context.Context, player *Player, room *gameworld.Room, ri *gameworld.RoomItem, itemDef *gameworld.ItemDef) *CommandResult {
	// Check if portal is closed
	state := strings.ToUpper(ri.State)
	if state == "CLOSED" || state == "LOCKED" {
		portalName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
		return &CommandResult{Messages: []string{fmt.Sprintf("The %s is closed.", e.getItemNounName(itemDef))}, RoomBroadcast: []string{fmt.Sprintf("%s bumps into %s.", player.FirstName, portalName)}}
	}

	// Run IFPREVERB GO scripts (can CLEARVERB to block)
	sc := e.RunPreverbScripts(player, room, "GO", ri, itemDef)
	result := &CommandResult{}
	if len(sc.Messages) > 0 {
		result.Messages = append(result.Messages, sc.Messages...)
	}
	if len(sc.RoomMsgs) > 0 {
		result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
	}
	if len(sc.GMMsgs) > 0 {
		result.GMBroadcast = append(result.GMBroadcast, sc.GMMsgs...)
	}
	if sc.Blocked && sc.MoveTo == 0 {
		if len(result.Messages) == 0 {
			result.Messages = []string{"You can't go that way."}
		}
		return result
	}

	// Script MOVE overrides destination
	destNum := ri.Val2
	if sc.MoveTo > 0 {
		destNum = sc.MoveTo
	}

	if destNum <= 0 {
		result.Messages = append(result.Messages, "That doesn't seem to lead anywhere.")
		return result
	}
	dest := e.rooms[destNum]
	if dest == nil {
		result.Messages = append(result.Messages, "That doesn't seem to lead anywhere.")
		return result
	}

	oldRoom := player.RoomNumber
	portalName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
	player.RoomNumber = destNum
	e.SavePlayer(ctx, player)

	// Group movement: if leader has followers, move them through the portal too
	if player.IsGroupLeader && len(player.GroupMembers) > 0 && e.sessions != nil {
		for _, memberName := range player.GroupMembers {
			for _, p := range e.sessions.OnlinePlayers() {
				if p.FirstName == memberName && p.RoomNumber == oldRoom && !p.Dead {
					p.RoomNumber = destNum
					p.Submitting = false
					e.disengageCombat(p)
					e.SavePlayer(ctx, p)
					if e.sendToPlayer != nil {
						followLook := e.doLook(p)
						e.sendToPlayer(p.FirstName, followLook.Messages)
					}
					e.applyEntryScripts(ctx, p, dest, &CommandResult{})
					break
				}
			}
		}
		result.OldRoomMsg = append(result.OldRoomMsg, fmt.Sprintf("%s's group goes through %s.", player.FirstName, portalName))
		result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s's group arrives.", player.FirstName))
	}

	// Commanded creatures following the player move through the portal too
	e.moveCommandedFollowers(player, oldRoom, destNum)

	lookResult := e.doLook(player)
	result.Messages = append(result.Messages, lookResult.Messages...)
	result.RoomName = lookResult.RoomName
	result.RoomDesc = lookResult.RoomDesc
	result.Exits = lookResult.Exits
	result.Items = lookResult.Items
	result.OldRoom = oldRoom
	result.OldRoomMsg = []string{fmt.Sprintf("%s goes through %s.", player.FirstName, portalName)}
	result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s arrives.", player.FirstName))

	// Run IFENTRY scripts at destination
	e.applyEntryScripts(ctx, player, dest, result)

	return result
}

func (e *GameEngine) moveCommandedFollowers(player *Player, oldRoom, newRoom int) {
	if e.monsterMgr == nil {
		return
	}

	var followers []int

	e.monsterMgr.mu.Lock()

	for i := range e.monsterMgr.instances {
		inst := &e.monsterMgr.instances[i]

		if !inst.Alive ||
			inst.RoomNumber != oldRoom ||
			inst.CommanderID != player.FirstName ||
			inst.Following != player.FirstName {
			continue
		}

		followers = append(followers, i)
	}

	e.monsterMgr.mu.Unlock()

	for _, idx := range followers {
		e.monsterMgr.MoveMonster(idx, newRoom)
	}
}

func (e *GameEngine) doRerollStats(ctx context.Context, player *Player, args []string) *CommandResult {

	if player.Level >= 2 {
		return &CommandResult{
			Messages: []string{
				"You may only reroll your attributes while you are level 1.",
			},
			PlayerState: player,
		}
	}

	argString := strings.ToUpper(strings.Join(args, " "))

	switch argString {

	case "STATS":

		if player.RoundTimeExpiry.After(time.Now()) {
			remaining := int(time.Until(player.RoundTimeExpiry).Seconds()) + 1

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf("[Wait %d seconds...]", remaining),
				},
				PlayerState: player,
			}
		}

		stats := rollStatsForRace(player.Race)
		player.PendingStatReroll = &stats

		player.RoundTimeExpiry = time.Now().Add(5 * time.Second)

		return &CommandResult{
			Messages: []string{
				"Potential new attributes:",
				"",
				fmt.Sprintf(
					"Strength: %d   Agility: %d   Quickness: %d",
					stats.Strength,
					stats.Agility,
					stats.Quickness,
				),
				fmt.Sprintf(
					"Constitution: %d   Perception: %d   Willpower: %d   Empathy: %d",
					stats.Constitution,
					stats.Perception,
					stats.Willpower,
					stats.Empathy,
				),
				"",
				"Type REROLL STATS to roll again.",
				"Type REROLL STATS CONFIRM to accept these attributes.",
				"[Reroll: 5 sec]",
			},
			PlayerState: player,
		}

	case "STATS CONFIRM":
		if player.PendingStatReroll == nil {
			return &CommandResult{
				Messages: []string{
					"You don't have a pending stat roll. Type REROLL STATS first.",
				},
				PlayerState: player,
			}
		}

		stats := player.PendingStatReroll

		player.Strength = stats.Strength
		player.Agility = stats.Agility
		player.Quickness = stats.Quickness
		player.Constitution = stats.Constitution
		player.Perception = stats.Perception
		player.Willpower = stats.Willpower
		player.Empathy = stats.Empathy

		// Recalculate derived stats using the same formulas
		// as character creation.
		bodyPts := 20 + player.Constitution/2
		fatigue := 20 + (player.Constitution+player.Strength)/3
		mana := player.Empathy / 2
		psi := player.Willpower / 2

		player.BodyPoints = bodyPts
		player.MaxBodyPoints = bodyPts

		player.Fatigue = fatigue
		player.MaxFatigue = fatigue

		player.Mana = mana
		player.MaxMana = mana

		player.Psi = psi
		player.MaxPsi = psi

		player.PendingStatReroll = nil

		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages: []string{
				"Your new attributes have been accepted.",
				"",
				fmt.Sprintf(
					"Strength: %d   Agility: %d   Quickness: %d",
					player.Strength,
					player.Agility,
					player.Quickness,
				),
				fmt.Sprintf(
					"Constitution: %d   Perception: %d   Willpower: %d   Empathy: %d",
					player.Constitution,
					player.Perception,
					player.Willpower,
					player.Empathy,
				),
			},
			PlayerState: player,
		}

	default:
		return &CommandResult{
			Messages: []string{
				"Usage: REROLL STATS",
				"       REROLL STATS CONFIRM",
			},
			PlayerState: player,
		}
	}
}

func (e *GameEngine) doClimb(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Climb what?"}}
	}
	if player.Position != 0 && player.Position != 4 {
		posNames := map[int]string{1: "sitting", 2: "laying down", 3: "kneeling"}
		posName := posNames[player.Position]
		if posName == "" {
			posName = "not standing"
		}
		return &CommandResult{Messages: []string{fmt.Sprintf("You can't climb while %s! Try STANDing first.", posName)}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}

	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			if skip > 0 {
				skip--
				continue
			}
			if isPortal(itemDef.Type) {
				return e.doGoPortal(ctx, player, room, &room.Items[i], itemDef)
			}
			// Run IFPREVERB CLIMB scripts on non-portal items
			sc := e.RunPreverbScripts(player, room, "CLIMB", &room.Items[i], itemDef)
			result := &CommandResult{}
			result.Messages = append(result.Messages, sc.Messages...)
			result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
			result.GMBroadcast = append(result.GMBroadcast, sc.GMMsgs...)
			if sc.MoveTo > 0 {
				dest := e.rooms[sc.MoveTo]
				if dest != nil {
					oldRoom := player.RoomNumber
					player.RoomNumber = sc.MoveTo
					leftBehind := e.breakCommandedFollowers(player, oldRoom)
					e.SavePlayer(ctx, player)
					lookResult := e.doLook(player)
					result.Messages = append(result.Messages, lookResult.Messages...)
					result.Messages = append(result.Messages, leftBehind...)
					result.RoomName = lookResult.RoomName
					result.RoomDesc = lookResult.RoomDesc
					result.Exits = lookResult.Exits
					result.Items = lookResult.Items
					result.OldRoom = oldRoom
					result.OldRoomMsg = []string{fmt.Sprintf("%s leaves.", player.FirstName)}
					result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s arrives.", player.FirstName))
					e.applyEntryScripts(ctx, player, dest, result)
				}
			}
			if len(result.Messages) == 0 {
				result.Messages = []string{"You can't climb that."}
			}
			return result
		}
	}

	return &CommandResult{Messages: []string{"You don't see that here."}}
}

func (e *GameEngine) breakCommandedFollowers(player *Player, roomNum int) []string {
	if e.monsterMgr == nil {
		return nil
	}

	var messages []string

	e.monsterMgr.mu.Lock()
	defer e.monsterMgr.mu.Unlock()

	for i := range e.monsterMgr.instances {
		inst := &e.monsterMgr.instances[i]

		if !inst.Alive ||
			inst.RoomNumber != roomNum ||
			inst.CommanderID != player.FirstName ||
			inst.Following != player.FirstName {
			continue
		}

		def := e.monsters[inst.DefNumber]
		if def == nil {
			continue
		}

		inst.Following = ""

		name := FormatMonsterName(def, e.monAdjs)

		messages = append(
			messages,
			fmt.Sprintf("Your %s cannot follow you and remains behind.", name),
		)
	}

	return messages
}

// doItemInteraction handles verbs like PULL, PUSH, TURN, RUB, TAP, TOUCH, SEARCH, DIG.
// These run IFPREVERB scripts on the target item. If no script handles it, a default message is shown.
func (e *GameEngine) doItemInteraction(ctx context.Context, player *Player, verb string, args []string) *CommandResult {
	verbLower := strings.ToLower(verb)

	if len(args) == 0 {
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("%s what?", strings.Title(verbLower)),
			},
		}
	}

	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{
			Messages: []string{"You can't do that here."},
		}
	}

	// ------------------------------------------------------------
	// ROOM ITEMS
	// ------------------------------------------------------------
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}

		name := e.getItemNounName(itemDef)

		if !matchesTarget(name, target, e.getAdjName(ri.Adj1)) &&
			!matchesTarget(name, target, e.getAdjName(ri.Adj2)) &&
			!matchesTarget(name, target, e.getAdjName(ri.Adj3)) {
			continue
		}

		if skip > 0 {
			skip--
			continue
		}

		// Identify currently applies to carried/worn/wielded item instances.
		if verb == "IDENTIFY" {
			return &CommandResult{
				Messages: []string{"You must be holding that to identify it."},
			}
		}

		// Detect Magic only needs item resolution + magic power.
		if verb == "DETECTMAGIC" {
			return &CommandResult{
				MagicPower: e.itemMagicPower(itemDef, &room.Items[i]),
			}
		}

		result := &CommandResult{}

		// Preverb scripts
		sc := e.RunPreverbScripts(
			player,
			room,
			verb,
			&room.Items[i],
			itemDef,
		)

		result.Messages = append(result.Messages, sc.Messages...)
		result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
		result.GMBroadcast = append(result.GMBroadcast, sc.GMMsgs...)

		// Verb scripts
		sc2 := e.RunVerbScripts(
			player,
			room,
			verb,
			&room.Items[i],
			itemDef,
		)

		result.Messages = append(result.Messages, sc2.Messages...)
		result.RoomBroadcast = append(result.RoomBroadcast, sc2.RoomMsgs...)
		result.GMBroadcast = append(result.GMBroadcast, sc2.GMMsgs...)

		if (sc.Blocked || sc2.Blocked) &&
			sc.MoveTo == 0 &&
			sc2.MoveTo == 0 {

			if len(result.Messages) == 0 {
				result.Messages = []string{"You can't do that."}
			}

			return result
		}

		moveTo := sc.MoveTo
		if sc2.MoveTo > 0 {
			moveTo = sc2.MoveTo
		}

		if moveTo > 0 {
			dest := e.rooms[moveTo]
			if dest != nil {
				oldRoom := player.RoomNumber
				player.RoomNumber = moveTo

				e.SavePlayer(ctx, player)

				lookResult := e.doLook(player)

				result.Messages = append(
					result.Messages,
					lookResult.Messages...,
				)

				result.RoomName = lookResult.RoomName
				result.RoomDesc = lookResult.RoomDesc
				result.Exits = lookResult.Exits
				result.Items = lookResult.Items

				result.OldRoom = oldRoom
				result.OldRoomMsg = []string{
					fmt.Sprintf("%s leaves.", player.FirstName),
				}

				result.RoomBroadcast = append(
					result.RoomBroadcast,
					fmt.Sprintf("%s arrives.", player.FirstName),
				)

				e.applyEntryScripts(ctx, player, dest, result)
			}
		}

		if len(result.Messages) == 0 {
			itemName := e.formatItemName(
				itemDef,
				ri.Adj1,
				ri.Adj2,
				ri.Adj3,
			)

			result.Messages = []string{
				fmt.Sprintf(
					"You %s %s. Nothing happens.",
					verbLower,
					itemName,
				),
			}
		}

		return result
	}

	// ------------------------------------------------------------
	// PLAYER ITEM HELPER
	// ------------------------------------------------------------
	runPlayerItem := func(ii *InventoryItem) *CommandResult {
		if ii == nil {
			return nil
		}

		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			return nil
		}

		name := e.getItemNounName(itemDef)

		if !matchesTarget(name, target, e.getAdjName(ii.Adj1)) &&
			!matchesTarget(name, target, e.getAdjName(ii.Adj2)) &&
			!matchesTarget(name, target, e.getAdjName(ii.Adj3)) {
			return nil
		}

		if skip > 0 {
			skip--
			return nil
		}

		// Scripts currently expect a RoomItem, so make a temporary
		// script-context item from the real InventoryItem.
		tempRI := gameworld.RoomItem{
			Ref:       -1,
			Archetype: ii.Archetype,
			Adj1:      ii.Adj1,
			Adj2:      ii.Adj2,
			Adj3:      ii.Adj3,
			Val1:      ii.Val1,
			Val2:      ii.Val2,
			Val3:      ii.Val3,
			Val4:      ii.Val4,
			Val5:      ii.Val5,
			Traits:    ii.Traits,
		}

		// Detect Magic only needs item resolution + magic power.
		if verb == "DETECTMAGIC" {
			return &CommandResult{
				MagicPower: e.itemMagicPower(itemDef, &tempRI),
			}
		}

		// Identify modifies the actual InventoryItem instance.
		if verb == "IDENTIFY" {
			result := &CommandResult{
				Messages: e.identifyItem(player, ii),
			}

			e.SavePlayer(ctx, player)

			return result
		}

		result := &CommandResult{}

		// Run IFPREVERB first.
		sc := e.RunPreverbScripts(
			player,
			room,
			verb,
			&tempRI,
			itemDef,
		)

		result.Messages = append(result.Messages, sc.Messages...)
		result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
		result.GMBroadcast = append(result.GMBroadcast, sc.GMMsgs...)

		// If the preverb did not block the command, run IFVERB.
		var sc2 *ScriptContext
		if !sc.Blocked {
			sc2 = e.RunVerbScripts(
				player,
				room,
				verb,
				&tempRI,
				itemDef,
			)

			result.Messages = append(result.Messages, sc2.Messages...)
			result.RoomBroadcast = append(result.RoomBroadcast, sc2.RoomMsgs...)
			result.GMBroadcast = append(result.GMBroadcast, sc2.GMMsgs...)
		}

		// Persist any ITEMVAL changes made by either script.
		ii.Val1 = tempRI.Val1
		ii.Val2 = tempRI.Val2
		ii.Val3 = tempRI.Val3
		ii.Val4 = tempRI.Val4
		ii.Val5 = tempRI.Val5

		e.SavePlayer(ctx, player)

		// CLEARVERB from the preverb stops normal handling.
		if sc.Blocked {
			if len(result.Messages) == 0 {
				result.Messages = []string{"You can't do that."}
			}
			return result
		}

		// IFVERB may also block.
		if sc2 != nil && sc2.Blocked {
			if len(result.Messages) == 0 {
				result.Messages = []string{"You can't do that."}
			}
			return result
		}

		// If either script produced output, return it.
		if len(result.Messages) > 0 ||
			len(result.RoomBroadcast) > 0 ||
			len(result.GMBroadcast) > 0 {
			return result
		}

		itemName := e.formatItemName(
			itemDef,
			ii.Adj1,
			ii.Adj2,
			ii.Adj3,
		)

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You %s %s. Nothing happens.",
					verbLower,
					itemName,
				),
			},
		}
	}

	// ------------------------------------------------------------
	// INVENTORY
	// ------------------------------------------------------------
	for i := range player.Inventory {
		if result := runPlayerItem(&player.Inventory[i]); result != nil {
			return result
		}
	}

	// ------------------------------------------------------------
	// WORN
	// ------------------------------------------------------------
	for i := range player.Worn {
		if result := runPlayerItem(&player.Worn[i]); result != nil {
			return result
		}
	}

	// ------------------------------------------------------------
	// WIELDED
	// ------------------------------------------------------------
	if player.Wielded != nil {
		if result := runPlayerItem(player.Wielded); result != nil {
			return result
		}
	}

	// Detect Magic / Identify need a straightforward not-found result.
	if verb == "DETECTMAGIC" || verb == "IDENTIFY" {
		return &CommandResult{
			Messages: []string{"You don't see that here."},
		}
	}

	// Fall back to emote if the verb has one.
	if _, hasEmote := emoteTable[verb]; hasEmote {
		return e.processEmote(player, verb, args)
	}

	return &CommandResult{
		Messages: []string{"You don't see that here."},
	}
}

func (e *GameEngine) doATest(ctx context.Context, player *Player) *CommandResult {

	for i := range player.Inventory {
		item := &player.Inventory[i]

		// Short sword archetype
		if item.Archetype != 45 {
			continue
		}

		hasTrait := false
		for _, trait := range item.Traits {
			if strings.EqualFold(trait, "GLOWING") {
				hasTrait = true
				break
			}
		}

		if !hasTrait {
			item.Traits = append(item.Traits, "GLOWING")
		}

		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"TEST: archetype=%d adj1=%d adj2=%d adj3=%d traits=%v",
					item.Archetype,
					item.Adj1,
					item.Adj2,
					item.Adj3,
					item.Traits,
				),
			},
			PlayerState: player,
		}
	}

	return &CommandResult{
		Messages: []string{
			"TEST: No archetype 45 sword found.",
		},
		PlayerState: player,
	}
}

func (e *GameEngine) doGet(ctx context.Context, player *Player, args []string) *CommandResult {

	if len(args) == 0 {
		return &CommandResult{
			Messages: []string{"Get what?"},
		}
	}

	raw := strings.ToLower(strings.Join(args, " "))

	if strings.Contains(raw, " from ") {
		return e.doGetFromContainer(ctx, player, raw)
	}

	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{
			Messages: []string{"You can't do that here."},
		}
	}

	for i, ri := range room.Items {
		if ri.IsPut {
			continue
		}

		// ------------------------------------------------------------
		// Coin piles without a valid item archetype.
		// ------------------------------------------------------------
		if ri.State == "MONEY" &&
			(target == "coins" ||
				target == "money" ||
				target == "coin" ||
				target == "gold" ||
				target == "silver" ||
				target == "copper") {

			coins := ri.Val1
			if coins <= 0 {
				coins = 1
			}

			room.Items = append(
				room.Items[:i],
				room.Items[i+1:]...,
			)

			e.notifyRoomChange(RoomChange{
				RoomNumber: player.RoomNumber,
				Type:       "item_remove",
				ItemRef:    ri.Ref,
			})

			player.Copper += coins

			player.Silver += player.Copper / 10
			player.Copper %= 10

			player.Gold += player.Silver / 10
			player.Silver %= 10

			e.SavePlayer(ctx, player)

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf("You pick up %d coins.", coins),
				},
				RoomBroadcast: []string{
					fmt.Sprintf(
						"%s picks up some coins.",
						player.FirstName,
					),
				},
			}
		}

		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}

		// ------------------------------------------------------------
		// First determine whether this is actually the target.
		//
		// IMPORTANT:
		// Do this BEFORE normal GET restrictions so a scripted
		// IFPREVERB GET can intercept normally un-gettable scenery.
		// ------------------------------------------------------------

		name := e.getItemNounName(itemDef)

		if !matchesTarget(
			name,
			target,
			e.getAdjName(ri.Adj1),
		) {
			continue
		}

		if skip > 0 {
			skip--
			continue
		}

		/*
			Copy the selected item before running its script.

			The script may execute NEWPUT, which can append another item
			to room.Items. Using a copy prevents ScriptContext.ItemRef from
			becoming an invalid pointer if the room item slice reallocates.
		*/
		pickedUp := ri

		// ------------------------------------------------------------
		// PREVERB GET
		//
		// Give room/item scripts a chance to replace normal GET before
		// we reject the item for being fixed, too heavy, a portal, etc.
		// ------------------------------------------------------------

		sc := e.RunPreverbScripts(
			player,
			room,
			"GET",
			&pickedUp,
			itemDef,
		)

		result := &CommandResult{}

		result.Messages = append(
			result.Messages,
			sc.Messages...,
		)

		result.RoomBroadcast = append(
			result.RoomBroadcast,
			sc.RoomMsgs...,
		)

		result.GMBroadcast = append(
			result.GMBroadcast,
			sc.GMMsgs...,
		)

		// CLEARVERB means the script completely handled GET.
		if sc.Blocked {
			if len(result.Messages) == 0 {
				result.Messages = append(
					result.Messages,
					"You can't get that.",
				)
			}

			e.SavePlayer(ctx, player)

			return result
		}

		// ------------------------------------------------------------
		// Normal GET restrictions.
		//
		// These happen AFTER PREVERB so scripts can override them.
		// ------------------------------------------------------------

		if itemDef.Weight >= 1000 {
			continue
		}

		if isPortal(itemDef.Type) {
			continue
		}

		if containsFlag(itemDef.Flags, "FIXED") ||
			itemDef.Type == "MANUSCRIPT" {
			continue
		}

		// ------------------------------------------------------------
		// MONEY item definitions automatically convert to currency.
		// ------------------------------------------------------------

		if itemDef.Type == "MONEY" || ri.State == "MONEY" {
			coins := ri.Val1
			if coins <= 0 {
				coins = 1
			}

			room.Items = append(
				room.Items[:i],
				room.Items[i+1:]...,
			)

			e.notifyRoomChange(RoomChange{
				RoomNumber: player.RoomNumber,
				Type:       "item_remove",
				ItemRef:    ri.Ref,
			})

			player.Copper += coins

			player.Silver += player.Copper / 10
			player.Copper %= 10

			player.Gold += player.Silver / 10
			player.Silver %= 10

			e.SavePlayer(ctx, player)

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf("You pick up %d coins.", coins),
				},
				RoomBroadcast: []string{
					fmt.Sprintf(
						"%s picks up some coins.",
						player.FirstName,
					),
				},
			}
		}

		// ------------------------------------------------------------
		// Collect container contents.
		// ------------------------------------------------------------

		var contents []InventoryItem

		for _, child := range room.Items {
			if !child.IsPut || child.PutIn != pickedUp.Ref {
				continue
			}

			contents = append(
				contents,
				InventoryItem{
					Archetype: child.Archetype,
					Adj1:      child.Adj1,
					Adj2:      child.Adj2,
					Adj3:      child.Adj3,
					Val1:      child.Val1,
					Val2:      child.Val2,
					Val3:      child.Val3,
					Val4:      child.Val4,
					Val5:      child.Val5,
					State:     child.State,
				},
			)
		}

		// ------------------------------------------------------------
		// Move item into player's inventory.
		// ------------------------------------------------------------

		player.Inventory = append(
			player.Inventory,
			InventoryItem{
				Archetype: pickedUp.Archetype,
				Adj1:      pickedUp.Adj1,
				Adj2:      pickedUp.Adj2,
				Adj3:      pickedUp.Adj3,
				Val1:      pickedUp.Val1,
				Val2:      pickedUp.Val2,
				Val3:      pickedUp.Val3,
				Val4:      pickedUp.Val4,
				Val5:      pickedUp.Val5,
				State:     pickedUp.State,
				Contents:  contents,
			},
		)

		// ------------------------------------------------------------
		// Remove anything that was inside the picked-up container
		// from the room.
		// ------------------------------------------------------------

		if len(contents) > 0 {
			filtered := room.Items[:0]

			for _, item := range room.Items {
				if item.IsPut &&
					item.PutIn == pickedUp.Ref {

					continue
				}

				filtered = append(
					filtered,
					item,
				)
			}

			room.Items = filtered
		}

		/*
			NEWPUT may have appended a replacement item while the script
			was running.

			Find the original item again instead of assuming the slice
			is unchanged.
		*/
		removeIndex := -1

		for j := range room.Items {
			candidate := room.Items[j]

			if candidate.Ref == pickedUp.Ref &&
				candidate.Archetype == pickedUp.Archetype &&
				candidate.Adj1 == pickedUp.Adj1 &&
				candidate.Adj2 == pickedUp.Adj2 &&
				candidate.Adj3 == pickedUp.Adj3 {

				removeIndex = j
				break
			}
		}

		if removeIndex >= 0 {
			room.Items = append(
				room.Items[:removeIndex],
				room.Items[removeIndex+1:]...,
			)

			e.notifyRoomChange(RoomChange{
				RoomNumber: player.RoomNumber,
				Type:       "item_remove",
				ItemRef:    pickedUp.Ref,
			})
		}

		e.SavePlayer(ctx, player)

		fullName := e.formatItemName(
			itemDef,
			pickedUp.Adj1,
			pickedUp.Adj2,
			pickedUp.Adj3,
		)

		result.Messages = append(
			result.Messages,
			fmt.Sprintf(
				"You pick up %s.",
				fullName,
			),
		)

		result.RoomBroadcast = append(
			result.RoomBroadcast,
			fmt.Sprintf(
				"%s picks up %s.",
				player.FirstName,
				fullName,
			),
		)

		return result
	}

	return &CommandResult{
		Messages: []string{"You don't see that here."},
	}
}

func (e *GameEngine) doGetFromContainer(ctx context.Context, player *Player, raw string) *CommandResult {

	parts := strings.SplitN(raw, " from ", 2)
	if len(parts) != 2 {
		return &CommandResult{
			Messages: []string{"Get what from what?"},
		}
	}

	itemTarget := strings.TrimSpace(parts[0])
	containerTarget := strings.TrimSpace(parts[1])

	if itemTarget == "" || containerTarget == "" {
		return &CommandResult{
			Messages: []string{"Get what from what?"},
		}
	}

	// ------------------------------------------------------------
	// Optional relationship:
	//
	//   GET DIRK FROM BED
	//   GET DIRK FROM BEHIND BED
	//   GET DIRK FROM ON BED
	//   GET COIN FROM IN CHEST
	//
	// This allows otherwise identical room items to be
	// distinguished by their container relationship.
	// ------------------------------------------------------------

	relation := ""

	for _, p := range []string{
		"in ",
		"on ",
		"under",
		"behind ",
	} {
		if strings.HasPrefix(containerTarget, p) {
			relation = strings.ToUpper(strings.TrimSpace(p))
			containerTarget = strings.TrimSpace(
				strings.TrimPrefix(containerTarget, p),
			)
			break
		}
	}

	if containerTarget == "" {
		return &CommandResult{
			Messages: []string{"Get what from what?"},
		}
	}

	// Support:
	// GET COIN FROM CHEST 2
	// GET COIN FROM SECOND CHEST
	containerTarget, containerOrdSkip := parseOrdinal(containerTarget)

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{
			Messages: []string{"You can't do that here."},
		}
	}

	// ------------------------------------------------------------
	// Find a matching room container.
	// ------------------------------------------------------------

	var container *gameworld.RoomItem
	var containerDef *gameworld.ItemDef

	containerSkip := containerOrdSkip

	for i := range room.Items {
		ri := &room.Items[i]

		// Containers themselves must be top-level room items.
		if ri.IsPut {
			continue
		}

		def := e.items[ri.Archetype]
		if def == nil {
			continue
		}

		name := e.getItemNounName(def)

		if !matchesTarget(
			name,
			containerTarget,
			e.getAdjName(ri.Adj1),
		) &&
			!matchesTarget(
				name,
				containerTarget,
				e.getAdjName(ri.Adj2),
			) &&
			!matchesTarget(
				name,
				containerTarget,
				e.getAdjName(ri.Adj3),
			) {
			continue
		}

		// Only count actual containers toward the ordinal.
		if def.Container != "IN" &&
			def.Container != "ON" &&
			def.Container != "BEHIND" &&
			def.Container != "UNDER" &&
			def.Type != "CONTAINER" &&
			!containsFlag(def.Flags, "CONTAINER") {
			continue
		}

		// If the player explicitly said FROM BEHIND / FROM ON /
		// FROM IN, the underlying object must match that relation.
		if relation != "" &&
			!strings.EqualFold(def.Container, relation) {
			continue
		}

		if containerSkip > 0 {
			containerSkip--
			continue
		}

		container = ri
		containerDef = def
		break
	}

	// ------------------------------------------------------------
	// If no room container matched, try carried containers.
	// ------------------------------------------------------------

	if container == nil {
		containerSkip = containerOrdSkip

		for ci := range player.Inventory {
			invContainer := &player.Inventory[ci]

			def := e.items[invContainer.Archetype]
			if def == nil {
				continue
			}

			name := e.getItemNounName(def)

			if !matchesTarget(
				name,
				containerTarget,
				e.getAdjName(invContainer.Adj1),
			) &&
				!matchesTarget(
					name,
					containerTarget,
					e.getAdjName(invContainer.Adj2),
				) &&
				!matchesTarget(
					name,
					containerTarget,
					e.getAdjName(invContainer.Adj3),
				) {
				continue
			}

			if def.Container != "IN" &&
				def.Container != "ON" &&
				def.Container != "BEHIND" &&
				def.Container != "UNDER" &&
				def.Type != "CONTAINER" &&
				!containsFlag(def.Flags, "CONTAINER") {
				continue
			}

			if relation != "" &&
				!strings.EqualFold(def.Container, relation) {
				continue
			}

			if containerSkip > 0 {
				containerSkip--
				continue
			}

			containerName := e.formatItemName(
				def,
				invContainer.Adj1,
				invContainer.Adj2,
				invContainer.Adj3,
			)

			if invContainer.State != "OPEN" && invContainer.State != "" {
				return &CommandResult{
					Messages: []string{
						fmt.Sprintf(
							"You'll need to open %s first.",
							containerName,
						),
					},
				}
			}

			// Support:
			// GET SECOND SCROLL FROM SACK
			parsedItemTarget, ordSkip := parseOrdinal(itemTarget)
			skip := ordSkip

			for ii := range invContainer.Contents {
				child := invContainer.Contents[ii]

				childDef := e.items[child.Archetype]
				if childDef == nil {
					continue
				}

				childName := e.getItemNounName(childDef)

				if !matchesTarget(
					childName,
					parsedItemTarget,
					e.getAdjName(child.Adj1),
				) &&
					!matchesTarget(
						childName,
						parsedItemTarget,
						e.getAdjName(child.Adj2),
					) &&
					!matchesTarget(
						childName,
						parsedItemTarget,
						e.getAdjName(child.Adj3),
					) {
					continue
				}

				if skip > 0 {
					skip--
					continue
				}

				// MONEY inside a carried container.
				if childDef.Type == "MONEY" || child.State == "MONEY" {
					coins := child.Val1
					if coins <= 0 {
						coins = 1
					}

					invContainer.Contents = append(
						invContainer.Contents[:ii],
						invContainer.Contents[ii+1:]...,
					)

					player.Copper += coins

					player.Silver += player.Copper / 10
					player.Copper %= 10

					player.Gold += player.Silver / 10
					player.Silver %= 10

					e.SavePlayer(ctx, player)

					return &CommandResult{
						Messages: []string{
							fmt.Sprintf(
								"You take %d coins from %s.",
								coins,
								containerName,
							),
						},
						RoomBroadcast: []string{
							fmt.Sprintf(
								"%s takes some coins from %s.",
								player.FirstName,
								containerName,
							),
						},
					}
				}

				// Remove from container.
				invContainer.Contents = append(
					invContainer.Contents[:ii],
					invContainer.Contents[ii+1:]...,
				)

				// Add to ordinary inventory.
				//
				// child already contains its own Contents, so nested
				// containers remain intact automatically.
				player.Inventory = append(
					player.Inventory,
					child,
				)

				e.SavePlayer(ctx, player)

				fullName := e.formatItemName(
					childDef,
					child.Adj1,
					child.Adj2,
					child.Adj3,
				)

				return &CommandResult{
					Messages: []string{
						fmt.Sprintf(
							"You take %s from %s.",
							fullName,
							containerName,
						),
					},
					RoomBroadcast: []string{
						fmt.Sprintf(
							"%s takes %s from %s.",
							player.FirstName,
							fullName,
							containerName,
						),
					},
				}
			}

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"You don't see that in %s.",
						containerName,
					),
				},
			}
		}

		return &CommandResult{
			Messages: []string{"You don't see that container here."},
		}
	}

	// ------------------------------------------------------------
	// Room container found.
	// ------------------------------------------------------------

	containerName := e.formatItemName(
		containerDef,
		container.Adj1,
		container.Adj2,
		container.Adj3,
	)

	// Only IN-style containers require opening.
	if containerDef.Container == "IN" &&
		container.State != "OPEN" &&
		container.State != "" {

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You'll need to open %s first.",
					containerName,
				),
			},
		}
	}

	// Support:
	// GET SECOND SCROLL FROM BIN
	parsedItemTarget, ordSkip := parseOrdinal(itemTarget)
	skip := ordSkip

	// Find only items PUT inside this specific container.
	for _, ri := range room.Items {
		if !ri.IsPut || ri.PutIn != container.Ref {
			continue
		}

		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}

		name := e.getItemNounName(itemDef)

		if !matchesTarget(
			name,
			parsedItemTarget,
			e.getAdjName(ri.Adj1),
		) &&
			!matchesTarget(
				name,
				parsedItemTarget,
				e.getAdjName(ri.Adj2),
			) &&
			!matchesTarget(
				name,
				parsedItemTarget,
				e.getAdjName(ri.Adj3),
			) {
			continue
		}

		if skip > 0 {
			skip--
			continue
		}

		// Make a stable copy before any scripts can mutate room.Items.
		pickedUp := ri

		// ------------------------------------------------------------
		// Handle money inside room containers.
		// ------------------------------------------------------------

		if itemDef.Type == "MONEY" || pickedUp.State == "MONEY" {
			coins := pickedUp.Val1
			if coins <= 0 {
				coins = 1
			}

			removeIndex := -1

			for j := range room.Items {
				candidate := room.Items[j]

				if candidate.IsPut &&
					candidate.PutIn == pickedUp.PutIn &&
					candidate.Ref == pickedUp.Ref &&
					candidate.Archetype == pickedUp.Archetype &&
					candidate.Adj1 == pickedUp.Adj1 &&
					candidate.Adj2 == pickedUp.Adj2 &&
					candidate.Adj3 == pickedUp.Adj3 {

					removeIndex = j
					break
				}
			}

			if removeIndex >= 0 {
				room.Items = append(
					room.Items[:removeIndex],
					room.Items[removeIndex+1:]...,
				)
			}

			player.Copper += coins

			player.Silver += player.Copper / 10
			player.Copper %= 10

			player.Gold += player.Silver / 10
			player.Silver %= 10

			e.SavePlayer(ctx, player)

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"You take %d coins from %s.",
						coins,
						containerName,
					),
				},
				RoomBroadcast: []string{
					fmt.Sprintf(
						"%s takes some coins from %s.",
						player.FirstName,
						containerName,
					),
				},
			}
		}

		// ------------------------------------------------------------
		// Run normal GET preverb scripts on the item.
		// ------------------------------------------------------------

		sc := e.RunPreverbScripts(
			player,
			room,
			"GET",
			&pickedUp,
			itemDef,
		)

		result := &CommandResult{
			Messages:      append([]string{}, sc.Messages...),
			RoomBroadcast: append([]string{}, sc.RoomMsgs...),
			GMBroadcast:   append([]string{}, sc.GMMsgs...),
		}

		if sc.Blocked {
			if len(result.Messages) == 0 {
				result.Messages = []string{"You can't get that."}
			}

			e.SavePlayer(ctx, player)
			return result
		}

		// ------------------------------------------------------------
		// If the item being taken is itself a container, collect
		// anything stored inside it before removing it from the room.
		//
		// Room-contained items use:
		//     IsPut = true
		//     PutIn = parent.Ref
		//
		// Inventory containers use InventoryItem.Contents.
		// ------------------------------------------------------------

		var contents []InventoryItem

		for _, child := range room.Items {
			if !child.IsPut || child.PutIn != pickedUp.Ref {
				continue
			}

			contents = append(
				contents,
				InventoryItem{
					Archetype: child.Archetype,
					Adj1:      child.Adj1,
					Adj2:      child.Adj2,
					Adj3:      child.Adj3,
					Val1:      child.Val1,
					Val2:      child.Val2,
					Val3:      child.Val3,
					Val4:      child.Val4,
					Val5:      child.Val5,
					State:     child.State,
				},
			)
		}

		// ------------------------------------------------------------
		// Add the item to player inventory, preserving its contents.
		// ------------------------------------------------------------

		player.Inventory = append(
			player.Inventory,
			InventoryItem{
				Archetype: pickedUp.Archetype,
				Adj1:      pickedUp.Adj1,
				Adj2:      pickedUp.Adj2,
				Adj3:      pickedUp.Adj3,
				Val1:      pickedUp.Val1,
				Val2:      pickedUp.Val2,
				Val3:      pickedUp.Val3,
				Val4:      pickedUp.Val4,
				Val5:      pickedUp.Val5,
				State:     pickedUp.State,
				Contents:  contents,
			},
		)

		// ------------------------------------------------------------
		// Remove the item's children from room.Items now that they
		// live inside InventoryItem.Contents.
		// ------------------------------------------------------------

		if len(contents) > 0 {
			filtered := room.Items[:0]

			for _, item := range room.Items {
				if item.IsPut && item.PutIn == pickedUp.Ref {
					continue
				}

				filtered = append(filtered, item)
			}

			room.Items = filtered
		}

		// ------------------------------------------------------------
		// The GET script may have modified room.Items, and removing
		// children above may also have shifted indexes.
		//
		// Locate the exact PUT item again before removing it.
		// ------------------------------------------------------------

		removeIndex := -1

		for j := range room.Items {
			candidate := room.Items[j]

			if candidate.IsPut &&
				candidate.PutIn == pickedUp.PutIn &&
				candidate.Ref == pickedUp.Ref &&
				candidate.Archetype == pickedUp.Archetype &&
				candidate.Adj1 == pickedUp.Adj1 &&
				candidate.Adj2 == pickedUp.Adj2 &&
				candidate.Adj3 == pickedUp.Adj3 {

				removeIndex = j
				break
			}
		}

		if removeIndex >= 0 {
			room.Items = append(
				room.Items[:removeIndex],
				room.Items[removeIndex+1:]...,
			)
		}

		e.SavePlayer(ctx, player)

		fullName := e.formatItemName(
			itemDef,
			pickedUp.Adj1,
			pickedUp.Adj2,
			pickedUp.Adj3,
		)

		result.Messages = append(
			result.Messages,
			fmt.Sprintf(
				"You take %s from %s.",
				fullName,
				containerName,
			),
		)

		result.RoomBroadcast = append(
			result.RoomBroadcast,
			fmt.Sprintf(
				"%s takes %s from %s.",
				player.FirstName,
				fullName,
				containerName,
			),
		)

		return result
	}

	return &CommandResult{
		Messages: []string{
			fmt.Sprintf(
				"You don't see that in %s.",
				containerName,
			),
		},
	}
}

func (e *GameEngine) doPut(ctx context.Context, player *Player, args []string) *CommandResult {

	if len(args) < 3 {
		return &CommandResult{
			Messages: []string{"Put what in/on/behind what?"},
		}
	}

	raw := strings.ToLower(strings.Join(args, " "))

	// ------------------------------------------------------------
	// Parse:
	//   PUT <item> IN <container>
	//   PUT <item> ON <container>
	//   PUT <item> BEHIND <container>
	// ------------------------------------------------------------

	relation := ""
	splitIdx := -1
	splitLen := 0

	if idx := strings.Index(raw, " in "); idx >= 0 {
		relation = "IN"
		splitIdx = idx
		splitLen = len(" in ")
	} else if idx := strings.Index(raw, " on "); idx >= 0 {
		relation = "ON"
		splitIdx = idx
		splitLen = len(" on ")
	} else if idx := strings.Index(raw, " behind "); idx >= 0 {
		relation = "BEHIND"
		splitIdx = idx
		splitLen = len(" behind ")
	} else if idx := strings.Index(raw, " under "); idx >= 0 {
		relation = "UNDER"
		splitIdx = idx
		splitLen = len(" under ")
	}

	if splitIdx < 0 {
		return &CommandResult{
			Messages: []string{"Put what in/on/behind/under what?"},
		}
	}

	itemTarget := strings.TrimSpace(raw[:splitIdx])
	containerTarget := strings.TrimSpace(raw[splitIdx+splitLen:])

	if itemTarget == "" || containerTarget == "" {
		return &CommandResult{
			Messages: []string{"Put what in/on/behind/under what?"},
		}
	}

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{
			Messages: []string{"You can't do that here."},
		}
	}

	// ------------------------------------------------------------
	// Find object #2 in the room.
	//
	// Multiple room items may have the same visible name but
	// represent different relationships, e.g.:
	//
	//   rusty bed -> CONTAINER ON
	//   rusty bed -> CONTAINER BEHIND
	//
	// Prefer the matching relationship. Keep the first ordinary
	// noun match as a fallback because an IFPREVERB2 PUT script
	// may intercept an otherwise non-container object.
	// ------------------------------------------------------------

	containerIndex := -1
	fallbackIndex := -1

	for i := range room.Items {
		ri := &room.Items[i]

		if ri.IsPut {
			continue
		}

		def := e.items[ri.Archetype]
		if def == nil {
			continue
		}

		name := e.getItemNounName(def)

		if !matchesTarget(
			name,
			containerTarget,
			e.getAdjName(ri.Adj1),
		) &&
			!matchesTarget(
				name,
				containerTarget,
				e.getAdjName(ri.Adj2),
			) &&
			!matchesTarget(
				name,
				containerTarget,
				e.getAdjName(ri.Adj3),
			) {
			continue
		}

		// Remember the first visible match in case a script
		// handles PUT against a non-container object.
		if fallbackIndex < 0 {
			fallbackIndex = i
		}

		// Prefer the object whose container relationship matches
		// the preposition the player actually used.
		if strings.EqualFold(def.Container, relation) {
			containerIndex = i
			break
		}
	}

	if containerIndex < 0 {
		containerIndex = fallbackIndex
	}

	// ------------------------------------------------------------
	// No room target found.
	// Existing carried-container code handles IN.
	// ------------------------------------------------------------

	if containerIndex < 0 {
		if relation == "IN" {
			return e.doPutInInventoryContainer(
				ctx,
				player,
				itemTarget,
				containerTarget,
			)
		}

		return &CommandResult{
			Messages: []string{
				"You don't see that here.",
			},
		}
	}

	container := &room.Items[containerIndex]
	containerDef := e.items[container.Archetype]

	if containerDef == nil {
		return &CommandResult{
			Messages: []string{"You can't put anything there."},
		}
	}

	// ------------------------------------------------------------
	// Find object #1 in player inventory.
	// ------------------------------------------------------------

	itemIndex := -1

	for i := range player.Inventory {
		ii := &player.Inventory[i]

		def := e.items[ii.Archetype]
		if def == nil {
			continue
		}

		name := e.getItemNounName(def)

		if !matchesTarget(
			name,
			itemTarget,
			e.getAdjName(ii.Adj1),
		) &&
			!matchesTarget(
				name,
				itemTarget,
				e.getAdjName(ii.Adj2),
			) &&
			!matchesTarget(
				name,
				itemTarget,
				e.getAdjName(ii.Adj3),
			) {
			continue
		}

		itemIndex = i
		break
	}

	if itemIndex < 0 {
		return &CommandResult{
			Messages: []string{"You aren't carrying that."},
		}
	}

	ii := player.Inventory[itemIndex]
	itemDef := e.items[ii.Archetype]

	if itemDef == nil {
		return &CommandResult{
			Messages: []string{"You aren't carrying that."},
		}
	}

	// ------------------------------------------------------------
	// Run PUT scripts first.
	//
	// Object #1 = item being put
	// Object #2 = destination object
	// ------------------------------------------------------------

	scriptItem := gameworld.RoomItem{
		Ref:       -1,
		Archetype: ii.Archetype,
		Adj1:      ii.Adj1,
		Adj2:      ii.Adj2,
		Adj3:      ii.Adj3,
		Val1:      ii.Val1,
		Val2:      ii.Val2,
		Val3:      ii.Val3,
		Val4:      ii.Val4,
		Val5:      ii.Val5,
		State:     ii.State,
	}

	sc := e.RunPreverbScripts(
		player,
		room,
		"PUT",
		&scriptItem,
		itemDef,
		container,
	)

	result := &CommandResult{
		Messages:      append([]string{}, sc.Messages...),
		RoomBroadcast: append([]string{}, sc.RoomMsgs...),
		GMBroadcast:   append([]string{}, sc.GMMsgs...),
	}

	if sc.Blocked {
		e.SavePlayer(ctx, player)

		if len(result.Messages) == 0 {
			result.Messages = []string{"You can't do that."}
		}

		return result
	}

	// ------------------------------------------------------------
	// From here onward this must be an ordinary container action.
	// ------------------------------------------------------------

	containerMode := strings.ToUpper(containerDef.Container)

	if containerMode != "" {
		if containerMode != relation {
			containerName := e.formatItemName(
				containerDef,
				container.Adj1,
				container.Adj2,
				container.Adj3,
			)

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"You can't put anything %s %s.",
						strings.ToLower(relation),
						containerName,
					),
				},
			}
		}
	} else {
		isGenericContainer :=
			containerDef.Type == "CONTAINER" ||
				containsFlag(containerDef.Flags, "CONTAINER")

		if !isGenericContainer || relation != "IN" {
			return &CommandResult{
				Messages: []string{
					"You can't put anything there.",
				},
			}
		}
	}

	// ------------------------------------------------------------
	// IN containers must be open.
	// ------------------------------------------------------------

	if relation == "IN" &&
		container.State != "OPEN" &&
		container.State != "" {

		containerName := e.formatItemName(
			containerDef,
			container.Adj1,
			container.Adj2,
			container.Adj3,
		)

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You'll need to open %s first.",
					containerName,
				),
			},
		}
	}

	// ------------------------------------------------------------
	// Capacity / volume rules.
	// ------------------------------------------------------------

	if itemDef.Volume >= containerDef.Volume {
		return &CommandResult{
			Messages: []string{
				"That item is too large.",
			},
		}
	}

	usedVolume := 0

	for _, ri := range room.Items {
		if !ri.IsPut || ri.PutIn != container.Ref {
			continue
		}

		if def := e.items[ri.Archetype]; def != nil {
			usedVolume += def.Volume
		}
	}

	if containerDef.Interior > 0 &&
		usedVolume+itemDef.Volume > containerDef.Interior {

		switch relation {
		case "ON":
			return &CommandResult{
				Messages: []string{
					"There isn't enough room on it.",
				},
			}

		case "BEHIND":
			return &CommandResult{
				Messages: []string{
					"There isn't enough room behind it.",
				},
			}

		default:
			return &CommandResult{
				Messages: []string{
					"There isn't enough room in the container.",
				},
			}
		}
	}

	// ------------------------------------------------------------
	// Find a unique room Ref for the item being PUT.
	//
	// Ref identifies THIS item.
	// PutIn identifies its parent container.
	//
	// They must NOT be the same thing.
	// ------------------------------------------------------------

	nextRef := 0

	for _, ri := range room.Items {
		if ri.Ref >= nextRef {
			nextRef = ri.Ref + 1
		}
	}

	putItem := gameworld.RoomItem{
		Ref:       nextRef,
		Archetype: ii.Archetype,
		Adj1:      ii.Adj1,
		Adj2:      ii.Adj2,
		Adj3:      ii.Adj3,
		Val1:      ii.Val1,
		Val2:      ii.Val2,
		Val3:      ii.Val3,
		Val4:      ii.Val4,
		Val5:      ii.Val5,
		State:     ii.State,
		IsPut:     true,
		PutIn:     container.Ref,
	}

	room.Items = append(room.Items, putItem)

	e.notifyRoomChange(RoomChange{
		RoomNumber: player.RoomNumber,
		Type:       "item_add",
		Item:       &putItem,
	})

	// ------------------------------------------------------------
	// If the item itself contains anything, convert its inventory
	// Contents into room PUT items beneath the new room item.
	// ------------------------------------------------------------

	parentRef := putItem.Ref

	for _, child := range ii.Contents {
		childRef := 0

		for _, ri := range room.Items {
			if ri.Ref >= childRef {
				childRef = ri.Ref + 1
			}
		}

		childRoomItem := gameworld.RoomItem{
			Ref:       childRef,
			Archetype: child.Archetype,
			Adj1:      child.Adj1,
			Adj2:      child.Adj2,
			Adj3:      child.Adj3,
			Val1:      child.Val1,
			Val2:      child.Val2,
			Val3:      child.Val3,
			Val4:      child.Val4,
			Val5:      child.Val5,
			State:     child.State,
			IsPut:     true,
			PutIn:     parentRef,
		}

		room.Items = append(room.Items, childRoomItem)

		e.notifyRoomChange(RoomChange{
			RoomNumber: player.RoomNumber,
			Type:       "item_add",
			Item:       &childRoomItem,
		})
	}

	// ------------------------------------------------------------
	// Remove original item from player inventory.
	// ------------------------------------------------------------

	player.Inventory = append(
		player.Inventory[:itemIndex],
		player.Inventory[itemIndex+1:]...,
	)

	e.SavePlayer(ctx, player)

	itemName := e.formatItemName(
		itemDef,
		ii.Adj1,
		ii.Adj2,
		ii.Adj3,
	)

	containerName := e.formatItemName(
		containerDef,
		container.Adj1,
		container.Adj2,
		container.Adj3,
	)

	preposition := strings.ToLower(relation)

	result.Messages = append(
		result.Messages,
		fmt.Sprintf(
			"You put %s %s %s.",
			itemName,
			preposition,
			containerName,
		),
	)

	result.RoomBroadcast = append(
		result.RoomBroadcast,
		fmt.Sprintf(
			"%s puts %s %s %s.",
			player.FirstName,
			itemName,
			preposition,
			containerName,
		),
	)

	return result
}

func (e *GameEngine) doPutInInventoryContainer(ctx context.Context, player *Player, itemTarget string, containerTarget string) *CommandResult {

	containerTarget, ordSkip := parseOrdinal(containerTarget)
	skip := ordSkip

	for containerIndex := range player.Inventory {
		container := &player.Inventory[containerIndex]

		containerDef := e.items[container.Archetype]
		if containerDef == nil {
			continue
		}

		containerNoun := e.getItemNounName(containerDef)

		if !matchesTarget(
			containerNoun,
			containerTarget,
			e.getAdjName(container.Adj1),
		) &&
			!matchesTarget(
				containerNoun,
				containerTarget,
				e.getAdjName(container.Adj3),
			) {
			continue
		}

		if containerDef.Container != "IN" &&
			containerDef.Type != "CONTAINER" &&
			!containsFlag(containerDef.Flags, "CONTAINER") {
			continue
		}

		// Ordinal support:
		// PUT CLAW IN CHEST 2
		// PUT CLAW IN SECOND CHEST
		if skip > 0 {
			skip--
			continue
		}

		containerName := e.formatItemName(
			containerDef,
			container.Adj1,
			container.Adj2,
			container.Adj3,
		)

		if container.State != "OPEN" && container.State != "" {
			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"You'll need to open %s first.",
						containerName,
					),
				},
			}
		}

		// Find the item being placed into the container.
		itemTargetParsed, itemOrdSkip := parseOrdinal(itemTarget)
		itemSkip := itemOrdSkip
		itemIndex := -1

		for i := range player.Inventory {
			// Don't allow putting a container into itself.
			if i == containerIndex {
				continue
			}

			item := &player.Inventory[i]

			itemDef := e.items[item.Archetype]
			if itemDef == nil {
				continue
			}

			itemNoun := e.getItemNounName(itemDef)

			if !matchesTarget(
				itemNoun,
				itemTargetParsed,
				e.getAdjName(item.Adj1),
			) &&
				!matchesTarget(
					itemNoun,
					itemTargetParsed,
					e.getAdjName(item.Adj3),
				) {
				continue
			}

			if itemSkip > 0 {
				itemSkip--
				continue
			}

			itemIndex = i
			break
		}

		if itemIndex < 0 {
			return &CommandResult{
				Messages: []string{
					"You aren't carrying that.",
				},
			}
		}

		item := player.Inventory[itemIndex]
		itemDef := e.items[item.Archetype]

		// ---------------------------------------------------------
		// Container volume rules from original game docs.
		// ---------------------------------------------------------

		// Item itself must be smaller than the container's VOLUME.
		if itemDef.Volume >= containerDef.Volume {
			return &CommandResult{
				Messages: []string{
					"That item is too large to fit in the container.",
				},
			}
		}

		// Total contents may not exceed container INTERIOR.
		usedVolume := 0

		for _, child := range container.Contents {
			if def := e.items[child.Archetype]; def != nil {
				usedVolume += def.Volume
			}
		}

		if usedVolume+itemDef.Volume > containerDef.Interior {
			return &CommandResult{
				Messages: []string{
					"There isn't enough room in the container.",
				},
			}
		}

		/*
			Remove the item before modifying the container.

			Removing from player.Inventory can shift the container's
			index, so adjust it if necessary.
		*/
		player.Inventory = append(
			player.Inventory[:itemIndex],
			player.Inventory[itemIndex+1:]...,
		)

		if itemIndex < containerIndex {
			containerIndex--
		}

		player.Inventory[containerIndex].Contents = append(
			player.Inventory[containerIndex].Contents,
			item,
		)

		e.SavePlayer(ctx, player)

		itemName := e.formatItemName(
			itemDef,
			item.Adj1,
			item.Adj2,
			item.Adj3,
		)

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You put %s in %s.",
					itemName,
					containerName,
				),
			},
			RoomBroadcast: []string{
				fmt.Sprintf(
					"%s puts %s in %s.",
					player.FirstName,
					itemName,
					containerName,
				),
			},
		}
	}

	return &CommandResult{
		Messages: []string{
			"You don't see that container here.",
		},
	}
}

func (e *GameEngine) doDrop(
	ctx context.Context,
	player *Player,
	args []string,
) *CommandResult {

	if len(args) == 0 {
		return &CommandResult{
			Messages: []string{"Drop what?"},
		}
	}

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{
			Messages: []string{"You can't do that here."},
		}
	}

	// ------------------------------------------------------------
	// DROP MONEY
	//
	// Examples:
	//   DROP 10 GOLD
	//   DROP 5 SILVER
	//   DROP 20 COPPER
	//
	// Currency normally lives directly on the Player, rather than
	// as inventory items. Dropping it creates a MONEY RoomItem.
	// ------------------------------------------------------------
	if len(args) >= 2 {
		amount, err := strconv.Atoi(args[0])

		if err == nil && amount > 0 {
			denom := strings.ToLower(args[1])

			var archetype int
			var available *int
			var denomName string
			var adjName string

			switch denom {
			case "gold", "crown", "crowns":
				archetype = 161
				available = &player.Gold
				denomName = "gold"
				adjName = "gold"

			case "silver", "shilling", "shillings":
				archetype = 162
				available = &player.Silver
				denomName = "silver"
				adjName = "silver"

			case "copper", "penny", "pennies":
				archetype = 163
				available = &player.Copper
				denomName = "copper"
				adjName = "copper"
			}

			// Recognized a currency denomination.
			if available != nil {

				if *available < amount {
					return &CommandResult{
						Messages: []string{
							fmt.Sprintf(
								"You don't have %d %s coins.",
								amount,
								denomName,
							),
						},
					}
				}

				// Find the legacy adjective number rather than
				// hard-coding gold=147, etc.
				adj := e.adjByName(adjName)

				*available -= amount

				droppedItem := gameworld.RoomItem{
					Ref:       len(room.Items),
					Archetype: archetype,
					Adj3:      adj, //after much searching I found adj 3 is the one that controls the coin type
					Val1:      amount,
					State:     "MONEY",
				}

				room.Items = append(
					room.Items,
					droppedItem,
				)

				e.notifyRoomChange(RoomChange{
					RoomNumber: player.RoomNumber,
					Type:       "item_add",
					Item:       &droppedItem,
				})

				e.SavePlayer(ctx, player)

				msgs := []string{}

				if player.GMTrace {
					msgs = append(
						msgs,
						fmt.Sprintf(
							"[TRACE] DROP MONEY archetype=%d adj3=%d val1=%d",
							archetype,
							adj,
							amount,
						),
					)
				}

				msgs = append(
					msgs,
					fmt.Sprintf(
						"You drop %d %s coins.",
						amount,
						denomName,
					),
				)

				return &CommandResult{
					Messages: msgs,
					RoomBroadcast: []string{
						fmt.Sprintf(
							"%s drops some %s coins.",
							player.FirstName,
							denomName,
						),
					},
					PlayerState: player,
				}
			}
		}
	}

	// ------------------------------------------------------------
	// NORMAL INVENTORY DROP
	// ------------------------------------------------------------

	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}

		name := e.getItemNounName(itemDef)

		if !matchesTarget(
			name,
			target,
			e.getAdjName(ii.Adj1),
		) {
			continue
		}

		if skip > 0 {
			skip--
			continue
		}

		/*
			Run the inventory item's IFPREVERB DROP script before
			performing the normal drop.

			RunPreverbScripts expects a RoomItem, so create a
			temporary script representation of the inventory item.
		*/
		scriptItem := gameworld.RoomItem{
			Ref:       -1,
			Archetype: ii.Archetype,
			Adj1:      ii.Adj1,
			Adj2:      ii.Adj2,
			Adj3:      ii.Adj3,
			Val1:      ii.Val1,
			Val2:      ii.Val2,
			Val3:      ii.Val3,
			Val4:      ii.Val4,
			Val5:      ii.Val5,
			State:     ii.State,
		}

		sc := e.RunPreverbScripts(
			player,
			room,
			"DROP",
			&scriptItem,
			itemDef,
		)

		result := &CommandResult{
			Messages:      append([]string{}, sc.Messages...),
			RoomBroadcast: append([]string{}, sc.RoomMsgs...),
			GMBroadcast:   append([]string{}, sc.GMMsgs...),
		}

		/*
			CLEARVERB means the script handled or blocked the drop.

			For example, REMOVEITEM -1 may already have deleted
			the inventory item, so normal DROP must not continue.
		*/
		if sc.Blocked {
			if len(result.Messages) == 0 {
				result.Messages = []string{
					"You can't drop that.",
				}
			}

			e.SavePlayer(ctx, player)

			return result
		}

		// --------------------------------------------------------
		// Ordinary inventory-to-room transfer.
		// --------------------------------------------------------

		droppedItem := gameworld.RoomItem{
			Ref:       len(room.Items),
			Archetype: ii.Archetype,
			Adj1:      ii.Adj1,
			Adj2:      ii.Adj2,
			Adj3:      ii.Adj3,
			Val1:      ii.Val1,
			Val2:      ii.Val2,
			Val3:      ii.Val3,
			Val4:      ii.Val4,
			Val5:      ii.Val5,
			State:     ii.State,
		}

		room.Items = append(
			room.Items,
			droppedItem,
		)

		e.notifyRoomChange(RoomChange{
			RoomNumber: player.RoomNumber,
			Type:       "item_add",
			Item:       &droppedItem,
		})

		// --------------------------------------------------------
		// Restore carried container contents as PUT items.
		// --------------------------------------------------------

		for _, child := range ii.Contents {
			putItem := gameworld.RoomItem{
				Ref:       droppedItem.Ref,
				Archetype: child.Archetype,
				Adj1:      child.Adj1,
				Adj2:      child.Adj2,
				Adj3:      child.Adj3,
				Val1:      child.Val1,
				Val2:      child.Val2,
				Val3:      child.Val3,
				Val4:      child.Val4,
				Val5:      child.Val5,
				State:     child.State,
				IsPut:     true,
				PutIn:     droppedItem.Ref,
			}

			room.Items = append(
				room.Items,
				putItem,
			)

			e.notifyRoomChange(RoomChange{
				RoomNumber: player.RoomNumber,
				Type:       "item_add",
				Item:       &putItem,
			})
		}

		// Remove item from inventory.
		player.Inventory = append(
			player.Inventory[:i],
			player.Inventory[i+1:]...,
		)

		e.SavePlayer(ctx, player)

		fullName := e.formatItemName(
			itemDef,
			ii.Adj1,
			ii.Adj2,
			ii.Adj3,
		)

		result.Messages = append(
			result.Messages,
			fmt.Sprintf(
				"You drop %s.",
				fullName,
			),
		)

		result.RoomBroadcast = append(
			result.RoomBroadcast,
			fmt.Sprintf(
				"%s drops %s.",
				player.FirstName,
				fullName,
			),
		)

		return result
	}

	return &CommandResult{
		Messages: []string{
			"You aren't carrying that.",
		},
	}
}

func (e *GameEngine) doInventory(player *Player) *CommandResult {
	var msgs []string

	msgs = append(msgs, "You are carrying:")

	if len(player.Inventory) == 0 &&
		len(player.Worn) == 0 &&
		player.Wielded == nil &&
		player.Offhand == nil {

		msgs = append(msgs, "  Nothing.")
		return &CommandResult{Messages: msgs}
	}

	// Main hand.
	if player.Wielded != nil {
		itemDef := e.items[player.Wielded.Archetype]
		if itemDef != nil {
			name := e.formatItemName(
				itemDef,
				player.Wielded.Adj1,
				player.Wielded.Adj2,
				player.Wielded.Adj3,
			)

			msgs = append(
				msgs,
				fmt.Sprintf("  %s (wielded)", name),
			)
		}
	}

	// Off hand.
	if player.Offhand != nil {
		itemDef := e.items[player.Offhand.Archetype]
		if itemDef != nil {
			name := e.formatItemName(
				itemDef,
				player.Offhand.Adj1,
				player.Offhand.Adj2,
				player.Offhand.Adj3,
			)

			msgs = append(
				msgs,
				fmt.Sprintf("  %s (off hand)", name),
			)
		}
	}

	// Worn equipment.
	for _, ii := range player.Worn {
		itemDef := e.items[ii.Archetype]
		if itemDef != nil {
			name := e.formatItemName(
				itemDef,
				ii.Adj1,
				ii.Adj2,
				ii.Adj3,
			)

			msgs = append(
				msgs,
				fmt.Sprintf("  %s (worn)", name),
			)
		}
	}

	// Normal inventory.
	for _, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef != nil {
			name := e.formatItemName(
				itemDef,
				ii.Adj1,
				ii.Adj2,
				ii.Adj3,
			)

			msgs = append(
				msgs,
				fmt.Sprintf("  %s", name),
			)
		}
	}

	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) doStatus(player *Player) *CommandResult {
	recalcBuildPoints(player)

	var msgs []string

	// Organization line (if any)
	if player.Organization > 0 {
		orgName := organizationName(player.Organization)
		if orgName != "" {
			msgs = append(msgs, fmt.Sprintf("You are a member of the %s.", orgName))
		}
	}

	msgs = append(msgs,
		fmt.Sprintf("Name: %s   Race: %s   Gender: %s   Level: %d", player.FullName(), player.RaceName(), genderName(player.Gender), player.Level),
		//fmt.Sprintf("Strength: %d   Agility: %d   Quickness: %d", player.EffectiveStat(StatStrength), player.EffectiveStat(StatAgility), player.EffectiveStat(StatQuickness)),
		//fmt.Sprintf("Constitution: %d   Perception: %d   Willpower: %d   Empathy: %d", player.EffectiveStat(StatConstitution), player.EffectiveStat(StatPerception), player.EffectiveStat(StatWillpower), player.EffectiveStat(StatEmpathy)),
		fmt.Sprintf("Strength: %s   Agility: %s   Quickness: %s",
			formatEffectiveStat(player, StatStrength, player.Strength),
			formatEffectiveStat(player, StatAgility, player.Agility),
			formatEffectiveStat(player, StatQuickness, player.Quickness)),

		fmt.Sprintf("Constitution: %s   Perception: %s   Willpower: %s   Empathy: %s",
			formatEffectiveStat(player, StatConstitution, player.Constitution),
			formatEffectiveStat(player, StatPerception, player.Perception),
			formatEffectiveStat(player, StatWillpower, player.Willpower),
			formatEffectiveStat(player, StatEmpathy, player.Empathy)),
	)

	spentBP := playerBPSpent(player)
	unspentBP := player.BuildPoints
	totalBP := unspentBP + spentBP

	xpUntilNextBP := xpUntilNextBuildPoint(player)

	msgs = append(msgs,
		fmt.Sprintf("Build Points to date: %d", totalBP),
		fmt.Sprintf("Unspent Build Points: %d", unspentBP),
		fmt.Sprintf("Experience Points until next Build Point: %d", xpUntilNextBP),
	)

	// Attack/Defense modifiers
	var weaponDef *gameworld.ItemDef
	if player.Wielded != nil {
		weaponDef = e.items[player.Wielded.Archetype]
	}
	atkRating := playerAttackRating(player, weaponDef)
	defRating := e.playerDefenseRating(player)
	stanceLabel := stanceNames[player.Stance]

	msgs = append(msgs,
		fmt.Sprintf("Current Attack Modifier: %d [%s]", atkRating, stanceLabel),
		fmt.Sprintf("Current Defend Modifier: %d", defRating),
	)

	//Room brightness modifiers
	room := e.rooms[player.RoomNumber]

	if visStatus := e.visibilityStatus(player, room); visStatus != "" {
		msgs = append(msgs, visStatus)
	}

	// Height/Weight/Load
	heightFeet := player.Height / 12
	heightInches := player.Height % 12
	loadWeight := playerLoadWeight(player, e.items)
	msgs = append(msgs,
		fmt.Sprintf("Height: %d'%d   Weight: %d lbs", heightFeet, heightInches, player.Weight),
		fmt.Sprintf("Load: %d lbs", loadWeight),
	)

	// Active temporary effects
	now := time.Now()
	activeEffectCount := 0

	for _, effect := range player.ActiveStatEffects {
		if !effect.Permanent && !effect.ExpiresAt.After(now) {
			continue
		}

		if activeEffectCount == 0 {
			msgs = append(msgs, "Active Effects:")
		}

		name := "Unknown Effect"

		switch effect.Source {
		case EffectSourceSpell:
			if spell := FindSpellByID(effect.EffectID); spell != nil {
				name = spell.Name
			}
		case EffectSourcePotion:
			name = "Potion Effect"
		case EffectSourcePoison:
			name = "Poison"
		case EffectSourceDisease:
			name = "Disease"
		case EffectSourceItem:
			name = "Item Effect"
		case EffectSourceScript:
			name = "Scripted Effect"
		case EffectSourceEncumbrance:
			name = "Encumbered"
		}

		// Permanent effects
		if effect.Permanent {
			if effect.Modifier != 0 {
				msgs = append(
					msgs,
					fmt.Sprintf(
						"  %s: %s%d %s",
						name,
						modifierSign(effect.Modifier),
						effect.Modifier,
						statName(effect.Stat),
					),
				)
			} else {
				msgs = append(msgs, fmt.Sprintf("  %s", name))
			}

			activeEffectCount++
			continue
		}

		remaining := time.Until(effect.ExpiresAt)
		if remaining < 0 {
			remaining = 0
		}

		// Numeric stat/status effect
		if effect.Modifier != 0 {
			msgs = append(
				msgs,
				fmt.Sprintf(
					"  %s: %s%d %s — %s remaining",
					name,
					modifierSign(effect.Modifier),
					effect.Modifier,
					statName(effect.Stat),
					formatEffectDuration(remaining),
				),
			)
		} else {
			// Non-numeric status effect such as Light, Night Vision, Invisibility, etc.
			msgs = append(
				msgs,
				fmt.Sprintf(
					"  %s — %s remaining",
					name,
					formatEffectDuration(remaining),
				),
			)
		}

		activeEffectCount++
	}

	if activeEffectCount == 0 {
		msgs = append(msgs, "Active Effects: None")
	}

	return &CommandResult{Messages: msgs}
}

func modifierSign(modifier int) string {
	if modifier >= 0 {
		return "+"
	}
	return ""
}

func statName(stat StatID) string {
	switch stat {
	case StatStrength:
		return "Strength"
	case StatAgility:
		return "Agility"
	case StatQuickness:
		return "Quickness"
	case StatConstitution:
		return "Constitution"
	case StatPerception:
		return "Perception"
	case StatWillpower:
		return "Willpower"
	case StatEmpathy:
		return "Empathy"
	case HasteBuff:
		return "Haste"
	case SlowDebuff:
		return "Slow"
	case StatBodyPoint:
		return "Body Points"
	case StatFatigue:
		return "Fatigue"
	case StatMana:
		return "Mana"
	case StatPsi:
		return "Psioncic Energy"
	default:
		return "Unknown Stat"
	}
}

func formatEffectDuration(duration time.Duration) string {
	duration = duration.Round(time.Second)

	if duration >= time.Minute {
		minutes := int(duration / time.Minute)
		seconds := int(duration % time.Minute / time.Second)
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}

	return fmt.Sprintf("%ds", int(duration/time.Second))
}
func (e *GameEngine) doHealth(player *Player) *CommandResult {
	healthPct := float64(player.BodyPoints) / float64(player.MaxBodyPoints) * 100
	var healthDesc string
	switch {
	case healthPct >= 100:
		healthDesc = "You are in perfect health."
	case healthPct >= 75:
		healthDesc = "You have minor injuries."
	case healthPct >= 50:
		healthDesc = "You are moderately wounded."
	case healthPct >= 25:
		healthDesc = "You are seriously wounded."
	case healthPct > 0:
		healthDesc = "You are critically wounded!"
	default:
		healthDesc = "You are dead."
	}
	return &CommandResult{Messages: []string{
		healthDesc,
		fmt.Sprintf("Body: %d/%d   Fatigue: %d/%d", player.BodyPoints, player.MaxBodyPoints, player.Fatigue, player.MaxFatigue),
		fmt.Sprintf("Mana: %d/%d   Psi: %d/%d", player.Mana, player.MaxMana, player.Psi, player.MaxPsi),
	}}
}

func (e *GameEngine) doWield(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Wield what?"}}
	}

	if player.WolfForm {
		return &CommandResult{
			Messages: []string{"You can't wield anything while in wolf form."},
		}
	}

	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}

	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}

		name := e.getItemNounName(itemDef)

		// Match any of the item's adjective slots.
		if !matchesTarget(
			name,
			target,
			e.getAdjName(ii.Adj1),
		) &&
			!matchesTarget(
				name,
				target,
				e.getAdjName(ii.Adj2),
			) &&
			!matchesTarget(
				name,
				target,
				e.getAdjName(ii.Adj3),
			) {

			continue
		}

		if skip > 0 {
			skip--
			continue
		}

		// Wearable non-weapons still route to WEAR.
		// Shields are handled here because they occupy the offhand.
		if !isShield(itemDef) &&
			itemDef.WornSlot != "" &&
			!isWeapon(itemDef.Type) {
			return e.doWear(ctx, player, args)
		}

		// ------------------------------------------------------------
		// SHIELD -> OFFHAND
		// ------------------------------------------------------------

		if isShield(itemDef) {
			if player.Offhand != nil {
				offDef := e.items[player.Offhand.Archetype]
				offName := "something"

				if offDef != nil {
					offName = e.formatItemName(
						offDef,
						player.Offhand.Adj1,
						player.Offhand.Adj2,
						player.Offhand.Adj3,
					)
				}

				return &CommandResult{
					Messages: []string{
						fmt.Sprintf(
							"You are already wielding %s in your off hand.",
							offName,
						),
					},
				}
			}

			// Can't use a shield while wielding a two-handed weapon.
			if player.Wielded != nil {
				mainDef := e.items[player.Wielded.Archetype]

				if isTwoHandedWeapon(mainDef) {
					return &CommandResult{
						Messages: []string{
							"You can't wield a shield while using a two-handed weapon.",
						},
					}
				}
			}

			// Run IFPREVERB WIELD before the action.
			scriptItem := gameworld.RoomItem{
				Ref:       -1,
				Archetype: ii.Archetype,
				Adj1:      ii.Adj1,
				Adj2:      ii.Adj2,
				Adj3:      ii.Adj3,
				Val1:      ii.Val1,
				Val2:      ii.Val2,
				Val3:      ii.Val3,
				Val4:      ii.Val4,
				Val5:      ii.Val5,
			}

			sc := e.RunPreverbScripts(
				player,
				room,
				"WIELD",
				&scriptItem,
				itemDef,
			)

			result := &CommandResult{
				Messages:      append([]string{}, sc.Messages...),
				RoomBroadcast: append([]string{}, sc.RoomMsgs...),
				GMBroadcast:   append([]string{}, sc.GMMsgs...),
			}

			if sc.Blocked {
				if len(result.Messages) == 0 {
					result.Messages = []string{"You can't wield that."}
				}

				e.SavePlayer(ctx, player)
				return result
			}

			// Move shield from inventory to offhand.
			shield := player.Inventory[i]

			player.Inventory = append(
				player.Inventory[:i],
				player.Inventory[i+1:]...,
			)

			player.Offhand = &shield

			e.SavePlayer(ctx, player)

			fullName := e.formatItemName(
				itemDef,
				shield.Adj1,
				shield.Adj2,
				shield.Adj3,
			)

			// Run IFVERB WIELD after the successful action.
			verbSC := e.RunVerbScripts(
				player,
				room,
				"WIELD",
				&scriptItem,
				itemDef,
			)

			// Script text overrides the built-in player message.
			if len(verbSC.Messages) > 0 {
				result.Messages = append(
					result.Messages,
					verbSC.Messages...,
				)
			} else {
				result.Messages = append(
					result.Messages,
					fmt.Sprintf(
						"You wield %s in your off hand.",
						fullName,
					),
				)
			}

			// Script room text overrides the built-in room message.
			if len(verbSC.RoomMsgs) > 0 {
				result.RoomBroadcast = append(
					result.RoomBroadcast,
					verbSC.RoomMsgs...,
				)
			} else {
				result.RoomBroadcast = append(
					result.RoomBroadcast,
					fmt.Sprintf(
						"%s wields %s.",
						player.FirstName,
						fullName,
					),
				)
			}

			result.GMBroadcast = append(
				result.GMBroadcast,
				verbSC.GMMsgs...,
			)

			return result
		}

		// ------------------------------------------------------------
		// NORMAL WEAPON
		// ------------------------------------------------------------

		if !isWeapon(itemDef.Type) {
			return &CommandResult{
				Messages: []string{"You can't wield that."},
			}
		}

		// Two-handed weapons require an empty offhand.
		if isTwoHandedWeapon(itemDef) && player.Offhand != nil {
			return &CommandResult{
				Messages: []string{
					"You need both hands free to wield that.",
				},
			}
		}

		// Run IFPREVERB WIELD before the action.
		scriptItem := gameworld.RoomItem{
			Ref:       -1,
			Archetype: ii.Archetype,
			Adj1:      ii.Adj1,
			Adj2:      ii.Adj2,
			Adj3:      ii.Adj3,
			Val1:      ii.Val1,
			Val2:      ii.Val2,
			Val3:      ii.Val3,
			Val4:      ii.Val4,
			Val5:      ii.Val5,
		}

		sc := e.RunPreverbScripts(
			player,
			room,
			"WIELD",
			&scriptItem,
			itemDef,
		)

		result := &CommandResult{
			Messages:      append([]string{}, sc.Messages...),
			RoomBroadcast: append([]string{}, sc.RoomMsgs...),
			GMBroadcast:   append([]string{}, sc.GMMsgs...),
		}

		if sc.Blocked {
			if len(result.Messages) == 0 {
				result.Messages = []string{"You can't wield that."}
			}

			e.SavePlayer(ctx, player)
			return result
		}

		// Put currently wielded weapon back into inventory.
		if player.Wielded != nil {
			player.Inventory = append(
				player.Inventory,
				*player.Wielded,
			)
		}

		// Move selected weapon into main hand.
		wielded := player.Inventory[i]

		player.Inventory = append(
			player.Inventory[:i],
			player.Inventory[i+1:]...,
		)

		player.Wielded = &wielded

		e.SavePlayer(ctx, player)

		fullName := e.formatItemName(
			itemDef,
			wielded.Adj1,
			wielded.Adj2,
			wielded.Adj3,
		)

		// Run IFVERB WIELD after the successful action.
		verbSC := e.RunVerbScripts(
			player,
			room,
			"WIELD",
			&scriptItem,
			itemDef,
		)

		// Script text overrides the built-in player message.
		if len(verbSC.Messages) > 0 {
			result.Messages = append(
				result.Messages,
				verbSC.Messages...,
			)
		} else {
			result.Messages = append(
				result.Messages,
				fmt.Sprintf("You wield %s.", fullName),
			)
		}

		// Script room text overrides the built-in room message.
		if len(verbSC.RoomMsgs) > 0 {
			result.RoomBroadcast = append(
				result.RoomBroadcast,
				verbSC.RoomMsgs...,
			)
		} else {
			result.RoomBroadcast = append(
				result.RoomBroadcast,
				fmt.Sprintf(
					"%s wields %s.",
					player.FirstName,
					fullName,
				),
			)
		}

		result.GMBroadcast = append(
			result.GMBroadcast,
			verbSC.GMMsgs...,
		)

		return result
	}

	return &CommandResult{
		Messages: []string{"You don't have that."},
	}
}

func (e *GameEngine) doUnwield(ctx context.Context, player *Player, args []string) *CommandResult {

	if player.WolfForm {
		return &CommandResult{
			Messages: []string{"You can't do that while in wolf form."},
		}
	}

	if player.Wielded == nil && player.Offhand == nil {
		return &CommandResult{
			Messages: []string{"You aren't wielding anything."},
		}
	}

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{
			Messages: []string{"You can't do that here."},
		}
	}

	var item *InventoryItem
	var itemDef *gameworld.ItemDef
	isOffhand := false

	// ------------------------------------------------------------
	// Determine which wielded item to remove.
	// ------------------------------------------------------------

	if len(args) > 0 {
		target := strings.ToLower(strings.Join(args, " "))
		target, _ = parseOrdinal(target)

		// Check main hand.
		if player.Wielded != nil {
			def := e.items[player.Wielded.Archetype]

			if def != nil {
				name := e.getItemNounName(def)

				if matchesTarget(
					name,
					target,
					e.getAdjName(player.Wielded.Adj1),
				) ||
					matchesTarget(
						name,
						target,
						e.getAdjName(player.Wielded.Adj2),
					) ||
					matchesTarget(
						name,
						target,
						e.getAdjName(player.Wielded.Adj3),
					) {

					item = player.Wielded
					itemDef = def
				}
			}
		}

		// If it wasn't the main hand, check offhand.
		if item == nil && player.Offhand != nil {
			def := e.items[player.Offhand.Archetype]

			if def != nil {
				name := e.getItemNounName(def)

				if matchesTarget(
					name,
					target,
					e.getAdjName(player.Offhand.Adj1),
				) ||
					matchesTarget(
						name,
						target,
						e.getAdjName(player.Offhand.Adj2),
					) ||
					matchesTarget(
						name,
						target,
						e.getAdjName(player.Offhand.Adj3),
					) {

					item = player.Offhand
					itemDef = def
					isOffhand = true
				}
			}
		}

		if item == nil {
			return &CommandResult{
				Messages: []string{"You aren't wielding that."},
			}
		}
	} else {
		// No target specified:
		// main hand first, then offhand.
		if player.Wielded != nil {
			item = player.Wielded
			itemDef = e.items[item.Archetype]
		} else {
			item = player.Offhand
			itemDef = e.items[item.Archetype]
			isOffhand = true
		}
	}

	// ------------------------------------------------------------
	// Run IFPREVERB UNWIELD before the action.
	// ------------------------------------------------------------

	scriptItem := gameworld.RoomItem{
		Ref:       -1,
		Archetype: item.Archetype,
		Adj1:      item.Adj1,
		Adj2:      item.Adj2,
		Adj3:      item.Adj3,
		Val1:      item.Val1,
		Val2:      item.Val2,
		Val3:      item.Val3,
		Val4:      item.Val4,
		Val5:      item.Val5,
	}

	sc := e.RunPreverbScripts(
		player,
		room,
		"UNWIELD",
		&scriptItem,
		itemDef,
	)

	result := &CommandResult{
		Messages:      append([]string{}, sc.Messages...),
		RoomBroadcast: append([]string{}, sc.RoomMsgs...),
		GMBroadcast:   append([]string{}, sc.GMMsgs...),
	}

	if sc.Blocked {
		if len(result.Messages) == 0 {
			result.Messages = []string{"You can't put that away."}
		}

		e.SavePlayer(ctx, player)
		return result
	}

	// ------------------------------------------------------------
	// Format the item name before clearing the slot.
	// ------------------------------------------------------------

	itemName := "your weapon"

	if itemDef != nil {
		itemName = e.formatItemName(
			itemDef,
			item.Adj1,
			item.Adj2,
			item.Adj3,
		)
	}

	// ------------------------------------------------------------
	// Move the item back to inventory.
	// ------------------------------------------------------------

	player.Inventory = append(
		player.Inventory,
		*item,
	)

	if isOffhand {
		player.Offhand = nil
	} else {
		player.Wielded = nil
	}

	e.SavePlayer(ctx, player)

	// ------------------------------------------------------------
	// Run IFVERB UNWIELD after the successful action.
	// ------------------------------------------------------------

	verbSC := e.RunVerbScripts(
		player,
		room,
		"UNWIELD",
		&scriptItem,
		itemDef,
	)

	// Script text overrides the built-in player message.
	if len(verbSC.Messages) > 0 {
		result.Messages = append(
			result.Messages,
			verbSC.Messages...,
		)
	} else {
		result.Messages = append(
			result.Messages,
			fmt.Sprintf("You put away %s.", itemName),
		)
	}

	// Script room text overrides the built-in room message.
	if len(verbSC.RoomMsgs) > 0 {
		result.RoomBroadcast = append(
			result.RoomBroadcast,
			verbSC.RoomMsgs...,
		)
	} else {
		result.RoomBroadcast = append(
			result.RoomBroadcast,
			fmt.Sprintf(
				"%s puts away %s.",
				player.FirstName,
				itemName,
			),
		)
	}

	result.GMBroadcast = append(
		result.GMBroadcast,
		verbSC.GMMsgs...,
	)

	return result
}

func isShield(def *gameworld.ItemDef) bool {
	return def != nil && def.Type == "SHIELD"
}

func isTwoHandedWeapon(def *gameworld.ItemDef) bool {
	if def == nil {
		return false
	}

	switch def.Type {
	case "BOW_WEAPON",
		"POLE_WEAPON",
		"POLETHROWN",
		"TWOHAND_WEAPON",
		"DRAKIN_POLE":
		return true
	}

	return false
}

func (e *GameEngine) doWear(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Wear what?"}}
	}

	if player.WolfForm {
		return &CommandResult{
			Messages: []string{"You can't wear anything while in wolf form."},
		}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}

	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}

		if itemDef.WornSlot == "" {
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

		// Run the inventory item's IFPREVERB WEAR script before
		// performing the normal wear action.
		scriptItem := gameworld.RoomItem{
			Ref:       -1,
			Archetype: ii.Archetype,
			Adj1:      ii.Adj1,
			Adj2:      ii.Adj2,
			Adj3:      ii.Adj3,
			Val1:      ii.Val1,
			Val2:      ii.Val2,
			Val3:      ii.Val3,
			Val4:      ii.Val4,
			Val5:      ii.Val5,
		}

		sc := e.RunPreverbScripts(
			player,
			room,
			"WEAR",
			&scriptItem,
			itemDef,
		)

		result := &CommandResult{
			Messages:      append([]string{}, sc.Messages...),
			RoomBroadcast: append([]string{}, sc.RoomMsgs...),
			GMBroadcast:   append([]string{}, sc.GMMsgs...),
		}

		// CLEARVERB cancels the normal wear action.
		if sc.Blocked {
			if len(result.Messages) == 0 {
				result.Messages = []string{"You can't wear that."}
			}

			e.SavePlayer(ctx, player)
			return result
		}

		for _, wi := range player.Worn {
			if wi.WornSlot == itemDef.WornSlot {
				wornDef := e.items[wi.Archetype]

				wornName := "something"
				if wornDef != nil {
					wornName = e.formatItemName(
						wornDef,
						wi.Adj1,
						wi.Adj2,
						wi.Adj3,
					)
				}

				return &CommandResult{
					Messages: []string{
						fmt.Sprintf(
							"You are already wearing %s there.",
							wornName,
						),
					},
				}
			}
		}
		worn := player.Inventory[i]
		worn.WornSlot = itemDef.WornSlot

		player.Inventory = append(
			player.Inventory[:i],
			player.Inventory[i+1:]...,
		)

		player.Worn = append(player.Worn, worn)

		e.SavePlayer(ctx, player)

		fullName := e.formatItemName(
			itemDef,
			ii.Adj1,
			ii.Adj2,
			ii.Adj3,
		)

		result.Messages = append(
			result.Messages,
			fmt.Sprintf("You put on %s.", fullName),
		)

		result.RoomBroadcast = append(
			result.RoomBroadcast,
			fmt.Sprintf("%s puts on %s.", player.FirstName, fullName),
		)

		return result
	}

	return &CommandResult{
		Messages: []string{"You don't have that."},
	}
}

func (e *GameEngine) doRemove(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Remove what?"}}
	}

	if player.WolfForm {
		return &CommandResult{
			Messages: []string{"You can't do that while in wolf form."},
		}
	}

	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}

	for i, ii := range player.Worn {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
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

		// Run the worn item's IFPREVERB REMOVE script before
		// performing the normal remove action.
		scriptItem := gameworld.RoomItem{
			Ref:       -1,
			Archetype: ii.Archetype,
			Adj1:      ii.Adj1,
			Adj2:      ii.Adj2,
			Adj3:      ii.Adj3,
			Val1:      ii.Val1,
			Val2:      ii.Val2,
			Val3:      ii.Val3,
			Val4:      ii.Val4,
			Val5:      ii.Val5,
		}

		sc := e.RunPreverbScripts(
			player,
			room,
			"REMOVE",
			&scriptItem,
			itemDef,
		)

		result := &CommandResult{
			Messages:      append([]string{}, sc.Messages...),
			RoomBroadcast: append([]string{}, sc.RoomMsgs...),
			GMBroadcast:   append([]string{}, sc.GMMsgs...),
		}

		// CLEARVERB cancels the normal remove action.
		if sc.Blocked {
			if len(result.Messages) == 0 {
				result.Messages = []string{"You can't remove that."}
			}

			e.SavePlayer(ctx, player)
			return result
		}

		removed := player.Worn[i]
		removed.WornSlot = ""

		player.Worn = append(
			player.Worn[:i],
			player.Worn[i+1:]...,
		)

		player.Inventory = append(player.Inventory, removed)

		e.SavePlayer(ctx, player)

		fullName := e.formatItemName(
			itemDef,
			ii.Adj1,
			ii.Adj2,
			ii.Adj3,
		)

		result.Messages = append(
			result.Messages,
			fmt.Sprintf("You remove %s.", fullName),
		)

		result.RoomBroadcast = append(
			result.RoomBroadcast,
			fmt.Sprintf("%s removes %s.", player.FirstName, fullName),
		)

		return result
	}

	return &CommandResult{
		Messages: []string{"You aren't wearing that."},
	}
}

func (e *GameEngine) doOpen(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Open what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			if skip > 0 {
				skip--
				continue
			}
			if !containsFlag(itemDef.Flags, "OPENABLE") && !isPortal(itemDef.Type) {
				return &CommandResult{Messages: []string{"You can't open that."}}
			}
			if ri.State == "LOCKED" {
				return &CommandResult{Messages: []string{"It's locked."}}
			}
			if ri.State == "LATCHED" {
				return &CommandResult{Messages: []string{"It's latched shut."}}
			}
			// Check for traps (VAL4 on item)
			trapMsgs := e.checkTrap(player, &room.Items[i])

			room.Items[i].State = "OPEN"
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "OPEN"})
			fullName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
			msgs := []string{fmt.Sprintf("You open %s.", fullName)}
			if len(trapMsgs) > 0 {
				msgs = append(msgs, trapMsgs...)
			}
			return &CommandResult{Messages: msgs}
		}
	}
	// Check inventory containers
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			if skip > 0 {
				skip--
				continue
			}
			if !containsFlag(itemDef.Flags, "OPENABLE") {
				return &CommandResult{Messages: []string{"You can't open that."}}
			}

			if ii.State == "LOCKED" {
				return &CommandResult{
					Messages: []string{"It's locked."},
				}
			}

			if ii.State == "LATCHED" {
				return &CommandResult{
					Messages: []string{"It's latched shut."},
				}
			}

			player.Inventory[i].State = "OPEN"
			fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			return &CommandResult{Messages: []string{fmt.Sprintf("You open %s.", fullName)}}
		}
	}
	return &CommandResult{Messages: []string{"You don't see that here."}}
}

// checkTrap checks if an item has a trap (VAL4) and triggers it. Returns messages.
func (e *GameEngine) checkTrap(player *Player, ri *gameworld.RoomItem) []string {
	if ri.Val4 == 0 {
		return nil
	}
	trapType := ri.Val4
	ri.Val4 = 0 // trap is consumed

	var msgs []string
	switch {
	case trapType == 1: // Needle, minor poison
		msgs = append(msgs, "A needle springs out and pricks your finger!")
		player.Poisoned = true
		player.ApplyStatEffect(ri.Val4, EffectSourcePoison, StatBodyPoint, PoisonMinor, time.Duration(PoisonMinor)*time.Minute)
	case trapType == 2: // Gas, minor poison
		msgs = append(msgs, "A cloud of noxious gas billows out!")
		player.Poisoned = true
		player.ApplyStatEffect(ri.Val4, EffectSourcePoison, StatBodyPoint, PoisonNerveGas, time.Duration(PoisonNerveGas)*time.Minute)
	case trapType == 3: // Acid
		dmg := 10 + rand.Intn(15)
		player.BodyPoints -= dmg
		if player.BodyPoints < 0 {
			player.BodyPoints = 0
		}
		msgs = append(msgs, fmt.Sprintf("Acid sprays out! [%d Damage]", dmg))
	case trapType == 4: // Blades
		dmg := 15 + rand.Intn(20)
		player.BodyPoints -= dmg
		if player.BodyPoints < 0 {
			player.BodyPoints = 0
		}
		msgs = append(msgs, fmt.Sprintf("Hidden blades slash at you! [%d Damage]", dmg))
	case trapType == 5: // Needle, moderate poison
		msgs = append(msgs, "A poison-coated needle jabs into your hand!")
		player.Poisoned = true
		player.ApplyStatEffect(ri.Val4, EffectSourcePoison, StatBodyPoint, PoisonModerate, time.Duration(PoisonModerate)*time.Minute)
	case trapType == 7: // Needle, major poison
		msgs = append(msgs, "A large needle drives deep into your finger, delivering a potent venom!")
		player.Poisoned = true
		player.ApplyStatEffect(ri.Val4, EffectSourcePoison, StatBodyPoint, PoisonMajor, time.Duration(PoisonMajor)*time.Minute)
	case trapType == 8: // Explosive
		dmg := 30 + rand.Intn(30)
		player.BodyPoints -= dmg
		if player.BodyPoints < 0 {
			player.BodyPoints = 0
		}
		msgs = append(msgs, fmt.Sprintf("The container explodes! [%d Damage]", dmg))
	case trapType == 9: // Acid, moderate
		dmg := 20 + rand.Intn(25)
		player.BodyPoints -= dmg
		if player.BodyPoints < 0 {
			player.BodyPoints = 0
		}
		msgs = append(msgs, fmt.Sprintf("A gout of acid sprays out! [%d Damage]", dmg))
	case trapType == 12: // Gas, moderate poison
		msgs = append(msgs, "A thick cloud of poisonous gas engulfs you!")
		player.Poisoned = true
		player.ApplyStatEffect(ri.Val4, EffectSourcePoison, StatBodyPoint, PoisonModerate, time.Duration(PoisonModerate)*time.Minute)
	case trapType == 13: // Black needle, lethal
		dmg := 40 + rand.Intn(30)
		player.BodyPoints -= dmg
		if player.BodyPoints < 0 {
			player.BodyPoints = 0
		}
		msgs = append(msgs, fmt.Sprintf("A black needle strikes you, delivering a lethal toxin! [%d Damage]", dmg))
		player.Poisoned = true
		player.ApplyStatEffect(ri.Val4, EffectSourcePoison, StatBodyPoint, PoisonLethal, time.Duration(PoisonLethal)*time.Minute)
	case trapType >= 1000: // Glyph traps (spell-based)
		spellDmg := 20 + rand.Intn(40)
		player.BodyPoints -= spellDmg
		if player.BodyPoints < 0 {
			player.BodyPoints = 0
		}
		glyphType := (trapType / 1000) % 10
		switch {
		case glyphType <= 2:
			msgs = append(msgs, fmt.Sprintf("An Inferno Glyph erupts in a blast of flame! [%d Damage]", spellDmg))
		case glyphType <= 4:
			msgs = append(msgs, fmt.Sprintf("An Ice Glyph detonates in a burst of freezing cold! [%d Damage]", spellDmg))
		case glyphType <= 6:
			msgs = append(msgs, fmt.Sprintf("A Thunder Glyph explodes with crackling energy! [%d Damage]", spellDmg))
		case glyphType <= 8:
			msgs = append(msgs, fmt.Sprintf("An Imprisonment Rune flares! You feel rooted to the spot!"))
			player.Immobilized = true
		default:
			msgs = append(msgs, fmt.Sprintf("A Symbol of Death erupts! [%d Damage]", spellDmg))
		}
	}
	return msgs
}

func (e *GameEngine) doClose(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Close what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			if skip > 0 {
				skip--
				continue
			}
			room.Items[i].State = "CLOSED"
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "CLOSED"})
			fullName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
			return &CommandResult{Messages: []string{fmt.Sprintf("You close %s.", fullName)}}
		}
	}
	// Check inventory containers
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			if skip > 0 {
				skip--
				continue
			}
			player.Inventory[i].State = "CLOSED"
			fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			return &CommandResult{Messages: []string{fmt.Sprintf("You close %s.", fullName)}}
		}
	}
	return &CommandResult{Messages: []string{"You don't see that here."}}
}

func (e *GameEngine) doWhisper(player *Player, args []string, rawInput string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Whisper to whom?"}}
	}
	targetName := strings.ToLower(args[0])

	// Proximity whisper: "whisper close ..." or "whisper those ..."
	if targetName == "close" || targetName == "those" {
		text := extractRawArgs(rawInput, 2)
		if text == "" {
			return &CommandResult{Messages: []string{"Whisper what?"}}
		}
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You whisper to those close, \"%s\"", text)},
			RoomBroadcast: []string{fmt.Sprintf("%s whispers to those close, \"%s\"", player.FirstName, text)},
		}
	}

	found := e.findPlayerInRoom(player, targetName)
	if found == nil {
		return &CommandResult{Messages: []string{"You don't see that person here."}}
	}
	// Get the whisper text (everything after the target name)
	text := extractRawArgs(rawInput, 2)
	if text == "" {
		return &CommandResult{Messages: []string{"Whisper what?"}}
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You whisper to %s.", found.FirstName)},
		RoomBroadcast: []string{fmt.Sprintf("%s whispers to %s.", player.FirstName, found.FirstName)},
		TargetName:    found.FirstName, // exclude target from room broadcast — they get WhisperMsg instead
		WhisperTarget: found.FirstName,
		WhisperMsg:    fmt.Sprintf("%s whispers to you, \"%s\"", player.FirstName, text),
	}
}

func (e *GameEngine) doYell(player *Player, args []string, rawInput string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Yell what?"}}
	}
	text := extractRawArgs(rawInput, 1)
	adverb := ""
	if player.SpeechAdverb != "" {
		adverb = player.SpeechAdverb + " "
	}

	// Yell is heard in adjacent rooms too
	room := e.rooms[player.RoomNumber]
	if room != nil && e.roomBroadcast != nil {
		adjacentMsg := fmt.Sprintf("You hear someone yell, \"%s\"", text)
		for _, destNum := range room.Exits {
			if destNum > 0 && destNum != player.RoomNumber {
				e.roomBroadcast(destNum, []string{adjacentMsg})
			}
		}
	}

	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You %syell, \"%s\"", adverb, text)},
		RoomBroadcast: []string{fmt.Sprintf("%s %syells, \"%s\"", player.FirstName, adverb, text)},
	}
}

// doPray handles the PRAY command — triggers IFVERB PRAY scripts or generic prayer.
func (e *GameEngine) doPray(player *Player) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room != nil {
		sc := &ScriptContext{Player: player, Room: room, Engine: e}
		for _, block := range room.Scripts {
			if block.Type == "IFVERB" && len(block.Args) >= 2 {
				if strings.ToUpper(block.Args[0]) == "PRAY" && block.Args[1] == "-1" {
					sc.execBlock(block)
				}
			}
		}
		if len(sc.Messages) > 0 {
			result := &CommandResult{Messages: sc.Messages}
			if len(sc.RoomMsgs) > 0 {
				result.RoomBroadcast = sc.RoomMsgs
			}
			return result
		}
	}
	pronoun := "his"
	if player.Gender == GenderFemale {
		pronoun = "her"
	}
	return &CommandResult{
		Messages:      []string{"You pray."},
		RoomBroadcast: []string{fmt.Sprintf("%s bows %s head and prays.", player.FirstName, pronoun)},
	}
}

// doContact handles the CONTACT command — psionic telepathic whisper.
func (e *GameEngine) doContact(player *Player, args []string, rawInput string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Contact whom with what message?"}}
	}
	// CONTACT requires psionic ability (Psionics skill 26 or any psionic school skill)
	if player.Skills[26] < 1 && player.Skills[27] < 1 && player.Skills[28] < 1 && player.Skills[29] < 1 {
		return &CommandResult{Messages: []string{"You do not possess psionic abilities."}}
	}
	targetName := strings.ToLower(args[0])
	// Find the target among all online players (not just same room)
	var found *Player
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.FirstName == player.FirstName && p.LastName == player.LastName {
				continue
			}
			if strings.HasPrefix(strings.ToLower(p.FirstName), targetName) {
				found = p
				break
			}
		}
	}
	if found == nil {
		return &CommandResult{Messages: []string{"You cannot sense that person."}}
	}
	text := extractRawArgs(rawInput, 2)
	if text == "" {
		return &CommandResult{Messages: []string{"Contact whom with what message?"}}
	}
	player.RoundTimeExpiry = time.Now().Add(2 * time.Second)
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You contact %s with your thoughts.", found.FirstName), "[Round: 2 sec]"},
		WhisperTarget: found.FirstName,
		WhisperMsg:    fmt.Sprintf("You feel the touch of %s's mind: \"%s\"", player.FirstName, text),
	}
}

// doGuard handles the GUARD command — protect another player.
func (e *GameEngine) doGuard(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		// Stop guarding
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

// doChant handles the CHANT command — scroll activation.
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

		// Consume the scroll
		player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)

		// Prepare the spell — match the field name used by doPrepareSpell
		player.PreparedSpell = spellNum // ← adjust field name to match your Player struct

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

// doFollow handles the FOLLOW command — join a group.
func (e *GameEngine) doFollow(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Follow whom?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	found := e.findPlayerInRoom(player, target)
	if found == nil {
		return &CommandResult{Messages: []string{"They are not here."}}
	}
	if player.Following != "" {
		return &CommandResult{Messages: []string{fmt.Sprintf("You are already following %s.", player.Following)}}
	}
	if found.Following == player.FirstName {
		return &CommandResult{Messages: []string{"You can't follow someone who is following you."}}
	}
	player.Following = found.FirstName
	found.IsGroupLeader = true
	// Add to leader's group members (avoid duplicates)
	alreadyIn := false
	for _, m := range found.GroupMembers {
		if m == player.FirstName {
			alreadyIn = true
			break
		}
	}
	if !alreadyIn {
		found.GroupMembers = append(found.GroupMembers, player.FirstName)
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You are now following %s.", found.FirstName)},
		RoomBroadcast: []string{fmt.Sprintf("%s is now following %s.", player.FirstName, found.FirstName)},
	}
}

// doHold handles the HOLD command (group) — leader adds a member.
func (e *GameEngine) doHold(player *Player, found *Player) *CommandResult {
	if found.Following != "" {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s is already following someone.", found.FirstName)}}
	}
	found.Following = player.FirstName
	player.IsGroupLeader = true
	alreadyIn := false
	for _, m := range player.GroupMembers {
		if m == found.FirstName {
			alreadyIn = true
			break
		}
	}
	if !alreadyIn {
		player.GroupMembers = append(player.GroupMembers, found.FirstName)
	}
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("%s takes %s by the hand.", player.FirstName, found.FirstName)},
		RoomBroadcast: []string{fmt.Sprintf("%s takes %s by the hand.", player.FirstName, found.FirstName)},
	}
}

// removeFromGroup silently removes a player from their leader's group.
// moveGroupToRoom moves all online players in srcRoom to destRoom (for MOVEGROUP script command).
func (e *GameEngine) moveGroupToRoom(ctx context.Context, srcRoom, destRoom int) {
	dest := e.rooms[destRoom]
	if dest == nil || e.sessions == nil {
		return
	}
	for _, p := range e.sessions.OnlinePlayers() {
		if p.RoomNumber == srcRoom && !p.Dead {
			p.RoomNumber = destRoom
			p.Submitting = false
			e.disengageCombat(p)
			e.SavePlayer(ctx, p)
			//	if e.sendToPlayer != nil {
			//		lookResult := e.doLook(p)
			//		e.sendToPlayer(p.FirstName, lookResult.Messages)
			//	}
			e.applyEntryScripts(ctx, p, dest, &CommandResult{})
		}
	}
}

func (e *GameEngine) removeFromGroup(player *Player) {
	if player.Following == "" {
		return
	}
	leaderName := player.Following
	player.Following = ""
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.FirstName == leaderName {
				for i, m := range p.GroupMembers {
					if m == player.FirstName {
						p.GroupMembers = append(p.GroupMembers[:i], p.GroupMembers[i+1:]...)
						break
					}
				}
				if len(p.GroupMembers) == 0 {
					p.IsGroupLeader = false
				}
				break
			}
		}
	}
}

// doLeave handles the LEAVE command — stop following.
func (e *GameEngine) doLeave(player *Player) *CommandResult {
	if player.Following == "" {
		return &CommandResult{Messages: []string{"You are not following anyone."}}
	}
	leaderName := player.Following
	e.removeFromGroup(player)
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You stop following %s.", leaderName)},
		RoomBroadcast: []string{fmt.Sprintf("%s stops following %s.", player.FirstName, leaderName)},
	}
}

// doDisband handles the DISBAND command — leader disbands their group.
func (e *GameEngine) doDisband(player *Player) *CommandResult {
	if !player.IsGroupLeader || len(player.GroupMembers) == 0 {
		return &CommandResult{Messages: []string{"You don't have a group to disband."}}
	}
	// Clear Following on all members
	if e.sessions != nil {
		for _, memberName := range player.GroupMembers {
			for _, p := range e.sessions.OnlinePlayers() {
				if p.FirstName == memberName {
					p.Following = ""
					if e.sendToPlayer != nil {
						e.sendToPlayer(p.FirstName, []string{fmt.Sprintf("%s disbands the group.", player.FirstName)})
					}
					break
				}
			}
		}
	}
	player.GroupMembers = nil
	player.IsGroupLeader = false
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You disband your group.")},
		RoomBroadcast: []string{fmt.Sprintf("%s disbands their group.", player.FirstName)},
	}
}

func (e *GameEngine) doGive(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{
			Messages: []string{"Give what to whom? (give <item> to <player>)"},
		}
	}

	// Parse: give <item> to <target>
	toIdx := -1
	for i, a := range args {
		if strings.ToUpper(a) == "TO" {
			toIdx = i
			break
		}
	}

	var itemName, targetName string
	if toIdx > 0 && toIdx < len(args)-1 {
		itemName = strings.ToLower(strings.Join(args[:toIdx], " "))
		targetName = strings.ToLower(strings.Join(args[toIdx+1:], " "))
	} else {
		return &CommandResult{
			Messages: []string{"Give what to whom? (give <item> to <player>)"},
		}
	}

	// Money giving does not run item scripts.
	if amount, currency, ok := parseMoneyAmount(itemName); ok {
		target := e.findPlayerInRoom(player, targetName)
		if target == nil {
			return &CommandResult{
				Messages: []string{"You don't see that person here."},
			}
		}

		return e.doGiveMoney(ctx, player, target, amount, currency)
	}

	itemName, ordSkip := parseOrdinal(itemName)
	skip := ordSkip

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{
			Messages: []string{"You can't do that here."},
		}
	}

	// Find the item in inventory.
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}

		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, itemName, e.getAdjName(ii.Adj1)) {
			continue
		}

		if skip > 0 {
			skip--
			continue
		}

		// Find the target before running the item script.
		target := e.findPlayerInRoom(player, targetName)
		if target == nil {
			return &CommandResult{
				Messages: []string{"You don't see that person here."},
			}
		}

		// Run the inventory item's IFPREVERB GIVE script before
		// performing the normal transfer.
		scriptItem := gameworld.RoomItem{
			Ref:       -1,
			Archetype: ii.Archetype,
			Adj1:      ii.Adj1,
			Adj2:      ii.Adj2,
			Adj3:      ii.Adj3,
			Val1:      ii.Val1,
			Val2:      ii.Val2,
			Val3:      ii.Val3,
			Val4:      ii.Val4,
			Val5:      ii.Val5,
		}

		sc := e.RunPreverbScripts(
			player,
			room,
			"GIVE",
			&scriptItem,
			itemDef,
		)

		result := &CommandResult{
			Messages:      append([]string{}, sc.Messages...),
			RoomBroadcast: append([]string{}, sc.RoomMsgs...),
			GMBroadcast:   append([]string{}, sc.GMMsgs...),
		}

		// CLEARVERB cancels the normal GIVE action.
		if sc.Blocked {
			if len(result.Messages) == 0 {
				result.Messages = []string{"You can't give that away."}
			}

			e.SavePlayer(ctx, player)
			return result
		}

		fullName := e.formatItemName(
			itemDef,
			ii.Adj1,
			ii.Adj2,
			ii.Adj3,
		)

		// Transfer the item.
		target.Inventory = append(target.Inventory, ii)

		player.Inventory = append(
			player.Inventory[:i],
			player.Inventory[i+1:]...,
		)

		e.SavePlayer(ctx, player)
		e.SavePlayer(ctx, target)

		result.Messages = append(
			result.Messages,
			fmt.Sprintf(
				"You give %s to %s.",
				fullName,
				target.FirstName,
			),
		)

		result.RoomBroadcast = append(
			result.RoomBroadcast,
			fmt.Sprintf(
				"%s gives %s to %s.",
				player.FirstName,
				fullName,
				target.FirstName,
			),
		)

		result.TargetName = target.FirstName
		result.TargetMsg = append(
			result.TargetMsg,
			fmt.Sprintf(
				"%s gives you %s.",
				player.FirstName,
				fullName,
			),
		)

		return result
	}

	return &CommandResult{
		Messages: []string{"You don't have that."},
	}
}

// parseMoneyAmount checks if a string like "5 gold" or "10 kragenmark" is a money amount.
// Returns (amount, currency_name, true) or (0, "", false).
func parseMoneyAmount(s string) (int, string, bool) {
	parts := strings.Fields(s)
	if len(parts) < 2 {
		return 0, "", false
	}
	amount, err := strconv.Atoi(parts[0])
	if err != nil || amount <= 0 {
		return 0, "", false
	}
	currency := strings.ToLower(strings.Join(parts[1:], " "))
	// Recognize all currency types
	switch currency {
	case "gold", "crown", "crowns", "gold crown", "gold crowns":
		return amount, "gold", true
	case "silver", "shilling", "shillings", "silver shilling", "silver shillings":
		return amount, "silver", true
	case "copper", "penny", "pennies", "copper penny", "copper pennies":
		return amount, "copper", true
	case "coin", "coins":
		return amount, "copper", true
	case "kragenmark", "kragenmarks":
		return amount, "kragenmark", true
	case "danir", "danirs":
		return amount, "danir", true
	case "shard", "shards":
		return amount, "shard", true
	case "darktar", "darktars":
		return amount, "darktar", true
	case "dollar", "dollars":
		return amount, "dollar", true
	}
	return 0, "", false
}

// doGiveMoney transfers currency from one player to another.
func (e *GameEngine) doGiveMoney(ctx context.Context, giver, receiver *Player, amount int, currency string) *CommandResult {
	// Check if giver has enough
	currencyDisplay := ""
	switch currency {
	case "gold":
		if giver.Gold < amount {
			return &CommandResult{Messages: []string{fmt.Sprintf("You only have %d gold.", giver.Gold)}}
		}
		giver.Gold -= amount
		receiver.Gold += amount
		currencyDisplay = fmt.Sprintf("%d gold crown", amount)
		if amount != 1 {
			currencyDisplay += "s"
		}
	case "silver":
		if giver.Silver < amount {
			return &CommandResult{Messages: []string{fmt.Sprintf("You only have %d silver.", giver.Silver)}}
		}
		giver.Silver -= amount
		receiver.Silver += amount
		currencyDisplay = fmt.Sprintf("%d silver shilling", amount)
		if amount != 1 {
			currencyDisplay += "s"
		}
	case "copper":
		if giver.Copper < amount {
			return &CommandResult{Messages: []string{fmt.Sprintf("You only have %d copper.", giver.Copper)}}
		}
		giver.Copper -= amount
		receiver.Copper += amount
		currencyDisplay = fmt.Sprintf("%d copper penn", amount)
		if amount == 1 {
			currencyDisplay += "y"
		} else {
			currencyDisplay += "ies"
		}
	default:
		// Regional currencies — these are handled as inventory items with MONEY type
		// Find the currency item in giver's inventory
		for i, ii := range giver.Inventory {
			def := e.items[ii.Archetype]
			if def == nil || def.Type != "MONEY" {
				continue
			}
			noun := strings.ToLower(e.nouns[def.NameID])
			if noun == currency || strings.HasPrefix(noun, currency) {
				coins := ii.Val1
				if coins < amount {
					return &CommandResult{Messages: []string{fmt.Sprintf("You only have %d %s.", coins, currency)}}
				}
				if coins == amount {
					// Transfer the whole stack
					receiver.Inventory = append(receiver.Inventory, ii)
					giver.Inventory = append(giver.Inventory[:i], giver.Inventory[i+1:]...)
				} else {
					// Split the stack
					giver.Inventory[i].Val1 -= amount
					newItem := ii
					newItem.Val1 = amount
					receiver.Inventory = append(receiver.Inventory, newItem)
				}
				currencyDisplay = fmt.Sprintf("%d %s", amount, currency)
				if amount != 1 {
					currencyDisplay += "s"
				}
				e.SavePlayer(ctx, giver)
				e.SavePlayer(ctx, receiver)
				return &CommandResult{
					Messages:      []string{fmt.Sprintf("You give %s to %s.", currencyDisplay, receiver.FirstName)},
					RoomBroadcast: []string{fmt.Sprintf("%s gives some coins to %s.", giver.FirstName, receiver.FirstName)},
					TargetName:    receiver.FirstName,
					TargetMsg:     []string{fmt.Sprintf("%s gives you %s.", giver.FirstName, currencyDisplay)},
				}
			}
		}
		return &CommandResult{Messages: []string{fmt.Sprintf("You don't have any %s.", currency)}}
	}

	e.SavePlayer(ctx, giver)
	e.SavePlayer(ctx, receiver)
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You give %s to %s.", currencyDisplay, receiver.FirstName)},
		RoomBroadcast: []string{fmt.Sprintf("%s gives some coins to %s.", giver.FirstName, receiver.FirstName)},
		TargetName:    receiver.FirstName,
		TargetMsg:     []string{fmt.Sprintf("%s gives you %s.", giver.FirstName, currencyDisplay)},
	}
}

func (e *GameEngine) doEat(ctx context.Context, player *Player, args []string) *CommandResult {
	if player.RoundTimeExpiry.After(time.Now()) {
		remaining := int(time.Until(player.RoundTimeExpiry).Seconds()) + 1
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("[Wait %d seconds...]", remaining),
			},
		}
	}

	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Eat what?"}}
	}

	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	room := e.rooms[player.RoomNumber]

	for i := range player.Inventory {
		ii := &player.Inventory[i]

		itemDef := e.items[ii.Archetype]
		if itemDef == nil || itemDef.Type != "FOOD" {
			continue
		}

		name := e.getItemNounName(itemDef)

		matched :=
			matchesTarget(name, target, e.getAdjName(ii.Adj1)) ||
				matchesTarget(name, target, e.getAdjName(ii.Adj2)) ||
				matchesTarget(name, target, e.getAdjName(ii.Adj3))

		if !matched {
			continue
		}

		if skip > 0 {
			skip--
			continue
		}

		fullName := e.formatItemName(
			itemDef,
			ii.Adj1,
			ii.Adj2,
			ii.Adj3,
		)

		// FOOD Parameter1 is the starting bite count.
		// Scripted foods need ITEMVAL2 populated before IFPREVERB.
		if ii.Val2 == 0 && itemDef.Parameter1 > 0 {
			ii.Val2 = itemDef.Parameter1
		}

		tempRI := gameworld.RoomItem{
			Ref:       -1,
			Archetype: ii.Archetype,
			Adj1:      ii.Adj1,
			Adj2:      ii.Adj2,
			Adj3:      ii.Adj3,
			Val1:      ii.Val1,
			Val2:      ii.Val2,
			Val3:      ii.Val3,
			Val4:      ii.Val4,
			Val5:      ii.Val5,
		}

		// Root-level item scripts may initialize ITEMVALs.
		rootSC := e.RunItemScripts(
			player,
			room,
			&tempRI,
			itemDef,
		)

		// PREVERB runs before normal eating and may CLEARVERB.
		preSC := e.RunPreverbScripts(
			player,
			room,
			"EAT",
			&tempRI,
			itemDef,
		)

		// Persist changes made before the normal action.
		ii.Val1 = tempRI.Val1
		ii.Val2 = tempRI.Val2
		ii.Val3 = tempRI.Val3
		ii.Val4 = tempRI.Val4
		ii.Val5 = tempRI.Val5

		var messages []string
		var roomMessages []string
		var gmMessages []string

		messages = append(messages, rootSC.Messages...)
		roomMessages = append(roomMessages, rootSC.RoomMsgs...)
		gmMessages = append(gmMessages, rootSC.GMMsgs...)

		messages = append(messages, preSC.Messages...)
		roomMessages = append(roomMessages, preSC.RoomMsgs...)
		gmMessages = append(gmMessages, preSC.GMMsgs...)

		// Script completely handled EAT.
		if preSC.Blocked {
			e.SavePlayer(ctx, player)

			return &CommandResult{
				Messages:      messages,
				RoomBroadcast: roomMessages,
				GMBroadcast:   gmMessages,
				PlayerState:   player,
			}
		}

		// --------------------------------------------------------
		// Normal eating
		// --------------------------------------------------------

		currentBites := ii.Val2

		if currentBites <= 1 {
			player.Inventory = append(
				player.Inventory[:i],
				player.Inventory[i+1:]...,
			)

			messages = append(
				messages,
				fmt.Sprintf("You finish eating %s.", fullName),
			)

			// IFVERB still gets the temporary item context.
			postSC := e.RunVerbScripts(
				player,
				room,
				"EAT",
				&tempRI,
				itemDef,
			)

			messages = append(messages, postSC.Messages...)
			roomMessages = append(roomMessages, postSC.RoomMsgs...)
			gmMessages = append(gmMessages, postSC.GMMsgs...)

		} else {
			ii.Val2--
			tempRI.Val2 = ii.Val2

			messages = append(
				messages,
				fmt.Sprintf(
					"You take a bite of %s. (%d bites remaining)",
					fullName,
					ii.Val2,
				),
			)

			// IFVERB runs after successful normal EAT.
			postSC := e.RunVerbScripts(
				player,
				room,
				"EAT",
				&tempRI,
				itemDef,
			)

			messages = append(messages, postSC.Messages...)
			roomMessages = append(roomMessages, postSC.RoomMsgs...)
			gmMessages = append(gmMessages, postSC.GMMsgs...)

			// Persist any ITEMVAL changes made by IFVERB.
			ii.Val1 = tempRI.Val1
			ii.Val2 = tempRI.Val2
			ii.Val3 = tempRI.Val3
			ii.Val4 = tempRI.Val4
			ii.Val5 = tempRI.Val5
		}

		roomMessages = append(
			roomMessages,
			fmt.Sprintf("%s eats %s.", player.FirstName, fullName),
		)

		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages:      messages,
			RoomBroadcast: roomMessages,
			GMBroadcast:   gmMessages,
			PlayerState:   player,
		}
	}

	return &CommandResult{
		Messages: []string{"You don't have that."},
	}
}

func (e *GameEngine) doSpeech(ctx context.Context, player *Player, args []string, rawInput string) *CommandResult {
	if len(args) == 0 {
		if player.SpeechAdverb != "" {
			return &CommandResult{Messages: []string{fmt.Sprintf("Your speech manner is: %s. Use SPEECH CLEAR to remove it.", player.SpeechAdverb)}}
		}
		return &CommandResult{Messages: []string{"You have no speech manner set. Use SPEECH <adverb> to set one (e.g. SPEECH gently)."}}
	}
	adverb := strings.ToLower(args[0])
	if adverb == "clear" || adverb == "none" || adverb == "off" {
		player.SpeechAdverb = ""
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{"Speech manner cleared."}}
	}
	player.SpeechAdverb = adverb
	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: []string{fmt.Sprintf("You will now %s say things.", adverb)}}
}

func (e *GameEngine) doInfo(player *Player) *CommandResult {
	return &CommandResult{Messages: []string{
		fmt.Sprintf("Name: %s", player.FullName()),
		fmt.Sprintf("Race: %s   Gender: %s   Level: %d", player.RaceName(), genderName(player.Gender), player.Level),
		"",
		fmt.Sprintf("Strength: %-3d   Agility: %-3d   Quickness: %d", player.EffectiveStat(StatStrength), player.EffectiveStat(StatAgility), player.EffectiveStat(StatQuickness)),
		fmt.Sprintf("Constitution: %-3d   Perception: %-3d   Willpower: %-3d   Empathy: %d", player.EffectiveStat(StatConstitution), player.EffectiveStat(StatPerception), player.EffectiveStat(StatWillpower), player.EffectiveStat(StatEmpathy)),
		"",
		fmt.Sprintf("Body Points: %d/%d   Fatigue: %d/%d", player.BodyPoints, player.MaxBodyPoints, player.Fatigue, player.MaxFatigue),
		fmt.Sprintf("Mana: %d/%d   Psi: %d/%d", player.Mana, player.MaxMana, player.Psi, player.MaxPsi),
		"",
		fmt.Sprintf("Experience: %d   Build Points: %d", player.Experience, player.Experience/100),
	}}
}

func (e *GameEngine) doBuy(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Buy what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil || len(room.StoreItems) == 0 {
		return &CommandResult{Messages: []string{"There is nothing for sale here."}}
	}

	for _, si := range room.StoreItems {
		itemDef := e.items[si.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		adjName := ""
		if si.Adj > 0 {
			adjName = e.getAdjName(si.Adj)
		}
		if !matchesTarget(name, target, adjName) {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}

		// Check affordability
		totalCopper := player.Gold*100 + player.Silver*10 + player.Copper
		if totalCopper < si.Price {
			priceStr := formatPrice(si.Price)
			return &CommandResult{Messages: []string{fmt.Sprintf("You can't afford that. %s costs %s.", name, priceStr)}}
		}

		// Deduct currency efficiently (spend copper first, then silver, then gold)
		remaining := si.Price
		if player.Copper >= remaining {
			player.Copper -= remaining
			remaining = 0
		} else {
			remaining -= player.Copper
			player.Copper = 0
		}
		if remaining > 0 {
			silverNeeded := (remaining + 9) / 10 // round up
			if player.Silver >= silverNeeded {
				player.Silver -= silverNeeded
				player.Copper += silverNeeded*10 - remaining
				remaining = 0
			} else {
				remaining -= player.Silver * 10
				player.Silver = 0
			}
		}
		if remaining > 0 {
			goldNeeded := (remaining + 99) / 100 // round up
			player.Gold -= goldNeeded
			player.Copper += goldNeeded*100 - remaining
		}

		// Give item to player
		item := InventoryItem{Archetype: si.Archetype}
		if si.Adj > 0 {
			// Store adjective goes in ADJ3 (last slot) — ADJ1/ADJ2 left open for
			// crafting/enchanting. Item scripts check ITEMADJ3 for the variety.
			item.Adj3 = si.Adj
		}
		player.Inventory = append(player.Inventory, item)
		e.SavePlayer(ctx, player)

		displayName := e.formatItemName(itemDef, item.Adj1, item.Adj2, item.Adj3)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You hand over your money and retrieve your %s.", displayName)},
			RoomBroadcast: []string{fmt.Sprintf("%s purchases the %s.", player.FirstName, displayName)},
		}
	}

	return &CommandResult{Messages: []string{"That item is not for sale here."}}
}

func (e *GameEngine) doSell(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Sell what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't sell anything here."}}
	}

	// Check if room has any BUY_ modifier
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

	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			if skip > 0 {
				skip--
				continue
			}
			displayName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			// Sell value: VAL1 on item is copper value, fallback to weight-based estimate
			sellValue := ii.Val1
			if sellValue <= 0 {
				sellValue = itemDef.Weight + 1 // minimal fallback
			}
			// Merchants pay ~50% of value
			sellValue = sellValue / 2
			if sellValue < 1 {
				sellValue = 1
			}
			player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
			// Add coins
			player.Gold += sellValue / 100
			player.Silver += (sellValue % 100) / 10
			player.Copper += sellValue % 10
			e.SavePlayer(ctx, player)
			return &CommandResult{Messages: []string{
				fmt.Sprintf("The merchant inspects %s closely.", displayName),
				fmt.Sprintf("The merchant takes the item and hands you %s.", formatPrice(sellValue)),
			}}
		}
	}

	return &CommandResult{Messages: []string{"You don't have that."}}
}

func (e *GameEngine) doAppraise(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Appraise what?"}}
	}
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}
	canBuy := false
	for _, mod := range room.Modifiers {
		if strings.HasPrefix(mod, "BUY_") {
			canBuy = true
			break
		}
	}
	if !canBuy {
		return &CommandResult{Messages: []string{"There is no merchant here to appraise your items."}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	for _, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			if skip > 0 {
				skip--
				continue
			}
			displayName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			sellValue := ii.Val1
			if sellValue <= 0 {
				sellValue = itemDef.Weight + 1
			}
			sellValue = sellValue / 2
			if sellValue < 1 {
				sellValue = 1
			}
			return &CommandResult{Messages: []string{
				fmt.Sprintf("The merchant examines %s carefully.", displayName),
				fmt.Sprintf("\"I'd give you %s for that.\"", formatPrice(sellValue)),
			}}
		}
	}
	return &CommandResult{Messages: []string{"You don't have that."}}
}

// formatPrice formats a copper amount as a readable price string.
func formatPrice(copper int) string {
	gold := copper / 100
	remainder := copper % 100
	silver := remainder / 10
	cop := remainder % 10
	var parts []string
	if gold > 0 {
		if gold == 1 {
			parts = append(parts, "1 gold crown")
		} else {
			parts = append(parts, fmt.Sprintf("%d gold crowns", gold))
		}
	}
	if silver > 0 {
		if silver == 1 {
			parts = append(parts, "1 silver shilling")
		} else {
			parts = append(parts, fmt.Sprintf("%d silver shillings", silver))
		}
	}
	if cop > 0 || len(parts) == 0 {
		if cop == 1 {
			parts = append(parts, "1 copper penny")
		} else {
			parts = append(parts, fmt.Sprintf("%d copper pennies", cop))
		}
	}
	return joinList(parts)
}

func (e *GameEngine) doDrink(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Drink what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		if itemDef.Type != "LIQUID" && itemDef.Type != "LIQCONTAINER" && itemDef.Type != "FOOD" {
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
		displayName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
		if itemDef.Type == "FOOD" {
			// EAT logic — redirect to doEat
			return e.doEat(ctx, player, args)
		}

		// Sip tracking: initialize Val2 from Parameter1 on first sip
		currentSips := ii.Val2
		if currentSips == 0 && itemDef.Parameter1 > 0 {
			currentSips = itemDef.Parameter1
		}
		isFirstSip := (currentSips == 0 && itemDef.Parameter1 == 0) || (ii.Val2 == 0 && itemDef.Parameter1 > 0)

		// Run item scripts for spell effects
		room := e.rooms[player.RoomNumber]
		tempRI := gameworld.RoomItem{Ref: -1, Archetype: ii.Archetype,
			Adj1: ii.Adj1, Adj2: ii.Adj2, Adj3: ii.Adj3,
			Val1: ii.Val1, Val2: ii.Val2, Val3: ii.Val3, Val4: ii.Val4, Val5: ii.Val5}
		sc := e.RunItemScripts(player, room, &tempRI, itemDef)
		spellNum := tempRI.Val3

		var msgs []string
		if currentSips <= 1 {
			// Last sip (or single-sip drink) — remove from inventory
			player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
			msgs = []string{fmt.Sprintf("You finish drinking %s.", displayName)}
		} else {
			newVal := currentSips - 1
			player.Inventory[i].Val2 = newVal
			msgs = []string{fmt.Sprintf("You take a sip from %s. (%d sips remaining)", displayName, newVal)}
		}

		msgs = append(msgs, sc.Messages...)

		// Spell effect fires on FIRST sip only
		if isFirstSip && spellNum != 0 {
			if spellNum == 403 {
				//TODO:This needs to be cast 403  or put in script
				player.TelepathyActive = true
				player.TelepathyExpiry = time.Now().Add(1 * time.Hour)
				msgs = append(msgs, "You feel your mind open to the thoughts of others.")
			} else {
				msgs = append(msgs, fmt.Sprintf("[Spell #%d effect coming soon.]", spellNum))
			}
		}
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      msgs,
			RoomBroadcast: []string{fmt.Sprintf("%s drinks from %s.", player.FirstName, displayName)},
			PlayerState:   player,
		}
	}
	return &CommandResult{Messages: []string{"You don't have that."}}
}

func (e *GameEngine) doLight(
	ctx context.Context,
	player *Player,
	args []string,
	lightOn bool,
) *CommandResult {

	if len(args) == 0 {
		if lightOn {
			return &CommandResult{Messages: []string{"Light what?"}}
		}
		return &CommandResult{Messages: []string{"Extinguish what?"}}
	}

	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)

	//
	// Check carried items first
	//
	skip := ordSkip

	for i := range player.Inventory {
		ii := &player.Inventory[i]

		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}

		if !containsFlag(itemDef.Flags, "LIGHTABLE") {
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

		if lightOn {
			if ii.State == "LIT" {
				displayName := e.formatItemName(
					itemDef,
					ii.Adj1,
					ii.Adj2,
					ii.Adj3,
				)

				return &CommandResult{
					Messages: []string{
						fmt.Sprintf("%s is already lit.", displayName),
					},
				}
			}

			// Get the name BEFORE applying the lit adjective.
			originalName := e.formatItemName(
				itemDef,
				ii.Adj1,
				ii.Adj2,
				ii.Adj3,
			)

			// Apply the lit adjective.
			if itemDef.Parameter1 > 0 {
				ii.Val5 = ii.Adj1
				ii.Adj1 = itemDef.Parameter1
			}

			ii.State = "LIT"

			e.SavePlayer(ctx, player)

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf("You light %s.", originalName),
				},
				RoomBroadcast: []string{
					fmt.Sprintf("%s lights %s.", player.FirstName, originalName),
				},
			}
		}

		// Extinguish
		if ii.State != "LIT" {
			displayName := e.formatItemName(
				itemDef,
				ii.Adj1,
				ii.Adj2,
				ii.Adj3,
			)

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf("%s is not lit.", displayName),
				},
			}
		}

		if itemDef.Parameter1 > 0 {
			ii.Adj1 = ii.Val5
			ii.Val5 = 0
		}

		ii.State = "UNLIT"

		displayName := e.formatItemName(
			itemDef,
			ii.Adj1,
			ii.Adj2,
			ii.Adj3,
		)

		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("You extinguish %s.", displayName),
			},
			RoomBroadcast: []string{
				fmt.Sprintf("%s extinguishes %s.", player.FirstName, displayName),
			},
		}
	}

	//
	// Then check items in the room
	//
	room := e.rooms[player.RoomNumber]
	if room != nil {
		skip = ordSkip

		for i := range room.Items {
			ri := &room.Items[i]

			itemDef := e.items[ri.Archetype]
			if itemDef == nil {
				continue
			}

			if !containsFlag(itemDef.Flags, "LIGHTABLE") {
				continue
			}

			name := e.getItemNounName(itemDef)
			if !matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
				continue
			}

			if skip > 0 {
				skip--
				continue
			}

			if lightOn {
				if ri.State == "LIT" {
					displayName := e.formatItemName(
						itemDef,
						ri.Adj1,
						ri.Adj2,
						ri.Adj3,
					)

					return &CommandResult{
						Messages: []string{
							fmt.Sprintf("%s is already lit.", displayName),
						},
					}
				}

				if itemDef.Parameter1 > 0 {
					ri.Val5 = ri.Adj1
					ri.Adj1 = itemDef.Parameter1
				}

				ri.State = "LIT"

				displayName := e.formatItemName(
					itemDef,
					ri.Adj1,
					ri.Adj2,
					ri.Adj3,
				)

				e.notifyRoomChange(RoomChange{
					RoomNumber: player.RoomNumber,
					Type:       "item_update",
					Item:       ri,
				})

				return &CommandResult{
					Messages: []string{
						fmt.Sprintf("You light %s.", displayName),
					},
					RoomBroadcast: []string{
						fmt.Sprintf("%s lights %s.", player.FirstName, displayName),
					},
				}
			}

			// Extinguish
			if ri.State != "LIT" {
				displayName := e.formatItemName(
					itemDef,
					ri.Adj1,
					ri.Adj2,
					ri.Adj3,
				)

				return &CommandResult{
					Messages: []string{
						fmt.Sprintf("%s is not lit.", displayName),
					},
				}
			}

			if itemDef.Parameter1 > 0 {
				ri.Adj1 = ri.Val5
				ri.Val5 = 0
			}

			ri.State = "UNLIT"

			displayName := e.formatItemName(
				itemDef,
				ri.Adj1,
				ri.Adj2,
				ri.Adj3,
			)

			e.notifyRoomChange(RoomChange{
				RoomNumber: player.RoomNumber,
				Type:       "item_update",
				Item:       ri,
			})

			return &CommandResult{
				Messages: []string{
					fmt.Sprintf("You extinguish %s.", displayName),
				},
				RoomBroadcast: []string{
					fmt.Sprintf("%s extinguishes %s.", player.FirstName, displayName),
				},
			}
		}
	}

	if lightOn {
		return &CommandResult{
			Messages: []string{"You don't see anything here that you can light."},
		}
	}

	return &CommandResult{
		Messages: []string{"You don't see anything here that you can extinguish."},
	}
}

func (e *GameEngine) doFlip(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Flip what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		if !containsFlag(itemDef.Flags, "FLIPABLE") {
			continue
		}
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}
		displayName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
		if ri.State == "FLIPPED" {
			room.Items[i].State = "UNFLIPPED"
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "UNFLIPPED"})
		} else {
			room.Items[i].State = "FLIPPED"
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "FLIPPED"})
		}
		// Run IFPREVERB FLIP scripts
		sc := e.RunPreverbScripts(player, room, "FLIP", &room.Items[i], itemDef)
		result := &CommandResult{Messages: []string{fmt.Sprintf("You flip %s.", displayName)}}
		result.Messages = append(result.Messages, sc.Messages...)
		result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
		return result
	}
	return &CommandResult{Messages: []string{"You don't see anything to flip here."}}
}

func (e *GameEngine) doLatch(player *Player, args []string, latch bool) *CommandResult {
	if len(args) == 0 {
		if latch {
			return &CommandResult{Messages: []string{"Latch what?"}}
		}
		return &CommandResult{Messages: []string{"Unlatch what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		if !containsFlag(itemDef.Flags, "LATCHABLE") {
			continue
		}
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}
		displayName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
		if latch {
			room.Items[i].State = "LATCHED"
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "LATCHED"})
			return &CommandResult{Messages: []string{fmt.Sprintf("You latch %s.", displayName)}}
		}
		room.Items[i].State = "UNLATCHED"
		e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "UNLATCHED"})
		return &CommandResult{Messages: []string{fmt.Sprintf("You unlatch %s.", displayName)}}
	}
	return &CommandResult{Messages: []string{"You don't see anything to latch here."}}
}

func (e *GameEngine) doLock(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Lock what?"}}
	}
	raw := strings.ToLower(strings.Join(args, " "))
	target, keyName := parseWithClause(raw)
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		if !containsFlag(itemDef.Flags, "LOCKABLE") {
			continue
		}
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}
		if ri.State == "LOCKED" {
			return &CommandResult{Messages: []string{"It's already locked."}}
		}
		// Find matching key
		keyItem := e.findKey(player, ri.Val3, keyName)
		if keyItem == nil {
			return &CommandResult{Messages: []string{"You don't have the right key."}}
		}
		room.Items[i].State = "LOCKED"
		e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "LOCKED"})
		displayName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
		return &CommandResult{Messages: []string{fmt.Sprintf("You lock %s.", displayName)}}
	}
	return &CommandResult{Messages: []string{"You don't see anything to lock here."}}
}

func (e *GameEngine) doUnlock(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Unlock what?"}}
	}
	raw := strings.ToLower(strings.Join(args, " "))
	target, keyName := parseWithClause(raw)
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		if !containsFlag(itemDef.Flags, "LOCKABLE") {
			continue
		}
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}
		if ri.State != "LOCKED" {
			return &CommandResult{Messages: []string{"It isn't locked."}}
		}
		// Find matching key
		keyItem := e.findKey(player, ri.Val3, keyName)
		if keyItem == nil {
			return &CommandResult{Messages: []string{"You don't have the right key."}}
		}
		room.Items[i].State = "CLOSED"
		e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "CLOSED"})
		displayName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
		return &CommandResult{Messages: []string{fmt.Sprintf("You unlock %s.", displayName)}}
	}
	return &CommandResult{Messages: []string{"You don't see anything to unlock here."}}
}

// parseWithClause splits "target with key" into (target, key). If no "with", key is "".
func parseWithClause(s string) (string, string) {
	idx := strings.Index(s, " with ")
	if idx < 0 {
		return s, ""
	}
	return strings.TrimSpace(s[:idx]), strings.TrimSpace(s[idx+6:])
}

// findKey searches the player's inventory for a KEY-type item whose Val3 matches lockVal3.
// If keyName is non-empty, the key must also match that name.
func (e *GameEngine) findKey(player *Player, lockVal3 int, keyName string) *InventoryItem {
	allItems := make([]InventoryItem, 0, len(player.Inventory))
	allItems = append(allItems, player.Inventory...)
	for i := range allItems {
		ii := &allItems[i]
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		if !strings.EqualFold(itemDef.Type, "KEY") {
			continue
		}
		if ii.Val3 != lockVal3 {
			continue
		}
		if keyName != "" {
			name := e.getItemNounName(itemDef)
			if !matchesTarget(name, keyName, e.getAdjName(ii.Adj1)) {
				continue
			}
		}
		return ii
	}
	return nil
}

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

func (e *GameEngine) doRoomScriptVerb(ctx context.Context, player *Player, verb string) *CommandResult {

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return nil
	}

	scriptItem := gameworld.RoomItem{Ref: -1}
	scriptDef := &gameworld.ItemDef{}

	sc := e.RunVerbScripts(
		player,
		room,
		verb,
		&scriptItem,
		scriptDef,
	)

	// No matching room script.
	if len(sc.Messages) == 0 &&
		len(sc.RoomMsgs) == 0 &&
		len(sc.GMMsgs) == 0 &&
		!sc.Blocked &&
		sc.MoveTo == 0 &&
		sc.MoveGroupTo == 0 {

		return nil
	}

	result := &CommandResult{
		Messages:      append([]string{}, sc.Messages...),
		RoomBroadcast: append([]string{}, sc.RoomMsgs...),
		GMBroadcast:   append([]string{}, sc.GMMsgs...),
	}

	// The script requested MOVE <room>.
	if sc.MoveTo > 0 {
		dest := e.rooms[sc.MoveTo]

		if dest != nil {
			oldRoom := player.RoomNumber

			player.RoomNumber = sc.MoveTo
			e.SavePlayer(ctx, player)

			lookResult := e.doLook(player)

			result.Messages = append(
				result.Messages,
				lookResult.Messages...,
			)

			result.RoomName = lookResult.RoomName
			result.RoomDesc = lookResult.RoomDesc
			result.Exits = lookResult.Exits
			result.Items = lookResult.Items
			result.OldRoom = oldRoom

			e.applyEntryScripts(ctx, player, dest, result)
		}
	}

	// The script requested MOVEGROUP <room>.
	if sc.MoveGroupTo > 0 {
		e.moveGroupToRoom(
			ctx,
			player.RoomNumber,
			sc.MoveGroupTo,
		)
	}

	e.SavePlayer(ctx, player)

	return result
}

func (e *GameEngine) doDeposit(ctx context.Context, player *Player, args []string) *CommandResult {

	room := e.rooms[player.RoomNumber]
	if room == nil || !containsModifier(room.Modifiers, "BANK") {
		return &CommandResult{Messages: []string{"There is no bank here."}}
	}

	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Deposit what?"}}
	}

	// If the first argument is a number, it's a money deposit.
	if amount, err := strconv.Atoi(args[0]); err == nil {
		return e.doDepositMoney(ctx, player, amount)
	}

	// Otherwise, treat it as an item.
	return e.doDepositItem(ctx, player, args)
}

func (e *GameEngine) doDepositMoney(ctx context.Context, player *Player, amount int) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil || !containsModifier(room.Modifiers, "BANK") {
		return &CommandResult{Messages: []string{"There is no bank here."}}
	}
	if amount == 0 {
		return &CommandResult{Messages: []string{"Deposit how much?"}}
	}

	if amount <= 0 {
		return &CommandResult{Messages: []string{"Invalid amount."}}
	}
	totalCopper := player.Gold*100 + player.Silver*10 + player.Copper
	if totalCopper < amount {
		return &CommandResult{Messages: []string{"You don't have that much money."}}
	}
	// Deduct from carried
	remaining := amount
	if player.Copper >= remaining {
		player.Copper -= remaining
		remaining = 0
	} else {
		remaining -= player.Copper
		player.Copper = 0
	}
	if remaining > 0 {
		sn := (remaining + 9) / 10
		if player.Silver >= sn {
			player.Silver -= sn
			player.Copper += sn*10 - remaining
			remaining = 0
		} else {
			remaining -= player.Silver * 10
			player.Silver = 0
		}
	}
	if remaining > 0 {
		gn := (remaining + 99) / 100
		player.Gold -= gn
		player.Copper += gn*100 - remaining
	}
	player.BankCopper += amount
	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: []string{fmt.Sprintf("You deposit %s.", formatPrice(amount))}}
}

func (e *GameEngine) doDepositItem(ctx context.Context, player *Player, args []string) *CommandResult {

	if len(args) == 0 {
		return &CommandResult{
			Messages: []string{"Deposit what?"},
		}
	}

	if len(player.BankInventory) >= MaxBankItems {
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"Your safety deposit box is full. You may store up to %d items.",
					MaxBankItems,
				),
			},
			PlayerState: player,
		}
	}

	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}

		name := e.getItemNounName(itemDef)

		if !matchesTarget(
			name,
			target,
			e.getAdjName(ii.Adj1),
		) {
			continue
		}

		if skip > 0 {
			skip--
			continue
		}

		// One gold crown = 100 copper.
		const depositFee = 100

		totalCopper := player.Gold*100 +
			player.Silver*10 +
			player.Copper

		if totalCopper < depositFee {
			return &CommandResult{
				Messages: []string{
					"You need one gold crown to deposit an item.",
				},
			}
		}

		// Deduct the 100-copper deposit fee.
		remaining := depositFee

		if player.Copper >= remaining {
			player.Copper -= remaining
			remaining = 0
		} else {
			remaining -= player.Copper
			player.Copper = 0
		}

		if remaining > 0 {
			sn := (remaining + 9) / 10

			if player.Silver >= sn {
				player.Silver -= sn
				player.Copper += sn*10 - remaining
				remaining = 0
			} else {
				remaining -= player.Silver * 10
				player.Silver = 0
			}
		}

		if remaining > 0 {
			gn := (remaining + 99) / 100

			player.Gold -= gn
			player.Copper += gn*100 - remaining
		}

		// Preserve the full inventory item, including contents.
		player.BankInventory = append(
			player.BankInventory,
			ii,
		)

		// Remove it from carried inventory.
		player.Inventory = append(
			player.Inventory[:i],
			player.Inventory[i+1:]...,
		)

		e.SavePlayer(ctx, player)

		fullName := e.formatItemName(
			itemDef,
			ii.Adj1,
			ii.Adj2,
			ii.Adj3,
		)

		return &CommandResult{
			Messages: []string{
				fmt.Sprintf(
					"You hand the clerk a gold crown and place %s in your safety deposit box.",
					fullName,
				),
			},
			PlayerState: player,
		}
	}

	return &CommandResult{
		Messages: []string{"You aren't carrying that."},
	}
}

func (e *GameEngine) doWithdraw(ctx context.Context, player *Player, args []string) *CommandResult {

	room := e.rooms[player.RoomNumber]
	if room == nil || !containsModifier(room.Modifiers, "BANK") {
		return &CommandResult{
			Messages: []string{"There is no bank here."},
		}
	}

	if len(args) == 0 {
		return &CommandResult{
			Messages: []string{"Withdraw what?"},
		}
	}

	if amount, err := strconv.Atoi(args[0]); err == nil {
		return e.doWithdrawMoney(ctx, player, amount)
	}

	return e.doWithdrawItem(ctx, player, args)
}

func (e *GameEngine) doWithdrawMoney(ctx context.Context, player *Player, amount int) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil || !containsModifier(room.Modifiers, "BANK") {
		return &CommandResult{Messages: []string{"There is no bank here."}}
	}
	if amount == 0 {
		return &CommandResult{Messages: []string{"Withdraw how much?"}}
	}

	if amount <= 0 {
		return &CommandResult{Messages: []string{"Invalid amount."}}
	}
	totalBank := player.BankGold*100 + player.BankSilver*10 + player.BankCopper
	if totalBank < amount {
		return &CommandResult{Messages: []string{"You don't have that much in the bank."}}
	}
	remaining := amount
	if player.BankCopper >= remaining {
		player.BankCopper -= remaining
		remaining = 0
	} else {
		remaining -= player.BankCopper
		player.BankCopper = 0
	}
	if remaining > 0 {
		sn := (remaining + 9) / 10
		if player.BankSilver >= sn {
			player.BankSilver -= sn
			player.BankCopper += sn*10 - remaining
			remaining = 0
		} else {
			remaining -= player.BankSilver * 10
			player.BankSilver = 0
		}
	}
	if remaining > 0 {
		gn := (remaining + 99) / 100
		player.BankGold -= gn
		player.BankCopper += gn*100 - remaining
	}
	player.Copper += amount
	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: []string{fmt.Sprintf("You withdraw %s.", formatPrice(amount))}}
}

func (e *GameEngine) doWithdrawItem(ctx context.Context, player *Player, args []string) *CommandResult {

	if len(args) == 0 {
		return &CommandResult{
			Messages: []string{"Withdraw what?"},
		}
	}

	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	for i, ii := range player.BankInventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}

		name := e.getItemNounName(itemDef)

		if !matchesTarget(
			name,
			target,
			e.getAdjName(ii.Adj1),
		) {
			continue
		}

		if skip > 0 {
			skip--
			continue
		}

		// Move the complete item back into carried inventory.
		// Contents come with it automatically because they're
		// stored inside the inventory item.
		player.Inventory = append(
			player.Inventory,
			ii,
		)

		// Remove it from the safety deposit box.
		player.BankInventory = append(
			player.BankInventory[:i],
			player.BankInventory[i+1:]...,
		)

		e.SavePlayer(ctx, player)

		return &CommandResult{
			Messages: []string{
				"You retrieve the item from your safety deposit box.",
			},
			PlayerState: player,
		}
	}

	return &CommandResult{
		Messages: []string{
			"That item is not in your safety deposit box.",
		},
	}
}

func containsModifier(mods []string, mod string) bool {
	for _, m := range mods {
		if m == mod {
			return true
		}
	}
	return false
}

// SkillNames maps skill IDs to names.
var SkillNames = map[int]string{
	0: "Jeweler", 1: "Two Weapons", 2: "Backstab", 3: "Missile Weapons",
	4: "Natural Weapons", 5: "Climbing", 6: "Dodging & Parrying", 7: "Conjuration",
	8: "Weaponsmithing", 9: "Crushing Weapons", 10: "Combat Maneuvering",
	11: "Endurance", 12: "Trap & Poison Lore", 13: "Edged Weapons",
	14: "Enchantment", 15: "Dyeing/Weaving", 16: "Drakin Weapons",
	17: "Druidic Magic", 18: "Wood Lore", 19: "Thrown Weapons",
	20: "Healing", 21: "Legerdemain", 22: "Lockpicking", 23: "Spellcraft",
	24: "Martial Arts", 25: "Polearms", 26: "Psionics",
	27: "Mind over Mind", 28: "Mind over Matter", 29: "Transcendence",
	30: "Necromancy", 31: "Alchemy", 32: "Sagecraft", 33: "Stealth",
	34: "Disguise", 35: "Mining",
}

// doTrain replaced by doTrainWithBP in skills.go

func (e *GameEngine) doUnlearn(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Unlearn what skill?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	for id, name := range SkillNames {
		if !strings.HasPrefix(strings.ToLower(name), target) {
			continue
		}
		currentLvl := player.Skills[id]
		if currentLvl <= 0 {
			return &CommandResult{Messages: []string{fmt.Sprintf("You don't have any ranks in %s.", name)}}
		}
		// Unlearn one rank, get back build points minus one
		bpBack := max(0, currentLvl-1) // return BP spent minus 1
		player.Skills[id] = currentLvl - 1
		player.BuildPoints += bpBack
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{
			fmt.Sprintf("You unlearn a rank of %s. (now rank %d, +%d build points, total BP: %d)", name, currentLvl-1, bpBack, player.BuildPoints),
		}}
	}
	return &CommandResult{Messages: []string{"You don't know that skill."}}
}

// doMine and doForage moved to crafting.go

func (e *GameEngine) doPositionWithScripts(ctx context.Context, player *Player, verb, selfMsg, roomMsg string) *CommandResult {
	result := &CommandResult{
		Messages:      []string{selfMsg},
		RoomBroadcast: []string{roomMsg},
		PlayerState:   player,
	}
	// Run room-level IFVERB scripts for the position verb (e.g., IFVERB SIT -1)
	room := e.rooms[player.RoomNumber]
	if room != nil {
		verbUpper := strings.ToUpper(verb)
		for _, block := range room.Scripts {
			if (block.Type == "IFVERB" || block.Type == "IFPREVERB") && len(block.Args) >= 2 {
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

func (e *GameEngine) doHide(ctx context.Context, player *Player) *CommandResult {
	if player.Hidden {
		return &CommandResult{Messages: []string{"You are already hidden."}}
	}
	if player.Joined {
		return &CommandResult{Messages: []string{"You can't hide while in combat!"}}
	}
	// Stealth skill check: base 25% + stealth*5 + AGI/10
	stealthSkill := player.Skills[33]
	hideChance := 25 + stealthSkill*5 + player.EffectiveStat(StatAgility)/10
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
	// Stealth check to stay hidden while moving
	stealthSkill := player.Skills[33]
	sneakChance := 30 + stealthSkill*5 + player.EffectiveStat(StatAgility)/10
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

func (e *GameEngine) doFly(ctx context.Context, player *Player) *CommandResult {
	if player.Position == 4 {
		return &CommandResult{Messages: []string{"You are already flying."}}
	}
	// Drakin can always fly; others need CanFly (spell/item)
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

func (e *GameEngine) doMark(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		// Show marks
		if player.Marks == nil || len(player.Marks) == 0 {
			return &CommandResult{Messages: []string{"You have no marks set."}}
		}
		var msgs []string
		msgs = append(msgs, "Your marks:")
		for i := 1; i <= 10; i++ {
			if roomNum, ok := player.Marks[i]; ok {
				name := fmt.Sprintf("Room %d", roomNum)
				if r := e.rooms[roomNum]; r != nil {
					name = r.Name
				}
				if player.IsGM {
					msgs = append(msgs, fmt.Sprintf("  Mark %d: %s (%d)", i, name, roomNum))
				} else {
					msgs = append(msgs, fmt.Sprintf("  Mark %d: %s", i, name))
				}
			}
		}
		return &CommandResult{Messages: msgs}
	}
	num := 0
	fmt.Sscanf(args[0], "%d", &num)
	if num < 1 || num > 10 {
		return &CommandResult{Messages: []string{"Mark number must be 1-10."}}
	}
	if player.Marks == nil {
		player.Marks = make(map[int]int)
	}
	player.Marks[num] = player.RoomNumber
	e.SavePlayer(ctx, player)
	room := e.rooms[player.RoomNumber]
	name := fmt.Sprintf("room %d", player.RoomNumber)
	if room != nil {
		name = room.Name
	}
	return &CommandResult{Messages: []string{fmt.Sprintf("Mark %d set to %s.", num, name)}}
}

func (e *GameEngine) doUndress(ctx context.Context, player *Player) *CommandResult {
	if len(player.Worn) == 0 {
		return &CommandResult{Messages: []string{"You aren't wearing anything to remove."}}
	}
	// Remove the last worn item
	item := player.Worn[len(player.Worn)-1]
	player.Worn = player.Worn[:len(player.Worn)-1]
	player.Inventory = append(player.Inventory, item)
	e.SavePlayer(ctx, player)
	itemDef := e.items[item.Archetype]
	name := "something"
	if itemDef != nil {
		name = e.formatItemName(itemDef, item.Adj1, item.Adj2, item.Adj3)
	}
	return &CommandResult{Messages: []string{fmt.Sprintf("You remove %s.", name)}}
}

func (e *GameEngine) doBalance(player *Player) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil || !containsModifier(room.Modifiers, "BANK") {
		return &CommandResult{Messages: []string{"You need to be at a bank to check your balance."}}
	}

	msgs := []string{"=== Bank Balance ==="}

	total := player.BankGold*100 + player.BankSilver*10 + player.BankCopper
	if total == 0 {
		msgs = append(msgs, "Your account is empty.")
	} else {
		msgs = append(msgs, fmt.Sprintf("Balance: %s", formatPrice(total)))
	}

	// Safety deposit box
	msgs = append(msgs, "", "Safety Deposit Box:")

	if len(player.BankInventory) == 0 {
		msgs = append(msgs, "  Empty.")
	} else {
		for _, ii := range player.BankInventory {
			itemDef := e.items[ii.Archetype]
			if itemDef == nil {
				continue
			}

			fullName := e.formatItemName(
				itemDef,
				ii.Adj1,
				ii.Adj2,
				ii.Adj3,
			)

			msgs = append(msgs, "  "+fullName)
		}
	}

	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) doSpellList(player *Player) *CommandResult {
	if player.KnownSpells == nil || len(player.KnownSpells) == 0 {
		return &CommandResult{Messages: []string{"You don't know any spells."}}
	}
	msgs := []string{"=== Known Spells ==="}
	for id := range player.KnownSpells {
		spell := FindSpellByID(id)
		if spell != nil {
			msgs = append(msgs, fmt.Sprintf("  %s (%s, Level %d)", spell.Name, spell.School, spell.Level))
		}
	}
	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) doThink(player *Player, rawInput string) *CommandResult {
	text := extractOriginalArgs(rawInput)
	if text == "" {
		return &CommandResult{Messages: []string{"Think what?"}}
	}
	if !player.HasTelepathy() {
		return &CommandResult{Messages: []string{"You don't have telepathic ability right now."}}
	}
	return &CommandResult{
		Messages:        []string{"You project your thoughts."},
		TelepathyMsg:    text,
		TelepathySender: player.FirstName,
	}
}

func (e *GameEngine) doCant(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Cant what?"}}
	}
	// Requires Legerdemain (skill 21) rank 6+ or Stealth (skill 5)
	if player.Skills[21] < 6 && player.Skills[5] < 1 && !player.IsGM {
		return &CommandResult{Messages: []string{"You don't know how to speak in cant."}}
	}
	text := strings.Join(args, " ")
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You cant, \"%s\"", text)},
		RoomBroadcast: []string{fmt.Sprintf("%s mutters something under their breath.", player.FirstName)},
		CantMsg:       text,
		CantSender:    player.FirstName,
	}
}

func (e *GameEngine) doRead(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{
			Messages: []string{"Read what?"},
		}
	}

	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{
			Messages: []string{"There is nothing written on it."},
		}
	}

	// ------------------------------------------------------------
	// Search room items
	// ------------------------------------------------------------
	for i := range room.Items {
		ri := &room.Items[i]

		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}

		name := e.getItemNounName(itemDef)

		if !matchesTarget(
			name,
			target,
			e.getAdjName(ri.Adj1),
		) {
			continue
		}

		if skip > 0 {
			skip--
			continue
		}

		// --------------------------------------------------------
		// Give IFPREVERB READ <ref> first crack.
		// --------------------------------------------------------
		sc := e.RunPreverbScripts(
			player,
			room,
			"READ",
			ri,
			itemDef,
		)

		// Add routing information to @trace output.
		if player.GMTrace {
			sc.Messages = append(
				[]string{
					fmt.Sprintf(
						"[TRACE] READ matched ref=%d archetype=%d",
						ri.Ref,
						ri.Archetype,
					),
				},
				sc.Messages...,
			)
		}

		// --------------------------------------------------------
		// CLEARVERB means the script completely handled READ.
		// --------------------------------------------------------
		if sc.Blocked {
			if player.GMTrace {
				sc.Messages = append(
					sc.Messages,
					"[TRACE] READ blocked by CLEARVERB — skipping normal ITEM READ",
				)
			}

			return &CommandResult{
				Messages:      sc.Messages,
				RoomBroadcast: sc.RoomMsgs,
				GMBroadcast:   sc.GMMsgs,
				PlayerState:   player,
			}
		}

		// --------------------------------------------------------
		// No script override — fall through to normal ITEM READ.
		// --------------------------------------------------------
		if player.GMTrace {
			sc.Messages = append(
				sc.Messages,
				"[TRACE] READ not blocked — falling through to ITEM READ",
			)
		}

		result := e.readRoomItem(
			room,
			itemDef,
			ri,
		)

		// Preserve any non-blocking script output.
		result.Messages = append(
			sc.Messages,
			result.Messages...,
		)

		result.RoomBroadcast = append(
			sc.RoomMsgs,
			result.RoomBroadcast...,
		)

		result.GMBroadcast = append(
			sc.GMMsgs,
			result.GMBroadcast...,
		)

		return result
	}

	// ------------------------------------------------------------
	// Search player items:
	// inventory + worn + wielded
	// ------------------------------------------------------------

	allReadItems := make(
		[]InventoryItem,
		0,
		len(player.Inventory)+len(player.Worn)+1,
	)

	allReadItems = append(
		allReadItems,
		player.Inventory...,
	)

	allReadItems = append(
		allReadItems,
		player.Worn...,
	)

	if player.Wielded != nil {
		allReadItems = append(
			allReadItems,
			*player.Wielded,
		)
	}

	for _, ii := range allReadItems {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}

		name := e.getItemNounName(itemDef)

		matched :=
			matchesTarget(
				name,
				target,
				e.getAdjName(ii.Adj1),
			) ||
				matchesTarget(
					name,
					target,
					e.getAdjName(ii.Adj3),
				)

		if !matched {
			continue
		}

		if skip > 0 {
			skip--
			continue
		}

		if player.GMTrace {
			return &CommandResult{
				Messages: []string{
					fmt.Sprintf(
						"[TRACE] READ matched carried archetype=%d",
						ii.Archetype,
					),
					"[TRACE] No carried-item READ implementation found",
					"There is nothing written on it.",
				},
			}
		}

		return &CommandResult{
			Messages: []string{
				"There is nothing written on it.",
			},
		}
	}

	return &CommandResult{
		Messages: []string{
			"You don't see that here.",
		},
	}
}

func (e *GameEngine) doWho(player *Player) *CommandResult {
	var names []string
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.GMHidden {
				continue
			}
			name := p.FirstName
			if p.IsBot {
				name += " [Bot]"
			}
			if p.IsGM && p.GMHat {
				name += " [Host]"
			}
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return &CommandResult{Messages: []string{"No adventurers are in the Realms."}}
	}
	sort.Strings(names)
	// Build 4-column grid, 19-char columns
	var msgs []string
	for i := 0; i < len(names); i += 4 {
		line := ""
		for j := 0; j < 4 && i+j < len(names); j++ {
			line += fmt.Sprintf("%-19s", names[i+j])
		}
		msgs = append(msgs, line)
	}
	msgs = append(msgs, "")
	count := len(names)
	if count == 1 {
		msgs = append(msgs, "There is 1 adventurer in the Realms.")
	} else {
		msgs = append(msgs, fmt.Sprintf("There are %d adventurers in the Realms.", count))
	}
	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) doHelp() *CommandResult {
	return &CommandResult{Messages: []string{
		"=== Legends of Future Past - Commands ===",
		"Movement: N, S, E, W, NE, NW, SE, SW, UP, DOWN, OUT, GO <portal>",
		"Looking: LOOK, LOOK <item>, LOOK IN/ON/UNDER <item>, EXAMINE <item>",
		"Items: GET <item>, DROP <item>, INVENTORY, WIELD <weapon>, UNWIELD",
		"Wear: WEAR <item>, REMOVE <item>",
		"Containers: OPEN <item>, CLOSE <item>",
		"Info: STATUS, HEALTH, WEALTH, SKILLS, WHO",
		"Combat: ATTACK <target>, ADVANCE <target>, RETREAT",
		"Social: '<message> (say), ACT <action>, WHISPER <person> <msg>",
		"Position: SIT, STAND, KNEEL, LAY",
		"Settings: BRIEF, FULL",
		"System: HELP, ADVICE, QUIT",
	}}
}

// CreateNewPlayer generates a fresh character and persists it to MongoDB.
func (e *GameEngine) CreateNewPlayer(ctx context.Context, firstName, lastName string, race, gender int, accountID ...string) *Player {

	stats := rollStatsForRace(race)

	str := stats.Strength
	agi := stats.Agility
	qui := stats.Quickness
	con := stats.Constitution
	per := stats.Perception
	wil := stats.Willpower
	emp := stats.Empathy

	bodyPts := 20 + con/2
	fatigue := 20 + (con+str)/3
	mana := emp / 2
	psi := wil / 2

	// Race-based height/weight ranges: [minHeight, maxHeight, minWeight, maxWeight]
	// Heights in inches, weights in lbs. Based on original GM manual.
	heightWeightRanges := map[int][4]int{
		1: {62, 76, 120, 220}, // Human
		2: {66, 80, 100, 170}, // Aelfen (tall, slender)
		3: {48, 58, 130, 200}, // Highlander (short, rugged)
		4: {64, 74, 130, 200}, // Wolfling
		5: {62, 74, 150, 230}, // Murg (burly)
		6: {68, 82, 150, 250}, // Drakin (large)
		7: {60, 74, 150, 250}, // Mechanoid
		8: {58, 72, 80, 130},  // Ephemeral (wispy)
	}
	hw := heightWeightRanges[race]
	if hw == [4]int{} {
		hw = [4]int{62, 76, 120, 220} // fallback to human
	}
	height := hw[0] + rand.Intn(hw[1]-hw[0]+1)
	weight := hw[2] + rand.Intn(hw[3]-hw[2]+1)
	// Females slightly smaller on average
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
		FirstName:      firstName,
		LastName:       lastName,
		Race:           race,
		Gender:         gender,
		Level:          1,
		BuildPoints:    30, // 30 starting build points for initial skills
		Strength:       str,
		Agility:        agi,
		Quickness:      qui,
		Constitution:   con,
		Perception:     per,
		Willpower:      wil,
		Empathy:        emp,
		BodyPoints:     bodyPts,
		MaxBodyPoints:  bodyPts,
		Fatigue:        fatigue,
		MaxFatigue:     fatigue,
		Mana:           mana,
		MaxMana:        mana,
		Psi:            psi,
		MaxPsi:         psi,
		Height:         height,
		HeightTrue:     height,
		Weight:         weight,
		WeightTrue:     weight,
		RoomNumber:     201, // Start at City Gate (tutorial room 3950 requires script execution)
		Position:       0,
		Skills:         make(map[int]int),
		IntNums:        make(map[int]int),
		Gold:           5,
		Silver:         10,
		Copper:         50,
		PromptMode:     true,
		SuppressLogon:  true, // login/logout messages off by default for new characters
		SuppressLogoff: true,
		CreatedAt:      now,
		UpdatedAt:      now,
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

	// Backfill height/weight for existing characters that have 0
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

	// Find the deleted character
	var player Player
	err := coll.FindOne(ctx, bson.M{"firstName": firstName, "deletedAt": bson.M{"$exists": true}}).Decode(&player)
	if err != nil {
		return nil, fmt.Errorf("deleted character '%s' not found", firstName)
	}

	// Check if name conflicts with existing non-deleted character
	targetName := firstName
	if newFirstName != "" {
		targetName = newFirstName
	}
	taken, _ := e.IsFirstNameTaken(ctx, targetName)
	if taken {
		return nil, fmt.Errorf("name '%s' is already taken by an active character", targetName)
	}

	// Recover
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

// doSet handles the SET command for toggling player settings.
func (e *GameEngine) doSet(ctx context.Context, player *Player, args []string) *CommandResult {
	// Helper to format ON/OFF
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

	// Handle shortcut verbs: ACTBRIEF toggles ActBrief, RPBRIEF toggles RPBrief
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

	// SET with no args: show all settings
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

// Helper: format item name with adjectives
// formatItemNameNoArticle returns item name with adjectives but no article prefix.
func (e *GameEngine) formatItemNameNoArticle(def *gameworld.ItemDef, adj1, adj2, adj3 int) string {
	var parts []string
	if adj1 > 0 {
		if name, ok := e.adjectives[adj1]; ok {
			parts = append(parts, name)
		}
	}
	if adj2 > 0 {
		if name, ok := e.adjectives[adj2]; ok {
			parts = append(parts, name)
		}
	}
	if adj3 > 0 {
		if name, ok := e.adjectives[adj3]; ok {
			parts = append(parts, name)
		}
	}
	parts = append(parts, e.getItemNounName(def))
	return strings.Join(parts, " ")
}

func (e *GameEngine) formatItemName(def *gameworld.ItemDef, adj1, adj2, adj3 int) string {
	var parts []string
	if adj1 > 0 {
		if name, ok := e.adjectives[adj1]; ok {
			parts = append(parts, name)
		}
	}
	if adj2 > 0 {
		if name, ok := e.adjectives[adj2]; ok {
			parts = append(parts, name)
		}
	}
	if adj3 > 0 {
		if name, ok := e.adjectives[adj3]; ok {
			parts = append(parts, name)
		}
	}
	nounName := e.getItemNounName(def)
	parts = append(parts, nounName)

	name := strings.Join(parts, " ")
	article := strings.ToUpper(def.Article)
	if article == "" || article == "A" {
		// Auto-detect "an" for words starting with a vowel sound
		first := strings.ToLower(name[:1])
		if first == "a" || first == "e" || first == "i" || first == "o" || first == "u" {
			return "an " + name
		}
		return "a " + name
	}
	if article == "AN" {
		return "an " + name
	}
	if article == "THE" {
		return "the " + name
	}
	if article == "SOME" {
		return "some " + name
	}
	return strings.ToLower(article) + " " + name
}

func (e *GameEngine) getItemNounName(def *gameworld.ItemDef) string {
	if name, ok := e.nouns[def.NameID]; ok {
		return name
	}
	return fmt.Sprintf("item#%d", def.Number)
}

// adjByName returns the adjective ID for a given name (case-insensitive), or 0 if not found.
func (e *GameEngine) adjByName(name string) int {
	target := strings.ToLower(name)
	for id, adj := range e.adjectives {
		if strings.ToLower(adj) == target {
			return id
		}
	}
	return 0
}

func (e *GameEngine) getAdjName(adjID int) string {
	if adjID > 0 {
		if name, ok := e.adjectives[adjID]; ok {
			return name
		}
	}
	return ""
}

func (e *GameEngine) lookInContainer(player *Player, def *gameworld.ItemDef, ii *InventoryItem) *CommandResult {

	name := e.formatItemName(
		def,
		ii.Adj1,
		ii.Adj2,
		ii.Adj3,
	)

	if ii.State != "OPEN" && ii.State != "" {
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("You'll need to open %s first.", name),
			},
		}
	}

	msgs := []string{
		fmt.Sprintf("You look in %s.", name),
	}

	found := false
	usedVolume := 0

	for _, child := range ii.Contents {
		childDef := e.items[child.Archetype]
		if childDef == nil {
			continue
		}

		if childDef.Type == "MONEY" {
			switch childDef.Parameter1 {
			case MoneyGold:
				msgs = append(msgs, "You see some gold coins.")
			case MoneySilver:
				msgs = append(msgs, "You see some silver coins.")
			case MoneyCopper:
				msgs = append(msgs, "You see some copper coins.")
			default:
				msgs = append(msgs, "You see some coins.")
			}

			found = true
			continue
		}

		childName := e.formatItemName(
			childDef,
			child.Adj1,
			child.Adj2,
			child.Adj3,
		)

		usedVolume += childDef.Volume

		msgs = append(
			msgs,
			fmt.Sprintf("You see %s.", childName),
		)

		found = true
	}

	if def.Interior > 0 {
		msgs = append(
			msgs,
			fmt.Sprintf("Capacity: %d/%d", usedVolume, def.Interior),
		)
	}

	if !found {
		msgs = append(msgs, "It is empty.")
	}

	return &CommandResult{
		Messages: msgs,
	}
}

func (e *GameEngine) examineRoomItem(player *Player, room *gameworld.Room, def *gameworld.ItemDef, ri *gameworld.RoomItem) *CommandResult {

	result := &CommandResult{}

	refStr := fmt.Sprintf("%d", ri.Ref)

	// Explicit room-scoped EXAMINE description has highest priority.
	if desc, ok := room.ItemDescriptions["EXAMINE:"+refStr]; ok {

		result.Messages = append(
			result.Messages,
			descriptionToMessages(desc)...,
		)

	} else if isPortal(def.Type) {

		result.Messages = append(
			result.Messages,
			"You can't clearly see where it leads.",
		)

	} else {

		// Build the complete room-instance description:
		// article + adjectives + noun
		name := e.formatItemName(
			def,
			ri.Adj1,
			ri.Adj2,
			ri.Adj3,
		)

		// EXTEND adds instance-specific descriptive text.
		if ri.Extend != "" {
			name += " " + ri.Extend
		}

		result.Messages = append(
			result.Messages,
			name+".",
		)
	}

	// Run IFVERB LOOK scripts on the item.
	sc := e.RunVerbScripts(
		player,
		room,
		"LOOK",
		ri,
		def,
	)

	result.Messages = append(
		result.Messages,
		sc.Messages...,
	)

	result.RoomBroadcast = append(
		result.RoomBroadcast,
		sc.RoomMsgs...,
	)

	return result
}

func (e *GameEngine) readRoomItem(room *gameworld.Room, def *gameworld.ItemDef, ri *gameworld.RoomItem) *CommandResult {
	refStr := fmt.Sprintf("%d", ri.Ref)
	if desc, ok := room.ItemDescriptions["READ:"+refStr]; ok {
		return &CommandResult{Messages: descriptionToMessages(desc)}
	}
	return &CommandResult{Messages: []string{"There is nothing written on it."}}
}

func (e *GameEngine) lookPrefixRoomItem(room *gameworld.Room, def *gameworld.ItemDef, ri *gameworld.RoomItem, prefix string) *CommandResult {
	refStr := fmt.Sprintf("%d", ri.Ref)
	if desc, ok := room.ItemDescriptions[prefix+":"+refStr]; ok {
		return &CommandResult{Messages: descriptionToMessages(desc)}
	}
	name := e.getItemNounName(def)
	return &CommandResult{Messages: []string{fmt.Sprintf("You see nothing noteworthy %s the %s.", strings.ToLower(prefix), name)}}
}

// descriptionToMessages splits a description into message lines.
// Formatted text (containing newlines) is split on newlines to preserve layout.
// Plain text is returned as a single message.
// joinList formats a list as "a, b and c" or "a and b" or "a".
func joinList(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
	}
}

func descriptionToMessages(desc string) []string {
	if strings.Contains(desc, "\n") {
		return strings.Split(desc, "\n")
	}
	return []string{desc}
}

func (e *GameEngine) formatItemLook(def *gameworld.ItemDef, ri *gameworld.RoomItem) []string {
	name := e.formatItemName(def, ri.Adj1, ri.Adj2, ri.Adj3)
	if ri.Extend != "" {
		name += " " + ri.Extend
	}
	msgs := []string{fmt.Sprintf("You examine %s.", name)}
	if isPortal(def.Type) && ri.Val2 > 0 {
		msgs = append(msgs, "It appears to lead somewhere.")
	}
	return msgs
}

// parseOrdinal extracts an ordinal prefix from a target string.
// Returns (cleanTarget, ordinalNumber). ordinal 0 means "first/default".
// "2 gate" → ("gate", 1), "other gate" → ("gate", 1), "second gate" → ("gate", 1),
// "first gate" → ("gate", 0), "3 gate" → ("gate", 2), "gate" → ("gate", 0)
func parseOrdinal(target string) (string, int) {
	ordinalWords := map[string]int{
		"first": 0, "second": 1, "third": 2, "fourth": 3, "fifth": 4,
		"sixth": 5, "seventh": 6, "eighth": 7, "ninth": 8, "tenth": 9,
		"other": 1,
	}
	parts := strings.SplitN(target, " ", 2)
	if len(parts) < 2 {
		return target, 0
	}
	first := strings.ToLower(parts[0])
	// Check word ordinals
	if ord, ok := ordinalWords[first]; ok {
		return parts[1], ord
	}
	// Check numeric: "2 gate" means 2nd (index 1)
	if num, err := strconv.Atoi(first); err == nil && num >= 1 {
		return parts[1], num - 1
	}
	// Check trailing number: "counter 2" means 2nd counter
	lastSpace := strings.LastIndex(target, " ")
	if lastSpace > 0 {
		last := target[lastSpace+1:]
		if num, err := strconv.Atoi(last); err == nil && num >= 1 {
			return target[:lastSpace], num - 1
		}
	}
	return target, 0
}

// matchesTargetOrdinal checks if a target matches, accounting for ordinal prefixes.
// It returns true and decrements *skip if matching. When *skip reaches 0, it's the right one.
func matchesTargetOrdinal(nounName, cleanTarget, adjName string, skip *int) bool {
	if !matchesTarget(nounName, cleanTarget, adjName) {
		return false
	}
	if *skip > 0 {
		*skip--
		return false
	}
	return true
}

func matchesTarget(nounName, target, adjName string) bool {
	t := strings.ToLower(target)
	n := strings.ToLower(nounName)
	a := strings.ToLower(adjName)

	if t == n {
		return true
	}
	if a != "" && t == a+" "+n {
		return true
	}
	// Partial match (prefix)
	if strings.HasPrefix(n, t) {
		return true
	}
	// Match last word of noun (e.g., "tooth" matches "rat tooth")
	if idx := strings.LastIndex(n, " "); idx >= 0 {
		if strings.HasPrefix(n[idx+1:], t) {
			return true
		}
	}
	return false
}

func containsFlag(flags []string, flag string) bool {
	for _, f := range flags {
		if strings.EqualFold(f, flag) {
			return true
		}
	}
	return false
}

func isPortal(itemType string) bool {
	switch itemType {
	case "PORTAL", "PORTAL_THROUGH", "PORTAL_CLIMB", "PORTAL_UP", "PORTAL_DOWN",
		"PORTAL_OVER", "PORTAL_CLIMBUP", "PORTAL_CLIMBDOWN":
		return true
	}
	return false
}

func isWeapon(itemType string) bool {
	switch itemType {
	case "SLASH_WEAPON", "CRUSH_WEAPON", "PUNCTURE_WEAPON", "POLE_WEAPON",
		"TWOHAND_WEAPON", "BOW_WEAPON", "THROWN_WEAPON", "STABTHROWN",
		"POLETHROWN", "HANDGUN", "RIFLE", "CLAW_WEAPON", "BITE_WEAPON",
		"DRAKIN_CRUSH", "DRAKIN_POLE", "DRAKIN_SLASH", "DRAKIN_THROWN":
		return true
	}
	return false
}

// a way to check if an item is magical.  It basically counts POWER from the  item scripts
func (e *GameEngine) itemMagicPower(def *gameworld.ItemDef, ri *gameworld.RoomItem) int {

	power := 0

	var traitNames []string

	if def != nil {
		traitNames = append(traitNames, def.Traits...)
	}

	if ri != nil {
		traitNames = append(traitNames, ri.Traits...)
	}

	for _, traitName := range traitNames {
		trait := e.traits[strings.ToUpper(traitName)]
		if trait == nil {
			continue
		}

		if trait.Power > 0 {
			power += trait.Power
		}
	}

	return power
}

// doLoadWeapon handles NOCK/LOAD <weapon> WITH <ammo>.
// Bows/crossbows need to be loaded with arrows/bolts before firing.
func (e *GameEngine) doLoadWeapon(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Load what? Usage: NOCK <weapon> WITH <ammo>"}}
	}
	raw := strings.ToLower(strings.Join(args, " "))
	weaponTarget, ammoTarget := parseWithClause(raw)
	if ammoTarget == "" {
		return &CommandResult{Messages: []string{"Load with what? Usage: NOCK <weapon> WITH <ammo>"}}
	}

	// Find the weapon (must be wielded or in inventory)
	if player.Wielded == nil {
		return &CommandResult{Messages: []string{"You aren't wielding a ranged weapon."}}
	}
	weaponDef := e.items[player.Wielded.Archetype]
	if weaponDef == nil || (weaponDef.Type != "BOW_WEAPON" && weaponDef.Type != "HANDGUN" && weaponDef.Type != "RIFLE") {
		return &CommandResult{Messages: []string{"You aren't wielding a ranged weapon."}}
	}
	wepName := e.getItemNounName(weaponDef)
	if !strings.HasPrefix(strings.ToLower(wepName), weaponTarget) && weaponTarget != "bow" && weaponTarget != "crossbow" && weaponTarget != "gun" {
		return &CommandResult{Messages: []string{"You aren't wielding that weapon."}}
	}

	// Check if already loaded
	if player.Wielded.Val3 > 0 {
		return &CommandResult{Messages: []string{fmt.Sprintf("Your %s is already loaded.", wepName)}}
	}

	// Find ammo in inventory (must match Parameter2 ammo type on weapon)
	ammoType := weaponDef.Parameter2 // what ammo this weapon takes
	for i, ii := range player.Inventory {
		ammoDef := e.items[ii.Archetype]
		if ammoDef == nil {
			continue
		}
		ammoName := e.getItemNounName(ammoDef)
		if !strings.HasPrefix(strings.ToLower(ammoName), ammoTarget) {
			continue
		}
		// Check ammo type match
		if ammoType > 0 && ii.Archetype != ammoType {
			return &CommandResult{Messages: []string{fmt.Sprintf("You can't load your %s with that.", wepName)}}
		}
		// Load the weapon: set Val3 > 0 to indicate loaded
		player.Wielded.Val3 = ii.Archetype
		// Remove one ammo from inventory (or reduce bundle count)
		player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
		e.SavePlayer(ctx, player)

		ammoDisplayName := e.formatItemName(ammoDef, ii.Adj1, ii.Adj2, ii.Adj3)
		wepDisplayName := e.formatItemName(weaponDef, player.Wielded.Adj1, player.Wielded.Adj2, player.Wielded.Adj3)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You load your %s with %s.", wepDisplayName, ammoDisplayName)},
			RoomBroadcast: []string{fmt.Sprintf("%s loads %s %s.", player.FirstName, player.Possessive(), wepDisplayName)},
		}
	}

	return &CommandResult{Messages: []string{fmt.Sprintf("You don't have any '%s' to load.", ammoTarget)}}
}

// GenerateAPIKey creates a new API key for a character. Returns the raw key (shown once).
func (e *GameEngine) GenerateAPIKey(ctx context.Context, firstName, accountID string, allowGM bool) (string, error) {
	if e.db == nil {
		return "", fmt.Errorf("no database")
	}
	// Generate 32 random bytes → 64 hex chars with prefix
	raw := make([]byte, 32)
	if _, err := cryptorand.Read(raw); err != nil {
		return "", err
	}
	key := "lofp_" + hex.EncodeToString(raw)
	prefix := key[:13] // "lofp_" + first 8 hex chars

	// Hash the key for storage
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

func isSelfOr(isSelf bool, selfText, otherText string) string {
	if isSelf {
		return selfText
	}
	return otherText
}

func genderName(g int) string {
	switch g {
	case 0:
		return "Male"
	case 1:
		return "Female"
	default:
		return "Unknown"
	}
}

// organizationName returns the display name for an organization number.
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
	// Walk XP through levels to find leftover in current level
	bp := 30
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
			// Partial progress within level — find XP to next BP
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
/*
func playerLoadWeight(player *Player, items map[int]*gameworld.ItemDef) int {
	total := 0
	for _, ii := range player.Inventory {
		if def := items[ii.Archetype]; def != nil {
			total += def.Weight
		}
	}
	return total
}
*/

func inventoryItemWeight(item InventoryItem, items map[int]*gameworld.ItemDef) int {
	total := 0

	if def := items[item.Archetype]; def != nil {
		total += def.Weight
	}

	for _, child := range item.Contents {
		total += inventoryItemWeight(child, items)
	}

	return total
}

func playerLoadWeight(player *Player, items map[int]*gameworld.ItemDef) int {

	total := 0

	for _, item := range player.Inventory {
		total += inventoryItemWeight(item, items)
	}

	return total
}

// doSkin handles the SKIN command — skin a dead monster for components.
func (e *GameEngine) doSkin(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Skin what?"}}
	}

	rawTarget := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(rawTarget)

	if e.monsterMgr == nil {
		return &CommandResult{Messages: []string{"You don't see that here."}}
	}

	matchCount := 0
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

		if !strings.HasPrefix(name, target) &&
			!strings.HasPrefix(noun, target) {
			continue
		}

		matchCount++
		if matchCount <= ordSkip {
			continue
		}

		if def.Discorporate {
			return &CommandResult{
				Messages: []string{"There is nothing left to skin."},
			}
		}

		if inst.Skinned {
			return &CommandResult{
				Messages: []string{"This corpse has already been skinned."},
			}
		}

		// Check for skin items
		if len(def.SkinItems) == 0 && def.SkinAdj == 0 {
			return &CommandResult{Messages: []string{fmt.Sprintf("You can't skin a %s.", def.Name)}}
		}

		// Weighted random skin selection
		displayName := FormatMonsterName(def, e.monAdjs)
		var skinMsgs []string

		if len(def.SkinItems) > 0 {
			// Sum probabilities for weighted selection
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

		e.monsterMgr.MarkSkinned(inst.ID)
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      skinMsgs,
			RoomBroadcast: []string{fmt.Sprintf("%s skins %s %s.", player.FirstName, articleFor(displayName, def.Unique), displayName)},
		}
	}

	return &CommandResult{Messages: []string{"You don't see a dead creature to skin here."}}
}

// ---- TEACH command ----

func (e *GameEngine) doTeach(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		// Stop teaching
		player.Teaching = 0
		player.TeachingLevel = 0
		return &CommandResult{Messages: []string{"You stop teaching."}}
	}
	skillNum, err := strconv.Atoi(args[0])
	if err != nil || skillNum < 0 {
		// Try matching by skill name
		target := strings.ToLower(strings.Join(args, " "))
		for id, name := range SkillNames {
			if strings.HasPrefix(strings.ToLower(name), target) {
				skillNum = id
				err = nil
				break
			}
		}
		if err != nil {
			return &CommandResult{Messages: []string{"Unknown skill."}}
		}
	}
	if skillNum == 0 {
		player.Teaching = 0
		player.TeachingLevel = 0
		return &CommandResult{Messages: []string{"You stop teaching."}}
	}
	name, ok := SkillNames[skillNum]
	if !ok {
		return &CommandResult{Messages: []string{"Unknown skill."}}
	}
	teacherLevel := player.Skills[skillNum]
	if teacherLevel < 1 {
		return &CommandResult{Messages: []string{"You don't have any training in that skill."}}
	}
	player.Teaching = skillNum
	player.TeachingLevel = teacherLevel
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You are now teaching %s (skill #%d, up to level %d).", name, skillNum, teacherLevel)},
		RoomBroadcast: []string{fmt.Sprintf("%s begins teaching %s.", player.FirstName, name)},
	}
}

// ---- FILL command ----

func (e *GameEngine) doFill(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"You don't have anything to fill."}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	// Find the container in player inventory (glass, cup, flask, mug, bottle, tankard, vial)
	fillableNouns := map[string]bool{"glass": true, "cup": true, "flask": true, "mug": true, "bottle": true, "tankard": true, "vial": true, "goblet": true, "chalice": true, "stein": true}
	var fillIdx int = -1
	var fillItem *InventoryItem
	var fillDef *gameworld.ItemDef
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
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
		// Check if it's a fillable container type
		if itemDef.Type == "LIQCONTAINER" || fillableNouns[strings.ToLower(name)] {
			fillIdx = i
			fillItem = &player.Inventory[i]
			fillDef = itemDef
			break
		}
	}
	if fillIdx < 0 || fillItem == nil {
		return &CommandResult{Messages: []string{"You don't have anything to fill."}}
	}

	// Find a source in the room (keg, barrel, fountain, well, spring, cauldron)
	sourceNouns := map[string]bool{"keg": true, "barrel": true, "fountain": true, "well": true, "spring": true, "cauldron": true, "cask": true, "tap": true, "spigot": true}
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"There is nothing to fill from here."}}
	}
	var sourceDef *gameworld.ItemDef
	var sourceRI *gameworld.RoomItem
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := strings.ToLower(e.getItemNounName(itemDef))
		if sourceNouns[name] || itemDef.Type == "LIQUID" || itemDef.Type == "LIQCONTAINER" {
			sourceDef = itemDef
			sourceRI = &room.Items[i]
			break
		}
	}
	if sourceDef == nil || sourceRI == nil {
		return &CommandResult{Messages: []string{"There is nothing to fill from here."}}
	}

	// Determine the drink type from the source
	drinkType := "water"
	sourceName := strings.ToLower(e.getItemNounName(sourceDef))
	if sourceRI.Extend != "" {
		drinkType = strings.ToLower(sourceRI.Extend)
	} else if strings.Contains(sourceName, "keg") || strings.Contains(sourceName, "barrel") || strings.Contains(sourceName, "cask") {
		drinkType = "ale"
	} else if strings.Contains(sourceName, "cauldron") {
		drinkType = "broth"
	}

	// Fill the container — set Val2 to a default number of sips
	fillItem.Val2 = 5
	fillItem.State = "filled"
	e.SavePlayer(ctx, player)

	displayName := e.formatItemNameNoArticle(fillDef, fillItem.Adj1, fillItem.Adj2, fillItem.Adj3)
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You fill the %s with some %s.", displayName, drinkType)},
		RoomBroadcast: []string{fmt.Sprintf("%s fills a %s with some %s.", player.FirstName, displayName, drinkType)},
	}
}

// ---- DISARM command ----

func (e *GameEngine) doDisarm(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Disarm what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}

	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}

		// Check if item has a trap (Val4 > 0)
		if ri.Val4 == 0 {
			return &CommandResult{Messages: []string{"That doesn't appear to be trapped."}}
		}

		// Requires Trap & Poison Lore (skill #12)
		trapSkill := player.Skills[12]
		if trapSkill < 1 {
			return &CommandResult{Messages: []string{"You have no training in Trap & Poison Lore."}}
		}

		// Skill check: base 20% + skill_level * 5%, capped at 95%
		successChance := 20 + trapSkill*5
		if successChance > 95 {
			successChance = 95
		}
		roll := rand.Intn(100) + 1

		// Apply round time (5 seconds)
		player.RoundTimeExpiry = time.Now().Add(5 * time.Second)

		if roll <= successChance {
			// Success — remove the trap
			room.Items[i].Val4 = 0
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_update", ItemRef: ri.Ref, Item: &room.Items[i]})
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("[Success: %d%%, Roll: %d] Success!", successChance, roll), "You carefully disarm the trap."},
				RoomBroadcast: []string{fmt.Sprintf("%s carefully disarms a trap.", player.FirstName)},
			}
		}

		// Failure — optionally trigger the trap
		msgs := []string{fmt.Sprintf("[Success: %d%%, Roll: %d] Failure.", successChance, roll), "You are unable to disarm the trap."}
		// Critical failure (roll > 90): trigger the trap
		if roll > 90 {
			trapMsgs := e.checkTrap(player, &room.Items[i])
			if len(trapMsgs) > 0 {
				msgs = append(msgs, trapMsgs...)
				e.SavePlayer(ctx, player)
				e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_update", ItemRef: ri.Ref, Item: &room.Items[i]})
			}
		}
		return &CommandResult{Messages: msgs}
	}
	return &CommandResult{Messages: []string{"You don't see that here."}}
}

// ---- TURN command (book page-turning) ----

func (e *GameEngine) doTurnPage(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return nil // fall through to item interaction ("Turn what?")
	}
	target := strings.ToLower(strings.Join(args, " "))
	target, ordSkip := parseOrdinal(target)
	skip := ordSkip

	// Check if the target word is "page" — shorthand for turning whatever book is around
	isPageKeyword := (target == "page")

	room := e.rooms[player.RoomNumber]

	// Search room items for a book (Val2 > 0 indicates total pages)
	if room != nil {
		for i, ri := range room.Items {
			itemDef := e.items[ri.Archetype]
			if itemDef == nil {
				continue
			}
			name := e.getItemNounName(itemDef)
			if isPageKeyword {
				// "turn page" — match any book in the room
				if ri.Val2 <= 0 {
					continue
				}
			} else {
				if !matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
					continue
				}
			}
			if skip > 0 {
				skip--
				continue
			}
			// Check if it's a book (has Val2 = total pages > 0)
			if ri.Val2 <= 0 {
				return nil // not a book, fall through to normal item interaction
			}
			// Increment page, wrap around
			currentPage := ri.Val1
			totalPages := ri.Val2
			currentPage++
			if currentPage > totalPages {
				currentPage = 1
			}
			room.Items[i].Val1 = currentPage
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_update", ItemRef: ri.Ref, Item: &room.Items[i]})
			return &CommandResult{
				Messages:      []string{"You carefully turn the page."},
				RoomBroadcast: []string{fmt.Sprintf("%s turns a page.", player.FirstName)},
			}
		}
	}

	// Search player inventory
	skip = ordSkip
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if isPageKeyword {
			if ii.Val2 <= 0 {
				continue
			}
		} else {
			if !matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
				continue
			}
		}
		if skip > 0 {
			skip--
			continue
		}
		if ii.Val2 <= 0 {
			return nil // not a book, fall through
		}
		currentPage := ii.Val1
		totalPages := ii.Val2
		currentPage++
		if currentPage > totalPages {
			currentPage = 1
		}
		player.Inventory[i].Val1 = currentPage
		e.SavePlayer(ctx, player)
		return &CommandResult{
			Messages:      []string{"You carefully turn the page."},
			RoomBroadcast: []string{fmt.Sprintf("%s turns a page.", player.FirstName)},
		}
	}

	if isPageKeyword {
		return &CommandResult{Messages: []string{"You can't turn pages on that."}}
	}

	return nil // fall through to item interaction
}

func rollStatsForRace(race int) PendingStats {
	ranges := RaceStatRanges[race]

	rollStat := func(idx int) int {
		r := ranges[idx]
		return r[0] + rand.Intn(r[1]-r[0]+1)
	}

	return PendingStats{
		Strength:     rollStat(0),
		Agility:      rollStat(1),
		Quickness:    rollStat(2),
		Constitution: rollStat(3),
		Perception:   rollStat(4),
		Willpower:    rollStat(5),
		Empathy:      rollStat(6),
	}
}

func (e *GameEngine) doReports(ctx context.Context, player *Player, args []string) *CommandResult {

	if !player.IsGM {
		return &CommandResult{
			Messages:    []string{"You are not authorized to view reports."},
			PlayerState: player,
		}
	}

	if e.db == nil {
		return &CommandResult{
			Messages:    []string{"Database is unavailable."},
			PlayerState: player,
		}
	}

	limit := int64(25)

	// Optional: REPORTS 50
	if len(args) > 0 {
		if n, err := strconv.Atoi(args[0]); err == nil && n > 0 {
			if n > 100 {
				n = 100
			}
			limit = int64(n)
		}
	}

	coll := e.db.Collection("game_logs")

	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetLimit(limit)

	cursor, err := coll.Find(
		ctx,
		bson.M{"event": "report"},
		opts,
	)

	if err != nil {
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("Unable to retrieve reports: %v", err),
			},
			PlayerState: player,
		}
	}
	defer cursor.Close(ctx)

	type reportLog struct {
		ID        bson.ObjectID `bson:"_id"`
		Timestamp time.Time     `bson:"timestamp"`
		Player    string        `bson:"player"`
		Details   string        `bson:"details"`
		RoomNum   int           `bson:"roomNum"`
	}

	var reports []reportLog

	if err := cursor.All(ctx, &reports); err != nil {
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("Unable to read reports: %v", err),
			},
			PlayerState: player,
		}
	}

	if len(reports) == 0 {
		return &CommandResult{
			Messages:    []string{"No player reports found."},
			PlayerState: player,
		}
	}

	msgs := []string{
		fmt.Sprintf("=== Player Reports (%d) ===", len(reports)),
	}

	for _, r := range reports {
		msgs = append(
			msgs,
			fmt.Sprintf(
				"%s | %s | Room %d | ID: %s",
				r.Timestamp.Local().Format("2006-01-02 15:04"),
				r.Player,
				r.RoomNum,
				r.ID.Hex(),
			),
			fmt.Sprintf("  %s", r.Details),
		)
	}

	return &CommandResult{
		Messages:    msgs,
		PlayerState: player,
	}
}

func (e *GameEngine) doReportComplete(ctx context.Context, player *Player, args []string) *CommandResult {

	if !player.IsGM {
		return &CommandResult{
			Messages:    []string{"You are not authorized to complete reports."},
			PlayerState: player,
		}
	}

	if len(args) == 0 {
		return &CommandResult{
			Messages:    []string{"Usage: REPORTCOMPLETE <report id>"},
			PlayerState: player,
		}
	}

	if e.db == nil {
		return &CommandResult{
			Messages:    []string{"Database is unavailable."},
			PlayerState: player,
		}
	}

	id, err := bson.ObjectIDFromHex(args[0])
	if err != nil {
		return &CommandResult{
			Messages:    []string{"Invalid report ID."},
			PlayerState: player,
		}
	}

	result, err := e.db.Collection("game_logs").UpdateOne(
		ctx,
		bson.M{
			"_id":   id,
			"event": "report",
		},
		bson.M{
			"$set": bson.M{
				"event": "reportcomplete",
			},
		},
	)

	if err != nil {
		return &CommandResult{
			Messages: []string{
				fmt.Sprintf("Unable to complete report: %v", err),
			},
			PlayerState: player,
		}
	}

	if result.MatchedCount == 0 {
		return &CommandResult{
			Messages:    []string{"Report not found."},
			PlayerState: player,
		}
	}

	return &CommandResult{
		Messages:    []string{"Report marked complete."},
		PlayerState: player,
	}
}

func (e *GameEngine) RefreshEncumbrance(player *Player) {
	// Remove previous encumbrance effects.
	active := player.ActiveStatEffects[:0]

	for _, effect := range player.ActiveStatEffects {
		if effect.Source != EffectSourceEncumbrance {
			active = append(active, effect)
		}
	}

	player.ActiveStatEffects = active

	load := playerLoadWeight(player, e.items)
	strength := player.EffectiveStat(StatStrength)

	penalty := 0

	switch {
	case load > strength*5/2:
		penalty = -50

	case load > strength*2:
		penalty = -25

	case load > strength+strength/2:
		penalty = -10
	}

	if penalty == 0 {
		return
	}

	player.ActiveStatEffects = append(
		player.ActiveStatEffects,
		StatEffect{
			Source:    EffectSourceEncumbrance,
			Stat:      StatAgility,
			Modifier:  penalty,
			Permanent: true,
		},
		StatEffect{
			Source:    EffectSourceEncumbrance,
			Stat:      StatQuickness,
			Modifier:  penalty,
			Permanent: true, // Encumbrance effects do not expire on their own
		},
	)
}

func (e *GameEngine) roomHasLightSource(roomNum int) bool {
	if e.sessions == nil {
		return false
	}

	now := time.Now()

	for _, p := range e.sessions.OnlinePlayers() {
		if p.RoomNumber != roomNum {
			continue
		}

		// Magical Light spell
		for _, effect := range p.ActiveStatEffects {
			if effect.Stat == LightBuff &&
				(effect.Permanent || effect.ExpiresAt.After(now)) {
				return true
			}
		}

		// Physical light sources carried by players
		for _, item := range p.Inventory {
			def := e.items[item.Archetype]
			if def == nil {
				continue
			}

			if containsFlag(def.Flags, "LIGHTABLE") &&
				item.State == "LIT" {
				return true
			}
		}
	}

	// Light sources lying in the room.
	room := e.rooms[roomNum]
	if room != nil {
		for _, item := range room.Items {
			def := e.items[item.Archetype]
			if def == nil {
				continue
			}

			if containsFlag(def.Flags, "LIGHTABLE") &&
				item.State == "LIT" {
				return true
			}
		}
	}

	return false
}

func playerCanSeeInDark(player *Player) bool {
	switch player.Race {
	case RaceMurg, RaceHighlander:
		return true
	default:
		return false
	}
}

func (e *GameEngine) canPlayerSee(player *Player, room *gameworld.Room) bool {
	if room == nil {
		return false
	}
	return e.roomVisibility(player, room) != VisibilityDark
}

func (e *GameEngine) roomVisibility(player *Player, room *gameworld.Room) VisibilityLevel {
	if room == nil {
		return VisibilityDark
	}

	// Racial dark vision
	if playerCanSeeInDark(player) {
		return VisibilityClear
	}

	// Personal night vision  PARTIAL_DARKNESS
	if player != nil && player.HasStatEffect(NightVisionBuff) {
		return VisibilityClear
	}

	// Any active light source in the room illuminates it for everyone.
	if e.roomHasLightSource(room.Number) {
		return VisibilityClear
	}

	switch room.Lighting {
	case "FIXED_LIGHT":
		return VisibilityClear

	case "DARKNESS":
		return VisibilityDark

	case "DAY_LIGHT", "OBSCURED_DAY_LIGHT":
		if IsDay() {
			return VisibilityClear
		}
		return VisibilityDark

	case "PARTIAL_DARKNESS":
		return VisibilityPartial

	case "LIMITED_VISION":
		return VisibilityLimited

	default:
		return VisibilityClear
	}
}

func (p *Player) HasStatEffect(stat StatID) bool {
	now := time.Now()

	for _, effect := range p.ActiveStatEffects {
		if effect.Stat != stat {
			continue
		}

		if effect.Permanent || effect.ExpiresAt.After(now) {
			return true
		}
	}

	return false
}

func (e *GameEngine) visibilityCombatModifier(player *Player, RoomNumber int) int {

	room := e.rooms[RoomNumber]

	switch e.roomVisibility(player, room) {
	case VisibilityDark:
		return -40

	case VisibilityPartial:
		return -20

	case VisibilityLimited:
		return -10

	default:
		return 0
	}
}

func (e *GameEngine) visibilityStatus(player *Player, room *gameworld.Room) string {
	if room == nil {
		return ""
	}

	mod := e.visibilityCombatModifier(player, room.Number)

	switch e.roomVisibility(player, room) {
	case VisibilityDark:
		return fmt.Sprintf(
			"Visibility: Darkness (%d attack, %d defense)",
			mod, mod,
		)

	case VisibilityPartial:
		return fmt.Sprintf(
			"Visibility: Partial darkness (%d attack, %d defense)",
			mod, mod,
		)

	case VisibilityLimited:
		return fmt.Sprintf(
			"Visibility: Limited vision (%d attack, %d defense)",
			mod, mod,
		)

	default:
		return ""
	}
}
