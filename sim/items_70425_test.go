package sim

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

func itemTestPlayer(class proto.Class, race proto.Race, spec any, slot proto.ItemSlot, itemID int32, apl string) *proto.Player {
	items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[slot] = &proto.ItemSpec{Id: itemID}
	player := &proto.Player{
		Class:       class,
		Race:        race,
		Equipment:   &proto.EquipmentSpec{Items: items},
		Consumables: &proto.ConsumesSpec{},
		Rotation:    &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
	}
	if apl != "" {
		player.Rotation = core.APLRotationFromJsonString(apl)
	}
	return core.WithSpec(player, spec)
}

func hitsOf(t *testing.T, player *proto.Player, spellID int32) int32 {
	t.Helper()
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  STEncounter,
		SimOptions: &proto.SimOptions{Iterations: 1, IsTest: true, RandomSeed: 7},
	})
	if result.Error != nil {
		t.Fatalf("%.300s", result.Error.Message)
	}
	var hits int32
	for _, action := range result.RaidMetrics.Parties[0].Players[0].Actions {
		if action.Id.GetSpellId() == spellID {
			for _, target := range action.Targets {
				hits += target.Hits // a proc's spell records hits, not casts
			}
		}
	}
	return hits
}

// Blade of Hanna (build 70425) perfects its form once every 30 main-hand hits, not on every swing:
// a 2.0 sec weapon lands well under 100 swings in 3 min, so a handful of Perfected Forms.
func TestBladeOfHannaPerfectsEveryThirtyHits(t *testing.T) {
	warrior := itemTestPlayer(proto.Class_ClassWarrior, proto.Race_RaceHuman,
		&proto.Player_DpsWarrior{DpsWarrior: &proto.DpsWarrior{Options: &proto.DpsWarrior_Options{ClassOptions: &proto.WarriorOptions{}}}},
		proto.ItemSlot_ItemSlotMainHand, 2801, "")
	if casts := hitsOf(t, warrior, 1325508); casts < 1 || casts > 5 {
		t.Errorf("Perfected Form struck %d times in 3 min, want 1-5 (one per 30 hits)", casts)
	}
}

// Lord Sakrasis' Scepter casts Holy Smite at about 2 a minute off harmful spells, not on every cast.
func TestLordSakrasisScepterProcsTwiceAMinute(t *testing.T) {
	mage := itemTestPlayer(proto.Class_ClassMage, proto.Race_RaceGnome,
		&proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{}}}},
		proto.ItemSlot_ItemSlotOffHand, 5028,
		`{"type": "TypeAPL", "priorityList": [{"action": {"castSpell": {"spellId": {"spellId": 133}}}}]}`)
	if casts := hitsOf(t, mage, 1312330); casts < 1 || casts > 15 {
		t.Errorf("Holy Smite procced %d times in 3 min, want about 6", casts)
	}
}

// Robes of Insight's use (build 70425): every spell costs 200 less mana for 12 sec.
func TestRobesOfInsightCutsEverySpellsCost(t *testing.T) {
	mage := itemTestPlayer(proto.Class_ClassMage, proto.Race_RaceGnome,
		&proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{}}}},
		proto.ItemSlot_ItemSlotChest, 940, "")
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(mage, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  STEncounter,
		SimOptions: &proto.SimOptions{Iterations: 1, IsTest: true, RandomSeed: 7},
	}, simsignals.CreateSignals())
	sim.Reset()
	character := sim.Raid.Parties[0].Players[0].GetCharacter()

	fireball := character.GetSpell(core.ActionID{SpellID: 133})
	insight := character.GetAura("Insight")
	if fireball == nil || insight == nil {
		t.Fatalf("Fireball %v, Insight %v: want both registered", fireball, insight)
	}
	before := fireball.Cost.GetCurrentCost()
	insight.Activate(sim)
	if got := fireball.Cost.GetCurrentCost(); got != max(before-200, 0) {
		t.Errorf("Fireball costs %v with Insight, %v without, want 200 less", got, before)
	}
}
