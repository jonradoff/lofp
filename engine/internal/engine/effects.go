package engine

import "time"

type StatID int

const (
	StatStrength StatID = iota
	StatAgility
	StatQuickness
	StatConstitution
	StatPerception
	StatWillpower
	StatEmpathy
	HasteBuff
	SlowDebuff
	StatBodyPoint
	StatFatigue
	StatMana
	StatPsi
	DefensiveBuff
	LightBuff
	NightVisionBuff
	RemovePoison
	RemoveDisease
	StunnedEffect
	HeatResistance
	ColdResistance
	RestrainedEffect
	ClawGrowth
	UnconsciousEffect
	FearEffect
	SleepEffect
	CharmEffect
	ResistWeatherEffect
	InvisibilityEffect
)

type EffectSource int

const (
	EffectSourceSpell EffectSource = iota
	EffectSourcePotion
	EffectSourcePoison
	EffectSourceDisease
	EffectSourceItem
	EffectSourceGM
	EffectSourceScript
	EffectSourceEncumbrance
	EffectStunned
	EffectRestrained
	EffectUnconscious
	EffectCharm
)

type StatEffect struct {
	EffectID  int          `bson:"effectId" json:"effectId"`
	Source    EffectSource `bson:"source" json:"source"`
	Stat      StatID       `bson:"stat" json:"stat"`
	Modifier  int          `bson:"modifier" json:"modifier"`
	ExpiresAt time.Time    `bson:"expiresAt,omitempty" json:"expiresAt,omitempty"`
	Permanent bool         `bson:"permanent,omitempty" json:"permanent,omitempty"`
	Ticks     int          `bson:"ticks,omitempty" json:"ticks,omitempty"`
}
