package mage

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
)

// Build 70425: Missile Barrage takes two ranks at 20/40% from Arcane Blast, an Arcane Missile hit gets a
// quarter of that and Arcane Wrath and the bolts half.
func TestMissileBarrageRanks(t *testing.T) {
	for rank, want := range map[int32]float64{1: 20, 2: 40} {
		if got := spellData.MissileBarrage.EffectAt(1).ValueAt(rank); got != want {
			t.Errorf("rank %d: Arcane Blast chance %v%%, want %v%%", rank, got, want)
		}
	}

	_, mage := newThreeTargetMage("000200000010001") // Missile Barrage 2/2, Arcane Blast, Arcane Wrath
	if mage.MissileBarrageAura == nil {
		t.Fatal("no Missile Barrage aura at 2/2")
	}
	for _, c := range []struct {
		name  string
		spell *core.Spell
		want  float64
	}{
		{"Arcane Blast", mage.GetSpell(core.ActionID{SpellID: spellData.ArcaneBlast.Highest().ID}), 0.4},
		{"an Arcane Missile", mage.GetSpell(core.ActionID{SpellID: spellData.ArcaneMissilesTriggered.Highest().ID}), 0.1},
		{"Arcane Wrath", mage.GetSpell(core.ActionID{SpellID: spellData.ArcaneWrath.Highest().ID}), 0.2},
		{"Frostbolt", mage.GetSpell(core.ActionID{SpellID: spellData.Frostbolt.Highest().ID}), 0.2},
	} {
		if c.spell == nil {
			t.Fatalf("%s is not registered", c.name)
		}
		if got := missileBarrageChance(c.spell, 0.4); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s procs Missile Barrage at %v, want %v", c.name, got, c.want)
		}
	}
}

// Build 70425 cut Ice Lance's bonus on a frozen target from +300% to +200%.
func TestIceLanceFrozenMultiplierIsThree(t *testing.T) {
	if IceLanceFrozenMultiplier != 3 {
		t.Errorf("Ice Lance against a frozen target x%v, want x3", IceLanceFrozenMultiplier)
	}
}

// Arcane Wrath rank 4 (1326082): 3.5 sec cast, 6 sec cooldown, and the hit adds 5% of maximum mana.
func TestArcaneWrathAddsFivePercentOfMaxMana(t *testing.T) {
	sim, mage := newThreeTargetMage("000000000000001") // Arcane Wrath alone
	rank := spellData.ArcaneWrath.Highest()
	spell := mage.GetSpell(core.ActionID{SpellID: rank.ID})
	if spell == nil {
		t.Fatal("Arcane Wrath is not registered with the talent")
	}
	if spell.DefaultCast.CastTime != 3500*time.Millisecond || spell.CD.Duration != 6*time.Second {
		t.Errorf("cast %v, cooldown %v, want 3.5s and 6s", spell.DefaultCast.CastTime, spell.CD.Duration)
	}
	if got, want := spell.Cost.BaseCost, int32(rank.Cost()); got != want {
		t.Errorf("cost %v, want the row's %v", got, want)
	}

	const n = 20000
	sum := 0.0
	for range n {
		sum += mage.arcaneWrathBaseDamage(sim, rank)
	}
	want := rank.DamageEffect().Average(core.CharacterLevel) + 0.05*mage.MaxMana()
	if got := sum / n; math.Abs(got-want)/want > 0.005 {
		t.Errorf("Arcane Wrath's base damage averages %.1f, want %.1f (the row's average plus 5%% of %.0f mana)", got, want, mage.MaxMana())
	}
}
