package gameworld

// Room represents a game room parsed from script files.
type Room struct {
	Number           int               `bson:"number" json:"number"`
	Name             string            `bson:"name" json:"name"`
	Description      string            `bson:"description" json:"description"`
	Exits            map[string]int    `bson:"exits" json:"exits"`        // direction -> room number
	Items            []RoomItem        `bson:"items" json:"items"`        // items placed in room
	ItemDescriptions map[string]string `bson:"itemDescriptions,omitempty" json:"itemDescriptions,omitempty"` // "EXAMINE:0" or "READ:5" -> description
	Terrain          string            `bson:"terrain" json:"terrain"`
	Lighting         string            `bson:"lighting" json:"lighting"`
	MonsterGroup     int               `bson:"monsterGroup" json:"monsterGroup"`
	StoreItems       []StoreItem       `bson:"storeItems,omitempty" json:"storeItems,omitempty"`
	TrainingSkills   []TrainingDef     `bson:"trainingSkills,omitempty" json:"trainingSkills,omitempty"`
	Modifiers        []string          `bson:"modifiers" json:"modifiers"` // FORGE, LOOM, MINEA, etc.
	Scripts          []ScriptBlock     `bson:"scripts" json:"scripts"`     // conditional blocks
	SourceFile       string            `bson:"sourceFile" json:"sourceFile"`
}

// RoomItem is an item placed in a room via ITEM or PUT commands.
type RoomItem struct {
	Ref       int            `bson:"ref" json:"ref"`             // reference number 0-9
	Archetype int            `bson:"archetype" json:"archetype"` // item INUMBER
	Adj1      int            `bson:"adj1,omitempty" json:"adj1,omitempty"`
	Adj2      int            `bson:"adj2,omitempty" json:"adj2,omitempty"`
	Adj3      int            `bson:"adj3,omitempty" json:"adj3,omitempty"`
	Val1      int            `bson:"val1,omitempty" json:"val1,omitempty"`
	Val2      int            `bson:"val2,omitempty" json:"val2,omitempty"`
	Val3      int            `bson:"val3,omitempty" json:"val3,omitempty"`
	Val4      int            `bson:"val4,omitempty" json:"val4,omitempty"`
	Val5      int            `bson:"val5,omitempty" json:"val5,omitempty"`
	State     string         `bson:"state,omitempty" json:"state,omitempty"` // OPEN, CLOSED, LOCKED, etc.
	Extend    string         `bson:"extend,omitempty" json:"extend,omitempty"` // extended description
	PutIn     int            `bson:"putIn,omitempty" json:"putIn,omitempty"` // if PUT, which ref it's inside
	IsPut     bool           `bson:"isPut,omitempty" json:"isPut,omitempty"`
}

// StoreItem represents a purchasable item in a shop room.
type StoreItem struct {
	Archetype int `bson:"archetype" json:"archetype"` // item INUMBER
	Adj       int `bson:"adj,omitempty" json:"adj,omitempty"` // adjective (-1 = none)
	Price     int `bson:"price" json:"price"` // cost in copper
}

// TrainingDef represents a trainable skill in a room.
type TrainingDef struct {
	SkillID  int `bson:"skillId" json:"skillId"`
	MaxLevel int `bson:"maxLevel" json:"maxLevel"`
}

// ItemDef is an item archetype definition from INUMBER blocks.
type ItemDef struct {
	Number      int      `bson:"number" json:"number"`       // INUMBER
	NameID      int      `bson:"nameId" json:"nameId"`       // NAME noun reference
	Type        string   `bson:"type" json:"type"`           // SLASH_WEAPON, ARMOR, FOOD, etc.
	Weight      int      `bson:"weight" json:"weight"`
	Volume      int      `bson:"volume" json:"volume"`
	Substance   string   `bson:"substance" json:"substance"`
	Article     string   `bson:"article" json:"article"`     // A, AN, THE, SOME
	Parameter1  int      `bson:"parameter1" json:"parameter1"`
	Parameter2  int      `bson:"parameter2" json:"parameter2"`
	Parameter3  int      `bson:"parameter3" json:"parameter3"`
	Container   string   `bson:"container,omitempty" json:"container,omitempty"` // IN, ON, UNDER, BEHIND
	Interior    int      `bson:"interior,omitempty" json:"interior,omitempty"`
	WornSlot    string   `bson:"wornSlot,omitempty" json:"wornSlot,omitempty"`
	Flags       []string `bson:"flags" json:"flags"` // HIDDEN, LOCKABLE, OPENABLE, etc.
	Scripts     []ScriptBlock `bson:"scripts,omitempty" json:"scripts,omitempty"`
	SourceFile  string   `bson:"sourceFile" json:"sourceFile"`
}

// MonsterDef defines a monster type.
type MonsterDef struct {
	Number      int      `bson:"number" json:"number"`
	Name        string   `bson:"name" json:"name"`
	Adjective   int      `bson:"adjective" json:"adjective"`
	Description string   `bson:"description" json:"description"`
	BodyType    string   `bson:"bodyType" json:"bodyType"` // HUMAN, ANIMAL, AVINE
	Body        int      `bson:"body" json:"body"`         // hit points
	Attack1     int      `bson:"attack1" json:"attack1"`
	Attack2     int      `bson:"attack2" json:"attack2"`
	Defense     int      `bson:"defense" json:"defense"`
	Strategy    int      `bson:"strategy" json:"strategy"`
	Treasure    int      `bson:"treasure" json:"treasure"`
	Speed       int      `bson:"speed" json:"speed"`
	Armor       int      `bson:"armor" json:"armor"`
	Race        int      `bson:"race" json:"race"`
	Gender      int      `bson:"gender" json:"gender"`
	Unique        bool              `bson:"unique" json:"unique"`
	Stealable     bool              `bson:"stealable,omitempty" json:"stealable,omitempty"`
	Eternal       bool              `bson:"eternal,omitempty" json:"eternal,omitempty"`
	Discorporate  bool              `bson:"discorporate,omitempty" json:"discorporate,omitempty"`
	Alignment     int               `bson:"alignment,omitempty" json:"alignment,omitempty"`
	MagicResist   int               `bson:"magicResist,omitempty" json:"magicResist,omitempty"`
	HideSkill     int               `bson:"hideSkill,omitempty" json:"hideSkill,omitempty"`
	GuardItem     int               `bson:"guardItem,omitempty" json:"guardItem,omitempty"`
	Mana          int               `bson:"mana,omitempty" json:"mana,omitempty"`
	SpellUse      int               `bson:"spellUse,omitempty" json:"spellUse,omitempty"`
	SpellSkill    int               `bson:"spellSkill,omitempty" json:"spellSkill,omitempty"`
	CastLevel     int               `bson:"castLevel,omitempty" json:"castLevel,omitempty"`
	PoisonChance  int               `bson:"poisonChance,omitempty" json:"poisonChance,omitempty"`
	PoisonLevel   int               `bson:"poisonLevel,omitempty" json:"poisonLevel,omitempty"`
	DiseaseChance int               `bson:"diseaseChance,omitempty" json:"diseaseChance,omitempty"`
	DiseaseLevel  int               `bson:"diseaseLevel,omitempty" json:"diseaseLevel,omitempty"`
	SkinAdj       int               `bson:"skinAdj,omitempty" json:"skinAdj,omitempty"`
	SkinItem      int               `bson:"skinItem,omitempty" json:"skinItem,omitempty"`
	TextOverrides map[string]string `bson:"textOverrides,omitempty" json:"textOverrides,omitempty"`
	Scripts       []ScriptBlock     `bson:"scripts,omitempty" json:"scripts,omitempty"`
	SourceFile    string            `bson:"sourceFile" json:"sourceFile"`
}

// ScriptBlock represents a conditional block (IFVERB...ENDIF, etc.)
type ScriptBlock struct {
	Type      string        `bson:"type" json:"type"`           // IFVERB, IFPREVERB, IFENTRY, IFSAY, etc.
	Args      []string      `bson:"args" json:"args"`
	Actions   []ScriptAction `bson:"actions" json:"actions"`
	Children  []ScriptBlock `bson:"children,omitempty" json:"children,omitempty"` // nested IFs
}

// ScriptAction represents a command inside a conditional block.
type ScriptAction struct {
	Command string   `bson:"command" json:"command"` // ECHO, MOVE, SPELL, CLEARVERB, NEWITEM, etc.
	Args    []string `bson:"args" json:"args"`
}

// NounDef maps noun IDs to names.
type NounDef struct {
	ID   int    `bson:"id" json:"id"`
	Name string `bson:"name" json:"name"`
}

// AdjDef maps adjective IDs to names.
type AdjDef struct {
	ID   int    `bson:"id" json:"id"`
	Name string `bson:"name" json:"name"`
}

// MonsterAdjDef maps monster adjective IDs to names.
type MonsterAdjDef struct {
	ID   int    `bson:"id" json:"id"`
	Name string `bson:"name" json:"name"`
}

// Variable is a named game variable.
type Variable struct {
	Name  string `bson:"name" json:"name"`
	Value int    `bson:"value" json:"value"`
}

// Region defines a game region with environmental properties.
type Region struct {
	ID         int               `bson:"id" json:"id"`
	Properties map[string]string `bson:"properties" json:"properties"`
}

// MonsterList maps rooms to monster spawn data.
type MonsterList struct {
	Room      int `bson:"room" json:"room"`
	MonsterID int `bson:"monsterId" json:"monsterId"`
	Min       int `bson:"min" json:"min"`
	Max       int `bson:"max" json:"max"`
}
