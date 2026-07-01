package engine

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// --- Summoning spell implementations ---

// castSummonFamiliar handles spell 122 — Summon Familiar.
// Summons a black cat (MNUMBER 7) or raven (MNUMBER 11) as the player's familiar.
func (e *GameEngine) castSummonFamiliar(player *Player) *CommandResult {
	if player.SummonedCreatureID != 0 {
		return &CommandResult{Messages: []string{"You already have a summoned creature. Use COMMAND BEGONE to dismiss it first."}}
	}
	mnumber := []int{7, 11}[rand.Intn(2)]
	return e.spawnSummonedCreature(player, mnumber, true, "A shimmer of magical light coalesces into")
}

// castCallAnimal handles druid spell 504 — Call Animal.
// Summons one of several small animals (cat, falcon, sparrow, owl, raven).
func (e *GameEngine) castCallAnimal(player *Player) *CommandResult {
	if player.SummonedCreatureID != 0 {
		return &CommandResult{Messages: []string{"You already have a summoned creature. Use COMMAND BEGONE to dismiss it first."}}
	}
	animalTypes := []int{7, 8, 9, 10, 11}
	mnumber := animalTypes[rand.Intn(len(animalTypes))]
	return e.spawnSummonedCreature(player, mnumber, true, "You call out to the wilds, and")
}

// castStickToSnake handles druid spell 517 — Stick to Snake.
// Transforms a stick into a viper (MNUMBER 15) that serves the caster.
func (e *GameEngine) castStickToSnake(player *Player) *CommandResult {
	if player.SummonedCreatureID != 0 {
		return &CommandResult{Messages: []string{"You already have a summoned creature. Use COMMAND BEGONE to dismiss it first."}}
	}
	return e.spawnSummonedCreature(player, 15, true, "A stick nearby writhes and transforms into")
}

// castSummonElemental handles conjuration summon spells 106, 107, 108, 109, 123.
// Reagents are consumed at PREPARE time; this function only needs to spawn.
func (e *GameEngine) castSummonElemental(player *Player, spell *SpellDef) *CommandResult {
	if player.SummonedCreatureID != 0 {
		return &CommandResult{Messages: []string{"You already have a summoned creature. Use COMMAND BEGONE to dismiss it first."}}
	}
	player.PreparedSpellReagentArch = 0

	mnumberMap := map[int]int{
		106: 5, // Fire Elemental
		107: 3, // Air Elemental
		108: 6, // Water Elemental
		109: 0, // Gargoyle
		123: 4, // Earth Elemental
	}
	flavorMap := map[int]string{
		106: "A pillar of fire erupts and takes form as",
		107: "The air swirls violently and coalesces into",
		108: "Water rises from nowhere and shapes itself into",
		109: "Stone and shadow merge in a grinding roar into",
		123: "The earth trembles and heaves upward as",
	}

	return e.spawnSummonedCreature(player, mnumberMap[spell.ID], false, flavorMap[spell.ID])
}

// spawnSummonedCreature creates a summoned monster in the player's room, links it to the player,
// and returns a CommandResult with flavor text. isFamiliar enables COMMAND WATCH WILL.
func (e *GameEngine) spawnSummonedCreature(player *Player, mnumber int, isFamiliar bool, flavorPrefix string) *CommandResult {
	def := e.monsters[mnumber]
	if def == nil {
		return &CommandResult{Messages: []string{"The summoning fails — no creature answers your call."}}
	}

	hp := def.Body
	if def.ExtraBody > 0 {
		hp += rand.Intn(def.ExtraBody/2+1) + def.ExtraBody/2
	}

	e.monsterMgr.mu.Lock()
	inst := MonsterInstance{
		ID:           e.monsterMgr.nextID,
		DefNumber:    mnumber,
		RoomNumber:   player.RoomNumber,
		Alive:        true,
		CurrentHP:    hp,
		DefenseBonus: monsterPsiDefenseBonus(def.Disciplines),
		SummonerName: player.FirstName,
		IsSummoned:   true,
		IsFamiliar:   isFamiliar,
		FollowTarget: player.FirstName,
	}
	idx := len(e.monsterMgr.instances)
	e.monsterMgr.instances = append(e.monsterMgr.instances, inst)
	e.monsterMgr.monstersByRoom[player.RoomNumber] = append(e.monsterMgr.monstersByRoom[player.RoomNumber], idx)
	player.SummonedCreatureID = e.monsterMgr.nextID
	e.monsterMgr.nextID++
	e.monsterMgr.mu.Unlock()

	cname := strings.ToLower(FormatMonsterName(def, e.monAdjs))
	carticle := articleFor(cname, def.Unique)

	summoned := fmt.Sprintf("%s %s%s!", flavorPrefix, carticle, cname)
	if isFamiliar {
		return &CommandResult{
			Messages:      []string{summoned, fmt.Sprintf("You have summoned %s%s as your familiar. (COMMAND FOLLOW ME, COMMAND WATCH WILL, COMMAND BEGONE)", carticle, cname)},
			RoomBroadcast: []string{fmt.Sprintf("%s%s appears in a shimmer of magical light.", capArticle(carticle), cname)},
		}
	}
	return &CommandResult{
		Messages:      []string{summoned, fmt.Sprintf("You have summoned %s%s. (COMMAND FOLLOW ME, COMMAND GUARD ME, COMMAND ATTACK <name>, COMMAND BEGONE)", carticle, cname)},
		RoomBroadcast: []string{fmt.Sprintf("%s%s appears in a burst of magical energy.", capArticle(carticle), cname)},
	}
}

// --- COMMAND verb ---

// doCommand handles the COMMAND verb for controlling summoned creatures.
func (e *GameEngine) doCommand(ctx context.Context, player *Player, args []string) *CommandResult {
	_ = ctx
	if player.SummonedCreatureID == 0 {
		return &CommandResult{Messages: []string{"You have no summoned creature to command."}}
	}
	if len(args) == 0 {
		return &CommandResult{Messages: []string{"Command it to do what? (FOLLOW ME, FOLLOW <name>, BEGONE, WATCH WILL, GUARD ME, GUARD <name>, ATTACK <name>)"}}
	}

	// Locate the summoned creature in the manager
	e.monsterMgr.mu.Lock()
	var inst *MonsterInstance
	for i := range e.monsterMgr.instances {
		if e.monsterMgr.instances[i].ID == player.SummonedCreatureID && e.monsterMgr.instances[i].Alive {
			inst = &e.monsterMgr.instances[i]
			break
		}
	}
	if inst == nil {
		player.SummonedCreatureID = 0
		e.monsterMgr.mu.Unlock()
		return &CommandResult{Messages: []string{"Your summoned creature is gone."}}
	}
	defNum := inst.DefNumber
	isFamiliar := inst.IsFamiliar
	instRoom := inst.RoomNumber
	instID := inst.ID
	e.monsterMgr.mu.Unlock()

	def := e.monsters[defNum]
	if def == nil {
		return &CommandResult{Messages: []string{"Your summoned creature seems confused."}}
	}

	cname := strings.ToLower(FormatMonsterName(def, e.monAdjs))
	carticle := articleFor(cname, def.Unique)
	capCname := capArticle(carticle) + cname

	sub := strings.ToLower(args[0])
	rest := ""
	if len(args) > 1 {
		rest = strings.ToLower(strings.Join(args[1:], " "))
	}

	// Determine how the player appears to others (hidden/invisible → "something")
	isHidden := player.Hidden || player.Invisible || player.GMInvis
	sameRoom := instRoom == player.RoomNumber

	// commandBroadcast returns the gaze+says lines to show in the player's room to others.
	commandBroadcast := func(commandText string) []string {
		var msgs []string
		if !isHidden {
			if sameRoom {
				msgs = append(msgs, fmt.Sprintf("%s gazes over at %s%s.", player.FirstName, carticle, cname))
			} else {
				msgs = append(msgs, fmt.Sprintf("%s gazes into the distance.", player.FirstName))
			}
		}
		speakerName := player.FirstName
		if isHidden {
			speakerName = "something"
		}
		msgs = append(msgs, fmt.Sprintf("%s says, \"I command you to %s!\"", speakerName, commandText))
		return msgs
	}

	switch sub {
	case "follow":
		// COMMAND FOLLOW ME — follow the summoner
		// COMMAND FOLLOW <name> — follow another player in the creature's room
		followTarget := player.FirstName
		followLabel := "follow me"
		if rest != "" && rest != "me" {
			// Find the named player in the creature's room
			found := false
			if e.sessions != nil {
				for _, p := range e.sessions.OnlinePlayers() {
					if strings.EqualFold(p.FirstName, rest) && p.RoomNumber == instRoom && !p.Dead {
						followTarget = p.FirstName
						followLabel = "follow " + p.FirstName
						found = true
						break
					}
				}
			}
			if !found {
				return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' where your %s is.", rest, cname)}}
			}
		}
		e.monsterMgr.mu.Lock()
		for i := range e.monsterMgr.instances {
			if e.monsterMgr.instances[i].ID == instID {
				e.monsterMgr.instances[i].FollowTarget = followTarget
				e.monsterMgr.instances[i].WatchMode = false
				e.monsterMgr.instances[i].GuardingPlayers = nil
				break
			}
		}
		e.monsterMgr.mu.Unlock()
		e.setWatching(player.FirstName, 0)
		actionMsg := fmt.Sprintf("%s is now following %s.", capCname, followTarget)
		if sameRoom {
			rb := commandBroadcast(followLabel)
			rb = append(rb, actionMsg)
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("%s acknowledges your command.", capCname)},
				RoomBroadcast: rb,
			}
		}
		if e.localRoomBroadcast != nil {
			e.localRoomBroadcast(instRoom, []string{actionMsg})
		}
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("%s acknowledges your command.", capCname)},
			RoomBroadcast: commandBroadcast(followLabel),
		}

	case "begone":
		// COMMAND BEGONE — dismiss the creature
		e.monsterMgr.mu.Lock()
		for i := range e.monsterMgr.instances {
			if e.monsterMgr.instances[i].ID == instID {
				instRoom = e.monsterMgr.instances[i].RoomNumber
				e.monsterMgr.instances[i].Alive = false
				break
			}
		}
		roomIdx := e.monsterMgr.monstersByRoom[instRoom]
		for i, ri := range roomIdx {
			if ri < len(e.monsterMgr.instances) && e.monsterMgr.instances[ri].ID == instID {
				e.monsterMgr.monstersByRoom[instRoom] = append(roomIdx[:i], roomIdx[i+1:]...)
				break
			}
		}
		if len(e.monsterMgr.monstersByRoom[instRoom]) == 0 {
			delete(e.monsterMgr.monstersByRoom, instRoom)
		}
		e.monsterMgr.mu.Unlock()
		player.SummonedCreatureID = 0
		e.setWatching(player.FirstName, 0)

		dismissMsg := fmt.Sprintf("%s suddenly vanishes in a puff of gray smoke.", capCname)
		rb := commandBroadcast("begone")
		if sameRoom {
			// Creature is here — append the dismissal to the same room broadcast
			rb = append(rb, dismissMsg)
			return &CommandResult{
				Messages:      []string{dismissMsg},
				RoomBroadcast: rb,
			}
		}
		// Creature is elsewhere — send its dismissal to its room
		if e.localRoomBroadcast != nil {
			e.localRoomBroadcast(instRoom, []string{dismissMsg})
		}
		return &CommandResult{
			Messages:      []string{dismissMsg},
			RoomBroadcast: rb,
		}

	case "watch":
		// COMMAND WATCH WILL — familiar watches its room and echoes events to the summoner
		if !isFamiliar {
			return &CommandResult{Messages: []string{"Only familiars can watch a room. Elementals and gargoyles use COMMAND GUARD ME instead."}}
		}
		e.monsterMgr.mu.Lock()
		for i := range e.monsterMgr.instances {
			if e.monsterMgr.instances[i].ID == instID {
				e.monsterMgr.instances[i].WatchMode = true
				e.monsterMgr.instances[i].FollowTarget = ""
				instRoom = e.monsterMgr.instances[i].RoomNumber
				break
			}
		}
		e.monsterMgr.mu.Unlock()
		e.setWatching(player.FirstName, instRoom)
		if instRoom == player.RoomNumber {
			return &CommandResult{Messages: []string{fmt.Sprintf("Your %s will watch this room and relay events to you via ** messages when you move away.", cname)}}
		}
		return &CommandResult{Messages: []string{fmt.Sprintf("Your %s is watching its current room and will relay ** messages to you.", cname)}}

	case "guard":
		// COMMAND GUARD ME / COMMAND GUARD <name>
		if isFamiliar {
			return &CommandResult{Messages: []string{"Familiars cannot guard. Use COMMAND FOLLOW ME to keep them with you."}}
		}
		guardTarget := rest
		if guardTarget == "" || guardTarget == "me" {
			guardTarget = player.FirstName
		}
		if guardTarget != player.FirstName {
			found := false
			if e.sessions != nil {
				for _, p := range e.sessions.OnlinePlayers() {
					if strings.EqualFold(p.FirstName, guardTarget) && p.RoomNumber == instRoom {
						guardTarget = p.FirstName
						found = true
						break
					}
				}
			}
			if !found {
				return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' where your %s is.", rest, cname)}}
			}
		}
		var alreadyGuarding bool
		e.monsterMgr.mu.Lock()
		for i := range e.monsterMgr.instances {
			if e.monsterMgr.instances[i].ID == instID {
				cur := e.monsterMgr.instances[i].GuardingPlayers
				if containsString(cur, guardTarget) {
					// Toggle off — remove from list
					var next []string
					for _, n := range cur {
						if n != guardTarget {
							next = append(next, n)
						}
					}
					e.monsterMgr.instances[i].GuardingPlayers = next
					alreadyGuarding = true
				} else {
					e.monsterMgr.instances[i].GuardingPlayers = append(cur, guardTarget)
					e.monsterMgr.instances[i].FollowTarget = ""
					e.monsterMgr.instances[i].Target = ""
				}
				break
			}
		}
		e.monsterMgr.mu.Unlock()

		if alreadyGuarding {
			guardLabel := "stop guarding " + guardTarget
			if guardTarget == player.FirstName {
				guardLabel = "stop guarding me"
			}
			rb := commandBroadcast(guardLabel)
			if sameRoom {
				rb = append(rb, fmt.Sprintf("%s stops guarding %s.", capCname, guardTarget))
			} else if e.localRoomBroadcast != nil {
				e.localRoomBroadcast(instRoom, []string{fmt.Sprintf("%s stops guarding %s.", capCname, guardTarget)})
			}
			if guardTarget != player.FirstName && e.sendToPlayer != nil {
				e.sendToPlayer(guardTarget, []string{fmt.Sprintf("%s is no longer guarding you.", capCname)})
			}
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("%s acknowledges your command.", capCname)},
				RoomBroadcast: rb,
			}
		}

		guardLabel := "guard " + guardTarget
		if guardTarget == player.FirstName {
			guardLabel = "guard me"
		}
		rb := commandBroadcast(guardLabel)
		if sameRoom {
			rb = append(rb, fmt.Sprintf("%s stands guard over %s.", capCname, guardTarget))
		} else if e.localRoomBroadcast != nil {
			e.localRoomBroadcast(instRoom, []string{fmt.Sprintf("%s stands guard over %s.", capCname, guardTarget)})
		}
		if guardTarget != player.FirstName && e.sendToPlayer != nil {
			e.sendToPlayer(guardTarget, []string{fmt.Sprintf("%s is now guarding you.", capCname)})
		}
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("%s acknowledges your command.", capCname)},
			RoomBroadcast: rb,
		}

	case "attack":
		// COMMAND ATTACK <name> — direct the creature to attack a player or monster
		if isFamiliar {
			return &CommandResult{Messages: []string{"Familiars cannot attack on command. Summon an elemental or gargoyle for combat support."}}
		}
		if rest == "" {
			return &CommandResult{Messages: []string{"Attack who? (COMMAND ATTACK <name>)"}}
		}

		// Check players first
		var targetPlayer *Player
		if e.sessions != nil {
			for _, p := range e.sessions.OnlinePlayers() {
				if strings.EqualFold(p.FirstName, rest) && p.RoomNumber == instRoom && !p.Dead {
					targetPlayer = p
					break
				}
			}
		}
		if targetPlayer != nil && targetPlayer.FirstName == player.FirstName {
			return &CommandResult{Messages: []string{"You cannot command your creature to attack you."}}
		}

		if targetPlayer != nil {
			e.monsterMgr.mu.Lock()
			for i := range e.monsterMgr.instances {
				if e.monsterMgr.instances[i].ID == instID {
					e.monsterMgr.instances[i].Target = targetPlayer.FirstName
					e.monsterMgr.instances[i].MonsterTargetID = 0
					e.monsterMgr.instances[i].GuardingPlayers = nil
					e.monsterMgr.instances[i].FollowTarget = ""
					break
				}
			}
			e.monsterMgr.mu.Unlock()
			attackLabel := "attack " + targetPlayer.FirstName
			rb := commandBroadcast(attackLabel)
			if sameRoom {
				rb = append(rb, fmt.Sprintf("%s moves to attack %s!", capCname, targetPlayer.FirstName))
			} else if e.localRoomBroadcast != nil {
				e.localRoomBroadcast(instRoom, []string{fmt.Sprintf("%s moves to attack %s!", capCname, targetPlayer.FirstName)})
			}
			return &CommandResult{
				Messages:      []string{fmt.Sprintf("%s acknowledges your command.", capCname)},
				RoomBroadcast: rb,
			}
		}

		// Check monsters in the creature's room
		var monsterTargetID int
		var monsterTargetName string
		e.monsterMgr.mu.Lock()
		for i := range e.monsterMgr.instances {
			m := &e.monsterMgr.instances[i]
			if !m.Alive || m.RoomNumber != instRoom || m.ID == instID {
				continue
			}
			mDef := e.monsters[m.DefNumber]
			if mDef == nil {
				continue
			}
			mName := strings.ToLower(FormatMonsterName(mDef, e.monAdjs))
			if strings.Contains(mName, rest) {
				monsterTargetID = m.ID
				monsterTargetName = mName
				break
			}
		}
		if monsterTargetID > 0 {
			for i := range e.monsterMgr.instances {
				if e.monsterMgr.instances[i].ID == instID {
					e.monsterMgr.instances[i].MonsterTargetID = monsterTargetID
					e.monsterMgr.instances[i].Target = ""
					e.monsterMgr.instances[i].GuardingPlayers = nil
					e.monsterMgr.instances[i].FollowTarget = ""
					break
				}
			}
		}
		e.monsterMgr.mu.Unlock()

		if monsterTargetID == 0 {
			return &CommandResult{Messages: []string{fmt.Sprintf("You don't see '%s' where your %s is.", rest, cname)}}
		}
		mArt := articleFor(monsterTargetName, false)
		attackLabel := "attack " + mArt + monsterTargetName
		rb := commandBroadcast(attackLabel)
		if sameRoom {
			rb = append(rb, fmt.Sprintf("%s moves to attack %s%s!", capCname, mArt, monsterTargetName))
		} else if e.localRoomBroadcast != nil {
			e.localRoomBroadcast(instRoom, []string{fmt.Sprintf("%s moves to attack %s%s!", capCname, mArt, monsterTargetName)})
		}
		return &CommandResult{
			Messages:      []string{fmt.Sprintf("%s acknowledges your command.", capCname)},
			RoomBroadcast: rb,
		}

	default:
		return &CommandResult{Messages: []string{"Unknown command. Options: FOLLOW ME, FOLLOW <name>, BEGONE, WATCH WILL, GUARD ME, GUARD <name>, ATTACK <name>"}}
	}
}

// forwardToWatchers sends room broadcast messages to players whose familiar is watching that room.
// Uses e.watching (protected by e.watchMu) — safe to call while monsterMgr.mu is held.
func (e *GameEngine) forwardToWatchers(roomNum int, messages []string) {
	if e.sendToPlayer == nil || e.sessions == nil || len(messages) == 0 {
		return
	}

	e.watchMu.RLock()
	var summoners []string
	for playerName, watchRoom := range e.watching {
		if watchRoom == roomNum {
			summoners = append(summoners, playerName)
		}
	}
	e.watchMu.RUnlock()

	if len(summoners) == 0 {
		return
	}

	prefixed := make([]string, len(messages))
	for i, m := range messages {
		prefixed[i] = "** " + m
	}

	players := e.sessions.OnlinePlayers()
	for _, name := range summoners {
		for _, p := range players {
			if p.FirstName == name && p.RoomNumber != roomNum {
				e.sendToPlayer(name, prefixed)
				break
			}
		}
	}
}

// dismissSummonedCreature removes the player's summoned creature from the world.
// Safe to call without any lock held. Does nothing if the player has no summoned creature.
func (e *GameEngine) dismissSummonedCreature(player *Player) {
	if player.SummonedCreatureID == 0 || e.monsterMgr == nil {
		return
	}
	e.monsterMgr.mu.Lock()
	idx := e.monsterMgr.indexOfID(player.SummonedCreatureID)
	if idx < 0 {
		player.SummonedCreatureID = 0
		e.monsterMgr.mu.Unlock()
		return
	}
	inst := &e.monsterMgr.instances[idx]
	if !inst.Alive {
		player.SummonedCreatureID = 0
		e.monsterMgr.mu.Unlock()
		return
	}

	def := e.monsters[inst.DefNumber]
	roomNum := inst.RoomNumber
	inst.Alive = false
	inst.DeathTime = time.Now()

	roomIdx := e.monsterMgr.monstersByRoom[roomNum]
	for i, ri := range roomIdx {
		if ri < len(e.monsterMgr.instances) && e.monsterMgr.instances[ri].ID == inst.ID {
			e.monsterMgr.monstersByRoom[roomNum] = append(roomIdx[:i], roomIdx[i+1:]...)
			break
		}
	}
	if len(e.monsterMgr.monstersByRoom[roomNum]) == 0 {
		delete(e.monsterMgr.monstersByRoom, roomNum)
	}
	e.monsterMgr.mu.Unlock()

	player.SummonedCreatureID = 0
	e.setWatching(player.FirstName, 0)

	if e.localRoomBroadcast != nil && def != nil {
		cname := strings.ToLower(FormatMonsterName(def, e.monAdjs))
		carticle := capArticle(articleFor(cname, def.Unique))
		texD := def.TextOverrides["TEXD"]
		var msg string
		if texD != "" {
			msg = fmt.Sprintf("%s%s %s", carticle, cname, texD)
		} else {
			msg = fmt.Sprintf("%s%s dissipates as its summoner departs.", carticle, cname)
		}
		e.localRoomBroadcast(roomNum, []string{msg})
	}
}

// clearPlayerFromGuards removes playerName from all summoned creatures' GuardingPlayers lists.
// Call this when a player dies or disconnects so guards no longer reference them.
func (e *GameEngine) clearPlayerFromGuards(playerName string) {
	if e.monsterMgr == nil {
		return
	}
	e.monsterMgr.mu.Lock()
	for i := range e.monsterMgr.instances {
		inst := &e.monsterMgr.instances[i]
		if !inst.IsSummoned || len(inst.GuardingPlayers) == 0 {
			continue
		}
		var next []string
		for _, n := range inst.GuardingPlayers {
			if n != playerName {
				next = append(next, n)
			}
		}
		inst.GuardingPlayers = next
	}
	e.monsterMgr.mu.Unlock()
}
