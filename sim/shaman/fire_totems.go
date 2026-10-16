package shaman

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/buffs"
)

// The totem lays down a pulse spell of its own; the damage and coefficient live on that spell, not on
// the totem, and the pulse's cast time is the interval between pulses.
var searingTotemRank = spellData.SearingTotem.Highest()
var searingTotemAttack = spellData.SearingTotemTriggered.Highest()
var magmaTotemRank = spellData.MagmaTotem.Highest()
var magmaTotemPulse = spellData.MagmaTotemTriggered.ByID(10581)

// Flametongue Totem has no pulse spell: its party aura (15036, procs on a landed melee auto) triggers
// 16389, whose 1363 is the imbue's shape, hundredths of damage per second of main-hand speed.
var flametongueTotemRank = spellData.FlametongueTotem.Highest()
var flametongueTotemProc = spellData.FlametongueTotemTriggered.ByID(16389)

// Forever has no Fire Nova Totem: the totem's ids are gone and the Fire Nova the spellbook teaches in
// its place is the caster-centred nova, on a 10 sec cooldown.
var fireNovaRank = spellData.FireNova.Highest()

// The nova's damage is 408428 (403 base, 0.214 coefficient), which the scripted dummy casts, not the Era
// row 11307 its tooltip cites: beta logs record every Fire Nova hit under the rank 1 sibling 408423.
var fireNovaDamage = spellData.FireNovaTriggered.ByID(408428)

func (shaman *Shaman) registerSearingTotemSpell() {
	attack := shaman.RegisterSpell(core.SpellConfig{
		ActionID:         core.ActionID{SpellID: searingTotemAttack.ID},
		SpellSchool:      searingTotemAttack.SpellSchool(),
		DefenseType:      searingTotemAttack.DefenseTypeCore(),
		ProcMask:         core.ProcMaskEmpty,
		Flags:            SpellFlagShamanSpell | core.SpellFlagPassiveSpell,
		ClassSpellMask:   SpellMaskSearingTotem,
		MissileSpeed:     float64(searingTotemAttack.Speed),
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: searingTotemAttack.DamageEffect().Coeff(),
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcDamage(sim, target, searingTotemAttack.DamageEffect().Average(core.CharacterLevel), spell.OutcomeMagicHitAndCrit)
			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	})

	// The totem hard casts the pulse (2.2 sec), then waits about 0.23 sec before starting the next one:
	// beta logs time 1,267 attacks 2.435 sec apart (median, ranks 1-2, foreverlogs.gg reports to 2717).
	tickLength := searingTotemAttack.CastTime() + 230*time.Millisecond
	duration := searingTotemRank.Duration()

	shaman.SearingTotem = shaman.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: searingTotemRank.ID},
		SpellSchool:    searingTotemRank.SpellSchool(),
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagAPL | SpellFlagShamanSpell | SpellFlagInstant,
		ClassSpellMask: SpellMaskSearingTotem,
		ManaCost: core.ManaCostOptions{
			FlatCost: int32(searingTotemRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: searingTotemRank.GCD(),
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Searing Totem",
			},
			NumberOfTicks: int32(duration / tickLength),
			TickLength:    tickLength,
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				attack.Cast(sim, target)
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			shaman.cancelFireTotems(sim)
			spell.Dot(sim.Encounter.ActiveTargetUnits[0]).Apply(sim)
			shaman.TotemExpirations[FireTotem] = sim.CurrentTime + duration
		},
	})
}

// Neither dummy deals damage itself; the totem's hit lands as Flametongue Attack 16368 (beta log 2713,
// foreverlogs.gg), whose client row has no spell power coefficient and a class mask (bit 25) that
// Elemental Fury and Elemental Weapons don't name. Ported from MythicSim patch 70 (sage3648).
func (shaman *Shaman) registerFlametongueTotemSpell() {
	duration := flametongueTotemRank.Duration()
	hit := shaman.newFlametongueAttackSpell(flametongueTotemProc, shaman.MainHand, SpellMaskNone, 0)
	// "Each main hand hit": the aura's row hears melee autos only, so specials and off-hand swings add nothing.
	trigger := shaman.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Flametongue Totem Trigger",
		Duration:           core.NeverExpires,
		ProcMask:           core.ProcMaskMeleeMHAuto,
		Outcome:            core.OutcomeLanded,
		Callback:           core.CallbackOnSpellHitDealt,
		TriggerImmediately: true,
		Handler: func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
			hit.Cast(sim, result.Target)
		},
	})

	config := shaman.newTotemSpellConfig(int32(flametongueTotemRank.Cost()), flametongueTotemRank.ID, SpellMaskFlametongueTotem, flametongueTotemRank.GCD())
	config.SpellSchool = flametongueTotemRank.SpellSchool()
	totemAura := shaman.RegisterAura(core.Aura{
		Label:    "Flametongue Totem (Self)",
		ActionID: config.ActionID,
		Duration: duration,
	})
	totemAura.NewExclusiveEffect(buffs.FlametongueTotemCategory, false, core.ExclusiveEffect{
		Priority: buffs.FlametongueTotemCast,
		OnGain: func(_ *core.ExclusiveEffect, sim *core.Simulation) {
			trigger.Activate(sim)
		},
		OnExpire: func(_ *core.ExclusiveEffect, sim *core.Simulation) {
			trigger.Deactivate(sim)
		},
	})

	config.RelatedSelfBuff = totemAura
	config.ApplyEffects = func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
		shaman.cancelFireTotems(sim)
		shaman.TotemExpirations[FireTotem] = sim.CurrentTime + duration
		totemAura.Activate(sim)
	}
	shaman.FlametongueTotem = shaman.RegisterSpell(config)
}

// Magma Totem pulses every 2 sec for its life.
const magmaTotemTickLength = 2 * time.Second

func (shaman *Shaman) registerMagmaTotemSpell() {
	duration := magmaTotemRank.Duration()
	tickLength := magmaTotemTickLength

	shaman.MagmaTotem = shaman.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: magmaTotemRank.ID},
		SpellSchool:    magmaTotemRank.SpellSchool(),
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagAPL | SpellFlagShamanSpell | SpellFlagInstant,
		ClassSpellMask: SpellMaskMagmaTotem,
		ManaCost: core.ManaCostOptions{
			FlatCost: int32(magmaTotemRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: magmaTotemRank.GCD(),
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			IsAOE: true,
			Aura: core.Aura{
				Label: "Magma Totem",
			},
			NumberOfTicks:    int32(duration / tickLength),
			TickLength:       tickLength,
			BonusCoefficient: magmaTotemPulse.DamageEffect().Coeff(),

			OnTick: func(sim *core.Simulation, _ *core.Unit, dot *core.Dot) {
				dot.Spell.CalcPeriodicAoeDamage(sim, magmaTotemPulse.DamageEffect().Average(core.CharacterLevel), dot.Spell.OutcomeTickMagicHitAndCrit)
				dot.Spell.DealBatchedPeriodicDamage(sim)
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			shaman.cancelFireTotems(sim)
			spell.AOEDot().Apply(sim)
			// The dot's duration, which carries Trial by Fire's extra pulses.
			shaman.TotemExpirations[FireTotem] = sim.CurrentTime + spell.AOEDot().BaseDuration()
		},
	})
}

func (shaman *Shaman) registerFireNovaSpell() {
	shaman.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: fireNovaRank.ID},
		SpellSchool:    fireNovaRank.SpellSchool(),
		DefenseType:    fireNovaRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL | SpellFlagShamanSpell | SpellFlagInstant,
		ClassSpellMask: SpellMaskFireNova,
		ManaCost: core.ManaCostOptions{
			FlatCost: int32(fireNovaRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: fireNovaRank.GCD(),
			},
			CD: core.Cooldown{
				Timer:    shaman.NewTimer(),
				Duration: max(fireNovaRank.Cooldown(), fireNovaRank.CategoryCooldown()),
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: fireNovaDamage.DamageEffect().Coeff(),

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			spell.CalcAoeDamage(sim, fireNovaDamage.DamageEffect().Average(core.CharacterLevel), spell.OutcomeMagicHitAndCrit)
			spell.DealBatchedAoeDamage(sim)
		},
	})
}

func (shaman *Shaman) cancelFireTotems(sim *core.Simulation) {
	shaman.MagmaTotem.AOEDot().Deactivate(sim)
	if searingTotemDot := shaman.SearingTotem.Dot(shaman.CurrentTarget); searingTotemDot != nil {
		searingTotemDot.Deactivate(sim)
	}
	if shaman.TotemOfWrath != nil {
		shaman.TotemOfWrath.RelatedSelfBuff.Deactivate(sim)
	}
	if shaman.FlametongueTotem != nil {
		shaman.FlametongueTotem.RelatedSelfBuff.Deactivate(sim)
	}
}
