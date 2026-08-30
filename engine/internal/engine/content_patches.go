package engine

import "github.com/jonradoff/lofp/internal/gameworld"

// applyContentPatches adds game content that was planned in the original scripts but left
// commented-out or missing — most commonly because the destination area was never built.
// This is called at the end of NewGameEngine after all script content is loaded.
func (e *GameEngine) applyContentPatches() {
	e.applyGrymwoodStatuePatch()
}

// applyGrymwoodStatuePatch wires up the "kiss the statue three times" puzzle in
// Grymwood, dead end (room 123). The IFVERB KISS 0 block was present in spriout.scr
// but commented out because room 170 (the destination) was never built.
//
// Mechanics:
//   - FLAG1 tracks how many times the player has kissed the statue.
//   - On the 3rd kiss the statue animates, sends a gender-specific echo to the room,
//     then teleports the player to room 170 (the tiny tunnels).
//   - AFFECT 170 / AFFECT 123 sequence: after MOVE 170 immediately updates
//     player.RoomNumber, AFFECT 170 makes affectRoom=false (player == sc.Room),
//     then AFFECT 123 makes affectRoom=true so the gender ECHO OTHERS fires directly
//     into room 123 via roomBroadcast.
func (e *GameEngine) applyGrymwoodStatuePatch() {
	// --- Room 170: Grymwood, the tiny tunnels ---
	// Created here because it was never defined in any original script file.
	if e.rooms[170] == nil {
		if ScriptParser != nil {
			const room170Script = `NUMBER 170
NAME Grymwood, the tiny tunnels
*DESCRIPTION_START
The world around you is impossibly, catastrophically large. What should be tangled forest roots and soil is now a cathedral of earth, each root thicker than a house, arching overhead like the vaulted ceiling of some vast underground temple. Strange bioluminescent fungi cling to the walls, casting a faint blue-green glow that is just enough to see by. The smell of rich earth fills your nose. You have been shrunk to the size of a small insect and deposited here — wherever here is. Somewhere far above, almost beyond sight, light filters through what must be a crack in the forest floor.
*DESCRIPTION_END
EXIT ABOVE 123
UNDERGROUND
PARTIAL_DARKNESS
;
`
			if parsed, err := ScriptParser(room170Script, "content_patches"); err == nil {
				for i := range parsed.Rooms {
					r := parsed.Rooms[i]
					e.rooms[r.Number] = &r
				}
			}
		}
	}

	// --- Room 123: add IFVERB KISS 0 script blocks ---
	room := e.rooms[123]
	if room == nil {
		return
	}

	// Don't double-apply if already patched.
	for _, blk := range room.Scripts {
		if blk.Type == "IFVERB" && len(blk.Args) >= 2 &&
			blk.Args[0] == "KISS" && blk.Args[1] == "0" {
			return
		}
	}

	// The IFVERB KISS 0 block increments FLAG1 (the kiss counter) and, on the
	// third kiss, teleports the player to room 170.
	//
	// Kisses 1 and 2 produce no script output, so the default emote fires
	// ("You kiss the statue.") — natural for observers in the room.
	//
	// Execution order on the 3rd kiss (steps run in the order listed below):
	//   ECHO PLAYER voice  — player receives the farewell message
	//   MOVE 170           — sc.MoveTo=170; player.RoomNumber=170 immediately
	//   AFFECT 170         — sc.Room=170; affectRoom=(170!=170)=false
	//   ECHO PLAYER shrink — goes to player
	//   AFFECT 123         — sc.Room=123; affectRoom=(123!=170)=true
	//   IFVAR GEN=0/1 → ECHO OTHERS gender_msg → roomBroadcast(123, msg)
	//
	// Room 123 players see the gender echo (via roomBroadcast) before "[Name] leaves."
	action := func(cmd string, args ...string) gameworld.ScriptStep {
		return gameworld.ScriptStep{Action: &gameworld.ScriptAction{Command: cmd, Args: args}}
	}
	block := func(b gameworld.ScriptBlock) gameworld.ScriptStep {
		return gameworld.ScriptStep{Block: &b}
	}

	kissBlock := gameworld.ScriptBlock{
		Type: "IFVERB",
		Args: []string{"KISS", "0"},
		Body: []gameworld.ScriptStep{
			action("ADD", "FLAG1", "1"),
			block(gameworld.ScriptBlock{
				Type: "IFVAR",
				Args: []string{"FLAG1", "=", "3"},
				Body: []gameworld.ScriptStep{
					action("ECHO", "PLAYER",
						`You hear the stone maiden's voice in your head, "You have freed me, if only for a moment, and as your reward, I am sending you on a great adventure, or at least you'll think so... if you live."%c%cSuddenly, you are jarred by a bolt of energy!%c`),
					action("MOVE", "170"),
					action("AFFECT", "170"),
					action("ECHO", "PLAYER",
						"Your body is wracked with pain and you realize that you are shrinking. How small you will get, you don't know.%cSuddenly the pain stops..."),
					action("AFFECT", "123"),
					block(gameworld.ScriptBlock{
						Type: "IFVAR",
						Args: []string{"GEN", "=", "0"},
						Body: []gameworld.ScriptStep{
							action("ECHO", "OTHERS",
								"As %n lustily kisses the stone maiden for a third time the statue comes to life! Smiling at %n, she points. Suddenly a spark flies from her fingertips and %n disappears.%cThe maiden becomes stone again."),
						},
					}),
					block(gameworld.ScriptBlock{
						Type: "IFVAR",
						Args: []string{"GEN", "=", "1"},
						Body: []gameworld.ScriptStep{
							action("ECHO", "OTHERS",
								"As %n playfully pecks the statue for a third time the statue comes to life! Smiling at %n, she points. Suddenly a spark flies from her fingertips and %n disappears.%cThe maiden becomes stone again."),
						},
					}),
				},
			}),
		},
	}

	room.Scripts = append(room.Scripts, kissBlock)
}
