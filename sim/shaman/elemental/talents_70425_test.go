package elemental

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

func newElementalSim(t *testing.T, talents string) (*core.Simulation, *core.Character) {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Race:          proto.Race_RaceTroll,
		Class:         proto.Class_ClassShaman,
		Equipment:     &proto.EquipmentSpec{},
		Consumables:   &proto.ConsumesSpec{},
		TalentsString: talents,
		Rotation:      &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
	}, &proto.Player_ElementalShaman{ElementalShaman: &proto.ElementalShaman{Options: &proto.ElementalShaman_Options{ClassOptions: &proto.ShamanOptions{}}}})
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 100},
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim, sim.Raid.Parties[0].Players[0].GetCharacter()
}

// Build 70425: Call of Flame 3/3 adds 9 sec to Flame Shock (16038 effect 3), three more 3 sec ticks.
func TestCallOfFlameLengthensFlameShock(t *testing.T) {
	ticks := func(talents string) int32 {
		_, char := newElementalSim(t, talents)
		return char.GetSpell(core.ActionID{SpellID: 29228}).Dot(char.CurrentTarget).BaseTickCount
	}
	if base, talented := ticks(""), ticks("0000030"); talented != base+3 {
		t.Errorf("Flame Shock ticks %d with Call of Flame 3/3, want %d + 3", talented, base)
	}
}

// Build 70425: Trial by Fire 2/2 adds 20 sec to Magma Totem (16086 effect 3), ten more 2 sec pulses.
func TestTrialByFireLengthensMagmaTotem(t *testing.T) {
	pulses := func(talents string) int32 {
		_, char := newElementalSim(t, talents)
		return char.GetSpell(core.ActionID{SpellID: 10587}).AOEDot().BaseTickCount
	}
	if base, talented := pulses(""), pulses("000000002"); talented != base+10 {
		t.Errorf("Magma Totem pulses %d with Trial by Fire 2/2, want %d + 10", talented, base)
	}
}

// Elemental Mastery (16166, new to the tree in build 70425): for 15 sec Lightning Bolt casts 0.5 sec
// faster and Chain Lightning has no cooldown; casting a bolt does not end it.
func TestElementalMastery(t *testing.T) {
	sim, char := newElementalSim(t, "0000000000000001")
	em := char.GetSpell(core.ActionID{SpellID: 16166})
	if em == nil {
		t.Fatal("Elemental Mastery is not registered")
	}
	bolt := char.GetSpell(core.ActionID{SpellID: 15208})
	chain := char.GetSpell(core.ActionID{SpellID: 10605})
	castTime, chainCD := bolt.DefaultCast.CastTime, chain.CD.Duration

	em.SkipCastAndApplyEffects(sim, char.CurrentTarget)
	aura := char.GetAura("Elemental Mastery")
	if !aura.IsActive() || aura.Duration != 15*time.Second {
		t.Fatalf("Elemental Mastery active %v for %v, want active for 15s", aura.IsActive(), aura.Duration)
	}
	if got := bolt.DefaultCast.CastTime; got != castTime-500*time.Millisecond {
		t.Errorf("Lightning Bolt casts in %v under Elemental Mastery, want %v", got, castTime-500*time.Millisecond)
	}
	if got := chain.CD.Duration; got != 0 {
		t.Errorf("Chain Lightning cooldown %v under Elemental Mastery, want none", got)
	}

	bolt.SkipCastAndApplyEffects(sim, char.CurrentTarget)
	if !aura.IsActive() {
		t.Error("a Lightning Bolt spent Elemental Mastery")
	}

	aura.Deactivate(sim)
	if bolt.DefaultCast.CastTime != castTime || chain.CD.Duration != chainCD {
		t.Errorf("after Elemental Mastery: bolt %v, chain cooldown %v, want %v and %v",
			bolt.DefaultCast.CastTime, chain.CD.Duration, castTime, chainCD)
	}
}
