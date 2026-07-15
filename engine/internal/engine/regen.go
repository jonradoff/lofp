package engine

import (
	"context"
	"fmt"
	"time"
)

// StartRegenCycle starts a background goroutine that ticks every real minute
// and regenerates fatigue, mana, PSI, and body points for all online players
// based on their stats and position.
//
// Regen rates (per tick):
//   - Fatigue: Constitution / 20 (min 1)
//   - Mana: (Willpower + Empathy) / 30 (min 1)
//   - PSI: Willpower / 20 (min 1)
//   - Body Points: Constitution / 50 (min 1, only when injured)
//
// Position multipliers:
//   - Standing (0): 1.0x
//   - Sitting (1): 2.0x
//   - Laying (2): 3.0x
//   - Kneeling (3): 1.5x
//   - Flying (4): 1.0x
func (e *GameEngine) StartRegenCycle() {
	go func() {
		ticker := time.NewTicker(60 * time.Second) // 1 real minute
		defer ticker.Stop()
		for range ticker.C {
			e.regenTick()
		}
	}()
}

func (e *GameEngine) regenTick() {
	if e.sessions == nil {
		return
	}
	players := e.sessions.OnlinePlayers()
	for _, p := range players {
		if p == nil || p.Dead {
			continue
		}

		// Position multiplier
		mult := positionMultiplier(p.Position)

		changed := false

		// Fatigue regen
		if p.Fatigue < p.MaxFatigue {
			base := p.Constitution / 20
			if base < 1 {
				base = 1
			}
			gain := int(float64(base) * mult)
			if gain < 1 {
				gain = 1
			}
			p.Fatigue += gain
			if p.Fatigue > p.MaxFatigue {
				p.Fatigue = p.MaxFatigue
			}
			changed = true
		}

		// Mana regen
		if p.Mana < p.MaxMana {
			base := (p.Willpower + p.Empathy) / 30
			if base < 1 {
				base = 1
			}
			gain := int(float64(base) * mult)
			if gain < 1 {
				gain = 1
			}
			p.Mana += gain
			if p.Mana > p.MaxMana {
				p.Mana = p.MaxMana
			}
			changed = true
		}

		// PSI regen
		if p.Psi < p.MaxPsi {
			base := p.Willpower / 20
			if base < 1 {
				base = 1
			}
			gain := int(float64(base) * mult)
			if gain < 1 {
				gain = 1
			}
			p.Psi += gain
			if p.Psi > p.MaxPsi {
				p.Psi = p.MaxPsi
			}
			changed = true
		}

		// Body point regen (slow natural healing)
		if p.BodyPoints < p.MaxBodyPoints {
			base := p.Constitution / 50
			if base < 1 {
				base = 1
			}
			gain := int(float64(base) * mult)
			if gain < 1 {
				gain = 1
			}
			p.BodyPoints += gain
			if p.BodyPoints > p.MaxBodyPoints {
				p.BodyPoints = p.MaxBodyPoints
			}
			changed = true
		}

		// Poison damage
		if p.Poisoned {
			lvl := p.PoisonLevel
			if lvl < 1 {
				lvl = 1
			}
			p.BodyPoints -= lvl
			if p.BodyPoints < 0 {
				p.BodyPoints = 0
			}
			changed = true
			if e.sendToPlayer != nil {
				e.sendToPlayer(p.FirstName, []string{fmt.Sprintf("The poison courses through your veins! [-%d BP, %d/%d]", lvl, p.BodyPoints, p.MaxBodyPoints)})
			}
		}

		// Disease damage
		if p.Diseased {
			lvl := p.DiseaseLevel
			if lvl < 1 {
				lvl = 1
			}
			p.BodyPoints -= lvl
			if p.BodyPoints < 0 {
				p.BodyPoints = 0
			}
			changed = true
			if e.sendToPlayer != nil {
				e.sendToPlayer(p.FirstName, []string{fmt.Sprintf("The sickness ravages your body! [-%d BP, %d/%d]", lvl, p.BodyPoints, p.MaxBodyPoints)})
			}
		}

		// Check for expired strength buff
		if p.StrengthBuffID > 0 && !p.StrengthBuffExpiry.IsZero() && time.Now().After(p.StrengthBuffExpiry) {
			p.Strength -= p.StrengthBuffBonus
			p.StrengthBuffID = 0
			p.StrengthBuffBonus = 0
			p.StrengthBuffExpiry = time.Time{}
			changed = true
			if e.sendToPlayer != nil {
				e.sendToPlayer(p.FirstName, []string{"The magical strength fades. You feel your normal strength return."})
			}
		}

		// Check for expired agility buff
		if p.AgilityBuffID > 0 && !p.AgilityBuffExpiry.IsZero() && time.Now().After(p.AgilityBuffExpiry) {
			p.Agility -= p.AgilityBuffBonus
			p.AgilityBuffID = 0
			p.AgilityBuffBonus = 0
			p.AgilityBuffExpiry = time.Time{}
			changed = true
			if e.sendToPlayer != nil {
				e.sendToPlayer(p.FirstName, []string{"The magical agility fades. You feel your normal reflexes return."})
			}
		}

		// Check for expired Heat Shield / Cold Shield
		if !p.HeatShieldExpiry.IsZero() && time.Now().After(p.HeatShieldExpiry) {
			p.HeatShieldExpiry = time.Time{}
			changed = true
			if e.sendToPlayer != nil {
				e.sendToPlayer(p.FirstName, []string{"Your resistance to heat fades."})
			}
		}
		if !p.ColdShieldExpiry.IsZero() && time.Now().After(p.ColdShieldExpiry) {
			p.ColdShieldExpiry = time.Time{}
			changed = true
			if e.sendToPlayer != nil {
				e.sendToPlayer(p.FirstName, []string{"Your resistance to cold fades."})
			}
		}

		// Check for expired Mystic Armor buff
		if p.MysticArmorBonus > 0 && !p.MysticArmorExpiry.IsZero() && time.Now().After(p.MysticArmorExpiry) {
			p.DefenseBonus -= p.MysticArmorBonus
			if p.DefenseBonus < 0 {
				p.DefenseBonus = 0
			}
			p.MysticArmorBonus = 0
			p.MysticArmorExpiry = time.Time{}
			changed = true
			if e.sendToPlayer != nil {
				e.sendToPlayer(p.FirstName, []string{"The Mystic Armor fades. The shimmering barrier around you dissipates."})
			}
		}

		// Check for expired timed defense buffs (all defense spells except Mystic Armor)
		var activeBuffs []TimedDefenseBuff
		for _, b := range p.TimedDefenseBuffs {
			if time.Now().After(b.Expiry) {
				p.DefenseBonus -= b.Bonus
				if p.DefenseBonus < 0 {
					p.DefenseBonus = 0
				}
				changed = true
				if e.sendToPlayer != nil {
					e.sendToPlayer(p.FirstName, []string{fmt.Sprintf("The %s fades.", b.SpellName)})
				}
			} else {
				activeBuffs = append(activeBuffs, b)
			}
		}
		if len(activeBuffs) != len(p.TimedDefenseBuffs) {
			p.TimedDefenseBuffs = activeBuffs
		}

		// Check for expired Haste buff
		if !p.HasteExpiry.IsZero() && time.Now().After(p.HasteExpiry) {
			p.HasteExpiry = time.Time{}
			changed = true
			if e.sendToPlayer != nil {
				e.sendToPlayer(p.FirstName, []string{"The magical haste fades. You feel yourself return to normal speed."})
			}
		}

		// Regeneration (spell 343) heal-over-time: once per minute for up to 5 more ticks.
		if p.RegenerationTicksLeft > 0 {
			if p.BodyPoints < p.MaxBodyPoints {
				p.BodyPoints += p.RegenerationAmount
				if p.BodyPoints > p.MaxBodyPoints {
					p.BodyPoints = p.MaxBodyPoints
				}
				if e.sendToPlayer != nil {
					e.sendToPlayer(p.FirstName, []string{fmt.Sprintf("Regeneration knits your wounds. [+%d BP, %d/%d]", p.RegenerationAmount, p.BodyPoints, p.MaxBodyPoints)})
				}
			}
			p.RegenerationTicksLeft--
			if p.RegenerationTicksLeft <= 0 {
				p.RegenerationAmount = 0
			}
			changed = true
		}

		// Check for expired Fly buff
		if !p.FlyExpiry.IsZero() && time.Now().After(p.FlyExpiry) {
			p.FlyExpiry = time.Time{}
			p.CanFly = false
			changed = true
			if p.Position == 4 {
				p.Position = 0
				if e.roomBroadcast != nil {
					e.roomBroadcast(p.RoomNumber, []string{fmt.Sprintf("%s drifts to the ground as the magic fades.", p.FirstName)})
				}
			}
			if e.sendToPlayer != nil {
				e.sendToPlayer(p.FirstName, []string{"The magic sustaining your flight fades. You settle back to the ground."})
			}
		}

		// Check for expired Slow debuff
		if !p.SlowExpiry.IsZero() && time.Now().After(p.SlowExpiry) {
			p.SlowExpiry = time.Time{}
			changed = true
			if e.sendToPlayer != nil {
				e.sendToPlayer(p.FirstName, []string{"The magical slowness fades. You feel yourself return to normal speed."})
			}
		}

		if changed {
			e.SavePlayer(context.Background(), p)
		}
	}
}

func positionMultiplier(position int) float64 {
	switch position {
	case 0: // standing
		return 1.0
	case 1: // sitting
		return 2.0
	case 2: // laying
		return 3.0
	case 3: // kneeling
		return 1.5
	case 4: // flying
		return 1.0
	default:
		return 1.0
	}
}
