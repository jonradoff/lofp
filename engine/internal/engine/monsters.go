package engine

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// MonsterInstance represents a spawned monster in the world.
type MonsterInstance struct {
	ID           int       `json:"id"`
	DefNumber    int       `json:"defNumber"`
	RoomNumber   int       `json:"roomNumber"`
	Alive        bool      `json:"alive"`
	Sedated      bool      `json:"sedated"`
	Stunned      bool      `json:"-"` // stunned: skip next combat tick, easier to hit
	Skinned      bool      `json:"-"` // already skinned
	DefenseBonus int       `json:"-"` // from active psi defenses
	CurrentHP  int       `json:"currentHP"`
	Target     string    `json:"-"`
	Searched   bool      `json:"-"` // already searched for loot
	DeathTime  time.Time `json:"-"` // when it died (for corpse decay)

	// Spell status effects
	Sleeping    bool      `json:"-"` // Slumber spell: no attack, no flee
	SleepExpiry time.Time `json:"-"`
	SleepStand  bool      `json:"-"` // woke up but still rising; skip one attack tick
	Webbed      bool      `json:"-"` // Web spell: no attack, no flee
	WebExpiry   time.Time `json:"-"`
	Feared      bool      `json:"-"` // Fear spell: only flees, never attacks
	FearExpiry  time.Time `json:"-"`
	Charmed     bool      `json:"-"` // Charm spell: won't attack the caster
	CharmExpiry time.Time `json:"-"`
	CharmTarget string    `json:"-"` // player name who charmed it
	Tentacled          bool      `json:"-"` // Siryx's Terrible Tentacles: immobilized, takes periodic crushing damage
	TentacleExpiry     time.Time `json:"-"`
	TentacleCasterName string    `json:"-"` // player who cast Tentacles, for death XP attribution on the DOT tick

	// Summoned creature fields (transient)
	SummonerName    string `json:"-"` // name of the player who summoned this creature
	IsSummoned      bool   `json:"-"` // true if summoned, not a regular world spawn
	IsFamiliar      bool   `json:"-"` // true if a familiar (cat/raven/bird/viper) — can WATCH WILL
	WatchMode       bool   `json:"-"` // familiar is watching a room and forwarding events to summoner
	FollowTarget    string   `json:"-"` // name of player this creature follows when they move; "" = not following
	GuardingPlayers []string `json:"-"` // names of players being guarded (for COMMAND GUARD)
	MonsterTargetID int      `json:"-"` // instance ID of a monster this creature is attacking; 0 = none
	ControlExpiry   time.Time `json:"-"` // Control Undead: when control lapses and the creature turns hostile again (zero = never expires)
}

// monsterManager handles monster spawning and tracking.
type monsterManager struct {
	mu             sync.RWMutex
	instances      []MonsterInstance
	nextID         int
	monstersByRoom map[int][]int    // roomNumber -> slice of instance indices
	roomLastPlayer map[int]time.Time // roomNumber -> last time a player was present
	lastSpawnCheck map[int]time.Time // roomNumber -> last time spawnForRoom actually ran its checks
}

func newMonsterManager() *monsterManager {
	return &monsterManager{
		nextID:         1, // 0 is the "no instance" sentinel used by SummonedCreatureID
		monstersByRoom: make(map[int][]int),
		roomLastPlayer: make(map[int]time.Time),
		lastSpawnCheck: make(map[int]time.Time),
	}
}

// spawnCheckCooldown throttles spawnForRoom per room so a group of players
// entering together (each triggers their own applyEntryScripts call) or a
// player rapidly leaving/re-entering only counts as one spawn opportunity
// instead of one per arrival.
const spawnCheckCooldown = 20 * time.Second

// SpawnInitialMonsters is now a no-op. Monsters spawn on demand when players are nearby.
func (mm *monsterManager) SpawnInitialMonsters(monsterLists []gameworld.MonsterList, monsters map[int]*gameworld.MonsterDef) int {
	return 0 // demand-based spawning handles this now
}

// monsterPsiDefenseBonus calculates defense bonus from a monster's psi disciplines.
// Defensive disciplines are considered always-active on monsters.
func monsterPsiDefenseBonus(disciplines []int) int {
	bonus := 0
	for _, d := range disciplines {
		switch d {
		case 9: // Wall of Force +25
			bonus += 25
		case 13: // Force Field +75
			bonus += 75
		case 54: // Psychic Screen +15
			bonus += 15
		case 57: // Psychic Shield +25
			bonus += 25
		case 58: // Psychic Barrier +35
			bonus += 35
		case 63: // Psychic Fortress +50
			bonus += 50
		}
	}
	return bonus
}

// spawnForRoom checks MLIST entries for a room and spawns monsters if needed.
// Called when a player enters a room or during periodic spawn checks.
func (e *GameEngine) spawnForRoom(roomNum int) {
	if e.monsterMgr == nil {
		return
	}

	// Track player presence for unload timer
	e.monsterMgr.mu.Lock()
	now := time.Now()
	e.monsterMgr.roomLastPlayer[roomNum] = now
	if last, ok := e.monsterMgr.lastSpawnCheck[roomNum]; ok && now.Sub(last) < spawnCheckCooldown {
		e.monsterMgr.mu.Unlock()
		return
	}
	e.monsterMgr.lastSpawnCheck[roomNum] = now
	e.monsterMgr.mu.Unlock()

	// Look up the room's monster group ID
	room := e.rooms[roomNum]
	if room == nil {
		return
	}
	groupID := room.MonsterGroup
	if groupID == 0 {
		return
	}

	// Don't spawn monsters in sky/above rooms — only flying players can reach these
	if room.Terrain == "ABOVE" || room.Terrain == "SKY" {
		return
	}

	// Check MLIST entries matching this room's monster group
	for _, ml := range e.monsterLists {
		if ml.Room != groupID {
			continue
		}
		def := e.monsters[ml.MonsterID]
		if def == nil {
			continue
		}

		// Count alive monsters of this type already in the room
		e.monsterMgr.mu.Lock()
		existingCount := 0
		for _, idx := range e.monsterMgr.monstersByRoom[roomNum] {
			if idx < len(e.monsterMgr.instances) {
				inst := &e.monsterMgr.instances[idx]
				if inst.Alive && inst.DefNumber == ml.MonsterID {
					existingCount++
				}
			}
		}

		// Attempt at most one spawn per check so population builds up gradually
		// toward MaxCount across repeated entries/ticks, rather than filling
		// the room in a single burst the moment a player first arrives.
		spawned := 0
		if existingCount < ml.MaxCount && (ml.Probability <= 0 || rand.Intn(100) < ml.Probability) {
			hp := def.Body
			if def.ExtraBody > 0 {
				hp += rand.Intn(def.ExtraBody/2+1) + def.ExtraBody/2
			}
			inst := MonsterInstance{
				ID:           e.monsterMgr.nextID,
				DefNumber:    ml.MonsterID,
				RoomNumber:   roomNum,
				Alive:        true,
				CurrentHP:    hp,
				DefenseBonus: monsterPsiDefenseBonus(def.Disciplines),
			}
			idx := len(e.monsterMgr.instances)
			e.monsterMgr.instances = append(e.monsterMgr.instances, inst)
			e.monsterMgr.monstersByRoom[roomNum] = append(e.monsterMgr.monstersByRoom[roomNum], idx)
			e.monsterMgr.nextID++
			spawned++
		}
		e.monsterMgr.mu.Unlock()

		if spawned > 0 {
			name := FormatMonsterName(def, e.monAdjs)
			genText := def.TextOverrides["TEXG"]
			if genText != "" && e.localRoomBroadcast != nil {
				e.localRoomBroadcast(roomNum, []string{genText})
			} else if spawned == 1 && e.localRoomBroadcast != nil {
				article := articleFor(name, def.Unique)
				e.localRoomBroadcast(roomNum, []string{fmt.Sprintf("%s%s appears.", capArticle(article), name)})
			}
		}
	}
}

// SpawnOne creates a single monster instance in a room. hp should include ExtraBody.
func (mm *monsterManager) SpawnOne(defNum, roomNum, hp int) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	inst := MonsterInstance{ID: mm.nextID, DefNumber: defNum, RoomNumber: roomNum, Alive: true, CurrentHP: hp}
	idx := len(mm.instances)
	mm.instances = append(mm.instances, inst)
	mm.monstersByRoom[roomNum] = append(mm.monstersByRoom[roomNum], idx)
	mm.nextID++
}

// lastSpawnedID returns the ID of the most recently spawned monster.
func (mm *monsterManager) lastSpawnedID() int {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	if len(mm.instances) == 0 {
		return -1
	}
	return mm.instances[len(mm.instances)-1].ID
}

// SetSedated sets the sedated state of a monster by ID.
func (mm *monsterManager) SetSedated(id int, sedated bool) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	for i := range mm.instances {
		if mm.instances[i].ID == id {
			mm.instances[i].Sedated = sedated
			return
		}
	}
}

// ClearRoom removes all monsters from a room.
func (mm *monsterManager) ClearRoom(roomNum int) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	for _, idx := range mm.monstersByRoom[roomNum] {
		if idx < len(mm.instances) {
			mm.instances[idx].Alive = false
		}
	}
	delete(mm.monstersByRoom, roomNum)
}

// MonstersInRoom returns alive monster instances in a given room.
func (mm *monsterManager) MonstersInRoom(roomNum int) []MonsterInstance {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	indices := mm.monstersByRoom[roomNum]
	var result []MonsterInstance
	for _, idx := range indices {
		if idx < len(mm.instances) && mm.instances[idx].Alive {
			result = append(result, mm.instances[idx])
		}
	}
	return result
}

// AllMonstersInRoom returns all monster instances in a room (alive and dead).
func (mm *monsterManager) AllMonstersInRoom(roomNum int) []MonsterInstance {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	indices := mm.monstersByRoom[roomNum]
	var result []MonsterInstance
	for _, idx := range indices {
		if idx < len(mm.instances) {
			result = append(result, mm.instances[idx])
		}
	}
	return result
}

// moveMonster moves a monster instance to a new room. Must be called under lock.
func (mm *monsterManager) moveMonster(idx int, newRoom int) {
	oldRoom := mm.instances[idx].RoomNumber
	mm.instances[idx].RoomNumber = newRoom

	// Remove from old room index
	oldIndices := mm.monstersByRoom[oldRoom]
	for i, oidx := range oldIndices {
		if oidx == idx {
			mm.monstersByRoom[oldRoom] = append(oldIndices[:i], oldIndices[i+1:]...)
			break
		}
	}
	if len(mm.monstersByRoom[oldRoom]) == 0 {
		delete(mm.monstersByRoom, oldRoom)
	}

	// Add to new room index
	mm.monstersByRoom[newRoom] = append(mm.monstersByRoom[newRoom], idx)
}

// FormatMonsterName builds a display name for a monster definition.
func FormatMonsterName(def *gameworld.MonsterDef, monAdjs map[int]string) string {
	name := def.Name
	if def.Adjective > 0 {
		if adj, ok := monAdjs[def.Adjective]; ok {
			name = adj + " " + name
		}
	}
	return name
}

// MonsterLookLines returns the lines to append to a room look showing monsters.
// Condenses multiple monsters into a single "You also see" line.
func (e *GameEngine) MonsterLookLines(roomNum int) []string {
	if e.monsterMgr == nil {
		return nil
	}
	monsters := e.monsterMgr.AllMonstersInRoom(roomNum)
	if len(monsters) == 0 {
		return nil
	}
	var names []string
	for _, inst := range monsters {
		def := e.monsters[inst.DefNumber]
		if def == nil {
			continue
		}
		name := FormatMonsterName(def, e.monAdjs)
		if !inst.Alive {
			name += " (dead)"
		} else if inst.Sleeping {
			name += " (sleeping)"
		} else if inst.Webbed {
			name += " (webbed)"
		} else if inst.Tentacled {
			name += " (entangled)"
		} else if inst.Feared {
			name += " (cowering)"
		}
		article := articleFor(name, def.Unique)
		names = append(names, article+name)
	}
	if len(names) == 0 {
		return nil
	}
	// Join with commas and "and": "a priest, a trader and a merchant"
	var joined string
	if len(names) == 1 {
		joined = names[0]
	} else {
		joined = strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
	}
	return []string{"You also see " + joined + "."}
}

// directionNames maps exit keys to direction words for monster movement text.
var directionNames = map[string]string{
	"N": "north", "S": "south", "E": "east", "W": "west",
	"NE": "northeast", "NW": "northwest", "SE": "southeast", "SW": "southwest",
	"U": "up", "D": "down", "O": "out",
}

// StartMonsterLoop starts the background goroutine for monster behavior.
// Monsters emit random text (TEX1-4) and wander between rooms based on Speed.
func (e *GameEngine) StartMonsterLoop() {
	go func() {
		tick := 0
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			tick++
			e.monsterTick(tick)
			// Corpse decay: remove dead monsters after 60 seconds
			if tick%20 == 0 {
				e.cleanupCorpses()
				e.tentacleDamageTick()
			}
			// Periodic respawn check near players every ~30 seconds.
			// (This was temporarily slowed to ~105s while chasing the Crescent
			// muldragun over-spawning, but that turned out to be caused by
			// real bugs — duplicate MLIST entries loaded from two script
			// files, and spawnForRoom being called once per group member on
			// entry — both fixed at the root now. Slowing this shared,
			// global tick was an unnecessary blanket nerf on top of that and
			// made every other monster group in the game respawn too
			// slowly, so it's reverted back to its original rate.)
			if tick%10 == 0 {
				e.respawnNearPlayers()
			}
			// Unload distant monsters every ~2 minutes
			if tick%40 == 0 {
				e.unloadDistantMonsters()
			}
		}
	}()
}

// respawnNearPlayers spawns monsters in rooms where players are present.
func (e *GameEngine) respawnNearPlayers() {
	if e.sessions == nil || e.monsterMgr == nil {
		return
	}
	// Get all rooms with online players
	roomSet := make(map[int]bool)
	now := time.Now()
	for _, p := range e.sessions.OnlinePlayers() {
		if !p.Dead && !p.GMInvis {
			roomSet[p.RoomNumber] = true
		}
	}
	// Track player presence and spawn
	e.monsterMgr.mu.Lock()
	for roomNum := range roomSet {
		e.monsterMgr.roomLastPlayer[roomNum] = now
	}
	e.monsterMgr.mu.Unlock()

	for roomNum := range roomSet {
		e.spawnForRoom(roomNum)
	}
}

// unloadDistantMonsters removes alive monsters from rooms where no player
// has been for over 3 minutes. ETERNAL monsters are never unloaded.
func (e *GameEngine) unloadDistantMonsters() {
	if e.monsterMgr == nil || e.sessions == nil {
		return
	}

	// Build set of rooms with players
	playerRooms := make(map[int]bool)
	for _, p := range e.sessions.OnlinePlayers() {
		if !p.Dead {
			playerRooms[p.RoomNumber] = true
		}
	}

	e.monsterMgr.mu.Lock()
	defer e.monsterMgr.mu.Unlock()

	now := time.Now()
	unloaded := 0

	for i := range e.monsterMgr.instances {
		inst := &e.monsterMgr.instances[i]
		if !inst.Alive {
			continue
		}

		// Don't unload if players are in the room
		if playerRooms[inst.RoomNumber] {
			continue
		}

		// Don't unload ETERNAL monsters
		def := e.monsters[inst.DefNumber]
		if def != nil && def.Eternal {
			continue
		}

		// Don't unload summoned/controlled creatures — they persist until their
		// summoner dismisses them, they die, or the summoner dies/disconnects.
		// Silently unloading them here left players with no explanation for why
		// a spectral warrior or elemental they'd left guarding/behind vanished.
		if inst.IsSummoned {
			continue
		}

		// Don't unload if room had a player recently (within 3 minutes)
		if lastSeen, ok := e.monsterMgr.roomLastPlayer[inst.RoomNumber]; ok {
			if now.Sub(lastSeen) < 3*time.Minute {
				continue
			}
		}

		// Unload: mark as dead and remove from room tracking
		inst.Alive = false
		roomIndices := e.monsterMgr.monstersByRoom[inst.RoomNumber]
		for j, idx := range roomIndices {
			if idx == i {
				e.monsterMgr.monstersByRoom[inst.RoomNumber] = append(roomIndices[:j], roomIndices[j+1:]...)
				break
			}
		}
		unloaded++
	}

	if unloaded > 0 {
		e.Events.Publish("monster", fmt.Sprintf("Unloaded %d distant monsters", unloaded))
	}
}

// cleanupCorpses removes dead monster instances that have been dead for > 60 seconds.
func (e *GameEngine) cleanupCorpses() {
	if e.monsterMgr == nil {
		return
	}
	e.monsterMgr.mu.Lock()
	defer e.monsterMgr.mu.Unlock()

	now := time.Now()
	for i := range e.monsterMgr.instances {
		inst := &e.monsterMgr.instances[i]
		if !inst.Alive && !inst.DeathTime.IsZero() && now.Sub(inst.DeathTime) > 60*time.Second {
			// Remove from room index
			roomIndices := e.monsterMgr.monstersByRoom[inst.RoomNumber]
			for j, idx := range roomIndices {
				if idx == i {
					e.monsterMgr.monstersByRoom[inst.RoomNumber] = append(roomIndices[:j], roomIndices[j+1:]...)
					break
				}
			}
			// Mark death time as zero so we don't process again
			inst.DeathTime = time.Time{}
		}
	}
}

// tentacleDamageTick applies Siryx's Terrible Tentacles' once-a-minute
// crushing damage to every monster currently held by tentacles. Runs on the
// same ~60-second cadence as corpse decay (StartMonsterLoop, tick%20==0).
// The immobilize status itself (Tentacled/TentacleExpiry) is set at cast
// time and expired in monsterCombatTick, same as Webbed/Sleeping/Feared.
func (e *GameEngine) tentacleDamageTick() {
	if e.monsterMgr == nil {
		return
	}
	spell := FindSpellByID(134)
	if spell == nil {
		return
	}

	type tentacleKill struct {
		inst MonsterInstance
		def  *gameworld.MonsterDef
	}
	var kills []tentacleKill

	e.monsterMgr.mu.Lock()
	for i := range e.monsterMgr.instances {
		inst := &e.monsterMgr.instances[i]
		if !inst.Alive || !inst.Tentacled {
			continue
		}
		def := e.monsters[inst.DefNumber]
		if def == nil {
			continue
		}

		dmg := rand.Intn(spell.DmgMax-spell.DmgMin+1) + spell.DmgMin
		if level, ok := def.Immunities[elementalImmunityType(spell.DmgType)]; ok {
			dmg = applyImmunity(dmg, level)
		}
		if dmg <= 0 {
			continue
		}

		if e.localRoomBroadcast != nil {
			name := strings.ToLower(FormatMonsterName(def, e.monAdjs))
			article := articleFor(name, def.Unique)
			crushLine := fmt.Sprintf("The tentacles crush %s%s!", article, name)
			dmgLine := fmt.Sprintf(" %s %s to %s. [%d Damage]", damageSeverity(dmg), spellDmgNoun(spell.DmgType), randomBodyPart(def.BodyType), dmg)
			e.localRoomBroadcast(inst.RoomNumber, []string{crushLine, dmgLine})
		}

		inst.CurrentHP -= dmg
		if inst.CurrentHP <= 0 {
			inst.Alive = false
			inst.CurrentHP = 0
			inst.DeathTime = time.Now()
			inst.Tentacled = false
			kills = append(kills, tentacleKill{inst: *inst, def: def})
		}
	}
	e.monsterMgr.mu.Unlock()

	for _, k := range kills {
		name := strings.ToLower(FormatMonsterName(k.def, e.monAdjs))
		if e.localRoomBroadcast != nil {
			deathText := k.def.TextOverrides["TEXD"]
			if deathText != "" {
				e.localRoomBroadcast(k.inst.RoomNumber, []string{fmt.Sprintf("A %s %s", name, deathText)})
			} else {
				e.localRoomBroadcast(k.inst.RoomNumber, []string{fmt.Sprintf("A %s collapses, dead!", name)})
			}
		}
		if e.sessions == nil || k.inst.TentacleCasterName == "" {
			continue
		}
		for _, p := range e.sessions.OnlinePlayers() {
			if p.FirstName == k.inst.TentacleCasterName {
				e.handleMonsterDeath(p, &k.inst, k.def)
				break
			}
		}
	}
}

func (e *GameEngine) monsterTick(tick int) {
	if e.monsterMgr == nil || e.localRoomBroadcast == nil {
		return
	}

	e.monsterMgr.mu.Lock()
	defer e.monsterMgr.mu.Unlock()

	for idx := range e.monsterMgr.instances {
		inst := &e.monsterMgr.instances[idx]
		if !inst.Alive || inst.Sedated {
			continue
		}

		def := e.monsters[inst.DefNumber]
		if def == nil {
			continue
		}

		// Control Undead expiry: the creature breaks free of its bonds and turns hostile again.
		if inst.IsSummoned && !inst.ControlExpiry.IsZero() && time.Now().After(inst.ControlExpiry) {
			name := strings.ToLower(FormatMonsterName(def, e.monAdjs))
			summoner := inst.SummonerName
			inst.IsSummoned = false
			inst.SummonerName = ""
			inst.FollowTarget = ""
			inst.GuardingPlayers = nil
			inst.MonsterTargetID = 0
			inst.ControlExpiry = time.Time{}
			if e.localRoomBroadcast != nil {
				e.localRoomBroadcast(inst.RoomNumber, []string{fmt.Sprintf("The %s's eyes flare with malevolent hunger as it breaks free of its bonds!", name)})
			}
			if summoner != "" {
				if e.sendToPlayer != nil {
					e.sendToPlayer(summoner, []string{fmt.Sprintf("Your control over the %s fades. It turns hostile!", name)})
				}
				if e.sessions != nil {
					for _, p := range e.sessions.OnlinePlayers() {
						if p.FirstName == summoner {
							p.SummonedCreatureID = 0
							break
						}
					}
				}
			}
			continue
		}

		// Speed determines action frequency: speed 1 = every tick, speed 3 (default) = every 3 ticks
		speed := def.Speed
		if speed <= 0 {
			speed = 3
		}
		if tick%speed != 0 {
			continue
		}

		if inst.Target != "" || inst.MonsterTargetID > 0 {
			e.monsterCombatTick(inst, def)
			continue
		}

		name := FormatMonsterName(def, e.monAdjs)

		// Hostile monsters without a target — look for players in room (skip summoned creatures)
		if def.Strategy >= 301 && inst.Target == "" && !inst.IsSummoned {
			if e.sessions != nil {
				for _, p := range e.sessions.OnlinePlayers() {
					if p.RoomNumber == inst.RoomNumber && !p.Dead && !p.Hidden && !p.GMInvis {
						inst.Target = p.FirstName
						if e.sendToPlayer != nil {
							e.sendToPlayer(p.FirstName, []string{fmt.Sprintf("A %s snarls and attacks you!", name)})
						}
						break
					}
				}
			}
			if inst.Target != "" {
				continue // start combat next tick
			}
		}

		// Random text (TEX1-4): ~8% chance per action tick (skip summoned creatures)
		if !inst.IsSummoned && rand.Intn(100) < 8 {
			var texts []string
			for _, key := range []string{"TEX1", "TEX2", "TEX3", "TEX4"} {
				if t, ok := def.TextOverrides[key]; ok && t != "" {
					texts = append(texts, t)
				}
			}
			if len(texts) > 0 {
				msg := texts[rand.Intn(len(texts))]
				e.localRoomBroadcast(inst.RoomNumber, []string{msg})
			}
		}

		// Wandering: ~5% chance per action tick for non-hostile, non-combat monsters (skip summoned)
		if def.Strategy < 301 && inst.Target == "" && !inst.IsSummoned && rand.Intn(100) < 5 {
			room := e.rooms[inst.RoomNumber]
			if room == nil {
				continue
			}

			// Collect valid exits
			type exitInfo struct {
				dir    string
				destID int
			}
			var exits []exitInfo
			for dir, destID := range room.Exits {
				if destID > 0 {
					// Non-flying monsters can't use ABOVE exits
					if strings.EqualFold(dir, "ABOVE") || strings.EqualFold(dir, "UP") {
						continue
					}
					exits = append(exits, exitInfo{dir, destID})
				}
			}
			if len(exits) == 0 {
				continue
			}

			// Pick a random exit
			chosen := exits[rand.Intn(len(exits))]
			destRoom := e.rooms[chosen.destID]
			if destRoom == nil {
				continue
			}

			dirName := directionNames[chosen.dir]
			if dirName == "" {
				dirName = strings.ToLower(chosen.dir)
			}

			// Departure message
			moveText := def.TextOverrides["TEXM"]
			if moveText != "" {
				e.localRoomBroadcast(inst.RoomNumber, []string{moveText + " " + dirName + "."})
			} else {
				article := "A"
				if len(name) > 0 && strings.ContainsRune("aeiouAEIOU", rune(name[0])) {
					article = "An"
				}
				if def.Unique {
					e.localRoomBroadcast(inst.RoomNumber, []string{fmt.Sprintf("%s wanders %s.", name, dirName)})
				} else {
					e.localRoomBroadcast(inst.RoomNumber, []string{fmt.Sprintf("%s %s wanders %s.", article, name, dirName)})
				}
			}

			// Move the monster
			e.monsterMgr.moveMonster(idx, chosen.destID)

			// Arrival message
			entryText := def.TextOverrides["TEXE"]
			if entryText != "" {
				e.localRoomBroadcast(chosen.destID, []string{entryText})
			} else {
				article := "A"
				if len(name) > 0 && strings.ContainsRune("aeiouAEIOU", rune(name[0])) {
					article = "An"
				}
				if def.Unique {
					e.localRoomBroadcast(chosen.destID, []string{fmt.Sprintf("%s has arrived.", name)})
				} else {
					e.localRoomBroadcast(chosen.destID, []string{fmt.Sprintf("%s %s has arrived.", article, name)})
				}
			}
		}
	}
}
