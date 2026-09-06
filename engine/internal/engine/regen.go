package engine

import (
	"context"
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

		healMult := 1.0

		if room := e.rooms[p.RoomNumber]; room != nil && containsModifier(room.Modifiers, "HEALER") {
			if p.Position == 1 || p.Position == 2 {
				healMult *= 2.0
			}
		}
		// Position multiplier
		mult := positionMultiplier(p.Position) * healMult

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
