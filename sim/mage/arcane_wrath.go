package mage

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// Arcane Wrath (1326079-82), new in build 70425: an Arcane r3c3 talent that teaches a 3.5 sec, 6 sec
// cooldown bolt. Its second effect "gains additional Spell Damage equal to 5% of your maximum Mana",
// which the sim adds to the hit at the damage effect's 1.0 coefficient. Every rank is registered, as
// the rotation may drop to a cheaper one.
func (mage *Mage) registerArcaneWrathSpell() {
	if !mage.Talents.ArcaneWrath {
		return
	}

	cooldown := mage.NewTimer()
	spellData.ArcaneWrath.Each(func(_ int32, rank *spelldata.Spell) { mage.registerArcaneWrathRank(rank, cooldown) })
}

// The rolled hit plus 5% of maximum mana as spell damage, at the damage effect's coefficient.
func (mage *Mage) arcaneWrathBaseDamage(sim *core.Simulation, rank *spelldata.Spell) float64 {
	damage := rank.DamageEffect()
	return damage.Roll(sim, core.CharacterLevel) + damage.Coeff()*rank.EffectN(2).Percent()*mage.MaxMana()
}

func (mage *Mage) registerArcaneWrathRank(rank *spelldata.Spell, cooldown *core.Timer) {
	damage := rank.DamageEffect()

	mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.ID},
		SpellSchool:    rank.SpellSchool(),
		DefenseType:    rank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL | core.SpellFlagBinary,
		ClassSpellMask: MageSpellArcaneWrath,
		MaxRange:       float64(rank.MaxRange),
		Rank:           rank.RankNumber(),
		MissileSpeed:   float64(rank.Speed),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(rank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      rank.GCD(),
				CastTime: rank.CastTime(),
			},
			CD: core.Cooldown{
				Timer:    cooldown,
				Duration: rank.Cooldown(),
			},
		},

		DamageMultiplier: 1,
		BonusCoefficient: damage.Coeff(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcDamage(sim, target, mage.arcaneWrathBaseDamage(sim, rank), spell.OutcomeMagicHitAndCrit)
			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	})
}
