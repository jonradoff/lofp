# Changelog

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
