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
	Region           int               `bson:"region,omitempty" json:"region,omitempty"`
	Modifiers        []string          `bson:"modifiers" json:"modifiers"` // FORGE, LOOM, MINEA, etc.
	Scripts          []ScriptBlock     `bson:"scripts" json:"scripts"`     // conditional blocks
	MacroCalls       []int             `bson:"macroCalls,omitempty" json:"macroCalls,omitempty"` // CALL directives; resolved into Scripts at load time
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
	Sharpness int            `bson:"sharpness,omitempty" json:"sharpness,omitempty"` // non-magical to-hit bonus forged into a weapon
	// HardnessMod is a per-instance modifier to a weapon's Weapon Clash break-resistance
	// (see weaponHardness in engine/combat.go), GM-editable via @editem.
	HardnessMod int        `bson:"hardnessMod,omitempty" json:"hardnessMod,omitempty"`
	// ItemBits holds the 0-19 boolean flags documented as ITEMBIT(#) in MANUAL.DOC — a
	// per-instance bitmask distinct from VAL1-5. Bit N corresponds to ITEMBIT<N>.
	ItemBits  int            `bson:"itemBits,omitempty" json:"itemBits,omitempty"`
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

// OrgDef represents an organization defined by an ORGDEF directive.
type OrgDef struct {
	Number       int    `bson:"number" json:"number"`
	JoinType     string `bson:"joinType" json:"joinType"`     // "OPEN" or "INVITE"
	OrgType      string `bson:"orgType" json:"orgType"`       // "GUILD", "TEMPLE", "CULT"
	TrainingRoom int    `bson:"trainingRoom" json:"trainingRoom"`
	Name         string `bson:"name" json:"name"`
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
	MacroCalls  []int    `bson:"macroCalls,omitempty" json:"macroCalls,omitempty"` // CALL directives; resolved into Scripts at load time
	ExamineDesc string   `bson:"examineDesc,omitempty" json:"examineDesc,omitempty"`
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
	SkinItems     []SkinDrop        `bson:"skinItems,omitempty" json:"skinItems,omitempty"`
	TextOverrides  map[string]string `bson:"textOverrides,omitempty" json:"textOverrides,omitempty"`
	Immunities     map[int]int       `bson:"immunities,omitempty" json:"immunities,omitempty"`
	Weapons        []MonsterWeapon   `bson:"weapons,omitempty" json:"weapons,omitempty"`
	WeaponPlus     int               `bson:"weaponPlus,omitempty" json:"weaponPlus,omitempty"`
	MagicWeapon    int               `bson:"magicWeapon,omitempty" json:"magicWeapon,omitempty"`
	SpecUse        int               `bson:"specUse,omitempty" json:"specUse,omitempty"`
	SpecUses       int               `bson:"specUses,omitempty" json:"specUses,omitempty"`
	SpecBase       int               `bson:"specBase,omitempty" json:"specBase,omitempty"`
	SpecDmg        int               `bson:"specDmg,omitempty" json:"specDmg,omitempty"`
	SpecDmgType    string            `bson:"specDmgType,omitempty" json:"specDmgType,omitempty"`
	ExtraBody      int               `bson:"extraBody,omitempty" json:"extraBody,omitempty"`
	NonDisruptable bool              `bson:"nonDisruptable,omitempty" json:"nonDisruptable,omitempty"`
	SilenceIgnore  bool              `bson:"silenceIgnore,omitempty" json:"silenceIgnore,omitempty"`
	FatigueChance  int               `bson:"fatigueChance,omitempty" json:"fatigueChance,omitempty"`
	FatigueLevel   int               `bson:"fatigueLevel,omitempty" json:"fatigueLevel,omitempty"`
	Spells         []int             `bson:"spells,omitempty" json:"spells,omitempty"`
	Psi            int               `bson:"psi,omitempty" json:"psi,omitempty"`
	PsiUse         int               `bson:"psiUse,omitempty" json:"psiUse,omitempty"`
	PsiSkill       int               `bson:"psiSkill,omitempty" json:"psiSkill,omitempty"`
	PsiResist      int               `bson:"psiResist,omitempty" json:"psiResist,omitempty"`
	PsiLevel       int               `bson:"psiLevel,omitempty" json:"psiLevel,omitempty"`
	Disciplines    []int             `bson:"disciplines,omitempty" json:"disciplines,omitempty"`
	Scripts        []ScriptBlock     `bson:"scripts,omitempty" json:"scripts,omitempty"`
	MacroCalls     []int             `bson:"macroCalls,omitempty" json:"macroCalls,omitempty"` // SCRIPTMACRO directives; resolved into Scripts at load time
	SourceFile     string            `bson:"sourceFile" json:"sourceFile"`
}

// SkinDrop represents a possible skin/loot drop from a monster.
type SkinDrop struct {
	Archetype   int `bson:"archetype" json:"archetype"`
	Probability int `bson:"probability" json:"probability"` // weighted probability
	Value       int `bson:"value" json:"value"`             // copper value
	Magic       int `bson:"magic" json:"magic"`             // 0 = non-magical
}

// MonsterWeapon represents a weapon a monster can use.
type MonsterWeapon struct {
	Archetype   int `bson:"archetype" json:"archetype"`
	Adj         int `bson:"adj" json:"adj"`
	Probability int `bson:"probability" json:"probability"`
}

// Macro is a reusable block of script content (IF*/ACTION blocks) defined once
// with MACRO N and attached to rooms/items via CALL N. Resolved into the
// referencing room/item's Scripts at load time.
type Macro struct {
	ID      int           `bson:"id" json:"id"`
	Scripts []ScriptBlock `bson:"scripts" json:"scripts"`
}

// CEvent represents a cyclic event that fires periodically.
type CEvent struct {
	ID      int           `bson:"id" json:"id"`
	Cycles  int           `bson:"cycles" json:"cycles"` // every N cycles (3sec each)
	Room    int           `bson:"room" json:"room"`
	Scripts []ScriptBlock `bson:"scripts" json:"scripts"`
}

// ScriptBlock represents a conditional block (IFVERB...ENDIF, etc.)
type ScriptBlock struct {
	Type string   `bson:"type" json:"type"` // IFVERB, IFPREVERB, IFENTRY, IFSAY, ACTION, etc.
	Args []string `bson:"args" json:"args"`

	// Actions holds a flat action list only for Type=="ACTION" pseudo-blocks — the
	// single-action wrappers used in top-level lists (room/item/monster Scripts,
	// CEVENT/MACRO bodies) for a bare action line with no surrounding conditional.
	Actions []ScriptAction `bson:"actions,omitempty" json:"actions,omitempty"`

	// Body and ElseBody hold a real conditional block's main/ELSE branch as a single
	// ordered sequence of actions and nested blocks, in original source order — e.g.
	// "action1; IFVAR...ENDIF; action2" executes action1, then the nested IFVAR, then
	// action2, rather than every flat action running before any nested block regardless
	// of where it appeared in the source (the old split Actions/Children representation
	// couldn't preserve that interleaving).
	Body     []ScriptStep `bson:"body,omitempty" json:"body,omitempty"`
	ElseBody []ScriptStep `bson:"elseBody,omitempty" json:"elseBody,omitempty"`
}

// ScriptStep is one item in a script block's Body/ElseBody: exactly one of Action or
// Block is set, preserving the original source order of actions and nested conditionals.
type ScriptStep struct {
	Action *ScriptAction `bson:"action,omitempty" json:"action,omitempty"`
	Block  *ScriptBlock  `bson:"block,omitempty" json:"block,omitempty"`
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

// BreakModDef maps an item adjective ID to a hardness/break-chance modifier
// (positive = harder to damage, negative = easier to damage). Parsed from
// BREAKMOD directives, e.g. "BREAKMOD 5 100" for the alzyron adjective.
type BreakModDef struct {
	AdjID    int `bson:"adjId" json:"adjId"`
	Modifier int `bson:"modifier" json:"modifier"`
}

// Variable is a named game variable.
type Variable struct {
	Name  string `bson:"name" json:"name"`
	Value int    `bson:"value" json:"value"`
}

// Region defines a game region with environmental properties.
type Region struct {
	ID               int               `bson:"id" json:"id"`
	Properties       map[string]string `bson:"properties" json:"properties"`
	DepartRoom       int               `bson:"departRoom,omitempty" json:"departRoom,omitempty"`
	HasWeather       bool              `bson:"hasWeather,omitempty" json:"hasWeather,omitempty"`
	Treasure         int               `bson:"treasure,omitempty" json:"treasure,omitempty"`
	TeleportAllowed  bool              `bson:"teleportAllowed,omitempty" json:"teleportAllowed,omitempty"`
	SummoningAllowed bool              `bson:"summoningAllowed,omitempty" json:"summoningAllowed,omitempty"`
	FireMod          int               `bson:"fireMod,omitempty" json:"fireMod,omitempty"`
	ColdMod          int               `bson:"coldMod,omitempty" json:"coldMod,omitempty"`
	ElectricMod      int               `bson:"electricMod,omitempty" json:"electricMod,omitempty"`
	MineAdj          int               `bson:"mineAdj,omitempty" json:"mineAdj,omitempty"`
}

// MoneyDef defines a currency type.
type MoneyDef struct {
	ID           int    `bson:"id" json:"id"`
	Name         string `bson:"name" json:"name"`
	ExchangeRate int    `bson:"exchangeRate" json:"exchangeRate"`
	ItemNum      int    `bson:"itemNum" json:"itemNum"`
}

// ForageDef defines a forageable item for a terrain type.
type ForageDef struct {
	Terrain string `bson:"terrain" json:"terrain"`
	ItemNum int    `bson:"itemNum" json:"itemNum"`
	AdjNum  int    `bson:"adjNum" json:"adjNum"`
	Ratio   int    `bson:"ratio" json:"ratio"`
	Val2    int    `bson:"val2" json:"val2"`
	Val5    int    `bson:"val5" json:"val5"`
}

// MineDef defines a mineable item.
type MineDef struct {
	ItemNum int    `bson:"itemNum" json:"itemNum"`
	AdjNum  int    `bson:"adjNum" json:"adjNum"`
	Grade   string `bson:"grade" json:"grade"` // A, B, or C
	Value   int    `bson:"value" json:"value"` // copper value
	Val2    int    `bson:"val2" json:"val2"`
}

// MonsterList maps rooms to monster spawn data.
// Format: MLIST <room> <monsterID> <probability> <maxCount>
// probability = % chance per slot (0-100), maxCount = max monsters in room.
type MonsterList struct {
	Room        int `bson:"room" json:"room"`
	MonsterID   int `bson:"monsterId" json:"monsterId"`
	Probability int `bson:"probability" json:"probability"` // % chance per spawn slot
	MaxCount    int `bson:"maxCount" json:"maxCount"`       // max number in room
}
