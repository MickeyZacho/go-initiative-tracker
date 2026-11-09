import { describe, expect, it } from "vitest";
import { applyHpInput } from "./hp";

describe("applyHpInput", () => {
	it("subtracts damage written as -N", () => {
		expect(applyHpInput("-7", 15, 20)).toBe(8);
	});
	it("adds healing written as +N", () => {
		expect(applyHpInput("+5", 10, 20)).toBe(15);
	});
	it("sets an absolute value written as N", () => {
		expect(applyHpInput("12", 3, 20)).toBe(12);
	});
	it("clamps damage at 0", () => {
		expect(applyHpInput("-50", 15, 20)).toBe(0);
	});
	it("clamps healing and absolute values at max HP", () => {
		expect(applyHpInput("+50", 15, 20)).toBe(20);
		expect(applyHpInput("99", 15, 20)).toBe(20);
	});
	it("tolerates whitespace", () => {
		expect(applyHpInput(" - 4 ", 10, 20)).toBe(6);
	});
	it("returns null for non-numeric input", () => {
		expect(applyHpInput("abc", 10, 20)).toBeNull();
		expect(applyHpInput("", 10, 20)).toBeNull();
		expect(applyHpInput("--3", 10, 20)).toBeNull();
	});
});
