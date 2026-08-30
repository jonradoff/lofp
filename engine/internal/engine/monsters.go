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
	// Stunned/KnockedDown: LEGENDS.DOC says an excellent hit (roll > 95) "may result
	// in stunning you or knocking you off your feet. In the case of a stun ... you are
	// unable to attack or move again until the stun wears off. If you are knocked down
	// ... until you stand up again" — so these are two distinct outcomes of the same
	// crit roll, not the same thing. Stunned lasts for StunExpiry (a duration; no exact
	// seconds is documented for a melee-crit stun specifically, only for psi spells, so
	// 3-6 seconds is a judgment call — see the roll site in doAttackMonster) and gives
	// a bonus to hit it (see the +20 attack rating above). KnockedDown has no duration;
	// it costs exactly one combat tick to stand back up, mirroring SleepStand below.
	Stunned      bool      `json:"-"`
	StunExpiry   time.Time `json:"-"`
	KnockedDown  bool      `json:"-"` // knocked off its feet; must stand up (one skipped tick) before acting
	Skinned      bool      `json:"-"` // already skinned
	DefenseBonus int       `json:"-"` // from active psi defenses
	CurrentHP  int       `json:"currentHP"`
	MaxHP      int       `json:"maxHP"`
	Wounds     []Wound   `json:"-"` // in-memory only, not persisted
	Target     string    `json:"-"`
	Searched   bool      `json:"-"` // already searched for loot
	DeathTime  time.Time `json:"-"` // when it died (for corpse decay)
	LastAttacker string  `json:"-"` // name of the last player to damage this monster (set in damageMonster), for death XP attribution when it dies with no single decisive blow (bleed-out, etc.)

	// Spell status effects
	Sleeping    bool      `json:"-"` // Slumber spell: no attack, no flee
	SleepExpiry time.Time `json:"-"`
	SleepStand  bool      `json:"-"` // woke up but still rising; skip one attack tick
	Webbed      bool      `json:"-"` // Web spell: no attack, no flee
	WebExpiry   time.Time `json:"-"`
	Feared      bool      `json:"-"` // Fear spell: only flees, never attacks
	FearExpiry  time.Time `json:"-"`
	Charmed        bool      `json:"-"` // Charm spell: won't attack the caster
	CharmExpiry    time.Time `json:"-"`
	CharmTarget    string    `json:"-"` // player name who charmed it
	Silenced       bool      `json:"-"` // Silence spell: cannot cast unless def.SilenceIgnore
	SilenceExpiry  time.Time `json:"-"`
	Imprisoned     bool      `json:"-"` // Imprison spell: cannot attack or cast at all
	ImprisonExpiry time.Time `json:"-"`
	Tentacled          bool      `json:"-"` // Siryx's Terrible Tentacles: immobilized, takes periodic crushing damage
	TentacleExpiry     time.Time `json:"-"`
	TentacleCasterName string    `json:"-"` // player who cast Tentacles, for death XP attribution on the DOT tick

	// Monster spellcasting (SPELLUSE/SPELL/MANA) and special attacks (SPECUSE/SPECUSES).
	// CurrentMana is seeded from def.Mana at spawn and never regenerates (per GMSCRIPT.DOC,
	// unlike players monsters have no mana recovery) — once spent it stays spent for the
	// rest of the monster's life. SpecAttacksUsed counts against def.SpecUses.
	CurrentMana     int       `json:"-"`
	SpecAttacksUsed int       `json:"-"`
	// Casting tracks an in-progress spell: GMSCRIPT.DOC's CASTLEVEL is the cast's duration
	// in seconds (TEXS shown when it starts, TEXL + resolution when CastExpiry passes) and
	// can be interrupted by taking damage before then unless the monster is NONDISRUPTABLE
	// (see damageMonster) — matching "A muldragun's spell is disrupted!" in original/log.txt.
	Casting     bool      `json:"-"`
	CastSpellID int       `json:"-"`
	CastTarget  string    `json:"-"` // player FirstName chosen as the spell's target when casting began
	CastExpiry  time.Time `json:"-"`

	// Monster psionics (PSIUSE/DISCIPLINE/PSI, per GMSCRIPT.DOC): mirrors the
	// spellcasting fields above, but only "damage"-effect disciplines from
	// def.Disciplines are used offensively — defensive/buff disciplines are treated
	// as always-active (see monsterPsiDefenseBonus) per GMSCRIPT.DOC ("if you give
	// them a defense discipline ... they get genned with those disciplines already
	// active"). CurrentPsi is seeded from def.Psi at spawn and never regenerates,
	// same as CurrentMana.
	CurrentPsi    int       `json:"-"`
	PsiCasting    bool      `json:"-"`
	PsiCastDiscID int       `json:"-"`
	PsiCastTarget string    `json:"-"` // player FirstName chosen as the discipline's target when casting began
	PsiCastExpiry time.Time `json:"-"`

	// Summoned creature fields (transient)
	SummonerName    string `json:"-"` // name of the player who summoned this creature
	IsSummoned      bool   `json:"-"` // true if summoned, not a regular world spawn
	IsFamiliar      bool   `json:"-"` // true if a familiar (cat/raven/bird/viper) — can WATCH WILL
	WatchMode       bool   `json:"-"` // familiar is watching a room and forwarding events to summoner
	FollowTarget    string   `json:"-"` // name of player this creature follows when they move; "" = not following
	GuardingPlayers []string `json:"-"` // names of players being guarded (for COMMAND GUARD)
	MonsterTargetID int      `json:"-"` // instance ID of a monster this creature is attacking; 0 = none
	ControlExpiry   time.Time `json:"-"` // Control Undead: when control lapses and the creature turns hostile again (zero = never expires)

	// Timed buff spells (Strength, Agility, Haste, Mystic Armor) cast on a summoned or
	// controlled creature by any caster, not just its summoner — see findSummonedCreatureInRoom
	// and castStrengthSpell/castAgilitySpell/castHasteSpell/castMysticArmor. Kept as
	// instance-level bonuses rather than baked into a base stat (unlike the player
	// equivalents in player.go) because MonsterDef's Attack1/Defense/Speed are shared,
	// immutable base stats — see monsterEffectiveAttack/monsterEffectiveDefense/
	// monsterEffectiveSpeed for where they're read. Expiry is checked live at each usage
	// site and cleared with a fade message in monsterTick, same pattern as the
	// Sleeping/Webbed/Feared status effects above.
	StrengthBuffID     int       `json:"-"` // spell ID of the active Strength tier (207/208/209), 0 = none
	StrengthBuffBonus  int       `json:"-"`
	StrengthBuffExpiry time.Time `json:"-"`
	AgilityBuffID      int       `json:"-"` // spell ID of the active Agility tier (513/514/515), 0 = none
	AgilityBuffBonus   int       `json:"-"`
	AgilityBuffExpiry  time.Time `json:"-"`
	HasteExpiry        time.Time `json:"-"`
	MysticArmorBonus   int       `json:"-"`
	MysticArmorExpiry  time.Time `json:"-"`
	// TimedDefenseBuffs holds every other flat-bonus defense spell (Globe of Protection
	// I/II, Mass Protection, Spectral Shield, Ride the Lightning — see
	// castTimedDefenseSpell) currently active, reusing the same TimedDefenseBuff struct
	// as the player equivalent (player.go). Kept as a list rather than a single field
	// since, like players, a creature can have several distinct named buffs stacked at
	// once — see monsterEffectiveDefense for where they're summed.
	TimedDefenseBuffs []TimedDefenseBuff `json:"-"`

	// Appearance is lazily rolled the first time a disguisable town NPC (commoner,
	// trader, merchant, lawkeeper, beggar — see disguisableNPCNames) is examined, then
	// cached here so repeated looks show the same person rather than rerolling. Not
	// persisted; regenerating on respawn is fine.
	AppearanceRolled    bool   `json:"-"`
	AppearanceRace      int    `json:"-"`
	AppearanceGender    int    `json:"-"`
	AppearanceAge       int    `json:"-"`
	AppearanceHeight    int    `json:"-"`
	AppearanceWeight    int    `json:"-"`
	AppearanceStrength  int    `json:"-"`
	AppearanceEyeColor  string `json:"-"`
	AppearanceSkinColor string `json:"-"`
	AppearanceHairColor string `json:"-"`
	AppearanceHairStyle string `json:"-"`
}

// disguisableNPCNames are the generic town NPC types a disguise can blend into —
// they get a full rolled appearance description on LOOK instead of a flat
// "You see a trader." Keyed by def.Name lowercased (MNUMBER 12/13/28/30/31 in
// MONSTERS.SCR: commoner, lawkeeper, beggar, trader, merchant).
var disguisableNPCNames = map[string]bool{
	"commoner":  true,
	"trader":    true,
	"merchant":  true,
	"lawkeeper": true,
	"beggar":    true,
	"priest":    true,
}

// rollMonsterAppearance randomly generates and caches an appearance for inst.
// Race is chosen from the full playable race list (not just RaceHuman) so
// repeated NPCs of the same type look like different people, matching town
// NPCs being a mix of the setting's races.
func rollMonsterAppearance(inst *MonsterInstance) {
	race := PlayableRaces[rand.Intn(len(PlayableRaces))]
	gender := rand.Intn(2)
	height, weight := RollHeightWeight(race, gender)
	str, _, _, _, _, _, _ := RollStats(race)
	inst.AppearanceRace = race
	inst.AppearanceGender = gender
	inst.AppearanceAge = RollAge(race)
	inst.AppearanceHeight = height
	inst.AppearanceWeight = weight
	inst.AppearanceStrength = str
	inst.AppearanceEyeColor = RandomEyeColor()
	inst.AppearanceSkinColor = RandomSkinColor()
	inst.AppearanceHairColor = RandomHairColor()
	inst.AppearanceHairStyle = RandomHairStyle()
	inst.AppearanceRolled = true
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

// monsterEffectiveAttack returns a monster's Attack1 rating plus any active Strength
// buff bonus. The bonus is divided by 5 to match the player equivalent's scaling
// (playerAttackRating adds player.Strength/5), so a Strength I/II/III cast on a
// summoned/controlled creature is worth the same +2/+4/+6 to-hit and damage swing a
// player gets from it, not the raw 10/20/30.
func monsterEffectiveAttack(def *gameworld.MonsterDef, inst *MonsterInstance) int {
	atk := def.Attack1
	if inst.StrengthBuffBonus > 0 && !inst.StrengthBuffExpiry.IsZero() && time.Now().Before(inst.StrengthBuffExpiry) {
		atk += inst.StrengthBuffBonus / 5
	}
	return atk
}

// monsterEffectiveDefense returns a monster's Defense rating plus its passive psi
// DefenseBonus, any active Agility buff (same /5 scaling as monsterEffectiveAttack,
// mirroring playerDefenseRating's player.Agility/5), any active Mystic Armor buff (added
// at full value, same as applyMysticArmorBuff does for players), and every active flat
// defense spell in TimedDefenseBuffs (Globe of Protection I/II, Mass Protection, Spectral
// Shield, Ride the Lightning — see castTimedDefenseSpell), which stack additively same as
// they do on a player.
func monsterEffectiveDefense(def *gameworld.MonsterDef, inst *MonsterInstance) int {
	d := def.Defense + inst.DefenseBonus
	now := time.Now()
	if inst.AgilityBuffBonus > 0 && !inst.AgilityBuffExpiry.IsZero() && now.Before(inst.AgilityBuffExpiry) {
		d += inst.AgilityBuffBonus / 5
	}
	if inst.MysticArmorBonus > 0 && !inst.MysticArmorExpiry.IsZero() && now.Before(inst.MysticArmorExpiry) {
		d += inst.MysticArmorBonus
	}
	for _, b := range inst.TimedDefenseBuffs {
		if now.Before(b.Expiry) {
			d += b.Bonus
		}
	}
	return d
}

// monsterEffectiveSpeed halves a monster's action-tick interval while Haste is active
// (minimum 1), the same way applyRoundTime halves a hasted player's round time.
func monsterEffectiveSpeed(def *gameworld.MonsterDef, inst *MonsterInstance) int {
	speed := def.Speed
	if speed <= 0 {
		speed = 3
	}
	if !inst.HasteExpiry.IsZero() && time.Now().Before(inst.HasteExpiry) {
		speed /= 2
		if speed < 1 {
			speed = 1
		}
	}
	return speed
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
				MaxHP:        hp,
				DefenseBonus: monsterPsiDefenseBonus(def.Disciplines),
				CurrentMana:  def.Mana,
				CurrentPsi:   def.Psi,
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
// mana seeds CurrentMana (pass the def's Mana field; 0 if the monster has no spells).
// psi seeds CurrentPsi likewise (pass the def's Psi field; 0 if the monster has no disciplines).
func (mm *monsterManager) SpawnOne(defNum, roomNum, hp, mana, psi int) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	inst := MonsterInstance{ID: mm.nextID, DefNumber: defNum, RoomNumber: roomNum, Alive: true, CurrentHP: hp, MaxHP: hp, CurrentMana: mana, CurrentPsi: psi}
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

// GetOrRollAppearance returns the appearance fields for a monster instance by
// ID, rolling and caching them on first call (under the same lock) so repeated
// looks at the same instance show a consistent person. ok is false if the
// instance no longer exists (e.g. it died and was cleaned up between calls).
func (mm *monsterManager) GetOrRollAppearance(id int) (inst MonsterInstance, ok bool) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	for i := range mm.instances {
		if mm.instances[i].ID == id {
			if !mm.instances[i].AppearanceRolled {
				rollMonsterAppearance(&mm.instances[i])
			}
			return mm.instances[i], true
		}
	}
	return MonsterInstance{}, false
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

// monsterNamesInRoom returns the display names (with article and status suffix)
// of every monster in a room, e.g. ["a priest", "a trader (dead)"]. Shared by
// MonsterLookLines (which wraps these into their own "You also see" sentence)
// and doLook (which merges them into the same sentence as players present).
func (e *GameEngine) monsterNamesInRoom(roomNum int) []string {
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
		} else if inst.Imprisoned {
			name += " (imprisoned)"
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
	return names
}

// MonsterLookLines returns the lines to append to a room look showing monsters.
// Condenses multiple monsters into a single "You also see" line.
func (e *GameEngine) MonsterLookLines(roomNum int) []string {
	names := e.monsterNamesInRoom(roomNum)
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
				e.bleedDamageTick()
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
			dmgLine := fmt.Sprintf(" %s %s to %s. [%d Damage]", damageSeverity(dmg, inst.MaxHP), spellDmgNoun(spell.DmgType), randomBodyPart(def.BodyType), dmg)
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
		if recipients := e.dotKillRecipients(k.inst.TentacleCasterName); len(recipients) > 0 {
			e.handleMonsterDeath(recipients, &k.inst, k.def)
		}
	}
}

// bleedDamageTick applies once-a-minute bleed-out damage to every monster
// instance with an active bleeding wound. Runs on the same ~60-second
// cadence as tentacleDamageTick (StartMonsterLoop, tick%20==0). Bleed-out
// kills have no single decisive blow (the wounds may have come from multiple
// attackers over time), so kill XP is attributed to whoever landed the last
// hit before it bled out — see MonsterInstance.LastAttacker (set in
// damageMonster) and dotKillRecipients.
func (e *GameEngine) bleedDamageTick() {
	if e.monsterMgr == nil {
		return
	}

	type bleedKill struct {
		inst MonsterInstance
		def  *gameworld.MonsterDef
	}
	var kills []bleedKill

	e.monsterMgr.mu.Lock()
	for i := range e.monsterMgr.instances {
		inst := &e.monsterMgr.instances[i]
		if !inst.Alive || !anyBleeding(inst.Wounds) {
			continue
		}
		def := e.monsters[inst.DefNumber]
		if def == nil {
			continue
		}

		total := 0
		for _, w := range inst.Wounds {
			if w.Bleeding {
				total += woundBleedDrainPerMinute(w.Level)
			}
		}
		if total <= 0 {
			continue
		}

		if e.localRoomBroadcast != nil {
			name := strings.ToLower(FormatMonsterName(def, e.monAdjs))
			article := articleFor(name, def.Unique)
			e.localRoomBroadcast(inst.RoomNumber, []string{fmt.Sprintf("%s%s's wounds bleed! [-%d Damage]", capArticle(article), name, total)})
		}

		inst.CurrentHP -= total
		if inst.CurrentHP <= 0 {
			inst.Alive = false
			inst.CurrentHP = 0
			inst.DeathTime = time.Now()
			kills = append(kills, bleedKill{inst: *inst, def: def})
		}
	}
	e.monsterMgr.mu.Unlock()

	for _, k := range kills {
		// Everyone who was fighting it needs to be taken out of combat regardless of
		// whether anyone ends up credited with the kill below — see clearCombatForMonster.
		e.clearCombatForMonster(k.inst.ID)
		if recipients := e.dotKillRecipients(k.inst.LastAttacker); len(recipients) > 0 {
			e.handleMonsterDeath(recipients, &k.inst, k.def)
		}
		if e.localRoomBroadcast == nil {
			continue
		}
		name := strings.ToLower(FormatMonsterName(k.def, e.monAdjs))
		deathText := k.def.TextOverrides["TEXD"]
		if deathText != "" {
			e.localRoomBroadcast(k.inst.RoomNumber, []string{fmt.Sprintf("A %s %s", name, deathText)})
		} else {
			e.localRoomBroadcast(k.inst.RoomNumber, []string{fmt.Sprintf("A %s collapses, dead!", name)})
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

		// Timed buff expiry (Strength, Agility, Haste, Mystic Armor) — see
		// monsterEffectiveAttack/monsterEffectiveDefense/monsterEffectiveSpeed for where
		// these are read, and castStrengthSpell/castAgilitySpell/castHasteSpell/
		// castMysticArmor for where they're applied. Checked every tick, same as the
		// Control Undead expiry above, not gated by the speed check below.
		now := time.Now()
		if inst.StrengthBuffBonus > 0 && !inst.StrengthBuffExpiry.IsZero() && now.After(inst.StrengthBuffExpiry) {
			inst.StrengthBuffID = 0
			inst.StrengthBuffBonus = 0
			inst.StrengthBuffExpiry = time.Time{}
			if e.localRoomBroadcast != nil {
				name := strings.ToLower(FormatMonsterName(def, e.monAdjs))
				e.localRoomBroadcast(inst.RoomNumber, []string{fmt.Sprintf("The magical strength fades from the %s.", name)})
			}
		}
		if inst.AgilityBuffBonus > 0 && !inst.AgilityBuffExpiry.IsZero() && now.After(inst.AgilityBuffExpiry) {
			inst.AgilityBuffID = 0
			inst.AgilityBuffBonus = 0
			inst.AgilityBuffExpiry = time.Time{}
			if e.localRoomBroadcast != nil {
				name := strings.ToLower(FormatMonsterName(def, e.monAdjs))
				e.localRoomBroadcast(inst.RoomNumber, []string{fmt.Sprintf("The magical agility fades from the %s.", name)})
			}
		}
		if !inst.HasteExpiry.IsZero() && now.After(inst.HasteExpiry) {
			inst.HasteExpiry = time.Time{}
			if e.localRoomBroadcast != nil {
				name := strings.ToLower(FormatMonsterName(def, e.monAdjs))
				e.localRoomBroadcast(inst.RoomNumber, []string{fmt.Sprintf("The magical haste fades from the %s.", name)})
			}
		}
		if inst.MysticArmorBonus > 0 && !inst.MysticArmorExpiry.IsZero() && now.After(inst.MysticArmorExpiry) {
			inst.MysticArmorBonus = 0
			inst.MysticArmorExpiry = time.Time{}
			if e.localRoomBroadcast != nil {
				name := strings.ToLower(FormatMonsterName(def, e.monAdjs))
				e.localRoomBroadcast(inst.RoomNumber, []string{fmt.Sprintf("The Mystic Armor fades from the %s.", name)})
			}
		}
		if len(inst.TimedDefenseBuffs) > 0 {
			var active []TimedDefenseBuff
			for _, b := range inst.TimedDefenseBuffs {
				if now.After(b.Expiry) {
					if e.localRoomBroadcast != nil {
						name := strings.ToLower(FormatMonsterName(def, e.monAdjs))
						e.localRoomBroadcast(inst.RoomNumber, []string{fmt.Sprintf("The %s fades from the %s.", b.SpellName, name)})
					}
				} else {
					active = append(active, b)
				}
			}
			inst.TimedDefenseBuffs = active
		}

		// Speed determines action frequency: speed 1 = every tick, speed 3 (default) = every 3 ticks
		speed := monsterEffectiveSpeed(def, inst)
		if tick%speed != 0 {
			continue
		}

		if inst.Target != "" || inst.MonsterTargetID > 0 {
			e.monsterCombatTick(inst, def)
			continue
		}

		name := FormatMonsterName(def, e.monAdjs)

		// Hostile monsters without a target — look for players in room (skip summoned
		// creatures and GUARDIAN-flagged monsters, which never aggro players on sight —
		// see the Guardian block below instead).
		if def.Strategy >= 301 && !def.Guardian && inst.Target == "" && !inst.IsSummoned {
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

		// Guardian monsters (e.g. the GUARDIAN-flagged large wolf) never aggro players on
		// sight — they only fight back if attacked (see the unconditional Target
		// assignment in doAttackMonster) — but they do aggro any other monster in the
		// room that's currently attacking a player, defending players from hostiles
		// rather than hunting players themselves.
		if def.Guardian && inst.Target == "" && inst.MonsterTargetID == 0 {
			for _, idx2 := range e.monsterMgr.monstersByRoom[inst.RoomNumber] {
				if idx2 == idx || idx2 >= len(e.monsterMgr.instances) {
					continue
				}
				other := &e.monsterMgr.instances[idx2]
				if !other.Alive || other.Target == "" {
					continue
				}
				otherDef := e.monsters[other.DefNumber]
				if otherDef == nil || otherDef.Guardian {
					continue
				}
				inst.MonsterTargetID = other.ID
				if e.localRoomBroadcast != nil {
					e.localRoomBroadcast(inst.RoomNumber, []string{fmt.Sprintf("%s snarls and lunges at %s!", capArticle(articleFor(name, def.Unique))+name, FormatMonsterName(otherDef, e.monAdjs))})
				}
				break
			}
			if inst.MonsterTargetID > 0 {
				continue // start fighting next tick
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
