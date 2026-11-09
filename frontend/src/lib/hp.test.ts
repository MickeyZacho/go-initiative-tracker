import { describe, expect, it } from "vitest";
import { applyHpInput } from "./hp";

const hp = (current: number, temp = 0) => ({ current, temp });

describe("applyHpInput", () => {
	it("subtracts damage written as -N", () => {
		expect(applyHpInput("-7", hp(15), 20)).toEqual(hp(8));
	});
	it("adds healing written as +N", () => {
		expect(applyHpInput("+5", hp(10), 20)).toEqual(hp(15));
	});
	it("sets an absolute value written as N", () => {
		expect(applyHpInput("12", hp(3), 20)).toEqual(hp(12));
	});
	it("clamps damage at 0", () => {
		expect(applyHpInput("-50", hp(15), 20)).toEqual(hp(0));
	});
	it("clamps healing and absolute values at max HP", () => {
		expect(applyHpInput("+50", hp(15), 20)).toEqual(hp(20));
		expect(applyHpInput("99", hp(15), 20)).toEqual(hp(20));
	});
	it("tolerates whitespace", () => {
		expect(applyHpInput(" - 4 ", hp(10), 20)).toEqual(hp(6));
	});
	it("returns null for non-numeric input", () => {
		expect(applyHpInput("abc", hp(10), 20)).toBeNull();
		expect(applyHpInput("", hp(10), 20)).toBeNull();
		expect(applyHpInput("--3", hp(10), 20)).toBeNull();
	});

	describe("temp HP", () => {
		it("absorbs damage before current HP", () => {
			expect(applyHpInput("-3", hp(10, 5), 20)).toEqual(hp(10, 2));
			expect(applyHpInput("-8", hp(10, 5), 20)).toEqual(hp(7, 0));
		});
		it("is not raised by healing", () => {
			expect(applyHpInput("+4", hp(10, 5), 20)).toEqual(hp(14, 5));
		});
		it("is untouched by setting current HP", () => {
			expect(applyHpInput("3", hp(10, 5), 20)).toEqual(hp(3, 5));
		});
	});
});
