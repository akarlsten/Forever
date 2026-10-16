package warlock

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/core/stats"
)

func (warlock *Warlock) registerDestructionTalents() {
	// Tier 1
	warlock.applyDestructiveReach()
	warlock.applyImprovedShadowBolt()
	warlock.applyBane()

	// Tier 2
	warlock.applyMoltenSkin()
	warlock.applyCataclysm()
	warlock.applyAftermath()

	// Tier 3
	warlock.applyRuin()
	warlock.applyShadowburn()

	// Tier 4
	warlock.applyIntensity()
	warlock.applyAgonizingFlames()
	warlock.applyConflagrate()

	// Tier 5
	warlock.applyPyroclasm()
	warlock.applyBaneOfHavoc()
	warlock.applyFireAndBrimstone()

	// Tier 6
	warlock.applyShadowAndFlame()

	// Tier 7
	// Incinerate: incinerate.go
}

// A Shadow Bolt crit leaves a debuff that only the warlock's own shadow damage benefits from -
// 17794 applies A_MOD_SCHOOL_MASK_DAMAGE_FROM_CASTER - and it has no charges, so it is not eaten by
// the first hit that lands. 4% a point.
func (warlock *Warlock) applyImprovedShadowBolt() {
	if warlock.Talents.ImprovedShadowBolt == 0 {
		return
	}

	triggered := spellData.ImprovedShadowBoltTriggered.Highest()
	multiplier := 1 + spellData.ImprovedShadowBolt.FractionAt(warlock.Talents.ImprovedShadowBolt)

	warlock.ImprovedShadowBoltAuras = warlock.NewEnemyAuraArray(func(unit *core.Unit) *core.Aura {
		return unit.RegisterAura(core.Aura{
			Label:    "Improved Shadow Bolt-" + warlock.Label,
			ActionID: core.ActionID{SpellID: triggered.ID},
			Duration: triggered.Duration(),
		})
	})

	for _, target := range warlock.Env.Encounter.AllTargetUnits {
		target.AddDynamicDamageTakenModifier(func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult, isPeriodic bool) {
			if spell.Unit == &warlock.Unit && spell.SpellSchool.Matches(core.SpellSchoolShadow) && warlock.ImprovedShadowBoltAuras.Get(result.Target).IsActive() {
				result.Damage *= multiplier
			}
		})
	}

	warlock.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Improved Shadow Bolt Trigger",
		Callback:           core.CallbackOnSpellHitDealt,
		ClassSpellMask:     WarlockSpellShadowBolt,
		Outcome:            core.OutcomeCrit,
		TriggerImmediately: true,
		Handler: func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
			warlock.ImprovedShadowBoltAuras.Get(result.Target).Activate(sim)
		},
	})
}

// 3/6/10% cheaper Destruction spells in Forever, not 3% a point (17778).
func (warlock *Warlock) applyCataclysm() {
	if warlock.Talents.Cataclysm == 0 {
		return
	}

	warlock.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_PowerCost_Pct_Add,
		FloatValue: spellData.Cataclysm.FractionAt(warlock.Talents.Cataclysm),
		ClassMask:  WarlockDestructionSpells,
	})
}

// 0.1 sec off Shadow Bolt, Immolate and Incinerate and 0.4 sec off Soul Fire, a point (17788).
func (warlock *Warlock) applyBane() {
	if warlock.Talents.Bane == 0 {
		return
	}

	points := warlock.Talents.Bane

	warlock.AddStaticMod(core.SpellModConfig{
		Kind:      core.SpellMod_CastTime_Flat,
		TimeValue: time.Millisecond * time.Duration(spellData.Bane.EffectAt(1).ValueAt(points)),
		ClassMask: WarlockSpellShadowBolt | WarlockSpellImmolate | WarlockSpellIncinerate,
	})

	warlock.AddStaticMod(core.SpellModConfig{
		Kind:      core.SpellMod_CastTime_Flat,
		TimeValue: time.Millisecond * time.Duration(spellData.Bane.EffectAt(2).ValueAt(points)),
		ClassMask: WarlockSpellSoulFire,
	})
}

func (warlock *Warlock) applyShadowburn() {
	if !warlock.Talents.Shadowburn {
		return
	}

	warlock.registerShadowBurn()
}

// Destructive Reach: 17917 adds 10/20% to the range of the warlock's Destruction spells. Build 70425
// narrowed its mask to them (Shadow Bolt, Immolate, Incinerate, Soul Fire, Shadowburn, Conflagrate,
// Searing Pain, Rain of Fire, Hellfire), where it had reached Affliction too; Grim Reach now covers
// those. The sim's range mod is flat yards, so the share is taken of each spell's own range as it
// registers. A spell with no range (0) keeps none.
func (warlock *Warlock) applyDestructiveReach() {
	if warlock.Talents.DestructiveReach == 0 {
		return
	}

	share := spellData.DestructiveReach.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_RANGE)).FractionAt(warlock.Talents.DestructiveReach)
	warlock.extendRange(WarlockSpellShadowBolt|WarlockSpellImmolate|WarlockSpellIncinerate|WarlockSpellSoulFire|
		WarlockSpellShadowBurn|WarlockSpellConflagrate|WarlockSpellSearingPain|WarlockSpellRainOfFire|WarlockSpellHellfire, share)
}

// Raises the range of the spells in a mask by a share of their own range as each one registers.
func (warlock *Warlock) extendRange(mask int64, share float64) {
	warlock.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Matches(mask) {
			spell.MaxRange *= 1 + share
		}
	})
}

// Forever grows Ruin from one rank to five: 20% more critical damage a point (17959).
func (warlock *Warlock) applyRuin() {
	if warlock.Talents.Ruin == 0 {
		return
	}

	warlock.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_CritMultiplier_Flat,
		FloatValue: spellData.Ruin.FractionAt(warlock.Talents.Ruin),
		ClassMask:  WarlockDestructionSpells,
	})
}

func (warlock *Warlock) applyConflagrate() {
	if !warlock.Talents.Conflagrate {
		return
	}

	warlock.registerConflagrate()
}

// Conflagrate leaves 2% more Shadow damage a point and Shadowburn 2% more Fire, for 20 sec
// (426311 and 1293816). The chance Conflagrate spares Immolate is in conflagrate.go.
func (warlock *Warlock) applyShadowAndFlame() {
	if warlock.Talents.ShadowAndFlame == 0 {
		return
	}

	fireRow := spellData.ShadowAndFlameTriggered.ByID(426311)
	shadowRow := spellData.ShadowAndFlameTriggered.ByID(1293816)
	multiplier := 1 + spellData.ShadowAndFlame.EffectAt(3).FractionAt(warlock.Talents.ShadowAndFlame)

	shadowAura := warlock.RegisterAura(core.Aura{
		Label:    "Shadow and Flame (Shadow)",
		ActionID: core.ActionID{SpellID: shadowRow.ID},
		Duration: shadowRow.Duration(),
	}).AttachMultiplicativePseudoStatBuff(&warlock.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexShadow], multiplier)

	fireAura := warlock.RegisterAura(core.Aura{
		Label:    "Shadow and Flame (Fire)",
		ActionID: core.ActionID{SpellID: fireRow.ID},
		Duration: fireRow.Duration(),
	}).AttachMultiplicativePseudoStatBuff(&warlock.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexFire], multiplier)

	warlock.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Shadow and Flame Trigger",
		Callback:           core.CallbackOnSpellHitDealt,
		ClassSpellMask:     WarlockSpellConflagrate | WarlockSpellShadowBurn,
		Outcome:            core.OutcomeLanded,
		TriggerImmediately: true,
		Handler: func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
			if spell.Matches(WarlockSpellConflagrate) {
				shadowAura.Activate(sim)
			} else {
				fireAura.Activate(sim)
			}
		},
	})
}

// 2% less damage taken a point (1225220).
func (warlock *Warlock) applyMoltenSkin() {
	if warlock.Talents.MoltenSkin == 0 {
		return
	}

	warlock.PseudoStats.DamageTakenMultiplier *= spellData.MoltenSkin.MultiplierAt(warlock.Talents.MoltenSkin)
}

// 10% more Immolate impact damage a point (18119, second effect). Its daze half, a 50% slow the
// first effect rolls for, changes no damage.
func (warlock *Warlock) applyAftermath() {
	if warlock.Talents.Aftermath == 0 {
		return
	}

	warlock.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_DamageDone_Flat,
		FloatValue: spellData.Aftermath.EffectAt(2).FractionAt(warlock.Talents.Aftermath),
		ClassMask:  WarlockSpellImmolate,
	})
}

// 3/7/10% more Destruction damage, and the same again as Searing Pain crit (17927). The direct
// half (mask 421/8388800) leaves out Immolate's dot and Hellfire; the dot half (mask 36) is
// Immolate's dot alone of what the sim casts, so its ticks take the bonus once.
func (warlock *Warlock) applyAgonizingFlames() {
	if warlock.Talents.AgonizingFlames == 0 {
		return
	}

	points := warlock.Talents.AgonizingFlames

	warlock.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_DamageDone_Flat,
		FloatValue: spellData.AgonizingFlames.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_DAMAGE)).FractionAt(points),
		ClassMask:  WarlockDestructionSpells &^ (WarlockSpellImmolateDot | WarlockSpellHellfire),
	})
	warlock.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_DotDamageDone_Pct,
		FloatValue: spellData.AgonizingFlames.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_DOT)).FractionAt(points),
		ClassMask:  WarlockSpellImmolateDot,
	})
	warlock.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_BonusCrit_Percent,
		FloatValue: spellData.AgonizingFlames.Effect(dbcenums.A_ADD_FLAT_MODIFIER, int32(dbcenums.SPELLMOD_CRITICAL_CHANCE)).ValueAt(points),
		ClassMask:  WarlockSpellSearingPain,
	})
}

// 8/17/25% Conflagrate crit, the talent's second effect (412751).
func (warlock *Warlock) applyFireAndBrimstone() {
	if warlock.Talents.FireAndBrimstone == 0 {
		return
	}

	warlock.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_BonusCrit_Percent,
		FloatValue: spellData.FireAndBrimstone.EffectAt(2).ValueAt(warlock.Talents.FireAndBrimstone),
		ClassMask:  WarlockSpellConflagrate,
	})
}

// applyIntensity implements Intensity, new in Forever.
//
// Pushback protection (SPELLMOD_NOT_LOSE_CASTING_TIME, 23/47/70%) has nothing to act on in the sim.
func (warlock *Warlock) applyIntensity() {
	if warlock.Talents.Intensity == 0 {
		return
	}
}

// applyPyroclasm implements Pyroclasm, new in Forever.
//
// TODO: 18073 rolls 13/26% for 18093, a 3 sec stun; the sim has nothing to stun.
func (warlock *Warlock) applyPyroclasm() {
	if warlock.Talents.Pyroclasm == 0 {
		return
	}
}

// applyBaneOfHavoc implements Bane of Havoc, new in Forever.
//
// 1225228: a 5 min bane on one target (A_DUMMY 15) that copies 15% of the warlock's damage to other
// targets onto the baned one. It takes the bane slot, so it replaces Agony or Doom there. The copy is
// the share of the damage already dealt, so nothing on either side modifies it again. The row states
// no GCD category, so the cast is off the GCD. Only the warlock's own damage copies, not the demon's.
func (warlock *Warlock) applyBaneOfHavoc() {
	if !warlock.Talents.BaneOfHavoc {
		return
	}

	rank := spellData.BaneOfHavoc.Highest()
	share := rank.Effect(dbcenums.A_DUMMY, 0).Percent()

	var havocTarget *core.Unit
	havocAuras := warlock.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return target.RegisterAura(core.Aura{
			Label:    "Bane of Havoc-" + warlock.Label,
			ActionID: core.ActionID{SpellID: rank.ID},
			Duration: rank.Duration(),
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				havocTarget = aura.Unit
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				if havocTarget == aura.Unit {
					havocTarget = nil
				}
			},
		})
	})

	config := spelldata.SpellConfig(&warlock.Unit, rank, spelldata.Flags(core.SpellFlagAPL))
	config.ProcMask = core.ProcMaskEmpty
	config.ThreatMultiplier = 1
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHitNoHitCounter)
		if result.Landed() {
			if havocTarget != nil && havocTarget != target {
				havocAuras.Get(havocTarget).Deactivate(sim)
			}
			aura := havocAuras.Get(target)
			warlock.takeBaneSlot(sim, target, aura)
			aura.Activate(sim)
		}
		spell.DealOutcome(sim, result)
	}
	config.RelatedAuraArrays = havocAuras.ToMap()
	warlock.RegisterSpell(config)

	copySpell := warlock.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: rank.ID, Tag: 1},
		SpellSchool: rank.SpellSchool(),
		ProcMask:    core.ProcMaskEmpty,
		Flags: core.SpellFlagIgnoreModifiers | core.SpellFlagIgnoreResists | core.SpellFlagNoOnDamageDealt |
			core.SpellFlagPassiveSpell | core.SpellFlagNoOnCastComplete,

		DamageMultiplierAdditive: 1,
		DamageMultiplier:         1,
		ThreatMultiplier:         1,
	})

	copyDamage := func(sim *core.Simulation, result *core.SpellResult) {
		if havocTarget != nil && result.Target != havocTarget && result.Damage > 0 {
			copySpell.CalcAndDealDamage(sim, havocTarget, result.Damage*share, copySpell.OutcomeAlwaysHit)
		}
	}
	core.MakePermanent(warlock.RegisterAura(core.Aura{
		Label: "Bane of Havoc - Copy",
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			copyDamage(sim, result)
		},
		OnPeriodicDamageDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			copyDamage(sim, result)
		},
	}))
}
