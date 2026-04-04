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
	ID         int       `json:"id"`
	DefNumber  int       `json:"defNumber"`
	RoomNumber int       `json:"roomNumber"`
	Alive      bool      `json:"alive"`
	Sedated    bool      `json:"sedated"`
	CurrentHP  int       `json:"currentHP"`
	Target     string    `json:"-"`
	Searched   bool      `json:"-"` // already searched for loot
	DeathTime  time.Time `json:"-"` // when it died (for corpse decay)
}

// monsterManager handles monster spawning and tracking.
type monsterManager struct {
	mu             sync.RWMutex
	instances      []MonsterInstance
	nextID         int
	monstersByRoom map[int][]int // roomNumber -> slice of instance indices
}

func newMonsterManager() *monsterManager {
	return &monsterManager{
		monstersByRoom: make(map[int][]int),
	}
}

// SpawnInitialMonsters populates the world from MonsterList entries. Returns total spawned.
func (mm *monsterManager) SpawnInitialMonsters(monsterLists []gameworld.MonsterList, monsters map[int]*gameworld.MonsterDef) int {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	total := 0
	for _, ml := range monsterLists {
		def := monsters[ml.MonsterID]
		if def == nil {
			continue
		}
		count := ml.Min
		if ml.Max > ml.Min {
			count = ml.Min + rand.Intn(ml.Max-ml.Min+1)
		}
		for i := 0; i < count; i++ {
			hp := def.Body
			if def.ExtraBody > 0 {
				hp += rand.Intn(def.ExtraBody/2+1) + def.ExtraBody/2 // 50-100% of ExtraBody
			}
			inst := MonsterInstance{
				ID:         mm.nextID,
				DefNumber:  ml.MonsterID,
				RoomNumber: ml.Room,
				Alive:      true,
				CurrentHP:  hp,
			}
			idx := len(mm.instances)
			mm.instances = append(mm.instances, inst)
			mm.monstersByRoom[ml.Room] = append(mm.monstersByRoom[ml.Room], idx)
			mm.nextID++
			total++
		}
	}
	return total
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
func (e *GameEngine) MonsterLookLines(roomNum int) []string {
	if e.monsterMgr == nil {
		return nil
	}
	monsters := e.monsterMgr.AllMonstersInRoom(roomNum)
	if len(monsters) == 0 {
		return nil
	}
	var lines []string
	for _, inst := range monsters {
		def := e.monsters[inst.DefNumber]
		if def == nil {
			continue
		}
		name := FormatMonsterName(def, e.monAdjs)
		deadSuffix := ""
		if !inst.Alive {
			deadSuffix = " (dead)"
		}
		article := articleFor(name, def.Unique)
		lines = append(lines, fmt.Sprintf("You also see %s%s%s.", article, name, deadSuffix))
	}
	return lines
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
			if tick%20 == 0 { // every ~60 seconds
				e.cleanupCorpses()
			}
		}
	}()
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
	if e.monsterMgr == nil || e.roomBroadcast == nil {
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

		// Speed determines action frequency: speed 1 = every tick, speed 3 (default) = every 3 ticks
		speed := def.Speed
		if speed <= 0 {
			speed = 3
		}
		if tick%speed != 0 {
			continue
		}

		// If monster is in combat, process combat instead of wandering/texting
		if inst.Target != "" {
			e.monsterCombatTick(inst, def)
			continue
		}

		name := FormatMonsterName(def, e.monAdjs)

		// Hostile monsters without a target — look for players in room
		if def.Strategy >= 301 && inst.Target == "" {
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

		// Random text (TEX1-4): ~15% chance per action tick
		if rand.Intn(100) < 15 {
			var texts []string
			for _, key := range []string{"TEX1", "TEX2", "TEX3", "TEX4"} {
				if t, ok := def.TextOverrides[key]; ok && t != "" {
					texts = append(texts, t)
				}
			}
			if len(texts) > 0 {
				msg := texts[rand.Intn(len(texts))]
				e.roomBroadcast(inst.RoomNumber, []string{msg})
			}
		}

		// Wandering: ~10% chance per action tick for non-hostile, non-combat monsters
		if def.Strategy < 301 && inst.Target == "" && rand.Intn(100) < 10 {
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
				e.roomBroadcast(inst.RoomNumber, []string{moveText + " " + dirName + "."})
			} else {
				article := "A"
				if len(name) > 0 && strings.ContainsRune("aeiouAEIOU", rune(name[0])) {
					article = "An"
				}
				if def.Unique {
					e.roomBroadcast(inst.RoomNumber, []string{fmt.Sprintf("%s wanders %s.", name, dirName)})
				} else {
					e.roomBroadcast(inst.RoomNumber, []string{fmt.Sprintf("%s %s wanders %s.", article, name, dirName)})
				}
			}

			// Move the monster
			e.monsterMgr.moveMonster(idx, chosen.destID)

			// Arrival message
			entryText := def.TextOverrides["TEXE"]
			if entryText != "" {
				e.roomBroadcast(chosen.destID, []string{entryText})
			} else {
				article := "A"
				if len(name) > 0 && strings.ContainsRune("aeiouAEIOU", rune(name[0])) {
					article = "An"
				}
				if def.Unique {
					e.roomBroadcast(chosen.destID, []string{fmt.Sprintf("%s has arrived.", name)})
				} else {
					e.roomBroadcast(chosen.destID, []string{fmt.Sprintf("%s %s has arrived.", article, name)})
				}
			}
		}
	}
}
