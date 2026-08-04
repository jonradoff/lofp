package engine

import (
	"strings"
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
    // HardnessMod is a per-instance modifier to a weapon's break-resistance in Weapon
    // Clash (see weaponHardness in combat.go), on top of the archetype's weight/damage
    // base and any adjective BREAKMODs. GM-editable via @editem; kept as its own field
    // rather than a Val slot since Val1-5 are all already claimed for other weapon uses.
    HardnessMod int    `bson:"hardnessMod,omitempty" json:"hardnessMod,omitempty"`
    // ItemBits holds the 0-19 boolean flags documented as ITEMBIT(#) in MANUAL.DOC — a
    // per-instance bitmask distinct from VAL1-5 (each of which already has its own
    // documented meaning; see project_item_val_fields memory). Bit N corresponds to
    // ITEMBIT<N>, read/written via IFVAR ITEMBIT#/EQUAL ITEMBIT# in scripts.
    ItemBits  int    `bson:"itemBits,omitempty" json:"itemBits,omitempty"`
    State     string `bson:"state,omitempty" json:"state,omitempty"`
    Tail      string `bson:"tail,omitempty" json:"tail,omitempty"`
    WornSlot  string `bson:"wornSlot,omitempty" json:"wornSlot,omitempty"`
    // Container contents — populated when this item is an open container.
    Contents  []InventoryItem `bson:"contents,omitempty" json:"contents,omitempty"`
    // BladeSpellExpiry marks a timed elemental-crit imbue from Storm/Inferno/Winter
    // Blade (135/136/137, see applyWeaponBladeBuff in spells.go). While active, Val3/
    // Val5 hold the spell's crit type/max damage; BladeSpellPrevVal3/5 hold the
    // weapon's original values (its own permanent crit, if any) to restore on expiry.
    BladeSpellExpiry   time.Time `bson:"bladeSpellExpiry,omitempty" json:"bladeSpellExpiry,omitempty"`
    BladeSpellPrevVal3 int       `bson:"bladeSpellPrevVal3,omitempty" json:"bladeSpellPrevVal3,omitempty"`
    BladeSpellPrevVal5 int       `bson:"bladeSpellPrevVal5,omitempty" json:"bladeSpellPrevVal5,omitempty"`
    // BladeSpellAdjApplied is true when Adj1 currently holds the blade spell's
    // elemental adjective (fiery/icy/electric) — meaning it was free (or freed via
    // shiftAdjToFreeSlot) at cast time. Only then does expiry clear Adj1 back to 0;
    // a weapon with all three adj slots already full at cast time keeps its existing
    // adjectives untouched (same convention as castEnchantmentSpell).
    BladeSpellAdjApplied bool `bson:"bladeSpellAdjApplied,omitempty" json:"bladeSpellAdjApplied,omitempty"`
}

// TimedDefenseBuff tracks one active defense spell with a bonus and expiry.
type TimedDefenseBuff struct {
	SpellID   int       `bson:"spellID" json:"spellID"`
	SpellName string    `bson:"spellName" json:"spellName"`
	Bonus     int       `bson:"bonus" json:"bonus"`
	Expiry    time.Time `bson:"expiry" json:"expiry"`
}

// PlayerEntangle tracks one active movement-restricting spell on a player
// (e.g. Plant Snare) so Freedom (505) can name and remove one at random.
type PlayerEntangle struct {
	SpellID   int       `bson:"spellId" json:"spellId"`
	SpellName string    `bson:"spellName" json:"spellName"`
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
	EyeColor   string `bson:"eyeColor,omitempty" json:"eyeColor,omitempty"`
	HairColor  string `bson:"hairColor,omitempty" json:"hairColor,omitempty"` // empty when HairStyle is "bald"
	HairStyle  string `bson:"hairStyle,omitempty" json:"hairStyle,omitempty"`
	SkinColor  string `bson:"skinColor,omitempty" json:"skinColor,omitempty"`

	// Status conditions
	Bleeding     bool    `bson:"bleeding" json:"bleeding"` // derived: recomputed from Wounds after every add/remove
	Wounds       []Wound `bson:"wounds,omitempty" json:"wounds,omitempty"`
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

	// Temporary Camouflage buff (spell 521): +10 effective Stealth skill (33).
	// Applied on top of the trained skill level in formulas, never written into
	// Skills[33] itself, so SKILLS display/prerequisites still reflect real training.
	CamouflageBonus  int       `bson:"camouflageBonus,omitempty" json:"camouflageBonus,omitempty"`
	CamouflageExpiry time.Time `bson:"camouflageExpiry,omitempty" json:"camouflageExpiry,omitempty"`

	// Resist Weather (spell 506): while active, ignores the Hurricane knockdown
	// chance (regen.go) and the weather-based to-hit penalty (combat.go weatherMod).
	ResistWeatherExpiry time.Time `bson:"resistWeatherExpiry,omitempty" json:"resistWeatherExpiry,omitempty"`

	// Claw Growth (spell 518): while active and the caster has no weapon wielded,
	// attacks use natural claws (ITEMWEAP.SCR #279, CLAW_WEAPON) instead of bare
	// hands/Martial Arts — see currentWeaponDef in combat.go. Self-only; re-casting
	// while active adds another 20 minutes rather than resetting the timer.
	ClawGrowthExpiry time.Time `bson:"clawGrowthExpiry,omitempty" json:"clawGrowthExpiry,omitempty"`

	// Repel Plants (509) / Repel Plants and Webs (510): grant immunity to being
	// newly entangled by Plant Snare (500), and — for 510 only — Web (127) as
	// well. Checked by castPlantSnareSpell before adding an Entangles entry.
	RepelPlantsExpiry        time.Time `bson:"repelPlantsExpiry,omitempty" json:"repelPlantsExpiry,omitempty"`
	RepelPlantsAndWebsExpiry time.Time `bson:"repelPlantsAndWebsExpiry,omitempty" json:"repelPlantsAndWebsExpiry,omitempty"`

	// Entangles tracks active movement-restricting spells (Plant Snare, and any
	// future player-targeting Web/Tentacles) so Freedom (505) can remove one at
	// random. Distinct from the older, unnamed Immobilized flag below (used by
	// psi Immobilize and Imprisonment Rune traps), which Freedom does not touch.
	Entangles []PlayerEntangle `bson:"entangles,omitempty" json:"entangles,omitempty"`

	// Pending summons (e.g. Call the Pack): room to teleport to via ANSWER, and when the
	// invite lapses. Transient — a short-lived (~1 minute) in-memory invite, not meant to
	// survive a restart. PendingSummonsRoom == 0 means no active summons.
	PendingSummonsRoom   int       `bson:"-" json:"-"`
	PendingSummonsExpiry time.Time `bson:"-" json:"-"`

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
	WeaponSpecialization map[int]int `bson:"weaponSpecialization,omitempty" json:"weaponSpecialization,omitempty"` // weapon noun ID -> specialization rank

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

	// Disguise (skill 34): DisguiseSlots are saved personas the player has
	// composed via DISGUISE <slot> <field> <value> (numbered 1-5, same convention
	// as Marks above); ActiveDisguise is the currently-worn copy, applied via
	// DISGUISE APPLY <slot> — kept separate from the slots so editing a saved
	// slot later doesn't retroactively change what's currently worn. Disguised
	// (declared earlier, read by scripts) is true while ActiveDisguise is worn.
	DisguiseSlots  map[int]DisguisePersona `bson:"disguiseSlots,omitempty" json:"disguiseSlots,omitempty"`
	ActiveDisguise DisguisePersona         `bson:"activeDisguise,omitempty" json:"activeDisguise,omitempty"`

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

// DisguisePersona holds a set of apparent-identity overrides for the Disguise
// skill (34). Zero/empty fields mean "use my real value" — a low-rank disguise
// might only set Name, leaving everything else the player's true self. Race,
// Gender and Strength are deliberately display-only overlays that never touch
// Player.Race/Gender/Strength, since those drive real mechanics elsewhere
// (racial abilities, carry capacity/combat) — see disguise.go for the field
// level-gating and command handling.
type DisguisePersona struct {
	// Name is the first-name part for every disguise — the only part shown in
	// room lists and broadcasts (see DisplayName), matching how real players
	// are shown by first name only. Below rank 10 it's restricted to one of
	// disguiseCommonNames and LastName is always empty. At rank 10+ it can be
	// a custom persona's first name, optionally paired with LastName — the
	// full "First Last" is only shown on EXAMINE (see DisplayFullName).
	Name      string `bson:"name,omitempty" json:"name,omitempty"`
	LastName  string `bson:"lastName,omitempty" json:"lastName,omitempty"`
	Gender    string `bson:"gender,omitempty" json:"gender,omitempty"` // "male" / "female"
	HairColor string `bson:"hairColor,omitempty" json:"hairColor,omitempty"`
	HairStyle string `bson:"hairStyle,omitempty" json:"hairStyle,omitempty"`
	SkinColor string `bson:"skinColor,omitempty" json:"skinColor,omitempty"`
	EyeColor  string `bson:"eyeColor,omitempty" json:"eyeColor,omitempty"`
	Age       int    `bson:"age,omitempty" json:"age,omitempty"`
	Strength  int    `bson:"strength,omitempty" json:"strength,omitempty"` // apparent build only; never the real combat stat
	Height    int    `bson:"height,omitempty" json:"height,omitempty"`
	Weight    int    `bson:"weight,omitempty" json:"weight,omitempty"`
	Race      int    `bson:"race,omitempty" json:"race,omitempty"`
}

// bareDisplayName is DisplayName lowercased with any leading article ("a ",
// "an ") stripped — the matching-friendly form of the player's current
// apparent identity, used by NameMatches/NameEquals below.
func (p *Player) bareDisplayName() string {
	name := strings.ToLower(p.DisplayName())
	if s, ok := strings.CutPrefix(name, "a "); ok {
		return s
	}
	if s, ok := strings.CutPrefix(name, "an "); ok {
		return s
	}
	return name
}

// NameMatches reports whether query is a case-insensitive prefix of the
// player's current apparent identity — the disguise-aware replacement for
// matching typed target names (LOOK/WHISPER/CONTACT/etc.) against a raw
// FirstName, so typing "wolf" or "commoner" still matches the way it always
// has despite DisplayName's leading article.
func (p *Player) NameMatches(query string) bool {
	return strings.HasPrefix(p.bareDisplayName(), strings.ToLower(query))
}

// NameEquals reports whether query is an exact case-insensitive match of the
// player's current apparent identity — the disguise-aware replacement for
// exact-match targeting (e.g. COMMAND FOLLOW/GUARD/ATTACK <name> in summons.go).
func (p *Player) NameEquals(query string) bool {
	return p.bareDisplayName() == strings.ToLower(query)
}

// EffectiveAppearance returns the player's apparent race/gender/age/height/
// weight/strength/eye/skin/hair-color/hair-style: real values, overridden by
// any non-zero ActiveDisguise field while Disguised is true. Used by LOOK/
// EXAMINE to render what an observer actually sees.
func (p *Player) EffectiveAppearance() (race, gender, age, height, weight, strength int, eye, skin, hairColor, hairStyle string) {
	race, gender, age, height, weight, strength = p.Race, p.Gender, p.Age, p.Height, p.Weight, p.Strength
	eye, skin, hairColor, hairStyle = p.EyeColor, p.SkinColor, p.HairColor, p.HairStyle
	if !p.Disguised {
		return
	}
	d := p.ActiveDisguise
	if d.Race != 0 {
		race = d.Race
	}
	switch d.Gender {
	case "male":
		gender = GenderMale
	case "female":
		gender = GenderFemale
	}
	if d.Age != 0 {
		age = d.Age
	}
	if d.Height != 0 {
		height = d.Height
	}
	if d.Weight != 0 {
		weight = d.Weight
	}
	if d.Strength != 0 {
		strength = d.Strength
	}
	if d.EyeColor != "" {
		eye = d.EyeColor
	}
	if d.SkinColor != "" {
		skin = d.SkinColor
	}
	if d.HairColor != "" {
		hairColor = d.HairColor
	}
	if d.HairStyle != "" {
		hairStyle = d.HairStyle
	}
	return
}

// effectiveStealthSkill returns the player's trained Stealth skill (33) plus
// any active Camouflage buff bonus (spell 521). Used in hide/sneak/detection
// formulas instead of Skills[33] directly so the buff never leaks into the
// persisted skill level shown by SKILLS or checked by Disguise's prerequisite.
func effectiveStealthSkill(p *Player) int {
	skill := p.Skills[33]
	if p.CamouflageBonus > 0 && !p.CamouflageExpiry.IsZero() && time.Now().Before(p.CamouflageExpiry) {
		skill += p.CamouflageBonus
	}
	return skill
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

// IsConcealed reports whether the player's presence should be kept off other
// players' echoes — stealthed (Hidden), under the Invisibility spell, or a GM
// using @hide or @invis. Used to suppress movement/follow/portal echoes
// entirely and to anonymize speech as "Something" instead of the player's name.
func (p *Player) IsConcealed() bool {
	return p.Hidden || p.Invisible || p.GMHidden || p.GMInvis
}

// DisplayName returns the name other players should see when this player acts
// or is looked at/addressed: "a wolf" for a Wolfling in WolfForm, mid-sentence;
// the active disguise's identity while one is worn (Disguise skill, 34) — with
// an article ("a commoner") for the common pre-level-10 names so a disguised
// player is indistinguishable from the real NPCs of that type, or bare for a
// custom level-10+ persona name; or their real FirstName otherwise. See
// DisplayNameCap for sentence-initial use.
func (p *Player) DisplayName() string {
	if p.WolfForm {
		return "a wolf"
	}
	if p.Disguised && p.ActiveDisguise.Name != "" {
		name := p.ActiveDisguise.Name
		if disguisableNPCNames[strings.ToLower(name)] {
			return articleFor(name, false) + name
		}
		return name
	}
	return p.FirstName
}

// DisplayFullName is DisplayName's counterpart for EXAMINE/LOOK <target>,
// which shows a full name rather than a bare first name: "a wolf" for
// WolfForm; the article-prefixed common-NPC name (same as DisplayName) for a
// disguise using one of those; "First Last" (or just "First" with no
// LastName set) for a custom level-10+ persona; or the player's real
// FullName() otherwise.
func (p *Player) DisplayFullName() string {
	if p.WolfForm {
		return "a wolf"
	}
	if p.Disguised && p.ActiveDisguise.Name != "" {
		if disguisableNPCNames[strings.ToLower(p.ActiveDisguise.Name)] {
			return p.DisplayName()
		}
		if p.ActiveDisguise.LastName != "" {
			return p.ActiveDisguise.Name + " " + p.ActiveDisguise.LastName
		}
		return p.ActiveDisguise.Name
	}
	return p.FullName()
}

// DisplayNameCap is DisplayName capitalized for sentence-initial placement
// (e.g. "A wolf growls...", "A commoner arrives."). FirstName and custom
// disguise names are already capitalized, so this only differs from
// DisplayName for WolfForm and the article-prefixed common disguise names.
func (p *Player) DisplayNameCap() string {
	return capitalize(p.DisplayName())
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
