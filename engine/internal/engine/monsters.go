package engine

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// MonsterInstance represents a spawned monster in the world.
type MonsterInstance struct {
	ID         int  `json:"id"`
	DefNumber  int  `json:"defNumber"`
	RoomNumber int  `json:"roomNumber"`
	Alive      bool `json:"alive"`
	CurrentHP  int  `json:"currentHP"`
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

// SpawnInitialMonsters populates the world from MonsterList entries.
func (mm *monsterManager) SpawnInitialMonsters(monsterLists []gameworld.MonsterList, monsters map[int]*gameworld.MonsterDef) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

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
			inst := MonsterInstance{
				ID:         mm.nextID,
				DefNumber:  ml.MonsterID,
				RoomNumber: ml.Room,
				Alive:      true,
				CurrentHP:  def.Body,
			}
			idx := len(mm.instances)
			mm.instances = append(mm.instances, inst)
			mm.monstersByRoom[ml.Room] = append(mm.monstersByRoom[ml.Room], idx)
			mm.nextID++
		}
	}
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
	monsters := e.monsterMgr.MonstersInRoom(roomNum)
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
		lines = append(lines, fmt.Sprintf("You also see a %s.", name))
	}
	return lines
}
