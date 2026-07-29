package engine

import (
	"context"
	"fmt"
	"math/rand"
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

		// Snapshot taken before any of this tick's regen/DoT effects run: were they
		// already unconscious (at 0 body points, presumably from a bleed/poison/disease
		// tick on some earlier minute) when this tick started? Poison/bleeding/disease
		// never kill outright on the tick that first knocks someone to 0 — that clamps
		// to 0 and knocks them out instead, giving other players a window to TEND them
		// or cast Body Restoration/Cure Poison/Cure Disease. But if the same condition
		// is still active and they're still unconscious from a PRIOR tick, the next tick
		// does kill them — this is what makes the reprieve a real race against time
		// rather than a permanent immunity. Using a snapshot from before this tick's own
		// effects run (rather than re-checking live after each DoT block) means multiple
		// conditions ticking in the same minute (e.g. both poisoned and bleeding) can only
		// ever knock a previously-conscious player out once, never stack into a same-tick
		// kill.
		wasUnconsciousAtZero := p.Unconscious && p.BodyPoints <= 0
		died := false

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
		if p.Poisoned && !died {
			lvl := p.PoisonLevel
			if lvl < 1 {
				lvl = 1
			}
			if wasUnconsciousAtZero {
				died = true
				changed = true
				if e.sendToPlayer != nil {
					e.sendToPlayer(p.FirstName, e.handlePlayerDeath(p, "poison"))
				}
			} else {
				p.BodyPoints -= lvl
				if p.BodyPoints <= 0 {
					p.BodyPoints = 0
					p.Unconscious = true
					p.Position = 2
					changed = true
					if e.sendToPlayer != nil {
						e.sendToPlayer(p.FirstName, []string{
							fmt.Sprintf("The poison courses through your veins! [-%d BP, %d/%d]", lvl, p.BodyPoints, p.MaxBodyPoints),
							"You collapse, unconscious!",
						})
					}
				} else {
					changed = true
					if e.sendToPlayer != nil {
						e.sendToPlayer(p.FirstName, []string{fmt.Sprintf("The poison courses through your veins! [-%d BP, %d/%d]", lvl, p.BodyPoints, p.MaxBodyPoints)})
					}
				}
			}
		}

		// Bleeding damage (from unhealed slash/puncture wounds)
		if p.Bleeding && !died {
			total := 0
			for _, w := range p.Wounds {
				if w.Bleeding {
					total += woundBleedDrainPerMinute(w.Level)
				}
			}
			if total > 0 {
				if wasUnconsciousAtZero {
					died = true
					changed = true
					if e.sendToPlayer != nil {
						e.sendToPlayer(p.FirstName, e.handlePlayerDeath(p, "blood loss"))
					}
				} else {
					p.BodyPoints -= total
					if p.BodyPoints <= 0 {
						p.BodyPoints = 0
						p.Unconscious = true
						p.Position = 2
						changed = true
						if e.sendToPlayer != nil {
							e.sendToPlayer(p.FirstName, []string{
								fmt.Sprintf("Your wounds bleed! [-%d BP, %d/%d]", total, p.BodyPoints, p.MaxBodyPoints),
								"You collapse, unconscious!",
							})
						}
					} else {
						changed = true
						if e.sendToPlayer != nil {
							e.sendToPlayer(p.FirstName, []string{fmt.Sprintf("Your wounds bleed! [-%d BP, %d/%d]", total, p.BodyPoints, p.MaxBodyPoints)})
						}
					}
				}
			}
		}

		// Disease damage
		if p.Diseased && !died {
			lvl := p.DiseaseLevel
			if lvl < 1 {
				lvl = 1
			}
			if wasUnconsciousAtZero {
				died = true
				changed = true
				if e.sendToPlayer != nil {
					e.sendToPlayer(p.FirstName, e.handlePlayerDeath(p, "disease"))
				}
			} else {
				p.BodyPoints -= lvl
				if p.BodyPoints <= 0 {
					p.BodyPoints = 0
					p.Unconscious = true
					p.Position = 2
					changed = true
					if e.sendToPlayer != nil {
						e.sendToPlayer(p.FirstName, []string{
							fmt.Sprintf("The sickness ravages your body! [-%d BP, %d/%d]", lvl, p.BodyPoints, p.MaxBodyPoints),
							"You collapse, unconscious!",
						})
					}
				} else {
					changed = true
					if e.sendToPlayer != nil {
						e.sendToPlayer(p.FirstName, []string{fmt.Sprintf("The sickness ravages your body! [-%d BP, %d/%d]", lvl, p.BodyPoints, p.MaxBodyPoints)})
					}
				}
			}
		}

		if died {
			if changed {
				e.SavePlayer(context.Background(), p)
			}
			continue
		}

		// Coming out of unconsciousness naturally (regen ticked them back above 0
		// without anyone's help) — healing spells and TEND wake the player immediately
		// themselves; this is the passive fallback.
		if wakeMsg := wakeFromUnconscious(p); wakeMsg != "" {
			changed = true
			if e.sendToPlayer != nil {
				e.sendToPlayer(p.FirstName, []string{wakeMsg})
			}
			if e.roomBroadcast != nil {
				e.roomBroadcast(p.RoomNumber, []string{fmt.Sprintf("%s stirs and regains consciousness.", p.FirstName)})
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

		// Check for expired Resist Weather buff (spell 506)
		if !p.ResistWeatherExpiry.IsZero() && time.Now().After(p.ResistWeatherExpiry) {
			p.ResistWeatherExpiry = time.Time{}
			changed = true
			if e.sendToPlayer != nil {
				e.sendToPlayer(p.FirstName, []string{"Your resistance to the weather fades."})
			}
		}

		// Check for expired Repel Plants / Repel Plants and Webs buffs (509/510)
		if !p.RepelPlantsExpiry.IsZero() && time.Now().After(p.RepelPlantsExpiry) {
			p.RepelPlantsExpiry = time.Time{}
			changed = true
			if e.sendToPlayer != nil {
				e.sendToPlayer(p.FirstName, []string{"Your immunity to plant snares fades."})
			}
		}
		if !p.RepelPlantsAndWebsExpiry.IsZero() && time.Now().After(p.RepelPlantsAndWebsExpiry) {
			p.RepelPlantsAndWebsExpiry = time.Time{}
			changed = true
			if e.sendToPlayer != nil {
				e.sendToPlayer(p.FirstName, []string{"Your immunity to plant snares and webs fades."})
			}
		}

		// Check for expired Entangles (e.g. Plant Snare) — each has its own
		// duration, independent of the others, unlike Freedom which removes one
		// early at random.
		var activeEntangles []PlayerEntangle
		for _, ent := range p.Entangles {
			if time.Now().After(ent.Expiry) {
				changed = true
				if e.sendToPlayer != nil {
					e.sendToPlayer(p.FirstName, []string{fmt.Sprintf("The %s releases its hold on you.", ent.SpellName)})
				}
			} else {
				activeEntangles = append(activeEntangles, ent)
			}
		}
		if len(activeEntangles) != len(p.Entangles) {
			p.Entangles = activeEntangles
		}

		// Check for expired Camouflage buff (spell 521)
		if p.CamouflageBonus > 0 && !p.CamouflageExpiry.IsZero() && time.Now().After(p.CamouflageExpiry) {
			p.CamouflageBonus = 0
			p.CamouflageExpiry = time.Time{}
			changed = true
			if e.sendToPlayer != nil {
				e.sendToPlayer(p.FirstName, []string{"Your camouflage fades."})
			}
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

		// Hurricane-force winds (Call Storm's max weather state, or a naturally
		// occurring Hurricane) have a chance each tick to knock an outdoor player
		// off their feet. Heavier, more agile players are less likely to fall —
		// see castCallStormSpell in spells.go for how regions reach this state.
		// Resist Weather (506) ignores this entirely.
		resistingWeather := !p.ResistWeatherExpiry.IsZero() && time.Now().Before(p.ResistWeatherExpiry)
		if !resistingWeather && (p.Position == 0 || p.Position == 1 || p.Position == 3) {
			if room := e.rooms[p.RoomNumber]; room != nil && isOutdoorTerrain(room.Terrain) {
				if e.RegionWeather[room.Region] == 8 { // Hurricane
					knockChance := 50 - p.Weight/10 - p.Agility/5
					if knockChance < 5 {
						knockChance = 5
					}
					if knockChance > 80 {
						knockChance = 80
					}
					if rand.Intn(100) < knockChance {
						p.Position = 2 // Laying
						changed = true
						if e.sendToPlayer != nil {
							e.sendToPlayer(p.FirstName, []string{"The hurricane-force winds knock you off your feet!"})
						}
						if e.roomBroadcast != nil {
							e.roomBroadcast(p.RoomNumber, []string{fmt.Sprintf("%s is knocked to the ground by the howling wind!", p.FirstName)})
						}
					}
				}
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
