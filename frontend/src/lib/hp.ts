// HP entry for the combat row. DMs think in deltas ("the goblin takes 7"), so
// the HP field accepts:
//   "-7"  → damage: subtract 7
//   "+5"  → healing: add 5
//   "12"  → set current HP to 12
// The result is clamped to 0..maxHP. Returns null when the input isn't a
// number, so the caller can treat it as "no change".
export function applyHpInput(
	input: string,
	currentHP: number,
	maxHP: number,
): number | null {
	const text = input.trim().replace(/\s+/g, "");
	const match = /^([+-]?)(\d+)$/.exec(text);
	if (!match) return null;
	const [, sign, digits] = match;
	const amount = Number(digits);
	let next: number;
	if (sign === "-") next = currentHP - amount;
	else if (sign === "+") next = currentHP + amount;
	else next = amount;
	return Math.max(0, Math.min(maxHP, next));
}
