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
	Hidden     bool `bson:"hidden" json:"hidden"`
	Dead       bool `bson:"dead" json:"dead"`

	// Status conditions
	Bleeding    bool `bson:"bleeding" json:"bleeding"`
	Stunned     bool `bson:"stunned" json:"stunned"`
	Diseased    bool `bson:"diseased" json:"diseased"`
	Poisoned    bool `bson:"poisoned" json:"poisoned"`
	Joined      bool `bson:"joined" json:"joined"`
	Unconscious bool `bson:"unconscious" json:"unconscious"`
	Immobilized bool `bson:"immobilized" json:"immobilized"`
	RoundTime   int  `bson:"roundTime" json:"roundTime"`
	CanFly      bool `bson:"canFly" json:"canFly"` // granted by spell or magical effect

	// Inventory
	Inventory []InventoryItem `bson:"inventory" json:"inventory"`
	Wielded   *InventoryItem  `bson:"wielded,omitempty" json:"wielded,omitempty"`
	Worn      []InventoryItem `bson:"worn" json:"worn"`

	// Currency (carried)
	Gold   int `bson:"gold" json:"gold"`
	Silver int `bson:"silver" json:"silver"`
	Copper int `bson:"copper" json:"copper"`

	// Currency (banked)
	BankGold   int `bson:"bankGold,omitempty" json:"bankGold,omitempty"`
	BankSilver int `bson:"bankSilver,omitempty" json:"bankSilver,omitempty"`
	BankCopper int `bson:"bankCopper,omitempty" json:"bankCopper,omitempty"`

	// Organization / Guild
	Organization int `bson:"organization,omitempty" json:"organization,omitempty"` // ORG
	OrgRank      int `bson:"orgRank,omitempty" json:"orgRank,omitempty"`           // ORGRANK
	Alignment    int `bson:"alignment,omitempty" json:"alignment,omitempty"`       // ALIGN
	BuildPoints  int `bson:"buildPoints,omitempty" json:"buildPoints,omitempty"`

	// Skills
	Skills      map[int]int  `bson:"skills" json:"skills"`           // skill# -> level
	KnownSpells map[int]bool `bson:"knownSpells,omitempty" json:"knownSpells,omitempty"` // spell# -> known

	// Internal variables (INTNUM0-99, flags, etc.)
	IntNums map[int]int `bson:"intNums" json:"intNums"`

	// Transient flags (reset on room entry)
	Flag1 int `bson:"-" json:"-"`
	Flag2 int `bson:"-" json:"-"`
	Flag3 int `bson:"-" json:"-"`
	Flag4 int `bson:"-" json:"-"`

	// Game state
	BriefMode    bool   `bson:"briefMode" json:"briefMode"`
	PromptMode   bool   `bson:"promptMode" json:"promptMode"`
	SpeechAdverb string `bson:"speechAdverb,omitempty" json:"speechAdverb,omitempty"` // e.g. "gently"
	IsGM         bool   `bson:"isGM" json:"isGM"`
	GMHat      bool `bson:"-" json:"gmHat,omitempty"`      // visible as GM on WHO list (transient)
	GMHidden   bool `bson:"-" json:"gmHidden,omitempty"`    // hidden from WHO list (transient)
	GMInvis    bool `bson:"-" json:"gmInvis,omitempty"`     // invisible to players (transient)

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// InventoryItem is an instance of an item held by a player.
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
	State     string `bson:"state,omitempty" json:"state,omitempty"`
	WornSlot  string `bson:"wornSlot,omitempty" json:"wornSlot,omitempty"`
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
// D diseased, P poisoned, J joined, K kneeling, L laying, R roundtime,
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
	if p.Joined {
		codes = append(codes, 'J')
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
	if p.Hidden {
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
	if name, ok := RaceNames[p.Race]; ok {
		return name
	}
	return "Unknown"
}

// IsFlying returns true if the player is able to fly (Drakin race or magical effect).
func (p *Player) IsFlying() bool {
	return p.Race == RaceDrakin || p.CanFly
}
