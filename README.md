# Legends of Future Past

### **[Play it now — free forever — at lofp.metavert.io](https://lofp.metavert.io)**

---

<p align="center">
  <img src="https://upload.wikimedia.org/wikipedia/en/a/a6/LegendsOfFuturePaste_flyer.jpg" alt="Legends of Future Past promotional flyer" width="400">
</p>

I'm [Jon Radoff](https://metavert.io), and I created *Legends of Future Past* in 1992. It was a text-based online multiplayer game — what we called a MUD — that ran for seven years on dial-up terminals, CompuServe, and packet-switching networks like Tymnet. The original engine was written in fully reentrant C, running on a heavily modified [The Major BBS](https://en.wikipedia.org/wiki/The_Major_BBS) platform — a single Pentium PC with 16MB of RAM serving dozens of simultaneous players.

The game shut down on December 31, 1999. The source code is lost.

In 2026, I brought it back.

## What Was Legends of Future Past?

*Legends* was one of the earliest commercial online multiplayer games. *Computer Gaming World* gave it a Special Award for Artistic Excellence in 1993 and called it "a rich, dynamic and lovingly supervised world of the imagination." It introduced one of the first crafting systems in an online game, had a skill-based character system with no class restrictions and no level caps, and featured paid Game Masters who ran live events years before that became standard in the MMO industry.

The game is set in the Shattered Realms of Andor — a world blending fantasy and ancient technology, with eight playable races, five schools of magic, a psionic combat system, and over two thousand rooms to explore. Some of the people who built and ran *Legends* went on to work on *Star Wars Galaxies* and other early MMOs.

You can read more about the game's history on [Wikipedia](https://en.wikipedia.org/wiki/Legends_of_Future_Past) and in [Raph Koster's Online World Timeline](https://www.raphkoster.com/games/the-online-world-timeline/).

## How We Brought It Back

The source code for the game engine is gone — it lived on a single server that was eventually decommissioned. But several former gamemasters, especially David Goodman, had kept copies of the game's script files. This is important: while the engine code is lost, these script files *are* the actual game content. Every room, item, monster, spell, quest trigger, and NPC interaction in the original game was defined in these scripts using a custom scripting language. The world data is authentic — this isn't a reimagining or a recreation from memory, it's the same content players experienced in the 1990s, now running on a new engine. Along with the scripts, we had the GM manual, player documentation, a scripting reference guide, and — crucially — a session capture log from 1996 that recorded actual gameplay with full combat output.

I used [Claude Code](https://claude.ai/claude-code) to reconstruct the game from these artifacts. The process was genuinely collaborative — not just "write me a game," but an iterative archaeological dig through decades-old files. Here's what that looked like:

**Reverse-engineering the script language.** The game world is defined in hundreds of `.SCR` files using a custom language with commands like `IFVAR`, `IFPREVERB`, `ECHO`, `MOVE`, and `CLEARVERB`. There's no formal specification — just a GM scripting guide written in 1998 and the scripts themselves. Claude Code parsed the language by reading the documentation, examining real script files, and handling edge cases like implicit `ENDIF` blocks and the `ELSE` keyword (which turned out to be critical for temple doorway puzzles).

**Matching original combat output.** The 1996 session capture was invaluable. It showed the exact format of combat messages: `[ToHit: 5, Roll: 31] Hit!` followed by `Puny slash to head. [14 Damage]`. From this, we could determine that ToHit is a d100 threshold (low = easy to hit), that damage has severity tiers (Puny through Dazzling), and that weapon types determine the attack verb (swings for swords, thrusts for polearms, slashes for claws). We also discovered weapon clash mechanics (`[Strength: 57, 2d100 Roll: 26]`) and the `[Round: 5 sec]` roundtime format.

**Reconstructing game mechanics from monster stats.** The script files define monster attributes like `ATTACK1 280`, `DEFENSE 200`, `BODY 250`, `STRATEGY 501`, and `SPEED 2`. By looking at the ranges across hundreds of monsters and cross-referencing with the combat capture, we could infer the formulas connecting attack ratings to hit chances and damage output. The `STRATEGY` field turned out to encode seven distinct AI behaviors, from "non-hostile, flee when attacked" (1-100) to "hostile, fight to the death" (501-700).

**Discovering systems from documentation fragments.** The alchemy system came from a file called `alchemy.bin` that turned out to be a plain-text recipe grid written by a player in 1995. The skill build-point costs came from `skills.txt`. The XP progression table came from the last page of the GM manual. Monster spawning rules were reverse-engineered from `MLIST` entries in the scripts — we initially misinterpreted the format as min/max counts, which caused 3,000+ monsters to spawn at once, before realizing the third field was a probability percentage.

**Handling DOS-era quirks.** The original game ran on MS-DOS, which is case-insensitive. Some script filenames were stored in git with different case than the `LEGENDS.CFG` references. This worked fine on macOS but broke silently on Linux Docker builds — the Temple of Amilor (room 592) was missing in production for this reason. We added case-insensitive file resolution to the script loader.

## Why Release This?

Classic games are [disappearing](https://medium.com/mr-plan-publication/classic-games-disappearing-what-it-means-for-gamings-future-2a885dc3febc). The problem is especially bad for online games — when the servers shut down, the game is gone. You can emulate a Super Nintendo cartridge, but you can't emulate a server that no longer exists.

I wanted *Legends* to be available forever, for free, so anyone could play it or study it. The code is released under the MIT License. The server at [lofp.metavert.io](https://lofp.metavert.io) is running today, and you can clone this repo to host your own.

More broadly, I think AI-powered code generation gives us a real shot at preserving online games that would otherwise be lost. If you have the data files, documentation, or even just captured gameplay from an old online game, it may be possible to reconstruct a working server from those artifacts. This project is proof of that concept. I hope it inspires others to try.

## Architecture

| Component | Technology | Path |
|-----------|-----------|------|
| Backend | Go + gorilla/mux + MongoDB | `engine/` |
| Frontend | React 19 + TypeScript + Vite + Tailwind 4 | `frontend/` |
| Scripts | Original 1992-1999 game data | `original/scripts/` |
| Documentation | GM Manual, player docs, session captures | `original/` |

## Running Your Own Server

```bash
# Prerequisites: Go 1.25+, Node.js 22+, MongoDB

git clone https://github.com/jonradoff/lofp.git
cd lofp

# Set up environment variables
cp .env.example .env
# Edit .env with your MONGODB_URI, JWT_SECRET, GOOGLE_CLIENT_ID

./start.sh
# Frontend: http://localhost:4992
# Backend: http://localhost:4993
```

## Credits

**Original Game (1992-1999)** — Copyright (c) 1992-1999 Inner Circle Software / NovaLink USA Corp

| Role | Name |
|------|------|
| Created & Programmed by | Jon Radoff |
| Additional Programming | Ichiro Lambe |
| Co-Producer | Angela Bull |
| Legends Manager | Gary Whitten |
| World Building | Gary Whitten, David Goodman, Tony Spataro, Stacy Jannis, Kevin Jepson, Daniel Brainerd, Michael Hjerppe |
| Documentation | Gary Whitten |
| Quality Assurance | David Goodman, Stacy Jannis |

**2026 Resurrection** — Reimplemented from original script files and documentation by [Jon Radoff](https://metavert.io) using [Claude Code](https://claude.ai/claude-code).

Special thanks to **David Goodman** for preserving the original game materials that made this reconstruction possible.

## License

Released under the [MIT License](LICENSE) — Copyright (c) 2026 Metavert LLC.

## Links

- **Play now**: [lofp.metavert.io](https://lofp.metavert.io)
- **Version Notes**: [lofp.metavert.io/version-notes](https://lofp.metavert.io/version-notes)
- **Wikipedia**: [Legends of Future Past](https://en.wikipedia.org/wiki/Legends_of_Future_Past)
- **Online World Timeline**: [raphkoster.com](https://www.raphkoster.com/games/the-online-world-timeline/)
- **Game Preservation**: [Classic Games Disappearing](https://medium.com/mr-plan-publication/classic-games-disappearing-what-it-means-for-gamings-future-2a885dc3febc)
