package engine

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/jonradoff/lofp/internal/gameworld"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var validNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z'-]{0,19}$`)

// ValidateCharacterInput checks character creation parameters.
func ValidateCharacterInput(firstName, lastName string, race, gender int) error {
	if !validNamePattern.MatchString(firstName) {
		return fmt.Errorf("first name must be 1-20 letters (may include ' and -)")
	}
	if !validNamePattern.MatchString(lastName) {
		return fmt.Errorf("last name must be 1-20 letters (may include ' and -)")
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

// GameEngine holds the loaded game world and processes commands.
type GameEngine struct {
	db              *mongo.Database
	nouns           map[int]string
	adjectives      map[int]string
	monAdjs         map[int]string
	items           map[int]*gameworld.ItemDef
	rooms           map[int]*gameworld.Room
	monsters        map[int]*gameworld.MonsterDef
	startRoom       int
	sessions        SessionProvider
	onRoomChange    RoomChangeCallback
	monsterMgr      *monsterManager
	RegionWeather   map[int]int // region -> weather state
	monsterLists    []gameworld.MonsterList
}

// SetSessionProvider sets the session provider (called by API layer after init).
func (e *GameEngine) SetSessionProvider(sp SessionProvider) {
	e.sessions = sp
}

// SetRoomChangeCallback sets the callback for cross-machine room state sync.
func (e *GameEngine) SetRoomChangeCallback(cb RoomChangeCallback) {
	e.onRoomChange = cb
}

// notifyRoomChange fires the callback if set.
func (e *GameEngine) notifyRoomChange(change RoomChange) {
	if e.onRoomChange != nil {
		e.onRoomChange(change)
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
		rooms:      make(map[int]*gameworld.Room),
		monsters:   make(map[int]*gameworld.MonsterDef),
		startRoom:  parsed.StartRoom,
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
	for i := range parsed.Rooms {
		e.rooms[parsed.Rooms[i].Number] = &parsed.Rooms[i]
	}
	for i := range parsed.Monsters {
		e.monsters[parsed.Monsters[i].Number] = &parsed.Monsters[i]
	}

	// Initialize monster manager and spawn initial monsters
	e.monsterMgr = newMonsterManager()
	e.monsterLists = parsed.MonsterLists
	e.monsterMgr.SpawnInitialMonsters(parsed.MonsterLists, e.monsters)

	// Initialize weather (all regions sunny)
	e.RegionWeather = make(map[int]int)

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
		adverb := ""
		if player.SpeechAdverb != "" {
			adverb = player.SpeechAdverb + " "
		}
		result := &CommandResult{
			Messages:      []string{fmt.Sprintf("You %s%s, \"%s\"", adverb, verb, msg)},
			RoomBroadcast: []string{fmt.Sprintf("%s %s%s, \"%s\"", player.FirstName, adverb, thirdVerb, msg)},
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

	// GM commands (@ prefix) — silent fail if not GM
	if strings.HasPrefix(verb, "@") {
		if !player.IsGM {
			return &CommandResult{Messages: []string{fmt.Sprintf("I don't understand \"%s\". Type HELP for commands.", strings.ToLower(input))}}
		}
		return e.processGMCommand(ctx, player, verb, args, input)
	}

	// Direction shortcuts — exact match only (single/double letter shortcuts)
	dirMap := map[string]string{
		"N": "N", "NORTH": "N", "S": "S", "SOUTH": "S",
		"E": "E", "EAST": "E", "W": "W", "WEST": "W",
		"NE": "NE", "NORTHEAST": "NE", "NW": "NW", "NORTHWEST": "NW",
		"SE": "SE", "SOUTHEAST": "SE", "SW": "SW", "SOUTHWEST": "SW",
		"U": "U", "UP": "U", "D": "D", "DOWN": "D",
		"O": "O", "OUT": "O",
	}

	if dir, ok := dirMap[verb]; ok {
		return e.doMove(ctx, player, dir)
	}

	// Resolve verb abbreviations — try exact match first, then unique prefix
	verb = resolveVerb(verb)

	switch verb {
	case "LOOK", "EXAMINE", "INSPECT":
		if len(args) == 0 {
			return e.doLook(player)
		}
		return e.doLookAt(player, args)
	case "GO":
		return e.doGo(ctx, player, args)
	case "CLIMB":
		return e.doClimb(ctx, player, args)
	case "GET", "TAKE":
		return e.doGet(ctx, player, args)
	case "DROP":
		return e.doDrop(ctx, player, args)
	case "INVENTORY":
		return e.doInventory(player)
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
		return e.doUnwield(ctx, player)
	case "WEAR":
		return e.doWear(ctx, player, args)
	case "REMOVE":
		return e.doRemove(ctx, player, args)
	case "OPEN":
		return e.doOpen(player, args)
	case "CLOSE":
		return e.doClose(player, args)
	case "SIT":
		player.Position = 1
		return &CommandResult{Messages: []string{"You sit down."}, PlayerState: player,
			RoomBroadcast: []string{fmt.Sprintf("%s sits down.", player.FirstName)}}
	case "STAND":
		player.Position = 0
		return &CommandResult{Messages: []string{"You stand up."}, PlayerState: player,
			RoomBroadcast: []string{fmt.Sprintf("%s stands up.", player.FirstName)}}
	case "KNEEL":
		player.Position = 3
		return &CommandResult{Messages: []string{"You kneel down."}, PlayerState: player,
			RoomBroadcast: []string{fmt.Sprintf("%s kneels down.", player.FirstName)}}
	case "LAY":
		player.Position = 2
		return &CommandResult{Messages: []string{"You lie down."}, PlayerState: player,
			RoomBroadcast: []string{fmt.Sprintf("%s lies down.", player.FirstName)}}
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
		return &CommandResult{Messages: []string{"You have no trained skills yet."}}
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
		return &CommandResult{Messages: []string{
			fmt.Sprintf("Level: %d", player.Level),
			fmt.Sprintf("Build Points: %d", player.Experience/100),
			fmt.Sprintf("Experience Points: %d", player.Experience),
		}}
	case "INFO":
		return e.doInfo(player)
	case "TIME":
		return &CommandResult{Messages: []string{"The sun hangs in an uncertain sky. Time flows strangely in the Shattered Realms."}}
	case "WHISPER":
		return e.doWhisper(player, args, input)
	case "YELL":
		return e.doYell(player, args, input)
	case "GIVE":
		return e.doGive(ctx, player, args)
	case "EAT":
		return e.doEat(ctx, player, args)
	case "SPEECH":
		return e.doSpeech(ctx, player, args, input)
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
	// Roleplay verbs — dispatched via emote table
	case "SMILE", "BOW", "CURTSEY", "WAVE", "NOD", "LAUGH", "CHUCKLE",
		"GRIN", "FROWN", "SIGH", "SHRUG", "WINK", "CRY", "DANCE",
		"HUG", "KISS", "POKE", "TICKLE", "SLAP", "HOWL", "SING",
		"PACE", "FIDGET", "SHIVER", "SNORT", "GROAN", "MUMBLE",
		"BABBLE", "BEAM", "SWOON", "TOAST", "SHUDDER", "POINT",
		"KICK", "KNOCK", "PET", "PUNCH", "SPIT",
		"GAZE", "GLARE", "SCOWL", "COMFORT", "YAWN",
		// New emotes from chat log
		"BLINK", "BLUSH", "CRINGE", "CUDDLE", "COUGH", "FURROW",
		"GASP", "GIGGLE", "GRIMACE", "GROWL", "GULP", "JUMP",
		"LEAN", "NUZZLE", "PANT", "PONDER", "POUT", "ROLL",
		"SCREAM", "SMIRK", "SNICKER", "SALUTE", "STRETCH",
		"TWIRL", "WINCE", "WHISTLE", "MUTTER", "CARESS", "NUDGE",
		"ARCH", "RAISE", "HEAD", "SCRATCH", "CLAP":
		return e.processEmote(player, verb, args)
	case "ACT":
		if len(args) == 0 {
			return &CommandResult{Messages: []string{"Act how?"}}
		}
		action := strings.Join(strings.Fields(strings.ToLower(input))[1:], " ")
		actMsg := fmt.Sprintf("(%s %s)", player.FirstName, action)
		return &CommandResult{Messages: []string{actMsg}, RoomBroadcast: []string{actMsg}}
	case "EMOTE":
		if len(args) == 0 {
			return &CommandResult{Messages: []string{"Emote what?"}}
		}
		text := extractOriginalArgs(input)
		msg := fmt.Sprintf("%s %s", player.FirstName, text)
		return &CommandResult{Messages: []string{msg}, RoomBroadcast: []string{msg}}
	case "RECITE":
		if len(args) == 0 {
			return &CommandResult{Messages: []string{"Recite what?"}}
		}
		text := extractOriginalArgs(input)
		text = strings.Trim(text, "'\"")
		selfMsg := fmt.Sprintf("You recite, '%s'", text)
		roomMsg := fmt.Sprintf("%s recites, '%s'", player.FirstName, text)
		return &CommandResult{Messages: []string{selfMsg}, RoomBroadcast: []string{roomMsg}}
	case "READ":
		return e.doRead(player, args)
	case "PULL", "PUSH", "TURN", "RUB", "TAP", "TOUCH", "SEARCH", "DIG", "RECALL":
		return e.doItemInteraction(ctx, player, verb, args)
	case "CAST":
		return e.doCast(player, args)
	case "CONCENTRATE":
		return &CommandResult{Messages: []string{"You concentrate deeply... [Psionics coming soon.]"}}
	case "BUY":
		return e.doBuy(ctx, player, args)
	case "SELL":
		return e.doSell(ctx, player, args)
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
		return e.doTrain(ctx, player, args)
	case "MINE":
		return e.doMine(player)
	case "FORAGE":
		return e.doForage(player)
	case "CRAFT", "FORGE", "SMELT", "WEAVE", "DYE", "BREW", "ANALYZE":
		return &CommandResult{Messages: []string{fmt.Sprintf("[%s system coming soon.]", strings.Title(strings.ToLower(verb)))}}
	case "ASSIST":
		room := e.rooms[player.RoomNumber]
		roomName := "unknown"
		if room != nil {
			roomName = room.Name
		}
		return &CommandResult{
			Messages:    []string{"Your request for assistance has been noted. A gamemaster will be with you as soon as possible."},
			GMBroadcast: []string{fmt.Sprintf("[GM] %s is requesting assistance at %s (room %d).", player.FirstName, roomName, player.RoomNumber)},
		}
	case "SNIFF", "SMELL":
		return e.processEmote(player, "SNIFF", args)
	case "LISTEN":
		return e.processEmote(player, "LISTEN", args)
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
	"PULL", "PUSH", "TURN", "RUB", "TAP", "TOUCH", "SEARCH", "DIG", "RECALL",
	"CAST", "CONCENTRATE", "BUY", "SELL",
	"DRINK", "SIP", "LIGHT", "EXTINGUISH", "DOUSE",
	"FLIP", "LATCH", "UNLATCH",
	"DEPOSIT", "WITHDRAW", "TRAIN",
	"MINE", "FORAGE",
	"CRAFT", "FORGE", "SMELT", "WEAVE", "DYE", "BREW", "ANALYZE",
	"SNIFF", "SMELL", "LISTEN",
	// Communication
	"WHISPER", "YELL", "SPEECH", "THINK", "CONTACT",
	// Interaction
	"GIVE", "EAT", "COUNT",
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
}

// verbAliases maps short exact aliases that should bypass prefix matching.
// These are kept for single-letter or legacy shortcuts.
var verbAliases = map[string]string{
	"L": "LOOK", "I": "INVENTORY", "Q": "QUIT", "X": "QUIT",
	"INV": "INVENTORY", "STAT": "STATUS", "USE": "WIELD", "UNUSE": "UNWIELD",
	"DON": "WEAR", "EXIT": "QUIT", "SKILL": "SKILLS",
	"WHI": "WHISPER", "THIN": "THINK", "CONTA": "CONTACT",
	"DI": "DIAGNOSE", "TURN": "TWIRL",
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
	if player.Position != 0 {
		posNames := map[int]string{1: "sitting", 2: "laying down", 3: "kneeling"}
		posName := posNames[player.Position]
		if posName == "" {
			posName = "not standing"
		}
		return &CommandResult{Messages: []string{fmt.Sprintf("You can't move while %s! Try STANDing first.", posName)}}
	}

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Error: "You are nowhere!"}
	}

	// Also check ABOVE/BELOW for U/D
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
		return &CommandResult{Messages: []string{"You can't go that way."}}
	}
	if requiresFlight && !player.IsFlying() {
		return &CommandResult{Messages: []string{"You leap into the air but come crashing back down. You need to be able to fly to go that way."}}
	}

	dest := e.rooms[destNum]
	if dest == nil {
		return &CommandResult{Messages: []string{"That way seems to lead nowhere."}}
	}

	oldRoom := player.RoomNumber
	dirNames := map[string]string{
		"N": "north", "S": "south", "E": "east", "W": "west",
		"NE": "northeast", "NW": "northwest", "SE": "southeast", "SW": "southwest",
		"U": "up", "D": "down", "O": "out", "ABOVE": "up", "BELOW": "down",
	}
	dirName := dirNames[dir]
	if dirName == "" {
		dirName = strings.ToLower(dir)
	}

	player.RoomNumber = destNum
	e.SavePlayer(ctx, player)
	result := e.doLook(player)
	result.OldRoom = oldRoom
	result.OldRoomMsg = []string{fmt.Sprintf("%s goes %s.", player.FirstName, dirName)}
	result.RoomBroadcast = []string{fmt.Sprintf("%s arrives.", player.FirstName)}

	// Run IFENTRY scripts for the destination room
	e.applyEntryScripts(ctx, player, dest, result)

	return result
}

// EnterRoom performs a look and runs IFENTRY scripts. Used on login/creation.
func (e *GameEngine) EnterRoom(ctx context.Context, player *Player) *CommandResult {
	result := e.doLook(player)
	room := e.rooms[player.RoomNumber]
	if room != nil {
		e.applyEntryScripts(ctx, player, room, result)
	}
	return result
}

// applyEntryScripts runs IFENTRY scripts and merges results into the command result.
func (e *GameEngine) applyEntryScripts(ctx context.Context, player *Player, room *gameworld.Room, result *CommandResult) {
	sc := e.RunEntryScripts(player, room)
	if len(sc.Messages) > 0 {
		result.Messages = append(result.Messages, sc.Messages...)
	}
	if len(sc.RoomMsgs) > 0 {
		result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
	}
	if len(sc.GMMsgs) > 0 {
		result.GMBroadcast = append(result.GMBroadcast, sc.GMMsgs...)
	}
	e.SavePlayer(ctx, player)
}

func (e *GameEngine) doLook(player *Player) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You are in a void."}}
	}

	result := &CommandResult{
		RoomName: fmt.Sprintf("[%s]", room.Name),
	}

	if !player.BriefMode {
		result.RoomDesc = room.Description
	}

	// List visible items
	for _, ri := range room.Items {
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
			if p.Hidden || p.GMInvis {
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
			playersHere = append(playersHere, fmt.Sprintf("%s the %s%s", p.FullName(), p.RaceName(), posDesc))
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

	// Check for directional look (LOOK N, LOOK NORTH, etc.)
	if dir, ok := lookDirMap[target]; ok {
		room := e.rooms[player.RoomNumber]
		if room != nil {
			destNum, hasExit := room.Exits[dir]
			if !hasExit && dir == "U" {
				destNum, hasExit = room.Exits["ABOVE"]
			}
			if !hasExit && dir == "D" {
				destNum, hasExit = room.Exits["BELOW"]
			}
			if hasExit {
				if dest := e.rooms[destNum]; dest != nil {
					msgs := []string{fmt.Sprintf("[%s]", dest.Name)}
					if dest.Description != "" {
						msgs = append(msgs, descriptionToMessages(dest.Description)...)
					}
					return &CommandResult{Messages: msgs}
				}
			}
			return &CommandResult{Messages: []string{"You see nothing of interest in that direction."}}
		}
	}

	// "look at me/myself" → examine self
	if target == "me" || target == "myself" || target == "self" {
		return e.examinePlayer(player, player)
	}

	// Check if target is a player (online, in same room)
	if found := e.findPlayerInRoom(player, target); found != nil {
		return e.examinePlayer(player, found)
	}

	// Check IN/ON/UNDER prefixes
	prefix := ""
	remaining := target
	for _, p := range []string{"in ", "on ", "under ", "behind "} {
		if strings.HasPrefix(target, p) {
			prefix = strings.ToUpper(strings.TrimSpace(p))
			remaining = strings.TrimPrefix(target, p)
			break
		}
	}

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You see nothing."}}
	}

	// Search room items
	for _, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, remaining, e.getAdjName(ri.Adj1)) {
			if prefix != "" {
				return e.lookPrefixRoomItem(room, itemDef, &ri, prefix)
			}
			return e.examineRoomItem(player, room, itemDef, &ri)
		}
	}

	// Search inventory
	for _, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, remaining, e.getAdjName(ii.Adj1)) {
			return &CommandResult{Messages: []string{fmt.Sprintf("You look at your %s.", name)}}
		}
	}

	return &CommandResult{Messages: []string{"You don't see that here."}}
}

// findPlayerInRoom finds an online player in the same room by name (first name match).
func (e *GameEngine) findPlayerInRoom(self *Player, target string) *Player {
	if e.sessions == nil {
		return nil
	}
	for _, p := range e.sessions.OnlinePlayers() {
		if p.RoomNumber != self.RoomNumber {
			continue
		}
		if p.FirstName == self.FirstName && p.LastName == self.LastName {
			continue // skip self, handled separately
		}
		if p.Hidden || p.GMInvis {
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

// examinePlayer returns a description of a player as seen by the observer.
func (e *GameEngine) examinePlayer(observer *Player, target *Player) *CommandResult {
	isSelf := observer.FirstName == target.FirstName && observer.LastName == target.LastName

	var pronoun, possessive string
	if isSelf {
		pronoun = "You are"
		possessive = "Your"
	} else if target.Gender == 0 {
		pronoun = "He is"
		possessive = "His"
	} else {
		pronoun = "She is"
		possessive = "Her"
	}

	msgs := []string{}
	if isSelf {
		msgs = append(msgs, "You examine yourself.")
	} else {
		msgs = append(msgs, fmt.Sprintf("You look at %s.", target.FullName()))
	}

	msgs = append(msgs, fmt.Sprintf("%s a %s %s.", pronoun, target.RaceName(), genderName(target.Gender)))

	// Health description
	healthPct := float64(target.BodyPoints) / float64(target.MaxBodyPoints) * 100
	switch {
	case healthPct >= 100:
		msgs = append(msgs, fmt.Sprintf("%s in perfect health.", pronoun))
	case healthPct >= 75:
		msgs = append(msgs, fmt.Sprintf("%s has minor injuries.", possessive[:len(possessive)-1]))
	case healthPct >= 50:
		msgs = append(msgs, fmt.Sprintf("%s moderately wounded.", pronoun))
	case healthPct >= 25:
		msgs = append(msgs, fmt.Sprintf("%s seriously wounded.", pronoun))
	case healthPct > 0:
		msgs = append(msgs, fmt.Sprintf("%s critically wounded!", pronoun))
	default:
		msgs = append(msgs, fmt.Sprintf("%s dead.", pronoun))
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

	// Visible conditions
	if target.Bleeding {
		msgs = append(msgs, fmt.Sprintf("%s bleeding.", pronoun))
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
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Go where?"}}
	}

	target := strings.ToLower(strings.Join(args, " "))
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Error: "You are nowhere!"}
	}

	// Try direction first
	dirMap := map[string]string{
		"north": "N", "south": "S", "east": "E", "west": "W",
		"northeast": "NE", "northwest": "NW", "southeast": "SE", "southwest": "SW",
		"up": "U", "down": "D", "out": "O",
	}
	if dir, ok := dirMap[target]; ok {
		return e.doMove(ctx, player, dir)
	}

	// Try portals (doors, trails, arches, etc.)
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		if !isPortal(itemDef.Type) {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			return e.doGoPortal(ctx, player, room, &room.Items[i], itemDef)
		}
	}

	return &CommandResult{Messages: []string{"You don't see that here."}}
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
	if sc.Blocked {
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

func (e *GameEngine) doClimb(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Climb what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
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
			if isPortal(itemDef.Type) {
				return e.doGoPortal(ctx, player, room, &room.Items[i], itemDef)
			}
			// Run IFPREVERB CLIMB scripts on non-portal items
			sc := e.RunPreverbScripts(player, room, "CLIMB", &room.Items[i], itemDef)
			result := &CommandResult{}
			if len(sc.Messages) > 0 {
				result.Messages = sc.Messages
			}
			if len(sc.RoomMsgs) > 0 {
				result.RoomBroadcast = sc.RoomMsgs
			}
			if len(result.Messages) == 0 {
				result.Messages = []string{"You can't climb that."}
			}
			return result
		}
	}

	return &CommandResult{Messages: []string{"You don't see that here."}}
}

// doItemInteraction handles verbs like PULL, PUSH, TURN, RUB, TAP, TOUCH, SEARCH, DIG.
// These run IFPREVERB scripts on the target item. If no script handles it, a default message is shown.
func (e *GameEngine) doItemInteraction(ctx context.Context, player *Player, verb string, args []string) *CommandResult {
	verbLower := strings.ToLower(verb)
	if len(args) == 0 {
		return &CommandResult{Messages: []string{fmt.Sprintf("%s what?", strings.Title(verbLower))}}
	}
	target := strings.ToLower(strings.Join(args, " "))
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
			result := &CommandResult{}
			// Run IFPREVERB scripts (room-level and item-level)
			sc := e.RunPreverbScripts(player, room, verb, &room.Items[i], itemDef)
			result.Messages = append(result.Messages, sc.Messages...)
			result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
			result.GMBroadcast = append(result.GMBroadcast, sc.GMMsgs...)
			// Also run IFVERB scripts on the item archetype
			sc2 := e.RunVerbScripts(player, room, verb, &room.Items[i], itemDef)
			result.Messages = append(result.Messages, sc2.Messages...)
			result.RoomBroadcast = append(result.RoomBroadcast, sc2.RoomMsgs...)
			result.GMBroadcast = append(result.GMBroadcast, sc2.GMMsgs...)
			if sc.Blocked || sc2.Blocked {
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
					result.Messages = append(result.Messages, lookResult.Messages...)
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
				itemName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
				result.Messages = []string{fmt.Sprintf("You %s %s. Nothing happens.", verbLower, itemName)}
			}
			return result
		}
	}

	// Check inventory items (for verbs like RECALL that work on carried items)
	for _, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			// Create a temporary RoomItem for script context
			tempRI := gameworld.RoomItem{Ref: -1, Archetype: ii.Archetype,
				Adj1: ii.Adj1, Adj2: ii.Adj2, Adj3: ii.Adj3,
				Val1: ii.Val1, Val2: ii.Val2, Val3: ii.Val3, Val4: ii.Val4, Val5: ii.Val5}
			sc := e.RunVerbScripts(player, room, verb, &tempRI, itemDef)
			if len(sc.Messages) > 0 || len(sc.RoomMsgs) > 0 {
				result := &CommandResult{}
				result.Messages = append(result.Messages, sc.Messages...)
				result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
				return result
			}
			itemName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			return &CommandResult{Messages: []string{fmt.Sprintf("You %s %s. Nothing happens.", verbLower, itemName)}}
		}
	}

	// Fall back to emote if the verb has one (e.g., RUB, TAP, TOUCH targeting a player)
	if _, hasEmote := emoteTable[verb]; hasEmote {
		return e.processEmote(player, verb, args)
	}

	return &CommandResult{Messages: []string{"You don't see that here."}}
}

func (e *GameEngine) doGet(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Get what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"You can't do that here."}}
	}

	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		if itemDef.Weight >= 1000 {
			continue // immovable
		}
		if isPortal(itemDef.Type) {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			// Add to inventory
			player.Inventory = append(player.Inventory, InventoryItem{
				Archetype: ri.Archetype,
				Adj1: ri.Adj1, Adj2: ri.Adj2, Adj3: ri.Adj3,
				Val1: ri.Val1, Val2: ri.Val2, Val3: ri.Val3, Val4: ri.Val4, Val5: ri.Val5,
			})
			// Remove from room
			room.Items = append(room.Items[:i], room.Items[i+1:]...)
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_remove", ItemRef: ri.Ref})
			e.SavePlayer(ctx, player)
			fullName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You pick up %s.", fullName)},
				RoomBroadcast: []string{fmt.Sprintf("%s picks up %s.", player.FirstName, fullName)},
			}
		}
	}

	return &CommandResult{Messages: []string{"You don't see that here."}}
}

func (e *GameEngine) doDrop(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Drop what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
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
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			droppedItem := gameworld.RoomItem{
				Ref: len(room.Items),
				Archetype: ii.Archetype,
				Adj1: ii.Adj1, Adj2: ii.Adj2, Adj3: ii.Adj3,
				Val1: ii.Val1, Val2: ii.Val2, Val3: ii.Val3, Val4: ii.Val4, Val5: ii.Val5,
			}
			room.Items = append(room.Items, droppedItem)
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_add", Item: &droppedItem})
			player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
			e.SavePlayer(ctx, player)
			fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You drop %s.", fullName)},
				RoomBroadcast: []string{fmt.Sprintf("%s drops %s.", player.FirstName, fullName)},
			}
		}
	}

	return &CommandResult{Messages: []string{"You aren't carrying that."}}
}

func (e *GameEngine) doInventory(player *Player) *CommandResult {
	var msgs []string
	msgs = append(msgs, "You are carrying:")
	if len(player.Inventory) == 0 && len(player.Worn) == 0 && player.Wielded == nil {
		msgs = append(msgs, "  Nothing.")
		return &CommandResult{Messages: msgs}
	}

	if player.Wielded != nil {
		itemDef := e.items[player.Wielded.Archetype]
		if itemDef != nil {
			name := e.formatItemName(itemDef, player.Wielded.Adj1, player.Wielded.Adj2, player.Wielded.Adj3)
			msgs = append(msgs, fmt.Sprintf("  %s (wielded)", name))
		}
	}

	for _, ii := range player.Worn {
		itemDef := e.items[ii.Archetype]
		if itemDef != nil {
			name := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			msgs = append(msgs, fmt.Sprintf("  %s (worn)", name))
		}
	}

	for _, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef != nil {
			name := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			msgs = append(msgs, fmt.Sprintf("  %s", name))
		}
	}

	return &CommandResult{Messages: msgs}
}

func (e *GameEngine) doStatus(player *Player) *CommandResult {
	return &CommandResult{Messages: []string{
		fmt.Sprintf("Name: %s", player.FullName()),
		fmt.Sprintf("Race: %s   Gender: %s   Level: %d", player.RaceName(), genderName(player.Gender), player.Level),
		fmt.Sprintf("Strength: %d   Agility: %d   Quickness: %d", player.Strength, player.Agility, player.Quickness),
		fmt.Sprintf("Constitution: %d   Perception: %d   Willpower: %d   Empathy: %d", player.Constitution, player.Perception, player.Willpower, player.Empathy),
		fmt.Sprintf("Experience: %d   Build Points: %d", player.Experience, player.Experience/100),
	}}
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
	target := strings.ToLower(strings.Join(args, " "))
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		if !isWeapon(itemDef.Type) {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			if player.Wielded != nil {
				player.Inventory = append(player.Inventory, *player.Wielded)
			}
			wielded := player.Inventory[i]
			player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
			player.Wielded = &wielded
			e.SavePlayer(ctx, player)
			fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You wield %s.", fullName)},
				RoomBroadcast: []string{fmt.Sprintf("%s wields %s.", player.FirstName, fullName)},
			}
		}
	}
	return &CommandResult{Messages: []string{"You don't have that."}}
}

func (e *GameEngine) doUnwield(ctx context.Context, player *Player) *CommandResult {
	if player.Wielded == nil {
		return &CommandResult{Messages: []string{"You aren't wielding anything."}}
	}
	itemDef := e.items[player.Wielded.Archetype]
	wepName := "their weapon"
	if itemDef != nil {
		wepName = e.formatItemName(itemDef, player.Wielded.Adj1, player.Wielded.Adj2, player.Wielded.Adj3)
	}
	player.Inventory = append(player.Inventory, *player.Wielded)
	player.Wielded = nil
	e.SavePlayer(ctx, player)
	return &CommandResult{
		Messages:      []string{"You put away your weapon."},
		RoomBroadcast: []string{fmt.Sprintf("%s puts away %s.", player.FirstName, wepName)},
	}
}

func (e *GameEngine) doWear(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Wear what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		if itemDef.WornSlot == "" {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			worn := player.Inventory[i]
			worn.WornSlot = itemDef.WornSlot
			player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
			player.Worn = append(player.Worn, worn)
			e.SavePlayer(ctx, player)
			fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You put on %s.", fullName)},
				RoomBroadcast: []string{fmt.Sprintf("%s puts on %s.", player.FirstName, fullName)},
			}
		}
	}
	return &CommandResult{Messages: []string{"You don't have that."}}
}

func (e *GameEngine) doRemove(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Remove what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	for i, ii := range player.Worn {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			removed := player.Worn[i]
			removed.WornSlot = ""
			player.Worn = append(player.Worn[:i], player.Worn[i+1:]...)
			player.Inventory = append(player.Inventory, removed)
			e.SavePlayer(ctx, player)
			fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You remove %s.", fullName)},
				RoomBroadcast: []string{fmt.Sprintf("%s removes %s.", player.FirstName, fullName)},
			}
		}
	}
	return &CommandResult{Messages: []string{"You aren't wearing that."}}
}

func (e *GameEngine) doOpen(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Open what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
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
			if !containsFlag(itemDef.Flags, "OPENABLE") && !isPortal(itemDef.Type) {
				return &CommandResult{Messages: []string{"You can't open that."}}
			}
			if ri.State == "LOCKED" {
				return &CommandResult{Messages: []string{"It's locked."}}
			}
			if ri.State == "LATCHED" {
				return &CommandResult{Messages: []string{"It's latched shut."}}
			}
			room.Items[i].State = "OPEN"
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "OPEN"})
			fullName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
			return &CommandResult{Messages: []string{fmt.Sprintf("You open %s.", fullName)}}
		}
	}
	return &CommandResult{Messages: []string{"You don't see that here."}}
}

func (e *GameEngine) doClose(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Close what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
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
			room.Items[i].State = "CLOSED"
			e.notifyRoomChange(RoomChange{RoomNumber: player.RoomNumber, Type: "item_state", ItemRef: ri.Ref, NewState: "CLOSED"})
			fullName := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
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
		// The actual whisper content is sent only to the target via WhisperTarget
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
	return &CommandResult{
		Messages:      []string{fmt.Sprintf("You %syell, \"%s\"", adverb, text)},
		RoomBroadcast: []string{fmt.Sprintf("%s %syells, \"%s\"", player.FirstName, adverb, text)},
	}
}

func (e *GameEngine) doGive(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) < 2 {
		return &CommandResult{Messages: []string{"Give what to whom? (give <item> to <player>)"}}
	}
	// Parse: give <item> to <target> OR give <target> <item>
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
		return &CommandResult{Messages: []string{"Give what to whom? (give <item> to <player>)"}}
	}

	// Find the item in inventory
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, itemName, e.getAdjName(ii.Adj1)) {
			continue
		}
		// Find the target player
		target := e.findPlayerInRoom(player, targetName)
		if target == nil {
			return &CommandResult{Messages: []string{"You don't see that person here."}}
		}
		// Transfer item
		fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
		target.Inventory = append(target.Inventory, ii)
		player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
		e.SavePlayer(ctx, player)
		e.SavePlayer(ctx, target)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You give %s to %s.", fullName, target.FirstName)},
			RoomBroadcast: []string{fmt.Sprintf("%s gives %s to %s.", player.FirstName, fullName, target.FirstName)},
			WhisperTarget: target.FirstName,
			WhisperMsg:    fmt.Sprintf("%s gives you %s.", player.FirstName, fullName),
		}
	}
	return &CommandResult{Messages: []string{"You don't have that."}}
}

func (e *GameEngine) doEat(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Eat what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			fullName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			// Remove from inventory (consumed)
			player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
			e.SavePlayer(ctx, player)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("You take a bite of %s.", fullName)},
				RoomBroadcast: []string{fmt.Sprintf("%s eats %s.", player.FirstName, fullName)},
			}
		}
	}
	return &CommandResult{Messages: []string{"You don't have that."}}
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
		fmt.Sprintf("Strength: %-3d   Agility: %-3d   Quickness: %d", player.Strength, player.Agility, player.Quickness),
		fmt.Sprintf("Constitution: %-3d   Perception: %-3d   Willpower: %-3d   Empathy: %d", player.Constitution, player.Perception, player.Willpower, player.Empathy),
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
			item.Adj1 = si.Adj
		}
		player.Inventory = append(player.Inventory, item)
		e.SavePlayer(ctx, player)

		displayName := e.formatItemName(itemDef, item.Adj1, 0, 0)
		priceStr := formatPrice(si.Price)
		return &CommandResult{Messages: []string{fmt.Sprintf("You buy %s for %s.", displayName, priceStr)}}
	}

	return &CommandResult{Messages: []string{"That item is not for sale here."}}
}

func (e *GameEngine) doSell(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Sell what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
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
			displayName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
			player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
			player.Copper += 1
			e.SavePlayer(ctx, player)
			return &CommandResult{Messages: []string{
				fmt.Sprintf("You sell %s for 1 copper.", displayName),
				"[SELLING VALUE TO BE IMPLEMENTED. COMING SOON]",
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
		parts = append(parts, fmt.Sprintf("%d gold", gold))
	}
	if silver > 0 {
		parts = append(parts, fmt.Sprintf("%d silver", silver))
	}
	if cop > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%d copper", cop))
	}
	return joinList(parts)
}

func (e *GameEngine) doDrink(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Drink what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil { continue }
		if itemDef.Type != "LIQUID" && itemDef.Type != "LIQCONTAINER" && itemDef.Type != "FOOD" { continue }
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ii.Adj1)) { continue }
		displayName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
		if itemDef.Type == "FOOD" {
			// EAT logic
			player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
			e.SavePlayer(ctx, player)
			return &CommandResult{Messages: []string{fmt.Sprintf("You eat %s.", displayName)}}
		}
		// Drink/sip
		return &CommandResult{Messages: []string{fmt.Sprintf("You take a drink from %s.", displayName)}}
	}
	return &CommandResult{Messages: []string{"You don't have that."}}
}

func (e *GameEngine) doLight(ctx context.Context, player *Player, args []string, lightOn bool) *CommandResult {
	if len(args) == 0 {
		if lightOn { return &CommandResult{Messages: []string{"Light what?"}} }
		return &CommandResult{Messages: []string{"Extinguish what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	for i, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil { continue }
		if !containsFlag(itemDef.Flags, "LIGHTABLE") { continue }
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ii.Adj1)) { continue }
		displayName := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
		if lightOn {
			player.Inventory[i].State = "LIT"
			e.SavePlayer(ctx, player)
			return &CommandResult{Messages: []string{fmt.Sprintf("You light %s.", displayName)}}
		}
		player.Inventory[i].State = "UNLIT"
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{fmt.Sprintf("You extinguish %s.", displayName)}}
	}
	return &CommandResult{Messages: []string{"You don't have anything to light."}}
}

func (e *GameEngine) doFlip(ctx context.Context, player *Player, args []string) *CommandResult {
	if len(args) == 0 { return &CommandResult{Messages: []string{"Flip what?"}} }
	target := strings.ToLower(strings.Join(args, " "))
	room := e.rooms[player.RoomNumber]
	if room == nil { return &CommandResult{Messages: []string{"You can't do that here."}} }
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil { continue }
		if !containsFlag(itemDef.Flags, "FLIPABLE") { continue }
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ri.Adj1)) { continue }
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
		if latch { return &CommandResult{Messages: []string{"Latch what?"}} }
		return &CommandResult{Messages: []string{"Unlatch what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))
	room := e.rooms[player.RoomNumber]
	if room == nil { return &CommandResult{Messages: []string{"You can't do that here."}} }
	for i, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil { continue }
		if !containsFlag(itemDef.Flags, "LATCHABLE") { continue }
		name := e.getItemNounName(itemDef)
		if !matchesTarget(name, target, e.getAdjName(ri.Adj1)) { continue }
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

func (e *GameEngine) doDeposit(ctx context.Context, player *Player, args []string) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil || !containsModifier(room.Modifiers, "BANK") {
		return &CommandResult{Messages: []string{"There is no bank here."}}
	}
	if len(args) == 0 { return &CommandResult{Messages: []string{"Deposit how much?"}} }
	amount := 0
	fmt.Sscanf(args[0], "%d", &amount)
	if amount <= 0 { return &CommandResult{Messages: []string{"Invalid amount."}} }
	totalCopper := player.Gold*100 + player.Silver*10 + player.Copper
	if totalCopper < amount {
		return &CommandResult{Messages: []string{"You don't have that much money."}}
	}
	// Deduct from carried
	remaining := amount
	if player.Copper >= remaining { player.Copper -= remaining; remaining = 0 } else { remaining -= player.Copper; player.Copper = 0 }
	if remaining > 0 { sn := (remaining+9)/10; if player.Silver >= sn { player.Silver -= sn; player.Copper += sn*10-remaining; remaining = 0 } else { remaining -= player.Silver*10; player.Silver = 0 } }
	if remaining > 0 { gn := (remaining+99)/100; player.Gold -= gn; player.Copper += gn*100-remaining }
	player.BankCopper += amount
	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: []string{fmt.Sprintf("You deposit %s.", formatPrice(amount))}}
}

func (e *GameEngine) doWithdraw(ctx context.Context, player *Player, args []string) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil || !containsModifier(room.Modifiers, "BANK") {
		return &CommandResult{Messages: []string{"There is no bank here."}}
	}
	if len(args) == 0 { return &CommandResult{Messages: []string{"Withdraw how much?"}} }
	amount := 0
	fmt.Sscanf(args[0], "%d", &amount)
	if amount <= 0 { return &CommandResult{Messages: []string{"Invalid amount."}} }
	totalBank := player.BankGold*100 + player.BankSilver*10 + player.BankCopper
	if totalBank < amount {
		return &CommandResult{Messages: []string{"You don't have that much in the bank."}}
	}
	remaining := amount
	if player.BankCopper >= remaining { player.BankCopper -= remaining; remaining = 0 } else { remaining -= player.BankCopper; player.BankCopper = 0 }
	if remaining > 0 { sn := (remaining+9)/10; if player.BankSilver >= sn { player.BankSilver -= sn; player.BankCopper += sn*10-remaining; remaining = 0 } else { remaining -= player.BankSilver*10; player.BankSilver = 0 } }
	if remaining > 0 { gn := (remaining+99)/100; player.BankGold -= gn; player.BankCopper += gn*100-remaining }
	player.Copper += amount
	e.SavePlayer(ctx, player)
	return &CommandResult{Messages: []string{fmt.Sprintf("You withdraw %s.", formatPrice(amount))}}
}

func containsModifier(mods []string, mod string) bool {
	for _, m := range mods { if m == mod { return true } }
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

func (e *GameEngine) doTrain(ctx context.Context, player *Player, args []string) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil || len(room.TrainingSkills) == 0 {
		return &CommandResult{Messages: []string{"There is no training available here."}}
	}
	if len(args) == 0 {
		// List available training
		var msgs []string
		msgs = append(msgs, "Training available here:")
		for _, ts := range room.TrainingSkills {
			name := SkillNames[ts.SkillID]
			if name == "" { name = fmt.Sprintf("Skill #%d", ts.SkillID) }
			currentLvl := 0
			if player.Skills != nil { currentLvl = player.Skills[ts.SkillID] }
			msgs = append(msgs, fmt.Sprintf("  %s (current: %d, max: %d)", name, currentLvl, ts.MaxLevel))
		}
		return &CommandResult{Messages: msgs}
	}
	target := strings.ToLower(strings.Join(args, " "))
	for _, ts := range room.TrainingSkills {
		name := SkillNames[ts.SkillID]
		if !strings.HasPrefix(strings.ToLower(name), target) { continue }
		if player.Skills == nil { player.Skills = make(map[int]int) }
		currentLvl := player.Skills[ts.SkillID]
		if currentLvl >= ts.MaxLevel {
			return &CommandResult{Messages: []string{fmt.Sprintf("You have already reached the maximum level of %s training available here.", name)}}
		}
		cost := (currentLvl + 1) * (currentLvl + 1) * 10 // copper
		totalCopper := player.Gold*100 + player.Silver*10 + player.Copper
		if totalCopper < cost {
			return &CommandResult{Messages: []string{fmt.Sprintf("Training in %s costs %s. You can't afford it.", name, formatPrice(cost))}}
		}
		// Deduct cost
		remaining := cost
		if player.Copper >= remaining { player.Copper -= remaining; remaining = 0 } else { remaining -= player.Copper; player.Copper = 0 }
		if remaining > 0 { sn := (remaining+9)/10; if player.Silver >= sn { player.Silver -= sn; player.Copper += sn*10-remaining; remaining = 0 } else { remaining -= player.Silver*10; player.Silver = 0 } }
		if remaining > 0 { gn := (remaining+99)/100; player.Gold -= gn; player.Copper += gn*100-remaining }
		player.Skills[ts.SkillID] = currentLvl + 1
		e.SavePlayer(ctx, player)
		return &CommandResult{Messages: []string{fmt.Sprintf("You train in %s. You are now level %d.", name, currentLvl+1)}}
	}
	return &CommandResult{Messages: []string{"That skill is not available for training here."}}
}

func (e *GameEngine) doMine(player *Player) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil { return &CommandResult{Messages: []string{"You can't mine here."}} }
	hasMine := containsModifier(room.Modifiers, "MINEA") || containsModifier(room.Modifiers, "MINEB") || containsModifier(room.Modifiers, "MINEC")
	if !hasMine { return &CommandResult{Messages: []string{"There is nothing to mine here."}} }
	return &CommandResult{Messages: []string{"You swing your pick at the rock face... [Mining system coming soon.]"}}
}

func (e *GameEngine) doForage(player *Player) *CommandResult {
	room := e.rooms[player.RoomNumber]
	if room == nil { return &CommandResult{Messages: []string{"You can't forage here."}} }
	switch room.Terrain {
	case "FOREST", "MOUNTAIN", "PLAIN", "SWAMP", "JUNGLE":
		return &CommandResult{Messages: []string{"You search the area for useful materials... [Foraging system coming soon.]"}}
	}
	return &CommandResult{Messages: []string{"There is nothing to forage here."}}
}

func (e *GameEngine) doRead(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Read what?"}}
	}
	target := strings.ToLower(strings.Join(args, " "))

	room := e.rooms[player.RoomNumber]
	if room == nil {
		return &CommandResult{Messages: []string{"There is nothing written on it."}}
	}

	// Search room items
	for _, ri := range room.Items {
		itemDef := e.items[ri.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ri.Adj1)) {
			return e.readRoomItem(room, itemDef, &ri)
		}
	}

	// Search inventory
	for _, ii := range player.Inventory {
		itemDef := e.items[ii.Archetype]
		if itemDef == nil {
			continue
		}
		name := e.getItemNounName(itemDef)
		if matchesTarget(name, target, e.getAdjName(ii.Adj1)) {
			return &CommandResult{Messages: []string{"There is nothing written on it."}}
		}
	}

	return &CommandResult{Messages: []string{"You don't see that here."}}
}

func (e *GameEngine) doWho(player *Player) *CommandResult {
	msgs := []string{"Players online:"}
	if e.sessions != nil {
		for _, p := range e.sessions.OnlinePlayers() {
			if p.GMHidden {
				continue // hidden GMs don't show on WHO
			}
			suffix := ""
			if p.IsGM && p.GMHat {
				suffix = " [Host]"
			}
			msgs = append(msgs, fmt.Sprintf("  %s the %s%s", p.FullName(), p.RaceName(), suffix))
		}
	}
	if len(msgs) == 1 {
		msgs = append(msgs, "  No other players online.")
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

	now := time.Now()
	player := &Player{
		FirstName:     firstName,
		LastName:      lastName,
		Race:          race,
		Gender:        gender,
		Level:         1,
		Strength:      str,
		Agility:       agi,
		Quickness:     qui,
		Constitution:  con,
		Perception:    per,
		Willpower:     wil,
		Empathy:       emp,
		BodyPoints:    bodyPts,
		MaxBodyPoints: bodyPts,
		Fatigue:       fatigue,
		MaxFatigue:    fatigue,
		Mana:          mana,
		MaxMana:       mana,
		Psi:           psi,
		MaxPsi:        psi,
		RoomNumber:    201, // Start at City Gate (tutorial room 3950 requires script execution)
		Position:      0,
		Skills:        make(map[int]int),
		IntNums:       make(map[int]int),
		Gold:          5,
		Silver:        10,
		Copper:        50,
		PromptMode:    true,
		CreatedAt:     now,
		UpdatedAt:     now,
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

// LoadPlayer loads a player from MongoDB by first+last name.
func (e *GameEngine) LoadPlayer(ctx context.Context, firstName, lastName string) (*Player, error) {
	if e.db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	coll := e.db.Collection("players")
	var player Player
	err := coll.FindOne(ctx, bson.M{"firstName": firstName, "lastName": lastName}).Decode(&player)
	if err != nil {
		return nil, err
	}
	return &player, nil
}

// ListPlayers returns all saved characters.
func (e *GameEngine) ListPlayers(ctx context.Context) ([]Player, error) {
	if e.db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	coll := e.db.Collection("players")
	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var players []Player
	if err := cursor.All(ctx, &players); err != nil {
		return nil, err
	}
	return players, nil
}

// ListPlayersByAccount returns all characters belonging to an account.
func (e *GameEngine) ListPlayersByAccount(ctx context.Context, accountID string) ([]Player, error) {
	if e.db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	coll := e.db.Collection("players")
	cursor, err := coll.Find(ctx, bson.M{"accountId": accountID})
	if err != nil {
		return nil, err
	}
	var players []Player
	if err := cursor.All(ctx, &players); err != nil {
		return nil, err
	}
	return players, nil
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

// Helper: format item name with adjectives
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
	article := strings.ToLower(def.Article)
	if article == "" || article == "a" {
		return "a " + name
	}
	return article + " " + name
}

func (e *GameEngine) getItemNounName(def *gameworld.ItemDef) string {
	if name, ok := e.nouns[def.NameID]; ok {
		return name
	}
	return fmt.Sprintf("item#%d", def.Number)
}

func (e *GameEngine) getAdjName(adjID int) string {
	if adjID > 0 {
		if name, ok := e.adjectives[adjID]; ok {
			return name
		}
	}
	return ""
}

func (e *GameEngine) examineRoomItem(player *Player, room *gameworld.Room, def *gameworld.ItemDef, ri *gameworld.RoomItem) *CommandResult {
	result := &CommandResult{}

	// Room-scoped EXAMINE description
	refStr := fmt.Sprintf("%d", ri.Ref)
	if desc, ok := room.ItemDescriptions["EXAMINE:"+refStr]; ok {
		result.Messages = append(result.Messages, descriptionToMessages(desc)...)
	} else if isPortal(def.Type) {
		result.Messages = append(result.Messages, "You can't clearly see where it leads.")
	} else {
		result.Messages = append(result.Messages, "It is nondescript.")
	}

	// Run IFVERB LOOK scripts on the item (can add SHOWROOM output, etc.)
	sc := e.RunVerbScripts(player, room, "LOOK", ri, def)
	if len(sc.Messages) > 0 {
		result.Messages = append(result.Messages, sc.Messages...)
	}
	if len(sc.RoomMsgs) > 0 {
		result.RoomBroadcast = append(result.RoomBroadcast, sc.RoomMsgs...)
	}

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
	// Partial match
	if strings.HasPrefix(n, t) {
		return true
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
