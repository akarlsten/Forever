package priest

import (
	"fmt"
	"math"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/core/stats"
)

func (priest *Priest) registerShadowTalents() {
	// Tier 1
	priest.applyShadowFocus()
	priest.applyBlackout()
	priest.applySpiritTap()

	// Tier 2
	priest.applyImprovedShadowWordPain()
	priest.applyShadowReach()

	// Tier 3
	priest.applyImprovedMindBlast()
	priest.applyImprovedPsychicScream()
	priest.applyMindFlay()

	priest.applyDarkInspiration()

	// Tier 4
	priest.applyVampiricEmbrace()
	priest.applyShadowWeaving()

	// Tier 5
	priest.applySilence()
	priest.applyDevouringContagion()

	// Tier 6
	priest.applyEarlyDemise()
	priest.applyDarkness()

	// Tier 7
	priest.applyShadowform()
}

// A school-specific hit bonus has to go through the pseudo-stat; a SpellMod carrying a school panics.
func (priest *Priest) applyShadowFocus() {
	if priest.Talents.ShadowFocus == 0 {
		return
	}

	priest.PseudoStats.SchoolBonusHitChance[stats.SchoolIndexShadow] +=
		spellData.ShadowFocus.Effect(dbcenums.A_ADD_FLAT_MODIFIER, int32(dbcenums.SPELLMOD_RESIST_MISS_CHANCE)).ValueAt(priest.Talents.ShadowFocus)
}

// applyBlackout implements Blackout.
//
// TODO: To be implemented. It stuns (15269), and a raid boss is immune.
func (priest *Priest) applyBlackout() {
	if priest.Talents.Blackout == 0 {
		return
	}
}

// applySpiritTap implements Spirit Tap.
//
// TODO: To be implemented. The buff (15271) is granted by a killing blow, which no encounter in the
// sim delivers.
func (priest *Priest) applySpiritTap() {
	if priest.Talents.SpiritTap == 0 {
		return
	}
}

// +3 sec of duration per point, which is one more tick at Shadow Word: Pain's 3 second cadence.
func (priest *Priest) applyImprovedShadowWordPain() {
	if priest.Talents.ImprovedShadowWordPain == 0 {
		return
	}

	added := time.Millisecond * time.Duration(spellData.ImprovedShadowWordPain.ValueAt(priest.Talents.ImprovedShadowWordPain))
	tickLength := spellData.ShadowWordPain.Highest().PeriodicEffect().Period()

	priest.AddStaticMod(core.SpellModConfig{
		ClassMask: PriestSpellShadowWordPain,
		IntValue:  int32(added / tickLength),
		Kind:      core.SpellMod_DotNumberOfTicks_Flat,
	})
}

// applyShadowReach implements Shadow Reach: 17322 adds 10/20% to the range of the spells its class
// mask reaches, of the priest's the five targeted Shadow spells.
func (priest *Priest) applyShadowReach() {
	if priest.Talents.ShadowReach == 0 {
		return
	}

	priest.extendRange(PriestSpellDevouringPlague|PriestSpellMindBlast|PriestSpellMindFlay|PriestSpellShadowWordDeath|PriestSpellShadowWordPain,
		spellData.ShadowReach.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_RANGE)).FractionAt(priest.Talents.ShadowReach))
}

// extendRange is a percent range modifier: the sim's range mod is flat yards, so the share is taken of
// each spell's own range as it registers. A spell with no range (0) keeps none.
func (priest *Priest) extendRange(mask int64, share float64) {
	priest.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Matches(mask) {
			spell.MaxRange *= 1 + share
		}
	})
}

func (priest *Priest) applyImprovedMindBlast() {
	if priest.Talents.ImprovedMindBlast == 0 {
		return
	}

	priest.AddStaticMod(core.SpellModConfig{
		ClassMask: PriestSpellMindBlast,
		TimeValue: time.Millisecond * time.Duration(spellData.ImprovedMindBlast.ValueAt(priest.Talents.ImprovedMindBlast)),
		Kind:      core.SpellMod_Cooldown_Flat,
	})
}

// applyImprovedPsychicScream implements Improved Psychic Scream.
//
// TODO: To be implemented. Psychic Scream fears, and a raid boss is immune.
func (priest *Priest) applyImprovedPsychicScream() {
	if priest.Talents.ImprovedPsychicScream == 0 {
		return
	}
}

func (priest *Priest) applyMindFlay() {
	if !priest.Talents.MindFlay {
		return
	}

	MindFlayRankMap.Each(func(_ int32, rank *spelldata.Spell) { priest.registerMindFlaySpell(rank) })
}

var MindFlayRankMap = spellData.MindFlay

// A three tick channel. Forever's ticks are not hastened, as in Classic and TBC.
func (priest *Priest) registerMindFlaySpell(rank *spelldata.Spell) {
	tick := rank.PeriodicEffect()
	tickLength := tick.Period()
	numTicks := int32(rank.Duration() / tickLength)
	improvedStep := priest.improvedMindFlayStep()
	var ticksDealt int

	priest.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: rank.ID},
		SpellSchool: rank.SpellSchool(),
		DefenseType: rank.DefenseTypeCore(),
		ProcMask:    core.ProcMaskSpellDamage,
		// Binary, as on master: the row slows (effect 1), so it resists whole or not at all.
		Flags:          core.SpellFlagAPL | core.SpellFlagChanneled | core.SpellFlagBinary,
		ClassSpellMask: PriestSpellMindFlay,
		Rank:           rank.RankNumber(),
		MaxRange:       float64(rank.MaxRange),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(rank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: rank.GCD(),
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: fmt.Sprintf("MindFlay-%d", rank.RankNumber()),
			},
			NumberOfTicks:       numTicks,
			TickLength:          tickLength,
			AffectedByCastSpeed: false,
			BonusCoefficient:    tick.Coeff(),

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.Snapshot(target, tick.Average(core.CharacterLevel))
				ticksDealt = 0
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				growth := math.Pow(1+improvedStep, float64(ticksDealt))
				ticksDealt++
				// A tick reads the current multipliers (PeriodicDamageMultiplier) or, for a dot on stored
				// numbers, the snapshot; the growth goes on both so either path takes it once.
				dot.SnapshotAttackerMultiplier *= growth
				dot.PeriodicDamageMultiplier *= growth
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, priestTickOutcome(rank.PeriodicCanCrit(), dot))
				dot.SnapshotAttackerMultiplier /= growth
				dot.PeriodicDamageMultiplier /= growth
				if int32(ticksDealt) == numTicks && priest.procDarkInspiration != nil {
					priest.procDarkInspiration(sim)
				}
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHitNoHitCounter)
			if result.Landed() {
				spell.Dot(target).Apply(sim)
			}
			spell.DealOutcome(sim, result)
		},

		ExpectedTickDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, useSnapshot bool) *core.SpellResult {
			if useSnapshot {
				return spell.Dot(target).CalcSnapshotDamage(sim, target, spell.OutcomeExpectedMagicHit)
			}
			return spell.CalcPeriodicDamage(sim, target, tick.Average(core.CharacterLevel), spell.OutcomeExpectedMagicHit)
		},
	})
}

// Improved Mind Flay (1225139, reworked in build 70425): "Each instance of damage of Mind Flay deals
// 10/20% more damage than its previous instance of damage." Read as compounding, tick n at
// (1+x)^(n-1); additive steps are the other reading. Unverified. Applied in the Mind Flay tick.
func (priest *Priest) improvedMindFlayStep() float64 {
	if priest.Talents.ImprovedMindFlay == 0 {
		return 0
	}
	return spellData.ImprovedMindFlay.EffectAt(1).FractionAt(priest.Talents.ImprovedMindFlay)
}

// The Forever client heals the priest's whole party for a share of the Shadow damage it deals, where
// Classic healed only the priest. Only the priest who applied the debuff heals from it.
func (priest *Priest) applyVampiricEmbrace() {
	if !priest.Talents.VampiricEmbrace {
		return
	}

	rank := spellData.VampiricEmbrace.Highest()
	healPct := rank.Effect(dbcenums.A_DUMMY, 0).Average(core.CharacterLevel) / 100
	healthMetrics := priest.NewHealthMetrics(core.ActionID{SpellID: rank.ID})
	partyPlayers := priest.Party.Players

	veAuras := priest.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		aura := target.GetOrRegisterAura(core.Aura{
			Label:    "Vampiric Embrace - " + target.Label,
			ActionID: core.ActionID{SpellID: rank.ID},
			Duration: rank.Duration(),
		})
		aura.AttachProcTriggerCallback(target, core.ProcTrigger{
			Name:               "Vampiric Embrace Proc",
			Callback:           core.CallbackOnSpellHitTaken | core.CallbackOnPeriodicDamageTaken,
			RequireDamageDealt: true,
			ExtraCondition: func(_ *core.Simulation, spell *core.Spell, _ *core.SpellResult) bool {
				return spell.Unit == &priest.Unit && spell.SpellSchool.Matches(core.SpellSchoolShadow)
			},
			Handler: func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
				for _, player := range partyPlayers {
					player.GetCharacter().GainHealth(sim, result.Damage*healPct, healthMetrics)
				}
			},
		})
		return aura
	})

	priest.VampiricEmbrace = priest.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.ID},
		SpellSchool:    core.SpellSchoolShadow,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagAPL,
		ClassSpellMask: PriestSpellVampiricEmbrace,

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(rank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    priest.NewTimer(),
				Duration: max(rank.Cooldown(), rank.CategoryCooldown()),
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHit)
			if result.Landed() {
				veAuras.Get(target).Activate(sim)
			}
			spell.DealOutcome(sim, result)
		},

		RelatedAuraArrays: veAuras.ToMap(),
	})
}

// Shadow Weaving's stacks (15258) are a debuff on the target, as in Classic, but they raise only the
// priest's own Shadow damage on that target (A_MOD_SCHOOL_MASK_DAMAGE_FROM_CASTER), 2% a stack to five.
// Each target keeps its own stacks: beta log 2726 (Serana) dots four Razorfen mobs and each carries
// one stack of its own, dropping off on its own 15 sec.
func (priest *Priest) applyShadowWeaving() {
	if priest.Talents.ShadowWeaving == 0 {
		return
	}

	stackAura := spellData.ShadowWeavingTriggered.Highest()
	perStack := stackAura.Effect(dbcenums.A_MOD_SCHOOL_MASK_DAMAGE_FROM_CASTER, 32).Average(core.CharacterLevel) / 100

	priest.ShadowWeavingAuras = priest.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return target.RegisterAura(core.Aura{
			Label:     "Shadow Weaving-" + priest.Label,
			ActionID:  core.ActionID{SpellID: stackAura.ID},
			Duration:  stackAura.Duration(),
			MaxStacks: 5,
		})
	})

	for _, target := range priest.Env.Encounter.AllTargetUnits {
		target.AddDynamicDamageTakenModifier(func(_ *core.Simulation, spell *core.Spell, result *core.SpellResult, _ bool) {
			if spell.Unit != &priest.Unit || !spell.SpellSchool.Matches(core.SpellSchoolShadow) {
				return
			}
			if aura := priest.ShadowWeavingAuras.Get(result.Target); aura.IsActive() {
				result.Damage *= 1 + perStack*float64(aura.GetStacks())
			}
		})
	}

	priest.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Shadow Weaving Trigger",
		Callback:           core.CallbackOnSpellHitDealt,
		ClassSpellMask:     PriestShadowSpells,
		Outcome:            core.OutcomeLanded,
		ProcChance:         spellData.ShadowWeaving.FractionAt(priest.Talents.ShadowWeaving),
		TriggerImmediately: true,
		Handler: func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
			aura := priest.ShadowWeavingAuras.Get(result.Target)
			aura.Activate(sim)
			aura.AddStack(sim)
		},
	})
}

// applySilence implements Silence, new in Forever.
//
// TODO: To be implemented. It interrupts a cast, which no encounter in the sim makes.
func (priest *Priest) applySilence() {
	if !priest.Talents.Silence {
		return
	}
}

// Devouring Contagion is new in Forever: Devouring Plague costs 25% less per point. The client's
// second effect is a dummy worth 5/10 that no tooltip names, so it is left out.
func (priest *Priest) applyDevouringContagion() {
	if priest.Talents.DevouringContagion == 0 {
		return
	}

	priest.AddStaticMod(core.SpellModConfig{
		ClassMask:  PriestSpellDevouringPlague,
		FloatValue: spellData.DevouringContagion.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_COST)).FractionAt(priest.Talents.DevouringContagion),
		Kind:       core.SpellMod_PowerCost_Pct_Add,
	})
}

// applyEarlyDemise implements Early Demise, new in Forever.
//
// The crit it grants Shadow Word: Death below 20% health is read in shadow_word_death.go, where the
// execute check belongs.
func (priest *Priest) applyEarlyDemise() {
}

// Classic's Darkness raised the damage of five named Shadow spells. The Forever client's (15259)
// raises all Shadow damage done, 2% per point.
func (priest *Priest) applyDarkness() {
	if priest.Talents.Darkness == 0 {
		return
	}

	priest.AddStaticMod(core.SpellModConfig{
		ClassMask:  PriestSpellsAll,
		School:     core.SpellSchoolShadow,
		FloatValue: spellData.Darkness.FractionAt(priest.Talents.Darkness),
		Kind:       core.SpellMod_DamageDone_Pct,
	})
}

// The beta client's 15473: +10% Shadow damage, -50% Shadow mana cost, +100% Shadow critical strike
// damage bonus, -15% Physical damage taken. Only healing is blocked, so Smite and Holy Fire stay
// castable inside it and do not break it. The crit bonus is not school-wide: its mask (41984016)
// names Mind Blast, Mind Flay, Shadow Word: Pain, Devouring Plague and Mana Burn. Shadow Word: Death
// is added: the devs called its absence a bug, fixed for the next build (forever-bugs #328).
func (priest *Priest) applyShadowform() {
	if !priest.Talents.Shadowform {
		return
	}

	rank := spellData.Shadowform.Highest()

	priest.ShadowformAura = priest.RegisterAura(core.Aura{
		Label:    "Shadowform",
		ActionID: core.ActionID{SpellID: rank.ID},
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			if priest.SelfBuffs.PreShadowform {
				aura.Activate(sim)
			}
		},
		// The client's form 28, which Holy Nova's and Chastise's rows exclude.
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			priest.ShapeshiftForm = rank.ShapeshiftForm()
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			priest.ShapeshiftForm = 0
		},
	}).AttachSpellMod(core.SpellModConfig{
		ClassMask:  PriestSpellsAll,
		School:     core.SpellSchoolShadow,
		FloatValue: rank.Effect(dbcenums.A_MOD_DAMAGE_PERCENT_DONE, 32).Average(core.CharacterLevel) / 100,
		Kind:       core.SpellMod_DamageDone_Pct,
	}).AttachSpellMod(core.SpellModConfig{
		ClassMask:  PriestSpellsAll,
		School:     core.SpellSchoolShadow,
		FloatValue: rank.Effect(dbcenums.A_MOD_POWER_COST_SCHOOL_PCT, 32).Average(core.CharacterLevel) / 100,
		Kind:       core.SpellMod_PowerCost_Pct,
	}).AttachSpellMod(core.SpellModConfig{
		ClassMask:  PriestSpellMindBlast | PriestSpellMindFlay | PriestSpellShadowWordPain | PriestSpellDevouringPlague | PriestSpellShadowWordDeath,
		FloatValue: rank.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_CRIT_DAMAGE_BONUS)).Average(core.CharacterLevel) / 100,
		Kind:       core.SpellMod_CritMultiplier_Flat,
	}).AttachMultiplicativePseudoStatBuff(
		&priest.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexPhysical],
		1+rank.Effect(dbcenums.A_MOD_DAMAGE_PERCENT_TAKEN, 1).Average(core.CharacterLevel)/100,
	)

	priest.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.ID},
		SpellSchool:    core.SpellSchoolShadow,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagAPL,
		ClassSpellMask: PriestSpellShadowform,

		ManaCost: rank.ManaCost(),
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			priest.ShadowformAura.Activate(sim)
		},

		RelatedSelfBuff: priest.ShadowformAura,
	})
}

// Dark Inspiration (1326159, new in build 70425): Mind Blast and a fully channeled Mind Flay raise the
// Spirit of the priest and the priest's party by 3/6/9/12/15% for 10 sec (1326168, aura 80 on Spirit).
func (priest *Priest) applyDarkInspiration() {
	if priest.Talents.DarkInspiration == 0 {
		return
	}

	buff := spellData.DarkInspirationTriggered.Highest()
	multiplier := spellData.DarkInspiration.MultiplierAt(priest.Talents.DarkInspiration)

	var auras []*core.StatBuffAura
	for _, player := range priest.Party.Players {
		auras = append(auras, player.GetCharacter().NewTemporaryStatMultiplierAura(core.Aura{
			Label:    "Dark Inspiration - " + priest.Label,
			ActionID: core.ActionID{SpellID: buff.ID},
			Duration: buff.Duration(),
		}, []core.StatMultiplier{{Stat: stats.Spirit, Multiplier: multiplier}}))
	}

	priest.procDarkInspiration = func(sim *core.Simulation) {
		for _, aura := range auras {
			aura.Activate(sim)
		}
	}

	priest.MakeProcTriggerAura(core.ProcTrigger{
		Name:           "Dark Inspiration Trigger",
		Callback:       core.CallbackOnCastComplete,
		ClassSpellMask: PriestSpellMindBlast,
		Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			priest.procDarkInspiration(sim)
		},
	})
}
