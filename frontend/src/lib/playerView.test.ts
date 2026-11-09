import { describe, expect, it } from "vitest";
import type { Character } from "../components/CharacterList";
import { healthStatus, redactForPlayers } from "./playerView";

const base: Character = {
	ID: 1,
	Name: "x",
	ArmorClass: 15,
	ToHitModifier: 4,
	MaxHP: 10,
	CurrentHP: 10,
	Initiative: 12,
	IsActive: false,
	OwnerID: "dm",
	Type: "npc",
	ReactionUsed: false,
};

describe("healthStatus", () => {
	it("buckets HP like the backend", () => {
		expect(healthStatus(10, 10)).toBe("Healthy");
		expect(healthStatus(6, 10)).toBe("Wounded");
		expect(healthStatus(5, 10)).toBe("Bloodied");
		expect(healthStatus(0, 10)).toBe("Down");
	});
});

describe("redactForPlayers", () => {
	it("hides NPC numbers and keeps PCs", () => {
		const [npc, pc] = redactForPlayers([
			{ ...base, CurrentHP: 4 },
			{ ...base, ID: 2, Type: "pc" },
		]);
		expect(npc).toMatchObject({
			Redacted: true,
			HealthStatus: "Bloodied",
			ArmorClass: 0,
			CurrentHP: 0,
			MaxHP: 0,
			ToHitModifier: 0,
			Initiative: 12,
		});
		expect(pc.Redacted).toBeUndefined();
		expect(pc.ArmorClass).toBe(15);
	});

	it("drops hidden combatants entirely", () => {
		const out = redactForPlayers([
			base,
			{ ...base, ID: 2, Hidden: true, RevealRound: 3 },
			{ ...base, ID: 3, Type: "pc", Hidden: true },
		]);
		expect(out.map((c) => c.ID)).toEqual([1]);
	});
});
