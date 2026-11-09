import { describe, expect, it } from "vitest";
import { concentrationDC } from "./concentration";

describe("concentrationDC", () => {
	it("is at least 10", () => {
		expect(concentrationDC(1)).toBe(10);
		expect(concentrationDC(21)).toBe(10);
	});
	it("is half the damage, rounded down, above 20 damage", () => {
		expect(concentrationDC(22)).toBe(11);
		expect(concentrationDC(45)).toBe(22);
	});
	it("caps at 30", () => {
		expect(concentrationDC(100)).toBe(30);
	});
});
