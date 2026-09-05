package engine

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// MonsterInstance represents a spawned monster in the world.
type MonsterInstance struct {
	ID             int                   `json:"id"`
	DefNumber      int                   `json:"defNumber"`
	RoomNumber     int                   `json:"roomNumber"`
	Alive          bool                  `json:"alive"`
	Sedated        bool                  `json:"sedated"`
	Stunned        bool                  `json:"-"` // stunned: skip next combat tick, easier to hit
	Skinned        bool                  `json:"-"` // already skinned
	DefenseBonus   int                   `json:"-"` // from active psi defenses
	CurrentHP      int                   `json:"currentHP"`
	Target         string                `json:"-"`
	Searched       bool                  `json:"-"` // already searched for loot
	DeathTime      time.Time             `json:"-"` // when it died (for corpse decay)
	DamageByPlayer map[bson.ObjectID]int `json:"-"` // tracks damage dealt by players for loot distribution

	CommanderID     string `json:"-"` //player NAME of the summoner/controller (if any)
	ControlType     string `json:"-"` // "SUMMONED" or "CONTROLLED" for player-controlled monsters
	Following       string `json:"-"` //is following a player (for summoned monsters)
	TargetMonsterID int    `json:"-"` //track controlled monster target // -1 means no monster target
	Guarding        string `json:"-"` //creature guarding a player (for summoned monsters)
	Watching        bool   `json:"-"` //creature is watching wich broadcasts the room back to the player on move
}

// monsterManager handles monster spawning and tracking.
type monsterManager struct {
	mu             sync.RWMutex
	instances      []MonsterInstance
	nextID         int
	monstersByRoom map[int][]int     // roomNumber -> slice of instance indices
	roomLastPlayer map[int]time.Time // roomNumber -> last time a player was present
}

func newMonsterManager() *monsterManager {
	return &monsterManager{
		monstersByRoom: make(map[int][]int),
		roomLastPlayer: make(map[int]time.Time),
	}
}

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
	e.monsterMgr.roomLastPlayer[roomNum] = time.Now()
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

		// Spawn up to MaxCount, each with Probability% chance
		spawned := 0
		for existingCount+spawned < ml.MaxCount {
			if ml.Probability > 0 && rand.Intn(100) >= ml.Probability {
				break // failed probability check — stop trying for this entry
			}
			hp := def.Body
			if def.ExtraBody > 0 {
				hp += rand.Intn(def.ExtraBody/2+1) + def.ExtraBody/2
			}
			inst := MonsterInstance{
				ID:              e.monsterMgr.nextID,
				DefNumber:       ml.MonsterID,
				RoomNumber:      roomNum,
				Alive:           true,
				CurrentHP:       hp,
				DefenseBonus:    monsterPsiDefenseBonus(def.Disciplines),
				TargetMonsterID: -1, // no monster target by default
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
func (mm *monsterManager) SpawnOne(defNum, roomNum, hp int) *MonsterInstance {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	inst := MonsterInstance{
		ID:              mm.nextID,
		DefNumber:       defNum,
		RoomNumber:      roomNum,
		Alive:           true,
		CurrentHP:       hp,
		TargetMonsterID: -1, // no monster target by default
	}

	idx := len(mm.instances)
	mm.instances = append(mm.instances, inst)
	mm.monstersByRoom[roomNum] = append(mm.monstersByRoom[roomNum], idx)
	mm.nextID++

	return &mm.instances[idx]
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

func (mm *monsterManager) MarkSkinned(instanceID int) bool {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	for i := range mm.instances {
		if mm.instances[i].ID != instanceID {
			continue
		}

		if mm.instances[i].Skinned {
			return false
		}

		mm.instances[i].Skinned = true
		return true
	}

	return false
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
			}
			// Periodic respawn check near players every ~30 seconds
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

		// Commanded creatures do nothing autonomously unless they
		// have been explicitly ordered to attack another monster.
		if inst.CommanderID != "" &&
			inst.TargetMonsterID < 0 {
			continue
		}

		// Speed determines action frequency:
		// speed 1 = every tick, speed 3 = every 3 ticks.
		speed := def.Speed
		if speed <= 0 {
			speed = 3
		}

		if tick%speed != 0 {
			continue
		}

		// ------------------------------------------------------------
		// COMBAT
		//
		// A monster may now be fighting either:
		//   Target          = player
		//   TargetMonsterID = monster
		// ------------------------------------------------------------

		if inst.Target != "" || inst.TargetMonsterID >= 0 {
			e.monsterCombatTick(inst, def)
			continue
		}

		name := FormatMonsterName(def, e.monAdjs)

		// ------------------------------------------------------------
		// AUTONOMOUS AGGRO
		//
		// Only acquire a player if the monster has NO target at all.
		// ------------------------------------------------------------

		if def.Strategy >= 301 &&
			inst.Target == "" &&
			inst.TargetMonsterID < 0 {

			if e.sessions != nil {
				for _, p := range e.sessions.OnlinePlayers() {
					if p.RoomNumber != inst.RoomNumber ||
						p.Dead ||
						p.Hidden ||
						p.Invisible ||
						p.GMInvis {
						continue
					}

					inst.Target = p.FirstName

					if e.sendToPlayer != nil {
						e.sendToPlayer(
							p.FirstName,
							[]string{
								fmt.Sprintf(
									"A %s snarls and attacks you!",
									name,
								),
							},
						)
					}

					break
				}
			}

			if inst.Target != "" {
				continue // start combat next action tick
			}
		}

		// ------------------------------------------------------------
		// RANDOM MONSTER TEXT
		// ------------------------------------------------------------

		if rand.Intn(100) < 8 {
			var texts []string

			for _, key := range []string{
				"TEX1",
				"TEX2",
				"TEX3",
				"TEX4",
			} {
				if t, ok := def.TextOverrides[key]; ok && t != "" {
					texts = append(texts, t)
				}
			}

			if len(texts) > 0 {
				msg := texts[rand.Intn(len(texts))]

				e.localRoomBroadcast(
					inst.RoomNumber,
					[]string{msg},
				)
			}
		}

		// ------------------------------------------------------------
		// WANDERING
		//
		// Never wander while fighting either a player or monster.
		// ------------------------------------------------------------

		if def.Strategy < 301 &&
			inst.Target == "" &&
			inst.TargetMonsterID < 0 &&
			rand.Intn(100) < 5 {

			room := e.rooms[inst.RoomNumber]
			if room == nil {
				continue
			}

			type exitInfo struct {
				dir    string
				destID int
			}

			var exits []exitInfo

			for dir, destID := range room.Exits {
				if destID <= 0 {
					continue
				}

				// Non-flying monsters can't use vertical exits.
				if strings.EqualFold(dir, "ABOVE") ||
					strings.EqualFold(dir, "UP") {
					continue
				}

				exits = append(
					exits,
					exitInfo{
						dir:    dir,
						destID: destID,
					},
				)
			}

			if len(exits) == 0 {
				continue
			}

			chosen := exits[rand.Intn(len(exits))]

			if !e.moveMonsterDirection(
				idx,
				inst,
				def,
				chosen.dir,
			) {
				continue
			}
		}
	}
}

func (e *GameEngine) moveMonsterDirection(idx int, inst *MonsterInstance, def *gameworld.MonsterDef, dir string) bool {
	room := e.rooms[inst.RoomNumber]
	if room == nil {
		return false
	}

	destID, ok := room.Exits[dir]
	if !ok || destID <= 0 {
		return false
	}

	destRoom := e.rooms[destID]
	if destRoom == nil {
		return false
	}

	name := FormatMonsterName(def, e.monAdjs)

	dirName := directionNames[dir]
	if dirName == "" {
		dirName = strings.ToLower(dir)
	}

	// Departure
	moveText := def.TextOverrides["TEXM"]
	if moveText != "" {
		e.localRoomBroadcast(
			inst.RoomNumber,
			[]string{moveText + " " + dirName + "."},
		)
	} else {
		article := "A"
		if len(name) > 0 && strings.ContainsRune("aeiouAEIOU", rune(name[0])) {
			article = "An"
		}

		if def.Unique {
			e.localRoomBroadcast(
				inst.RoomNumber,
				[]string{fmt.Sprintf("%s wanders %s.", name, dirName)},
			)
		} else {
			e.localRoomBroadcast(
				inst.RoomNumber,
				[]string{fmt.Sprintf("%s %s wanders %s.", article, name, dirName)},
			)
		}
	}

	e.monsterMgr.moveMonster(idx, destID)

	// Arrival
	entryText := def.TextOverrides["TEXE"]
	if entryText != "" {
		e.localRoomBroadcast(destID, []string{entryText})
	} else {
		article := "A"
		if len(name) > 0 && strings.ContainsRune("aeiouAEIOU", rune(name[0])) {
			article = "An"
		}

		if def.Unique {
			e.localRoomBroadcast(
				destID,
				[]string{fmt.Sprintf("%s has arrived.", name)},
			)
		} else {
			e.localRoomBroadcast(
				destID,
				[]string{fmt.Sprintf("%s %s has arrived.", article, name)},
			)
		}
	}

	return true
}
