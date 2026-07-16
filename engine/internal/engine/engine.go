package engine

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jonradoff/lofp/internal/gameworld"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

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

// RoomBroadcastExcludeFunc sends messages to all players in a room except the named player.
// Used when a scheduled script segment already delivered the same text directly to that
// player (e.g. ECHO ALL, which populates both the direct-message and room-broadcast lists).
type RoomBroadcastExcludeFunc func(roomNumber int, excludeName string, messages []string)

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
	breakMods            map[int]int // adjective ID -> weapon hardness modifier (BREAKMOD)
	items                map[int]*gameworld.ItemDef
	rooms                map[int]*gameworld.Room
	monsters             map[int]*gameworld.MonsterDef
	startRoom            int
	departRoom           int // safe room for DEPART (bump room)
	sessions             SessionProvider
	onRoomChange         RoomChangeCallback
	roomBroadcast        RoomBroadcastFunc
	roomBroadcastExclude RoomBroadcastExcludeFunc
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
	macros               map[int][]gameworld.ScriptBlock // macro# -> scripts, for inline CALL N actions
	forageDefs           []gameworld.ForageDef
	regions              map[int]*gameworld.Region
	PVals                map[int]int               // persistent global values
	NamedVars            map[string]int            // VARIABLE-defined global named variables (DANWATER, etc.)
	namedVarNames        map[string]bool           // set of valid named variable names
	orgDefs              map[int]*gameworld.OrgDef // org# -> OrgDef
	Events               *EventBus
	Banner               string // active login banner; in-memory so it works even if MongoDB is down
	lastAssistName       string // last player who used ASSIST (for @answer)
	lastAssistRoom       int    // room number of last ASSIST
	// roomContainerContents stores the contents of room-level containers (transient).
	// Key: "<roomNumber>:<itemRef>"
	roomContainerContents map[string][]InventoryItem
	watchMu               sync.RWMutex
	watching              map[string]int // playerFirstName → roomNum their familiar is watching (0 = not watching)
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

// SetRoomBroadcastExclude sets the function used to send messages to a room while
// excluding one player by name (used by scheduled script segments — see RoomBroadcastExcludeFunc).
func (e *GameEngine) SetRoomBroadcastExclude(fn RoomBroadcastExcludeFunc) {
	e.roomBroadcastExclude = fn
}

// SetLocalRoomBroadcast sets a local-only broadcast (no hub). Used for monster activity.
// The broadcast is wrapped to also forward messages to watchers (familiar WATCH WILL mode).
func (e *GameEngine) SetLocalRoomBroadcast(fn LocalRoomBroadcastFunc) {
	e.localRoomBroadcast = func(roomNum int, messages []string) {
		fn(roomNum, messages)
		e.forwardToWatchers(roomNum, messages)
	}
}

// setWatching registers or clears a player's familiar watch on a room.
// roomNum == 0 clears the watch. Safe to call from any goroutine.
func (e *GameEngine) setWatching(playerName string, roomNum int) {
	e.watchMu.Lock()
	if e.watching == nil {
		e.watching = make(map[string]int)
	}
	if roomNum == 0 {
		delete(e.watching, playerName)
	} else {
		e.watching[playerName] = roomNum
	}
	e.watchMu.Unlock()
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
		breakMods:  make(map[int]int),
		items:      make(map[int]*gameworld.ItemDef),
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
	for i := range parsed.BreakMods {
		e.breakMods[parsed.BreakMods[i].AdjID] = parsed.BreakMods[i].Modifier
	}
	for i := range parsed.Items {
		e.items[parsed.Items[i].Number] = &parsed.Items[i]
	}
	for i := range parsed.Rooms {
		e.rooms[parsed.Rooms[i].Number] = &parsed.Rooms[i]
	}
	for i := range parsed.Monsters {
		e.monsters[parsed.Monsters[i].Number] = &parsed.Monsters[i]
	}

	// Load forage definitions
	e.forageDefs = parsed.ForageDefs

	// Build region lookup map
	e.regions = make(map[int]*gameworld.Region)
	for i := range parsed.Regions {
		reg := &parsed.Regions[i]
		e.regions[reg.ID] = reg
	}
	log.Printf("Loaded %d regions", len(e.regions))

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

	// Store macros for inline CALL N actions (subroutine-style invocation from within
	// an already-running script, as opposed to CALL/SCRIPTMACRO at room/item/monster
	// definition time, which is resolved statically into Scripts at parse time).
	e.macros = make(map[int][]gameworld.ScriptBlock, len(parsed.Macros))
	for _, m := range parsed.Macros {
		e.macros[m.ID] = m.Scripts
	}

	// Load organization definitions
	e.orgDefs = make(map[int]*gameworld.OrgDef)
	for i := range parsed.OrgDefs {
		def := &parsed.OrgDefs[i]
		e.orgDefs[def.Number] = def
	}
	log.Printf("Loaded %d organization definitions", len(e.orgDefs))

	// Initialize PVals
	e.PVals = make(map[int]int)
	e.loadPVals()

	e.initContainerMap()

	// Apply content patches for planned-but-commented-out script content.
	e.applyContentPatches()

	return e
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

	// "." repeats the last command entered.
	if input == "." {
		if player.LastCommand == "" {
			return &CommandResult{Messages: []string{"You haven't entered a command yet."}}
		}
		input = player.LastCommand
	} else {
		player.LastCommand = input
	}

	// Being carried is passive — any action the carried player takes breaks it.
	if player.CarriedBy != "" {
		e.breakCarryAsCarried(ctx, player, fmt.Sprintf("%s stirs and slips from your grip!", player.FirstName))
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

	isQuoteSpeech := strings.HasPrefix(input, "'") || strings.HasPrefix(input, "\"")
	isSayVerb := len(input) >= 3 && strings.EqualFold(input[:3], "say") && (len(input) == 3 || input[3] == ' ')

	// Dead players can only DEPART, LOOK, WHO, QUIT, EXP, STATUS, HEALTH — unless
	// Speak with Dead (spell 311) has granted them the power of speech.
	if player.Dead {
		isSpeech := isQuoteSpeech || isSayVerb
		if !(isSpeech && player.SpeakWhileDead) {
			verb := strings.ToUpper(strings.Fields(input)[0])
			switch verb {
			case "DEPART", "LOOK", "WHO", "QUIT", "EXP", "EXPERIENCE", "STATUS", "HEALTH", "HELP":
				// allowed — fall through to normal processing
			default:
				return &CommandResult{Messages: []string{"You are dead and can't do much of anything. Type DEPART to allow Eternity, Inc. to retrieve you."}}
			}
		}
	}

	// Handle speech: '<msg>, "<msg>, or SAY <msg>
	if isQuoteSpeech || isSayVerb {
		var msg string
		if isQuoteSpeech {
			msg = input[1:]
		} else {
			msg = strings.TrimSpace(input[3:])
			if msg == "" {
				return &CommandResult{Messages: []string{"Say what?"}}
			}
		}
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
				if sc.MoveTo > 0 {
					e.applySayMove(ctx, player, sc, result)
				}
				e.SavePlayer(ctx, player)
				// PLREVENT/CONTPLREVENT-deferred actions (e.g. a multi-part scripted
				// reply) must be scheduled, or everything after the delay is lost.
				if len(sc.DeferredSegments) > 0 {
					e.scheduleScriptSegments(player, sc.DeferredSegments)
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
			if sc.MoveTo > 0 {
				e.applySayMove(ctx, player, sc, result)
			}
			e.SavePlayer(ctx, player)
			// PLREVENT/CONTPLREVENT-deferred actions (e.g. a multi-part scripted
			// reply) must be scheduled, or everything after the delay is lost.
			if len(sc.DeferredSegments) > 0 {
				e.scheduleScriptSegments(player, sc.DeferredSegments)
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

	// Resolve verb abbreviations to canonical full form for ALL verbs.
	// This ensures IFPREVERB matching works regardless of abbreviation used.
	// Direction shortcuts map to both full name (for scripts) and short form (for movement).
	dirFullNames := map[string]string{
		"N": "NORTH", "S": "SOUTH", "E": "EAST", "W": "WEST",
		"NE": "NORTHEAST", "NW": "NORTHWEST", "SE": "SOUTHEAST", "SW": "SOUTHWEST",
		"U": "UP", "D": "DOWN", "O": "OUT",
	}
	dirMap := map[string]string{
		"N": "N", "NORTH": "N", "S": "S", "SOUTH": "S",
		"E": "E", "EAST": "E", "W": "W", "WEST": "W",
		"NE": "NE", "NORTHEAST": "NE", "NW": "NW", "NORTHWEST": "NW",
		"SE": "SE", "SOUTHEAST": "SE", "SW": "SW", "SOUTHWEST": "SW",
		"U": "U", "UP": "U", "D": "D", "DOWN": "D",
		"O": "O", "OUT": "O",
	}
	// Resolve the verb to its full canonical name for script matching
	canonicalVerb := verb
	if full, ok := dirFullNames[verb]; ok {
		canonicalVerb = full
	}

	if dir, ok := dirMap[verb]; ok {
		// Roundtime check BEFORE scripts: scripts may set roundtime as a post-move
		// penalty (EQUAL ROUNDTIME without CLEARVERB), so we must check here rather
		// than relying on doMove's check (which would see the newly-set expiry and block).
		if player.RoundTimeExpiry.After(time.Now()) {
			remaining := int(player.RoundTimeExpiry.Sub(time.Now()).Seconds()) + 1
			return &CommandResult{Messages: []string{fmt.Sprintf("[Wait %d seconds...]", remaining)}}
		}
		// Check for IFPREVERB scripts on the direction using canonical verb name
		room := e.rooms[player.RoomNumber]
		if room != nil {
			origRoom := player.RoomNumber // capture before scripts may MOVE the player
			sc := &ScriptContext{Player: player, Room: room, Engine: e}
			for _, block := range room.Scripts {
				if block.Type == "IFPREVERB" && len(block.Args) >= 2 {
					if strings.ToUpper(block.Args[0]) == canonicalVerb && block.Args[1] == "-1" {
						sc.execBlock(block)
					}
				}
			}
			// MOVEGROUP: move all players in this room to destination
			if sc.MoveGroupTo > 0 {
				e.moveGroupToRoom(ctx, player.RoomNumber, sc.MoveGroupTo)
			}
			if sc.Blocked || sc.MoveTo > 0 {
				result := &CommandResult{}
				result.Messages = append(result.Messages, sc.Messages...)
				result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
				if sc.MoveTo > 0 && !sc.Blocked {
					// Script provided a MOVE destination
					dest := e.rooms[sc.MoveTo]
					if dest != nil {
						player.RoomNumber = sc.MoveTo
						e.SavePlayer(ctx, player)
						lookResult := e.doLook(player)
						result.Messages = append(result.Messages, lookResult.Messages...)
						result.RoomName = lookResult.RoomName
						result.RoomDesc = lookResult.RoomDesc
						result.Exits = lookResult.Exits
						result.Items = lookResult.Items
						result.OldRoom = origRoom
						if len(sc.PreMoveMsgs) > 0 {
							result.OldRoomMsg = sc.PreMoveMsgs
						} else {
							result.OldRoomMsg = []string{fmt.Sprintf("%s leaves.", player.FirstName)}
						}
						result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s arrives.", player.FirstName))
						e.applyEntryScripts(ctx, player, dest, result)
					}
				} else if sc.MoveTo > 0 {
					// CLEARVERB + MOVE = script-controlled movement
					dest := e.rooms[sc.MoveTo]
					if dest != nil {
						player.RoomNumber = sc.MoveTo
						e.SavePlayer(ctx, player)
						lookResult := e.doLook(player)
						result.Messages = append(result.Messages, lookResult.Messages...)
						result.RoomName = lookResult.RoomName
						result.RoomDesc = lookResult.RoomDesc
						result.Exits = lookResult.Exits
						result.Items = lookResult.Items
						result.OldRoom = origRoom
						if len(sc.PreMoveMsgs) > 0 {
							result.OldRoomMsg = sc.PreMoveMsgs
						} else {
							result.OldRoomMsg = []string{fmt.Sprintf("%s leaves.", player.FirstName)}
						}
						result.RoomBroadcast = append(result.RoomBroadcast, fmt.Sprintf("%s arrives.", player.FirstName))
						e.applyEntryScripts(ctx, player, dest, result)
					}
				}
				if len(result.Messages) == 0 {
					result.Messages = []string{"You can't go that way."}
				}
				return result
			}
			// Scripts ran but didn't block — movement proceeds. If scripts set a
			// roundtime (post-move penalty), temporarily clear it so doMove's own
			// roundtime check doesn't block the move, then restore it after.
			postMoveExpiry := player.RoundTimeExpiry
			postMoveRT := sc.RoundTimeSet
			if postMoveRT > 0 {
				player.RoundTimeExpiry = time.Time{}
			}
			var moveResult *CommandResult
			if len(sc.Messages) > 0 {
				moveResult = e.doMove(ctx, player, dir)
				moveResult.Messages = append(sc.Messages, moveResult.Messages...)
			} else {
				moveResult = e.doMove(ctx, player, dir)
			}
			if postMoveRT > 0 && postMoveExpiry.After(time.Now()) {
				player.RoundTimeExpiry = postMoveExpiry
				player.RoundTime = postMoveRT
				e.SavePlayer(ctx, player)
				if !messagesHaveRoundTime(moveResult.Messages) {
					moveResult.Messages = append(moveResult.Messages, fmt.Sprintf("[Round: %d sec]", postMoveRT))
				}
			}
			return moveResult
		}
		return e.doMove(ctx, player, dir)
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
		return e.doGetEnhanced(ctx, player, verb, args)
	case "DROP":
		return e.doDrop(ctx, player, args)
	case "INVENTORY":
		return e.doInventoryEnhanced(player)
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
		return e.doWield(ctx, player, args)
	case "UNWIELD":
		return e.doUnwield(ctx, player, args)
	case "WEAR":
		return e.doWear(ctx, player, args)
	case "REMOVE":
		return e.doRemove(ctx, player, args)
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
		if len(args) > 0 {
			return e.doLayCarried(ctx, player, args)
		}
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
		skillMsgs = append(skillMsgs, fmt.Sprintf("%-2s %-26s%-10s", "#", "Skill", "Level"))
		skillMsgs = append(skillMsgs, fmt.Sprintf("%-2s %-26s%-10s", "--", "-----", "-----"))
		hasSkills := false
		for id := 0; id <= 35; id++ {
			lvl := player.Skills[id]
			if lvl > 0 {
				name := SkillNames[id]
				if name == "" {
					name = fmt.Sprintf("Skill #%d", id)
				}
				skillMsgs = append(skillMsgs, fmt.Sprintf("%-2d %-26s%-10d", id, name, lvl))
				hasSkills = true
			}
		}
		if !hasSkills {
			skillMsgs = append(skillMsgs, "You have no trained skills yet.")
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
		return &CommandResult{Messages: []string{
			fmt.Sprintf("Experience: %d", player.Experience),
			fmt.Sprintf("Build Points to date: %d", totalBP),
			fmt.Sprintf("Unspent Build Points: %d", player.BuildPoints),
			fmt.Sprintf("Experience Points until next Build Point: %d", xpUntilNext),
		}}
	case "INFO":
		return e.doInfo(player)
	case "TIME":
		moonPhases := []string{"new", "waxing crescent", "first quarter", "waxing gibbous", "full", "waning gibbous", "last quarter", "waning crescent"}
		greatMoon := moonPhases[GameDay()%8]
		phulcrus := moonPhases[(GameDay()+4)%8]
		return &CommandResult{Messages: []string{
			fmt.Sprintf("It is %s %d, %d (Year of the Wyrm).", GameMonthName(), GameDay()%28+1, GameYear()),
			fmt.Sprintf("It is %s. The season is %s.", TimePeriod(), SeasonName()),
			fmt.Sprintf("The Great Moon is %s and Phulcrus is %s.", greatMoon, phulcrus),
		}}
	case "PAY":
		return e.doPay(ctx, player)
	case "WHISPER":
		return e.doWhisper(player, args, input)
	case "CONTACT":
		return e.doContact(player, args, input)
	case "YELL":
		return e.doYell(player, args, input)
	case "GIVE":
		return e.doGive(ctx, player, args)
	case "EAT":
		return e.doEat(ctx, player, args)
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
			// Support \ as line break for poetry/songs, same as RECITE
			lines := strings.Split(text, "\\")
			var selfMsgs, roomMsgs []string
			for i, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if i == 0 {
					selfMsgs = append(selfMsgs, fmt.Sprintf("You %ssing, \"%s", adverb, line))
					roomMsgs = append(roomMsgs, fmt.Sprintf("%s %ssings, \"%s", player.FirstName, adverb, line))
				} else {
					selfMsgs = append(selfMsgs, fmt.Sprintf("  %s", line))
					roomMsgs = append(roomMsgs, fmt.Sprintf("  %s", line))
				}
			}
			if len(selfMsgs) > 0 {
				selfMsgs[len(selfMsgs)-1] += "\""
				roomMsgs[len(roomMsgs)-1] += "\""
			}
			return &CommandResult{Messages: selfMsgs, RoomBroadcast: roomMsgs}
		}
		return e.processEmote(player, verb, args)
	case "PLAY":
		// Let a named item's own script (e.g. item 631's IFPREVERB PLAY -1, which
		// distinguishes wielded vs not-wielded) run before falling back to the generic
		// instrument flavor text below.
		if len(args) > 0 {
			target := strings.ToLower(strings.Join(args, " "))
			room := e.rooms[player.RoomNumber]
			if result := e.runVerbScriptsForTarget(ctx, player, room, "PLAY", target); result != nil {
				return result
			}
		}
		// If wielding an instrument, produce special music output
		if player.Wielded != nil {
			wDef := e.items[player.Wielded.Archetype]
			if wDef != nil {
				wieldedNoun := strings.ToLower(e.getItemNounName(wDef))
				wieldedFullName := e.formatItemNameNoArticle(wDef, player.Wielded.Adj1, player.Wielded.Adj2, player.Wielded.Adj3, player.Wielded.Tail)
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
				wieldedFullName := e.formatItemName(wDef, player.Wielded.Adj1, player.Wielded.Adj2, player.Wielded.Adj3, player.Wielded.Tail)
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
	case "PICK":
		if len(args) > 0 {
			return e.doPickLock(ctx, player, args)
		}
		return e.doEmoteWithScripts(ctx, player, verb, args)
	case "WAVE":
		if len(args) > 0 {
			result := e.doItemInteraction(ctx, player, verb, args)
			if result != nil && len(result.Messages) > 0 && result.Messages[0] != "You don't see that here." {
				return result
			}
		}
		return e.processEmote(player, verb, args)
	case "SMILE", "BOW", "CURTSEY", "CURTSY", "NOD", "LAUGH", "CHUCKLE",
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
		"DIZZY", "BAT", "FLAIL", "WAKE", "SOB",
		// Race-specific emotes (handled by race check in processEmote)
		"FLICK", "BARE", "SPREAD", "FOLD", "SWISH",
		"RUBEARS", "PULLBEARD", "SCENT", "WHINE", "DROOP", "CHASE":
		return e.doEmoteWithScripts(ctx, player, verb, args)
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
		// Bare SEARCH: scan the area for hidden players
		searchRT := applyRoundTime(player, 5)
		player.RoundTimeExpiry = time.Now().Add(time.Duration(searchRT) * time.Second)
		msgs := []string{"You search the area.", fmt.Sprintf("[Round: %d sec]", searchRT)}
		perceptionCheck := player.Perception + player.Skills[33]*5 // Stealth skill helps detection
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
		// Run room bare-verb SEARCH scripts (e.g. IFVERB SEARCH -1 to echo hidden items)
		var roomBroadcast []string
		if room := e.rooms[player.RoomNumber]; room != nil {
			sc := e.RunRoomVerbScripts(player, room, "SEARCH")
			msgs = append(msgs, sc.Messages...)
			roomBroadcast = sc.RoomMsgs
		}
		return &CommandResult{Messages: msgs, RoomBroadcast: roomBroadcast}
	case "RUB":
		if len(args) > 0 {
			if part := strings.ToLower(args[len(args)-1]); part == "back" || part == "foot" || part == "feet" {
				return e.processRubPart(player, args[:len(args)-1], part)
			}
		}
		result := e.doItemInteraction(ctx, player, verb, args)
		if result != nil && len(result.Messages) > 0 && result.Messages[0] != "You don't see that here." {
			return result
		}
		return e.processEmote(player, verb, args)
	case "PULL", "PUSH", "TOUCH", "DIG", "USE", "THUMP":
		result := e.doItemInteraction(ctx, player, verb, args)
		// If item interaction found nothing, fall back to emote for verbs that have emote entries
		if result != nil && len(result.Messages) > 0 && result.Messages[0] != "You don't see that here." {
			return result
		}
		if verb == "THUMP" || verb == "TOUCH" {
			return e.processEmote(player, verb, args)
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
	case "CONCENTRATE":
		if len(args) > 0 {
			return e.doItemInteraction(ctx, player, verb, args)
		}
		return &CommandResult{Messages: []string{"You concentrate deeply."}}
	case "BUY", "ORDER":
		return e.doBuy(ctx, player, args)
	case "SELL":
		if len(args) >= 2 && strings.ToUpper(args[0]) == "ALL" {
			noun := strings.ToLower(strings.Join(args[1:], " "))
			return e.doSellAll(ctx, player, noun)
		}
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
	case "ENCRUST":
		return e.doEncrust(ctx, player, args)
	case "INLAY":
		return e.doInlay(ctx, player, args)
	case "INSET":
		return e.doInset(ctx, player, args)
	case "ENGRAVE":
		return e.doEngrave(ctx, player, args, input)
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
	case "ASCEND", "DESCEND":
		if player.Position != 4 {
			return &CommandResult{Messages: []string{fmt.Sprintf("You must be flying to %s.", strings.ToLower(verb))}}
		}
		exitKey := "ABOVE"
		cantMsg := "There is nowhere to ascend to from here."
		if verb == "DESCEND" {
			exitKey = "BELOW"
			cantMsg = "There is nowhere to descend to from here."
		}
		flyRoom := e.rooms[player.RoomNumber]
		if flyRoom == nil {
			return &CommandResult{Error: "You are nowhere!"}
		}
		// Run IFPREVERB ASCEND/DESCEND -1 scripts before attempting movement.
		flyOrigRoom := player.RoomNumber
		flySC := &ScriptContext{Player: player, Room: flyRoom, Engine: e}
		for _, block := range flyRoom.Scripts {
			if block.Type == "IFPREVERB" && len(block.Args) >= 2 &&
				strings.ToUpper(block.Args[0]) == verb && block.Args[1] == "-1" {
				flySC.execBlock(block)
			}
		}
		if flySC.MoveGroupTo > 0 {
			e.moveGroupToRoom(ctx, player.RoomNumber, flySC.MoveGroupTo)
		}
		if flySC.Blocked || flySC.MoveTo > 0 {
			flyResult := &CommandResult{}
			flyResult.Messages = append(flyResult.Messages, flySC.Messages...)
			flyResult.RoomBroadcast = append(flyResult.RoomBroadcast, flySC.RoomMsgs...)
			if flySC.MoveTo > 0 && !flySC.Blocked {
				if flyDest := e.rooms[flySC.MoveTo]; flyDest != nil {
					player.RoomNumber = flySC.MoveTo
					e.SavePlayer(ctx, player)
					flyLook := e.doLook(player)
					flyResult.Messages = append(flyResult.Messages, flyLook.Messages...)
					flyResult.RoomName = flyLook.RoomName
					flyResult.RoomDesc = flyLook.RoomDesc
					flyResult.Exits = flyLook.Exits
					flyResult.Items = flyLook.Items
					flyResult.OldRoom = flyOrigRoom
					flyResult.OldRoomMsg = []string{fmt.Sprintf("%s leaves.", player.FirstName)}
					flyResult.RoomBroadcast = append(flyResult.RoomBroadcast, fmt.Sprintf("%s arrives.", player.FirstName))
					e.applyEntryScripts(ctx, player, flyDest, flyResult)
				}
			}
			if len(flyResult.Messages) == 0 {
				flyResult.Messages = []string{cantMsg}
			}
			return flyResult
		}
		if _, hasAbove := flyRoom.Exits[exitKey]; !hasAbove {
			return &CommandResult{Messages: []string{cantMsg}}
		}
		return e.doMove(ctx, player, exitKey)
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
	case "DUMP":
		return e.doDump(ctx, player, args)
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
		return &CommandResult{Messages: []string{"Legends of Future Past v11.11.0"}}
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
	case "TARGET":
		return e.doTarget(player, args)
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
			e.breakCarryAsCarrier(ctx, player)
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
		return e.doAvoid(ctx, player, args)
	case "UNAVOID":
		return e.doUnavoid(ctx, player, args)
	case "ALLOW":
		return e.doAllow(ctx, player, args)
	case "UNALLOW":
		return e.doUnallow(ctx, player, args)
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
		return e.doCommand(ctx, player, args, input)
	case "APPEARANCE":
		return e.doAppearance(ctx, player, input)
	case "MASTER":
		return e.doMasterSpell(ctx, player, args)
	case "NOCK", "LOAD":
		return e.doLoadWeapon(ctx, player, args)
	case "SPECIALIZE":
		return &CommandResult{Messages: []string{"[Weapon specialization coming soon.]"}}
	// === SKILL-BASED (TODO: implement) ===
	case "DISARM":
		return e.doDisarm(ctx, player, args)
	case "STEAL", "FILCH", "ROB":
		return e.doSteal(ctx, player, args)
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
		return e.doSplit(ctx, player, args)
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
		transformRT := applyRoundTime(player, 7)
		player.RoundTimeExpiry = time.Now().Add(time.Duration(transformRT) * time.Second)
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
		e.breakCarryAsCarried(ctx, player, fmt.Sprintf("%s stops submitting and slips from your grip!", player.FirstName))
		return &CommandResult{
			Messages:      []string{"You stop submitting."},
			RoomBroadcast: []string{fmt.Sprintf("%s stops submitting.", player.FirstName)},
		}
	case "CARRY":
		return e.doCarry(ctx, player, args)
	case "RELEASE":
		return e.doRelease(ctx, player)
	case "PUTDOWN":
		return e.doReleaseCarry(ctx, player)
	case "ARREST":
		return &CommandResult{Messages: []string{"[Arrest coming soon.]"}} // TODO: lawkeeper arrest
	case "ENROLL":
		return e.doEnroll(ctx, player)
	case "INITIATE":
		return &CommandResult{Messages: []string{"Only GMs may initiate players into organizations. Ask a GM for assistance."}}
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
		return e.doPour(ctx, player, args)
	case "ACTBRIEF":
		return e.doSet(ctx, player, []string{"ACTBRIEF"})
	case "RPBRIEF":
		return e.doSet(ctx, player, []string{"RPBRIEF"})
	case "SET":
		return e.doSet(ctx, player, args)
	case "SNIFF", "SMELL":
		return e.doEmoteWithScripts(ctx, player, verb, args)
	case "LISTEN":
		return e.doEmoteWithScripts(ctx, player, "LISTEN", args)
	default:
		return &CommandResult{Messages: []string{fmt.Sprintf("I don't understand \"%s\". Type HELP for commands.", strings.ToLower(input))}}
	}
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
	"CAST", "CONCENTRATE", "BUY", "SELL", "PAY",
	"DRINK", "SIP", "LIGHT", "EXTINGUISH", "DOUSE",
	"FLIP", "LATCH", "UNLATCH",
	"DEPOSIT", "WITHDRAW", "TRAIN",
	"MINE", "FORAGE",
	"CRAFT", "FORGE", "SMELT", "WEAVE", "DYE", "BREW", "ANALYZE", "WORK", "REPAIR",
	// Movement/stealth
	"HIDE", "SNEAK", "FLY", "ASCEND", "DESCEND", "LAND",
	// Interaction
	"PUT", "PLACE", "FILL", "MARK", "UNDRESS", "SKIN",
	// Info
	"BALANCE", "SPELL", "BRIEF", "FULL", "PROMPT", "UNPROMPT", "VERSION", "CREDITS",
	// Communication
	"THINK", "CANT",
	// Combat (TODO: implement)
	"ATTACK", "KILL", "SLAY", "SMITE", "ADVANCE", "RETREAT", "GUARD", "TARGET",
	"BACKSTAB", "BITE", "AVOID", "UNAVOID", "ALLOW", "UNALLOW", "BERSERK", "FRENZY",
	"DEFENSIVE", "OFFENSIVE", "WARY", "NORMAL",
	"INVOKE", "PREPARE", "CHANT", "COMMAND", "MASTER", "APPEARANCE",
	"NOCK", "LOAD", "SPECIALIZE",
	// Skill-based (TODO: implement)
	"DISARM", "STEAL", "FILCH", "ROB", "STALK",
	"TEACH", "SELFTRAIN", "UNLEARN", "LEARN",
	"ANOINT", "POISON", "TRAP",
	"SURVEY", "SPLIT",
	// Racial (TODO: implement)
	"BLEND", "CALL", "TRANSFORM", "MOLD",
	"DISGUISE", "SUBMIT", "UNSUBMIT", "ARREST", "CARRY", "RELEASE", "PUTDOWN",
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
	"DIZZY", "BAT", "FLAIL", "WAKE", "SOB",
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
