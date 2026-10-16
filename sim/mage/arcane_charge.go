package mage

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/stats"
)

// Forever's Arcane Blast buff (400573), as reworked in build 70425: each stack gives 15% of the mage's
// Intellect as spell damage (aura 174, all magic schools) and raises Arcane Blast's own cost. Nothing
// spends the stacks any more, and they last 15 sec "from time of initial application": a new stack
// does not refresh the timer.
func (mage *Mage) registerArcaneCharges() {
	if !mage.Talents.ArcaneBlast {
		return
	}

	buffRank := spellData.ArcaneBlastTriggered.Highest()
	intPerStack := buffRank.Effect(dbcenums.A_MOD_SPELL_DAMAGE_OF_STAT_PERCENT, 126).Percent()
	costPerStack := buffRank.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_COST)).Average(core.CharacterLevel) / 100

	spellDamageDep := mage.NewDynamicStatDependency(stats.Intellect, stats.SpellDamage, 0)
	costMod := mage.AddDynamicMod(core.SpellModConfig{
		ClassMask: MageSpellArcaneBlast,
		Kind:      core.SpellMod_PowerCost_Pct_Add,
	})

	mage.ArcaneBlastAura = mage.RegisterAura(core.Aura{
		Label:     "Arcane Blast",
		ActionID:  core.ActionID{SpellID: buffRank.ID},
		Duration:  buffRank.Duration(),
		MaxStacks: int32(buffRank.MaxStack),
		OnGain: func(_ *core.Aura, sim *core.Simulation) {
			mage.EnableDynamicStatDep(sim, spellDamageDep)
			costMod.Activate()
		},
		OnExpire: func(_ *core.Aura, sim *core.Simulation) {
			mage.DisableDynamicStatDep(sim, spellDamageDep)
			costMod.Deactivate()
		},
		OnStacksChange: func(_ *core.Aura, sim *core.Simulation, _ int32, newStacks int32) {
			mage.UpdateDynamicStatDep(sim, spellDamageDep, intPerStack*float64(newStacks))
			costMod.UpdateFloatValue(costPerStack * float64(newStacks))
		},
	})
}
