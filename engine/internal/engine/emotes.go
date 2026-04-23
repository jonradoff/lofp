package engine

import (
	"fmt"
	"strings"
)

// emoteEntry defines the self/room messages for an emote.
// Placeholders: %N = actor first name, %P = his/her, %O = him/her, %E = he/she
type emoteEntry struct {
	Self       string // what the actor sees (no target)
	Room       string // what the room sees (no target)
	SelfTarget string // what the actor sees (with target) — %T = target name
	RoomTarget string // what the room sees (with target) — %T = target name
}

var emoteTable = map[string]emoteEntry{
	"SMILE":     {Self: "You smile.", Room: "%N smiles.", SelfTarget: "You smile at %T.", RoomTarget: "%N smiles at %T."},
	"BOW":       {Self: "You bow.", Room: "%N bows.", SelfTarget: "You bow to %T.", RoomTarget: "%N bows to %T."},
	"CURTSEY":   {Self: "You curtsey.", Room: "%N curtseys.", SelfTarget: "You curtsey to %T.", RoomTarget: "%N curtseys to %T."},
	"CURTSY":    {Self: "You curtsy.", Room: "%N curtsies.", SelfTarget: "You curtsy to %T.", RoomTarget: "%N curtsies to %T."},
	"WAVE":      {Self: "You wave.", Room: "%N waves.", SelfTarget: "You wave to %T.", RoomTarget: "%N waves to %T."},
	"NOD":       {Self: "You nod.", Room: "%N nods.", SelfTarget: "You nod to %T.", RoomTarget: "%N nods to %T."},
	"LAUGH":     {Self: "You laugh.", Room: "%N laughs.", SelfTarget: "You laugh at %T.", RoomTarget: "%N laughs at %T."},
	"CHUCKLE":   {Self: "You chuckle.", Room: "%N chuckles.", SelfTarget: "You chuckle at %T.", RoomTarget: "%N chuckles at %T."},
	"GRIN":      {Self: "You grin.", Room: "%N grins.", SelfTarget: "You grin at %T.", RoomTarget: "%N grins at %T."},
	"FROWN":     {Self: "You frown.", Room: "%N frowns.", SelfTarget: "You frown at %T.", RoomTarget: "%N frowns at %T."},
	"SIGH":      {Self: "You sigh.", Room: "%N sighs.", SelfTarget: "You sigh at %T.", RoomTarget: "%N sighs at %T."},
	"SHRUG":     {Self: "You shrug.", Room: "%N shrugs.", SelfTarget: "You shrug at %T.", RoomTarget: "%N shrugs at %T."},
	"WINK":      {Self: "You wink.", Room: "%N winks.", SelfTarget: "You wink at %T.", RoomTarget: "%N winks at %T."},
	"CRY":       {Self: "You cry.", Room: "%N cries.", SelfTarget: "You cry on %T's shoulder.", RoomTarget: "%N cries on %T's shoulder."},
	"DANCE":     {Self: "You dance.", Room: "%N dances.", SelfTarget: "You dance with %T.", RoomTarget: "%N dances with %T."},
	"HUG":       {Self: "You hug yourself.", Room: "%N hugs themselves.", SelfTarget: "You hug %T.", RoomTarget: "%N hugs %T."},
	"KISS":      {Self: "You blow a kiss.", Room: "%N blows a kiss.", SelfTarget: "You kiss %T.", RoomTarget: "%N kisses %T."},
	"POKE":      {Self: "You poke yourself.", Room: "%N pokes themselves.", SelfTarget: "You poke %T.", RoomTarget: "%N pokes %T."},
	"TICKLE":    {Self: "You tickle yourself.", Room: "%N tickles themselves.", SelfTarget: "You tickle %T.", RoomTarget: "%N tickles %T."},
	"SLAP":      {Self: "You slap yourself.", Room: "%N slaps themselves.", SelfTarget: "You slap %T.", RoomTarget: "%N slaps %T."},
	"HOWL":      {Self: "You howl.", Room: "%N howls.", SelfTarget: "You howl at %T.", RoomTarget: "%N howls at %T."},
	"SING":      {Self: "You sing.", Room: "%N sings.", SelfTarget: "You sing to %T.", RoomTarget: "%N sings to %T."},
	"PACE":      {Self: "You pace back and forth.", Room: "%N paces back and forth."},
	"FIDGET":    {Self: "You fidget.", Room: "%N fidgets."},
	"SHIVER":    {Self: "You shiver.", Room: "%N shivers."},
	"SNORT":     {Self: "You snort.", Room: "%N snorts.", SelfTarget: "You snort at %T.", RoomTarget: "%N snorts at %T."},
	"GROAN":     {Self: "You groan.", Room: "%N groans."},
	"MUMBLE":    {Self: "You mumble something.", Room: "%N mumbles something."},
	"BABBLE":    {Self: "You babble.", Room: "%N babbles."},
	"BEAM":      {Self: "You beam.", Room: "%N beams.", SelfTarget: "You beam at %T.", RoomTarget: "%N beams at %T."},
	"SWOON":     {Self: "You swoon.", Room: "%N swoons."},
	"TOAST":     {Self: "You raise your glass in a toast.", Room: "%N raises a toast.", SelfTarget: "You raise a toast to %T.", RoomTarget: "%N raises a toast to %T."},
	"SHUDDER":   {Self: "You shudder.", Room: "%N shudders."},
	"POINT":     {Self: "You point.", Room: "%N points.", SelfTarget: "You point at %T.", RoomTarget: "%N points at %T."},
	"KICK":      {Self: "You kick at the ground.", Room: "%N kicks at the ground.", SelfTarget: "You kick %T.", RoomTarget: "%N kicks %T."},
	"KNOCK":     {Self: "You knock.", Room: "%N knocks.", SelfTarget: "You knock on %T.", RoomTarget: "%N knocks on %T."},
	"TOUCH":     {Self: "You touch yourself.", Room: "%N touches themselves.", SelfTarget: "You touch %T.", RoomTarget: "%N touches %T."},
	"RUB":       {Self: "You rub your hands together.", Room: "%N rubs %P hands together.", SelfTarget: "You rub %T.", RoomTarget: "%N rubs %T."},
	"PET":       {Self: "You pet yourself.", Room: "%N pets themselves.", SelfTarget: "You pet %T.", RoomTarget: "%N pets %T."},
	"PUNCH":     {Self: "You punch the air.", Room: "%N punches the air.", SelfTarget: "You punch %T.", RoomTarget: "%N punches %T."},
	"SPIT":      {Self: "You spit.", Room: "%N spits.", SelfTarget: "You spit at %T.", RoomTarget: "%N spits at %T."},
	"GAZE":      {Self: "You gaze about.", Room: "%N gazes about.", SelfTarget: "You gaze at %T.", RoomTarget: "%N gazes at %T."},
	"GLARE":     {Self: "You glare.", Room: "%N glares.", SelfTarget: "You glare at %T.", RoomTarget: "%N glares at %T."},
	"SCOWL":     {Self: "You scowl.", Room: "%N scowls.", SelfTarget: "You scowl at %T.", RoomTarget: "%N scowls at %T."},
	"COMFORT":   {Self: "You comfort yourself.", Room: "%N comforts themselves.", SelfTarget: "You comfort %T.", RoomTarget: "%N comforts %T."},
	"RECITE":    {Self: "You recite.", Room: "%N recites."},
	"YAWN":      {Self: "You yawn.", Room: "%N yawns.", SelfTarget: "You yawn at %T.", RoomTarget: "%N yawns at %T."},
	"BLINK":     {Self: "You blink.", Room: "%N blinks.", SelfTarget: "You blink at %T.", RoomTarget: "%N blinks at %T."},
	"BLUSH":     {Self: "You blush.", Room: "%N blushes."},
	"CRINGE":    {Self: "You cringe.", Room: "%N cringes."},
	"CUDDLE":    {Self: "You cuddle up.", Room: "%N cuddles up.", SelfTarget: "You cuddle up to %T.", RoomTarget: "%N cuddles up to %T."},
	"COUGH":     {Self: "You cough.", Room: "%N coughs."},
	"FURROW":    {Self: "You furrow your brow.", Room: "%N furrows %P brow."},
	"GASP":      {Self: "You gasp.", Room: "%N gasps."},
	"GIGGLE":    {Self: "You giggle.", Room: "%N giggles.", SelfTarget: "You giggle at %T.", RoomTarget: "%N giggles at %T."},
	"GRIMACE":   {Self: "You grimace.", Room: "%N grimaces."},
	"GROWL":     {Self: "You growl.", Room: "%N growls.", SelfTarget: "You growl at %T.", RoomTarget: "%N growls at %T."},
	"GULP":      {Self: "You gulp.", Room: "%N gulps."},
	"JUMP":      {Self: "You jump up and down.", Room: "%N jumps up and down."},
	"LEAN":      {Self: "You lean back.", Room: "%N leans back.", SelfTarget: "You lean on %T.", RoomTarget: "%N leans on %T."},
	"NUZZLE":    {Self: "You nuzzle.", Room: "%N nuzzles.", SelfTarget: "You nuzzle %T affectionately.", RoomTarget: "%N nuzzles %T affectionately."},
	"PANT":      {Self: "You pant.", Room: "%N pants."},
	"PONDER":    {Self: "You ponder for a moment.", Room: "%N ponders."},
	"POUT":      {Self: "You pout.", Room: "%N pouts."},
	"ROLL":      {Self: "You roll your eyes.", Room: "%N rolls %P eyes.", SelfTarget: "You roll your eyes at %T.", RoomTarget: "%N rolls %P eyes at %T."},
	"SCREAM":    {Self: "You scream!", Room: "%N screams!"},
	"SMIRK":     {Self: "You smirk.", Room: "%N smirks.", SelfTarget: "You smirk at %T.", RoomTarget: "%N smirks at %T."},
	"SNICKER":   {Self: "You snicker.", Room: "%N snickers."},
	"SALUTE":    {Self: "You salute.", Room: "%N salutes.", SelfTarget: "You salute %T.", RoomTarget: "%N salutes %T."},
	"STRETCH":   {Self: "You stretch your arms lazily.", Room: "%N stretches %P arms lazily."},
	"TAP":       {Self: "You tap your foot.", Room: "%N taps %P foot.", SelfTarget: "You tap %T on the shoulder.", RoomTarget: "%N taps %T on the shoulder."},
	"TWIRL":     {Self: "You twirl around.", Room: "%N twirls around."},
	"WINCE":     {Self: "You wince.", Room: "%N winces."},
	"WHISTLE":   {Self: "You whistle innocently.", Room: "%N whistles innocently."},
	"MUTTER":    {Self: "You mutter something under your breath.", Room: "%N mutters something you can't quite make out."},
	"CARESS":    {Self: "You caress yourself.", Room: "%N caresses themselves.", SelfTarget: "You caress %T.", RoomTarget: "%N caresses %T."},
	"NUDGE":     {Self: "You nudge.", Room: "%N nudges.", SelfTarget: "You nudge %T.", RoomTarget: "%N nudges %T."},
	"ARCH":      {Self: "You arch an eyebrow.", Room: "%N arches %P eyebrow.", SelfTarget: "You arch an eyebrow at %T.", RoomTarget: "%N arches %P eyebrow at %T."},
	"RAISE":     {Self: "You raise an eyebrow.", Room: "%N raises an eyebrow.", SelfTarget: "You raise an eyebrow towards %T.", RoomTarget: "%N raises an eyebrow towards %T."},
	"HEAD":      {Self: "You shake your head.", Room: "%N shakes %P head back and forth.", SelfTarget: "You shake your head at %T.", RoomTarget: "%N shakes %P head at %T."},
	"SCRATCH":   {Self: "You scratch your head.", Room: "%N scratches %P head."},
	"CLAP":      {Self: "You clap.", Room: "%N claps.", SelfTarget: "You clap for %T.", RoomTarget: "%N claps for %T."},
	"SNIFF":     {Self: "You sniff.", Room: "%N sniffs.", SelfTarget: "You sniff %T.", RoomTarget: "%N sniffs %T."},
	"LISTEN":    {Self: "You listen carefully.", Room: "%N listens carefully.", SelfTarget: "You listen carefully to %T.", RoomTarget: "%N listens carefully to %T."},

	// Additional emotes from alphabetical list
	"LICK":      {Self: "You lick your lips.", Room: "%N licks %P lips.", SelfTarget: "You kiss %T passionately.", RoomTarget: "%N kisses %T passionately."},
	"NIBBLE":    {Self: "You nibble.", Room: "%N nibbles.", SelfTarget: "You nibble on %T.", RoomTarget: "%N nibbles on %T."},
	"BARK":      {Self: "You bark.", Room: "%N barks."},
	"CLAW":      {Self: "You claw at the air.", Room: "%N claws at the air.", SelfTarget: "You claw at %T.", RoomTarget: "%N claws at %T."},
	"CURSE":     {Self: "You curse.", Room: "%N curses."},
	"DUCK":      {Self: "You duck.", Room: "%N ducks."},
	"HISS":      {Self: "You hiss.", Room: "%N hisses.", SelfTarget: "You hiss at %T.", RoomTarget: "%N hisses at %T."},
	"HOLD":      {Self: "You hold your hands together.", Room: "%N holds %P hands together.", SelfTarget: "You hold %T.", RoomTarget: "%N holds %T."},
	"HULA":      {Self: "You do a little hula dance.", Room: "%N does a little hula dance."},
	"JIG":       {Self: "You do a little jig.", Room: "%N does a little jig."},
	"MOAN":      {Self: "You moan.", Room: "%N moans."},
	"MASSAGE":   {Self: "You massage your temples.", Room: "%N massages %P temples.", SelfTarget: "You massage %T.", RoomTarget: "%N massages %T."},
	"PINCH":     {Self: "You pinch yourself.", Room: "%N pinches themselves.", SelfTarget: "You pinch %T.", RoomTarget: "%N pinches %T."},
	"PLAY":      {Self: "You play around.", Room: "%N plays around."},
	"PURR":      {Self: "You purr.", Room: "%N purrs.", SelfTarget: "You purr at %T.", RoomTarget: "%N purrs at %T."},
	"ROAR":      {Self: "You roar!", Room: "%N roars!"},
	"SNARL":     {Self: "You snarl.", Room: "%N snarls.", SelfTarget: "You snarl at %T.", RoomTarget: "%N snarls at %T."},
	"SNUGGLE":   {Self: "You snuggle up.", Room: "%N snuggles up.", SelfTarget: "You snuggle up to %T.", RoomTarget: "%N snuggles up to %T."},
	"WAG":       {Self: "You wag your finger.", Room: "%N wags %P finger.", SelfTarget: "You wag your finger at %T.", RoomTarget: "%N wags %P finger at %T."},
	"WAIT":      {Self: "You wait patiently.", Room: "%N waits patiently."},
	"WRITE":     {Self: "You write something.", Room: "%N writes something."},
	"YOWL":      {Self: "You yowl!", Room: "%N yowls!"},
	"THUMP":     {Self: "You thump the ground.", Room: "%N thumps the ground.", SelfTarget: "You thump %T.", RoomTarget: "%N thumps %T."},
	"STOMP":     {Self: "You stomp your feet.", Room: "%N stomps %p feet.", SelfTarget: "You stomp towards %T.", RoomTarget: "%N stomps towards %T."},
	"APPLAUD":   {Self: "You applaud.", Room: "%N applauds.", SelfTarget: "You applaud %T.", RoomTarget: "%N applauds %T."},
	"PEER":      {Self: "You peer around.", Room: "%N peers around.", SelfTarget: "You peer at %T.", RoomTarget: "%N peers at %T."},
	"GRUNT":     {Self: "You grunt.", Room: "%N grunts."},
	"DIP":       {Self: "You dip.", Room: "%N dips.", SelfTarget: "You dip %T.", RoomTarget: "%N dips %T."},
	"HANDRAISE": {Self: "You raise your hand.", Room: "%N raises %P hand."},
	"HANDSHAKE": {Self: "You extend your hand.", Room: "%N extends %P hand.", SelfTarget: "You shake hands with %T.", RoomTarget: "%N shakes hands with %T."},
	"HEADSHAKE": {Self: "You shake your head.", Room: "%N shakes %P head."},
	"PICK":      {Self: "You pick at something.", Room: "%N picks at something."},
	"GESTURE":   {Self: "You gesture.", Room: "%N gestures.", SelfTarget: "You gesture at %T.", RoomTarget: "%N gestures at %T."},
}

// selfOverrides maps verb+target to special self-targeted messages.
// Key format: "VERB:me" for self-targeting overrides.
var selfOverrides = map[string][2]string{
	"SPIT:me":   {"You drool all over yourself.", "%N drools all over themselves."},
	"LICK:me":   {"You lick your lips.", "%N licks %P lips."},
	"LAUGH:me":  {"You laugh at yourself.", "%N laughs at themselves."},
	"KICK:me":   {"You kick yourself.", "%N kicks themselves."},
	"THUMP:me":  {"You thump yourself on the head.", "%N thumps themselves on the head."},
}

// raceEmotes are emotes only available to specific races. Key = "RACE:VERB".
var raceEmotes = map[string]emoteEntry{
	// Drakin
	"6:FLICK":   {Self: "You flick your forked tongue.", Room: "%N flicks %P forked tongue."},
	"6:BARE":    {Self: "You bare your teeth menacingly.", Room: "%N bares %P teeth menacingly."},
	"6:SPREAD":  {Self: "You spread your wings wide.", Room: "%N spreads %P wings wide."},
	"6:FOLD":    {Self: "You fold your wings against your body.", Room: "%N folds %P wings."},
	"6:SWISH":   {Self: "You swish your tail.", Room: "%N swishes %P tail."},
	// Aelfen
	"2:RUBEARS": {Self: "You rub the points of your ears.", Room: "%N rubs the points of %P ears."},
	// Highlander
	"3:PULLBEARD": {Self: "You pull on your beard.", Room: "%N pulls on %P beard."},
	// Wolf form emotes (wolfling)
	"4:SCRATCH": {Self: "You scratch behind your ear.", Room: "%N scratches behind %P ear."},
	"4:BARE":    {Self: "You bare your fangs.", Room: "%N bares %P fangs."},
	"4:CHASE":   {Self: "You chase your tail.", Room: "%N chases %P tail."},
	"4:SCENT":   {Self: "You lift your nose and scent the air.", Room: "%N lifts %P nose and scents the air."},
	"4:WHINE":   {Self: "You whine softly.", Room: "%N whines softly."},
	"4:DROOP":   {Self: "You droop your tail.", Room: "%N droops %P tail."},
}

// additionalSelfEmotes — emotes with no target that aren't in the main table.
// These are triggered by specific verbs with no arguments.
var additionalSelfEmotes = map[string][2]string{
	"FUME":     {"You fume.", "%N fumes."},
	"SQUINT":   {"You squint your eyes.", "%N squints %P eyes."},
	"HUM":      {"You hum softly to yourself.", "%N hums softly."},
	"SNIFFLE":  {"You sniffle.", "%N sniffles."},
	"SLOUCH":   {"You slouch.", "%N slouches."},
	"SNORE":    {"You snore loudly.", "%N snores loudly."},
	"SNEEZE":   {"You sneeze.", "%N sneezes."},
	"STARE":    {"You stare off into space.", "%N stares off into space."},
	"PUCKER":   {"You pucker your lips.", "%N puckers %P lips."},
	"CRACK":    {"You crack your knuckles.", "%N cracks %P knuckles."},
	"BITE":     {"You bite your lower lip.", "%N bites %P lower lip."},
	"BOUNCE":   {"You bounce up and down.", "%N bounces up and down."},
	"STRIKE":   {"You strike a heroic pose.", "%N strikes a heroic pose."},
	"CLUTCH":   {"You clutch your head.", "%N clutches %P head."},
	"WIPE":     {"You wipe your brow.", "%N wipes %P brow."},
	"GRIT":     {"You grit your teeth.", "%N grits %P teeth."},
	"TOSS":     {"You toss your hands up in the air.", "%N tosses %P hands up in the air."},
	"ATTENTION": {"You stand at attention.", "%N stands at attention."},
	"TONGUE":   {"You stick your tongue out.", "%N sticks %P tongue out."},
	"WRINKLE":  {"You wrinkle your nose.", "%N wrinkles %P nose."},
	"PUFF":     {"You puff out your cheeks.", "%N puffs out %P cheeks."},
	"DIZZY":    {"You feel dizzy.", "%N looks dizzy."},
	"BAT":      {"You bat your eyelashes.", "%N bats %P eyelashes."},
}

// kissBodyParts defines the body parts that can be kissed and whether they require submit.
var kissBodyParts = map[string]struct {
	requiresSubmit bool
	selfMsg        string
	roomMsg        string
	targetMsg      string
}{
	"head":  {false, "You kiss %T on the head.", "%N kisses %T on the head.", "%N kisses you on the head."},
	"nose":  {false, "You kiss %T on the nose.", "%N kisses %T on the nose.", "%N kisses you on the nose."},
	"ears":  {false, "You kiss %T on the ears.", "%N kisses %T on the ears.", "%N kisses you on the ears."},
	"neck":  {false, "You kiss %T on the neck.", "%N kisses %T on the neck.", "%N kisses you on the neck."},
	"chest": {false, "You kiss %T on the chest.", "%N kisses %T on the chest.", "%N kisses you on the chest."},
	"hand":  {false, "You kiss %T's hand.", "%N kisses %T's hand.", "%N kisses your hand."},
	"lips":  {true, "You kiss %T on the lips.", "%N kisses %T on the lips.", "%N kisses you on the lips."},
	"navel": {true, "You kiss %T on the navel.", "%N kisses %T on the navel.", "%N kisses you on the navel."},
	"leg":   {true, "You kiss %T on the leg.", "%N kisses %T on the leg.", "%N kisses you on the leg."},
	"knee":  {true, "You kiss %T on the knee.", "%N kisses %T on the knee.", "%N kisses you on the knee."},
	"feet":  {true, "You kiss %T's feet.", "%N kisses %T's feet.", "%N kisses your feet."},
}

// expandEmote replaces %N, %P, %O, %E, %T placeholders in emote strings.
func expandEmote(template string, actor *Player, targetName string) string {
	result := template
	for i := 0; i < len(result); i++ {
		if result[i] == '%' && i+1 < len(result) {
			var replacement string
			switch result[i+1] {
			case 'N':
				replacement = actor.FirstName
			case 'P':
				replacement = actor.Possessive()
			case 'O':
				replacement = actor.Objective()
			case 'E':
				replacement = actor.Pronoun()
			case 'T':
				replacement = targetName
			default:
				continue
			}
			result = result[:i] + replacement + result[i+2:]
			i += len(replacement) - 1
		}
	}
	return result
}

// processEmote handles emote commands using the emote table.
func (e *GameEngine) processEmote(player *Player, verb string, args []string) *CommandResult {
	// Emotes always reveal you
	if player.Hidden {
		player.Hidden = false
	}

	// Check race-specific emotes first
	raceKey := fmt.Sprintf("%d:%s", player.Race, verb)
	if raceEntry, ok := raceEmotes[raceKey]; ok {
		selfMsg := expandEmote(raceEntry.Self, player, "")
		roomMsg := expandEmote(raceEntry.Room, player, "")
		return &CommandResult{Messages: []string{selfMsg}, RoomBroadcast: []string{roomMsg}}
	}

	// Check additional self-emotes (no-arg only)
	if len(args) == 0 {
		if selfEmote, ok := additionalSelfEmotes[verb]; ok {
			selfMsg := expandEmote(selfEmote[0], player, "")
			roomMsg := expandEmote(selfEmote[1], player, "")
			return &CommandResult{Messages: []string{selfMsg}, RoomBroadcast: []string{roomMsg}}
		}
	}

	entry, ok := emoteTable[verb]
	if !ok {
		// Fallback generic
		v := strings.ToLower(verb)
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("You %s.", v)},
			RoomBroadcast: []string{fmt.Sprintf("%s %ss.", player.FirstName, v)},
		}
	}

	if len(args) > 0 {
		targetName := strings.ToLower(strings.Join(args, " "))

		// Check for "me"/"myself" with special self-override
		if targetName == "me" || targetName == "myself" || targetName == "self" {
			key := verb + ":me"
			if override, ok := selfOverrides[key]; ok {
				selfMsg := expandEmote(override[0], player, player.FirstName)
				roomMsg := expandEmote(override[1], player, player.FirstName)
				return &CommandResult{Messages: []string{selfMsg}, RoomBroadcast: []string{roomMsg}}
			}
			selfMsg := expandEmote(entry.Self, player, player.FirstName)
			roomMsg := expandEmote(entry.Room, player, player.FirstName)
			return &CommandResult{Messages: []string{selfMsg}, RoomBroadcast: []string{roomMsg}}
		}

		// KISS with body part qualifier: "kiss <person> <bodypart>"
		if verb == "KISS" {
			return e.processKiss(player, args)
		}

		// LICK with submit-dependent behavior
		if verb == "LICK" {
			return e.processLick(player, args)
		}

		// If targeted emote templates exist
		if entry.SelfTarget != "" && entry.RoomTarget != "" {
			// Try to resolve as a player in the room
			found := e.findPlayerInRoom(player, targetName)
			if found != nil {
				displayTarget := found.FirstName
				selfMsg := expandEmote(entry.SelfTarget, player, displayTarget)
				roomMsg := expandEmote(entry.RoomTarget, player, displayTarget)
				targetMsg := expandEmote(entry.RoomTarget, player, "you")
				return &CommandResult{
					Messages:      []string{selfMsg},
					RoomBroadcast: []string{roomMsg},
					TargetName:    found.FirstName,
					TargetMsg:     []string{targetMsg},
				}
			}

			// Try to resolve as a monster in the room
			if _, monDef := e.findMonsterInRoom(player, targetName); monDef != nil {
				monName := FormatMonsterName(monDef, e.monAdjs)
				displayTarget := articleFor(monName, monDef.Unique) + monName
				selfMsg := expandEmote(entry.SelfTarget, player, displayTarget)
				roomMsg := expandEmote(entry.RoomTarget, player, displayTarget)
				return &CommandResult{Messages: []string{selfMsg}, RoomBroadcast: []string{roomMsg}}
			}

			// Try to resolve as a room item
			room := e.rooms[player.RoomNumber]
			if room != nil {
				for _, ri := range room.Items {
					itemDef := e.items[ri.Archetype]
					if itemDef == nil {
						continue
					}
					name := e.getItemNounName(itemDef)
					if matchesTarget(name, targetName, e.getAdjName(ri.Adj1)) {
						displayTarget := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
						selfMsg := expandEmote(entry.SelfTarget, player, displayTarget)
						roomMsg := expandEmote(entry.RoomTarget, player, displayTarget)
						return &CommandResult{Messages: []string{selfMsg}, RoomBroadcast: []string{roomMsg}}
					}
				}
			}

			// Also check player's items (inventory + worn + wielded)
			allItems := make([]InventoryItem, 0)
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
				if matchesTarget(name, targetName, e.getAdjName(ii.Adj1)) || matchesTarget(name, targetName, e.getAdjName(ii.Adj3)) {
					displayTarget := e.formatItemName(itemDef, ii.Adj1, ii.Adj2, ii.Adj3)
					selfMsg := expandEmote(entry.SelfTarget, player, displayTarget)
					roomMsg := expandEmote(entry.RoomTarget, player, displayTarget)
					return &CommandResult{Messages: []string{selfMsg}, RoomBroadcast: []string{roomMsg}}
				}
			}

			// Nothing matched
			return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' here.", targetName)}}
		}
	}

	selfMsg := expandEmote(entry.Self, player, "")
	roomMsg := expandEmote(entry.Room, player, "")
	return &CommandResult{Messages: []string{selfMsg}, RoomBroadcast: []string{roomMsg}}
}

// processKiss handles KISS with body part qualifiers.
// "kiss <person>" — basic kiss
// "kiss <person> <bodypart>" — kiss on specific body part (some require submit)
func (e *GameEngine) processKiss(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"You blow a kiss."}, RoomBroadcast: []string{fmt.Sprintf("%s blows a kiss.", player.FirstName)}}
	}

	// Check if last arg is a body part
	var targetArgs []string
	var bodyPart string
	if len(args) >= 2 {
		lastArg := strings.ToLower(args[len(args)-1])
		if _, ok := kissBodyParts[lastArg]; ok {
			bodyPart = lastArg
			targetArgs = args[:len(args)-1]
		}
	}
	if bodyPart == "" {
		targetArgs = args
	}

	targetName := strings.ToLower(strings.Join(targetArgs, " "))
	found := e.findPlayerInRoom(player, targetName)
	if found == nil {
		// Fall back to basic emote behavior for non-players
		entry := emoteTable["KISS"]
		if entry.SelfTarget != "" {
			// Try room items, inventory, etc.
			room := e.rooms[player.RoomNumber]
			if room != nil {
				for _, ri := range room.Items {
					itemDef := e.items[ri.Archetype]
					if itemDef == nil {
						continue
					}
					name := e.getItemNounName(itemDef)
					if matchesTarget(name, targetName, e.getAdjName(ri.Adj1)) {
						displayTarget := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
						selfMsg := expandEmote(entry.SelfTarget, player, displayTarget)
						roomMsg := expandEmote(entry.RoomTarget, player, displayTarget)
						return &CommandResult{Messages: []string{selfMsg}, RoomBroadcast: []string{roomMsg}}
					}
				}
			}
		}
		return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' here.", targetName)}}
	}

	displayTarget := found.FirstName

	if bodyPart != "" {
		bp := kissBodyParts[bodyPart]
		if bp.requiresSubmit && !found.Submitting {
			return &CommandResult{Messages: []string{fmt.Sprintf("%s is not allowing that.", displayTarget)}}
		}
		selfMsg := expandEmote(bp.selfMsg, player, displayTarget)
		roomMsg := expandEmote(bp.roomMsg, player, displayTarget)
		targetMsg := expandEmote(bp.targetMsg, player, displayTarget)
		return &CommandResult{
			Messages:      []string{selfMsg},
			RoomBroadcast: []string{roomMsg},
			TargetName:    found.FirstName,
			TargetMsg:     []string{targetMsg},
		}
	}

	// Basic kiss (no body part)
	selfMsg := fmt.Sprintf("You kiss %s.", displayTarget)
	roomMsg := fmt.Sprintf("%s kisses %s.", player.FirstName, displayTarget)
	targetMsg := fmt.Sprintf("%s kisses you.", player.FirstName)
	return &CommandResult{
		Messages:      []string{selfMsg},
		RoomBroadcast: []string{roomMsg},
		TargetName:    found.FirstName,
		TargetMsg:     []string{targetMsg},
	}
}

// processLick handles LICK with submit-dependent behavior.
// lick <person> — "You kiss <person> passionately." (if not submitted)
// lick <person> — "You lick <person> all over their body." (if submitted)
func (e *GameEngine) processLick(player *Player, args []string) *CommandResult {
	if len(args) == 0 {
		return &CommandResult{
			Messages:      []string{"You lick your lips."},
			RoomBroadcast: []string{fmt.Sprintf("%s licks %s lips.", player.FirstName, player.Possessive())},
		}
	}

	targetName := strings.ToLower(strings.Join(args, " "))
	found := e.findPlayerInRoom(player, targetName)
	if found != nil {
		displayTarget := found.FirstName
		if found.Submitting {
			selfMsg := fmt.Sprintf("You lick %s all over their body.", displayTarget)
			roomMsg := fmt.Sprintf("%s licks %s all over their body.", player.FirstName, displayTarget)
			targetMsg := fmt.Sprintf("%s licks you all over your body.", player.FirstName)
			return &CommandResult{
				Messages:      []string{selfMsg},
				RoomBroadcast: []string{roomMsg},
				TargetName:    found.FirstName,
				TargetMsg:     []string{targetMsg},
			}
		}
		selfMsg := fmt.Sprintf("You kiss %s passionately.", displayTarget)
		roomMsg := fmt.Sprintf("%s kisses %s passionately.", player.FirstName, displayTarget)
		targetMsg := fmt.Sprintf("%s kisses you passionately.", player.FirstName)
		return &CommandResult{
			Messages:      []string{selfMsg},
			RoomBroadcast: []string{roomMsg},
			TargetName:    found.FirstName,
			TargetMsg:     []string{targetMsg},
		}
	}

	// Fall back to standard emote item targeting
	entry := emoteTable["LICK"]
	room := e.rooms[player.RoomNumber]
	if room != nil && entry.SelfTarget != "" {
		for _, ri := range room.Items {
			itemDef := e.items[ri.Archetype]
			if itemDef == nil {
				continue
			}
			name := e.getItemNounName(itemDef)
			if matchesTarget(name, targetName, e.getAdjName(ri.Adj1)) {
				displayTarget := e.formatItemName(itemDef, ri.Adj1, ri.Adj2, ri.Adj3)
				selfMsg := expandEmote(entry.SelfTarget, player, displayTarget)
				roomMsg := expandEmote(entry.RoomTarget, player, displayTarget)
				return &CommandResult{Messages: []string{selfMsg}, RoomBroadcast: []string{roomMsg}}
			}
		}
	}
	return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' here.", targetName)}}
}
