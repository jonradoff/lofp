package engine

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Race constants
const (
	RaceHuman      = 1
	RaceAelfen     = 2
	RaceHighlander = 3
	RaceWolfling   = 4
	RaceMurg       = 5
	RaceDrakin     = 6
	RaceMechanoid  = 7
	RaceEphemeral  = 8
)

var RaceNames = map[int]string{
	RaceHuman: "Human", RaceAelfen: "Aelfen", RaceHighlander: "Highlander",
	RaceWolfling: "Wolfling", RaceMurg: "Murg", RaceDrakin: "Drakin",
	RaceMechanoid: "Mechanoid", RaceEphemeral: "Ephemeral",
}

// Stat ranges per race: [min, max] for each of 7 stats
var RaceStatRanges = map[int][7][2]int{
	RaceHuman:      {{30, 100}, {30, 100}, {30, 100}, {30, 100}, {30, 100}, {40, 110}, {30, 100}},
	RaceAelfen:     {{20, 90}, {40, 110}, {40, 110}, {1, 70}, {40, 110}, {30, 100}, {40, 110}},
	RaceDrakin:     {{40, 110}, {10, 80}, {40, 110}, {40, 110}, {30, 100}, {30, 100}, {40, 110}},
	RaceEphemeral:  {{1, 10}, {30, 100}, {50, 120}, {1, 10}, {30, 100}, {30, 100}, {30, 100}},
	RaceHighlander: {{40, 110}, {20, 90}, {20, 90}, {50, 120}, {30, 100}, {30, 100}, {10, 80}},
	RaceMechanoid:  {{40, 110}, {30, 100}, {30, 100}, {40, 110}, {40, 110}, {30, 100}, {1, 60}},
	RaceMurg:       {{40, 110}, {30, 100}, {30, 100}, {40, 110}, {40, 110}, {20, 90}, {20, 90}},
	RaceWolfling:   {{30, 100}, {40, 110}, {40, 110}, {30, 100}, {40, 110}, {30, 100}, {30, 100}},
}

// Gender constants
const (
	GenderMale   = 0
	GenderFemale = 1
)

// Container type
type InventoryItem struct {
    Archetype int    `bson:"archetype" json:"archetype"`
    Adj1      int    `bson:"adj1,omitempty" json:"adj1,omitempty"`
    Adj2      int    `bson:"adj2,omitempty" json:"adj2,omitempty"`
    Adj3      int    `bson:"adj3,omitempty" json:"adj3,omitempty"`
    Val1      int    `bson:"val1,omitempty" json:"val1,omitempty"`
    Val2      int    `bson:"val2,omitempty" json:"val2,omitempty"`
    Val3      int    `bson:"val3,omitempty" json:"val3,omitempty"`
    Val4      int    `bson:"val4,omitempty" json:"val4,omitempty"`
    Val5      int    `bson:"val5,omitempty" json:"val5,omitempty"`
    // Sharpness is the non-magical to-hit bonus forged into a weapon (weaponSharpnessBonus).
    // Kept separate from Val1, which per GMSCRIPT.DOC is the item's copper sell value.
    Sharpness int    `bson:"sharpness,omitempty" json:"sharpness,omitempty"`
    State     string `bson:"state,omitempty" json:"state,omitempty"`
    Tail      string `bson:"tail,omitempty" json:"tail,omitempty"`
    WornSlot  string `bson:"wornSlot,omitempty" json:"wornSlot,omitempty"`
    // Container contents — populated when this item is an open container.
    Contents  []InventoryItem `bson:"contents,omitempty" json:"contents,omitempty"`
}

// TimedDefenseBuff tracks one active defense spell with a bonus and expiry.
type TimedDefenseBuff struct {
	SpellID   int       `bson:"spellID" json:"spellID"`
	SpellName string    `bson:"spellName" json:"spellName"`
	Bonus     int       `bson:"bonus" json:"bonus"`
	Expiry    time.Time `bson:"expiry" json:"expiry"`
}

// Player represents a player's current state.
type Player struct {
	ID         bson.ObjectID     `bson:"_id,omitempty" json:"id"`
	AccountID  string            `bson:"accountId,omitempty" json:"accountId,omitempty"`
	FirstName  string            `bson:"firstName" json:"firstName"`
	LastName   string            `bson:"lastName" json:"lastName"`
	Race       int               `bson:"race" json:"race"`
	Gender     int               `bson:"gender" json:"gender"`
	Level      int               `bson:"level" json:"level"`
	Experience int               `bson:"experience" json:"experience"`

	// Stats
	Strength     int `bson:"strength" json:"strength"`
	Agility      int `bson:"agility" json:"agility"`
	Quickness    int `bson:"quickness" json:"quickness"`
	Constitution int `bson:"constitution" json:"constitution"`
	Perception   int `bson:"perception" json:"perception"`
	Willpower    int `bson:"willpower" json:"willpower"`
	Empathy      int `bson:"empathy" json:"empathy"`

	// Derived
	BodyPoints    int `bson:"bodyPoints" json:"bodyPoints"`
	MaxBodyPoints int `bson:"maxBodyPoints" json:"maxBodyPoints"`
	Fatigue       int `bson:"fatigue" json:"fatigue"`
	MaxFatigue    int `bson:"maxFatigue" json:"maxFatigue"`
	Mana          int `bson:"mana" json:"mana"`
	MaxMana       int `bson:"maxMana" json:"maxMana"`
	Psi           int `bson:"psi" json:"psi"`
	MaxPsi        int `bson:"maxPsi" json:"maxPsi"`

	// Position
	RoomNumber int  `bson:"roomNumber" json:"roomNumber"`
	Position   int  `bson:"position" json:"position"` // 0=standing, 1=sitting, 2=laying, 3=kneeling, 4=flying
	Hidden     bool `bson:"hidden" json:"hidden"`         // stealth: revealed by movement, emotes, attacks
	Invisible  bool `bson:"invisible" json:"invisible"`   // spell effect: not revealed by movement, only by attacks or dispel
	Dead       bool `bson:"dead" json:"dead"`

	// Physical attributes
	Height     int `bson:"height,omitempty" json:"height,omitempty"`       // inches
	HeightTrue int `bson:"heightTrue,omitempty" json:"heightTrue,omitempty"`
	Weight     int `bson:"weight,omitempty" json:"weight,omitempty"`       // pounds (base, not inventory)
	WeightTrue int `bson:"weightTrue,omitempty" json:"weightTrue,omitempty"`
	Age        int `bson:"age,omitempty" json:"age,omitempty"`
	AgeTrue    int `bson:"ageTrue,omitempty" json:"ageTrue,omitempty"`

	// Status conditions
	Bleeding     bool `bson:"bleeding" json:"bleeding"`
	Stunned      bool `bson:"stunned" json:"stunned"`
	Diseased     bool `bson:"diseased" json:"diseased"`
	DiseaseLevel int  `bson:"diseaseLevel,omitempty" json:"diseaseLevel,omitempty"`
	Poisoned     bool `bson:"poisoned" json:"poisoned"`
	PoisonLevel  int  `bson:"poisonLevel,omitempty" json:"poisonLevel,omitempty"`
	Joined      bool `bson:"joined" json:"joined"`
	Unconscious bool `bson:"unconscious" json:"unconscious"`
	Immobilized bool `bson:"immobilized" json:"immobilized"`
	Sleeping    bool `bson:"sleeping,omitempty" json:"sleeping,omitempty"`
	Submitting  bool `bson:"submitting,omitempty" json:"submitting,omitempty"`
	Undead      bool `bson:"undead,omitempty" json:"undead,omitempty"`
	WolfForm    bool `bson:"wolfForm,omitempty" json:"wolfForm,omitempty"`
	SlimeForm   bool `bson:"slimeForm,omitempty" json:"slimeForm,omitempty"`
	Disguised   bool `bson:"disguised,omitempty" json:"disguised,omitempty"`
	RoundTime       int       `bson:"roundTime" json:"roundTime"`
	RoundTimeExpiry time.Time `bson:"-" json:"-"` // transient: when roundtime ends
	CanFly          bool      `bson:"canFly" json:"canFly"`
	EtherealActive  bool      `bson:"etherealActive" json:"etherealActive"` // PSI Ethereal Projection active
	PreparedSpell   int       `bson:"preparedSpell,omitempty" json:"preparedSpell,omitempty"`

	// Combat
	Stance        int           `bson:"-" json:"-"`          // StanceNormal..StanceBerserk
	CombatTarget  *CombatTarget `bson:"-" json:"-"`          // current combat target
	Targets       []int         `bson:"-" json:"-"`          // TARGET command: up to 6 monster instance IDs for multi-target spells
	DefenseBonus  int           `bson:"-" json:"-"`          // from spells/psi
	PreparedPsi   int           `bson:"-" json:"-"`          // prepared psi discipline ID
	ActivePsi     map[int]bool `bson:"-" json:"-"`          // currently maintained psi disciplines
	BackstabNext  bool          `bson:"-" json:"-"`          // next attack is a backstab
	TelepathyActive bool      `bson:"telepathyActive,omitempty" json:"telepathyActive,omitempty"`
	TelepathyExpiry time.Time `bson:"telepathyExpiry,omitempty" json:"telepathyExpiry,omitempty"`
	Emotional       bool      `bson:"emotional,omitempty" json:"emotional,omitempty"`

	// Temporary strength buff (Strength I/II/III spells)
	StrengthBuffID     int       `bson:"strengthBuffId,omitempty" json:"strengthBuffId,omitempty"`
	StrengthBuffExpiry time.Time `bson:"strengthBuffExpiry,omitempty" json:"strengthBuffExpiry,omitempty"`
	StrengthBuffBonus  int       `bson:"strengthBuffBonus,omitempty" json:"strengthBuffBonus,omitempty"`

	// Temporary agility buff (Agility I/II/III spells)
	AgilityBuffID     int       `bson:"agilityBuffId,omitempty" json:"agilityBuffId,omitempty"`
	AgilityBuffExpiry time.Time `bson:"agilityBuffExpiry,omitempty" json:"agilityBuffExpiry,omitempty"`
	AgilityBuffBonus  int       `bson:"agilityBuffBonus,omitempty" json:"agilityBuffBonus,omitempty"`

	// Temporary Mystic Armor buff (spell 102)
	MysticArmorExpiry time.Time `bson:"mysticArmorExpiry,omitempty" json:"mysticArmorExpiry,omitempty"`
	MysticArmorBonus  int       `bson:"mysticArmorBonus,omitempty" json:"mysticArmorBonus,omitempty"`

	// Timed defense buffs for all defense spells other than Mystic Armor.
	// Each entry tracks one spell's bonus and expiry independently.
	TimedDefenseBuffs []TimedDefenseBuff `bson:"timedDefenseBuffs,omitempty" json:"timedDefenseBuffs,omitempty"`

	// Haste / Slow spell timers (spell 210 / 211)
	HasteExpiry time.Time `bson:"hasteExpiry,omitempty" json:"hasteExpiry,omitempty"`
	SlowExpiry  time.Time `bson:"slowExpiry,omitempty" json:"slowExpiry,omitempty"`

	// Fly spell timer (spell 224) — CanFly from this spell only lasts until FlyExpiry.
	// Zero value means flight (if any) came from a race ability or maintained psi power instead.
	FlyExpiry time.Time `bson:"flyExpiry,omitempty" json:"flyExpiry,omitempty"`

	// Elemental resistance shields (Heat Shield / Cold Shield, spells 507/508):
	// while active, damage of that element taken is reduced by 50%.
	HeatShieldExpiry time.Time `bson:"heatShieldExpiry,omitempty" json:"heatShieldExpiry,omitempty"`
	ColdShieldExpiry time.Time `bson:"coldShieldExpiry,omitempty" json:"coldShieldExpiry,omitempty"`

	// Spell preparation reagent (transient — which item arch was verified at PREPARE time)
	PreparedSpellReagentArch int `bson:"-" json:"-"`

	// Crafting state (transient)
	CraftingItem  string `bson:"-" json:"-"` // what they're making (e.g., "greatsword")
	CraftingMetal string `bson:"-" json:"-"` // what material (e.g., "copper", "hide")
	CraftingStep  int    `bson:"-" json:"-"` // 0=not crafting, 1=planned, 2+=steps
	CraftingSkill string `bson:"-" json:"-"` // "weaponsmithing", "jewelry", "weaving", "wood"
	CraftingAdj1  int    `bson:"-" json:"-"` // material instance Adj1 (e.g. bronze adjective ID)
	CraftingAdj2  int    `bson:"-" json:"-"`
	CraftingAdj3  int    `bson:"-" json:"-"`
	CraftingVal1  int    `bson:"-" json:"-"`
	CraftingVal2  int    `bson:"-" json:"-"`
	CraftingVal3  int    `bson:"-" json:"-"`
	CraftingVal4  int    `bson:"-" json:"-"`
	CraftingVal5  int    `bson:"-" json:"-"`

	// Teaching: skill or spell being taught to others (transient)
	Teaching      int `bson:"-" json:"-"` // skill number being taught (0 = not teaching a skill)
	TeachingLevel int `bson:"-" json:"-"` // max level to teach up to (teacher's own level)
	TeachingSpell int `bson:"-" json:"-"` // spell number being taught (0 = not teaching a spell)

	// Guard: who/what this player is guarding (transient, room-specific)
	GuardTargets []string `bson:"-" json:"-"` // player FirstNames being guarded
	GuardPortals []int    `bson:"-" json:"-"` // portal item archetypes being guarded (CM level 3+)
	GuardItems   []int    `bson:"-" json:"-"` // item archetypes on the ground being guarded

	// Group system (transient)
	Following     string   `bson:"-" json:"-"` // who this player is following
	GroupMembers  []string `bson:"-" json:"-"` // if this player is a leader, who's in their group
	IsGroupLeader bool     `bson:"-" json:"-"`

	// Social blocking (persistent): AVOID/UNAVOID/ALLOW/UNALLOW. AvoidList
	// blocks physical interactive emotes (KISS, NIBBLE, etc.) and HOLD from
	// the named players; AllowList overrides AvoidList for named players.
	AvoidList []string `bson:"avoidList,omitempty" json:"avoidList,omitempty"`
	AllowList []string `bson:"allowList,omitempty" json:"allowList,omitempty"`

	// Carry system (transient): a submitting or dead player can be picked up
	// and carried between rooms by another player.
	CarriedBy string `bson:"-" json:"-"` // FirstName of the player carrying this player, if any
	Carrying  string `bson:"-" json:"-"` // FirstName of the player this player is carrying, if any

	// Summoned creature (transient — cleared on server restart)
	SummonedCreatureID int `bson:"-" json:"-"` // monster instance ID of active summoned creature (0 = none)

	// Speak with Dead (spell 311): lets a dead player speak despite being otherwise incapacitated
	SpeakWhileDead bool `bson:"-" json:"-"`

	// Regeneration (spell 343): heal-over-time state
	RegenerationAmount    int `bson:"-" json:"-"` // per-tick heal amount
	RegenerationTicksLeft int `bson:"-" json:"-"` // remaining once-per-minute ticks

	// Last command entered, for the "." repeat shortcut
	LastCommand string `bson:"-" json:"-"`

	// Teleport marks (1-10) → room number
	Marks map[int]int `bson:"marks,omitempty" json:"marks,omitempty"`

	// Inventory
	Inventory []InventoryItem `bson:"inventory" json:"inventory"`
	Wielded   *InventoryItem  `bson:"wielded,omitempty" json:"wielded,omitempty"`
	OffHand   *InventoryItem  `bson:"offHand,omitempty" json:"offHand,omitempty"` // shield or off-hand weapon (Two Weapons skill)
	Worn      []InventoryItem `bson:"worn" json:"worn"`

	// Currency (carried)
	Gold   int `bson:"gold" json:"gold"`
	Silver int `bson:"silver" json:"silver"`
	Copper int `bson:"copper" json:"copper"`

	// Currency (banked)
	BankGold   int `bson:"bankGold,omitempty" json:"bankGold,omitempty"`
	BankSilver int `bson:"bankSilver,omitempty" json:"bankSilver,omitempty"`
	BankCopper int `bson:"bankCopper,omitempty" json:"bankCopper,omitempty"`

	// Safety deposit box (up to 20 items)
	BankItems []InventoryItem `bson:"bankItems,omitempty" json:"bankItems,omitempty"`

	// Organization / Guild — OrgMemberships is the source of truth (org# → rank).
	// Organization and OrgRank are kept in sync for legacy MongoDB documents.
	Organization    int         `bson:"organization,omitempty" json:"organization,omitempty"`
	OrgRank         int         `bson:"orgRank,omitempty" json:"orgRank,omitempty"`
	OrgMemberships  map[int]int `bson:"orgMemberships,omitempty" json:"orgMemberships,omitempty"`
	Alignment    int `bson:"alignment,omitempty" json:"alignment,omitempty"`       // ALIGN
	Warrant      int `bson:"warrant,omitempty" json:"warrant,omitempty"`           // warrant level 0-9
	BuildPoints  int `bson:"buildPoints,omitempty" json:"buildPoints,omitempty"`

	// Skills
	Skills      map[int]int  `bson:"skills" json:"skills"`           // skill# -> level
	KnownSpells  map[int]bool `bson:"knownSpells,omitempty" json:"knownSpells,omitempty"` // spell# -> known
	SpellMastery map[int]int  `bson:"spellMastery,omitempty" json:"spellMastery,omitempty"` // spell# -> mastery rank

	// Internal variables (INTNUM0-99, flags, etc.)
	IntNums map[int]int `bson:"intNums" json:"intNums"`

	// Transient flags (reset on room entry)
	Flag1 int `bson:"-" json:"-"`
	Flag2 int `bson:"-" json:"-"`
	Flag3 int `bson:"-" json:"-"`
	Flag4 int `bson:"-" json:"-"`

	// Appearance / Description
	DescLine1  string `bson:"descLine1,omitempty" json:"descLine1,omitempty"` // custom description lines (visible on EXAMINE)
	DescLine2  string `bson:"descLine2,omitempty" json:"descLine2,omitempty"`
	DescLine3  string `bson:"descLine3,omitempty" json:"descLine3,omitempty"`
	Appearance string `bson:"appearance,omitempty" json:"appearance,omitempty"` // player-set line shown on EXAMINE after worn items (APPEARANCE command)
	EntryEcho  string `bson:"entryEcho,omitempty" json:"entryEcho,omitempty"` // custom room entry text (replaces "X arrives.")
	ExitEcho   string `bson:"exitEcho,omitempty" json:"exitEcho,omitempty"`   // custom room exit text (replaces "X goes north.")

	// Bot / API Key
	APIKeyHash    string `bson:"apiKeyHash,omitempty" json:"-"`                       // bcrypt hash of API key (never sent to client)
	APIKeyPrefix  string `bson:"apiKeyPrefix,omitempty" json:"apiKeyPrefix,omitempty"` // first 8 chars for display
	BotGMAllowed  bool   `bson:"botGMAllowed,omitempty" json:"botGMAllowed,omitempty"` // whether bot can use GM commands
	IsBot         bool   `bson:"-" json:"-"`                                           // transient: connected via API key

	// Settings (persistent toggles via SET command)
	// Note: SuppressLogon/Logoff default to true for new characters (set in CreateNewPlayer).
	SuppressLogon      bool `bson:"suppressLogon,omitempty" json:"-"`
	SuppressLogoff     bool `bson:"suppressLogoff,omitempty" json:"-"`
	SuppressDisconnect bool `bson:"suppressDisconnect,omitempty" json:"-"`
	RPBrief            bool `bson:"rpBrief,omitempty" json:"-"`
	BattleBrief        bool `bson:"battleBrief,omitempty" json:"-"`
	ActionBrief        bool `bson:"actionBrief,omitempty" json:"-"`
	ActBrief           bool `bson:"actBrief,omitempty" json:"-"`

	// Game state
	BriefMode    bool   `bson:"briefMode" json:"briefMode"`
	PromptMode   bool   `bson:"promptMode" json:"promptMode"`
	SpeechAdverb string `bson:"speechAdverb,omitempty" json:"speechAdverb,omitempty"` // e.g. "gently"
	IsGM         bool   `bson:"isGM" json:"isGM"`
	GMTrace    bool `bson:"-" json:"-"`                                     // @trace: show script debug output
	GMHat      bool `bson:"gmHat,omitempty" json:"gmHat,omitempty"`        // visible as GM on WHO list
	GMHidden   bool `bson:"gmHidden,omitempty" json:"gmHidden,omitempty"`  // hidden from WHO list
	GMInvis      bool   `bson:"gmInvis,omitempty" json:"gmInvis,omitempty"`    // invisible to players
	GMEditTarget string `bson:"-" json:"-"` // @edpl target name for subsequent @set commands

	// Player title (e.g., "the Baroness") — shown on LOOK/EXAMINE
	Title string `bson:"title,omitempty" json:"title,omitempty"`

	CreatedAt time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt *time.Time `bson:"deletedAt,omitempty" json:"deletedAt,omitempty"` // soft-delete timestamp
}

// FullName returns the player's display name.
func (p *Player) FullName() string {
	return p.FirstName + " " + p.LastName
}

// Pronoun returns "he" or "she".
func (p *Player) Pronoun() string {
	if p.Gender == 0 {
		return "he"
	}
	return "she"
}

// PronounCap returns "He" or "She".
func (p *Player) PronounCap() string {
	if p.Gender == 0 {
		return "He"
	}
	return "She"
}

// Possessive returns "his" or "her".
func (p *Player) Possessive() string {
	if p.Gender == 0 {
		return "his"
	}
	return "her"
}

// PossessiveCap returns "His" or "Her".
func (p *Player) PossessiveCap() string {
	if p.Gender == 0 {
		return "His"
	}
	return "Her"
}

// Objective returns "him" or "her".
func (p *Player) Objective() string {
	if p.Gender == 0 {
		return "him"
	}
	return "her"
}

// PromptIndicators returns the status code string for prompt mode.
// Each condition maps to a letter: ! bleeding, s sitting, S stunned,
// D diseased, J group-joined, P pressed/combat-joined, K kneeling, L laying, R roundtime,
// H hidden/invisible, U unconscious, I immobilized, DEAD dead.
func (p *Player) PromptIndicators() string {
	if !p.PromptMode {
		return ""
	}
	if p.Dead {
		return "DEAD"
	}
	var codes []byte
	if p.Bleeding {
		codes = append(codes, '!')
	}
	if p.Position == 1 { // sitting
		codes = append(codes, 's')
	}
	if p.Stunned {
		codes = append(codes, 'S')
	}
	if p.Diseased {
		codes = append(codes, 'D')
	}
	if p.Poisoned {
		codes = append(codes, 'P')
	}
	if p.Following != "" {
		codes = append(codes, 'J')
	}
	if p.Joined {
		codes = append(codes, 'C') // Combat-joined/pressed
	}
	if p.Position == 3 { // kneeling
		codes = append(codes, 'K')
	}
	if p.Position == 2 { // laying
		codes = append(codes, 'L')
	}
	if p.RoundTime > 0 {
		codes = append(codes, 'R')
	}
	if p.Hidden || p.GMInvis {
		codes = append(codes, 'H')
	}
	if p.Unconscious {
		codes = append(codes, 'U')
	}
	if p.Immobilized {
		codes = append(codes, 'I')
	}
	return string(codes)
}

// RaceName returns the string name of the player's race.
func (p *Player) RaceName() string {
	return RaceNameByID(p.Race)
}

// RaceNameByID returns the race name for a given race ID.
func RaceNameByID(race int) string {
	if name, ok := RaceNames[race]; ok {
		return name
	}
	return "Unknown"
}

// IsFlying returns true if the player is able to fly (Drakin race or magical effect).
func (p *Player) IsFlying() bool {
	return p.Race == RaceDrakin || p.CanFly
}

// RestoreTransientState re-applies persisted buff effects to in-memory transient fields.
// Call once after loading a player from the database on login/reconnect.
// IsMemberOf returns true if the player belongs to the given organization.
func (p *Player) IsMemberOf(orgNum int) bool {
	if len(p.OrgMemberships) > 0 {
		_, ok := p.OrgMemberships[orgNum]
		return ok
	}
	return p.Organization == orgNum && orgNum != 0
}

// RankIn returns the player's rank in the given organization (0 if not a member).
func (p *Player) RankIn(orgNum int) int {
	if len(p.OrgMemberships) > 0 {
		return p.OrgMemberships[orgNum]
	}
	if p.Organization == orgNum {
		return p.OrgRank
	}
	return 0
}

// AddOrg adds the player to an organization with the given rank.
// If the player already belongs, the rank is updated.
// Migrates legacy Organization/OrgRank fields on first call.
func (p *Player) AddOrg(orgNum, rank int) {
	if p.OrgMemberships == nil {
		p.OrgMemberships = make(map[int]int)
		if p.Organization != 0 {
			p.OrgMemberships[p.Organization] = p.OrgRank
		}
	}
	p.OrgMemberships[orgNum] = rank
	// Keep legacy fields pointing at the primary (lowest-numbered) org.
	p.syncLegacyOrgFields()
}

// RemoveOrg removes the player from an organization.
func (p *Player) RemoveOrg(orgNum int) {
	if p.OrgMemberships == nil {
		p.OrgMemberships = make(map[int]int)
		if p.Organization != 0 {
			p.OrgMemberships[p.Organization] = p.OrgRank
		}
	}
	delete(p.OrgMemberships, orgNum)
	p.syncLegacyOrgFields()
}

// OrgList returns all organization numbers the player belongs to, sorted ascending.
func (p *Player) OrgList() []int {
	if len(p.OrgMemberships) > 0 {
		orgs := make([]int, 0, len(p.OrgMemberships))
		for num := range p.OrgMemberships {
			orgs = append(orgs, num)
		}
		for i := 0; i < len(orgs)-1; i++ {
			for j := i + 1; j < len(orgs); j++ {
				if orgs[j] < orgs[i] {
					orgs[i], orgs[j] = orgs[j], orgs[i]
				}
			}
		}
		return orgs
	}
	if p.Organization != 0 {
		return []int{p.Organization}
	}
	return nil
}

// syncLegacyOrgFields keeps Organization/OrgRank in sync with OrgMemberships.
func (p *Player) syncLegacyOrgFields() {
	if len(p.OrgMemberships) == 0 {
		p.Organization = 0
		p.OrgRank = 0
		return
	}
	// Pick lowest org number as the primary for legacy field.
	primary := 0
	for num := range p.OrgMemberships {
		if primary == 0 || num < primary {
			primary = num
		}
	}
	p.Organization = primary
	p.OrgRank = p.OrgMemberships[primary]
}

func (p *Player) RestoreTransientState() {
	if p.MysticArmorBonus > 0 && !p.MysticArmorExpiry.IsZero() && time.Now().Before(p.MysticArmorExpiry) {
		p.DefenseBonus += p.MysticArmorBonus
	} else if p.MysticArmorBonus > 0 {
		p.MysticArmorBonus = 0
		p.MysticArmorExpiry = time.Time{}
	}
	// Restore timed defense buffs, pruning any that expired while offline.
	var active []TimedDefenseBuff
	for _, b := range p.TimedDefenseBuffs {
		if time.Now().Before(b.Expiry) {
			p.DefenseBonus += b.Bonus
			active = append(active, b)
		}
	}
	p.TimedDefenseBuffs = active
}

// applyRoundTime adjusts a base round-time duration (in seconds) for Haste or Slow effects.
// Haste halves the time (floor division); Slow doubles it. Returns base if neither is active.
func applyRoundTime(player *Player, seconds int) int {
	if !player.HasteExpiry.IsZero() && time.Now().Before(player.HasteExpiry) {
		return seconds / 2
	}
	if !player.SlowExpiry.IsZero() && time.Now().Before(player.SlowExpiry) {
		return seconds * 2
	}
	return seconds
}
