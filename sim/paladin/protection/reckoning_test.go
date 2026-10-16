package protection

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/paladin"
)

// Build 70425: Reckoning stores up to 4 extra attacks (1326084) and the next main-hand swing spends
// them all, each a swing at once that leaves the swing timer alone (beta logs 2747-2749). Crits store
// with no cooldown; blocks at most once per 1.5 s (client 20177 ProcCategoryRecovery).
func TestReckoningStoresAttacksForTheNextSwing(t *testing.T) {
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: core.SinglePlayerRaidProto(&proto.Player{
			Name: "prot", Class: proto.Class_ClassPaladin, Race: proto.Race_RaceHuman, TalentsString: ProtDeepTalents,
			Equipment: WeaponAndShield,
			Spec:      DefaultOptions,
			Rotation:  &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter: core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()
	pal := sim.Raid.Parties[0].Players[0].(paladin.PaladinAgent).GetPaladin()
	if pal.Talents.Reckoning == 0 || pal.ReckoningAura == nil {
		t.Fatal("test build has no Reckoning")
	}
	pal.AutoAttacks.EnableAutoSwing(sim)
	pal.AutoAttacks.StopMeleeUntil(sim, sim.CurrentTime) // a swing just landed
	next := pal.AutoAttacks.NextAttackAt()

	mh := pal.AutoAttacks.MHAuto()
	casts := func() int32 { return mh.SpellMetrics[pal.CurrentTarget.UnitIndex].Casts }
	stored := func() int32 {
		if !pal.ReckoningAura.IsActive() {
			return 0
		}
		return pal.ReckoningAura.GetStacks()
	}

	// 50 hits taken at once, read once the procs' batch window has passed.
	taken := func(trigger string, outcome core.HitOutcome, at time.Duration) {
		aura := pal.GetAura(trigger + pal.Label)
		result := &core.SpellResult{Outcome: outcome, Target: &pal.Unit}
		done := false
		sim.AddPendingAction(core.NewDelayedAction(core.DelayedActionOptions{DoAt: at, OnAction: func(sim *core.Simulation) {
			for range 50 {
				aura.OnSpellHitTaken(aura, sim, mh, result)
			}
		}}))
		sim.AddPendingAction(core.NewDelayedAction(core.DelayedActionOptions{DoAt: at + core.SpellBatchWindow + time.Millisecond, OnAction: func(sim *core.Simulation) {
			done = true
		}}))
		for !done {
			sim.Step()
		}
	}

	taken("Reckoning - Crit", core.OutcomeCrit, sim.CurrentTime)
	if got := stored(); got != 4 || casts() != 0 {
		t.Fatalf("after the crits: %d stored and %d swings, want 4 stored and no swing yet", got, casts())
	}

	for casts() == 0 {
		sim.Step()
	}
	swingAt := sim.CurrentTime
	for sim.CurrentTime < swingAt+core.SpellBatchWindow+time.Millisecond {
		sim.Step()
	}
	if swingAt != next || casts() != 5 || stored() != 0 {
		t.Errorf("the next swing at %s made %d swings and left %d stored; want it at %s, 5 swings, none left", swingAt, casts(), stored(), next)
	}
	if want := next + pal.AutoAttacks.MainhandSwingSpeed(); pal.AutoAttacks.NextAttackAt() != want {
		t.Errorf("the stored attacks moved the swing after to %s, want %s", pal.AutoAttacks.NextAttackAt(), want)
	}

	taken("Reckoning - Block", core.OutcomeBlock, sim.CurrentTime)
	if got := stored(); got != 1 {
		t.Errorf("50 blocks at once stored %d, want 1 (1.5 s cooldown)", got)
	}
	taken("Reckoning - Block", core.OutcomeBlock, sim.CurrentTime+1400*time.Millisecond)
	if got := stored(); got != 1 {
		t.Errorf("blocks inside the 1.5 s lockout stored %d in total, want 1", got)
	}
}
