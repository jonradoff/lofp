# LoFP — Legends of Future Past

Resurrecting a 1990s MUD from original script files. The original game engine source code is lost; only the content scripts and documentation survive. We reverse-engineer a working game from those.

## Architecture

- **Backend**: Go + gorilla/mux + MongoDB at `engine/`
- **Frontend**: React 19 + TypeScript + Vite + Tailwind 4 at `frontend/`
- **Original Scripts**: `original/scripts/` (read-only reference, 333 .SCR files)
- Frontend dev server: port 4992, Backend server: port 4993
- Start both: `./start.sh`

## Build Check

```sh
cd engine && go build ./...
cd frontend && npx tsc --noEmit
```

## Reference Documentation

These original documents are the primary sources for game mechanics and world data:

| Document | Path | Description |
|----------|------|-------------|
| GM Manual | `original/GM Pages/MANUAL.DOC` | Comprehensive GM reference: combat, stats, items, monsters, skills, XP tables, all GM commands |
| GM Script Guide | `original/GMSCRIPT.DOC` | Script language reference: room/item/monster definitions, IFVERB/IFENTRY/IFSAY blocks, variables |
| Player Manual | `original/legends/LEGENDS.DOC` | Player-facing docs: commands, spells, psionics, skills, races, crafting |
| Session Capture | `original/legends/shirla.cap` | 1996 gameplay session capture — invaluable for combat output format, spell interactions, original message text |
| Script Config | `original/scripts/LEGENDS.CFG` | Master config listing all .SCR files to load in order |
| GM Pages | `original/GM Pages/` | Additional GM reference materials (various .DOC files) |

When implementing new features, always cross-reference these documents and the session capture to match original behavior.

## Multi-Machine Coordination

Production runs multiple Fly.io machines. ALL mutable world state must be coordinated via the MongoDB-backed hub (`engine/internal/hub/`):

- **Messages**: broadcasts, whispers, global announcements → published to `events` collection, delivered via Change Streams
- **Player presence**: WHO list, room occupancy → `presence` collection with TTL heartbeat
- **Room state changes**: item open/close/lock, item drops/pickups, script mutations (vals, itembits) → `room_state_change` events via hub
- **Any new mutable state** added to rooms, items, or global data MUST call `notifyRoomChange()` or publish through the hub, or players on different machines will see inconsistent worlds

## After Server Changes

After making changes to the backend (engine/), restart the Go server:
```sh
kill $(lsof -ti:4993) 2>/dev/null; sleep 1; cd engine && go run cmd/lofp/main.go &
```
Load .env first if needed: `source .env`

## Deploying to Production

**CRITICAL: The Google Client ID MUST be passed as a build arg or Google login will break.**

The frontend uses `VITE_GOOGLE_CLIENT_ID` at build time (baked into the JS bundle by Vite). If it's missing, the "Sign in with Google" button disappears and users see "Authentication Not Configured".

### Deploy command (always use this exact pattern):
```sh
GCID=$(grep GOOGLE_CLIENT_ID .env | cut -d= -f2) && fly deploy --build-arg "VITE_GOOGLE_CLIENT_ID=$GCID"
```

**Why this specific pattern?** The Bash tool's shell does not persist exported variables between commands. `source .env && fly deploy --build-arg VITE_GOOGLE_CLIENT_ID=$GOOGLE_CLIENT_ID` silently passes an empty string because `source` sets but does not export the variable, and the Bash tool may run commands in separate shell contexts. Using `grep | cut` directly extracts the value inline, which is reliable.

If using `--no-cache` to force a fresh build, append it:
```sh
GCID=$(grep GOOGLE_CLIENT_ID .env | cut -d= -f2) && fly deploy --build-arg "VITE_GOOGLE_CLIENT_ID=$GCID" --no-cache
```

### After deploying, verify Google login:
```sh
fly ssh console -a lofp -C "grep -c 718491 /app/static/assets/index-*.js"
```
This should return `1`. If it returns `0`, the build arg was not passed correctly.

### Local dev vs. production:
- **Local**: `.env` file at project root contains `GOOGLE_CLIENT_ID=...` (among other secrets). Vite reads `VITE_GOOGLE_CLIENT_ID` from the environment when running `npm run dev`.
- **Production**: Fly.io secrets store `MONGODB_URI`, `JWT_SECRET`, `RESEND_API_KEY`, `SSH_HOST_KEY` (set via `fly secrets set`). The Google Client ID is NOT a Fly secret — it's a **build arg** because Vite needs it at build time, not runtime.
- **Never commit `.env`** — it contains production secrets. It is in `.gitignore`.

## MUD Client Protocols

Reference: https://wiki.mudlet.org/w/Manual:Supported_Protocols

Telnet server (`engine/internal/api/telnet.go`) implements these protocols:

### GMCP (option 201)
- **Core.Hello** — sent on connect: `{"client":"LoFP","version":"11.1.0"}`
- **Core.Supports.Set** — handled from client to track subscribed packages
- **Char.Vitals** — sent after every command: `bp`, `maxbp`, `mana`, `maxmana`, `psi`, `maxpsi`, `fatigue`, `maxfatigue`, `position`, `conditions`
- **Char.Status** — sent on login: `name`, `fullname`, `race`, `gender`, `level`, `experience`, `gold`, `silver`, `copper`
- **Char.Stats** — sent on login: `strength`, `agility`, `quickness`, `constitution`, `perception`, `willpower`, `empathy`
- **Room.Info** — sent on every room change (powers Mudlet automapper): `num` (int), `name` (string), `area` (string), `environment` (string), `exits` (map direction→room number)

### MXP (option 91)
- Line mode `\033[1z` = secure line (allows `<send>` tags until newline). NOT `\033[4z`.
- Used for clickable exits: `\033[1z<send href="north">north</send>, <send href="east">east</send>`
- `<send>` is a SECURE tag — only works in secure line mode (`\033[1z`)
- MXP output is gated on `tc.mxpEnabled` — plain clients see normal text

### MCCP2 (option 86)
- After negotiation, sends `IAC SB 86 IAC SE` uncompressed, then ALL subsequent output goes through zlib
- **IMPORTANT**: When MCCP2 is active, ALL data including IAC sequences (like WILL ECHO for password mode) MUST go through `t.write()` (the compressor), NOT `t.conn.Write()` (raw). Sending raw bytes corrupts the zlib stream and causes Mudlet to disconnect.

### MSSP (option 70)
- Sends game metadata (name, player count, website, genre, etc.) for MUD directory crawlers

### MSDP (option 69)
- Variable reporting for TinTin++ compatibility
- Handles REPORT subscriptions, pushes CHARACTER_NAME, HEALTH, MANA, ROOM, etc.

### Password Echo Suppression
- `WILL ECHO` / `WONT ECHO` toggle for password fields
- Must be sent through the compressor when MCCP2 is active
- `enterPasswordMode()` before prompt, `exitPasswordMode()` after reading

### NAWS (option 31)
- Window size negotiation, updates `t.width` for `wordWrap()`

## Script Language

The game world is defined in a custom scripting language (documented in `original/GMSCRIPT.DOC`):
- **Rooms**: `NUMBER`, `NAME`, `*DESCRIPTION_START/END`, `EXIT`, `ITEM`, terrain, lighting
- **Items**: `INUMBER`, `NAME` (noun ref), type, weight, volume, substance, worn slots
- **Monsters**: `MNUMBER`, body parts, stats, AI strategy, weapons, spells
- **Events**: `IFVERB/IFPREVERB/IFSAY/IFENTRY/IFVAR...ENDIF` conditional blocks
- **Variables**: Named variables + internal vars (stats, time, flags, item vals)
- Config file: `original/scripts/LEGENDS.CFG` lists all scripts to load in order

## Current State (v11.2.1)

- Script parser loads 2273+ rooms, 1990+ items, 297 monsters with case-insensitive file loading
- Full combat system: original [ToHit/Roll] format, weapon crits/slayers, fatigue, weapon clash
- 60+ spells across 5 schools, 30+ psionic disciplines
- Complete crafting: mining, smelting, forging, weaving, dyeing, foraging, alchemy (32 recipes)
- 36 skills with build point costs, prerequisites, and mechanical effects
- Treasure system: coin drops, weapon/armor/scroll/chest drops scaled by monster TREASURE level
- Monster AI: hostile aggro, flee behavior, special attacks, guard, demand-based spawning
- ELSE branches in script conditionals, all portal types supported
- 150+ emotes, race-specific, submit-gated interactions
- Character soft-delete with admin recovery, unique first names
- Security: rate limiting, connection caps, chat flood protection, HTML sanitization
- WebSocket-based real-time multiplayer
- Admin panel for rooms/items/monsters/players/logs
- Production: Fly.io (1 machine, ord region) at lofp.metavert.io
  - Single machine to avoid multi-machine state desync (monsters, combat, player visibility)
  - Scale to 2+ machines only after implementing full hub-based monster/combat coordination

## Units

| Unit | Code | Path | Description |
|------|------|------|-------------|
| Engine | ENG | `engine` | Go backend: script parser, game engine, command interpreter, MongoDB persistence |
| Frontend | FRONT | `frontend` | React + Tailwind: player UI (text client) and admin interface |
| Scripts | SCR | `original` | Original game script files and documentation (read-only reference) |
