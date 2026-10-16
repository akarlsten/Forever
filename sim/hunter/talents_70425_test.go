package hunter

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

func newTalentedHunter(t *testing.T, talents string) (*core.Simulation, *Hunter) {
	t.Helper()
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{Parties: []*proto.Party{{Buffs: &proto.PartyBuffs{}, Players: []*proto.Player{{
			Name: "hunter", Class: proto.Class_ClassHunter, Race: proto.Race_RaceOrc, TalentsString: talents,
			Equipment: WeaponsOnly, Buffs: &proto.IndividualBuffs{},
			Spec: &proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{ClassOptions: &proto.HunterOptions{
				Ammo: proto.HunterOptions_Doomshot, PetType: proto.HunterOptions_PetNone}}}},
			Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}}}}},
		Encounter: core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim, sim.Raid.Parties[0].Players[0].(HunterAgent).GetHunter()
}

// Build 70425 added Sniper Shot to Efficiency's (19416) and Mortal Shots' (19485) class masks.
func TestSniperShotTakesEfficiencyAndMortalShots(t *testing.T) {
	_, hunter := newTalentedHunter(t, MarksmanshipTalents)
	if hunter.Talents.Efficiency == 0 || hunter.Talents.MortalShots == 0 || hunter.SniperShot == nil {
		t.Fatal("MarksmanshipTalents no longer take Efficiency, Mortal Shots and Sniper Shot; pick a build that does")
	}

	discount := spellData.Efficiency.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_COST)).FractionAt(hunter.Talents.Efficiency)
	if got := hunter.SniperShot.Cost.AdditivePercentModifier; !core.WithinToleranceFloat64(1+discount, got, 1e-9) {
		t.Errorf("Sniper Shot cost modifier %v, want 1 + Efficiency's %v", got, discount)
	}
	bonus := spellData.MortalShots.FractionAt(hunter.Talents.MortalShots)
	if got := hunter.SniperShot.CritMultiplierAdditive; !core.WithinToleranceFloat64(bonus, got, 1e-9) {
		t.Errorf("Sniper Shot crit damage bonus %v, want Mortal Shots' %v", got, bonus)
	}
}

// Build 70425: Resourcefulness (440529) procs on melee and ranged ability crits only (ProcTypeMask 272).
func TestResourcefulnessProcsOnAbilityCritsOnly(t *testing.T) {
	sim, hunter := newTalentedHunter(t, SurvivalTalents)
	if hunter.Talents.Resourcefulness != 2 {
		t.Fatal("SurvivalTalents no longer take Resourcefulness 2/2 (a 100% proc); pick a build that does")
	}
	trigger := hunter.GetAura("Resourcefulness Trigger")
	buff := hunter.GetAura("Resourcefulness")
	if trigger == nil || buff == nil {
		t.Fatal("no Resourcefulness trigger or buff")
	}

	procsOn := func(spell *core.Spell) bool {
		buff.Deactivate(sim)
		trigger.OnSpellHitDealt(trigger, sim, spell, &core.SpellResult{Outcome: core.OutcomeCrit, Target: hunter.CurrentTarget})
		until := sim.CurrentTime + core.SpellBatchWindow + 1
		for sim.CurrentTime < until {
			sim.Step()
		}
		return buff.IsActive()
	}
	for _, c := range []struct {
		name  string
		spell *core.Spell
		want  bool
	}{
		{"Multi-Shot", hunter.MultiShot, true},
		{"Raptor Strike", hunter.RaptorStrikeHit, true},
		{"Auto Shot", hunter.AutoAttacks.RangedAuto(), false},
		{"melee swing", hunter.AutoAttacks.MHAuto(), false},
	} {
		if c.spell == nil {
			t.Fatalf("%s is not registered", c.name)
		}
		if got := procsOn(c.spell); got != c.want {
			t.Errorf("a %s crit procs Resourcefulness: %v, want %v", c.name, got, c.want)
		}
	}
}
