package engine

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/metavert-io/lofp/internal/gameworld"
	"go.mongodb.org/mongo-driver/v2/bson"
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
	Restrained     bool                  `json:"-"` // webbed/ snared/ etc
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

	PreparedSpell int
	SpellReadyAt  time.Time
	Mana          int

	// Temporary spell/stat modifiers
	ActiveStatEffects []StatEffect `bson:"activeStatEffects,omitempty" json:"activeStatEffects,omitempty"`
}

// monsterManager handles monster spawning and tracking.
type monsterManager struct {
	mu             sync.RWMutex
	instances      []MonsterInstance
	nextID         int
	monstersByRoom map[int][]int     // roomNumber -> slice of instance indices
	roomLastPlayer map[int]time.Time // roomNumber -> last time a player was present

	// Number of 5-second generation ticks since this room last
	// successfully generated a monster.
	roomSpawnTicks map[int]int
}

func newMonsterManager() *monsterManager {
	return &monsterManager{
		monstersByRoom: make(map[int][]int),
		roomLastPlayer: make(map[int]time.Time),
		roomSpawnTicks: make(map[int]int),
	}
}

// monsterSpawnChance returns the chance that a room will generate
// one monster on the current 5-second generation tick.
//
// This is an experimental reconstruction based on historical play logs.
func monsterSpawnChance(ticksSinceSpawn int) int {
	switch ticksSinceSpawn {
	case 1:
		return 0
	case 2:
		return 15
	case 3:
		return 30
	case 4:
		return 45
	default:
		return 0
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
				Mana:            def.Mana,
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

func (e *GameEngine) generateMonsterForRoom(roomNum int) {
	log.Printf("[SPAWN] generateMonsterForRoom called for room %d", roomNum)

	if e.monsterMgr == nil {
		log.Printf("[SPAWN] room %d: no monster manager", roomNum)
		return
	}

	room := e.rooms[roomNum]
	if room == nil {
		log.Printf("[SPAWN] room %d: room not found", roomNum)
		return
	}

	if room.MonsterGroup == 0 {
		log.Printf("[SPAWN] room %d: no monster group", roomNum)
		return
	}

	if room.Terrain == "ABOVE" || room.Terrain == "SKY" {
		log.Printf("[SPAWN] room %d: terrain %s does not allow spawning", roomNum, room.Terrain)
		return
	}

	groupID := room.MonsterGroup
	mm := e.monsterMgr

	mm.mu.Lock()

	// ------------------------------------------------------------
	// GENERATION CYCLE
	//
	// Each occupied room gets one generation tick every 5 seconds:
	//
	//   Tick 1:  0%
	//   Tick 2: 25%
	//   Tick 3: 55%
	//   Tick 4: 65%
	//
	// A successful spawn resets the cycle.
	// If tick 4 fails, the cycle also resets with no monster.
	// ------------------------------------------------------------

	mm.roomSpawnTicks[roomNum]++

	ticks := mm.roomSpawnTicks[roomNum]
	chance := monsterSpawnChance(ticks)

	log.Printf(
		"[SPAWN] room %d group %d: cycle tick=%d chance=%d%%",
		roomNum,
		groupID,
		ticks,
		chance,
	)

	// Tick 1 deliberately cannot generate anything.
	if chance <= 0 {
		// Defensive reset in case the counter somehow gets outside
		// the expected 1-4 range.
		if ticks >= 4 {
			mm.roomSpawnTicks[roomNum] = 0
		}

		mm.mu.Unlock()
		return
	}

	chanceRoll := rand.Intn(100)

	if chanceRoll >= chance {
		log.Printf(
			"[SPAWN] room %d: generation roll %d failed against %d%%",
			roomNum,
			chanceRoll,
			chance,
		)

		// Tick 4 is the final attempt in this cycle.
		// Failure here means this cycle produces no monster.
		if ticks >= 4 {
			log.Printf(
				"[SPAWN] room %d group %d: generation cycle ended with no monster",
				roomNum,
				groupID,
			)

			mm.roomSpawnTicks[roomNum] = 0
		}

		mm.mu.Unlock()
		return
	}

	log.Printf(
		"[SPAWN] room %d: generation roll %d PASSED against %d%%",
		roomNum,
		chanceRoll,
		chance,
	)

	// ------------------------------------------------------------
	// BUILD ELIGIBLE MLIST CANDIDATES
	//
	// Probability is used as relative selection weight.
	// MaxCount is the maximum number of that monster type
	// currently alive in this room.
	// ------------------------------------------------------------

	type candidate struct {
		monsterID int
		weight    int
	}

	var candidates []candidate
	totalWeight := 0

	for _, ml := range e.monsterLists {
		if ml.Room != groupID {
			continue
		}

		def := e.monsters[ml.MonsterID]
		if def == nil {
			log.Printf(
				"[SPAWN] room %d: MLIST monster %d has no definition",
				roomNum,
				ml.MonsterID,
			)
			continue
		}

		existingCount := 0

		for _, idx := range mm.monstersByRoom[roomNum] {
			if idx < 0 || idx >= len(mm.instances) {
				continue
			}

			inst := &mm.instances[idx]

			if inst.Alive && inst.DefNumber == ml.MonsterID {
				existingCount++
			}
		}

		log.Printf(
			"[SPAWN] room %d: monster=%d weight=%d cap=%d alive=%d",
			roomNum,
			ml.MonsterID,
			ml.Probability,
			ml.MaxCount,
			existingCount,
		)

		if existingCount >= ml.MaxCount {
			continue
		}

		if ml.Probability <= 0 {
			continue
		}

		candidates = append(
			candidates,
			candidate{
				monsterID: ml.MonsterID,
				weight:    ml.Probability,
			},
		)

		totalWeight += ml.Probability
	}

	// The generation roll succeeded, but there is nothing eligible
	// to generate because everything is capped or invalid.
	if len(candidates) == 0 || totalWeight <= 0 {
		log.Printf(
			"[SPAWN] room %d group %d: no eligible MLIST candidates",
			roomNum,
			groupID,
		)

		mm.roomSpawnTicks[roomNum] = 0
		mm.mu.Unlock()
		return
	}

	// ------------------------------------------------------------
	// WEIGHTED MLIST SELECTION
	// ------------------------------------------------------------

	weightRoll := rand.Intn(totalWeight)
	originalWeightRoll := weightRoll

	chosenID := 0

	for _, candidate := range candidates {
		if weightRoll < candidate.weight {
			chosenID = candidate.monsterID
			break
		}

		weightRoll -= candidate.weight
	}

	if chosenID == 0 {
		log.Printf(
			"[SPAWN] room %d: weighted selection failed unexpectedly",
			roomNum,
		)

		mm.roomSpawnTicks[roomNum] = 0
		mm.mu.Unlock()
		return
	}

	def := e.monsters[chosenID]
	if def == nil {
		log.Printf(
			"[SPAWN] room %d: selected monster %d has no definition",
			roomNum,
			chosenID,
		)

		mm.roomSpawnTicks[roomNum] = 0
		mm.mu.Unlock()
		return
	}

	log.Printf(
		"[SPAWN] room %d: weight roll %d/%d selected monster %d (%s)",
		roomNum,
		originalWeightRoll,
		totalWeight,
		chosenID,
		FormatMonsterName(def, e.monAdjs),
	)

	// ------------------------------------------------------------
	// CREATE ONE MONSTER
	// ------------------------------------------------------------

	hp := def.Body

	if def.ExtraBody > 0 {
		hp += rand.Intn(def.ExtraBody/2+1) + def.ExtraBody/2
	}

	inst := MonsterInstance{
		ID:              mm.nextID,
		DefNumber:       chosenID,
		RoomNumber:      roomNum,
		Alive:           true,
		CurrentHP:       hp,
		DefenseBonus:    monsterPsiDefenseBonus(def.Disciplines),
		TargetMonsterID: -1,
		Mana:            def.Mana,
	}

	idx := len(mm.instances)

	mm.instances = append(mm.instances, inst)
	mm.monstersByRoom[roomNum] = append(
		mm.monstersByRoom[roomNum],
		idx,
	)

	mm.nextID++

	// Successful generation ends the cycle.
	mm.roomSpawnTicks[roomNum] = 0

	log.Printf(
		"[SPAWN] room %d: SPAWNED monster %d (%s), hp=%d",
		roomNum,
		chosenID,
		FormatMonsterName(def, e.monAdjs),
		hp,
	)

	// Don't hold the manager mutex while broadcasting.
	mm.mu.Unlock()

	// ------------------------------------------------------------
	// GENERATION TEXT
	// ------------------------------------------------------------

	if e.localRoomBroadcast == nil {
		return
	}

	genText := def.TextOverrides["TEXG"]

	if genText != "" {
		e.localRoomBroadcast(
			roomNum,
			[]string{genText},
		)
		return
	}

	name := FormatMonsterName(def, e.monAdjs)
	article := articleFor(name, def.Unique)

	e.localRoomBroadcast(
		roomNum,
		[]string{
			fmt.Sprintf(
				"%s%s appears.",
				capArticle(article),
				name,
			),
		},
	)
}

func (e *GameEngine) StartMonsterSpawnLoop() {
	log.Printf("[SPAWN] monster spawn loop STARTED")

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			log.Printf("[SPAWN] spawn loop tick")
			e.monsterSpawnTick()
		}
	}()
}

func (e *GameEngine) monsterSpawnTick() {
	if e.monsterMgr == nil || e.sessions == nil {
		return
	}

	roomSet := make(map[int]bool)

	for _, p := range e.sessions.OnlinePlayers() {
		if p.Dead || p.GMInvis {
			continue
		}

		roomSet[p.RoomNumber] = true

		e.monsterMgr.mu.Lock()
		e.monsterMgr.roomLastPlayer[p.RoomNumber] = time.Now()
		e.monsterMgr.mu.Unlock()
	}

	for roomNum := range roomSet {
		e.generateMonsterForRoom(roomNum)
	}
}

//end new code

// SpawnOne creates a single monster instance in a room. hp should include ExtraBody.
func (mm *monsterManager) SpawnOne(defNum, roomNum, hp int, mana int) *MonsterInstance {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	inst := MonsterInstance{
		ID:              mm.nextID,
		DefNumber:       defNum,
		RoomNumber:      roomNum,
		Alive:           true,
		CurrentHP:       hp,
		TargetMonsterID: -1, // no monster target by default
		Mana:            mana,
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

		// Invisible monsters are not shown in LOOK.
		if inst.HasStatEffect(InvisibilityEffect) {
			continue
		}

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

	// Join with commas and "and":
	// "a priest, a trader and a merchant"
	var joined string

	if len(names) == 1 {
		joined = names[0]
	} else {
		joined = strings.Join(
			names[:len(names)-1],
			", ",
		) + " and " + names[len(names)-1]
	}

	return []string{
		"You also see " + joined + ".",
	}
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
			// commented out while we try the new spawn code
			// Periodic respawn check near players every ~30 seconds
			//if tick%10 == 0 {
			//	e.respawnNearPlayers()
			//}
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

		// Summoned creatures are controlled by their summoner
		// and should not be removed by distant-room cleanup.
		if inst.ControlType == "SUMMONED" {
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
			// While waiting for the monster's next action tick,
			// occasionally emit ambient TEX1-TEX4 flavor text.
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

			continue
		}

		// ------------------------------------------------------------
		// TEMPORARY EFFECTS
		// ------------------------------------------------------------

		expiredEffects := inst.RemoveExpiredStatEffects()

		for _, effect := range expiredEffects {
			name := FormatMonsterName(def, e.monAdjs)

			switch effect.Stat {
			case FearEffect:
				e.localRoomBroadcast(
					inst.RoomNumber,
					[]string{
						fmt.Sprintf(
							"The %s is no longer afraid.",
							name,
						),
					},
				)

			case CharmEffect:
				e.localRoomBroadcast(
					inst.RoomNumber,
					[]string{
						fmt.Sprintf(
							"The %s no longer looks quite so friendly.",
							name,
						),
					},
				)
			}
		}

		// ------------------------------------------------------------
		// CHARM
		//
		// Charmed monsters remain peaceful and take no hostile
		// actions. Occasionally show a little flavor text instead.
		// ------------------------------------------------------------

		if inst.HasStatEffect(CharmEffect) {
			if rand.Intn(100) < 20 {
				name := FormatMonsterName(def, e.monAdjs)

				e.localRoomBroadcast(
					inst.RoomNumber,
					[]string{
						fmt.Sprintf(
							"The %s smiles warmly at you.",
							name,
						),
					},
				)
			}

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

					// Only establish player-side engagement if the player
					// isn't already engaged with another opponent.
					if p.CombatTarget == nil || !p.Joined {
						p.CombatTarget = &CombatTarget{
							IsMonster: true,
							MonsterID: inst.ID,
						}
						p.Joined = true
					}

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

		//def.Strategy < 301 &&  removed this check so that monsters with strategy >= 301 can wander when not in combat

		if inst.Target == "" &&
			inst.TargetMonsterID < 0 {

			wanderRoll := rand.Intn(100)

			// Only log successful wander rolls so we don't flood the console.
			if wanderRoll < 6 {
				log.Printf(
					"[WANDER] %s (id=%d, monster=%d) room=%d passed wander roll=%d",
					FormatMonsterName(def, e.monAdjs),
					inst.ID,
					inst.DefNumber,
					inst.RoomNumber,
					wanderRoll,
				)

				room := e.rooms[inst.RoomNumber]
				if room == nil {
					log.Printf(
						"[WANDER] monster id=%d FAILED: room %d not found",
						inst.ID,
						inst.RoomNumber,
					)
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
					log.Printf(
						"[WANDER] %s (id=%d) FAILED: no usable exits from room %d",
						FormatMonsterName(def, e.monAdjs),
						inst.ID,
						inst.RoomNumber,
					)
					continue
				}

				chosen := exits[rand.Intn(len(exits))]

				oldRoom := inst.RoomNumber

				log.Printf(
					"[WANDER] %s (id=%d) attempting %s: room %d -> %d",
					FormatMonsterName(def, e.monAdjs),
					inst.ID,
					chosen.dir,
					oldRoom,
					chosen.destID,
				)

				if !e.moveMonsterDirection(
					idx,
					inst,
					def,
					chosen.dir,
				) {
					log.Printf(
						"[WANDER] %s (id=%d) MOVE FAILED: %s from room %d -> %d",
						FormatMonsterName(def, e.monAdjs),
						inst.ID,
						chosen.dir,
						oldRoom,
						chosen.destID,
					)
					continue
				}

				log.Printf(
					"[WANDER] %s (id=%d) MOVED: room %d -> %d via %s",
					FormatMonsterName(def, e.monAdjs),
					inst.ID,
					oldRoom,
					chosen.destID,
					chosen.dir,
				)
			}
		}
	}
}

func (e *GameEngine) chooseMonsterSpell(inst *MonsterInstance, def *gameworld.MonsterDef) *SpellDef {

	if inst == nil || def == nil {
		return nil
	}

	var choices []*SpellDef

	for _, spellID := range def.Spells {
		spell := FindSpellByID(spellID)
		if spell == nil {
			continue
		}

		// Monster cannot cast above its configured cast level.
		//if def.CastLevel > 0 && spell.Level > def.CastLevel {
		//	continue
		//	}

		// Must currently have enough mana.
		if inst.Mana < spell.ManaCost {
			continue
		}

		// Monsters can cast damage, debuff, and self-buff spells.
		if spell.Effect != "damage" &&
			spell.Effect != "debuff" &&
			spell.Effect != "buff" {
			continue
		}

		choices = append(choices, spell)
	}

	if len(choices) == 0 {
		return nil
	}

	return choices[rand.Intn(len(choices))]
}

func (e *GameEngine) beginMonsterSpell(inst *MonsterInstance, def *gameworld.MonsterDef, spell *SpellDef) string {

	inst.PreparedSpell = spell.ID
	inst.SpellReadyAt = time.Now().Add(
		time.Duration(spell.CastTime) * time.Second,
	)

	name := FormatMonsterName(def, e.monAdjs)
	article := articleFor(name, def.Unique)
	monsterName := capArticle(article) + name

	// TEXS appears to be the monster's preparation/incantation text.
	if msg := def.TextOverrides["TEXS"]; msg != "" {
		return msg
	}

	return fmt.Sprintf("%s incants a spell.", monsterName)
}

func (e *GameEngine) releaseMonsterSpell(inst *MonsterInstance, def *gameworld.MonsterDef, player *Player, spell *SpellDef) ([]string, []string) {

	if inst == nil || def == nil || player == nil || spell == nil {
		return nil, nil
	}

	name := FormatMonsterName(def, e.monAdjs)
	article := articleFor(name, def.Unique)
	monsterName := capArticle(article) + name

	var playerMsgs []string
	var roomMsgs []string

	// If mana was drained while preparing, the spell cannot be released.
	if inst.Mana < spell.ManaCost {
		msg := fmt.Sprintf(
			"%s's spell fades before it can be released.",
			monsterName,
		)

		roomMsgs = append(roomMsgs, msg)
		return playerMsgs, roomMsgs
	}

	// Mana is spent on release, even if the casting roll fails.
	inst.Mana -= spell.ManaCost

	// ------------------------------------------------------------
	// DETERMINE TARGET / CAST CHANCE
	// ------------------------------------------------------------

	selfCast := spell.Effect == "buff"

	var castChance int

	if selfCast {
		// Buff spells are cast by the monster on itself.
		roomMsgs = append(
			roomMsgs,
			fmt.Sprintf(
				"%s gestures.",
				monsterName,
			),
		)

		// Self-cast buffs are not opposed by the player's
		// innate magic resistance.
		castChance = 50 + def.SpellSkill
	} else {
		// Offensive/targeted spell.
		roomMsgs = append(
			roomMsgs,
			fmt.Sprintf(
				"%s gestures at %s.",
				monsterName,
				player.FirstName,
			),
		)

		// Monster spellcasting is opposed by the target's innate
		// level-based magic resistance.
		//
		// SPELLSKILL is already stored on the 5%-per-level scale.
		//
		// Equal skill/level:
		//   SPELLSKILL 75 vs level 15
		//   50 + 75 - (15 * 5) = 50%
		castChance = 50 + def.SpellSkill - (player.Level * 5)
	}

	if castChance < 5 {
		castChance = 5
	}

	if castChance > 95 {
		castChance = 95
	}

	roll := rand.Intn(100) + 1

	if roll > castChance {
		failMsg := def.TextOverrides["TEXQ"]

		if failMsg == "" {
			failMsg = fmt.Sprintf(
				"%s's spell fizzles.",
				monsterName,
			)
		}

		roomMsgs = append(roomMsgs, failMsg)
		return playerMsgs, roomMsgs
	}

	// ------------------------------------------------------------
	// BUFF SPELL
	// ------------------------------------------------------------

	if spell.Effect == "buff" {
		switch spell.StatusType {

		case InvisibilityEffect:
			// Do not stack Invisibility.

			if inst.HasStatEffect(InvisibilityEffect) {
				//	roomMsgs = append(roomMsgs, "Nothing happens.")
				return playerMsgs, roomMsgs
			}

			inst.ApplyStatEffect(
				spell.ID,
				EffectSourceSpell,
				InvisibilityEffect,
				spell.DefBonus,
				spell.Duration,
				spell.Ticks,
			)

			releaseMsg := def.TextOverrides["TEXL"]
			if releaseMsg == "" {
				releaseMsg = fmt.Sprintf(
					"%s fades from view.",
					monsterName,
				)
			}

			roomMsgs = append(roomMsgs, releaseMsg)

			return playerMsgs, roomMsgs
		}

		// Generic fallback for monster buff spells that do not
		// yet have special behavior.
		releaseMsg := def.TextOverrides["TEXL"]

		if releaseMsg == "" {
			releaseMsg = fmt.Sprintf(
				"%s casts %s on itself.",
				monsterName,
				spell.Name,
			)
		}

		roomMsgs = append(roomMsgs, releaseMsg)

		return playerMsgs, roomMsgs
	}

	// ------------------------------------------------------------
	// DEBUFF SPELL
	// ------------------------------------------------------------

	if spell.Effect == "debuff" {
		switch spell.ID {

		case 127: // Web
			if _, ok := player.HasStatEffect(RestrainedEffect); ok {
				roomMsgs = append(roomMsgs, "Nothing happens.")
				return playerMsgs, roomMsgs
			}

			player.ApplyStatEffect(
				spell.ID,
				EffectRestrained,
				RestrainedEffect,
				spell.DefBonus,
				spell.Duration,
				0,
			)

			releaseMsg := def.TextOverrides["TEXL"]
			if releaseMsg == "" {
				releaseMsg = fmt.Sprintf(
					"%s casts Web.",
					monsterName,
				)
			}

			roomMsgs = append(roomMsgs, releaseMsg)

			playerMsgs = append(
				playerMsgs,
				"You are covered with strands of sticky webbing!",
			)

			return playerMsgs, roomMsgs

		case 500: // Plant Snare
			if _, ok := player.HasStatEffect(RestrainedEffect); ok {
				roomMsgs = append(roomMsgs, "Nothing happens.")
				return playerMsgs, roomMsgs
			}

			player.ApplyStatEffect(
				spell.ID,
				EffectSourceSpell,
				RestrainedEffect,
				spell.DefBonus,
				spell.Duration,
				0,
			)

			releaseMsg := def.TextOverrides["TEXL"]
			if releaseMsg == "" {
				releaseMsg = fmt.Sprintf(
					"%s casts Plant Snare.",
					monsterName,
				)
			}

			roomMsgs = append(roomMsgs, releaseMsg)

			playerMsgs = append(
				playerMsgs,
				"Thick vines and roots coil around you, holding you fast!",
			)

			return playerMsgs, roomMsgs

		case 200: // Fear
			if _, ok := player.HasStatEffect(FearEffect); ok {
				roomMsgs = append(roomMsgs, "Nothing happens.")
				return playerMsgs, roomMsgs
			}

			player.ApplyStatEffect(
				spell.ID,
				EffectSourceSpell,
				FearEffect,
				spell.DefBonus,
				spell.Duration,
				spell.Ticks,
			)

			releaseMsg := def.TextOverrides["TEXL"]
			if releaseMsg == "" {
				releaseMsg = fmt.Sprintf(
					"%s casts Fear.",
					monsterName,
				)
			}

			roomMsgs = append(roomMsgs, releaseMsg)

			playerMsgs = append(
				playerMsgs,
				"You are overcome with fear!",
			)

			return playerMsgs, roomMsgs

		case 201: // Charm
			if _, ok := player.HasStatEffect(CharmEffect); ok {
				roomMsgs = append(roomMsgs, "Nothing happens.")
				return playerMsgs, roomMsgs
			}

			player.ApplyStatEffect(
				spell.ID,
				EffectSourceSpell,
				CharmEffect,
				spell.DefBonus,
				spell.Duration,
				spell.Ticks,
			)

			if spell.StatusMsg != "" {
				roomMsgs = append(
					roomMsgs,
					fmt.Sprintf(
						spell.StatusMsg,
						player.FirstName,
					),
				)
			} else {
				roomMsgs = append(
					roomMsgs,
					fmt.Sprintf(
						"%s gets a friendly look.",
						player.FirstName,
					),
				)
			}

			return playerMsgs, roomMsgs
		}
	}

	// ------------------------------------------------------------
	// DAMAGE SPELL
	// ------------------------------------------------------------

	if spell.Effect != "damage" {
		return playerMsgs, roomMsgs
	}

	var dmg int

	if spell.ID == 220 || spell.ID == 221 {
		// Dancing weapons use the monster's spell skill
		// in place of the player's Empathy bonus.
		dmg = rand.Intn(spell.DmgMax) + 1
		dmg += def.SpellSkill / 10
	} else {
		dmg = spell.DmgMin

		if spell.DmgMax > spell.DmgMin {
			dmg += rand.Intn(
				spell.DmgMax - spell.DmgMin + 1,
			)
		}
	}

	damageType := strings.ToUpper(spell.DmgType)

	// ------------------------------------------------------------
	// ELEMENTAL RESISTANCE
	// ------------------------------------------------------------

	switch damageType {

	case "HEAT", "FIRE", "BURN":
		resistance := player.EffectiveStat(HeatResistance)

		if resistance < 0 {
			resistance = 0
		}

		if resistance > 100 {
			resistance = 100
		}

		dmg = dmg * (100 - resistance) / 100

	case "COLD", "ICE":
		resistance := player.EffectiveStat(ColdResistance)

		if resistance < 0 {
			resistance = 0
		}

		if resistance > 100 {
			resistance = 100
		}

		dmg = dmg * (100 - resistance) / 100
	}

	wasConscious := player.BodyPoints > 0

	player.BodyPoints -= dmg

	// ------------------------------------------------------------
	// DAMAGE BREAKS CHARM
	// ------------------------------------------------------------

	if dmg > 0 {
		if player.RemoveStatEffect(CharmEffect) {
			playerMsgs = append(
				playerMsgs,
				"The pain snaps you out of the charm.",
			)
		}
	}

	// ------------------------------------------------------------
	// SPELL INTERRUPTION
	// ------------------------------------------------------------

	disruptedSpell := e.tryInterruptPlayerSpell(player)

	if disruptedSpell {
		playerMsgs = append(
			playerMsgs,
			"Your spell is disrupted!",
		)
	}

	// The initial knockout cannot take the player below 0.
	// Damage received while already unconscious can.
	if wasConscious && player.BodyPoints < 0 {
		player.BodyPoints = 0
	}

	part := randomBodyPart("HUMAN")
	severity := damageSeverity(dmg)

	damageNoun := "blast"

	switch damageType {

	case "SLASHING":
		damageNoun = "slash"

	case "HEAT", "FIRE", "BURN":
		damageNoun = "burn"

	case "COLD", "ICE":
		damageNoun = "blast"

	case "ELECTRIC", "LIGHTNING":
		damageNoun = "shock"

	case "CRUSHING":
		damageNoun = "strike"
	}

	playerMsgs = append(
		playerMsgs,
		fmt.Sprintf(
			"%s %s to %s. [%d Damage]",
			severity,
			damageNoun,
			part,
			dmg,
		),
	)

	// ------------------------------------------------------------
	// DAMAGE SPELL RELEASE MESSAGE
	// ------------------------------------------------------------

	if spell.StatusMsg != "" {
		roomMsgs = append(
			roomMsgs,
			fmt.Sprintf(
				spell.StatusMsg,
				monsterName,
				player.FirstName,
			),
		)
	} else {
		roomMsgs = append(
			roomMsgs,
			fmt.Sprintf(
				"%s casts %s at %s.",
				monsterName,
				spell.Name,
				player.FirstName,
			),
		)
	}

	// ------------------------------------------------------------
	// PLAYER DEATH
	// ------------------------------------------------------------

	if player.BodyPoints <= 0 {
		if e.isArenaRoom(player.RoomNumber) {
			player.BodyPoints = 1

			playerMsgs = append(
				playerMsgs,
				"The arena's enchantment prevents your death!",
			)
		} else {
			deathMsgs := e.handlePlayerDeath(
				player,
				name,
			)

			playerMsgs = append(
				playerMsgs,
				deathMsgs...,
			)

			roomMsgs = append(
				roomMsgs,
				fmt.Sprintf(
					"%s slays %s!",
					monsterName,
					player.FirstName,
				),
			)
		}
	}

	return playerMsgs, roomMsgs
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

func (m *MonsterInstance) ApplyStatEffect(effectID int, source EffectSource, stat StatID, modifier int, duration time.Duration, ticks int) {

	expiresAt := time.Now().Add(duration)

	for i := range m.ActiveStatEffects {
		effect := &m.ActiveStatEffects[i]

		if effect.EffectID == effectID && effect.Source == source {
			effect.Stat = stat
			effect.Modifier = modifier
			effect.ExpiresAt = expiresAt
			effect.Ticks = ticks
			return
		}
	}

	m.ActiveStatEffects = append(m.ActiveStatEffects, StatEffect{
		EffectID:  effectID,
		Source:    source,
		Stat:      stat,
		Modifier:  modifier,
		ExpiresAt: expiresAt,
		Ticks:     ticks,
	})
}

func (m *MonsterInstance) RemoveStatEffectsBySource(source EffectSource) bool {
	removed := false
	active := m.ActiveStatEffects[:0]

	for _, effect := range m.ActiveStatEffects {
		if effect.Source == source {
			removed = true
			continue
		}

		active = append(active, effect)
	}

	m.ActiveStatEffects = active
	return removed
}

func (m *MonsterInstance) RemoveStatEffect(stat StatID) bool {
	removed := false
	active := m.ActiveStatEffects[:0]

	for _, effect := range m.ActiveStatEffects {
		if effect.Stat == stat {
			removed = true
			continue
		}

		active = append(active, effect)
	}

	m.ActiveStatEffects = active
	return removed
}

func (m *MonsterInstance) RemoveActiveStatEffect(effectID int, source EffectSource) bool {
	for i, effect := range m.ActiveStatEffects {
		if effect.EffectID == effectID && effect.Source == source {
			m.ActiveStatEffects = append(
				m.ActiveStatEffects[:i],
				m.ActiveStatEffects[i+1:]...,
			)
			return true
		}
	}

	return false
}

func (m *MonsterInstance) HasStatEffect(stat StatID) bool {
	now := time.Now()

	for _, effect := range m.ActiveStatEffects {
		if effect.Stat != stat {
			continue
		}

		if effect.Permanent || effect.ExpiresAt.After(now) {
			return true
		}
	}

	return false
}

func (m *MonsterInstance) RemoveExpiredStatEffects() []StatEffect {
	now := time.Now()

	if len(m.ActiveStatEffects) == 0 {
		return nil
	}

	var expired []StatEffect
	active := m.ActiveStatEffects[:0]

	for _, effect := range m.ActiveStatEffects {
		if effect.Permanent || effect.ExpiresAt.After(now) {
			active = append(active, effect)
			continue
		}

		// Periodic effect still has ticks remaining.
		if effect.Ticks > 1 {
			effect.Ticks--
			effect.ExpiresAt = now.Add(5 * time.Second)

			active = append(active, effect)
			continue
		}

		// Final tick has expired.
		expired = append(expired, effect)
	}

	m.ActiveStatEffects = active

	return expired
}
