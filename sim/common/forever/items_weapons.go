package forever

import (
	"time"

	"github.com/wowsims/forever/sim/common/itemhelpers"
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/core/stats"
)

func init() {
	// The Lobotomizer: Forever's Brain Damage (client 1290950) wounds for 250 +-40% (200 to 300)
	// and slows the target's casting. 0.4 PPM and the magic hit table as on master.
	itemhelpers.CreateWeaponProcSpell(itemhelpers.WeaponProcSpell{
		ItemID: 19324,
		Name:   "The Lobotomizer",
		PPM:    0.4,
		Spell: func(character *core.Character) *core.Spell {
			return character.GetOrRegisterSpell(core.SpellConfig{
				ActionID:    core.ActionID{SpellID: 1290950},
				ProcMask:    core.ProcMaskEmpty,
				SpellSchool: core.SpellSchoolPhysical,
				DefenseType: core.DefenseTypeMelee,
				Flags:       core.SpellFlagPassiveSpell,

				DamageMultiplier: 1,
				ThreatMultiplier: 1,

				ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
					spell.CalcAndDealDamage(sim, target, sim.Roll(200, 300), spell.OutcomeMagicHitAndCrit)
				},
			})
		},
	})

	// Bonereaver's Edge
	itemhelpers.CreateWeaponProcTrigger(itemhelpers.WeaponProcTrigger{
		ItemID: 17076,
		Name:   "Bonereaver's Edge",
		PPM:    2,
		Handler: func(character *core.Character) core.ProcHandler {
			arpAura := core.MakeStackingAura(
				character,
				core.StackingStatAura{
					Aura: core.Aura{
						Label:     "Bonereaver's Edge",
						ActionID:  core.ActionID{SpellID: 21153},
						Duration:  time.Second * 10,
						MaxStacks: 3,
					},
					BonusPerStack: stats.Stats{
						stats.ArmorPenetration: 700,
					},
				},
			)

			return func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
				arpAura.Activate(sim)
				arpAura.AddStack(sim)
			}
		},
	})

	// Bashguuder, Rivenspike: chance on hit, Puncture Armor (client 17315: -100 armor, 3 stacks,
	// 30s). The client has no proc rate; 2 PPM is master's (Armaments Discord).
	for itemID, name := range map[int32]string{13204: "Bashguuder", 13286: "Rivenspike"} {
		itemhelpers.CreateWeaponProcTrigger(itemhelpers.WeaponProcTrigger{
			ItemID: itemID,
			Name:   name,
			PPM:    2,
			Handler: func(character *core.Character) core.ProcHandler {
				auras := character.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
					return target.GetOrRegisterAura(core.Aura{
						Label:     "Puncture Armor",
						ActionID:  core.ActionID{SpellID: 17315},
						Duration:  time.Second * 30,
						MaxStacks: 3,
						OnStacksChange: func(aura *core.Aura, sim *core.Simulation, oldStacks int32, newStacks int32) {
							aura.Unit.AddStatDynamic(sim, stats.Armor, -100*float64(newStacks-oldStacks))
						},
					})
				})

				return func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
					aura := auras.Get(result.Target)
					aura.Activate(sim)
					aura.AddStack(sim)
				}
			},
		})
	}

	// Chance on hit procs the client states no rate for, measured in foreverlogs beta timelines as
	// procs per landed hit of the weapon (white and yellow) times 60 / weapon speed:
	// Barbaric Crossbow, Wound (1291551, 14 Physical, ranged table): 303 procs off 2671 Auto Shot
	// and Arcane Shot hits of 3 hunters (report 2650) = 3.1-3.4 PPM.
	// Plaguefang, Poison (1309315, 4 Nature a sec for 10 sec, ticks crit): 59 procs off 699 hits of a
	// warrior whose swing timer held Plaguefang's 2.1 sec in every fight (report 2677) = 2.4 PPM.
	// Wolfsbane, Blazewind Blast (1282503, 49 Holystorm): 107 procs off 706 landed hits of one paladin
	// over 61 fights (report 2678) = 2.7 +- 0.24 PPM. Seal of Command and Judgement of Command hits
	// trigger it as well as swings and Holy Strike; Consecration ticks never do. The triple damage to
	// Wolves and Worgen is left out (no target of that kind is simulated).
	// Venomstrike, Venom Shot (29653, 28 Nature): 59 procs off 899 landed Auto Shot, Arcane Shot,
	// Multi-Shot and Aimed Shot hits of 2 hunters over 66 fights (reports 2668, 2674; auto shot
	// gaps 2.2-2.3 sec, Venomstrike's 2.4 with quiver haste) = 1.6 +- 0.2 PPM.
	// Stinging Viper, Poison (1291663, 7 Nature every 3 sec for 12 sec): 51 procs off 345 landed
	// swing, Cleave and Victory Rush hits of a protection warrior over 35 fights (report 2679;
	// swing gaps 2.77 sec, the mace's 2.8) = 3.2 +- 0.4 PPM.
	for _, proc := range []struct {
		itemID  int32
		name    string
		ppm     float64
		spellID int32
	}{
		{272999, "Barbaric Crossbow", 3.2, 1291551},
		{279876, "Plaguefang", 2.4, 1309315},
		{267369, "Wolfsbane", 2.7, 1282503},
		{6469, "Venomstrike", 1.6, 29653},
		{6472, "Stinging Viper", 3.2, 1291663},
		// No beta log carries these yet. Their damage rows match Era's (Coldrage Dagger's is Forever's
		// own), so the rate is master's Classic one from the Armaments Discord, as for Bashguuder.
		{14555, "Alcor's Sunrazor", 1, 18833},
		{11744, "Bloodfist", 4, 16433},
		{14487, "Bonechill Hammer", 1, 18276},
		{13984, "Darrowspike", 1, 18276},
		{10761, "Coldrage Dagger", 2.2, 1293790},
		{19099, "Glacial Blade", 1.4, 18398},
		{11809, "Flame Wrath", 1, 16559},
		{12794, "Masterwork Stormhammer", 0.5, 16921},
		// Electrified Dagger (Alliance) mirrors Glacial Blade (Horde): same 45 damage bolt, so master
		// gave it Glacial Blade's 1.4.
		{19100, "Electrified Dagger", 1.4, 23592},
		// The same for three that drain life: only the damage is simulated, not what it heals.
		{17074, "Shadowstrike", 2.2, 21170},
		{13401, "The Cruel Hand of Timmy", 0.65, 17505},
		{13361, "Skullforge Reaver", 1.7, 17484},
	} {
		itemhelpers.CreateWeaponProcSpell(itemhelpers.WeaponProcSpell{
			ItemID: proc.itemID,
			Name:   proc.name,
			PPM:    proc.ppm,
			Spell: func(character *core.Character) *core.Spell {
				return character.GetOrRegisterSpell(shared.SpellDataProcDamageSpell(character, spelldata.MustFind(proc.spellID)))
			},
		})
	}

	// Flurry Axe: chance on hit, 1 extra attack (client 18797). No rate in the client or a beta log;
	// 1.9 PPM is master's from the Armaments Discord (wowsims/classic 39c7a8071d raised it from 1).
	itemhelpers.CreateWeaponProcSpell(itemhelpers.WeaponProcSpell{
		ItemID: 871,
		Name:   "Flurry Axe",
		PPM:    1.9,
		Spell: func(character *core.Character) *core.Spell {
			return character.GetOrRegisterSpell(core.SpellConfig{
				ActionID:    core.ActionID{SpellID: 18797},
				SpellSchool: core.SpellSchoolPhysical,
				ProcMask:    core.ProcMaskEmpty,
				ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
					character.AutoAttacks.ExtraMHAttack(sim)
				},
			})
		},
	})

	// Chance on hit buffs the client states no rate for either, from their rows: Sword of Zeal's Zeal
	// (8191: +10 physical damage, +150 armor, 15 sec) and Argent Avenger's (17352: +100 attack power
	// and ranged attack power, 10 sec). No beta log has them yet, so the rate is master's from the
	// Armaments Discord: Sword of Zeal's comment there says 1.8 (its code said 1), Argent Avenger 1.
	// Argent Avenger's further 100 attack power against Undead is left out: the parser skips it.
	for _, proc := range []struct {
		itemID  int32
		name    string
		ppm     float64
		spellID int32
	}{
		{6622, "Sword of Zeal", 1.8, 8191},
		{13246, "Argent Avenger", 1, 17352},
	} {
		itemhelpers.CreateWeaponProcAura(itemhelpers.WeaponProcAura{
			ItemID: proc.itemID,
			Name:   proc.name,
			PPM:    proc.ppm,
			Aura: func(character *core.Character) *core.Aura {
				row := spelldata.MustFind(proc.spellID)
				aura := character.GetOrRegisterAura(core.Aura{
					Label:    proc.name,
					ActionID: core.ActionID{SpellID: proc.spellID},
					Duration: row.Duration(),
				})
				spelldata.ParseEffects(character, aura, row)
				return aura
			},
		})
	}
}

// Build 70425's new item effects, which the generator registers wrongly or not at all.
func init() {
	// Blade of Hanna (2801): each main-hand melee hit adds a stack of Improving Form (1325535 ->
	// 1325509, 30 stacks). The 30th perfects the form (1325508): 490 Physical damage to the target,
	// +20 all stats for 20 sec, and the stacks are spent (its E_REMOVE_AURA on 1325509). The generated
	// entry cast the 490 on every swing. Whether stacks build while Perfected Form is up is not
	// stated; the sim lets them.
	core.NewItemEffect(2801, func(agent core.Agent) {
		character := agent.GetCharacter()
		perfectedRow := spelldata.MustFind(1325508)
		allStats := perfectedRow.EffectN(2).Average(core.CharacterLevel)

		perfectedForm := character.NewTemporaryStatsAura("Perfected Form", core.ActionID{SpellID: perfectedRow.ID},
			stats.Stats{stats.Strength: allStats, stats.Agility: allStats, stats.Stamina: allStats, stats.Intellect: allStats, stats.Spirit: allStats},
			perfectedRow.Duration())
		strike := character.GetOrRegisterSpell(shared.SpellDataProcDamageSpell(character, perfectedRow))

		stackRow := spelldata.MustFind(1325509)
		improvingForm := character.RegisterAura(core.Aura{
			Label:     "Improving Form",
			ActionID:  core.ActionID{SpellID: stackRow.ID},
			Duration:  core.NeverExpires,
			MaxStacks: int32(stackRow.MaxStack),
		})

		character.MakeProcTriggerAura(core.ProcTrigger{
			Name:     "Blade of Hanna",
			ActionID: core.ActionID{SpellID: 1325535},
			Callback: core.CallbackOnSpellHitDealt,
			ProcMask: core.ProcMaskMeleeMHAuto,
			Outcome:  core.OutcomeLanded,
			Handler: func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
				if !improvingForm.IsActive() {
					improvingForm.Activate(sim)
				}
				improvingForm.AddStack(sim)
				if improvingForm.GetStacks() < improvingForm.MaxStacks {
					return
				}
				improvingForm.Deactivate(sim)
				strike.Cast(sim, result.Target)
				perfectedForm.Activate(sim)
			},
		})
	})

	// Lord Sakrasis' Scepter (5028), held in the off hand: its equip aura 1326019 casts Holy Smite
	// (1312330, 56 Holy) off harmful spells at SpellProcsPerMinute 455, 2 procs a minute with the
	// haste flag. The store does not read that table, so the generated entry fired on every cast.
	// The rate here is a real-PPM one: 2 x the time since the last chance (capped at 3.5 sec) x
	// spell haste / 60, without the client's bad-luck protection. Unverified.
	core.NewItemEffect(5028, func(agent core.Agent) {
		character := agent.GetCharacter()
		const ppm, maxGap = 2.0, 3500 * time.Millisecond
		smite := character.GetOrRegisterSpell(shared.SpellDataProcDamageSpell(character, spelldata.MustFind(1312330)))

		var lastChance time.Duration
		character.RegisterResetEffect(func(_ *core.Simulation) { lastChance = 0 })
		character.MakeProcTriggerAura(core.ProcTrigger{
			Name:     "Lord Sakrasis' Scepter",
			ActionID: core.ActionID{SpellID: 1326019},
			Callback: core.CallbackOnSpellHitDealt,
			ProcMask: core.ProcMaskSpellDamage,
			Outcome:  core.OutcomeLanded,
			Handler: func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
				gap := min(sim.CurrentTime-lastChance, maxGap)
				lastChance = sim.CurrentTime
				chance := ppm * gap.Seconds() / 60 * character.TotalSpellHasteMultiplier()
				if sim.Proc(chance, "Lord Sakrasis' Scepter") {
					smite.Cast(sim, result.Target)
				}
			},
		})
	})

	// Robes of Insight (940): build 70425's Insight (18820) is -200 mana on every spell (aura 73,
	// magic schools, mana) for 12 sec on a 10 min use, where it was one spell's cost up to 500. The
	// generator has no shape for a cost aura. A mana cooldown, so it stays out of the all-items DPS
	// goldens.
	core.AddEffectsToTest = false
	defer func() { core.AddEffectsToTest = true }()
	core.NewItemEffect(940, func(agent core.Agent) {
		character := agent.GetCharacter()
		row := spelldata.MustFind(18820)
		costMod := character.AddDynamicMod(core.SpellModConfig{
			Kind:         core.SpellMod_PowerCost_Flat,
			ResourceType: proto.ResourceType_ResourceTypeMana,
			School:       core.SpellSchoolChaos,
			IntValue:     int32(row.EffectN(1).BasePoints),
		})
		aura := character.RegisterAura(core.Aura{
			Label:    "Insight",
			ActionID: core.ActionID{SpellID: row.ID},
			Duration: row.Duration(),
			OnGain: func(_ *core.Aura, _ *core.Simulation) {
				costMod.Activate()
			},
			OnExpire: func(_ *core.Aura, _ *core.Simulation) {
				costMod.Deactivate()
			},
		})
		spell := character.RegisterSpell(core.SpellConfig{
			ActionID: core.ActionID{ItemID: 940},
			Flags:    core.SpellFlagNoOnCastComplete,
			Cast: core.CastConfig{
				CD: core.Cooldown{
					Timer:    character.NewTimer(),
					Duration: time.Minute * 10,
				},
			},
			ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
				aura.Activate(sim)
			},
		})
		character.AddMajorCooldown(core.MajorCooldown{
			Spell: spell,
			Type:  core.CooldownTypeMana,
			ShouldActivate: func(_ *core.Simulation, character *core.Character) bool {
				return character.HasManaBar()
			},
		})
	})
}
