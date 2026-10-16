package overrides

// A spell a server-side handler casts off another, which no effect edge, $<id> token or skill-line
// row names. Reason and Source weigh the same as an Override's, and the generator refuses an entry
// without a reason.
type HandTrigger struct {
	Spell    int32
	Triggers int32
	Reason   string
	Source   string
}

// The trigger edges the generator adds by hand to the store.
var HandTriggers = []HandTrigger{
	{
		Spell:    20230,
		Triggers: 20240,
		Reason: "Retaliation's dummy aura (aura 4) casts the counterattack 20240: same name, class set and " +
			"icon, weapon damage with no base, a cost of 1 in SpellPower that is a tenth of a rage",
		Source: "client-rows",
	},
	{
		Spell:    408341,
		Triggers: 408423,
		Reason: "Fire Nova's scripted dummy (level 12) casts the nova 408423: same name, level and fire school, " +
			"0.143 coefficient (0.214 before build 70425); beta logs record every Fire Nova hit as 408423 and none as the $-cited Era row",
		Source: "foreverlogs:2650,2671,2673,32,35",
	},
	{
		Spell:    408342,
		Triggers: 408424,
		Reason: "Fire Nova's scripted dummy (level 22) casts the nova 408424: same name, level and fire school, " +
			"0.143 coefficient (0.214 before build 70425); beta logs record every Fire Nova hit as 408423 and none as the $-cited Era row",
		Source: "foreverlogs:2650,2671,2673,32,35",
	},
	{
		Spell:    408343,
		Triggers: 408426,
		Reason: "Fire Nova's scripted dummy (level 32) casts the nova 408426: same name, level and fire school, " +
			"0.143 coefficient (0.214 before build 70425); beta logs record every Fire Nova hit as 408423 and none as the $-cited Era row",
		Source: "foreverlogs:2650,2671,2673,32,35",
	},
	{
		Spell:    408344,
		Triggers: 408427,
		Reason: "Fire Nova's scripted dummy (level 42) casts the nova 408427: same name, level and fire school, " +
			"0.143 coefficient (0.214 before build 70425); beta logs record every Fire Nova hit as 408423 and none as the $-cited Era row",
		Source: "foreverlogs:2650,2671,2673,32,35",
	},
	{
		Spell:    408345,
		Triggers: 408428,
		Reason: "Fire Nova's scripted dummy (level 52) casts the nova 408428: same name, level and fire school, " +
			"0.143 coefficient (0.214 before build 70425); beta logs record every Fire Nova hit as 408423 and none as the $-cited Era row",
		Source: "foreverlogs:2650,2671,2673,32,35",
	},
	{
		Spell:    13795,
		Triggers: 13797,
		Reason: "Immolation Trap rank 1's trap object casts the burn 13797. Client 1.60.1.70291 renamed the burn " +
			"from \"Immolation Trap Effect\" to \"Immolation Trap\", the cast's own name, so it no longer forms a family " +
			"of its own (#758); same rank, fire school and 15 s periodic damage as before the rename",
		Source: "client-rows",
	},
	{
		Spell:    14302,
		Triggers: 14298,
		Reason: "Immolation Trap rank 2's trap object casts the burn 14298. Client 1.60.1.70291 renamed the burn " +
			"from \"Immolation Trap Effect\" to \"Immolation Trap\", the cast's own name, so it no longer forms a family " +
			"of its own (#758); same rank, fire school and 15 s periodic damage as before the rename",
		Source: "client-rows",
	},
	{
		Spell:    14303,
		Triggers: 14299,
		Reason: "Immolation Trap rank 3's trap object casts the burn 14299. Client 1.60.1.70291 renamed the burn " +
			"from \"Immolation Trap Effect\" to \"Immolation Trap\", the cast's own name, so it no longer forms a family " +
			"of its own (#758); same rank, fire school and 15 s periodic damage as before the rename",
		Source: "client-rows",
	},
	{
		Spell:    14304,
		Triggers: 14300,
		Reason: "Immolation Trap rank 4's trap object casts the burn 14300. Client 1.60.1.70291 renamed the burn " +
			"from \"Immolation Trap Effect\" to \"Immolation Trap\", the cast's own name, so it no longer forms a family " +
			"of its own (#758); same rank, fire school and 15 s periodic damage as before the rename",
		Source: "client-rows",
	},
	{
		Spell:    14305,
		Triggers: 14301,
		Reason: "Immolation Trap rank 5's trap object casts the burn 14301. Client 1.60.1.70291 renamed the burn " +
			"from \"Immolation Trap Effect\" to \"Immolation Trap\", the cast's own name, so it no longer forms a family " +
			"of its own (#758); same rank, fire school and 15 s periodic damage as before the rename",
		Source: "client-rows",
	},
}
