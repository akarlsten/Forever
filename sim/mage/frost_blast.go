package mage

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// Frost Blast (1326053-55), trained from build 70425: a 1.5 sec bolt with a 40% slow for 9 sec. The
// row carries Frostbolt's family flags, chill bit included, so Fingers of Frost rolls on it. The slow
// does nothing against a boss and is left out.
func (mage *Mage) registerFrostBlastSpell() {
	spellData.FrostBlast.Each(func(_ int32, rank *spelldata.Spell) { mage.registerFrostBlastRank(rank) })
}

func (mage *Mage) registerFrostBlastRank(rank *spelldata.Spell) {
	damage := rank.DamageEffect()

	mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.ID},
		SpellSchool:    rank.SpellSchool(),
		DefenseType:    rank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL | core.SpellFlagBinary,
		ClassSpellMask: MageSpellFrostBlast,
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
		},

		DamageMultiplier: 1,
		BonusCoefficient: damage.Coeff(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcDamage(sim, target, damage.Roll(sim, core.CharacterLevel), spell.OutcomeMagicHitAndCrit)
			if mage.procFingersOfFrost != nil {
				mage.procFingersOfFrost(sim, spell, result)
			}
			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	})
}
