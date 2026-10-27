# Changelog

## v11.4 — 2026-04-12

### GM Script Editor
- New "GM" nav button visible when playing a GM character
- Web-based script editor: upload, edit, and manage `.scr` files
- Scripts parsed, validated, and hot-loaded into running engine without restart
- Stored in MongoDB with priority ordering (higher priority loads first)
- Version history: last 10 versions with one-click restore
- Shared GM filespace with audit trail (who uploaded what, when)
- Upload `.scr` files from disk or edit directly in the text editor
- Script size limit: 262 KB (110% of largest disk script)

## v11.3 — 2026-04-11

### Mobile UI
- Responsive layout throughout — status bar, nav, admin panel, character creation all adapt to small screens
- Send button (↵) visible on touch devices; keyboard dismissed after each command so iOS zooms back out
- Predictive text / QuickType bar suppressed — autocorrect no longer mangles game commands
- Input font-size 16px on mobile, preventing iOS auto-zoom on focus
- `dvh` viewport units account for mobile browser chrome
- Login/logout messages off by default for new characters

### GM Server Banner
- `@banner <text>` — set a login banner and broadcast notice to all online players
- `@banner` (no args) — clears the banner silently
- Banner displayed on web login screen and telnet/SSH login menu
- In-memory primary storage (works even if MongoDB is offline); persisted to MongoDB as backup for restarts

### Fixes
- `@announce` now broadcasts to all online players (was only echoing back to the sender)
- Weather room descriptions now show prose instead of "The weather is Heavy Snow"
- Rare Weapons Exchange parlor door now works with GO as well as PUSH (`modern_fixes.scr`)

## v11.2.2 — 2026-04-09

### Game Calendar & Seasons
- **In-game time** runs at 6:1 ratio (4 real hours = 1 game day)
- **Full calendar**: 12 named months, 28 days each, with tracked years
- **Seasons** follow game calendar: Spring (months 1-3), Summer (4-6), Autumn (7-9), Winter (10-12)
- **Hot-swap seasonal MLISTs**: monster spawns change with the seasons
- **Season change broadcasts** to outdoor players with atmospheric messages
- **Game time persistence**: calendar state saved to MongoDB, survives restarts
- **TIME command** shows date, season, and dual moon phases (Great Moon + Phulcrus)

### Passive Regeneration
- **Fatigue** regenerates based on Constitution (/20 per tick, min 1)
- **Mana** regenerates based on Willpower + Empathy (/30 per tick, min 1)
- **PSI** regenerates based on Willpower (/20 per tick, min 1)
- **Body Points** regenerate slowly based on Constitution (/50 per tick, min 1)
- **Position multipliers**: laying 3x, sitting 2x, kneeling 1.5x, standing/flying 1x
- Ticks every real minute for all online players

## v11.2.1 — 2026-04-08

### Player Manual
- Full original Player Manual (V3.1, 1994) as in-game modal overlay
- Sticky table of contents with 29 sections
- Accessible from top nav and footer

### SET Command & Settings
- SET command: view/toggle all display settings
- Full/Brief room descriptions, prompt mode
- Filter logon/logoff/disconnect, RPbrief, Battlebrief, Actionbrief, Actbrief

### World
- Dynamic weather system with regional transitions
- Sunrise/sunset broadcasts to outdoor rooms
- PSI command: list disciplines, activate by number, toggle maintained powers
- Fixed monster spawning for seasonal scripts
- Fixed height/weight assignment on character creation

## v10.0.5 — 2026-04-06

### GM Tools
- **@zap <monster>**: actually destroys monsters (was a stub)
- **@trace**: toggle script execution debug output (shows IFVAR conditions, block types)
- **@verb/@verbs**: comprehensive alphabetical listing of every game command with parameters
- **@go/@goplr**: broadcast entry/exit echoes using custom @entry/@exit text
- **@invis** GMs move completely silently (no echoes of any kind)
- **GM flags persist**: @invis, @hide, @gm survive reconnections and server restarts

### Combat & Movement
- **ADVANCE <target>**: properly engages a monster or player in combat
- **RETREAT**: disengages from combat without fleeing the room
- **YELL heard in adjacent rooms**: "You hear someone yell..." through all exits
- **DEPART**: sends to City Gate (201) instead of tutorial room (3950)

### Fixes
- **USE**: now an item interaction verb (triggers IFPREVERB/IFVERB scripts) instead of alias for WIELD
- **ACT**: preserves original text case (was lowercasing)
- **THINK**: preserves full text including first word and punctuation
- **GIVE**: no longer shows duplicate messages to recipient (TargetMsg instead of Whisper)
- **QUIT**: no longer broadcasts "left the Realms" twice
- **Directional LOOK**: shows players and monsters in adjacent rooms
- **SEARCH**: fixed double-space in "You search a skeleton"
- **RECITE**: backslash line breaks for poetry/songs

## v10.0.4 — 2026-04-06

### Multiplayer
- **Room arrival/departure on login/logout**: players in a room see "X arrives." when someone logs in and "X fades from the Realms." on logout
- **Fix GIVE double-message**: recipient was seeing both room broadcast and whisper; now uses TargetName/TargetMsg (excluded from broadcast)
- **Fix QUIT double-broadcast**: "left the Realms" no longer sent twice (once from QUIT command, once from disconnect cleanup)
- **Fix @heal on live players**: GM commands now modify the live session player, not a stale DB copy

### Commands
- **Fix THINK preserving text**: was dropping the first word ("mmm..." in "think mmm... thesnia leaf"); now uses original input text with correct case
- **RECITE line breaks**: backslash (`\`) creates multi-line poetry/song output

### Security
- **Two-tier rate limiting**: 4 commands/sec burst + 10 commands per 10 seconds sustained
- **Max 8 characters per account**

## v10.0.3 — 2026-04-04

### Bot API (New Feature)
Bots were not part of the original game — this is a new feature added to help repopulate the world. Bots can't do anything a human player can't; they follow the same rules, rate limits, and game mechanics.

- **API key system**: generate per-character keys from the main menu (SHA-256 hashed, shown once)
- **WebSocket `auth_apikey`**: bots connect and authenticate via the same WebSocket as regular players
- **[Bot] marker** on WHO list so other players can see who's a bot
- **GM scope control**: GM bots can optionally be restricted from using GM commands
- **SDK examples**: Python, Node.js, and TypeScript in `/bots/` directory with full documentation

### Currency & Trading
- **GIVE money**: `GIVE 5 GOLD TO Taliesin`, `GIVE 10 SILVER TO player`, etc.
- **All currency types**: gold crowns, silver shillings, copper pennies
- **Regional currencies**: kragenmark, danir, shard, darktar, dollar (transferred as MONEY inventory items)
- Accepts plural forms and full names (crowns, shillings, pennies, etc.)

## v10.0.2 — 2026-04-04

### Script Engine
- **ELSE branches** in script conditionals now work (IFVAR, IFITEM, IFNOITEM, IFCARRY, IFFULLDESC, IFIN)
- **Case-insensitive file loading** for DOS-era script filenames — scans directory for matching files regardless of case
- **PORTAL_CLIMBUP, PORTAL_CLIMBDOWN, PORTAL_OVER** types added to parser (were silently falling back to MISC)
- **New script variables**: WARRANT, GFLAG1-4, NUMPLRS, ARENADEATH, SITTING, LAYING, STANDING, KNEELING, WIELDED, WEALTH, REGION

### GM Tools
- **@line1/2/3 [player] <text>** — set persistent description lines on any character, visible on EXAMINE
- **@entry/@exit <text>** — custom room entry/exit messages (replaces default "arrives"/"goes north")
- **@speech <player> <verb phrase>** — set custom speech patterns (e.g., "says grimly", "squawks", "giggle")
- **REPORT <message>** — player command to file reports (broadcast to all GMs, logged with "report" event type)
- **@help** updated with all new commands

### Security Hardening
- **Per-IP WebSocket connection limit** (5 max per IP, 500 total)
- **Command rate limiting** (10 commands/sec, then throttled)
- **Chat flood protection** (5 broadcast messages per 10 seconds)
- **HTML entity sanitization** on all outgoing messages (defense in depth against XSS)
- **REST API auth gates** — /api/characters GET and POST now require authentication
- **MongoDB regex injection fix** in resolvePlayerByName (regexp.QuoteMeta)
- **JWT lifetime reduced** from 30 days to 7 days
- **Dead player command restriction** — limited to DEPART, LOOK, WHO, QUIT, EXP, STATUS, HEALTH

### Fixes
- **Room 225 stairway** — PORTAL_CLIMBUP type was parsed as MISC, blocking GO STAIR
- **Temple of Amilor** — room 592 missing due to AMILOR.SCR case mismatch in git; ELSE branches not firing
- **SKILLS command** — was hardcoded to "no skills yet", now shows actual trained skills
- **Grammar fixes** — "A a fist" → proper article, "your an axe" → "your axe", natural weapons don't drop
- **SPEECH** — removed as player command, now GM-only via @speech

## v10.0.1 — 2026-04-04

### Character Management
- **Soft-delete characters** from main menu with confirmation modal
- **Unique first names** enforced on character creation
- **Admin character recovery**: browse deleted characters, recover with optional rename if name conflicts
- **Name validation split**: monster/game names blocked as exact match only; slurs blocked as substring
- Removed over-aggressive substring matching (e.g., "Pendragon" no longer blocked for containing "dragon")

### Script Engine Variables
- **WARRANT**: player warrant level for law/crime system
- **GFLAG1-4**: global flags accessible in scripts
- **NUMPLRS**: total online player count
- **ARENADEATH**: whether player died in arena
- **SITTING, LAYING, STANDING, KNEELING**: position state variables
- **WIELDED, WEALTH, REGION**: equipment and location variables

### Fixes
- **Fixed Taliesin login**: reserved-word validation was running on login, not just creation — "Pendragon" was blocked because it contains "dragon"
- **Name validation moved to creation only** — existing characters log in without name checks
- **Version notes deeplink**: /version-notes URL works as direct link

### Infrastructure
- **Engine test harness**: first unit test (TestLoadPlayerTaliesin) verifying MongoDB player lookup
- **Warrant field** added to Player struct for future law system

## v10.0.0 — 2026-04-04

### Crafting System
- **MINE** ore from MINEA/B/C rooms — ore purity based on grade (A=50-100%, B=30-70%, C=10-40%)
- **SMELT** ore into refined material at FORGE rooms — purity + skill determines success
- **CRAFT/FORGE** weapons and armor at FORGE rooms (Weaponsmithing skill ≥ weapon PARAMETER1)
- **WEAVE** clothing at LOOM rooms (Dyeing/Weaving skill ≥ PARAMETER2)
- **CRAFT** wood items at FLETCHER rooms (Wood Lore skill ≥ PARAMETER1)
- **FORAGE** terrain-based gathering using ForageDef tables (forest/mountain/plain/swamp/jungle)
- **DYE** materials at LOOM rooms — apply color adjectives from DYE items to DYEABLE materials
- **ANALYZE** ore purity (Mining 3+) and reagent properties (Alchemy)
- **BREW** potions via alchemy — 32 recipes from original alchemy.bin (3 reagents = catalyst + 2 types)
- Recipes: Body Restoration, Strength I-III, Agility I-III, Mystic Armor, Haste, Invisibility, Globe of Protection, Cure Poison/Disease, and more
- Potions yield 2-5 sips, skill level gates recipe access

### Full Skill System (36 skills)
- **Build point costs** from skills.txt (first rank + per-rank costs, e.g., Edged 12/5, Healing 20/2)
- **Skill prerequisites** enforced (magic schools need Spellcraft, Mind over Mind/Matter needs Psionics, Dodge needs weapon skill, Disguise needs Stealth)
- **Weapon skills**: +5 attack per rank (Edged, Crushing, Polearms, Missile, Drakin, Natural, Thrown)
- **Dodge & Parry**: +5 defense per rank (requires any weapon skill)
- **Martial Arts**: +5 attack unarmed, +2 defense unarmed per rank; 10+ ranks = hit magic-required monsters
- **Combat Maneuvering**: -1 sec roundtime per rank; 2% per rank chance to dodge monster special attacks (max 95%)
- **Endurance**: +4 max body points per rank; 1% elemental damage reduction per rank (max 50%)
- **ANOINT**: apply poison to wielded weapon (Trap & Poison Lore skill, level = rank)
- **TEND**: heal wounds with Healing skill (2 + skill×2 + random, +50% same-race bonus, stops bleeding)
- **UNLEARN**: remove one skill rank, get back build points minus one
- 30 starting build points for new characters

### Treasure & Loot System
- **Treasure tables** based on monster TREASURE level (0-127)
- **Coin drops** always (copper amount scales with level)
- **Weapon drops**: level-appropriate from item database, chance for magic bonus and premium materials
- **Armor drops**: level-appropriate with magic bonus chance
- **Spell scrolls**: random learnable spell on scroll (spell level = treasure/3)
- **Locked chests**: with lock difficulty and traps (13 types + spell glyphs)
- **Monster weapon drops** on death (with WeaponPlus bonus)
- **SEARCH** dead monsters for treasure (one-time loot, corpses decay after 60s)
- **MONEY items** auto-convert to currency on GET; visible as "some coins" in LOOK
- **SELL** uses VAL1 as copper value (merchants pay 50%)

### Combat Fidelity (from LEGENDS.DOC / shirla.cap)
- **Fatigue drain** on melee attacks based on weapon weight (ranged exempt)
- **Fatigue ToHit penalties**: -10 at half fatigue, -25 at quarter fatigue
- **Weapon clash** on roll < 3 vs weapon-wielding monsters (2d100 vs weapon strength)
- **Damaged weapons**: -10 ToHit penalty; break on second clash
- **Backstab** requires puncture weapon only (daggers, rapiers)
- **Death**: 90% XP penalty toward current build point
- **Spellcraft**: base 25% + EMP/10 + spellcraft×5 (max 95%), fumble on 98+
- **Mana cost** = spell level (from LEGENDS.DOC)
- **NOCK/LOAD** ranged weapons with ammunition; must reload between shots
- **Invisible vs Hidden** distinction: Invisibility spell not broken by movement

### Monster System Improvements
- **Demand-based spawning**: monsters only spawn when players enter rooms
- **Correct MLIST format**: probability/maxCount (not min/max)
- **Monster unloading**: despawn after 3 minutes with no players (ETERNAL exempt)
- **Psi defense auto-activation** on spawn (Wall of Force, Psychic Shield, etc.)
- **Local-only monster broadcasts** (no cross-machine ghost monsters)
- **Corpse decay** after 60 seconds; dead monsters show as "(dead)" in LOOK

### Other
- **@mlist** GM command: show all spawned monsters worldwide
- **@lsk** GM command: list all skills with IDs
- **@edpl** alias for @edplayer, **@edsk** alias for @eds
- **@help** sorted alphabetically with all commands listed
- **CREDITS** updated with full original team and 2026 re-release info
- **REVEAL/UNHIDE** commands; auto-reveal on movement, emotes, attacks
- Item value fields (VAL1-5) fully implemented for all documented uses

## v0.97 — 2026-04-04

### Combat System
- **ATTACK/KILL <monster>** — original format output: `[ToHit: X, Roll: Y] Hit!/Miss./Excellent Hit!`
- **Damage severity tiers**: Puny, Grazing, Insignificant, Minor, Passable, Good, Masterful, Grisly, Severe, Ghastly
- **Attack verbs by weapon type**: swings (slash), thrusts (pierce/pole), slashes (claw)
- **Body part targeting**: head, body, right/left arm, right/left leg, back; animal parts for non-humanoids
- **Weapon elemental crits (VAL3)**: 10-50% chance heat/cold/electric bonus damage with VAL5 max
- **Racial slayer weapons (VAL3 21-32)**: bonus damage vs Undead, Dragon, Giant, Troll, etc.
- **Weapon poison (VAL4 51-100)**: delivers poison on hit
- **MAGICWEAPON gating**: some monsters require enchanted weapons (level 1/2/3), shows TEXI text
- **Monster guard behavior**: guard monsters intercept attacks on their charge
- **Cry for law (strategy 1-25, 101-125)**: attacking lawful NPCs alerts nearby guards
- **Monster poison/disease/fatigue**: % chance per hit to inflict conditions
- **EXTRABODY**: monsters have 50-100% extra HP not counted toward XP
- **Weather combat modifiers**: rain -10, heavy rain -20, thunderstorm -30, gale -40, hurricane -50 (outdoor only)
- **Arena rooms**: ExtraMod ARENA prevents lethal damage
- **Alignment shifts**: killing evil monsters → more good, good → more evil
- **Hostile monsters** (strategy 301+) auto-attack players entering rooms
- **Monster flee AI**: strategy-based (fight to death 501+, flee when wounded 301-500, animals flee 201-300)
- **Monster special attacks**: SPECUSE/SPECDMG with TEXX text overrides
- **FLEE command**: escape combat (quickness/agility based chance, random exit)
- **Combat stances**: OFFENSIVE (+15/-15), DEFENSIVE (-15/+15), BERSERK (+25/-25, Murg), WARY (-5/+5)
- **Roundtime**: `[Round: X sec]` format, quickness-based
- **Death**: Eternity, Inc. message (matching original text) → DEPART to respawn
- **Real XP table**: 100-level XP/build-point table from GM Manual, build points = 20 + 10*level

### Magic System
- **PREPARE <spell>** then **CAST [target]** — two-step spell casting
- **60+ spells** across 5 schools with mana costs and casting times
- **Offensive**: Flame Bolt, Force Blade, Lightning Bolt, Freezing Sphere, Call Meteor, Chain Lightning, Energy Maelstrom
- **Healing**: Body Restoration I/II/III, Invigoration I/II, Reconstruction, Regeneration
- **Defense**: Mystic Armor (+20), Globe of Protection (+50/+100), Spectral Shield (+20), Spell Shield (+25)
- **Buffs**: Strength I/II/III, Agility I/II/III, Fly, Invisibility, Haste
- **Spellcraft skill checks**: school skill + spellcraft + willpower vs spell level
- **Magic resistance**: monsters can resist spells, elemental immunities apply

### Psionic System
- **PSI <discipline>** then **PROJECT [target]** — two-step psionic projection
- **Mind over Matter**: Kinetic Thrust, Pyrokinetics, Cryokinetics, Electrify, Wall of Force (+25), Force Field (+75), Flight
- **Mind over Mind**: Psychic Blast, Psychic Crush, Terror, Pain, Screen/Shield/Barrier/Fortress defense chain
- **Psi point costs**, skill checks (Psionics + school skill + willpower), psi resistance

### World Systems
- **SKIN <dead monster>** — weighted random drops from SkinItem definitions with SkinAdj
- **Container traps (VAL4)**: 13 trap types (needle, gas, acid, blade, explosive, glyph spells) trigger on OPEN
- **Highlander BLEND**: hide in mountain/cave terrain (race-restricted)

### Race-Specific Emotes
- **Drakin**: flick tongue, bare teeth, spread/fold wings, swish tail
- **Aelfen**: rub ears · **Highlander**: pull beard
- **Wolfling**: scratch behind ear, bare fangs, chase tail, scent air, whine, droop tail
- **23 self-emotes**: fume, squint, hum, sneeze, crack knuckles, bat eyelashes, bounce, strike pose, and more

### Infrastructure
- **PlayerMessageFunc**: targeted messages from background tasks (combat, aggro)
- **Monster combat tick**: integrated into 3-second behavior loop
- **Combat disengagement**: moving rooms, fleeing, or target dying clears combat state
- **Character name validation**: reserved-word filter (monster names, profanity, game terms)

## v0.96 — 2026-04-04

### Monster System
- **Monster spawning**: Monsters appear in room descriptions from original MonsterList data
- **TEX1-4 ambient text**: Monsters emit random flavor text on a timer based on Speed
- **Wandering**: Non-hostile monsters (strategy < 301) wander between rooms via exits
- **TEXG/TEXE/TEXM**: Custom text for spawn, room entry, and movement (direction appended)
- **Examine monsters**: `exam skeleton` shows monster description
- **Emote targeting**: Monsters targetable by all emotes (`point skeleton`, etc.)
- **@spawn/@genmon fixed**: GM commands now actually create monster instances
- **@genmon creates sedated** (inactive), **@spawn creates active** (will act)
- **Speed-based ticking**: Speed 1 = every 3s, Speed 3 (default) = every 9s

### New Emotes (30+)
- **lick, nibble, bark, claw, curse, duck, hiss, hold, hula, jig, moan, massage, pinch, play, purr, roar, snarl, snuggle, wag, wait, write, yowl, thump, applaud, peer, grunt, dip, handraise, handshake, headshake, pick, gesture**
- **Self-targeting overrides**: `spit me` → drool, `lick me` → lick lips, `laugh me` → laugh at self, `kick me` → kick self, `thump me` → thump head
- **KISS body parts**: head, nose, lips, ears, neck, chest, hand, navel, leg, knee, feet
- **Submit-gated**: lips, navel, leg, knee, feet require target to be submitting

### Submit System
- **SUBMIT/UNSUBMIT** commands — accept intimate emotes from other players
- **LICK** behavior changes: non-submitted → passionate kiss, submitted → full body lick
- Moving to a new room automatically clears submit state

### Infrastructure
- **CEVENT ECHO delivery**: Script ECHO messages from cyclic events now broadcast to players in rooms
- **RoomBroadcast callback**: Engine background tasks (monsters, CEVENTs) can push messages to rooms via WebSocket
- Monster article handling: "an orc" vs "a skeleton", unique monsters without article

## v0.95 — 2026-04-03

### Lock & Unlock
- **LOCK/UNLOCK commands** — match KEY items via Val3, proper feedback for missing/wrong keys
- Latched items block open; locked items block passage

### Ordinal Targeting
- **"2 gate", "other gate", "second gate"** — target the Nth matching item in a room
- Works across all 19 item-matching functions (get, drop, look, open, close, wield, wear, etc.)

### Mechanoid Emote
- **EMOTE/UNEMOTE** — Mechanoid racial ability to toggle emotional state (race 7 only)
- **ACT** remains the general-purpose roleplaying command for all races

### Verb Aliases & Commands
- **ORDER** as BUY synonym
- **UNLIGHT/IGNITE** as EXTINGUISH/LIGHT aliases
- **QUAFF** as DRINK alias, **SHOUT** as YELL alias, **PLACE** as DROP alias
- **RECALL** with no args runs room-level IFVERB RECALL -1 scripts
- **ACTBRIEF/RPBRIEF** toggle commands
- **POUR** verb stub
- **A** as shorthand for ACT (freeform roleplay)

## v0.94 — 2026-04-03

### Script Engine
- **226 named global variables** (DANWATER, TECHSWITCH, etc.) loaded from VARIABLE definitions, synchronized across machines
- **MUL/DIV/MOD** arithmetic operations in scripts
- **GENMON/ZAPMON** — spawn and remove monsters from scripts
- **GFLAG** — set flags for all players in a room
- **RELOGIN/IFLOGIN** — force player room on login
- **IFFULLDESC/IFIN** — new conditional blocks
- **NEWPUT** — place items inside containers from scripts
- **DUMMY1-5** volatile scratch variables
- **PVAL 0-19** persistent system-wide variables (MongoDB-backed)
- **CEVENT system** — cyclic timed events parsed and executing on 3-second ticks
- **Implicit ENDIF** — parser handles missing ENDIF when new verb block starts (matches original engine)
- **Room-level IFVERB blocks** now fire from doItemInteraction and position commands
- **Direction verb resolution** — "O" resolves to "OUT" for IFPREVERB matching, all abbreviations expanded

### Parsing
- **REGIONDEF** expanded: DepartRoom, Weather, Treasure, Teleport, Summoning, spell modifiers, MineAdj
- **MONEYDEF** — multi-currency per region definitions
- **FORAGEDEF/MINEDEF** — full forage and mining definitions parsed
- **Room REGION field** parsed and accessible in scripts
- **Monster psionics** — PSI, PSIUSE, PSISKILL, PSIRESIST, PSILEVEL, DISCIPLINE
- **Monster combat** — IMMUNITY (10 types), WEAPON, WEAPONPLUS, MAGICWEAPON, SPECUSE/SPECDMG, EXTRABODY, FATIGUE

### Food & Drink
- **Bite/sip tracking** — PARAMETER1 defines bites for food, sips for liquid. VAL2 tracks remaining. Items removed only when empty.
- **Spell on consume** — ITEMVAL3 triggers spell effect on first bite (e.g., Mindlink from thesnia leaf)
- **Item scripts run on eat** — root-level IFVAR blocks execute before checking spell effects

### Items & Verbs
- **Store adjective in ADJ3** — bought items place variety adjective in last slot, leaving ADJ1/ADJ2 for crafting
- **Worn/wielded item visibility** — all verbs (touch, sniff, wave, emotes) now find inventory, worn, and wielded items
- **SIT/LAY/KNEEL/STAND trigger IFVERB scripts** — position commands run room-level verb scripts
- **CLIMB handles MOVE** — non-portal items with IFPREVERB CLIMB + MOVE now work
- **SNIFF/SMELL/LISTEN** try item interaction first, fall back to emote
- **@peek named variables** — `@peek DANWATER` works, plus PVAL and full getVar fallback
- **@yank fix** — updates live session player, not just database

### Telepathy & Spells
- **Mindlink (#403)** — first working spell, grants 1hr telepathy via food consumption
- **Telepathy system** — THINK broadcasts to telepathy-enabled players only
- **Ephemeral innate telepathy** — race 8 starts with telepathy active
- **TELEPATHY command** — toggle on/off

### Original Fidelity
- Global messages: "has just entered/left the Realms"
- WHO: 4-column grid with "There are N adventurers in the Realms."
- EAT: "You take a bite of X. (N bites remaining)"
- Death: Eternity, Inc. message + DEPART stub
- HP → BP (Body Points) throughout UI
- Article fix: "an axe" not "a axe"

### Admin & Infrastructure
- Event Monitor: real-time admin WebSocket feed
- Session Capture: record, view, download gameplay
- Capture delete with confirmation
- /healthz endpoint for vibectl
- Backend connection indicator
- Admin Players/Users sorted by recency with relative timestamps

## v0.93 — 2026-04-03

### New Commands
- **HIDE/SNEAK**: Stealth system — conceal yourself and move while hidden
- **FLY/ASCEND/DESCEND/LAND**: Flight system for Drakin race and spell-granted flight
- **MARK**: Set teleport anchor points (1-10) for future spell use
- **BALANCE**: View bank account balance
- **SPELL**: List all known spells
- **THINK**: Telepathy broadcast to all players
- **CANT**: Covert message requiring Legerdemain skill
- **UNDRESS**: Remove outermost worn item
- **UNPROMPT**: Toggle off prompt indicators
- **VERSION/CREDITS**: Game info

### Combat & Spell Stubs (40+ verbs)
- All combat verbs stubbed with TODO: ATTACK, KILL, ADVANCE, RETREAT, GUARD, BACKSTAB, BITE, AVOID
- Combat stances: BERSERK, FRENZY, DEFENSIVE, OFFENSIVE, WARY, NORMAL
- Spell verbs: INVOKE, PREPARE, CHANT, COMMAND, MASTER
- Ranged: NOCK, LOAD, SPECIALIZE
- Skills: DISARM, STEAL, STALK, TEACH, SELFTRAIN, UNLEARN, ANOINT, TRAP, SURVEY, SPLIT
- Racial: BLEND, CALL, TRANSFORM, MOLD, DISGUISE
- Social: SUBMIT, ARREST, ENROLL, INITIATE, JOIN, FOLLOW, LEAVE, DISBAND
- Other: TEND, BREAK, PUT, FILL, SKIN, REPAIR, WORK

### Script Variables Expansion
- Physical attributes: HEI/HEIT, WEI/WEIT, AGE/AGET
- Form states: WOLFFORM, SLIMEFORM, OTHERFORM, UNDEAD, DISGUISED
- Status: SLEEPING, SUBMITTING, ROUNDTIME, SPELLNUM, POSITION
- Wealth: WEALTH (total copper)
- Room: WILDERNESS, ASTRAL, TERRAIN (numeric), OBJWEIGHT

### Player System
- New fields: Height, Weight, Age (with true variants), Marks, PreparedSpell
- Form states: WolfForm, SlimeForm, Disguised
- Status: Sleeping, Submitting, Undead

### Fixes
- **Article fix**: "an axe" instead of "a axe" — auto-detects vowel sounds
- **Body Points**: All "HP" references changed to "BP" throughout UI
- **GO command**: Non-portal items with IFPREVERB GO scripts now work (e.g., stairways)
- **CLEARVERB + MOVE**: Scripts that block default GO but provide MOVE now work correctly
- **Session Capture**: Fixed null array bug preventing line recording
- **Copy-paste**: Terminal text now selectable on Windows
- **Stop Recording**: Fixed stale WebSocket reference

### Infrastructure
- `/healthz` endpoint for vibectl health monitoring
- Backend connection indicator in nav bar
- Character list shows "Connecting to server..." when backend is unavailable
- Create Character button hidden until backend responds
- Event Monitor disconnect messages

## v0.92 — 2026-04-03

### Admin Event Monitor
- Real-time WebSocket event feed for admin monitoring
- Categories: system, time, monster, script, world state, weather
- Background time cycle publishes hour/day/night transitions
- Category filter, clear button, auto-scroll

### Session Capture
- Record game sessions to MongoDB
- Start/stop from modal overlay during gameplay
- View previous captures with color-coded output
- Download as .txt file
- Auto-stops on disconnect

## v0.91 — 2026-04-03

### Script Engine Expansion
- **80+ variables** supported in IFVAR conditions: player stats (STR/AGI/CON/QUI/WIL/PER/EMP), resources (BODYPOINTS/MANAPOINTS/PSIPOINTS/FATPOINTS), state (DEAD/FLYING/KNEELING/HIDDEN), organization (ORG/ORGRANK/ALIGN), room info (RNUM/OUTDOOR/TERRAIN/PLRSINROOM/MONINROOM), time (TIM/DAY/NIGHT/DATE/MONTH/YEAR), weather (WEA), exits (EXITN/EXITS/etc.), flags (FLAG1-4), skills (SKILL0-35)
- **IFSAY** script blocks now execute on player speech (enables NPC dialogue and quest triggers)
- **IFCARRY** condition checks if player carries a specific item
- **IFNOITEM** condition (negation of IFITEM)
- **IFTOUCH** block execution for touch-type verbs
- **AFFECT** action switches script context to another room for multi-room effects
- **RANDOM** action for random number generation in scripts
- **DAMAGEPLR** action for environmental damage (traps, falls)
- **STRCVT** action with %0-%9 string substitution in ECHO text
- **POSITION** action forces player position from scripts
- **String substitution** expanded: %s (he/she), %i (him/her), %h (his/her), %p (group name), %m (monster), %c (newline), %0-%9 (STRCVT)

### World Systems
- **In-game clock and calendar**: 343-day year, 12 months, day/night cycle (1 real minute = 1 game hour)
- **Monster spawning**: Monsters populate rooms from MonsterList data, displayed in room descriptions
- **Weather system**: 15 weather states per region, shown in outdoor room descriptions
- **Room lighting**: Dark rooms require light sources; DAY_LIGHT rooms dark at night

### New Commands
- **DRINK/SIP**: Consume liquid/food items
- **LIGHT/EXTINGUISH**: Light and douse LIGHTABLE items for dark room navigation
- **FLIP**: Toggle FLIPPED/UNFLIPPED state on FLIPABLE items
- **LATCH/UNLATCH**: Toggle latched state; latched items can't be opened
- **DEPOSIT/WITHDRAW**: Banking in BANK rooms
- **TRAIN**: Skill training in rooms with TRAINING definitions (36 skills)
- **MINE**: Mining in MINEA/B/C rooms (stub)
- **FORAGE**: Foraging in wilderness terrain (stub)
- **CAST**: Spell casting with 150+ registered spells across 5 schools (stub behavior)
- **CRAFT/FORGE/SMELT/WEAVE/DYE/BREW/ANALYZE**: Crafting stubs

### Spell Registry
- 150+ spells registered across 5 schools: Conjuration (100-144), Enchantment (200-250), Necromancy (301-356), General Magic (400-415), Druidic (500-538)
- CAST command recognizes all spells by name prefix

### Monster System
- Expanded MonsterDef with 20+ new fields: alignment, magic resistance, mana, spell use, poison/disease, skin items, text overrides, guard, stealable, eternal, discorporate
- All monster fields parsed from scripts
- Passive monster spawning in rooms

### Player Systems
- Organization/Guild fields (ORG, ORGRANK)
- Alignment tracking
- Build points
- Banking (BankGold/Silver/Copper)
- Known spells registry
- Transient flags (FLAG1-4, reset on room entry)
- 36 named skills with training system

### Training System
- TRAINING room definitions parsed from scripts
- TRAIN command with skill listing, level caps, and gold cost
- Cost formula: (current_level+1)^2 * 10 copper per level

## v0.9 — 2026-04-03

### Game Engine
- Script parser loads 2273 rooms, 1990 items, 297 monsters from original .SCR files
- Fixed description parser: `*DESCRIPTION_START ITEM READ/EXAM` no longer overwrites room descriptions
- Fixed item deduplication: later script definitions correctly override earlier ones (ANTI.SCR placeholders replaced)
- Fixed script block parser: missing ENDIF no longer consumes subsequent item definitions
- Seasonal scripts (ASCRIPT/WSCRIPT/SSCRIPT/PSCRIPT) skipped; base definitions used
- Formatted text preservation: descriptions with leading whitespace or blank lines retain formatting (poems, maps, etc.)

### Movement & Portals
- Portal traversal via GO command with VAL2 destination
- PORTAL_CLIMB, PORTAL_UP, PORTAL_DOWN, PORTAL_OVER, PORTAL_THROUGH support
- CLIMB verb for climbable portals
- Closed/locked portals block passage
- Directional look (LOOK N, LOOK NORTH, etc.) shows adjacent room description

### Script Execution
- IFENTRY blocks execute on room entry (movement, login, character creation)
- IFVAR conditions: INTNUM, ITEMBIT, ITEMVAL, ITEMADJ, LEV, RAC, SKILL, MISTFORM
- IFITEM conditions: check item open/closed/locked/unlocked state
- IFPREVERB/IFVERB execution on item interactions
- Actions: ECHO (PLAYER/ALL/OTHERS), EQUAL, ADD, SUB, NEWITEM, GMMSG, CLEARVERB, MOVE, SHOWROOM
- Actions: LOCK, UNLOCK, OPEN, CLOSE, REMOVEITEM, SETITEMVAL
- Nested conditional blocks supported
- Script text placeholders: %N, %n, %a, %h, %e, %o

### Commands
- Movement: N/S/E/W/NE/NW/SE/SW/UP/DOWN/OUT, GO, CLIMB
- Looking: LOOK, EXAMINE, INSPECT, LOOK IN/ON/UNDER/BEHIND, directional LOOK
- Items: GET, DROP, INVENTORY, WIELD, UNWIELD, WEAR, REMOVE
- Containers: OPEN, CLOSE
- Interaction: PULL, PUSH, TURN, RUB, TAP, TOUCH, SEARCH, DIG, RECALL
- Communication: SAY ('), WHISPER, YELL, RECITE, EMOTE
- Commerce: BUY, SELL (1482 store items across 20+ shops)
- Social: 60+ emotes (SMILE, BOW, KICK, etc.) with targeted second-person messages
- Info: STATUS, HEALTH, WEALTH, SKILLS, WHO, HELP, ASSIST
- Position: SIT, STAND, KNEEL, LAY
- Roleplay: ACT, EMOTE, RECITE
- READ items with room-scoped descriptions

### Multiplayer
- WebSocket-based real-time gameplay
- Cross-machine coordination via MongoDB Change Streams
- Player presence synced across multiple Fly.io machines
- Room state changes (item state, drops, pickups, script mutations) coordinated
- Emote targeting: second-person messages to target, third-person to room
- Room broadcasts, global broadcasts, GM broadcasts, whispers
- WebSocket reconnection with exponential backoff

### Characters & Auth
- Google OAuth login with 30-day JWT sessions
- Character creation with 8 races, stat rolling, name validation
- Character name uniqueness enforced (no logging into others' characters)
- Session persistence across server restarts
- Starting gear granted via IFENTRY scripts at room 201

### Commerce
- BUY command with store items (STOREITEM parsed from scripts)
- Efficient currency deduction (copper first, then silver, then gold with change)
- SELL command in rooms with BUY_ARMOR/BUY_SKINS/BUY_JEWELRY modifiers
- Price display as gold/silver/copper

### Admin Interface
- Rooms tab: searchable index with detail view, clickable exits, enriched items
- Items tab: searchable index with full properties, type/slot expansion
- Monsters tab: searchable index with combat stats and properties
- Players tab: character detail, GM toggle, account reassignment
- Users tab: account detail, admin toggle, character list
- Logs tab: filterable game event log with hyperlinked players/users
- Exact number match sorting in search results

### Logging
- MongoDB-backed game log with 90-day TTL
- Events: user login/logout, character game enter/exit, character creation, GM grant/revoke
- Log entries include user name, email, account ID
- Admin logs UI with event/player filtering

### Security
- Admin auth required for GM toggle, game world API endpoints
- WebSocket origin validation against frontend URL
- 64KB WebSocket message size limit
- Character name validation (alpha + ' + -, max 20 chars)
- Race/gender input validation
- Command input truncated to 500 characters

### Infrastructure
- Go + gorilla/mux backend, React 19 + TypeScript + Vite + Tailwind 4 frontend
- MongoDB Atlas for persistence
- Fly.io production deployment (2 machines, ord region)
- Custom domain: lofp.metavert.io
- Multi-stage Docker build (14MB production image)
- Separate dev/prod configs
